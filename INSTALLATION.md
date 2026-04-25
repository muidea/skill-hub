# skill-hub 详细安装和使用指南

> 如果您是首次使用，建议先查看 [README.md](README.md) 中的快速开始部分。

## 安装方法

skill-hub 提供多种安装方式，您可以根据自己的需求选择最适合的方法。

### 1. 一键安装脚本（最常用）

使用自动安装脚本下载并安装最新版本，这是最简单快捷的安装方式。

#### 使用自动安装脚本（推荐）

```bash
curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh | bash
```

**安装脚本特点**：
- 自动检测系统架构（Linux/macOS/Windows）
- 自动下载对应的预编译二进制文件
- 验证文件完整性（SHA256校验）
- 提供交互式安装选项
- 支持自动安装到系统路径
- 处理下载错误和网络问题

**脚本工作流程**：
1. 检测系统信息（操作系统和架构）
2. 获取最新版本号
3. 下载对应平台的压缩包和校验文件
4. 验证文件完整性
5. 解压文件
6. 提供交互式安装选项：
   - 安装到 `/usr/local/bin/`（需要sudo权限）
   - 安装到 `~/.local/bin/`（用户目录）
   - 跳过安装，仅保留在临时目录
7. 验证安装结果

### 2. 使用预编译二进制

如果您希望手动控制安装过程，可以从GitHub Releases页面下载预编译二进制文件。

#### 下载步骤

