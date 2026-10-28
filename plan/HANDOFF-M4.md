# HANDOFF-M4 · 交给第 4 棒（M4 时间戳判定）

> **命名约定**：`plan/HANDOFF-M<k>.md` = **交给第 k 棒的入口交接单**（由第 k-1 棒收尾时写）。
> 第 k 棒收尾时在本文件补 DoD 结果，并**新建** `HANDOFF-M<k+1>.md`。布局门 `LAYOUT.HANDOFF_ENTRY`
> 断言"当前棒必须有入口交接单（七节齐）"，`LAYOUT.HANDOFFS` 断言"已 ✅ 的里程碑有七节交接单"。
>
> 本文件由 **M3（时长口径判定）收尾**产生。M3 的产物是**模块 M2 的判定器**
> （`internal/rules/duration`，六条 RuleID），对 92 例夹具**缺陷召回精确、对照误报 0**。
> M4 是第一次判"**时间戳序列本身**是否自洽"——把只读盒子树用起来，读 `stts`/`ctts`/`tfdt`。

---

## 一、当前进度 + 证据

| 项 | 状态 | 证据（本机实测，2026-10-28） |
|---|---|---|
| M1 · 数据先行 | ✅ | 92 例夹具入库 manifest；C1–C7 每类 ≥1；同 seed 双跑逐字节一致 |
| M2 · 结构判定 | ✅ | 模块 M1（`internal/rules/structure`，八条规则）；92 例**集合相等** |
| M3 · 时长口径判定 | ✅ | 模块 M2（`internal/rules/duration`，六条规则）；见 §六 DoD 结果 |
| `internal/rules/duration` | ✅ | `Audit(root, raw, fileName)`；六条 RuleID；公共时基 `mvhd.timescale`（D12） |
| `--modules M2` 接线 | ✅ | `bin/mediatrace audit fixtures/out/c4a_duration_2x_mvhd.mp4 --modules M2` → exit 1 + 四元组；`e1…` 报条目号 1 |
| 模块 M2 对账 | ✅ | `cmd/mediatrace` 的 `TestM2MatchesFixtureTruth`：M2 相关缺陷 8 例命中、对照组 **60 例 0 finding** |
| `--modules` 子集一致（G9） | ✅ | `TestM2SubsetConclusionIsStable`：`--modules M2` == `--modules M1,M2` 里的 M2 一半 |
| 两门 + 变异验证 | ✅ | `node tools/tasks.mjs gates`（11 判据 / 4 规则）；`verify-gates`（21 组逐组变红） |
| 本地全套 | ✅ | `node tools/tasks.mjs test` 全绿（含整夹具集对账） |
| **M4 实现** | ✅ **已收尾** | 模块 M3（`internal/rules/timestamps`，六条 RuleID）；见 §六 DoD 结果 |
| `internal/rules/timestamps` | ✅ | `Audit(root, raw, fileName)`；六条 RuleID；`ctts` 分宽、`tfdt` v0/v1 分宽；DTS/PTS 分家 |
| `--modules M3` 接线 | ✅ | `bin/mediatrace audit fixtures/out/c5b_tfdt_regression.mp4 --modules M3` → exit 1 + `tfdt_regression`（fragment #1，偏移 0x5E8） |
| 模块 M3 对账 | ✅ | `cmd/mediatrace`：`TestM3MatchesFixtureTruth`（`C5`/`C5b`/`C5c`/`T1`/`T2` 命中 + 构造点偏移命中 + 对照组 **60 例 0 finding**） |
| `--modules` 子集一致（G9） | ✅ | `TestM3SubsetConclusionIsStable`：`--modules M3` == `--modules M1,M2,M3` 里的 M3 一半 |
| B 帧不误伤 | ✅ | `TestM3DoesNotMisjudgeBFrames`：`m3-regression` 24 例（含 v1 负偏移 B 帧）M3 下 0 finding |
| 两门 + 变异验证 | ✅ | `gates`（11 判据 / 4 规则）；`verify-gates`（21 组逐组变红；跑后 `git diff --stat` 无残留） |
| 本地全套 | ✅ | `node tools/tasks.mjs all` 全绿 |

提交：见 `git log --oneline`（末条为 `M4 · 时间戳判定：模块 M3 六条规则（92 例对账，对照 60 例误报 0）`）。

