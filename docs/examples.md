# Examples

Comprehensive examples demonstrating the Blaze web framework's features and capabilities.

## Table of Contents

- [Basic Server Setup](#basic-server-setup)
- [Routing Examples](#routing-examples)
- [Middleware Examples](#middleware-examples)
- [Context and Request/Response](#context-and-requestresponse)
- [File Handling](#file-handling)
- [WebSocket Examples](#websocket-examples)
- [TLS and Security](#tls-and-security)
- [HTTP/2 Support](#http2-support)
- [Database Integration](#database-integration)
- [Caching Strategies](#caching-strategies)
- [Testing Examples](#testing-examples)
- [Production Deployment](#production-deployment)

## Basic Server Setup

### Simple HTTP Server

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
            "message": "Hello, Blaze!",
            "status":  "success",
        })
    })
    
    log.Fatal(app.ListenAndServe())
}
```

### Server with Custom Configuration

```go
package main

import (
    "time"
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
    // Custom configuration
    config := &blaze.Config{
        Host:               "0.0.0.0",
        Port:               3000,
        ReadTimeout:        15 * time.Second,
        WriteTimeout:       15 * time.Second,
        MaxRequestBodySize: 10 * 1024 * 1024, // 10MB
        Concurrency:        1000,
        Development:        true,
    }
    
    app := blaze.NewWithConfig(config)
    
    app.GET("/config", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "host": config.Host,
            "port": config.Port,
            "dev":  config.Development,
        })
    })
    
    app.ListenAndServe()
}
```

### Graceful Shutdown

```go
package main

import (
    "context"
    "log"
    "syscall"
    "time"
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
    app := blaze.New()
    
    // Register graceful cleanup tasks
    app.RegisterGracefulTask(func(ctx context.Context) error {
        log.Println("Cleaning up database connections...")
        // Your cleanup logic here
        return nil
    })
    
    app.GET("/", func(c *blaze.Context) error {
        // Check if server is shutting down
        if c.IsShuttingDown() {
            return c.Status(503).JSON(blaze.Map{
                "error": "Server shutting down",
            })
        }
        
        return c.JSON(blaze.Map{"status": "ok"})
    })
    
    // Start with graceful shutdown handling
    log.Fatal(app.ListenAndServeGraceful(syscall.SIGINT, syscall.SIGTERM))
}
```

## Routing Examples

### Basic Routes

```go
func setupRoutes(app *blaze.App) {
    // GET route
    app.GET("/users", getUsers)
    
    // POST route
    app.POST("/users", createUser)
    
    // PUT route
    app.PUT("/users/:id", updateUser)
    
    // DELETE route
    app.DELETE("/users/:id", deleteUser)
    
    // PATCH route
    app.PATCH("/users/:id", patchUser)
    
    // HEAD route
    app.HEAD("/users/:id", checkUser)
    
    // OPTIONS route
    app.OPTIONS("/users", optionsUsers)
    
    // Match multiple methods
    app.Match([]string{"GET", "POST"}, "/multi", multiHandler)
    
    // ANY matches all methods
    app.ANY("/catch-all", catchAllHandler)
}

func getUsers(c *blaze.Context) error {
    return c.JSON(blaze.Map{
        "users": []blaze.Map{
            {"id": 1, "name": "Alice"},
            {"id": 2, "name": "Bob"},
        },
    })
}

func createUser(c *blaze.Context) error {
    var user struct {
        Name  string `json:"name" validate:"required"`
        Email string `json:"email" validate:"required,email"`
    }
    
    if err := c.BindJSONAndValidate(&user); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }
    
    return c.Status(201).JSON(blaze.Map{
        "message": "User created",
        "user":    user,
    })
}
```

### Route Parameters with Constraints

```go
func routeConstraints(app *blaze.App) {
    // Integer constraint
    app.GET("/users/:id", getUserByID,
        blaze.WithIntConstraint("id"))
    
    // UUID constraint
    app.GET("/posts/:uuid", getPostByUUID,
        blaze.WithUUIDConstraint("uuid"))
    
    // Regex constraint
    app.GET("/files/:filename", getFile,
        blaze.WithRegexConstraint("filename", `^[a-zA-Z0-9_\-]+\.(jpg|png|pdf)$`))
    
    // Custom constraint
    app.GET("/products/:sku", getProduct,
        blaze.WithConstraint("sku", blaze.RouteConstraint{
            Name:    "sku",
            Pattern: regexp.MustCompile(`^[A-Z]{3}-\d{5}$`),
            Type:    blaze.RegexConstraint,
        }))
}

func getUserByID(c *blaze.Context) error {
    id, _ := c.ParamInt("id") // Already validated by constraint
    return c.JSON(blaze.Map{"user_id": id})
}
```

### Query Parameters

```go
func queryParams(app *blaze.App) {
    app.GET("/search", func(c *blaze.Context) error {
        query := c.Query("q")
        page := c.QueryIntDefault("page", 1)
        limit := c.QueryIntDefault("limit", 10)
        sortBy := c.QueryDefault("sort", "created_at")
        
        // Multiple values for same parameter
        args := c.QueryArgs()
        var tags []string
        args.VisitAll(func(key, value []byte) {
            if string(key) == "tag" {
                tags = append(tags, string(value))
            }
        })
        
        return c.JSON(blaze.Map{
            "query":   query,
            "page":    page,
            "limit":   limit,
            "sort_by": sortBy,
            "tags":    tags,
        })
    })
}
```

### Route Groups

```go
func routeGroups(app *blaze.App) {
    // API v1 group
    v1 := app.Group("/api/v1")
    v1.Use(blaze.LoggerMiddleware())
    v1.Use(blaze.BodyLimitMB(5))
    v1.Use(blaze.CacheAPI(2 * time.Minute))
    
    v1.GET("/users", listUsers)
    v1.POST("/users", createUser)
    
    // Nested admin group
    admin := v1.Group("/admin")
    admin.Use(RequireAdminMiddleware())
    
    admin.GET("/users", adminListUsers)
    admin.DELETE("/users/:id", adminDeleteUser)
    
    // Public group with different middleware
    public := app.Group("/public")
    public.Use(blaze.CacheStatic())
    public.Use(blaze.Compress())
    
    public.GET("/files/:name", servePublicFile)
}

func RequireAdminMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            role := c.Header("X-User-Role")
            if role != "admin" {
                return c.Status(403).JSON(blaze.Map{
                    "error": "Admin access required",
                })
            }
            return next(c)
        }
    }
}
```

### Named Routes

```go
func namedRoutes(app *blaze.App) {
    // Named routes for URL generation
    app.GET("/users/:id", getUserHandler,
        blaze.WithName("user.show"))
    
    app.GET("/users/:id/edit", editUserHandler,
        blaze.WithName("user.edit"))
    
    app.POST("/users", createUserHandler,
        blaze.WithName("user.create"))
}
```

## Middleware Examples

### Comprehensive Middleware Stack

```go
func setupMiddleware(app *blaze.App) {
    // Recovery middleware (should be first)
    app.Use(blaze.Recovery())
    
    // Logging
    logConfig := blaze.DefaultLoggerMiddlewareConfig()
    logConfig.SlowRequestThreshold = 2 * time.Second
    logConfig.SkipPaths = []string{"/health", "/metrics"}
    app.Use(blaze.LoggerMiddlewareWithConfig(logConfig))
    
    // Request ID
    app.Use(blaze.RequestIDMiddleware())
    
    // CORS
    corsOpts := blaze.CORSOptions{
        AllowOrigins:     []string{"https://example.com"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
        ExposeHeaders:    []string{"X-Request-ID"},
        AllowCredentials: true,
        MaxAge:           3600,
    }
    app.Use(blaze.CORS(corsOpts))
    
    // CSRF Protection
    csrfOpts := blaze.DefaultCSRFOptions()
    csrfOpts.Secret = []byte("your-32-byte-secret-key-here!!!")
    csrfOpts.CookieSecure = true
    app.Use(blaze.CSRF(csrfOpts))
    
    // Body Limit
    app.Use(blaze.BodyLimitMB(10))
    
    // Compression
    app.Use(blaze.CompressWithLevel(blaze.CompressionLevelBest))
    
    // Rate Limiting
    rateLimitOpts := blaze.RateLimitOptions{
        MaxRequests: 100,
        Window:      time.Minute,
        KeyGenerator: func(c *blaze.Context) string {
            return c.IP()
        },
    }
    app.Use(blaze.RateLimitMiddleware(rateLimitOpts))
    
    // Cache
    cacheOpts := blaze.DefaultCacheOptions()
    cacheOpts.DefaultTTL = 5 * time.Minute
    app.Use(blaze.Cache(cacheOpts))
    
    // Shutdown awareness
    app.Use(blaze.ShutdownAware())
}
```

### Custom Authentication Middleware

```go
func AuthMiddleware(secret string) blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            token := c.Header("Authorization")
            
            if token == "" {
                return c.Status(401).JSON(blaze.Map{
                    "error": "Authorization header required",
                })
            }
            
            // Remove "Bearer " prefix
            if len(token) > 7 && token[:7] == "Bearer " {
                token = token[7:]
            }
            
            // Validate JWT token
            claims, err := validateJWT(token, secret)
            if err != nil {
                return c.Status(401).JSON(blaze.Map{
                    "error": "Invalid token",
                })
            }
            
            // Store user info in context
            c.SetLocals("user_id", claims.UserID)
            c.SetLocals("email", claims.Email)
            c.SetLocals("role", claims.Role)
            
            return next(c)
        }
    }
}

