# HANDOFF-M5 · 交给第 5 棒（M5 偏移与显示判定；模块 M4 + M5）

> **命名约定**：`plan/HANDOFF-M<k>.md` = **交给第 k 棒的入口交接单**（由第 k-1 棒收尾时写）。
> 第 k 棒收尾时在本文件补 DoD 结果，并**新建** `plan/HANDOFF-M<k+1>.md`。布局门 `LAYOUT.HANDOFF_ENTRY`
> 断言"当前棒必须有入口交接单（七节齐）"，`LAYOUT.HANDOFFS` 断言"已 ✅ 的里程碑有七节交接单"。
>
> 本文件由 **M4（时间戳判定）收尾**产生。M4 的产物是**模块 M3**（`internal/rules/timestamps`，
> 六个 RuleID），对 92 例夹具**缺陷召回精确、对照 60 例误报 0**。
> **M5 是本项目最后一个判定里程碑**：模块 M4（偏移/priming）+ 模块 M5（显示与色彩）+
> **报告三态与首屏三段收口 + 退出码**，随后只剩 M6 发布。

---

## 一、当前进度 + 证据

| 项 | 状态 | 证据（本机实测，2026-10-28） |
|---|---|---|
| M1 · 数据先行 | ✅ | 92 例夹具入库 manifest；C1–C7 每类 ≥1；同 seed 双跑逐字节一致 |
| M2 · 结构判定 | ✅ | 模块 M1（`internal/rules/structure`，八条规则）；92 例**集合相等** |
| M3 · 时长口径判定 | ✅ | 模块 M2（`internal/rules/duration`，六条规则）；对照组 60 例误报 0 |
| M4 · 时间戳判定 | ✅ | 模块 M3（`internal/rules/timestamps`，六条规则）；见 §六 DoD 结果 |
| `internal/rules/timestamps` | ✅ | `Audit(root, raw, fileName)`；六条 RuleID；DTS/PTS 分家、`tfdt` v0/v1 分宽 |
| `--modules M3` 接线 | ✅ | `bin/mediatrace audit fixtures/out/c5b_tfdt_regression.mp4 --modules M3` → exit 1 + `tfdt_regression`（fragment #1） |
| 模块 M3 对账 | ✅ | `cmd/mediatrace`：`TestM3MatchesFixtureTruth`（M3 相关 5 例命中 + 构造点偏移命中 + 对照组 **60 例 0 finding**） |
| `--modules` 子集一致（G9） | ✅ | `TestM3SubsetConclusionIsStable`：`--modules M3` == `--modules M1,M2,M3` 里的 M3 一半 |
| 两门 + 变异验证 | ✅ | `node tools/tasks.mjs gates`（11 判据 / 4 规则）；`verify-gates`（21 组逐组变红；跑后 `git diff --stat` 无残留） |
| 本地全套 | ✅ | `node tools/tasks.mjs all` 全绿 |
| **M5 实现** | ⬜ **未开工** | 本文件即其入口 |

提交：见 `git log --oneline`（末条为 `M4 · 时间戳判定 …`）。

---

## 二、当前事实 · 状态不变量（下一棒动手前必须知道，否则一定踩）

