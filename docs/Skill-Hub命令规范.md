# skill-hub 命令规范

## 1. 简介

本文档定义了 skill-hub CLI 工具的所有命令规范，旨在统一各设计文档中的命令定义，消除冲突和歧义。

## 安装脚本约定

一键安装脚本安装二进制、Shell 补全和 release 内置 agent workflow skills。Agent workflow skills 的主安装目录是工具无关路径：

```bash
${XDG_DATA_HOME:-$HOME/.local/share}/skill-hub/agent-skills
```

安装脚本会检测当前系统中已安装或已有配置目录的 agent，再同步镜像到对应全局 skills 目录。当前支持的自动检测镜像包括 Codex 与 OpenCode；Claude 仅在 `~/.claude/skills` 已存在或用户显式指定 `CLAUDE_SKILLS_DIR` / `SKILL_HUB_INSTALL_CLAUDE_SKILLS=1` 时镜像。Codex 默认目录示例：

```bash
${CODEX_HOME:-$HOME/.codex}/skills
```

OpenCode 默认目录示例：

```bash
${OPENCODE_HOME:-$HOME/.config/opencode}/skills
```

可通过 `SKILL_HUB_INSTALL_AGENT_SKILLS=0` 跳过所有 agent skills 安装，通过 `SKILL_HUB_AGENT_SKILLS_DIR=/path/to/skills` 覆盖工具无关目录。Codex、OpenCode、Claude 镜像分别可用 `SKILL_HUB_INSTALL_CODEX_SKILLS=0`、`SKILL_HUB_INSTALL_OPENCODE_SKILLS=0`、`SKILL_HUB_INSTALL_CLAUDE_SKILLS=0` 关闭，也可用 `CODEX_SKILLS_DIR`、`OPENCODE_SKILLS_DIR`、`CLAUDE_SKILLS_DIR` 覆盖目标目录。安装脚本只覆盖 release 内置的 `skill-hub-*` workflow skills，不扫描或删除其他用户 skill。

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

## 3. 核心命令列表（按功能分组）

### 3.1. 环境配置
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `init` | 初始化本地仓库 | `skill-hub init [git_url]` |
| `serve` | 以本地服务模式运行 | `skill-hub serve [--host <value>] [--port <value>] [--secret-key <value>] [--open-browser]` |

### 3.2. 技能发现
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `list` | 列出可用技能 | `skill-hub list [--verbose]` |
| `search` | 搜索远程技能 | `skill-hub search <keyword> [--limit <number>]` |

### 3.3. 技能创建/移除/验证/使用
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `create` | 创建新技能模板 | `skill-hub create <id>` |
| `register` | 登记已有项目本地技能 | `skill-hub register <id> [--skip-validate]` |
| `import` | 批量登记、验证并可选归档已有技能 | `skill-hub import <skills-dir> [--fix-frontmatter] [--archive] [--force] [--dry-run] [--fail-fast]` |
| `dedupe` | 检测嵌套项目中的重复技能副本 | `skill-hub dedupe <scope> [--canonical <dir>] [--strategy newest|canonical|fail-on-conflict] [--json]` |
| `sync-copies` | 从 canonical 目录同步同 ID 技能副本 | `skill-hub sync-copies --canonical <dir> [--scope <dir>] [--dry-run] [--no-backup] [--json]` |
| `lint` | 审计项目技能内容 | `skill-hub lint [scope] --paths [--project-root <dir>] [--fix] [--dry-run] [--no-backup] [--json]` |
| `audit` | 生成技能刷新审计报告 | `skill-hub audit [scope] [--output <file>] [--format markdown|json] [--canonical <dir>] [--project-root <dir>]` |
| `remove` | 移除项目技能 | `skill-hub remove <id>` |
| `validate` | 验证技能合规性 | `skill-hub validate <id> [--fix] [--links] [--json]` 或 `skill-hub validate --all [--fix] [--links] [--json]` |
| `use` | 使用本地仓库里的指定技能 | `skill-hub use <id>` |

补充：`use` / `remove` 支持 `--global [--agent codex|opencode|claude]`，用于本机全局 agent skills 目录；`list` 仍只表示本地仓库中的可用 skill 清单，不存在项目或全局 scope。


### 3.4. 技能状态
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `status` | 检查技能状态 | `skill-hub status [id] [--verbose] [--json]` |

