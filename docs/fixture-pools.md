# 夹具池与构造纪律（C1–C7）

> 状态：**骨架期**——C1–C3 的**构造器**已实现（能把样本字节造出来并进 manifest）；
> 这**不等于**判定已实现：模块 M1 的规则仍是 `ErrNotImplemented`（待 M2）。
> C4–C7 的构造器待 M1 补齐。权威编号：`data/README.md §4`。
> 实现：`tools/fixture-gen/gen/gen.go`；清单：`fixtures/out/manifest.json`（生成物，不入库）。

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

| 编号 | 类 | 手法 | 期望 | 状态 |
|---|---|---|---|---|
| **C1** | 双 `moov` | 复制整个 `moov` 插在第二处 | `duplicate_moov`，**报两处偏移** | 🟡 构造器✅ / 判定待 M2 |
| **C2** | `stco` 越界 | chunk offset 指向 `mdat` 载荷之外 | `stco_out_of_mdat` + 条目号 + 区间 | 🟡 构造器✅ / 判定待 M2 |
| **C3** | 截断 `mdat` | 声明 size 大于实际剩余字节 | `truncated_mdat` + 声明 vs 实际 | 🟡 构造器✅ / 判定待 M2 |
| **C4** | 时长口径互斥 | **C4a**：`mvhd.duration` ≠ `Σ(stts)`（含 2× 形态）；**C4b**：起点 + 时长越界 | `duration_mismatch` + 三口径原值 | ⬜ 待 M1 |
| **C5** | 回退 DTS | `stts` 插入 duration 使累计回退 | `dts_regression` + 样本序号 + 偏移 | ⬜ 待 M1 |
| **C6** | 1024 samples 延迟 | 音频首帧 PTS 后移 1024，`elst` 未裁剪 | `priming_undeclared`，**报出 1024** | ⬜ 待 M1 |
| **C7** | `matrix=±90` 无 `pasp` | `tkhd` matrix 置旋转，`width<height`，删 `pasp` | `rotation_dimension_conflict` | ⬜ 待 M1 |

> **C4 不得写负数字段**：`mvhd/tkhd.duration` 是无符号整数。C4 分 C4a/C4b 两态，
> 落盘时必须写明用的是哪一个（`data/README §4` 末段）。

---

## 三、对照组（证明"不误伤"）

| 组 | 规模 | 期望 | 状态 |
|---|---|---|---|
| M3 回归集 | **≥20** 个正常 H.264/H.265（**含 B 帧**） | 全 `pass`，误报 **0** | ⬜ 待 M1 |
| M4 正反例 | 各 **5** | 正确裁剪不报 / 未声明报 | ⬜ 待 M1 |
| M5 正反例 | 各 **5** | 正常竖屏不报 / 冲突报 | ⬜ 待 M1 |

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
      "expected_findings": ["duplicate_moov"],
      "note": "复制整个 moov 盒插在第二处..."
    }
  ]
}
```

- `expected_findings` 为空 = 期望 `pass`（对照组）。
- **不写**"实现算出来的具体数值"——那会让真值变成自证（`plan/04` T1）。
