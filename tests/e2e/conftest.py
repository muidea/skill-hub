import pytest
import os
import sys
import tempfile
import shutil
import json
from pathlib import Path

# 添加utils目录到Python路径
sys.path.insert(0, str(Path(__file__).parent / "utils"))

# 自定义标记
def pytest_configure(config):
    config.addinivalue_line(
        "markers", "scenario1: 场景1测试 - 开发者全流程"
    )
    config.addinivalue_line(
        "markers", "scenario2: 场景2测试 - 项目应用流程"
    )
    config.addinivalue_line(
        "markers", "scenario3: 场景3测试 - 迭代反馈流程"
    )
    config.addinivalue_line(
        "markers", "scenario4: 场景4测试 - 取消与清理流程"
    )
    config.addinivalue_line(
        "markers", "scenario5: 场景5测试 - 更新与校验流程"
    )
    config.addinivalue_line(
        "markers", "requires_network: 需要网络连接的测试"
    )
    config.addinivalue_line(
        "markers", "no_debug: 测试失败时不保留临时文件"
    )

@pytest.fixture(scope="function")
def isolated_env(request):
    """为每个测试提供完全隔离的环境（失败时保留）"""
    from utils.test_environment import TestEnvironment
    
    # 从测试标记获取是否保留临时文件
    keep_on_failure = True  # 默认保留
    if hasattr(request.node, 'get_closest_marker'):
        if request.node.get_closest_marker('no_debug'):
            keep_on_failure = False
    
    with TestEnvironment(
        prefix=f"test_{request.node.name}_",
        keep_on_failure=keep_on_failure
    ) as env:
        yield env

@pytest.fixture
def command_runner(isolated_env):
    """命令运行器（带调试模式）"""
    from utils.command_runner import CommandRunner
    return CommandRunner(debug=True)

@pytest.fixture
def file_validator():
    """文件验证器（严格模式）"""
    from utils.file_validator import FileValidator
    return FileValidator(strict=True)

@pytest.fixture
def test_skill_data():
    """加载测试技能数据"""
    data_dir = Path(__file__).parent / "data" / "test_skills" / "my-logic-skill"
    
    skill_data = {}
    
    # 加载SKILL.md
    skill_path = data_dir / "SKILL.md"
    if skill_path.exists():
        with open(skill_path, 'r', encoding='utf-8') as f:
            skill_data['skill_md'] = f.read()
    else:
        skill_data['skill_md'] = ""
    
    # 加载期望输出
    expected_dir = data_dir / "expected_output"
    if expected_dir.exists():
        for adapter in ['opencode', 'cursor', 'claude']:
            adapter_dir = expected_dir / adapter
            if adapter_dir.exists():
                for file in adapter_dir.glob("*.md"):
                    with open(file, 'r', encoding='utf-8') as f:
                        key = f"expected_{adapter}_{file.stem}"
                        skill_data[key] = f.read()
    
    return skill_data

@pytest.fixture
def network_checker():
    """网络检查器"""
    from utils.network_checker import NetworkChecker
    return NetworkChecker

@pytest.fixture
def temp_home_dir():
    """临时HOME目录fixture"""
    import tempfile
    import os
    
    # 创建临时目录作为HOME
    temp_dir = tempfile.mkdtemp(prefix="skill_hub_test_home_")
    
    # 保存原始HOME
    original_home = os.environ.get('HOME')
    
    # 设置临时HOME
    os.environ['HOME'] = temp_dir
    
    yield temp_dir
    
    # 恢复原始HOME
    if original_home:
        os.environ['HOME'] = original_home
    else:
        del os.environ['HOME']
    
    # 清理临时目录
    import shutil
    try:
        shutil.rmtree(temp_dir)
    except:
        pass  # 忽略清理错误

@pytest.fixture
def temp_project_dir():
    """临时项目目录fixture"""
    import tempfile
    import os
    
    # 创建临时项目目录
    temp_dir = tempfile.mkdtemp(prefix="skill_hub_test_project_")
    
    yield temp_dir
    
    # 清理临时目录
    import shutil
    try:
        shutil.rmtree(temp_dir)
    except:
        pass  # 忽略清理错误

@pytest.fixture
def test_skill_template():
    """测试技能模板fixture"""
    data_dir = Path(__file__).parent / "data" / "test_skills" / "my-logic-skill"
    skill_file = data_dir / "SKILL.md"
    
    if skill_file.exists():
        with open(skill_file, 'r', encoding='utf-8') as f:
            return f.read()
    else:
        # 返回默认模板
        return """# Test Skill

A test skill for end-to-end testing.

## Description
This is a test skill used for automated testing.

## Usage
Test usage instructions.

## Testing
This skill is used for testing purposes only."""