# RELEASE-M6 —— 发布留档（mediatrace v0.1.0）

> 发布里程碑：**M6 · 发布**（脱敏复核 + 公开仓库发布）。两栏格式由 `plan/05 §三` 规定：
> **§一** 必须在 tag **之前**写完并进 tag 树；**§二** 在 tag **之后**逐行追加。
>
> **为什么 §一 不含附件 sha256**：本文件在 tag 树内，而附件由 tag 树构建——
> 把哈希写进本文件会改变构建产物的输入（自指失效）。故 §一 只写**复验方法**，
> 哈希值写进 Release 正文与 §二 追加栏（阶段 7 出口门「附件 sha256 只有两个不动点」）。
>
> 口径真相源：`data/README.md §5`；计划状态只查 `plan/00 §五`。

---

## 一、tag 前事实

> 本节全部为本机实测（Windows 11 / Go 1.24.5 / Node 24.15.0），时点 = 提交 `3aa6233`。

| 项 | 事实 |
|---|---|
| 版本（两处一致，机器断言） | `0.1.0` —— `internal/policy.Version` + `tools/tasks.mjs` 的 `VERSION`；由 `TestVersionSingleSourceMatchesBuildScript`（G11）守 |
| 发布物自证 | `bin/mediatrace.exe --version` → `mediatrace 0.1.0`（与 tag `v0.1.0` 同名去掉 `v`） |
| 模块路径 | `github.com/yuluo554/10-27-mediatrace`（与仓库名一致，19 个 Go 文件 58 处 import 已同步） |
| 仓库 | `10-27-mediatrace`（与 `plan/00` §头部、`plan/01` §头部口径一致；同系列 `10-20-cfgdelta` 同形态） |
| 提交数 / 跟踪文件 | 10 个里程碑提交 / **63** 个跟踪文件 |
| 布局门 | `node tools/tasks.mjs gates` → **全绿**（11 条判据；状态副本 4 处 18 行一致） |
| 脱敏门 | 同上 → **0 命中**（扫 59 份文本，豁免 2 份 = 门自己） |
| 门自证（变异验证） | `node tools/tasks.mjs verify-gates` → **21/21 按预期**（11 布局判据 + 4 脱敏规则 + 基线/对照）；跑后 `git diff --stat` 无残留 |
| gitignore 三态 | `node tools/tasks.mjs check-ignore-rules` → 4 条线上判据全 `ok`（manifest 入库 / 样本·bin·.mimosa 不入库） |
| 全量测试 | `node tools/tasks.mjs all` → 全绿；`go vet` 干净；`go test ./...` **99 项 PASS / 0 FAIL**（13 个包） |
| 夹具对账 | `-check` → `92 例与入库 manifest 逐字节一致`（defects 32 / controls 16 / m3-regression 24 / m4 10 / m5 10，seed=20261027） |
| 跨平台编译 | `node tools/tasks.mjs cross` → linux/amd64 + darwin/arm64 + windows/amd64 **三产物全绿** |
| 构建可复现 | 同参数连编两次 `cmp` **逐字节相同**；二进制内嵌 `vcs.revision` + `vcs.modified=false`（`go version -m` 可核） |
| 冒烟（退出码） | `unknown command` → 3；`missing file` → 3；`k1_single_moov` 默认全集 → 0；C2/C5b/C6/C7 单模块 → 各 1 |
| 交付面清单 | `cmd/`、`internal/`、`tools/`、`data/`、`docs/`、`plan/`、`fixtures/out/manifest.json`、`.github/workflows/ci.yml`、`README.md`、`AGENTS.md`、`LICENSE`、`Makefile`、`go.mod` |
| 附件形态 | **构建产物本身**（单二进制 ×3 平台 + `SHA256SUMS`）；**不发源码包**——源码由 GitHub 的 tag 树提供 |
| 状态副本 | 三处（`plan/00 §五` / `plan/05 §一` / `README.md`）此刻仍为 M6 ⬜——按纪律，tag + Release + 留档三件未落地前不得写 ✅ |
| CI | **未首推**：仓库尚未创建。一切"CI 绿"必须写成"待首推验证"（`plan/05 §四`） |

