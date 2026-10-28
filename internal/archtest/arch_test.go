// Package archtest 是仓库级架构约束的**可跑红断言**（plan/03 §七 G7/G8、plan/06 R3）。
//
// 这些判据不在业务包里跑（避免污染），集中在 archtest 用 AST / 文件扫描实现。
package archtest

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/yuluo554/10-27-mediatrace/internal/policy"
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

// 版本号两处定义必须一致：`internal/policy.Version`（源码内可见的默认值）与
// `tools/tasks.mjs` 的 VERSION（构建时由 -ldflags 注入、覆盖前者）。
//
// 为什么这条要机器断言而不是靠人看：发布二进制用 `--version` 自证 tag，而注入值来自
// tasks.mjs；两处漂移时 `--version` 会打印一个与 tag 不同的号，且**本地全绿**——
// 只有拿到产物的人才会发现（阶段 7 出口门「版本号一致要由机器断言，别靠手抄」）。
// 本判据只断言"一致"：开发期带 `-dev` 后缀合法，收干由发布流程管。
func TestVersionSingleSourceMatchesBuildScript(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "tools", "tasks.mjs"))
	if err != nil {
		t.Fatalf("read tools/tasks.mjs: %v", err)
	}
	const marker = `const VERSION = "`
	src := string(raw)
	idx := strings.Index(src, marker)
	if idx < 0 {
		t.Fatalf("tools/tasks.mjs 里找不到 %q —— 版本注入点被搬走或改写", marker)
	}
	rest := src[idx+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		t.Fatal("tools/tasks.mjs 的 VERSION 字面量没有收尾引号")
	}
	if got := rest[:end]; policy.Version != got {
		t.Errorf("版本不一致：internal/policy.Version=%q，tools/tasks.mjs VERSION=%q——发布物 --version 会与 tag 对不上", policy.Version, got)
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

// ── 自述数字守门（诊断 D5）──────────────────────────────────────────────
//
// 为什么要有这两条：README / docs 里的「92 例」「32 条规则」当前全对，但**没有任何门守着**。
// 覆盖关系是当时手工 grep 出来的，下一棒加一条规则或加一个夹具，数字就悄悄漂了，
// 而所有既有测试仍然全绿（它们都不读文档）。判据必须机器化，且**不许写常量白名单**：
// 两个数每次都从代码与 manifest 现取，所以"改常量让它变绿"这条路不存在。
//
// 两条都带阴性对照：抽到 0 条规则 / README 里一处「N 例」都没有 → 直接 FAIL。
// 否则"扫描没找到东西"会被输出成"全部一致"。

// TestRulesDocCoversAllRuleIDs：每个模块声明的 RuleIDs 必须逐条出现在 docs/rules.md。
//
// 抽取方式是 AST 读 `var RuleIDs = []string{...}`（不是正则扫源码），这样：
//   - 注释里的同名文字不会被误当规则；
//   - `_test.go` 里的对账表不参与（真值只在实现包里声明）。
//
// 阴性对照：抽到 0 条 → FAIL（否则 docs 里什么都不写也能过）。
func TestRulesDocCoversAllRuleIDs(t *testing.T) {
	root := repoRoot(t)
	docRaw, err := os.ReadFile(filepath.Join(root, "docs", "rules.md"))
	if err != nil {
		t.Fatalf("read docs/rules.md: %v", err)
	}
	doc := string(docRaw)

	fset := token.NewFileSet()
	ids := []string{}
	decls := 0
	walkGoFiles(t, filepath.Join(root, "internal", "rules"), func(path string) {
		if strings.HasSuffix(path, "_test.go") {
			return
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			gs, ok := n.(*ast.GenDecl)
			if !ok || gs.Tok != token.VAR {
				return true
			}
			for _, spec := range gs.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, id := range vs.Names {
					if id.Name != "RuleIDs" || i >= len(vs.Values) {
						continue
					}
					lit, ok := vs.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					decls++
					for _, elt := range lit.Elts {
						bl, ok := elt.(*ast.BasicLit)
						if !ok || bl.Kind != token.STRING {
							continue
						}
						val := strings.Trim(bl.Value, `"`)
						ids = append(ids, val)
					}
				}
			}
			return true
		})
	})

	if decls == 0 || len(ids) == 0 {
		t.Fatalf("RuleIDs 抽到 0 条（声明块 %d 个）—— 判据是空的：抽取点被搬走或改名了", decls)
	}
	for _, id := range ids {
		if !strings.Contains(doc, id) {
			t.Errorf("docs/rules.md 缺规则 %q（实现里有、文档里没有）", id)
		}
	}
	t.Logf("docs/rules.md 覆盖 %d 条 RuleID（来自 %d 个声明块）", len(ids), decls)
}

// TestFixtureCountDocumented：README 里每一处「N 例」都必须等于 manifest 的用例数。
//
// 数字来源是 `fixtures/out/manifest.json` 的 `len(cases)`（**不是** `ls fixtures/out | wc -l`：
// 那里有 93 个文件 = 92 例 + manifest.json 自身，诊断 D1 附近踩过这个坑）。
//
// 阴性对照：README 里一处「N 例」都没有 → FAIL（否则删掉那句话也能过）。
func TestFixtureCountDocumented(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "fixtures", "out", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest.json: %v", err)
	}
	var manifest struct {
		Cases []json.RawMessage `json:"cases"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode manifest.json: %v", err)
	}
	want := len(manifest.Cases)
	if want == 0 {
		t.Fatal("manifest.json 有 0 条用例 —— 判据是空的")
	}

	readmeRaw, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	re := regexp.MustCompile(`(\d+)\s*例`)
	matches := re.FindAllStringSubmatch(string(readmeRaw), -1)
	if len(matches) == 0 {
		t.Fatal("README 里一处「N 例」都没有 —— 判据是空的（数字被删或被改写单位）")
	}
	drift := 0
	for _, m := range matches {
		got, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("解析 %q: %v", m[0], err)
		}
		if got != want {
			drift++
			t.Errorf("README 写「%s」，manifest 实际 %d 例 —— 自述数字已漂移", m[0], want)
		}
	}
	// 汇总行只在确实一致时打印：t.Errorf 不中断，否则失败输出里也会出现"全部等于"，误导读者。
	if drift == 0 {
		t.Logf("README 的 %d 处「N 例」全部等于 manifest 的 %d 例", len(matches), want)
	}
}
