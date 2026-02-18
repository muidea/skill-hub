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
		// 简要模式显示 - 使用动态列宽
		fmt.Println("可用技能列表:")

		// 计算动态列宽
		widths := calculateColumnWidths(skillsMetadata)

		// 手动格式化标题行
		// 计算每个标题单元格需要的空格数
		idTitleSpaces := widths.idMin - displayWidth("ID")
		nameTitleSpaces := widths.nameMin - displayWidth("名称")
		versionTitleSpaces := widths.versionMin - displayWidth("版本")
		repoTitleSpaces := widths.repoMin - displayWidth("仓库")
		toolsTitleSpaces := widths.toolsMin - displayWidth("适用工具")

		// 确保空格数不为负
		if idTitleSpaces < 0 {
			idTitleSpaces = 0
		}
		if nameTitleSpaces < 0 {
			nameTitleSpaces = 0
		}
		if versionTitleSpaces < 0 {
			versionTitleSpaces = 0
		}
		if repoTitleSpaces < 0 {
			repoTitleSpaces = 0
		}
		if toolsTitleSpaces < 0 {
			toolsTitleSpaces = 0
		}

		fmt.Printf("%s%s %s%s %s%s %s%s %s%s\n",
			"ID", strings.Repeat(" ", idTitleSpaces),
			"名称", strings.Repeat(" ", nameTitleSpaces),
			"版本", strings.Repeat(" ", versionTitleSpaces),
			"仓库", strings.Repeat(" ", repoTitleSpaces),
			"适用工具", strings.Repeat(" ", toolsTitleSpaces))

		// 生成分隔线
		totalWidth := widths.idMin + widths.nameMin + widths.versionMin + widths.repoMin + widths.toolsMin + 4 // 4个空格
		separator := strings.Repeat("-", totalWidth)
		fmt.Println(separator)

		// 显示技能数据
		for _, skill := range skillsMetadata {
			// 获取工具字符串
			toolsStr := getToolsString(skill.Compatibility)

			// 格式化仓库名称
			repoName := formatRepoName(skill.Repository, widths.repoMin)

			// 手动格式化以确保对齐
			// 计算每个单元格需要的空格数
			idSpaces := widths.idMin - displayWidth(skill.ID)
			nameSpaces := widths.nameMin - displayWidth(skill.Name)
			versionSpaces := widths.versionMin - displayWidth(skill.Version)
			repoSpaces := widths.repoMin - displayWidth(repoName)
			toolsSpaces := widths.toolsMin - displayWidth(toolsStr)

			// 确保空格数不为负
			if idSpaces < 0 {
				idSpaces = 0
			}
			if nameSpaces < 0 {
				nameSpaces = 0
			}
			if versionSpaces < 0 {
				versionSpaces = 0
			}
			if repoSpaces < 0 {
				repoSpaces = 0
			}
			if toolsSpaces < 0 {
				toolsSpaces = 0
			}

			fmt.Printf("%s%s %s%s %s%s %s%s %s%s\n",
				skill.ID, strings.Repeat(" ", idSpaces),
				skill.Name, strings.Repeat(" ", nameSpaces),
				skill.Version, strings.Repeat(" ", versionSpaces),
				repoName, strings.Repeat(" ", repoSpaces),
				toolsStr, strings.Repeat(" ", toolsSpaces))
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

// columnWidths 定义列宽配置
type columnWidths struct {
	idMin      int
	idMax      int
	nameMin    int
	nameMax    int
	versionMin int
	versionMax int
	repoMin    int
	repoMax    int
	toolsMin   int
	toolsMax   int
}

// calculateColumnWidths 计算每列的最佳宽度
func calculateColumnWidths(skills []spec.SkillMetadata) columnWidths {
	widths := columnWidths{
		idMin:      2,  // "ID" 最小宽度
		idMax:      30, // ID最大宽度
		nameMin:    4,  // "名称" 最小宽度
		nameMax:    30, // 名称最大宽度
		versionMin: 4,  // "版本" 最小宽度
		versionMax: 10, // 版本最大宽度
		repoMin:    4,  // "仓库" 最小宽度
		repoMax:    20, // 仓库最大宽度
		toolsMin:   6,  // "适用工具" 最小宽度
		toolsMax:   30, // 工具最大宽度
	}

	// 计算每列的实际最大数据长度
	for _, skill := range skills {
		// 获取工具字符串
		toolsStr := getToolsString(skill.Compatibility)

		// 更新ID列宽度（使用显示宽度）
		updateWidth(&widths.idMin, displayWidth(skill.ID), widths.idMax)

		// 更新名称列宽度（使用显示宽度）
		updateWidth(&widths.nameMin, displayWidth(skill.Name), widths.nameMax)

		// 更新版本列宽度（使用显示宽度）
		updateWidth(&widths.versionMin, displayWidth(skill.Version), widths.versionMax)

		// 更新仓库列宽度（使用显示宽度）
		// 使用formatRepoName计算格式化后的仓库名称显示宽度
		repoName := formatRepoName(skill.Repository, widths.repoMax)
		updateWidth(&widths.repoMin, displayWidth(repoName), widths.repoMax)

		// 更新工具列宽度（使用显示宽度）
		updateWidth(&widths.toolsMin, displayWidth(toolsStr), widths.toolsMax)
	}

	// 确保每列至少有标题的字节长度，并为中文字符标题添加额外空间
	// 中文字符在fmt.Printf中需要更多空间来正确对齐
	if widths.idMin < len("ID") {
		widths.idMin = len("ID")
	}
	// 基于显示宽度设置列宽
	// 计算每列的最大显示宽度
	for _, skill := range skills {
		// 获取工具字符串
		toolsStr := getToolsString(skill.Compatibility)

		// 更新ID列显示宽度
		updateWidth(&widths.idMin, displayWidth(skill.ID), widths.idMax)

		// 更新名称列显示宽度
		updateWidth(&widths.nameMin, displayWidth(skill.Name), widths.nameMax)

		// 更新版本列显示宽度
		updateWidth(&widths.versionMin, displayWidth(skill.Version), widths.versionMax)

		// 更新仓库列显示宽度
		repoName := formatRepoName(skill.Repository, widths.repoMax)
		updateWidth(&widths.repoMin, displayWidth(repoName), widths.repoMax)

		// 更新工具列显示宽度
		updateWidth(&widths.toolsMin, displayWidth(toolsStr), widths.toolsMax)
	}

	// 确保每列至少有标题的显示宽度
	titleDisplays := map[string]int{
		"ID":   displayWidth("ID"),
		"名称":   displayWidth("名称"),
		"版本":   displayWidth("版本"),
		"仓库":   displayWidth("仓库"),
		"适用工具": displayWidth("适用工具"),
	}

	if widths.idMin < titleDisplays["ID"] {
		widths.idMin = titleDisplays["ID"]
	}
	if widths.nameMin < titleDisplays["名称"] {
		widths.nameMin = titleDisplays["名称"]
	}
	if widths.versionMin < titleDisplays["版本"] {
		widths.versionMin = titleDisplays["版本"]
	}
	if widths.repoMin < titleDisplays["仓库"] {
		widths.repoMin = titleDisplays["仓库"]
	}
	if widths.toolsMin < titleDisplays["适用工具"] {
		widths.toolsMin = titleDisplays["适用工具"]
	}

	// 为后三列添加额外显示宽度补偿
	// 经验值：每个中文字符需要额外1显示宽度补偿
	widths.versionMin += 2 // "版本"有2个中文字符
	widths.repoMin += 2    // "仓库"有2个中文字符
	widths.toolsMin += 4   // "适用工具"有4个中文字符

	// 确保不超过最大宽度限制
	if widths.idMin > widths.idMax {
		widths.idMin = widths.idMax
	}
	if widths.nameMin > widths.nameMax {
		widths.nameMin = widths.nameMax
	}
	if widths.versionMin > widths.versionMax {
		widths.versionMin = widths.versionMax
	}
	if widths.repoMin > widths.repoMax {
		widths.repoMin = widths.repoMax
	}
	if widths.toolsMin > widths.toolsMax {
		widths.toolsMin = widths.toolsMax
	}
	// 为中文标题添加额外空间补偿
	if widths.nameMin < len("名称")+4 { // "名称"需要额外空间
		widths.nameMin = len("名称") + 4
	}
	if widths.versionMin < len("版本")+4 { // "版本"需要额外空间
		widths.versionMin = len("版本") + 4
	}
	if widths.repoMin < len("仓库")+4 { // "仓库"需要额外空间
		widths.repoMin = len("仓库") + 4
	}
	if widths.toolsMin < len("适用工具")+8 { // "适用工具"需要更多额外空间
		widths.toolsMin = len("适用工具") + 8
	}

	return widths
}

// getToolsString 从兼容性字符串提取工具列表
func getToolsString(compatibility string) string {
	if compatibility == "" {
		return "all"
	}

	compatLower := strings.ToLower(compatibility)
	tools := []string{}

	// 检查各种兼容性格式
	if strings.Contains(compatLower, "cursor") {
		tools = append(tools, "cursor")
	}
	if strings.Contains(compatLower, "claude") {
		tools = append(tools, "claude_code")
	}
	if strings.Contains(compatLower, "shell") {
		tools = append(tools, "shell")
	}
	if strings.Contains(compatLower, "opencode") || strings.Contains(compatLower, "open_code") {
		tools = append(tools, "open_code")
	}

	if len(tools) == 0 {
		// 如果没有找到特定工具，但兼容性字段不为空，显示"all"
		return "all"
	}

	// 限制最多显示3个工具，避免过长
	if len(tools) > 3 {
		return tools[0] + "," + tools[1] + ",..."
	}

	return strings.Join(tools, ",")
}

// formatRepoName 格式化仓库名称显示，考虑显示宽度
func formatRepoName(repo string, maxDisplayWidth int) string {
	if repo == "" {
		return "local"
	}

	// 如果仓库名称的显示宽度不超过最大宽度，直接返回
	if displayWidth(repo) <= maxDisplayWidth {
		return repo
	}

	// 尝试截断路径部分，保留仓库名
	parts := strings.Split(repo, "/")
	if len(parts) > 1 {
		repoName := parts[len(parts)-1]
		if displayWidth(repoName) <= maxDisplayWidth {
			return repoName
		}
	}

	// 如果还是太长，截断并添加省略号
	// 我们需要找到合适的截断点，使得显示宽度不超过maxDisplayWidth-3（为"..."留空间）
	if maxDisplayWidth > 3 {
		targetWidth := maxDisplayWidth - 3
		truncated := ""
		currentWidth := 0

		// 逐个字符添加，直到达到目标宽度
		for _, r := range repo {
			charWidth := 1
			if r >= 0x4E00 && r <= 0x9FFF { // 基本CJK统一表意文字
				charWidth = 2
			} else if r >= 0x3400 && r <= 0x4DBF { // CJK统一表意文字扩展A
				charWidth = 2
			} else if r >= 0x20000 && r <= 0x2A6DF { // CJK统一表意文字扩展B
				charWidth = 2
			}

			if currentWidth+charWidth > targetWidth {
				break
			}
			truncated += string(r)
			currentWidth += charWidth
		}

		return truncated + "..."
	}

	// 如果最大宽度很小，直接截断
	truncated := ""
	currentWidth := 0
	for _, r := range repo {
		charWidth := 1
		if r >= 0x4E00 && r <= 0x9FFF {
			charWidth = 2
		} else if r >= 0x3400 && r <= 0x4DBF {
			charWidth = 2
		} else if r >= 0x20000 && r <= 0x2A6DF {
			charWidth = 2
		}

		if currentWidth+charWidth > maxDisplayWidth {
			break
		}
		truncated += string(r)
		currentWidth += charWidth
	}
	return truncated
}

// updateWidth 更新列宽，确保在最小和最大范围内
func updateWidth(currentWidth *int, newLength int, maxWidth int) {
	if newLength > *currentWidth && newLength <= maxWidth {
		*currentWidth = newLength
	}
}

// displayWidth 计算字符串在终端中的显示宽度（中文字符算2个宽度）
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF { // 基本CJK统一表意文字
			width += 2
		} else if r >= 0x3400 && r <= 0x4DBF { // CJK统一表意文字扩展A
			width += 2
		} else if r >= 0x20000 && r <= 0x2A6DF { // CJK统一表意文字扩展B
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// padRightDisplay 右填充字符串以达到目标显示宽度
func padRightDisplay(s string, targetByteWidth, currentDisplayWidth int) string {
	// 计算需要多少空格来达到目标显示宽度
	// 每个空格占用1显示宽度和1字节
	spacesNeeded := targetByteWidth - currentDisplayWidth
	if spacesNeeded <= 0 {
		// 如果已经达到或超过目标显示宽度，但可能需要确保字节长度
		if len(s) >= targetByteWidth {
			return s
		}
		// 添加空格以达到字节长度
		return s + strings.Repeat(" ", targetByteWidth-len(s))
	}

	// 添加空格以达到显示宽度
	return s + strings.Repeat(" ", spacesNeeded)
}

// addChineseCompensation 为中文字符串添加字节补偿
// 中文字符显示宽度小但字节长度大，需要额外字节空间来正确对齐
func addChineseCompensation(s string, displayWidth int) int {
	// 计算中文字符比例
	chineseCount := 0
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF { // 基本CJK统一表意文字
			chineseCount++
		} else if r >= 0x3400 && r <= 0x4DBF { // CJK统一表意文字扩展A
			chineseCount++
		} else if r >= 0x20000 && r <= 0x2A6DF { // CJK统一表意文字扩展B
			chineseCount++
		}
	}

	// 如果有中文字符，增加补偿
	if chineseCount > 0 {
		// 每个中文字符需要额外补偿
		// 经验值：每个中文字符需要额外1-2字节补偿
		return displayWidth + chineseCount*2
	}
	return displayWidth
}
