package git

import (
	"github.com/muidea/skill-hub/internal/modules/blocks/git/biz"
	"github.com/muidea/skill-hub/internal/modules/blocks/git/service"
)

type Git struct {
	bizPtr     *biz.Git
	servicePtr *service.Git
}

func New() *Git {
	return &Git{
		bizPtr:     biz.New(),
		servicePtr: service.New(),
	}
}

func (g *Git) Service() *service.Git {
	return g.servicePtr
}
