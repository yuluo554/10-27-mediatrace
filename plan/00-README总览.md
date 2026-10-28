# 00 · README 总览 — 10-27-mediatrace（媒体时间轴语义审计）

> 方法论：**`ai-tool-project-sprint`** ｜ 布局门：**`node tools/check-sprint-layout.mjs`**（另有 `node tools/check-desensitized.mjs`）｜ 装订日期：**2026-10-27**
>
> **新对话第一件事 = 跑布局门**（`node tools/tasks.mjs gates`；本机无 `make`）。门红先修布局，再干活；`AGENTS.md` 是仓库锚点。

> 方法论：出题阶段 4（骨架期）｜续接对话第一件事 = 读本文档 **§五 里程碑表** 定位当前棒次（§六 是续接操作步骤）

- **性质**：不参赛项目题 · 发布 GitHub **公开仓库** · 署名 yuluo554
- **产出**：仓库 `10-27-mediatrace`，命令入口 `mediatrace`；**纯本地单二进制 CLI，只读字节、不解码、运行时不依赖 ffmpeg**
- **状态**：**M1–M6 全部收尾，v0.1.0 已发布**（<https://github.com/yuluo554/10-27-mediatrace>）：五个判定模块 M1–M5 全部实现并对 92 例夹具对账；公开仓库 + `v0.1.0` annotated tag + Release（三平台单二进制 + `SHA256SUMS`，从 tag 树重建、真下载复核）+ 发布留档 `plan/RELEASE-M6.md`；CI 首推全绿（tag 提交 run `40630946597` = 6/6）
- **定位宣言**："ffprobe、MediaInfo、MP4Box、Bento4、mkvalidator 各给一份'我看到的样子'，彼此不对话；mediabunny 明知 `elst` 有它不支持的指令却只 `console.warn`；唯一做过策略判定的 MediaConch 已于 2021 停更，且只查格式合规不查同文件内部字段互相矛盾。**所有人都在'打印'，没有人在'判定'。**"
- **基准日**：2026-10-27（代理实际检索窗口 **2026-10-27T10:56Z–11:22Z**；标签 `10-27` 是系列编号，不是检索日期）

---

## 一、文档索引表

