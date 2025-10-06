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
// Aligned with slog.Level for compatibility with standard library
//
// Log Level Usage:
//   - Debug: Detailed diagnostic information for development
//   - Info: General informational messages about application flow
//   - Warn: Warning messages for potentially harmful situations
//   - Error: Error messages for failure conditions
//
// Level Filtering:
//   - Setting level to Info filters out Debug messages
//   - Higher levels (Error) filter out lower levels (Info, Warn, Debug)
//   - Production typically uses Info or Warn
//   - Development typically uses Debug
type LogLevel int

const (
	// LogLevelDebug logs detailed diagnostic information
	// Use for: Development debugging, tracing execution flow
	// Level: -4 (slog.LevelDebug)
	LogLevelDebug LogLevel = -4

	// LogLevelInfo logs general informational messages
	// Use for: Normal application events, user actions, system events
	// Level: 0 (slog.LevelInfo)
	LogLevelInfo LogLevel = 0

	// LogLevelWarn logs warning messages
	// Use for: Potentially harmful situations, deprecated features
	// Level: 4 (slog.LevelWarn)
	LogLevelWarn LogLevel = 4

	// LogLevelError logs error messages
	// Use for: Error conditions, failures, exceptions
	// Level: 8 (slog.LevelError)
	LogLevelError LogLevel = 8
)

// LogFormat defines output format for log messages
// Determines how log entries are serialized
//
// Format Comparison:
//   - JSON: Machine-readable, structured, ideal for log aggregation
//   - Text: Human-readable, easier to read in development, less structured
//
// Use Cases:
//   - JSON: Production, log aggregation (ELK, Splunk), monitoring systems
//   - Text: Development, debugging, console output
type LogFormat string

const (
	// LogFormatJSON outputs structured JSON logs
	// Format: {"time":"2024-01-01T12:00:00Z","level":"INFO","msg":"message","key":"value"}
	// Best for: Production, log aggregation, parsing
	LogFormatJSON LogFormat = "json"

	// LogFormatText outputs human-readable text logs
	// Format: time=2024-01-01T12:00:00Z level=INFO msg=message key=value
	// Best for: Development, console output, readability
	LogFormatText LogFormat = "text"
)

// LoggerConfig holds comprehensive logger configuration
// Provides full control over logging behavior, format, and output
//
// Configuration Philosophy:
//   - Development: Verbose, colored, with source locations
//   - Production: Structured JSON, minimal verbosity, no colors
//   - Testing: Configurable output, captured logs
//
// Performance Considerations:
//   - AddSource adds overhead (file/line lookup)
//   - JSON format is faster than text format
//   - Buffered writers improve throughput
//   - Async logging reduces latency (implement externally)
type LoggerConfig struct {
	// Level specifies minimum log level to output
	// Messages below this level are discarded
	// Options: LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError
	Level LogLevel

	// Format specifies output format (json or text)
	// JSON for production/aggregation, Text for development
	Format LogFormat

	// Output specifies where logs are written
	// Can be: os.Stdout, os.Stderr, file, network writer, or custom io.Writer
	// Use MultiWriter for multiple outputs
	// Default: os.Stdout
	Output io.Writer

	// AddSource includes source code location in logs
	// Adds file path, line number, and function name
	// Useful for debugging but adds performance overhead
	// Format: source={"file":"main.go","line":42,"function":"main.handler"}
	// Default: false (performance)
	AddSource bool

	// AddTimestamp includes timestamps in log entries
	// Timestamps use format specified in TimeFormat
	// Should always be enabled for production
	// Default: true
	AddTimestamp bool

	// TimeFormat specifies timestamp format
	// Common formats:
	//   - time.RFC3339: "2006-01-02T15:04:05Z07:00" (ISO 8601)
	//   - "15:04:05": Simple time for development
	//   - time.RFC3339Nano: High precision timestamps
	// Default: time.RFC3339
	TimeFormat string

	// EnableColors enables ANSI color codes in text output
	// Only applies to text format, ignored for JSON
	// Makes logs easier to read in terminals
	// Should be disabled when logging to files or non-terminal outputs
	// Default: false (compatibility)
	EnableColors bool

	// AppName is added to all log entries as "app" field
	// Useful for identifying logs from different applications
	// Example: "app":"api-server"
	AppName string

	// AppVersion is added to all log entries as "version" field
	// Useful for correlating logs with deployments
	// Example: "version":"1.2.3"
	AppVersion string

	// Environment is added to all log entries as "env" field
	// Distinguishes logs from different environments
	// Common values: "development", "staging", "production"
	// Example: "env":"production"
	Environment string

	// StaticFields are custom fields added to every log entry
	// Useful for adding context like region, datacenter, hostname
	// Example: map[string]interface{}{
	//     "region": "us-east-1",
	//     "datacenter": "dc1",
	//     "hostname": os.Hostname(),
	// }
	StaticFields map[string]interface{}
}

