package skill

import (
	"github.com/muidea/skill-hub/internal/modules/kernel/skill/biz"
	"github.com/muidea/skill-hub/internal/modules/kernel/skill/service"
)

type Skill struct {
	bizPtr     *biz.Skill
	servicePtr *service.Skill
}

func New() *Skill {
	return &Skill{
		bizPtr:     biz.New(),
		servicePtr: service.New(),
	}
}

func (s *Skill) Service() *service.Skill {
	return s.servicePtr
}
