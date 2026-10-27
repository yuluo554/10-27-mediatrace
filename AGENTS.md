# AGENTS.md

本仓库遵循 **`ai-tool-project-sprint`** 方法论（装订日期：2026-10-27）。

## 任何新对话的第一件事

```bash
node tools/tasks.mjs gates     # = 布局门 + 脱敏门（本机没有 make，见"环境事实"）
```

必须全绿；红了**先修布局/脱敏**，再干活。

- 续接对话：读 `plan/HANDOFF-M<k>.md`（**交给第 k 棒**的入口交接单；当前棒 = `plan/00 §五` 第一个 ⬜），
  **先跑上面这条门**。
- **收尾每一棒必须写下一棒的 `HANDOFF-M<k+1>.md`**——这是硬约束①，不是可选项；
  `LAYOUT.HANDOFF_ENTRY` 会在"当前棒没有入口交接单"时变红（骨架期实测漏写过一次，见 `plan/06 §六 B7`）。
- 接手对话（无 HANDOFF）：先读 `plan/00`、`plan/05`、`plan/06` 与各 HANDOFF，
  重建"当前状态 + K 口径清单"再动手。
- **门不存在 = 本项目没装订** → 先补装订（第零步），再做别的。

## 强制布局（布局门逐条断言，改动前先看这里）

| 项 | 位置 | 判据 |
| --- | --- | --- |
| 计划套件 | `plan/00`–`06` + `data/README.md` | 存在且各含必需小节 |
| 数据台账 | `data/README.md`、`data/real-samples.json` | 台账在位（来源 + 三态） |
| 交接单 | `plan/HANDOFF-M<k>.md` | **当前棒必须有入口交接单（七节齐）**——`LAYOUT.HANDOFF_ENTRY`；每个 ✅ 里程碑各一份——`LAYOUT.HANDOFFS`。命名 = "交给第 k 棒"（第 k-1 棒收尾时写） |
| 里程碑状态副本 | `plan/00` 台账行 + `plan/00` 里程碑表 + `plan/05` 状态表 + `README.md` 状态表 | **逐处一致**；副本必须含表格行才可机读，散文状态行贡献 0 行且门会把每副本命中行数打出来 |
| 发布留档 | `plan/RELEASE-M6.md`（两栏）或 `plan/WRAPUP-M6.md` | 标题齐、无占位 |
| 发布门覆盖对账 | `plan/RELEASE-M6.md` §覆盖对账 | 阶段 7 出口门逐条：实现 / 不适用+理由 / 待首推 |
| 文档点名的工具 | `tools/*.mjs` / `tools/**/*.go` | **`git ls-files` 指得到实物**（布局门 `LAYOUT.DOC_TOOLS`） |

## 门命令

| 命令 | 作用 | 出口 |
| --- | --- | --- |
| `node tools/check-sprint-layout.mjs` | 布局门（11 条判据） | 0 全绿 / 1 FAIL / 2 用法错误 |
| `node tools/check-desensitized.mjs` | 脱敏门（本机路径/用户名/回环代理，4 条规则） | 0 零命中 / 1 命中 |
| `node tools/verify-layout-gate.mjs` | **两门自己的变异验证**：21 组（11 判据 + 4 规则 + 基线/对照）逐组必须按预期变红 | 0 全部按预期 / 1 有假门 |
| `node tools/tasks.mjs gates` | 前两条并跑 | 全绿 |
| `node tools/tasks.mjs test` | `go vet` + `go test ./...` | 全绿 |
| `node tools/tasks.mjs all` | gates + verify-gates + check-ignore-rules + test + check-fixtures + smoke | 全绿 |
| `node tools/tasks.mjs check-ignore-rules` | gitignore 三态（台账必须入库 / 样本与产物不入库） | 全绿 |
| `node tools/tasks.mjs cross` | 三平台交叉编译 | 全绿 |
| `make <target>` | 上表的薄委托（**本机无 make**） | 同上 |

## 环境事实与禁忌

- **Go 不在本机 `PATH`**（实测 2026-10-27）：实际路径 `C:\Users\<user>\go-sdk\go\bin\go.exe`，
  版本 `go1.24.5 windows/amd64`。`tools/tasks.mjs` 会自动探测该路径；也可显式
  `GO=/path/to/go node tools/tasks.mjs test`。
- **本机没有 `make`**（实测 2026-10-27）：`Makefile` 只是对 `node tools/tasks.mjs <target>`
  的薄委托。**文档里凡写 `make x`，等价且可执行的是 `node tools/tasks.mjs x`**。
- **Windows 上 `bin/mediatrace` 不能直接执行**（CreateProcess 需要 `.exe`；文件存在也 ENOENT，
  实测）。`tasks.mjs` 按平台拼 `bin/mediatrace.exe`；CI 的 Linux 产物名不带后缀。
- Node `v24.15.0` 在 `PATH`；Python 仅 `3.8.8`——**本项目不用 Python**（生成器与 CLI 同为 Go）。
- **禁忌**：把"未执行"写成已完成；在 tag 之后改代码面；把 `bin/`、`dist/`、`.mimosa/`、媒体样本入库。
- **不得**把 `check-sprint-layout` 的判据改宽以换绿灯（要改就同时更新本文件与变异清单）。
