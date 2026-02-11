---
name: multi-file-skill
description: A skill with multiple files for testing feedback and apply commands
version: 1.0.0
author: Test User
category: testing
tags: [multi-file, test, feedback, apply]
dependencies:
  - python>=3.8
  - requests>=2.25.0
---

# Multi-File Skill

A comprehensive skill that includes multiple files to test feedback and apply commands in skill-hub.

## Description

This skill demonstrates how skill-hub handles skills with multiple files including:
- Configuration files
- Template files
- Utility scripts
- Documentation
- Test files

## File Structure

```
multi-file-skill/
├── SKILL.md (this file)
├── config.yaml (configuration)
├── templates/
│   ├── template1.j2
│   └── template2.j2
├── scripts/
│   └── setup.sh
├── utils/
│   └── helper.py
├── docs/
│   └── README.md
└── tests/
    └── test_basic.py
```

## Usage

This skill is designed for testing purposes only. It verifies that:

1. **feedback** command correctly copies all files from project to repository
2. **apply** command correctly copies all files from repository to project
3. File permissions and timestamps are preserved
4. Nested directory structures are handled correctly
5. File synchronization works across multiple operations

## Testing

Run the test suite with:

```bash
python -m pytest tests/test_multi_file_skill.py
```

## Configuration

See `config.yaml` for configuration options.

## License

MIT License