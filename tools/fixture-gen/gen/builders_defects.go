// builders_defects.go 是缺陷套件（defects）的构造器集合。
//
// 覆盖 `data/README.md §4` 的 C1–C7 七类，并补齐同类变体，使合成缺陷集 ≥25 例。
// 所有样本由 movie.go 的同一份合法布局派生；每类期望的首证据偏移取自构造点 marks。
//
// 真值口径（与 docs/fixture-pools.md §六 一致）：
//   - stts 只可能"平推"（duration 无符号、累计单调不减），故"回退 DTS"在本文件集里
//     由 C5b（分片 tfdt 序倒退，字面判据 tfdt[k] < tfdt[k-1]）与 C5（zero_duration_entry）覆盖；
//   - priming 的合成约定：sgpd/sbgp roll_distance = +priming 样本数；
//     真实文件里 ffmpeg 写 roll_distance=-1 + elst.media_time=1024（2026-10-27 实测），
//     该符号语义**不在 M1 阶段下结论**，留给 M4/M5 用真实样本复核。
package gen

// defectBuilders 是缺陷样本清单（期望状态：fail / gray）。
var defectBuilders = []Builder{
	// ── C1 双 moov ──────────────────────────────────────────────────────
	{
		ID: "C1", Suite: "defects", Filename: "c1_dual_moov.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"duplicate_moov"},
		Note: "第二份 moov 追加在文件末尾（mdat 之后）；期望 duplicate_moov 报两处偏移，首证据为第 2 处",
		Build: buildC1DualMoov,
	},
	{
		ID: "C1b", Suite: "defects", Filename: "c1b_dual_moov_adjacent.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"duplicate_moov"},
		Note: "第二份 moov 紧贴第一份（mdat 之前）；防止实现只识别末尾追加这一种形态",
		Build: buildC1bDualMoovAdjacent,
	},

	// ── C2 stco 越界 ────────────────────────────────────────────────────
	{
		ID: "C2", Suite: "defects", Filename: "c2_stco_out_of_mdat.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"stco_out_of_mdat"},
		Note: "chunk offset 指向文件尾之外；证据在 stco 条目字节上（条目号 + 目标区间 vs mdat 区间）",
		Build: buildC2StcoOutOfMdat,
	},
	{
		ID: "C2b", Suite: "defects", Filename: "c2b_stco_points_at_moov.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"stco_out_of_mdat"},
		Note: "chunk offset 落在 moov 区间（文件内、非 mdat 载荷）——同样算越界（docs/rules.md §M1 反例说明）",
		Build: buildC2bStcoPointsAtMoov,
	},
	{
		ID: "C2c", Suite: "defects", Filename: "c2c_stco_tail_truncated.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"stco_out_of_mdat"},
		Note: "偏移起点在 mdat 内，但 offset + 样本长度越过 mdat 载荷尾部（被尾部截断形态）",
		Build: buildC2cStcoTailTruncated,
	},
	{
		ID: "C2d", Suite: "defects", Filename: "c2d_co64_huge_offset.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"stco_out_of_mdat"},
		Note: "co64 巨型偏移（2^40）——覆盖 64 位条目路径",
		Build: buildC2dCo64HugeOffset,
	},

	// ── C3 截断 mdat ────────────────────────────────────────────────────
	{
		ID: "C3", Suite: "defects", Filename: "c3_truncated_mdat.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"truncated_mdat"},
		Note: "mdat 声明 size 大于实际剩余字节；报声明 vs 实际（证据 = mdat size 字段）",
		Build: buildC3TruncatedMdat,
	},
	{
		ID: "C3b", Suite: "defects", Filename: "c3b_cut_tail.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"truncated_mdat"},
		Note: "文件尾被砍 64 字节（size 字段合法、可读字节不足）——与 C3 仅成因不同",
		Build: buildC3bCutTail,
	},

	// ── C4a 时长口径互斥（mvhd / tkhd 与实际不等）──────────────────────
	{
		ID: "C4a", Suite: "defects", Filename: "c4a_duration_2x_mvhd.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"duration_mismatch"},
		Note: "mvhd.duration = 2×实际（S7 类 2× 时长）；报三口径原值与差值（1 tick 容差之外）",
		Build: buildC4aDuration2x,
	},
	{
		ID: "C4a2", Suite: "defects", Filename: "c4a2_duration_2x_tkhd.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"duration_mismatch"},
		Note: "只有 tkhd.duration 是 2×（mvhd 正确）——证明实现不只看 mvhd",
		Build: buildC4a2Tkhd2x,
	},
	{
		ID: "C4a3", Suite: "defects", Filename: "c4a3_duration_short_mvhd.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"duration_mismatch"},
		Note: "mvhd.duration 比实际短 500 tick（小差值 > 1 tick 容差即 fail）",
		Build: buildC4a3ShortMvhd,
	},

	// ── C4b 起点 + 时长越过声明轨道时长 ─────────────────────────────────
	{
		ID: "C4b", Suite: "defects", Filename: "c4b_elst_overrun.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"duration_mismatch", "first_pts_outside_media"},
		Note: "elst 起点后移到 media_time=3000（首帧 PTS=0 不在区间内）+ Σelst(1500ms) > 声明轨道时长(1000ms)" +
			"（负剩余时长的无符号等价形态；**未写任何负数字段**）",
		Build: buildC4bElstOverrun,
	},

	// ── C5 回退 DTS 等价形态 ────────────────────────────────────────────
	{
		ID: "C5", Suite: "defects", Filename: "c5_zero_duration_entry.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"zero_duration_entry", "vfr_detected"},
		Note: "stts 第 2 条 duration=0（多个样本落在同一时刻）；按 data/README §5.4 报 fail + 条目号；" +
			"同轨 stts 条目数 3 → 依 §5.5 同时落 vfr_detected（gray，二者并列不互斥）",
		Build: buildC5ZeroDuration,
	},
	{
		ID: "C5b", Suite: "defects", Filename: "c5b_tfdt_regression.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"tfdt_regression"},
		Note: "分片 1 tfdt=90000、分片 2 tfdt=30000（字面判据 tfdt[k] < tfdt[k-1] 命中）；报分片序号",
		Build: buildC5bTfdtRegression,
	},
	{
		ID: "C5c", Suite: "defects", Filename: "c5c_tfdt_v1_regression.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"tfdt_regression"},
		Note: "tfdt version=1（64 位）同样回退——覆盖分宽读取路径",
		Build: buildC5cTfdtV1Regression,
	},

	// ── C6 1024 samples 延迟 ───────────────────────────────────────────
	{
		ID: "C6", Suite: "defects", Filename: "c6_priming_undeclared_1024.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"priming_undeclared"},
		Note: "音频轨首帧 PTS 后移 1024 样本（ctts 偏移 1024）且无 edts 裁剪；期望 M4 报出 1024",
		Build: buildC6Priming1024,
	},
	{
		ID: "C6b", Suite: "defects", Filename: "c6b_no_roll_distance.mp4",
		ExpectStatus: ExpectGray, Expected: []string{"no_roll_distance"}, GrayReason: "no_roll_distance",
		Note: "AAC 轨：编辑表裁剪 1024、但无 sgpd/sbgp roll → priming 期望值不可推，落 gray（不得猜 fail）；" +
			"与 M4N5（无裁剪、起点 0、无 roll 的合法形态）字节不同",
		Build: buildC6bNoRoll,
	},
	{
		ID: "C6c", Suite: "defects", Filename: "c6c_priming_mismatch.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"priming_mismatch"},
		Note: "已用 elst 裁剪 1024，但 roll 距离 = 1152 ≠ 裁剪量（声明 > 裁剪 = 少裁）——" +
			"roll > 裁剪 才叫\"声明不一致\"；裁剪 > roll 属起始裁剪过量（start_trim）",
		Build: buildC6cPrimingMismatch,
	},

	// ── C7 matrix=±90 无 pasp ──────────────────────────────────────────
	{
		ID: "C7", Suite: "defects", Filename: "c7_rot90_dim_conflict.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"rotation_dimension_conflict"},
		Note: "matrix = +90°（0x00010000 位形）、呈现宽 < 高（tkhd 1080×1920 位形）、无 pasp",
		Build: buildC7Rot90Conflict,
	},
	{
		ID: "C7b", Suite: "defects", Filename: "c7b_rot270_dim_conflict.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"rotation_dimension_conflict"},
		Note: "matrix = -90°（270°）同冲突——覆盖负向旋转",
		Build: buildC7bRot270Conflict,
	},

	// ── M1 结构：子盒越界 / 双 mdat ───────────────────────────────────
	{
		ID: "S1", Suite: "defects", Filename: "s1_child_overruns_parent.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"child_overruns_parent"},
		Note: "moov 尾部 udta 子盒声明长度大于父盒剩余——子盒区间 ⊄ 父盒载荷",
		Build: buildS1ChildOverruns,
	},
	{
		ID: "S2", Suite: "defects", Filename: "s2_duplicate_mdat.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"duplicate_mdat"},
		Note: "两个 mdat（第二个合法空盒）；按 plan/03 §二 duplicate_mdat = fail",
		Build: buildS2DuplicateMdat,
	},
	{
		ID: "S3", Suite: "defects", Filename: "s3_deep_nest_over_cap.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"max_depth_exceeded"},
		Note: "moov 下嵌 100 层 udta（超过 MaxDepth=64）；防御样本，必须不 panic，且深度上限可判",
		Build: buildS3DeepNest,
	},

	// ── M5 变体：非正交 matrix / 多 colr ───────────────────────────────
	{
		ID: "D1", Suite: "defects", Filename: "d1_matrix_shear.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"matrix_not_orthogonal"},
		Note: "matrix 含剪切项（行点积 ≠ 1 超容差）——整数正交性检查用；不得用浮点近似",
		Build: buildD1MatrixShear,
	},
	{
		ID: "D2", Suite: "defects", Filename: "d2_colr_multi_instance.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"colr_multi_instance"},
		Note: "两个 colr（bt709 → bt2020）+ 完整 mdcv/clli；多实例必须按文件内顺序全列，不取第一个",
		Build: buildD2ColrMulti,
	},

	// ── M4 变体：未知编码 priming → gray ───────────────────────────────
	{
		ID: "P1", Suite: "defects", Filename: "p1_unknown_codec_priming.mp4",
		ExpectStatus: ExpectGray, Expected: []string{"unknown_codec_priming"}, GrayReason: "unknown_codec_priming",
		Note: "ac-3 编码 + 无 roll + 起点 0；编码不在已知集（AAC/Opus）→ 期望值不可判，落 gray",
		Build: buildP1UnknownCodec,
	},

	// ── M2 变体：media_rate ≠ 1 / 空 edit / 段次序可疑 ──────────────────
	{
		ID: "E1", Suite: "defects", Filename: "e1_media_rate_not_one.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"elst_item_rate_not_one"},
		Note: "elst[1].media_rate = 2.0（0x00020000）；报条目号 1（按文件内顺序、不排序、不折算）",
		Build: buildE1RateNotOne,
	},
	{
		ID: "E2", Suite: "defects", Filename: "e2_empty_edit.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"duration_mismatch"},
		Note: "空 edit（media_time=-1）贡献 500ms 延迟但 mvhd 未计入（mvhd=1000ms）→ 三口径互斥；" +
			"empty_edit 是否单独 fail 由 M3 定义，本条不预判（docs/rules.md：空 edit 本身是信息）",
		Build: buildE2EmptyEdit,
	},
	{
		ID: "E3", Suite: "defects", Filename: "e3_elst_order_suspect.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"elst_item_order_suspect"},
		Note: "elst[0] media_time=45000、elst[1] media_time=0 → 首帧 PTS=0 落在第 1 段而非第 0 段（次序可疑）；" +
			"本条不预判 first_pts_outside_media 的区间口径（首段 vs 并集）",
		Build: buildE3OrderSuspect,
	},

	// ── M3 变体：极小 timescale → gray ─────────────────────────────────
	{
		ID: "T1", Suite: "defects", Filename: "t1_timescale_suspect.mp4",
		ExpectStatus: ExpectGray, Expected: []string{"timescale_suspect"}, GrayReason: "timescale_suspect",
		Note: "视频轨 timescale = 30（< 100 阈值，且未溢出）→ gray timescale_suspect（不是 fail）",
		Build: buildT1TimescaleSuspect,
	},

	// ── M3 变体：32 位累计溢出前兆 ────────────────────────────────────
	{
		ID: "T2", Suite: "defects", Filename: "t2_dts_overflow_risk.mp4",
		ExpectStatus: ExpectFail, Expected: []string{"dts_overflow_risk"},
		Note: "stts 累计越过 32 位（2×0xFFFFFF00 ≈ 2^32）→ fail 并报越界位置",
		Build: buildT2OverflowRisk,
	},

	// ── 分片：mvhd.duration=0 且时长推不出 → duration_unknown ──────────
	{
		ID: "F1", Suite: "defects", Filename: "f1_duration_unknown.mp4",
		ExpectStatus: ExpectGray, Expected: []string{"duration_unknown"}, GrayReason: "duration_unknown",
		Note: "分片 + mvhd.duration=0 + trun 未写 sample_duration + 无 mehd → 三口径缺一方，落 gray",
		Build: buildF1DurationUnknown,
	},
}

