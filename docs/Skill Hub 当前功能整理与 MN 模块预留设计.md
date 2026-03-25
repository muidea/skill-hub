# Skill Hub 当前功能整理与 MN 模块预留设计

**文档状态**：面向后续 `mn` 模块开发前的整理说明  
**适用范围**：当前 `skill-hub` 已有能力边界、模块归类、后续迁移顺序  
**当前验证状态**：`go test ./...` 通过；`tests/e2e` 当前总结果为 `102 passed, 3 skipped`

---

## 1. 目标

在将 `mn` 作为 `skill-hub` 的一个功能模块开发之前，先把当前已有能力整理成清晰边界，避免出现：

- 新功能直接耦合到 CLI 命令层
- 配置、状态、多仓库、适配器被分散调用
- 后续 `mn` 模块无法复用现有能力

当前整理原则：

1. 不大规模搬动已有成熟包
2. 先增加标准入口和模块壳层
3. 先把依赖访问统一到稳定服务层
4. 后续按模块边界逐步把旧包迁入 `internal/modules/...`

---

## 2. 当前功能归类

### 2.1 主应用入口

当前推荐入口：

- `application/skill-hub/cmd/main.go`

说明：

- 该入口用于对齐 `go-multi-module-dev`
- `application/skill-validate/cmd/main.go` 作为独立校验工具入口保留

### 2.2 已建立的核心模块壳层

#### `internal/modules/kernel/hub`

职责：

- 作为当前应用主链路壳层
- 承接主入口调用
- 后续可逐步接入更多 kernel 模块编排

#### `internal/modules/kernel/runtime`

职责：

- 统一封装 CLI 与服务模式最常用的底层依赖

当前已收口：

- 配置读取
- 根目录定位
- 状态管理器创建
- 多仓库管理器创建
- 目标适配器获取
- 默认仓库技能内容读取
- 服务模式下项目状态 / use / apply / feedback 服务封装

这是后续 `mn` 模块最重要的预留点之一。

#### `internal/modules/kernel/repository`

职责：

- 统一承接多仓库管理、默认仓库定位、仓库路径解析
- 对外提供默认仓库技能内容读取入口

#### `internal/modules/kernel/project_state`

职责：

- 统一承接项目状态管理器创建
- 为 CLI、服务模式、后续 `mn` 复用项目状态能力

#### `internal/modules/kernel/skill`

职责：

- 统一承接技能目录定位和技能管理器创建

#### `internal/modules/blocks/adapter`

职责：

- 统一承接目标适配器选择与可用适配器枚举

#### `internal/modules/blocks/git`

职责：

- 统一承接 Git 高频动作

当前已承接的高频动作包括：

- 技能仓库同步并刷新索引
- 技能仓库状态读取
- 提交并推送本地修改
- 推送已存在提交
- 设置默认仓库远程地址

### 2.3 现有旧目录能力的逻辑归属

当前仍保留在旧目录、但逻辑上已可归类到 kernel / blocks 的能力：

- `internal/config`
- `internal/state`
- `internal/multirepo`
- `internal/engine`
- `internal/adapter`
- `internal/git`
- `internal/template`

当前策略仍是“先收口依赖，再渐进迁移”，不强求一次性搬目录。

### 2.4 当前索引实现状态

当前索引能力已经从“单一根目录索引”收口到“repo 级索引优先，根索引兼容保留”的状态：

- 每个仓库目录下维护自己的 `registry.json`
- `list` / completion 优先读取 repo 级 `registry.json`
- repo 级索引不存在或不可解析时，回退到文件系统扫描
- 根目录 `~/.skill-hub/registry.json` 目前保留为兼容产物，不再作为唯一真相源

当前已接入索引刷新的链路：

- `init`
- `pull`
- `git sync`
- `git pull`
- `repo add`
- `repo sync`
- `feedback`

### 2.5 当前新增的服务模式模块

#### `internal/modules/kernel/server`

职责：

- 本地服务启动编排
- HTTP server 生命周期管理

#### `internal/modules/blocks/httpapi`

职责：

- 本地管理 API
- 健康检查
- repo / skills / projects / project-status / project-use / project-apply / project-feedback 路由

#### `internal/modules/blocks/hubclient`

职责：

- CLI 到本地服务的桥接 client
- 优先服务、失败回退本地的能力基石

#### `internal/modules/blocks/webui`

职责：

- 托管本地 Web UI
- 提供仓库、技能、项目操作界面

#### `internal/modules/kernel/project_inventory`

职责：

- 基于 `state.json` 聚合本地项目与技能视图

#### `internal/modules/kernel/project_status`

职责：

- 统一项目技能状态摘要

#### `internal/modules/kernel/project_use`

职责：

- 统一项目技能启用逻辑

#### `internal/modules/kernel/project_apply`

职责：

- 统一项目技能分发逻辑

#### `internal/modules/kernel/project_feedback`

职责：

- 统一反馈预览、版本推进、归档与索引刷新逻辑

---

## 3. 当前 CLI 层整理结果

当前 CLI 的依赖不再优先直接访问分散底层包，而是优先通过统一 helper 进入 `runtime` 模块服务。

当前已整理或接入统一边界的命令包括：

- `serve`
- `apply`
- `use`
- `repo`
- `list`
- `feedback`
- `status`
- `prune`
- `git`
- `create`
- `init`
- `completer`
- `dependencies`

当前已经下沉到 `runtime / repository / git` 服务层的典型动作包括：

- 默认仓库归档与索引刷新
- repo 级索引重建
- 仓库 add / list / remove / sync / enable / disable / default
- 默认技能仓库 sync / status / push / remote set
- 技能列表元数据读取与 completion 热路径
- 服务模式下的项目状态、启用、分发、反馈

