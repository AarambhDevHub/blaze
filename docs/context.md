# Context

The Context is the heart of Blaze, providing a powerful and ergonomic interface for handling HTTP requests and responses. It wraps fasthttp's RequestCtx while adding convenient methods and additional functionality specific to web development.

## Overview

The Context struct represents the complete state of an HTTP request and response cycle. It provides methods for accessing request data, setting response data, handling parameters, working with headers, cookies, and much more.

```go
type Context struct {
    *fasthttp.RequestCtx
    params map[string]string
    locals map[string]interface{}
}
```

## Core Features

### Request Data Access

#### Path Parameters

Extract path parameters defined in your routes:

```go
// Route: /users/:id
app.GET("/users/:id", func(c *blaze.Context) error {
    id := c.Param("id")
    
    // Convert to integer with error handling
    userID, err := c.ParamInt("id")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "Invalid user ID"})
    }
    
    // Get with default value
    page := c.ParamIntDefault("page", 1)
    
    return c.JSON(blaze.Map{"user_id": userID, "page": page})
})
```

#### Query Parameters

Access URL query parameters:

```go
app.GET("/search", func(c *blaze.Context) error {
    query := c.Query("q")
    limit := c.QueryIntDefault("limit", 10)
    category := c.QueryDefault("category", "all")
    
    return c.JSON(blaze.Map{
        "query": query,
        "limit": limit,
        "category": category,
    })
})
```

#### Headers

Read and manipulate HTTP headers:

```go
app.GET("/api/data", func(c *blaze.Context) error {
    auth := c.Header("Authorization")
    userAgent := c.UserAgent()
    contentType := c.GetContentType()
    
    // Set response headers
    c.SetHeader("X-API-Version", "v1.0")
    c.SetHeader("X-Rate-Limit", "1000")
    
    return c.JSON(blaze.Map{"auth": auth != ""})
})
```

#### Request Body

Handle different types of request bodies:

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

app.POST("/users", func(c *blaze.Context) error {
    var user User
    
    // Auto-detect and bind (JSON or form)
    if err := c.Bind(&user); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    // Or bind specific format
    if err := c.BindJSON(&user); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    // Access raw body
    body := c.Body()
    bodyString := c.BodyString()
    
    return c.JSON(user)
})
```

### Response Generation

#### JSON Responses

Send JSON data with automatic serialization:

```go
app.GET("/users/:id", func(c *blaze.Context) error {
    user := getUserByID(c.Param("id"))
    
    // Simple JSON response
    return c.JSON(user)
    
    // JSON with status code
    return c.JSONStatus(201, blaze.Map{
        "user": user,
        "created": true,
    })
})
```

#### Text and HTML Responses

Send plain text or HTML content:

```go
app.GET("/health", func(c *blaze.Context) error {
    return c.Text("OK")
})

app.GET("/welcome", func(c *blaze.Context) error {
    html := "<h1>Welcome to Blaze!</h1>"
    return c.HTML(html)
})

app.GET("/error", func(c *blaze.Context) error {
    return c.TextStatus(500, "Internal Server Error")
})
```

#### Redirects

Redirect requests to other URLs:

```go
app.GET("/old-path", func(c *blaze.Context) error {
    c.Redirect("/new-path", 301) // Permanent redirect
    return nil
})

app.POST("/login", func(c *blaze.Context) error {
    // Process login...
    c.Redirect("/dashboard") // Default 302 redirect
    return nil
})
```

### File Operations

#### File Serving

Serve static files with various options:

```go
app.GET("/download/:file", func(c *blaze.Context) error {
    filename := c.Param("file")
    filepath := path.Join("/uploads", filename)
    
    // Serve file for download
    return c.ServeFileDownload(filepath, filename)
})

app.GET("/view/:file", func(c *blaze.Context) error {
    filepath := path.Join("/images", c.Param("file"))
    
    // Serve file inline (for viewing in browser)
    return c.ServeFileInline(filepath)
})

