import socket
import urllib.request
import urllib.error
from typing import Optional
from functools import wraps

class NetworkChecker:
    """网络检查工具"""
    
    @staticmethod
    def is_network_available(timeout: int = 3) -> bool:
        """
        检查网络是否可用
        
        Args:
            timeout: 超时时间（秒）
            
        Returns:
            bool: 网络是否可用
        """
        test_urls = [
            "https://github.com",  # skill-hub update需要
            "https://raw.githubusercontent.com",
            "https://8.8.8.8",  # Google DNS
        ]
        
        for url in test_urls:
            try:
                host = url.split("://")[1] if "://" in url else url
                port = 443 if url.startswith("https") else 80
                
                # 测试TCP连接
                socket.create_connection((host, port), timeout=timeout)
                
                # 对于HTTP/HTTPS URL，也测试HTTP响应
                if url.startswith("http"):
                    try:
                        response = urllib.request.urlopen(url, timeout=timeout)
                        if response.getcode() < 400:
                            return True
                    except (urllib.error.URLError, urllib.error.HTTPError):
                        # HTTP测试失败，但TCP连接成功也算网络可用
                        return True
                else:
                    return True
                    
            except (socket.timeout, socket.error, urllib.error.URLError):
                continue
            except Exception:
                # 其他异常，继续测试下一个URL
                continue
        
        return False
    
    @staticmethod
    def check_github_access(timeout: int = 5) -> bool:
        """
        检查GitHub访问是否正常（skill-hub update需要）
        
        Args:
            timeout: 超时时间（秒）
            
        Returns:
            bool: GitHub是否可访问
        """
        try:
            # 测试GitHub API（轻量级请求）
            req = urllib.request.Request(
                "https://api.github.com",
                headers={"User-Agent": "Skill-Hub-Test"}
            )
            response = urllib.request.urlopen(req, timeout=timeout)
            return response.getcode() == 200
        except Exception:
            return False
    
    @staticmethod
    def get_network_status() -> dict:
        """
        获取详细的网络状态
        
        Returns:
            dict: 网络状态信息
        """
        status = {
            "network_available": NetworkChecker.is_network_available(),
            "github_accessible": NetworkChecker.check_github_access(),
            "test_time": __import__('datetime').datetime.now().isoformat()
        }
        
        # 尝试获取更多信息
        try:
            import subprocess
            # 检查DNS解析
            result = subprocess.run(
                ["nslookup", "github.com"],
                capture_output=True, text=True, timeout=3
            )
            status["dns_working"] = result.returncode == 0
        except Exception:
            status["dns_working"] = False
        
        return status
    
    @staticmethod
    def skip_if_no_network(func=None, reason: str = "需要网络连接"):
        """
        装饰器：网络不可用时跳过测试
        
        Args:
            func: 被装饰的函数
            reason: 跳过原因
            
        Returns:
            装饰后的函数
        """
        def decorator(test_func):
            @wraps(test_func)
            def wrapper(*args, **kwargs):
                # 这里只是定义装饰器，实际跳过逻辑由pytest标记处理
                return test_func(*args, **kwargs)
            return wrapper
        
        if func is None:
            return decorator
        return decorator(func)
    
    @classmethod
    def require_network(cls, timeout: int = 3):
        """
        装饰器：网络不可用时抛出异常
        
        Args:
            timeout: 网络检查超时时间
            
        Returns:
            装饰器函数
        """
        def decorator(func):
            @wraps(func)
            def wrapper(*args, **kwargs):
                if not cls.is_network_available(timeout):
                    raise RuntimeError(f"网络不可用，无法执行测试: {func.__name__}")
                return func(*args, **kwargs)
            return wrapper
        return decorator
    
    @staticmethod
    def wait_for_network(timeout: int = 30, interval: int = 2) -> bool:
        """
        等待网络可用
        
        Args:
            timeout: 总超时时间（秒）
            interval: 检查间隔（秒）
            
        Returns:
            bool: 网络是否在超时前可用
        """
        import time
        
        start_time = time.time()
        while time.time() - start_time < timeout:
            if NetworkChecker.is_network_available():
                return True
            time.sleep(interval)
        
        return False