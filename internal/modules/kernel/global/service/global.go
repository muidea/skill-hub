package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/muidea/skill-hub/internal/config"
	repositorymodule "github.com/muidea/skill-hub/internal/modules/kernel/repository"
	"github.com/muidea/skill-hub/pkg/errors"
)

const (
	ManifestFileName = ".skill-hub-manifest.json"

	StatusOK              = "ok"
	StatusNotApplied      = "not_applied"
	StatusModified        = "modified"
	StatusStale           = "stale"
	StatusConflict        = "conflict"
	StatusOrphaned        = "orphaned"
	StatusMissingAgentDir = "missing_agent_dir"
	StatusRemoved         = "removed"
	StatusPlanned         = "planned"
	StatusApplied         = "applied"
	StatusError           = "error"
)

type Global struct {
	repositorySvc *repositorymodule.Repository
}

func New() *Global {
	return &Global{
		repositorySvc: repositorymodule.New(),
	}
}

type State struct {
	EnabledSkills map[string]SkillState `json:"enabled_skills"`
}

type SkillState struct {
	SkillID          string            `json:"skill_id"`
	Version          string            `json:"version"`
	SourceRepository string            `json:"source_repository"`
	Agents           []string          `json:"agents"`
	Variables        map[string]string `json:"variables,omitempty"`
	ContentHash      string            `json:"content_hash,omitempty"`
	UpdatedAt        string            `json:"updated_at,omitempty"`
	AppliedAt        string            `json:"applied_at,omitempty"`
}

type AgentInfo struct {
	Name       string `json:"name"`
	SkillsDir  string `json:"skills_dir"`
	Detected   bool   `json:"detected"`
	Configured bool   `json:"configured"`
	Reason     string `json:"reason,omitempty"`
}

type Manifest struct {
	ManagedBy        string `json:"managed_by"`
	Scope            string `json:"scope"`
	Agent            string `json:"agent"`
	SkillID          string `json:"skill_id"`
	SourceRepository string `json:"source_repository"`
	SourceHash       string `json:"source_hash"`
	AppliedHash      string `json:"applied_hash"`
	AppliedAt        string `json:"applied_at"`
}

type UseResult struct {
	SkillID    string   `json:"skill_id"`
	Version    string   `json:"version"`
	Repository string   `json:"repository"`
	Agents     []string `json:"agents"`
}

type StatusSummary struct {
	Scope       string       `json:"scope"`
	GlobalPath  string       `json:"global_path"`
	SkillCount  int          `json:"skill_count"`
	Agents      []AgentInfo  `json:"agents"`
	Items       []StatusItem `json:"items"`
	Orphaned    []StatusItem `json:"orphaned,omitempty"`
	GeneratedAt string       `json:"generated_at"`
}

type StatusItem struct {
	SkillID          string `json:"skill_id"`
	Agent            string `json:"agent"`
	Status           string `json:"status"`
	Message          string `json:"message,omitempty"`
	SourceRepository string `json:"source_repository,omitempty"`
	Version          string `json:"version,omitempty"`
	SourceHash       string `json:"source_hash,omitempty"`
	AppliedHash      string `json:"applied_hash,omitempty"`
	ActualHash       string `json:"actual_hash,omitempty"`
	SourcePath       string `json:"source_path,omitempty"`
	TargetPath       string `json:"target_path,omitempty"`
}

type ApplyResult struct {
	Scope      string       `json:"scope"`
	DryRun     bool         `json:"dry_run"`
	Force      bool         `json:"force"`
	Items      []StatusItem `json:"items"`
	AppliedAt  string       `json:"applied_at"`
	GlobalPath string       `json:"global_path"`
}

type RemoveResult struct {
	Scope     string       `json:"scope"`
	SkillID   string       `json:"skill_id"`
	Force     bool         `json:"force"`
	Items     []StatusItem `json:"items"`
	RemovedAt string       `json:"removed_at"`
}