// DefaultLoggerConfig returns default logger configuration
// Suitable for most applications with balanced settings
//
// Default Configuration:
//   - Level: Info (filters debug messages)
//   - Format: JSON (structured)
//   - Output: stdout
//   - Source locations: disabled (performance)
//   - Timestamps: enabled (RFC3339)
//   - Colors: disabled (compatibility)
//   - No static fields
//
// Returns:
//   - LoggerConfig: Default configuration
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

// DevelopmentLoggerConfig returns logger config optimized for development
// Provides maximum visibility for debugging
//
// Development Configuration:
//   - Level: Debug (shows everything)
//   - Format: Text (human-readable)
//   - Output: stdout
//   - Source locations: enabled (debugging)
//   - Timestamps: enabled (simple format)
//   - Colors: enabled (readability)
//   - Environment: "development"
//
// Returns:
//   - LoggerConfig: Development-optimized configuration
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

// ProductionLoggerConfig returns logger config optimized for production
// Balances information with performance and security
//
// Production Configuration:
//   - Level: Info (excludes debug logs)
//   - Format: JSON (structured for aggregation)
//   - Output: stdout
//   - Source locations: disabled (performance)
//   - Timestamps: enabled (RFC3339)
//   - Colors: disabled (not needed in production)
//   - Environment: "production"
//
// Returns:
//   - LoggerConfig: Production-optimized configuration
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

// Loggerlog wraps slog.Logger with additional functionality
// Provides Blaze-specific features on top of standard library logging
//
// Features:
//   - Structured logging with key-value pairs
//   - Multiple output formats (JSON, text)
//   - Colored output for terminals
//   - Context-aware logging
//   - Child loggers with additional fields
//   - Log level filtering
//
// Thread Safety:
//   - Safe for concurrent use
//   - Multiple goroutines can log simultaneously
//
// Performance:
//   - Built on slog (fast, efficient)
//   - Zero-allocation fast paths
//   - Minimal overhead for disabled levels
type Loggerlog struct {
	slog   *slog.Logger // Underlying slog logger
	config LoggerConfig // Logger configuration
}

// NewLogger creates a new structured logger with specified configuration
// Initializes slog with appropriate handler and applies all configuration
//
// Logger Creation Process:
//  1. Validate and set default values
//  2. Create appropriate handler (JSON or Text)
//  3. Configure handler options (level, source, timestamps)
//  4. Apply color handler if enabled
//  5. Add static fields (app name, version, environment)
//
// Parameters:
//   - config: Logger configuration
//
// Returns:
//   - *Loggerlog: Configured logger instance
//
// Example - Basic Logger:
//
//	config := blaze.DefaultLoggerConfig()
//	logger := blaze.NewLogger(config)
//
// Example - Development Logger:
//
//	logger := blaze.NewLogger(blaze.DevelopmentLoggerConfig())
//
// Example - Custom Logger:
//
//	config := blaze.LoggerConfig{
//	    Level: blaze.LogLevelInfo,
//	    Format: blaze.LogFormatJSON,
//	    Output: os.Stderr,
//	    AddSource: false,
//	    AddTimestamp: true,
//	    TimeFormat: time.RFC3339,
//	    AppName: "my-api",
//	    AppVersion: "1.0.0",
//	    Environment: "production",
//	    StaticFields: map[string]interface{}{
//	        "region": "us-west-2",
//	        "hostname": hostname,
//	    },
//	}
//	logger := blaze.NewLogger(config)
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
// Child loggers inherit parent configuration and add new fields
//
// Use Cases:
//   - Request-scoped logging with request ID
//   - User-scoped logging with user ID
//   - Component-specific logging
//
// Parameters:
//   - args: Key-value pairs to add (must be even number)
//
// Returns:
//   - *Loggerlog: Child logger with additional fields
//
// Example:
//
//	childLogger := logger.With("request_id", "abc123", "user_id", 42)
//	childLogger.Info("user action") // Includes request_id and user_id
func (l *Loggerlog) With(args ...interface{}) *Loggerlog {
	return &Loggerlog{
		slog:   l.slog.With(args...),
		config: l.config,
	}
}

