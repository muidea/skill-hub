"""
Test Scenario: Feedback Command Version Auto-Upgrade
Tests that the feedback command automatically upgrades the patch version
when skill content is modified but version number is not updated by the user.
"""

import os
import re
import pytest
from pathlib import Path

from tests.e2e.utils.command_runner import CommandRunner
from tests.e2e.utils.file_validator import FileValidator
from tests.e2e.utils.test_environment import TestEnvironment
from tests.e2e.utils.debug_utils import DebugUtils


class TestFeedbackVersionAutoUpgrade:
    """Test feedback command version auto-upgrade functionality"""

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

        self.skill_hub_dir = self.home_dir / ".skill-hub"
        self.repositories_dir = self.skill_hub_dir / "repositories"
        self.main_repo_dir = self.repositories_dir / "main"
        self.repo_skills_dir = self.main_repo_dir / "skills"

        self.project_skill_hub = self.project_dir / ".skill-hub"
        self.project_agents_dir = self.project_dir / ".agents"
        self.project_agents_skills_dir = self.project_agents_dir / "skills"

        self.project_agents_dir.mkdir(exist_ok=True)

        os.chdir(self.project_dir)

    def _create_skill_with_version(self, skill_name: str, version: str, content: str = None) -> Path:
        """Create a skill with a specific version using skill-hub create"""
        # Ensure .agents/skills directory exists
        agents_skills_dir = self.project_dir / ".agents" / "skills"
        agents_skills_dir.mkdir(parents=True, exist_ok=True)

        # Use skill-hub create command
        result = self.cmd.run("create", [skill_name], cwd=str(self.project_dir))
        if not result.success:
            raise Exception(f"Failed to create skill: {result.stderr}")

        skill_dir = agents_skills_dir / skill_name
        skill_md = skill_dir / "SKILL.md"

        if content is None:
            content = f"""---
name: {skill_name}
description: Test skill for version auto-upgrade testing
metadata:
  version: {version}
---

# {skill_name}

This is a test skill for version auto-upgrade testing.

## Instructions

- Follow these instructions
- Version: {version}
"""
        skill_md.write_text(content)
        return skill_dir

    def _get_version_from_skill_md(self, skill_md_path: Path) -> str:
        """Extract version from SKILL.md file"""
        content = skill_md_path.read_text()

        # Try metadata.version first
        metadata_match = re.search(r'metadata:\s*\n\s*version:\s*["\']?([0-9.]+)["\']?', content)
        if metadata_match:
            return metadata_match.group(1)

        # Try root-level version
        root_version_match = re.search(r'^version:\s*["\']?([0-9.]+)["\']?', content, re.MULTILINE)
        if root_version_match:
            return root_version_match.group(1)

        return "1.0.0"

    def _modify_skill_content(self, skill_dir: Path, modification: str = "Added new content"):
        """Modify the skill content without changing version"""
        skill_md = skill_dir / "SKILL.md"
        content = skill_md.read_text()
        skill_md.write_text(content + f"\n\n## Modification\n\n{modification}\n")

    def test_01_auto_upgrade_patch_version(self):
        """Test 1: Version should be auto-upgraded when content changes without version update"""
        print("\n=== Test 1: Auto Upgrade Patch Version ===")

        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success, f"skill-hub init failed: {result.stderr}"

        # Create skill with version 1.0.0
        skill_name = "auto-upgrade-test"
        skill_dir = self._create_skill_with_version(skill_name, "1.0.0")

        # First feedback to repository
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"Initial feedback failed: {result.stderr}"

        # Modify skill content without changing version
        self._modify_skill_content(skill_dir, "First modification")

        # Second feedback - should auto-upgrade version
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"Second feedback failed: {result.stderr}"

        # Verify version was auto-upgraded
        project_skill_md = skill_dir / "SKILL.md"
        project_version = self._get_version_from_skill_md(project_skill_md)

        assert project_version == "1.0.1", f"Expected version 1.0.1, got {project_version}"

        # Verify repository version matches
        repo_skill_md = self.repo_skills_dir / skill_name / "SKILL.md"
        repo_version = self._get_version_from_skill_md(repo_skill_md)

        assert repo_version == "1.0.1", f"Expected repo version 1.0.1, got {repo_version}"

        print(f"✓ Version auto-upgraded from 1.0.0 to {project_version}")

    def test_02_multiple_auto_upgrades(self):
        """Test 2: Multiple modifications should result in multiple patch upgrades"""
        print("\n=== Test 2: Multiple Auto Upgrades ===")

        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success

        skill_name = "multi-upgrade-test"
        skill_dir = self._create_skill_with_version(skill_name, "1.0.0")

        # First feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success

        expected_versions = ["1.0.1", "1.0.2", "1.0.3"]

        for i, expected_version in enumerate(expected_versions):
            # Modify content
            self._modify_skill_content(skill_dir, f"Modification {i+1}")

            # Feedback
            result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
            assert result.success, f"Feedback {i+1} failed: {result.stderr}"

            # Check version
            project_version = self._get_version_from_skill_md(skill_dir / "SKILL.md")
            assert project_version == expected_version, f"Expected {expected_version}, got {project_version}"

            print(f"  Upgrade {i+1}: 1.0.0 -> {project_version}")

        print(f"✓ Multiple auto-upgrades working correctly")

    def test_03_user_specified_version_preserved(self):
        """Test 3: User-specified version should be preserved when it's higher than repo"""
        print("\n=== Test 3: User Specified Version Preserved ===")

        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success

        skill_name = "user-version-test"
        skill_dir = self._create_skill_with_version(skill_name, "1.0.0")

        # First feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success

        # Modify content AND update version manually to 2.0.0
        self._modify_skill_content(skill_dir, "Major update")

        # Update version manually
        skill_md = skill_dir / "SKILL.md"
        content = skill_md.read_text()
        content = content.replace("version: 1.0.0", "version: 2.0.0")
        skill_md.write_text(content)

        # Feedback - should use user-specified version
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"Feedback failed: {result.stderr}"

        # Verify user version was preserved
        project_version = self._get_version_from_skill_md(skill_md)
        assert project_version == "2.0.0", f"Expected user version 2.0.0, got {project_version}"

        print(f"✓ User-specified version 2.0.0 was preserved")

    def test_04_version_format_variations(self):
        """Test 4: Different version format variations"""
        print("\n=== Test 4: Version Format Variations ===")

        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success

        test_cases = [
            ("1.0.0", "1.0.1"),
            ("2.3.4", "2.3.5"),
            ("0.0.1", "0.0.2"),
            ("10.20.30", "10.20.31"),
        ]

        for original_version, expected_version in test_cases:
            skill_name = f"version-fmt-{original_version.replace('.', '-')}"

            # Use skill-hub create to register the skill
            result = self.cmd.run("create", [skill_name], cwd=str(self.project_dir))
            assert result.success, f"Create failed for {skill_name}: {result.stderr}"

            skill_dir = self.project_agents_skills_dir / skill_name
            skill_md = skill_dir / "SKILL.md"

            # Update the skill with the test version
            content = skill_md.read_text()
            content = re.sub(r'version:\s*["\']?[0-9.]+["\']?', f'version: {original_version}', content)
            skill_md.write_text(content)

            # First feedback
            result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
            assert result.success

            # Modify and feedback
            self._modify_skill_content(skill_dir, "Modified")

            result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
            assert result.success

            # Check version
            actual_version = self._get_version_from_skill_md(skill_md)
            assert actual_version == expected_version, f"Version {original_version}: expected {expected_version}, got {actual_version}"

            print(f"  {original_version} -> {actual_version} ✓")

        print(f"✓ All version format variations handled correctly")

    def test_05_new_skill_first_feedback(self):
        """Test 5: First feedback of a new skill should keep initial version"""
        print("\n=== Test 5: New Skill First Feedback ===")

        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success

        skill_name = "new-skill-test"
        skill_dir = self._create_skill_with_version(skill_name, "1.0.0")

        # First feedback - should keep version 1.0.0
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"First feedback failed: {result.stderr}"

        # Verify version is still 1.0.0 (no auto-upgrade for new skill)
        project_version = self._get_version_from_skill_md(skill_dir / "SKILL.md")
        assert project_version == "1.0.0", f"Expected version 1.0.0 for new skill, got {project_version}"

        print(f"✓ New skill keeps initial version 1.0.0")

    def test_06_version_upgrade_output_message(self):
        """Test 6: Verify version upgrade message is shown"""
        print("\n=== Test 6: Version Upgrade Output Message ===")

        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success

        skill_name = "message-test"
        skill_dir = self._create_skill_with_version(skill_name, "1.0.0")

        # First feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success

        # Modify and feedback
        self._modify_skill_content(skill_dir, "Test modification")

        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success

        # Check output contains upgrade message
        output = result.stdout
        assert "自动升级版本号" in output or "1.0.0 -> 1.0.1" in output or "1.0.1" in output, \
            f"Expected version upgrade message in output: {output}"

        print(f"✓ Version upgrade message shown correctly")

    def test_07_dry_run_no_version_change(self):
        """Test 7: Dry-run mode should not modify version"""
        print("\n=== Test 7: Dry Run No Version Change ===")

        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success

        skill_name = "dry-run-test"
        skill_dir = self._create_skill_with_version(skill_name, "1.0.0")

        # First feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success

        # Modify content
        self._modify_skill_content(skill_dir, "Dry run modification")

        # Dry-run feedback
        result = self.cmd.run("feedback", [skill_name, "--dry-run"], cwd=str(self.project_dir))
        assert result.success

        # Version should still be 1.0.0
        project_version = self._get_version_from_skill_md(skill_dir / "SKILL.md")
        assert project_version == "1.0.0", f"Dry-run should not change version, got {project_version}"

        print(f"✓ Dry-run mode does not modify version")

    def test_08_force_mode_with_version_upgrade(self):
        """Test 8: Force mode should still auto-upgrade version"""
        print("\n=== Test 8: Force Mode with Version Upgrade ===")

        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success

        skill_name = "force-mode-test"
        skill_dir = self._create_skill_with_version(skill_name, "1.0.0")

        # First feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success

        # Modify content
        self._modify_skill_content(skill_dir, "Force mode modification")

        # Force feedback (no confirmation)
        result = self.cmd.run("feedback", [skill_name, "--force"], cwd=str(self.project_dir))
        assert result.success

        # Version should be upgraded
        project_version = self._get_version_from_skill_md(skill_dir / "SKILL.md")
        assert project_version == "1.0.1", f"Expected version 1.0.1 with force mode, got {project_version}"

        print(f"✓ Force mode still auto-upgrades version")

    def test_09_version_comparison_logic(self):
        """Test 9: Verify version comparison logic"""
        print("\n=== Test 9: Version Comparison Logic ===")

        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success

        # Test case 1: Project version lower than repo
        skill_name = "version-compare-1"
        skill_dir = self._create_skill_with_version(skill_name, "1.0.5")

        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success

        # Modify but set version lower than repo (1.0.3)
        self._modify_skill_content(skill_dir, "Lower version test")
        skill_md = skill_dir / "SKILL.md"
        content = skill_md.read_text()
        content = content.replace("version: 1.0.5", "version: 1.0.3")
        skill_md.write_text(content)

        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success

        # Should upgrade to 1.0.6 (based on repo version 1.0.5)
        project_version = self._get_version_from_skill_md(skill_md)
        assert project_version == "1.0.6", f"Expected version 1.0.6, got {project_version}"

        print(f"  Lower version (1.0.3) upgraded to {project_version} ✓")

        # Test case 2: Project version higher than repo
        skill_name2 = "version-compare-2"
        skill_dir2 = self._create_skill_with_version(skill_name2, "2.0.0")

        result = self.cmd.run("feedback", [skill_name2], cwd=str(self.project_dir), input_text="y\n")
        assert result.success

        # Modify with higher version (3.0.0)
        self._modify_skill_content(skill_dir2, "Higher version test")
        skill_md2 = skill_dir2 / "SKILL.md"
        content2 = skill_md2.read_text()
        content2 = content2.replace("version: 2.0.0", "version: 3.0.0")
        skill_md2.write_text(content2)

        result = self.cmd.run("feedback", [skill_name2], cwd=str(self.project_dir), input_text="y\n")
        assert result.success

        # Should keep user version 3.0.0
        project_version2 = self._get_version_from_skill_md(skill_md2)
        assert project_version2 == "3.0.0", f"Expected version 3.0.0, got {project_version2}"

        print(f"  Higher version (3.0.0) preserved ✓")

        print(f"✓ Version comparison logic working correctly")


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
