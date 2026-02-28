package cli

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"skill-hub/internal/config"
	"skill-hub/internal/state"
	"skill-hub/pkg/spec"
	"skill-hub/pkg/utils"
)

var statusCmd = &cobra.Command{
	Use:   "status [id]",
	Short: "检查技能状态",
	Long: `对比项目本地工作区文件与技能仓库源文件的差异，显示技能状态：
- Synced: 本地与仓库一致
- Modified: 本地有未反馈的修改
- Outdated: 仓库版本领先于本地
- Missing: 技能已启用但本地文件缺失`,
	RunE: func(cmd *cobra.Command, args []string) error {
		skillID := ""
		if len(args) > 0 {
			skillID = args[0]
		}
		verbose, _ := cmd.Flags().GetBool("verbose")
		return runStatus(skillID, verbose)
	},
}

func init() {
	statusCmd.Flags().Bool("verbose", false, "显示详细差异信息")
}

func runStatus(skillID string, verbose bool) error {
	if err := CheckInitDependency(); err != nil {
		return err
	}

	fmt.Println("检查技能状态...")

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
		return err
	}

	projectState, err := stateManager.LoadProjectState(cwd)
	if err != nil {
		return err
	}

	skills := projectState.Skills
	if len(skills) == 0 {
		fmt.Println("ℹ️  当前项目未启用任何技能")
		return nil
	}

	targetSkillID := skillID
	if skillID != "" {
		if _, exists := skills[skillID]; !exists {
			return fmt.Errorf("技能 %s 未在当前项目中启用", skillID)
		}
		singleSkill := map[string]spec.SkillVars{
			skillID: skills[skillID],
		}
		skills = singleSkill
	}

	fmt.Printf("项目路径: %s\n", cwd)
	fmt.Printf("启用技能数: %d\n", len(skills))
	if skillID != "" {
		fmt.Printf("检查特定技能: %s\n", skillID)
	}
	fmt.Println()

	fmt.Println("检查项目本地工作区文件...")

	results := make(map[string]string)

	for currentSkillID, skillVars := range skills {
		agentsSkillDir := filepath.Join(cwd, ".agents", "skills", currentSkillID)
		skillMdPath := filepath.Join(agentsSkillDir, "SKILL.md")

		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			results[currentSkillID] = spec.SkillStatusMissing
			updateSkillStatus(cwd, currentSkillID, spec.SkillStatusMissing, skillVars.Version)
			continue
		}

		localVersion, localHash, err := getLocalSkillInfo(skillMdPath)
		if err != nil {
			fmt.Printf("⚠️  获取技能 %s 信息失败，标记为Modified: %v\n", currentSkillID, err)
			results[currentSkillID] = spec.SkillStatusModified
			updateSkillStatus(cwd, currentSkillID, spec.SkillStatusModified, "unknown")
			continue
		}

		repoVersion, repoHash, err := getRepoSkillInfo(currentSkillID)
		if err != nil {
			results[currentSkillID] = spec.SkillStatusModified
			if verbose {
				fmt.Printf("  ℹ️  技能 %s 在仓库中不存在，标记为 Modified\n", currentSkillID)
			}
			updateSkillStatus(cwd, currentSkillID, spec.SkillStatusModified, localVersion)
			continue
		}

		status := determineSkillStatus(localVersion, localHash, repoVersion, repoHash)
		results[currentSkillID] = status
		updateSkillStatus(cwd, currentSkillID, status, localVersion)
	}

	fmt.Println("\n=== 技能状态 ===")

	maxIDLength := 2
	for currentSkillID := range results {
		if len(currentSkillID) > maxIDLength {
			maxIDLength = len(currentSkillID)
		}
	}

	fmt.Printf("%-*s 状态\n", maxIDLength, "ID")
	fmt.Println(strings.Repeat("-", maxIDLength+4))

	for currentSkillID, status := range results {
		statusSymbol := "❓"
		switch status {
		case spec.SkillStatusSynced:
			statusSymbol = "✅"
		case spec.SkillStatusModified:
			statusSymbol = "⚠️"
		case spec.SkillStatusOutdated:
			statusSymbol = "🔄"
		case spec.SkillStatusMissing:
			statusSymbol = "❌"
		}
		fmt.Printf("%-*s %s %s\n", maxIDLength, currentSkillID, statusSymbol, status)
	}

	if verbose {
		fmt.Println("\n=== 详细差异信息 ===")
		for currentSkillID := range results {
			showSkillDiff(cwd, currentSkillID)
		}
	}

	fmt.Println("\n说明:")
	fmt.Println("✅ Synced: 本地与仓库一致")
	fmt.Println("⚠️  Modified: 本地有未反馈的修改")
	fmt.Println("🔄 Outdated: 仓库版本领先于本地")
	fmt.Println("❌ Missing: 技能已启用但本地文件缺失")

	if targetSkillID != "" {
		showSkillDetails(cwd, targetSkillID, results[targetSkillID])
	} else if !verbose {
		fmt.Println("\n使用 'skill-hub status <id>' 检查特定技能状态")
		fmt.Println("使用 'skill-hub status --verbose' 显示详细差异")
	}

	return nil
}

