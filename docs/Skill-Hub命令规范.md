# skill-hub 命令规范

## 1. 简介

本文档定义了 skill-hub CLI 工具的所有命令规范，旨在统一各设计文档中的命令定义，消除冲突和歧义。

## 名词说明

* **项目**：对应OpenCode,Cursor,Claude的项目

* **工作区**：项目所属的工作目录，OpenCode、Claude Code 和 Cursor 都会在工作区里创建管理 Skill 或行为规范的目录与文件，针对 OpenCode 来说对应的是 .agents 目录，针对 Claude (Claude Code) 来说对应 .claude 目录，针对 Cursor 来说对应 .cursorrules 文件。

* **本地仓库**：skill-hub使用的本地配置目录，采用多仓库架构，支持管理多个Git仓库。默认仓库为归档仓库，所有通过`feedback`命令修改的技能都会归档到默认仓库。

* **多仓库架构**：skill-hub支持同时管理多个技能仓库，如个人仓库、社区仓库、官方仓库等。每个仓库可以独立启用、禁用、同步。


## 2. 命令语法约定

- `<参数>`：必需参数
- `[参数]`：可选参数
- `[--选项 值]`：可选选项
- `...`：可重复参数
- `|`：多个值中选择一个

## 3. 核心命令列表（按功能分组）

### 3.1. 环境配置
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `init` | 初始化本地仓库 | `skill-hub init [git_url] [--target <value>]` |
| `set-target` | 设置项目目标环境 | `skill-hub set-target <value>` |

### 3.2. 技能发现
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `list` | 列出可用技能 | `skill-hub list [--target <value>] [--verbose]` |
| `search` | 搜索远程技能 | `skill-hub search <keyword> [--target <value>] [--limit <number>]` |

### 3.3. 技能创建/移除/验证/使用
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `create` | 创建新技能模板 | `skill-hub create <id> [--target <value>]` |
| `remove` | 移除项目技能 | `skill-hub remove <id>` |
| `validate` | 验证技能合规性 | `skill-hub validate <id>` |
| `use` | 使用本地仓库里的指定技能 | `skill-hub use <id> [--target <value>]` |


### 3.4. 技能状态
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `status` | 检查技能状态 | `skill-hub status [id] [--verbose]` |

### 3.5. 项目工作区-本地仓库交互
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `apply` | 应用技能到项目 | `skill-hub apply [--dry-run] [--force]` |
| `feedback` | 将项目工作区技能修改内容更新至到本地仓库 | `skill-hub feedback <id> [--dry-run] [--force]` |

### 3.6. 多仓库管理
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `repo add` | 添加新仓库 | `skill-hub repo add <name> <url> [--branch BRANCH] [--type TYPE] [--description DESC]` |
| `repo list` | 列出所有仓库 | `skill-hub repo list` |
| `repo remove` | 移除仓库 | `skill-hub repo remove <name>` |
| `repo enable` | 启用仓库 | `skill-hub repo enable <name>` |
| `repo disable` | 禁用仓库 | `skill-hub repo disable <name>` |
| `repo default` | 设置默认仓库 | `skill-hub repo default <name>` |
| `repo sync` | 同步仓库 | `skill-hub repo sync [name]` |

### 3.7. 本地仓库同步
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `pull` | 从远程仓库拉取最新技能 | `skill-hub pull [--force] [--check]` |
| `push` | 推送本地更改到远程仓库 | `skill-hub push [--message MESSAGE] [--force] [--dry-run]` |
| `git` | Git仓库操作 | `skill-hub git <subcommand>` |

## 4. 命令详细规范

### 4.1 init - 初始化本地仓库

**语法**: `skill-hub init [git_url] [--target <value>]`

**参数**:
- `git_url` (可选): Git 仓库 URL，用于初始化技能仓库。如未提供，表示不使用远端仓库，只进行本地管理
- `--target <value>` (可选): 技能目标环境，默认为 `open_code`。

**功能描述**:

创建 `~/.skill-hub` 目录结构，初始化全局配置。采用多仓库架构，默认创建名为"main"的本地仓库。

如提供了`git_url` 参数，则克隆远程技能仓库到默认仓库；否则创建空的本地仓库。初始化后默认 target 为 `open_code`。

完成本地仓库初始化后，`registry.json`根据实际仓库里管理的skill进行刷新，保持与仓库里管理的列表一致。

如果多次执行`init`，在出现冲突时，会提示用户选择是否覆盖。

