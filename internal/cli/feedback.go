package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"skill-hub/internal/config"
	"skill-hub/internal/multirepo"
	"skill-hub/internal/state"
	"skill-hub/pkg/skill"
	"skill-hub/pkg/utils"
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
3. 检查版本号，必要时自动升级 patch 版本
4. 经用户确认后更新本地仓库文件

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
	if err := CheckInitDependency(); err != nil {
		return err
	}

	fmt.Printf("收集技能 '%s' 的反馈...\n", skillID)

	cwd, err := os.Getwd()
	if err != nil {
		return utils.GetCwdErr(err)
	}

	_, err = EnsureProjectWorkspace(cwd, "")
	if err != nil {
		return fmt.Errorf("检查项目工作区失败: %w", err)
	}

	stateManager, err := state.NewStateManager()
	if err != nil {
		return fmt.Errorf("初始化状态管理器失败: %w", err)
	}

	hasSkill, err := stateManager.ProjectHasSkill(cwd, skillID)
	if err != nil {
		return fmt.Errorf("检查项目技能状态失败: %w", err)
	}

	if !hasSkill {
		return fmt.Errorf("技能 '%s' 未在项目工作区中启用", skillID)
	}

	projectSkillDir := filepath.Join(cwd, ".agents", "skills", skillID)
	projectSkillPath := filepath.Join(projectSkillDir, "SKILL.md")
	if _, err := os.Stat(projectSkillPath); os.IsNotExist(err) {
		return fmt.Errorf("项目工作区中未找到技能文件: %s", projectSkillPath)
	}

	projectContent, err := os.ReadFile(projectSkillPath)
	if err != nil {
		return fmt.Errorf("读取项目工作区文件失败: %w", err)
	}

	repoManager, err := multirepo.NewManager()
	if err != nil {
		return fmt.Errorf("初始化多仓库管理器失败: %w", err)
	}

	skillExists, err := repoManager.CheckSkillInDefaultRepository(skillID)
	if err != nil {
		return fmt.Errorf("检查技能存在状态失败: %w", err)
	}

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
		repoContent, err = os.ReadFile(repoSkillPath)
		if err != nil {
			return fmt.Errorf("读取本地仓库文件失败: %w", err)
		}
	} else {
		fmt.Printf("ℹ️  技能 '%s' 在本地仓库中不存在，将作为新技能创建\n", skillID)
		repoContent = []byte{}
	}

	projectStr := strings.TrimSpace(string(projectContent))
	repoStr := strings.TrimSpace(string(repoContent))

	changes, err := compareSkillDirectories(projectSkillDir, repoSkillDir, skillExists)
	if err != nil {
		return fmt.Errorf("比较技能目录失败: %w", err)
	}

	hasContentChanges := projectStr != repoStr

	if !skillExists {
		fmt.Println("\n📝 新建技能内容:")
		fmt.Println("========================================")
		fmt.Printf("技能目录: %s\n", skillID)
		fmt.Printf("文件数量: %d\n", len(changes))
		for _, change := range changes {
			fmt.Printf("  - %s\n", change)
		}
		fmt.Println("========================================")
	} else if len(changes) == 0 && !hasContentChanges {
		fmt.Println("✅ 技能内容未修改")
		return nil
	} else {
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

		if hasContentChanges {
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

	versionUpdated := false
	if (hasContentChanges || len(changes) > 0) && !feedbackDryRun {
		projectVersion := getSkillVersionFromContent(projectContent)
		repoVersion := "0.0.0"
		if skillExists && len(repoContent) > 0 {
			repoVersion = getSkillVersionFromContent(repoContent)
		}

		if compareVersions(projectVersion, repoVersion) <= 0 {
			newVersion := bumpPatchVersion(repoVersion)
			fmt.Printf("\n🔧 自动升级版本号: %s -> %s\n", projectVersion, newVersion)

			if err := updateSkillMdVersion(projectSkillPath, newVersion); err != nil {
				return fmt.Errorf("更新版本号失败: %w", err)
			}
			fmt.Printf("✓ 已更新项目工作区 SKILL.md 版本号\n")

			_, err = os.ReadFile(projectSkillPath)
			if err != nil {
				return fmt.Errorf("重新读取项目文件失败: %w", err)
			}
			versionUpdated = true
		} else {
			fmt.Printf("\n✓ 使用用户指定的版本号: %s\n", projectVersion)
		}
	}

	fmt.Println("========================================")

	if feedbackDryRun {
		fmt.Println("\n✅ 演习模式完成，未进行实际修改")
		return nil
	}

	if feedbackForce {
		fmt.Println("\n🔧 强制模式，直接更新本地仓库...")
	} else {
		fmt.Print("\n是否将这些修改更新到本地仓库？ [y/N]: ")
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(response)

		if response != "y" && response != "Y" {
			fmt.Println("❌ 取消反馈操作")
			return nil
		}
	}

	if err := os.MkdirAll(repoSkillDir, 0755); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}

	if err := copySkillDirectory(projectSkillDir, repoSkillDir); err != nil {
		return fmt.Errorf("复制技能目录失败: %w", err)
	}

	fmt.Println("✓ 更新本地仓库文件")

	fmt.Println("✓ 技能已归档到默认仓库")

	fmt.Println("\n✅ 反馈完成！")
	fmt.Printf("技能 '%s' 已保存到默认仓库: %s\n", skillID, defaultRepo.Name)
	if versionUpdated {
		fmt.Println("提示: 版本号已自动升级，可使用 'skill-hub status' 查看更新后的状态")
	}
	fmt.Println("使用 'skill-hub push' 同步到远程仓库")

	return nil
}

