import os
import sys
import tempfile
import shutil
import datetime
import json
from pathlib import Path
from typing import Dict, Any, Optional

class TestEnvironment:
    """测试环境管理（失败时保留临时文件）"""
    
    def __init__(self, prefix: str = "skill_hub_test", keep_on_failure: bool = True):
        """
        初始化测试环境
        
        Args:
            prefix: 临时目录前缀
            keep_on_failure: 测试失败时是否保留临时文件
        """
        self.temp_dir = tempfile.mkdtemp(prefix=f"{prefix}_")
        self.home_dir = os.path.join(self.temp_dir, "home")
        self.project_dir = os.path.join(self.temp_dir, "project")
        self.keep_on_failure = keep_on_failure
        self.test_passed = False
        self.original_home: Optional[str] = None
        self.original_env_vars: Dict[str, Optional[str]] = {}
        
        # 创建基础目录
        os.makedirs(self.home_dir, exist_ok=True)
        os.makedirs(self.project_dir, exist_ok=True)
    
    def __enter__(self):
        """进入上下文，设置环境"""
        # 备份原始环境变量
        self.original_home = os.environ.get("HOME")
        self._backup_env_vars(["HOME", "SKILL_HUB_TEST_DEBUG"])
        
        # 设置新的HOME目录
        os.environ["HOME"] = self.home_dir
        
        # 设置测试调试标志
        os.environ["SKILL_HUB_TEST_DEBUG"] = "1"
        
        # 记录环境信息
        self._log_environment()
        
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        """退出上下文，根据测试结果决定是否清理"""
        # 标记测试是否通过
        self.test_passed = exc_type is None
        
        # 恢复环境变量
        self._restore_env_vars()
        
        # 根据测试结果决定是否清理
        if self.test_passed or not self.keep_on_failure:
            # 测试通过或配置不保留，清理临时文件
            self._cleanup()
        else:
            # 测试失败且配置保留，打印调试信息
            self._log_failure_info(exc_type, exc_val, exc_tb)
    
    def _backup_env_vars(self, env_vars: list):
        """备份环境变量"""
        for var in env_vars:
            self.original_env_vars[var] = os.environ.get(var)
    
    def _restore_env_vars(self):
        """恢复环境变量"""
        for var, value in self.original_env_vars.items():
            if value is not None:
                os.environ[var] = value
            elif var in os.environ:
                del os.environ[var]
    
    def _log_environment(self):
        """记录环境信息用于调试"""
        env_info = {
            "temp_dir": self.temp_dir,
            "home_dir": self.home_dir,
            "project_dir": self.project_dir,
            "original_home": self.original_home,
            "timestamp": datetime.datetime.now().isoformat(),
            "python_version": sys.version,
            "current_user": os.getenv("USER", "unknown"),
            "environment_vars": {
                "HOME": os.environ.get("HOME"),
                "PATH": os.environ.get("PATH", "").split(':')[:5],  # 只记录前5个
            }
        }
        
        env_file = os.path.join(self.temp_dir, "environment_info.json")
        with open(env_file, 'w', encoding='utf-8') as f:
            json.dump(env_info, f, indent=2, ensure_ascii=False)
    
    def _log_failure_info(self, exc_type, exc_val, exc_tb):
        """记录失败信息"""
        failure_info = {
            "exception_type": str(exc_type.__name__) if exc_type else None,
            "exception_value": str(exc_val) if exc_val else None,
            "test_passed": self.test_passed,
            "failure_time": datetime.datetime.now().isoformat()
        }
        
        failure_file = os.path.join(self.temp_dir, "failure_info.json")
        with open(failure_file, 'w', encoding='utf-8') as f:
            json.dump(failure_info, f, indent=2, ensure_ascii=False)
        
        # 打印调试信息
        print(f"\n{'='*60}")
        print("⚠️ 测试失败，临时文件已保留用于调试")
        print('='*60)
        print(f"临时目录: {self.temp_dir}")
        print(f"HOME目录: {self.home_dir}")
        print(f"项目目录: {self.project_dir}")
        print(f"异常类型: {exc_type.__name__ if exc_type else 'None'}")
        print(f"异常信息: {exc_val}")
        print('='*60)
        print("调试命令:")
        print(f"  # 查看目录结构")
        print(f"  ls -la {self.temp_dir}")
        print(f"  # 查看环境信息")
        print(f"  cat {self.temp_dir}/environment_info.json")
        print(f"  # 查看失败信息")
        print(f"  cat {self.temp_dir}/failure_info.json 2>/dev/null || echo '无失败信息'")
        print('='*60)
    
    def _cleanup(self):
        """清理临时文件"""
        try:
            shutil.rmtree(self.temp_dir, ignore_errors=True)
        except Exception as e:
            # 清理失败时记录但不抛出异常
            print(f"警告: 清理临时目录失败: {e}")
    
    def create_test_file(self, relative_path: str, content: str = "") -> str:
        """
        在项目目录中创建测试文件
        
        Args:
            relative_path: 相对路径
            content: 文件内容
            
        Returns:
            文件的完整路径
        """
        file_path = os.path.join(self.project_dir, relative_path)
        os.makedirs(os.path.dirname(file_path), exist_ok=True)
        
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)
        
        return file_path
    
    def create_home_file(self, relative_path: str, content: str = "") -> str:
        """
        在HOME目录中创建测试文件
        
        Args:
            relative_path: 相对路径
            content: 文件内容
            
        Returns:
            文件的完整路径
        """
        file_path = os.path.join(self.home_dir, relative_path)
        os.makedirs(os.path.dirname(file_path), exist_ok=True)
        
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)
        
        return file_path
    
    def get_file_content(self, relative_path: str, from_home: bool = False) -> str:
        """
        获取文件内容
        
        Args:
            relative_path: 相对路径
            from_home: 是否从HOME目录读取
            
        Returns:
            文件内容
        """
        base_dir = self.home_dir if from_home else self.project_dir
        file_path = os.path.join(base_dir, relative_path)
        
        if not os.path.exists(file_path):
            raise FileNotFoundError(f"文件不存在: {file_path}")
        
        with open(file_path, 'r', encoding='utf-8') as f:
            return f.read()
    
    def list_files(self, relative_path: str = "", from_home: bool = False) -> list:
        """
        列出目录中的文件
        
        Args:
            relative_path: 相对路径
            from_home: 是否从HOME目录列出
            
        Returns:
            文件列表
        """
        base_dir = self.home_dir if from_home else self.project_dir
        target_dir = os.path.join(base_dir, relative_path)
        
        if not os.path.exists(target_dir):
            return []
        
        files = []
        for root, dirs, filenames in os.walk(target_dir):
            rel_root = os.path.relpath(root, base_dir)
            for filename in filenames:
                files.append(os.path.join(rel_root, filename))
        
        return sorted(files)
    
    @property
    def skill_hub_dir(self) -> str:
        """获取skill-hub配置目录路径"""
        return os.path.join(self.home_dir, ".skill-hub")
    
    @property
    def skill_hub_repo_dir(self) -> str:
        """获取skill-hub仓库目录路径"""
        return os.path.join(self.skill_hub_dir, "repo")
    
    @property
    def skill_hub_skills_dir(self) -> str:
        """获取skill-hub技能目录路径"""
        return os.path.join(self.skill_hub_repo_dir, "skills")