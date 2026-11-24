import json
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


class TestGlobalSkillManagement:
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir, temp_project_dir, monkeypatch):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = Path(temp_project_dir)
        self.codex_skills_dir = self.home_dir / "codex" / "skills"
        self.agy_skills_dir = self.home_dir / "agy" / "skills"
        monkeypatch.setenv("CODEX_SKILLS_DIR", str(self.codex_skills_dir))
        monkeypatch.setenv("AGY_SKILLS_DIR", str(self.agy_skills_dir))
        self.cmd = CommandRunner()

    def _init(self):
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}\n{result.stdout}"

    def _write_repo_skill(self, skill_id: str) -> Path:
        skill_dir = self.home_dir / ".skill-hub" / "repositories" / "main" / "skills" / skill_id
        skill_dir.mkdir(parents=True, exist_ok=True)
        skill_md = skill_dir / "SKILL.md"
        skill_md.write_text(
            f"""---
name: {skill_id}
description: Skill used by the global skill management e2e test.
metadata:
  version: "1.0.0"
  author: "tester"
---
# {skill_id}

Use this skill for global management coverage.
""",
            encoding="utf-8",
        )
        return skill_md

    def _global_state(self):
        state_path = self.home_dir / ".skill-hub" / "global-state.json"
        assert state_path.exists(), f"global-state.json not found: {state_path}"
        return json.loads(state_path.read_text(encoding="utf-8"))

    @pytest.mark.no_debug
    def test_use_status_apply_and_remove_global_skill(self):
        self._init()
        self._write_repo_skill("global-demo")

        use_result = self.cmd.run(
            "use",
            ["global-demo", "--global"],
            cwd=str(self.project_dir),
        )
        assert use_result.success, f"use --global failed: {use_result.stderr}\n{use_result.stdout}"
        assert "已启用 global-demo" in use_result.stdout

        state = self._global_state()
        skill_state = state["enabled_skills"]["global-demo"]
        assert skill_state["source_repository"] == "main"
        assert "codex" in skill_state["agents"]
        assert "agy" in skill_state["agents"]
        assert skill_state["content_hash"].startswith("sha256:")

        status_before = self.cmd.run(
            "status",
            ["--pattern", "global-demo", "--global", "--json"],
            cwd=str(self.project_dir),
        )
        assert status_before.success, f"status --global before apply failed: {status_before.stderr}\n{status_before.stdout}"
        before_data = json.loads(status_before.stdout)
        assert before_data["skill_count"] >= 1
        for expected_agent in ("codex", "agy"):
            before_item = next(item for item in before_data["items"] if item["agent"] == expected_agent)
            assert before_item["status"] == "unavailable"
            assert before_item["legacy_status"] == "missing_agent_dir"

        dry_run = self.cmd.run(
            "apply",
            ["--pattern", "global-demo", "--global", "--dry-run"],
            cwd=str(self.project_dir),
        )
        assert dry_run.success, f"apply --global --dry-run failed: {dry_run.stderr}\n{dry_run.stdout}"
        assert not (self.codex_skills_dir / "global-demo").exists()
        assert not (self.agy_skills_dir / "global-demo").exists()

        # Pre-create an unmanaged conflict in AGY skills_dir: apply without --force
        # must reject overwriting it, and apply with --force must move the backup
        # outside AGY's 1-level scan directory.
        agy_conflict = self.agy_skills_dir / "global-demo"
        agy_conflict.mkdir(parents=True, exist_ok=True)
        (agy_conflict / "SKILL.md").write_text("unmanaged agy content", encoding="utf-8")

        conflict_apply = self.cmd.run(
            "apply",
            ["--pattern", "global-demo", "--global", "--json"],
            cwd=str(self.project_dir),
        )
        assert not conflict_apply.success, "apply --global without --force should fail on AGY conflict"
        conflict_data = json.loads(conflict_apply.stdout)
        conflict_items = conflict_data["results"][0]["items"]
        agy_conflict_item = next(item for item in conflict_items if item["agent"] == "agy")
        assert agy_conflict_item["status"] == "conflict"
        assert (agy_conflict / "SKILL.md").read_text(encoding="utf-8") == "unmanaged agy content"

        apply_result = self.cmd.run(
            "apply",
            ["--pattern", "global-demo", "--global", "--force"],
            cwd=str(self.project_dir),
        )
        assert apply_result.success, f"apply --global --force failed: {apply_result.stderr}\n{apply_result.stdout}"
        assert not list(self.agy_skills_dir.glob("*.skill-hub-backup.*"))
        assert list((self.home_dir / ".skill-hub" / "backups" / "global" / "agy").glob("global-demo.*"))

        for expected_agent, target_dir in (("codex", self.codex_skills_dir), ("agy", self.agy_skills_dir)):
            assert (target_dir / "global-demo" / "SKILL.md").exists()
            manifest_path = target_dir / "global-demo" / ".skill-hub-manifest.json"
            assert manifest_path.exists()
            manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
            assert manifest["managed_by"] == "skill-hub"
            assert manifest["scope"] == "global"
            assert manifest["agent"] == expected_agent
            assert manifest["skill_id"] == "global-demo"

        status_after = self.cmd.run(
            "status",
            ["--pattern", "global-demo", "--global", "--json"],
            cwd=str(self.project_dir),
        )
        assert status_after.success, f"status --global after apply failed: {status_after.stderr}\n{status_after.stdout}"
        after_data = json.loads(status_after.stdout)
        for expected_agent in ("codex", "agy"):
            after_item = next(item for item in after_data["items"] if item["agent"] == expected_agent)
            assert after_item["status"] == "synced"
            assert after_item["legacy_status"] == "ok"

        # Locally modify the AGY copy and confirm status reports modified, non-force
        # apply refuses to overwrite, and --force externally backs up & resyncs.
        agy_skill_md = self.agy_skills_dir / "global-demo" / "SKILL.md"
        agy_skill_md.write_text("locally edited agy skill", encoding="utf-8")

        status_modified = self.cmd.run(
            "status",
            ["--pattern", "global-demo", "--global", "--json"],
            cwd=str(self.project_dir),
        )
        assert status_modified.success, f"status --global modified failed: {status_modified.stderr}\n{status_modified.stdout}"
        modified_data = json.loads(status_modified.stdout)
        agy_modified_item = next(item for item in modified_data["items"] if item["agent"] == "agy")
        assert agy_modified_item["status"] == "modified"

        refused_apply = self.cmd.run(
            "apply",
            ["--pattern", "global-demo", "--global", "--json"],
            cwd=str(self.project_dir),
        )
        assert not refused_apply.success, "apply --global without --force should refuse overwriting modified AGY copy"
        refused_data = json.loads(refused_apply.stdout)
        refused_items = refused_data["results"][0]["items"]
        agy_refused_item = next(item for item in refused_items if item["agent"] == "agy")
        assert agy_refused_item["status"] == "conflict"
        assert agy_refused_item["reason"] == "target_modified"
        assert agy_skill_md.read_text(encoding="utf-8") == "locally edited agy skill"

        resync_apply = self.cmd.run(
            "apply",
            ["--pattern", "global-demo", "--global", "--force"],
            cwd=str(self.project_dir),
        )
        assert resync_apply.success, f"apply --global --force resync failed: {resync_apply.stderr}\n{resync_apply.stdout}"
        assert not list(self.agy_skills_dir.glob("*.skill-hub-backup.*"))
        assert len(list((self.home_dir / ".skill-hub" / "backups" / "global" / "agy").glob("global-demo.*"))) >= 2
        assert "locally edited agy skill" not in agy_skill_md.read_text(encoding="utf-8")

        remove_result = self.cmd.run(
            "remove",
            ["global-demo", "--global"],
            cwd=str(self.project_dir),
        )
        assert remove_result.success, f"remove --global failed: {remove_result.stderr}\n{remove_result.stdout}"
        assert not (self.codex_skills_dir / "global-demo").exists()
        assert not (self.agy_skills_dir / "global-demo").exists()
        assert "global-demo" not in self._global_state().get("enabled_skills", {})