### 1.1 脱敏复核（四条规则 + 人工过查，逐条留结论）

| # | 检查 | 命令 | 结论 |
|---|---|---|---|
| D1 | 脚本类文件名泄露 | `git ls-files \| grep -icE "\.env$\|\.key$\|secret\|token\|credential"` | **0** |
| D2 | 本机绝对路径（`<盘>:\\Users\\<名>`）| `git grep -nIE "[A-Za-z]:[\\\\/]Users[\\\\/]"` | 仅 2 处**占位符**写法 `C:\Users\<user>\…`（`AGENTS.md` / `plan/02`，属脱敏后的正确形态）；`tools/check-desensitized.mjs` 自身豁免 |
| D3 | 全历史 blob 同扫 | `git rev-list --all` × `git grep` | **0** 命中（未推送，全历史即本地 10 个提交） |
| D4 | 邮箱 / 手机号 | `git grep -nIE "1[3-9][0-9]{9}\|…@(qq\|163\|gmail\|outlook)\."` | 0 真命中；4 条为 `manifest.json` 的 sha256 长十六进制串（advisory 级，判据已定：**是功能指纹，不是泄露**） |
| D5 | 私网 / 回环地址 | `git grep -nIE "192\.168\.\|10\.\d+\.\|172\.(1[6-9]\|2[0-9]\|3[01])\."` | 0 命中 |
| D6 | 构建产物本体扫描 | `strings -a dist/* \| grep -iE "C:\\\\Users\|/home/\|<本机用户名>\|ProgramData"` | **三产物 0 命中**（`-trimpath` 生效；二进制内只有 `vcs.revision` 与模块路径） |
| D7 | git 提交元数据 | `git log --format='%ae %ce'` | 仅 `282769740+yuluo554@users.noreply.github.com`（GitHub noreply，作者自愿公开的账号坐标） |

### 1.2 发布前瞻：已经排除的"首推必红"

本次发布前发现并修掉一条**只在远端显形**的缺陷（提交 `3aa6233`）：

- **旧写法**：cross job 断言 `if command -v ffmpeg; then exit 1`。
- **问题**：`ubuntu-latest` **预装 ffmpeg** → 首推必红。它验的是 runner 环境事实，不是产品性质。
- **正解**：`PATH=/nonexistent` 下跑 `--version` 与 `audit --help`——运行期若 spawn 任何外部
  程序会立刻失败，这才证明"无运行时 ffmpeg 依赖"。"判定路径不 import `net`、不起子进程"
  仍由 `internal/archtest` G7/G8 常驻断言覆盖。

### 1.3 两处 tag 前修掉的定版缺陷（M6 定版提交）

| 提交 | 问题 | 处置 |
|---|---|---|
| `bebecbd` | `go.mod` 模块路径 `.../mediatrace` 与 `plan/00` 声明的仓库名 `10-27-mediatrace` 不一致；版本串仍是 `0.1.0-dev`，发布物无法自证 tag | 19 文件 58 处 import 同步改名；版本串收为 `0.1.0`（两处唯一定义同改） |
| `d9bd959` | "版本号两处一致"只靠人记，漂移时本地全绿 | 新增 `TestVersionSingleSourceMatchesBuildScript`（G11），并登记到 `plan/03 §七`；变异实测改成 `0.9.9` → 红并打印两处实际值，还原复绿 |

---

## 二、tag 后追加

> 本节在发布动作执行时逐行回填；未执行的项保持 ⬜（**不许把未执行写成已完成**）。
> 附件 sha256 属"tag 后追加"的合法位置：tag 树内写不了自己的哈希。

