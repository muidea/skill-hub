# skill-hub 命令规范

## 1. 简介

本文档定义了 skill-hub CLI 工具的所有命令规范，旨在统一各设计文档中的命令定义，消除冲突和歧义。

## 安装脚本约定

一键安装脚本安装二进制、Shell 补全和 release 内置 agent workflow skills。Agent workflow skills 的主安装目录是工具无关路径：

```bash
${XDG_DATA_HOME:-$HOME/.local/share}/skill-hub/agent-skills
```

安装脚本与 `upgrade` 只同步到上述工具无关目录，不会直接写入任何全局 skills 目录。全局目录只能由 `skill-hub use --global` 与 `skill-hub apply --global` 管理，以确保每个托管副本都带有 `.skill-hub-manifest.json`，不会因二进制升级而被覆盖为非托管目录。

可通过 `SKILL_HUB_INSTALL_AGENT_SKILLS=0` 跳过所有内置 workflow skills 同步，通过 `SKILL_HUB_AGENT_SKILLS_DIR=/path/to/skills` 覆盖工具无关目录。安装脚本只覆盖该专属目录下 release 内置的 `skill-hub-*` workflow skills，不扫描或删除其他用户 skill。

内置 `upgrade` 命令复用同一套 GitHub Release 资产和 agent workflow skills 安装约定。Linux 与 macOS 支持自动替换当前二进制；Windows 当前只提示使用安装脚本或手动下载 Release 包。

## 名词说明

* **项目**：需要启用和管理 Skill 的代码工作区。

* **工作区**：项目所属的工作目录。项目本地 Skill 统一使用 `.agents/skills/` 目录；历史 target 参数不再决定工作区结构。

* **本机全局工作区**：`~/.skill-hub/global/skills/` 目录，以及当前机器已配置 agent 使用的全局 skills 目录。全局期望状态保存在 `~/.skill-hub/global-state.json`，实际 agent 目录通过 `.skill-hub-manifest.json` 标记 Skill-Hub 托管副本。

* **本地仓库**：skill-hub使用的本地配置目录，采用多仓库架构，支持管理多个Git仓库。默认仓库为归档仓库，所有通过`feedback`命令修改的技能都会归档到默认仓库。

* **多仓库架构**：skill-hub支持同时管理多个技能仓库，如个人仓库、社区仓库、官方仓库等。每个仓库可以独立启用、禁用、同步。


## 2. 命令语法约定

- `<参数>`：必需参数
- `[参数]`：可选参数
- `[--选项 值]`：可选选项
- `...`：可重复参数
- `|`：多个值中选择一个

## CLI JSON 调用合同

`--json` 是 `skill-hub` 客户端的全局选项，可放在子命令前后。它适用于客户端本地执行和服务桥接，不改变运行时选择，也不增加服务端模式。

- stdout 只输出一个完整 JSON 文档；首次登记的工作区创建提示、进度及诊断写入 stderr。调用方应分别捕获两个通道，直接解析全部 stdout。
- JSON 模式禁止读取 stdin，不隐含 `--force`、`--yes` 或覆盖授权。需要输入但未提供相应参数时返回结构化错误和非零退出码。
- 成功时保留各命令既有 JSON 字段，不增加统一外层。失败时增加 `command_error` 对象，包含 `code`、`message` 和可选 `details`；批量部分失败仍保留摘要和逐项结果，并以非零退出码结束。
- `apply --pattern` 和 `remove --pattern` 将各次执行结果放入 `results` 数组；未匹配时数组为空。`list`、`search` 使用 `items`、`total`，没有匹配时返回空数组。
- `init --json` 显式初始化全局仓库。首次 `register`、`use`、需要登记的 `import`、`create` 可无交互建立项目工作区；项目只读查询在未登记时返回 `PROJECT_NOT_FOUND`。CLI 不在 dry-run 或 archive-only 导入前创建项目工作区、写入项目状态。
- `create --json` 使用 `--description` 或默认描述，不覆盖验证失败的已有 `SKILL.md`。项目 `remove --json`、`repo remove --json`、`repo disable --json` 需要 `--yes`。
- `feedback`、`push` 保留已有 `--force` / `--dry-run` 约定；`git commit --json` 需要 `--message`。`upgrade --json` 保留原有检测行为，实际升级仍需要 `--yes`。
- `audit --format json` 同样遵守上述输出、非交互约定；使用 `--output` 时，文件内容按 `--format` 写入，stdout 返回 `output`、`format` 和 `report`。
- `--help`、`help` 和 `--version` 在 JSON 模式下返回结构化元数据；shell completion 不支持 JSON，返回参数错误。

示例：

```bash
skill-hub init --json
skill-hub register my-skill --json
skill-hub import .agents/skills --dry-run --json
skill-hub status --json
skill-hub remove my-skill --yes --json
```

参数错误示例（退出码非零）：

```json
{
  "command_error": {
    "code": "INVALID_INPUT",
    "message": "accepts 1 arg(s), received 0"
  }
}
```

普通文本模式保留既有交互行为；JSON 模式从不读取 stdin，因此不会将 EOF 或空输入视为确认。

## 3. 核心命令列表（按功能分组）

### 3.1. 环境配置
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `init` | 初始化本地仓库 | `skill-hub init [git_url]` |
| `project register` | 幂等登记空项目、查询原请求回执 | `skill-hub project register [path] --request-id <id> [--query] [--json]` |
| `skill-hubd` | 启动本地 daemon | `skill-hubd [--host <value>] [--port <value>] [--secret-key <value>]` |

### 3.2. 技能发现
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `list` | 列出可用技能 | `skill-hub list [--pattern <glob>...] [--verbose] [--repo <name>...]` |
| `search` | 搜索远程技能 | `skill-hub search <keyword> [--limit <number>]` |

### 3.3. 技能创建/移除/验证/使用
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `create` | 创建新技能模板 | `skill-hub create <id>` |
| `register` | 登记已有项目本地技能 | `skill-hub register <id> [--skip-validate]` |
| `import` | 批量登记、验证并可选归档已有技能 | `skill-hub import <skills-dir> [--fix-frontmatter] [--archive] [--archive-only] [--force] [--dry-run] [--fail-fast]` |
| `dedupe` | 检测嵌套项目中的重复技能副本 | `skill-hub dedupe <scope> [--canonical <dir>] [--strategy newest|canonical|fail-on-conflict] [--json]` |
| `sync-copies` | 从 canonical 目录同步同 ID 技能副本 | `skill-hub sync-copies --canonical <dir> [--scope <dir>] [--dry-run] [--no-backup] [--json]` |
| `lint` | 审计项目技能内容 | `skill-hub lint [scope] --paths [--project-root <dir>] [--fix] [--dry-run] [--no-backup] [--json]` |
| `audit` | 生成技能刷新审计报告 | `skill-hub audit [scope] [--output <file>] [--format markdown|json] [--canonical <dir>] [--project-root <dir>]` |
| `remove` | 移除项目技能 | `skill-hub remove <id>` 或 `skill-hub remove --pattern <glob>...` |
| `validate` | 验证技能合规性 | `skill-hub validate <id> [--fix] [--links] [--json]`、`skill-hub validate --all ...` 或 `skill-hub validate --pattern <glob>... ...` |
| `use` | 使用本地仓库里的指定技能 | `skill-hub use <id> | --pattern <glob>... [--global] [--repo <name>] [--dry-run] [--json] [--non-interactive]` |

补充：`use` / `remove` 支持 `--global`，用于本机全局 agent skills 目录；全局操作始终作用于已检测或已配置的 agent。`list` 仍只表示本地仓库中的可用 skill 清单，不存在项目或全局 scope。所有支持 `--pattern` 的命令也接受一个精确位置 ID；位置通配符（如 `list magic*` / `apply magic*`）仍会被拒绝，必须通过 `--pattern` 传入。


### 3.4. 技能状态
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `status` | 检查技能状态 | `skill-hub status [id] | [--pattern <glob>...] [--verbose] [--json] [--global]` |

补充：`status --global [--json]` 会检查全局期望状态与当前机器实际 agent skills 目录是否一致。文本输出与项目模式一致，按每个全局 skill 显示一行及版本；不同 agent 的状态不一致时，会在该行下显示 agent 明细。JSON 保留逐 agent 的原始状态项，便于自动化处理。

### 3.5. 项目工作区-本地仓库交互
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `apply` | 应用技能到项目 | `skill-hub apply [id] | [--pattern <glob>...] [--dry-run] [--force]` |
| `feedback` | 将项目工作区技能修改内容更新至到本地仓库 | `skill-hub feedback <id> ...`、`skill-hub feedback --all ...` 或 `skill-hub feedback --pattern <glob>... ...` |

补充：`apply --pattern <id> --global [--dry-run] [--force]` 会按 `global-state.json` 刷新全部已检测或已配置的本机 agent 全局 skills 目录。默认不覆盖没有 `.skill-hub-manifest.json` 的同名目录；`--force` 会先创建备份再覆盖。

