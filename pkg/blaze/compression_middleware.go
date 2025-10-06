package blaze

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"sync"
)

// CompressionLevel defines compression levels
type CompressionLevel int

// Compression level constants
const (
	// CompressionLevelDefault uses the default compression level (-1)
	CompressionLevelDefault CompressionLevel = -1

	// CompressionLevelNone disables compression (0)
	CompressionLevelNone CompressionLevel = 0

	// CompressionLevelBest provides maximum compression (9)
	CompressionLevelBest CompressionLevel = 9

	// CompressionLevelFastest provides fastest compression (1)
	CompressionLevelFastest CompressionLevel = 1
)

// CompressionConfig holds compression middleware configuration
// Provides fine-grained control over compression behavior, content type filtering,
// and performance tuning for production deployments
type CompressionConfig struct {
	// Level specifies compression level (0-9, -1 for default)
	// Higher levels provide better compression but use more CPU
	// Recommended: 6 for balanced performance, 9 for maximum compression
	Level CompressionLevel

	// MinLength sets minimum response size to compress in bytes
	// Responses smaller than this threshold are not compressed
	// Default: 1024 bytes (1KB) - compressing smaller responses often increases size
	MinLength int

	// IncludeContentTypes specifies content types to compress
	// If empty, compresses all content types except those in ExcludeContentTypes
	// Common compressible types: text/html, text/css, application/json, etc.
	IncludeContentTypes []string

	// ExcludeContentTypes specifies content types to never compress
	// Takes precedence over IncludeContentTypes
	// Common excluded types: images, videos, audio, already-compressed formats
	ExcludeContentTypes []string

	// EnableGzip enables gzip compression (default: true)
	// Most widely supported compression format across all browsers
	EnableGzip bool

	// EnableDeflate enables deflate compression (default: true)
	// Slightly less common than gzip but supported by most browsers
	EnableDeflate bool

	// EnableBrotli enables brotli compression (requires external package)
	// Provides better compression than gzip but requires br support
	// Note: Currently not implemented, requires github.com/andybalholm/brotli
	EnableBrotli bool

	// ExcludePaths lists URL paths to exclude from compression
	// Exact match only - paths must match exactly
	// Example: []string{"/api/raw", "/download"}
	ExcludePaths []string

	// ExcludeExtensions lists file extensions to exclude from compression
	// Common excluded extensions: .jpg, .png, .mp4, .zip, .pdf
	ExcludeExtensions []string

	// EnableForHTTPS enables compression for HTTPS requests
	// Disabled by default for security (prevents CRIME/BREACH attacks)
	// Only enable if you're confident about security implications
	EnableForHTTPS bool
}

// DefaultCompressionConfig returns default compression configuration
// Provides production-ready defaults with sensible security and performance settings
//
// Default settings:
//   - Compression level: 6 (gzip.DefaultCompression)
//   - Minimum length: 1024 bytes (1KB)
//   - Enabled formats: gzip, deflate
//   - Includes: text/*, application/json, application/javascript, application/xml
//   - Excludes: images, videos, audio, archives, fonts
//   - HTTPS compression: disabled for security
//
// Returns:
//   - CompressionConfig with production-ready defaults
func DefaultCompressionConfig() CompressionConfig {
	return CompressionConfig{
		Level:     CompressionLevel(gzip.DefaultCompression),
		MinLength: 1024, // 1KB minimum
		IncludeContentTypes: []string{
			"text/html",
			"text/css",
			"text/plain",
			"text/xml",
			"text/javascript",
			"application/json",
			"application/javascript",
			"application/xml",
			"application/xml+rss",
			"application/rss+xml",
			"application/atom+xml",
			"application/x-javascript",
			"application/ld+json",
			"image/svg+xml",
		},
		ExcludeContentTypes: []string{
			"image/jpeg",
			"image/png",
			"image/gif",
			"image/webp",
			"image/bmp",
			"video/mp4",
			"video/mpeg",
			"video/webm",
			"audio/mpeg",
			"audio/ogg",
			"audio/wav",
			"application/zip",
			"application/gzip",
			"application/x-gzip",
			"application/x-compress",
			"application/x-compressed",
			"application/pdf",
		},
		EnableGzip:    true,
		EnableDeflate: true,
		EnableBrotli:  false,
		ExcludePaths:  []string{},
		ExcludeExtensions: []string{
			".jpg", ".jpeg", ".png", ".gif", ".webp",
			".mp4", ".mpeg", ".webm", ".mp3", ".ogg",
			".zip", ".gz", ".pdf", ".woff", ".woff2",
		},
		EnableForHTTPS: false,
	}
}

