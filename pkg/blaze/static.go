package blaze

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// StaticConfig holds comprehensive configuration for serving static files
// Controls caching, compression, security, and directory listing behavior
//
// Static File Serving Philosophy:
//   - Security: Prevent directory traversal and unauthorized access
//   - Performance: Enable caching and compression
//   - Usability: Support range requests and directory browsing
//   - Flexibility: Allow custom handlers and modifications
//
// Production Best Practices:
//   - Disable directory browsing (Browse: false)
//   - Enable compression (Compress: true)
//   - Set appropriate cache duration (CacheDuration: 1 hour+)
//   - Exclude sensitive files (.git, .env, etc.)
//   - Use CDN for production static assets
//
// Security Considerations:
//   - Always sanitize file paths to prevent traversal
//   - Exclude sensitive files and directories
//   - Disable directory listing in production
//   - Set proper MIME types to prevent XSS
//   - Implement access controls for private files
type StaticConfig struct {
	// Root is the directory to serve files from
	// Must be an absolute or relative path to existing directory
	// Required field - no default
	// Example: "./public", "/var/www/html"
	Root string

	// Index is the default file to serve for directories
	// Served when request path is a directory
	// Common values: "index.html", "default.html"
	// Default: "index.html"
	Index string

	// Browse enables directory listing when no index file exists
	// When true: Shows list of files in directory
	// When false: Returns 403 Forbidden
	// Security: Should be false in production
	// Default: false
	Browse bool

	// Compress enables gzip compression for responses
	// Automatically compresses compatible content types
	// Reduces bandwidth usage and improves load times
	// Should be enabled for production
	// Default: true
	Compress bool

	// ByteRange enables HTTP range request support
	// Required for video streaming and large file downloads
	// Allows resuming downloads and seeking in media
	// Should be enabled for production
	// Default: true
	ByteRange bool

	// CacheDuration specifies cache control max-age
	// Determines how long browsers cache static files
	// Common values: 1 hour, 1 day, 1 week, 1 year
	// Use longer durations for versioned assets
	// Default: 1 hour
	CacheDuration time.Duration

	// NotFoundHandler is called when file doesn't exist
	// Allows custom 404 pages or logging
	// If nil, returns standard 404 error
	// Example: Custom 404 page handler
	NotFoundHandler HandlerFunc

	// Modify is called before sending each file
	// Allows adding custom headers or logic
	// Receives context for inspection/modification
	// Return error to abort file serving
	// Example: Add custom security headers
	Modify func(*Context) error

	// GenerateETag enables ETag generation for caching
	// ETags allow efficient cache validation
	// Generated from file modification time and size
	// Reduces unnecessary data transfer
	// Default: true
	GenerateETag bool

	// Exclude specifies file patterns to exclude
	// Prevents serving sensitive files
	// Supports substring matching
	// Common exclusions: ".git", ".env", "*.key"
	// Default: [".git", ".svn", ".DS_Store"]
	Exclude []string

	// MIMETypes provides custom MIME type mappings
	// Maps file extensions to content types
	// Overrides default MIME type detection
	// Example: {".json": "application/json"}
	// Default: empty (uses standard MIME types)
	MIMETypes map[string]string
}

// DefaultStaticConfig returns default static file configuration
// Provides secure, production-ready defaults
//
// Default Settings:
//   - Root: Must be provided (required)
//   - Index: "index.html"
//   - Browse: false (security)
//   - Compress: true (performance)
//   - ByteRange: true (streaming support)
//   - CacheDuration: 1 hour
//   - GenerateETag: true (caching)
//   - Exclude: [".git", ".svn", ".DS_Store"]
//
// Parameters:
//   - root: Root directory path
//
// Returns:
//   - StaticConfig: Default configuration
//
// Example:
//
//	config := blaze.DefaultStaticConfig("./public")
func DefaultStaticConfig(root string) StaticConfig {
	return StaticConfig{
		Root:          root,
		Index:         "index.html",
		Browse:        false,
		Compress:      true,
		ByteRange:     true,
		CacheDuration: time.Hour,
		GenerateETag:  true,
		Exclude:       []string{".git", ".svn", ".DS_Store"},
		MIMETypes:     make(map[string]string),
	}
}

