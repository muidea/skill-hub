package fs

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FileSystem 定义文件系统操作接口
type FileSystem interface {
	Stat(name string) (os.FileInfo, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	ReadDir(name string) ([]fs.DirEntry, error)
	MkdirAll(path string, perm os.FileMode) error
	RemoveAll(path string) error
	IsNotExist(err error) bool
}

// RealFileSystem 真实的文件系统实现
type RealFileSystem struct{}

// Stat 获取文件信息
func (r *RealFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// ReadFile 读取文件内容
func (r *RealFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

// WriteFile 写入文件
func (r *RealFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

// ReadDir 读取目录
func (r *RealFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

// MkdirAll 创建目录
func (r *RealFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// RemoveAll 删除目录或文件
func (r *RealFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// IsNotExist 检查错误是否为文件不存在
func (r *RealFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

// MockFileSystem 用于测试的模拟文件系统
type MockFileSystem struct {
	StatFunc      func(name string) (os.FileInfo, error)
	ReadFileFunc  func(name string) ([]byte, error)
	WriteFileFunc func(name string, data []byte, perm os.FileMode) error
	ReadDirFunc   func(name string) ([]fs.DirEntry, error)
	MkdirAllFunc  func(path string, perm os.FileMode) error
	RemoveAllFunc func(path string) error
}

// Stat 获取文件信息
func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc(name)
	}
	return nil, os.ErrNotExist
}

// ReadFile 读取文件内容
func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(name)
	}
	return nil, os.ErrNotExist
}

// WriteFile 写入文件
func (m *MockFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(name, data, perm)
	}
	return nil
}

// ReadDir 读取目录
func (m *MockFileSystem) ReadDir(name string) ([]fs.DirEntry, error) {
	if m.ReadDirFunc != nil {
		return m.ReadDirFunc(name)
	}
	return nil, os.ErrNotExist
}

// MkdirAll 创建目录
func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if m.MkdirAllFunc != nil {
		return m.MkdirAllFunc(path, perm)
	}
	return nil
}

// RemoveAll 删除目录或文件
func (m *MockFileSystem) RemoveAll(path string) error {
	if m.RemoveAllFunc != nil {
		return m.RemoveAllFunc(path)
	}
	return nil
}

// IsNotExist 检查错误是否为文件不存在
func (m *MockFileSystem) IsNotExist(err error) bool {
	return err == os.ErrNotExist
}

// NewRealFileSystem 创建真实文件系统实例
func NewRealFileSystem() FileSystem {
	return &RealFileSystem{}
}

// Path 路径操作辅助函数
type Path interface {
	Join(elem ...string) string
	Dir(path string) string
	Base(path string) string
}

// RealPath 真实路径操作
type RealPath struct{}

// Join 连接路径
func (r *RealPath) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Dir 返回路径的目录部分
func (r *RealPath) Dir(path string) string {
	return filepath.Dir(path)
}

// Base 返回路径的最后一部分
func (r *RealPath) Base(path string) string {
	return filepath.Base(path)
}

// NewRealPath 创建真实路径操作实例
func NewRealPath() Path {
	return &RealPath{}
}