func (g *Global) EnableSkill(skillID, repoName string, agents []string, variables map[string]string) (*UseResult, error) {
	if skillID == "" {
		return nil, errors.NewWithCode("EnableGlobalSkill", errors.ErrInvalidInput, "技能 ID 不能为空")
	}

	candidates, err := g.repositorySvc.Service().FindSkill(skillID)
	if err != nil {
		return nil, errors.Wrap(err, "EnableGlobalSkill: 查找技能失败")
	}
	if len(candidates) == 0 {
		return nil, errors.SkillNotFound("EnableGlobalSkill", skillID)
	}

	selectedRepo := repoName
	if selectedRepo == "" {
		if len(candidates) != 1 {
			return nil, errors.NewWithCode("EnableGlobalSkill", errors.ErrInvalidInput, "技能存在多个候选仓库，必须指定 repository")
		}
		selectedRepo = candidates[0].Repository
	}

	fullSkill, err := g.repositorySvc.Service().LoadSkill(skillID, selectedRepo)
	if err != nil {
		return nil, errors.Wrap(err, "EnableGlobalSkill: 加载技能详情失败")
	}

	resolvedAgents, err := g.resolveAgentNames(agents, true)
	if err != nil {
		return nil, err
	}

	srcDir, err := g.sourceSkillDir(selectedRepo, skillID)
	if err != nil {
		return nil, err
	}
	sourceHash, err := hashDirectory(srcDir)
	if err != nil {
		return nil, errors.Wrap(err, "EnableGlobalSkill: 计算技能内容哈希失败")
	}

	state, err := loadState()
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	existing := state.EnabledSkills[skillID]
	mergedAgents := mergeStrings(existing.Agents, resolvedAgents)
	state.EnabledSkills[skillID] = SkillState{
		SkillID:          skillID,
		Version:          fullSkill.Version,
		SourceRepository: selectedRepo,
		Agents:           mergedAgents,
		Variables:        variables,
		ContentHash:      sourceHash,
		UpdatedAt:        now,
		AppliedAt:        existing.AppliedAt,
	}

	if err := saveState(state); err != nil {
		return nil, errors.Wrap(err, "EnableGlobalSkill: 保存全局状态失败")
	}

	return &UseResult{
		SkillID:    skillID,
		Version:    fullSkill.Version,
		Repository: selectedRepo,
		Agents:     mergedAgents,
	}, nil
}

func (g *Global) Inspect(skillID string, agentFilters []string) (*StatusSummary, error) {
	state, err := loadState()
	if err != nil {
		return nil, err
	}

	globalPath, err := globalSkillsDir()
	if err != nil {
		return nil, err
	}

	agentInfos, err := g.resolveAgents(agentFilters, false)
	if err != nil {
		return nil, err
	}
	agentByName := make(map[string]AgentInfo, len(agentInfos))
	for _, agent := range agentInfos {
		agentByName[agent.Name] = agent
	}

	skillIDs, err := selectEnabledSkillIDs(state, skillID, "InspectGlobalSkills")
	if err != nil {
		return nil, err
	}

	summarySkillCount := 0
	for _, id := range skillIDs {
		if len(filterDesiredAgents(state.EnabledSkills[id].Agents, agentFilters)) > 0 {
			summarySkillCount++
		}
	}

	summary := &StatusSummary{
		Scope:       "global",
		GlobalPath:  globalPath,
		SkillCount:  summarySkillCount,
		Agents:      agentInfos,
		Items:       []StatusItem{},
		GeneratedAt: time.Now().Format(time.RFC3339),
	}

	for _, id := range skillIDs {
		skillState := state.EnabledSkills[id]
		targetAgents := filterDesiredAgents(skillState.Agents, agentFilters)
		if skillID != "" && len(agentFilters) > 0 && len(targetAgents) == 0 {
			return nil, errors.NewWithCodef("InspectGlobalSkills", errors.ErrSkillNotFound, "技能 %s 未对指定 agent 启用: %s", skillID, strings.Join(normalizeAgentNames(agentFilters), ", "))
		}
		if len(targetAgents) == 0 {
			continue
		}
		for _, agentName := range targetAgents {
			agent, ok := agentByName[agentName]
			if !ok {
				agent = ResolveAgent(agentName)
			}
			item := g.inspectDesiredSkill(skillState, agent)
			summary.Items = append(summary.Items, item)
		}
	}

	orphaned, err := scanOrphanedManagedSkills(state, agentInfos)
	if err == nil {
		summary.Orphaned = orphaned
	}

	return summary, nil
}

