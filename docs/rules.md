# 规则表（人类可读 + 反例）

> 状态：**M5 已收尾**——五个模块（M1 结构自洽 / M2 时长口径 / M3 时间戳 / M4 偏移与 priming /
> M5 显示与色彩）**全部实现**并对 92 例夹具对账；对照组误报 0。
> 规则 ID 与 `internal/rules/` 实现一一对应。
> 口径数字只在 `data/README.md §5`；本文只写"规则在判什么、什么不算违规"。
> 本节的反例样本 id 一律指向 `fixtures/out/manifest.json` 里的 `case_id`。

---

## 判定的唯一对象

> **同一个文件内部，多条时间轴口径是否互相自洽；不自洽时，差在哪个盒子、哪一字节、差多少。**

不是格式合规，不是两文件比对，不是播放器行为预测。

---

## 三态语义

| 状态 | 含义 | 出口 |
|---|---|---|
| `pass` | 该规则下无异常 | 计入 `pass` |
| `fail` | 有确定性违规，带四元组证据 | 计入 `fail`，退出码 ≥1 |
| `gray` | **证据不足以判定**（不是"大概没错"） | 带 `reason`，可导出，**不计入 fail** |

`gray` **永不折算成 pass，也不升级成 fail**。解析失败不是 gray，是 error（退出码 3）。

---

## M1 · 结构自洽

实现：`internal/rules/structure/`（模块号 `M1`）+ `internal/isobmff/`（迭代遍历 IR）。
遍历**迭代不递归**，深盒树（≥5000 层）在 `MaxDepth` 处停止展开并记一条异常——不爆栈、不 panic。

| RuleID | 判什么 | 证据（四元组要点） | 反例（不算违规） | 反例样本 id |
|---|---|---|---|---|
| `too_small_size` | 盒 `size` < 8（最小盒头） | 盒偏移 + 声明 size；`delta` = 8 − size 的短缺字节 | `size==0`（到文件尾）、`size==1`（largesize）两形态合法 | `K3-mdat`、`K3b-largesize` |
| `child_overruns_parent` | 子盒区间 ⊄ 父盒载荷区间 | 子盒偏移 + 子盒结束 + 父盒载荷结束；`delta` = 越界字节 | 子盒恰好等于父载荷 | `S1`（反例 `K1-moov`） |
| `duplicate_moov` | `moov` 出现 ≥2 次 | **列出全部**偏移；首证据指向第 2 处 | 单个 `moov` + 多个 `moof`（分片合法） | `C1`、`C1b`（反例 `K1-moov`、`K9-frag`） |
| `duplicate_mdat` | 非分片文件 `mdat` ≥2 次 | 列出全部偏移；首证据指向第 2 处 | **含 `moof` 的分片文件每片自带 `mdat`，合法** | `S2`（反例 `K9-frag`） |
| `stco_out_of_mdat` | `stco`/`co64` 偏移 + `stsz` 长度 ⊄ 任一 `mdat` 载荷 | `stco` 条目号 + 目标区间 vs `mdat` 区间 + 文件大小 | 指向**任一** `mdat` 载荷内（多 `mdat` 合法）；被尾部截断亦算越界 | `C2`、`C2b`、`C2c`、`C2d`（反例 `K2-stco`、`K14-co64`） |
| `truncated_mdat` | 顶层 `mdat` 声明 size > 实际可读字节 | `mdat` 偏移 + 声明 vs 可读 | `size==0` 的"到文件尾"形态 | `C3`、`C3b`（反例 `K3-mdat`） |
| `ftyp_brand_mismatch` | 以编码命名的 major brand 与 `stsd` 样本条目矛盾 | `ftyp` 偏移 + brand(fourcc) vs 实际编码(fourcc) | 通用 brand（`isom`/`iso2`/`mp41`/`qt`…）不与任何编码冲突 | 无夹具（单测覆盖）；见 `internal/rules/structure/structure_test.go` |
| `max_depth_exceeded` | 嵌套深度到达上限，更深子盒未展开（防御） | 盒偏移 + 深度上限 vs 实际深度 | 正常 4–6 层；单测 6000 层不 panic | `S3`（反例 `K1-moov`） |

