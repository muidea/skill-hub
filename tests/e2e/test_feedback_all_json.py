import json
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


@pytest.mark.no_debug
class TestFeedbackAllJSON:
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir, temp_project_dir):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = Path(temp_project_dir)
        self.cmd = CommandRunner()
        self.env = {"SKILL_HUB_DISABLE_SERVICE_BRIDGE": "1"}

    def _write_skill(self, skill_id: str, body: str):
        skill_dir = self.project_dir / ".agents" / "skills" / skill_id
        skill_dir.mkdir(parents=True, exist_ok=True)
        (skill_dir / "SKILL.md").write_text(
            f"""---
name: {skill_id}
description: Feedback all test skill {skill_id}.
compatibility: Compatible with open_code
metadata:
  version: "1.0.0"
  author: "tester"
---
# {skill_id}

{body}
""",
            encoding="utf-8",
        )

    def test_feedback_all_force_json_archives_registered_skills(self):
        init_result = self.cmd.run("init", cwd=str(self.project_dir), env=self.env)
        assert init_result.success, f"init failed: {init_result.stderr}\n{init_result.stdout}"

        self._write_skill("feedback-one", "first body")
        self._write_skill("feedback-two", "second body")
        for skill_id in ("feedback-one", "feedback-two"):
            register_result = self.cmd.run("register", [skill_id], cwd=str(self.project_dir), env=self.env)
            assert register_result.success, f"register {skill_id} failed: {register_result.stderr}\n{register_result.stdout}"

        feedback_result = self.cmd.run(
            "feedback",
            ["--all", "--force", "--json"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert feedback_result.success, f"feedback --all failed: {feedback_result.stderr}\n{feedback_result.stdout}"
        data = json.loads(feedback_result.stdout)
        assert data["total"] == 2
        assert data["applied"] == 2
        assert data["failed"] == 0
        assert {item["skill_id"] for item in data["items"]} == {"feedback-one", "feedback-two"}

        repo_skills = self.home_dir / ".skill-hub" / "repositories" / "main" / "skills"
        assert (repo_skills / "feedback-one" / "SKILL.md").exists()
        assert (repo_skills / "feedback-two" / "SKILL.md").exists()

        push_result = self.cmd.run(
            "push",
            ["--dry-run", "--json"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert push_result.success, f"push --dry-run --json failed: {push_result.stderr}\n{push_result.stdout}"
        push_data = json.loads(push_result.stdout)
        assert push_data["dry_run"] is True
        assert push_data["status"] == "planned"
        assert push_data["has_changes"] is True
        assert any("feedback-one" in item for item in push_data["changed_files"])

        pull_check = self.cmd.run(
            "pull",
            ["--check", "--json"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert pull_check.success, f"pull --check --json failed: {pull_check.stderr}\n{pull_check.stdout}"
        pull_data = json.loads(pull_check.stdout)
        assert pull_data["check"] is True
        assert pull_data["status"] in {"no_remote", "up_to_date", "updates_available", "ahead", "divergent"}

        git_status = self.cmd.run(
            "git",
            ["status", "--json"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert git_status.success, f"git status --json failed: {git_status.stderr}\n{git_status.stdout}"
        git_data = json.loads(git_status.stdout)
        assert git_data["state"] == "dirty"
        assert any("feedback-one" in item for item in git_data["changed_files"])

        git_sync = self.cmd.run(
            "git",
            ["sync", "--json"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert not git_sync.success
        git_sync_data = json.loads(git_sync.stdout)
        assert git_sync_data["command"] == "sync"
        assert git_sync_data["status"] == "failed"
