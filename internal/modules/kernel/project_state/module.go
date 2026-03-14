package projectstate

import (
	"github.com/muidea/skill-hub/internal/modules/kernel/project_state/biz"
	"github.com/muidea/skill-hub/internal/modules/kernel/project_state/service"
)

type ProjectState struct {
	bizPtr     *biz.ProjectState
	servicePtr *service.ProjectState
}

func New() *ProjectState {
	return &ProjectState{
		bizPtr:     biz.New(),
		servicePtr: service.New(),
	}
}

func (p *ProjectState) Service() *service.ProjectState {
	return p.servicePtr
}
