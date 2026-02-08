package adapter

// Adapter 定义所有适配器的统一接口
type Adapter interface {
	// Apply 应用技能到目标文件
	Apply(skillID string, content string, variables map[string]string) error

	// Extract 从目标文件提取技能内容
	Extract(skillID string) (string, error)

	// Remove 从目标文件移除技能
	Remove(skillID string) error

	// List 列出目标文件中的所有技能
	List() ([]string, error)

	// Supports 检查是否支持当前环境
	Supports() bool

	// Cleanup 清理临时文件（备份文件、临时文件等）
	Cleanup() error

	// GetBackupPath 获取备份文件路径
	GetBackupPath() string
}
