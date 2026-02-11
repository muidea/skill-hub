#!/usr/bin/env python3
"""
Basic tests for multi-file-skill
"""

import os
import sys
import pytest
from pathlib import Path
import yaml
import json

# Add parent directory to path to import helper
sys.path.insert(0, str(Path(__file__).parent.parent))

from utils.helper import load_config, calculate_file_hash, list_files


def test_config_file_exists():
    """Test that config.yaml exists and is valid YAML"""
    config_path = Path(__file__).parent.parent / "config.yaml"
    assert config_path.exists(), f"config.yaml not found at {config_path}"
    
    # Try to load the config
    config = load_config(str(config_path))
    assert isinstance(config, dict), "Config should be a dictionary"
    assert "skill" in config, "Config should have 'skill' section"
    assert "files" in config, "Config should have 'files' section"


def test_skill_md_exists():
    """Test that SKILL.md exists"""
    skill_md_path = Path(__file__).parent.parent / "SKILL.md"
    assert skill_md_path.exists(), f"SKILL.md not found at {skill_md_path}"
    
    # Check it contains expected content
    content = skill_md_path.read_text()
    assert "multi-file-skill" in content, "SKILL.md should mention skill name"
    assert "feedback" in content.lower() or "apply" in content.lower(), \
        "SKILL.md should mention feedback or apply commands"


def test_template_files():
    """Test that template files exist"""
    templates_dir = Path(__file__).parent.parent / "templates"
    assert templates_dir.exists(), f"Templates directory not found at {templates_dir}"
    
    template_files = list(templates_dir.glob("*.j2"))
    assert len(template_files) >= 2, f"Expected at least 2 template files, found {len(template_files)}"
    
    # Check specific template files
    template1 = templates_dir / "template1.j2"
    template2 = templates_dir / "template2.j2"
    
    assert template1.exists(), f"template1.j2 not found at {template1}"
    assert template2.exists(), f"template2.j2 not found at {template2}"
    
    # Check template content
    template1_content = template1.read_text()
    assert "Project:" in template1_content, "template1.j2 should contain 'Project:'"
    assert "{{" in template1_content and "}}" in template1_content, \
        "template1.j2 should contain template variables"


def test_script_file():
    """Test that setup.sh script exists"""
    scripts_dir = Path(__file__).parent.parent / "scripts"
    assert scripts_dir.exists(), f"Scripts directory not found at {scripts_dir}"
    
    setup_script = scripts_dir / "setup.sh"
    assert setup_script.exists(), f"setup.sh not found at {setup_script}"
    
    # Check script content
    script_content = setup_script.read_text()
    assert "#!/bin/bash" in script_content, "setup.sh should start with shebang"
    assert "multi-file-skill" in script_content, "setup.sh should mention skill name"


def test_helper_module():
    """Test that helper.py module exists and can be imported"""
    utils_dir = Path(__file__).parent.parent / "utils"
    assert utils_dir.exists(), f"Utils directory not found at {utils_dir}"
    
    helper_file = utils_dir / "helper.py"
    assert helper_file.exists(), f"helper.py not found at {helper_file}"
    
    # Test helper functions
    config = load_config(str(Path(__file__).parent.parent / "config.yaml"))
    assert isinstance(config, dict), "load_config should return a dictionary"
    
    # Test file listing
    files = list_files(str(Path(__file__).parent.parent))
    assert len(files) > 0, "Should list at least one file"
    assert any(f.endswith(".py") for f in files), "Should list Python files"


def test_documentation():
    """Test that documentation exists"""
    docs_dir = Path(__file__).parent.parent / "docs"
    assert docs_dir.exists(), f"Docs directory not found at {docs_dir}"
    
    readme_file = docs_dir / "README.md"
    assert readme_file.exists(), f"README.md not found at {readme_file}"
    
    # Check documentation content
    docs_content = readme_file.read_text()
    assert "# Multi-File Skill Documentation" in docs_content, \
        "README.md should have correct title"
    assert "skill-hub" in docs_content.lower(), \
        "README.md should mention skill-hub"


def test_file_structure():
    """Test the complete file structure"""
    skill_dir = Path(__file__).parent.parent
    
    expected_dirs = [
        skill_dir,
        skill_dir / "templates",
        skill_dir / "scripts", 
        skill_dir / "utils",
        skill_dir / "docs",
        skill_dir / "tests"
    ]
    
    expected_files = [
        skill_dir / "SKILL.md",
        skill_dir / "config.yaml",
        skill_dir / "templates" / "template1.j2",
        skill_dir / "templates" / "template2.j2",
        skill_dir / "scripts" / "setup.sh",
        skill_dir / "utils" / "helper.py",
        skill_dir / "docs" / "README.md",
        skill_dir / "tests" / "test_basic.py"
    ]
    
    # Check directories
    for dir_path in expected_dirs:
        assert dir_path.exists(), f"Directory not found: {dir_path}"
        assert dir_path.is_dir(), f"Not a directory: {dir_path}"
    
    # Check files
    for file_path in expected_files:
        assert file_path.exists(), f"File not found: {file_path}"
        assert file_path.is_file(), f"Not a file: {file_path}"
        
        # Check file is not empty
        if file_path.stat().st_size == 0:
            pytest.fail(f"File is empty: {file_path}")


def test_file_hashes():
    """Test that file hashes can be calculated (basic file integrity)"""
    skill_dir = Path(__file__).parent.parent
    
    # Test a few key files
    test_files = [
        skill_dir / "SKILL.md",
        skill_dir / "config.yaml",
        skill_dir / "utils" / "helper.py"
    ]
    
    for file_path in test_files:
        file_hash = calculate_file_hash(str(file_path))
        assert file_hash, f"Failed to calculate hash for {file_path}"
        assert len(file_hash) == 64, f"Invalid hash length for {file_path}: {len(file_hash)}"
        
        # Hash should be hexadecimal
        try:
            int(file_hash, 16)
        except ValueError:
            pytest.fail(f"Hash is not hexadecimal for {file_path}: {file_hash}")


if __name__ == "__main__":
    # Run tests directly
    pytest.main([__file__, "-v"])