# data — 依据查证、数据计划与数值口径

**基准日**：2026-10-27。
**代理实际检索窗口**：**2026-10-27T10:56Z–11:22Z**（其中 GitHub 仓库检索 `gh api` 子窗口 **2026-10-27T10:58Z–11:18Z**）。
**标签说明**：仓库名与文档里的 `10-27` 是**系列编号**，不是检索日期；检索窗口以本行 UTC 区间为准。星数与 `total_count` 全部是**该窗口快照**，复跑前请先记录当次时间。
**三态语义**：✅ 已核对（源码/原文/规范条款可直连，或本机可复现）；🟡 中等置信（有来源但边界未逐条确认）；⬜ 未验证 / 软论据（登记路径与阻断原因，**不得写成承诺**）。

---

## 1. 依据查证记录

### 1.1 需求证据（✅ 原话，强证据）

引用纪律：以下只用于证明"需求真实存在"，**不用于推导任何阈值或规范条款**。

| 编号 | 出处 | 原话（节选） | 强度 |
|---|---|---|---|
| S1 | Stack Overflow **Q56468034** | "the resulting output contains incorrect durations… **introducing A/V sync issues**" — `input audio stream duration=21.868776`, `output audio stream duration=23.821769`；输入含 **3 条 `elst`**，media time = 322560 / 645120 | ✅ 强 |
| S2 | **mediabunny #494** | "Mediabunny **logs warnings when an MP4/MOV edit list (`elst` box) contains unsupported instructions**, such as multiple media edits or a `media_rate` other than 1… decoding can succeed while these instructions are **ignored**" | ✅ 强 |
| S3 | **w3c/webcodecs #626** | "the number of **priming samples is not returned by the encoder**, so there's no way to mux the file properly to signal this (e.g., using an **edit list in an mp4**)… This is important for accurate synchronization" | ✅ 强 |
| S4 | **macshot #450** | "Screen recordings made with macshot do not play when sent… **The inline preview autoplays normally, but opening the video in the full-screen player results in an indefinite loading state**"（疑 timescale 错误） | ✅ 强 |
| S5 | **livekit/egress #1434**（标题 `backwards DTS in flvmux`） | "after approximately 15-20 minutes viewers start hearing **audio that is out of sync with video**" | ✅ 强 |
| S6 | **mpv #18691**（mpv 37583★ 快照） | "**invalid BlockDuration in an unselected subtitle track discards the rest of the Cluster**" | ✅ 强 |
| S7 | **MediaInfoLib #2171** | "MediaInfo is showing a **duration that is 2x the actual video duration**… actual 01:00:12.480, MediaInfo gives 02:00:24.960" | ✅ 强 |
| S8 | **video.stackexchange 16692** | "**video files recorded by the camcorder are about 0.0036% shorter** than a stopwatch… the **bad lip-syncing is noticeable**" | ✅ 强 |
| S9 | **video.stackexchange 17448** | "after about an hour there is a **noticeable drift**… **no issue on Linux**, out of sync on every Windows and OSX computer" | ✅ 强（**只作动机**，跨平台差异不进判定路径） |
| S10 | **transmute #315** | "**Video compression fails on VFR screen recordings with long static gaps (mov timescale overflow)**" | ✅ 强 |
| S11 | **Veyra-NRVideo #36** | "**Portrait video (Display Matrix rotation=±90) displayed unrotated** — rotation only affects aspect ratio" | ✅ 强 |
| S12 | **HN 28836727**（2021-10-12）评论 | "wait until you get to the **edit lists part of the MP4 specs**. Some real powerful stuff in that." | ✅ 强 |

### 1.2 弱-中证据（⬜ 0–1★，**不单独支撑任何规则**）

| 编号 | 出处 | 原话（节选） | 强度 |
|---|---|---|---|
| W1 | `storytold/soundcraft` #707 | "AAC/M4A import: the MP4 **edit list is ignored, so the encoder delay stays in** (audio starts 1024 samples late)" | ⬜ **弱**：0–1★ |
| W2 | `storytold/filmcraft` #1276 | "MP4 with a **leading empty edit** (start offset) **renders one frame early**" | ⬜ **弱**：0–1★ |

