package blaze

import (
	"fmt"
)

// BodyLimitConfig holds body size limit configuration.
// This configuration allows fine-grained control over request body size limits
// with support for path-based and content-type-based exemptions.
//
// Security Considerations:
//   - Always set reasonable limits based on your application needs
//   - Lower limits reduce memory consumption and DoS risk
//   - Consider different limits for different content types
//   - Monitor actual body sizes in production to tune limits
//
// Performance Impact:
//   - Body size checks add minimal overhead
//   - Content-Length header check is very fast
//   - Actual body reading only occurs when needed
type BodyLimitConfig struct {
	// MaxSize specifies the maximum allowed body size in bytes.
	// Requests exceeding this size will be rejected with 413 Payload Too Large.
	// Default: 4MB (4 * 1024 * 1024 bytes)
	//
	// Example values:
	//   - 1MB: 1024 * 1024
	//   - 10MB: 10 * 1024 * 1024
	//   - 100MB: 100 * 1024 * 1024
	MaxSize int64

	// ErrorMessage is the custom error message returned when limit is exceeded.
	// This message is included in the JSON error response.
	// Default: "Request body too large"
	ErrorMessage string

	// SkipPaths is a list of URL paths to exclude from body limit checks.
	// Use this for endpoints that legitimately need larger request bodies.
	//
	// Example:
	//   SkipPaths: []string{"/api/upload", "/api/import"}
	SkipPaths []string

	// SkipContentTypes is a list of content types to exclude from body limit checks.
	// Useful for allowing larger payloads for specific content types.
	//
	// Example:
	//   SkipContentTypes: []string{"multipart/form-data", "application/octet-stream"}
	SkipContentTypes []string
}

// DefaultBodyLimitConfig returns default body limit configuration.
// This configuration provides a secure starting point suitable for most web applications.
//
// Default Values:
//   - MaxSize: 4MB - Balances functionality with security
//   - ErrorMessage: "Request body too large" - Clear, user-friendly message
//   - SkipPaths: Empty - No paths excluded by default
//   - SkipContentTypes: Empty - All content types checked
//
// Usage:
//
//	config := DefaultBodyLimitConfig()
//	config.MaxSize = 10 * 1024 * 1024 // Increase to 10MB
//	app.Use(BodyLimitWithConfig(config))
func DefaultBodyLimitConfig() BodyLimitConfig {
	return BodyLimitConfig{
		MaxSize:          4 * 1024 * 1024, // 4MB
		ErrorMessage:     "Request body too large",
		SkipPaths:        []string{},
		SkipContentTypes: []string{},
	}
}

// BodyLimit middleware limits the size of request bodies.
// This is the simplest way to add body size limits with a single size value.
//
// How It Works:
//  1. Checks Content-Length header first (fast check)
//  2. Validates actual body size if Content-Length is not present
//  3. Returns 413 Payload Too Large if limit exceeded
//
// Parameters:
//   - maxSize: Maximum body size in bytes
//
// Returns:
//   - MiddlewareFunc: Middleware function to apply to routes
//
// Example:
//
//	// Limit all requests to 5MB
//	app.Use(BodyLimit(5 * 1024 * 1024))
//
//	// Apply to specific routes
//	app.POST("/api/upload", uploadHandler, BodyLimit(10 * 1024 * 1024))
func BodyLimit(maxSize int64) MiddlewareFunc {
	config := DefaultBodyLimitConfig()
	config.MaxSize = maxSize
	return BodyLimitWithConfig(config)
}

// BodyLimitWithConfig creates body limit middleware with custom configuration.
// This provides full control over body limit behavior including path and
// content-type based exemptions.
//
// Configuration Options:
//   - Custom error messages
//   - Path-based exemptions
//   - Content-type based exemptions
//   - Fine-grained size control
//
// Returns:
//   - MiddlewareFunc: Configured middleware function
//
// Example:
//
//	config := DefaultBodyLimitConfig()
//	config.MaxSize = 10 * 1024 * 1024
//	config.SkipPaths = []string{"/api/large-upload"}
//	config.SkipContentTypes = []string{"multipart/form-data"}
//	config.ErrorMessage = "File too large. Maximum size is 10MB"
//	app.Use(BodyLimitWithConfig(config))
func BodyLimitWithConfig(config BodyLimitConfig) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Skip if path is in skip list
			path := c.Path()
			for _, skipPath := range config.SkipPaths {
				if path == skipPath {
					return next(c)
				}
			}

			// Skip if content type is in skip list
			contentType := c.GetContentType()
			for _, skipType := range config.SkipContentTypes {
				if contentType == skipType {
					return next(c)
				}
			}

			// Get content length from header
			contentLength := c.RequestCtx.Request.Header.ContentLength()

			// Check if content length exceeds max size
			if contentLength > int(config.MaxSize) {
				return c.Status(413).JSON(Map{
					"error":      config.ErrorMessage,
					"maxSize":    config.MaxSize,
					"actualSize": contentLength,
				})
			}

			// Check actual body size (for chunked encoding)
			bodySize := int64(len(c.Body()))
			if bodySize > config.MaxSize {
				return c.Status(413).JSON(Map{
					"error":      config.ErrorMessage,
					"maxSize":    config.MaxSize,
					"actualSize": bodySize,
				})
			}

			return next(c)
		}
	}
}

