# Skill Hub 当前功能整理与 MN 模块预留设计

**文档状态**：面向后续 `mn` 模块开发前的整理说明  
**适用范围**：当前 `skill-hub` 已有能力边界、模块归类、后续迁移顺序

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

- 统一封装 CLI 最常用的底层依赖
- 当前已收口：
  - 配置读取
  - 根目录定位
  - 状态管理器创建
  - 多仓库管理器创建
  - 目标适配器获取
  - 默认仓库技能内容读取

这是后续 `mn` 模块最重要的预留点之一。

#### `internal/modules/kernel/repository`

职责：

- 统一承接多仓库管理、默认仓库定位、仓库路径解析
- 对外提供默认仓库技能内容读取入口

当前状态：

- 已建立模块壳层
- `runtime` 已改为通过该模块访问仓库能力

#### `internal/modules/kernel/project_state`

职责：

- 统一承接项目状态管理器创建
- 为 CLI、后续 `mn`、未来 daemon/API 复用项目状态能力

当前状态：

- 已建立模块壳层
- `runtime` 已改为通过该模块访问状态能力

#### `internal/modules/kernel/skill`

职责：

- 统一承接技能目录定位和技能管理器创建
- 为后续技能相关能力继续内聚提供边界

当前状态：

- 已建立模块壳层
- `runtime` 已通过该模块暴露技能目录能力

#### `internal/modules/blocks/adapter`

职责：

- 统一承接目标适配器选择与可用适配器枚举
- 为后续 `mn` 或 API 层调用适配器时提供稳定入口

当前状态：

- 已建立模块壳层
- `runtime` 已改为通过该模块访问适配器能力

### 2.3 现有业务能力归类

#### 当前仍保留在旧目录、但逻辑上已可归类到 kernel 的能力

- `internal/config`
  - 配置模型与路径约定
- `internal/state`
  - 项目状态持久化
- `internal/multirepo`
  - 多仓库管理
- `internal/engine`
  - 技能加载与读取

建议后续逻辑归属：

- `config` 仍可保持独立基础设施包
- `state` 可逐步收进 `internal/modules/kernel/project_state`
- `multirepo` 可逐步收进 `internal/modules/kernel/repository`
- `engine` 可逐步收进 `internal/modules/kernel/skill`

#### 当前可归类到 blocks 的能力

- `internal/adapter`
  - Cursor / Claude / OpenCode 适配器
- `internal/git`
  - Git 操作封装
- `internal/template`
  - 模板处理

建议后续逻辑归属：

- `adapter` -> `internal/modules/blocks/adapter`
- `git` -> `internal/modules/blocks/git`
- `template` -> `internal/modules/blocks/template`

当前状态补充：

- `internal/modules/blocks/adapter` 已建立第一层模块壳层
- `internal/modules/blocks/git` 已建立第一层模块壳层
- `runtime` 与 CLI helper 已开始通过这两层模块访问适配器与 Git 能力
- `blocks/git` 当前已承接的高频动作包括：
  - 技能仓库同步并刷新索引
  - 技能仓库状态读取
  - 提交并推送本地修改
  - 推送已存在提交
  - 设置默认仓库远程地址

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

这意味着“仓库内容变化 -> 索引变化 -> list/completion 感知到变化”已经具备基本闭环。

---

## 3. 当前 CLI 层整理结果

本轮整理后，CLI 的依赖不再优先直接访问分散底层包，而是优先通过统一 helper 进入 `runtime` 模块服务。

当前已整理的命令包括：

- `apply`
- `use`
- `repo`
- `list`
- `feedback`
- `status`
- `git`
- `create`
- `init`
- `completer`
- `dependencies`

整理收益：

- 新功能不需要再到每个命令里各自 new `StateManager` / `RepoManager`
- 后续引入 `mn` 模块时，可以直接复用统一运行时服务
- 将 CLI 从“业务实现层”往“调用层”收了一步

当前已经下沉到 `runtime / repository / git` 服务层的典型动作包括：

- 默认仓库归档与索引刷新
- repo 级索引重建
- 仓库 add / list / remove / sync / enable / disable / default
- 默认技能仓库 sync / status / push / remote set
- 技能列表元数据读取与 completion 热路径

