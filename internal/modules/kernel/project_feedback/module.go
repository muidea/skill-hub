package project_feedback

import "github.com/muidea/skill-hub/internal/modules/kernel/project_feedback/service"

type ProjectFeedback struct {
	servicePtr *service.ProjectFeedback
}

func New() *ProjectFeedback {
	return &ProjectFeedback{
		servicePtr: service.New(),
	}
}

func (p *ProjectFeedback) Service() *service.ProjectFeedback {
	return p.servicePtr
}
