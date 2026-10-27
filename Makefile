# Makefile — 10-27-mediatrace
#
# ⚠️ 本机（Windows）**没有 make**（实测 2026-10-27）。因此每个目标都只是对
#    `node tools/tasks.mjs <target>` 的**薄委托**——那才是跨平台可执行的等价物。
#    无 make 时直接跑：node tools/tasks.mjs all
#
# 本机 Go 也不在 PATH（见 AGENTS.md），tasks.mjs 会自动探测
# `%USERPROFILE%\go-sdk\go\bin\go.exe`；也可显式：GO=/path/to/go node tools/tasks.mjs test

SHELL := /bin/sh
NODE ?= node
TASKS := $(NODE) tools/tasks.mjs

.PHONY: all bin test vet gates layout desens verify-gates check-ignore-rules fixtures check-fixtures smoke cross clean help

all: ## 一键本地等价校验（gates + test + check-fixtures + smoke）
	@$(TASKS) all

bin: ## 构建单二进制到 bin/
	@$(TASKS) bin

test: ## go vet + go test
	@$(TASKS) test

vet: ## go vet
	@$(NODE) -e "require('node:child_process').spawnSync(process.env.GO||'go',['vet','./...'],{stdio:'inherit'})"

gates: layout desens ## 布局门 + 脱敏门

layout: ## 布局门（方法论装订）
	@$(NODE) tools/check-sprint-layout.mjs

desens: ## 脱敏门
	@$(NODE) tools/check-desensitized.mjs

verify-gates: ## 门自己的变异验证（每组变异必须变红）
	@$(TASKS) verify-gates

check-ignore-rules: ## gitignore 三态（台账入库 / 样本与产物不入库）
	@$(TASKS) check-ignore-rules

fixtures: ## 生成夹具 + manifest
	@$(TASKS) fixtures

check-fixtures: ## 双跑逐字节比对
	@$(TASKS) check-fixtures

smoke: ## 构建 + 三条退出码冒烟
	@$(TASKS) smoke

cross: ## 三平台交叉编译
	@$(TASKS) cross

clean: ## 清本地产物（不动入库文件）
	rm -rf bin dist fixtures/out .cache

help: ## 列出目标
	@grep -hE '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-16s %s\n", $$1, $$2}'
