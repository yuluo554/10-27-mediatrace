# HANDOFF-M3 · 交给第 3 棒（M3 时长口径判定）

> **命名约定**：`plan/HANDOFF-M<k>.md` = **交给第 k 棒的入口交接单**（由第 k-1 棒收尾时写）。
> 第 k 棒收尾时在本文件补 DoD 结果，并**新建** `HANDOFF-M<k+1>.md`。布局门 `LAYOUT.HANDOFF_ENTRY`
> 断言"当前棒必须有入口交接单（七节齐）"，`LAYOUT.HANDOFFS` 断言"已 ✅ 的里程碑有七节交接单"。
>
> 本文件由 **M2（结构判定）收尾**产生。M2 的产物是**模块 M1 的判定器**
> （`internal/rules/structure` + `internal/isobmff` 的 IR），对 92 例夹具**召回精确、对照误报 0**。
> M3 是第一次判"时间轴**口径之间**是否自洽"——把 M2 的只读盒子树用起来。

---

## 一、当前进度 + 证据

| 项 | 状态 | 证据（本机实测，2026-10-28） |
|---|---|---|
| M1 · 数据先行 | ✅ | 92 例夹具入库 manifest；C1–C7 每类 ≥1；同 seed 双跑逐字节一致 |
| M2 · 结构判定 | ✅ | 见 §六 DoD 结果；产物 = `internal/isobmff` IR + `internal/rules/structure`（八条规则） |
| `internal/isobmff` 迭代 IR | ✅ | `Parse/Walk/Find/FindAll`；`size==0/1` 两形态；深盒树 6000 层不爆栈；异常记 `Box.Anomalies` |
| 模块 M1 八条 RuleID | ✅ | `too_small_size` / `child_overruns_parent` / `duplicate_moov` / `duplicate_mdat` / `stco_out_of_mdat` / `truncated_mdat` / `ftyp_brand_mismatch` / `max_depth_exceeded` |
| 整夹具集对账 | ✅ | `cmd/mediatrace` 的 `TestM1MatchesFixtureTruth`：92 例**集合相等**（缺陷 11 例召回、对照 60 例 0 finding） |
| `--modules M1` 接线 | ✅ | `bin/mediatrace audit fixtures/out/c2_stco_out_of_mdat.mp4 --modules M1` → exit 1 + 四元组 |
| 两门 + 变异验证 | ✅ | `node tools/tasks.mjs gates`（11 判据 / 4 规则）；`verify-gates`（21 组逐组变红） |
| 本地全套 | ✅ | `node tools/tasks.mjs all` → exit 0 |
| **M3 实现** | ✅ **已完成** | 见 §六 DoD 结果；产物 = `internal/rules/duration`（六条规则）+ `rules.go` 的 `m2` 转调 |

提交：见 `git log --oneline`（末条为 `M3 · 时长口径判定 …`）。
> 注：本交接单 §一 的第 5 行原为 `⬜ 未开工`，M3 收尾时已同提交翻 ✅（见 §六 DoD 结果）。

---

## 二、当前事实 · 状态不变量（下一棒动手前必须知道，否则一定踩）

