// movie.go 提供"合法 MP4"的参数化构造器：所有 C1–C7 变体与对照组都从同一份
// 布局代码派生，避免每类各写一遍字节拼装。
//
// 纪律（plan/04 §二、data/README §4）：
//   - 不读时钟 / 环境变量 / 不起外部进程；无随机项，seed 仅记录；
//   - 关键字段的字节偏移在**构造点**确定（marks，plan/04 T5）：由已知的
//     盒长逐层累加得到，不用任何"回读生成物"的搜索；
//   - 合成样本内部约定：sgpd/sbgp 的 roll_distance = priming 样本数（正数，
//     AAC=1024）。真实文件里 ffmpeg 写 roll_distance=-1 + elst.media_time=1024
//     （2026-10-27 实测），符号语义留给 M4/M5 用真实样本复核（见夹具文档 §六）。
package gen

import (
	"encoding/binary"
	"fmt"
)

// marks 记录"构造点已知的字节偏移"（plan/04 T5 的真值）。
type marks map[string]uint64

func (m marks) set(k string, v uint64) { m[k] = v }

// absorb 把子结构的相对标记搬到绝对偏移。
func (m marks) absorb(prefix string, delta uint64, src marks) {
	for k, v := range src {
		m.set(prefix+k, delta+v)
	}
}

// ── 通用字段类型 ─────────────────────────────────────────────────────────

// sttsEntry 是 stts 的一条（count 个样本，每个 duration tick）。
type sttsEntry struct{ Count, Duration uint32 }

// cttsEntry 是 ctts 的一条（count 个样本，合成偏移 offset；version1 时为有符号）。
type cttsEntry struct {
	Count  uint32
	Offset int32
}

// elstEntry 是 elst 的一条；MediaTime = -1 表示空 edit。
// SegmentDuration 单位是 **movie timescale**；MediaTime 单位是媒体 timescale。
// MediaRate 是 16.16 定点；1.0 = 0x00010000。
type elstEntry struct {
	SegmentDuration uint32
	MediaTime       int32
	MediaRate       uint32
}

// track 描述一条轨道（默认：单 stts 条目、stco 单 chunk、均匀样本）。
type track struct {
	kind      string // "vide" | "soun"
	timescale uint32
	duration  uint32 // mdhd.duration（媒体时基）＝默认 tkhd/mvhd 换算来源
	tkhdDur   uint32 // 0 = 由 duration 按 mvhd 时基换算（整数向下取整）
	stts      []sttsEntry
	ctts      []cttsEntry // nil = 不写 ctts
	cttsV1    bool
	elst      []elstEntry // nil = 不写 edts
	matrix    *[9]uint32  // nil = identity
	w, h      uint32      // tkhd 宽高（16.16 定点，**呈现**方向声明）
	stsdW     uint32      // 0 = 与 w 相同；stsd 样本条目的存储宽（16.16）
	stsdH     uint32      // 0 = 与 h 相同
	codec     string      // "avc1" | "hvc1" | "mp4a" | "ac-3" | "Opus"
	sampleSz  uint32
	co64      bool    // true = 用 co64 而非 stco
	paspNum   uint32  // 0 = 不写 pasp
	paspDen   uint32
	colrs     [][]byte // 每项 = 一个 colr 盒的**载荷**（可多实例，按序）
	mdcv      []byte
	clli      []byte
	roll      *int16 // 非 nil = 写 sgpd+sbgp（roll 距离，正数 = priming 样本数）
	opusPre   uint16 // Opus 的 dOps.preSkip（默认 312）
	emptyStbl bool   // true = 样本表全空（stts/stsz/stco 零条目）
}

// frag 是一段分片：tfdt 基准解码时间 + 单 trun。
type frag struct {
	tfdt   uint64
	tfdtV1 bool
	count  uint32
	dur    uint32
	size   uint32
	noDur  bool // true = trun 不写 sample_duration（时长推不出，duration_unknown 用）
}

