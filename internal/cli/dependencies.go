package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	adapterpkg "github.com/muidea/skill-hub/internal/adapter"
	"github.com/muidea/skill-hub/internal/config"
	gitpkg "github.com/muidea/skill-hub/internal/git"
	runtimemodule "github.com/muidea/skill-hub/internal/modules/kernel/runtime"
	"github.com/muidea/skill-hub/internal/multirepo"
	"github.com/muidea/skill-hub/internal/state"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/muidea/skill-hub/pkg/utils"
)

// RunContext 命令运行上下文，包含 init + 可选 workspace + StateManager 的公共结果
type RunContext struct {
	Cwd          string
	ProjectState *spec.ProjectState
	StateManager *state.StateManager
}

var runtimeSvc = runtimemodule.New().Service()

func loadHubConfig() (*config.Config, error) {
	return runtimeSvc.Config()
}

func defaultRepository() (*config.RepositoryConfig, error) {
	return runtimeSvc.DefaultRepository()
}

func listRepositories(includeDisabled bool) ([]config.RepositoryConfig, error) {
	return runtimeSvc.ListRepositories(includeDisabled)
}

func getHubRootDir() (string, error) {
	return runtimeSvc.RootDir()
}

func repositoryPath(repoName string) (string, error) {
	return runtimeSvc.RepositoryPath(repoName)
}

func newStateManager() (*state.StateManager, error) {
	return runtimeSvc.StateManager()
}

func newRepositoryManager() (*multirepo.Manager, error) {
	return runtimeSvc.RepositoryManager()
}

func getTargetAdapter(target string) (adapterpkg.Adapter, error) {
	return runtimeSvc.Adapter(target)
}

func readDefaultRepositorySkillContent(skillID string) (string, error) {
	return runtimeSvc.ReadDefaultRepositorySkillContent(skillID)
}

func readRepositorySkillContent(repoName, skillID string) (string, error) {
	repoPath, err := repositoryPath(repoName)
	if err != nil {
		return "", errors.Wrap(err, "readRepositorySkillContent: 获取仓库路径失败")
	}

	skillPath := filepath.Join(repoPath, "skills", skillID, "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		return "", errors.Wrap(err, "readRepositorySkillContent: 读取技能文件失败")
	}
	return string(content), nil
}

func getRepoSkillDirPath(skillID string) (string, error) {
	defaultRepo, err := defaultRepository()
	if err != nil {
		return "", errors.Wrap(err, "getRepoSkillDirPath: 获取默认仓库失败")
	}
	repoPath, err := repositoryPath(defaultRepo.Name)
	if err != nil {
		return "", errors.Wrap(err, "getRepoSkillDirPath: 获取仓库路径失败")
	}
	repoSkillDir := filepath.Join(repoPath, "skills", skillID)
	if _, err := os.Stat(repoSkillDir); os.IsNotExist(err) {
		return "", errors.NewWithCode("getRepoSkillDirPath", errors.ErrSkillNotFound, "技能在仓库中不存在")
	}
	return repoSkillDir, nil
}

func listSkillMetadata(repoNames []string) ([]spec.SkillMetadata, error) {
	return runtimeSvc.ListSkillMetadata(repoNames)
}

func rebuildRepositoryIndex(repoName string) error {
	return runtimeSvc.RebuildRepositoryIndex(repoName)
}

func archiveToDefaultRepository(skillID, sourcePath string) error {
	return runtimeSvc.ArchiveToDefaultRepository(skillID, sourcePath)
}

func addRepository(repoConfig config.RepositoryConfig) error {
	return runtimeSvc.AddRepository(repoConfig)
}

func removeRepository(name string) error {
	return runtimeSvc.RemoveRepository(name)
}

func syncRepository(name string) error {
	return runtimeSvc.SyncRepository(name)
}

func enableRepository(name string) error {
	return runtimeSvc.EnableRepository(name)
}

func disableRepository(name string) error {
	return runtimeSvc.DisableRepository(name)
}

func getRepository(name string) (*config.RepositoryConfig, error) {
	return runtimeSvc.GetRepository(name)
}

func setDefaultRepository(name string) error {
	return runtimeSvc.SetDefaultRepository(name)
}

func updateRepositoryURL(name, url string) error {
	return runtimeSvc.UpdateRepositoryURL(name, url)
}

func newGitRepository(repoPath string) (*gitpkg.Repository, error) {
	return runtimeSvc.GitRepository(repoPath)
}

func newSkillRepository() (*gitpkg.SkillRepository, error) {
	return runtimeSvc.SkillRepository()
}

func cleanupTimestampedBackupDirs(basePath string) error {
	return runtimeSvc.CleanupTimestampedBackupDirs(basePath)
}

