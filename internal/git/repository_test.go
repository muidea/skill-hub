package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestConvertSSHToHTTPS(t *testing.T) {
	tests := []struct {
		name     string
		sshURL   string
		expected string
	}{
		{
			name:     "GitHub SSH URL",
			sshURL:   "git@github.com:muidea/skills-repo.git",
			expected: "https://github.com/muidea/skills-repo",
		},
		{
			name:     "GitLab SSH URL",
			sshURL:   "git@gitlab.com:group/project.git",
			expected: "https://gitlab.com/group/project",
		},
		{
			name:     "SSH protocol URL",
			sshURL:   "ssh://git@github.com/user/repo.git",
			expected: "https://github.com/user/repo",
		},
		{
			name:     "Invalid URL",
			sshURL:   "not-a-valid-url",
			expected: "",
		},
		{
			name:     "HTTPS URL should return empty",
			sshURL:   "https://github.com/user/repo.git",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertSSHToHTTPS(tt.sshURL)
			if result != tt.expected {
				t.Errorf("convertSSHToHTTPS(%q) = %q, want %q", tt.sshURL, result, tt.expected)
			}
		})
	}
}

func TestGetSSHAuth(t *testing.T) {
	// This is a basic test to ensure the function doesn't panic
	repo := &Repository{}
	_, err := repo.getSSHAuth()

	// We expect an error because we're not in a real environment with SSH keys
	// But the function should not panic
	if err == nil {
		t.Log("getSSHAuth returned no error (might be in CI environment with SSH agent)")
	} else {
		t.Logf("getSSHAuth returned expected error: %v", err)
	}
}

func TestRepositoryCheckRemoteUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote.git")
	seedDir := filepath.Join(tmpDir, "seed")
	localDir := filepath.Join(tmpDir, "local")

	if _, err := gogit.PlainInitWithOptions(remoteDir, &gogit.PlainInitOptions{
		Bare: true,
		InitOptions: gogit.InitOptions{
			DefaultBranch: plumbing.NewBranchReferenceName("main"),
		},
	}); err != nil {
		t.Fatalf("init remote: %v", err)
	}
	seedRepo, err := gogit.PlainInitWithOptions(seedDir, &gogit.PlainInitOptions{
		InitOptions: gogit.InitOptions{
			DefaultBranch: plumbing.NewBranchReferenceName("main"),
		},
	})
	if err != nil {
		t.Fatalf("init seed: %v", err)
	}
	seedWorktree, err := seedRepo.Worktree()
	if err != nil {
		t.Fatalf("seed worktree: %v", err)
	}
	if _, err := seedRepo.CreateRemote(&gitconfig.RemoteConfig{Name: "origin", URLs: []string{remoteDir}}); err != nil {
		t.Fatalf("create seed remote: %v", err)
	}

	writeAndCommit(t, seedWorktree, seedDir, "one")
	if err := seedRepo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []gitconfig.RefSpec{"refs/heads/main:refs/heads/main"},
	}); err != nil {
		t.Fatalf("push initial: %v", err)
	}

	if _, err := gogit.PlainClone(localDir, false, &gogit.CloneOptions{
		URL:           remoteDir,
		ReferenceName: plumbing.NewBranchReferenceName("main"),
		SingleBranch:  true,
	}); err != nil {
		t.Fatalf("clone local: %v", err)
	}
	repo, err := NewRepository(localDir)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}

	status, err := repo.CheckRemoteUpdates()
	if err != nil {
		t.Fatalf("CheckRemoteUpdates up-to-date error: %v", err)
	}
	if status.Status != "up_to_date" || status.HasUpdates {
		t.Fatalf("status = %+v, want up_to_date without updates", status)
	}

	writeAndCommit(t, seedWorktree, seedDir, "two")
	if err := seedRepo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []gitconfig.RefSpec{"refs/heads/main:refs/heads/main"},
	}); err != nil {
		t.Fatalf("push update: %v", err)
	}

	status, err = repo.CheckRemoteUpdates()
	if err != nil {
		t.Fatalf("CheckRemoteUpdates update error: %v", err)
	}
	if status.Status != "updates_available" || !status.HasUpdates || status.Behind != 1 || status.Ahead != 0 {
		t.Fatalf("status = %+v, want updates_available behind=1 ahead=0", status)
	}
}

