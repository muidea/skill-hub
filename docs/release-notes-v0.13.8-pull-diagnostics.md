# v0.13.8 pull 受阻诊断

## 行为改进

- `pull` 被未提交改动阻止时，展示文件路径、暂存区/工作区状态及本地差异；与远端涉及同一路径时补充远端差异，明确标为潜在冲突。
- 已有合并冲突展示冲突类型与冲突差异；提交分叉展示双方领先计数及独有提交摘要。二进制文件、预览截断和检查失败均给出说明。
- 业务失败仍返回非零退出码，不再附带 Cobra Usage；参数错误保留用法提示。
- `pull --json` 继续保持完整 stdout JSON，错误详情位于 `command_error.details.pull`。现有 HTTP 错误响应透传诊断，本地执行与服务桥接一致。
- `pull --force` 输出保留的 stash 哈希，JSON 摘要增加可选 `stash_ref`，明确说明未自动恢复改动，包括后续拉取失败的情况。

## 实现边界

- 保留现有未提交改动拦截策略与 `--ff-only` 拉取策略，不自动解决冲突、merge 或 rebase。远端没有更新时保留本地改动且不创建 stash。
- `--force` 无法解决已有冲突或提交分叉；服务桥接仍不支持 `pull --force`。不新增服务端接口。
- 最多展示 100 个文件，每段差异最多 4 KiB，文件差异预览总预算 64 KiB；省略和截断显式标明。
- CLI 通过 focused port 获取结果，Git 层收集诊断；只读差异检查禁用外部 diff/textconv helpers。

## 验证范围

- Go 测试覆盖真实 Git 仓库中的暂存/未暂存改动、未跟踪特殊文件名、重命名、文本/二进制与修改/删除冲突、提交分叉、差异截断、保留 stash 和传输错误。
- CLI 进程、HTTP 与客户端适配器测试覆盖错误详情透传、完整 JSON、非交互调用及业务失败不附带 Usage。
- 新增 5 项离线 pull e2e，纳入默认 runner 与 CI 必过的 JSON/pull 回归选择。
- 同步 README、命令规范及 e2e 文档。
- `make test`、`make lint`、`make build` 通过；JSON/pull 回归选择共 15 项 e2e 通过。
