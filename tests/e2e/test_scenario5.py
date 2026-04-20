"""
Test Scenario 5: Target entrypoints are removed from business workflows.

The target concept is retained only as legacy compatibility metadata when a
skill explicitly documents it. CLI inputs and project state must not use it to
select, validate, filter, or apply skills.
"""

import json
import shutil
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


class TestScenario5TargetBusinessRemoval:
    """Regression coverage for removing target from active business logic."""

    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir):
        self.home_dir = Path(temp_home_dir)
        self.cmd = CommandRunner()
        self.skill_hub_dir = self.home_dir / ".skill-hub"
        self.project_dir = self.home_dir / "test-project"
        self.project_skills_dir = self.project_dir / ".agents" / "skills"
        self.repo_skills_dir = self.skill_hub_dir / "repositories" / "main" / "skills"
        self.project_dir.mkdir(exist_ok=True)

        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}"

    def test_01_removed_cli_target_entrypoints_fail(self):
        """Removed target commands and flags should not be accepted."""
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert not result.success, "set-target should be removed"

        result = self.cmd.run("list", ["--target", "open_code"], cwd=str(self.project_dir))
        assert not result.success, "list --target should be removed"

        result = self.cmd.run("create", ["target-flag-skill", "--target", "open_code"], cwd=str(self.project_dir))
        assert not result.success, "create --target should be removed"

        result = self.cmd.run("use", ["missing-skill", "--target", "open_code"], cwd=str(self.project_dir))
        assert not result.success, "use --target should be removed"

    def test_02_standard_workflows_do_not_write_preferred_target(self):
        """create/use/apply should operate without preferred_target state."""
        skill_id = "standard-targetless-skill"

        result = self.cmd.run("create", [skill_id], cwd=str(self.project_dir), input_text="\n")
        assert result.success, f"create failed: {result.stderr}"

        result = self.cmd.run("feedback", [skill_id], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"feedback failed: {result.stderr}"

        project_skill_dir = self.project_skills_dir / skill_id
        if project_skill_dir.exists():
            shutil.rmtree(project_skill_dir)

        result = self.cmd.run("use", [skill_id], cwd=str(self.project_dir), input_text="\n")
        assert result.success, f"use failed: {result.stderr}"

        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"apply failed: {result.stderr}"
        assert (self.project_skills_dir / skill_id / "SKILL.md").exists()

        project_state = self._project_state()
        assert project_state.get("preferred_target", "") == ""
        assert skill_id in project_state.get("skills", {})

    def test_03_compatibility_metadata_does_not_filter_list(self):
        """List should return all skills and only display compatibility metadata."""
        self._create_repo_skill("compat-open-code", "open_code")
        self._create_repo_skill("compat-cursor", "cursor")

        result = self.cmd.run("list", cwd=str(self.project_dir))
        assert result.success, f"list failed: {result.stderr}"
        assert "compat-open-code" in result.stdout
        assert "compat-cursor" in result.stdout

    def _create_repo_skill(self, skill_id: str, compatibility: str):
        result = self.cmd.run("create", [skill_id], cwd=str(self.project_dir), input_text="\n")
        assert result.success, f"create failed for {skill_id}: {result.stderr}"
        skill_dir = self.project_skills_dir / skill_id
        (skill_dir / "SKILL.md").write_text(
            "\n".join(
                [
                    "---",
                    f"name: {skill_id}",
                    "description: compatibility metadata regression skill",
                    "version: 1.0.0",
                    f"compatibility: {compatibility}",
                    "---",
                    "",
                    "# Regression Skill",
                    "",
                ]
            ),
            encoding="utf-8",
        )
        result = self.cmd.run("feedback", [skill_id], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"feedback failed for {skill_id}: {result.stderr}"

    def _project_state(self):
        state_path = self.skill_hub_dir / "state.json"
        assert state_path.exists(), "state.json should exist"
        with open(state_path, "r", encoding="utf-8") as f:
            state = json.load(f)
        project_path = str(self.project_dir)
        assert project_path in state, "project should be present in state"
        return state[project_path]
