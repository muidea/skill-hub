package biz

import (
	basebiz "github.com/muidea/skill-hub/internal/modules/base/biz"
	"github.com/muidea/skill-hub/internal/modules/blocks/project_state/pkg/common"
	"github.com/muidea/skill-hub/internal/modules/blocks/project_state/pkg/events"
	"github.com/muidea/skill-hub/internal/modules/blocks/project_state/service"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type ProjectState struct {
	basebiz.Base
	service *service.ProjectState
}

func New(hub event.Hub, background task.BackgroundRoutine, stateService *service.ProjectState) *ProjectState {
	biz := &ProjectState{Base: basebiz.New(common.ModuleName, hub, background), service: stateService}
	biz.SubscribeFunc(events.TopicLoadProject, biz.handleLoadProject)
	biz.SubscribeFunc(events.TopicSaveProject, biz.handleSaveProject)
	biz.SubscribeFunc(events.TopicListProjects, biz.handleListProjects)
	biz.SubscribeFunc(events.TopicLoadProjectState, biz.handleLoadProjectState)
	biz.SubscribeFunc(events.TopicProjectHasSkill, biz.handleProjectHasSkill)
	biz.SubscribeFunc(events.TopicRemoveSkill, biz.handleRemoveSkill)
	biz.SubscribeFunc(events.TopicPruneProjects, biz.handlePruneProjects)
	biz.SubscribeFunc(events.TopicRegisterProject, biz.handleRegisterProject)
	biz.SubscribeFunc(events.TopicQueryRegistration, biz.handleQueryRegistration)
	return biz
}

func (b *ProjectState) Teardown() {
	b.UnsubscribeFunc(events.TopicLoadProject)
	b.UnsubscribeFunc(events.TopicSaveProject)
	b.UnsubscribeFunc(events.TopicListProjects)
	b.UnsubscribeFunc(events.TopicLoadProjectState)
	b.UnsubscribeFunc(events.TopicProjectHasSkill)
	b.UnsubscribeFunc(events.TopicRemoveSkill)
	b.UnsubscribeFunc(events.TopicPruneProjects)
	b.UnsubscribeFunc(events.TopicRegisterProject)
	b.UnsubscribeFunc(events.TopicQueryRegistration)
}

func (b *ProjectState) handleLoadProject(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.LoadProjectCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid project-state load command"))
		return
	}
	manager, err := b.service.Manager()
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	project, err := manager.FindProjectByPath(command.ProjectPath)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	if project == nil {
		result.Set(events.LoadProjectResult{}, nil)
		return
	}
	result.Set(events.LoadProjectResult{Project: *project, Found: true}, nil)
}

func (b *ProjectState) handleSaveProject(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.SaveProjectCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid project-state save command"))
		return
	}
	manager, err := b.service.Manager()
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	if err := manager.SaveProjectState(&command.Project); err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.SaveProjectResult{}, nil)
}

func (b *ProjectState) handleListProjects(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	if _, ok := e.Data().(events.ListProjectsCommand); !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid project-state list command"))
		return
	}
	manager, err := b.service.Manager()
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	projects, err := manager.LoadAllProjectStates()
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.ListProjectsResult{Projects: projects}, nil)
}

func (b *ProjectState) handleLoadProjectState(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.LoadProjectStateCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid project-state load-state command"))
		return
	}
	manager, err := b.service.Manager()
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	project, err := manager.LoadProjectState(command.ProjectPath)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.LoadProjectStateResult{Project: *project}, nil)
}

func (b *ProjectState) handleProjectHasSkill(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.ProjectHasSkillCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid project-state has-skill command"))
		return
	}
	manager, err := b.service.Manager()
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	hasSkill, err := manager.ProjectHasSkill(command.ProjectPath, command.SkillID)
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.ProjectHasSkillResult{HasSkill: hasSkill}, nil)
}

func (b *ProjectState) handleRemoveSkill(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.RemoveSkillCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid project-state remove-skill command"))
		return
	}
	manager, err := b.service.Manager()
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	if err := manager.RemoveSkillFromProject(command.ProjectPath, command.SkillID); err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.RemoveSkillResult{}, nil)
}

func (b *ProjectState) handlePruneProjects(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	if _, ok := e.Data().(events.PruneProjectsCommand); !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid project-state prune command"))
		return
	}
	manager, err := b.service.Manager()
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	removed, err := manager.PruneInvalidProjectStates()
	if err != nil {
		result.Set(nil, cd.NewError(cd.Unexpected, err.Error()))
		return
	}
	result.Set(events.PruneProjectsResult{Removed: removed}, nil)
}
