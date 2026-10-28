# v0.13.6 CLI JSON 调用合同

## 问题修复

- 修复首次项目登记时工作区提示、目录创建进度混入 stdout，导致 JSON 无法整体解析的问题。
- 将 `--json` 收口为 CLI 全局选项，补齐登记、导入、创建、列表、应用、移除及仓库管理等命令的结构化输出。
- JSON 模式禁止等待 stdin，保留各命令原有确认与冲突处理规则；项目移除和仓库移除、禁用增加显式 `--yes` 确认。
- 参数错误、业务错误与部分失败通过单份 JSON 返回；失败对象使用 `command_error`，原有摘要与逐项结果保留，退出码非零。
- 修复空匹配时部分 JSON 分支没有输出的问题。只读查询和预演不由 CLI 隐式建立项目工作区。

## 工具链

- Go 最低版本升级到 1.26，推荐工具链为 Go 1.26.7；同步更新 CI、发布构建、Docker 构建镜像和开发安装文档。
- CI 固定 staticcheck v0.8.2，并使用当前 Go 工具链编译，避免旧版检查器与 Go 1.26 不兼容。

## 兼容性

- 已有 JSON 成功字段不增加统一外层；调用方应分别捕获 stdout 和 stderr。
- 普通文本模式保留既有交互行为。JSON 模式不读取 stdin，不将空输入或 EOF 当作操作确认。
- 仅修改 CLI 客户端；未增加 agent 模式，不修改 daemon、HTTP API 或 Web UI。

## 验证

- `go test ./... --count=1`
- `make build`
- `make lint`
- 首次登记、重复登记、开放 stdin 不阻塞、结构化错误、部分失败、空匹配、审计文件与服务桥接的进程级测试。
- 相关 e2e：`test_cli_json.py`、`test_bulk_import_register_status.py`、`test_feedback_all_json.py`，共 8 项通过。
- JSON 回归选择已接入默认 e2e runner，并作为 CI 必须通过的独立步骤；同步补齐 e2e README、用户入口和命令参数说明。