// ── C1 ──────────────────────────────────────────────────────────────────

func buildC1DualMoov(seed uint64) ([]byte, uint64, error) {
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{videoTrack(90000, 90000, 30)},
	})
	moovStart := mk["moov0_start"]
	moov := raw[moovStart:mk["mdat0_start"]]
	dupStart := uint64(len(raw))
	return concat(raw, moov), dupStart, nil
}

func buildC1bDualMoovAdjacent(seed uint64) ([]byte, uint64, error) {
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{videoTrack(90000, 90000, 30)},
	})
	moovStart := mk["moov0_start"]
	moov := raw[moovStart:mk["mdat0_start"]]
	insert := mk["mdat0_start"]
	dupStart := insert
	return concat(raw[:insert], moov, raw[insert:]), dupStart, nil
}

// ── C2 ──────────────────────────────────────────────────────────────────

func buildC2StcoOutOfMdat(seed uint64) ([]byte, uint64, error) {
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{videoTrack(90000, 90000, 30)},
	})
	off := mk["trak0_stbl_stco0_entry0"]
	putU32(raw, off, uint32(len(raw)+4096)) // 远超文件尾
	return raw, off, nil
}

func buildC2bStcoPointsAtMoov(seed uint64) ([]byte, uint64, error) {
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{videoTrack(90000, 90000, 30)},
	})
	off := mk["trak0_stbl_stco0_entry0"]
	putU32(raw, off, uint32(mk["moov0_start"])+32) // 落在 moov 内
	return raw, off, nil
}

