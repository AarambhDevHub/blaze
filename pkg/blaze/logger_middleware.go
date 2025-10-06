package blaze

import (
	"fmt"
	"time"
)

// LoggerMiddlewareConfig configures the logger middleware behavior
// Provides fine-grained control over what information is logged for each request
//
// Logging Philosophy:
//   - Development: Log everything for debugging
//   - Production: Log essential information, exclude sensitive data
//   - Performance: Balance information vs overhead
//
// Security Considerations:
//   - Never log sensitive headers (Authorization, Cookie, API keys)
//   - Be cautious with request/response bodies
//   - Filter query parameters that contain secrets
//   - Sanitize user input in logs
//
// Performance Considerations:
//   - Logging adds overhead to every request
//   - Request/response body logging is expensive
//   - Use SkipPaths for high-traffic health check endpoints
//   - Consider async logging for high-throughput applications
//
// Common Use Cases:
//   - Request tracking and debugging
//   - Performance monitoring (slow requests)
//   - Audit trails for compliance
//   - API usage analytics
//   - Error investigation
type LoggerMiddlewareConfig struct {
	// Logger instance to use for logging
	// If nil, uses the global default logger
	// Allows custom logger implementations with different outputs
	Logger *Loggerlog

	// SkipPaths specifies paths to exclude from logging
	// Useful for health checks, metrics endpoints, or high-frequency requests
	// Exact path matching - must match complete path
	// Example: []string{"/health", "/metrics", "/favicon.ico"}
	SkipPaths []string

	// LogRequestBody enables logging of request body content
	// WARNING: Be very careful with sensitive data (passwords, tokens, PII)
	// Can be expensive for large payloads
	// Consider size limits and content type filtering
	// Default: false (security and performance)
	LogRequestBody bool

	// LogResponseBody enables logging of response body content
	// Can be very expensive for large responses
	// Useful for debugging but not recommended in production
	// Consider implementing size limits
	// Default: false (performance)
	LogResponseBody bool

	// LogQueryParams enables logging of URL query parameters
	// Useful for debugging API calls
	// Be careful with sensitive data in query strings
	// Consider filtering specific parameters
	// Default: true
	LogQueryParams bool

	// LogHeaders enables logging of request headers
	// Headers are filtered using ExcludeHeaders list
	// Useful for debugging client behavior
	// Always exclude sensitive headers
	// Default: false (security)
	LogHeaders bool

	// ExcludeHeaders lists headers to exclude from logging
	// Used when LogHeaders is enabled
	// Always include: Authorization, Cookie, Set-Cookie, API keys
	// Case-sensitive header name matching
	// Default: ["Authorization", "Cookie", "Set-Cookie"]
	ExcludeHeaders []string

	// CustomFields adds custom fields to every log entry
	// Function receives context and returns map of fields
	// Useful for adding user ID, tenant ID, correlation IDs
	// Example: func(c *Context) map[string]interface{} {
	//     return map[string]interface{}{
	//         "user_id": c.Locals("user_id"),
	//         "tenant": c.Locals("tenant"),
	//     }
	// }
	CustomFields func(*Context) map[string]interface{}

	// SlowRequestThreshold defines duration for slow request warnings
	// Requests taking longer than this are logged with warning level
	// Helps identify performance issues
	// Set to 0 to disable slow request detection
	// Recommended: 1-5 seconds depending on application
	// Default: 3 seconds
	SlowRequestThreshold time.Duration
}

// DefaultLoggerMiddlewareConfig returns sensible defaults for request logging
// Suitable for most production applications with security-conscious settings
//
// Default Configuration:
//   - Uses global logger instance
//   - Skips /health and /metrics endpoints
//   - Logs query parameters
//   - Excludes request/response bodies (security/performance)
//   - Excludes headers (security)
//   - Filters sensitive headers
//   - Slow request threshold: 3 seconds
//
// Production Checklist:
//   - Review SkipPaths for your health check endpoints
//   - Ensure sensitive headers are in ExcludeHeaders
//   - Consider custom fields for user/tenant identification
//   - Adjust SlowRequestThreshold based on SLAs
//
// Returns:
//   - LoggerMiddlewareConfig: Production-ready configuration
func DefaultLoggerMiddlewareConfig() LoggerMiddlewareConfig {
	return LoggerMiddlewareConfig{
		Logger:               GetDefaultLogger(),
		SkipPaths:            []string{"/health", "/metrics"},
		LogRequestBody:       false,
		LogResponseBody:      false,
		LogQueryParams:       true,
		LogHeaders:           false,
		ExcludeHeaders:       []string{"Authorization", "Cookie", "Set-Cookie"},
		SlowRequestThreshold: 3 * time.Second,
	}
}

