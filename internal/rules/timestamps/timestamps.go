// Package timestamps 实现模块 M3（时间戳单调性与 timescale 精度）。
//
// 规则集合见 plan/03 §四；口径（0 容差、极小 timescale 阈值、位宽）只在
// data/README §5.2/§5.4 与 internal/policy 定义。
//
// 纪律：
//   - 纯函数：只读已解析的盒子树与只读字节，不做 I/O、不读时钟、不读随机、不联网；
//   - 与生成器不共享代码（K8）：真值来自构造参数，本包只从字节推结论；
//   - **只看 DTS，不看 PTS**（data/README §5.4）：B 帧使 PTS 天然乱序，ctts 只用于
//     「不把 PTS 乱序当回退」——本包根本不参与 PTS 判定；
//   - **`ctts` 与 stts 内部分宽读**：`ctts` v1 偏移为**有符号**（本包不判 PTS，但仍按
//     分宽读以防把有符号偏移误当无符号大数——单元测试守）；`tfdt` v0 32 位 / v1 64 位；
//   - **同一根因合并**（F15）：stts 累计越过字段位宽（wrap）时，回退只是溢出的表象，
//     只报 `dts_overflow_risk`，**不**再报 `dts_regression`；一处 stts 根因只报一条。
//   - 样本级遍历**不落全量样本表**（R4）：按 `stts` **条目**推进累计 DTS（每条的
//     `count×duration` 用整数一次算完），只在越界点定位到条目号 + 样本序号。
//
// 为什么 `dts_regression` 在本夹具集里没有对应样本：`stts.duration` 是**无符号**，
// 累计 DTS 单调不减——**字面判据 `dts[n] < dts[n-1]` 对良构 stts 不可能为真**。
// 因此 `data/README §4` 的 C5「回退 DTS」在夹具里由两条**可达**形态表达：
// `zero_duration_entry`（C5：多条样本落在同一时刻）与 `tfdt_regression`
// （C5b/C5c：跨分片 tfdt 序倒退）。`dts_regression` 仍是模块 M3 的**完整判据**
// （证据塑形由本包单测守），只是未被本夹具集触发。
package timestamps

import (
	"encoding/binary"
	"fmt"

	"github.com/yuluo554/mediatrace/internal/isobmff"
	"github.com/yuluo554/mediatrace/internal/policy"
	"github.com/yuluo554/mediatrace/internal/report"
)

// ModuleID 是模块号。
const ModuleID = "M3"

// RuleIDs 是本模块实现的规则 ID 全集（plan/03 §四 六条），供测试与文档对账。
var RuleIDs = []string{
	"dts_regression",
	"tfdt_regression",
	"zero_duration_entry",
	"dts_overflow_risk",
	"timescale_suspect",
	"vfr_detected",
}

// maxEntries 收敛上限（畸形 count 字段的防御，避免无界循环）。
const maxEntries = 1 << 20

// 字段位宽上限（规范事实，非工程口径）：
//   - stts.duration 与 tfdt v0 的 baseMediaDecodeTime 是 32 位；
//   - tfdt v1 的 baseMediaDecodeTime 是 64 位（int64 表达不了 2^64，故 v1 不设越界）。
const (
	sttsWidthMax = uint64(0xFFFFFFFF)
	// tfdt 的越界上限按 version 取：v0 = 32 位，v1 = 64 位（不可越界）。
	tfdtV0WidthMax = uint64(0xFFFFFFFF)
)

// Audit 审计一个已解析文件，返回模块 M3 的全部 finding（已按确定性键排序）。
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
	a.checkTracks()
	a.checkFragments()
}

// ── 轨道级：stts（DTS）+ mdhd.timescale ─────────────────────────────────