func (g *Global) Apply(skillID string, agentFilters []string, dryRun, force bool) (*ApplyResult, error) {
	state, err := loadState()
	if err != nil {
		return nil, err
	}
	globalPath, err := globalSkillsDir()
	if err != nil {
		return nil, err
	}
	if !dryRun {
		if err := os.MkdirAll(globalPath, 0755); err != nil {
			return nil, errors.WrapWithCode(err, "ApplyGlobalSkills", errors.ErrFileOperation, "创建全局镜像目录失败")
		}
	}

	agentInfos, err := g.resolveAgents(agentFilters, false)
	if err != nil {
		return nil, err
	}
	agentByName := make(map[string]AgentInfo, len(agentInfos))
	for _, agent := range agentInfos {
		agentByName[agent.Name] = agent
	}

	skillIDs, err := selectEnabledSkillIDs(state, skillID, "ApplyGlobalSkills")
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	result := &ApplyResult{
		Scope:      "global",
		DryRun:     dryRun,
		Force:      force,
		Items:      []StatusItem{},
		AppliedAt:  now,
		GlobalPath: globalPath,
	}

	for _, id := range skillIDs {
		skillState := state.EnabledSkills[id]
		targetAgents := filterDesiredAgents(skillState.Agents, agentFilters)
		// When no agent filters are specified, merge currently detected agents
		// so that newly installed agents (e.g., claude) also receive the skill.
		if len(agentFilters) == 0 {
			detectedNames := make([]string, 0, len(agentInfos))
			for _, a := range agentInfos {
				detectedNames = append(detectedNames, a.Name)
			}
			targetAgents = mergeStrings(targetAgents, detectedNames)
		}
		if skillID != "" && len(agentFilters) > 0 && len(targetAgents) == 0 {
			return nil, errors.NewWithCodef("ApplyGlobalSkills", errors.ErrSkillNotFound, "技能 %s 未对指定 agent 启用: %s", skillID, strings.Join(normalizeAgentNames(agentFilters), ", "))
		}
		if len(targetAgents) == 0 {
			continue
		}
		srcDir, srcErr := g.sourceSkillDir(skillState.SourceRepository, id)
		sourceHash := ""
		if srcErr == nil {
			sourceHash, srcErr = hashDirectory(srcDir)
		}
		if srcErr != nil {
			for _, agentName := range targetAgents {
				result.Items = append(result.Items, StatusItem{
					SkillID:          id,
					Agent:            agentName,
					Status:           StatusError,
					Message:          srcErr.Error(),
					SourceRepository: skillState.SourceRepository,
				})
			}
			continue
		}

		mirrorDir := filepath.Join(globalPath, filepath.FromSlash(id))
		if dryRun {
			result.Items = append(result.Items, g.plannedMirrorItem(skillState, mirrorDir, sourceHash))
		} else if err := syncDirectory(srcDir, mirrorDir, ""); err != nil {
			for _, agentName := range targetAgents {
				result.Items = append(result.Items, StatusItem{
					SkillID:          id,
					Agent:            agentName,
					Status:           StatusError,
					Message:          err.Error(),
					SourceRepository: skillState.SourceRepository,
					SourceHash:       sourceHash,
					TargetPath:       mirrorDir,
				})
			}
			continue
		}

		for _, agentName := range targetAgents {
			agent, ok := agentByName[agentName]
			if !ok {
				agent = ResolveAgent(agentName)
			}
			item := g.applyToAgent(skillState, agent, srcDir, sourceHash, dryRun, force, now)
			result.Items = append(result.Items, item)
		}

		if !dryRun {
			skillState.ContentHash = sourceHash
			skillState.AppliedAt = now
			skillState.Agents = targetAgents
			state.EnabledSkills[id] = skillState
		}
	}

	if !dryRun {
		if err := saveState(state); err != nil {
			return nil, errors.Wrap(err, "ApplyGlobalSkills: 保存全局状态失败")
		}
	}

	return result, nil
}

