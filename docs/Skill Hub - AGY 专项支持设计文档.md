# Skill Hub - AGY 专项支持设计文档

## 1. 背景与目标

`skill-hub` 目前在跨 Agent 全局技能管理中已支持 `codex`、`opencode`、`claude` 三个目标（见 `DetectAgents` 与 `ResolveAgent`，位于 `internal/modules/application/global/service/global.go`），在项目级支持统一的 `.agents/skills/<skill-id>/SKILL.md` 工作区规范（见 `ProjectLifecycle`，位于 `internal/modules/application/project_lifecycle/service/lifecycle.go`）。

Antigravity CLI（下文简称 AGY，当前实测基线为 `1.5.16`）已作为独立 AI 编程代理广泛使用。为让 `skill-hub` 在不破坏现有 Gemini CLI 与现有三大 Agent 兼容性的前提下，完整支持 AGY 的项目级与全局级技能发现、分发、校验、状态比对与升级隔离，特编写本专项设计文档。

本文档仅定义架构、目录合同、安全检测机制、代码改造范围与验收标准；**不涉及本次同步修改任何 Go/Python 源码**。

---

## 2. AGY 真实技能机制调研与事实基线

结合本机 AGY `1.5.16` 内置文档与宿主目录实测：

1. **项目级技能发现（Project Scope）**：
   - AGY 原生沿项目根目录向上遍历至 `.git` 搜索 `.agents/skills/<skill-id>/SKILL.md`。
   - 这与 `skill-hub` 当前的 `.agents/skills/` 项目目录约定完全一致（`ProjectLifecycle.Register` 与 `project_apply` 已直接操作该路径），项目级复用无需改动现有目录命名。
2. **全局级技能与配置布局（Global Scope）**：
   - **全局技能目录**：AGY 全局用户配置位于 `~/.gemini/config/`，技能读取 `~/.gemini/config/skills/<skill-id>/SKILL.md`。
   - **一级扁平扫描约束（关键差异）**：根据 `agy-customizations` 对 `skills.md` 的规范，AGY 对 `~/.gemini/config/skills/` 执行 **1-level 扫描**——即直接遍历该目录下的第一层子目录并尝试解析其中的技能声明。
   - **CLI 运行态目录**：AGY CLI 运行时、内置技能与用户运行时状态位于 `~/.gemini/antigravity-cli/`；`~/.gemini/config/` 存放用户配置清单（如 `.migrated`、`config.json`、`mcp_config.json`、`skills.json` 等）。
3. **与普通 Gemini CLI 共存风险**：
   - 普通 Gemini CLI 同样使用 `~/.gemini/` 根目录，但并不包含 AGY 的 `~/.gemini/config/` 与 `~/.gemini/antigravity-cli/` 结构。
   - 因此**严禁**仅凭 `~/.gemini` 目录存在就判定宿主安装/启用了 `agy`。

---

## 3. 核心设计方案

### 3.1 目标标识与兼容性声明

- **Agent 名称**：`agy`（CLI `--agent agy`、状态文件 `global-state.json` 及 `Manifest.Agent` 统一使用规范值 `agy`；可在输入解析层可选接受 `antigravity` 作为 CLI 别名并归一化为 `agy`，内部存储与清单严格保持单一规范名称 `agy`）。
- **Spec 目标常量与校验声明**：
  - 在 `pkg/spec/skill.go` 新增目标常量 `TargetAgy = "agy"`（与 `TargetCursor`、`TargetClaudeCode`、`TargetOpenCode` 并列）。
  - 在 `pkg/validator/rules.go` 的 `compatibility` 检查与作者提示中支持 `agy` / `antigravity` 描述说明（保持现有通用规则与 `SKILL.md` 规范，不引入破坏性版本字段）。

### 3.2 目录解析与环境变量约定（`ResolveAgent`）

在 `internal/modules/application/global/service/global.go` 中扩展 `DetectAgents` 与 `ResolveAgent` 候选列表，从 `["codex", "opencode", "claude"]` 扩展为 `["codex", "opencode", "claude", "agy"]`：

1. **环境变量优先级**（沿用既有 `CODEX_SKILLS_DIR`/`CODEX_HOME`、`OPENCODE_SKILLS_DIR`/`OPENCODE_HOME` 模式）：
   - 第一优先级：`AGY_SKILLS_DIR` — 直接作为目标技能根目录（例如用于隔离单测或自定义路径）。
   - 第二优先级：`AGY_CONFIG_HOME` — 指向 AGY 配置根目录，解析为 `filepath.Join(agyConfigHome, "skills")`（对应 `~/.gemini/config/skills`）。
   - 第三优先级：`AGY_HOME` — 指向 AGY/Gemini 主目录：
     - 若 `$AGY_HOME/config` 目录已存在，或路径基名为 `.gemini`，解析为 `filepath.Join(agyHome, "config", "skills")`；
     - 否则解析为 `filepath.Join(agyHome, "skills")`（避免测试注入专用根目录时出现多余嵌套），同时在设计中约定优先使用 `AGY_SKILLS_DIR` 或 `AGY_CONFIG_HOME` 以消除歧义。
   - 第四优先级（配置文件显式覆盖，可选）：在 `internal/config/config.go` 的 `Config` 结构体中预留 `AgyConfigPath`（`mapstructure:"agy_config_path"`），默认空时走标准路径。
   - 第五优先级（默认宿主路径）：`filepath.Join(homeDir, ".gemini", "config", "skills")`。

