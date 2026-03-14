package service

import (
	"os"
	"path/filepath"

	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/internal/multirepo"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

type Repository struct{}

func New() *Repository {
	return &Repository{}
}

func (r *Repository) Manager() (*multirepo.Manager, error) {
	return multirepo.NewManager()
}

func (r *Repository) ListRepositories(includeDisabled bool) ([]config.RepositoryConfig, error) {
	manager, err := r.Manager()
	if err != nil {
		return nil, err
	}
	if includeDisabled {
		return manager.ListAllRepositories()
	}
	return manager.ListRepositories()
}

func (r *Repository) ListSkills(repoNames []string) ([]spec.SkillMetadata, error) {
	manager, err := r.Manager()
	if err != nil {
		return nil, err
	}
	if len(repoNames) == 0 {
		return manager.ListSkills("")
	}
	return manager.ListSkillsInRepositories(repoNames)
}

func (r *Repository) RebuildRepositoryIndex(repoName string) error {
	manager, err := r.Manager()
	if err != nil {
		return err
	}
	return manager.RebuildRepositoryIndex(repoName)
}

func (r *Repository) ArchiveToDefaultRepository(skillID, sourcePath string) error {
	manager, err := r.Manager()
	if err != nil {
		return err
	}
	return manager.ArchiveToDefaultRepository(skillID, sourcePath)
}

func (r *Repository) AddRepository(repoConfig config.RepositoryConfig) error {
	manager, err := r.Manager()
	if err != nil {
		return err
	}
	return manager.AddRepository(repoConfig)
}

func (r *Repository) RemoveRepository(name string) error {
	manager, err := r.Manager()
	if err != nil {
		return err
	}
	return manager.RemoveRepository(name)
}

func (r *Repository) SyncRepository(name string) error {
	manager, err := r.Manager()
	if err != nil {
		return err
	}
	return manager.SyncRepository(name)
}

func (r *Repository) EnableRepository(name string) error {
	manager, err := r.Manager()
	if err != nil {
		return err
	}
	return manager.EnableRepository(name)
}

func (r *Repository) DisableRepository(name string) error {
	manager, err := r.Manager()
	if err != nil {
		return err
	}
	return manager.DisableRepository(name)
}

func (r *Repository) GetRepository(name string) (*config.RepositoryConfig, error) {
	manager, err := r.Manager()
	if err != nil {
		return nil, err
	}
	return manager.GetRepository(name)
}

func (r *Repository) DefaultRepository() (*config.RepositoryConfig, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return cfg.GetArchiveRepository()
}

func (r *Repository) Path(repoName string) (string, error) {
	return config.GetRepositoryPath(repoName)
}

func (r *Repository) ReadSkillContent(repoName, skillID string) (string, error) {
	repoDir, err := r.Path(repoName)
	if err != nil {
		return "", errors.Wrap(err, "ReadSkillContent: 获取仓库路径失败")
	}

	srcPath := filepath.Join(repoDir, "skills", skillID, "SKILL.md")
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return "", errors.NewWithCodef("ReadSkillContent", errors.ErrFileNotFound, "技能文件在仓库中不存在: %s", srcPath)
	}

	content, err := os.ReadFile(srcPath)
	if err != nil {
		return "", errors.WrapWithCode(err, "ReadSkillContent", errors.ErrFileOperation, "读取技能文件失败")
	}

	return string(content), nil
}

func (r *Repository) ReadDefaultRepositorySkillContent(skillID string) (string, error) {
	defaultRepo, err := r.DefaultRepository()
	if err != nil {
		return "", errors.Wrap(err, "ReadDefaultRepositorySkillContent: 获取默认仓库失败")
	}
	return r.ReadSkillContent(defaultRepo.Name, skillID)
}

func (r *Repository) SetDefaultRepository(name string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	if cfg.MultiRepo == nil {
		cfg.MultiRepo = &config.MultiRepoConfig{
			Enabled:      true,
			DefaultRepo:  name,
			Repositories: make(map[string]config.RepositoryConfig),
		}
	} else {
		cfg.MultiRepo.Enabled = true
		cfg.MultiRepo.DefaultRepo = name
	}

	return config.SaveConfig(cfg)
}

func (r *Repository) UpdateRepositoryURL(name, url string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	if cfg.MultiRepo == nil {
		return errors.NewWithCode("UpdateRepositoryURL", errors.ErrConfigInvalid, "多仓库配置未初始化")
	}

	repoCfg, ok := cfg.MultiRepo.Repositories[name]
	if !ok {
		return errors.NewWithCodef("UpdateRepositoryURL", errors.ErrConfigInvalid, "仓库 '%s' 不存在", name)
	}

	repoCfg.URL = url
	cfg.MultiRepo.Repositories[name] = repoCfg
	return config.SaveConfig(cfg)
}
