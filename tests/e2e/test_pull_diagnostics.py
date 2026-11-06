"""Offline pull diagnostics against isolated, real Git repositories."""

import json
import subprocess

import pytest

from tests.e2e.utils.command_runner import CommandRunner


def git(directory, *args, check=True):
    result = subprocess.run(
        ["git", "-C", str(directory), *args], capture_output=True, text=True,
        timeout=15, check=False,
    )
    if check:
        assert result.returncode == 0, result.stderr
    return result


def skill_content(body):
    return (
        "---\nname: demo\ndescription: Pull diagnostic fixture.\n"
        'metadata:\n  version: "1.0.0"\n  author: tester\n---\n' + body + "\n"
    )


@pytest.fixture
def pull_repo(tmp_path):
    remote, seed, project, hub = (tmp_path / name for name in ("remote.git", "seed", "project", "hub"))
    seed.mkdir()
    project.mkdir()
    git(tmp_path, "init", "--bare", "--initial-branch=main", str(remote))
    git(seed, "init", "--initial-branch=main")
    git(seed, "config", "user.name", "Tester")
    git(seed, "config", "user.email", "tester@example.com")
    skill = seed / "skills" / "demo" / "SKILL.md"
    skill.parent.mkdir(parents=True)
    skill.write_text(skill_content("base-content"), encoding="utf-8")
    (seed / "other.md").write_text("other base\n", encoding="utf-8")
    git(seed, "add", ".")
    git(seed, "commit", "-m", "initial")
    git(seed, "remote", "add", "origin", str(remote))
    git(seed, "push", "-u", "origin", "main")
    runner = CommandRunner()
    env = {"SKILL_HUB_HOME": str(hub), "SKILL_HUB_DISABLE_SERVICE_BRIDGE": "1"}

    def run(*args):
        return runner.run(args[0], list(args[1:]), cwd=str(project), env=env)

    initialized = run("init", str(remote))
    assert initialized.success, initialized
    local = hub / "repositories" / "remote"
    assert local.is_dir()
    git(local, "config", "user.name", "Tester")
    git(local, "config", "user.email", "tester@example.com")
    assert git(local, "status", "--porcelain").stdout == ""

    def remote_update():
        skill.write_text(skill_content("remote-content"), encoding="utf-8")
        git(seed, "add", ".")
        git(seed, "commit", "-m", "remote-only-commit")
        git(seed, "push")

    return run, local, remote_update


def test_pull_dirty_shows_staged_worktree_and_remote_content(pull_repo):
    run, local, remote_update = pull_repo
    remote_update()
    path = local / "skills" / "demo" / "SKILL.md"
    path.write_text(skill_content("staged-content"), encoding="utf-8")
    git(local, "add", "skills/demo/SKILL.md")
    path.write_text(skill_content("unstaged-content"), encoding="utf-8")
    (local / "notes with spaces.md").write_text("untracked\n", encoding="utf-8")
    head = git(local, "rev-parse", "HEAD").stdout
    before = git(local, "status", "--porcelain=v1", "-z").stdout
    result = run("pull")
    assert not result.success
    for text in (str(local), "skills/demo/SKILL.md", "notes with spaces.md", "staged-content", "unstaged-content", "remote-content", "潜在冲突", "未跟踪文件"):
        assert text in result.stderr, result
    assert "Usage:" not in result.stderr
    assert "Error:" not in result.stderr
    assert git(local, "rev-parse", "HEAD").stdout == head
    assert git(local, "status", "--porcelain=v1", "-z").stdout == before
    assert git(local, "stash", "list").stdout == ""


def test_pull_unrelated_dirty_json_and_argument_usage(pull_repo):
    run, local, remote_update = pull_repo
    remote_update()
    (local / "other.md").write_text("unrelated-local-content\n", encoding="utf-8")
    result = run("pull", "--json")
    assert not result.success
    data = json.loads(result.stdout)
    assert data["status"] == "failed"
    assert data["command_error"]["code"] == "GIT_WORKTREE_DIRTY"
    diagnostic = data["command_error"]["details"]["pull"]
    assert diagnostic["kind"] == "blocked_dirty"
    assert diagnostic["overlap_known"] is True
    assert diagnostic["behind"] == 1
    assert diagnostic["files"][0]["path"] == "other.md"
    assert diagnostic["files"][0]["overlaps_remote"] is False
    assert "unrelated-local-content" in diagnostic["files"][0]["unstaged_diff"]
    plain = run("pull")
    assert not plain.success
    assert "未发现同路径修改" in plain.stderr
    assert "Usage:" not in plain.stderr
    bad_argument = run("pull", "--unknown-option")
    assert not bad_argument.success
    assert "Usage:" in bad_argument.stderr


def test_pull_existing_conflict_shows_actual_blocks(pull_repo):
    run, local, remote_update = pull_repo
    path = local / "skills" / "demo" / "SKILL.md"
    path.write_text(skill_content("local-content"), encoding="utf-8")
    git(local, "add", ".")
    git(local, "commit", "-m", "local-only-commit")
    remote_update()
    git(local, "fetch")
    merge = git(local, "merge", "origin/main", check=False)
    assert merge.returncode != 0
    original = path.read_bytes()
    result = run("pull", "--force")
    assert not result.success
    for text in ("未解决", "双方修改", "<<<<<<<", "local-content", "remote-content"):
        assert text in result.stderr, result
    assert "Usage:" not in result.stderr
    assert path.read_bytes() == original
    assert git(local, "stash", "list").stdout == ""
    structured = run("pull", "--json")
    assert not structured.success
    failure = json.loads(structured.stdout)["command_error"]
    assert failure["code"] == "GIT_CONFLICT"
    assert failure["details"]["pull"]["files"][0]["conflict_type"] == "UU"


def test_pull_divergence_shows_commits(pull_repo):
    run, local, remote_update = pull_repo
    (local / "other.md").write_text("local commit\n", encoding="utf-8")
    git(local, "add", ".")
    git(local, "commit", "-m", "local-only-commit")
    remote_update()
    head = git(local, "rev-parse", "HEAD").stdout
    result = run("pull")
    assert not result.success
    for text in ("提交分叉", "本地领先：1", "远端领先：1", "local-only-commit", "remote-only-commit"):
        assert text in result.stderr, result
    assert "Usage:" not in result.stderr
    assert git(local, "rev-parse", "HEAD").stdout == head


def test_pull_force_json_identifies_retained_stash(pull_repo):
    run, local, remote_update = pull_repo
    remote_update()
    path = local / "skills" / "demo" / "SKILL.md"
    path.write_text(skill_content("my-uncommitted-content"), encoding="utf-8")
    result = run("pull", "--force", "--json")
    assert result.success, result
    data = json.loads(result.stdout)
    assert data["status"] == "synced"
    assert data["stash_ref"]
    assert "my-uncommitted-content" in git(local, "stash", "show", "-p", data["stash_ref"]).stdout
    assert "remote-content" in path.read_text(encoding="utf-8")
    assert "my-uncommitted-content" not in path.read_text(encoding="utf-8")
