package cli

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	httpapibiz "github.com/muidea/skill-hub/internal/modules/application/httpapi/biz"
	"github.com/muidea/skill-hub/internal/pkg/configport"
	"github.com/muidea/skill-hub/internal/pkg/gitport"
	"github.com/muidea/skill-hub/internal/pkg/projectstateport"
	localruntime "github.com/muidea/skill-hub/internal/runtime/local"
	"github.com/muidea/skill-hub/pkg/errors"
	"github.com/muidea/skill-hub/pkg/spec"
	"github.com/muidea/skill-hub/pkg/utils"
)

// RunContext 命令运行上下文，包含 init + 可选 workspace + 项目状态 port 的公共结果。
type RunContext struct {
	Cwd          string
	ProjectState *spec.ProjectState
}

var localRuntime = localruntime.New()

func defaultRepository() (*configport.RepositoryConfig, error) {
	return localRuntime.Repository().Default()
}

func listRepositories(includeDisabled bool) ([]configport.RepositoryConfig, error) {
	return localRuntime.Repository().List(includeDisabled)
}

func getHubRootDir() (string, error) {
	return localRuntime.Config().RootDir()
}

func repositoryPath(repoName string) (string, error) {
	return localRuntime.Repository().Path(repoName)
}

func getRepoSkillDirPath(skillID string) (string, error) {
	defaultRepo, err := defaultRepository()
	if err != nil {
		return "", errors.Wrap(err, "getRepoSkillDirPath: 获取默认仓库失败")
	}
	repoPath, err := localRuntime.Repository().Path(defaultRepo.Name)
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
	return localRuntime.Repository().ListSkills(repoNames)
}

func rebuildRepositoryIndex(repoName string) error {
	return localRuntime.Repository().RebuildIndex(repoName)
}

func archiveToDefaultRepository(skillID, sourcePath string) error {
	return localRuntime.Repository().Archive(skillID, sourcePath)
}

func addRepository(repoConfig configport.RepositoryConfig) error {
	return localRuntime.Repository().Add(repoConfig)
}

func removeRepository(name string) error {
	return localRuntime.Repository().Remove(name)
}

func syncRepository(name string) (*httpapibiz.RepoSyncData, error) {
	result, err := localRuntime.Repository().Sync(name)
	if err != nil {
		return nil, err
	}
	return &httpapibiz.RepoSyncData{
		Status:  result.Status,
		Message: result.Message,
		Ahead:   result.Ahead,
		Behind:  result.Behind,
	}, nil
}

func enableRepository(name string) error {
	return localRuntime.Repository().Enable(name)
}

func disableRepository(name string) error {
	return localRuntime.Repository().Disable(name)
}

func getRepository(name string) (*configport.RepositoryConfig, error) {
	return localRuntime.Repository().Get(name)
}

func setDefaultRepository(name string) error {
	return localRuntime.Repository().SetDefault(name)
}

func updateRepositoryURL(name, url string) error {
	return localRuntime.Repository().UpdateURL(name, url)
}

func cleanupTimestampedBackupDirs(basePath string) error {
	return localRuntime.Maintenance().CleanupTimestampedBackupDirs(basePath)
}

func checkSkillRepositoryUpdates() (*gitport.RemoteUpdateStatus, error) {
	return localRuntime.Git().CheckSkillRepositoryUpdates()
}

func skillRepositoryStatus() (string, error) {
	return localRuntime.Git().SkillRepositoryStatus()
}

func pushSkillRepositoryChanges(message string) error {
	return localRuntime.Git().PushSkillRepositoryChanges(message)
}

func pushSkillRepositoryCommits() error {
	return localRuntime.Git().PushSkillRepositoryCommits()
}

func setSkillRepositoryRemote(url string) error {
	return localRuntime.Git().SetSkillRepositoryRemote(url)
}

// RequireInitAndWorkspace 执行 CheckInitDependency 与 EnsureProjectWorkspace，返回 RunContext。
func RequireInitAndWorkspace(cwd string) (*RunContext, error) {
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
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return nil, errors.Wrap(err, "获取当前目录绝对路径失败")
	}
	cwd = absCwd
	projectState, err := EnsureProjectWorkspace(cwd)
	if err != nil {
		return nil, err
	}
	projectRoot := cwd
	if projectState != nil && projectState.ProjectPath != "" {
		projectRoot = projectState.ProjectPath
	}
	return &RunContext{Cwd: projectRoot, ProjectState: projectState}, nil
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
	err := localRuntime.Config().Initialized()
	if err != nil {
		return errors.NewWithCode("CheckInitDependency", errors.ErrConfigNotFound, "本地仓库未初始化，请先运行 'skill-hub init'")
	}
	return nil
}

