# Repository Guidelines

## Project Structure & Module Organization

`application/skill-hub/cmd` is the main CLI entry point. Core command behavior lives in `internal/cli`, while service-oriented kernel modules live under `internal/modules/kernel/*`. Shared packages are in `pkg/`, Git and repository logic is in `internal/git` and `internal/multirepo`, and adapter code is in `internal/adapter`. Python end-to-end tests are in `tests/e2e`. Release scripts and their Go tests are in `scripts/`. Bundled agent workflow skills are stored in `agent-skills/`.

## Build, Test, and Development Commands

- `make build`: builds `bin/skill-hub` with version ldflags.
- `make test`: runs all Go tests with `go test ./... --count 1`.
- `make lint`: runs `gofmt`, `go vet`, and `staticcheck` when available.
- `make test-pkg PKG=./internal/cli`: runs one Go package.
- `~/codespace/venv/bin/python3 -m pytest -p no:rerunfailures tests/e2e -c tests/e2e/pytest.ini`: runs e2e tests in this environment.
- `./scripts/create-release.sh --version X.Y.Z --from vA.B.C --yes`: the only supported release path.

If local caches under `$HOME` are read-only, set `GOCACHE`, `GOMODCACHE`, and `XDG_CACHE_HOME` to `/tmp` paths.

## Coding Style & Naming Conventions

Use Go 1.24 and standard `gofmt` formatting. Keep packages lowercase and focused. Prefer clear exported names such as `ApplyResult` and unexported camelCase helpers. CLI flags use kebab-case, for example `--dry-run` and `--skip-agent-skills`. Use `pkg/errors` for stable error codes and contextual wrapping. Keep comments sparse and useful; exported Go symbols should have GoDoc when appropriate.

## Testing Guidelines

Add table-driven Go tests beside the package under test, using `*_test.go` and descriptive `Test...` names. Use `t.TempDir()` for filesystem isolation. For CLI behavior, cover both direct CLI rendering and service/kernel logic when possible. E2e tests belong in `tests/e2e/test_*.py`; avoid relying on network unless the scenario explicitly requires it.

## Commit & Pull Request Guidelines

History follows Conventional Commit-style prefixes: `feat:`, `fix:`, `docs:`, `test:`, and `ci:`. Keep commits scoped and include docs/tests with behavior changes. Pull requests should describe the user-visible change, list validation commands, link issues when relevant, and include screenshots only for Web UI changes.

## Security & Release Notes

Do not commit secrets or local `~/.skill-hub` state. Document user-facing release changes under `docs/release-notes-vX.Y.Z-*.md` before releasing. Never create release tags manually; use `./scripts/create-release.sh` so tests, build, notes, branch push, and tag push stay consistent.
