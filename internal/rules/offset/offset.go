// Package offset 实现模块 M4（起始偏移与 priming / encoder delay）。
//
// 规则集合见 plan/03 §五；口径（期望 priming = 声明值）只在 data/README §2/§5 与
// internal/policy 定义。**本包不判"必须等于 1024"**：AAC 1024 / 2112、Opus 312 都是
// 观测值而非规范条款（K14），只判"与声明是否一致"。
//
// 纪律：
//   - 纯函数：只读已解析的盒子树与只读字节，不做 I/O、不读时钟、不读随机、不联网；
//   - 与生成器不共享代码（K8）：真值来自构造参数，本包只从字节推结论；
//   - priming / encoder delay 是**音频轨**语义：视频轨的 `ctts` 偏移是 B 帧重排，
//     不是起始偏移，本包对非音频轨**不做**任何判定（否则会误伤所有含 B 帧的视频）；
//   - **同一根因合并**（F16）：`priming_undeclared` 与 `start_trim` 互斥；一处根因只报一条；
//   - 判定顺序（可判性优先）：
//     1) 编码不在已知集（AAC/Opus）⇒ gray `unknown_codec_priming`（期望值不可判）；
//     2) 无裁剪 edit 且首帧后移：有 roll 声明 ⇒ gray `no_edts`（无法判断是裁剪还是缺陷）；
//        无 roll 声明 ⇒ fail `priming_undeclared`（后移量就是证据，报出该值）；
//     3) 有裁剪 edit（`elst[0].media_time > 0`）：
//        · AAC 无 roll 距离 ⇒ gray `no_roll_distance`（期望值不可推）；
//        · 期望值可判：裁剪 == 期望 ⇒ 合法（`K6`/`M4N1`/`M4N2`/`M4N3`）；
//          裁剪 > 期望 ⇒ fail `start_trim`（越过 priming 剪进正片）；
//          裁剪 < 期望 ⇒ fail `priming_mismatch`（声明与裁剪不一致）。
//
// 为什么"裁剪 > 期望"是 start_trim、而"裁剪 < 期望"是 priming_mismatch：
// 前者把该播的正片剪掉了（起始裁剪过量），后者是两处声明对不上（容器说多、编辑说少）。
// 证据落点统一在 `elst[0].media_time`（编辑声明的裁剪量），`values` 里同时报声明值与期望值。
package offset

import (
	"encoding/binary"
	"fmt"

	"github.com/yuluo554/10-27-mediatrace/internal/isobmff"
	"github.com/yuluo554/10-27-mediatrace/internal/report"
)

// ModuleID 是模块号。
const ModuleID = "M4"

// RuleIDs 是本模块实现的规则 ID 全集（plan/03 §五 六条），供测试与文档对账。
var RuleIDs = []string{
	"start_trim",
	"priming_undeclared",
	"priming_mismatch",
	"no_edts_but_nonzero_start",
	"no_roll_distance",
	"unknown_codec_priming",
}

// maxEntries 收敛上限（畸形 count 字段的防御）。
const maxEntries = 1 << 20

// Audit 审计一个已解析文件，返回模块 M4 的全部 finding（已按确定性键排序）。
// fileName 仅用于报告显示，本模块不产出绝对路径。
func Audit(root *isobmff.Box, raw []byte, fileName string) ([]report.Finding, error) {
	if root == nil {
		return nil, nil
	}
	a := &auditor{root: root, raw: raw, file: fileName}
	a.run()
	report.SortFindings(a.findings)
	return a.findings, nil
}

type auditor struct {
	root     *isobmff.Box
	raw      []byte
	file     string
	findings []report.Finding
}

func (a *auditor) add(f report.Finding) {
	f.Module = ModuleID
	a.findings = append(a.findings, f)
}

func (a *auditor) run() {
	for _, trak := range isobmff.FindAll(a.root, "trak") {
		t := a.readTrack(trak)
		if t.kind != "soun" {
			// priming / encoder delay 只对音频轨有语义（视频轨的 ctts 是 B 帧重排）。
			continue
		}
		a.checkPriming(t)
	}
}

