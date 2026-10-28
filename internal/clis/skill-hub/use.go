package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
	pkgutils "github.com/muidea/skill-hub/pkg/utils"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <id> | --pattern <id-or-glob>... [flags]",
	Short: "使用技能",
	Long: `将技能标记为在当前项目或本机全局范围内使用。

直接传入 <id> 时按精确 ID 选择技能；--pattern 保留用于批量 glob 匹配。
若同一 ID 来自多个仓库，可用 --repo 精确选择来源；自动化环境可配合
--non-interactive 和 --json 使用。此命令只更新状态，不直接分发文件；请随后执行 apply。`,
	Args: validateUseArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		patterns, err := readPatternOrExactID(cmd, args)
		if err != nil {
			return err
		}
		if len(args) == 1 {
			patterns = []string{args[0]}
		}
		if len(patterns) == 0 {
			return errors.NewWithCode("use", errors.ErrInvalidInput, "缺少技能 ID；请使用 use <id> 或 --pattern '<id-or-glob>'")
		}
		global, _ := cmd.Flags().GetBool("global")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		jsonOutput, _ := cmd.Flags().GetBool("json")
		nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
		repository, _ := cmd.Flags().GetString("repo")
		variables, err := readUseVariables(cmd)
		if err != nil {
			return err
		}

		summary, runErr := runUseWithOptions(patterns, useOptions{
			Global:         global,
			Repository:     repository,
			DryRun:         dryRun,
			NonInteractive: nonInteractive || jsonOutput,
			Variables:      variables,
			ExactInput:     len(args) == 1,
		})
		if jsonOutput {
			if err := writeJSON(summary); err != nil {
				return err
			}
		} else {
			renderUseSummary(summary)
		}
		return runErr
	},
}

func init() {
	useCmd.Flags().Bool("global", false, "将技能标记为本机全局使用（默认处理全部检测/配置的 agent）")
	useCmd.Flags().StringArray("pattern", nil, "技能 ID 通配符（可重复）。请引用通配符避免 shell 展开。")
	useCmd.Flags().String("repo", "", "精确指定技能来源仓库")
	useCmd.Flags().Bool("dry-run", false, "仅解析并预览启用结果，不写入状态")

	useCmd.Flags().Bool("non-interactive", false, "禁止交互；歧义来源会报错，变量使用默认值或 --var")
	useCmd.Flags().StringArray("var", nil, "设置技能变量，格式 key=value，可重复使用")
}

// validateUseArgs accepts one exact skill ID or one or more --pattern values,
// but deliberately keeps glob patterns out of positional arguments.
func validateUseArgs(cmd *cobra.Command, args []string) error {
	patterns, err := readPatternOrExactID(cmd, args)
	if err != nil {
		return err
	}
	if len(args) == 0 && len(patterns) == 0 {
		return errors.NewWithCode("use", errors.ErrInvalidInput, "缺少技能 ID；请使用 use <id> 或 --pattern '<id-or-glob>'")
	}
	return nil
}

type useOptions struct {
	Global         bool
	Repository     string
	DryRun         bool
	NonInteractive bool
	Variables      map[string]string
	ExactInput     bool
}

// UseSummary is the stable command result shared by project and global use.
// JSON output intentionally contains no presentation-only fields.
type UseSummary struct {
	Scope   string    `json:"scope"`
	DryRun  bool      `json:"dry_run"`
	Total   int       `json:"total"`
	Enabled int       `json:"enabled"`
	Planned int       `json:"planned"`
	Skipped int       `json:"skipped"`
	Failed  int       `json:"failed"`
	Items   []UseItem `json:"items"`
}

type UseItem struct {
	Input          string `json:"input"`
	SkillID        string `json:"skill_id,omitempty"`
	Repository     string `json:"repository,omitempty"`
	Version        string `json:"version,omitempty"`
	Status         string `json:"status"`
	NeedsVariables bool   `json:"needs_variables,omitempty"`
	Error          string `json:"error,omitempty"`
}

func readUseVariables(cmd *cobra.Command) (map[string]string, error) {
	raw, _ := cmd.Flags().GetStringArray("var")
	if len(raw) == 0 {
		return nil, nil
	}
	variables := make(map[string]string, len(raw))
	for _, value := range raw {
		key, val, ok := strings.Cut(value, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, errors.NewWithCodef("use", errors.ErrInvalidInput, "无效的 --var %q；应使用 key=value", value)
		}
		key = strings.TrimSpace(key)
		if _, exists := variables[key]; exists {
			return nil, errors.NewWithCodef("use", errors.ErrInvalidInput, "变量 %q 被重复指定", key)
		}
		variables[key] = val
	}
	return variables, nil
}