**同源疑云（必须随证据记录）**：W1/W2 两个仓库名形如 **`storytold/*`**，与已知的刷星/同源生成模式吻合，**疑与本次调研同源**。故：①只作动机；②每条规则另有源码级/强证据支撑；③若同源被确证，仅删这两条引用，规则不受影响（`plan/00` 决策 D7）。

**冷门但真实的讨论场（⬜ 仅说明话题存在）**：HN **43370225**（2025-03-15，2pts）标题 "**MP4 Edit Lists: Precise Video Edits Without Re-Encoding but No One Uses Them**"。

### 1.3 检索失败与限流记录（**必须与"0 结果"区分**）

| 项 | 现象 | 处理 |
|---|---|---|
| `edit list elst mp4` / `fragmented mp4 parser` / `video metadata analyzer` / `mp4 validator` / `ffprobe wrapper` 五条检索式 | 窗口内均返回 **HTTP 403 secondary rate limit** | ⬜ **这是限流，不是"检索结果为 0"**。已作废，**不计入任何论据**（既不当 0 结果，也不当有结果） |
| Reddit JSON | 解析失败 | ⬜ **无 Reddit 原始证据**，不引用 |
| old.reddit.com | 超时 | ⬜ 同上 |
| Stack Overflow 网页（非 API） | 403 | ⬜ 仅能引已知 Q 编号与原话节选（S1），不声称抓取了该页完整内容 |
| `wiki.gpac.io` | 超时 | ⬜ GPAC 文档站点**本次未读**；GPAC 能力结论**只依据源码 `MP4BOX_ARG` 表与 issue**，不依据其 wiki |

> 写作纪律：题面与 README 中凡出现"检索不到/空白"，必须同时指明是"0 结果"还是"被限流/超时"，**不得混用**。

### 1.4 竞品与开源空白检索（⬜ 软论据，快照）

`gh api search/repositories`，子窗口 **2026-10-27T10:58Z–11:18Z**：

| 检索式 | `total_count` | 头部项目（星数为快照） |
|---|---|---|
| `mp4 atom parser` | 12 | macmade/MP4Parse **69★** |
| `mp4 box parser` | 31 | mp4analyzer **17★**、peaBerberian/AISOBMFFWVDFBUTFAII **15★** |
| `mkv ebml parser` | 8 | mkvdump **18★** |
| `mp4 duration` | 106 | music-metadata **1334★** |
| `media conformance checker` | 1 | MediaArea/MediaConch_SourceCode **39★** |
| `mkv validator` | 6 | mkv-validator **6★** |
| `video repair mp4` | 40 | 多为 **0–21★** |
| `mkvatcher` / `mkvalidator` | 4 | Matroska-Org/foundation-source **67★** |
| `video sync analyzer` | 12 | 均为语义视频分析 **0–5★** |
| `hdr metadata parser` | 1 | 无关 |
| `video file auditor` | 5 | Direct-Play-Auditor **0★** |
| `mp4 track offset` | 2 | parse-mp4 **1★** |

**⬜ 0 结果（软论据：无结果 ≠ 不存在，可能只是命名不重合）**：

`media file linter video` 0 · `isobmff validator` 0 · `pts dts parser` 0 · `timeline metadata audit video` 0 · `ffprobe compare mediainfo` 0 · `video player inconsistency test` 0 · `audio priming delay fix` 0 · `elst edit list duration checker` 0

**Issue 侧（`search/issues`，只说明话题反复出现，非全量统计）**：

`"edit list" mp4` 12734 · `"encoder delay" OR "priming samples"` 1848 · `"timescale" mp4` 2736 · `"backwards DTS" OR "non-monotonic"` 83514 · `rotation metadata video` 8278

### 1.5 工具覆盖矩阵（✅ 能力逐项核实；星数 ⬜ 快照，2026-10-27）

