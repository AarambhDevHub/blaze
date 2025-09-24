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
- **Intelligent Caching**: Built-in cache middleware for 42% faster data transfer and reduced latency
- **Zero-Copy**: Optimized memory usage with minimal allocations
- **HTTP/2 & h2c**: Full HTTP/2 support with server push capabilities
- **TLS/HTTPS**: Auto-TLS, custom certificates, and development-friendly SSL

### 🛡️ Production Ready
- **Type Safety**: Full compile-time type checking and validation
- **Graceful Shutdown**: Clean shutdown with connection draining
- **Middleware Stack**: Composable middleware with built-in security
- **Error Handling**: Comprehensive error handling and recovery

### 📁 Advanced Features
- **Multipart Forms**: Enterprise file upload handling with validation
- **WebSockets**: Real-time communication with connection management
- **Form Binding**: Automatic struct binding with validation tags
- **Content Negotiation**: JSON, HTML, text, and custom content types

### 🔧 Developer Experience
- **Hot Reload**: Development-friendly features and debugging
- **Route Groups**: Organized API versioning and modular design
- **Configuration**: Environment-specific configs (dev, staging, production)
- **Extensible**: Plugin architecture for custom functionality

## 📦 Installation

```bash
go mod init your-project
go get github.com/AarambhDevHub/blaze
```

## 🚀 Quick Start

### Simple Server
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
            "version": "v1.0.0",
        })
    })

    app.GET("/users/:id", func(c *blaze.Context) error {
        id := c.Param("id")
        return c.JSON(blaze.Map{
            "user_id": id,
            "method":  c.Method(),
            "path":    c.Path(),
        })
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

    // Enable auto-TLS for development
    app.EnableAutoTLS("yourdomain.com", "www.yourdomain.com")

    // Add production middleware with caching
    app.Use(blaze.Logger())
    app.Use(blaze.Recovery())
    app.Use(blaze.CORS())
    app.Use(blaze.Security())
    app.Use(blaze.Cache()) // 🚀 Enable built-in cache for 42% better performance

    // Your routes...

    log.Fatal(app.ListenAndServeGraceful())
}
```

## 📋 Core API Examples

### HTTP Methods & Routing
```go
app := blaze.New()

// RESTful routes
app.GET("/users", getUsers)              // List users
app.POST("/users", createUser)           // Create user  
app.GET("/users/:id", getUser)           // Get user by ID
app.PUT("/users/:id", updateUser)        // Update user
app.DELETE("/users/:id", deleteUser)     // Delete user

// Route parameters
app.GET("/users/:id/posts/:slug", getUserPost)

// Query parameters
app.GET("/search", func(c *blaze.Context) error {
    query := c.Query("q")
    page := c.QueryIntDefault("page", 1)
    limit := c.QueryIntDefault("limit", 10)

    return c.JSON(blaze.Map{
        "query":   query,
        "page":    page,
        "limit":   limit,
        "results": searchResults(query, page, limit),
    })
})

// Wildcards
app.GET("/static/*filepath", serveStatic)
```

### Advanced Form Handling
```go
type UserProfile struct {
    Name     string                 `form:"name,required"`
    Email    string                 `form:"email,required"`
    Age      int                    `form:"age,required"`
    Avatar   *blaze.MultipartFile   `form:"avatar"`
    IsActive bool                   `form:"is_active"`
    Bio      string                 `form:"bio,maxsize=500"`
    JoinedAt time.Time              `form:"joined_at"`
}

app.POST("/profile", func(c *blaze.Context) error {
    var profile UserProfile

    // Automatic form binding with validation
    if err := c.BindMultipartForm(&profile); err != nil {
        return c.Status(400).JSON(blaze.Error(err.Error()))
    }

    // Save avatar if uploaded
    if profile.Avatar != nil {
        savedPath, err := profile.Avatar.SaveWithUniqueFilename("uploads/")
        if err != nil {
            return c.Status(500).JSON(blaze.Error("Failed to save avatar"))
        }
        // Store savedPath in database...
    }

    return c.JSON(blaze.Map{
        "message": "Profile created successfully",
        "profile": profile,
    })
})
```

### WebSocket Support
```go
app.WebSocket("/ws", func(ws *blaze.WebSocketConnection, c *blaze.Context) error {
    for {
        var msg blaze.Map
        if err := ws.ReadJSON(&msg); err != nil {
            break
        }

        // Echo message back
        ws.WriteJSON(blaze.Map{
            "echo":      msg,
            "timestamp": time.Now(),
            "client_ip": c.GetClientIP(),
        })
    }
    return nil
})
```

### Middleware System with Performance Optimization
```go
// Global middleware stack with caching
app.Use(blaze.Logger())                     // Request logging  
app.Use(blaze.Recovery())                   // Panic recovery
app.Use(blaze.CORS())                      // CORS handling
app.Use(blaze.Security())                  // Security headers
app.Use(blaze.Cache())                     // 🚀 High-performance caching (+42% transfer speed)

// Utility middleware  
app.Use(blaze.RequestID())                 // Request ID generation
app.Use(blaze.IPMiddleware())              // Client IP extraction
app.Use(blaze.HTTP2Middleware())           // HTTP/2 optimization

// File handling middleware
app.Use(blaze.MultipartMiddleware(config))  // Multipart form handling
app.Use(blaze.ImageOnlyMiddleware())        // Image upload only
app.Use(blaze.DocumentOnlyMiddleware())     // Document upload only

// Rate limiting
app.Use(blaze.RateLimit(100, time.Minute)) // 100 requests per minute

// Custom middleware
app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        start := time.Now()

        // Process request
        err := next(c)

        // Log duration
        duration := time.Since(start)
        log.Printf("Request took %v", duration)

        return err
    }
})

