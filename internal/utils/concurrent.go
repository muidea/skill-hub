package utils

import (
	"path/filepath"
	"sync"

	"skill-hub/pkg/errors"
)

// FileLockManager 文件锁管理器
type FileLockManager struct {
	locks sync.Map // map[string]*sync.RWMutex
}

// NewFileLockManager 创建新的文件锁管理器
func NewFileLockManager() *FileLockManager {
	return &FileLockManager{}
}

// getLock 获取文件的锁
func (m *FileLockManager) getLock(path string) *sync.RWMutex {
	// 使用绝对路径作为键
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path // 如果获取绝对路径失败，使用原路径
	}

	lock, _ := m.locks.LoadOrStore(absPath, &sync.RWMutex{})
	return lock.(*sync.RWMutex)
}

// Lock 获取文件的写锁
func (m *FileLockManager) Lock(path string) {
	m.getLock(path).Lock()
}

// Unlock 释放文件的写锁
func (m *FileLockManager) Unlock(path string) {
	m.getLock(path).Unlock()
}

// RLock 获取文件的读锁
func (m *FileLockManager) RLock(path string) {
	m.getLock(path).RLock()
}

// RUnlock 释放文件的读锁
func (m *FileLockManager) RUnlock(path string) {
	m.getLock(path).RUnlock()
}

// WithWriteLock 在写锁保护下执行函数
func (m *FileLockManager) WithWriteLock(path string, fn func() error) error {
	m.Lock(path)
	defer m.Unlock(path)
	return fn()
}

// WithReadLock 在读锁保护下执行函数
func (m *FileLockManager) WithReadLock(path string, fn func() error) error {
	m.RLock(path)
	defer m.RUnlock(path)
	return fn()
}

// SafeWriteFile 安全的文件写入（带锁）
func (m *FileLockManager) SafeWriteFile(path string, data []byte) error {
	return m.WithWriteLock(path, func() error {
		return WriteFile(path, data)
	})
}

// SafeReadFile 安全的文件读取（带锁）
func (m *FileLockManager) SafeReadFile(path string) ([]byte, error) {
	var result []byte
	var err error

	err = m.WithReadLock(path, func() error {
		result, err = ReadFile(path)
		return err
	})

	return result, err
}

// SafeCopyFile 安全的文件复制（带锁）
func (m *FileLockManager) SafeCopyFile(src, dst string) error {
	// 对源文件加读锁，目标文件加写锁
	m.RLock(src)
	defer m.RUnlock(src)

	m.Lock(dst)
	defer m.Unlock(dst)

	return CopyFile(src, dst)
}

// ConcurrentFileProcessor 并发文件处理器
type ConcurrentFileProcessor struct {
	lockManager *FileLockManager
	workerCount int
}

// NewConcurrentFileProcessor 创建新的并发文件处理器
func NewConcurrentFileProcessor(workerCount int) *ConcurrentFileProcessor {
	if workerCount <= 0 {
		workerCount = 4 // 默认4个worker
	}

	return &ConcurrentFileProcessor{
		lockManager: NewFileLockManager(),
		workerCount: workerCount,
	}
}

// ProcessFiles 并发处理文件
func (p *ConcurrentFileProcessor) ProcessFiles(
	files []string,
	processFunc func(string) error,
) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(files))
	semaphore := make(chan struct{}, p.workerCount)

	for _, file := range files {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()

			// 获取信号量限制并发数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 对文件加读锁并处理
			err := p.lockManager.WithReadLock(f, func() error {
				return processFunc(f)
			})

			if err != nil {
				errCh <- errors.Wrapf(err, "处理文件失败: %s", f)
			}
		}(file)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errCh)

	// 收集所有错误
	multiErr := errors.NewMultiError()
	for err := range errCh {
		multiErr.Add(err)
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

// UpdateFiles 并发更新文件
func (p *ConcurrentFileProcessor) UpdateFiles(
	fileUpdates map[string][]byte,
	atomic bool,
) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(fileUpdates))
	semaphore := make(chan struct{}, p.workerCount)

	for file, content := range fileUpdates {
		wg.Add(1)
		go func(f string, c []byte) {
			defer wg.Done()

			// 获取信号量限制并发数
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 对文件加写锁并更新
			err := p.lockManager.WithWriteLock(f, func() error {
				if atomic {
					return WriteFile(f, c)
				}
				return WriteFileDirect(f, c)
			})

			if err != nil {
				errCh <- errors.Wrapf(err, "更新文件失败: %s", f)
			}
		}(file, content)
	}

	// 等待所有goroutine完成
	wg.Wait()
	close(errCh)

	// 收集所有错误
	multiErr := errors.NewMultiError()
	for err := range errCh {
		multiErr.Add(err)
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

// 全局文件锁管理器实例
var globalFileLockManager = NewFileLockManager()

// GlobalFileLockManager 获取全局文件锁管理器
func GlobalFileLockManager() *FileLockManager {
	return globalFileLockManager
}
