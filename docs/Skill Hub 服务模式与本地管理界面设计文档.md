# Skill Hub 服务模式与本地管理界面设计文档

**文档状态**：第一阶段已实现，文档按当前代码状态刷新  
**适用范围**：`skill-hub` 在保留现有命令行模式基础上，新增本地服务运行模式、Web 管理界面，以及 CLI 通过服务交互的能力  
**当前阶段目标**：先完成服务模式本身，不先做 `skill-node` 对接  
**当前验证状态**：`go test ./...` 通过；`tests/e2e` 当前总结果为 `102 passed, 3 skipped`

---

## 1. 目标

当前 `skill-hub` 已从单一 CLI 工具扩展为双运行模式：

- 命令行模式
- 服务模式

本阶段已经完成：

1. `skill-hub` 支持以服务方式运行
2. 提供本地 Web 化管理界面
3. 当前 CLI 在服务存在时，可以通过服务进行功能交互

本阶段仍未做：

- `skill-node` 注册与调度
- 远程多用户访问
- 复杂认证系统
- 将所有 CLI 命令一次性 API 化

---

## 2. 总体方案

当前 `skill-hub` 在保留现有 CLI 能力的同时，新增：

- `skill-hub serve`

服务模式运行时提供两类访问入口：

- 本地 HTTP API
- 本地 Web UI

CLI 与服务的关系：

- 默认仍然可以本地直接执行
- 当检测到本地 service 可用时，优先通过 service API 交互
- 当 service 不可用时，回退到当前本地执行逻辑

这样保证：

- 不破坏现有 CLI 使用习惯
- 服务模式可以逐步扩展
- 后续对接 `skill-node` / `MN` 时不需要重新设计本地管理面
- `serve` 只是增强能力，不是 CLI 的前置依赖

补充约束：

- `serve` 托管的是用户本地 `~/.skill-hub/` 全局管理目录
- 项目实际使用的 skill 内容仍位于项目根目录 `.agents/skills/`
- `create` / `remove` 只针对项目本地工作区，不要求服务化
- `validate` 只校验项目本地工作区内容，但可通过服务桥接复用 `serve` 托管的状态与路径上下文
- `search` 属于远端能力入口，在本地服务可用时优先由服务实例统一承接远端交互；服务不可用时回退到本地执行
- 默认 loopback 监听会校验 Host header 也必须为 loopback，避免本地服务在浏览器场景下被非本机 Host 间接访问；显式非 loopback 监听保持远程访问兼容

---

## 3. 运行模式设计

当前服务入口：

```bash
skill-hub serve
skill-hub serve --secret-key write-secret
```

当前支持参数：

- `--host 127.0.0.1`
- `--port 5525`
- `--secret-key <value>`
- `--open-browser`

当前未做：

- `--daemon`
- systemd/service install

服务启动后职责：

- 加载现有 `runtime`
- 暴露本地 HTTP API
- 托管 Web UI 静态资源
- 提供健康检查接口
- 为 CLI 提供本地进程间交互入口

---

## 4. 当前已实现功能范围

### 4.1 仓库管理

支持：

- 查看仓库列表
- 添加仓库
- 删除仓库
- 启用仓库
- 禁用仓库
- 设置默认仓库
- 同步仓库

### 4.2 技能查看

支持：

- 查看技能列表
- 按 repo 过滤
- 按 target 过滤
- 查看技能基础元数据

这部分对应当前 `list` 命令。

### 4.3 本地工作区技能查看与操作

支持：

- 查看当前机器已登记的项目列表
- 查看某个项目下启用的 skill
- 查看 skill 的版本、状态、目标环境
- 查看项目状态摘要
- 设置项目首选目标
- 对项目执行 `use`
- 对项目执行 `apply`
- 对项目执行 `feedback`

项目视图与操作都以 `state.json` 和本地仓库内容为准。

---

## 5. 当前模块设计

当前已新增以下模块。

### 5.1 `internal/modules/kernel/server`

职责：

- 服务启动编排
- 生命周期管理
- 注入 runtime
- 管理 HTTP server

### 5.2 `internal/modules/blocks/httpapi`

职责：

- REST API handler
- 请求/响应结构
- 路由注册
- 健康检查

