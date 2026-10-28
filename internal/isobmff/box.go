// Package isobmff 提供 ISO BMFF（MP4/MOV）盒子树的**只读**迭代遍历与 IR。
//
// 纪律：
//   - 只读字节、不解码任何样本载荷（mdat 不读入内存）；
//   - 遍历**迭代**（显式栈），深盒树不得爆栈；
//   - size==1（64 位 largesize）与 size==0（到文件尾）两种合法形态都要正确展开；
//   - 结构异常只报一次，不连环报（同一根因合并）：越界的子盒**子树不再展开**；
//   - 内容级异常（size<8、子盒越界、声明 size 越过文件、深度超限）**不 panic、
//     不硬失败**，而是记成 `Anomaly` 交给规则层转 finding；返回 error 的只有两种
//     "盒子树无法建立"的情形（CLI 均映射退出码 3，不进判定通道）：
//     ① 剩余 < 8 字节（`ErrShortHeader`）；② 首盒子不像 ISO BMFF（`ErrNotISOBMFF`，
//     纯文本 / MKV / 随机字节）。
package isobmff

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Box 是盒子树的一个节点。
type Box struct {
	Type       string
	Path       string // 斜杠盒路径（索引从 0 起，按文件内顺序），Parse 后填充
	Offset     uint64 // 盒头起始绝对偏移
	HeaderSize uint64 // 8（普通）/ 16（largesize）/ 0（伪根）
	Size       uint64 // 含头的总长（size==0 时=到文件尾）
	Depth      int
	Children   []*Box
	// ParentEnd 是该盒载荷的结束偏移（父盒载荷末尾或文件尾），用于越界判定。
	ParentEnd uint64
	// Anomalies 是遍历期记录的**内容级结构异常**；只在伪根上有意义。
	Anomalies []Anomaly
}

// PayloadStart 返回载荷起始偏移。
func (b *Box) PayloadStart() uint64 { return b.Offset + b.HeaderSize }

// PayloadEnd 返回载荷结束偏移。
func (b *Box) PayloadEnd() uint64 { return b.Offset + b.Size }

// PayloadLen 返回载荷长度（HeaderSize<=Size 时恒非负）。
func (b *Box) PayloadLen() uint64 {
	if b.Size < b.HeaderSize {
		return 0
	}
	return b.Size - b.HeaderSize
}

// MaxDepth 是遍历深度上限（防御畸形输入；超限产生结构 finding 而非 panic）。
// 深盒树（≥5000 层）在此处停止展开，故不爆栈。
const MaxDepth = 64

// AnomalyKind 是遍历期记录的内容级结构异常类别。
type AnomalyKind string

const (
	// AnomTooSmallSize：盒 size < 8（且非 0/1 形态），越界/截断的**结构性证据**。
	AnomTooSmallSize AnomalyKind = "too_small_size"
	// AnomMaxDepth：容器盒嵌套达到深度上限，更深的子盒未展开（防御）。
	AnomMaxDepth AnomalyKind = "max_depth_exceeded"
)

// Anomaly 是遍历期记录的一条结构异常（offset 恒有值，绝不吞）。
type Anomaly struct {
	Kind     AnomalyKind
	Offset   uint64
	BoxType  string
	// Parent 是异常所在的父盒（AnomTooSmallSize 用其 Path 拼字段路径；
	// AnomMaxDepth 时 Parent 就是触发上限的那个盒本身）。
	Parent *Box
	// Declared 是声明的 size（AnomTooSmallSize 用）。
	Declared uint64
	// Depth 是触发异常处的深度。
	Depth int
}

var (
	// ErrShortHeader 表示剩余字节不足以构成盒头（树无法建立 → CLI 退出码 3）。
	ErrShortHeader = errors.New("isobmff: short box header")
	// ErrNotISOBMFF 表示输入**根本不是 ISO BMFF**：首盒子的类型不在已知顶层集合里，
	// 或首盒子的 size 不自洽（非 0/1 形态下 < 8 或 > 文件长度）。
	// 这是"看不懂"而不是"文件里的盒子有问题"——CLI 映射退出码 3（data/README §5.6），
	// **不进判定通道**。刻意不复用 ErrShortHeader：后者的语义是"剩余字节不够构盒头"，
	// 拿来报"这是一段纯文本"会让错误信息说成 too short（诊断 D1 的教训）。
	ErrNotISOBMFF = errors.New("isobmff: not an ISO BMFF stream")
	// ErrBadSize 保留给"size 字段非法"的历史语义；当前实现把 size<8 记为
	// AnomTooSmallSize（转 finding），不再以此 error 直接失败。
	ErrBadSize = errors.New("isobmff: box size smaller than header")
)

