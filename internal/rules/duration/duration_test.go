package duration

import (
	"encoding/binary"
	"testing"

	"github.com/yuluo554/mediatrace/internal/isobmff"
	"github.com/yuluo554/mediatrace/internal/policy"
	"github.com/yuluo554/mediatrace/internal/report"
)

// ── 本地字节构造（不复用生成器：K8 生成器与解析器不共享代码）──────────────

func bx(typ string, payload []byte) []byte {
	b := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(b[0:4], uint32(8+len(payload)))
	copy(b[4:8], typ)
	copy(b[8:], payload)
	return b
}

func fb(typ string, version byte, flags uint32, payload []byte) []byte {
	head := make([]byte, 4)
	binary.BigEndian.PutUint32(head, uint32(version)<<24|(flags&0x00FFFFFF))
	return bx(typ, cat(head, payload))
}

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func u32v(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

func i32v(v int32) []byte { return u32v(uint32(v)) }

// mvhdBox / tkhdBox / mdhdBox 只写本模块会读的字段（v0 形态）。
func mvhdBox(ts, dur uint32) []byte {
	return fb("mvhd", 0, 0, cat(u32v(0), u32v(0), u32v(ts), u32v(dur), make([]byte, 80)))
}

func tkhdBox(dur uint32) []byte {
	return fb("tkhd", 0, 3, cat(u32v(0), u32v(0), u32v(1), u32v(0), u32v(dur), make([]byte, 60)))
}

func mdhdBox(ts, dur uint32) []byte {
	return fb("mdhd", 0, 0, cat(u32v(0), u32v(0), u32v(ts), u32v(dur), u32v(0)))
}

func sttsBox(entries [][2]uint32) []byte {
	b := u32v(uint32(len(entries)))
	for _, e := range entries {
		b = append(b, cat(u32v(e[0]), u32v(e[1]))...)
	}
	return fb("stts", 0, 0, b)
}

func cttsBox(offsets []int32, v1 bool) []byte {
	ver := byte(0)
	if v1 {
		ver = 1
	}
	b := u32v(uint32(len(offsets)))
	for _, o := range offsets {
		b = append(b, cat(u32v(1), i32v(o))...)
	}
	return fb("ctts", ver, 0, b)
}

type elstSpec struct {
	segDur    uint32
	mediaTime int32
	rate      uint32 // 0 = 1.0
}

func elstBox(items []elstSpec) []byte {
	b := u32v(uint32(len(items)))
	for _, e := range items {
		rate := e.rate
		if rate == 0 {
			rate = rateOne
		}
		b = append(b, cat(u32v(e.segDur), i32v(e.mediaTime), u32v(rate))...)
	}
	return fb("elst", 0, 0, b)
}

// layout 描述一个最小单轨影片，并给出构造点已知的字段偏移。
type layout struct {
	data                        []byte
	mvhdDurOff, tkhdDurOff      uint64
	elstEntryOff                uint64 // 第 0 条 elst 条目起始
}

// buildMovie 装配 `moov{mvhd, trak{tkhd, [edts{elst}], mdia{mdhd, minf{stbl{stts,[ctts]}}}}}`。
func buildMovie(movieTs, mvhdDur, mediaTs, mediaDur, tkhdDur uint32, stts [][2]uint32, items []elstSpec, ctts []int32) layout {
	mvhd := mvhdBox(movieTs, mvhdDur)
	tkhd := tkhdBox(tkhdDur)
	mdhd := mdhdBox(mediaTs, mediaDur)
	sttsB := sttsBox(stts)
	stblPayload := sttsB
	if ctts != nil {
		stblPayload = cat(sttsB, cttsBox(ctts, false))
	}
	stbl := bx("stbl", stblPayload)
	mdia := bx("mdia", cat(mdhd, bx("minf", stbl)))
	trakPayload := cat(tkhd)
	var elstEntry uint64
	if items != nil {
		elst := elstBox(items)
		trakPayload = cat(trakPayload, bx("edts", elst))
		// moov 头 8 + mvhd + trak 头 8 + tkhd + edts 头 8 + elst 头 8 + v/f 4 + count 4
		elstEntry = uint64(8+len(mvhd)+8+len(tkhd)+8+8+4+4)
	}
	trakPayload = cat(trakPayload, mdia)
	moov := bx("moov", cat(mvhd, bx("trak", trakPayload)))
	return layout{
		data:         moov,
		mvhdDurOff:   8 + 8 + 16,
		tkhdDurOff:   uint64(8+len(mvhd)) + 8 + 8 + 20,
		elstEntryOff: elstEntry,
	}
}

func auditBytes(t *testing.T, data []byte) []report.Finding {
	t.Helper()
	root, err := isobmff.Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fs, err := Audit(root, data, "sample.mp4")
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	for _, f := range fs {
		if err := f.Validate(); err != nil {
			t.Fatalf("finding %s/%s invalid: %v", f.Module, f.RuleID, err)
		}
	}
	return fs
}

func rulesOf(fs []report.Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.RuleID]++
	}
	return m
}

