#!/usr/bin/env node
// ═══ 布局门（方法论装订第②件）· 10-27-mediatrace ═══
// 来源：ai-tool-project-sprint references/templates/check-sprint-layout.mjs（2026-10-27 拷入并按本项目改写）。
//   · 判据只落**仓库内文件**（不读 git 元数据：浅克隆下 tag/历史深度会翻转）。
//   · 每加一条判据都要在 tools/verify-layout-gate.mjs 里加一组变异并**逐组验红**。
//   · 每条判据都要报"这轮扫到了几个"——否则"0 命中"与"什么都没扫"输出相同。
// 退出码：0 全绿 / 1 有 FAIL / 2 用法错误。
//
// ── 本项目的判据集（共 11 条）─────────────────────────────────────────────
//   LAYOUT.FILES         必备文件齐（plan/00-06、data 台账、docs、代码骨架、tools、CI）
//   LAYOUT.BINDING_HEAD  plan/00 顶部装订行（方法论名 + 门命令 + 装订日期 同现一行）
//   LAYOUT.SECTIONS      各文档必需小节
//   LAYOUT.STATUS        里程碑状态在多处副本上互相一致（逐副本命中行数可见）
//   LAYOUT.HANDOFF_ENTRY **当前棒必须有七节入口交接单**（本项目实测漏项后新增）
//   LAYOUT.HANDOFFS      每个 ✅ 里程碑有七节交接单
//   LAYOUT.GAPS          plan/06 缺口列不得再提已 ✅ 的里程碑
//   LAYOUT.LEDGER        plan/00 承诺-证据台账（第四份状态副本）
//   LAYOUT.RELEASE       发布留档存在且两栏齐（仅在发布里程碑 ✅ 后要求）
//   LAYOUT.DOC_TOOLS     文档点名的 tools/ 脚本必须有实物
//   LAYOUT.GITIGNORE     本地状态/产物不得入库（.mimosa/ bin/ 媒体样本）
// 另：失败时 --json 输出（供 verify-layout-gate.mjs 解析"红在哪个判据"）。

import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const JSON_OUT = process.argv.includes("--json");

const checks = [];
const add = (id, ok, detail) => checks.push({ id, ok, detail });
const read = (rel) => (existsSync(join(ROOT, rel)) ? readFileSync(join(ROOT, rel), "utf8") : null);

const scanned = { docs: 0, handoffs: 0, copies: 0, docTools: 0, statusRows: 0 };

const REQUIRED_FILES = [
  "AGENTS.md",
  "README.md",
  "LICENSE",
  "Makefile",
  "go.mod",
  "data/README.md",
  "data/real-samples.json",
  "plan/00-README总览.md",
  "plan/01-题目文档.md",
  "plan/02-架构与技术选型.md",
  "plan/03-模块详设.md",
  "plan/04-数据计划.md",
  "plan/05-里程碑与发布.md",
  "plan/06-交付对标与决策.md",
  "plan/HANDOFF-M1.md",
  "docs/report-schema.md",
  "docs/rules.md",
  "docs/fixture-pools.md",
  "cmd/mediatrace/main.go",
  "internal/policy/policy.go",
  "internal/report/report.go",
  "internal/isobmff/box.go",
  "tools/fixture-gen/main.go",
  "tools/check-sprint-layout.mjs",
  "tools/check-desensitized.mjs",
  "tools/verify-layout-gate.mjs",
  "tools/tasks.mjs",
  "fixtures/out/manifest.json",
  ".github/workflows/ci.yml",
];

/** 装订行：三个锚点必须同现于一行（前瞻，顺序无关）。 */
const BINDING_LINE = /^>(?=.*ai-tool-project-sprint)(?=.*check-sprint-layout)(?=.*20\d{2}-\d{2}-\d{2}).*$/m;

