// Package display 实现模块 M5（旋转 / SAR / HDR / 色彩标记矛盾）。
//
// 规则集合见 plan/03 §六；口径（正交性容差、整数运算）只在 data/README §5.3 与
// internal/policy 定义。matrix 按 **16.16 定点整数**读取，正交性用行列点积与 1.0 的偏差
// 判定（`policy.MatrixOrthoTolerance` + `policy.Square`），**不做浮点近似**（K15）。
//
// 纪律：
//   - 纯函数：只读已解析的盒子树与只读字节，不做 I/O、不读时钟、不读随机、不联网；
//   - 与生成器不共享代码（K8）：真值来自构造参数，本包只从字节推结论；
//   - **`colr` 多实例按文件内顺序全列，不取第一个**（D10/K6）：只报一条 `colr_multi_instance`；
//   - **不做**绝对亮度 / 绝对色彩正确性判定：只判量程自检与 `clli`/`mdcv` 互斥（plan/01 §五 M5）。
package display

import (
	"encoding/binary"
	"fmt"

	"github.com/yuluo554/mediatrace/internal/isobmff"
	"github.com/yuluo554/mediatrace/internal/policy"
	"github.com/yuluo554/mediatrace/internal/report"
)

// ModuleID 是模块号。
const ModuleID = "M5"

// RuleIDs 是本模块实现的规则 ID 全集（plan/03 §六 六条），供测试与文档对账。
var RuleIDs = []string{
	"matrix_not_orthogonal",
	"rotation_dimension_conflict",
	"sar_dar_conflict",
	"hdr_metadata_missing",
	"hdr_range_suspect",
	"colr_multi_instance",
}

// q16One 是 16.16 定点里的 1.0（=65536）。
const q16One int64 = 1 << 16

// Audit 审计一个已解析文件，返回模块 M5 的全部 finding（已按确定性键排序）。
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
		a.checkTrack(trak)
	}
}

// ── 轨道级 ──────────────────────────────────────────────────────────────

type colorBox struct {
	primaries uint16
	offset    uint64
}

type trackDisplay struct {
	tkhdPath  string
	matrixOff uint64
	matrix    [9]int32
	hasTkhd   bool
	widthQ16  int64 // tkhd.width（16.16，呈现方向）
	heightQ16 int64

	stsdPath string
	storedW  int64 // stsd 样本条目存储宽（像素）
	storedH  int64

	hasPasp bool
	paspNum int64
	paspDen int64
	paspOff uint64

	colrs []colorBox

	hasMdcv  bool
	mdcvMaxL int64
	mdcvMinL int64
	mdcvMaxC int64
	mdcvMinC int64
	mdcvOff  uint64

	hasClli    bool
	clliMaxCLL int64
	clliAvgCLL int64
	clliOff    uint64
}

func (a *auditor) checkTrack(trak *isobmff.Box) {
	t := a.readTrack(trak)
	if !t.hasTkhd {
		return
	}
	a.checkMatrixOrtho(t)
	rot := rotationDeg(t.matrix)
	a.checkRotationConflict(t, rot)
	a.checkSarDar(t, rot)
	a.checkColr(t)
	a.checkHdr(t)
}

// ── 规则 1：matrix 正交性 ───────────────────────────────────────────────

func (a *auditor) checkMatrixOrtho(t *trackDisplay) {
	a11, a12 := int64(t.matrix[0]), int64(t.matrix[1])
	a21, a22 := int64(t.matrix[3]), int64(t.matrix[4])

	// 两行（也即两列，3×3 的旋转只在左上 2×2）应各自是单位向量且互相垂直。
	row1 := policy.Square(a11) + policy.Square(a12)
	row2 := policy.Square(a21) + policy.Square(a22)
	cross := a11*a21 + a12*a22
	one := policy.Square(q16One) // 1.0 在 16.16 点积里的期望值 = 65536²

	tol := policy.MatrixOrthoTolerance * q16One // 容差换算到点积原单位
	dev1, dev2, dev3 := absInt(row1-one), absInt(row2-one), absInt(cross)
	worst := dev1
	if dev2 > worst {
		worst = dev2
	}
	if dev3 > worst {
		worst = dev3
	}
	if worst <= tol {
		return
	}
	a.add(report.Finding{
		RuleID:     "matrix_not_orthogonal",
		Status:     report.Fail,
		ByteOffset: t.matrixOff,
		FieldPath:  t.tkhdPath + ".matrix",
		Values: []report.Value{
			{Name: "row1_dot", Value: row1, Unit: "1/65536^2"},
			{Name: "row2_dot", Value: row2, Unit: "1/65536^2"},
			{Name: "row1_row2_dot", Value: cross, Unit: "1/65536^2"},
			{Name: "expected_unit_dot", Value: one, Unit: "1/65536^2"},
			{Name: "matrix_ortho_tolerance", Value: policy.MatrixOrthoTolerance, Unit: "1/65536"},
		},
		Delta:   &report.Delta{Value: worst / q16One, Unit: "1/65536"},
		Message: fmt.Sprintf("display matrix is not orthogonal: row dot products deviate from 1.0 by up to %d (1/65536 units)", worst/q16One),
	})
}

