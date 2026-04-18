import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import random


class ServiceRunner:
    def __init__(self, binary_path: str, env: dict[str, str], cwd: str, host: str = "127.0.0.1"):
        self.binary_path = binary_path
        self.env = env.copy()
        self.cwd = cwd
        self.host = host
        self.port = None
        self.process = None
        self.log_file = None
        self.log_path = None

    @property
    def base_url(self) -> str:
        return f"http://{self.host}:{self.port}"

    def start(self, timeout: float = 10.0):
        last_error = None
        for _ in range(5):
            self.port = self._reserve_port()
            self.log_file = tempfile.NamedTemporaryFile(
                mode="w+",
                encoding="utf-8",
                prefix="skill_hub_service_",
                suffix=".log",
                delete=False,
            )
            self.log_path = self.log_file.name
            cmd = [
                self.binary_path,
                "serve",
                "--host",
                self.host,
                "--port",
                str(self.port),
            ]
            self.process = subprocess.Popen(
                cmd,
                cwd=self.cwd,
                env=self.env,
                stdout=self.log_file,
                stderr=subprocess.STDOUT,
                text=True,
            )
            try:
                self._wait_until_ready(timeout=timeout)
                return self
            except RuntimeError as err:
                last_error = err
                output = self.read_output()
                self.stop()
                if "bind: address already in use" not in output:
                    raise
                time.sleep(0.1)
        raise last_error

    def stop(self):
        try:
            if self.process is None:
                return
            if self.process.poll() is None:
                self.process.terminate()
                try:
                    self.process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    self.process.kill()
                    self.process.wait(timeout=5)
        finally:
            if self.log_file is not None:
                self.log_file.close()
                self.log_file = None

    def read_output(self) -> str:
        if not self.log_path:
            return ""
        with open(self.log_path, encoding="utf-8") as handle:
            return handle.read()

    def _reserve_port(self) -> int:
        for _ in range(50):
            port = random.randint(20000, 60999)
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
                sock.settimeout(0.1)
                if sock.connect_ex((self.host, port)) != 0:
                    return port
        raise RuntimeError("unable to find available localhost port")

    def _wait_until_ready(self, timeout: float):
        deadline = time.time() + timeout
        last_error = None
        while time.time() < deadline:
            if self.process is not None and self.process.poll() is not None:
                output = self.read_output()
                raise RuntimeError(f"service exited early: {output}")
            try:
                with urllib.request.urlopen(f"{self.base_url}/api/v1/health", timeout=0.5) as response:
                    if response.status == 200:
                        return
            except (urllib.error.URLError, TimeoutError, ConnectionError, OSError) as err:
                last_error = err
                time.sleep(0.2)
        raise RuntimeError(f"service did not become ready: {last_error}")