**纪律与合并**（`plan/03 §1.3`）：

- **同一根因合并**：越界子盒只报 `child_overruns_parent` **一条**，其子树不再逐条报；
  重复 `moov` 时偏移表已不可信，**跳过** `stco_out_of_mdat` 区间校验（只报 `duplicate_moov`）。
- **`stco` 越界的两种形态**：指向文件尾之外、或被 `mdat` 尾部截断；指向 `moov` 区间**同样**算越界。
- **`ftyp_brand_mismatch` 取窄口径**：只在 major brand **就是编码名**（`avc1`/`hvc1`/`hev1`/`av01`/`vp09`/`mp4v`）
  且与 `stsd` 条目矛盾时报——避免把"`isom` 容器装 `hvc1` 轨"这类**合法**组合误伤（对照组误报 0 是门禁）。
- **对照组误报 0**：`controls` / `m3-regression` / `m4` / `m5` 四组共 60 例在模块 M1 下**一条 fail 都不许有**，
  由 `cmd/mediatrace` 的 `TestM1MatchesFixtureTruth` 对整个夹具集**集合相等**断言（既抓漏报也抓多报）。

---

## M2 · 时长口径

实现：`internal/rules/duration/`（模块号 `M2`；`rules.go` 里 `m2` 转调它）。
判定对象是**同一个文件里多条时长口径是否互相自洽**，不是"这个时长对不对"。

**参与互比的口径**（全部换算到公共时基，见下）：

| 口径 | 来自 | 原生时基 |
|---|---|---|
| `stts_sum` | `Σ(stts.count × stts.duration)` | 轨道 `mdhd.timescale` |
| `mdhd_duration` | `mdhd.duration`（**判定基准**） | 轨道 `mdhd.timescale` |
| `tkhd_duration` | `tkhd.duration` | 影片 `mvhd.timescale` |
| `mvhd_duration` | `mvhd.duration` | 影片 `mvhd.timescale` |
| `elst_sum` | `Σ elst.segment_duration`（有 `edts` 时） | 影片 `mvhd.timescale` |

**公共时基 = 影片时基 `mvhd.timescale`**（`data/README §5.1`）：`mvhd`/`tkhd`/`elst` 三个口径
原生就在那里；媒体侧向下取整换算过来，量化误差严格小于 1 个影片 tick，落在容差内。
反向（影片→`mdhd`）换算会把粗时基的量化误差放大 `mdhd.timescale/mvhd.timescale` 倍，
把逐样本取整的合法文件判成超容差（假阳性），故不采用。

**判定基准用 `mdhd.duration`（声明值），不用 `Σstts`**：`Σstts` 由逐样本
`floor(duration/frames)` 拼出，与声明值天然差几个 tick（如 `M3R03`：`Σstts`=990 vs
`mdhd.duration`=1000 @600）。`Σstts` 仍作为证据原值报出（`values.stts_sum`），但不参与阈值比较。

