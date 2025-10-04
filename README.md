# Blaze 🔥

A blazing-fast, production-ready web framework for Go that combines the performance of FastHTTP with the elegance of modern web frameworks like Axum and Actix Web.

## 🚀 Performance First

Blaze delivers **exceptional performance** with enterprise-grade features:

### Cache Performance (Optimized)
```
Requests/sec: 190,376.62
Transfer/sec:  118.38MB
Latency:       527.70μs avg (±765.78μs)
Max Latency:   11.73ms
Memory Usage:  Ultra-low footprint with intelligent caching
```

### Standard Performance (Without Cache)
```
Requests/sec: 182,505.60
Transfer/sec:  83.20MB
Latency:       790.07μs avg (±1.04ms)
Max Latency:   11.99ms
Memory Usage:  Ultra-low footprint
```

*Benchmarked with `wrk -c100 -d30s` on production-grade endpoints with 100 concurrent connections over 30 seconds.*

**🎯 Cache Performance Boost**: **+4.3% throughput**, **+42% data transfer**, **-33% latency** with built-in caching middleware.

## ✨ Enterprise Features

### 🔥 Core Performance
- **Lightning Fast**: Built on FastHTTP - 190K+ req/sec with caching, 182K+ req/sec sustained throughput
- **Intelligent Caching**: Built-in cache middleware with LRU/LFU/FIFO/Random eviction strategies
- **Zero-Copy**: Optimized memory usage with minimal allocations
- **HTTP/2 & h2c**: Full HTTP/2 support with server push capabilities
- **TLS/HTTPS**: Auto-TLS, custom certificates, and development-friendly SSL

### 🛡️ Production Ready
- **Type Safety**: Full compile-time type checking and validation with go-playground/validator
- **Graceful Shutdown**: Clean shutdown with connection draining and context awareness
- **Middleware Stack**: Composable middleware with CORS, CSRF, rate limiting, compression
- **Error Handling**: Comprehensive error handling with recovery and stack traces

### 📁 Advanced Features
- **Struct-Based Binding**: Bind multipart forms, JSON, and form data to structs with validation tags
- **File Upload System**: Single/multiple file uploads with validation and unique filename generation
- **WebSockets**: Real-time communication with connection management and broadcasting
- **Static File Serving**: Advanced configuration with caching, compression, ETag, and range requests
- **Validation System**: Integrated validation with automatic error formatting

### 🔧 Developer Experience
- **All HTTP Methods**: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, CONNECT, TRACE, ANY, Match
- **Route Constraints**: Integer, UUID, regex, and custom parameter validation
- **Route Groups**: Organized API versioning with shared middleware
- **Configuration Profiles**: Environment-specific configs (dev, staging, production)
- **Comprehensive Context**: Rich request/response handling with locals, timeouts, and shutdown awareness

## 📦 Installation

```go
go get github.com/AarambhDevHub/blaze
```

## 🚀 Quick Start

### Simple Server with Validation
```go
package main

import (
    "log"
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
    app := blaze.New()

    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "message": "Hello, Blaze! 🔥",
            "status":  "success",
            "version": "v0.1.3",
        })
    })

    // Route with validation
    type User struct {
        Name  string `json:"name" validate:"required,min=2,max=100"`
        Email string `json:"email" validate:"required,email"`
        Age   int    `json:"age" validate:"gte=18,lte=100"`
    }

    app.POST("/users", func(c *blaze.Context) error {
        var user User
        
        // Bind and validate in one call
        if err := c.BindJSONAndValidate(&user); err != nil {
            return c.Status(400).JSON(blaze.Map{"error": err.Error()})
        }
        
        return c.Status(201).JSON(user)
    })

    log.Printf("🔥 Blaze server starting on http://localhost:8080")
    log.Fatal(app.ListenAndServeGraceful())
}
```

