# HANDOFF-M6 · 交给第 6 棒（M6 发布：脱敏复核 + 公开仓库发布）

> **命名约定**：`plan/HANDOFF-M<k>.md` = **交给第 k 棒的入口交接单**（由第 k-1 棒收尾时写）。
> 第 k 棒收尾时在本文件补 DoD 结果；若还有下一棒则**新建** `plan/HANDOFF-M<k+1>.md`。
> 布局门 `LAYOUT.HANDOFF_ENTRY` 断言"当前棒必须有入口交接单（七节齐）"，
> `LAYOUT.HANDOFFS` 断言"已 ✅ 的里程碑有七节交接单"。
>
> 本文件由 **M5（偏移与显示判定 + 报告收口）收尾**产生。M5 的产物是**模块 M4**
> （`internal/rules/offset`）+ **模块 M5**（`internal/rules/display`），对 92 例夹具
> **召回精确、误报门每模块 83 例 0 finding**。至此**五个判定模块全部实现**，只剩发布。

---

## 一、当前进度 + 证据

| 项 | 状态 | 证据（本机实测，2026-10-28） |
|---|---|---|
| M1 · 数据先行 | ✅ | 92 例夹具入库 manifest；C1–C7 每类 ≥1；同 seed 双跑逐字节一致 |
| M2 · 结构判定 | ✅ | 模块 M1（`internal/rules/structure`，八条规则）；92 例**集合相等** |
| M3 · 时长口径判定 | ✅ | 模块 M2（`internal/rules/duration`，六条规则）；误报门 0 |
| M4 · 时间戳判定 | ✅ | 模块 M3（`internal/rules/timestamps`，六条规则）；误报门 0 |
| M5 · 偏移与显示判定 | ✅ | 模块 M4（`internal/rules/offset`）+ 模块 M5（`internal/rules/display`）；见 §六 |
| `--modules M4` 接线 | ✅ | `bin/mediatrace audit fixtures/out/c6_priming_undeclared_1024.mp4 --modules M4` → exit 1 + `priming_undeclared`（delta=1024 sample） |
| `--modules M5` 接线 | ✅ | `bin/mediatrace audit fixtures/out/c7_rot90_dim_conflict.mp4 --modules M5` → exit 1 + `rotation_dimension_conflict` |
| 默认全集 | ✅ | `--modules` 缺省 = M1–M5（默认全集不再退回退出码 3）；`main_test.go` 已改（`TestDefaultAuditRunsAllModules`） |
| 两门 + 变异验证 | ✅ | `node tools/tasks.mjs gates`；`verify-gates`（21 组逐组变红；跑后 `git diff --stat` 无残留） |
| 本地全套 | ✅ | `node tools/tasks.mjs all` 全绿 |
| M6 发布 | ⬜ **未开工** | 本文件即其入口 |

提交：见 `git log --oneline`（末条为 `M5 · 偏移与显示判定 …`）。

---

## 二、当前事实 · 状态不变量（下一棒动手前必须知道，否则一定踩）

