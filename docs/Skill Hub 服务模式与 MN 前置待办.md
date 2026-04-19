# Skill Hub 服务模式与 MN 前置待办

**文档状态**：主业务收口清单
**适用范围**：按 `skill-hub-improvement-requirements.md` 收口主业务；服务安全与 `mn` 作为后续事项暂缓扩展

---

## 1. 主业务收口状态

当前先冻结新增能力，按需求文件完成主业务闭环：

- 已完成：批量导入与归档，覆盖 `import --fix-frontmatter --archive --force` 与 `feedback --all --force --json`
- 已完成：legacy frontmatter 修复，覆盖 `validate --fix` 与 `import --fix-frontmatter`
- 已完成：绝对路径审计与重写，覆盖 `lint --paths --fix --project-root`
- 已完成：重复技能检测与 canonical 同步，覆盖 `dedupe` 与 `sync-copies`
- 已完成：Markdown 本地链接校验，覆盖 `validate --links` 与 `validate --all --links`
- 已完成：审计报告，覆盖 `audit --output` 与 `audit --format json`
- 已完成：机器可读输出，覆盖 `status --json`、`validate --json`、`feedback --json`、`repo list --json`、`repo sync --json`、`pull --json`、`push --json`
- 已完成：显式登记命令，覆盖 `register <id>`，不创建、不覆盖已有技能内容
- 已完成：主业务 e2e 覆盖，包含批量导入、登记、去重同步、路径审计、链接校验、审计报告、批量反馈 JSON 与服务桥接兼容

当前结论：主业务功能进入收口态；后续只处理回归、文档一致性和缺陷修复，不继续增加新功能。

收口推进记录（2026-04-19）：

- 已完成：diff review，确认当前改动集中在主业务命令、serve 兼容层、文档与 e2e 覆盖，没有继续扩展 `mn` 或新的服务能力
- 已完成：文档一致性收口，修正 `search`、`pull/push`、`git status/sync/pull --json` 的服务桥接状态描述
- 已完成：完整验证，`go test ./...`、`go build -o bin/skill-hub ./application/skill-hub/cmd`、`pytest tests/e2e -q` 均通过，其中 e2e 为 `120 passed`
- 已完成：按主业务边界拆分提交，分为 Go 实现与单测、e2e 覆盖、文档收口三组

## 2. 服务模式收尾

1. 为 Web UI 补页面级自动化测试，至少覆盖首页加载、repo 操作、project 操作入口
2. 为服务模式补更稳定的错误码与错误消息约定，避免前后端和 CLI bridge 各自解释错误
3. 为本地服务增加更明确的安全边界，例如本地 token、session 或仅本机访问校验增强
4. 推进 `search` 的服务桥接，使 CLI 通过本地服务实例承接远端搜索交互
5. 收敛服务模式的配置项说明，例如 host、port、browser 行为和服务 URL 覆盖方式

当前进度（2026-04-18）：

- 已完成：`search` 服务桥接
- 已完成：项目技能生命周期命令 `register` / `import` / `dedupe` / `sync-copies` / `lint --paths` / `validate` / `audit` 的服务桥接
- 已完成：默认仓库 `pull --check/--json`、`push --dry-run/--json`、`git status/sync/pull --json` 的服务桥接
- 已完成：Web UI 管理端接入默认仓库 `sync-check/status/sync` API
- 已完成：默认仓库 `push-preview` API 与 `push confirm=true` 服务端保护
- 已完成：Web UI 默认仓库 push 预览与二次确认 UI，推送请求携带 `expected_changed_files`
- 已完成：HTTP API 包装错误按 `pkg/errors` 稳定错误码返回，并映射到 4xx/5xx 状态
- 已完成：默认 loopback 监听下增加 Host header loopback 校验，降低本地服务被非本机 Host 访问的风险
- 已完成：Web UI/API 增加基础安全响应头；默认 loopback 监听下拒绝非 loopback Origin/Referer 或 cross-site Fetch Metadata 的写请求
- 后续暂缓：更完整的服务安全边界，例如本地 token、session 或登录态

## 3. CLI / Service 边界整理

1. 明确哪些命令长期保留为本地命令，哪些应该逐步服务化
2. 收紧 CLI 中残留的直连底层逻辑，继续优先走 `runtime / repository / git`
3. 把服务 bridge 的错误回退策略写成文档约定，而不是只依赖当前代码实现

当前已明确的边界：

- `create` / `remove` 长期保留为项目本地工作区命令
- `validate` 仍校验项目本地工作区内容，但已接入服务桥接；CLI 会把项目路径解析为绝对路径后交给 `serve` 执行
- `prune` 长期保留为本地状态维护命令，直接维护 `state.json`
- `search` 已服务化；服务不可用时保留本地回退
- 默认仓库同步与推送类高级命令已服务化；`git status/sync/pull --json` 已复用相同桥接能力

## 4. MN 设计前置

1. 明确 `mn` 模块与本地管理服务的边界
2. 单独设计 `mn` 的节点管理、任务管理、连接管理 service contract
3. 设计 `mn` 所需的数据模型与状态机，不直接复用本地页面 API
4. 规划 `skill-node` 对接协议、认证和生命周期管理
5. 规划 `mn` 所需的审计、日志、指标与告警模型

## 5. 后续暂缓事项

1. 服务模式页面级测试与错误模型
2. 服务模式安全边界
3. `mn` service contract
4. `mn` 最小骨架与本地假节点集成测试
5. 真实 `skill-node` 对接
