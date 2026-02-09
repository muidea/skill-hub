package spec

// Skill 表示一个技能的完整定义
type Skill struct {
	ID            string        `yaml:"id" json:"id"`
	Name          string        `yaml:"name" json:"name"`
	Version       string        `yaml:"version" json:"version"`
	Author        string        `yaml:"author" json:"author"`
	Description   string        `yaml:"description" json:"description"`
	Tags          []string      `yaml:"tags" json:"tags"`
	Compatibility string        `yaml:"compatibility,omitempty" json:"compatibility,omitempty"`
	Variables     []Variable    `yaml:"variables" json:"variables"`
	Dependencies  []string      `yaml:"dependencies" json:"dependencies"`
	Claude        *ClaudeConfig `yaml:"claude,omitempty" json:"claude,omitempty"`
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
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Author        string   `json:"author"`
	Description   string   `json:"description"`
	Tags          []string `json:"tags"`
	Compatibility string   `json:"compatibility,omitempty"`
}

// Registry 表示技能仓库的索引
type Registry struct {
	Version string          `json:"version"`
	Skills  []SkillMetadata `json:"skills"`
}

// ProjectConfig 表示项目的配置信息（符合文档设计）
type ProjectConfig struct {
	PreferredTarget string            `json:"preferred_target,omitempty"` // cursor, claude_code, 或空
	EnabledSkills   []string          `json:"enabled_skills,omitempty"`   // 技能ID数组
	Vars            map[string]string `json:"vars,omitempty"`             // 项目级变量
	LastSync        string            `json:"last_sync,omitempty"`
}

// 目标类型常量
const (
	TargetCursor     = "cursor"
	TargetClaudeCode = "claude_code"
	TargetOpenCode   = "open_code" // OpenCode支持
	TargetClaude     = "claude"    // 向后兼容
	TargetUnknown    = "unknown"
	TargetAll        = "all"
)

// NormalizeTarget 规范化目标类型（处理向后兼容）
func NormalizeTarget(target string) string {
	if target == TargetClaude {
		return TargetClaudeCode
	}
	if target == "opencode" {
		return TargetOpenCode
	}
	return target
}

// ProjectState 表示项目与技能的关联状态（向后兼容）
type ProjectState struct {
	ProjectPath     string               `json:"project_path"`
	PreferredTarget string               `json:"preferred_target,omitempty"` // cursor, claude_code, 或空
	Skills          map[string]SkillVars `json:"skills"`
	LastSync        string               `json:"last_sync,omitempty"`
}

// 技能状态常量
const (
	SkillStatusSynced   = "Synced"   // 本地与仓库一致
	SkillStatusModified = "Modified" // 本地有未反馈的修改
	SkillStatusOutdated = "Outdated" // 仓库版本领先于本地
	SkillStatusMissing  = "Missing"  // 技能已启用但本地文件缺失
)

// SkillVars 表示项目中某个技能的变量配置和状态
type SkillVars struct {
	SkillID   string            `json:"skill_id"`
	Version   string            `json:"version"`
	Status    string            `json:"status,omitempty"` // 技能状态：Synced, Modified, Outdated, Missing
	Variables map[string]string `json:"variables"`
}

// CreateOptions 创建技能选项
type CreateOptions struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Compatibility string `json:"compatibility"` // cursor, claude, opencode, all
	OutputDir     string `json:"output_dir"`
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
