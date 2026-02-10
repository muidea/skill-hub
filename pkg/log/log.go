package log

import (
	"fmt"
	"log/slog"
	"os"
)

// Logger 日志接口
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// ConsoleLogger 控制台日志实现
type ConsoleLogger struct {
	level slog.Level
}

// NewConsoleLogger 创建控制台日志器
func NewConsoleLogger(level slog.Level) *ConsoleLogger {
	return &ConsoleLogger{level: level}
}

// Debug 输出调试日志
func (l *ConsoleLogger) Debug(msg string, args ...any) {
	if l.level <= slog.LevelDebug {
		fmt.Printf("🔍 DEBUG: %s", msg)
		if len(args) > 0 {
			fmt.Printf(" %v", args)
		}
		fmt.Println()
	}
}

// Info 输出信息日志
func (l *ConsoleLogger) Info(msg string, args ...any) {
	if l.level <= slog.LevelInfo {
		fmt.Printf("ℹ️  INFO: %s", msg)
		if len(args) > 0 {
			fmt.Printf(" %v", args)
		}
		fmt.Println()
	}
}

// Warn 输出警告日志
func (l *ConsoleLogger) Warn(msg string, args ...any) {
	if l.level <= slog.LevelWarn {
		fmt.Printf("⚠️  WARN: %s", msg)
		if len(args) > 0 {
			fmt.Printf(" %v", args)
		}
		fmt.Println()
	}
}

// Error 输出错误日志
func (l *ConsoleLogger) Error(msg string, args ...any) {
	if l.level <= slog.LevelError {
		fmt.Printf("❌ ERROR: %s", msg)
		if len(args) > 0 {
			fmt.Printf(" %v", args)
		}
		fmt.Println()
	}
}

// SimpleLogger 简单日志包装器（向后兼容）
type SimpleLogger struct{}

// Printf 格式化输出
func (l *SimpleLogger) Printf(format string, args ...any) {
	fmt.Printf(format, args...)
}

// Println 换行输出
func (l *SimpleLogger) Println(args ...any) {
	fmt.Println(args...)
}

// Print 输出
func (l *SimpleLogger) Print(args ...any) {
	fmt.Print(args...)
}

// Default 默认日志器
var Default Logger = NewConsoleLogger(slog.LevelInfo)

// SetDefault 设置默认日志器
func SetDefault(logger Logger) {
	Default = logger
}

// Debug 使用默认日志器输出调试日志
func Debug(msg string, args ...any) {
	Default.Debug(msg, args...)
}

// Info 使用默认日志器输出信息日志
func Info(msg string, args ...any) {
	Default.Info(msg, args...)
}

// Warn 使用默认日志器输出警告日志
func Warn(msg string, args ...any) {
	Default.Warn(msg, args...)
}

// Error 使用默认日志器输出错误日志
func Error(msg string, args ...any) {
	Default.Error(msg, args...)
}

// StdLogger 标准输出日志器（用于CLI命令）
var StdLogger = &SimpleLogger{}

// NewSlogLogger 创建slog日志器
func NewSlogLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}