### Production Configuration with Caching
```go
func main() {
    // Production-ready configuration
    config := blaze.ProductionConfig()
    config.Host = "0.0.0.0"
    config.Port = 80
    config.TLSPort = 443
    config.EnableHTTP2 = true
    config.EnableTLS = true

    app := blaze.NewWithConfig(config)

    // Enable auto-TLS
    app.EnableAutoTLS("yourdomain.com", "www.yourdomain.com")

    // Production middleware stack
    app.Use(blaze.Recovery())
    app.Use(blaze.LoggerMiddleware())
    app.Use(blaze.RequestIDMiddleware())
    app.Use(blaze.CORS(blaze.CORSOptions{
        AllowOrigins: []string{"https://yourdomain.com"},
        AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    }))
    app.Use(blaze.CSRF(blaze.ProductionCSRFOptions([]byte("secret"))))
    app.Use(blaze.RateLimitMiddleware(blaze.RateLimitOptions{
        MaxRequests: 1000,
        Window:      time.Hour,
    }))
    app.Use(blaze.CompressWithLevel(blaze.CompressionLevelBest))
    app.Use(blaze.Cache(blaze.ProductionCacheOptions()))

    // Your routes...

    log.Fatal(app.ListenAndServeGraceful(syscall.SIGINT, syscall.SIGTERM))
}
```

## 📋 Core API Examples

### All HTTP Methods & Routing
```go
app := blaze.New()

// Standard RESTful routes
app.GET("/users", getUsers)              // List users
app.POST("/users", createUser)           // Create user  
app.GET("/users/:id", getUser)           // Get user by ID
app.PUT("/users/:id", updateUser)        // Update user
app.DELETE("/users/:id", deleteUser)     // Delete user
app.PATCH("/users/:id", patchUser)       // Partial update
app.HEAD("/users/:id", checkUser)        // Headers only
app.OPTIONS("/users", optionsUsers)      // CORS preflight

// Extended HTTP methods
app.CONNECT("/tunnel/:target", tunnelHandler)  // Tunnel connections
app.TRACE("/debug", traceHandler)              // Request tracing

// ANY route (handles all methods)
app.ANY("/api/health", func(c *blaze.Context) error {
    return c.JSON(blaze.Map{
        "status": "healthy",
        "method": c.Method(),
    })
})

// Match specific methods
app.Match([]string{"GET", "POST", "PUT"}, "/api/data", dataHandler)

// Route parameters with constraints
app.GET("/users/:id", getUserHandler,
    blaze.WithIntConstraint("id"))

app.GET("/items/:uuid", getItemHandler,
    blaze.WithUUIDConstraint("uuid"))

app.GET("/products/:sku", getProductHandler,
    blaze.WithRegexConstraint("sku", `^[A-Z]{2}-\d{4}$`))

// Wildcards
app.GET("/static/*filepath", serveStatic)
```

### Advanced Form Handling with Struct Binding
```go
type UserProfile struct {
    Name     string                `form:"name,required,minsize:2,maxsize:100"`
    Email    string                `form:"email,required"`
    Age      int                   `form:"age,required,default:18"`
    Avatar   *blaze.MultipartFile  `form:"avatar"`
    IsActive bool                  `form:"is_active"`
    Bio      string                `form:"bio,maxsize:500"`
    JoinedAt *time.Time            `form:"joined_at"`
    Tags     []string              `form:"tags"`
}

app.POST("/profile", func(c *blaze.Context) error {
    var profile UserProfile

    // Automatic form binding with validation
    if err := c.BindMultipartFormAndValidate(&profile); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }

    // Save avatar if uploaded
    if profile.Avatar != nil {
        savedPath, err := c.SaveUploadedFileWithUniqueFilename(profile.Avatar, "uploads/")
        if err != nil {
            return c.Status(500).JSON(blaze.Map{"error": "Failed to save avatar"})
        }
        log.Printf("Avatar saved: %s", savedPath)
    }

    return c.JSON(blaze.Map{
        "message": "Profile created successfully",
        "profile": profile,
    })
})
```