app.GET("/stream/:video", func(c *blaze.Context) error {
    filepath := path.Join("/videos", c.Param("video"))
    
    // Stream with range request support
    return c.StreamFile(filepath)
})
```

#### File Upload Handling

Process uploaded files with comprehensive multipart form support:

```go
type FileUpload struct {
    Title       string                `form:"title,required"`
    Description string                `form:"description"`
    File        *blaze.MultipartFile  `form:"file,required"`
    Tags        []string              `form:"tags"`
    CreatedAt   *time.Time           `form:"created_at"`
}

app.POST("/upload", func(c *blaze.Context) error {
    var upload FileUpload
    
    // Bind multipart form to struct
    if err := c.BindMultipartForm(&upload); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    // Save the uploaded file
    savedPath, err := c.SaveUploadedFileWithUniqueFilename(upload.File, "/uploads")
    if err != nil {
        return c.Status(500).JSON(blaze.Map{"error": "Failed to save file"})
    }
    
    return c.JSON(blaze.Map{
        "title": upload.Title,
        "file_path": savedPath,
        "file_size": upload.File.Size,
    })
})
```

### Cookies

Manage HTTP cookies:

```go
app.POST("/login", func(c *blaze.Context) error {
    // Process login...
    
    // Set cookie with expiration
    expires := time.Now().Add(24 * time.Hour)
    c.SetCookie("session_id", sessionID, expires)
    
    return c.JSON(blaze.Map{"logged_in": true})
})

app.GET("/profile", func(c *blaze.Context) error {
    sessionID := c.Cookie("session_id")
    if sessionID == "" {
        return c.Status(401).JSON(blaze.Map{"error": "Not authenticated"})
    }
    
    return c.JSON(getProfile(sessionID))
})
```

### Local Storage

Store request-scoped data using locals:

```go
// Middleware to set user context
func AuthMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            token := c.Header("Authorization")
            if token != "" {
                user := validateToken(token)
                c.SetLocals("user", user)
                c.SetLocals("authenticated", true)
            }
            return next(c)
        }
    }
}

app.GET("/dashboard", func(c *blaze.Context) error {
    user := c.Locals("user")
    isAuth := c.Locals("authenticated").(bool)
    
    if !isAuth {
        return c.Status(401).JSON(blaze.Map{"error": "Unauthorized"})
    }
    
    return c.JSON(blaze.Map{"user": user})
})
```

## Advanced Features

### Client IP Detection

Get real client IP addresses:

```go
app.GET("/location", func(c *blaze.Context) error {
    // Various methods to get client IP
    ip := c.IP()                    // Basic IP
    realIP := c.GetRealIP()         // Real IP (considers proxies)
    clientIP := c.GetClientIP()     // Client IP from headers
    remoteAddr := c.GetRemoteAddr() // Full remote address
    
    return c.JSON(blaze.Map{
        "ip": ip,
        "real_ip": realIP,
        "client_ip": clientIP,
        "remote_addr": remoteAddr,
    })
})
```

### Request Information

Access comprehensive request information:

```go
app.GET("/debug", func(c *blaze.Context) error {
    return c.JSON(blaze.Map{
        "method": c.Method(),
        "path": c.Path(),
        "uri": c.URI().String(),
        "user_agent": c.UserAgent(),
        "content_type": c.GetContentType(),
        "is_multipart": c.IsMultipartForm(),
        "protocol": c.Protocol(),
        "is_http2": c.IsHTTP2(),
        "stream_id": c.StreamID(),
    })
})
```

### Form Handling

Handle both URL-encoded and multipart forms:

```go
type ContactForm struct {
    Name    string `form:"name,required"`
    Email   string `form:"email,required"`
    Message string `form:"message,required,minsize=10,maxsize=1000"`
    Phone   string `form:"phone,default=Not provided"`
}

