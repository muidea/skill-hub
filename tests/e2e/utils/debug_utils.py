import os
import sys
import json
import shutil
import tempfile
import traceback
from pathlib import Path
from typing import Dict, Any, Optional, List
from datetime import datetime

class DebugUtils:
    """调试工具类"""
    
    @staticmethod
    def capture_environment() -> Dict[str, Any]:
        """捕获当前环境信息"""
        import platform
        
        env_info = {
            "timestamp": datetime.now().isoformat(),
            "python": {
                "version": sys.version,
                "executable": sys.executable,
                "path": sys.path
            },
            "system": {
                "platform": platform.platform(),
                "system": platform.system(),
                "release": platform.release(),
                "version": platform.version(),
                "machine": platform.machine(),
                "processor": platform.processor()
            },
            "environment": {
                "cwd": os.getcwd(),
                "user": os.getenv("USER", "unknown"),
                "home": os.getenv("HOME", "unknown"),
                "path": os.getenv("PATH", "").split(":")[:10]  # 只取前10个
            },
            "skill_hub": {
                "installed": shutil.which("skill-hub") is not None,
                "path": shutil.which("skill-hub") or "未安装"
            }
        }
        
        # 尝试获取skill-hub版本
        try:
            import subprocess
            result = subprocess.run(
                ["skill-hub", "--version"],
                capture_output=True, text=True, timeout=3
            )
            if result.returncode == 0:
                env_info["skill_hub"]["version"] = result.stdout.strip()
        except Exception:
            env_info["skill_hub"]["version"] = "获取失败"
        
        return env_info
    
    @staticmethod
    def save_debug_info(directory: str, info: Dict[str, Any], filename: str = "debug_info.json"):
        """保存调试信息到文件"""
        debug_file = os.path.join(directory, filename)
        with open(debug_file, 'w', encoding='utf-8') as f:
            json.dump(info, f, indent=2, ensure_ascii=False)
    
    @staticmethod
    def capture_exception_info(exception: Exception) -> Dict[str, Any]:
        """捕获异常信息"""
        return {
            "type": type(exception).__name__,
            "message": str(exception),
            "traceback": traceback.format_exc(),
            "timestamp": datetime.now().isoformat()
        }
    
    @staticmethod
    def create_debug_snapshot(test_name: str, temp_dir: str, exception: Optional[Exception] = None) -> str:
        """
        创建调试快照
        
        Args:
            test_name: 测试名称
            temp_dir: 临时目录
            exception: 异常对象（如果有）
            
        Returns:
            快照目录路径
        """
        # 创建快照目录
        snapshot_dir = tempfile.mkdtemp(prefix=f"snapshot_{test_name}_")
        
        # 复制临时目录内容
        if os.path.exists(temp_dir):
            try:
                # 复制整个目录
                dest_dir = os.path.join(snapshot_dir, "test_environment")
                shutil.copytree(temp_dir, dest_dir)
            except Exception as e:
                print(f"复制目录失败: {e}")
        
        # 保存环境信息
        env_info = DebugUtils.capture_environment()
        DebugUtils.save_debug_info(snapshot_dir, env_info, "environment.json")
        
        # 保存异常信息
        if exception:
            exc_info = DebugUtils.capture_exception_info(exception)
            DebugUtils.save_debug_info(snapshot_dir, exc_info, "exception.json")
        
        # 创建README文件
        readme_content = f"""# 调试快照: {test_name}

## 快照信息
- 创建时间: {datetime.now().isoformat()}
- 测试名称: {test_name}
- 快照目录: {snapshot_dir}
- 原始临时目录: {temp_dir}

## 包含内容
1. `test_environment/` - 测试时的完整环境
2. `environment.json` - 系统环境信息
3. `exception.json` - 异常信息（如果有）

## 调试命令
```bash
# 查看目录结构
ls -la {snapshot_dir}

# 查看环境信息
cat {snapshot_dir}/environment.json | jq .  # 需要jq命令

# 查看异常信息
cat {snapshot_dir}/exception.json 2>/dev/null || echo "无异常信息"

# 查看skill-hub配置
find {snapshot_dir} -name ".skill-hub" -type d | head -1 | xargs ls -la 2>/dev/null
```

## 注意事项
此快照包含测试时的完整环境，可能包含临时文件。
调试完成后请手动删除快照目录。
"""
        
        readme_path = os.path.join(snapshot_dir, "README.md")
        with open(readme_path, 'w', encoding='utf-8') as f:
            f.write(readme_content)
        
        print(f"\n🔍 调试快照已创建: {snapshot_dir}")
        print(f"   查看README: cat {snapshot_dir}/README.md")
        
        return snapshot_dir
    
    @staticmethod
    def analyze_directory_structure(directory: str, max_depth: int = 3) -> Dict[str, Any]:
        """分析目录结构"""
        if not os.path.exists(directory):
            return {"error": "目录不存在"}
        
        def scan_dir(current_dir: str, current_depth: int) -> Dict[str, Any]:
            if current_depth > max_depth:
                return {"type": "directory", "depth_exceeded": True}
            
            result = {
                "type": "directory",
                "path": current_dir,
                "files": [],
                "directories": {}
            }
            
            try:
                items = os.listdir(current_dir)
                for item in sorted(items):
                    item_path = os.path.join(current_dir, item)
                    
                    if os.path.isdir(item_path):
                        result["directories"][item] = scan_dir(item_path, current_depth + 1)
                    else:
                        file_info = {
                            "name": item,
                            "size": os.path.getsize(item_path),
                            "modified": datetime.fromtimestamp(os.path.getmtime(item_path)).isoformat()
                        }
                        
                        # 尝试读取小文件的内容
                        if file_info["size"] < 10240:  # 10KB以下
                            try:
                                with open(item_path, 'r', encoding='utf-8') as f:
                                    content = f.read()
                                    # 只保留前500字符
                                    file_info["preview"] = content[:500] + ("..." if len(content) > 500 else "")
                            except:
                                file_info["preview"] = "[二进制文件或编码错误]"
                        
                        result["files"].append(file_info)
            except PermissionError:
                result["error"] = "权限不足"
            except Exception as e:
                result["error"] = str(e)
            
            return result
        
        return scan_dir(directory, 0)
    
    @staticmethod
    def compare_files(file1: str, file2: str) -> Dict[str, Any]:
        """比较两个文件"""
        result = {
            "file1": file1,
            "file2": file2,
            "exist_file1": os.path.exists(file1),
            "exist_file2": os.path.exists(file2),
            "equal": False,
            "differences": []
        }
        
        if not result["exist_file1"] or not result["exist_file2"]:
            return result
        
        # 检查文件大小
        size1 = os.path.getsize(file1)
        size2 = os.path.getsize(file2)
        
        if size1 != size2:
            result["differences"].append(f"文件大小不同: {size1} != {size2}")
        
        # 检查文件内容
        try:
            with open(file1, 'r', encoding='utf-8') as f1, open(file2, 'r', encoding='utf-8') as f2:
                content1 = f1.read()
                content2 = f2.read()
                
                if content1 == content2:
                    result["equal"] = True
                else:
                    # 简单的行比较
                    lines1 = content1.splitlines()
                    lines2 = content2.splitlines()
                    
                    for i, (line1, line2) in enumerate(zip(lines1, lines2)):
                        if line1 != line2:
                            result["differences"].append(f"第{i+1}行不同:\n  文件1: {line1[:100]}...\n  文件2: {line2[:100]}...")
                    
                    # 检查行数差异
                    if len(lines1) != len(lines2):
                        result["differences"].append(f"行数不同: {len(lines1)} != {len(lines2)}")
        except Exception as e:
            result["error"] = f"比较文件时出错: {e}"
        
        return result
    
    @staticmethod
    def find_pattern_in_directory(directory: str, pattern: str, file_pattern: str = "*.md") -> List[Dict[str, Any]]:
        """在目录中查找模式"""
        import re
        
        results = []
        regex = re.compile(pattern, re.IGNORECASE)
        
        for root, dirs, files in os.walk(directory):
            for file in files:
                if not file.endswith(file_pattern.replace("*", "")):
                    continue
                
                file_path = os.path.join(root, file)
                try:
                    with open(file_path, 'r', encoding='utf-8') as f:
                        content = f.read()
                        matches = list(regex.finditer(content))
                        
                        if matches:
                            results.append({
                                "file": file_path,
                                "matches": [
                                    {
                                        "line": m.group(0),
                                        "start": m.start(),
                                        "end": m.end()
                                    }
                                    for m in matches[:5]  # 只取前5个匹配
                                ],
                                "match_count": len(matches)
                            })
                except Exception:
                    continue
        
        return results
    
    @staticmethod
    def create_test_report(test_results: List[Dict[str, Any]], output_dir: str) -> str:
        """创建测试报告"""
        report_file = os.path.join(output_dir, "test_report.md")
        
        total_tests = len(test_results)
        passed_tests = sum(1 for r in test_results if r.get("passed", False))
        failed_tests = total_tests - passed_tests
        
        report_content = f"""# 测试报告

## 概要
- 测试时间: {datetime.now().isoformat()}
- 总测试数: {total_tests}
- 通过测试: {passed_tests}
- 失败测试: {failed_tests}
- 通过率: {passed_tests/total_tests*100:.1f}%

## 详细结果

"""
        
        for i, result in enumerate(test_results, 1):
            status = "✅ 通过" if result.get("passed", False) else "❌ 失败"
            report_content += f"### 测试 {i}: {result.get('name', f'测试{i}')}\n"
            report_content += f"- 状态: {status}\n"
            report_content += f"- 耗时: {result.get('duration', 0):.2f}秒\n"
            
            if not result.get("passed", False):
                report_content += f"- 错误: {result.get('error', '未知错误')}\n"
            
            if result.get("debug_info"):
                report_content += f"- 调试信息: {result['debug_info']}\n"
            
            report_content += "\n"
        
        with open(report_file, 'w', encoding='utf-8') as f:
            f.write(report_content)
        
        return report_file