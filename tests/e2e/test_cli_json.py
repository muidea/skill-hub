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
