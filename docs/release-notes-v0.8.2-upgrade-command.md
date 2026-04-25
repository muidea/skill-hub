# v0.8.2 Upgrade Command

发布日期：2026-04-25

## 变更摘要

- 新增 `skill-hub upgrade` 自升级命令，支持 `--check`、`--yes`、`--version`、`--dry-run`、`--force`、`--json`、`--skip-agent-skills` 和 `--no-restart-serve`。
- `upgrade` 从 GitHub Releases 检测最新版本，按当前系统架构下载 `.tar.gz` 发布包和 `.sha256` 文件，校验压缩包后替换当前二进制。
- Linux 与 macOS 支持自动替换当前运行中的二进制；Windows 当前返回明确提示，继续使用安装脚本或手动下载 Release 包。
- 升级完成后默认同步 release 内置 `skill-hub-*` agent workflow skills，并尝试重启已注册且正在运行的 `serve` 实例。
- 修正 `scripts/install-latest.sh` 的校验语义：安装脚本现在校验 Release 压缩包 sha256，与 Makefile 生成的 `.sha256` 文件保持一致。

## 验证

- 新增 upgrade service 单元测试，覆盖 latest release 检测、下载校验替换、dry-run 不写文件、archive sha256 校验。
- 更新安装脚本测试，确保校验对象为 `.tar.gz` 发布包，而不是解压后的二进制。