func (g *Global) Remove(skillID string, agentFilters []string, force bool) (*RemoveResult, error) {
	if skillID == "" {
		return nil, errors.NewWithCode("RemoveGlobalSkill", errors.ErrInvalidInput, "技能 ID 不能为空")
	}

	state, err := loadState()
	if err != nil {
		return nil, err
	}
	skillState, ok := state.EnabledSkills[skillID]
	if !ok {
		return nil, errors.SkillNotFound("RemoveGlobalSkill", skillID)
	}

	targetAgents := filterDesiredAgents(skillState.Agents, agentFilters)
	if len(targetAgents) == 0 && len(agentFilters) > 0 {
		targetAgents = normalizeAgentNames(agentFilters)
	}

	agentInfos, err := g.resolveAgents(targetAgents, false)
	if err != nil {
		return nil, err
	}
	agentByName := make(map[string]AgentInfo, len(agentInfos))
	for _, agent := range agentInfos {
		agentByName[agent.Name] = agent
	}

	result := &RemoveResult{
		Scope:     "global",
		SkillID:   skillID,
		Force:     force,
		Items:     []StatusItem{},
		RemovedAt: time.Now().Format(time.RFC3339),
	}

	remainingAgents := stringSet(skillState.Agents)
	for _, agentName := range targetAgents {
		agent, ok := agentByName[agentName]
		if !ok {
			agent = ResolveAgent(agentName)
		}
		item := removeFromAgent(skillState, agent, force)
		result.Items = append(result.Items, item)
		if item.Status == StatusRemoved || item.Status == StatusNotApplied || item.Status == StatusMissingAgentDir {
			delete(remainingAgents, agentName)
		}
	}

	if len(remainingAgents) == 0 {
		delete(state.EnabledSkills, skillID)
		if globalPath, err := globalSkillsDir(); err == nil {
			_ = os.RemoveAll(filepath.Join(globalPath, filepath.FromSlash(skillID)))
		}
	} else {
		skillState.Agents = sortedKeys(remainingAgents)
		state.EnabledSkills[skillID] = skillState
	}

	if err := saveState(state); err != nil {
		return nil, errors.Wrap(err, "RemoveGlobalSkill: 保存全局状态失败")
	}

	return result, nil
}

func (g *Global) inspectDesiredSkill(skillState SkillState, agent AgentInfo) StatusItem {
	item := StatusItem{
		SkillID:          skillState.SkillID,
		Agent:            agent.Name,
		SourceRepository: skillState.SourceRepository,
		Version:          skillState.Version,
		TargetPath:       filepath.Join(agent.SkillsDir, filepath.FromSlash(skillState.SkillID)),
	}

	srcDir, err := g.sourceSkillDir(skillState.SourceRepository, skillState.SkillID)
	if err != nil {
		item.Status = StatusError
		item.Message = err.Error()
		return item
	}
	item.SourcePath = srcDir
	sourceHash, err := hashDirectory(srcDir)
	if err != nil {
		item.Status = StatusError
		item.Message = err.Error()
		return item
	}
	item.SourceHash = sourceHash

	if !agent.Configured {
		item.Status = StatusMissingAgentDir
		item.Message = "agent 全局 skills 目录不存在"
		return item
	}

	info, err := os.Stat(item.TargetPath)
	if os.IsNotExist(err) {
		item.Status = StatusNotApplied
		item.Message = "目标 agent 目录中未应用该技能"
		return item
	}
	if err != nil {
		item.Status = StatusError
		item.Message = err.Error()
		return item
	}
	if !info.IsDir() {
		item.Status = StatusConflict
		item.Message = "目标路径已存在但不是目录"
		return item
	}

	manifest, err := readManifest(filepath.Join(item.TargetPath, ManifestFileName))
	if err != nil {
		item.Status = StatusConflict
		item.Message = "目标目录存在但不是 Skill-Hub 托管目录"
		return item
	}
	if !isManagedManifest(manifest, skillState.SkillID, agent.Name) {
		item.Status = StatusConflict
		item.Message = "manifest 不属于当前全局技能目标"
		return item
	}
	item.AppliedHash = manifest.AppliedHash

	actualHash, err := hashDirectory(item.TargetPath)
	if err != nil {
		item.Status = StatusError
		item.Message = err.Error()
		return item
	}
	item.ActualHash = actualHash

	switch {
	case actualHash != manifest.AppliedHash && actualHash != sourceHash:
		item.Status = StatusModified
		item.Message = "目标目录内容与 manifest 不一致"
	case actualHash != sourceHash:
		item.Status = StatusStale
		item.Message = "来源仓库内容已变化，目标目录需要刷新"
	default:
		item.Status = StatusOK
	}

	return item
}

