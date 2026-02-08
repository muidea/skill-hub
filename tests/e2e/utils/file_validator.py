import os
import difflib
from pathlib import Path
from typing import Optional, List, Dict, Any

class FileValidator:
    """文件内容和结构验证工具（完全匹配）"""
    
    def __init__(self, strict: bool = True):
        self.strict = strict  # 严格模式：完全匹配
    
    def assert_file_exists(self, path: str, msg: Optional[str] = None):
        """断言文件存在"""
        if not os.path.exists(path):
            raise AssertionError(msg or f"文件不存在: {path}")
    
    def assert_directory_exists(self, path: str, msg: Optional[str] = None):
        """断言目录存在"""
        if not os.path.isdir(path):
            raise AssertionError(msg or f"目录不存在: {path}")
    
    def assert_file_content_exact(self, path: str, expected_content: str, msg: Optional[str] = None):
        """
        断言文件内容完全匹配
        
        Args:
            path: 文件路径
            expected_content: 期望的内容
            msg: 自定义错误信息
        """
        self.assert_file_exists(path)
        
        try:
            with open(path, 'r', encoding='utf-8') as f:
                actual_content = f.read()
        except UnicodeDecodeError:
            # 如果是二进制文件，用二进制模式读取
            with open(path, 'rb') as f:
                actual_content = f.read().decode('utf-8', errors='ignore')
        
        if actual_content != expected_content:
            # 提供详细的差异信息
            diff = self._generate_diff(expected_content, actual_content)
            error_msg = msg or f"文件内容不匹配: {path}\n{diff}"
            raise AssertionError(error_msg)
    
    def assert_file_contains(self, path: str, expected_text: str, msg: Optional[str] = None):
        """
        断言文件包含特定文本（非严格模式）
        
        Args:
            path: 文件路径
            expected_text: 期望包含的文本
            msg: 自定义错误信息
        """
        self.assert_file_exists(path)
        
        with open(path, 'r', encoding='utf-8') as f:
            actual_content = f.read()
        
        if expected_text not in actual_content:
            raise AssertionError(msg or f"文件不包含预期文本: {path}\n预期: {expected_text}")
    
    def assert_file_not_contains(self, path: str, unexpected_text: str, msg: Optional[str] = None):
        """
        断言文件不包含特定文本
        
        Args:
            path: 文件路径
            unexpected_text: 不应包含的文本
            msg: 自定义错误信息
        """
        self.assert_file_exists(path)
        
        with open(path, 'r', encoding='utf-8') as f:
            actual_content = f.read()
        
        if unexpected_text in actual_content:
            raise AssertionError(msg or f"文件包含不应有的文本: {path}\n文本: {unexpected_text}")
    
    def assert_directory_structure(self, dir_path: str, expected_structure: Dict[str, Any], msg: Optional[str] = None):
        """
        断言目录结构
        
        Args:
            dir_path: 目录路径
            expected_structure: 期望的目录结构
                {
                    "file1.txt": True,  # 文件必须存在
                    "subdir": {         # 子目录
                        "file2.txt": True
                    }
                }
            msg: 自定义错误信息
        """
        self.assert_directory_exists(dir_path)
        
        def check_structure(current_path: str, structure: Dict[str, Any], prefix: str = ""):
            for name, expected in structure.items():
                full_path = os.path.join(current_path, name)
                
                if isinstance(expected, dict):
                    # 期望是子目录
                    if not os.path.isdir(full_path):
                        raise AssertionError(f"{prefix}子目录不存在: {name}")
                    # 递归检查子目录结构
                    check_structure(full_path, expected, prefix + "  ")
                elif expected is True:
                    # 期望是文件
                    if not os.path.isfile(full_path):
                        raise AssertionError(f"{prefix}文件不存在: {name}")
                elif expected is False:
                    # 期望不存在
                    if os.path.exists(full_path):
                        raise AssertionError(f"{prefix}文件/目录不应存在: {name}")
        
        check_structure(dir_path, expected_structure)
    
    def assert_yaml_structure(self, yaml_path: str, expected_structure: Dict[str, Any], msg: Optional[str] = None):
        """
        断言YAML文件结构
        
        Args:
            yaml_path: YAML文件路径
            expected_structure: 期望的结构
            msg: 自定义错误信息
        """
        self.assert_file_exists(yaml_path)
        
        try:
            import yaml
            with open(yaml_path, 'r', encoding='utf-8') as f:
                content = yaml.safe_load(f)
        except ImportError:
            raise AssertionError("PyYAML未安装，无法验证YAML结构")
        except Exception as e:
            raise AssertionError(f"YAML解析失败: {e}")
        
        self._assert_dict_structure(content, expected_structure, "YAML")
    
    def assert_skill_structure(self, skill_dir: str, msg: Optional[str] = None):
        """
        断言技能目录结构完整
        
        Args:
            skill_dir: 技能目录路径
            msg: 自定义错误信息
        """
        self.assert_directory_exists(skill_dir)
        
        # 检查必要的文件
        required_files = ["SKILL.md"]
        for file in required_files:
            file_path = os.path.join(skill_dir, file)
            self.assert_file_exists(file_path, f"技能缺少必要文件: {file}")
        
        # 检查SKILL.md的前言部分
        skill_md_path = os.path.join(skill_dir, "SKILL.md")
        with open(skill_md_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # 检查是否包含YAML前言
        if not content.startswith('---'):
            raise AssertionError("SKILL.md缺少YAML前言")
        
        # 检查必要的元数据字段
        required_fields = ["name", "description"]
        for field in required_fields:
            if f"{field}:" not in content:
                raise AssertionError(f"SKILL.md缺少必要字段: {field}")
    
    def _generate_diff(self, expected: str, actual: str) -> str:
        """生成详细的差异信息"""
        expected_lines = expected.splitlines(keepends=True)
        actual_lines = actual.splitlines(keepends=True)
        
        diff = difflib.unified_diff(
            expected_lines, actual_lines,
            fromfile='expected', tofile='actual',
            lineterm=''
        )
        return '\n'.join(diff)
    
    def _assert_dict_structure(self, actual: Dict[str, Any], expected: Dict[str, Any], path: str = ""):
        """递归断言字典结构"""
        for key, expected_value in expected.items():
            full_path = f"{path}.{key}" if path else key
            
            if key not in actual:
                raise AssertionError(f"缺少键: {full_path}")
            
            actual_value = actual[key]
            
            if isinstance(expected_value, dict):
                if not isinstance(actual_value, dict):
                    raise AssertionError(f"{full_path} 应为字典，实际为 {type(actual_value)}")
                self._assert_dict_structure(actual_value, expected_value, full_path)
            elif isinstance(expected_value, type):
                # expected_value是类型，如 str, int, list
                if not isinstance(actual_value, expected_value):
                    raise AssertionError(f"{full_path} 应为 {expected_value.__name__}，实际为 {type(actual_value).__name__}")
            else:
                # expected_value是具体值
                if actual_value != expected_value:
                    raise AssertionError(f"{full_path} 值不匹配: 期望 {expected_value}，实际 {actual_value}")