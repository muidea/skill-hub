# Skill Hub 服务模式与本地管理界面设计文档

**文档状态**：待开发  
**适用范围**：`skill-hub` 在保留现有命令行模式基础上，新增本地服务运行模式、Web 管理界面，以及 CLI 通过服务交互的能力  
**当前阶段目标**：先完成服务模式本身，不先做 `skill-node` 对接

---

## 1. 目标

当前 `skill-hub` 是一个纯命令行工具。为了后续与 `skill-node` 的服务化管理能力对齐，需要先把 `skill-hub` 扩展成“双运行模式”：

- 命令行模式
- 服务模式

本阶段只完成以下能力：

1. `skill-hub` 支持以服务方式运行
2. 提供本地 Web 化管理界面
3. 当前 CLI 在服务存在时，可以通过服务进行功能交互

本阶段暂不做：

- `skill-node` 注册与调度
- 远程多用户访问
- 复杂认证系统
- 所有 CLI 命令一次性 API 化

---

## 2. 总体方案

`skill-hub` 维持现有 CLI 能力不变，同时新增：

- `skill-hub serve`

当以服务模式运行时，提供两类访问入口：

- 本地 HTTP API
- 本地 Web UI

CLI 与服务的关系：

- 默认仍然可以本地直接执行
- 当检测到本地 service 可用时，优先通过 service API 交互
- 当 service 不可用时，回退到当前本地执行逻辑
- 第一阶段只代理一部分命令，不要求全覆盖

这样可以保证：

- 不破坏现有 CLI 使用习惯
- 可以逐步把能力迁移到服务层
- 后续对接 `skill-node` 时不需要重新设计本地管理面

---

## 3. 运行模式设计

建议新增命令：

```bash
skill-hub serve
```

建议参数：

- `--host 127.0.0.1`
- `--port 5525`
- `--open-browser`

本阶段先不做：

- `--daemon`
- systemd/service install

服务启动后职责：

- 加载现有 `runtime`
- 暴露本地 HTTP API
- 托管 Web UI 静态资源
- 提供健康检查接口
- 为 CLI 提供本地进程间交互入口

---

## 4. 第一阶段功能范围

第一阶段只实现以下功能。

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

这部分本质上对应当前 `list` 命令。

### 4.3 本地工作区技能查看

支持：

- 查看当前机器已登记的项目列表
- 查看某个项目下启用的 skill
- 查看 skill 的版本、状态、目标环境

这部分以 `state.json` 为准，不要求第一阶段就做到完整的 Web 化 `status/apply/feedback` 操作。

---

## 5. 服务端模块设计

建议新增以下模块。

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
- 中间件
- 健康检查

### 5.3 `internal/modules/blocks/webui`

职责：

- 管理前端静态资源
- 提供页面路由
- 托管本地 Web UI

### 5.4 `internal/modules/kernel/project_inventory`

职责：

- 从 `state.json` 聚合“当前机器有哪些项目”
- 聚合每个项目启用了哪些 skill
- 给 Web UI 和 HTTP API 提供统一项目视图

---

## 6. 与现有模块的关系

本阶段尽量不重写已有能力，优先复用当前已经收口的模块服务：

- `internal/modules/kernel/runtime`
- `internal/modules/kernel/repository`
- `internal/modules/kernel/project_state`
- `internal/modules/blocks/git`

服务端 API handler 只调用这些 service，不直接依赖 CLI 命令层。

CLI bridge 也只调用 HTTP API，不重复实现业务逻辑。

---

## 7. HTTP API 设计

建议统一前缀：

```text
/api/v1
```

### 7.1 健康检查

- `GET /api/v1/health`

返回示例：

```json
{
  "code": "OK",
  "message": "",
  "data": {
    "status": "ok"
  }
}
```

### 7.2 仓库管理

- `GET /api/v1/repos`
- `POST /api/v1/repos`
- `POST /api/v1/repos/{name}/sync`
- `POST /api/v1/repos/{name}/enable`
- `POST /api/v1/repos/{name}/disable`
- `POST /api/v1/repos/{name}/set-default`
- `DELETE /api/v1/repos/{name}`

### 7.3 技能列表

- `GET /api/v1/skills`

支持查询参数：

- `repo`
- `target`

