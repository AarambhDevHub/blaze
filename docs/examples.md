# Examples

Comprehensive examples demonstrating the Blaze web framework's features and capabilities.

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
        Name  string `json:"name"`
        Email string `json:"email"`
    }
    
    if err := c.BindJSON(&user); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid JSON",
        })
    }
    
    return c.Status(201).JSON(blaze.Map{
        "message": "User created",
        "user":    user,
    })
}

func updateUser(c *blaze.Context) error {
    id := c.Param("id")
    
    var updates map[string]interface{}
    if err := c.BindJSON(&updates); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid JSON",
        })
    }
    
    return c.JSON(blaze.Map{
        "message": "User updated",
        "id":      id,
        "updates": updates,
    })
}
```

### Route Parameters

```go
func routeParams(app *blaze.App) {
    // Single parameter
    app.GET("/users/:id", func(c *blaze.Context) error {
        id := c.Param("id")
        return c.JSON(blaze.Map{"user_id": id})
    })
    
    // Multiple parameters
    app.GET("/users/:id/posts/:postId", func(c *blaze.Context) error {
        userID := c.Param("id")
        postID := c.Param("postId")
        
        return c.JSON(blaze.Map{
            "user_id": userID,
            "post_id": postID,
        })
    })
    
    // Parameter with type conversion
    app.GET("/users/:id/age", func(c *blaze.Context) error {
        id, err := c.ParamInt("id")
        if err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "Invalid user ID",
            })
        }
        
        return c.JSON(blaze.Map{
            "user_id": id,
            "age":     25, // Example
        })
    })
    
    // Parameter with default value
    app.GET("/users/:id/score", func(c *blaze.Context) error {
        id := c.ParamIntDefault("id", 0)
        
        return c.JSON(blaze.Map{
            "user_id": id,
            "score":   100,
        })
    })
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
        
        return c.JSON(blaze.Map{
            "query":   query,
            "page":    page,
            "limit":   limit,
            "sort_by": sortBy,
        })
    })
    
    // Multiple values for same parameter
    app.GET("/tags", func(c *blaze.Context) error {
        args := c.QueryArgs()
        var tags []string
        
        args.VisitAll(func(key, value []byte) {
            if string(key) == "tag" {
                tags = append(tags, string(value))
            }
        })
        
        return c.JSON(blaze.Map{
            "tags": tags,
        })
    })
}
```

### Route Groups

```go
func routeGroups(app *blaze.App) {
    // API v1 group
    v1 := app.Group("/api/v1")
    v1.Use(blaze.Logger())
    v1.Use(blaze.Auth(func(token string) bool {
        return token == "valid-token"
    }))
    
    v1.GET("/users", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{"version": "v1", "users": []string{}})
    })
    
    v1.POST("/users", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{"version": "v1", "created": true})
    })
    
    // Admin group with additional middleware
    admin := v1.Group("/admin")
    admin.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Admin-specific validation
            role := c.Header("X-User-Role")
            if role != "admin" {
                return c.Status(403).JSON(blaze.Map{
                    "error": "Admin access required",
                })
            }
            return next(c)
        }
    })
    
    admin.GET("/users", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{"admin": true, "users": []string{}})
    })
    
    admin.DELETE("/users/:id", func(c *blaze.Context) error {
        id := c.Param("id")
        return c.JSON(blaze.Map{"admin": true, "deleted": id})
    })
}
```

## Middleware Examples

### Built-in Middleware

```go
func builtinMiddleware(app *blaze.App) {
    // Global middleware
    app.Use(blaze.Logger())
    app.Use(blaze.Recovery())
    app.Use(blaze.CORS(&blaze.CORSOptions{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:     []string{"*"},
        AllowCredentials: true,
        MaxAge:           86400,
    }))
    
    // Rate limiting
    app.Use(blaze.RateLimit(&blaze.RateLimitOptions{
        Max:      100,
        Duration: time.Minute,
        KeyGenerator: func(c *blaze.Context) string {
            return c.IP()
        },
    }))
    
    // Request ID
    app.Use(blaze.RequestID())
    
    // Cache middleware for GET requests
    app.Use(blaze.Cache(&blaze.CacheOptions{
        Duration: 5 * time.Minute,
        KeyGenerator: func(c *blaze.Context) string {
            return c.Method() + ":" + c.Path()
        },
    }))
}
```

### Custom Middleware

```go
// Authentication middleware
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
            
            // Validate token (implement your logic)
            if !validateJWT(token, secret) {
                return c.Status(401).JSON(blaze.Map{
                    "error": "Invalid token",
                })
            }
            
            // Store user info in context
            userID := extractUserID(token)
            c.SetLocals("user_id", userID)
            
            return next(c)
        }
    }
}

