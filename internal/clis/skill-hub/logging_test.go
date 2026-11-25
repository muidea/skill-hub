package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runCLIPlainProcess(t *testing.T, home, project string, extraEnv []string, args ...string) (string, string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestCLIJSONProcess$", "--"}, args...)...)
	cmd.Dir = project
	env := []string{
		"SKILL_HUB_JSON_TEST_PROCESS=1",
		"SKILL_HUB_HOME=" + home,
		"SKILL_HUB_DISABLE_SERVICE_BRIDGE=1",
	}
	cmd.Env = append(os.Environ(), append(env, extraEnv...)...)

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	cmd.Stdin = reader

	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("CLI timed out: %v\nstdout=%s\nstderr=%s", args, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String(), err
}

func TestAllCommandsSilenceUsageOnRuntimeError(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	tmpDir := t.TempDir()
	env := []string{"TMPDIR=" + tmpDir}

	for _, args := range [][]string{
		{"status"},
		{"list"},
		{"push", "--force"},
		{"pull"},
		{"git", "status"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stdout, stderr, err := runCLIPlainProcess(t, home, project, env, args...)
			if err == nil {
				t.Fatalf("expected error for uninitialized home on %v", args)
			}
			if strings.Contains(stdout, "Usage:") || strings.Contains(stderr, "Usage:") {
				t.Fatalf("runtime error for %v printed Usage:\nstdout=%s\nstderr=%s", args, stdout, stderr)
			}
			if strings.Contains(stderr, "详细日志已保存至:") {
				t.Fatalf("simple error without detailed logs should not create log file: %s", stderr)
			}
		})
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no temp log files for simple errors, found %d", len(entries))
	}

	// Argument/flag validation errors before RunE must still print Usage in text mode.
	_, stderr, err := runCLIPlainProcess(t, home, project, env, "push", "--no-such-flag")
	if err == nil {
		t.Fatal("expected error for invalid flag")
	}
	if !strings.Contains(stderr, "Usage:") {
		t.Fatalf("invalid flag should still print Usage in non-JSON mode, got stderr=%q", stderr)
	}
}

