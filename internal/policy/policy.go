// Package policy 是全项目**唯一的数值口径实现处**。
//
// 真相源：data/README.md §5。本包只把那些口径落成常量与判定函数，
// 其它任何包（含测试）都**不得**再写一份同样的数字（plan/04 §四、
// plan/06 R3）。改动这里的任何常量，必须同时改 data/README.md §5。
package policy

// Version 是 CLI 的语义版本；发布时由 -ldflags 覆盖。
// 唯一定义处：这里。`node tools/tasks.mjs bin|cross` 注入同一值（见 tasks.mjs 的 VERSION）。
const Version = "0.1.0"

// TickTolerance 是时长口径互比的容差（单位：对齐后的 tick）。
// 出处：data/README.md §5.1（工程口径，非规范条款）。
const TickTolerance int64 = 1

// TimescaleSuspectBelow 是"极小 timescale"阈值（低于此值 → gray: timescale_suspect）。
// 出处：data/README.md §5.2。
const TimescaleSuspectBelow int64 = 100

// MatrixOrthoTolerance 是 display matrix 正交性容差，单位 1/65536。
// 出处：data/README.md §5.3。
const MatrixOrthoTolerance int64 = 1

// DTSRegressionTolerance 是 DTS 单调判定的容差；0 表示任何回退都算缺陷。
// 出处：data/README.md §5.4。
const DTSRegressionTolerance int64 = 0

// 退出码（判定通道 0/1/2；用法/解析通道 3 不计入判定）。
// 出处：data/README.md §5.6。
const (
	ExitPass         = 0 // fail == 0 && gray == 0
	ExitFail         = 1 // fail >= 1
	ExitInconclusive = 2 // fail == 0 && gray >= 1
	ExitUsage        = 3 // 参数非法 / 不可读 / 解析失败（不进判定通道）
)

// ExitCodeFor 把三态计数映射为退出码。这是唯一实现处（测试断言优先级 1 > 2 > 0）。
func ExitCodeFor(fails, grays int) int {
	switch {
	case fails > 0:
		return ExitFail
	case grays > 0:
		return ExitInconclusive
	default:
		return ExitPass
	}
}

// DurationWithinTolerance 判断两个已对齐到同一 timescale 的时长是否在容差内一致。
func DurationWithinTolerance(a, b int64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= TickTolerance
}

// Square 返回 v 的平方（定点正交性检查用，避免浮点）。
func Square(v int64) int64 { return v * v }