| 工具 | 星数 / fork | 它给什么 | 它**不**给什么 |
|---|---|---|---|
| FFmpeg / FFmpeg | 65869 / 14431 | 解码/复用；`ffprobe -show_frames` 给 pts/dts；读 `elst`/`tfdt`/`pasp`/`mdcv`/`clli`/`colr` | **不做一致性判定**；报出的 duration 是"它自己算的那个" |
| mpv | 37583 | 播放 | 无审计输出 |
| VLC | 20243 | 播放 | 同上 |
| iina | 46926 | 播放前端 | 同上 |
| MediaInfo / MediaInfoLib | 2048 / 801（2026-10-07 快照） | 单一工具统一展示全部字段 | **单一权威无交叉验证**（自身 #2171 报 2× 时长） |
| MediaArea/MediaConch（+ SourceCode） | 33（39），**最后 push 2021-09-29** | 策略判定 | **已停更**；策略偏**格式合规**（FFV1/MKV），非"时间轴自洽"；能力边界未逐条读源码 → 🟡 |
| gpac/gpac（MP4Box） | 3317 / 608 | `-diso`/`-dmp4`/`-disox`/`-info`/`-infox` **均为打印**（`MP4BOX_ARG` 表逐条核实） | 不输出"哪条口径与哪条冲突"；`-edits` 是造、不是判 |
| gpac/mp4box.js | 2478 / 392 | 浏览器侧 dump | 同上 |
| Bento4 | 2519 / 530 | `Mp4Dump`/`Mp4Info`/`Mp4Edit`/`Mp4Fragment`/`Mp4Diff` | `Mp4Diff` 是**两文件原子比对**，非单文件规则判定 |
| atomicparsley | 771 / 75 | 读写 iTunes 标签 | **完全不碰时间轴/索引** |
| mkvtoolnix | **GitHub 404**（已迁 Codeberg，本次未能访问） | — | 见 §3 |
| Matroska-Org/foundation-source | 67 | 描述含 `mkvalidator` + `mkclean` | **本次未亲自运行 `mkvalidator`** |
| mediabunny | 7387 / 326（2026-10-26 活跃） | 纯 TS 读写 | 对 `elst` 不支持项**只 `console.warn`**（#494 原话） |
| abema/go-mp4 | 548 | 解析库 | 无规则 |
| alfg/mp4-rust | 356 | 解析库 | 无规则 |
| macmade/MP4Parse | 69 | 解析库 | 无规则 |
| peaBerberian/AISOBMFFWVDFBUTFAII | 15 / 3（2026-06-13） | **最接近的竞品**："MP4/ISOBMFF Box Inspector" | **只做盒子检查，无时间轴语义判定** |
| ponchio/untrunc | 1954 / 260 | **修复**（写操作），需参考文件 | 非审计、非只读 |
| AlexHayton/av-drift-detector | 0 | 需解码测 A/V 漂移 | 与"只读字节不解码"冲突 |

### 1.6 证据索引（可直连形式 / 不可直连项，**不编造 URL**）

只登记**由编号可确定**的链接形式；仓库 owner 未在检索记录中出现者，**不猜 owner**，标注"URL 待补"。

| 编号 | 链接 | 状态 |
|---|---|---|
| S1 | `https://stackoverflow.com/q/56468034` | ✅ 由 Q 编号唯一确定 |
| S8 / S9 | `https://video.stackexchange.com/q/16692` · `https://video.stackexchange.com/q/17448` | ✅ 由 Q 编号唯一确定 |
| S12 | `https://news.ycombinator.com/item?id=28836727` | ✅ 由 item id 唯一确定 |
| HN 43370225 | `https://news.ycombinator.com/item?id=43370225` | ✅ 同上 |
| S3 | `https://github.com/w3c/webcodecs/issues/626` | ✅ owner 已知 |
| S5 | `https://github.com/livekit/egress/issues/1434` | ✅ owner 已知 |
| S6 | `https://github.com/mpv-player/mpv/issues/18691` | ✅ 仓库名在检索记录中（owner 为标准名） |
| S7 | `https://github.com/MediaArea/MediaInfoLib/issues/2171` | ✅ 仓库名在检索记录中（owner 为标准名） |
| W1 / W2 | `https://github.com/storytold/soundcraft/issues/707` · `https://github.com/storytold/filmcraft/issues/1276` | ⬜ owner 已知，但**证据强度弱且疑同源**（§1.2） |
| S2 | `mediabunny #494` | ⬜ **owner 未在检索记录中**，URL 待补；引用时只写 `mediabunny #494` |
| S4 | `macshot #450` | ⬜ owner 未在检索记录中，URL 待补 |
| S10 | `transmute #315` | ⬜ owner 未在检索记录中，URL 待补 |
| S11 | `Veyra-NRVideo #36` | ⬜ owner 未在检索记录中，URL 待补 |

