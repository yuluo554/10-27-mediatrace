// builders_controls.go 是对照组（"不误伤"是门禁，不是加分项）。
//
// 四组（规模与期望由 gen_test.go 断言）：
//
//	controls       每类至少 1 个合法样本（C1–C7 的"合法近邻"）
//	m3-regression  ≥20 个正常 H.264/H.265（**必须含 B 帧**：ctts v0/v1 有符号偏移路径）
//	m4             正反例各 5（正确裁剪 vs 未声明）
//	m5             正反例各 5（正常竖屏/横屏/SAR vs 冲突）
//
// 所有对照组样本期望 **pass**（findings 为空），误报 0 是 M2–M5 的出口门。
package gen

import "fmt"

// controlBuilders 是对照组样本（期望 pass，findings 为空）。
var controlBuilders = []Builder{
	// ── 每类至少 1 个合法样本 ──────────────────────────────────────────
	{
		ID: "K1-moov", Suite: "controls", Filename: "k1_single_moov.mp4",
		ExpectStatus: ExpectPass,
		Note:         "单个 moov + 单个 mdat（C1 的合法近邻）",
		Build:        buildK1SingleMoov,
	},
	{
		ID: "K2-stco", Suite: "controls", Filename: "k2_stco_in_mdat.mp4",
		ExpectStatus: ExpectPass,
		Note:         "stco 指向 mdat 载荷内且不越尾（C2 的合法近邻）",
		Build:        buildK2StcoInMdat,
	},
	{
		ID: "K3-mdat", Suite: "controls", Filename: "k3_mdat_size0_tail.mp4",
		ExpectStatus: ExpectPass,
		Note:         "mdat 用 size==0（到文件尾）合法形态（C3 的合法近邻）",
		Build:        buildK3MdatSizeZero,
	},
	{
		ID: "K3b-largesize", Suite: "controls", Filename: "k3b_mdat_largesize.mp4",
		ExpectStatus: ExpectPass,
		Note:         "mdat 用 size==1 + 64 位 largesize 合法形态（C3 的第二合法形态）",
		Build:        buildK3bMdatLargesize,
	},
	{
		ID: "K4-duration", Suite: "controls", Filename: "k4_duration_aligned.mp4",
		ExpectStatus: ExpectPass,
		Note:         "mvhd/tkhd/Σstts 三口径对齐（1 tick 容差内）（C4 的合法近邻）",
		Build:        buildK4DurationAligned,
	},
	{
		ID: "K5-dts", Suite: "controls", Filename: "k5_monotonic_dts.mp4",
		ExpectStatus: ExpectPass,
		Note:         "DTS 严格单调、单 stts 条目（C5 的合法近邻；不触发 vfr_detected）",
		Build:        buildK5MonotonicDts,
	},
	{
		ID: "K6-priming", Suite: "controls", Filename: "k6_priming_trimmed.mp4",
		ExpectStatus: ExpectPass,
		Note:         "已用 elst 正确裁剪 1024 且 roll 距离一致（C6 的合法近邻）",
		Build:        buildK6PrimingTrimmed,
	},
	{
		ID: "K7-portrait", Suite: "controls", Filename: "k7_portrait_rot90_pasp.mp4",
		ExpectStatus: ExpectPass,
		Note:         "正常竖屏：matrix=90 + 存储 1080×1920 + pasp 说明（C7 的合法近邻）",
		Build:        buildK7PortraitRot90,
	},

	// ── 其余合法形态（扩大"不误伤"覆盖）──────────────────────────────
	{
		ID: "K8-rot180", Suite: "controls", Filename: "k8_rot180_landscape.mp4",
		ExpectStatus: ExpectPass,
		Note:         "matrix=180° 正交旋转、横屏宽 > 高、无 pasp——180° 不进入宽高互换规则",
		Build:        buildK8Rot180,
	},
	{
		ID: "K9-frag", Suite: "controls", Filename: "k9_frag_ordered_tfdt.mp4",
		ExpectStatus: ExpectPass,
		Note:         "分片 tfdt 正确递增 + trun 带 sample_duration（C5b/F1 的合法近邻）",
		Build:        buildK9FragOrdered,
	},
	{
		ID: "K10-empty-edit", Suite: "controls", Filename: "k10_empty_edit_consistent.mp4",
		ExpectStatus: ExpectPass,
		Note:         "空 edit（延迟 500ms）+ 显示时长已计入（mvhd=1500）→ 三口径一致，不报",
		Build:        buildK10EmptyEditConsistent,
	},
	{
		ID: "K11-colr-hdr", Suite: "controls", Filename: "k11_colr_hdr_complete.mp4",
		ExpectStatus: ExpectPass,
		Note:         "单个 bt2020 colr + 完整 mdcv/clli 且量程合法（D2/hdr gray 的合法近邻）",
		Build:        buildK11ColrHdr,
	},
	{
		ID: "K12-matrix-ortho", Suite: "controls", Filename: "k12_matrix_ortho_identity.mp4",
		ExpectStatus: ExpectPass,
		Note:         "单位 matrix（行列点积精确 = 1）——正交容差下不报",
		Build:        buildK12MatrixIdentity,
	},
	{
		ID: "K13-opus", Suite: "controls", Filename: "k13_opus_priming_trimmed.mp4",
		ExpectStatus: ExpectPass,
		Note:         "Opus 轨：elst 裁剪 312 + dOps.preSkip=312 一致（已知编码集内，不误报）",
		Build:        buildK13OpusTrimmed,
	},
	{
		ID: "K14-co64", Suite: "controls", Filename: "k14_co64_valid.mp4",
		ExpectStatus: ExpectPass,
		Note:         "co64 指向 mdat 载荷内（64 位条目合法路径）",
		Build:        buildK14Co64Valid,
	},
	{
		ID: "K15-bframes", Suite: "controls", Filename: "k15_video_bframes.mp4",
		ExpectStatus: ExpectPass,
		Note:         "含 B 帧的 ctts v0（正偏移乱序）——PTS 乱序不得当 DTS 回退（M4 回归集的最小前哨）",
		Build:        buildK15BFrames,
	},
}