// compressionWriter wraps response writer with compression
// Used for streaming compression (currently not fully implemented)
type compressionWriter struct {
	ctx          *Context
	encoding     string
	writer       io.WriteCloser
	buffer       *bytes.Buffer
	headersSent  bool
	minLength    int
	originalBody []byte
	compressed   bool
}

// Pool for gzip writers - reduces GC pressure by reusing writer instances
var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

// Pool for flate writers - reduces GC pressure by reusing writer instances
var flateWriterPool = sync.Pool{
	New: func() interface{} {
		w, _ := flate.NewWriter(io.Discard, flate.DefaultCompression)
		return w
	},
}

// Pool for buffers - reduces allocations by reusing buffers
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// Compress creates compression middleware with default configuration
// Uses DefaultCompressionConfig() with gzip and deflate enabled
//
// Compression flow:
//  1. Checks if request accepts compression (Accept-Encoding header)
//  2. Executes handler to get response
//  3. Validates response (size, content type, already compressed)
//  4. Compresses response using best available encoding
//  5. Sets appropriate headers (Content-Encoding, Vary, etc.)
//
// Example:
//
//	app.Use(blaze.Compress())
//
// Returns:
//   - MiddlewareFunc that applies compression to responses
func Compress() MiddlewareFunc {
	return CompressWithConfig(DefaultCompressionConfig())
}

// CompressWithConfig creates compression middleware with custom configuration
// Allows full control over compression behavior through CompressionConfig
//
// Configuration options:
//   - Compression level (0-9)
//   - Minimum response size threshold
//   - Content type filtering (include/exclude lists)
//   - Path and extension exclusions
//   - HTTPS compression control
//
// The middleware:
//   - Only compresses when client supports it (Accept-Encoding header)
//   - Skips already compressed responses
//   - Only compresses if result is smaller than original
//   - Sets proper HTTP headers (Content-Encoding, Vary)
//
// Parameters:
//   - config: CompressionConfig with compression settings
//
// Returns:
//   - MiddlewareFunc that applies compression based on config
//
// Example:
//
//	config := blaze.DefaultCompressionConfig()
//	config.Level = blaze.CompressionLevelBest
//	config.MinLength = 2048
//	app.Use(blaze.CompressWithConfig(config))
func CompressWithConfig(config CompressionConfig) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Skip if path is excluded
			if isPathExcluded(c.Path(), config.ExcludePaths) {
				return next(c)
			}

			// Skip if extension is excluded
			if isExtensionExcluded(c.Path(), config.ExcludeExtensions) {
				return next(c)
			}

			// Skip compression for HTTPS if disabled
			if !config.EnableForHTTPS && string(c.URI().Scheme()) == "https" {
				return next(c)
			}

			// Get accepted encodings from client
			acceptEncoding := string(c.RequestCtx.Request.Header.Peek("Accept-Encoding"))
			if acceptEncoding == "" {
				return next(c)
			}

			// Determine best encoding
			encoding := selectEncoding(acceptEncoding, config)
			if encoding == "" {
				return next(c)
			}

			// Execute handler first
			err := next(c)
			if err != nil {
				return err
			}

			// Get response body
			body := c.Response().Body()
			if len(body) == 0 {
				return nil
			}

			// Check minimum length
			if len(body) < config.MinLength {
				return nil
			}

			// Check content type
			contentType := string(c.Response().Header.ContentType())
			if !shouldCompress(contentType, config) {
				return nil
			}

			// Check if already compressed
			if string(c.Response().Header.Peek("Content-Encoding")) != "" {
				return nil
			}

			// Compress the response
			compressed, err := compressBody(body, encoding, int(config.Level))
			if err != nil {
				return nil // Fail silently, return uncompressed
			}

			// Only use compressed version if it's actually smaller
			if len(compressed) >= len(body) {
				return nil
			}

			// Set compressed body
			c.Response().SetBody(compressed)

			// Set headers
			c.Response().Header.Set("Content-Encoding", encoding)
			c.Response().Header.Set("Content-Length", fmt.Sprintf("%d", len(compressed)))
			c.Response().Header.Del("Accept-Ranges") // Disable range requests for compressed content
			c.Response().Header.Set("Vary", "Accept-Encoding")

			return nil
		}
	}
}

