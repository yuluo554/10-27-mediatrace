package gen

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// T2：同 seed 双跑逐字节一致（确定性的最小断言）。
func TestBuildIsDeterministic(t *testing.T) {
	a, err := Build(Config{Seed: 20261027, Suite: "all"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build(Config{Seed: 20261027, Suite: "all"})
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) || len(a) == 0 {
		t.Fatalf("case count mismatch or empty: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatalf("case order drift at %d: %s vs %s", i, a[i].ID, b[i].ID)
		}
		if string(a[i].Bytes) != string(b[i].Bytes) {
			t.Fatalf("bytes drift for %s", a[i].ID)
		}
		sum := sha256.Sum256(a[i].Bytes)
		if hex.EncodeToString(sum[:]) != a[i].SHA256 {
			t.Fatalf("%s: recorded sha256 does not match bytes", a[i].ID)
		}
	}
}

// 子集 suite 的结论必须与全集里的同 ID 样本逐字节一致（K9 的生成侧对应）。
func TestSuiteSubsetIsConsistentWithAll(t *testing.T) {
	all, err := Build(Config{Seed: 20261027, Suite: "all"})
	if err != nil {
		t.Fatal(err)
	}
	index := map[string]Case{}
	for _, c := range all {
		index[c.ID] = c
	}
	for _, suite := range []string{"defects", "controls", "m3-regression", "m4", "m5"} {
		sub, err := Build(Config{Seed: 20261027, Suite: suite})
		if err != nil {
			t.Fatal(err)
		}
		if len(sub) == 0 {
			t.Errorf("suite %s produced 0 cases", suite)
			continue
		}
		for _, c := range sub {
			ref, ok := index[c.ID]
			if !ok {
				t.Errorf("suite %s case %s missing from all", suite, c.ID)
				continue
			}
			if c.SHA256 != ref.SHA256 {
				t.Errorf("suite %s case %s bytes drift vs all", suite, c.ID)
			}
		}
	}
}

// 未知 suite 不得静默返回非空集（空套件冒充失败）。
func TestUnknownSuiteProducesNoCases(t *testing.T) {
	cases, err := Build(Config{Seed: 1, Suite: "does-not-exist"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 0 {
		t.Fatalf("unknown suite produced %d cases", len(cases))
	}
}

// 每条真值必须完整：pass 组不得带 Expected；fail/gray 组必须带 Expected，gray 必带闭集 reason。
func TestEveryBuilderDeclaresCompleteTruth(t *testing.T) {
	validReasons := map[string]bool{
		"vfr_detected": true, "no_roll_distance": true, "unknown_codec_priming": true,
		"media_rate_not_one": true, "no_edts": true, "hdr_metadata_missing": true,
		"timescale_suspect": true, "duration_unknown": true,
	}
	for _, b := range builders {
		switch b.ExpectStatus {
		case ExpectPass:
			if len(b.Expected) != 0 {
				t.Errorf("%s: pass case must declare no expected findings", b.ID)
			}
			if b.GrayReason != "" {
				t.Errorf("%s: pass case must not declare a gray reason", b.ID)
			}
		case ExpectFail, ExpectGray:
			if len(b.Expected) == 0 {
				t.Errorf("%s: %s case declares no expected findings", b.ID, b.ExpectStatus)
			}
		default:
			t.Errorf("%s: unknown expect_status %q", b.ID, b.ExpectStatus)
		}
		if b.ExpectStatus == ExpectGray {
			if !validReasons[b.GrayReason] {
				t.Errorf("%s: gray reason %q not in closed set", b.ID, b.GrayReason)
			}
		} else if b.GrayReason != "" {
			t.Errorf("%s: only gray cases may declare a gray reason", b.ID)
		}
		if b.Note == "" {
			t.Errorf("%s has no note", b.ID)
		}
		if !strings.HasSuffix(b.Filename, ".mp4") {
			t.Errorf("%s filename %q is not .mp4", b.ID, b.Filename)
		}
	}
}

// 分类门槛（计数据由集合算出，不落易漂移常量）：
//   - 合成缺陷集（defects）≥25 例，且 C1–C7 每类 ≥1；
//   - controls 每类至少 1 个合法样本；
//   - m3-regression ≥20 且含 B 帧（ctts 有偏移）；
//   - m4 / m5 正反例各 5。
func TestSuiteThresholdsHoldByConstruction(t *testing.T) {
	all, err := Build(Config{Seed: 20261027, Suite: "all"})
	if err != nil {
		t.Fatal(err)
	}
	bySuite := map[string][]Case{}
	for _, c := range all {
		bySuite[c.Suite] = append(bySuite[c.Suite], c)
	}

	if n := len(bySuite["defects"]); n < 25 {
		t.Errorf("defects count = %d, want >= 25", n)
	}
	for _, class := range []string{"C1", "C2", "C3", "C4", "C5", "C6", "C7"} {
		found := false
		for _, c := range bySuite["defects"] {
			if strings.HasPrefix(c.ID, class) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("defects suite has no case of class %s", class)
		}
	}

	if n := len(bySuite["controls"]); n < 7 {
		t.Errorf("controls count = %d, want >= 7 (one legal sample per class)", n)
	}

	m3 := bySuite["m3-regression"]
	if len(m3) < 20 {
		t.Errorf("m3-regression count = %d, want >= 20", len(m3))
	}
	// B 帧覆盖：至少 1 例 ctts v0 正偏移，至少 1 例 v1 有符号（负偏移）
	v0, v1 := 0, 0
	for _, b := range m3RegressionBuilders {
		if b.ExpectStatus != ExpectPass {
			continue
		}
		switch b.BuildShape {
		case "ctts-v0-bframes":
			v0++
		case "ctts-v1-signed":
			v1++
		}
	}
	if v0 == 0 {
		t.Error("m3-regression has no ctts v0 B-frame case")
	}
	if v1 == 0 {
		t.Error("m3-regression has no ctts version=1 case — negative offsets would go untested")
	}

	for _, suite := range []string{"m4", "m5"} {
		pos, neg := 0, 0
		for _, c := range bySuite[suite] {
			if c.ExpectStatus == ExpectPass {
				neg++
			} else {
				pos++
			}
		}
		if pos != 5 || neg != 5 {
			t.Errorf("%s: positives=%d negatives=%d, want 5/5", suite, pos, neg)
		}
	}

	// 每个 ID 与文件名唯一
	ids := map[string]bool{}
	names := map[string]bool{}
	for _, c := range all {
		if ids[c.ID] {
			t.Errorf("duplicate case id %s", c.ID)
		}
		ids[c.ID] = true
		if names[c.Filename] {
			t.Errorf("duplicate filename %s", c.Filename)
		}
		names[c.Filename] = true
	}
}

// 期望偏移必须落在真实字节内（构造点真值不得空跑）。
func TestExpectedOffsetPointsAtDeclaredField(t *testing.T) {
	all, err := Build(Config{Seed: 20261027, Suite: "all"})
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, c := range all {
		if c.ExpectStatus == ExpectPass || c.Offset == 0 {
			continue
		}
		if c.Offset >= uint64(len(c.Bytes)) {
			t.Errorf("%s: expected offset 0x%X beyond file size %d", c.ID, c.Offset, len(c.Bytes))
			continue
		}
		checked++
	}
	if checked == 0 {
		t.Error("no fail/gray case declares an expected offset — the construction-time truth is vacuous")
	}
	t.Logf("checked %d construction-time offsets", checked)
}

// 顶层盒流自洽（生成侧自证）：除声明过"截断/子盒越界"的样本外，
// 顶层盒的 size 必须落在文件内且不重叠。
func TestAllSamplesParseAsBoxStream(t *testing.T) {
	all, err := Build(Config{Seed: 20261027, Suite: "all"})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range all {
		off := 0
		for off+8 <= len(c.Bytes) {
			size := uint64(binary.BigEndian.Uint32(c.Bytes[off : off+4]))
			if size == 1 {
				if off+16 > len(c.Bytes) {
					break
				}
				size = binary.BigEndian.Uint64(c.Bytes[off+8 : off+16])
			} else if size == 0 {
				size = uint64(len(c.Bytes) - off)
			}
			if size < 8 || uint64(off)+size > uint64(len(c.Bytes)) {
				// 截断类允许提前结束；其余视为生成缺陷
				if !strings.HasPrefix(c.ID, "C3") {
					t.Errorf("%s: top-level box at 0x%X size=%d overruns file (%d bytes)", c.ID, off, size, len(c.Bytes))
				}
				break
			}
			off += int(size)
		}
	}
}

// plan/04 §二纪律 3：生成器不得读时钟 / 环境变量 / 起外部进程。
func TestGeneratorHasNoClockEnvOrExec(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// 生成器目录 = tools/fixture-gen（本包在 gen/ 下）
	root := filepath.Clean(filepath.Join(wd, ".."))
	banned := map[string]string{
		"time": "read clock", "os/exec": "spawn external process", "math/rand": "non-seeded randomness",
	}
	fset := token.NewFileSet()
	checked := 0
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		checked++
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if why, bad := banned[p]; bad {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s imports %q (%s) — violates determinism discipline", rel, p, why)
			}
			if p == "os" {
				src, rerr := os.ReadFile(path)
				if rerr == nil && strings.Contains(string(src), "os.Getenv") {
					rel, _ := filepath.Rel(root, path)
					t.Errorf("%s calls os.Getenv — violates determinism discipline", rel)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("scanned 0 generator Go files — assertion is vacuous")
	}
	t.Logf("scanned %d generator files", checked)
}

var _ = ast.Inspect // 保持 ast 导入（后续扩检测时用）