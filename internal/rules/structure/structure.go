// Package structure 实现模块 M1（结构自洽）。
//
// 规则集合见 plan/03 §二；口径（四元组、三态）只在 data/README §5 与 internal/report 定义。
//
// 纪律：
//   - 纯函数：只读已解析的盒子树与只读字节，不做 I/O、不读时钟、不读随机、不联网；
//   - 与生成器不共享代码（K8）：真值来自构造参数，本包只从字节推结论；
//   - 所有对 raw 的读取都带边界检查（截断文件不得 panic）；
//   - **同一根因合并**（plan/03 §1.3）：越界子盒的子树不再逐条报；重复 moov 时
//     stco/co64 偏移已不可信，跳过区间校验（只报 duplicate_moov）。
package structure

import (
	"encoding/binary"
	"fmt"

	"github.com/yuluo554/mediatrace/internal/isobmff"
	"github.com/yuluo554/mediatrace/internal/report"
)

// ModuleID 是模块号。
const ModuleID = "M1"

// RuleIDs 是本模块实现的规则 ID 全集（plan/03 §二 八条），供测试与文档对账。
var RuleIDs = []string{
	"too_small_size",
	"child_overruns_parent",
	"duplicate_moov",
	"duplicate_mdat",
	"stco_out_of_mdat",
	"truncated_mdat",
	"ftyp_brand_mismatch",
	"max_depth_exceeded",
}

// Audit 审计一个已解析文件，返回模块 M1 的全部 finding（已按确定性键排序）。
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
	a.checkAnomalies()
	a.checkOverruns()
	a.checkDuplicateMoov()
	a.checkDuplicateMdat()
	a.checkChunks()
	a.checkFtypBrand()
}

// ── 遍历期异常 → finding ────────────────────────────────────────────────

func (a *auditor) checkAnomalies() {
	for _, an := range a.root.Anomalies {
		switch an.Kind {
		case isobmff.AnomTooSmallSize:
			a.add(report.Finding{
				RuleID:     "too_small_size",
				Status:     report.Fail,
				ByteOffset: an.Offset,
				FieldPath:  childPath(an.Parent, an.BoxType),
				Values: []report.Value{
					{Name: "declared_size", Value: int64(an.Declared), Unit: "byte"},
					{Name: "minimum_header", Value: 8, Unit: "byte"},
				},
				Delta:   &report.Delta{Value: 8 - int64(an.Declared), Unit: "byte"},
				Message: fmt.Sprintf("box %q declares size %d (< 8-byte minimum header)", an.BoxType, an.Declared),
			})
		case isobmff.AnomMaxDepth:
			a.add(report.Finding{
				RuleID:     "max_depth_exceeded",
				Status:     report.Fail,
				ByteOffset: an.Offset,
				FieldPath:  an.Parent.Path,
				Values: []report.Value{
					{Name: "max_depth", Value: int64(isobmff.MaxDepth), Unit: "level"},
					{Name: "observed_depth", Value: int64(an.Depth), Unit: "level"},
				},
				Delta:   &report.Delta{Value: int64(an.Depth) - int64(isobmff.MaxDepth) + 1, Unit: "level"},
				Message: fmt.Sprintf("box nesting reached depth %d (cap %d); deeper containers left unexpanded", an.Depth, isobmff.MaxDepth),
			})
		}
	}
}

// ── 子盒越界 / 截断 ─────────────────────────────────────────────────────

func (a *auditor) checkOverruns() {
	fileLen := uint64(len(a.raw))
	isobmff.Walk(a.root, func(b *isobmff.Box) bool {
		if b.PayloadEnd() <= b.ParentEnd {
			return true
		}
		// 顶层 mdat 声明长度越过文件尾 → truncated_mdat（C3 形态）
		if b.Type == "mdat" && b.Depth == 1 {
			available := int64(fileLen) - int64(b.Offset)
			a.add(report.Finding{
				RuleID:     "truncated_mdat",
				Status:     report.Fail,
				ByteOffset: b.Offset,
				FieldPath:  b.Path,
				Values: []report.Value{
					{Name: "declared_size", Value: int64(b.Size), Unit: "byte"},
					{Name: "available_bytes", Value: available, Unit: "byte"},
				},
				Delta:   &report.Delta{Value: int64(b.Size) - available, Unit: "byte"},
				Message: fmt.Sprintf("mdat declares %d bytes but only %d are readable from offset 0x%X", b.Size, available, b.Offset),
			})
			return true
		}
		a.add(report.Finding{
			RuleID:     "child_overruns_parent",
			Status:     report.Fail,
			ByteOffset: b.Offset,
			FieldPath:  b.Path,
			Values: []report.Value{
				{Name: "child_end", Value: int64(b.PayloadEnd()), Unit: "byte"},
				{Name: "parent_end", Value: int64(b.ParentEnd), Unit: "byte"},
			},
			Delta: &report.Delta{Value: int64(b.PayloadEnd() - b.ParentEnd), Unit: "byte"},
			Message: fmt.Sprintf("box %q (offset 0x%X) ends at 0x%X beyond its parent payload end 0x%X",
				b.Type, b.Offset, b.PayloadEnd(), b.ParentEnd),
		})
		return true
	})
}

