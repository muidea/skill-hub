package local

import (
	gitpkg "github.com/muidea/skill-hub/internal/git"
	"github.com/muidea/skill-hub/internal/pkg/gitport"
	"github.com/muidea/skill-hub/pkg/spec"
)

type Git struct{}

var _ gitport.Git = (*Git)(nil)

func NewGit() *Git { return &Git{} }
func (g *Git) CloneRepository(path, url string) error {
	repo, err := gitpkg.NewRepository(path)
	if err != nil {
		return err
	}
	return repo.Clone(url)
}
func (g *Git) InitializeRepository(path string) error {
	_, err := gitpkg.NewRepository(path)
	return err
}
func (g *Git) RepositoryRemoteURLs(path string) ([]string, error) {
	repo, err := gitpkg.NewRepository(path)
	if err != nil {
		return nil, err
	}
	return repo.GetRemote()
}
func (g *Git) SyncDefaultRepository(force bool) (int, error) {
	result, err := g.SyncDefaultRepositoryWithResult(force)
	return result.SkillCount, err
}

func (g *Git) SyncDefaultRepositoryWithResult(force bool) (*spec.PullResult, error) {
	result := &spec.PullResult{}
	repo, err := gitpkg.NewSkillRepository()
	if err != nil {
		return result, err
	}
	err = repo.SyncWithOptions(gitpkg.SyncOptions{Force: force})
	result.StashRef = repo.StashRef()
	if err != nil {
		return result, err
	}
	if err := repo.UpdateRegistry(); err != nil {
		return result, err
	}
	skills, err := repo.ListLocalSkills()
	if err != nil {
		return result, err
	}
	result.SkillCount = len(skills)
	return result, nil
}
func (g *Git) SyncSkillRepositoryAndRefresh() error {
	_, err := g.SyncDefaultRepository(false)
	return err
}
func (g *Git) CheckSkillRepositoryUpdates() (*gitpkg.RemoteUpdateStatus, error) {
	repo, err := gitpkg.NewSkillRepository()
	if err != nil {
		return nil, err
	}
	return repo.CheckUpdates()
}
func (g *Git) SkillRepositoryStatus() (string, error) {
	repo, err := gitpkg.NewSkillRepository()
	if err != nil {
		return "", err
	}
	return repo.GetStatus()
}
func (g *Git) PushSkillRepositoryChanges(message string) error {
	repo, err := gitpkg.NewSkillRepository()
	if err != nil {
		return err
	}
	return repo.PushChanges(message)
}
func (g *Git) PushSkillRepositoryCommits() error {
	repo, err := gitpkg.NewSkillsRepository()
	if err != nil {
		return err
	}
	return repo.Push()
}
func (g *Git) SetSkillRepositoryRemote(url string) error {
	repo, err := gitpkg.NewSkillsRepository()
	if err != nil {
		return err
	}
	return repo.SetRemote(url)
}
