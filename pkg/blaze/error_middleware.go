package blaze

import (
	"fmt"
	"log"
	"runtime/debug"
)

// ErrorHandlerMiddleware creates middleware for centralized error handling
// Handles errors returned by handlers and converts them to appropriate HTTP responses
//
// Error Handling Flow:
//  1. Execute next handler in chain
//  2. If handler returns error, process it
//  3. Determine error type (HTTPError, ValidationErrors, generic)
//  4. Format appropriate JSON response
//  5. Set HTTP status code
//  6. Log error if configured
//
// Supported Error Types:
//   - HTTPError: Structured errors with status codes and metadata
//   - ValidationErrors: Field validation errors from validator
//   - Standard errors: Converted to 500 Internal Server Error
//
// Response Format:
//
//	{
//	  "success": false,
//	  "error": {
//	    "code": "ERROR_CODE",
//	    "message": "Human-readable message",
//	    "details": {...},
//	    "stack": [...] // Only in development
//	  },
//	  "timestamp": "2024-01-01T00:00:00Z",
//	  "path": "/api/users",
//	  "method": "POST",
//	  "requestid": "uuid-here"
//	}
//
// Parameters:
//   - config: Error handler configuration
//
// Returns:
//   - MiddlewareFunc: Error handling middleware
//
// Example - Basic Usage:
//
//	app.Use(blaze.ErrorHandlerMiddleware(blaze.DefaultErrorHandlerConfig()))
//
// Example - Development Mode:
//
//	app.Use(blaze.ErrorHandlerMiddleware(blaze.DevelopmentErrorHandlerConfig()))
//
// Example - Custom Configuration:
//
//	config := &blaze.ErrorHandlerConfig{
//	    IncludeStackTrace: false,
//	    LogErrors: true,
//	    HideInternalErrors: true,
//	    Logger: func(err error) {
//	        logger.Error("http_error", zap.Error(err))
//	    },
//	}
//	app.Use(blaze.ErrorHandlerMiddleware(config))
//
// Example - Custom Error Handler:
//
//	config := blaze.DefaultErrorHandlerConfig()
//	config.CustomHandler = func(c *blaze.Context, err error) error {
//	    // Custom error handling logic
//	    log.Printf("Custom error handler: %v", err)
//	    return c.Status(500).JSON(blaze.Map{
//	        "error": "Something went wrong",
//	    })
//	}
//	app.Use(blaze.ErrorHandlerMiddleware(config))
func ErrorHandlerMiddleware(config *ErrorHandlerConfig) MiddlewareFunc {
	if config == nil {
		config = DefaultErrorHandlerConfig()
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Execute handler and capture error
			err := next(c)

			// Handle the error
			if err != nil {
				return HandleError(c, err, config)
			}

			return nil
		}
	}
}

// RecoveryMiddleware recovers from panics and converts them to errors
// Prevents application crashes by catching panics and converting to HTTP 500 errors
//
// Recovery Flow:
//  1. Set up panic recovery using defer/recover
//  2. Execute next handler
//  3. If panic occurs:
//     - Capture panic value and stack trace
//     - Convert panic to HTTPError
//     - Log panic with stack trace
//     - Return 500 Internal Server Error
//
// Panic Handling:
//   - String panics: Converted to error messages
//   - Error panics: Wrapped in HTTPError
//   - Other types: Formatted using %v
//   - Stack trace: Captured using runtime/debug
//
// Security Considerations:
//   - Panic details hidden in production (use HideInternalErrors)
//   - Stack traces only in development mode
//   - Sensitive information never exposed to clients
//
// Performance Impact:
//   - Minimal overhead (defer is fast in Go)
//   - Only activates on panic (rare occurrence)
//   - Stack trace capture only on panic
//
// Parameters:
//   - config: Error handler configuration
//
// Returns:
//   - MiddlewareFunc: Panic recovery middleware
//
// Example - Basic Usage:
//
//	app.Use(blaze.RecoveryMiddleware(blaze.DefaultErrorHandlerConfig()))
//
// Example - Development Mode:
//
//	config := blaze.DevelopmentErrorHandlerConfig()
//	app.Use(blaze.RecoveryMiddleware(config))
//
// Example - Custom Panic Handling:
//
//	config := blaze.DefaultErrorHandlerConfig()
//	config.Logger = func(err error) {
//	    log.Printf("PANIC RECOVERED: %v", err)
//	    // Send to error tracking service
//	    sentry.CaptureException(err)
//	}
//	app.Use(blaze.RecoveryMiddleware(config))
func RecoveryMiddleware(config *ErrorHandlerConfig) MiddlewareFunc {
	if config == nil {
		config = DefaultErrorHandlerConfig()
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					// Log panic
					log.Printf("[PANIC RECOVERED] %v\n%s", r, debug.Stack())

					// Convert panic to error
					var panicErr error
					switch x := r.(type) {
					case string:
						panicErr = fmt.Errorf("panic: %s", x)
					case error:
						panicErr = fmt.Errorf("panic: %w", x)
					default:
						panicErr = fmt.Errorf("panic: %v", x)
					}

					// Create HTTP error
					httpErr := ErrInternalServerWithInternal("A panic occurred", panicErr)

					if config.IncludeStackTrace {
						httpErr.WithStack(0)
					}

					// Handle error
					err = HandleError(c, httpErr, config)
				}
			}()

			return next(c)
		}
	}
}

// NotFoundHandler creates a 404 handler for undefined routes
// Returns a structured 404 error when no route matches the request
//
// Response Format:
//
//	{
//	  "success": false,
//	  "error": {
//	    "code": "NOT_FOUND",
//	    "message": "Route GET /invalid not found"
//	  },
//	  "timestamp": "2024-01-01T00:00:00Z",
//	  "path": "/invalid",
//	  "method": "GET"
//	}
//
// Usage:
//
//	app.SetNotFoundHandler(blaze.NotFoundHandler())
//
// Returns:
//   - HandlerFunc: Handler for 404 errors
func NotFoundHandler() HandlerFunc {
	return func(c *Context) error {
		return ErrNotFound(fmt.Sprintf("Route %s %s not found", c.Method(), c.Path()))
	}
}

// MethodNotAllowedHandler creates a 405 handler for invalid methods
// Returns a structured 405 error when method is not allowed for the route
//
// Response Format:
//
//	{
//	  "success": false,
//	  "error": {
//	    "code": "METHOD_NOT_ALLOWED",
//	    "message": "Method POST not allowed for /api/users"
//	  },
//	  "timestamp": "2024-01-01T00:00:00Z",
//	  "path": "/api/users",
//	  "method": "POST"
//	}
//
// Usage:
//
//	app.SetMethodNotAllowedHandler(blaze.MethodNotAllowedHandler())
//
// Returns:
//   - HandlerFunc: Handler for 405 errors
func MethodNotAllowedHandler() HandlerFunc {
	return func(c *Context) error {
		return ErrMethodNotAllowed(fmt.Sprintf("Method %s not allowed for %s", c.Method(), c.Path()))
	}
}