---

## 二、当前事实 · 状态不变量（下一棒动手前必须知道，否则一定踩）

| # | 事实 | 后果 / 禁区 |
|---|---|---|
| F1 | `internal/rules/structure`（M1）与 `internal/rules/duration`（M2）是**仅有的两个已实现模块**；`rules.go` 里 `m3`–`m5` 仍是桩，`Audit()` 返回 `ErrNotImplemented` → CLI 退出码 3 | **不要**把桩改成"返回空 findings"（冒充 PASS）。M4 只替换 `m3` 的转调，`m4`/`m5` 保持桩 |
| F2 | `internal/policy` 是口径常量的**唯一实现处**（`DTSRegressionTolerance=0`、`TimescaleSuspectBelow=100` 等）；`TestR3PolicyConstantsHaveSingleSource` 扫全仓 `const` | 回归判据**必须**用 `policy.DTSRegressionTolerance`；极小 timescale 阈值**必须**用 `policy.TimescaleSuspectBelow`；**不得**在 `timestamps` 包再写一份数字 |
| F3 | `go.mod` 的 `require` 必须为空、`go.sum` 必须不存在（`TestG8`） | 加任何三方依赖即红 |
| F4 | `internal/` 与 `cmd/` 禁 `net*` import（`TestG7`） | 加遥测/版本检查 = 直接违反产品红线 |
| F5 | 里程碑状态有 **3 处副本**（`plan/00 §五` 表 / `plan/05 §一` 表 / `README` 状态表），另有 `plan/00 §三` 台账行 | 翻状态必须**同一提交**改：三处副本 + 台账里标 `⬜ M4` 的行（`LAYOUT.LEDGER` 会红） |
| F6 | `fixtures/out/manifest.json` **必须入库**；`fixtures/out/*.mp4` 不入库（`.gitignore` 只写 `/out/`） | 改 `.gitignore` 的 `out/` 会连 manifest 一起忽略（`E4`） |
| F7 | `tools/verify-layout-gate.mjs` 会**临时改写仓库文件**，靠 `compose` 回滚 | 手工跑前先 `git status` 记基线；跑后 `git diff --stat` 应只剩自己的改动（`plan/06 §六 B2/B8`） |
| F8 | `tools/fixture-gen` **禁** `time` / `os.Getenv` / `os/exec` / `math/rand`（`TestGeneratorHasNoClockEnvOrExec`） | M4 **不得**改生成器引入时钟/进程 |
| F9 | `manifest.json` 现为 **92 例**；`check-fixtures` 与 CI `diff -r` 拿它当真值 | 任何生成器改动必须 `node tools/tasks.mjs fixtures` 重生成并 commit |
| F10 | 文档里**不得**出现 `make <target>` | 可执行等价物一律 `node tools/tasks.mjs <target>` |
| F11 | 92 例的**载荷字节是盒子级占位**（`ffprobe` 报 H.264 解码错误属**预期**） | 只读盒子、不解码；别"修"样本 |
| F12 | **`isobmff.Parse` 是宽容的**：`size<8`、子盒越界、声明 size 越过文件、深度超限都记进 `root.Anomalies`，**不返回 error**；只有"读不出根盒头（剩余 < 8 字节）"才是 parse error（CLI 3） | M4 若发现"盒子越界"，那是 finding（或 M1 已报的），**不是** gray/退出码 3 |
| F13 | IR 已带 **`Box.Path`**（斜杠路径；**同名兄弟才带 `[i]`**，索引从 0 起、按文件内顺序）与 `Box.Offset/PayloadStart()` | 直接用它拼 `field_path`，别再手写索引。单轨/单盒文件的路径是 `moov/trak/…`（**不带** `trak[0]`）——M3 实测 |
| F14 | **`stts`/`ctts`/`tfdt`/`trun` 都不是容器盒**，IR 里只有盒本身（`Box`），逐条字段要用 `raw` + `PayloadStart()` 自己读 | M2 的 `readStts/readCttsFirst/readElst` 是可直接照搬的**只读小工具**（带边界检查、`maxEntries` 收敛） |
| F15 | **同一根因合并**（M1/M2 纪律）：一处根因只报一条，别连环报 | M4 沿用：一处 `stts` 根因只报一条 `zero_duration_entry`/`dts_regression` |
| F16 | **M2 的判定基准与时基**（`D12`）：三口径互比在**影片时基 `mvhd.timescale`** 上做，媒体侧向下取整；判定用 `mdhd.duration` 而非 `Σstts` | M4 判的是**时间戳序列**（媒体时基），不要复用 M2 的时基选择去判 `stts` 内部——两者独立 |
| F17 | **`ctts` version 1 的偏移是**有符号**（负偏移合法，B 帧） | M4 必须按有符号读（`i32`）；否则 `m3r18`–`m3r21`、`k15` 这类 B 帧文件会被判成回退（这正是回归集存在的原因） |
| F18 | `vfr_detected` 与 `timescale_suspect` 是**gray**，`zero_duration_entry`/`dts_regression`/`tfdt_regression`/`dts_overflow_risk` 是**fail**；`gray.reason` 是**封闭集合** | 直接复用 `report.GrayVFRDetected` / `report.GrayTimescaleSuspect`，**不新增 reason** |