**示例**:
```bash
# 不使用远端仓库，只进行本地管理
skill-hub init

# 使用自定义仓库初始化
skill-hub init https://github.com/example/skills-repo.git

# 初始化并设置项目为 OpenCode 环境
skill-hub init https://github.com/example/skills-repo.git --target open_code
```

### 4.2 set-target - 设置项目工作区目标环境

**语法**: `skill-hub set-target <value>`

**参数**:
- `value` (必需): 目标环境值，支持 `cursor`、`claude`、`open_code`。

**功能描述**:

设置当前项目的首选目标环境，该设置会持久化到 `state.json` 中，影响后续 `create`, `use`, `apply`、`status`、`feedback` 等命令的行为。

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区，如果需要新建项目工作区，则需要同时根据`target`初始化对应的文件和目录，并刷新`state.json`


**示例**:
```bash
# 设置项目为 OpenCode 环境
skill-hub set-target open_code

# 设置项目为 Cursor 环境
skill-hub set-target cursor
```

### 4.3 list - 列出可用技能

**语法**: `skill-hub list [--target <value>] [--verbose]`

**选项**:
- `--target <value>`: 按目标环境过滤技能列表。
- `--verbose`: 显示详细信息，包括技能描述、版本、兼容性等。

**功能描述**:

显示所有已启用仓库中的技能，支持按目标环境过滤。默认显示简要列表，包含技能 ID、状态、版本和所属仓库信息。

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

**多仓库说明**：默认显示所有已启用仓库中的技能。技能列表会标注技能所属的仓库名称。

**示例**:
```bash
# 显示所有技能
skill-hub list

# 仅显示兼容 Cursor 的技能
skill-hub list --target cursor

# 显示详细信息
skill-hub list --verbose
```

### 4.4 search - 搜索远程技能

**语法**: `skill-hub search <keyword> [--target <value>] [--limit <number>]`

**参数**:
- `keyword` (必需): 搜索关键词。

**选项**:
- `--target <value>`: 按目标环境过滤搜索结果。
- `--limit <number>`: 限制返回结果数量，默认 20。

**功能描述**:

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

当前未实现

**示例**:
```bash
# 搜索 git 相关技能
skill-hub search git

# 搜索兼容 OpenCode 的database相关技能，限制 10 个结果
skill-hub search database --target open_code --limit 10
```

### 4.5 create - 在项目工作区创建一个新技能

**语法**: `skill-hub create <id> [--target <value>]`

**参数**:
- `id` (必需): 新技能的标识符。
- `--target <value>` (可选): 技能目标环境。

**功能描述**:

在项目工作区创建一个新技能。如果指定了 `--target` 选项，则创建的技能将用于该目标环境。否则将使用`init`初始化时设置的默认目标环境作为项目工作区的目标环境。

创建的技能仅存在于项目本地，需要通过 `feedback` 命令同步到仓库。

`create` 命令将会刷新本地仓库的`state.json`，标记当前项目工作区在使用该技能

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区，如果需要新建项目工作区，则需要同时根据`target`初始化对应的文件和目录，并刷新`state.json`


**示例**:
```bash
# 创建名为 my-logic 的技能
skill-hub create my-logic

# 创建兼容 OpenCode 的 my-logic 技能
skill-hub create my-logic --target open_code
```

### 4.6 remove - 移除项目技能

**语法**: `skill-hub remove <id>`

**参数**:
- `id` (必需): 要移除的技能标识符。

**功能描述**:

从当前项目工作区中移除指定的技能：
1. 从 `state.json` 中移除技能标记
2. 物理删除项目工作区对应的文件/配置
3. 保留本地仓库中的源文件不受影响

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区，如果需要新建项目工作区，则需要同时根据`target`初始化对应的文件和目录，并刷新`state.json`

**安全机制**: 如果检测到本地有未反馈的修改，会弹出警告并要求确认。

**示例**:
```bash
# 移除 git-expert 技能
skill-hub remove git-expert
```

### 4.7 validate - 验证技能合规性

**语法**: `skill-hub validate <id>`

**参数**:
- `id` (必需): 要验证的技能标识符。

**功能描述**:

验证指定技能的项目工作区的文件是否合规，包括检查 `SKILL.md` 的 YAML 语法、必需字段、文件结构等。验证范围包括项目本地文件和仓库源文件。

如果该技能未在`state.json`里项目工作区登记，则提示该技能非法

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区，如果需要新建项目工作区，则需要同时根据`target`初始化对应的文件和目录，并刷新`state.json`

**示例**:
```bash
# 验证 my-logic 技能的合规性
skill-hub validate my-logic
```

### 4.8 use - 使用技能

**语法**: `skill-hub use <id> [--target <value>]`

