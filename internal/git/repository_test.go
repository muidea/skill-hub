package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/muidea/skill-hub/internal/config"
	apperrors "github.com/muidea/skill-hub/pkg/errors"
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

func TestRepositorySetRemoteUpdatesExistingOrigin(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	firstRemote := filepath.Join(tmpDir, "first.git")
	secondRemote := filepath.Join(tmpDir, "second.git")

	runGitCommand(t, tmpDir, "init", "--bare", firstRemote)
	runGitCommand(t, tmpDir, "init", "--bare", secondRemote)
	runGitCommand(t, tmpDir, "init", repoDir)
	runGitCommand(t, repoDir, "remote", "add", "origin", firstRemote)

	repo, err := NewRepository(repoDir)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err := repo.SetRemote(secondRemote); err != nil {
		t.Fatalf("SetRemote() error = %v", err)
	}

	remotes, err := repo.GetRemote()
	if err != nil {
		t.Fatalf("GetRemote() error = %v", err)
	}
	if len(remotes) != 1 || remotes[0] != secondRemote {
		t.Fatalf("remote URLs = %v, want [%s]", remotes, secondRemote)
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

func TestSkillRepositoryPushChangesPushesExistingLocalCommit(t *testing.T) {
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
	writeSystemGitCommit(t, localDir, "README.md", "one\n", "initial commit")
	runGitCommand(t, localDir, "push", "-u", "origin", "main")
	writeSystemGitCommit(t, localDir, "README.md", "two\n", "local commit")

	repo, err := NewRepository(localDir)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	if err := (&SkillRepository{repo: repo}).PushChanges(""); err != nil {
		t.Fatalf("PushChanges() error = %v", err)
	}

	remoteContent := runGitCommand(t, remoteDir, "show", "main:README.md")
	if remoteContent != "two\n" {
		t.Fatalf("remote README = %q, want committed local content", remoteContent)
	}
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

func TestSkillRepositoryResolvesRemoteSkillConflictByHigherVersion(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}

	tmpDir := t.TempDir()
	t.Setenv("SKILL_HUB_HOME", tmpDir)
	defer config.ResetForTest()

	remoteDir := filepath.Join(tmpDir, "remote.git")
	seedDir := filepath.Join(tmpDir, "seed")
	localDir := filepath.Join(tmpDir, "repositories", "main")
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
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	runGitCommand(t, tmpDir, "init", "--bare", remoteDir)
	runGitCommand(t, tmpDir, "init", "-b", "main", seedDir)
	runGitCommand(t, seedDir, "config", "user.name", "tester")
	runGitCommand(t, seedDir, "config", "user.email", "tester@example.com")
	runGitCommand(t, seedDir, "remote", "add", "origin", remoteDir)
	writeSkillSystemGitCommit(t, seedDir, "demo", "1.0.0", "initial", "initial skill")
	runGitCommand(t, seedDir, "push", "origin", "main")

	runGitCommand(t, tmpDir, "clone", "--branch", "main", remoteDir, localDir)
	runGitCommand(t, localDir, "config", "user.name", "tester")
	runGitCommand(t, localDir, "config", "user.email", "tester@example.com")

	writeSkillSystemGitCommit(t, seedDir, "demo", "2.0.0", "remote", "remote higher version")
	runGitCommand(t, seedDir, "push", "origin", "main")

	writeSkillSystemGitCommit(t, localDir, "demo", "1.1.0", "local", "local lower version")

	repo, err := NewRepository(localDir)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}
	sr := &SkillRepository{repo: repo}
	if err := sr.resolveRemoteSkillUpdatesByVersion(); err != nil {
		t.Fatalf("resolveRemoteSkillUpdatesByVersion() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(localDir, "skills", "demo", "SKILL.md"))
	if err != nil {
		t.Fatalf("read resolved skill: %v", err)
	}
	if got := string(content); !strings.Contains(got, "version: 2.0.0") || !strings.Contains(got, "remote") {
		t.Fatalf("resolved skill should keep remote higher version, got:\n%s", got)
	}
	if output := runGitCommand(t, localDir, "status", "--porcelain"); output != "" {
		t.Fatalf("working tree should be clean after auto resolve, got %q", output)
	}
	if err := repo.Push(); err != nil {
		t.Fatalf("push resolved history: %v", err)
	}
}

func TestRepositoryCommitCleanWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}

	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, "local")
	runGitCommand(t, tmpDir, "init", "-b", "main", localDir)
	runGitCommand(t, localDir, "config", "user.name", "tester")
	runGitCommand(t, localDir, "config", "user.email", "tester@example.com")
	writeSystemGitCommit(t, localDir, "README.md", "one\n", "initial commit")

	repo, err := NewRepository(localDir)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	err = repo.Commit("should fail on clean repo")
	if err == nil || !strings.Contains(err.Error(), "没有要提交的更改") {
		t.Fatalf("Commit() error = %v, want 没有要提交的更改", err)
	}
}

