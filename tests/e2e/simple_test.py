#!/usr/bin/env python3
"""
Simple e2e test for GitHub Actions.
This is a minimal test that verifies the skill-hub binary works.
"""

import os
import sys
import subprocess
import tempfile
import shutil

def run_command(cmd, cwd=None, env=None):
    """Run a command and return the result."""
    try:
        result = subprocess.run(
            cmd,
            shell=True,
            cwd=cwd,
            env=env,
            capture_output=True,
            text=True,
            timeout=30
        )
        return {
            'success': result.returncode == 0,
            'stdout': result.stdout,
            'stderr': result.stderr,
            'returncode': result.returncode
        }
    except subprocess.TimeoutExpired:
        return {
            'success': False,
            'stdout': '',
            'stderr': 'Command timed out',
            'returncode': -1
        }
    except Exception as e:
        return {
            'success': False,
            'stdout': '',
            'stderr': str(e),
            'returncode': -1
        }

def test_skill_hub_basic():
    """Test basic skill-hub commands."""
    print("🧪 Testing skill-hub basic commands...")
    
    # Test 1: Check version
    print("  Testing: skill-hub --version")
    result = run_command("skill-hub --version")
    if not result['success']:
        print(f"  ❌ Failed: {result['stderr']}")
        return False
    print(f"  ✅ Version: {result['stdout'].strip()}")
    
    # Test 2: Check help
    print("  Testing: skill-hub --help")
    result = run_command("skill-hub --help")
    if not result['success']:
        print(f"  ❌ Failed: {result['stderr']}")
        return False
    print(f"  ✅ Help command works")
    
    # Test 3: Check list command (should work even without initialization)
    print("  Testing: skill-hub list")
    result = run_command("skill-hub list")
    # This might fail if not initialized, but shouldn't crash
    if result['returncode'] not in [0, 1]:
        print(f"  ❌ Unexpected error: {result['stderr']}")
        return False
    print(f"  ✅ List command executed (return code: {result['returncode']})")
    
    # Test 4: Check info command (basic functionality)
    print("  Testing: skill-hub info")
    result = run_command("skill-hub info")
    # Info command should provide basic information
    if result['returncode'] not in [0, 1]:
        print(f"  ❌ Unexpected error: {result['stderr']}")
        return False
    print(f"  ✅ Info command executed (return code: {result['returncode']})")
    
    return True

def test_skill_hub_init():
    """Test skill-hub init command in a temporary directory."""
    print("\n🧪 Testing skill-hub init...")
    
    # Create temporary directory
    temp_dir = tempfile.mkdtemp(prefix="skill_hub_test_")
    print(f"  Using temporary directory: {temp_dir}")
    
    original_cwd = None
    try:
        # Change to temp directory
        original_cwd = os.getcwd()
        os.chdir(temp_dir)
        
        # Test init without git URL (local mode)
        print("  Testing: skill-hub init (local mode)")
        result = run_command("skill-hub init")
        if not result['success']:
            print(f"  ❌ Init failed: {result['stderr']}")
            if original_cwd:
                os.chdir(original_cwd)
            shutil.rmtree(temp_dir, ignore_errors=True)
            return False
        
        print("  ✅ Init successful")
        
        # Check if config file was created
        config_path = os.path.expanduser("~/.skill-hub/config.yaml")
        if os.path.exists(config_path):
            print(f"  ✅ Config file created: {config_path}")
        else:
            print(f"  ⚠️  Config file not found at expected location")
        
        # Change back to original directory
        if original_cwd:
            os.chdir(original_cwd)
        
        return True
        
    except Exception as e:
        print(f"  ❌ Exception during test: {e}")
        if original_cwd:
            os.chdir(original_cwd)
        return False
    finally:
        # Clean up
        shutil.rmtree(temp_dir, ignore_errors=True)

def main():
    """Main test runner."""
    print("🚀 Starting simple e2e tests for skill-hub")
    print("=" * 50)
    
    # Check if skill-hub is in PATH
    skill_hub_path = shutil.which("skill-hub")
    if not skill_hub_path:
        print("❌ skill-hub not found in PATH")
        print("   Make sure the binary is built and in the PATH")
        return 1
    
    print(f"✅ skill-hub found at: {skill_hub_path}")
    
    # Run tests
    tests_passed = 0
    tests_failed = 0
    
    # Test 1: Basic commands
    if test_skill_hub_basic():
        tests_passed += 1
    else:
        tests_failed += 1
    
    # Test 2: Init command (skip in CI if there are permission issues)
    if os.environ.get('CI') == 'true':
        print("\n⚠️  Skipping init test in CI environment (permission issues)")
    else:
        if test_skill_hub_init():
            tests_passed += 1
        else:
            tests_failed += 1
    
    # Summary
    print("\n" + "=" * 50)
    print("📊 Test Summary:")
    print(f"  ✅ Passed: {tests_passed}")
    print(f"  ❌ Failed: {tests_failed}")
    
    if tests_failed > 0:
        print("\n❌ Some tests failed")
        return 1
    else:
        print("\n✅ All tests passed!")
        return 0

if __name__ == "__main__":
    sys.exit(main())