### WebSocket Support with Broadcasting
```go
type ChatHub struct {
    clients    map[*blaze.WebSocketConnection]bool
    broadcast  chan []byte
    register   chan *blaze.WebSocketConnection
    unregister chan *blaze.WebSocketConnection
}

hub := NewChatHub()
go hub.Run()

app.WebSocket("/ws/chat", func(ws *blaze.WebSocketConnection) error {
    hub.register <- ws
    defer func() { hub.unregister <- ws }()
    
    for {
        _, message, err := ws.ReadMessage()
        if err != nil {
            break
        }
        hub.broadcast <- message
    }
    
    return nil
})
```

### Comprehensive Middleware Stack
```go
// Global middleware with all features
app.Use(blaze.Recovery())                       // Panic recovery with stack traces
app.Use(blaze.LoggerMiddlewareWithConfig(logConfig))  // Configurable logging
app.Use(blaze.RequestIDMiddleware())           // Unique request IDs
app.Use(blaze.CORS(corsOpts))                  // CORS with fine-grained control
app.Use(blaze.CSRF(csrfOpts))                  // CSRF protection
app.Use(blaze.RateLimitMiddleware(rateOpts))   // Rate limiting per IP
app.Use(blaze.BodyLimitMB(10))                 // Request body size limits
app.Use(blaze.CompressWithLevel(9))            // Gzip/Deflate/Brotli compression
app.Use(blaze.Cache(cacheOpts))                // LRU/LFU/FIFO/Random caching
app.Use(blaze.ShutdownAware())                 // Graceful shutdown support

// Route-specific middleware
app.GET("/protected", protectedHandler,
    blaze.WithMiddleware(authMiddleware),
    blaze.WithMiddleware(rateLimitMiddleware))
```

### Route Groups & API Versioning
```go
// API v1 with shared middleware
v1 := app.Group("/api/v1")
v1.Use(blaze.LoggerMiddleware())
v1.Use(blaze.Auth(tokenValidator))
v1.Use(blaze.RateLimitMiddleware(rateLimitOpts))

v1.GET("/users", listUsers)
v1.POST("/users", createUser)
v1.GET("/users/:id", getUser, blaze.WithIntConstraint("id"))

// Admin nested group
admin := v1.Group("/admin")
admin.Use(RequireAdminMiddleware())

admin.GET("/stats", getAdminStats)
admin.POST("/users/:id/ban", banUser)
admin.ANY("/system/*path", adminSystemHandler)

// API v2 with different structure
v2 := app.Group("/api/v2")
v2.Use(authMiddleware)
v2.Use(validationMiddleware)

v2.GET("/profiles", getProfiles)
v2.CONNECT("/stream/:id", streamConnection)
v2.TRACE("/debug/:session", debugSession)
```

### File Uploads & Static Serving
```go
// Configure multipart handling
multipartConfig := blaze.ProductionMultipartConfig()
multipartConfig.MaxFileSize = 10 << 20  // 10MB
multipartConfig.MaxFiles = 5
multipartConfig.AllowedExtensions = []string{".jpg", ".png", ".pdf"}

app.Use(blaze.MultipartMiddleware(multipartConfig))

// Single file upload with validation
app.POST("/upload", func(c *blaze.Context) error {
    file, err := c.FormFile("document")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file uploaded"})
    }

    // Validate file type
    if !file.IsDocument() {
        return c.Status(400).JSON(blaze.Map{"error": "Only documents allowed"})
    }

    // Save with unique filename
    path, err := c.SaveUploadedFileWithUniqueFilename(file, "uploads/")
    if err != nil {
        return c.Status(500).JSON(blaze.Map{"error": "Save failed"})
    }

    return c.JSON(blaze.Map{
        "filename":     file.Filename,
        "saved_path":   path,
        "size":         file.Size,
        "content_type": file.ContentType,
    })
})

// Static file serving with advanced configuration
staticConfig := blaze.DefaultStaticConfig("./public")
staticConfig.Compress = true
staticConfig.CacheDuration = 24 * time.Hour
staticConfig.GenerateETag = true
staticConfig.ByteRange = true  // Enable range requests

app.StaticFS("/static", staticConfig)

// File download with range support
app.GET("/download/:filename", func(c *blaze.Context) error {
    filepath := "uploads/" + c.Param("filename")
    
    if !c.FileExists(filepath) {
        return c.Status(404).JSON(blaze.Map{"error": "File not found"})
    }
    
    // Stream file with range request support for videos
    return c.StreamFile(filepath)
})
```

