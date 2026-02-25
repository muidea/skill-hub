package utils

import (
	"fmt"
	"os"

	"skill-hub/pkg/errors"
)

// WrapErr wraps an error with a formatted message
func WrapErr(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format+": %w", append(args, err)...)
}

// WrapErrWithCode wraps an error with a specific error code
func WrapErrWithCode(err error, op string, code errors.ErrorCode, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return errors.WrapWithCode(err, op, code, fmt.Sprintf(format, args...))
}

// GetCwdErr wraps os.Getwd errors with consistent message
func GetCwdErr(err error) error {
	return WrapErr(err, "获取当前目录失败")
}

// GetCwdErrWithCode wraps os.Getwd errors with error code
func GetCwdErrWithCode(err error, op string) error {
	return WrapErrWithCode(err, op, errors.ErrSystem, "获取当前目录失败")
}

// FileOpErr wraps file operation errors
func FileOpErr(err error, operation, path string) error {
	return WrapErr(err, "%s文件失败: %s", operation, path)
}

// FileOpErrWithCode wraps file operation errors with error code
func FileOpErrWithCode(err error, op string, operation, path string) error {
	return WrapErrWithCode(err, op, errors.ErrFileOperation, "%s文件失败: %s", operation, path)
}

// CreateDirErr wraps directory creation errors
func CreateDirErr(err error, dir string) error {
	return WrapErr(err, "创建目录失败: %s", dir)
}

// ReadFileErr wraps file reading errors
func ReadFileErr(err error, path string) error {
	return WrapErr(err, "读取文件失败: %s", path)
}

// WriteFileErr wraps file writing errors
func WriteFileErr(err error, path string) error {
	return WrapErr(err, "写入文件失败: %s", path)
}

// DeleteFileErr wraps file deletion errors
func DeleteFileErr(err error, path string) error {
	return WrapErr(err, "删除文件失败: %s", path)
}

// GitOpErr wraps git operation errors
func GitOpErr(err error, operation string) error {
	return WrapErr(err, "Git操作失败: %s", operation)
}

// ValidationErr wraps validation errors
func ValidationErr(err error, field string) error {
	return WrapErr(err, "验证失败: %s", field)
}

// NetworkErr wraps network errors
func NetworkErr(err error, operation string) error {
	return WrapErr(err, "网络操作失败: %s", operation)
}

// MustGetCwd gets current working directory or panics (for initialization only)
func MustGetCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(GetCwdErr(err))
	}
	return cwd
}