| RuleID | 判什么 | 证据（四元组要点） | 反例（不算违规） | 反例样本 id |
|---|---|---|---|---|
| `duration_mismatch` | 三口径互比超容差（容差 = `policy.TickTolerance`，1 影片 tick） | 偏移 + 字段路径指向**越界的那一侧**（`mvhd.duration` / `tkhd.duration` / `elst[i].segment_duration`）；`values` 含 `stts_sum`/`mdhd_duration`/`tkhd_duration`/`mvhd_duration`（+ 有 `edts` 时的 `elst_sum`）；`delta` = 最大值−最小值 | 差值 ≤ 1 tick；`mvhd.duration==0` 的分片文件（落 gray）；`Σstts` 与 `mdhd.duration` 的逐样本取整差 | `C4a`(mvhd 2×)、`C4a2`(tkhd 2×)、`C4a3`(mvhd 偏短)、`C4b`、`E2`（对应反例 `K4-duration`、`K10-empty-edit`、`M3R23`） |
| `elst_item_rate_not_one` | `elst[i].media_rate ≠ 1.0`（16.16） | 偏移指向该条目的 `media_rate` 字段；`values` 含**条目号**（按文件内顺序，**不排序**）+ 原值与期望；**同段不按等速换算** | `media_rate == 1` 的普通段 | `E1`（条目号 **1**） |
| `elst_item_order_suspect` | 首帧 PTS 落在**后面**的一段，而非首个非空段 | 偏移指向首个非空段的 `media_time`；`values` 含首帧 PTS + 落在哪条 + 应在哪条 | 段次序正确（时长有别 → 那是 `duration_mismatch`）；首帧 PTS 不在任何段（→ 交给 `first_pts_outside_media`） | `E3`（`elst[0].media_time`=45000、`elst[1].media_time`=0） |
| `empty_edit` | 空 edit（`media_time == -1`）**出现在媒体段之后**或其 `segment_duration == 0` ——自相矛盾 | 偏移指向该条目的 `media_time`；`values` 含 `segment_duration` + 由此产生的**起始偏移** | 空 edit 位于最前且时长非 0：**它只贡献放映延迟、不引用媒体数据，这是信息不是缺陷**；此时若显示时长没计入，由 `duration_mismatch` 报（同一根因不连环报） | 无夹具（`E2`/`K10-empty-edit` 都是合法形态，见下行） |
| `first_pts_outside_media` | 首帧 PTS 不落在**任何**段的 media 区间 | 偏移指向首个非空段的 `media_time`；`values` 含 PTS 与区间两端；`delta` = 越出字节数 | `media_time == -1`（无 media 区间）；**跨度一致的编辑表**（`Σ非空段` = `mdhd.duration`）：那是合法起点裁剪，属 M4 的 `start_trim` | `C4b`（反例 `K6-priming`、`M4N1`、`M4P3`、`M4P4`） |
| `duration_unknown` (gray) | `mvhd.duration == 0` 且 `mehd`/末段 `tfdt`+`trun` 都推不出预期时长 | `gray.reason = duration_unknown`（**闭集内，不新增 reason**）；不猜 | 推得出（`trun` 带 `sample_duration` / `trex` 默认值 / `mehd` 有值）→ 不落 gray | `F1`（反例 `K9-frag`、`C5b`、`C5c`） |

**关键区分**（`plan/01 §八` 创新点）：**"段次序错"与"时长错"必须分开报**——`E3` 只报
`elst_item_order_suspect`（三口径本身对齐），`C4b`/`E2` 只报 `duration_mismatch`（段次序正常）。
排序 `elst` 会掩盖前者，故**编号不排序**（`plan/00` D10）。

**同一根因合并**（`plan/03 §1.3`）：一个轨道最多产出一条 `duration_mismatch`（优先级
影片级 → `tkhd` → 编辑表跨度），不连环报；空 edit 与显示时长的矛盾就由那一条表达。

**分片与空样本表**：轨道 `stts` 为空（分片轨）时不判其时长——时长由 `moof` 决定，属二期；
此时 `mvhd.duration == 0` 且推不出才落 `duration_unknown`。

**字段路径的索引口径**（`F14`）：直接用 IR 的 `Box.Path`，**同名兄弟才带 `[i]`**——
单轨/单 `elst` 文件的路径是 `moov/trak/edts/elst[0].media_time`（不带 `trak[0]`）；
`elst[i]` 的条目号按文件内顺序、从 0 起，**不排序**。

**对照组误报 0**：`controls` / `m3-regression` / `m4` / `m5` 四组共 **60 例**在模块 M2 下
**一条 fail/gray 都不许有**，由 `cmd/mediatrace` 的 `TestM2MatchesFixtureTruth` 对整个夹具集
对账（M2 相关缺陷 8 例**集合相等** + 对照组零 finding）；`--modules M2` 的结论与
`--modules M1,M2` 中的 M2 那一半必须一致（`TestM2SubsetConclusionIsStable`，G9）。

---

## M3 · 时间戳

实现：`internal/rules/timestamps/`（模块号 `M3`；`rules.go` 里 `m3` 转调它）。
判定对象是**同一个文件里时间戳序列是否自洽**：`stts` 推出的 DTS 是否单调、累计是否
越过字段位宽、`timescale` 是否小到出现精度/溢出前兆、`stts` 条目是否多到使恒帧率推断失效。

