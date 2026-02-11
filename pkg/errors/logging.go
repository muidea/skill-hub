package errors

import (
	"log/slog"
	"os"
	"time"
)

// Logger 日志记录器接口
type Logger interface {
	Error(msg string, args ...any)
	Warn(msg string, args ...any)
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
}

// defaultLogger 默认日志记录器
var defaultLogger Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

// SetLogger 设置全局日志记录器
func SetLogger(logger Logger) {
	defaultLogger = logger
}

// GetLogger 获取全局日志记录器
func GetLogger() Logger {
	return defaultLogger
}

// LogError 记录错误到日志
func LogError(err error, operation string, additionalFields ...map[string]interface{}) {
	if err == nil {
		return
	}

	// 构建日志字段
	fields := map[string]interface{}{
		"operation": operation,
		"error":     err.Error(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 添加错误详情
	if appErr, ok := err.(*AppError); ok {
		fields["error_code"] = string(appErr.Code)
		fields["error_op"] = appErr.Op

		if appErr.Details != nil {
			for k, v := range appErr.Details {
				fields["detail_"+k] = v
			}
		}
	}

	// 添加额外字段
	if len(additionalFields) > 0 {
		for k, v := range additionalFields[0] {
			fields[k] = v
		}
	}

	// 转换字段为slog格式
	slogArgs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		slogArgs = append(slogArgs, k, v)
	}

	// 记录错误
	defaultLogger.Error("操作失败", slogArgs...)
}

// LogErrorWithContext 记录错误和上下文信息
func LogErrorWithContext(err error, operation string, context map[string]interface{}) {
	if context == nil {
		context = make(map[string]interface{})
	}
	context["operation"] = operation
	LogError(err, operation, context)
}

// LogWarning 记录警告
func LogWarning(msg string, operation string, fields ...map[string]interface{}) {
	logFields := map[string]interface{}{
		"operation": operation,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if len(fields) > 0 {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}

	slogArgs := make([]any, 0, len(logFields)*2)
	for k, v := range logFields {
		slogArgs = append(slogArgs, k, v)
	}

	defaultLogger.Warn(msg, slogArgs...)
}

// LogInfo 记录信息
func LogInfo(msg string, operation string, fields ...map[string]interface{}) {
	logFields := map[string]interface{}{
		"operation": operation,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if len(fields) > 0 {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}

	slogArgs := make([]any, 0, len(logFields)*2)
	for k, v := range logFields {
		slogArgs = append(slogArgs, k, v)
	}

	defaultLogger.Info(msg, slogArgs...)
}

// LogDebug 记录调试信息
func LogDebug(msg string, operation string, fields ...map[string]interface{}) {
	logFields := map[string]interface{}{
		"operation": operation,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if len(fields) > 0 {
		for k, v := range fields[0] {
			logFields[k] = v
		}
	}

	slogArgs := make([]any, 0, len(logFields)*2)
	for k, v := range logFields {
		slogArgs = append(slogArgs, k, v)
	}

	defaultLogger.Debug(msg, slogArgs...)
}

// ErrorMonitor 错误监控接口
type ErrorMonitor interface {
	RecordError(err error, tags map[string]string)
	RecordMetric(name string, value float64, tags map[string]string)
}

// defaultMonitor 默认错误监控（空实现）
type defaultMonitor struct{}

func (d *defaultMonitor) RecordError(err error, tags map[string]string) {
	// 空实现，可以集成到实际的监控系统
}

func (d *defaultMonitor) RecordMetric(name string, value float64, tags map[string]string) {
	// 空实现，可以集成到实际的监控系统
}

var globalMonitor ErrorMonitor = &defaultMonitor{}

// SetErrorMonitor 设置全局错误监控
func SetErrorMonitor(monitor ErrorMonitor) {
	globalMonitor = monitor
}

// GetErrorMonitor 获取全局错误监控
func GetErrorMonitor() ErrorMonitor {
	return globalMonitor
}

// MonitorError 记录错误到监控系统
func MonitorError(err error, tags map[string]string) {
	if err == nil {
		return
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	// 添加错误信息到标签
	if appErr, ok := err.(*AppError); ok {
		tags["error_code"] = string(appErr.Code)
		tags["error_op"] = appErr.Op
	}

	globalMonitor.RecordError(err, tags)
}

// MonitorMetric 记录指标到监控系统
func MonitorMetric(name string, value float64, tags map[string]string) {
	if tags == nil {
		tags = make(map[string]string)
	}
	globalMonitor.RecordMetric(name, value, tags)
}

// WithMonitoring 包装错误并记录到监控系统
func WithMonitoring(err error, operation string) error {
	if err == nil {
		return nil
	}

	// 记录到监控系统
	tags := map[string]string{
		"operation": operation,
	}
	MonitorError(err, tags)

	// 记录到日志
	LogError(err, operation)

	return err
}

// NewWithMonitoring 创建新错误并记录到监控系统
func NewWithMonitoring(operation string, code ErrorCode, msg string) error {
	err := NewWithCode(operation, code, msg)
	return WithMonitoring(err, operation)
}

// WrapWithMonitoring 包装错误并记录到监控系统
func WrapWithMonitoring(err error, operation string, code ErrorCode, msg string) error {
	if err == nil {
		return nil
	}

	wrappedErr := WrapWithCode(err, operation, code, msg)
	return WithMonitoring(wrappedErr, operation)
}