### HTTP/2 with Server Push
```go
// HTTP/2 configuration
config := blaze.ProductionConfig()
config.EnableHTTP2 = true
config.EnableTLS = true

app := blaze.NewWithConfig(config)

// Configure TLS
tlsConfig := &blaze.TLSConfig{
    CertFile: "server.crt",
    KeyFile:  "server.key",
    MinVersion: tls.VersionTLS12,
    NextProtos: []string{"h2", "http/1.1"},
}
app.SetTLSConfig(tlsConfig)

// Configure HTTP/2
http2Config := &blaze.HTTP2Config{
    Enabled:              true,
    MaxConcurrentStreams: 1000,
    EnablePush:           true,
}
app.SetHTTP2Config(http2Config)

// Server push example
app.GET("/", func(c *blaze.Context) error {
    if c.IsHTTP2() {
        // Push critical resources
        c.PushResources(map[string]string{
            "/static/app.css": "style",
            "/static/app.js":  "script",
            "/static/logo.png": "image",
        })
        
        log.Printf("Processing on HTTP/2 stream %d", c.StreamID())
    }

    return c.HTML(`<!DOCTYPE html>
        <html>
        <head>
            <link rel="stylesheet" href="/static/app.css">
            <script src="/static/app.js"></script>
        </head>
        <body><h1>HTTP/2 with Server Push!</h1></body>
        </html>`)
})
```

## 🧪 Built-in Middleware

Blaze provides a comprehensive middleware ecosystem:

```go
// Core Middleware
app.Use(blaze.Recovery())                       // Panic recovery
app.Use(blaze.LoggerMiddleware())              // Request logging
app.Use(blaze.LoggerMiddlewareWithConfig(cfg)) // Configurable logging

// Security Middleware
app.Use(blaze.CORS(corsOpts))                  // CORS handling
app.Use(blaze.CSRF(csrfOpts))                  // CSRF protection
app.Use(blaze.Auth(tokenValidator))            // Authentication
app.Use(blaze.HTTP2Security())                 // HTTP/2 security headers

// Performance Middleware
app.Use(blaze.Cache(cacheOpts))                // LRU/LFU/FIFO/Random cache
app.Use(blaze.Compress())                      // Gzip compression
app.Use(blaze.CompressWithLevel(9))            // Custom compression level
app.Use(blaze.CompressTypes("text/html"))      // Compress specific types

// Request Control Middleware
app.Use(blaze.BodyLimit(10*1024*1024))         // Body size limits
app.Use(blaze.BodyLimitMB(10))                 // Body limit in MB
app.Use(blaze.RateLimitMiddleware(rateOpts))   // Rate limiting
app.Use(blaze.RequestIDMiddleware())           // Request ID generation

// Specialized Middleware
app.Use(blaze.ValidationMiddleware())          // Validation support
app.Use(blaze.MultipartMiddleware(config))     // Multipart form handling
app.Use(blaze.ShutdownAware())                 // Graceful shutdown
app.Use(blaze.GracefulTimeout(30*time.Second)) // Request timeouts
app.Use(blaze.HTTP2Info())                     // HTTP/2 protocol info
app.Use(blaze.StreamInfo())                    // HTTP/2 stream debugging
```

## 📊 Performance Comparison

