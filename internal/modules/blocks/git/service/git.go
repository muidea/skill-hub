package service

import gitpkg "github.com/muidea/skill-hub/internal/git"

type Git struct{}

func New() *Git {
	return &Git{}
}

func (g *Git) Repository(repoPath string) (*gitpkg.Repository, error) {
	return gitpkg.NewRepository(repoPath)
}

func (g *Git) SkillsRepository() (*gitpkg.Repository, error) {
	return gitpkg.NewSkillsRepository()
}

func (g *Git) SkillRepository() (*gitpkg.SkillRepository, error) {
	return gitpkg.NewSkillRepository()
}

func (g *Git) SyncSkillRepositoryAndRefresh() error {
	repo, err := g.SkillRepository()
	if err != nil {
		return err
	}
	if err := repo.Sync(); err != nil {
		return err
	}
	return repo.UpdateRegistry()
}

func (g *Git) SkillRepositoryStatus() (string, error) {
	repo, err := g.SkillRepository()
	if err != nil {
		return "", err
	}
	return repo.GetStatus()
}

func (g *Git) PushSkillRepositoryChanges(message string) error {
	repo, err := g.SkillRepository()
	if err != nil {
		return err
	}
	return repo.PushChanges(message)
}

func (g *Git) PushSkillRepositoryCommits() error {
	repo, err := g.SkillsRepository()
	if err != nil {
		return err
	}
	return repo.Push()
}

func (g *Git) SetSkillRepositoryRemote(url string) error {
	repo, err := g.SkillsRepository()
	if err != nil {
		return err
	}
	return repo.SetRemote(url)
}
