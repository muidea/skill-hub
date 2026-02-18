package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"skill-hub/pkg/errors"
)

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirExists 检查目录是否存在
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// EnsureDir 确保目录存在
func EnsureDir(path string) error {
	if DirExists(path) {
		return nil
	}
	return os.MkdirAll(path, 0755)
}

// ReadFile 读取文件内容
func ReadFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "读取文件失败")
	}
	return data, nil
}

// ReadFileString 读取文件内容为字符串
func ReadFileString(path string) (string, error) {
	data, err := ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile 写入文件（原子操作）
func WriteFile(path string, data []byte) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return errors.Wrap(err, "创建目录失败")
	}

	// 创建临时文件
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return errors.Wrap(err, "写入临时文件失败")
	}

	// 原子重命名
	if err := os.Rename(tmpPath, path); err != nil {
		// 清理临时文件
		os.Remove(tmpPath)
		return errors.Wrap(err, "重命名文件失败")
	}

	return nil
}

// WriteFileDirect 直接写入文件（非原子操作，性能更高）
func WriteFileDirect(path string, data []byte) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return errors.Wrap(err, "创建目录失败")
	}

	// 直接写入文件（非原子操作）
	if err := os.WriteFile(path, data, 0644); err != nil {
		return errors.Wrap(err, "写入文件失败")
	}

	return nil
}

// WriteFileStringDirect 直接写入字符串到文件（非原子操作，性能更高）
func WriteFileStringDirect(path, content string) error {
	return WriteFileDirect(path, []byte(content))
}

// WriteFileString 写入字符串到文件
func WriteFileString(path, content string) error {
	return WriteFile(path, []byte(content))
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err, "打开源文件失败")
	}
	defer srcFile.Close()

	// 确保目标目录存在
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return errors.Wrap(err, "创建目标文件失败")
	}
	defer dstFile.Close()

	// 使用带缓冲区的复制以提高大文件性能
	buf := make([]byte, 32*1024) // 32KB缓冲区
	if _, err := io.CopyBuffer(dstFile, srcFile, buf); err != nil {
		return errors.Wrap(err, "复制文件内容失败")
	}

	return nil
}

// CopyFileWithBuffer 复制文件（可指定缓冲区大小）
func CopyFileWithBuffer(src, dst string, bufferSize int) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return errors.Wrap(err, "打开源文件失败")
	}
	defer srcFile.Close()

	// 确保目标目录存在
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return errors.Wrap(err, "创建目标文件失败")
	}
	defer dstFile.Close()

	// 使用指定大小的缓冲区
	if bufferSize <= 0 {
		bufferSize = 32 * 1024 // 默认32KB
	}
	buf := make([]byte, bufferSize)
	if _, err := io.CopyBuffer(dstFile, srcFile, buf); err != nil {
		return errors.Wrap(err, "复制文件内容失败")
	}

	return nil
}

// CopyDir 复制目录
func CopyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return errors.Wrap(err, "获取源目录信息失败")
	}
	if !srcInfo.IsDir() {
		return errors.NewWithCode("CopyDir", errors.ErrFileOperation, "源路径不是目录")
	}

	// 创建目标目录
	if err := EnsureDir(dst); err != nil {
		return err
	}

	// 遍历源目录
	entries, err := os.ReadDir(src)
	if err != nil {
		return errors.Wrap(err, "读取源目录失败")
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// RemoveFile 删除文件
func RemoveFile(path string) error {
	if err := os.Remove(path); err != nil {
		return errors.Wrap(err, "删除文件失败")
	}
	return nil
}

// RemoveDir 删除目录
func RemoveDir(path string) error {
	if err := os.RemoveAll(path); err != nil {
		return errors.Wrap(err, "删除目录失败")
	}
	return nil
}

// ListFiles 列出目录中的文件
func ListFiles(dir string, pattern string) ([]string, error) {
	if !DirExists(dir) {
		return nil, errors.NewWithCodef("ListFiles", errors.ErrFileNotFound, "目录不存在: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, "读取目录失败")
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if pattern == "" {
				files = append(files, filepath.Join(dir, name))
			} else {
				matched, err := filepath.Match(pattern, name)
				if err == nil && matched {
					files = append(files, filepath.Join(dir, name))
				}
			}
		}
	}

	return files, nil
}

// ListDirs 列出目录中的子目录
func ListDirs(dir string) ([]string, error) {
	if !DirExists(dir) {
		return nil, errors.NewWithCodef("ListDirs", errors.ErrFileNotFound, "目录不存在: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, "读取目录失败")
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(dir, entry.Name()))
		}
	}

	return dirs, nil
}

// FileSize 获取文件大小
func FileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, errors.Wrap(err, "获取文件信息失败")
	}
	return info.Size(), nil
}

// IsEmptyDir 检查目录是否为空
func IsEmptyDir(path string) (bool, error) {
	if !DirExists(path) {
		return false, errors.NewWithCodef("IsEmptyDir", errors.ErrFileNotFound, "目录不存在: %s", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return false, errors.Wrap(err, "读取目录失败")
	}

	return len(entries) == 0, nil
}

// CreateTempFile 创建临时文件
func CreateTempFile(dir, pattern string) (string, error) {
	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", errors.Wrap(err, "创建临时文件失败")
	}
	defer file.Close()
	return file.Name(), nil
}

// CreateTempDir 创建临时目录
func CreateTempDir(dir, pattern string) (string, error) {
	path, err := os.MkdirTemp(dir, pattern)
	if err != nil {
		return "", errors.Wrap(err, "创建临时目录失败")
	}
	return path, nil
}

// BatchCopyFiles 批量复制文件
func BatchCopyFiles(filePairs map[string]string) error {
	multiErr := errors.NewMultiError()

	for src, dst := range filePairs {
		if err := CopyFile(src, dst); err != nil {
			multiErr.Add(errors.Wrapf(err, "复制文件失败: %s -> %s", src, dst))
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

// BatchWriteFiles 批量写入文件
func BatchWriteFiles(fileContents map[string][]byte, atomic bool) error {
	multiErr := errors.NewMultiError()

	for path, content := range fileContents {
		var err error
		if atomic {
			err = WriteFile(path, content)
		} else {
			err = WriteFileDirect(path, content)
		}

		if err != nil {
			multiErr.Add(errors.Wrapf(err, "写入文件失败: %s", path))
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

// ReadFileChunk 读取文件片段
func ReadFileChunk(path string, offset, length int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "打开文件失败")
	}
	defer file.Close()

	// 定位到指定偏移量
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, errors.Wrap(err, "定位文件失败")
	}

	// 读取指定长度的数据
	buffer := make([]byte, length)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "读取文件失败")
	}

	return buffer[:n], nil
}

// FileChecksum 计算文件校验和（简单实现）
func FileChecksum(path string) (string, error) {
	data, err := ReadFile(path)
	if err != nil {
		return "", err
	}

	// 简单的校验和计算（实际应用中可以使用更复杂的算法）
	var sum uint32
	for _, b := range data {
		sum = sum*31 + uint32(b)
	}

	return fmt.Sprintf("%08x", sum), nil
}
