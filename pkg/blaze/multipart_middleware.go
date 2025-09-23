package blaze

import (
	"log"
)

// MultipartMiddleware creates middleware for handling multipart forms
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
func ImageOnlyMiddleware() MiddlewareFunc {
	return FileTypeMiddleware(
		[]string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"},
		[]string{"image/jpeg", "image/png", "image/gif", "image/bmp", "image/webp"},
	)
}

// DocumentOnlyMiddleware restricts uploads to documents only
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