| Framework | Req/sec | Latency | Memory | HTTP/2 | Validation | Cache | Notes |
|-----------|---------|---------|--------|---------|------------|-------|-------|
| **Blaze (Cache)** | **190K** | **0.53ms** | **Ultra Low** | ✅ | ✅ | ✅ | **+42% transfer, All features** |
| **Blaze** | **182K** | **0.79ms** | **Ultra Low** | ✅ | ✅ | ✅ | **Production Ready** |
| Fiber | 165K | 0.60ms | Low | ❌ | ❌ | ❌ | FastHTTP-based |
| FastHTTP | 200K+ | 0.5ms | Very Low | ❌ | ❌ | ❌ | Raw performance |
| Gin | 50K | 10ms | Medium | ❌ | Limited | ❌ | Most popular |
| Echo | 40K | 15ms | Medium | ❌ | Limited | ❌ | Minimalist |
| Chi | 35K | 20ms | Low | ❌ | ❌ | ❌ | Lightweight router |
| Go stdlib | 17K | 30ms | Medium | ✅ | ❌ | ❌ | Standard library |

**🏆 Performance Leader**: Blaze delivers the best real-world performance with comprehensive features.

## 🏗️ Complete Feature List

### Routing & Request Handling
- ✅ All HTTP methods: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, CONNECT, TRACE
- ✅ ANY route (handles all methods)
- ✅ Match route (handles specific multiple methods)
- ✅ Named parameters with type conversion (`:param`)
- ✅ Wildcard parameters (`*param`)
- ✅ Route constraints (int, UUID, regex, custom)
- ✅ Route groups with shared middleware
- ✅ Named routes with priorities and tags
- ✅ Query parameter handling with defaults

### Data Binding & Validation
- ✅ JSON body binding with validation
- ✅ Form data binding with validation
- ✅ Multipart form binding with struct tags
- ✅ Automatic validation with go-playground/validator
- ✅ Combined bind and validate methods (`BindAndValidate`, `BindJSONAndValidate`, `BindMultipartFormAndValidate`)
- ✅ Single variable validation
- ✅ Body size validation
- ✅ Custom validators and struct-level validation

### Response Types
- ✅ JSON responses with helpers (OK, Created, Error, Paginate)
- ✅ HTML responses
- ✅ Text responses
- ✅ File serving and downloads
- ✅ File streaming with range requests
- ✅ Redirects (301, 302, 307, 308)
- ✅ Custom status codes and headers
- ✅ Chainable response methods

### Middleware (Built-in)
- ✅ Logger with configurable options
- ✅ Recovery with stack traces
- ✅ CORS with fine-grained control
- ✅ CSRF protection with tokens
- ✅ Rate limiting (per IP or custom key)
- ✅ Caching (LRU, LFU, FIFO, Random)
- ✅ Compression (Gzip, Deflate, Brotli)
- ✅ Body limits (global and per-route)
- ✅ Authentication (token-based)
- ✅ Request ID generation
- ✅ Graceful shutdown awareness
- ✅ HTTP/2 specific middleware

### File Handling
- ✅ Single file uploads
- ✅ Multiple file uploads
- ✅ Struct-based multipart binding with validation
- ✅ File validation (size, type, extension)
- ✅ Unique filename generation
- ✅ Static file serving with advanced configuration
- ✅ Directory browsing (configurable)
- ✅ ETag generation
- ✅ Byte-range requests for video streaming
- ✅ MIME type detection
- ✅ Compression for static files

### WebSocket Support
- ✅ WebSocket upgrade
- ✅ Message reading/writing (text, binary)
- ✅ JSON message support
- ✅ Connection management
- ✅ Broadcasting with hub pattern
- ✅ Ping/Pong support
- ✅ Configurable timeouts and buffer sizes

### HTTP/2 Features
- ✅ Native HTTP/2 support
- ✅ Server push (single and multiple resources)
- ✅ Stream ID access
- ✅ Protocol detection
- ✅ h2c (HTTP/2 over cleartext)
- ✅ Configurable stream limits
- ✅ HTTP/2 specific middleware

### Security
- ✅ TLS configuration (production and development)
- ✅ Auto-generated self-signed certificates
- ✅ CSRF protection with tokens
- ✅ CORS configuration
- ✅ Security headers
- ✅ Directory traversal protection
- ✅ Rate limiting
- ✅ Body size limits
- ✅ Input validation

