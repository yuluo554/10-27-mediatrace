package structure

import (
	"encoding/binary"
	"testing"

	"github.com/yuluo554/10-27-mediatrace/internal/isobmff"
	"github.com/yuluo554/10-27-mediatrace/internal/report"
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
	head[0] = version
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

// ── too_small_size（无夹具覆盖 → 本地构造）───────────────────────────────

func TestTooSmallSizeIsFindingNotParseError(t *testing.T) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[0:4], 4) // < 8
	copy(buf[4:8], "free")

	fs := auditBytes(t, buf)
	if len(fs) != 1 || fs[0].RuleID != "too_small_size" {
		t.Fatalf("findings = %+v, want one too_small_size", fs)
	}
	f := fs[0]
	if f.Status != report.Fail || f.ByteOffset != 0 || f.FieldPath == "" {
		t.Fatalf("bad evidence: %+v", f)
	}
	if f.Delta == nil || f.Delta.Value != 4 || f.Delta.Unit != "byte" {
		t.Fatalf("delta = %+v, want 4 byte (8-minimum shortfall)", f.Delta)
	}
}

// ── ftyp brand ↔ stsd 样本条目 ─────────────────────────────────────────

func movieWithStsdEntry(entryType string) []byte {
	entry := bx(entryType, make([]byte, 16))
	stsd := fb("stsd", 0, 0, cat(u32v(1), entry))
	stbl := bx("stbl", stsd)
	minf := bx("minf", stbl)
	mdia := bx("mdia", minf)
	trak := bx("trak", mdia)
	return bx("moov", trak)
}

func ftypWithBrand(major string) []byte {
	return bx("ftyp", cat([]byte(major), u32v(0x200), []byte("isomiso2")))
}

func TestFtypBrandMismatchFiresOnCodecBrandContradiction(t *testing.T) {
	data := cat(ftypWithBrand("avc1"), movieWithStsdEntry("hvc1"))
	fs := auditBytes(t, data)
	if got := rulesOf(fs); got["ftyp_brand_mismatch"] != 1 {
		t.Fatalf("rules = %v, want one ftyp_brand_mismatch", got)
	}
}

func TestFtypBrandGenericNeverFires(t *testing.T) {
	// 通用 brand（isom）不与任何编码冲突——对照组误报 0 的关键。
	for _, major := range []string{"isom", "iso2", "mp41", "qt  "} {
		fs := auditBytes(t, cat(ftypWithBrand(major), movieWithStsdEntry("hvc1")))
		if got := rulesOf(fs); len(got) != 0 {
			t.Fatalf("brand %q produced findings %v, want none", major, got)
		}
	}
}

func TestFtypBrandMatchIsClean(t *testing.T) {
	fs := auditBytes(t, cat(ftypWithBrand("hvc1"), movieWithStsdEntry("hvc1")))
	if got := rulesOf(fs); len(got) != 0 {
		t.Fatalf("matching brand produced findings %v", got)
	}
}

// ── duplicate_moov / duplicate_mdat ────────────────────────────────────

func TestDuplicateMoovListsAllOffsets(t *testing.T) {
	moov := bx("moov", nil)
	data := cat(bx("ftyp", []byte("isom")), moov, moov)
	fs := auditBytes(t, data)
	if len(fs) != 1 || fs[0].RuleID != "duplicate_moov" {
		t.Fatalf("findings = %+v", fs)
	}
	if len(fs[0].Values) != 2 {
		t.Fatalf("values = %+v, want both moov offsets", fs[0].Values)
	}
	// 首个证据指向第 2 处
	if fs[0].ByteOffset != uint64(len(bx("ftyp", []byte("isom")))+len(moov)) {
		t.Fatalf("byte_offset = %d, want second moov", fs[0].ByteOffset)
	}
}

func TestDuplicateMdatFiresWithoutMoof(t *testing.T) {
	data := cat(bx("ftyp", []byte("isom")), bx("mdat", make([]byte, 4)), bx("mdat", make([]byte, 4)))
	fs := auditBytes(t, data)
	if got := rulesOf(fs); got["duplicate_mdat"] != 1 {
		t.Fatalf("rules = %v, want duplicate_mdat", got)
	}
}

// 分片（含 moof）每片自带 mdat 是合法形态 → 不得报 duplicate_mdat（对照组误报 0）。
func TestDuplicateMdatSuppressedWhenFragmented(t *testing.T) {
	data := cat(
		bx("ftyp", []byte("isom")),
		bx("moof", nil), bx("mdat", make([]byte, 4)),
		bx("moof", nil), bx("mdat", make([]byte, 4)),
	)
	fs := auditBytes(t, data)
	if got := rulesOf(fs); len(got) != 0 {
		t.Fatalf("fragmented file produced findings %v, want none", got)
	}
}

// ── child_overruns_parent ──────────────────────────────────────────────

func TestChildOverrunsParent(t *testing.T) {
	udta := bx("udta", make([]byte, 16))
	binary.BigEndian.PutUint32(udta[0:4], uint32(len(udta)+4096)) // 声明远大于父盒剩余
	data := cat(bx("ftyp", []byte("isom")), bx("moov", udta))
	fs := auditBytes(t, data)
	if got := rulesOf(fs); got["child_overruns_parent"] != 1 {
		t.Fatalf("rules = %v, want child_overruns_parent", got)
	}
}