func (g *Global) applyToAgent(skillState SkillState, agent AgentInfo, srcDir, sourceHash string, dryRun, force bool, appliedAt string) StatusItem {
	targetDir := filepath.Join(agent.SkillsDir, filepath.FromSlash(skillState.SkillID))
	item := StatusItem{
		SkillID:          skillState.SkillID,
		Agent:            agent.Name,
		SourceRepository: skillState.SourceRepository,
		Version:          skillState.Version,
		SourceHash:       sourceHash,
		SourcePath:       srcDir,
		TargetPath:       targetDir,
	}

	if dryRun {
		item.Status = StatusPlanned
		item.Message = "dry-run: 将刷新 agent 全局 skill"
		return item
	}

	if err := os.MkdirAll(agent.SkillsDir, 0755); err != nil {
		item.Status = StatusError
		item.Message = err.Error()
		return item
	}

	if info, err := os.Stat(targetDir); err == nil {
		if !info.IsDir() {
			if !force {
				item.Status = StatusConflict
				item.Message = "目标路径已存在但不是目录；使用 --force 覆盖"
				return item
			}
			if err := backupAndRemove(targetDir); err != nil {
				item.Status = StatusError
				item.Message = err.Error()
				return item
			}
		} else if manifest, err := readManifest(filepath.Join(targetDir, ManifestFileName)); err != nil || !isManagedManifest(manifest, skillState.SkillID, agent.Name) {
			if !force {
				item.Status = StatusConflict
				item.Message = "目标目录存在但不是 Skill-Hub 托管目录；使用 --force 覆盖"
				return item
			}
			if err := backupAndRemove(targetDir); err != nil {
				item.Status = StatusError
				item.Message = err.Error()
				return item
			}
		}
	} else if err != nil && !os.IsNotExist(err) {
		item.Status = StatusError
		item.Message = err.Error()
		return item
	}

	if err := syncDirectory(srcDir, targetDir, ManifestFileName); err != nil {
		item.Status = StatusError
		item.Message = err.Error()
		return item
	}

	manifest := Manifest{
		ManagedBy:        "skill-hub",
		Scope:            "global",
		Agent:            agent.Name,
		SkillID:          skillState.SkillID,
		SourceRepository: skillState.SourceRepository,
		SourceHash:       sourceHash,
		AppliedHash:      sourceHash,
		AppliedAt:        appliedAt,
	}
	if err := writeManifest(filepath.Join(targetDir, ManifestFileName), manifest); err != nil {
		item.Status = StatusError
		item.Message = err.Error()
		return item
	}

	item.Status = StatusApplied
	item.AppliedHash = sourceHash
	item.ActualHash = sourceHash
	return item
}

func (g *Global) plannedMirrorItem(skillState SkillState, mirrorDir, sourceHash string) StatusItem {
	return StatusItem{
		SkillID:          skillState.SkillID,
		Agent:            "skill-hub",
		Status:           StatusPlanned,
		Message:          "dry-run: 将刷新 Skill-Hub 全局镜像",
		SourceRepository: skillState.SourceRepository,
		Version:          skillState.Version,
		SourceHash:       sourceHash,
		TargetPath:       mirrorDir,
	}
}

func (g *Global) sourceSkillDir(repoName, skillID string) (string, error) {
	if repoName == "" {
		defaultRepo, err := g.repositorySvc.Service().DefaultRepository()
		if err != nil {
			return "", errors.Wrap(err, "sourceSkillDir: 获取默认仓库失败")
		}
		repoName = defaultRepo.Name
	}
	repoPath, err := g.repositorySvc.Service().Path(repoName)
	if err != nil {
		return "", errors.Wrap(err, "sourceSkillDir: 获取仓库路径失败")
	}
	srcDir := filepath.Join(repoPath, "skills", filepath.FromSlash(skillID))
	if _, err := os.Stat(filepath.Join(srcDir, "SKILL.md")); err != nil {
		if os.IsNotExist(err) {
			return "", errors.NewWithCodef("sourceSkillDir", errors.ErrFileNotFound, "技能文件在仓库中不存在: %s", srcDir)
		}
		return "", errors.Wrap(err, "sourceSkillDir: 检查仓库技能失败")
	}
	return srcDir, nil
}