// movie 描述一个文件（mvhd + 若干轨道 + 可选分片）。
type movie struct {
	mvhdTimescale uint32
	mvhdDuration  uint32 // 直接进 mvhd；durationAuto 时被覆盖
	durationAuto  bool   // true = 取首轨时长按 mvhd 时基换算
	tracks        []*track
	frags         []frag // 非空 = 分片形态（moov 空表 + moof+mdat 序列）
	brand         string // ftyp major brand（默认 isom）
	stcoOverride  *uint64
	mdatDeclare   *uint32 // 非 nil = mdat 声明 size 覆盖实际
	tailCut       uint32  // >0 = 从文件尾砍掉这么多字节
	dualMdat      bool    // 追加第二个空 mdat 盒
	moovTrailer   []byte  // 追加到 moov 末尾的原始字节（child_overruns_parent 用）
	largesizeMdat bool    // true = mdat 用 size==1 + 64 位 largesize 形态
	sizeZeroMdat  bool    // true = mdat 用 size==0（到文件尾）形态
}

// ── 字节小工具（box/fullBox/u32/concat 在 gen.go）────────────────────────

func u16(v uint16) []byte { b := make([]byte, 2); binary.BigEndian.PutUint16(b, v); return b }
func i16(v int16) []byte  { return u16(uint16(v)) }
func i32(v int32) []byte  { return u32(uint32(v)) }
func u64(v uint64) []byte { b := make([]byte, 8); binary.BigEndian.PutUint64(b, v); return b }

// identityMatrix 是 3×3 单位矩阵（16.16 定点）。
func identityMatrix() [9]uint32 { return [9]uint32{0x00010000, 0, 0, 0, 0x00010000, 0, 0, 0, 0x40000000} }

// 常见显示变换矩阵（16.16 定点）。
func rot90Matrix() [9]uint32 {
	return [9]uint32{0, 0x00010000, 0, 0xFFFF0000, 0, 0, 0, 0, 0x40000000}
}
func rot270Matrix() [9]uint32 {
	return [9]uint32{0, 0xFFFF0000, 0, 0x00010000, 0, 0, 0, 0x40000000}
}
func rot180Matrix() [9]uint32 {
	return [9]uint32{0xFFFF0000, 0, 0, 0, 0xFFFF0000, 0, 0, 0, 0x40000000}
}

// shearMatrix 行点积 ≠ 1 → 非正交（matrix_not_orthogonal 用）。
func shearMatrix() [9]uint32 {
	return [9]uint32{0x00010000, 0x00008000, 0, 0, 0x00010000, 0, 0, 0, 0x40000000}
}

func matrixBytes(m [9]uint32) []byte {
	out := make([]byte, 0, 36)
	for _, v := range m {
		out = append(out, u32(v)...)
	}
	return out
}

// filler 是确定性样本载荷（0xAB，避免与 ASCII 盒名相撞）。
func filler(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = 0xAB
	}
	return b
}

// ── 样本条目（stsd entry）───────────────────────────────────────────────

// visualSampleEntry 拼 avc1/hvc1 的 78 字节固定头 + 子盒。
func visualSampleEntry(typ string, w, h uint16, children ...[]byte) []byte {
	head := concat(
		make([]byte, 6), u16(1), // reserved + data_reference_index
		u16(0), u16(0), make([]byte, 12),
		u16(w), u16(h),
		u32(0x00480000), u32(0x00480000),
		u32(0), u16(1),
		make([]byte, 32),
		u16(0x0018), u16(0xFFFF),
	)
	return box(typ, concat(head, concat(children...)))
}

// audioSampleEntryHead 是 AudioSampleEntry 的 28 字节固定头。
func audioSampleEntryHead(rate uint32) []byte {
	return concat(
		make([]byte, 6), u16(1),
		make([]byte, 8),
		u16(2), u16(16), u16(0), u16(0),
		u32(rate<<16),
	)
}

