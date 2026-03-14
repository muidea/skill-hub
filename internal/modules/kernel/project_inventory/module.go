package project_inventory

import "github.com/muidea/skill-hub/internal/modules/kernel/project_inventory/service"

type ProjectInventory struct {
	servicePtr *service.ProjectInventory
}

func New() *ProjectInventory {
	return &ProjectInventory{
		servicePtr: service.New(),
	}
}

func (p *ProjectInventory) Service() *service.ProjectInventory {
	return p.servicePtr
}
