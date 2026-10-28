package offset

import (
	"encoding/binary"
	"testing"

	"github.com/yuluo554/mediatrace/internal/isobmff"
	"github.com/yuluo554/mediatrace/internal/report"
)

// ── 字节构造小工具（测试内自足，不引用生成器：K8 生成器与解析器不共享代码）─────

func box(typ string, payload []byte) []byte {
	b := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(b[0:4], uint32(8+len(payload)))
	copy(b[4:8], typ)
	copy(b[8:], payload)
	return b
}

func full(typ string, ver byte, payload []byte) []byte {
	h := make([]byte, 4)
	h[0] = ver
	return box(typ, append(h, payload...))
}

func cat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func u32(v uint32) []byte { b := make([]byte, 4); binary.BigEndian.PutUint32(b, v); return b }
func i32(v int32) []byte  { return u32(uint32(v)) }
func u16(v uint16) []byte { b := make([]byte, 2); binary.BigEndian.PutUint16(b, v); return b }

func identityMatrix() []byte {
	m := []uint32{0x00010000, 0, 0, 0, 0x00010000, 0, 0, 0, 0x40000000}
	var out []byte
	for _, v := range m {
		out = append(out, u32(v)...)
	}
	return out
}

// tkhd 用 v0，matrix 为单位阵；w/h 由参数给（16.16）。
func tkhdMatrix(w, h uint32) []byte {
	payload := cat(
		u32(0), u32(0), u32(1), u32(0), u32(0), // co/mod/id/reserved/duration
		make([]byte, 8), make([]byte, 8), // reserved2 + layer..reserved
		identityMatrix(), u32(w), u32(h),
	)
	return full("tkhd", 0, payload)
}

func mdhdTS(ts uint32) []byte {
	payload := cat(u32(0), u32(0), u32(ts), u32(0), u16(0x55C4), u16(0))
	return full("mdhd", 0, payload)
}

func hdlrKind(kind string) []byte {
	payload := cat(u32(0), []byte(kind), make([]byte, 12), []byte("H\x00"))
	return full("hdlr", 0, payload)
}

// audioEntry 拼一个 mp4a 样本条目（28 字节固定头），children 追加其后。
func audioEntry(children ...[]byte) []byte {
	head := cat(make([]byte, 6), u16(1), make([]byte, 8), u16(2), u16(16), u16(0), u16(0), u32(48000<<16))
	return box("mp4a", cat(head, cat(children...)))
}

func stsdOf(entry []byte) []byte { return full("stsd", 0, cat(u32(1), entry)) }

func cttsFirst(offset uint32) []byte {
	return full("ctts", 0, cat(u32(1), u32(1), u32(offset)))
}

func sgpdRoll(roll int16) []byte {
	return full("sgpd", 1, cat([]byte("roll"), u32(2), u32(1), u16(uint16(roll))))
}

func elstTrim(segDur uint32, mediaTime int32) []byte {
	e := full("elst", 0, cat(u32(1), u32(segDur), i32(mediaTime), u32(0x00010000)))
	return box("edts", e)
}

// buildAudio 组装一个只含单音频轨的 moov；stbl 里按需放 stsd/ctts/sgpd，edts 可选。
func buildAudio(withEdts, withCtts, withRoll bool, trimMediaTime int32) []byte {
	var parts [][]byte
	parts = append(parts, stsdOf(audioEntry()))
	if withCtts {
		parts = append(parts, cttsFirst(1024))
	}
	parts = append(parts, full("stts", 0, cat(u32(1), u32(48), u32(1024))))
	if withRoll {
		parts = append(parts, sgpdRoll(1024))
	}
	stbl := box("stbl", cat(parts...))
	minf := box("minf", stbl)
	mdia := box("mdia", cat(mdhdTS(48000), hdlrKind("soun"), minf))
	kids := [][]byte{tkhdMatrix(0, 0)}
	if withEdts {
		kids = append(kids, elstTrim(1024, trimMediaTime))
	}
	kids = append(kids, mdia)
	trak := box("trak", cat(kids...))
	moov := box("moov", trak)
	return cat(box("ftyp", []byte("isom")), moov)
}

func auditM4(t *testing.T, raw []byte) []report.Finding {
	t.Helper()
	root, err := isobmff.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	fs, err := Audit(root, raw, "t.mp4")
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	return fs
}

