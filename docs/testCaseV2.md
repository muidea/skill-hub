# skill-hub e2e测试场景设计文档

## 概述

本文档定义了Skill Hub的完整e2e测试场景，基于9个核心业务场景，为后续代码开发和测试实现提供详细指导。

## 测试架构

### 测试文件结构
```
tests/e2e/
├── test_scenario1.py    # 场景1：全新技能的"本地孵化"流程
├── test_scenario2.py    # 场景2：现有技能的"状态激活与物理分发"流程
├── test_scenario3.py    # 场景3：技能的"反馈迭代"流程
├── test_scenario4.py    # 场景4：技能的"完全注销"流程
├── test_scenario5.py    # 场景5：Target优先级与默认值继承
├── test_scenario6.py    # 场景6：远程同步与多端协作（Update链路）
├── test_scenario7.py    # 场景7：Git仓库基础操作
├── test_scenario8.py    # 场景8：远程技能搜索
└── test_scenario9.py    # 场景9：本地更改推送与同步
```

### 测试依赖关系
- **本地测试**：不依赖网络，可独立运行
- **网络依赖测试**：需要远程仓库访问，可选执行

## 详细测试场景设计

### 场景1：全新技能的"本地孵化"流程（Create -> Feedback）
**测试文件**：`test_scenario1.py`
**测试目的**：验证从零开发一个技能并归档至仓库，同时自动激活状态

**测试用例设计**：
1. **test_01_environment_initialization()** - 环境初始化验证
   - 执行 `skill-hub init`
   - 验证 `~/.skill-hub` 目录结构
   - 验证默认配置
   - 执行 `skill-hub init https://github.com/example/skills-repo.git`
   - 验证远程仓库克隆
   - 执行 `skill-hub init https://github.com/example/skills-repo.git --target open_code`
   - 验证target参数设置

2. **test_02_command_dependency_check()** - 命令依赖检查验证
   - 测试未初始化时执行 `skill-hub create my-logic`
   - 验证提示需要先进行初始化
   - 验证 `skill-hub validate my-logic` 依赖检查
   - 验证 `skill-hub feedback my-logic` 依赖检查

3. **test_03_skill_creation()** - 本地技能创建验证
   - 执行 `skill-hub create my-logic`
   - 验证项目本地文件生成
   - 验证仓库无此技能
   - 验证 `state.json` 更新记录（技能标记为使用）

4. **test_04_project_workspace_check()** - 项目工作区检查验证
   - 测试不在项目目录执行命令
   - 验证 `skill-hub create my-logic` 提示新建项目工作区
   - 验证项目工作区初始化逻辑

5. **test_05_edit_and_feedback()** - 编辑与反馈验证
   - 修改项目内技能文件
   - 执行 `skill-hub validate my-logic`
   - 执行 `skill-hub feedback my-logic`
   - 验证仓库同步、索引更新、状态激活

6. **test_06_skill_listing()** - 技能列表验证
   - 执行 `skill-hub list`
   - 验证列表包含新技能
   - 验证状态显示正确
   - 执行 `skill-hub list --target open_code`
   - 验证目标环境过滤
   - 执行 `skill-hub list --verbose`
   - 验证详细信息显示

7. **test_07_full_workflow_integration()** - 完整工作流集成测试
   - 端到端测试整个创建流程
   - 验证各步骤状态一致性

8. **test_08_network_operations()** - 网络操作测试
   - 测试网络相关操作（可选）

---

### 场景2：现有技能的"状态激活与物理分发"流程（Use -> Apply）
**测试文件**：`test_scenario2.py`
**测试目的**：验证use标记状态与apply物理刷新的解耦逻辑

**测试用例设计**：
1. **test_01_command_dependency_check()** - 命令依赖检查验证
   - 测试未初始化时执行 `skill-hub set-target open_code`
   - 验证提示需要先进行初始化
   - 验证 `skill-hub use git-expert` 依赖检查
   - 验证 `skill-hub apply` 依赖检查

2. **test_02_set_project_target()** - 项目目标设置
   - 执行 `skill-hub set-target open_code`
   - 验证 `state.json` 更新
   - 验证项目工作区检查逻辑

3. **test_03_enable_skill()** - 技能启用验证
   - 执行 `skill-hub list` 发现技能
   - 执行 `skill-hub use git-expert`
   - 验证 `state.json` 状态记录（技能标记为启用）
   - 验证无物理文件生成