func getSkillVersionFromContent(content []byte) string {
	return skill.ExtractVersion(content)
}

func bumpPatchVersion(version string) string {
	version = strings.Trim(version, `" `)
	parts := strings.Split(version, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	patch, _ := strconv.Atoi(parts[2])
	parts[2] = strconv.Itoa(patch + 1)
	return strings.Join(parts[:3], ".")
}

func updateSkillMdVersion(skillMdPath, newVersion string) error {
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	inFrontmatter := false
	inMetadata := false
	updated := false

	for i, line := range lines {
		if line == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}

		if !inFrontmatter {
			continue
		}

		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "metadata:" || trimmedLine == "\"metadata\":" {
			inMetadata = true
			continue
		}

		if inMetadata {
			if strings.HasPrefix(trimmedLine, "version:") || strings.HasPrefix(trimmedLine, "\"version\":") {
				indent := ""
				for _, c := range line {
					if c == ' ' || c == '\t' {
						indent += string(c)
					} else {
						break
					}
				}
				lines[i] = fmt.Sprintf("%sversion: %s", indent, newVersion)
				updated = true
				break
			} else if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmedLine != "" {
				inMetadata = false
			}
		}
	}

	if !updated {
		inFrontmatter = false
		for i, line := range lines {
			if line == "---" {
				if inFrontmatter {
					break
				}
				inFrontmatter = true
				continue
			}

			if inFrontmatter {
				trimmedLine := strings.TrimSpace(line)
				if (strings.HasPrefix(trimmedLine, "version:") || strings.HasPrefix(trimmedLine, "\"version\":")) &&
					!strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
					lines[i] = fmt.Sprintf("version: %s", newVersion)
					updated = true
					break
				}
			}
		}
	}

	if !updated {
		return fmt.Errorf("未找到版本号字段")
	}

	return os.WriteFile(skillMdPath, []byte(strings.Join(lines, "\n")), 0644)
}

func compareSkillDirectories(projectDir, repoDir string, repoExists bool) ([]string, error) {
	var changes []string

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

			if _, err := os.Stat(projectPath); os.IsNotExist(err) {
				changes = append(changes, fmt.Sprintf("删除: %s", relPath))
			} else {
				projectContent, err1 := os.ReadFile(projectPath)
				repoContent, err2 := os.ReadFile(repoPath)

				if err1 != nil || err2 != nil {
					changes = append(changes, fmt.Sprintf("修改: %s (读取错误)", relPath))
				} else if string(projectContent) != string(repoContent) {
					changes = append(changes, fmt.Sprintf("修改: %s", relPath))
				}

				delete(projectFiles, relPath)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for relPath := range projectFiles {
		changes = append(changes, fmt.Sprintf("新增: %s", relPath))
	}

	return changes, nil
}

func copySkillDirectory(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

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

	for relPath := range srcFiles {
		srcPath := filepath.Join(srcDir, relPath)
		dstPath := filepath.Join(dstDir, relPath)

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return fmt.Errorf("创建目录失败 %s: %w", filepath.Dir(dstPath), err)
		}

		content, err := os.ReadFile(srcPath)
		if err != nil {
			return utils.ReadFileErr(err, srcPath)
		}

		info, err := os.Stat(srcPath)
		if err != nil {
			return fmt.Errorf("获取文件权限失败 %s: %w", srcPath, err)
		}

		if err := os.WriteFile(dstPath, content, info.Mode()); err != nil {
			return utils.WriteFileErr(err, dstPath)
		}

		delete(dstFiles, relPath)
	}

	for relPath := range dstFiles {
		dstPath := filepath.Join(dstDir, relPath)
		if err := os.Remove(dstPath); err != nil {
			return utils.DeleteFileErr(err, dstPath)
		}
	}

	return nil
}
