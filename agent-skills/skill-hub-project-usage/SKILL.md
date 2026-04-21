---
name: skill-hub-project-usage
description: "Use when helping an application or business project consume skills already managed by skill-hub. Guides agents through initializing the project, syncing repositories, listing/searching managed skills before use, selecting a suitable skill only when one exists, enabling skills with use, applying them to .agents/skills, checking status, and feeding project-local improvements back to the local skill repository without pushing remotely unless explicitly requested."
compatibility: "Designed for Claude Code, Cursor, OpenCode, and other AI coding assistants using skill-hub"
metadata:
  author: skill-hub Team
  tags: skill-hub,project-usage,skills,apply,use
  version: 1.0.1
---

# Skill Hub Project Usage

Use this skill when the user wants to use existing managed skills inside another project.

This is a consumer workflow. If the task is to create or maintain the reusable skill itself, use `skill-hub-skill-authoring`.

## Project Usage Model

- The project workspace is `<project>/.agents/skills/`.
- `use` records selected skills in project state.
- `apply` copies enabled skills into the project workspace.
- `status` shows whether project skill copies match the repository versions.
- `feedback` can archive project-local improvements back to the local default skill repository.
- `push` is the only remote publication step and must be explicit.
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

Use dry-run if the user wants to preview file changes:

```bash
skill-hub apply --dry-run
```

## Selecting Skills

Before enabling anything, inspect available managed skills:

```bash
skill-hub list
skill-hub search <keyword>
```

Use `list` to see the available managed skill inventory. Use `search` with project, domain, language, framework, tool, or workflow keywords to narrow candidates.

Only run `skill-hub use <skill-id>` when a listed or searched skill clearly matches the current task. If no suitable skill exists, tell the user that no managed skill matched and continue without `use`; do not guess an unrelated skill ID.

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
skill-hub serve status
skill-hub serve stop <name>
skill-hub serve start <name>
```

For fresh installation, the latest installer restarts running registered `serve` instances:

```bash
curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh | bash
```

Manually started foreground `serve` processes must still be restarted manually.

## Safety Rules

- Never run `push` automatically.
- Do not treat `use`, `apply`, `feedback`, `pull`, or `repo sync` as remote publication.
- Keep project skill files in `.agents/skills/`.
- Preserve project edits and inspect `status` before overwriting or archiving.
- Do not write `preferred_target` or create target-specific workflows.
- Ask before `feedback --all --force` when many skills are modified.