func (g *Global) resolveAgentNames(agentNames []string, requireDetected bool) ([]string, error) {
	agents, err := g.resolveAgents(agentNames, requireDetected)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(agents))
	for _, agent := range agents {
		names = append(names, agent.Name)
	}
	return names, nil
}

func (g *Global) resolveAgents(agentNames []string, requireDetected bool) ([]AgentInfo, error) {
	if len(agentNames) > 0 {
		names := normalizeAgentNames(agentNames)
		agents := make([]AgentInfo, 0, len(names))
		for _, name := range names {
			agent := ResolveAgent(name)
			if agent.SkillsDir == "" || agent.Reason == "unknown_agent" {
				return nil, errors.NewWithCodef("resolveAgents", errors.ErrInvalidInput, "未知 agent: %s", name)
			}
			if !agent.Detected && requireDetected {
				return nil, errors.NewWithCodef("resolveAgents", errors.ErrInvalidInput, "未检测到 agent: %s", name)
			}
			agents = append(agents, agent)
		}
		return agents, nil
	}

	agents := DetectAgents()
	if requireDetected && len(agents) == 0 {
		return nil, errors.NewWithCode("resolveAgents", errors.ErrInvalidInput, "未检测到可用 agent，请使用 --agent 指定")
	}
	return agents, nil
}

func DetectAgents() []AgentInfo {
	candidates := []string{"codex", "opencode", "claude"}
	var agents []AgentInfo
	for _, name := range candidates {
		agent := ResolveAgent(name)
		if agent.Detected || agent.Configured {
			agents = append(agents, agent)
		}
	}
	return agents
}

func ResolveAgent(name string) AgentInfo {
	normalized := normalizeAgentName(name)
	homeDir, _ := os.UserHomeDir()
	info := AgentInfo{Name: normalized}

	switch normalized {
	case "codex":
		if dir := os.Getenv("CODEX_SKILLS_DIR"); dir != "" {
			info.SkillsDir = dir
			info.Detected = true
			info.Reason = "CODEX_SKILLS_DIR"
		} else {
			base := os.Getenv("CODEX_HOME")
			if base == "" && homeDir != "" {
				base = filepath.Join(homeDir, ".codex")
			}
			info.SkillsDir = filepath.Join(base, "skills")
			info.Detected = pathExists(base) || commandExists("codex")
		}
	case "opencode":
		if dir := os.Getenv("OPENCODE_SKILLS_DIR"); dir != "" {
			info.SkillsDir = dir
			info.Detected = true
			info.Reason = "OPENCODE_SKILLS_DIR"
		} else {
			base := os.Getenv("OPENCODE_HOME")
			if base == "" && homeDir != "" {
				base = filepath.Join(homeDir, ".config", "opencode")
			}
			info.SkillsDir = filepath.Join(base, "skills")
			info.Detected = pathExists(base) || commandExists("opencode")
		}
	case "claude":
		if dir := os.Getenv("CLAUDE_SKILLS_DIR"); dir != "" {
			info.SkillsDir = dir
			info.Detected = true
			info.Reason = "CLAUDE_SKILLS_DIR"
		} else if homeDir != "" {
			info.SkillsDir = filepath.Join(homeDir, ".claude", "skills")
			info.Detected = pathExists(filepath.Join(homeDir, ".claude")) || commandExists("claude")
		}
	default:
		info.SkillsDir = ""
		info.Detected = false
		info.Reason = "unknown_agent"
	}

	info.Configured = info.SkillsDir != "" && pathExists(info.SkillsDir)
	if info.Reason == "" {
		switch {
		case info.Configured:
			info.Reason = "skills_dir"
		case info.Detected:
			info.Reason = "agent_detected"
		default:
			info.Reason = "not_detected"
		}
	}
	return info
}