| # | 事实 | 后果 / 禁区 |
|---|---|---|
| F1 | `internal/rules/structure`（模块 M1）、`duration`（M2）、`timestamps`（M3）是**仅有的三个已实现模块**；`rules.go` 里 `m4`/`m5` 仍是桩，`Audit()` 返回 `ErrNotImplemented` → CLI 退出码 3 | **不要**把桩改成"返回空 findings"（冒充 PASS）。M5 **只有** `m4`/`m5` 两个转调要接（各建 `internal/rules/offset`、`internal/rules/display`），接完本项目五模块即全实现 |
| F2 | **M5 接完 `m4`/`m5` 后，默认全集不再是退出码 3** | ⚠️ **必须同提交改** `cmd/mediatrace/main_test.go`：`TestSkeletonModuleIsNotImplemented`（断言默认 `audit p` == 3）与 `TestFlagOrderDoesNotMatter`（`{audit p --json}` / `{audit --json p}` 期望 3）——改完后默认全集对"只有 ftyp"的样本应是 **pass(0)**。**否则 M5 收尾 `node tools/tasks.mjs test` 必红**。同时更新 `main.go` 里 "skeleton stage" 相关注释（保留 `rules.ErrNotImplemented` 定义无妨） |
| F3 | `internal/policy` 是口径常量的**唯一实现处**（`MatrixOrthoTolerance=1`、`Square()`、`TickTolerance`…）；`TestR3PolicyConstantsHaveSingleSource` 扫全仓 `const` | matrix 正交性判据**必须**用 `policy.MatrixOrthoTolerance` + `policy.Square`；**不得**在 `display` 包再写一份数字/浮点 |
| F4 | `go.mod` 的 `require` 必须为空、`go.sum` 必须不存在（`TestG8`）；`internal/`+`cmd/` 禁 `net*` import（`TestG7`） | 加任何三方依赖 / 遥测 / 在线检查 = 红 |
| F5 | 里程碑状态有 **3 处副本**（`plan/00 §五` 表 / `plan/05 §一` 表 / `README` 状态表），另有 `plan/00 §三` 台账行 | 翻状态必须**同一提交**改：三处副本 + 台账里标 `⬜ M5` 的行（`LAYOUT.LEDGER` 会红）。**注意**：台账行"priming… ⬜ M5"（本次 M4 收尾把误标的 `⬜ M4` 改成 `⬜ M5`）与"matrix… ⬜ M5"、"colr… ⬜ M5"、"每条 fail 带四元组… ⬜ M5"、"三态+gray… ⬜ M5"、"报告同源… ⬜ M5"、"退出码… ⬜ M5"、"确定性… ⬜ M5"、"纯本地零网络… ⬜ M5" 共 9 行待 M5 翻 ✅ |
| F6 | `fixtures/out/manifest.json` **必须入库**；`fixtures/out/*.mp4` 不入库（`.gitignore` 只写 `/out/`） | 改 `.gitignore` 的 `out/` 会连 manifest 一起忽略（`E4`） |
| F7 | `tools/verify-layout-gate.mjs` 会**临时改写仓库文件**，靠 `compose` 回滚 | 手工跑前先 `git status` 记基线；跑后 `git diff --stat` 应只剩自己的改动（`plan/06 §六 B2/B8`） |
| F8 | `tools/fixture-gen` **禁** `time`/`os.Getenv`/`os/exec`/`math/rand`（`TestGeneratorHasNoClockEnvOrExec`） | M5 **不得**改生成器引入时钟/进程。**M5 若真缺夹具**（见 F15）也要先想清楚——改生成器必须 `node tools/tasks.mjs fixtures` 重生成并 commit |
| F9 | `manifest.json` 现为 **92 例**（defects 32 / controls 16 / m3-regression 24 / m4 10 / m5 10）；`check-fixtures` 与 CI `diff -r` 拿它当真值 | 生成器不改，manifest 不动 |
| F10 | 文档里**不得**出现 `make <target>` | 可执行等价物一律 `node tools/tasks.mjs <target>` |
| F11 | 92 例的**载荷字节是盒子级占位**（`ffprobe` 报 H.264 解码错误属**预期**） | 只读盒子、不解码；别"修"样本 |
| F12 | **`isobmff.Parse` 是宽容的**：`size<8`、子盒越界、声明 size 越过文件、深度超限都记进 `root.Anomalies`，**不返回 error**；只有"读不出根盒头"才是 parse error（CLI 3） | M5 若发现"盒子越界"，那是 finding（或 M1 已报的），**不是** gray/退出码 3 |
| F13 | IR 已带 **`Box.Path`**（斜杠路径；**同名兄弟才带 `[i]`**，索引从 0 起）与 `Box.Offset/PayloadStart()`；`stsd` 不是容器盒——样本条目与 `pasp`/`colr`/`mdcv`/`clli` 在 IR 里**不在 `stsd` 下**（`stsd` 的 `IsContainer` 为 false） | 直接用它拼 `field_path`。**M5 读 `tkhd.matrix/width/height`（tkhd 是叶子盒，字段自读）+ `stsd` 内子盒（`pasp`/`colr`/`mdcv`/`clli` 要按字节自读 `stsd`）**——参考 M1 的 `sampleEntryTypes()` 读 `stsd` 子条目的写法（`internal/rules/structure/structure.go`）与 `gen/` 里 `stsd_*` 偏移构造 |
| F14 | `stts`/`ctts`/`tfdt`/`trun`/`stsd` 内子盒**都不是容器盒**，逐字段要用 `raw` + `PayloadStart()` 自读（带边界检查、`maxEntries` 收敛） | M3 的 `readSttsEntries/readTfdt/readTfhd/readTrunSum` 是可直接照搬的**只读小工具**（各包**复制**，不得 import 兄弟规则包——`TestModulesDoNotImportEachOther` 会红） |
| F15 | **模块 M4/M5 的若干规则/ gray 在夹具里没有对应样本**（如实登记，别当已覆盖）：`start_trim` 只有 `M4P4`；`no_edts_but_nonzero_start`（gray `no_edts`）**无夹具**；`hdr_metadata_missing`（gray）**无夹具**；`hdr_range_suspect` **无夹具**；`media_rate_not_one`（gray）**无夹具**（M2 已判 `elst_item_rate_not_one` fail） | 这几条要么在 `m4`/`m5` 或缺的组里补夹具（改生成器 = 重生成 manifest），要么写**包级单测**钉住判据与证据塑形（M3 的 `dts_regression` 就是这样处理的），并在 `docs/rules.md` 如实写明"无夹具，单测覆盖"。**不得**声称夹具已覆盖 |
| F16 | **同一根因合并**（M1/M2/M3 纪律）：一处根因只报一条 | M5 沿用：`priming_undeclared` 与 `start_trim` 别对同一根因连环报；`colr` 多实例只报一条 `colr_multi_instance`（但值里**按序全列**每个 colr） |
| F17 | **`elst` 按文件内顺序编号、不排序**（`K6`/D10）；`ctts` v1 偏移**有符号**；`colr` 多实例**按序全列不取第一个** | M4 的 `start_trim` 取 `elst[0].media_time`；M5 的 `colr` 多实例按文件内顺序全列 |
| F18 | **三态与闭集**：`gray.reason` 是**封闭集合**（`report.GrayReason`，8 个）；`no_roll_distance`/`unknown_codec_priming`/`no_edts`/`media_rate_not_one`/`hdr_metadata_missing` 已定义 | 直接复用 `report.GrayNoRollDistance`/`GrayUnknownCodecPriming`/`GrayNoEdts`/`GrayHDRMetadataMissing`，**不新增 reason**（要加就先改 `data/README §5.5` + `docs/report-schema.md`，`TestGrayReasonMustBeClosedSet` 会红） |
| F19 | **`start_trim`/`priming_*` 是 fail，`no_edts`/`no_roll_distance`/`unknown_codec_priming` 是 gray**（plan/03 §五）；M4 期望"未声明报 1024"（`C6`），"正确裁剪不报"（正反例各 5） | 期望 priming 值 = **声明值**（roll 距离 / `dOps.preSkip`），**不是**"必须 1024"（`data/README §2`：1024/2112/312 是观测值非常规条款） |
| F20 | **M4 的判定基准与 M2 已有重合面**：`elst[0].media_time>0` 在 M2 里被刻意排除（`first_pts_outside_media` 只在"跨度不一致"时报，见 `duration.go` 注释），**留给 M4 的 `start_trim`** | 不要因为 M4 判 `start_trim` 就去改 `duration` 包；也不要让 M4 重复 M2 的 `duration_mismatch` |
| F21 | **报告首屏三段**：`VerdictLine()` + `FirstLine()` 已实现（`internal/report`）；**`GATE` 段与 `--suite` 批跑模式未实现**（docs/report-schema §二 明确 GATE **只在 `--suite` 出现**） | M5 需定夺：实现 `--suite` + `GATE` 三数，或**明确记录"首期不做批跑、GATE 段不出现"**并同步 README/`plan/01 §七.3`。别留半拉子 |
| F22 | `main_test.go` 已有 G10（审计不改输入）、G5（退出码）、`TestParseInterspersed` 等；`report` 有 G1–G4 单测 | M5 新增 CLI 断言用 `run()`（收 `io.Writer`，`E8`），不要改回 `*os.File` |