// BodyLimitBytes creates a body limit middleware with size in bytes.
// This is an alias for BodyLimit for clarity in code.
//
// Parameters:
//   - bytes: Maximum body size in bytes
//
// Example:
//
//	app.Use(BodyLimitBytes(4194304)) // 4MB in bytes
func BodyLimitBytes(bytes int64) MiddlewareFunc {
	return BodyLimit(bytes)
}

// BodyLimitKB creates a body limit middleware with size in kilobytes.
// Convenience method for specifying limits in KB rather than bytes.
//
// Parameters:
//   - kb: Maximum body size in kilobytes
//
// Example:
//
//	app.Use(BodyLimitKB(4096)) // 4MB (4096 KB)
func BodyLimitKB(kb int) MiddlewareFunc {
	return BodyLimit(int64(kb) * 1024)
}

// BodyLimitMB creates a body limit middleware with size in megabytes.
// Most commonly used method for specifying body limits.
//
// Parameters:
//   - mb: Maximum body size in megabytes
//
// Example:
//
//	app.Use(BodyLimitMB(10)) // 10MB limit
//
//	// Different limits for different route groups
//	api := app.Group("/api")
//	api.Use(BodyLimitMB(5)) // 5MB for API
//
//	upload := app.Group("/upload")
//	upload.Use(BodyLimitMB(50)) // 50MB for uploads
func BodyLimitMB(mb int) MiddlewareFunc {
	return BodyLimit(int64(mb) * 1024 * 1024)
}

// BodyLimitGB creates a body limit middleware with size in gigabytes.
// Use with caution - very large limits can cause memory issues.
//
// Parameters:
//   - gb: Maximum body size in gigabytes
//
// Example:
//
//	// For large file upload endpoint
//	app.POST("/api/large-upload", handler, BodyLimitGB(1)) // 1GB limit
//
// Warning:
//   - Large limits increase memory consumption
//   - Can lead to OOM if multiple concurrent requests
//   - Consider streaming for very large files
func BodyLimitGB(gb int) MiddlewareFunc {
	return BodyLimit(int64(gb) * 1024 * 1024 * 1024)
}

// BodyLimitForRoute creates a body limit middleware for specific routes.
// This allows different body size limits for different URL paths without
// affecting global configuration.
//
// Parameters:
//   - maxSize: Maximum body size in bytes
//   - paths: List of URL paths to apply this limit to
//
// Returns:
//   - MiddlewareFunc: Middleware that only applies to specified paths
//
// Example:
//
//	// Global 5MB limit, but 50MB for specific upload endpoints
//	app.Use(BodyLimitMB(5))
//	app.Use(BodyLimitForRoute(
//	    50 * 1024 * 1024,
//	    "/api/upload",
//	    "/api/import",
//	))
//
//	// Apply to route group
//	admin := app.Group("/admin")
//	admin.Use(BodyLimitForRoute(1024, "/admin/webhook"))
func BodyLimitForRoute(maxSize int64, paths ...string) MiddlewareFunc {
	config := DefaultBodyLimitConfig()
	config.MaxSize = maxSize

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			path := c.Path()

			// Check if current path matches any of the specified paths
			shouldApply := false
			for _, p := range paths {
				if path == p {
					shouldApply = true
					break
				}
			}

			if !shouldApply {
				return next(c)
			}

			// Check body size
			contentLength := c.RequestCtx.Request.Header.ContentLength()
			if contentLength > int(config.MaxSize) {
				return c.Status(413).JSON(Map{
					"error":      config.ErrorMessage,
					"maxSize":    config.MaxSize,
					"actualSize": contentLength,
				})
			}

			bodySize := int64(len(c.Body()))
			if bodySize > config.MaxSize {
				return c.Status(413).JSON(Map{
					"error":      config.ErrorMessage,
					"maxSize":    config.MaxSize,
					"actualSize": bodySize,
				})
			}

			return next(c)
		}
	}
}

