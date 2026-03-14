package hub

import (
	"github.com/muidea/skill-hub/internal/modules/kernel/hub/biz"
	"github.com/muidea/skill-hub/internal/modules/kernel/hub/service"
)

type Hub struct {
	bizPtr     *biz.Hub
	servicePtr *service.Hub
}

func New() *Hub {
	return &Hub{
		bizPtr:     biz.New(),
		servicePtr: service.New(),
	}
}

func (h *Hub) Execute() error {
	return h.servicePtr.Execute()
}
