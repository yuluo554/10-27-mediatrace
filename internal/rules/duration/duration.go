// Package duration 实现模块 M2（时长口径 / edit list 自洽）。
//
// 规则集合见 plan/03 §三；口径数字只在 data/README §5 与 internal/policy 定义。
//
// 纪律：
//   - 纯函数：只读已解析的盒子树与只读字节，不做 I/O、不读时钟、不读随机、不联网；
//   - 与生成器不共享代码（K8）：真值来自构造参数，本包只从字节推结论；
//   - **`elst` 按文件内顺序编号、不排序**（D10 / K6）：排序会掩盖"段次序错了"这类缺陷；
//   - **同一根因只报一条**（F15）：三口径互比只产出一条 `duration_mismatch`；
//     空 edit 与显示时长的矛盾由那条 `duration_mismatch` 表达，不连环报；
//   - 时长互比统一在**影片时基 `mvhd.timescale`** 上进行（整数、向下取整，无浮点），
//     容差用 `policy.TickTolerance`。
//
// 为什么公共时基取 `mvhd.timescale` 而不是 `mdhd.timescale`：`mvhd`/`tkhd`/`elst`
// 三个口径**原生**就写在影片时基里；媒体侧（`mdhd.duration`）只能向下取整换算过来，
// 量化误差严格小于 1 个影片 tick。反向（影片→媒体）换算会把粗时基的量化误差放大
// `mdhd.timescale/mvhd.timescale` 倍，把逐样本取整的合法文件（夹具 M3R23：
// tkhd=33@1000、mdhd=3000@90000）判成超容差——那是假阳性。故公共时基取粗的一侧。
//
// 判定用 `mdhd.duration`（声明值）而非 `Σstts`：夹具的 `stts` 是逐样本
// `floor(duration/frames)` 拼出来的，`Σstts` 与声明值天然差几个 tick（M3R03：990 vs 1000）；
// `Σstts` 仍作为证据原值报出（values 里的 `stts_sum`），但不参与判定的阈值比较。
package duration

import (
	"encoding/binary"
	"fmt"

	"github.com/yuluo554/mediatrace/internal/isobmff"
	"github.com/yuluo554/mediatrace/internal/policy"
	"github.com/yuluo554/mediatrace/internal/report"
)

// ModuleID 是模块号。
const ModuleID = "M2"

// RuleIDs 是本模块实现的规则 ID 全集（plan/03 §三 六条），供测试与文档对账。
var RuleIDs = []string{
	"duration_mismatch",
	"elst_item_rate_not_one",
	"elst_item_order_suspect",
	"empty_edit",
	"first_pts_outside_media",
	"duration_unknown",
}

// rateOne 是 16.16 定点的 1.0（elst.media_rate 的"等速"值）。
const rateOne uint32 = 0x00010000

// maxEntries 收敛上限（畸形 count 字段的防御，避免无界循环）。
const maxEntries = 1 << 20

// Audit 审计一个已解析文件，返回模块 M2 的全部 finding（已按确定性键排序）。
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

	mvhd    *isobmff.Box
	movieTs int64
	mvhdDur int64 // 影片时基 tick
	mvhdOff uint64
	ok      bool
}

func (a *auditor) add(f report.Finding) {
	f.Module = ModuleID
	a.findings = append(a.findings, f)
}

