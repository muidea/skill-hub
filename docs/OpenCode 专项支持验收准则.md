# skill-hub - OpenCode 专项支持验收准则 (v1.0)

本准则旨在对 **skill-hub** 针对 **OpenCode** 技能体系的支持能力进行全面验证。验收范围涵盖了从初始化、项目绑定、分发部署到闭环反馈及清理的完整生命周期场景。

---

## 模块 1：项目绑定与上下文感知 (Context & Binding)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 1.1 | **首选目标绑定** | 执行 `skill-hub set-target open_code` 后，`state.json` 必须准确记录当前项目路径与 `open_code` 的绑定关系。 | 运行指令后检查 `~/.skill-hub/state.json` 内容。 |
| 1.2 | **自动特征探测** | 在包含 `.agents/skills/` 文件夹的项目中运行 `apply`（未显式绑定时），工具应能通过目录特征辅助识别目标为 `open_code`。 | 在新项目创建 `.agents/skills/` 目录并运行 `apply`，观察输出。 |
| 1.3 | **子目录识别逻辑** | 在项目深层子目录下运行 `apply`，必须能准确继承根目录绑定的 `open_code` 设置并操作根目录下的 `.agents/skills/` 文件夹。 | 在 `[ProjectRoot]/src/test/` 下运行 `apply`。 |

---

## 模块 2：分发与部署逻辑 (Apply/Distribution)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 2.1 | **目录结构生成** | 执行 `apply` 后，项目根目录必须生成 `.agents/skills/[skill-id]/` 结构。 | 检查物理文件路径是否存在。 |
| 2.2 | **元数据转换** | 生成的 `manifest.yaml` 必须包含正确的 `name`、`version` 且符合 OpenCode 官方 Schema 规范。 | 使用 YAML 校验工具或 OpenCode Agent 加载。 |
| 2.3 | **指令渲染** | `instructions.md` 必须由仓库中的 `SKILL.md` 渲染而成，且 `{{.VAR}}` 变量被正确替换。 | 在 `SKILL.md` 定义变量并在应用后检查文件。 |
| 2.4 | **原子性操作** | 若 `apply` 过程中断，已存在的 `.agents/skills/` 目录不应处于损坏状态（通过临时目录或备份机制）。 | 模拟写入大文件时中断。 |

---

## 3. 闭环反馈逻辑 (Feedback Loop)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 3.1 | **变更状态识别** | 手动修改 `.agents/skills/[id]/SKILL.md` 后，执行 `skill-hub status` 必须显示该 Skill 为 `Modified` 状态。 | 手动改动文字后运行指令。 |
| 3.2 | **精确提取回传** | 执行 `feedback` 必须能完整提取 `SKILL.md` 的内容并原样同步回 Git 仓库的 `SKILL.md`。 | 运行 `feedback` 并检查本地 Git 仓库文件。 |
| 3.3 | **Diff 展示** | 在 `feedback` 同步前，终端必须展示清晰的 Diff（旧文本 vs 新文本），并要求用户确认。 | 观察交互界面。 |

---

## 4. 技能取消与清理 (Remove/Cleanup)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 4.1 | **目录物理清理** | 执行 `remove [id]` 后，对应的 `.agents/skills/[id]/` 文件夹及其所有子文件必须被彻底删除。 | 检查目录是否存在。 |
| 4.2 | **状态同步移除** | 取消后，该项目的 `state.json` 记录中不再包含该技能 ID。 | 检查 `state.json` 的 `enabled_skills`。 |
| 4.3 | **安全保护** | 若标记为 OpenCode 的技能文件夹内有未同步的修改，执行 `remove` 必须触发二次确认警告。 | 修改 `SKILL.md` 后直接运行 `remove`。 |

---

## 5. 工具与脚本支持 (Tools/Scripts)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 5.1 | **脚本同步** | 仓库中 `scripts/` 目录的内容必须被递归拷贝到 OpenCode 项目的 `scripts/` 目录下。 | 检查 `.agents/skills/[id]/scripts/` 内容。 |
| 5.2 | **权限保留** | 同步后的脚本文件必须保留其原始的“可执行”权限 (`chmod +x`)，确保 Agent 可调用。 | 执行 `ls -l` 检查权限位。 |

---

## 6. 健壮性与安全 (Robustness & Safety)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 6.1 | **演习模式 (Dry Run)** | 使用 `apply --dry-run` 时，终端应输出“将要创建目录和文件”的预览，且磁盘不产生任何实际变更。 | 运行后校验目录是否存在。 |
| 6.2 | **备份一致性** | 每次 `apply` 覆盖现有 OpenCode 技能前，应自动备份旧的技能目录。 | 检查 `~/.skill-hub/backups/` 或同级目录。 |
| 6.3 | **路径冲突处理** | 若本地已有一个非 skill-hub 创建的同名文件夹，工具应报错提示，不得直接覆盖。 | 手动创建一个同名文件夹后运行 `apply`。 |

---

## 7. 综合场景验证 (Full Workflow)

**验收路径：**
1. 在项目 A 执行 `skill-hub set-target open_code`。
2. 执行 `skill-hub use git-expert` 并通过变量设置语言为 `中文`。
3. 执行 `skill-hub apply`。
4. **检查**：`.agents/skills/git-expert/SKILL.md` 是否生成且为中文。
5. 手动将该文件中的某句指令改写。
6. 执行 `skill-hub feedback git-expert`。
7. **检查**：Git 仓库文件已更新。
8. 执行 `skill-hub remove git-expert`。
9. **检查**：`.agents/skills/git-expert/` 文件夹已消失。

---

**结论：**
当以上 **22 项指标** 全部通过（Pass）时，方可确认 skill-hub 对 OpenCode 的专项支持达到发布标准。重点需关注 **AC 2.2 (元数据兼容性)** 和 **AC 5.2 (执行权限保留)**。