package isobmff

import (
	"encoding/binary"
	"errors"
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
//
// 注意构造形态：**首盒必须是合法顶层盒**（此处用 `ftyp`），畸形盒放在其后。
// 契约变化（首盒嗅探，见 Parse 的 ErrNotISOBMFF）后，"整个文件就是一个 size=4 的盒子"
// 按定义不是 ISO BMFF，会在嗅探处 exit 3 —— 那属于另一条规则（非媒体输入），
// 与本条要验的"盒子内部 size<8 转 finding"是两件事。断言本身（返回 anomaly 而非 error）未改。
func TestParseRecordsTooSmallSize(t *testing.T) {
	data := append(box("ftyp", nil), make([]byte, 8)...)
	binary.BigEndian.PutUint32(data[8:12], 4) // < 8
	copy(data[12:16], "free")
	root, err := Parse(data)
	if err != nil {
		t.Fatalf("size<8 must not be a parse error: %v", err)
	}
	if len(root.Anomalies) != 1 {
		t.Fatalf("anomalies = %d, want 1 (%+v)", len(root.Anomalies), root.Anomalies)
	}
	a := root.Anomalies[0]
	if a.Kind != AnomTooSmallSize || a.Offset != 8 || a.BoxType != "free" || a.Declared != 4 {
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

// ── 首盒子嗅探（诊断 D1：非媒体输入必须是退出码 3，不是 FAIL）────────────────

// 合法首盒（ftyp + moov）必须照旧解析成功——嗅探只在"不像 BMFF"时拦截。
func TestSniffAcceptsLegalFtyp(t *testing.T) {
	data := append(box("ftyp", []byte("isom")), box("moov", box("mvhd", nil))...)
	root, err := Parse(data)
	if err != nil {
		t.Fatalf("legal ftyp must parse: %v", err)
	}
	if len(root.Children) != 2 {
		t.Fatalf("children = %d, want 2", len(root.Children))
	}
}

// 纯文本（40 字节，实测量级）→ ErrNotISOBMFF，绝不能走判定通道。
// 实测对照（诊断 D1）：旧行为得到 VERDICT: FAIL + exit 1，box 名 "o wo"。
func TestSniffRejectsPlainText(t *testing.T) {
	data := []byte("hello world this is not a media file at all")
	if len(data) != 43 {
		t.Fatalf("fixture drift: want 43 bytes, got %d", len(data))
	}
	_, err := Parse(data)
	if !errors.Is(err, ErrNotISOBMFF) {
		t.Fatalf("plain text must be ErrNotISOBMFF, got %v", err)
	}
}

// 真 MKV 头（EBML magic 0x1A45DFA3 + 4 字节 0）→ ErrNotISOBMFF。
// 诊断 D1 实测旧行为：box " mkv" child_overruns_parent, exit 1。
func TestSniffRejectsMatroskaHeader(t *testing.T) {
	data := []byte{0x1a, 0x45, 0xdf, 0xa3, 0x00, 0x00, 0x00, 0x00}
	_, err := Parse(data)
	if !errors.Is(err, ErrNotISOBMFF) {
		t.Fatalf("matroska header must be ErrNotISOBMFF, got %v", err)
	}
}

// 已知顶层类型但 size 不自洽（声明 6 < 8）→ 也是"不像 BMFF"，不是 too_small_size。
// 这条是嗅探里 size 判据的阳性对照：只改 type 白名单不足以拦住随机字节。
func TestSniffRejectsImpossibleFirstSize(t *testing.T) {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint32(buf[0:4], 6) // < 8，且非 0/1 形态
	copy(buf[4:8], "ftyp")
	if _, err := Parse(buf); !errors.Is(err, ErrNotISOBMFF) {
		t.Fatalf("size=6 first box must be ErrNotISOBMFF, got %v", err)
	}

	binary.BigEndian.PutUint32(buf[0:4], 4096) // 声明超过文件长度 16
	if _, err := Parse(buf); !errors.Is(err, ErrNotISOBMFF) {
		t.Fatalf("size > file must be ErrNotISOBMFF, got %v", err)
	}
}

// 空文件（0 字节）保持既有行为：ErrShortHeader，不是 ErrNotISOBMFF。
// 两个错误的区别是"太短"与"不像盒子"，错误信息要能讲清（别耦合）。
func TestSniffEmptyFileStaysShortHeader(t *testing.T) {
	_, err := Parse([]byte{})
	if !errors.Is(err, ErrShortHeader) {
		t.Fatalf("empty file must stay ErrShortHeader, got %v", err)
	}
	if errors.Is(err, ErrNotISOBMFF) {
		t.Fatalf("empty file must not be ErrNotISOBMFF: %v", err)
	}
}

// 首盒用 size==1（64 位 largesize）是合法形态，嗅探必须放行。
func TestSniffAcceptsLargesizeFirstBox(t *testing.T) {
	payload := []byte{1, 2, 3}
	buf := make([]byte, 16+len(payload))
	binary.BigEndian.PutUint32(buf[0:4], 1) // size == 1 → largesize
	copy(buf[4:8], "mdat")
	binary.BigEndian.PutUint64(buf[8:16], uint64(len(buf)))
	copy(buf[16:], payload)

	root, err := Parse(buf)
	if err != nil {
		t.Fatalf("largesize first box must parse: %v", err)
	}
	if b := root.Children[0]; b.HeaderSize != 16 || b.Size != uint64(len(buf)) {
		t.Fatalf("largesize first box header=%d size=%d", b.HeaderSize, b.Size)
	}
}

// 首盒 size==0（到文件尾）同样是合法形态。
func TestSniffAcceptsSizeZeroFirstBox(t *testing.T) {
	buf := make([]byte, 8+4)
	binary.BigEndian.PutUint32(buf[0:4], 0) // size == 0
	copy(buf[4:8], "mdat")
	if _, err := Parse(buf); err != nil {
		t.Fatalf("size==0 first box must parse: %v", err)
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
//
// 构造形态：首盒显式用合法顶层盒 `ftyp`，深链从第二个盒开始。首盒嗅探
// （Parse 的 ErrNotISOBMFF）之后，"文件就是一棵 udta 树"按定义不是 ISO BMFF；
// 本条要验的是**深链展开**，故前面垫一个合法顶层盒。断言本身一条未改。
func TestDeepNestingAbove5000RecordsDepthAnomaly(t *testing.T) {
	const depth = 6000
	var data []byte
	for i := 0; i < depth; i++ {
		data = box("udta", data)
	}
	data = append(box("ftyp", nil), data...)
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
