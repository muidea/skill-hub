# skill-hub 🚀

一款专为 AI 时代开发者设计的"技能（Prompt/Script）生命周期管理工具"。它旨在解决 AI 指令碎片化、跨工具同步难、缺乏版本控制等痛点。

[![CI](https://github.com/muidea/skill-hub/actions/workflows/ci.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/ci.yml)
[![Tests](https://github.com/muidea/skill-hub/actions/workflows/test.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/test.yml)
[![Release](https://github.com/muidea/skill-hub/actions/workflows/release.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/release.yml)

## 简介

`skill-hub` 首先是一个命令行工具，用于管理 skill、在项目中启用 skill，并维护 skill 的仓库生命周期。

同时它支持以 `serve` 模式运行。`serve` 模式会托管用户本地的 `~/.skill-hub/` 目录，提供 Web 管理能力；CLI 在执行命令时会优先与本地 `serve` 实例交互，但 `serve` 不是必选项，本地服务不可用时所有命令都必须回退到本地执行。

当前除了前台直接运行：

```bash
skill-hub serve
# 推送本地仓库到远端需开启 secretKey；未配置时仅禁止远端推送
skill-hub serve --secret-key write-secret
```

也支持服务实例管理：

```bash
skill-hub serve register local --host 127.0.0.1 --port 6600 --secret-key write-secret
skill-hub serve start local
skill-hub serve status
skill-hub serve stop local
skill-hub serve remove local
```

服务注册信息会持久化到 `~/.skill-hub/services.json`，`start` 会记录后台进程 `pid` 与日志路径，`status` 会显示 `running` / `stopped` / `stale` 三种状态，并以 `push=blocked` 或 `push=secret-key` 标识远端推送保护模式。

### 概念模型

1. **全局管理目录**：`~/.skill-hub/`
   承接多仓库配置、本地仓库存储、默认仓库、全局状态、索引和缓存。

2. **项目本地工作区**：`<project>/.agents/skills/`
   承接当前项目实际使用的 skill 内容，是项目侧的工作目录。

3. **来源仓库**
   指某个 skill 在 `use` 时选中的仓库。它决定 `apply` 的读取来源，以及 `status` 的上游对比基线。

4. **默认仓库**
   指当前配置中的默认仓库，同时也是归档仓库。`feedback` 只归档到默认仓库，`pull` / `push` 默认也只操作默认仓库。

5. **多仓库管理**
   指 `~/.skill-hub/` 下多个本地仓库的配置、启停、默认仓库切换和同步能力。

6. **服务实例**
   指 `skill-hub serve` 启动的本地服务。它托管 `~/.skill-hub/`，为 Web 管理和 CLI 复用提供统一执行入口。

## 项目结构

当前代码结构包含模块化内核与现有 CLI 实现。当前构建入口统一位于 `application/skill-hub/cmd`，实际命令执行仍由 `internal/cli` 承载，`internal/modules/kernel` 主要提供服务化封装与后续重构落点：

```text
skill-hub/
├── application/skill-hub/      # 主应用入口
│   ├── cmd/                    # 推荐构建入口
│   └── docker/                 # Docker 启动定义
├── internal/modules/kernel/    # 核心模块壳层
├── internal/cli/               # CLI 命令实现
├── internal/adapter/           # 外部工具适配器
├── internal/engine/            # 技能引擎
├── internal/state/             # 状态管理
└── pkg/                        # 公共包
```

当前推荐构建命令：

```bash
go build -o bin/skill-hub ./application/skill-hub/cmd
```

### 核心理念

- **Git 为中心**：所有技能存储在Git仓库中，作为单一可信源
- **多仓库架构**：支持同时管理多个技能仓库（个人、社区、官方）
- **一键分发**：将技能快速应用到不同的AI工具
- **闭环反馈**：将项目中的手动修改反馈回技能仓库
- **现代架构**：采用 Go 1.24+ 特性，遵循 Effective Go 最佳实践

### 功能特性

- **技能管理**：查看、启用、禁用技能
- **技能创建**：从当前项目创建新的技能模板
- **已有技能登记**：将 `.agents/skills/<id>/SKILL.md` 中的存量技能登记到项目状态
- **批量导入**：扫描 `.agents/skills/*/SKILL.md`，批量登记、修复、验证并可选归档
- **重复检测与副本同步**：扫描嵌套项目中的同 ID 技能副本，报告冲突，并可从 canonical 目录同步副本
- **本地验证**：校验项目本地 skill 是否符合规范，并可用 `--fix` 修复 legacy frontmatter
- **技能归档**：将验证通过的技能归档到正式仓库
- **跨工具同步**：支持 Cursor、Claude Code、OpenCode 等AI工具
- **版本控制**：基于Git的技能版本管理
- **差异检测**：自动检测手动修改并支持反馈
- **安全操作**：原子文件写入和备份机制
- **全面测试**：单元测试 + 端到端测试覆盖

### 命令分层

- **项目本地工作区命令**：`use`、`status`、`apply`、`feedback`、`pull`、`push`、`create`、`register`、`import`、`dedupe`、`sync-copies`、`lint`、`audit`、`remove`、`validate`
- **多仓库管理命令**：`repo *`
- **状态维护命令**：`prune`
- **远端搜索命令**：`search`
- **底层运维命令**：`git *`、`serve`
- **引导命令**：`init`

其中：

- `create` / `remove` 只作用于项目本地工作区，不参与服务化托管
- `search` 面向远端能力；当本地 `serve` 实例可用时优先通过服务承接远端交互，不可用时回退到本地执行
- `repo *` 面向 `~/.skill-hub/` 下的多仓库管理
- `prune` 用于清理 `state.json` 中因项目目录移动、删除而残留的失效项目记录
- `use` / `status` / `apply` / `feedback` / `pull` / `push` 都服务于项目工作流，但会依赖全局管理目录中的仓库与状态信息
- `register` / `import` / `dedupe` / `sync-copies` / `lint --paths` / `validate` / `audit` / `pull` / `push` 在本地 `serve` 实例可用时会优先通过服务桥接执行；涉及项目路径的命令会把路径解析为绝对路径再交给服务端，避免 `serve` 进程工作目录不同导致误扫或误改

## 🚀 快速开始

### 安装

使用一键安装脚本（最简单的方式）：

```bash
curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh | bash
```

### 基本使用

安装完成后，按照以下工作流程开始使用：

项目本地 Skill 目录统一使用 `.agents/skills/`。`target` 命令和参数入口已从 CLI/API 移除，不再写入项目状态、不参与适配器选择、不参与列表/搜索过滤，也不作为任何业务校验依据。Skill `compatibility` 仅在存在说明时作为基础信息展示。

#### 多仓库初始化流程
```bash
# 1. 初始化多仓库工作区（可指定初始仓库URL）
skill-hub init https://github.com/muidea/skill-hub-examples.git

# 2. 添加更多技能仓库（可选）
skill-hub repo add community https://github.com/community/skills.git
skill-hub repo add personal https://github.com/yourname/skills.git

# 3. 设置默认归档仓库
skill-hub repo default main

# 4. 启用技能
skill-hub use git-expert

# 5. 应用技能到项目
skill-hub apply
```

#### 技能创建与验证流程
```bash
# 1. 从当前项目创建新技能模板
skill-hub create my-new-skill

# 2. 在 feedback 前校验本地新建 skill
skill-hub validate my-new-skill

# 3. 反馈手动修改并归档到默认仓库
skill-hub feedback my-new-skill

# 批量反馈当前项目中已登记的所有技能，适合批量刷新后归档
skill-hub feedback --all --force --json

# 预览默认仓库待推送变更，输出机器可读摘要
skill-hub push --dry-run --json

# 检查默认仓库拉取动作，输出机器可读摘要
skill-hub pull --check --json

# 查看默认仓库工作区状态，输出机器可读摘要
skill-hub git status --json

# 执行底层默认仓库同步，输出机器可读摘要
skill-hub git sync --json
```

`pull --check --json` 会返回 `no_remote`、`up_to_date`、`updates_available`、`ahead` 或 `divergent`，并包含本地/远端提交与 `ahead` / `behind` 计数，便于自动化判断是否需要执行实际拉取。

服务 API 的默认仓库推送必须先预览再确认：`GET /api/v1/skill-repository/push-preview` 返回 `changed_files`，`POST /api/v1/skill-repository/push` 必须携带 `confirm: true`，并可携带 `expected_changed_files` 防止预览后文件变化仍继续推送。

Web UI 管理端的默认仓库推送入口也遵循同一约束：先点击“预览推送”取得变更列表，勾选确认后才会启用“推送默认仓库”。

服务 API 的业务错误会保留稳定错误码，例如 `SKILL_NOT_FOUND`、`PROJECT_NOT_FOUND`、`VALIDATION_FAILED`、`INVALID_INPUT`，并按错误类别映射到对应 HTTP 状态，便于 CLI bridge 和 Web UI 统一处理。

`serve` 默认绑定 `127.0.0.1`，在 loopback 监听下还会校验 Host header 必须为 loopback；显式绑定到非 loopback 地址时保留远程访问兼容行为。

Web UI/API 会设置基础安全响应头；默认 loopback 监听下，修改类请求会拒绝非 loopback `Origin` / `Referer` 和 `Sec-Fetch-Site: cross-site`，CLI bridge 仍保持兼容。

`serve` 远端推送采用可选 `secretKey`：未配置 `--secret-key` 时，读取、项目本地写入、仓库拉取/同步和 Web UI 仍可访问，但 `POST /api/v1/skill-repository/push` 会返回 `READ_ONLY`，禁止将本地仓库推送至远端；配置后该推送 API 必须携带 `X-Skill-Hub-Secret-Key`。当前阶段 Web UI 管理端不开放写入密钥能力，只展示只读或密钥错误；CLI bridge 可通过 `SKILL_HUB_SERVICE_SECRET_KEY` 传递该值。

#### 存量技能登记与批量导入流程
```bash
# 登记已有 .agents/skills/<id>/SKILL.md，不创建或覆盖内容
skill-hub register existing-skill

# 修复 legacy SKILL.md frontmatter（修改前自动创建 SKILL.md.bak.<timestamp>）
skill-hub validate existing-skill --fix

# 检查 SKILL.md 和技能目录内 Markdown 文件的本地链接
skill-hub validate existing-skill --links

# 批量扫描、修复、登记、验证，并归档到默认仓库
skill-hub import .agents/skills \
  --fix-frontmatter \
  --archive \
  --force

# 面向自动化脚本输出状态 JSON
skill-hub status --json
```

#### 重复检测与副本同步流程
```bash
# 检测 scope 下所有 .agents/skills/<id>/SKILL.md 重复副本
skill-hub dedupe . --canonical .agents/skills --strategy newest

# 输出机器可读报告
skill-hub dedupe . --canonical .agents/skills --json

# 从 canonical 目录同步所有同 ID 副本，默认先创建 <skill-dir>.bak.<timestamp>
skill-hub sync-copies --canonical .agents/skills --scope .

# 只预览将同步的副本
skill-hub sync-copies --canonical .agents/skills --scope . --dry-run
```

#### 路径可移植性审计流程
```bash
# 扫描技能内容中的本机绝对路径、file://、vscode:// 链接
skill-hub lint . --paths --project-root "$PWD"

# 自动将 project-root 内的本机路径改写为相对路径，默认创建 SKILL.md.bak.<timestamp>
skill-hub lint . --paths --project-root "$PWD" --fix

# 演习模式输出 JSON，适合脚本审计
skill-hub lint . --paths --project-root "$PWD" --fix --dry-run --json
```

#### 技能刷新审计报告
```bash
# 生成 Markdown 审计报告，聚合数量、登记、validate、status、dedupe、lint 和默认仓库推送状态
skill-hub audit .agents/skills --output .agents/skills-refresh-progress.md

# 输出 JSON 报告，适合 CI 或 agent 自动化消费
skill-hub audit .agents/skills --format json
```

#### 多仓库管理示例
```bash
# 查看所有仓库
skill-hub repo list

# 机器可读仓库列表
skill-hub repo list --json

# 同步所有仓库
skill-hub repo sync

# 机器可读同步摘要
skill-hub repo sync --json

# 启用/禁用仓库
skill-hub repo enable community
skill-hub repo disable personal
```

#### 状态维护示例
```bash
# 项目目录移动或删除后，清理 state.json 中的失效项目记录
skill-hub prune
```

## 🛠️ 命令参考

### 核心命令

| 命令 | 参数 | 功能说明 |
|------|------|----------|
| `init` | `[git_url]` | 初始化多仓库工作区 |
| `list` | `[--verbose]` | 列出可用技能；`compatibility` 仅作为说明展示，不过滤结果 |
| `search` | `<keyword> [--limit <number>]` | 搜索远程技能；CLI 在本地 `serve` 可用时优先通过服务承接 |
| `create` | `<id>` | 创建新技能模板 |
| `register` | `<id> [--skip-validate]` | 登记已有项目本地技能，不覆盖内容 |
| `import` | `<skills-dir> [--fix-frontmatter] [--archive] [--force] [--dry-run] [--fail-fast]` | 批量登记、验证并可选归档已有技能 |
| `dedupe` | `<scope> [--canonical <dir>] [--strategy newest\|canonical\|fail-on-conflict] [--json]` | 检测嵌套项目中的重复技能副本 |
| `sync-copies` | `--canonical <dir> [--scope <dir>] [--dry-run] [--no-backup] [--json]` | 从 canonical 目录同步同 ID 技能副本 |
| `lint` | `[scope] --paths [--project-root <dir>] [--fix] [--dry-run] [--no-backup] [--json]` | 审计并可修复技能内容中的本机路径 |
| `audit` | `[scope] [--output <file>] [--format markdown\|json] [--canonical <dir>] [--project-root <dir>]` | 生成技能刷新审计报告 |
| `remove` | `<id>` | 移除项目技能 |
| `validate` | `<id> [--fix] [--links] [--json]` 或 `--all [--fix] [--links] [--json]` | 验证项目工作区 skill 的合规性，可修复 legacy frontmatter 并检查本地 Markdown 链接 |
| `use` | `<id>` | 使用指定技能，仅更新项目状态 |
| `status` | `[id] [--verbose] [--json]` | 检查技能状态；`--json` 便于CI和脚本处理 |
| `apply` | `[--dry-run] [--force]` | 应用技能到项目 |
| `feedback` | `<id> [--dry-run] [--force] [--json]` 或 `--all [--dry-run] [--force] [--json]` | 反馈单个或全部已登记技能修改到默认仓库 |
| `prune` | `无` | 清理 state.json 中失效的项目记录 |
| `pull` | `[--force] [--check] [--json]` | 拉取默认仓库的远程更新 |
| `push` | `[--message MESSAGE] [--force] [--dry-run] [--json]` | 推送默认仓库的本地更改 |

### 多仓库管理命令

| 命令 | 参数 | 功能说明 |
|------|------|----------|
| `repo add` | `<name> <git_url>` | 添加新技能仓库 |
| `repo list` | `[--json]` | 列出所有技能仓库 |
| `repo remove` | `<name>` | 移除技能仓库 |
| `repo enable` | `<name>` | 启用技能仓库 |
| `repo disable` | `<name>` | 禁用技能仓库 |
| `repo default` | `<name>` | 设置默认（归档）仓库 |
| `repo sync` | `[name] [--all] [--json]` | 同步指定仓库或所有启用仓库 |

**语法说明**：`<参数>`为必需参数，`[参数]`为可选参数

**全局选项**：
- `-h, --help` - 显示帮助信息
- `-v, --version` - 显示版本信息
- `--dry-run` - 演习模式
- `--force` - 强制模式

## 📚 文档导航

### 用户文档

- **[详细安装和使用指南](INSTALLATION.md)** - 完整的安装方法、命令参考、技能管理和故障排除
  - 4种安装方法详解（一键脚本、预编译二进制、源码编译、本地开发）
  - 完整命令参考和常用工作流程
  - 技能规范、目录结构和变量系统
  - 项目工作区、技能元数据和适用说明
  - 常见问题故障排除

### 开发文档

- **[开发指南](DEVELOPMENT.md)** - 构建、发布、贡献和架构设计
  - 项目结构和代码架构
  - 开发环境设置和构建系统
  - 测试策略和发布流程
  - 贡献指南和代码审查
  - 性能优化和安全考虑

## 📋 其他信息

### 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

### 问题反馈

如遇到问题或有功能建议，请：
1. 查看现有Issue是否已解决
2. 创建新的Issue，详细描述问题
3. 提供复现步骤和环境信息

### 贡献指南

欢迎贡献代码！请参考 [DEVELOPMENT.md](DEVELOPMENT.md) 中的贡献指南。

---

**快速链接**:
- [GitHub仓库](https://github.com/muidea/skill-hub)
- [最新发布版本](https://github.com/muidea/skill-hub/releases)
- [问题反馈](https://github.com/muidea/skill-hub/issues)
- [开发文档](DEVELOPMENT.md)
- [安装指南](INSTALLATION.md)