**参数**:
- `id` (必需): 要启用的技能标识符。
- `--target <value>` (可选): 技能目标环境，默认为 `open_code`。

**功能描述**:

将技能标记为在当前项目中使用。此命令仅更新 `state.json` 中的状态记录，不生成物理文件。需要通过 `apply` 命令进行物理分发。

如果项目工作区里首次使用技能，也会同步在`state.json`里完成项目工作区信息刷新

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区，如果需要新建项目工作区，则需要同时根据`target`初始化对应的文件和目录，并刷新`state.json`

**示例**:
```bash
# 启用 git-expert 技能
skill-hub use git-expert

# 启用兼容 OpenCode 的 git-expert 技能
skill-hub use git-expert --target open_code
```

### 4.9 status - 检查技能状态

**语法**: `skill-hub status [id] [--verbose]`

**参数**:
- `id` (可选): 特定技能的标识符。如未提供，检查所有技能。

**选项**:
- `--verbose`: 显示详细差异信息。

**功能描述**:

对比项目本地工作区文件与技能仓库源文件的差异，显示技能状态：
- `Synced`: 本地与仓库一致
- `Modified`: 本地有未反馈的修改
- `Outdated`: 仓库版本领先于本地
- `Missing`: 技能已启用但本地文件缺失

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区，如果需要新建项目工作区，则需要同时根据`target`初始化对应的文件和目录，并刷新`state.json`

**示例**:
```bash
# 检查所有技能状态
skill-hub status

# 检查特定技能状态
skill-hub status git-expert

# 显示详细差异
skill-hub status --verbose
```

### 4.10 apply - 应用技能到项目工作区

**语法**: `skill-hub apply [--dry-run] [--force]`

**选项**:
- `--dry-run`: 演习模式，仅显示将要执行的变更，不实际修改文件。
- `--force`: 强制应用，即使检测到冲突也继续执行。

**功能描述**:

根据 `state.json` 中的启用记录和目标环境设置，将技能物理分发到项目工作区。具体行为取决于项目工作区设置的目标环境

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区，如果需要新建项目工作区，则需要同时根据`target`初始化对应的文件和目录，并刷新`state.json`

**示例**:
```bash
# 应用启用技能，使用项目工作区设置的目标环境
skill-hub apply

# 演习模式查看将要进行的变更
skill-hub apply --dry-run
```

### 4.11 feedback - 将项目工作区里技能修改更新至本地仓库

**语法**: `skill-hub feedback <id> [--dry-run] [--force]`

**参数**:
- `id` (必需): 要更新的技能标识符。

**选项**:
- `--dry-run`: 演习模式，仅显示将要同步的差异。
- `--force`: 强制更新，即使有冲突也继续执行。

**功能描述**:

将项目工作区的指定技能修改同步回本地技能仓库。此命令会：
1. 提取项目工作区本地文件内容所有内容
2. 与本地仓库源文件对比，显示差异
3. 经用户确认后更新本地仓库文件
4. 更新 `registry.json` 中的版本/哈希信息

**多仓库说明**：技能会被归档到默认仓库（通过 `skill-hub repo default` 命令设置）。如果技能在默认仓库中不存在则新增，存在则覆盖更新。

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区，如果需要新建项目工作区，则需要同时根据`target`初始化对应的文件和目录，并刷新`state.json`


**示例**:
```bash
# 反馈 my-logic 技能的修改
skill-hub feedback my-logic

# 演习模式查看将要同步的差异
skill-hub feedback git-expert --dry-run
```

### 4.12 pull - 从远程仓库拉取最新技能

**语法**: `skill-hub pull [--force] [--check]`

**选项**:
- `--force`: 强制拉取，忽略本地未提交的修改。
- `--check`: 检查模式，仅显示可用的更新，不实际执行拉取操作。

**功能描述**:

从远程技能仓库拉取最新更改到本地仓库（`~/.skill-hub/repo/`），并更新技能注册表（`registry.json`）。此命令仅同步仓库层，不涉及项目工作目录的更新。

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化


**关键行为**:
1. 执行 `git pull` 从远程仓库获取最新技能
2. 更新本地技能注册表（作为只读缓存）
3. 不修改项目状态或项目工作目录中的文件
4. 不更新项目的 `LastSync` 时间戳

**后续操作**:
- 使用 `skill-hub status` 检查项目技能状态
- 使用 `skill-hub apply` 将仓库更新应用到项目工作目录

**示例**:
```bash
# 从远程仓库拉取最新技能
skill-hub pull

# 检查可用更新但不实际执行拉取
skill-hub pull --check
```