// avcCBox 是结构完整但内容为占位的最小 avcC。
func avcCBox() []byte {
	return box("avcC", []byte{
		0x01, 0x42, 0xC0, 0x1E,
		0xFF, 0xE1,
		0x00, 0x04, 0x67, 0x42, 0xC0, 0x1E,
		0x01, 0x00, 0x02, 0x68, 0xCE,
	})
}

// hvcCBox 是占位 hvcC。
func hvcCBox() []byte {
	return box("hvcC", []byte{
		0x01, 0x01, 0x60, 0x00, 0x00, 0x00, 0x90, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x96, 0xF0, 0x00, 0xFC, 0xFD, 0xF8, 0xF8, 0x00, 0x00, 0x0F,
	})
}

// esdsBox 是 mp4a 的最小符合规 esds + AAC-LC AudioSpecificConfig。
func esdsBox(freq uint32) []byte {
	asc := []byte{0x12, 0x10} // AAC-LC 44100
	if freq == 48000 {
		asc = []byte{0x11, 0x90}
	}
	payload := concat(
		[]byte{0x03, 0x19}, u16(2), []byte{0x00},
		[]byte{0x04, 0x11, 0x40, 0x15}, make([]byte, 3), u32(128000), u32(128000),
		[]byte{0x05, byte(len(asc))}, asc,
		[]byte{0x06, 0x01, 0x02},
	)
	return fullBox("esds", 0, 0, payload)
}

// dac3Box 是 ac-3 的最小 dac3（未知编码样本用）。
func dac3Box() []byte { return box("dac3", []byte{0x10, 0x3D, 0x28, 0x00}) }

// dOpsBox 是 Opus 的 dOps（preSkip 是 priming 的声明处）。
func dOpsBox(freq uint32, preSkip uint16) []byte {
	return box("dOps", concat(
		[]byte{0x00}, []byte{0x02}, u16(preSkip), u32(freq), u16(0), []byte{0x00},
	))
}

// stsdBox 用给定样本条目拼 stsd（entry_count=1）。
func stsdBox(entry []byte) []byte { return fullBox("stsd", 0, 0, concat(u32(1), entry)) }

// ── 各轨子盒 ────────────────────────────────────────────────────────────

func mvhdBox(ts, dur uint32) []byte {
	return fullBox("mvhd", 0, 0, concat(
		u32(0), u32(0), u32(ts), u32(dur),
		u32(0x00010000), u16(0x0100), u16(0), u32(0), u32(0),
		matrixBytes(identityMatrix()),
		make([]byte, 24), u32(2),
	))
}

func tkhdBox(trackID uint32, dur uint32, volume uint16, m [9]uint32, w, h uint32) []byte {
	return fullBox("tkhd", 0, 3, concat(
		u32(0), u32(0), u32(trackID), u32(0), u32(dur),
		u32(0), u32(0),
		u16(0), u16(0), u16(volume), u16(0),
		matrixBytes(m),
		u32(w), u32(h),
	))
}

func mdhdBox(ts, dur uint32) []byte {
	return fullBox("mdhd", 0, 0, concat(u32(0), u32(0), u32(ts), u32(dur), u16(0x55C4), u16(0)))
}

func hdlrBox(handler string, name []byte) []byte {
	return fullBox("hdlr", 0, 0, concat(u32(0), []byte(handler), make([]byte, 12), name, []byte{0}))
}

func elstBox(entries []elstEntry) []byte {
	payload := u32(uint32(len(entries)))
	for _, e := range entries {
		rate := e.MediaRate
		if rate == 0 {
			rate = 0x00010000
		}
		payload = append(payload, concat(u32(e.SegmentDuration), i32(e.MediaTime), u32(rate))...)
	}
	return fullBox("elst", 0, 0, payload)
}