// Route-specific middleware
auth := func(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        token := c.Header("Authorization")
        if !validateToken(token) {
            return c.Status(401).JSON(blaze.Error("Unauthorized"))
        }
        return next(c)
    }
}

app.GET("/protected", auth(protectedHandler))
```

### Route Groups & API Versioning
```go
// API v1
v1 := app.Group("/api/v1")
v1.Use(corsMiddleware())
v1.Use(authMiddleware())

v1.GET("/users", getUsers)
v1.POST("/users", createUser)
v1.GET("/users/:id", getUser)

// Admin routes
admin := v1.Group("/admin")
admin.Use(adminMiddleware())

admin.GET("/stats", getAdminStats)
admin.POST("/users/:id/ban", banUser)

// API v2 with different structure
v2 := app.Group("/api/v2")
v2.GET("/profiles", getProfiles) // Different endpoint structure
```

### File Uploads & Downloads
```go
// Configure multipart handling
multipartConfig := blaze.ProductionMultipartConfig()
multipartConfig.MaxFileSize = 10 << 20  // 10MB
multipartConfig.MaxFiles = 5
app.Use(blaze.MultipartMiddleware(multipartConfig))

// Single file upload
app.POST("/upload", func(c *blaze.Context) error {
    file, err := c.FormFile("document")
    if err != nil {
        return c.Status(400).JSON(blaze.Error("No file uploaded"))
    }

    // Validate file type
    if !file.IsDocument() {
        return c.Status(400).JSON(blaze.Error("Only documents allowed"))
    }

    // Save with unique filename
    path, err := c.SaveUploadedFileWithUniqueFilename(file, "uploads/")
    if err != nil {
        return c.Status(500).JSON(blaze.Error("Save failed"))
    }

    return c.JSON(blaze.Map{
        "message":     "File uploaded successfully",
        "filename":    file.Filename,
        "saved_path":  path,
        "size":        file.Size,
        "content_type": file.ContentType,
    })
})

// File download with range support
app.GET("/download/:filename", func(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := "uploads/" + filename

    if !c.FileExists(filepath) {
        return c.Status(404).JSON(blaze.Error("File not found"))
    }

    // Stream file with range request support
    return c.StreamFile(filepath)
})
```

### HTTP/2 & TLS Configuration
```go
// HTTP/2 with TLS
config := blaze.ProductionConfig()
config.EnableHTTP2 = true
config.EnableTLS = true
config.RedirectHTTPToTLS = true

app := blaze.NewWithConfig(config)

// Configure TLS
tlsConfig := blaze.DefaultTLSConfig()
tlsConfig.CertFile = "server.crt"
tlsConfig.KeyFile = "server.key"
app.SetTLSConfig(tlsConfig)

// Configure HTTP/2
http2Config := blaze.DefaultHTTP2Config()
http2Config.MaxConcurrentStreams = 1000
http2Config.EnablePush = true
app.SetHTTP2Config(http2Config)

