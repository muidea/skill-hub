import json
import shutil
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


class TestStatePrune:
    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = self.home_dir / "test-project"
        self.project_dir.mkdir(exist_ok=True)
        self.skill_hub_dir = self.home_dir / ".skill-hub"
        self.cmd = CommandRunner()

    def test_01_prune_keeps_clean_state_unchanged(self):
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}"

        result = self.cmd.run("create", ["prune-state-skill"], cwd=str(self.project_dir), input_text="\n")
        assert result.success, f"create failed: {result.stderr}"

        result = self.cmd.run("prune", cwd=str(self.home_dir))
        assert result.success, f"prune failed: {result.stderr}"
        assert "未发现失效项目记录" in result.stdout, result.stdout

        state = self._load_state()
        assert str(self.project_dir) in state, "有效项目记录不应被 prune 删除"

    def test_02_prune_removes_missing_project_record(self):
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}"

        result = self.cmd.run("create", ["prune-stale-skill"], cwd=str(self.project_dir), input_text="\n")
        assert result.success, f"create failed: {result.stderr}"

        stale_project_path = str(self.project_dir)
        shutil.rmtree(self.project_dir)
        assert not self.project_dir.exists(), "测试前提失败：项目目录仍存在"

        result = self.cmd.run("prune", cwd=str(self.home_dir))
        assert result.success, f"prune failed: {result.stderr}"
        assert "已清理 1 条失效项目记录" in result.stdout, result.stdout
        assert stale_project_path in result.stdout, result.stdout

        state = self._load_state()
        assert stale_project_path not in state, "失效项目记录应被 prune 删除"

    def _load_state(self):
        state_path = self.skill_hub_dir / "state.json"
        assert state_path.exists(), f"state.json not found: {state_path}"
        with open(state_path, "r", encoding="utf-8") as f:
            return json.load(f)
