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

## Table of Contents

- [Request Data Access](#request-data-access)
- [Response Generation](#response-generation)
- [File Operations](#file-operations)
- [Data Binding](#data-binding)
- [Body Validation](#body-validation)
- [Cookies](#cookies)
- [Local Storage](#local-storage)
- [Client Information](#client-information)
- [HTTP/2 Features](#http2-features)
- [Graceful Shutdown Support](#graceful-shutdown-support)
- [Application State](#application-state)
- [Logging](#logging)
- [Best Practices](#best-practices)

## Request Data Access

### Path Parameters

Extract path parameters defined in your routes:

```go
// Route: /users/:id
app.GET("/users/:id", func(c *blaze.Context) error {
    // Get string parameter
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

// Set parameter (useful in middleware)
c.SetParam("key", "value")
```

**Available Methods:**
- `Param(key string) string` - Get string parameter
- `ParamInt(key string) (int, error)` - Get integer parameter with error
- `ParamIntDefault(key string, defaultValue int) int` - Get integer with default
- `SetParam(key, value string)` - Set parameter value

### Query Parameters

Access URL query parameters:

```go
app.GET("/search", func(c *blaze.Context) error {
    // Get string query parameter
    query := c.Query("q")
    
    // Get with default value
    category := c.QueryDefault("category", "all")
    
    // Convert to integer
    limit, err := c.QueryInt("limit")
    if err != nil {
        limit = 10 // default
    }
    
    // Get with default integer value
    page := c.QueryIntDefault("page", 1)
    
    // Access raw query args
    queryArgs := c.QueryArgs()
    
    return c.JSON(blaze.Map{
        "query":    query,
        "category": category,
        "limit":    limit,
        "page":     page,
    })
})
```

**Available Methods:**
- `Query(key string) string` - Get query parameter
- `QueryDefault(key, defaultValue string) string` - Get with default
- `QueryInt(key string) (int, error)` - Get as integer
- `QueryIntDefault(key string, defaultValue int) int` - Get integer with default
- `QueryArgs() *fasthttp.Args` - Get raw query arguments

### Headers

Read and manipulate HTTP headers:

```go
app.GET("/api/data", func(c *blaze.Context) error {
    // Read request headers
    auth := c.Header("Authorization")
    userAgent := c.UserAgent()
    contentType := c.GetContentType()
    
    // Set response headers
    c.SetHeader("X-API-Version", "v1.0")
    c.SetHeader("X-Rate-Limit", "1000")
    c.SetHeader("Cache-Control", "public, max-age=3600")
    
    return c.JSON(blaze.Map{"auth": auth != ""})
})
```

**Available Methods:**
- `Header(key string) string` - Get request header
- `SetHeader(key, value string) *Context` - Set response header (chainable)
- `GetContentType() string` - Get request content type
- `SetContentType(contentType string)` - Set response content type
- `UserAgent() string` - Get User-Agent header

### Request Body

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
    postBody := c.PostBody()
    
    // Get body size
    bodySize := c.GetBodySize()
    contentLength := c.GetContentLength()
    
    return c.JSON(user)
})
```

**Available Methods:**
- `Body() []byte` - Get request body
- `PostBody() []byte` - Get POST body
- `BodyString() string` - Get body as string
- `GetBodySize() int64` - Get body size in bytes
- `GetContentLength() int` - Get Content-Length header value
- `Bind(v interface{}) error` - Auto-detect and bind
- `BindJSON(v interface{}) error` - Bind JSON body

### Request Information

Access comprehensive request information:

```go
app.GET("/debug", func(c *blaze.Context) error {
    return c.JSON(blaze.Map{
        "method":         c.Method(),
        "path":           c.Path(),
        "uri":            c.URI().String(),
        "user_agent":     c.UserAgent(),
        "content_type":   c.GetContentType(),
        "is_multipart":   c.IsMultipartForm(),
        "protocol":       c.Protocol(),
        "is_http2":       c.IsHTTP2(),
        "stream_id":      c.StreamID(),
    })
})
```

**Available Methods:**
- `Method() string` - Get HTTP method
- `Path() string` - Get request path
- `URI() *fasthttp.URI` - Get URI object
- `Request() *fasthttp.Request` - Get fasthttp request
- `Response() *fasthttp.Response` - Get fasthttp response

## Response Generation

### JSON Responses

Send JSON data with automatic serialization:

```go
app.GET("/users/:id", func(c *blaze.Context) error {
    user := getUserByID(c.Param("id"))
    
    // Simple JSON response (200 OK)
    return c.JSON(user)
})

app.POST("/users", func(c *blaze.Context) error {
    var user User
    c.BindJSON(&user)
    
    // JSON with custom status code
    return c.JSONStatus(201, blaze.Map{
        "user":    user,
        "created": true,
    })
})

// Using helper functions
app.GET("/api/data", func(c *blaze.Context) error {
    // 200 OK
    return c.JSON(blaze.OK(data))
    
    // 201 Created
    return c.JSON(blaze.Created(data))
    
    // 400 Bad Request
    return c.JSON(blaze.Error("Invalid input"))
})
```

**Available Methods:**
- `JSON(data interface{}) error` - Send JSON with 200 status
- `JSONStatus(status int, data interface{}) error` - Send JSON with custom status
- `Status(status int) *Context` - Set status code (chainable)

### Text and HTML Responses

Send plain text or HTML content:

```go
app.GET("/health", func(c *blaze.Context) error {
    return c.Text("OK")
})

app.GET("/error", func(c *blaze.Context) error {
    return c.TextStatus(500, "Internal Server Error")
})

app.GET("/welcome", func(c *blaze.Context) error {
    html := "<h1>Welcome to Blaze!</h1>"
    return c.HTML(html)
})

app.GET("/page", func(c *blaze.Context) error {
    html := "<h1>Page Title</h1><p>Content</p>"
    return c.HTMLStatus(200, html)
})
```

**Available Methods:**
- `Text(text string) error` - Send plain text with 200 status
- `TextStatus(status int, text string) error` - Send text with custom status
- `HTML(html string) error` - Send HTML with 200 status
- `HTMLStatus(status int, html string) error` - Send HTML with custom status
- `WriteString(s string) (int, error)` - Write string to response

### Redirects

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

// Using helper
app.GET("/external", func(c *blaze.Context) error {
    return blaze.Redirect(c, "https://example.com", 302)
})
```

**Available Methods:**
- `Redirect(url string, status ...int)` - Redirect to URL (default 302)

## File Operations

### File Serving

Serve static files with various options:

```go
app.GET("/download/:file", func(c *blaze.Context) error {
    filename := c.Param("file")
    filepath := path.Join("/uploads", filename)
    
    // Check if file exists
    if !c.FileExists(filepath) {
        return c.Status(404).Text("File not found")
    }
    
    // Get file info
    fileInfo, err := c.GetFileInfo(filepath)
    if err != nil {
        return err
    }
    
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

app.GET("/file/:name", func(c *blaze.Context) error {
    filepath := path.Join("/files", c.Param("name"))
    
    // Simple file serving
    return c.SendFile(filepath)
})
```

**Available Methods:**
- `SendFile(filepath string) error` - Send file as response
- `ServeFile(filepath string) error` - Serve file with proper headers
- `ServeFileDownload(filepath, filename string) error` - Force download
- `ServeFileInline(filepath string) error` - Inline display
- `StreamFile(filepath string) error` - Stream with range request support
- `FileExists(filepath string) bool` - Check if file exists
- `GetFileInfo(filepath string) (os.FileInfo, error)` - Get file information
- `Download(filepath, filename string) error` - Alias for ServeFileDownload
- `Attachment(filepath, filename string) error` - Alias for ServeFileDownload

### File Upload Handling

Process uploaded files with comprehensive multipart form support:

```go
type FileUpload struct {
    Title       string                `form:"title,required"`
    Description string                `form:"description"`
    File        *blaze.MultipartFile  `form:"file,required"`
    Tags        []string              `form:"tags"`
    Files       []*blaze.MultipartFile `form:"files"`
    CreatedAt   *time.Time            `form:"created_at"`
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
        "title":     upload.Title,
        "file_path": savedPath,
        "file_size": upload.File.Size,
    })
})

// Single file upload
app.POST("/upload-single", func(c *blaze.Context) error {
    // Get single file
    file, err := c.FormFile("upload")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file provided"})
    }
    
    // Save with original filename
    if err := c.SaveUploadedFile(file, "/uploads/"+file.Filename); err != nil {
        return c.Status(500).JSON(blaze.Map{"error": "Failed to save"})
    }
    
    return c.JSON(blaze.Map{"uploaded": true})
})

// Multiple file upload
app.POST("/upload-multiple", func(c *blaze.Context) error {
    files, err := c.FormFiles("files")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No files provided"})
    }
    
    var savedPaths []string
    for _, file := range files {
        path, err := c.SaveUploadedFileToDir(file, "/uploads")
        if err != nil {
            continue
        }
        savedPaths = append(savedPaths, path)
    }
    
    return c.JSON(blaze.Map{"files": savedPaths})
})
```

**Available Methods:**
- `MultipartForm() (*MultipartForm, error)` - Parse multipart form
- `MultipartFormWithConfig(config MultipartConfig) (*MultipartForm, error)` - Parse with config
- `FormFile(name string) (*MultipartFile, error)` - Get single file
- `FormFiles(name string) ([]*MultipartFile, error)` - Get multiple files
- `SaveUploadedFile(file *MultipartFile, dst string) error` - Save file
- `SaveUploadedFileToDir(file *MultipartFile, dir string) (string, error)` - Save to directory
- `SaveUploadedFileWithUniqueFilename(file *MultipartFile, dir string) (string, error)` - Save with unique name
- `IsMultipartForm() bool` - Check if request is multipart

## Data Binding

### Form Binding

Handle both URL-encoded and multipart forms:

```go
type ContactForm struct {
    Name    string `form:"name,required"`
    Email   string `form:"email,required"`
    Message string `form:"message,required,minsize:10,maxsize:1000"`
    Phone   string `form:"phone,default:Not provided"`
}

app.POST("/contact", func(c *blaze.Context) error {
    var form ContactForm
    
    // Works with both multipart and URL-encoded forms
    if err := c.BindMultipartForm(&form); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    // Or access individual form values
    name := c.FormValue("name")
    email := c.FormValue("email")
    tags := c.FormValues("tags") // Multiple values
    
    return c.JSON(form)
})
```

**Available Methods:**
- `BindMultipartForm(v interface{}) error` - Bind multipart form to struct
- `FormValue(name string) string` - Get single form value
- `FormValues(name string) []string` - Get multiple form values

## Body Validation

### Validation Methods

Validate request body size and struct data:

```go
app.POST("/upload", func(c *blaze.Context) error {
    // Validate body size (10MB max)
    if err := c.ValidateBodySize(10 * 1024 * 1024); err != nil {
        return c.Status(413).JSON(blaze.Map{"error": err.Error()})
    }
    
    var data MyStruct
    
    // Bind and validate in one call
    if err := c.BindAndValidate(&data); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    return c.JSON(data)
})

// JSON binding with validation
app.POST("/api/data", func(c *blaze.Context) error {
    var data MyStruct
    
    if err := c.BindJSONAndValidate(&data); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    return c.JSON(data)
})

// Validate struct without binding
app.POST("/validate", func(c *blaze.Context) error {
    data := getDataFromSomewhere()
    
    if err := c.Validate(data); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    return c.JSON(blaze.Map{"valid": true})
})

// Validate single variable
app.GET("/email/:email", func(c *blaze.Context) error {
    email := c.Param("email")
    
    if err := c.ValidateVar(email, "email"); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "Invalid email"})
    }
    
    return c.JSON(blaze.Map{"email": email})
})
```

**Available Methods:**
- `ValidateBodySize(maxSize int64) error` - Validate request body size
- `BindAndValidate(v interface{}) error` - Bind and validate
- `BindJSONAndValidate(v interface{}) error` - Bind JSON and validate
- `BindFormAndValidate(v interface{}) error` - Bind form and validate
- `BindMultipartFormAndValidate(v interface{}) error` - Bind multipart and validate
- `Validate(v interface{}) error` - Validate struct
- `ValidateVar(field interface{}, tag string) error` - Validate single variable

## Cookies

Manage HTTP cookies:

```go
app.POST("/login", func(c *blaze.Context) error {
    // Process login...
    
    // Set cookie with expiration
    expires := time.Now().Add(24 * time.Hour)
    c.SetCookie("session_id", sessionID, expires)
    
    // Set cookie without expiration (session cookie)
    c.SetCookie("temp_token", token)
    
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

**Available Methods:**
- `Cookie(name string) string` - Get cookie value
- `SetCookie(name, value string, expires ...time.Time) *Context` - Set cookie (chainable)

## Local Storage

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
    isAuth, _ := c.Locals("authenticated").(bool)
    
    if !isAuth {
        return c.Status(401).JSON(blaze.Map{"error": "Unauthorized"})
    }
    
    return c.JSON(blaze.Map{"user": user})
})

// User values (alternative to locals)
app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        c.SetUserValue("request_id", generateRequestID())
        c.SetUserValue("start_time", time.Now())
        
        err := next(c)
        
        requestID := c.GetUserValueString("request_id")
        startTime := c.GetUserValue("start_time").(time.Time)
        duration := time.Since(startTime)
        
        log.Printf("Request %s completed in %v", requestID, duration)
        return err
    }
})
```

**Available Methods:**
- `Locals(key string) interface{}` - Get local value
- `SetLocals(key string, value interface{}) *Context` - Set local value (chainable)
- `GetUserValue(key string) interface{}` - Get user value
- `SetUserValue(key string, value interface{}) *Context` - Set user value (chainable)
- `GetUserValueString(key string) string` - Get user value as string
- `GetUserValueInt(key string) int` - Get user value as int

## Client Information

Get real client IP addresses and information:

```go
app.GET("/location", func(c *blaze.Context) error {
    // Various methods to get client IP
    ip := c.IP()                    // Basic IP
    realIP := c.GetRealIP()         // Real IP (considers proxies)
    clientIP := c.GetClientIP()     // Client IP from headers
    remoteAddr := c.GetRemoteAddr() // Full remote address
    remoteIP := c.RemoteIP()        // RemoteIP as net.IP
    
    // User agent
    userAgent := c.UserAgent()
    
    return c.JSON(blaze.Map{
        "ip":          ip,
        "real_ip":     realIP,
        "client_ip":   clientIP,
        "remote_addr": remoteAddr,
        "user_agent":  userAgent,
    })
})
```

**Available Methods:**
- `IP() string` - Get client IP address
- `RemoteIP() net.IP` - Get remote IP as net.IP
- `GetRealIP() string` - Get real IP (considers X-Real-IP, X-Forwarded-For)
- `GetClientIP() string` - Get client IP from headers
- `GetRemoteAddr() string` - Get full remote address
- `UserAgent() string` - Get User-Agent header

## HTTP/2 Features

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
        
        // Or push individual resource
        c.ServerPush("/css/main.css", "style")
        
        // Get HTTP/2 stream information
        streamID := c.StreamID()
        log.Printf("Processing request on HTTP/2 stream %d", streamID)
    }
    
    // Get protocol version
    protocol := c.Protocol() // "HTTP/1.1" or "HTTP/2.0"
    
    return c.HTML(dashboardHTML)
})
```

**Available Methods:**
- `IsHTTP2() bool` - Check if request uses HTTP/2
- `Protocol() string` - Get protocol version (HTTP/1.1 or HTTP/2.0)
- `StreamID() uint32` - Get HTTP/2 stream ID
- `ServerPush(path, contentType string) error` - Push single resource
- `PushResources(resources map[string]string) error` - Push multiple resources

## Graceful Shutdown Support

Context provides shutdown-aware functionality:

```go
app.GET("/long-task", func(c *blaze.Context) error {
    // Check if server is shutting down
    if c.IsShuttingDown() {
        return c.Status(503).JSON(blaze.Map{
            "error": "Server is shutting down"
        })
    }
    
    // Get shutdown context
    shutdownCtx := c.ShutdownContext()
    
    // Create timeout context that respects shutdown
    ctx, cancel := c.WithTimeout(30 * time.Second)
    defer cancel()
    
    // Or use deadline
    deadline := time.Now().Add(1 * time.Minute)
    ctx, cancel = c.WithDeadline(deadline)
    defer cancel()
    
    // Use context in long-running operations
    result, err := performLongTask(ctx)
    if err != nil {
        return c.Status(500).JSON(blaze.Map{"error": err.Error()})
    }
    
    return c.JSON(result)
})
```

**Available Methods:**
- `IsShuttingDown() bool` - Check if server is shutting down
- `ShutdownContext() context.Context` - Get shutdown context
- `WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc)` - Create timeout context
- `WithDeadline(deadline time.Time) (context.Context, context.CancelFunc)` - Create deadline context

## Application State

Access application-level state from context:

```go
// Set state at app level
app.SetState("api_key", "secret123")
app.SetState("max_uploads", 10)
app.SetState("enabled", true)

// Access in handlers
app.GET("/config", func(c *blaze.Context) error {
    // Get state with type assertion
    apiKey, exists := c.State("api_key")
    if !exists {
        return c.Status(500).JSON(blaze.Map{"error": "Config not found"})
    }
    
    // Get state or panic
    maxUploads := c.MustState("max_uploads")
    
    // Get state with type helpers
    apiKeyStr := c.StateString("api_key")
    maxUploadsInt := c.StateInt("max_uploads")
    enabledBool := c.StateBool("enabled")
    
    return c.JSON(blaze.Map{
        "api_key":     apiKeyStr,
        "max_uploads": maxUploadsInt,
        "enabled":     enabledBool,
    })
})
```

**Available Methods:**
- `State(key string) (interface{}, bool)` - Get state value
- `MustState(key string) interface{}` - Get state or panic
- `StateString(key string) string` - Get state as string
- `StateInt(key string) int` - Get state as int
- `StateBool(key string) bool` - Get state as bool

## Logging

Context provides request-specific logging:

```go
app.GET("/api/data", func(c *blaze.Context) error {
    // Get logger with request context
    logger := c.Logger()
    
    // Log with different levels
    c.LogDebug("Processing request", "user_id", 123)
    c.LogInfo("Data retrieved successfully")
    c.LogWarn("Rate limit approaching", "remaining", 10)
    c.LogError("Failed to fetch data", "error", err)
    
    // Logger includes request context automatically
    // (request_id, method, path, etc.)
    
    return c.JSON(data)
})
```

**Available Methods:**
- `Logger() *Logger` - Get request-specific logger
- `LogDebug(msg string, args ...interface{})` - Log debug message
- `LogInfo(msg string, args ...interface{})` - Log info message
- `LogWarn(msg string, args ...interface{})` - Log warning message
- `LogError(msg string, args ...interface{})` - Log error message

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
    Name   string              `json:"name" form:"name,required"`
    Email  string              `json:"email" form:"email,required"`
    Age    int                 `json:"age" form:"age"`
    Avatar *blaze.MultipartFile `form:"avatar"`
}

app.POST("/users", func(c *blaze.Context) error {
    var req CreateUserRequest
    if err := c.BindAndValidate(&req); err != nil {
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

### 5. Use Chainable Methods

```go
app.GET("/api/data", func(c *blaze.Context) error {
    return c.
        Status(200).
        SetHeader("X-API-Version", "v1.0").
        SetHeader("Cache-Control", "public, max-age=3600").
        JSON(blaze.Map{"data": "value"})
})
```

The Context in Blaze provides a comprehensive, type-safe, and ergonomic interface for handling all aspects of HTTP request and response processing, making it easy to build robust web applications and APIs.