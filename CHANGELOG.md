# Changelog

本项目从 `v0.1.1` 起维护本文件（`plan/RELEASE-M6.md` U6 登记的缺口：首版 `v0.1.0` 无历史可比，故未建）。
格式参照 Keep a Changelog；版本号遵循 [SemVer](https://semver.org/lang/zh-CN/)。

## [0.1.1] — 2026-10-28

> **这是一个行为变更版本**：非 ISO BMFF 输入的退出码从 `1`（FAIL）改为 `3`（无法解析）。
> 若你用退出码做门禁（`0/1/2/3` 四态），请把 `3` 当作"不在判定通道"，不要当"文件有问题"。

### Fixed

- **非媒体输入不再被判定为 FAIL**（诊断 D1）：`internal/isobmff` 增加首盒子嗅探
  （首盒 `type` 必须属已知顶层集合、`size` 必须为 `0`/`1` 或落在 `[8, 文件长]` 内），
  不满足返回新错误 `ErrNotISOBMFF`；`cmd/mediatrace` 把它归到 **exit 3**，措辞只用 basename。
  修复前：任意纯文本 / MKV 字节 → `VERDICT: FAIL` + exit 1（"看不懂"被说成"文件有问题"）；
  修复后：→ exit 3 且 stdout 无 `VERDICT`。**92 例夹具的判定行为一字不变**。

### Changed

- **文档与目录一致**（D2）：删除空目录 `internal/mkv/`；`plan/01`/`plan/02`/`plan/00`/`plan/06`/`data/README`/`docs/rules.md`
  里"MKV 侧做 EBML 结构对应项"的承诺统一改为"**首期不做（非目标）**"，`README` 补「首期只支持 ISO BMFF」。
- **CI 构建形态 == 发布形态**（D3）：`cross` job 的三条 `go build` 补
  `-ldflags "-s -w -X main.version=0.1.1"`（与 `tools/tasks.mjs` 逐字一致），并加 linux 产物 `< 2500000` B 断言。
- **CI 工具链钉到 `1.24.5`**（D4/DEC5）：三处 `go-version` 从 `1.21` 改为 `1.24.5`，与发布物实际构建版本一致。
  `go.mod` 的 `go 1.21` 指令**未改**（那是语言下限，与工具链版本是两件事）。
- **`update/` 收编**（DEC4）：`.gitignore` 加 `update/`，`tools/check-desensitized.mjs` 的 `SKIP_DIRS` 加 `"update"`。
- GitHub topics 补齐 7 个；`actions/checkout`、`actions/setup-go`、`actions/setup-node` 升到 `v7`。

### Added

- 两条**自述数字守门**（D5）：`TestRulesDocCoversAllRuleIDs`（AST 抽 `RuleIDs`，断言 `docs/rules.md` 全覆盖，
  抽到 0 条即 FAIL）、`TestFixtureCountDocumented`（`manifest.json` 的 `len(cases)` vs README 每处「N 例」，
  README 一处都没有即 FAIL）。两个数都现取，无常量白名单。
- 10 条嗅探相关测试（7 条 `internal/isobmff` 单测 + 3 条 `cmd/mediatrace` 端到端）。
- `data/README §6.1` 的**复核锚**（`review_interval: annual` / `next_review: 2027-10-28`，只登记不建门禁）。

### Documented

- 把"逐字节可复现"的**两条边界**写进 `README` / `plan/02` / `plan/RELEASE-M6.md`：
  ①只在同一 Go 工具链版本内成立（发布物由 go1.24.5 构建）；②必须在**真 clone** 里构建——
  `git archive` 导出树没有 `.git`，`vcs.*` 退化为 `(devel)`，产物尺寸完全相同但 sha256 不同（实测）。

### 未改

规则判定条件（M1–M5）、`internal/policy` 的四个口径常量、`fixtures/out/manifest.json` 与 92 例字节、
`data/README §5` 数值口径、退出码语义（0/1/2/3）、`v0.1.0` 的 tag/Release 与两条失败 run 记录。

## [0.1.0] — 2026-10-28

首个公开版本：五个判定模块 M1–M5 + 92 例程序化夹具 + 三平台单二进制。留档见 `plan/RELEASE-M6.md`。