// ── 轨道信息 ────────────────────────────────────────────────────────────

type trackInfo struct {
	kind      string // "soun" | "vide" | ""
	codec     string // stsd 样本条目 fourcc
	stsdPath  string
	codecOff  uint64 // 编码 fourcc 的偏移
	timescale int64  // mdhd.timescale（音频轨即采样率）

	hasCtts  bool
	cttsOff  uint64 // 首条 ctts 条目偏移
	cttsPath string
	start    int64 // 首帧 PTS 相对 DTS 的后移（媒体 tick；音频 tick 即采样单位）

	hasElst bool
	trim    int64 // elst[0].media_time（>0 才算裁剪；-1 空 edit / 0 都是无裁剪）
	trimOff uint64
	elstPath string

	hasRoll  bool
	roll     int64 // roll 距离（取绝对值：规范里负数表示"回滚"，幅度相同）
	rollOff  uint64
	sgpdPath string

	hasPreskip bool
	preskip    int64
	preskipOff uint64
}

// hasEdit 报告该轨是否有起点裁剪 edit。
func (t *trackInfo) hasEdit() bool { return t.trim > 0 }

// expectedPriming 返回容器声明的期望 priming（可判时 ok=true）。
// Opus 取 dOps.preSkip；AAC 取 sgpd/sbgp 的 roll 距离。
func (t *trackInfo) expectedPriming() (int64, string, bool) {
	switch t.codec {
	case "Opus":
		if t.hasPreskip {
			return t.preskip, "dOps.preSkip", true
		}
	case "mp4a":
		if t.hasRoll {
			return t.roll, "roll_distance", true
		}
	}
	return 0, "", false
}

func knownPrimingCodec(codec string) bool {
	return codec == "mp4a" || codec == "Opus"
}

func (a *auditor) readTrack(trak *isobmff.Box) *trackInfo {
	t := &trackInfo{kind: readHandlerKind(a.raw, trak)}
	if ts, _, _, ok := readMdhdTimescale(a.raw, trak); ok {
		t.timescale = int64(ts)
	}
	if stsd := isobmff.Find(trak, "stsd"); stsd != nil {
		t.stsdPath = stsd.Path
		if codec, entryStart, entryEnd, childStart, ok := sampleEntry(a.raw, stsd); ok {
			t.codec = codec
			t.codecOff = entryStart + 4
			if codec == "Opus" {
				if off, pre, ok := findChildU16(a.raw, "dOps", 2, childStart, entryEnd); ok {
					t.hasPreskip, t.preskip, t.preskipOff = true, int64(pre), off
				}
			}
		}
	}
	if ctts := isobmff.Find(trak, "ctts"); ctts != nil {
		t.cttsPath = ctts.Path
		if first, off, ok := readCttsFirst(a.raw, ctts); ok {
			t.hasCtts, t.start, t.cttsOff = true, first, off
		}
	}
	if elst := isobmff.Find(trak, "elst"); elst != nil {
		t.elstPath = elst.Path
		if mt, off, ok := readElstFirstMediaTime(a.raw, elst); ok {
			t.hasElst, t.trim, t.trimOff = true, mt, off
		}
	}
	if sgpd := isobmff.Find(trak, "sgpd"); sgpd != nil {
		t.sgpdPath = sgpd.Path
		if roll, off, ok := readRollDistance(a.raw, sgpd); ok {
			t.roll, t.rollOff, t.hasRoll = roll, off, true
		}
	}
	return t
}

// ── 规则 ────────────────────────────────────────────────────────────────

