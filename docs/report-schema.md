# 报告格式规范（`pass` / `fail` / `gray` 三态）

> 状态：**定稿（M5）**。口径真相源：`data/README.md §5.5–§5.7`。
> 实现：`internal/report/report.go`（模型 + 校验 + 渲染），`cmd/mediatrace`（入口）。

---

## 一、JSON 结构

```json
{
  "tool": "mediatrace",
  "version": "0.1.1",
  "file": "sample.mp4",
  "modules": ["M1", "M2"],
  "findings": [
    {
      "module": "M2",
      "rule_id": "duration_mismatch",
      "status": "fail",
      "byte_offset": 45122,
      "field_path": "moov/trak[0]/edts/elst[1].media_time",
      "values": [
        { "name": "stts_sum",       "value": 1000, "unit": "tick" },
        { "name": "mvhd_duration",  "value": 1000, "unit": "tick" },
        { "name": "elst_sum",       "value": 1002, "unit": "tick" }
      ],
      "delta": { "value": 2, "unit": "tick" },
      "message": "three duration accounts disagree beyond tolerance"
    },
    {
      "module": "M3",
      "rule_id": "vfr_detected",
      "status": "gray",
      "gray_reason": "vfr_detected",
      "message": "stts has 3 entries; constant-frame-rate inference unavailable"
    }
  ],
  "summary": {
    "pass": 0,
    "fail": 1,
    "gray": 1,
    "by_module_fail": { "M2": 1 }
  }
}
```

### 1.1 字段约束（渲染层强制校验，缺项即拒绝输出）

| 字段 | `pass` | `fail` | `gray` |
|---|---|---|---|
| `byte_offset` | 可省 | **必填**（`0x` 十六进制；0 是合法值） | 可省 |
| `field_path` | 可省 | **必填**（斜杠盒路径，索引从 0 起，文件内顺序） | 可省 |
| `values` | 可省 | **必填**（≥1 项，整数 + 单位） | 可省 |
| `delta` | 可省 | **必填**（整数 + 单位） | 可省 |
| `gray_reason` | 禁止 | 禁止 | **必填**，闭集枚举 |

**`gray_reason` 闭集**（`data/README §5.5`）：`vfr_detected` / `no_roll_distance` /
`unknown_codec_priming` / `media_rate_not_one` / `no_edts` / `hdr_metadata_missing` /
`timescale_suspect` / `duration_unknown`。

**禁止**：`parse_error` 之类"解析失败"不得作为 `gray_reason`——解析失败走退出码 3，不出报告。

---

## 二、首屏三段（字面与顺序固定）

```
VERDICT: FAIL (3 findings: M1=1 M2=2, gray=2)
FIRST: moov/trak[0]/edts/elst[1].media_time @0x0000B042 (elst item #1, delta=-1024 samples)
GATE: synthetic 25/25 real 20/20 false_positive 0
```

- **`VERDICT`**：`PASS` / `FAIL` / `INCONCLUSIVE`，附计数。
- **`FIRST`**：**必须**给出 `byte_offset` 与 `field_path`，否则视为不合格
  （`plan/01 §七.3`）。无 fail/gray 时为 `FIRST: none`。
- **`GATE`**：**只在 `--suite` 批跑时出现**；单文件审计不打印（避免"单文件全绿 = 通过门禁"的误读，
  `data/README §5.6`）。
- **首期决定（M5）**：`--suite` 批跑模式**首期不实现**，因此 `GATE` 段**不出现**。门禁三数
  （合成样本数 / 真实样本数 / 误报数）改由 `node tools/tasks.mjs all` 的夹具门与对账测试承担；
  单文件审计只打印 `VERDICT` 与 `FIRST` 两段。此决定与 `plan/01 §七.3`、`README.md` 同步。

---

## 三、Markdown 渲染规则

- 与 JSON **同源**：先算一次 `Report`，再渲染两份（`plan/01 §七.3`）。
- 只列 `fail` 与 `gray`；`pass` 计数进 `VERDICT` 行。
- `fail` 下逐项列出 `offset` / `path` / 每个 `values` / `delta`。
- **默认不写绝对路径**：只写 `filepath.Base(path)`。

---

## 四、退出码

| 码 | 语义 | 条件 |
|---|---|---|
| `0` | PASS | `fail == 0 && gray == 0` |
| `1` | FAIL | `fail >= 1`（不论 gray） |
| `2` | INCONCLUSIVE | `fail == 0 && gray >= 1` |
| `3` | 用法/解析错误 | 参数非法、不可读、盒子树无法建立 |

唯一实现处：`internal/policy.ExitCodeFor`（测试逐位断言优先级 `1 > 2 > 0`）。

---

## 五、测试点（对应 `plan/03 §七`）

| # | 断言 | 测试 |
|---|---|---|
| G1 | fail 四元组缺一即拒绝 | `internal/report` `TestFailRequiresAllFourEvidenceFields` |
| G2 | gray reason 必须闭集 | `TestGrayReasonMustBeClosedSet` |
| G3 | gray 不计入 fail | `TestBuildSummaryExcludesGrayFromFail` |
| G4 | JSON/MD 同源 | `TestJSONAndMarkdownShareSameFindings` |
| G5 | 退出码 0/1/2/3 | `internal/policy` + `cmd/mediatrace` |
| G6 | 双跑逐字节 | `tools/fixture-gen` `TestBuildIsDeterministic` |