func validateJWT(token, secret string) (*Claims, error) {
    // Implement JWT validation
    return &Claims{
        UserID: 123,
        Email:  "user@example.com",
        Role:   "user",
    }, nil
}
```

### Request/Response Logging Middleware

```go
func DetailedLoggingMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            start := time.Now()
            requestID := c.Header("X-Request-ID")
            
            // Log request
            c.LogInfo("Incoming request",
                "request_id", requestID,
                "method", c.Method(),
                "path", c.Path(),
                "ip", c.IP(),
                "user_agent", c.UserAgent())
            
            // Execute handler
            err := next(c)
            
            // Log response
            duration := time.Since(start)
            status := c.Response().StatusCode()
            
            logLevel := "info"
            if status >= 500 {
                logLevel = "error"
            } else if status >= 400 {
                logLevel = "warn"
            }
            
            switch logLevel {
            case "error":
                c.LogError("Request failed",
                    "request_id", requestID,
                    "status", status,
                    "duration", duration,
                    "error", err)
            case "warn":
                c.LogWarn("Request completed with client error",
                    "request_id", requestID,
                    "status", status,
                    "duration", duration)
            default:
                c.LogInfo("Request completed",
                    "request_id", requestID,
                    "status", status,
                    "duration", duration)
            }
            
            return err
        }
    }
}
```

### Security Headers Middleware

```go
func SecurityHeadersMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Set security headers
            c.SetHeader("X-Content-Type-Options", "nosniff")
            c.SetHeader("X-Frame-Options", "DENY")
            c.SetHeader("X-XSS-Protection", "1; mode=block")
            c.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")
            c.SetHeader("Content-Security-Policy", "default-src 'self'")
            c.SetHeader("Permissions-Policy", "geolocation=(), microphone=()")
            
            if c.Request().URI().Scheme() == "https" {
                c.SetHeader("Strict-Transport-Security", 
                    "max-age=31536000; includeSubDomains; preload")
            }
            
            return next(c)
        }
    }
}
```

## Context and Request/Response

### Comprehensive Request Binding

```go
type UserRegistration struct {
    Name     string                `json:"name" form:"name,required,minsize:2,maxsize:100"`
    Email    string                `json:"email" form:"email,required"`
    Password string                `json:"password" form:"password,required,minsize:8"`
    Age      *int                  `json:"age" form:"age"`
    Avatar   *blaze.MultipartFile  `form:"avatar"`
    Tags     []string              `json:"tags" form:"tags"`
    BirthDate *time.Time           `json:"birth_date" form:"birth_date"`
}