func sgpdRollBox(roll int16) []byte {
	return fullBox("sgpd", 1, 0, concat([]byte("roll"), u32(2), u32(1), i16(roll)))
}

func sbgpRollBox(total uint32) []byte {
	return fullBox("sbgp", 0, 0, concat([]byte("roll"), u32(1), u32(total), u32(1)))
}

func sttsBox(entries []sttsEntry) []byte {
	b := u32(uint32(len(entries)))
	for _, e := range entries {
		b = append(b, concat(u32(e.Count), u32(e.Duration))...)
	}
	return fullBox("stts", 0, 0, b)
}

func cttsBox(entries []cttsEntry, v1 bool) []byte {
	version := byte(0)
	if v1 {
		version = 1
	}
	b := u32(uint32(len(entries)))
	for _, e := range entries {
		b = append(b, concat(u32(e.Count), i32(e.Offset))...)
	}
	return fullBox("ctts", version, 0, b)
}

func stscBox(chunks, samplesPerChunk uint32) []byte {
	if chunks == 0 || samplesPerChunk == 0 {
		return fullBox("stsc", 0, 0, u32(0))
	}
	return fullBox("stsc", 0, 0, concat(u32(1), u32(1), u32(samplesPerChunk), u32(1)))
}

func stszBox(sampleSize, count uint32) []byte {
	return fullBox("stsz", 0, 0, concat(u32(sampleSize), u32(count)))
}

func stcoBox(offsets []uint32) []byte {
	b := u32(uint32(len(offsets)))
	for _, o := range offsets {
		b = append(b, u32(o)...)
	}
	return fullBox("stco", 0, 0, b)
}

func co64Box(offsets []uint64) []byte {
	b := u32(uint32(len(offsets)))
	for _, o := range offsets {
		b = append(b, u64(o)...)
	}
	return fullBox("co64", 0, 0, b)
}

func paspBox(num, den uint32) []byte { return box("pasp", concat(u32(num), u32(den))) }

// colrPayload 是 colr 盒的载荷（nclx）。
func colrPayload(primaries, transfer, matrix uint16, fullRange bool) []byte {
	fr := byte(0)
	if fullRange {
		fr = 0x80
	}
	return concat([]byte("nclx"), u16(primaries), u16(transfer), u16(matrix), []byte{fr})
}

// mdcvBox / clliBox 亮度元数据（量程自检用）。
func mdcvBox(maxL, minL, maxC, minC uint32) []byte {
	return box("mdcv", concat(u16(0x3548), u16(0x1000), u16(0x0550), u16(0x0001), u32(maxL), u32(minL), u32(maxC), u32(minC)))
}

func clliBox(maxCLL, avgCLL uint16) []byte { return box("clli", concat(u16(maxCLL), u16(avgCLL))) }

// ── stbl / trak（返回相对标记）──────────────────────────────────────────

