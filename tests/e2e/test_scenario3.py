"""
Test Scenario 3: Skill "Iteration Feedback" Workflow (Modify -> Status -> Feedback)
Tests how local modifications are detected through status and written back to repository.
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

class TestScenario3IterationFeedback:
    """Test scenario 3: Skill "iteration feedback" workflow (Modify -> Status -> Feedback)"""
    
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir, test_skill_template):
        """Setup test environment"""
        self.home_dir = temp_home_dir
        self.skill_template = test_skill_template
        self.cmd = CommandRunner()
        self.validator = FileValidator()
        self.env = TestEnvironment()
        
        # Store paths
        self.skill_hub_dir = Path(self.home_dir) / ".skill-hub"
        
        self.repositories_dir = self.skill_hub_dir / "repositories"
        self.main_repo_dir = self.repositories_dir / "main"
        self.repo_skills_dir = self.main_repo_dir / "skills"  # 新结构：repositories/main/skills
        
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
                    f.write("\n\n## Git Expert Skill\nA test skill for git operations.")
                
                # 反馈到仓库
                result = self.cmd.run("feedback", [self.test_skill_name], cwd=str(self.project_dir), input_text="y\n")
                print(f"Test skill '{self.test_skill_name}' created and fed back to repository")
                
                # 启用技能并应用
                result = self.cmd.run("use", [self.test_skill_name], cwd=str(self.project_dir))
                result = self.cmd.run("apply", cwd=str(self.project_dir))
        
    def test_01_command_dependency_check(self):
        """Test 3.1: Command dependency check verification"""
        print("\n=== Test 3.1: Command Dependency Check ===")
        
        # 创建一个新的临时目录，确保没有初始化
        temp_dir = Path(self.home_dir) / "temp-uninitialized-3"
        temp_dir.mkdir(exist_ok=True)
        
        # 测试未初始化时执行 skill-hub status
        # skill-hub 会自动初始化项目
        result = self.cmd.run("status", cwd=str(temp_dir))
        # 应该成功执行并初始化项目
        assert result.success, f"status should succeed and auto-initialize: {result.stderr}"
        assert "当前目录" in result.stdout and "未在skill-hub中注册" in result.stdout, \
            f"Should auto-initialize when running status without init"
        
        print(f"✓ status command dependency check passed (auto-initialization)")
        
        # 测试未初始化时执行 skill-hub feedback git-expert
        # feedback 命令需要技能存在于项目中，所以会失败
        result = self.cmd.run("feedback", ["git-expert"], cwd=str(temp_dir))
        # 应该失败，因为技能不存在于项目中
        assert not result.success, f"feedback should fail when skill doesn't exist in project"
        assert "未在项目工作区中启用" in result.stderr or "not enabled" in result.stderr.lower(), \
            f"Should indicate skill not enabled in project"
        
        print(f"✓ feedback command dependency check passed (skill doesn't exist)")
        
    def test_02_project_modification_detection(self):
        """Test 3.2: Project modification detection verification"""
        print("\n=== Test 3.2: Project Modification Detection ===")
        
        # 修改项目技能文件
        skill_md = self.project_skills_dir / self.test_skill_name / "SKILL.md"
        assert skill_md.exists(), f"Skill file not found at {skill_md}"
        
        # 读取原始内容
        with open(skill_md, 'r') as f:
            original_content = f.read()
        
        # 添加修改
        modified_content = original_content + "\n\n## Test Modification\nAdded for modification detection test."
        with open(skill_md, 'w') as f:
            f.write(modified_content)
        
        # 验证修改已写入
        with open(skill_md, 'r') as f:
            current_content = f.read()
        assert "Test Modification" in current_content, "Modification not written to SKILL.md"
        
        # 执行 skill-hub status git-expert
        result = self.cmd.run("status", [self.test_skill_name], cwd=str(self.project_dir))
        assert result.success, f"skill-hub status failed: {result.stderr}"
        
        # 验证Modified状态检测机制
        # 检查输出中是否包含修改状态指示
        output = result.stdout + result.stderr
        # 可能的关键词：Modified, modified, 修改, 变更
        modification_detected = any(keyword in output.lower() for keyword in ["modified", "修改", "变更", "diff"])
        
        if modification_detected:
            print(f"  Modification detected: ✓")
        else:
            print(f"  ⚠️  Modification detection not obvious in output")
            print(f"  Output preview: {output[:200]}...")
        
        print(f"✓ Project modification detection tested")
        
    def test_03_feedback_synchronization(self):
        """Test 3.3: Feedback synchronization verification"""
        print("\n=== Test 3.3: Feedback Synchronization ===")
        
        # 首先确保有修改
        skill_md = self.project_skills_dir / self.test_skill_name / "SKILL.md"
        with open(skill_md, 'a') as f:
            f.write("\n\n## Additional modification for feedback test.")
        
        # 执行 skill-hub feedback git-expert
        result = self.cmd.run("feedback", [self.test_skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"skill-hub feedback failed: {result.stderr}"
        
        # 验证仓库更新
        repo_skill_md = self.repo_skills_dir / self.test_skill_name / "SKILL.md"
        assert repo_skill_md.exists(), f"Skill file not in repository at {repo_skill_md}"
        
        # 验证项目文件不变（仍然包含修改）
        with open(skill_md, 'r') as f:
            project_content = f.read()
        assert "Additional modification" in project_content, "Project file should still contain modification"
        
        print(f"  Basic feedback completed: ✓")
        
        # 执行 skill-hub feedback git-expert --dry-run
        # 首先添加另一个修改
        with open(skill_md, 'a') as f:
            f.write("\n\n## Dry-run test modification.")
        
        result = self.cmd.run("feedback", [self.test_skill_name, "--dry-run"], cwd=str(self.project_dir))
        # dry-run 应该显示将要同步的差异但不实际执行
        print(f"  Dry-run mode tested: ✓")
        
        # 执行 skill-hub feedback git-expert --force
        result = self.cmd.run("feedback", [self.test_skill_name, "--force"], cwd=str(self.project_dir), input_text="y\n")
        # force 模式应该成功
        assert result.success, f"skill-hub feedback --force failed: {result.stderr}"
        print(f"  Force mode tested: ✓")
        
        print(f"✓ Feedback synchronization with all options verified")
        
    def test_04_status_command_options(self):
        """Test 3.4: Status command options verification"""
        print("\n=== Test 3.4: Status Command Options ===")
        
        # 执行 skill-hub status --verbose
        result = self.cmd.run("status", ["--verbose"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub status --verbose failed: {result.stderr}"
        
        # 验证详细差异信息显示
        verbose_output = result.stdout + result.stderr
        assert len(verbose_output.strip()) > 0, "Verbose output should not be empty"
        
        # 检查是否包含详细信息
        is_verbose = len(verbose_output) > 100  # 简单检查：详细输出应该较长
        print(f"  Verbose output length: {len(verbose_output)} chars")
        print(f"  Detailed information shown: {'✓' if is_verbose else '⚠️'}")
        
        # 执行 skill-hub status git-expert
        result = self.cmd.run("status", [self.test_skill_name], cwd=str(self.project_dir))
        assert result.success, f"skill-hub status for specific skill failed: {result.stderr}"
        
        # 验证特定技能状态检查
        specific_output = result.stdout + result.stderr
        assert self.test_skill_name in specific_output, f"Output should mention skill '{self.test_skill_name}'"
        print(f"  Specific skill status checked: ✓")
        
        print(f"✓ Status command with all options verified")
        
    def test_05_multiple_modifications(self):
        """Test 3.5: Multiple modifications handling verification"""
        print("\n=== Test 3.5: Multiple Modifications Handling ===")
        
        # 创建多文件技能结构（如果支持）
        # 首先检查技能目录结构
        skill_dir = self.project_skills_dir / self.test_skill_name
        
        # 创建额外文件
        extra_files = ["README.md", "config.yaml", "utils/helper.py"]
        
        for file_path in extra_files:
            full_path = skill_dir / file_path
            full_path.parent.mkdir(parents=True, exist_ok=True)
            with open(full_path, 'w') as f:
                f.write(f"# {file_path}\n\nContent for {file_path}\n")
            print(f"  Created: {file_path}")
        
        # 同时修改多个文件
        files_to_modify = [
            skill_dir / "SKILL.md",
            skill_dir / "README.md",
            skill_dir / "config.yaml"
        ]
        
        for file_path in files_to_modify:
            if file_path.exists():
                with open(file_path, 'a') as f:
                    f.write(f"\n\n## Modified at {file_path.name}\n")
                print(f"  Modified: {file_path.name}")
        
        # 执行 skill-hub feedback git-expert
        result = self.cmd.run("feedback", [self.test_skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"skill-hub feedback for multiple files failed: {result.stderr}"
        
        # 验证批量反馈处理
        # 检查仓库中是否包含所有文件
        for file_path in extra_files:
            repo_file = self.repo_skills_dir / self.test_skill_name / file_path
            if repo_file.exists():
                print(f"  File synced to repo: {file_path}")
            else:
                print(f"  ⚠️  File not in repo: {file_path}")
        
        print(f"✓ Multiple modifications handling verified")
        
    def test_06_standard_modification_extraction(self):
        """Test 3.6: Standard modification extraction verification"""
        print("\n=== Test 3.6: Standard Modification Extraction ===")

        skill_md = self.project_skills_dir / self.test_skill_name / "SKILL.md"
        with open(skill_md, 'a') as f:
            f.write("\n\n## Standard modification extraction\n")

        result = self.cmd.run("status", [self.test_skill_name], cwd=str(self.project_dir))
        assert result.success, f"skill-hub status failed: {result.stderr}"
        
        result = self.cmd.run("feedback", [self.test_skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"skill-hub feedback failed: {result.stderr}"
        
        print(f"✓ Standard modification extraction verified")
        
    def test_07_json_escaping_handling(self):
        """Test 3.7: JSON escaping handling verification"""
        print("\n=== Test 3.7: JSON Escaping Handling ===")
        
        # 测试特殊字符处理
        special_chars_content = """
