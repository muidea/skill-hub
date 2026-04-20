# skill-hub End-to-End Tests

Python-based end-to-end test framework for skill-hub, covering the scenario test files and supplemental regression cases under `tests/e2e/`.

## Overview

This test framework provides comprehensive end-to-end testing for skill-hub's core functionality. It tests the complete workflow from skill creation to application, modification, and cleanup.

## Test Scenarios

Based on the business scenarios defined in the repo test documents and current CLI behavior:

1. **Developer Full Workflow** (`test_scenario1.py`)
   - Environment initialization with `skill-hub init`
   - Skill creation with `skill-hub create`
   - Editing and feedback with `skill-hub feedback`
   - Skill listing verification

2. **Project Application Workflow** (`test_scenario2.py`)
   - Enabling skills with `skill-hub use`
   - Physical application with `skill-hub apply`
   - State updates without target inputs

3. **Iteration Feedback Workflow** (`test_scenario3.py`)
   - Project file modification detection
   - Status checking with `skill-hub status`
   - Synchronization back to repository with `skill-hub feedback`
   - Standard modification extraction

4. **Cancel and Cleanup Workflow** (`test_scenario4.py`)
   - Skill removal with `skill-hub remove`
   - Physical cleanup verification
   - Repository safety (never delete source files)

5. **Target Business Removal Workflow** (`test_scenario5.py`)
   - Removed target command/flag entrypoints fail
   - Standard workflows do not write `preferred_target`
   - Compatibility metadata does not filter list results

6. **Skill Content Commands** (`test_skill_content_commands.py`)
   - **create**: New skill has standard structure (SKILL.md + scripts/, references/, assets/); when skill already exists, create validates and refreshes state for registration/archiving.
   - **status**: Changes under scripts/references/assets are reflected as Modified.
   - **feedback**: Full skill directory (including subdirs) is synced to repository.
   - **apply** (open_code): Full skill directory is copied from repo to project (see also `test_feedback_apply_multifile.py`).
   - **use**: Only updates state.json; skill files are not copied until `apply` is run.

7. **Service Mode** (`test_service_mode.py`)
   - `skill-hub serve` health check
   - `skill-hub serve register/start/status/stop/remove` instance management flow
   - Web UI homepage availability
   - Web UI page-level structure checks for the catalog total, admin repo form, project workflow controls, and no admin secretKey write entry
   - CLI bridge for `repo list` / `repo list --json` / `list` / `status`
   - CLI bridge preserves service error codes for read-only write attempts
   - CLI bridge write path for `use` / `apply` / `feedback`
   - CLI bridge lifecycle path for `register` / `import --fix-frontmatter --archive`
   - CLI bridge duplicate-management path for `dedupe` / `sync-copies`
   - CLI bridge portability audit path for `lint --paths --fix`
   - CLI bridge validation path for `validate --links --json`
   - CLI bridge audit report path for `audit --output`
   - CLI bridge batch feedback path for `feedback --all --force --json`
   - Unit-covered service bridge path for `repo sync --json`
   - Verify service-managed project skill files and repo archive updates

8. **State Prune** (`test_state_prune.py`)
   - `skill-hub prune` keeps valid `state.json` project records unchanged
   - After project directory deletion, stale state records are removed

9. **Bulk Import, Register, and JSON Status** (`test_bulk_import_register_status.py`)
   - `skill-hub register` records existing `.agents/skills/<id>/SKILL.md` without overwriting content
   - `skill-hub status --json` returns machine-readable project status
   - `skill-hub validate <id> --fix` repairs legacy frontmatter and creates backups
   - `skill-hub import .agents/skills --fix-frontmatter --archive --force` registers, validates, and archives multiple skills

10. **Duplicate Detection and Canonical Sync** (`test_dedupe_sync_copies.py`)
   - `skill-hub dedupe . --canonical .agents/skills --json` reports same-id duplicate groups and content conflicts
   - `skill-hub sync-copies --canonical .agents/skills --scope . --dry-run` previews copy synchronization
   - `skill-hub sync-copies --canonical .agents/skills --scope .` backs up and synchronizes non-canonical copies

11. **Path Portability Lint** (`test_path_lint.py`)
   - `skill-hub lint . --paths --project-root <dir> --json` reports fixable and manual-review local paths
   - `skill-hub lint . --paths --project-root <dir> --fix` rewrites project-local paths and creates backups
   - `skill-hub lint . --paths --project-root <dir> --fix --dry-run --json` reports would-rewrite items without modifying files

