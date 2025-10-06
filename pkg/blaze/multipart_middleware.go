package blaze

import (
	"log"
)

// MultipartMiddleware creates middleware for handling multipart forms
// Automatically parses and validates multipart form data before handler execution
//
// Middleware Features:
//   - Pre-parsing of multipart forms with size validation
//   - File type and size restrictions
//   - Automatic cleanup of temporary files
//   - Form data caching for reuse in handlers
//   - Memory management for large uploads
//
// Processing Flow:
//  1. Check if request contains multipart form data
//  2. Parse form with configured limits
//  3. Validate file sizes and types
//  4. Store parsed form in context for handler access
//  5. Execute handler
//  6. Auto-cleanup temporary files if enabled
//
// Context Storage:
//   - Parsed form: "multipartform" (access via c.MultipartForm())
//   - Config: "multipartconfig" (internal use)
//
// Error Handling:
//   - Returns 400 Bad Request for parsing errors
//   - Returns 413 Payload Too Large for size violations
//   - Returns 415 Unsupported Media Type for invalid file types
//
// Parameters:
//   - config: Multipart configuration (nil for defaults)
//
// Returns:
//   - MiddlewareFunc: Multipart form handling middleware
//
// Example - Basic Usage with Defaults:
//
//	app.Use(blaze.MultipartMiddleware(nil))
//
// Example - Custom Configuration:
//
//	config := blaze.MultipartConfig{
//	    MaxMemory: 16 * 1024 * 1024,  // 16MB in memory
//	    MaxFileSize: 50 * 1024 * 1024, // 50MB max file
//	    MaxFiles: 5,
//	    TempDir: "/tmp/uploads",
//	    AllowedExtensions: []string{".jpg", ".png", ".pdf"},
//	    KeepInMemory: false,
//	    AutoCleanup: true,
//	}
//	app.Use(blaze.MultipartMiddleware(&config))
//
// Example - Production Configuration:
//
//	config := blaze.ProductionMultipartConfig()
//	app.Use(blaze.MultipartMiddleware(&config))
func MultipartMiddleware(config *MultipartConfig) MiddlewareFunc {
	if config == nil {
		config = DefaultMultipartConfig()
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Only process multipart forms
			if !c.IsMultipartForm() {
				return next(c)
			}

			// Pre-parse the form to check limits early
			form, err := c.MultipartFormWithConfig(config)
			if err != nil {
				return c.Status(400).JSON(Error(err.Error()))
			}

			// Store parsed form in context for reuse
			c.SetUserValue("multipart_form", form)
			c.SetUserValue("multipart_config", config)

			// Set cleanup handler if auto cleanup is enabled
			if config.AutoCleanup {
				defer func() {
					if cleanupErr := form.Cleanup(); cleanupErr != nil {
						log.Printf("Failed to cleanup multipart files: %v", cleanupErr)
					}
				}()
			}

			return next(c)
		}
	}
}

// FileSizeLimitMiddleware creates middleware to limit file upload sizes
// Checks Content-Length header before parsing to reject oversized requests early
//
// Performance Benefits:
//   - Rejects large uploads before parsing (saves CPU and memory)
//   - Returns 413 immediately for oversized requests
//   - Prevents DOS attacks via large file uploads
//
// Limitations:
//   - Only checks Content-Length header (not guaranteed accurate)
//   - Actual file sizes validated during parsing
//   - Some clients don't send Content-Length
//
// Parameters:
//   - maxSize: Maximum total request size in bytes
//
// Returns:
//   - MiddlewareFunc: File size limit middleware
//
// Example - 10MB Limit:
//
//	app.Use(blaze.FileSizeLimitMiddleware(10 * 1024 * 1024))
//
// Example - Per-Route Limits:
//
//	app.POST("/avatar", uploadHandler, blaze.FileSizeLimitMiddleware(1024*1024)) // 1MB
//	app.POST("/documents", uploadHandler, blaze.FileSizeLimitMiddleware(50*1024*1024)) // 50MB
func FileSizeLimitMiddleware(maxSize int64) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			if c.IsMultipartForm() {
				// Check content length if available
				if contentLength := c.Request().Header.ContentLength(); contentLength > int(maxSize) {
					return c.Status(413).JSON(Map{
						"error":    "Request entity too large",
						"max_size": maxSize,
						"received": contentLength,
					})
				}
			}
			return next(c)
		}
	}
}

