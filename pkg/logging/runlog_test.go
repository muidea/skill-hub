package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	apperrors "github.com/muidea/skill-hub/pkg/errors"
)

func TestScrubSecrets(t *testing.T) {
	t.Setenv("SKILL_HUB_SECRET_KEY", "env-secret-value-12345")
	t.Setenv("GITHUB_TOKEN", "ghp_env_secret_token_98765")

	input := strings.Join([]string{
		"remote=https://alice:supersecret@github.com/org/repo.git",
		"Authorization: Bearer bearer-secret-abc",
		"Authorization: Basic dXNlcjpwYXNz",
		"git_token=tok_plain_12345",
		`secretKey: "my-secret-key"`,
		"X-Skill-Hub-Secret: header-secret-xyz",
		"leaked=env-secret-value-12345",
		"gh=ghp_env_secret_token_98765",
	}, "\n")

	got := ScrubSecrets(input)
	for _, secret := range []string{
		"supersecret",
		"bearer-secret-abc",
		"dXNlcjpwYXNz",
		"tok_plain_12345",
		"my-secret-key",
		"header-secret-xyz",
		"env-secret-value-12345",
		"ghp_env_secret_token_98765",
	} {
		if strings.Contains(got, secret) {
			t.Fatalf("ScrubSecrets left %q unmasked in:\n%s", secret, got)
		}
	}
	if !strings.Contains(got, "https://alice:***@github.com/org/repo.git") {
		t.Fatalf("expected username preserved in scrubbed URL, got:\n%s", got)
	}
}

func TestRunLogBoundedMaxBytes(t *testing.T) {
	rl := NewRunLog(RunLogOptions{
		Command:  "skill-hub push",
		MaxBytes: 32,
	})
	payload := strings.Repeat("a", 100)
	n, err := rl.Write([]byte(payload))
	if err != nil || n != len(payload) {
		t.Fatalf("Write() = (%d, %v), want (%d, nil)", n, err, len(payload))
	}
	if !rl.HasEntries() {
		t.Fatal("expected HasEntries() = true")
	}

	path, err := rl.Persist()
	if err != nil {
		t.Fatalf("Persist() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "... [log truncated] ...") {
		t.Fatalf("expected truncation notice, got:\n%s", text)
	}
	if strings.Contains(text, strings.Repeat("a", 50)) {
		t.Fatalf("expected buffered content to be capped at MaxBytes, got:\n%s", text)
	}
}

func TestRunLogPersistPermissionsAndExplicitLogFile(t *testing.T) {
	rl := NewRunLog(RunLogOptions{Command: "skill-hub git sync"})
	rl.WriteSection("step", "detail line")
	rl.RecordError(apperrors.New("boom"))

	tempPath, err := rl.Persist()
	if err != nil {
		t.Fatalf("Persist() temp error = %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(tempPath) })

	if !strings.Contains(filepath.Base(tempPath), "skill-hub-git-sync-") {
		t.Fatalf("unexpected temp log filename: %s", tempPath)
	}
	info, err := os.Stat(tempPath)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", tempPath, err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("temp log permissions = %o, want 0600", info.Mode().Perm())
	}

	explicitPath := filepath.Join(t.TempDir(), "nested", "custom.log")
	rl2 := NewRunLog(RunLogOptions{Command: "skill-hub list", LogFile: explicitPath})
	saved, err := rl2.Persist()
	if err != nil {
		t.Fatalf("Persist() explicit error = %v", err)
	}
	if saved != explicitPath {
		t.Fatalf("Persist() = %q, want %q", saved, explicitPath)
	}
	info2, err := os.Stat(explicitPath)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", explicitPath, err)
	}
	if info2.Mode().Perm() != 0600 {
		t.Fatalf("explicit log permissions = %o, want 0600", info2.Mode().Perm())
	}
}

func TestDetailWriterAndBindGlobalLogger(t *testing.T) {
	rl := NewRunLog(RunLogOptions{Command: "skill-hub pull"})
	restoreDetail := SetDetailWriter(rl)
	defer restoreDetail()

	DetailPrintf("fetching %s\n", "https://user:secretpass@example.com/repo.git")
	LogCommandOutput("git", []string{"fetch", "origin"}, []byte("From origin\n"), apperrors.New("exit status 1"))

	restoreLogger := BindGlobalLogger(rl, LevelDebug)
	defer restoreLogger()
	GetGlobalLogger().Debug("debug entry", "key", "val")
	apperrors.GetLogger().Info("err package entry")

	path, err := rl.Persist()
	if err != nil {
		t.Fatalf("Persist() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(data)
	if strings.Contains(text, "secretpass") {
		t.Fatalf("secret leaked into persisted log:\n%s", text)
	}
	for _, want := range []string{"fetching https://user:***@example.com/repo.git", "$ git fetch origin", "From origin", "[exit error] exit status 1", "debug entry", "err package entry"} {
		if !strings.Contains(text, want) {
			t.Fatalf("expected %q in log:\n%s", want, text)
		}
	}
}
