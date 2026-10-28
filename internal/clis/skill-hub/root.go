package cli

import (
	"fmt"
	"os"

	apperrors "github.com/muidea/skill-hub/pkg/errors"
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
	Use:           "skill-hub",
	Short:         "skill-hub - AI技能生命周期管理工具",
	SilenceErrors: true,
	Long: `skill-hub 是一款专为 AI 时代开发者设计的"技能（Prompt/Script）生命周期管理工具"。
它旨在解决 AI 指令碎片化、跨工具同步难、缺乏版本控制等痛点。

核心理念：Git 为中心，一键分发，闭环反馈。`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: false,
		HiddenDefaultCmd:  true,
	},
}

func Execute() error {
	return executeCommand(rootCmd, os.Args[1:])
}

func FormatError(err error) string {
	if err == nil {
		return ""
	}
	var reported *reportedError
	if apperrors.As(err, &reported) {
		err = reported.error
	}
	if msg := apperrors.Message(err); msg != "" {
		return msg
	}
	return err.Error()
}

func init() {
	rootCmd.PersistentFlags().Bool("json", false, "输出完整 JSON，禁止交互，适合 agent 和自动化调用")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if activeOutput != nil {
			activeOutput.command = cmd
		}
		if jsonMode() && (cmd.Name() == "__complete" || cmd.Name() == "__completeNoDesc" || cmd.Name() == "completion") {
			return apperrors.NewWithCode("cli", apperrors.ErrInvalidInput, "shell 补全不支持 --json")
		}
		return nil
	}
	rootCmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return apperrors.NewWithCode("flags", apperrors.ErrInvalidInput, err.Error())
	})
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if jsonMode() {
			_ = writeJSON(map[string]any{"command": cmd.CommandPath(), "usage": cmd.UsageString(), "description": cmd.Long})
			return
		}
		defaultHelp(cmd, args)
	})
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(projectCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(importCmd)
	rootCmd.AddCommand(dedupeCmd)
	rootCmd.AddCommand(syncCopiesCmd)
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(auditCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(feedbackCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.AddCommand(gitCmd)
	rootCmd.AddCommand(repoCmd)
	rootCmd.AddCommand(pruneCmd)
	markCommandExecution(rootCmd)
}

// Mark execution after Cobra has validated arguments, required flags and flag groups.
func markCommandExecution(command *cobra.Command) {
	if run := command.RunE; run != nil {
		command.RunE = func(cmd *cobra.Command, args []string) error {
			if activeOutput != nil {
				activeOutput.started = true
			}
			return run(cmd, args)
		}
	}
	for _, child := range command.Commands() {
		markCommandExecution(child)
	}
}