// ── controls 构造器 ─────────────────────────────────────────────────────

func buildK1SingleMoov(seed uint64) ([]byte, uint64, error) {
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{videoTrack(90000, 90000, 30)},
	})
	return raw, 0, nil
}

func buildK2StcoInMdat(seed uint64) ([]byte, uint64, error) {
	// 默认布局即是"stco 指向 mdat 载荷内"；此样本保持默认（同 K1 不同 seed 域不引入随机）
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{videoTrack(90000, 180000, 60)},
	})
	return raw, 0, nil
}

func buildK3MdatSizeZero(seed uint64) ([]byte, uint64, error) {
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks:       []*track{videoTrack(90000, 90000, 30)},
		sizeZeroMdat: true,
	})
	return raw, 0, nil
}

func buildK3bMdatLargesize(seed uint64) ([]byte, uint64, error) {
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks:        []*track{videoTrack(90000, 90000, 30)},
		largesizeMdat: true,
	})
	return raw, 0, nil
}

func buildK4DurationAligned(seed uint64) ([]byte, uint64, error) {
	// 30×3000 @90000 = 90000 tick = 1000ms；mvhd=1000、tkhd=1000、stts 和=90000/90000=1000ms
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 1000,
		tracks: []*track{videoTrack(90000, 90000, 30)},
	})
	return raw, 0, nil
}

func buildK5MonotonicDts(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30)
	vt.stts = []sttsEntry{{Count: 30, Duration: 3000}} // 单条目 → 不触发 vfr_detected
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, 0, nil
}

func buildK6PrimingTrimmed(seed uint64) ([]byte, uint64, error) {
	roll := int16(1024)
	at := audioTrack(48000, 48*1024, 1024)
	at.roll = &roll
	at.elst = []elstEntry{{SegmentDuration: 1024, MediaTime: 1024}} // 正确裁剪
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 1024,
		tracks: []*track{at},
	})
	return raw, 0, nil
}

func buildK7PortraitRot90(seed uint64) ([]byte, uint64, error) {
	m := rot90Matrix()
	vt := videoTrack(90000, 90000, 30)
	vt.matrix = &m
	// 存储 1080×1920（stsd），呈现声明 1920×1080（tkhd 宽 > 高）：
	// 旋转后的呈现宽高正确互换，且带 pasp 说明 → 不报
	vt.w, vt.h = 1920<<16, 1080<<16
	vt.stsdW, vt.stsdH = 1080<<16, 1920<<16
	vt.paspNum, vt.paspDen = 1, 1
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, 0, nil
}

func buildK8Rot180(seed uint64) ([]byte, uint64, error) {
	m := rot180Matrix()
	vt := videoTrack(90000, 90000, 30)
	vt.matrix = &m
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, 0, nil
}

