# HANDOFF-M1 · 交给第 1 棒（M1 数据先行）

> **命名约定**：`plan/HANDOFF-M<k>.md` = **交给第 k 棒的入口交接单**（由第 k-1 棒收尾时写）。
> 第 k 棒收尾时在本文件补 DoD 结果，并**新建** `HANDOFF-M<k+1>.md`。布局门 `LAYOUT.HANDOFF_ENTRY`
> 断言"当前棒必须有入口交接单"，`LAYOUT.HANDOFFS` 断言"已 ✅ 的棒有七节交接单"。
>
> 本文件由**阶段 0（装订 / 骨架期）收尾**产生。上一棒没有里程碑号（阶段 0 豁免），
> 故不前移为 `HANDOFF-M0`。

---

## 一、当前进度 + 证据

| 项 | 状态 | 证据（本机实测，2026-10-27） |
|---|---|---|
| 阶段 0 方法论装订 | ✅ | `AGENTS.md` + `plan/00` 顶部装订行 + 两门在仓库内 |
| 计划套件 `plan/00`–`06` | ✅ | 7 份齐；`LAYOUT.SECTIONS` 7/7 |
| `data/README.md` 台账（口径唯一源） | ✅ | §5 已定义 tick 容差 / 阈值 / gray 闭集 / 退出码 / 四元组 |
| 代码骨架 | ✅ | `cmd/`、`internal/{policy,report,isobmff,rules,archtest}`、`tools/fixture-gen` |
| 两道门 + 变异验证 | ✅ | `check-sprint-layout`（11 判据）/ `check-desensitized`（4 规则）/ `verify-layout-gate`（21 组） |
| CI 工作流 | 🟡 已入库**未首推** | `.github/workflows/ci.yml`；**不得**写成"CI 绿" |
| 本地等价校验 | ✅ | `node tools/tasks.mjs all` → exit 0（29 PASS） |
| 干净克隆 + tarball 双形态 | ✅ | clone 全绿；`git archive` 解包后门全绿、`check-ignore-rules` 显式 SKIP |
| 安全扫描 | ✅ 0 发现 | 静态扫描（`static_only_no_runtime_execution`） |
| **M1 实现** | ✅ **已完成** | 见 §六 DoD 结果；产物 = 92 例夹具 + 22 条真实样本登记 |

提交：`666deaa 骨架期：计划套件、方法论装订、代码骨架与两道门`（M1 收尾提交见 `git log` 末条）。

---

## 二、当前事实 · 状态不变量（下一棒动手前必须知道，否则一定踩）

| # | 事实 | 后果 / 禁区 |
|---|---|---|
| F1 | `internal/rules` 五个模块是**桩**，`Audit()` 返回 `ErrNotImplemented`，CLI 映射退出码 3 | **不要**改成"返回空 findings"——那是冒充 PASS，会打挂 `TestSkeletonModuleIsNotImplemented` |
| F2 | `internal/policy` 是口径常量的**唯一实现处**；`TestR3PolicyConstantsHaveSingleSource` 扫全仓 `const` 定义 | 在第二个包再写一份 `TickTolerance` 之类即红 |
| F3 | `go.mod` 的 `require` 必须为空、`go.sum` 必须不存在（`TestG8`） | 加任何三方依赖即红（含测试库） |
| F4 | `internal/` 与 `cmd/` 禁 `net*` import（`TestG7`） | 想加遥测/版本检查 = 直接违反产品红线 |
| F5 | 里程碑状态有 **3 处副本**（`plan/00` 里程碑表 / `plan/05` 状态表 / `README` 状态表） | 翻状态必须三处同一提交改；`plan/00 §四` 的提交材料表**不是**状态表（门按表头定位） |
| F6 | `fixtures/out/manifest.json` **必须入库**；`fixtures/out/*.mp4` 不入库 | `.gitignore` 里只能写 `/out/`（顶层），写成 `out/` 会把 manifest 一起忽略（已实测踩过） |
| F7 | 变异验证脚本会**临时改写仓库文件**，靠 `compose` 回滚 | 手工跑前先 `cp` 备份；若中途 `Ctrl-C` 可能残留（详见 `plan/06 §六 B2`） |
| F8 | `tools/fixture-gen` **禁** `time` / `os.Getenv` / `os/exec` / `math/rand`（`TestGeneratorHasNoClockEnvOrExec`） | 想"用 ffmpeg 造样本"就撞 D4：真值字节不得依赖工具链版本 |
| F9 | 骨架期 `manifest.json` 只有 **C1–C3 三例**（**M1 收尾后已重生成为 92 例**） | 任何生成器改动后必须重生成并 commit，否则 `--check` 与 CI 会不一致 |
| F10 | 文档里**不得**出现 `make <target>` 形式（本机无 make） | 可执行等价物一律 `node tools/tasks.mjs <target>` |