func TestRepositoryCommitUsesSystemGitEnvAndIncludeIf(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}

	setupRepoWithInitialCommit := func(t *testing.T) (string, string) {
		t.Helper()
		tmpDir := t.TempDir()
		remoteDir := filepath.Join(tmpDir, "remote.git")
		seedDir := filepath.Join(tmpDir, "seed")
		localDir := filepath.Join(tmpDir, "local")

		runGitCommand(t, tmpDir, "init", "--bare", remoteDir)
		runGitCommand(t, tmpDir, "init", "-b", "main", seedDir)
		runGitCommand(t, seedDir, "config", "user.name", "seed")
		runGitCommand(t, seedDir, "config", "user.email", "seed@example.com")
		runGitCommand(t, seedDir, "remote", "add", "origin", remoteDir)
		writeSystemGitCommit(t, seedDir, "README.md", "one\n", "initial commit")
		writeSystemGitCommit(t, seedDir, "obsolete.txt", "bye\n", "add obsolete")
		runGitCommand(t, seedDir, "push", "-u", "origin", "main")

		runGitCommand(t, tmpDir, "clone", "--branch", "main", remoteDir, localDir)
		return remoteDir, localDir
	}

	t.Run("include.path in gitconfig", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
		unsetEnvForTest(t, "GIT_CONFIG_GLOBAL")
		unsetEnvForTest(t, "XDG_CONFIG_HOME")
		unsetEnvForTest(t, "GIT_AUTHOR_NAME")
		unsetEnvForTest(t, "GIT_AUTHOR_EMAIL")
		unsetEnvForTest(t, "GIT_COMMITTER_NAME")
		unsetEnvForTest(t, "GIT_COMMITTER_EMAIL")
		unsetEnvForTest(t, "EMAIL")

		remoteDir, localDir := setupRepoWithInitialCommit(t)

		includedCfg := filepath.Join(homeDir, ".gitconfig-user")
		if err := os.WriteFile(includedCfg, []byte("[user]\n\tname = Included User\n\temail = included@example.com\n"), 0644); err != nil {
			t.Fatalf("write included gitconfig: %v", err)
		}
		mainCfg := filepath.Join(homeDir, ".gitconfig")
		if err := os.WriteFile(mainCfg, []byte("[include]\n\tpath = "+includedCfg+"\n"), 0644); err != nil {
			t.Fatalf("write gitconfig: %v", err)
		}

		if err := os.WriteFile(filepath.Join(localDir, "README.md"), []byte("updated\n"), 0644); err != nil {
			t.Fatalf("modify README.md: %v", err)
		}
		if err := os.WriteFile(filepath.Join(localDir, "added.txt"), []byte("new\n"), 0644); err != nil {
			t.Fatalf("write added.txt: %v", err)
		}
		if err := os.Remove(filepath.Join(localDir, "obsolete.txt")); err != nil {
			t.Fatalf("remove obsolete.txt: %v", err)
		}

		repo, err := NewRepository(localDir)
		if err != nil {
			t.Fatalf("NewRepository() error = %v", err)
		}
		if err := (&SkillRepository{repo: repo}).PushChanges("include config commit"); err != nil {
			t.Fatalf("PushChanges() error = %v", err)
		}

		if got := runGitCommand(t, remoteDir, "show", "main:README.md"); got != "updated\n" {
			t.Fatalf("remote README.md = %q, want %q", got, "updated\n")
		}
		if got := runGitCommand(t, remoteDir, "show", "main:added.txt"); got != "new\n" {
			t.Fatalf("remote added.txt = %q, want %q", got, "new\n")
		}
		if err := exec.Command("git", "-C", remoteDir, "cat-file", "-e", "main:obsolete.txt").Run(); err == nil {
			t.Fatalf("remote obsolete.txt should have been deleted")
		}
	})

	t.Run("GIT_AUTHOR and GIT_COMMITTER env vars", func(t *testing.T) {
		homeDir := t.TempDir()
		t.Setenv("HOME", homeDir)
		t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
		unsetEnvForTest(t, "GIT_CONFIG_GLOBAL")
		unsetEnvForTest(t, "XDG_CONFIG_HOME")
		t.Setenv("GIT_AUTHOR_NAME", "Env Author")
		t.Setenv("GIT_AUTHOR_EMAIL", "env-author@example.com")
		t.Setenv("GIT_COMMITTER_NAME", "Env Committer")
		t.Setenv("GIT_COMMITTER_EMAIL", "env-committer@example.com")

		remoteDir, localDir := setupRepoWithInitialCommit(t)

		if err := os.WriteFile(filepath.Join(localDir, "env.txt"), []byte("from env\n"), 0644); err != nil {
			t.Fatalf("write env.txt: %v", err)
		}

		repo, err := NewRepository(localDir)
		if err != nil {
			t.Fatalf("NewRepository() error = %v", err)
		}
		if err := (&SkillRepository{repo: repo}).PushChanges("env identity commit"); err != nil {
			t.Fatalf("PushChanges() error = %v", err)
		}

		if got := runGitCommand(t, remoteDir, "show", "main:env.txt"); got != "from env\n" {
			t.Fatalf("remote env.txt = %q, want %q", got, "from env\n")
		}
	})
}

