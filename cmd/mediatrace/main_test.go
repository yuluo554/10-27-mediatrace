package main

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/yuluo554/mediatrace/internal/policy"
)

// mkFile 写一个最小合法 MP4 结构到临时文件并返回路径。
func mkFile(t *testing.T, payload []byte) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "sample.mp4")
	if err := os.WriteFile(p, payload, 0o444); err != nil {
		t.Fatal(err)
	}
	return p
}

func boxBytes(typ string, payload []byte) []byte {
	buf := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(buf[0:4], uint32(8+len(payload)))
	copy(buf[4:8], typ)
	copy(buf[8:], payload)
	return buf
}

func runCaptured(t *testing.T, args ...string) int {
	t.Helper()
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer devnull.Close()
	return run(args, devnull, devnull)
}

func TestVersionExitZero(t *testing.T) {
	if code := runCaptured(t, "--version"); code != policy.ExitPass {
		t.Fatalf("--version exit=%d, want %d", code, policy.ExitPass)
	}
}

func TestNoArgsIsUsageError(t *testing.T) {
	if code := runCaptured(t); code != policy.ExitUsage {
		t.Fatalf("no args exit=%d, want %d", code, policy.ExitUsage)
	}
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	if code := runCaptured(t, "frobnicate"); code != policy.ExitUsage {
		t.Fatalf("unknown cmd exit=%d, want %d", code, policy.ExitUsage)
	}
}

func TestAuditMissingFileIsUsageError(t *testing.T) {
	if code := runCaptured(t, "audit", filepath.Join(t.TempDir(), "nope.mp4")); code != policy.ExitUsage {
		t.Fatalf("missing file exit=%d, want %d", code, policy.ExitUsage)
	}
}

func TestAuditUnparsableFileIsUsageError(t *testing.T) {
	// 连根盒头都读不出（< 8 字节）→ 树无法建立 → 退出码 3。
	// 注意：size<8 的**完整盒头**不是 parse error，而是结构 finding（too_small_size），
	// 见 internal/isobmff 与 internal/rules/structure。
	p := mkFile(t, []byte{0, 0, 0})
	if code := runCaptured(t, "audit", p); code != policy.ExitUsage {
		t.Fatalf("truncated header exit=%d, want %d (parse error is NOT gray)", code, policy.ExitUsage)
	}
}

func TestAuditUnknownModuleIsUsageError(t *testing.T) {
	p := mkFile(t, boxBytes("ftyp", []byte("isom")))
	if code := runCaptured(t, "audit", p, "--modules", "M9"); code != policy.ExitUsage {
		t.Fatalf("unknown module exit=%d, want %d", code, policy.ExitUsage)
	}
}

// 五个模块全部实现后，默认全集对"只有 ftyp"的最小样本应给出真实结论 pass(0)，
// 不再有"未实现 → 退出码 3"的骨架期通道。
func TestDefaultAuditRunsAllModules(t *testing.T) {
	p := mkFile(t, boxBytes("ftyp", []byte("isom")))
	if code := runCaptured(t, "audit", p); code != policy.ExitPass {
		t.Fatalf("audit exit=%d, want %d (all modules implemented; minimal ftyp-only file is clean)", code, policy.ExitPass)
	}
}

// G10: 审计不改输入文件字节。
func TestAuditDoesNotModifyInput(t *testing.T) {
	data := boxBytes("ftyp", []byte("isom"))
	p := mkFile(t, data)
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	runCaptured(t, "audit", p)
	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("input file bytes changed by audit")
	}
}

