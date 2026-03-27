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
```

也支持服务实例管理：

```bash
skill-hub serve register local --host 127.0.0.1 --port 6600
skill-hub serve start local
skill-hub serve status
skill-hub serve stop local
skill-hub serve remove local
```

服务注册信息会持久化到 `~/.skill-hub/services.json`，`start` 会记录后台进程 `pid` 与日志路径，`status` 会显示 `running` / `stopped` / `stale` 三种状态。

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
- **本地验证**：校验 create 生成的项目本地 skill 是否符合规范
- **技能归档**：将验证通过的技能归档到正式仓库
- **跨工具同步**：支持 Cursor、Claude Code、OpenCode 等AI工具
- **版本控制**：基于Git的技能版本管理
- **差异检测**：自动检测手动修改并支持反馈
- **安全操作**：原子文件写入和备份机制
- **全面测试**：单元测试 + 端到端测试覆盖

### 命令分层

- **项目本地工作区命令**：`set-target`、`use`、`status`、`apply`、`feedback`、`pull`、`push`、`create`、`remove`、`validate`
- **多仓库管理命令**：`repo *`
- **状态维护命令**：`prune`
- **远端搜索命令**：`search`
- **底层运维命令**：`git *`、`serve`
- **引导命令**：`init`

其中：

- `create` / `remove` / `validate` 只作用于项目本地工作区，不参与服务化托管
- `search` 面向远端能力；当本地 `serve` 实例可用时优先通过服务承接远端交互，不可用时回退到本地执行
- `repo *` 面向 `~/.skill-hub/` 下的多仓库管理
- `prune` 用于清理 `state.json` 中因项目目录移动、删除而残留的失效项目记录
- `use` / `status` / `apply` / `feedback` / `set-target` / `pull` / `push` 都服务于项目工作流，但会依赖全局管理目录中的仓库与状态信息

## 🚀 快速开始

### 安装

使用一键安装脚本（最简单的方式）：

```bash
curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh | bash
```

### 基本使用

安装完成后，按照以下工作流程开始使用：

项目本地 Skill 目录默认使用 `.agents/skills/`，`target` 主要用于记录和过滤兼容目标，不再要求每个命令都绑定某个具体工具的目录说明。

#### 多仓库初始化流程
```bash
# 1. 初始化多仓库工作区（可指定初始仓库URL）
skill-hub init https://github.com/muidea/skill-hub-examples.git

# 2. 添加更多技能仓库（可选）
skill-hub repo add community https://github.com/community/skills.git
skill-hub repo add personal https://github.com/yourname/skills.git

# 3. 设置默认归档仓库
skill-hub repo default main

# 4. 设置项目兼容目标
skill-hub set-target open_code

# 5. 启用技能
skill-hub use git-expert

# 6. 应用技能到项目
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
```

#### 多仓库管理示例
```bash
# 查看所有仓库
skill-hub repo list

# 同步所有仓库
skill-hub repo sync

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
| `init` | `[git_url] [--target <value>]` | 初始化多仓库工作区 |
| `set-target` | `<value>` | 设置项目兼容目标 |
| `list` | `[--target <value>] [--verbose]` | 列出可用技能 |
| `search` | `<keyword> [--target <value>] [--limit <number>]` | 搜索远程技能 |
| `create` | `<id> [--target <value>]` | 创建新技能模板 |
| `remove` | `<id>` | 移除项目技能 |
| `validate` | `<id>` | 验证项目工作区中新建 skill 的合规性 |
| `use` | `<id> [--target <value>]` | 使用指定技能 |
| `status` | `[id] [--verbose]` | 检查技能状态 |
| `apply` | `[--dry-run] [--force]` | 应用技能到项目 |
| `feedback` | `<id> [--dry-run] [--force]` | 反馈修改到仓库 |
| `prune` | `无` | 清理 state.json 中失效的项目记录 |
| `pull` | `[--force] [--check]` | 拉取默认仓库的远程更新 |
| `push` | `[--message MESSAGE] [--force] [--dry-run]` | 推送默认仓库的本地更改 |

### 多仓库管理命令

| 命令 | 参数 | 功能说明 |
|------|------|----------|
| `repo add` | `<name> <git_url>` | 添加新技能仓库 |
| `repo list` | `无` | 列出所有技能仓库 |
| `repo remove` | `<name>` | 移除技能仓库 |
| `repo enable` | `<name>` | 启用技能仓库 |
| `repo disable` | `<name>` | 禁用技能仓库 |
| `repo default` | `<name>` | 设置默认（归档）仓库 |
| `repo sync` | `[name] [--all]` | 同步指定仓库或所有启用仓库 |

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
  - 支持的AI工具和兼容性说明
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