func handleRegistration(app *blaze.App) {
    // JSON binding
    app.POST("/register/json", func(c *blaze.Context) error {
        var reg UserRegistration
        
        if err := c.BindJSONAndValidate(&reg); err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "Validation failed",
                "details": err.Error(),
            })
        }
        
        return c.Status(201).JSON(reg)
    })
    
    // Multipart form binding
    app.POST("/register/form", func(c *blaze.Context) error {
        var reg UserRegistration
        
        if err := c.BindMultipartFormAndValidate(&reg); err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "Validation failed",
                "details": err.Error(),
            })
        }
        
        // Save avatar if provided
        if reg.Avatar != nil {
            avatarPath, err := c.SaveUploadedFileWithUniqueFilename(
                reg.Avatar, "./uploads/avatars")
            if err != nil {
                return c.Status(500).JSON(blaze.Map{
                    "error": "Failed to save avatar",
                })
            }
            reg.Avatar.TempFilePath = avatarPath
        }
        
        return c.Status(201).JSON(reg)
    })
}
```

### Response Types

```go
func responseExamples(app *blaze.App) {
    // JSON responses with helpers
    app.GET("/success", func(c *blaze.Context) error {
        return c.JSON(blaze.OK(blaze.Map{"data": "value"}))
    })
    
    app.POST("/create", func(c *blaze.Context) error {
        return c.JSON(blaze.Created(blaze.Map{"id": 123}))
    })
    
    app.GET("/error", func(c *blaze.Context) error {
        return c.JSON(blaze.Error("Something went wrong"))
    })
    
    // Paginated response
    app.GET("/users", func(c *blaze.Context) error {
        users := getUsersFromDB()
        total := getTotalUsers()
        page := c.QueryIntDefault("page", 1)
        perPage := c.QueryIntDefault("per_page", 10)
        
        return c.JSON(blaze.Paginate(users, total, page, perPage))
    })
    
    // Text response
    app.GET("/health", func(c *blaze.Context) error {
        return c.Text("OK")
    })
    
    // HTML response
    app.GET("/welcome", func(c *blaze.Context) error {
        html := "<h1>Welcome!</h1>"
        return c.HTML(html)
    })
    
    // Custom headers
    app.GET("/api/data", func(c *blaze.Context) error {
        return c.
            Status(200).
            SetHeader("X-API-Version", "v1.0").
            SetHeader("X-Rate-Limit", "1000").
            JSON(blaze.Map{"data": "value"})
    })
}
```

## File Handling

### Complete File Upload Example

```go
func fileUploadExample(app *blaze.App) {
    // Single file with validation
    app.POST("/upload/single", func(c *blaze.Context) error {
        file, err := c.FormFile("file")
        if err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "No file provided",
            })
        }
        
        // Validate file
        if err := validateUploadedFile(file); err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": err.Error(),
            })
        }
        
        // Save with unique filename
        savedPath, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
        if err != nil {
            return c.Status(500).JSON(blaze.Map{
                "error": "Failed to save file",
            })
        }
        
        return c.JSON(blaze.Map{
            "message":       "File uploaded successfully",
            "filename":      file.Filename,
            "saved_path":    savedPath,
            "size":          file.Size,
            "content_type":  file.ContentType,
        })
    })
    
    // Multiple files
    app.POST("/upload/multiple", func(c *blaze.Context) error {
        files, err := c.FormFiles("files")
        if err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "No files provided",
            })
        }
        
        var results []blaze.Map
        for _, file := range files {
            if err := validateUploadedFile(file); err != nil {
                continue
            }
            
            savedPath, err := c.SaveUploadedFileWithUniqueFilename(
                file, "./uploads")
            if err != nil {
                continue
            }
            
            results = append(results, blaze.Map{
                "original_name": file.Filename,
                "saved_path":    savedPath,
                "size":          file.Size,
            })
        }
        
        return c.JSON(blaze.Map{
            "message": "Files uploaded",
            "files":   results,
            "count":   len(results),
        })
    })
}

