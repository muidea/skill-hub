package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunServeRegisterAndRemove(t *testing.T) {
	t.Setenv("SKILL_HUB_HOME", t.TempDir())

	prevRunning := serveProcessRunning
	t.Cleanup(func() {
		serveProcessRunning = prevRunning
	})
	serveProcessRunning = func(pid int) bool { return false }

	if err := runServeRegister("local-dev", "127.0.0.1", 6600, "write-secret"); err != nil {
		t.Fatalf("runServeRegister() error = %v", err)
	}

	registry, err := loadServeRegistry()
	if err != nil {
		t.Fatalf("loadServeRegistry() error = %v", err)
	}

	entry, ok := registry.Services["local-dev"]
	if !ok {
		t.Fatalf("expected registered service to exist")
	}
	if entry.Host != "127.0.0.1" || entry.Port != 6600 {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if entry.SecretKey != "write-secret" {
		t.Fatalf("expected secret key to be stored for process startup, got %q", entry.SecretKey)
	}

	if err := runServeRemove("local-dev"); err != nil {
		t.Fatalf("runServeRemove() error = %v", err)
	}

	registry, err = loadServeRegistry()
	if err != nil {
		t.Fatalf("loadServeRegistry() error = %v", err)
	}
	if _, ok := registry.Services["local-dev"]; ok {
		t.Fatalf("expected service to be removed")
	}
}

func TestRunServeStartStopAndRemove(t *testing.T) {
	t.Setenv("SKILL_HUB_HOME", t.TempDir())

	if err := runServeRegister("local-dev", "127.0.0.1", 6600, ""); err != nil {
		t.Fatalf("runServeRegister() error = %v", err)
	}

	prevStart := serveStartProcess
	prevStop := serveStopProcess
	prevRunning := serveProcessRunning
	prevWait := serveWaitUntilReady
	t.Cleanup(func() {
		serveStartProcess = prevStart
		serveStopProcess = prevStop
		serveProcessRunning = prevRunning
		serveWaitUntilReady = prevWait
	})

	running := map[int]bool{}
	serveStartProcess = func(entry serveRegistration) (int, string, error) {
		running[4321] = true
		return 4321, "/tmp/local-dev.log", nil
	}
	serveStopProcess = func(pid int) error {
		delete(running, pid)
		return nil
	}
	serveProcessRunning = func(pid int) bool {
		return running[pid]
	}
	serveWaitUntilReady = func(entry serveRegistration, pid int) error {
		return nil
	}

	if err := runServeStart("local-dev"); err != nil {
		t.Fatalf("runServeStart() error = %v", err)
	}

	registry, err := loadServeRegistry()
	if err != nil {
		t.Fatalf("loadServeRegistry() error = %v", err)
	}

	entry := registry.Services["local-dev"]
	if entry.PID != 4321 {
		t.Fatalf("expected PID 4321, got %d", entry.PID)
	}
	if entry.LogFile != "/tmp/local-dev.log" {
		t.Fatalf("expected log path to be saved, got %q", entry.LogFile)
	}

	if err := runServeRemove("local-dev"); err == nil {
		t.Fatalf("expected removing running service to fail")
	}

	if err := runServeStop("local-dev"); err != nil {
		t.Fatalf("runServeStop() error = %v", err)
	}

	registry, err = loadServeRegistry()
	if err != nil {
		t.Fatalf("loadServeRegistry() error = %v", err)
	}

	entry = registry.Services["local-dev"]
	if entry.PID != 0 {
		t.Fatalf("expected PID to be cleared, got %d", entry.PID)
	}

	if err := runServeRemove("local-dev"); err != nil {
		t.Fatalf("runServeRemove() error = %v", err)
	}
}

func TestRunServeStatus(t *testing.T) {
	t.Setenv("SKILL_HUB_HOME", t.TempDir())

	if err := runServeRegister("running-svc", "127.0.0.1", 6600, ""); err != nil {
		t.Fatalf("runServeRegister() error = %v", err)
	}
	if err := runServeRegister("stopped-svc", "127.0.0.1", 6601, ""); err != nil {
		t.Fatalf("runServeRegister() error = %v", err)
	}

	registry, err := loadServeRegistry()
	if err != nil {
		t.Fatalf("loadServeRegistry() error = %v", err)
	}
	runningEntry := registry.Services["running-svc"]
	runningEntry.PID = 1001
	runningEntry.LogFile = "/tmp/running.log"
	registry.Services["running-svc"] = runningEntry

	stoppedEntry := registry.Services["stopped-svc"]
	stoppedEntry.PID = 1002
	registry.Services["stopped-svc"] = stoppedEntry

	if err := saveServeRegistry(registry); err != nil {
		t.Fatalf("saveServeRegistry() error = %v", err)
	}

	prevRunning := serveProcessRunning
	t.Cleanup(func() {
		serveProcessRunning = prevRunning
	})
	serveProcessRunning = func(pid int) bool {
		return pid == 1001
	}

	output := captureStdout(t, func() {
		if err := runServeStatus(""); err != nil {
			t.Fatalf("runServeStatus() error = %v", err)
		}
	})

	if !strings.Contains(output, "running-svc\trunning") {
		t.Fatalf("expected running service in output, got %q", output)
	}
	if !strings.Contains(output, "push=blocked") {
		t.Fatalf("expected blocked remote push marker, got %q", output)
	}
	if !strings.Contains(output, "stopped-svc\tstale") {
		t.Fatalf("expected stale service in output, got %q", output)
	}

	singleOutput := captureStdout(t, func() {
		if err := runServeStatus("running-svc"); err != nil {
			t.Fatalf("runServeStatus(name) error = %v", err)
		}
	})
	if !strings.Contains(singleOutput, "running-svc\trunning") {
		t.Fatalf("expected named service output, got %q", singleOutput)
	}
}

func TestDefaultServeStartProcessReturnsPIDAfterRelease(t *testing.T) {
	rootDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", rootDir)

	scriptPath := filepath.Join(rootDir, "fake-skill-hub.sh")
	scriptContent := "#!/bin/sh\nsleep 5\n"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	prevExecutable := serveExecutablePath
	t.Cleanup(func() {
		serveExecutablePath = prevExecutable
	})
	serveExecutablePath = func() (string, error) {
		return scriptPath, nil
	}

	pid, logFile, err := defaultServeStartProcess(serveRegistration{
		Name: "fake",
		Host: "127.0.0.1",
		Port: 5525,
	})
	if err != nil {
		t.Fatalf("defaultServeStartProcess() error = %v", err)
	}
	if pid <= 0 {
		t.Fatalf("expected positive pid, got %d", pid)
	}
	if logFile == "" {
		t.Fatal("expected log file to be returned")
	}

	process, err := os.FindProcess(pid)
	if err == nil {
		_ = process.Kill()
	}
}

func TestServeHealthCheckURL(t *testing.T) {
	tests := []struct {
		name  string
		entry serveRegistration
		want  string
	}{
		{
			name: "loopback host unchanged",
			entry: serveRegistration{
				Host: "127.0.0.1",
				Port: 5525,
			},
			want: "http://127.0.0.1:5525/api/v1/health",
		},
		{
			name: "wildcard host converted to loopback",
			entry: serveRegistration{
				Host: "0.0.0.0",
				Port: 5525,
			},
			want: "http://127.0.0.1:5525/api/v1/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serveHealthCheckURL(tt.entry)
			if got != tt.want {
				t.Fatalf("serveHealthCheckURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