| 文档 | 内容 | 状态 |
|---|---|---|
| `AGENTS.md` | 仓库锚点：方法论 / 强制布局 / 门命令 / 环境事实 | ✅ 装订期 |
| `plan/00-README总览.md` | 本文档：索引 / 决策记录 / 弃选项 / 承诺-证据台账 / 提交材料实物表 / 里程碑 | ✅ v1（骨架期） |
| `plan/01-题目文档.md` | 十段式题目定稿（头部 / 介绍 / 需求证据 / 空白检索 / 任务 / 验收门槛 / 提交材料 / 评分 / 锚点 / 边界） | ✅ v1 |
| `plan/02-架构与技术选型.md` | 分层架构 / 盒子解析 IR / 依赖与单二进制度量 / 退出码 / 确定性纪律 | ✅ v1（骨架期） |
| `plan/03-模块详设.md` | M1–M5 接口与测试点 + 跨模块门禁清单 G1–G10 | ✅ v1（骨架期） |
| `plan/04-数据计划.md` | 合成缺陷集 / 真实样本池 / fixture schema / 真值标签口径 | ✅ v1（骨架期） |
| `plan/05-里程碑与发布.md` | M1–M6 DoD / 出口门 / 依赖图 / 已知未验证项 | ✅ v1（骨架期） |
| `plan/06-交付对标与决策.md` | 提交材料对照 / 评分对照 / 缺口 / 风险登记 | ✅ v1（骨架期） |
| `plan/HANDOFF-M1.md` | **交给第 1 棒（M1 数据先行）的入口交接单**（七节：进度+证据 / 当前事实·状态不变量 F1–F10 / 待办 / K1–K10 / 环境坑 E1–E6 / DoD / 命令速查）；**M1 收尾已补 DoD 结果** | ✅ M1 |
| `plan/HANDOFF-M2.md` | **交给第 2 棒（M2 结构判定）的入口交接单**（七节齐）；**M2 收尾已补 DoD 结果** | ✅ M2 |
| `plan/HANDOFF-M3.md` | **交给第 3 棒（M3 时长口径判定）的入口交接单**（七节齐）；**M3 收尾已补 DoD 结果** | ✅ M3 |
| `plan/HANDOFF-M4.md` | **交给第 4 棒（M4 时间戳判定）的入口交接单**（七节齐）；**M4 收尾已补 DoD 结果** | ✅ M4 |
| `plan/HANDOFF-M5.md` | **交给第 5 棒（M5 偏移与显示判定）的入口交接单**（七节齐）；**M5 收尾已补 DoD 结果** | ✅ M5 |
| `plan/HANDOFF-M6.md` | **交给第 6 棒（M6 发布）的入口交接单**（七节齐） | ✅ M5 收尾时写 |
| `plan/RELEASE-M6.md` | 发布留档（§一 tag 前事实 / §二 tag 后追加 / §三 阶段 7 出口门覆盖对账） | 🟡 §一 已落盘（待打 tag） |
| `README.md`（仓库根） | 用户向：定位 / 快速开始 / 退出码 / 隐私 | ✅ 骨架期（M5 定稿措辞） |
| `LICENSE` | MIT | ✅ 骨架期 |
| `data/README.md` | 依据查证三态 + 技术事实核实台账 + 数据源可获取性 + 合成数据纪律 + **数值口径单一定义** | ✅ v1 |
| `data/real-samples.json` | 真实样本登记表（sha256 + URL + 许可；样本本体不入库） | ✅ M1（22 条，全部 `verified: null`） |
| `docs/rules.md` | 每条规则的人类可读说明与反例 | ✅ M5（模块 M1–M5 的规则与反例样本 id 全覆盖） |
| `docs/report-schema.md` | `pass/fail/gray` 三态 JSON 结构 + 首屏三段 + 退出码 | ✅ M5（定稿；首期不做 `--suite`，`GATE` 段不出现） |
| `docs/fixture-pools.md` | 强制构造 C1–C7 的生成纪律 | ✅ M5（C1–C7 构造器全实现 + 四组对照；M1–M5 五个模块均已对账） |
| `tools/check-sprint-layout.mjs` | 布局门（11 条判据） | ✅ 骨架期 |
| `tools/check-desensitized.mjs` | 脱敏门（4 条规则） | ✅ 骨架期 |
| `tools/verify-layout-gate.mjs` | 两门自己的变异验证（21 组，每组必按预期变红） | ✅ 骨架期 |
| `tools/tasks.mjs` | **跨平台任务入口**（本机无 `make`；`Makefile` 只是薄委托） | ✅ 骨架期 |
| `fixtures/out/manifest.json` | 夹具真值台账（样本字节不入库，manifest 入库） | ✅ M4（92 例：defects 32 / controls 16 / m3-regression 24 / m4 10 / m5 10；结构判定、时长判定与时间戳判定均已对账） |
| `.github/workflows/ci.yml` | CI：layout / test 矩阵 / fixtures / cross | ✅ 骨架期（**未首推**） |

> 状态图例：✅ 已落盘 · 🟡 进行中 · ⬜ 待做。**状态列必须与里程碑同步翻转（同一提交里改）。**

---

## 二、决策记录表

