import os
import pytest
from typing import Dict, Any

@pytest.fixture
def initialized_project(isolated_env, command_runner):
    """
    返回已初始化的项目目录
    
    这个fixture会：
    1. 创建一个临时项目目录
    2. 执行 skill-hub init
    3. 返回项目目录路径
    """
    project_dir = isolated_env.project_dir
    
    # 初始化项目
    print(f"初始化项目: {project_dir}")
    result = command_runner.run("init", cwd=project_dir)
    
    if not result.success:
        # 如果初始化失败，尝试提供更多信息
        error_msg = f"项目初始化失败:\n命令: {result.command}\n退出码: {result.exit_code}\n错误: {result.stderr}"
        pytest.fail(error_msg)
    
    # 验证初始化成功
    skill_hub_dir = os.path.join(isolated_env.home_dir, ".skill-hub")
    if not os.path.exists(skill_hub_dir):
        pytest.fail(f"skill-hub配置目录未创建: {skill_hub_dir}")
    
    print(f"✅ 项目初始化成功: {project_dir}")
    return project_dir

@pytest.fixture
def project_with_target(initialized_project, command_runner):
    """
    返回已设置目标的项目目录
    
    这个fixture会：
    1. 初始化项目
    2. 设置目标为 open_code
    3. 返回项目目录路径
    """
    # 设置项目目标
    result = command_runner.run("set-target open_code", cwd=initialized_project)
    
    if not result.success:
        pytest.fail(f"设置项目目标失败: {result.stderr}")
    
    print(f"✅ 项目目标已设置为: open_code")
    return initialized_project

@pytest.fixture
def project_with_skill(project_with_target, command_runner, test_skill_data):
    """
    返回已创建技能的项目目录
    
    这个fixture会：
    1. 初始化项目并设置目标
    2. 创建测试技能
    3. 返回项目目录路径
    """
    # 创建测试技能
    skill_name = "my-logic-skill"
    result = command_runner.run(
        f"create {skill_name} --description '逻辑测试技能'",
        cwd=project_with_target
    )
    
    if not result.success:
        pytest.fail(f"创建技能失败: {result.stderr}")
    
    # 启用技能
    result = command_runner.run(f"use {skill_name}", cwd=project_with_target)
    
    if not result.success:
        # 如果use失败，可能技能已启用或不需要启用
        print(f"⚠️  启用技能失败（可能已启用）: {result.stderr}")
    
    print(f"✅ 技能已创建并启用: {skill_name}")
    return project_with_target

@pytest.fixture
def project_with_applied_skill(project_with_skill, command_runner):
    """
    返回已应用技能的项目目录
    
    这个fixture会：
    1. 初始化项目、设置目标、创建技能
    2. 应用技能到项目
    3. 返回项目目录路径
    """
    # 应用技能
    result = command_runner.run("apply", cwd=project_with_skill)
    
    if not result.success:
        pytest.fail(f"应用技能失败: {result.stderr}")
    
    print("✅ 技能已应用到项目")
    return project_with_skill

@pytest.fixture
def project_state(isolated_env) -> Dict[str, Any]:
    """
    返回项目状态信息
    
    这个fixture会读取并解析state.json文件
    """
    state_file = os.path.join(isolated_env.skill_hub_dir, "state.json")
    
    if not os.path.exists(state_file):
        return {}
    
    try:
        import json
        with open(state_file, 'r', encoding='utf-8') as f:
            return json.load(f)
    except Exception as e:
        print(f"⚠️  读取state.json失败: {e}")
        return {}

@pytest.fixture
def skill_hub_config(isolated_env) -> Dict[str, Any]:
    """
    返回skill-hub配置信息
    
    这个fixture会读取并解析config.yaml文件
    """
    config_file = os.path.join(isolated_env.skill_hub_dir, "config.yaml")
    
    if not os.path.exists(config_file):
        return {}
    
    try:
        import yaml
        with open(config_file, 'r', encoding='utf-8') as f:
            return yaml.safe_load(f) or {}
    except Exception as e:
        print(f"⚠️  读取config.yaml失败: {e}")
        return {}

@pytest.fixture
def clean_project():
    """
    返回一个干净的项目目录（未初始化）
    
    这个fixture用于需要从头开始的测试
    """
    import tempfile
    import shutil
    
    # 创建临时目录
    temp_dir = tempfile.mkdtemp(prefix="clean_project_")
    project_dir = os.path.join(temp_dir, "project")
    os.makedirs(project_dir, exist_ok=True)
    
    yield project_dir
    
    # 清理
    shutil.rmtree(temp_dir, ignore_errors=True)

@pytest.fixture
def project_with_git_repo(clean_project):
    """
    返回包含Git仓库的项目目录
    
    这个fixture会：
    1. 创建干净的项目目录
    2. 初始化Git仓库
    3. 返回项目目录路径
    """
    import subprocess
    
    # 初始化Git仓库
    try:
        subprocess.run(
            ["git", "init"],
            cwd=clean_project,
            capture_output=True,
            check=True
        )
        
        # 创建初始提交
        readme_file = os.path.join(clean_project, "README.md")
        with open(readme_file, 'w') as f:
            f.write("# Test Project\n\nThis is a test project for skill-hub e2e tests.")
        
        subprocess.run(
            ["git", "add", "."],
            cwd=clean_project,
            capture_output=True,
            check=True
        )
        
        subprocess.run(
            ["git", "commit", "-m", "Initial commit"],
            cwd=clean_project,
            capture_output=True,
            check=True
        )
        
        print(f"✅ Git仓库已初始化: {clean_project}")
    except Exception as e:
        print(f"⚠️  Git仓库初始化失败: {e}")
        # 继续使用目录，即使Git初始化失败
    
    return clean_project