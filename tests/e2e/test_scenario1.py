"""
Test Scenario 1: Developer Full Workflow
Tests the complete developer workflow from initialization to skill creation and submission.
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


class TestScenario1DeveloperWorkflow:
    """Test scenario 1: Developer full workflow (init -> create -> feedback)"""
    
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
        self.repo_dir = self.skill_hub_dir / "repo"
        self.skills_dir = self.repo_dir / "skills"
        
    def test_01_environment_initialization(self):
        """Test 1.1: Environment initialization with skill-hub init"""
        print("\n=== Test 1.1: Environment Initialization ===")
        
        # Run skill-hub init
        result = self.cmd.run("init")
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        # Verify ~/.skill-hub directory was created
        assert self.skill_hub_dir.exists(), f"~/.skill-hub directory not created at {self.skill_hub_dir}"
        assert self.skill_hub_dir.is_dir(), f"~/.skill-hub is not a directory"
        
        # Verify repo directory was created
        assert self.repo_dir.exists(), f"Repo directory not created at {self.repo_dir}"
        assert self.repo_dir.is_dir(), f"Repo is not a directory"
        
        # Verify skills directory exists (empty at this point)
        assert self.skills_dir.exists(), f"Skills directory not created at {self.skills_dir}"
        assert self.skills_dir.is_dir(), f"Skills is not a directory"
        
        # Check that skills directory is empty initially
        skills_list = list(self.skills_dir.iterdir())
        assert len(skills_list) == 0, f"Skills directory should be empty, found: {skills_list}"
        
        # Check global configuration
        config_file = self.skill_hub_dir / "config.json"
        if config_file.exists():
            with open(config_file, 'r') as f:
                config = json.load(f)
            # Default target should be 'all' as per documentation
            assert config.get('default_target') == 'all', f"Default target should be 'all', got: {config.get('default_target')}"
        
        print(f"✓ Environment initialized successfully")
        print(f"  - Created: {self.skill_hub_dir}")
        print(f"  - Repo: {self.repo_dir}")
        print(f"  - Skills directory: {self.skills_dir}")
        
    def test_02_skill_creation(self):
        """Test 1.2: Create a new skill"""
        print("\n=== Test 1.2: Skill Creation ===")
        
        # First initialize
        result = self.cmd.run("init")
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        # Create a new skill
        skill_name = "my-logic-skill"
        result = self.cmd.run("create", {skill_name})
        assert result.success, f"skill-hub create failed: {result.stderr}"
        
        # Verify skill directory was created in repo
        skill_dir = self.skills_dir / skill_name
        assert skill_dir.exists(), f"Skill directory not created at {skill_dir}"
        assert skill_dir.is_dir(), f"Skill directory is not a directory"
        
        # Verify skill.yaml was created
        skill_yaml = skill_dir / "skill.yaml"
        assert skill_yaml.exists(), f"skill.yaml not created at {skill_yaml}"
        assert skill_yaml.is_file(), f"skill.yaml is not a file"
        
        # Verify prompt.md was created
        prompt_md = skill_dir / "prompt.md"
        assert prompt_md.exists(), f"prompt.md not created at {prompt_md}"
        assert prompt_md.is_file(), f"prompt.md is not a file"
        
        # Verify skill.yaml has basic structure
        with open(skill_yaml, 'r') as f:
            yaml_content = f.read()
        assert "name:" in yaml_content, "skill.yaml missing name field"
        assert "description:" in yaml_content, "skill.yaml missing description field"
        assert "version:" in yaml_content, "skill.yaml missing version field"
        
        # Verify prompt.md has template content
        with open(prompt_md, 'r') as f:
            prompt_content = f.read()
        assert len(prompt_content.strip()) > 0, "prompt.md is empty"
        assert "# " in prompt_content, "prompt.md missing header"
        
        print(f"✓ Skill '{skill_name}' created successfully")
        print(f"  - Created: {skill_dir}")
        print(f"  - Files: skill.yaml, prompt.md")
        print(f"  - skill.yaml size: {len(yaml_content)} chars")
        print(f"  - prompt.md size: {len(prompt_content)} chars")
        
    def test_03_edit_and_feedback(self):
        """Test 1.3: Edit skill and provide feedback"""
        print("\n=== Test 1.3: Edit and Feedback ===")
        
        # First initialize and create skill
        result = self.cmd.run("init")
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        skill_name = "my-logic-skill"
        result = self.cmd.run("create", {skill_name})
        assert result.success, f"skill-hub create failed: {result.stderr}"
        
        # Get the prompt.md file path
        prompt_md = self.skills_dir / skill_name / "prompt.md"
        
        # Read original content
        with open(prompt_md, 'r') as f:
            original_content = f.read()
        
        # Modify the prompt.md content
        modified_content = original_content + "\n\n## Test Modification\nThis is a test modification added during the edit phase."
        
        # Write modified content
        with open(prompt_md, 'w') as f:
            f.write(modified_content)
        
        # Verify the modification was written
        with open(prompt_md, 'r') as f:
            current_content = f.read()
        assert "Test Modification" in current_content, "Modification not written to prompt.md"
        
        # Run skill-hub feedback
        result = self.cmd.run("feedback", {skill_name})
        assert result.success, f"skill-hub feedback failed: {result.stderr}"
        
        # Check that skill is now enabled in the project
        # First, we need to check the project state
        project_state_file = Path.cwd() / ".skill-hub" / "state.json"
        
        # Note: This test assumes we're running in a test project directory
        # In actual tests, we would use a temporary project directory
        print(f"  Note: Project state check would verify skill is 'Enabled'")
        print(f"  Note: skill-hub list would show the skill exists")
        
        # Verify the modified content is still in the file
        with open(prompt_md, 'r') as f:
            final_content = f.read()
        assert "Test Modification" in final_content, "Modification lost after feedback"
        
        print(f"✓ Skill edited and feedback provided successfully")
        print(f"  - Modified: {prompt_md}")
        print(f"  - Added test section")
        print(f"  - Ran skill-hub feedback")
        
    def test_04_skill_listing(self):
        """Test 1.4: List skills after creation"""
        print("\n=== Test 1.4: Skill Listing ===")
        
        # First initialize and create skill
        result = self.cmd.run("init")
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        skill_name = "my-logic-skill"
        result = self.cmd.run("create", {skill_name})
        assert result.success, f"skill-hub create failed: {result.stderr}"
        
        # Run skill-hub list
        result = self.cmd.run("list")
        assert result.success, f"skill-hub list failed: {result.stderr}"
        
        # Check that the skill appears in the list
        output = result.stdout.lower()
        assert skill_name.lower() in output, f"Skill '{skill_name}' not found in list output"
        
        print(f"✓ Skill listing works correctly")
        print(f"  - Found '{skill_name}' in skill list")
        print(f"  - Output length: {len(output)} chars")
        
    def test_05_full_workflow_integration(self):
        """Test 1.5: Full workflow integration test"""
        print("\n=== Test 1.5: Full Workflow Integration ===")
        
        # Track all steps
        steps_passed = []
        
        try:
            # Step 1: Initialize
            result = self.cmd.run("init")
            assert result.success
            steps_passed.append("init")
            print(f"  ✓ Step 1: Initialized skill-hub")
            
            # Step 2: Create skill
            skill_name = "my-logic-skill"
            result = self.cmd.run("create", {skill_name})
            assert result.success
            steps_passed.append("create")
            print(f"  ✓ Step 2: Created skill '{skill_name}'")
            
            # Step 3: Verify skill files
            skill_dir = self.skills_dir / skill_name
            assert skill_dir.exists()
            assert (skill_dir / "skill.yaml").exists()
            assert (skill_dir / "prompt.md").exists()
            steps_passed.append("verify_files")
            print(f"  ✓ Step 3: Verified skill files exist")
            
            # Step 4: Edit prompt
            prompt_file = skill_dir / "prompt.md"
            with open(prompt_file, 'a') as f:
                f.write("\n\n## Integration Test Edit\nAdded during integration test.")
            steps_passed.append("edit")
            print(f"  ✓ Step 4: Edited prompt.md")
            
            # Step 5: Provide feedback
            result = self.cmd.run("feedback", {skill_name})
            assert result.success
            steps_passed.append("feedback")
            print(f"  ✓ Step 5: Provided feedback")
            
            # Step 6: List skills
            result = self.cmd.run("list")
            assert result.success
            assert skill_name.lower() in result.stdout.lower()
            steps_passed.append("list")
            print(f"  ✓ Step 6: Listed skills (found '{skill_name}')")
            
            # All steps passed
            assert len(steps_passed) == 6
            print(f"\n✓ All {len(steps_passed)} steps passed successfully!")
            
        except AssertionError as e:
            print(f"\n✗ Workflow failed at step: {steps_passed[-1] if steps_passed else 'unknown'}")
            print(f"  Error: {e}")
            raise
    
    @pytest.mark.skipif(not NetworkChecker.is_network_available(), reason="Network required for this test")
    def test_06_network_operations(self):
        """Test 1.6: Network operations (if applicable)"""
        print("\n=== Test 1.6: Network Operations ===")
        
        # This test would check network-dependent operations
        # For now, just verify network is available
        assert NetworkChecker.is_network_available(), "Network should be available for this test"
        
        print(f"✓ Network operations test placeholder")
        print(f"  - Network is available")
        print(f"  - Future: Would test git operations, remote updates, etc.")


if __name__ == "__main__":
    # For direct execution
    pytest.main([__file__, "-v"])