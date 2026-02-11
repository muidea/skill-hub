package utils

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestFileLockManager(t *testing.T) {
	lockManager := NewFileLockManager()
	tmpDir := t.TempDir()

	t.Run("BasicLockUnlock", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test1.txt")

		// 测试写锁
		lockManager.Lock(testFile)

		// 在另一个goroutine中尝试获取锁
		var wg sync.WaitGroup
		wg.Add(1)
		locked := false

		go func() {
			defer wg.Done()
			lockManager.Lock(testFile)
			locked = true
			lockManager.Unlock(testFile)
		}()

		// 等待一小段时间，确保goroutine已经尝试获取锁
		time.Sleep(10 * time.Millisecond)

		if locked {
			t.Error("锁不应该被获取，因为主goroutine还持有锁")
		}

		// 释放锁
		lockManager.Unlock(testFile)

		// 等待goroutine完成
		wg.Wait()

		if !locked {
			t.Error("锁应该已经被goroutine获取")
		}
	})

	t.Run("ReadWriteLock", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test2.txt")

		// 获取读锁
		lockManager.RLock(testFile)

		// 另一个读锁应该可以同时获取
		var wg sync.WaitGroup
		wg.Add(1)
		readLocked := false

		go func() {
			defer wg.Done()
			lockManager.RLock(testFile)
			readLocked = true
			lockManager.RUnlock(testFile)
		}()

		time.Sleep(10 * time.Millisecond)

		if !readLocked {
			t.Error("读锁应该可以同时被多个goroutine获取")
		}

		// 写锁应该被阻塞
		wg.Add(1)
		writeLocked := false

		go func() {
			defer wg.Done()
			lockManager.Lock(testFile)
			writeLocked = true
			lockManager.Unlock(testFile)
		}()

		time.Sleep(10 * time.Millisecond)

		if writeLocked {
			t.Error("写锁应该被阻塞，因为还有读锁")
		}

		// 释放读锁
		lockManager.RUnlock(testFile)

		// 等待goroutine完成
		wg.Wait()

		if !writeLocked {
			t.Error("写锁应该已经被获取")
		}
	})

	t.Run("WithWriteLock", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test3.txt")
		counter := 0

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := lockManager.WithWriteLock(testFile, func() error {
					// 模拟一些工作
					current := counter
					time.Sleep(1 * time.Millisecond)
					counter = current + 1
					return nil
				})
				if err != nil {
					t.Errorf("WithWriteLock failed: %v", err)
				}
			}()
		}

		wg.Wait()

		if counter != 10 {
			t.Errorf("计数器应该是10，但是是%d", counter)
		}
	})

	t.Run("SafeWriteRead", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test4.txt")
		testContent := []byte("test content")

		// 安全写入
		if err := lockManager.SafeWriteFile(testFile, testContent); err != nil {
			t.Fatalf("SafeWriteFile failed: %v", err)
		}

		// 安全读取
		content, err := lockManager.SafeReadFile(testFile)
		if err != nil {
			t.Fatalf("SafeReadFile failed: %v", err)
		}

		if string(content) != string(testContent) {
			t.Errorf("读取的内容不匹配: 期望 %q, 得到 %q", testContent, content)
		}
	})
}