// ── 规则 2：旋转 ±90° 时的呈现宽高冲突 ─────────────────────────────────

func (a *auditor) checkRotationConflict(t *trackDisplay, rot int) {
	if rot != 90 && rot != -90 {
		return
	}
	if t.hasPasp {
		return // 有 pasp 说明：宽高由像素宽高比补齐，不算冲突
	}
	if t.widthQ16 >= t.heightQ16 {
		return // 呈现宽 >= 高：不进入宽高互换规则
	}
	wp, hp := t.widthQ16>>16, t.heightQ16>>16
	a.add(report.Finding{
		RuleID:     "rotation_dimension_conflict",
		Status:     report.Fail,
		ByteOffset: t.matrixOff,
		FieldPath:  t.tkhdPath + ".matrix",
		Values: []report.Value{
			{Name: "rotation_deg", Value: int64(rot), Unit: "deg"},
			{Name: "tkhd_width", Value: wp, Unit: "px"},
			{Name: "tkhd_height", Value: hp, Unit: "px"},
		},
		Delta: &report.Delta{Value: hp - wp, Unit: "px"},
		Message: fmt.Sprintf("display matrix rotates %d degrees but tkhd declares width %d < height %d with no pasp — presentation dimensions were not swapped",
			rot, wp, hp),
	})
}

// ── 规则 3：pasp 与 DAR 一致性 ──────────────────────────────────────────

func (a *auditor) checkSarDar(t *trackDisplay, rot int) {
	if !t.hasPasp || t.paspNum <= 0 || t.paspDen <= 0 {
		return
	}
	if t.storedW <= 0 || t.storedH <= 0 || t.widthQ16 <= 0 || t.heightQ16 <= 0 {
		return
	}
	// 无旋转：tkhd_DAR == stored_W*pasp.num / (stored_H*pasp.den)
	// ±90°：呈现宽高互换，DAR 取倒数
	var left, right int64
	if rot == 90 || rot == -90 {
		left = t.widthQ16 * t.storedW * t.paspNum
		right = t.heightQ16 * t.storedH * t.paspDen
	} else {
		left = t.widthQ16 * t.storedH * t.paspDen
		right = t.heightQ16 * t.storedW * t.paspNum
	}
	if left == right {
		return
	}
	wp, hp := t.widthQ16>>16, t.heightQ16>>16
	impliedW := t.storedW * t.paspNum / t.paspDen
	var delta int64
	if rot == 90 || rot == -90 {
		delta = hp - impliedW
	} else {
		delta = wp - impliedW
	}
	a.add(report.Finding{
		RuleID:     "sar_dar_conflict",
		Status:     report.Fail,
		ByteOffset: t.paspOff,
		FieldPath:  t.stsdPath + "/pasp",
		Values: []report.Value{
			{Name: "pasp_num", Value: t.paspNum, Unit: "ratio"},
			{Name: "pasp_den", Value: t.paspDen, Unit: "ratio"},
			{Name: "stored_width", Value: t.storedW, Unit: "px"},
			{Name: "stored_height", Value: t.storedH, Unit: "px"},
			{Name: "tkhd_width", Value: wp, Unit: "px"},
			{Name: "tkhd_height", Value: hp, Unit: "px"},
		},
		Delta: &report.Delta{Value: absInt(delta), Unit: "px"},
		Message: fmt.Sprintf("pasp %d:%d with stored %dx%d yields a display aspect that contradicts tkhd %dx%d",
			t.paspNum, t.paspDen, t.storedW, t.storedH, wp, hp),
	})
}

// ── 规则 4/5：HDR 元数据缺失（gray）/ 量程自检（fail）────────────────────

func (a *auditor) checkColr(t *trackDisplay) {
	if len(t.colrs) < 2 {
		return
	}
	vals := make([]report.Value, 0, len(t.colrs)+1)
	for i, c := range t.colrs {
		vals = append(vals, report.Value{Name: fmt.Sprintf("colr[%d].primaries", i), Value: int64(c.primaries), Unit: "index"})
	}
	vals = append(vals, report.Value{Name: "colr_count", Value: int64(len(t.colrs)), Unit: "index"})
	a.add(report.Finding{
		RuleID:     "colr_multi_instance",
		Status:     report.Fail,
		ByteOffset: t.colrs[0].offset,
		FieldPath:  t.stsdPath + "/colr[0]",
		Values:     vals,
		Delta:      &report.Delta{Value: int64(len(t.colrs) - 1), Unit: "index"},
		Message: fmt.Sprintf("sample entry carries %d colr boxes; they are listed in file order (D10: not just the first)", len(t.colrs)),
	})
}

