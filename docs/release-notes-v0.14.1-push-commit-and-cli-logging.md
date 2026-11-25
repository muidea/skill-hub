# v0.14.1 系统 Git 提交修复与全命令详细日志收口

## 问题修复

- 将仓库提交路径 `Repository.Commit` 从 `go-git` 迁移至系统 `git`（`CommitWithSystemGit`），完整支持 Git `include` / `includeIf` 配置、`GIT_AUTHOR_*` / `GIT_COMMITTER_*` 环境变量以及删除与忽略规则。
- 在 `git add -A` 之前通过 `git var GIT_AUTHOR_IDENT` 与 `git var GIT_COMMITTER_IDENT`（固定 `LC_ALL=C`）预检提交身份；缺失 `user.name` / `user.email` 时返回 `GIT_OPERATION_FAILED` 结构化错误与中文化 `git config` 指引，不再泄露 `author field is required`，也不会残留已暂存文件。
- 统一在 CLI 命令分发包装器 `markCommandExecution` 中为全部命令的运行时执行开启 `SilenceUsage`，业务报错不再刷屏输出 `Usage:`，同时保留参数与 flag 校验阶段的用法提示。

## 行为改进

- 新增命令级详细日志采集器 `RunLog`（`pkg/logging/runlog.go`）：自动对 URL 凭据、`Authorization` 头及常见 token/secret 变量脱敏，支持 2 MiB 内存上限截断，并在失败且存在详细日志或显式指定 `--log-file` 时以 `0600` 权限落盘。
- CLI 全局新增 `--log-file <path>` 与 `--debug`（也支持 `SKILL_HUB_VERBOSE=1`），其中 `--debug` 可在 `stderr` 实时输出脱敏后的详细运行日志，不与既有局部 `--verbose`/`-v` 旗标冲突。
- 将 `internal/git` 下 `fetch` / `pull` / `push` / `merge` / `commit` / `stash` / `clone` 等原始子进程输出与过程提示统一收口到详细日志，移除全局 `os.Stdout` 劫持辅助函数 `runSilencingStdout`，控制台仅保留面向用户的简洁摘要、交互确认与可操作错误。
- 非 `--json` 模式下若落盘了日志文件，会在 `stderr` 追加 `详细日志已保存至: <path>`；`--json` 模式下失败时保持 `stdout` 为单个合法 JSON，并通过 `command_error.details.log_file` 透出日志路径。

## 实现边界

- 简单参数校验或未初始化配置等未产生详细过程日志的失败不会创建空白临时日志文件；`__complete`、`__completeNoDesc` 与 `completion` 跳过详细日志捕获。
- `skill-hubd` 服务端日志保持原有机制不变，本次仅收口 CLI 与共享 Git/日志基础能力。

## 验证范围

- 单元测试覆盖：`RunLog` 脱敏/截断/`0600` 落盘、`BindGlobalLogger`、`Repository.Commit`（`include.path`、环境变量提交身份、缺失身份不暂存）、全量命令运行时 `SilenceUsage`、`push` 失败时控制台清洁与临时日志透出。
- 端到端回归覆盖：`tests/e2e/test_cli_json.py`、`tests/e2e/test_pull_diagnostics.py`、`tests/e2e/test_feedback_all_json.py` 共 12 项离线用例全部通过。
- 工程门禁：`make lint`（`gofmt` / `go vet` / `staticcheck`）、`make test`（`go test ./... --count 1`）与 `make build` 全部通过。