> 纪律：**不得**为了"看起来完整"而补造 owner 或题号。待补项在 `plan/01 §九` 锚点表同样标 ⬜。

---

## 2. 技术事实核实台账（防编造，逐条带来源）

> 纪律：本表之外的技术断言，**不得**写进 `plan/01` 或代码。凡"行号"均为**检索窗口快照**下的位置，随上游版本漂移，只作定位参考。

| # | 事实 | 来源（✅/🟡/⬜） | 原文/形态 |
|---|---|---|---|
| F1 | **`edts` 是容器、内含 `elst`** | ✅ 源码原文 · FFmpeg `libavformat/mov.c` | 同一张 box 表里：`{MKTAG('e','d','t','s'), mov_read_default}`（容器，用 default 读子盒）+ `{MKTAG('e','l','s','t'), mov_read_elst}` |
| F2 | **`elst` 每条含 `media_time` / `duration`** | ✅ 源码原文 · 同上 | `msc->elst_data[i].time` / `msc->elst_data[i].duration`（`mov_read_elst` 内） |
| F3 | **`mfhd` 在 `moof` 内** | ✅ 源码原文 · Chromium `media/formats/mp4/box_definitions.h` | `struct MovieFragment { MovieFragmentHeader header; vector<TrackFragment> tracks; };` |
| F4 | **`tfdt` 在 `traf` 内，语义 = 解码基准时间** | ✅ 源码原文 · 同上 + FFmpeg | Chromium：`struct TrackFragment { … TrackFragmentDecodeTime decode_time; … };`；FFmpeg `mov_read_tfdt` → `frag_stream_info->tfdt_dts` |
| F5 | **旋转用 3×3 matrix、16.16 定点，出自 ISO 14496-12 §6.2.2** | ✅ 规范条款号见 FFmpeg 注释原文 | `mov.c`：`//read in the display matrix (outlined in ISO 14496-12, Section 6.2.2)`、`// 16.16 fixed point` |
| F6 | **`pasp` / `mdcv` / `clli` / `colr` 均为真实盒子** | ✅ 源码原文 · FFmpeg `mov.c` | `mov_read_pasp`(1483) / `mov_read_mdcv`(6967) / `mov_read_clli`(7048) / `mov_read_colr`(2211)（**行号为快照**） |
| F7 | **`stco` / `stsz` / `stts` / `stsc` 在 `stbl` 下** | ✅ 源码原文 · FFmpeg | `mov_read_stco` / `mov_read_stsz`（`stbl` 子盒表内注册） |
| F8 | **不可断言**：`pasp` / `mdcv` / `clli` 的**层级**，以及 `matrix` 字段，**在 mp4ra boxes 页未给出** | ⬜ **明确不可断言** | 本次 WebFetch 对 mp4ra boxes 页明确答 "**no matrix**"。故：**层级与 matrix 的论断只引 FFmpeg 源码**，**不用 mp4ra 推断**；也不写"mp4ra 规定了 X" |

**台账外但必须写明的空白**：

- 🟡 ISO 14496-12 的**原文**（PDF）本次**未逐条读取**，"§6.2.2"这一条款号来自 FFmpeg 注释的转引。若后续要引用规范正文措辞，必须另行获取原文并补充本表。
- 🟡 "AAC priming 常见 1024 / 亦见 2112 / Opus 312" 是**工程观测值**，本次**未取得规范条款**；因此 M4 只判"与声明是否一致"，**不判"必须等于 1024"**（`plan/01 §五 M4`）。

---

## 3. 数据源可获取性记录

