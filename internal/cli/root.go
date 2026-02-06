package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "skill-hub",
	Short: "Skill Hub - AI技能生命周期管理工具",
	Long: `Skill Hub 是一款专为 AI 时代开发者设计的"技能（Prompt/Script）生命周期管理工具"。
它旨在解决 AI 指令碎片化、跨工具同步难、缺乏版本控制等痛点。

核心理念：Git 为中心，一键分发，闭环反馈。`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(feedbackCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(gitCmd)
}
