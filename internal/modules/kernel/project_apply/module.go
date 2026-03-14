package project_apply

import "github.com/muidea/skill-hub/internal/modules/kernel/project_apply/service"

type ProjectApply struct {
	servicePtr *service.ProjectApply
}

func New() *ProjectApply {
	return &ProjectApply{
		servicePtr: service.New(),
	}
}

func (p *ProjectApply) Service() *service.ProjectApply {
	return p.servicePtr
}