// ── moov / mdat 唯一性 ──────────────────────────────────────────────────

func (a *auditor) checkDuplicateMoov() {
	moovs := isobmff.FindAll(a.root, "moov")
	if len(moovs) < 2 {
		return
	}
	second := moovs[1]
	vals := make([]report.Value, 0, len(moovs))
	offs := make([]string, 0, len(moovs))
	for i, b := range moovs {
		vals = append(vals, report.Value{Name: fmt.Sprintf("moov[%d]", i), Value: int64(b.Offset), Unit: "byte"})
		offs = append(offs, fmt.Sprintf("0x%X", b.Offset))
	}
	a.add(report.Finding{
		RuleID:     "duplicate_moov",
		Status:     report.Fail,
		ByteOffset: second.Offset,
		FieldPath:  second.Path,
		Values:     vals,
		Delta:      &report.Delta{Value: int64(second.Offset - moovs[0].Offset), Unit: "byte"},
		Message:    fmt.Sprintf("%d moov boxes found at %s (a single moov is required)", len(moovs), joinComma(offs)),
	})
}

func (a *auditor) checkDuplicateMdat() {
	// 分片文件（含 moof）每片自带一个 mdat，是合法形态——不报。
	if len(isobmff.FindAll(a.root, "moof")) > 0 {
		return
	}
	mdats := isobmff.FindAll(a.root, "mdat")
	if len(mdats) < 2 {
		return
	}
	second := mdats[1]
	vals := make([]report.Value, 0, len(mdats))
	offs := make([]string, 0, len(mdats))
	for i, b := range mdats {
		vals = append(vals, report.Value{Name: fmt.Sprintf("mdat[%d]", i), Value: int64(b.Offset), Unit: "byte"})
		offs = append(offs, fmt.Sprintf("0x%X", b.Offset))
	}
	a.add(report.Finding{
		RuleID:     "duplicate_mdat",
		Status:     report.Fail,
		ByteOffset: second.Offset,
		FieldPath:  second.Path,
		Values:     vals,
		Delta:      &report.Delta{Value: int64(second.Offset - mdats[0].Offset), Unit: "byte"},
		Message:    fmt.Sprintf("%d mdat boxes found at %s (a non-fragmented file must have exactly one)", len(mdats), joinComma(offs)),
	})
}

// ── stco/co64 目标区间 ⊆ mdat ───────────────────────────────────────────

type region struct{ start, end uint64 }

func (a *auditor) checkChunks() {
	// 同一根因合并：重复 moov 时，偏移表指向哪份 moov 都不可信 → 跳过区间校验。
	if len(isobmff.FindAll(a.root, "moov")) >= 2 {
		return
	}
	mdats := isobmff.FindAll(a.root, "mdat")
	if len(mdats) == 0 {
		return
	}
	regions := make([]region, 0, len(mdats))
	for _, m := range mdats {
		regions = append(regions, region{start: m.PayloadStart(), end: m.PayloadEnd()})
	}
	for _, trak := range isobmff.FindAll(a.root, "trak") {
		stbl := isobmff.Find(trak, "stbl")
		if stbl == nil {
			continue
		}
		a.checkTrackChunks(stbl, regions)
	}
}

// maxChunks 是单轨校验收敛上限（畸形 count 字段的防御，避免无界循环）。
const maxChunks = 1 << 20