| 步骤 | 事实 | 状态 |
|---|---|---|
| 1. `commit`（tag 前） | `bebecbd`（模块路径 + 版本定版）→ `d9bd959`（G11）→ `3aa6233`（CI ffmpeg 断言）→ 本留档提交 | ✅ |
| 2. `git tag -a v0.1.0` | ⬜ 待打（在 §一 与状态副本落盘之后） | ⬜ |
| 3. 建 GitHub 远程仓库并首推 `main` | ⬜ 待执行 | ⬜ |
| 4. `push origin main --tags` | ⬜ 待执行 | ⬜ |
| 5. `SHA256SUMS` + `gh release create` | ⬜ 待执行（附件 = 三平台单二进制 + `SHA256SUMS`） | ⬜ |
| 6. 附件 sha256（三行，来自 Release 正文） | ⬜ 待回填 | ⬜ |
| 7. 真下载复核（匿名 clone + 下载件 sha256 比对 + 二进制 `--version`） | ⬜ 待执行 | ⬜ |
| 8. Actions run id 台账（`<sha> branch=<结论>` 逐行） | ⬜ 待首推后回填 | ⬜ |
| 9. `git diff --name-only v0.1.0..HEAD` 核对（只允许文档面） | ⬜ 待 tag 后核对 | ⬜ |

---

## 三、阶段 7 出口门覆盖对账

> 对齐方法论 `SKILL.md` §「阶段 7 出口门」逐条。三态：**实现**（有门/有命令 + 输出）/
> **不适用**（写具体理由，不许"暂不"/"后续"）/ **待首推**（无远端期如实标注）。
> 判别 = "出口门几十条，本项目盖了哪几条？剩下的谁判不适用的？"——答不出就是静默消失。

