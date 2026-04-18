import json
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


@pytest.mark.no_debug
class TestPathLint:
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir, temp_project_dir):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = Path(temp_project_dir)
        self.cmd = CommandRunner()
        self.env = {"SKILL_HUB_DISABLE_SERVICE_BRIDGE": "1"}

    def _write_skill_with_paths(self) -> Path:
        skill_dir = self.project_dir / ".agents" / "skills" / "path-skill"
        skill_dir.mkdir(parents=True, exist_ok=True)
        skill_md = skill_dir / "SKILL.md"
        skill_md.write_text(
            """---
name: path-skill
description: Skill with local paths for lint coverage.
compatibility: Compatible with open_code
metadata:
  version: "1.0.0"
  author: "tester"
---
# Path Skill

Read /home/tester/workspace/docs/foo.md before running.
Open file:///home/tester/workspace/scripts/setup.sh for setup.
Keep /home/other/tool as an external path.
Review vscode://file/home/tester/workspace/docs/foo.md manually.
""",
            encoding="utf-8",
        )
        return skill_md

    def test_lint_paths_reports_and_fixes_project_paths(self):
        skill_md = self._write_skill_with_paths()

        report_result = self.cmd.run(
            "lint",
            [".", "--paths", "--project-root", "/home/tester/workspace", "--json"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert report_result.success, f"lint --json failed: {report_result.stderr}\n{report_result.stdout}"
        report = json.loads(report_result.stdout)
        assert report["finding_count"] == 4
        assert report["rewritten"] == 0
        assert report["manual_review"] == 2
        fixable = [item for item in report["findings"] if item["status"] == "fixable"]
        assert {item["replacement"] for item in fixable} == {"docs/foo.md", "scripts/setup.sh"}

        fix_result = self.cmd.run(
            "lint",
            [".", "--paths", "--project-root", "/home/tester/workspace", "--fix"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert fix_result.success, f"lint --fix failed: {fix_result.stderr}\n{fix_result.stdout}"
        assert "rewritten:     2" in fix_result.stdout
        assert "manual-review: 2" in fix_result.stdout

        updated = skill_md.read_text(encoding="utf-8")
        assert "Read /home/tester/workspace/docs/foo.md" not in updated
        assert "file:///home/tester/workspace/scripts/setup.sh" not in updated
        assert "docs/foo.md" in updated
        assert "scripts/setup.sh" in updated
        assert "/home/other/tool" in updated
        assert "vscode://file/home/tester/workspace/docs/foo.md" in updated
        assert list(skill_md.parent.glob("SKILL.md.bak.*")), "backup file was not created"

    def test_lint_paths_dry_run_does_not_modify_files(self):
        skill_md = self._write_skill_with_paths()
        before = skill_md.read_text(encoding="utf-8")

        dry_run_result = self.cmd.run(
            "lint",
            [".", "--paths", "--project-root", "/home/tester/workspace", "--fix", "--dry-run", "--json"],
            cwd=str(self.project_dir),
            env=self.env,
        )
        assert dry_run_result.success, f"lint --dry-run failed: {dry_run_result.stderr}\n{dry_run_result.stdout}"
        report = json.loads(dry_run_result.stdout)
        assert report["rewritten"] == 2
        assert {item["status"] for item in report["findings"] if item.get("replacement")} == {"would-rewrite"}
        assert skill_md.read_text(encoding="utf-8") == before
        assert not list(skill_md.parent.glob("SKILL.md.bak.*"))