func buildC2cStcoTailTruncated(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30) // 30 样本 × 16 字节 = 480 字节载荷
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	off := mk["trak0_stbl_stco0_entry0"]
	// 起点仍在 mdat 内，但 offset + 480 越过 mdat 尾：把起点设为 mdat 载荷末 - 100
	mdatEnd := uint64(len(raw))
	putU32(raw, off, uint32(mdatEnd-100))
	return raw, off, nil
}

func buildC2dCo64HugeOffset(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30)
	vt.co64 = true
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	off := mk["trak0_stbl_stco0_entry0"]
	// co64：占位回填时写入的是合法值，这里改写为 2^40
	putU64(raw, off, 1<<40)
	return raw, off, nil
}

// ── C3 ──────────────────────────────────────────────────────────────────

func buildC3TruncatedMdat(seed uint64) ([]byte, uint64, error) {
	decl := uint32(4096)
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks:      []*track{videoTrack(90000, 90000, 30)},
		mdatDeclare: &decl,
	})
	return raw, mk["mdat0_start"], nil
}

func buildC3bCutTail(seed uint64) ([]byte, uint64, error) {
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks:       []*track{videoTrack(90000, 90000, 30)},
		tailCut:      64,
	})
	return raw, mk["mdat0_start"], nil
}