// CheckProjectWorkspace 检查项目工作区状态
// 符合规范要求：检查当前目录是否存在于state.json中
func CheckProjectWorkspace(cwd string) (*spec.ProjectState, error) {
	projectState, err := localRuntime.ProjectState().Load(cwd)
	if err != nil {
		return nil, errors.WrapWithCode(err, "CheckProjectWorkspace", errors.ErrSystem, "加载项目状态失败")
	}

	return projectState, nil
}

// EnsureProjectWorkspace 确保项目工作区存在
// 符合规范要求：如果当前目录不存在于state.json中，则提示是否需要新建项目工作区
func EnsureProjectWorkspace(cwd string) (*spec.ProjectState, error) {
	// 检查项目是否真正存在于状态文件中
	projectState, err := localRuntime.ProjectState().Find(cwd)
	if err != nil {
		return nil, errors.WrapWithCode(err, "EnsureProjectWorkspace", errors.ErrSystem, "查找项目失败：请检查 ~/.skill-hub/state.json 是否为有效 JSON")
	}
	// .agents/skills 是项目技能工作区的明确边界。若当前目录尚未登记、
	// 但 Find 命中了父项目，不能把当前目录的独立技能目录误归属给父项目。
	if shouldCreateNestedWorkspace(cwd, projectState) {
		projectState = nil
	}

	// 如果项目不存在于状态文件中，需要初始化
	if projectState == nil {
		if nonInteractiveMode() {
			command := activeOutput.command
			switch command.Name() {
			case "register", "use", "import", "create":
				// Previews and archive-only imports must not initialize state.
				if mustGetBoolFlag(command, "dry-run") || mustGetBoolFlag(command, "archive-only") {
					return &spec.ProjectState{ProjectPath: cwd, Skills: map[string]spec.SkillVars{}}, nil
				}
				return createNewProjectWorkspace(cwd, localRuntime.ProjectState())
			default:
				return nil, errors.ProjectNotFound("EnsureProjectWorkspace", cwd)
			}
		}
		printfText("当前目录 '%s' 未在skill-hub中注册\n", filepath.Base(cwd))
		printText("是否创建新的项目工作区？ [Y/n]: ")

		reader := bufio.NewReader(inputSource())
		response, _ := readInputLine(reader)
		response = strings.TrimSpace(response)

		if response == "" || strings.ToLower(response) == "y" {
			// 创建项目工作区
			return createNewProjectWorkspace(cwd, localRuntime.ProjectState())
		} else {
			return nil, errors.NewWithCode("EnsureProjectWorkspace", errors.ErrUserCancel, "操作取消")
		}
	}

	return projectState, nil
}

func shouldCreateNestedWorkspace(cwd string, projectState *spec.ProjectState) bool {
	if projectState == nil || projectState.ProjectPath == "" {
		return false
	}
	currentPath, err := filepath.Abs(cwd)
	if err != nil || projectState.ProjectPath == currentPath {
		return false
	}
	info, err := os.Stat(filepath.Join(currentPath, ".agents", "skills"))
	return err == nil && info.IsDir()
}

// createNewProjectWorkspace 创建新的项目工作区。
func createNewProjectWorkspace(cwd string, projectStatePort projectstateport.ProjectState) (*spec.ProjectState, error) {
	printlnText("正在创建新的项目工作区...")

	if err := initializeWorkspaceFiles(cwd); err != nil {
		return nil, errors.WrapWithCode(err, "createNewProjectWorkspace", errors.ErrFileOperation, "初始化工作区文件失败")
	}

	// 创建项目状态
	projectState := &spec.ProjectState{
		ProjectPath: cwd,
		Skills:      make(map[string]spec.SkillVars),
	}

	// 保存项目状态
	if err := projectStatePort.Save(*projectState); err != nil {
		return nil, errors.WrapWithCode(err, "createNewProjectWorkspace", errors.ErrSystem, "保存项目状态失败")
	}

	printlnText("✅ 已创建项目工作区")
	return projectState, nil
}

// initializeWorkspaceFiles 初始化标准 .agents/skills 工作区。
func initializeWorkspaceFiles(cwd string) error {
	agentsDir := filepath.Join(cwd, ".agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return errors.WrapWithCode(err, "initializeWorkspaceFiles", errors.ErrFileOperation, "创建.agents目录失败")
	}
	printfText("✓ 创建目录: %s\n", agentsDir)

	skillsDir := filepath.Join(agentsDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return errors.WrapWithCode(err, "initializeWorkspaceFiles", errors.ErrFileOperation, "创建skills目录失败")
	}
	printfText("✓ 创建目录: %s\n", skillsDir)

	return nil
}

// CheckSkillExists 检查技能是否存在
func CheckSkillExists(skillID string) error {
	// 检查init依赖
	if err := CheckInitDependency(); err != nil {
		return err
	}

	// 在所有仓库中查找技能
	skills, err := localRuntime.Repository().FindSkill(skillID)
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
	return localRuntime.ProjectState().HasSkill(cwd, skillID)
}