### 3.6. 多仓库管理
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `repo add` | 添加新仓库 | `skill-hub repo add <name> <url> [--branch BRANCH] [--type TYPE] [--description DESC]` |
| `repo list` | 列出所有仓库 | `skill-hub repo list [--json]` |
| `repo remove` | 移除仓库 | `skill-hub repo remove <name>` |
| `repo enable` | 启用仓库 | `skill-hub repo enable <name>` |
| `repo disable` | 禁用仓库 | `skill-hub repo disable <name>` |
| `repo default` | 设置默认仓库 | `skill-hub repo default <name>` |
| `repo sync` | 同步仓库 | `skill-hub repo sync [name] [--all] [--json]` |
| `repo rebuild-index` | 重建仓库索引 | `skill-hub repo rebuild-index [name] [--json]` |

### 3.7. 本地状态维护
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `prune` | 清理 `state.json` 中失效的项目记录 | `skill-hub prune` |

### 3.8. 本地仓库同步
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `pull` | 从远程仓库拉取最新技能 | `skill-hub pull [--force] [--check] [--json]` |
| `push` | 推送本地更改到远程仓库 | `skill-hub push [--message MESSAGE] [--force] [--dry-run] [--json]` |
| `git` | Git仓库操作 | `skill-hub git <subcommand>` |
| `upgrade` | 检测并升级 skill-hub 二进制 | `skill-hub upgrade [--check] [--yes] [--version vX.Y.Z] [--dry-run] [--force] [--json]` |

## 4. 命令详细规范

### 4.0 skill-hubd - 本地 daemon

`skill-hubd` 是唯一的 HTTP API 与 Web UI 服务宿主，不是 `skill-hub` 的子命令。

```bash
skill-hubd
skill-hubd --port 6600
skill-hubd --host 127.0.0.1 --port 6600 --secret-key write-secret
```

- `--host` 默认 `127.0.0.1`，`--port` 默认 `5525`。
- `--secret-key` 只保护 `POST /api/v1/skill-repository/push`；未配置时该 API 返回 `READ_ONLY`，拉取、同步和项目本地操作不受影响。
- loopback 绑定会校验 Host、浏览器来源和安全响应头；CLI bridge 通过 `SKILL_HUB_SERVICE_URL` 与可选的 `SKILL_HUB_SERVICE_SECRET_KEY` 调用 daemon。
- Linux 可使用 `skill-hubd service install|start|stop|restart|status|uninstall` 管理当前用户的 systemd 单元；`install` 默认启用并启动服务，支持 `--host`、`--port`、`--secret-key` 和 `--no-start`，`status --json` 适合自动化检查。
- macOS 与 Windows 当前由对应平台的服务管理器负责后台启动、停止与重启；不再维护 `~/.skill-hub/services.json` 或 `serve` 子命令。

### 4.1 init - 初始化本地仓库

项目边界默认由已登记的项目状态确定。若当前目录本身存在 `.agents/skills/`，它会优先作为独立项目工作区，不会继承父目录的项目状态；首次执行项目命令时会提示创建该目录的状态记录。没有该标记的普通子目录继续继承最近的父项目状态。

**语法**: `skill-hub init [git_url]`

**参数**:
- `git_url` (可选): Git 仓库 URL，用于初始化技能仓库。如未提供，表示不使用远端仓库，只进行本地管理

**功能描述**:

创建 `~/.skill-hub` 目录结构，初始化全局配置。采用多仓库架构，默认创建名为"main"的本地仓库。

如提供了`git_url` 参数，则克隆远程技能仓库到默认仓库；否则创建空的本地仓库。`target` 不再作为项目默认值写入，也不再作为 CLI 参数入口。

完成本地仓库初始化后，会刷新技能索引。当前实现中：

- 每个仓库目录维护自己的 `registry.json`
- 根目录 `~/.skill-hub/registry.json` 继续保留为兼容索引
- `list` 和联想补全优先读取 repo 级索引，索引不可用时回退到仓库扫描

如果多次执行`init`，在出现冲突时，会提示用户选择是否覆盖。

**示例**:
```bash
# 不使用远端仓库，只进行本地管理
skill-hub init

# 使用自定义仓库初始化
skill-hub init https://github.com/example/skills-repo.git

```

### 4.2 project register - 空项目登记与请求回执

**登记**: `skill-hub project register [path] --request-id <id> [--json]`

**查询**: `skill-hub project register --query --request-id <id> [--json]`

- `path` 默认当前目录，须为已有目录。登记前需已执行 `skill-hub init`。
- `--request-id` 必填，由调用方在首次发送前生成并保存；允许 1–128 个字母、数字、点、下划线、冒号或连字符，首字符为字母或数字。
- 同一 `SKILL_HUB_HOME` 内，相同 ID 绑定同一个清理后的绝对项目路径。相同请求重放原始回执；ID 用于其他路径时返回 `REQUEST_CONFLICT`，不执行新登记。相对路径按每次调用的 cwd 解析，不做 symlink 路径合并。
- 登记精确指定目录，即使它在已登记项目之内，也不会被归属到父项目。创建 `.agents/skills` 和空技能状态，不创建占位技能，不扫描或导入目录中的技能。已有项目保留技能、变量、时间戳和历史字段，不清空状态。
- 新 ID 登记已存在项目返回 `result.project_created=false`。同 ID 重放则保留原值；重放不重新读取项目或创建目录，即使项目已删除。
- `--query` 不接受 `path`；只读取原请求回执及执行锁，不调用 `status`，不登记项目、不创建文件。可从其他目录查询，也不依赖当前配置或 `state.json` 仍然可用。

回执字段：

| 字段 | 含义 |
|------|------|
| `request_id`, `project_path` | 原请求身份及绑定的绝对路径 |
| `status` | `running`、`succeeded`、`failed` 或 `unknown` |
| `executed` | 终态时为 `true`，表示已执行登记处理；不保证发生变更。进行中或无法确认时为 `null` |
| `finished` | 终态为 `true`；进行中为 `false`；异常中断且没有最终回执时为 `null` |
| `started_at`, `completed_at` | 原请求开始、完成时间（UTC）；未确认完成时没有 `completed_at` |
| `result.registered`, `result.project_created` | 原操作确认的登记状态、是否新建项目状态记录 |
| `result.created_directories`, `result.skill_count` | 原操作创建的目录、当时已登记技能数；失败时保留已确认的部分效果 |
| `command_error` | 原操作失败原因或结果未知说明 |

登记命令失败、重放失败回执或结果未知时返回非零退出码。查询成功读取回执时退出码为 0，包括原请求失败或结果未知的回执；调用方通过 `status`、`finished` 和 `result` 判断原操作，而非使用查询进程退出码代替原操作结果。并发提交同 ID 时可返回 `running`，需继续查询。

回执按请求独立保存于 `${SKILL_HUB_HOME:-~/.skill-hub}/project-registrations/`，先持久化请求占用，再执行登记，最终回执持久化后才输出。跨进程请求锁防止同 ID 并发执行，项目状态写锁保护不同 ID 的登记。失败回执也不会因重试而被重新执行；修复失败原因后需要提交新的请求 ID。

进程异常中断、最终回执写入失败时，已占用的 ID 不自动重跑，查询返回 `unknown`。项目状态写入和回执写入不是跨文件事务，不能把中断后的结果推断为成功或未执行，需人工核对后决定后续操作。未找到持久化回执返回 `REQUEST_NOT_FOUND`，不等同于证明未执行。回执损坏或无读取权限时返回错误并禁止重复执行。不要删除回执及其锁文件；丢失回执、切换 Hub home 或沿用旧版本未记录的请求，都无法保证原请求查询与去重。

当前入口仅供 CLI，本地通过项目状态 Block 的类型化 EventHub port 执行和查询；不增加 daemon、HTTP API、Web UI 或其他命令的请求回执机制。

```bash
skill-hub init --json
skill-hub project register /path/to/project --request-id setup-001 --json
# 原回复丢失、agent 超时或需要恢复会话时，使用同一 Hub home 和原 ID
skill-hub project register --query --request-id setup-001 --json
```

### 4.3 list - 列出可用技能

**语法**: `skill-hub list [--pattern <glob>...] [--verbose] [--repo <repo-name>...]`

**选项**:
- `--pattern <glob>`: 使用 skill-hub 类 glob 语法匹配仓库技能 ID 字段，`*` 可跨 `/`。可重复使用以匹配多个 pattern。详见 §6。
- `--verbose`: 显示详细信息，包括技能描述、版本、标签和作者等。
- `--repo <repo-name>`: 按仓库名称过滤技能列表（可多次使用指定多个仓库）。

**功能描述**:

显示所有已启用仓库中的技能，支持按仓库过滤和按 ID pattern 过滤。默认显示简要列表，包含技能 ID、名称、版本和所属仓库。提供 pattern 时按 `ID` 字段做 glob 匹配；不带 pattern 时等同于 `**`（全部）。

当前实现补充：

- 当本地服务模式可用时，CLI 会优先通过服务桥接执行该命令
- 当服务不可用时，回退到本地扫描/索引逻辑

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