---

## 三、下一棒待办（M5 · 偏移与显示判定 + 报告收口）

顺序即依赖顺序。**模块 M4 与模块 M5 可分别实现、分别对账，最后合并收口。**

### A. 模块 M4（`internal/rules/offset/`）

1. 建 `internal/rules/offset/`，与 M1/M2/M3 同构：`func Audit(root *isobmff.Box, raw []byte, fileName string) ([]report.Finding, error)`；
   在 `rules.go` 里把 `m4` 转调它（**不得**让 `offset` import `internal/rules` 或兄弟规则包）
2. 六条 RuleID（`plan/03 §五`）：
   | RuleID | 判什么 | 关键点 |
   |---|---|---|
   | `start_trim` | `elst[0].media_time > 0` | 报**秒值 + 样本数**；`values` 含 media_time(tick) 与 timescale |
   | `priming_undeclared` | 首帧 PTS 后移且**无**裁剪 edit | 报样本数（`C6`/`M4P1` **必须报 1024**；`M4P2` 报 2112） |
   | `priming_mismatch` | `sgpd`/`sbgp` roll 距离 ≠ 期望（`dOps.preSkip` 或声明值） | 报**声明 vs 期望**（`C6c`/`M4P3`/`M4P5`：roll 704/512 vs 1024） |
   | `no_edts_but_nonzero_start` (gray) | `edts` 缺失但首帧起点非 0 | `reason=no_edts`（**无夹具**，见 F15） |
   | `no_roll_distance` (gray) | 音频轨无 `sgpd`/`sbgp` roll 且**起点非 0**（起点为 0 不落 gray——`M4N5`/`C6b`… 见下） | `reason=no_roll_distance`；`C6b`（AAC 无 roll、起点 0）在当前真值里期望 gray `no_roll_distance`——**以 manifest 真值为准，动手前先核对** `C6b`/`M4N5` 的差异 |
   | `unknown_codec_priming` (gray) | 编码不在已知集（AAC/Opus） | `reason=unknown_codec_priming`（`P1`） |
