---
name: skill-hub-workflow
description: "Use when deciding which skill-hub workflow applies or when the user asks broadly how to use skill-hub for skill management. When skill-hub is installed, available, or managing the project, route skill creation/update work to skill-hub-skill-authoring before generic skill-creator. Routes between two business scenarios: creating/managing reusable local skills and consuming already managed skills inside other projects. Prefer skill-hub-skill-authoring for authoring work and skill-hub-project-usage for application-project consumption work."
compatibility: "Designed for Claude Code, Cursor, OpenCode, and other AI coding assistants using skill-hub"
metadata:
  author: skill-hub Team
  tags: skill-hub,skills,workflow,router
  version: 1.1.2
---

# Skill Hub Workflow Router

Use this skill when the user asks generally about using `skill-hub`, or when it is unclear which skill-hub business scenario applies.

When `skill-hub` is installed or the current project uses `.agents/skills`, route skill creation and skill update requests to `skill-hub-skill-authoring` before considering the generic `skill-creator`. The generic skill-creator can provide content-design principles, but skill-hub lifecycle rules control storage, validation, archiving, and publication.

There are two primary business scenarios:

1. Creating and managing reusable local skills.
2. Using already managed skills inside another project.

Prefer the dedicated scenario skill whenever the task is clear.

## Scenario Selection

Use `skill-hub-skill-authoring` when the user wants to:

- create a new reusable skill
- edit or repair `SKILL.md`
- register or import existing skill directories
- validate frontmatter or local links
- archive project-local skill content back to the local default skill repository
- maintain duplicate skill copies, paths, or skill repository content

Use `skill-hub-project-usage` when the user wants to:

- initialize a business/application project for skill-hub
- discover managed skills
- list/search available managed skills before choosing one
- enable a skill with `use`
- apply enabled skills into `.agents/skills`
- inspect project skill status
- feed project-local improvements back to the local skill repository
- synchronize repositories before consuming skills

## Shared Rules

- `use` records a skill in project state; it does not copy files.
- `apply` copies enabled skills into `.agents/skills/`.
- `feedback` writes project-local skill changes to the local default skill repository.
- `push` is the explicit remote publication step. Never run it automatically.
- `pull` and `repo sync` synchronize remote repositories into local repositories; they are not remote publication.
- In `serve` mode, `secretKey` is only required for remote push.
- `target` and compatibility metadata are descriptive; do not branch business logic by target.

## Minimal Command Map

For authoring reusable skills:

```bash
skill-hub create <skill-id>
skill-hub validate <skill-id> --links
skill-hub feedback <skill-id> --dry-run
skill-hub feedback <skill-id> --force
```

For consuming managed skills in a project:

```bash
skill-hub init
skill-hub repo sync --json
skill-hub list
skill-hub search <keyword>
skill-hub use <skill-id>
skill-hub apply
skill-hub status
```

For remote publication, only after explicit user approval:

```bash
skill-hub push --dry-run --json
skill-hub push --message "update skills"
```

## Serve Troubleshooting

If a non-push command returns an old read-only error, the running `serve` process is stale. Update or restart the running service instead of changing the project workflow.

```bash
skill-hub serve status
skill-hub serve stop <name>
skill-hub serve start <name>
```

The latest installer restarts running registered `serve` instances after replacing the binary:

```bash
curl -s https://raw.githubusercontent.com/muidea/skill-hub/master/scripts/install-latest.sh | bash
```
