// Package archtest 是仓库级架构约束的**可跑红断言**（plan/03 §七 G7/G8、plan/06 R3）。
//
// 这些判据不在业务包里跑（避免污染），集中在 archtest 用 AST / 文件扫描实现。
package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot 从测试文件所在目录向上两级（internal/archtest → repo root）。
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

// relSlash 返回使用 `/` 分隔的相对路径（Windows 下 filepath.Rel 给的是 `\`）。
func relSlash(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

// walkGoFiles 遍历 srcDir 下所有 .go 文件（跳过测试文件由调用方决定）。
func walkGoFiles(t *testing.T, dir string, fn func(path string)) {
	t.Helper()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "testdata" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			fn(path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
}

// G7: internal/ 与 cmd/ 零网络 import。
func TestG7NoNetworkImports(t *testing.T) {
	root := repoRoot(t)
	banned := map[string]bool{
		"net": true, "net/http": true, "net/url": true,
		"net/rpc": true, "net/smtp": true, "net/httptrace": true,
	}
	fset := token.NewFileSet()
	checked := 0
	for _, dir := range []string{filepath.Join(root, "internal"), filepath.Join(root, "cmd")} {
		walkGoFiles(t, dir, func(path string) {
			f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			checked++
			for _, imp := range f.Imports {
				p := strings.Trim(imp.Path.Value, `"`)
				if banned[p] {
					rel := relSlash(root, path)
					t.Errorf("%s imports banned network package %q", rel, p)
				}
			}
		})
	}
	if checked == 0 {
		t.Fatal("G7 scanned 0 Go files — assertion is vacuous")
	}
	t.Logf("G7 scanned %d Go files", checked)
}

// G8: go.mod 无 require 块；go.sum 不存在。
func TestG8ZeroThirdPartyDeps(t *testing.T) {
	root := repoRoot(t)
	modPath := filepath.Join(root, "go.mod")
	raw, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, "require (") || strings.Contains(text, "require ") {
		// 允许注释里的说明；只拒真 require 指令行
		for _, line := range strings.Split(text, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "require ") || trimmed == "require (" {
				t.Errorf("go.mod has third-party requirement: %q", trimmed)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(root, "go.sum")); err == nil {
		t.Fatal("go.sum exists — project must have zero third-party dependencies")
	}
}

// R3: 口径常量只有一个实现处（internal/policy）。
func TestR3PolicyConstantsHaveSingleSource(t *testing.T) {
	root := repoRoot(t)
	names := []string{"TickTolerance", "TimescaleSuspectBelow", "MatrixOrthoTolerance", "DTSRegressionTolerance"}
	fset := token.NewFileSet()
	for _, name := range names {
		hits := 0
		walkGoFiles(t, root, func(path string) {
			if strings.HasSuffix(path, "_test.go") {
				return
			}
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			ast.Inspect(f, func(n ast.Node) bool {
				// 只统计"定义处"：const 声明里出现该名字
				gs, ok := n.(*ast.GenDecl)
				if !ok || gs.Tok != token.CONST {
					return true
				}
				for _, spec := range gs.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, id := range vs.Names {
						if id.Name == name {
							hits++
							rel := relSlash(root, path)
							if !strings.Contains(rel, "internal/policy") {
								t.Errorf("%s defined outside internal/policy", name)
							}
						}
					}
				}
				return true
			})
		})
		if hits == 0 {
			t.Errorf("%s defined 0 times — policy constant missing", name)
		}
	}
}

// 模块之间不得互相 import（plan/03 §1.2）。
func TestModulesDoNotImportEachOther(t *testing.T) {
	root := repoRoot(t)
	// rules 层当前是单包（rules.go 注册表）；一旦拆成 rules/<module> 子包，
	// 这条断言会检查 sibling import。
	pkgDir := filepath.Join(root, "internal", "rules")
	fset := token.NewFileSet()
	checked := 0
	walkGoFiles(t, pkgDir, func(path string) {
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		checked++
		rel := relSlash(root, path)
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(p, "internal/rules/") && !strings.HasSuffix(rel, "/rules.go") {
				t.Errorf("%s imports sibling rules package %q (modules must not depend on each other)", rel, p)
			}
		}
	})
	if checked == 0 {
		t.Fatal("rules package scanned 0 files")
	}
}