func (a *auditor) checkTracks() {
	for _, trak := range isobmff.FindAll(a.root, "trak") {
		ts, tsOff, tsPath, hasTs := readMdhdTimescale(a.raw, trak)

		var entries []sttsEntry
		sttsPath := ""
		if stts := isobmff.Find(trak, "stts"); stts != nil {
			entries = readSttsEntries(a.raw, stts)
			sttsPath = stts.Path
		}

		overflowed := false
		if len(entries) > 0 {
			overflowed = a.checkStts(sttsPath, entries)
		}

		// vfr_detected（gray）：同轨 stts 条目数 > 1（VFR 本身不是缺陷）。
		// zero/overflow 已按根因分报，本条只表达"恒帧率推断不可用"。
		if len(entries) > 1 {
			a.add(report.Finding{
				RuleID:     "vfr_detected",
				Status:     report.Gray,
				ByteOffset: boxOffset(trak, "stts"),
				FieldPath:  sttsPath,
				GrayReason: report.GrayVFRDetected,
				Values: []report.Value{
					{Name: "stts_entries", Value: int64(len(entries)), Unit: "index"},
				},
				Message: fmt.Sprintf("stts has %d entries; constant-frame-rate inference is unavailable", len(entries)),
			})
		}

		// timescale_suspect（gray）：极小时基且**未**触发溢出（§5.2）。
		if hasTs && int64(ts) > 0 && int64(ts) < policy.TimescaleSuspectBelow && !overflowed {
			a.add(report.Finding{
				RuleID:     "timescale_suspect",
				Status:     report.Gray,
				ByteOffset: tsOff,
				FieldPath:  tsPath + ".timescale",
				GrayReason: report.GrayTimescaleSuspect,
				Values: []report.Value{
					{Name: "mdhd_timescale", Value: int64(ts), Unit: "hz"},
					{Name: "timescale_suspect_below", Value: policy.TimescaleSuspectBelow, Unit: "hz"},
				},
				Message: fmt.Sprintf("mdhd.timescale %d is below the precision floor %d (overflow-risk precursor, not a hard violation)", ts, policy.TimescaleSuspectBelow),
			})
		}
	}
}

// sttsEntry 是 stts 的一条（count 个样本，每个 duration tick）。
type sttsEntry struct {
	Count    uint32
	Duration uint32
	DurOff   uint64 // 该条的 duration 字段绝对偏移
}

// checkStts 按条目推进累计 DTS：判位宽越界（32 位）与回退（0 容差）。
// 返回该轨是否触发越界（供 timescale_suspect 排除）。
//
// 同一根因合并（F15）：越界一旦发生即返回，wrap 造成的"回退"不再单报。
func (a *auditor) checkStts(sttsPath string, entries []sttsEntry) (overflowed bool) {
	tol := policy.DTSRegressionTolerance
	var cum uint64       // 累计 DTS（真实值；早退保证 cum < sttsWidthMax + one step）
	prevEnd := int64(0)  // 上一段终点的 int64 视图（回退判据用）
	havePrev := false

	for i, e := range entries {
		prod := uint64(e.Count) * uint64(e.Duration)
		end := cum + prod

		if end > sttsWidthMax {
			// 越界点：定位到该条目内**第一个**使累计越过上限的样本。
			k := 0
			if d := uint64(e.Duration); d != 0 {
				k = int((sttsWidthMax - cum) / d)
			}
			a.addSttsOverflow(sttsPath, i, k, e.DurOff, int64(end), int64(sttsWidthMax))
			return true
		}

		// 回退判据（§5.4：dts[n] < dts[n-1] - 容差）。stts.duration 无符号 ⇒ 累计单调，
		// 该分支对良构 stts 恒不成立；保留为完整判据（见包注释）。
		if havePrev {
			if int64(cum) < prevEnd-tol {
				a.addDTSRegression(sttsPath, i, int(e.Count), e.DurOff, prevEnd, int64(cum))
				return false
			}
		}
		cum = end
		prevEnd = int64(end)
		havePrev = true

		// zero_duration_entry（fail）：duration == 0（多条样本落在同一时刻）。
		if e.Duration == 0 {
			nominal := firstNonZeroDuration(entries)
			a.addZeroDuration(sttsPath, i, e.Count, e.DurOff, nominal)
		}
	}
	return false
}

// ── 分片级：moof/traf/tfdt（跨片回退 + 位宽）────────────────────────────

type fragInfo struct {
	index  int    // moof 在文件内的序号（0 起）
	tfdt   uint64 // baseMediaDecodeTime 原值
	v1     bool   // tfdt version
	valOff uint64 // 该值的绝对偏移
	path   string // tfdt 盒路径
	dur    int64  // 该分片的样本时长和（媒体时基；推不出为 0）
}