---

## 三、下一棒待办（M4 · 时间戳判定）

顺序即依赖顺序。**只做模块 M3（时间戳），不要顺手写偏移/priming/显示规则。**

1. **建 `internal/rules/timestamps/`**（模块 M3）
   - 与 M1/M2 同构：`func Audit(root *isobmff.Box, raw []byte, fileName string) ([]report.Finding, error)`，
     在 `rules.go` 里把 `m3` 转调它（**不要**让 `timestamps` import `internal/rules` 或兄弟规则包——
     archtest `TestModulesDoNotImportEachOther` 会红）
   - 需要读的盒子（`stts`/`ctts` 用 `raw` 读；`tfdt`/`trun`/`tfhd` 同理）：
     每轨 `stbl/stts`（条目数 + 每样本累计 DTS）、`stbl/ctts`（**v1 有符号**）、
     分片 `moof/traf/tfdt`（v0 32 位 / v1 64 位）与 `trun`、`tkhd`/`mdhd.timescale`
2. **模块 M3 全部 RuleID 实现**（`plan/03 §四`，RuleID 见下表）：

   | RuleID | 判什么 | 关键点 |
   |---|---|---|
   | `dts_regression` | `dts[n] < dts[n-1]`（容差 `policy.DTSRegressionTolerance`=0） | 报**样本序号** + `stts` 条目号 + 字节偏移；默认首个即止；**只看 DTS，B 帧的 PTS 乱序不算** |
   | `tfdt_regression` | 跨分片 `tfdt[k] < tfdt[k-1]` | 报**分片序号** + 两处 `tfdt` 偏移；v0/v1 分宽读 |
   | `zero_duration_entry` | `stts` 某条 `duration == 0` | 报 `stts` **条目号** + `duration` 字段偏移；首期按 **fail**（`data/README §5.4`），**不新增 gray** |
   | `dts_overflow_risk` | 累计越过字段位宽（`stts` 32 位；`tfdt` 按 version 0/1 取 32/64 位） | 报"在什么位置越过上限"；用整数运算，不浮点 |
   | `timescale_suspect` (gray) | `timescale < policy.TimescaleSuspectBelow`（100）且未触发 `dts_overflow_risk` | `gray.reason = timescale_suspect`（**闭集内**） |
   | `vfr_detected` (gray) | 同轨 `stts` 条目数 > 1 | `gray.reason = vfr_detected`（**闭集内**）；VFR 本身不是缺陷 |
3. **拿 M3（模块）的样本对表**（真值已在 manifest，别自己编）：
   - `C5` → `zero_duration_entry` + `vfr_detected`（gray，二者**并列不互斥**：fail 计 1、gray 计 1 → exit 1）
   - `C5b`/`C5c` → `tfdt_regression`（v0 / v1 两形态）
   - `T1` → `timescale_suspect`（gray，`timescale=30`）
   - `T2` → `dts_overflow_risk`（`stts` 累计 ≈ 2³²）
   - **对照组误报 0**：`controls` / `m3-regression` / `m4` / `m5` 共 60 例在 M3 下**一个 fail/gray 都不许报**；
     其中 **`m3-regression` 24 例就是本模块的门禁主体**（`≥20` 正常 H.264/H.265，**含 B 帧**：ctts v0 正偏移乱序 + v1 有符号负偏移）
   - 其它 `C1–C4`/`C6`/`C7`/`D`/`E`/`F`/`P`/`S` 与 M4（模块）无关的样本，本棒不判它们