**多仓库说明**：默认显示所有已启用仓库中的技能。技能列表会标注技能所属的仓库名称。使用 `--repo` 选项可以指定只显示特定仓库中的技能，这在有大量技能时可以提高性能。

**示例**:
```bash
# 显示所有技能
skill-hub list

# 显示详细信息
skill-hub list --verbose

# 仅显示指定仓库中的技能
skill-hub list --repo skills-repo

# 显示多个仓库中的技能
skill-hub list --repo skills-repo --repo openclaw

# 组合使用过滤选项
skill-hub list --repo skills-repo --verbose

# 按 ID pattern 过滤
skill-hub list --pattern 'magic*'
skill-hub list --pattern '**'
# 多个 --pattern 取并集
skill-hub list --pattern 'magic*' --pattern 'git-*'
```

### 4.4 search - 搜索远程技能

**语法**: `skill-hub search <keyword> [--limit <number>]`

**参数**:
- `keyword` (必需): 搜索关键词。

**选项**:
- `--limit <number>`: 限制返回结果数量，默认 20。

**功能描述**:

搜索远端技能入口。命令行仍作为用户入口；当本地 `skill-hubd` 实例可用时，远端搜索交互会优先由服务实例统一承接，以复用本地托管的 `~/.skill-hub/` 上下文、远端访问策略和 Web/CLI 共用能力。

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

当前实现状态：

- CLI 会优先通过服务桥接执行
- 服务不可用时，保留本地直连回退以兼容离线或未启动服务的使用方式

**示例**:
```bash
# 搜索 git 相关技能
skill-hub search git

# 搜索 database 相关技能，限制 10 个结果
skill-hub search database --limit 10
```

### 4.5 create - 在项目工作区创建一个新技能

**语法**: `skill-hub create <id> [--description <text>] [--json]`

**参数**:
- `id` (必需): 新技能的标识符。
- `--description` (可选): 技能描述；JSON 模式未提供时使用默认描述，不读取 stdin。
- `--json` (可选): 按 CLI JSON 调用合同输出，已有文件验证失败时返回错误，不进入覆盖确认。

**功能描述**:

在项目工作区创建一个新技能。不写入项目 target 或工具适配范围元数据。生成的 `SKILL.md` 默认使用中文定义，包括适用场景、工作流程、输出要求和注意事项；命令、文件路径、API 名称等技术标识保留原始写法。

生成的技能目录结构包含：`SKILL.md`（核心定义）、`scripts/`（可执行脚本）、`references/`（参考资料）、`assets/`（静态资源）。`SKILL.md` 模板必须包含 `Formatter` 段，默认声明 Markdown/YAML 校验方式，并要求在新增脚本或代码时补充对应的可执行 formatter 命令。若该技能已存在（同名目录下已有 `SKILL.md`），则主动进行验证。验证通过时：若该技能已在本地仓库 state 中登记且与仓库内容一致，则不执行任何操作；否则刷新项目状态（state.json），便于技能登记与归档。验证不通过时提示是否重新创建。

创建的技能仅存在于项目本地，需要通过 `feedback` 命令同步到仓库。

`create` 命令将会刷新本地仓库的`state.json`，标记当前项目工作区在使用该技能

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区；新建项目工作区统一初始化 `.agents/skills`，不会根据 target 创建专属文件。


**示例**:
```bash
# 创建名为 my-logic 的技能
skill-hub create my-logic

```

### 4.5.1 register - 登记已有项目本地技能

**语法**: `skill-hub register <id> [--skip-validate] [--json]`

**参数**:
- `id` (必需): 已存在于 `.agents/skills/<id>/SKILL.md` 的技能标识符。
- `--skip-validate` (可选): 跳过 `SKILL.md` 验证，直接登记项目状态。
- `--json` (可选): 首次登记可无交互建立项目工作区，创建进度写入 stderr；stdout 返回 `project_path`、`skill_id`、`version`、`local_path`、`registered`。

**功能描述**:

将当前项目中已经存在的技能登记到 `state.json`。该命令不会创建、重写或覆盖 `SKILL.md`，用于替代过去依赖 `create` 登记已有技能的语义。

默认会先验证 `SKILL.md` 的基础 frontmatter。若 legacy 技能缺失 frontmatter，可先执行 `skill-hub validate --pattern <id> --fix`，或在明确知道内容可接受时使用 `--skip-validate` 完成登记。

当本地服务模式可用时，CLI 会通过服务桥接执行登记，服务端负责读写其托管的 `~/.skill-hub/state.json`。

**示例**:
```bash
skill-hub register existing-skill
skill-hub register existing-skill --json
skill-hub register legacy-skill --skip-validate
```

### 4.5.2 import - 批量导入已有技能

**语法**: `skill-hub import <skills-dir> [--fix-frontmatter] [--archive] [--archive-only] [--force] [--dry-run] [--fail-fast] [--json]`

**参数**:
- `skills-dir` (必需): 包含 `<id>/SKILL.md` 子目录的技能目录，典型值为 `.agents/skills`。
- `--fix-frontmatter` (可选): 导入前修复缺失或不完整的 frontmatter，修改前创建 `SKILL.md.bak.<timestamp>`。
- `--archive` (可选): 验证通过后归档到默认仓库，并在命令结束时刷新默认仓库索引。
- `--archive-only` (可选): 仅归档到默认仓库，不登记当前项目状态；必须与 `--archive` 一起使用。适用于发布包内置目录如 `agent-skills`。
- `--force` (可选): 批量流程不进行交互确认；不会自动删除源技能。
- `--dry-run` (可选): 只输出将要登记、修复、归档的动作。
- `--fail-fast` (可选): 遇到首个失败技能时停止，否则默认继续处理并在最后汇总失败项。
- `--json` (可选): 输出完整导入摘要；部分失败保留计数与 `failures`，增加 `command_error` 并以非零退出码结束。dry-run 或 archive-only 不由 CLI 初始化项目工作区。

**功能描述**:

扫描 `skills-dir/*/SKILL.md`，逐个执行 frontmatter 修复（可选）、验证、项目状态登记，并在指定 `--archive` 时归档到默认仓库。`--archive-only` 会跳过项目状态登记，只执行验证与归档。即使某个 skill 内容与默认仓库一致并计入 `unchanged`，也会刷新默认仓库 `registry.json`，避免目录内容和索引版本不一致。命令结束时输出固定摘要：

- `discovered`
- `registered`
- `valid`
- `archived`
- `unchanged`
- `failed`

失败项会列出 `id`、执行阶段、路径和错误信息，便于批量刷新和 CI 审计。

当本地服务模式可用时，CLI 会通过服务桥接执行导入和归档，确保批量操作使用服务端托管的默认仓库与状态文件。

**示例**:
```bash
skill-hub import .agents/skills --fix-frontmatter --archive --force
skill-hub import agent-skills --archive --archive-only --force
skill-hub import .agents/skills --dry-run --json
```

### 4.5.3 dedupe - 检测重复技能副本

**语法**: `skill-hub dedupe <scope> [--canonical <dir>] [--strategy newest|canonical|fail-on-conflict] [--json]`

**参数**:
- `scope` (必需): 扫描范围。
- `--canonical <dir>` (可选): canonical 技能目录，典型值为 `.agents/skills`。
- `--strategy` (可选): canonical source 选择策略，支持 `newest`、`canonical`、`fail-on-conflict`，默认 `newest`。
- `--json` (可选): 输出机器可读 JSON 报告。

**功能描述**:

扫描 `scope` 下所有 `.agents/skills/<id>/SKILL.md`，按技能 ID 分组，计算技能目录内容 hash，报告所有位置、hash、修改时间、是否 canonical、是否内容冲突以及选定的 canonical source。该命令只报告，不修改文件。

当 `--strategy fail-on-conflict` 且发现同 ID 内容差异时，命令返回失败，适合 CI 严格检查。

当本地服务模式可用时，CLI 会通过服务桥接执行重复检测，服务端按 CLI 当前工作目录解析后的 scope 进行扫描。

**示例**:
```bash
skill-hub dedupe . --canonical .agents/skills
skill-hub dedupe . --canonical .agents/skills --strategy fail-on-conflict
skill-hub dedupe . --canonical .agents/skills --json
```

### 4.5.4 sync-copies - 同步重复技能副本

**语法**: `skill-hub sync-copies --canonical <dir> [--scope <dir>] [--dry-run] [--no-backup] [--json]`

**参数**:
- `--canonical <dir>` (必需): canonical 技能目录，典型值为 `.agents/skills`。
- `--scope <dir>` (可选): 扫描范围，默认当前目录。
- `--dry-run` (可选): 只报告将同步的副本，不修改文件。
- `--no-backup` (可选): 同步前不创建备份。
- `--json` (可选): 输出机器可读 JSON 结果。

**功能描述**:

扫描 `scope` 下所有重复技能副本。对 canonical 目录中存在的同 ID 技能，将非 canonical 副本同步为 canonical 内容。默认会在修改每个副本前创建 `<skill-dir>.bak.<timestamp>` 目录备份。该命令不会删除任何技能副本目录。

