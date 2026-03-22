"""
Service mode end-to-end tests.
"""

import os
import tempfile
import urllib.request
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner
from tests.e2e.utils.network_checker import NetworkChecker
from tests.e2e.utils.service_runner import ServiceRunner


@pytest.mark.no_debug
class TestServiceMode:
    def setup_method(self):
        self.cmd = CommandRunner()
        self.service_home = tempfile.mkdtemp(prefix="skill_hub_service_home_")
        self.client_home = tempfile.mkdtemp(prefix="skill_hub_client_home_")
        self.project_dir = Path(self.service_home) / "project"
        self.project_dir.mkdir(parents=True, exist_ok=True)
        self.consumer_project_dir = Path(self.service_home) / "consumer-project"
        self.consumer_project_dir.mkdir(parents=True, exist_ok=True)

        self.service_env = os.environ.copy()
        self.service_env["HOME"] = self.service_home
        self.client_env = os.environ.copy()
        self.client_env["HOME"] = self.client_home

    def teardown_method(self):
        import shutil

        shutil.rmtree(self.service_home, ignore_errors=True)
        shutil.rmtree(self.client_home, ignore_errors=True)

    def _prepare_service_skill(self, skill_name: str = "service-skill"):
        init_result = self.cmd.run("init", cwd=str(self.project_dir), env=self.service_env)
        assert init_result.success, init_result.stderr

        create_result = self.cmd.run("create", [skill_name], cwd=str(self.project_dir), env=self.service_env)
        assert create_result.success, create_result.stderr

        feedback_result = self.cmd.run(
            "feedback",
            [skill_name, "--force"],
            cwd=str(self.project_dir),
            env=self.service_env,
        )
        assert feedback_result.success, feedback_result.stderr

        repo_skill_dir = Path(self.service_home) / ".skill-hub" / "repositories" / "main" / "skills" / skill_name
        repo_skill_file = repo_skill_dir / "SKILL.md"
        assert repo_skill_file.exists()
        return repo_skill_dir, repo_skill_file

    def _start_service(self):
        try:
            return ServiceRunner(self.cmd.skill_hub_bin, self.service_env, str(self.project_dir)).start()
        except PermissionError:
            pytest.skip("localhost bind not permitted in current environment")
        except OSError as err:
            if "operation not permitted" in str(err).lower():
                pytest.skip("localhost bind not permitted in current environment")
            raise

    def test_service_health_ui_and_cli_bridge(self):
        self._prepare_service_skill()
        service = self._start_service()

        try:
            health = urllib.request.urlopen(f"{service.base_url}/api/v1/health", timeout=2).read().decode("utf-8")
            assert '"status":"ok"' in health

            ui = urllib.request.urlopen(f"{service.base_url}/", timeout=2).read().decode("utf-8")
            assert "Skill Hub" in ui
            assert "本地服务模式管理界面" in ui

            bridge_env = self.client_env.copy()
            bridge_env["SKILL_HUB_SERVICE_URL"] = service.base_url

            repo_list = self.cmd.run("repo", ["list"], cwd=str(self.project_dir), env=bridge_env)
            assert repo_list.success, repo_list.stderr
            assert "main" in repo_list.stdout

            skill_list = self.cmd.run("list", cwd=str(self.project_dir), env=bridge_env)
            assert skill_list.success, skill_list.stderr
            assert "service-skill" in skill_list.stdout

            status_result = self.cmd.run("status", cwd=str(self.project_dir), env=bridge_env)
            assert status_result.success, status_result.stderr
            assert "service-skill" in status_result.stdout
        finally:
            service.stop()

    def test_service_bridge_use_apply_feedback_flow(self):
        repo_skill_dir, repo_skill_file = self._prepare_service_skill()
        initial_repo_content = repo_skill_file.read_text(encoding="utf-8")
        consumer_init = self.cmd.run("init", cwd=str(self.consumer_project_dir), env=self.service_env)
        assert consumer_init.success, consumer_init.stderr
        service = self._start_service()

        try:
            bridge_env = self.client_env.copy()
            bridge_env["SKILL_HUB_SERVICE_URL"] = service.base_url

            use_result = self.cmd.run("use", ["service-skill"], cwd=str(self.consumer_project_dir), env=bridge_env)
            assert use_result.success, use_result.stderr
            assert "已成功标记为使用" in use_result.stdout

            apply_result = self.cmd.run("apply", cwd=str(self.consumer_project_dir), env=bridge_env)
            assert apply_result.success, apply_result.stderr
            assert "所有技能应用完成" in apply_result.stdout

            project_skill_dir = self.consumer_project_dir / ".agents" / "skills" / "service-skill"
            project_skill_file = project_skill_dir / "SKILL.md"
            assert project_skill_file.exists()

            extra_file = project_skill_dir / "notes.md"
            extra_file.write_text("service mode feedback\n", encoding="utf-8")
            project_skill_file.write_text(
                project_skill_file.read_text(encoding="utf-8") + "\n<!-- updated via service mode -->\n",
                encoding="utf-8",
            )

            feedback_result = self.cmd.run(
                "feedback",
                ["service-skill", "--force"],
                cwd=str(self.consumer_project_dir),
                env=bridge_env,
            )
            assert feedback_result.success, feedback_result.stderr
            assert "反馈完成" in feedback_result.stdout

            synced_extra_file = repo_skill_dir / "notes.md"
            assert synced_extra_file.exists()
            assert synced_extra_file.read_text(encoding="utf-8") == "service mode feedback\n"

            updated_repo_content = repo_skill_file.read_text(encoding="utf-8")
            assert "<!-- updated via service mode -->" in updated_repo_content
            assert updated_repo_content != initial_repo_content
        finally:
            service.stop()

    @pytest.mark.requires_network
    def test_service_bridge_search_prefers_service_when_available(self):
        if not NetworkChecker.is_network_available():
            pytest.skip("network required for remote search test")

        self._prepare_service_skill()
        service = self._start_service()

        try:
            bridge_env = self.client_env.copy()
            bridge_env["SKILL_HUB_SERVICE_URL"] = service.base_url

            search_result = self.cmd.run("search", ["git", "--limit", "5"], cwd=str(self.project_dir), env=bridge_env)
            assert search_result.success, search_result.stderr

            output = search_result.stdout + search_result.stderr
            assert "正在通过本地服务搜索远端技能" in output
            assert "搜索结果" in output or "未找到相关技能" in output
        finally:
            service.stop()
