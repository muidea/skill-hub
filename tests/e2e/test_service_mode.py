"""
Service mode end-to-end tests.
"""

import json
import http.client
import os
import random
import socket
import tempfile
import urllib.error
import urllib.parse
import urllib.request
from html.parser import HTMLParser
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner
from tests.e2e.utils.network_checker import NetworkChecker
from tests.e2e.utils.service_runner import ServiceRunner


class WebUIHTMLParser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.ids = set()
        self.forms = set()
        self.inputs = set()
        self.selects = set()
        self.buttons = []
        self.text = []
        self._in_button = False
        self._button_text = []

    def handle_starttag(self, tag, attrs):
        attr_map = dict(attrs)
        element_id = attr_map.get("id")
        if element_id:
            self.ids.add(element_id)
        if tag == "form" and element_id:
            self.forms.add(element_id)
        if tag == "input" and element_id:
            self.inputs.add(element_id)
        if tag == "select" and element_id:
            self.selects.add(element_id)
        if tag == "button":
            self._in_button = True
            self._button_text = []

    def handle_endtag(self, tag):
        if tag == "button" and self._in_button:
            label = "".join(self._button_text).strip()
            if label:
                self.buttons.append(label)
            self._in_button = False
            self._button_text = []

    def handle_data(self, data):
        value = data.strip()
        if value:
            self.text.append(value)
        if self._in_button:
            self._button_text.append(data)


