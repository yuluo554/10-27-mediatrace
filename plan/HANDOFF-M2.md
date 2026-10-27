# HANDOFF-M2 · 交给第 2 棒（M2 结构判定）

> **命名约定**：`plan/HANDOFF-M<k>.md` = **交给第 k 棒的入口交接单**（由第 k-1 棒收尾时写）。
> 第 k 棒收尾时在本文件补 DoD 结果，并**新建** `HANDOFF-M<k+1>.md`。布局门 `LAYOUT.HANDOFF_ENTRY`
> 断言"当前棒必须有入口交接单（七节齐）"，`LAYOUT.HANDOFFS` 断言"已 ✅ 的里程碑有七节交接单"。
>
> 本文件由 **M1（数据先行）收尾**产生。M1 的产物是**带真值的样本**（92 例 + 22 条真实登记），
> **不是判定器**——判定规则一行都还没写。M2 就是从"有样本"跨到"能判定"的第一棒。

---

## 一、当前进度 + 证据

| 项 | 状态 | 证据（本机实测，2026-10-27） |
|---|---|---|
| M1 · 数据先行 | ✅ | 92 例夹具入库 manifest；C1–C7 每类 ≥1；同 seed 双跑逐字节一致 |
| 合成缺陷集 | ✅ 92 例 | `fixtures/out/manifest.json`：defects 32 / controls 16 / m3-regression 24 / m4 10 / m5 10 |
| 真实样本登记 | ✅ 22 条 | `data/real-samples.json`（sha256 + source_url + license；**样本本体不入库**） |
| 两门 + 变异验证 | ✅ | `node tools/tasks.mjs gates`（11 判据 / 4 规则）；`verify-gates`（21 组逐组变红） |
| 本地全套 | ✅ | `node tools/tasks.mjs all` → exit 0 |
| **M2 实现** | ✅ **已完成** | 见 §六 DoD 结果；产物 = `internal/isobmff` IR + `internal/rules/structure`（八条规则） |

提交：见 `git log --oneline`（末条为 `M2 · 结构判定 …`）。

---

## 二、当前事实 · 状态不变量（下一棒动手前必须知道，否则一定踩）

| # | 事实 | 后果 / 禁区 |
|---|---|---|
| F1 | `internal/rules` 五个模块仍是**桩**，`Audit()` 返回 `ErrNotImplemented`，CLI 映射退出码 3 | **不要**改成"返回空 findings"——那是冒充 PASS，会打挂 `TestSkeletonModuleIsNotImplemented` |
| F2 | `internal/policy` 是口径常量的**唯一实现处**；`TestR3PolicyConstantsHaveSingleSource` 扫全仓 `const` 定义 | 在第二个包再写一份 `TickTolerance` 之类即红 |
| F3 | `go.mod` 的 `require` 必须为空、`go.sum` 必须不存在（`TestG8`） | 加任何三方依赖即红（含测试库） |
| F4 | `internal/` 与 `cmd/` 禁 `net*` import（`TestG7`） | 想加遥测/版本检查 = 直接违反产品红线 |
| F5 | 里程碑状态有 **3 处副本**（`plan/00 §五` 表 / `plan/05 §一` 表 / `README` 状态表） | 翻状态必须三处同一提交改；`plan/00 §四` 的提交材料表**不是**状态表（门按表头定位） |
| F6 | `fixtures/out/manifest.json` **必须入库**；`fixtures/out/*.mp4` 不入库 | `.gitignore` 里只能写 `/out/`（顶层），写成 `out/` 会把 manifest 一起忽略 |
| F7 | 变异验证脚本会**临时改写仓库文件**，靠 `compose` 回滚 | 手工跑前先 `cp` 备份；若中途 `Ctrl-C` 可能残留（详见 `plan/06 §六 B2/B8`） |
| F8 | `tools/fixture-gen` **禁** `time` / `os.Getenv` / `os/exec` / `math/rand`（`TestGeneratorHasNoClockEnvOrExec`） | M2 **不得**为了让样本"更好看"去改生成器读时钟/起进程 |
| F9 | `manifest.json` 现为 **92 例**；`check-fixtures` 与 CI `diff -r` 都拿它当真值 | 任何生成器改动后必须 `node tools/tasks.mjs fixtures` 重生成并 commit，否则 `--check` 会红 |
| F10 | 文档里**不得**出现 `make <target>` 形式（本机无 make） | 可执行等价物一律 `node tools/tasks.mjs <target>` |
| F11 | 92 例里的**载荷字节是盒子级占位**：`ffprobe` 报"H.264 解码错误"属**预期**——本项目**只读盒子、不解码** | **不得**因为 ffprobe 报错就"修"样本；真值只按盒子字段构造（`plan/04` T5） |
| F12 | C4b 是**非负越界**形态：没有字段写成负数，越界关系写在 `Σelst`(1500) > 轨道声明时长 的**结构**里 | 判定器必须从结构算出越界，**不得**指望读到负的 `duration` |

