#!/usr/bin/env python3
"""
Fix CommandResult attribute references in test files.
"""

import re
import os
from pathlib import Path

def fix_test_file(file_path):
    """Fix CommandResult attribute references in a test file."""
    with open(file_path, 'r') as f:
        content = f.read()
    
    # Replace result.returncode with result.exit_code
    content = content.replace('result.returncode', 'result.exit_code')
    
    # Also replace assert result.returncode == 0 with assert result.success
    # But be careful - only when comparing to 0
    pattern = r'assert\s+result\.exit_code\s*==\s*0'
    content = re.sub(pattern, 'assert result.success', content)
    
    # Write back
    with open(file_path, 'w') as f:
        f.write(content)
    
    print(f"Fixed {file_path}")

def main():
    """Main function."""
    test_dir = Path(__file__).parent / "tests" / "e2e"
    
    # Fix all test scenario files
    for test_file in test_dir.glob("test_scenario*.py"):
        fix_test_file(test_file)
    
    print("Done fixing CommandResult references.")

if __name__ == "__main__":
    main()