func (a *auditor) checkHdr(t *trackDisplay) {
	// 广色域 colr（bt2020 primaries=9 / Display-P3 primaries=12）但无 mdcv/clli ⇒ gray。
	if !t.hasMdcv && !t.hasClli {
		for _, c := range t.colrs {
			if c.primaries == 9 || c.primaries == 12 {
				a.add(report.Finding{
					RuleID:     "hdr_metadata_missing",
					Status:     report.Gray,
					ByteOffset: c.offset,
					FieldPath:  t.stsdPath + "/colr",
					GrayReason: report.GrayHDRMetadataMissing,
					Values: []report.Value{
						{Name: "colr_primaries", Value: int64(c.primaries), Unit: "index"},
						{Name: "has_mdcv", Value: 0, Unit: "flag"},
						{Name: "has_clli", Value: 0, Unit: "flag"},
					},
					Message: "wide-gamut colr present but neither mdcv nor clli is declared — HDR mastering metadata is missing (not necessarily a defect)",
				})
				break
			}
		}
	}

	// 量程自检：只判 max>min 与 clli/mdcv 互斥，不判绝对亮度。
	if t.hasMdcv && (t.mdcvMaxL <= t.mdcvMinL || t.mdcvMaxC <= t.mdcvMinC) {
		delta := t.mdcvMinL - t.mdcvMaxL
		name := "max_luminance"
		if t.mdcvMaxC <= t.mdcvMinC {
			delta = t.mdcvMinC - t.mdcvMaxC
			name = "max_chromaticity"
		}
		a.add(report.Finding{
			RuleID:     "hdr_range_suspect",
			Status:     report.Fail,
			ByteOffset: t.mdcvOff,
			FieldPath:  t.stsdPath + "/mdcv." + name,
			Values: []report.Value{
				{Name: "max_luminance", Value: t.mdcvMaxL, Unit: "nits"},
				{Name: "min_luminance", Value: t.mdcvMinL, Unit: "nits"},
				{Name: "max_chromaticity", Value: t.mdcvMaxC, Unit: "1e5"},
				{Name: "min_chromaticity", Value: t.mdcvMinC, Unit: "1e5"},
			},
			Delta:   &report.Delta{Value: delta, Unit: "nits"},
			Message: "mdcv range is self-contradictory: max <= min",
		})
	}
	if t.hasClli && t.hasMdcv && t.clliMaxCLL != t.mdcvMaxL {
		a.add(report.Finding{
			RuleID:     "hdr_range_suspect",
			Status:     report.Fail,
			ByteOffset: t.clliOff,
			FieldPath:  t.stsdPath + "/clli.max_content_light_level",
			Values: []report.Value{
				{Name: "clli_max_cll", Value: t.clliMaxCLL, Unit: "nits"},
				{Name: "mdcv_max_luminance", Value: t.mdcvMaxL, Unit: "nits"},
			},
			Delta:   &report.Delta{Value: absInt(t.clliMaxCLL - t.mdcvMaxL), Unit: "nits"},
			Message: fmt.Sprintf("clli max_content_light_level %d contradicts mdcv max_luminance %d", t.clliMaxCLL, t.mdcvMaxL),
		})
	}
	if t.hasClli && t.clliAvgCLL > t.clliMaxCLL {
		a.add(report.Finding{
			RuleID:     "hdr_range_suspect",
			Status:     report.Fail,
			ByteOffset: t.clliOff,
			FieldPath:  t.stsdPath + "/clli.max_content_light_level",
			Values: []report.Value{
				{Name: "clli_max_cll", Value: t.clliMaxCLL, Unit: "nits"},
				{Name: "clli_avg_cll", Value: t.clliAvgCLL, Unit: "nits"},
			},
			Delta:   &report.Delta{Value: t.clliAvgCLL - t.clliMaxCLL, Unit: "nits"},
			Message: "clli average_content_light_level exceeds max_content_light_level",
		})
	}
}

// ── 读数 ────────────────────────────────────────────────────────────────