func setupIsolatedPushRepo(t *testing.T, home string) (string, string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}

	workRoot := t.TempDir()
	remoteDir := filepath.Join(workRoot, "remote.git")
	seedDir := filepath.Join(workRoot, "seed")
	repoDir := filepath.Join(home, "repositories", "main")
	gitHome := filepath.Join(workRoot, "git-home")
	if err := os.MkdirAll(gitHome, 0755); err != nil {
		t.Fatal(err)
	}

	runGit := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"HOME="+gitHome,
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=seed",
			"GIT_AUTHOR_EMAIL=seed@example.com",
			"GIT_COMMITTER_NAME=seed",
			"GIT_COMMITTER_EMAIL=seed@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s: %v", args, string(out), err)
		}
	}

	runGit(workRoot, "init", "--bare", remoteDir)
	runGit(workRoot, "init", "-b", "main", seedDir)
	runGit(seedDir, "remote", "add", "origin", remoteDir)
	if err := os.WriteFile(filepath.Join(seedDir, "README.md"), []byte("init\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(seedDir, "add", ".")
	runGit(seedDir, "commit", "-m", "initial")
	runGit(seedDir, "push", "-u", "origin", "main")

	configContent := `multi_repo:
  enabled: true
  default_repo: "main"
  repositories:
    main:
      name: "main"
      url: "` + remoteDir + `"
      branch: "main"
      enabled: true
      type: "user"
`
	if err := os.WriteFile(filepath.Join(home, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(repoDir), 0755); err != nil {
		t.Fatal(err)
	}
	runGit(workRoot, "clone", "--branch", "main", remoteDir, repoDir)
	runGit(repoDir, "config", "user.useConfigOnly", "true")

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("updated\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return remoteDir, repoDir, gitHome
}

func TestPushMissingIdentityPersistsTempLogAndKeepsConsoleClean(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	tmpDir := t.TempDir()
	_, _, gitHome := setupIsolatedPushRepo(t, home)

	isolatedEnv := []string{
		"TMPDIR=" + tmpDir,
		"HOME=" + gitHome,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=",
		"XDG_CONFIG_HOME=",
		"GIT_AUTHOR_NAME=",
		"GIT_AUTHOR_EMAIL=",
		"GIT_COMMITTER_NAME=",
		"GIT_COMMITTER_EMAIL=",
		"EMAIL=",
	}

	// Plain text failure: console must stay clean, temp log file created and reported on stderr.
	stdout, stderr, err := runCLIPlainProcess(t, home, project, isolatedEnv, "push", "--force")
	if err == nil {
		t.Fatal("expected push --force to fail without git user identity")
	}
	if strings.Contains(stdout, "FETCH_HEAD") || strings.Contains(stdout, "From ") {
		t.Fatalf("raw git fetch output leaked to stdout:\n%s", stdout)
	}
	if strings.Contains(stdout, "Usage:") || strings.Contains(stderr, "Usage:") {
		t.Fatalf("runtime error printed Usage:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	marker := "详细日志已保存至: "
	idx := strings.Index(stderr, marker)
	if idx < 0 {
		t.Fatalf("expected temp log hint in stderr, got:\n%s", stderr)
	}
	logPath := strings.TrimSpace(strings.SplitN(stderr[idx+len(marker):], "\n", 2)[0])
	info, statErr := os.Stat(logPath)
	if statErr != nil {
		t.Fatalf("Stat(%s) error = %v", logPath, statErr)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("temp log mode = %o, want 0600", info.Mode().Perm())
	}
	logBytes, readErr := os.ReadFile(logPath)
	if readErr != nil {
		t.Fatalf("ReadFile(%s) error = %v", logPath, readErr)
	}
	logText := string(logBytes)
	if !strings.Contains(logText, "FETCH_HEAD") || !strings.Contains(logText, "var GIT_AUTHOR_IDENT") {
		t.Fatalf("persisted log missing expected git diagnostics:\n%s", logText)
	}

	// JSON failure: stdout remains single JSON object, details.log_file points to temp log.
	jsonStdout, jsonStderr, err := runCLIPlainProcess(t, home, project, isolatedEnv, "push", "--force", "--json")
	if err == nil {
		t.Fatal("expected push --force --json to fail without git user identity")
	}
	var payload map[string]any
	if decodeErr := json.Unmarshal([]byte(jsonStdout), &payload); decodeErr != nil {
		t.Fatalf("stdout must be valid JSON: %v\nstdout=%q\nstderr=%s", decodeErr, jsonStdout, jsonStderr)
	}
	cmdErr := payload["command_error"].(map[string]any)
	if cmdErr["code"] != "GIT_OPERATION_FAILED" {
		t.Fatalf("unexpected error code: %v", cmdErr)
	}
	details, ok := cmdErr["details"].(map[string]any)
	if !ok {
		t.Fatalf("expected details in command_error: %v", cmdErr)
	}
	jsonLogFile, _ := details["log_file"].(string)
	if jsonLogFile == "" {
		t.Fatalf("expected log_file in details: %v", details)
	}
	if _, err := os.Stat(jsonLogFile); err != nil {
		t.Fatalf("json log_file does not exist: %v", err)
	}
}

func TestLogFileAndDebugFlagsOnSuccess(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	tmpDir := t.TempDir()
	remoteDir, repoDir, gitHome := setupIsolatedPushRepo(t, home)

	// Configure valid local repo identity for push.
	for _, kv := range [][2]string{{"user.name", "tester"}, {"user.email", "tester@example.com"}} {
		cmd := exec.Command("git", "-C", repoDir, "config", kv[0], kv[1])
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git config %v failed: %s: %v", kv, string(out), err)
		}
	}

	explicitLog := filepath.Join(t.TempDir(), "push-success.log")
	env := []string{
		"TMPDIR=" + tmpDir,
		"HOME=" + gitHome,
		"GIT_CONFIG_NOSYSTEM=1",
	}

	stdout, stderr, err := runCLIPlainProcess(t, home, project, env, "push", "--force", "--debug", "--log-file", explicitLog)
	if err != nil {
		t.Fatalf("push --force --debug --log-file failed: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout, "FETCH_HEAD") {
		t.Fatalf("raw git output should not go to stdout even with --debug:\n%s", stdout)
	}
	if !strings.Contains(stderr, "FETCH_HEAD") {
		t.Fatalf("expected --debug to mirror git output to stderr, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "详细日志已保存至: "+explicitLog) {
		t.Fatalf("expected explicit log path reported on stderr, got:\n%s", stderr)
	}
	info, err := os.Stat(explicitLog)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", explicitLog, err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("explicit log mode = %o, want 0600", info.Mode().Perm())
	}

	showCmd := exec.Command("git", "-C", remoteDir, "show", "main:README.md")
	remoteContent, err := showCmd.CombinedOutput()
	if err != nil || string(remoteContent) != "updated\n" {
		t.Fatalf("remote README = %q, err=%v", string(remoteContent), err)
	}
}