4. **test_04_physical_application()** - 物理文件分发
   - 执行 `skill-hub apply`
   - 验证文件从仓库复制到项目
   - 执行 `skill-hub apply --dry-run`
   - 验证演习模式功能
   - 执行 `skill-hub apply --force`
   - 验证强制应用功能

5. **test_05_command_line_target_override()** - 命令行目标覆盖
   - 测试 `skill-hub use git-expert --target cursor`
   - 验证命令行参数覆盖项目设置
   - 验证目标优先级逻辑

6. **test_06_multiple_skills_application()** - 多技能批量应用
   - 启用多个技能
   - 执行 `skill-hub apply`
   - 验证批量应用正确性

7. **test_07_target_specific_adapters()** - 目标特定适配器
   - 测试不同Target的适配器行为
   - 验证适配器正确性

8. **test_08_apply_without_enable()** - 未启用时的应用测试
   - 测试未启用技能时执行 `skill-hub apply`
   - 验证错误处理

---

### 场景3：技能的"反馈迭代"流程（Modify -> Status -> Feedback）
**测试文件**：`test_scenario3.py`
**测试目的**：验证本地修改如何通过状态检测写回仓库

**测试用例设计**：
1. **test_01_command_dependency_check()** - 命令依赖检查验证
   - 测试未初始化时执行 `skill-hub status`
   - 验证提示需要先进行初始化
   - 验证 `skill-hub feedback git-expert` 依赖检查

2. **test_02_project_modification_detection()** - 项目修改检测
   - 修改项目技能文件
   - 执行 `skill-hub status git-expert`
   - 验证Modified状态检测机制

3. **test_03_feedback_synchronization()** - 反馈同步验证
   - 执行 `skill-hub feedback git-expert`
   - 验证仓库更新，项目文件不变
   - 执行 `skill-hub feedback git-expert --dry-run`
   - 验证演习模式功能
   - 执行 `skill-hub feedback git-expert --force`
   - 验证强制更新功能

4. **test_04_status_command_options()** - status命令选项验证
   - 执行 `skill-hub status --verbose`
   - 验证详细差异信息显示
   - 执行 `skill-hub status git-expert`
   - 验证特定技能状态检查

5. **test_05_multiple_modifications()** - 多文件修改处理
   - 同时修改多个文件
   - 执行 `skill-hub feedback git-expert`
   - 验证批量反馈处理

6. **test_06_target_specific_modification_extraction()** - 目标特定修改提取
   - 测试不同Target的修改提取逻辑
   - 验证提取准确性

7. **test_07_json_escaping_handling()** - JSON转义处理
   - 测试特殊字符处理
   - 验证转义逻辑正确性

8. **test_08_partial_modifications()** - 部分修改处理
   - 测试部分文件修改场景
   - 验证选择性反馈

---

### 场景4：技能的"完全注销"流程（Remove）
**测试文件**：`test_scenario4.py`
**测试目的**：验证状态抹除与物理清理的联动

**测试用例设计**：
1. **test_01_command_dependency_check()** - 命令依赖检查验证
   - 测试未初始化时执行 `skill-hub remove git-expert`
   - 验证提示需要先进行初始化

2. **test_02_basic_skill_removal()** - 基础技能移除
   - 执行 `skill-hub remove git-expert`
   - 验证物理删除
   - 验证 `state.json` 状态移除（技能从使用列表中移除）
   - 验证仓库文件安全

3. **test_03_remove_nonexistent_skill()** - 不存在的技能移除
   - 测试移除不存在技能
   - 验证错误处理

4. **test_04_remove_multiple_skills()** - 多技能批量移除
   - 批量移除多个技能
   - 验证批量处理正确性

5. **test_05_cleanup_with_modified_files()** - 带修改文件的清理
   - 测试有未提交修改时的清理
   - 验证清理策略和安全警告

6. **test_06_cleanup_preserves_other_skills()** - 清理时保护其他技能
   - 测试选择性清理
   - 验证其他技能不受影响

7. **test_07_cleanup_with_nested_directories()** - 嵌套目录清理
   - 测试嵌套目录结构清理
   - 验证递归清理