func buildK9FragOrdered(seed uint64) ([]byte, uint64, error) {
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 2000,
		tracks: []*track{emptyFragTrack(90000, 0)},
		frags: []frag{
			{tfdt: 0, count: 30, dur: 3000, size: 16},
			{tfdt: 90000, count: 30, dur: 3000, size: 16},
		},
	})
	return raw, 0, nil
}

func buildK10EmptyEditConsistent(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30)
	vt.elst = []elstEntry{
		{SegmentDuration: 500, MediaTime: -1},
		{SegmentDuration: 1000, MediaTime: 0},
	}
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 1500, // 已计入空 edit 的 500ms
		tracks: []*track{vt},
	})
	return raw, 0, nil
}

func buildK11ColrHdr(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30)
	vt.colrs = [][]byte{colrPayload(9, 16, 9, false)} // bt2020
	vt.mdcv = mdcvBox(1000, 1, 500, 1)
	vt.clli = clliBox(1000, 400)
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, 0, nil
}

func buildK12MatrixIdentity(seed uint64) ([]byte, uint64, error) {
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{videoTrack(90000, 90000, 30)},
	})
	return raw, 0, nil
}

func buildK13OpusTrimmed(seed uint64) ([]byte, uint64, error) {
	at := &track{
		kind: "soun", timescale: 48000, duration: 48 * 960,
		stts:  []sttsEntry{{Count: 48, Duration: 960}}, // 48 样本 × 960 tick（20ms @48k）
		codec: "Opus", sampleSz: 8,
		opusPre: 312,
		elst:    []elstEntry{{SegmentDuration: 960, MediaTime: 312}},
	}
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 960,
		tracks: []*track{at},
	})
	return raw, 0, nil
}

func buildK14Co64Valid(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30)
	vt.co64 = true
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, 0, nil
}

func buildK15BFrames(seed uint64) ([]byte, uint64, error) {
	// B 帧形态：ctts v0 正偏移乱序（1,2,1,3,1,2…），DTS 由 stts 单调。
	vt := videoTrack(90000, 90000, 30)
	vt.ctts = []cttsEntry{
		{Count: 1, Offset: 3000}, {Count: 1, Offset: 6000}, {Count: 1, Offset: 3000},
		{Count: 1, Offset: 6000}, {Count: 26, Offset: 0},
	}
	raw, _ := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, 0, nil
}

// ── m3-regression：≥20 个正常 H.264/H.265（含 B 帧）─────────────────────

// m3RegressionBuilders 是 M3 回归集。
//
// 覆盖维度：编码（avc1/hvc1）× ctts 形态（无 / v0 正偏移 / v1 有符号含负）×
// timescale（90000/600/30000/24000/1000/15360 实测常见值）× 帧数（1/12/30/60/240）。
// **必须含 B 帧**：ctts 负偏移在 v1 里合法，v0 里靠"正偏移乱序"表达 —— 两者都在集里。
var m3RegressionBuilders = buildM3Regression()

// m3Spec 是回归集一行的构造参数（真值 = 这些参数，不由实现回算）。
type m3Spec struct {
	codec  string
	ts     uint32
	frames uint32
	ctts   []cttsEntry
	v1     bool
}

func (s m3Spec) kind() string {
	if len(s.ctts) == 0 {
		return "无 ctts（PTS=DTS 同序）"
	}
	if s.v1 {
		return "v1 有符号（含负偏移 = B 帧）"
	}
	return "v0 正偏移乱序（B 帧）"
}

