# v0.13.7 空项目登记与原请求回执

## 新增能力

- 新增 CLI `project register [path] --request-id <id> --json`，非交互登记空项目，不要求已有技能或占位文件，保留已有项目状态。
- 同一 Hub home 内，相同 ID 和绝对路径返回原始持久化回执，不重新执行；不同路径复用 ID 返回 `REQUEST_CONFLICT`。
- 同一入口增加 `--query`，按原请求 ID 查询执行状态、是否结束及原操作效果；不查询当前项目状态，不重跑登记。
- 回复丢失或 stdout 写入失败后仍可查询原结果；失败回执也可查询和重放。异常中断且缺少最终回执时返回 `unknown`，不自动重试。

## 实现边界

- 只增加 CLI 客户端入口，沿用项目状态 Block、类型化 EventHub 合同和 focused port，无新增 framework 运行单元或 daemon/HTTP/Web UI 接口。
- 回执保存在 `SKILL_HUB_HOME/project-registrations`，独立于当前项目状态。去重依赖同一 Hub home 及保留原请求回执；无法补查历史版本未记录的请求。
- 增加 Windows 项目状态文件锁，保护不同请求 ID 的并发登记；登记请求锁在 Unix/Windows 均为跨进程锁。

## 验证范围

- 状态层覆盖原回执重放、状态变化后查询、请求 ID 冲突、失败重放、进行中与异常中断、并发登记和查询无副作用。
- CLI 进程测试使用开放 stdin，覆盖空项目、跨目录查询、丢失 stdout 后查询与不重复登记。
- e2e 补齐登记/查询/冲突/失败回执场景，复用默认 runner 和 CI 必过的 JSON 回归选择。
- 同步 README、CLI 命令规范及 e2e 文档。
- `make test`、`make lint`、`make build` 均通过；登记及回执相关竞态检查通过，Windows 状态包交叉编译通过。
- JSON 回归选择共 10 项 e2e 通过，包含空项目登记、原回执查询、失败重放和请求 ID 冲突。
