package httpapi

import "github.com/muidea/skill-hub/internal/modules/blocks/httpapi/service"

type HTTPAPI struct {
	servicePtr *service.HTTPAPI
}

func New() *HTTPAPI {
	return &HTTPAPI{
		servicePtr: service.New(),
	}
}

func (h *HTTPAPI) Service() *service.HTTPAPI {
	return h.servicePtr
}
