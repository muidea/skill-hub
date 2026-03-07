package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/muidea/skill-hub/internal/config"
)

func TestRunGitRemoteView_NoRemote(t *testing.T) {
	tmpDir := t.TempDir()

	os.Setenv("SKILL_HUB_HOME", tmpDir)
	t.Cleanup(func() {
		os.Unsetenv("SKILL_HUB_HOME")
	})

	cfg := &config.Config{
		MultiRepo: &config.MultiRepoConfig{
			Enabled:     true,
			DefaultRepo: "main",
			Repositories: map[string]config.RepositoryConfig{
				"main": {
					Name:    "main",
					URL:     "",
					Branch:  "main",
					Enabled: true,
					Type:    "user",
				},
			},
		},
	}

	if err := config.SaveConfig(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	repoPath, err := config.GetRepositoryPath("main")
	if err != nil {
		t.Fatalf("GetRepositoryPath error: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0755); err != nil {
		t.Fatalf("failed to create fake git dir: %v", err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = runGitRemoteView(false)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("runGitRemoteView returned error: %v", err)
	}

	if output == "" {
		t.Fatalf("expected some output, got empty string")
	}

	if !bytes.Contains([]byte(output), []byte("默认仓库: main")) {
		t.Fatalf("expected output to contain default repo name, got: %s", output)
	}
}
