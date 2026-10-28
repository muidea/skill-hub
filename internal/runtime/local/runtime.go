// Package local implements the explicit in-process execution mode used when a
// reachable skill-hubd daemon is unavailable. It assembles focused
// capabilities; it is intentionally not a cross-owner service registry.
package local

import (
	"context"

	projectstatemodule "github.com/muidea/skill-hub/internal/modules/blocks/project_state"
	repositorymodule "github.com/muidea/skill-hub/internal/modules/blocks/repository"
	"github.com/muidea/skill-hub/internal/pkg/configport"
	"github.com/muidea/skill-hub/internal/pkg/gitport"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	"github.com/muidea/skill-hub/internal/pkg/repositoryport"
	"github.com/muidea/skill-hub/internal/pkg/skillport"

	"github.com/muidea/magicCommon/event"
	"github.com/muidea/magicCommon/task"
)

type Runtime struct {
	ownsInfrastructure bool
	hub                event.Hub
	background         task.BackgroundRoutine
	stateOwner         *projectstatemodule.ProjectState
	repoOwner          *repositorymodule.Repository
	repository         repositoryport.Repository
	projectState       projectstateport.ProjectState
	registration       projectstateport.Registration
	git                gitport.Git
	config             configport.Config
	skill              skillport.Skill
	project            *Project
	global             *Global
	maintenance        *Maintenance
	upgrade            *Upgrade
}

func New() *Runtime {
	hub := event.NewHub(32)
	background := task.NewBackgroundRoutine(4)
	stateOwner := projectstatemodule.New()
	repoOwner := repositorymodule.New()
	if err := stateOwner.Setup(context.Background(), hub, background); err != nil {
		panic("set up project-state Block: " + err.Error())
	}
	if err := repoOwner.Setup(context.Background(), hub, background); err != nil {
		stateOwner.Teardown(context.Background())
		hub.Terminate(context.Background())
		background.Shutdown(context.Background())
		panic("set up repository Block: " + err.Error())
	}
	return newRuntime(hub, background, stateOwner, repoOwner, true)
}

// NewWithEventHub assembles local capabilities around framework-owned
// infrastructure. The caller retains ownership of the Hub, BackgroundRoutine
// and registered Blocks.
func NewWithEventHub(hub event.Hub, background task.BackgroundRoutine) *Runtime {
	return newRuntime(hub, background, nil, nil, false)
}

func newRuntime(hub event.Hub, background task.BackgroundRoutine, stateOwner *projectstatemodule.ProjectState, repoOwner *repositorymodule.Repository, ownsInfrastructure bool) *Runtime {
	return &Runtime{
		ownsInfrastructure: ownsInfrastructure,
		hub:                hub,
		background:         background,
		stateOwner:         stateOwner,
		repoOwner:          repoOwner,
		repository:         repositoryport.NewRepository(hub, "local_runtime"),
		projectState:       projectstateport.New(hub, "local_runtime"),
		registration:       projectstateport.NewRegistration(hub, "local_runtime"),
		git:                NewGit(),
		config:             NewConfig(),
		skill:              NewSkill(),
		project:            NewProject(hub, background),
		global:             NewGlobal(hub, background),
		maintenance:        NewMaintenance(),
		upgrade:            NewUpgrade(),
	}
}

// Close releases the explicit in-process EventHub assembly. Callers that
// retain a Runtime for a bounded operation should defer Close.
func (r *Runtime) Close() {
	if r == nil {
		return
	}
	if r.project != nil {
		r.project.Close()
		r.project = nil
	}
	if r.global != nil {
		r.global.Close()
		r.global = nil
	}
	if r.ownsInfrastructure && r.stateOwner != nil {
		r.stateOwner.Teardown(context.Background())
		r.stateOwner = nil
	}
	if r.ownsInfrastructure && r.repoOwner != nil {
		r.repoOwner.Teardown(context.Background())
		r.repoOwner = nil
	}
	if r.ownsInfrastructure && r.hub != nil {
		r.hub.Terminate(context.Background())
		r.hub = nil
	}
	if r.ownsInfrastructure && r.background != nil {
		r.background.Shutdown(context.Background())
		r.background = nil
	}
}

func (r *Runtime) Repository() repositoryport.Repository              { return r.repository }
func (r *Runtime) ProjectState() projectstateport.ProjectState        { return r.projectState }
func (r *Runtime) ProjectRegistration() projectstateport.Registration { return r.registration }
func (r *Runtime) Git() gitport.Git                                   { return r.git }
func (r *Runtime) Config() configport.Config                          { return r.config }
func (r *Runtime) Skill() skillport.Skill                             { return r.skill }
func (r *Runtime) Project() *Project                                  { return r.project }
func (r *Runtime) Global() *Global                                    { return r.global }
func (r *Runtime) Maintenance() *Maintenance                          { return r.maintenance }
func (r *Runtime) Upgrade() *Upgrade                                  { return r.upgrade }
