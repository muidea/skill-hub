# Multi-File Skill Documentation

This document provides detailed documentation for the multi-file-skill, which is designed to test feedback and apply commands in skill-hub.

## Overview

The multi-file-skill is a comprehensive test skill that includes multiple files and directories to verify that skill-hub correctly handles:

1. File synchronization between project and repository
2. Nested directory structures
3. Different file types and permissions
4. State management for multiple files

## File Structure

```
multi-file-skill/
├── SKILL.md              # Main skill definition
├── config.yaml           # Configuration file
├── templates/            # Template files
│   ├── template1.j2     # Project structure template
│   └── template2.j2     # API client template
├── scripts/              # Shell scripts
│   └── setup.sh         # Setup script
├── utils/                # Python utilities
│   └── helper.py        # Helper functions
├── docs/                 # Documentation
│   └── README.md        # This file
└── tests/               # Test files
    └── test_basic.py    # Basic tests
```

## Configuration

The `config.yaml` file contains skill configuration including:

- Skill metadata (name, version, description)
- File handling settings (max size, allowed extensions)
- Template settings
- Script execution settings
- Test configuration

## Templates

### template1.j2
Generates a standard Python project structure with proper organization for source code, tests, and documentation.

### template2.j2
Generates API client configuration and code for interacting with REST APIs.

## Scripts

### setup.sh
Bash script that sets up the development environment:
- Checks Python version
- Creates virtual environment
- Installs dependencies
- Creates necessary directories
- Sets up environment variables

## Utilities

### helper.py
Python module with utility functions for:
- Configuration loading/saving
- File hash calculation
- Directory listing
- Directory structure creation
- Permission validation

## Testing

The skill includes a basic test suite to verify functionality. Run tests with:

```bash
python -m pytest tests/
```

## Usage with skill-hub

### Creating the Skill
```bash
skill-hub create multi-file-skill
```

### Providing Feedback
```bash
skill-hub feedback multi-file-skill
```

This command should copy all files from the project directory to the repository.

### Applying the Skill
```bash
skill-hub use multi-file-skill
skill-hub apply
```

This command should copy all files from the repository to the project directory.

### Verification
After feedback and apply operations, verify that:
1. All files are present in both locations
2. File contents are identical
3. Directory structure is preserved
4. File permissions are maintained

## Test Scenarios

The multi-file-skill is designed to test the following scenarios:

1. **Basic file synchronization**: All files should be copied correctly
2. **Nested directories**: Directory structure should be preserved
3. **Different file types**: Various file extensions should be handled
4. **File modifications**: Modified files should be detected and synced
5. **State management**: skill-hub state should track all files

## Troubleshooting

### Common Issues

1. **Missing files after apply**: Check if all files are included in the repository
2. **Permission errors**: Verify file permissions are preserved
3. **Directory structure mismatch**: Ensure nested directories are created correctly
4. **State synchronization**: Check skill-hub state.json for file tracking

### Debugging

Enable debug logging:
```bash
skill-hub --debug feedback multi-file-skill
```

Check skill-hub state:
```bash
skill-hub status
```

## License

MIT License - See SKILL.md for details