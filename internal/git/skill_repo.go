package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"skill-hub/internal/adapter"
	"skill-hub/internal/config"
	"skill-hub/pkg/spec"
)

// SkillRepository 管理技能Git仓库
type SkillRepository struct {
	repo *Repository
}

// NewSkillRepository 创建技能仓库管理器
func NewSkillRepository() (*SkillRepository, error) {
	repo, err := NewSkillsRepository()
	if err != nil {
		return nil, err
	}
	return &SkillRepository{repo: repo}, nil
}

// Sync 同步技能仓库（拉取最新更改）
func (sr *SkillRepository) Sync() error {
	fmt.Println("正在同步技能仓库...")

	if !sr.repo.IsInitialized() {
		return fmt.Errorf("技能仓库未初始化，请先设置远程仓库URL")
	}

	// 获取当前状态
	status, err := sr.repo.GetStatus()
	if err != nil {
		return fmt.Errorf("获取仓库状态失败: %w", err)
	}

	// 检查是否有未提交的更改
	if strings.Contains(status, " M ") || strings.Contains(status, "?? ") {
		fmt.Println("⚠️  检测到未提交的更改，建议先提交或暂存")
		fmt.Println("   使用 'skill-hub git commit' 提交更改")
		fmt.Println("   或使用 'skill-hub git stash' 暂存更改")
	}

	// 拉取最新更改
	fmt.Println("从远程仓库拉取最新更改...")
	if err := sr.repo.Pull(); err != nil {
		return fmt.Errorf("拉取失败: %w", err)
	}

	fmt.Println("✅ 技能仓库同步完成")
	return nil
}

// PushChanges 推送本地更改到远程仓库
func (sr *SkillRepository) PushChanges(message string) error {
	if !sr.repo.IsInitialized() {
		return fmt.Errorf("技能仓库未初始化，请先设置远程仓库URL")
	}

	// 检查是否有更改
	status, err := sr.repo.GetStatus()
	if err != nil {
		return fmt.Errorf("获取仓库状态失败: %w", err)
	}

	if !strings.Contains(status, " M ") && !strings.Contains(status, "?? ") && !strings.Contains(status, " D ") {
		return fmt.Errorf("没有要推送的更改")
	}

	// 提交更改
	if message == "" {
		message = fmt.Sprintf("更新技能: %s", time.Now().Format("2006-01-02 15:04:05"))
	}

	fmt.Println("提交更改...")
	if err := sr.repo.Commit(message); err != nil {
		return fmt.Errorf("提交失败: %w", err)
	}

	// 推送到远程
	fmt.Println("推送到远程仓库...")
	if err := sr.repo.Push(); err != nil {
		return fmt.Errorf("推送失败: %w", err)
	}

	fmt.Println("✅ 更改已推送到远程仓库")
	return nil
}

// CloneRemote 克隆远程技能仓库
func (sr *SkillRepository) CloneRemote(url string) error {
	fmt.Printf("正在克隆远程技能仓库: %s\n", url)

	// 获取技能目录路径
	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return err
	}

	// 如果目录已存在，备份
	if _, err := os.Stat(skillsDir); err == nil {
		backupDir := skillsDir + ".bak." + time.Now().Format("20060102-150405")
		fmt.Printf("备份现有技能目录到: %s\n", backupDir)
		if err := os.Rename(skillsDir, backupDir); err != nil {
			return fmt.Errorf("备份失败: %w", err)
		}
	}

	// 克隆仓库
	if err := sr.repo.Clone(url); err != nil {
		return fmt.Errorf("克隆失败: %w", err)
	}

	// 更新配置中的远程URL（多仓库模式）
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	// 保存配置 - 更新默认仓库的URL
	if cfg.MultiRepo != nil {
		if defaultRepo, exists := cfg.MultiRepo.Repositories[cfg.MultiRepo.DefaultRepo]; exists {
			defaultRepo.URL = url
			cfg.MultiRepo.Repositories[cfg.MultiRepo.DefaultRepo] = defaultRepo

			// 保存更新后的配置
			if err := config.SaveConfig(cfg); err != nil {
				return fmt.Errorf("保存配置失败: %w", err)
			}
		}
	}

	fmt.Println("✅ 远程技能仓库克隆完成")

	// 清理可能创建的备份目录
	if err := adapter.CleanupTimestampedBackupDirs(skillsDir); err != nil {
		fmt.Printf("⚠️  清理备份目录失败: %v\n", err)
	}

	return nil
}