// F15：`no_edts_but_nonzero_start`（gray no_edts）无夹具，由本单测钉住判据与证据塑形。
// 形态：音频轨首帧后移 1024、**有** roll 声明但**无** edit —— 无法判断是裁剪还是缺陷 → gray。
func TestNoEdtsButNonzeroStartIsGray(t *testing.T) {
	raw := buildAudio(false, true, true, 0)
	fs := auditM4(t, raw)
	if len(fs) != 1 {
		t.Fatalf("want 1 finding, got %d: %+v", len(fs), fs)
	}
	f := fs[0]
	if f.RuleID != "no_edts_but_nonzero_start" || f.Status != report.Gray {
		t.Fatalf("want gray no_edts_but_nonzero_start, got %+v", f)
	}
	if f.GrayReason != report.GrayNoEdts {
		t.Fatalf("gray reason = %q, want %q", f.GrayReason, report.GrayNoEdts)
	}
	if f.FieldPath == "" {
		t.Fatalf("gray finding should still name the field: %+v", f)
	}
}

// 无 roll 声明且首帧后移 → fail `priming_undeclared`（同一形态的可判分支）。
func TestPrimingUndeclaredWithoutRoll(t *testing.T) {
	raw := buildAudio(false, true, false, 0)
	fs := auditM4(t, raw)
	if len(fs) != 1 || fs[0].RuleID != "priming_undeclared" || fs[0].Status != report.Fail {
		t.Fatalf("want fail priming_undeclared, got %+v", fs)
	}
	if fs[0].Delta == nil || fs[0].Delta.Value != 1024 || fs[0].Delta.Unit != "sample" {
		t.Fatalf("undeclared shift must be reported as 1024 samples: %+v", fs[0].Delta)
	}
}

// K14：期望 priming = **声明值**，不是"必须等于 1024"。
// 同一裁剪量在不同 roll 声明下给出相反结论，证明判定用的是声明而非硬编码。
func TestPrimingExpectationComesFromDeclaration(t *testing.T) {
	// 裁剪 1024 + roll 1024 → 合法
	if fs := auditM4(t, buildAudio(true, false, true, 1024)); len(fs) != 0 {
		t.Fatalf("trim==roll 1024 should be clean, got %+v", fs)
	}
	// 裁剪 2112 + roll 2112 → 合法（2112 也是声明值，不是"必须 1024"）
	raw := buildAudioWithRoll(true, 2112, 2112)
	if fs := auditM4(t, raw); len(fs) != 0 {
		t.Fatalf("trim==roll 2112 should be clean, got %+v", fs)
	}
}

// start_trim 证据：报秒值 + 样本数（裁剪 2048、roll 1024 → 越过声明）。
func TestStartTrimCarriesSecondsAndSamples(t *testing.T) {
	raw := buildAudio(true, false, true, 2048)
	fs := auditM4(t, raw)
	if len(fs) != 1 || fs[0].RuleID != "start_trim" || fs[0].Status != report.Fail {
		t.Fatalf("want fail start_trim, got %+v", fs)
	}
	vals := map[string]int64{}
	for _, v := range fs[0].Values {
		vals[v.Name] = v.Value
	}
	if vals["start_trim_samples"] != 2048 {
		t.Fatalf("start_trim_samples = %d, want 2048 (%+v)", vals["start_trim_samples"], vals)
	}
	if vals["start_trim_millis"] != 42 { // 2048/48000*1000 = 42.66 → 42
		t.Fatalf("start_trim_millis = %d, want 42 (%+v)", vals["start_trim_millis"], vals)
	}
	if fs[0].Delta == nil || fs[0].Delta.Value != 1024 {
		t.Fatalf("start_trim delta should be trim-roll = 1024: %+v", fs[0].Delta)
	}
}

// buildAudioWithRoll 允许指定 roll 与裁剪量（K14 的正反例）。
func buildAudioWithRoll(withEdts bool, roll, trim int32) []byte {
	stbl := box("stbl", cat(
		stsdOf(audioEntry()),
		full("stts", 0, cat(u32(1), u32(48), u32(1024))),
		sgpdRoll(int16(roll)),
	))
	minf := box("minf", stbl)
	mdia := box("mdia", cat(mdhdTS(48000), hdlrKind("soun"), minf))
	kids := [][]byte{tkhdMatrix(0, 0)}
	if withEdts {
		kids = append(kids, elstTrim(1024, trim))
	}
	kids = append(kids, mdia)
	trak := box("trak", cat(kids...))
	moov := box("moov", trak)
	return cat(box("ftyp", []byte("isom")), moov)
}

// RuleIDs 必须与文档对账的六条一致（防漏做/多做）。
func TestRuleIDsAreComplete(t *testing.T) {
	want := map[string]bool{
		"start_trim": true, "priming_undeclared": true, "priming_mismatch": true,
		"no_edts_but_nonzero_start": true, "no_roll_distance": true, "unknown_codec_priming": true,
	}
	got := map[string]bool{}
	for _, r := range RuleIDs {
		got[r] = true
	}
	if len(got) != len(want) {
		t.Fatalf("RuleIDs = %v, want %v", RuleIDs, want)
	}
	for r := range want {
		if !got[r] {
			t.Fatalf("missing rule id %q", r)
		}
	}
}