func TestConcurrentFileProcessor(t *testing.T) {
	tmpDir := t.TempDir()
	processor := NewConcurrentFileProcessor(2) // 2个worker

	// 创建测试文件
	files := make([]string, 10)
	for i := 0; i < 10; i++ {
		files[i] = filepath.Join(tmpDir, "file_"+string(rune('a'+i))+".txt")
		if err := WriteFileString(files[i], "initial content"); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	t.Run("ProcessFiles", func(t *testing.T) {
		processed := make(map[string]bool)
		var mu sync.Mutex

		err := processor.ProcessFiles(files, func(file string) error {
			// 模拟处理工作
			time.Sleep(5 * time.Millisecond)

			mu.Lock()
			processed[file] = true
			mu.Unlock()

			return nil
		})

		if err != nil {
			t.Fatalf("ProcessFiles failed: %v", err)
		}

		if len(processed) != len(files) {
			t.Errorf("不是所有文件都被处理了: 处理了 %d 个文件，总共 %d 个", len(processed), len(files))
		}
	})

	t.Run("UpdateFiles", func(t *testing.T) {
		fileUpdates := make(map[string][]byte)
		for i, file := range files {
			fileUpdates[file] = []byte("updated content " + string(rune('0'+i)))
		}

		err := processor.UpdateFiles(fileUpdates, false) // 非原子写入
		if err != nil {
			t.Fatalf("UpdateFiles failed: %v", err)
		}

		// 验证文件内容
		for file, expectedContent := range fileUpdates {
			content, err := ReadFileString(file)
			if err != nil {
				t.Errorf("Failed to read file %s: %v", file, err)
				continue
			}

			if content != string(expectedContent) {
				t.Errorf("文件内容不匹配 %s: 期望 %q, 得到 %q", file, expectedContent, content)
			}
		}
	})

	t.Run("ConcurrentSafety", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "concurrent_test.txt")
		if err := WriteFileString(testFile, "0"); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// 创建多个processor同时操作同一个文件
		var wg sync.WaitGroup
		processors := make([]*ConcurrentFileProcessor, 5)
		for i := range processors {
			processors[i] = NewConcurrentFileProcessor(1)
		}

		for i, p := range processors {
			wg.Add(1)
			go func(idx int, proc *ConcurrentFileProcessor) {
				defer wg.Done()

				updates := map[string][]byte{
					testFile: []byte(string(rune('0' + idx + 1))),
				}

				// 每个processor尝试更新文件
				_ = proc.UpdateFiles(updates, false)
			}(i, p)
		}

		wg.Wait()

		// 文件应该被正确更新（虽然顺序不确定）
		content, err := ReadFileString(testFile)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		// 内容应该是某个数字（1-5）
		if len(content) != 1 || content[0] < '1' || content[0] > '5' {
			t.Errorf("文件内容应该是1-5之间的数字，但是是 %q", content)
		}
	})
}

func BenchmarkFileLockManager(b *testing.B) {
	lockManager := NewFileLockManager()
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "benchmark.txt")

	b.Run("LockUnlock", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			lockManager.Lock(testFile)
			lockManager.Unlock(testFile)
		}
	})

	b.Run("RLockRUnlock", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			lockManager.RLock(testFile)
			lockManager.RUnlock(testFile)
		}
	})

	b.Run("WithWriteLock", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = lockManager.WithWriteLock(testFile, func() error {
				return nil
			})
		}
	})

	b.Run("ConcurrentLocks", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			counter := 0
			for pb.Next() {
				counter++
				filename := filepath.Join(tmpDir, "file_"+string(rune('a'+(counter%26)))+".txt")

				_ = lockManager.WithWriteLock(filename, func() error {
					// 模拟一些工作
					_ = WriteFileString(filename, "test")
					return nil
				})

				_ = lockManager.WithReadLock(filename, func() error {
					_, _ = ReadFileString(filename)
					return nil
				})
			}
		})
	})
}

func BenchmarkConcurrentFileProcessor(b *testing.B) {
	tmpDir := b.TempDir()
	processor := NewConcurrentFileProcessor(4)

	// 创建测试文件
	files := make([]string, 100)
	for i := range files {
		files[i] = filepath.Join(tmpDir, "benchmark_file_"+string(rune('a'+(i%26)))+".txt")
		_ = WriteFileString(files[i], "initial content")
	}

	b.Run("ProcessFiles", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = processor.ProcessFiles(files[:10], func(file string) error {
				// 简单的处理
				content, _ := ReadFileString(file)
				_ = WriteFileString(file, content+" processed")
				return nil
			})
		}
	})

	b.Run("UpdateFiles", func(b *testing.B) {
		fileUpdates := make(map[string][]byte, 10)
		for i := 0; i < 10; i++ {
			fileUpdates[files[i]] = []byte("updated content")
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = processor.UpdateFiles(fileUpdates, false)
		}
	})
}
