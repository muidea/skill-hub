# Repository Guidelines

## Project Structure & Module Organization

`application/skill-hub/cmd` builds the Cobra CLI; `application/skill-hubd/cmd` builds the sole daemon, HTTP API, and Web UI host. CLI adapters live in `internal/clis/skill-hub`. Runtime selection is in `internal/runtime/{remote,local}`, and HTTP client adapters are in `internal/adapters/hubclient`.

Framework resource owners are Blocks under `internal/modules/blocks` (`project_state`, `repository`); cross-owner workflows and inbound HTTP/UI code are under `internal/modules/application`. Stable owner boundaries belong in `internal/pkg/*port`. Keep process lifecycle code in `internal/services/skill-hubd`. Go unit tests sit beside code; Python e2e tests are in `tests/e2e`; bundled skills are in `agent-skills/`.

## Build, Test, and Development Commands

- `make build`: builds `bin/skill-hub` and `bin/skill-hubd` with version metadata.
- `make test`: runs `go test ./... --count 1`.
- `make lint`: runs formatting checks, `go vet`, and `staticcheck` when installed.
- `make test-pkg PKG=./internal/clis/skill-hub`: tests one Go package.
- `~/codespace/venv/bin/python3 -m pytest -p no:rerunfailures tests/e2e -c tests/e2e/pytest.ini`: runs e2e tests.

If home-directory caches are unavailable, set `GOCACHE`, `GOMODCACHE`, and `XDG_CACHE_HOME` to `/tmp` paths.

## Coding Style & Architecture

Use Go 1.26 and `gofmt`. Keep package names lowercase, exported symbols GoDoc-worthy, and flags kebab-case (for example, `--dry-run`). Cobra code parses, renders, and selects a runtime; it must use focused ports rather than directly accessing config, Git, state, or module internals. Blocks own one resource; Application Modules coordinate workflows through typed EventHub contracts. `skill-hubd` is the only HTTP listener—do not restore `skill-hub serve`.

## Testing, Commits, and Releases

Use table-driven `Test...` tests and `t.TempDir()` for filesystem isolation. Name e2e files `tests/e2e/test_*.py`. Follow Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `ci:`). PRs should describe user-visible behavior and validation performed. Never commit secrets or local `~/.skill-hub` state. Add release notes under `docs/release-notes-vX.Y.Z-*.md`; release only with `./scripts/create-release.sh`.
