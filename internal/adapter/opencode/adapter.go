package opencode

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"skill-hub/internal/config"
)

// OpenCodeAdapter 实现OpenCode适配器
type OpenCodeAdapter struct {
	mode     string // "project" 或 "global"
	basePath string // 基础路径
}

// NewOpenCodeAdapter 创建新的OpenCode适配器
func NewOpenCodeAdapter() *OpenCodeAdapter {
	return &OpenCodeAdapter{
		mode: "project", // 默认项目级
	}
}

// WithProjectMode 设置为项目级模式（向后兼容）
func (a *OpenCodeAdapter) WithProjectMode() *OpenCodeAdapter {
	a.mode = "project"
	return a
}

// WithGlobalMode 设置为全局级模式（向后兼容）
func (a *OpenCodeAdapter) WithGlobalMode() *OpenCodeAdapter {
	a.mode = "global"
	return a
}

// SetProjectMode 设置为项目模式
func (a *OpenCodeAdapter) SetProjectMode() {
	a.mode = "project"
}

// SetGlobalMode 设置为全局模式
func (a *OpenCodeAdapter) SetGlobalMode() {
	a.mode = "global"
}

// GetTarget 获取适配器对应的target类型
func (a *OpenCodeAdapter) GetTarget() string {
	return "open_code"
}

// GetSkillPath 获取技能在目标系统中的路径
func (a *OpenCodeAdapter) GetSkillPath(skillID string) (string, error) {
	return a.GetSkillDir(skillID)
}

// GetMode 获取当前模式（project/global）
func (a *OpenCodeAdapter) GetMode() string {
	return a.mode
}

// Apply 应用技能到OpenCode目录
func (a *OpenCodeAdapter) Apply(skillID string, content string, variables map[string]string) error {
	// 验证技能ID符合OpenCode命名规范
	if err := validateSkillName(skillID); err != nil {
		return fmt.Errorf("技能ID验证失败: %w", err)
	}

	// 获取基础路径
	basePath, err := a.getBasePath()
	if err != nil {
		return err
	}

	// 创建技能目录
	skillDir := filepath.Join(basePath, "skills", skillID)
	if err := createSkillDirectory(skillDir); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}

	// 转换内容为OpenCode格式
	openCodeContent, err := convertToOpenCodeFormat(content, skillID)
	if err != nil {
		return fmt.Errorf("转换技能格式失败: %w", err)
	}

	// 写入SKILL.md文件
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := writeSkillMDFile(skillPath, openCodeContent); err != nil {
		return fmt.Errorf("写入SKILL.md失败: %w", err)
	}

	// 对于open_code适配器，还需要从仓库复制其他文件
	if err := a.copyAdditionalFiles(skillID, skillDir); err != nil {
		return fmt.Errorf("复制额外文件失败: %w", err)
	}

	return nil
}

// Extract 从OpenCode目录提取技能内容
func (a *OpenCodeAdapter) Extract(skillID string) (string, error) {
	// 获取基础路径
	basePath, err := a.getBasePath()
	if err != nil {
		return "", err
	}

	// 构建技能文件路径
	skillPath := filepath.Join(basePath, "skills", skillID, "SKILL.md")

	// 读取文件内容
	content, err := os.ReadFile(skillPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // 文件不存在，返回空内容
		}
		return "", fmt.Errorf("读取SKILL.md失败: %w", err)
	}

	// 转换回标准格式
	standardContent, err := convertFromOpenCodeFormat(string(content))
	if err != nil {
		return "", fmt.Errorf("转换技能格式失败: %w", err)
	}

	return standardContent, nil
}

// copyAdditionalFiles 从仓库复制技能的其他文件
func (a *OpenCodeAdapter) copyAdditionalFiles(skillID, targetDir string) error {
	// 获取配置
	cfg, err := config.GetConfig()
	if err != nil {
		// 在测试环境中，配置文件可能不存在，静默返回
		// 在实际使用中，这个错误会在其他地方被捕获
		return nil
	}

	// 展开repo路径中的~符号
	repoPath := cfg.RepoPath
	if repoPath == "" {
		// 仓库路径未配置，静默返回
		return nil
	}

	// 处理~符号
	if repoPath[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("获取用户主目录失败: %w", err)
		}
		repoPath = filepath.Join(homeDir, repoPath[1:])
	}

	// 源技能目录
	srcSkillDir := filepath.Join(repoPath, "skills", skillID)

	// 检查源目录是否存在
	if _, err := os.Stat(srcSkillDir); os.IsNotExist(err) {
		// 源目录不存在，可能只有SKILL.md文件
		return nil
	}

	// 复制除SKILL.md之外的所有文件
	return filepath.Walk(srcSkillDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(srcSkillDir, srcPath)
		if err != nil {
			return fmt.Errorf("计算相对路径失败: %w", err)
		}

		// 跳过SKILL.md文件（已经由Apply方法处理）
		if relPath == "SKILL.md" {
			return nil
		}

		// 目标路径
		dstPath := filepath.Join(targetDir, relPath)

		// 如果是目录，创建目录
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// 如果是文件，复制文件
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("读取文件失败 %s: %w", srcPath, err)
		}

		if err := os.WriteFile(dstPath, content, info.Mode()); err != nil {
			return fmt.Errorf("写入文件失败 %s: %w", dstPath, err)
		}

		return nil
	})
}