// runUseByPatterns remains as a small compatibility seam for internal callers.
// Agent selection is intentionally ignored: CLI global operations now always
// target the configured/detected agent set.
func runUseByPatterns(patterns []string, _ []string, isGlobal bool) error {
	summary, err := runUseWithOptions(patterns, useOptions{Global: isGlobal})
	renderUseSummary(summary)
	return err
}

func runUseWithOptions(patterns []string, options useOptions) (*UseSummary, error) {
	summary := &UseSummary{Scope: "project", DryRun: options.DryRun, Items: []UseItem{}}
	if options.Global {
		summary.Scope = "global"
	}
	matchers, err := compilePatterns(patterns)
	if err != nil {
		return summary, err
	}

	matches, err := resolveSkillsByPatternsFromRepos(patterns, options.Repository)
	if err != nil {
		return summary, err
	}
	// Each literal gets its own direct scan fallback. A partly stale registry
	// must not make a later literal ID disappear merely because another pattern
	// happened to resolve from the index.
	for i, matcher := range matchers {
		if !matcher.IsLiteral() || hasMatchingSkill(matches, matcher, options.Repository) {
			continue
		}
		direct, directErr := resolveSingleLiteralSkill(patterns[i])
		if directErr != nil {
			return summary, directErr
		}
		matches = append(matches, filterSkillsByRepository(direct, options.Repository)...)
	}
	matches = uniqueSkillMetadata(matches)

	selectedIDs := make(map[string]struct{})
	var lastErr error
	for i, pattern := range patterns {
		candidates := matchingSkills(matches, matchers[i], options.Repository)
		item := UseItem{Input: pattern}
		if len(candidates) == 0 {
			if options.ExactInput {
				item.Status = "error"
				item.Error = errors.SkillNotFound("use", pattern).Error()
				lastErr = errors.SkillNotFound("use", pattern)
			} else {
				item.Status = "skipped"
			}
			summary.Items = append(summary.Items, item)
			continue
		}

		selectedSkills, selectErr := selectUseCandidates(candidates, options)
		if selectErr != nil {
			item.Status = "error"
			item.Error = selectErr.Error()
			summary.Items = append(summary.Items, item)
			lastErr = selectErr
			continue
		}
		for _, selected := range selectedSkills {
			item := UseItem{Input: pattern, SkillID: selected.ID, Repository: selected.Repository, Version: selected.Version}
			if _, alreadySelected := selectedIDs[selected.ID]; alreadySelected {
				item.Status = "skipped"
				summary.Items = append(summary.Items, item)
				continue
			}
			selectedIDs[selected.ID] = struct{}{}

			result, useErr := useSelectedSkill(selected, options)
			item.Version = result.Version
			item.NeedsVariables = result.NeedsVariables
			if useErr == errUseCancelled {
				item.Status = "skipped"
			} else if useErr != nil {
				item.Status = "error"
				item.Error = useErr.Error()
				lastErr = useErr
			} else if options.DryRun {
				item.Status = "planned"
			} else {
				item.Status = "enabled"
			}
			summary.Items = append(summary.Items, item)
		}
	}

	for _, item := range summary.Items {
		summary.Total++
		switch item.Status {
		case "enabled":
			summary.Enabled++
		case "planned":
			summary.Planned++
		case "skipped":
			summary.Skipped++
		case "error":
			summary.Failed++
		}
	}
	return summary, lastErr
}

type selectedUseSkill struct {
	Version        string
	NeedsVariables bool
}

// errUseCancelled is a normal interactive outcome. It is rendered as a
// skipped item so declining reconfiguration does not fail a batch command.
var errUseCancelled = fmt.Errorf("操作已取消")

func useSelectedSkill(selected spec.SkillMetadata, options useOptions) (selectedUseSkill, error) {
	fullSkill, err := loadUseSkillDetail(selected)
	if err != nil {
		return selectedUseSkill{}, err
	}
	result := selectedUseSkill{Version: fullSkill.Version, NeedsVariables: len(fullSkill.Variables) > 0}
	if options.DryRun {
		return result, nil
	}
	variables, err := resolveUseVariables(fullSkill, options)
	if err != nil {
		return result, err
	}
	if options.Global {
		return result, enableGlobalUse(selected, variables)
	}
	return result, enableProjectUse(selected, variables, options.NonInteractive)
}

func loadUseSkillDetail(selected spec.SkillMetadata) (*spec.Skill, error) {
	if client, ok := hubClientIfAvailable(); ok {
		fullSkill, err := client.GetSkillDetail(context.Background(), selected.ID, selected.Repository)
		if err != nil {
			return nil, errors.Wrap(err, "通过服务加载技能详情失败")
		}
		return fullSkill, nil
	}
	fullSkill, err := localRuntime.Repository().LoadSkill(selected.ID, selected.Repository)
	if err != nil {
		return nil, errors.Wrap(err, "加载技能详情失败")
	}
	return fullSkill, nil
}