func valueOf(t *testing.T, f report.Finding, name string) int64 {
	t.Helper()
	for _, v := range f.Values {
		if v.Name == name {
			return v.Value
		}
	}
	t.Fatalf("finding %s has no value %q (%+v)", f.RuleID, name, f.Values)
	return 0
}

// ── duration_mismatch ──────────────────────────────────────────────────

func TestDurationMismatchFiresOnMvhdDouble(t *testing.T) {
	// mvhd.duration = 2×实际（S7 形态）。
	l := buildMovie(1000, 2000, 90000, 90000, 1000, [][2]uint32{{30, 3000}}, nil, nil)
	fs := auditBytes(t, l.data)
	if len(fs) != 1 || fs[0].RuleID != "duration_mismatch" {
		t.Fatalf("findings = %+v, want one duration_mismatch", fs)
	}
	f := fs[0]
	if f.ByteOffset != l.mvhdDurOff || f.FieldPath != "moov/mvhd.duration" {
		t.Fatalf("evidence points at %s @%d, want moov/mvhd.duration @%d", f.FieldPath, f.ByteOffset, l.mvhdDurOff)
	}
	// 三口径原值必须齐（对齐到影片时基 tick）。
	if got := valueOf(t, f, "mvhd_duration"); got != 2000 {
		t.Fatalf("mvhd_duration = %d, want 2000", got)
	}
	if got := valueOf(t, f, "mdhd_duration"); got != 1000 {
		t.Fatalf("mdhd_duration = %d, want 1000", got)
	}
	if got := valueOf(t, f, "stts_sum"); got != 1000 {
		t.Fatalf("stts_sum = %d, want 1000", got)
	}
	if f.Delta == nil || f.Delta.Value != 1000 || f.Delta.Unit != "tick" {
		t.Fatalf("delta = %+v, want 1000 tick", f.Delta)
	}
	if f.Status != report.Fail {
		t.Fatalf("status = %s, want fail", f.Status)
	}
}

func TestDurationMismatchFiresOnTkhdOnly(t *testing.T) {
	// 只有 tkhd.duration 是 2×（mvhd 正确）——证明实现不只看 mvhd。
	l := buildMovie(1000, 1000, 90000, 90000, 2000, [][2]uint32{{30, 3000}}, nil, nil)
	fs := auditBytes(t, l.data)
	if len(fs) != 1 || fs[0].RuleID != "duration_mismatch" {
		t.Fatalf("findings = %+v, want one duration_mismatch", fs)
	}
	if fs[0].ByteOffset != l.tkhdDurOff || fs[0].FieldPath != "moov/trak/tkhd.duration" {
		t.Fatalf("evidence = %s @%d, want tkhd.duration @%d", fs[0].FieldPath, fs[0].ByteOffset, l.tkhdDurOff)
	}
}