| # | 事实 | 后果 / 禁区 |
|---|---|---|
| F1 | `internal/rules/structure` 是**唯一已实现**的模块；`rules.go` 里 `m2`–`m5` 仍是桩，`Audit()` 返回 `ErrNotImplemented` → CLI 退出码 3 | **不要**把桩改成"返回空 findings"（冒充 PASS）。M3 只替换 `m2` 的转调，`m3`–`m5` 保持桩 |
| F2 | `internal/policy` 是口径常量的**唯一实现处**（`TickTolerance=1` 等）；`TestR3PolicyConstantsHaveSingleSource` 扫全仓 `const` | 时长容差**必须**用 `policy.TickTolerance`，**不得**在 `duration` 包再写一份数字 |
| F3 | `go.mod` 的 `require` 必须为空、`go.sum` 必须不存在（`TestG8`） | 加任何三方依赖即红 |
| F4 | `internal/` 与 `cmd/` 禁 `net*` import（`TestG7`） | 加遥测/版本检查 = 直接违反产品红线 |
| F5 | 里程碑状态有 **3 处副本**（`plan/00 §五` 表 / `plan/05 §一` 表 / `README` 状态表），另有 `plan/00 §三` 台账行 | 翻状态必须**同一提交**改：三处副本 + 台账里标 `⬜ M3` 的行（`LAYOUT.LEDGER` 会红） |
| F6 | `fixtures/out/manifest.json` **必须入库**；`fixtures/out/*.mp4` 不入库（`.gitignore` 只写 `/out/`） | 改 `.gitignore` 的 `out/` 会连 manifest 一起忽略 |
| F7 | `tools/verify-layout-gate.mjs` 会**临时改写仓库文件**，靠 `compose` 回滚 | 手工跑前先 `git status` 记基线；跑后 `git diff --stat` 应只剩自己的改动（`plan/06 §六 B2/B8`） |
| F8 | `tools/fixture-gen` **禁** `time` / `os.Getenv` / `os/exec` / `math/rand`（`TestGeneratorHasNoClockEnvOrExec`） | M3 **不得**改生成器引入时钟/进程 |
| F9 | `manifest.json` 现为 **92 例**；`check-fixtures` 与 CI `diff -r` 拿它当真值 | 任何生成器改动必须 `node tools/tasks.mjs fixtures` 重生成并 commit |
| F10 | 文档里**不得**出现 `make <target>` | 可执行等价物一律 `node tools/tasks.mjs <target>` |
| F11 | 92 例的**载荷字节是盒子级占位**（`ffprobe` 报 H.264 解码错误属**预期**） | 只读盒子、不解码；别"修"样本 |
| F12 | **C4b 是非负越界形态**：没有字段写成负数，越界关系写在 `Σelst`(1500) > 轨道声明时长(1000) 的**结构**里 | M3 必须从结构算越界，**不得**指望读到负的 `duration` |
| F13 | **`isobmff.Parse` 是宽容的**：`size<8`、子盒越界、声明 size 越过文件、深度超限都记进 `root.Anomalies`，**不返回 error**；只有"读不出根盒头（剩余 < 8 字节）"才是 parse error（CLI 3） | M3 若发现"盒子越界"，那是 finding（或 M1 已报的），**不是** gray/退出码 3 |
| F14 | IR 已带 **`Box.Path`**（斜杠路径，索引从 0 起、**按文件内顺序**，如 `moov/trak[0]/edias…`）与 `Box.Offset/ParentEnd` | 直接用它拼 `field_path`，别再手写索引；`elst`[i] **按序编号、不排序**（D10） |
| F15 | M1 的**同一根因合并**：重复 `moov` 时跳过 `stco` 校验；越界子盒子树不展开 | M3 沿用此纪律：一处根因只报一条，别连环报 |

---

## 三、下一棒待办（M3 · 时长口径判定）

顺序即依赖顺序。**只做模块 M2（时长口径），不要顺手写时间戳/偏移/显示规则。**

1. **建 `internal/rules/duration/`**（模块 M2）
   - 与 M1 同构：`func Audit(root *isobmff.Box, raw []byte, fileName string) ([]report.Finding, error)`，
     在 `rules.go` 里把 `m2` 转调它（**不要**让 `duration` import `internal/rules`——会成环；
     archtest `TestModulesDoNotImportEachOther` 也会红）
   - 需要读的盒子（全在 IR 上，或用 `raw` 按偏移读字段）：
     `moov/mvhd`（timescale+duration）、每轨 `tkhd.duration`、`mdia/mdhd`（timescale+duration）、
     `stbl/stts`（Σ count×duration）、`edts/elst`（各段 segment_duration / media_time / media_rate）、
     `stbl/stsz`+`stco`/`ctts`（推**首帧 PTS**）