func validateUploadedFile(file *blaze.MultipartFile) error {
    // Size validation
    maxSize := int64(10 * 1024 * 1024) // 10MB
    if file.Size > maxSize {
        return fmt.Errorf("file too large (max 10MB)")
    }
    
    // Type validation
    allowedTypes := []string{"image/jpeg", "image/png", "application/pdf"}
    allowed := false
    for _, t := range allowedTypes {
        if file.ContentType == t {
            allowed = true
            break
        }
    }
    
    if !allowed {
        return fmt.Errorf("file type not allowed")
    }
    
    return nil
}
```

### File Download and Streaming

```go
func fileDownloadExample(app *blaze.App) {
    // Serve file
    app.GET("/files/:filename", func(c *blaze.Context) error {
        filename := c.Param("filename")
        filepath := "./uploads/" + filename
        
        if !c.FileExists(filepath) {
            return c.Status(404).JSON(blaze.Map{
                "error": "File not found",
            })
        }
        
        return c.ServeFile(filepath)
    })
    
    // Force download
    app.GET("/download/:filename", func(c *blaze.Context) error {
        filename := c.Param("filename")
        filepath := "./uploads/" + filename
        
        if !c.FileExists(filepath) {
            return c.Status(404).JSON(blaze.Map{
                "error": "File not found",
            })
        }
        
        return c.ServeFileDownload(filepath, filename)
    })
    
    // Inline display (for images, PDFs)
    app.GET("/view/:filename", func(c *blaze.Context) error {
        filename := c.Param("filename")
        filepath := "./uploads/" + filename
        
        return c.ServeFileInline(filepath)
    })
    
    // Stream large files with range support
    app.GET("/stream/:filename", func(c *blaze.Context) error {
        filename := c.Param("filename")
        filepath := "./videos/" + filename
        
        return c.StreamFile(filepath)
    })
    
    // File info
    app.GET("/info/:filename", func(c *blaze.Context) error {
        filename := c.Param("filename")
        filepath := "./uploads/" + filename
        
        info, err := c.GetFileInfo(filepath)
        if err != nil {
            return c.Status(404).JSON(blaze.Map{
                "error": "File not found",
            })
        }
        
        return c.JSON(blaze.Map{
            "name":     info.Name(),
            "size":     info.Size(),
            "modified": info.ModTime(),
            "is_dir":   info.IsDir(),
        })
    })
}
```

## WebSocket Examples

### Basic WebSocket Echo Server

```go
func websocketEcho(app *blaze.App) {
    app.WebSocket("/ws/echo", func(ws *blaze.WebSocketConnection) error {
        log.Printf("New WebSocket connection from %s", ws.RemoteAddr())
        
        // Set local data
        ws.SetLocal("connected_at", time.Now())
        
        for {
            messageType, data, err := ws.ReadMessage()
            if err != nil {
                log.Printf("WebSocket read error: %v", err)
                break
            }
            
            log.Printf("Received: %s", data)
            
            // Echo back
            if err := ws.WriteMessage(messageType, data); err != nil {
                log.Printf("WebSocket write error: %v", err)
                break
            }
        }
        
        return nil
    })
}
```

### WebSocket Chat Server with Hub

```go
type ChatHub struct {
    clients    map[*blaze.WebSocketConnection]string
    broadcast  chan Message
    register   chan *blaze.WebSocketConnection
    unregister chan *blaze.WebSocketConnection
    mutex      sync.RWMutex
}