当本地服务模式可用时，CLI 会通过服务桥接执行同步，确保由服务端统一访问托管工作区与文件系统。

**示例**:
```bash
skill-hub sync-copies --canonical .agents/skills --scope .
skill-hub sync-copies --canonical .agents/skills --scope . --dry-run
skill-hub sync-copies --canonical .agents/skills --scope . --json
```

### 4.5.5 lint - 审计项目技能内容

**语法**: `skill-hub lint [scope] --paths [--project-root <dir>] [--fix] [--dry-run] [--no-backup] [--json]`

**参数**:
- `scope` (可选): 扫描范围，默认当前目录。
- `--paths` (必需): 启用本机路径审计。当前 `lint` 仅支持该模式。
- `--project-root <dir>` (可选): 用于将本机绝对路径改写为相对路径的项目根目录。未指定时会尝试从 `.agents` 目录位置推断。
- `--fix` (可选): 将 `project-root` 内的 `/home/...`、`/Users/...`、`file://...` 改写为相对路径。
- `--dry-run` (可选): 只报告将改写的路径，不修改文件。
- `--no-backup` (可选): 修复前不创建备份文件。
- `--json` (可选): 输出机器可读 JSON 报告。

**功能描述**:

扫描 `scope` 下所有 `.agents/skills/<id>/` 技能目录中的 UTF-8 文本文件，报告 `file://`、`vscode://`、`/home/...`、`/Users/...` 等本机路径。`--fix` 只自动改写位于 `project-root` 内的普通本机路径和 `file://` 链接；`vscode://` 链接以及外部路径会标记为 `manual-review`。

默认修复前会创建 `<file>.bak.<timestamp>` 备份。CLI 会先把 `scope` 和 `project-root` 按调用方当前工作目录解析为绝对路径，再发送给服务端，因此在本地 `skill-hubd` 可用时仍能正确操作调用方项目。

**示例**:
```bash
skill-hub lint . --paths --project-root "$PWD"
skill-hub lint . --paths --project-root "$PWD" --fix
skill-hub lint . --paths --project-root "$PWD" --fix --dry-run --json
```

### 4.5.6 audit - 生成技能刷新审计报告

**语法**: `skill-hub audit [scope] [--output <file>] [--format markdown|json] [--canonical <dir>] [--project-root <dir>]`

**参数**:
- `scope` (可选): 技能扫描范围，默认 `.agents/skills`。

**选项**:
- `--output <file>`: 报告输出文件。未指定时输出到标准输出。
- `--format markdown|json`: 报告格式，默认 `markdown`。
- `--canonical <dir>`: 用于重复检测的 canonical 技能目录。
- `--project-root <dir>`: 用于路径和链接审计的项目根目录。

**功能描述**:

聚合项目技能刷新状态，报告目标技能数量、已登记数量、未登记技能、`validate --links` 结果、`status` 摘要、重复技能冲突、绝对路径命中、本地 Markdown 链接问题、默认仓库以及默认仓库是否存在未提交或未推送内容。

当本地 `skill-hubd` 可用时，CLI 会通过服务桥接执行审计聚合；CLI 会把 `scope`、`canonical`、`project-root` 解析为绝对路径后传给服务端。`--output` 文件由 CLI 在调用方侧写入，避免服务进程当前目录不同导致报告写错位置。

**示例**:
```bash
skill-hub audit .agents/skills --output .agents/skills-refresh-progress.md
skill-hub audit .agents/skills --format json
skill-hub audit . --canonical .agents/skills --project-root "$PWD"
```

### 4.6 remove - 移除项目技能

**语法**: `skill-hub remove <id> [--global] [--force] [--yes] [--json]` 或 `skill-hub remove --pattern <id-or-glob>... [--global] [--force] [--yes] [--json]`

**参数**:
- `id` (与 `--pattern` 二选一): 要移除的技能精确标识符。
- `--pattern` (与 `id` 二选一，可重复): 按已启用技能 ID 做 glob 匹配，批量移除匹配项。

**功能描述**:

从当前项目工作区中移除指定的技能：
1. 从 `state.json` 中移除技能标记
2. 物理删除项目工作区对应的文件/配置
3. 保留本地仓库中的源文件不受影响

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区；新建项目工作区统一初始化 `.agents/skills`，不会根据 target 创建专属文件。

**安全机制**: 如果检测到本地有未反馈的修改，会弹出警告并要求确认。

项目模式使用 `--json` 时必须提供 `--yes`，否则返回 `USER_INPUT_ERROR`，不会读取 stdin 或删除技能。`--yes` 只作为 JSON 项目移除的显式确认；普通文本模式保留原有交互。全局模式继续使用 `--force` 处理非托管目录冲突。

使用 `--global` 时，命令不要求当前目录是项目工作区，而是从 `~/.skill-hub/global-state.json` 中移除对应 agent 的全局期望状态，并删除带有 `.skill-hub-manifest.json` 的 agent 全局 skill 目录。默认不会删除没有 Skill-Hub manifest 的同名目录；`--force` 才允许强制删除冲突目录。

**示例**:
```bash
# 移除 git-expert 技能
skill-hub remove git-expert

# 从所有受管全局目录移除 git-expert
skill-hub remove git-expert --global

# 批量移除当前项目中 git-* 前缀的已启用技能
skill-hub remove --pattern 'git-*'
```

### 4.7 validate - 验证技能合规性

**语法**:
- `skill-hub validate --pattern <id-or-glob>... [--fix] [--links] [--project-root <dir>] [--check-remote] [--json]`
- `skill-hub validate --all [--fix] [--links] [--project-root <dir>] [--check-remote] [--json]`

**参数**:
- `--pattern` (必需，可重复): 单字面量无通配符按精确 ID 处理；含通配符或多个值时按当前项目登记的 `ID` 字段做 glob 匹配。详见 §6。

**选项**:
- `--fix`: 修复缺失或不完整的 `SKILL.md` frontmatter，并在修改前创建 `SKILL.md.bak.<timestamp>`。
- `--all`: 验证当前项目状态中登记的全部技能。
- `--links`: 检查 `SKILL.md` 和技能目录内 Markdown 文件中的链接。
- `--project-root <dir>`: 解析项目相对链接的根目录，默认使用当前项目目录或 `metadata.project_root`。
- `--check-remote`: 同时检查 HTTP/HTTPS 远端链接。默认忽略远端链接。
- `--json`: 输出机器可读 JSON 验证报告。

**功能描述**:

验证指定技能在项目工作区中的本地文件是否合规，包括检查 `SKILL.md` 的 YAML 语法、必需字段和基本文件结构。

`--fix` 会为 legacy `SKILL.md` 补齐最小 frontmatter：`name`、`description`、`metadata.version`、`metadata.author`。`name` 使用技能目录名，`description` 优先从正文首个有效段落推断，版本默认 `1.0.0`。修复只重写 frontmatter，不改写正文内容。

`--links` 会解析技能目录内 Markdown 文件中的本地链接。外部 HTTP/HTTPS 链接默认跳过；本地链接会按源文件目录、技能目录、项目根目录依次解析，任一目标存在即视为通过。缺失目标会以 `source_file`、`line`、`link`、`resolved_path` 输出，JSON 报告中汇总为 `link_issues`。

该命令与 `create` 配套，主要用于 `feedback` 前的本地校验，不负责校验仓库侧内容。当前本地 `skill-hubd` 实例可用时，CLI 会通过服务桥接执行 validate；CLI 会把 `project_path` 和 `project-root` 解析为绝对路径后传给服务端。

如果该技能未在`state.json`里项目工作区登记，则提示该技能非法

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区；新建项目工作区统一初始化 `.agents/skills`，不会根据 target 创建专属文件。

**示例**:
```bash
# 验证 my-logic 技能的合规性
skill-hub validate --pattern my-logic

# 修复 legacy frontmatter 后验证
skill-hub validate --pattern my-logic --fix

# 验证项目中全部已登记技能
skill-hub validate --all

# 检查本地 Markdown 链接并输出 JSON 报告
skill-hub validate --all --links --json

# 按 ID pattern 验证一组技能
skill-hub validate --pattern 'magic*' --links
```

### 4.8 use - 使用技能

**语法**: `skill-hub use <id> | --pattern <id-or-glob>... [--global] [--repo <name>] [--dry-run] [--json] [--non-interactive] [--var key=value]`

**参数**:
- `<id>` (二选一): 精确指定一个 skill ID；未找到时返回 `SKILL_NOT_FOUND`。
- `--pattern` (可重复): 按仓库技能 `ID` 字段做 glob 匹配。详见 §6。命中 0 个技能时以 `skipped` 结果静默通过。
- `--repo`: 精确选择技能来源仓库；多来源自动化调用应显式提供。
- `--dry-run`: 仅解析来源、版本和变量需求，不写入状态。
- `--json`: 输出统一的 `scope`、计数和逐项 `items` 结果，便于自动化处理。
- `--non-interactive`: 禁止候选和变量交互；来源有歧义时返回可操作错误，未传 `--var` 的变量使用默认值。
- `--var key=value`: 设置技能变量，可重复使用。

