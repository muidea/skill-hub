package project_status

import "github.com/muidea/skill-hub/internal/modules/kernel/project_status/service"

type ProjectStatus struct {
	servicePtr *service.ProjectStatus
}

func New() *ProjectStatus {
	return &ProjectStatus{
		servicePtr: service.New(),
	}
}

func (p *ProjectStatus) Service() *service.ProjectStatus {
	return p.servicePtr
}
