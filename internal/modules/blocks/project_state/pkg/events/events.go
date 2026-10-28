// Package events owns typed commands for the project-state Block.
package events

import "github.com/muidea/skill-hub/pkg/spec"

const (
	TopicLoadProject       = "skillhub.project-state.load"
	TopicSaveProject       = "skillhub.project-state.save"
	TopicListProjects      = "skillhub.project-state.list"
	TopicLoadProjectState  = "skillhub.project-state.load-state"
	TopicProjectHasSkill   = "skillhub.project-state.has-skill"
	TopicRemoveSkill       = "skillhub.project-state.remove-skill"
	TopicPruneProjects     = "skillhub.project-state.prune-projects"
	TopicRegisterProject   = "skillhub.project-state.register-project"
	TopicQueryRegistration = "skillhub.project-state.query-registration"
)

type LoadProjectCommand struct{ ProjectPath string }
type LoadProjectResult struct {
	Project spec.ProjectState
	Found   bool
}
type SaveProjectCommand struct{ Project spec.ProjectState }
type SaveProjectResult struct{}

type ListProjectsCommand struct{}
type ListProjectsResult struct{ Projects map[string]spec.ProjectState }

type LoadProjectStateCommand struct{ ProjectPath string }
type LoadProjectStateResult struct{ Project spec.ProjectState }

type ProjectHasSkillCommand struct {
	ProjectPath string
	SkillID     string
}
type ProjectHasSkillResult struct{ HasSkill bool }

type RemoveSkillCommand struct {
	ProjectPath string
	SkillID     string
}
type RemoveSkillResult struct{}

type PruneProjectsCommand struct{}
type PruneProjectsResult struct{ Removed []string }

type RegisterProjectCommand struct {
	ProjectPath string
	RequestID   string
}
type RegisterProjectResult struct {
	Receipt *spec.ProjectRegistrationReceipt
}

type QueryRegistrationCommand struct{ RequestID string }
type QueryRegistrationResult struct {
	Receipt *spec.ProjectRegistrationReceipt
}