### 3.3 安全探测规则（`DetectAgents` — 禁止仅凭 `~/.gemini` 误判）

为了避免在仅使用标准 Gemini CLI 的机器上误激活 `agy` 全局同步，`DetectAgents` 中的 `agy` 探测必须满足**以下任一明确证据**：

1. **显式环境变量**：`AGY_SKILLS_DIR`、`AGY_CONFIG_HOME` 或 `AGY_HOME` 任一非空；
2. **显式配置**：`Config.AgyConfigPath` 非空；
3. **AGY 特有目录标记**（至少具备其一，绝不单独检查 `~/.gemini`）：
   - `~/.gemini/config/skills` 已存在且为目录；
   - `~/.gemini/config/config.json`、`~/.gemini/config/.migrated` 或 `~/.gemini/config/skills.json` 存在；
   - `~/.gemini/antigravity-cli` 目录存在；
4. **CLI 二进制可达**（可选辅助信号）：通过 `exec.LookPath("agy")` 找到可执行文件（可在未首次打开设置前识别新安装 CLI）。

`AgentInfo.Reason` 返回清晰的可观测依据（如 `"env:AGY_SKILLS_DIR"`、`"agy_config_dir"`、`"agy_cli_dir"`、`"path:agy"` 等），当未探测到时 `Detected = false`、`Configured = false`、`Reason = "not_detected"`。

### 3.4 解决 1-Level Scan 与备份目录冲突（`backupAndRemove`）

**现有隐患**：在当前 `internal/modules/application/global/service/global.go`（约第 593–616 行）中，当 `applyToAgent` 遇到同名未托管冲突目录且携带 `--force` 时，调用 `backupAndRemove(targetDir)` 在同一父目录下生成：

`<targetDir>.skill-hub-backup.<timestamp>`（即 `~/.gemini/config/skills/<skill-id>.skill-hub-backup.<ts>`）。

对于 `claude`/`codex` 等非严格扫描或忽略特定模式的工具该做法问题较小，但在 AGY 全局目录下，`~/.gemini/config/skills/` 为 **1-level scan**，残留的 `.skill-hub-backup.<ts>` 目录同样是一级子目录，会导致 AGY 扫描并报错/加载失效副本或重复技能条目。

**改造方案**：

1. **备份目录外置**：当检测到目标 Agent 的 `SkillsDir` 对同级备份敏感（或统一改造 `global` 服务的全局备份位置）时，将冲突备份移至 `SkillsDir` 外部的受管备份根：
   `~/.skill-hub/backups/global/<agent>/<skill-id>.<timestamp>/`（若设置了 `SKILL_HUB_HOME`，则为 `$SKILL_HUB_HOME/backups/global/...`）。
2. **严格目录清洁度**：确保写入 `~/.gemini/config/skills/` 下的每个直接子目录均为合法的 `<skill-id>/`（内含 `SKILL.md`、`references/` 与 `ManifestFileName` 即 `.skill-hub-manifest.json`），不在 `SkillsDir` 同级目录遗留任何 `*.skill-hub-backup.*`、`.tmp` 或状态文件。
3. **只读识别安全**：`global status --global` 检查 `agy` 目录时，忽略非目录条目并仅把含 `.skill-hub-manifest.json` 或受管状态的目录识别为受管/冲突/孤儿项。

### 3.5 `upgrade` 捆绑技能与 AGY 目录隔离

在 `internal/modules/application/upgrade/service/upgrade.go`（`installBundledAgentSkills` 及配套 `upgrade_test.go`）中：

- 当前 `upgrade_test.go` 强制断言捆绑升级路径不会写入用户的受管全局 Agent 目录（通过设置隔离 env 验证 `CODEX_SKILLS_DIR`、`OPENCODE_SKILLS_DIR`、`CLAUDE_SKILLS_DIR` 未被修改）。
- 新增 `agy` 后，必须将该隔离契约扩展到 `AGY_SKILLS_DIR`、`AGY_CONFIG_HOME` 与 `AGY_HOME`：
  - `skill-hub upgrade` 仅写入自身或显式允许的专属目录，不得越权直接写入 `~/.gemini/config/skills/` 绕过用户的 `global use` / `global apply` 控制流。
  - 若未来需要为 AGY 刷新内置技能，只能归档至默认仓库再经标准 `global` 管理路径显式 `apply`。

### 3.6 可选配置协同边界（`~/.gemini/config/skills.json`）

根据 AGY 自定义规范，`~/.gemini/config/skills.json` 可用于配置启用/禁用或扩展声明。`skill-hub` 对其定位如下：