// selectEncoding selects the best compression encoding based on client support
// Prioritizes encodings in order: brotli > gzip > deflate
//
// The Accept-Encoding header indicates which compression algorithms the client supports
// This function selects the best available option based on configuration
//
// Parameters:
//   - acceptEncoding: Accept-Encoding header value from client
//   - config: CompressionConfig specifying enabled encodings
//
// Returns:
//   - string: Selected encoding ("br", "gzip", "deflate") or empty string if none available
func selectEncoding(acceptEncoding string, config CompressionConfig) string {
	acceptEncoding = strings.ToLower(acceptEncoding)

	// Priority: brotli > gzip > deflate
	if config.EnableBrotli && strings.Contains(acceptEncoding, "br") {
		return "br"
	}

	if config.EnableGzip && strings.Contains(acceptEncoding, "gzip") {
		return "gzip"
	}

	if config.EnableDeflate && strings.Contains(acceptEncoding, "deflate") {
		return "deflate"
	}

	return ""
}

// compressBody compresses the body using the specified encoding
// Uses sync.Pool for efficient buffer and writer reuse
//
// Supported encodings:
//   - "gzip": gzip compression (RFC 1952)
//   - "deflate": deflate compression (RFC 1951)
//   - "br": brotli compression (not currently implemented)
//
// Parameters:
//   - body: Response body bytes to compress
//   - encoding: Compression algorithm ("gzip", "deflate", "br")
//   - level: Compression level (0-9)
//
// Returns:
//   - []byte: Compressed data
//   - error: Compression error or nil on success
func compressBody(body []byte, encoding string, level int) ([]byte, error) {
	buffer := bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer bufferPool.Put(buffer)

	var writer io.WriteCloser

	switch encoding {
	case "gzip":
		gzWriter := gzipWriterPool.Get().(*gzip.Writer)
		defer gzipWriterPool.Put(gzWriter)

		gzWriter.Reset(buffer)
		writer = gzWriter

	case "deflate":
		flWriter := flateWriterPool.Get().(*flate.Writer)
		defer flateWriterPool.Put(flWriter)

		flWriter.Reset(buffer)
		writer = flWriter

	case "br":
		// Brotli support would require external package
		return nil, fmt.Errorf("brotli not supported")

	default:
		return nil, fmt.Errorf("unsupported encoding: %s", encoding)
	}

	// Write and close
	if _, err := writer.Write(body); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	// Copy compressed data
	compressed := make([]byte, buffer.Len())
	copy(compressed, buffer.Bytes())

	return compressed, nil
}

// shouldCompress checks if content type should be compressed
// Evaluates content type against include and exclude lists
//
// Logic:
//  1. Check excluded types first (takes precedence)
//  2. If include list is empty, compress all except excluded
//  3. If include list exists, only compress types in the list
//
// Parameters:
//   - contentType: Response Content-Type header value
//   - config: CompressionConfig with include/exclude lists
//
// Returns:
//   - bool: true if content type should be compressed
func shouldCompress(contentType string, config CompressionConfig) bool {
	contentType = strings.ToLower(strings.Split(contentType, ";")[0])
	contentType = strings.TrimSpace(contentType)

	// Check excluded types first
	for _, excluded := range config.ExcludeContentTypes {
		if strings.Contains(contentType, strings.ToLower(excluded)) {
			return false
		}
	}

	// If include list is empty, compress all (except excluded)
	if len(config.IncludeContentTypes) == 0 {
		return true
	}

	// Check include list
	for _, included := range config.IncludeContentTypes {
		if strings.Contains(contentType, strings.ToLower(included)) {
			return true
		}
	}

	return false
}

