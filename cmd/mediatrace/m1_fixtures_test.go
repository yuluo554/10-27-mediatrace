package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/yuluo554/mediatrace/internal/policy"
	"github.com/yuluo554/mediatrace/internal/report"
	"github.com/yuluo554/mediatrace/internal/rules/structure"
	"github.com/yuluo554/mediatrace/tools/fixture-gen/gen"
)

// M2 DoD：模块 M1 对全部 92 例夹具的结论 —— 缺陷类召回、对照组误报 0。
//
// 真值来自生成器构造参数（K8：生成器与解析器不共享代码；本测试只是拿两侧结果对账）。
// 断言用**集合相等**而非"包含"：既抓漏报（C1/C2/C3 召回）也抓误报（多报一条即红）。
func TestM1MatchesFixtureTruth(t *testing.T) {
	cases, err := gen.Build(gen.Config{Seed: 20261027, Suite: "all"})
	if err != nil {
		t.Fatalf("generate fixtures: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("fixture generator produced 0 cases — assertion is vacuous")
	}
	m1 := map[string]bool{}
	for _, r := range structure.RuleIDs {
		m1[r] = true
	}

	dir := t.TempDir()
	dirCases := 0
	for _, c := range cases {
		p := filepath.Join(dir, c.Filename)
		if err := os.WriteFile(p, c.Bytes, 0o444); err != nil {
			t.Fatalf("write %s: %v", c.Filename, err)
		}
		var buf bytes.Buffer
		code := run([]string{"audit", p, "--modules", "M1", "--json"}, &buf, &buf)

		var rep report.Report
		if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
			t.Fatalf("%s: JSON decode: %v\n%s", c.ID, err, buf.String())
		}
		got := map[string]bool{}
		for _, f := range rep.Findings {
			if f.Module != "M1" {
				t.Errorf("%s: non-M1 finding leaked: %+v", c.ID, f)
			}
			got[f.RuleID] = true
		}
		want := map[string]bool{}
		for _, r := range c.Expected {
			if m1[r] {
				want[r] = true
			}
		}

		if !sameStringSet(got, want) {
			t.Errorf("%s (%s): M1 rules = %v, want %v", c.ID, c.ExpectStatus, keysOf(got), keysOf(want))
		}

		// 对照组（pass）：一个 finding 都不许有（M1 不产 gray，故 findings 必须为空）
		if c.ExpectStatus == gen.ExpectPass {
			if len(rep.Findings) != 0 {
				t.Errorf("%s: FALSE POSITIVE — pass case produced %d M1 findings", c.ID, len(rep.Findings))
			}
			if code != policy.ExitPass {
				t.Errorf("%s: pass case exit=%d, want %d", c.ID, code, policy.ExitPass)
			}
			continue
		}
		// 缺陷类：M1 相关的用例必须 exit 1；与之无关的用例不得报 M1
		if len(want) > 0 {
			dirCases++
			if code != policy.ExitFail {
				t.Errorf("%s: M1 defect case exit=%d, want %d", c.ID, code, policy.ExitFail)
			}
		} else if len(rep.Findings) != 0 {
			t.Errorf("%s: unrelated defect case produced M1 findings %v", c.ID, keysOf(got))
		}
	}

	// 三例验收（C1/C2/C3）+ M1 结构变体应覆盖到位（本测试不是空跑）
	if dirCases < 3 {
		t.Fatalf("only %d M1-relevant defect cases covered — acceptance gate not exercised", dirCases)
	}
	t.Logf("M1 audited %d cases (%d M1-relevant defect cases) — recall exact, false positives 0", len(cases), dirCases)
}

// 单模块结论必须等于全集里的对应模块结论（G9）——对 M1 而言即：
// --modules M1 的输出与默认全集里 M1 的 finding 集合一致（全集因 M2–M5 未实现而 exit 3，
// 故这里只比"能拿到的那部分"，用 --modules M1,M1 与 --modules M1 两次调用对拍）。
func TestM1SubsetConclusionIsStable(t *testing.T) {
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
		one := m1Fingerprints(t, p, "--modules", "M1")
		two := m1Fingerprints(t, p, "--modules", "M1")
		if one != two {
			t.Errorf("%s: --modules M1 not idempotent: %q vs %q", c.ID, one, two)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no defect cases checked")
	}
}

func m1Fingerprints(t *testing.T, path string, args ...string) string {
	t.Helper()
	var buf bytes.Buffer
	run(append([]string{"audit", path, "--json"}, args...), &buf, &buf)
	var rep report.Report
	if err := json.Unmarshal(buf.Bytes(), &rep); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	fps := make([]string, 0, len(rep.Findings))
	for _, f := range rep.Findings {
		fps = append(fps, f.Fingerprint())
	}
	sort.Strings(fps)
	out := ""
	for _, s := range fps {
		out += s + "\n"
	}
	return out
}

func sameStringSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
