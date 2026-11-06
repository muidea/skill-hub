// Package gitport defines the Git operations needed by local command and API
// adapters without exposing the Git Block implementation.
package gitport

import (
	gitpkg "github.com/muidea/skill-hub/internal/git"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
)

type RemoteUpdateStatus = gitpkg.RemoteUpdateStatus

type Git interface {
	CloneRepository(path, url string) error
	InitializeRepository(path string) error
	RepositoryRemoteURLs(path string) ([]string, error)
	SyncDefaultRepository(force bool) (int, error)
	SyncDefaultRepositoryWithResult(force bool) (*spec.PullResult, error)
	SyncSkillRepositoryAndRefresh() error
	CheckSkillRepositoryUpdates() (*gitpkg.RemoteUpdateStatus, error)
	SkillRepositoryStatus() (string, error)
	PushSkillRepositoryChanges(message string) error
	PushSkillRepositoryCommits() error
	SetSkillRepositoryRemote(url string) error
}

// PullDiagnostic extracts diagnostics shared by local and HTTP bridge errors.
func PullDiagnostic(err error) *spec.PullDiagnostic {
	details := errors.Details(err)
	diagnostic, _ := details["pull"].(*spec.PullDiagnostic)
	return diagnostic
}

func SuggestedCommitMessage(files []string) string { return gitpkg.SuggestedCommitMessage(files) }
func SuggestedCommitMessageFromStatus(status string) string {
	return gitpkg.SuggestedCommitMessageFromStatus(status)
}
