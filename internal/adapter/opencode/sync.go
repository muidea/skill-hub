package opencode

import (
	"fmt"
	"os"
	"path/filepath"
)

// createSkillDirectory 创建技能目录（原子操作）
func createSkillDirectory(skillDir string) error {
	// 检查目录是否已存在
	if _, err := os.Stat(skillDir); err == nil {
		// 目录已存在，备份现有目录
		if err := backupSkill(skillDir); err != nil {
			return fmt.Errorf("备份现有技能失败: %w", err)
		}
	}

	// 创建父目录（如果不存在）
	parentDir := filepath.Dir(skillDir)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("创建父目录失败: %w", err)
	}

	// 创建临时目录
	tmpDir := skillDir + ".tmp"
	if err := os.RemoveAll(tmpDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理临时目录失败: %w", err)
	}

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 重命名为目标目录
	if err := os.Rename(tmpDir, skillDir); err != nil {
		// 清理临时目录
		os.RemoveAll(tmpDir)
		// 尝试恢复备份
		if backupDir := skillDir + ".bak"; fileExists(backupDir) {
			os.Rename(backupDir, skillDir)
		}
		return fmt.Errorf("重命名目录失败: %w", err)
	}

	// 清理备份目录
	if backupDir := skillDir + ".bak"; fileExists(backupDir) {
		os.RemoveAll(backupDir)
	}

	return nil
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// backupSkill 备份现有技能
func backupSkill(skillDir string) error {
	// 检查目录是否存在
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return nil // 目录不存在，无需备份
	}

	// 创建备份目录
	backupDir := skillDir + ".bak"

	// 删除旧的备份
	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("删除旧备份失败: %w", err)
	}

	// 重命名现有目录为备份
	if err := os.Rename(skillDir, backupDir); err != nil {
		return fmt.Errorf("创建备份失败: %w", err)
	}

	return nil
}

// restoreBackup 恢复备份
func restoreBackup(skillDir string) error {
	backupDir := skillDir + ".bak"

	// 检查备份是否存在
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil // 备份不存在，无需恢复
	}

	// 删除当前目录（如果存在）
	if err := os.RemoveAll(skillDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除当前目录失败: %w", err)
	}

	// 恢复备份
	if err := os.Rename(backupDir, skillDir); err != nil {
		return fmt.Errorf("恢复备份失败: %w", err)
	}

	return nil
}
