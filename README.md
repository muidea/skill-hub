# Skill Hub

一款专为 AI 时代开发者设计的"技能（Prompt/Script）生命周期管理工具"。它旨在解决 AI 指令碎片化、跨工具同步难、缺乏版本控制等痛点。

## 核心理念

- **Git 为中心**：所有技能存储在Git仓库中，作为单一可信源
- **一键分发**：将技能快速应用到不同的AI工具
- **闭环反馈**：将项目中的手动修改反馈回技能仓库

## 功能特性

- **技能管理**：创建、查看、启用、禁用技能
- **变量支持**：技能模板支持变量替换
- **跨工具同步**：支持 Cursor、Claude Code、OpenCode 等AI工具
- **版本控制**：基于Git的技能版本管理
- **差异检测**：自动检测手动修改并支持反馈
- **安全操作**：原子文件写入和备份机制

## 快速开始

### 安装

```bash
# 从源码编译
git clone <repository-url>
cd skill-hub
make build
sudo make install

# 或直接使用二进制
./skill-hub --help
```

### 基本使用

1. **初始化工作区**
   ```bash
   skill-hub init
   ```

2. **查看可用技能**
   ```bash
   skill-hub list
   ```

3. **在当前项目启用技能**
   ```bash
   skill-hub use git-expert
   ```

4. **设置项目首选目标**
   ```bash
   skill-hub set-target open_code
   ```

5. **应用技能到项目**
   ```bash
   skill-hub apply
   ```

6. **检查技能状态**
   ```bash
   skill-hub status
   ```

7. **反馈手动修改**
   ```bash
   skill-hub feedback git-expert
   ```

## 命令参考

| 命令 | 描述 | 示例 |
|------|------|------|
| `init` | 初始化Skill Hub工作区 | `skill-hub init [git-url]` |
| `list` | 列出所有可用技能 | `skill-hub list` |
| `use` | 在当前项目启用技能 | `skill-hub use git-expert --target open_code` |
| `set-target` | 设置项目首选目标 | `skill-hub set-target open_code` |
| `apply` | 将技能应用到项目 | `skill-hub apply --dry-run` |
| `status` | 检查技能状态 | `skill-hub status` |
| `feedback` | 反馈手动修改 | `skill-hub feedback git-expert` |
| `update` | 更新技能仓库 | `skill-hub update` |
| `remove` | 从项目移除技能 | `skill-hub remove git-expert` |
| `git` | Git仓库操作 | `skill-hub git --help` |

### Git子命令

| 命令 | 描述 | 示例 |
|------|------|------|
| `git clone` | 克隆远程技能仓库 | `skill-hub git clone <url>` |
| `git sync` | 同步技能仓库 | `skill-hub git sync` |
| `git status` | 查看仓库状态 | `skill-hub git status` |
| `git commit` | 提交更改 | `skill-hub git commit` |
| `git push` | 推送更改 | `skill-hub git push` |
| `git remote` | 设置远程仓库 | `skill-hub git remote <url>` |

## 技能规范

### 目录结构

```
/skills
  /git-expert
    ├── skill.yaml       # 技能元数据
    ├── prompt.md        # 核心指令 (支持Go Template语法)
    └── scripts/         # (可选) 伴随执行的脚本
```

### skill.yaml 格式

```yaml
id: "git-expert"
name: "Git 提交专家"
version: "1.0.0"
author: "dev-team"
description: "根据变更自动生成符合 Conventional Commits 规范的说明"
tags: ["git", "workflow"]
compatibility:
  cursor: true
  claude_code: true
  open_code: true
variables:
  - name: "LANGUAGE"
    default: "zh-CN"
    description: "输出语言"
dependencies: []
```

### 模板变量

在 `prompt.md` 中使用 Go Template 语法：

```markdown
# 技能说明
语言: {{.LANGUAGE}}
```

## 支持的AI工具

| 工具 | 支持状态 | 配置文件位置 |
|------|----------|--------------|
| **Cursor** | ✅ 完全支持 | `~/.cursor/rules` |
| **Claude Code** | ✅ 完全支持 | `~/.claude/config.json` |
| **OpenCode** | ✅ 完全支持 | `~/.config/opencode/skills/` 或项目级 `.agents/skills/` |

## 项目状态管理

Skill Hub 使用状态文件跟踪项目与技能的关联：

```json
{
  "/path/to/project": {
    "project_path": "/path/to/project",
    "preferred_target": "open_code",
    "skills": {
      "web3-testing": {
        "skill_id": "web3-testing",
        "version": "1.0.0",
        "variables": {}
      }
    }
  }
}
```

## 贡献指南

欢迎提交Issue和Pull Request！

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启Pull Request

### 开发要求

- 遵循现有代码风格
- 添加适当的测试
- 更新相关文档
- 确保向后兼容性

## 构建和发布

### 本地构建

```bash
# 开发构建
make build

# 发布构建（所有平台）
make release-all VERSION=1.0.0

# 查看帮助
make help
```

### 自动发布

项目使用GitHub Actions实现自动发布：

1. **CI流程**：每次推送到main分支或PR时运行测试
2. **发布流程**：创建git标签时自动构建并发布预编译二进制

#### 使用发布脚本（推荐）：

```bash
# 使用发布助手脚本
./scripts/create-release.sh
```

#### 手动创建发布版本：

```bash
# 1. 确保代码是最新的
git pull origin main

# 2. 运行测试
make test

# 3. 创建标签
git tag -a v1.0.0 -m "Release v1.0.0"

# 4. 推送标签到GitHub
git push origin v1.0.0
```

GitHub Actions将自动：
- 为Linux (amd64/arm64)、macOS (amd64/arm64)、Windows (amd64/arm64)构建二进制
- 生成SHA256校验和
- 创建GitHub Release并上传所有文件

### 发布文件说明

每个发布版本包含以下文件：
- `skill-hub-{platform}-{arch}.tar.gz` - 压缩包（包含二进制、README、LICENSE）
- `skill-hub-{platform}-{arch}.sha256` - 校验和文件
- `checksums.txt` - 所有文件的校验和汇总

## CI/CD状态

[![CI](https://github.com/your-username/skill-hub/actions/workflows/ci.yml/badge.svg)](https://github.com/your-username/skill-hub/actions/workflows/ci.yml)
[![Release](https://github.com/your-username/skill-hub/actions/workflows/release.yml/badge.svg)](https://github.com/your-username/skill-hub/actions/workflows/release.yml)

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 问题反馈

如遇到问题或有功能建议，请：
1. 查看现有Issue是否已解决
2. 创建新的Issue，详细描述问题
3. 提供复现步骤和环境信息