// Request logging middleware
func RequestLogger() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            start := time.Now()
            
            // Log request
            log.Printf("➡️  %s %s from %s", 
                c.Method(), 
                c.Path(), 
                c.IP())
            
            err := next(c)
            
            // Log response
            duration := time.Since(start)
            status := c.Response().StatusCode()
            
            log.Printf("⬅️  %s %s - %d (%v)", 
                c.Method(), 
                c.Path(), 
                status, 
                duration)
            
            return err
        }
    }
}

// CSRF protection middleware
func CSRFProtection() blaze.MiddlewareFunc {
    return blaze.CSRF(&blaze.CSRFOptions{
        Secret:      []byte("your-32-byte-secret-key-here!!!"),
        TokenLookup: []string{"header:X-CSRF-Token", "form:csrf_token"},
        CookieName:  "_csrf",
        CookieMaxAge: 3600,
    })
}
```

### Conditional Middleware

```go
func conditionalMiddleware(app *blaze.App) {
    // Skip middleware based on path
    skipAuth := func(c *blaze.Context) bool {
        skipPaths := []string{"/health", "/metrics", "/public"}
        for _, path := range skipPaths {
            if strings.HasPrefix(c.Path(), path) {
                return true
            }
        }
        return false
    }
    
    // Conditional authentication
    authMiddleware := func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            if skipAuth(c) {
                return next(c)
            }
            
            token := c.Header("Authorization")
            if token == "" {
                return c.Status(401).JSON(blaze.Map{
                    "error": "Authentication required",
                })
            }
            
            return next(c)
        }
    }
    
    app.Use(authMiddleware)
}
```

## Context and Request/Response Handling

### Request Data Binding

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name" validate:"required"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=0,max=150"`
}

