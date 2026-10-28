#!/usr/bin/env node
// ═══ 脱敏门（发布前必跑，进 CI 的 layout job） · 10-27-mediatrace ═══
// 来源：ai-tool-project-sprint references/templates/check-desensitized.mjs（2026-10-27 拷入并按本项目改写）。
// 为什么是门而不是一次性 grep：发布后任何一次改动都可能把本机路径带回来。
// 退出码：0 零命中 / 1 有命中 / 2 用法错误。
//
// ── 变异验证（每组必须变红；见 tools/verify-layout-gate.mjs）────────────────
//   V1 写 `D:/ProgramData/<x>`                → absolute-session-path
//   V2 写 `C:\Users\<真实用户名>\x`            → host-username
//   V3 写 `/c/Users/<真实用户名>/x`            → msys-user-path
//   V4 写 `127.0.0.1:7897`                    → loopback-proxy
//   还原后必须复绿。
//
// ── 三条实测口径 ────────────────────────────────────────────────────────
//   ① 占位符必须放行：`C:\Users\<user>\…` / `<repo>` 是脱敏后的正确写法（负向前瞻 `(?!<)`）。
//   ② 公开镜像不是私有代理：只认回环/私网地址。
//   ③ 门要豁免自己：本文件必须写出被判据禁止的字面量（规则本身），按**文件路径**显式列。

import { readdirSync, readFileSync } from "node:fs";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const SELF = new Set(["tools/check-desensitized.mjs", "tools/check_desensitized.py"]);

// "update" = 迭代交接物（诊断/路线/执行/发布记录），内容是**本机实测命令原文**，
// 按定义包含盘符路径与回环地址；它与 .gitignore 的 `update/` 成对出现（缺一即不一致）。
const SKIP_DIRS = new Set([".git", ".tmp", "dist", "node_modules", ".cache", ".venv", "build", "bin", "fixtures", ".mimosa", "update"]);
const SKIP_SUFFIXES = [".egg-info", ".dist-info"];
const SKIP_EXTS = new Set([".png", ".jpg", ".jpeg", ".gif", ".ico", ".zip", ".gz", ".exe", ".dll", ".pdf", ".woff", ".woff2"]);

const RULES = [
  {
    id: "absolute-session-path",
    re: /\b[A-Za-z]:[\\/](?:ProgramData|Users|zcode|repos?|projects?|program)[\\/](?!<)/,
  },
  {
    id: "msys-user-path",
    re: /\/[a-z]\/Users\/(?!<)[A-Za-z0-9._-]+\//,
  },
  {
    id: "host-username",
    // 本机用户名（本项目实测值；换机器要改这一行，别用环境实算——runner 上叫 runner）
    re: /\bASUS\b/,
  },
  {
    id: "loopback-proxy",
    re: /\b127\.0\.0\.1:\d{2,5}\b/,
  },
];

for (const a of process.argv.slice(2)) {
  process.stderr.write(`check-desensitized: unknown argument ${a}\n`);
  process.exit(2);
}

function* walk(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === ".git") continue; // 目录形态与 worktree 文件形态一起挡
    if (entry.isDirectory()) {
      if (SKIP_DIRS.has(entry.name) || SKIP_SUFFIXES.some((s) => entry.name.endsWith(s))) continue;
      yield* walk(join(dir, entry.name));
      continue;
    }
    if (!entry.isFile()) continue;
    const dot = entry.name.lastIndexOf(".");
    if (dot >= 0 && SKIP_EXTS.has(entry.name.slice(dot))) continue;
    yield join(dir, entry.name);
  }
}

const hits = [];
let scanned = 0;
for (const file of walk(ROOT)) {
  const rel = relative(ROOT, file).split(sep).join("/");
  if (SELF.has(rel)) continue;
  let text;
  try {
    text = readFileSync(file, "utf8");
  } catch {
    continue;
  }
  if (text.includes("\u0000")) continue; // 二进制
  scanned++;
  const lines = text.split("\n");
  for (let i = 0; i < lines.length; i++) {
    for (const rule of RULES) {
      if (rule.re.test(lines[i])) {
        hits.push(`${rel}:${i + 1}: [${rule.id}] ${lines[i].trim().slice(0, 160)}`);
      }
    }
  }
}

if (hits.length > 0) {
  process.stdout.write(`脱敏门：${hits.length} 处命中（扫了 ${scanned} 个文本文件，豁免 ${SELF.size} 份）\n`);
  for (const h of hits) process.stdout.write(`FAIL  ${h}\n`);
  process.exit(1);
}
process.stdout.write(`脱敏门：扫了 ${scanned} 个文本文件，0 命中（豁免 ${SELF.size} 份）\n`);