### 5.3 `internal/modules/blocks/webui`

职责：

- 管理前端静态资源
- 提供页面路由
- 托管本地 Web UI

### 5.4 `internal/modules/blocks/hubclient`

职责：

- CLI 到本地服务的桥接 client
- 统一访问本地 HTTP API

### 5.5 `internal/modules/kernel/project_inventory`

职责：

- 从 `state.json` 聚合“当前机器有哪些项目”
- 聚合每个项目启用了哪些 skill
- 给 Web UI 和 HTTP API 提供统一项目视图

### 5.6 `internal/modules/kernel/project_status`

职责：

- 聚合项目技能状态
- 为 `status` 命令的服务桥接和 Web UI 状态视图提供统一输出

### 5.7 `internal/modules/kernel/project_use`

职责：

- 处理项目启用技能
- 负责服务模式下的仓库候选技能解析、变量提交和状态写入

### 5.8 `internal/modules/kernel/project_apply`

职责：

- 处理项目技能分发
- 负责服务模式下的 dry-run / force 行为和适配器调用

### 5.9 `internal/modules/kernel/project_feedback`

职责：

- 处理反馈预览与执行
- 负责版本推进、归档和索引刷新

---

## 6. 与现有模块的关系

服务模式优先复用当前已经收口的模块服务：

- `internal/modules/kernel/runtime`
- `internal/modules/kernel/repository`
- `internal/modules/kernel/project_state`
- `internal/modules/blocks/git`

服务端 API handler 只调用这些 service，不直接依赖 CLI 命令层。  
CLI bridge 也只调用 HTTP API，不重复实现业务逻辑。

---

## 7. 当前 HTTP API

统一前缀：

```text
/api/v1
```

### 7.1 健康检查

- `GET /api/v1/health`

### 7.2 仓库管理

- `GET /api/v1/repos`
- `POST /api/v1/repos`
- `POST /api/v1/repos/{name}/sync`
- `POST /api/v1/repos/{name}/enable`
- `POST /api/v1/repos/{name}/disable`
- `POST /api/v1/repos/{name}/set-default`
- `DELETE /api/v1/repos/{name}`

### 7.3 技能列表与详情

- `GET /api/v1/skills`
- `GET /api/v1/search?keyword=<keyword>[&target=<value>][&limit=<n>]`
- `GET /api/v1/skills/{id}/candidates`
- `GET /api/v1/skills/{id}?repo=<repo-name>`

支持查询参数：

- `repo`
- `target`

### 7.4 项目与工作区技能

- `GET /api/v1/projects`
- `GET /api/v1/projects/{id}`
- `GET /api/v1/projects/{id}/skills`
- `GET /api/v1/project-status?path=<project-path>[&skill_id=<id>]`

### 7.5 项目写操作

- `POST /api/v1/project-skills/use`
- `POST /api/v1/project-apply`
- `POST /api/v1/project-feedback/preview`
- `POST /api/v1/project-feedback/apply`

---

## 8. API 响应规范

统一返回结构：

```json
{
  "code": "OK",
  "message": "",
  "data": {}
}
```

错误结构：

```json
{
  "code": "REPO_NOT_FOUND",
  "message": "仓库不存在"
}
```

要求：

- `code` 可机器识别
- `message` 可直接展示
- `data` 仅在成功时返回业务内容
- 业务层抛出的 `pkg/errors.AppError` 会保留原始错误码，例如 `SKILL_NOT_FOUND`、`PROJECT_NOT_FOUND`、`VALIDATION_FAILED`、`INVALID_INPUT`
- HTTP 状态按错误类别映射：未找到类为 `404`，权限类为 `403`，网络或远端 Git 类为 `502`，未实现为 `501`，系统错误为 `500`，其余输入或校验类错误为 `400`

---

## 9. 当前 Web UI 设计

当前仍然以 3 个页面区域为主。

### 9.1 仓库页

展示：

- 仓库列表
- 默认仓库标记
- 启用/禁用状态
- 同步按钮
- 添加仓库表单
- 设置默认仓库按钮

### 9.2 技能页

展示：

- 当前 `list` 结果
- repo 过滤
- target 过滤
- `name/version/repository/compatibility/description`

### 9.3 项目页

展示：

