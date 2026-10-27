module github.com/yuluo554/mediatrace

go 1.21

// 零第三方依赖是硬约束（plan/02 §2.2）：require 块必须为空，go.sum 不得存在。
// internal/archtest 有测试断言这一点。
