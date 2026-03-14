package runtime

import (
	"github.com/muidea/skill-hub/internal/modules/kernel/runtime/biz"
	"github.com/muidea/skill-hub/internal/modules/kernel/runtime/service"
)

type Runtime struct {
	bizPtr     *biz.Runtime
	servicePtr *service.Runtime
}

func New() *Runtime {
	return &Runtime{
		bizPtr:     biz.New(),
		servicePtr: service.New(),
	}
}

func (r *Runtime) Service() *service.Runtime {
	return r.servicePtr
}