| 日期 | 决策 | 结论与理由 |
|---|---|---|
| **2026-10-27** | **D1·判定对象是"同一文件内的多口径互斥"，不是"格式合规"** | MediaConch 走的路线是**策略合规**（FFV1/MKV 规范符合性），而本题要判的是"这个文件的时间轴自己跟自己打不打架"。两者输入相同、结论完全不同 → **不重走 MediaConch 路线**，也避免与其（虽已停更）历史定位重叠。 |
| **2026-10-27** | **D2·"判不准"必须是 `gray` 一等公民** | 现有工具的二值输出迫使实现要么猜（产出假 fail）、要么静默（`console.warn` 的形态）。本项目设三态 `pass/fail/gray`，`gray` 带 `reason` 枚举、可导出、**不计入 fail 数**。同时禁止把解析失败降级成 `gray`（那是 `error`）。 |
| **2026-10-27** | **D3·只读字节、不解码，运行时不依赖 ffmpeg** | 一旦解码，就滑向 `av-drift-detector`（0★，需解码）与"跨工具一致性预言"（不可复现）。故运行时零 ffmpeg 依赖；ffmpeg/GPAC **只在 CI 用于造测试真值**（`-itsoffset`/`-copyts`/`-edits`），绝不进交付二进制的运行路径。 |
| **2026-10-27** | **D4·首期五模块固定，模块可单独运行** | M1 结构 / M2 时长口径 / M3 时间戳 / M4 偏移与 priming / M5 旋转与色彩标记。每模块独立入口（`--modules`），避免"要跑全流水线才能看一个结论"。 |
| **2026-10-27** | **D5·MediaConch 的停更日期是硬事实，但其"策略能力边界"只是中等置信** | ✅ **硬事实**：MediaArea/MediaConch 最后 push **2021-09-29**（33★，SourceCode 39★）——这足以支撑"唯一做过策略判定的工具已停更五年"。⬜ **中等置信**：其策略文件（schema/Policy）的**能力边界未逐条读源码**，故题面只写"偏格式合规"，**不写**"它明确不能做 X"这类逐条断言。 |
| **2026-10-27** | **D6·mkvtoolnix 的可获取性阻断须原样记录** | ✅ `mkvtoolnix` 的 GitHub 仓库**返回 404**（项目已迁 Codeberg）；⬜ Codeberg 侧**本次未能访问**（mbunkus 仓库），⬜ 因此 **`mkvalidator` 从未被亲自运行过**。题面凡涉及 `mkvalidator` 一律标"存在但未验证"，不得写成"已确认其能力边界"。MKV 侧因此只做 M1 的结构对应项。 |
| **2026-10-27** | **D7·E4/E5 类低星证据疑与本次调研同源，不得单独支撑结论** | 弱证据（原调研编号 **E4/E5**，本文档统一记作 **W1/W2**：`storytold/soundcraft` #707、`storytold/filmcraft` #1276）均为 **0–1★** 仓库，且**两个仓库名形如 `storytold/*`，与已知刷星/同源生成模式吻合**——**疑与本次调研同源**。故：①证据强度在 `plan/01 §3.2` 明确标"弱-中"；②**不单独支撑任何规则的存在必要性**（每条规则另有源码级或强证据支撑）；③若将来发现同源确证，规则不受影响，只删这两条引用。 |
| 2026-10-27 | **D8·竞品星数只在题面快照，不进代码与输出** | 星数/分支数是**检索窗口快照**（2026-10-27），会漂移。写入 `plan/01 §2.1` 与 `data/README §1.4–1.5`，**不写入任何源码、不进报告默认输出**。 |
| 2026-10-27 | **D9·数值口径单点定义** | 时长容差（1 tick）、timescale 白名单/极小值阈值、正交性容差（1/65536）、gray 判定条件、退出码 —— **只在 `data/README.md` §5 定义一次**；`plan/01`、`plan/02`、代码全部引用不复述。复核方式：`grep` 口径常量只应命中 `data/README.md` 与实现该口径的单一模块。 |
| 2026-10-27 | **D10·`elst` 条目按文件内顺序编号，不排序** | 排序会掩盖"段次序错了"这类真实缺陷（S1 的 3 段 `elst` 正需要按序对账）。同理 `colr` 多实例按序全列，**不取第一个**。 |
| 2026-10-27 | **D11·"gray 不兜解析错误"** | `gray` 只能由**已知的语义不确定**触发（`no_roll_distance` / `unknown_codec_priming` / `media_rate_not_one` / `no_edts` / VFR）。盒子越界、字段缺失属 `error`/`fail`。**否则 `gray` 会变成"什么都判不了"的垃圾桶**，D2 的承诺落空。 |
| 2026-10-28 | **D12·时长互比的公共时基取影片时基 `mvhd.timescale`** | 时长三口径里 `mvhd`/`tkhd`/`elst` **原生**写在影片时基；媒体侧（`mdhd.duration`）只能向下取整换算过来（量化误差 < 1 影片 tick，落在 1 tick 容差内）。反向（影片→`mdhd`）会把粗时基量化误差放大 `mdhd.timescale/mvhd.timescale` 倍，把逐样本取整的**合法**文件判成超容差（M3 实测：夹具 `M3R23` `tkhd`=33@1000、`mdhd`=3000@90000，反算差 30 个媒体 tick —— 那是假阳性）。故 `data/README §5.1` 的对齐时基由"`mdhd.timescale`"细化为"**公共时基 = `mvhd.timescale`**"，并把判定基准明确为 `mdhd.duration`（声明值）而非 `Σstts`（逐样本取整，只作证据原值报出）。 |