12. **Markdown Link Validation** (`test_validate_links.py`)
   - `skill-hub validate <id> --links --json` reports broken local Markdown links with source file and line
   - `skill-hub validate --all --links --json` passes after missing bundled/project-local links are restored

13. **Audit Report** (`test_audit_report.py`)
   - `skill-hub audit .agents/skills --output <file>` writes a Markdown refresh progress report
   - `skill-hub audit .agents/skills --format json` emits machine-readable audit metrics

14. **Batch Feedback JSON** (`test_feedback_all_json.py`)
   - `skill-hub feedback --all --force --json` archives all registered project skills
   - JSON output reports total, applied, skipped, planned, failed, and per-skill results

## Architecture

### Directory Structure
```
tests/e2e/
├── README.md                    # This file
├── pytest.ini                   # Pytest configuration
├── requirements.txt             # Python dependencies
├── conftest.py                  # Pytest fixtures configuration
├── run_tests.py                 # Main test runner
├── environment_check.py         # Environment validation
├── utils/                       # Core utility classes
│   ├── __init__.py
│   ├── command_runner.py       # skill-hub command execution
│   ├── file_validator.py       # File content validation
│   ├── test_environment.py     # Test environment management
│   ├── yaml_validator.py       # YAML validation
│   ├── network_checker.py      # Network connectivity checks
│   └── debug_utils.py          # Debug utilities
├── fixtures/                    # Pytest fixtures
│   ├── __init__.py
│   ├── project_fixtures.py     # Project-related fixtures
│   ├── skill_fixtures.py       # Skill-related fixtures
│   └── adapter_fixtures.py     # Adapter-related fixtures
├── data/                        # Test data
│   └── test_skills/
│       └── my-logic-skill/     # Test skill template
│           ├── SKILL.md
│           └── expected_output/
│               ├── README.md
│               ├── skill_info.json
│               └── skill_validate.json
├── test_scenario*.py           # Scenario test implementations
├── test_skill_content_commands.py  # create/status/feedback/apply/use skill content
├── test_state_prune.py           # prune invalid state.json project records
├── test_bulk_import_register_status.py # register/status JSON/frontmatter fix/import archive
├── test_dedupe_sync_copies.py      # duplicate detection and canonical copy sync
├── test_path_lint.py              # local path portability lint and fix
├── test_validate_links.py         # Markdown local link validation
├── test_audit_report.py           # Markdown/JSON audit report generation
├── test_feedback_all_json.py      # batch feedback, pull/push previews, and JSON output
├── test_feedback_apply_multifile.py # Multi-file skill feedback & apply
└── test_feedback_version_upgrade.py # Version upgrade on feedback
```

### Core Components

1. **CommandRunner** - Executes `skill-hub` commands with timeout and retry logic
2. **FileValidator** - Validates file contents and structures
3. **TestEnvironment** - Manages test environments and cleanup
4. **YAMLValidator** - Validates YAML syntax and structure
5. **NetworkChecker** - Checks network connectivity for relevant tests
6. **DebugUtils** - Provides debugging tools and snapshots

## Prerequisites

### System Requirements
- Python 3.8 or higher
- `skill-hub` binary: when running from the repo, `bin/skill-hub` is used if present (run `make build` first); otherwise PATH or `SKILL_HUB_BIN` is used
- Network connectivity (for pull/push/git remote tests)

`pull --check --json`, `push --dry-run --json`, `git status --json`, and `git sync --json` failure summaries are covered without network access. Actual pull/push tests still require a configured remote.

### Python Dependencies
Install with: `pip install -r tests/e2e/requirements.txt`

Required packages:
- `pytest` - Test framework
- `pyyaml` - YAML parsing and validation
- `requests` - Network checks

## Usage

### Using Makefile (Recommended)
```bash
# Run all end-to-end tests
make test-e2e

# Run specific test scenario (1-5)
make test-e2e-scenario SCENARIO=1

# Check test environment
make test-e2e-check

# Install Python dependencies
make test-e2e-deps

# Clean temporary files
make test-e2e-clean
```

### Using Python Scripts Directly
```bash
# Run all tests
cd tests/e2e
python3 run_tests.py

# Run specific scenarios
python3 run_tests.py -s 1 3 5

# Run specific test class
python3 run_tests.py -t TestScenario1DeveloperWorkflow

# Check environment
python3 environment_check.py

# List available tests
python3 run_tests.py --list

# Clean up temporary files
python3 run_tests.py --cleanup
```

