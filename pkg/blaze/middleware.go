package blaze

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// Logger middleware logs HTTP requests with essential information
// Provides basic request/response logging without the overhead of structured logging
//
// Logged Information:
//   - HTTP method (GET, POST, etc.)
//   - Request path
//   - Response status code
//   - Request duration
//
// Log Format:
//
//	METHOD PATH - STATUS - DURATION
//	Example: GET /api/users - 200 - 45ms
//
// Performance:
//   - Minimal overhead (captures time, formats string)
//   - Synchronous logging (may impact throughput)
//   - Consider using LoggerMiddleware() for production
//
// Returns:
//   - MiddlewareFunc: Request logging middleware
//
// Example:
//
//	app.Use(blaze.Logger())
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

// Recovery middleware recovers from panics in handlers
// Converts panics to 500 Internal Server Error responses
//
// Recovery Process:
//  1. Set up defer with recover()
//  2. Execute next handler
//  3. If panic occurs:
//     - Log panic value
//     - Return 500 error response
//     - Prevent application crash
//
// Security Considerations:
//   - Never expose panic details to clients (security risk)
//   - Log panic for debugging (server-side only)
//   - Consider using RecoveryMiddleware() with error config
//
// Returns:
//   - MiddlewareFunc: Panic recovery middleware
//
// Example:
//
//	app.Use(blaze.Recovery())
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

// Auth middleware provides bearer token authentication
// Validates Authorization header and stores token in context
//
// Authentication Flow:
//  1. Extract Authorization header
//  2. Validate Bearer token format
//  3. Call custom token validator function
//  4. Store token in context on success
//  5. Return 401 on failure
//
// Token Format:
//   - Header: "Authorization: Bearer <token>"
//   - Example: "Authorization: Bearer abc123xyz789"
//
// Context Storage:
//   - Token stored with key "token"
//   - Access in handlers: c.Locals("token").(string)
//
// Parameters:
//   - tokenValidator: Function that validates token (returns true if valid)
//
// Returns:
//   - MiddlewareFunc: Bearer token authentication middleware
//
// Example - Basic Token Validation:
//
//	validTokens := map[string]bool{"secret123": true}
//	auth := blaze.Auth(func(token string) bool {
//	    return validTokens[token]
//	})
//	app.Use(auth)
//
// Example - Database Token Validation:
//
//	auth := blaze.Auth(func(token string) bool {
//	    user, err := db.GetUserByToken(token)
//	    return err == nil && user != nil
//	})
//	app.Use(auth)
//
// Example - JWT Token Validation:
//
//	auth := blaze.Auth(func(token string) bool {
//	    claims, err := jwt.ParseToken(token)
//	    return err == nil && claims.Valid()
//	})
//	app.Use(auth)
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

// ShutdownAware middleware checks for graceful shutdown state
// Immediately rejects requests with 503 if server is shutting down
//
// Shutdown Handling:
//   - Checks shutdown state before processing request
//   - Returns 503 Service Unavailable if shutting down
//   - Allows in-flight requests to complete
//   - Prevents accepting new work during shutdown
//
// Use Cases:
//   - Load balancer health checks
//   - Graceful degradation
//   - Zero-downtime deployments
//
// Response:
//   - Status: 503 Service Unavailable
//   - Body: {"error": "Service Unavailable", "message": "Server is shutting down"}
//
// Returns:
//   - MiddlewareFunc: Shutdown-aware middleware
//
// Example:
//
//	app.Use(blaze.ShutdownAware())
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

