package project_use

import "github.com/muidea/skill-hub/internal/modules/kernel/project_use/service"

type ProjectUse struct {
	servicePtr *service.ProjectUse
}

func New() *ProjectUse {
	return &ProjectUse{
		servicePtr: service.New(),
	}
}

func (p *ProjectUse) Service() *service.ProjectUse {
	return p.servicePtr
}