func (a *auditor) checkPriming(t *trackInfo) {
	// 1) 编码不在已知集：期望值不可判 → gray（不是 fail，也不猜）。
	if !knownPrimingCodec(t.codec) {
		a.add(report.Finding{
			RuleID:     "unknown_codec_priming",
			Status:     report.Gray,
			ByteOffset: t.codecOff,
			FieldPath:  t.stsdPath + ".sample_entry_type",
			GrayReason: report.GrayUnknownCodecPriming,
			Values: []report.Value{
				{Name: "codec_fourcc", Value: fourCCValue(t.codec), Unit: "fourcc"},
			},
			Message: fmt.Sprintf("codec %q is outside the known priming set (mp4a / Opus) — expected priming is undeterminable", t.codec),
		})
		return
	}

	// 2) 无裁剪 edit 且首帧后移。
	if !t.hasEdit() {
		if t.start <= 0 {
			return // 起点 0：无延迟可判（`M4N5`）
		}
		if _, _, known := t.expectedPriming(); known {
			// roll 声明了 priming，但没有 edit 说明是否该裁剪 → 判不准，落 gray。
			a.add(report.Finding{
				RuleID:     "no_edts_but_nonzero_start",
				Status:     report.Gray,
				ByteOffset: t.cttsOff,
				FieldPath:  t.cttsPath + "[0].sample_offset",
				GrayReason: report.GrayNoEdts,
				Values: []report.Value{
					{Name: "first_pts_shift", Value: t.start, Unit: "sample"},
					{Name: "roll_distance", Value: t.roll, Unit: "sample"},
				},
				Message: fmt.Sprintf("track shifts its first frame by %d samples but declares no edit list; cannot tell whether that is a trim or a defect", t.start),
			})
			return
		}
		// 无 roll 声明：后移量本身就是未声明 priming 的直接证据 → fail，报出该值。
		a.add(report.Finding{
			RuleID:     "priming_undeclared",
			Status:     report.Fail,
			ByteOffset: t.cttsOff,
			FieldPath:  t.cttsPath + "[0].sample_offset",
			Values: []report.Value{
				{Name: "first_pts_shift", Value: t.start, Unit: "sample"},
				{Name: "mdhd_timescale", Value: t.timescale, Unit: "hz"},
			},
			Delta:   &report.Delta{Value: t.start, Unit: "sample"},
			Message: fmt.Sprintf("first frame PTS is shifted by %d samples and no edit list trims it — encoder priming is undeclared", t.start),
		})
		return
	}

	// 3) 有裁剪 edit：与期望 priming 比对。
	expected, src, known := t.expectedPriming()
	if !known {
		// AAC 无 roll 距离：期望值不可推 → gray（`C6b`）。
		a.add(report.Finding{
			RuleID:     "no_roll_distance",
			Status:     report.Gray,
			ByteOffset: t.trimOff,
			FieldPath:  t.elstPath + "[0].media_time",
			GrayReason: report.GrayNoRollDistance,
			Values: []report.Value{
				{Name: "elst_media_time", Value: t.trim, Unit: "tick"},
				{Name: "mdhd_timescale", Value: t.timescale, Unit: "hz"},
			},
			Message: "audio track trims the start but declares no sgpd/sbgp roll distance — expected priming cannot be derived",
		})
		return
	}

	switch {
	case t.trim == expected:
		return // 正确裁剪（`K6`/`M4N1`/`M4N2`/`M4N3`）
	case t.trim > expected:
		a.add(report.Finding{
			RuleID:     "start_trim",
			Status:     report.Fail,
			ByteOffset: t.trimOff,
			FieldPath:  t.elstPath + "[0].media_time",
			Values: []report.Value{
				{Name: "elst_media_time", Value: t.trim, Unit: "tick"},
				{Name: "start_trim_samples", Value: t.trim, Unit: "sample"},
				{Name: "start_trim_millis", Value: millis(t.trim, t.timescale), Unit: "millis"},
				{Name: "expected_priming", Value: expected, Unit: "sample"},
				{Name: "expected_from_preskip", Value: flagValue(src == "dOps.preSkip"), Unit: "flag"},
			},
			Delta:   &report.Delta{Value: t.trim - expected, Unit: "tick"},
			Message: fmt.Sprintf("edit trims %d samples (%d ms) from the start, exceeding the declared priming %d (%s) — it cuts into content",
				t.trim, millis(t.trim, t.timescale), expected, src),
		})
	default: // t.trim < expected
		a.add(report.Finding{
			RuleID:     "priming_mismatch",
			Status:     report.Fail,
			ByteOffset: t.trimOff,
			FieldPath:  t.elstPath + "[0].media_time",
			Values: []report.Value{
				{Name: "elst_media_time", Value: t.trim, Unit: "tick"},
				{Name: "expected_priming", Value: expected, Unit: "sample"},
				{Name: "expected_from_preskip", Value: flagValue(src == "dOps.preSkip"), Unit: "flag"},
			},
			Delta:   &report.Delta{Value: expected - t.trim, Unit: "tick"},
			Message: fmt.Sprintf("declared priming %d (%s) and the edit trim %d samples disagree", expected, src, t.trim),
		})
	}
}

