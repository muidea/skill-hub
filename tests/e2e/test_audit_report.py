import json
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


@pytest.mark.no_debug
class TestAuditReport:
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir, temp_project_dir):
        self.project_dir = Path(temp_project_dir)
        self.cmd = CommandRunner()
        self.env = {"SKILL_HUB_DISABLE_SERVICE_BRIDGE": "1"}

    def _prepare_project(self):
        init_result = self.cmd.run("init", cwd=str(self.project_dir), env=self.env)
        assert init_result.success, f"init failed: {init_result.stderr}\n{init_result.stdout}"

        (self.project_dir / "docs").mkdir(parents=True, exist_ok=True)
        (self.project_dir / "docs" / "guide.md").write_text("# Guide\n", encoding="utf-8")
        skill_dir = self.project_dir / ".agents" / "skills" / "audit-skill"
        skill_dir.mkdir(parents=True, exist_ok=True)
        (skill_dir / "SKILL.md").write_text(
            """---
name: audit-skill
description: Skill used by audit report tests.
compatibility: Compatible with open_code
metadata:
  version: "1.0.0"
  author: "tester"
---
# Audit Skill

Read [guide](docs/guide.md).
""",
            encoding="utf-8",
        )
        register_result = self.cmd.run("register", ["audit-skill"], cwd=str(self.project_dir), env=self.env)
        assert register_result.success, f"register failed: {register_result.stderr}\n{register_result.stdout}"

    def test_audit_writes_markdown_and_json_reports(self):
        self._prepare_project()

        markdown_path = self.project_dir / ".agents" / "skills-refresh-progress.md"
        markdown_result = self.cmd.run(
            "audit",
            [".agents/skills", "--output", str(markdown_path)],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert markdown_result.success, f"audit markdown failed: {markdown_result.stderr}\n{markdown_result.stdout}"
        report_text = markdown_path.read_text(encoding="utf-8")
        assert "# Skill Hub Audit Report" in report_text
        assert "| Target Skills | 1 |" in report_text
        assert "| Registered | 1 |" in report_text
        assert "| Link Issues | 0 |" in report_text

        json_result = self.cmd.run(
            "audit",
            [".agents/skills", "--format", "json"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert json_result.success, f"audit json failed: {json_result.stderr}\n{json_result.stdout}"
        data = json.loads(json_result.stdout)
        assert data["target_skill_count"] == 1
        assert data["registered_count"] == 1
        assert data["link_issue_count"] == 0
        assert data["absolute_path_hits"] == 0
