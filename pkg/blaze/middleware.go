package blaze

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// Logger middleware logs requests
func Logger() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			start := time.Now()

			// Process request
			err := next(c)

			// Log request
			log.Printf("%s %s - %d - %v",
				c.Method(),
				c.Path(),
				c.Response().StatusCode(),
				time.Since(start),
			)

			return err
		}
	}
}

// Recovery middleware recovers from panics
func Recovery() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC: %v", r)
					err = c.Status(500).JSON(Map{
						"error": "Internal Server Error",
					})
				}
			}()

			return next(c)
		}
	}
}

// CORS middleware handles Cross-Origin Resource Sharing
func CORS(origins ...string) MiddlewareFunc {
	allowedOrigins := origins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			origin := c.Header("Origin")

			// Set CORS headers
			if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
				c.SetHeader("Access-Control-Allow-Origin", "*")
			} else {
				for _, allowedOrigin := range allowedOrigins {
					if origin == allowedOrigin {
						c.SetHeader("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}

			c.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			c.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.SetHeader("Access-Control-Allow-Credentials", "true")

			// Handle preflight request
			if c.Method() == "OPTIONS" {
				return c.Status(204).Text("")
			}

			return next(c)
		}
	}
}

// RateLimit middleware (simplified implementation)
func RateLimit(requests int, window time.Duration) MiddlewareFunc {
	// In a real implementation, you'd use a proper rate limiting algorithm
	// like token bucket or sliding window with Redis or in-memory store
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Simplified: just pass through for this example
			// Real implementation would check rate limits per IP
			return next(c)
		}
	}
}

// Auth middleware for bearer token authentication
func Auth(tokenValidator func(string) bool) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			auth := c.Header("Authorization")

			if auth == "" {
				return c.Status(401).JSON(Map{
					"error": "Authorization header required",
				})
			}

			// Extract Bearer token
			if len(auth) < 7 || auth[:7] != "Bearer " {
				return c.Status(401).JSON(Map{
					"error": "Invalid authorization format",
				})
			}

			token := auth[7:]

			if !tokenValidator(token) {
				return c.Status(401).JSON(Map{
					"error": "Invalid token",
				})
			}

			// Store token in context for later use
			c.SetLocals("token", token)

			return next(c)
		}
	}
}

// ShutdownAware middleware that cancels long-running operations during shutdown
func ShutdownAware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Check if we're already shutting down
			if c.IsShuttingDown() {
				return c.Status(503).JSON(Map{
					"error":   "Service Unavailable",
					"message": "Server is shutting down",
				})
			}

			return next(c)
		}
	}
}

// GracefulTimeout middleware adds timeout that respects shutdown context
func GracefulTimeout(timeout time.Duration) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			ctx, cancel := c.WithTimeout(timeout)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				done <- next(c)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				if c.IsShuttingDown() {
					return c.Status(503).JSON(Map{
						"error":   "Service Unavailable",
						"message": "Server is shutting down",
					})
				}
				return c.Status(408).JSON(Map{
					"error":   "Request Timeout",
					"message": "Request exceeded timeout limit",
				})
			}
		}
	}
}

// HTTP2Info middleware adds HTTP/2 information to response headers
func HTTP2Info() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Add HTTP/2 protocol information
			if c.Locals("http2_enabled").(bool) {
				c.SetHeader("X-Protocol", "HTTP/2.0")
				c.SetHeader("X-HTTP2-Enabled", "true")
			} else {
				c.SetHeader("X-Protocol", "HTTP/1.1")
				c.SetHeader("X-HTTP2-Enabled", "false")
			}

			return next(c)
		}
	}
}

// HTTP2Security middleware adds HTTP/2 specific security headers
func HTTP2Security() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Add HTTP/2 specific security headers
			c.SetHeader("X-Content-Type-Options", "nosniff")
			c.SetHeader("X-Frame-Options", "DENY")
			c.SetHeader("X-XSS-Protection", "1; mode=block")

			// Add Strict Transport Security for HTTPS/HTTP2
			if strings.HasPrefix(c.URI().String(), "https") {
				c.SetHeader("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			return next(c)
		}
	}
}

// StreamInfo middleware adds HTTP/2 stream information (for debugging)
func StreamInfo() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			if c.Locals("http2_enabled").(bool) {
				// Add stream information for debugging
				c.SetHeader("X-Stream-ID", fmt.Sprintf("%d", c.RequestCtx.ID()))
				c.SetHeader("X-Stream-Priority", "0") // Default priority
			}

			return next(c)
		}
	}
}

// HTTP2Metrics middleware for collecting HTTP/2 specific metrics
func HTTP2Metrics() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Collect HTTP/2 specific metrics
			if c.Locals("http2_enabled").(bool) {
				// Track stream count, frame count, etc.
				// This is a placeholder - implement actual metrics collection
				c.SetLocals("http2_metrics_enabled", true)
			}

			return next(c)
		}
	}
}

// CompressHTTP2 middleware for HTTP/2 specific compression
func CompressHTTP2(level int) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Enable compression for HTTP/2
			if c.Locals("http2_enabled").(bool) {
				// Set compression headers
				acceptEncoding := c.Header("Accept-Encoding")
				if strings.Contains(acceptEncoding, "gzip") {
					c.SetHeader("Content-Encoding", "gzip")
				} else if strings.Contains(acceptEncoding, "deflate") {
					c.SetHeader("Content-Encoding", "deflate")
				} else if strings.Contains(acceptEncoding, "br") {
					c.SetHeader("Content-Encoding", "br")
				}
			}

			return next(c)
		}
	}
}