### Command Line Options for `run_tests.py`
```
-s, --scenarios     Run specific test scenarios (1-5)
-t, --test          Run a specific test by name
-l, --list          List available tests
-c, --check         Check environment only
--cleanup           Clean up temporary test files
-v, --verbose       Verbose output
-d, --debug         Enter debugger on test failure
--no-check          Skip environment check
```

## Test Design Principles

### 1. Isolation
- Each test runs in its own temporary HOME directory
- Project tests use temporary project directories
- No interference with user's actual skill-hub configuration

### 2. Network Awareness
- Tests requiring network are automatically skipped when offline
- `NetworkChecker` class detects network availability
- `@pytest.mark.skipif` decorator handles skipping

### 3. Debug Friendliness
- Failed tests preserve temporary directories for inspection
- `DebugUtils` creates snapshots of test state
- Detailed logging with clear error messages

### 4. Validation
- `FileValidator` performs exact file content matching
- `YAMLValidator` checks YAML syntax and structure
- Expected outputs stored in `data/test_skills/expected_output/`

### 5. Safety
- Repository files are NEVER deleted by tests
- Cleanup operations verified for safety
- State consistency checks after each operation

## Test Data

### Test Skill Template
Located at `data/test_skills/my-logic-skill/`:
- `SKILL.md` - Complete skill template for testing
- `expected_output/` - Expected command outputs for validation

### Expected Output Files
- `skill_info.json` - Expected `skill-hub skill info` output
- `skill_validate.json` - Expected `skill-hub validate` output
- Additional files can be added as needed

## Writing New Tests

### 1. Follow Existing Patterns
- Use the established utility classes
- Follow the same import structure
- Use provided fixtures (`temp_project_dir`, `temp_home_dir`, etc.)

### 2. Test Structure
```python
class TestNewScenario:
    @pytest.fixture(autouse=True)
    def setup(self, temp_project_dir, temp_home_dir):
        # Setup code here
    
    def test_specific_functionality(self):
        # Test implementation
        # Use self.cmd for command execution
        # Use self.validator for file validation
```

### 3. Best Practices
- One assertion per test concept
- Clear test names describing the behavior
- Setup and teardown in fixtures
- Network-dependent tests properly marked

## Troubleshooting

### Common Issues

1. **`skill-hub` command not found**
   - Ensure `skill-hub` is built and in PATH
   - Run `make build` to build the binary
   - Check with `which skill-hub`

2. **Python dependencies missing**
   - Run `make test-e2e-deps` or `pip install -r tests/e2e/requirements.txt`

3. **Network tests failing**
   - Tests requiring network are skipped when offline
   - Check network connectivity
   - Use `--no-check` to skip network checks

4. **Permission errors**
   - Ensure write permissions in test directories
   - Run tests as a user with appropriate permissions

### Debugging Failed Tests
- Use `-v` flag for verbose output
- Use `-d` flag to enter debugger on failure
- Check preserved temporary directories in `/tmp/skill_hub_test_*`
- Use `DebugUtils.create_snapshot()` in tests

## Integration with CI/CD

### Environment Variables
- `SKIP_NETWORK_TESTS=1` - Skip network-dependent tests
- `PYTEST_ARGS` - Additional pytest arguments
- `TEST_SCENARIOS` - Comma-separated list of scenarios to run

### Sample CI Configuration
```yaml
test-e2e:
  stage: test
  script:
    - make build
    - make test-e2e-deps
    - make test-e2e-check
    - make test-e2e
  artifacts:
    paths:
      - tests/e2e/test-reports/
    when: always
```

## Contributing

### Adding New Test Scenarios
1. Create `test_scenarioX.py` or a dedicated test file (e.g. `test_skill_content_commands.py`) following existing patterns
2. Add to `run_tests.py` in the `additional_tests` list if not a numbered scenario
3. Update `README.md` documentation
4. Add to Makefile if needed

### Modifying Test Utilities
1. Update utility classes in `utils/`
2. Maintain backward compatibility
3. Update dependent tests if needed
4. Document changes in `README.md`

### Updating Test Data
1. Update files in `data/test_skills/`
2. Ensure expected outputs match current `skill-hub` behavior
3. Version test data if format changes significantly

## License

Part of the skill-hub project. See main project LICENSE for details.
