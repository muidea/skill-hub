import json
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


class TestBulkImportRegisterStatus:
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir, temp_project_dir):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = Path(temp_project_dir)
        self.cmd = CommandRunner()

    def _init(self):
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}\n{result.stdout}"

    def _write_skill(self, skill_id: str, content: str) -> Path:
        skill_dir = self.project_dir / ".agents" / "skills" / skill_id
        skill_dir.mkdir(parents=True, exist_ok=True)
        skill_md = skill_dir / "SKILL.md"
        skill_md.write_text(content, encoding="utf-8")
        return skill_md

    def _state(self):
        state_path = self.home_dir / ".skill-hub" / "state.json"
        assert state_path.exists(), f"state.json not found: {state_path}"
        return json.loads(state_path.read_text(encoding="utf-8"))

    def test_register_existing_skill_and_status_json(self):
        self._init()
        self._write_skill(
            "manual-skill",
            """---
name: manual-skill
description: Existing manual skill for register coverage.
compatibility: Compatible with open_code
metadata:
  version: "1.0.0"
  author: "tester"
---
# Manual Skill

Use this skill in tests.
""",
        )

        result = self.cmd.run("register", ["manual-skill"], cwd=str(self.project_dir))
        assert result.success, f"register failed: {result.stderr}\n{result.stdout}"
        assert "不会创建或覆盖" in result.stdout

        result = self.cmd.run("status", ["--json"], cwd=str(self.project_dir))
        assert result.success, f"status --json failed: {result.stderr}\n{result.stdout}"
        data = json.loads(result.stdout)
        assert data["skill_count"] == 1
        assert data["items"][0]["skill_id"] == "manual-skill"
        assert data["items"][0]["status"] == "Modified"
        assert data["items"][0]["local_version"] == "1.0.0"

    def test_validate_fix_repairs_legacy_frontmatter(self):
        self._init()
        body = "# Legacy Skill\n\nThis legacy body should stay exactly the same.\n"
        skill_md = self._write_skill("legacy-skill", body)

        result = self.cmd.run(
            "register",
            ["legacy-skill", "--skip-validate"],
            cwd=str(self.project_dir),
        )
        assert result.success, f"register failed: {result.stderr}\n{result.stdout}"

        result = self.cmd.run("validate", ["legacy-skill", "--fix"], cwd=str(self.project_dir))
        assert result.success, f"validate --fix failed: {result.stderr}\n{result.stdout}"
        assert "备份文件" in result.stdout

        repaired = skill_md.read_text(encoding="utf-8")
        assert repaired.startswith("---\n")
        assert "name: legacy-skill" in repaired
        assert "description: This legacy body should stay exactly the same." in repaired
        assert "compatibility:" not in repaired
        assert "version: 1.0.0" in repaired
        assert repaired.split("---\n", 2)[2] == body
        assert list(skill_md.parent.glob("SKILL.md.bak.*")), "backup file was not created"

    def test_import_fix_frontmatter_and_archive(self):
        self._init()
        self._write_skill(
            "import-legacy",
            "# Import Legacy\n\nLegacy import body used as inferred description.\n",
        )
        self._write_skill(
            "import-valid",
            """---
name: import-valid
description: Valid imported skill for archive coverage.
compatibility: Compatible with open_code
metadata:
  version: "1.2.3"
  author: "tester"
---
# Import Valid
""",
        )

        result = self.cmd.run(
            "import",
            [".agents/skills", "--fix-frontmatter", "--archive", "--force"],
            cwd=str(self.project_dir),
        )
        assert result.success, f"import failed: {result.stderr}\n{result.stdout}"
        assert "discovered: 2" in result.stdout
        assert "registered: 2" in result.stdout
        assert "valid:      2" in result.stdout
        assert "archived:   2" in result.stdout
        assert "failed:     0" in result.stdout

        state = self._state()
        project_state = state[str(self.project_dir)]
        assert "import-legacy" in project_state["skills"]
        assert "import-valid" in project_state["skills"]

        repo_skills = self.home_dir / ".skill-hub" / "repositories" / "main" / "skills"
        assert (repo_skills / "import-legacy" / "SKILL.md").exists()
        assert (repo_skills / "import-valid" / "SKILL.md").exists()
        archived_legacy = (repo_skills / "import-legacy" / "SKILL.md").read_text(encoding="utf-8")
        assert "name: import-legacy" in archived_legacy
        assert "Legacy import body used as inferred description." in archived_legacy