### 4.13 push - 推送本地更改到远程仓库

**语法**: `skill-hub push [--message MESSAGE] [--force] [--dry-run]`

**选项**:
- `--message MESSAGE`, `-m MESSAGE`: 提交消息。如未提供，使用默认消息"更新技能"。
- `--force`: 强制推送，跳过确认检查。
- `--dry-run`: 演习模式，仅显示将要推送的更改，不实际执行。

**功能描述**:

自动检测并提交所有未提交的更改，然后推送到远程技能仓库。此命令将本地仓库（`~/.skill-hub/repo/`）中的更改同步到远程仓库，完成反馈闭环。

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

**关键行为**:
1. 检查本地仓库是否有未提交的更改（Modified、Untracked、Deleted文件）
2. 如有更改，自动提交（使用指定或默认消息）
3. 将提交推送到远程仓库
4. 如无更改可推送，提示"没有要推送的更改"

**注意**: 此命令仅操作本地Git仓库到远程仓库的同步，不涉及项目工作目录或技能状态。

**示例**:
```bash
# 推送所有本地更改，使用默认提交消息
skill-hub push

# 使用自定义提交消息推送
skill-hub push --message "修复技能描述"

# 演习模式，仅查看将要推送的更改
skill-hub push --dry-run
```

### 4.14 git - Git仓库操作

**语法**: `skill-hub git <subcommand> [options]`

**子命令**:
- `clone [url]` - 克隆远程技能仓库到本地
- `sync` - 同步技能仓库（拉取最新更改）
- `status` - 查看仓库状态（未提交的更改等）
- `commit` - 提交更改到本地仓库（交互式输入消息）
- `push` - 推送本地提交到远程仓库（检查未提交更改）
- `pull` - 从远程仓库拉取更新（同 `sync`）
- `remote [url]` - 设置或更新远程仓库URL

**子命令语法**:
- `skill-hub git clone [url]` - 克隆远程技能仓库到本地
- `skill-hub git sync` - 同步技能仓库（拉取最新更改）
- `skill-hub git status` - 查看仓库状态（未提交的更改等）
- `skill-hub git commit` - 提交更改到本地仓库（交互式输入消息）
- `skill-hub git push` - 推送本地提交到远程仓库（检查未提交更改）
- `skill-hub git pull` - 从远程仓库拉取更新（同 `sync`）
- `skill-hub git remote [url]` - 设置或更新远程仓库URL

**功能描述**:
提供底层Git仓库操作接口，适用于需要精细控制Git工作流的用户。这些命令直接操作本地技能仓库（`~/.skill-hub/repo/`），与高级命令（如 `pull`、`push`）功能重叠但提供更多控制选项。

**与高级命令的关系**:
- `skill-hub pull` = `skill-hub git sync`（但包含注册表更新）
- `skill-hub push` = `skill-hub git commit` + `skill-hub git push`（自动化版本）
- `skill-hub git` 命令提供更多手动控制和详细选项

**示例**:
```bash
# 克隆远程技能仓库
skill-hub git clone https://github.com/example/skills-repo.git

# 查看仓库状态
skill-hub git status

# 提交更改（交互式输入消息）
skill-hub git commit

# 设置远程仓库URL
skill-hub git remote https://github.com/your-username/skills-repo.git
```

## 5. 全局选项

### 5.1 帮助选项
- `-h, --help`: 显示帮助信息
  - `skill-hub --help`: 显示所有命令
  - `skill-hub <command> --help`: 显示特定命令帮助

### 5.2 版本选项
- `-v, --version`: 显示版本信息
  ```bash
  skill-hub --version
  ```

### 5.3 通用选项
- `--dry-run`: 演习模式（支持命令: `apply`, `feedback`, `push`）
- `--force`: 强制模式（支持命令: `apply`, `feedback`, `pull`, `push`）

## 6. 使用示例

### 6.1 完整工作流示例

```bash
# 1. 初始化环境
skill-hub init

# 2. 设置项目目标
skill-hub set-target open_code

# 3. 拉取远端技能仓库，以刷新本地仓库
skill-hub pull

# 4. 查看可用技能
skill-hub list --target open_code

# 5. 启用技能
skill-hub use git-expert

# 6. 应用技能到项目
skill-hub apply

# 7. 本地修改后检查状态
skill-hub status

# 8. 反馈修改到仓库
skill-hub feedback git-expert

# 9. 将本地仓库技能推送到远程仓库
skill-hub push
```

### 7.2 技能开发工作流