app.POST("/contact", func(c *blaze.Context) error {
    var form ContactForm
    
    // Works with both multipart and URL-encoded forms
    if err := c.BindForm(&form); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    // Or access individual form values
    name := c.FormValue("name")
    email := c.FormValue("email")
    tags := c.FormValues("tags") // Multiple values
    
    return c.JSON(form)
})
```

### Graceful Shutdown Support

Context provides shutdown-aware functionality:

```go
app.GET("/long-task", func(c *blaze.Context) error {
    // Check if server is shutting down
    if c.IsShuttingDown() {
        return c.Status(503).JSON(blaze.Map{
            "error": "Server is shutting down"
        })
    }
    
    // Create timeout context that respects shutdown
    ctx, cancel := c.WithTimeout(30 * time.Second)
    defer cancel()
    
    // Use context in long-running operations
    result, err := performLongTask(ctx)
    if err != nil {
        return c.Status(500).JSON(blaze.Map{"error": err.Error()})
    }
    
    return c.JSON(result)
})
```

### HTTP/2 Features

Take advantage of HTTP/2 capabilities:

```go
app.GET("/dashboard", func(c *blaze.Context) error {
    if c.IsHTTP2() {
        // Push resources for better performance
        resources := map[string]string{
            "/css/dashboard.css": "style",
            "/js/dashboard.js":   "script",
            "/img/logo.png":      "image",
        }
        c.PushResources(resources)
        
        // Get HTTP/2 stream information
        streamID := c.StreamID()
        log.Printf("Processing request on HTTP/2 stream %d", streamID)
    }
    
    return c.HTML(dashboardHTML)
})
```

### User Values

Store arbitrary data in the request context:

```go
app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        // Set user values
        c.SetUserValue("request_id", generateRequestID())
        c.SetUserValue("start_time", time.Now())
        
        err := next(c)
        
        // Access user values
        requestID := c.GetUserValueString("request_id")
        startTime := c.GetUserValue("start_time").(time.Time)
        duration := time.Since(startTime)
        
        log.Printf("Request %s completed in %v", requestID, duration)
        return err
    }
})
```

## Error Handling

The Context integrates seamlessly with Blaze's error handling:

```go
app.GET("/users/:id", func(c *blaze.Context) error {
    id, err := c.ParamInt("id")
    if err != nil {
        // Return error - middleware can handle it
        return blaze.NewHTTPError(400, "Invalid user ID")
    }
    
    user, err := getUserByID(id)
    if err != nil {
        if err == ErrUserNotFound {
            return c.Status(404).JSON(blaze.Map{"error": "User not found"})
        }
        return err // Let error middleware handle it
    }
    
    return c.JSON(user)
})
```

## Best Practices

### 1. Always Check for Errors

```go
app.POST("/upload", func(c *blaze.Context) error {
    file, err := c.FormFile("upload")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file provided"})
    }
    
    if file.Size > maxFileSize {
        return c.Status(413).JSON(blaze.Map{"error": "File too large"})
    }
    
    // Process file...
    return c.JSON(blaze.Map{"uploaded": true})
})
```

### 2. Use Struct Binding for Complex Data

```go
type CreateUserRequest struct {
    Name     string    `json:"name" form:"name,required"`
    Email    string    `json:"email" form:"email,required"`
    Age      int       `json:"age" form:"age"`
    Avatar   *MultipartFile `form:"avatar"`
}

app.POST("/users", func(c *blaze.Context) error {
    var req CreateUserRequest
    if err := c.Bind(&req); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    // Validate and process...
    return c.JSONStatus(201, createUser(req))
})
```

### 3. Leverage Context Locals for Middleware

```go
func RequireAuth() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            user := authenticateUser(c)
            if user == nil {
                return c.Status(401).JSON(blaze.Map{"error": "Unauthorized"})
            }
            
            c.SetLocals("user", user)
            return next(c)
        }
    }
}
```

### 4. Handle Different Content Types

```go
app.POST("/data", func(c *blaze.Context) error {
    contentType := c.GetContentType()
    
    var data map[string]interface{}
    
    switch {
    case strings.Contains(contentType, "application/json"):
        if err := c.BindJSON(&data); err != nil {
            return c.Status(400).JSON(blaze.Map{"error": "Invalid JSON"})
        }
    case strings.Contains(contentType, "multipart/form-data"):
        if err := c.BindMultipartForm(&data); err != nil {
            return c.Status(400).JSON(blaze.Map{"error": "Invalid form data"})
        }
    default:
        return c.Status(415).JSON(blaze.Map{"error": "Unsupported content type"})
    }
    
    return c.JSON(data)
})
```

The Context in Blaze provides a comprehensive, type-safe, and ergonomic interface for handling all aspects of HTTP request and response processing, making it easy to build robust web applications and APIs.