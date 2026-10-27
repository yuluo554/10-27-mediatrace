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
 * 在"里程碑状态表"里把某里程碑的状态格改成 cell（只改一处副本）。
 *
 * **必须按表头定位**（与布局门的 `milestoneRows` 同一口径）：早先用"行首像 M1 就改"
 * 会命中 `plan/00 §四` 的提交材料表（首格 `M1 结构规则`），改坏的是**核验命令列**，
 * 而且 status 根本没动 —— 变异静默无害、门也不红（本项目实测）。
 */
function setStatus(rel, key, cell) {
  return mutate(rel, (t) => {
    const lines = t.split("\n");
    const headerIdx = lines.findIndex(
      (l) => l.startsWith("|") && l.includes("状态") && (l.includes("阶段") || l.includes("里程碑")),
    );
    if (headerIdx < 0) throw new Error(`${rel} 找不到里程碑状态表头`);
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
    const prev = cells[cells.length - 2].trim();
    if (prev === cell) throw new Error(`${rel} 的 ${key} 状态已是 ${cell}，变异无意义`);
    cells[cells.length - 2] = ` ${cell} `;
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

// ── 0. 基线：两门必须绿 ────────────────────────────────────────────────
expectGreen("BASELINE.layout", LAYOUT);
expectGreen("BASELINE.desens", DESENS);

// ── 布局门变异（每条判据一组）─────────────────────────────────────────
// 注意：`compose(...)` 收的是**工厂**（`() => restore`），不是已应用的 restore——
// 只有这样，第 2 个工厂抛错时才能回滚第 1 个（见 compose 的注释）。
const GAP_CELL_TEXT = "`tools/fixture-gen/gen/gen.go` 缺 C4a/C4b、C5、C6、C7 与 m3-regression/controls 套件";

expectFail("M1.FILES", LAYOUT, () => hide("LICENSE"), "LAYOUT.FILES");
expectFail("M2.BINDING_HEAD", LAYOUT, () => mutate("plan/00-README总览.md", (t) => t.replace(/^>.*ai-tool-project-sprint.*$/m, "> （装订行被删）")), "LAYOUT.BINDING_HEAD");
// plan/02 的「分层」只出现一次，删标题即真的缺该小节
expectFail("M3.SECTIONS", LAYOUT, () => mutate("plan/02-架构与技术选型.md", (t) => t.replace("## 一、分层", "## 一、其它")), "LAYOUT.SECTIONS");
// 只改一处副本的状态 → 跨副本不一致
expectFail("M4.STATUS", LAYOUT, () => setStatus("README.md", "M2", "✅ 已完成"), "LAYOUT.STATUS");
const flipM1 = () => [
  () => setStatus("plan/00-README总览.md", "M1", "✅ 已完成"),
  () => setStatus("plan/05-里程碑与发布.md", "M1", "✅ 已完成"),
  () => setStatus("README.md", "M1", "✅ 已完成"),
];
// 三处副本一致翻 ✅，但没有 HANDOFF-M1 → 交接单判据必须红
expectFail("M5.HANDOFFS", LAYOUT, () => compose(...flipM1()), "LAYOUT.HANDOFFS");
// 同上，再在 plan/06 的缺口格写入已 ✅ 的 M1
expectFail("M6.GAPS", LAYOUT, () =>
  compose(
    ...flipM1(),
    () => mutate("plan/06-交付对标与决策.md", (t) => t.replace(GAP_CELL_TEXT, GAP_CELL_TEXT + " 仍挂 M1")),
  ), "LAYOUT.GAPS");
// 同上，再往承诺台账塞一行"未完成但挂 M1"
expectFail("M7.LEDGER", LAYOUT, () =>
  compose(
    ...flipM1(),
    () => mutate("plan/00-README总览.md", (t) =>
      t.replace("| 三口径时长互比", "| 台账变异靶子 | 构造 | 纯本地 | grep | ⬜ 待 M1 |\n| 三口径时长互比")),
  ), "LAYOUT.LEDGER");
expectFail("M8.DOC_TOOLS", LAYOUT, () =>
  mutate("plan/06-交付对标与决策.md", (t) => t.replace(GAP_CELL_TEXT, GAP_CELL_TEXT + " + `tools/not-exist-gate.mjs`")),
  "LAYOUT.DOC_TOOLS");
expectFail("M9.GITIGNORE", LAYOUT, () => mutate(".gitignore", (t) => t.replace(/^\.mimosa\/?$/m, "# (规则被删)")), "LAYOUT.GITIGNORE");
// 发布里程碑翻 ✅ 但没有 RELEASE-M6.md → 留档判据必须红
expectFail("M10.RELEASE", LAYOUT, () =>
  compose(
    () => setStatus("plan/00-README总览.md", "M6", "✅ 已完成"),
    () => setStatus("plan/05-里程碑与发布.md", "M6", "✅ 已完成"),
    () => setStatus("README.md", "M6", "✅ 已完成"),
  ), "LAYOUT.RELEASE");

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