```bash
# 1. 创建新技能模板
skill-hub create my-new-skill

# 2. 编辑技能文件
vim .agents/skills/my-new-skill/SKILL.md

# 3. 验证技能合规性
skill-hub validate my-new-skill

# 4. 反馈到仓库
skill-hub feedback my-new-skill
```

## 7. 多仓库管理命令详细规范

### 7.1 repo add - 添加新仓库

**语法**: `skill-hub repo add <name> <url> [--branch BRANCH] [--type TYPE] [--description DESC]`

**参数**:
- `name` (必需): 仓库名称，用于标识仓库
- `url` (必需): Git仓库URL，支持HTTP/HTTPS和SSH协议。如不提供URL，则创建本地空仓库

**选项**:
- `--branch BRANCH`: Git分支，默认为 "main"
- `--type TYPE`: 仓库类型，支持 `user`（用户）、`community`（社区）、`official`（官方），默认为 "community"
- `--description DESC`: 仓库描述信息

**功能描述**:
添加新的Git仓库到技能库。支持从多个来源获取技能，如个人仓库、社区仓库、官方仓库等。

如果提供了URL，会克隆远程仓库到本地 `~/.skill-hub/repositories/<name>/` 目录。
如果没有提供URL，会创建一个空的本地仓库。

**示例**:
```bash
# 添加社区仓库
skill-hub repo add community https://github.com/skill-hub-community/awesome-skills.git

# 添加团队仓库，指定分支
skill-hub repo add team git@github.com:company/skills.git --branch develop --type user

# 创建本地空仓库
skill-hub repo add local --type user
```

### 7.2 repo list - 列出所有仓库

**语法**: `skill-hub repo list`

**功能描述**:
列出所有已配置的Git仓库，显示仓库状态和基本信息，包括：
- 仓库名称
- 类型（user/community/official）
- 是否启用
- 是否为默认仓库（归档仓库）
- 描述信息

**示例**:
```bash
# 列出所有仓库
skill-hub repo list
```

### 7.3 repo remove - 移除仓库

**语法**: `skill-hub repo remove <name>`

**参数**:
- `name` (必需): 要移除的仓库名称

**功能描述**:
从配置中移除指定的仓库。如果仓库是默认仓库，需要先设置其他仓库为默认仓库才能移除。

**安全机制**: 会提示用户确认，防止误操作。

**示例**:
```bash
# 移除名为test的仓库
skill-hub repo remove test
```

### 7.4 repo enable - 启用仓库

**语法**: `skill-hub repo enable <name>`

**参数**:
- `name` (必需): 要启用的仓库名称

**功能描述**:
启用指定的仓库。启用后，该仓库中的技能会出现在 `skill-hub list` 命令的结果中。

**示例**:
```bash
# 启用名为community的仓库
skill-hub repo enable community
```

### 7.5 repo disable - 禁用仓库

**语法**: `skill-hub repo disable <name>`

**参数**:
- `name` (必需): 要禁用的仓库名称

**功能描述**:
禁用指定的仓库。禁用后，该仓库中的技能不会出现在 `skill-hub list` 命令的结果中，但仓库配置仍然保留。

**示例**:
```bash
# 禁用名为test的仓库
skill-hub repo disable test
```

### 7.6 repo default - 设置默认仓库

**语法**: `skill-hub repo default <name>`

**参数**:
- `name` (必需): 要设置为默认仓库的仓库名称

**功能描述**:
设置默认仓库（归档仓库）。所有通过 `feedback` 命令修改的技能都会归档到默认仓库。
如果技能在默认仓库中不存在则新增，存在则覆盖更新。

默认仓库必须处于启用状态。

**示例**:
```bash
# 设置main为默认仓库
skill-hub repo default main
```

### 7.7 repo sync - 同步仓库

**语法**: `skill-hub repo sync [name]`

**参数**:
- `name` (可选): 要同步的仓库名称。如未提供，同步所有启用的仓库

**功能描述**:
同步指定仓库或所有启用的仓库。对于远程仓库，会执行 `git pull` 获取最新内容。
同步后会自动刷新技能索引。

**示例**:
```bash
# 同步所有启用的仓库
skill-hub repo sync

# 同步特定仓库
skill-hub repo sync community
```

## 8. 更新记录

| 版本 | 日期 | 更新说明 |
|------|------|----------|
| 1.0 | 2026-02-08 | 初始版本，统一所有设计文档中的命令定义 |
| 1.1 | 2026-02-09 | 核对各个命令描述，增加依赖信息说明 |
| 1.2 | 2026-02-17 | 添加多仓库管理命令，更新init命令描述以支持多仓库架构 |
