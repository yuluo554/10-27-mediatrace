package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yuluo554/mediatrace/internal/policy"
	"github.com/yuluo554/mediatrace/internal/report"
	"github.com/yuluo554/mediatrace/internal/rules/display"
	"github.com/yuluo554/mediatrace/tools/fixture-gen/gen"
)

// M5 的误报门覆盖"所有不是 M5 目标的样本"：controls 16 + m3-regression 24 +
// m4 全 10 + m5 反例 5 + 其他模块专属缺陷——共 83 例，远超交接单要求的 60 例下限。
// M5 DoD：模块 M5（显示与色彩）对夹具集的结论 —— 缺陷命中、对照组 60 例误报 0。
//
// 断言用**集合相等**：既抓漏报（C7/C7b/D1/D2/M5P1..5），也抓误报。
// fail 额外钉住构造点已知的字节偏移；gray 钉住闭集 reason（M5 的 gray 分支由包级单测覆盖）。
func TestM5MatchesFixtureTruth(t *testing.T) {
	cases, err := gen.Build(gen.Config{Seed: 20261027, Suite: "all"})
	if err != nil {
		t.Fatalf("generate fixtures: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("fixture generator produced 0 cases — assertion is vacuous")
	}
	ruleset := map[string]bool{}
	for _, r := range display.RuleIDs {
		ruleset[r] = true
	}

	dir := t.TempDir()
	relevant, controls := 0, 0
	for _, c := range cases {
		p := filepath.Join(dir, c.Filename)
		if err := os.WriteFile(p, c.Bytes, 0o444); err != nil {
			t.Fatalf("write %s: %v", c.Filename, err)
		}
		var buf bytes.Buffer
		code := run([]string{"audit", p, "--modules", "M5", "--json"}, &buf, &buf)

		var rep report.Report
		if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
			t.Fatalf("%s: JSON decode: %v\n%s", c.ID, err, buf.String())
		}
		got := map[string]bool{}
		offsets := map[uint64]bool{}
		reasons := map[report.GrayReason]bool{}
		for _, f := range rep.Findings {
			if f.Module != "M5" {
				t.Errorf("%s: non-M5 finding leaked: %+v", c.ID, f)
			}
			got[f.RuleID] = true
			offsets[f.ByteOffset] = true
			if f.GrayReason != "" {
				reasons[f.GrayReason] = true
			}
		}

		want := map[string]bool{}
		for _, r := range c.Expected {
			if ruleset[r] {
				want[r] = true
			}
		}

		// 与 M5 无关的样本（含对照组、其他模块的缺陷、以及本模块的 pass 反例）都进误报门。
		if len(want) == 0 {
			controls++
			if len(rep.Findings) != 0 {
				t.Errorf("%s (%s): FALSE POSITIVE — non-M5 case produced M5 findings %v", c.ID, c.Suite, keysOf(got))
			}
			if code != policy.ExitPass {
				t.Errorf("%s: control case exit=%d, want %d", c.ID, code, policy.ExitPass)
			}
			continue
		}

		relevant++
		if !sameStringSet(got, want) {
			t.Errorf("%s (%s): M5 rules = %v, want %v", c.ID, c.Suite, keysOf(got), keysOf(want))
		}
		wantExit := policy.ExitFail
		if c.ExpectStatus == gen.ExpectGray {
			wantExit = policy.ExitInconclusive
			if !reasons[report.GrayReason(c.GrayReason)] {
				t.Errorf("%s: gray reason %q absent from M5 findings %v", c.ID, c.GrayReason, keysOf(got))
			}
		} else if c.Offset != 0 && !offsets[c.Offset] {
			t.Errorf("%s: expected byte offset 0x%X absent from M5 finding offsets %v", c.ID, c.Offset, offsetKeys(offsets))
		}
		if code != wantExit {
			t.Errorf("%s: exit=%d, want %d", c.ID, code, wantExit)
		}
	}

	if relevant < 5 {
		t.Fatalf("only %d M5-relevant cases covered — acceptance gate not exercised", relevant)
	}
	if controls < 60 {
		t.Fatalf("only %d control cases checked — false-positive gate not exercised (want >= 60)", controls)
	}
	t.Logf("M5 audited %d cases: %d M5-relevant defects hit, %d controls clean", len(cases), relevant, controls)
}

// 每条 fail 必带四元组（G1 的夹具级复核）。
func TestM5FailFindingsCarryFullEvidence(t *testing.T) {
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
		run([]string{"audit", p, "--modules", "M5", "--json"}, &buf, &buf)
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
		t.Fatal("no M5 fail findings checked — assertion is vacuous")
	}
}

// G9：单模块结论 == 合集里该模块的结论，且与默认全集一致。
func TestM5SubsetConclusionIsStable(t *testing.T) {
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
		alone := fingerprintsOfModule(t, p, "M5", "--modules", "M5")
		withOthers := fingerprintsOfModule(t, p, "M5", "--modules", "M1,M2,M3,M4,M5")
		full := fingerprintsOfModule(t, p, "M5")
		if alone != withOthers || alone != full {
			t.Errorf("%s: M5 conclusion depends on module selection: %q vs %q vs %q", c.ID, alone, withOthers, full)
		}
		again := fingerprintsOfModule(t, p, "M5", "--modules", "M5")
		if alone != again {
			t.Errorf("%s: --modules M5 not idempotent: %q vs %q", c.ID, alone, again)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no cases checked")
	}
}