| 项 | 结论 | 影响 |
|---|---|---|
| **运行时不依赖 ffmpeg** | ✅ 设计红线（`plan/00` D3） | 交付二进制不调用 `ffmpeg`/`ffprobe`、不链接 `libav*`；核验方式见 `plan/00 §三` 第一行 |
| **CI / 生成期使用 ffmpeg 与 GPAC 造真值** | ✅ 允许（仅在生成与回归阶段） | `ffmpeg -itsoffset` / `-copyts` 造起始偏移与时长偏差；GPAC `-edits` 造多段 `elst`（含 `media_rate≠1`）；**产出的样本与真值标签入库前须经 `sha256` 固定** |
| **真实公开样本（≥20 例）** | ✅ 计划：只读字节、不解码 | 来源与许可须逐条登记（样本本体**不入库**，只入 `sha256` + 来源 URL + 许可）；未登记许可的一律不用 |
| **mkvtoolnix** | ⬜ **GitHub 404**（已迁 Codeberg） | 本次**未能访问**；不引用其源码/文档 |
| **Codeberg（mbunkus 侧）** | ⬜ 未访问 | 同上 |
| **`mkvalidator`** | ⬜ **从未亲自运行** | 题面凡提 `mkvalidator`：只写"存在（`foundation-source` 描述含之）"，**不写能力边界**；MKV 首期不做（非目标），MKV 输入走 exit 3 |
| **mp4ra boxes 页** | ⬜ 已访问但**未给出 matrix / pasp 层级** | 见 F8：**不用它推断** |
| **`wiki.gpac.io`** | ⬜ 超时未读 | GPAC 能力只引源码与 issue |
| **MediaConch 策略源码** | 🟡 未逐条读 | 只写"偏格式合规"，不写逐条能力断言 |

---

## 4. 合成数据纪律

**总则**：所有缺陷夹具**全程序化构造**（直接写盒子字节），不手工编辑、不依赖任何下载样本；`--seed` 决定一切随机选择；同 `--seed` 与同参数生成**逐字节一致**的输出，且生成器**不读系统时间**。

**强制构造类目（题面原文曾记作"六类"；本表实列 **C1–C7 共 7 项**，编号即权威，`plan/00` 与 `plan/01` 均以此为准）**：

| 编号 | 构造类 | 关键手法 | 期望判定 |
|---|---|---|---|
| **C1** | **双 `moov`** | 复制整个 `moov` 盒（含全部子盒）插在第二处 | 模块 M1 `duplicate_moov` = fail，报两处偏移 |
| **C2** | **`stco` 越界** | 单个 chunk offset 指向 `mdat` 载荷之外（或被 `mdat` 尾部截断） | 模块 M1 `stco_out_of_mdat` = fail，报 `stco` 条目号 + 目标区间 vs `mdat` 区间 |
| **C3** | **截断 `mdat`** | 声明 `mdat` size 大于实际剩余字节（或文件尾被砍） | 模块 M1 `truncated_mdat` = fail，报声明 size 与实际可读字节数 |
| **C4** | **时长口径互斥 / 起点晚于时长**（题面记作"负 duration"，**实际不能写负数字段**，见本节末 C4a/C4b） | C4a：`mvhd.duration` 与 `Σ(stts.count × stts.duration)` 不等（含 2× 形态，对应 S7）；C4b：`Σelst` 或首帧 PTS 使"起点 + 时长"越过声明的轨道时长 | 模块 M2 三口径互斥 = fail，报三值与差值（**1 tick 容差**） |
| **C5** | **回退 DTS** | `stts` 中插入 `duration` 使累计 DTS 回退，或分片 `tfdt` 序倒退 | 模块 M3 `dts_regression` = fail，报首个回退的样本序号 + 字节偏移 |
| **C6** | **1024 samples 延迟（priming 未声明 / 不一致）** | 音频首帧 PTS 后移 1024 样本，且 `elst` 未裁剪 / `sgpd`+`sbgp` roll 距离与声明不符 | 模块 M4 `priming_undeclared` / `priming_mismatch`，**必须报出 1024** |
| **C7** | **`matrix=90` 但 `width<height` 且无 `pasp`** | `tkhd` matrix 置 ±90° 旋转，`width`/`height` 保持存储方向，删 `pasp` | 模块 M5 `rotation_dimension_conflict` = fail |

**样本与真值的配比（门槛）**：合成缺陷集 **≥25 例**（C1–C7 每类 ≥1，其余为同类的变体与边界）；真实公开样本 **≥20 例**（只读、不解码）。

**反例（控制）组**：每类必须配**至少 1 个合法样本**，用于证明"不误伤"。其中 M3 的回归集必须 **≥20 个正常 H.264/H.265 文件**（含 B 帧），M4/M5 各 **正反例 5 个**。控制组**误报必须为 0**，否则视为门禁失败。

**许可与隐私**：