### 弃选项（写题期评估后不采用）

| 弃选项 | 为什么弃 | 去向 |
|---|---|---|
| **C2·"容器结构自洽"独立成题** | 单独立项太窄：结构自洽本身只值一两条规则，且 `AISOBMFF Box Inspector`（15★）已在做盒子检查；但它是所有语义判定的**前置**，不能没有。 | **降级为 M1** |
| **C3·"旋转 / HDR / 色彩标记"独立成题** | 单独立项太窄，且"像素实际色彩是否正确"**只读元数据无法确证**（须解码或读帧），与 D3 冲突。 | **降级为 M5**，只做"matrix 正交性 / 呈现宽高 / SAR / 元数据量程与互斥"这类**纯字节可判**的部分 |
| **C4·"跨工具一致性预言"（同一文件在 mpv/Chrome/QuickTime 表现是否一致）** | 三条硬伤：①违反"只读字节、不解码"前提；②需拉多平台播放器，**跨平台不可复现**；③结论**非确定性**（版本/GPU/系统解码器），无法做 precision/recall 门禁；④与"从黑盒行为反推规则"的形态相邻（那类工具不判文件自身、只从外部表现反推）。 | **弃选**；但把"**同一文件内两条时长口径不一致**"吸收进 **M2**（纯字节可判） |
| **媒体标签 / 元数据审计** | AtomicParsley（771★）只读写 iTunes 标签、mutagen（1981★）已覆盖；本题的时间轴判定与标签无关。 | 弃（写进 `plan/01 §十` 非目标） |
| **损坏文件修复** | `untrunc`（1954★）已占，且是**写操作**、需参考文件；审计器写回即失去第三方可复核立场。 | 弃 |
| **纯 MKV 校验器** | `mkvalidator` 已存在（`foundation-source` 67★，**未亲自运行**），**单格式太窄**；MKV 时间轴语义留二期。 | 弃（MKV 仅作 M1 结构对应项） |
| **A/V 同步测量** | `av-drift-detector`（0★）方向已占，且**必须解码**才能测漂移；本题只在元数据层判"声明之间是否自洽"。 | 弃 |
| **"哪个工具算错了"的裁决** | 点名 ffprobe/MediaInfo 谁错需要跨工具复现，回到 C4 的坑；且上游行为会升级变化。 | 弃（只报"本文件内 N 条口径互斥 + 差值"） |
| **LLM 辅助判定** | 判定必须可复算、可做门禁；LLM 不可复现。 | 弃（全规则化） |

---

## 三、承诺-证据台账（全项目唯一一份）

> 规则：**只此一份**，其他文档只引用不复述；状态列必须与里程碑同步翻转。状态：✅ 已达 · 🟡 进行中 · ⬜ 待做。