func millis(ticks, timescale int64) int64 {
	if timescale <= 0 {
		return 0
	}
	return ticks * 1000 / timescale
}

// ── 盒子读数（全部带边界检查）───────────────────────────────────────────

// readHandlerKind 读 hdlr.handler_type 的 fourcc；取前 4 字节里的 handler 字符串。
func readHandlerKind(raw []byte, trak *isobmff.Box) string {
	hdlr := isobmff.Find(trak, "hdlr")
	if hdlr == nil {
		return ""
	}
	off := hdlr.PayloadStart() + 8 // vf4 + pre_defined4
	if off+4 > uint64(len(raw)) {
		return ""
	}
	return string(raw[off : off+4])
}

// sampleEntry 返回 stsd 首个样本条目的编码 fourcc 与其子盒区间。
// stsd 不是容器盒（IR 里是叶子），故这里按字节自读。
func sampleEntry(raw []byte, stsd *isobmff.Box) (codec string, entryStart, entryEnd, childStart uint64, ok bool) {
	p := stsd.PayloadStart()
	count, ok := u32at(raw, p+4)
	if !ok || count == 0 {
		return "", 0, 0, 0, false
	}
	entryStart = p + 8
	size, ok1 := u32at(raw, entryStart)
	codec, ok2 := fourCC(raw, entryStart+4)
	if !ok1 || !ok2 || size < 8 {
		return "", 0, 0, 0, false
	}
	entryEnd = entryStart + uint64(size)
	if entryEnd > uint64(len(raw)) {
		entryEnd = uint64(len(raw))
	}
	childStart = entryStart + 8 + entryHeadLen(codec)
	if childStart > entryEnd {
		childStart = entryEnd
	}
	return codec, entryStart, entryEnd, childStart, true
}

// entryHeadLen 返回样本条目固定头长度（视觉 78 / 音频 28），未识别的按 0（无子盒）。
func entryHeadLen(codec string) uint64 {
	switch codec {
	case "avc1", "avc3", "hvc1", "hev1", "av01", "vp09", "vp08", "mp4v", "dvh1", "dvhe", "dav1":
		return 78
	case "mp4a", "Opus", "ac-3", "ec-3", "ac-4", "alac", "fLaC", "lpcm", "sowt", "twos", "ulaw", "alaw":
		return 28
	}
	return 0
}

// findChildU16 在 [start,end) 的子盒序列里找 typ，返回其载荷内 fieldOffset 处的 u16 值与
// 该值的绝对偏移。
func findChildU16(raw []byte, typ string, fieldOffset uint64, start, end uint64) (valOff uint64, v uint16, ok bool) {
	for off := start; off+8 <= end; {
		sz, ok1 := u32at(raw, off)
		t, ok2 := fourCC(raw, off+4)
		if !ok1 || !ok2 || sz < 8 {
			return 0, 0, false
		}
		if t == typ {
			p := off + 8
			if x, ok := u16at(raw, p+fieldOffset); ok {
				return p + fieldOffset, x, true
			}
			return 0, 0, false
		}
		off += uint64(sz)
	}
	return 0, 0, false
}

func readMdhdTimescale(raw []byte, trak *isobmff.Box) (ts uint32, fieldOff uint64, path string, ok bool) {
	mdhd := isobmff.Find(trak, "mdhd")
	if mdhd == nil {
		return 0, 0, "", false
	}
	p := mdhd.PayloadStart()
	ver, ok := u8at(raw, p)
	if !ok {
		return 0, 0, "", false
	}
	if ver == 1 {
		v, ok := u32at(raw, p+20)
		return v, p + 20, mdhd.Path, ok
	}
	v, ok := u32at(raw, p+12)
	return v, p + 12, mdhd.Path, ok
}