**判定只用 DTS，不判 PTS**（`data/README §5.4`）：B 帧使 PTS 天然乱序——`ctts` 只用于
"不把 PTS 乱序当回退"，本模块**根本不参与 PTS 判定**。回归判据只有一条：
`dts[n] < dts[n-1] - policy.DTSRegressionTolerance`（容差 **0**，见 §5.4）。

| RuleID | 判什么 | 证据（四元组要点） | 反例（不算违规） | 反例样本 id |
|---|---|---|---|---|
| `dts_regression` | 样本级 DTS **严格单调不减**被破坏（§5.4，0 容差） | 偏移 = 该 `stts` 条目的 `duration` 字段；`field_path` = `…/stts[n].duration`；`values` 含**样本序号** + **`stts` 条目号** + 前后 DTS；`delta` = 回退量 | **B 帧的 PTS 乱序**（只判 DTS）；`stts.duration` 无符号 ⇒ 良构 `stts` 的累计 DTS **不可能回退** | 无夹具（见下行合纪） |
| `tfdt_regression` | 跨分片（同轨）`tfdt[k] < tfdt[k-1]`（0 容差） | 偏移 = **回退那一片** `tfdt` 值字段；`values` 含**分片序号** + 两条 `tfdt` 值 + **两处偏移**；`delta` = 回退量；`tfdt` **v0 32 位 / v1 64 位**分宽读 | 分片间 `tfdt` 正确递增 | `C5b`(v0)、`C5c`(v1)（反例 `K9-frag`） |
| `zero_duration_entry` | `stts` 某条 `duration == 0`（多条样本落在同一时刻） | 偏移 = 该条的 `duration` 字段；`values` 含**条目号** + 样本数 + 原值 + 同轨标称时长；`delta` = 亏空 | ——（首期按 **fail** 报出，**不新增 gray**，§5.4） | `C5`（条目号 1） |
| `dts_overflow_risk` | 累计 DTS 越过字段位宽（`stts` **32 位**；`tfdt` v0 32 / v1 64 分宽） | 偏移 = 越界处所在条目的 `duration` 字段；`values` 含**条目号 + 样本序号** + 越界累计值 + 字段位宽上限；`delta` = 超出量；用**整数运算，不浮点** | 未越界；`tfdt` v1（64 位头寸）不设越界 | `T2`（`stts` 累计 ≈ 2³²） |
| `timescale_suspect` (gray) | `mdhd.timescale < policy.TimescaleSuspectBelow`（100）**且未触发** `dts_overflow_risk` | 偏移 = `mdhd.timescale` 字段；`gray.reason = timescale_suspect`（**闭集内，不新增 reason**） | 低 timescale 但已溢出 → 报 `dts_overflow_risk`（fail），不落本条；白名单值（90000 等）不报 | `T1`（`timescale=30`） |
| `vfr_detected` (gray) | 同轨 `stts` 条目数 **> 1**（VFR 本身不是缺陷，只使恒帧率推断不可用） | `gray.reason = vfr_detected`（**闭集内**）；`values` 含条目数 | 单条目（恒帧率）；空 `stts`（分片轨，0 条不算多） | `C5`（同时落 `zero_duration_entry`+`vfr_detected`，**并列不互斥**；反例 `K5-dts`、`m3r*`） |

**同一根因合并**（`plan/03 §1.3` / `HANDOFF-M4 §二 F15`）：`stts` 累计越过位宽时，字段宽度内的
"回退"只是**溢出的表象**——同一根因只报 **`dts_overflow_risk` 一条**，不再单报 `dts_regression`。
一处 `stts` 根因只报一条 `zero_duration_entry`/`dts_regression`。

**为什么 `dts_regression` 在本夹具集里没有对应样本**（如实登记，非漏做）：
`stts.duration` 是**无符号**，累计 DTS 单调不减，字面判据 `dts[n] < dts[n-1]` 对良构 `stts`
**不可能为真**（`tools/fixture-gen/gen/builders_defects.go` 头注同样写明）。故 `data/README §4`
的 C5「回退 DTS」用两条**可达**形态表达：`zero_duration_entry`（`C5`）与 `tfdt_regression`
（`C5b`/`C5c`）。`dts_regression` 仍是模块 M3 的**完整判据**，"报样本序号 + 条目号 + 字节偏移"
的证据契约由 `TestDTSRegressionCarriesSampleIndexEntryAndOffset` 钉住。