- 当前机器已登记项目
- 项目下启用的 skill
- version/status/target
- 项目 `status`
- 项目 `use`
- 项目 `apply`
- 项目 `feedback`

### 9.4 前端技术约束

当前未单独起前端工程，直接由 Go `embed` 托管轻量 HTML/CSS/JS。

---

## 10. CLI 与服务交互设计

当前本地客户端层：

- `internal/modules/blocks/hubclient`

职责：

- 探测本地 service 是否存在
- 访问本地 HTTP API
- 将响应转换为 CLI 所需结果

### 10.1 当前已优先通过服务代理的命令

- `repo list`
- `repo add`
- `repo remove`
- `repo sync`
- `repo enable`
- `repo disable`
- `repo default`
- `list`
- `search`
- `status`
- `use`
- `apply`
- `feedback`
- `pull`
- `push`
- `register`
- `import`
- `dedupe`
- `sync-copies`
- `lint --paths`
- `validate`
- `audit`

### 10.2 当前继续本地执行的命令

- `init`
- `create`
- `remove`

补充说明：

- `search` 当前已接入服务桥接；当本地服务不可用时，CLI 仍保留兼容性本地回退
- `create` / `remove` 仍明确限定为项目本地工作区命令，不参与服务化托管
- `validate` 已接入服务桥接，但仍只校验调用方传入的项目工作区路径
- `pull` / `push` 已接入默认仓库同步、status/push 服务桥接，支持客户端 HOME 无本地配置时通过 `serve` 处理默认仓库同步闭环

### 10.3 CLI 代理策略

当前策略：

1. 默认先尝试本地 service
2. 若本地 service 可达，则通过 API 执行
3. 若 service 不可达，则回退到本地逻辑

支持通过环境变量关闭：

```text
SKILL_HUB_DISABLE_SERVICE_BRIDGE=1
```

---

## 11. 本地通信方式

当前直接使用：

- `127.0.0.1:5525` HTTP

原因：

- 实现简单
- 浏览器可直接访问
- CLI 可直接复用
- 集成测试容易搭建

当前未做：

- Unix Domain Socket
- WebSocket
- 远程监听

---

## 12. 安全约束

当前服务模式的安全约束：

- 默认仅监听 `127.0.0.1`
- 默认不监听 `0.0.0.0`
- 默认 loopback 监听下校验 Host header 必须为 loopback
- 默认 loopback 监听下，修改类 HTTP 方法会拒绝非 loopback `Origin` / `Referer`，并拒绝 `Sec-Fetch-Site: cross-site`
- 显式绑定到非 loopback 地址时保留远程访问兼容性，不强制套用本地浏览器来源校验
- 响应统一增加基础安全响应头：`Content-Security-Policy`、`X-Frame-Options`、`X-Content-Type-Options`、`Referrer-Policy`
- 修改类 API 使用 `secretKey` 控制写权限：未配置 `--secret-key` 时服务按只读模式运行，读取类 `GET` 接口和 Web UI 可访问，`POST` / `DELETE` 等写操作返回 `READ_ONLY`
- 配置 `--secret-key` 后，修改类 API 必须携带 `X-Skill-Hub-Secret-Key`；Web UI 管理端在写操作需要时提示输入并暂存于浏览器会话，CLI bridge 通过 `SKILL_HUB_SERVICE_SECRET_KEY` 传递
- `serve status` 只显示 `write=read-only` 或 `write=secret-key`，不输出密钥明文
- 不在 Web 页面显示 `git_token`
- 修改类接口仅面向本地访问场景设计

当前未做：

- 多用户登录认证
- RBAC
- 远程会话管理
- token / session 级别的本地登录态

---

## 13. 项目工作区技能聚合设计

Web 展示“当前机器本地工作区里管理的 skill”的真相源为 `state.json`。

项目列表示例：

```json
[
  {
    "project_id": "hash-or-path-key",
    "project_path": "/path/to/project",
    "preferred_target": "open_code",
    "skill_count": 3
  }
]
```

项目技能示例：

```json
[
  {
    "skill_id": "xxx",
    "version": "1.0.0",
    "status": "Synced",
    "target": "open_code"
  }
]
```

---

## 14. 当前实现状态

当前服务模式已经形成完整闭环：