- 合成样本全部为**程序化字节**，不含任何真实个人/商户信息、无版权内容。
- 真实样本**不入库**（`.gitignore` 排除 `*.mp4`/`*.mkv`/`fixtures/out/`），只登记 `sha256` + 来源 + 许可。
- 报告默认**不写绝对路径**（只写文件名与偏移）。

**"负 duration"的构造口径（C4 的一半）**：`tkhd`/`mvhd` 的 duration 为**无符号整数**，"负 duration"不能靠写负数字节实现。本项目的 C4 因此分两种形态：**C4a 口径互斥**（`mvhd.duration` 与 `Σ(stts.count×stts.duration)` 不等，含 2× 形态）与 **C4b 起点晚于时长**（`Σelst` 或首帧 PTS 使 "start + duration" 超过声明的轨道时长，等效于"出现负的剩余时长"）。**统一记作 C4**，`docs/fixture-pools.md` 落盘时必须写明用的是 a 还是 b，**不得**声称写过负数字段。

---

## 5. 数值口径（**全项目唯一真相源**，代码与其它文档只引用不复述）

### 5.1 时长容差

- **`tick_tolerance = 1`**：任一两口径的时长差值 **≤ 1 tick（对齐到同一时基后）** 视为一致。
- **对齐方式**：统一换算到一个**公共时基**上比较（整数运算，向下取整；**不引入浮点**）。
  公共时基取**影片时基 `mvhd.timescale`**——`mvhd`/`tkhd`/`elst` 三个口径**原生**就写在那里，
  媒体侧（`mdhd.duration`、`Σstts`）向下取整换算过来，量化误差严格小于 1 个影片 tick，
  恰好落在容差内。**不采用反向换算**（影片→`mdhd`）：那会把粗时基的量化误差放大
  `mdhd.timescale/mvhd.timescale` 倍，使逐样本取整的合法文件被判成超容差（假阳性）。
- **判定的媒体口径用 `mdhd.duration`（声明值），不用 `Σstts`**：`Σstts` 由逐样本
  `floor(duration/frames)` 拼出，与声明值天然差几个 tick。`Σstts` 仍作为证据原值报出
  （`values.stts_sum`），但不参与阈值比较。
- 超出容差即 `fail`，并且**必须同时报出三口径原值与差值**。
- 说明：1 tick 是**工程口径**（吸收规范允许的舍入），**不是规范条款**。

### 5.2 `timescale` 白名单与"极小值"

- **常见值（白名单，仅用于 `gray`/提示，不用于判 fail）**：视频侧 `90000` / `600` / `1000` / `10000` / `30000` / `24000` / `25000` / `48000` / `50000` / `60000`；音频侧等于采样率（`44100` / `48000` 等）。
- **极小值阈值 `timescale_suspect_below = 100`**：低于此值 → `timescale_suspect`，措辞为"溢出前兆/精度不足"，**默认按 `gray` 处理**（因为低 timescale 也可能是有意的低成本时间基）。
- **溢出前兆判据（硬）**：若 `timescale × 期望秒数` 超过该字段位宽上限（`stts` 为 32 位；`tfdt` 按 version 0/1 分别取 32/64 位），则 `dts_overflow_risk` = `fail`，并报出"在什么位置会越过上限"。
- 说明：白名单与阈值都是**工程口径**，**非规范条款**；报告中不得写成"规范要求 timescale ≥ 100"。

### 5.3 `matrix` 正交性容差

- matrix 以 **16.16 定点整数**读取，**全程整数运算**。
- **正交性容差 `matrix_ortho_tolerance = 1`（单位 1/65536）**：行/列点积与 1.0（= 65536·65536）的偏差在容差内视为正交。
- `rotation_dimension_conflict` 的另一半判据（呈现宽高）**不依赖容差**：只看"是否 ±90°"与"`width` 与 `height` 的大小关系 + 是否有 `pasp`"。

### 5.4 回归判定阈值（DTS / PTS / 分片时序）