func (a *auditor) readTrack(trak *isobmff.Box) *trackDisplay {
	t := &trackDisplay{}
	if tkhd := isobmff.Find(trak, "tkhd"); tkhd != nil {
		t.tkhdPath = tkhd.Path
		if m, mOff, w, h, ok := readTkhd(a.raw, tkhd); ok {
			t.hasTkhd, t.matrix, t.matrixOff = true, m, mOff
			t.widthQ16, t.heightQ16 = w, h
		}
	}
	if stsd := isobmff.Find(trak, "stsd"); stsd != nil {
		t.stsdPath = stsd.Path
		codec, entryStart, entryEnd, childStart, ok := sampleEntry(a.raw, stsd)
		if ok {
			if entryHeadLen(codec) == 78 {
				if w, ok := u16at(a.raw, entryStart+32); ok {
					t.storedW = int64(w)
				}
				if h, ok := u16at(a.raw, entryStart+34); ok {
					t.storedH = int64(h)
				}
			}
			a.readEntryChildren(t, childStart, entryEnd)
		}
	}
	return t
}

func (a *auditor) readEntryChildren(t *trackDisplay, start, end uint64) {
	for off := start; off+8 <= end; {
		sz, ok1 := u32at(a.raw, off)
		typ, ok2 := fourCC(a.raw, off+4)
		if !ok1 || !ok2 || sz < 8 {
			return
		}
		p := off + 8
		switch typ {
		case "pasp":
			if n, ok1 := u32at(a.raw, p); ok1 {
				if d, ok2 := u32at(a.raw, p+4); ok2 {
					t.hasPasp, t.paspNum, t.paspDen, t.paspOff = true, int64(n), int64(d), off
				}
			}
		case "colr":
			if a.field(a.raw, p) == "nclx" {
				if pr, ok := u16at(a.raw, p+4); ok {
					t.colrs = append(t.colrs, colorBox{primaries: pr, offset: off})
				}
			}
		case "mdcv":
			if maxL, ok := u32at(a.raw, p+8); ok {
				minL, _ := u32at(a.raw, p+12)
				maxC, _ := u32at(a.raw, p+16)
				minC, _ := u32at(a.raw, p+20)
				t.hasMdcv, t.mdcvOff = true, off
				t.mdcvMaxL, t.mdcvMinL = int64(maxL), int64(minL)
				t.mdcvMaxC, t.mdcvMinC = int64(maxC), int64(minC)
			}
		case "clli":
			if maxCLL, ok := u16at(a.raw, p); ok {
				avgCLL, _ := u16at(a.raw, p+2)
				t.hasClli, t.clliOff = true, off
				t.clliMaxCLL, t.clliAvgCLL = int64(maxCLL), int64(avgCLL)
			}
		}
		off += uint64(sz)
	}
}

func (a *auditor) field(raw []byte, off uint64) string {
	if off+4 > uint64(len(raw)) {
		return ""
	}
	return string(raw[off : off+4])
}

// readTkhd 返回 matrix（9×int32）、matrix 字段偏移，以及 width/height（16.16）。
func readTkhd(raw []byte, tkhd *isobmff.Box) (m [9]int32, matrixOff uint64, w, h int64, ok bool) {
	p := tkhd.PayloadStart()
	ver, ok := u8at(raw, p)
	if !ok {
		return m, 0, 0, 0, false
	}
	base := p + 40 // v0：vf4 + co/mod/id/reserved/duration(20) + reserved2(8) + layer..(8)
	if ver == 1 {
		base = p + 44
	}
	for i := 0; i < 9; i++ {
		v, ok := i32at(raw, base+uint64(i)*4)
		if !ok {
			return m, 0, 0, 0, false
		}
		m[i] = v
	}
	wv, ok1 := u32at(raw, base+36)
	hv, ok2 := u32at(raw, base+40)
	if !ok1 || !ok2 {
		return m, 0, 0, 0, false
	}
	return m, base, int64(wv), int64(hv), true
}

// rotationDeg 从 16.16 matrix 识别旋转角：仅识别 ±90°（宽高互换规则只关心这一档），返回 0 表示非 ±90°。
func rotationDeg(m [9]int32) int {
	m00, m01, m10, m11 := int64(m[0]), int64(m[1]), int64(m[3]), int64(m[4])
	if m00 != 0 || m11 != 0 {
		return 0
	}
	if absInt(m01) != q16One || absInt(m10) != q16One {
		return 0
	}
	if m01 > 0 {
		return 90
	}
	return -90
}

// sampleEntry 返回 stsd 首个样本条目的编码 fourcc 与其子盒区间。
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

func entryHeadLen(codec string) uint64 {
	switch codec {
	case "avc1", "avc3", "hvc1", "hev1", "av01", "vp09", "vp08", "mp4v", "dvh1", "dvhe", "dav1":
		return 78
	case "mp4a", "Opus", "ac-3", "ec-3", "ac-4", "alac", "fLaC", "lpcm", "sowt", "twos", "ulaw", "alaw":
		return 28
	}
	return 0
}

// ── 只读小工具（全部带边界检查）────────────────────────────────────────

func absInt(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func fourCC(raw []byte, off uint64) (string, bool) {
	if off+4 > uint64(len(raw)) {
		return "", false
	}
	return string(raw[off : off+4]), true
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
