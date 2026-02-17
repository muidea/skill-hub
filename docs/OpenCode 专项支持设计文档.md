# skill-hub - OpenCode 专项支持设计文档 (v1.0)

## 1. 设计遵循规范 (Compliance)
本设计完全遵循 skill-hub 核心架构要求：
*   **Git 驱动**：OpenCode 的技能元数据与指令必须由 Git 仓库统一管理。
*   **目录同步模型**：不同于 Cursor/Claude 的“文件内注入”，OpenCode 适配器采用“**结构化目录同步**”逻辑。
*   **状态绑定**：项目与 OpenCode 的绑定关系持久化于 `state.json`。
*   **闭环反馈**：支持从 OpenCode 的 `instructions.md` 提取修改并回写至 Git 仓库。

---

## 2. OpenCode 技能模型分析
根据 skill-hub 的设计决策，我们使用统一的 `.agents/skills/` 目录结构，而不是 OpenCode 官方的 `.skills/` 目录。结构如下：
*   **位置**：
    *   项目级：`[ProjectRoot]/.agents/skills/[skill-id]/`
    *   全局级：`~/.config/opencode/skills/[skill-id]/`（适配器内部使用）
*   **核心文件**：
    *   `SKILL.md`：包含技能元数据和指令内容（YAML frontmatter + Markdown）。

---

## 3. 数据规约与映射 (Data Mapping)

skill-hub 将通过 `internal/adapter/opencode` 模块执行以下映射转换：

| skill-hub 原始文件 | 映射至 OpenCode 文件 | 处理逻辑 |
| :--- | :--- | :--- |
| `SKILL.md` | `SKILL.md` | 保持相同格式，但验证技能名称符合 OpenCode 规范 |
| `scripts/` | `scripts/` | 递归拷贝脚本文件并确保执行权限 (chmod +x) |

---

## 4. 适配器逻辑设计 (OpenCode Adapter)

### 4.1 分发逻辑 (Apply)
1.  **路径解析**：识别当前项目的 `ProjectRoot`。
2.  **目录创建**：在 `[ProjectRoot]/.agents/skills/[id]/` 创建目标文件夹。
3.  **内容生成**：
    *   验证技能名称符合 OpenCode 命名规范。
    *   写入 `SKILL.md` 文件。
4.  **工具同步**：若 `SKILL.md` 中定义了工具，将脚本同步至 `scripts/` 目录。

### 4.2 反馈逻辑 (Feedback Loop)
1.  **Status 检查**：直接对比 `.agents/skills/[id]/SKILL.md` 与 Git 仓库内原始 `SKILL.md` 的 MD5 值。
2.  **提取回传**：若有差异，读取 `SKILL.md` 全文，展示 Diff，确认后回写至 Git 仓库。

### 4.3 清理逻辑 (Remove)
1.  **状态更新**：从 `state.json` 移除该技能 ID。
2.  **物理删除**：直接递归删除整个 `.agents/skills/[id]/` 文件夹。

---

## 5. 核心代码结构 (Go Layout)

```text
internal/
├── adapter/
│   ├── opencode/
│   │   ├── adapter.go          # 实现统一接口 Apply/Remove/Feedback
│   │   ├── manifest.go         # 处理 YAML 转换逻辑
│   │   └── sync.go             # 处理文件夹同步与权限管理
```

---

## 6. 指令集行为 (CLI Commands)

| 指令 | 针对 OpenCode 的行为 |
| :--- | :--- |
| `skill-hub set-target open_code` | 将当前项目绑定为 OpenCode 环境。 |
| `skill-hub apply [--dry-run] [--force]` | 自动在项目根目录生成或更新 `.agents/skills/` 文件夹。 |
| `skill-hub remove <id>` | 删除对应的技能文件夹并更新状态。 |
| `skill-hub feedback <id> [--dry-run] [--force]` | 从 `.agents/skills/[id]/SKILL.md` 同步改动。 |

---

## 7. 验收准则 (Acceptance Criteria)

### 7.1 部署验收
*   **AC 1.1**: 执行 `apply` 后，`.agents/skills/[id]/` 目录必须包含 `SKILL.md`。
*   **AC 1.2**: 生成的 `SKILL.md` 必须包含有效的 YAML frontmatter 和技能内容。
*   **AC 1.3**: 所有拷贝到 `scripts/` 目录下的脚本必须保留其原始执行权限。

### 7.2 反馈验收
*   **AC 2.1**: 手动修改 `SKILL.md` 后，`skill-hub status` 必须能识别出 `Modified` 状态。
*   **AC 2.2**: `feedback` 操作必须能准确回传内容，且不丢失任何换行或特殊字符。

### 7.3 清理验收
*   **AC 3.1**: 执行 `remove` 后，对应的 `.agents/skills/[id]/` 文件夹必须被彻底删除，且不影响 `.agents/skills/` 目录下的其他技能。
*   **AC 3.2**: 若项目中所有技能均被移除，`.agents/skills/` 文件夹应根据用户首选项决定是否保留（默认清理空目录）。

---

## 8. 开发处理清单 (Backlog)
*   [ ] **State 枚举更新**：在 `TargetType` 中增加 `open_code` 常量。
*   [ ] **适配器开发**：编写 `internal/adapter/opencode` 模块。
*   [ ] **特征探测**：在自动识别逻辑中增加对项目根目录 `.agents/skills/` 文件夹的探测，作为辅助识别手段。
*   [ ] **变量渲染适配**：确保 OpenCode 目标下的指令渲染符合其 Markdown 渲染标准。

---