4. **CLI 接线**：`--modules M3` 走通并给出真实结论；`audit <file> --json` 输出仍符合 `docs/report-schema.md`
5. **文档同步**：`docs/rules.md` 的 **M3 节**写全（每条 RuleID 的人类可读说明 + 反例指向样本 id）
6. **M4 收尾**（顺序固定）：DoD 全验 → 三处状态副本 + 台账行同提交翻 ✅ → **新建 `plan/HANDOFF-M5.md`**
   → 在**本文件**补 DoD 结果 → commit

> ⚠️ **M4 不碰**：priming/起始裁剪（模块 M4）、matrix/色彩（模块 M5）、`elst` 时长口径（模块 M2，已实现）；
> **不改** 夹具真值，**也不改** `internal/rules/structure` 与 `internal/rules/duration`
> （除非发现它们真错——那要同时说清并重跑对应模块的整夹具对账）。

---

## 四、既定口径清单（K 编号 · 动了会打挂测试）

| # | 约定 | 落在哪 | 违反时谁会红 |
|---|---|---|---|
| **K1** | 数值口径只在 `data/README §5` 定义、只在 `internal/policy` 实现 | `internal/policy/policy.go` | `archtest` R3 |
| **K2** | 三态 `pass/fail/gray`；`gray` 永不折算 pass、永不升级 fail | `internal/report` | G1/G2/G3 单测 |
| **K3** | `gray.reason` 是**封闭集合**；新增先改 `data/README §5.5` + `docs/report-schema.md` | 同上 | `TestGrayReasonMustBeClosedSet` |
| **K4** | 解析失败是 **error（退出码 3）**，不是 `gray`；但**内容级异常**（越界/截断/size 异常）是 finding（F12） | `cmd/mediatrace` / `isobmff` | `TestAuditUnparsableFileIsUsageError` |
| **K5** | 每条 `fail` 必须四元组齐（offset / field_path / values / delta） | `internal/report` | `ValidateAll` 拒绝 + G1 |
| **K6** | `elst` 按**文件内顺序**编号，**不排序**；`colr` 多实例同理（D10） | M2（已实现）/ M5 | 条目号写入 `field_path` 与 `values` |
| **K7** | 模块之间**不得**互相 import；规则是纯函数（无 I/O / 时钟 / 随机 / 网络） | `internal/rules` | `TestModulesDoNotImportEachOther` |
| **K8** | 生成器与解析器**不共享代码**（只共享 `policy` 常量） | `tools/fixture-gen` | 代码评审；`--check` 不证此事 |
| **K9** | 同输入双跑**逐字节一致**（含 `--modules` 子集） | 全链 | `check-fixtures` / CI `diff -r` |
| **K10** | 文档里的命令必须是**能跑的形态**（`node tools/tasks.mjs …`） | `plan/` `docs/` `README` | 部署期人工核对 |
| **K11** | 时长三口径互比前**先对齐到公共时基**（整数向下取整，**不引入浮点**）；公共时基 = **影片时基 `mvhd.timescale`**（决策 **D12** 细化：原"对齐到 `mdhd.timescale`"会把粗时基量化误差放大成假阳性） | `internal/rules/duration` | `data/README §5.1`；浮点会破双跑一致 |
| **K12** | `mvhd.duration==0`（分片常见）推不出预期时长时落 `duration_unknown` gray，**不猜** | 同上 | §5.5 闭集 |
| **K13** | **DTS 判据只有一条**：`dts[n] < dts[n-1]`（容差 0）。**PTS 不参与单调判定**（B 帧天然乱序）；`ctts` v1 必须按**有符号**读 | M3（本棒） | §5.4；`m3-regression` 24 例误报 0 |

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
| E7 | **C7 的载荷是占位字节** | `ffprobe` 报 H.264 解码错误 | 预期（F11）：只读盒子；判结构/口径不判编码 |
| E8 | **`cmd/mediatrace` 的 `run()` 收 `io.Writer`** | 便于测试捕获输出 | 新增 CLI 断言用 `bytes.Buffer` 即可，不要改回 `*os.File` |
| E9 | **夹具的 `stts` 是逐样本 `floor(duration/frames)` 拼的** | `Σstts` 与 `mdhd.duration` 可差几个 tick（`M3R03`：990 vs 1000 @600） | 判"时长"用声明值；判"时间戳序列"用 `stts` 本身（这是 M4 的正题） |