**功能描述**:

将技能标记为在当前项目中使用。此命令仅更新 `state.json` 中的技能记录，不直接修改项目文件，不写入 target。需要通过 `apply` 命令进行物理分发。

使用 `--global` 时，命令会将技能写入 `~/.skill-hub/global-state.json`，并以当前机器已检测到或已配置的全部 agent 为目标。全局命令不提供单 agent 筛选，确保全局期望状态和分发范围一致。

使用前应先通过 `list` 和/或 `search` 检查是否存在适合当前项目和任务的已管理技能。只有当候选技能明确匹配时才执行 `use`；如果没有合适技能，应继续当前任务或创建新技能，不应猜测无关技能 ID。

当前实现补充：

- 当本地服务模式可用时，CLI 会优先通过服务桥接执行
- 服务桥接路径支持候选技能选择、技能详情读取、变量输入后写入服务端状态

如果项目工作区里首次使用技能，也会同步在`state.json`里完成项目工作区信息刷新

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区；新建项目工作区统一初始化 `.agents/skills`，不会根据 target 创建专属文件。

**示例**:
```bash
# 先发现候选技能，再启用明确匹配的技能
skill-hub list
skill-hub search git
skill-hub use git-expert

# 启用为本机全局 skill
skill-hub use git-expert --global

# 按 ID pattern 一次性启用一组技能
skill-hub use --pattern 'magic*'

# CI 中预览并锁定来源
skill-hub use git-expert --repo team-skills --dry-run --json --non-interactive
```

### 4.9 status - 检查技能状态

**语法**: `skill-hub status [id] | [--pattern <id-or-glob>...] [--verbose] [--json] [--global]`

**参数**:
- `--pattern` (可选，可重复): 不传时检查全部；传 `**` 时匹配全部已启用技能；单字面量无通配符时按精确 ID 检查；含通配符或多个值时按当前项目或全局状态中的 `ID` 字段做 glob 匹配。详见 §6。

**选项**:
- `--verbose`: 显示详细差异信息。
- `--json`: 以机器可读 JSON 输出状态摘要，字段包含 `scope`、`skill_id`、`status`、`reason`、`legacy_status`、版本、路径与来源仓库等；`legacy_status` 仅用于旧调用方迁移。
- `--global`: 检查本机全局期望状态与实际 agent skills 目录是否一致；文本表格按 skill 聚合显示版本与 agent，状态不一致时显示 agent 明细。JSON 保留逐 agent 的原始状态项。

**功能描述**:

对比项目本地工作区与技能仓库源文件的差异，显示技能状态。对比范围包括技能目录下全部文件（含 `SKILL.md` 及 `scripts/`、`references/`、`assets/` 等子目录）。项目与全局共用下列规范状态：
- `synced`: 目标副本与来源一致
- `modified`: 目标副本被本地修改
- `outdated`: 来源已更新，目标需要刷新
- `diverged`: 来源已更新且目标也有修改；`feedback` 会阻断该状态，需先 `apply --pattern <id>` 刷新并手工迁移必要修改
- `missing`: 已启用但目标副本不存在
- `conflict`: 存在同名但不可安全接管的目标
- `orphaned`: 存在 Skill-Hub 托管副本，但状态记录已不存在
- `unavailable`: 目标环境或来源不可用

全局聚合表格还可能显示 `mixed`，表示同一 skill 在不同 agent 上的状态不同；逐 agent JSON 项始终使用实际规范状态。`legacy_status` 保留旧名称（如 `Synced`、`ok`、`stale`、`error`），仅供迁移期间的兼容处理。

使用 `--global` 时，状态仍按上述规范状态表达：`synced` 表示 `global-state.json`、来源仓库与 agent 目录一致，`missing` 表示尚未写入 agent 目录，`outdated` 表示来源已变化，`unavailable` 表示 agent 目录或来源不可用。其他状态含义不变。

当前实现补充：

- 当本地服务模式可用时，CLI 会优先通过服务桥接执行
- 当显式指定的全局 skill 尚未启用时，返回 `SKILL_NOT_FOUND`，不再输出空状态摘要

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区；新建项目工作区统一初始化 `.agents/skills`，不会根据 target 创建专属文件。

**示例**:
```bash
# 检查所有技能状态
skill-hub status

# 检查特定技能状态
skill-hub status git-expert

# 显示详细差异
skill-hub status --verbose

# 输出机器可读状态
skill-hub status --json

# 检查本机全局技能一致性
skill-hub status --global
skill-hub status git-expert --global
```

### 4.10 apply - 应用技能到项目工作区

**语法**: `skill-hub apply [id] | [--pattern <id-or-glob>...] [--dry-run] [--force] [--global]`

**参数**:
- `--pattern` (可选，可重复): 不传时刷新全部已启用技能；传 `**` 时匹配全部已启用技能；单字面量无通配符时按精确 ID 刷新；含通配符或多个值时按当前项目或全局状态中的 `ID` 字段做 glob 匹配，逐个处理并打印成功/失败汇总。详见 §6。

**选项**:
- `--dry-run`: 演习模式，仅显示将要执行的变更，不实际修改文件。
- `--force`: 强制应用，即使检测到冲突也继续执行。
- `--global`: 按 `global-state.json` 刷新本机 agent 全局 skills 目录。

**功能描述**:

根据 `state.json` 中的启用记录，将技能物理分发到标准项目工作区 `.agents/skills/<id>/`。不传 `--pattern` 时刷新当前项目所有已启用技能；传入 `--pattern <id>` 时只刷新该已启用技能，可用于把 `Outdated` 状态的项目本地 skill 更新到来源仓库版本。历史 target 不参与分发路径选择。

使用 `--global` 时，命令会从来源仓库刷新 `~/.skill-hub/global/skills/<id>/` 镜像，再同步到目标 agent 全局 skills 目录。写入 agent 目录时会生成 `.skill-hub-manifest.json`；同名目录存在但没有 manifest 时默认报告 `conflict`，不会覆盖，除非用户显式提供 `--force`。`--force` 覆盖前会创建 `*.skill-hub-backup.<timestamp>` 备份。

当前实现补充：

- 当本地服务模式可用时，CLI 会优先通过服务桥接执行
- 服务端负责实际适配器调用和项目文件分发
- 项目级 `apply --pattern <id>` 会在刷新成功后将项目状态中的版本、状态和来源目录指纹更新为仓库版本、`Synced` 与 `applied_hash`
- 项目级 `apply --pattern <id>` 未匹配已启用技能时静默通过；底层单技能刷新显式指定未启用技能仍返回 `SKILL_NOT_FOUND`
- 当显式指定的全局 skill 尚未启用时，返回 `SKILL_NOT_FOUND`，不会静默跳过刷新

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区；新建项目工作区统一初始化 `.agents/skills`，不会根据 target 创建专属文件。

**示例**:
```bash
# 应用启用技能到标准项目工作区
skill-hub apply

# 只刷新一个已启用项目技能，常用于修复 Outdated 状态
skill-hub apply --pattern git-expert

# 演习模式查看将要进行的变更
skill-hub apply --dry-run

# 预览和刷新本机全局技能
skill-hub apply --global --dry-run
skill-hub apply --global
skill-hub apply --pattern git-expert --global

# 按 ID pattern 批量应用
skill-hub apply --pattern 'magic*'
```

### 4.11 feedback - 将项目工作区里技能修改更新至本地仓库

**语法**:
- `skill-hub feedback --pattern <id-or-glob>... [--dry-run] [--force] [--json]`
- `skill-hub feedback --all [--dry-run] [--force] [--json]`

**参数**:
- `--pattern` (必需，可重复): 单字面量无通配符按精确 ID 处理；含通配符或多个值时按当前项目登记的 `ID` 字段做 glob 匹配，逐个处理并打印成功/失败汇总。详见 §6。

**选项**:
- `--dry-run`: 演习模式，仅显示将要同步的差异。
- `--force`: 跳过交互确认并执行更新；不会绕过版本新鲜度或核心信息完整性保护。
- `--all`: 反馈当前项目状态中登记的全部技能。
- `--json`: 输出机器可读反馈摘要，包含 `total`、`applied`、`planned`、`skipped`、`failed` 和每个技能的预览/结果。实际写入时需要配合 `--force`；也可以配合 `--dry-run` 仅预览。

**功能描述**:

将项目工作区的指定技能修改同步回本地技能仓库。同步范围包括该技能目录下全部文件（含 `SKILL.md` 及 `scripts/`、`references/`、`assets/` 等子目录）。此命令会：
1. 提取项目工作区该技能目录下所有文件内容
2. 与本地仓库源文件对比，显示差异
3. 经用户确认后更新本地仓库文件
4. 更新 `registry.json` 中的版本/哈希信息

当前实现补充：

