#!/usr/bin/env python3
"""
Main test runner for skill-hub end-to-end tests.
"""

import os
import sys
import argparse
import subprocess
import tempfile
from pathlib import Path

# Change to project root directory
project_root = Path(__file__).parent.parent.parent
os.chdir(project_root)

# Add the project root to Python path
sys.path.insert(0, str(project_root))

from tests.e2e.utils.command_runner import CommandRunner
from tests.e2e.utils.network_checker import NetworkChecker
from tests.e2e.utils.debug_utils import DebugUtils


def check_environment():
    """Check if the test environment is properly set up."""
    print("🔍 Checking test environment...")
    
    checks = []
    
    # Check Python version
    python_version = sys.version_info
    checks.append(("Python version", f"{python_version.major}.{python_version.minor}.{python_version.micro}"))
    
    # Check if skill-hub command is available
    try:
        cmd = CommandRunner(timeout=5)
        # Try to run a simple command to check if skill-hub is available
        result = cmd.run("--version")
        skill_hub_available = result.success
        checks.append(("skill-hub command", "Available" if skill_hub_available else "NOT FOUND"))
        
        if skill_hub_available:
            checks.append(("skill-hub version", result.stdout.strip()))
    except Exception as e:
        checks.append(("skill-hub command", f"NOT FOUND: {str(e)}"))
        skill_hub_available = False
    
    # Check network connectivity
    network_available = NetworkChecker.is_network_available()
    checks.append(("Network connectivity", "Available" if network_available else "Unavailable"))
    
    # Check test directory structure
    test_dir = Path(__file__).parent
    required_dirs = [
        ("utils", test_dir / "utils"),
        ("fixtures", test_dir / "fixtures"),
        ("data", test_dir / "data"),
        ("data/test_skills", test_dir / "data" / "test_skills"),
    ]
    
    for name, dir_path in required_dirs:
        exists = dir_path.exists()
        checks.append((f"Directory: {name}", "Exists" if exists else "MISSING"))
    
    # Print check results
    print("\n" + "="*60)
    print("ENVIRONMENT CHECK RESULTS")
    print("="*60)
    
    all_passed = True
    for check_name, status in checks:
        passed = "MISSING" not in status and "NOT FOUND" not in status
        status_symbol = "✅" if passed else "❌"
        print(f"{status_symbol} {check_name:30} {status}")
        if not passed:
            all_passed = False
    
    print("="*60)
    
    if not all_passed:
        print("\n⚠️  WARNING: Some environment checks failed!")
        print("Tests may not run correctly.")
        response = input("Continue anyway? (y/N): ")
        if response.lower() != 'y':
            print("Exiting.")
            sys.exit(1)
    
    return all_passed


def run_tests(scenarios=None, verbose=False, debug=False):
    """Run the specified test scenarios."""
    print("\n🚀 Running skill-hub End-to-End Tests")
    print("="*60)
    
    # Default to all scenarios if none specified
    if scenarios is None:
        scenarios = [1, 2, 3, 4, 5]
    
    test_files = []
    for scenario_num in scenarios:
        test_file = Path(__file__).parent / f"test_scenario{scenario_num}.py"
        if test_file.exists():
            test_files.append(test_file)
        else:
            print(f"⚠️  Test file not found: {test_file}")
    
    if not test_files:
        print("❌ No test files found to run!")
        return False
    
    print(f"📋 Running {len(test_files)} test scenario(s): {scenarios}")
    
    # Build pytest command
    pytest_cmd = [
        sys.executable, "-m", "pytest",
        "-v",  # Verbose output
        "--tb=short",  # Short traceback
    ]
    
    if verbose:
        pytest_cmd.append("-s")  # Don't capture stdout/stderr
    
    if debug:
        pytest_cmd.append("--pdb")  # Enter debugger on failure
    
    # Add test files
    pytest_cmd.extend([str(f) for f in test_files])
    
    # Add pytest.ini location
    pytest_cmd.extend(["-c", str(Path(__file__).parent / "pytest.ini")])
    
    print(f"\n📝 Command: {' '.join(pytest_cmd)}")
    print("="*60 + "\n")
    
    # Run tests
    try:
        result = subprocess.run(pytest_cmd, check=False)
        return result.returncode == 0
    except KeyboardInterrupt:
        print("\n\n⏹️  Tests interrupted by user.")
        return False
    except Exception as e:
        print(f"\n❌ Error running tests: {e}")
        return False