| 承诺项 | 素材来源 | 交付物内可达性 | 核验方式 | 状态 |
|---|---|---|---|---|
| 单二进制 CLI，运行时不依赖 ffmpeg | 设计 D3 | 纯本地，无外部进程 | `bin/mediatrace --version`（实测）；零三方依赖 + `internal/`/`cmd/` 零 `net` import（`archtest` G7/G8 常驻断言）→ 判定路径不调用 ffmpeg、不链接 `libav*` | ✅ M2 |
| 盒子树迭代遍历（不递归），`size==1`/`size==0` 展开正确 | ISO BMFF 布局（`plan/01` 锚点 1/3/7） | 纯本地 | 深盒树（6000 层）不爆栈单测；三例 M1 验收 3/3、整夹具集**集合相等**（对照 60 例误报 0） | ✅ M2 |
| 双 `moov` = fail（不取首/末了事） | 设计 D1/M1 | 纯本地 | 合成"双 moov"必命中；报告列出两处偏移（`C1`/`C1b`） | ✅ M2 |
| `stco+stsz` 累计区间 ⊆ `mdat` | 设计 M1 | 纯本地 | "stco 越界"（`C2`–`C2d`）/"截断 mdat"（`C3`/`C3b`）逐例命中 | ✅ M2 |
| 三口径时长互比（`stts` / `mvhd`+`tkhd` / `Σelst`）容差 1 tick | `data/README §5` | 纯本地 | S1 类"copy 前后差"复现；差 2× 样本（S7 类）命中 | ✅ M3 |
| `elst` 条目按序编号，`media_rate≠1` 定位到条目号 | 决策 D10；mediabunny #494 原话 | 纯本地 | GPAC `-edits` 造 3 段（含 `media_rate≠1`）逐条定位 | ✅ M3 |
| DTS 单调 + 位宽溢出前兆判定 | S5 / S10 / S4 | 纯本地 | "回退 DTS"合成例命中并给样本序号；≥20 正常 H.264/H.265 文件误报 0 | ✅ M4 |
| priming / encoder delay 报值与期望比对 | S3 原话；`sgpd`/`sbgp` roll | 纯本地 | "1024 samples 晚"报出 1024；正反例各 5 个 | ✅ M5 |
| matrix 正交性 + 呈现宽高 + SAR | 锚点 5（ISO 14496-12 §6.2.2）；S11 | 纯本地 | "matrix=±90 且 width<height 无 pasp"命中；正常竖屏不报 | ✅ M5 |
| `colr`/`mdcv`/`clli` 量程互斥 + 缺失落 gray | 锚点 6/8 | 纯本地 | bt2020 无 `mdcv`/`clli` → `gray` 且可导出 | ✅ M5 |
| 每条 fail 带 偏移+字段路径+值+差值 | 设计 | 报告结构 | `docs/report-schema.md` 校验测试：四字段缺一即测试红 | ✅ M5 |
| 三态 + `gray.reason` 枚举 + 不计入 fail | 决策 D2/D11 | 报告结构 | 计数一致性测试；`gray` 出现 `parse_error` 类 reason 即测试红 | ✅ M5 |
| 报告同源（先算一次再渲染 MD/JSON） | 设计 | 纯本地 | 两份渲染的 finding 集合按 fingerprint 相等 | ✅ M5 |
| 退出码语义 | `data/README §5` | CLI 契约 | 各退出码各一条测试（进程级 + `main()` 级） | ✅ M5 |
| 确定性：同输入两次运行逐字节一致 | 设计 | 纯本地 | 双跑 diff 为空（含 `--modules` 子集） | ✅ M5 |
| 纯本地零网络 | 红线 | 无网络客户端依赖 | AST/符号扫描 0 命中 | ✅ M5 |
| 合成缺陷集 ≥25 例 + 真实样本 ≥20 例 | `data/README §4` | 生成器随包 | 门禁三数打印在首屏 | ✅ M1（合成 92 例、真实登记 22 条；真实样本**复跑**待各判定模块） |
| 发布三件套：tag + Release 附产物 + 发布留档 | 发布纪律 | 公开仓库 | `plan/RELEASE-M6.md` 两栏 + 覆盖对账齐；`git cat-file -t v0.1.0` = tag；`gh release view` 四附件 | ✅ M6 |

**台账外的"永不进判定路径"项**（写清以免被当欠债）：MKV 时间轴语义（`TimestampScale`/`Cluster`/`DefaultDuration` 交叉校验）⬜ 二期；分片 `sidx` ↔ `moof` 自洽 ⬜ 二期；逐样本 PTS/DTS 全量导出 ⬜ 二期；`mkvalidator` 能力边界 ⬜ 未验证（D6）。

---

## 四、提交材料实物表（承诺 → 文件路径 → 里程碑 → 核对命令）