- 当本地服务模式可用时，CLI 会优先通过服务桥接执行
- 服务桥接路径会先执行反馈预览，再根据参数或确认执行实际归档
- 服务端负责版本推进、归档和索引刷新
- 当项目工作区 skill 版本低于默认仓库版本时，`feedback` 默认拒绝归档并提示先执行 `skill-hub apply --pattern <id>`，避免旧版本工作区副本覆盖仓库高版本内容。`--force` 不会绕过该版本新鲜度防护。
- 当更新会删除默认仓库副本已有的 frontmatter 字段、一级或二级工作章节，或 `references/`、`scripts/`、`assets/`、`agents/` 中的资源时，`feedback` 和 `import --archive` 会拒绝归档，避免已知核心信息被不完整副本覆盖。`--force` 同样不会绕过此保护。
- 当项目工作区与默认仓库同版本但内容不同，仍沿用现有逻辑自动 bump patch 后归档。
- `--all` 会按当前项目状态中登记的技能逐个执行反馈。为避免误批量归档，实际写入时必须显式提供 `--force`；也可以使用 `--dry-run` 预览。`--json` 模式不会进入交互确认，实际写入同样必须显式提供 `--force`。

**多仓库说明**：技能会被归档到默认仓库（通过 `skill-hub repo default` 命令设置）。如果技能在默认仓库中不存在则新增；存在时仅在不丢失既有核心信息的前提下更新。

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区；新建项目工作区统一初始化 `.agents/skills`，不会根据 target 创建专属文件。


**示例**:
```bash
# 反馈 my-logic 技能的修改
skill-hub feedback --pattern my-logic

# 演习模式查看将要同步的差异
skill-hub feedback --pattern git-expert --dry-run

# 批量反馈并输出机器可读摘要
skill-hub feedback --all --force --json

# 按 ID pattern 批量反馈（需 --force 或 --dry-run）
skill-hub feedback --pattern 'magic*' --dry-run
```

### 4.12 prune - 清理失效项目状态

**语法**: `skill-hub prune`

**功能描述**:

扫描 `~/.skill-hub/state.json` 中记录的项目路径，清理已经失效的项目项。以下记录会被移除：

- 项目路径为空
- 项目路径不存在
- 项目路径存在但不是目录

适用场景：

- 项目目录被移动到新位置后，旧路径残留在 `state.json`
- 项目目录被删除后，`status` 等命令仍命中旧项目记录

当前实现补充：

- 该命令为一级命令，不依赖当前工作目录是否已初始化
- 该命令只清理失效项目记录，不会自动补建新的项目路径记录
- 该命令当前仍主要在本地执行，不走服务桥接

**示例**:
```bash
# 清理失效项目记录
skill-hub prune
```

### 4.13 pull - 从远程仓库拉取最新技能

**语法**: `skill-hub pull [--force] [--check] [--json]`

**选项**:
- `--force`: 当远端有更新且默认仓库存在未提交更改时，先自动 `git stash push -u` 暂存本地更改，再执行拉取。
- `--check`: 检查模式，仅显示可用的更新，不实际执行拉取操作。
- `--json`: 输出机器可读拉取摘要。

**功能描述**:

从默认仓库对应的远程拉取最新更改到本地仓库（`~/.skill-hub/repositories/<default>/`），并更新技能索引。此命令仅同步仓库层，不涉及项目工作目录的更新。

`--check` 会执行不修改工作树的远端检查：有远端时通过 fetch 更新远端引用并比较 `main` 分支提交；无远端时返回 `no_remote`。JSON 输出中的 `status` 可能为 `no_remote`、`up_to_date`、`updates_available`、`ahead` 或 `divergent`，并包含 `ahead` / `behind` 计数。若默认仓库存在未提交更改但远端已是最新，普通 `pull` 不应失败；若远端有更新，则要求用户先提交/暂存，或使用 `--force` 触发自动 stash。

`git sync`、`git pull` 以及 `repo sync` 也遵循同样的索引刷新原则：仓库内容变化后会重建对应 repo 的 `registry.json`，默认仓库同时会刷新根目录兼容索引。

**多仓库说明**：此命令仅处理默认仓库。如需同步指定仓库或所有启用仓库，请使用 `skill-hub repo sync`。

本地执行时该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化。本地 `skill-hubd` 可用时，CLI 会优先通过服务桥接执行默认仓库同步，以兼容客户端 HOME 中没有本地 skill-hub 配置的场景。


**关键行为**:
1. `--check` 只检查默认仓库远端引用，不修改工作树或项目目录
2. 对默认仓库执行 `git pull` 从远程仓库获取最新技能
3. 更新本地技能注册表（作为只读缓存）
4. 不修改项目状态或项目工作目录中的文件
5. 不更新项目的 `LastSync` 时间戳

**后续操作**:
- 使用 `skill-hub status` 检查项目技能状态
- 使用 `skill-hub apply` 将仓库更新应用到项目工作目录

**示例**:
```bash
# 从远程仓库拉取最新技能
skill-hub pull

# 检查可用更新但不实际执行拉取
skill-hub pull --check

# 检查模式，输出 JSON 摘要供 CI 或脚本读取
skill-hub pull --check --json
```

### 4.14 push - 推送本地更改到远程仓库

**语法**: `skill-hub push [--message MESSAGE] [--force] [--dry-run] [--json]`

**选项**:
- `--message MESSAGE`, `-m MESSAGE`: 提交消息。如未提供，根据待推送文件自动生成，例如 `更新技能: skill-a` 或 `更新技能: skill-a, skill-b`。
- `--force`: 强制推送，跳过确认检查。
- `--dry-run`: 演习模式，仅显示将要推送的更改，不实际执行。
- `--json`: 输出机器可读推送摘要；实际写入推送时必须同时使用 `--force`，或使用 `--dry-run` 只预览。

**功能描述**:

自动检测默认仓库中的未提交更改或已提交但尚未推送的本地提交，然后推送到对应远程仓库。此命令用于完成 `feedback -> push` 的归档闭环。

**多仓库说明**：默认推送到默认仓库（归档仓库）。可使用 `skill-hub repo default` 命令设置默认仓库。

本地执行时该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化。本地 `skill-hubd` 实例可用时，CLI 会优先通过服务桥接获取默认仓库状态并执行推送，以兼容客户端 HOME 中没有本地 skill-hub 配置的场景。

服务 API 中默认仓库推送为受保护写操作：

- `GET /api/v1/skill-repository/push-preview` 返回默认仓库、远端 URL、待推送文件列表、建议提交消息、原始状态和 `ahead` / `behind` 计数；`has_pending_push` 同时覆盖未提交更改和已提交待推送内容。
- `POST /api/v1/skill-repository/push` 必须携带 `confirm: true`。
- 请求可携带 `expected_changed_files`；服务端执行前会重新检查默认仓库状态，若文件列表已变化则拒绝推送。
- 无待推送变更时服务端拒绝推送。
- 未配置 `skill-hubd --secret-key` 时该推送 API 返回 `READ_ONLY`；仓库拉取/同步类 API 不受该密钥限制。

**关键行为**:
1. 检查默认仓库是否有未提交的更改（Modified、Untracked、Deleted 文件）或本地领先远端的提交
2. 如有未提交更改，自动提交（优先使用指定消息；未指定时根据变更的 skill ID 自动生成消息）
3. 将已有的本地提交推送到远程仓库
4. 如无更改可推送，提示"没有要推送的更改"

**多仓库注意**: 此命令仅操作默认仓库（归档仓库）到远程仓库的同步。如需操作其他仓库，请使用 `skill-hub git` 命令。

**示例**:
```bash
# 推送所有本地更改，使用默认提交消息
skill-hub push

# 使用自定义提交消息推送
skill-hub push --message "修复技能描述"

# 演习模式，仅查看将要推送的更改
skill-hub push --dry-run

# 演习模式，输出 JSON 摘要供 CI 或脚本读取
skill-hub push --dry-run --json
```

### 4.15 upgrade - 检测并升级 skill-hub

**语法**: `skill-hub upgrade [--check] [--yes] [--version vX.Y.Z] [--dry-run] [--force] [--json] [--skip-agent-skills]`

**选项**:
- `--check`: 仅检测 GitHub Releases 是否存在新版本，不修改文件。
- `--yes`: 跳过交互确认，检测到可升级版本后自动执行升级。
- `--version vX.Y.Z`: 指定目标 Release 版本；未指定时使用 latest release。
- `--dry-run`: 生成升级计划，不下载或替换二进制。
- `--force`: 允许重新安装当前版本或安装低于当前版本的指定版本。
- `--json`: 输出机器可读检测或执行结果。
- `--skip-agent-skills`: 升级二进制后跳过 release 内置 `skill-hub-*` workflow skills 同步。

**功能描述**:

检测 `muidea/skill-hub` GitHub Releases，按当前 `GOOS/GOARCH` 下载对应 `.tar.gz` 发布包和 `.sha256` 文件。命令会校验发布包 sha256，解压后预执行新版 `skill-hub` 与 `skill-hubd` 的版本检查，再通过可回滚事务替换同目录的两个二进制。

