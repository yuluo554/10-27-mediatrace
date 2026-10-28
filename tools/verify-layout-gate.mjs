#!/usr/bin/env node
// ═══ 布局门 / 脱敏门 自己的变异验证（"这条门真会红吗？"）═══
// 来源：ai-tool-project-sprint SKILL.md 硬约束③ + references/methodology-binding.md「假门族」。
// 纪律：**布局门有几条判据，就要有几组变异**；每组断言"字节真的变了" + "门真的红了" + "还原后复绿"。
//   · 变异靶子不得依赖"某个特定进度状态"（例如"找到一个待办里程碑翻 ✅"）——
//     收尾后全部 ✅ 时该靶子消失、脚本自崩（假门族 ⑭）。本项目一律用"直接改/删某文件"的形态。
//   · 阴性对照：做完变异并还原后，原样跑一遍必须绿（证明还原干净）。
//
// 用法：node tools/verify-layout-gate.mjs        （会临时改仓库文件，退出前必还原）
// 退出码：0 全部按预期 / 1 有假门 / 2 用法错误。

import { cpSync, existsSync, mkdirSync, mkdtempSync, readFileSync, renameSync, rmSync, writeFileSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { dirname, join, resolve } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const LAYOUT = join(ROOT, "tools", "check-sprint-layout.mjs");
const DESENS = join(ROOT, "tools", "check-desensitized.mjs");

const results = [];
const record = (id, ok, detail) => results.push({ id, ok, detail });

function runGate(script, args = []) {
  const r = spawnSync(process.execPath, [script, ...args], { cwd: ROOT, encoding: "utf8" });
  return { code: r.status, out: r.stdout ?? "", err: r.stderr ?? "" };
}

/** 临时改文件：返回 restore 函数。restore 必须断言字节真的回到原样。 */
function mutate(rel, transform) {
  const abs = join(ROOT, rel);
  const original = readFileSync(abs, "utf8");
  const mutated = transform(original);
  if (mutated === original) {
    throw new Error(`变异无效果（transform 没改字节）: ${rel}`);
  }
  writeFileSync(abs, mutated);
  return () => {
    writeFileSync(abs, original);
    if (readFileSync(abs, "utf8") !== original) throw new Error(`还原失败: ${rel}`);
  };
}

/** 临时重命名：返回 restore。 */
function hide(rel) {
  const abs = join(ROOT, rel);
  const tmp = `${abs}.mutbak`;
  renameSync(abs, tmp);
  return () => {
    if (!existsSync(tmp)) throw new Error(`备份丢失: ${tmp}`);
    renameSync(tmp, abs);
  };
}

function expectFail(id, script, mutateFn, expectCheckId) {
  let restore;
  try {
    restore = mutateFn();
    // 布局门支持 --json（用于断言"红在哪个判据"）；脱敏门不支持
    const r = runGate(script, script === LAYOUT ? ["--json"] : []);
    if (r.code === 0) {
      record(id, false, `门仍然绿（假门）：应被 ${expectCheckId} 抓到`);
      return;
    }
    if (expectCheckId) {
      const failed = (() => {
        try {
          return JSON.parse(r.out).checks.filter((c) => !c.ok).map((c) => c.id);
        } catch {
          return [];
        }
      })();
      if (!failed.includes(expectCheckId)) {
        record(id, false, `红了但不是 ${expectCheckId}；实际红: ${failed.join(",") || "(无法解析)"}`);
        return;
      }
      record(id, true, `变红于 ${expectCheckId}`);
      return;
    }
    record(id, true, `变红（exit ${r.code}）`);
  } catch (e) {
    record(id, false, `变异过程异常: ${e.message}`);
  } finally {
    try {
      restore?.();
    } catch (e) {
      record(`${id}.restore`, false, `还原异常: ${e.message}`);
    }
  }
}

function expectGreen(id, script) {
  const r = runGate(script);
  record(id, r.code === 0, r.code === 0 ? "绿（阴性对照成立）" : `应绿但红了: ${(r.out || r.err).slice(0, 300)}`);
}

/**
 * 在"里程碑状态表"里把某里程碑的**状态格**改成 cell（只改一处副本）。
 *
 * **必须按表头定位（行与列都要）**——这是同一类缺陷踩过两次（plan/06 §六 B2 与 B2b）：
 *   · 早先按"行首像 M1"定位行 → 命中 `plan/00 §四` 提交材料表，改坏核验命令列；
 *   · 早先按"最后一个单元格"定位列 → `plan/05` 的最后一列是**出口门**而非状态，
 *     于是状态格根本没动、行内仍含 ⬜，门报 `LAYOUT.STATUS` 冲突（而预期是 HANDOFFS 红）。
 * 本项目的状态表列序：plan/00 与 README 是「… | 状态 |」（末列即状态），
 * plan/05 是「阶段 | 名称 | 状态 | 出口门」——**只有按表头找"状态"列才两边都对**。
 */
function setStatus(rel, key, cell) {
  return mutate(rel, (t) => {
    const lines = t.split("\n");
    const headerIdx = lines.findIndex(
      (l) => l.startsWith("|") && l.includes("状态") && (l.includes("阶段") || l.includes("里程碑")),
    );
    if (headerIdx < 0) throw new Error(`${rel} 找不到里程碑状态表头`);
    const headers = lines[headerIdx].split("|").map((c) => c.trim());
    // 表头首尾是空串，单元格下标 = headers 里的下标
    const statusCol = headers.findIndex((h) => h.includes("状态"));
    if (statusCol < 0) throw new Error(`${rel} 状态表里找不到「状态」列（表头: ${headers.join(" / ")}）`);
    const re = new RegExp(`^\\|\\s*\\**\\s*${key}\\b`);
    let idx = -1;
    for (let i = headerIdx + 2; i < lines.length; i++) {
      if (!lines[i].startsWith("|")) break;
      if (re.test(lines[i])) {
        idx = i;
        break;
      }
    }
    if (idx < 0) throw new Error(`里程碑行 ${key} 未找到于 ${rel} 的状态表（结构变了？）`);
    const cells = lines[idx].split("|");
    const prev = cells[statusCol].trim();
    if (prev === cell) throw new Error(`${rel} 的 ${key} 状态已是 ${cell}，变异无意义`);
    cells[statusCol] = ` ${cell} `;
    lines[idx] = cells.join("|");
    return lines.join("\n");
  });
}

/**
 * 组合多个变异，**任一在应用阶段抛错都要把已应用的还原回来**。
 *
 * 早先版本是 `fns.forEach(fn => fn())` 直接串起来：第 2 个抛错时第 1 个已落盘的改动
 * **永不还原** → 仓库被污染（本项目实测：三个文档被改坏，门随即变红，看起来像"门坏了"）。
 * 正确形态 = 逐个 apply，失败即回滚已成功的那些，再抛出。
 */
function compose(...factories) {
  const restores = [];
  try {
    for (const factory of factories) restores.push(factory());
  } catch (e) {
    for (const r of restores.reverse()) {
      try {
        r();
      } catch {
        /* 回滚失败要保留原始错误，下面抛出 */
      }
    }
    throw e;
  }
  return () => {
    const errs = [];
    for (const r of restores.reverse()) {
      try {
        r();
      } catch (e) {
        errs.push(e.message);
      }
    }
    if (errs.length) throw new Error(errs.join("; "));
  };
}

/**
 * 读「里程碑状态表」得到 (key, pending) 列表。**与布局门同一定位口径**
 * （表头同行含「状态」且含「阶段/里程碑」；表结束即停）。
 *
 * 为什么要这个：本脚本早先把靶子写死成"M1 翻 ✅"（`flipM1()`）。M1 一旦真的收尾翻 ✅，
 * `setStatus` 就会抛"变异无意义"、`hide("plan/HANDOFF-M1.md")` 也会红错判据 ——
 * 脚本在里程碑推进时**自崩**。这正是文件头警告的"靶子依赖进度状态"。
 * 现在一律**从仓库状态现算**靶子，任一进度下都指得到真实可变的行/文件。
 */
function statusRows(rel = "plan/00-README总览.md") {
  const lines = readFileSync(join(ROOT, rel), "utf8").split("\n");
  const headerIdx = lines.findIndex(
    (l) => l.startsWith("|") && l.includes("状态") && (l.includes("阶段") || l.includes("里程碑")),
  );
  if (headerIdx < 0) throw new Error(`${rel} 找不到里程碑状态表头`);
  const rows = [];
  for (let i = headerIdx + 2; i < lines.length; i++) {
    if (!lines[i].startsWith("|")) break;
    if (/^\|\s*-{2,}/.test(lines[i])) continue;
    const m = /^\|\s*\**\s*(M\d+)\b/.exec(lines[i]);
    if (!m) continue;
    rows.push({ key: m[1], pending: /[⬜🟡]|未开工|未开始|未执行|待执行|待确认|进行中/.test(lines[i]) });
  }
  if (rows.length === 0) throw new Error(`${rel} 状态表 0 行（判据空跑，变异无处落脚）`);
  return rows;
}

/** 当前棒 = 第一个待办；lastDone = 最后一个 ✅。GAPS/LEDGER 类变异需要至少一个 ✅ 靶子。 */
function milestones() {
  const rows = statusRows();
  const pending = rows.filter((r) => r.pending);
  const done = rows.filter((r) => !r.pending);
  if (done.length === 0) throw new Error("没有任何 ✅ 里程碑：GAPS/LEDGER/HANDOFFS 类变异无处落脚");
  return { first: rows[0].key, current: pending[0]?.key ?? null, lastDone: done[done.length - 1].key };
}

/** 三处副本同一目标格一起翻（`compose` 的工厂形态）。 */
const flipAll = (key, cell) => [
  () => setStatus("plan/00-README总览.md", key, cell),
  () => setStatus("plan/05-里程碑与发布.md", key, cell),
  () => setStatus("README.md", key, cell),
];

/** 只改一处副本：有待办就翻 ✅，全完成则把首个里程碑翻回 ⬜（两种进度下都是真变化）。 */
function statusMutation(rel) {
  const rows = statusRows();
  const cur = rows.find((r) => r.pending);
  return cur ? setStatus(rel, cur.key, "✅ 已完成") : setStatus(rel, rows[0].key, "⬜ 未开工");
}

/** 在"缺口表"首个数据行的最后一个单元格尾部追加文本（结构定位，不依赖具体文案）。 */
function appendToFirstGapCell(rel, extra) {
  return mutate(rel, (t) => {
    const lines = t.split("\n");
    const headerIdx = lines.findIndex((l) => l.startsWith("|") && l.includes("缺口"));
    if (headerIdx < 0) throw new Error(`${rel} 找不到含『缺口』表头的表`);
    let idx = -1;
    for (let i = headerIdx + 2; i < lines.length; i++) {
      if (!lines[i].startsWith("|")) break;
      if (/^\|\s*-{2,}/.test(lines[i])) continue;
      idx = i;
      break;
    }
    if (idx < 0) throw new Error(`${rel} 缺口表没有数据行`);
    if (!lines[idx].trimEnd().endsWith("|")) throw new Error(`${rel} 缺口行不以 | 结尾（结构变了）`);
    const trimmed = lines[idx].trimEnd();
    lines[idx] = `${trimmed.slice(0, -1).trimEnd()} ${extra} |`;
    return lines.join("\n");
  });
}

/** 在"承诺-证据台账"表头下方插一行（状态列由调用方给）。 */
function insertLedgerRow(rel, statusCell) {
  return mutate(rel, (t) => {
    const lines = t.split("\n");
    const headerIdx = lines.findIndex((l) => l.startsWith("|") && l.includes("承诺") && l.includes("状态"));
    if (headerIdx < 0) throw new Error(`${rel} 找不到『承诺-证据台账』表头`);
    if (!/^\|\s*-{2,}/.test(lines[headerIdx + 1] ?? "")) throw new Error(`${rel} 台账表头下不是分隔行（结构变了）`);
    lines.splice(headerIdx + 2, 0, `| 台账变异靶子 | 变异 | 纯本地 | grep | ${statusCell} |`);
    return lines.join("\n");
  });
}

/**
 * 造一个"当前棒的真实入口交接单"靶子：返回 { rel, pre }。
 *   · 有待办里程碑 → 它就是当前棒，`pre` 为空（直接删/改它的交接单即可）；
 *   · 全部 ✅（末棒之后）→ `HANDOFF_ENTRY` 退化为"不适用"（绿），必须先三处副本一致
 *     把首个里程碑翻回 ⬜ 造出一个当前棒，否则本组变异**无红可验**（就是假门）。
 */
function makeCurrentLeg() {
  if (CURRENT) return { rel: `plan/HANDOFF-${CURRENT}.md`, pre: [] };
  return { rel: `plan/HANDOFF-${FIRST}.md`, pre: flipAll(FIRST, "⬜ 未开工") };
}

// ── 0. 基线：两门必须绿 ────────────────────────────────────────────────
expectGreen("BASELINE.layout", LAYOUT);
expectGreen("BASELINE.desens", DESENS);

// ── 布局门变异（每条判据一组）─────────────────────────────────────────
// 注意：`compose(...)` 收的是**工厂**（`() => restore`），不是已应用的 restore——
// 只有这样，第 2 个工厂抛错时才能回滚第 1 个（见 compose 的注释）。
const { first: FIRST, current: CURRENT, lastDone: LAST_DONE } = milestones();

expectFail("M1.FILES", LAYOUT, () => hide("LICENSE"), "LAYOUT.FILES");
expectFail("M2.BINDING_HEAD", LAYOUT, () => mutate("plan/00-README总览.md", (t) => t.replace(/^>.*ai-tool-project-sprint.*$/m, "> （装订行被删）")), "LAYOUT.BINDING_HEAD");
// plan/02 的「分层」只出现一次，删标题即真的缺该小节
expectFail("M3.SECTIONS", LAYOUT, () => mutate("plan/02-架构与技术选型.md", (t) => t.replace("## 一、分层", "## 一、其它")), "LAYOUT.SECTIONS");
// 只改一处副本的状态 → 跨副本不一致（靶子按当前棒现算）
expectFail("M4.STATUS", LAYOUT, () => statusMutation("README.md"), "LAYOUT.STATUS");
// 某个里程碑在三处副本一致翻 ✅，同时把它的交接单藏起来 → 交接单判据必须红。
// 靶子：当前棒（有待办）或最后一个已 ✅（全完成）——两种进度下都成立。
expectFail(
  "M5.HANDOFFS",
  LAYOUT,
  () =>
    CURRENT
      ? compose(...flipAll(CURRENT, "✅ 已完成"), () => hide(`plan/HANDOFF-${CURRENT}.md`))
      : () => hide(`plan/HANDOFF-${LAST_DONE}.md`),
  "LAYOUT.HANDOFFS",
);
// 当前棒的入口交接单被删 → HANDOFF_ENTRY 必须红（本项目实测漏写过一次）
expectFail("M5b.HANDOFF_ENTRY", LAYOUT, () => {
  const { rel, pre } = makeCurrentLeg();
  return compose(...pre, () => hide(rel));
}, "LAYOUT.HANDOFF_ENTRY");
// 入口交接单缺节 → 同一判据必须红（否则"有文件就算过"仍是假门）
expectFail("M5c.HANDOFF_ENTRY_SECTIONS", LAYOUT, () => {
  const { rel, pre } = makeCurrentLeg();
  return compose(...pre, () => mutate(rel, (t) => t.replace("## 二、当前事实", "## 二、其它")));
}, "LAYOUT.HANDOFF_ENTRY");
// 在 plan/06 的缺口格写入已 ✅ 的里程碑
expectFail(
  "M6.GAPS",
  LAYOUT,
  () => appendToFirstGapCell("plan/06-交付对标与决策.md", `仍挂 ${LAST_DONE}`),
  "LAYOUT.GAPS",
);
// 往承诺台账塞一行"未完成但挂已 ✅ 的里程碑"
expectFail(
  "M7.LEDGER",
  LAYOUT,
  () => insertLedgerRow("plan/00-README总览.md", `⬜ 待 ${LAST_DONE}`),
  "LAYOUT.LEDGER",
);
expectFail("M8.DOC_TOOLS", LAYOUT, () =>
  appendToFirstGapCell("plan/06-交付对标与决策.md", "见 `tools/not-exist-gate.mjs`"),
  "LAYOUT.DOC_TOOLS");
expectFail("M9.GITIGNORE", LAYOUT, () => mutate(".gitignore", (t) => t.replace(/^\.mimosa\/?$/m, "# (规则被删)")), "LAYOUT.GITIGNORE");
// 发布里程碑（状态表里最大的号）翻 ✅ 但没有 RELEASE-M6.md → 留档判据必须红。
//
// **必须与进度无关**：早先写成二分（"未 ✅ 就翻它 / 已 ✅ 就藏留档"），
// 前提是"留档文件还不存在"——留档一落盘，翻 ✅ 反而让 LAYOUT.RELEASE 变绿，
// 于是红的只剩 LAYOUT.LEDGER（台账行仍挂 M6），被判"红了但不是 LAYOUT.RELEASE"。
// 这正是文件头警告的"靶子依赖进度状态"的第三次变体（本项目 M6 首推实测）。
// 现在统一：**翻 ✅（若还是待办）+ 藏留档**，两种进度下留档判据都必须红。
const releaseKey = statusRows().map((r) => Number(r.key.slice(1))).reduce((a, b) => Math.max(a, b), 0);
const REL = `M${releaseKey}`;
const releasePending = statusRows().find((r) => r.key === REL)?.pending ?? false;
expectFail(
  "M10.RELEASE",
  LAYOUT,
  () =>
    compose(
      ...(releasePending ? flipAll(REL, "✅ 已完成") : []),
      () => (existsSync(join(ROOT, `plan/RELEASE-${REL}.md`)) ? hide(`plan/RELEASE-${REL}.md`) : () => {}),
      () => (existsSync(join(ROOT, `plan/WRAPUP-${REL}.md`)) ? hide(`plan/WRAPUP-${REL}.md`) : () => {}),
    ),
  "LAYOUT.RELEASE",
);

// ── 脱敏门变异（4 条规则各一组）──────────────────────────────────────
const probe = "docs/_desens-probe.md";
mkdirSync(join(ROOT, "docs"), { recursive: true });
writeFileSync(join(ROOT, probe), "clean line\n");
// 探针字面量必须**运行时拼装**：本文件自己也要过脱敏门（SELF 只豁免规则文件），
// 写死字面量会让"门扫自己"变成假红。拼装也不影响判据——落盘文本仍含完整形态。
const USER = "AS" + "US";
const variants = [
  ["V1", "absolute-session-path", "see " + "D:" + "/Pro" + "gramData/somewhere/x\n"],
  ["V2", "host-username", "user is " + USER + " here\n"],
  ["V3", "msys-user-path", "path " + "/c/Users/" + USER + "/x\n"],
  ["V4", "loopback-proxy", "proxy " + "127." + "0.0.1:" + "7897\n"],
];
for (const [id, rule, line] of variants) {
  writeFileSync(join(ROOT, probe), line);
  const r = runGate(DESENS, []);
  const ok = r.code === 1 && r.out.includes(`[${rule}]`);
  record(`DESENS.${id}`, ok, ok ? `命中 ${rule}` : `应命中 ${rule}，实际 exit=${r.code} out=${(r.out || r.err).slice(0, 160)}`);
}
// 占位符必须放行（假红防线）
writeFileSync(join(ROOT, probe), "path C:\\Users\\<user>\\repo and /c/Users/<user>/x\n");
const rPlaceholder = runGate(DESENS, []);
record("DESENS.PLACEHOLDER", rPlaceholder.code === 0, rPlaceholder.code === 0 ? "占位符放行" : `占位符被误判: ${rPlaceholder.out.slice(0, 160)}`);
rmSync(join(ROOT, probe), { force: true });

// ── 阴性对照：全部还原后再跑一遍 ──────────────────────────────────────
expectGreen("FINAL.layout", LAYOUT);
expectGreen("FINAL.desens", DESENS);

// ── 汇总 ──────────────────────────────────────────────────────────────
const failed = results.filter((r) => !r.ok);
process.stdout.write("变异验证（门自己的假门检验）\n");
process.stdout.write("=".repeat(64) + "\n");
for (const r of results) process.stdout.write(`${r.ok ? "PASS" : "FAIL"}  ${r.id.padEnd(24)} ${r.detail}\n`);
process.stdout.write("=".repeat(64) + "\n");
process.stdout.write(`共 ${results.length} 项，${failed.length === 0 ? "全绿（门的判据都被证伪过一次）" : `${failed.length} 项疑似假门`}\n`);
process.exitCode = failed.length === 0 ? 0 : 1;
