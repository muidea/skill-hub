package webui

import "github.com/muidea/skill-hub/internal/modules/blocks/webui/service"

type WebUI struct {
	servicePtr *service.WebUI
}

func New() *WebUI {
	return &WebUI{
		servicePtr: service.New(),
	}
}

func (w *WebUI) Service() *service.WebUI {
	return w.servicePtr
}
