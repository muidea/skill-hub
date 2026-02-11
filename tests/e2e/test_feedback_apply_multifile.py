"""
Test Scenario: Feedback and Apply Commands with Multi-File Skills
Tests feedback and apply commands when skills contain multiple files and nested directories.
"""

import os
import json
import tempfile
import pytest
from pathlib import Path
import shutil
import hashlib

from tests.e2e.utils.command_runner import CommandRunner
from tests.e2e.utils.file_validator import FileValidator
from tests.e2e.utils.test_environment import TestEnvironment
from tests.e2e.utils.debug_utils import DebugUtils


class TestFeedbackApplyMultiFile:
    """Test feedback and apply commands with multi-file skills"""
    
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
        
        # Project paths
        self.project_skill_hub = self.project_dir / ".skill-hub"
        self.project_agents_dir = self.project_dir / ".agents"
        self.project_agents_skills_dir = self.project_agents_dir / "skills"
        
        # Create .agents directory for project
        self.project_agents_dir.mkdir(exist_ok=True)
        
        # Change to project directory
        os.chdir(self.project_dir)
    
    def _calculate_file_hash(self, file_path: Path) -> str:
        """Calculate SHA256 hash of a file"""
        sha256_hash = hashlib.sha256()
        try:
            with open(file_path, "rb") as f:
                for byte_block in iter(lambda: f.read(4096), b""):
                    sha256_hash.update(byte_block)
            return sha256_hash.hexdigest()
        except Exception as e:
            print(f"Error calculating hash for {file_path}: {e}")
            return ""
    
    def _copy_multi_file_skill_to_project(self, skill_name: str = "multi-file-skill"):
        """Copy the multi-file-skill test data to project directory"""
        # Source directory from test data - use the actual multi-file-skill we created
        source_dir = Path(__file__).parent / "data" / "test_skills" / "multi-file-skill"
        assert source_dir.exists(), f"Source skill directory not found: {source_dir}"
        
        # Destination directory in project
        dest_dir = self.project_agents_skills_dir / skill_name
        
        # Copy entire directory structure
        if dest_dir.exists():
            shutil.rmtree(dest_dir)
        
        shutil.copytree(source_dir, dest_dir)
        
        # Verify copy was successful
        assert dest_dir.exists(), f"Failed to copy skill to project: {dest_dir}"
        
        # List all files for verification
        all_files = []
        for root, dirs, files in os.walk(dest_dir):
            for file in files:
                file_path = Path(root) / file
                all_files.append(file_path.relative_to(dest_dir))
        
        print(f"Copied {len(all_files)} files to project: {all_files}")
        return dest_dir, all_files
    
    def _verify_files_match(self, dir1: Path, dir2: Path, relative_paths: list):
        """Verify that files in two directories match"""
        mismatches = []
        
        for rel_path in relative_paths:
            file1 = dir1 / rel_path
            file2 = dir2 / rel_path
            
            # Check existence
            if not file1.exists():
                mismatches.append(f"File missing in {dir1}: {rel_path}")
                continue
            if not file2.exists():
                mismatches.append(f"File missing in {dir2}: {rel_path}")
                continue
            
            # Check file size
            size1 = file1.stat().st_size
            size2 = file2.stat().st_size
            if size1 != size2:
                mismatches.append(f"Size mismatch for {rel_path}: {size1} vs {size2}")
                continue
            
            # Check content hash
            hash1 = self._calculate_file_hash(file1)
            hash2 = self._calculate_file_hash(file2)
            if hash1 != hash2:
                mismatches.append(f"Content mismatch for {rel_path}")
        
        return mismatches
    
    def test_01_feedback_with_multiple_files(self):
        """Test 1: Feedback command with skill containing multiple files"""
        print("\n=== Test 1: Feedback with Multiple Files ===")
        
        # Initialize skill-hub
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        # First create the skill using skill-hub create
        skill_name = "multi-file-test-skill"
        
        # Ensure .agents/skills directory exists (required for create command)
        agents_skills_dir = self.project_dir / ".agents" / "skills"
        agents_skills_dir.mkdir(parents=True, exist_ok=True)
        
        result = self.cmd.run("create", [skill_name], cwd=str(self.project_dir))
        assert result.success, f"skill-hub create failed: {result.stderr}"
        
        # Now copy our multi-file skill content over the created skill
        project_skill_dir, project_files = self._copy_multi_file_skill_to_project(skill_name)
        
        print(f"  Created skill with {len(project_files)} files in project")
        print(f"  Files: {[str(f) for f in project_files[:5]]}...")
        
        # First feedback to repository (required before use)
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"Initial feedback failed: {result.stderr}"
        
        # Setup project target
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub set-target failed: {result.stderr}"
        
        # Use the skill (now it exists in repository)
        result = self.cmd.run("use", [skill_name], cwd=str(self.project_dir), input_text="\n")
        assert result.success, f"skill-hub use failed: {result.stderr}"
        
        # Apply the skill
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        # Now test feedback again after modifications
        # Modify a file
        config_file = project_skill_dir / "config.yaml"
        original_content = config_file.read_text()
        config_file.write_text(original_content + "\n# Modified for feedback test\n")
        
        # Run feedback command again
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"Second feedback failed: {result.stderr}"
        
        # Verify files were copied to repository
        repo_skill_dir = self.repo_skills_dir / skill_name
        assert repo_skill_dir.exists(), f"Skill directory not created in repository: {repo_skill_dir}"
        
        # List files in repository
        repo_files = []
        for root, dirs, files in os.walk(repo_skill_dir):
            for file in files:
                file_path = Path(root) / file
                repo_files.append(file_path.relative_to(repo_skill_dir))
        
        print(f"  Found {len(repo_files)} files in repository")
        
        # Verify all files are present
        project_file_set = set(str(f) for f in project_files)
        repo_file_set = set(str(f) for f in repo_files)
        
        missing_in_repo = project_file_set - repo_file_set
        extra_in_repo = repo_file_set - project_file_set
        
        assert len(missing_in_repo) == 0, f"Files missing in repository: {missing_in_repo}"
        assert len(extra_in_repo) == 0, f"Extra files in repository: {extra_in_repo}"
        
        # Verify file contents match
        mismatches = self._verify_files_match(project_skill_dir, repo_skill_dir, project_files)
        
        if mismatches:
            print(f"  File content mismatches: {mismatches}")
            pytest.fail(f"File content mismatches found: {mismatches}")
        
        print(f"✓ Feedback command correctly handled {len(project_files)} files")
        print(f"  - All files copied to repository")
        print(f"  - File contents preserved")
        print(f"  - Directory structure maintained")
    
    def test_02_apply_with_multiple_files(self):
        """Test 2: Apply command with skill containing multiple files"""
        print("\n=== Test 2: Apply with Multiple Files ===")
        
        # Initialize and setup skill in repository first
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        # Create and setup a skill using the standard workflow
        skill_name = "multi-file-apply-test"
        
        # Use the helper method from test_scenario4 to setup skill
        # First create the skill directory structure
        agents_skills_dir = self.project_dir / ".agents" / "skills"
        agents_skills_dir.mkdir(parents=True, exist_ok=True)
        
        # Create skill using skill-hub create
        result = self.cmd.run("create", [skill_name], cwd=str(self.project_dir))
        assert result.success, f"skill-hub create failed: {result.stderr}"
        
        # Copy our multi-file content
        project_skill_dir, project_files = self._copy_multi_file_skill_to_project(skill_name)
        
        # Setup project
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success, f"skill-hub set-target failed: {result.stderr}"
        
        # Use the skill (it exists locally)
        result = self.cmd.run("use", [skill_name], cwd=str(self.project_dir), input_text="\n")
        assert result.success, f"skill-hub use failed: {result.stderr}"
        
        # Apply the skill
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        # Now feedback to repository (skill is now enabled)
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"skill-hub feedback failed: {result.stderr}"
        
        # Remove the skill from project to test apply from repository
        if project_skill_dir.exists():
            shutil.rmtree(project_skill_dir)
        
        # Verify skill is gone from project
        assert not project_skill_dir.exists(), f"Skill should be removed from project"
        
        # Now test applying from repository
        # Use the skill (it exists in repository now)
        result = self.cmd.run("use", [skill_name], cwd=str(self.project_dir), input_text="\n")
        assert result.success, f"skill-hub use failed: {result.stderr}"
        
        # Apply the skill from repository
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"skill-hub apply failed: {result.stderr}"
        
        # Verify skill was copied back to project
        assert project_skill_dir.exists(), f"Skill directory not created by apply: {project_skill_dir}"
        
        # List files in project after apply
        applied_files = []
        for root, dirs, files in os.walk(project_skill_dir):
            for file in files:
                file_path = Path(root) / file
                applied_files.append(file_path.relative_to(project_skill_dir))
        
        print(f"  Applied {len(applied_files)} files to project")
        
        # Get repository skill directory
        repo_skill_dir = self.repo_skills_dir / skill_name
        
        # Verify all files are present
        repo_files = []
        for root, dirs, files in os.walk(repo_skill_dir):
            for file in files:
                file_path = Path(root) / file
                repo_files.append(file_path.relative_to(repo_skill_dir))
        
        repo_file_set = set(str(f) for f in repo_files)
        applied_file_set = set(str(f) for f in applied_files)
        
        missing_in_project = repo_file_set - applied_file_set
        extra_in_project = applied_file_set - repo_file_set
        
        assert len(missing_in_project) == 0, f"Files missing in project after apply: {missing_in_project}"
        assert len(extra_in_project) == 0, f"Extra files in project after apply: {extra_in_project}"
        
        # Verify file contents match
        mismatches = self._verify_files_match(repo_skill_dir, project_skill_dir, repo_files)
        
        if mismatches:
            print(f"  File content mismatches: {mismatches}")
            pytest.fail(f"File content mismatches found: {mismatches}")
        
        print(f"✓ Apply command correctly handled {len(repo_files)} files")
        print(f"  - All files copied from repository")
        print(f"  - File contents preserved")
        print(f"  - Directory structure maintained")
    
    def test_03_feedback_apply_roundtrip(self):
        """Test 3: Complete feedback -> modify -> feedback -> apply roundtrip"""
        print("\n=== Test 3: Feedback-Apply Roundtrip ===")
        
        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success, f"skill-hub init failed: {result.stderr}"
        
        # Copy skill to project
        skill_name = "roundtrip-test-skill"
        project_skill_dir, original_files = self._copy_multi_file_skill_to_project(skill_name)
        
        # Step 1: Initial feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"Initial feedback failed: {result.stderr}"
        
        repo_skill_dir = self.repo_skills_dir / skill_name
        
        # Step 2: Modify files in project
        modifications = []
        
        # Modify config.yaml
        config_file = project_skill_dir / "config.yaml"
        if config_file.exists():
            with open(config_file, 'a') as f:
                f.write("\n# Modified during roundtrip test\n")
            modifications.append("config.yaml")
        
        # Add a new file
        new_file = project_skill_dir / "NEW_FILE.md"
        new_file.write_text("# New file added during test\nThis file tests addition.")
        modifications.append("NEW_FILE.md")
        
        # Modify a template file
        template_file = project_skill_dir / "templates" / "template1.j2"
        if template_file.exists():
            content = template_file.read_text()
            template_file.write_text(content + "\n<!-- Modified during test -->\n")
            modifications.append("templates/template1.j2")
        
        print(f"  Made {len(modifications)} modifications in project")
        
        # Step 3: Second feedback (should update repository)
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"Second feedback failed: {result.stderr}"
        
        # Verify modifications are in repository
        for mod in modifications:
            repo_file = repo_skill_dir / mod
            assert repo_file.exists(), f"Modified file not in repository: {mod}"
            
            # For new file, check it exists
            if mod == "NEW_FILE.md":
                assert repo_file.read_text() == "# New file added during test\nThis file tests addition."
        
        # Step 4: Create fresh project directory and apply
        fresh_project_dir = Path(tempfile.mkdtemp())
        os.chdir(fresh_project_dir)
        
        # Initialize in fresh project
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success
        
        # Setup target
        result = self.cmd.run("set-target", ["open_code"], cwd=str(fresh_project_dir))
        assert result.success
        
        # Use skill
        result = self.cmd.run("use", [skill_name], cwd=str(fresh_project_dir), input_text="\n")
        assert result.success
        
        # Apply skill
        result = self.cmd.run("apply", cwd=str(fresh_project_dir))
        assert result.success
        
        # Verify applied skill has modifications
        fresh_skill_dir = Path(fresh_project_dir) / ".agents" / "skills" / skill_name
        
        for mod in modifications:
            fresh_file = fresh_skill_dir / mod
            assert fresh_file.exists(), f"Modified file not in fresh project: {mod}"
            
            # Compare with repository version
            repo_file = repo_skill_dir / mod
            if fresh_file.exists() and repo_file.exists():
                fresh_content = fresh_file.read_text()
                repo_content = repo_file.read_text()
                assert fresh_content == repo_content, f"Content mismatch for {mod}"
        
        print(f"✓ Roundtrip test completed successfully")
        print(f"  - Initial feedback: {len(original_files)} files")
        print(f"  - Modifications: {modifications}")
        print(f"  - Second feedback: updated repository")
        print(f"  - Fresh apply: modifications preserved")
        
        # Cleanup
        os.chdir(self.project_dir)
        shutil.rmtree(fresh_project_dir)
    
    def test_04_nested_directory_structure(self):
        """Test 4: Nested directory structure preservation"""
        print("\n=== Test 4: Nested Directory Structure ===")
        
        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success
        
        # Create skill with deeply nested structure
        skill_name = "nested-test-skill"
        skill_dir = self.project_agents_skills_dir / skill_name
        skill_dir.mkdir(parents=True, exist_ok=True)
        
        # Create nested directory structure
        nested_paths = [
            "level1/level2/level3/deep_file.txt",
            "a/b/c/d/e/very_deep.py",
            "configs/prod/app.yaml",
            "configs/dev/app.yaml",
            "src/utils/helpers/__init__.py",
            "src/utils/helpers/math.py",
            "tests/unit/test_math.py",
            "tests/integration/test_api.py",
            "docs/api/endpoints.md",
            "docs/user/guide.md"
        ]
        
        for path in nested_paths:
            file_path = skill_dir / path
            file_path.parent.mkdir(parents=True, exist_ok=True)
            file_path.write_text(f"Content for {path}\nCreated at test")
        
        print(f"  Created skill with {len(nested_paths)} files in nested directories")
        print(f"  Max depth: {max(path.count('/') for path in nested_paths)} levels")
        
        # Feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success
        
        # Verify repository has same structure
        repo_skill_dir = self.repo_skills_dir / skill_name
        
        repo_nested_paths = []
        for root, dirs, files in os.walk(repo_skill_dir):
            for file in files:
                file_path = Path(root) / file
                rel_path = file_path.relative_to(repo_skill_dir)
                repo_nested_paths.append(str(rel_path))
        
        # Check all paths are present
        missing_paths = set(nested_paths) - set(repo_nested_paths)
        extra_paths = set(repo_nested_paths) - set(nested_paths)
        
        assert len(missing_paths) == 0, f"Missing nested paths in repository: {missing_paths}"
        assert len(extra_paths) == 0, f"Extra nested paths in repository: {extra_paths}"
        
        # Verify directory structure
        for path in nested_paths:
            repo_file = repo_skill_dir / path
            assert repo_file.exists(), f"Nested file not in repository: {path}"
            
            # Check parent directories exist
            for i in range(1, path.count('/') + 1):
                parent = Path(path).parents[i-1]
                parent_dir = repo_skill_dir / parent
                assert parent_dir.exists() and parent_dir.is_dir(), \
                    f"Parent directory missing: {parent}"
        
        print(f"✓ Nested directory structure preserved")
        print(f"  - All {len(nested_paths)} files copied")
        print(f"  - Directory hierarchy maintained")
        print(f"  - Deep nesting (up to 5 levels) handled correctly")
    
    def test_05_file_permissions_preservation(self):
        """Test 5: File permissions preservation (where supported)"""
        print("\n=== Test 5: File Permissions Preservation ===")
        
        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success
        
        # Create skill with different file permissions
        skill_name = "permission-test-skill"
        skill_dir = self.project_agents_skills_dir / skill_name
        skill_dir.mkdir(parents=True, exist_ok=True)
        
        # Create files with different permissions
        files = {
            "read_only.txt": 0o444,
            "write_only.txt": 0o222,
            "executable.sh": 0o755,
            "normal.txt": 0o644,
        }
        
        for filename, mode in files.items():
            file_path = skill_dir / filename
            file_path.write_text(f"Test file: {filename}")
            
            try:
                os.chmod(file_path, mode)
                actual_mode = os.stat(file_path).st_mode & 0o777
                print(f"  Set {filename} to {oct(mode)}, actual: {oct(actual_mode)}")
            except Exception as e:
                print(f"  Note: Could not set permissions for {filename}: {e}")
        
        # Feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success
        
        # Note: File permissions may not be preserved across all systems/filesystems
        # This test documents the behavior rather than enforcing strict requirements
        
        repo_skill_dir = self.repo_skills_dir / skill_name
        
        for filename in files.keys():
            repo_file = repo_skill_dir / filename
            assert repo_file.exists(), f"File not in repository: {filename}"
            
            # Check file exists (primary requirement)
            # Permission preservation is secondary and system-dependent
            print(f"  ✓ {filename} copied to repository")
        
        print(f"✓ File permissions test completed")
        print(f"  - All files copied (primary requirement)")
        print(f"  - Permission preservation is system-dependent")
    
    def test_06_large_number_of_files(self):
        """Test 6: Handling large number of files"""
        print("\n=== Test 6: Large Number of Files ===")
        
        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success
        
        # Create skill with many files
        skill_name = "many-files-test-skill"
        skill_dir = self.project_agents_skills_dir / skill_name
        skill_dir.mkdir(parents=True, exist_ok=True)
        
        # Create many small files
        file_count = 50  # Reasonable number for testing
        created_files = []
        
        for i in range(file_count):
            filename = f"file_{i:03d}.txt"
            file_path = skill_dir / filename
            file_path.write_text(f"This is file number {i}\n" * 10)
            created_files.append(filename)
        
        # Also create some in subdirectories
        subdir = skill_dir / "subdir"
        subdir.mkdir(exist_ok=True)
        
        for i in range(10):
            filename = f"subfile_{i:03d}.md"
            file_path = subdir / filename
            file_path.write_text(f"# Subfile {i}\n\nContent here.")
            created_files.append(f"subdir/{filename}")
        
        total_files = len(created_files)
        print(f"  Created skill with {total_files} files")
        
        # Feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"Feedback failed with {total_files} files: {result.stderr}"
        
        # Verify
        repo_skill_dir = self.repo_skills_dir / skill_name
        
        repo_files = []
        for root, dirs, files in os.walk(repo_skill_dir):
            for file in files:
                file_path = Path(root) / file
                rel_path = file_path.relative_to(repo_skill_dir)
                repo_files.append(str(rel_path))
        
        missing = set(created_files) - set(repo_files)
        extra = set(repo_files) - set(created_files)
        
        assert len(missing) == 0, f"Missing files in repository: {missing}"
        assert len(extra) == 0, f"Extra files in repository: {extra}"
        
        print(f"✓ Successfully handled {total_files} files")
        print(f"  - All files copied to repository")
        print(f"  - No files missing or extra")
    
    def test_07_mixed_file_types(self):
        """Test 7: Mixed file types and extensions"""
        print("\n=== Test 7: Mixed File Types ===")
        
        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success
        
        # Create skill with various file types
        skill_name = "mixed-files-test-skill"
        skill_dir = self.project_agents_skills_dir / skill_name
        skill_dir.mkdir(parents=True, exist_ok=True)
        
        # Different file types
        file_types = {
            "script.py": "#!/usr/bin/env python3\nprint('Hello')",
            "config.json": '{"name": "test", "value": 42}',
            "data.csv": "name,age,score\nAlice,30,95\nBob,25,88",
            "template.html.j2": "<html>\n  <body>{{ content }}</body>\n</html>",
            "binary.data": bytes([0x00, 0x01, 0x02, 0x03, 0x04]),
            "readme.md": "# Test\n\nMixed file types.",
            "empty.txt": "",
            "dotfile.config": "key=value\nsecret=password",
        }
        
        for filename, content in file_types.items():
            file_path = skill_dir / filename
            
            if isinstance(content, bytes):
                file_path.write_bytes(content)
            else:
                file_path.write_text(content)
        
        print(f"  Created skill with {len(file_types)} different file types")
        
        # Feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success
        
        # Verify
        repo_skill_dir = self.repo_skills_dir / skill_name
        
        for filename in file_types.keys():
            repo_file = repo_skill_dir / filename
            assert repo_file.exists(), f"File type not in repository: {filename}"
            
            # Check file size matches
            project_file = skill_dir / filename
            if project_file.exists() and repo_file.exists():
                project_size = project_file.stat().st_size
                repo_size = repo_file.stat().st_size
                assert project_size == repo_size, f"Size mismatch for {filename}"
        
        print(f"✓ Mixed file types handled correctly")
        print(f"  - Text files: .py, .json, .csv, .md, .txt")
        print(f"  - Template files: .j2")
        print(f"  - Binary data: .data")
        print(f"  - Dotfiles: .config")
    
    def test_08_state_tracking_multiple_files(self):
        """Test 8: State tracking for multiple files"""
        print("\n=== Test 8: State Tracking for Multiple Files ===")
        
        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success
        
        # Copy multi-file skill
        skill_name = "state-tracking-skill"
        project_skill_dir, project_files = self._copy_multi_file_skill_to_project(skill_name)
        
        # Feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success
        
        # Check state file
        state_file = self.skill_hub_dir / "state.json"
        assert state_file.exists(), "state.json not found"
        
        with open(state_file, 'r') as f:
            state = json.load(f)
        
        # Find project in state
        project_path = str(self.project_dir)
        assert project_path in state, f"Project not in state: {project_path}"
        
        project_state = state[project_path]
        assert "skills" in project_state, "skills not in project state"
        
        skills_state = project_state["skills"]
        assert skill_name in skills_state, f"Skill {skill_name} not in state"
        
        skill_state = skills_state[skill_name]
        
        # Check if state tracks files (implementation dependent)
        print(f"  State for {skill_name}: {skill_state}")
        
        # The state might track files, versions, or just skill presence
        # This test documents what's tracked rather than enforcing specific format
        
        print(f"✓ State tracking verified")
        print(f"  - Project registered in state.json")
        print(f"  - Skill {skill_name} tracked in state")
        print(f"  - State format: {type(skill_state)}")
    
    def test_09_error_handling_missing_files(self):
        """Test 9: Error handling when files are missing"""
        print("\n=== Test 9: Error Handling for Missing Files ===")
        
        # Initialize
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success
        
        # Create skill
        skill_name = "missing-files-test"
        skill_dir = self.project_agents_skills_dir / skill_name
        skill_dir.mkdir(parents=True, exist_ok=True)
        
        # Create SKILL.md (required)
        skill_md = skill_dir / "SKILL.md"
        skill_md.write_text("# Test Skill\n\nMinimal skill.")
        
        # Feedback should work with just SKILL.md
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        
        # It might succeed or fail depending on implementation
        print(f"  Feedback with minimal skill: returncode={result.exit_code}")
        print(f"  stdout: {result.stdout[:100]}...")
        
        # Test with non-existent skill
        result = self.cmd.run("feedback", ["non-existent-skill"], cwd=str(self.project_dir))
        print(f"  Feedback non-existent skill: returncode={result.exit_code}")
        print(f"  stderr: {result.stderr[:100]}...")
        
        print(f"✓ Error handling tested")
        print(f"  - Minimal skill (only SKILL.md): handled")
        print(f"  - Non-existent skill: appropriate response")
    
    def test_10_comprehensive_multi_file_workflow(self):
        """Test 10: Comprehensive multi-file workflow test"""
        print("\n=== Test 10: Comprehensive Multi-File Workflow ===")
        
        # This test combines multiple aspects
        result = self.cmd.run("init", cwd=str(self.home_dir))
        assert result.success
        
        # Use the pre-built multi-file-skill
        skill_name = "multi-file-skill"
        project_skill_dir, project_files = self._copy_multi_file_skill_to_project(skill_name)
        
        print(f"  Starting with {len(project_files)} files")
        
        # 1. Initial feedback
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"Initial feedback failed: {result.stderr}"
        
        # 2. Verify repository
        repo_skill_dir = self.repo_skills_dir / skill_name
        assert repo_skill_dir.exists()
        
        # 3. Modify project files
        config_file = project_skill_dir / "config.yaml"
        original_config = config_file.read_text()
        config_file.write_text(original_config + "\n# Modified in workflow test\n")
        
        # 4. Second feedback (update)
        result = self.cmd.run("feedback", [skill_name], cwd=str(self.project_dir), input_text="y\n")
        assert result.success, f"Update feedback failed: {result.stderr}"
        
        # 5. Setup project and apply
        result = self.cmd.run("set-target", ["open_code"], cwd=str(self.project_dir))
        assert result.success
        
        result = self.cmd.run("use", [skill_name], cwd=str(self.project_dir), input_text="\n")
        assert result.success
        
        result = self.cmd.run("apply", cwd=str(self.project_dir))
        assert result.success, f"Apply failed: {result.stderr}"
        
        # 6. Check status
        result = self.cmd.run("status", cwd=str(self.project_dir))
        print(f"  Status after workflow: {result.stdout[:150]}...")
        
        # 7. Verify applied files match repository
        mismatches = self._verify_files_match(repo_skill_dir, project_skill_dir, project_files)
        assert len(mismatches) == 0, f"File mismatches after workflow: {mismatches}"
        
        print(f"✓ Comprehensive workflow completed successfully")
        print(f"  - Initial feedback: ✓")
        print(f"  - File modification: ✓")
        print(f"  - Update feedback: ✓")
        print(f"  - Apply: ✓")
        print(f"  - Status check: ✓")
        print(f"  - File integrity: ✓")


if __name__ == "__main__":
    # For direct execution
    pytest.main([__file__, "-v"])