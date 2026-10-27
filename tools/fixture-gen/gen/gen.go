// Package gen 是 C1–C7 缺陷夹具与对照组的构造器集合。
//
// M1 交付的构造类（权威编号 `data/README.md §4`）：
//
//	C1 双 moov · C2 stco 越界 · C3 截断 mdat · C4a/C4b 时长口径互斥（不写负数字段）
//	C5 回退 DTS（stts 零时长 + 分片 tfdt 倒退） · C6 1024 samples 延迟 · C7 matrix=±90 无 pasp
//
// 对照组（证明"不误伤"，误报 0 是门禁）：
//
//	controls     每类至少 1 个合法样本
//	m3-regression ≥20 个正常 H.264/H.265（含 B 帧 + ctts 有符号偏移）
//	m4 / m5      正反例各 5
//
// 纪律：
//   - 只用 Config.Seed 做"随机选择"（本包当前无随机项，保留接口）；
//   - 不读时钟、不读环境变量、不起外部进程（gen_test.go 的 AST 扫描守）；
//   - 每个 Case 自带 Bytes 与 SHA256；真值在构造点确定（ExpectPass 与 marks）。
package gen

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
)

// Config 是生成参数。
type Config struct {
	Seed  uint64
	Suite string // all|defects|controls|m3-regression|m4|m5
}

// 期望状态（真值三态；与 report.Status 同字面）。
const (
	ExpectPass = "pass"
	ExpectFail = "fail"
	ExpectGray = "gray"
)

// Case 是一个生成的样本 + 真值标签。
//
// 真值纪律（plan/04 T1/T5）：ExpectStatus / Expected / GrayReason / Offset
// 全部来自**构造参数**，不由任何实现回算；Offset 是构造点已知的
// "首条 finding 应指向的字节偏移"（pass 类为 0）。
type Case struct {
	ID           string
	Suite        string
	Filename     string
	Bytes        []byte
	SHA256       string
	Seed         uint64
	ExpectStatus string   // pass | fail | gray
	Expected     []string // 期望命中的规则 ID（fail/gray 必带）
	GrayReason   string   // ExpectStatus==gray 时的期望 reason（§5.5 闭集）
	Offset       uint64   // 期望的首字节偏移（构造点已知；pass 为 0）
	Note         string
}

// Builder 是一个样本构造器。Build 返回字节 + 构造点已知的首证据偏移。
type Builder struct {
	ID           string
	Suite        string
	Filename     string
	ExpectStatus string
	Expected     []string
	GrayReason   string
	Note         string
	// BuildShape 是给"覆盖维度"断言用的形状标签（非空时由 gen_test 按名字计数）。
	BuildShape string
	Build      func(seed uint64) (raw []byte, offset uint64, err error)
}

// builders 是全部构造器。新增一类：在这里追加 + 更新 docs/fixture-pools.md。
var builders = append(append(append(append(
	defectBuilders, controlBuilders...),
	m3RegressionBuilders...),
	m4Builders...),
	m5Builders...)

// suites 是合法的 suite 名（未知 suite 返回空集，由 CLI 判为非法）。
var suites = map[string]bool{
	"all": true, "defects": true, "controls": true,
	"m3-regression": true, "m4": true, "m5": true,
}

// Build 按 suite 过滤并构造全部样本（按 ID 排序，确定性）。
func Build(cfg Config) ([]Case, error) {
	if !suites[cfg.Suite] {
		// 未知 suite：返回空集，由 CLI 判为非法（不得静默当成 pass）
		return nil, nil
	}
	selected := make([]Builder, 0, len(builders))
	for _, b := range builders {
		if cfg.Suite == "all" || cfg.Suite == b.Suite {
			selected = append(selected, b)
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })
	out := make([]Case, 0, len(selected))
	for _, b := range selected {
		raw, off, err := b.Build(cfg.Seed)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", b.ID, err)
		}
		sum := sha256.Sum256(raw)
		out = append(out, Case{
			ID: b.ID, Suite: b.Suite, Filename: b.Filename, Bytes: raw,
			SHA256: hex.EncodeToString(sum[:]), Seed: cfg.Seed,
			ExpectStatus: b.ExpectStatus, Expected: b.Expected,
			GrayReason: b.GrayReason, Offset: off, Note: b.Note,
		})
	}
	return out, nil
}

// ── 字节构造工具 ─────────────────────────────────────────────────────────

// box 拼一个普通盒：size + type + payload。
func box(typ string, payload []byte) []byte {
	buf := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(buf[0:4], uint32(8+len(payload)))
	copy(buf[4:8], typ)
	copy(buf[8:], payload)
	return buf
}

// fullBox 拼一个 FullBox：size + type + version/flags + payload。
func fullBox(typ string, version byte, flags uint32, payload []byte) []byte {
	head := make([]byte, 4)
	head[0] = version
	binary.BigEndian.PutUint32(head, uint32(version)<<24|(flags&0x00FFFFFF))
	return box(typ, append(head, payload...))
}

// u32 序列化一个大端 uint32。
func u32(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

// concat 拼接多段字节。
func concat(parts ...[]byte) []byte {
	total := 0
	for _, p := range parts {
		total += len(p)
	}
	out := make([]byte, 0, total)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}