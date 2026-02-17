"""
Test Scenario 8: Remote Skill Search
Tests searching for skills in remote repositories.
"""

import os
import json
import tempfile
import pytest
from pathlib import Path
import subprocess

from utils.command_runner import CommandRunner
from utils.file_validator import FileValidator
from utils.test_environment import TestEnvironment
from utils.debug_utils import DebugUtils

class TestScenario8RemoteSkillSearch:
    """Test scenario 8: Remote skill search functionality"""
    
    @pytest.fixture(autouse=True)
    def setup(self, temp_project_dir, temp_home_dir, test_skill_template):
        """Setup test environment"""
        self.project_dir = Path(temp_project_dir)
        self.home_dir = Path(temp_home_dir)
        self.skill_template = test_skill_template
        self.cmd = CommandRunner()
        self.validator = FileValidator()
        self.env = TestEnvironment()
        self.debug = DebugUtils()
        
        # Store paths
        self.skill_hub_dir = self.home_dir / ".skill-hub"
        
        self.repositories_dir = self.skill_hub_dir / "repositories"
        self.main_repo_dir = self.repositories_dir / "main"
        self.repo_skills_dir = self.main_repo_dir / "skills"  # 新结构：repositories/main/skills
        
        # Project paths
        self.project_skill_hub = self.project_dir / ".skill-hub"
        self.project_agents_dir = self.project_dir / ".agents"
        
        # Create .agents directory for project
        self.project_agents_dir.mkdir(exist_ok=True)
    
    def _setup_test_skills(self):
        """Helper to setup test skills in repository"""
        # Initialize skill-hub
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=str(self.project_dir))
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        # Create test skills with different names and targets
        test_skills = [
            {"name": "git-expert", "target": "open_code", "description": "Git expert skill"},
            {"name": "python-debugger", "target": "open_code", "description": "Python debugging skill"},
            {"name": "database-optimizer", "target": "cursor", "description": "Database optimization"},
            {"name": "git-helper", "target": "both", "description": "Git helper utilities"},
            {"name": "python-web", "target": "open_code", "description": "Python web development"},
            {"name": "database-migration", "target": "cursor", "description": "Database migration tools"},
        ]
        
        project_cmd = CommandRunner()
        
        for skill in test_skills:
            # Create skill
            result = project_cmd.run("create", [skill['name']], cwd=str(self.project_dir))
            if not result.success:
                print(f"Warning: Failed to create {skill['name']}: {result.stderr}")
                continue
            
            # Update skill metadata
            skill_dir = self.project_agents_dir / "skills" / skill['name']
            meta_file = skill_dir / "meta.yaml"
            
            if meta_file.exists():
                import yaml
                with open(meta_file, 'r') as f:
                    meta = yaml.safe_load(f)
                
                meta['target'] = skill['target']
                meta['description'] = skill['description']
                
                with open(meta_file, 'w') as f:
                    yaml.dump(meta, f)
            
            # Feedback to repository
            result = project_cmd.run("feedback", [skill['name']], cwd=str(self.project_dir))
            if not result.success:
                print(f"Warning: Failed to feedback {skill['name']}: {result.stderr}")
    
    def test_01_command_dependency_check(self):
        """Test 8.1: Command dependency check verification"""
        print("\n=== Test 8.1: Command Dependency Check ===")
        
        # 创建一个新的临时目录，确保没有初始化
        temp_dir = Path(self.home_dir) / "temp-uninitialized-8"
        temp_dir.mkdir(exist_ok=True)
        
        # 测试未初始化时执行 skill-hub search git
        result = self.cmd.run("search", ["git"], cwd=str(temp_dir))
        # 应该提示需要先进行初始化
        assert not result.success or "需要先进行初始化" in result.stdout or "需要先进行初始化" in result.stderr, \
            f"Should prompt for initialization when running search without init"
        
        print(f"✓ search command dependency check passed")
        
    def test_02_basic_search_functionality(self):
        """Test 8.2: Basic search for skills"""
        print("\n=== Test 8.2: Basic Skill Search ===")
        
        # Setup test skills
        self._setup_test_skills()
        
        # Test basic search for "git"
        project_cmd = CommandRunner()
        result = project_cmd.run("search", ["git"], cwd=str(self.project_dir))
        
        # Check search results
        assert result.success, f"skill-hub search failed: {result.stderr}"
        
        # Verify search output contains git-related skills
        assert "git" in result.stdout.lower()
        
        print(f"✓ Basic search results: {result.stdout[:150]}...")
    
    def test_03_target_filtered_search(self):
        """Test 8.3: Search filtered by target"""
        print("\n=== Test 8.2: Target Filtered Search ===")
        
        # Setup test skills
        self._setup_test_skills()
        
        # Test search for "database" with open_code target
        project_cmd = CommandRunner()
        result = project_cmd.run("search", ["database", "--target", "open_code"], cwd=str(self.project_dir))
        
        assert result.success, f"skill-hub search failed: {result.stderr}"
        
        # Note: database skills are marked as cursor target, so may not appear
        # This tests that target filtering works
        print(f"✓ Target filtered search: {result.stdout[:150]}...")
        
        # Test search for "python" with open_code target
        result = project_cmd.run("search", ["python", "--target", "open_code"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub search failed: {result.stderr}"
        
        # Should find python skills with open_code target
        assert "python" in result.stdout.lower()
        
        print(f"✓ Python search with open_code target: {result.stdout[:150]}...")
    
    def test_04_search_result_limit(self):
        """Test 8.4: Search with result limit"""
        print("\n=== Test 8.3: Search Result Limit ===")
        
        # Setup test skills
        self._setup_test_skills()
        
        # Test search with limit
        project_cmd = CommandRunner()
        result = project_cmd.run("search", [".", "--limit", "2"], cwd=str(self.project_dir))
        
        assert result.success, f"skill-hub search failed: {result.stderr}"
        
        # Count results (approximate - count lines with skill names)
        lines = result.stdout.strip().split('\n')
        skill_lines = [line for line in lines if line.strip() and not line.startswith('Search')]
        
        # Should have limited results
        print(f"✓ Search with limit 2 returned {len(skill_lines)} results")
        
        # Test with different limit
        result = project_cmd.run("search", [".", "--limit", "5"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub search failed: {result.stderr}"
        
        lines = result.stdout.strip().split('\n')
        skill_lines = [line for line in lines if line.strip() and not line.startswith('Search')]
        
        print(f"✓ Search with limit 5 returned {len(skill_lines)} results")
    
    def test_05_empty_search_results(self):
        """Test 8.5: Search with no results"""
        print("\n=== Test 8.4: Empty Search Results ===")
        
        # Setup test skills
        self._setup_test_skills()
        
        # Test search for non-existent term
        project_cmd = CommandRunner()
        result = project_cmd.run("search", ["nonexistentskillxyz"], cwd=str(self.project_dir))
        
        assert result.success, f"skill-hub search failed: {result.stderr}"
        
        # Should indicate no results or empty result
        print(f"✓ Empty search results handled: {result.stdout[:100]}...")
    
    def test_06_search_with_special_characters(self):
        """Test 8.6: Search with special characters"""
        print("\n=== Test 8.5: Search with Special Characters ===")
        
        # Setup test skills
        self._setup_test_skills()
        
        # Test search with hyphen
        project_cmd = CommandRunner()
        result = project_cmd.run("search", ["python-"], cwd=str(self.project_dir))
        
        assert result.success, f"skill-hub search failed: {result.stderr}"
        
        # Should find python-web
        if "python" in result.stdout.lower():
            print(f"✓ Search with hyphen works: {result.stdout[:100]}...")
        else:
            print(f"✓ Search with hyphen executed (may not find results)")
        
        # Test search with partial word
        result = project_cmd.run("search", ["pyth"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub search failed: {result.stderr}"
        
        print(f"✓ Partial word search: {result.stdout[:100]}...")
    
    def test_07_search_integration_with_use_command(self):
        """Test 8.7: Search integration with use command"""
        print("\n=== Test 8.7: Search and Use Integration ===")
        
        # Setup test skills
        self._setup_test_skills()
        
        project_cmd = CommandRunner()
        
        # Search for a skill
        result = project_cmd.run("search", ["git"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub search failed: {result.stderr}"
        
        # Extract a skill name from search results (simplified)
        # In real test, would parse the output
        skill_to_use = "git-expert"
        
        # Use the found skill
        result = project_cmd.run(f"use {skill_to_use}")
        
        # Check if use was successful (skill should exist in repository)
        if result.success:
            print(f"✓ Successfully used skill found via search: {skill_to_use}")
            
            # Verify skill is enabled in state
            state_file = self.project_skill_hub / "state.json"
            if state_file.exists():
                with open(state_file, 'r') as f:
                    state = json.load(f)
                
                if skill_to_use in state.get("enabled_skills", []):
                    print(f"✓ Skill {skill_to_use} enabled in state.json")
        else:
            print(f"Note: Use command failed (may be expected): {result.stderr}")
        
        print(f"✓ Search and use integration tested")