// ── C4 ──────────────────────────────────────────────────────────────────

func buildC4aDuration2x(seed uint64) ([]byte, uint64, error) {
	// 视频 30×3000 tick @90000 = 90000 tick = 1000ms；mvhd 写 2000（2×，S7 形态）
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 2000,
		tracks: []*track{videoTrack(90000, 90000, 30)},
	})
	return raw, mk["mvhd_duration"], nil
}

func buildC4a2Tkhd2x(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30)
	vt.tkhdDur = 2000 // tkhd 单独 2×
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 1000,
		tracks: []*track{vt},
	})
	return raw, mk["trak0_tkhd_duration"], nil
}

func buildC4a3ShortMvhd(seed uint64) ([]byte, uint64, error) {
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 500, // 实际 1000ms
		tracks: []*track{videoTrack(90000, 90000, 30)},
	})
	return raw, mk["mvhd_duration"], nil
}

func buildC4bElstOverrun(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30)
	vt.elst = []elstEntry{{SegmentDuration: 1500, MediaTime: 3000}}
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 1000,
		tracks: []*track{vt},
	})
	return raw, mk["trak0_elst0_media_time"], nil
}

// ── C5 ──────────────────────────────────────────────────────────────────

func buildC5ZeroDuration(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 150000, 30)
	vt.stts = []sttsEntry{{Count: 10, Duration: 3000}, {Count: 5, Duration: 0}, {Count: 15, Duration: 3000}}
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, mk["trak0_stbl_stts_entry1_duration"], nil
}

