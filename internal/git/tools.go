package git

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
)

// Clone 克隆远程仓库到本地目录
func Clone(url, dir string) error {
	fmt.Printf("正在克隆仓库: %s -> %s\n", url, dir)

	// 确保目录不存在或为空
	if _, err := os.Stat(dir); err == nil {
		// 目录存在，检查是否为空
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("检查目录失败: %w", err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("目录 %s 不为空", dir)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("检查目录失败: %w", err)
	}

	// 创建目录
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 配置克隆选项
	options := &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
	}

	// 执行克隆
	_, err := git.PlainClone(dir, false, options)
	if err != nil {
		// 提供更详细的错误信息
		errMsg := fmt.Sprintf("克隆失败: %v", err)
		if strings.Contains(err.Error(), "SSH_AUTH_SOCK") {
			errMsg += "\nSSH认证失败: 请确保SSH agent正在运行或使用HTTPS URL"
		} else if strings.Contains(err.Error(), "authentication required") {
			errMsg += "\n认证失败: 请检查Git token配置或使用SSH key"
		}
		return fmt.Errorf("%s", errMsg)
	}

	fmt.Println("✅ 克隆完成")
	return nil
}

// Init 初始化新的Git仓库
func Init(dir string) error {
	fmt.Printf("正在初始化Git仓库: %s\n", dir)

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 初始化仓库
	_, err := git.PlainInit(dir, false)
	if err != nil {
		return fmt.Errorf("初始化失败: %w", err)
	}

	fmt.Println("✅ 初始化完成")
	return nil
}

// IsGitRepo 检查目录是否为Git仓库
func IsGitRepo(dir string) bool {
	_, err := git.PlainOpen(dir)
	return err == nil
}

// Pull 拉取远程仓库更新
func Pull(dir string) error {
	fmt.Printf("正在拉取更新: %s\n", dir)

	// 打开仓库
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return fmt.Errorf("打开仓库失败: %w", err)
	}

	// 获取工作树
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("获取工作树失败: %w", err)
	}

	// 获取远程URL
	remote, err := repo.Remote("origin")
	var remoteURL string
	if err == nil && remote != nil {
		if urls := remote.Config().URLs; len(urls) > 0 {
			remoteURL = urls[0]
		}
	}

	// 配置拉取选项
	options := &git.PullOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
	}

	// 执行拉取
	err = w.Pull(options)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		// 如果SSH拉取失败，尝试转换为HTTPS URL
		if strings.Contains(err.Error(), "SSH_AUTH_SOCK") && remoteURL != "" && strings.HasPrefix(remoteURL, "git@") {
			httpsURL := ConvertSSHToHTTPS(remoteURL)
			if httpsURL != "" {
				fmt.Printf("SSH拉取失败，尝试HTTPS URL: %s\n", httpsURL)

				// 更新远程URL为HTTPS
				if err := updateRemoteURL(repo, httpsURL); err != nil {
					fmt.Printf("⚠️  更新远程URL失败: %v\n", err)
				} else {
					// 重试拉取
					err = w.Pull(options)
					if err == nil || err == git.NoErrAlreadyUpToDate {
						fmt.Println("✅ 使用HTTPS URL拉取成功")
						if err == git.NoErrAlreadyUpToDate {
							fmt.Println("✅ 仓库已是最新")
						} else {
							fmt.Println("✅ 拉取完成")
						}
						return nil
					}
				}
			}
		}

		// 提供更详细的错误信息
		errMsg := fmt.Sprintf("拉取失败: %v", err)
		if strings.Contains(err.Error(), "SSH_AUTH_SOCK") {
			errMsg += "\nSSH认证失败: 请确保SSH agent正在运行或使用HTTPS URL"
			if remoteURL != "" && strings.HasPrefix(remoteURL, "git@") {
				httpsURL := ConvertSSHToHTTPS(remoteURL)
				if httpsURL != "" {
					errMsg += fmt.Sprintf("\n可以手动更新远程URL: git -C %s remote set-url origin %s", dir, httpsURL)
				}
			}
		} else if strings.Contains(err.Error(), "authentication required") {
			errMsg += "\n认证失败: 请检查Git token配置或使用SSH key"
		}
		return fmt.Errorf("%s", errMsg)
	}

	if err == git.NoErrAlreadyUpToDate {
		fmt.Println("✅ 仓库已是最新")
	} else {
		fmt.Println("✅ 拉取完成")
	}

	return nil
}

// GetCurrentCommit 获取当前提交哈希
func GetCurrentCommit(dir string) (string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", fmt.Errorf("打开仓库失败: %w", err)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("获取HEAD失败: %w", err)
	}

	return ref.Hash().String()[:8], nil // 返回短哈希
}

// updateRemoteURL 更新远程仓库URL
func updateRemoteURL(repo *git.Repository, newURL string) error {
	// 删除现有远程
	_ = repo.DeleteRemote("origin")

	// 添加新远程
	_, err := repo.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{newURL},
	})
	return err
}
