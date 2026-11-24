package spec

// Skill 表示一个技能的完整定义
type Skill struct {
	ID           string        `yaml:"id" json:"id"`
	Name         string        `yaml:"name" json:"name"`
	Version      string        `yaml:"version" json:"version"`
	Author       string        `yaml:"author" json:"author"`
	Description  string        `yaml:"description" json:"description"`
	Tags         []string      `yaml:"tags" json:"tags"`
	Variables    []Variable    `yaml:"variables" json:"variables"`
	Dependencies []string      `yaml:"dependencies" json:"dependencies"`
	Claude       *ClaudeConfig `yaml:"claude,omitempty" json:"claude,omitempty"`

	// 多仓库扩展字段
	Repository       string `yaml:"repository,omitempty" json:"repository,omitempty"`               // 源仓库名称
	RepositoryPath   string `yaml:"repository_path,omitempty" json:"repository_path,omitempty"`     // 仓库内路径
	RepositoryCommit string `yaml:"repository_commit,omitempty" json:"repository_commit,omitempty"` // 仓库提交哈希
}

// ClaudeConfig Claude专项配置
type ClaudeConfig struct {
	Mode       string    `yaml:"mode,omitempty" json:"mode,omitempty"` // instruction | tool
	Runtime    string    `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Entrypoint string    `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	ToolSpec   *ToolSpec `yaml:"tool_spec,omitempty" json:"tool_spec,omitempty"`
}

// ToolSpec 工具定义规范
type ToolSpec struct {
	Name        string                 `yaml:"name" json:"name"`
	Description string                 `yaml:"description" json:"description"`
	InputSchema map[string]interface{} `yaml:"input_schema" json:"input_schema"`
}

// Variable 表示技能模板中的变量
type Variable struct {
	Name        string `yaml:"name" json:"name"`
	Default     string `yaml:"default" json:"default"`
	Description string `yaml:"description" json:"description"`
}

// SkillMetadata 用于技能索引的简化信息
type SkillMetadata struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Version          string   `json:"version"`
	Author           string   `json:"author"`
	Description      string   `json:"description"`
	Tags             []string `json:"tags"`
	Repository       string   `json:"repository,omitempty"`        // 源仓库名称
	RepositoryPath   string   `json:"repository_path,omitempty"`   // 仓库内路径
	RepositoryCommit string   `json:"repository_commit,omitempty"` // 仓库提交哈希
}

// Registry 表示技能仓库的索引
type Registry struct {
	Version string          `json:"version"`
	Skills  []SkillMetadata `json:"skills"`
}

// ProjectConfig 表示项目的配置信息（符合文档设计）
type ProjectConfig struct {
	PreferredTarget string            `json:"preferred_target,omitempty"` // 历史兼容字段，不参与业务逻辑
	EnabledSkills   []string          `json:"enabled_skills,omitempty"`   // 技能ID数组
	Vars            map[string]string `json:"vars,omitempty"`             // 项目级变量
	LastSync        string            `json:"last_sync,omitempty"`
}

// 目标类型常量
const (
	TargetCursor     = "cursor"
	TargetClaudeCode = "claude_code"
	TargetOpenCode   = "open_code"
	TargetAgy        = "agy"
)

// ProjectState 表示项目与技能的关联状态（向后兼容）
type ProjectState struct {
	ProjectPath     string               `json:"project_path"`
	PreferredTarget string               `json:"preferred_target,omitempty"` // 历史兼容字段，不参与业务逻辑
	Skills          map[string]SkillVars `json:"skills"`
	LastSync        string               `json:"last_sync,omitempty"`
}