// LoggerMiddleware creates request logging middleware with default configuration
// Logs every request with essential information: method, path, status, duration
//
// Logged Information (Default):
//   - Request ID (if set by RequestID middleware)
//   - HTTP method (GET, POST, etc.)
//   - Request path
//   - Client IP address
//   - User agent
//   - Query parameters
//   - Response status code
//   - Request duration in milliseconds
//
// Log Levels:
//   - INFO: Successful requests (2xx, 3xx)
//   - WARN: Client errors (4xx) and slow requests
//   - ERROR: Server errors (5xx)
//
// Execution Order:
//   - Before: Capture start time, log incoming request
//   - After: Calculate duration, log response with status
//
// Returns:
//   - MiddlewareFunc: Request logging middleware
//
// Example - Basic Usage:
//
//	app.Use(blaze.LoggerMiddleware())
//
// Example - After RequestID Middleware:
//
//	app.Use(blaze.RequestIDMiddleware())
//	app.Use(blaze.LoggerMiddleware())
func LoggerMiddleware() MiddlewareFunc {
	return LoggerMiddlewareWithConfig(DefaultLoggerMiddlewareConfig())
}

// LoggerMiddlewareWithConfig creates logging middleware with custom configuration
// Provides full control over logging behavior for different environments
//
// Configuration Scenarios:
//
// Development (Verbose):
//
//	config := blaze.DefaultLoggerMiddlewareConfig()
//	config.LogRequestBody = true
//	config.LogResponseBody = true
//	config.LogHeaders = true
//	config.SlowRequestThreshold = 1 * time.Second
//
// Production (Secure):
//
//	config := blaze.DefaultLoggerMiddlewareConfig()
//	config.LogRequestBody = false
//	config.LogResponseBody = false
//	config.LogHeaders = false
//	config.SkipPaths = []string{"/health", "/metrics", "/ready"}
//
// Debugging (Detailed):
//
//	config := blaze.DefaultLoggerMiddlewareConfig()
//	config.LogHeaders = true
//	config.ExcludeHeaders = []string{"Authorization", "Cookie"}
//	config.CustomFields = func(c *blaze.Context) map[string]interface{} {
//	    return map[string]interface{}{
//	        "session_id": c.Cookie("session_id"),
//	        "user_id": c.Locals("user_id"),
//	    }
//	}
//
// Parameters:
//   - config: Logger middleware configuration
//
// Returns:
//   - MiddlewareFunc: Configured logging middleware
//
// Example - Custom Configuration:
//
//	config := blaze.LoggerMiddlewareConfig{
//	    Logger: blaze.GetDefaultLogger(),
//	    SkipPaths: []string{"/health", "/metrics", "/favicon.ico"},
//	    LogQueryParams: true,
//	    LogHeaders: true,
//	    ExcludeHeaders: []string{"Authorization", "Cookie", "X-API-Key"},
//	    SlowRequestThreshold: 5 * time.Second,
//	    CustomFields: func(c *blaze.Context) map[string]interface{} {
//	        return map[string]interface{}{
//	            "user_id": c.Locals("user_id"),
//	            "tenant_id": c.Locals("tenant_id"),
//	            "trace_id": c.Header("X-Trace-ID"),
//	        }
//	    },
//	}
//	app.Use(blaze.LoggerMiddlewareWithConfig(config))
func LoggerMiddlewareWithConfig(config LoggerMiddlewareConfig) MiddlewareFunc {
	if config.Logger == nil {
		config.Logger = GetDefaultLogger()
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Check if path should be skipped
			for _, path := range config.SkipPaths {
				if c.Path() == path {
					return next(c)
				}
			}

			start := time.Now()
			requestID := c.GetUserValueString("request_id")

			// Create request logger with context
			reqLogger := config.Logger.With(
				"request_id", requestID,
				"method", c.Method(),
				"path", c.Path(),
				"ip", c.IP(),
				"user_agent", c.UserAgent(),
			)

			// Add query params if enabled
			if config.LogQueryParams {
				queryParams := make(map[string]string)
				c.QueryArgs().VisitAll(func(key, value []byte) {
					queryParams[string(key)] = string(value)
				})
				if len(queryParams) > 0 {
					reqLogger = reqLogger.With("query", queryParams)
				}
			}

			// Add headers if enabled
			if config.LogHeaders {
				headers := make(map[string]string)
				c.Request().Header.VisitAll(func(key, value []byte) {
					keyStr := string(key)
					// Exclude sensitive headers
					excluded := false
					for _, excludedHeader := range config.ExcludeHeaders {
						if keyStr == excludedHeader {
							excluded = true
							break
						}
					}
					if !excluded {
						headers[keyStr] = string(value)
					}
				})
				reqLogger = reqLogger.With("headers", headers)
			}

			// Add request body if enabled
			if config.LogRequestBody && len(c.Body()) > 0 {
				reqLogger = reqLogger.With("request_body", string(c.Body()))
			}

			// Add custom fields
			if config.CustomFields != nil {
				customFields := config.CustomFields(c)
				for key, value := range customFields {
					reqLogger = reqLogger.With(key, value)
				}
			}

			// Log incoming request
			reqLogger.Info("incoming request")

			// Execute handler
			err := next(c)

			// Calculate duration
			duration := time.Since(start)
			statusCode := c.Response().StatusCode()

			// Create response logger
			resLogger := reqLogger.With(
				"status", statusCode,
				"duration_ms", duration.Milliseconds(),
				"duration", duration.String(),
			)

			// Add response body if enabled and present
			if config.LogResponseBody {
				body := c.Response().Body()
				if len(body) > 0 && len(body) < 1024*10 { // Limit to 10KB
					resLogger = resLogger.With("response_body", string(body))
				}
			}

			// Log based on status code and duration
			logMsg := "request completed"

			// Check for slow requests
			if config.SlowRequestThreshold > 0 && duration > config.SlowRequestThreshold {
				resLogger = resLogger.With("slow_request", true)
				resLogger.Warn(fmt.Sprintf("slow request: %s", logMsg))
			} else if statusCode >= 500 {
				if err != nil {
					resLogger = resLogger.With("error", err.Error())
				}
				resLogger.Error(logMsg)
			} else if statusCode >= 400 {
				if err != nil {
					resLogger = resLogger.With("error", err.Error())
				}
				resLogger.Warn(logMsg)
			} else {
				resLogger.Info(logMsg)
			}

			return err
		}
	}
}