// GetStatus 获取技能仓库状态
func (sr *SkillRepository) GetStatus() (string, error) {
	// 检查仓库目录是否存在且有.git目录
	repoPath := sr.repo.GetPath()
	gitDir := filepath.Join(repoPath, ".git")

	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return "技能仓库未初始化", nil
	}

	// 尝试获取git状态
	status, err := sr.repo.GetStatus()
	if err != nil {
		return "", err
	}

	result := "技能仓库状态:\n"

	// 显示远程URL（如果有）
	if sr.repo.remoteURL != "" {
		result += fmt.Sprintf("远程仓库: %s\n", sr.repo.remoteURL)
	} else {
		result += "远程仓库: 未设置\n"
	}

	// 显示最新提交（如果有）
	latestCommit, err := sr.repo.GetLatestCommit()
	if err == nil {
		result += fmt.Sprintf("最新提交: %s\n", latestCommit)
	}

	result += "文件状态:\n"
	result += status

	return result, nil
}

// ListSkillsFromRemote 从远程仓库列出技能
func (sr *SkillRepository) ListSkillsFromRemote() ([]*spec.Skill, error) {
	// 先同步到最新
	if err := sr.Sync(); err != nil {
		return nil, err
	}

	// 加载所有技能
	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return nil, err
	}

	// 只使用标准结构：直接从skills目录加载
	skills, err := sr.loadSkillsFromDirectory(skillsDir, false)
	if err != nil {
		return nil, err
	}

	return skills, nil
}

// loadSkillsFromDirectory 从目录加载技能
func (sr *SkillRepository) loadSkillsFromDirectory(dir string, recursive bool) ([]*spec.Skill, error) {
	var skills []*spec.Skill

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillID := entry.Name()
		skillDir := filepath.Join(dir, skillID)

		// 尝试加载技能
		skill, err := sr.loadSkill(skillDir, skillID)
		if err != nil {
			// 如果是递归模式，继续检查子目录
			if recursive {
				subSkills, _ := sr.loadSkillsFromDirectory(skillDir, true)
				skills = append(skills, subSkills...)
			}
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// loadSkill 加载单个技能
func (sr *SkillRepository) loadSkill(skillDir, skillID string) (*spec.Skill, error) {
	// 只支持SKILL.md格式
	skillMdPath := filepath.Join(skillDir, "SKILL.md")
	if _, err := os.Stat(skillMdPath); err == nil {
		return sr.loadSkillFromMarkdown(skillMdPath, skillID)
	}

	return nil, fmt.Errorf("未找到SKILL.md文件")
}

// loadSkillFromMarkdown 从SKILL.md文件加载技能
func (sr *SkillRepository) loadSkillFromMarkdown(mdPath, skillID string) (*spec.Skill, error) {
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, fmt.Errorf("读取SKILL.md失败: %w", err)
	}

	// 解析frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 2 || lines[0] != "---" {
		return nil, fmt.Errorf("无效的SKILL.md格式: 缺少frontmatter")
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
		return nil, fmt.Errorf("解析frontmatter失败: %w", err)
	}

	// 转换为Skill对象
	skill := &spec.Skill{
		ID: skillID,
	}

	// 设置名称
	if name, ok := skillData["name"].(string); ok {
		skill.Name = name
	} else {
		skill.Name = skillID
	}

	// 设置描述
	if desc, ok := skillData["description"].(string); ok {
		skill.Description = desc
	}

	// 设置版本
	skill.Version = "1.0.0"
	if version, ok := skillData["version"].(string); ok {
		skill.Version = version
	}

	// 设置作者
	if source, ok := skillData["source"].(string); ok {
		skill.Author = source
	} else {
		skill.Author = "unknown"
	}

	// 设置标签
	if tagsStr, ok := skillData["tags"].(string); ok {
		skill.Tags = strings.Split(tagsStr, ",")
		for i, tag := range skill.Tags {
			skill.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// 设置兼容性（默认为所有工具）
	skill.Compatibility = "Designed for Cursor and Claude Code (or similar AI coding assistants)"

	return skill, nil
}

// ImportSkill 从远程仓库导入单个技能
func (sr *SkillRepository) ImportSkill(skillID string) error {
	// 先同步到最新
	if err := sr.Sync(); err != nil {
		return err
	}

	// 检查技能是否存在
	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return err
	}

	skillDir := filepath.Join(skillsDir, skillID)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return fmt.Errorf("技能 '%s' 在远程仓库中不存在", skillID)
	}

	// 检查技能文件
	skillMdPath := filepath.Join(skillDir, "SKILL.md")

	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		return fmt.Errorf("技能 '%s' 缺少SKILL.md文件", skillID)
	}

	fmt.Printf("✅ 技能 '%s' 已从远程仓库导入\n", skillID)
	return nil
}