- **`dts_regression_tolerance = 0`**：DTS **严格单调不减**，**任何一步下降（哪怕 1 tick）即 `fail`**。理由：时间戳回退没有"合理容差"可言，帧序被打乱的最小单位就是 1 tick。
- **"回退"的定义**：`dts[n] < dts[n-1]`（相邻两步），以及**分片间** `tfdt[k] < tfdt[k-1]`（同一轨道）。
- **`stts.duration == 0` 不算回退但单独报**：`zero_duration_entry`（会让多个样本落在同一时刻，属"精度/语义疑点"），首期按 **`fail`** 报出并定位到 `stts` 条目号——**不新增 gray 原因**（§5.5 的 gray 集合是封闭的）。
- **PTS 不参与单调判定**：B 帧使 PTS 天然乱序；PTS 只用于 M2 的"首帧 PTS 落在 elst media 区间内"与 M4 的偏移计算。
- **`ctts` version 1 的有符号偏移**必须按有符号解析（负偏移合法），**不得**当无符号读——否则会把正常 B 帧文件判成回退（这就是 M3 回归集必须含 B 帧的原因）。

### 5.5 `gray` 判定条件（枚举，**封闭集合**）

`gray` **只能**由以下原因产生，新增原因必须改本节 + `docs/report-schema.md`：

| `reason` | 触发条件 |
|---|---|
| `vfr_detected` | `stts` 条目数 > 1（VFR 本身不是缺陷，但使"恒帧率推断"不可用） |
| `no_roll_distance` | 音频轨无 `sgpd`/`sbgp` 的 roll 距离，priming 期望值不可推 |
| `unknown_codec_priming` | 编码不在已知集（AAC / Opus）内，priming 期望值不可判 |
| `media_rate_not_one` | 相关 `elst` 段 `media_rate ≠ 1`，采样数不可线性换算 |
| `no_edts` | `edts` 缺失但首帧起点非 0（多轨起点亦不一致），无法判断"是裁剪还是缺陷" |
| `hdr_metadata_missing` | `colr` 为广色域（如 bt2020 类）但无 `mdcv`/`clli` |
| `timescale_suspect` | `timescale < 100` 且未触发 `dts_overflow_risk` |
| `duration_unknown` | `mvhd.duration == 0`（分片文件常见合法形态）且 `mehd` / 末段 `tfdt`+`trun` 均推不出预期时长，故三口径互比缺一方 |

**禁止**：把解析失败、盒越界、字段缺失降级成 `gray` —— 那属 `error` 或 `fail`（`plan/00` D11）。

### 5.6 退出码（判定通道 0/1/2；用法/解析通道 3 不计入判定）

| 码 | 语义 | 条件 |
|---|---|---|
| `0` | **PASS** | `fail == 0` 且 `gray == 0` |
| `1` | **FAIL** | `fail ≥ 1`（无论 `gray` 多少） |
| `2` | **INCONCLUSIVE** | `fail == 0` 且 `gray ≥ 1`（判不准必须显式表达，**不得默默给 0**） |
| `3` | **用法错误 / 无法解析** | 参数非法、文件不可读、盒子树无法建立（`parse_error`）。**不进判定通道**，报告不出 `VERDICT` |

- 首屏 `GATE` 行只在合成/真实样本批跑（`--suite`）时出现；单文件审计不打印门禁三数（避免"单文件全绿=通过了 25 例门禁"的误读）。

### 5.7 证据四元组（每条 `fail` 必带）

`byte_offset`（绝对偏移，十六进制，`0x` 前缀）· `field_path`（如 `moov/trak[0]/edts/elst[1].media_time`）· `values`（各口径原值，整数 + 单位）· `delta`（差值，带单位）。四者缺一 → 该条 finding 视为**不合格**，报告渲染层直接拒绝输出（测试红）。

---

## 6. 数值口径的复核方式

- **唯一性检查**：对 `tick_tolerance` / `timescale_suspect_below` / `matrix_ortho_tolerance` 三个常量做仓库级 `grep`，**只应命中** 本文件与实现该口径的**单一常量模块**（`internal/policy/`）。命中第三处即视为违约。
- **不漂移检查**：`plan/01` 与 `plan/00` 中**不得**出现星数以外的可漂移数字（测试条数、行数、覆盖率）；星数只允许出现在 `plan/01 §2.1` 与本文件 §1.4–1.5，且必须标"快照"。
- **重跑记录**：任何再次检索，都必须在本文 §1.4 表下追加一行"复跑窗口 + 变化项"，**不覆盖**历史快照。
