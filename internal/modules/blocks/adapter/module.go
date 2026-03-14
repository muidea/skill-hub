package adapter

import (
	"github.com/muidea/skill-hub/internal/modules/blocks/adapter/biz"
	"github.com/muidea/skill-hub/internal/modules/blocks/adapter/service"
)

type Adapter struct {
	bizPtr     *biz.Adapter
	servicePtr *service.Adapter
}

func New() *Adapter {
	return &Adapter{
		bizPtr:     biz.New(),
		servicePtr: service.New(),
	}
}

func (a *Adapter) Service() *service.Adapter {
	return a.servicePtr
}