func buildC5bTfdtRegression(seed uint64) ([]byte, uint64, error) {
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 0,
		tracks: []*track{emptyFragTrack(90000, 0)},
		frags: []frag{
			{tfdt: 90000, count: 30, dur: 3000, size: 16},
			{tfdt: 30000, count: 30, dur: 3000, size: 16}, // 字面回退：tfdt[1] < tfdt[0]
		},
	})
	return raw, mk["moof1_tfdt"], nil
}

func buildC5cTfdtV1Regression(seed uint64) ([]byte, uint64, error) {
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 0,
		tracks: []*track{emptyFragTrack(90000, 0)},
		frags: []frag{
			{tfdt: 90000, tfdtV1: true, count: 30, dur: 3000, size: 16},
			{tfdt: 30000, tfdtV1: true, count: 30, dur: 3000, size: 16},
		},
	})
	return raw, mk["moof1_tfdt"], nil
}

// ── C6 ──────────────────────────────────────────────────────────────────

func buildC6Priming1024(seed uint64) ([]byte, uint64, error) {
	at := audioTrack(48000, 48*1024, 1024*1024) // 48 样本 × 1024 tick = 1024ms
	at.ctts = []cttsEntry{{Count: 48, Offset: 1024}} // 首帧 PTS 后移 1024 样本
	at.roll = nil
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{at},
	})
	return raw, mk["trak0_stbl_ctts_entry0_offset"], nil
}

func buildC6bNoRoll(seed uint64) ([]byte, uint64, error) {
	// 编辑表声明裁剪 1024，但容器**不**给 roll 距离 → 无法交叉核对 ⇒ gray `no_roll_distance`。
	// （M4N5 是"无裁剪、起点 0、无 roll"的合法形态，二者必须有不同字节，故此处带裁剪。）
	at := audioTrack(48000, 48*1024, 1024)
	at.roll = nil // 无 roll 距离 → 期望值不可推
	at.elst = []elstEntry{{SegmentDuration: 1024, MediaTime: 1024}}
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 1024, tracks: []*track{at},
	})
	return raw, mk["trak0_elst0_media_time"], nil
}

func buildC6cPrimingMismatch(seed uint64) ([]byte, uint64, error) {
	// 容器声明 roll = 1152，但编辑表只裁剪 1024：两处声明对不上（roll > 裁剪 = 少裁）。
	// roll > 裁剪 才是"声明不一致"；裁剪 > roll 属起始裁剪过量（start_trim，见 M4P4）。
	roll := int16(1152) // ≠ 编辑表裁剪的 1024
	at := audioTrack(48000, 48*1024, 1024)
	at.roll = &roll
	at.elst = []elstEntry{{SegmentDuration: 1024, MediaTime: 1024}} // 已裁剪 1024
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 1024,
		tracks: []*track{at},
	})
	return raw, mk["trak0_elst0_media_time"], nil
}

