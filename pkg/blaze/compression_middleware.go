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

const (
	// Compression levels
	CompressionLevelDefault CompressionLevel = -1
	CompressionLevelNone    CompressionLevel = 0
	CompressionLevelBest    CompressionLevel = 9
	CompressionLevelFastest CompressionLevel = 1
)

// CompressionConfig holds compression middleware configuration
type CompressionConfig struct {
	// Compression level (0-9, -1 for default)
	Level CompressionLevel

	// Minimum response size to compress (in bytes)
	MinLength int

	// Content types to compress (if empty, compresses all except excluded)
	IncludeContentTypes []string

	// Content types to exclude from compression
	ExcludeContentTypes []string

	// Enable gzip compression (default: true)
	EnableGzip bool

	// Enable deflate compression (default: true)
	EnableDeflate bool

	// Enable brotli compression (requires external package)
	EnableBrotli bool

	// Exclude paths from compression
	ExcludePaths []string

	// Exclude extensions from compression
	ExcludeExtensions []string

	// Enable compression for HTTPS (disabled by default for security)
	EnableForHTTPS bool
}

// DefaultCompressionConfig returns default compression configuration
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

// Pool for gzip writers
var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

// Pool for flate writers
var flateWriterPool = sync.Pool{
	New: func() interface{} {
		w, _ := flate.NewWriter(io.Discard, flate.DefaultCompression)
		return w
	},
}

// Pool for buffers
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// Compress creates compression middleware with default configuration
func Compress() MiddlewareFunc {
	return CompressWithConfig(DefaultCompressionConfig())
}

// CompressWithConfig creates compression middleware with custom configuration
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
func isPathExcluded(path string, excludePaths []string) bool {
	for _, excluded := range excludePaths {
		if strings.HasPrefix(path, excluded) {
			return true
		}
	}
	return false
}

// isExtensionExcluded checks if file extension should be excluded
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
func CompressWithLevel(level CompressionLevel) MiddlewareFunc {
	config := DefaultCompressionConfig()
	config.Level = level
	return CompressWithConfig(config)
}

// CompressMinLength creates compression middleware with minimum length threshold
func CompressMinLength(minLength int) MiddlewareFunc {
	config := DefaultCompressionConfig()
	config.MinLength = minLength
	return CompressWithConfig(config)
}

// CompressGzipOnly creates middleware that only uses gzip compression
func CompressGzipOnly() MiddlewareFunc {
	config := DefaultCompressionConfig()
	config.EnableGzip = true
	config.EnableDeflate = false
	config.EnableBrotli = false
	return CompressWithConfig(config)
}

// CompressTypes creates compression middleware for specific content types
func CompressTypes(contentTypes ...string) MiddlewareFunc {
	config := DefaultCompressionConfig()
	config.IncludeContentTypes = contentTypes
	return CompressWithConfig(config)
}