func loadState() (*State, error) {
	path, err := statePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &State{EnabledSkills: map[string]SkillState{}}, nil
	}
	if err != nil {
		return nil, errors.WrapWithCode(err, "loadGlobalState", errors.ErrFileOperation, "读取全局状态失败")
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return &State{EnabledSkills: map[string]SkillState{}}, nil
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, errors.WrapWithCode(err, "loadGlobalState", errors.ErrConfigInvalid, "解析全局状态失败")
	}
	if state.EnabledSkills == nil {
		state.EnabledSkills = map[string]SkillState{}
	}
	return &state, nil
}

func saveState(state *State) error {
	path, err := statePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return errors.WrapWithCode(err, "saveGlobalState", errors.ErrFileOperation, "创建全局状态目录失败")
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.Wrap(err, "saveGlobalState: 序列化全局状态失败")
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func statePath() (string, error) {
	rootDir, err := config.GetRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootDir, "global-state.json"), nil
}

func globalSkillsDir() (string, error) {
	rootDir, err := config.GetRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootDir, "global", "skills"), nil
}

func readManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

func writeManifest(path string, manifest Manifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func isManagedManifest(manifest *Manifest, skillID, agentName string) bool {
	return manifest != nil &&
		manifest.ManagedBy == "skill-hub" &&
		manifest.Scope == "global" &&
		manifest.SkillID == skillID &&
		manifest.Agent == agentName
}

func scanOrphanedManagedSkills(state *State, agents []AgentInfo) ([]StatusItem, error) {
	var orphaned []StatusItem
	for _, agent := range agents {
		if !agent.Configured {
			continue
		}
		err := filepath.Walk(agent.SkillsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() || info.Name() != ManifestFileName {
				return nil
			}
			manifest, err := readManifest(path)
			if err != nil || manifest.ManagedBy != "skill-hub" || manifest.Scope != "global" {
				return nil
			}
			skillState, ok := state.EnabledSkills[manifest.SkillID]
			if ok && containsString(skillState.Agents, agent.Name) {
				return nil
			}
			targetDir := filepath.Dir(path)
			orphaned = append(orphaned, StatusItem{
				SkillID:          manifest.SkillID,
				Agent:            agent.Name,
				Status:           StatusOrphaned,
				Message:          "agent 目录存在 Skill-Hub manifest，但全局状态不再记录",
				SourceRepository: manifest.SourceRepository,
				SourceHash:       manifest.SourceHash,
				AppliedHash:      manifest.AppliedHash,
				TargetPath:       targetDir,
			})
			return nil
		})
		if err != nil {
			return orphaned, err
		}
	}
	sort.Slice(orphaned, func(i, j int) bool {
		if orphaned[i].Agent == orphaned[j].Agent {
			return orphaned[i].SkillID < orphaned[j].SkillID
		}
		return orphaned[i].Agent < orphaned[j].Agent
	})
	return orphaned, nil
}

func removeFromAgent(skillState SkillState, agent AgentInfo, force bool) StatusItem {
	targetDir := filepath.Join(agent.SkillsDir, filepath.FromSlash(skillState.SkillID))
	item := StatusItem{
		SkillID:          skillState.SkillID,
		Agent:            agent.Name,
		SourceRepository: skillState.SourceRepository,
		Version:          skillState.Version,
		TargetPath:       targetDir,
	}

	if !agent.Configured {
		item.Status = StatusMissingAgentDir
		item.Message = "agent 全局 skills 目录不存在"
		return item
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		item.Status = StatusNotApplied
		item.Message = "目标目录不存在，仅清理全局状态"
		return item
	} else if err != nil {
		item.Status = StatusError
		item.Message = err.Error()
		return item
	}

	manifest, err := readManifest(filepath.Join(targetDir, ManifestFileName))
	if err != nil || !isManagedManifest(manifest, skillState.SkillID, agent.Name) {
		if !force {
			item.Status = StatusConflict
			item.Message = "目标目录不是 Skill-Hub 托管目录；使用 --force 删除"
			return item
		}
	}

	if err := os.RemoveAll(targetDir); err != nil {
		item.Status = StatusError
		item.Message = err.Error()
		return item
	}
	item.Status = StatusRemoved
	return item
}