// topLevelTypes 是"允许出现在文件首部"的已知顶层盒类型白名单。
//
// 为什么需要它：没有首盒嗅探时，任意非媒体字节都会被当成盒子树解析
// （`hello world…` → size=0x68656c6c、type="o wo"），得到 FAIL + 退出码 1 ——
// "看不懂"被判定成"文件有问题"（诊断 D1）。白名单取 ISO BMFF 的顶层盒全集：
// `uuid`/`pdin` 等少见但与媒体无关的盒也放行，因为它们的出现不构成"不是 BMFF"。
var topLevelTypes = map[string]struct{}{
	"ftyp": {}, "styp": {}, "moov": {}, "mdat": {}, "free": {}, "skip": {},
	"wide": {}, "pnot": {}, "uuid": {}, "sidx": {}, "moof": {}, "mfra": {},
	"meta": {}, "pdin": {}, "emsg": {},
}

// sniffFirstBox 校验首盒子像不像 ISO BMFF 的顶层盒。
//
// 判据（两条都过才继续解析）：
//   - 首盒子 type ∈ topLevelTypes；
//   - 首盒子 size 为 0（到文件尾）/ 1（largesize）或落在 [8, len(data)] 内。
//
// 只嗅探**第一个**盒子：后续盒子由 parseChildren 的正常遍历规则处理，
// 内容级异常（`too_small_size` / 越界）仍走 Anomaly → finding 的通道，不在这里拦。
func sniffFirstBox(data []byte) error {
	size32 := binary.BigEndian.Uint32(data[0:4])
	typ := string(data[4:8])
	if _, ok := topLevelTypes[typ]; !ok {
		return fmt.Errorf("%w: first box type %q is not a known top-level box", ErrNotISOBMFF, typ)
	}
	switch size32 {
	case 0, 1: // 到文件尾 / 64 位 largesize：两种合法形态
		return nil
	}
	size := uint64(size32)
	if size < 8 {
		return fmt.Errorf("%w: first box %q declares size %d (< 8-byte header)", ErrNotISOBMFF, typ, size)
	}
	if size > uint64(len(data)) {
		return fmt.Errorf("%w: first box %q declares size %d > file size %d", ErrNotISOBMFF, typ, size, len(data))
	}
	return nil
}

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
// 返回的 root 是伪根（Type=""、Size=len(data)），其 Children 为顶层盒，
// Anomalies 为遍历期记录的内容级结构异常。
//
// 返回 error 的两种情形（都映射 CLI 退出码 3，都不进判定通道）：
//   - ErrShortHeader：剩余字节不足以读出盒头（连根盒头都读不出）；
//   - ErrNotISOBMFF：首盒子不像 ISO BMFF 顶层盒（纯文本、MKV/EBML、随机字节）。
func Parse(data []byte) (*Box, error) {
	root := &Box{Offset: 0, HeaderSize: 0, Size: uint64(len(data)), ParentEnd: uint64(len(data))}
	if len(data) < 8 {
		return nil, fmt.Errorf("%w: file too short (%d bytes < 8)", ErrShortHeader, len(data))
	}
	if err := sniffFirstBox(data); err != nil {
		return nil, err
	}
	if err := parseChildren(root, data); err != nil {
		return nil, err
	}
	assignPaths(root)
	return root, nil
}

