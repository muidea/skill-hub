# Agent Guidelines for skill-hub

This document provides guidelines for AI agents working on the skill-hub project.

## Build Commands

### Basic Build & Test
```bash
make build        # Build binary
make test         # Run all Go tests
make test-verbose # Tests with verbose output
make test-coverage # Tests with coverage report
make lint         # Run linting checks (gofmt, go vet, staticcheck)
make deps         # Update dependencies (go mod tidy)
make clean        # Clean build artifacts
```

### Running Single Tests
```bash
go test ./internal/cli -v                   # Test specific package
go test ./internal/cli -v -run TestLoadSkill # Test single function
go test ./pkg/errors -bench=.              # Run benchmarks
go clean -testcache                         # Clear test cache
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

### Import Organization (3 groups)
```go
import (
    // Standard library
    "fmt"
    "os"

    // Third-party packages
    "gopkg.in/yaml.v3"
    "github.com/spf13/cobra"

    // Internal packages
    "skill-hub/internal/config"
    "skill-hub/pkg/errors"
)
```

### Naming Conventions
- **Packages**: lowercase, single-word (e.g., `engine`, `cli`)
- **Interfaces**: `-er` suffix (e.g., `FileSystem`, `Manager`)
- **Methods/Variables**: camelCase
- **Constants**: PascalCase for exported, camelCase for internal
- **Error variables**: Prefix with `Err` (e.g., `ErrSkillNotFound`)
- **Type parameters**: Single uppercase letters (e.g., `T`, `K`, `V`)

### Error Handling
- Use custom error package `pkg/errors`
- Define error codes as constants in `pkg/errors/errors.go`
- Wrap errors with context using `errors.Wrap()`
- Always check errors, return early on errors

Example:
```go
func LoadSkill(id string) (*Skill, error) {
    skill, err := findSkill(id)
    if err != nil {
        return nil, errors.Wrap(err, "LoadSkill: 查找技能失败")
    }
    return skill, nil
}
```

### Logging
- Use `log/slog` for structured logging (Go 1.21+)
- Avoid using the older `log` package
- Use contextual logging with `With()` and `WithGroup()`

### Testing Patterns
- Use table-driven tests with `t.Run()` for subtests
- Use `t.TempDir()` for temporary directories
- Mock dependencies using interfaces
- Test both success and error cases

### Formatting & Documentation
- Use `gofmt` for consistent formatting
- Line length: 80-100 characters
- Document exported functions with GoDoc comments
- Use Chinese comments for business logic, English for technical details

## Available Skills

### go-refactor-pro
A specialized skill for advanced Go refactoring. Use when:
- Code has significant duplication (DRY violations)
- Need to migrate to modern Go features (slog, generics, errors.Join)
- Performance optimization is needed
- Code needs decoupling for testability

Located at: `.agents/skills/go-refactor-pro/SKILL.md`

## Multi-Repository Architecture
- **Storage**: `~/.skill-hub/repositories/{repo-name}/`
- **Default repository**: "main" as the archive repository
- **Config**: `~/.skill-hub/config.yaml`
- **State file**: `~/.skill-hub/state.json`

## Skill Structure
- Skills use path format IDs (e.g., `owner/skill-name`)
- Each skill requires `SKILL.md` with YAML frontmatter
- Variables use `{{.VARIABLE_NAME}}` syntax

## Quality Assurance
Before committing:
1. Run `make test` - ensure tests pass
2. Run `make lint` - check code style
3. Run `make build` - ensure compilation succeeds

## Project Structure
- `cmd/`: Command-line interfaces
- `internal/`: Internal packages (not for external use)
- `pkg/`: Public packages
- `examples/`: Example skills
- `tests/`: Test files
- `.agents/`: Skill definitions

## Key Files
- `Makefile`: Build and test commands
- `go.mod`: Go module dependencies
- `DEVELOPMENT.md`: Developer documentation
