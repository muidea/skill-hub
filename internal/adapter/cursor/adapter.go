package cursor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"skill-hub/internal/adapter"
	"skill-hub/internal/config"
)

// CursorAdapter 实现Cursor规则的适配器
type CursorAdapter struct {
	filePath string
	mode     string // "global" 或 "project"
}

// NewCursorAdapter 创建新的Cursor适配器
func NewCursorAdapter() *CursorAdapter {
	return &CursorAdapter{
		mode: "project", // 默认项目模式
	}
}

// WithProjectMode 设置为项目模式
func (a *CursorAdapter) WithProjectMode() *CursorAdapter {
	a.mode = "project"
	return a
}

// WithGlobalMode 设置为全局模式
func (a *CursorAdapter) WithGlobalMode() *CursorAdapter {
	a.mode = "global"
	return a
}

// markerPattern 匹配技能标记块的正则表达式
var markerPattern = regexp.MustCompile(`(?s)# === SKILL-HUB BEGIN: (?P<id>.*?) ===\n(?P<content>.*?)\n# === SKILL-HUB END: (?P<id2>.*?) ===`)

// Apply 应用技能到.cursorrules文件
func (a *CursorAdapter) Apply(skillID string, content string, variables map[string]string) error {
	// 获取配置文件路径
	filePath, err := a.getFilePath()
	if err != nil {
		return err
	}
	a.filePath = filePath

	fmt.Printf("应用技能到Cursor配置文件: %s\n", filePath)

	// 渲染模板内容
	renderedContent, err := a.renderTemplate(content, variables)
	if err != nil {
		return fmt.Errorf("渲染模板失败: %w", err)
	}

	// 创建标记块
	markerBlock := a.createMarkerBlock(skillID, renderedContent)

	// 读取现有文件内容
	existingContent, err := a.readFile()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// 替换或添加标记块
	newContent := a.replaceOrAddMarker(existingContent, skillID, markerBlock)

	// 写入文件
	return a.writeFile(newContent)
}

// Extract 从.cursorrules文件提取技能内容
func (a *CursorAdapter) Extract(skillID string) (string, error) {
	filePath, err := a.getFilePath()
	if err != nil {
		return "", err
	}
	a.filePath = filePath

	content, err := a.readFile()
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("文件不存在: %s", filePath)
		}
		return "", err
	}

	// 查找标记块
	matches := markerPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 4 && match[1] == skillID && match[3] == skillID {
			// 提取标记块内的内容
			return a.extractMarkedContent(content, skillID)
		}
	}

	return "", fmt.Errorf("未找到技能 '%s' 的标记块", skillID)
}

// Remove 从.cursorrules文件移除技能
func (a *CursorAdapter) Remove(skillID string) error {
	filePath, err := a.getFilePath()
	if err != nil {
		return err
	}
	a.filePath = filePath

	content, err := a.readFile()
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，无需移除
		}
		return err
	}

	// 移除指定技能的标记块
	pattern := regexp.MustCompile(fmt.Sprintf(`(?s)# === SKILL-HUB BEGIN: %s ===\n.*?\n# === SKILL-HUB END: %s ===\n?`, regexp.QuoteMeta(skillID), regexp.QuoteMeta(skillID)))
	newContent := pattern.ReplaceAllString(content, "")

	// 如果内容为空，删除文件
	newContent = strings.TrimSpace(newContent)
	if newContent == "" {
		return os.Remove(filePath)
	}

	return a.writeFile(newContent)
}

// List 列出.cursorrules文件中的所有技能
func (a *CursorAdapter) List() ([]string, error) {
	filePath, err := a.getFilePath()
	if err != nil {
		return nil, err
	}
	a.filePath = filePath

	content, err := a.readFile()
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var skillIDs []string
	matches := markerPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 2 && match[1] == match[3] { // 确保BEGIN和END的ID匹配
			skillIDs = append(skillIDs, match[1])
		}
	}

	return skillIDs, nil
}

// Supports 检查是否支持当前环境
func (a *CursorAdapter) Supports() bool {
	// Cursor适配器总是可用
	return true
}