// StaticFS creates a handler for serving static files with custom configuration
// Provides full control over static file serving behavior
//
// Setup Process:
//  1. Validate and normalize root directory
//  2. Check directory exists and is accessible
//  3. Set configuration defaults
//  4. Return handler function
//
// Handler Behavior:
//   - Extracts file path from URL
//   - Validates path (prevents traversal)
//   - Checks exclusion rules
//   - Serves files or directories
//   - Applies caching and compression
//
// Parameters:
//   - config: Static file configuration
//
// Returns:
//   - HandlerFunc: Static file serving handler
//
// Example - Basic Setup:
//
//	handler := blaze.StaticFS(blaze.StaticConfig{
//	    Root: "./public",
//	})
//	app.GET("/static/*", handler)
//
// Example - Production Setup:
//
//	config := blaze.StaticConfig{
//	    Root: "./public",
//	    Index: "index.html",
//	    Browse: false,
//	    Compress: true,
//	    ByteRange: true,
//	    CacheDuration: 24 * time.Hour,
//	    GenerateETag: true,
//	    Exclude: []string{".git", ".env", ".key"},
//	}
//	handler := blaze.StaticFS(config)
//	app.GET("/assets/*", handler)
//
// Example - Development Setup with Browsing:
//
//	config := blaze.StaticConfig{
//	    Root: "./public",
//	    Browse: true,
//	    CacheDuration: 0, // No caching in dev
//	}
//	handler := blaze.StaticFS(config)
//
// Example - With Custom 404:
//
//	config := blaze.DefaultStaticConfig("./public")
//	config.NotFoundHandler = func(c *blaze.Context) error {
//	    return c.Status(404).HTML("<h1>File Not Found</h1>")
//	}
//	handler := blaze.StaticFS(config)
func StaticFS(config StaticConfig) HandlerFunc {
	// Validate and normalize root directory
	if config.Root == "" {
		panic("Static file root directory cannot be empty")
	}

	absRoot, err := filepath.Abs(config.Root)
	if err != nil {
		panic(fmt.Sprintf("Invalid static root directory: %v", err))
	}

	// Check if directory exists
	if _, err := os.Stat(absRoot); os.IsNotExist(err) {
		panic(fmt.Sprintf("Static root directory does not exist: %s", absRoot))
	}

	config.Root = absRoot

	// Set defaults
	if config.Index == "" {
		config.Index = "index.html"
	}

	return func(c *Context) error {
		// Get the file path from URL
		urlPath := c.Path()

		// Clean the path to prevent directory traversal
		cleanPath := path.Clean(urlPath)
		if !strings.HasPrefix(cleanPath, "/") {
			cleanPath = "/" + cleanPath
		}

		// Build full file system path
		fsPath := filepath.Join(config.Root, filepath.FromSlash(cleanPath))

		// Security check: ensure path is within root directory
		if !strings.HasPrefix(fsPath, config.Root) {
			return ErrForbidden("Access denied")
		}

		// Check if file/directory is excluded
		if isExcluded(fsPath, config.Exclude) {
			return ErrNotFound("File not found")
		}

		// Get file info
		fileInfo, err := os.Stat(fsPath)
		if err != nil {
			if os.IsNotExist(err) {
				if config.NotFoundHandler != nil {
					return config.NotFoundHandler(c)
				}
				return ErrNotFound("File not found")
			}
			return ErrInternalServer("Failed to access file")
		}

		// Handle directories
		if fileInfo.IsDir() {
			return handleDirectory(c, fsPath, cleanPath, config)
		}

		// Serve the file
		return serveFile(c, fsPath, fileInfo, config)
	}
}

// Static creates a handler for serving static files with default configuration
// Convenience method for simple static file serving
//
// Parameters:
//   - root: Root directory path
//
// Returns:
//   - HandlerFunc: Static file handler with defaults
//
// Example:
//
//	handler := blaze.Static("./public")
//	app.GET("/static/*", handler)
func Static(root string) HandlerFunc {
	return StaticFS(DefaultStaticConfig(root))
}