// parseChildren 迭代填充 parent.Children（显式栈，深盒树不爆栈）。
//
// 游标推进纪律：每个子盒消费 [off, off+size)；越界时游标越过 frame.end，
// 下一轮即 pop——**越界子盒的子树不再展开**（同一根因合并）。
func parseChildren(root *Box, data []byte) error {
	type frame struct {
		parent *Box
		start  uint64
		end    uint64
	}
	n := uint64(len(data))
	stack := []frame{{parent: root, start: root.PayloadStart(), end: root.PayloadEnd()}}
	if _, ok := fullBoxContainers[root.Type]; ok {
		stack[0].start += 4
	}

	for len(stack) > 0 {
		i := len(stack) - 1
		fr := stack[i]
		off := fr.start
		if off >= fr.end {
			stack = stack[:i]
			continue
		}
		if fr.end-off < 8 {
			return fmt.Errorf("%w: at 0x%X (remaining %d bytes, parent %q)", ErrShortHeader, off, fr.end-off, fr.parent.Type)
		}

		size32 := binary.BigEndian.Uint32(data[off : off+4])
		typ := string(data[off+4 : off+8])
		var size uint64
		headerSize := uint64(8)
		switch {
		case size32 == 0:
			// size==0：到文件尾（合法形态）
			size = n - off
		case size32 == 1:
			// size==1：64 位 largesize
			if fr.end-off < 16 {
				return fmt.Errorf("%w: largesize header truncated at 0x%X", ErrShortHeader, off)
			}
			size = binary.BigEndian.Uint64(data[off+8 : off+16])
			headerSize = 16
			if size < 16 {
				root.Anomalies = append(root.Anomalies, Anomaly{
					Kind: AnomTooSmallSize, Offset: off, BoxType: typ,
					Parent: fr.parent, Declared: size, Depth: fr.parent.Depth + 1,
				})
				stack = stack[:i] // 边界不可知，停止本层兄弟序列
				continue
			}
		default:
			size = uint64(size32)
			if size < 8 {
				root.Anomalies = append(root.Anomalies, Anomaly{
					Kind: AnomTooSmallSize, Offset: off, BoxType: typ,
					Parent: fr.parent, Declared: size, Depth: fr.parent.Depth + 1,
				})
				stack = stack[:i] // 边界不可知，停止本层兄弟序列
				continue
			}
		}

		box := &Box{
			Type: typ, Offset: off, HeaderSize: headerSize, Size: size,
			Depth: fr.parent.Depth + 1, ParentEnd: fr.end,
		}
		fr.parent.Children = append(fr.parent.Children, box)
		stack[i].start = off + size // 可能越过 fr.end → 下轮 pop（越界盒子树不展开）

		if IsContainer(typ) && box.PayloadLen() > 0 {
			if box.Depth < MaxDepth {
				start := box.PayloadStart()
				if _, ok := fullBoxContainers[typ]; ok {
					start += 4
				}
				end := box.PayloadEnd()
				if end > n {
					end = n // 声明越过文件的容器：只在可读区间内展开
				}
				if start < end {
					stack = append(stack, frame{parent: box, start: start, end: end})
				}
			} else {
				root.Anomalies = append(root.Anomalies, Anomaly{
					Kind: AnomMaxDepth, Offset: off, BoxType: typ,
					Parent: box, Declared: size, Depth: box.Depth,
				})
			}
		}
	}
	return nil
}

// assignPaths 为每个盒填 Path（斜杠路径，索引从 0 起，按文件内顺序）。
// 同一父盒下**同名兄弟**才带 [i]；这与会话约定的 `moov/trak[0]/edts/elst[1]` 一致。
// 迭代实现（不用递归），且只按 Children 切片顺序输出（确定性）。
func assignPaths(root *Box) {
	type frame struct {
		b      *Box
		prefix string
	}
	root.Path = ""
	stack := []frame{{root, ""}}
	for len(stack) > 0 {
		i := len(stack) - 1
		f := stack[i]
		stack = stack[:i]
		kids := f.b.Children
		if len(kids) == 0 {
			continue
		}
		counts := make(map[string]int, len(kids))
		for _, c := range kids {
			counts[c.Type]++
		}
		names := make([]string, len(kids))
		seen := make(map[string]int, len(kids))
		for j, c := range kids {
			name := c.Type
			if counts[c.Type] > 1 {
				name = fmt.Sprintf("%s[%d]", c.Type, seen[c.Type])
			}
			seen[c.Type]++
			names[j] = name
		}
		for j := len(kids) - 1; j >= 0; j-- {
			full := names[j]
			if f.prefix != "" {
				full = f.prefix + "/" + names[j]
			}
			kids[j].Path = full
			stack = append(stack, frame{kids[j], full})
		}
	}
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
