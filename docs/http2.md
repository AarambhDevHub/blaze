# HTTP/2 Documentation

HTTP/2 support in Blaze provides enhanced performance, multiplexing capabilities, and modern web protocol features including server push, stream prioritization, and binary framing.

## Overview

HTTP/2 is a major revision of the HTTP protocol that addresses performance limitations of HTTP/1.1 through binary framing, stream multiplexing, and server push capabilities. Blaze implements HTTP/2 using Go's `net/http` and `golang.org/x/net/http2` packages with full FastHTTP integration.

## Configuration

### Basic HTTP/2 Setup

```go
package main

import (
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
    // Create app with production config (HTTP/2 enabled by default)
    app := blaze.NewWithConfig(blaze.ProductionConfig())
    
    // Enable TLS (required for HTTP/2 over TLS)
    app.EnableAutoTLS("localhost", "127.0.0.1")
    
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "message": "Hello HTTP/2!",
            "protocol": c.Protocol(),
            "http2": c.IsHTTP2(),
        })
    })
    
    app.ListenAndServe()
}
```

### HTTP/2 Configuration Options

Blaze provides comprehensive HTTP/2 configuration through the `HTTP2Conf.

```go
config := &blaze.HTTP2Config{
    Enabled:                      true,
    H2C:                          false,        // HTTP/2 over cleartext
    MaxConcurrentStreams:         1000,         // Max concurrent streams
    MaxUploadBufferPerStream:     1048576,      // 1MB per stream
    MaxUploadBufferPerConnection: 1048576,      // 1MB per connection
    EnablePush:                   true,         // Server push support
    IdleTimeout:                  300 * time.Second,
    ReadTimeout:                  30 * time.Second,
    WriteTimeout:                 30 * time.Second,
    MaxDecoderHeaderTableSize:    4096,         // HPACK compression
    MaxEncoderHeaderTableSize:    4096,
    MaxReadFrameSize:             1048576,      // 1MB max frame size
}

app := blaze.New()
app.SetHTTP2Config(config)
```

## Development vs Production

### Development Configuration

For development environments, Blaze supports HTTP/2 over cleartext (h2c).

```go
func developmentSetup() {
    config := blaze.DevelopmentHTTP2Config()
    config.H2C = true // Enable cleartext HTTP/2
    
    app := blaze.NewWithConfig(blaze.DevelopmentConfig())
    app.SetHTTP2Config(config)
    
    // No TLS required for h2c
    app.ListenAndServe() // Runs on http://localhost:3000
}
```

### Production Configuration

Production environments use HTTP/2 over TLS wit.

```go
func productionSetup() {
    app := blaze.NewWithConfig(blaze.ProductionConfig())
    
    // Configure TLS with HTTP/2 support
    tlsConfig := blaze.DefaultTLSConfig()
    tlsConfig.NextProtos = []string{"h2", "http/1.1"}
    app.SetTLSConfig(tlsConfig)
    
    app.ListenAndServe() // Runs on https://0.0.0.0:443
}
```

## Server Push

HTTP/2 Server Push allows the server to proactively send resources to clients before they request them, reducing latency and improving performance.

### Basic Server Push

```go
app.GET("/", func(c *blaze.Context) error {
    // Push CSS and JavaScript resources
    resources := map[string]string{
        "/static/style.css":  "style",
        "/static/script.js":  "script",
        "/static/logo.png":   "image",
    }
    
    if err := c.PushResources(resources); err != nil {
        // Fallback gracefully if push fails
        log.Printf("Server push failed: %v", err)
    }
    
    return c.HTML(`
        <!DOCTYPE html>
        <html>
        <head>
            <link rel="stylesheet" href="/static/style.css">
            <script src="/static/script.js"></script>
        </head>
        <body>
            <h1>HTTP/2 Server Push Demo</h1>
            <img src="/static/logo.png" alt="Logo">
        </body>
        </html>
    `)
})
```

### Advanced Server Push

```go
func advancedPushHandler(c *blaze.Context) error {
    // Check if HTTP/2 is available
    if !c.IsHTTP2() {
        return c.HTML(htmlWithoutPush)
    }
    
    // Conditional push based on user agent or other factors
    userAgent := c.UserAgent()
    if strings.Contains(userAgent, "Mobile") {
        // Push mobile-optimized resources
        c.ServerPush("/static/mobile.css", "style")
        c.ServerPush("/static/mobile.js", "script")
    } else {
        // Push desktop resources
        c.ServerPush("/static/desktop.css", "style")
        c.ServerPush("/static/desktop.js", "script")
    }
    
    return c.HTML(htmlTemplate)
}
```

## Stream Management

HTTP/2 uses streams for multiplexed communication, allowing multiple requests over a single connection.

### Stream Information

```go
app.Use(blaze.StreamInfo()) // Adds stream debugging info