def parse_webui_html(content: str) -> WebUIHTMLParser:
    parser = WebUIHTMLParser()
    parser.feed(content)
    return parser


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
        self.service_env["SKILL_HUB_DISABLE_SERVICE_BRIDGE"] = "1"
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

    def _start_service(self, secret_key: str = ""):
        try:
            return ServiceRunner(self.cmd.skill_hub_bin, self.service_env, str(self.project_dir), secret_key=secret_key).start()
        except PermissionError:
            pytest.skip("localhost bind not permitted in current environment")
        except OSError as err:
            if "operation not permitted" in str(err).lower():
                pytest.skip("localhost bind not permitted in current environment")
            raise

    def _reserve_port(self) -> int:
        for _ in range(50):
            port = random.randint(20000, 60999)
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
                sock.settimeout(0.1)
                if sock.connect_ex(("127.0.0.1", port)) != 0:
                    return port
        raise OSError("unable to find available localhost port")

    def test_service_health_ui_and_cli_bridge(self):
        self._prepare_service_skill()
        service = self._start_service()

        try:
            health_resp = urllib.request.urlopen(f"{service.base_url}/api/v1/health", timeout=2)
            assert health_resp.headers.get("X-Content-Type-Options") == "nosniff"
            assert health_resp.headers.get("X-Frame-Options") == "DENY"
            health = health_resp.read().decode("utf-8")
            assert '"status":"ok"' in health

            parsed_url = urllib.parse.urlparse(service.base_url)
            bad_host_conn = http.client.HTTPConnection(parsed_url.hostname, parsed_url.port, timeout=2)
            try:
                bad_host_conn.request("GET", "/api/v1/health", headers={"Host": "example.com"})
                bad_host_resp = bad_host_conn.getresponse()
                assert bad_host_resp.status == 403
            finally:
                bad_host_conn.close()

            cross_site_conn = http.client.HTTPConnection(parsed_url.hostname, parsed_url.port, timeout=2)
            try:
                cross_site_conn.request(
                    "POST",
                    "/api/v1/skill-repository/sync",
                    headers={"Origin": "https://example.com"},
                )
                cross_site_resp = cross_site_conn.getresponse()
                assert cross_site_resp.status == 403
            finally:
                cross_site_conn.close()

            ui = urllib.request.urlopen(f"{service.base_url}/", timeout=2).read().decode("utf-8")
            assert "Skill Hub" in ui
            assert "技能目录" in ui

            admin_ui = urllib.request.urlopen(f"{service.base_url}/admin.html", timeout=2).read().decode("utf-8")
            assert "默认仓库状态" in admin_ui
            assert "/api/v1/skill-repository/sync-check" in admin_ui
            assert "/api/v1/skill-repository/push-preview" in admin_ui
            assert "expected_changed_files" in admin_ui
            assert "default-repo-push-confirm" in admin_ui
            assert "写入密钥" not in admin_ui
            assert "skillHubSecretKey" not in admin_ui

            push_preview = urllib.request.urlopen(
                f"{service.base_url}/api/v1/skill-repository/push-preview",
                timeout=2,
            ).read().decode("utf-8")
            assert '"has_changes":true' in push_preview

            push_without_confirm_req = urllib.request.Request(
                f"{service.base_url}/api/v1/skill-repository/push",
                data=b'{"message":"test"}',
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with pytest.raises(urllib.error.HTTPError) as push_err:
                urllib.request.urlopen(push_without_confirm_req, timeout=2)
            assert push_err.value.code == 403
            assert "READ_ONLY" in push_err.value.read().decode("utf-8")

            bridge_env = self.client_env.copy()
            bridge_env["SKILL_HUB_SERVICE_URL"] = service.base_url

            repo_list = self.cmd.run("repo", ["list"], cwd=str(self.project_dir), env=bridge_env)
            assert repo_list.success, repo_list.stderr
            assert "main" in repo_list.stdout

            repo_list_json = self.cmd.run("repo", ["list", "--json"], cwd=str(self.project_dir), env=bridge_env)
            assert repo_list_json.success, repo_list_json.stderr
            assert '"default_repo": "main"' in repo_list_json.stdout

            pull_check_json = self.cmd.run("pull", ["--check", "--json"], cwd=str(self.project_dir), env=bridge_env)
            assert pull_check_json.success, pull_check_json.stderr
            pull_data = json.loads(pull_check_json.stdout)
            assert pull_data["check"] is True
            assert pull_data["status"] in {"no_remote", "up_to_date", "updates_available", "ahead", "divergent"}

            skill_list = self.cmd.run("list", cwd=str(self.project_dir), env=bridge_env)
            assert skill_list.success, skill_list.stderr
            assert "service-skill" in skill_list.stdout

            status_result = self.cmd.run("status", cwd=str(self.project_dir), env=bridge_env)
            assert status_result.success, status_result.stderr
            assert "service-skill" in status_result.stdout

            repo_sync_read_only = self.cmd.run("repo", ["sync", "main"], cwd=str(self.project_dir), env=bridge_env)
            assert not repo_sync_read_only.success
            assert "READ_ONLY" in repo_sync_read_only.stderr
            assert "SYSTEM_ERROR" not in repo_sync_read_only.stderr

            git_status_json = self.cmd.run("git", ["status", "--json"], cwd=str(self.project_dir), env=bridge_env)
            assert git_status_json.success, git_status_json.stderr
            git_status_data = json.loads(git_status_json.stdout)
            assert git_status_data["state"] in {"clean", "dirty", "not_initialized"}
            assert "raw_status" in git_status_data

            git_sync_json = self.cmd.run("git", ["sync", "--json"], cwd=str(self.project_dir), env=bridge_env)
            assert not git_sync_json.success
            git_sync_data = json.loads(git_sync_json.stdout)
            assert git_sync_data["command"] == "sync"
            assert git_sync_data["status"] == "failed"
        finally:
            service.stop()

    def test_service_write_requires_secret_key_when_configured(self):
        self._prepare_service_skill()
        service = self._start_service(secret_key="write-secret")

        try:
            health = urllib.request.urlopen(f"{service.base_url}/api/v1/health", timeout=2).read().decode("utf-8")
            assert '"status":"ok"' in health

            req = urllib.request.Request(
                f"{service.base_url}/api/v1/skill-repository/push",
                data=b'{"message":"test"}',
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with pytest.raises(urllib.error.HTTPError) as unauthorized_err:
                urllib.request.urlopen(req, timeout=2)
            assert unauthorized_err.value.code == 401

            authed_req = urllib.request.Request(
                f"{service.base_url}/api/v1/skill-repository/push",
                data=b'{"message":"test"}',
                headers={
                    "Content-Type": "application/json",
                    "X-Skill-Hub-Secret-Key": "write-secret",
                },
                method="POST",
            )
            with pytest.raises(urllib.error.HTTPError) as validation_err:
                urllib.request.urlopen(authed_req, timeout=2)
            assert validation_err.value.code == 400

            bridge_env = self.client_env.copy()
            bridge_env["SKILL_HUB_SERVICE_URL"] = service.base_url
            bridge_env["SKILL_HUB_SERVICE_SECRET_KEY"] = "write-secret"
            git_sync_json = self.cmd.run("git", ["sync", "--json"], cwd=str(self.project_dir), env=bridge_env)
            assert not git_sync_json.success
            git_sync_data = json.loads(git_sync_json.stdout)
            assert git_sync_data["command"] == "sync"
            assert git_sync_data["status"] == "failed"
        finally:
            service.stop()

    def test_webui_pages_expose_catalog_admin_and_read_only_write_controls(self):
        self._prepare_service_skill()
        service = self._start_service()

        try:
            index_resp = urllib.request.urlopen(f"{service.base_url}/", timeout=2)
            assert index_resp.headers.get("X-Content-Type-Options") == "nosniff"
            index_html = index_resp.read().decode("utf-8")
            index_doc = parse_webui_html(index_html)

            assert {"refresh-skills", "repo-filter", "skills", "skills-pagination"} <= index_doc.ids
            assert {"repo-filter"} <= index_doc.inputs
            assert "target-filter" not in index_doc.ids
            assert "target-filter" not in index_doc.selects
            assert 'name="target"' not in index_html
            assert 'compatibility || "通用"' not in index_html
            assert "刷新技能目录" in index_doc.buttons
            assert "进入管理端" in index_doc.buttons
            assert "/api/v1/skills" in index_html

            admin_resp = urllib.request.urlopen(f"{service.base_url}/admin.html", timeout=2)
            assert admin_resp.headers.get("X-Frame-Options") == "DENY"
            admin_html = admin_resp.read().decode("utf-8")
            admin_doc = parse_webui_html(admin_html)

            assert {
                "refresh-admin",
                "repo-form",
                "repo-name",
                "repo-url",
                "repos",
                "projects",
                "default-repo-panel",
                "project-filter",
            } <= admin_doc.ids
            assert {"repo-form"} <= admin_doc.forms
            assert {"repo-name", "repo-url", "project-filter"} <= admin_doc.inputs
            assert "project-target-filter" not in admin_doc.ids
            assert "project-target-filter" not in admin_doc.selects
            assert "目标过滤" not in admin_html
            assert 'name="target"' not in admin_html
            assert "preferred_target" not in admin_html
            assert "item.target" not in admin_html
            assert "刷新管理视图" in admin_doc.buttons
            assert "添加仓库" in admin_doc.buttons
            assert "/api/v1/repos" in admin_html
            assert "/api/v1/projects" in admin_html
            assert "/api/v1/project-apply" in admin_html
            assert "/api/v1/project-skills/use" in admin_html
            assert "/api/v1/project-feedback/apply" in admin_html
            assert "X-Skill-Hub-Secret-Key" not in admin_html
            assert "skillHubSecretKey" not in admin_html
            assert "写入密钥" not in admin_doc.buttons

            skills_payload = json.loads(
                urllib.request.urlopen(f"{service.base_url}/api/v1/skills", timeout=2).read().decode("utf-8")
            )
            assert skills_payload["code"] == "OK"
            assert skills_payload["data"]["total"] == len(skills_payload["data"]["items"])
            assert any(item["id"] == "service-skill" for item in skills_payload["data"]["items"])

            projects_payload = json.loads(
                urllib.request.urlopen(f"{service.base_url}/api/v1/projects", timeout=2).read().decode("utf-8")
            )
            assert projects_payload["code"] == "OK"
            assert any(item["project_path"] == str(self.project_dir) for item in projects_payload["data"]["items"])
        finally:
            service.stop()

    def test_named_service_instance_management(self):
        try:
            port = self._reserve_port()
        except PermissionError:
            pytest.skip("localhost bind not permitted in current environment")
        except OSError as err:
            if "operation not permitted" in str(err).lower():
                pytest.skip("localhost bind not permitted in current environment")
            raise

        register_result = self.cmd.run(
            "serve",
            ["register", "managed", "--host", "127.0.0.1", "--port", str(port), "--secret-key", "write-secret"],
            cwd=str(self.project_dir),
            env=self.service_env,
        )
        assert register_result.success, register_result.stderr
        assert "写权限: secret-key" in register_result.stdout

        start_result = self.cmd.run(
            "serve",
            ["start", "managed"],
            cwd=str(self.project_dir),
            env=self.service_env,
        )
        assert start_result.success, start_result.stderr
        assert "已启动" in start_result.stdout

        try:
            status_result = self.cmd.run(
                "serve",
                ["status", "managed"],
                cwd=str(self.project_dir),
                env=self.service_env,
            )
            assert status_result.success, status_result.stderr
            assert "managed\trunning" in status_result.stdout
            assert f"http://127.0.0.1:{port}" in status_result.stdout
            assert "write=secret-key" in status_result.stdout

            health = urllib.request.urlopen(f"http://127.0.0.1:{port}/api/v1/health", timeout=2).read().decode("utf-8")
            assert '"status":"ok"' in health
        finally:
            stop_result = self.cmd.run(
                "serve",
                ["stop", "managed"],
                cwd=str(self.project_dir),
                env=self.service_env,
            )
            assert stop_result.success, stop_result.stderr

        stopped_status = self.cmd.run(
            "serve",
            ["status", "managed"],
            cwd=str(self.project_dir),
            env=self.service_env,
        )
        assert stopped_status.success, stopped_status.stderr
        assert "managed\tstopped" in stopped_status.stdout

        remove_result = self.cmd.run(
            "serve",
            ["remove", "managed"],
            cwd=str(self.project_dir),
            env=self.service_env,
        )
        assert remove_result.success, remove_result.stderr

        final_status = self.cmd.run(
            "serve",
            ["status"],
            cwd=str(self.project_dir),
            env=self.service_env,
        )
        assert final_status.success, final_status.stderr
        assert "managed" not in final_status.stdout

    def test_service_bridge_use_apply_feedback_flow(self):
        repo_skill_dir, repo_skill_file = self._prepare_service_skill()
        initial_repo_content = repo_skill_file.read_text(encoding="utf-8")
        consumer_init = self.cmd.run("init", cwd=str(self.consumer_project_dir), env=self.service_env)
        assert consumer_init.success, consumer_init.stderr
        service = self._start_service(secret_key="write-secret")

        try:
            bridge_env = self.client_env.copy()
            bridge_env["SKILL_HUB_SERVICE_URL"] = service.base_url
            bridge_env["SKILL_HUB_SERVICE_SECRET_KEY"] = "write-secret"

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

            feedback_all_result = self.cmd.run(
                "feedback",
                ["--all", "--force", "--json"],
                cwd=str(self.consumer_project_dir),
                env=bridge_env,
            )
            assert feedback_all_result.success, feedback_all_result.stderr
            assert '"total": 1' in feedback_all_result.stdout
            assert '"failed": 0' in feedback_all_result.stdout

            push_preview = self.cmd.run(
                "push",
                ["--dry-run", "--json"],
                cwd=str(self.consumer_project_dir),
                env=bridge_env,
            )
            assert push_preview.success, push_preview.stderr
            push_data = json.loads(push_preview.stdout)
            assert push_data["status"] == "planned"
            assert push_data["has_changes"] is True
            assert any("service-skill" in item for item in push_data["changed_files"])
        finally:
            service.stop()

    def test_service_bridge_register_and_import_flow(self):
        init_result = self.cmd.run("init", cwd=str(self.project_dir), env=self.service_env)
        assert init_result.success, init_result.stderr

        skills_dir = self.project_dir / ".agents" / "skills"
        skills_dir.mkdir(parents=True, exist_ok=True)
        docs_dir = self.project_dir / "docs"
        docs_dir.mkdir(parents=True, exist_ok=True)
        (docs_dir / "service.md").write_text("# Service Doc\n", encoding="utf-8")
        manual_dir = skills_dir / "service-register"
        manual_dir.mkdir(parents=True, exist_ok=True)
        (manual_dir / "SKILL.md").write_text(
            """---
name: service-register
description: Service bridge register skill.
compatibility: Compatible with open_code
metadata:
  version: "1.0.0"
  author: "tester"
---
# Service Register

Use /home/tester/workspace/docs/service.md during setup.
Read [service doc](docs/service.md).
""",
            encoding="utf-8",
        )
        imported_dir = skills_dir / "service-import"
        imported_dir.mkdir(parents=True, exist_ok=True)
        (imported_dir / "SKILL.md").write_text(
            "# Service Import\n\nLegacy service import body.\n",
            encoding="utf-8",
        )

        service = self._start_service(secret_key="write-secret")
        try:
            bridge_env = self.client_env.copy()
            bridge_env["SKILL_HUB_SERVICE_URL"] = service.base_url
            bridge_env["SKILL_HUB_SERVICE_SECRET_KEY"] = "write-secret"

            lint_result = self.cmd.run(
                "lint",
                [".", "--paths", "--project-root", "/home/tester/workspace", "--fix"],
                cwd=str(self.project_dir),
                env=bridge_env,
            )
            assert lint_result.success, lint_result.stderr
            assert "rewritten:     1" in lint_result.stdout
            assert "docs/service.md" in (manual_dir / "SKILL.md").read_text(encoding="utf-8")

            register_result = self.cmd.run(
                "register",
                ["service-register"],
                cwd=str(self.project_dir),
                env=bridge_env,
            )
            assert register_result.success, register_result.stderr
            assert "已登记到项目状态" in register_result.stdout

            validate_result = self.cmd.run(
                "validate",
                ["service-register", "--links", "--json"],
                cwd=str(self.project_dir),
                env=bridge_env,
            )
            assert validate_result.success, validate_result.stderr
            assert '"link_issue_count": 0' in validate_result.stdout

            audit_path = self.project_dir / ".agents" / "service-audit.md"
            audit_result = self.cmd.run(
                "audit",
                [".agents/skills", "--output", str(audit_path)],
                cwd=str(self.project_dir),
                env=bridge_env,
            )
            assert audit_result.success, audit_result.stderr
            assert audit_path.exists()
            assert "Skill Hub Audit Report" in audit_path.read_text(encoding="utf-8")

            import_result = self.cmd.run(
                "import",
                [".agents/skills", "--fix-frontmatter", "--archive", "--force"],
                cwd=str(self.project_dir),
                env=bridge_env,
            )
            assert import_result.success, import_result.stderr
            assert "discovered: 2" in import_result.stdout
            assert "failed:     0" in import_result.stdout

            repo_skills = Path(self.service_home) / ".skill-hub" / "repositories" / "main" / "skills"
            assert (repo_skills / "service-register" / "SKILL.md").exists()
            imported_skill = repo_skills / "service-import" / "SKILL.md"
            assert imported_skill.exists()
            assert "name: service-import" in imported_skill.read_text(encoding="utf-8")

            status_result = self.cmd.run("status", ["--json"], cwd=str(self.project_dir), env=bridge_env)
            assert status_result.success, status_result.stderr
            assert "service-register" in status_result.stdout
            assert "service-import" in status_result.stdout
        finally:
            service.stop()

    def test_service_bridge_dedupe_and_sync_copies_flow(self):
        init_result = self.cmd.run("init", cwd=str(self.project_dir), env=self.service_env)
        assert init_result.success, init_result.stderr

        root_skill_dir = self.project_dir / ".agents" / "skills" / "service-dup"
        child_skill_dir = self.project_dir / "child" / ".agents" / "skills" / "service-dup"
        root_skill_dir.mkdir(parents=True, exist_ok=True)
        child_skill_dir.mkdir(parents=True, exist_ok=True)
        root_content = """---
name: service-dup
description: Canonical service duplicate skill.
compatibility: Compatible with open_code
metadata:
  version: "1.0.0"
  author: "tester"
---
# service-dup

canonical service body
"""
        child_content = root_content.replace("canonical service body", "child service body")
        (root_skill_dir / "SKILL.md").write_text(root_content, encoding="utf-8")
        child_skill_file = child_skill_dir / "SKILL.md"
        child_skill_file.write_text(child_content, encoding="utf-8")

        service = self._start_service(secret_key="write-secret")
        try:
            bridge_env = self.client_env.copy()
            bridge_env["SKILL_HUB_SERVICE_URL"] = service.base_url
            bridge_env["SKILL_HUB_SERVICE_SECRET_KEY"] = "write-secret"

            dedupe_result = self.cmd.run(
                "dedupe",
                [".", "--canonical", ".agents/skills", "--json"],
                cwd=str(self.project_dir),
                env=bridge_env,
            )
            assert dedupe_result.success, dedupe_result.stderr
            assert '"conflicts": 1' in dedupe_result.stdout

            sync_result = self.cmd.run(
                "sync-copies",
                ["--canonical", ".agents/skills", "--scope", "."],
                cwd=str(self.project_dir),
                env=bridge_env,
            )
            assert sync_result.success, sync_result.stderr
            assert "synced:    1" in sync_result.stdout
            assert child_skill_file.read_text(encoding="utf-8") == root_content
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
            assert "搜索结果" in output or "未找到相关技能" in output or "本地服务搜索失败" in output
        finally:
            service.stop()
