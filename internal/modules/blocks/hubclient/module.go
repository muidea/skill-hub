package hubclient

import "github.com/muidea/skill-hub/internal/modules/blocks/hubclient/service"

type HubClient struct {
	servicePtr *service.Client
}

func New(baseURL string) *HubClient {
	return &HubClient{
		servicePtr: service.New(baseURL),
	}
}

func NewWithSecret(baseURL, secretKey string) *HubClient {
	return &HubClient{
		servicePtr: service.NewWithSecret(baseURL, secretKey),
	}
}

func (h *HubClient) Service() *service.Client {
	return h.servicePtr
}
