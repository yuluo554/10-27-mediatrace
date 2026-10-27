# 夹具池与构造纪律（C1–C7）

> 状态：**M1 已收尾**——C1–C7 的**构造器**已全部实现（92 例，能把样本字节造出来并进 manifest）；
> 这**不等于**判定已实现：五个判定模块的规则仍是 `ErrNotImplemented`（待 M2–M5）。
> 权威编号：`data/README.md §4`。
> 实现：`tools/fixture-gen/gen/`（`movie.go` 合法构造器 + `builders_defects.go` + `builders_controls.go`）；
> 清单：`fixtures/out/manifest.json`（**入库**——真值台账；样本字节不入库）。

---

## 一、构造纪律（每条都是出口门判据）

| # | 纪律 | 守门 |
|---|---|---|
| D1 | **全程序化构造**：直接写盒子字节，不手工编辑、不依赖下载样本 | code review + `gen_test.go` |
| D2 | **`--seed` 决定一切随机选择**；同 seed 逐字节一致 | `TestBuildIsDeterministic` + `node tools/tasks.mjs check-fixtures` |
| D3 | **生成器不读时钟 / 环境变量 / 不起外部进程** | `TestGeneratorHasNoClockEnvOrExec`（AST/import 扫描） |
| D4 | **不调用 ffmpeg / MP4Box**：真值字节不得依赖工具链版本 | 同上（`os/exec` 禁用） |
| D5 | **真值在构造点确定**：`Expected` 由构造参数给出，不由实现回算 | manifest 结构 |
| D6 | **空套件不得当通过**：`len(cases)==0` → 退出码 3 | `fixture-gen main` |
| D7 | **样本本体不入库**：`fixtures/out/`、`*.mp4` 均在 `.gitignore` | 布局门 `LAYOUT.GITIGNORE` |

---

## 二、强制构造类

| 编号 | 类 | 手法 | 期望 | 状态（构造器 / 判定） |
|---|---|---|---|---|
| **C1** | 双 `moov` | 复制整个 `moov` 插在第二处 | `duplicate_moov`，**报两处偏移** | 🟡 构造器✅ `C1`+`C1b`（相邻 moov） / 判定待 M2 |
| **C2** | `stco` 越界 | chunk offset 指向 `mdat` 载荷之外 | `stco_out_of_mdat` + 条目号 + 区间 | 🟡 构造器✅ `C2`+`C2b`(指向 moov)+`C2c`(尾部截断)+`C2d`(co64 2^40) / 判定待 M2 |
| **C3** | 截断 `mdat` | 声明 size 大于实际剩余字节 | `truncated_mdat` + 声明 vs 实际 | 🟡 构造器✅ `C3`+`C3b`(尾部截断) / 判定待 M2 |
| **C4** | 时长口径互斥 | **C4a**：`mvhd.duration` ≠ `Σ(stts)`（含 2× 形态）；**C4b**：起点 + 时长越界 | `duration_mismatch` + 三口径原值 | 🟡 构造器✅ `C4a`(mvhd 2×)+`C4a2`(tkhd 2×)+`C4a3`(mvhd 偏短)+`C4b`(elst 越界) / 判定待 M3 |
| **C5** | 回退 DTS | `stts` 插入 duration 使累计回退 | `dts_regression` + 样本序号 + 偏移 | 🟡 构造器✅ `C5`(zero_duration_entry+vfr)+`C5b`(tfdt v0 回退)+`C5c`(tfdt v1 回退) / 判定待 M4 |
| **C6** | 1024 samples 延迟 | 音频首帧 PTS 后移 1024，`elst` 未裁剪 | `priming_undeclared`，**报出 1024** | 🟡 构造器✅ `C6`(ctts 1024 无 edts)+`C6b`(gray `no_roll_distance`)+`C6c`(roll 704 不匹配) / 判定待 M5 |
| **C7** | `matrix=±90` 无 `pasp` | `tkhd` matrix 置旋转，`width<height`，删 `pasp` | `rotation_dimension_conflict` | 🟡 构造器✅ `C7`(rot90)+`C7b`(rot270) / 判定待 M5 |

