package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	RepoPath         string `mapstructure:"repo_path"`
	ClaudeConfigPath string `mapstructure:"claude_config_path"`
	CursorConfigPath string `mapstructure:"cursor_config_path"`
	DefaultTool      string `mapstructure:"default_tool"`
	GitRemoteURL     string `mapstructure:"git_remote_url"`
	GitToken         string `mapstructure:"git_token"`
	GitBranch        string `mapstructure:"git_branch"`
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
			return fmt.Errorf("获取用户主目录失败: %w", err)
		}
		configDir = filepath.Join(homeDir, ".skill-hub")
	}

	configFile := filepath.Join(configDir, "config.yaml")

	// 检查配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在，请先运行 'skill-hub init'")
	}

	viper.SetConfigFile(configFile)
	viper.SetConfigType("yaml")

	// 设置默认值
	viper.SetDefault("repo_path", filepath.Join(configDir, "repo"))

	// 获取用户主目录用于其他路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("获取用户主目录失败: %w", err)
	}

	viper.SetDefault("claude_config_path", filepath.Join(homeDir, ".claude", "config.json"))
	viper.SetDefault("cursor_config_path", filepath.Join(homeDir, ".cursor", "rules"))
	viper.SetDefault("default_tool", "cursor")
	viper.SetDefault("git_remote_url", "")
	viper.SetDefault("git_token", "")
	viper.SetDefault("git_branch", "main")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	globalConfig = &Config{}
	if err := viper.Unmarshal(globalConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	configLoaded = true
	return nil
}

// GetRepoPath 获取仓库路径
func GetRepoPath() (string, error) {
	cfg, err := GetConfig()
	if err != nil {
		return "", err
	}
	return expandPath(cfg.RepoPath), nil
}

// expandPath 展开路径中的~为用户主目录
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
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
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(homeDir, ".skill-hub"), nil
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
