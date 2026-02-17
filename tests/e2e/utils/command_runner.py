import os
import subprocess
import shutil
from dataclasses import dataclass
from typing import Optional, Dict, List, Union

@dataclass
class CommandResult:
    """命令执行结果"""
    exit_code: int
    stdout: str
    stderr: str
    command: str
    
    @property
    def success(self) -> bool:
        """命令是否成功执行"""
        return self.exit_code == 0
    
    def __str__(self) -> str:
        return f"Command: {self.command}\nExit Code: {self.exit_code}\nStdout: {self.stdout}\nStderr: {self.stderr}"

class CommandRunner:
    """封装skill-hub命令执行，支持输入交互"""
    
    def __init__(self, timeout: int = 30, debug: bool = False):
        self.timeout = timeout
        self.debug = debug
        self._verify_installation()
    
    def _verify_installation(self):
        """验证skill-hub已安装"""
        # 首先检查环境变量指定的二进制
        self.skill_hub_bin = os.environ.get("SKILL_HUB_BIN")
        if self.skill_hub_bin:
            if not os.path.exists(self.skill_hub_bin):
                raise RuntimeError(f"SKILL_HUB_BIN环境变量指定的二进制不存在: {self.skill_hub_bin}")
        else:
            # 检查项目目录中的二进制
            project_bin = os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname(__file__)))), "skill-hub")
            if os.path.exists(project_bin):
                self.skill_hub_bin = project_bin
            else:
                # 回退到PATH中的二进制
                self.skill_hub_bin = shutil.which("skill-hub")
                if not self.skill_hub_bin:
                    raise RuntimeError("skill-hub未安装或不在PATH中")
        
        # 记录版本信息用于调试
        try:
            result = subprocess.run(
                [self.skill_hub_bin, "--version"],
                capture_output=True, text=True, timeout=5
            )
            if self.debug and result.returncode == 0:
                print(f"skill-hub版本: {result.stdout.strip()}")
        except Exception as e:
            if self.debug:
                print(f"获取skill-hub版本失败: {e}")
    
    def run(self, 
            command: str, 
            args: Optional[Union[str, List[str]]] = None,
            cwd: Optional[str] = None,
            env: Optional[Dict[str, str]] = None,
            input_text: Optional[str] = None) -> CommandResult:
        """
        执行skill-hub命令
        
        Args:
            command: skill-hub子命令（如 'init', 'create'）
            args: 命令参数，可以是字符串或列表
            cwd: 工作目录
            env: 环境变量
            input_text: 标准输入内容
            
        Returns:
            CommandResult: 命令执行结果
            
        Raises:
            TimeoutError: 命令执行超时
            RuntimeError: 命令执行失败
        """
        # 构建完整命令
        cmd = [self.skill_hub_bin, command]
        if args:
            if isinstance(args, str):
                cmd.extend(args.split())
            else:
                cmd.extend(args)
        
        # 准备环境
        exec_env = os.environ.copy()
        if env:
            exec_env.update(env)
        
        # 调试信息
        if self.debug:
            print(f"执行命令: {' '.join(cmd)}")
            if cwd:
                print(f"工作目录: {cwd}")
            print(f"HOME环境变量: {exec_env.get('HOME')}")
            print(f"使用二进制: {self.skill_hub_bin}")
        
        # 执行命令
        try:
            if input_text:
                result = subprocess.run(
                    cmd, 
                    cwd=cwd, 
                    env=exec_env, 
                    timeout=self.timeout,
                    input=input_text, 
                    capture_output=True, 
                    text=True,
                    encoding='utf-8'
                )
            else:
                result = subprocess.run(
                    cmd, 
                    cwd=cwd, 
                    env=exec_env, 
                    timeout=self.timeout,
                    capture_output=True, 
                    text=True,
                    encoding='utf-8'
                )
            
            # 创建结果对象
            command_result = CommandResult(
                exit_code=result.returncode,
                stdout=result.stdout,
                stderr=result.stderr,
                command=' '.join(cmd)
            )
            
            # 调试输出
            if self.debug:
                print(f"退出码: {command_result.exit_code}")
                if command_result.stdout:
                    print(f"标准输出:\n{command_result.stdout}")
                if command_result.stderr:
                    print(f"标准错误:\n{command_result.stderr}")
            
            return command_result
            
        except subprocess.TimeoutExpired:
            error_msg = f"命令执行超时 ({self.timeout}s): {' '.join(cmd)}"
            if self.debug:
                print(f"❌ {error_msg}")
            raise TimeoutError(error_msg)
        except Exception as e:
            error_msg = f"命令执行失败: {' '.join(cmd)} - {str(e)}"
            if self.debug:
                print(f"❌ {error_msg}")
            raise RuntimeError(error_msg)
    
    def run_with_retry(self, 
                      command: str, 
                      args: Optional[Union[str, List[str]]] = None,
                      cwd: Optional[str] = None,
                      env: Optional[Dict[str, str]] = None,
                      max_retries: int = 3,
                      retry_delay: int = 1) -> CommandResult:
        """
        带重试的命令执行
        
        Args:
            command: skill-hub子命令
            args: 命令参数
            cwd: 工作目录
            env: 环境变量
            max_retries: 最大重试次数
            retry_delay: 重试延迟（秒）
            
        Returns:
            CommandResult: 命令执行结果
        """
        last_result = None
        last_exception = None
        
        for attempt in range(max_retries):
            try:
                result = self.run(command, args, cwd, env)
                last_result = result
                
                if result.success:
                    return result
                
                # 检查是否可重试的错误
                if attempt < max_retries - 1:
                    if self.debug:
                        print(f"命令失败，{retry_delay}秒后重试 ({attempt + 1}/{max_retries})")
                    import time
                    time.sleep(retry_delay)
            except (TimeoutError, RuntimeError) as e:
                last_exception = e
                if attempt < max_retries - 1:
                    if self.debug:
                        print(f"命令异常，{retry_delay}秒后重试 ({attempt + 1}/{max_retries}): {e}")
                    import time
                    time.sleep(retry_delay)
                else:
                    raise
        
        # 所有重试都失败
        if last_result is not None:
            return last_result
        elif last_exception is not None:
            raise last_exception
        else:
            return CommandResult(
                exit_code=1,
                stdout="",
                stderr="所有重试都失败",
                command=f"skill-hub {command}"
            )