---

## 三、下一棒待办（M2 · 结构判定）

顺序即依赖顺序，前一项不做完不要碰后一项。**只做模块 M1（结构），不要顺手写时长/时间戳/偏移/显示规则。**

1. **`internal/isobmff` 迭代遍历 + IR**（不递归，防爆栈）
   - `size==0`（到文件尾）与 `size==1`（`largesize` 64 位）两形态各自正确展开
   - 深盒树 **≥5000 层**不 panic、不爆栈（C7 里的 `S3` 是 100 层，另需合成更深样本或直接构造 IR 测）
   - 越界/截断必须**带偏移**返回，不得 panic
2. **模块 M1 全部 RuleID 实现**（`internal/rules/structure`，RuleID 见 `plan/03 §二`）：
   `too_small_size` / `child_overruns_parent` / `duplicate_moov`（**列出全部**偏移）/
   `duplicate_mdat` / `stco_out_of_mdat`（条目号 + 目标区间 vs `mdat` 区间）/
   `truncated_mdat` / `ftyp_brand_mismatch` / `max_depth_exceeded`
   - **同一根因合并**：`size` 异常导致的子盒越界只报 `truncated_parent` 一条（`plan/03 §1.3`）
   - 每条 `fail` 必须四元组齐（offset / field_path / values / delta）
3. **拿 M1 的样本对表**（真值已在 manifest，别自己编）：
   - `C1`/`C1b` → `duplicate_moov`+偏移；`C2*` → `stco_out_of_mdat`；`C3*` → `truncated_mdat`；
     `S1` 子盒越界 → `child_overruns_parent`；`S2` 双 `mdat`；`S3` 深嵌 → `max_depth_exceeded`
   - **对照组误报 0**：`controls`/`m3-regression` 里的正常样本一个 `fail` 都不许报
     （C 系列以外的 D/E/P/T/F 属**别的模块**，本棒不判它们）
4. **CLI 接线**：`--modules M1` 走通并给出真实结论；`audit <file> --json` 输出符合 `docs/report-schema.md`
5. **文档同步**：`docs/rules.md` 的 M1 节写全（每条 RuleID 的人类可读说明 + 反例指向样本 id）
6. **M2 收尾**（顺序固定）：DoD 全验 → 三处状态副本同提交翻 ✅ → **新建 `plan/HANDOFF-M3.md`**
   → 在**本文件**补 DoD 结果 → commit

> ⚠️ **M2 不碰**：`.MP4` 内时长/时间戳/priming/色彩判定（属 M3–M5）；**不改** M1 的夹具真值
> （除非发现样本本身构造错——那要同时重生成 manifest 并在提交信息里写明）。

---

## 四、既定口径清单（K 编号 · 动了会打挂测试）

