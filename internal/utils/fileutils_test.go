package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	// 文件不存在
	if FileExists(tmpFile) {
		t.Error("FileExists should return false for non-existent file")
	}

	// 创建文件
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 文件存在
	if !FileExists(tmpFile) {
		t.Error("FileExists should return true for existing file")
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	// 目录存在
	if !DirExists(tmpDir) {
		t.Error("DirExists should return true for existing directory")
	}

	// 目录不存在
	if DirExists(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("DirExists should return false for non-existent directory")
	}

	// 路径是文件不是目录
	tmpFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if DirExists(tmpFile) {
		t.Error("DirExists should return false for file path")
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "a", "b", "c")

	// 确保目录存在
	if err := EnsureDir(newDir); err != nil {
		t.Errorf("EnsureDir failed: %v", err)
	}

	if !DirExists(newDir) {
		t.Error("EnsureDir should create directory")
	}

	// 再次调用应该成功
	if err := EnsureDir(newDir); err != nil {
		t.Errorf("EnsureDir should succeed for existing directory: %v", err)
	}
}

func TestReadWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!"

	// 写入文件
	if err := WriteFileString(tmpFile, content); err != nil {
		t.Fatalf("WriteFileString failed: %v", err)
	}

	// 读取文件
	readContent, err := ReadFileString(tmpFile)
	if err != nil {
		t.Fatalf("ReadFileString failed: %v", err)
	}

	if readContent != content {
		t.Errorf("Read content mismatch: got %q, want %q", readContent, content)
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "src.txt")
	dstFile := filepath.Join(tmpDir, "dst.txt")
	content := "Test content"

	// 创建源文件
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// 复制文件
	if err := CopyFile(srcFile, dstFile); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// 验证目标文件
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != content {
		t.Errorf("Copied content mismatch: got %q, want %q", string(dstContent), content)
	}
}

func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试文件
	files := []string{"a.txt", "b.txt", "c.md", "subdir/d.txt"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// 列出所有文件
	allFiles, err := ListFiles(tmpDir, "")
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(allFiles) != 3 { // 只返回顶层文件，不包括子目录中的文件
		t.Errorf("Expected 3 files, got %d", len(allFiles))
	}

	// 列出.txt文件
	txtFiles, err := ListFiles(tmpDir, "*.txt")
	if err != nil {
		t.Fatalf("ListFiles with pattern failed: %v", err)
	}

	if len(txtFiles) != 2 {
		t.Errorf("Expected 2 .txt files, got %d", len(txtFiles))
	}
}

func TestCreateTempFileDir(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建临时文件
	tempFile, err := CreateTempFile(tmpDir, "test-*.txt")
	if err != nil {
		t.Fatalf("CreateTempFile failed: %v", err)
	}

	if !FileExists(tempFile) {
		t.Error("CreateTempFile should create file")
	}

	// 创建临时目录
	tempDir, err := CreateTempDir(tmpDir, "test-*")
	if err != nil {
		t.Fatalf("CreateTempDir failed: %v", err)
	}

	if !DirExists(tempDir) {
		t.Error("CreateTempDir should create directory")
	}
}
