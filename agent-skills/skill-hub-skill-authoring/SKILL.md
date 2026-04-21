---
name: skill-hub-skill-authoring
description: "Primary workflow when creating or updating skills in an environment where skill-hub is installed, available, or used to manage skills. Use this before the generic skill-creator for skill-hub managed skills. Covers creating, repairing, validating, registering, importing, archiving, and otherwise managing reusable local skills with skill-hub; authoring Chinese-by-default SKILL.md files; declaring the matching formatter; maintaining .agents/skills content; syncing local project skill edits back to the default skill repository; and previewing remote push without publishing unless explicitly requested."
compatibility: "Designed for Claude Code, Cursor, OpenCode, and other AI coding assistants using skill-hub"
metadata:
  author: skill-hub Team
  tags: skill-hub,skill-authoring,skills,validation,feedback
  version: 1.0.2
---

# Skill Hub Skill Authoring

Use this skill when the task is to create or manage reusable skills themselves.

This is an authoring and maintenance workflow. If the user only wants to consume an existing managed skill in an application project, use `skill-hub-project-usage` instead.

## Scheduling Priority

When `skill-hub` is installed, available on `PATH`, or the current project/repository uses skill-hub conventions such as `.agents/skills`, this skill is the primary workflow for creating or updating skills.

Use the generic `skill-creator` only as secondary design guidance. If `skill-creator` and this skill differ, follow this skill for storage location, lifecycle commands, validation, feedback, default language, formatter declaration, and remote-push rules.

For skill-hub managed skills, always anchor creation on:

```bash
skill-hub create <skill-id>
skill-hub validate <skill-id> --links
skill-hub feedback <skill-id> --dry-run
```

## Authoring Model

- Local project skill content lives under `.agents/skills/<skill-id>/`.
- The required entry point is `.agents/skills/<skill-id>/SKILL.md`.
- The default skill repository under `~/.skill-hub/repositories/<default>/` is the archive source for reusable skills.
- `feedback` copies project-local skill content back to the local default skill repository.
- `push` publishes local repository changes to a remote and must only run after explicit user approval.
- Compatibility metadata is descriptive. Do not create target-specific branches or write `preferred_target`.

## Start Or Inspect A Skill Workspace

Check the current project state:

```bash
skill-hub status --json
```

Initialize the project if needed:

```bash
skill-hub init
```

Before creating a skill, check whether the ID already exists:

```bash
skill-hub list
skill-hub search <keyword>
```

Prefer stable lowercase IDs such as `go-runtime-patterns` or `skill-hub-project-usage`.

## Create A New Skill

Create the project-local skill directory:

```bash
skill-hub create <skill-id>
```

Then edit `.agents/skills/<skill-id>/SKILL.md`.

Keep `SKILL.md` focused:

- Frontmatter must include `name` and `description`.
- Make `description` clear enough to trigger the skill for the right task.
- New skills must use Chinese by default for human-facing definitions, headings, workflow instructions, variables, examples, and final guidance unless the user explicitly requests another language.
- Keep technical identifiers, command names, code symbols, API names, file paths, and upstream product names in their canonical form.
- Put only essential workflow instructions in the body.
- Use `references/` for detailed docs that should be loaded only when needed.
- Use `scripts/` only for deterministic repeated operations.
- Use `assets/` for reusable output files or templates.

If the skill needs user-provided values at `use` time, add `variables` in frontmatter. If not, leaving variables absent is valid and `use` will print that the skill has no configurable variables.

Every new skill must include a concise formatter section in the body:

```markdown
## Formatter

- Markdown/YAML: run `skill-hub validate <skill-id> --links` before feedback.
- Scripts/code: use the formatter already configured by the target project or repository. If none exists, state the expected formatter explicitly.
- Run formatting before `skill-hub feedback <skill-id> --force`.
```

Choose formatter commands by content:

- Markdown/YAML-only skills: use stable Markdown formatting plus `skill-hub validate <skill-id> --links`.
- Go scripts or examples: use `gofmt -w <files>`.
- Python scripts: use the repository formatter such as `ruff format <files>` or `black <files>`.
- JavaScript or TypeScript examples: use the repository script such as `npm run format` or the configured `prettier`.
- Shell scripts: use the repository formatter if present; otherwise keep POSIX/Bash style consistent and run the relevant shell syntax check when available.

Do not invent a formatter that the project cannot run. If the repository has no formatter for a file type, record the manual formatting expectation in the skill.

## Register Or Import Existing Skills

Register an existing project-local skill without overwriting files:

```bash
skill-hub register <skill-id>
```

Import skills from an existing directory when migrating content:

```bash
skill-hub import --path <dir>
```

Use `--skip-validate` only when intentionally staging invalid content for later repair.

## Validate And Repair

Validate frontmatter and local links:

```bash
skill-hub validate <skill-id>
skill-hub validate <skill-id> --links
```

Preview project status:

```bash
skill-hub status <skill-id>
skill-hub status <skill-id> --json
```

Use automatic frontmatter repair only when the user accepts file edits:

```bash
skill-hub validate <skill-id> --fix
```

Run path and duplicate checks when reorganizing skills:

```bash
skill-hub lint --paths --json
skill-hub dedupe --json
```

Repair duplicate non-canonical copies only after a dry run:

```bash
skill-hub sync-copies --canonical .agents/skills --scope . --dry-run
skill-hub sync-copies --canonical .agents/skills --scope .
```

## Archive To The Local Skill Repository

Preview the archive diff:

```bash
skill-hub feedback <skill-id> --dry-run
```

Archive the skill to the local default repository:

```bash
skill-hub feedback <skill-id> --force
```

For many skills:

```bash
skill-hub feedback --all --force --json
```

After feedback, confirm the project and repository copies are synced:

```bash
skill-hub status <skill-id> --json
```

## Publish Only When Explicitly Requested

Preview local repository changes:

```bash
skill-hub push --dry-run --json
```

Only publish when the user explicitly requests remote publication:

```bash
skill-hub push --message "update skills"
```

In `serve` mode, remote push requires the service-side `secretKey`:

```bash
SKILL_HUB_SERVICE_SECRET_KEY=<secretKey> skill-hub push
```

Do not treat `feedback`, `pull`, or `repo sync` as remote publication.

## Safety Rules

- Never run `push` automatically.
- Preserve existing user-authored skill content.
- Validate before feedback and after repair.
- Keep reusable skill content under `.agents/skills/<skill-id>/`.
- Do not write target-specific state or rely on compatibility filtering.
- If non-push commands return an old read-only serve error, update or restart the running `serve` instance instead of changing the workflow.
