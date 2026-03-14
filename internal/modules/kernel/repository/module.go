package repository

import (
	"github.com/muidea/skill-hub/internal/modules/kernel/repository/biz"
	"github.com/muidea/skill-hub/internal/modules/kernel/repository/service"
)

type Repository struct {
	bizPtr     *biz.Repository
	servicePtr *service.Repository
}

func New() *Repository {
	return &Repository{
		bizPtr:     biz.New(),
		servicePtr: service.New(),
	}
}

func (r *Repository) Service() *service.Repository {
	return r.servicePtr
}