**分宽读**（`F14`/`F17`）：`stts`/`ctts`/`tfdt` 都不是容器盒，IR 里只有盒本身，逐条字段用
只读字节 + `PayloadStart()` 自己读，**全部带边界检查**（截断文件不 panic）。
`ctts` v1 的偏移是**有符号**（负偏移合法，B 帧）——本模块不判 PTS，但分宽读的纪律仍由单测守，
且 `m3-regression` 的 v1 负偏移样本必须 **0 finding**（`TestM3DoesNotMisjudgeBFrames`）。

**字段路径的索引口径**（`F13`）：直接用 IR 的 `Box.Path`——单轨文件的路径是
`moov/trak/mdia/minf/stbl/stts`（**不带** `trak[0]`）；条目号写在方括号里
（`stts[1].duration`，从 0 起，**不排序**）；分片路径带同名兄弟索引（`moof[1]/traf/tfdt`）。

**样本级遍历不落全量样本表**（`plan/01 §五 M3` 确定性纪律 / R4）：按 `stts` **条目**推进累计
DTS（每条的 `count×duration` 用整数一次算完），只在越界点定位到条目号 + 样本序号——避免大文件内存爆炸。

**对照组误报 0**：`controls` / `m3-regression` / `m4` / `m5` 四组共 **60 例**在模块 M3 下
**一条 fail/gray 都不许有**，由 `cmd/mediatrace` 的 `TestM3MatchesFixtureTruth` 对整个夹具集
对账（M3 相关缺陷 5 例 `C5`/`C5b`/`C5c`/`T1`/`T2` **集合相等** + 构造点字节偏移命中 +
对照组零 finding）；`--modules M3` 的结论与 `--modules M1,M2,M3` 中的 M3 那一半必须一致
（`TestM3SubsetConclusionIsStable`，G9）。**`m3-regression` 24 例（含 v1 负偏移 B 帧）就是本模块的门禁主体**。

---

## M4 · 偏移 / priming

实现：`internal/rules/offset/`（模块号 `M4`；`rules.go` 里 `m4` 转调它）。
判定对象是**音频轨的起始偏移与 encoder delay**：编辑表声明的起点裁剪、首帧 PTS 的后移量、
容器用 `sgpd`/`sbgp` roll 距离（或 Opus `dOps.preSkip`）声明的 priming —— 三者是否互相自洽。

**只判音频轨**：视频轨的 `ctts` 偏移是 B 帧重排，不是起始偏移；本模块对非音频轨**不做**判定
（否则会误伤所有含 B 帧的正常视频）。

**期望 priming = 声明值**（`K14` / `data/README §2`）：容器声明多少就按多少比。AAC 1024 / 2112、
Opus 312 都是**观测值，不是规范条款**——本模块**不判"必须等于 1024"**，只判"裁剪与声明是否一致"。

