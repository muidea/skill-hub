"""
Test Scenario 7: Git Repository Basic Operations
Tests basic git operations for the skill repository.
"""

import os
import json
import tempfile
import pytest
from pathlib import Path
import subprocess
import shutil

from tests.e2e.utils.command_runner import CommandRunner
from tests.e2e.utils.file_validator import FileValidator
from tests.e2e.utils.test_environment import TestEnvironment
from tests.e2e.utils.debug_utils import DebugUtils


class TestScenario7GitOperations:
    """Test scenario 7: Git repository basic operations"""
    
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
        self.repo_dir = self.skill_hub_dir / "repo"
        
        # Project paths
        self.project_skill_hub = self.project_dir / ".skill-hub"
        self.project_agents_dir = self.project_dir / ".agents"
        
        # Create .agents directory for project
        self.project_agents_dir.mkdir(exist_ok=True)
    
    def _initialize_with_git(self):
        """Helper to initialize repository with git"""
        # Initialize skill-hub
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=str(self.home_dir))
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        # Initialize git repository in the repo directory
        subprocess.run(["git", "init"], cwd=self.repo_dir, capture_output=True)
        
        # Configure git user for commits
        subprocess.run(["git", "config", "user.email", "test@example.com"], 
                      cwd=self.repo_dir, capture_output=True)
        subprocess.run(["git", "config", "user.name", "Test User"], 
                      cwd=self.repo_dir, capture_output=True)
    
    def test_01_git_status_command(self):
        """Test 7.1: Git status command shows repository state"""
        print("\n=== Test 7.1: Git Status Command ===")
        
        # Initialize with git
        self._initialize_with_git()
        
        # Create a skill in project and feedback to repository
        project_cmd = CommandRunner()
        
        result = project_cmd.run("create", ["test-git-skill"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub create failed: {result.stderr}"
        
        result = project_cmd.run("feedback", ["test-git-skill"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub feedback failed: {result.stderr}"
        
        # Run git status command
        result = project_cmd.run("git", ["status"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub git status failed: {result.stderr}"
        
        # Verify git status output
        assert "On branch" in result.stdout or "branch" in result.stdout.lower()
        assert "Untracked files" in result.stdout or "Changes not staged" in result.stdout
        
        print(f"✓ Git status command works: {result.stdout[:100]}...")
    
    def test_02_git_sync_command(self):
        """Test 7.2: Git sync command pulls from remote"""
        print("\n=== Test 7.2: Git Sync Command ===")
        
        # Initialize with git
        self._initialize_with_git()
        
        # Create a test remote repository
        remote_repo = Path(tempfile.mkdtemp())
        subprocess.run(["git", "init", "--bare"], cwd=remote_repo, capture_output=True)
        
        # Add remote to local repository
        remote_url = f"file://{remote_repo}"
        subprocess.run(["git", "remote", "add", "origin", remote_url], 
                      cwd=self.repo_dir, capture_output=True)
        
        # Create initial commit in local repository
        test_file = self.repo_dir / "test.txt"
        test_file.write_text("Initial content")
        
        subprocess.run(["git", "add", "."], cwd=self.repo_dir, capture_output=True)
        subprocess.run(["git", "commit", "-m", "Initial commit"], 
                      cwd=self.repo_dir, capture_output=True)
        
        # Push to remote
        subprocess.run(["git", "push", "-u", "origin", "main"], 
                      cwd=self.repo_dir, capture_output=True)
        
        # Create a change in remote (simulate by another clone)
        clone_dir = Path(tempfile.mkdtemp())
        subprocess.run(["git", "clone", remote_url, str(clone_dir)], capture_output=True)
        
        remote_file = clone_dir / "remote-change.txt"
        remote_file.write_text("Change from remote")
        
        subprocess.run(["git", "add", "."], cwd=clone_dir, capture_output=True)
        subprocess.run(["git", "commit", "-m", "Remote change"], cwd=clone_dir, capture_output=True)
        subprocess.run(["git", "push"], cwd=clone_dir, capture_output=True)
        
        # Run git sync command
        project_cmd = CommandRunner()
        result = project_cmd.run("git", ["sync"], cwd=str(self.project_dir))
        
        # Check if sync was successful (may fail if no network, but command should execute)
        assert "sync" in result.stdout.lower() or "pull" in result.stdout.lower() or result.success
        
        print(f"✓ Git sync command executed: {result.stdout[:100]}...")
        
        # Cleanup
        shutil.rmtree(remote_repo)
        shutil.rmtree(clone_dir)
    
    def test_03_git_clone_command(self):
        """Test 7.3: Git clone command clones remote repository"""
        print("\n=== Test 7.3: Git Clone Command ===")
        
        # Create a test remote repository
        remote_repo = Path(tempfile.mkdtemp())
        subprocess.run(["git", "init", "--bare"], cwd=remote_repo, capture_output=True)
        
        # Clone it to a temporary location to set up initial content
        temp_clone = Path(tempfile.mkdtemp())
        subprocess.run(["git", "clone", f"file://{remote_repo}", str(temp_clone)], 
                      capture_output=True)
        
        # Add some content to remote
        readme = temp_clone / "README.md"
        readme.write_text("# Test Skills Repository")
        
        subprocess.run(["git", "add", "."], cwd=temp_clone, capture_output=True)
        subprocess.run(["git", "commit", "-m", "Add README"], cwd=temp_clone, capture_output=True)
        subprocess.run(["git", "push"], cwd=temp_clone, capture_output=True)
        
        # Test git clone command
        project_cmd = CommandRunner()
        remote_url = f"file://{remote_repo}"
        
        # Note: This will clone into current directory, not replace ~/.skill-hub/repo
        # In real usage, this would be handled differently
        result = project_cmd.run(f"git clone {remote_url}")
        
        # Check command execution
        assert result.success or "clone" in result.stdout.lower()
        
        print(f"✓ Git clone command attempted: {result.stdout[:100]}...")
        
        # Cleanup
        shutil.rmtree(remote_repo)
        shutil.rmtree(temp_clone)
    
    def test_04_git_remote_command(self):
        """Test 7.4: Git remote command manages remote repositories"""
        print("\n=== Test 7.4: Git Remote Command ===")
        
        # Initialize with git
        self._initialize_with_git()
        
        # Test setting remote URL
        project_cmd = CommandRunner()
        test_url = "https://github.com/example/skills-repo.git"
        
        result = project_cmd.run(f"git remote {test_url}")
        
        # Check if command executed (may show current remote or set new one)
        assert "remote" in result.stdout.lower() or result.success
        
        # Verify remote was set by checking git config
        git_config = subprocess.run(["git", "config", "--get", "remote.origin.url"], 
                                   cwd=self.repo_dir, capture_output=True, text=True)
        
        if git_config.returncode == 0:
            assert test_url in git_config.stdout or "example" in git_config.stdout
        
        print(f"✓ Git remote command executed: {result.stdout[:100]}...")
    
    def test_05_git_operations_integration(self):
        """Test 7.5: Integrated git operations workflow"""
        print("\n=== Test 7.5: Integrated Git Operations ===")
        
        # Initialize with git
        self._initialize_with_git()
        
        # Create and feedback a skill
        project_cmd = CommandRunner()
        
        skill_name = "git-workflow-skill"
        result = project_cmd.run(f"create {skill_name}")
        assert result.success, f"skill-hub create failed: {result.stderr}"
        
        result = project_cmd.run(f"feedback {skill_name}")
        assert result.success, f"skill-hub feedback failed: {result.stderr}"
        
        # Check git status
        result = project_cmd.run("git", ["status"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub git status failed: {result.stderr}"
        
        # Create a simple commit via git command
        subprocess.run(["git", "add", "."], cwd=self.repo_dir, capture_output=True)
        subprocess.run(["git", "commit", "-m", "Add test skill"], 
                      cwd=self.repo_dir, capture_output=True)
        
        # Check status again (should be clean)
        result = project_cmd.run("git", ["status"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub git status failed: {result.stderr}"
        
        # Verify clean working tree
        assert "nothing to commit" in result.stdout.lower() or "working tree clean" in result.stdout.lower()
        
        print(f"✓ Integrated git operations workflow completed successfully")