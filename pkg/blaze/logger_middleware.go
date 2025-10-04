package blaze

import (
	"fmt"
	"time"
)

// LoggerMiddlewareConfig configures the logger middleware
type LoggerMiddlewareConfig struct {
	// Logger instance to use
	Logger *Loggerlog

	// Skip logging for specific paths
	SkipPaths []string

	// Log request body (be careful with sensitive data)
	LogRequestBody bool

	// Log response body (can be expensive)
	LogResponseBody bool

	// Log query parameters
	LogQueryParams bool

	// Log headers (filtered for sensitive data)
	LogHeaders bool

	// Headers to exclude from logging
	ExcludeHeaders []string

	// Custom fields to add to every log
	CustomFields func(*Context) map[string]interface{}

	// Log slow requests (requests taking longer than this duration)
	SlowRequestThreshold time.Duration
}

// DefaultLoggerMiddlewareConfig returns default config
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

// LoggerMiddleware creates request logging middleware
func LoggerMiddleware() MiddlewareFunc {
	return LoggerMiddlewareWithConfig(DefaultLoggerMiddlewareConfig())
}

// LoggerMiddlewareWithConfig creates logging middleware with custom config
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

// ErrorLogMiddleware logs errors with full context
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