// stblBox 拼 stbl 并返回相对 stbl 起始的标记。fragMode 时样本表全空。
func stblBox(t *track, fragMode bool) ([]byte, marks) {
	mk := marks{}
	var sampleCount, sampleSize = uint32(0), t.sampleSz
	if !fragMode && !t.emptyStbl {
		for _, e := range t.stts {
			sampleCount += e.Count
		}
	} else {
		sampleSize = 0
	}

	// entryHead：样本条目的固定头长度（视觉 78 / 音频 28）；
	// kids：固定头之后的子盒（顺序即文件内顺序），用于生成"构造点已知"的子盒偏移。
	entryHead := uint32(78)
	var kids [][]byte
	switch t.codec {
	case "hvc1":
		kids = append(kids, hvcCBox())
		if t.paspNum != 0 {
			kids = append(kids, paspBox(t.paspNum, t.paspDen))
		}
		for _, c := range t.colrs {
			kids = append(kids, box("colr", c))
		}
	case "ac-3":
		entryHead = 28
		kids = append(kids, dac3Box())
	case "Opus":
		entryHead = 28
		pre := t.opusPre
		if pre == 0 {
			pre = 312
		}
		kids = append(kids, dOpsBox(t.timescale, pre))
	case "mp4a":
		entryHead = 28
		kids = append(kids, esdsBox(t.timescale))
	default: // avc1
		kids = append(kids, avcCBox())
		if t.paspNum != 0 {
			kids = append(kids, paspBox(t.paspNum, t.paspDen))
		}
		for _, c := range t.colrs {
			kids = append(kids, box("colr", c))
		}
		if t.mdcv != nil {
			kids = append(kids, t.mdcv)
		}
		if t.clli != nil {
			kids = append(kids, t.clli)
		}
	}

	var entry []byte
	if entryHead == 78 {
		ew, eh := t.w>>16, t.h>>16
		if t.stsdW != 0 {
			ew = t.stsdW >> 16
		}
		if t.stsdH != 0 {
			eh = t.stsdH >> 16
		}
		entry = visualSampleEntry(t.codec, uint16(ew), uint16(eh), kids...)
	} else {
		head := audioSampleEntryHead(t.timescale)
		entry = box(t.codec, concat(head, concat(kids...)))
	}

stsd := stsdBox(entry)
	// stsd 内偏移（相对 stbl）：头 8 + stsd 头 8 + v/f 4 + count 4 + entry 头 8 + 固定头
	childBase := uint64(8+8+4+4+8) + uint64(entryHead)
	childOff := childBase
	seen := map[string]int{}
	for _, k := range kids {
		typ := string(k[4:8])
		seen[typ]++
		name := typ
		if seen[typ] > 1 {
			name = fmt.Sprintf("%s%d", typ, seen[typ]-1)
		}
		mk.set("stsd_"+name, childOff)
		childOff += uint64(len(k))
	}
	if entryHead == 28 {
		mk.set("esds_off", childBase)
	}
	var stts []byte
	if t.emptyStbl {
		stts = sttsBox(nil)
	} else {
		stts = sttsBox(t.stts)
	}
	// stts 第 n 个条目的 duration 偏移（相对 stbl）：头 8 + stsd + stts 头 8 + v/f 4 + count 4 + n*8 + 4
	for n := 0; n < len(t.stts); n++ {
		mk.set(fmt.Sprintf("stts_entry%d_duration", n), 8+uint64(len(stsd))+8+4+4+uint64(n)*8+4)
	}

	parts := [][]byte{stsd, stts}
	if len(t.ctts) > 0 {
		parts = append(parts, cttsBox(t.ctts, t.cttsV1))
		mk.set("ctts_entry0", 8+uint64(len(stsd))+uint64(len(stts))+8+4+4)
		// 首条 ctts 的 sample_offset 字段（条目内 +4）：priming_undeclared 的证据落点
		mk.set("ctts_entry0_offset", 8+uint64(len(stsd))+uint64(len(stts))+8+4+4+4)
	}
	parts = append(parts, stscBox(1, sampleCount), stszBox(sampleSize, sampleCount))

	var offsetBox []byte
	switch {
	case sampleCount == 0 || t.emptyStbl:
		if t.co64 {
			offsetBox = co64Box(nil)
		} else {
			offsetBox = stcoBox(nil)
		}
	case t.co64:
		offsetBox = co64Box([]uint64{0}) // 占位；buildMovie 回填/覆盖
	default:
		offsetBox = stcoBox([]uint32{0}) // 占位；buildMovie 回填/覆盖
	}
	// stco/co64 条目（相对 stbl）：头 8 + 之前各段 + 头 8 + v/f 4 + count 4
	before := uint64(0)
	for _, p := range parts {
		before += uint64(len(p))
	}
	mk.set("stco0_entry0", 8+before+8+4+4)
	parts = append(parts, offsetBox)

	if t.roll != nil && !fragMode {
		beforeSgpd := uint64(0)
		for _, p := range parts {
			beforeSgpd += uint64(len(p))
		}
		beforeSgpd += uint64(len(offsetBox))
		sgpd := sgpdRollBox(*t.roll)
		mk.set("sgpd", 8+beforeSgpd)
		// sgpd 条目值：头 8 + v/f 4 + grouping_type 4 + default_length 4 + entry_count 4
		mk.set("sgpd_roll", 8+beforeSgpd+24)
		mk.set("sbgp", 8+beforeSgpd+uint64(len(sgpd)))
		parts = append(parts, sgpd, sbgpRollBox(sampleCount))
	}
	out := box("stbl", concat(parts...))
	return out, mk
}

