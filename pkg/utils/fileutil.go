package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// EnsureDir 确保目录存在，如果不存在则创建
func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
	}
	return nil
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SafeWriteFile 安全地写入文件，包含备份和原子操作
func SafeWriteFile(path, content string) error {
	return SafeWriteFileWithMode(path, []byte(content), 0644)
}

// SafeWriteFileWithMode 安全地写入文件，包含备份和原子操作，可指定文件模式
func SafeWriteFileWithMode(path string, data []byte, mode os.FileMode) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	// 创建备份（如果文件存在）
	backupPath := path + ".bak"
	if FileExists(path) {
		if err := os.Rename(path, backupPath); err != nil {
			return fmt.Errorf("创建备份失败: %w", err)
		}
	}

	// 写入临时文件
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, mode); err != nil {
		// 尝试恢复备份
		if FileExists(backupPath) {
			os.Rename(backupPath, path)
		}
		return fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 重命名为目标文件
	if err := os.Rename(tmpPath, path); err != nil {
		// 尝试恢复备份
		if FileExists(backupPath) {
			os.Rename(backupPath, path)
		}
		return fmt.Errorf("重命名文件失败: %w", err)
	}

	// 清理备份文件
	if FileExists(backupPath) {
		os.Remove(backupPath)
	}

	return nil
}

// SafeWriteJSONFile 安全地写入JSON文件，包含备份和原子操作
func SafeWriteJSONFile(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化JSON失败: %w", err)
	}

	return SafeWriteFileWithMode(path, jsonData, 0644)
}

// ReadFileIfExists 如果文件存在则读取，否则返回空内容
func ReadFileIfExists(path string) ([]byte, error) {
	if !FileExists(path) {
		return []byte{}, nil
	}
	return os.ReadFile(path)
}
