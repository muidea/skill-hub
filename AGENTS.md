# Agent Guidelines for skill-hub

This document provides guidelines for AI agents working on the skill-hub project.

## Build Commands

### Basic Build & Test
```bash
make build        # Build binary to bin/skill-hub
make test         # Run all Go tests
make test-verbose # Tests with verbose output
make lint         # Run linting (gofmt, go vet, staticcheck)
make deps         # Update dependencies (go mod tidy)
```

### Running Single Tests
```bash
go test ./internal/cli -v                   # Test specific package
go test ./internal/cli -v -run TestLoadSkill # Test single function
go test ./... -v -run "TestFeedback"        # Test with pattern matching
go test ./pkg/errors -bench=.               # Run benchmarks
go clean -testcache                          # Clear test cache
```

### End-to-End Testing (Python)
```bash
make test-e2e           # Run all e2e tests
make test-e2e-scenario SCENARIO=1  # Specific scenario (1-5)

# Run specific test file directly
cd tests/e2e && python3 -m pytest test_feedback_version_upgrade.py -v

# Run specific test method
cd tests/e2e && python3 -m pytest test_feedback_version_upgrade.py::TestFeedbackVersionAutoUpgrade::test_01_auto_upgrade_patch_version -v
```

## Code Style Guidelines

### Go Version & Toolchain
- Go 1.24.0 with toolchain go1.24.11
- Use Go modules, always run `go mod tidy` before committing

### Import Organization (3 groups, separated by blank lines)
```go
import (
    // Standard library
    "fmt"
    "os"

    // Third-party packages
    "gopkg.in/yaml.v3"
    "github.com/spf13/cobra"

    // Internal packages (skill-hub prefix)
    "skill-hub/internal/config"
    "skill-hub/pkg/errors"
)
```

### Naming Conventions
| Type | Convention | Example |
|------|-----------|---------|
| Packages | lowercase, single-word | `engine`, `cli`, `state` |
| Interfaces | `-er` suffix | `FileSystem`, `Manager`, `StateLoader` |
| Methods/Variables | camelCase | `loadSkill`, `projectPath` |
| Constants | PascalCase (exported), camelCase (internal) | `SkillStatusSynced`, `defaultTimeout` |
| Error variables | Prefix `Err` | `ErrSkillNotFound`, `ErrConfigInvalid` |
| Type parameters | Single uppercase | `T`, `K`, `V` |

### Error Handling

Use custom error package `pkg/errors`:
```go
// Define error codes in pkg/errors/errors.go
const ErrSkillNotFound ErrorCode = "SKILL_NOT_FOUND"

// Wrap errors with context
func LoadSkill(id string) (*Skill, error) {
    skill, err := findSkill(id)
    if err != nil {
        return nil, errors.Wrap(err, "LoadSkill: 查找技能失败")
    }
    return skill, nil
}

// Check errors early, return early
func Process(path string) error {
    if path == "" {
        return errors.NewValidationError("path cannot be empty")
    }
    // ... continue processing
}
```

### Logging
- Use `log/slog` for structured logging (Go 1.21+)
- Avoid the older `log` package
- Use contextual logging: `logger.With("key", value)`

### Testing Patterns
```go
// Table-driven tests with t.Run()
func TestLoadSkill(t *testing.T) {
    tests := []struct {
        name    string
        id      string
        wantErr bool
    }{
        {"valid", "test-skill", false},
        {"invalid", "", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := t.TempDir()  // Use t.TempDir() for temp directories
            // Test implementation
        })
    }
}
```

### Formatting & Documentation
- Use `gofmt` for formatting (run `make lint`)
- Line length: 80-100 characters
- **NO COMMENTS** unless explicitly requested
- Document exported functions with GoDoc comments

### Comments Policy
- **Do NOT add comments** unless explicitly asked by user
- Chinese comments acceptable for business logic
- English for technical details

## Multi-Repository Architecture
```
~/.skill-hub/
├── config.yaml          # Configuration
├── state.json           # Project states
└── repositories/main/skills/  # Archived skills
```

## Skill Structure
```
.agents/skills/{skill-id}/
├── SKILL.md              # Required: skill definition with YAML frontmatter
├── references/           # Optional: reference documents
└── scripts/              # Optional: helper scripts
```

### SKILL.md Format
```yaml
---
name: skill-name
description: Brief description
metadata:
  version: "1.0.0"
  author: "author-name"
---
# Skill content in Markdown
```

## Quality Assurance Checklist
Before committing:
1. `make test` - ensure tests pass
2. `make lint` - check code style
3. `make build` - ensure compilation succeeds

## Project Structure
```
skill-hub/
├── cmd/skill-hub/        # CLI entry point
├── internal/             # Internal packages (cli, config, engine, state, multirepo)
├── pkg/                  # Public packages (errors, spec, utils)
├── tests/e2e/            # End-to-end tests (Python)
└── .agents/              # Skill definitions
```

## Available Skills

### go-refactor-pro
Advanced Go refactoring skill. Use when:
- Code has significant duplication (DRY violations)
- Need to migrate to modern Go features (slog, generics, errors.Join)
- Performance optimization needed

Location: `.agents/skills/go-refactor-pro/SKILL.md`

## Key Dependencies
- `github.com/spf13/cobra` - CLI framework
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/go-git/go-git/v5` - Git operations
