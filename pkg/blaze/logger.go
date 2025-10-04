package blaze

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// LogLevel represents log severity levels
type LogLevel int

const (
	// Log levels (aligned with slog)
	LogLevelDebug LogLevel = -4
	LogLevelInfo  LogLevel = 0
	LogLevelWarn  LogLevel = 4
	LogLevelError LogLevel = 8
)

// LogFormat defines output format
type LogFormat string

const (
	LogFormatJSON LogFormat = "json"
	LogFormatText LogFormat = "text"
)

// LoggerConfig holds logger configuration
type LoggerConfig struct {
	// Log level (debug, info, warn, error)
	Level LogLevel

	// Output format (json or text)
	Format LogFormat

	// Output writer (default: os.Stdout)
	Output io.Writer

	// Include source code location
	AddSource bool

	// Include timestamps
	AddTimestamp bool

	// Custom time format (default: RFC3339)
	TimeFormat string

	// Enable colors for text format
	EnableColors bool

	// Application name (added to all logs)
	AppName string

	// Application version (added to all logs)
	AppVersion string

	// Environment (development, production, staging)
	Environment string

	// Additional static fields to include in all logs
	StaticFields map[string]interface{}
}

// DefaultLoggerConfig returns default logger configuration
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:        LogLevelInfo,
		Format:       LogFormatJSON,
		Output:       os.Stdout,
		AddSource:    false,
		AddTimestamp: true,
		TimeFormat:   time.RFC3339,
		EnableColors: false,
		StaticFields: make(map[string]interface{}),
	}
}

// DevelopmentLoggerConfig returns logger config for development
func DevelopmentLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:        LogLevelDebug,
		Format:       LogFormatText,
		Output:       os.Stdout,
		AddSource:    true,
		AddTimestamp: true,
		TimeFormat:   "15:04:05",
		EnableColors: true,
		Environment:  "development",
		StaticFields: make(map[string]interface{}),
	}
}

// ProductionLoggerConfig returns logger config for production
func ProductionLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Level:        LogLevelInfo,
		Format:       LogFormatJSON,
		Output:       os.Stdout,
		AddSource:    false,
		AddTimestamp: true,
		TimeFormat:   time.RFC3339,
		EnableColors: false,
		Environment:  "production",
		StaticFields: make(map[string]interface{}),
	}
}

// Loggerlog wraps slog.Loggerlog with additional functionality
type Loggerlog struct {
	slog   *slog.Logger
	config LoggerConfig
}

// NewLogger creates a new structured logger
func NewLogger(config LoggerConfig) *Loggerlog {
	// Set defaults
	if config.Output == nil {
		config.Output = os.Stdout
	}
	if config.TimeFormat == "" {
		config.TimeFormat = time.RFC3339
	}

	// Create handler options
	handlerOpts := &slog.HandlerOptions{
		Level:     slog.Level(config.Level),
		AddSource: config.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize timestamp format
			if a.Key == slog.TimeKey && config.TimeFormat != "" {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(config.TimeFormat))
				}
			}
			return a
		},
	}

	// Create handler based on format
	var handler slog.Handler
	if config.Format == LogFormatJSON {
		handler = slog.NewJSONHandler(config.Output, handlerOpts)
	} else {
		handler = slog.NewTextHandler(config.Output, handlerOpts)
	}

	// Wrap with custom handler for colors if enabled
	if config.EnableColors && config.Format == LogFormatText {
		handler = NewColorHandler(handler)
	}

	// Create logger
	logger := slog.New(handler)

	// Add static fields
	if config.AppName != "" {
		logger = logger.With("app", config.AppName)
	}
	if config.AppVersion != "" {
		logger = logger.With("version", config.AppVersion)
	}
	if config.Environment != "" {
		logger = logger.With("env", config.Environment)
	}

	// Add custom static fields
	for key, value := range config.StaticFields {
		logger = logger.With(key, value)
	}

	return &Loggerlog{
		slog:   logger,
		config: config,
	}
}

// With creates a child logger with additional attributes
func (l *Loggerlog) With(args ...interface{}) *Loggerlog {
	return &Loggerlog{
		slog:   l.slog.With(args...),
		config: l.config,
	}
}

// WithGroup creates a child logger with a named group
func (l *Loggerlog) WithGroup(name string) *Loggerlog {
	return &Loggerlog{
		slog:   l.slog.WithGroup(name),
		config: l.config,
	}
}

// Debug logs a debug message
func (l *Loggerlog) Debug(msg string, args ...interface{}) {
	l.slog.Debug(msg, args...)
}