// 技能检查状态常量。项目与全局 status 共用这些值。
const (
	StatusSynced      = "synced"      // 目标副本与来源一致
	StatusModified    = "modified"    // 目标副本被本地修改
	StatusOutdated    = "outdated"    // 来源已更新，目标需要刷新
	StatusDiverged    = "diverged"    // 来源已更新且目标也有修改，需人工合并
	StatusMissing     = "missing"     // 期望存在的目标副本不存在
	StatusConflict    = "conflict"    // 存在同名但不可安全接管的目标
	StatusOrphaned    = "orphaned"    // 存在受管副本，但状态记录已不存在
	StatusUnavailable = "unavailable" // 目标环境或来源不可用
	StatusMixed       = "mixed"       // 聚合行中存在不同目标状态

	// 以下名称保留给内部调用方；值已经统一为规范状态。
	SkillStatusSynced                      = StatusSynced
	SkillStatusModified                    = StatusModified
	SkillStatusOutdated                    = StatusOutdated
	SkillStatusMissing                     = StatusMissing
	SkillStatusModifiedAgainstOutdatedRepo = StatusDiverged
)

// NormalizeSkillStatus upgrades persisted pre-unification project state values
// to the canonical status vocabulary.
func NormalizeSkillStatus(status string) string {
	switch status {
	case "Synced", StatusSynced, "ok":
		return StatusSynced
	case "Modified", StatusModified:
		return StatusModified
	case "Outdated", StatusOutdated, "stale":
		return StatusOutdated
	case "ModifiedAgainstOutdatedRepo", StatusDiverged:
		return StatusDiverged
	case "Missing", StatusMissing, "not_applied":
		return StatusMissing
	case StatusConflict:
		return StatusConflict
	case StatusOrphaned:
		return StatusOrphaned
	case StatusUnavailable, "missing_agent_dir", "error":
		return StatusUnavailable
	default:
		return status
	}
}

// LegacyProjectSkillStatus returns the pre-unification project status name for
// temporary JSON compatibility. Empty means the status had no project legacy name.
func LegacyProjectSkillStatus(status string) string {
	switch NormalizeSkillStatus(status) {
	case StatusSynced:
		return "Synced"
	case StatusModified:
		return "Modified"
	case StatusOutdated:
		return "Outdated"
	case StatusDiverged:
		return "ModifiedAgainstOutdatedRepo"
	case StatusMissing:
		return "Missing"
	default:
		return ""
	}
}

// SkillVars 表示项目中某个技能的变量配置和状态
type SkillVars struct {
	SkillID          string            `json:"skill_id"`
	Version          string            `json:"version"`
	Status           string            `json:"status,omitempty"`            // 规范技能状态：synced、modified、outdated、diverged、missing
	SourceRepository string            `json:"source_repository,omitempty"` // 首次 use 时选中的来源仓库
	AppliedHash      string            `json:"applied_hash,omitempty"`      // 最近一次 apply 后的技能目录内容指纹
	AppliedCommit    string            `json:"applied_commit,omitempty"`    // 最近一次 apply 时来源仓库提交，预留字段
	Variables        map[string]string `json:"variables"`
}

// CreateOptions 创建技能选项
type CreateOptions struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	OutputDir   string `json:"output_dir"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	SkillID  string   `json:"skill_id"`
	IsValid  bool     `json:"is_valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ArchiveInfo 归档信息
type ArchiveInfo struct {
	SkillID    string `json:"skill_id"`
	Version    string `json:"version"`
	ArchivedAt string `json:"archived_at"`
}

// ConflictResolution 冲突解决记录
type ConflictResolution struct {
	SkillID    string `json:"skill_id"`
	Repository string `json:"selected_repo"` // 用户选择的仓库
	Timestamp  string `json:"timestamp"`
	UserChoice bool   `json:"user_choice"` // 是否为用户选择
}

// Conflict 冲突检测结果
type Conflict struct {
	SkillID      string         `json:"skill_id"`
	SkillName    string         `json:"skill_name"`
	Repositories []ConflictRepo `json:"repositories"` // 包含此技能的仓库列表
}

// ConflictRepo 冲突仓库信息
type ConflictRepo struct {
	Repository string `json:"repository"` // 仓库名称
	Version    string `json:"version"`    // 技能版本
	Commit     string `json:"commit"`     // 提交哈希
}
