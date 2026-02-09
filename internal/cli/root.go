package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version string
	commit  string
	date    string
)

func init() {
	// 设置默认值（如果未在构建时设置）
	if version == "" {
		version = "dev"
	}
	if commit == "" {
		commit = "none"
	}
	if date == "" {
		date = "unknown"
	}
}

var rootCmd = &cobra.Command{
	Use:   "skill-hub",
	Short: "Skill Hub - AI技能生命周期管理工具",
	Long: `Skill Hub 是一款专为 AI 时代开发者设计的"技能（Prompt/Script）生命周期管理工具"。
它旨在解决 AI 指令碎片化、跨工具同步难、缺乏版本控制等痛点。

核心理念：Git 为中心，一键分发，闭环反馈。`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(setTargetCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(feedbackCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(gitCmd)
}