// isPathExcluded checks if path should be excluded from compression
// Performs exact match against excluded paths list
//
// Parameters:
//   - path: Request path to check
//   - excludePaths: List of paths to exclude
//
// Returns:
//   - bool: true if path is in exclude list
func isPathExcluded(path string, excludePaths []string) bool {
	for _, excluded := range excludePaths {
		if strings.HasPrefix(path, excluded) {
			return true
		}
	}
	return false
}

// isExtensionExcluded checks if file extension should be excluded
// Case-insensitive comparison against excluded extensions list
//
// Parameters:
//   - path: Request path to check
//   - excludeExtensions: List of extensions to exclude (e.g., ".jpg", ".png")
//
// Returns:
//   - bool: true if extension is in exclude list

func isExtensionExcluded(path string, excludeExtensions []string) bool {
	path = strings.ToLower(path)
	for _, ext := range excludeExtensions {
		if strings.HasSuffix(path, strings.ToLower(ext)) {
			return true
		}
	}
	return false
}

// CompressWithLevel creates compression middleware with specific compression level
// Convenience method for setting compression level without full config
//
// Compression levels:
//   - 0: No compression (CompressionLevelNone)
//   - 1: Fastest compression (CompressionLevelFastest)
//   - 6: Default compression (CompressionLevelDefault)
//   - 9: Best compression (CompressionLevelBest)
//
// # Higher levels use more CPU but provide better compression ratios
//
// Parameters:
//   - level: CompressionLevel (0-9, or -1 for default)
//
// Returns:
//   - MiddlewareFunc with specified compression level
//
// Example:
//
//	app.Use(blaze.CompressWithLevel(blaze.CompressionLevelBest))
func CompressWithLevel(level CompressionLevel) MiddlewareFunc {
	config := DefaultCompressionConfig()
	config.Level = level
	return CompressWithConfig(config)
}

// CompressMinLength creates compression middleware with minimum length threshold
// Only compresses responses larger than the specified size
//
// Useful for avoiding compression overhead on small responses
// Small responses often don't benefit from compression and may even increase in size
//
// Parameters:
//   - minLength: Minimum response size in bytes to trigger compression
//
// Returns:
//   - MiddlewareFunc that only compresses responses above threshold
//
// Example:
//
//	app.Use(blaze.CompressMinLength(2048)) // Only compress responses > 2KB
func CompressMinLength(minLength int) MiddlewareFunc {
	config := DefaultCompressionConfig()
	config.MinLength = minLength
	return CompressWithConfig(config)
}

// CompressGzipOnly creates middleware that only uses gzip compression
// Disables deflate and brotli, using only gzip encoding
//
// Gzip is the most widely supported compression format and often the best choice
// for maximum compatibility across all browsers and HTTP clients
//
// Returns:
//   - MiddlewareFunc using only gzip compression
//
// Example:
//
//	app.Use(blaze.CompressGzipOnly())
func CompressGzipOnly() MiddlewareFunc {
	config := DefaultCompressionConfig()
	config.EnableGzip = true
	config.EnableDeflate = false
	config.EnableBrotli = false
	return CompressWithConfig(config)
}

// CompressTypes creates compression middleware for specific content types
// Only compresses responses with content types in the provided list
//
// # Useful for targeting specific content types while excluding everything else
//
// Parameters:
//   - contentTypes: List of content types to compress (e.g., "application/json")
//
// Returns:
//   - MiddlewareFunc that only compresses specified content types
//
// Example:
//
//	app.Use(blaze.CompressTypes("application/json", "text/html"))
func CompressTypes(contentTypes ...string) MiddlewareFunc {
	config := DefaultCompressionConfig()
	config.IncludeContentTypes = contentTypes
	return CompressWithConfig(config)
}
