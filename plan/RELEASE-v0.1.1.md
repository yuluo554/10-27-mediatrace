# RELEASE — mediatrace `v0.1.1`（补丁版：非媒体输入退出码修正）

> **状态：定版完成、待推送 / 待 tag / 待 Release。** 本文按 `plan/RELEASE-M6.md` 的两栏结构写：
> 「一、tag 前事实」在 tag 之前落盘；「二、tag 后追加」由**推送并建 Release 的那一棒**回填（不许提前写）。
> 判定分档：**实测** / **源码直读** / **推断**。

## 一、tag 前事实

| 项 | 事实 | 证据 |
| --- | --- | --- |
| 版本 | `0.1.1` | 单源两处：`internal/policy.Version` + `tools/tasks.mjs` 的 `VERSION`；由 `TestVersionSingleSourceMatchesBuildScript` 机器断言（实测 `ok`） |
| 为什么发版 | **行为变更**：非 ISO BMFF 输入从 `FAIL/exit 1` 改为 `exit 3` | 诊断 D1；`CHANGELOG.md` §0.1.1 |
| 变更清单 | 见 `CHANGELOG.md` §0.1.1 | 源码直读 |
| 测试（`go vet` + `go test ./... -count=1`） | **11 包全 `ok`**（全仓 111 条 `Test*`；新增 10 条） | 实测（工作树 + 干净 clone 各一次） |
| 夹具 | `92 例与入库 manifest 逐字节一致`；`len(cases)` = 92 | 实测 `node tools/tasks.mjs check-fixtures` |
| 92 例退出码 | **0 不符**（fail→1 / gray→2 / pass→0） | 实测（工作树 + 干净 clone） |
| 布局门 | 11 PASS / 结论: 全绿 / exit 0 | 实测 |
| 脱敏门 | 扫 60 份 / 0 命中 / exit 0（`update/` 已在 `SKIP_DIRS`） | 实测 |
| 门自证 | 21 项全绿（含 `M10.RELEASE` 真被咬住） | 实测 `node tools/tasks.mjs verify-gates` |
| 新增守门 | `TestRulesDocCoversAllRuleIDs`（32 条全覆盖）+ `TestFixtureCountDocumented`（README 每处「N 例」== 92），均含空跑守卫，且各做过一次真负控 | 实测 |
| CI 构建 == 发布构建 | `cross` 三条 build 带 `-trimpath -ldflags "-s -w -X main.version=0.1.1"`（与 `tasks.mjs` 逐字一致）+ linux 产物 `< 2500000` B 断言 | 源码直读 + 本地 `cross` 实测（linux 2068664 B，`< 2500000` 成立） |
| 构建工具链 | **go1.24.5**（CI 三处 `go-version` 已钉同版本）；`go.mod` 的 `go 1.21` 是语言下限，未改 | 实测 |
| 复现边界 | ①只在同一 Go 版本内逐字节复现；②**必须真 clone**（`git archive` 导出树无 `.git` → `vcs.*` 退化 `(devel)`，尺寸相同但 sha256 不同） | 实测（本轮新发现） |
| 发布物来源 | 必须从 **`v0.1.1` tag 树的真 clone** 重建；仓库内 `dist/` 是陈旧残留（既不匹配 `v0.1.0` 也不匹配 HEAD） | 实测三组 hash 互不相同 |
| 已知未验证 | macOS/Linux 实机运行；go1.21 重建差异（推断）；真 `.mov`/`.m4a`；`--modules` 全组合 | 见 `update/03-执行记录.md §4` |

**发布动作（由推送方执行，顺序不可换）**

1. `git push origin main` → **确认 CI 全绿**（尤其 `cross` 的体积断言与 `@v7` actions 能正常拉取）
2. `git tag -a v0.1.1 -m "mediatrace v0.1.1"` → `git push origin v0.1.1`
3. `git clone --branch v0.1.1 <repo> /tmp/mt-v011` → `GO=<go1.24.5> node tools/tasks.mjs cross`
4. `sha256sum dist/*` → 写 `SHA256SUMS`（**沿用 v0.1.0 的实际形态**：`<hash>` + 一个空格 + `*` + 文件名，见下）→ `gh release create v0.1.1 dist/* --title ... --notes-file ...`
   实测 v0.1.0 的 `SHA256SUMS` 是 `3983f851…ce08 *mediatrace-linux-amd64`（274 B，三行，LF 结尾）——`*` 是 Git Bash 下 `sha256sum` 的二进制模式标记，
   `sha256sum -c` 与 `shasum -a 256 -c` 都认；**不要手工改成两空格形态**，那会与上一版不一致。
5. 中立目录 `gh release download v0.1.1` → `sha256sum -c SHA256SUMS` → 与本机重建件 `cmp`
6. 回填本文「二、tag 后追加」

> **不要用**仓库里的 `dist/` 上传：它既不匹配 `v0.1.0` 也不匹配 HEAD。

## 二、tag 后追加

> **待回填**（推送与 Release 完成后由执行方补齐；在此之前本栏为空是**如实状态**，不是遗漏）。

| 项 | 事实 |
| --- | --- |
| tag 对象 / 剥壳提交 | 待填 |
| Release URL / draft 状态 | 待填 |
| 三附件尺寸 + sha256 | 待填（须与第 3 步的本机重建件逐字节相同） |
| `SHA256SUMS` 的 sha256 | 待填 |
| 真下载复核 | 待填 |
| CI run id 与结论 | 待填 |
| `v0.1.0` 是否动过 | **未动**（历史不改写） |
