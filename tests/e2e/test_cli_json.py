import json
from pathlib import Path

import pytest

from tests.e2e.utils.command_runner import CommandRunner


@pytest.fixture
def json_cli(tmp_path):
    project = tmp_path / "project"
    project.mkdir()
    hub = tmp_path / "hub"
    env = {
        "SKILL_HUB_HOME": str(hub),
        "SKILL_HUB_DISABLE_SERVICE_BRIDGE": "1",
    }
    runner = CommandRunner()

    def run(command, *args):
        return runner.run(command, list(args), cwd=str(project), env=env)

    initialized = run("init", "--json")
    assert initialized.success, initialized
    assert json.loads(initialized.stdout)["already_initialized"] is False
    return run, project, hub


def write_skill(project: Path, skill_id: str):
    path = project / ".agents" / "skills" / skill_id / "SKILL.md"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        f"---\nname: {skill_id}\ndescription: Test JSON CLI output.\n"
        'metadata:\n  version: "1.0.0"\n  author: tester\n---\n# Test\n',
        encoding="utf-8",
    )
    return path


def test_first_register_stdout_is_one_json_document(json_cli):
    run, project, hub = json_cli
    path = write_skill(project, "manual-skill")
    original = path.read_bytes()

    first = run("register", "manual-skill", "--json")
    assert first.success, first
    assert json.loads(first.stdout)["registered"] is True
    assert "已创建项目工作区" in first.stderr
    assert "[Y/n]" not in first.stderr

    second = run("register", "manual-skill", "--json")
    assert second.success, second
    assert json.loads(second.stdout)["registered"] is False
    assert path.read_bytes() == original

    state = json.loads((hub / "state.json").read_text(encoding="utf-8"))
    assert "manual-skill" in state[str(project)]["skills"]


def test_json_argument_error_is_reported_once(json_cli):
    run, _, _ = json_cli
    result = run("register", "--json")
    assert not result.success
    error = json.loads(result.stdout)["command_error"]
    assert error["code"] == "INVALID_INPUT"
    assert "Error:" not in result.stderr
    assert "Usage:" not in result.stderr


def test_json_import_keeps_partial_results(json_cli):
    run, project, _ = json_cli
    write_skill(project, "valid-skill")
    invalid = write_skill(project, "invalid-skill")
    invalid.write_text("# Missing frontmatter\n", encoding="utf-8")

    result = run("import", ".agents/skills", "--json")
    assert not result.success
    data = json.loads(result.stdout)
    assert data["registered"] == 1
    assert data["failed"] == 1
    assert data["command_error"]["code"] == "VALIDATION_FAILED"
    assert data["failures"][0]["skill_id"] == "invalid-skill"


def test_json_status_and_import_preview_do_not_register_project(json_cli):
    run, project, hub = json_cli
    state_path = hub / "state.json"
    original = state_path.read_bytes()

    status = run("status", "--json")
    assert not status.success
    assert json.loads(status.stdout)["command_error"]["code"] == "PROJECT_NOT_FOUND"
    assert not (project / ".agents").exists()

    write_skill(project, "preview-skill")
    preview = run("import", ".agents/skills", "--dry-run", "--json")
    assert preview.success, preview
    assert json.loads(preview.stdout)["discovered"] == 1
    assert state_path.read_bytes() == original


def test_empty_project_registration_replays_original_request_receipt(json_cli):
    run, project, hub = json_cli
    args = ("register", "--request-id", "lost-register-response", "--json")
    first = run("project", *args)
    assert first.success, first
    receipt = json.loads(first.stdout)
    assert receipt["status"] == "succeeded"
    assert receipt["executed"] is True
    assert receipt["finished"] is True
    assert receipt["project_path"] == str(project)
    assert receipt["result"]["project_created"] is True
    assert receipt["result"]["skill_count"] == 0
    state_path = hub / "state.json"
    state = json.loads(state_path.read_text(encoding="utf-8"))
    assert state[str(project)]["skills"] == {}

    # Current state is intentionally different from the original effect.
    write_skill(project, "added-after-request")
    registered = run("register", "added-after-request", "--json")
    assert registered.success, registered
    state_before = state_path.read_bytes()
    for extra in ((), ("--query",)):
        replay = run("project", *args, *extra)
        assert replay.success, replay
        assert json.loads(replay.stdout) == receipt
        assert state_path.read_bytes() == state_before

    # Another request may register an existing project, but cannot erase skills.
    existing = run("project", "register", "--request-id", "another-request", "--json")
    assert existing.success, existing
    effects = json.loads(existing.stdout)["result"]
    assert effects["project_created"] is False
    assert effects["skill_count"] == 1
    assert effects["created_directories"] == []
    assert state_path.read_bytes() == state_before

    other = project.parent / "other-project"
    other.mkdir()
    conflict = run("project", "register", str(other), "--request-id", "lost-register-response", "--json")
    assert not conflict.success
    assert json.loads(conflict.stdout)["command_error"]["code"] == "REQUEST_CONFLICT"
    assert not (other / ".agents").exists()

    # Historical queries must survive removal of both live state and config.
    state_path.unlink()
    (hub / "config.yaml").unlink()
    queried = run("project", *args, "--query")
    assert queried.success, queried
    assert json.loads(queried.stdout) == receipt
    assert not state_path.exists()


def test_empty_project_registration_missing_receipt_and_failed_receipt(json_cli):
    run, project, hub = json_cli
    before = (hub / "state.json").read_bytes()
    missing = run("project", "register", "--query", "--request-id", "never-submitted", "--json")
    assert not missing.success
    assert json.loads(missing.stdout)["command_error"]["code"] == "REQUEST_NOT_FOUND"
    assert not (hub / "project-registrations").exists()
    assert not (project / ".agents").exists()
    assert (hub / "state.json").read_bytes() == before

    blocked = project / ".agents"
    blocked.write_text("workspace blocked", encoding="utf-8")
    args = ("register", "--request-id", "failed-register", "--json")
    failed = run("project", *args)
    assert not failed.success
    receipt = json.loads(failed.stdout)
    assert receipt["status"] == "failed"
    assert receipt["finished"] is True
    assert receipt["result"]["project_created"] is False
    blocked.unlink()
    replay = run("project", *args)
    assert not replay.success
    assert json.loads(replay.stdout) == receipt
    queried = run("project", *args, "--query")
    assert queried.success, queried
    assert json.loads(queried.stdout) == receipt
    assert not blocked.exists()