func (a *auditor) checkFragments() {
	// 按文件内顺序收集 tfdt；按 track_id 分组（同一轨道的分片才互相比较）。
	order := []uint32{}
	byTrack := map[uint32][]fragInfo{}
	for mi, moof := range isobmff.FindAll(a.root, "moof") {
		for _, traf := range isobmff.FindAll(moof, "traf") {
			trackID, defaultDur := readTfhd(a.raw, traf)
			tfdtBox := isobmff.Find(traf, "tfdt")
			if tfdtBox == nil {
				continue
			}
			v1, val, valOff, ok := readTfdt(a.raw, tfdtBox)
			if !ok {
				continue
			}
			dur := int64(0)
			if trun := isobmff.Find(traf, "trun"); trun != nil {
				if total, ok := readTrunSum(a.raw, trun, defaultDur); ok {
					dur = total
				}
			}
			if _, seen := byTrack[trackID]; !seen {
				order = append(order, trackID)
			}
			byTrack[trackID] = append(byTrack[trackID], fragInfo{
				index: mi, tfdt: val, v1: v1, valOff: valOff, path: tfdtBox.Path, dur: dur,
			})
		}
	}

	tol := policy.DTSRegressionTolerance
	for _, id := range order {
		frags := byTrack[id]
		for k := 1; k < len(frags); k++ {
			cur, prev := frags[k], frags[k-1]
			if int64(cur.tfdt) < int64(prev.tfdt)-tol {
				a.addTfdtRegression(cur, prev)
				continue
			}
			a.checkTfdtOverflow(cur)
		}
		if len(frags) > 0 {
			a.checkTfdtOverflow(frags[0])
		}
	}
}

// ── finding 塑形（每条 fail 四元组齐：偏移 + 字段路径 + 值 + 差值）───────

func (a *auditor) addZeroDuration(sttsPath string, entry int, count uint32, durOff, nominal uint64) {
	delta := int64(nominal) // 相对同轨首个非零样本时长的**亏空**；无参考则为 0
	a.add(report.Finding{
		RuleID:     "zero_duration_entry",
		Status:     report.Fail,
		ByteOffset: durOff,
		FieldPath:  fmt.Sprintf("%s[%d].duration", sttsPath, entry),
		Values: []report.Value{
			{Name: "stts_entry", Value: int64(entry), Unit: "index"},
			{Name: "samples", Value: int64(count), Unit: "sample"},
			{Name: "duration", Value: 0, Unit: "tick"},
			{Name: "stts_nominal_duration", Value: int64(nominal), Unit: "tick"},
		},
		Delta:   &report.Delta{Value: delta, Unit: "tick"},
		Message: fmt.Sprintf("stts entry #%d declares duration 0 for %d samples; they collapse onto one instant", entry, count),
	})
}

func (a *auditor) addSttsOverflow(sttsPath string, entry, sample int, durOff uint64, crossed, widthMax int64) {
	a.add(report.Finding{
		RuleID:     "dts_overflow_risk",
		Status:     report.Fail,
		ByteOffset: durOff,
		FieldPath:  fmt.Sprintf("%s[%d].duration", sttsPath, entry),
		Values: []report.Value{
			{Name: "stts_entry", Value: int64(entry), Unit: "index"},
			{Name: "sample_index", Value: int64(sample), Unit: "index"},
			{Name: "cumulative_dts", Value: crossed, Unit: "tick"},
			{Name: "field_width_max", Value: widthMax, Unit: "tick"},
		},
		Delta:   &report.Delta{Value: crossed - widthMax, Unit: "tick"},
		Message: fmt.Sprintf("cumulative DTS %d crosses the 32-bit stts field limit %d at stts entry #%d (sample #%d)", crossed, widthMax, entry, sample),
	})
}

func (a *auditor) addDTSRegression(sttsPath string, entry, sample int, durOff uint64, prev, cur int64) {
	a.add(report.Finding{
		RuleID:     "dts_regression",
		Status:     report.Fail,
		ByteOffset: durOff,
		FieldPath:  fmt.Sprintf("%s[%d].duration", sttsPath, entry),
		Values: []report.Value{
			{Name: "sample_index", Value: int64(sample), Unit: "index"},
			{Name: "stts_entry", Value: int64(entry), Unit: "index"},
			{Name: "dts", Value: cur, Unit: "tick"},
			{Name: "dts_previous", Value: prev, Unit: "tick"},
		},
		Delta:   &report.Delta{Value: prev - cur, Unit: "tick"},
		Message: fmt.Sprintf("DTS regresses at sample #%d (%d < %d) in stts entry #%d", sample, cur, prev, entry),
	})
}

func (a *auditor) addTfdtRegression(cur, prev fragInfo) {
	a.add(report.Finding{
		RuleID:     "tfdt_regression",
		Status:     report.Fail,
		ByteOffset: cur.valOff,
		FieldPath:  cur.path + ".base_media_decode_time",
		Values: []report.Value{
			{Name: "fragment_index", Value: int64(cur.index), Unit: "index"},
			{Name: "tfdt", Value: int64(cur.tfdt), Unit: "tick"},
			{Name: "tfdt_previous", Value: int64(prev.tfdt), Unit: "tick"},
			{Name: "tfdt_offset", Value: int64(cur.valOff), Unit: "byte"},
			{Name: "tfdt_previous_offset", Value: int64(prev.valOff), Unit: "byte"},
		},
		Delta:   &report.Delta{Value: int64(prev.tfdt) - int64(cur.tfdt), Unit: "tick"},
		Message: fmt.Sprintf("tfdt regresses at fragment #%d (%d < %d)", cur.index, cur.tfdt, prev.tfdt),
	})
}

