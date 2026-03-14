package service

import "github.com/muidea/skill-hub/internal/cli"

type Hub struct{}

func New() *Hub {
	return &Hub{}
}

func (h *Hub) Execute() error {
	return cli.Execute()
}
