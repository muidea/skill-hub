package multirepo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"skill-hub/internal/config"
	"skill-hub/internal/git"
	"skill-hub/pkg/errors"
	"skill-hub/pkg/spec"
)

// Manager 多仓库管理器
type Manager struct {
	config *config.Config
}

// NewManager 创建多仓库管理器
func NewManager() (*Manager, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "NewManager: 获取配置失败")
	}

	return &Manager{
		config: cfg,
	}, nil
}

// ListRepositories 列出所有仓库
func (m *Manager) ListRepositories() ([]config.RepositoryConfig, error) {
	// 只支持多仓库模式
	if m.config.MultiRepo == nil {
		return nil, errors.NewWithCode("ListRepositories", errors.ErrConfigInvalid, "多仓库配置未初始化")
	}

	// 返回所有启用的仓库
	var repos []config.RepositoryConfig
	for _, repo := range m.config.MultiRepo.Repositories {
		if repo.Enabled {
			repos = append(repos, repo)
		}
	}

	// 按名称排序
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	return repos, nil
}

// GetRepository 获取指定仓库配置
func (m *Manager) GetRepository(name string) (*config.RepositoryConfig, error) {
	// 只支持多仓库模式
	if m.config.MultiRepo == nil {
		return nil, errors.NewWithCode("GetRepository", errors.ErrConfigInvalid, "多仓库配置未初始化")
	}

	repo, exists := m.config.MultiRepo.Repositories[name]
	if !exists {
		return nil, errors.NewWithCodef("GetRepository", errors.ErrConfigInvalid, "仓库 '%s' 不存在", name)
	}

	if !repo.Enabled {
		return nil, errors.NewWithCodef("GetRepository", errors.ErrConfigInvalid, "仓库 '%s' 已禁用", name)
	}

	return &repo, nil
}

// AddRepository 添加新仓库
func (m *Manager) AddRepository(repoConfig config.RepositoryConfig) error {
	if m.config.MultiRepo == nil {
		m.config.MultiRepo = &config.MultiRepoConfig{
			Enabled:      true,
			DefaultRepo:  "main",
			Repositories: make(map[string]config.RepositoryConfig),
		}
	}

	// 启用多仓库功能
	m.config.MultiRepo.Enabled = true

	// 检查仓库是否已存在
	if _, exists := m.config.MultiRepo.Repositories[repoConfig.Name]; exists {
		return errors.NewWithCodef("AddRepository", errors.ErrConfigInvalid, "仓库 '%s' 已存在", repoConfig.Name)
	}

	// 设置默认值
	if repoConfig.Branch == "" {
		repoConfig.Branch = "main"
	}
	if repoConfig.Type == "" {
		repoConfig.Type = "community"
	}
	repoConfig.Enabled = true

	// 添加到配置
	m.config.MultiRepo.Repositories[repoConfig.Name] = repoConfig

	// 创建仓库目录
	repoDir, err := config.GetRepositoryPath(repoConfig.Name)
	if err != nil {
		return errors.Wrap(err, "AddRepository: 获取仓库路径失败")
	}

	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return errors.WrapWithCode(err, "AddRepository", errors.ErrFileOperation, "创建仓库目录失败")
	}

	// 克隆或初始化仓库
	if repoConfig.URL != "" {
		if err := git.Clone(repoConfig.URL, repoDir); err != nil {
			return errors.WrapWithCode(err, "AddRepository", errors.ErrGitOperation, "克隆仓库失败")
		}
	} else {
		if err := git.Init(repoDir); err != nil {
			return errors.WrapWithCode(err, "AddRepository", errors.ErrGitOperation, "初始化仓库失败")
		}
	}

	return nil
}

// RemoveRepository 移除仓库
func (m *Manager) RemoveRepository(name string) error {
	if m.config.MultiRepo == nil || !m.config.MultiRepo.Enabled {
		return errors.NewWithCode("RemoveRepository", errors.ErrConfigInvalid, "多仓库功能未启用")
	}

	// 检查仓库是否存在
	if _, exists := m.config.MultiRepo.Repositories[name]; !exists {
		return errors.NewWithCodef("RemoveRepository", errors.ErrConfigInvalid, "仓库 '%s' 不存在", name)
	}

	// 不能移除默认仓库
	if name == m.config.MultiRepo.DefaultRepo {
		return errors.NewWithCode("RemoveRepository", errors.ErrConfigInvalid, "不能移除默认仓库")
	}

	// 从配置中移除
	delete(m.config.MultiRepo.Repositories, name)

	// 可选：删除仓库目录（需要用户确认）
	// 这里暂时不删除目录，保留数据

	return nil
}

