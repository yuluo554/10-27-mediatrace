# 规则表（人类可读 + 反例）

> 状态：**M3 已收尾**——已实现**模块 M1（结构自洽）**与**模块 M2（时长口径）**；
> 模块 M3–M5（时间戳/偏移/显示）的规则 ID 已定名，实现待 M4–M5。规则 ID 与
> `internal/rules/` 实现一一对应。
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

| RuleID | 判什么 | 反例（不算违规） |
|---|---|---|
| `dts_regression` | `dts[n] < dts[n-1]`（0 容差） | **PTS 乱序**（B 帧）——只判 DTS |
| `tfdt_regression` | 跨分片 `tfdt` 回退 | 分片间正常递增 |
| `zero_duration_entry` | `stts.duration == 0` | — |
| `dts_overflow_risk` | 累计越过字段位宽 | 未越界 |
| `timescale_suspect` | 极小 timescale（gray） | 白名单值；低 timescale 但未溢出仍 gray |
| `vfr_detected` | `stts` 条目 >1（gray） | 单条目恒帧率 |

**反例说明**：`ctts` version 1 的负偏移是合法的（B 帧），**不得**当回退。
这条必须有专门单测，否则正常文件会被大面积误伤。

---

## M4 · 偏移 / priming

| RuleID | 判什么 | 反例（不算违规） |
|---|---|---|
| `start_trim` | `elst[0].media_time > 0` | 起点 0 |
| `priming_undeclared` | 首帧后移且无裁剪 edit | 已用 `elst` 正确裁剪 |
| `priming_mismatch` | roll 距离 ≠ 期望 | 编码未知 → gray |
| `no_edts_but_nonzero_start` | 无 `edts` 且起点非 0（gray） | 起点为 0 |
| `no_roll_distance` | 无 roll 距离（gray） | — |
| `unknown_codec_priming` | 编码不在已知集（gray） | AAC/Opus |

**注意**：AAC 1024 / Opus 312 是**常见观测值，不是规范强制**。只判"与声明是否一致"，
不判"必须等于 1024"。

---

## M5 · 显示与色彩

| RuleID | 判什么 | 反例（不算违规） |
|---|---|---|
| `matrix_not_orthogonal` | 行列式/点积超容差（整数） | 正常 0°/90°/180° 矩阵 |
| `rotation_dimension_conflict` | ±90° 且 `width<height` 且无 `pasp` | 正常竖屏（有合理 `pasp`） |
| `sar_dar_conflict` | `pasp` 与 DAR 不一致 | 无 `pasp`（默认方形像素） |
| `hdr_metadata_missing` | 广色域无 `mdcv/clli`（gray） | 有完整 HDR 元数据 |
| `hdr_range_suspect` | `max<=min` 或 `clli`/`mdcv` 矛盾 | — |
| `colr_multi_instance` | 多 `colr` | 单 `colr` |

**不做**：绝对亮度/绝对色彩是否正确（只读元数据无法确证）。

---

## 非目标（首期红线）

1. 不做跨工具一致性预言（mpv/Chrome/QuickTime 行为一致）。
2. 不做媒体标签审计（`©nam`/`©too`）。
3. 不做损坏文件修复（不写回）。
4. 不做纯 MKV 校验器（MKV 仅 M1 结构对应项）。
5. 不做 A/V 同步测量（需解码）。
6. 不做绝对亮度/色彩正确性。
7. 不做跨平台差异归因。
8. 不做自动修复 / 重写容器。
9. 不裁决"哪个工具对"。