1. **访问 [GitHub Releases](https://github.com/muidea/skill-hub/releases)** 页面
2. **下载对应平台的压缩包**：
   - **Linux**:
     - `skill-hub-linux-amd64.tar.gz` (x86_64/AMD64架构)
     - `skill-hub-linux-arm64.tar.gz` (ARM64架构)
   - **macOS**:
     - `skill-hub-darwin-amd64.tar.gz` (Intel芯片)
     - `skill-hub-darwin-arm64.tar.gz` (Apple Silicon芯片)
   - **Windows**:
     - `skill-hub-windows-amd64.tar.gz` (x86_64架构)
     - `skill-hub-windows-arm64.tar.gz` (ARM64架构)

3. **解压并安装**：

   **Linux/macOS**:
   ```bash
   # 下载并解压
   tar -xzf skill-hub-linux-amd64.tar.gz
   
   # 安装到系统路径（需要sudo权限）
   sudo cp skill-hub /usr/local/bin/
   
   # 或安装到用户目录（无需sudo）
   cp skill-hub ~/.local/bin/
   
   # 确保目录在PATH中
   export PATH="$HOME/.local/bin:$PATH"
   
   # 验证安装
   skill-hub --version
   ```

   **Windows**:
   ```powershell
   # 解压文件
   tar -xzf skill-hub-windows-amd64.tar.gz
   
   # 将 skill-hub.exe 添加到系统 PATH
   # 或在解压目录中直接运行
   .\skill-hub.exe --help
   ```

#### 验证安装

安装完成后，运行以下命令验证安装：

```bash
skill-hub --version
skill-hub --help
```

### 3. 从源码编译

如果您需要自定义构建、参与开发或使用最新代码，可以从源码编译。

#### 环境要求

- **Go 1.24+** 环境
- **Git** 版本控制工具
- **make** 构建工具（可选，推荐）

#### 编译步骤

```bash
# 1. 克隆仓库
git clone https://github.com/muidea/skill-hub.git
cd skill-hub

# 2. 构建项目
make build
# 或直接使用 go build
# go build -o bin/skill-hub ./application/skill-hub/cmd

# 3. 安装到系统
sudo make install
# 或手动安装
# sudo cp bin/skill-hub /usr/local/bin/

# 4. 验证安装
skill-hub --version
```

#### 自定义构建选项

```bash
# 指定版本号构建
make build VERSION=1.0.0

# 构建所有平台的发布版本
make release-all VERSION=1.0.0

# 查看所有构建选项
make help
```

### 4. 本地开发安装

如果您已经在Skill Hub项目目录中（例如参与开发），可以使用本地安装脚本。

#### 使用本地安装脚本

```bash
# 运行本地安装脚本
./install-local.sh

# 脚本会自动：
# 1. 检测系统信息
# 2. 构建项目
# 3. 提供安装选项
# 4. 验证安装
```

#### 手动构建安装

```bash
# 构建项目
make build

# 安装到系统路径
sudo cp bin/skill-hub /usr/local/bin/

# 或安装到用户目录
cp bin/skill-hub ~/.local/bin/

# 验证安装
skill-hub --version
```

#### 开发环境验证

```bash
# 检查Go版本
go version

# 检查构建工具
make --version

# 运行测试
make test

# 运行完整测试套件
go test ./...
```

## 命令参考

skill-hub 提供了一系列命令来管理技能和项目。

### 核心命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `init` | 初始化多仓库工作区 | `skill-hub init [git_url]` |
| `list` | 列出可用技能 | `skill-hub list [--verbose] [--repo <repo-name>...]` |
| `search` | 搜索远程技能 | `skill-hub search <keyword> [--limit <number>]` |
| `create` | 创建新技能模板 | `skill-hub create <id>` |
| `remove` | 移除项目技能；`--global` 时移除本机全局技能 | `skill-hub remove <id> [--global] [--agent codex|opencode|claude] [--force]` |
| `validate` | 验证项目工作区中新建 skill 的合规性 | `skill-hub validate <id> [--fix] [--links] [--json]` |
| `use` | 使用指定技能；`--global` 时写入本机全局期望状态 | `skill-hub use <id> [--global] [--agent codex|opencode|claude]` |
| `status` | 检查项目或本机全局技能状态 | `skill-hub status [id] [--verbose] [--json] [--global] [--agent codex|opencode|claude]` |
| `apply` | 应用技能到项目；`--global` 时刷新本机 agent 全局 skills 目录 | `skill-hub apply [id] [--dry-run] [--force] [--global] [--agent codex|opencode|claude]` |
| `feedback` | 反馈项目修改到默认仓库 | `skill-hub feedback <id> [--dry-run] [--force] [--json]` |
| `prune` | 清理 `state.json` 中失效的项目记录 | `skill-hub prune` |
| `pull` | 拉取默认仓库的远程更新 | `skill-hub pull [--force] [--check] [--json]` |
| `push` | 推送默认仓库的本地更改 | `skill-hub push [--message MESSAGE] [--force] [--dry-run] [--json]` |

### 多仓库管理命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `repo add` | 添加新技能仓库 | `skill-hub repo add <name> <git_url>` |
| `repo list` | 列出所有技能仓库 | `skill-hub repo list` |
| `repo remove` | 移除技能仓库 | `skill-hub repo remove <name>` |
| `repo enable` | 启用技能仓库 | `skill-hub repo enable <name>` |
| `repo disable` | 禁用技能仓库 | `skill-hub repo disable <name>` |
| `repo default` | 设置默认（归档）仓库 | `skill-hub repo default <name>` |
| `repo sync` | 同步指定仓库或所有启用仓库 | `skill-hub repo sync [name] [--all]` |

### 常用工作流程

#### 初始化新项目（多仓库模式）
```bash
# 初始化多仓库工作区（可指定初始仓库URL）
skill-hub init https://github.com/muidea/skill-hub-examples.git

# 添加更多技能仓库（可选）
skill-hub repo add community https://github.com/community/skills.git
skill-hub repo add personal https://github.com/yourname/skills.git

# 设置默认归档仓库
skill-hub repo default main

# 项目本地技能统一应用到 .agents/skills/，不再需要设置 target
```

#### 启用和管理技能
```bash
# 查看可用技能
skill-hub list

# 启用技能
skill-hub use golang-best-practices

# 应用技能到项目
skill-hub apply

# 只刷新一个已启用技能，例如 status 显示 Outdated 时
skill-hub apply golang-best-practices

# 检查技能状态
skill-hub status

# 项目目录迁移或删除后，清理失效状态记录
skill-hub prune

# 显示详细信息
skill-hub list --verbose

# 按仓库过滤技能列表
skill-hub list --repo skills-repo

# 组合过滤选项
skill-hub list --repo skills-repo --verbose
```

#### 本机全局技能管理
```bash
# 将技能启用为 Codex 全局 skill
skill-hub use golang-best-practices --global --agent codex

# 检查本机全局期望状态与 agent skills 目录是否一致
skill-hub status --global
skill-hub status golang-best-practices --global --agent codex

# 预览并刷新本机 agent 全局 skills 目录
skill-hub apply --global --dry-run
skill-hub apply --global

# 从本机全局状态和 agent 全局目录移除
skill-hub remove golang-best-practices --global --agent codex
```

#### 技能反馈和更新
```bash
# 反馈手动修改
skill-hub feedback golang-best-practices

# 演习模式查看将要同步的差异
skill-hub feedback golang-best-practices --dry-run

# 从远程仓库拉取最新技能
skill-hub pull

# 检查可用更新但不实际执行拉取
skill-hub pull --check

# 推送本地更改到远程仓库
skill-hub push --message "修复技能描述"

# 移除不再需要的技能
skill-hub remove golang-best-practices
```

#### 技能创建和验证
```bash
# 从当前项目创建新技能模板
skill-hub create my-new-skill

# 在本地项目中验证技能有效性
skill-hub validate my-new-skill
```

## 技能管理

### 目录结构

skill-hub 使用标准的目录结构来组织技能：

```
/skills
  /git-expert                    # 技能目录（技能ID）
    ├── SKILL.md                # 技能元数据和内容（必需，Markdown + YAML frontmatter）
    ├── README.md               # 技能说明文档（可选）
    └── scripts/                # 伴随执行的脚本（可选）
        ├── setup.sh           # 安装脚本
        └── cleanup.sh         # 清理脚本
```

### SKILL.md 格式

每个技能必须包含一个 `SKILL.md` 文件，使用Markdown格式并包含YAML frontmatter定义技能的元数据和配置。

```markdown
---
name: git-expert              # 技能名称（必需）
description: Git 提交专家      # 技能描述（必需）
compatibility: Designed for Claude Code, Cursor, and OpenCode (or similar AI coding assistants) # 目标工具兼容性
metadata:                     # 元数据（可选）
  version: 1.0.0              # 版本号
  author: dev-team            # 作者/团队
  tags: git,workflow          # 标签，用于分类和搜索
---

# Git 提交专家

根据代码变更自动生成符合 Conventional Commits 规范的提交说明。

## 使用说明
1. 分析代码变更
2. 识别变更类型（feat, fix, docs, style, refactor, test, chore）
3. 生成简洁明了的提交说明

## 示例
当检测到新功能时，生成：
feat: 添加用户登录功能

当修复bug时，生成：
fix: 修复登录页面样式错位问题
```

## 支持的AI工具

skill-hub 支持多种AI开发工具，可以将技能同步到不同的工具配置中。

### Cursor
- **支持状态**: ✅ 完全支持
- **配置文件位置**:
  - 用户级: `~/.cursor/rules`
  - 项目级: `.cursorrules`
- **同步方式**: 将技能内容写入Cursor规则文件
- **特点**: 实时同步，支持项目级配置

### Claude Code
- **支持状态**: ✅ 完全支持
- **配置文件位置**:
  - 用户级: `~/.claude/config.json`
  - 项目级: `.clauderc`
- **同步方式**: 更新Claude配置文件中的指令部分
- **特点**: JSON格式配置，支持复杂结构

### OpenCode
- **支持状态**: ✅ 完全支持
- **配置文件位置**:
  - 用户级: `~/.config/opencode/skills/`
  - 项目级: `.agents/skills/`
- **同步方式**: 创建技能文件在技能目录中
- **特点**: 文件系统存储，易于管理

### 工具兼容性说明

1. **配置优先级**: 项目级配置优先于用户级配置
2. **原子操作**: 所有文件写入都是原子操作，确保配置安全
3. **备份机制**: 修改前自动备份原配置，支持回滚
4. **差异检测**: 自动检测手动修改，支持反馈到技能仓库

### 多工具同步示例

```bash
# 启用技能
skill-hub use git-expert

# 应用技能到项目
skill-hub apply

# 只刷新一个已启用技能，例如 status 显示 Outdated 时
skill-hub apply git-expert

# 演习模式查看将要进行的变更
skill-hub apply --dry-run

# 检查各工具的配置状态
skill-hub status

# 显示详细差异信息
skill-hub status --verbose

# 刷新本机全局技能
skill-hub use git-expert --global --agent opencode
skill-hub apply --global
```

## 故障排除

### 常见问题

#### 1. 安装脚本返回404错误
```bash
# 错误信息: bash: line 1: 404:: command not found
# 可能原因: GitHub API限制或网络问题
# 解决方案:
# 1. 等待几分钟后重试
# 2. 检查网络连接
# 3. 手动从GitHub Releases页面下载
```

#### 2. 权限不足无法安装
```bash
# 错误信息: Permission denied
# 解决方案: 使用sudo或安装到用户目录
sudo cp skill-hub /usr/local/bin/
# 或
cp skill-hub ~/.local/bin/
```

#### 3. 命令未找到
```bash
# 错误信息: command not found: skill-hub
# 解决方案: 确保安装目录在PATH中
export PATH="$HOME/.local/bin:$PATH"
# 或重新登录使PATH生效
```

#### 4. 技能同步失败
```bash
# 检查目标工具是否安装
# 检查配置文件权限
# 使用演习模式预览更改
skill-hub apply --dry-run

# 强制应用，即使检测到冲突也继续执行
skill-hub apply --force
```

### 获取帮助

```bash
# 查看所有命令
skill-hub --help

# 查看特定命令帮助
skill-hub init --help
skill-hub use --help
skill-hub apply --help
skill-hub feedback --help
skill-hub pull --help
skill-hub push --help

# 查看版本信息
skill-hub --version
```

### 报告问题

如果遇到无法解决的问题，请：
1. 查看 [GitHub Issues](https://github.com/muidea/skill-hub/issues)
2. 创建新的Issue，包含：
   - 错误信息
   - 复现步骤
   - 系统环境信息
   - 期望的行为

## 下一步

- 学习如何创建自定义技能，请参考技能开发文档
- 参与项目开发，请查看 [DEVELOPMENT.md](DEVELOPMENT.md)
- 返回主文档 [README.md](README.md)