---

## 六、DoD 逐项 checklist（M4，取自 `plan/00 §五` 与 `plan/05 §二`）—— ✅ 收尾已验（2026-10-28）

- [x] `internal/rules/timestamps/` 建成，模块 M3 六个 RuleID 全实现 — `RuleIDs` 与 `docs/rules.md` M3 节对账（`dts_regression` / `tfdt_regression` / `zero_duration_entry` / `dts_overflow_risk` / `timescale_suspect` / `vfr_detected`）
- [x] `dts_regression` 报**样本序号** + `stts` 条目号 + 字节偏移；容差引用 `policy.DTSRegressionTolerance` — 塑形由 `TestDTSRegressionCarriesSampleIndexEntryAndOffset` 钉住（`field_path=…/stts[2].duration`，`values` 带 `sample_index`/`stts_entry`）。**如实登记**：`stts.duration` 无符号 ⇒ 良构 `stts` 的累计 DTS **不可能回退**，故本夹具集**没有** `dts_regression` 样本（`gen/builders_defects.go` 头注同此）；C5「回退 DTS」由下两条**可达**形态覆盖
- [x] `tfdt_regression`（v0/v1 两形态）报分片序号 — `C5b`(v0)/`C5c`(v1) 命中：`fragment_index=1`、`tfdt=30000 < 90000`、`values` 含**两处偏移**、`delta=60000`；`tfdt` 分宽读见单测 `TestReadTfdtHonoursFieldWidth`
- [x] `zero_duration_entry` 报条目号；`dts_overflow_risk` 报越界位置（32/64 分宽） — `C5`（条目 #1，偏移 0x230=560）；`T2`（条目 #0、样本 #1，`cumulative_dts=8589934080` 越过 32 位上限，偏移 0x228=552）；分宽见 `TestTfdtOverflowWidths` / `TestSttsOverflowLocatesEntryAndSample`
- [x] `ctts` v1 **有符号**偏移单测（B 帧不得被判回退） — 模块 M3 **只判 DTS、不判 PTS**（`ctts` 不参与判定，故负偏移不可能被当回退）；`TestM3DoesNotMisjudgeBFrames` 断言 `m3-regression` 24 例（gen 侧已证含 v0 正偏移与 **v1 有符号负偏移**两类）M3 下 **0 finding**
- [x] `T1` 落 `timescale_suspect` gray；`C5` 同时落 `zero_duration_entry`(fail) + `vfr_detected`(gray) — `T1` exit 2、偏移 0x114=276（`mdhd.timescale`）；`C5` exit 1、两 finding **并列不互斥**
- [x] `T2` 命中 `dts_overflow_risk`；`C5b`/`C5c` 命中 `tfdt_regression` — `TestM3MatchesFixtureTruth` 逐例**集合相等** + **构造点偏移命中**（`C5`=560 / `C5b`=1512 / `C5c`=1516 / `T1`=276 / `T2`=552）
- [x] **`m3-regression` 24 例（含 B 帧）误报 0**；四组对照组 60 例在 M3 下 0 finding（**集合相等**断言，与 M1/M2 同构） — `TestM3MatchesFixtureTruth`：`controls`+`m3-regression`+`m4`+`m5` 共 **60** 例 0 finding、exit 0
- [x] 每条 `fail` 带四元组（偏移 + 字段路径 + 值 + 差值） — `TestM3FailFindingsCarryFullEvidence` + 渲染层 `ValidateAll` 拒绝
- [x] `--modules M3` 单独跑通且结论 == 全集里 M3 的结论（G9） — `TestM3SubsetConclusionIsStable`（`--modules M3` == `--modules M1,M2,M3` 的 M3 一半，且幂等）
- [x] `docs/rules.md` M3 节写完（反例指向样本 id） — 六条规则 + 证据要点 + 同一根因合并 + 分宽读 + "为什么 dts_regression 无夹具"如实说明
- [x] 三处状态副本 + `plan/00 §三` 台账行同提交翻 ✅ + `plan/HANDOFF-M5.md` 已写
- [x] `node tools/tasks.mjs all` 全绿