type Message struct {
    Type     string `json:"type"`
    Username string `json:"username"`
    Content  string `json:"content"`
    Time     string `json:"time"`
}

func NewChatHub() *ChatHub {
    return &ChatHub{
        clients:    make(map[*blaze.WebSocketConnection]string),
        broadcast:  make(chan Message),
        register:   make(chan *blaze.WebSocketConnection),
        unregister: make(chan *blaze.WebSocketConnection),
    }
}

func (h *ChatHub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mutex.Lock()
            username := client.GetLocal("username").(string)
            h.clients[client] = username
            h.mutex.Unlock()
            
            // Broadcast join message
            h.broadcast <- Message{
                Type:     "join",
                Username: username,
                Time:     time.Now().Format(time.RFC3339),
            }
            
        case client := <-h.unregister:
            h.mutex.Lock()
            if username, ok := h.clients[client]; ok {
                delete(h.clients, client)
                client.Close()
                
                // Broadcast leave message
                h.broadcast <- Message{
                    Type:     "leave",
                    Username: username,
                    Time:     time.Now().Format(time.RFC3339),
                }
            }
            h.mutex.Unlock()
            
        case message := <-h.broadcast:
            h.mutex.RLock()
            for client := range h.clients {
                if err := client.WriteJSON(message); err != nil {
                    log.Printf("Broadcast error: %v", err)
                }
            }
            h.mutex.RUnlock()
        }
    }
}