8. **test_08_repository_safety()** - 仓库安全性验证
   - 验证仓库文件永不删除
   - 验证仓库完整性

---

### 场景5：Target优先级与默认值继承
**测试文件**：`test_scenario5.py`
**测试目的**：验证项目级设定、命令行参数与全局默认值的级联逻辑

**测试用例设计**：
1. **test_01_command_dependency_check()** - 命令依赖检查验证
   - 测试未初始化时执行 `skill-hub validate git-expert`
   - 验证提示需要先进行初始化

2. **test_02_global_default_target()** - 全局默认target验证
   - 执行 `skill-hub init`
   - 验证默认target为 `open_code`
   - 验证 `skill-hub list` 使用默认target

3. **test_03_project_target_override()** - 项目target覆盖验证
   - 执行 `skill-hub set-target cursor`
   - 验证 `state.json` 更新
   - 验证 `skill-hub list --target cursor` 过滤正确

4. **test_04_command_line_target_override()** - 命令行target覆盖验证
   - 执行 `skill-hub create my-skill --target claude`
   - 验证命令行参数覆盖项目设置
   - 验证 `skill-hub use my-skill --target claude` 优先级

5. **test_05_target_inheritance_logic()** - target继承逻辑验证
   - 测试 `skill-hub create my-skill`（无target参数）
   - 验证使用项目target
   - 测试项目无target时使用全局默认

6. **test_06_validate_command_target_handling()** - validate命令target处理
   - 执行 `skill-hub validate git-expert`
   - 验证项目工作区检查
   - 验证非法技能提示

7. **test_07_multi_level_target_priority()** - 多级target优先级验证
   - 测试：命令行target > 项目target > 全局默认
   - 验证优先级顺序正确性

8. **test_08_target_consistency_across_commands()** - 跨命令target一致性
   - 验证 `create`、`use`、`list` 等命令的target一致性
   - 测试target变更后的命令行为

---

### 场景6：远程同步与多端协作（Update链路）
**测试文件**：`test_scenario6.py`
**测试目的**：验证仓库更新后如何刷新到本地项目

**测试用例设计**：
1. **test_01_command_dependency_check()** - 命令依赖检查验证
   - 测试未初始化时执行 `skill-hub pull`
   - 验证提示需要先进行初始化

2. **test_02_pull_command_options()** - pull命令选项验证 ✅可本地
   - 执行 `skill-hub pull --check`
   - 验证检查模式功能
   - 测试 `skill-hub pull --force` 模拟

3. **test_03_detect_outdated_skills()** - 检测过时技能 ✅可本地
   - 模拟本地仓库更新
   - 执行 `skill-hub status`
   - 验证Outdated状态显示

4. **test_04_refresh_outdated_skills()** - 刷新过时技能 ✅可本地
   - 执行 `skill-hub apply`
   - 验证从更新仓库刷新到项目

5. **test_05_pull_updates_from_remote()** - 从远程拉取更新 ⚠️网络依赖
   - 执行 `skill-hub pull`
   - 验证仓库和注册表更新

6. **test_06_multi_device_collaboration_workflow()** - 多设备协作工作流 ⚠️网络依赖
   - 模拟多设备协作场景
   - 验证同步一致性

---

### 场景7：Git仓库基础操作
**测试文件**：`test_scenario7.py`
**测试目的**：验证git子命令的基本功能

**测试用例设计**：
1. **test_01_command_dependency_check()** - 命令依赖检查验证
   - 测试未初始化时执行 `skill-hub git status`
   - 验证提示需要先进行初始化

2. **test_02_git_status_command()** - Git状态命令 ✅可本地
   - 执行 `skill-hub git status`
   - 验证本地仓库状态显示

3. **test_03_git_commit_command()** - Git提交命令 ✅可本地
   - 执行 `skill-hub git commit`
   - 验证交互式提交功能

4. **test_04_git_sync_command()** - Git同步命令 ⚠️网络依赖
   - 执行 `skill-hub git sync`
   - 验证从远程拉取更改

5. **test_05_git_clone_command()** - Git克隆命令 ⚠️网络依赖
   - 执行 `skill-hub git clone <repo-url>`
   - 验证远程仓库克隆

6. **test_06_git_remote_command()** - Git远程命令 ⚠️网络依赖
   - 执行 `skill-hub git remote <repo-url>`
   - 验证远程仓库设置

