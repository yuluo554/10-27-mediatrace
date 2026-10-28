package display

import (
	"encoding/binary"
	"testing"

	"github.com/yuluo554/10-27-mediatrace/internal/isobmff"
	"github.com/yuluo554/10-27-mediatrace/internal/report"
)

// ── 字节构造小工具（测试内自足，不引用生成器：K8）──────────────────────────

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
func u16(v uint16) []byte { b := make([]byte, 2); binary.BigEndian.PutUint16(b, v); return b }

func matrixBytes(m [9]uint32) []byte {
	var out []byte
	for _, v := range m {
		out = append(out, u32(v)...)
	}
	return out
}

func identity() [9]uint32 { return [9]uint32{0x00010000, 0, 0, 0, 0x00010000, 0, 0, 0, 0x40000000} }

func tkhd(m [9]uint32, w, h uint32) []byte {
	payload := cat(
		u32(0), u32(0), u32(1), u32(0), u32(0),
		make([]byte, 8), make([]byte, 8),
		matrixBytes(m), u32(w), u32(h),
	)
	return full("tkhd", 0, payload)
}

func mdhd(ts uint32) []byte {
	return full("mdhd", 0, cat(u32(0), u32(0), u32(ts), u32(0), u16(0x55C4), u16(0)))
}

func hdlr(kind string) []byte {
	return full("hdlr", 0, cat(u32(0), []byte(kind), make([]byte, 12), []byte("V\x00")))
}

// visualEntry 拼 avc1 视觉样本条目（78 字节固定头 + 子盒），存储宽高写入固定头。
func visualEntry(storedW, storedH uint16, children ...[]byte) []byte {
	head := make([]byte, 78)
	binary.BigEndian.PutUint16(head[6:8], 1)  // data_reference_index
	binary.BigEndian.PutUint16(head[24:26], storedW)
	binary.BigEndian.PutUint16(head[26:28], storedH)
	return box("avc1", cat(head, cat(children...)))
}

func stsdOf(entry []byte) []byte { return full("stsd", 0, cat(u32(1), entry)) }

func colrNCLX(primaries, transfer, mtrx uint16) []byte {
	return box("colr", cat([]byte("nclx"), u16(primaries), u16(transfer), u16(mtrx), []byte{0}))
}

func mdcv(maxL, minL, maxC, minC uint32) []byte {
	return box("mdcv", cat(make([]byte, 8), u32(maxL), u32(minL), u32(maxC), u32(minC)))
}

func clli(maxCLL, avgCLL uint16) []byte {
	return box("clli", cat(u16(maxCLL), u16(avgCLL)))
}

// buildVideo 组装单视频轨文件。
func buildVideo(m [9]uint32, dispW, dispH, storedW, storedH uint16, children ...[]byte) []byte {
	stbl := box("stbl", cat(
		stsdOf(visualEntry(storedW, storedH, children...)),
		full("stts", 0, cat(u32(1), u32(30), u32(3000))),
	))
	minf := box("minf", stbl)
	mdia := box("mdia", cat(mdhd(90000), hdlr("vide"), minf))
	trak := box("trak", cat(tkhd(m, uint32(dispW)<<16, uint32(dispH)<<16), mdia))
	return cat(box("ftyp", []byte("isom")), box("moov", trak))
}

func auditM5(t *testing.T, raw []byte) []report.Finding {
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

// F15：`hdr_metadata_missing`（gray）无夹具，由本单测钉住：bt2020 colr 但无 mdcv/clli。
func TestHDRMetadataMissingIsGray(t *testing.T) {
	raw := buildVideo(identity(), 640, 480, 640, 480, colrNCLX(9, 16, 9))
	fs := auditM5(t, raw)
	if len(fs) != 1 {
		t.Fatalf("want 1 finding, got %d: %+v", len(fs), fs)
	}
	f := fs[0]
	if f.RuleID != "hdr_metadata_missing" || f.Status != report.Gray {
		t.Fatalf("want gray hdr_metadata_missing, got %+v", f)
	}
	if f.GrayReason != report.GrayHDRMetadataMissing {
		t.Fatalf("gray reason = %q, want %q", f.GrayReason, report.GrayHDRMetadataMissing)
	}
}

// bt2020 colr + 完整 mdcv/clli 且量程合法 → 不报（无夹具，单测覆盖）。
func TestHDRCompleteIsClean(t *testing.T) {
	raw := buildVideo(identity(), 640, 480, 640, 480, colrNCLX(9, 16, 9), mdcv(1000, 1, 500, 1), clli(1000, 400))
	if fs := auditM5(t, raw); len(fs) != 0 {
		t.Fatalf("complete HDR metadata should be clean, got %+v", fs)
	}
}

// F15：`hdr_range_suspect` 无夹具，由本单测钉住：mdcv max<=min → fail。
func TestHDRRangeSuspect(t *testing.T) {
	raw := buildVideo(identity(), 640, 480, 640, 480, mdcv(0, 0, 500, 1))
	fs := auditM5(t, raw)
	if len(fs) != 1 || fs[0].RuleID != "hdr_range_suspect" || fs[0].Status != report.Fail {
		t.Fatalf("want fail hdr_range_suspect, got %+v", fs)
	}
	if fs[0].Delta == nil || fs[0].Delta.Unit != "nits" {
		t.Fatalf("range finding needs a nits delta: %+v", fs[0].Delta)
	}
}

// matrix 正交性是整数判据：单位阵通过，剪切阵失败（K15）。
func TestMatrixOrthogonalityIsExact(t *testing.T) {
	if fs := auditM5(t, buildVideo(identity(), 640, 480, 640, 480)); len(fs) != 0 {
		t.Fatalf("identity matrix should be orthogonal, got %+v", fs)
	}
	shear := [9]uint32{0x00010000, 0x00008000, 0, 0, 0x00010000, 0, 0, 0, 0x40000000}
	fs := auditM5(t, buildVideo(shear, 640, 480, 640, 480))
	if len(fs) != 1 || fs[0].RuleID != "matrix_not_orthogonal" || fs[0].Status != report.Fail {
		t.Fatalf("shear matrix should fail orthogonality, got %+v", fs)
	}
}

// RuleIDs 必须与文档对账的六条一致。
func TestRuleIDsAreComplete(t *testing.T) {
	want := map[string]bool{
		"matrix_not_orthogonal": true, "rotation_dimension_conflict": true, "sar_dar_conflict": true,
		"hdr_metadata_missing": true, "hdr_range_suspect": true, "colr_multi_instance": true,
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