func websocketChat(app *blaze.App) {
    hub := NewChatHub()
    go hub.Run()
    
    config := blaze.DefaultWebSocketConfig()
    config.MaxMessageSize = 1024 * 1024 // 1MB
    config.PingInterval = 30 * time.Second
    
    app.WebSocketWithConfig("/ws/chat", func(ws *blaze.WebSocketConnection) error {
        // Get username from query parameter
        username := ws.Context().Query("username")
        if username == "" {
            username = "Anonymous"
        }
        
        ws.SetLocal("username", username)
        hub.register <- ws
        
        defer func() {
            hub.unregister <- ws
        }()
        
        // Ping routine
        go func() {
            ticker := time.NewTicker(30 * time.Second)
            defer ticker.Stop()
            
            for range ticker.C {
                if err := ws.Ping([]byte{}); err != nil {
                    return
                }
            }
        }()
        
        for {
            var msg Message
            if err := ws.ReadJSON(&msg); err != nil {
                break
            }
            
            msg.Username = username
            msg.Time = time.Now().Format(time.RFC3339)
            hub.broadcast <- msg
        }
        
        return nil
    }, config)
}
```

## TLS and Security

### Production HTTPS Server

```go
func productionHTTPS() {
    config := blaze.ProductionConfig()
    config.EnableTLS = true
    config.RedirectHTTPToTLS = true
    
    app := blaze.NewWithConfig(config)
    
    // TLS configuration
    tlsConfig := &blaze.TLSConfig{
        CertFile:   "/etc/ssl/certs/server.crt",
        KeyFile:    "/etc/ssl/private/server.key",
        MinVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        },
        NextProtos:    []string{"h2", "http/1.1"},
        OCSPStapling:  true,
    }
    
    app.SetTLSConfig(tlsConfig)
    
    // Security middleware
    app.Use(SecurityHeadersMiddleware())
    
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "secure":   true,
            "protocol": c.Protocol(),
        })
    })
    
    log.Fatal(app.ListenAndServe())
}
```

### Auto TLS for Development

```go
func developmentAutoTLS() {
    app := blaze.New()
    
    // Enable auto TLS with self-signed certificate
    app.EnableAutoTLS("localhost", "127.0.0.1", "myapp.local")
    
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "message": "Auto TLS enabled",
            "secure":  true,
        })
    })
    
    log.Fatal(app.ListenAndServe())
}
```

## HTTP/2 Support

### HTTP/2 with Server Push

```go
func http2Example() {
    config := blaze.ProductionConfig()
    config.EnableHTTP2 = true
    config.EnableTLS = true
    
    app := blaze.NewWithConfig(config)
    
    // HTTP/2 configuration
    http2Config := &blaze.HTTP2Config{
        Enabled:              true,
        MaxConcurrentStreams: 1000,
        EnablePush:           true,
        IdleTimeout:          300 * time.Second,
    }
    
    app.SetHTTP2Config(http2Config)
    
    // TLS required for HTTP/2
    tlsConfig := &blaze.TLSConfig{
        CertFile:   "server.crt",
        KeyFile:    "server.key",
        NextProtos: []string{"h2", "http/1.1"},
    }
    
    app.SetTLSConfig(tlsConfig)
    
    // Server push example
    app.GET("/", func(c *blaze.Context) error {
        if c.IsHTTP2() {
            // Push resources
            resources := map[string]string{
                "/css/main.css":   "style",
                "/js/app.js":      "script",
                "/img/logo.png":   "image",
            }
            c.PushResources(resources)
            
            log.Printf("Handling request on HTTP/2 stream %d", c.StreamID())
        }
        
        html := `
        <html>
            <head>
                <link rel="stylesheet" href="/css/main.css">
                <script src="/js/app.js"></script>
            </head>
            <body>
                <h1>HTTP/2 Page</h1>
                <img src="/img/logo.png" alt="Logo">
            </body>
        </html>`
        
        return c.HTML(html)
    })
    
    log.Fatal(app.ListenAndServe())
}
```

## Caching Strategies

### Multi-level Caching

```go
func cachingStrategies(app *blaze.App) {
    // Static files - long cache
    staticGroup := app.Group("/static")
    staticGroup.Use(blaze.CacheStatic())
    staticGroup.Use(blaze.Compress())
    
    // API endpoints - short cache
    apiGroup := app.Group("/api")
    apiGroup.Use(blaze.CacheAPI(2 * time.Minute))
    
    // Custom cache configuration
    cacheOpts := blaze.CacheOptions{
        DefaultTTL: 10 * time.Minute,
        MaxSize:    500 * 1024 * 1024, // 500MB
        MaxEntries: 50000,
        Algorithm:  blaze.LRU,
        VaryHeaders: []string{"Accept-Encoding", "Accept-Language"},
        Public:     true,
        EnableCompression: true,
        CompressionLevel:  9,
    }
    
    app.Use(blaze.Cache(cacheOpts))
    
    // Cache status endpoint
    app.GET("/cache/status", blaze.CacheStatus)
    
    // Invalidate cache
    app.POST("/cache/invalidate", func(c *blaze.Context) error {
        pattern := c.Query("pattern")
        count := blaze.InvalidateCache(cacheStore, pattern)
        
        return c.JSON(blaze.Map{
            "invalidated": count,
        })
    })
}
```

## Database Integration

### PostgreSQL with Connection Pool

```go
import (
    "database/sql"
    _ "github.com/lib/pq"
)