// readCttsFirst 返回首条 ctts 条目的合成偏移（v1 为有符号）与偏移字段的绝对地址。
func readCttsFirst(raw []byte, ctts *isobmff.Box) (offset int64, fieldOff uint64, ok bool) {
	p := ctts.PayloadStart()
	ver, ok := u8at(raw, p)
	if !ok {
		return 0, 0, false
	}
	cnt, ok := u32at(raw, p+4)
	if !ok || cnt == 0 {
		return 0, 0, false
	}
	if ver == 1 {
		v, ok := i32at(raw, p+12)
		return int64(v), p + 12, ok
	}
	v, ok := u32at(raw, p+12)
	return int64(v), p + 12, ok
}

// readElstFirstMediaTime 返回 elst 首个条目的 media_time 与其字段偏移（按文件内顺序，不排序）。
func readElstFirstMediaTime(raw []byte, elst *isobmff.Box) (mediaTime int64, fieldOff uint64, ok bool) {
	p := elst.PayloadStart()
	ver, ok := u8at(raw, p)
	if !ok {
		return 0, 0, false
	}
	cnt, ok := u32at(raw, p+4)
	if !ok || cnt == 0 {
		return 0, 0, false
	}
	if ver == 1 {
		v, ok := i64at(raw, p+16)
		return v, p + 16, ok
	}
	v, ok := i32at(raw, p+12)
	return int64(v), p + 12, ok
}

// readRollDistance 读 sgpd 的 roll 距离（grouping_type == "roll"），取幅度（绝对值）。
func readRollDistance(raw []byte, sgpd *isobmff.Box) (roll int64, fieldOff uint64, ok bool) {
	p := sgpd.PayloadStart()
	gt, ok := fourCC(raw, p+4)
	if !ok || gt != "roll" {
		return 0, 0, false
	}
	cnt, ok := u32at(raw, p+12) // default_length 之后是 entry_count
	if !ok || cnt == 0 {
		return 0, 0, false
	}
	v, ok := i16at(raw, p+16)
	if !ok {
		return 0, 0, false
	}
	r := int64(v)
	if r < 0 {
		r = -r
	}
	return r, p + 16, true
}

// ── 只读小工具（全部带边界检查）────────────────────────────────────────

func fourCC(raw []byte, off uint64) (string, bool) {
	if off+4 > uint64(len(raw)) {
		return "", false
	}
	return string(raw[off : off+4]), true
}

func fourCCValue(s string) int64 {
	if len(s) < 4 {
		return 0
	}
	return int64(binary.BigEndian.Uint32([]byte(s[:4])))
}

// flagValue 把布尔转成 0/1（report.Value 只收整数）。
func flagValue(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func u8at(raw []byte, off uint64) (byte, bool) {
	if off+1 > uint64(len(raw)) {
		return 0, false
	}
	return raw[off], true
}

func u16at(raw []byte, off uint64) (uint16, bool) {
	if off+2 > uint64(len(raw)) {
		return 0, false
	}
	return binary.BigEndian.Uint16(raw[off : off+2]), true
}

func i16at(raw []byte, off uint64) (int16, bool) {
	v, ok := u16at(raw, off)
	return int16(v), ok
}

func u32at(raw []byte, off uint64) (uint32, bool) {
	if off+4 > uint64(len(raw)) {
		return 0, false
	}
	return binary.BigEndian.Uint32(raw[off : off+4]), true
}

func i32at(raw []byte, off uint64) (int32, bool) {
	v, ok := u32at(raw, off)
	return int32(v), ok
}

func i64at(raw []byte, off uint64) (int64, bool) {
	if off+8 > uint64(len(raw)) {
		return 0, false
	}
	return int64(binary.BigEndian.Uint64(raw[off : off+8])), true
}
