package projectstateport

import (
	projectstatecommon "github.com/muidea/skill-hub/internal/modules/blocks/project_state/pkg/common"
	"github.com/muidea/skill-hub/internal/modules/blocks/project_state/pkg/events"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"

	"github.com/muidea/magicCommon/event"
)

// Registration is the focused CLI contract for idempotent empty-project
// registration and historical request queries. It does not inspect status.
type Registration interface {
	RegisterEmptyProject(projectPath, requestID string) (*spec.ProjectRegistrationReceipt, error)
	QueryProjectRegistration(requestID string) (*spec.ProjectRegistrationReceipt, error)
}

// NewRegistration creates an EventHub-backed project registration port.
func NewRegistration(hub event.Hub, source string) Registration {
	return &eventHubPort{hub: hub, source: source}
}

func (p *eventHubPort) RegisterEmptyProject(projectPath, requestID string) (*spec.ProjectRegistrationReceipt, error) {
	result := p.hub.Send(event.NewEvent(events.TopicRegisterProject, p.source, projectstatecommon.ModuleName, nil,
		events.RegisterProjectCommand{ProjectPath: projectPath, RequestID: requestID}))
	if result == nil {
		return nil, errors.NewWithCode("project.register", errors.ErrSystem, "项目登记未返回结果")
	}
	value, err := event.GetAs[events.RegisterProjectResult](result)
	if err != nil {
		return value.Receipt, err
	}
	if value.Receipt == nil {
		return nil, errors.NewWithCode("project.register", errors.ErrSystem, "项目登记未返回回执")
	}
	return value.Receipt, nil
}

func (p *eventHubPort) QueryProjectRegistration(requestID string) (*spec.ProjectRegistrationReceipt, error) {
	result := p.hub.Send(event.NewEvent(events.TopicQueryRegistration, p.source, projectstatecommon.ModuleName, nil,
		events.QueryRegistrationCommand{RequestID: requestID}))
	if result == nil {
		return nil, errors.NewWithCode("project.register.query", errors.ErrSystem, "回执查询未返回结果")
	}
	value, err := event.GetAs[events.QueryRegistrationResult](result)
	if err != nil {
		return value.Receipt, err
	}
	if value.Receipt == nil {
		return nil, errors.NewWithCode("project.register.query", errors.ErrSystem, "回执查询未返回回执")
	}
	return value.Receipt, nil
}