整理收益：

- 新功能不需要再到每个命令里各自 new `StateManager` / `RepoManager`
- 服务模式和后续 `mn` 都可以直接复用统一运行时服务
- CLI 从“业务实现层”进一步收成“调用层”

---

## 4. Runtime 能力清单

### 4.1 已适合直接复用的能力

以下能力已经具备相对稳定的服务边界，后续 `mn` 可以优先通过 `internal/modules/kernel/runtime/service.Runtime` 访问：

- 配置与路径
  - `Config()`
  - `RootDir()`
  - `RepositoryPath(repoName)`
  - `DefaultRepository()`
- 项目状态
  - `StateManager()`
- 仓库元数据读取
  - `ListRepositories(includeDisabled)`
  - `ListSkillMetadata(repoNames)`
  - `GetRepository(name)`
  - `ReadDefaultRepositorySkillContent(skillID)`
- 仓库内容维护
  - `RebuildRepositoryIndex(repoName)`
  - `ArchiveToDefaultRepository(skillID, sourcePath)`
  - `AddRepository(...)`
  - `RemoveRepository(name)`
  - `SyncRepository(name)`
  - `EnableRepository(name)`
  - `DisableRepository(name)`
  - `SetDefaultRepository(name)`
  - `UpdateRepositoryURL(name, url)`
- Git 高频动作
  - `SyncSkillRepositoryAndRefresh()`
  - `SkillRepositoryStatus()`
  - `PushSkillRepositoryChanges(message)`
  - `PushSkillRepositoryCommits()`
  - `SetSkillRepositoryRemote(url)`
- 适配器与辅助能力
  - `Adapter(target)`
  - `CleanupTimestampedBackupDirs(basePath)`
- 项目服务能力
  - `ProjectStatus(...)`
  - `EnableSkill(...)`
  - `ApplyProject(...)`
  - `PreviewFeedback(...)`
  - `ApplyFeedback(...)`

这些接口的共同特点是：

- 已经被现有 CLI 或服务模式主链复用
- 已通过当前回归测试和 e2e 验证
- 输入输出语义相对明确

### 4.2 当前仍属于过渡层的能力

以下能力当前可以使用，但更适合作为“兼容或过渡接口”，不建议 `mn` 长期直接依赖：

- `RepositoryManager()`
- `GitRepository(repoPath)` / `SkillsRepository()` / `SkillRepository()`

### 4.3 当前暂不建议 MN 直接依赖的部分

- `internal/cli/*`
- `internal/config` 的全局单例语义
- `internal/git` / `internal/multirepo` / `internal/state` 的底层对象本身

### 4.4 对 MN 的直接建议

如果现在开始设计 `mn`，建议优先只依赖下面这组最小接口面：

1. `Config()` / `RootDir()`
2. `ListRepositories()` / `DefaultRepository()` / `RepositoryPath()`
3. `ListSkillMetadata()` / `ReadDefaultRepositorySkillContent()`
4. `StateManager()`
5. `RebuildRepositoryIndex()` / `SyncRepository()`

这组能力已经足够支撑：

- 节点配置加载
- 仓库与技能元数据枚举
- 默认仓库查询
- 项目状态读取
- 仓库同步与索引刷新

---

## 5. 当前服务模式落地状态

当前 `skill-hub` 已不再是单一 CLI 工具，而是双运行模式：

- CLI 模式
- Service 模式：`skill-hub serve`

当前服务模式已经提供：

- 本地 HTTP API
- 本地 Web UI
- CLI bridge

当前 CLI 已能优先通过服务完成的命令：

- `repo *`
- `list`
- `set-target`
- `status`
- `use`
- `apply`
- `feedback`

当前仍以本地执行为主的命令：

- `init`
- `search`
- `create`
- `remove`
- `validate`
- `prune`
- `git/pull/push` 主链路

---

## 6. 对后续 MN 模块的具体帮助

未来 `mn` 模块建议作为：

- `internal/modules/kernel/mn`

其最初不应该直接依赖 `internal/cli`，而应优先复用：

- `internal/modules/kernel/runtime`
- `internal/modules/kernel/repository`
- `internal/modules/kernel/project_state`
- `internal/modules/kernel/skill`
- `internal/modules/blocks/adapter`
- `internal/modules/blocks/git`

这样 `mn` 可以直接专注于：

- 节点注册与连接管理
- 任务状态机
- 消息分发
- 数据表与 API

而不是重新实现仓库、状态、技能或适配器逻辑。

---

## 7. 建议的下一步迁移顺序

### Phase 1：继续完成当前功能边界整理

1. 服务模式页面级测试补齐
2. 服务模式错误模型继续收敛
3. 推进 `search` 的服务桥接闭环

### Phase 2：提炼 MN 所需服务接口

建议补接口层：

- `NodeService`
- `TaskService`
- `ConnectionService`
- `AuditService`

### Phase 3：引入 MN 模块

待前两阶段完成后，再新增：

- `internal/modules/kernel/mn`

初版职责：

- 连接管理
- 任务调度
- 协议编解码
- 数据存储协调

---

## 8. 当前结论

当前 `skill-hub` 已经完成了几项关键准备工作：

1. 主应用入口已对齐 `go-multi-module-dev`
2. CLI 依赖已经统一收口到 `kernel/runtime`
3. `repository / project_state / skill / adapter / git` 已有第一层模块边界
4. 服务模式已经打通 HTTP API、Web UI 和 CLI bridge
5. 服务模式已有专项 e2e，并已并入总体验证

因此，下一步可以不再回头做基础入口清理，而是继续进入：

- 服务模式收尾
- `mn` service contract 设计
- `mn` 模块最小骨架实现