> **C4 不得写负数字段**：`mvhd/tkhd.duration` 是无符号整数。C4 分 C4a/C4b 两态，
> 落盘时必须写明用的是哪一个（`data/README §4` 末段）。
> **本项目的 C4 全部用 a 与"非负越界"b 两种形态实现**：`C4a`/`C4a2`/`C4a3` 改的是
> `mvhd`/`tkhd` 的声明时长（无符号，取 2× 或偏小值）；`C4b` 让 `Σelst`（1500）大于
> 轨道声明时长（**1500 > 1000**），**没有任何字段被写成负数**——越界关系写在样本结构里，
> 不在任何一个无符号字段里。
>
> 其余已构造的缺陷/边界类（同属 `defects` 套件，共 32 例）：`S1` 子盒越界、`S2` 双 `mdat`、
> `S3` 100 层深嵌（> MaxDepth 64）、`D1` 剪切 matrix、`D2` 多 `colr`、`P1` 未知编码
> （ac-3，gray）、`E1` `media_rate≠1`、`E2` 空 edit、`E3` `elst` 次序、`T1` timescale=30（gray）、
> `T2` 位宽溢出前兆、`F1` 分片 `duration_unknown`（gray）。

---

## 三、对照组（证明"不误伤"）

| 组 | 规模（实测构造数） | 期望 | 状态（构造器 / 判定） |
|---|---|---|---|
| M3 回归集 | **24** 个正常 H.264/H.265（**含 B 帧**：v0 正偏移与 v1 有符号负偏移两形态） | 全 `pass`，误报 **0** | 🟡 构造器✅（`m3-regression`，24 例） / 判定待 M4 |
| M4 正反例 | fail **5** / pass **5** | 正确裁剪不报 / 未声明报 | 🟡 构造器✅（`m4`，10 例） / 判定待 M5 |
| M5 正反例 | fail **5** / pass **5** | 正常竖屏不报 / 冲突报 | 🟡 构造器✅（`m5`，10 例） / 判定待 M5 |
| 通用控制组（`controls`） | **16** 个合法样本 | 全 `pass` | 🟡 构造器✅（moov/stco/size0/largesize/duration/dts/priming/portrait/rot180/frag/empty-edit/colr+hmd/matrix-ortho/opus/co64/B 帧） / 判定待 M2–M5 |

> **"构造器✅"≠"判定已实现"**：样本能造出来、能算 sha256、真值标签进 manifest，这只证明
> **输入侧**齐备。对照组是否真的全 `pass`，要等对应模块实现后由 `mediatrace audit` 验证——
> 在那之前，本节任何一行都**不得**写成"误报 0 已验证"。

---

## 四、套件与命令

```bash
node tools/tasks.mjs fixtures        # all → fixtures/out + manifest.json
node tools/tasks.mjs check-fixtures  # 双跑逐字节比对（CI 必跑）

# 更细的控制直接调生成器（Go 不在 PATH 时先 export，或传 GO=）
go run ./tools/fixture-gen -list                        # 只打印 case id / sha256 / size
go run ./tools/fixture-gen -suite defects -out /tmp/fx  # 只生成缺陷组
```

**`--check` 的两层断言**（`plan/04 §二.4`）：
1. 双跑生成到两个临时目录 → 两棵树逐字节相等；
2. 与入库 `manifest.json` 的 `sha256` 逐例相等。

---

## 五、manifest 结构

```json
{
  "generator": "fixture-gen",
  "version": "0",
  "seed": 20261027,
  "cases": [
    {
      "case_id": "C1",
      "suite": "defects",
      "filename": "c1_dual_moov.mp4",
      "sha256": "<64 hex>",
      "size_bytes": 0,
      "seed": 20261027,
      "expect_status": "fail",
      "expected_findings": ["duplicate_moov"],
      "gray_reason": "<仅 gray 例出现>",
      "expected_offset": 1234,
      "note": "复制整个 moov 盒插在第二处..."
    }
  ]
}
```

- `expect_status` ∈ `pass|fail|gray`（三态，与报告同一口径）。
- `expected_findings` 为空 **且** `expect_status == "pass"` = 对照组；`gray` 例两条都必填
  `gray_reason`（闭集见 `data/README §5.5`）。
- `expected_offset` = **构造时就知道**的字节偏移（`plan/04` T5）；只对能唯一指认字段的例填，
  其余例省略（当前 92 例中 42 例有）。
- **不写**"实现算出来的具体数值"——那会让真值变成自证（`plan/04` T1）。

### 当前规模（2026-10-27 实测）

| 套件 | 例数 |
|---|---|
| `defects` | 32 |
| `controls` | 16 |
| `m3-regression` | 24 |
| `m4` | 10 |
| `m5` | 10 |
| **合计** | **92**（pass 50 / fail 38 / gray 4） |