| # | 事实 | 后果 / 禁区 |
|---|---|---|
| F1 | **五个判定模块全部实现**：`structure`/`duration`/`timestamps`/`offset`/`display`，`rules.go` 里 `m1`–`m5` 全部转调；`rules.ErrNotImplemented` 仅作防御保留（已无模块返回它） | **不要再把桩加回来**；不要把 `ErrNotImplemented` 改成"返回空 findings" |
| F2 | **默认全集**（无 `--modules`）= M1–M5，对最小 ftyp-only 样本给 **pass(0)** | `main_test.go` 的 `TestFlagOrderDoesNotMatter`/`TestDefaultAuditRunsAllModules` 已锁定；改 CLI 参数面先跑它们 |
| F3 | `internal/policy` 是口径常量**唯一实现处**（`MatrixOrthoTolerance`/`Square`/`TickTolerance`/`DTSRegressionTolerance`） | `TestR3PolicyConstantsHaveSingleSource` 扫全仓 `const`；**不得**在 rules 包再写一份数字 |
| F4 | `go.mod` 的 `require` 必须为空、`go.sum` 必须不存在（`TestG8`）；`internal/`+`cmd/` 禁 `net*` import（`TestG7`） | 加任何三方依赖 / 遥测 / 在线检查 = 红 |
| F5 | 里程碑状态有 **3 处副本**（`plan/00 §五` 表 / `plan/05 §一` 表 / `README` 状态表）+ `plan/00 §三` 台账行 | 翻状态必须**同一提交**改；`LAYOUT.STATUS`/`LAYOUT.LEDGER` 会红。M5 已全部 ✅ |
| F6 | `fixtures/out/manifest.json` **必须入库**；`fixtures/out/*.mp4` 不入库（`.gitignore` 只写 `/out/`） | 改 `.gitignore` 的 `out/` 会连 manifest 一起忽略 |
| F7 | `tools/verify-layout-gate.mjs` 会**临时改写仓库文件**，靠 `compose` 回滚 | 手工跑前先 `git status` 记基线；跑后 `git diff --stat` 应只剩自己的改动 |
| F8 | `fixtures/out/manifest.json` 现为 **92 例**（defects 32 / controls 16 / m3-regression 24 / m4 10 / m5 10）；`check-fixtures` 与 CI `diff -r` 拿它当真值 | **M5 已修正三处真值冲突并重生成 manifest**（见 §三 末"M5 修正记录"）；发布棒**不要再动夹具** |
| F9 | 文档里**不得**出现 `make <target>` | 可执行等价物一律 `node tools/tasks.mjs <target>` |
| F10 | 92 例的**载荷字节是盒子级占位**（`ffprobe` 报 H.264 解码错误属**预期**） | 只读盒子、不解码；别"修"样本 |
| F11 | **首期不做 `--suite` 批跑**：首屏只有 `VERDICT` + `FIRST`，`GATE` 段**不出现**（`docs/report-schema.md §二`，M5 定夺） | README / report-schema / plan 均已同步；发布说明不要写"GATE 三数" |
| F12 | `plan/RELEASE-M6.md` **尚未创建**（`LAYOUT.RELEASE` 在 M6 未 ✅ 前不要求） | 发布棒要**在打 tag 之前**创建它并写"tag 前事实"栏 |
| F13 | **未接远端**：CI 工作流（`.github/workflows/ci.yml`）已入库但**未首推**，一切"CI 绿"必须写成"待首推验证" | 发布后首推才可写 CI 绿；否则保留"待首推"措辞 |
| F14 | 真实样本登记 22 条全部 `verified: null`（`data/real-samples.json`）；样本本体不入库 | 发布说明不得声称"真实样本已复核"；只写登记事实 |

---

## 三、下一棒待办（M6 · 发布）

顺序即依赖顺序。

1. **脱敏复核**：`node tools/tasks.mjs gates`（布局门 + 脱敏门）全绿；`node tools/check-desensitized.mjs`
   零命中（本机路径 / 用户名 / 回环代理 4 条规则）。发布前再人工过一遍 `git grep` 绝对路径。
2. **全量校验**：`node tools/tasks.mjs all`（gates + verify-gates + check-ignore-rules + test +
   check-fixtures + smoke）全绿；`node tools/tasks.mjs cross` 三平台交叉编译全绿（发布二进制要跨平台）。
3. **发布留档 `plan/RELEASE-M6.md`**（**tag 之前**落盘，两栏）：
   - 栏一"tag 时点事实"：`git rev-parse HEAD`、`node tools/tasks.mjs all` 输出摘要、
     `bin/mediatrace --version`、产物名与平台、`SHA256SUMS` 的生成命令；
   - 栏二"tag 后追加事实"：Release 附件 sha256、真下载复核（匿名 clone + 二进制跑 `--version`）。
   - **不得留占位**（`LAYOUT.RELEASE` 在 M6 ✅ 后要求两栏齐、无占位）。
4. **`v0.1.0` annotated tag**（tag 之后**不再改代码面**；如需改，先删 tag 重打）。
5. **GitHub Release**：附产物 = 单二进制（按 `cross` 产物命名）+ `SHA256SUMS`。
6. **公开仓库可匿名克隆**：用未认证的 `git clone`（或 `git archive` tarball 形态）复核。
7. **状态收尾**：`plan/00 §五`/`plan/05 §一`/`README` 三处 + `plan/00 §三` 台账（M6 行）同提交翻 ✅；
   在本文件补 DoD 结果；commit。

### M5 修正记录（发布棒需要知道的三处真值修正）

M5 收尾时发现并修正了三处**互相矛盾/不可判**的夹具真值（改 `tools/fixture-gen/gen/` 后
`node tools/tasks.mjs fixtures` 重生成 manifest）：

