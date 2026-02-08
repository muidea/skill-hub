import os
import yaml
from typing import Dict, Any, List, Optional, Union
from pathlib import Path

class YAMLValidator:
    """专门的YAML文件验证工具"""
    
    def __init__(self, strict: bool = True):
        """
        初始化YAML验证器
        
        Args:
            strict: 严格模式，验证失败时抛出异常
        """
        self.strict = strict
    
    def load_yaml(self, file_path: str) -> Dict[str, Any]:
        """
        加载YAML文件
        
        Args:
            file_path: YAML文件路径
            
        Returns:
            Dict: 解析后的YAML内容
            
        Raises:
            FileNotFoundError: 文件不存在
            yaml.YAMLError: YAML解析错误
        """
        if not os.path.exists(file_path):
            raise FileNotFoundError(f"YAML文件不存在: {file_path}")
        
        with open(file_path, 'r', encoding='utf-8') as f:
            try:
                content = yaml.safe_load(f)
                if content is None:
                    return {}
                return content
            except yaml.YAMLError as e:
                error_msg = f"YAML解析失败: {file_path}\n错误: {e}"
                if self.strict:
                    raise yaml.YAMLError(error_msg)
                else:
                    print(f"警告: {error_msg}")
                    return {}
    
    def validate_skill_yaml(self, yaml_path: str, skill_id: Optional[str] = None) -> Dict[str, Any]:
        """
        验证skill.yaml文件格式
        
        Args:
            yaml_path: skill.yaml文件路径
            skill_id: 期望的技能ID（用于验证）
            
        Returns:
            Dict: 验证结果
            
        Raises:
            AssertionError: 验证失败（严格模式下）
        """
        try:
            content = self.load_yaml(yaml_path)
        except (FileNotFoundError, yaml.YAMLError) as e:
            return {
                "valid": False,
                "errors": [str(e)],
                "warnings": []
            }
        
        errors = []
        warnings = []
        
        # 必需字段检查
        required_fields = ["name", "description", "version"]
        for field in required_fields:
            if field not in content:
                errors.append(f"缺少必需字段: {field}")
        
        # 名称格式检查
        if "name" in content:
            name = content["name"]
            if not isinstance(name, str):
                errors.append("name字段应为字符串")
            elif skill_id and name != skill_id:
                warnings.append(f"技能名称不匹配: 期望 '{skill_id}', 实际 '{name}'")
        
        # 描述检查
        if "description" in content:
            if not isinstance(content["description"], str):
                errors.append("description字段应为字符串")
            elif len(content["description"]) < 10:
                warnings.append("描述可能过短")
        
        # 版本格式检查
        if "version" in content:
            version = content["version"]
            if not isinstance(version, str):
                errors.append("version字段应为字符串")
            elif not self._is_valid_version(version):
                warnings.append(f"版本格式可能无效: {version}")
        
        # 变量检查
        if "variables" in content:
            variables = content["variables"]
            if not isinstance(variables, list):
                errors.append("variables字段应为列表")
            else:
                for i, var in enumerate(variables):
                    var_errors = self._validate_variable(var, i)
                    errors.extend(var_errors)
        
        # 兼容性检查
        if "compatibility" in content:
            compat = content["compatibility"]
            valid_targets = ["cursor", "claude_code", "open_code", "all"]
            if isinstance(compat, str):
                if compat.lower() not in valid_targets and "designed for" not in compat.lower():
                    warnings.append(f"兼容性声明可能无效: {compat}")
        
        # 元数据检查
        if "metadata" in content:
            metadata = content["metadata"]
            if not isinstance(metadata, dict):
                errors.append("metadata字段应为字典")
            else:
                if "author" in metadata and not isinstance(metadata["author"], str):
                    errors.append("metadata.author字段应为字符串")
                if "tags" in metadata:
                    if not isinstance(metadata["tags"], list):
                        errors.append("metadata.tags字段应为列表")
                    elif metadata["tags"]:
                        for tag in metadata["tags"]:
                            if not isinstance(tag, str):
                                errors.append("metadata.tags中的标签应为字符串")
        
        result = {
            "valid": len(errors) == 0,
            "errors": errors,
            "warnings": warnings,
            "content": content
        }
        
        if self.strict and errors:
            error_msg = f"skill.yaml验证失败: {yaml_path}\n" + "\n".join(errors)
            raise AssertionError(error_msg)
        
        return result
    
    def validate_manifest_yaml(self, manifest_path: str) -> Dict[str, Any]:
        """
        验证OpenCode的manifest.yaml
        
        Args:
            manifest_path: manifest.yaml文件路径
            
        Returns:
            Dict: 验证结果
        """
        try:
            content = self.load_yaml(manifest_path)
        except (FileNotFoundError, yaml.YAMLError) as e:
            return {
                "valid": False,
                "errors": [str(e)],
                "warnings": []
            }
        
        errors = []
        warnings = []
        
        # OpenCode manifest基本字段
        required_fields = ["name", "version"]
        for field in required_fields:
            if field not in content:
                errors.append(f"缺少必需字段: {field}")
        
        # 名称检查
        if "name" in content:
            name = content["name"]
            if not isinstance(name, str):
                errors.append("name字段应为字符串")
            elif not name.replace("-", "").replace("_", "").isalnum():
                warnings.append(f"名称包含特殊字符: {name}")
        
        # 版本检查
        if "version" in content:
            version = content["version"]
            if not isinstance(version, str):
                errors.append("version字段应为字符串")
        
        # 描述检查（可选但推荐）
        if "description" in content:
            if not isinstance(content["description"], str):
                errors.append("description字段应为字符串")
        
        # 工具定义检查
        if "tools" in content:
            tools = content["tools"]
            if not isinstance(tools, list):
                errors.append("tools字段应为列表")
            else:
                for i, tool in enumerate(tools):
                    if not isinstance(tool, dict):
                        errors.append(f"tools[{i}]应为字典")
                    else:
                        if "name" not in tool:
                            errors.append(f"tools[{i}]缺少name字段")
                        if "description" not in tool:
                            warnings.append(f"tools[{i}]缺少description字段")
        
        result = {
            "valid": len(errors) == 0,
            "errors": errors,
            "warnings": warnings,
            "content": content
        }
        
        if self.strict and errors:
            error_msg = f"manifest.yaml验证失败: {manifest_path}\n" + "\n".join(errors)
            raise AssertionError(error_msg)
        
        return result
    
    def _validate_variable(self, variable: Any, index: int) -> List[str]:
        """验证单个变量定义"""
        errors = []
        
        if not isinstance(variable, dict):
            return [f"variables[{index}]应为字典"]
        
        # 必需字段
        if "name" not in variable:
            errors.append(f"variables[{index}]缺少name字段")
        else:
            name = variable["name"]
            if not isinstance(name, str):
                errors.append(f"variables[{index}].name应为字符串")
            elif not name.replace("_", "").isalnum():
                errors.append(f"variables[{index}].name包含无效字符: {name}")
        
        if "description" not in variable:
            warnings = []  # 注意：这里应该是errors，但为了保持一致性
            # 描述是可选的，但推荐有
            pass
        
        # 默认值检查
        if "default" in variable:
            default = variable["default"]
            if not isinstance(default, (str, int, float, bool)):
                errors.append(f"variables[{index}].default应为基本类型")
        
        return errors
    
    def _is_valid_version(self, version: str) -> bool:
        """检查版本号格式"""
        import re
        # 简单的版本号检查：x.y.z 或 vx.y.z
        pattern = r'^v?\d+(\.\d+){0,2}(-[a-zA-Z0-9]+)?$'
        return bool(re.match(pattern, version))
    
    def compare_yaml(self, file1: str, file2: str, ignore_fields: Optional[List[str]] = None) -> Dict[str, Any]:
        """
        比较两个YAML文件
        
        Args:
            file1: 第一个YAML文件
            file2: 第二个YAML文件
            ignore_fields: 忽略比较的字段
            
        Returns:
            Dict: 比较结果
        """
        ignore_fields = ignore_fields or []
        
        try:
            yaml1 = self.load_yaml(file1)
            yaml2 = self.load_yaml(file2)
        except Exception as e:
            return {
                "equal": False,
                "error": str(e),
                "differences": []
            }
        
        differences = self._find_yaml_differences(yaml1, yaml2, ignore_fields)
        
        return {
            "equal": len(differences) == 0,
            "differences": differences,
            "file1": yaml1,
            "file2": yaml2
        }
    
    def _find_yaml_differences(self, dict1: Dict, dict2: Dict, ignore_fields: List[str], path: str = "") -> List[str]:
        """递归查找YAML差异"""
        differences = []
        
        # 合并所有键
        all_keys = set(dict1.keys()) | set(dict2.keys())
        
        for key in all_keys:
            if key in ignore_fields:
                continue
                
            current_path = f"{path}.{key}" if path else key
            
            if key in dict1 and key not in dict2:
                differences.append(f"{current_path}: 只在第一个文件中存在")
            elif key not in dict1 and key in dict2:
                differences.append(f"{current_path}: 只在第二个文件中存在")
            else:
                val1 = dict1[key]
                val2 = dict2[key]
                
                if isinstance(val1, dict) and isinstance(val2, dict):
                    # 递归比较字典
                    sub_diffs = self._find_yaml_differences(val1, val2, ignore_fields, current_path)
                    differences.extend(sub_diffs)
                elif val1 != val2:
                    differences.append(f"{current_path}: 值不同 - {val1} != {val2}")
        
        return differences
    
    def generate_yaml_template(self, skill_id: str, description: str = "") -> Dict[str, Any]:
        """
        生成skill.yaml模板
        
        Args:
            skill_id: 技能ID
            description: 技能描述
            
        Returns:
            Dict: YAML模板
        """
        return {
            "name": skill_id,
            "description": description or f"{skill_id}技能",
            "version": "1.0.0",
            "compatibility": "Designed for Cursor, Claude Code, and OpenCode",
            "metadata": {
                "author": "Skill Hub User",
                "tags": ["skill", "template"],
                "created": __import__('datetime').datetime.now().strftime("%Y-%m-%d")
            },
            "variables": [
                {
                    "name": "LANGUAGE",
                    "default": "zh-CN",
                    "description": "输出语言"
                }
            ]
        }