// trakBox 拼一条轨道并返回相对 trak 起始的标记。
func trakBox(t *track, trackID uint32, mvhdTS uint32, fragMode bool) ([]byte, marks) {
	mk := marks{}
	m := identityMatrix()
	if t.matrix != nil {
		m = *t.matrix
	}
	tkhdDur := t.tkhdDur
	if tkhdDur == 0 {
		tkhdDur = uint32(uint64(t.duration) * uint64(mvhdTS) / uint64(t.timescale))
	}
	volume := uint16(0)
	handler, name := "vide", []byte("VideoHandler")
	if t.kind == "soun" {
		volume, handler, name = 0x0100, "soun", []byte("SoundHandler")
	}

	tkhd := tkhdBox(trackID, tkhdDur, volume, m, t.w, t.h)
	// tkhd 内字段（相对 trak）：trak 头 8 + tkhd 头 8 + v/f 4
	//   duration = +4+4+4+4+4（creation/modification/track_ID/reserved）→ 8+8+20
	mk.set("tkhd_duration", 8+8+20)
	mk.set("tkhd_matrix", 8+8+40)
	mk.set("tkhd_width", 8+8+76)

	var minfChildren [][]byte
	if t.kind == "vide" {
		minfChildren = append(minfChildren, fullBox("vmhd", 0, 1, concat(u16(0), u16(0), u16(0), u16(0))))
	} else {
		minfChildren = append(minfChildren, fullBox("smhd", 0, 0, concat(u16(0), u16(0))))
	}
	dref := fullBox("dref", 0, 0, concat(u32(1), fullBox("url ", 0, 1, nil)))
	minfChildren = append(minfChildren, box("dinf", dref))
	stbl, stblMK := stblBox(t, fragMode)
	minfChildren = append(minfChildren, stbl)

	mdhd := mdhdBox(t.timescale, t.duration)
	hdlr := hdlrBox(handler, name)
	minf := box("minf", concat(minfChildren...))
	mdia := box("mdia", concat(mdhd, hdlr, minf))

	trakChildren := [][]byte{tkhd}
	afterTkhd := uint64(8 + len(tkhd)) // mdia/edts 的起点（相对 trak）
	if len(t.elst) > 0 {
		edts := box("edts", elstBox(t.elst))
		trakChildren = append(trakChildren, edts)
		// edts 头 8 + elst 头 8 + v/f 4 + count 4 = 首条目起点
		elstFirst := afterTkhd + 8 + 8 + 8
		mk.set("elst0_segment_duration", elstFirst)
		mk.set("elst0_media_time", elstFirst+4)
		mk.set("elst0_media_rate", elstFirst+8)
		afterTkhd += uint64(len(edts))
	}
	trakChildren = append(trakChildren, mdia)
	out := box("trak", concat(trakChildren...))

	// mdhd.duration 相对 trak：afterTkhd + mdia 头 8 + mdhd 头 8 + v/f 4 + co 4 + mod 4 + ts 4
	mk.set("mdhd_duration", afterTkhd+8+8+4+12)
	mk.set("mdhd_timescale", afterTkhd+8+8+4+8)
	// stbl 起点相对 trak：afterTkhd + mdia 头 + mdhd + hdlr + minf 头 + vmhd/smhd + dinf
	stblBase := afterTkhd + 8 + uint64(len(mdhd)) + uint64(len(hdlr)) + 8 +
		uint64(len(minfChildren[0])) + uint64(len(minfChildren[1]))
	mk.absorb("stbl_", stblBase, stblMK)
	return out, mk
}

