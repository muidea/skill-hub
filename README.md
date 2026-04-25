# skill-hub

`skill-hub` 是面向 AI 编程工具的 skill 生命周期管理工具。它用 Git 仓库管理可复用的 Prompt、脚本和工作流说明，并把这些 skill 分发到项目工作区或本机全局 agent skills 目录。

[![CI](https://github.com/muidea/skill-hub/actions/workflows/ci.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/ci.yml)
[![Tests](https://github.com/muidea/skill-hub/actions/workflows/test.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/test.yml)
[![Release](https://github.com/muidea/skill-hub/actions/workflows/release.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/release.yml)

## 能做什么

- 管理多个 skill 仓库，支持查看、搜索、同步和切换默认归档仓库。
- 在项目中启用 skill，并应用到标准目录 `.agents/skills/`。
- 检查项目 skill 是否与来源仓库一致，并刷新 `Outdated` 的本地副本。
- 将项目中改进过的 skill 反馈回默认仓库，形成 `use -> apply -> feedback -> push` 闭环。
- 将托管 skill 启用到本机全局 agent skills 目录，服务 Codex、OpenCode、Claude 等工具。
- 支持已有 skill 登记、批量导入、验证、重复副本检测、路径可移植性审计和刷新审计报告。
- 支持本地 `serve` 模式，为 CLI 和 Web 管理提供统一的本机执行入口。
- 支持从 GitHub Releases 检测和升级已安装的 `skill-hub`。

## 安装

推荐使用一键安装脚本：

```bash
curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh | bash
```

安装后检查版本：

```bash
skill-hub --version
```

已安装用户可检测和升级到最新 Release：

```bash
skill-hub upgrade --check
skill-hub upgrade --yes
```

更多安装方式、环境变量和故障排查见 [INSTALLATION.md](INSTALLATION.md)。

## 快速开始

在需要使用 skill 的项目目录中初始化：

```bash
skill-hub init
```

同步仓库并查找可用 skill：

```bash
skill-hub repo sync --json
skill-hub list
skill-hub search git
```

启用并应用 skill：

```bash
skill-hub use git-expert
skill-hub apply
skill-hub status
```

只刷新一个已启用 skill：

```bash
skill-hub apply git-expert
skill-hub status git-expert
```

将项目中的改进反馈回默认仓库：

```bash
skill-hub feedback git-expert --dry-run
skill-hub feedback git-expert --force
skill-hub push --dry-run --json
```

## 本机全局 Skill

如果希望某个 skill 对本机 agent 全局可用，可以使用 `--global`：

```bash
skill-hub use git-expert --global --agent codex
skill-hub status --global
skill-hub apply --global --dry-run
skill-hub apply --global
```

`use --global` 只记录期望状态，`apply --global` 才会刷新本机 agent 全局 skills 目录。

## 常用工作流

创建新 skill：

```bash
skill-hub create my-skill
skill-hub validate my-skill --links
skill-hub feedback my-skill --force
```

登记已有项目 skill：

```bash
skill-hub register existing-skill
skill-hub validate existing-skill --fix
```

批量导入和归档：

```bash
skill-hub import .agents/skills --fix-frontmatter --archive --force
```

检查重复副本和路径可移植性：

```bash
skill-hub dedupe . --canonical .agents/skills --json
skill-hub lint . --paths --project-root "$PWD" --json
```

启动本地服务：

```bash
skill-hub serve
```

## 文档

- [安装和使用指南](INSTALLATION.md)
- [命令规范](docs/Skill-Hub命令规范.md)
- [开发指南](DEVELOPMENT.md)

历史版本变更请查看 [GitHub Releases](https://github.com/muidea/skill-hub/releases)。仓库内可能保留用于发布流程的版本说明文档，但 README 只维护长期有效的入口文档。

## 链接

- [GitHub 仓库](https://github.com/muidea/skill-hub)
- [Releases](https://github.com/muidea/skill-hub/releases)
- [Issues](https://github.com/muidea/skill-hub/issues)

## 许可证

MIT License，详见 [LICENSE](LICENSE)。
