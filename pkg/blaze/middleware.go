package blaze

import (
	"log"
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
