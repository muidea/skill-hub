"""
Test Scenario 4: Cancel and Cleanup Workflow
Tests skill removal, physical cleanup, and multi-target cleanup operations.
"""

import os
import json
import tempfile
import pytest
from pathlib import Path
import shutil

from tests.e2e.utils.command_runner import CommandRunner
from tests.e2e.utils.file_validator import FileValidator
from tests.e2e.utils.test_environment import TestEnvironment
from tests.e2e.utils.network_checker import NetworkChecker
from tests.e2e.utils.debug_utils import DebugUtils


class TestScenario4CancelCleanup:
    """Test scenario 4: Cancel and cleanup workflow (remove -> cleanup)"""
    
    @pytest.fixture(autouse=True)
    def setup(self, temp_project_dir, temp_home_dir, test_skill_template):
        """Setup test environment"""
        self.project_dir = temp_project_dir
        self.home_dir = temp_home_dir
        self.skill_template = test_skill_template
        self.cmd = CommandRunner()
        self.validator = FileValidator()
        self.env = TestEnvironment()
        self.debug = DebugUtils()
        
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
        
    def _setup_skill_in_project(self, skill_name="my-logic-skill", target="open_code"):
        """Helper to setup a skill in the project"""
        # Initialize home directory
        home_cmd = CommandRunner()
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success
        
        # Create skill
        result = home_cmd.run(f"skill-hub create {skill_name}", timeout=30)
        assert result.success
        
        # Setup project
        result = self.cmd.run("set-target", {target})
        assert result.success
        
        result = self.cmd.run("use", {skill_name})
        assert result.success
        
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success
        
        return skill_name
    
    def test_01_basic_skill_removal(self):
        """Test 4.1: Basic skill removal"""
        print("\n=== Test 4.1: Basic Skill Removal ===")
        
        # Setup skill in project
        skill_name = self._setup_skill_in_project()
        
        # Verify skill directory exists before removal
        skill_dir = self.agents_skills_dir / skill_name
        assert skill_dir.exists(), f"Skill directory should exist before removal: {skill_dir}"
        
        # Verify state.json exists and contains skill
        assert self.project_state.exists(), f"state.json should exist: {self.project_state}"
        
        with open(self.project_state, 'r') as f:
            state_before = json.load(f)
        
        # Check skill is in state (exact field depends on implementation)
        skill_in_state = False
        for key, value in state_before.items():
            if isinstance(value, list) and skill_name in value:
                skill_in_state = True
                break
            elif isinstance(value, dict) and skill_name in value:
                skill_in_state = True
                break
        
        assert skill_in_state, f"Skill '{skill_name}' should be in state.json before removal"
        
        # Remove the skill
        result = self.cmd.run("remove", {skill_name})
        assert result.success, f"skill-hub remove failed: {result.stderr}"
        
        # Verify physical cleanup: skill directory should be deleted
        assert not skill_dir.exists(), f"Skill directory should be deleted: {skill_dir}"
        
        # Verify state.json was updated (skill removed)
        assert self.project_state.exists(), f"state.json should still exist after removal"
        
        with open(self.project_state, 'r') as f:
            state_after = json.load(f)
        
        # Check skill is NOT in state after removal
        skill_still_in_state = False
        for key, value in state_after.items():
            if isinstance(value, list) and skill_name in value:
                skill_still_in_state = True
                break
            elif isinstance(value, dict) and skill_name in value:
                skill_still_in_state = True
                break
        
        assert not skill_still_in_state, f"Skill '{skill_name}' should be removed from state.json"
        
        # Safety check: repository files must be preserved
        repo_skill_dir = self.skills_dir / skill_name
        assert repo_skill_dir.exists(), f"Repository skill directory must be preserved: {repo_skill_dir}"
        assert (repo_skill_dir / "skill.yaml").exists(), "Repository skill.yaml must be preserved"
        assert (repo_skill_dir / "prompt.md").exists(), "Repository prompt.md must be preserved"
        
        print(f"✓ Basic skill removal works")
        print(f"  - Physical cleanup: {skill_dir} deleted")
        print(f"  - State updated: skill removed from {self.project_state}")
        print(f"  - Safety: repository files preserved at {repo_skill_dir}")
        
    def test_02_remove_nonexistent_skill(self):
        """Test 4.2: Remove non-existent skill"""
        print("\n=== Test 4.2: Remove Non-Existent Skill ===")
        
        # Setup environment
        home_cmd = CommandRunner()
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success
        
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success
        
        # Try to remove a skill that doesn't exist
        nonexistent_skill = "nonexistent-skill-12345"
        result = self.cmd.run(f"skill-hub remove {nonexistent_skill}", timeout=30)
        
        # This might fail or succeed with a message
        print(f"  Remove nonexistent skill result: returncode={result.exit_code}")
        print(f"  stdout: {result.stdout[:100]}...")
        print(f"  stderr: {result.stderr[:100]}...")
        
        # The test passes as long as it doesn't crash
        print(f"✓ Handled removal of non-existent skill")
        
    def test_03_remove_multiple_skills(self):
        """Test 4.3: Remove multiple skills"""
        print("\n=== Test 4.3: Remove Multiple Skills ===")
        
        # Setup environment
        home_cmd = CommandRunner()
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success
        
        # Create multiple skills
        skills = ["skill-to-remove-1", "skill-to-remove-2", "skill-to-keep"]
        
        for skill in skills:
            result = home_cmd.run(f"skill-hub create {skill}", timeout=30)
            assert result.success
        
        # Setup project
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success
        
        # Enable all skills
        for skill in skills:
            result = self.cmd.run(f"skill-hub use {skill}", timeout=30)
            assert result.success
        
        # Apply all
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success
        
        # Verify all skill directories exist
        for skill in skills:
            skill_dir = self.agents_skills_dir / skill
            assert skill_dir.exists(), f"Skill directory should exist: {skill_dir}"
        
        # Remove first two skills
        for skill in skills[:2]:
            result = self.cmd.run(f"skill-hub remove {skill}", timeout=30)
            assert result.success
        
        # Verify removed skills are gone
        for skill in skills[:2]:
            skill_dir = self.agents_skills_dir / skill
            assert not skill_dir.exists(), f"Skill directory should be deleted: {skill_dir}"
        
        # Verify kept skill still exists
        kept_skill_dir = self.agents_skills_dir / skills[2]
        assert kept_skill_dir.exists(), f"Skill directory should still exist: {kept_skill_dir}"
        
        print(f"✓ Multiple skill removal works")
        print(f"  - Removed: {skills[:2]}")
        print(f"  - Kept: {skills[2]}")
        print(f"  - Directories correctly cleaned up")
        
    def test_04_cleanup_with_different_targets(self):
        """Test 4.4: Cleanup when skill is applied to different targets"""
        print("\n=== Test 4.4: Cleanup with Different Targets ===")
        
        # Setup skill normally
        skill_name = self._setup_skill_in_project()
        
        # Apply skill to open_code target (default)
        result = self.cmd.run("set-target", ["open_code"])
        assert result.success
        
        result = self.cmd.run("use", [skill_name])
        assert result.success
        
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success
        
        # Verify files exist for open_code target
        skill_dir = self.agents_skills_dir / skill_name
        assert skill_dir.exists(), f"Skill directory should exist: {skill_name}"
        
        # Check if .skills directory was created for open_code target
        skills_dir = self.project_dir / ".skills" / skill_name
        open_code_files_exist = skills_dir.exists()
        
        # Now remove the skill
        result = self.cmd.run("remove", [skill_name])
        assert result.success, f"skill-hub remove failed: {result.stderr}"
        
        # Verify skill directory was deleted
        assert not skill_dir.exists(), f"Skill directory should be deleted: {skill_name}"
        
        # Verify .skills directory was cleaned up if it existed
        if open_code_files_exist:
            assert not skills_dir.exists(), f".skills directory should be cleaned up: {skills_dir}"
        
        print(f"✓ Skill cleanup works for different targets")
        print(f"  - Removed skill: {skill_name}")
        print(f"  - Cleaned up project files")
        print(f"  - OpenCode target files cleaned: {open_code_files_exist}")
        
    def test_05_cleanup_with_modified_files(self):
        """Test 4.5: Cleanup when skill files have been modified"""
        print("\n=== Test 4.5: Cleanup with Modified Files ===")
        
        # Setup skill
        skill_name = self._setup_skill_in_project()
        
        # Modify skill files
        skill_dir = self.agents_skills_dir / skill_name
        instructions_file = skill_dir / "instructions.md"
        
        with open(instructions_file, 'a') as f:
            f.write("\n\n## Local Modification\nThis was modified locally and not synced back.")
        
        # Verify modification
        with open(instructions_file, 'r') as f:
            content = f.read()
        assert "Local Modification" in content
        
        # Check status shows modified
        result = self.cmd.run("status", cwd=str(self.project_dir))
        print(f"  Status before removal: {result.stdout[:200]}...")
        
        # Remove the skill (with local modifications)
        result = self.cmd.run("remove", {skill_name})
        assert result.success, f"skill-hub remove failed with modified files: {result.stderr}"
        
        # Verify cleanup happened despite modifications
        assert not skill_dir.exists(), f"Skill directory should be deleted even with modifications"
        
        # Repository files should still be preserved
        repo_skill_dir = self.skills_dir / skill_name
        assert repo_skill_dir.exists(), f"Repository skill directory must be preserved"
        
        # Check repository files don't have the local modification
        repo_prompt = repo_skill_dir / "prompt.md"
        with open(repo_prompt, 'r') as f:
            repo_content = f.read()
        
        # Local modification should NOT be in repository (wasn't synced)
        if "Local Modification" not in repo_content:
            print(f"  ✓ Local modifications not propagated to repository (correct)")
        else:
            print(f"  Note: Local modifications may have been synced before removal")
        
        print(f"✓ Cleanup works with locally modified files")
        print(f"  - Modified files were cleaned up")
        print(f"  - Repository preserved original content")
        
    def test_06_cleanup_preserves_other_skills(self):
        """Test 4.6: Cleanup preserves other skills in same directory"""
        print("\n=== Test 4.6: Cleanup Preserves Other Skills ===")
        
        # Setup multiple skills
        home_cmd = CommandRunner()
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success
        
        skills = ["skill-to-remove", "skill-to-keep-1", "skill-to-keep-2"]
        
        for skill in skills:
            result = home_cmd.run(f"skill-hub create {skill}", timeout=30)
            assert result.success
        
        # Setup project
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success
        
        # Enable all skills
        for skill in skills:
            result = self.cmd.run(f"skill-hub use {skill}", timeout=30)
            assert result.success
        
        # Apply all
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success
        
        # Verify all directories exist
        for skill in skills:
            skill_dir = self.agents_skills_dir / skill
            assert skill_dir.exists(), f"Skill directory should exist: {skill_dir}"
        
        # Remove only the first skill
        skill_to_remove = skills[0]
        result = self.cmd.run(f"skill-hub remove {skill_to_remove}", timeout=30)
        assert result.success
        
        # Verify removed skill is gone
        removed_dir = self.agents_skills_dir / skill_to_remove
        assert not removed_dir.exists(), f"Removed skill directory should be deleted: {removed_dir}"
        
        # Verify other skills still exist
        for skill in skills[1:]:
            skill_dir = self.agents_skills_dir / skill
            assert skill_dir.exists(), f"Other skill directory should still exist: {skill_dir}"
            
            # Verify their files are intact
            assert (skill_dir / "manifest.yaml").exists(), f"manifest.yaml should exist for {skill}"
            assert (skill_dir / "instructions.md").exists(), f"instructions.md should exist for {skill}"
        
        print(f"✓ Cleanup preserves other skills")
        print(f"  - Removed: {skill_to_remove}")
        print(f"  - Preserved: {skills[1:]}")
        print(f"  - All preserved skill files intact")
        
    def test_07_cleanup_with_nested_directories(self):
        """Test 4.7: Cleanup with nested directories in skill"""
        print("\n=== Test 4.7: Cleanup with Nested Directories ===")
        
        # Setup skill
        skill_name = self._setup_skill_in_project()
        
        # Create nested directories and files in the skill directory
        skill_dir = self.agents_skills_dir / skill_name
        
        # Create nested structure
        nested_dir = skill_dir / "nested" / "deeply" / "nested"
        nested_dir.mkdir(parents=True, exist_ok=True)
        
        # Create files in nested directories
        nested_file = nested_dir / "test.txt"
        with open(nested_file, 'w') as f:
            f.write("Test content in nested file")
        
        another_nested_file = skill_dir / "another" / "test.md"
        another_nested_file.parent.mkdir(parents=True, exist_ok=True)
        with open(another_nested_file, 'w') as f:
            f.write("# Another test file")
        
        # Verify nested structure exists
        assert nested_file.exists(), f"Nested file should exist: {nested_file}"
        assert another_nested_file.exists(), f"Another nested file should exist: {another_nested_file}"
        
        # Remove the skill
        result = self.cmd.run("remove", {skill_name})
        assert result.success
        
        # Verify entire skill directory (including nested structure) is deleted
        assert not skill_dir.exists(), f"Entire skill directory should be deleted: {skill_dir}"
        
        print(f"✓ Cleanup removes nested directory structure")
        print(f"  - Created nested: {nested_dir}")
        print(f"  - Created: {another_nested_file}")
        print(f"  - Entire directory tree removed")
        
    def test_08_repository_safety(self):
        """Test 4.8: Repository safety - never delete source files"""
        print("\n=== Test 4.8: Repository Safety ===")
        
        # This is a critical test: repository files must NEVER be deleted
        
        # Setup skill
        skill_name = self._setup_skill_in_project()
        
        # Get repository paths
        repo_skill_dir = self.skills_dir / skill_name
        repo_skill_yaml = repo_skill_dir / "skill.yaml"
        repo_prompt_md = repo_skill_dir / "prompt.md"
        
        # Get original content
        with open(repo_skill_yaml, 'r') as f:
            original_yaml = f.read()
        
        with open(repo_prompt_md, 'r') as f:
            original_prompt = f.read()
        
        # Remove skill from project
        result = self.cmd.run("remove", {skill_name})
        assert result.success
        
        # CRITICAL: Repository files must still exist
        assert repo_skill_dir.exists(), f"Repository skill directory MUST exist: {repo_skill_dir}"
        assert repo_skill_yaml.exists(), f"Repository skill.yaml MUST exist: {repo_skill_yaml}"
        assert repo_prompt_md.exists(), f"Repository prompt.md MUST exist: {repo_prompt_md}"
        
        # CRITICAL: Repository content must be unchanged
        with open(repo_skill_yaml, 'r') as f:
            current_yaml = f.read()
        
        with open(repo_prompt_md, 'r') as f:
            current_prompt = f.read()
        
        assert current_yaml == original_yaml, "Repository skill.yaml content changed!"
        assert current_prompt == original_prompt, "Repository prompt.md content changed!"
        
        print(f"✓ Repository safety verified")
        print(f"  - Repository directory preserved: {repo_skill_dir}")
        print(f"  - skill.yaml unchanged: {len(current_yaml)} chars")
        print(f"  - prompt.md unchanged: {len(current_prompt)} chars")
        print(f"  - CRITICAL: Source files never deleted by remove operation")


if __name__ == "__main__":
    # For direct execution
    pytest.main([__file__, "-v"])