// 容差是 policy.TickTolerance（1 tick）：差 1 tick 不报，差 2 tick 报。
func TestDurationToleranceIsTickTolerance(t *testing.T) {
	if policy.TickTolerance != 1 {
		t.Fatalf("TickTolerance = %d; this test encodes the 1-tick caliber", policy.TickTolerance)
	}
	for _, dur := range []uint32{1000, 1001} {
		l := buildMovie(1000, dur, 90000, 90000, 1000, [][2]uint32{{30, 3000}}, nil, nil)
		if fs := auditBytes(t, l.data); len(fs) != 0 {
			t.Fatalf("mvhd.duration=%d (diff=%d) produced %+v, want none", dur, dur-1000, fs)
		}
	}
	l := buildMovie(1000, 1002, 90000, 90000, 1000, [][2]uint32{{30, 3000}}, nil, nil)
	if got := rulesOf(auditBytes(t, l.data)); got["duration_mismatch"] != 1 {
		t.Fatalf("mvhd.duration=1002 (diff=2 > tolerance) rules = %v, want duration_mismatch", got)
	}
}

// 比较时基是影片时基（mvhd.timescale）：媒体时基更细时，影片侧的取整不得被判成超容差。
// 夹具 M3R23 同构：tkhd=33@1000、mdhd=3000@90000（差 30 个媒体 tick = 0.33 个影片 tick）。
func TestDurationComparesInMovieTimescale(t *testing.T) {
	l := buildMovie(1000, 33, 90000, 3000, 33, [][2]uint32{{1, 3000}}, nil, nil)
	if fs := auditBytes(t, l.data); len(fs) != 0 {
		t.Fatalf("coarse movie timescale rounding produced %+v, want none", fs)
	}
	// 反证：把 mvhd 写到真的超容差，必须报。
	l = buildMovie(1000, 2500, 90000, 3000, 33, [][2]uint32{{1, 3000}}, nil, nil)
	if got := rulesOf(auditBytes(t, l.data)); got["duration_mismatch"] != 1 {
		t.Fatalf("rules = %v, want duration_mismatch", got)
	}
}

// elst 的 Σ（放映时长）与 mvhd 对不上 → duration_mismatch，证据指向编辑表。
func TestDurationMismatchOnElstOverrun(t *testing.T) {
	l := buildMovie(1000, 1000, 90000, 90000, 1000, [][2]uint32{{30, 3000}},
		[]elstSpec{{segDur: 1500, mediaTime: 3000}}, nil)
	fs := auditBytes(t, l.data)
	if len(fs) != 2 {
		t.Fatalf("findings = %+v, want duration_mismatch + first_pts_outside_media", fs)
	}
	if got := rulesOf(fs); got["duration_mismatch"] != 1 || got["first_pts_outside_media"] != 1 {
		t.Fatalf("rules = %v", got)
	}
}

// ── elst_item_rate_not_one（条目号按文件内顺序、不排序）──────────────────

func TestElstRateNotOneReportsFileOrderItem(t *testing.T) {
	// elst[1].media_rate = 2.0；不得因"排序找错段"而报成条目 0。
	l := buildMovie(1000, 1000, 90000, 90000, 1000, [][2]uint32{{30, 3000}},
		[]elstSpec{{segDur: 400, mediaTime: 0}, {segDur: 600, mediaTime: 45000, rate: 0x00020000}}, nil)
	fs := auditBytes(t, l.data)
	if len(fs) != 1 || fs[0].RuleID != "elst_item_rate_not_one" {
		t.Fatalf("findings = %+v, want one elst_item_rate_not_one", fs)
	}
	f := fs[0]
	if got := valueOf(t, f, "elst_item"); got != 1 {
		t.Fatalf("elst_item = %d, want 1 (file order, not sorted)", got)
	}
	// trak 只有同名兄弟时才带索引（IR 的 Path 口径，F14）→ 单轨文件是 `moov/trak`。
	if want := "moov/trak/edts/elst[1].media_rate"; f.FieldPath != want {
		t.Fatalf("field_path = %q, want %q", f.FieldPath, want)
	}
	if want := l.elstEntryOff + 12 + 8; f.ByteOffset != want {
		t.Fatalf("byte_offset = %d, want %d (item #1 media_rate)", f.ByteOffset, want)
	}
	if f.Delta == nil || f.Delta.Value != 65536 || f.Delta.Unit != "1/65536" {
		t.Fatalf("delta = %+v, want 65536 1/65536", f.Delta)
	}
}