func (a *auditor) checkTrackChunks(stbl *isobmff.Box, regions []region) {
	offsets, base, width, offBox, ok := a.readChunkOffsets(stbl)
	if !ok || len(offsets) == 0 {
		return
	}
	sampleSize, _, perSample, ok := a.readSampleSizes(stbl)
	if !ok {
		return
	}
	spc := a.readStsc(stbl)
	if len(spc) == 0 {
		return
	}

	cursor := 0 // 样本游标
	for ci, off := range offsets {
		if ci >= maxChunks {
			break
		}
		per := samplesPerChunk(spc, uint32(ci+1))
		if per == 0 {
			continue
		}
		var length uint64
		for k := uint32(0); k < per; k++ {
			if sampleSize != 0 {
				length += uint64(sampleSize)
				continue
			}
			if cursor+int(k) >= len(perSample) {
				break
			}
			length += uint64(perSample[cursor+int(k)])
		}
		cursor += int(per)

		start := off
		end := off + length
		if containsRegion(regions, start, end) {
			continue
		}
		best, overshoot := nearestRegion(regions, start, end)
		entryOff := base + uint64(ci)*uint64(width)
		a.add(report.Finding{
			RuleID:     "stco_out_of_mdat",
			Status:     report.Fail,
			ByteOffset: entryOff,
			FieldPath:  fmt.Sprintf("%s[%d]", offBox.Path, ci),
			Values: []report.Value{
				{Name: "chunk_index", Value: int64(ci), Unit: "index"},
				{Name: "chunk_start", Value: int64(start), Unit: "byte"},
				{Name: "chunk_end", Value: int64(end), Unit: "byte"},
				{Name: "mdat_start", Value: int64(best.start), Unit: "byte"},
				{Name: "mdat_end", Value: int64(best.end), Unit: "byte"},
				{Name: "file_size", Value: int64(len(a.raw)), Unit: "byte"},
			},
			Delta: &report.Delta{Value: overshoot, Unit: "byte"},
			Message: fmt.Sprintf("chunk #%d targets [0x%X,0x%X) which is not inside any mdat payload [0x%X,0x%X)",
				ci, start, end, best.start, best.end),
		})
	}
}

// readChunkOffsets 返回条目值、条目区基址、条目宽度与承载盒。
func (a *auditor) readChunkOffsets(stbl *isobmff.Box) (offs []uint64, base uint64, width uint32, box *isobmff.Box, ok bool) {
	if stco := isobmff.Find(stbl, "stco"); stco != nil {
		count, g := u32(a.raw, stco.PayloadStart()+4)
		if !g {
			return nil, 0, 0, nil, false
		}
		base = stco.PayloadStart() + 8
		for i := uint32(0); i < count; i++ {
			v, g2 := u32(a.raw, base+uint64(i)*4)
			if !g2 {
				return nil, 0, 0, nil, false
			}
			offs = append(offs, uint64(v))
		}
		return offs, base, 4, stco, true
	}
	if co64 := isobmff.Find(stbl, "co64"); co64 != nil {
		count, g := u32(a.raw, co64.PayloadStart()+4)
		if !g {
			return nil, 0, 0, nil, false
		}
		base = co64.PayloadStart() + 8
		for i := uint32(0); i < count; i++ {
			v, g2 := u64(a.raw, base+uint64(i)*8)
			if !g2 {
				return nil, 0, 0, nil, false
			}
			offs = append(offs, v)
		}
		return offs, base, 8, co64, true
	}
	return nil, 0, 0, nil, false
}

func (a *auditor) readSampleSizes(stbl *isobmff.Box) (sampleSize, sampleCount uint32, per []uint32, ok bool) {
	stsz := isobmff.Find(stbl, "stsz")
	if stsz == nil {
		return 0, 0, nil, false
	}
	ss, g1 := u32(a.raw, stsz.PayloadStart()+4)
	sc, g2 := u32(a.raw, stsz.PayloadStart()+8)
	if !g1 || !g2 {
		return 0, 0, nil, false
	}
	if ss != 0 {
		return ss, sc, nil, true
	}
	base := stsz.PayloadStart() + 12
	n := sc
	if n > maxChunks {
		n = maxChunks
	}
	for i := uint32(0); i < n; i++ {
		v, g := u32(a.raw, base+uint64(i)*4)
		if !g {
			break
		}
		per = append(per, v)
	}
	return 0, sc, per, true
}

type stscEntry struct{ firstChunk, samplesPerChunk uint32 }

func (a *auditor) readStsc(stbl *isobmff.Box) []stscEntry {
	stsc := isobmff.Find(stbl, "stsc")
	if stsc == nil {
		return nil
	}
	cnt, g := u32(a.raw, stsc.PayloadStart()+4)
	if !g {
		return nil
	}
	base := stsc.PayloadStart() + 8
	n := cnt
	if n > maxChunks {
		n = maxChunks
	}
	out := make([]stscEntry, 0, n)
	for i := uint32(0); i < n; i++ {
		fc, g1 := u32(a.raw, base+uint64(i)*12)
		sp, g2 := u32(a.raw, base+uint64(i)*12+4)
		if !g1 || !g2 {
			break
		}
		out = append(out, stscEntry{firstChunk: fc, samplesPerChunk: sp})
	}
	return out
}

// samplesPerChunk 返回第 chunk（1 起）适用的 samples_per_chunk。
func samplesPerChunk(entries []stscEntry, chunk uint32) uint32 {
	var per uint32
	for _, e := range entries {
		if e.firstChunk <= chunk {
			per = e.samplesPerChunk
			continue
		}
		break
	}
	return per
}

// containsRegion 报告 [start,end) 是否落在任一 mdat 载荷区间内。
func containsRegion(regions []region, start, end uint64) bool {
	for _, r := range regions {
		if start >= r.start && end <= r.end {
			return true
		}
	}
	return false
}