func (a *auditor) run() {
	a.readMovieHeader()
	if !a.ok {
		return
	}
	tracks := a.readTracks()

	if a.mvhdDur == 0 {
		// §5.5：mvhd.duration==0（分片常见）且推不出预期时长 → duration_unknown（gray）。
		// 推得出（mehd / trun / trex 给了样本时长）时不落 gray，也不猜。
		if !a.fragDurationDerivable() && !anyTrackHasMedia(tracks) {
			a.add(report.Finding{
				RuleID:     "duration_unknown",
				Status:     report.Gray,
				ByteOffset: a.mvhdOff,
				FieldPath:  a.mvhd.Path + ".duration",
				GrayReason: report.GrayDurationUnknown,
				Values:     []report.Value{{Name: "mvhd_duration", Value: 0, Unit: "tick"}},
				Message:    "mvhd.duration is 0 and no mehd / last-tfdt+trun sample durations make the expected duration derivable",
			})
		}
		return
	}

	// ── 影片级：mvhd.duration vs 放映时长（多轨取最大）────────────────────
	// 同一根因只报一条（F15）：已被影片级判过的轨道不再重复出 duration_mismatch。
	reported := map[*trackInfo]bool{}
	var presTrack *trackInfo
	presMovie := int64(0)
	for _, t := range tracks {
		if !t.hasMedia {
			continue
		}
		p := t.presentationMovie()
		if presTrack == nil || p > presMovie {
			presMovie, presTrack = p, t
		}
	}
	if presTrack != nil && !policy.DurationWithinTolerance(a.mvhdDur, presMovie) {
		off, path, culprit := a.mvhdOff, a.mvhd.Path+".duration", "mvhd_duration"
		if presTrack.hasEdts() {
			it := presTrack.elst[0]
			off, path, culprit = it.segDurOff, presTrack.elstFieldPath("segment_duration", 0), "elst_sum"
		}
		a.add(a.mismatch(presTrack, off, path, culprit,
			fmt.Sprintf("movie duration %d and presentation duration %d differ beyond tolerance", a.mvhdDur, presMovie)))
		reported[presTrack] = true
	}

	// ── 轨道级：tkhd.duration vs mdhd.duration；编辑表媒体跨度 vs mdhd.duration ──
	for _, t := range tracks {
		if !t.hasMedia || reported[t] {
			continue
		}
		switch {
		case !policy.DurationWithinTolerance(t.tkhdMovie, t.mediaMovie()):
			a.add(a.mismatch(t, t.tkhdOff, t.tkhd.Path+".duration", "tkhd_duration",
				fmt.Sprintf("tkhd.duration %d and mdhd.duration %d (movie ticks) differ beyond tolerance",
					t.tkhdMovie, t.mediaMovie())))
		case t.hasEdts() && t.ratesAllOne() && !policy.DurationWithinTolerance(t.nonEmptySpanMovie(), t.mediaMovie()):
			var off uint64
			idx := 0
			for _, it := range t.elst {
				if !it.empty() {
					off, idx = it.segDurOff, it.Index
					break
				}
			}
			a.add(a.mismatch(t, off, t.elstFieldPath("segment_duration", idx), "elst_sum",
				fmt.Sprintf("edit segments consume %d movie ticks of media but mdhd.duration is %d",
					t.nonEmptySpanMovie(), t.mediaMovie())))
		}
	}

	// ── 编辑表条目级规则（按文件内顺序）─────────────────────────────────
	for _, t := range tracks {
		a.checkEditList(t)
	}
}

// mismatch 组装一条 duration_mismatch（四元组齐：偏移 + 字段路径 + 值 + 差值）。
func (a *auditor) mismatch(t *trackInfo, off uint64, path, culprit, msg string) report.Finding {
	vals := []report.Value{
		{Name: "stts_sum", Value: t.sttsMovie(), Unit: "tick"},
		{Name: "mdhd_duration", Value: t.mediaMovie(), Unit: "tick"},
		{Name: "tkhd_duration", Value: t.tkhdMovie, Unit: "tick"},
		{Name: "mvhd_duration", Value: a.mvhdDur, Unit: "tick"},
	}
	if t.hasEdts() {
		vals = append(vals, report.Value{Name: "elst_sum", Value: t.presentationMovie(), Unit: "tick"})
	}
	lo, hi := vals[0].Value, vals[0].Value
	for _, v := range vals {
		if v.Value < lo {
			lo = v.Value
		}
		if v.Value > hi {
			hi = v.Value
		}
	}
	return report.Finding{
		RuleID:     "duration_mismatch",
		Status:     report.Fail,
		ByteOffset: off,
		FieldPath:  path,
		Values:     vals,
		Delta:      &report.Delta{Value: hi - lo, Unit: "tick"},
		Message:    fmt.Sprintf("%s (culprit: %s; all calibers in mvhd.timescale ticks)", msg, culprit),
	}
}

