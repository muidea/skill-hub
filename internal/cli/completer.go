package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/internal/config"
	"github.com/muidea/skill-hub/internal/multirepo"
	"github.com/muidea/skill-hub/internal/state"
	"github.com/muidea/skill-hub/pkg/spec"
)

const shellCompNoFile = cobra.ShellCompDirectiveNoFileComp

var targetValues = []string{spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode, spec.TargetAll}

func filterPrefix(candidates []string, toComplete string) []string {
	if toComplete == "" {
		return candidates
	}
	var out []string
	for _, c := range candidates {
		if strings.HasPrefix(c, toComplete) {
			out = append(out, c)
		}
	}
	return out
}

func completeSkillIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	mgr, err := multirepo.NewManager()
	if err != nil {
		return nil, shellCompNoFile
	}
	repoFilter := ""
	if idx := strings.Index(toComplete, "/"); idx >= 0 {
		repoFilter = toComplete[:idx]
	}
	skills, err := mgr.ListSkills(repoFilter)
	if err != nil {
		return nil, shellCompNoFile
	}
	var ids []string
	for _, s := range skills {
		item := s.Repository + "/" + s.ID
		ids = append(ids, item)
	}
	return filterPrefix(ids, toComplete), shellCompNoFile
}

func completeEnabledSkillIDsForCwd(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, shellCompNoFile
	}
	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return nil, shellCompNoFile
	}
	stateMgr, err := state.NewStateManager()
	if err != nil {
		return nil, shellCompNoFile
	}
	skills, err := stateMgr.GetProjectSkills(cwd)
	if err != nil {
		return nil, shellCompNoFile
	}
	var ids []string
	for id := range skills {
		ids = append(ids, id)
	}
	return filterPrefix(ids, toComplete), shellCompNoFile
}

func completeTargetValues(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return filterPrefix(targetValues, toComplete), shellCompNoFile
}

func completeRepoNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, shellCompNoFile
	}
	if cfg.MultiRepo == nil || cfg.MultiRepo.Repositories == nil {
		return nil, shellCompNoFile
	}
	var names []string
	for name := range cfg.MultiRepo.Repositories {
		names = append(names, name)
	}
	return filterPrefix(names, toComplete), shellCompNoFile
}
