# skill-hub

`skill-hub` 是面向 AI 编程工具的 skill 生命周期管理工具。它用 Git 仓库管理可复用的 Prompt、脚本和工作流说明，并把这些 skill 分发到项目工作区或本机全局 agent skills 目录。

[![CI](https://github.com/muidea/skill-hub/actions/workflows/ci.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/ci.yml)
[![Tests](https://github.com/muidea/skill-hub/actions/workflows/test.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/test.yml)
[![Release](https://github.com/muidea/skill-hub/actions/workflows/release.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/release.yml)

## 能做什么

- 管理多个 skill 仓库，支持查看、搜索、同步和切换默认归档仓库。
- 在项目中启用 skill，并应用到标准目录 `.agents/skills/`。
- 检查项目 skill 是否与来源仓库一致，识别基于旧版本发生的本地修改，并刷新 `Outdated` 的本地副本。
- 将项目中改进过的 skill 反馈回默认仓库；当工作区版本低于仓库版本时会先阻断反馈，避免旧副本覆盖高版本仓库内容。
- 将托管 skill 启用到本机全局 agent skills 目录，服务 Codex、OpenCode、Claude 等工具。
- 支持已有 skill 登记、批量导入、验证、重复副本检测、路径可移植性审计和刷新审计报告。
- 提供独立 `skill-hubd` daemon，为 CLI 和 Web 管理提供统一的本机执行入口。
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
skill-hub apply --pattern git-expert
skill-hub status --pattern git-expert
```

将项目中的改进反馈回默认仓库：

```bash
skill-hub feedback --pattern git-expert --dry-run
skill-hub feedback --pattern git-expert --force
skill-hub push --dry-run --json
```

`feedback` 只用于标准项目工作区 `.agents/skills/<id>`。发布包内置或其他批量 skill 目录使用 `import --archive --archive-only` 归档，不要手工复制到 `~/.skill-hub/repositories/.../skills`。

如果 `status` 显示 `ModifiedAgainstOutdatedRepo`，先执行 `skill-hub apply --pattern <id>` 刷新到仓库高版本，再把必要修改迁移到新版本后反馈。

## Agent 与自动化调用

CLI 客户端统一使用 `--json`：stdout 只返回一个完整 JSON 文档，进度与诊断写入 stderr，执行期间不读取 stdin。调用方应分别捕获两个通道，并检查退出码；失败和部分失败会在结果中保留 `command_error`。

```bash
skill-hub init --json
skill-hub project register --request-id project-setup-001 --json
# 回复丢失时，以原请求 ID 查询
skill-hub project register --query --request-id project-setup-001 --json
skill-hub register my-skill --json
skill-hub import .agents/skills --dry-run --json
skill-hub status --json
```

`--json` 不自动确认删除或覆盖；例如移除项目技能需要 `remove <id> --yes --json`。完整规则见 [CLI JSON 调用合同](docs/Skill-Hub命令规范.md#cli-json-调用合同)，回归用例与运行方式见 [e2e 文档](tests/e2e/README.md#cli-json-regression-selection)。

`project register [path]` 显式登记空项目，无需创建占位技能；重复登记已有项目保留其技能状态。调用方须先保存 `--request-id`：同一 Hub home 内，同 ID、同绝对路径重放原回执，不重新执行；同 ID、不同路径返回 `REQUEST_CONFLICT`。回执保留原请求的 `status`、`executed`、`finished` 与 `result`（是否新建项目、创建的目录和当时技能数），不随后续项目变化而改变。

`--query` 只读取原回执，允许从其他目录查询，且不要求当前项目或配置文件仍然存在。查询成功的退出码为 0，原请求是否成功须读取 `status`；未找到回执返回 `REQUEST_NOT_FOUND`。异常中断后没有最终回执时，返回 `unknown`，`executed` 和 `finished` 为 `null`，相同 ID 不会重新执行。回执保存在 `${SKILL_HUB_HOME:-~/.skill-hub}/project-registrations/`；保留该目录和原请求 ID 才能继续查询，v0.13.6 及以前未记录的请求无法补查。详细约定见 [空项目登记与请求回执](docs/Skill-Hub命令规范.md#42-project-register---空项目登记与请求回执)。

## 本机全局 Skill

如果希望某个 skill 对本机 agent 全局可用，可以使用 `--global`：

```bash
skill-hub use git-expert --global
skill-hub status --global
skill-hub apply --global --dry-run
skill-hub apply --global
```

`use --global` 只记录期望状态，`apply --global` 才会刷新本机 agent 全局 skills 目录。

## 常用工作流

创建新 skill：

```bash
skill-hub create my-skill
skill-hub validate --pattern my-skill --links
skill-hub feedback --pattern my-skill --force
```

登记已有项目 skill：

```bash
skill-hub register existing-skill
skill-hub validate --pattern existing-skill --fix
```

批量导入和归档：

```bash
skill-hub import .agents/skills --fix-frontmatter --archive --force
skill-hub import agent-skills --archive --archive-only --force
skill-hub repo rebuild-index
```

检查重复副本和路径可移植性：

```bash
skill-hub dedupe . --canonical .agents/skills --json
skill-hub lint . --paths --project-root "$PWD" --json
```

启动本地服务：

```bash
skill-hubd
```

在 Linux 上将 daemon 注册为当前用户的 systemd 服务：

```bash
skill-hubd service install --port 5525
skill-hubd service status
skill-hubd service restart
```

`skill-hubd service` 管理的是 systemd 用户单元；不再使用已移除的 `skill-hub serve` 或 `~/.skill-hub/services.json`。

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