3. 样本对表（真值在 manifest，别自己编）：`C6`→`priming_undeclared`；`C6b`(gray `no_roll_distance`)；`C6c`→`priming_mismatch`；`P1`(gray `unknown_codec_priming`)；`m4` 组 `M4P1..P5` fail / `M4N1..N5` pass；`controls` 里 `K6-priming`/`K13-opus`（正确裁剪）必须 **pass**
4. **对照组误报 0**：`controls`/`m3-regression`/`m4`/`m5` 共 60 例在模块 M4 下 **0 finding**（与 M1/M2/M3 同构的集合相等断言）

### B. 模块 M5（`internal/rules/display/`）

1. 建 `internal/rules/display/`（同构 + `rules.go` 转调 `m5`）
2. 六条 RuleID（`plan/03 §六`）：
   | RuleID | 判什么 | 关键点 |
   |---|---|---|
   | `matrix_not_orthogonal` | 行/列点积与 1.0 偏差超 `policy.MatrixOrthoTolerance`（**整数**，`policy.Square`） | `D1`（剪切 matrix）；`K12`/`M5N3`（单位矩阵）不报 |
   | `rotation_dimension_conflict` | ±90° 且 `tkhd.width<height` 且无 `pasp` | `C7`/`C7b`/`M5P1`/`M5P2`；`K7`（正常竖屏 + pasp）不报 |
   | `sar_dar_conflict` | `pasp.num/den` 与 DAR 不一致 | `M5P5`；`M5N5`（pasp 1:1）不报 |
   | `hdr_metadata_missing` (gray) | 广色域 `colr` 但无 `mdcv`/`clli` | `reason=hdr_metadata_missing`（**无夹具**，见 F15） |
   | `hdr_range_suspect` | `max<=min` 或 `clli`/`mdcv` 矛盾 | **无夹具**（F15）；只判量程，不判绝对亮度 |
   | `colr_multi_instance` | 多 `colr` | **按序全列**（`D2`：bt709→bt2020 两条），不取第一个 |
