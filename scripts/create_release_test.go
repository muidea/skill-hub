package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateReleaseNotesUseCommitMessagesOnly(t *testing.T) {
	repoDir := t.TempDir()
	scriptSource := filepath.Join("create-release.sh")
	scriptContent, err := os.ReadFile(scriptSource)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}

	scriptsDir := filepath.Join(repoDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}

	scriptPath := filepath.Join(scriptsDir, "create-release.sh")
	if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	runCmd(t, repoDir, "git", "init")
	runCmd(t, repoDir, "git", "config", "user.name", "Test User")
	runCmd(t, repoDir, "git", "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repoDir, "secret_code.go"), "package main\n")
	runCmd(t, repoDir, "git", "add", "secret_code.go")
	runCmd(t, repoDir, "git", "commit", "-m", "feat(cli): add release generator")

	writeFile(t, filepath.Join(repoDir, "hidden_impl.txt"), "internal implementation\n")
	runCmd(t, repoDir, "git", "add", "hidden_impl.txt")
	runCmd(t, repoDir, "git", "commit", "-m", "fix(release): keep notes sourced from commit messages")

	outputPath := filepath.Join(repoDir, "release-notes.md")
	runCmd(t, repoDir, "bash", scriptPath, "--notes-only", "--yes", "--version", "0.1.0", "--output", outputPath)

	notesContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read notes: %v", err)
	}

	notes := string(notesContent)
	if !strings.Contains(notes, "## 新功能") {
		t.Fatalf("expected feature section in notes: %s", notes)
	}
	if !strings.Contains(notes, "## 问题修复") {
		t.Fatalf("expected fix section in notes: %s", notes)
	}
	if !strings.Contains(notes, "**cli**: add release generator") {
		t.Fatalf("expected normalized feature message in notes: %s", notes)
	}
	if !strings.Contains(notes, "**release**: keep notes sourced from commit messages") {
		t.Fatalf("expected normalized fix message in notes: %s", notes)
	}
	if strings.Contains(notes, "secret_code.go") || strings.Contains(notes, "hidden_impl.txt") {
		t.Fatalf("notes should not contain changed file names: %s", notes)
	}
	if strings.Contains(notes, "Release v0.1.0") {
		t.Fatalf("notes should not contain release title metadata: %s", notes)
	}
	if strings.Contains(notes, "## 文件变更统计") {
		t.Fatalf("notes should not contain diff stat section: %s", notes)
	}
	if strings.Contains(notes, "## 提交者统计") {
		t.Fatalf("notes should not contain author section: %s", notes)
	}
}

func TestCreateReleaseNotesBetweenTwoReleaseTags(t *testing.T) {
	repoDir := t.TempDir()
	scriptPath := copyReleaseScript(t, repoDir)

	runCmd(t, repoDir, "git", "init")
	runCmd(t, repoDir, "git", "config", "user.name", "Test User")
	runCmd(t, repoDir, "git", "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repoDir, "README.md"), "initial\n")
	runCmd(t, repoDir, "git", "add", "README.md")
	runCmd(t, repoDir, "git", "commit", "-m", "chore: initial release baseline")
	runCmd(t, repoDir, "git", "tag", "v0.1.0")

	writeFile(t, filepath.Join(repoDir, "feature.txt"), "feature\n")
	runCmd(t, repoDir, "git", "add", "feature.txt")
	runCmd(t, repoDir, "git", "commit", "-m", "feat(webui): show real skill totals")

	writeFile(t, filepath.Join(repoDir, "fix.txt"), "fix\n")
	runCmd(t, repoDir, "git", "add", "fix.txt")
	runCmd(t, repoDir, "git", "commit", "-m", "fix(api): remove project target endpoint")
	runCmd(t, repoDir, "git", "tag", "v0.2.0")

	writeFile(t, filepath.Join(repoDir, "after.txt"), "after\n")
	runCmd(t, repoDir, "git", "add", "after.txt")
	runCmd(t, repoDir, "git", "commit", "-m", "docs: update unreleased draft")

	outputPath := filepath.Join(repoDir, "release-notes.md")
	runCmd(t, repoDir, "bash", scriptPath, "--notes-only", "--yes", "--from", "v0.1.0", "--to", "v0.2.0", "--output", outputPath)

	notesContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read notes: %v", err)
	}

	notes := string(notesContent)
	if !strings.Contains(notes, "**webui**: show real skill totals") {
		t.Fatalf("expected feature between tags in notes: %s", notes)
	}
	if !strings.Contains(notes, "**api**: remove project target endpoint") {
		t.Fatalf("expected fix between tags in notes: %s", notes)
	}
	if strings.Contains(notes, "initial release baseline") {
		t.Fatalf("notes should exclude commits before --from: %s", notes)
	}
	if strings.Contains(notes, "update unreleased draft") {
		t.Fatalf("notes should exclude commits after --to: %s", notes)
	}
}

func copyReleaseScript(t *testing.T, repoDir string) string {
	t.Helper()

	scriptSource := filepath.Join("create-release.sh")
	scriptContent, err := os.ReadFile(scriptSource)
	if err != nil {
		t.Fatalf("read script: %v", err)
	}

	scriptsDir := filepath.Join(repoDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}

	scriptPath := filepath.Join(scriptsDir, "create-release.sh")
	if err := os.WriteFile(scriptPath, scriptContent, 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return scriptPath
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(output))
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