func resolveUseVariables(fullSkill *spec.Skill, options useOptions) (map[string]string, error) {
	declared := make(map[string]spec.Variable, len(fullSkill.Variables))
	for _, variable := range fullSkill.Variables {
		declared[variable.Name] = variable
	}
	for key := range options.Variables {
		if _, ok := declared[key]; !ok {
			return nil, errors.NewWithCodef("use", errors.ErrInvalidInput, "技能 %s 未声明变量 %q", fullSkill.ID, key)
		}
	}
	if len(fullSkill.Variables) == 0 {
		return map[string]string{}, nil
	}
	values := make(map[string]string, len(fullSkill.Variables))
	if options.NonInteractive {
		for _, variable := range fullSkill.Variables {
			if value, ok := options.Variables[variable.Name]; ok {
				values[variable.Name] = value
			} else {
				values[variable.Name] = variable.Default
			}
		}
		return values, nil
	}

	reader := bufio.NewReader(inputSource())
	printlnText("请设置技能变量 (按Enter使用默认值):")
	for _, variable := range fullSkill.Variables {
		if value, ok := options.Variables[variable.Name]; ok {
			values[variable.Name] = value
			continue
		}
		printfText("%s [%s]: ", variable.Name, variable.Default)
		input, _ := readInputLine(reader)
		input = strings.TrimSpace(input)
		if input == "" {
			values[variable.Name] = variable.Default
		} else {
			values[variable.Name] = input
		}
	}
	return values, nil
}

func enableProjectUse(selected spec.SkillMetadata, variables map[string]string, nonInteractive bool) error {
	if client, ok := hubClientIfAvailable(); ok {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		if !nonInteractive {
			if status, statusErr := client.GetProjectStatus(context.Background(), cwd, selected.ID); statusErr == nil && status.Item != nil && len(status.Item.Items) > 0 && !confirmSkillReconfigure() {
				return errUseCancelled
			}
		}
		_, err = client.UseSkill(context.Background(), httpapibiz.UseSkillRequest{ProjectPath: cwd, SkillID: selected.ID, Repository: selected.Repository, Variables: variables})
		if err != nil {
			return errors.Wrap(err, "通过服务启用技能失败")
		}
		return nil
	}
	ctx, err := RequireInitAndWorkspace("")
	if err != nil {
		return err
	}
	if !nonInteractive {
		hasSkill, hasErr := localRuntime.ProjectState().HasSkill(ctx.Cwd, selected.ID)
		if hasErr != nil {
			return hasErr
		}
		if hasSkill && !confirmSkillReconfigure() {
			return errUseCancelled
		}
	}
	if _, err := localRuntime.Project().Enable(ctx.Cwd, selected.ID, selected.Repository, variables); err != nil {
		return errors.Wrap(err, "保存项目状态失败")
	}
	return nil
}

func enableGlobalUse(selected spec.SkillMetadata, variables map[string]string) error {
	if client, ok := hubClientIfAvailable(); ok {
		_, err := client.UseGlobalSkill(context.Background(), httpapibiz.UseGlobalSkillRequest{SkillID: selected.ID, Repository: selected.Repository, Variables: variables})
		if err != nil {
			return errors.Wrap(err, "通过服务启用全局技能失败")
		}
		return nil
	}
	if err := CheckInitDependency(); err != nil {
		return err
	}
	if _, err := localRuntime.Global().Enable(selected.ID, selected.Repository, nil, variables); err != nil {
		return errors.Wrap(err, "保存全局技能状态失败")
	}
	return nil
}

func resolveSkillsByPatternsFromRepos(patterns []string, repository string) ([]spec.SkillMetadata, error) {
	repositories := []string(nil)
	if repository != "" {
		repositories = []string{repository}
	}
	if client, ok := hubClientIfAvailable(); ok {
		matches, err := client.FindSkillsByPatterns(context.Background(), patterns, repositories)
		if err != nil {
			return nil, errors.Wrap(err, "通过服务按pattern查找技能失败")
		}
		return matches, nil
	}
	if err := CheckInitDependency(); err != nil {
		return nil, err
	}
	matches, err := localRuntime.Repository().FindSkills(patterns, repositories)
	if err != nil {
		return nil, errors.Wrap(err, "按pattern查找技能失败")
	}
	return matches, nil
}

func resolveSingleLiteralSkill(skillID string) ([]spec.SkillMetadata, error) {
	if err := CheckInitDependency(); err != nil {
		return nil, err
	}
	return localRuntime.Repository().FindSkill(skillID)
}

