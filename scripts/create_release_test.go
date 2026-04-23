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

func TestCreateReleaseNotesPreferTrackedVersionDocument(t *testing.T) {
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
	runCmd(t, repoDir, "git", "commit", "-m", "feat(global): add managed skill release flow")

	curatedNotes := "# v0.2.0 Managed Skill Release\n\nCurated release notes from docs.\n"
	if err := os.MkdirAll(filepath.Join(repoDir, "docs"), 0755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	writeFile(t, filepath.Join(repoDir, "docs", "release-notes-v0.2.0-managed-skills.md"), curatedNotes)
	runCmd(t, repoDir, "git", "add", "docs/release-notes-v0.2.0-managed-skills.md")
	runCmd(t, repoDir, "git", "commit", "-m", "docs: add managed skill release notes")

	outputPath := filepath.Join(repoDir, "release-notes.md")
	runCmd(t, repoDir, "bash", scriptPath, "--notes-only", "--yes", "--version", "0.2.0", "--from", "v0.1.0", "--output", outputPath)

	notesContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read notes: %v", err)
	}

	notes := string(notesContent)
	if notes != curatedNotes {
		t.Fatalf("expected tracked release note document, got: %s", notes)
	}
	if strings.Contains(notes, "add managed skill release flow") {
		t.Fatalf("notes should use the tracked document instead of generated commit subjects: %s", notes)
	}
}

func TestCreateReleaseNotesFailsOnMultipleTrackedVersionDocuments(t *testing.T) {
	repoDir := t.TempDir()
	scriptPath := copyReleaseScript(t, repoDir)

	runCmd(t, repoDir, "git", "init")
	runCmd(t, repoDir, "git", "config", "user.name", "Test User")
	runCmd(t, repoDir, "git", "config", "user.email", "test@example.com")

	writeFile(t, filepath.Join(repoDir, "README.md"), "initial\n")
	runCmd(t, repoDir, "git", "add", "README.md")
	runCmd(t, repoDir, "git", "commit", "-m", "chore: initial release baseline")
	runCmd(t, repoDir, "git", "tag", "v0.1.0")

	if err := os.MkdirAll(filepath.Join(repoDir, "docs"), 0755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	writeFile(t, filepath.Join(repoDir, "docs", "release-notes-v0.2.0-first.md"), "first\n")
	writeFile(t, filepath.Join(repoDir, "docs", "release-notes-v0.2.0-second.md"), "second\n")
	runCmd(t, repoDir, "git", "add", "docs/release-notes-v0.2.0-first.md", "docs/release-notes-v0.2.0-second.md")
	runCmd(t, repoDir, "git", "commit", "-m", "docs: add duplicate release notes")

	outputPath := filepath.Join(repoDir, "release-notes.md")
	cmd := exec.Command("bash", scriptPath, "--notes-only", "--yes", "--version", "0.2.0", "--from", "v0.1.0", "--output", outputPath)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected duplicate tracked release notes to fail, output:\n%s", string(output))
	}

	notesOutput := string(output)
	if !strings.Contains(notesOutput, "release-notes-v0.2.0-first.md") ||
		!strings.Contains(notesOutput, "release-notes-v0.2.0-second.md") {
		t.Fatalf("expected duplicate release note paths in output, got:\n%s", notesOutput)
	}
}

func TestCreateReleasePushesCurrentBranchBeforeTag(t *testing.T) {
	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote.git")
	seedDir := filepath.Join(tmpDir, "seed")
	releaseDir := filepath.Join(tmpDir, "release")

	runCmd(t, tmpDir, "git", "init", "--bare", "--initial-branch", "master", remoteDir)

	if err := os.MkdirAll(seedDir, 0755); err != nil {
		t.Fatalf("mkdir seed repo: %v", err)
	}
	runCmd(t, seedDir, "git", "init", "--initial-branch", "master")
	runCmd(t, seedDir, "git", "config", "user.name", "Test User")
	runCmd(t, seedDir, "git", "config", "user.email", "test@example.com")
	writeFile(t, filepath.Join(seedDir, "README.md"), "initial\n")
	runCmd(t, seedDir, "git", "add", "README.md")
	runCmd(t, seedDir, "git", "commit", "-m", "chore: initial release")
	runCmd(t, seedDir, "git", "tag", "v0.1.0")
	runCmd(t, seedDir, "git", "remote", "add", "origin", remoteDir)
	runCmd(t, seedDir, "git", "push", "origin", "master")
	runCmd(t, seedDir, "git", "push", "origin", "v0.1.0")

	runCmd(t, tmpDir, "git", "clone", remoteDir, releaseDir)
	runCmd(t, releaseDir, "git", "config", "user.name", "Test User")
	runCmd(t, releaseDir, "git", "config", "user.email", "test@example.com")
	scriptPath := copyReleaseScript(t, releaseDir)

	writeFile(t, filepath.Join(releaseDir, "fix.txt"), "release fix\n")
	runCmd(t, releaseDir, "git", "add", "fix.txt")
	runCmd(t, releaseDir, "git", "commit", "-m", "fix(release): push branch before tag")
	localHead := strings.TrimSpace(runCmdOutput(t, releaseDir, "git", "rev-parse", "HEAD"))

	fakeBin := createFakeMake(t, tmpDir)
	env := append(os.Environ(), "LC_ALL=C", "PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	runCmdEnv(t, releaseDir, env, "bash", scriptPath, "--yes", "--version", "0.2.0")

	remoteHead := firstField(t, runCmdOutput(t, releaseDir, "git", "ls-remote", "origin", "refs/heads/master"))
	if remoteHead != localHead {
		t.Fatalf("remote master was not pushed before release: got %s want %s", remoteHead, localHead)
	}

	tagHead := firstField(t, runCmdOutput(t, releaseDir, "git", "ls-remote", "origin", "refs/tags/v0.2.0^{}"))
	if tagHead != localHead {
		t.Fatalf("remote release tag points to %s, want %s", tagHead, localHead)
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

func runCmdEnv(t *testing.T, dir string, env []string, name string, args ...string) {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(output))
	}
}

func runCmdOutput(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(output))
	}
	return string(output)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func createFakeMake(t *testing.T, dir string) string {
	t.Helper()

	fakeBin := filepath.Join(dir, "fake-bin")
	if err := os.MkdirAll(fakeBin, 0755); err != nil {
		t.Fatalf("mkdir fake bin: %v", err)
	}

	makePath := filepath.Join(fakeBin, "make")
	makeScript := `#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
    test)
        exit 0
        ;;
    clean)
        rm -rf bin
        ;;
    build)
        version="dev"
        for arg in "$@"; do
            case "$arg" in
                VERSION=*) version="${arg#VERSION=}" ;;
            esac
        done
        mkdir -p bin
        cat > bin/skill-hub <<EOF
#!/usr/bin/env bash
echo "skill-hub version $version (commit: test, built: now)"
EOF
        chmod +x bin/skill-hub
        ;;
    *)
        echo "unexpected fake make command: $*" >&2
        exit 1
        ;;
esac
`
	if err := os.WriteFile(makePath, []byte(makeScript), 0755); err != nil {
		t.Fatalf("write fake make: %v", err)
	}
	return fakeBin
}

func firstField(t *testing.T, value string) string {
	t.Helper()

	fields := strings.Fields(value)
	if len(fields) == 0 {
		t.Fatalf("expected command output to contain at least one field, got %q", value)
	}
	return fields[0]
}