// renderTemplate 渲染模板内容
func (a *CursorAdapter) renderTemplate(content string, variables map[string]string) (string, error) {
	// 简单替换变量
	result := content
	for key, value := range variables {
		placeholder := "{{." + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result, nil
}

// createMarkerBlock 创建标记块
func (a *CursorAdapter) createMarkerBlock(skillID string, content string) string {
	return fmt.Sprintf("# === SKILL-HUB BEGIN: %s ===\n%s\n# === SKILL-HUB END: %s ===\n", skillID, content, skillID)
}

// readFile 读取文件内容
func (a *CursorAdapter) readFile() (string, error) {
	data, err := os.ReadFile(a.filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeFile 写入文件内容（原子操作）
func (a *CursorAdapter) writeFile(content string) error {
	// 确保目录存在
	dir := filepath.Dir(a.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 创建备份（如果文件存在）
	if _, err := os.Stat(a.filePath); err == nil {
		backupPath := a.filePath + ".bak"
		if err := os.Rename(a.filePath, backupPath); err != nil {
			return fmt.Errorf("创建备份失败: %w", err)
		}
	}

	// 写入临时文件
	tmpPath := a.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		// 尝试恢复备份
		if backupPath := a.filePath + ".bak"; fileExists(backupPath) {
			os.Rename(backupPath, a.filePath)
		}
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 重命名为目标文件
	if err := os.Rename(tmpPath, a.filePath); err != nil {
		// 尝试恢复备份
		if backupPath := a.filePath + ".bak"; fileExists(backupPath) {
			os.Rename(backupPath, a.filePath)
		}
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	// 清理备份文件
	if backupPath := a.filePath + ".bak"; fileExists(backupPath) {
		os.Remove(backupPath)
	}

	return nil
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// extractMarkedContent 从标记块中提取内容
func (a *CursorAdapter) extractMarkedContent(content, skillID string) (string, error) {
	beginMarker := fmt.Sprintf("# === SKILL-HUB BEGIN: %s ===", skillID)
	endMarker := fmt.Sprintf("# === SKILL-HUB END: %s ===", skillID)

	beginIdx := strings.Index(content, beginMarker)
	if beginIdx == -1 {
		return "", fmt.Errorf("未找到开始标记")
	}

	endIdx := strings.Index(content, endMarker)
	if endIdx == -1 {
		return "", fmt.Errorf("未找到结束标记")
	}

	// 提取标记块内的内容
	start := beginIdx + len(beginMarker)
	extracted := strings.TrimSpace(content[start:endIdx])

	return extracted, nil
}

// replaceOrAddMarker 替换或添加标记块
func (a *CursorAdapter) replaceOrAddMarker(existingContent, skillID, markerBlock string) string {
	// 尝试替换现有标记块
	pattern := regexp.MustCompile(fmt.Sprintf(`(?s)# === SKILL-HUB BEGIN: %s ===\n.*?\n# === SKILL-HUB END: %s ===`, regexp.QuoteMeta(skillID), regexp.QuoteMeta(skillID)))

	if pattern.MatchString(existingContent) {
		return pattern.ReplaceAllString(existingContent, markerBlock)
	}

	// 没有现有标记块，添加到文件末尾
	existingContent = strings.TrimSpace(existingContent)
	if existingContent == "" {
		return markerBlock
	}

	return existingContent + "\n\n" + markerBlock
}

// GetFilePath 获取适配器管理的文件路径（公开方法）
func (a *CursorAdapter) GetFilePath() (string, error) {
	return a.getFilePath()
}

// getFilePath 获取配置文件路径
func (a *CursorAdapter) getFilePath() (string, error) {
	if a.mode == "project" {
		// 项目级配置
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("获取当前目录失败: %w", err)
		}
		return filepath.Join(cwd, ".cursorrules"), nil
	}

	// 全局配置
	cfg, err := config.GetConfig()
	if err != nil {
		return "", err
	}

	// 展开路径中的~
	return expandPath(cfg.CursorConfigPath), nil
}

// expandPath 展开路径中的~为用户主目录
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// Cleanup 清理临时文件（备份文件、临时文件等）
func (a *CursorAdapter) Cleanup() error {
	if a.filePath == "" {
		// 如果没有设置文件路径，尝试获取
		filePath, err := a.getFilePath()
		if err != nil {
			return err
		}
		a.filePath = filePath
	}

	// 使用统一的清理函数
	return adapter.CleanupTempFiles(a.filePath)
}

// GetBackupPath 获取备份文件路径
func (a *CursorAdapter) GetBackupPath() string {
	if a.filePath == "" {
		// 如果没有设置文件路径，返回空
		return ""
	}
	return a.filePath + ".bak"
}
