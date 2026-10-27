package gen

import (
	"crypto/sha256"
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

// 未知 suite 不得静默返回空集（空集冒充失败）。
func TestUnknownSuiteProducesNoCases(t *testing.T) {
	cases, err := Build(Config{Seed: 1, Suite: "does-not-exist"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 0 {
		t.Fatalf("unknown suite produced %d cases", len(cases))
	}
}

// 每个构造器的真值必须非空（缺陷样本必须声明期望）。
func TestEveryBuilderDeclaresExpectation(t *testing.T) {
	for _, b := range builders {
		if len(b.Expected) == 0 {
			t.Errorf("%s declares no expected findings", b.ID)
		}
		if b.Note == "" {
			t.Errorf("%s has no note", b.ID)
		}
		if !strings.HasSuffix(b.Filename, ".mp4") {
			t.Errorf("%s filename %q is not .mp4", b.ID, b.Filename)
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
				// os 允许用于写文件，但不得读环境变量
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