2. **模块 M2 全部 RuleID 实现**（`plan/03 §三`，RuleID 见下表）：

   | RuleID | 判什么 | 关键点 |
   |---|---|---|
   | `duration_mismatch` | 三口径互比超容差 | `values` 必含三口径原值（`stts_sum`/`mvhd_duration`/`elst_sum`，对齐到轨道 timescale）；`delta` = 最大差值；**容差用 `policy.TickTolerance`** |
   | `elst_item_rate_not_one` | `media_rate ≠ 1` | 报**条目号**（`elst[i]`，按文件内顺序）；同段**不按等速换算** |
   | `elst_item_order_suspect` | 段次序使 `Σ` 与首帧 PTS 对不上 | 与"时长错"**分开报**（`plan/01 §八` 创新点） |
   | `empty_edit` | `media_time == -1` | 报 `segment_duration` + 由此产生的起始偏移（空 edit 本身是**信息**，只有与显示时长矛盾才 fail） |
   | `first_pts_outside_media` | 首帧 PTS 不落在 `elst` media 区间 | 报 PTS 与区间；`media_time == -1` 段无 media 区间（不算） |
   | `duration_unknown` (gray) | `mvhd.duration==0` 且推不出 | `gray.reason = duration_unknown`（闭集内，**不新增 reason**） |
3. **拿 M2（模块）的样本对表**（真值已在 manifest，别自己编）：
   - `C4a`/`C4a2`/`C4a3` → `duration_mismatch`；`C4b` → `duration_mismatch` + `first_pts_outside_media`
   - `E1` → `elst_item_rate_not_one`（条目号 **1**）；`E2` → `duration_mismatch`；`E3` → `elst_item_order_suspect`
   - `F1` → `duration_unknown`（gray，**不计 fail**）
   - **对照组误报 0**：`controls` / `m3-regression` / `m4` / `m5` 共 60 例在 M2 下**一个 fail/gray 都不许报**
     （`K4-duration`、`K10-empty-edit` 是"三口径对齐 / 空 edit 已计入"的合法近邻）
   - 其它 `D/E/P/T` 与 M3 模块无关的样本，本棒不判它们
4. **CLI 接线**：`--modules M2` 走通并给出真实结论；`audit <file> --json` 输出仍符合 `docs/report-schema.md`
5. **文档同步**：`docs/rules.md` 的 **M2 节**写全（每条 RuleID 的人类可读说明 + 反例指向样本 id）
6. **M3 收尾**（顺序固定）：DoD 全验 → 三处状态副本 + 台账行同提交翻 ✅ → **新建 `plan/HANDOFF-M4.md`**
   → 在**本文件**补 DoD 结果 → commit

> ⚠️ **M3 不碰**：`.MP4` 内 DTS 单调/溢出、priming/起始裁剪、matrix/色彩（属 M4/M5）；
> **不改** M1 的夹具真值，**也不改** `internal/rules/structure`（除非发现 M1 真错——那要同时说清并重跑 M1 对账）。

---

## 四、既定口径清单（K 编号 · 动了会打挂测试）