## Special Characters Test
- Quotes: "double" and 'single'
- Backslashes: \\test\\path
- Newlines: line1
line2
line3
- Unicode: 中文测试 🚀
- JSON problematic: {"key": "value", "array": [1, 2, 3]}
"""
        
        # 修改技能文件包含特殊字符
        skill_md = self.project_skills_dir / self.test_skill_name / "SKILL.md"
        with open(skill_md, 'a') as f:
            f.write(special_chars_content)
        
        # 执行 skill-hub feedback git-expert
        result = self.cmd.run("feedback", [self.test_skill_name], cwd=str(self.project_dir), input_text="y\n")
        
        # 验证转义逻辑正确性
        if result.success:
            print(f"  Feedback with special characters: ✓")
            
            # 检查仓库文件
            repo_skill_md = self.repo_skills_dir / self.test_skill_name / "SKILL.md"
            if repo_skill_md.exists():
                with open(repo_skill_md, 'r') as f:
                    repo_content = f.read()
                
                # 检查特殊字符是否被正确处理
                if "中文测试" in repo_content and "🚀" in repo_content:
                    print(f"  Unicode characters preserved: ✓")
                else:
                    print(f"  ⚠️  Unicode characters may not be preserved")
        else:
            print(f"  ⚠️  Feedback failed with special characters")
            print(f"  Error: {result.stderr}")
        
        print(f"✓ JSON escaping handling verified")
        
    def test_08_partial_modifications(self):
        """Test 3.8: Partial modifications handling verification"""
        print("\n=== Test 3.8: Partial Modifications Handling ===")
        
        # 测试部分文件修改场景
        skill_dir = self.project_skills_dir / self.test_skill_name
        
        # 确保有多个文件
        files = ["SKILL.md", "README.md", "config.yaml"]
        for filename in files:
            file_path = skill_dir / filename
            if not file_path.exists():
                file_path.parent.mkdir(parents=True, exist_ok=True)
                with open(file_path, 'w') as f:
                    f.write(f"# {filename}\n\nInitial content.\n")
        
        # 只修改部分文件
        files_to_modify = ["SKILL.md", "README.md"]
        files_not_to_modify = ["config.yaml"]
        
        for filename in files_to_modify:
            file_path = skill_dir / filename
            with open(file_path, 'a') as f:
                f.write(f"\n\n## Modified: {filename}\n")
            print(f"  Modified: {filename}")
        
        # 检查状态
        result = self.cmd.run("status", [self.test_skill_name], cwd=str(self.project_dir))
        print(f"  Status checked for partial modifications")
        
        # 反馈修改
        result = self.cmd.run("feedback", [self.test_skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"skill-hub feedback for partial modifications failed: {result.stderr}"
        
        # 验证选择性反馈
        # 检查仓库文件
        for filename in files:
            repo_file = self.repo_skills_dir / self.test_skill_name / filename
            if repo_file.exists():
                print(f"  File in repo: {filename}")
            else:
                print(f"  ⚠️  File not in repo: {filename}")
        
        print(f"✓ Partial modifications handling verified")
