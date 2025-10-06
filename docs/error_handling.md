# Error Handling Guide

Complete guide to error handling in the Blaze web framework, covering error types, middleware, and best practices.

## Table of Contents

- [Overview](#overview)
- [Error Types](#error-types)
- [Error Middleware](#error-middleware)
- [Creating Errors](#creating-errors)
- [Error Responses](#error-responses)
- [Configuration](#configuration)
- [Best Practices](#best-practices)
- [Examples](#examples)

## Overview

Blaze provides a comprehensive error handling system with:

- Structured HTTP error types
- Automatic panic recovery
- Validation error handling
- Configurable error responses
- Stack trace support
- Custom error handlers

## Error Types

### HTTPError

Core error type that represents HTTP errors with status codes and metadata.

```go
type HTTPError struct {
    StatusCode int                    // HTTP status code
    Code       ErrorCode              // Error code for client identification
    Message    string                 // Human-readable error message
    Details    interface{}            // Additional error details
    Internal   error                  // Internal error (hidden in production)
    Stack      []StackFrame           // Stack trace (development only)
    Path       string                 // Request path
    Method     string                 // HTTP method
    RequestID  string                 // Request ID for tracing
    Timestamp  time.Time              // Error timestamp
    Metadata   map[string]interface{} // Custom metadata
}
```

### Error Codes

Predefined error codes for common scenarios:

```go
const (
    ErrCodeBadRequest        ErrorCode = "BAD_REQUEST"
    ErrCodeUnauthorized      ErrorCode = "UNAUTHORIZED"
    ErrCodeForbidden         ErrorCode = "FORBIDDEN"
    ErrCodeNotFound          ErrorCode = "NOT_FOUND"
    ErrCodeMethodNotAllowed  ErrorCode = "METHOD_NOT_ALLOWED"
    ErrCodeConflict          ErrorCode = "CONFLICT"
    ErrCodeValidation        ErrorCode = "VALIDATION_ERROR"
    ErrCodeRateLimit         ErrorCode = "RATE_LIMIT_EXCEEDED"
    ErrCodeInternalServer    ErrorCode = "INTERNAL_SERVER_ERROR"
    ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)
```

### ValidationErrors

Special error type for validation failures:

```go
type ValidationErrors struct {
    Errors []ValidationError `json:"errors"`
}

type ValidationError struct {
    Field   string      `json:"field"`
    Tag     string      `json:"tag"`
    Value   interface{} `json:"value,omitempty"`
    Message string      `json:"message"`
}
```

## Error Middleware

### Error Handler Middleware

Centralized error handling that converts errors to JSON responses:

```go
// Basic usage
app.Use(blaze.ErrorHandlerMiddleware(nil)) // Uses defaults

// Custom configuration
config := blaze.ErrorHandlerConfig{
    IncludeStackTrace:    true,  // Include stack traces (development)
    LogErrors:            true,  // Automatically log errors
    HideInternalErrors:   true,  // Hide internal details (production)
    Logger:               customLogger,
    CustomHandler:        customErrorHandler,
}
app.Use(blaze.ErrorHandlerMiddleware(&config))
```

### Recovery Middleware

Recovers from panics and converts them to errors:

```go
// Automatic panic recovery
app.Use(blaze.RecoveryMiddleware(nil))

// With stack traces
config := blaze.DevelopmentErrorHandlerConfig()
app.Use(blaze.RecoveryMiddleware(&config))
```

### Validation Middleware

Automatically handles validation errors:

```go
// Catches ValidationErrors and returns 400
app.Use(blaze.ValidationMiddleware())
```

## Creating Errors

### Standard HTTP Errors

```go
// 400 Bad Request
return blaze.ErrBadRequest("Invalid input format")

// 401 Unauthorized
return blaze.ErrUnauthorized("Authentication required")

// 403 Forbidden
return blaze.ErrForbidden("Access denied")

// 404 Not Found
return blaze.ErrNotFound("Resource not found")

// 409 Conflict
return blaze.ErrConflict("Email already exists")

// 429 Too Many Requests
return blaze.ErrTooManyRequests("Rate limit exceeded")

// 500 Internal Server Error
return blaze.ErrInternalServer("Database connection failed")

// 503 Service Unavailable
return blaze.ErrServiceUnavailable("Service temporarily unavailable")
```

### Custom HTTP Errors

```go
// Create custom error
httpErr := blaze.NewHTTPError(
    http.StatusTeapot,
    "TEAPOT",
    "I'm a teapot",
)

// With metadata
httpErr.WithMetadata("teapot_id", "123")

// With internal error
httpErr.WithInternal(originalErr)

// With details
httpErr.WithDetails(map[string]interface{}{
    "reason": "coffee not supported",
})

// With stack trace
httpErr.WithStack(0)

return httpErr
```

### Error Chaining

```go
// Wrap errors
if err := database.Query(); err != nil {
    return blaze.ErrInternalServer("Query failed").
        WithInternal(err).
        WithMetadata("query", "SELECT * FROM users")
}

// Add context
err := processPayment()
if err != nil {
    return blaze.ErrBadRequest("Payment processing failed").
        WithInternal(err).
        WithDetails(map[string]interface{}{
            "payment_id": "123",
            "amount":     99.99,
        })
}
```

## Error Responses

### Standard Format

All errors return consistent JSON structure:

```go
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed",
    "details": [
      {
        "field": "email",
        "tag": "email",
        "value": "invalid-email",
        "message": "email must be a valid email address"
      }
    ]
  },
  "timestamp": "2024-01-01T12:00:00Z",
  "path": "/api/users",
  "method": "POST",
  "request_id": "abc123"
}
```

### Development vs Production

**Development** (with stack traces):

```go
{
  "success": false,
  "error": {
    "code": "INTERNAL_SERVER_ERROR",
    "message": "Database connection failed",
    "details": {
      "internal": "dial tcp: connection refused"
    },
    "stack": [
      {
        "file": "handler.go",
        "line": 42,
        "function": "createUser"
      }
    ]
  }
}
```

**Production** (sanitized):

```go
{
  "success": false,
  "error": {
    "code": "INTERNAL_SERVER_ERROR",
    "message": "An internal server error occurred"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Configuration

### Development Configuration

```go
config := blaze.DevelopmentErrorHandlerConfig()
// IncludeStackTrace: true
// LogErrors: true
// HideInternalErrors: false
app.Use(blaze.ErrorHandlerMiddleware(&config))
```

### Production Configuration

```go
config := blaze.DefaultErrorHandlerConfig()
// IncludeStackTrace: false
// LogErrors: true
// HideInternalErrors: true
app.Use(blaze.ErrorHandlerMiddleware(&config))
```

### Custom Error Handler

```go
config := blaze.ErrorHandlerConfig{
    CustomHandler: func(c *blaze.Context, err error) error {
        // Log to external service
        errorTracker.Report(err)
        
        // Return custom response
        return c.Status(500).JSON(blaze.Map{
            "error": "Something went wrong",
            "support": "contact@example.com",
        })
    },
}
app.Use(blaze.ErrorHandlerMiddleware(&config))
```

## Best Practices

### 1. Use Specific Error Types

```go
// ✅ Good - specific error
if !user.Active {
    return blaze.ErrForbidden("Account is inactive")
}

// ❌ Bad - generic error
if !user.Active {
    return fmt.Errorf("error")
}
```

### 2. Add Context to Errors

```go
// ✅ Good - with context
if err := db.DeleteUser(id); err != nil {
    return blaze.ErrInternalServer("Failed to delete user").
        WithInternal(err).
        WithMetadata("user_id", id)
}

// ❌ Bad - no context
if err := db.DeleteUser(id); err != nil {
    return err
}
```

### 3. Handle Validation Errors

```go
// ✅ Good - structured validation
var req CreateUserRequest
if err := c.BindJSON(&req); err != nil {
    return blaze.ErrBadRequest("Invalid JSON")
}
if err := c.Validate(&req); err != nil {
    return err // Auto-formatted by ValidationMiddleware
}

// ❌ Bad - manual validation
if req.Email == "" {
    return fmt.Errorf("email required")
}
```

### 4. Use Recovery Middleware

```go
// ✅ Good - panic protection
app.Use(blaze.RecoveryMiddleware(nil))

func handler(c *blaze.Context) error {
    // Panic is caught and converted to 500 error
    panic("something went wrong")
}
```

### 5. Hide Internal Details in Production

```go
// ✅ Good - sensitive info hidden
config := blaze.ErrorHandlerConfig{
    HideInternalErrors: true, // Production
}

// Internal error not exposed to client
return blaze.ErrInternalServer("Query failed").
    WithInternal(fmt.Errorf("SELECT * FROM users WHERE password='secret'"))
```

## Examples

### Complete Error Handling Setup

```go
func main() {
    app := blaze.New()
    
    // Middleware order matters
    app.Use(blaze.RequestIDMiddleware())
    app.Use(blaze.LoggerMiddleware())
    app.Use(blaze.RecoveryMiddleware(nil))
    app.Use(blaze.ValidationMiddleware())
    app.Use(blaze.ErrorHandlerMiddleware(nil))
    
    app.POST("/users", createUser)
    
    app.Listen(":8080")
}
```

### Handler with Error Handling

```go
func createUser(c *blaze.Context) error {
    var req CreateUserRequest
    
    // Binding error
    if err := c.BindJSON(&req); err != nil {
        return blaze.ErrBadRequest("Invalid request body")
    }
    
    // Validation error
    if err := c.Validate(&req); err != nil {
        return err // Handled by ValidationMiddleware
    }
    
    // Business logic error
    if exists, _ := db.EmailExists(req.Email); exists {
        return blaze.ErrConflict("Email already registered")
    }
    
    // Database error
    user, err := db.CreateUser(req)
    if err != nil {
        return blaze.ErrInternalServer("Failed to create user").
            WithInternal(err).
            WithMetadata("email", req.Email)
    }
    
    return c.Status(201).JSON(user)
}
```

### Custom Error Responses

```go
// Custom 404 handler
app.SetNotFoundHandler(func(c *blaze.Context) error {
    return blaze.ErrNotFound(fmt.Sprintf(
        "Route %s %s not found",
        c.Method(),
        c.Path(),
    ))
})

// Custom 405 handler
app.SetMethodNotAllowedHandler(func(c *blaze.Context) error {
    return blaze.ErrMethodNotAllowed(fmt.Sprintf(
        "Method %s not allowed for %s",
        c.Method(),
        c.Path(),
    ))
})
```

### Error Tracking Integration

```go
config := blaze.ErrorHandlerConfig{
    LogErrors: true,
    Logger: func(err error) {
        // Log to console
        log.Printf("Error: %v", err)
        
        // Send to error tracking service
        sentry.CaptureException(err)
        
        // Custom metrics
        metrics.IncrementErrorCount()
    },
}
app.Use(blaze.ErrorHandlerMiddleware(&config))
```

---

For more information:
- [Validation Guide](./validation.md)
- [Middleware Guide](./middleware.md)
- [API Reference](./api_reference.md)