| # | 约定 | 落在哪 | 违反时谁会红 |
|---|---|---|---|
| **K1** | 数值口径只在 `data/README §5` 定义、只在 `internal/policy` 实现 | `internal/policy/policy.go` | `archtest` R3 |
| **K2** | 三态 `pass/fail/gray`；`gray` 永不折算 pass、永不升级 fail | `internal/report` | G1/G2/G3 单测 |
| **K3** | `gray.reason` 是**封闭集合**；新增先改 `data/README §5.5` + `docs/report-schema.md` | 同上 | `TestGrayReasonMustBeClosedSet` |
| **K4** | 解析失败是 **error（退出码 3）**，不是 `gray`；但**内容级异常**（越界/截断/size 异常）是 finding（F13） | `cmd/mediatrace` / `isobmff` | `TestAuditUnparsableFileIsUsageError` |
| **K5** | 每条 `fail` 必须四元组齐（offset / field_path / values / delta） | `internal/report` | `ValidateAll` 拒绝 + G1 |
| **K6** | `elst` 按**文件内顺序**编号，**不排序**；`colr` 多实例同理（D10） | M2（本棒）/ M5 | `Box.Path` 已按序；写测试自证条目号 |
| **K7** | 模块之间**不得**互相 import；规则是纯函数（无 I/O / 时钟 / 随机 / 网络） | `internal/rules` | `TestModulesDoNotImportEachOther` |
| **K8** | 生成器与解析器**不共享代码**（只共享 `policy` 常量） | `tools/fixture-gen` | 代码评审；`--check` 不证此事 |
| **K9** | 同输入双跑**逐字节一致**（含 `--modules` 子集） | 全链 | `check-fixtures` / CI `diff -r` |
| **K10** | 文档里的命令必须是**能跑的形态**（`node tools/tasks.mjs …`） | `plan/` `docs/` `README` | 部署期人工核对 |
| **K11** | 三口径比较前先**对齐到该轨 `mdhd.timescale`**（整数向下取整），**不引入浮点** | `internal/rules/duration` | `data/README §5.1`；浮点会破双跑一致 |
| **K12** | `mvhd.duration==0`（分片常见）推不出预期时长时落 `duration_unknown` gray，**不猜** | 同上 | §5.5 闭集 |

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
| E8 | **`cmd/mediatrace` 的 `run()` 现收 `io.Writer`** | 便于测试捕获输出 | 新增 CLI 断言用 `bytes.Buffer` 即可，不要改回 `*os.File` |

---

## 六、DoD 逐项 checklist（M3，取自 `plan/00 §五` 与 `plan/05 §二`）—— ✅ 收尾已验（2026-10-28）

- [x] `internal/rules/duration/` 建成，模块 M2 六个 RuleID 全实现 — `RuleIDs` 与 `docs/rules.md` M2 节对账（`duration_mismatch` / `elst_item_rate_not_one` / `elst_item_order_suspect` / `empty_edit` / `first_pts_outside_media` / `duration_unknown`）
- [x] 三口径互比引用 `policy.TickTolerance`（不新写数字），对齐到公共时基 — 公共时基 = **影片时基 `mvhd.timescale`**（决策 **D12**，`data/README §5.1` 已细化）；比较走 `policy.DurationWithinTolerance`
- [x] `elst` 按文件内顺序编号、**不排序**；`media_rate≠1` 报条目号 — `E1` → 条目 **1**（`field_path=moov/trak/edts/elst[1].media_rate`，偏移指向该条目 `media_rate` 字段）
- [x] 空 edit（`media_time==-1`）语义实现 — 只贡献放映延迟、不引用媒体；**领先且非零不报**（`K10-empty-edit` 0 finding）；与显示时长的矛盾由 `duration_mismatch` 一条表达（同一根因不连环报，`E2` 只报 1 条）
- [x] `C4a`/`C4a2`/`C4a3` 与 `C4b`、`E1/E2/E3` 全部命中；`F1` 落 `duration_unknown` gray — `TestM2MatchesFixtureTruth` 逐例**集合相等**：`C4a`(mvhd 2×)/`C4a2`(tkhd 2×)/`C4a3`(mvhd 偏短)/`C4b`(duration_mismatch + first_pts_outside_media)/`E1`(条目号 1)/`E2`(duration_mismatch)/`E3`(order_suspect)/`F1`(gray, exit 2)
- [x] 对照组（60 例）误报 0（**集合相等**断言，与 M1 同构） — `controls` + `m3-regression` + `m4` + `m5` 共 **60** 例在 M2 下 0 finding、exit 0（含 `K6-priming`/`M4P3`/`M4P4` 这类"合法起点裁剪"近邻）
- [x] 每条 `fail` 带四元组（偏移 + 字段路径 + 值 + 差值） — `TestM2FailFindingsCarryFullEvidence` + 渲染层 `ValidateAll` 拒绝
- [x] `--modules M2` 单独跑通且结论 == 全集里 M2 的结论（G9） — `TestM2SubsetConclusionIsStable`（`--modules M2` == `--modules M1,M2` 的 M2 一半，且幂等）
- [x] `docs/rules.md` M2 节写完（反例指向样本 id） — 六条规则 + 口径表 + 时基说明 + 反例 `case_id`
- [x] 三处状态副本 + `plan/00 §三` 台账行同提交翻 ✅ + `plan/HANDOFF-M4.md` 已写
- [x] `node tools/tasks.mjs all` 全绿