- `skill-hub serve` 可启动本地 HTTP 服务与 Web UI
- `skill-hub serve register/start/stop/status/remove` 可管理本地命名服务实例
- `skill-hub serve --secret-key` 可开启写操作密钥；未配置时服务只读
- Web UI 支持仓库管理、技能查看、项目查看与项目操作
- CLI 在服务可用时会优先通过服务桥接执行 `repo/list/status/use/apply/feedback/pull/push` 以及项目技能生命周期命令
- 服务不可用时，上述命令仍会回退到本地逻辑
- `prune` 为本地状态维护命令，当前不通过服务桥接，直接维护 `state.json`

当前服务实例管理补充：

- 注册信息落盘到 `~/.skill-hub/services.json`
- `start` 后台拉起 `skill-hub serve --host ... --port ...`
- `status` 支持查看全部或单个实例状态
- 后台日志默认写入 `~/.skill-hub/services/logs/<name>.log`

相关实现入口：

- 服务启动：`internal/cli/serve.go`
- HTTP API：`internal/modules/blocks/httpapi/service`
- Web UI：`internal/modules/blocks/webui/service/assets/index.html`
- CLI bridge：`internal/cli/service_bridge.go`

---

## 15. 当前测试覆盖

当前服务模式已具备以下自动化验证：

- service 层单测
- HTTP API handler 单测
- hubclient 单测
- CLI bridge 单测
- Python e2e

其中 `tests/e2e/test_service_mode.py` 已覆盖：

- `/api/v1/health`
- Web UI 首页可访问
- Web UI 页面级结构回归，覆盖技能目录页、管理端仓库表单、项目工作流入口、secretKey 写入入口和页面初始读取 API
- CLI bridge 的 `repo list` / `list` / `status`
- CLI bridge 的 `use -> apply -> feedback -> push --dry-run --json` 完整写操作链路
- 未配置 `secretKey` 时修改类 API 返回只读错误，配置后修改类 API 要求 `X-Skill-Hub-Secret-Key`
- `serve register -> start -> status -> stop -> remove` 的服务实例管理链路

---

## 16. 验收标准

当前第一阶段应满足：

1. 可以执行：

```bash
skill-hub serve
skill-hub serve --secret-key write-secret
skill-hub serve register local --host 127.0.0.1 --port 6600 --secret-key write-secret
skill-hub serve start local
skill-hub serve status local
skill-hub serve stop local
```

2. 浏览器可访问：

```text
http://127.0.0.1:5525
```

3. 页面可以：

- 查看 repo
- 管理 repo
- 查看 skills
- 查看本地项目及其 skills
- 触发项目状态与技能相关操作

4. 当 service 已启动时：

- `skill-hub repo list`
- `skill-hub repo sync`
- `skill-hub list`
- `skill-hub status`
- `skill-hub use`
- `skill-hub apply`
- `skill-hub feedback`

可通过 service 完成功能交互

5. 当 service 未启动时：

上述命令仍可本地工作

6. 对于命名服务实例管理：

- `serve status` 能正确区分 `running` / `stopped` / `stale`
- `serve remove` 不允许删除运行中的实例
- `serve stop` 会清理注册表中的运行态 `pid`
- 默认 loopback 监听会拒绝非 loopback Host header，并通过 service mode e2e 覆盖
- 默认 loopback 监听会拒绝跨站写请求，并通过 service mode e2e 覆盖
- 响应安全头通过 server handler 单测与 service mode e2e 覆盖
- 写权限只读模式和 `secretKey` 校验通过 server handler 单测、hubclient 单测与 service mode e2e 覆盖

---

## 17. 后续待办

主业务收口期间不继续扩展以下事项，仅作为后续阶段保留：

1. 为 Web UI 补页面级自动化测试，而不是只靠 API 与 CLI e2e
2. 为服务模式补更细的错误态展示，例如反馈冲突、仓库同步失败、apply 失败原因
3. 继续评估更细的本地会话体验，例如密钥轮换、失败次数限制或浏览器端更友好的输入状态
4. 继续评估 `remove` 是否需要服务化，并补齐 validate 在 Web UI 侧的入口
5. 为后续 `skill-node` / `MN` 对接预留节点管理 API，而不是把本地管理 API 继续做重
