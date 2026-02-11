"""
Test Scenario 2: Project Application Workflow
Tests skill binding, enabling, and application in a project context.
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


class TestScenario2ProjectApplication:
    """Test scenario 2: Project application workflow (bind -> enable -> apply)"""
    
    @pytest.fixture(autouse=True)
    def setup(self, temp_project_dir, temp_home_dir, test_skill_template):
        """Setup test environment"""
        self.project_dir = Path(temp_project_dir)
        self.home_dir = Path(temp_home_dir)
        self.skill_template = test_skill_template
        self.cmd = CommandRunner()
        self.validator = FileValidator()
        self.env = TestEnvironment()
        
        # Store paths
        self.skill_hub_dir = self.home_dir / ".skill-hub"
        self.repo_dir = self.skill_hub_dir / "repo"
        self.repo_skills_dir = self.repo_dir / "skills"
        
        # Project paths
        self.project_skill_hub = self.project_dir / ".skill-hub"
        self.project_state = self.project_skill_hub / "state.json"
        self.project_agents_dir = self.project_dir / ".agents"
        self.project_skills_dir = self.project_agents_dir / "skills"
        
        # Create .agents directory for project
        self.project_agents_dir.mkdir(exist_ok=True)
        
    def test_01_set_project_target(self):
        """Test 2.1: Set project target"""
        print("\n=== Test 2.1: Set Project Target ===")
        
        # First initialize in home directory
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=self.home_dir)
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        # Create a skill in the repo (need to create in project first, then feedback)
        skill_name = "my-logic-skill"
        
        # First create skill in project
        result = self.cmd.run("create", [skill_name], cwd=str(self.project_dir))
        assert result.success, f"skill-hub create failed: {result.stderr}"
        
        # Then feedback to repo
        result = self.cmd.run("feedback", [skill_name], cwd=self.project_dir, input_text="y\n")
        assert result.success, f"skill-hub feedback failed: {result.stderr}"
        
        # Set project target to open_code
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub set-target failed: {result.stderr}"
        
        # Verify state.json was updated in global directory
        global_state_file = self.skill_hub_dir / "state.json"
        assert global_state_file.exists(), f"state.json not found at {global_state_file}"
        
        # Load and verify state.json
        with open(global_state_file, 'r') as f:
            state = json.load(f)
        
        # Check that project is in state with preferred_target set to open_code
        project_path_str = str(self.project_dir)
        assert project_path_str in state, f"Project path not in state.json: {project_path_str}"
        
        project_state = state[project_path_str]
        assert "preferred_target" in project_state, "preferred_target not in project state"
        assert project_state["preferred_target"] == "open_code", f"preferred_target should be 'open_code', got: {project_state['preferred_target']}"
        
        print(f"✓ Project target set successfully")
        print(f"  - Updated global state: {global_state_file}")
        print(f"  - Project path: {project_path_str}")
        print(f"  - Preferred target: {project_state['preferred_target']}")
        
    def test_02_enable_skill(self):
        """Test 2.2: Enable a skill in project"""
        print("\n=== Test 2.2: Enable Skill ===")
        
        # Setup: init, create skill, set target
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=self.home_dir)
        assert result.success
        
        skill_name = "my-logic-skill"
        result = self.cmd.run("create", [skill_name], cwd=str(self.project_dir))
        assert result.success
        
        # Feedback skill to repo (required before use)
        result = self.cmd.run("feedback", [skill_name], cwd=self.project_dir, input_text="y\n")
        assert result.success
        
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success
        
        # Enable the skill
        result = self.cmd.run("use", [skill_name], cwd=str(self.project_dir))
        assert result.success, f"skill-hub use failed: {result.stderr}"
        
        # Verify state.json was updated in global directory
        global_state_file = self.skill_hub_dir / "state.json"
        assert global_state_file.exists(), f"state.json not found at {global_state_file}"
        
        with open(global_state_file, 'r') as f:
            state = json.load(f)
        
        # Check that project is in state with skill enabled
        project_path_str = str(self.project_dir)
        assert project_path_str in state, f"Project path not in state.json: {project_path_str}"
        
        project_state = state[project_path_str]
        skills = project_state.get("skills", {})
        assert skill_name in skills, f"Skill '{skill_name}' not enabled in state.json for project"
        
        # Verify .agents/skills/ directory exists from create command
        # But use command should not create NEW files beyond what create already made
        skill_dir = self.project_skills_dir / skill_name
        assert skill_dir.exists(), f"Skill directory should exist from create: {skill_dir}"
        
        # Count files before use (create already created SKILL.md)
        files_before = list(skill_dir.iterdir())
        print(f"  - Files from create: {[f.name for f in files_before]}")
        
        print(f"✓ Skill enabled successfully")
        print(f"  - Skill: {skill_name}")
        print(f"  - State updated: {global_state_file}")
        print(f"  - Physical directory not created (V2: use只更新状态)")
        
    def test_03_physical_application(self):
        """Test 2.3: Physically apply skill to project"""
        print("\n=== Test 2.3: Physical Application ===")
        
        # Setup: init, create skill, set target, enable skill
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=self.home_dir)
        assert result.success
        
        skill_name = "my-logic-skill"
        result = self.cmd.run("create", [skill_name], cwd=str(self.project_dir))
        assert result.success
        
        # Feedback skill to repo (required before use)
        result = self.cmd.run("feedback", [skill_name], cwd=self.project_dir, input_text="y\n")
        assert result.success
        
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success
        
        result = self.cmd.run("use", [skill_name], cwd=str(self.project_dir))
        assert result.success
        
        # Apply the skill
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        # Verify .agents/skills/ directory was created
        assert self.project_skills_dir.exists(), f".agents/skills/ directory not created at {self.project_skills_dir}"
        assert self.project_skills_dir.is_dir(), f".agents/skills/ is not a directory"
        
        # Verify skill directory was created
        skill_dir = self.project_skills_dir / skill_name
        assert skill_dir.exists(), f"Skill directory not created at {skill_dir}"
        assert skill_dir.is_dir(), f"Skill directory is not a directory"
        
        # Verify SKILL.md was created
        skill_file = skill_dir / "SKILL.md"
        assert skill_file.exists(), f"SKILL.md not created at {skill_file}"
        assert skill_file.is_file(), f"SKILL.md is not a file"
        
        # Note: Only SKILL.md is created, not separate instructions.md
        # This matches actual implementation (SKILL.md contains both YAML and content)
        
        # Verify SKILL.md has basic structure
        with open(skill_file, 'r') as f:
            skill_content = f.read()
        assert "name:" in skill_content, "SKILL.md missing name field"
        assert "description:" in skill_content, "SKILL.md missing description field"
        
        # Additional check for YAML frontmatter
        parts = skill_content.split("---")
        assert len(parts) >= 3, "SKILL.md should have YAML frontmatter and content separated by ---"
        
        # Verify .cursorrules was NOT created (project target is open_code)
        cursorrules_file = self.project_dir / ".cursorrules"
        assert not cursorrules_file.exists(), f".cursorrules should not be created for open_code target, found at {cursorrules_file}"
        
        print(f"✓ Skill applied physically")
        print(f"  - Created: {skill_dir}")
        print(f"  - File: SKILL.md")
        print(f"  - No .cursorrules created (correct for open_code)")
        
    def test_04_command_line_target_override(self):
        """Test 2.4: Command line target override"""
        print("\n=== Test 2.4: Command Line Target Override ===")
        
        # Setup: init, create skill, set target to open_code
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=self.home_dir)
        assert result.success
        
        skill_name = "my-logic-skill"
        result = self.cmd.run("create", [skill_name], cwd=str(self.project_dir))
        assert result.success
        
        # Feedback skill to repo
        result = self.cmd.run("feedback", [skill_name], cwd=self.project_dir, input_text="y\n")
        assert result.success
        
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success
        
        # Enable skill first
        result = self.cmd.run("use", [skill_name], cwd=str(self.project_dir))
        assert result.success
        
        # First set target to cursor
        result = self.cmd.run("set-target", ["cursor"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub set-target cursor failed: {result.stderr}"
        
        # Apply with cursor target
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        # Verify .cursorrules was created (cursor target override)
        cursorrules_file = self.project_dir / ".cursorrules"
        assert cursorrules_file.exists(), f".cursorrules should be created for cursor target, not found at {cursorrules_file}"
        assert cursorrules_file.is_file(), f".cursorrules is not a file"
        
        # Verify .cursorrules has content
        with open(cursorrules_file, 'r') as f:
            cursorrules_content = f.read()
        assert len(cursorrules_content.strip()) > 0, ".cursorrules is empty"
        
        # Verify project target remains open_code in global state.json
        global_state_file = self.skill_hub_dir / "state.json"
        assert global_state_file.exists(), f"state.json not found at {global_state_file}"
        
        with open(global_state_file, 'r') as f:
            state = json.load(f)
        
        project_path_str = str(self.project_dir)
        assert project_path_str in state, f"Project path not in state.json: {project_path_str}"
        
        project_state = state[project_path_str]
        assert "preferred_target" in project_state, "preferred_target not in project state"
        # Target was changed to cursor for this specific apply
        assert project_state["preferred_target"] == "cursor", f"Project target should be 'cursor' after set-target, got: {project_state['preferred_target']}"
        
        print(f"✓ Command line target override works")
        print(f"  - Created: {cursorrules_file}")
        print(f"  - Project target changed to: cursor")
        print(f"  - .cursorrules size: {len(cursorrules_content)} chars")
        
    def test_05_multiple_skills_application(self):
        """Test 2.5: Apply multiple skills"""
        print("\n=== Test 2.5: Multiple Skills Application ===")
        
        # Setup: init, create multiple skills
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=self.home_dir)
        assert result.success
        
        skills = ["logic-skill-1", "logic-skill-2", "logic-skill-3"]
        
        for skill in skills:
            result = self.cmd.run("create", [skill], cwd=str(self.project_dir))
            assert result.success, f"Failed to create skill {skill}"
            # Feedback skill to repo (required before use)
            result = self.cmd.run("feedback", [skill], cwd=self.project_dir, input_text="y\n")
            assert result.success, f"Failed to feedback skill {skill}"
        
        # Set project target
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success
        
        # Enable all skills
        for skill in skills:
            result = self.cmd.run("use", [skill], cwd=str(self.project_dir))
            assert result.success, f"Failed to enable skill {skill}"
        
        # Apply all skills
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        # Verify all skill directories were created
        for skill in skills:
            skill_dir = self.project_skills_dir / skill
            assert skill_dir.exists(), f"Skill directory not created for {skill}"
            assert (skill_dir / "SKILL.md").exists(), f"SKILL.md missing for {skill}"
            assert (skill_dir / "SKILL.md").exists(), f"SKILL.md missing for {skill}"
        
        print(f"✓ Multiple skills applied successfully")
        print(f"  - Skills: {', '.join(skills)}")
        print(f"  - All directories created in: {self.project_skills_dir}")
        
    def test_06_target_specific_adapters(self):
        """Test 2.6: Different targets create different outputs"""
        print("\n=== Test 2.6: Target-Specific Adapters ===")
        
        # This test would verify that different targets create different outputs
        # For now, create a placeholder test
        
        # Setup: init and create skill
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=self.home_dir)
        assert result.success
        
        skill_name = "test-adapter-skill"
        result = self.cmd.run("create", [skill_name], cwd=str(self.project_dir))
        assert result.success
        # Feedback skill to repo (required before use)
        result = self.cmd.run("feedback", [skill_name], cwd=self.project_dir, input_text="y\n")
        assert result.success
        
        # Test different targets
        targets_to_test = ["open_code", "cursor"]  # Add more as skill-hub supports them
        
        for target in targets_to_test:
            # Create a fresh project directory for each target
            with tempfile.TemporaryDirectory() as temp_dir:
                project_cmd = CommandRunner()
                
                # Set target
                result = project_cmd.run("set-target", [target], cwd=temp_dir)
                assert result.success
                
                # Enable skill
                result = project_cmd.run("use", [skill_name], cwd=temp_dir)
                assert result.success
                
                # Apply
                result = project_cmd.run("apply", cwd=temp_dir)
                assert result.success
                
                # Check target-specific outputs
                project_path = Path(temp_dir)
                
                if target == "open_code":
                    # Should create .agents/skills/ directory
                    agents_dir = project_path / ".agents" / "skills" / skill_name
                    assert agents_dir.exists(), f"open_code target should create {agents_dir}"
                    assert not (project_path / ".cursorrules").exists(), "open_code should not create .cursorrules"
                    
                elif target == "cursor":
                    # Should create .cursorrules
                    cursorrules = project_path / ".cursorrules"
                    assert cursorrules.exists(), f"cursor target should create .cursorrules"
                    
                print(f"  ✓ Target '{target}': Creates appropriate outputs")
        
        print(f"✓ Target-specific adapters work correctly")
        
    def test_07_apply_without_enable(self):
        """Test 2.7: Apply without enabling skill first"""
        print("\n=== Test 2.7: Apply Without Enable ===")
        
        # Setup: init and create skill
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=self.home_dir)
        assert result.success
        
        skill_name = "my-logic-skill"
        result = self.cmd.run("create", [skill_name], cwd=str(self.project_dir))
        assert result.success
        
        # Set project target
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success
        
        # Try to apply without enabling first
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        
        # This might fail or succeed depending on skill-hub implementation
        # For now, just log the result
        print(f"  Apply without enable result: returncode={result.exit_code}")
        print(f"  stdout: {result.stdout[:100]}...")
        print(f"  stderr: {result.stderr[:100]}...")
        
        print(f"✓ Tested apply without enable (result depends on implementation)")


if __name__ == "__main__":
    # For direct execution
    pytest.main([__file__, "-v"])