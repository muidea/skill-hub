"""
E2E tests for skill content handling: create, status, feedback, apply, use.

Verifies:
- create: new skill has SKILL.md + scripts/, references/, assets/; existing skill triggers validation and state refresh for registration/archiving.
- status: subdir (scripts/references/assets) changes are reflected as Modified.
- feedback: syncs full skill directory (covered in test_feedback_apply_multifile).
- apply (open_code): copies full skill dir from repo (covered in test_feedback_apply_multifile).
- use: only updates state.json, does not copy skill files.
"""

import json
import os
import shutil
import pytest
from pathlib import Path

from tests.e2e.utils.command_runner import CommandRunner
from tests.e2e.utils.test_environment import TestEnvironment


class TestCreateSkillStructure:
    """create 命令：技能目录结构；已存在时验证并刷新状态"""

    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = self.home_dir / "test-project"
        self.project_dir.mkdir(exist_ok=True)
        self.skill_hub_dir = self.home_dir / ".skill-hub"
        self.agents_skills_dir = self.project_dir / ".agents" / "skills"
        self.cmd = CommandRunner()

    def test_01_create_new_skill_has_standard_structure(self):
        """create 新建技能应包含 SKILL.md 与 scripts/references/assets 子目录"""
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}"

        skill_id = "e2e-structure-skill"
        result = self.cmd.run(
            "create", [skill_id], cwd=str(self.project_dir), input_text="\n"
        )
        assert result.success, f"create failed: {result.stderr}"

        skill_dir = self.agents_skills_dir / skill_id
        assert skill_dir.exists(), f"技能目录未创建: {skill_dir}"
        assert (skill_dir / "SKILL.md").is_file(), "SKILL.md 未创建"
        for sub in ("scripts", "references", "assets"):
            subdir = skill_dir / sub
            assert subdir.is_dir(), f"子目录 {sub}/ 未创建"

    def test_02_create_existing_skill_validates_and_refreshes(self):
        """create 已存在的技能时应主动验证并按需刷新状态，便于登记与归档"""
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}"

        skill_id = "e2e-existing-skill"
        result = self.cmd.run(
            "create", [skill_id], cwd=str(self.project_dir), input_text="\n"
        )
        assert result.success, f"首次 create 失败: {result.stderr}"

        result = self.cmd.run(
            "create", [skill_id], cwd=str(self.project_dir), input_text="\n"
        )
        assert result.success, "对已存在技能 create 应成功（验证并刷新状态）"
        combined = result.stdout + result.stderr
        assert "技能文件已存在" in combined or "已存在" in combined
        assert "验证" in combined or "刷新" in combined or "登记" in combined

    def test_03_create_existing_skill_synced_with_repo_does_nothing(self):
        """create 已存在且已在 state 登记且与仓库一致时，不做任何操作"""
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}"

        skill_id = "e2e-synced-skill"
        result = self.cmd.run(
            "create", [skill_id], cwd=str(self.project_dir), input_text="\n"
        )
        assert result.success, f"首次 create 失败: {result.stderr}"

        result = self.cmd.run("feedback", [skill_id], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"feedback 失败: {result.stderr}"

        result = self.cmd.run(
            "create", [skill_id], cwd=str(self.project_dir), input_text="\n"
        )
        assert result.success, "已登记且与仓库一致时 create 应成功"
        combined = result.stdout + result.stderr
        assert "无需操作" in combined or "与仓库一致" in combined


class TestStatusSkillContent:
    """status 命令：子目录变更应显示为 Modified"""

    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = self.home_dir / "test-project"
        self.project_dir.mkdir(exist_ok=True)
        self.repo_skills_dir = self.home_dir / ".skill-hub" / "repositories" / "main" / "skills"
        self.agents_skills_dir = self.project_dir / ".agents" / "skills"
        self.cmd = CommandRunner()

    def test_01_status_shows_modified_when_subdir_file_changes(self):
        """子目录（如 scripts/）有变更时，status 应显示 Modified"""
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}"

        skill_id = "e2e-status-subdir-skill"
        result = self.cmd.run(
            "create", [skill_id], cwd=str(self.project_dir), input_text="\n"
        )
        assert result.success, f"create failed: {result.stderr}"

        result = self.cmd.run("feedback", [skill_id], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"feedback failed: {result.stderr}"

        skill_dir = self.agents_skills_dir / skill_id
        (skill_dir / "scripts").mkdir(exist_ok=True)
        (skill_dir / "scripts" / "run.sh").write_text("#!/bin/bash\necho test\n")

        result = self.cmd.run("status", cwd=str(self.project_dir))
        assert result.success, f"status failed: {result.stderr}"
        assert "Modified" in result.stdout, "子目录新增文件后 status 应显示 Modified"


class TestUseOnlyUpdatesState:
    """use 命令：仅更新 state，不复制技能文件内容"""

    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = self.home_dir / "test-project"
        self.project_dir.mkdir(exist_ok=True)
        self.skill_hub_dir = self.home_dir / ".skill-hub"
        self.agents_skills_dir = self.project_dir / ".agents" / "skills"
        self.cmd = CommandRunner()

    def test_01_use_registers_skill_in_state_only(self):
        """use 仅更新 state.json，技能文件由 apply 从仓库拉取"""
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}"

        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success, f"set-target failed: {result.stderr}"

        skill_id = "e2e-use-state-skill"
        result = self.cmd.run(
            "create", [skill_id], cwd=str(self.project_dir), input_text="\n"
        )
        assert result.success, f"create failed: {result.stderr}"

        result = self.cmd.run("feedback", [skill_id], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"feedback failed: {result.stderr}"

        project_skill_dir = self.agents_skills_dir / skill_id
        assert project_skill_dir.exists()
        shutil.rmtree(project_skill_dir)
        assert not project_skill_dir.exists(), "已删除项目内技能目录以便测试 use"

        result = self.cmd.run("use", [skill_id], cwd=str(self.project_dir), input_text="\n")
        assert result.success, f"use failed: {result.stderr}"

        state_file = self.skill_hub_dir / "state.json"
        assert state_file.exists(), "state.json 未创建"
        with open(state_file, "r", encoding="utf-8") as f:
            state = json.load(f)
        project_path = str(self.project_dir)
        assert project_path in state, "当前项目未写入 state"
        assert "skills" in state[project_path], "项目 state 无 skills"
        assert skill_id in state[project_path]["skills"], "use 后 state 中应有该技能"

        assert not project_skill_dir.exists(), "use 不应创建技能目录，应由 apply 创建"


class TestFeedbackApplyFullDirectory:
    """feedback/apply 对完整技能目录的同步（与 test_feedback_apply_multifile 互补）"""

    @pytest.fixture(autouse=True)
    def setup(self, temp_home_dir):
        self.home_dir = Path(temp_home_dir)
        self.project_dir = self.home_dir / "test-project"
        self.project_dir.mkdir(exist_ok=True)
        self.repo_skills_dir = self.home_dir / ".skill-hub" / "repositories" / "main" / "skills"
        self.agents_skills_dir = self.project_dir / ".agents" / "skills"
        self.cmd = CommandRunner()

    def test_01_feedback_syncs_scripts_references_assets(self):
        """feedback 应将 scripts、references、assets 下的文件一并同步到仓库"""
        result = self.cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"init failed: {result.stderr}"

        skill_id = "e2e-feedback-subdirs-skill"
        result = self.cmd.run(
            "create", [skill_id], cwd=str(self.project_dir), input_text="\n"
        )
        assert result.success, f"create failed: {result.stderr}"

        skill_dir = self.agents_skills_dir / skill_id
        (skill_dir / "scripts" / "run.sh").parent.mkdir(parents=True, exist_ok=True)
        (skill_dir / "scripts" / "run.sh").write_text("#!/bin/bash\necho run\n")
        (skill_dir / "references" / "doc.md").parent.mkdir(parents=True, exist_ok=True)
        (skill_dir / "references" / "doc.md").write_text("# Ref\n")
        (skill_dir / "assets" / "icon.png").parent.mkdir(parents=True, exist_ok=True)
        (skill_dir / "assets" / "icon.png").write_bytes(b"\x89PNG\r\n\x1a\n")

        result = self.cmd.run("feedback", [skill_id], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"feedback failed: {result.stderr}"

        repo_skill_dir = self.repo_skills_dir / skill_id
        assert (repo_skill_dir / "scripts" / "run.sh").exists(), "scripts/run.sh 未同步"
        assert (repo_skill_dir / "references" / "doc.md").exists(), "references/doc.md 未同步"
        assert (repo_skill_dir / "assets" / "icon.png").exists(), "assets/icon.png 未同步"
