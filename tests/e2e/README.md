# skill-hub End-to-End Tests

Python-based end-to-end test framework for skill-hub, covering the 5 business scenarios from `docs/testCase.md`.

## Overview

This test framework provides comprehensive end-to-end testing for skill-hub's core functionality. It tests the complete workflow from skill creation to application, modification, and cleanup.

## Test Scenarios

Based on the business scenarios defined in `docs/testCase.md`:

1. **Developer Full Workflow** (`test_scenario1.py`)
   - Environment initialization with `skill-hub init`
   - Skill creation with `skill-hub create`
   - Editing and feedback with `skill-hub feedback`
   - Skill listing verification

2. **Project Application Workflow** (`test_scenario2.py`)
   - Setting project target with `skill-hub set-target`
   - Enabling skills with `skill-hub use`
   - Physical application with `skill-hub apply`
   - Command-line target override

3. **Iteration Feedback Workflow** (`test_scenario3.py`)
   - Project file modification detection
   - Status checking with `skill-hub status`
   - Synchronization back to repository with `skill-hub feedback`
   - Target-specific modification extraction

4. **Cancel and Cleanup Workflow** (`test_scenario4.py`)
   - Skill removal with `skill-hub remove`
   - Physical cleanup verification
   - Multi-target cleanup with `--target all`
   - Repository safety (never delete source files)

5. **Update and Validation Workflow** (`test_scenario5.py`)
   - Repository updates with `skill-hub update`
   - Skill validation with `skill-hub validate`
   - Invalid YAML detection
   - Outdated skill detection

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
└── test_scenario[1-5].py       # Test scenario implementations
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
- `skill-hub` command available in PATH
- Network connectivity (for update tests)

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
- `skill_validate.json` - Expected `skill-hub skill validate` output
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
1. Create `test_scenarioX.py` following existing patterns
2. Add to `run_tests.py` scenario detection
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