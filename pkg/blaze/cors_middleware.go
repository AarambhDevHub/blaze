package blaze

import (
	"fmt"
	"strings"
)

// CORSOptions specifies settings for the CORS middleware
type CORSOptions struct {
	AllowedOrigins   []string // "*" for all origins or explicit list
	AllowedMethods   []string // e.g., []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	AllowedHeaders   []string // e.g., []string{"Content-Type", "Authorization"}
	ExposedHeaders   []string // headers clients can access
	AllowCredentials bool     // true if cookies/credentials allowed
	MaxAge           int      // seconds browsers can cache preflight
}

// DefaultCORSOptions returns secure and practical defaults
func DefaultCORSOptions() CORSOptions {
	return CORSOptions{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposedHeaders:   nil,
		AllowCredentials: false,
		MaxAge:           600,
	}
}

// CORS returns a MiddlewareFunc for fully configurable CORS handling
func CORS(opts CORSOptions) MiddlewareFunc {
	origins := opts.AllowedOrigins
	methods := strings.Join(opts.AllowedMethods, ", ")
	headers := strings.Join(opts.AllowedHeaders, ", ")
	exposed := strings.Join(opts.ExposedHeaders, ", ")
	maxAge := ""
	if opts.MaxAge > 0 {
		maxAge = intToString(opts.MaxAge)
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			origin := c.Header("Origin")
			allowOrigin := ""

			// Compute Access-Control-Allow-Origin
			if len(origins) == 1 && origins[0] == "*" {
				allowOrigin = "*"
			} else if origin != "" {
				for _, o := range origins {
					if o == origin {
						allowOrigin = o
						break
					}
				}
			}

			if allowOrigin != "" {
				c.SetHeader("Access-Control-Allow-Origin", allowOrigin)
			}
			if opts.AllowCredentials {
				c.SetHeader("Access-Control-Allow-Credentials", "true")
			}
			if methods != "" {
				c.SetHeader("Access-Control-Allow-Methods", methods)
			}
			if headers != "" {
				c.SetHeader("Access-Control-Allow-Headers", headers)
			}
			if exposed != "" {
				c.SetHeader("Access-Control-Expose-Headers", exposed)
			}
			if maxAge != "" {
				c.SetHeader("Access-Control-Max-Age", maxAge)
			}

			// Preflight request - respond without further handler
			if c.Method() == "OPTIONS" {
				return c.Status(204).Text("")
			}
			return next(c)
		}
	}
}

// Helper for int to string conversion (no strconv dependency for minimal builds)
func intToString(v int) string {
	return fmt.Sprintf("%d", v)
}