1. **`C6b` 与 `M4N5` 曾字节完全相同**（都"无 roll、无 ctts、无 edts"），但真值一个 gray、一个 pass。
   已给 `C6b` 加"编辑表裁剪 1024 但无 roll"→ 干净的 `no_roll_distance` gray；`M4N5` 保持 pass。
2. **`C6c`/`M4P5` 的 roll 由 704 改为 1152**：旧值使"裁剪(1024) > roll(704)"与 `M4P4`（裁剪 2048 > roll 1024）
   同属"裁剪 > roll"一档，无法区分 `start_trim` 与 `priming_mismatch`。改为 roll(1152) > 裁剪(1024) 后，
   **裁剪 > roll ⇒ `start_trim`；裁剪 < roll ⇒ `priming_mismatch`** 成为可判的单一判据。
3. **`M5P5` 的存储宽高由 720×480 改为 640×480**（pasp 仍 8:9）：旧值使 `DAR=720*8/(480*9)=4:3` 与 tkhd 一致，
   根本不该报；改为 640×480 后 `DAR=640*8/(480*9)≠4:3`，`sar_dar_conflict` 才成立。

另：新增 `movie.go` 的 `ctts_entry0_offset` 标记，使 `priming_undeclared` 的证据偏移精确落在
`ctts[0].sample_offset` 字段上。

---

## 四、既定口径清单（K 编号 · 动了会打挂测试）

| # | 约定 | 落在哪 | 违反时谁会红 |
|---|---|---|---|
| **K1** | 数值口径只在 `data/README §5` 定义、只在 `internal/policy` 实现 | `internal/policy/policy.go` | `archtest` R3 |
| **K2** | 三态 `pass/fail/gray`；`gray` 永不折算 pass、永不升级 fail | `internal/report` | G1/G2/G3 单测 |
| **K3** | `gray.reason` 是**封闭集合**；新增先改 `data/README §5.5` + `docs/report-schema.md` | 同上 | `TestGrayReasonMustBeClosedSet` |
| **K4** | 解析失败是 **error（退出码 3）**，不是 `gray`；**内容级异常**是 finding | `cmd/mediatrace` / `isobmff` | `TestAuditUnparsableFileIsUsageError` |
| **K5** | 每条 `fail` 必须四元组齐（offset / field_path / values / delta） | `internal/report` | `ValidateAll` 拒绝 + G1 |
| **K6** | `elst` / `colr` 多实例按**文件内顺序**编号、**不排序**、**全列不取第一** | M2/M4/M5 | 条目号写入 `field_path` 与 `values` |
| **K7** | 模块之间**不得**互相 import；规则是纯函数（无 I/O / 时钟 / 随机 / 网络） | `internal/rules` | `TestModulesDoNotImportEachOther` |
| **K8** | 生成器与解析器**不共享代码**（只共享 `policy` 常量） | `tools/fixture-gen` | 代码评审 |
| **K9** | 同输入双跑**逐字节一致**（含 `--modules` 子集） | 全链 | `check-fixtures` / CI `diff -r` |
| **K10** | 文档里的命令必须是**能跑的形态**（`node tools/tasks.mjs …`） | `plan/` `docs/` `README` | 部署期人工核对 |
| **K11** | 时长三口径互比前先对齐到公共时基（整数向下取整，**无浮点**）；公共时基 = **`mvhd.timescale`** | `internal/rules/duration` | `data/README §5.1` |
| **K12** | `mvhd.duration==0` 推不出预期时长时落 `duration_unknown` gray，**不猜** | 同上 | §5.5 闭集 |
| **K13** | **DTS 判据只有一条**：`dts[n] < dts[n-1]`（容差 0）。**PTS 不参与单调判定**；`ctts` v1 按**有符号**读；位宽分宽 | M3（已实现） | §5.4；`m3-regression` 24 例误报 0 |
| **K14** | **priming 期望值 = 声明值**（roll 距离 / `dOps.preSkip`）；**不判"必须 1024"** | M4（已实现） | `data/README §2` 末段；`C6` 报 1024、`M4P2` 报 2112 |
| **K15** | **matrix 全程 16.16 定点整数**（`policy.Square`），正交容差 `policy.MatrixOrthoTolerance`；**不做浮点近似** | M5（已实现） | §5.3；双跑一致 |
| **K16** | **priming/offset 与时长是两码事**：`elst` 起点裁剪归 M4；三口径时长互比归 M2——同一根因不连环报 | M2/M4 | `duration.go` 注释 |