// ── 编辑表条目级规则（按文件内顺序，不排序）────────────────────────────

func (a *auditor) checkEditList(t *trackInfo) {
	if !t.hasEdts() {
		return
	}

	// elst_item_rate_not_one：同段不按等速换算，只报条目号。
	for _, it := range t.elst {
		if it.Rate == rateOne {
			continue
		}
		delta := int64(it.Rate) - int64(rateOne)
		if delta < 0 {
			delta = -delta
		}
		a.add(report.Finding{
			RuleID:     "elst_item_rate_not_one",
			Status:     report.Fail,
			ByteOffset: it.rateOff,
			FieldPath:  t.elstFieldPath("media_rate", it.Index),
			Values: []report.Value{
				{Name: "elst_item", Value: int64(it.Index), Unit: "index"},
				{Name: "media_rate_16_16", Value: int64(it.Rate), Unit: "1/65536"},
				{Name: "media_rate_expected", Value: int64(rateOne), Unit: "1/65536"},
			},
			Delta:   &report.Delta{Value: delta, Unit: "1/65536"},
			Message: fmt.Sprintf("elst item #%d declares media_rate %d (16.16) != 1.0; the segment must not be converted at constant rate", it.Index, it.Rate),
		})
	}

	// empty_edit：空 edit 只贡献放映延迟、不引用媒体数据。它必须位于**最前**且时长非 0；
	// 违反即自相矛盾。（与显示时长的矛盾由 duration_mismatch 表达，同一根因不连环报。）
	seenMedia := false
	start := int64(0)
	for _, it := range t.elst {
		if !it.empty() {
			seenMedia = true
			start += it.SegDur
			continue
		}
		if seenMedia || it.SegDur == 0 {
			why := "empty edit appears after a media segment"
			if it.SegDur == 0 {
				why = "empty edit declares zero segment_duration"
			}
			a.add(report.Finding{
				RuleID:     "empty_edit",
				Status:     report.Fail,
				ByteOffset: it.mediaTimeOff,
				FieldPath:  t.elstFieldPath("media_time", it.Index),
				Values: []report.Value{
					{Name: "elst_item", Value: int64(it.Index), Unit: "index"},
					{Name: "segment_duration", Value: it.SegDur, Unit: "tick"},
					{Name: "start_offset_movie_ticks", Value: start, Unit: "tick"},
				},
				Delta:   &report.Delta{Value: it.SegDur, Unit: "tick"},
				Message: fmt.Sprintf("empty edit (media_time == -1): %s", why),
			})
		}
		start += it.SegDur
	}

	if !t.hasPTS {
		return
	}
	firstNonEmpty := -1
	for i, it := range t.elst {
		if !it.empty() {
			firstNonEmpty = i
			break
		}
	}
	if firstNonEmpty < 0 {
		return
	}

	// elst_item_order_suspect：首帧 PTS 落在**后面**的一段而非首个非空段 → 段次序可疑。
	// 与"时长错"分开报（plan/01 §八 创新点）。
	for i, it := range t.elst {
		if it.empty() || i == firstNonEmpty {
			continue
		}
		lo, hi := t.mediaInterval(it)
		if t.firstPTS >= lo && t.firstPTS < hi {
			it0 := t.elst[firstNonEmpty]
			a.add(report.Finding{
				RuleID:     "elst_item_order_suspect",
				Status:     report.Fail,
				ByteOffset: it0.mediaTimeOff,
				FieldPath:  t.elstFieldPath("media_time", firstNonEmpty),
				Values: []report.Value{
					{Name: "first_pts", Value: t.firstPTS, Unit: "tick"},
					{Name: "containing_item", Value: int64(i), Unit: "index"},
					{Name: "expected_item", Value: int64(firstNonEmpty), Unit: "index"},
				},
				Delta:   &report.Delta{Value: int64(i - firstNonEmpty), Unit: "index"},
				Message: fmt.Sprintf("first frame PTS %d falls in elst item #%d, not in the first media item #%d — segment order suspect", t.firstPTS, i, firstNonEmpty),
			})
			break
		}
	}

	// first_pts_outside_media：首帧 PTS 不落在**任何**段的 media 区间。
	//
	// 只在编辑表本身"时长对不上"（非空段媒体跨度 ≠ mdhd.duration）时才判：跨度一致的
	// 编辑表就是一次合法裁剪（media_time>0 的起点裁剪属 M4 的 start_trim），此时首帧
	// PTS<media_time 是裁剪的正常后果，不得当缺陷。
	if !t.ratesAllOne() || policy.DurationWithinTolerance(t.nonEmptySpanMovie(), t.mediaMovie()) {
		return
	}
	inside := false
	for _, it := range t.elst {
		if it.empty() {
			continue
		}
		lo, hi := t.mediaInterval(it)
		if t.firstPTS >= lo && t.firstPTS < hi {
			inside = true
			break
		}
	}
	if inside {
		return
	}
	it := t.elst[firstNonEmpty]
	lo, hi := t.mediaInterval(it)
	var delta int64
	switch {
	case t.firstPTS < lo:
		delta = lo - t.firstPTS
	default:
		delta = t.firstPTS - hi + 1
	}
	a.add(report.Finding{
		RuleID:     "first_pts_outside_media",
		Status:     report.Fail,
		ByteOffset: it.mediaTimeOff,
		FieldPath:  t.elstFieldPath("media_time", firstNonEmpty),
		Values: []report.Value{
			{Name: "first_pts", Value: t.firstPTS, Unit: "tick"},
			{Name: "media_interval_start", Value: lo, Unit: "tick"},
			{Name: "media_interval_end", Value: hi, Unit: "tick"},
		},
		Delta:   &report.Delta{Value: delta, Unit: "tick"},
		Message: fmt.Sprintf("first frame PTS %d is outside the edit media interval [%d,%d) while the edit consumes more media than mdhd.duration declares", t.firstPTS, lo, hi),
	})
}