// ── C7 ──────────────────────────────────────────────────────────────────

func buildC7Rot90Conflict(seed uint64) ([]byte, uint64, error) {
	m := rot90Matrix()
	vt := videoTrack(90000, 90000, 30)
	vt.matrix = &m
	vt.w, vt.h = 1080<<16, 1920<<16 // 呈现宽 < 高（竖屏位形），无 pasp → 冲突
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, mk["trak0_tkhd_matrix"], nil
}

func buildC7bRot270Conflict(seed uint64) ([]byte, uint64, error) {
	m := rot270Matrix()
	vt := videoTrack(90000, 90000, 30)
	vt.matrix = &m
	vt.w, vt.h = 1080<<16, 1920<<16
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, mk["trak0_tkhd_matrix"], nil
}

// ── M1 结构边界 ─────────────────────────────────────────────────────────

func buildS1ChildOverruns(seed uint64) ([]byte, uint64, error) {
	// moov 尾部追加"声明长度大于父盒剩余"的子盒：把 udta 声明 size 放大到超出 moov。
	// 解析层会读出一个 child 结束点越过 moov 载荷（child_overruns_parent）。
	trailer := box("udta", make([]byte, 16))
	putU32(trailer, 0, uint32(len(trailer)+4096)) // 声明远大于实际
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks:      []*track{videoTrack(90000, 90000, 30)},
		moovTrailer: trailer,
	})
	// 该 trailer 在 moov 内、紧邻 moov 载荷尾部：偏移 = mdat0 起点 - trailer 全长
	return raw, mk["mdat0_start"] - uint64(len(trailer)), nil
}

func buildS2DuplicateMdat(seed uint64) ([]byte, uint64, error) {
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks:   []*track{videoTrack(90000, 90000, 30)},
		dualMdat: true,
	})
	return raw, mk["mdat1_start"], nil
}

func buildS3DeepNest(seed uint64) ([]byte, uint64, error) {
	// moov 下嵌 100 层 udta：解析层在 MaxDepth=64 处停止展开，
	// 最后一层可见节点仍有载荷 → 可判 max_depth_exceeded（防御判据，不得 panic）。
	depth := 100
	inner := box("udta", make([]byte, 8)) // 最内层 8 字节载荷
	for i := 1; i < depth; i++ {
		inner = box("udta", inner)
	}
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks:      []*track{videoTrack(90000, 90000, 30)},
		moovTrailer: inner,
	})
	return raw, mk["mdat0_start"] - uint64(len(inner)), nil
}

// ── M5 变体 ─────────────────────────────────────────────────────────────

func buildD1MatrixShear(seed uint64) ([]byte, uint64, error) {
	m := shearMatrix()
	vt := videoTrack(90000, 90000, 30)
	vt.matrix = &m
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, mk["trak0_tkhd_matrix"], nil
}

func buildD2ColrMulti(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30)
	vt.colrs = [][]byte{
		colrPayload(1, 1, 1, false),  // bt709
		colrPayload(9, 16, 9, false), // bt2020（广色域）
	}
	vt.mdcv = mdcvBox(1000, 1, 500, 1)
	vt.clli = clliBox(1000, 400)
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, mk["trak0_stbl_stsd_colr"], nil
}

// ── M4 变体 ─────────────────────────────────────────────────────────────

func buildP1UnknownCodec(seed uint64) ([]byte, uint64, error) {
	at := audioTrack(48000, 48*1024, 0)
	at.codec = "ac-3"
	at.roll = nil
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{at},
	})
	return raw, mk["trak0_stbl_stsd_dac3"], nil
}

// ── M2 变体 ─────────────────────────────────────────────────────────────

func buildE1RateNotOne(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30)
	vt.elst = []elstEntry{
		{SegmentDuration: 400, MediaTime: 0},
		{SegmentDuration: 600, MediaTime: 45000, MediaRate: 0x00020000}, // 2.0
	}
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 1000,
		tracks: []*track{vt},
	})
	return raw, mk["trak0_elst0_media_rate"] + 12, nil // 第 2 条目的 media_rate
}