7. **test_07_git_push_command()** - Git推送命令 ⚠️网络依赖
   - 执行 `skill-hub git push`
   - 验证推送功能

8. **test_08_git_pull_command()** - Git拉取命令 ⚠️网络依赖
   - 执行 `skill-hub git pull`
   - 验证拉取功能

9. **test_09_git_operations_integration()** - Git操作集成测试
   - 测试Git操作集成
   - 验证操作一致性

---

### 场景8：远程技能搜索
**测试文件**：`test_scenario8.py`
**测试目的**：验证search命令的功能

**测试用例设计**：
1. **test_01_command_dependency_check()** - 命令依赖检查验证
   - 测试未初始化时执行 `skill-hub search git`
   - 验证提示需要先进行初始化

2. **test_02_basic_search_functionality()** - 基础搜索功能 ⚠️网络依赖
   - 执行 `skill-hub search git`
   - 验证关键词搜索

3. **test_03_target_filtered_search()** - 目标过滤搜索 ⚠️网络依赖
   - 执行 `skill-hub search database --target open_code`
   - 验证目标过滤

4. **test_04_search_result_limit()** - 搜索结果限制 ⚠️网络依赖
   - 执行 `skill-hub search python --limit 10`
   - 验证结果数量限制

5. **test_05_empty_search_results()** - 空搜索结果处理 ⚠️网络依赖
   - 测试无结果搜索
   - 验证空结果处理

6. **test_06_search_with_special_characters()** - 特殊字符搜索 ⚠️网络依赖
   - 测试特殊字符搜索
   - 验证字符处理

7. **test_07_search_integration_with_use_command()** - 搜索与use命令集成 ⚠️网络依赖
   - 测试搜索后直接启用
   - 验证集成流程

---

### 场景9：本地更改推送与同步
**测试文件**：`test_scenario9.py`
**测试目的**：验证push命令的功能

**测试用例设计**：
1. **test_01_command_dependency_check()** - 命令依赖检查验证
   - 测试未初始化时执行 `skill-hub push`
   - 验证提示需要先进行初始化

2. **test_02_local_change_detection()** - 本地更改检测 ✅可本地
   - 执行 `skill-hub git status`
   - 验证本地更改检测

3. **test_03_auto_commit_and_push()** - 自动提交推送 ⚠️网络依赖
   - 执行 `skill-hub push --message "更新描述"`
   - 验证自动提交推送

4. **test_04_dry_run_mode()** - 演习模式测试 ✅可本地
   - 执行 `skill-hub push --dry-run`
   - 验证预览输出，无实际推送

5. **test_05_force_push_operation()** - 强制推送测试 ⚠️网络依赖
   - 执行 `skill-hub push --force`
   - 验证强制推送逻辑

6. **test_06_push_conflict_resolution()** - 推送冲突处理 ⚠️网络依赖
   - 测试推送冲突场景
   - 验证冲突处理

## 测试执行策略

### 阶段1：核心本地功能测试（高优先级）
**目标**：不依赖网络，覆盖核心功能
**覆盖场景**：
- 场景1：test_01-test_07（全部本地测试）
- 场景2：test_01-test_08（全部本地测试）
- 场景3：test_01-test_08（全部本地测试）
- 场景4：test_01-test_08（全部本地测试）
- 场景5：test_01-test_08（全部本地测试）
- 场景6：test_01-test_04（本地模拟测试）
- 场景7：test_01-test_03（本地Git操作测试）
- 场景9：test_01、test_02、test_04（本地测试）

### 阶段2：网络依赖功能测试（中优先级，可选）
**目标**：需要远程仓库访问
**覆盖场景**：
- 场景6：test_05、test_06（网络依赖测试）
- 场景7：test_04-test_08（网络依赖Git操作）
- 场景8：test_02-test_07（全部网络依赖）
- 场景9：test_03、test_05、test_06（网络依赖测试）

### 阶段3：集成与边界测试（低优先级）
**目标**：完善测试覆盖
**测试内容**：
- 跨场景工作流集成
- 错误处理和边界条件
- 性能和大数据量测试
- 并发操作测试

## 测试数据需求

