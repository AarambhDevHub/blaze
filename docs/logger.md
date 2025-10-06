# Logger Documentation

Complete guide to structured logging in the Blaze web framework using Go's standard `log/slog` package.

## Table of Contents

- [Overview](#overview)
- [Configuration](#configuration)
- [Log Levels](#log-levels)
- [Log Formats](#log-formats)
- [Basic Usage](#basic-usage)
- [Logger Middleware](#logger-middleware)
- [Structured Logging](#structured-logging)
- [Child Loggers](#child-loggers)
- [File Logging](#file-logging)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Overview

Blaze uses Go's `log/slog` package for structured logging with additional features:

- **Structured logging** with key-value pairs
- **Multiple output formats** (JSON, text)
- **Log levels** (Debug, Info, Warn, Error)
- **Colored output** for development
- **Context-aware logging** with request information
- **Static fields** for application metadata
- **File and multi-output support**

## Configuration

### LoggerConfig

```go
type LoggerConfig struct {
    Level        LogLevel              // Minimum log level
    Format       LogFormat             // Output format (json/text)
    Output       io.Writer             // Output destination
    AddSource    bool                  // Include source location
    AddTimestamp bool                  // Include timestamps
    TimeFormat   string                // Timestamp format
    EnableColors bool                  // Enable ANSI colors (text only)
    AppName      string                // Application name
    AppVersion   string                // Application version
    Environment  string                // Environment (dev/staging/prod)
    StaticFields map[string]interface{} // Custom static fields
}
```

### Default Configuration

```go
config := blaze.DefaultLoggerConfig()
// Level: Info
// Format: JSON
// Output: stdout
// AddSource: false
// AddTimestamp: true
// TimeFormat: RFC3339
// EnableColors: false
```

### Development Configuration

```go
config := blaze.DevelopmentLoggerConfig()
// Level: Debug (shows everything)
// Format: Text (human-readable)
// Output: stdout
// AddSource: true (file and line numbers)
// EnableColors: true
// Environment: "development"
```

### Production Configuration

```go
config := blaze.ProductionLoggerConfig()
// Level: Info (excludes debug)
// Format: JSON (structured for aggregation)
// Output: stdout
// AddSource: false (performance)
// EnableColors: false
// Environment: "production"
```

## Log Levels

### Available Levels

```go
const (
    LogLevelDebug LogLevel = -4  // Detailed diagnostic information
    LogLevelInfo  LogLevel = 0   // General informational messages
    LogLevelWarn  LogLevel = 4   // Warning messages
    LogLevelError LogLevel = 8   // Error messages
)
```

### Level Filtering

- Setting level to `Info` filters out `Debug` messages
- Higher levels filter out lower levels
- Production typically uses `Info` or `Warn`
- Development typically uses `Debug`

## Log Formats

### JSON Format

Structured, machine-readable format for production:

```go
{
  "time": "2024-01-01T12:00:00Z",
  "level": "INFO",
  "msg": "request completed",
  "request_id": "abc123",
  "method": "GET",
  "path": "/api/users",
  "status": 200,
  "duration_ms": 45
}
```

### Text Format

Human-readable format for development:

```go
time=2024-01-01T12:00:00Z level=INFO msg="request completed" request_id=abc123 method=GET path=/api/users status=200 duration_ms=45
```

### Colored Text Format

Text format with ANSI color codes for terminal readability:

- **Debug**: Cyan
- **Info**: Green
- **Warn**: Yellow
- **Error**: Red

## Basic Usage

### Creating a Logger

```go
// Default configuration
logger := blaze.NewLogger(blaze.DefaultLoggerConfig())

// Development configuration
logger := blaze.NewLogger(blaze.DevelopmentLoggerConfig())

// Custom configuration
config := blaze.LoggerConfig{
    Level:        blaze.LogLevelDebug,
    Format:       blaze.LogFormatJSON,
    Output:       os.Stdout,
    AppName:      "my-api",
    AppVersion:   "1.0.0",
    Environment:  "production",
}
logger := blaze.NewLogger(config)
```

### Logging Methods

```go
// Debug level - detailed diagnostics
logger.Debug("processing request", "user_id", 123, "action", "create")

// Info level - normal operations
logger.Info("user created", "user_id", 123, "email", "user@example.com")

// Warn level - warnings
logger.Warn("slow query", "duration_ms", 1500, "query", "SELECT * FROM users")

// Error level - errors
logger.Error("database error", "error", err, "query", "INSERT INTO users")
```

### Context-Aware Logging

```go
// With context (for cancellation, deadlines)
ctx := context.Background()
logger.InfoContext(ctx, "processing batch", "batch_id", 456)
logger.ErrorContext(ctx, "batch failed", "error", err)
```

## Logger Middleware

### Basic Logger Middleware

```go
app.Use(blaze.LoggerMiddleware())
```

### Custom Configuration

```go
config := blaze.LoggerMiddlewareConfig{
    Logger:               customLogger,
    SkipPaths:            []string{"/health", "/metrics"},
    LogRequestBody:       false,
    LogResponseBody:      false,
    LogQueryParams:       true,
    LogHeaders:           false,
    ExcludeHeaders:       []string{"Authorization", "Cookie"},
    SlowRequestThreshold: 3 * time.Second,
    CustomFields: func(c *blaze.Context) map[string]interface{} {
        return map[string]interface{}{
            "tenant_id": c.GetLocal("tenant_id"),
        }
    },
}
app.Use(blaze.LoggerMiddlewareWithConfig(config))
```

### Middleware Features

- **Request logging**: Method, path, status, duration
- **Skip paths**: Exclude health checks, metrics
- **Query parameters**: Log request parameters
- **Headers**: Log headers (filtered for sensitive data)
- **Request/Response bodies**: Optional (careful with sensitive data)
- **Slow requests**: Warn for requests exceeding threshold
- **Custom fields**: Add application-specific context

### Access Log Middleware

Apache/Nginx style access logs:

```go
app.Use(blaze.AccessLogMiddleware(logger))
```

Output format:
```
192.168.1.1 - - [01/Jan/2024:12:00:00 -0700] "GET /api/users HTTP/1.1" 200 1234 "https://example.com" "Mozilla/5.0"
```

### Error Log Middleware

Logs only errors with full context:

```go
app.Use(blaze.ErrorLogMiddleware(logger))
```

## Structured Logging

### Key-Value Pairs

```
logger.Info("user login",
    "user_id", 123,
    "email", "user@example.com",
    "ip", "192.168.1.1",
    "user_agent", "Mozilla/5.0",
)
```

Output (JSON):
```
{
  "time": "2024-01-01T12:00:00Z",
  "level": "INFO",
  "msg": "user login",
  "user_id": 123,
  "email": "user@example.com",
  "ip": "192.168.1.1",
  "user_agent": "Mozilla/5.0"
}
```

### Structured Attributes

```go
import "log/slog"

logger.LogAttrs(ctx, blaze.LogLevelInfo, "event",
    slog.String("type", "user_login"),
    slog.Int("user_id", 123),
    slog.Time("timestamp", time.Now()),
)
```

## Child Loggers

### With Additional Fields

```go
// Create child logger with request ID
requestLogger := logger.With("request_id", "abc123", "user_id", 123)

// All logs from this logger include the fields
requestLogger.Info("processing request")
requestLogger.Info("request completed")
```

### With Groups

```go
// Group related fields
reqLogger := logger.WithGroup("request")
reqLogger.Info("handling",
    "method", "GET",
    "path", "/users",
)
```

Output (JSON):
```
{
  "msg": "handling",
  "request": {
    "method": "GET",
    "path": "/users"
  }
}
```

## File Logging

### Log to File

```go
logger, err := blaze.FileLogger("/var/log/app.log", config)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()
```

### Multiple Outputs

Log to both console and file:

```go
logFile, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
defer logFile.Close()

writer := blaze.MultiWriter(os.Stdout, logFile)
config := blaze.DefaultLoggerConfig()
config.Output = writer

logger := blaze.NewLogger(config)
```

### Log Rotation

For production, use external log rotation (logrotate):

```
# /etc/logrotate.d/myapp
/var/log/myapp/*.log {
    daily
    rotate 7
    compress
    delaycompress
    notifempty
    create 0644 www-data www-data
    sharedscripts
    postrotate
        systemctl reload myapp
    endscript
}
```

## Best Practices

### 1. Use Structured Logging

```go
// ✅ Good - structured with context
logger.Info("user created",
    "user_id", user.ID,
    "email", user.Email,
    "role", user.Role,
)

// ❌ Bad - unstructured string interpolation
logger.Info(fmt.Sprintf("User %d (%s) created with role %s", user.ID, user.Email, user.Role))
```

### 2. Choose Appropriate Log Levels

```go
// Debug - detailed diagnostics
logger.Debug("cache lookup", "key", key, "hit", true)

// Info - normal operations
logger.Info("server started", "port", 8080)

// Warn - potential issues
logger.Warn("slow query", "duration_ms", 1500)

// Error - failures
logger.Error("database error", "error", err)
```

### 3. Add Request Context

```go
func handler(c *blaze.Context) error {
    logger := c.Logger() // Request-scoped logger with context
    logger.Info("processing request")
    
    // Logs include request_id, method, path automatically
}
```

### 4. Use Child Loggers for Related Operations

```go
func processOrder(orderID int) error {
    logger := blaze.GetDefaultLogger().With("order_id", orderID)
    
    logger.Info("processing order")
    
    if err := validateOrder(orderID); err != nil {
        logger.Error("validation failed", "error", err)
        return err
    }
    
    logger.Info("order processed")
    return nil
}
```

### 5. Don't Log Sensitive Data

```go
// ✅ Good - sanitized
logger.Info("user login", "user_id", user.ID)

// ❌ Bad - contains sensitive data
logger.Info("user login",
    "user_id", user.ID,
    "password", user.Password,  // Never log passwords!
    "ssn", user.SSN,            // Never log PII!
)
```

### 6. Configure for Environment

```go
// Development
if config.Development {
    logger := blaze.NewLogger(blaze.DevelopmentLoggerConfig())
} else {
    // Production
    logger := blaze.NewLogger(blaze.ProductionLoggerConfig())
}
```

## Examples

### Complete Application Setup

```go
func main() {
    // Configure logger
    var loggerConfig blaze.LoggerConfig
    if os.Getenv("ENV") == "production" {
        loggerConfig = blaze.ProductionLoggerConfig()
    } else {
        loggerConfig = blaze.DevelopmentLoggerConfig()
    }
    
    loggerConfig.AppName = "my-api"
    loggerConfig.AppVersion = "1.0.0"
    loggerConfig.StaticFields = map[string]interface{}{
        "environment": os.Getenv("ENV"),
        "hostname":    getHostname(),
    }
    
    logger := blaze.NewLogger(loggerConfig)
    blaze.SetDefaultLogger(logger)
    
    // Create app
    app := blaze.New()
    
    // Add logger middleware
    app.Use(blaze.LoggerMiddleware())
    
    logger.Info("server starting", "port", 8080)
    log.Fatal(app.Listen(":8080"))
}
```

### Handler with Logging

```go
func createUser(c *blaze.Context) error {
    logger := c.Logger()
    
    var req CreateUserRequest
    if err := c.BindJSON(&req); err != nil {
        logger.Warn("invalid request", "error", err)
        return blaze.ErrBadRequest("Invalid request body")
    }
    
    logger.Info("creating user", "email", req.Email)
    
    user, err := db.CreateUser(req)
    if err != nil {
        logger.Error("failed to create user",
            "error", err,
            "email", req.Email,
        )
        return blaze.ErrInternalServer("Failed to create user")
    }
    
    logger.Info("user created successfully", "user_id", user.ID)
    return c.Status(201).JSON(user)
}
```

### Background Task Logging

```go
func backgroundWorker() {
    logger := blaze.GetDefaultLogger().With("worker", "email_sender")
    
    for {
        logger.Debug("checking for pending emails")
        
        emails, err := fetchPendingEmails()
        if err != nil {
            logger.Error("failed to fetch emails", "error", err)
            time.Sleep(time.Minute)
            continue
        }
        
        logger.Info("processing emails", "count", len(emails))
        
        for _, email := range emails {
            emailLogger := logger.With("email_id", email.ID)
            
            if err := sendEmail(email); err != nil {
                emailLogger.Error("failed to send", "error", err)
            } else {
                emailLogger.Info("email sent")
            }
        }
        
        time.Sleep(10 * time.Second)
    }
}
```

---

For more information:
- [Error Handling Guide](./error_handling.md)
- [Middleware Guide](./middleware.md)
- [Configuration Guide](./configuration.md)