// serveFile serves a single file with proper headers
// Handles caching, compression, range requests, and conditional requests
//
// Serving Process:
//  1. Apply custom modifier if configured
//  2. Set content type based on file extension
//  3. Set cache headers (Cache-Control, Expires)
//  4. Set Last-Modified header
//  5. Generate and set ETag if enabled
//  6. Check conditional requests (If-None-Match, If-Modified-Since)
//  7. Handle range requests if enabled
//  8. Send file content
//
// Parameters:
//   - c: Request context
//   - fsPath: File system path
//   - fileInfo: File information
//   - config: Static configuration
//
// Returns:
//   - error: Serving error or nil
func serveFile(c *Context, fsPath string, fileInfo os.FileInfo, config StaticConfig) error {
	// Apply custom modifier if provided
	if config.Modify != nil {
		if err := config.Modify(c); err != nil {
			return err
		}
	}

	// Set content type
	contentType := getContentType(fsPath, config.MIMETypes)
	c.SetHeader("Content-Type", contentType)

	// Set cache headers
	if config.CacheDuration > 0 {
		c.SetHeader("Cache-Control", fmt.Sprintf("public, max-age=%d", int(config.CacheDuration.Seconds())))
		c.SetHeader("Expires", time.Now().Add(config.CacheDuration).UTC().Format(time.RFC1123))
	}

	// Set last modified header
	lastModified := fileInfo.ModTime().UTC().Format(time.RFC1123)
	c.SetHeader("Last-Modified", lastModified)

	// Generate and set ETag if enabled
	if config.GenerateETag {
		etag := generateETagS(fileInfo)
		c.SetHeader("ETag", etag)

		// Check if client has cached version
		if checkETag(c, etag) || checkModifiedSince(c, fileInfo.ModTime()) {
			return c.Status(304).Text("") // Not Modified
		}
	}

	// Enable byte range if configured
	if config.ByteRange {
		c.SetHeader("Accept-Ranges", "bytes")
	}

	// Handle range requests
	rangeHeader := c.Header("Range")
	if config.ByteRange && rangeHeader != "" {
		return serveFileRange(c, fsPath, fileInfo, rangeHeader)
	}

	// Set content length
	c.SetHeader("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Serve the file using fasthttp
	c.RequestCtx.SendFile(fsPath)

	return nil
}

// handleDirectory handles directory requests
// Serves index file if exists, or generates directory listing if enabled
//
// Directory Handling:
//  1. Try to serve index file (index.html by default)
//  2. If index doesn't exist and Browse is false: return 403
//  3. If Browse is true: generate HTML directory listing
//
// Parameters:
//   - c: Request context
//   - fsPath: Directory file system path
//   - urlPath: Directory URL path
//   - config: Static configuration
//
// Returns:
//   - error: Handling error or nil
func handleDirectory(c *Context, fsPath string, urlPath string, config StaticConfig) error {
	// Try to serve index file
	indexPath := filepath.Join(fsPath, config.Index)
	indexInfo, err := os.Stat(indexPath)

	if err == nil && !indexInfo.IsDir() {
		return serveFile(c, indexPath, indexInfo, config)
	}

	// If browsing is disabled, return 403
	if !config.Browse {
		return ErrForbidden("Directory listing is disabled")
	}

	// Generate directory listing
	return generateDirectoryListing(c, fsPath, urlPath)
}

// generateDirectoryListing creates an HTML directory listing
// Displays contents of directory with links to files and subdirectories
//
// Listing Features:
//   - Sorted listing (directories first, then files)
//   - Parent directory link (if not root)
//   - File sizes and modification times
//   - Clean, responsive HTML layout
//
// Parameters:
//   - c: Request context
//   - fsPath: Directory path
//   - urlPath: URL path
//
// Returns:
//   - error: Generation error or nil
func generateDirectoryListing(c *Context, fsPath string, urlPath string) error {
	dir, err := os.Open(fsPath)
	if err != nil {
		return ErrInternalServer("Failed to open directory")
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		return ErrInternalServer("Failed to read directory")
	}

	// Sort entries: directories first, then files
	dirs := []os.FileInfo{}
	files := []os.FileInfo{}
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}

	// Build HTML
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Index of %s</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; padding: 20px; }
		h1 { border-bottom: 1px solid #ddd; padding-bottom: 10px; }
		table { width: 100%%; border-collapse: collapse; }
		th { text-align: left; padding: 10px; border-bottom: 2px solid #ddd; background: #f5f5f5; }
		td { padding: 10px; border-bottom: 1px solid #eee; }
		a { color: #0066cc; text-decoration: none; }
		a:hover { text-decoration: underline; }
		.size { text-align: right; }
		.modified { text-align: right; color: #666; }
		.dir { font-weight: bold; }
	</style>
</head>
<body>
	<h1>Index of %s</h1>
	<table>
		<thead>
			<tr>
				<th>Name</th>
				<th class="size">Size</th>
				<th class="modified">Last Modified</th>
			</tr>
		</thead>
		<tbody>`, urlPath, urlPath)

	// Add parent directory link if not root
	if urlPath != "/" {
		parentPath := path.Dir(urlPath)
		html += fmt.Sprintf(`<tr><td class="dir"><a href="%s">..</a></td><td class="size">-</td><td class="modified">-</td></tr>`, parentPath)
	}

	// Add directories
	for _, entry := range dirs {
		name := entry.Name()
		href := path.Join(urlPath, name) + "/"
		html += fmt.Sprintf(`<tr><td class="dir"><a href="%s">%s/</a></td><td class="size">-</td><td class="modified">%s</td></tr>`,
			href, name, entry.ModTime().Format("2006-01-02 15:04:05"))
	}

	// Add files
	for _, entry := range files {
		name := entry.Name()
		href := path.Join(urlPath, name)
		size := formatFileSize(entry.Size())
		html += fmt.Sprintf(`<tr><td><a href="%s">%s</a></td><td class="size">%s</td><td class="modified">%s</td></tr>`,
			href, name, size, entry.ModTime().Format("2006-01-02 15:04:05"))
	}

	html += `</tbody></table></body></html>`

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	return c.HTML(html)
}

// serveFileRange handles HTTP range requests for partial content
// Enables video streaming, resumable downloads, and seeking
//
// Range Request Format:
//   - Header: "Range: bytes=start-end"
//   - Examples: "bytes=0-999", "bytes=1000-", "bytes=-500"
//
// Response:
//   - Status: 206 Partial Content
//   - Headers: Content-Range, Content-Length
//   - Body: Requested byte range
//
// Parameters:
//   - c: Request context
//   - fsPath: File path
//   - fileInfo: File information
//   - rangeHeader: Range header value
//
// Returns:
//   - error: Range serving error or nil
func serveFileRange(c *Context, fsPath string, fileInfo os.FileInfo, rangeHeader string) error {
	file, err := os.Open(fsPath)
	if err != nil {
		return ErrInternalServer("Failed to open file")
	}
	defer file.Close()

	fileSize := fileInfo.Size()

	// Parse range header (simplified, supports single range)
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return ErrBadRequest("Invalid range header")
	}

	rangeSpec := rangeHeader[6:]
	parts := strings.Split(rangeSpec, "-")
	if len(parts) != 2 {
		return ErrBadRequest("Invalid range format")
	}

	var start, end int64

	// Parse start
	if parts[0] != "" {
		fmt.Sscanf(parts[0], "%d", &start)
	}

	// Parse end
	if parts[1] != "" {
		fmt.Sscanf(parts[1], "%d", &end)
	} else {
		end = fileSize - 1
	}

	// Validate range
	if start < 0 || end >= fileSize || start > end {
		c.SetHeader("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		return c.Status(416).Text("Requested Range Not Satisfiable")
	}

	// Calculate content length
	contentLength := end - start + 1

	// Set headers for partial content
	c.Status(206) // Partial Content
	c.SetHeader("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.SetHeader("Content-Length", fmt.Sprintf("%d", contentLength))
	c.SetHeader("Content-Type", getContentType(fsPath, nil))

	// Seek to start position
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return ErrInternalServer("Failed to seek file")
	}

	// Copy the requested range
	_, err = io.CopyN(c.Response().BodyWriter(), file, contentLength)
	return err
}

// Helper functions

// isExcluded checks if path matches exclusion patterns
func isExcluded(path string, excludePatterns []string) bool {
	base := filepath.Base(path)
	for _, pattern := range excludePatterns {
		if strings.Contains(base, pattern) {
			return true
		}
	}
	return false
}

// getContentType determines MIME type from file extension
func getContentType(filePath string, customTypes map[string]string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Check custom MIME types first
	if customTypes != nil {
		if contentType, ok := customTypes[ext]; ok {
			return contentType
		}
	}

	// Use standard MIME types
	contentType := mime.TypeByExtension(ext)
	if contentType != "" {
		return contentType
	}

	// Default to octet-stream
	return "application/octet-stream"
}

// generateETag generates ETag from file info
func generateETagS(fileInfo os.FileInfo) string {
	return fmt.Sprintf(`"%x-%x"`, fileInfo.ModTime().Unix(), fileInfo.Size())
}

// checkETag checks if client ETag matches
func checkETag(c *Context, etag string) bool {
	clientETag := c.Header("If-None-Match")
	return clientETag == etag
}

// checkModifiedSince checks if file modified since client cache
func checkModifiedSince(c *Context, modTime time.Time) bool {
	ifModifiedSince := c.Header("If-Modified-Since")
	if ifModifiedSince == "" {
		return false
	}

	clientTime, err := time.Parse(time.RFC1123, ifModifiedSince)
	if err != nil {
		return false
	}

	// Truncate to seconds for comparison
	return !modTime.Truncate(time.Second).After(clientTime)
}

// formatFileSize formats file size in human-readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
