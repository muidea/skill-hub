package utils

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkFileExists 测试FileExists性能
func BenchmarkFileExists(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	// 创建测试文件
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FileExists(tmpFile)
	}
}

// BenchmarkDirExists 测试DirExists性能
func BenchmarkDirExists(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DirExists(tmpDir)
	}
}

// BenchmarkEnsureDir 测试EnsureDir性能
func BenchmarkEnsureDir(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newDir := filepath.Join(tmpDir, "benchmark", "test", "dir")
		_ = EnsureDir(newDir)
	}
}

// BenchmarkReadWriteFile 测试ReadFile和WriteFile性能
func BenchmarkReadWriteFile(b *testing.B) {
	tmpDir := b.TempDir()

	// 测试数据
	testData := []byte("This is a test file content for benchmarking read and write operations.")

	b.Run("WriteFile", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tmpFile := filepath.Join(tmpDir, "write_test", "file.txt")
			_ = WriteFile(tmpFile, testData)
		}
	})

	b.Run("ReadFile", func(b *testing.B) {
		tmpFile := filepath.Join(tmpDir, "read_test", "file.txt")
		if err := EnsureDir(filepath.Dir(tmpFile)); err != nil {
			b.Fatalf("Failed to create directory: %v", err)
		}
		if err := WriteFile(tmpFile, testData); err != nil {
			b.Fatalf("Failed to write test file: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = ReadFile(tmpFile)
		}
	})

	b.Run("WriteFileString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tmpFile := filepath.Join(tmpDir, "write_string_test", "file.txt")
			_ = WriteFileString(tmpFile, string(testData))
		}
	})

	b.Run("ReadFileString", func(b *testing.B) {
		tmpFile := filepath.Join(tmpDir, "read_string_test", "file.txt")
		if err := EnsureDir(filepath.Dir(tmpFile)); err != nil {
			b.Fatalf("Failed to create directory: %v", err)
		}
		if err := WriteFileString(tmpFile, string(testData)); err != nil {
			b.Fatalf("Failed to write test file: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = ReadFileString(tmpFile)
		}
	})
}

// BenchmarkCopyFile 测试CopyFile性能
func BenchmarkCopyFile(b *testing.B) {
	tmpDir := b.TempDir()

	// 创建源文件
	srcFile := filepath.Join(tmpDir, "source.txt")
	testData := make([]byte, 1024*1024) // 1MB文件
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	if err := WriteFile(srcFile, testData); err != nil {
		b.Fatalf("Failed to create source file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dstFile := filepath.Join(tmpDir, "dest", "file.txt")
		_ = CopyFile(srcFile, dstFile)
	}
}

// BenchmarkListFiles 测试ListFiles性能
func BenchmarkListFiles(b *testing.B) {
	tmpDir := b.TempDir()

	// 创建测试文件
	for i := 0; i < 100; i++ {
		filename := filepath.Join(tmpDir, "file_"+string(rune('a'+(i%26)))+".txt")
		_ = WriteFileString(filename, "test content")
	}

	b.Run("ListAllFiles", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = ListFiles(tmpDir, "")
		}
	})

	b.Run("ListTxtFiles", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = ListFiles(tmpDir, "*.txt")
		}
	})
}

// BenchmarkFileSize 测试FileSize性能
func BenchmarkFileSize(b *testing.B) {
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "size_test.txt")

	// 创建测试文件
	testData := make([]byte, 1024*1024) // 1MB文件
	if err := WriteFile(tmpFile, testData); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = FileSize(tmpFile)
	}
}

// BenchmarkCreateTempFileDir 测试临时文件/目录创建性能
func BenchmarkCreateTempFileDir(b *testing.B) {
	tmpDir := b.TempDir()

	b.Run("CreateTempFile", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = CreateTempFile(tmpDir, "benchmark-*.tmp")
		}
	})

	b.Run("CreateTempDir", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = CreateTempDir(tmpDir, "benchmark-*")
		}
	})
}

// BenchmarkConcurrentOperations 测试并发操作性能
func BenchmarkConcurrentOperations(b *testing.B) {
	tmpDir := b.TempDir()
	testData := []byte("test data")

	b.RunParallel(func(pb *testing.PB) {
		counter := 0
		for pb.Next() {
			counter++
			// 混合操作
			filename := filepath.Join(tmpDir, "concurrent", "file_"+string(rune('a'+(counter%26)))+".txt")
			_ = WriteFile(filename, testData)
			_, _ = ReadFile(filename)
			_ = FileExists(filename)
			_, _ = FileSize(filename)
		}
	})
}
