# skill-hub - Claude 专项支持设计文档

## 1. 设计遵循规范 (Compliance)
本设计必须且完全遵循 skill-hub 核心架构要求：
*   **Git 驱动**：所有 Claude 指令和工具定义必须版本化存储于 Git。
*   **Go 工程化**：使用 `internal/adapter/claude` 模块实现，复用核心 `engine`。
*   **标记块标准**：必须使用标记对（Markers）识别注入内容，确保非侵入性。
*   **闭环反馈**：支持从 Claude 配置文件回写改动至 Git。

---

## 2. Claude 技能模型 (Skill Model)

在 skill-hub 规范下，Claude 技能分为两类：

### 2.1 指令型技能 (Instructional Skill)
*   **目标**：注入 Claude 的系统指令（System Prompts）。
*   **适配路径**：
    *   `~/.claude/config.json` (Global)
    *   `[Project]/.clauderc` (Project)
*   **注入方式**：纯文本字符串注入。

### 2.2 工具型技能 (Tool/MCP Skill)
*   **目标**：将本地脚本（Python/Node/Go）暴露给 Claude 调用。
*   **适配路径**：`~/Library/Application Support/Claude/claude_desktop_config.json` (MCP 节点)。
*   **注入方式**：JSON 结构化定义注入。

---

## 3. 核心数据规范 (Data Schema)

遵循 skill-hub 的 `.agents/skills/[id]/` 目录结构，对 `SKILL.md` 进行 Claude 专项增强。

### 3.1 增强型 SKILL.md
```yaml
id: "claude-log-analyzer"
name: "日志分析工具"
version: "1.0.0"
compatibility: Designed for Claude Code (or similar AI coding assistants)

# Claude 专属扩展块 (遵循 skill-hub 规范)
claude:
  # 模式: instruction | tool
  mode: "tool"
  
  # 工具执行配置 (仅 tool 模式)
  runtime: "python3"
  entrypoint: "./analyze.py"
  
  # 工具定义 (符合 MCP/JSON Schema 规范)
  tool_spec:
    name: "analyze_logs"
    description: "分析项目日志文件并识别潜在错误"
    input_schema:
      type: "object"
      properties:
        log_path: { type: "string", description: "日志路径" }
      required: ["log_path"]

variables:
  - name: "LOG_LEVEL"
    default: "DEBUG"
```

---

## 4. 核心组件设计 (Engine & Adapter)

### 4.1 Claude JSON 适配引擎
由于 Claude 配置多为 JSON，`internal/adapter/claude.go` 需处理以下逻辑：
*   **字符串转义安全**：在将 `SKILL.md` 内容注入 JSON 字符串字段时，必须进行合法的 JSON 转义处理。
*   **标记块嵌入**：在 JSON 字符串内使用注释风格的标记：
    ```json
    "customInstructions": "/* SKILL-HUB BEGIN: id */ ...指令内容... /* SKILL-HUB END: id */"
    ```
*   **路径动态化**：在注入 `mcpServers` 时，将相对路径 `./analyze.py` 自动转换为 Git 仓库的**绝对路径**。

### 4.2 环境同步 (Dependency Sync)
针对工具型技能，`apply` 指令需额外执行：
*   自动检测并在技能目录下运行 `pip install` 或 `npm install`（如果配置允许）。
*   确保 `entrypoint` 脚本具有执行权限 (`chmod +x`)。

---

## 5. 闭环反馈流程 (The Feedback Loop)

遵循 skill-hub 的“发现-差异-同步”流程：

1.  **检测 (Status)**：
    *   解析 `config.json`，利用正则表达式提取 `/* SKILL-HUB BEGIN: id */` 块内的内容。
    *   将提取的内容与 Git 仓库中渲染后的 `SKILL.md` 进行对比。
2.  **提取 (Extract)**：
    *   如果是工具定义的 `tool_spec` 发生变化（例如用户手动在 Claude 配置文件改了参数描述），Adapter 需将 JSON 片段逆向映射回 YAML 结构。
3.  **同步 (Sync)**：
    *   调用 `skill-hub feedback` 指令，展示 Diff，经确认后覆盖仓库文件。

---

## 6. 项目目录布局 (Go Layout Integration)

```text
internal/
├── adapter/
│   ├── claude/
│   │   ├── adapter.go          # 实现统一的 Adapter 接口
│   │   ├── instruction_task.go # 处理系统指令注入
│   │   └── mcp_task.go         # 处理 MCP/Tool 定义注入
│   └── cursor/
├── engine/
│   ├── json_handler.go         # 专门负责带标记位的 JSON 读写
│   └── template_engine.go      # 复用已有的变量渲染引擎
```

---

## 7. 验收准则 (Acceptance Criteria)

### 7.1 指令注入验收
*   执行 `skill-hub apply` 后，`~/.claude/config.json` 的 `customInstructions` 字段必须包含正确转义的内容。
*   Claude Code 运行过程中能正确识别并执行注入的指令。

### 7.2 工具/MCP 验收
*   注入后，Claude Desktop 配置文件中出现对应的 `mcpServers` 条目。
*   路径必须为绝对路径，且 `runtime`（如 python3）能被系统正确调用。

### 7.3 反馈闭环验收
*   手动修改 `config.json` 中的指令，执行 `skill-hub feedback` 必须能精准回传至对应的 `SKILL.md`。
*   JSON 格式在多次 `apply/feedback` 循环后，必须保持语法正确，无非法字符。

---

## 8. 总结
本专项设计在不破坏 **skill-hub** 统一规范的前提下，通过增强 `SKILL.md` 定义和开发 `JSON 专用适配器`，实现了对 Claude 指令和工具的高效管理。这使得开发者可以通过一套 Git 流程，同时管理 Cursor 的规则和 Claude 的 MCP 工具链。