def run_single_test(test_name, verbose=False, debug=False):
    """Run a single test by name."""
    print(f"\n🔬 Running single test: {test_name}")
    
    # Build pytest command for specific test
    pytest_cmd = [
        sys.executable, "-m", "pytest",
        "-v",
        "--tb=short",
        "-k", test_name,  # Run tests matching this name
    ]
    
    if verbose:
        pytest_cmd.append("-s")
    
    if debug:
        pytest_cmd.append("--pdb")
    
    # Add test directory
    pytest_cmd.append(str(Path(__file__).parent))
    
    print(f"📝 Command: {' '.join(pytest_cmd)}")
    print("="*60 + "\n")
    
    try:
        result = subprocess.run(pytest_cmd, check=False)
        return result.returncode == 0
    except KeyboardInterrupt:
        print("\n\n⏹️  Test interrupted by user.")
        return False


def list_tests():
    """List all available tests."""
    print("\n📋 Available Test Scenarios:")
    print("="*60)
    
    test_dir = Path(__file__).parent
    
    # List scenario files
    scenario_files = list(test_dir.glob("test_scenario*.py"))
    
    if not scenario_files:
        print("No test scenarios found!")
        return
    
    for test_file in sorted(scenario_files):
        # Extract scenario number from filename
        scenario_num = test_file.stem.replace("test_scenario", "")
        
        # Read first few lines to get description
        with open(test_file, 'r') as f:
            first_line = f.readline().strip()
            description = first_line.replace('"""', '').strip()
        
        print(f"Scenario {scenario_num}: {description}")
    
    print("\n📋 Available Test Classes:")
    print("="*60)
    
    # Try to discover test classes (simplified)
    for test_file in scenario_files:
        with open(test_file, 'r') as f:
            content = f.read()
            # Look for class definitions
            import re
            class_matches = re.findall(r'class (\w+)\s*\(', content)
            for class_name in class_matches:
                if 'Test' in class_name:
                    print(f"  {class_name} in {test_file.name}")


def cleanup_temp_files():
    """Clean up temporary test files."""
    print("\n🧹 Cleaning up temporary files...")
    
    # Clean up any leftover temporary directories
    temp_dir = Path(tempfile.gettempdir())
    skill_hub_temp_dirs = list(temp_dir.glob("skill_hub_test_*"))
    
    if skill_hub_temp_dirs:
        print(f"Found {len(skill_hub_temp_dirs)} temporary directories to clean up.")
        
        for temp_dir_path in skill_hub_temp_dirs:
            try:
                import shutil
                if temp_dir_path.exists():
                    shutil.rmtree(temp_dir_path)
                    print(f"  Removed: {temp_dir_path}")
            except Exception as e:
                print(f"  Failed to remove {temp_dir_path}: {e}")
    else:
        print("No temporary directories found to clean up.")


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description="Run skill-hub end-to-end tests",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s                    # Run all tests
  %(prog)s -s 1 3 5           # Run scenarios 1, 3, and 5
  %(prog)s --test TestScenario1DeveloperWorkflow  # Run specific test class
  %(prog)s --list             # List available tests
  %(prog)s --check            # Check environment only
  %(prog)s --cleanup          # Clean up temporary files
        """
    )
    
    parser.add_argument(
        "-s", "--scenarios",
        nargs="+",
        type=int,
        choices=[1, 2, 3, 4, 5],
        help="Run specific test scenarios (1-5)"
    )
    
    parser.add_argument(
        "-t", "--test",
        help="Run a specific test by name (e.g., TestScenario1DeveloperWorkflow)"
    )
    
    parser.add_argument(
        "-l", "--list",
        action="store_true",
        help="List available tests"
    )
    
    parser.add_argument(
        "-c", "--check",
        action="store_true",
        help="Check environment only (don't run tests)"
    )
    
    parser.add_argument(
        "--cleanup",
        action="store_true",
        help="Clean up temporary test files"
    )
    
    parser.add_argument(
        "-v", "--verbose",
        action="store_true",
        help="Verbose output"
    )
    
    parser.add_argument(
        "-d", "--debug",
        action="store_true",
        help="Enter debugger on test failure"
    )
    
    parser.add_argument(
        "--no-check",
        action="store_true",
        help="Skip environment check"
    )
    
    args = parser.parse_args()
    
    print("🎯 skill-hub End-to-End Test Runner")
    print("="*60)
    
    # Handle cleanup
    if args.cleanup:
        cleanup_temp_files()
        return 0
    
    # Handle list
    if args.list:
        list_tests()
        return 0
    
    # Check environment (unless disabled)
    if not args.no_check and not args.check:
        environment_ok = check_environment()
        if not environment_ok and not args.verbose:
            print("\n⚠️  Environment issues detected. Use --verbose for details.")
    
    # If only checking environment
    if args.check:
        return 0
    
    # Run tests
    success = False
    
    if args.test:
        # Run specific test
        success = run_single_test(args.test, args.verbose, args.debug)
    else:
        # Run scenarios
        success = run_tests(args.scenarios, args.verbose, args.debug)
    
    # Print summary
    print("\n" + "="*60)
    if success:
        print("✅ All tests passed!")
        return 0
    else:
        print("❌ Some tests failed.")
        return 1


if __name__ == "__main__":
    sys.exit(main())