func buildM3Regression() []Builder {
	specs := []m3Spec{
		{"avc1", 90000, 30, nil, false},
		{"avc1", 90000, 60, nil, false},
		{"avc1", 600, 30, nil, false},
		{"avc1", 30000, 30, nil, false},
		{"avc1", 24000, 24, nil, false},
		{"avc1", 1000, 30, nil, false},
		{"hvc1", 90000, 30, nil, false},
		{"hvc1", 90000, 60, nil, false},
		{"hvc1", 30000, 60, nil, false},
		{"hvc1", 24000, 48, nil, false},
		{"hvc1", 15360, 30, nil, false},
		{"hvc1", 600, 30, nil, false},
		// B 帧：v0 正偏移乱序（I P B B P…）
		{"avc1", 90000, 30, []cttsEntry{{1, 3000}, {1, 6000}, {1, 3000}, {1, 6000}, {26, 0}}, false},
		{"avc1", 90000, 60, []cttsEntry{{1, 3000}, {1, 6000}, {1, 3000}, {1, 6000}, {56, 0}}, false},
		{"avc1", 30000, 30, []cttsEntry{{1, 1000}, {1, 2000}, {1, 1000}, {1, 2000}, {26, 0}}, false},
		{"hvc1", 90000, 30, []cttsEntry{{1, 3000}, {1, 6000}, {1, 3000}, {1, 6000}, {26, 0}}, false},
		{"hvc1", 90000, 60, []cttsEntry{{1, 3000}, {1, 6000}, {1, 3000}, {1, 6000}, {56, 0}}, false},
		// B 帧：v1 有符号含**负偏移**（合法形态，误判风险最高）
		{"avc1", 90000, 30, []cttsEntry{{1, 3000}, {1, 6000}, {1, 3000}, {1, 6000}, {2, -6000}, {24, 0}}, true},
		{"avc1", 90000, 60, []cttsEntry{{1, 3000}, {1, 6000}, {1, 3000}, {1, 6000}, {2, -6000}, {54, 0}}, true},
		{"hvc1", 90000, 30, []cttsEntry{{1, 3000}, {1, 6000}, {1, 3000}, {1, 6000}, {2, -6000}, {24, 0}}, true},
		{"hvc1", 30000, 30, []cttsEntry{{1, 1000}, {1, 2000}, {1, 1000}, {1, 2000}, {2, -2000}, {24, 0}}, true},
		{"avc1", 24000, 24, []cttsEntry{{1, 1000}, {1, 2000}, {1, 1000}, {1, 2000}, {20, 0}}, false},
	}

	out := make([]Builder, 0, len(specs)+2)
	for i, s := range specs {
		s := s
		shape := "ctts-none"
		if len(s.ctts) > 0 {
			if s.v1 {
				shape = "ctts-v1-signed"
			} else {
				shape = "ctts-v0-bframes"
			}
		}
		out = append(out, Builder{
			ID:           fmt.Sprintf("M3R%02d", i+1),
			Suite:        "m3-regression",
			Filename:     fmt.Sprintf("m3r%02d_%s_%d.mp4", i+1, s.codec, s.ts),
			ExpectStatus: ExpectPass,
			BuildShape:   shape,
			Note: fmt.Sprintf("正常 %s 文件：timescale=%d、%d 帧、ctts=%d 条（%s）——DTS 单调、误报必须为 0",
				s.codec, s.ts, s.frames, len(s.ctts), s.kind()),
			Build: makeM3Builder(s),
		})
	}
	// 单帧最小样本与长时长样本（边界）
	out = append(out, Builder{
		ID: "M3R23", Suite: "m3-regression", Filename: "m3r23_single_frame.mp4",
		ExpectStatus: ExpectPass, Note: "单帧最小样本（样本数为 1 的边界）",
		Build: func(seed uint64) ([]byte, uint64, error) {
			r, _ := buildMovie(movie{mvhdTimescale: 1000, durationAuto: true, tracks: []*track{videoTrack(90000, 3000, 1)}})
			return r, 0, nil
		},
	})
	out = append(out, Builder{
		ID: "M3R24", Suite: "m3-regression", Filename: "m3r24_long_240_frames.mp4",
		ExpectStatus: ExpectPass, Note: "240 帧长时长（累计 DTS 大数）",
		Build: func(seed uint64) ([]byte, uint64, error) {
			r, _ := buildMovie(movie{mvhdTimescale: 1000, mvhdDuration: 8000, tracks: []*track{videoTrack(90000, 720000, 240)}})
			return r, 0, nil
		},
	})
	return out
}

func makeM3Builder(s m3Spec) func(uint64) ([]byte, uint64, error) {
	return func(seed uint64) ([]byte, uint64, error) {
		dur := uint32(30000)
		if s.ts == 600 || s.ts == 1000 {
			dur = 1000
		}
		vt := videoTrack(s.ts, dur, s.frames)
		vt.codec = s.codec
		vt.ctts = s.ctts
		vt.cttsV1 = s.v1
		r, _ := buildMovie(movie{mvhdTimescale: 1000, durationAuto: true, tracks: []*track{vt}})
		return r, 0, nil
	}
}

// ── m4：正反例各 5 ──────────────────────────────────────────────────────