| RuleID | 判什么 | 证据（四元组要点） | 反例（不算违规） | 反例/正例样本 id |
|---|---|---|---|---|
| `priming_undeclared` | 首帧 PTS 后移且**无**裁剪 edit、且容器**无** roll 声明（后移量就是未声明 priming 的直接证据） | 偏移 = 首条 `ctts` 的 `sample_offset`；`field_path` = `…/ctts[0].sample_offset`；`values` 含后移样本数 + `mdhd.timescale`；`delta` = 后移量 | 已用 `elst` 正确裁剪；无后移；有 roll 声明（→ `no_edts`） | `C6`/`M4P1`（报 **1024**）、`M4P2`（报 **2112**） |
| `start_trim` | 编辑表裁剪量 **> 期望 priming** —— 越过 priming 剪进正片 | 偏移 = `elst[0].media_time`；`values` 含 media_time(tick) + 裁剪样本数 + **秒值(millis)** + 期望值与来源；`delta` = 裁剪 − 期望 | 裁剪 == 期望（正确裁剪）；裁剪 < 期望（→ `priming_mismatch`） | `M4P4`（裁剪 2048 > roll 1024） |
| `priming_mismatch` | 编辑表裁剪量 **< 期望 priming** —— 容器说多、编辑说少，两处声明对不上 | 偏移 = `elst[0].media_time`；`values` 含裁剪量 + 期望值与来源；`delta` = 期望 − 裁剪 | 裁剪 == 期望 | `C6c`/`M4P5`（roll 1152 > 裁剪 1024）、`M4P3`（roll 1024 > 裁剪 512） |
| `no_roll_distance` (gray) | 有声轨裁剪且有 edit，但**无** `sgpd`/`sbgp` roll 距离 —— 期望值不可推 | 偏移 = `elst[0].media_time`；`gray.reason = no_roll_distance`（闭集内） | 有 roll 声明（可判）；无裁剪 | `C6b` |
| `no_edts_but_nonzero_start` (gray) | 首帧后移且**有** roll 声明，但**无** edit —— 无法判断后移是"裁剪"还是"缺陷" | 偏移 = 首条 `ctts` 的 `sample_offset`；`gray.reason = no_edts` | 无 roll 声明（→ `priming_undeclared` fail）；无后移 | **无夹具**（单测覆盖：`internal/rules/offset/offset_test.go` `TestNoEdtsButNonzeroStartIsGray`） |
| `unknown_codec_priming` (gray) | 音频编码不在已知集（`mp4a` / `Opus`）—— 期望值不可判 | 偏移 = `stsd` 样本条目编码 fourcc；`gray.reason = unknown_codec_priming` | 编码在已知集内 | `P1`（`ac-3`） |

**同一根因合并**（`F16`）：`priming_undeclared` 与 `start_trim` 互斥；一处裁剪只报一条。

**判定顺序**（可判性优先）：编码未知 → gray；无裁剪 edit → 有 roll 落 `no_edts`、无 roll 报 `priming_undeclared`；
有裁剪 edit → 无 roll 落 `no_roll_distance`、有 roll 则按"裁剪 vs 期望"分三支。

**为什么裁剪 > 期望是 `start_trim`、裁剪 < 期望是 `priming_mismatch`**：前者把该播的正片剪掉了
（起始裁剪过量），后者是两处声明对不上（少裁）。**M4P4 与 C6c 的字节差别就在这一点**：`M4P4`
裁剪(2048) > roll(1024) 是过量裁剪；`C6c` roll(1152) > 裁剪(1024) 是少裁。二者必须有不同字节，
故 `C6c` 的 `roll` 取 1152（旧真值里的 704 与 `M4P4` 同属"裁剪 > roll"一档，无法区分，已在 M5 修正）。

**对照组误报 0**：M4 的误报门覆盖**所有非 M4 目标样本**（`controls` 16 + `m3-regression` 24 +
`m4` 反例 5 + `m5` 10 + 其他模块专属缺陷，共 83 例）在模块 M4 下 **0 finding**，由
`cmd/mediatrace` 的 `TestM4MatchesFixtureTruth` 集合相等断言；`--modules M4` 的结论与合集/默认全集中的
M4 那一半一致（`TestM4SubsetConclusionIsStable`，G9）。

---

## M5 · 显示与色彩

实现：`internal/rules/display/`（模块号 `M5`；`rules.go` 里 `m5` 转调它）。
判定对象是 `tkhd` 的 3×3 显示矩阵（**16.16 定点整数**）、呈现宽高、`pasp`、`colr`/`mdcv`/`clli`
之间是否互相自洽。**不做**绝对亮度 / 绝对色彩正确性判定（只读元数据无法确证）。

