package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/muidea/skill-hub/pkg/validator"
)

var (
	strictMode     bool
	ignoreWarnings bool
	autoFix        bool
	outputFormat   string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "validate",
		Short: "验证技能文件是否符合Agent Skills规范",
		Long: `验证技能文件是否符合Agent Skills规范。

此工具会检查技能文件的格式、必需字段、命名规范等，
确保技能文件能够被Skill Hub和其他兼容Agent Skills的工具正确识别和使用。`,
		Args: cobra.MinimumNArgs(1),
		RunE: runValidate,
	}

	rootCmd.Flags().BoolVar(&strictMode, "strict", false, "严格模式：警告也视为错误")
	rootCmd.Flags().BoolVar(&ignoreWarnings, "ignore-warnings", false, "忽略警告")
	rootCmd.Flags().BoolVar(&autoFix, "auto-fix", false, "自动修复可修复的问题（实验性功能）")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "输出格式：text, json")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func runValidate(cmd *cobra.Command, args []string) error {
	// 创建校验器
	v := validator.NewValidator()
	options := validator.ValidationOptions{
		IgnoreWarnings: ignoreWarnings,
		StrictMode:     strictMode,
	}

	// 收集所有要验证的文件
	var skillFiles []string
	for _, arg := range args {
		// 检查是否是目录
		info, err := os.Stat(arg)
		if err != nil {
			return fmt.Errorf("无法访问 %s: %w", arg, err)
		}

		if info.IsDir() {
			// 如果是目录，查找其中的SKILL.md文件
			err := filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if !info.IsDir() && info.Name() == "SKILL.md" {
					skillFiles = append(skillFiles, path)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("遍历目录 %s 失败: %w", arg, err)
			}
		} else {
			// 如果是文件，直接添加
			skillFiles = append(skillFiles, arg)
		}
	}

	if len(skillFiles) == 0 {
		fmt.Println("未找到要验证的技能文件")
		return nil
	}

	fmt.Printf("找到 %d 个技能文件进行验证\n", len(skillFiles))

	// 验证每个文件
	totalErrors := 0
	totalWarnings := 0
	allResults := make([]*validator.ValidationResult, 0, len(skillFiles))

	for _, skillFile := range skillFiles {
		result, err := v.ValidateWithOptions(skillFile, options)
		if err != nil {
			fmt.Printf("❌ 验证失败 %s: %v\n", skillFile, err)
			continue
		}

		allResults = append(allResults, result)

		// 根据输出格式显示结果
		switch outputFormat {
		case "json":
			// TODO: 实现JSON输出
			fmt.Printf("JSON输出尚未实现，使用文本格式\n")
			fallthrough
		default:
			result.Print()
		}

		totalErrors += len(result.Errors)
		totalWarnings += len(result.Warnings)
	}

	// 显示总结
	fmt.Printf("\n=== 验证总结 ===\n")
	fmt.Printf("验证文件数: %d\n", len(skillFiles))
	fmt.Printf("总错误数: %d\n", totalErrors)
	fmt.Printf("总警告数: %d\n", totalWarnings)

	// 显示可修复的问题
	fixableErrors := 0
	fixableWarnings := 0
	for _, result := range allResults {
		fixableErrors += len(result.GetFixableErrors())
		fixableWarnings += len(result.GetFixableWarnings())
	}

	if fixableErrors > 0 || fixableWarnings > 0 {
		fmt.Printf("\n可自动修复的问题:\n")
		if fixableErrors > 0 {
			fmt.Printf("  - %d 个错误\n", fixableErrors)
		}
		if fixableWarnings > 0 {
			fmt.Printf("  - %d 个警告\n", fixableWarnings)
		}
		if autoFix {
			fmt.Println("\n🔧 正在尝试自动修复...")
			// 这里可以添加自动修复逻辑
			// 目前通过apply命令的--auto-fix选项提供自动修复功能
			fmt.Println("  使用 'skill-hub apply --auto-fix' 进行自动修复")
		} else if fixableErrors > 0 || fixableWarnings > 0 {
			fmt.Println("\n使用 --auto-fix 参数查看修复建议，或使用 'skill-hub apply --auto-fix' 进行自动修复")
		}
	}

	// 根据结果决定退出码
	if totalErrors > 0 {
		fmt.Println("\n❌ 发现规范不符合项，需要修复")
		os.Exit(1)
	} else if strictMode && totalWarnings > 0 {
		fmt.Println("\n❌ 严格模式：发现警告项")
		os.Exit(1)
	} else if totalWarnings > 0 {
		fmt.Println("\n⚠️  发现警告项，建议检查")
	} else {
		fmt.Println("\n✅ 所有技能文件符合规范")
	}

	return nil
}