### M4 期间的设计取舍（写清以免被当漏项）

| 取舍 | 为什么 |
|---|---|
| **`dts_regression` 保留完整判据但不设夹具** | `stts.duration` 无符号 ⇒ 累计 DTS 单调不减，字面判据 `dts[n] < dts[n-1]` 对良构 `stts` **不可能为真**（`gen/builders_defects.go` 头注已写明）。故 `data/README §4` 的 C5 用两条**可达**形态表达：`zero_duration_entry`（`C5`）与 `tfdt_regression`（`C5b`/`C5c`）。"报样本序号+条目号+偏移"的契约由包单测钉住 |
| **溢出**压过**回退**（同一根因合并，F15） | `T2` 的累计 DTS 越过 32 位时，字段宽度内的"回退"只是溢出的表象；真值只期望 `dts_overflow_risk`，故检测到越界即返回，不再单报 `dts_regression` |
| **按 `stts` 条目推进（不落全量样本表）** | `plan/01 §五 M3` 确定性纪律 / R4：每条的 `count×duration` 整数一次算完，只在越界点定位到条目号 + 样本序号，避免大文件内存爆炸。越界样本序号由 `floor((上限−累计)/时长)` 闭式算出 |
| **`tfdt` v1（64 位）不设越界** | int64 表达不了 2⁶⁴ 之上；v0 的 32 位越界由 `checkTfdtOverflow` 判（有单测，夹具无此形态） |
| **`timescale_suspect` 排除已溢出轨** | §5.2 明写"低于 100 **且未触发** `dts_overflow_risk`"——溢出是 fail，不该再落 gray |
| **空 `stts`（分片轨）不算 VFR** | 0 条不多于 1 条；`K9-frag`/`C5b`/`C5c`/`F1` 的分片轨不落 `vfr_detected` |

### M4 期间顺手修正的真实缺陷

- **`plan/00 §三` 台账一行误标里程碑**：`priming / encoder delay…` 一行原标 `⬜ M4`，但它属**模块 M4（priming）**=**里程碑 M5**（同组的 `matrix…`/`colr…` 两行都标 `⬜ M5`）。M4 一翻 ✅，`LAYOUT.LEDGER` 立刻把这一行判成"已 ✅ 里程碑仍挂未完成"。本次改为 `⬜ M5`（如实回填，未放宽带宽）。

**M4 未做（如实登记）**：真实样本（22 条）的实际审计复跑（需样本本体，属发布前复核）；
`GATE`/`--suite` 批跑模式（属 M5 报告收口，见 `HANDOFF-M5 §二 F21`）。

---

## 七、关键命令速查

```bash
# 门（新对话第一件事）
node tools/tasks.mjs gates              # 布局门 + 脱敏门
node tools/tasks.mjs verify-gates       # 两门自己的变异验证（21 组，逐组必须按预期变红）

# 夹具（M1 产物，本棒只消费、不改真值）
node tools/tasks.mjs fixtures           # 生成 fixtures/out + manifest.json（92 例）
node tools/tasks.mjs check-fixtures     # 双跑逐字节 + 与入库 manifest 比对
go run ./tools/fixture-gen -list        # 只打印 case id / sha256 / size

# 代码
node tools/tasks.mjs test               # go vet + go test ./...（含整夹具集对账）
node tools/tasks.mjs bin                # 构建 bin/mediatrace（Win: .exe）
node tools/tasks.mjs all                # 一键本地等价校验

# 单文件审计（M1/M2 已给真实结论；M3–M5 未实现前仍退出码 3）
bin/mediatrace audit fixtures/out/c5_zero_duration_entry.mp4 --json --modules M3
bin/mediatrace audit fixtures/out/c5b_tfdt_regression.mp4   --json --modules M3
bin/mediatrace audit fixtures/out/c4a_duration_2x_mvhd.mp4  --json --modules M2
```

**口径只查** `data/README.md §5`；**计划状态只查** `plan/00 §五`；**决策只查** `plan/00 §二`。