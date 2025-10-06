package blaze

import (
	"fmt"
	"strings"
)

// CORSOptions specifies comprehensive settings for the CORS middleware
// CORS (Cross-Origin Resource Sharing) is a security feature that controls
// which web applications can access resources from different origins
//
// Security Considerations:
//   - Use specific origins instead of "*" in production
//   - Don't enable credentials with wildcard origins
//   - Be cautious with exposed headers containing sensitive data
//   - Set reasonable MaxAge to balance security and performance
//
// Browser Behavior:
//   - Browsers send preflight OPTIONS requests for complex requests
//   - Simple requests (GET, POST with standard headers) don't require preflight
//   - CORS headers must be present on actual responses, not just preflight
type CORSOptions struct {
	// AllowedOrigins specifies which origins can access the resource
	// Use "*" for all origins (not recommended in production)
	// Or provide explicit list: []string{"https://example.com", "https://app.example.com"}
	// The middleware automatically handles Origin header matching
	AllowedOrigins []string

	// AllowedMethods specifies HTTP methods allowed for CORS requests
	// Common methods: GET, POST, PUT, DELETE, PATCH, OPTIONS
	// Default: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	// Set in Access-Control-Allow-Methods header
	AllowedMethods []string

	// AllowedHeaders specifies request headers allowed in CORS requests
	// Common headers: Content-Type, Authorization, X-Requested-With
	// Clients can only send these headers in cross-origin requests
	// Default: []string{"Content-Type", "Authorization", "X-Requested-With"}
	// Set in Access-Control-Allow-Headers header
	AllowedHeaders []string

	// ExposedHeaders specifies response headers exposed to browser JavaScript
	// By default, only simple response headers are accessible to JS
	// List additional headers clients can read: []string{"X-Custom-Header", "X-Request-ID"}
	// Set in Access-Control-Expose-Headers header
	ExposedHeaders []string

	// AllowCredentials indicates whether requests can include credentials
	// Credentials include: cookies, authorization headers, TLS client certificates
	// When true, AllowedOrigins cannot be "*" (security requirement)
	// Default: false
	// Set in Access-Control-Allow-Credentials header
	AllowCredentials bool

	// MaxAge specifies how long browsers can cache preflight responses in seconds
	// Caching reduces preflight requests, improving performance
	// Common values: 600 (10 minutes), 3600 (1 hour), 86400 (24 hours)
	// Default: 600 seconds
	// Set in Access-Control-Max-Age header
	MaxAge int
}

// DefaultCORSOptions returns secure and practical CORS defaults
// Suitable for development and testing environments
//
// Default Configuration:
//   - AllowedOrigins: ["*"] - Accepts all origins (change in production)
//   - AllowedMethods: Standard REST methods
//   - AllowedHeaders: Common headers for API requests
//   - ExposedHeaders: None
//   - AllowCredentials: false - Safer default
//   - MaxAge: 600 seconds (10 minutes)
//
// Production Checklist:
//   - Replace "*" with specific origins
//   - Set AllowCredentials appropriately
//   - Configure ExposedHeaders if needed
//   - Adjust MaxAge based on API stability
//
// Returns:
//   - CORSOptions: Default CORS configuration
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
// Implements the complete CORS specification with preflight support
//
// CORS Flow:
//  1. Browser sends preflight OPTIONS request (for complex requests)
//  2. Server responds with Access-Control-* headers
//  3. Browser validates headers and allows/blocks actual request
//  4. Actual request includes Origin header
//  5. Server includes CORS headers in response
//
// Preflight Requests:
//   - Method: OPTIONS
//   - Headers: Origin, Access-Control-Request-Method, Access-Control-Request-Headers
//   - Response: 204 No Content with CORS headers
//
// Actual Requests:
//   - Include Origin header
//   - Server validates origin against AllowedOrigins
//   - Response includes Access-Control-Allow-Origin
//
// Security Model:
//   - Same-origin requests don't need CORS
//   - Cross-origin requests require CORS headers
//   - Browser enforces CORS policy, not the server
//   - Server controls what's allowed via headers
//
// Parameters:
//   - opts: CORS configuration options
//
// Returns:
//   - MiddlewareFunc: CORS middleware for Blaze
//
// Example - Development (permissive):
//
//	app.Use(blaze.CORS(blaze.DefaultCORSOptions()))
//
// Example - Production (strict):
//
//	corsOpts := blaze.CORSOptions{
//	    AllowedOrigins: []string{"https://app.example.com", "https://www.example.com"},
//	    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
//	    AllowedHeaders: []string{"Content-Type", "Authorization"},
//	    ExposedHeaders: []string{"X-Request-ID"},
//	    AllowCredentials: true,
//	    MaxAge: 3600,
//	}
//	app.Use(blaze.CORS(corsOpts))
//
// Example - API with authentication:
//
//	corsOpts := blaze.CORSOptions{
//	    AllowedOrigins: []string{"https://app.example.com"},
//	    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
//	    AllowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key"},
//	    ExposedHeaders: []string{"X-RateLimit-Remaining", "X-RateLimit-Reset"},
//	    AllowCredentials: true,
//	    MaxAge: 7200,
//	}
//	app.Use(blaze.CORS(corsOpts))
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

// Helper for int to string conversion (minimal implementation without strconv dependency)
// Converts integer to string representation
// Used internally for MaxAge header value
//
// Parameters:
//   - v: Integer value to convert
//
// Returns:
//   - string: String representation of the integer
func intToString(v int) string {
	return fmt.Sprintf("%d", v)
}