// Info logs an info message
func (l *Loggerlog) Info(msg string, args ...interface{}) {
	l.slog.Info(msg, args...)
}

// Warn logs a warning message
func (l *Loggerlog) Warn(msg string, args ...interface{}) {
	l.slog.Warn(msg, args...)
}

// Error logs an error message
func (l *Loggerlog) Error(msg string, args ...interface{}) {
	l.slog.Error(msg, args...)
}

// DebugContext logs with context
func (l *Loggerlog) DebugContext(ctx context.Context, msg string, args ...interface{}) {
	l.slog.DebugContext(ctx, msg, args...)
}

// InfoContext logs with context
func (l *Loggerlog) InfoContext(ctx context.Context, msg string, args ...interface{}) {
	l.slog.InfoContext(ctx, msg, args...)
}

// WarnContext logs with context
func (l *Loggerlog) WarnContext(ctx context.Context, msg string, args ...interface{}) {
	l.slog.WarnContext(ctx, msg, args...)
}

// ErrorContext logs with context
func (l *Loggerlog) ErrorContext(ctx context.Context, msg string, args ...interface{}) {
	l.slog.ErrorContext(ctx, msg, args...)
}

// LogAttrs logs with structured attributes
func (l *Loggerlog) LogAttrs(ctx context.Context, level LogLevel, msg string, attrs ...slog.Attr) {
	l.slog.LogAttrs(ctx, slog.Level(level), msg, attrs...)
}

// Underlying returns the underlying slog.Logger
func (l *Loggerlog) Underlying() *slog.Logger {
	return l.slog
}

// ColorHandler wraps a handler to add colors to text output
type ColorHandler struct {
	handler slog.Handler
}

// NewColorHandler creates a new color handler
func NewColorHandler(handler slog.Handler) *ColorHandler {
	return &ColorHandler{handler: handler}
}

// Enabled implements slog.Handler
func (h *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle implements slog.Handler with colors
func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	// Add color based on level
	levelColor := getLevelColor(r.Level)
	r.Message = levelColor + r.Message + "\033[0m"
	return h.handler.Handle(ctx, r)
}

// WithAttrs implements slog.Handler
func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ColorHandler{handler: h.handler.WithAttrs(attrs)}
}

// WithGroup implements slog.Handler
func (h *ColorHandler) WithGroup(name string) slog.Handler {
	return &ColorHandler{handler: h.handler.WithGroup(name)}
}

// getLevelColor returns ANSI color code for log level
func getLevelColor(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "\033[36m" // Cyan
	case slog.LevelInfo:
		return "\033[32m" // Green
	case slog.LevelWarn:
		return "\033[33m" // Yellow
	case slog.LevelError:
		return "\033[31m" // Red
	default:
		return ""
	}
}

// Global logger instance
var defaultLogger *Loggerlog

// SetDefaultLogger sets the global default logger
func SetDefaultLogger(logger *Loggerlog) {
	defaultLogger = logger
	slog.SetDefault(logger.slog)
}

// GetDefaultLogger returns the global logger
func GetDefaultLogger() *Loggerlog {
	if defaultLogger == nil {
		defaultLogger = NewLogger(DefaultLoggerConfig())
	}
	return defaultLogger
}

// Package-level convenience functions
func Debug(msg string, args ...interface{}) {
	GetDefaultLogger().Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	GetDefaultLogger().Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	GetDefaultLogger().Warn(msg, args...)
}

func Errorlog(msg string, args ...interface{}) {
	GetDefaultLogger().Error(msg, args...)
}

// FileLogger creates a logger that writes to a file
func FileLogger(filename string, config LoggerConfig) (*Loggerlog, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	config.Output = file
	return NewLogger(config), nil
}

// MultiWriter creates a logger that writes to multiple outputs
func MultiWriter(outputs ...io.Writer) io.Writer {
	return io.MultiWriter(outputs...)
}

// LoggerFromContext extracts logger from context
func LoggerFromContext(ctx context.Context) *Loggerlog {
	if logger, ok := ctx.Value("logger").(*Loggerlog); ok {
		return logger
	}
	return GetDefaultLogger()
}

// ContextWithLogger adds logger to context
func ContextWithLogger(ctx context.Context, logger *Loggerlog) context.Context {
	return context.WithValue(ctx, "logger", logger)
}

// Helper function to get caller information
func getCaller(skip int) (file string, line int, function string) {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown", 0, "unknown"
	}

	fn := runtime.FuncForPC(pc)
	if fn != nil {
		function = fn.Name()
	}

	// Get just the filename, not the full path
	file = filepath.Base(file)

	return file, line, function
}