| RuleID | 判什么 | 证据（四元组要点） | 反例（不算违规） | 反例/正例样本 id |
|---|---|---|---|---|
| `matrix_not_orthogonal` | 左上 2×2 的行/列点积与 1.0 的偏差超 `policy.MatrixOrthoTolerance`（**整数运算**，`policy.Square`） | 偏移 = `tkhd.matrix`；`values` 含两行点积、行间点积、期望单位点积、容差；`delta` = 最大偏差（1/65536 单位） | 单位阵 / 0°·90°·180° 正交旋转 | `D1`/`M5P3`（剪切矩阵）；反例 `K12-matrix-ortho`/`M5N3` |
| `rotation_dimension_conflict` | matrix 为 ±90° 且 `tkhd.width < height` 且**无** `pasp` | 偏移 = `tkhd.matrix`；`values` 含旋转角 + 呈现宽高；`delta` = 高 − 宽 | 呈现宽 > 高（正常竖屏）；有 `pasp`；旋转 0°/180° | `C7`/`M5P1`（+90°）、`C7b`/`M5P2`（−90°）；反例 `K7-portrait`/`M5N1`（rot90 + 宽>高 + pasp）、`K8-rot180`/`M5N2` |
| `sar_dar_conflict` | `pasp` 推出的 DAR 与 `tkhd` 声明的宽高比不一致（±90° 时取倒数） | 偏移 = `pasp`；`values` 含 pasp 分子/分母 + 存储宽高 + 呈现宽高；`delta` = 宽度差（px） | 无 `pasp`（默认方形像素）；`pasp` 与声明一致 | `M5P5`（pasp 8:9 与 tkhd 4:3 矛盾）；反例 `M5N5`（pasp 1:1）、`K7-portrait`（rot90 + pasp 1:1） |
| `colr_multi_instance` | 样本条目含 **≥2** 个 `colr` | 偏移 = 首个 `colr`；`values` **按文件内顺序全列**每个 `colr` 的 primaries（D10，不取第一个）；`delta` = 实例数 − 1 | 单个 `colr` | `D2`/`M5P4`（bt709 → bt2020）；反例 `K11-colr-hdr`/`M5N4` |
| `hdr_metadata_missing` (gray) | 广色域 `colr`（bt2020 primaries=9 / Display-P3=12）但**无** `mdcv`/`clli` | 偏移 = 该 `colr`；`gray.reason = hdr_metadata_missing` | 有完整 HDR 元数据；非广色域 | **无夹具**（单测覆盖：`internal/rules/display/display_test.go` `TestHDRMetadataMissingIsGray`） |
| `hdr_range_suspect` | `mdcv` 量程 `max<=min`，或 `clli`/`mdcv` 互相矛盾（`max_content_light_level` ≠ `mdcv.max_luminance`；`avg > max`）——**只判量程与互斥，不判绝对亮度** | 偏移 = `mdcv`/`clli` 字段；`values` 含量程原值；`delta` = 越界量 | 量程合法且两处元数据一致 | **无夹具**（单测覆盖：`TestHDRRangeSuspect`、`TestHDRCompleteIsClean`） |

**matrix 读数口径**（`F5`/`K15`）：`tkhd` 是叶子盒，`matrix` 按 16.16 定点**整数**读（v0 偏移 40、v1 偏移 44）；
正交性 = 2×2 的行/列点积；容差换算到点积原单位（`tol × 65536`），**全程整数，不做浮点近似**。

**`colr`/`pasp`/`mdcv`/`clli` 的层级**（`F13`）：`stsd` 不是容器盒，样本条目与其子盒在 IR 里不存在，
本包按字节自读样本条目固定头（视觉 78B / 音频 28B）后的子盒序列。**多 `colr` 按序全列，不取第一个**（D10）。

**对照组误报 0**：M5 的误报门覆盖**所有非 M5 目标样本**（同上 83 例）在模块 M5 下 **0 finding**，
由 `TestM5MatchesFixtureTruth` 集合相等断言；G9 由 `TestM5SubsetConclusionIsStable` 守。

---

## 非目标（首期红线）

1. 不做跨工具一致性预言（mpv/Chrome/QuickTime 行为一致）。
2. 不做媒体标签审计（`©nam`/`©too`）。
3. 不做损坏文件修复（不写回）。
4. 不做纯 MKV 校验器（MKV 首期不做，含 EBML 结构对应项；MKV 输入走 exit 3，不是 FAIL）。
5. 不做 A/V 同步测量（需解码）。
6. 不做绝对亮度/色彩正确性。
7. 不做跨平台差异归因。
8. 不做自动修复 / 重写容器。
9. 不裁决"哪个工具对"。