func syncSkillRepositoryAndRefresh() error {
	return runtimeSvc.SyncSkillRepositoryAndRefresh()
}

func checkSkillRepositoryUpdates() (*gitpkg.RemoteUpdateStatus, error) {
	return runtimeSvc.CheckSkillRepositoryUpdates()
}

func skillRepositoryStatus() (string, error) {
	return runtimeSvc.SkillRepositoryStatus()
}

func pushSkillRepositoryChanges(message string) error {
	return runtimeSvc.PushSkillRepositoryChanges(message)
}

func pushSkillRepositoryCommits() error {
	return runtimeSvc.PushSkillRepositoryCommits()
}

func setSkillRepositoryRemote(url string) error {
	return runtimeSvc.SetSkillRepositoryRemote(url)
}

// RequireInitAndWorkspace 执行 CheckInitDependency、EnsureProjectWorkspace 并创建 StateManager，返回 RunContext
func RequireInitAndWorkspace(cwd, target string) (*RunContext, error) {
	if err := CheckInitDependency(); err != nil {
		return nil, err
	}
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return nil, utils.GetCwdErr(err)
		}
	}
	projectState, err := EnsureProjectWorkspace(cwd, target)
	if err != nil {
		return nil, err
	}
	stateManager, err := newStateManager()
	if err != nil {
		return nil, errors.WrapWithCode(err, "RequireInitAndWorkspace", errors.ErrSystem, "创建状态管理器失败")
	}
	return &RunContext{Cwd: cwd, ProjectState: projectState, StateManager: stateManager}, nil
}

// RequireInitOnly 仅执行 CheckInitDependency 并获取当前目录，不要求 workspace
func RequireInitOnly() (*RunContext, error) {
	if err := CheckInitDependency(); err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, utils.GetCwdErr(err)
	}
	return &RunContext{Cwd: cwd}, nil
}

// CheckInitDependency 检查init依赖，如果本地仓库不存在则返回错误
// 符合规范要求：所有命令（除init外）都需要检查本地仓库是否存在
func CheckInitDependency() error {
	// 尝试加载配置，如果失败说明未初始化
	_, err := loadHubConfig()
	if err != nil {
		return errors.NewWithCode("CheckInitDependency", errors.ErrConfigNotFound, "本地仓库未初始化，请先运行 'skill-hub init'")
	}
	return nil
}

// CheckProjectWorkspace 检查项目工作区状态
// 符合规范要求：检查当前目录是否存在于state.json中
func CheckProjectWorkspace(cwd string) (*spec.ProjectState, error) {
	stateManager, err := newStateManager()
	if err != nil {
		return nil, errors.WrapWithCode(err, "CheckProjectWorkspace", errors.ErrSystem, "创建状态管理器失败")
	}

	projectState, err := stateManager.LoadProjectState(cwd)
	if err != nil {
		return nil, errors.WrapWithCode(err, "CheckProjectWorkspace", errors.ErrSystem, "加载项目状态失败")
	}

	return projectState, nil
}

// EnsureProjectWorkspace 确保项目工作区存在
// 符合规范要求：如果当前目录不存在于state.json中，则提示是否需要新建项目工作区
func EnsureProjectWorkspace(cwd, target string) (*spec.ProjectState, error) {
	stateManager, err := newStateManager()
	if err != nil {
		return nil, errors.WrapWithCode(err, "EnsureProjectWorkspace", errors.ErrSystem, "创建状态管理器失败")
	}

	// 检查项目是否真正存在于状态文件中
	projectState, err := stateManager.FindProjectByPath(cwd)
	if err != nil {
		return nil, errors.WrapWithCode(err, "EnsureProjectWorkspace", errors.ErrSystem, "查找项目失败")
	}

	// 如果项目不存在于状态文件中，需要初始化
	if projectState == nil {
		fmt.Printf("当前目录 '%s' 未在skill-hub中注册\n", filepath.Base(cwd))
		fmt.Print("是否创建新的项目工作区？ [Y/n]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response == "" || strings.ToLower(response) == "y" {
			// 创建项目工作区
			return createNewProjectWorkspace(cwd, target, stateManager)
		} else {
			return nil, errors.NewWithCode("EnsureProjectWorkspace", errors.ErrUserCancel, "操作取消")
		}
	}

	return projectState, nil
}