// BodyLimitByContentType creates body limit middleware based on content type.
// This allows setting different size limits for different content types,
// useful when some content types legitimately need more space.
//
// Parameters:
//   - limits: Map of content types to their maximum sizes in bytes
//
// Returns:
//   - MiddlewareFunc: Middleware that applies content-type specific limits
//
// Example:
//
//	app.Use(BodyLimitByContentType(map[string]int64{
//	    "application/json":           1024 * 1024,      // 1MB for JSON
//	    "multipart/form-data":        50 * 1024 * 1024, // 50MB for file uploads
//	    "application/octet-stream":   100 * 1024 * 1024, // 100MB for binary
//	    "text/plain":                 512 * 1024,       // 512KB for text
//	}))
//
// Use Cases:
//   - JSON APIs with small payloads
//   - File upload endpoints with large files
//   - Mixed content applications
//   - API versioning with different limits
func BodyLimitByContentType(limits map[string]int64) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			contentType := c.GetContentType()

			// Find matching limit for content type
			var maxSize int64
			found := false

			for ct, limit := range limits {
				if contentType == ct || (contentType != "" && len(contentType) >= len(ct) && contentType[:len(ct)] == ct) {
					maxSize = limit
					found = true
					break
				}
			}

			// If no specific limit found, proceed without checking
			if !found {
				return next(c)
			}

			// Check body size
			contentLength := c.RequestCtx.Request.Header.ContentLength()
			if contentLength > int(maxSize) {
				return c.Status(413).JSON(Map{
					"error":       "Request body too large for content type",
					"contentType": contentType,
					"maxSize":     maxSize,
					"actualSize":  contentLength,
				})
			}

			bodySize := int64(len(c.Body()))
			if bodySize > maxSize {
				return c.Status(413).JSON(Map{
					"error":       "Request body too large for content type",
					"contentType": contentType,
					"maxSize":     maxSize,
					"actualSize":  bodySize,
				})
			}

			return next(c)
		}
	}
}

// ValidateBodySize validates body size without middleware.
// This is a utility method for manual body size validation in handlers.
//
// Parameters:
//   - maxSize: Maximum allowed body size in bytes
//
// Returns:
//   - error: Error if body exceeds limit, nil otherwise
//
// Example:
//
//	func uploadHandler(c *blaze.Context) error {
//	    // Manual validation with custom limit
//	    if err := c.ValidateBodySize(10 * 1024 * 1024); err != nil {
//	        return c.Status(413).JSON(blaze.Map{"error": err.Error()})
//	    }
//
//	    // Process upload
//	    return c.JSON(blaze.Map{"status": "uploaded"})
//	}
func (c *Context) ValidateBodySize(maxSize int64) error {
	contentLength := c.RequestCtx.Request.Header.ContentLength()
	if contentLength > int(maxSize) {
		return fmt.Errorf("content length %d exceeds maximum %d", contentLength, maxSize)
	}

	bodySize := int64(len(c.Body()))
	if bodySize > maxSize {
		return fmt.Errorf("body size %d exceeds maximum %d", bodySize, maxSize)
	}

	return nil
}

// GetBodySize returns the size of the request body.
// This is a utility method for handlers that need to check body size.
//
// Returns:
//   - int64: Size of the request body in bytes
//
// Example:
//
//	func handler(c *blaze.Context) error {
//	    size := c.GetBodySize()
//	    c.Locals("bodySize", size)
//
//	    // Log large requests
//	    if size > 1024*1024 {
//	        log.Printf("Large request: %d bytes", size)
//	    }
//
//	    return c.JSON(blaze.Map{"bodySize": size})
//	}
func (c *Context) GetBodySize() int64 {
	return int64(len(c.Body()))
}

// GetContentLength returns the Content-Length header value.
// This method provides the declared content length from HTTP headers.
//
// Returns:
//   - int: Content length in bytes, or 0 if not set
//
// Example:
//
//	func handler(c *blaze.Context) error {
//	    contentLength := c.GetContentLength()
//	    actualSize := c.GetBodySize()
//
//	    // Verify Content-Length matches actual body
//	    if contentLength != int(actualSize) {
//	        log.Printf("Content-Length mismatch: declared=%d, actual=%d",
//	            contentLength, actualSize)
//	    }
//
//	    return c.JSON(blaze.Map{
//	        "contentLength": contentLength,
//	        "actualSize":    actualSize,
//	    })
//	}
func (c *Context) GetContentLength() int {
	return c.RequestCtx.Request.Header.ContentLength()
}
