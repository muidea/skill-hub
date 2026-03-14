package service

import (
	adapterpkg "github.com/muidea/skill-hub/internal/adapter"
	"github.com/muidea/skill-hub/internal/config"
	gitpkg "github.com/muidea/skill-hub/internal/git"
	adaptermodule "github.com/muidea/skill-hub/internal/modules/blocks/adapter"
	gitmodule "github.com/muidea/skill-hub/internal/modules/blocks/git"
	projectstatemodule "github.com/muidea/skill-hub/internal/modules/kernel/project_state"
	repositorymodule "github.com/muidea/skill-hub/internal/modules/kernel/repository"
	skillmodule "github.com/muidea/skill-hub/internal/modules/kernel/skill"
	"github.com/muidea/skill-hub/internal/multirepo"
	"github.com/muidea/skill-hub/internal/state"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

type Runtime struct {
	repositorySvc   *repositorymodule.Repository
	projectStateSvc *projectstatemodule.ProjectState
	skillSvc        *skillmodule.Skill
	adapterSvc      *adaptermodule.Adapter
	gitSvc          *gitmodule.Git
}

func New() *Runtime {
	return &Runtime{
		repositorySvc:   repositorymodule.New(),
		projectStateSvc: projectstatemodule.New(),
		skillSvc:        skillmodule.New(),
		adapterSvc:      adaptermodule.New(),
		gitSvc:          gitmodule.New(),
	}
}

func (r *Runtime) Config() (*config.Config, error) {
	return config.GetConfig()
}

func (r *Runtime) RootDir() (string, error) {
	return config.GetRootDir()
}

func (r *Runtime) StateManager() (*state.StateManager, error) {
	return r.projectStateSvc.Service().Manager()
}

func (r *Runtime) RepositoryManager() (*multirepo.Manager, error) {
	return r.repositorySvc.Service().Manager()
}

func (r *Runtime) Adapter(target string) (adapterpkg.Adapter, error) {
	return r.adapterSvc.Service().ForTarget(target)
}

func (r *Runtime) DefaultRepository() (*config.RepositoryConfig, error) {
	return r.repositorySvc.Service().DefaultRepository()
}

func (r *Runtime) ListRepositories(includeDisabled bool) ([]config.RepositoryConfig, error) {
	return r.repositorySvc.Service().ListRepositories(includeDisabled)
}

func (r *Runtime) RepositoryPath(repoName string) (string, error) {
	return r.repositorySvc.Service().Path(repoName)
}

func (r *Runtime) ReadDefaultRepositorySkillContent(skillID string) (string, error) {
	content, err := r.repositorySvc.Service().ReadDefaultRepositorySkillContent(skillID)
	if err != nil {
		return "", errors.Wrap(err, "ReadDefaultRepositorySkillContent: 读取默认仓库技能内容失败")
	}
	return content, nil
}

func (r *Runtime) SkillsDir() (string, error) {
	return r.skillSvc.Service().SkillsDir()
}

func (r *Runtime) ListSkillMetadata(repoNames []string) ([]spec.SkillMetadata, error) {
	return r.repositorySvc.Service().ListSkills(repoNames)
}

func (r *Runtime) RebuildRepositoryIndex(repoName string) error {
	return r.repositorySvc.Service().RebuildRepositoryIndex(repoName)
}

func (r *Runtime) ArchiveToDefaultRepository(skillID, sourcePath string) error {
	return r.repositorySvc.Service().ArchiveToDefaultRepository(skillID, sourcePath)
}

func (r *Runtime) AddRepository(repoConfig config.RepositoryConfig) error {
	return r.repositorySvc.Service().AddRepository(repoConfig)
}

func (r *Runtime) RemoveRepository(name string) error {
	return r.repositorySvc.Service().RemoveRepository(name)
}

func (r *Runtime) SyncRepository(name string) error {
	return r.repositorySvc.Service().SyncRepository(name)
}

func (r *Runtime) EnableRepository(name string) error {
	return r.repositorySvc.Service().EnableRepository(name)
}

func (r *Runtime) DisableRepository(name string) error {
	return r.repositorySvc.Service().DisableRepository(name)
}

func (r *Runtime) GetRepository(name string) (*config.RepositoryConfig, error) {
	return r.repositorySvc.Service().GetRepository(name)
}

func (r *Runtime) SetDefaultRepository(name string) error {
	return r.repositorySvc.Service().SetDefaultRepository(name)
}

func (r *Runtime) UpdateRepositoryURL(name, url string) error {
	return r.repositorySvc.Service().UpdateRepositoryURL(name, url)
}

func (r *Runtime) GitRepository(repoPath string) (*gitpkg.Repository, error) {
	return r.gitSvc.Service().Repository(repoPath)
}

func (r *Runtime) SkillsRepository() (*gitpkg.Repository, error) {
	return r.gitSvc.Service().SkillsRepository()
}

func (r *Runtime) SkillRepository() (*gitpkg.SkillRepository, error) {
	return r.gitSvc.Service().SkillRepository()
}

func (r *Runtime) SyncSkillRepositoryAndRefresh() error {
	return r.gitSvc.Service().SyncSkillRepositoryAndRefresh()
}

func (r *Runtime) SkillRepositoryStatus() (string, error) {
	return r.gitSvc.Service().SkillRepositoryStatus()
}

func (r *Runtime) PushSkillRepositoryChanges(message string) error {
	return r.gitSvc.Service().PushSkillRepositoryChanges(message)
}

func (r *Runtime) PushSkillRepositoryCommits() error {
	return r.gitSvc.Service().PushSkillRepositoryCommits()
}

func (r *Runtime) SetSkillRepositoryRemote(url string) error {
	return r.gitSvc.Service().SetSkillRepositoryRemote(url)
}

func (r *Runtime) CleanupTimestampedBackupDirs(basePath string) error {
	return r.adapterSvc.Service().CleanupTimestampedBackupDirs(basePath)
}
