import json
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


class TestProjectApplyOutdated:
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = self.home_dir / "project"
        self.project_dir.mkdir(exist_ok=True)
        self.repo_skills_dir = self.home_dir / ".skill-hub" / "repositories" / "main" / "skills"
        self.project_skills_dir = self.project_dir / ".agents" / "skills"
        self.cmd = CommandRunner()

    def _write_repo_skill(self, skill_id: str, version: str, marker: str):
        skill_dir = self.repo_skills_dir / skill_id
        skill_dir.mkdir(parents=True, exist_ok=True)
        (skill_dir / "SKILL.md").write_text(
            f"""---
name: {skill_id}
description: Outdated apply coverage.
metadata:
  version: "{version}"
---
# {skill_id}

{marker}
""",
            encoding="utf-8",
        )

    def _status_item(self, skill_id: str):
        result = self.cmd.run("status", [skill_id, "--json"], cwd=str(self.project_dir))
        assert result.success, f"status --json failed: {result.stderr}\n{result.stdout}"
        data = json.loads(result.stdout)
        assert data["skill_count"] == 1
        assert data["items"][0]["skill_id"] == skill_id
        return data["items"][0]

    def test_apply_specific_outdated_skill_refreshes_project_copy(self):
        skill_id = "outdated-apply-demo"

        init_result = self.cmd.run("init", cwd=str(self.project_dir))
        assert init_result.success, f"init failed: {init_result.stderr}\n{init_result.stdout}"

        self._write_repo_skill(skill_id, "1.0.0", "repo-v1")

        use_result = self.cmd.run("use", [skill_id], cwd=str(self.project_dir))
        assert use_result.success, f"use failed: {use_result.stderr}\n{use_result.stdout}"

        apply_initial = self.cmd.run("apply", [skill_id], cwd=str(self.project_dir))
        assert apply_initial.success, f"initial apply failed: {apply_initial.stderr}\n{apply_initial.stdout}"

        project_skill_md = self.project_skills_dir / skill_id / "SKILL.md"
        assert "repo-v1" in project_skill_md.read_text(encoding="utf-8")

        self._write_repo_skill(skill_id, "1.1.0", "repo-v2")

        before = self._status_item(skill_id)
        assert before["status"] == "Outdated", before

        apply_refresh = self.cmd.run("apply", [skill_id], cwd=str(self.project_dir))
        assert apply_refresh.success, f"refresh apply failed: {apply_refresh.stderr}\n{apply_refresh.stdout}"
        assert "成功应用技能" in apply_refresh.stdout

        refreshed_content = project_skill_md.read_text(encoding="utf-8")
        assert "repo-v2" in refreshed_content
        assert 'version: "1.1.0"' in refreshed_content

        after = self._status_item(skill_id)
        assert after["status"] == "Synced", after
        assert after["local_version"] == "1.1.0"
        assert after["repo_version"] == "1.1.0"
