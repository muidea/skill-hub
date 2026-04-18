package project_lifecycle

import "github.com/muidea/skill-hub/internal/modules/kernel/project_lifecycle/service"

type ProjectLifecycle struct {
	servicePtr *service.ProjectLifecycle
}

func New() *ProjectLifecycle {
	return &ProjectLifecycle{
		servicePtr: service.New(),
	}
}

func (p *ProjectLifecycle) Service() *service.ProjectLifecycle {
	return p.servicePtr
}