// WithGroup creates a child logger with a named group
// Groups organize related fields under a namespace
//
// Parameters:
//   - name: Group name
//
// Returns:
//   - *Loggerlog: Logger with group
//
// Example:
//
//	reqLogger := logger.WithGroup("request")
//	reqLogger.Info("handling", "method", "GET", "path", "/users")
//	// Output: {"request":{"method":"GET","path":"/users"}}
func (l *Loggerlog) WithGroup(name string) *Loggerlog {
	return &Loggerlog{
		slog:   l.slog.WithGroup(name),
		config: l.config,
	}
}

// Debug logs a debug message with optional key-value pairs
// Debug messages are typically filtered out in production
//
// Parameters:
//   - msg: Log message
//   - args: Key-value pairs (must be even number)
//
// Example:
//
//	logger.Debug("processing request", "path", "/api/users", "method", "GET")
func (l *Loggerlog) Debug(msg string, args ...interface{}) {
	l.slog.Debug(msg, args...)
}

// Info logs an informational message with optional key-value pairs
// Info messages are the standard log level for normal operations
//
// Parameters:
//   - msg: Log message
//   - args: Key-value pairs (must be even number)
//
// Example:
//
//	logger.Info("request completed", "status", 200, "duration_ms", 45)
func (l *Loggerlog) Info(msg string, args ...interface{}) {
	l.slog.Info(msg, args...)
}

// Warn logs a warning message with optional key-value pairs
// Warning messages indicate potentially harmful situations
//
// Parameters:
//   - msg: Log message
//   - args: Key-value pairs (must be even number)
//
// Example:
//
//	logger.Warn("slow query", "duration_ms", 1500, "query", "SELECT * FROM users")
func (l *Loggerlog) Warn(msg string, args ...interface{}) {
	l.slog.Warn(msg, args...)
}

// Error logs an error message with optional key-value pairs
// Error messages indicate failure conditions
//
// Parameters:
//   - msg: Log message
//   - args: Key-value pairs (must be even number)
//
// Example:
//
//	logger.Error("database connection failed", "error", err, "retries", 3)
func (l *Loggerlog) Error(msg string, args ...interface{}) {
	l.slog.Error(msg, args...)
}

// DebugContext logs with context
// Context provides cancellation and deadlines
//
// Parameters:
//   - ctx: Context
//   - msg: Log message
//   - args: Key-value pairs
func (l *Loggerlog) DebugContext(ctx context.Context, msg string, args ...interface{}) {
	l.slog.DebugContext(ctx, msg, args...)
}

// InfoContext logs with context
//
// Parameters:
//   - ctx: Context
//   - msg: Log message
//   - args: Key-value pairs
func (l *Loggerlog) InfoContext(ctx context.Context, msg string, args ...interface{}) {
	l.slog.InfoContext(ctx, msg, args...)
}

// WarnContext logs with context
//
// Parameters:
//   - ctx: Context
//   - msg: Log message
//   - args: Key-value pairs
func (l *Loggerlog) WarnContext(ctx context.Context, msg string, args ...interface{}) {
	l.slog.WarnContext(ctx, msg, args...)
}

// ErrorContext logs with context
//
// Parameters:
//   - ctx: Context
//   - msg: Log message
//   - args: Key-value pairs
func (l *Loggerlog) ErrorContext(ctx context.Context, msg string, args ...interface{}) {
	l.slog.ErrorContext(ctx, msg, args...)
}

// LogAttrs logs with structured attributes
// Provides type-safe attribute logging
//
// Parameters:
//   - ctx: Context
//   - level: Log level
//   - msg: Log message
//   - attrs: Structured attributes
//
// Example:
//
//	logger.LogAttrs(ctx, blaze.LogLevelInfo, "event",
//	    slog.String("type", "user_login"),
//	    slog.Int("user_id", 123),
//	)
func (l *Loggerlog) LogAttrs(ctx context.Context, level LogLevel, msg string, attrs ...slog.Attr) {
	l.slog.LogAttrs(ctx, slog.Level(level), msg, attrs...)
}

