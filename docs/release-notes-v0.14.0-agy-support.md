# v0.14.0 AGY 全局技能专项支持

## 新增能力

- 全局多 Agent 技能管理新增 `agy` 目标（支持 `codex`、`opencode`、`claude`、`agy`），服务与 HTTP API 输入支持将 `antigravity` 归一化为 `agy`。
- `ResolveAgent("agy")` 按 `AGY_SKILLS_DIR` > `AGY_CONFIG_HOME` > `AGY_HOME` > `config.yaml` 中可选 `agy_config_path` > 默认 `~/.gemini/config/skills` 顺序解析。
- 安全探测仅依据 AGY 特有标记（`~/.gemini/config/skills` 目录、`~/.gemini/config/{config.json,.migrated,skills.json}`、`~/.gemini/antigravity-cli` 目录或 `PATH` 中的 `agy` 可执行文件），不因普通 Gemini CLI 的 `~/.gemini` 目录误判。
- 全局 `apply --force` 冲突与本地修改备份统一外置至 `$SKILL_HUB_HOME/backups/global/<agent>/<skill-id>.<timestamp>`（跨设备时保留符号链接安全拷贝并清理原路径），保障 AGY 全局目录 1-level scan 清洁；非 `--force` 时对本地修改的托管目录同样拒绝覆盖并上报 `conflict`（`target_modified`）。
- 修复按 Agent 过滤执行全局 `apply` 时覆盖其他已启用 Agent 状态的问题，保留未选中的已启用 Agent 记录。

## 实现边界

- 保持既有 CLI 交互契约（`use`/`apply`/`status`/`remove` 的 `--global` 统一处理已检测或已配置的 Agent，不重新引入 `--agent` 旗标）与通用 `SKILL.md` 校验规则。
- 项目级继续沿用 `.agents/skills/<skill-id>/SKILL.md`，不生成冗余 `.agy/` 目录；全局模式不自动改写 `~/.gemini/config/skills.json`。
- `upgrade` 内置技能同步保持隔离，不写入 `AGY_SKILLS_DIR`、`AGY_CONFIG_HOME`、`AGY_HOME` 或 `~/.gemini`。

## 验证范围

- 单元测试覆盖防误判、各类探测标记与优先级、`agy_config_path`、1-level scan 孤儿识别与外置备份、符号链接拷贝、多 Agent 本地修改防护与单 Agent 移除状态保留、以及升级隔离断言。
- E2E 测试（`tests/e2e/test_global_skill_management.py`）覆盖 AGY 全局启用、未托管冲突拒绝覆盖、`--force` 外置备份、本地修改检测与重新同步及移除闭环。
- `make lint`、`make test`、`make build`、Windows/macOS 交叉编译及全局管理 E2E 回归全部通过。