// SyncRepository 同步仓库
func (m *Manager) SyncRepository(name string) error {
	// 检查仓库是否存在且启用
	if _, err := m.GetRepository(name); err != nil {
		return err
	}

	repoDir, err := config.GetRepositoryPath(name)
	if err != nil {
		return errors.Wrap(err, "SyncRepository: 获取仓库路径失败")
	}

	// 检查是否为Git仓库
	if !git.IsGitRepo(repoDir) {
		return errors.NewWithCodef("SyncRepository", errors.ErrGitOperation, "目录 '%s' 不是Git仓库", repoDir)
	}

	// 执行git pull
	if err := git.Pull(repoDir); err != nil {
		return errors.WrapWithCode(err, "SyncRepository", errors.ErrGitOperation, "同步仓库失败")
	}

	// 更新最后同步时间
	// 这里需要保存配置，暂时先不实现

	return nil
}

// EnableRepository 启用仓库
func (m *Manager) EnableRepository(name string) error {
	if m.config.MultiRepo == nil || !m.config.MultiRepo.Enabled {
		return errors.NewWithCode("EnableRepository", errors.ErrConfigInvalid, "多仓库功能未启用")
	}

	repo, exists := m.config.MultiRepo.Repositories[name]
	if !exists {
		return errors.NewWithCodef("EnableRepository", errors.ErrConfigInvalid, "仓库 '%s' 不存在", name)
	}

	repo.Enabled = true
	m.config.MultiRepo.Repositories[name] = repo

	return nil
}

// DisableRepository 禁用仓库
func (m *Manager) DisableRepository(name string) error {
	if m.config.MultiRepo == nil || !m.config.MultiRepo.Enabled {
		return errors.NewWithCode("DisableRepository", errors.ErrConfigInvalid, "多仓库功能未启用")
	}

	// 检查仓库是否存在
	if _, exists := m.config.MultiRepo.Repositories[name]; !exists {
		return errors.NewWithCodef("DisableRepository", errors.ErrConfigInvalid, "仓库 '%s' 不存在", name)
	}

	// 不能禁用默认仓库
	if name == m.config.MultiRepo.DefaultRepo {
		return errors.NewWithCode("DisableRepository", errors.ErrConfigInvalid, "不能禁用默认仓库")
	}

	// 更新仓库状态
	repo := m.config.MultiRepo.Repositories[name]
	repo.Enabled = false
	m.config.MultiRepo.Repositories[name] = repo

	return nil
}

// FindSkill 在所有仓库中查找技能
func (m *Manager) FindSkill(skillID string) ([]spec.SkillMetadata, error) {
	var skills []spec.SkillMetadata

	repos, err := m.ListRepositories()
	if err != nil {
		return nil, err
	}

	for _, repo := range repos {
		repoSkills, err := m.findSkillInRepository(skillID, repo.Name)
		if err != nil {
			// 跳过出错的仓库，继续查找其他仓库
			continue
		}
		skills = append(skills, repoSkills...)
	}

	return skills, nil
}

// findSkillInRepository 在指定仓库中查找技能
func (m *Manager) findSkillInRepository(skillID string, repoName string) ([]spec.SkillMetadata, error) {
	repoDir, err := config.GetRepositoryPath(repoName)
	if err != nil {
		return nil, err
	}

	// 检查技能目录是否存在
	skillDir := filepath.Join(repoDir, "skills", skillID)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return nil, nil
	}

	// 读取技能文件
	skillFile := filepath.Join(skillDir, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, errors.WrapWithCode(err, "findSkillInRepository", errors.ErrFileOperation, "读取技能文件失败")
	}

	// 解析技能元数据
	skill, err := parseSkillMetadata(content, repoName, skillID)
	if err != nil {
		return nil, err
	}

	return []spec.SkillMetadata{*skill}, nil
}

// parseSkillMetadata 从技能文件内容解析元数据
func parseSkillMetadata(content []byte, repoName, skillID string) (*spec.SkillMetadata, error) {
	// 这里简化实现，实际需要解析YAML frontmatter
	// 暂时返回基本元数据
	return &spec.SkillMetadata{
		ID:             skillID,
		Name:           skillID,
		Version:        "1.0.0",
		Author:         "unknown",
		Description:    fmt.Sprintf("技能来自 %s 仓库", repoName),
		Repository:     repoName,
		RepositoryPath: filepath.Join("skills", skillID),
	}, nil
}

