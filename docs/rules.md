# 规则表（人类可读 + 反例）

> 状态：**骨架期**（M2 起逐模块补齐）。规则 ID 与 `internal/rules` 实现一一对应。
> 口径数字只在 `data/README.md §5`；本文只写"规则在判什么、什么不算违规"。

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

| RuleID | 判什么 | 反例（不算违规） |
|---|---|---|
| `too_small_size` | `size` < 8 且非 0/1 形态 | `size==0`（到文件尾）、`size==1`（largesize） |
| `child_overruns_parent` | 子盒区间 ⊄ 父盒载荷 | 子盒恰好等于父载荷 |
| `duplicate_moov` | `moov` ≥2 次 | 单个 `moov` + 多个 `moof`（分片合法） |
| `duplicate_mdat` | `mdat` ≥2 次 | — |
| `stco_out_of_mdat` | 偏移 + 长度 ⊄ 任一 `mdat` 载荷 | 指向另一 `mdat`（多 mdat 合法） |
| `truncated_mdat` | 声明 size > 可读字节 | `size==0` 的"到文件尾"形态 |
| `ftyp_brand_mismatch` | brand 与实际编码矛盾 | `iso2` + `mp41` 这类兼容组合 |
| `max_depth_exceeded` | 深度超限（防御） | 正常 4–6 层 |

**反例说明**：`stco` 指向文件尾之外才算越界；指向 `moov` 区间**也**算越界（不是 mdat）。

---

## M2 · 时长口径

| RuleID | 判什么 | 反例（不算违规） |
|---|---|---|
| `duration_mismatch` | 三口径互比超容差 | 差值 ≤ 容差（舍入）；`mvhd.duration==0` 的分片文件（落 gray） |
| `elst_item_rate_not_one` | `media_rate ≠ 1` | `media_rate == 1` 的普通段 |
| `elst_item_order_suspect` | 段次序与首帧 PTS 对不上 | 段次序正确但时长有别（那是 `duration_mismatch`） |
| `empty_edit` | `media_time == -1` 空 edit | 空 edit 仅贡献延迟——**这是信息，不是缺陷**；只有与显示时长矛盾才 fail |
| `first_pts_outside_media` | 首帧 PTS 不落在 media 区间 | `media_time == -1`（无 media 区间） |

**关键区分**（`plan/01 §八` 创新点）：**"段次序错"与"时长错"必须分开报**。
排序 `elst` 会掩盖前者，所以**编号不排序**（`plan/00` D10）。

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
