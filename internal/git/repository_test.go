package git

import (
	"os"
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