app.GET("/stream-info", func(c *blaze.Context) error {
    return c.JSON(blaze.Map{
        "stream_id":     c.StreamID(),
        "protocol":      c.Protocol(),
        "http2_enabled": c.IsHTTP2(),
        "method":        c.Method(),
        "path":          c.Path(),
    })
})
```

### Stream Prioritization

While Blaze handles stream prioritization automatically, you can influence it through response headers :

```go
app.GET("/priority/:level", func(c *blaze.Context) error {
    level := c.Param("level")
    
    switch level {
    case "high":
        c.SetHeader("X-Priority", "high")
    case "low":
        c.SetHeader("X-Priority", "low")
    }
    
    return c.JSON(blaze.Map{
        "priority": level,
        "stream_id": c.StreamID(),
    })
})
```

## Performance Benefits

HTTP/2 provides significant performance improvements over HTTP/1.1 :

### Multiplexing

```go
// Multiple concurrent requests over single connection
app.GET("/api/users", func(c *blaze.Context) error {
    // Simulate database query
    users := fetchUsers()
    return c.JSON(users)
})

app.GET("/api/posts", func(c *blaze.Context) error {
    // Concurrent with users request
    posts := fetchPosts()
    return c.JSON(posts)
})

app.GET("/api/comments", func(c *blaze.Context) error {
    // All three can execute concurrently
    comments := fetchComments()
    return c.JSON(comments)
})
```

### Header Compression

HTTP/2 automatically compresses headers using HPACK :

```go
app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        // Headers are automatically compressed in HTTP/2
        c.SetHeader("X-API-Version", "v2.0")
        c.SetHeader("X-Rate-Limit", "1000")
        c.SetHeader("X-Response-Time", "50ms")
        
        return next(c)
    }
})
```

## Middleware Integration

### HTTP/2 Specific Middleware

```go
// HTTP/2 detection and optimization
app.Use(blaze.HTTP2Middleware())

// Security headers optimized for HTTP/2
app.Use(blaze.HTTP2Security())

// Compression for HTTP/2
app.Use(blaze.CompressHTTP2(6))

// Metrics collection for HTTP/2
app.Use(blaze.HTTP2Metrics())
```

### Custom HTTP/2 Middleware

```go
func HTTP2OptimizationMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            if c.IsHTTP2() {
                // Enable HTTP/2 specific optimizations
                c.SetHeader("X-HTTP2-Optimized", "true")
                c.SetLocals("http2_enabled", true)
                c.SetLocals("protocol", "HTTP/2.0")
                
                // Set connection reuse headers
                c.SetHeader("Connection", "keep-alive")
                c.SetHeader("Keep-Alive", "timeout=30, max=1000")
            }
            
            return next(c)
        }
    }
}
```

## Protocol Negotiation

Blaze handles HTTP/2 protocol negotiation automatically through ALPN (Application-Layer Protocol Negotiation) :

### TLS Configuration with ALPN

```go
tlsConfig := &blaze.TLSConfig{
    CertFile:   "server.crt",
    KeyFile:    "server.key",
    NextProtos: []string{"h2", "http/1.1"}, // ALPN negotiation
    MinVersion: tls.VersionTLS12,
}

