package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/yuluo554/mediatrace/internal/policy"
	"github.com/yuluo554/mediatrace/internal/report"
	"github.com/yuluo554/mediatrace/internal/rules/timestamps"
	"github.com/yuluo554/mediatrace/tools/fixture-gen/gen"
)

// m3ControlSuites 是"必须误报 0"的四个对照组（plan/05 §二；共 60 例）。
var m3ControlSuites = map[string]bool{
	"controls":      true,
	"m3-regression": true,
	"m4":            true,
	"m5":            true,
}

// M4 DoD：模块 M3 对夹具集的结论 —— 缺陷类命中、**对照组 60 例误报 0**。
//
// 真值来自生成器构造参数（K8：生成器与解析器不共享代码）。
// 断言用**集合相等**而非"包含"：既抓漏报（C5/C5b/C5c/T1/T2）也抓误报。
// 额外钉住**构造点已知的字节偏移**（manifest 的 expected_offset）必须出现在该模块的
// finding 偏移集里——这是"每条 fail 带偏移"的直接证据（M1/M2 只比对规则集）。
// 与 M3 无关的缺陷样本（C1–C4、C6、C7、D/P/S/E 组）本棒**不判**（HANDOFF-M4 §三.3）。
func TestM3MatchesFixtureTruth(t *testing.T) {
	cases, err := gen.Build(gen.Config{Seed: 20261027, Suite: "all"})
	if err != nil {
		t.Fatalf("generate fixtures: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("fixture generator produced 0 cases — assertion is vacuous")
	}
	m3rules := map[string]bool{}
	for _, r := range timestamps.RuleIDs {
		m3rules[r] = true
	}

	dir := t.TempDir()
	relevant, controls := 0, 0
	for _, c := range cases {
		p := filepath.Join(dir, c.Filename)
		if err := os.WriteFile(p, c.Bytes, 0o444); err != nil {
			t.Fatalf("write %s: %v", c.Filename, err)
		}
		var buf bytes.Buffer
		code := run([]string{"audit", p, "--modules", "M3", "--json"}, &buf, &buf)

		var rep report.Report
		if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
			t.Fatalf("%s: JSON decode: %v\n%s", c.ID, err, buf.String())
		}
		got := map[string]bool{}
		offsets := map[uint64]bool{}
		for _, f := range rep.Findings {
			if f.Module != "M3" {
				t.Errorf("%s: non-M3 finding leaked: %+v", c.ID, f)
			}
			got[f.RuleID] = true
			offsets[f.ByteOffset] = true
		}

		if m3ControlSuites[c.Suite] {
			controls++
			if len(rep.Findings) != 0 {
				t.Errorf("%s (%s): FALSE POSITIVE — control case produced M3 findings %v", c.ID, c.Suite, keysOf(got))
			}
			if code != policy.ExitPass {
				t.Errorf("%s: control case exit=%d, want %d", c.ID, code, policy.ExitPass)
			}
			continue
		}

		want := map[string]bool{}
		for _, r := range c.Expected {
			if m3rules[r] {
				want[r] = true
			}
		}
		if len(want) == 0 {
			continue // 与 M3 无关的缺陷样本：本棒不判
		}
		relevant++
		if !sameStringSet(got, want) {
			t.Errorf("%s (%s): M3 rules = %v, want %v", c.ID, c.Suite, keysOf(got), keysOf(want))
		}
		// 构造点已知的首证据偏移（manifest）必须落在本模块的 finding 偏移集里。
		if !offsets[c.Offset] {
			t.Errorf("%s: expected byte offset 0x%X absent from M3 finding offsets %v", c.ID, c.Offset, offsetKeys(offsets))
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
	if relevant < 5 {
		t.Fatalf("only %d M3-relevant cases covered — acceptance gate not exercised", relevant)
	}
	if controls < 60 {
		t.Fatalf("only %d control cases checked — false-positive gate not exercised (want >= 60)", controls)
	}
	t.Logf("M3 audited %d cases: %d M3-relevant defects hit, %d controls clean", len(cases), relevant, controls)
}

// B 帧不得被判回退（DoD：`ctts` v1 有符号偏移不算 DTS 回退）。
//
// 链路：`gen_test.go` 的 `TestSuiteThresholdsHoldByConstruction` 断言 `m3-regression`
// **同时**含 ctts v0（正偏移乱序）与 v1（**有符号含负偏移**）两类 B 帧样本；本测试断言
// 该组**全部 24 例**经模块 M3 判**0 finding**（DTS 单调，PTS 不参与）。两者合起来 =
// 「含负偏移的 B 帧文件不被当回退」。
func TestM3DoesNotMisjudgeBFrames(t *testing.T) {
	cases, err := gen.Build(gen.Config{Seed: 20261027, Suite: "m3-regression"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) < 20 {
		t.Fatalf("m3-regression produced %d cases — B-frame gate not exercised (want >= 20)", len(cases))
	}
	dir := t.TempDir()
	for _, c := range cases {
		p := filepath.Join(dir, c.Filename)
		if err := os.WriteFile(p, c.Bytes, 0o444); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		code := run([]string{"audit", p, "--modules", "M3", "--json"}, &buf, &buf)
		var rep report.Report
		if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		if len(rep.Findings) != 0 {
			t.Errorf("%s: B-frame file misjudged — M3 findings %+v (PTS 乱序 must not be read as DTS regression)", c.ID, rep.Findings)
		}
		if code != policy.ExitPass {
			t.Errorf("%s: exit=%d, want 0", c.ID, code)
		}
	}
}

// 每条 fail 必带四元组（G1 的夹具级复核）：M3 的 fail 经渲染层校验后原样落地。
func TestM3FailFindingsCarryFullEvidence(t *testing.T) {
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
		run([]string{"audit", p, "--modules", "M3", "--json"}, &buf, &buf)
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
			if f.ByteOffset == 0 {
				t.Errorf("%s: fail must point at a real byte offset: %+v", c.ID, f)
			}
			if f.Delta.Unit == "" {
				t.Errorf("%s: fail delta lacks unit: %+v", c.ID, f.Delta)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no M3 fail findings checked — assertion is vacuous")
	}
}

// G9：单模块结论 == 集合里该模块的结论。全集因 M4/M5 未实现而 exit 3，
// 故用 `--modules M1,M2,M3` 与 `--modules M3` 对拍 M3 那一半。
func TestM3SubsetConclusionIsStable(t *testing.T) {
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
		alone := fingerprintsOfModule(t, p, "M3", "--modules", "M3")
		withOthers := fingerprintsOfModule(t, p, "M3", "--modules", "M1,M2,M3")
		if alone != withOthers {
			t.Errorf("%s: M3 conclusion depends on module selection: %q vs %q", c.ID, alone, withOthers)
		}
		again := fingerprintsOfModule(t, p, "M3", "--modules", "M3")
		if alone != again {
			t.Errorf("%s: --modules M3 not idempotent: %q vs %q", c.ID, alone, again)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no cases checked")
	}
}

func offsetKeys(m map[uint64]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, fmt.Sprintf("0x%X", k))
	}
	sort.Strings(out)
	return out
}