// CreateSkill 创建新技能并推送到远程
func (sr *SkillRepository) CreateSkill(skill *spec.Skill, promptContent string) error {
	// 验证技能信息
	if skill.ID == "" {
		return fmt.Errorf("技能ID不能为空")
	}
	if skill.Name == "" {
		return fmt.Errorf("技能名称不能为空")
	}
	if skill.Version == "" {
		skill.Version = "1.0.0"
	}

	// 创建技能目录
	skillsDir, err := config.GetSkillsDir()
	if err != nil {
		return err
	}

	skillDir := filepath.Join(skillsDir, skill.ID)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("创建技能目录失败: %w", err)
	}

	// 保存SKILL.md（包含frontmatter和内容）
	skillMdPath := filepath.Join(skillDir, "SKILL.md")

	// 构建frontmatter
	frontmatter := fmt.Sprintf(`---
name: %s
description: %s
%smetadata:
  version: %s
  author: %s
  tags: %s
---
`,
		skill.Name,
		skill.Description,
		func() string {
			if skill.Compatibility != "" {
				return fmt.Sprintf("compatibility: %s\n", skill.Compatibility)
			}
			return ""
		}(),
		skill.Version,
		skill.Author,
		strings.Join(skill.Tags, ","))

	// 组合frontmatter和内容
	skillContent := frontmatter + promptContent

	if err := os.WriteFile(skillMdPath, []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("保存SKILL.md失败: %w", err)
	}

	fmt.Printf("✅ 技能 '%s' 创建成功\n", skill.ID)

	// 推送到远程仓库
	if sr.repo.IsInitialized() {
		message := fmt.Sprintf("添加新技能: %s", skill.ID)
		if err := sr.PushChanges(message); err != nil {
			fmt.Printf("⚠️  技能创建成功，但推送到远程失败: %v\n", err)
			fmt.Println("   使用 'skill-hub git push' 手动推送")
		}
	}

	return nil
}

// UpdateRegistry 更新技能注册表
func (sr *SkillRepository) UpdateRegistry() error {
	skills, err := sr.ListSkillsFromRemote()
	if err != nil {
		return err
	}

	// 创建注册表
	registry := spec.Registry{
		Version: "1.0",
		Skills:  make([]spec.SkillMetadata, 0, len(skills)),
	}

	for _, skill := range skills {
		metadata := spec.SkillMetadata{
			ID:            skill.ID,
			Name:          skill.Name,
			Version:       skill.Version,
			Author:        skill.Author,
			Description:   skill.Description,
			Tags:          skill.Tags,
			Compatibility: skill.Compatibility,
		}
		registry.Skills = append(registry.Skills, metadata)
	}

	// 保存注册表
	registryPath, err := config.GetRegistryPath()
	if err != nil {
		return err
	}

	registryData, err := yaml.Marshal(registry)
	if err != nil {
		return fmt.Errorf("序列化注册表失败: %w", err)
	}

	if err := os.WriteFile(registryPath, registryData, 0644); err != nil {
		return fmt.Errorf("保存注册表失败: %w", err)
	}

	// 提交更改
	if sr.repo.IsInitialized() {
		message := "更新技能注册表"
		if err := sr.PushChanges(message); err != nil {
			fmt.Printf("⚠️  注册表更新成功，但推送到远程失败: %v\n", err)
		}
	}

	return nil
}