var m4Builders = []Builder{
	// 正例（违法：未声明/不一致）
	{
		ID: "M4P1", Suite: "m4", Filename: "m4p1_undeclared_1024.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"priming_undeclared"},
		Note: "AAC PTS 后移 1024、无 elst → 报 1024", Build: buildC6Priming1024,
	},
	{
		ID: "M4P2", Suite: "m4", Filename: "m4p2_undeclared_2112.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"priming_undeclared"},
		Note: "AAC PTS 后移 2112（常见观测值亦见 2112）、无 elst → 报 2112",
		Build: func(seed uint64) ([]byte, uint64, error) {
			at := audioTrack(48000, 48*1024, 0)
			at.ctts = []cttsEntry{{Count: 48, Offset: 2112}}
			at.roll = nil
			r, mk := buildMovie(movie{mvhdTimescale: 1000, durationAuto: true, tracks: []*track{at}})
			return r, mk["trak0_stbl_ctts_entry0"], nil
		},
	},
	{
		ID: "M4P3", Suite: "m4", Filename: "m4p3_trim_wrong_amount.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"priming_mismatch"},
		Note: "elst 裁剪 512（少于声明的 1024）→ 声明 vs 期望不一致",
		Build: func(seed uint64) ([]byte, uint64, error) {
			roll := int16(1024)
			at := audioTrack(48000, 48*1024, 1024)
			at.roll = &roll
			at.elst = []elstEntry{{SegmentDuration: 1024, MediaTime: 512}}
			r, mk := buildMovie(movie{mvhdTimescale: 1000, mvhdDuration: 1024, tracks: []*track{at}})
			return r, mk["trak0_elst0_media_time"], nil
		},
	},
	{
		ID: "M4P4", Suite: "m4", Filename: "m4p4_start_trim_positive.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"start_trim"},
		Note: "elst[0].media_time=1024（起始裁剪）→ start_trim 报秒值 + 样本数",
		Build: func(seed uint64) ([]byte, uint64, error) {
			roll := int16(1024)
			at := audioTrack(48000, 48*1024, 1024)
			at.roll = &roll
			at.elst = []elstEntry{{SegmentDuration: 1024, MediaTime: 2048}} // 多裁 1024
			r, mk := buildMovie(movie{mvhdTimescale: 1000, mvhdDuration: 1024, tracks: []*track{at}})
			return r, mk["trak0_elst0_media_time"], nil
		},
	},
	{
		ID: "M4P5", Suite: "m4", Filename: "m4p5_roll_mismatch_704.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"priming_mismatch"},
		Note: "roll=704 vs 声明 1024（同 C6c，独立成组以凑 5 正例）", Build: buildC6cPrimingMismatch,
	},

	// 反例（合法：正确裁剪，不得报）
	{
		ID: "M4N1", Suite: "m4", Filename: "m4n1_trimmed_1024.mp4",
		ExpectStatus: ExpectPass, Note: "elst 裁剪 1024 + roll=1024 一致 → 不报（K6 同构）",
		Build: buildK6PrimingTrimmed,
	},
	{
		ID: "M4N2", Suite: "m4", Filename: "m4n2_trimmed_2112.mp4",
		ExpectStatus: ExpectPass, Note: "elst 裁剪 2112 + roll=2112 一致 → 不报",
		Build: func(seed uint64) ([]byte, uint64, error) {
			roll := int16(2112)
			at := audioTrack(48000, 48*1024, 2112)
			at.roll = &roll
			at.elst = []elstEntry{{SegmentDuration: 1024, MediaTime: 2112}}
			r, _ := buildMovie(movie{mvhdTimescale: 1000, mvhdDuration: 1024, tracks: []*track{at}})
			return r, 0, nil
		},
	},
	{
		ID: "M4N3", Suite: "m4", Filename: "m4n3_opus_312_trimmed.mp4",
		ExpectStatus: ExpectPass, Note: "Opus 312 已裁剪且 dOps.preSkip=312 一致 → 不报", Build: buildK13OpusTrimmed,
	},
	{
		ID: "M4N4", Suite: "m4", Filename: "m4n4_video_no_edts_start0.mp4",
		ExpectStatus: ExpectPass, Note: "视频轨无 edts 且起点为 0（多数正常视频形态）→ 不落 no_edts",
		Build: buildK12MatrixIdentity,
	},
	{
		ID: "M4N5", Suite: "m4", Filename: "m4n5_audio_no_roll_start0.mp4",
		ExpectStatus: ExpectPass, Note: "AAC 无 roll 但起点为 0（无延迟可判）→ 不报、不落 gray",
		Build: func(seed uint64) ([]byte, uint64, error) {
			at := audioTrack(48000, 48*1024, 0)
			at.roll = nil
			r, _ := buildMovie(movie{mvhdTimescale: 1000, durationAuto: true, tracks: []*track{at}})
			return r, 0, nil
		},
	},
}