func TestRepositoryCommitMissingIdentityReturnsActionableError(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not available")
	}

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	unsetEnvForTest(t, "GIT_CONFIG_GLOBAL")
	unsetEnvForTest(t, "XDG_CONFIG_HOME")
	unsetEnvForTest(t, "GIT_AUTHOR_NAME")
	unsetEnvForTest(t, "GIT_AUTHOR_EMAIL")
	unsetEnvForTest(t, "GIT_COMMITTER_NAME")
	unsetEnvForTest(t, "GIT_COMMITTER_EMAIL")
	unsetEnvForTest(t, "EMAIL")

	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote.git")
	seedDir := filepath.Join(tmpDir, "seed")
	localDir := filepath.Join(tmpDir, "local")

	runGitCommand(t, tmpDir, "init", "--bare", remoteDir)
	runGitCommand(t, tmpDir, "init", "-b", "main", seedDir)
	runGitCommand(t, seedDir, "config", "user.name", "seed")
	runGitCommand(t, seedDir, "config", "user.email", "seed@example.com")
	runGitCommand(t, seedDir, "remote", "add", "origin", remoteDir)
	writeSystemGitCommit(t, seedDir, "README.md", "one\n", "initial commit")
	runGitCommand(t, seedDir, "push", "-u", "origin", "main")

	runGitCommand(t, tmpDir, "clone", "--branch", "main", remoteDir, localDir)
	// Ensure git does not fall back to OS username/hostname regardless of host FQDN.
	runGitCommand(t, localDir, "config", "user.useConfigOnly", "true")

	if err := os.WriteFile(filepath.Join(localDir, "README.md"), []byte("changed\n"), 0644); err != nil {
		t.Fatalf("modify README.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "untracked.txt"), []byte("new\n"), 0644); err != nil {
		t.Fatalf("write untracked.txt: %v", err)
	}

	repo, err := NewRepository(localDir)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}
	err = (&SkillRepository{repo: repo}).PushChanges("should fail")
	if err == nil {
		t.Fatal("PushChanges() succeeded without git user identity, want error")
	}
	if !apperrors.IsCode(err, apperrors.ErrGitOperation) {
		t.Fatalf("error code = %q, want %q (err=%v)", apperrors.Code(err), apperrors.ErrGitOperation, err)
	}
	msg := err.Error()
	if strings.Contains(msg, "author field is required") {
		t.Fatalf("error should not leak go-git ErrMissingAuthor: %v", err)
	}
	if !strings.Contains(msg, "git config --global user.name") || !strings.Contains(msg, "user.email") {
		t.Fatalf("error should contain actionable git config guidance, got: %v", err)
	}
	if staged := strings.TrimSpace(runGitCommand(t, localDir, "diff", "--cached", "--name-only")); staged != "" {
		t.Fatalf("failed commit should not stage files, got staged: %q", staged)
	}
}

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()
	prev, had := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unsetenv %s: %v", key, err)
	}
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
			return
		}
		_ = os.Unsetenv(key)
	})
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

func writeSkillSystemGitCommit(t *testing.T, dir, skillID, version, body, message string) {
	t.Helper()
	skillDir := filepath.Join(dir, "skills", skillID)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	content := "---\nname: Demo\nversion: " + version + "\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	runGitCommand(t, dir, "add", ".")
	runGitCommand(t, dir, "commit", "-m", message)
}