func showSkillDetails(cwd, skillID, status string) {
	fmt.Println("\n=== 技能详情 ===")

	agentsSkillDir := filepath.Join(cwd, ".agents", "skills", skillID)
	localSkillMdPath := filepath.Join(agentsSkillDir, "SKILL.md")

	statusSymbol := "❓"
	switch status {
	case spec.SkillStatusSynced:
		statusSymbol = "✅"
	case spec.SkillStatusModified:
		statusSymbol = "⚠️"
	case spec.SkillStatusOutdated:
		statusSymbol = "🔄"
	case spec.SkillStatusMissing:
		statusSymbol = "❌"
	}

	fmt.Printf("ID:         %s\n", skillID)
	fmt.Printf("状态:       %s %s\n", statusSymbol, status)

	localVersion := "N/A"
	if status != spec.SkillStatusMissing {
		if v, _, err := getLocalSkillInfo(localSkillMdPath); err == nil {
			localVersion = v
		}
	}
	fmt.Printf("本地版本:   %s\n", localVersion)

	repoVersion := "N/A"
	if v, _, err := getRepoSkillInfo(skillID); err == nil {
		repoVersion = v
	}
	fmt.Printf("仓库版本:   %s\n", repoVersion)

	fmt.Printf("本地路径:   %s\n", localSkillMdPath)

	rootDir, _ := config.GetRootDir()
	cfg, _ := config.GetConfig()
	repoName := "main"
	if cfg != nil && cfg.MultiRepo != nil && cfg.MultiRepo.DefaultRepo != "" {
		repoName = cfg.MultiRepo.DefaultRepo
	}
	repoSkillPath := filepath.Join(rootDir, "repositories", repoName, "skills", skillID, "SKILL.md")
	fmt.Printf("仓库路径:   %s\n", repoSkillPath)
}

func showSkillDiff(cwd, skillID string) {
	agentsSkillDir := filepath.Join(cwd, ".agents", "skills", skillID)
	localSkillMdPath := filepath.Join(agentsSkillDir, "SKILL.md")
	repoSkillMdPath := ""

	rootDir, err := config.GetRootDir()
	if err == nil {
		cfg, cfgErr := config.GetConfig()
		if cfgErr == nil && cfg.MultiRepo != nil {
			repoName := cfg.MultiRepo.DefaultRepo
			if repoName == "" {
				repoName = "main"
			}
			repoSkillMdPath = filepath.Join(rootDir, "repositories", repoName, "skills", skillID, "SKILL.md")
		}
	}

	localContent, localErr := os.ReadFile(localSkillMdPath)
	repoContent, repoErr := os.ReadFile(repoSkillMdPath)

	fmt.Printf("\n--- %s ---\n", skillID)

	if localErr != nil && repoErr != nil {
		fmt.Println("⚠️  无法读取本地和仓库文件")
		return
	}

	if localErr != nil {
		fmt.Println("⚠️  无法读取本地文件")
		fmt.Printf("仓库文件: %s\n", repoSkillMdPath)
		return
	}

	if repoErr != nil {
		fmt.Println("⚠️  无法读取仓库文件（技能可能不在仓库中）")
		fmt.Printf("本地文件: %s\n", localSkillMdPath)
		return
	}

	localLines := strings.Split(string(localContent), "\n")
	repoLines := strings.Split(string(repoContent), "\n")

	if string(localContent) == string(repoContent) {
		fmt.Println("✅ 本地与仓库内容完全一致")
		return
	}

	fmt.Printf("差异统计: 本地 %d 行, 仓库 %d 行\n", len(localLines), len(repoLines))
	fmt.Println("\n差异预览 (最多显示20行):")

	diffLines := computeSimpleDiff(localLines, repoLines)
	displayCount := 0
	for _, line := range diffLines {
		if displayCount >= 20 {
			fmt.Printf("... 还有 %d 行差异未显示\n", len(diffLines)-20)
			break
		}
		fmt.Println(line)
		displayCount++
	}
}