3. 样本对表：`C7`/`C7b`→`rotation_dimension_conflict`；`D1`→`matrix_not_orthogonal`；`D2`→`colr_multi_instance`；`m5` 组 `M5P1..P5` fail / `M5N1..N5` pass；`controls` 的 `K7`/`K8`/`K11`/`K12` 必须 **pass**
4. **对照组误报 0**：四组 60 例在模块 M5 下 **0 finding**

### C. 报告收口 + 全线收尾

5. **报告三态与首屏三段 + 退出码**（M5 里程碑内容）：`VerdictLine`/`FirstLine` 已在；定夺 `GATE`/`--suite`（F21）；`docs/report-schema.md` 去"骨架期草稿"标注
6. **CLI 接线**：`--modules M4` / `--modules M5` 走通并给真实结论；`--json` 仍符合 `docs/report-schema.md`
7. **改 `main_test.go`**（F2）：默认全集期望从 3 改成真实结论
8. **文档同步**：`docs/rules.md` 的 **M4 节 + M5 节**写全（反例指向样本 id）、`README.md`、`docs/fixture-pools.md`
9. **M5 收尾**（顺序固定）：DoD 全验 → 三处状态副本 + 台账行（9 行）同提交翻 ✅ → **新建 `plan/HANDOFF-M6.md`**（发布棒）→ 在**本文件**补 DoD 结果 → commit

> ⚠️ **M5 不碰**：`internal/rules/structure`、`internal/rules/duration`、`internal/rules/timestamps`（除非发现它们真错——那要同时说清并重跑对应模块的整夹具对账）；**不改**夹具真值（除 F15 决定的补夹具，须重生成 manifest 并 commit）。

---

## 四、既定口径清单（K 编号 · 动了会打挂测试）

