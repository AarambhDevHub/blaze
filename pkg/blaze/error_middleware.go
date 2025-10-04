package blaze

import (
	"fmt"
	"log"
	"runtime/debug"
)

// ErrorHandlerMiddleware creates middleware for centralized error handling
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

// NotFoundHandler creates a 404 handler
func NotFoundHandler() HandlerFunc {
	return func(c *Context) error {
		return ErrNotFound(fmt.Sprintf("Route %s %s not found", c.Method(), c.Path()))
	}
}

// MethodNotAllowedHandler creates a 405 handler
func MethodNotAllowedHandler() HandlerFunc {
	return func(c *Context) error {
		return ErrMethodNotAllowed(fmt.Sprintf("Method %s not allowed for %s", c.Method(), c.Path()))
	}
}