func postgresIntegration() {
    // Database connection pool
    connStr := "postgres://user:pass@localhost/dbname?sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal(err)
    }
    
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    app := blaze.New()
    
    // Inject database into context
    app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            c.SetLocals("db", db)
            return next(c)
        }
    })
    
    // CRUD operations
    app.GET("/users/:id", func(c *blaze.Context) error {
        db := c.Locals("db").(*sql.DB)
        id, _ := c.ParamInt("id")
        
        var user User
        err := db.QueryRow(
            "SELECT id, name, email FROM users WHERE id = $1", id,
        ).Scan(&user.ID, &user.Name, &user.Email)
        
        if err == sql.ErrNoRows {
            return c.Status(404).JSON(blaze.Map{
                "error": "User not found",
            })
        }
        
        if err != nil {
            return c.Status(500).JSON(blaze.Map{
                "error": "Database error",
            })
        }
        
        return c.JSON(user)
    })
    
    app.POST("/users", func(c *blaze.Context) error {
        db := c.Locals("db").(*sql.DB)
        
        var user User
        if err := c.BindJSONAndValidate(&user); err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": err.Error(),
            })
        }
        
        err := db.QueryRow(
            "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id",
            user.Name, user.Email,
        ).Scan(&user.ID)
        
        if err != nil {
            return c.Status(500).JSON(blaze.Map{
                "error": "Failed to create user",
            })
        }
        
        return c.Status(201).JSON(user)
    })
    
    log.Fatal(app.ListenAndServe())
}
```

## Testing Examples

### Comprehensive Unit Tests

```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func TestUserAPI(t *testing.T) {
    app := setupTestApp()
    
    tests := []struct {
        name           string
        method         string
        path           string
        body           interface{}
        expectedStatus int
        checkResponse  func(*testing.T, *httptest.ResponseRecorder)
    }{
        {
            name:           "Get Users",
            method:         "GET",
            path:           "/api/users",
            expectedStatus: http.StatusOK,
            checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
                var result map[string]interface{}
                json.Unmarshal(resp.Body.Bytes(), &result)
                users := result["users"].([]interface{})
                if len(users) == 0 {
                    t.Error("Expected users array")
                }
            },
        },
        {
            name:   "Create User",
            method: "POST",
            path:   "/api/users",
            body: map[string]string{
                "name":  "Test User",
                "email": "test@example.com",
            },
            expectedStatus: http.StatusCreated,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var req *http.Request
            
            if tt.body != nil {
                jsonData, _ := json.Marshal(tt.body)
                req = httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer(jsonData))
                req.Header.Set("Content-Type", "application/json")
            } else {
                req = httptest.NewRequest(tt.method, tt.path, nil)
            }
            
            resp := httptest.NewRecorder()
            app.ServeHTTP(resp, req)
            
            if resp.Code != tt.expectedStatus {
                t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.Code)
            }
            
            if tt.checkResponse != nil {
                tt.checkResponse(t, resp)
            }
        })
    }
}