func requestBinding(app *blaze.App) {
    // JSON binding
    app.POST("/users", func(c *blaze.Context) error {
        var user User
        
        if err := c.BindJSON(&user); err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "Invalid JSON format",
                "details": err.Error(),
            })
        }
        
        // Validate struct (using external validation library)
        if err := validate.Struct(&user); err != nil {
            return c.Status(422).JSON(blaze.Map{
                "error": "Validation failed",
                "details": err.Error(),
            })
        }
        
        return c.Status(201).JSON(user)
    })
    
    // Form binding
    app.POST("/contact", func(c *blaze.Context) error {
        var contact struct {
            Name    string `form:"name"`
            Email   string `form:"email"`
            Message string `form:"message"`
        }
        
        if err := c.BindForm(&contact); err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "Invalid form data",
            })
        }
        
        return c.JSON(blaze.Map{
            "message": "Contact form received",
            "data":    contact,
        })
    })
    
    // Raw body access
    app.POST("/webhook", func(c *blaze.Context) error {
        body := c.Body()
        signature := c.Header("X-Hub-Signature")
        
        // Verify webhook signature
        if !verifySignature(body, signature) {
            return c.Status(401).JSON(blaze.Map{
                "error": "Invalid signature",
            })
        }
        
        return c.JSON(blaze.Map{"status": "processed"})
    })
}
```

### Response Types

```go
func responseTypes(app *blaze.App) {
    // JSON response
    app.GET("/json", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "message": "Hello JSON",
            "timestamp": time.Now(),
        })
    })
    
    // JSON with custom status
    app.GET("/json-status", func(c *blaze.Context) error {
        return c.JSONStatus(201, blaze.Map{
            "created": true,
        })
    })
    
    // Text response
    app.GET("/text", func(c *blaze.Context) error {
        return c.Text("Hello, World!")
    })
    
    // HTML response
    app.GET("/html", func(c *blaze.Context) error {
        html := `
        <html>
            <body>
                <h1>Hello HTML</h1>
                <p>Welcome to Blaze!</p>
            </body>
        </html>`
        return c.HTML(html)
    })
    
    // Custom headers
    app.GET("/headers", func(c *blaze.Context) error {
        c.SetHeader("X-Custom-Header", "MyValue")
        c.SetHeader("X-API-Version", "1.0")
        
        return c.JSON(blaze.Map{"status": "success"})
    })
    
    // Redirect
    app.GET("/redirect", func(c *blaze.Context) error {
        c.Redirect("https://example.com")
        return nil
    })
    
    // Custom redirect with status
    app.GET("/redirect-permanent", func(c *blaze.Context) error {
        c.Redirect("https://example.com", 301)
        return nil
    })
}
```

### Headers and Cookies

```go
func headersAndCookies(app *blaze.App) {
    // Read headers
    app.GET("/headers", func(c *blaze.Context) error {
        userAgent := c.Header("User-Agent")
        contentType := c.Header("Content-Type")
        customHeader := c.Header("X-Custom")
        
        return c.JSON(blaze.Map{
            "user_agent":    userAgent,
            "content_type":  contentType,
            "custom_header": customHeader,
        })
    })
    
    // Set cookies
    app.GET("/set-cookie", func(c *blaze.Context) error {
        c.SetCookie("session_id", "abc123", time.Now().Add(24*time.Hour))
        c.SetCookie("preferences", "dark_mode=true")
        
        return c.JSON(blaze.Map{"status": "cookies set"})
    })
    
    // Read cookies
    app.GET("/cookies", func(c *blaze.Context) error {
        sessionID := c.Cookie("session_id")
        preferences := c.Cookie("preferences")
        
        return c.JSON(blaze.Map{
            "session_id":   sessionID,
            "preferences":  preferences,
        })
    })
    
    // Client IP information
    app.GET("/ip-info", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "ip":          c.IP(),
            "client_ip":   c.GetClientIP(),
            "real_ip":     c.GetRealIP(),
            "remote_addr": c.GetRemoteAddr(),
            "user_agent":  c.UserAgent(),
        })
    })
}
```

## File Handling

### File Upload

```go
func fileUpload(app *blaze.App) {
    // Single file upload
    app.POST("/upload", func(c *blaze.Context) error {
        file, err := c.FormFile("file")
        if err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "No file uploaded",
            })
        }
        
        // Save file
        filename, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
        if err != nil {
            return c.Status(500).JSON(blaze.Map{
                "error": "Failed to save file",
            })
        }
        
        return c.JSON(blaze.Map{
            "message":  "File uploaded successfully",
            "filename": filename,
            "size":     file.Size,
        })
    })
    
    // Multiple file upload
    app.POST("/upload-multiple", func(c *blaze.Context) error {
        files, err := c.FormFiles("files")
        if err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "No files uploaded",
            })
        }
        
        var savedFiles []blaze.Map
        for _, file := range files {
            filename, err := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
            if err != nil {
                continue
            }
            
            savedFiles = append(savedFiles, blaze.Map{
                "original_name": file.Filename,
                "saved_name":    filename,
                "size":         file.Size,
            })
        }
        
        return c.JSON(blaze.Map{
            "message": "Files uploaded",
            "files":   savedFiles,
        })
    })
    
    // File upload with validation
    app.POST("/upload-image", func(c *blaze.Context) error {
        file, err := c.FormFile("image")
        if err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "No image uploaded",
            })
        }
        
        // Validate file type
        if !file.IsImage() {
            return c.Status(400).JSON(blaze.Map{
                "error": "Only image files are allowed",
            })
        }
        
        // Validate file size (5MB max)
        if file.Size > 5*1024*1024 {
            return c.Status(400).JSON(blaze.Map{
                "error": "File too large (max 5MB)",
            })
        }
        
        filename, err := c.SaveUploadedFileToDir(file, "./uploads/images")
        if err != nil {
            return c.Status(500).JSON(blaze.Map{
                "error": "Failed to save image",
            })
        }
        
        return c.JSON(blaze.Map{
            "message":  "Image uploaded successfully",
            "filename": filename,
        })
    })
}
```

### File Download and Serving

```go
func fileDownload(app *blaze.App) {
    // Serve static files
    app.GET("/files/:filename", func(c *blaze.Context) error {
        filename := c.Param("filename")
        filepath := "./uploads/" + filename
        
        // Check if file exists
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
    
    // Stream large files
    app.GET("/stream/:filename", func(c *blaze.Context) error {
        filename := c.Param("filename")
        filepath := "./uploads/" + filename
        
        return c.StreamFile(filepath)
    })
    
    // Get file info
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

### Multipart Form Handling

```go
func multipartForms(app *blaze.App) {
    // Handle multipart form with files and data
    app.POST("/profile", func(c *blaze.Context) error {
        // Get form values
        name := c.FormValue("name")
        email := c.FormValue("email")
        bio := c.FormValue("bio")
        
        // Get uploaded avatar
        avatar, err := c.FormFile("avatar")
        var avatarPath string
        if err == nil {
            avatarPath, _ = c.SaveUploadedFileToDir(avatar, "./uploads/avatars")
        }
        
        // Get multiple photos
        photos, err := c.FormFiles("photos")
        var photoPaths []string
        if err == nil {
            for _, photo := range photos {
                if photo.IsImage() {
                    path, _ := c.SaveUploadedFileToDir(photo, "./uploads/photos")
                    photoPaths = append(photoPaths, path)
                }
            }
        }
        
        return c.JSON(blaze.Map{
            "name":        name,
            "email":       email,
            "bio":         bio,
            "avatar":      avatarPath,
            "photos":      photoPaths,
            "photo_count": len(photoPaths),
        })
    })
    
    // Custom multipart configuration
    app.POST("/upload-config", func(c *blaze.Context) error {
        config := &blaze.MultipartConfig{
            MaxMemory:   10 << 20, // 10MB
            MaxFileSize: 50 << 20, // 50MB
            MaxFiles:    5,
            AllowedExtensions: []string{".jpg", ".png", ".pdf"},
            KeepInMemory: false,
        }
        
        form, err := c.MultipartFormWithConfig(config)
        if err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": err.Error(),
            })
        }
        
        return c.JSON(blaze.Map{
            "files_count": form.GetFileCount(),
            "total_size":  form.GetTotalSize(),
            "fields":      len(form.Value),
        })
    })
}
```

## WebSocket Examples

### Basic WebSocket

```go
func webSocketBasic(app *blaze.App) {
    app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) error {
        log.Println("New WebSocket connection")
        
        for {
            // Read message
            messageType, data, err := ws.ReadMessage()
            if err != nil {
                log.Printf("WebSocket read error: %v", err)
                break
            }
            
            log.Printf("Received: %s", data)
            
            // Echo message back
            if err := ws.WriteMessage(messageType, data); err != nil {
                log.Printf("WebSocket write error: %v", err)
                break
            }
        }
        
        return nil
    })
}
```

### WebSocket Chat Server

```go
type ChatServer struct {
    clients    map[*blaze.WebSocketConnection]bool
    broadcast  chan []byte
    register   chan *blaze.WebSocketConnection
    unregister chan *blaze.WebSocketConnection
    mutex      sync.RWMutex
}

func NewChatServer() *ChatServer {
    return &ChatServer{
        clients:    make(map[*blaze.WebSocketConnection]bool),
        broadcast:  make(chan []byte),
        register:   make(chan *blaze.WebSocketConnection),
        unregister: make(chan *blaze.WebSocketConnection),
    }
}

func (cs *ChatServer) Run() {
    for {
        select {
        case client := <-cs.register:
            cs.mutex.Lock()
            cs.clients[client] = true
            cs.mutex.Unlock()
            log.Printf("Client registered. Total: %d", len(cs.clients))
            
        case client := <-cs.unregister:
            cs.mutex.Lock()
            if _, ok := cs.clients[client]; ok {
                delete(cs.clients, client)
                client.Close()
            }
            cs.mutex.Unlock()
            log.Printf("Client unregistered. Total: %d", len(cs.clients))
            
        case message := <-cs.broadcast:
            cs.mutex.RLock()
            for client := range cs.clients {
                if err := client.WriteMessage(1, message); err != nil {
                    log.Printf("Broadcast error: %v", err)
                    client.Close()
                    delete(cs.clients, client)
                }
            }
            cs.mutex.RUnlock()
        }
    }
}

func webSocketChat(app *blaze.App) {
    chatServer := NewChatServer()
    go chatServer.Run()
    
    app.WebSocket("/chat", func(ws *blaze.WebSocketConnection) error {
        chatServer.register <- ws
        defer func() {
            chatServer.unregister <- ws
        }()
        
        for {
            _, message, err := ws.ReadMessage()
            if err != nil {
                log.Printf("WebSocket error: %v", err)
                break
            }
            
            // Broadcast to all clients
            chatServer.broadcast <- message
        }
        
        return nil
    })
}
```

### WebSocket with Custom Configuration

```go
func webSocketCustom(app *blaze.App) {
    config := &blaze.WebSocketConfig{
        HandshakeTimeout: 10 * time.Second,
        ReadBufferSize:   1024,
        WriteBufferSize:  1024,
        CheckOrigin: func(r *http.Request) bool {
            origin := r.Header.Get("Origin")
            return origin == "https://yourdomain.com"
        },
        EnableCompression: true,
    }
    
    app.WebSocketWithConfig("/ws-custom", func(ws *blaze.WebSocketConnection) error {
        // Set read deadline
        ws.SetReadDeadline(time.Now().Add(60 * time.Second))
        
        // Set pong handler
        ws.SetPongHandler(func(string) error {
            ws.SetReadDeadline(time.Now().Add(60 * time.Second))
            return nil
        })
        
        // Start ping routine
        go func() {
            ticker := time.NewTicker(30 * time.Second)
            defer ticker.Stop()
            
            for range ticker.C {
                if err := ws.WriteMessage(9, []byte{}); err != nil {
                    return
                }
            }
        }()
        
        for {
            messageType, data, err := ws.ReadMessage()
            if err != nil {
                break
            }
            
            if err := ws.WriteMessage(messageType, data); err != nil {
                break
            }
        }
        
        return nil
    }, config)
}
```

## TLS and Security

### HTTPS Server

```go
func httpsServer() {
    config := blaze.ProductionConfig()
    config.EnableTLS = true
    
    app := blaze.NewWithConfig(config)
    
    // Configure TLS
    tlsConfig := &blaze.TLSConfig{
        CertFile: "/path/to/cert.pem",
        KeyFile:  "/path/to/key.pem",
        MinVersion: tls.VersionTLS12,
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        },
        NextProtos: []string{"h2", "http/1.1"},
    }
    
    app.SetTLSConfig(tlsConfig)
    
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "message": "Secure HTTPS connection",
            "tls":     true,
        })
    })
    
    log.Fatal(app.ListenAndServe())
}
```

### Auto TLS for Development

```go
func autoTLS() {
    app := blaze.New()
    
    // Enable auto TLS with self-signed certificates
    app.EnableAutoTLS("localhost", "127.0.0.1")
    
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "message": "Auto TLS enabled",
            "secure":  true,
        })
    })
    
    log.Fatal(app.ListenAndServe())
}
```

### Security Middleware

```go
func securityMiddleware(app *blaze.App) {
    // CSRF Protection
    app.Use(blaze.CSRF(&blaze.CSRFOptions{
        Secret:      []byte("your-32-byte-secret-key-here!!!"),
        TokenLookup: []string{"header:X-CSRF-Token", "form:csrf_token"},
    }))
    
    // Rate Limiting
    app.Use(blaze.RateLimit(&blaze.RateLimitOptions{
        Max:      100,
        Duration: time.Minute,
        KeyGenerator: func(c *blaze.Context) string {
            return c.IP()
        },
    }))
    
    // Security Headers
    app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            c.SetHeader("X-Content-Type-Options", "nosniff")
            c.SetHeader("X-Frame-Options", "DENY")
            c.SetHeader("X-XSS-Protection", "1; mode=block")
            c.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")
            
            if c.Request().URI().Scheme() == "https" {
                c.SetHeader("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
            }
            
            return next(c)
        }
    })
}
```

## HTTP/2 Support

### HTTP/2 Server

```go
func http2Server() {
    config := blaze.ProductionConfig()
    config.EnableHTTP2 = true
    config.EnableTLS = true
    
    app := blaze.NewWithConfig(config)
    
    // Configure HTTP/2
    http2Config := &blaze.HTTP2Config{
        Enabled:              true,
        MaxConcurrentStreams: 1000,
        EnablePush:           true,
        IdleTimeout:          300 * time.Second,
    }
    
    app.SetHTTP2Config(http2Config)
    
    // Configure TLS for HTTP/2
    tlsConfig := &blaze.TLSConfig{
        CertFile:   "/path/to/cert.pem",
        KeyFile:    "/path/to/key.pem",
        NextProtos: []string{"h2", "http/1.1"},
    }
    
    app.SetTLSConfig(tlsConfig)
    
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "protocol": c.Protocol(),
            "http2":    c.IsHTTP2(),
            "stream_id": c.StreamID(),
        })
    })
    
    // Server push example
    app.GET("/page", func(c *blaze.Context) error {
        if c.IsHTTP2() {
            // Push resources
            resources := map[string]string{
                "/assets/style.css":  "style",
                "/assets/script.js":  "script",
                "/assets/image.png":  "image",
            }
            c.PushResources(resources)
        }
        
        html := `
        <html>
            <head>
                <link rel="stylesheet" href="/assets/style.css">
                <script src="/assets/script.js"></script>
            </head>
            <body>
                <h1>HTTP/2 Page</h1>
                <img src="/assets/image.png" alt="Image">
            </body>
        </html>`
        
        return c.HTML(html)
    })
    
    log.Fatal(app.ListenAndServe())
}
```

### HTTP/2 Cleartext (H2C) for Development

```go
func http2Cleartext() {
    config := blaze.DevelopmentConfig()
    config.EnableHTTP2 = true
    
    app := blaze.NewWithConfig(config)
    
    // Configure H2C (HTTP/2 without TLS)
    http2Config := &blaze.HTTP2Config{
        Enabled: true,
        H2C:     true, // Enable HTTP/2 over cleartext
    }
    
    app.SetHTTP2Config(http2Config)
    
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "protocol": c.Protocol(),
            "http2":    c.IsHTTP2(),
            "h2c":      true,
        })
    })
    
    log.Fatal(app.ListenAndServe())
}
```

## Database Integration

### MySQL Example

```go
import (
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
)

func mysqlIntegration(app *blaze.App) {
    // Database connection
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Middleware to inject database
    app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            c.SetLocals("db", db)
            return next(c)
        }
    })
    
    // Get users
    app.GET("/users", func(c *blaze.Context) error {
        db := c.Locals("db").(*sql.DB)
        
        rows, err := db.Query("SELECT id, name, email FROM users")
        if err != nil {
            return c.Status(500).JSON(blaze.Map{
                "error": "Database query failed",
            })
        }
        defer rows.Close()
        
        var users []blaze.Map
        for rows.Next() {
            var id int
            var name, email string
            
            if err := rows.Scan(&id, &name, &email); err != nil {
                continue
            }
            
            users = append(users, blaze.Map{
                "id":    id,
                "name":  name,
                "email": email,
            })
        }
        
        return c.JSON(blaze.Map{
            "users": users,
        })
    })
    
    // Create user
    app.POST("/users", func(c *blaze.Context) error {
        db := c.Locals("db").(*sql.DB)
        
        var user struct {
            Name  string `json:"name"`
            Email string `json:"email"`
        }
        
        if err := c.BindJSON(&user); err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "Invalid JSON",
            })
        }
        
        result, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)", user.Name, user.Email)
        if err != nil {
            return c.Status(500).JSON(blaze.Map{
                "error": "Failed to create user",
            })
        }
        
        id, _ := result.LastInsertId()
        
        return c.Status(201).JSON(blaze.Map{
            "id":    id,
            "name":  user.Name,
            "email": user.Email,
        })
    })
}
```

### Redis Integration

```go
import (
    "github.com/go-redis/redis/v8"
    "context"
)

func redisIntegration(app *blaze.App) {
    // Redis client
    rdb := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
    
    // Test connection
    ctx := context.Background()
    _, err := rdb.Ping(ctx).Result()
    if err != nil {
        log.Fatal("Redis connection failed:", err)
    }
    
    // Middleware to inject Redis client
    app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            c.SetLocals("redis", rdb)
            return next(c)
        }
    })
    
    // Cache endpoint
    app.GET("/cache/:key", func(c *blaze.Context) error {
        rdb := c.Locals("redis").(*redis.Client)
        key := c.Param("key")
        
        val, err := rdb.Get(ctx, key).Result()
        if err == redis.Nil {
            return c.Status(404).JSON(blaze.Map{
                "error": "Key not found",
            })
        } else if err != nil {
            return c.Status(500).JSON(blaze.Map{
                "error": "Redis error",
            })
        }
        
        return c.JSON(blaze.Map{
            "key":   key,
            "value": val,
        })
    })
    
    // Set cache
    app.POST("/cache/:key", func(c *blaze.Context) error {
        rdb := c.Locals("redis").(*redis.Client)
        key := c.Param("key")
        
        var data struct {
            Value string        `json:"value"`
            TTL   time.Duration `json:"ttl"`
        }
        
        if err := c.BindJSON(&data); err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "Invalid JSON",
            })
        }
        
        err := rdb.Set(ctx, key, data.Value, data.TTL).Err()
        if err != nil {
            return c.Status(500).JSON(blaze.Map{
                "error": "Failed to set cache",
            })
        }
        
        return c.JSON(blaze.Map{
            "message": "Cache set successfully",
            "key":     key,
        })
    })
}
```

## Testing Examples

### Unit Tests

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

func TestGetUsers(t *testing.T) {
    app := blaze.New()
    app.GET("/users", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "users": []blaze.Map{
                {"id": 1, "name": "Alice"},
                {"id": 2, "name": "Bob"},
            },
        })
    })
    
    req := httptest.NewRequest("GET", "/users", nil)
    resp := httptest.NewRecorder()
    
    app.ServeHTTP(resp, req)
    
    if resp.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", resp.Code)
    }
    
    var result map[string]interface{}
    if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
        t.Errorf("Failed to parse response: %v", err)
    }
    
    users := result["users"].([]interface{})
    if len(users) != 2 {
        t.Errorf("Expected 2 users, got %d", len(users))
    }
}

