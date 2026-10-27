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

// size<8 不是"树无法建立"，而是**内容级结构异常**：记成 Anomaly 交给规则层，
// 不硬失败（M2 起 `too_small_size` 是 finding，不是退出码 3）。
func TestParseRecordsTooSmallSize(t *testing.T) {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[0:4], 4) // < 8
	copy(buf[4:8], "free")
	root, err := Parse(buf)
	if err != nil {
		t.Fatalf("size<8 must not be a parse error: %v", err)
	}
	if len(root.Anomalies) != 1 {
		t.Fatalf("anomalies = %d, want 1 (%+v)", len(root.Anomalies), root.Anomalies)
	}
	a := root.Anomalies[0]
	if a.Kind != AnomTooSmallSize || a.Offset != 0 || a.BoxType != "free" || a.Declared != 4 {
		t.Fatalf("anomaly = %+v", a)
	}
}

// 连根盒头都读不出（< 8 字节）→ 树无法建立 → error（CLI 退出码 3）。
func TestParseRejectsTruncatedHeader(t *testing.T) {
	if _, err := Parse([]byte{0, 0, 0, 1, 'm', 'd'}); err == nil {
		t.Fatal("expected ErrShortHeader")
	}
	if _, err := Parse(nil); err == nil {
		t.Fatal("expected error for empty file")
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

// 深盒树（≥5000 层）不 panic、不爆栈，且记一条 max_depth_exceeded 异常（M2 DoD）。
func TestDeepNestingAbove5000RecordsDepthAnomaly(t *testing.T) {
	const depth = 6000
	var data []byte
	for i := 0; i < depth; i++ {
		data = box("udta", data)
	}
	root, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	maxDepth, tooSmall := 0, 0
	for _, a := range root.Anomalies {
		switch a.Kind {
		case AnomMaxDepth:
			maxDepth++
			if a.Depth != MaxDepth {
				t.Fatalf("depth anomaly at depth %d, want %d", a.Depth, MaxDepth)
			}
		case AnomTooSmallSize:
			tooSmall++
		}
	}
	if maxDepth != 1 {
		t.Fatalf("max_depth anomalies = %d, want exactly 1", maxDepth)
	}
	if tooSmall != 0 {
		t.Fatalf("unexpected too_small_size anomalies: %d", tooSmall)
	}
	// 迭代遍历同样不得爆栈
	visited := 0
	Walk(root, func(*Box) bool { visited++; return true })
	if visited == 0 {
		t.Fatal("walk visited nothing")
	}
}

// PayloadLen 在 size<header 时不得下溢（防 uint64 回绕）。
func TestPayloadLenDoesNotUnderflow(t *testing.T) {
	b := &Box{Size: 4, HeaderSize: 8}
	if b.PayloadLen() != 0 {
		t.Fatalf("PayloadLen = %d, want 0", b.PayloadLen())
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
