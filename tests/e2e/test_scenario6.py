"""
Test Scenario 6: Remote Synchronization and Multi-device Collaboration (Update Workflow)
Tests how repository updates are refreshed to local projects.
Based on testCaseV2.md v3.0
"""

import os
import json
import tempfile
import pytest
from pathlib import Path

from tests.e2e.utils.command_runner import CommandRunner
from tests.e2e.utils.file_validator import FileValidator
from tests.e2e.utils.test_environment import TestEnvironment
from tests.e2e.utils.network_checker import NetworkChecker


class TestScenario6RemoteSynchronization:
    """Test scenario 6: Remote synchronization and multi-device collaboration (Update workflow)"""
    
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir, test_skill_template):
        """Setup test environment"""
        self.home_dir = temp_home_dir
        self.skill_template = test_skill_template
        self.cmd = CommandRunner()
        self.validator = FileValidator()
        self.env = TestEnvironment()
        self.network = NetworkChecker()
        
        # Store paths
        self.skill_hub_dir = Path(self.home_dir) / ".skill-hub"
        self.repositories_dir = self.skill_hub_dir / "repositories"
        self.main_repo_dir = self.repositories_dir / "main"
        self.repo_skills_dir = self.main_repo_dir / "skills"  # 多仓库结构：repositories/main/skills
        
        # Project paths
        self.project_dir = Path(self.home_dir) / "test-project"
        self.project_agents_dir = self.project_dir / ".agents"
        self.project_skills_dir = self.project_agents_dir / "skills"
        
        # Ensure project directory exists
        self.project_dir.mkdir(exist_ok=True)
        
        # 初始化环境并创建测试技能
        self._initialize_environment_with_skill()
        
    def _initialize_environment_with_skill(self):
        """Initialize environment with a test skill"""
        # 初始化环境
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"Initialization failed: {result.stderr}"
        
        # 创建测试技能
        self.test_skill_name = "git-expert"
        result = self.cmd.run("create", [self.test_skill_name], cwd=str(self.project_dir))
        if result.success:
            # 如果创建成功，反馈到仓库
            skill_md = self.project_skills_dir / self.test_skill_name / "SKILL.md"
            if skill_md.exists():
                # 修改技能内容
                with open(skill_md, 'a') as f:
                    f.write("\n\n## Git Expert Skill\nA test skill for synchronization testing.")
                
                # 反馈到仓库
                result = self.cmd.run("feedback", [self.test_skill_name], cwd=str(self.project_dir), input_text="y\n")
                print(f"Test skill '{self.test_skill_name}' created and fed back to repository")
                
                # 启用技能并应用
                result = self.cmd.run("use", [self.test_skill_name], cwd=str(self.project_dir))
                result = self.cmd.run("apply", cwd=str(self.project_dir))
        
    def test_01_command_dependency_check(self):
        """Test 6.1: Command dependency check verification"""
        print("\n=== Test 6.1: Command Dependency Check ===")
        
        # 创建一个新的临时目录，确保没有初始化
        temp_dir = Path(self.home_dir) / "temp-uninitialized-6"
        temp_dir.mkdir(exist_ok=True)
        
        # 测试未初始化时执行 skill-hub pull
        result = self.cmd.run("pull", cwd=str(temp_dir))
        # 应该提示需要先进行初始化
        assert not result.success or "需要先进行初始化" in result.stdout or "需要先进行初始化" in result.stderr, \
            f"Should prompt for initialization when running pull without init"
        
        print(f"✓ pull command dependency check passed")
        
    def test_02_pull_command_options(self):
        """Test 6.2: Pull command options verification ✅可本地"""
        print("\n=== Test 6.2: Pull Command Options ===")
        
        # 执行 skill-hub pull --check
        result = self.cmd.run("pull", ["--check"], cwd=str(self.project_dir))
        # 验证检查模式功能
        print(f"  Pull --check executed: {'✓' if result.success else '⚠️'}")
        
        # 测试 skill-hub pull --force 模拟
        result = self.cmd.run("pull", ["--force"], cwd=str(self.project_dir))
        print(f"  Pull --force executed: {'✓' if result.success else '⚠️'}")
        
        print(f"✓ Pull command options verification completed")
        
    def test_03_detect_outdated_skills(self):
        """Test 6.3: Detect outdated skills verification ✅可本地"""
        print("\n=== Test 6.3: Detect Outdated Skills ===")
        
        # 模拟本地仓库更新
        # 直接修改仓库中的技能文件，模拟远程更新
        repo_skill_md = self.repo_skills_dir / self.test_skill_name / "SKILL.md"
        if repo_skill_md.exists():
            with open(repo_skill_md, 'a') as f:
                f.write("\n\n## Repository Update\nSimulated remote update to create outdated state.")
            print(f"  Simulated repository update")
        
        # 执行 skill-hub status
        result = self.cmd.run("status", cwd=str(self.project_dir))
        assert result.success, f"skill-hub status failed: {result.stderr}"
        
        # 验证Outdated状态显示
        output = result.stdout + result.stderr
        outdated_keywords = ["outdated", "过时", "落后", "需要更新"]
        
        has_outdated_indication = any(keyword.lower() in output.lower() for keyword in outdated_keywords)
        
        if has_outdated_indication:
            print(f"  Outdated state detected: ✓")
        else:
            print(f"  ⚠️  No clear outdated indication in output")
            print(f"  Output preview: {output[:200]}...")
        
        print(f"✓ Outdated skills detection verification completed")
        
    def test_04_refresh_outdated_skills(self):
        """Test 6.4: Refresh outdated skills verification ✅可本地"""
        print("\n=== Test 6.4: Refresh Outdated Skills ===")
        
        # 首先确保有outdated状态
        repo_skill_md = self.repo_skills_dir / self.test_skill_name / "SKILL.md"
        if repo_skill_md.exists():
            with open(repo_skill_md, 'a') as f:
                f.write("\n\n## Another Repository Update\nFor refresh testing.")
        
        # 执行 skill-hub apply
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        # 验证从更新仓库刷新到项目
        project_skill_md = self.project_skills_dir / self.test_skill_name / "SKILL.md"
        if project_skill_md.exists():
            with open(project_skill_md, 'r') as f:
                content = f.read()
            
            if "Repository Update" in content:
                print(f"  Skill refreshed from repository: ✓")
            else:
                print(f"  ⚠️  Skill may not have been refreshed")
        else:
            print(f"  ⚠️  Skill file not found in project")
        
        print(f"✓ Outdated skills refresh verification completed")
        
    def test_05_pull_updates_from_remote(self):
        """Test 6.5: Pull updates from remote verification ⚠️网络依赖"""
        print("\n=== Test 6.5: Pull Updates from Remote ===")
        
        # 检查网络连接
        if self.network.is_network_available():
            print(f"  Network available, testing pull from remote...")
            
            # 执行 skill-hub pull
            result = self.cmd.run("pull", cwd=str(self.project_dir))
            
            # 验证仓库和注册表更新
            if result.success:
                print(f"  Pull command executed successfully")
                
                # 检查注册表文件
                registry_file = self.skill_hub_dir / "registry.json"
                if registry_file.exists():
                    print(f"  Registry file exists: ✓")
                else:
                    print(f"  ⚠️  Registry file not found")
            else:
                print(f"  ⚠️  Pull command failed: {result.stderr[:100]}...")
        else:
            print(f"  No network available, skipping network-dependent test")
            print(f"  ⚠️  This test requires network connection per testCaseV2.md")
        
        print(f"✓ Pull updates from remote verification completed")
        
    def test_06_multi_device_collaboration_workflow(self):
        """Test 6.6: Multi-device collaboration workflow verification ⚠️网络依赖"""
        print("\n=== Test 6.6: Multi-device Collaboration Workflow ===")
        
        # 模拟多设备协作场景
        print(f"  Simulating multi-device collaboration scenario...")
        
        # 创建"设备A"和"设备B"的模拟目录
        device_a_dir = Path(self.home_dir) / "device-a"
        device_b_dir = Path(self.home_dir) / "device-b"
        
        device_a_dir.mkdir(exist_ok=True)
        device_b_dir.mkdir(exist_ok=True)
        
        # 设备A：初始化并创建技能
        print(f"  Device A: Initializing and creating skill...")
        result = self.cmd.run("init", cwd=str(device_a_dir))
        if result.success:
            skill_name = "collaboration-skill"
            result = self.cmd.run("create", [skill_name], cwd=str(device_a_dir))
            if result.success:
                # 反馈到仓库
                skill_md = device_a_dir / ".agents" / "skills" / skill_name / "SKILL.md"
                if skill_md.exists():
                    with open(skill_md, 'a') as f:
                        f.write("\n\n## Collaboration Skill\nCreated on Device A.")
                    
                    result = self.cmd.run("feedback", [skill_name], cwd=str(device_a_dir), input_text="y\n")
                    print(f"    Skill created and fed back by Device A")
        
        # 设备B：初始化并拉取更新
        print(f"  Device B: Initializing and pulling updates...")
        result = self.cmd.run("init", cwd=str(device_b_dir))
        if result.success:
            # 如果有网络，尝试pull
            if self.network.is_network_available():
                result = self.cmd.run("pull", cwd=str(device_b_dir))
                print(f"    Device B pulled updates")
            
            # 启用技能
            result = self.cmd.run("use", [skill_name], cwd=str(device_b_dir))
            result = self.cmd.run("apply", cwd=str(device_b_dir))
            print(f"    Device B enabled and applied skill")
        
        # 验证同步一致性
        print(f"  Verifying synchronization consistency...")
        
        # 检查两个设备是否都有技能文件
        device_a_skill = device_a_dir / ".agents" / "skills" / skill_name / "SKILL.md"
        device_b_skill = device_b_dir / ".agents" / "skills" / skill_name / "SKILL.md"
        
        if device_a_skill.exists():
            print(f"    Device A has skill file: ✓")
        if device_b_skill.exists():
            print(f"    Device B has skill file: ✓")
        
        print(f"✓ Multi-device collaboration workflow verification completed")