// ── 轨道信息 ────────────────────────────────────────────────────────────

type elstItem struct {
	Index        int
	SegDur       int64 // 影片时基 tick
	MediaTime    int64 // 媒体时基 tick；-1 = 空 edit
	Rate         uint32
	segDurOff    uint64
	mediaTimeOff uint64
	rateOff      uint64
}

func (e elstItem) empty() bool { return e.MediaTime == -1 }

type trackInfo struct {
	hasMedia bool
	mediaTs  int64 // mdhd.timescale
	mediaDur int64 // mdhd.duration（媒体时基）
	movieTs  int64 // mvhd.timescale（公共时基）
	elstPath string

	tkhd *isobmff.Box

	tkhdMovie int64  // tkhd.duration（影片时基 tick）
	tkhdOff   uint64 // tkhd.duration 字段偏移

	stts int64 // Σ(count×duration)（媒体时基）

	firstPTS int64
	hasPTS   bool

	elst []elstItem
}

func (t *trackInfo) hasEdts() bool { return len(t.elst) > 0 }

// mediaMovie 返回 mdhd.duration 换算到影片时基的 tick（向下取整）。
func (t *trackInfo) mediaMovie() int64 { return scale(t.mediaDur, t.movieTs, t.mediaTs) }

// sttsMovie 返回 Σstts 换算到影片时基的 tick（向下取整）；仅作证据原值报出。
func (t *trackInfo) sttsMovie() int64 { return scale(t.stts, t.movieTs, t.mediaTs) }

// presentationMovie 返回放映时长（影片时基 tick）：有编辑表取 Σ 段时长，否则取 mdhd.duration。
func (t *trackInfo) presentationMovie() int64 {
	if !t.hasEdts() {
		return t.mediaMovie()
	}
	var sum int64
	for _, it := range t.elst {
		sum += it.SegDur
	}
	return sum
}

