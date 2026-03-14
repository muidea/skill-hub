package service

import "github.com/muidea/skill-hub/internal/engine"

type Skill struct{}

func New() *Skill {
	return &Skill{}
}

func (s *Skill) Manager() (*engine.SkillManager, error) {
	return engine.NewSkillManager()
}

func (s *Skill) SkillsDir() (string, error) {
	return engine.GetSkillsDir()
}