---

## 三、下一棒待办（M1 · 数据先行）

顺序即依赖顺序，前一项不做完不要碰后一项。

1. **补 C4–C7 构造器**（`tools/fixture-gen/gen/gen.go`，按现有 `Builder` 形态追加）
   - **C4a**：`mvhd.duration` ≠ `Σ(stts.count × stts.duration)`（含 2× 形态，对应 S7）
   - **C4b**：`Σelst` 或首帧 PTS 使 "起点 + 时长" 越过声明轨道时长
     ⚠️ **不得写负数字段**：`mvhd/tkhd.duration` 是无符号整数；落盘必须写明 a 还是 b
   - **C5**：`stts` 插入 `duration` 使累计 DTS 回退（or 分片 `tfdt` 倒退）
   - **C6**：音频首帧 PTS 后移 **1024** 样本且 `elst` 未裁剪
   - **C7**：`tkhd` matrix 置 ±90°，`width < height`，删 `pasp`
2. **对照组**（"不误伤"是门禁，不是加分项）
   - `m3-regression`：**≥20** 个正常 H.264/H.265（**必须含 B 帧**，否则 `ctts` 负偏移的坑测不出来）
   - `m4` / `m5`：正反例各 **5**
   - `controls`：每类至少 1 个合法样本
3. **重生成 manifest 并入库**：`node tools/tasks.mjs fixtures` → `git add fixtures/out/manifest.json`
   → `node tools/tasks.mjs check-fixtures`
4. **真实样本登记**：`data/real-samples.json` 补 ≥20 条（`sha256` + `source_url` + `license`）；
   样本本体**不入库**；未复跑的写 `"verified": null`（不写 `true`）
5. **文档状态翻转**：`docs/fixture-pools.md`（C1–C7 与对照组状态列）、`plan/06 §三` 缺口表首行
6. **M1 收尾**（顺序固定）：DoD 全验 → 三处状态副本同提交翻 ✅ → **新建 `plan/HANDOFF-M2.md`**
   → 在**本文件**补 DoD 结果 → commit

> **不要**在 M1 里顺手写 M2 的判定规则。M1 的产物是**带真值的样本**，不是判定器。

---

## 四、既定口径清单（K 编号 · 动了会打挂测试）

| # | 约定 | 落在哪 | 违反时谁会红 |
|---|---|---|---|
| **K1** | 数值口径只在 `data/README §5` 定义、只在 `internal/policy` 实现 | `internal/policy/policy.go` | `archtest` R3 |
| **K2** | 三态 `pass/fail/gray`；`gray` 永不折算 pass、永不升级 fail | `internal/report` | G1/G2/G3 单测 |
| **K3** | `gray.reason` 是**封闭集合**；新增先改 `data/README §5.5` + `docs/report-schema.md` | 同上 | `TestGrayReasonMustBeClosedSet` |
| **K4** | 解析失败是 **error（退出码 3）**，不是 `gray` | `cmd/mediatrace` | `TestAuditUnparsableFileIsUsageError` |
| **K5** | 每条 `fail` 必须四元组齐（offset / field_path / values / delta） | `internal/report` | `ValidateAll` 拒绝 + G1 |
| **K6** | `elst` 按**文件内顺序**编号，**不排序**；`colr` 多实例同理 | M2 / M5 实现 | 无自动门——写测试时自证（排序会掩盖"段次序错"） |
| **K7** | 模块之间**不得**互相 import；规则是纯函数（无 I/O / 时钟 / 随机 / 网络） | `internal/rules` | `TestModulesDoNotImportEachOther` |
| **K8** | 生成器与解析器**不共享代码**（只共享 `policy` 常量） | `tools/fixture-gen` | 代码评审；`--check` 不证此事 |
| **K9** | 同输入双跑**逐字节一致**（含 `--modules` 子集） | 全链 | `check-fixtures` / CI `diff -r` |
| **K10** | 文档里的命令必须是**能跑的形态**（`node tools/tasks.mjs …`），不是 `make …` | `plan/` `docs/` `README` | 部署期人工核对；`LAYOUT.DOC_TOOLS` 只管脚本存在性 |

---

## 五、本机环境坑（只写实测过的）