// ── m5：正反例各 5 ──────────────────────────────────────────────────────

var m5Builders = []Builder{
	// 正例（违法）
	{ID: "M5P1", Suite: "m5", Filename: "m5p1_rot90_nopasp.mp4", ExpectStatus: ExpectFail,
		Expected: []string{"rotation_dimension_conflict"}, Note: "rot90 + 宽<高 + 无 pasp", Build: buildC7Rot90Conflict},
	{ID: "M5P2", Suite: "m5", Filename: "m5p2_rot270_nopasp.mp4", ExpectStatus: ExpectFail,
		Expected: []string{"rotation_dimension_conflict"}, Note: "rot270（-90）+ 宽<高 + 无 pasp", Build: buildC7bRot270Conflict},
	{ID: "M5P3", Suite: "m5", Filename: "m5p3_matrix_shear.mp4", ExpectStatus: ExpectFail,
		Expected: []string{"matrix_not_orthogonal"}, Note: "剪切 matrix（行点积 ≠ 1）", Build: buildD1MatrixShear},
	{ID: "M5P4", Suite: "m5", Filename: "m5p4_colr_multi.mp4", ExpectStatus: ExpectFail,
		Expected: []string{"colr_multi_instance"}, Note: "两个 colr（bt709+bt2020）", Build: buildD2ColrMulti},
	{ID: "M5P5", Suite: "m5", Filename: "m5p5_sar_dar_conflict.mp4", ExpectStatus: ExpectFail,
		Expected: []string{"sar_dar_conflict"},
		Note: "pasp 8:9 与 tkhd 声明宽高比（4:3）矛盾 → SAR/DAR 不一致",
		Build: func(seed uint64) ([]byte, uint64, error) {
			vt := videoTrack(90000, 90000, 30)
			vt.w, vt.h = 640<<16, 480<<16 // 呈现 4:3
			vt.stsdW, vt.stsdH = 720<<16, 480<<16
			vt.paspNum, vt.paspDen = 8, 9 // 说明像素非方形且方向与声明不符
			r, mk := buildMovie(movie{mvhdTimescale: 1000, durationAuto: true, tracks: []*track{vt}})
			return r, mk["trak0_stbl_stsd_pasp"], nil
		},
	},

	// 反例（合法）
	{ID: "M5N1", Suite: "m5", Filename: "m5n1_portrait_rot90_pasp.mp4", ExpectStatus: ExpectPass,
		Note: "正常竖屏（rot90 + 宽>高 + pasp）", Build: buildK7PortraitRot90},
	{ID: "M5N2", Suite: "m5", Filename: "m5n2_rot180_landscape.mp4", ExpectStatus: ExpectPass,
		Note: "rot180 横屏（不进入宽高互换规则）", Build: buildK8Rot180},
	{ID: "M5N3", Suite: "m5", Filename: "m5n3_identity_matrix.mp4", ExpectStatus: ExpectPass,
		Note: "单位 matrix 正交", Build: buildK12MatrixIdentity},
	{ID: "M5N4", Suite: "m5", Filename: "m5n4_colr_hdr_complete.mp4", ExpectStatus: ExpectPass,
		Note: "单 colr bt2020 + 完整 mdcv/clli 量程合法", Build: buildK11ColrHdr},
	{ID: "M5N5", Suite: "m5", Filename: "m5n5_pasp_square_consistent.mp4", ExpectStatus: ExpectPass,
		Note: "pasp 1:1（方形像素）与声明宽高比一致 → 不报 sar_dar_conflict",
		Build: func(seed uint64) ([]byte, uint64, error) {
			vt := videoTrack(90000, 90000, 30)
			vt.w, vt.h = 640<<16, 480<<16
			vt.stsdW, vt.stsdH = 640<<16, 480<<16
			vt.paspNum, vt.paspDen = 1, 1
			r, _ := buildMovie(movie{mvhdTimescale: 1000, durationAuto: true, tracks: []*track{vt}})
			return r, 0, nil
		},
	},
}

// ── m3-regression：≥20 个正常 H.264/H.265（含 B 帧）─────────────────────