const REQUIRED_SECTIONS = {
  "plan/00-README总览.md": ["文档索引", "决策记录", "承诺-证据台账", "里程碑"],
  "plan/01-题目文档.md": ["任务要求", "评分标准", "边界与非目标"],
  "plan/02-架构与技术选型.md": ["分层", "构建"],
  "plan/04-数据计划.md": ["数据清单", "真值纪律", "数值口径"],
  "plan/05-里程碑与发布.md": ["DoD", "已知未验证项"],
  "plan/06-交付对标与决策.md": ["评分对标", "风险登记"],
  "data/README.md": ["依据查证记录", "数值口径"],
};

const HANDOFF_SECTIONS = ["当前进度", "当前事实", "待办", "既定口径", "环境坑", "DoD", "命令速查"];

const EXEMPT_MILESTONES = [0];
const HANDOFF_FORMAT_FROM = 1; // 本项目从 M1 起交接单就按七节写
const HANDOFF_EXEMPT_LAST = false; // 末棒也要收官交接单

/**
 * 状态副本：同一里程碑在这几处的状态必须互相一致。
 *
 * **按表头定位**，不是"行首像 M 号就算"——`plan/00 §四` 的提交材料实物表里
 * `| M1 结构规则 | … |` 这类行首格是**模块名**（不是里程碑状态），旧版会把它们
 * 当成"已完成"的里程碑，从而让 HANDOFFS/LEDGER 连环误报（本项目实测）。
 * 定位口径：表头同行含 "状态" 且含 "阶段" 或 "里程碑"。
 */
const STATUS_COPIES = [
  { file: "plan/00-README总览.md", kind: "里程碑表" },
  { file: "plan/05-里程碑与发布.md", kind: "状态表" },
  { file: "README.md", kind: "状态表" },
];

const STATUS_LINE = /^\|\s*\**\s*(M\d+)\b/;
const PENDING_MARKERS = ["⬜", "🟡", "未开工", "未开始", "未执行", "待执行", "待确认", "进行中"];

/** 在文本里定位"里程碑状态表"，返回数据行（首格 M 号的行）。找不到返回 null。 */
function milestoneRows(text) {
  const lines = text.split("\n");
  const headerIdx = lines.findIndex(
    (l) => l.startsWith("|") && l.includes("状态") && (l.includes("阶段") || l.includes("里程碑")),
  );
  if (headerIdx < 0) return null;
  const rows = [];
  for (let i = headerIdx + 2; i < lines.length; i++) {
    if (!lines[i].startsWith("|")) break; // 表结束
    if (/^\|\s*-{2,}/.test(lines[i])) continue; // 分隔行
    if (!STATUS_LINE.test(lines[i])) continue;
    rows.push(lines[i]);
  }
  return rows;
}

function checkRequiredFiles() {
  const missing = REQUIRED_FILES.filter((rel) => !existsSync(join(ROOT, rel)));
  scanned.docs = REQUIRED_FILES.length - missing.length;
  add(
    "LAYOUT.FILES",
    missing.length === 0,
    missing.length ? `缺 ${missing.length} 项: ${missing.join(",")}` : `${scanned.docs} 份必备件齐备`,
  );
}

function checkBindingHead() {
  const text = read("plan/00-README总览.md") ?? "";
  const ok = BINDING_LINE.test(text);
  add("LAYOUT.BINDING_HEAD", ok, ok ? "plan/00 顶部有装订行（三锚点齐）" : "plan/00 顶部缺装订行（需同行含：方法论名 + check-sprint-layout + 装订日期）");
}

function checkSections() {
  const bad = [];
  let okCount = 0;
  for (const [rel, needles] of Object.entries(REQUIRED_SECTIONS)) {
    const text = read(rel);
    if (text === null) {
      bad.push(`${rel}: 文件缺失`);
      continue;
    }
    const missing = needles.filter((n) => !text.includes(n));
    if (missing.length) bad.push(`${rel}: 缺 ${missing.join("/")}`);
    else okCount += 1;
  }
  add(
    "LAYOUT.SECTIONS",
    bad.length === 0,
    bad.length ? bad.join("；") : `${okCount}/${Object.keys(REQUIRED_SECTIONS).length} 份文档小节齐备`,
  );
}

