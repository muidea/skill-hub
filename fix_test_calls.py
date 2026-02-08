#!/usr/bin/env python3
"""
Fix CommandRunner.run() calls in test files.
"""

import re
import os
from pathlib import Path

def fix_test_file(file_path):
    """Fix CommandRunner.run() calls in a test file."""
    with open(file_path, 'r') as f:
        content = f.read()
    
    # Pattern 1: self.cmd.run("skill-hub <command>", timeout=XX)
    # Should be: self.cmd.run("<command>")
    pattern1 = r'self\.cmd\.run\("skill-hub\s+([^"]+)"\s*,\s*timeout\s*=\s*\d+\s*\)'
    replacement1 = r'self.cmd.run("\1")'
    content = re.sub(pattern1, replacement1, content)
    
    # Pattern 2: self.cmd.run(f"skill-hub {var}", timeout=XX)
    # Should be: self.cmd.run("command", var)
    # This is more complex and needs to handle different commands
    # For now, we'll do manual fixes for common patterns
    
    # Pattern for create command: self.cmd.run(f"skill-hub create {skill_name}", timeout=XX)
    pattern_create = r'self\.cmd\.run\(f"skill-hub create (\{skill_name\})"\s*,\s*timeout\s*=\s*\d+\s*\)'
    content = re.sub(pattern_create, r'self.cmd.run("create", \1)', content)
    
    # Pattern for feedback command: self.cmd.run(f"skill-hub feedback {skill_name}", timeout=XX)
    pattern_feedback = r'self\.cmd\.run\(f"skill-hub feedback (\{skill_name\})"\s*,\s*timeout\s*=\s*\d+\s*\)'
    content = re.sub(pattern_feedback, r'self.cmd.run("feedback", \1)', content)
    
    # Pattern for use command: self.cmd.run(f"skill-hub use {skill_name}", timeout=XX)
    pattern_use = r'self\.cmd\.run\(f"skill-hub use (\{skill_name\})"\s*,\s*timeout\s*=\s*\d+\s*\)'
    content = re.sub(pattern_use, r'self.cmd.run("use", \1)', content)
    
    # Pattern for remove command: self.cmd.run(f"skill-hub remove {skill_name}", timeout=XX)
    pattern_remove = r'self\.cmd\.run\(f"skill-hub remove (\{skill_name\})"\s*,\s*timeout\s*=\s*\d+\s*\)'
    content = re.sub(pattern_remove, r'self.cmd.run("remove", \1)', content)
    
    # Pattern for set-target command: self.cmd.run(f"skill-hub set-target {target}", timeout=XX)
    pattern_settarget = r'self\.cmd\.run\(f"skill-hub set-target (\{target\})"\s*,\s*timeout\s*=\s*\d+\s*\)'
    content = re.sub(pattern_settarget, r'self.cmd.run("set-target", \1)', content)
    
    # Pattern for validate-local command: self.cmd.run(f"skill-hub validate-local {skill_name}", timeout=XX)
    pattern_validate = r'self\.cmd\.run\(f"skill-hub validate-local (\{skill_name\})"\s*,\s*timeout\s*=\s*\d+\s*\)'
    content = re.sub(pattern_validate, r'self.cmd.run("validate-local", \1)', content)
    
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
    
    print("Done fixing test files.")

if __name__ == "__main__":
    main()