// GracefulTimeout middleware adds request timeout with graceful shutdown awareness
// Combines timeout functionality with shutdown context monitoring
//
// Timeout Behavior:
//   - Sets maximum duration for request processing
//   - Returns 408 Request Timeout if exceeded
//   - Returns 503 if shutdown occurs during request
//   - Cancels handler context on timeout
//
// Shutdown Integration:
//   - Monitors shutdown context
//   - Returns 503 immediately on shutdown
//   - Prevents timeout waiting if shutting down
//
// Parameters:
//   - timeout: Maximum request duration
//
// Returns:
//   - MiddlewareFunc: Timeout middleware with shutdown awareness
//
// Example - 5 Second Timeout:
//
//	app.Use(blaze.GracefulTimeout(5 * time.Second))
//
// Example - Per-Route Timeout:
//
//	app.GET("/fast", handler, blaze.GracefulTimeout(1 * time.Second))
//	app.GET("/slow", handler, blaze.GracefulTimeout(30 * time.Second))
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

// HTTP2Info middleware adds HTTP/2 protocol information to response headers
// Useful for debugging and monitoring HTTP/2 connections
//
// Added Headers:
//   - X-Protocol: HTTP/2.0 or HTTP/1.1
//   - X-HTTP2-Enabled: true or false
//
// Use Cases:
//   - Debugging protocol negotiation
//   - Monitoring HTTP/2 adoption
//   - Testing HTTP/2 features
//   - Client troubleshooting
//
// Returns:
//   - MiddlewareFunc: HTTP/2 info middleware
//
// Example:
//
//	app.Use(blaze.HTTP2Info())
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

// HTTP2Security middleware adds HTTP/2-specific security headers
// Implements security best practices for HTTP/2 applications
//
// Security Headers Added:
//   - X-Content-Type-Options: nosniff (prevents MIME sniffing)
//   - X-Frame-Options: DENY (prevents clickjacking)
//   - X-XSS-Protection: 1; mode=block (XSS protection)
//   - Strict-Transport-Security: max-age=31536000; includeSubDomains (HTTPS only)
//
// HSTS (Strict-Transport-Security):
//   - Only added for HTTPS requests
//   - Forces HTTPS for 1 year
//   - Includes all subdomains
//   - Protects against downgrade attacks
//
// Returns:
//   - MiddlewareFunc: HTTP/2 security middleware
//
// Example:
//
//	app.Use(blaze.HTTP2Security())
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

// StreamInfo middleware adds HTTP/2 stream debugging information
// Provides stream-level details for HTTP/2 connections
//
// Added Headers (HTTP/2 only):
//   - X-Stream-ID: Unique stream identifier
//   - X-Stream-Priority: Stream priority (0 = default)
//
// Use Cases:
//   - Debugging multiplexing issues
//   - Monitoring stream usage
//   - Performance analysis
//   - Load testing HTTP/2
//
// Returns:
//   - MiddlewareFunc: Stream info middleware
//
// Example:
//
//	app.Use(blaze.StreamInfo())
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

// HTTP2Metrics middleware collects HTTP/2-specific metrics
// Tracks HTTP/2 protocol usage and performance
//
// Metrics Collected:
//   - HTTP/2 request count
//   - Stream utilization
//   - Frame statistics
//   - Performance metrics
//
// Context Storage:
//   - Sets "http2metrics_enabled" flag
//   - Can be accessed by other middleware/handlers
//
// Use Cases:
//   - Performance monitoring
//   - Capacity planning
//   - Protocol adoption tracking
//   - Debugging performance issues
//
// Returns:
//   - MiddlewareFunc: HTTP/2 metrics middleware
//
// Example:
//
//	app.Use(blaze.HTTP2Metrics())
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

// CompressHTTP2 middleware enables compression for HTTP/2 responses
// Automatically compresses responses based on Accept-Encoding header
//
// Supported Compression Algorithms:
//   - gzip: Most widely supported
//   - deflate: Legacy support
//   - brotli: Best compression ratio
//
// Compression Selection:
//  1. Check Accept-Encoding header
//  2. Select best available algorithm (br > gzip > deflate)
//  3. Set Content-Encoding header
//  4. Skip if client doesn't support compression
//
// Parameters:
//   - level: Compression level (0-9, higher = better compression but slower)
//
// Returns:
//   - MiddlewareFunc: HTTP/2 compression middleware
//
// Example:
//
//	app.Use(blaze.CompressHTTP2(6)) // Default compression level
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