func setupTestApp() *blaze.App {
    app := blaze.New()
    app.Use(blaze.Recovery())
    setupRoutes(app)
    return app
}
```

## Production Deployment

### Complete Production Setup

```go
func productionDeployment() {
    config := blaze.ProductionConfig()
    config.Concurrency = 10000
    
    app := blaze.NewWithConfig(config)
    
    // TLS Configuration
    tlsConfig := &blaze.TLSConfig{
        CertFile:   os.Getenv("TLS_CERT_FILE"),
        KeyFile:    os.Getenv("TLS_KEY_FILE"),
        MinVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        },
        NextProtos: []string{"h2", "http/1.1"},
    }
    app.SetTLSConfig(tlsConfig)
    
    // HTTP/2 Configuration
    http2Config := &blaze.HTTP2Config{
        Enabled:              true,
        MaxConcurrentStreams: 5000,
        EnablePush:           true,
    }
    app.SetHTTP2Config(http2Config)
    
    // Production middleware stack
    app.Use(blaze.Recovery())
    app.Use(DetailedLoggingMiddleware())
    app.Use(blaze.RequestIDMiddleware())
    app.Use(SecurityHeadersMiddleware())
    app.Use(blaze.CORS(blaze.CORSOptions{
        AllowOrigins: []string{os.Getenv("ALLOWED_ORIGIN")},
        AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    }))
    app.Use(blaze.RateLimitMiddleware(blaze.RateLimitOptions{
        MaxRequests: 1000,
        Window:      time.Hour,
    }))
    app.Use(blaze.CompressWithLevel(blaze.CompressionLevelBest))
    app.Use(blaze.Cache(blaze.ProductionCacheOptions()))
    
    // Health checks
    app.GET("/health", func(c *blaze.Context) error {
        return c.JSON(blaze.Health("1.0.0", getUptime()))
    })
    
    app.GET("/ready", func(c *blaze.Context) error {
        if !isReady() {
            return c.Status(503).JSON(blaze.Map{
                "status": "not ready",
            })
        }
        return c.JSON(blaze.Map{"status": "ready"})
    })
    
    // Metrics
    app.GET("/metrics", func(c *blaze.Context) error {
        return c.JSON(getMetrics())
    })
    
    // Setup application routes
    setupRoutes(app)
    
    // Graceful shutdown
    log.Fatal(app.ListenAndServeGraceful(syscall.SIGINT, syscall.SIGTERM))
}
```

This comprehensive examples documentation covers all major features of the Blaze framework with production-ready code examples, best practices, and real-world usage patterns.