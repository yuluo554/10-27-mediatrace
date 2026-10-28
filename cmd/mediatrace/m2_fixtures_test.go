package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yuluo554/mediatrace/internal/policy"
	"github.com/yuluo554/mediatrace/internal/report"
	"github.com/yuluo554/mediatrace/internal/rules/duration"
	"github.com/yuluo554/mediatrace/tools/fixture-gen/gen"
)

// m2ControlSuites 是"必须误报 0"的四个对照组（plan/05 §二；共 60 例）。
var m2ControlSuites = map[string]bool{
	"controls":      true,
	"m3-regression": true,
	"m4":            true,
	"m5":            true,
}

// M3 DoD：模块 M2 对夹具集的结论 —— 缺陷类命中、**对照组 60 例误报 0**。
//
// 真值来自生成器构造参数（K8：生成器与解析器不共享代码）。
// 断言用**集合相等**而非"包含"：既抓漏报（C4a/C4a2/C4a3/C4b、E1/E2/E3）也抓误报。
// 与 M2 无关的其它缺陷样本（C1–C3、C5–C7、D/P/S/T 组）本棒**不判**（HANDOFF-M3 §三.3）。
func TestM2MatchesFixtureTruth(t *testing.T) {
	cases, err := gen.Build(gen.Config{Seed: 20261027, Suite: "all"})
	if err != nil {
		t.Fatalf("generate fixtures: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("fixture generator produced 0 cases — assertion is vacuous")
	}
	m2rules := map[string]bool{}
	for _, r := range duration.RuleIDs {
		m2rules[r] = true
	}

	dir := t.TempDir()
	relevant, controls := 0, 0
	for _, c := range cases {
		p := filepath.Join(dir, c.Filename)
		if err := os.WriteFile(p, c.Bytes, 0o444); err != nil {
			t.Fatalf("write %s: %v", c.Filename, err)
		}
		var buf bytes.Buffer
		code := run([]string{"audit", p, "--modules", "M2", "--json"}, &buf, &buf)

		var rep report.Report
		if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
			t.Fatalf("%s: JSON decode: %v\n%s", c.ID, err, buf.String())
		}
		got := map[string]bool{}
		for _, f := range rep.Findings {
			if f.Module != "M2" {
				t.Errorf("%s: non-M2 finding leaked: %+v", c.ID, f)
			}
			got[f.RuleID] = true
		}

		if m2ControlSuites[c.Suite] {
			controls++
			if len(rep.Findings) != 0 {
				t.Errorf("%s (%s): FALSE POSITIVE — control case produced M2 findings %v", c.ID, c.Suite, keysOf(got))
			}
			if code != policy.ExitPass {
				t.Errorf("%s: control case exit=%d, want %d", c.ID, code, policy.ExitPass)
			}
			continue
		}

		want := map[string]bool{}
		for _, r := range c.Expected {
			if m2rules[r] {
				want[r] = true
			}
		}
		if len(want) == 0 {
			continue // 与 M2 无关的缺陷样本：本棒不判
		}
		relevant++
		if !sameStringSet(got, want) {
			t.Errorf("%s (%s): M2 rules = %v, want %v", c.ID, c.ExpectStatus, keysOf(got), keysOf(want))
		}
		wantExit := policy.ExitFail
		if c.ExpectStatus == gen.ExpectGray {
			wantExit = policy.ExitInconclusive
		}
		if code != wantExit {
			t.Errorf("%s: exit=%d, want %d", c.ID, code, wantExit)
		}
	}

	// 门禁不得空跑：验收样本与对照组都要有量。
	if relevant < 8 {
		t.Fatalf("only %d M2-relevant cases covered — acceptance gate not exercised", relevant)
	}
	if controls < 60 {
		t.Fatalf("only %d control cases checked — false-positive gate not exercised (want >= 60)", controls)
	}
	t.Logf("M2 audited %d cases: %d M2-relevant defects hit, %d controls clean", len(cases), relevant, controls)
}

// 每条 fail 必带四元组（G1 的夹具级复核）：M2 的 fail 经渲染层校验后原样落地。
func TestM2FailFindingsCarryFullEvidence(t *testing.T) {
	cases, err := gen.Build(gen.Config{Seed: 20261027, Suite: "defects"})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	checked := 0
	for _, c := range cases {
		p := filepath.Join(dir, c.Filename)
		if err := os.WriteFile(p, c.Bytes, 0o444); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		run([]string{"audit", p, "--modules", "M2", "--json"}, &buf, &buf)
		var rep report.Report
		if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		for _, f := range rep.Findings {
			if f.Status != report.Fail {
				continue
			}
			checked++
			if f.FieldPath == "" || len(f.Values) == 0 || f.Delta == nil {
				t.Errorf("%s: fail lacks evidence four-tuple: %+v", c.ID, f)
			}
			if f.Delta.Unit == "" {
				t.Errorf("%s: fail delta lacks unit: %+v", c.ID, f.Delta)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no M2 fail findings checked — assertion is vacuous")
	}
}

// G9：单模块结论 == 全集里该模块的结论。全集因 M3–M5 未实现而 exit 3，
// 故用 `--modules M1,M2` 与 `--modules M2` 对拍 M2 那一半。
func TestM2SubsetConclusionIsStable(t *testing.T) {
	cases, err := gen.Build(gen.Config{Seed: 20261027, Suite: "all"})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	checked := 0
	for _, c := range cases {
		p := filepath.Join(dir, c.Filename)
		if err := os.WriteFile(p, c.Bytes, 0o444); err != nil {
			t.Fatal(err)
		}
		alone := fingerprintsOfModule(t, p, "M2", "--modules", "M2")
		withM1 := fingerprintsOfModule(t, p, "M2", "--modules", "M1,M2")
		if alone != withM1 {
			t.Errorf("%s: M2 conclusion depends on module selection: %q vs %q", c.ID, alone, withM1)
		}
		again := fingerprintsOfModule(t, p, "M2", "--modules", "M2")
		if alone != again {
			t.Errorf("%s: --modules M2 not idempotent: %q vs %q", c.ID, alone, again)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no cases checked")
	}
}

func fingerprintsOfModule(t *testing.T, path, module string, args ...string) string {
	t.Helper()
	var buf bytes.Buffer
	run(append([]string{"audit", path, "--json"}, args...), &buf, &buf)
	var rep report.Report
	if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	out := ""
	for _, f := range rep.Findings {
		if f.Module == module {
			out += f.Fingerprint() + "\n"
		}
	}
	return out
}