function collectStates() {
  const states = new Map();
  const rowsByCopy = new Map();
  const note = (k, state) => {
    if (!states.has(k)) states.set(k, new Set());
    states.get(k).add(state);
  };
  for (const copy of STATUS_COPIES) {
    const text = read(copy.file);
    if (text === null) {
      rowsByCopy.set(copy.file, null);
      continue;
    }
    scanned.copies += 1;
    const rows = milestoneRows(text);
    if (rows === null) {
      rowsByCopy.set(copy.file, null); // 表头搬家 → 该副本不参与判定，但必须可见
      continue;
    }
    for (const line of rows) {
      const m = STATUS_LINE.exec(line);
      if (!m) continue;
      const pending = PENDING_MARKERS.some((mark) => line.includes(mark));
      note(m[1], pending ? "pending" : "done");
    }
    scanned.statusRows += rows.length;
    rowsByCopy.set(copy.file, rows.length);
  }
  return { states, rowsByCopy };
}

function checkStatusAgreement({ states, rowsByCopy }) {
  const conflicts = [];
  for (const [k, set] of [...states.entries()].sort((a, b) => a[0].localeCompare(b[0], "en", { numeric: true }))) {
    if (set.size > 1) conflicts.push(`${k} 在副本间不一致（${[...set].sort().join(" vs ")}）`);
  }
  const total = [...rowsByCopy.values()].reduce((a, b) => a + (b ?? 0), 0);
  const perCopy = [...rowsByCopy.entries()]
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([file, rows]) => `${file.split("/").pop()}:${rows === null ? "缺文件" : rows}`)
    .join(" ");
  if (total === 0) {
    add("LAYOUT.STATUS", false, `所有副本合计 0 行状态行（判据空跑）：${perCopy}`);
    return;
  }
  add(
    "LAYOUT.STATUS",
    conflicts.length === 0,
    conflicts.length ? conflicts.join("；") : `${states.size} 个里程碑 × ${scanned.copies} 处副本状态全一致（行数 ${perCopy}）`,
  );
}

function checkHandoffs(done, allMilestones) {
  const nums = allMilestones.map((k) => Number(k.slice(1)));
  const last = Math.max(0, ...nums);
  const missing = [];
  let present = 0;
  for (const k of done) {
    const n = Number(k.slice(1));
    if (EXEMPT_MILESTONES.includes(n)) continue;
    if (HANDOFF_EXEMPT_LAST && n === last) continue;
    const rel = `plan/HANDOFF-${k}.md`;
    const text = read(rel);
    if (text === null) {
      missing.push(`${rel}(缺${n === last ? "：末棒要么写收官交接单，要么置 HANDOFF_EXEMPT_LAST 并在 AGENTS.md 说明" : ""})`);
      continue;
    }
    present += 1;
    scanned.handoffs += 1;
    if (n < HANDOFF_FORMAT_FROM) continue;
    const lack = HANDOFF_SECTIONS.filter((s) => !text.includes(s));
    if (lack.length) missing.push(`${rel}(缺节: ${lack.join("/")})`);
  }
  add(
    "LAYOUT.HANDOFFS",
    missing.length === 0,
    missing.length ? missing.join("；") : `${present} 份交接单齐（已扫 ${done.length} 个 ✅ 里程碑）`,
  );
}

/**
 * **当前棒必须有入口交接单**（本项目实测漏项，2026-10-27）。
 *
 * 为什么单列一条：`checkHandoffs` 只对**已 ✅ 的里程碑**要求交接单，
 * 于是"阶段 0 装订完成、M1 未开工"这个状态**没有任何判据要求 `HANDOFF-M1.md` 存在**——
 * 而恰恰在这个状态下，下一棒最需要它（跨对话上下文不可恢复）。
 * 实测后果：骨架期收尾时漏写 `HANDOFF-M1.md`，两道门全绿，由用户发现。
 *
 * 判据 = "当前棒（第一个未完成里程碑）必须有**七节齐**的入口交接单"。
 * 全部完成时退化为"末棒交接单在位"（`HANDOFF_EXEMPT_LAST=false` 下沿用 checkHandoffs）。
 */
