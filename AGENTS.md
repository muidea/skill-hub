# Agent Guidelines for skill-hub

This document provides guidelines for AI agents working on the skill-hub project.

## Build Commands

### Basic Build & Test
```bash
make build        # Build binary
make test         # Run all Go tests
make test-verbose # Tests with verbose output
make test-coverage # Tests with coverage report
make coverage-html # Generate HTML coverage report
make lint         # Run linting checks
make deps         # Update dependencies
make clean        # Clean build artifacts
```

### Running Single Tests
```bash
make test-pkg PKG=./internal/cli  # Test specific package
go test ./internal/cli -v         # Test specific file
go test ./internal/cli -v -run TestLoadSkill  # Test single function
go test ./pkg/errors -bench=.     # Run benchmarks
```

### End-to-End Testing
```bash
make test-e2e           # Python e2e tests (requires deps)
make test-e2e-simple    # Simple e2e tests (no deps)
make test-e2e-scenario SCENARIO=1  # Specific test scenario
make test-e2e-deps      # Install Python test dependencies
```

### Release & Installation
```bash
make release-all  # Build release packages for all platforms
make install      # Install to ~/.local/bin
```

## Code Style Guidelines

### Go Version & Toolchain
- Go 1.24.0 with toolchain go1.24.11
- Use Go modules, always run `go mod tidy` before committing

### Import Organization
```go
import (
    // Standard library
    "fmt"
    "os"
    "path/filepath"

    // Third-party packages
    "gopkg.in/yaml.v3"
    "github.com/spf13/cobra"

    // Internal packages
    "skill-hub/internal/config"
    "skill-hub/pkg/errors"
)
```

### Naming Conventions
- **Packages**: lowercase, single-word (e.g., `engine`, `cli`, `errors`)
- **Interfaces**: `-er` suffix (e.g., `FileSystem`, `Manager`)
- **Methods/Variables**: camelCase, descriptive
- **Constants**: PascalCase for exported, camelCase for internal
- **Error variables**: Prefix with `Err` (e.g., `ErrSkillNotFound`)

### Error Handling
- Use custom error package `pkg/errors`
- Define error codes as constants in `pkg/errors/errors.go`
- Wrap errors with context using `errors.Wrap()`
- Always check errors, return early on errors

Example from `pkg/errors/errors.go:18`:
```go
const (
    ErrSkillNotFound ErrorCode = "SKILL_NOT_FOUND"
    ErrSkillInvalid  ErrorCode = "SKILL_INVALID"
)
```

### Logging
- Use `log/slog` for structured logging (Go 1.21+)
- Avoid using the older `log` package
- Use contextual logging with `With()` and `WithGroup()`

### Testing Patterns
- Use table-driven tests for multiple test cases
- Use `t.Run()` for subtests
- Use `t.TempDir()` for temporary directories
- Mock dependencies using interfaces
- Test both success and error cases

Example from test files:
```go
func TestSkillManager(t *testing.T) {
    tmpDir := t.TempDir()
    t.Run("Create skill manager", func(t *testing.T) {
        // test implementation
    })
}
```

### File Structure
- **Internal packages**: Code not for external use
- **Public packages**: Reusable components in `pkg/`
- **Command packages**: Executable binaries in `cmd/`
- **Test files**: Use `_test.go` suffix

### Formatting & Documentation
- Use `gofmt` for consistent formatting
- Line length: 80-100 characters
- Tabs for indentation (not spaces)
- Document exported functions, types, packages
- Use GoDoc format comments

## Multi-Repository Architecture

The project uses a multi-repository architecture:
- **Storage**: `~/.skill-hub/repositories/{repo-name}/`
- **Default repository**: Named "main" as the archive repository
- **Config location**: `~/.skill-hub/config.yaml` with `multi_repo` section
- **State file**: `~/.skill-hub/state.json`

All skills modified via `feedback` are archived to the default repository.

## Quality Assurance

Before committing changes:
1. Run `make test` to ensure tests pass
2. Run `make lint` to check code style
3. Run `make build` to ensure compilation succeeds
4. For significant changes, run `make test-e2e-simple`
5. Update documentation if needed

## Project Structure

### Key Directories
- `cmd/`: Command-line interfaces
- `internal/`: Internal packages
- `pkg/`: Public packages
- `examples/`: Example skills
- `tests/`: Test files and data
- `.agents/`: Skill definitions

### Important Files
- `Makefile`: Build and test commands
- `go.mod`: Go module dependencies
- `DEVELOPMENT.md`: Developer documentation