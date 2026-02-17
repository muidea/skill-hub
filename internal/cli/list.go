package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"skill-hub/internal/config"
	"skill-hub/internal/multirepo"
	"skill-hub/pkg/errors"
	"skill-hub/pkg/logging"
	"skill-hub/pkg/spec"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出可用技能",
	Long:  "显示本地技能仓库中的所有技能，支持按目标环境过滤。",
	RunE: func(cmd *cobra.Command, args []string) error {
		target, _ := cmd.Flags().GetString("target")
		verbose, _ := cmd.Flags().GetBool("verbose")
		return runList(target, verbose)
	},
}

func init() {
	listCmd.Flags().String("target", "", "按目标环境过滤技能列表")
	listCmd.Flags().Bool("verbose", false, "显示详细信息，包括技能描述、版本、兼容性等")
}

func runList(target string, verbose bool) error {
	// 检查init依赖（规范4.3：该命令依赖init命令）
	if err := CheckInitDependency(); err != nil {
		return err
	}

	// 创建多仓库管理器
	repoManager, err := multirepo.NewManager()
	if err != nil {
		return fmt.Errorf("创建多仓库管理器失败: %w", err)
	}

	// 获取所有技能
	skillsMetadata, err := repoManager.ListSkills("")
	if err != nil {
		return fmt.Errorf("获取技能列表失败: %w", err)
	}

	// 按目标环境过滤技能
	var filteredSkills []spec.SkillMetadata
	if target != "" {
		for _, skill := range skillsMetadata {
			compatLower := strings.ToLower(skill.Compatibility)
			targetLower := strings.ToLower(target)

			// 检查技能是否兼容指定的目标环境
			isCompatible := false
			if targetLower == "cursor" && strings.Contains(compatLower, "cursor") {
				isCompatible = true
			} else if (targetLower == "claude" || targetLower == "claude_code") &&
				(strings.Contains(compatLower, "claude") || strings.Contains(compatLower, "claude_code")) {
				isCompatible = true
			} else if (targetLower == "open_code" || targetLower == "opencode") &&
				(strings.Contains(compatLower, "open_code") || strings.Contains(compatLower, "opencode")) {
				isCompatible = true
			}

			if isCompatible {
				filteredSkills = append(filteredSkills, skill)
			}
		}
		skillsMetadata = filteredSkills
	}

	if len(skillsMetadata) == 0 {
		if target != "" {
			fmt.Printf("ℹ️  未找到兼容 %s 目标的技能\n", target)
		} else {
			fmt.Println("ℹ️  未找到任何技能")
		}
		return nil
	}

	if verbose {
		// 详细模式显示
		fmt.Println("可用技能列表 (详细模式):")
		fmt.Println(strings.Repeat("=", 60))
		for i, skill := range skillsMetadata {
			fmt.Printf("%d. ID: %s\n", i+1, skill.ID)
			fmt.Printf("   名称: %s\n", skill.Name)
			fmt.Printf("   版本: %s\n", skill.Version)
			fmt.Printf("   仓库: %s\n", skill.Repository)
			if skill.Description != "" {
				fmt.Printf("   描述: %s\n", skill.Description)
			}
			if skill.Compatibility != "" {
				fmt.Printf("   兼容性: %s\n", skill.Compatibility)
			}
			if len(skill.Tags) > 0 {
				fmt.Printf("   标签: %s\n", strings.Join(skill.Tags, ", "))
			}
			if skill.Author != "" && skill.Author != "unknown" {
				fmt.Printf("   作者: %s\n", skill.Author)
			}
			fmt.Println()
		}
	} else {
		// 简要模式显示
		fmt.Println("可用技能列表:")
		fmt.Println("ID          名称                版本      仓库          适用工具")
		fmt.Println("---------------------------------------------------------------")

		for _, skill := range skillsMetadata {
			tools := []string{}
			compatLower := strings.ToLower(skill.Compatibility)
			if strings.Contains(compatLower, "cursor") {
				tools = append(tools, "cursor")
			}
			if strings.Contains(compatLower, "claude code") || strings.Contains(compatLower, "claude_code") {
				tools = append(tools, "claude_code")
			}
			if strings.Contains(compatLower, "shell") {
				tools = append(tools, "shell")
			}
			if strings.Contains(compatLower, "opencode") || strings.Contains(compatLower, "open_code") {
				tools = append(tools, "open_code")
			}

			toolsStr := ""
			if len(tools) > 0 {
				toolsStr = tools[0]
				for i := 1; i < len(tools); i++ {
					toolsStr += "," + tools[i]
				}
			}

			// 截断仓库名称，如果太长
			repoName := skill.Repository
			if len(repoName) > 10 {
				repoName = repoName[:10] + "..."
			}

			fmt.Printf("%-12s %-20s %-10s %-12s %s\n",
				skill.ID,
				skill.Name,
				skill.Version,
				repoName,
				toolsStr)
		}
	}

	if target != "" {
		fmt.Printf("\n已过滤显示兼容 %s 目标的技能\n", target)
	}
	fmt.Println("\n使用 'skill-hub use <skill-id>' 在当前项目启用技能")
	return nil
}

