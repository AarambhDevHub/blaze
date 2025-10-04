# Static File Serving

Blaze provides comprehensive static file serving capabilities with advanced features including caching, compression, directory browsing, range requests, and security controls. This guide covers all aspects of serving static files in your Blaze applications.

## Table of Contents

- [Overview](#overview)
- [Basic Static File Serving](#basic-static-file-serving)
- [Static File Configuration](#static-file-configuration)
- [Static File Methods](#static-file-methods)
- [Static File Security](#static-file-security)
- [Performance Optimization](#performance-optimization)
- [Advanced Features](#advanced-features)
- [Production Configuration](#production-configuration)
- [Best Practices](#best-practices)

## Overview

Blaze's static file serving system provides:

- **Fast File Serving**: Optimized file delivery using FastHTTP
- **Directory Browsing**: Optional directory listing generation
- **Compression**: Automatic gzip compression for supported files
- **Caching**: ETag and Last-Modified header support
- **Range Requests**: Byte-range support for large files and video streaming
- **MIME Types**: Automatic content-type detection with custom overrides
- **Security**: Directory traversal protection and file exclusion patterns

## Basic Static File Serving

### Serve Entire Directory

```go
app := blaze.New()

// Serve files from ./public directory at /static prefix
app.Static("/static", "./public")

// Access files at:
// /static/css/main.css -> ./public/css/main.css
// /static/js/app.js -> ./public/js/app.js
// /static/images/logo.png -> ./public/images/logo.png
```

### Serve Single File

```go
// Serve a specific file at a route
app.File("/favicon.ico", "./public/favicon.ico")
app.File("/robots.txt", "./public/robots.txt")

// Access at:
// /favicon.ico -> ./public/favicon.ico
// /robots.txt -> ./public/robots.txt
```

## Static File Configuration

### StaticConfig Structure

```go
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
    
    // Cache control max-age in seconds (default: 3600 / 1 hour)
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
```

### Default Configuration

```go
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
```

### Custom Static Configuration

```go
app := blaze.New()

// Create custom static config
staticConfig := blaze.DefaultStaticConfig("./public")
staticConfig.Index = "home.html"
staticConfig.Browse = false  // Disable directory browsing
staticConfig.Compress = true
staticConfig.CacheDuration = 24 * time.Hour
staticConfig.GenerateETag = true
staticConfig.ByteRange = true  // Enable range requests for video
staticConfig.Exclude = []string{".git", ".env", ".config"}

// Custom MIME types
staticConfig.MIMETypes = map[string]string{
    ".md":   "text/markdown",
    ".wasm": "application/wasm",
}

// Custom 404 handler
staticConfig.NotFoundHandler = func(c *blaze.Context) error {
    return c.Status(404).JSON(blaze.Map{
        "error": "File not found",
        "path":  c.Path(),
    })
}

// Modify response before sending
staticConfig.Modify = func(c *blaze.Context) error {
    c.SetHeader("X-Served-By", "Blaze")
    return nil
}

// Use custom configuration
app.StaticFS("/static", staticConfig)
```

## Static File Methods

### App-Level Methods

```go
// Static serves files from directory with default config
func (a *App) Static(prefix, root string) *App

// StaticFS serves files with custom configuration
func (a *App) StaticFS(prefix string, config StaticConfig) *App

// File serves a single specific file
func (a *App) File(path, filepath string) *App
```

### Context-Level File Operations

```go
// SendFile sends a file as response
func (c *Context) SendFile(filepath string) error

// ServeFile serves a file with proper headers
func (c *Context) ServeFile(filepath string) error

// ServeFileDownload forces download with custom filename
func (c *Context) ServeFileDownload(filepath, filename string) error

// ServeFileInline serves file for inline display
func (c *Context) ServeFileInline(filepath string) error

// StreamFile streams file with range request support
func (c *Context) StreamFile(filepath string) error

// FileExists checks if a file exists
func (c *Context) FileExists(filepath string) bool

// GetFileInfo returns file information
func (c *Context) GetFileInfo(filepath string) (os.FileInfo, error)

// Download is an alias for ServeFileDownload
func (c *Context) Download(filepath, filename string) error

// Attachment is an alias for ServeFileDownload
func (c *Context) Attachment(filepath, filename string) error
```

## Static File Security

### Directory Traversal Protection

Blaze automatically prevents directory traversal attacks:

```go
// Automatically protected against:
// /static/../../../etc/passwd
// /static/../../secret/config.yaml
// /static/..\windows\system32\config\sam

// Security checks ensure path stays within root directory
```

### File Exclusion Patterns

Exclude sensitive files and directories:

```go
staticConfig := blaze.DefaultStaticConfig("./public")

// Exclude patterns
staticConfig.Exclude = []string{
    ".git",          // Git repository
    ".svn",          // SVN repository
    ".env",          // Environment files
    ".DS_Store",     // macOS metadata
    "Thumbs.db",     // Windows thumbnails
    ".htaccess",     // Apache config
    ".gitignore",    // Git ignore
    "*.bak",         // Backup files
    "*.tmp",         // Temporary files
}

app.StaticFS("/static", staticConfig)
```

### Disable Directory Browsing

```go
staticConfig := blaze.DefaultStaticConfig("./public")
staticConfig.Browse = false  // Disable directory listing for security

app.StaticFS("/static", staticConfig)

// Requests to directories without index files return 403 Forbidden
```

## Performance Optimization

### Compression

Enable automatic compression for text-based files:

```go
staticConfig := blaze.DefaultStaticConfig("./public")
staticConfig.Compress = true  // Enable gzip compression

app.StaticFS("/static", staticConfig)

// Automatically compresses:
// - HTML, CSS, JavaScript
// - JSON, XML
// - SVG images
// - Text files
```

### Caching with ETags

Enable ETag generation for efficient caching:

```go
staticConfig := blaze.DefaultStaticConfig("./public")
staticConfig.GenerateETag = true
staticConfig.CacheDuration = 24 * time.Hour

app.StaticFS("/static", staticConfig)

// Generates ETags based on:
// - File modification time
// - File size
// - Supports If-None-Match header for 304 Not Modified
```

### Byte Range Requests

Enable range requests for streaming large files:

```go
staticConfig := blaze.DefaultStaticConfig("./videos")
staticConfig.ByteRange = true  // Enable range requests

app.StaticFS("/videos", staticConfig)

// Supports:
// - Video seeking in browser
// - Resume downloads
// - Partial content delivery (206 Partial Content)
// - Range: bytes=0-1023 requests
```

### Combined Optimization

```go
func setupOptimizedStatic(app *blaze.App) {
    // Images - long cache, no compression
    imagesConfig := blaze.DefaultStaticConfig("./public/images")
    imagesConfig.CacheDuration = 7 * 24 * time.Hour  // 7 days
    imagesConfig.Compress = false  // Don't compress images
    imagesConfig.GenerateETag = true
    app.StaticFS("/images", imagesConfig)
    
    // CSS/JS - long cache with compression
    assetsConfig := blaze.DefaultStaticConfig("./public/assets")
    assetsConfig.CacheDuration = 30 * 24 * time.Hour  // 30 days
    assetsConfig.Compress = true
    assetsConfig.GenerateETag = true
    app.StaticFS("/assets", assetsConfig)
    
    // Videos - range requests enabled
    videosConfig := blaze.DefaultStaticConfig("./public/videos")
    videosConfig.ByteRange = true
    videosConfig.Compress = false
    videosConfig.CacheDuration = 24 * time.Hour
    app.StaticFS("/videos", videosConfig)
    
    // HTML - short cache
    htmlConfig := blaze.DefaultStaticConfig("./public/html")
    htmlConfig.CacheDuration = 1 * time.Hour
    htmlConfig.Compress = true
    app.StaticFS("/pages", htmlConfig)
}
```

## Advanced Features

### Custom MIME Types

Define custom content types for file extensions:

```go
staticConfig := blaze.DefaultStaticConfig("./public")

// Custom MIME type mappings
staticConfig.MIMETypes = map[string]string{
    ".md":        "text/markdown",
    ".wasm":      "application/wasm",
    ".webmanifest": "application/manifest+json",
    ".apk":       "application/vnd.android.package-archive",
    ".ts":        "video/mp2t",
}

app.StaticFS("/static", staticConfig)
```

### Response Modification

Modify responses before sending files:

```go
staticConfig := blaze.DefaultStaticConfig("./public")

staticConfig.Modify = func(c *blaze.Context) error {
    // Add custom headers
    c.SetHeader("X-Served-By", "Blaze")
    c.SetHeader("X-Content-Source", "CDN")
    
    // CORS headers for fonts
    if strings.HasSuffix(c.Path(), ".woff2") || 
       strings.HasSuffix(c.Path(), ".woff") {
        c.SetHeader("Access-Control-Allow-Origin", "*")
    }
    
    // Security headers
    c.SetHeader("X-Content-Type-Options", "nosniff")
    
    return nil
}

app.StaticFS("/static", staticConfig)
```

### Custom 404 Handler

Handle missing files with custom logic:

```go
staticConfig := blaze.DefaultStaticConfig("./public")

staticConfig.NotFoundHandler = func(c *blaze.Context) error {
    // Log missing file
    log.Printf("File not found: %s", c.Path())
    
    // Return custom JSON error
    return c.Status(404).JSON(blaze.Map{
        "error":     "File not found",
        "path":      c.Path(),
        "timestamp": time.Now(),
    })
}

app.StaticFS("/static", staticConfig)
```

### Directory Browsing

Enable directory listing with custom styling:

```go
staticConfig := blaze.DefaultStaticConfig("./public")
staticConfig.Browse = true  // Enable directory browsing

app.StaticFS("/files", staticConfig)

// Generates HTML directory listing with:
// - Parent directory link
// - File names with links
// - File sizes (formatted)
// - Last modified timestamps
// - Separate directory and file sections
```

## Production Configuration

### High-Performance Static Server

```go
func setupProductionStatic(app *blaze.App) {
    // Main static assets
    staticConfig := blaze.DefaultStaticConfig("./public")
    staticConfig.Compress = true
    staticConfig.CacheDuration = 30 * 24 * time.Hour  // 30 days
    staticConfig.GenerateETag = true
    staticConfig.ByteRange = true
    staticConfig.Browse = false  // Disable for security
    
    // Exclude sensitive files
    staticConfig.Exclude = []string{
        ".git", ".svn", ".env", ".config",
        ".htaccess", "*.bak", "*.tmp",
    }
    
    // Security headers
    staticConfig.Modify = func(c *blaze.Context) error {
        c.SetHeader("X-Content-Type-Options", "nosniff")
        c.SetHeader("X-Frame-Options", "DENY")
        c.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")
        return nil
    }
    
    app.StaticFS("/static", staticConfig)
}
```

### CDN Integration Pattern

```go
func setupCDNStatic(app *blaze.App) {
    staticConfig := blaze.DefaultStaticConfig("./public")
    
    // Long cache for CDN
    staticConfig.CacheDuration = 365 * 24 * time.Hour  // 1 year
    staticConfig.GenerateETag = true
    
    // Add CDN headers
    staticConfig.Modify = func(c *blaze.Context) error {
        c.SetHeader("Cache-Control", 
            fmt.Sprintf("public, max-age=%d, immutable", 
                int(staticConfig.CacheDuration.Seconds())))
        c.SetHeader("X-CDN-Cache", "HIT")
        return nil
    }
    
    app.StaticFS("/cdn", staticConfig)
}
```

### Multi-Domain Static Serving

```go
func setupMultiDomainStatic(app *blaze.App) {
    // Main website assets
    mainConfig := blaze.DefaultStaticConfig("./public/main")
    mainConfig.CacheDuration = 24 * time.Hour
    app.StaticFS("/static", mainConfig)
    
    // Admin panel assets
    adminConfig := blaze.DefaultStaticConfig("./public/admin")
    adminConfig.CacheDuration = 1 * time.Hour
    app.StaticFS("/admin/static", adminConfig)
    
    // API documentation assets
    docsConfig := blaze.DefaultStaticConfig("./public/docs")
    docsConfig.Browse = true  // Allow directory browsing for docs
    docsConfig.CacheDuration = 1 * time.Hour
    app.StaticFS("/docs", docsConfig)
}
```

## Best Practices

### 1. Use Appropriate Cache Durations

```go
// Images, fonts: Long cache (30-365 days)
imagesConfig := blaze.DefaultStaticConfig("./public/images")
imagesConfig.CacheDuration = 30 * 24 * time.Hour

// CSS/JS with versioning: Very long cache
assetsConfig := blaze.DefaultStaticConfig("./public/assets")
assetsConfig.CacheDuration = 365 * 24 * time.Hour

// HTML: Short cache (1 hour)
htmlConfig := blaze.DefaultStaticConfig("./public/html")
htmlConfig.CacheDuration = 1 * time.Hour
```

### 2. Security Hardening

```go
staticConfig := blaze.DefaultStaticConfig("./public")

// Disable directory browsing
staticConfig.Browse = false

// Exclude sensitive patterns
staticConfig.Exclude = []string{
    ".git", ".svn", ".env", ".htaccess",
    "*.bak", "*.tmp", "*.log",
}

// Add security headers
staticConfig.Modify = func(c *blaze.Context) error {
    c.SetHeader("X-Content-Type-Options", "nosniff")
    c.SetHeader("X-Frame-Options", "SAMEORIGIN")
    return nil
}

app.StaticFS("/static", staticConfig)
```

### 3. Performance Optimization

```go
// Combine with compression middleware
app.Use(blaze.Compress())

// Use cache middleware for API responses
app.Use(blaze.CacheStatic())

// Optimize static config
staticConfig := blaze.DefaultStaticConfig("./public")
staticConfig.Compress = true
staticConfig.ByteRange = true
staticConfig.GenerateETag = true
staticConfig.CacheDuration = 24 * time.Hour

app.StaticFS("/static", staticConfig)
```

### 4. Development vs Production

```go
func setupStatic(app *blaze.App, isDev bool) {
    staticConfig := blaze.DefaultStaticConfig("./public")
    
    if isDev {
        // Development: Short cache, browsing enabled
        staticConfig.CacheDuration = 0  // No caching
        staticConfig.Browse = true
        staticConfig.GenerateETag = false
    } else {
        // Production: Long cache, security hardened
        staticConfig.CacheDuration = 30 * 24 * time.Hour
        staticConfig.Browse = false
        staticConfig.GenerateETag = true
        staticConfig.Exclude = []string{".git", ".env", "*.bak"}
    }
    
    app.StaticFS("/static", staticConfig)
}
```

### 5. Error Handling

```go
staticConfig := blaze.DefaultStaticConfig("./public")

staticConfig.NotFoundHandler = func(c *blaze.Context) error {
    // Log for monitoring
    log.Printf("Static file not found: %s from IP: %s", 
        c.Path(), c.IP())
    
    // Return appropriate error
    if strings.Contains(c.Header("Accept"), "application/json") {
        return c.Status(404).JSON(blaze.Map{
            "error": "File not found",
            "path":  c.Path(),
        })
    }
    
    return c.Status(404).HTML("<h1>404 - File Not Found</h1>")
}

app.StaticFS("/static", staticConfig)
```

### 6. Monitoring and Logging

```go
staticConfig := blaze.DefaultStaticConfig("./public")

staticConfig.Modify = func(c *blaze.Context) error {
    // Log file access for analytics
    log.Printf("Static file served: %s, Size: %d, IP: %s",
        c.Path(), c.Response().Header.ContentLength(), c.IP())
    
    // Add monitoring headers
    c.SetHeader("X-Request-ID", c.GetUserValueString("request_id"))
    c.SetHeader("X-Served-At", time.Now().Format(time.RFC3339))
    
    return nil
}

app.StaticFS("/static", staticConfig)
```

Blaze's static file serving system provides a production-ready, high-performance solution for serving static assets with comprehensive security, caching, and optimization features built-in.