package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"skill-hub/internal/config"
)

// Repository 表示一个Git仓库
type Repository struct {
	path       string
	repo       *git.Repository
	remoteURL  string
	remoteName string
}

// NewRepository 创建或打开一个Git仓库
func NewRepository(repoPath string) (*Repository, error) {
	// 确保目录存在
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	// 尝试打开现有仓库
	repo, err := git.PlainOpen(repoPath)
	if err == git.ErrRepositoryNotExists {
		// 仓库不存在，初始化新仓库
		repo, err = git.PlainInit(repoPath, false)
		if err != nil {
			return nil, fmt.Errorf("初始化Git仓库失败: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("打开Git仓库失败: %w", err)
	}

	return &Repository{
		path:       repoPath,
		repo:       repo,
		remoteName: "origin",
	}, nil
}

// NewSkillsRepository 创建技能仓库实例
func NewSkillsRepository() (*Repository, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return nil, err
	}

	repo, err := NewRepository(skillsDir)
	if err != nil {
		return nil, err
	}

	// 设置远程仓库URL
	if cfg.GitRemoteURL != "" {
		repo.remoteURL = cfg.GitRemoteURL
		if err := repo.SetRemote(cfg.GitRemoteURL); err != nil {
			return nil, fmt.Errorf("设置远程仓库失败: %w", err)
		}
	}

	return repo, nil
}

// SetRemote 设置远程仓库URL
func (r *Repository) SetRemote(url string) error {
	r.remoteURL = url

	// 删除现有远程（如果存在）
	_ = r.repo.DeleteRemote(r.remoteName)

	// 添加新远程
	_, err := r.repo.CreateRemote(&gitconfig.RemoteConfig{
		Name: r.remoteName,
		URLs: []string{url},
	})
	return err
}

// Clone 克隆远程仓库
func (r *Repository) Clone(url string) error {
	// 如果目录非空，先清理
	if entries, _ := os.ReadDir(r.path); len(entries) > 0 {
		// 备份现有内容
		backupDir := r.path + ".bak"
		if err := os.Rename(r.path, backupDir); err != nil {
			return fmt.Errorf("备份现有目录失败: %w", err)
		}
		// 重新创建空目录
		if err := os.MkdirAll(r.path, 0755); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}
	}

	// 准备克隆选项
	cloneOpts := &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	}

	// 根据URL类型设置认证
	if strings.HasPrefix(url, "git@") || strings.Contains(url, "ssh://") {
		// SSH URL
		auth, err := r.getSSHAuth()
		if err != nil {
			return fmt.Errorf("SSH认证失败: %w", err)
		}
		cloneOpts.Auth = auth
	} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		// HTTP/HTTPS URL
		auth, err := r.getAuth()
		if err != nil {
			return err
		}
		cloneOpts.Auth = auth
	}

	// 克隆仓库
	repo, err := git.PlainClone(r.path, false, cloneOpts)
	if err != nil {
		// 如果SSH克隆失败，尝试转换为HTTPS URL
		if strings.HasPrefix(url, "git@") {
			httpsURL := convertSSHToHTTPS(url)
			if httpsURL != "" {
				fmt.Printf("SSH克隆失败，尝试HTTPS URL: %s\n", httpsURL)
				cloneOpts.URL = httpsURL
				cloneOpts.Auth, _ = r.getAuth() // 使用HTTP认证
				repo, err = git.PlainClone(r.path, false, cloneOpts)
				if err == nil {
					fmt.Println("✅ 使用HTTPS URL克隆成功")
					r.repo = repo
					r.remoteURL = httpsURL // 更新为HTTPS URL
					return nil
				}
			}
		}

		if err != nil {
			// 提供更详细的错误信息
			errMsg := fmt.Sprintf("克隆仓库失败: %v", err)
			if strings.Contains(err.Error(), "SSH_AUTH_SOCK") {
				errMsg += "\nSSH认证失败: 请确保SSH agent正在运行或使用HTTPS URL"
			} else if strings.Contains(err.Error(), "authentication required") {
				errMsg += "\n认证失败: 请检查Git token配置或使用SSH key"
			}
			return fmt.Errorf(errMsg)
		}
	}

	r.repo = repo
	r.remoteURL = url
	return nil
}

// Pull 拉取最新更改
func (r *Repository) Pull() error {
	if r.remoteURL == "" {
		return fmt.Errorf("未设置远程仓库URL")
	}

	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("获取工作树失败: %w", err)
	}

	// 获取认证信息
	var auth transport.AuthMethod
	if strings.HasPrefix(r.remoteURL, "git@") || strings.Contains(r.remoteURL, "ssh://") {
		auth, err = r.getSSHAuth()
		if err != nil {
			return fmt.Errorf("SSH认证失败: %w", err)
		}
	} else {
		httpAuth, err := r.getAuth()
		if err != nil {
			return err
		}
		auth = httpAuth
	}

	err = worktree.Pull(&git.PullOptions{
		RemoteName:    r.remoteName,
		Auth:          auth,
		Progress:      os.Stdout,
		ReferenceName: plumbing.NewBranchReferenceName("main"),
		SingleBranch:  true,
	})

	if err == git.NoErrAlreadyUpToDate {
		return nil // 已经是最新
	}

	return err
}

// Push 推送本地更改
func (r *Repository) Push() error {
	if r.remoteURL == "" {
		return fmt.Errorf("未设置远程仓库URL")
	}

	auth, err := r.getAuth()
	if err != nil {
		return err
	}

	return r.repo.Push(&git.PushOptions{
		RemoteName: r.remoteName,
		Auth:       auth,
		Progress:   os.Stdout,
	})
}