| # | 约定 | 落在哪 | 违反时谁会红 |
|---|---|---|---|
| **K1** | 数值口径只在 `data/README §5` 定义、只在 `internal/policy` 实现 | `internal/policy/policy.go` | `archtest` R3 |
| **K2** | 三态 `pass/fail/gray`；`gray` 永不折算 pass、永不升级 fail | `internal/report` | G1/G2/G3 单测 |
| **K3** | `gray.reason` 是**封闭集合**；新增先改 `data/README §5.5` + `docs/report-schema.md` | 同上 | `TestGrayReasonMustBeClosedSet` |
| **K4** | 解析失败是 **error（退出码 3）**，不是 `gray` | `cmd/mediatrace` | `TestAuditUnparsableFileIsUsageError` |
| **K5** | 每条 `fail` 必须四元组齐（offset / field_path / values / delta） | `internal/report` | `ValidateAll` 拒绝 + G1 |
| **K6** | `elst` 按**文件内顺序**编号，**不排序**；`colr` 多实例同理 | M3 / M5 实现 | 无自动门——写测试时自证 |
| **K7** | 模块之间**不得**互相 import；规则是纯函数（无 I/O / 时钟 / 随机 / 网络） | `internal/rules` | `TestModulesDoNotImportEachOther` |
| **K8** | 生成器与解析器**不共享代码**（只共享 `policy` 常量） | `tools/fixture-gen` | 代码评审；`--check` 不证此事 |
| **K9** | 同输入双跑**逐字节一致**（含 `--modules` 子集） | 全链 | `check-fixtures` / CI `diff -r` |
| **K10** | 文档里的命令必须是**能跑的形态**（`node tools/tasks.mjs …`），不是 `make …` | `plan/` `docs/` `README` | 部署期人工核对 |

---

## 五、本机环境坑（只写实测过的）

| # | 坑 | 实测表现 | 处置 |
|---|---|---|---|
| E1 | **Go 不在 `PATH`** | `go: command not found` | 实际在 `%USERPROFILE%\go-sdk\go\bin\go.exe`；`tasks.mjs` 自动探测；也可 `GO=… node tools/tasks.mjs test` |
| E2 | **本机无 `make`** | `make: command not found` | 一律 `node tools/tasks.mjs <target>` |
| E3 | **Windows 可执行必须 `.exe`** | `spawnSync bin/mediatrace` = ENOENT（文件其实在） | `tasks.mjs` 按平台拼名；CI 的 Linux 产物不带后缀 |
| E4 | **`git check-ignore` 的目录语义** | 父目录被忽略后，其内文件的 `!` 例外**不生效** | 忽略顶层产物用 `/out/` 而不是 `out/`；常驻断言 `check-ignore-rules` |
| E5 | **`flag` 包遇位置参数即停** | `audit <file> --json` 曾直接 exit 3 | 已用 `parseInterspersed` 修；改参数面时**先跑** `TestFlagOrderDoesNotMatter` |
| E6 | **变异脚本会临时改仓库文件** | 中途 `Ctrl-C` 可能残留（`plan/06 §六 B2`） | 跑前 `git status` 记基线；跑后 `git diff --stat` 应只剩自己的改动 |
| E7 | **C7 的载荷是占位字节** | `ffprobe` 报 H.264 解码错误 | 预期（`F11`）：只读盒子；判结构不要判编码 |

---

## 六、DoD 逐项 checklist（M2，取自 `plan/00 §五` 与 `plan/05 §二`）—— ✅ 收尾已验（2026-10-28）