func hasMatchingSkill(skills []spec.SkillMetadata, matcher pkgutils.Matcher, repository string) bool {
	return len(matchingSkills(skills, matcher, repository)) > 0
}

func matchingSkills(skills []spec.SkillMetadata, matcher pkgutils.Matcher, repository string) []spec.SkillMetadata {
	filtered := make([]spec.SkillMetadata, 0)
	for _, candidate := range skills {
		if matcher.Match(candidate.ID) && (repository == "" || candidate.Repository == repository) {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func filterSkillsByRepository(skills []spec.SkillMetadata, repository string) []spec.SkillMetadata {
	if repository == "" {
		return skills
	}
	filtered := make([]spec.SkillMetadata, 0, len(skills))
	for _, candidate := range skills {
		if candidate.Repository == repository {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

func uniqueSkillMetadata(skills []spec.SkillMetadata) []spec.SkillMetadata {
	seen := make(map[string]struct{}, len(skills))
	unique := make([]spec.SkillMetadata, 0, len(skills))
	for _, candidate := range skills {
		key := candidate.Repository + "\x00" + candidate.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, candidate)
	}
	return unique
}

func selectUseCandidate(candidates []spec.SkillMetadata, options useOptions) (spec.SkillMetadata, error) {
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if options.NonInteractive {
		repositories := make([]string, 0, len(candidates))
		for _, candidate := range candidates {
			repositories = append(repositories, candidate.Repository)
		}
		return spec.SkillMetadata{}, errors.NewWithCodef("use", errors.ErrInvalidInput, "技能存在多个来源仓库，请通过 --repo 指定：%s", strings.Join(repositories, ", "))
	}
	return chooseSkillCandidate(candidates)
}

// selectUseCandidates enables every distinct matched skill ID. Repository
// ambiguity is resolved per ID, which retains batch-pattern behavior while
// making a selected source explicit and deterministic.
func selectUseCandidates(matches []spec.SkillMetadata, options useOptions) ([]spec.SkillMetadata, error) {
	byID := make(map[string][]spec.SkillMetadata, len(matches))
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		if _, ok := byID[match.ID]; !ok {
			ids = append(ids, match.ID)
		}
		byID[match.ID] = append(byID[match.ID], match)
	}
	sort.Strings(ids)
	selected := make([]spec.SkillMetadata, 0, len(ids))
	for _, id := range ids {
		candidate, err := selectUseCandidate(byID[id], options)
		if err != nil {
			return nil, err
		}
		selected = append(selected, candidate)
	}
	return selected, nil
}

func chooseSkillCandidate(skills []spec.SkillMetadata) (spec.SkillMetadata, error) {
	if len(skills) == 1 {
		return skills[0], nil
	}
	printfText("发现 %d 个同名技能，请选择要使用的技能:\n", len(skills))
	for i, skill := range skills {
		printfText("  %d. [%s] %s - %s\n", i+1, skill.Repository, skill.Name, skill.Description)
	}
	reader := bufio.NewReader(inputSource())
	printText("请选择 (输入编号): ")
	input, _ := readInputLine(reader)
	input = strings.TrimSpace(input)
	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(skills) {
		return spec.SkillMetadata{}, errors.NewWithCode("chooseSkillCandidate", errors.ErrInvalidInput, "无效的选择")
	}
	return skills[choice-1], nil
}

func confirmSkillReconfigure() bool {
	printlnText("⚠️  该技能已在当前项目启用")
	printText("是否重新配置变量？ [y/N]: ")
	reader := bufio.NewReader(inputSource())
	response, _ := readInputLine(reader)
	response = strings.TrimSpace(response)
	return response == "y" || response == "Y"
}

func renderUseSummary(summary *UseSummary) {
	if summary == nil {
		return
	}
	if summary.DryRun {
		printlnText("=== use 预览（dry-run） ===")
	} else {
		printlnText("=== use 结果 ===")
	}
	for _, item := range summary.Items {
		switch item.Status {
		case "enabled":
			printfText("✓ 已启用 %s [%s]", item.SkillID, item.Repository)
		case "planned":
			printfText("[计划] 将启用 %s [%s]", item.SkillID, item.Repository)
		case "skipped":
			printfText("- 已跳过 %s", item.Input)
		case "error":
			printfText("✗ %s", item.Input)
		}
		if item.Version != "" {
			printfText(" (v%s)", item.Version)
		}
		if item.NeedsVariables {
			printText("；包含可配置变量")
		}
		if item.Error != "" {
			printfText("：%s", item.Error)
		}
		printlnText()
	}
	printfText("范围: %s；总计: %d，已启用: %d，计划: %d，跳过: %d，失败: %d\n", summary.Scope, summary.Total, summary.Enabled, summary.Planned, summary.Skipped, summary.Failed)
}