| 出口门 | 本项目形态 | 状态 |
|---|---|---|
| 发布物本体已验（真下载 + 中立目录） | `gh release download` → sha256 与 Release 正文比对 → 中立目录跑冻结二进制 `--version` / `audit` | 待首推 |
| 入仓工具在无 `.git` 树里也要能跑 | `git archive HEAD \| tar -x` 到 `.tmp/export`（无 `.git`）实跑：布局门全绿、脱敏门 0 命中、`check-ignore-rules` 退化为文本断言并**显式 SKIP**、`smoke` 全绿、`check-fixtures` 92 例一致 | 实现（本机实测） |
| `container.options: --network none` 与 job 容器互斥 | 本项目 CI 不用 job 容器、不跑"无网络 job"——零网络由 `internal/archtest` G7（AST 扫 `net`/`net/http` import）在**每次 `go test`** 上常驻断言 | 不适用（无容器化 CI；零网络另有常驻 AST 断言） |
| 首推第四色·配置面（`src` 布局 / 编码 / 零依赖断言 / 离线镜像） | 纯 Go、无 `src` 布局、无 `PYTHONPATH`；零依赖由 G8（`go.mod` 无 `require`、无 `go.sum`）常驻断言；控制台编码面：门的输出含中文，**Windows runner 上未经验证**（见下"已知未验证"） | 待首推 |
| 发布包实测 = README 每条命令在解包树真敲 | 在 `git archive HEAD` 解包树里真敲：`fixtures` → 92 例；`bin` → `0.1.0`；C2/C5b/C6/C7 四条 README 审计命令退出码分别 **1/1/1/1**，`k1` 默认全集 **0** | 实现（本机实测） |
| 干净环境验证要连真实输入一起验 | 本机无真实样本本体（`data/real-samples.json` 22 条全部 `verified: null`，样本不入库），故"真实输入"面**未验** | 不适用（首期只登记未复跑，`plan/04` T3） |
| "干净环境"第一分支 = 无本地数据缓存 | 无下载件、无外部数据目录；夹具全部由 `--seed` 程序化生成且产物不入库，`-check` 双跑逐字节一致 | 实现 |
| tag 树内留档与 tag 自身事实自指 → 留档第三节写成骨架 | §一 不写任何附件哈希；哈希只进 §二 与 Release 正文 | 实现 |
| 附件与 tag 树逐字节同源 | 附件 = 由 tag 树 `go build -trimpath` 直接产出（无打包步骤）；同参数双编 `cmp` 逐字节相同 | 待首推（tag 后复验） |
| 源码包要自足（`MANIFEST.in`） | 不发源码包（附件 = 构建产物）；源码由 tag 树提供 | 不适用（无 sdist 发布面） |
| 构建必须排在交付面定稿之后 | 顺序已遵守：交付面（模块路径/版本/CI）定稿 → 全量测试 → `cross` 构建 → 留档 → tag | 实现 |
| README 提到的附件名必须真在 Release 里 | README「下载即用」尚未写附件名；附件名在 §二 回填后同步进 README | 待首推 |
| 对外样例产物在中立目录生成 | 无样例报告附件（附件是二进制本身） | 不适用 |
| 附件口径（手工包 vs 构建产物 vs GitHub Source code） | 择一：**构建产物本身**（三平台单二进制 + `SHA256SUMS`）；GitHub 自动 Source code 包不参与复核（其 sha256 本地复算不出） | 实现（口径已定，待首推复核） |
| 附件 sha256 只写两个不动点 | §一（tag 树内）零哈希；哈希写 §二 + Release 正文 | 实现 |
| 版本号"四处一致"机器断言 | 两处定义（`policy.Version` / `tasks.mjs VERSION`）由 G11 机器断言；`--version` 输出与 tag 名一致由 §二 真下载复核；`docs/report-schema.md` 示例里的 `"version": "0.1.0"` 与本版同值（示例，非断言源） | 实现（G11）+ 待首推（产物面） |
| 已发布窗口关闭后的缺陷处置 | 窗口未开（尚无 tag）；本次三处定版缺陷在 tag **前**修完 | 实现（前置处置） |
| DoD 打勾只能在事实落地后 | `plan/05 §二` M6 五框此刻全空；`plan/00 §五`/`README` 仍 ⬜ | 实现（纪律遵守中） |
| 随包 README 状态行不承载"发布到哪一步" | README 状态表列的是**里程碑完成度**（M1–M6），不写"已发布到 PyPI/Release"这类进度 | 实现 |
| 定版改变指纹字段 → 样例同轮重生成 | 夹具指纹（sha256 + seed）与版本号无关；本次定版后 `-check` 仍 `92 例逐字节一致`（已实测） | 实现 |
| `POST_TAG` 处置 = 声明 + 可核对 | §二 会写"tag 后只有追加类提交"并给 `git diff --name-only v0.1.0..HEAD` 命令 | 待首推 |
| 发布门报红先看读 tag 树还是工作树 | 本项目无独立发布门脚本；两门读工作树（或解包树），故 `git archive` 后的树必须复跑（已实测全绿） | 实现 |
| annotated tag 两端一致 = 对象哈希一致 | 单端发布（无镜像），只需 `git cat-file -t v0.1.0` = `tag` + `git tag -v` 可验 | 待首推 |
| tag 之后只允许文档面提交 | 已约定 §二 回填只动 `plan/`、`README.md`；tag 后核对命令写进 §二 | 待首推 |
| push 数 = run 数对账 | §二 第 8 行用 `<sha> branch=<结论>` 逐行台账（sha 是既成事实，不写会自追的总数） | 待首推 |
| 四矩阵同点全红 = 接线缺陷 | test 矩阵三平台（ubuntu/windows/macos）；首推若三点同红则按接线排查 | 待首推 |
| 多平台冻结产物走 CI 原生构建 | 三平台二进制由本机 `cross`（纯 Go 可交叉编译，非 PyInstaller）；CI 也独立交叉编译三产物作对照 | 实现 |
| CI 内联脚本优先落 `tools/` 脚本 | 现有内联脚本均无 heredoc；断言逻辑尽量落 `tools/*.mjs`（`tasks.mjs` 已有 `check-ignore-rules`/`smoke`） | 实现 |
| 每条已交付命令都要有离线 CI 冒烟 | CI 覆盖：两门 + 变异验证 + gitignore 三态 + `go test`（含 99 项，覆盖五模块与四退出码）+ 夹具双跑 + 交叉编译 + `PATH=/nonexistent` 烟测 | 待首推（配置已就位） |
| "能力缺失"降级不许用 PATH 隔离伪造缺失 | 本项目无"能力探测"面；反向应用了同一条：`PATH=/nonexistent` 用来**证明不依赖外部程序**，不是伪造缺失 | 不适用（无该面）+ 反向实现 |
| tag push 与 branch push 各一条 run | `on: push: branches: [main]` + `pull_request` **带分支过滤**，故 tag push **不触发** run（与无 filters 的项目不同）；首推后以 `gh run list` 实测为准 | 待首推 |
| CI 检出形态使"标注 tag 对象"整类不存在 | 无依赖 tag 对象的检查（无提交元数据断言入 CI） | 不适用 |
| 推送前先本地解析 workflow 断言 job/step 数 | 本机无 `js-yaml`；改用结构性断言：4 个 job（layout/test/fixtures/cross）+ 13 个具名 step + 无 tab + step `name` 无裸 `: ` | 实现（近似形态，已留档口径） |
| 发布终态自检门可复跑（离线/`--online`/`--pre-tag`） | 无独立发布门脚本；等价面 = `node tools/tasks.mjs all` + `cross`，可随时复跑、不依赖网络 | 不适用（无独立门脚本；等价命令常驻可跑） |
| 产物必须包含"能核对自身"的判据 | **关键判据**：`tools/tasks.mjs all` 与两道门随仓库发布，可在解包树里复跑；已在 `git archive` 树内实测全绿（见本节第 2 行） | 实现（本机实测） |
| 离线模式报 SKIP 不报 PASS | `check-ignore-rules` 在无 `.git` 树里输出 `SKIP: 无 .git 树（发布包形态）——已退化为 .gitignore 文本断言，4 条线上判据未执行` | 实现 |
| 发布门自己是"读外部输入 + 起子进程"的程序，三处输入各自校验 | 两门读仓库内文件（不读 git 元数据，`LAYOUT.*` 判据落文件文本）；`tasks.mjs` 的 Go 探测有三级回退并显式报错 | 实现 |
| 门的结果会被"为满足它而写入的产物"改变 | 两门判据全部落**仓库内文件文本**，不读 git 元数据（浅克隆下 tag/历史深度会翻转） | 实现 |
| 门读"人写的文档"时样例行被当真值 | `LAYOUT.GAPS` 会先剥行内代码与文件名再匹配里程碑号（`plan/06 §B9` 同族，已实现） | 实现 |
| `SHA256SUMS` 有两种标准形态 | 用 `sha256sum` 原生两列形态（`<hash>  <file>`），随附件上传；README 给 `sha256sum -c` 复验命令 | 待首推 |
| 门要自己数"这轮扫到了几个" | 两门都打印扫描计数：布局门 `必备件 29 / 交接单 5 / 状态副本 4 / 状态行 18 / 文档点名工具 7`；脱敏门 `扫了 59 个文本文件，0 命中（豁免 2 份）` | 实现 |
| `gh` 是原生程序，`/tmp/...` 路径它不认 | 发布动作在仓库内跑；附件路径用相对路径 `dist/...`，不传 `/tmp` 绝对路径给 `gh` | 实现（约定） |
| 发布门落地分层（离线常驻 / 终态不进 CI） | 两门 + 变异验证**常驻 CI**（layout job）；"tag/Release 终态"项不进 CI，只在本文件登记 | 实现 |
| 发布门判据按"真实发布动作序列"设计 | 判据按发布动作顺序组织：门 → 测试 → 夹具 → 交叉编译 → 留档 → tag → Release → 真下载 | 实现 |