| # | 约定 | 落在哪 | 违反时谁会红 |
|---|---|---|---|
| **K1** | 数值口径只在 `data/README §5` 定义、只在 `internal/policy` 实现 | `internal/policy/policy.go` | `archtest` R3 |
| **K2** | 三态 `pass/fail/gray`；`gray` 永不折算 pass、永不升级 fail | `internal/report` | G1/G2/G3 单测 |
| **K3** | `gray.reason` 是**封闭集合**；新增先改 `data/README §5.5` + `docs/report-schema.md` | 同上 | `TestGrayReasonMustBeClosedSet` |
| **K4** | 解析失败是 **error（退出码 3）**，不是 `gray`；**内容级异常**是 finding（F12） | `cmd/mediatrace` / `isobmff` | `TestAuditUnparsableFileIsUsageError` |
| **K5** | 每条 `fail` 必须四元组齐（offset / field_path / values / delta） | `internal/report` | `ValidateAll` 拒绝 + G1 |
| **K6** | `elst` 按**文件内顺序**编号、**不排序**；`colr` 多实例同理（D10） | M2/M4/M5 | 条目号写入 `field_path` 与 `values` |
| **K7** | 模块之间**不得**互相 import；规则是纯函数（无 I/O / 时钟 / 随机 / 网络） | `internal/rules` | `TestModulesDoNotImportEachOther` |
| **K8** | 生成器与解析器**不共享代码**（只共享 `policy` 常量） | `tools/fixture-gen` | 代码评审；`--check` 不证此事 |
| **K9** | 同输入双跑**逐字节一致**（含 `--modules` 子集） | 全链 | `check-fixtures` / CI `diff -r` |
| **K10** | 文档里的命令必须是**能跑的形态**（`node tools/tasks.mjs …`） | `plan/` `docs/` `README` | 部署期人工核对 |
| **K11** | 时长三口径互比前先对齐到公共时基（整数向下取整，**无浮点**）；公共时基 = **`mvhd.timescale`**（D12） | `internal/rules/duration` | `data/README §5.1`；浮点破双跑一致 |
| **K12** | `mvhd.duration==0` 推不出预期时长时落 `duration_unknown` gray，**不猜** | 同上 | §5.5 闭集 |
| **K13** | **DTS 判据只有一条**：`dts[n] < dts[n-1]`（容差 0）。**PTS 不参与单调判定**（B 帧天然乱序）；`ctts` v1 必须按**有符号**读。**位宽分宽**：`stts` 32 / `tfdt` v0 32 / v1 64 | M3（已实现） | §5.4；`m3-regression` 24 例误报 0 |
| **K14** | **priming 期望值 = 声明值**（roll 距离 / `dOps.preSkip`）；**不判"必须 1024"**——1024/2112/312 是**观测值非规范条款** | M4（本棒） | `data/README §2` 末段；`C6` 报 1024、`M4P2` 报 2112 |
| **K15** | **matrix 全程 16.16 定点整数**（`policy.Square`），正交容差 `policy.MatrixOrthoTolerance`；**不做浮点近似** | M5 里程碑 · 模块 M5 | §5.3；双跑一致 |
| **K16** | **priming/offset 与时长是两码事**：`elst` 起点裁剪（`start_trim`）归 M4；三口径时长互比归 M2——**同一根因不连环报** | M2/M4 | `duration.go` 注释 + F20 |

---

## 五、本机环境坑（只写实测过的）

| # | 坑 | 实测表现 | 处置 |
|---|---|---|---|
| E1 | **Go 不在 `PATH`** | `go: command not found` | 实际在 `%USERPROFILE%\go-sdk\go\bin\go.exe`；`tasks.mjs` 自动探测；也可 `GO=… node tools/tasks.mjs test` |
| E2 | **本机无 `make`** | `make: command not found` | 一律 `node tools/tasks.mjs <target>` |
| E3 | **Windows 可执行必须 `.exe`** | `spawnSync bin/mediatrace` = ENOENT（文件其实在） | `tasks.mjs` 按平台拼名；也可直接 `bin/mediatrace.exe` |
| E4 | **`git check-ignore` 的目录语义** | 父目录被忽略后，其内文件的 `!` 例外**不生效** | 忽略顶层产物用 `/out/`；常驻断言 `check-ignore-rules` |
| E5 | **`flag` 包遇位置参数即停** | `audit <file> --json` 曾直接 exit 3 | 已用 `parseInterspersed` 修；改参数面时**先跑** `TestFlagOrderDoesNotMatter` |
| E6 | **变异脚本会临时改仓库文件** | 中途 `Ctrl-C` 可能残留 | 跑前 `git status` 记基线；跑后核对 `git diff --stat`（本次 M4 实测无残留） |
| E7 | **C7 的载荷是占位字节** | `ffprobe` 报 H.264 解码错误 | 预期（F11）：只读盒子；判结构/口径不判编码 |
| E8 | **`cmd/mediatrace` 的 `run()` 收 `io.Writer`** | 便于测试捕获输出 | 新增 CLI 断言用 `bytes.Buffer` 即可，不要改回 `*os.File` |
| E9 | **夹具的 `stts` 是逐样本 `floor(duration/frames)` 拼的** | `Σstts` 与 `mdhd.duration` 可差几个 tick | 判"时长"用声明值；判"时间戳序列"用 `stts` 本身 |
| E10 | **`m4`/`m5` 组里有多例复用同一构造器**（如 `M4P1`==`C6`、`M4P5`==`C6c`、`M5P1`==`C7`） | 同一字节会被两个 case_id 覆盖 | 对账时按 `case_id` 逐一断言，别按"文件名唯一"去重 |