func TestCreateUser(t *testing.T) {
    app := blaze.New()
    app.POST("/users", func(c *blaze.Context) error {
        var user struct {
            Name  string `json:"name"`
            Email string `json:"email"`
        }
        
        if err := c.BindJSON(&user); err != nil {
            return c.Status(400).JSON(blaze.Map{
                "error": "Invalid JSON",
            })
        }
        
        return c.Status(201).JSON(blaze.Map{
            "id":    1,
            "name":  user.Name,
            "email": user.Email,
        })
    })
    
    userData := map[string]string{
        "name":  "John Doe",
        "email": "john@example.com",
    }
    
    jsonData, _ := json.Marshal(userData)
    req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(jsonData))
    req.Header.Set("Content-Type", "application/json")
    resp := httptest.NewRecorder()
    
    app.ServeHTTP(resp, req)
    
    if resp.Code != http.StatusCreated {
        t.Errorf("Expected status 201, got %d", resp.Code)
    }
}

func TestMiddleware(t *testing.T) {
    app := blaze.New()
    
    // Add test middleware
    app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            c.SetHeader("X-Test-Middleware", "true")
            return next(c)
        }
    })
    
    app.GET("/test", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{"status": "ok"})
    })
    
    req := httptest.NewRequest("GET", "/test", nil)
    resp := httptest.NewRecorder()
    
    app.ServeHTTP(resp, req)
    
    if resp.Header().Get("X-Test-Middleware") != "true" {
        t.Error("Middleware header not set")
    }
}
```

### Integration Tests

```go
func TestFullAPIFlow(t *testing.T) {
    app := setupTestApp()
    
    // Test creating a user
    userData := map[string]string{
        "name":  "Integration Test User",
        "email": "test@example.com",
    }
    
    jsonData, _ := json.Marshal(userData)
    req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(jsonData))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer test-token")
    resp := httptest.NewRecorder()
    
    app.ServeHTTP(resp, req)
    
    if resp.Code != http.StatusCreated {
        t.Fatalf("Failed to create user: %d", resp.Code)
    }
    
    var createResp map[string]interface{}
    json.Unmarshal(resp.Body.Bytes(), &createResp)
    userID := int(createResp["id"].(float64))
    
    // Test getting the user
    req = httptest.NewRequest("GET", fmt.Sprintf("/api/v1/users/%d", userID), nil)
    req.Header.Set("Authorization", "Bearer test-token")
    resp = httptest.NewRecorder()
    
    app.ServeHTTP(resp, req)
    
    if resp.Code != http.StatusOK {
        t.Errorf("Failed to get user: %d", resp.Code)
    }
    
    // Test updating the user
    updateData := map[string]string{
        "name": "Updated User Name",
    }
    
    jsonData, _ = json.Marshal(updateData)
    req = httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/users/%d", userID), bytes.NewBuffer(jsonData))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer test-token")
    resp = httptest.NewRecorder()
    
    app.ServeHTTP(resp, req)
    
    if resp.Code != http.StatusOK {
        t.Errorf("Failed to update user: %d", resp.Code)
    }
    
    // Test deleting the user
    req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/users/%d", userID), nil)
    req.Header.Set("Authorization", "Bearer test-token")
    resp = httptest.NewRecorder()
    
    app.ServeHTTP(resp, req)
    
    if resp.Code != http.StatusOK {
        t.Errorf("Failed to delete user: %d", resp.Code)
    }
}