// 文档写的形态必须能被 parser 接受：`audit <file> --json` 与 `audit --json <file>`。
// 这是发布包实测里最贵的一族（skill：「文档化的参数形态 parser 认不认」）。
//
// 五个模块全部实现后，默认全集对最小合法样本给出 pass(0)；只跑单模块同样是 pass(0)。
// 两条路径都断言，正是为了区分"参数形态没被认出来"（会是 3）与"真实结论"（0）。
func TestFlagOrderDoesNotMatter(t *testing.T) {
	p := mkFile(t, boxBytes("ftyp", []byte("isom")))
	cases := []struct {
		args []string
		want int
	}{
		// 默认全集：五模块全实现，只有 ftyp 的最小样本结构/时长/时间戳/偏移/显示均无异常 → pass(0)
		{[]string{"audit", p, "--json"}, policy.ExitPass},
		{[]string{"audit", "--json", p}, policy.ExitPass},
		// 只跑 M1：合法的"只有 ftyp"文件结构无异常 → pass(0)
		{[]string{"audit", p, "--modules", "M1"}, policy.ExitPass},
		{[]string{"audit", "--modules", "M1", p}, policy.ExitPass},
		{[]string{"audit", p, "--json", "--modules", "M1"}, policy.ExitPass},
	}
	for _, c := range cases {
		code := runCaptured(t, c.args...)
		if code != c.want {
			t.Errorf("%v exit=%d want %d", c.args, code, c.want)
		}
	}
}

// 直接测 parseInterspersed：位置参数与旗标任意交织都要被正确收集。
func TestParseInterspersed(t *testing.T) {
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"f.mp4", "--json"}, []string{"f.mp4"}},
		{[]string{"--json", "f.mp4"}, []string{"f.mp4"}},
		{[]string{"f.mp4"}, []string{"f.mp4"}},
		{[]string{"--modules", "M1,M3", "f.mp4"}, []string{"f.mp4"}},
		{[]string{"f.mp4", "--modules", "M1", "--json"}, []string{"f.mp4"}},
		{[]string{"a", "b"}, []string{"a", "b"}},
		{nil, nil},
	}
	for _, c := range cases {
		opts := newAuditOpts(io.Discard)
		got, err := parseInterspersed(opts.fs, c.args)
		if err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		if len(got) != len(c.want) {
			t.Fatalf("%v: got %v want %v", c.args, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%v: got %v want %v", c.args, got, c.want)
			}
		}
	}
}

func TestAuditRejectsTwoPositionalArgs(t *testing.T) {
	p := mkFile(t, boxBytes("ftyp", []byte("isom")))
	if code := runCaptured(t, "audit", p, p); code != policy.ExitUsage {
		t.Fatalf("two paths exit=%d want %d", code, policy.ExitUsage)
	}
}

// 阳性对照：证明上面那条测试不是空的。
//
// 用**朴素解析**（直接 fs.Parse，不交织）跑同一批 args，断言它确实把
// `--json` 当成了位置参数——也就是说"如果实现退化成朴素解析，TestParseInterspersed
// 一定会红"。没有这条对照，交错的测试可能只是碰巧通过（例如构造的 args 恰好都合法）。
func TestNaiveParseWouldFailTheRegression(t *testing.T) {
	naive := func(args []string) (pos []string, jsonSeen bool) {
		opts := newAuditOpts(io.Discard)
		if err := opts.fs.Parse(args); err != nil {
			return nil, false
		}
		return opts.fs.Args(), *opts.asJSON
	}
	// 文档形态：文件在前、旗标在后 → 朴素解析把 --json 留在位置参数里
	pos, jsonSeen := naive([]string{"f.mp4", "--json"})
	if jsonSeen {
		t.Fatal("positive control invalid: naive parse unexpectedly saw --json")
	}
	if len(pos) != 2 || pos[1] != "--json" {
		t.Fatalf("positive control invalid: naive parse gave %v", pos)
	}
	// 而交错解析给出正确结果
	opts := newAuditOpts(io.Discard)
	got, err := parseInterspersed(opts.fs, []string{"f.mp4", "--json"})
	if err != nil || len(got) != 1 || got[0] != "f.mp4" || !*opts.asJSON {
		t.Fatalf("interspersed parse wrong: %v %v json=%v", got, err, *opts.asJSON)
	}
}