function checkEntryHandoff(states, allMilestones) {
  const pending = [...states.entries()]
    .filter(([, set]) => set.has("pending"))
    .map(([k]) => k)
    .sort((a, b) => Number(a.slice(1)) - Number(b.slice(1)));
  if (pending.length === 0) {
    add("LAYOUT.HANDOFF_ENTRY", true, `无待办里程碑（已扫 ${allMilestones.length} 个），入口交接单不适用`);
    return;
  }
  const current = pending[0];
  const rel = `plan/HANDOFF-${current}.md`;
  const text = read(rel);
  if (text === null) {
    add("LAYOUT.HANDOFF_ENTRY", false, `当前棒 ${current} 缺入口交接单 ${rel}（下一棒无从接手）`);
    return;
  }
  const lack = HANDOFF_SECTIONS.filter((s) => !text.includes(s));
  add(
    "LAYOUT.HANDOFF_ENTRY",
    lack.length === 0,
    lack.length
      ? `${rel} 缺节: ${lack.join("/")}`
      : `${rel} 在位且七节齐（当前棒 ${current}；待办 ${pending.length} 个）`,
  );
}

function checkStaleGaps(done) {
  const lines = (read("plan/06-交付对标与决策.md") ?? "").split("\n");
  const headerIdx = lines.findIndex((l) => l.startsWith("|") && l.includes("缺口"));
  if (headerIdx < 0) {
    add("LAYOUT.GAPS", false, "plan/06 找不到含『缺口』表头的表（判据表搬走/改名了？）");
    return;
  }
  const header = lines[headerIdx].split("|");
  const col = header.length - 2; // 最后一个有效列
  const stale = [];
  let scannedRows = 0;
  for (let i = headerIdx + 2; i < lines.length; i++) {
    if (!lines[i].startsWith("|")) break;
    if (/^\|\s*-{2,}/.test(lines[i])) continue; // 分隔行
    scannedRows += 1;
    const cells = lines[i].split("|");
    // 先剥行内代码与文件名（否则 `plan/RELEASE-M6.md` 会被判成"仍提 M6"）
    const cell = (cells[col] ?? "")
      .replace(/`[^`]*`/g, " ")
      .replace(/[\w./-]+\.(?:md|go|mjs|js|json|toml|txt|yml|yaml|mp4|mkv)/g, " ");
    for (const k of done) {
      const num = k.slice(1);
      if (new RegExp(`(^|[^0-9A-Za-z])M${num}([^0-9]|$)`).test(cell)) {
        stale.push(`plan/06 缺口列仍提 ${k}: ${cell.trim().slice(0, 80)}`);
      }
    }
  }
  add(
    "LAYOUT.GAPS",
    stale.length === 0,
    stale.length ? stale.join("；") : `plan/06 缺口列扫了 ${scannedRows} 行，无已 ✅ 里程碑`,
  );
}

function checkLedger(done) {
  const lines = (read("plan/00-README总览.md") ?? "").split("\n");
  const headerIdx = lines.findIndex((l) => l.startsWith("|") && l.includes("承诺") && l.includes("状态"));
  if (headerIdx < 0) {
    add("LAYOUT.LEDGER", false, "plan/00 找不到『承诺-证据台账』表头（承诺项 + 状态列）");
    return;
  }
  scanned.copies += 1;
  const bad = [];
  let rows = 0;
  for (let i = headerIdx + 2; i < lines.length; i++) {
    if (!lines[i].startsWith("|")) break;
    if (/^\|\s*-{2,}/.test(lines[i])) continue;
    rows += 1;
    const cells = lines[i].split("|").map((c) => c.trim());
    const status = cells[cells.length - 2] ?? "";
    const pending = /[⬜🟡]|未开始|未开工|待 M\d|待开工|待做/.test(status);
    if (!pending) continue;
    for (const k of done) {
      const num = k.slice(1);
      if (new RegExp(`(^|[^0-9A-Za-z])M${num}([^0-9]|$)`, "i").test(status.replace(/[`*]/g, ""))) {
        bad.push(`台账行仍挂未完成: ${status.slice(0, 60)}`);
      }
    }
  }
  add(
    "LAYOUT.LEDGER",
    bad.length === 0 && rows > 0,
    rows === 0 ? "台账表 0 行——判据空跑" : bad.length ? bad.join("；") : `台账 ${rows} 行，已成 ✅ 的里程碑无未完成标记`,
  );
}

