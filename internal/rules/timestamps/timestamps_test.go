package timestamps

import (
	"encoding/binary"
	"testing"

	"github.com/yuluo554/10-27-mediatrace/internal/isobmff"
	"github.com/yuluo554/10-27-mediatrace/internal/policy"
	"github.com/yuluo554/10-27-mediatrace/internal/report"
)

// 证据塑形：`dts_regression` 必须报**样本序号 + stts 条目号 + 字节偏移**，
// 且四元组齐（渲染层 Validate 必须接受）。
//
// 说明：`stts.duration` 无符号 ⇒ 良构 stts 的累计 DTS 单调不减，字面判据
// `dts[n] < dts[n-1]` 在本夹具集里不可能被触发（见包注释）。故此处**直接**驱动
// finding 塑形函数，钉住"报什么、带什么证据"的契约；可达的 C5「回退 DTS」形态由
// `zero_duration_entry`（C5）与 `tfdt_regression`（C5b/C5c）在夹具级覆盖。
func TestDTSRegressionCarriesSampleIndexEntryAndOffset(t *testing.T) {
	a := &auditor{}
	a.addDTSRegression("moov/trak/stbl/stts", 2, 7, 0x1234, 9000, 8000)

	if len(a.findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(a.findings))
	}
	f := a.findings[0]
	if f.Module != ModuleID || f.RuleID != "dts_regression" || f.Status != report.Fail {
		t.Fatalf("unexpected finding header: %+v", f)
	}
	if f.ByteOffset != 0x1234 {
		t.Errorf("byte offset = 0x%X, want 0x1234", f.ByteOffset)
	}
	if want := "moov/trak/stbl/stts[2].duration"; f.FieldPath != want {
		t.Errorf("field path = %q, want %q", f.FieldPath, want)
	}
	if f.Delta == nil || f.Delta.Value != 1000 {
		t.Errorf("delta = %+v, want value 1000", f.Delta)
	}
	if f.Delta.Unit == "" {
		t.Error("delta lacks a unit")
	}
	// 样本序号与条目号必须出现在证据里。
	sampleOK, entryOK := false, false
	for _, v := range f.Values {
		switch v.Name {
		case "sample_index":
			sampleOK = v.Value == 7
		case "stts_entry":
			entryOK = v.Value == 2
		}
	}
	if !sampleOK || !entryOK {
		t.Errorf("values must carry sample_index=7 and stts_entry=2: %+v", f.Values)
	}
	if err := f.Validate(); err != nil {
		t.Errorf("fail finding must pass renderer validation (four-tuple): %v", err)
	}
}

// 位宽越界定位：`stts=[{count:2, duration:0xFFFFFF00}]` 的跨点必须落在
// 条目 0 的第 1 个样本（0 起），越过值为 2×0xFFFFFF00（≈2^32）。
// 这与夹具 `T2` 的构造点真值一致，但不依赖生成器（包级单测）。
func TestSttsOverflowLocatesEntryAndSample(t *testing.T) {
	a := &auditor{}
	overflowed := a.checkStts("moov/trak/stbl/stts", []sttsEntry{
		{Count: 2, Duration: 0xFFFFFF00, DurOff: 552},
	})
	if !overflowed {
		t.Fatal("expected overflow to be reported")
	}
	if len(a.findings) != 1 {
		t.Fatalf("want exactly one finding (same root cause merged), got %d: %+v", len(a.findings), a.findings)
	}
	f := a.findings[0]
	if f.RuleID != "dts_overflow_risk" || f.Status != report.Fail {
		t.Fatalf("unexpected finding: %+v", f)
	}
	if f.ByteOffset != 552 {
		t.Errorf("byte offset = %d, want 552 (entry 0 duration field)", f.ByteOffset)
	}
	vals := map[string]int64{}
	for _, v := range f.Values {
		vals[v.Name] = v.Value
	}
	if vals["stts_entry"] != 0 || vals["sample_index"] != 1 {
		t.Errorf("overflow must locate entry 0 / sample 1: %+v", vals)
	}
	if got, want := vals["cumulative_dts"], int64(2)*0xFFFFFF00; got != want {
		t.Errorf("cumulative_dts = %d, want %d", got, want)
	}
	if vals["field_width_max"] != int64(sttsWidthMax) {
		t.Errorf("field_width_max = %d, want %d", vals["field_width_max"], sttsWidthMax)
	}
	if err := f.Validate(); err != nil {
		t.Errorf("fail finding must carry the four-tuple: %v", err)
	}
}

// 单调 stts 不得报回退/越界（对照组纪律的最小单元）。
func TestMonotonicSttsIsClean(t *testing.T) {
	a := &auditor{}
	if a.checkStts("moov/trak/stbl/stts", []sttsEntry{{Count: 240, Duration: 3000, DurOff: 100}}) {
		t.Fatal("monotonic stts must not report overflow")
	}
	if len(a.findings) != 0 {
		t.Fatalf("monotonic stts must be clean, got %+v", a.findings)
	}
}

