#!/usr/bin/env python3
"""
Helper utilities for multi-file-skill
"""

import os
import sys
import json
import yaml
from pathlib import Path
from typing import Dict, Any, List, Optional
import hashlib
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


def load_config(config_path: str = "config.yaml") -> Dict[str, Any]:
    """
    Load configuration from YAML file
    
    Args:
        config_path: Path to configuration file
        
    Returns:
        Dictionary with configuration
    """
    try:
        with open(config_path, 'r') as f:
            config = yaml.safe_load(f)
        logger.info(f"Loaded configuration from {config_path}")
        return config or {}
    except FileNotFoundError:
        logger.warning(f"Configuration file not found: {config_path}")
        return {}
    except yaml.YAMLError as e:
        logger.error(f"Error parsing YAML configuration: {e}")
        return {}


def save_config(config: Dict[str, Any], config_path: str = "config.yaml") -> bool:
    """
    Save configuration to YAML file
    
    Args:
        config: Configuration dictionary
        config_path: Path to save configuration
        
    Returns:
        True if successful, False otherwise
    """
    try:
        with open(config_path, 'w') as f:
            yaml.dump(config, f, default_flow_style=False)
        logger.info(f"Saved configuration to {config_path}")
        return True
    except Exception as e:
        logger.error(f"Error saving configuration: {e}")
        return False


def calculate_file_hash(file_path: str) -> str:
    """
    Calculate SHA256 hash of a file
    
    Args:
        file_path: Path to file
        
    Returns:
        SHA256 hash string
    """
    sha256_hash = hashlib.sha256()
    try:
        with open(file_path, "rb") as f:
            # Read file in chunks to handle large files
            for byte_block in iter(lambda: f.read(4096), b""):
                sha256_hash.update(byte_block)
        return sha256_hash.hexdigest()
    except FileNotFoundError:
        logger.error(f"File not found: {file_path}")
        return ""
    except Exception as e:
        logger.error(f"Error calculating hash for {file_path}: {e}")
        return ""


def list_files(directory: str, pattern: str = "**/*") -> List[str]:
    """
    List all files in directory matching pattern
    
    Args:
        directory: Directory to search
        pattern: Glob pattern to match
        
    Returns:
        List of file paths relative to directory
    """
    files = []
    try:
        dir_path = Path(directory)
        for file_path in dir_path.glob(pattern):
            if file_path.is_file():
                # Get relative path
                rel_path = file_path.relative_to(dir_path)
                files.append(str(rel_path))
    except Exception as e:
        logger.error(f"Error listing files in {directory}: {e}")
    
    return files


def create_directory_structure(base_dir: str, structure: Dict[str, Any]) -> bool:
    """
    Create directory structure
    
    Args:
        base_dir: Base directory
        structure: Dictionary defining directory structure
        
    Returns:
        True if successful, False otherwise
    """
    try:
        base_path = Path(base_dir)
        
        def create_structure(current_path: Path, current_structure: Dict[str, Any]):
            for name, content in current_structure.items():
                item_path = current_path / name
                
                if isinstance(content, dict):
                    # It's a directory
                    item_path.mkdir(exist_ok=True, parents=True)
                    create_structure(item_path, content)
                else:
                    # It's a file
                    item_path.parent.mkdir(exist_ok=True, parents=True)
                    if content is not None:
                        item_path.write_text(str(content))
        
        create_structure(base_path, structure)
        logger.info(f"Created directory structure in {base_dir}")
        return True
    except Exception as e:
        logger.error(f"Error creating directory structure: {e}")
        return False


def validate_file_permissions(file_path: str, expected_permissions: int = 0o644) -> bool:
    """
    Validate file permissions
    
    Args:
        file_path: Path to file
        expected_permissions: Expected permissions (octal)
        
    Returns:
        True if permissions match, False otherwise
    """
    try:
        stat_info = os.stat(file_path)
        actual_permissions = stat_info.st_mode & 0o777
        
        if actual_permissions == expected_permissions:
            return True
        else:
            logger.warning(
                f"File permissions mismatch for {file_path}: "
                f"expected {oct(expected_permissions)}, got {oct(actual_permissions)}"
            )
            return False
    except FileNotFoundError:
        logger.error(f"File not found: {file_path}")
        return False
    except Exception as e:
        logger.error(f"Error checking permissions for {file_path}: {e}")
        return False


def main():
    """Main function for testing helper utilities"""
    print("Testing helper utilities...")
    
    # Test configuration loading
    config = load_config()
    print(f"Loaded config: {json.dumps(config, indent=2)}")
    
    # Test file listing
    files = list_files(".", "*.py")
    print(f"Python files in current directory: {files}")
    
    # Test hash calculation
    if files:
        file_hash = calculate_file_hash(files[0])
        print(f"SHA256 hash of {files[0]}: {file_hash}")
    
    print("Helper utilities test completed.")


if __name__ == "__main__":
    main()