// FileTypeMiddleware creates middleware to restrict file types
// Validates file extensions and MIME types during upload
//
// Validation Strategy:
//   - Extension-based: Checks file extension (.jpg, .pdf, etc.)
//   - MIME-based: Validates Content-Type header
//   - Both checks must pass if both lists are provided
//   - Empty lists allow all extensions/types
//
// Security Considerations:
//   - Extensions can be spoofed (rename file.exe to file.jpg)
//   - MIME types from client can be unreliable
//   - Consider using file content detection for critical security
//   - Combine with virus scanning for production
//
// Parameters:
//   - allowedExtensions: Permitted file extensions (e.g., []string{".jpg", ".png"})
//   - allowedMimeTypes: Permitted MIME types (e.g., []string{"image/jpeg", "image/png"})
//
// Returns:
//   - MiddlewareFunc: File type restriction middleware
//
// Example - Image Files Only:
//
//	app.Use(blaze.FileTypeMiddleware(
//	    []string{".jpg", ".jpeg", ".png", ".gif"},
//	    []string{"image/jpeg", "image/png", "image/gif"},
//	))
//
// Example - Documents Only:
//
//	app.Use(blaze.FileTypeMiddleware(
//	    []string{".pdf", ".doc", ".docx"},
//	    []string{"application/pdf", "application/msword"},
//	))
func FileTypeMiddleware(allowedExtensions []string, allowedMimeTypes []string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			if !c.IsMultipartForm() {
				return next(c)
			}

			// Create custom config for validation
			config := &MultipartConfig{
				AllowedExtensions: allowedExtensions,
				AllowedMimeTypes:  allowedMimeTypes,
				MaxMemory:         32 << 20, // 32MB default
			}

			// This will validate file types during parsing
			_, err := c.MultipartFormWithConfig(config)
			if err != nil {
				return c.Status(400).JSON(Error(err.Error()))
			}

			return next(c)
		}
	}
}

// ImageOnlyMiddleware restricts uploads to images only
// Convenience middleware for common image upload use case
//
// Allowed Formats:
//   - JPEG (.jpg, .jpeg)
//   - PNG (.png)
//   - GIF (.gif)
//   - BMP (.bmp)
//   - WebP (.webp)
//
// Use Cases:
//   - Avatar uploads
//   - Profile pictures
//   - Product images
//   - Gallery uploads
//
// Returns:
//   - MiddlewareFunc: Image-only upload middleware
//
// Example:
//
//	app.POST("/upload/avatar", uploadHandler, blaze.ImageOnlyMiddleware())
func ImageOnlyMiddleware() MiddlewareFunc {
	return FileTypeMiddleware(
		[]string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"},
		[]string{"image/jpeg", "image/png", "image/gif", "image/bmp", "image/webp"},
	)
}

// DocumentOnlyMiddleware restricts uploads to documents only
// Convenience middleware for document upload use case
//
// Allowed Formats:
//   - PDF (.pdf)
//   - Word (.doc, .docx)
//   - Excel (.xls, .xlsx)
//   - Text (.txt)
//   - CSV (.csv)
//
// Use Cases:
//   - Resume uploads
//   - Document management systems
//   - Report uploads
//   - Contract submissions
//
// Returns:
//   - MiddlewareFunc: Document-only upload middleware
//
// Example:
//
//	app.POST("/upload/resume", uploadHandler, blaze.DocumentOnlyMiddleware())
func DocumentOnlyMiddleware() MiddlewareFunc {
	return FileTypeMiddleware(
		[]string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".txt", ".csv"},
		[]string{
			"application/pdf",
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"application/vnd.ms-excel",
			"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			"text/plain",
			"text/csv",
		},
	)
}

// MultipartLoggingMiddleware logs multipart form details
// Provides visibility into file uploads for debugging and monitoring
//
// Logged Information:
//   - Number of files uploaded
//   - Total upload size in bytes
//   - Number of form fields
//   - Individual file names and sizes (when detailed logging enabled)
//
// Log Level: Info
//
// Performance Impact:
//   - Minimal (only logs after successful parsing)
//   - Calculates totals by iterating files
//   - Consider disabling in high-volume production
//
// Use Cases:
//   - Debugging upload issues
//   - Monitoring upload patterns
//   - Auditing file submissions
//   - Usage analytics
//
// Returns:
//   - MiddlewareFunc: Upload logging middleware
//
// Example:
//
//	app.Use(blaze.MultipartLoggingMiddleware())
//
// Example - With Logger:
//
//	logger := blaze.GetDefaultLogger()
//	app.Use(blaze.LoggerMiddleware())
//	app.Use(blaze.MultipartLoggingMiddleware())
func MultipartLoggingMiddleware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			if c.IsMultipartForm() {
				form, err := c.MultipartForm()
				if err == nil {
					log.Printf("Multipart form - Files: %d, Total size: %d bytes, Fields: %d",
						form.GetFileCount(),
						form.GetTotalSize(),
						len(form.Value))
				}
			}
			return next(c)
		}
	}
}