Linux 与 macOS 支持自动替换当前运行中的二进制。Windows 当前不做运行中自动替换，命令会返回明确提示，用户需继续使用安装脚本或手动下载 Release 包。

升级成功后，默认同步 release 内置 `skill-hub-*` workflow skills 到工具无关目录，并安装同版本的 `skill-hub` 与 `skill-hubd`。运行中的 daemon 由系统服务管理器重启。该命令不依赖项目初始化，也不修改技能仓库、项目工作区或全局托管 skills 目录。

**示例**:
```bash
# 只检测是否有新版本
skill-hub upgrade --check

# 自动升级到 latest release
skill-hub upgrade --yes

# 预览升级计划并输出 JSON
skill-hub upgrade --dry-run --json

# 安装指定版本
skill-hub upgrade --version v0.8.1 --yes
```

### 4.16 git - Git仓库操作

**语法**: `skill-hub git <subcommand> [options]`

**子命令**:
- `clone [url]` - 克隆远程技能仓库到本地
- `sync` - 同步技能仓库（拉取最新更改，支持 `--json`）
- `status` - 查看仓库状态（未提交的更改等，支持 `--json`）
- `commit` - 提交更改到本地仓库（交互式输入消息）
- `push` - 推送本地提交到远程仓库（检查未提交更改）
- `pull` - 从远程仓库拉取更新（同 `sync`，支持 `--json`）
- `remote [url]` - 设置或更新远程仓库URL

**子命令语法**:
- `skill-hub git clone [url]` - 克隆远程技能仓库到本地
- `skill-hub git sync [--json]` - 同步技能仓库（拉取最新更改）
- `skill-hub git status [--json]` - 查看仓库状态（未提交的更改等）
- `skill-hub git commit [--message <text>] [--json]` - 提交并推送更改；JSON 模式必须显式提供 `--message`
- `skill-hub git push [--json]` - 推送本地提交；JSON 模式遇到未提交更改时返回输入错误，不交互决定是否提交
- `skill-hub git pull [--json]` - 从远程仓库拉取更新（同 `sync`）
- `skill-hub git remote [url]` - 设置或更新远程仓库URL

**功能描述**:
提供底层Git仓库操作接口，适用于需要精细控制Git工作流的用户。这些命令默认操作本地技能仓库（`~/.skill-hub/repositories/<name>/`），与高级命令（如 `repo sync`、`push`）功能重叠但提供更多控制选项。其中 `git status --json`、`git sync --json`、`git pull --json` 在本地 `skill-hubd` 实例可用时会优先通过服务桥接读取或同步默认仓库，以便客户端 HOME 无本地配置时仍可用于脚本检查。

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

# 查看仓库状态，输出 JSON
skill-hub git status --json

# 同步技能仓库，输出 JSON
skill-hub git sync --json

# 拉取更新，输出 JSON
skill-hub git pull --json

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
- `--dry-run`: 演习模式（支持命令: `use`, `apply`, `feedback`, `import`, `sync-copies`, `lint`, `push`, `upgrade`）
- `--force`: 强制模式（支持命令: `apply`, `feedback`, `import`, `pull`, `push`, `upgrade`）
- `--json`: CLI 全局机器输出选项，适用于业务命令和帮助、版本元数据；遵循上文 CLI JSON 调用合同。shell completion 不支持该选项。

## 6. Pattern 语法与批量操作

下列命令通过 `--pattern` 标志接受 skill-hub 类 glob pattern，匹配**技能 ID 字段**，用于按 ID 前缀或通配符筛选。`list` / `use` 匹配仓库技能集合；`status` / `apply` 匹配当前项目或全局状态中的已启用技能；`feedback` / `validate` 匹配当前项目登记的技能：

- `list --pattern <glob>...`
- `use --pattern <glob>...`
- `status --pattern <glob>...`
- `apply --pattern <glob>...`
- `feedback --pattern <glob>...`
- `validate --pattern <glob>...`

**为什么通配符要用 `--pattern`**：`pattern` 形如 `magic*` 在大多数 shell 中会被提前展开为 cwd 下的文件名，再交给 CLI 进程，破坏 glob 语义。所有 pattern 命令都接受一个精确位置 ID，例如 `status git-expert`；`list magic*` / `use foo*` / `apply magic*` 等位置通配符会被拒绝并提示改用 `--pattern`。带通配符的 flag 值仍应引用，例如 `--pattern 'magic*'`；如果不引用，shell 仍可能在 skill-hub 启动前展开它。

**语法**：

| 符号 | 含义 | 示例 |
|------|------|------|
| `*`  | 匹配零或多个任意字符，可跨 `/` | `--pattern 'magic*'` 匹配 `magic-helper`、`magic-pack`、`magic-team/magic-skill` |
| `?`  | 匹配恰好一个任意字符 | `--pattern 'git-?'` 匹配 `git-x` |
| `[abc]` | 字符类（Go 风格，否定用 `[^abc]`） | `--pattern '[mg]it-*'` 匹配 `git-*` 或 `mit-*` |
| `**` | 匹配全部技能 | `--pattern '**'` 等同不传 `--pattern` |

**保留规则**：

- 所有 pattern 命令的单字面量 ID 都可以写成位置参数，例如 `use git-expert`、`status git-expert`、`apply git-expert`。位置参数不能与 `--pattern` 同时使用，且不接受通配符。
- 单独的 `*` 会被拒绝（`ErrInvalidInput`），避免歧义。如需匹配全部请使用 `**`。
- `--pattern` 的空字符串值（`--pattern ''`）会被拒绝（`ErrInvalidInput`）。
- 语法错误（未闭合的 `[` 等）会立即返回 `ErrInvalidInput`，并在消息中标明出错 pattern。
- 同一命令上可重复使用 `--pattern`，多个值之间取**并集**：`--pattern 'magic*' --pattern 'git-*'` 命中 magic 与 git 两个系列。

**批量行为**：

- `use --pattern <glob>...` 命中 0 个技能时**静默通过**，命中 1 个直接启用，命中多个时按现有 `chooseSkillCandidate` 交互选择。
- `apply` / `feedback` / `validate` 命中多个时**逐个处理，单个失败不影响后续**，最后打印成功/失败汇总。
- `list` / `status` 只展示命中的技能；输出仍走原有渲染器，JSON 模式同样生效。

**示例**：

```bash
# 列举所有 magic 前缀的技能
skill-hub list --pattern 'magic*'

# 一次性启用所有 git 相关技能
skill-hub use --pattern 'git-*'

# 多个 --pattern 取并集
skill-hub list --pattern 'magic*' --pattern 'git-*'

# 检查所有 magic 系列技能的项目状态
skill-hub status --pattern 'magic*' --json

# 批量应用所有匹配 skill，并打印汇总
skill-hub apply --pattern 'magic*'

# 批量反馈（需 --force 或 --dry-run）
skill-hub feedback --pattern 'magic*' --dry-run
```

## 7. 使用示例

### 7.1 完整工作流示例

```bash
# 1. 初始化环境
skill-hub init

# 2. 拉取远端技能仓库，以刷新本地仓库
skill-hub pull

# 3. 查看可用技能
skill-hub list

# 4. 启用技能
skill-hub use git-expert

# 5. 应用技能到项目
skill-hub apply

# 6. 本地修改后检查状态
skill-hub status

# 8. 反馈修改到仓库
skill-hub feedback --pattern git-expert

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
skill-hub validate --pattern my-new-skill

# 4. 反馈到仓库
skill-hub feedback --pattern my-new-skill
```

## 8. 多仓库管理命令详细规范

### 8.1 repo add - 添加新仓库

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

### 8.2 repo list - 列出所有仓库

**语法**: `skill-hub repo list [--json]`

**选项**:
- `--json`: 输出机器可读仓库列表，包含 `default_repo` 和 `items`。

**功能描述**:
列出所有已配置的Git仓库，显示仓库状态和基本信息，包括：
- 仓库名称
- 类型（user/community/official）
- 是否启用
- 是否为默认仓库（归档仓库）
- 描述信息

当前实现补充：

- 当本地服务模式可用时，`repo add/list/remove/sync/enable/disable/default` 都会优先通过服务桥接执行

**示例**:
```bash
# 列出所有仓库
skill-hub repo list

# 输出 JSON 仓库列表
skill-hub repo list --json
```

### 8.3 repo remove - 移除仓库

**语法**: `skill-hub repo remove <name> [--yes] [--json]`

**参数**:
- `name` (必需): 要移除的仓库名称
- `--yes` (JSON 模式必需): 显式确认本次操作，缺少时返回 `USER_INPUT_ERROR`。
- `--json` (可选): 输出结构化结果；普通文本模式仍保留交互确认。

**功能描述**:
从配置中移除指定的仓库。如果仓库是默认仓库，需要先设置其他仓库为默认仓库才能移除。

**安全机制**: 会提示用户确认，防止误操作。

**示例**:
```bash
# 移除名为test的仓库
skill-hub repo remove test
```

### 8.4 repo enable - 启用仓库

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

### 8.5 repo disable - 禁用仓库