本轮之后，`runtime` 不再直接面向 `internal/state`、`internal/multirepo`、`internal/adapter` 暴露底层依赖入口，而是优先通过对应模块壳层进行访问。
同时，CLI 中 `init / pull / push / git` 这些命令对 `git`、`adapter` 的直接依赖也已经收回到统一 helper 层。
另外，`list` 和 shell completion 已经不再把“递归扫目录”作为唯一主路径，而是优先走 repo 级索引读取。

---

## 4. Runtime 能力清单

这一节的目标不是重复代码，而是明确：

- 哪些能力已经适合被后续 `mn` 直接复用
- 哪些能力虽然存在，但仍属于过渡接口
- 哪些能力目前仍不建议 `mn` 直接依赖

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

这些接口的共同特点是：

- 已经被现有 CLI 主链复用
- 已通过当前回归测试和 e2e 验证
- 输入输出语义相对明确

### 4.2 当前仍属于过渡层的能力

以下能力当前可以使用，但更适合作为“兼容或过渡接口”，不建议 `mn` 长期直接依赖：

- `RepositoryManager()`
  - 这是对 `multirepo.Manager` 的透传入口
  - 适合当前整理阶段复用，但长期应尽量被更细粒度的 repository service 方法替代
- `GitRepository(repoPath)` / `SkillsRepository()` / `SkillRepository()`
  - 这些接口仍然暴露底层仓库对象
  - CLI 整理过程中还在少量使用，但 `mn` 更适合优先依赖上面的动作级接口

### 4.3 当前暂不建议 MN 直接依赖的部分

- `internal/cli/*`
  - 命令层仍包含交互、格式化输出、用户确认逻辑
- `internal/config` 的全局单例语义
  - 对 CLI 足够，但对未来长期运行的 `mn` 来说仍偏全局态
- `internal/git` / `internal/multirepo` / `internal/state` 的底层对象本身
  - 这些包未来仍可能继续被模块服务包裹和收紧

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

如果 `mn` 的第一阶段不涉及 Git 写操作，则不必一开始就依赖 `PushSkillRepositoryChanges()` 或 `SetSkillRepositoryRemote()`。

---

## 5. 对后续 MN 模块的具体帮助

未来 `mn` 模块建议作为：

- `internal/modules/kernel/mn`

其最初不应该直接依赖 `internal/cli`，而应优先复用：

- `internal/modules/kernel/runtime`
- 后续逐步形成的：
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

## 6. 建议的下一步迁移顺序

### Phase 1：完成当前功能的模块边界整理

建议继续做：

1. `internal/multirepo` -> 包装为 `kernel/repository`，已完成第一层
2. `internal/state` -> 包装为 `kernel/project_state`，已完成第一层
3. `internal/engine` -> 包装为 `kernel/skill`，已完成第一层
4. `internal/adapter` -> 包装为 `blocks/adapter`，已完成第一层
5. `internal/git` -> 包装为 `blocks/git`，已完成第一层

注意：

- 这一阶段以“包裹和收口依赖”为主，不强求立即物理搬目录

### Phase 2：提炼服务接口

建议补接口层：

- `RepositoryService`
- `ProjectStateService`
- `SkillService`
- `AdapterService`

用途：

- 降低 CLI 与底层实现的耦合
- 为 `mn` 模块、未来 HTTP API 或 daemon 模式复用

### Phase 3：引入 MN 模块

待前两阶段完成后，再新增：

- `internal/modules/kernel/mn`

初版职责：

- 连接管理
- 任务调度
- 协议编解码
- 数据存储协调

---

## 6. 当前结论

当前 `skill-hub` 已经完成了三件关键准备工作：

1. 主应用入口已对齐 `go-multi-module-dev`
2. 当前 CLI 依赖已经开始统一收口到 `kernel/runtime`
3. `repository / project_state / skill / adapter` 已有第一层模块边界

因此，下一步不需要回头再做“入口层清理”，可以继续进入“业务模块服务接口提炼与 CLI 继续解耦”，然后再开始 `mn` 功能开发。
