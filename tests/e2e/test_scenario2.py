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
        self.project_dir = temp_project_dir
        self.home_dir = temp_home_dir
        self.skill_template = test_skill_template
        self.cmd = CommandRunner(workdir=str(self.project_dir))
        self.validator = FileValidator()
        self.env = TestEnvironment()
        
        # Store paths
        self.skill_hub_dir = Path(self.home_dir) / ".skill-hub"
        self.repo_dir = self.skill_hub_dir / "repo"
        self.skills_dir = self.repo_dir / "skills"
        
        # Project paths
        self.project_skill_hub = self.project_dir / ".skill-hub"
        self.project_state = self.project_skill_hub / "state.json"
        self.agents_skills_dir = self.project_dir / ".agents" / "skills"
        
        # Change to project directory
        os.chdir(self.project_dir)
        
    def test_01_set_project_target(self):
        """Test 2.1: Set project target"""
        print("\n=== Test 2.1: Set Project Target ===")
        
        # First initialize in home directory
        home_cmd = CommandRunner()
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        # Create a skill in the repo
        skill_name = "my-logic-skill"
        result = home_cmd.run(f"skill-hub create {skill_name}", timeout=30)
        assert result.success, f"skill-hub create failed: {result.stderr}"
        
        # Set project target to open_code
        result = self.cmd.run("set-target open_code")
        assert result.success, f"skill-hub set-target failed: {result.stderr}"
        
        # Verify state.json was created
        assert self.project_state.exists(), f"state.json not created at {self.project_state}"
        assert self.project_state.is_file(), f"state.json is not a file"
        
        # Verify state.json contains the target
        with open(self.project_state, 'r') as f:
            state = json.load(f)
        
        assert 'target' in state, "state.json missing 'target' field"
        assert state['target'] == 'open_code', f"Target should be 'open_code', got: {state['target']}"
        
        print(f"✓ Project target set successfully")
        print(f"  - Created: {self.project_state}")
        print(f"  - Target: {state['target']}")
        print(f"  - Full state: {json.dumps(state, indent=2)}")
        
    def test_02_enable_skill(self):
        """Test 2.2: Enable a skill in project"""
        print("\n=== Test 2.2: Enable Skill ===")
        
        # Setup: init, create skill, set target
        home_cmd = CommandRunner()
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success
        
        skill_name = "my-logic-skill"
        result = home_cmd.run(f"skill-hub create {skill_name}", timeout=30)
        assert result.success
        
        result = self.cmd.run("set-target open_code")
        assert result.success
        
        # Enable the skill
        result = self.cmd.run("use", {skill_name})
        assert result.success, f"skill-hub use failed: {result.stderr}"
        
        # Verify state.json was updated
        assert self.project_state.exists(), f"state.json not found at {self.project_state}"
        
        with open(self.project_state, 'r') as f:
            state = json.load(f)
        
        # Check that skill is in enabled_skills or similar field
        # The exact field name depends on skill-hub implementation
        skill_found = False
        for key, value in state.items():
            if isinstance(value, list) and skill_name in value:
                skill_found = True
                break
            elif isinstance(value, dict) and skill_name in value:
                skill_found = True
                break
        
        assert skill_found, f"Skill '{skill_name}' not found in state.json"
        
        # Verify .agents/skills/ directory should NOT exist yet (only state enabled)
        assert not self.agents_skills_dir.exists(), f".agents/skills/ should not exist yet, found at {self.agents_skills_dir}"
        
        print(f"✓ Skill enabled successfully")
        print(f"  - Skill: {skill_name}")
        print(f"  - State updated: {self.project_state}")
        print(f"  - Physical directory not created (correct)")
        
    def test_03_physical_application(self):
        """Test 2.3: Physically apply skill to project"""
        print("\n=== Test 2.3: Physical Application ===")
        
        # Setup: init, create skill, set target, enable skill
        home_cmd = CommandRunner()
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success
        
        skill_name = "my-logic-skill"
        result = home_cmd.run(f"skill-hub create {skill_name}", timeout=30)
        assert result.success
        
        result = self.cmd.run("set-target open_code")
        assert result.success
        
        result = self.cmd.run("use", {skill_name})
        assert result.success
        
        # Apply the skill
        result = self.cmd.run("apply")
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        # Verify .agents/skills/ directory was created
        assert self.agents_skills_dir.exists(), f".agents/skills/ directory not created at {self.agents_skills_dir}"
        assert self.agents_skills_dir.is_dir(), f".agents/skills/ is not a directory"
        
        # Verify skill directory was created
        skill_dir = self.agents_skills_dir / skill_name
        assert skill_dir.exists(), f"Skill directory not created at {skill_dir}"
        assert skill_dir.is_dir(), f"Skill directory is not a directory"
        
        # Verify manifest.yaml was created
        manifest_file = skill_dir / "manifest.yaml"
        assert manifest_file.exists(), f"manifest.yaml not created at {manifest_file}"
        assert manifest_file.is_file(), f"manifest.yaml is not a file"
        
        # Verify instructions.md was created
        instructions_file = skill_dir / "instructions.md"
        assert instructions_file.exists(), f"instructions.md not created at {instructions_file}"
        assert instructions_file.is_file(), f"instructions.md is not a file"
        
        # Verify manifest.yaml has basic structure
        with open(manifest_file, 'r') as f:
            manifest_content = f.read()
        assert "name:" in manifest_content, "manifest.yaml missing name field"
        assert "description:" in manifest_content, "manifest.yaml missing description field"
        
        # Verify instructions.md has content
        with open(instructions_file, 'r') as f:
            instructions_content = f.read()
        assert len(instructions_content.strip()) > 0, "instructions.md is empty"
        
        # Verify .cursorrules was NOT created (project target is open_code)
        cursorrules_file = self.project_dir / ".cursorrules"
        assert not cursorrules_file.exists(), f".cursorrules should not be created for open_code target, found at {cursorrules_file}"
        
        print(f"✓ Skill applied physically")
        print(f"  - Created: {skill_dir}")
        print(f"  - Files: manifest.yaml, instructions.md")
        print(f"  - No .cursorrules created (correct for open_code)")
        
    def test_04_command_line_target_override(self):
        """Test 2.4: Command line target override"""
        print("\n=== Test 2.4: Command Line Target Override ===")
        
        # Setup: init, create skill, set target to open_code
        home_cmd = CommandRunner()
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success
        
        skill_name = "my-logic-skill"
        result = home_cmd.run(f"skill-hub create {skill_name}", timeout=30)
        assert result.success
        
        result = self.cmd.run("set-target open_code")
        assert result.success
        
        # Apply with cursor target override
        result = self.cmd.run("apply --target cursor")
        assert result.success, f"skill-hub apply --target cursor failed: {result.stderr}"
        
        # Verify .cursorrules was created (cursor target override)
        cursorrules_file = self.project_dir / ".cursorrules"
        assert cursorrules_file.exists(), f".cursorrules should be created for cursor target, not found at {cursorrules_file}"
        assert cursorrules_file.is_file(), f".cursorrules is not a file"
        
        # Verify .cursorrules has content
        with open(cursorrules_file, 'r') as f:
            cursorrules_content = f.read()
        assert len(cursorrules_content.strip()) > 0, ".cursorrules is empty"
        
        # Verify project target remains open_code in state.json
        assert self.project_state.exists(), f"state.json not found at {self.project_state}"
        
        with open(self.project_state, 'r') as f:
            state = json.load(f)
        
        assert state.get('target') == 'open_code', f"Project target should remain 'open_code', got: {state.get('target')}"
        
        print(f"✓ Command line target override works")
        print(f"  - Created: {cursorrules_file}")
        print(f"  - Project target remains: open_code")
        print(f"  - .cursorrules size: {len(cursorrules_content)} chars")
        
    def test_05_multiple_skills_application(self):
        """Test 2.5: Apply multiple skills"""
        print("\n=== Test 2.5: Multiple Skills Application ===")
        
        # Setup: init, create multiple skills
        home_cmd = CommandRunner()
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success
        
        skills = ["logic-skill-1", "logic-skill-2", "logic-skill-3"]
        
        for skill in skills:
            result = home_cmd.run(f"skill-hub create {skill}", timeout=30)
            assert result.success, f"Failed to create skill {skill}"
        
        # Set project target
        result = self.cmd.run("set-target open_code")
        assert result.success
        
        # Enable all skills
        for skill in skills:
            result = self.cmd.run(f"skill-hub use {skill}", timeout=30)
            assert result.success, f"Failed to enable skill {skill}"
        
        # Apply all skills
        result = self.cmd.run("apply")
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        # Verify all skill directories were created
        for skill in skills:
            skill_dir = self.agents_skills_dir / skill
            assert skill_dir.exists(), f"Skill directory not created for {skill}"
            assert (skill_dir / "manifest.yaml").exists(), f"manifest.yaml missing for {skill}"
            assert (skill_dir / "instructions.md").exists(), f"instructions.md missing for {skill}"
        
        print(f"✓ Multiple skills applied successfully")
        print(f"  - Skills: {', '.join(skills)}")
        print(f"  - All directories created in: {self.agents_skills_dir}")
        
    def test_06_target_specific_adapters(self):
        """Test 2.6: Different targets create different outputs"""
        print("\n=== Test 2.6: Target-Specific Adapters ===")
        
        # This test would verify that different targets create different outputs
        # For now, create a placeholder test
        
        # Setup: init and create skill
        home_cmd = CommandRunner()
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success
        
        skill_name = "test-adapter-skill"
        result = home_cmd.run(f"skill-hub create {skill_name}", timeout=30)
        assert result.success
        
        # Test different targets
        targets_to_test = ["open_code", "cursor"]  # Add more as skill-hub supports them
        
        for target in targets_to_test:
            # Create a fresh project directory for each target
            with tempfile.TemporaryDirectory() as temp_dir:
                project_cmd = CommandRunner(workdir=temp_dir)
                
                # Set target
                result = project_cmd.run(f"skill-hub set-target {target}", timeout=30)
                assert result.success
                
                # Enable skill
                result = project_cmd.run(f"skill-hub use {skill_name}", timeout=30)
                assert result.success
                
                # Apply
                result = project_cmd.run("skill-hub apply", timeout=30)
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
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success
        
        skill_name = "my-logic-skill"
        result = home_cmd.run(f"skill-hub create {skill_name}", timeout=30)
        assert result.success
        
        # Set project target
        result = self.cmd.run("set-target open_code")
        assert result.success
        
        # Try to apply without enabling first
        result = self.cmd.run("apply")
        
        # This might fail or succeed depending on skill-hub implementation
        # For now, just log the result
        print(f"  Apply without enable result: returncode={result.exit_code}")
        print(f"  stdout: {result.stdout[:100]}...")
        print(f"  stderr: {result.stderr[:100]}...")
        
        print(f"✓ Tested apply without enable (result depends on implementation)")


if __name__ == "__main__":
    # For direct execution
    pytest.main([__file__, "-v"])