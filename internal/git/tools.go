package git

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/muidea/skill-hub/pkg/logging"
	"github.com/muidea/skill-hub/pkg/utils"
)

// RepositorySyncResult describes whether a repository can be safely synchronized
// without publishing or merging local commits.
type RepositorySyncResult struct {
	Status  string
	Message string
	Ahead   int
	Behind  int
}

// Clone 克隆远程仓库到本地目录
func Clone(url, dir string) error {
	logger := logging.GetGlobalLogger().WithOperation("Clone")
	logger.Info("正在克隆仓库", "url", url, "dir", dir)
	logging.DetailPrintf("正在克隆仓库: %s -> %s\n", url, dir)

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
		return utils.CreateDirErr(err, dir)
	}

	// 配置克隆选项
	options := &git.CloneOptions{
		URL:      url,
		Progress: logging.DetailWriter(),
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

	logging.DetailPrintln("✅ 克隆完成")
	return nil
}

// Init 初始化新的Git仓库
func Init(dir string) error {
	logging.DetailPrintf("正在初始化Git仓库: %s\n", dir)

	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return utils.CreateDirErr(err, dir)
	}

	// 初始化仓库
	_, err := git.PlainInit(dir, false)
	if err != nil {
		return fmt.Errorf("初始化失败: %w", err)
	}

	logging.DetailPrintln("✅ 初始化完成")
	return nil
}

// IsGitRepo 检查目录是否为Git仓库
func IsGitRepo(dir string) bool {
	_, err := git.PlainOpen(dir)
	return err == nil
}

// Pull 拉取远程仓库更新
func Pull(dir string) error {
	result, err := Sync(dir)
	if err != nil {
		return err
	}
	if result.Status == "divergent" || result.Status == "blocked_dirty" {
		return fmt.Errorf("%s", result.Message)
	}
	return nil
}

// Sync fetches the tracked remote branch and only fast-forwards the worktree.
// It never publishes local commits or creates merge commits.
func Sync(dir string) (*RepositorySyncResult, error) {
	logging.DetailPrintf("正在拉取更新: %s\n", dir)

	fetch := exec.Command("git", "-C", dir, "fetch")
	output, err := fetch.CombinedOutput()
	logging.LogCommandOutput("git", fetch.Args[1:], output, err)
	if err != nil {
		return nil, formatPullError(err, output)
	}

	statusCmd := exec.Command("git", "-C", dir, "status", "--porcelain=v2", "--branch")
	statusOutput, err := statusCmd.CombinedOutput()
	logging.LogCommandOutput("git", statusCmd.Args[1:], statusOutput, err)
	if err != nil {
		return nil, fmt.Errorf("获取同步状态失败: %s: %w", strings.TrimSpace(string(statusOutput)), err)
	}
	ahead, behind, dirty := parseBranchSyncStatus(string(statusOutput))
	result := &RepositorySyncResult{Ahead: ahead, Behind: behind}
	switch {
	case ahead > 0 && behind > 0:
		result.Status = "divergent"
		result.Message = fmt.Sprintf("本地领先 %d 个提交且落后远端 %d 个提交；请先合并或 rebase 后再同步", ahead, behind)
		return result, nil
	case behind > 0 && dirty:
		result.Status = "blocked_dirty"
		result.Message = fmt.Sprintf("远端有 %d 个提交待拉取，但工作区存在未提交更改；请先提交或暂存后再同步", behind)
		return result, nil
	case ahead > 0:
		result.Status = "local_ahead"
		result.Message = fmt.Sprintf("本地有 %d 个提交尚未推送；未执行 pull", ahead)
		logging.DetailPrintf("⚠️  %s\n", result.Message)
		return result, nil
	case behind == 0:
		result.Status = "up_to_date"
		result.Message = "远程仓库已是最新"
		logging.DetailPrintln("✅ 拉取完成")
		return result, nil
	}

	pull := exec.Command("git", "-C", dir, "pull", "--ff-only")
	output, err = pull.CombinedOutput()
	logging.LogCommandOutput("git", pull.Args[1:], output, err)
	if err != nil {
		return nil, formatPullError(err, output)
	}
	result.Status = "synced"
	result.Message = fmt.Sprintf("已拉取 %d 个远端提交", behind)
	logging.DetailPrintln("✅ 拉取完成")
	return result, nil
}

func parseBranchSyncStatus(status string) (ahead, behind int, dirty bool) {
	for _, line := range strings.Split(status, "\n") {
		if strings.HasPrefix(line, "# branch.ab ") {
			fields := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))
			if len(fields) == 2 {
				ahead, _ = strconv.Atoi(strings.TrimPrefix(fields[0], "+"))
				behind, _ = strconv.Atoi(strings.TrimPrefix(fields[1], "-"))
			}
			continue
		}
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "# ") {
			dirty = true
		}
	}
	return ahead, behind, dirty
}

func formatPullError(err error, output []byte) error {
	outputText := strings.TrimSpace(string(output))
	errMsg := fmt.Sprintf("拉取失败: %v", err)
	if outputText != "" {
		errMsg = fmt.Sprintf("拉取失败: %s: %v", outputText, err)
	}
	diagnosticText := err.Error() + "\n" + outputText
	if strings.Contains(diagnosticText, "SSH_AUTH_SOCK") {
		errMsg += "\nSSH认证失败: 请确保SSH agent正在运行或使用HTTPS URL"
	} else if strings.Contains(diagnosticText, "authentication required") ||
		strings.Contains(diagnosticText, "Permission denied") ||
		strings.Contains(diagnosticText, "unable to authenticate") {
		errMsg += "\n认证失败: 请检查Git token配置或使用SSH key"
	}
	return fmt.Errorf("%s", errMsg)
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