- [x] `internal/isobmff` 迭代遍历 + `size==0/1` 两形态（golden 字节） — `Parse` 迭代（显式栈）；`TestParseSizeZeroMeansToEnd` / `TestParseLargesize` 保真
- [x] 模块 M1 全部 RuleID 实现（`plan/03 §二` 八条） — `internal/rules/structure`；`RuleIDs` 与 docs 对账
- [x] `C1`/`C2`/`C3` 命中 **3/3**、对照组**误报 0** — `TestM1MatchesFixtureTruth` 对 **92 例集合相等**：缺陷 11 例（C1/C1b、C2–C2d、C3/C3b、S1/S2/S3）召回精确，对照 60 例 0 finding
- [x] 深盒树（≥5000 层）不 panic、不爆栈 — `TestDeepNestingAbove5000RecordsDepthAnomaly`（6000 层，记 1 条 `max_depth_exceeded`）
- [x] 每条 `fail` 带四元组（偏移 + 字段路径 + 值 + 差值） — 所有 finding 过 `report.Finding.Validate()`；单测逐条断言
- [x] `--modules M1` 单独跑通且结论 == 全集里 M1 的结论（G9） — `TestM1SubsetConclusionIsStable` + `TestFlagOrderDoesNotMatter`
- [x] `docs/rules.md` M1 节写完（反例指向样本 id） — 八条规则的判据/证据/反例 + `case_id`
- [x] 三处状态副本同提交翻 ✅ + `plan/HANDOFF-M3.md` 已写
- [x] `node tools/tasks.mjs all` 全绿

### M2 期间的设计取舍（写清以免被当漏项）

| 取舍 | 为什么 |
|---|---|
| **`Parse` 改为宽容**：内容级异常（`size<8`、子盒越界、声明 size 越过文件、深度超限）记进 `root.Anomalies`，**不返回 error** | `plan/03 §二` 要求 `too_small_size` / `max_depth_exceeded` 是 **finding**；若 `size<8` 仍硬失败，规则层永远看不到它。只有"连根盒头都读不出"（剩余 < 8 字节）才判 parse error（退出码 3）。据此更新了两条骨架期单测（`TestParseRejectsTooSmallSize` → `TestParseRecordsTooSmallSize`；`TestAuditUnparsableFileIsUsageError` 的输入改成真正读不出盒头的 3 字节） |
| **`duplicate_mdat` 在含 `moof` 的分片文件里不报** | 分片文件每片自带 `mdat` 是**合法形态**（`K9-frag`/`C5b`/`C5c`/`F1` 全是 2 个 `mdat`）。不排除会把对照组打爆 → 误报 |
| **重复 `moov` ⇒ 跳过 `stco/co64` 区间校验** | 同"同一根因合并"（`plan/03 §1.3`）：`C1b` 把第二份 `moov` 紧贴插入，原 `stco` 值随即落在第二份 `moov` 里——若照报会多出一条 `stco_out_of_mdat`，与真值 `["duplicate_moov"]` 不符 |
| **`ftyp_brand_mismatch` 取窄口径** | 只在 major brand **就是编码名**（`avc1`/`hvc1`/…）且与 `stsd` 条目矛盾时报；否则 `isom` 容器装 `hvc1` 轨这类**合法**组合会被误伤（`m3-regression` 里有 6 例 `hvc1`） |
| **`TestFlagOrderDoesNotMatter` 分路径断言** | M1 实现后 `--modules M1` 对合法样本是 pass(0)，而默认全集仍因 M2–M5 未实现落 3——两条路径分别断言，避免"参数形态没被认出来"与"模块未实现的 3"混为一谈 |
| **`cmd/mediatrace` 的 `run()`/`usage()` 改收 `io.Writer`** | 便于测试用 `bytes.Buffer` 捕获 JSON（不改语义；既有调用仍传 `*os.File`） |

**M2 未做（如实登记，见 `plan/06 §三` 缺口表）**：MKV EBML 结构对应项（`plan/01 §十`）——
首期夹具与生成器均为 MP4，无 EBML 真值可对，顺延二期；已把 `plan/00 §五` M2 内容格写明。

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
node tools/tasks.mjs test               # go vet + go test ./...
node tools/tasks.mjs bin                # 构建 bin/mediatrace（Win: .exe）
node tools/tasks.mjs all                # 一键本地等价校验

# 单文件审计（M2 起应给出真实结论；未实现的模块仍返回退出码 3）
bin/mediatrace audit fixtures/out/c1_dual_moov.mp4 --json --modules M1
```

**口径只查** `data/README.md §5`；**计划状态只查** `plan/00 §五`；**决策只查** `plan/00 §二`。
