import os
import pytest
import tempfile
import shutil
from pathlib import Path
from typing import Dict, Any, List, Optional

@pytest.fixture
def test_skill_template():
    """
    返回测试技能模板
    
    这个fixture提供标准的测试技能模板内容
    """
    return """---
name: my-logic-skill
description: 逻辑测试技能，用于端到端测试 - 完全匹配验证
compatibility: Designed for Cursor, Claude Code, and OpenCode
metadata:
  version: 1.0.0
  author: E2E Test Team
  tags: [test, e2e, validation]
  created: 2026-02-08
variables:
  - name: LANGUAGE
    default: zh-CN
    description: 输出语言配置
  - name: STYLE
    default: concise
    description: 输出风格设置
  - name: DEBUG
    default: "false"
    description: 调试模式开关
---
# 逻辑测试技能 - 端到端测试专用

这是一个专门用于端到端测试的技能，所有内容都需要完全匹配验证。

## 核心功能
1. **变量替换测试**: 语言 = {{.LANGUAGE}}
2. **风格配置测试**: 风格 = {{.STYLE}}
3. **调试模式测试**: 调试 = {{.DEBUG}}

## 验证要求
- 文件内容必须完全匹配
- 变量必须正确替换
- 结构必须完整无缺

## 测试数据
- 创建时间: 2026-02-08
- 测试ID: e2e-test-001
- 验证级别: strict

注意：此文件用于完全匹配验证测试，任何修改都会导致测试失败。
"""

@pytest.fixture
def simple_skill_template():
    """
    返回简单技能模板
    
    这个fixture提供简化的测试技能模板
    """
    return """---
name: simple-test-skill
description: 简单测试技能
compatibility: all
metadata:
  version: 1.0.0
  author: Test User
variables:
  - name: NAME
    default: World
    description: 名称
---
# 简单测试技能

Hello {{.NAME}}!

这是一个简单的测试技能。
"""

@pytest.fixture
def invalid_skill_template():
    """
    返回无效技能模板
    
    这个fixture提供格式错误的技能模板，用于测试错误处理
    """
    return """---
name: invalid-skill
# 缺少必需的description字段
compatibility: Invalid Tool  # 无效的兼容性声明
metadata:
  version: not-a-version  # 无效的版本格式
variables:
  - name: VAR1
    # 缺少default和description
---
# 无效技能

这个技能模板包含多个格式错误。
"""

@pytest.fixture
def skill_with_variables():
    """
    返回带变量的技能模板
    
    这个fixture提供包含多个变量的技能模板
    """
    return """---
name: variable-test-skill
description: 变量测试技能
compatibility: all
metadata:
  version: 1.0.0
variables:
  - name: PROJECT_NAME
    default: MyProject
    description: 项目名称
  - name: LANGUAGE
    default: Go
    description: 编程语言
  - name: FRAMEWORK
    default: Gin
    description: Web框架
  - name: DATABASE
    default: PostgreSQL
    description: 数据库
---
# {{.PROJECT_NAME}} 项目配置

## 技术栈
- 语言: {{.LANGUAGE}}
- 框架: {{.FRAMEWORK}}
- 数据库: {{.DATABASE}}

## 使用说明
这是一个使用 {{.LANGUAGE}} 和 {{.FRAMEWORK}} 框架的项目模板。
"""

@pytest.fixture
def opencode_skill_template():
    """
    返回OpenCode专用技能模板
    
    这个fixture提供符合OpenCode格式的技能模板
    """
    return """---
name: opencode-test-skill
description: OpenCode测试技能
compatibility: open_code
metadata:
  version: 1.0.0
  author: OpenCode Test
  tags: [opencode, test]
variables:
  - name: TASK
    default: code review
    description: 任务类型
---
# OpenCode测试技能

## 功能说明
这是一个专门为OpenCode适配器设计的测试技能。

## 任务: {{.TASK}}

请根据以下要求执行{{.TASK}}:
1. 检查代码规范
2. 识别潜在问题
3. 提供改进建议

## OpenCode特定要求
- 必须生成manifest.yaml
- 必须生成instructions.md
- 必须符合OpenCode目录结构
"""

@pytest.fixture
def cursor_skill_template():
    """
    返回Cursor专用技能模板
    
    这个fixture提供符合Cursor格式的技能模板
    """
    return """---
name: cursor-test-skill
description: Cursor测试技能
compatibility: cursor
metadata:
  version: 1.0.0
  author: Cursor Test
  tags: [cursor, test]
variables:
  - name: CONTEXT
    default: development
    description: 上下文环境
---
# Cursor测试技能

## 使用场景
这是一个专门为Cursor适配器设计的测试技能。

## 上下文: {{.CONTEXT}}

在{{.CONTEXT}}环境中，请遵循以下规则:
1. 使用一致的代码风格
2. 添加必要的注释
3. 遵循最佳实践

## Cursor特定要求
- 必须注入到.cursorrules文件
- 使用正确的标记块格式
- 支持变量替换
"""

