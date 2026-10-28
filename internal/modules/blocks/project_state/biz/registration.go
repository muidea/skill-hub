package biz

import (
	"github.com/muidea/skill-hub/internal/modules/blocks/project_state/pkg/events"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/event"
)

func (b *ProjectState) handleRegisterProject(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.RegisterProjectCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid project registration command"))
		return
	}
	manager, err := b.service.Manager()
	if err != nil {
		result.Set(nil, cd.WrapError(cd.Unexpected, err, "open project state"))
		return
	}
	receipt, err := manager.RegisterEmptyProject(command.ProjectPath, command.RequestID)
	result.Set(events.RegisterProjectResult{Receipt: receipt}, cd.WrapError(cd.Unexpected, err, "register project"))
}

func (b *ProjectState) handleQueryRegistration(e event.Event, result event.Result) {
	if result == nil {
		return
	}
	command, ok := e.Data().(events.QueryRegistrationCommand)
	if !ok {
		result.Set(nil, cd.NewError(cd.IllegalParam, "invalid project registration query"))
		return
	}
	manager, err := b.service.Manager()
	if err != nil {
		result.Set(nil, cd.WrapError(cd.Unexpected, err, "open project state"))
		return
	}
	receipt, err := manager.QueryProjectRegistration(command.RequestID)
	result.Set(events.QueryRegistrationResult{Receipt: receipt}, cd.WrapError(cd.Unexpected, err, "query project registration"))
}