// ── 空 edit 语义 ───────────────────────────────────────────────────────

// 空 edit 只贡献放映延迟：mvhd 已计入 → 不报（K10 同构）。
func TestLeadingEmptyEditAccountedIsSilent(t *testing.T) {
	l := buildMovie(1000, 1500, 90000, 90000, 1000, [][2]uint32{{30, 3000}},
		[]elstSpec{{segDur: 500, mediaTime: -1}, {segDur: 1000, mediaTime: 0}}, nil)
	if fs := auditBytes(t, l.data); len(fs) != 0 {
		t.Fatalf("consistent empty edit produced %+v, want none", fs)
	}
}

// 空 edit 的延迟未被 mvhd 计入 → 三口径互斥，报一条 duration_mismatch（同一根因不连环报）。
func TestUnaccountedEmptyEditIsDurationMismatchOnly(t *testing.T) {
	l := buildMovie(1000, 1000, 90000, 90000, 1000, [][2]uint32{{30, 3000}},
		[]elstSpec{{segDur: 500, mediaTime: -1}, {segDur: 1000, mediaTime: 0}}, nil)
	fs := auditBytes(t, l.data)
	if got := rulesOf(fs); len(fs) != 1 || got["duration_mismatch"] != 1 {
		t.Fatalf("findings = %+v, want exactly one duration_mismatch", fs)
	}
}

// 空 edit 出现在媒体段之后 = 自相矛盾 → empty_edit。
func TestNonLeadingEmptyEditFires(t *testing.T) {
	l := buildMovie(1000, 1500, 90000, 90000, 1000, [][2]uint32{{30, 3000}},
		[]elstSpec{{segDur: 1000, mediaTime: 0}, {segDur: 500, mediaTime: -1}}, nil)
	fs := auditBytes(t, l.data)
	if got := rulesOf(fs); got["empty_edit"] != 1 {
		t.Fatalf("rules = %v, want empty_edit", got)
	}
	for _, f := range fs {
		if f.RuleID != "empty_edit" {
			continue
		}
		if got := valueOf(t, f, "segment_duration"); got != 500 {
			t.Fatalf("segment_duration = %d, want 500", got)
		}
		if got := valueOf(t, f, "start_offset_movie_ticks"); got != 1000 {
			t.Fatalf("start_offset_movie_ticks = %d, want 1000", got)
		}
	}
}

// ── 段次序 vs 时长错（分开报）──────────────────────────────────────────

func TestElstOrderSuspectIsSeparateFromDurationMismatch(t *testing.T) {
	l := buildMovie(1000, 1000, 90000, 90000, 1000, [][2]uint32{{30, 3000}},
		[]elstSpec{{segDur: 500, mediaTime: 45000}, {segDur: 500, mediaTime: 0}}, nil)
	fs := auditBytes(t, l.data)
	if got := rulesOf(fs); len(fs) != 1 || got["elst_item_order_suspect"] != 1 {
		t.Fatalf("findings = %+v, want one elst_item_order_suspect (no duration_mismatch)", fs)
	}
}

// 合法起点裁剪（跨度一致）不得被判成 first_pts_outside_media —— 那是 M4 的 start_trim。
func TestConsistentStartTrimIsNotFirstPtsOutside(t *testing.T) {
	l := buildMovie(1000, 1024, 48000, 49152, 1024, [][2]uint32{{48, 1024}},
		[]elstSpec{{segDur: 1024, mediaTime: 1024}}, nil)
	if fs := auditBytes(t, l.data); len(fs) != 0 {
		t.Fatalf("start trim produced %+v, want none", fs)
	}
}