// nearestRegion 返回"与目标区间最接近"的 mdat 区间及其越界字节数。
func nearestRegion(regions []region, start, end uint64) (region, int64) {
	var best region
	var overshoot int64 = -1
	for _, r := range regions {
		var o int64
		switch {
		case start > r.end:
			o = int64(start - r.end)
		case end > r.end:
			o = int64(end - r.end)
		case start < r.start:
			o = int64(r.start - start)
		default:
			o = 0
		}
		if overshoot < 0 || o < overshoot {
			overshoot, best = o, r
		}
	}
	if overshoot < 0 {
		overshoot = 0
	}
	return best, overshoot
}

// ── ftyp brand 与实际编码自洽 ───────────────────────────────────────────

// codecBrands 是"以编码命名的 major brand"。只有这些 brand 才可能与 stsd 样本条目
// 产生**确定矛盾**；通用 brand（isom/iso2/mp41/mp42/qt 等）不与任何编码冲突。
var codecBrands = map[string]struct{}{
	"avc1": {}, "hvc1": {}, "hev1": {}, "av01": {}, "vp09": {}, "mp4v": {},
}

// brandFamily 把一个 major brand 映射到它在 stsd 中应出现的样本条目类型集合。
var brandFamily = map[string][]string{
	"avc1": {"avc1"},
	"hvc1": {"hvc1", "hev1"},
	"hev1": {"hev1", "hvc1"},
	"av01": {"av01"},
	"vp09": {"vp09"},
	"mp4v": {"mp4v"},
}

func (a *auditor) checkFtypBrand() {
	ftyp := isobmff.Find(a.root, "ftyp")
	if ftyp == nil {
		return
	}
	major, ok := a.fourCC(ftyp.PayloadStart())
	if !ok {
		return
	}
	if _, isCodec := codecBrands[major]; !isCodec {
		return
	}
	present := a.sampleEntryTypes()
	if len(present) == 0 {
		// 无样本条目：无法断言编码与 brand 矛盾，留给别的规则（不制造 fail）。
		return
	}
	for _, want := range brandFamily[major] {
		if present[want] {
			return
		}
	}
	actual := ""
	for t := range present {
		if _, isVideo := brandFamily[t]; isVideo && actual == "" {
			actual = t
		}
	}
	a.add(report.Finding{
		RuleID:     "ftyp_brand_mismatch",
		Status:     report.Fail,
		ByteOffset: ftyp.Offset,
		FieldPath:  ftyp.Path,
		Values: []report.Value{
			{Name: "major_brand", Value: fourCCValue(major), Unit: "fourcc"},
			{Name: "stsd_entry", Value: fourCCValue(actual), Unit: "fourcc"},
		},
		Delta: &report.Delta{Value: fourCCValue(major) - fourCCValue(actual), Unit: "fourcc"},
		Message: fmt.Sprintf("ftyp major brand %q contradicts stsd sample entry %q",
			major, actual),
	})
}

// sampleEntryTypes 收集所有 stsd 里的样本条目类型（stsd 不是容器盒，
// 其子条目不在 IR 里，故直接读字节）。
func (a *auditor) sampleEntryTypes() map[string]bool {
	out := map[string]bool{}
	for _, stsd := range isobmff.FindAll(a.root, "stsd") {
		count, ok := u32(a.raw, stsd.PayloadStart()+4)
		if !ok {
			continue
		}
		off := stsd.PayloadStart() + 8
		for i := uint32(0); i < count && i < 64; i++ {
			size, ok1 := u32(a.raw, off)
			typ, ok2 := a.fourCC(off + 4)
			if !ok1 || !ok2 || size < 8 {
				break
			}
			out[typ] = true
			off += uint64(size)
		}
	}
	return out
}

// ── 只读小工具（全部带边界检查）────────────────────────────────────────

func (a *auditor) fourCC(off uint64) (string, bool) {
	if off+4 > uint64(len(a.raw)) {
		return "", false
	}
	return string(a.raw[off : off+4]), true
}

func u32(raw []byte, off uint64) (uint32, bool) {
	if off+4 > uint64(len(raw)) {
		return 0, false
	}
	return binary.BigEndian.Uint32(raw[off : off+4]), true
}

func u64(raw []byte, off uint64) (uint64, bool) {
	if off+8 > uint64(len(raw)) {
		return 0, false
	}
	return binary.BigEndian.Uint64(raw[off : off+8]), true
}

func fourCCValue(s string) int64 {
	if len(s) < 4 {
		return 0
	}
	return int64(binary.BigEndian.Uint32([]byte(s[:4])))
}

func childPath(parent *isobmff.Box, typ string) string {
	if parent == nil || parent.Path == "" {
		return typ
	}
	return parent.Path + "/" + typ
}

func joinComma(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}