func mvexBox(trackCount uint32) []byte {
	kids := [][]byte{}
	for i := uint32(1); i <= trackCount; i++ {
		kids = append(kids, fullBox("trex", 0, 0, concat(u32(i), u32(1), u32(0), u32(0), u32(0))))
	}
	return box("mvex", concat(kids...))
}

// ── 分片 ────────────────────────────────────────────────────────────────

// moofBox 拼一个 moof（tfhd default-base-is-moof + tfdt + trun）。
// 返回 box 与相对 moof 起始的 tfdt **值**偏移。
func moofBox(seq uint32, f frag, trackID uint32, dataOffset int32) ([]byte, marks) {
	tfhd := fullBox("tfhd", 0, 0x020000, u32(trackID))
	var tfdt []byte
	if f.tfdtV1 {
		tfdt = fullBox("tfdt", 1, 0, u64(f.tfdt))
	} else {
		tfdt = fullBox("tfdt", 0, 0, u32(uint32(f.tfdt)))
	}
	flags := uint32(0x000001) // data-offset-present
	if !f.noDur {
		flags |= 0x000100
	}
	flags |= 0x000200 // sample-size-present
	perSample := make([]byte, 0, int(f.count)*8)
	for i := uint32(0); i < f.count; i++ {
		if !f.noDur {
			perSample = append(perSample, u32(f.dur)...)
		}
		perSample = append(perSample, u32(f.size)...)
	}
	trun := fullBox("trun", 0, flags, concat(u32(f.count), i32(dataOffset), perSample))
	mfhd := fullBox("mfhd", 0, 0, u32(seq))
	out := box("moof", concat(mfhd, box("traf", concat(tfhd, tfdt, trun))))

	mk := marks{}
	// moof 头 8 + mfhd 全长 + traf 头 8 + tfhd 全长 + tfdt 头 8 + v/f 4
	mk.set("tfdt", 8+uint64(len(mfhd))+8+uint64(len(tfhd))+8+4)
	return out, mk
}

// ── 顶层装配 ────────────────────────────────────────────────────────────

