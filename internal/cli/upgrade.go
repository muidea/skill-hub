package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	upgradeservice "github.com/muidea/skill-hub/internal/modules/kernel/upgrade/service"
	"github.com/muidea/skill-hub/pkg/errors"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "检测并升级 skill-hub 到最新 Release",
	Long: `检测 GitHub Releases 是否存在新版 skill-hub，并在确认后下载、校验、替换当前二进制。

默认只在发现新版本时交互确认。使用 --check 仅检测版本，使用 --yes 自动确认升级。`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := upgradeservice.Options{
			CurrentVersion:  version,
			TargetVersion:   mustGetStringFlag(cmd, "version"),
			CheckOnly:       mustGetBoolFlag(cmd, "check"),
			DryRun:          mustGetBoolFlag(cmd, "dry-run"),
			Force:           mustGetBoolFlag(cmd, "force"),
			SkipAgentSkills: mustGetBoolFlag(cmd, "skip-agent-skills"),
			NoRestartServe:  mustGetBoolFlag(cmd, "no-restart-serve"),
		}
		return runUpgrade(opts, mustGetBoolFlag(cmd, "yes"), mustGetBoolFlag(cmd, "json"))
	},
}

func init() {
	upgradeCmd.Flags().Bool("check", false, "仅检测是否存在新版本，不修改文件")
	upgradeCmd.Flags().Bool("yes", false, "发现新版本后自动确认升级")
	upgradeCmd.Flags().String("version", "", "升级到指定版本，例如 v0.8.1")
	upgradeCmd.Flags().Bool("dry-run", false, "演习模式，显示将要下载和替换的资产，不修改文件")
	upgradeCmd.Flags().Bool("force", false, "允许重新安装当前版本或安装低于当前版本的指定版本")
	upgradeCmd.Flags().Bool("json", false, "以JSON格式输出升级检测或执行结果")
	upgradeCmd.Flags().Bool("skip-agent-skills", false, "升级二进制后跳过内置 agent skills 同步")
	upgradeCmd.Flags().Bool("no-restart-serve", false, "升级后不自动重启已注册且正在运行的 serve 实例")
}

func mustGetStringFlag(cmd *cobra.Command, name string) string {
	value, _ := cmd.Flags().GetString(name)
	return value
}

func runUpgrade(opts upgradeservice.Options, yes bool, jsonOutput bool) error {
	service := upgradeservice.New()
	result, err := service.Check(context.Background(), opts)
	if err != nil {
		return errors.Wrap(err, "检测 skill-hub 新版本失败")
	}

	if opts.CheckOnly {
		if jsonOutput {
			return writeJSON(result)
		}
		renderUpgradeResult(result)
		return nil
	}
	if !result.UpdateAvailable && !opts.Force {
		if jsonOutput {
			return writeJSON(result)
		}
		renderUpgradeResult(result)
		return nil
	}
	if opts.DryRun {
		result, err = service.Upgrade(context.Background(), opts)
		if err != nil {
			return errors.Wrap(err, "生成 skill-hub 升级计划失败")
		}
		if jsonOutput {
			return writeJSON(result)
		}
		renderUpgradeResult(result)
		return nil
	}
	if jsonOutput && !yes {
		return writeJSON(result)
	}
	if !yes {
		renderUpgradeResult(result)
		if !confirmUpgrade(result) {
			return errors.NewWithCode("runUpgrade", errors.ErrUserCancel, "用户取消升级")
		}
	}

	result, err = service.Upgrade(context.Background(), opts)
	if err != nil {
		return errors.Wrap(err, "升级 skill-hub 失败")
	}
	if jsonOutput {
		return writeJSON(result)
	}
	renderUpgradeResult(result)
	return nil
}

func renderUpgradeResult(result *upgradeservice.Result) {
	if result == nil {
		fmt.Println("未返回升级结果")
		return
	}
	fmt.Printf("当前版本: %s\n", valueOr(result.CurrentVersion, "unknown"))
	if result.LatestVersion != "" {
		fmt.Printf("最新版本: %s\n", result.LatestVersion)
	}
	if result.TargetVersion != "" && result.TargetVersion != result.LatestVersion {
		fmt.Printf("目标版本: %s\n", result.TargetVersion)
	}
	fmt.Printf("状态: %s\n", result.Status)

	if result.ArchiveName != "" {
		fmt.Printf("Release 资产: %s\n", result.ArchiveName)
	}
	if result.InstallPath != "" {
		fmt.Printf("安装路径: %s\n", result.InstallPath)
	}

	switch result.Status {
	case "up_to_date":
		fmt.Println("skill-hub 已是最新版本")
	case "target_not_newer":
		fmt.Println("目标版本不高于当前版本；如需重新安装或回退，请使用 --force")
	case "planned":
		fmt.Println("dry-run 已完成，未修改任何文件")
	case "upgraded":
		fmt.Println("skill-hub 升级完成")
		if result.AgentSkillsInstalled > 0 {
			fmt.Printf("已同步 agent skills: %d\n", result.AgentSkillsInstalled)
		}
		if len(result.ServeRestarted) > 0 {
			fmt.Printf("已重启 serve 实例: %s\n", strings.Join(result.ServeRestarted, ", "))
		}
	case "update_available":
		fmt.Println("发现可升级版本")
	case "check_complete":
		fmt.Println("版本检测完成")
	}
	for _, warning := range result.Warnings {
		fmt.Printf("警告: %s\n", warning)
	}
}

func confirmUpgrade(result *upgradeservice.Result) bool {
	fmt.Printf("是否升级到 %s？(y/n): ", result.TargetVersion)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
