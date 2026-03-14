package server

import "github.com/muidea/skill-hub/internal/modules/kernel/server/service"

type Server struct {
	servicePtr *service.Server
}

func New() *Server {
	return &Server{
		servicePtr: service.New(),
	}
}

func (s *Server) Service() *service.Server {
	return s.servicePtr
}
