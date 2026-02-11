"""
Test Scenario 6: Remote Synchronization and Multi-device Collaboration (Update Workflow)
Tests the workflow for updating local repository and refreshing project files from remote changes.
"""

import os
import json
import tempfile
import pytest
from pathlib import Path
import time
import shutil

from tests.e2e.utils.command_runner import CommandRunner
from tests.e2e.utils.file_validator import FileValidator
from tests.e2e.utils.test_environment import TestEnvironment
from tests.e2e.utils.debug_utils import DebugUtils


class TestScenario6RemoteSynchronization:
    """Test scenario 6: Remote synchronization and multi-device collaboration (Update workflow)"""
    
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
        self.repo_skills_dir = self.repo_dir / "skills"
        self.registry_file = self.skill_hub_dir / "registry.json"
        
        # Project paths
        self.project_skill_hub = self.project_dir / ".skill-hub"
        self.project_state = self.project_skill_hub / "state.json"
        self.project_agents_dir = self.project_dir / ".agents"
        self.project_skills_dir = self.project_agents_dir / "skills"
        
        # Create .agents directory for project
        self.project_agents_dir.mkdir(exist_ok=True)
        
    def _setup_initial_skill(self, skill_name="test-skill"):
        """Helper to setup initial skill in repository and project"""
        # Initialize home directory
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=str(self.home_dir))
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        # Create skill in project
        project_cmd = CommandRunner()
        result = project_cmd.run(f"create {skill_name}")
        assert result.success, f"skill-hub create failed: {result.stderr}"
        
        # Feedback to repository
        result = project_cmd.run(f"feedback {skill_name}")
        assert result.success, f"skill-hub feedback failed: {result.stderr}"
        
        # Enable skill in project
        result = project_cmd.run(f"use {skill_name}")
        assert result.success, f"skill-hub use failed: {result.stderr}"
        
        # Apply skill to project
        result = project_cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        return skill_name
    
    def test_01_pull_updates_from_remote(self):
        """Test 6.1: Pull updates from remote repository"""
        print("\n=== Test 6.1: Pull Updates from Remote ===")
        
        # Setup initial skill
        skill_name = self._setup_initial_skill("pull-test-skill")
        
        # Simulate remote update by modifying repository directly
        skill_repo_dir = self.repo_skills_dir / skill_name
        prompt_file = skill_repo_dir / "prompt.md"
        
        # Modify repository content (simulating remote update)
        with open(prompt_file, 'a') as f:
            f.write("\n\n# Updated from remote repository")
        
        # Update registry.json to reflect the change
        with open(self.registry_file, 'r') as f:
            registry = json.load(f)
        
        # Update hash to simulate version change
        import hashlib
        new_content = prompt_file.read_text()
        new_hash = hashlib.sha256(new_content.encode()).hexdigest()
        registry[skill_name]["hash"] = new_hash
        registry[skill_name]["version"] = "1.0.1"
        
        with open(self.registry_file, 'w') as f:
            json.dump(registry, f, indent=2)
        
        # Run pull command
        project_cmd = CommandRunner()
        result = project_cmd.run("pull", cwd=str(self.project_dir))
        assert result.success, f"skill-hub pull failed: {result.stderr}"
        
        # Verify repository was updated
        assert "Updated repository" in result.stdout or "Pulled" in result.stdout
        
        print(f"✓ Pull command executed successfully")
    
    def test_02_detect_outdated_skills(self):
        """Test 6.2: Detect outdated skills in project"""
        print("\n=== Test 6.2: Detect Outdated Skills ===")
        
        # Setup initial skill
        skill_name = self._setup_initial_skill("outdated-test-skill")
        
        # Simulate remote update
        skill_repo_dir = self.repo_skills_dir / skill_name
        prompt_file = skill_repo_dir / "prompt.md"
        
        with open(prompt_file, 'a') as f:
            f.write("\n\n# Remote update")
        
        # Update registry
        with open(self.registry_file, 'r') as f:
            registry = json.load(f)
        
        import hashlib
        new_content = prompt_file.read_text()
        new_hash = hashlib.sha256(new_content.encode()).hexdigest()
        registry[skill_name]["hash"] = new_hash
        registry[skill_name]["version"] = "1.1.0"
        
        with open(self.registry_file, 'w') as f:
            json.dump(registry, f, indent=2)
        
        # Run status command to detect outdated skills
        project_cmd = CommandRunner()
        result = project_cmd.run("status", cwd=str(self.project_dir))
        assert result.success, f"skill-hub status failed: {result.stderr}"
        
        # Verify outdated detection
        assert "Outdated" in result.stdout or "outdated" in result.stdout.lower()
        
        print(f"✓ Outdated skills detected: {result.stdout}")
    
    def test_03_refresh_outdated_skills(self):
        """Test 6.3: Refresh outdated skills with apply"""
        print("\n=== Test 6.3: Refresh Outdated Skills ===")
        
        # Setup initial skill
        skill_name = self._setup_initial_skill("refresh-test-skill")
        
        # Get original project file content
        skill_project_dir = self.project_skills_dir / skill_name
        original_prompt_file = skill_project_dir / "prompt.md"
        original_content = original_prompt_file.read_text()
        
        # Simulate remote update with significant change
        skill_repo_dir = self.repo_skills_dir / skill_name
        repo_prompt_file = skill_repo_dir / "prompt.md"
        
        new_content = original_content + "\n\n# IMPORTANT REMOTE UPDATE\nThis is a major update from the remote repository."
        repo_prompt_file.write_text(new_content)
        
        # Update registry
        with open(self.registry_file, 'r') as f:
            registry = json.load(f)
        
        import hashlib
        new_hash = hashlib.sha256(new_content.encode()).hexdigest()
        registry[skill_name]["hash"] = new_hash
        registry[skill_name]["version"] = "2.0.0"
        
        with open(self.registry_file, 'w') as f:
            json.dump(registry, f, indent=2)
        
        # First pull to update local repository
        project_cmd = CommandRunner()
        result = project_cmd.run("pull", cwd=str(self.project_dir))
        assert result.success, f"skill-hub pull failed: {result.stderr}"
        
        # Run apply to refresh project files
        result = project_cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        # Verify project file was updated
        updated_content = original_prompt_file.read_text()
        assert "IMPORTANT REMOTE UPDATE" in updated_content
        assert new_content == updated_content
        
        print(f"✓ Outdated skill refreshed successfully")
    
    def test_04_multi_device_collaboration_workflow(self):
        """Test 6.4: Complete multi-device collaboration workflow"""
        print("\n=== Test 6.4: Multi-device Collaboration Workflow ===")
        
        # Device A: Create and publish skill
        print("Device A: Creating and publishing skill...")
        skill_name = "collab-skill"
        
        # Initialize and create skill
        home_cmd = CommandRunner()
        result = home_cmd.run("init", cwd=str(self.home_dir))
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        project_cmd = CommandRunner()
        result = project_cmd.run(f"create {skill_name}")
        assert result.success, f"skill-hub create failed: {result.stderr}"
        
        # Modify skill
        skill_project_dir = self.project_skills_dir / skill_name
        prompt_file = skill_project_dir / "prompt.md"
        modified_content = prompt_file.read_text() + "\n\n# Added by Device A"
        prompt_file.write_text(modified_content)
        
        # Feedback to repository
        result = project_cmd.run(f"feedback {skill_name}")
        assert result.success, f"skill-hub feedback failed: {result.stderr}"
        
        # Simulate Device B: Pull updates
        print("Device B: Pulling updates...")
        
        # Create a separate "device B" repository by copying
        device_b_home = Path(tempfile.mkdtemp())
        device_b_skill_hub = device_b_home / ".skill-hub"
        
        # Copy repository from device A to device B
        shutil.copytree(self.skill_hub_dir, device_b_skill_hub)
        
        # Device B project
        device_b_project = Path(tempfile.mkdtemp())
        device_b_agents = device_b_project / ".agents"
        device_b_agents.mkdir(exist_ok=True)
        
        # Initialize device B project
        device_b_cmd = CommandRunner())
        
        # Enable skill in device B
        result = device_b_cmd.run(f"use {skill_name}")
        assert result.success, f"skill-hub use failed: {result.stderr}"
        
        # Apply skill in device B
        result = device_b_cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        # Verify device B has the skill
        device_b_skill_dir = device_b_project / ".agents" / "skills" / skill_name
        assert device_b_skill_dir.exists()
        
        device_b_prompt = device_b_skill_dir / "prompt.md"
        device_b_content = device_b_prompt.read_text()
        assert "Added by Device A" in device_b_content
        
        # Cleanup
        shutil.rmtree(device_b_home)
        shutil.rmtree(device_b_project)
        
        print(f"✓ Multi-device collaboration workflow completed successfully")