// refreshRegistry 刷新技能索引，确保registry.json与skills目录同步
func refreshRegistry() error {
	// 获取日志记录器
	logger := logging.GetGlobalLogger().WithOperation("refreshRegistry")
	startTime := time.Now()

	// 获取repo目录
	repoPath, err := config.GetRepoPath()
	if err != nil {
		return errors.Wrap(err, "refreshRegistry: 获取repo路径失败")
	}

	// registry.json在根目录
	rootDir, err := config.GetRootDir()
	if err != nil {
		return errors.Wrap(err, "refreshRegistry: 获取根目录失败")
	}
	registryPath := filepath.Join(rootDir, "registry.json")
	skillsDir := filepath.Join(repoPath, "skills")

	// 检查skills目录是否存在
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		// 如果skills目录不存在，创建空的registry.json
		registryContent := `{
  "version": "1.0.0",
  "skills": []
}`
		if err := os.WriteFile(registryPath, []byte(registryContent), 0644); err != nil {
			return errors.Wrap(err, "refreshRegistry: 创建空registry.json失败")
		}
		logger.Info("创建空registry.json", "registry_path", registryPath)
		return nil
	}

	// 扫描skills目录下的所有子目录
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return errors.Wrap(err, "refreshRegistry: 读取skills目录失败")
	}

	logger.Debug("开始扫描skills目录", "skills_dir", skillsDir, "entry_count", len(entries))

	var skills []spec.SkillMetadata
	validCount := 0
	invalidCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		skillDir := filepath.Join(skillsDir, skillID)
		skillMdPath := filepath.Join(skillDir, "SKILL.md")

		// 检查是否存在SKILL.md文件
		if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
			logger.Debug("跳过无SKILL.md文件的技能目录", "skill_id", skillID)
			invalidCount++
			continue
		}

		skillMeta, err := parseSkillMetadataFromFile(skillMdPath, skillID)
		if err != nil {
			// 不输出错误，继续处理其他技能
			logger.Debug("解析技能元数据失败", "skill_id", skillID, "error", err.Error())
			invalidCount++
			continue
		}

		skills = append(skills, *skillMeta)
		validCount++
		logger.Debug("成功解析技能", "skill_id", skillID, "name", skillMeta.Name, "version", skillMeta.Version)
	}

	logger.Info("技能扫描完成", "total_entries", len(entries), "valid_skills", validCount, "invalid_entries", invalidCount)

	// 创建registry对象
	registry := spec.Registry{
		Version: "1.0.0",
		Skills:  skills,
	}

	// 转换为JSON
	registryJSON, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return errors.Wrap(err, "refreshRegistry: 序列化registry失败")
	}

	// 写入文件
	if err := os.WriteFile(registryPath, registryJSON, 0644); err != nil {
		return errors.Wrap(err, "refreshRegistry: 写入registry.json失败")
	}

	// 记录成功日志
	logger.Info("registry.json刷新成功",
		"registry_path", registryPath,
		"skill_count", len(skills),
		"duration_ms", time.Since(startTime).Milliseconds())

	return nil
}

// parseSkillMetadataFromFile 从SKILL.md文件解析技能元数据
func parseSkillMetadataFromFile(mdPath, skillID string) (*spec.SkillMetadata, error) {
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, errors.Wrap(err, "parseSkillMetadataFromFile: 读取SKILL.md失败")
	}

	// 解析frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return nil, errors.New("parseSkillMetadataFromFile: 无效的SKILL.md格式: 缺少frontmatter")
	}

	var frontmatterLines []string
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, lines[i])
	}

	frontmatter := strings.Join(frontmatterLines, "\n")

	// 解析YAML frontmatter
	var skillData map[string]interface{}
	if err := yaml.Unmarshal([]byte(frontmatter), &skillData); err != nil {
		return nil, errors.Wrap(err, "parseSkillMetadataFromFile: 解析frontmatter失败")
	}

	// 创建技能元数据对象
	skillMeta := &spec.SkillMetadata{
		ID: skillID,
	}

	// 设置名称
	if name, ok := skillData["name"].(string); ok {
		skillMeta.Name = name
	} else {
		skillMeta.Name = skillID
	}

	// 设置描述
	if desc, ok := skillData["description"].(string); ok {
		skillMeta.Description = desc
	}

	// 设置版本
	skillMeta.Version = "1.0.0"
	if version, ok := skillData["version"].(string); ok {
		skillMeta.Version = version
	}

	// 设置作者
	if author, ok := skillData["author"].(string); ok {
		skillMeta.Author = author
	} else if source, ok := skillData["source"].(string); ok {
		skillMeta.Author = source
	} else {
		skillMeta.Author = "unknown"
	}

	// 设置标签
	if tagsStr, ok := skillData["tags"].(string); ok {
		skillMeta.Tags = strings.Split(tagsStr, ",")
		for i, tag := range skillMeta.Tags {
			skillMeta.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// 设置兼容性
	if compatData, ok := skillData["compatibility"]; ok {
		switch v := compatData.(type) {
		case string:
			skillMeta.Compatibility = v
		case map[string]interface{}:
			// 向后兼容：将对象格式转换为字符串
			var compatList []string
			if cursorVal, ok := v["cursor"].(bool); ok && cursorVal {
				compatList = append(compatList, "Cursor")
			}
			if claudeVal, ok := v["claude_code"].(bool); ok && claudeVal {
				compatList = append(compatList, "Claude Code")
			}
			if openCodeVal, ok := v["open_code"].(bool); ok && openCodeVal {
				compatList = append(compatList, "OpenCode")
			}
			if shellVal, ok := v["shell"].(bool); ok && shellVal {
				compatList = append(compatList, "Shell")
			}
			if len(compatList) > 0 {
				skillMeta.Compatibility = "Designed for " + strings.Join(compatList, ", ") + " (or similar AI coding assistants)"
			}
		}
	}

	return skillMeta, nil
}