func TestRepositoryCheckRemoteUpdatesNoRemote(t *testing.T) {
	repo, err := NewRepository(t.TempDir())
	if err != nil {
		t.Fatalf("NewRepository error: %v", err)
	}
	status, err := repo.CheckRemoteUpdates()
	if err != nil {
		t.Fatalf("CheckRemoteUpdates error: %v", err)
	}
	if status.Status != "no_remote" {
		t.Fatalf("status = %+v, want no_remote", status)
	}
}

func TestRepositoryPushUsesSystemGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}

	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote.git")
	localDir := filepath.Join(tmpDir, "local")

	runGitCommand(t, tmpDir, "init", "--bare", remoteDir)
	runGitCommand(t, tmpDir, "init", "-b", "main", localDir)
	runGitCommand(t, localDir, "config", "user.name", "tester")
	runGitCommand(t, localDir, "config", "user.email", "tester@example.com")
	runGitCommand(t, localDir, "remote", "add", "origin", remoteDir)

	if err := os.WriteFile(filepath.Join(localDir, "README.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGitCommand(t, localDir, "add", "README.md")
	runGitCommand(t, localDir, "commit", "-m", "initial commit")

	repo, err := NewRepository(localDir)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}
	if err := repo.Push(); err != nil {
		t.Fatalf("Push error: %v", err)
	}

	runGitCommand(t, remoteDir, "rev-parse", "--verify", "refs/heads/main")
}

func TestRepositoryPullUsesSystemGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}

	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote.git")
	seedDir := filepath.Join(tmpDir, "seed")
	localDir := filepath.Join(tmpDir, "local")

	runGitCommand(t, tmpDir, "init", "--bare", remoteDir)
	runGitCommand(t, tmpDir, "init", "-b", "main", seedDir)
	runGitCommand(t, seedDir, "config", "user.name", "tester")
	runGitCommand(t, seedDir, "config", "user.email", "tester@example.com")
	runGitCommand(t, seedDir, "remote", "add", "origin", remoteDir)
	writeSystemGitCommit(t, seedDir, "README.md", "one\n", "initial commit")
	runGitCommand(t, seedDir, "push", "origin", "main")

	runGitCommand(t, tmpDir, "clone", "--branch", "main", remoteDir, localDir)
	writeSystemGitCommit(t, seedDir, "README.md", "two\n", "update readme")
	runGitCommand(t, seedDir, "push", "origin", "main")

	repo, err := NewRepository(localDir)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}
	if err := repo.Pull(); err != nil {
		t.Fatalf("Pull error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(localDir, "README.md"))
	if err != nil {
		t.Fatalf("read pulled file: %v", err)
	}
	if string(content) != "two\n" {
		t.Fatalf("content = %q, want pulled update", string(content))
	}
}

func TestPullUsesSystemGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}

	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote.git")
	seedDir := filepath.Join(tmpDir, "seed")
	localDir := filepath.Join(tmpDir, "local")

	runGitCommand(t, tmpDir, "init", "--bare", remoteDir)
	runGitCommand(t, tmpDir, "init", "-b", "main", seedDir)
	runGitCommand(t, seedDir, "config", "user.name", "tester")
	runGitCommand(t, seedDir, "config", "user.email", "tester@example.com")
	runGitCommand(t, seedDir, "remote", "add", "origin", remoteDir)
	writeSystemGitCommit(t, seedDir, "README.md", "one\n", "initial commit")
	runGitCommand(t, seedDir, "push", "origin", "main")

	runGitCommand(t, tmpDir, "clone", "--branch", "main", remoteDir, localDir)
	writeSystemGitCommit(t, seedDir, "README.md", "two\n", "update readme")
	runGitCommand(t, seedDir, "push", "origin", "main")

	if err := Pull(localDir); err != nil {
		t.Fatalf("Pull error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(localDir, "README.md"))
	if err != nil {
		t.Fatalf("read pulled file: %v", err)
	}
	if string(content) != "two\n" {
		t.Fatalf("content = %q, want pulled update", string(content))
	}
}

func writeAndCommit(t *testing.T, worktree *gogit.Worktree, repoDir, content string) {
	t.Helper()
	filePath := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(filePath, []byte(content+"\n"), 0644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("worktree add: %v", err)
	}
	if _, err := worktree.Commit("update "+content, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "tester",
			Email: "tester@example.com",
			When:  time.Now(),
		},
	}); err != nil {
		t.Fatalf("worktree commit: %v", err)
	}
}

func runGitCommand(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %s: %v", args, string(output), err)
	}
	return string(output)
}

func writeSystemGitCommit(t *testing.T, dir, fileName, content, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", fileName, err)
	}
	runGitCommand(t, dir, "add", fileName)
	runGitCommand(t, dir, "commit", "-m", message)
}