### 本地测试数据
1. **测试技能模板**：`my-logic-skill`（位于`tests/e2e/data/test_skills/`）
2. **预期输出文件**：`expected_output/`目录
3. **本地Git仓库配置**：用于模拟本地仓库操作
4. **状态文件模板**：`state.json`、`registry.json`模板

### 远程测试数据（网络依赖）
1. **远程技能仓库URL**：用于pull、clone、search测试
2. **测试技能数据**：包含各种技能的远程仓库
3. **认证配置**：Git操作所需的认证信息

## 测试环境要求

### 基础环境
- Python 3.8+
- `skill-hub`命令在PATH中
- 临时目录读写权限
- Git命令行工具

### 网络环境（可选）
- 互联网连接（用于远程仓库测试）
- Git远程仓库访问权限
- 稳定的网络环境

## 测试覆盖率目标

### 命令覆盖率
| 命令 | 测试状态 | 测试场景 | 依赖检查 | state.json更新 |
|------|----------|----------|----------|----------------|
| init | ✅ 100% | 场景1 | - | ✓ |
| set-target | ✅ 100% | 场景2、5 | ✓ | ✓ |
| list | ✅ 100% | 场景1、2 | ✓ | - |
| search | ⚠️ 部分 | 场景8（网络依赖） | ✓ | - |
| create | ✅ 100% | 场景1 | ✓ | ✓ |
| remove | ✅ 100% | 场景4 | ✓ | ✓ |
| validate | ✅ 100% | 场景1、5 | ✓ | - |
| use | ✅ 100% | 场景2 | ✓ | ✓ |
| status | ✅ 100% | 场景3、6 | ✓ | - |
| apply | ✅ 100% | 场景2、6 | ✓ | - |
| feedback | ✅ 100% | 场景1、3 | ✓ | - |
| pull | ⚠️ 部分 | 场景6（本地+网络） | ✓ | - |
| push | ⚠️ 部分 | 场景9（本地+网络） | ✓ | - |
| git | ✅ 100% | 场景7、9 | ✓ | - |

### 状态文件验证矩阵
| 命令 | 更新state.json | 检查项目工作区 | 依赖init检查 | 测试验证点 |
|------|---------------|---------------|-------------|------------|
| init | ✓ | - | - | 目录结构、默认配置 |
| set-target | ✓ | ✓ | ✓ | state.json更新、项目初始化 |
| list | - | - | ✓ | 过滤显示、verbose选项 |
| search | - | - | ✓ | 关键词搜索、目标过滤 |
| create | ✓ | ✓ | ✓ | 本地文件生成、state记录 |
| remove | ✓ | ✓ | ✓ | 物理删除、状态移除、仓库安全 |
| validate | - | ✓ | ✓ | 合规性检查、非法技能提示 |
| use | ✓ | ✓ | ✓ | 状态记录、无物理文件生成 |
| status | - | ✓ | ✓ | 状态检测、详细差异显示 |
| apply | - | ✓ | ✓ | 物理分发、dry-run选项 |
| feedback | - | ✓ | ✓ | 仓库同步、dry-run选项 |
| pull | - | - | ✓ | 仓库更新、注册表刷新 |
| push | - | - | ✓ | 自动提交、dry-run选项 |
| git | - | - | ✓ | 子命令功能、集成测试 |

### 业务场景覆盖率
- ✅ 场景1-5：100%覆盖（包含依赖检查和state.json测试）
- ✅ 场景6-9：本地功能100%覆盖，远程功能需要网络
- ✅ 所有命令：100%依赖检查测试覆盖
- ✅ 关键命令：100% state.json更新逻辑测试覆盖

## 实施建议

### 开发顺序
1. **验证核心场景测试**：确保场景1-5的测试正常运行，包含依赖检查
2. **实现命令依赖检查**：为所有命令添加init依赖检查测试
3. **完善state.json逻辑测试**：为相关命令添加state.json更新验证
4. **实现本地化测试**：完成场景6、7、9的本地测试部分
5. **配置测试环境**：设置测试数据和环境
6. **实现网络测试**：（可选）配置远程仓库，实现网络依赖测试
7. **集成测试**：进行跨场景集成测试

### 质量保证
- 每个测试用例应有明确的验证点
- 测试应具备良好的错误处理和日志
- 重要功能应有边界条件测试
- 测试数据应易于维护和更新

---

*本文档最后更新：2026-02-11*
*版本：v3.0*