func computeSimpleDiff(local, repo []string) []string {
	var result []string

	localSet := make(map[string]bool)
	for _, line := range local {
		localSet[line] = true
	}

	repoSet := make(map[string]bool)
	for _, line := range repo {
		repoSet[line] = true
	}

	for _, line := range repo {
		if !localSet[line] && strings.TrimSpace(line) != "" {
			result = append(result, fmt.Sprintf("-%s", line))
		}
	}

	for _, line := range local {
		if !repoSet[line] && strings.TrimSpace(line) != "" {
			result = append(result, fmt.Sprintf("+%s", line))
		}
	}

	if len(result) == 0 {
		for i := 0; i < len(local) && i < len(repo); i++ {
			if local[i] != repo[i] {
				result = append(result, fmt.Sprintf("-%s", repo[i]))
				result = append(result, fmt.Sprintf("+%s", local[i]))
			}
		}
	}

	return result
}

func getLocalSkillInfo(skillMdPath string) (string, string, error) {
	content, err := os.ReadFile(skillMdPath)
	if err != nil {
		return "", "", utils.ReadFileErr(err, skillMdPath)
	}

	hash := md5.Sum(content)
	hashStr := fmt.Sprintf("%x", hash)

	version := "1.0.0"
	lines := strings.Split(string(content), "\n")
	if len(lines) > 2 && lines[0] == "---" {
		var frontmatterLines []string
		for i := 1; i < len(lines); i++ {
			if lines[i] == "---" {
				break
			}
			frontmatterLines = append(frontmatterLines, lines[i])
		}

		frontmatter := strings.Join(frontmatterLines, "\n")
		var skillData map[string]interface{}
		if err := yaml.Unmarshal([]byte(frontmatter), &skillData); err == nil {
			if metadata, ok := skillData["metadata"].(map[string]interface{}); ok {
				if v, ok := metadata["version"].(string); ok {
					version = v
				}
			} else if v, ok := skillData["version"].(string); ok {
				version = v
			}
		}
	}

	return version, hashStr, nil
}

func getRepoSkillInfo(skillID string) (string, string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", "", fmt.Errorf("获取配置失败: %w", err)
	}

	var repoPath string
	if cfg.MultiRepo != nil {
		rootDir, err := config.GetRootDir()
		if err != nil {
			return "", "", fmt.Errorf("获取根目录失败: %w", err)
		}
		repoPath = filepath.Join(rootDir, "repositories", cfg.MultiRepo.DefaultRepo)
	} else {
		return "", "", fmt.Errorf("多仓库配置未初始化")
	}

	repoSkillPath := filepath.Join(repoPath, "skills", skillID, "SKILL.md")
	if _, err := os.Stat(repoSkillPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("技能在仓库中不存在")
	}

	return getLocalSkillInfo(repoSkillPath)
}

func determineSkillStatus(localVersion, localHash, repoVersion, repoHash string) string {
	if localHash != repoHash {
		if compareVersions(localVersion, repoVersion) < 0 {
			return spec.SkillStatusOutdated
		} else {
			return spec.SkillStatusModified
		}
	}

	if compareVersions(localVersion, repoVersion) < 0 {
		return spec.SkillStatusOutdated
	}

	return spec.SkillStatusSynced
}

func compareVersions(v1, v2 string) int {
	v1 = strings.Trim(v1, `"`)
	v2 = strings.Trim(v2, `" `)

	if v1 == v2 {
		return 0
	}

	v1Parts := strings.Split(v1, ".")
	v2Parts := strings.Split(v2, ".")

	for i := 0; i < len(v1Parts) && i < len(v2Parts); i++ {
		num1 := 0
		num2 := 0
		fmt.Sscanf(v1Parts[i], "%d", &num1)
		fmt.Sscanf(v2Parts[i], "%d", &num2)

		if num1 > num2 {
			return 1
		} else if num1 < num2 {
			return -1
		}
	}

	if len(v1Parts) > len(v2Parts) {
		return 1
	} else if len(v1Parts) < len(v2Parts) {
		return -1
	}

	if v1 > v2 {
		return 1
	}
	return -1
}

func updateSkillStatus(projectPath, skillID, status, version string) error {
	stateManager, err := state.NewStateManager()
	if err != nil {
		return fmt.Errorf("创建状态管理器失败: %w", err)
	}

	projectState, err := stateManager.LoadProjectState(projectPath)
	if err != nil {
		return fmt.Errorf("加载项目状态失败: %w", err)
	}

	if skillVars, exists := projectState.Skills[skillID]; exists {
		skillVars.Status = status
		skillVars.Version = version
		projectState.Skills[skillID] = skillVars
	} else {
		projectState.Skills[skillID] = spec.SkillVars{
			SkillID: skillID,
			Version: version,
			Status:  status,
			Variables: map[string]string{
				"target": "open_code",
			},
		}
	}

	if err := stateManager.SaveProjectState(projectState); err != nil {
		return fmt.Errorf("保存项目状态失败: %w", err)
	}

	return nil
}