func buildE2EmptyEdit(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30)
	vt.elst = []elstEntry{
		{SegmentDuration: 500, MediaTime: -1}, // 空 edit：贡献 500ms 延迟
		{SegmentDuration: 1000, MediaTime: 0},
	}
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 1000, // 未计入空 edit 的 500ms → 互斥
		tracks: []*track{vt},
	})
	return raw, mk["trak0_elst0_media_time"], nil
}

func buildE3OrderSuspect(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 90000, 30)
	vt.elst = []elstEntry{
		{SegmentDuration: 500, MediaTime: 45000},
		{SegmentDuration: 500, MediaTime: 0},
	}
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 1000,
		tracks: []*track{vt},
	})
	return raw, mk["trak0_elst0_media_time"], nil
}

// ── M3 变体 ─────────────────────────────────────────────────────────────

func buildT1TimescaleSuspect(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(30, 30, 30) // timescale=30 < 100 阈值
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, mk["trak0_mdhd_timescale"], nil
}

func buildT2OverflowRisk(seed uint64) ([]byte, uint64, error) {
	vt := videoTrack(90000, 2000000000, 30)
	vt.stts = []sttsEntry{{Count: 2, Duration: 0xFFFFFF00}} // 2× ≈ 2^32
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, durationAuto: true,
		tracks: []*track{vt},
	})
	return raw, mk["trak0_stbl_stts_entry0_duration"], nil
}

// ── 分片：duration_unknown ──────────────────────────────────────────────

func buildF1DurationUnknown(seed uint64) ([]byte, uint64, error) {
	raw, mk := buildMovie(movie{
		mvhdTimescale: 1000, mvhdDuration: 0,
		tracks: []*track{emptyFragTrack(90000, 0)},
		frags: []frag{
			{tfdt: 0, count: 30, size: 16, noDur: true},
			{tfdt: 90000, count: 30, size: 16, noDur: true},
		},
	})
	return raw, mk["moov0_start"], nil
}

// ── 轨模板 ──────────────────────────────────────────────────────────────

// videoTrack 是默认视频轨（avc1，单 stts 条目，stco 单 chunk）。
func videoTrack(timescale, duration uint32, frames uint32) *track {
	if frames == 0 {
		frames = 1
	}
	per := uint32(0)
	if duration > 0 {
		per = duration / frames
	}
	return &track{
		kind: "vide", timescale: timescale, duration: duration,
		stts:  []sttsEntry{{Count: frames, Duration: per}},
		w:     640 << 16, h: 480 << 16,
		codec: "avc1", sampleSz: 16,
	}
}

// audioTrack 是默认音频轨（mp4a）；shift 是"rolling distance"（合成约定：正数 = priming 样本数）。
//
// duration 是媒体时基下的总时长；样本数按 1024 tick/样本铺满。
func audioTrack(timescale, duration, shift uint32) *track {
	samples := duration / 1024
	if samples == 0 {
		samples = 1
	}
	roll := int16(shift)
	at := &track{
		kind: "soun", timescale: timescale, duration: duration,
		stts:  []sttsEntry{{Count: samples, Duration: 1024}},
		codec: "mp4a", sampleSz: 8,
	}
	if shift != 0 {
		at.roll = &roll
	}
	return at
}

// emptyFragTrack 是分片形态的轨道（样本表全空，数据在 moof 里）。
func emptyFragTrack(timescale, duration uint32) *track {
	return &track{
		kind: "vide", timescale: timescale, duration: duration,
		w: 640 << 16, h: 480 << 16,
		codec: "avc1", sampleSz: 0,
		emptyStbl: true,
	}
}

// putU32 就地改写 4 字节（构造器用）。
func putU32(b []byte, off uint64, v uint32) {
	b[off] = byte(v >> 24)
	b[off+1] = byte(v >> 16)
	b[off+2] = byte(v >> 8)
	b[off+3] = byte(v)
}

// putU64 就地改写 8 字节。
func putU64(b []byte, off uint64, v uint64) {
	putU32(b, off, uint32(v>>32))
	putU32(b, off+4, uint32(v))
}