app.SetTLSConfig(tlsConfig)
```

### Protocol Detection

```go
app.GET("/protocol", func(c *blaze.Context) error {
    return c.JSON(blaze.Map{
        "protocol":           c.Protocol(),
        "is_http2":          c.IsHTTP2(),
        "alpn_negotiated":   c.GetLocals("alpn_protocol"),
        "connection_reused": c.GetLocals("connection_reused"),
    })
})
```

## Error Handling

HTTP/2 has specific error handling mechanisms for streams and connections :

### Stream Errors

```go
app.GET("/stream-error", func(c *blaze.Context) error {
    if c.IsHTTP2() {
        // HTTP/2 stream-level error
        c.SetHeader("X-Stream-Error", "INTERNAL_ERROR")
        return c.Status(500).JSON(blaze.Map{
            "error": "Stream processing failed",
            "stream_id": c.StreamID(),
        })
    }
    
    return c.Status(500).JSON(blaze.Map{
        "error": "Request processing failed",
    })
})
```

### Connection Management

```go
app.Use(blaze.ShutdownAware()) // Graceful HTTP/2 connection handling

app.RegisterGracefulTask(func(ctx context.Context) error {
    // Custom cleanup for HTTP/2 connections
    log.Println("Closing HTTP/2 connections gracefully...")
    return nil
})
```

## Monitoring and Debugging

### HTTP/2 Health Checks

```go
app.GET("/health/http2", func(c *blaze.Context) error {
    serverInfo := app.GetServerInfo()
    
    return c.JSON(blaze.Map{
        "http2_enabled": serverInfo.EnableHTTP2,
        "tls_enabled":   serverInfo.EnableTLS,
        "http2_stats":   serverInfo.HTTP2,
        "protocol":      c.Protocol(),
        "stream_id":     c.StreamID(),
    })
})
```

### Performance Monitoring

```go
app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        start := time.Now()
        
        err := next(c)
        
        duration := time.Since(start)
        if c.IsHTTP2() {
            // Log HTTP/2 specific metrics
            log.Printf("HTTP/2 Request: %s %s - %d - %v - Stream: %d",
                c.Method(), c.Path(), 
                c.Response().StatusCode(),
                duration, c.StreamID())
        }
        
        return err
    }
})
```

## Best Practices

### Resource Optimization

1. **Minimize Server Push**: Only push critical resources that are definitely needed
2. **Use Stream Prioritization**: Prioritize critical resources like CSS over images
3. **Enable Compression**: HPACK compression is automatic, but optimize payload size

### Connection Management

```go
// Optimize connection settings
config := &blaze.HTTP2Config{
    MaxConcurrentStreams:         1000,  // Balance concurrency vs memory
    MaxUploadBufferPerConnection: 2 << 20, // 2MB for large uploads
    IdleTimeout:                  300 * time.Second, // Keep connections alive
}
```

### Security Considerations

```go
app.Use(blaze.HTTP2Security()) // Security headers for HTTP/2

// Custom security middleware
app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        if c.IsHTTP2() {
            // HTTP/2 specific security headers
            c.SetHeader("Strict-Transport-Security", "max-age=31536000")
            c.SetHeader("X-Content-Type-Options", "nosniff")
        }
        return next(c)
    }
})
```

## Troubleshooting

### Common Issues

1. **TLS Configuration**: HTTP/2 requires proper TLS setu.
2. **Certificate Problems**: Ensure certificates include HTT.
3. **Firewall Issues**: HTTP/2 uses the same ports (443/80) but different protocols

### Debugging Tools

```go
// Enable HTTP/2 debugging
app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        if c.IsHTTP2() {
            log.Printf("HTTP/2 Debug: Stream %d, Method: %s, Path: %s",
                c.StreamID(), c.Method(), c.Path())
        }
        return next(c)
    }
})
```

HTTP/2 support in Blaze provides a modern, high-performance foundation for web applications with automatic protocol negotiation, server push capabilities, and comprehensive configuration options suitable for both development and production environments.