// buildMovie 装配完整文件，并返回构造点已知的绝对字节偏移（marks）。
//
// # 摆放纪律（对齐 FFmpeg 现实：moov 常跟在 mdat 之后，但本项目统一用 moov 前置）
//
// 顶层顺序：ftyp → moov → mdat（+ 可选第二 mdat / 分片序列）。
// 所有 stco/co64 条目在装配末尾按"已知的 moov 长度 + headLen"精确回填，
// 因而 **重构 moov 后必须重算 stco**——这里是一条装配函数，不做局部替换。
func buildMovie(m movie) ([]byte, marks) {
	brand := m.brand
	if brand == "" {
		brand = "isom"
	}
	ftyp := box("ftyp", concat([]byte(brand), u32(0x200), []byte("isomiso2avc1mp41")))

	moov, moovMK := assembleMoov(m)
	mk := marks{}
	mk.absorb("", uint64(len(ftyp)), moovMK)
	mk.set("moov0_start", uint64(len(ftyp)))

	out := concat(ftyp, moov)

	if m.frags == nil {
		payload := samplePayload(m)
		declared := uint32(len(payload) + 8)
		if m.mdatDeclare != nil {
			declared = *m.mdatDeclare
		}
		mdatStart := uint64(len(out))
		mk.set("mdat0_start", mdatStart)

		var mdatHead []byte
		headLen := uint64(8)
		switch {
		case m.largesizeMdat:
			headLen = 16
			mdatHead = concat(u32(1), []byte("mdat"), u64(uint64(len(payload))+16))
		case m.sizeZeroMdat:
			// size==0（到文件尾）：size 字段写 0，合法形态
			mdatHead = concat(u32(0), []byte("mdat"))
		default:
			mdatHead = concat(u32(declared), []byte("mdat"))
		}
		out = concat(out, mdatHead, payload)
		// 逐轨回填 stco/co64：轨道 k 的样本区 = 前轨样本累计之后
		cur := uint64(len(ftyp)) + uint64(len(moov)) + headLen
		for i, t := range m.tracks {
			key := fmt.Sprintf("trak%d_stbl_stco0_entry0", i)
			if off, ok := mk[key]; ok && !t.emptyStbl {
				val := cur
				if i == 0 && m.stcoOverride != nil {
					val = *m.stcoOverride
				}
				if t.co64 {
					binary.BigEndian.PutUint64(out[off:off+8], val)
				} else {
					binary.BigEndian.PutUint32(out[off:off+4], uint32(val))
				}
			}
			if !t.emptyStbl {
				var n uint32
				for _, e := range t.stts {
					n += e.Count
				}
				cur += uint64(n) * uint64(t.sampleSz)
			}
		}
		if m.dualMdat {
			mk.set("mdat1_start", uint64(len(out)))
			out = concat(out, concat(u32(8), []byte("mdat")))
		}
	} else {
		seq := uint32(1)
		for i, f := range m.frags {
			mo, _ := moofBox(seq, f, 1, 0)
			dataOff := int32(len(mo) + 8)
			mo, fragMK := moofBox(seq, f, 1, dataOff)
			moofOff := uint64(len(out))
			out = concat(out, mo, concat(u32(f.count*f.size+8), []byte("mdat"), filler(int(f.count*f.size))))
			mk.absorb(fmt.Sprintf("moof%d_", i), moofOff, fragMK)
			mk.set(fmt.Sprintf("moof%d_start", i), moofOff)
			seq++
		}
	}
	if m.tailCut > 0 && uint64(m.tailCut) <= uint64(len(out)) {
		out = out[:len(out)-int(m.tailCut)]
	}
	return out, mk
}

// samplePayload 按轨道顺序铺样本字节（每轨一段连续 chunk）。
func samplePayload(m movie) []byte {
	var out []byte
	for _, t := range m.tracks {
		if t.emptyStbl {
			continue
		}
		var n uint32
		for _, e := range t.stts {
			n += e.Count
		}
		out = append(out, filler(int(n*t.sampleSz))...)
	}
	return out
}

// assembleMoov 拼 moov 并返回相对 moov 起始的标记。
func assembleMoov(m movie) ([]byte, marks) {
	mk := marks{}
	mvhdDur := m.mvhdDuration
	if m.durationAuto && len(m.tracks) > 0 {
		t := m.tracks[0]
		mvhdDur = uint32(uint64(t.duration) * uint64(m.mvhdTimescale) / uint64(t.timescale))
	}
	mvhd := mvhdBox(m.mvhdTimescale, mvhdDur)
	// mvhd.duration：头 8 + v/f 4 + co 4 + mod 4 + timescale 4
	mk.set("mvhd_duration", 8+8+4+12)

	fragMode := m.frags != nil
	kids := [][]byte{mvhd}
	cur := uint64(8 + len(mvhd))
	for i, t := range m.tracks {
		trak, tmk := trakBox(t, uint32(i+1), m.mvhdTimescale, fragMode)
		mk.absorb(fmt.Sprintf("trak%d_", i), cur, tmk)
		kids = append(kids, trak)
		cur += uint64(len(trak))
	}
	if fragMode {
		kids = append(kids, mvexBox(uint32(len(m.tracks))))
	}
	if len(m.moovTrailer) > 0 {
		kids = append(kids, m.moovTrailer)
	}
	return box("moov", concat(kids...)), mk
}