---

## 五、本机环境坑（只写实测过的）

| # | 坑 | 实测表现 | 处置 |
|---|---|---|---|
| E1 | **Go 不在 `PATH`** | `go: command not found` | 实际在 `%USERPROFILE%\go-sdk\go\bin\go.exe`；`tasks.mjs` 自动探测；也可 `GO=… node tools/tasks.mjs test` |
| E2 | **本机无 `make`** | `make: command not found` | 一律 `node tools/tasks.mjs <target>` |
| E3 | **Windows 可执行必须 `.exe`** | `spawnSync bin/mediatrace` = ENOENT（文件其实在） | `tasks.mjs` 按平台拼名；也可直接 `bin/mediatrace.exe` |
| E4 | **`git check-ignore` 的目录语义** | 父目录被忽略后，其内文件的 `!` 例外**不生效** | 忽略顶层产物用 `/out/`；常驻断言 `check-ignore-rules` |
| E5 | **`flag` 包遇位置参数即停** | `audit <file> --json` 曾直接 exit 3 | 已用 `parseInterspersed` 修；改参数面时**先跑** `TestFlagOrderDoesNotMatter` |
| E6 | **变异脚本会临时改仓库文件** | 中途 `Ctrl-C` 可能残留 | 跑前 `git status` 记基线；跑后核对 `git diff --stat` |
| E7 | **C7 的载荷是占位字节** | `ffprobe` 报 H.264 解码错误 | 预期（F10）：只读盒子；判结构/口径不判编码 |
| E8 | **`cmd/mediatrace` 的 `run()` 收 `io.Writer`** | 便于测试捕获输出 | 新增 CLI 断言用 `bytes.Buffer` 即可，不要改回 `*os.File` |
| E9 | **本机未接远端** | `git remote -v` 为空（装订期实测） | 发布棒需先 `git remote add` 并首推；CI 在此之前一律"待首推验证" |

---

## 六、DoD 逐项 checklist（M6，取自 `plan/00 §五` 与 `plan/05 §二`）

- [ ] `node tools/tasks.mjs gates` 全绿（布局门 + 脱敏门 0 命中）
- [ ] `node tools/tasks.mjs all` 全绿（含 `verify-gates` 21 组、`check-fixtures`、smoke）
- [ ] `node tools/tasks.mjs cross` 三平台交叉编译全绿
- [ ] `plan/RELEASE-M6.md` 在 tag **之前**落盘（两栏、无占位）
- [ ] `v0.1.0` annotated tag 打好；tag 后未再改代码面
- [ ] GitHub Release 附 `mediatrace` 单二进制 + `SHA256SUMS`
- [ ] 公开仓库可匿名克隆（未认证 clone 或 `git archive` tarball 复核）
- [ ] 三处状态副本 + `plan/00 §三` 台账（M6 行）同提交翻 ✅
- [ ] 本文件补 DoD 结果 + commit

---

## 七、关键命令速查

```bash
# 门（新对话第一件事）
node tools/tasks.mjs gates              # 布局门 + 脱敏门
node tools/tasks.mjs verify-gates       # 两门的变异验证（21 组）

# 全套 + 交叉编译
node tools/tasks.mjs all                # gates + verify-gates + check-ignore-rules + test + check-fixtures + smoke
node tools/tasks.mjs test               # go vet + go test ./...（含五模块对账）
node tools/tasks.mjs cross              # 三平台交叉编译
node tools/tasks.mjs bin                # 构建 bin/mediatrace（Win: .exe）

# 夹具（M1 产物，发布棒只消费、不改）
node tools/tasks.mjs fixtures           # 生成 fixtures/out + manifest.json（92 例）
node tools/tasks.mjs check-fixtures     # 双跑逐字节 + 与入库 manifest 比对

# 单文件审计（五模块全实现）
bin/mediatrace --version
bin/mediatrace audit fixtures/out/c6_priming_undeclared_1024.mp4 --json --modules M4
bin/mediatrace audit fixtures/out/c7_rot90_dim_conflict.mp4      --json --modules M5
bin/mediatrace audit fixtures/out/k1_single_moov.mp4             --json   # 默认全集 → pass(0)
```

**口径只查** `data/README.md §5`；**计划状态只查** `plan/00 §五`；**决策只查** `plan/00 §二`。