### Production Features
- ✅ Graceful shutdown with context awareness
- ✅ Health check endpoints
- ✅ Configuration profiles (dev, prod, custom)
- ✅ Application state management
- ✅ Request-scoped locals
- ✅ Comprehensive error handling
- ✅ Logging system
- ✅ Request timeouts with shutdown awareness
- ✅ Metrics and monitoring hooks

## 🧪 Load Testing Results

### With Cache Enabled - Superior Performance
```
wrk -c100 -d30s http://localhost:3000/
Running 30s test @ http://localhost:3000/
  2 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   527.70us  765.78us  11.73ms   89.78%
    Req/Sec    95.69k    20.89k  134.42k    68.83%
  5711615 requests in 30.00s, 3.47GB read
Requests/sec: 190376.62
Transfer/sec:    118.38MB
```

### Without Cache - Still Excellent Performance
```
wrk -c100 -d30s http://localhost:3000/
Running 30s test @ http://localhost:3000/
  2 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   790.07us    1.04ms  11.99ms   85.35%
    Req/Sec    91.74k    19.41k  120.94k    48.33%
  5475380 requests in 30.00s, 2.44GB read
Requests/sec: 182505.60
Transfer/sec:     83.20MB
```

## 📈 Production Deployment

### Docker
```
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o blaze-app

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/blaze-app .
COPY --from=builder /app/static ./static

EXPOSE 8080
CMD ["./blaze-app"]
```

### Kubernetes
```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: blaze-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: blaze-app
  template:
    metadata:
      labels:
        app: blaze-app
    spec:
      containers:
      - name: blaze-app
        image: your-registry/blaze-app:latest
        ports:
        - containerPort: 8080
        env:
        - name: ENV
          value: "production"
        - name: CACHE_ENABLED
          value: "true"
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"
```

## 📚 Documentation

Comprehensive documentation is available in the `/docs` directory:

- **[Quick Start](docs/quick-start.md)** - Get started in minutes
- **[Configuration](docs/configuration.md)** - Application configuration
- **[Routing](docs/routing.md)** - Advanced routing with all HTTP methods
- **[Handlers](docs/handlers.md)** - Request handlers and patterns
- **[Middleware](docs/middleware.md)** - Built-in and custom middleware
- **[Validation](docs/validator.md)** - Struct validation system
- **[File Handling](docs/file-handling.md)** - File uploads and multipart forms
- **[Static Files](docs/static-files.md)** - Static file serving
- **[WebSockets](docs/websockets.md)** - Real-time communication
- **[HTTP/2](docs/http2.md)** - HTTP/2 configuration and features
- **[Examples](docs/examples.md)** - Complete application examples

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup
```
# Clone repository
git clone https://github.com/AarambhDevHub/blaze.git
cd blaze

# Install dependencies
go mod download

# Run tests
go test ./...

# Run benchmarks
go test -bench=. ./...

# Start development server
go run examples/basic/main.go
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## ☕ Support the Project

If you find Blaze helpful, consider supporting its development:

[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-ffdd00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://buymeacoffee.com/aarambhdevhub)

## 👨‍💻 Author & Community

**AarambhDevHub** - Building the future of Go web development

- 🌐 **GitHub**: [@AarambhDevHub](https://github.com/AarambhDevHub)
- 📺 **YouTube**: [AarambhDevHub](https://youtube.com/@aarambhdevhub)
- 💬 **Discord**: [Join our community](https://discord.gg/HDth6PfCnp)

## 🌟 Show Your Support

If Blaze has helped you build amazing applications:

- ⭐ **Star this repository**
- 🐦 **Share on social media**
- 📝 **Write about your experience**
- 🤝 **Contribute to the project**

---

**Built with ❤️ by [Aarambh Dev Hub](https://youtube.com/@aarambhdevhub)**

*Blaze - Where performance meets elegance in Go web development.*