func (a *auditor) checkTfdtOverflow(f fragInfo) {
	if f.v1 {
		return // 64 位头寸：v1 的值本身表达不了 2^64 之上，不设越界
	}
	end := f.tfdt + uint64(f.dur)
	if end <= tfdtV0WidthMax {
		return
	}
	a.add(report.Finding{
		RuleID:     "dts_overflow_risk",
		Status:     report.Fail,
		ByteOffset: f.valOff,
		FieldPath:  f.path + ".base_media_decode_time",
		Values: []report.Value{
			{Name: "fragment_index", Value: int64(f.index), Unit: "index"},
			{Name: "tfdt", Value: int64(f.tfdt), Unit: "tick"},
			{Name: "fragment_duration", Value: f.dur, Unit: "tick"},
			{Name: "field_width_max", Value: int64(tfdtV0WidthMax), Unit: "tick"},
		},
		Delta:   &report.Delta{Value: int64(end) - int64(tfdtV0WidthMax), Unit: "tick"},
		Message: fmt.Sprintf("fragment #%d tfdt %d + duration %d crosses the 32-bit tfdt v0 limit %d", f.index, f.tfdt, f.dur, tfdtV0WidthMax),
	})
}

// firstNonZeroDuration 返回同轨首个非零样本时长（作 zero_duration 的差值参考）；无则 0。
func firstNonZeroDuration(entries []sttsEntry) uint64 {
	for _, e := range entries {
		if e.Duration != 0 {
			return uint64(e.Duration)
		}
	}
	return 0
}

// ── 盒子读数（带边界检查；version 0/1 两形态）───────────────────────────

func boxOffset(root *isobmff.Box, typ string) uint64 {
	if b := isobmff.Find(root, typ); b != nil {
		return b.Offset
	}
	return 0
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

func readSttsEntries(raw []byte, stts *isobmff.Box) []sttsEntry {
	p := stts.PayloadStart()
	cnt, ok := u32at(raw, p+4)
	if !ok {
		return nil
	}
	if cnt > maxEntries {
		cnt = maxEntries
	}
	base := p + 8
	out := make([]sttsEntry, 0, 8)
	for i := uint32(0); i < cnt; i++ {
		off := base + uint64(i)*8
		c, ok1 := u32at(raw, off)
		d, ok2 := u32at(raw, off+4)
		if !ok1 || !ok2 {
			break
		}
		out = append(out, sttsEntry{Count: c, Duration: d, DurOff: off + 4})
	}
	return out
}

// readTfhd 返回 track_id 与默认样本时长（flags 决定字段布局）。
func readTfhd(raw []byte, traf *isobmff.Box) (trackID, defaultDur uint32) {
	tfhd := isobmff.Find(traf, "tfhd")
	if tfhd == nil {
		return 0, 0
	}
	p := tfhd.PayloadStart()
	flags, ok := u32at(raw, p)
	if !ok {
		return 0, 0
	}
	f := flags & 0x00FFFFFF
	if id, ok := u32at(raw, p+4); ok {
		trackID = id
	}
	pos := p + 8
	if f&0x000001 != 0 {
		pos += 8 // base-data-offset
	}
	if f&0x000002 != 0 {
		pos += 4 // sample-description-index
	}
	if f&0x000008 != 0 {
		if v, ok := u32at(raw, pos); ok {
			defaultDur = v
		}
	}
	return trackID, defaultDur
}

func readTfdt(raw []byte, tfdt *isobmff.Box) (v1 bool, value, valOff uint64, ok bool) {
	p := tfdt.PayloadStart()
	ver, ok := u8at(raw, p)
	if !ok {
		return false, 0, 0, false
	}
	if ver == 1 {
		v, ok := u64at(raw, p+4)
		return true, v, p + 4, ok
	}
	v, ok := u32at(raw, p+4)
	return false, uint64(v), p + 4, ok
}

// readTrunSum 返回一个 trun 各样本时长之和（媒体时基 tick）。
func readTrunSum(raw []byte, trun *isobmff.Box, fallback uint32) (int64, bool) {
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

func u64at(raw []byte, off uint64) (uint64, bool) {
	if off+8 > uint64(len(raw)) {
		return 0, false
	}
	return binary.BigEndian.Uint64(raw[off : off+8]), true
}