---

## 六、DoD 逐项 checklist（M5，取自 `plan/00 §五` 与 `plan/05 §二`）

- [ ] `internal/rules/offset/` 建成，模块 M4 六条 RuleID 全实现；`--modules M4` 走通
- [ ] `internal/rules/display/` 建成，模块 M5 六条 RuleID 全实现；`--modules M5` 走通
- [ ] `C6`/`M4P1` 报出 **1024**；`M4P2` 报 2112；`M4P3`/`M4P5`/`C6c` 报 roll 声明 vs 期望
- [ ] `M4P4` 落 `start_trim`（报秒值 + 样本数）；`C7`/`C7b`/`M5P1`/`M5P2` 落 `rotation_dimension_conflict`
- [ ] `D1` 落 `matrix_not_orthogonal`（**整数**运算）；`D2`/`M5P4` 落 `colr_multi_instance`（按序全列）；`M5P5` 落 `sar_dar_conflict`
- [ ] `P1` 落 `unknown_codec_priming` gray；F15 各"无夹具"分支有**包级单测**或**补夹具**，并在 `docs/rules.md` 如实写明
- [ ] **`m4`/`m5` 正反例各 5 + `controls`/`m3-regression`**：四组 60 例在 M4、M5 两模块下**误报 0**（集合相等断言）
- [ ] 每条 `fail` 带四元组（偏移 + 字段路径 + 值 + 差值）；`gray` 可导出且不计 fail
- [ ] `--modules M4`/`M5` 单独跑通且结论 == 合集切片（G9）
- [ ] **`main_test.go` 默认全集断言已改**（F2）——否则 `test` 红
- [ ] `docs/rules.md` M4 节 + M5 节写完；`docs/report-schema.md` 去草稿标注；`GATE`/`--suite` 有定夺（F21）
- [ ] 三处状态副本 + `plan/00 §三` 台账行（9 行）同提交翻 ✅ + `plan/HANDOFF-M6.md` 已写
- [ ] `node tools/tasks.mjs all` 全绿（含 smoke / cross 前的本地全套）

---

## 七、关键命令速查

```bash
# 门（新对话第一件事）
node tools/tasks.mjs gates              # 布局门 + 脱敏门
node tools/tasks.mjs verify-gates       # 两门的变异验证（21 组，逐组必须按预期变红）

# 夹具（M1 产物，本棒只消费、不改真值）
node tools/tasks.mjs fixtures           # 生成 fixtures/out + manifest.json（92 例）
node tools/tasks.mjs check-fixtures     # 双跑逐字节 + 与入库 manifest 比对
go run ./tools/fixture-gen -list        # 只打印 case id / sha256 / size

# 代码
node tools/tasks.mjs test               # go vet + go test ./...（含整夹具集对账）
node tools/tasks.mjs bin                # 构建 bin/mediatrace（Win: .exe）
node tools/tasks.mjs all                # 一键本地等价校验

# 单文件审计（M4/M5 未实现前仍退出码 3）
bin/mediatrace audit fixtures/out/c6_priming_undeclared_1024.mp4 --json --modules M4
bin/mediatrace audit fixtures/out/c7_rot90_dim_conflict.mp4       --json --modules M5
bin/mediatrace audit fixtures/out/c5b_tfdt_regression.mp4         --json --modules M3
```

**口径只查** `data/README.md §5`；**计划状态只查** `plan/00 §五`；**决策只查** `plan/00 §二`。