示例：

```text
/api/v1/skills?repo=main&target=open_code
```

### 7.4 项目与工作区技能

- `GET /api/v1/projects`
- `GET /api/v1/projects/{id}`
- `GET /api/v1/projects/{id}/skills`

---

## 8. API 响应规范

建议统一返回结构：

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

---

## 9. Web UI 设计

第一阶段只做 3 个页面。

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

### 9.4 前端技术约束

本阶段不单独起前端工程。

建议：

- 直接由 Go `embed` 静态资源
- 使用轻量 HTML/CSS/JS
- 页面直接调用 `/api/v1/*`

目标是：

- 先完成本地可用
- 降低运行和构建复杂度
- 方便后续再拆前后端

---

## 10. CLI 与服务交互设计

建议新增一个很薄的本地客户端层：

- `internal/modules/blocks/hubclient`

职责：

- 探测本地 service 是否存在
- 访问本地 HTTP API
- 将响应转换为 CLI 所需结果

### 10.1 第一批优先通过服务代理的命令

- `repo list`
- `repo add`
- `repo remove`
- `repo sync`
- `repo enable`
- `repo disable`
- `repo default`
- `list`

### 10.2 第二批后续再接的命令

- `status`
- `use`
- `apply`
- `feedback`

### 10.3 当前继续本地执行的命令

- `init`
- `validate`
- `create`
- `remove`

### 10.4 CLI 代理策略

建议策略：

1. 默认先尝试本地 service
2. 若本地 service 可达，则通过 API 执行
3. 若 service 不可达，则回退到本地逻辑

可增加开关：

```text
SKILL_HUB_DISABLE_SERVICE_BRIDGE=1
```

用于强制关闭服务代理。

---

## 11. 本地通信方式

第一阶段建议直接使用：

- `127.0.0.1:5525` HTTP

原因：

- 实现最简单
- 浏览器可直接访问
- CLI 可直接复用
- 集成测试容易搭建

本阶段先不做：

- Unix Domain Socket
- WebSocket
- 远程监听

---

## 12. 安全约束

第一阶段只做最小安全约束：

- 默认仅监听 `127.0.0.1`
- 不监听 `0.0.0.0`
- 不在 Web 页面显示 `git_token`
- 修改类接口仅面向本地访问场景设计

本阶段不做：

- 登录认证
- RBAC
- 远程会话管理

说明：

当前目标是“本机服务化”，不是多租户平台化。

---

## 13. 项目工作区技能聚合设计

Web 需要展示“当前机器本地工作区里管理的 skill”，其真相源为 `state.json`。

建议项目列表结构：

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

建议项目技能结构：

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

本阶段不要求主动扫描磁盘修复状态，以 `state.json` 为准。

---

## 14. 建议开发顺序

### Phase 1：服务模式骨架

实现：

- `skill-hub serve`
- HTTP server
- `/api/v1/health`

### Phase 2：仓库管理 API

实现：

- `GET /repos`
- `POST /repos`
- `DELETE /repos/{name}`
- `POST /repos/{name}/sync`
- `POST /repos/{name}/enable`
- `POST /repos/{name}/disable`
- `POST /repos/{name}/set-default`

### Phase 3：技能列表 API

实现：

- `GET /skills`

### Phase 4：项目视图 API

实现：

- `GET /projects`
- `GET /projects/{id}`
- `GET /projects/{id}/skills`

### Phase 5：Web UI

实现：

- 仓库页
- 技能页
- 项目页

### Phase 6：CLI bridge

第一批先代理：

- `repo *`
- `list`

---

## 15. 验收标准

第一阶段完成后，应满足：

1. 可以执行：

```bash
skill-hub serve
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

4. 当 service 已启动时：

- `skill-hub repo list`
- `skill-hub repo sync`
- `skill-hub list`

可通过 service 完成功能交互

5. 当 service 未启动时：

上述命令仍可本地工作

---

## 16. 当前定稿建议

建议按以下边界进入开发：

- 先做 `serve + HTTP API + Web UI + CLI bridge(repo/list)`
- 暂不做 `skill-node` 对接
- 暂不做远程认证
- 暂不做全量命令服务化

这是风险最低、最容易与当前实现兼容的一条路径。
