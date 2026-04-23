# v0.8.0 Global Skill Management Release Notes

Release date: 2026-04-23

## Summary

`v0.8.0` adds machine-global skill management for Skill-Hub managed skills. The feature lets users enable, inspect, apply, and remove managed skills for agent global skill directories without binding the operation to a project workspace.

The feature commit is:

```text
5949a63 feat: add global skill management
```

The post-release lint cleanup commit is:

```text
b54329d test: remove unused global test helper
```

## User-Facing Scope

- Adds `skill-hub use <id> --global [--agent codex|opencode|claude]`.
- Adds `skill-hub status [id] --global [--agent codex|opencode|claude] [--json]`.
- Adds `skill-hub apply [id] --global [--agent codex|opencode|claude] [--dry-run] [--force]`.
- Adds `skill-hub remove <id> --global [--agent codex|opencode|claude] [--force]`.
- Stores desired global state in `~/.skill-hub/global-state.json`.
- Maintains a Skill-Hub global mirror under `~/.skill-hub/global/skills/<id>`.
- Applies managed skills to Codex, OpenCode, and Claude global skills directories.
- Writes `.skill-hub-manifest.json` into managed agent skill directories.
- Reports conflicts for same-name unmanaged directories, with explicit `--force` backup-and-replace behavior.
- Returns explicit `SKILL_NOT_FOUND` for a requested global skill or agent mapping that is not enabled.

## Implementation Scope

- Adds the global kernel service package under `internal/modules/kernel/global/service`.
- Extends CLI handling for `use`, `status`, `apply`, and `remove`.
- Extends shell completion for `--global`.
- Extends the service bridge, HTTP API types/routes, and hub client.
- Keeps project-local skill usage and machine-global skill usage separate.
- Preserves the existing rule that remote publication only happens through explicit `push`.

## Documentation And Skill Updates

- Updates `README.md`, `INSTALLATION.md`, and `docs/Skill-Hub命令规范.md`.
- Updates `agent-skills/skill-hub-workflow/SKILL.md` to route machine-global skill usage.
- Updates `agent-skills/skill-hub-project-usage/SKILL.md` with the global usage flow.
- Leaves `agent-skills/skill-hub-skill-authoring/SKILL.md` unchanged because `--global` is a consumption workflow, not an authoring workflow.

## Validation

The release scope has been validated with:

```bash
make lint
GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod-cache go test ./... --count=1
GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod-cache go build -o bin/skill-hub ./application/skill-hub/cmd
SKILL_HUB_BIN=/home/rangh/aispace/skill/skill-hub/bin/skill-hub /home/rangh/codespace/venv/bin/pytest -q -p no:rerunfailures tests/e2e
```

Latest observed e2e result:

```text
109 passed, 9 skipped
```

## Release Notes Source

`dist/release-notes-v0.8.0.md` is generated release output and is ignored by Git. This tracked document is the durable repository-side release note for the `v0.8.0` global skill management scope and the follow-up lint cleanup.
