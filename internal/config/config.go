package config

import (
	"os"
	"path/filepath"

	"skill-hub/pkg/errors"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type RepositoryConfig struct {
	Name        string `mapstructure:"name" yaml:"name" json:"name"`                      // 仓库名称
	URL         string `mapstructure:"url" yaml:"url" json:"url"`                         // Git远程URL
	Branch      string `mapstructure:"branch" yaml:"branch" json:"branch"`                // 默认分支
	Enabled     bool   `mapstructure:"enabled" yaml:"enabled" json:"enabled"`             // 是否启用
	Description string `mapstructure:"description" yaml:"description" json:"description"` // 描述
	Type        string `mapstructure:"type" yaml:"type" json:"type"`                      // 类型：user/community/official
	IsArchive   bool   `mapstructure:"is_archive" yaml:"is_archive" json:"is_archive"`    // 是否为归档仓库
	LastSync    string `mapstructure:"last_sync,omitempty" yaml:"last_sync,omitempty" json:"last_sync,omitempty"`
}

type MultiRepoConfig struct {
	Enabled      bool                        `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	DefaultRepo  string                      `mapstructure:"default_repo" yaml:"default_repo" json:"default_repo"` // 默认仓库（同时是归档仓库）
	Repositories map[string]RepositoryConfig `mapstructure:"repositories" yaml:"repositories" json:"repositories"`
}

type Config struct {
	ClaudeConfigPath string           `mapstructure:"claude_config_path"`
	CursorConfigPath string           `mapstructure:"cursor_config_path"`
	DefaultTool      string           `mapstructure:"default_tool"`
	GitToken         string           `mapstructure:"git_token"`
	MultiRepo        *MultiRepoConfig `mapstructure:"multi_repo" yaml:"multi_repo" json:"multi_repo"`
}

var (
	globalConfig *Config
	configLoaded = false
)

// GetConfig 返回全局配置，如果未加载则先加载
func GetConfig() (*Config, error) {
	if !configLoaded {
		if err := LoadConfig(); err != nil {
			return nil, err
		}
	}
	return globalConfig, nil
}

// LoadConfig 加载配置文件
func LoadConfig() error {
	// 支持通过环境变量指定skill-hub目录
	configDir := os.Getenv("SKILL_HUB_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errors.WrapWithCode(err, "LoadConfig", errors.ErrSystem, "获取用户主目录失败")
		}
		configDir = filepath.Join(homeDir, ".skill-hub")
	}

	configFile := filepath.Join(configDir, "config.yaml")

	// 检查配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return errors.ConfigNotFound("LoadConfig", configFile)
	}

	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	// 获取用户主目录用于其他路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.WrapWithCode(err, "LoadConfig", errors.ErrSystem, "获取用户主目录失败")
	}

	// 设置默认值
	viper.SetDefault("claude_config_path", filepath.Join(homeDir, ".claude", "config.json"))
	viper.SetDefault("cursor_config_path", filepath.Join(homeDir, ".cursor", "rules"))
	viper.SetDefault("default_tool", "cursor")
	viper.SetDefault("git_token", "")

	// 多仓库配置默认值 - 强制启用多仓库模式
	viper.SetDefault("multi_repo.enabled", true)
	viper.SetDefault("multi_repo.default_repo", "main")

	if err := viper.ReadInConfig(); err != nil {
		return errors.WrapWithCode(err, "LoadConfig", errors.ErrConfigInvalid, "读取配置文件失败")
	}

	globalConfig = &Config{}
	if err := viper.Unmarshal(globalConfig); err != nil {
		return errors.WrapWithCode(err, "LoadConfig", errors.ErrConfigInvalid, "解析配置文件失败")
	}

	configLoaded = true
	return nil
}

// GetRepoPath 获取默认仓库路径（多仓库模式）
func GetRepoPath() (string, error) {
	cfg, err := GetConfig()
	if err != nil {
		return "", errors.Wrap(err, "GetRepoPath: 获取配置失败")
	}

	// 多仓库模式：获取默认仓库路径
	if cfg.MultiRepo == nil {
		return "", errors.NewWithCode("GetRepoPath", errors.ErrConfigInvalid, "多仓库配置未初始化")
	}

	rootDir, err := GetRootDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(rootDir, "repositories", cfg.MultiRepo.DefaultRepo), nil
}

// GetSkillsDir 获取技能目录路径
func GetSkillsDir() (string, error) {
	repoPath, err := GetRepoPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(repoPath, "skills"), nil
}

// GetRootDir 获取Skill Hub根目录
func GetRootDir() (string, error) {
	// 支持通过环境变量指定skill-hub目录
	configDir := os.Getenv("SKILL_HUB_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", errors.WrapWithCode(err, "GetRootDir", errors.ErrSystem, "获取用户主目录失败")
		}
		configDir = filepath.Join(homeDir, ".skill-hub")
	}
	return configDir, nil
}

// GetRegistryPath 获取索引文件路径
func GetRegistryPath() (string, error) {
	rootDir, err := GetRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootDir, "registry.json"), nil
}

// GetStatePath 获取状态文件路径
func GetStatePath() (string, error) {
	rootDir, err := GetRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootDir, "state.json"), nil
}

// GetArchiveRepository 获取归档仓库配置
func (c *Config) GetArchiveRepository() (*RepositoryConfig, error) {
	// 只支持多仓库模式，默认仓库即为归档仓库
	if c.MultiRepo == nil {
		return nil, errors.NewWithCode("GetArchiveRepository", errors.ErrConfigInvalid, "多仓库配置未初始化")
	}

	repo, exists := c.MultiRepo.Repositories[c.MultiRepo.DefaultRepo]
	if !exists {
		return nil, errors.NewWithCodef("GetArchiveRepository", errors.ErrConfigInvalid, "默认仓库 '%s' 不存在", c.MultiRepo.DefaultRepo)
	}

	// 确保默认仓库标记为归档仓库
	repo.IsArchive = true
	return &repo, nil
}

// IsArchiveRepo 检查是否为归档仓库
func (rc *RepositoryConfig) IsArchiveRepo() bool {
	return rc.IsArchive
}

// GetRepositoryPath 获取指定仓库的路径
func GetRepositoryPath(repoName string) (string, error) {
	rootDir, err := GetRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootDir, "repositories", repoName), nil
}

// GetRepositoriesDir 获取所有仓库的目录
func GetRepositoriesDir() (string, error) {
	rootDir, err := GetRootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(rootDir, "repositories"), nil
}

// SaveConfig 保存配置到文件
func SaveConfig(cfg *Config) error {
	configDir, err := GetRootDir()
	if err != nil {
		return errors.Wrap(err, "SaveConfig: 获取配置目录失败")
	}

	configFile := filepath.Join(configDir, "config.yaml")

	// 使用yaml库序列化配置
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.WrapWithCode(err, "SaveConfig", errors.ErrFileOperation, "序列化配置失败")
	}

	if err := os.WriteFile(configFile, yamlData, 0644); err != nil {
		return errors.WrapWithCode(err, "SaveConfig", errors.ErrFileOperation, "写入配置文件失败")
	}

	// 更新全局配置
	globalConfig = cfg
	configLoaded = true

	return nil
}
