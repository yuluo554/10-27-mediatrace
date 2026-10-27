package isobmff

import (
	"encoding/binary"
	"testing"
)

// box 构造一个最小盒：size + type + payload。
func box(typ string, payload []byte) []byte {
	buf := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(buf[0:4], uint32(8+len(payload)))
	copy(buf[4:8], typ)
	copy(buf[8:], payload)
	return buf
}

func TestParseFlatAndNested(t *testing.T) {
	inner := box("mvhd", []byte{0, 0, 0, 0})
	moov := box("moov", inner)
	data := append(box("ftyp", []byte("isom")), moov...)

	root, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(root.Children) != 2 {
		t.Fatalf("top-level boxes = %d, want 2", len(root.Children))
	}
	if root.Children[0].Type != "ftyp" || root.Children[1].Type != "moov" {
		t.Fatalf("order/types wrong: %+v", root.Children)
	}
	if len(root.Children[1].Children) != 1 || root.Children[1].Children[0].Type != "mvhd" {
		t.Fatalf("nested child not parsed: %+v", root.Children[1].Children)
	}
	if got := Find(root, "mvhd"); got == nil {
		t.Fatal("Find(mvhd) returned nil")
	}
}

func TestParseSizeZeroMeansToEnd(t *testing.T) {
	// mdat with size==0 → extends to end of file
	payload := []byte{1, 2, 3, 4, 5}
	buf := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(buf[0:4], 0) // size == 0
	copy(buf[4:8], "mdat")
	copy(buf[8:], payload)

	root, err := Parse(buf)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(root.Children) != 1 {
		t.Fatalf("children = %d", len(root.Children))
	}
	b := root.Children[0]
	if b.Size != uint64(len(buf)) {
		t.Fatalf("size-zero box size = %d, want %d", b.Size, len(buf))
	}
}

func TestParseLargesize(t *testing.T) {
	payload := []byte{9, 9}
	buf := make([]byte, 16+len(payload))
	binary.BigEndian.PutUint32(buf[0:4], 1) // size == 1 → largesize
	copy(buf[4:8], "mdat")
	binary.BigEndian.PutUint64(buf[8:16], uint64(len(buf)))
	copy(buf[16:], payload)

	root, err := Parse(buf)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	b := root.Children[0]
	if b.HeaderSize != 16 || b.Size != uint64(len(buf)) {
		t.Fatalf("largesize header=%d size=%d want 16/%d", b.HeaderSize, b.Size, len(buf))
	}
}

func TestParseRejectsTooSmallSize(t *testing.T) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[0:4], 4) // < 8
	copy(buf[4:8], "free")
	if _, err := Parse(buf); err == nil {
		t.Fatal("expected ErrBadSize for size < header")
	}
}

func TestParseRejectsTruncatedHeader(t *testing.T) {
	if _, err := Parse([]byte{0, 0, 0, 1, 'm', 'd'}); err == nil {
		t.Fatal("expected ErrShortHeader")
	}
}

// 深盒树不得爆栈：10000 层嵌套。
func TestDeepNestingDoesNotStackOverflow(t *testing.T) {
	var data []byte
	for i := 0; i < 10000; i++ {
		data = box("moov", data)
	}
	root, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	count := 0
	Walk(root, func(b *Box) bool { count++; return true })
	if count == 0 {
		t.Fatal("walk visited nothing")
	}
	// 深度被 MaxDepth 截断是预期行为（防御），但不得 panic
	if count > 10000 {
		t.Fatalf("walk visited %d nodes (more than built)", count)
	}
}

func TestWalkPreOrderAndFindAll(t *testing.T) {
	data := append(box("ftyp", nil), box("moov", box("mvhd", nil))...)
	data = append(data, box("moov", nil)...) // second moov (C1 shape)
	root, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	moovs := FindAll(root, "moov")
	if len(moovs) != 2 {
		t.Fatalf("FindAll(moov) = %d, want 2", len(moovs))
	}
	if moovs[0].Offset >= moovs[1].Offset {
		t.Fatal("FindAll must return file order")
	}
}