| 承诺项 | 应存在的文件路径 | 由哪个里程碑产出 | 用哪条命令核对 |
|---|---|---|---|
| 单二进制 CLI 入口 | `cmd/mediatrace/main.go` + `Makefile`（委托 `tools/tasks.mjs`） | M2 | `git ls-files cmd/ Makefile tools/tasks.mjs`；`node tools/tasks.mjs bin` |
| 盒子解析核心（迭代遍历 + IR） | `internal/isobmff/` | M2 | `git ls-files internal/isobmff/` |
| M1 结构规则 | `internal/rules/structure/` | M2 | `git ls-files internal/rules/structure/`；`mediatrace audit X --modules M1` |
| M2 时长口径规则 | `internal/rules/duration/` | M3 | `git ls-files internal/rules/duration/` |
| M3 时间戳规则 | `internal/rules/timestamps/` | M4 | `git ls-files internal/rules/timestamps/` |
| M4 偏移/priming 规则 + M5 显示与色彩规则 | `internal/rules/offset/`、`internal/rules/display/` | M5 | `git ls-files internal/rules/` |
| fixture 生成器（C1–C7 强制构造） | `tools/fixture-gen/` | M1 | `git ls-files tools/fixture-gen/`；`node tools/tasks.mjs fixtures` → 复算 sha256 与 manifest 比对 |
| 报告渲染（JSON + Markdown 同源） | `internal/report/` | M5 | `git ls-files internal/report/`；`mediatrace audit X --json` |
| 报告格式文档 | `docs/report-schema.md` | M5 | `git ls-files docs/report-schema.md` |
| 规则表（人类可读 + 反例） | `docs/rules.md` | M2 | `git ls-files docs/rules.md` |
| 强制构造纪律（C1–C7） | `docs/fixture-pools.md` | M4 | `git ls-files docs/fixture-pools.md` |
| 用户 README | `README.md` | M5 | `git ls-files README.md` |
| 发布留档 | `plan/RELEASE-M6.md` | M6 | `git cat-file -e <tag>:plan/RELEASE-M6.md` |

> **每一项都必须指得到一个文件**；指不出的现在就删，不留到评分表里变成无主承诺。

---

## 五、里程碑表（阶段编号 M1–M6；与模块 M1–M5 的对应关系见"对应模块"列）

> ⚠️ 命名说明：**阶段**与**模块**都用 `M` 编号，但编号**不一一对应**。文中凡写"阶段 M2"，指里程碑表里的"结构判定"；凡写"模块 M1"，指题目文档 §五 的模块。此表"对应模块"列是唯一权威映射。

| 阶段 | 内容 | 对应模块 | DoD（完成判据） | 状态 |
|---|---|---|---|---|
| **M1 · 数据先行** | **先造真值，再写判定**：用 `ffmpeg -itsoffset` / `-copyts` 与 GPAC `-edits` 程序化生成**已知缺陷**文件，逐例登记真值标签；落 `tools/fixture-gen/` 与 `docs/fixture-pools.md` | 无（前置） | 强制构造类（题面原文记作"六类"，`data/README §4` 实列 **C1–C7 共 7 项**：双 `moov` / `stco` 越界 / 截断 `mdat` / 时长口径互斥（题面记作"负 duration"，实为 C4a/C4b 两态，**不能写负数字段**）/ 回退 DTS / 1024 samples 延迟 / `matrix=90` 无 `pasp`）各有 ≥1 个可复现样本；同 seed 逐字节一致；**ffmpeg 只在 CI/生成期使用**，交付二进制零依赖 | ✅ |
| **M2 · 结构判定** | 盒子树迭代遍历 IR + 模块 M1 全部规则（`size` 展开、子盒越界、`moov`/`mdat` 唯一、`stco+stsz ⊆ mdat`、`ftyp` 品牌一致）；MKV EBML 结构对应项顺延（首期夹具与生成器均为 MP4，见 `plan/06 §三` 缺口行） | 模块 M1 | 三例（双 moov / stco 越界 / 截断 mdat）命中 **3/3**、误报 **0**；深盒树不爆栈；每 fail 带偏移 | ✅ |
| **M3 · 时长口径判定** | 模块 M2：三口径互比（1 tick 容差）、`elst` 按序编号、`media_time` 与首帧 PTS、空 edit 语义、`media_rate≠1` 报条目号 | 模块 M2 | GPAC 造 3 段 `elst` 逐条定位；S1 类"copy 前后差"可复现；S7 类 2× 时长命中 | ✅ |
| **M4 · 时间戳判定** | 模块 M3：DTS 单调、`ctts`→PTS、位宽溢出前兆、timescale 白名单/极小值、VFR 落 gray | 模块 M3 | "回退 DTS"命中并给样本序号；**≥20 正常 H.264/H.265 文件误报 0** | ✅ |
| **M5 · 偏移与显示判定** | 模块 M4（priming/encoder delay、起始裁剪）+ 模块 M5（matrix 正交性、呈现宽高、SAR、`colr`/`mdcv`/`clli` 量程与互斥）+ 报告三态与首屏三段 + 退出码 | 模块 M4 + M5 | "1024 晚"报出 1024；正反例各 5 个；matrix=±90 命中且正常竖屏不报；`gray` 可导出且不计 fail；同输入逐字节一致 | ✅ |
| **M6 · 发布** | 脱敏复核 + 公开仓库发布 | — | **DoD 三件套 = `git tag` + GitHub Release（附产物：单二进制 + `SHA256SUMS`）+ 发布留档 `plan/RELEASE-M6.md`（tag 前落盘进 tag 树，含"tag 时点事实 / tag 后追加事实"两栏）**；公开仓库可匿名克隆 | ✅ |

