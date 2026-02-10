package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test", "nested", "directory")

	// 测试创建目录
	if err := EnsureDir(testDir); err != nil {
		t.Fatalf("EnsureDir 失败: %v", err)
	}

	// 验证目录是否存在
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Fatal("目录未创建")
	}

	// 测试目录已存在的情况（应该成功）
	if err := EnsureDir(testDir); err != nil {
		t.Fatalf("EnsureDir 在目录已存在时失败: %v", err)
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	nonExistingFile := filepath.Join(tmpDir, "non-existing.txt")

	// 创建测试文件
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 测试文件存在
	if !FileExists(existingFile) {
		t.Error("FileExists 应该返回 true 对于存在的文件")
	}

	// 测试文件不存在
	if FileExists(nonExistingFile) {
		t.Error("FileExists 应该返回 false 对于不存在的文件")
	}
}

func TestSafeWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	// 测试写入新文件
	if err := SafeWriteFile(testFile, testContent); err != nil {
		t.Fatalf("SafeWriteFile 失败: %v", err)
	}

	// 验证文件内容
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("文件内容不匹配: 期望 %q, 得到 %q", testContent, string(content))
	}

	// 测试覆盖现有文件
	newContent := "Updated content"
	if err := SafeWriteFile(testFile, newContent); err != nil {
		t.Fatalf("SafeWriteFile 覆盖文件失败: %v", err)
	}

	// 验证文件内容已更新
	content, err = os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("读取更新后的文件失败: %v", err)
	}
	if string(content) != newContent {
		t.Errorf("更新后的文件内容不匹配: 期望 %q, 得到 %q", newContent, string(content))
	}

	// 验证备份文件已被清理
	backupFile := testFile + ".bak"
	if FileExists(backupFile) {
		t.Error("备份文件应该已被清理")
	}

	// 验证临时文件已被清理
	tmpFile := testFile + ".tmp"
	if FileExists(tmpFile) {
		t.Error("临时文件应该已被清理")
	}
}

func TestSafeWriteFileWithMode(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("Test data")
	testMode := os.FileMode(0600) // 只读权限

	// 测试写入文件并设置权限
	if err := SafeWriteFileWithMode(testFile, testData, testMode); err != nil {
		t.Fatalf("SafeWriteFileWithMode 失败: %v", err)
	}

	// 验证文件权限
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}
	if info.Mode()&0777 != testMode {
		t.Errorf("文件权限不匹配: 期望 %o, 得到 %o", testMode, info.Mode()&0777)
	}
}

func TestReadFileIfExists(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	nonExistingFile := filepath.Join(tmpDir, "non-existing.txt")
	expectedContent := []byte("Test content")

	// 创建测试文件
	if err := os.WriteFile(existingFile, expectedContent, 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 测试读取存在的文件
	content, err := ReadFileIfExists(existingFile)
	if err != nil {
		t.Fatalf("ReadFileIfExists 失败: %v", err)
	}
	if string(content) != string(expectedContent) {
		t.Errorf("文件内容不匹配: 期望 %q, 得到 %q", string(expectedContent), string(content))
	}

	// 测试读取不存在的文件
	content, err = ReadFileIfExists(nonExistingFile)
	if err != nil {
		t.Fatalf("ReadFileIfExists 对于不存在的文件失败: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("对于不存在的文件应该返回空内容, 得到: %q", string(content))
	}
}