// Commit 提交更改
func (r *Repository) Commit(message string) error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("获取工作树失败: %w", err)
	}

	// 添加所有更改
	_, err = worktree.Add(".")
	if err != nil {
		return fmt.Errorf("添加文件失败: %w", err)
	}

	// 检查是否有更改
	status, err := worktree.Status()
	if err != nil {
		return fmt.Errorf("检查状态失败: %w", err)
	}

	if status.IsClean() {
		return fmt.Errorf("没有要提交的更改")
	}

	// 提交更改
	_, err = worktree.Commit(message, &git.CommitOptions{
		All: true,
	})
	return err
}

// GetStatus 获取仓库状态
func (r *Repository) GetStatus() (string, error) {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("获取工作树失败: %w", err)
	}

	status, err := worktree.Status()
	if err != nil {
		return "", fmt.Errorf("获取状态失败: %w", err)
	}

	return status.String(), nil
}

// GetLatestCommit 获取最新提交信息
func (r *Repository) GetLatestCommit() (string, error) {
	ref, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("获取HEAD失败: %w", err)
	}

	commit, err := r.repo.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("获取提交对象失败: %w", err)
	}

	return fmt.Sprintf("%s: %s", commit.Hash.String()[:8], commit.Message), nil
}

// IsInitialized 检查仓库是否已初始化
func (r *Repository) IsInitialized() bool {
	return r.remoteURL != ""
}

// GetPath 获取仓库路径
func (r *Repository) GetPath() string {
	return r.path
}

// getAuth 获取认证信息
func (r *Repository) getAuth() (*http.BasicAuth, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	if cfg.GitToken != "" {
		return &http.BasicAuth{
			Username: "token", // GitHub等使用token作为用户名
			Password: cfg.GitToken,
		}, nil
	}

	return nil, nil // 无需认证
}

// ListBranches 列出所有分支
func (r *Repository) ListBranches() ([]string, error) {
	branches := []string{}

	// 获取本地分支
	iter, err := r.repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("获取分支失败: %w", err)
	}

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		branches = append(branches, ref.Name().Short())
		return nil
	})

	return branches, err
}

// CheckoutBranch 切换到指定分支
func (r *Repository) CheckoutBranch(branchName string) error {
	worktree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("获取工作树失败: %w", err)
	}

	// 检查分支是否存在
	branchRef := plumbing.NewBranchReferenceName(branchName)
	_, err = r.repo.Reference(branchRef, true)
	if err != nil {
		// 分支不存在，创建新分支
		headRef, err := r.repo.Head()
		if err != nil {
			return fmt.Errorf("获取HEAD失败: %w", err)
		}

		ref := plumbing.NewHashReference(branchRef, headRef.Hash())
		if err := r.repo.Storer.SetReference(ref); err != nil {
			return fmt.Errorf("创建分支失败: %w", err)
		}
	}

	return worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Create: false,
	})
}

// CreateBranch 创建新分支
func (r *Repository) CreateBranch(branchName string) error {
	headRef, err := r.repo.Head()
	if err != nil {
		return fmt.Errorf("获取HEAD失败: %w", err)
	}

	branchRef := plumbing.NewBranchReferenceName(branchName)
	ref := plumbing.NewHashReference(branchRef, headRef.Hash())

	return r.repo.Storer.SetReference(ref)
}

// MergeBranch 合并分支
func (r *Repository) MergeBranch(sourceBranch string) error {
	// 简化实现：切换到目标分支并拉取最新更改
	// 在实际实现中应该使用更复杂的合并逻辑
	return r.Pull()
}

// getSSHAuth 获取SSH认证信息
func (r *Repository) getSSHAuth() (transport.AuthMethod, error) {
	// 尝试使用SSH agent
	sshAuth, err := ssh.NewSSHAgentAuth("git")
	if err != nil {
		// 如果SSH agent不可用，尝试使用默认的SSH key
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("获取用户主目录失败: %w", err)
		}

		// 尝试常见的SSH key路径
		sshKeyPaths := []string{
			filepath.Join(homeDir, ".ssh", "id_rsa"),
			filepath.Join(homeDir, ".ssh", "id_ed25519"),
			filepath.Join(homeDir, ".ssh", "id_dsa"),
		}

		for _, keyPath := range sshKeyPaths {
			if _, err := os.Stat(keyPath); err == nil {
				sshAuth, err := ssh.NewPublicKeysFromFile("git", keyPath, "")
				if err == nil {
					return sshAuth, nil
				}
			}
		}

		return nil, fmt.Errorf("SSH认证失败: 请确保SSH agent运行或配置了SSH key")
	}

	return sshAuth, nil
}

// convertSSHToHTTPS 将SSH URL转换为HTTPS URL
func convertSSHToHTTPS(sshURL string) string {
	// 处理 git@github.com:user/repo.git 格式
	if strings.HasPrefix(sshURL, "git@") {
		parts := strings.Split(sshURL, ":")
		if len(parts) == 2 {
			host := strings.TrimPrefix(parts[0], "git@")
			repoPath := strings.TrimSuffix(parts[1], ".git")
			return fmt.Sprintf("https://%s/%s", host, repoPath)
		}
	}

	// 处理 ssh://git@github.com/user/repo.git 格式
	if strings.HasPrefix(sshURL, "ssh://") {
		sshURL = strings.TrimPrefix(sshURL, "ssh://")
		sshURL = strings.TrimPrefix(sshURL, "git@")
		sshURL = strings.Replace(sshURL, ":", "/", 1)
		return fmt.Sprintf("https://%s", strings.TrimSuffix(sshURL, ".git"))
	}

	return ""
}
