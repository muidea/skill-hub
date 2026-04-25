---
name: skill-hub-project-usage
description: "Use when helping an application or business project consume skills already managed by skill-hub, or when enabling managed skills globally on this machine for Codex, OpenCode, or Claude. Guides agents through initializing the project when needed, syncing repositories, listing/searching managed skills before use, selecting a suitable skill only when one exists, enabling skills with use or use --global, applying them to .agents/skills or agent global skills directories, checking status, and feeding project-local improvements back to the local skill repository without pushing remotely unless explicitly requested."
compatibility: "Designed for Claude Code, Cursor, OpenCode, and other AI coding assistants using skill-hub"
metadata:
  author: skill-hub Team
  tags: skill-hub,project-usage,skills,apply,use
  version: 1.0.4
---

# Skill Hub Project Usage

Use this skill when the user wants to use existing managed skills inside another project.

This is a consumer workflow. If the task is to create or maintain the reusable skill itself, use `skill-hub-skill-authoring`.

## Project Usage Model

- The project workspace is `<project>/.agents/skills/`.
- `use` records selected skills in project state.
- `apply` copies enabled skills into the project workspace.
- `status` shows whether project skill copies match the repository versions.
- `use --global` records selected skills in `~/.skill-hub/global-state.json`.
- `apply --global` refreshes `~/.skill-hub/global/skills/` and configured agent global skills directories.
- `status --global` checks global desired state, source repository content, target agent directories, and `.skill-hub-manifest.json`.
- `remove --global` removes global desired state and only deletes Skill-Hub managed global skill directories unless `--force` is explicit.
- `feedback` can archive project-local improvements back to the local default skill repository.
- `push` is the only remote publication step and must be explicit.
- `upgrade` updates the installed skill-hub binary and bundled workflow skills; it does not sync skill repositories or publish local changes.
- Discovery comes before selection: run `list` and/or `search` before `use`, and only run `use` after confirming a suitable managed skill exists.

## Standard Flow

Run from the target project directory:

```bash
skill-hub --version
skill-hub status --json
```

Initialize when needed:

```bash
skill-hub init
```

Synchronize managed skill repositories before discovery:

```bash
skill-hub repo sync --json
```

Find the right skill:

```bash
skill-hub list
skill-hub search <keyword>
```

Enable and apply only after selecting a suitable skill from `list` or `search` output:

```bash
skill-hub use <skill-id>
skill-hub apply
skill-hub status
```

Refresh one enabled project skill when `status <skill-id>` shows `Outdated`:

```bash
skill-hub apply <skill-id>
skill-hub status <skill-id>
```

Use dry-run if the user wants to preview file changes:

```bash
skill-hub apply --dry-run
```

## Machine-Global Usage Flow

Use global mode when the user wants a managed skill available to an agent outside one project workspace.

Discover first:

```bash
skill-hub list
skill-hub search <keyword>
```

Enable globally only after selecting a suitable managed skill:

```bash
skill-hub use <skill-id> --global --agent codex
```

Inspect and preview before writing agent global directories:

```bash
skill-hub status --global
skill-hub status <skill-id> --global --agent codex
skill-hub apply --global --dry-run
```

Apply after preview:

```bash
skill-hub apply --global
```

Remove global usage when requested:

```bash
skill-hub remove <skill-id> --global --agent codex
```

Use `--agent codex`, `--agent opencode`, or `--agent claude` to scope global operations. If no agent is specified, skill-hub uses detected or configured agents. If `status --global <skill-id>` or `apply <skill-id> --global` reports `SKILL_NOT_FOUND`, the skill is not globally enabled for the requested agent; do not treat an empty result as success.

Do not use `--force` unless the user explicitly accepts overwriting a same-name global skill directory that is not managed by Skill-Hub. `--force` creates a backup before replacing conflicts.

## Selecting Skills

Before enabling anything, inspect available managed skills:

```bash
skill-hub list
skill-hub search <keyword>
```

Use `list` to see the available managed skill inventory. Use `search` with project, domain, language, framework, tool, or workflow keywords to narrow candidates.

Only run `skill-hub use <skill-id>` or `skill-hub use <skill-id> --global` when a listed or searched skill clearly matches the current task. If no suitable skill exists, tell the user that no managed skill matched and continue without `use`; do not guess an unrelated skill ID.

When multiple repositories contain the same skill ID, choose based on project intent and repository source. Ask the user when the right repository is ambiguous.

If `use` prints `该技能没有可配置的变量`, continue normally. It means the skill has no `variables` entries to prompt for.

Do not choose skills by `target`, `preferred_target`, or hard compatibility filtering. Compatibility text is descriptive metadata.

## Working With Applied Skills

Applied skill files live under:

```text
.agents/skills/<skill-id>/
```

After applying, inspect status:

```bash
skill-hub status <skill-id>
skill-hub status <skill-id> --json
```

If the user edits an applied skill in the project and wants to keep those improvements, preview and archive them:

```bash
skill-hub feedback <skill-id> --dry-run
skill-hub feedback <skill-id> --force
```

For all enabled skills:

```bash
skill-hub feedback --all --force --json
```

`feedback` writes to the local default skill repository only. It does not push to a remote.

## Repository Sync And Remote Push

Use these for remote-to-local synchronization:

```bash
skill-hub pull --check --json
skill-hub pull
skill-hub repo sync --json
```

Preview remote publication:

```bash
skill-hub push --dry-run --json
```

Only publish when the user explicitly asks:

```bash
skill-hub push --message "update skills"
```

In `serve` mode, no `secretKey` should be needed for `use`, `apply`, `feedback`, `pull`, or `repo sync`. `secretKey` is only required for remote push:

```bash
SKILL_HUB_SERVICE_SECRET_KEY=<secretKey> skill-hub push
```

## Troubleshooting

If a non-push command returns an old read-only error, the background `serve` process is probably outdated. Update or restart the running service; do not change the project workflow.

For registered services:

```bash
skill-hub upgrade --check
skill-hub upgrade --yes
skill-hub serve status
skill-hub serve stop <name>
skill-hub serve start <name>
```

For fresh installation, the latest installer restarts running registered `serve` instances:

```bash
curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh | bash
```

`skill-hub upgrade --yes` also refreshes release-bundled `skill-hub-*` workflow skills. Manually started foreground `serve` processes must still be restarted manually.

## Safety Rules

- Never run `push` automatically.
- Do not treat `use`, `apply`, `feedback`, `pull`, or `repo sync` as remote publication.
- Keep project skill files in `.agents/skills/`.
- Preserve project edits and inspect `status` before overwriting or archiving.
- Do not write `preferred_target` or create target-specific workflows.
- Ask before `feedback --all --force` when many skills are modified.
