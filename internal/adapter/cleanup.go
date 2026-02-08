package adapter

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// CleanupTempFiles 清理临时文件（备份文件、临时文件等）
func CleanupTempFiles(basePath string) error {
	// 清理 .bak 文件
	backupPath := basePath + ".bak"
	if err := os.Remove(backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理备份文件失败 %s: %w", backupPath, err)
	}

	// 清理 .tmp 文件
	tmpPath := basePath + ".tmp"
	if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理临时文件失败 %s: %w", tmpPath, err)
	}

	return nil
}

// CleanupBackupDir 清理备份目录
func CleanupBackupDir(dirPath string) error {
	backupDir := dirPath + ".bak"
	if err := os.RemoveAll(backupDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("清理备份目录失败 %s: %w", backupDir, err)
	}
	return nil
}

// RestoreFileBackup 恢复文件备份
func RestoreFileBackup(targetPath, backupPath string) error {
	// 检查备份文件是否存在
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return nil // 备份不存在，无需恢复
	}

	// 删除当前文件（如果存在）
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除当前文件失败: %w", err)
	}

	// 恢复备份
	if err := os.Rename(backupPath, targetPath); err != nil {
		return fmt.Errorf("恢复备份失败: %w", err)
	}

	return nil
}

// RestoreDirBackup 恢复目录备份
func RestoreDirBackup(targetDir, backupDir string) error {
	// 检查备份目录是否存在
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return nil // 备份不存在，无需恢复
	}

	// 删除当前目录（如果存在）
	if err := os.RemoveAll(targetDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除当前目录失败: %w", err)
	}

	// 恢复备份
	if err := os.Rename(backupDir, targetDir); err != nil {
		return fmt.Errorf("恢复备份失败: %w", err)
	}

	return nil
}

// CleanupAllTempFiles 清理指定目录下的所有临时文件
func CleanupAllTempFiles(dirPath string) error {
	// 读取目录中的所有文件
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，无需清理
		}
		return fmt.Errorf("读取目录失败: %w", err)
	}

	// 清理所有 .bak 和 .tmp 文件
	for _, entry := range entries {
		if !entry.IsDir() {
			filename := entry.Name()
			filePath := filepath.Join(dirPath, filename)

			// 检查是否是临时文件
			if filepath.Ext(filename) == ".bak" || filepath.Ext(filename) == ".tmp" {
				if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("清理临时文件失败 %s: %w", filePath, err)
				}
			}
		}
	}

	return nil
}

// CleanupTimestampedBackupDirs 清理带时间戳的备份目录
func CleanupTimestampedBackupDirs(basePath string) error {
	// 获取basePath的父目录
	parentDir := filepath.Dir(basePath)
	baseName := filepath.Base(basePath)

	// 读取父目录中的所有条目
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 父目录不存在，无需清理
		}
		return fmt.Errorf("读取目录失败: %w", err)
	}

	// 正则表达式匹配带时间戳的备份目录
	// 格式: baseName.bak.YYYYMMDD-HHMMSS
	backupPattern := regexp.MustCompile(`^` + regexp.QuoteMeta(baseName) + `\.bak\.\d{8}-\d{6}$`)

	// 清理匹配的备份目录
	for _, entry := range entries {
		if entry.IsDir() && backupPattern.MatchString(entry.Name()) {
			backupDir := filepath.Join(parentDir, entry.Name())
			if err := os.RemoveAll(backupDir); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("清理备份目录失败 %s: %w", backupDir, err)
			}
		}
	}

	return nil
}
