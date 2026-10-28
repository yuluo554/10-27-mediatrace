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

> **已回填**（推送与 Release 完成后由执行方补齐；以下均为第 2–6 步实测值）。

| 项 | 事实 |
| --- | --- |
| tag 对象 / 剥壳提交 | tag OID `33ad94c417bbfbf1c3357aa5a919ac40b6bb3818`（`git cat-file -t` = `tag`，annotated）；剥壳提交 `22df1e733b95efa52eece1608a558d31637dd3b4`（== 已验 CI 的 HEAD）；tagger `yuluo554 <282769740+yuluo554@users.noreply.github.com>` |
| Release URL / draft 状态 | https://github.com/yuluo554/10-27-mediatrace/releases/tag/v0.1.1 ；实测 `isDraft=false` / `isPrerelease=false` |
| 三附件尺寸 + sha256 | linux-amd64 2068664 B `67c3e65f19291458fefa7ab069ae6ee7ff3ead9ffbec2982a66fd0086e5ab199`；darwin-arm64 2111730 B `486d669af9ceb46df075a93b0b16c1b957aa8468aa6f87da53464aaf6b98da2b`；windows-amd64.exe 2201088 B `ab094631a57ae14d935b132a11b2a5297d0a44d2f7254fba33c51da470a35323`（与 tag 树真 clone 重建件 `cmp` 逐字节相同） |
| `SHA256SUMS` 的 sha256 | `dc60a32ed00dedc3fdf135b08eeed2403670ae673c8081c2e2aaf2ad12069ad7`（274 B，三行 LF，`*` 二进制模式，沿用 v0.1.0 形态） |
| 真下载复核 | 中立目录 `gh release download v0.1.1` → `sha256sum -c SHA256SUMS` 三件 OK / rc=0；`cmp` 三件均 `IDENTICAL`（LINUX/DARWIN/WINDOWS） |
| CI run id 与结论 | push main：run `40695823858`（headSha `22df1e7`，event=push）6 条 check-run 全 `success`（layout / test(ubuntu) / test(macos) / test(windows) / fixtures / cross）。注解 5 条，全部 notice：ubuntu-latest→Ubuntu26 迁移 4 条（cross / fixtures / layout / test-ubuntu）+ macOS 队列 1 条（test-macos）；**Node.js 20 弃用 warning 0 条**。注：本仓 workflow 仅触发 `push: branches:[main]` 与 `pull_request`，**无 `tags:` 触发**，故 tag 推送不产生第二条 run（本轮实测 run 列表 event 全为 push）。 |
| `v0.1.0` 是否动过 | **未动**（历史不改写）：四附件尺寸仍为 2064568 / 2095202 / 2198528 / 274，`isDraft=false` |
| ⚠ **POST_TAG 声明（tag 树内已知陈旧文字）** | **v0.1.1 tag 树里有三处状态文字仍写"待推送/待 tag"**，而本版实际已发布：<br>`README.md:27`（`v0.1.1 已定版待推送`）、`plan/00-README总览.md:11`（`已定版，待推送/待 tag`）、`plan/05-里程碑与发布.md:23`（`待推送/待 tag`）。<br>原因：tag 打在 `22df1e7`（已验 CI 的提交）上，这三行是随后才改对的。**处置 = 声明 + 追加修正，不改写已发布的 tag**（与本仓 `v0.1.0` 的 POST_TAG 做法一致）。<br>核对命令：`git show v0.1.1:README.md | grep -c 待推送` → 期望 **1**（旧 tag 树如此）；`git show HEAD:README.md | grep -c 待推送` → 期望 **0**。<br>**不影响任何二进制与附件**（该三行是文档文字，不参与任何产物的构建）。 |
