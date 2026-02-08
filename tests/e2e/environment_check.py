#!/usr/bin/env python3
"""
Environment check script for Skill Hub end-to-end tests.
"""

import os
import sys
import subprocess
import platform
from pathlib import Path

# Change to project root directory
project_root = Path(__file__).parent.parent.parent
os.chdir(project_root)

# Add the project root to Python path
sys.path.insert(0, str(project_root))


def check_python_version():
    """Check Python version."""
    version = sys.version_info
    print(f"Python: {version.major}.{version.minor}.{version.micro}")
    
    if version.major == 3 and version.minor >= 8:
        return True, "✅ Python 3.8+ detected"
    else:
        return False, f"❌ Python 3.8+ required, found {version.major}.{version.minor}"


def check_skill_hub_command():
    """Check if skill-hub command is available."""
    try:
        result = subprocess.run(
            ["which", "skill-hub"],
            capture_output=True,
            text=True,
            timeout=5
        )
        
        if result.returncode == 0:
            skill_hub_path = result.stdout.strip()
            
            # Check version
            version_result = subprocess.run(
                ["skill-hub", "--version"],
                capture_output=True,
                text=True,
                timeout=5
            )
            
            if version_result.returncode == 0:
                version = version_result.stdout.strip()
                return True, f"✅ skill-hub found at {skill_hub_path} (version: {version})"
            else:
                return True, f"✅ skill-hub found at {skill_hub_path} (version unknown)"
        else:
            return False, "❌ skill-hub command not found in PATH"
            
    except subprocess.TimeoutExpired:
        return False, "❌ Timeout checking skill-hub command"
    except Exception as e:
        return False, f"❌ Error checking skill-hub: {e}"


def check_test_directories():
    """Check if test directory structure exists."""
    test_dir = Path(__file__).parent
    required_dirs = [
        ("utils", test_dir / "utils"),
        ("fixtures", test_dir / "fixtures"),
        ("data", test_dir / "data"),
        ("data/test_skills", test_dir / "data" / "test_skills"),
    ]
    
    results = []
    all_exist = True
    
    for name, dir_path in required_dirs:
        exists = dir_path.exists() and dir_path.is_dir()
        results.append((name, exists))
        if not exists:
            all_exist = False
    
    if all_exist:
        return True, "✅ All test directories exist"
    else:
        missing = [name for name, exists in results if not exists]
        return False, f"❌ Missing directories: {', '.join(missing)}"


def check_python_dependencies():
    """Check Python dependencies."""
    required_packages = [
        ("pytest", "pytest"),
        ("yaml", "pyyaml"),  # Package is yaml, commonly called pyyaml
        ("requests", "requests"),
    ]
    
    missing = []
    
    for package, display_name in required_packages:
        try:
            __import__(package.replace("-", "_"))
        except ImportError:
            missing.append(display_name)
    
    if not missing:
        return True, "✅ All Python dependencies available"
    else:
        return False, f"❌ Missing Python packages: {', '.join(missing)}"


def check_network_connectivity():
    """Check network connectivity."""
    try:
        import socket
        socket.create_connection(("8.8.8.8", 53), timeout=3)
        return True, "✅ Network connectivity available"
    except OSError:
        return False, "⚠️  Network connectivity unavailable (some tests may be skipped)"


def check_file_permissions():
    """Check file permissions for test directories."""
    test_dir = Path(__file__).parent
    
    try:
        # Try to create a test file
        test_file = test_dir / ".permission_test"
        test_file.touch()
        test_file.unlink()
        
        return True, "✅ File permissions OK"
    except PermissionError:
        return False, "❌ Permission denied in test directory"
    except Exception as e:
        return False, f"❌ File permission error: {e}"


def check_system_info():
    """Display system information."""
    print(f"Platform: {platform.system()} {platform.release()}")
    print(f"Architecture: {platform.machine()}")
    print(f"Working directory: {os.getcwd()}")
    print(f"Python executable: {sys.executable}")
    print()


def run_all_checks():
    """Run all environment checks."""
    print("🔍 Skill Hub Test Environment Check")
    print("="*60)
    
    check_system_info()
    
    checks = [
        ("Python Version", check_python_version()),
        ("skill-hub Command", check_skill_hub_command()),
        ("Test Directories", check_test_directories()),
        ("Python Dependencies", check_python_dependencies()),
        ("Network Connectivity", check_network_connectivity()),
        ("File Permissions", check_file_permissions()),
    ]
    
    print("CHECK RESULTS:")
    print("="*60)
    
    all_passed = True
    for check_name, (passed, message) in checks:
        symbol = "✅" if passed else "❌"
        if not passed and "⚠️" not in message:
            all_passed = False
        print(f"{symbol} {check_name:25} {message}")
    
    print("="*60)
    
    if all_passed:
        print("\n🎉 All environment checks passed! Ready to run tests.")
        return True
    else:
        print("\n⚠️  Some environment checks failed or have warnings.")
        print("   Some tests may fail or be skipped.")
        return False


def generate_setup_instructions():
    """Generate setup instructions based on failed checks."""
    print("\n📝 SETUP INSTRUCTIONS:")
    print("="*60)
    
    instructions = []
    
    # Python version
    version = sys.version_info
    if version.major < 3 or (version.major == 3 and version.minor < 8):
        instructions.append("""
Python 3.8+ Required:
  - Install Python 3.8 or newer
  - Update your PATH to include the new Python
  - Verify with: python3 --version
        """)
    
    # skill-hub command
    try:
        subprocess.run(["which", "skill-hub"], capture_output=True, check=False)
    except:
        instructions.append("""
skill-hub Command Missing:
  - Build skill-hub from source: go build -o skill-hub
  - Or install via package manager if available
  - Add to PATH or use full path in tests
        """)
    
    # Test directories
    test_dir = Path(__file__).parent
    if not (test_dir / "utils").exists():
        instructions.append("""
Test Directories Missing:
  - The test framework needs to be set up
  - Run the test setup script or create directories manually
        """)
    
    # Python dependencies
    missing_deps = []
    for package in ["pytest", "pyyaml", "requests"]:
        try:
            __import__(package.replace("-", "_"))
        except ImportError:
            missing_deps.append(package)
    
    if missing_deps:
        instructions.append(f"""
Missing Python Dependencies:
  - Install missing packages: pip install {' '.join(missing_deps)}
  - Or: pip install -r tests/e2e/requirements.txt
        """)
    
    if instructions:
        for instruction in instructions:
            print(instruction.strip())
            print()
    else:
        print("All dependencies appear to be satisfied!")
    
    print("="*60)


def main():
    """Main entry point."""
    import argparse
    
    parser = argparse.ArgumentParser(description="Check test environment")
    parser.add_argument("--fix", action="store_true", help="Show setup instructions")
    parser.add_argument("--quiet", action="store_true", help="Minimal output")
    
    args = parser.parse_args()
    
    if not args.quiet:
        print("🎯 Skill Hub Test Environment Checker")
        print()
    
    environment_ok = run_all_checks()
    
    if args.fix or not environment_ok:
        generate_setup_instructions()
    
    return 0 if environment_ok else 1


if __name__ == "__main__":
    sys.exit(main())