**语法**: `skill-hub repo disable <name> [--yes] [--json]`

**参数**:
- `name` (必需): 要禁用的仓库名称
- `--yes` (JSON 模式必需): 显式确认本次操作，缺少时返回 `USER_INPUT_ERROR`。
- `--json` (可选): 输出结构化结果；普通文本模式仍保留交互确认。

**功能描述**:
禁用指定的仓库。禁用后，该仓库中的技能不会出现在 `skill-hub list` 命令的结果中，但仓库配置仍然保留。

**示例**:
```bash
# 禁用名为test的仓库
skill-hub repo disable test
```

### 8.6 repo default - 设置默认仓库

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

### 8.7 repo sync - 同步仓库

**语法**: `skill-hub repo sync [name] [--all] [--json]`

**参数**:
- `name` (可选): 要同步的仓库名称。如未提供，同步所有启用的仓库

**选项**:
- `--all`: 同步所有仓库，包括已禁用仓库。默认只同步启用仓库。
- `--json`: 输出机器可读同步摘要，包含 `total`、`synced`、`skipped`、`failed` 和每个仓库的同步状态。

**功能描述**:
同步指定仓库或所有启用的仓库。对于远程仓库，会先 fetch 并检查分支关系，再仅以 fast-forward 方式执行 `git pull` 获取最新内容；该命令绝不推送本地提交，也不自动创建 merge commit。
同步后会自动刷新技能索引。若本地仅领先远端，则返回 `local_ahead` 并保留待推送提交；若本地与远端分叉，返回 `divergent`；若远端有更新但工作区存在未提交更改，返回 `blocked_dirty`，要求先提交或暂存。

**示例**:
```bash
# 同步所有启用的仓库
skill-hub repo sync

# 同步特定仓库
skill-hub repo sync community

# 输出 JSON 同步摘要
skill-hub repo sync --json
```

### 8.8 repo rebuild-index - 重建仓库索引

**语法**: `skill-hub repo rebuild-index [name] [--json]`

**参数**:
- `name` (可选): 要重建索引的仓库名称。如未提供，重建默认仓库。

**选项**:
- `--json`: 输出机器可读结果，包含 `repo_name` 和 `status`。

**功能描述**:
扫描指定仓库的 `skills` 目录并重建该仓库的 `registry.json`。当重建默认仓库时，会同步刷新 `~/.skill-hub/registry.json` 兼容索引。

该命令用于修复索引异常，不替代 skill 归档入口。标准项目工作区改动仍应使用 `feedback`，发布包内置或批量 skill 目录仍应使用 `import --archive --archive-only`，不要手工复制到 `~/.skill-hub/repositories/.../skills`。

**示例**:
```bash
skill-hub repo rebuild-index
skill-hub repo rebuild-index skills-repo --json
```

## 9. 服务模式与 CLI 交互说明

当前 `skill-hub` 已支持双运行模式：

- CLI 模式
- Service 模式

服务模式启动后：

- Web UI 可通过浏览器访问
- CLI 中的部分命令会优先通过服务桥接执行
- 远端推送桥接到配置了 `secretKey` 的服务时，调用方需设置 `SKILL_HUB_SERVICE_SECRET_KEY=<secretKey>`；未配置 `secretKey` 的服务允许读取、项目本地操作和仓库拉取/同步，但禁止默认仓库 push 到远端
- 当前阶段 Web UI 管理端不提供密钥输入或浏览器会话保存能力，远端推送权限只通过后端配置和调用方显式携带 header / CLI bridge 环境变量获得

当前已桥接的命令：

- `repo add/list/remove/sync/enable/disable/default`
- `list`
- `search`
- `status`
- `use`
- `register`
- `import`
- `dedupe`
- `sync-copies`
- `lint --paths`
- `validate`
- `audit`
- `apply`
- `remove --global`
- `feedback`
- `pull`
- `push`
- `git status --json`
- `git sync --json`
- `git pull --json`

当前未桥接、仍主要本地执行的命令：

- `init`
- `create`
- `project register`（含 `--query` 原请求查询）
- `remove` 的项目模式
- `prune`
- `upgrade`
- `git` 的交互式或非 JSON 子命令

## 10. 更新记录

| 版本 | 日期 | 更新说明 |
|------|------|----------|
| 1.0 | 2026-02-08 | 初始版本，统一所有设计文档中的命令定义 |
| 1.1 | 2026-02-09 | 核对各个命令描述，增加依赖信息说明 |
| 1.2 | 2026-02-17 | 添加多仓库管理命令，更新init命令描述以支持多仓库架构 |
| 1.3 | 2026-03-14 | 补充 `serve` 命令、服务模式桥接行为，以及 `repo/list/status/use/apply/feedback` 的当前实现状态 |
| 1.4 | 2026-03-22 | 同步项目工作区/默认仓库/服务化边界：`search` 目标服务化，`validate` 仅本地校验，`pull/push` 收口为默认仓库语义 |
| 1.5 | 2026-03-25 | 增加一级命令 `prune`，用于清理 `state.json` 中因项目目录移动或删除导致的失效项目记录 |
| 1.6 | 2026-04-18 | 增加 `register`、`import`、`dedupe`、`sync-copies`、`validate --fix/--all` 与 `status --json`，支撑批量技能刷新、重复副本治理和自动化审计 |
| 1.7 | 2026-04-18 | 增加 `lint --paths` 路径可移植性审计与服务桥接说明 |
| 1.8 | 2026-04-18 | 增加 `validate --links/--json` 与 validate 服务桥接说明 |
| 1.9 | 2026-04-18 | 增加 `audit` Markdown/JSON 审计报告与服务桥接说明 |
| 1.10 | 2026-04-18 | 增加 `feedback --all` 与 `feedback --json` 批量反馈说明 |
| 1.11 | 2026-04-18 | 增加 `repo list --json` 机器可读输出 |
| 1.12 | 2026-04-18 | 增加 `repo sync --json` 同步摘要输出 |
| 1.13 | 2026-04-18 | 增加 `push --json` 推送摘要输出，并补齐默认仓库 push/status 服务桥接 |
| 1.14 | 2026-04-18 | 增加 `pull --json` 拉取摘要输出，并补齐默认仓库 pull 服务桥接 |
| 1.15 | 2026-04-18 | 增加 `git status --json` 默认仓库状态摘要，并复用 serve 状态桥接 |
| 1.16 | 2026-04-18 | 增加 `git sync --json` 与 `git pull --json` 底层同步摘要，并复用默认仓库 sync 服务桥接 |
| 1.17 | 2026-04-18 | 增加默认仓库 `push-preview` API，并要求服务端 push 显式 `confirm=true` 与变更复核 |
| 1.18 | 2026-04-18 | Web UI 管理端增加默认仓库 push 预览与二次确认流程 |
| 1.19 | 2026-04-18 | 服务 API 包装错误保留 `pkg/errors` 稳定错误码并按错误类别映射 HTTP 状态 |
| 1.20 | 2026-04-18 | `serve` 默认 loopback 监听下增加 Host header loopback 校验，并保留非 loopback 绑定兼容性 |
| 1.21 | 2026-04-19 | Web UI/API 增加基础安全响应头，并在默认 loopback 监听下拒绝跨站写请求 |
| 1.22 | 2026-04-19 | `serve` 增加 `--secret-key` 保护配置，后续收口为默认仓库远端推送保护 |
| 1.23 | 2026-04-19 | CLI bridge 保留服务端错误码，Web UI 管理端增强只读与密钥错误提示 |
| 1.24 | 2026-04-25 | 增加 `upgrade` 自升级命令，复用 GitHub Release 资产、sha256 校验、agent workflow skills 同步和 serve 重启约定 |
| 1.24 | 2026-04-20 | Web UI 目录页技能总数改用服务端 `total`，管理端移除写入密钥入口 |
| 1.25 | 2026-04-20 | 移除技能工具适配范围的数据模型、展示和筛选逻辑 |
| 1.26 | 2026-04-20 | `target` 降级为兼容输入，Web UI 不再展示目标选择，项目业务统一使用 `.agents/skills` |
| 1.27 | 2026-04-20 | `serve --secret-key` 收口为远端推送保护，未配置时允许仓库拉取/同步，仅禁止默认仓库 push |
| 1.28 | 2026-04-23 | 增加 `use/apply/status/remove --global` 本机全局技能状态、agent 目录检查刷新、服务桥接和 manifest 冲突保护 |
| 1.29 | 2026-05-23 | 增加项目工作区 skill 来源指纹、`ModifiedAgainstOutdatedRepo` 状态和低版本 `feedback` 防护 |
| 1.30 | 2026-06-03 | 增加 `list/use/status/apply/feedback/validate` 基于技能 ID 字段的 glob pattern 支持（`*` / `?` / `**`），单字面量参数仍按精确 ID 兼容处理，详见 §6 |
| 1.31 | 2026-10-28 | 增加 `project register --request-id` 空项目登记与 `--query` 原请求回执查询，覆盖回复丢失后的安全重放 |
