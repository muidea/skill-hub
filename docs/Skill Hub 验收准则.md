# skill-hub 验收准则 (Detailed Acceptance Criteria)

> 文档状态说明：
> 本文档已按当前产品定位做最小必要同步。对于运行模型、命令分层和服务边界，以 [README.md](../README.md)、[DEVELOPMENT.md](../DEVELOPMENT.md) 和 [Skill-Hub命令规范.md](./Skill-Hub命令规范.md) 为准。

## 模块 1：环境初始化与基础架构 (Initialization)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 1.1 | **工作区创建** | 首次运行 `init` 后，用户家目录下必须生成 `.skill-hub` 文件夹，权限为 755。 | 执行 `skill-hub init [git_url] [--target <value>]`，检查 `~/.skill-hub`。 |
| 1.2 | **Git 仓库同步** | `~/.skill-hub/repositories/<repo-name>/` 目录下应包含远程仓库的所有文件，且 `.git` 目录完整。 | 执行 `ls -a ~/.skill-hub/repositories/main` 查看内容。 |
| 1.3 | **配置文件格式** | `config.yaml` 必须包含 `multi_repo` 配置节来管理多个仓库。 | 查看 `config.yaml` 内容，确认为合法的 YAML。 |
| 1.4 | **二进制自包含** | 工具编译后为单二进制文件，不依赖系统安装 Python、Git 或其他运行库。 | 在纯净系统环境下直接运行 `skill-hub` 二进制。 |

## 模块 2：Skill 数据合规性 (Data Compliance)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 2.1 | **目录结构校验** | 每个 Skill 必须包含 `SKILL.md` 文件，否则 `list` 指令应报错或跳过并提示。 | 删除某个 Skill 的 `SKILL.md`，运行 `skill-hub list [--target <value>] [--verbose]`。 |
| 2.2 | **YAML 解析** | 能够正确读取 `SKILL.md` 里的 `compatibility` 字符串，过滤不支持当前工具的技能。 | 在 `SKILL.md` 中设置不支持 `cursor`，检查 `apply` 是否跳过。 |
| 2.3 | **模板变量支持** | `SKILL.md` 中的 `{{.VAR}}` 在应用时能被正确替换为具体值。 | 编写带变量的 Skill，执行 `use --var K=V` 后查看导出结果。 |

## 模块 3：分发与部署逻辑 (Distribution & Apply)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 3.1 | **标记块注入** | 在 `.cursorrules` 中准确生成 `BEGIN` 和 `END` 标记，且标记后紧跟 Skill ID。 | 执行 `apply`，检查 `.cursorrules` 文件内容。 |
| 3.2 | **增量更新** | 再次执行 `apply` 时，仅更新标记块内内容，块外手写内容（如文件首部注释）不得丢失。 | 在标记块外手动添加一行文字，再次 `apply` 确认该行存在。 |
| 3.3 | **多技能合并** | 一个项目启用多个 Skill 时，文件内应有序排列多个标记块，且块之间有空行分隔。 | `use` 两个不同技能，执行 `apply` 检查目标文件。 |
| 3.4 | **原子写入保障** | 在写入过程中人为中断进程，原始配置文件应保持修改前状态或自动恢复备份。 | 写入大文件时尝试强杀进程，检查 `.bak` 文件和原文件。 |

## 模块 4：状态追踪 (State Persistence)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 4.1 | **项目绑定记录** | `state.json` 必须实时记录当前项目路径与启用的 Skill 列表、变量值，以及 skill 的来源仓库信息。 | 运行 `skill-hub use <id> [--target <value>]` 后，查看 `~/.skill-hub/state.json`。 |
| 4.2 | **路径一致性** | 工具需处理软链接路径、相对路径转绝对路径，确保在不同目录下运行状态识别一致。 | 在项目子目录下运行 `skill-hub status [id] [--verbose]`，应能识别根目录状态。 |

## 模块 5：闭环反馈功能 (Feedback Loop) —— **重中之重**

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 5.1 | **内容差异检测** | 手动修改标记块内的一个字符，`status` 命令必须输出 `Modified` 状态。 | 修改内容后运行 `skill-hub status [id] [--verbose]`。 |
| 5.2 | **正则提取精度** | 无论标记块位于文件头部、中部或尾部，`feedback` 都能精准提取块内文本，不带多余空行。 | 在文件中间修改 Skill 内容，运行 `feedback` 并查看仓库文件。 |
| 5.3 | **交互式确认** | `feedback` 必须展示归档预览，由用户输入 `y/n` 确认后才可归档到默认仓库。 | 运行 `feedback`，观察是否出现确认与预览信息。 |
| 5.4 | **反向变量处理** | (进阶) 反馈回仓库的内容应尽量保持通用性，或提示用户手动移除项目特定变量值。 | 检查 `feedback` 后的 `SKILL.md` 内容。 |

## 6. 远端发现与搜集 (Discovery)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 6.1 | **关键词检索** | `search` 指令在本地 `serve` 可用时应通过服务承接远端搜索交互；服务不可用时也应能回退到本地执行，并返回与关键词相关的候选技能。 | 分别在服务可用与不可用场景下运行 `skill-hub search <keyword> [--target <value>] [--limit <number>]`。 |
| 6.2 | **多仓库同步入口** | `repo sync` 应负责同步指定仓库或所有启用仓库；`pull` 不承担全多仓库同步职责。 | 运行 `skill-hub repo sync` 与 `skill-hub pull`，检查两者输出和影响范围。 |

## 7. 工程鲁棒性与性能 (Engineering)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 7.1 | **跨平台路径** | Windows 下路径使用 `\`，macOS/Linux 下使用 `/`，工具需自动适配，不产生乱码。 | 在 Windows 和 Mac 上交叉测试。 |
| 7.2 | **错误提示友好度** | 遇到无权限写入、Git 冲突、YAML 语法错误时，必须输出易懂的中文/英文错误信息。 | 故意制造 YAML 语法错误，观察报错。 |
| 7.3 | **Dry Run 演习** | `--dry-run` 模式下，控制台输出所有变更预览，文件哈希值及修改时间戳完全不变。 | 运行 `apply --dry-run` 或 `push --dry-run`，用 `md5sum` 校验原文件。 |

## 8. 合规性与发布 (License & Release)

| 编号 | 验收项 | 验收指标 (Expectation) | 验证操作 (How to Verify) |
| :--- | :--- | :--- | :--- |
| 8.1 | **许可证清单** | 二进制分发包中包含 `THIRD-PARTY-NOTICES`，列出所有 Apache 2.0/MIT 依赖。 | 检查发布包中的法律声明文件。 |
| 8.2 | **版本一致性** | `skill-hub --version` 输出的版本号必须与 Git Tag 及编译时的内部变量一致。 | 运行版本指令查看输出。 |

---

### 验收评估结论 (Summary Table)

| 核心维度 | 状态 (Pass/Fail) | 备注 |
| :--- | :--- | :--- |
| **基础架构** | [ ] | |
| **部署引擎** | [ ] | |
| **反馈闭环** | [ ] | |
| **数据合规** | [ ] | |

**建议：** 在开发第一个迭代版本（MVP）时，优先验收 **3.1, 4.1, 5.1, 5.3** 这四项，它们构成了 skill-hub 的核心闭环价值。
