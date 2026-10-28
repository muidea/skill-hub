package cli

import (
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "清理 state.json 里的失效项目记录",
	Long:  "清理 state.json 中已经失效的项目路径记录，例如项目目录被移动或删除后遗留的状态信息。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPrune()
	},
}

func runPrune() error {
	printlnText("清理 state.json 失效项目记录...")

	removed, err := localRuntime.ProjectState().PruneInvalidProjects()
	if err != nil {
		return err
	}

	if jsonMode() {
		if removed == nil {
			removed = []string{}
		}
		return writeJSON(map[string]any{"removed": removed, "total": len(removed)})
	}
	if len(removed) == 0 {
		printlnText("✓ 未发现失效项目记录")
		return nil
	}

	printfText("✓ 已清理 %d 条失效项目记录\n", len(removed))
	for _, projectPath := range removed {
		printfText("  - %s\n", projectPath)
	}

	return nil
}
