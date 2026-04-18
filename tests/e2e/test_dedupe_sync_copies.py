import json
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


class TestDedupeSyncCopies:
    @pytest.fixture(autouse=True)
    def setup(self, temp_project_dir):
        self.project_dir = Path(temp_project_dir)
        self.cmd = CommandRunner()

    def _write_skill(self, base: Path, skill_id: str, description: str, body: str) -> Path:
        skill_dir = base / ".agents" / "skills" / skill_id
        skill_dir.mkdir(parents=True, exist_ok=True)
        skill_md = skill_dir / "SKILL.md"
        skill_md.write_text(
            f"""---
name: {skill_id}
description: {description}
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
        return skill_md

    def test_dedupe_reports_conflict_and_sync_copies_repairs_it(self):
        canonical_skill = self._write_skill(
            self.project_dir,
            "dup-skill",
            "Canonical duplicate skill.",
            "canonical body",
        )
        nested_project = self.project_dir / "packages" / "child"
        nested_skill = self._write_skill(
            nested_project,
            "dup-skill",
            "Nested duplicate skill.",
            "nested body",
        )
        self._write_skill(
            self.project_dir,
            "same-skill",
            "Canonical identical skill.",
            "same body",
        )
        self._write_skill(
            nested_project,
            "same-skill",
            "Canonical identical skill.",
            "same body",
        )

        result = self.cmd.run(
            "dedupe",
            [".", "--canonical", ".agents/skills", "--json"],
            cwd=str(self.project_dir),
        )
        assert result.success, f"dedupe failed: {result.stderr}\n{result.stdout}"
        report = json.loads(result.stdout)
        assert report["skill_count"] == 2
        conflict_group = next(group for group in report["groups"] if group["skill_id"] == "dup-skill")
        assert conflict_group["content_differs"] is True
        assert report["conflicts"] == 1

        dry_run = self.cmd.run(
            "sync-copies",
            ["--canonical", ".agents/skills", "--scope", ".", "--dry-run", "--json"],
            cwd=str(self.project_dir),
        )
        assert dry_run.success, f"sync dry-run failed: {dry_run.stderr}\n{dry_run.stdout}"
        dry_data = json.loads(dry_run.stdout)
        assert any(item["status"] == "planned" for item in dry_data["items"])
        assert "nested body" in nested_skill.read_text(encoding="utf-8")

        sync = self.cmd.run(
            "sync-copies",
            ["--canonical", ".agents/skills", "--scope", "."],
            cwd=str(self.project_dir),
        )
        assert sync.success, f"sync-copies failed: {sync.stderr}\n{sync.stdout}"
        assert "synced:    1" in sync.stdout
        assert nested_skill.read_text(encoding="utf-8") == canonical_skill.read_text(encoding="utf-8")
        assert list(nested_skill.parent.parent.glob("dup-skill.bak.*")), "backup directory was not created"

        after = self.cmd.run(
            "dedupe",
            [".", "--canonical", ".agents/skills", "--json"],
            cwd=str(self.project_dir),
        )
        assert after.success, f"dedupe after sync failed: {after.stderr}\n{after.stdout}"
        after_report = json.loads(after.stdout)
        assert after_report["conflicts"] == 0