const RELEASE_MILESTONE = null; // null → 自动取状态表里最大的里程碑号

function checkReleaseDoc(done, allMilestones) {
  const nums = allMilestones.map((k) => Number(k.slice(1)));
  const releaseMilestone = RELEASE_MILESTONE ?? Math.max(0, ...nums);
  const releaseKey = `M${releaseMilestone}`;
  if (releaseMilestone === 0 || !done.includes(releaseKey)) {
    add("LAYOUT.RELEASE", true, `发布里程碑 ${releaseKey} 未完成，收尾记录尚不适用（已扫 ${done.length} 个 ✅ 里程碑）`);
    return;
  }
  const release = read(`plan/RELEASE-${releaseKey}.md`);
  const wrapup = read(`plan/WRAPUP-${releaseKey}.md`);
  if (!release && !wrapup) {
    add("LAYOUT.RELEASE", false, `缺 plan/RELEASE-${releaseKey}.md 或 plan/WRAPUP-${releaseKey}.md`);
    return;
  }
  if (release) {
    const cols = /^##\s*一、tag 前事实/m.test(release) && /^##\s*二、tag 后追加/m.test(release);
    const heading = /^#{2,3}.*覆盖对账/m.test(release);
    const rows = (release.match(/^\|.*(实现|不适用|待首推).*\|$/gm) ?? []).length;
    const ok = cols && heading && rows >= 3;
    add("LAYOUT.RELEASE", ok, `两栏:${cols} 覆盖对账小节:${heading} 三态行数:${rows}`);
    return;
  }
  add("LAYOUT.RELEASE", true, "WRAPUP 形态");
}

/**
 * 文档点名的工具必须有实物。三态：在位 / PENDING（里程碑未到）/ 缺失 → FAIL。
 * 扫 plan/、AGENTS.md、README.md、docs/ 里的 `tools/<name>.<ext>` 与 `tools/<dir>/`。
 */
function checkDocumentedToolsExist(done) {
  const EXEMPT = new Set([]);
  /** 形如 { "tools/x.mjs": "M6" }：文档已点名、里程碑未完成时可见 SKIP；翻 ✅ 后必须删除。 */
  const PENDING_TOOLS = {};
  const docs = ["AGENTS.md", "README.md"];
  for (const dir of ["plan", "docs"]) {
    if (!existsSync(join(ROOT, dir))) continue;
    for (const f of readdirSync(join(ROOT, dir))) {
      if (f.endsWith(".md")) docs.push(`${dir}/${f}`);
    }
  }
  const referenced = new Set();
  for (const rel of docs) {
    const text = read(rel);
    if (text === null) continue;
    for (const m of text.matchAll(/tools\/[\w./-]+\.(?:mjs|js|cjs|py|sh|ps1|go)/g)) referenced.add(m[0]);
  }
  if (referenced.size === 0) {
    add("LAYOUT.DOC_TOOLS", false, `扫了 ${docs.length} 份文档，0 处 tools/ 引用——判据是空的（阳性对照失败）`);
    return;
  }
  scanned.docTools = referenced.size;
  const missing = [];
  const deferred = [];
  const stale = [];
  for (const rel of [...referenced].sort()) {
    if (EXEMPT.has(rel)) continue;
    // 目录形态（tools/fixture-gen/）→ 断言目录存在且有 .go 文件
    if (rel.endsWith("/")) {
      const abs = join(ROOT, rel);
      if (!existsSync(abs)) missing.push(rel);
      continue;
    }
    if (existsSync(join(ROOT, rel))) continue;
    const milestone = PENDING_TOOLS[rel];
    if (milestone === undefined) missing.push(rel);
    else if (done.includes(milestone)) stale.push(`${rel}（${milestone} 已 ✅，登记项必须清理）`);
    else deferred.push(`${rel}(${milestone})`);
  }
  const problems = [...missing, ...stale];
  let detail = `${referenced.size} 个文档点名的 tools/ 路径`;
  if (problems.length) detail += `；问题: ${problems.join(",")}`;
  else if (deferred.length) detail += `；${deferred.length} 个待里程碑落位: ${deferred.join(",")}`;
  else detail += " 全部在位";
  add("LAYOUT.DOC_TOOLS", problems.length === 0, detail);
}

