# mediatrace

**媒体时间轴语义审计器** —— 只读字节、不解码、零网络、单二进制 CLI。

> ffprobe、MediaInfo、MP4Box、Bento4 各给一份"我看到的样子"，彼此不对话；
> mediabunny 明知 `elst` 有它不支持的指令却只 `console.warn`；唯一做过策略判定的
> MediaConch 已于 2021 停更，且只查格式合规不查同文件内部字段互相矛盾。
> **所有人都在"打印"，没有人在"判定"。**

## 它回答什么问题

> **同一个文件内部，多条时间轴口径是否互相自洽；不自洽时，差在哪个盒子、哪一字节、差多少。**

一个 MP4 里的时间轴至少有六种"说法"：`stts` 累加、`mvhd`/`tkhd.duration`、`Σ elst`、
`elst.media_time`、`ctts`→PTS 与 `stts`→DTS、分片 `tfdt`。
规范允许它们在编辑/裁剪/分片/转码后各自漂移——工具把每条都打印出来，但不告诉你它们互相矛盾。

## 状态

| 里程碑 | 内容 | 状态 |
| --- | --- | --- |
| **M1** | 数据先行（C1–C7 夹具 + 真值） | ✅ 已完成 |
| **M2** | 结构判定（模块 M1） | ✅ 已完成 |
| **M3** | 时长口径判定（模块 M2） | ✅ 已完成 |
| **M4** | 时间戳判定（模块 M3） | ✅ 已完成 |
| **M5** | 偏移与显示判定（模块 M4+M5）+ 报告 | ⬜ 未开工 |
| **M6** | 发布（tag + Release + 留档） | ⬜ 未开工 |

> **当前进度**：**M1 数据先行 + M2 结构判定 + M3 时长口径判定 + M4 时间戳判定已收尾**。模块
> **M1（结构自洽）**、**M2（时长口径）** 与 **M3（时间戳）** 已实现：
> - `mediatrace audit <file> --modules M1` → `duplicate_moov` / `stco_out_of_mdat` /
>   `truncated_mdat` / `child_overruns_parent` / `duplicate_mdat` / `max_depth_exceeded` 等八条规则；
> - `mediatrace audit <file> --modules M2` → `duration_mismatch`（三口径互比，1 tick 容差）/
>   `elst_item_rate_not_one`（报条目号）/ `elst_item_order_suspect` / `empty_edit` /
>   `first_pts_outside_media` / `duration_unknown`（gray）；
> - `mediatrace audit <file> --modules M3` → `tfdt_regression`（跨分片回退）/ `zero_duration_entry` /
>   `dts_overflow_risk`（位宽越界）/ `dts_regression`（DTS 0 容差单调）/ `timescale_suspect`（gray）/
>   `vfr_detected`（gray）——**只看 DTS，B 帧的 PTS 乱序不误伤**。
>
> 每条 `fail` 带四元组证据；对 92 例合成夹具**召回精确、对照组 60 例误报 0**（M1、M2、M3 各自核对）。
> 模块 M4–M5（偏移/显示）尚未实现（`--modules` 对它们明确返回退出码 3，**不冒充 PASS**）。
> 计划与口径见 `plan/`，续接入口是 `plan/00-README总览.md`。

## 快速开始

```bash
# 任务入口用 node（本机可能没有 make；Makefile 只是它的薄委托）
node tools/tasks.mjs gates          # 两个门：布局 + 脱敏
node tools/tasks.mjs test           # go vet + go test（含"整夹具集跑 M1"的对账测试）
node tools/tasks.mjs bin            # 构建 bin/mediatrace
node tools/tasks.mjs fixtures       # 生成夹具（92 例，C1–C7 + 三组对照）
node tools/tasks.mjs check-fixtures # 双跑逐字节比对
node tools/tasks.mjs all            # 以上一键跑

bin/mediatrace --version
# 结构判定（模块 M1 已实现）：
bin/mediatrace audit fixtures/out/c2_stco_out_of_mdat.mp4 --modules M1
# 时间戳判定（模块 M3 已实现）：跨分片 tfdt 回退 / stts 零时长 / 位宽越界 / 极小时基
bin/mediatrace audit fixtures/out/c5b_tfdt_regression.mp4 --json --modules M3
```

> Go 不在 `PATH` 时：`GO=/path/to/go node tools/tasks.mjs test`。
> Windows 产物名是 `bin/mediatrace.exe`。

## 退出码

| 码 | 语义 |
| --- | --- |
| 0 | PASS（`fail == 0` 且 `gray == 0`） |
| 1 | FAIL（`fail >= 1`） |
| 2 | INCONCLUSIVE（判不准——`fail == 0` 且 `gray >= 1`，**不默默给 0**） |
| 3 | 用法错误 / 无法解析（不进判定通道） |

定义处：`data/README.md §5.6`。

## 三态

`pass` / `fail` / `gray`。`gray` 表示**证据不足以判定**，带 `reason` 枚举、可导出、
**永不计入 fail，也永不折算成 pass**。解析失败不是 `gray`，是退出码 3。

每条 `fail` **必须**带四项证据：`byte_offset` + `field_path` + `values` + `delta`。

## 隐私与边界

- **纯本地、零网络**（无遥测、无版本检查、无在线样本拉取）；判定路径不 import `net`。
- **只读**：不写回输入文件、不做修复、不重写容器。
- **不解码**：不读 `mdat` 载荷语义；运行时**不依赖 ffmpeg**，不链接 `libav*`。
- 报告默认**不写绝对路径**（只写文件名与字节偏移）。
- **不做**跨工具一致性预言（"同一文件在 mpv/Chrome/QuickTime 表现是否一致"）——
  它需解码、跨平台不可复现、结论非确定性。详见 `plan/01 §十`。

## 文档

| 文档 | 内容 |
| --- | --- |
| `plan/00-README总览.md` | 索引 / 决策 / 承诺台账 / 里程碑（**续接入口**） |
| `plan/01-题目文档.md` | 需求证据、模块定义、验收门槛 |
| `plan/02`–`plan/06` | 架构 / 模块详设 / 数据计划 / 里程碑 / 交付对标 |
| `data/README.md` | 依据查证 + **数值口径唯一真相源** |
| `docs/rules.md` | 每条规则的人类可读说明与反例 |
| `docs/report-schema.md` | 三态 JSON 结构 + 首屏三段 + 退出码 |
| `docs/fixture-pools.md` | C1–C7 构造纪律 |
| `AGENTS.md` | 方法论锚点与门命令（新对话第一件事） |

## 许可

MIT，见 `LICENSE`。