| # | 坑 | 实测表现 | 处置 |
|---|---|---|---|
| E1 | **Go 不在 `PATH`** | `go: command not found` | 实际在 `%USERPROFILE%\go-sdk\go\bin\go.exe`；`tasks.mjs` 自动探测；也可 `GO=… node tools/tasks.mjs test` |
| E2 | **本机无 `make`** | `make: command not found` | 一律 `node tools/tasks.mjs <target>`；`Makefile` 只是薄委托 |
| E3 | **Windows 可执行必须 `.exe`** | `spawnSync bin/mediatrace` = ENOENT（文件其实在） | `tasks.mjs` 按平台拼名；CI 的 Linux 产物不带后缀 |
| E4 | **`git check-ignore` 的目录语义** | 父目录被忽略后，其内文件的 `!` 例外**不生效** | 忽略顶层产物用 `/out/` 而不是 `out/`；已有常驻断言 `check-ignore-rules` |
| E5 | **`flag` 包遇位置参数即停** | `audit <file> --json` 曾直接 exit 3 | 已用 `parseInterspersed` 修；改参数面时**先跑** `TestFlagOrderDoesNotMatter` |
| E6 | Go 本体可用 Node 跑门，但**门不依赖 Go** | 只装 Node 也能跑 `gates` | 这是刻意的：装订门必须在任何机器上可跑 |

---

## 六、DoD 逐项 checklist（M1，取自 `plan/00 §五`）—— ✅ 收尾已验（2026-10-27）

- [x] C1–C7 **七类**各有 ≥1 个可复现样本 — C1/C1b、C2–C2d、C3/C3b、C4a/C4a2/C4a3/C4b、C5/C5b/C5c、C6/C6b/C6c、C7/C7b；共 92 例
- [x] 同 `--seed` 双跑**逐字节一致** — `node tools/tasks.mjs check-fixtures` → 「92 例与入库 manifest 逐字节一致」
- [x] 生成器**零外部进程** — `TestGeneratorHasNoClockEnvOrExec`（AST/import 扫描）；`go test ./tools/fixture-gen/...` 绿
- [x] 控制组合法样本全 `pass` — 构造侧 50 例 `expect_status=pass`；M3 回归 24（含 B 帧 v0/v1）、M4 5+5、M5 5+5、通用 controls 16。**运行时误报 0 待 M2–M5 复核**
- [x] `manifest.json` 入库且 `--check` 与入库版本一致 — 92 例（defects 32 / controls 16 / m3-regression 24 / m4 10 / m5 10）
- [x] `data/real-samples.json` ≥20 条（含 `license`；未复跑 `verified: null`）— 22 条，全部 `verified: null`
- [x] `docs/fixture-pools.md` 状态列与实现一致（**不写**"判定已实现"）— C1–C7 + 三组对照均标「构造器✅ / 判定待 Mx」
- [x] 三处状态副本同提交翻 ✅ + `plan/HANDOFF-M2.md` 已写
- [x] `node tools/tasks.mjs all` 全绿

**M1 期间发现并已修的脚本缺陷**：`tools/verify-layout-gate.mjs` 的变异靶子原先把 M1 写死
（`flipM1()` / `hide("plan/HANDOFF-M1.md")`），M1 真收尾翻 ✅ 后**自崩**（正是文件头警告的
"靶子依赖进度状态"）。已改为**从仓库状态现算**（当前棒 / 最后一个 ✅ / 发布里程碑），
任一进度下都指得到真实可变的行与文件；本机复跑 21/21 全绿。详见 `plan/06 §六 B8`。

---

## 七、关键命令速查

```bash
# 门（新对话第一件事）
node tools/tasks.mjs gates              # 布局门 + 脱敏门
node tools/tasks.mjs verify-gates       # 两门自己的变异验证（21 组，逐组必须按预期变红）

# 夹具
node tools/tasks.mjs fixtures           # 生成 fixtures/out + manifest.json
node tools/tasks.mjs check-fixtures     # 双跑逐字节 + 与入库 manifest 比对
go run ./tools/fixture-gen -list        # 只打印 case id / sha256 / size
go run ./tools/fixture-gen -suite defects -out /tmp/fx

# 代码
node tools/tasks.mjs test               # go vet + go test ./...
node tools/tasks.mjs bin                # 构建 bin/mediatrace（Win: .exe）
node tools/tasks.mjs check-ignore-rules # gitignore 三态（台账入库/样本不入库）
node tools/tasks.mjs all                # 以上一键

# 单文件审计（骨架期返回 3，因模块未实现）
bin/mediatrace audit <file> --json
```

**口径只查** `data/README.md §5`；**计划状态只查** `plan/00 §五`；**决策只查** `plan/00 §二`。
