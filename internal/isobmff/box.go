// Package isobmff 提供 ISO BMFF（MP4/MOV）盒子树的**只读**迭代遍历与 IR。
//
// 纪律：
//   - 只读字节、不解码任何样本载荷（mdat 不读入内存）；
//   - 遍历**迭代**（显式栈），深盒树不得爆栈；
//   - size==1（64 位 largesize）与 size==0（到文件尾）两种合法形态都要正确展开；
//   - 结构异常只报一次，不连环报（同一根因合并）。
package isobmff

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Box 是盒子树的一个节点。
type Box struct {
	Type       string
	Offset     uint64 // 盒头起始绝对偏移
	HeaderSize uint64 // 8（普通）/ 16（largesize）/ 8（到文件尾形态）
	Size       uint64 // 含头的总长
	Depth      int
	Children   []*Box
	// ParentEnd 是该盒载荷的结束偏移（父盒或文件尾），用于越界判定。
	ParentEnd uint64
}

// PayloadStart 返回载荷起始偏移。
func (b *Box) PayloadStart() uint64 { return b.Offset + b.HeaderSize }

// PayloadEnd 返回载荷结束偏移。
func (b *Box) PayloadEnd() uint64 { return b.Offset + b.Size }

// PayloadLen 返回载荷长度。
func (b *Box) PayloadLen() uint64 { return b.Size - b.HeaderSize }

// MaxDepth 是遍历深度上限（防御畸形输入；超限产生结构 finding 而非 panic）。
const MaxDepth = 64

var (
	// ErrShortHeader 表示剩余字节不足以构成盒头。
	ErrShortHeader = errors.New("isobmff: short box header")
	// ErrBadSize 表示 size 字段非法（小于头长）。
	ErrBadSize = errors.New("isobmff: box size smaller than header")
)

// containerTypes 是"已知容器盒"集合：只有这些盒才递归展开子盒。
// 用白名单而非"尝试解析"：避免把样本数据误当盒头。
var containerTypes = map[string]struct{}{
	"moov": {}, "trak": {}, "mdia": {}, "minf": {}, "stbl": {}, "edts": {},
	"dinf": {}, "mvex": {}, "moof": {}, "traf": {}, "mfra": {}, "udta": {},
	"meta": {}, "ilst": {}, "sinf": {}, "schi": {}, "stsd_": {}, "meco": {},
	"iprp": {}, "ipco": {}, "strk": {}, "sthd": {}, "tref": {}, "gmhd": {},
}

// fullBoxContainers 是带 4 字节 version/flags 前缀的容器盒（子盒从 +4 开始）。
var fullBoxContainers = map[string]struct{}{
	"meta": {},
}

// IsContainer 报告 type 是否为已知容器盒。
func IsContainer(typ string) bool {
	_, ok := containerTypes[typ]
	return ok
}

// Parse 从字节流构建盒子树。只读取盒头；不读 mdat 载荷。
//
// 返回的 root 是伪根（Type="", Size=len(data)），其 Children 为顶层盒。
func Parse(data []byte) (*Box, error) {
	root := &Box{Type: "", Offset: 0, HeaderSize: 0, Size: uint64(len(data)), ParentEnd: uint64(len(data))}
	if err := parseChildren(root, data); err != nil {
		return nil, err
	}
	return root, nil
}

// parseChildren 迭代填充 parent.Children。
//
// 用显式栈：元素 = (父盒, 游标区间)。深度超限时停止展开并记录（由规则层转 finding）。
func parseChildren(parent *Box, data []byte) error {
	type frame struct {
		parent *Box
		start  uint64
		end    uint64
	}
	stack := []frame{{parent: parent, start: parent.PayloadStart(), end: parent.PayloadEnd()}}
	if _, ok := fullBoxContainers[parent.Type]; ok {
		stack[0].start += 4
	}

	for len(stack) > 0 {
		fr := stack[len(stack)-1]
		off := fr.start
		if off >= fr.end {
			stack = stack[:len(stack)-1]
			continue
		}
		if fr.end > uint64(len(data)) {
			return fmt.Errorf("%w: parent %q end=%d beyond file len=%d", ErrBadSize, fr.parent.Type, fr.end, len(data))
		}
		if off+8 > fr.end {
			return fmt.Errorf("%w: at 0x%X (remaining %d bytes)", ErrShortHeader, off, fr.end-off)
		}
		size32 := binary.BigEndian.Uint32(data[off : off+4])
		typ := string(data[off+4 : off+8])
		var size uint64
		headerSize := uint64(8)
		switch size32 {
		case 0:
			// 到文件尾（合法形态）
			size = fr.end - off
		case 1:
			if off+16 > fr.end {
				return fmt.Errorf("%w: largesize header truncated at 0x%X", ErrShortHeader, off)
			}
			size = binary.BigEndian.Uint64(data[off+8 : off+16])
			headerSize = 16
			if size < headerSize {
				return fmt.Errorf("%w: at 0x%X (%s) size=%d", ErrBadSize, off, typ, size)
			}
		default:
			size = uint64(size32)
			if size < headerSize {
				return fmt.Errorf("%w: at 0x%X (%s) size=%d", ErrBadSize, off, typ, size)
			}
		}
		box := &Box{
			Type: typ, Offset: off, HeaderSize: headerSize, Size: size,
			Depth: fr.parent.Depth + 1, ParentEnd: fr.end,
		}
		fr.parent.Children = append(fr.parent.Children, box)

		// 推进游标：子盒消费掉自己的区间
		fr.start = off + size
		stack[len(stack)-1] = fr

		if IsContainer(typ) && box.Depth < MaxDepth && box.PayloadLen() > 0 {
			stack = append(stack, frame{parent: box, start: box.PayloadStart(), end: box.PayloadEnd()})
		}
	}
	return nil
}

// Walk 以迭代方式访问全部节点（先序）。callback 返回 false 时停止。
// 这是唯一允许的遍历入口：调用方不得自己递归。
func Walk(root *Box, fn func(*Box) bool) {
	stack := []*Box{root}
	for len(stack) > 0 {
		b := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if b != root && !fn(b) {
			return
		}
		for i := len(b.Children) - 1; i >= 0; i-- {
			stack = append(stack, b.Children[i])
		}
	}
}

// Find 返回第一个类型匹配的盒（深度优先，文件内顺序）。
func Find(root *Box, typ string) *Box {
	var found *Box
	Walk(root, func(b *Box) bool {
		if b.Type == typ {
			found = b
			return false
		}
		return true
	})
	return found
}

// FindAll 返回全部类型匹配的盒（按文件内顺序）。
func FindAll(root *Box, typ string) []*Box {
	var out []*Box
	Walk(root, func(b *Box) bool {
		if b.Type == typ {
			out = append(out, b)
		}
		return true
	})
	return out
}