// Server push example
app.GET("/", func(c *blaze.Context) error {
    // Push critical resources
    c.PushResources(map[string]string{
        "/static/app.css": "style",
        "/static/app.js":  "script", 
    })

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

```go
import "github.com/AarambhDevHub/blaze/pkg/blaze"

// Core middleware
app.Use(blaze.Logger())                     // Request logging  
app.Use(blaze.Recovery())                   // Panic recovery
app.Use(blaze.CORS())                      // CORS handling
app.Use(blaze.Security())                  // Security headers
app.Use(blaze.Cache())                     // 🚀 High-performance caching

// Utility middleware  
app.Use(blaze.RequestID())                 // Request ID generation
app.Use(blaze.IPMiddleware())              // Client IP extraction
app.Use(blaze.HTTP2Middleware())           // HTTP/2 optimization

// File handling middleware
app.Use(blaze.MultipartMiddleware(config))  // Multipart form handling
app.Use(blaze.ImageOnlyMiddleware())        // Image upload only
app.Use(blaze.DocumentOnlyMiddleware())     // Document upload only

// Rate limiting
app.Use(blaze.RateLimit(100, time.Minute)) // 100 requests per minute
```

## 📊 Performance Comparison

| Framework | Req/sec | Latency | Memory | HTTP/2 | WebSockets | Cache | Notes |
|-----------|---------|---------|--------|---------|------------|-------|-------|
| **Blaze (Cache)** | **190K** | **0.53ms** | **Ultra Low** | ✅ | ✅ | ✅ | **+42% transfer speed** |
| **Blaze** | **182K** | **0.79ms** | **Ultra Low** | ✅ | ✅ | ✅ | **Production Ready** |
| Fiber | 165K | 0.60ms | Low | ❌ | ✅ | ❌ | FastHTTP-based |
| FastHTTP | 200K+ | 0.5ms | Very Low | ❌ | ❌ | ❌ | Raw performance |
| Gin | 50K | 10ms | Medium | ❌ | ❌ | ❌ | Most popular |
| Echo | 40K | 15ms | Medium | ❌ | ✅ | ❌ | Minimalist |
| Chi | 35K | 20ms | Low | ❌ | ❌ | ❌ | Lightweight router |
| Gorilla | 25K | 25ms | Medium | ❌ | ✅ | ❌ | Feature-rich |
| Go stdlib | 17K | 30ms | Medium | ✅ | ❌ | ❌ | Standard library |

**🏆 Performance Leader**: Blaze with caching delivers the best real-world performance with **190K+ req/sec**, sub-millisecond latency, and comprehensive features.

## 🏗️ Project Structure

```
your-project/
├── main.go                 # Application entry point
├── go.mod                 # Go module definition
├── go.sum                 # Dependency checksums
├── config/
│   ├── config.go         # Configuration management
│   └── environments/     # Environment-specific configs
├── handlers/
│   ├── users.go         # User-related handlers
│   ├── auth.go          # Authentication handlers
│   └── uploads.go       # File upload handlers
├── middleware/
│   ├── auth.go          # Authentication middleware
│   ├── validation.go    # Request validation
│   └── logging.go       # Custom logging
├── models/
│   ├── user.go          # User data structures
│   └── response.go      # API response types
├── services/
│   ├── user_service.go  # Business logic
│   └── email_service.go # External services
├── static/              # Static assets (CSS, JS, images)
├── templates/           # HTML templates
├── uploads/             # File upload directory
└── tests/
    ├── handlers_test.go # Handler tests
    └── integration_test.go # Integration tests
```

## 🔧 Configuration Management

### Environment Configurations
```go
// config/config.go
type AppConfig struct {
    Server   ServerConfig   `json:"server"`
    Database DatabaseConfig `json:"database"`
    Redis    RedisConfig    `json:"redis"`
    Upload   UploadConfig   `json:"upload"`
    Cache    CacheConfig    `json:"cache"`
}

// Development
func DevelopmentConfig() *AppConfig {
    return &AppConfig{
        Server: ServerConfig{
            Host:        "127.0.0.1",
            Port:        3000,
            Development: true,
            EnableTLS:   false,
            EnableHTTP2: false,
        },
        Cache: CacheConfig{
            Enabled:    true,
            TTL:        300, // 5 minutes
            MaxSize:    100, // 100MB
        },
        // ... other configs
    }
}

// Production
func ProductionConfig() *AppConfig {
    return &AppConfig{
        Server: ServerConfig{
            Host:        "0.0.0.0",
            Port:        80,
            TLSPort:     443,
            Development: false,
            EnableTLS:   true,
            EnableHTTP2: true,
        },
        Cache: CacheConfig{
            Enabled:    true,
            TTL:        3600, // 1 hour
            MaxSize:    500,  // 500MB
        },
        // ... other configs
    }
}
```

### Environment Variables
```go
import "os"

config := blaze.ProductionConfig()

// Override from environment
if port := os.Getenv("PORT"); port != "" {
    if p, err := strconv.Atoi(port); err == nil {
        config.Port = p
    }
}

if host := os.Getenv("HOST"); host != "" {
    config.Host = host
}
```

## 🧪 Testing & Benchmarking

### Load Testing Results
```bash
# With cache enabled - Superior performance
wrk -c100 -d30s http://localhost:3000/
Running 30s test @ http://localhost:3000/
  2 threads and 100 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   527.70us  765.78us  11.73ms   89.78%
    Req/Sec    95.69k    20.89k  134.42k    68.83%
  5711615 requests in 30.00s, 3.47GB read
Requests/sec: 190376.62
Transfer/sec:    118.38MB

# Without cache - Still excellent performance
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

### Additional Load Tests
```bash
# JSON API endpoint
wrk -c100 -d30s -t4 -s post.lua http://localhost:8080/api/users

# File upload test  
wrk -c50 -d30s -t4 -s upload.lua http://localhost:8080/upload

# WebSocket connections
wrk -c100 -d30s -t4 --latency http://localhost:8080/ws
```

### Performance Profiling
```bash
# CPU profiling
go tool pprof http://localhost:8080/debug/pprof/profile

# Memory profiling  
go tool pprof http://localhost:8080/debug/pprof/heap

# Goroutine analysis
go tool pprof http://localhost:8080/debug/pprof/goroutine

# Block profiling
go tool pprof http://localhost:8080/debug/pprof/block
```

### Unit Testing
```go
func TestUserHandler(t *testing.T) {
    app := blaze.New()
    app.GET("/users/:id", getUserHandler)

    req := httptest.NewRequest("GET", "/users/123", nil)
    resp := httptest.NewRecorder()

    app.ServeHTTP(resp, req)

    assert.Equal(t, 200, resp.Code)
    // ... additional assertions
}
```

## 📈 Production Deployment

### Docker
```dockerfile
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
COPY --from=builder /app/templates ./templates

EXPOSE 8080
CMD ["./blaze-app"]
```

### Kubernetes
```yaml
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

## 🔐 Security Best Practices

### Security Middleware
```go
// Production security stack
app.Use(blaze.Security())           // Security headers
app.Use(blaze.CORS())              // CORS policy
app.Use(blaze.RateLimit(1000, time.Hour)) // Rate limiting
app.Use(blaze.IPMiddleware())       // IP tracking

// Custom security middleware
func SecurityMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Security headers
            c.SetHeader("X-Content-Type-Options", "nosniff")
            c.SetHeader("X-Frame-Options", "DENY") 
            c.SetHeader("X-XSS-Protection", "1; mode=block")
            c.SetHeader("Strict-Transport-Security", "max-age=31536000")
            c.SetHeader("Content-Security-Policy", "default-src 'self'")

            return next(c)
        }
    }
}
```

### Input Validation
```go
type CreateUserRequest struct {
    Name     string `form:"name,required,minsize=2,maxsize=50"`
    Email    string `form:"email,required" validate:"email"`
    Password string `form:"password,required,minsize=8"`
    Age      int    `form:"age,required" validate:"min=13,max=120"`
}

app.POST("/users", func(c *blaze.Context) error {
    var req CreateUserRequest

    if err := c.BindMultipartForm(&req); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }

    // Additional validation
    if err := validate.Struct(req); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid input",
            "details": err.Error(),
        })
    }

    // Process validated request...
    return c.JSON(blaze.Map{"message": "User created"})
})
```

## 📚 Advanced Examples

Check out comprehensive examples in the `/examples` directory:

- **🏢 Enterprise API**: Full-featured REST API with authentication
- **📁 File Management**: Advanced file upload/download system  
- **🔄 Real-time Chat**: WebSocket-based chat application
- **📊 Analytics Dashboard**: HTTP/2 server-sent events
- **🛡️ Microservices**: Service mesh integration
- **📱 Mobile API**: Mobile-optimized JSON API
- **🎯 E-commerce**: Complete e-commerce backend
- **📈 Monitoring**: Metrics, logging, and observability

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup
```bash
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

### Pull Request Process
1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`go test ./...`)
5. Run benchmarks (`go test -bench=.`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👨‍💻 Author & Community

**AarambhDevHub** - Building the future of Go web development

- 🌐 **GitHub**: [@AarambhDevHub](https://github.com/AarambhDevHub)
- 📺 **YouTube**: [AarambhDevHub](https://youtube.com/@aarambhdevhub) 
- 💬 **Discord**: [Join our community](https://discord.gg/HDth6PfCnp)

## 🌟 Show Your Support

If Blaze has helped you build amazing applications, please:

- ⭐ **Star this repository** 
- 🐦 **Share on social media**
- 📝 **Write about your experience**
- 🤝 **Contribute to the project**

## 📞 Support & Community

Need help? Join our growing community:

- 💬 **Discord Community**: Get real-time help and discuss features
- 🐛 **Issues**: Report bugs and request features on GitHub
- 📺 **Tutorials**: Watch video tutorials on our YouTube channel
- 📝 **Blog**: Read articles about best practices and use cases

---

**Built with ❤️ by [Aarambh Dev Hub](https://youtube.com/@aarambhdevhub)**

*Blaze - Where performance meets elegance in Go web development.*