@pytest.fixture
def temporary_skill_dir(test_skill_template):
    """
    返回临时技能目录
    
    这个fixture创建包含测试技能的临时目录
    """
    # 创建临时目录
    temp_dir = tempfile.mkdtemp(prefix="skill_dir_")
    skill_dir = os.path.join(temp_dir, "my-logic-skill")
    os.makedirs(skill_dir, exist_ok=True)
    
    # 创建SKILL.md文件
    skill_file = os.path.join(skill_dir, "SKILL.md")
    with open(skill_file, 'w', encoding='utf-8') as f:
        f.write(test_skill_template)
    
    # 创建skill.yaml文件（如果需要）
    yaml_file = os.path.join(skill_dir, "skill.yaml")
    yaml_content = """name: my-logic-skill
description: 逻辑测试技能
version: 1.0.0
compatibility: all
metadata:
  author: Test User
  tags: [test]
variables:
  - name: LANGUAGE
    default: zh-CN
    description: 语言
"""
    with open(yaml_file, 'w', encoding='utf-8') as f:
        f.write(yaml_content)
    
    yield skill_dir
    
    # 清理
    shutil.rmtree(temp_dir, ignore_errors=True)

@pytest.fixture
def skill_data_directory():
    """
    返回技能数据目录路径
    
    这个fixture提供测试数据目录的路径
    """
    return Path(__file__).parent.parent / "data" / "test_skills"

@pytest.fixture
def expected_output_directory():
    """
    返回期望输出目录路径
    
    这个fixture提供期望输出文件的目录路径
    """
    data_dir = Path(__file__).parent.parent / "data" / "test_skills" / "my-logic-skill"
    expected_dir = data_dir / "expected_output"
    
    # 如果目录不存在，创建它
    expected_dir.mkdir(parents=True, exist_ok=True)
    
    # 创建子目录
    for adapter in ["opencode", "cursor", "claude"]:
        (expected_dir / adapter).mkdir(exist_ok=True)
    
    return expected_dir

@pytest.fixture
def create_skill_file(tmp_path):
    """
    创建技能文件的工厂fixture
    
    这个fixture返回一个函数，用于在临时目录中创建技能文件
    """
    def _create_skill(skill_name: str, content: str, file_type: str = "SKILL.md") -> Path:
        """
        创建技能文件
        
        Args:
            skill_name: 技能名称
            content: 文件内容
            file_type: 文件类型（SKILL.md 或 skill.yaml）
            
        Returns:
            文件路径
        """
        skill_dir = tmp_path / skill_name
        skill_dir.mkdir(exist_ok=True)
        
        file_path = skill_dir / file_type
        file_path.write_text(content, encoding='utf-8')
        
        return file_path
    
    return _create_skill

@pytest.fixture
def skill_validation_data():
    """
    返回技能验证测试数据
    
    这个fixture提供用于验证测试的数据
    """
    return {
        "valid_skill": {
            "name": "valid-test-skill",
            "description": "有效的测试技能",
            "version": "1.0.0",
            "compatibility": "all",
            "variables": [
                {"name": "VAR1", "default": "value1", "description": "变量1"}
            ]
        },
        "missing_fields": {
            "name": "missing-fields-skill",
            # 缺少description
            "version": "1.0.0"
        },
        "invalid_version": {
            "name": "invalid-version-skill",
            "description": "无效版本技能",
            "version": "not-a-version",  # 无效版本
            "compatibility": "all"
        },
        "invalid_variables": {
            "name": "invalid-vars-skill",
            "description": "无效变量技能",
            "version": "1.0.0",
            "variables": [
                {"name": "VAR1"}  # 缺少default和description
            ]
        }
    }

@pytest.fixture
def adapter_specific_skills():
    """
    返回适配器特定的技能模板
    
    这个fixture提供不同适配器的技能模板
    """
    return {
        "opencode": {
            "template": opencode_skill_template,
            "expected_files": ["manifest.yaml", "instructions.md"],
            "directory": ".agents/skills"
        },
        "cursor": {
            "template": cursor_skill_template,
            "expected_files": [".cursorrules"],
            "directory": "."
        },
        "claude": {
            "template": simple_skill_template,
            "expected_files": [".claude-config"],
            "directory": "."
        }
    }