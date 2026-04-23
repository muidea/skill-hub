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
        monkeypatch.setenv("CODEX_SKILLS_DIR", str(self.codex_skills_dir))
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
            ["global-demo", "--global", "--agent", "codex"],
            cwd=str(self.project_dir),
        )
        assert use_result.success, f"use --global failed: {use_result.stderr}\n{use_result.stdout}"
        assert "已成功标记为本机全局使用" in use_result.stdout

        state = self._global_state()
        skill_state = state["enabled_skills"]["global-demo"]
        assert skill_state["source_repository"] == "main"
        assert skill_state["agents"] == ["codex"]
        assert skill_state["content_hash"].startswith("sha256:")

        status_before = self.cmd.run(
            "status",
            ["global-demo", "--global", "--agent", "codex", "--json"],
            cwd=str(self.project_dir),
        )
        assert status_before.success, f"status --global before apply failed: {status_before.stderr}\n{status_before.stdout}"
        before_data = json.loads(status_before.stdout)
        assert before_data["skill_count"] == 1
        assert before_data["items"][0]["status"] == "missing_agent_dir"

        dry_run = self.cmd.run(
            "apply",
            ["global-demo", "--global", "--agent", "codex", "--dry-run"],
            cwd=str(self.project_dir),
        )
        assert dry_run.success, f"apply --global --dry-run failed: {dry_run.stderr}\n{dry_run.stdout}"
        assert not (self.codex_skills_dir / "global-demo").exists()

        apply_result = self.cmd.run(
            "apply",
            ["global-demo", "--global", "--agent", "codex"],
            cwd=str(self.project_dir),
        )
        assert apply_result.success, f"apply --global failed: {apply_result.stderr}\n{apply_result.stdout}"
        assert (self.codex_skills_dir / "global-demo" / "SKILL.md").exists()
        manifest_path = self.codex_skills_dir / "global-demo" / ".skill-hub-manifest.json"
        assert manifest_path.exists()
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        assert manifest["managed_by"] == "skill-hub"
        assert manifest["scope"] == "global"
        assert manifest["agent"] == "codex"
        assert manifest["skill_id"] == "global-demo"

        status_after = self.cmd.run(
            "status",
            ["global-demo", "--global", "--agent", "codex", "--json"],
            cwd=str(self.project_dir),
        )
        assert status_after.success, f"status --global after apply failed: {status_after.stderr}\n{status_after.stdout}"
        after_data = json.loads(status_after.stdout)
        assert after_data["items"][0]["status"] == "ok"

        mismatch = self.cmd.run(
            "status",
            ["global-demo", "--global", "--agent", "opencode", "--json"],
            cwd=str(self.project_dir),
        )
        assert not mismatch.success
        assert "SKILL_NOT_FOUND" in mismatch.stderr or "SKILL_NOT_FOUND" in mismatch.stdout

        remove_result = self.cmd.run(
            "remove",
            ["global-demo", "--global", "--agent", "codex"],
            cwd=str(self.project_dir),
        )
        assert remove_result.success, f"remove --global failed: {remove_result.stderr}\n{remove_result.stdout}"
        assert not (self.codex_skills_dir / "global-demo").exists()
        assert "global-demo" not in self._global_state().get("enabled_skills", {})
