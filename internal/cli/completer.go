package cli

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/muidea/skill-hub/pkg/spec"
)

const shellCompNoFile = cobra.ShellCompDirectiveNoFileComp
const completionCacheTTL = 5 * time.Second

var targetValues = []string{spec.TargetCursor, spec.TargetClaudeCode, spec.TargetOpenCode, spec.TargetAll}

type completionCacheEntry struct {
	items     []string
	expiresAt time.Time
}

var skillCompletionCache = struct {
	mu      sync.Mutex
	entries map[string]completionCacheEntry
}{
	entries: make(map[string]completionCacheEntry),
}

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
	rootDir, err := getHubRootDir()
	if err != nil {
		return nil, shellCompNoFile
	}
	repoFilter := ""
	if idx := strings.Index(toComplete, "/"); idx >= 0 {
		repoFilter = toComplete[:idx]
	}

	cacheKey := rootDir + "::" + repoFilter
	if cached := readSkillCompletionCache(cacheKey); cached != nil {
		return filterPrefix(cached, toComplete), shellCompNoFile
	}

	var repoNames []string
	if repoFilter != "" {
		repoNames = []string{repoFilter}
	}

	skills, err := listSkillMetadata(repoNames)
	if err != nil {
		return nil, shellCompNoFile
	}
	var ids []string
	for _, s := range skills {
		item := s.Repository + "/" + s.ID
		ids = append(ids, item)
	}
	writeSkillCompletionCache(cacheKey, ids)
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
	stateMgr, err := newStateManager()
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
	repos, err := listRepositories(true)
	if err != nil {
		return nil, shellCompNoFile
	}
	if len(repos) == 0 {
		return nil, shellCompNoFile
	}
	var names []string
	for _, repo := range repos {
		names = append(names, repo.Name)
	}
	return filterPrefix(names, toComplete), shellCompNoFile
}

func readSkillCompletionCache(key string) []string {
	skillCompletionCache.mu.Lock()
	defer skillCompletionCache.mu.Unlock()

	entry, ok := skillCompletionCache.entries[key]
	if !ok {
		return nil
	}
	if time.Now().After(entry.expiresAt) {
		delete(skillCompletionCache.entries, key)
		return nil
	}

	items := make([]string, len(entry.items))
	copy(items, entry.items)
	return items
}

func writeSkillCompletionCache(key string, items []string) {
	skillCompletionCache.mu.Lock()
	defer skillCompletionCache.mu.Unlock()

	copied := make([]string, len(items))
	copy(copied, items)
	skillCompletionCache.entries[key] = completionCacheEntry{
		items:     copied,
		expiresAt: time.Now().Add(completionCacheTTL),
	}
}

func resetCompletionCache() {
	skillCompletionCache.mu.Lock()
	defer skillCompletionCache.mu.Unlock()
	skillCompletionCache.entries = make(map[string]completionCacheEntry)
}