// duration==0 的条目必须报 `zero_duration_entry`（fail），差值 = 亏空样本时长。
func TestZeroDurationEntryReportsDeficit(t *testing.T) {
	a := &auditor{}
	a.checkStts("moov/trak/stbl/stts", []sttsEntry{
		{Count: 10, Duration: 3000, DurOff: 10},
		{Count: 5, Duration: 0, DurOff: 560},
		{Count: 15, Duration: 3000, DurOff: 30},
	})
	if len(a.findings) != 1 {
		// 注意：本函数不产 vfr（那条在 checkTracks 里），故只应有一条。
		t.Fatalf("want exactly one zero_duration finding, got %d: %+v", len(a.findings), a.findings)
	}
	f := a.findings[0]
	if f.RuleID != "zero_duration_entry" || f.ByteOffset != 560 {
		t.Fatalf("unexpected finding: %+v", f)
	}
	if want := "moov/trak/stbl/stts[1].duration"; f.FieldPath != want {
		t.Errorf("field path = %q, want %q", f.FieldPath, want)
	}
	if f.Delta == nil || f.Delta.Value != 3000 {
		t.Errorf("delta = %+v, want 3000 (deficit vs nominal)", f.Delta)
	}
}

// tfdt 分宽读：v0 是 32 位、v1 是 64 位——v1 的值必须按 64 位读出（不得截断）。
func TestReadTfdtHonoursFieldWidth(t *testing.T) {
	v0 := make([]byte, 16) // 头 8 + v/f 4 + u32
	copy(v0[4:8], "tfdt")
	binary.BigEndian.PutUint32(v0[12:16], 100) // value = 100
	b0 := &isobmff.Box{Type: "tfdt", Offset: 0, HeaderSize: 8, Size: 16}
	v1, val, off, ok := readTfdt(v0, b0)
	if !ok || v1 || val != 100 || off != 12 {
		t.Errorf("v0 decode = (%v,%d,%d,%v), want (false,100,12,true)", v1, val, off, ok)
	}

	raw := make([]byte, 20) // 头 8 + v/f (version=1) + u64
	copy(raw[4:8], "tfdt")
	raw[8] = 1
	binary.BigEndian.PutUint64(raw[12:20], 0x1_0000_0002)
	b1 := &isobmff.Box{Type: "tfdt", Offset: 0, HeaderSize: 8, Size: 20}
	v1, val, _, ok = readTfdt(raw, b1)
	if !ok || !v1 || val != 0x1_0000_0002 {
		t.Errorf("v1 decode = (%v,%d,%v), want (true, 0x100000002, true) — 64-bit value truncated?", v1, val, ok)
	}
}

// tfdt 越界：v0 的 tfdt + 分片时长越过 32 位上限即 fail；v1（64 位）不设越界。
func TestTfdtOverflowWidths(t *testing.T) {
	a := &auditor{}
	a.checkTfdtOverflow(fragInfo{index: 0, tfdt: 0xFFFFFFFF, v1: false, valOff: 1512, path: "moof/traf/tfdt", dur: 1})
	if len(a.findings) != 1 || a.findings[0].RuleID != "dts_overflow_risk" {
		t.Fatalf("v0 overflow must be reported: %+v", a.findings)
	}
	if err := a.findings[0].Validate(); err != nil {
		t.Errorf("fail finding must carry the four-tuple: %v", err)
	}

	b := &auditor{}
	b.checkTfdtOverflow(fragInfo{index: 0, tfdt: 0xFFFFFFFF, v1: true, valOff: 1512, path: "moof/traf/tfdt", dur: 1})
	if len(b.findings) != 0 {
		t.Fatalf("v1 (64-bit) must not report overflow: %+v", b.findings)
	}
}

// 阈值必须来自 policy（不得在本包另写一份数字）。
func TestTimescaleThresholdComesFromPolicy(t *testing.T) {
	if policy.TimescaleSuspectBelow <= 0 {
		t.Fatal("policy.TimescaleSuspectBelow must be positive")
	}
	if policy.DTSRegressionTolerance != 0 {
		t.Fatalf("policy.DTSRegressionTolerance = %d, want 0 (§5.4)", policy.DTSRegressionTolerance)
	}
	// RuleIDs 与 docs/rules.md 的 M3 节对账（六条，无新增）。
	want := map[string]bool{
		"dts_regression": true, "tfdt_regression": true, "zero_duration_entry": true,
		"dts_overflow_risk": true, "timescale_suspect": true, "vfr_detected": true,
	}
	if len(RuleIDs) != len(want) {
		t.Fatalf("RuleIDs = %v, want %d entries", RuleIDs, len(want))
	}
	for _, r := range RuleIDs {
		if !want[r] {
			t.Errorf("unexpected RuleID %q (added without updating docs/rules.md + data/README §5.5?)", r)
		}
	}
}