// Command fixture-gen 程序化构造 C1–C7 缺陷夹具与正常对照组。
//
// 纪律（plan/04 §二、data/README §4）：
//   - 不读系统时间、不读环境变量、不调用 ffmpeg/MP4Box（零外部进程）；
//   - 同 --seed 双跑逐字节一致；
//   - 真值只落"构造参数 + 期望 finding 集合"，不落实现算出来的数。
//
// 骨架期：命令行与 --check 契约已就位，C1–C7 构造器在 gen/ 下逐类实现。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/yuluo554/mediatrace/tools/fixture-gen/gen"
)

// manifestEntry 是 manifest.json 的一行（真值标签）。
type manifestEntry struct {
	CaseID   string   `json:"case_id"`
	Suite    string   `json:"suite"`
	Filename string   `json:"filename"`
	SHA256   string   `json:"sha256"`
	Size     int      `json:"size_bytes"`
	Seed     uint64   `json:"seed"`
	Expected []string `json:"expected_findings"` // 规则 ID 列表；空 = 期望 pass
	Note     string   `json:"note,omitempty"`
}

type manifest struct {
	Generator string          `json:"generator"`
	Version   string          `json:"version"`
	Seed      uint64          `json:"seed"`
	Cases     []manifestEntry `json:"cases"`
}

func main() {
	out := flag.String("out", "fixtures/out", "output directory")
	seed := flag.Uint64("seed", 20261027, "seed for all deterministic choices")
	suite := flag.String("suite", "all", "all|defects|controls|m3-regression|m4|m5")
	check := flag.Bool("check", false, "regenerate to a temp dir and compare byte-for-byte with the committed manifest")
	list := flag.Bool("list", false, "print case ids and sha256 only")
	flag.Parse()

	cases, err := gen.Build(gen.Config{Seed: *seed, Suite: *suite})
	if err != nil {
		fail(err)
	}
	if len(cases) == 0 {
		fail(fmt.Errorf("no cases produced for suite %q (an empty suite is not a pass)", *suite))
	}

	if *list {
		for _, c := range cases {
			fmt.Printf("%s\t%s\t%d\n", c.ID, c.SHA256, len(c.Bytes))
		}
		return
	}

	if *check {
		if err := checkCommitted(cases, *out); err != nil {
			fail(err)
		}
		fmt.Printf("fixture-gen --check: %d 例与入库 manifest 逐字节一致\n", len(cases))
		return
	}

	if err := writeAll(cases, *out, *seed); err != nil {
		fail(err)
	}
	fmt.Printf("fixture-gen: 写入 %d 例到 %s（seed=%d）\n", len(cases), *out, *seed)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "fixture-gen: %v\n", err)
	os.Exit(3)
}

// writeAll 写出样本字节 + manifest.json。样本本体落 fixtures/out/（不入库）。
func writeAll(cases []gen.Case, outDir string, seed uint64) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
	m := manifest{Generator: "fixture-gen", Version: "0", Seed: seed}
	for _, c := range cases {
		path := filepath.Join(outDir, c.Filename)
		if err := os.WriteFile(path, c.Bytes, 0o644); err != nil {
			return err
		}
		m.Cases = append(m.Cases, manifestEntry{
			CaseID: c.ID, Suite: c.Suite, Filename: c.Filename,
			SHA256: c.SHA256, Size: len(c.Bytes), Seed: c.Seed,
			Expected: c.Expected, Note: c.Note,
		})
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "manifest.json"), append(raw, '\n'), 0o644)
}

// checkCommitted 与入库 manifest 比对：既比 sha256 清单，也比双跑字节。
func checkCommitted(cases []gen.Case, outDir string) error {
	committed, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("committed manifest missing at %s (run make fixtures first)", filepath.Join(outDir, "manifest.json"))
		}
		return err
	}
	var m manifest
	if err := json.Unmarshal(committed, &m); err != nil {
		return fmt.Errorf("committed manifest unparsable: %w", err)
	}
	if len(m.Cases) != len(cases) {
		return fmt.Errorf("case count drift: committed %d vs regenerated %d", len(m.Cases), len(cases))
	}
	byID := map[string]manifestEntry{}
	for _, e := range m.Cases {
		byID[e.CaseID] = e
	}
	for _, c := range cases {
		e, ok := byID[c.ID]
		if !ok {
			return fmt.Errorf("case %s missing from committed manifest", c.ID)
		}
		sum := sha256.Sum256(c.Bytes)
		if hex.EncodeToString(sum[:]) != e.SHA256 || c.SHA256 != e.SHA256 {
			return fmt.Errorf("case %s bytes drift (committed %s, regenerated %s)", c.ID, e.SHA256, c.SHA256)
		}
	}
	return nil
}