// ── duration_unknown（gray）────────────────────────────────────────────

func TestDurationUnknownGrayWhenUnderivable(t *testing.T) {
	// 分片 + mvhd.duration=0 + trun 无 sample_duration + 无 mehd → gray。
	moov := bx("moov", cat(
		mvhdBox(1000, 0),
		bx("trak", cat(tkhdBox(0), bx("mdia", cat(mdhdBox(90000, 0), bx("minf", bx("stbl", sttsBox(nil))))))),
	))
	tfhd := fb("tfhd", 0, 0x020000, u32v(1))
	tfdt := fb("tfdt", 0, 0, u32v(0))
	trun := fb("trun", 0, 0x000001, cat(u32v(30), i32v(0))) // 只有 data-offset，无样本时长
	moof := bx("moof", bx("traf", cat(tfhd, tfdt, trun)))
	data := cat(moov, moof)

	fs := auditBytes(t, data)
	if len(fs) != 1 || fs[0].RuleID != "duration_unknown" {
		t.Fatalf("findings = %+v, want one duration_unknown", fs)
	}
	if fs[0].Status != report.Gray || fs[0].GrayReason != report.GrayDurationUnknown {
		t.Fatalf("finding = %+v, want gray duration_unknown", fs[0])
	}
}

// mehd 给出分片总时长 → 推得出，不落 gray（也不猜）。
func TestDurationUnknownSilentWhenMehdGivesIt(t *testing.T) {
	moov := bx("moov", cat(
		mvhdBox(1000, 0),
		bx("trak", cat(tkhdBox(0), bx("mdia", cat(mdhdBox(90000, 0), bx("minf", bx("stbl", sttsBox(nil))))))),
		bx("mvex", fb("mehd", 0, 0, u32v(2000))),
	))
	tfhd := fb("tfhd", 0, 0x020000, u32v(1))
	trun := fb("trun", 0, 0x000001, cat(u32v(30), i32v(0)))
	moof := bx("moof", bx("traf", cat(tfhd, trun)))
	if fs := auditBytes(t, cat(moov, moof)); len(fs) != 0 {
		t.Fatalf("derivable duration produced %+v, want none", fs)
	}
}

// ── 没 mvhd / 空树：不 panic、不报 ─────────────────────────────────────

func TestNoMovieHeaderProducesNothing(t *testing.T) {
	if fs := auditBytes(t, bx("ftyp", []byte("isom"))); len(fs) != 0 {
		t.Fatalf("no moov produced %+v", fs)
	}
}

// 截断文件不得 panic（本模块对 raw 的读取全带边界检查）。
func TestTruncatedInputDoesNotPanic(t *testing.T) {
	l := buildMovie(1000, 2000, 90000, 90000, 1000, [][2]uint32{{30, 3000}},
		[]elstSpec{{segDur: 1500, mediaTime: 3000}}, nil)
	root, err := isobmff.Parse(l.data)
	if err != nil {
		t.Fatal(err)
	}
	for cut := len(l.data); cut > 8; cut -= 4 {
		if _, err := Audit(root, l.data[:cut], "sample.mp4"); err != nil {
			t.Fatalf("cut=%d: audit error %v", cut, err)
		}
	}
}

// C11 对齐：媒体时基 tick → 影片时基 tick（整数、向下取整，无浮点）。
func TestScaleDownIsIntegerFloor(t *testing.T) {
	cases := []struct {
		v, to, from, want int64
	}{
		{3000, 1000, 90000, 33},
		{90000, 1000, 90000, 1000},
		{49152, 1000, 48000, 1024},
		{1000, 1000, 600, 1666},
		{1500, 90000, 1000, 135000},
		{7, 1000, 0, 7}, // 病态 from=0：原样返回，不 panic
	}
	for _, c := range cases {
		if got := scale(c.v, c.to, c.from); got != c.want {
			t.Errorf("scale(%d,to=%d,from=%d) = %d, want %d", c.v, c.to, c.from, got, c.want)
		}
	}
}
