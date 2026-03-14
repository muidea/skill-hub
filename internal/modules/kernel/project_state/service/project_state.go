package service

import "github.com/muidea/skill-hub/internal/state"

type ProjectState struct{}

func New() *ProjectState {
	return &ProjectState{}
}

func (p *ProjectState) Manager() (*state.StateManager, error) {
	return state.NewStateManager()
}
