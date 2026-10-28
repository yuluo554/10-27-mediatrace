// Package rules 定义五个判定模块的统一接口与注册表。
//
// 纪律（plan/03 §1.2）：
//   - 每个模块可单独运行，互不依赖对方内部结构；
//   - Audit 是纯函数：不含 I/O、不读时钟、不读随机、不访问网络；
//   - 模块之间不得互相 import（由 internal/archtest 的 AST 测试守）。
package rules

import (
	"errors"

	"github.com/yuluo554/mediatrace/internal/isobmff"
	"github.com/yuluo554/mediatrace/internal/report"
	"github.com/yuluo554/mediatrace/internal/rules/duration"
	"github.com/yuluo554/mediatrace/internal/rules/structure"
	"github.com/yuluo554/mediatrace/internal/rules/timestamps"
)

// ErrNotImplemented 表示该模块在骨架期尚未实现。
// CLI 必须把它映射成退出码 3（用法/不可判通道），**不得**当 PASS 输出。
var ErrNotImplemented = errors.New("rules: module not implemented")

// Input 是规则层的输入（只读）。
type Input struct {
	// FileName 只用于报告显示（basename），不得含目录。
	FileName string
	// Root 是已解析的盒子树伪根。
	Root *isobmff.Box
	// Raw 是文件只读字节；大 mdat 由解析层决定不入内存（见 plan/02 §1.3）。
	Raw []byte
}

// Module 是一个判定模块。
type Module interface {
	// ID 返回模块号：M1..M5。
	ID() string
	// Audit 返回该模块的全部 finding（含 pass/gray/fail）。
	Audit(Input) ([]report.Finding, error)
}

// registry 是模块注册表；顺序即 CLI 的默认执行顺序。
// 新增模块必须同时改 plan/00 §五 的映射表与 docs/rules.md。
var registry = []Module{
	m1{}, // 结构自洽（M2 已实现 → internal/rules/structure）
	m2{}, // 时长口径（M3 已实现 → internal/rules/duration）
	m3{}, // 时间戳（M4 已实现 → internal/rules/timestamps）
	m4{}, // 偏移 / priming
	m5{}, // 显示与色彩
}

// All 返回全部模块（副本，防止调用方改写注册表）。
func All() []Module {
	out := make([]Module, len(registry))
	copy(out, registry)
	return out
}

// ByIDs 按模块号筛选；未知模块号返回错误（CLI 映射成退出码 3）。
func ByIDs(ids []string) ([]Module, error) {
	if len(ids) == 0 {
		return All(), nil
	}
	index := map[string]Module{}
	for _, m := range registry {
		index[m.ID()] = m
	}
	out := make([]Module, 0, len(ids))
	for _, id := range ids {
		m, ok := index[id]
		if !ok {
			return nil, errors.New("rules: unknown module " + id + " (want M1..M5)")
		}
		out = append(out, m)
	}
	return out, nil
}

// ── 模块实现 ─────────────────────────────────────────────────────────────
// m1（结构自洽）转调 internal/rules/structure；m2（时长口径）转调
// internal/rules/duration；m3（时间戳）转调 internal/rules/timestamps。
// 模块间不互相 import（archtest 守）；三个实现包都只依赖 isobmff/report/policy，
// 故此处单向依赖无环。
// m4/m5 仍是桩：Audit 返回 ErrNotImplemented，**不得**改成"返回空 findings"冒充 PASS。

type m1 struct{}

func (m1) ID() string { return structure.ModuleID }

func (m1) Audit(in Input) ([]report.Finding, error) {
	return structure.Audit(in.Root, in.Raw, in.FileName)
}

type m2 struct{}

func (m2) ID() string { return duration.ModuleID }

func (m2) Audit(in Input) ([]report.Finding, error) {
	return duration.Audit(in.Root, in.Raw, in.FileName)
}

type m3 struct{}

func (m3) ID() string { return timestamps.ModuleID }

func (m3) Audit(in Input) ([]report.Finding, error) {
	return timestamps.Audit(in.Root, in.Raw, in.FileName)
}

type m4 struct{}

func (m4) ID() string                            { return "M4" }
func (m4) Audit(Input) ([]report.Finding, error) { return nil, ErrNotImplemented }

type m5 struct{}

func (m5) ID() string                            { return "M5" }
func (m5) Audit(Input) ([]report.Finding, error) { return nil, ErrNotImplemented }
