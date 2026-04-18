import json
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


@pytest.mark.no_debug
class TestValidateLinks:
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir, temp_project_dir):
        self.project_dir = Path(temp_project_dir)
        self.cmd = CommandRunner()
        self.env = {"SKILL_HUB_DISABLE_SERVICE_BRIDGE": "1"}

    def _init_project(self):
        result = self.cmd.run("init", cwd=str(self.project_dir), env=self.env)
        assert result.success, f"init failed: {result.stderr}\n{result.stdout}"

    def _write_skill(self) -> Path:
        (self.project_dir / "docs").mkdir(parents=True, exist_ok=True)
        (self.project_dir / "docs" / "guide.md").write_text("# Guide\n", encoding="utf-8")
        skill_dir = self.project_dir / ".agents" / "skills" / "link-skill"
        (skill_dir / "references").mkdir(parents=True, exist_ok=True)
        (skill_dir / "references" / "note.md").write_text("# Note\n", encoding="utf-8")
        skill_md = skill_dir / "SKILL.md"
        skill_md.write_text(
            """---
name: link-skill
description: Skill with markdown links.
compatibility: Compatible with open_code
metadata:
  version: "1.0.0"
  author: "tester"
---
# Link Skill

Read [project guide](docs/guide.md).
Read [bundled note](references/note.md).
Ignore [external docs](https://example.invalid/docs).
Report [missing doc](missing.md).
""",
            encoding="utf-8",
        )
        return skill_dir / "missing.md"

    def test_validate_links_reports_broken_local_links_and_passes_after_fix(self):
        self._init_project()
        missing_doc = self._write_skill()

        register_result = self.cmd.run(
            "register",
            ["link-skill"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert register_result.success, f"register failed: {register_result.stderr}\n{register_result.stdout}"

        broken_result = self.cmd.run(
            "validate",
            ["link-skill", "--links", "--json"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert not broken_result.success, "validate --links should fail with a broken local link"
        report = json.loads(broken_result.stdout)
        assert report["failed"] == 1
        assert report["link_issue_count"] == 1
        assert report["link_issues"][0]["link"] == "missing.md"
        assert report["link_issues"][0]["status"] == "broken"

        missing_doc.write_text("# Missing\n", encoding="utf-8")
        fixed_result = self.cmd.run(
            "validate",
            ["--all", "--links", "--json"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert fixed_result.success, f"validate --all --links failed: {fixed_result.stderr}\n{fixed_result.stdout}"
        fixed_report = json.loads(fixed_result.stdout)
        assert fixed_report["passed"] == 1
        assert fixed_report["link_issue_count"] == 0