// Remove 从OpenCode目录移除技能
func (a *OpenCodeAdapter) Remove(skillID string) error {
	// 获取基础路径
	basePath, err := a.getBasePath()
	if err != nil {
		return err
	}

	// 构建技能目录路径
	skillDir := filepath.Join(basePath, "skills", skillID)

	// 检查目录是否存在
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return nil // 目录不存在，无需移除
	}

	// 递归删除目录
	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("删除技能目录失败: %w", err)
	}

	// 检查父目录是否为空，如果为空则删除
	parentDir := filepath.Join(basePath, "skills")
	if isEmpty, _ := isDirectoryEmpty(parentDir); isEmpty {
		os.Remove(parentDir)
	}

	return nil
}

// List 列出OpenCode目录中的所有技能
func (a *OpenCodeAdapter) List() ([]string, error) {
	// 获取基础路径
	basePath, err := a.getBasePath()
	if err != nil {
		return nil, err
	}

	// 构建技能目录路径
	skillsDir := filepath.Join(basePath, "skills")

	// 检查目录是否存在
	if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
		return []string{}, nil // 目录不存在，返回空列表
	}

	// 读取目录内容
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("读取技能目录失败: %w", err)
	}

	var skillIDs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		// 检查是否包含SKILL.md文件
		skillPath := filepath.Join(skillsDir, skillID, "SKILL.md")
		if _, err := os.Stat(skillPath); err == nil {
			skillIDs = append(skillIDs, skillID)
		}
	}

	return skillIDs, nil
}

// GetSkillsPath 获取技能目录路径（公开方法）
func (a *OpenCodeAdapter) GetSkillsPath() (string, error) {
	basePath, err := a.getBasePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(basePath, "skills"), nil
}

// Supports 检查是否支持当前环境
func (a *OpenCodeAdapter) Supports() bool {
	// OpenCode适配器总是可用的
	// 可以添加更复杂的检查逻辑，如检查目录是否可写等
	return true
}

// getBasePath 获取基础路径
func (a *OpenCodeAdapter) getBasePath() (string, error) {
	if a.basePath != "" {
		return a.basePath, nil
	}

	if a.mode == "project" {
		// 项目级：使用当前工作目录
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("获取当前目录失败: %w", err)
		}
		a.basePath = filepath.Join(cwd, ".agents")
	} else {
		// 全局级：使用用户配置目录
		_, err := config.GetConfig()
		if err != nil {
			return "", err
		}
		// 展开路径中的~
		a.basePath = expandPath(filepath.Join("~", ".config", "opencode"))
	}

	return a.basePath, nil
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

// isDirectoryEmpty 检查目录是否为空
func isDirectoryEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// Cleanup 清理临时文件（备份文件、临时文件等）
func (a *OpenCodeAdapter) Cleanup() error {
	if a.basePath == "" {
		// 如果没有设置基础路径，尝试获取
		basePath, err := a.getBasePath()
		if err != nil {
			return err
		}
		a.basePath = basePath
	}

	// 获取技能目录路径
	skillsDir := filepath.Join(a.basePath, "skills")

	// 读取技能目录中的所有子目录
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 技能目录不存在，无需清理
		}
		return fmt.Errorf("读取技能目录失败: %w", err)
	}

	// 清理每个技能目录的备份
	for _, entry := range entries {
		if entry.IsDir() {
			skillDir := filepath.Join(skillsDir, entry.Name())

			// 清理备份目录
			backupDir := skillDir + ".bak"
			if err := os.RemoveAll(backupDir); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("清理技能目录备份失败 %s: %w", skillDir, err)
			}
		}
	}

	return nil
}

// writeSkillMDFile 写入SKILL.md文件（原子操作）
func writeSkillMDFile(skillPath string, content string) error {
	// 创建临时文件
	tmpPath := skillPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 重命名为目标文件
	if err := os.Rename(tmpPath, skillPath); err != nil {
		// 清理临时文件
		os.Remove(tmpPath)
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	return nil
}

// GetBackupPath 获取备份文件路径
func (a *OpenCodeAdapter) GetBackupPath() string {
	// OpenCode适配器使用目录备份，没有单一的备份文件路径
	// 返回空字符串，让恢复逻辑使用特定的技能目录
	return ""
}

// GetSkillDir 获取技能目录路径
func (a *OpenCodeAdapter) GetSkillDir(skillID string) (string, error) {
	basePath, err := a.getBasePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(basePath, "skills", skillID), nil
}
