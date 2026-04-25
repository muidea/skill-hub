# v0.8.1 Release Notes Pipeline Follow-Up

Release date: 2026-04-25

## Summary

`v0.8.1` is a follow-up release for `v0.8.0` global skill management. It closes the lint issue found after the initial `v0.8.0` tag, makes release notes durable by letting both the local release script and GitHub Release workflow prefer tracked release note documents, and fixes project-level `apply <id>` refresh for `Outdated` skills.

## Included Changes

- Removes the unused global service test helper that caused `staticcheck` `U1000`.
- Adds a tracked `v0.8.0` global skill management release note under `docs/`.
- Adds and updates the tracked `v0.8.1` release note so the patch release can publish a complete curated body.
- Updates `scripts/create-release.sh` so `--notes-only`, dry-run, and tag creation prefer `docs/release-notes-v<version>-*.md` when exactly one matching document exists.
- Keeps the previous commit-message-only release note generation path as the fallback when no tracked release note document exists.
- Updates GitHub Release workflow to use the same tracked-document-first behavior.
- Fails release note resolution when multiple tracked documents match the same version, preventing ambiguous release bodies.
- Fixes the duplicate tracked release notes edge case so the local release script fails explicitly instead of silently falling back to generated notes.
- Restores project-level `skill-hub apply <id>` so a single enabled project skill in `Outdated` state can be refreshed from its source repository.
- Updates project apply state after refresh so the skill version matches the repository and status returns to `Synced`.

## Commit Scope

This release is expected to include these core follow-up commits after `v0.8.0`:

```text
b54329d test: remove unused global test helper
9614ade docs: add global skill management release notes
e0df341 feat: prefer tracked release notes documents
0b3e53e ci: prefer tracked release notes for GitHub releases
50a8257 fix: fail on duplicate tracked release notes
b449efd fix: refresh outdated project skills with apply
```

The tracked `v0.8.1` release note document commits are also part of this release line, including the final release note update used to publish this release:

```text
f0eaed5 docs: add v0.8.1 release notes
2f95992 docs: update v0.8.1 release notes
<current> docs: prepare v0.8.1 release notes
```

These release note commits keep the published `v0.8.1` body aligned with the actual post-`v0.8.0` follow-up scope instead of relying on terse tag annotations.

## Validation

The follow-up scope has been validated with:

```bash
make lint
GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod-cache go test ./... --count=1
GOCACHE=/tmp/go-build GOMODCACHE=/tmp/go-mod-cache go test ./scripts --count=1
./scripts/create-release.sh --notes-only --yes --version 0.8.0 --from v0.7.0 --to v0.8.0 --output /tmp/skill-hub-v0.8.0-notes.md
./scripts/create-release.sh --notes-only --yes --version 0.8.1 --from v0.8.0 --to HEAD --output /tmp/skill-hub-v0.8.1-notes.md
SKILL_HUB_BIN=/home/rangh/aispace/skill/skill-hub/bin/skill-hub /home/rangh/codespace/venv/bin/pytest -q -p no:rerunfailures tests/e2e/test_project_apply_outdated.py
SKILL_HUB_BIN=/home/rangh/aispace/skill/skill-hub/bin/skill-hub /home/rangh/codespace/venv/bin/pytest -q -p no:rerunfailures tests/e2e
```

Latest observed e2e result:

```text
110 passed, 9 skipped
```

The GitHub workflow release note resolver has also been locally simulated for:

- `v0.8.0`: tracked document is used.
- `v9.9.9`: fallback release note body is generated when no tracked document exists.
- duplicate `docs/release-notes-v0.2.0-*.md`: local release script exits with an explicit error and lists the conflicting files.

## Operational Notes

- `dist/` remains generated output and ignored by Git.
- Tracked release notes live under `docs/release-notes-v<version>-*.md`.
- For a given version, keep exactly one tracked release note document.
- Remote publication is still explicit; this change does not push branches, tags, or releases automatically.