> **DoD 产物口径**：Release 附产物（单二进制 + `SHA256SUMS`）。该口径自首提交起不得降级。
> **tag 时点口径**：`plan/RELEASE-M6.md` 必须在 tag **之前**写好并进 tag 树；tag 后才产生的事实（附件 sha256、真下载复核）作为追加提交，分两栏写。
> **门禁不可绕过**：M2–M5 任一模块的"误报 0"不成立，**不得进入 M6**；不得通过放宽规则换取绿灯（对应 D2/D11）。

---

## 六、续接入口

1. 读本文档 §五，找第一个 `⬜` 的阶段 → 该阶段即当前棒次。
2. **读该棒的入口交接单** `plan/HANDOFF-M<k>.md`（当前是 `plan/HANDOFF-M6.md`）——
   它的 §二「当前事实·状态不变量」是"不读就会踩"的清单，§四 是 K 口径清单。
3. 读 `plan/01` 对应模块的"输入/输出/验收/确定性纪律"四栏。
4. 一切数值口径（容差、阈值、白名单、退出码）**只查 `data/README.md §5`**，不要在其它文档里找第二份。
5. 涉及竞品能力/星数，只引用 `data/README.md §1.4` 与 `plan/01 §2.1` 的快照并注明窗口；**不得现编**。
6. **收尾每一棒**：DoD 全验 → 三处状态副本同一提交翻转 → 新建 `HANDOFF-M<k+1>.md` →
   在本棒交接单补 DoD 结果 → commit。（漏写入口交接单 = `LAYOUT.HANDOFF_ENTRY` 变红。）

---

## 七、装订记录（阶段 0，不占里程碑号）

**装订日期 2026-10-27**。三件齐：

| 件 | 实物 | 核对 |
|---|---|---|
| ① 仓库锚点 | `AGENTS.md` | 含方法论名 + 装订日期 + 强制布局清单 + 门命令 |
| ② 布局门 | `tools/check-sprint-layout.mjs`（11 条判据） | `node tools/tasks.mjs gates` |
| ② 附属 | `tools/check-desensitized.mjs`（4 条规则）+ `tools/verify-layout-gate.mjs`（21 组变异） | `node tools/tasks.mjs verify-gates` |
| ③ `plan/00` 顶部装订行 | 本文档第 3 行 | `LAYOUT.BINDING_HEAD` |

**本机实测（2026-10-27）**：`node tools/tasks.mjs all` → exit 0（29 PASS）；
变异验证 21/21 按预期变红；干净克隆与 `git archive` tarball（无 `.git`）两形态均绿。
**未接远端**——CI 工作流已入库但未首推，一切"CI 绿"的措辞一律写成"待首推验证"。

**装订期抓到的 7 个真实缺陷**（B1–B7）见 `plan/06 §六`，其中 B7 是**门全绿却漏做**的形态
（当前棒缺入口交接单），已补判据 `LAYOUT.HANDOFF_ENTRY`。**M1 收尾又抓到一个（B8**：
变异脚本靶子写死 M1，M1 一翻 ✅ 脚本即自崩——已改为从仓库状态现算，见同节）。