func syncDirectory(srcDir, dstDir, preserveRelPath string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return errors.WrapWithCode(err, "syncDirectory", errors.ErrFileOperation, "创建目标目录失败")
	}

	srcFiles := make(map[string]bool)
	if err := filepath.Walk(srcDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(srcDir, srcPath)
		if err != nil {
			return err
		}
		srcFiles[relPath] = true
		return nil
	}); err != nil {
		return errors.Wrap(err, "syncDirectory: 遍历源目录失败")
	}

	dstFiles := make(map[string]bool)
	if err := filepath.Walk(dstDir, func(dstPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(dstDir, dstPath)
		if err != nil {
			return err
		}
		if preserveRelPath != "" && relPath == preserveRelPath {
			return nil
		}
		dstFiles[relPath] = true
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "syncDirectory: 遍历目标目录失败")
	}

	for relPath := range srcFiles {
		srcPath := filepath.Join(srcDir, relPath)
		dstPath := filepath.Join(dstDir, relPath)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return errors.Wrapf(err, "syncDirectory: 创建目录失败 %s", filepath.Dir(dstPath))
		}
		content, err := os.ReadFile(srcPath)
		if err != nil {
			return errors.Wrap(err, "syncDirectory: 读取源文件失败")
		}
		info, err := os.Stat(srcPath)
		if err != nil {
			return errors.Wrap(err, "syncDirectory: 获取源文件权限失败")
		}
		if err := os.WriteFile(dstPath, content, info.Mode()); err != nil {
			return errors.Wrap(err, "syncDirectory: 写入目标文件失败")
		}
		delete(dstFiles, relPath)
	}

	for relPath := range dstFiles {
		if err := os.Remove(filepath.Join(dstDir, relPath)); err != nil {
			return errors.Wrap(err, "syncDirectory: 删除目标多余文件失败")
		}
	}

	return nil
}

func hashDirectory(dir string) (string, error) {
	var files []string
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || info.IsDir() || info.Name() == ManifestFileName {
			return nil
		}
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	}); err != nil {
		return "", err
	}
	sort.Strings(files)

	h := sha256.New()
	for _, relPath := range files {
		normalized := filepath.ToSlash(relPath)
		_, _ = io.WriteString(h, normalized)
		_, _ = h.Write([]byte{0})
		content, err := os.ReadFile(filepath.Join(dir, relPath))
		if err != nil {
			return "", err
		}
		_, _ = h.Write(content)
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

func backupAndRemove(path string) error {
	parent := filepath.Dir(path)
	base := filepath.Base(path)
	backupPath := filepath.Join(parent, fmt.Sprintf("%s.skill-hub-backup.%s", base, time.Now().Format("20060102150405")))
	if err := os.Rename(path, backupPath); err != nil {
		return err
	}
	return nil
}

func normalizeAgentNames(agentNames []string) []string {
	set := make(map[string]struct{}, len(agentNames))
	for _, name := range agentNames {
		normalized := normalizeAgentName(name)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	return sortedKeys(set)
}

func normalizeAgentName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "codex":
		return "codex"
	case "opencode", "open_code":
		return "opencode"
	case "claude", "claude_code":
		return "claude"
	default:
		return strings.ToLower(strings.TrimSpace(name))
	}
}

func filterDesiredAgents(desiredAgents, filters []string) []string {
	desired := normalizeAgentNames(desiredAgents)
	if len(filters) == 0 {
		return desired
	}
	filterSet := stringSet(normalizeAgentNames(filters))
	var result []string
	for _, agent := range desired {
		if _, ok := filterSet[agent]; ok {
			result = append(result, agent)
		}
	}
	return result
}

func selectEnabledSkillIDs(state *State, skillID, operation string) ([]string, error) {
	if skillID != "" {
		if _, ok := state.EnabledSkills[skillID]; !ok {
			return nil, errors.SkillNotFound(operation, skillID)
		}
		return []string{skillID}, nil
	}

	skillIDs := make([]string, 0, len(state.EnabledSkills))
	for id := range state.EnabledSkills {
		skillIDs = append(skillIDs, id)
	}
	sort.Strings(skillIDs)
	return skillIDs, nil
}

func mergeStrings(existing, incoming []string) []string {
	set := stringSet(existing)
	for _, value := range incoming {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	return sortedKeys(set)
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	return set
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func pathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