- **第一阶段（核心闭环）**：以 `~/.gemini/config/skills/<skill-id>/SKILL.md` 文件同步 + `.skill-hub-manifest.json` 为唯一控制事实，不强行改写 `~/.gemini/config/skills.json`，以免覆盖用户通过 `/skills` UI 手动切换的局部开关或注释/格式化配置。
- **扩展预留**：若需在状态诊断提示中提醒用户在 `skills.json` 显式禁用某全局技能，应在 `status --global --json` 中保持只读观测，不静默修改 `skills.json`。

---

## 4. 模块与代码改动清单（实施蓝图）

| 模块/文件 | 改动内容 | 职责说明 |
| --- | --- | --- |
| `internal/modules/application/global/service/global.go` | 在 `DetectAgents` / `ResolveAgent` 加入 `agy`、环境变量解析、`Detect` 防误判逻辑；改造 `backupAndRemove` 至 `$SKILL_HUB_HOME/backups/global/<agent>/` | 拥有全局多 Agent 同步、备份外置与状态比对权威实现 |
| `internal/config/config.go` | 为 `Config` 增加可选字段 `AgyConfigPath` | 支持配置层显式指定 AGY 配置路径 |
| `pkg/spec/skill.go` | 新增 `TargetAgy = "agy"` | 统一目标常量定义 |
| `pkg/validator/rules.go` | 在兼容目标与提示规则中纳入 `agy` 描述对齐 | 保持 `SKILL.md` 校验规则同步 |
| `internal/clis/skill-hub/`（`use.go` / `apply.go` / `status.go` / `completer.go`） | CLI 帮助文本、Shell 自动补全候选列表加入 `agy` | 无需改变交互契约，补齐 CLI 发现与 `--agent agy` 自动补全 |
| `internal/modules/application/upgrade/service/upgrade_test.go` | 增加 `AGY_SKILLS_DIR` / `AGY_CONFIG_HOME` / `AGY_HOME` 未被捆绑安装污染的隔离断言 | 守住升级隔离红线 |
| `internal/modules/application/global/service/global_test.go` | 新增 AGY 探测（正例/反例：仅 `~/.gemini` 不触发）、环境变量优先级、1-level 目录外置备份、冲突 `--force`、状态比对（`synced`/`modified`/`outdated`/`conflict`/`missing`）测试 | 核心单测覆盖 |

---

## 5. CLI 交互与状态输出设计

扩展后无需引入新子命令，完全复用既有 CLI 与 `--json` 契约：

- `skill-hub apply --global [--agent agy] [--dry-run] [--force]`：仅同步已开启 `use --global` 的技能到 `~/.gemini/config/skills/<id>/`。
- `skill-hub status --global [--json]`：`agents` 数组新增 `{"name": "agy", "skills_dir": "...", "detected": true, "configured": true, "reason": "..."}`；`items` 逐项报告每个 `skill_id` 在 `agy` 下的 `synced`/`modified`/`outdated`/`missing`/`conflict`/`orphaned`。
- `skill-hub remove <id> --global` / 停用并执行 `apply`：精准清理 `agy` 下带 `.skill-hub-manifest.json` 的托管目录，绝不删除非 `skill-hub` 创建的原生技能目录。

---

## 6. 验收准则与测试计划

对照 `docs/Skill Hub 验收准则.md`，AGY 专项支持完成需满足以下可观测验证项：

1. **防误判验证**：在仅存在空 `~/.gemini/`（无 `config/`、无 `antigravity-cli/`、无 `agy` 二进制且无 AGY 环境变量）的临时 `$HOME` 中运行 `DetectAgents`，断言 `agy.Detected == false`。
2. **精准探测与优先级验证**：
   - `AGY_SKILLS_DIR` 覆盖 `AGY_CONFIG_HOME`、`AGY_HOME` 与默认路径；
   - `~/.gemini/antigravity-cli/` 或 `~/.gemini/config/.migrated` 存在时默认解析为 `~/.gemini/config/skills`。
3. **1-Level Scan 保护验证**：在 `SkillsDir` 已存在未托管同名目录 `<id>/` 时执行 `apply --global --force --agent agy`：
   - 原目录被迁移到外部备份目录（`$SKILL_HUB_HOME/backups/global/agy/<id>.<ts>`）；
   - `SkillsDir` 内**不存在** `*.skill-hub-backup.*` 残留子目录；
   - 新写入的 `<SkillsDir>/<id>/SKILL.md` 与 `.skill-hub-manifest.json` 完整且 `status --global --json` 为 `synced`。
4. **多 Agent 一致性验证**：与 `codex`、`opencode`、`claude` 并行启用和卸载（`remove`），互不干扰；对 `agy` 副本做本地修改后，`status` 正确上报 `modified`，`apply --global` 在未带 `--force` 时拒绝覆盖、带 `--force` 时外部备份并重新同步。
5. **项目级共享验证**：确认 `.agents/skills/<id>/SKILL.md` 下项目级 `init` / `use` / `apply` / `import` 正常工作，无冗余 `.agy/` 目录产生。
6. **回归套件通过**：在 `skill-hub` 根目录执行 `go test ./... -count=1` 全部通过，且升级隔离测试 `upgrade_test.go` 明确验证不会改写 AGY 受管全局目录。
