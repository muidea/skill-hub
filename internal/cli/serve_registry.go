package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/muidea/skill-hub/internal/config"
	httpapibiz "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/biz"
	"github.com/muidea/skill-hub/pkg/errors"
)

type serveRegistration struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	SecretKey   string `json:"secret_key,omitempty"`
	PID         int    `json:"pid,omitempty"`
	LogFile     string `json:"log_file,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	LastStarted string `json:"last_started,omitempty"`
}

type serveRegistryFile struct {
	Services map[string]serveRegistration `json:"services"`
}

var (
	serveExecutablePath = os.Executable
	serveStartProcess   = defaultServeStartProcess
	serveStopProcess    = defaultServeStopProcess
	serveProcessRunning = defaultServeProcessRunning
	serveWaitUntilReady = defaultServeWaitUntilReady
)

func runServeRegister(name, host string, port int, secretKey string) error {
	if !isValidRepoName(name) {
		return errors.NewWithCode("runServeRegister", errors.ErrInvalidInput, "服务名称只能包含字母、数字、下划线和连字符")
	}
	if host == "" {
		return errors.NewWithCode("runServeRegister", errors.ErrInvalidInput, "监听地址不能为空")
	}
	if port <= 0 || port > 65535 {
		return errors.NewWithCode("runServeRegister", errors.ErrInvalidInput, "监听端口必须在 1-65535 之间")
	}

	registry, err := loadServeRegistry()
	if err != nil {
		return errors.Wrap(err, "加载服务注册表失败")
	}

	entry, existed := registry.Services[name]
	entry.Name = name
	entry.Host = host
	entry.Port = port
	entry.SecretKey = strings.TrimSpace(secretKey)
	entry.UpdatedAt = time.Now().Format(time.RFC3339)

	if entry.PID > 0 && !serveProcessRunning(entry.PID) {
		entry.PID = 0
		entry.LogFile = ""
	}

	registry.Services[name] = entry
	if err := saveServeRegistry(registry); err != nil {
		return errors.Wrap(err, "保存服务注册表失败")
	}

	action := "注册"
	if existed {
		action = "更新"
	}

	fmt.Printf("✅ 服务 '%s' 已%s\n", name, action)
	fmt.Printf("   地址: http://%s:%d\n", host, port)
	fmt.Printf("   写权限: %s\n", serveWriteAccessLabel(entry.SecretKey))
	return nil
}

func runServeRemove(name string) error {
	registry, err := loadServeRegistry()
	if err != nil {
		return errors.Wrap(err, "加载服务注册表失败")
	}

	entry, exists := registry.Services[name]
	if !exists {
		return errors.NewWithCodef("runServeRemove", errors.ErrFileNotFound, "服务 '%s' 未注册", name)
	}

	if entry.PID > 0 && serveProcessRunning(entry.PID) {
		return errors.NewWithCodef("runServeRemove", errors.ErrValidation, "服务 '%s' 正在运行，请先执行 'skill-hub serve stop %s'", name, name)
	}

	delete(registry.Services, name)
	if err := saveServeRegistry(registry); err != nil {
		return errors.Wrap(err, "保存服务注册表失败")
	}

	fmt.Printf("✅ 服务 '%s' 已删除\n", name)
	return nil
}

func runServeStart(name string) error {
	registry, err := loadServeRegistry()
	if err != nil {
		return errors.Wrap(err, "加载服务注册表失败")
	}

	entry, exists := registry.Services[name]
	if !exists {
		return errors.NewWithCodef("runServeStart", errors.ErrFileNotFound, "服务 '%s' 未注册", name)
	}

	if entry.PID > 0 {
		if serveProcessRunning(entry.PID) {
			return errors.NewWithCodef("runServeStart", errors.ErrValidation, "服务 '%s' 已在运行中 (PID: %d)", name, entry.PID)
		}
		entry.PID = 0
		entry.LogFile = ""
	}

	pid, logFile, err := serveStartProcess(entry)
	if err != nil {
		return errors.Wrap(err, "启动服务失败")
	}

	entry.PID = pid
	entry.LogFile = logFile
	entry.LastStarted = time.Now().Format(time.RFC3339)
	entry.UpdatedAt = time.Now().Format(time.RFC3339)
	registry.Services[name] = entry
	if err := saveServeRegistry(registry); err != nil {
		if serveProcessRunning(pid) {
			_ = serveStopProcess(pid)
		}
		return errors.Wrap(err, "保存服务运行状态失败")
	}

	if err := serveWaitUntilReady(entry, pid); err != nil {
		if serveProcessRunning(pid) {
			_ = serveStopProcess(pid)
		}
		entry.PID = 0
		registry.Services[name] = entry
		_ = saveServeRegistry(registry)
		return errors.Wrap(err, "服务启动后未就绪")
	}

	fmt.Printf("✅ 服务 '%s' 已启动\n", name)
	fmt.Printf("   PID: %d\n", pid)
	fmt.Printf("   地址: http://%s:%d\n", entry.Host, entry.Port)
	if logFile != "" {
		fmt.Printf("   日志: %s\n", logFile)
	}
	return nil
}

func runServeStop(name string) error {
	registry, err := loadServeRegistry()
	if err != nil {
		return errors.Wrap(err, "加载服务注册表失败")
	}

	entry, exists := registry.Services[name]
	if !exists {
		return errors.NewWithCodef("runServeStop", errors.ErrFileNotFound, "服务 '%s' 未注册", name)
	}

	if entry.PID == 0 {
		fmt.Printf("服务 '%s' 当前未运行\n", name)
		return nil
	}

	if !serveProcessRunning(entry.PID) {
		entry.PID = 0
		registry.Services[name] = entry
		if err := saveServeRegistry(registry); err != nil {
			return errors.Wrap(err, "保存服务运行状态失败")
		}
		fmt.Printf("服务 '%s' 当前未运行，已清理失效 PID\n", name)
		return nil
	}

	if err := serveStopProcess(entry.PID); err != nil {
		return errors.Wrap(err, "停止服务失败")
	}

	entry.PID = 0
	entry.UpdatedAt = time.Now().Format(time.RFC3339)
	registry.Services[name] = entry
	if err := saveServeRegistry(registry); err != nil {
		return errors.Wrap(err, "保存服务运行状态失败")
	}

	fmt.Printf("✅ 服务 '%s' 已停止\n", name)
	return nil
}

func runServeStatus(name string) error {
	registry, err := loadServeRegistry()
	if err != nil {
		return errors.Wrap(err, "加载服务注册表失败")
	}

	if name != "" {
		entry, exists := registry.Services[name]
		if !exists {
			return errors.NewWithCodef("runServeStatus", errors.ErrFileNotFound, "服务 '%s' 未注册", name)
		}
		renderServeStatus(entry)
		return nil
	}

	if len(registry.Services) == 0 {
		fmt.Println("暂无已注册服务")
		return nil
	}

	fmt.Println("已注册服务:")
	for _, entry := range registry.Services {
		renderServeStatus(entry)
	}
	return nil
}

func loadServeRegistry() (*serveRegistryFile, error) {
	registryPath, err := getServeRegistryPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &serveRegistryFile{Services: map[string]serveRegistration{}}, nil
		}
		return nil, err
	}

	registry := &serveRegistryFile{}
	if err := json.Unmarshal(data, registry); err != nil {
		return nil, err
	}
	if registry.Services == nil {
		registry.Services = map[string]serveRegistration{}
	}
	return registry, nil
}

func saveServeRegistry(registry *serveRegistryFile) error {
	registryPath, err := getServeRegistryPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(registryPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(registryPath, data, 0644)
}

func getServeRegistryPath() (string, error) {
	rootDir, err := config.GetRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootDir, "services.json"), nil
}

func listServeRegistrations() ([]serveRegistration, error) {
	registry, err := loadServeRegistry()
	if err != nil {
		return nil, err
	}

	result := make([]serveRegistration, 0, len(registry.Services))
	for _, entry := range registry.Services {
		result = append(result, entry)
	}
	return result, nil
}

func renderServeStatus(entry serveRegistration) {
	status := "stopped"
	if entry.PID > 0 {
		if serveProcessRunning(entry.PID) {
			status = "running"
		} else {
			status = "stale"
		}
	}

	fmt.Printf("%s\t%s\t%s\tpid=%d\twrite=%s\n", entry.Name, status, fmt.Sprintf("http://%s:%d", entry.Host, entry.Port), entry.PID, serveWriteAccessLabel(entry.SecretKey))
	if entry.LogFile != "" {
		fmt.Printf("  log: %s\n", entry.LogFile)
	}
}

func defaultServeStartProcess(entry serveRegistration) (int, string, error) {
	executable, err := serveExecutablePath()
	if err != nil {
		return 0, "", err
	}

	rootDir, err := config.GetRootDir()
	if err != nil {
		return 0, "", err
	}

	logDir := filepath.Join(rootDir, "services", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return 0, "", err
	}

	logFile := filepath.Join(logDir, entry.Name+".log")
	fileHandle, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, "", err
	}
	defer fileHandle.Close()

	args := []string{"serve", "--host", entry.Host, "--port", strconv.Itoa(entry.Port)}
	if strings.TrimSpace(entry.SecretKey) != "" {
		args = append(args, "--secret-key", strings.TrimSpace(entry.SecretKey))
	}
	cmd := exec.Command(executable, args...)
	cmd.Stdin = nil
	cmd.Stdout = fileHandle
	cmd.Stderr = fileHandle

	if err := cmd.Start(); err != nil {
		return 0, "", err
	}
	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		return 0, "", err
	}

	return pid, logFile, nil
}

func defaultServeStopProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	if err := process.Signal(os.Interrupt); err != nil {
		if killErr := process.Kill(); killErr != nil {
			return killErr
		}
		return nil
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !defaultServeProcessRunning(pid) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	if err := process.Kill(); err != nil {
		return err
	}
	return nil
}

func defaultServeProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	return process.Signal(syscall.Signal(0)) == nil
}

func serveHealthCheckURL(entry serveRegistration) string {
	host := entry.Host
	if host == "0.0.0.0" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%d/api/v1/health", host, entry.Port)
}

func defaultServeWaitUntilReady(entry serveRegistration, pid int) error {
	deadline := time.Now().Add(5 * time.Second)
	healthURL := serveHealthCheckURL(entry)
	client := &http.Client{Timeout: 300 * time.Millisecond}
	var lastErr error

	for time.Now().Before(deadline) {
		if !serveProcessRunning(pid) {
			return errors.NewWithCodef("defaultServeWaitUntilReady", errors.ErrSystem, "服务进程已退出 (PID: %d)", pid)
		}

		req, reqErr := http.NewRequest(http.MethodGet, healthURL, nil)
		if reqErr != nil {
			lastErr = reqErr
			time.Sleep(150 * time.Millisecond)
			continue
		}
		if strings.TrimSpace(entry.SecretKey) != "" {
			req.Header.Set(httpapibiz.SecretKeyHeader, strings.TrimSpace(entry.SecretKey))
		}
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = errors.NewWithCodef("defaultServeWaitUntilReady", errors.ErrAPIRequest, "健康检查返回状态码 %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		time.Sleep(150 * time.Millisecond)
	}

	if lastErr == nil {
		lastErr = errors.NewWithCode("defaultServeWaitUntilReady", errors.ErrSystem, "等待服务就绪超时")
	}
	return lastErr
}

func serveWriteAccessLabel(secretKey string) string {
	if strings.TrimSpace(secretKey) == "" {
		return "read-only"
	}
	return "secret-key"
}
