# skill-hub 🚀

一款专为 AI 时代开发者设计的"技能（Prompt/Script）生命周期管理工具"。它旨在解决 AI 指令碎片化、跨工具同步难、缺乏版本控制等痛点。

[![CI](https://github.com/muidea/skill-hub/actions/workflows/ci.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/ci.yml)
[![Tests](https://github.com/muidea/skill-hub/actions/workflows/test.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/test.yml)
[![Release](https://github.com/muidea/skill-hub/actions/workflows/release.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/release.yml)

## 简介

### 核心理念

- **Git 为中心**：所有技能存储在Git仓库中，作为单一可信源
- **一键分发**：将技能快速应用到不同的AI工具
- **闭环反馈**：将项目中的手动修改反馈回技能仓库
- **现代架构**：采用 Go 1.24+ 特性，遵循 Effective Go 最佳实践

### 功能特性

- **技能管理**：查看、启用、禁用技能
- **技能创建**：从当前项目创建新的技能模板
- **本地验证**：在本地项目中验证技能有效性
- **技能归档**：将验证通过的技能归档到正式仓库
- **跨工具同步**：支持 Cursor、Claude Code、OpenCode 等AI工具
- **版本控制**：基于Git的技能版本管理
- **差异检测**：自动检测手动修改并支持反馈
- **安全操作**：原子文件写入和备份机制
- **全面测试**：单元测试 + 端到端测试覆盖

## 🚀 快速开始

### 安装

使用一键安装脚本（最简单的方式）：

```bash
curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh | bash
```

### 基本使用

安装完成后，按照以下工作流程开始使用：

#### 基础使用流程
```bash
# 1. 初始化工作区
skill-hub init

# 2. 启用技能并设置目标
skill-hub use git-expert

# 3. 应用技能到项目
skill-hub apply
```

#### 技能创建与验证流程
```bash
# 1. 从当前项目创建新技能模板
skill-hub create my-new-skill

# 2. 在本地项目中验证技能有效性
skill-hub validate my-new-skill

# 3. 反馈手动修改并归档技能
skill-hub feedback my-new-skill
```

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

### CI/CD状态

[![CI](https://github.com/muidea/skill-hub/actions/workflows/ci.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/ci.yml)
[![Tests](https://github.com/muidea/skill-hub/actions/workflows/test.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/test.yml)
[![Release](https://github.com/muidea/skill-hub/actions/workflows/release.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/release.yml)

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