### M3 期间的设计取舍（写清以免被当漏项）

| 取舍 | 为什么 |
|---|---|
| **公共时基取 `mvhd.timescale`，不取 `mdhd.timescale`** | `mvhd`/`tkhd`/`elst` 原生就在影片时基；媒体侧向下取整过来的量化误差 < 1 影片 tick，落在 1 tick 容差内。反向换算会把误差放大 `mdhd.timescale/mvhd.timescale` 倍——夹具 `M3R23`（`tkhd`=33@1000、`mdhd`=3000@90000）反算差 30 个媒体 tick，会被判成超容差（假阳性）。已按 D9 改 `data/README §5.1` 并记决策 **D12**；`TestDurationComparesInMovieTimescale` 把这条钉住 |
| **判定基准用 `mdhd.duration`（声明值），不用 `Σstts`** | 夹具的 `stts` 是逐样本 `floor(duration/frames)` 拼的，`Σstts` 与声明值天然差几个 tick（`M3R03`：990 vs 1000 @600）。若拿 `Σstts` 当判据，`M3R03`/`M3R06`/`M3R12` 三个对照组会被误报。`Σstts` 仍作为证据原值报出（`values.stts_sum`） |
| **`first_pts_outside_media` 只在"编辑表跨度对不上"时才判** | 跨度一致（`Σ非空段` == `mdhd.duration`）就是一次**合法起点裁剪**：`media_time>0` 且首帧 PTS=0 是裁剪的正常后果，属 M4 的 `start_trim`。若照字面判，`K6-priming`/`M4N1`/`M4P3`/`M4P4` 四个对照/正例会被误报 |
| **空 edit 不单独成 finding（同一根因合并，F15）** | 真值台账里 `E2` 期望只有 `duration_mismatch`（空 edit 的延迟没被 `mvhd` 计入 → 三口径互斥）。若同时报 `empty_edit`，就变成同一根因连环报。`empty_edit` 的**触发条件**收窄为"空 edit 不在最前 **或** `segment_duration==0`"这类自相矛盾形态（夹具无此形态，由单测覆盖） |
| **一个轨道最多一条 `duration_mismatch`** | 优先级：影片级（`mvhd` vs 放映时长）→ 轨道级（`tkhd` vs `mdhd`）→ 编辑表跨度 vs `mdhd`。`C4b` 同时违反"影片级"与"编辑表跨度"两条，只报第一条（证据指向 `elst[0].segment_duration`） |
| **分隔片空样本表轨道** | 分片轨 `stts` 为空、`mdhd.duration`=0，其时长由 `moof` 决定（属二期）。此时只在 `mvhd.duration==0` **且** `mehd`/`trun`/`trex` 都推不出时才落 `duration_unknown`（`F1`）；`K9-frag`/`C5b`/`C5c` 推得出 → 0 finding |
| **`S1` 类（"差出在 `elst` 而非 `stts`"）以 `C4b`/`E2` 覆盖** | 真值台账里没有独立的"输入 3 段 `elst` + 输出失配"合成对；`C4b`/`E2` 的 `Σelst` 越界形态正是"差值出在编辑表"的同类。已如实写进 `plan/05 §二` M3 DoD |

**M3 未做（如实登记）**：真实样本（22 条，`data/real-samples.json`，本体不入库、`verified: null`）的
实际审计复跑——需要样本本体才谈得上，属发布前复核；首期门禁以 92 例合成夹具为准。

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

# 单文件审计（M2 起 M1 给真实结论；M2 模块未实现前仍退出码 3）
bin/mediatrace audit fixtures/out/c4a_duration_2x_mvhd.mp4 --json --modules M2
bin/mediatrace audit fixtures/out/e1_media_rate_not_one.mp4 --json --modules M2
```

**口径只查** `data/README.md §5`；**计划状态只查** `plan/00 §五`；**决策只查** `plan/00 §二`。