补充：`status --global [--agent codex|opencode|claude] [--json]` 会检查全局期望状态与当前机器实际 agent skills 目录是否一致。

### 3.5. 项目工作区-本地仓库交互
| 命令 | 功能描述 | 语法 |
|------|----------|------|
| `apply` | 应用技能到项目 | `skill-hub apply [--dry-run] [--force]` |
| `feedback` | 将项目工作区技能修改内容更新至到本地仓库 | `skill-hub feedback <id> [--dry-run] [--force] [--json]` 或 `skill-hub feedback --all [--dry-run] [--force] [--json]` |

补充：`apply [id] --global [--agent codex|opencode|claude] [--dry-run] [--force]` 会按 `global-state.json` 刷新本机 agent 全局 skills 目录。默认不覆盖没有 `.skill-hub-manifest.json` 的同名目录；`--force` 会先创建备份再覆盖。

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

## 4. 命令详细规范

### 4.0 serve - 以本地服务模式运行

**语法**:

- `skill-hub serve [--host <value>] [--port <value>] [--secret-key <value>] [--open-browser]`
- `skill-hub serve register <name> [--host <value>] [--port <value>] [--secret-key <value>]`
- `skill-hub serve start <name>`
- `skill-hub serve stop <name>`
- `skill-hub serve status [name]`
- `skill-hub serve remove <name>`

**选项**:
- `--host <value>`: 监听地址，默认 `127.0.0.1`
- `--port <value>`: 监听端口，默认 `5525`
- `--secret-key <value>`: 远端推送密钥；未配置时禁止将本地仓库推送至远端，但不影响仓库拉取/同步和项目本地操作
- `--open-browser`: 启动后尝试打开浏览器

**功能描述**:

启动 `skill-hub` 本地服务模式，提供：

- 本地 HTTP API
- 本地 Web UI
- 供 CLI 复用的服务桥接入口

当前实现补充：

- `serve` 裸命令仍用于前台直接运行服务
- `serve register/start/stop/status/remove` 用于管理命名服务实例
- 服务实例注册表保存在 `~/.skill-hub/services.json`
- `serve start` 会后台启动当前 `skill-hub` 可执行文件，并记录 `pid` 与日志文件路径
- `serve register` 会保存 `secret_key` 用于后续 `serve start`，但 `serve status` 只显示 `push=blocked` 或 `push=secret-key`，不输出密钥明文
- `serve status` 会输出服务地址与运行状态，其中：
  - `running` 表示已记录 `pid` 且进程仍存活
  - `stopped` 表示当前未记录运行中进程
  - `stale` 表示注册表中仍有 `pid`，但对应进程已不存在
- `serve remove` 只删除注册信息，不会删除日志文件；若服务仍在运行会拒绝删除
- 当服务绑定在 loopback 地址（默认 `127.0.0.1` 或 `localhost`）时，HTTP server 会拒绝非 loopback Host header；显式绑定到非 loopback 地址时保留已有远程访问兼容行为
- 当服务绑定在 loopback 地址时，修改类 HTTP 方法会拒绝非 loopback `Origin` / `Referer`，并拒绝 `Sec-Fetch-Site: cross-site`；CLI bridge 不携带浏览器来源头，继续保持兼容
- 服务响应会设置基础安全响应头，包括 `Content-Security-Policy`、`X-Frame-Options`、`X-Content-Type-Options` 与 `Referrer-Policy`
- 仅远端推送 API 需要 `secretKey`：未配置 `secretKey` 时，`POST /api/v1/skill-repository/push` 返回 `READ_ONLY`；配置后该请求必须携带 `X-Skill-Hub-Secret-Key`。当前阶段 Web UI 管理端不开放写入密钥能力，只展示只读或密钥错误；CLI bridge 可通过 `SKILL_HUB_SERVICE_SECRET_KEY` 传递该值
- CLI bridge 会保留服务端返回的 `code/message`，因此远端推送被禁止或密钥错误会显示为 `READ_ONLY` / `UNAUTHORIZED`，不会被泛化成 `SYSTEM_ERROR`
- 服务 API 会保留业务层 `pkg/errors` 稳定错误码；未找到类错误返回 `404`，权限类返回 `403`，网络或远端 Git 类返回 `502`，未实现返回 `501`，系统错误返回 `500`，其余输入或校验类错误返回 `400`

当前 Web UI 支持：

- 仓库管理
- 技能列表查看
- 项目列表查看
- 项目 `status` / `use` / `apply` / `feedback` 操作

