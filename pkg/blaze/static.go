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

// StaticConfig holds configuration for serving static files
type StaticConfig struct {
	// Root directory to serve files from
	Root string

	// Index file to serve for directories (default: "index.html")
	Index string

	// Enable directory browsing (default: false for security)
	Browse bool

	// Enable compression (gzip) for responses (default: true)
	Compress bool

	// Enable byte range requests for large files (default: true)
	ByteRange bool

	// Cache control max-age in seconds (default: 3600 = 1 hour)
	CacheDuration time.Duration

	// Custom 404 handler when file not found
	NotFoundHandler HandlerFunc

	// Modify response function called before sending file
	Modify func(*Context) error

	// Generate ETag for caching (default: true)
	GenerateETag bool

	// File patterns to exclude (e.g., []string{".git", ".env"})
	Exclude []string

	// Custom MIME type mappings
	MIMETypes map[string]string
}

// DefaultStaticConfig returns default static file configuration
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
func Static(root string) HandlerFunc {
	return StaticFS(DefaultStaticConfig(root))
}

// serveFile serves a single file with proper headers
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

// serveFileRange handles HTTP range requests
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

func isExcluded(path string, excludePatterns []string) bool {
	base := filepath.Base(path)
	for _, pattern := range excludePatterns {
		if strings.Contains(base, pattern) {
			return true
		}
	}
	return false
}

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

func generateETagS(fileInfo os.FileInfo) string {
	return fmt.Sprintf(`"%x-%x"`, fileInfo.ModTime().Unix(), fileInfo.Size())
}

func checkETag(c *Context, etag string) bool {
	clientETag := c.Header("If-None-Match")
	return clientETag == etag
}

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