/**
 * 本地状态与产物不得入库：`.mimosa/`、`bin/`、`fixtures/out/`、媒体样本。
 * 判据落 `.gitignore` 文本（不读 git 元数据），并断言 `bin/` 与 `.mimosa/` 有显式规则。
 */
function checkGitignore() {
  const text = read(".gitignore");
  if (text === null) {
    add("LAYOUT.GITIGNORE", false, ".gitignore 缺失");
    return;
  }
  const rules = [
    { id: ".mimosa/", re: /^\.mimosa\/?$/m },
    { id: "bin/", re: /^bin\/?$/m },
    { id: "*.mp4", re: /^\*\.mp4$/m },
    { id: "*.mkv", re: /^\*\.mkv$/m },
  ];
  const missing = rules.filter((r) => !r.re.test(text)).map((r) => r.id);
  add("LAYOUT.GITIGNORE", missing.length === 0, missing.length ? `缺规则: ${missing.join(",")}` : `${rules.length} 条排除规则在位`);
}

function main() {
  for (const a of process.argv.slice(2)) {
    if (a !== "--json") {
      process.stderr.write(`check-sprint-layout: unknown argument ${a}\n`);
      process.exit(2);
    }
  }
  for (const run of [checkRequiredFiles, checkBindingHead, checkSections, checkGitignore]) {
    try {
      run();
    } catch (error) {
      add(run.name, false, error.message);
    }
  }
  const { states, rowsByCopy } = collectStates();
  const done = [...states.entries()].filter(([, set]) => set.has("done")).map(([k]) => k);
  const allMilestones = [...states.keys()];
  for (const run of [
    () => checkDocumentedToolsExist(done),
    () => checkStatusAgreement({ states, rowsByCopy }),
    () => checkEntryHandoff(states, allMilestones),
    () => checkHandoffs(done, allMilestones),
    () => checkStaleGaps(done),
    () => checkLedger(done),
    () => checkReleaseDoc(done, allMilestones),
  ]) {
    try {
      run();
    } catch (error) {
      add(run.name, false, error.message);
    }
  }

  const failed = checks.filter((c) => !c.ok);
  if (JSON_OUT) {
    process.stdout.write(
      JSON.stringify(
        {
          checks: checks.map((c) => ({ id: c.id, ok: c.ok, detail: c.detail })),
          scanned,
          failed: failed.length,
        },
        null,
        2,
      ) + "\n",
    );
  } else {
    process.stdout.write("布局门（方法论装订） · 判据只落仓库内文件\n");
    process.stdout.write("=".repeat(64) + "\n");
    for (const c of checks) process.stdout.write(`${c.ok ? "PASS" : "FAIL"}  ${c.id.padEnd(20)} ${c.detail}\n`);
    process.stdout.write("=".repeat(64) + "\n");
    process.stdout.write(
      `这轮扫到：必备件 ${scanned.docs} / 交接单 ${scanned.handoffs} / 状态副本 ${scanned.copies} / 状态行 ${scanned.statusRows} / 文档点名工具 ${scanned.docTools}\n`,
    );
    process.stdout.write(`结论: ${failed.length === 0 ? "全绿" : `${failed.length} 项 FAIL`}\n`);
  }
  process.exitCode = failed.length === 0 ? 0 : 1;
}

main();
