package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"

	"skill-hub/pkg/errors"
)

// LogLevel 日志级别
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Config 日志配置
type Config struct {
	Level     LogLevel
	Format    string // "text" 或 "json"
	Output    string // "stdout", "stderr", 或文件路径
	AddSource bool   // 是否添加源代码位置
}

// DefaultConfig 默认配置
var DefaultConfig = Config{
	Level:     LevelInfo,
	Format:    "text",
	Output:    "stderr",
	AddSource: false,
}

// Logger 包装slog.Logger
type Logger struct {
	*slog.Logger
	config Config
}

// NewLogger 创建新的日志记录器
func NewLogger(config Config) (*Logger, error) {
	// 设置日志级别
	var level slog.Level
	switch config.Level {
	case LevelDebug:
		level = slog.LevelDebug
	case LevelInfo:
		level = slog.LevelInfo
	case LevelWarn:
		level = slog.LevelWarn
	case LevelError:
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// 设置输出
	var output *os.File
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// 尝试打开文件
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, errors.Wrap(err, "NewLogger: 打开日志文件失败")
		}
		output = file
	}

	// 创建handler选项
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if config.AddSource {
		opts.AddSource = true
	}

	// 创建handler
	var handler slog.Handler
	if config.Format == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	// 创建logger
	logger := &Logger{
		Logger: slog.New(handler),
		config: config,
	}

	// 设置为全局日志记录器
	errors.SetLogger(logger)

	return logger, nil
}

// WithContext 添加上下文到日志记录
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// 这里可以添加上下文信息，如请求ID、用户ID等
	return l
}

// WithOperation 添加操作名称到日志记录
func (l *Logger) WithOperation(operation string) *Logger {
	return &Logger{
		Logger: l.Logger.With("operation", operation),
		config: l.config,
	}
}

// WithFields 添加字段到日志记录
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{
		Logger: l.Logger.With(args...),
		config: l.config,
	}
}

// ErrorWithErr 记录错误和错误对象
func (l *Logger) ErrorWithErr(msg string, err error, args ...any) {
	if err == nil {
		return
	}

	// 添加错误信息
	allArgs := append([]any{"error", err.Error()}, args...)

	// 如果是AppError，添加更多信息
	if appErr, ok := err.(*errors.AppError); ok {
		allArgs = append(allArgs, "error_code", string(appErr.Code))
		allArgs = append(allArgs, "error_op", appErr.Op)

		if appErr.Details != nil {
			for k, v := range appErr.Details {
				allArgs = append(allArgs, "detail_"+k, v)
			}
		}
	}

	l.Error(msg, allArgs...)
}

// DebugWithCaller 记录调试信息并包含调用者信息
func (l *Logger) DebugWithCaller(msg string, args ...any) {
	if pc, file, line, ok := runtime.Caller(1); ok {
		funcName := runtime.FuncForPC(pc).Name()
		callerArgs := append([]any{"caller_file", file, "caller_line", line, "caller_func", funcName}, args...)
		l.Debug(msg, callerArgs...)
	} else {
		l.Debug(msg, args...)
	}
}

// InfoWithDuration 记录信息并包含持续时间
func (l *Logger) InfoWithDuration(msg string, startTime time.Time, args ...any) {
	duration := time.Since(startTime)
	allArgs := append([]any{"duration_ms", duration.Milliseconds()}, args...)
	l.Info(msg, allArgs...)
}

// Error 实现errors.Logger接口
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Warn 实现errors.Logger接口
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Info 实现errors.Logger接口
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Debug 实现errors.Logger接口
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Global logger instance
var globalLogger *Logger

// InitGlobalLogger 初始化全局日志记录器
func InitGlobalLogger(config Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetGlobalLogger 获取全局日志记录器
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		// 使用默认配置创建
		logger, _ := NewLogger(DefaultConfig)
		globalLogger = logger
	}
	return globalLogger
}

// DiscardLogger 返回一个丢弃所有输出的logger，用于测试
func DiscardLogger() *Logger {
	return &Logger{
		Logger: slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})),
		config: Config{Level: LevelError}, // 只记录错误级别，但输出被丢弃
	}
}