**示例**:
```bash
skill-hub serve
skill-hub serve --port 6600
skill-hub serve --host 127.0.0.1 --port 6600 --secret-key write-secret --open-browser
skill-hub serve register local --host 127.0.0.1 --port 6600 --secret-key write-secret
skill-hub serve start local
skill-hub serve status
skill-hub serve status local
skill-hub serve stop local
skill-hub serve remove local
```

### 4.1 init - 初始化本地仓库

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

### 4.3 list - 列出可用技能

**语法**: `skill-hub list [--verbose] [--repo <repo-name>...]`

**选项**:
- `--verbose`: 显示详细信息，包括技能描述、版本、适用说明等。
- `--repo <repo-name>`: 按仓库名称过滤技能列表（可多次使用指定多个仓库）。

**功能描述**:

显示所有已启用仓库中的技能，支持按仓库过滤。默认显示简要列表，包含技能 ID、状态、版本、所属仓库和适用范围信息。

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
```

### 4.4 search - 搜索远程技能

**语法**: `skill-hub search <keyword> [--limit <number>]`

**参数**:
- `keyword` (必需): 搜索关键词。

**选项**:
- `--limit <number>`: 限制返回结果数量，默认 20。

**功能描述**:

搜索远端技能入口。命令行仍作为用户入口；当本地 `skill-hub serve` 实例可用时，远端搜索交互会优先由服务实例统一承接，以复用本地托管的 `~/.skill-hub/` 上下文、远端访问策略和 Web/CLI 共用能力。

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

**语法**: `skill-hub create <id>`

**参数**:
- `id` (必需): 新技能的标识符。

**功能描述**:

在项目工作区创建一个新技能。不写入项目 target，也不主动写入技能 `compatibility` 声明。生成的 `SKILL.md` 默认使用中文定义，包括适用场景、工作流程、输出要求和注意事项；命令、文件路径、API 名称等技术标识保留原始写法。

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

**语法**: `skill-hub register <id> [--skip-validate]`

**参数**:
- `id` (必需): 已存在于 `.agents/skills/<id>/SKILL.md` 的技能标识符。
- `--skip-validate` (可选): 跳过 `SKILL.md` 验证，直接登记项目状态。

**功能描述**:

将当前项目中已经存在的技能登记到 `state.json`。该命令不会创建、重写或覆盖 `SKILL.md`，用于替代过去依赖 `create` 登记已有技能的语义。

默认会先验证 `SKILL.md` 的基础 frontmatter。若 legacy 技能缺失 frontmatter，可先执行 `skill-hub validate <id> --fix`，或在明确知道内容可接受时使用 `--skip-validate` 完成登记。

当本地服务模式可用时，CLI 会通过服务桥接执行登记，服务端负责读写其托管的 `~/.skill-hub/state.json`。

**示例**:
```bash
skill-hub register existing-skill
skill-hub register legacy-skill --skip-validate
```

### 4.5.2 import - 批量导入已有技能

**语法**: `skill-hub import <skills-dir> [--fix-frontmatter] [--archive] [--force] [--dry-run] [--fail-fast]`

**参数**:
- `skills-dir` (必需): 包含 `<id>/SKILL.md` 子目录的技能目录，典型值为 `.agents/skills`。
- `--fix-frontmatter` (可选): 导入前修复缺失或不完整的 frontmatter，修改前创建 `SKILL.md.bak.<timestamp>`。
- `--archive` (可选): 验证通过后归档到默认仓库。
- `--force` (可选): 批量流程不进行交互确认；不会自动删除源技能。
- `--dry-run` (可选): 只输出将要登记、修复、归档的动作。
- `--fail-fast` (可选): 遇到首个失败技能时停止，否则默认继续处理并在最后汇总失败项。

**功能描述**:

扫描 `skills-dir/*/SKILL.md`，逐个执行 frontmatter 修复（可选）、验证、项目状态登记，并在指定 `--archive` 时归档到默认仓库。命令结束时输出固定摘要：

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
skill-hub import .agents/skills --dry-run
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

默认修复前会创建 `<file>.bak.<timestamp>` 备份。CLI 会先把 `scope` 和 `project-root` 按调用方当前工作目录解析为绝对路径，再发送给服务端，因此在本地 `serve` 实例可用时仍能正确操作调用方项目。

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

当本地 `serve` 实例可用时，CLI 会通过服务桥接执行审计聚合；CLI 会把 `scope`、`canonical`、`project-root` 解析为绝对路径后传给服务端。`--output` 文件由 CLI 在调用方侧写入，避免服务进程当前目录不同导致报告写错位置。

**示例**:
```bash
skill-hub audit .agents/skills --output .agents/skills-refresh-progress.md
skill-hub audit .agents/skills --format json
skill-hub audit . --canonical .agents/skills --project-root "$PWD"
```

### 4.6 remove - 移除项目技能

**语法**: `skill-hub remove <id> [--global] [--agent codex|opencode|claude] [--force]`

**参数**:
- `id` (必需): 要移除的技能标识符。

**功能描述**:

从当前项目工作区中移除指定的技能：
1. 从 `state.json` 中移除技能标记
2. 物理删除项目工作区对应的文件/配置
3. 保留本地仓库中的源文件不受影响

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区；新建项目工作区统一初始化 `.agents/skills`，不会根据 target 创建专属文件。

**安全机制**: 如果检测到本地有未反馈的修改，会弹出警告并要求确认。

使用 `--global` 时，命令不要求当前目录是项目工作区，而是从 `~/.skill-hub/global-state.json` 中移除对应 agent 的全局期望状态，并删除带有 `.skill-hub-manifest.json` 的 agent 全局 skill 目录。默认不会删除没有 Skill-Hub manifest 的同名目录；`--force` 才允许强制删除冲突目录。

**示例**:
```bash
# 移除 git-expert 技能
skill-hub remove git-expert

# 从 Codex 全局目录移除 git-expert
skill-hub remove git-expert --global --agent codex
```

### 4.7 validate - 验证技能合规性

**语法**: `skill-hub validate <id> [--fix] [--links] [--project-root <dir>] [--check-remote] [--json]` 或 `skill-hub validate --all [--fix] [--links] [--project-root <dir>] [--check-remote] [--json]`

**参数**:
- `id` (必需): 要验证的技能标识符。

**选项**:
- `--fix`: 修复缺失或不完整的 `SKILL.md` frontmatter，并在修改前创建 `SKILL.md.bak.<timestamp>`。
- `--all`: 验证当前项目状态中登记的全部技能。
- `--links`: 检查 `SKILL.md` 和技能目录内 Markdown 文件中的链接。
- `--project-root <dir>`: 解析项目相对链接的根目录，默认使用当前项目目录或 `metadata.project_root`。
- `--check-remote`: 同时检查 HTTP/HTTPS 远端链接。默认忽略远端链接。
- `--json`: 输出机器可读 JSON 验证报告。

**功能描述**:

验证指定技能在项目工作区中的本地文件是否合规，包括检查 `SKILL.md` 的 YAML 语法、必需字段和基本文件结构。

`--fix` 会为 legacy `SKILL.md` 补齐最小 frontmatter：`name`、`description`、`metadata.version`、`metadata.author`。`compatibility` 已降级为可选说明字段，不再由修复流程主动补写。`name` 使用技能目录名，`description` 优先从正文首个有效段落推断，版本默认 `1.0.0`。修复只重写 frontmatter，不改写正文内容。

`--links` 会解析技能目录内 Markdown 文件中的本地链接。外部 HTTP/HTTPS 链接默认跳过；本地链接会按源文件目录、技能目录、项目根目录依次解析，任一目标存在即视为通过。缺失目标会以 `source_file`、`line`、`link`、`resolved_path` 输出，JSON 报告中汇总为 `link_issues`。

该命令与 `create` 配套，主要用于 `feedback` 前的本地校验，不负责校验仓库侧内容。当前本地 `serve` 实例可用时，CLI 会通过服务桥接执行 validate；CLI 会把 `project_path` 和 `project-root` 解析为绝对路径后传给服务端。

如果该技能未在`state.json`里项目工作区登记，则提示该技能非法

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区；新建项目工作区统一初始化 `.agents/skills`，不会根据 target 创建专属文件。

**示例**:
```bash
# 验证 my-logic 技能的合规性
skill-hub validate my-logic

# 修复 legacy frontmatter 后验证
skill-hub validate my-logic --fix

# 验证项目中全部已登记技能
skill-hub validate --all

# 检查本地 Markdown 链接并输出 JSON 报告
skill-hub validate --all --links --json
```

### 4.8 use - 使用技能

**语法**: `skill-hub use <id> [--global] [--agent codex|opencode|claude]`

**参数**:
- `id` (必需): 要启用的技能标识符。

**功能描述**:

将技能标记为在当前项目中使用。此命令仅更新 `state.json` 中的技能记录，不直接修改项目文件，不写入 target。需要通过 `apply` 命令进行物理分发。

使用 `--global` 时，命令会将技能写入 `~/.skill-hub/global-state.json`，表示用户希望该 skill 对指定 agent 在本机全局有效。未传 `--agent` 时会根据当前机器已检测到或已配置的 agent 目录选择目标；显式传入 `--agent` 可重复指定。

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

# 启用为本机全局 Codex skill
skill-hub use git-expert --global --agent codex

```

### 4.9 status - 检查技能状态

**语法**: `skill-hub status [id] [--verbose] [--json] [--global] [--agent codex|opencode|claude]`

**参数**:
- `id` (可选): 特定技能的标识符。如未提供，检查所有技能。

**选项**:
- `--verbose`: 显示详细差异信息。
- `--json`: 以机器可读 JSON 输出状态摘要，字段包含 `skill_id`、`status`、`local_version`、`repo_version`、`local_path`、`repo_path`、`source_repository` 等。
- `--global`: 检查本机全局期望状态与实际 agent skills 目录是否一致。
- `--agent`: 限制全局检查的 agent，可重复使用。仅在 `--global` 下生效。

**功能描述**:

对比项目本地工作区与技能仓库源文件的差异，显示技能状态。对比范围包括技能目录下全部文件（含 `SKILL.md` 及 `scripts/`、`references/`、`assets/` 等子目录），任一文件内容或清单差异即可能显示为 Modified：
- `Synced`: 本地与仓库一致
- `Modified`: 本地有未反馈的修改（含子目录内文件变更）
- `Outdated`: 仓库版本领先于本地
- `Missing`: 技能已启用但本地文件缺失

使用 `--global` 时，状态语义切换为全局一致性检查：
- `ok`: `global-state.json`、来源仓库与 agent 全局目录一致
- `not_applied`: 已全局启用但尚未写入 agent 目录
- `modified`: agent 目录中的 Skill-Hub 托管副本被手动修改
- `stale`: 来源仓库内容已变化，agent 目录需要刷新
- `conflict`: agent 目录存在同名但非 Skill-Hub 托管目录
- `orphaned`: agent 目录存在 Skill-Hub manifest，但全局状态已不再记录
- `missing_agent_dir`: agent skills 目录不存在

当前实现补充：

- 当本地服务模式可用时，CLI 会优先通过服务桥接执行
- 当显式指定的全局 skill 尚未启用，或 `--agent` 与该 skill 的全局启用目标不匹配时，返回 `SKILL_NOT_FOUND`，不再输出空状态摘要

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
skill-hub status git-expert --global --agent codex
```

### 4.10 apply - 应用技能到项目工作区

**语法**: `skill-hub apply [id] [--dry-run] [--force] [--global] [--agent codex|opencode|claude]`

**选项**:
- `--dry-run`: 演习模式，仅显示将要执行的变更，不实际修改文件。
- `--force`: 强制应用，即使检测到冲突也继续执行。
- `--global`: 按 `global-state.json` 刷新本机 agent 全局 skills 目录。
- `--agent`: 限制全局刷新的 agent，可重复使用。仅在 `--global` 下生效。

**功能描述**:

根据 `state.json` 中的启用记录，将技能物理分发到标准项目工作区 `.agents/skills/<id>/`。历史 target 不参与分发路径选择。

使用 `--global` 时，命令会从来源仓库刷新 `~/.skill-hub/global/skills/<id>/` 镜像，再同步到目标 agent 全局 skills 目录。写入 agent 目录时会生成 `.skill-hub-manifest.json`；同名目录存在但没有 manifest 时默认报告 `conflict`，不会覆盖，除非用户显式提供 `--force`。`--force` 覆盖前会创建 `*.skill-hub-backup.<timestamp>` 备份。

当前实现补充：

- 当本地服务模式可用时，CLI 会优先通过服务桥接执行
- 服务端负责实际适配器调用和项目文件分发
- 当显式指定的全局 skill 尚未启用，或 `--agent` 与该 skill 的全局启用目标不匹配时，返回 `SKILL_NOT_FOUND`，不会静默跳过刷新

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区；新建项目工作区统一初始化 `.agents/skills`，不会根据 target 创建专属文件。

**示例**:
```bash
# 应用启用技能到标准项目工作区
skill-hub apply

# 演习模式查看将要进行的变更
skill-hub apply --dry-run

# 预览和刷新本机全局技能
skill-hub apply --global --dry-run
skill-hub apply --global
skill-hub apply git-expert --global --agent codex
```

### 4.11 feedback - 将项目工作区里技能修改更新至本地仓库

**语法**: `skill-hub feedback <id> [--dry-run] [--force] [--json]` 或 `skill-hub feedback --all [--dry-run] [--force] [--json]`

**参数**:
- `id` (单技能模式必需): 要更新的技能标识符。

**选项**:
- `--dry-run`: 演习模式，仅显示将要同步的差异。
- `--force`: 强制更新，即使有冲突也继续执行。
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
- `--all` 会按当前项目状态中登记的技能逐个执行反馈。为避免误批量归档，实际写入时必须显式提供 `--force`；也可以使用 `--dry-run` 预览。`--json` 模式不会进入交互确认，实际写入同样必须显式提供 `--force`。

**多仓库说明**：技能会被归档到默认仓库（通过 `skill-hub repo default` 命令设置）。如果技能在默认仓库中不存在则新增，存在则覆盖更新。

该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化

该命令会检查当前所在目录是否存在于本地仓库的`state.json`中，如果存在则更新，不存在则提示是否需要新建项目工作区；新建项目工作区统一初始化 `.agents/skills`，不会根据 target 创建专属文件。


**示例**:
```bash
# 反馈 my-logic 技能的修改
skill-hub feedback my-logic

# 演习模式查看将要同步的差异
skill-hub feedback git-expert --dry-run

# 批量反馈并输出机器可读摘要
skill-hub feedback --all --force --json
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
- `--force`: 强制拉取，忽略本地未提交的修改。
- `--check`: 检查模式，仅显示可用的更新，不实际执行拉取操作。
- `--json`: 输出机器可读拉取摘要。

**功能描述**:

从默认仓库对应的远程拉取最新更改到本地仓库（`~/.skill-hub/repositories/<default>/`），并更新技能索引。此命令仅同步仓库层，不涉及项目工作目录的更新。

`--check` 会执行不修改工作树的远端检查：有远端时通过 fetch 更新远端引用并比较 `main` 分支提交；无远端时返回 `no_remote`。JSON 输出中的 `status` 可能为 `no_remote`、`up_to_date`、`updates_available`、`ahead` 或 `divergent`，并包含 `ahead` / `behind` 计数。

`git sync`、`git pull` 以及 `repo sync` 也遵循同样的索引刷新原则：仓库内容变化后会重建对应 repo 的 `registry.json`，默认仓库同时会刷新根目录兼容索引。

**多仓库说明**：此命令仅处理默认仓库。如需同步指定仓库或所有启用仓库，请使用 `skill-hub repo sync`。

本地执行时该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化。本地 `serve` 实例可用时，CLI 会优先通过服务桥接执行默认仓库同步，以兼容客户端 HOME 中没有本地 skill-hub 配置的场景。


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
- `--message MESSAGE`, `-m MESSAGE`: 提交消息。如未提供，使用默认消息"更新技能"。
- `--force`: 强制推送，跳过确认检查。
- `--dry-run`: 演习模式，仅显示将要推送的更改，不实际执行。
- `--json`: 输出机器可读推送摘要；实际写入推送时必须同时使用 `--force`，或使用 `--dry-run` 只预览。

**功能描述**:

自动检测并提交默认仓库中的未提交更改，然后推送到对应远程仓库。此命令用于完成 `feedback -> push` 的归档闭环。

**多仓库说明**：默认推送到默认仓库（归档仓库）。可使用 `skill-hub repo default` 命令设置默认仓库。

本地执行时该命令依赖`init`命令，如果检查本地仓库不存在，则提示需要先进行初始化。本地 `serve` 实例可用时，CLI 会优先通过服务桥接获取默认仓库状态并执行推送，以兼容客户端 HOME 中没有本地 skill-hub 配置的场景。

服务 API 中默认仓库推送为受保护写操作：

- `GET /api/v1/skill-repository/push-preview` 返回默认仓库、远端 URL、待推送文件列表、建议提交消息和原始状态。
- `POST /api/v1/skill-repository/push` 必须携带 `confirm: true`。
- 请求可携带 `expected_changed_files`；服务端执行前会重新检查默认仓库状态，若文件列表已变化则拒绝推送。
- 无待推送变更时服务端拒绝推送。
- 未配置 `serve --secret-key` 时该推送 API 返回 `READ_ONLY`；仓库拉取/同步类 API 不受该密钥限制。

**关键行为**:
1. 检查默认仓库是否有未提交的更改（Modified、Untracked、Deleted文件）
2. 如有更改，自动提交（使用指定或默认消息）
3. 将提交推送到远程仓库
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

### 4.15 git - Git仓库操作

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
- `skill-hub git commit` - 提交更改到本地仓库（交互式输入消息）
- `skill-hub git push` - 推送本地提交到远程仓库（检查未提交更改）
- `skill-hub git pull [--json]` - 从远程仓库拉取更新（同 `sync`）
- `skill-hub git remote [url]` - 设置或更新远程仓库URL

**功能描述**:
提供底层Git仓库操作接口，适用于需要精细控制Git工作流的用户。这些命令默认操作本地技能仓库（`~/.skill-hub/repositories/<name>/`），与高级命令（如 `repo sync`、`push`）功能重叠但提供更多控制选项。其中 `git status --json`、`git sync --json`、`git pull --json` 在本地 `serve` 实例可用时会优先通过服务桥接读取或同步默认仓库，以便客户端 HOME 无本地配置时仍可用于脚本检查。

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
- `--dry-run`: 演习模式（支持命令: `apply`, `feedback`, `import`, `sync-copies`, `lint`, `push`）
- `--force`: 强制模式（支持命令: `apply`, `feedback`, `import`, `pull`, `push`）
- `--json`: 机器可读输出（支持命令: `status`, `repo list`, `repo sync`, `feedback`, `validate`, `dedupe`, `sync-copies`, `lint`, `audit`, `pull`, `push`）

## 6. 使用示例

### 6.1 完整工作流示例

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

**语法**: `skill-hub repo sync [name] [--all] [--json]`

**参数**:
- `name` (可选): 要同步的仓库名称。如未提供，同步所有启用的仓库

**选项**:
- `--all`: 同步所有仓库，包括已禁用仓库。默认只同步启用仓库。
- `--json`: 输出机器可读同步摘要，包含 `total`、`synced`、`skipped`、`failed` 和每个仓库的同步状态。

**功能描述**:
同步指定仓库或所有启用的仓库。对于远程仓库，会执行 `git pull` 获取最新内容。
同步后会自动刷新技能索引。

**示例**:
```bash
# 同步所有启用的仓库
skill-hub repo sync

# 同步特定仓库
skill-hub repo sync community

# 输出 JSON 同步摘要
skill-hub repo sync --json
```

## 8. 服务模式与 CLI 交互说明

当前 `skill-hub` 已支持双运行模式：

- CLI 模式
- Service 模式

服务模式启动后：

- Web UI 可通过浏览器访问
- CLI 中的部分命令会优先通过服务桥接执行
- 远端推送桥接到配置了 `secretKey` 的服务时，调用方需设置 `SKILL_HUB_SERVICE_SECRET_KEY=<secretKey>`；未配置 `secretKey` 的服务允许读取、项目本地操作和仓库拉取/同步，但禁止默认仓库 push 到远端
- 当前阶段 Web UI 管理端不提供密钥输入或浏览器会话保存能力，远端推送权限只通过后端配置和调用方显式携带 header / CLI bridge 环境变量获得

当前已桥接的命令：

- `repo *`
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
- `remove` 的项目模式
- `prune`
- `git` 的交互式或非 JSON 子命令

## 9. 更新记录

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
| 1.24 | 2026-04-20 | Web UI 目录页技能总数改用服务端 `total`，管理端移除写入密钥入口 |
| 1.25 | 2026-04-20 | 弱化 Skill `compatibility` 处理，列表/搜索/Web UI 不再按兼容性硬过滤 |
| 1.26 | 2026-04-20 | `target` 降级为兼容输入，Web UI 不再展示目标选择，项目业务统一使用 `.agents/skills` |
| 1.27 | 2026-04-20 | `serve --secret-key` 收口为远端推送保护，未配置时允许仓库拉取/同步，仅禁止默认仓库 push |
| 1.28 | 2026-04-23 | 增加 `use/apply/status/remove --global` 本机全局技能状态、agent 目录检查刷新、服务桥接和 manifest 冲突保护 |
