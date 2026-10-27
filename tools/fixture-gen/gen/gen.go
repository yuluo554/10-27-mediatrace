// Package gen 是 C1–C7 缺陷夹具的构造器集合。
//
// 骨架期已实现：C1（双 moov）、C2（stco 越界）、C3（截断 mdat）。
// 待实现（对应里程碑 M1 出口门）：C4a/C4b（时长口径互斥）、C5（回退 DTS）、
// C6（1024 samples 延迟）、C7（matrix=90 无 pasp），以及对照组与 m3-regression 套件。
//
// 纪律：
//   - 只用 Config.Seed 做"随机选择"（本包当前无随机项，保留接口）；
//   - 不读时钟、不读环境变量、不起外部进程；
//   - 每个 Case 自带 Bytes 与 SHA256，真值（Expected）在构造点确定。
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

// Case 是一个生成的样本 + 真值标签。
type Case struct {
	ID       string
	Suite    string
	Filename string
	Bytes    []byte
	SHA256   string
	Seed     uint64
	Expected []string // 期望命中的规则 ID；空 = 期望 pass
	Note     string
}

// Builder 是一个样本构造器。
type Builder struct {
	ID       string
	Suite    string
	Filename string
	Expected []string
	Note     string
	Build    func(seed uint64) ([]byte, error)
}

// builders 是已实现的构造器清单。新增一类：在这里追加 + 更新 docs/fixture-pools.md。
var builders = []Builder{
	{
		ID: "C1", Suite: "defects", Filename: "c1_dual_moov.mp4",
		Expected: []string{"duplicate_moov"},
		Note:     "复制整个 moov 盒插在第二处；期望 M1 duplicate_moov，报两处偏移",
		Build:    buildC1DualMoov,
	},
	{
		ID: "C2", Suite: "defects", Filename: "c2_stco_out_of_mdat.mp4",
		Expected: []string{"stco_out_of_mdat"},
		Note:     "单个 chunk offset 指向 mdat 载荷之外",
		Build:    buildC2StcoOutOfMdat,
	},
	{
		ID: "C3", Suite: "defects", Filename: "c3_truncated_mdat.mp4",
		Expected: []string{"truncated_mdat"},
		Note:     "声明 mdat size 大于实际剩余字节",
		Build:    buildC3TruncatedMdat,
	},
}

// Build 按 suite 过滤并构造全部样本（按 ID 排序，确定性）。
func Build(cfg Config) ([]Case, error) {
	selected := make([]Builder, 0, len(builders))
	for _, b := range builders {
		if cfg.Suite == "all" || cfg.Suite == b.Suite {
			selected = append(selected, b)
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })
	out := make([]Case, 0, len(selected))
	for _, b := range selected {
		raw, err := b.Build(cfg.Seed)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", b.ID, err)
		}
		sum := sha256.Sum256(raw)
		out = append(out, Case{
			ID: b.ID, Suite: b.Suite, Filename: b.Filename, Bytes: raw,
			SHA256: hex.EncodeToString(sum[:]), Seed: cfg.Seed,
			Expected: b.Expected, Note: b.Note,
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

// ── C1：双 moov ─────────────────────────────────────────────────────────
func buildC1DualMoov(seed uint64) ([]byte, error) {
	ftyp := box("ftyp", concat([]byte("isom"), u32(0x200), []byte("isomiso2mp41")))
	mvhd := fullBox("mvhd", 0, 0, append(concat(u32(0), u32(0), u32(1000), u32(1000)),
		concat(make([]byte, 80))...))
	moov := box("moov", mvhd)
	mdat := box("mdat", make([]byte, 32))
	return concat(ftyp, moov, moov, mdat), nil
}

// ── C2：stco 越界 ────────────────────────────────────────────────────────
func buildC2StcoOutOfMdat(seed uint64) ([]byte, error) {
	ftyp := box("ftyp", concat([]byte("isom"), u32(0x200), []byte("isomiso2mp41")))
	// stsd: 空表（entry_count=0 也可解析）
	stsd := fullBox("stsd", 0, 0, u32(0))
	// stts: 1 entry, count=1, duration=1000
	stts := fullBox("stts", 0, 0, concat(u32(1), u32(1), u32(1000)))
	// stsz: sample_size=0, sample_count=1, entry=4
	stsz := fullBox("stsz", 0, 0, concat(u32(0), u32(1), u32(4)))
	// stsc: 1 entry
	stsc := fullBox("stsc", 0, 0, concat(u32(1), u32(1), u32(1), u32(1)))
	// stco: 1 entry pointing far beyond mdat payload (0x7FFFFFF0)
	stco := fullBox("stco", 0, 0, concat(u32(1), u32(0x7FFFFFF0)))
	stbl := box("stbl", concat(stsd, stts, stsz, stsc, stco))
	minf := box("minf", stbl)
	mdia := box("mdia", minf)
	trak := box("trak", mdia)
	mvhd := fullBox("mvhd", 0, 0, append(concat(u32(0), u32(0), u32(1000), u32(1000)),
		make([]byte, 80)...))
	moov := box("moov", concat(mvhd, trak))
	mdat := box("mdat", make([]byte, 64)) // payload 区间远小于 0x7FFFFFF0
	return concat(ftyp, moov, mdat), nil
}

// ── C3：截断 mdat ────────────────────────────────────────────────────────
func buildC3TruncatedMdat(seed uint64) ([]byte, error) {
	ftyp := box("ftyp", concat([]byte("isom"), u32(0x200), []byte("isomiso2mp41")))
	mvhd := fullBox("mvhd", 0, 0, append(concat(u32(0), u32(0), u32(1000), u32(1000)),
		make([]byte, 80)...))
	moov := box("moov", mvhd)
	// mdat 头声明 size=1024，但实际只给 16 字节载荷
	head := make([]byte, 8)
	binary.BigEndian.PutUint32(head[0:4], 1024)
	copy(head[4:8], "mdat")
	mdat := append(head, make([]byte, 16)...)
	return concat(ftyp, moov, mdat), nil
}
