#!/usr/bin/env node
// ═══ 跨平台任务运行器（Makefile 的等价实现） · 10-27-mediatrace ═══
//
// 为什么存在：本机（Windows）**没有 make**（实测 2026-10-27）。文档里写的 `make x`
// 在无 make 的机器上会直接 command not found —— 那会让"计划里点名的命令"变成空承诺。
// 本文件是那些目标的**唯一可执行等价物**，Makefile 只是薄薄一层委托。
//
// 用法：
//   node tools/tasks.mjs bin|test|gates|fixtures|check-fixtures|smoke|cross|all|help
//
// Go 定位：GO 环境变量 > PATH 中的 go > 本机已知路径（仅 Windows 兜底）。
// 退出码：0 成功 / 1 步骤失败 / 2 用法错误。

import { spawnSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

/** 读文件，失败返回空串（用于无 .git 树的降级断言）。 */
function readFileSyncSafe(p) {
  try {
    return readFileSync(p, "utf8");
  } catch {
    return "";
  }
}

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const VERSION = "0.1.0-dev";
const LDFLAGS = `-s -w -X main.version=${VERSION}`;
// Windows 的 CreateProcess 要求可执行文件带 `.exe` 后缀：
// `bin/mediatrace` 即使文件存在也会 ENOENT（本机实测 2026-10-27）。
const EXE = process.platform === "win32" ? ".exe" : "";
const BIN_REL = join("bin", `mediatrace${EXE}`);

function findGo() {
  if (process.env.GO) return process.env.GO;
  const probe = spawnSync("go", ["version"], { encoding: "utf8" });
  if (probe.status === 0) return "go";
  // 仅 Windows 兜底：本机实测路径（不含用户名，用 os.homedir 拼）
  if (process.platform === "win32" && process.env.USERPROFILE) {
    const p = join(process.env.USERPROFILE, "go-sdk", "go", "bin", "go.exe");
    if (existsSync(p)) return p;
  }
  return null;
}

const GO = findGo();

function run(cmd, args, opts = {}) {
  process.stdout.write(`$ ${[cmd, ...args].join(" ")}\n`);
  const r = spawnSync(cmd, args, { cwd: ROOT, stdio: "inherit", ...opts });
  if (r.error) throw r.error;
  return r.status ?? 1;
}

function needGo() {
  if (!GO) {
    process.stderr.write(
      "ERROR: Go 不存在（GO 未设、PATH 无 go、Windows 兜底路径也不在）。\n" +
        "  用法：GO=/path/to/go node tools/tasks.mjs <target>\n",
    );
    process.exit(1);
  }
}

const tasks = {
  bin() {
    needGo();
    mkdirSync(join(ROOT, "bin"), { recursive: true });
    if (run(GO, ["build", "-trimpath", "-ldflags", LDFLAGS, "-o", BIN_REL, "./cmd/mediatrace"]) !== 0) return 1;
    const built = join(ROOT, BIN_REL);
    if (!existsSync(built)) {
      process.stderr.write(`FAIL: built binary not found at ${BIN_REL}\n`);
      return 1;
    }
    return run(built, ["--version"]);
  },

  test() {
    needGo();
    if (run(GO, ["vet", "./..."]) !== 0) return 1;
    return run(GO, ["test", "./...", "-count=1"]);
  },

  gates() {
    // 两个门都由 node 跑；不做 shell 串联，避免 Windows 无 `&&`
    if (run(process.execPath, [join("tools", "check-sprint-layout.mjs")]) !== 0) return 1;
    return run(process.execPath, [join("tools", "check-desensitized.mjs")]);
  },

  /**
   * gitignore 三态核对（常驻断言，不靠人记）。
   * 实测踩过：`out/`（任意层级）把 `fixtures/out/manifest.json` 一起忽略，
   * 而后面的 `!fixtures/out/manifest.json` **无法复活**已被忽略目录内的文件。
   * 判据：
   *   · manifest 必须**不被忽略**（`git check-ignore` 退出码 1）
   *   · 样本 mp4 必须**被忽略**（退出码 0）
   *   · bin/ 必须**被忽略**（退出码 0）
   * 不在 git 树里（发布包解包后无 .git）时，退化为"断言 .gitignore 规则文本存在"并显式 SKIP。
   */
  "check-ignore-rules"() {
    const hasGit = existsSync(join(ROOT, ".git"));
    const cases = [
      { path: "fixtures/out/manifest.json", ignored: false, why: "真值台账必须入库" },
      { path: "fixtures/out/c1_dual_moov.mp4", ignored: true, why: "样本字节不入库" },
      { path: "bin/mediatrace", ignored: true, why: "构建产物不入库" },
      { path: ".mimosa/hook-state/x.json", ignored: true, why: "本地会话状态不入库" },
    ];
    if (!hasGit) {
      const gi = readFileSyncSafe(join(ROOT, ".gitignore"));
      const missing = [".mimosa/", "bin/", "*.mp4", "!fixtures/out/manifest.json"].filter((r) => !gi.includes(r));
      if (missing.length) {
        process.stderr.write(`FAIL(no-git fallback): .gitignore 缺规则 ${missing.join(", ")}\n`);
        return 1;
      }
      process.stdout.write(`SKIP: 无 .git 树（发布包形态）——已退化为 .gitignore 文本断言，${cases.length} 条线上判据未执行\n`);
      return 0;
    }
    for (const c of cases) {
      const r = spawnSync("git", ["check-ignore", "-q", c.path], { cwd: ROOT });
      const ignored = r.status === 0;
      if (ignored !== c.ignored) {
        process.stderr.write(
          `FAIL: ${c.path} ignored=${ignored} want ${c.ignored}（${c.why}）\n`,
        );
        return 1;
      }
      process.stdout.write(`ok: ${c.path} ignored=${ignored}\n`);
    }
    return 0;
  },

  "verify-gates"() {
    return run(process.execPath, [join("tools", "verify-layout-gate.mjs")]);
  },

  fixtures() {
    needGo();
    return run(GO, [
      "run", "./tools/fixture-gen",
      "-out", "fixtures/out", "-seed", "20261027", "-suite", "all",
    ]);
  },

  "check-fixtures"() {
    needGo();
    const manifest = join(ROOT, "fixtures", "out", "manifest.json");
    if (!existsSync(manifest)) {
      process.stderr.write("fixtures/out/manifest.json 缺失：先跑 `node tools/tasks.mjs fixtures`\n");
      return 1;
    }
    return run(GO, [
      "run", "./tools/fixture-gen",
      "-out", "fixtures/out", "-seed", "20261027", "-suite", "all", "-check",
    ]);
  },

  smoke() {
    // 构建 + 三条退出码断言（进程级）
    if (tasks.bin() !== 0) return 1;
    const bin = join(ROOT, "bin", process.platform === "win32" ? "mediatrace.exe" : "mediatrace");
    const cases = [
      { args: ["frobnicate"], want: 3, label: "unknown command" },
      { args: ["audit", join(ROOT, "definitely-missing.mp4")], want: 3, label: "missing file" },
    ];
    for (const c of cases) {
      const r = spawnSync(bin, c.args, { cwd: ROOT, encoding: "utf8" });
      if (r.status !== c.want) {
        process.stderr.write(`FAIL: ${c.label} exit=${r.status} want ${c.want}\n`);
        return 1;
      }
      process.stdout.write(`ok: ${c.label} -> ${r.status}\n`);
    }
    return 0;
  },

  cross() {
    needGo();
    mkdirSync(join(ROOT, "dist"), { recursive: true });
    const targets = [
      ["linux", "amd64", "dist/mediatrace-linux-amd64"],
      ["darwin", "arm64", "dist/mediatrace-darwin-arm64"],
      ["windows", "amd64", "dist/mediatrace-windows-amd64.exe"],
    ];
    for (const [goos, goarch, out] of targets) {
      const status = run(GO, ["build", "-trimpath", "-ldflags", LDFLAGS, "-o", out, "./cmd/mediatrace"], {
        env: { ...process.env, GOOS: goos, GOARCH: goarch },
      });
      if (status !== 0) return status;
    }
    return 0;
  },

  all() {
    for (const t of ["gates", "verify-gates", "check-ignore-rules", "test", "check-fixtures", "smoke"]) {
      const status = tasks[t]();
      if (status !== 0) {
        process.stderr.write(`\ntarget ${t} failed\n`);
        return status;
      }
    }
    return 0;
  },

  help() {
    process.stdout.write(
      "targets:\n" +
        Object.keys(tasks)
          .filter((k) => k !== "help")
          .map((k) => `  ${k}`)
          .join("\n") +
        "\n",
    );
    return 0;
  },
};

const target = process.argv[2] ?? "help";
if (!Object.hasOwn(tasks, target)) {
  process.stderr.write(`tasks: unknown target ${target}\n`);
  tasks.help();
  process.exit(2);
}
process.exit(tasks[target]());