### 3.1 已知未验证（诚实口径，不写成已实现）

| # | 项 | 状态 | 原因 / 后续 |
|---|---|---|---|
| U1 | CI 实跑（任何 job） | ⬜ 未验 | 仓库尚未创建；首推后回填 run id 到 §二 第 8 行 |
| U2 | `windows-latest` runner 上门的四字节中文输出（代码页） | ⬜ 未验 | 本机 Windows 控制台输出正常，但 CI 控制台代码页可能不同；若首推红，处置 = 门输出加 `PYTHONUTF8` 等价物或改 ASCII 前缀（**本项目是 Node 门，Node 默认 UTF-8 输出，风险较低但未实测**） |
| U3 | 真实样本 22 条的复跑 | ⬜ 未验 | 样本本体不入库（`data/real-samples.json` 全部 `verified: null`）；`plan/04` T3 登记 |
| U4 | GPAC `-edits` 造 3 段 `elst` 真值 | ⬜ 未验 | 本机未装 GPAC（`plan/05 §五` U1）；夹具以程序化构造覆盖 |
| U5 | `mkvalidator` 能力边界 | ⬜ 未验 | 题面只写"存在"，不写能力边界（`plan/00` D6） |
| U6 | changelog | 未创建 | 首期 v0.1.0 为首次发布，无历史可比；后续版本再建 |