// nonEmptySpanMovie 返回非空段的 Σ segment_duration（影片时基 tick）。
func (t *trackInfo) nonEmptySpanMovie() int64 {
	var sum int64
	for _, it := range t.elst {
		if !it.empty() {
			sum += it.SegDur
		}
	}
	return sum
}

func (t *trackInfo) ratesAllOne() bool {
	for _, it := range t.elst {
		if it.Rate != rateOne {
			return false
		}
	}
	return true
}

// mediaInterval 返回某非空段的媒体区间 [start,end)（媒体时基 tick）。
func (t *trackInfo) mediaInterval(it elstItem) (int64, int64) {
	span := scale(it.SegDur, t.mediaTs, t.movieTs)
	return it.MediaTime, it.MediaTime + span
}

// elstFieldPath 拼 `moov/trak[i]/edts/elst[j].<field>`；条目号按文件内顺序（不排序）。
func (t *trackInfo) elstFieldPath(field string, i int) string {
	base := t.elstPath
	if base == "" {
		base = "elst"
	}
	return fmt.Sprintf("%s[%d].%s", base, i, field)
}

func anyTrackHasMedia(ts []*trackInfo) bool {
	for _, t := range ts {
		if t.hasMedia {
			return true
		}
	}
	return false
}

// ── 盒子读数（全部带边界检查；version 0/1 两形态）───────────────────────

func (a *auditor) readMovieHeader() {
	mvhd := isobmff.Find(a.root, "mvhd")
	if mvhd == nil {
		return
	}
	ts, dur, off, ok := readMvhd(a.raw, mvhd)
	if !ok || ts == 0 {
		return
	}
	a.mvhd, a.movieTs, a.mvhdDur, a.mvhdOff, a.ok = mvhd, int64(ts), dur, off, true
}

func (a *auditor) readTracks() []*trackInfo {
	var out []*trackInfo
	for _, trak := range isobmff.FindAll(a.root, "trak") {
		t := &trackInfo{movieTs: a.movieTs}
		if tkhd := isobmff.Find(trak, "tkhd"); tkhd != nil {
			if dur, off, ok := readTkhd(a.raw, tkhd); ok {
				t.tkhd, t.tkhdMovie, t.tkhdOff = tkhd, dur, off
			}
		}
		if mdhd := isobmff.Find(trak, "mdhd"); mdhd != nil {
			if ts, dur, _, ok := readMdhd(a.raw, mdhd); ok {
				t.mediaTs, t.mediaDur = int64(ts), dur
			}
		}
		if stts := isobmff.Find(trak, "stts"); stts != nil {
			t.stts, _ = readStts(a.raw, stts)
		}
		// 首帧 PTS：无 ctts 时 = 首样本 DTS = 0；有 ctts 时 = 首样本合成偏移（v1 为有符号）。
		if ctts := isobmff.Find(trak, "ctts"); ctts != nil {
			if off, ok := readCttsFirst(a.raw, ctts); ok {
				t.firstPTS = off
			}
		}
		if elstBox := isobmff.Find(trak, "elst"); elstBox != nil {
			t.elstPath = elstBox.Path
			t.elst, _ = readElst(a.raw, elstBox)
		}
		// 只看有样本表的轨道：分片轨（空 stts）的时长由 moof 决定，本模块不判。
		t.hasMedia = t.mediaTs > 0 && t.stts > 0
		t.hasPTS = t.hasMedia
		out = append(out, t)
	}
	return out
}

func readMvhd(raw []byte, b *isobmff.Box) (ts uint32, dur int64, durOff uint64, ok bool) {
	p := b.PayloadStart()
	ver, ok := u8at(raw, p)
	if !ok {
		return 0, 0, 0, false
	}
	if ver == 1 {
		ts, ok = u32at(raw, p+20)
		d, ok2 := u64at(raw, p+24)
		if !ok || !ok2 {
			return 0, 0, 0, false
		}
		return ts, int64(d), p + 24, true
	}
	ts, ok = u32at(raw, p+12)
	d, ok2 := u32at(raw, p+16)
	if !ok || !ok2 {
		return 0, 0, 0, false
	}
	return ts, int64(d), p + 16, true
}

