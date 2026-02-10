# skill-hub 命令规范

## 1. 简介

本文档定义了 skill-hub CLI 工具的所有命令规范，旨在统一各设计文档中的命令定义，消除冲突和歧义。

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
| `init` | 初始化环境 | `skill-hub init [git_url] [--target <value>]` |
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

### 3.5. 项目-仓库交互
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `apply` | 应用技能到项目 | `skill-hub apply [--dry-run] [--force]` |
| `feedback` | 将项目工作区技能修改内容更新至到本地仓库 | `skill-hub feedback <id> [--dry-run] [--force]` |

### 3.6. 仓库同步
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `pull` | 从远程仓库拉取最新技能 | `skill-hub pull [--force] [--check]` |
| `push` | 推送本地更改到远程仓库 | `skill-hub push [--message MESSAGE] [--force] [--dry-run]` |
| `git` | Git仓库操作 | `skill-hub git <subcommand>` |

## 4. 命令详细规范

### 4.1 init - 初始化环境

**语法**: `skill-hub init [git_url] [--target <value>]`

**参数**:
- `git_url` (可选): Git 仓库 URL，用于初始化技能仓库。如未提供，表示不使用远端仓库，只进行本地管理
- `--target <value>` (可选): 技能目标环境，默认为 `open_code`。

**功能描述**:
创建 `~/.skill-hub` 目录结构，初始化全局配置。如提供了 git_url 参数，则克隆远程技能仓库；否则仅进行本地管理。初始化后默认 target 为 `open_code`。

**示例**:
```bash
# 不使用远端仓库，只进行本地管理
skill-hub init

# 使用自定义仓库初始化
skill-hub init https://github.com/example/skills-repo.git

# 初始化并设置项目为 OpenCode 环境
skill-hub init https://github.com/example/skills-repo.git --target open_code
```

### 4.2 set-target - 设置项目目标环境

**语法**: `skill-hub set-target <value>`

**参数**:
- `value` (必需): 目标环境值，支持 `cursor`、`claude`、`open_code`。

**功能描述**:
设置当前项目的首选目标环境，该设置会持久化到 `state.json` 中，影响后续 `apply`、`status`、`feedback` 等命令的行为。

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
显示本地技能仓库中的所有技能，支持按目标环境过滤。默认显示简要列表，包含技能 ID、状态和版本信息。

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
通过 GitHub API 搜索带有 `agent-skills` 标签的远程技能仓库。返回包含技能描述、星标、最后更新时间等信息。

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
- `--target <value>` (可选): 技能目标环境，默认为 `open_code`。

**功能描述**:
在项目当前工作区创建一个新技能。如果指定了 `--target` 选项，则创建的技能将用于该目标环境。否则将用于init 初始化时设置的默认目标环境。

创建的技能仅存在于项目本地，需要通过 `feedback` 命令同步到仓库。

create 命令将会刷新state.json，标记当前项目工作区在使用该技能。

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
从当前项目中移除指定的技能：
1. 从 `state.json` 中移除技能标记
2. 物理删除项目本地工作区对应的文件/配置
3. 保留仓库中的源文件不受影响

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
验证指定技能的项目本地工作区的文件是否合规，包括检查 `SKILL.md` 的 YAML 语法、必需字段、文件结构等。验证范围包括项目本地文件和仓库源文件。

如果该技能未在`state.json`里项目工作区登记，则提示该技能非法

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

**示例**:
```bash
# 检查所有技能状态
skill-hub status

# 检查特定技能状态
skill-hub status git-expert

# 显示详细差异
skill-hub status --verbose
```

### 4.10 apply - 应用技能到项目

**语法**: `skill-hub apply [--dry-run] [--force]`

**选项**:
- `--dry-run`: 演习模式，仅显示将要执行的变更，不实际修改文件。
- `--force`: 强制应用，即使检测到冲突也继续执行。

**功能描述**:
根据 `state.json` 中的启用记录和目标环境设置，将技能物理分发到项目。具体行为取决于项目工作区设置的目标环境：
- `cursor`: 注入到 `.cursorrules` 文件
- `claude`: 更新 Claude 配置文件
- `open_code`: 创建 `.agents/skills/[id]/` 目录结构

**示例**:
```bash
# 应用启用技能，使用项目工作区设置的目标环境
skill-hub apply

# 演习模式查看将要进行的变更
skill-hub apply --dry-run
```

### 4.11 feedback - 将项目工作技能修改更新至本地仓库

**语法**: `skill-hub feedback <id> [--dry-run] [--force]`

**参数**:
- `id` (必需): 要更新的技能标识符。

**选项**:
- `--dry-run`: 演习模式，仅显示将要同步的差异。
- `--force`: 强制更新，即使有冲突也继续执行。

**功能描述**:
将项目工作区本地的技能修改同步回本地技能仓库。此命令会：
1. 提取项目工作区本地文件内容
2. 与本地仓库源文件对比，显示差异
3. 经用户确认后更新本地仓库文件
4. 更新 `registry.json` 中的版本/哈希信息

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

## 8. 更新记录

| 版本 | 日期 | 更新说明 |
|------|------|----------|
| 1.0 | 2026-02-08 | 初始版本，统一所有设计文档中的命令定义 |