// Underlying returns the underlying slog.Logger
// Allows direct access to slog functionality
//
// Returns:
//   - *slog.Logger: Underlying logger
func (l *Loggerlog) Underlying() *slog.Logger {
	return l.slog
}

// ColorHandler wraps a handler to add ANSI colors to text output
// Only used when EnableColors is true and format is text
//
// Color Scheme:
//   - Debug: Cyan
//   - Info: Green
//   - Warn: Yellow
//   - Error: Red
type ColorHandler struct {
	handler slog.Handler
}

// NewColorHandler creates a new color handler
// Wraps existing handler to add color functionality
//
// Parameters:
//   - handler: Handler to wrap
//
// Returns:
//   - *ColorHandler: Color-enabled handler
func NewColorHandler(handler slog.Handler) *ColorHandler {
	return &ColorHandler{handler: handler}
}

// Enabled implements slog.Handler
func (h *ColorHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

// Handle implements slog.Handler with colors
// Adds ANSI color codes to log messages based on level
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
// Maps log levels to terminal colors
//
// Parameters:
//   - level: slog.Level
//
// Returns:
//   - string: ANSI color code
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
// Affects package-level logging functions
//
// Parameters:
//   - logger: Logger to set as default
func SetDefaultLogger(logger *Loggerlog) {
	defaultLogger = logger
	slog.SetDefault(logger.slog)
}

// GetDefaultLogger returns the global logger
// Creates default logger if not initialized
//
// Returns:
//   - *Loggerlog: Global logger instance
func GetDefaultLogger() *Loggerlog {
	if defaultLogger == nil {
		defaultLogger = NewLogger(DefaultLoggerConfig())
	}
	return defaultLogger
}

// Package-level convenience functions
// Use global logger for simple logging

// Debug logs a debug message using global logger
func Debug(msg string, args ...interface{}) {
	GetDefaultLogger().Debug(msg, args...)
}

// Info logs an info message using global logger
func Info(msg string, args ...interface{}) {
	GetDefaultLogger().Info(msg, args...)
}

// Warn logs a warning message using global logger
func Warn(msg string, args ...interface{}) {
	GetDefaultLogger().Warn(msg, args...)
}

// Errorlog logs an error message using global logger
func Errorlog(msg string, args ...interface{}) {
	GetDefaultLogger().Error(msg, args...)
}

// FileLogger creates a logger that writes to a file
// Handles file creation and directory creation
//
// Parameters:
//   - filename: Path to log file
//   - config: Logger configuration
//
// Returns:
//   - *Loggerlog: File-backed logger
//   - error: File creation error or nil
//
// Example:
//
//	logger, err := blaze.FileLogger("/var/log/app.log", config)
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
// Useful for logging to both file and console
//
// Parameters:
//   - outputs: Multiple io.Writer destinations
//
// Returns:
//   - io.Writer: Multi-writer combining all outputs
//
// Example:
//
//	file, _ := os.Create("app.log")
//	writer := blaze.MultiWriter(os.Stdout, file)
//	config.Output = writer
func MultiWriter(outputs ...io.Writer) io.Writer {
	return io.MultiWriter(outputs...)
}

// LoggerFromContext extracts logger from context
// Returns default logger if not found in context
//
// Parameters:
//   - ctx: Context potentially containing logger
//
// Returns:
//   - *Loggerlog: Logger from context or default
func LoggerFromContext(ctx context.Context) *Loggerlog {
	if logger, ok := ctx.Value("logger").(*Loggerlog); ok {
		return logger
	}
	return GetDefaultLogger()
}

// ContextWithLogger adds logger to context
// Allows passing logger through context chain
//
// Parameters:
//   - ctx: Base context
//   - logger: Logger to add
//
// Returns:
//   - context.Context: Context with logger
//
// Example:
//
//	ctx := blaze.ContextWithLogger(context.Background(), logger)
//	// Later: logger := blaze.LoggerFromContext(ctx)
func ContextWithLogger(ctx context.Context, logger *Loggerlog) context.Context {
	return context.WithValue(ctx, "logger", logger)
}

// Helper function to get caller information
// Used internally for source location tracking
//
// Parameters:
//   - skip: Number of stack frames to skip
//
// Returns:
//   - file: Source file name
//   - line: Line number
//   - function: Function name
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