// createNewProjectWorkspace 创建新的项目工作区
func createNewProjectWorkspace(cwd, target string, stateManager *state.StateManager) (*spec.ProjectState, error) {
	fmt.Println("正在创建新的项目工作区...")

	// 如果target为空，使用默认值open_code
	if target == "" {
		target = spec.TargetOpenCode
	}

	// 验证目标值
	normalizedTarget := spec.NormalizeTarget(target)
	if normalizedTarget != spec.TargetCursor && normalizedTarget != spec.TargetClaudeCode && normalizedTarget != spec.TargetOpenCode {
		return nil, errors.NewWithCode("createNewProjectWorkspace", errors.ErrInvalidInput,
			fmt.Sprintf("无效的兼容目标: %s，可用选项: cursor, claude, open_code", target))
	}

	// 根据target初始化对应的工作区文件和目录
	if err := initializeTargetFiles(cwd, normalizedTarget); err != nil {
		return nil, errors.WrapWithCode(err, "createNewProjectWorkspace", errors.ErrFileOperation, "初始化工作区文件失败")
	}

	// 创建项目状态
	projectState := &spec.ProjectState{
		ProjectPath:     cwd,
		PreferredTarget: normalizedTarget,
		Skills:          make(map[string]spec.SkillVars),
	}

	// 保存项目状态
	if err := stateManager.SaveProjectState(projectState); err != nil {
		return nil, errors.WrapWithCode(err, "createNewProjectWorkspace", errors.ErrSystem, "保存项目状态失败")
	}

	fmt.Printf("✅ 已创建项目工作区，兼容目标: %s\n", normalizedTarget)
	return projectState, nil
}

// initializeTargetFiles 根据兼容目标初始化对应的工作区文件和目录
func initializeTargetFiles(cwd, target string) error {
	switch target {
	case spec.TargetOpenCode:
		// 创建.agents目录结构
		agentsDir := filepath.Join(cwd, ".agents")
		if err := os.MkdirAll(agentsDir, 0755); err != nil {
			return errors.WrapWithCode(err, "initializeTargetFiles", errors.ErrFileOperation, "创建.agents目录失败")
		}
		fmt.Printf("✓ 创建目录: %s\n", agentsDir)

		skillsDir := filepath.Join(agentsDir, "skills")
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			return errors.WrapWithCode(err, "initializeTargetFiles", errors.ErrFileOperation, "创建skills目录失败")
		}
		fmt.Printf("✓ 创建目录: %s\n", skillsDir)

	case spec.TargetClaudeCode:
		// 创建.claude目录
		claudeDir := filepath.Join(cwd, ".claude")
		if err := os.MkdirAll(claudeDir, 0755); err != nil {
			return errors.WrapWithCode(err, "initializeTargetFiles", errors.ErrFileOperation, "创建.claude目录失败")
		}
		fmt.Printf("✓ 创建目录: %s\n", claudeDir)

		// 创建空的config.json
		configPath := filepath.Join(claudeDir, "config.json")
		configContent := `{
  "skills": {}
}`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return errors.WrapWithCode(err, "initializeTargetFiles", errors.ErrFileOperation, "创建config.json失败")
		}
		fmt.Printf("✓ 创建文件: %s\n", configPath)

	case spec.TargetCursor:
		// 创建.cursorrules文件
		cursorRulesPath := filepath.Join(cwd, ".cursorrules")
		cursorRulesContent := `# Cursor Rules
# This file is managed by skill-hub

# Available skills will be injected here`
		if err := os.WriteFile(cursorRulesPath, []byte(cursorRulesContent), 0644); err != nil {
			return errors.WrapWithCode(err, "initializeTargetFiles", errors.ErrFileOperation, "创建.cursorrules文件失败")
		}
		fmt.Printf("✓ 创建文件: %s\n", cursorRulesPath)

	default:
		return errors.NewWithCode("initializeTargetFiles", errors.ErrInvalidInput,
			fmt.Sprintf("不支持的目标环境: %s", target))
	}

	return nil
}

// CheckSkillExists 检查技能是否存在
func CheckSkillExists(skillID string) error {
	// 检查init依赖
	if err := CheckInitDependency(); err != nil {
		return err
	}

	// 创建多仓库管理器
	repoManager, err := newRepositoryManager()
	if err != nil {
		return errors.Wrap(err, "CheckSkillExists: 创建多仓库管理器失败")
	}

	// 在所有仓库中查找技能
	skills, err := repoManager.FindSkill(skillID)
	if err != nil {
		return errors.Wrap(err, "CheckSkillExists: 查找技能失败")
	}

	// 如果没有找到任何技能
	if len(skills) == 0 {
		return errors.SkillNotFound("CheckSkillExists", skillID)
	}

	return nil
}

// CheckSkillInProject 检查技能是否在项目中
func CheckSkillInProject(cwd, skillID string) (bool, error) {
	stateManager, err := newStateManager()
	if err != nil {
		return false, errors.Wrap(err, "CheckSkillInProject: 创建状态管理器失败")
	}

	return stateManager.ProjectHasSkill(cwd, skillID)
}