func readTkhd(raw []byte, b *isobmff.Box) (dur int64, durOff uint64, ok bool) {
	p := b.PayloadStart()
	ver, ok := u8at(raw, p)
	if !ok {
		return 0, 0, false
	}
	if ver == 1 {
		d, ok := u64at(raw, p+28)
		if !ok {
			return 0, 0, false
		}
		return int64(d), p + 28, true
	}
	d, ok := u32at(raw, p+20)
	if !ok {
		return 0, 0, false
	}
	return int64(d), p + 20, true
}

func readMdhd(raw []byte, b *isobmff.Box) (ts uint32, dur int64, durOff uint64, ok bool) {
	p := b.PayloadStart()
	ver, ok := u8at(raw, p)
	if !ok {
		return 0, 0, 0, false
	}
	if ver == 1 {
		ts, ok = u32at(raw, p+20)
		d, ok2 := u64at(raw, p+24)
		if !ok || !ok2 {
			return 0, 0, 0, false
		}
		return ts, int64(d), p + 24, true
	}
	ts, ok = u32at(raw, p+12)
	d, ok2 := u32at(raw, p+16)
	if !ok || !ok2 {
		return 0, 0, 0, false
	}
	return ts, int64(d), p + 16, true
}

func readStts(raw []byte, b *isobmff.Box) (sum int64, entries int) {
	p := b.PayloadStart()
	cnt, ok := u32at(raw, p+4)
	if !ok {
		return 0, 0
	}
	if cnt > maxEntries {
		cnt = maxEntries
	}
	base := p + 8
	for i := uint32(0); i < cnt; i++ {
		c, ok1 := u32at(raw, base+uint64(i)*8)
		d, ok2 := u32at(raw, base+uint64(i)*8+4)
		if !ok1 || !ok2 {
			break
		}
		sum += int64(c) * int64(d)
		entries++
	}
	return sum, entries
}

func readCttsFirst(raw []byte, b *isobmff.Box) (int64, bool) {
	p := b.PayloadStart()
	ver, ok := u8at(raw, p)
	if !ok {
		return 0, false
	}
	cnt, ok := u32at(raw, p+4)
	if !ok || cnt == 0 {
		return 0, false
	}
	if ver == 1 {
		v, ok := i32at(raw, p+12)
		return int64(v), ok
	}
	v, ok := u32at(raw, p+12)
	return int64(v), ok
}

func readElst(raw []byte, b *isobmff.Box) ([]elstItem, bool) {
	p := b.PayloadStart()
	ver, ok := u8at(raw, p)
	if !ok {
		return nil, false
	}
	cnt, ok := u32at(raw, p+4)
	if !ok {
		return nil, false
	}
	if cnt > maxEntries {
		cnt = maxEntries
	}
	width := uint64(12)
	if ver == 1 {
		width = 20
	}
	base := p + 8
	items := make([]elstItem, 0, cnt)
	for i := uint32(0); i < cnt; i++ {
		off := base + uint64(i)*width
		var seg, mt int64
		var rate uint32
		if ver == 1 {
			s, ok1 := u64at(raw, off)
			m, ok2 := i64at(raw, off+8)
			r, ok3 := u32at(raw, off+16)
			if !ok1 || !ok2 || !ok3 {
				break
			}
			seg, mt, rate = int64(s), m, r
		} else {
			s, ok1 := u32at(raw, off)
			m, ok2 := i32at(raw, off+4)
			r, ok3 := u32at(raw, off+8)
			if !ok1 || !ok2 || !ok3 {
				break
			}
			seg, mt, rate = int64(s), int64(m), r
		}
		items = append(items, elstItem{
			Index: int(i), SegDur: seg, MediaTime: mt, Rate: rate,
			segDurOff: off, mediaTimeOff: off + 4, rateOff: off + 8,
		})
	}
	return items, true
}