func setupTestApp() *blaze.App {
    app := blaze.New()
    
    app.Use(blaze.Logger())
    app.Use(blaze.Recovery())
    
    // Test authentication
    app.Use(blaze.Auth(func(token string) bool {
        return token == "test-token"
    }))
    
    setupAPIRoutes(app)
    
    return app
}
```

## Production Deployment

### Production Configuration

```go
func productionSetup() {
    config := blaze.ProductionConfig()
    
    app := blaze.NewWithConfig(config)
    
    // Production middleware stack
    app.Use(blaze.Recovery())
    app.Use(blaze.Logger())
    app.Use(blaze.RequestID())
    app.Use(blaze.CORS(&blaze.CORSOptions{
        AllowOrigins: []string{
            "https://yourdomain.com",
            "https://www.yourdomain.com",
        },
        AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
        AllowHeaders:     []string{"*"},
        AllowCredentials: true,
    }))
    
    // Rate limiting
    app.Use(blaze.RateLimit(&blaze.RateLimitOptions{
        Max:      1000,
        Duration: time.Hour,
        KeyGenerator: func(c *blaze.Context) string {
            return c.IP()
        },
    }))
    
    // Security headers
    app.Use(blaze.HTTP2Security())
    
    // Cache static resources
    app.Use(blaze.Cache(&blaze.CacheOptions{
        Duration: 24 * time.Hour,
        KeyGenerator: func(c *blaze.Context) string {
            if strings.HasPrefix(c.Path(), "/static/") {
                return c.Path()
            }
            return ""
        },
    }))
    
    setupRoutes(app)
    
    log.Fatal(app.ListenAndServeGraceful())
}
```

### Health Checks and Monitoring

```go
func healthChecks(app *blaze.App) {
    // Basic health check
    app.GET("/health", func(c *blaze.Context) error {
        return c.JSON(blaze.Health("1.0.0", "24h"))
    })
    
    // Detailed health check
    app.GET("/health/detailed", func(c *blaze.Context) error {
        serverInfo := app.GetServerInfo()
        
        return c.JSON(blaze.Map{
            "status":    "healthy",
            "timestamp": time.Now(),
            "version":   "1.0.0",
            "server":    serverInfo,
            "uptime":    "24h",
            "memory":    getMemoryUsage(),
            "database":  checkDatabaseHealth(),
            "redis":     checkRedisHealth(),
        })
    })
    
    // Metrics endpoint
    app.GET("/metrics", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "requests_total":     getRequestCount(),
            "response_time_avg":  getAverageResponseTime(),
            "memory_usage":       getMemoryUsage(),
            "active_connections": getActiveConnections(),
        })
    })
    
    // Readiness probe
    app.GET("/ready", func(c *blaze.Context) error {
        if !isApplicationReady() {
            return c.Status(503).JSON(blaze.Map{
                "status": "not ready",
            })
        }
        
        return c.JSON(blaze.Map{
            "status": "ready",
        })
    })
    
    // Liveness probe
    app.GET("/alive", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "status": "alive",
            "timestamp": time.Now(),
        })
    })
}
```

This comprehensive examples documentation covers all major features of the Blaze framework, providing practical code examples for building production-ready web applications with Go. Each example includes proper error handling, security considerations, and best practices for real-world usage.