// AccessLogMiddleware creates Apache/Nginx style access log middleware
// Provides standardized logging format compatible with log analysis tools
//
// Apache Combined Log Format:
//
//	remote_addr - - [time] "method path protocol" status bytes "referer" "user_agent" duration_ms
//
// Log Fields:
//   - remote_addr: Client IP address
//   - time: Request timestamp (Apache format)
//   - method: HTTP method
//   - path: Request path
//   - protocol: HTTP protocol version
//   - status: Response status code
//   - bytes: Response body size in bytes
//   - referer: Referer header
//   - user_agent: User-Agent header
//   - duration_ms: Request duration in milliseconds
//
// Use Cases:
//   - Standard web server access logs
//   - Log aggregation and analysis
//   - Traffic monitoring
//   - Integration with existing log tools
//
// Parameters:
//   - logger: Logger instance (nil for default)
//
// Returns:
//   - MiddlewareFunc: Access log middleware
//
// Example:
//
//	app.Use(blaze.AccessLogMiddleware(nil))
//
// Example - Custom Logger:
//
//	fileLogger, _ := blaze.FileLogger("/var/log/app/access.log", config)
//	app.Use(blaze.AccessLogMiddleware(fileLogger))
func AccessLogMiddleware(logger *Loggerlog) MiddlewareFunc {
	if logger == nil {
		logger = GetDefaultLogger()
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			start := time.Now()

			// Execute handler
			err := next(c)

			// Apache Combined Log Format:
			// %h %l %u %t "%r" %>s %b "%{Referer}i" "%{User-agent}i"
			logger.Info("access",
				"remote_addr", c.IP(),
				"time", start.Format("02/Jan/2006:15:04:05 -0700"),
				"method", c.Method(),
				"path", c.Path(),
				"protocol", c.Protocol(),
				"status", c.Response().StatusCode(),
				"bytes", len(c.Response().Body()),
				"referer", c.Header("Referer"),
				"user_agent", c.UserAgent(),
				"duration_ms", time.Since(start).Milliseconds(),
			)

			return err
		}
	}
}

// ErrorLogMiddleware logs errors with full request context
// Provides detailed error information for debugging and monitoring
//
// Error Logging Features:
//   - Full request context (method, path, IP, user agent)
//   - Request ID for correlation
//   - HTTPError details (code, status, message)
//   - Internal error information (root cause)
//   - Error details and metadata
//
// Integration with Error Handling:
//   - Works with HTTPError for structured errors
//   - Captures error chain with internal errors
//   - Logs before error handler formats response
//
// Use Cases:
//   - Error investigation and debugging
//   - Error rate monitoring
//   - Alert triggers for critical errors
//   - Audit trails for failures
//
// Parameters:
//   - logger: Logger instance (nil for default)
//
// Returns:
//   - MiddlewareFunc: Error logging middleware
//
// Example - Basic Usage:
//
//	app.Use(blaze.ErrorLogMiddleware(nil))
//
// Example - With Error Tracking Service:
//
//	logger := blaze.GetDefaultLogger()
//	app.Use(blaze.ErrorLogMiddleware(logger))
//	app.Use(blaze.ErrorHandlerMiddleware(config))
func ErrorLogMiddleware(logger *Loggerlog) MiddlewareFunc {
	if logger == nil {
		logger = GetDefaultLogger()
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			err := next(c)

			if err != nil {
				// Create error logger with full context
				errLogger := logger.With(
					"error", err.Error(),
					"request_id", c.GetUserValueString("request_id"),
					"method", c.Method(),
					"path", c.Path(),
					"ip", c.IP(),
				)

				// Log HTTPError with additional details
				if httpErr, ok := err.(*HTTPError); ok {
					errLogger = errLogger.With(
						"error_code", httpErr.Code,
						"status_code", httpErr.StatusCode,
					)
					if httpErr.Internal != nil {
						errLogger = errLogger.With("internal_error", httpErr.Internal.Error())
					}
					if httpErr.Details != nil {
						errLogger = errLogger.With("details", httpErr.Details)
					}
				}

				errLogger.Error("request error")
			}

			return err
		}
	}
}
