package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skill-hub/internal/config"
	"skill-hub/internal/multirepo"
	"skill-hub/internal/state"

	"github.com/spf13/cobra"
)

var (
	feedbackDryRun bool
	feedbackForce  bool
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback <id>",
	Short: "将项目工作区技能修改内容更新至到本地仓库",
	Long: `将项目工作区本地的技能修改同步回本地技能仓库。

此命令会：
1. 提取项目工作区本地文件内容
2. 与本地仓库源文件对比，显示差异
3. 经用户确认后更新本地仓库文件
4. 更新 registry.json 中的版本/哈希信息

使用 --dry-run 参数演习模式，仅显示将要同步的差异。
使用 --force 参数强制更新，即使有冲突也继续执行。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFeedback(args[0])
	},
}

func init() {
	feedbackCmd.Flags().BoolVar(&feedbackDryRun, "dry-run", false, "演习模式，仅显示将要同步的差异")
	feedbackCmd.Flags().BoolVar(&feedbackForce, "force", false, "强制更新，即使有冲突也继续执行")
}

func runFeedback(skillID string) error {
	// 检查init依赖（规范4.11：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	fmt.Printf("收集技能 '%s' 的反馈...\n", skillID)

	// 获取当前目录
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 检查项目工作区状态（规范4.11：检查当前目录是否存在于state.json中）
	_, err = EnsureProjectWorkspace(cwd, "")
	if err != nil {
		return fmt.Errorf("检查项目工作区失败: %w", err)
	}

	// 检查技能是否在项目工作区中启用
	stateManager, err := state.NewStateManager()
	if err != nil {
		return fmt.Errorf("初始化状态管理器失败: %w", err)
	}

	// 检查项目是否已启用该技能
	hasSkill, err := stateManager.ProjectHasSkill(cwd, skillID)
	if err != nil {
		return fmt.Errorf("检查项目技能状态失败: %w", err)
	}

	if !hasSkill {
		return fmt.Errorf("技能 '%s' 未在项目工作区中启用", skillID)
	}

	// 检查项目工作区本地技能目录
	projectSkillDir := filepath.Join(cwd, ".agents", "skills", skillID)
	projectSkillPath := filepath.Join(projectSkillDir, "SKILL.md")
	if _, err := os.Stat(projectSkillPath); os.IsNotExist(err) {
		return fmt.Errorf("项目工作区中未找到技能文件: %s", projectSkillPath)
	}

	// 读取项目工作区文件内容
	projectContent, err := os.ReadFile(projectSkillPath)
	if err != nil {
		return fmt.Errorf("读取项目工作区文件失败: %w", err)
	}

	// 创建多仓库管理器
	repoManager, err := multirepo.NewManager()
	if err != nil {
		return fmt.Errorf("初始化多仓库管理器失败: %w", err)
	}

	// 检查技能是否在默认仓库中存在
	skillExists, err := repoManager.CheckSkillInDefaultRepository(skillID)
	if err != nil {
		return fmt.Errorf("检查技能存在状态失败: %w", err)
	}

	// 获取默认仓库路径
	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("获取配置失败: %w", err)
	}

	defaultRepo, err := cfg.GetArchiveRepository()
	if err != nil {
		return fmt.Errorf("获取默认仓库失败: %w", err)
	}

	repoDir, err := config.GetRepositoryPath(defaultRepo.Name)
	if err != nil {
		return fmt.Errorf("获取仓库路径失败: %w", err)
	}

	repoSkillDir := filepath.Join(repoDir, "skills", skillID)
	repoSkillPath := filepath.Join(repoSkillDir, "SKILL.md")

	var repoContent []byte
	if skillExists {
		// 技能在仓库中存在，读取仓库文件内容
		repoContent, err = os.ReadFile(repoSkillPath)
		if err != nil {
			return fmt.Errorf("读取本地仓库文件失败: %w", err)
		}
	} else {
		// 技能在仓库中不存在，这是新建的技能
		fmt.Printf("ℹ️  技能 '%s' 在本地仓库中不存在，将作为新技能创建\n", skillID)
		repoContent = []byte{} // 空内容，表示新建
	}

	// 比较SKILL.md文件内容
	projectStr := strings.TrimSpace(string(projectContent))
	repoStr := strings.TrimSpace(string(repoContent))

	// 检查整个目录的差异
	changes, err := compareSkillDirectories(projectSkillDir, repoSkillDir, skillExists)
	if err != nil {
		return fmt.Errorf("比较技能目录失败: %w", err)
	}

	// 如果是新建技能（仓库内容为空）
	if !skillExists {
		fmt.Println("\n📝 新建技能内容:")
		fmt.Println("========================================")
		fmt.Printf("技能目录: %s\n", skillID)
		fmt.Printf("文件数量: %d\n", len(changes))
		for _, change := range changes {
			fmt.Printf("  - %s\n", change)
		}
		fmt.Println("========================================")
	} else if len(changes) == 0 && projectStr == repoStr {
		// 技能已存在且内容相同
		fmt.Println("✅ 技能内容未修改")
		return nil
	} else {
		// 显示差异
		fmt.Println("\n🔍 检测到修改:")
		fmt.Println("========================================")
		fmt.Printf("技能目录: %s\n", skillID)
		fmt.Printf("修改文件数: %d\n", len(changes))

		if len(changes) > 0 {
			fmt.Println("\n修改的文件:")
			for _, change := range changes {
				fmt.Printf("  - %s\n", change)
			}
		}

		// 如果SKILL.md有修改，显示内容差异
		if projectStr != repoStr {
			fmt.Println("\nSKILL.md 内容差异:")
			fmt.Println("行号 | 修改前                      | 修改后")
			fmt.Println("-----|---------------------------|---------------------------")

			projectLines := strings.Split(projectStr, "\n")
			repoLines := strings.Split(repoStr, "\n")
			maxLines := len(projectLines)
			if len(repoLines) > maxLines {
				maxLines = len(repoLines)
			}

			for i := 0; i < maxLines; i++ {
				var projectLine, repoLine string
				if i < len(projectLines) {
					projectLine = projectLines[i]
				}
				if i < len(repoLines) {
					repoLine = repoLines[i]
				}

				if projectLine != repoLine {
					lineNum := i + 1
					fmt.Printf("%4d | %-25s | %-25s\n", lineNum, repoLine, projectLine)
				}
			}
		}
	}

	fmt.Println("========================================")

	// 如果是演习模式，只显示差异
	if feedbackDryRun {
		fmt.Println("\n✅ 演习模式完成，未进行实际修改")
		return nil
	}

	// 如果是强制模式，直接更新
	if feedbackForce {
		fmt.Println("\n🔧 强制模式，直接更新本地仓库...")
	} else {
		// 确认反馈
		fmt.Print("\n是否将这些修改更新到本地仓库？ [y/N]: ")
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(response)

		if response != "y" && response != "Y" {
			fmt.Println("❌ 取消反馈操作")
			return nil
		}
	}

	// 更新本地仓库文件
	// 确保目录存在
	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}

	// 复制整个技能目录
	if err := copySkillDirectory(projectSkillDir, repoSkillDir); err != nil {
		return fmt.Errorf("复制技能目录失败: %w", err)
	}

	fmt.Println("✓ 更新本地仓库文件")

	// 在多仓库模式下，不再更新registry.json
	// 技能已归档到默认仓库
	fmt.Println("✓ 技能已归档到默认仓库")

	fmt.Println("\n✅ 反馈完成！")
	fmt.Printf("技能 '%s' 已保存到默认仓库: %s\n", skillID, defaultRepo.Name)
	fmt.Println("使用 'skill-hub push' 同步到远程仓库")

	return nil
}

// compareSkillDirectories 比较两个技能目录的差异
func compareSkillDirectories(projectDir, repoDir string, repoExists bool) ([]string, error) {
	var changes []string

	// 如果仓库目录不存在，则所有文件都是新增的
	if !repoExists {
		err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.IsDir() {
				relPath, err := filepath.Rel(projectDir, path)
				if err != nil {
					return err
				}
				changes = append(changes, fmt.Sprintf("新增: %s", relPath))
			}
			return nil
		})
		return changes, err
	}

	// 收集项目目录中的所有文件
	projectFiles := make(map[string]bool)
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(projectDir, path)
			if err != nil {
				return err
			}
			projectFiles[relPath] = true
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 收集仓库目录中的所有文件，并比较
	err = filepath.Walk(repoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(repoDir, path)
			if err != nil {
				return err
			}

			projectPath := filepath.Join(projectDir, relPath)
			repoPath := path

			// 检查文件是否存在
			if _, err := os.Stat(projectPath); os.IsNotExist(err) {
				// 文件在项目目录中不存在，可能被删除
				changes = append(changes, fmt.Sprintf("删除: %s", relPath))
			} else {
				// 比较文件内容
				projectContent, err1 := os.ReadFile(projectPath)
				repoContent, err2 := os.ReadFile(repoPath)

				if err1 != nil || err2 != nil {
					// 读取错误，标记为修改
					changes = append(changes, fmt.Sprintf("修改: %s (读取错误)", relPath))
				} else if string(projectContent) != string(repoContent) {
					// 内容不同
					changes = append(changes, fmt.Sprintf("修改: %s", relPath))
				}

				// 从projectFiles中移除，表示已处理
				delete(projectFiles, relPath)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 剩余在projectFiles中的文件是新增的
	for relPath := range projectFiles {
		changes = append(changes, fmt.Sprintf("新增: %s", relPath))
	}

	return changes, nil
}

// copySkillDirectory 复制整个技能目录，同步删除操作
func copySkillDirectory(srcDir, dstDir string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 收集源目录中的所有文件
	srcFiles := make(map[string]bool)
	err := filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(srcDir, srcPath)
			if err != nil {
				return err
			}
			srcFiles[relPath] = true
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("遍历源目录失败: %w", err)
	}

	// 收集目标目录中的所有文件，用于删除操作
	dstFiles := make(map[string]bool)
	err = filepath.Walk(dstDir, func(dstPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(dstDir, dstPath)
			if err != nil {
				return err
			}
			dstFiles[relPath] = true
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("遍历目标目录失败: %w", err)
	}

	// 复制源目录中的所有文件
	for relPath := range srcFiles {
		srcPath := filepath.Join(srcDir, relPath)
		dstPath := filepath.Join(dstDir, relPath)

		// 确保目标目录存在
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("创建目录失败 %s: %w", filepath.Dir(dstPath), err)
		}

		// 读取源文件
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("读取文件失败 %s: %w", srcPath, err)
		}

		// 获取文件权限
		info, err := os.Stat(srcPath)
		if err != nil {
			return fmt.Errorf("获取文件权限失败 %s: %w", srcPath, err)
		}

		// 写入目标文件
		if err := os.WriteFile(dstPath, content, info.Mode()); err != nil {
			return fmt.Errorf("写入文件失败 %s: %w", dstPath, err)
		}

		// 从dstFiles中移除，表示已处理
		delete(dstFiles, relPath)
	}

	// 删除目标目录中多余的文件（在源目录中不存在的文件）
	for relPath := range dstFiles {
		dstPath := filepath.Join(dstDir, relPath)
		if err := os.Remove(dstPath); err != nil {
			return fmt.Errorf("删除文件失败 %s: %w", dstPath, err)
		}
	}

	// 清理空目录（可选）
	return nil
}