// fragDurationDerivable 报告分片文件的预期时长是否可从 mehd / trun / trex 推出。
func (a *auditor) fragDurationDerivable() bool {
	if mehd := isobmff.Find(a.root, "mehd"); mehd != nil {
		p := mehd.PayloadStart()
		ver, _ := u8at(a.raw, p)
		if ver == 1 {
			if v, ok := u64at(a.raw, p+4); ok && v > 0 {
				return true
			}
		} else if v, ok := u32at(a.raw, p+4); ok && v > 0 {
			return true
		}
	}
	defaults := map[uint32]uint32{}
	for _, trex := range isobmff.FindAll(a.root, "trex") {
		p := trex.PayloadStart()
		tid, ok1 := u32at(a.raw, p+4)
		d, ok2 := u32at(a.raw, p+12)
		if ok1 && ok2 {
			defaults[tid] = d
		}
	}
	for _, moof := range isobmff.FindAll(a.root, "moof") {
		for _, traf := range isobmff.FindAll(moof, "traf") {
			trackID, defaultDur := uint32(0), uint32(0)
			if tfhd := isobmff.Find(traf, "tfhd"); tfhd != nil {
				p := tfhd.PayloadStart()
				flags, _ := u32at(a.raw, p)
				f := flags & 0x00FFFFFF
				if tid, ok := u32at(a.raw, p+4); ok {
					trackID = tid
				}
				pos := p + 8
				if f&0x000001 != 0 {
					pos += 8 // base-data-offset
				}
				if f&0x000002 != 0 {
					pos += 4 // sample-description-index
				}
				if f&0x000008 != 0 {
					if v, ok := u32at(a.raw, pos); ok {
						defaultDur = v
					}
				}
			}
			if defaultDur == 0 {
				defaultDur = defaults[trackID]
			}
			if trun := isobmff.Find(traf, "trun"); trun != nil {
				if total, ok := readTrunDuration(a.raw, trun, defaultDur); ok && total > 0 {
					return true
				}
			}
		}
	}
	return false
}

// readTrunDuration 返回一个 trun 各样本时长之和（媒体时基 tick）。
func readTrunDuration(raw []byte, trun *isobmff.Box, fallback uint32) (int64, bool) {
	p := trun.PayloadStart()
	flags, ok := u32at(raw, p)
	if !ok {
		return 0, false
	}
	f := flags & 0x00FFFFFF
	count, ok := u32at(raw, p+4)
	if !ok {
		return 0, false
	}
	if count > maxEntries {
		count = maxEntries
	}
	pos := p + 8
	if f&0x0001 != 0 {
		pos += 4 // data-offset
	}
	if f&0x0004 != 0 {
		pos += 4 // first-sample-flags
	}
	if f&0x0100 == 0 {
		return int64(count) * int64(fallback), true
	}
	var stride uint64
	for _, bit := range []uint32{0x0100, 0x0200, 0x0400, 0x0800} {
		if f&bit != 0 {
			stride += 4
		}
	}
	if stride == 0 {
		return 0, false
	}
	var sum int64
	for i := uint32(0); i < count; i++ {
		v, ok := u32at(raw, pos+uint64(i)*stride)
		if !ok {
			break
		}
		sum += int64(v)
	}
	return sum, true
}

// ── 只读小工具（全部带边界检查）────────────────────────────────────────

func u8at(raw []byte, off uint64) (byte, bool) {
	if off+1 > uint64(len(raw)) {
		return 0, false
	}
	return raw[off], true
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

func u64at(raw []byte, off uint64) (uint64, bool) {
	if off+8 > uint64(len(raw)) {
		return 0, false
	}
	return binary.BigEndian.Uint64(raw[off : off+8]), true
}

func i64at(raw []byte, off uint64) (int64, bool) {
	v, ok := u64at(raw, off)
	return int64(v), ok
}

// scale 把 v 从 from 时基换算到 to 时基（整数、向下取整；不引入浮点）。
// K11（对齐方式）的唯一实现处。
func scale(v, to, from int64) int64 {
	if from <= 0 || to == from {
		return v
	}
	return v * to / from
}
