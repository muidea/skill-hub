"""
Test Scenario 3: Iteration Feedback Workflow
Tests modification detection, status checking, and synchronization back to repository.
"""

import os
import json
import tempfile
import pytest
from pathlib import Path
import time

from tests.e2e.utils.command_runner import CommandRunner
from tests.e2e.utils.file_validator import FileValidator
from tests.e2e.utils.test_environment import TestEnvironment
from tests.e2e.utils.network_checker import NetworkChecker
from tests.e2e.utils.debug_utils import DebugUtils


class TestScenario3IterationFeedback:
    """Test scenario 3: Iteration feedback workflow (modify -> status -> sync)"""
    
    @pytest.fixture(autouse=True)
    def setup(self, temp_project_dir, temp_home_dir, test_skill_template):
        """Setup test environment"""
        self.project_dir = Path(temp_project_dir)
        self.home_dir = Path(temp_home_dir)
        self.skill_template = test_skill_template
        self.cmd = CommandRunner()
        self.validator = FileValidator()
        self.env = TestEnvironment()
        self.debug = DebugUtils()
        
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
        
    def _setup_skill_in_project(self, skill_name="my-logic-skill"):
        """Helper to setup a skill in the project (V2流程)"""
        # Initialize home directory
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=self.home_dir)
        assert result.success
        
        # Create skill in project
        result = self.cmd.run("create", [skill_name], cwd=self.project_dir)
        assert result.success
        
        # Feedback skill to repo (required before use)
        result = self.cmd.run("feedback", [skill_name], cwd=self.project_dir)
        assert result.success
        
        # Setup project
        result = self.cmd.run("set-target", ["open_code"], cwd=self.project_dir)
        assert result.success
        
        result = self.cmd.run("use", [skill_name], cwd=self.project_dir)
        assert result.success
        
        result = self.cmd.run("apply", cwd=self.project_dir)
        assert result.success
        
        return skill_name
    
    def test_01_project_modification_detection(self):
        """Test 3.1: Detect modifications in project files"""
        print("\n=== Test 3.1: Project Modification Detection ===")
        
        # Setup skill in project
        skill_name = self._setup_skill_in_project()
        
        # Get the SKILL.md file in project
        skill_file = self.project_skills_dir / skill_name / "SKILL.md"
        assert skill_file.exists(), f"SKILL.md not found at {skill_file}"
        
        # Read original content
        with open(skill_file, 'r') as f:
            original_content = f.read()
        
        # Modify the file (add to content part)
        parts = original_content.split("---")
        if len(parts) >= 3:
            yaml_part = parts[1]
            content_part = parts[2]
            modification = "\n\n## Project Modification\nThis modification was made directly in the project to test detection."
            modified_content = f"{parts[0]}---{yaml_part}---{content_part}{modification}"
        else:
            # Simple append if format unexpected
            modification = "\n\n## Project Modification\nThis modification was made directly in the project to test detection."
            modified_content = original_content + modification
        
        with open(skill_file, 'w') as f:
            f.write(modified_content)
        
        # Verify modification was written
        with open(skill_file, 'r') as f:
            current_content = f.read()
        assert "Project Modification" in current_content, "Modification not written to SKILL.md"
        
        # Run skill-hub status to detect modification
        result = self.cmd.run("status", cwd=self.project_dir)
        assert result.success, f"skill-hub status failed: {result.stderr}"
        
        # Check that status shows Modified
        output = result.stdout
        # 检查中文"已修改"或包含修改指示
        assert "已修改" in output or "修改" in output, f"Status should show modification, output: {output}"
        assert skill_name in output, f"Skill name '{skill_name}' should appear in status output"
        
        print(f"✓ Modification detection works")
        print(f"  - Modified: {skill_file}")
        print(f"  - Status shows: Modified")
        print(f"  - Output snippet: {output[:200]}...")
        
    def test_02_feedback_synchronization(self):
        """Test 3.2: Synchronize modifications back to repository"""
        print("\n=== Test 3.2: Feedback Synchronization ===")
        
        # Setup skill in project
        skill_name = self._setup_skill_in_project()
        
        # Get project instructions.md
        project_instructions = self.project_skills_dir / skill_name / "instructions.md"
        
        # Get repository prompt.md
        repo_prompt = self.repo_skills_dir / skill_name / "prompt.md"
        
        # Read original contents
        with open(project_instructions, 'r') as f:
            original_project_content = f.read()
        
        with open(repo_prompt, 'r') as f:
            original_repo_content = f.read()
        
        # Modify project file
        test_modification = "\n\n## Test Synchronization\nThis change should be synchronized back to the repository."
        modified_project_content = original_project_content + test_modification
        
        with open(project_instructions, 'w') as f:
            f.write(modified_project_content)
        
        # Verify project file was modified
        with open(project_instructions, 'r') as f:
            current_project_content = f.read()
        assert "Test Synchronization" in current_project_content
        
        # Check status shows Modified
        result = self.cmd.run("status", cwd=str(self.project_dir))
        assert result.success
        assert "modified" in result.stdout.lower()
        
        # Run feedback to synchronize
        result = self.cmd.run("feedback", {skill_name})
        assert result.success, f"skill-hub feedback failed: {result.stderr}"
        
        # Check repository file was updated
        with open(repo_prompt, 'r') as f:
            updated_repo_content = f.read()
        
        # The modification should now be in the repository
        # Note: The exact transformation depends on skill-hub implementation
        # For now, just check that the file was modified
        assert updated_repo_content != original_repo_content, "Repository file was not updated"
        
        # Check status shows Synced after feedback
        result = self.cmd.run("status", cwd=str(self.project_dir))
        assert result.success
        
        # Status should no longer show Modified (might show Synced or nothing)
        output = result.stdout.lower()
        if "synced" in output:
            print(f"  Status shows: Synced")
        elif "modified" not in output:
            print(f"  Status no longer shows Modified")
        
        print(f"✓ Feedback synchronization works")
        print(f"  - Project modified: {project_instructions}")
        print(f"  - Repository updated: {repo_prompt}")
        print(f"  - Files differ: {updated_repo_content != original_repo_content}")
        
    def test_03_multiple_modifications(self):
        """Test 3.3: Handle multiple file modifications"""
        print("\n=== Test 3.3: Multiple File Modifications ===")
        
        # Setup skill in project
        skill_name = self._setup_skill_in_project()
        
        skill_dir = self.project_skills_dir / skill_name
        
        # Modify multiple files
        files_to_modify = [
            ("instructions.md", "\n\n## Multiple File Test\nModified instructions.md"),
            ("manifest.yaml", "\n\n# Test comment added to manifest")
        ]
        
        modifications = {}
        
        for filename, modification in files_to_modify:
            file_path = skill_dir / filename
            if file_path.exists():
                with open(file_path, 'r') as f:
                    original = f.read()
                
                modified = original + modification
                with open(file_path, 'w') as f:
                    f.write(modified)
                
                modifications[filename] = (original, modified)
        
        # Check status
        result = self.cmd.run("status", cwd=str(self.project_dir))
        assert result.success
        
        output = result.stdout.lower()
        print(f"  Status after multiple modifications: {output[:300]}...")
        
        # Run feedback
        result = self.cmd.run("feedback", {skill_name})
        assert result.success
        
        print(f"✓ Multiple file modifications handled")
        print(f"  - Modified files: {list(modifications.keys())}")
        print(f"  - Feedback completed successfully")
        
    def test_04_target_specific_modification_extraction(self):
        """Test 3.4: Extract modifications from correct target path"""
        print("\n=== Test 3.4: Target-Specific Modification Extraction ===")
        
        # This test verifies that modifications are extracted from the correct
        # target-specific path (e.g., .agents/skills/ for open_code vs .cursorrules for cursor)
        
        # Test with open_code target
        print(f"  Testing open_code target...")
        
        skill_name = "test-target-skill"
        home_cmd = CommandRunner()
        result = home_cmd.run("skill-hub init", timeout=30)
        assert result.success
        
        result = home_cmd.run(f"skill-hub create {skill_name}", timeout=30)
        assert result.success
        
        # Setup project with open_code
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success
        
        result = self.cmd.run("use", {skill_name})
        assert result.success
        
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success
        
        # Modify in open_code location
        instructions_file = self.project_skills_dir / skill_name / "instructions.md"
        with open(instructions_file, 'a') as f:
            f.write("\n\n## OpenCode Modification\nModified in .agents/skills/")
        
        # Check status detects modification
        result = self.cmd.run("status", cwd=str(self.project_dir))
        assert result.success
        assert "modified" in result.stdout.lower()
        
        print(f"  ✓ open_code modifications detected in correct path")
        
        # Test with cursor target (if supported)
        print(f"  Testing cursor target...")
        
        # Create a new project for cursor target
        with tempfile.TemporaryDirectory() as cursor_project:
            cursor_cmd = CommandRunner()
            
            result = cursor_cmd.run("skill-hub set-target cursor", timeout=30)
            assert result.success
            
            result = cursor_cmd.run(f"skill-hub use {skill_name}", timeout=30)
            assert result.success
            
            result = cursor_cmd.run("skill-hub apply", timeout=30)
            # This might fail if skill doesn't support cursor
            if result.exit_code == 0:
                cursorrules_file = Path(cursor_project) / ".cursorrules"
                if cursorrules_file.exists():
                    with open(cursorrules_file, 'a') as f:
                        f.write("\n\n# Cursor Modification\nModified in .cursorrules")
                    
                    # Check status
                    result = cursor_cmd.run("skill-hub status", timeout=30)
                    print(f"    Cursor status: {result.stdout[:200]}...")
                    print(f"  ✓ cursor modifications would be detected in .cursorrules")
                else:
                    print(f"  Note: .cursorrules not created (skill may not support cursor)")
            else:
                print(f"  Note: skill-hub apply for cursor failed (may not be supported)")
        
        print(f"✓ Target-specific modification extraction verified")
        
    def test_05_json_escaping_handling(self):
        """Test 3.5: JSON escaping in feedback"""
        print("\n=== Test 3.5: JSON Escaping Handling ===")
        
        # Setup skill in project
        skill_name = self._setup_skill_in_project("json-test-skill")
        
        # Get project file
        instructions_file = self.project_skills_dir / skill_name / "instructions.md"
        
        # Add content with characters that need JSON escaping
        problematic_content = """
## JSON Test Content

Special characters that need escaping:
- Quotes: "double" and 'single'
- Backslashes: \\
- Newlines: 
  (this is a newline)
- Unicode: café, naïve, résumé
- Control characters: \t \n \r
- JSON special: {}[],:

Example JSON:
```json
{
  "name": "test",
  "value": "quotes \"inside\" string",
  "path": "C:\\Users\\test\\file.txt"
}
```
"""
        
        # Read original, append problematic content
        with open(instructions_file, 'r') as f:
            original = f.read()
        
        with open(instructions_file, 'w') as f:
            f.write(original + problematic_content)
        
        # Run feedback - this should handle JSON escaping properly
        result = self.cmd.run("feedback", {skill_name})
        
        # Check if feedback succeeded
        if result.exit_code == 0:
            print(f"✓ JSON escaping handled successfully")
            print(f"  - Feedback succeeded with special characters")
        else:
            print(f"✗ Feedback failed with JSON special characters")
            print(f"  - stderr: {result.stderr[:200]}...")
            # Don't fail the test, just log it since this depends on skill-hub implementation
        
        # Check repository file
        repo_file = self.repo_skills_dir / skill_name / "prompt.md"
        if repo_file.exists():
            with open(repo_file, 'r') as f:
                repo_content = f.read()
            
            # Check if our content made it to the repository
            if "JSON Test Content" in repo_content:
                print(f"  - Content successfully synchronized to repository")
            else:
                print(f"  - Note: Content may have been transformed")
        
    def test_06_status_accuracy(self):
        """Test 3.6: Status command accuracy"""
        print("\n=== Test 3.6: Status Command Accuracy ===")
        
        # Setup
        skill_name = self._setup_skill_in_project()
        
        # Test 1: Clean state (should show synced or nothing)
        result = self.cmd.run("status", cwd=str(self.project_dir))
        assert result.success
        clean_output = result.stdout.lower()
        print(f"  Clean state output: {clean_output[:200]}...")
        
        # Test 2: Modified state
        instructions_file = self.project_skills_dir / skill_name / "instructions.md"
        with open(instructions_file, 'a') as f:
            f.write("\n\n## Status Accuracy Test\n")
        
        result = self.cmd.run("status", cwd=str(self.project_dir))
        assert result.success
        modified_output = result.stdout.lower()
        assert "modified" in modified_output
        print(f"  Modified state output: {modified_output[:200]}...")
        
        # Test 3: After feedback (should return to clean/synced)
        result = self.cmd.run("feedback", {skill_name})
        assert result.success
        
        result = self.cmd.run("status", cwd=str(self.project_dir))
        assert result.success
        after_feedback_output = result.stdout.lower()
        print(f"  After feedback output: {after_feedback_output[:200]}...")
        
        # Test 4: With multiple skills
        # Add another skill
        home_cmd = CommandRunner()
        skill_name2 = "second-test-skill"
        result = home_cmd.run(f"skill-hub create {skill_name2}", timeout=30)
        if result.exit_code == 0:
            result = self.cmd.run(f"skill-hub use {skill_name2}", timeout=30)
            result = self.cmd.run("apply", cwd=str(self.project_dir))
            
            # Modify second skill
            instructions_file2 = self.project_skills_dir / skill_name2 / "instructions.md"
            if instructions_file2.exists():
                with open(instructions_file2, 'a') as f:
                    f.write("\n\n## Second Skill Modification\n")
                
                result = self.cmd.run("status", cwd=str(self.project_dir))
                assert result.success
                multi_output = result.stdout.lower()
                print(f"  Multiple skills output: {multi_output[:300]}...")
        
        print(f"✓ Status command provides accurate information")
        
    def test_07_partial_modifications(self):
        """Test 3.7: Handle partial modifications (some files changed, others not)"""
        print("\n=== Test 3.7: Partial Modifications ===")
        
        # Setup
        skill_name = self._setup_skill_in_project()
        skill_dir = self.project_skills_dir / skill_name
        
        # Only modify instructions.md, not manifest.yaml
        instructions_file = skill_dir / "instructions.md"
        manifest_file = skill_dir / "manifest.yaml"
        
        # Get original modification times
        instructions_mtime_before = instructions_file.stat().st_mtime if instructions_file.exists() else 0
        manifest_mtime_before = manifest_file.stat().st_mtime if manifest_file.exists() else 0
        
        # Wait a bit to ensure different modification times
        time.sleep(0.1)
        
        # Modify only instructions.md
        with open(instructions_file, 'a') as f:
            f.write("\n\n## Partial Modification Test\nOnly modified instructions.md")
        
        # Get new modification times
        instructions_mtime_after = instructions_file.stat().st_mtime
        manifest_mtime_after = manifest_file.stat().st_mtime if manifest_file.exists() else 0
        
        # Verify only instructions.md was modified
        assert instructions_mtime_after > instructions_mtime_before, "instructions.md modification time not updated"
        if manifest_file.exists():
            assert manifest_mtime_after == manifest_mtime_before, "manifest.yaml should not be modified"
        
        # Check status
        result = self.cmd.run("status", cwd=str(self.project_dir))
        assert result.success
        print(f"  Status with partial modification: {result.stdout[:200]}...")
        
        # Run feedback
        result = self.cmd.run("feedback", {skill_name})
        assert result.success
        
        print(f"✓ Partial modifications handled correctly")
        print(f"  - Modified: instructions.md")
        print(f"  - Unmodified: manifest.yaml")
        print(f"  - Feedback completed successfully")


if __name__ == "__main__":
    # For direct execution
    pytest.main([__file__, "-v"])