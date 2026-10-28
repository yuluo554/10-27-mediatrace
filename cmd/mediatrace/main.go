// Command mediatrace 是媒体时间轴语义审计器。
//
// 用法：
//
//	mediatrace audit <file> [--json] [--modules M1,M3]
//	mediatrace --version
//
// 退出码见 data/README.md §5.6（0 pass / 1 fail / 2 inconclusive / 3 usage）。
// 隐私：纯本地、零网络；报告默认不写绝对路径。
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuluo554/10-27-mediatrace/internal/isobmff"
	"github.com/yuluo554/10-27-mediatrace/internal/policy"
	"github.com/yuluo554/10-27-mediatrace/internal/report"
	"github.com/yuluo554/10-27-mediatrace/internal/rules"
)

// version 可被 -ldflags "-X main.version=..." 覆盖。
var version = policy.Version

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run 是可测试的入口（进程级退出码测试与单元测试共用）。
// stdout/stderr 用 io.Writer，便于测试捕获输出（注入 bytes.Buffer）。
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return policy.ExitUsage
	}
	switch args[0] {
	case "--version", "-v", "version":
		fmt.Fprintf(stdout, "mediatrace %s\n", version)
		return policy.ExitPass
	case "--help", "-h", "help":
		usage(stdout)
		return policy.ExitPass
	case "audit":
		return runAudit(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "mediatrace: unknown command %q\n", args[0])
		usage(stderr)
		return policy.ExitUsage
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `mediatrace — 媒体时间轴语义审计（只读字节、不解码、零网络）

Usage:
  mediatrace audit <file> [flags]
  mediatrace --version

Flags (audit):
  --json            输出 JSON（默认人读 Markdown 摘要）
  --modules M1,M3   只跑指定模块（默认 M1–M5）

Exit codes (data/README §5.6):
  0 pass   1 fail   2 inconclusive   3 usage/parse error
`)
}

// auditOpts 是 audit 子命令的旗标集合（构造与解析分离，便于单测参数面）。
type auditOpts struct {
	fs      *flag.FlagSet
	asJSON  *bool
	modules *string
}

func newAuditOpts(output io.Writer) *auditOpts {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.SetOutput(output)
	return &auditOpts{
		fs:      fs,
		asJSON:  fs.Bool("json", false, "emit JSON"),
		modules: fs.String("modules", "", "comma-separated module ids (M1..M5)"),
	}
}

func runAudit(args []string, stdout, stderr io.Writer) int {
	opts := newAuditOpts(stderr)
	fs, asJSON, modules := opts.fs, opts.asJSON, opts.modules

	// Go 的 flag 包遇到第一个位置参数就停止解析，于是 `audit <file> --json`
	// 会把 `--json` 当成第二个位置参数（本机实测：文档写的形态直接 exit 3）。
	// 这里做"旗标可出现在位置参数之后"的归一化：逐段解析，把每个位置参数收起来，
	// 再继续解析它后面的旗标。
	positional, err := parseInterspersed(fs, args)
	if err != nil {
		return policy.ExitUsage
	}
	if len(positional) != 1 {
		fmt.Fprintln(stderr, "mediatrace audit: exactly one file path is required")
		return policy.ExitUsage
	}
	path := positional[0]

	data, err := os.ReadFile(path) // 只读打开
	if err != nil {
		fmt.Fprintf(stderr, "mediatrace: cannot read %s: %v\n", filepath.Base(path), err)
		return policy.ExitUsage
	}

	root, err := isobmff.Parse(data)
	if err != nil {
		// 解析失败是 error，不是 gray（plan/00 D11）。两条通道都归退出码 3：
		//   · ErrNotISOBMFF —— 首盒子不像 ISO BMFF（纯文本 / MKV / 随机字节）：
		//     "看不懂"不等于"文件有问题"，绝不能落进判定通道给出 FAIL(1)。
		//   · ErrShortHeader / 其它 —— 剩余字节不足以构盒头。
		// 措辞只用 basename（不泄漏绝对路径）。
		if errors.Is(err, isobmff.ErrNotISOBMFF) {
			fmt.Fprintf(stderr, "mediatrace: not an ISO BMFF file: %s (%v)\n", filepath.Base(path), err)
			return policy.ExitUsage
		}
		fmt.Fprintf(stderr, "mediatrace: parse error in %s: %v\n", filepath.Base(path), err)
		return policy.ExitUsage
	}

	selected, err := rules.ByIDs(splitCSV(*modules))
	if err != nil {
		fmt.Fprintf(stderr, "mediatrace: %v\n", err)
		return policy.ExitUsage
	}

	in := rules.Input{FileName: filepath.Base(path), Root: root, Raw: data}
	all := make([]report.Finding, 0, 64)
	ids := make([]string, 0, len(selected))
	for _, m := range selected {
		ids = append(ids, m.ID())
		found, err := m.Audit(in)
		if err != nil {
			if errors.Is(err, rules.ErrNotImplemented) {
				// 五个模块均已实现；此分支保留为防御（若将来新增模块忘接实现，仍不得冒充 PASS）。
				fmt.Fprintf(stderr, "mediatrace: module %s not implemented yet\n", m.ID())
				return policy.ExitUsage
			}
			fmt.Fprintf(stderr, "mediatrace: module %s failed: %v\n", m.ID(), err)
			return policy.ExitUsage
		}
		all = append(all, found...)
	}

	rep := report.Report{
		Tool: "mediatrace", Version: version,
		File: filepath.Base(path), Modules: ids, Findings: all,
	}
	if err := rep.ValidateAll(); err != nil {
		// 渲染层拒绝不合格 finding（G1）
		fmt.Fprintf(stderr, "mediatrace: internal finding validation failed: %v\n", err)
		return policy.ExitUsage
	}

	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rep); err != nil {
			fmt.Fprintf(stderr, "mediatrace: encode: %v\n", err)
			return policy.ExitUsage
		}
	} else {
		fmt.Fprint(stdout, rep.RenderMarkdown())
	}
	return policy.ExitCodeFor(rep.Summary.Fail, rep.Summary.Gray)
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseInterspersed 允许旗标与位置参数任意交织（`audit --json f.mp4` 与
// `audit f.mp4 --json` 等价）。返回位置参数列表。
//
// 实现要点：每次把剩余 args 交给 fs.Parse，它会在第一个非旗标处停下；
// 取出那个位置参数，再从它之后继续 —— 但必须先"记住"它，否则下一轮又会看到它。
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}