// LoadSkill 加载完整技能信息
func (m *Manager) LoadSkill(skillID, repoName string) (*spec.Skill, error) {
	repoDir, err := config.GetRepositoryPath(repoName)
	if err != nil {
		return nil, err
	}

	// 检查技能目录是否存在
	skillDir := filepath.Join(repoDir, "skills", skillID)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return nil, errors.SkillNotFound("LoadSkill", skillID)
	}

	// 读取技能文件
	skillFile := filepath.Join(skillDir, "SKILL.md")
	_, err = os.ReadFile(skillFile)
	if err != nil {
		return nil, errors.WrapWithCode(err, "LoadSkill", errors.ErrFileOperation, "读取技能文件失败")
	}

	// 解析技能文件（简化实现，实际需要解析YAML frontmatter）
	// 这里暂时返回基本技能信息
	return &spec.Skill{
		ID:             skillID,
		Name:           skillID,
		Version:        "1.0.0",
		Author:         "unknown",
		Description:    fmt.Sprintf("技能来自 %s 仓库", repoName),
		Tags:           []string{},
		Variables:      []spec.Variable{},
		Repository:     repoName,
		RepositoryPath: filepath.Join("skills", skillID),
	}, nil
}

// SearchSkills 在所有仓库中搜索技能
func (m *Manager) SearchSkills(query string, repoFilter string) ([]spec.SkillMetadata, error) {
	// 简化实现，实际需要遍历所有技能文件
	// 这里暂时返回空结果
	return []spec.SkillMetadata{}, nil
}

// ListSkills 列出所有技能
func (m *Manager) ListSkills(repoFilter string) ([]spec.SkillMetadata, error) {
	var allSkills []spec.SkillMetadata

	repos, err := m.ListRepositories()
	if err != nil {
		return nil, err
	}

	for _, repo := range repos {
		// 如果指定了仓库过滤器，跳过不匹配的仓库
		if repoFilter != "" && repo.Name != repoFilter {
			continue
		}

		repoDir, err := config.GetRepositoryPath(repo.Name)
		if err != nil {
			// 跳过出错的仓库，继续处理其他仓库
			continue
		}

		skillsDir := filepath.Join(repoDir, "skills")
		if _, err := os.Stat(skillsDir); os.IsNotExist(err) {
			// 仓库没有skills目录，跳过
			continue
		}

		// 读取skills目录
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			// 跳过出错的仓库，继续处理其他仓库
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			skillID := entry.Name()
			skillDir := filepath.Join(skillsDir, skillID)
			skillFile := filepath.Join(skillDir, "SKILL.md")

			// 检查是否存在SKILL.md文件
			if _, err := os.Stat(skillFile); os.IsNotExist(err) {
				continue
			}

			// 读取技能文件
			content, err := os.ReadFile(skillFile)
			if err != nil {
				// 跳过无法读取的技能，继续处理其他技能
				continue
			}

			// 解析技能元数据
			skill, err := parseSkillMetadata(content, repo.Name, skillID)
			if err != nil {
				// 跳过解析失败的技能，继续处理其他技能
				continue
			}

			allSkills = append(allSkills, *skill)
		}
	}

	return allSkills, nil
}

// CheckSkillInDefaultRepository 检查技能是否在默认仓库中存在
func (m *Manager) CheckSkillInDefaultRepository(skillID string) (bool, error) {
	defaultRepo, err := m.config.GetArchiveRepository()
	if err != nil {
		return false, err
	}

	repoDir, err := config.GetRepositoryPath(defaultRepo.Name)
	if err != nil {
		return false, err
	}

	skillDir := filepath.Join(repoDir, "skills", skillID)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return false, nil
	}

	return true, nil
}

// ArchiveToDefaultRepository 归档到默认仓库
func (m *Manager) ArchiveToDefaultRepository(skillID, sourcePath string) error {
	defaultRepo, err := m.config.GetArchiveRepository()
	if err != nil {
		return err
	}

	repoDir, err := config.GetRepositoryPath(defaultRepo.Name)
	if err != nil {
		return err
	}

	targetDir := filepath.Join(repoDir, "skills", skillID)

	// 创建目标目录
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return errors.WrapWithCode(err, "ArchiveToDefaultRepository", errors.ErrFileOperation, "创建目标目录失败")
	}

	// 复制技能文件
	if err := copyDirectory(sourcePath, targetDir); err != nil {
		return errors.WrapWithCode(err, "ArchiveToDefaultRepository", errors.ErrFileOperation, "复制技能文件失败")
	}

	return nil
}

// copyDirectory 复制目录
func copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		// 复制文件
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		return os.WriteFile(targetPath, data, info.Mode())
	})
}

// GetDefaultRepository 获取默认仓库
func (m *Manager) GetDefaultRepository() (*config.RepositoryConfig, error) {
	return m.config.GetArchiveRepository()
}
