# skill-hub 验收准则补遗：Claude 专项模块

## 模块 9：Claude 环境与路径识别 (Environment)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 9.1 | **配置自动定位** | 工具必须能自动识别并定位 `~/.claude/config.json` (Code) 和 Claude Desktop 的配置文件路径。 | 在不同 OS (Mac/Win) 下运行 `check claude`，观察输出路径是否正确。 |
| 9.2 | **备份机制** | 在修改 Claude 的 JSON 配置文件前，必须在同目录下生成 `.bak` 备份文件。 | 执行 `skill-hub set-target claude`，然后执行 `skill-hub apply`，检查是否存在 `config.json.bak`。 |
| 9.3 | **权限预检** | 若配置文件由于系统权限无法写入，工具应返回 `ERR_FS_PERMISSION` 而非静默失败。 | 将 `config.json` 设为只读，执行 `apply` 观察报错。 |

## 模块 10：指令型技能注入 (Instructional Apply)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 10.1 | **JSON 转义安全** | 注入 `SKILL.md` 内容时，必须处理引号、换行符的转义，确保注入后 JSON 依然合法。 | 在 `SKILL.md` 中写入双引号和多行文字，`apply` 后使用 `jq` 或在线校验器检查 JSON。 |
| 10.2 | **标记位精确性** | 注入的指令必须被包裹在 `/* SKILL-HUB BEGIN: id */` 和 `/* SKILL-HUB END: id */` 之间。 | 打开 `config.json` 搜索该标记位字符串。 |
| 10.3 | **全局与项目隔离** | 使用 `--global` 时修改系统配置，不带参数时应优先寻找并修改当前目录的 `.clauderc`。 | 分别在全局和项目目录下执行 `apply`，检查对应文件。 |

## 模块 11：工具型技能/MCP 注入 (Tool/MCP Apply)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 11.1 | **绝对路径转换** | 注入 `mcpServers` 的 `args` 时，必须将相对路径（如 `./main.py`）转换为 Git 仓库的**绝对路径**。 | 执行 `apply`，检查 `mcp_config.json` 中的路径是否为全路径。 |
| 11.2 | **Tool Schema 合法性** | `SKILL.md` 中的 `tool_spec` 必须被原样且合法地转换为 Claude 识别的 JSON 节点。 | 检查配置文件中 `tools` 数组下的 `input_schema` 是否完整。 |
| 11.3 | **运行环境透传** | 必须根据 `SKILL.md` 中的 `runtime` 自动前缀化命令（如 `python3 -u`），确保脚本可执行。 | 检查 `mcpServers` 下的 `command` 字段是否匹配预期。 |
| 11.4 | **依赖自动安装** | (可选) 若配置了自动同步，检测到 `requirements.txt` 时应提示或自动运行安装。 | 准备一个带依赖的 Skill，执行 `apply` 观察控制台输出。 |

## 模块 12：Claude 反馈闭环 (Feedback Loop)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 12.1 | **JSON 片段提取** | `feedback` 必须能从 JSON 字符串中精准剥离出标记块内的文本，且去除转义字符恢复为 MD 格式。 | 手动在 `config.json` 的指令中改一个字，运行 `feedback` 观察 Diff。 |
| 12.2 | **无损同步** | 如果只反馈了“指令”，同一 Skill 下的“工具定义 (Tool Spec)”不应被意外修改或覆盖。 | 执行指令反馈，检查 `SKILL.md` 中的 `tool_spec` 是否保持原样。 |
| 12.3 | **配置有效性维护** | 反馈回传并重新 `apply` 后，Claude 配置文件不应产生冗余的转义符或导致格式错乱。 | 多次循环执行 `feedback` 和 `apply`，检查 JSON 文件的整洁度。 |

---

### 综合验收场景 (Claude 特色场景)

**场景：部署并调试一个 Claude 数据分析工具**

1.  **部署验证**：
    *   执行 `skill-hub set-target claude`。
    *   执行 `skill-hub apply`。
    *   **预期**：`config.json` 被更新，包含了技能指令，并在 `mcpServers` 中注册了该工具的 Python 脚本绝对路径。
2.  **可用性验证**：
    *   打开 Claude Code，输入 `tools`。
    *   **预期**：列表中出现 `SKILL.md` 中定义的 `tool_name`。
3.  **闭环验证**：
    *   在对话中发现指令微调更好，于是手动修改 `~/.claude/config.json` 里的指令部分。
    *   执行 `skill-hub feedback`。
    *   **预期**：skill-hub 弹出 Diff 窗口，展示刚才在 JSON 里的修改，确认后 Git 仓库中的 `SKILL.md` 被更新。

---

### 验收结论要求

所有 Claude 专项 AC 必须达到 **100% 通过**，方可认为 skill-hub 具备了完善的 Claude 生态支持能力。特别是 **AC 10.1 (JSON 转义安全)** 和 **AC 11.1 (绝对路径转换)**，是保证工具在用户生产环境下不“写坏”配置文件的底线。