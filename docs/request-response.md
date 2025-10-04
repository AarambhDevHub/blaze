# Request and Response Handling

The request-response documentation covers how to work with HTTP requests and responses in the Blaze Go web framework. Blaze provides a comprehensive `Context` type that wraps FastHTTP functionality and offers extensive methods for handling both incoming requests and outgoing responses.

## Table of Contents

- [Context Structure](#context-structure)
- [Request Handling](#request-handling)
- [Request Body Processing](#request-body-processing)
- [Validation](#validation)
- [Response Handling](#response-handling)
- [File Operations](#file-operations)
- [Multipart Forms](#multipart-forms)
- [Cookies](#cookies)
- [HTTP/2 Features](#http2-features)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)

## Context Structure

The `Context` struct serves as the central interface for request-response handling:

```go
type Context struct {
    *fasthttp.RequestCtx
    params map[string]string
    locals map[string]interface{}
}
```

The context embeds FastHTTP's `RequestCtx` and extends it with parameter storage and local variables for request-scoped data.

## Request Handling

### Route Parameters

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
```

**Available Methods:**
- `Param(key string) string` - Get string parameter
- `ParamInt(key string) (int, error)` - Get integer parameter with error
- `ParamIntDefault(key string, defaultValue int) int` - Get integer with default
- `SetParam(key, value string)` - Set parameter value (useful in middleware)

### Query Parameters

Access URL query parameters with type conversion:

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
    
    // Access raw query args for advanced use
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

### Request Headers

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

### Request Metadata

Access comprehensive request information:

```go
app.GET("/info", func(c *blaze.Context) error {
    return c.JSON(blaze.Map{
        "method":         c.Method(),
        "path":           c.Path(),
        "uri":            c.URI().String(),
        "ip":             c.IP(),
        "real_ip":        c.GetRealIP(),
        "remote_ip":      c.RemoteIP().String(),
        "remote_addr":    c.GetRemoteAddr(),
        "user_agent":     c.UserAgent(),
        "content_type":   c.GetContentType(),
        "protocol":       c.Protocol(),
        "is_http2":       c.IsHTTP2(),
    })
})
```

**Available Methods:**
- `Method() string` - Get HTTP method
- `Path() string` - Get request path
- `URI() *fasthttp.URI` - Get URI object
- `IP() string` - Get client IP address
- `RemoteIP() net.IP` - Get remote IP as net.IP
- `GetRealIP() string` - Get real IP (considers proxies)
- `GetClientIP() string` - Get client IP from headers
- `GetRemoteAddr() string` - Get full remote address
- `Request() *fasthttp.Request` - Get fasthttp request
- `Response() *fasthttp.Response` - Get fasthttp response

## Request Body Processing

### JSON Binding

Bind JSON request bodies to structs:

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

app.POST("/users", func(c *blaze.Context) error {
    var user User
    
    // Bind JSON body
    if err := c.BindJSON(&user); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid JSON data",
        })
    }
    
    return c.Status(201).JSON(user)
})
```

### Auto-Detect Binding

Automatically detect and bind content:

```go
app.POST("/data", func(c *blaze.Context) error {
    var data map[string]interface{}
    
    // Automatically detects JSON or form data
    if err := c.Bind(&data); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid request body",
        })
    }
    
    return c.JSON(data)
})
```

### Raw Body Access

Access raw request body:

```go
app.POST("/webhook", func(c *blaze.Context) error {
    // Get raw body as bytes
    body := c.Body()
    
    // Or as string
    bodyString := c.BodyString()
    
    // Get body metadata
    bodySize := c.GetBodySize()
    contentLength := c.GetContentLength()
    
    // Process webhook payload
    return c.Text("Webhook received")
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

## Validation

### Struct Validation with Binding

Bind and validate in one call:

```go
type UserRegistration struct {
    Name     string `json:"name" validate:"required,min=2,max=100"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Age      int    `json:"age" validate:"gte=18,lte=100"`
}

app.POST("/register", func(c *blaze.Context) error {
    var reg UserRegistration
    
    // Bind and validate in one call
    if err := c.BindAndValidate(&reg); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }
    
    // Process validated data
    return c.Status(201).JSON(reg)
})
```

### JSON-Specific Validation

```go
app.POST("/api/user", func(c *blaze.Context) error {
    var user User
    
    // Bind JSON and validate
    if err := c.BindJSONAndValidate(&user); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    return c.JSON(user)
})
```

### Form Validation

```go
app.POST("/form", func(c *blaze.Context) error {
    var form MyForm
    
    // Bind form data and validate
    if err := c.BindFormAndValidate(&form); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    return c.JSON(form)
})
```

### Single Variable Validation

```go
app.GET("/email/:email", func(c *blaze.Context) error {
    email := c.Param("email")
    
    // Validate single variable
    if err := c.ValidateVar(email, "email"); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid email address",
        })
    }
    
    return c.JSON(blaze.Map{"email": email})
})
```

### Body Size Validation

```go
app.POST("/upload", func(c *blaze.Context) error {
    // Validate body size before processing (10MB max)
    if err := c.ValidateBodySize(10 * 1024 * 1024); err != nil {
        return c.Status(413).JSON(blaze.Map{
            "error": "Request too large",
            "max_size": "10MB",
        })
    }
    
    // Process request
    return c.JSON(blaze.Map{"status": "ok"})
})
```

**Available Methods:**
- `BindAndValidate(v interface{}) error` - Bind and validate
- `BindJSONAndValidate(v interface{}) error` - Bind JSON and validate
- `BindFormAndValidate(v interface{}) error` - Bind form and validate
- `BindMultipartFormAndValidate(v interface{}) error` - Bind multipart and validate
- `Validate(v interface{}) error` - Validate struct
- `ValidateVar(field interface{}, tag string) error` - Validate single variable
- `ValidateBodySize(maxSize int64) error` - Validate body size

## Response Handling

### Status Codes

Set custom HTTP status codes:

```go
app.GET("/resource", func(c *blaze.Context) error {
    return c.Status(200).JSON(blaze.Map{"data": "value"})
})

app.POST("/create", func(c *blaze.Context) error {
    return c.Status(201).JSON(blaze.Map{"created": true})
})

app.GET("/error", func(c *blaze.Context) error {
    return c.Status(500).JSON(blaze.Map{"error": "Internal error"})
})
```

**Method:**
- `Status(status int) *Context` - Set status code (chainable)

### JSON Responses

Send JSON data with automatic serialization:

```go
app.GET("/user", func(c *blaze.Context) error {
    user := User{Name: "John", Email: "john@example.com"}
    
    // Simple JSON response (200 OK)
    return c.JSON(user)
})

app.POST("/user", func(c *blaze.Context) error {
    // JSON with custom status code
    return c.JSONStatus(201, blaze.Map{
        "user": newUser,
        "created": true,
    })
})

// Using helper functions
app.GET("/success", func(c *blaze.Context) error {
    return c.JSON(blaze.OK(data))
})

app.GET("/error", func(c *blaze.Context) error {
    return c.JSON(blaze.Error("Something went wrong"))
})

app.POST("/create", func(c *blaze.Context) error {
    return c.JSON(blaze.Created(resource))
})
```

**Available Methods:**
- `JSON(data interface{}) error` - Send JSON with 200 status
- `JSONStatus(status int, data interface{}) error` - Send JSON with custom status

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
    return c.HTMLStatus(200, "<h1>Page Title</h1>")
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
```

**Method:**
- `Redirect(url string, status ...int)` - Redirect to URL (default 302)

### Chainable Response Methods

Build responses with method chaining:

```go
app.GET("/api/data", func(c *blaze.Context) error {
    return c.
        Status(200).
        SetHeader("X-API-Version", "v1.0").
        SetHeader("X-Rate-Limit", "1000").
        SetHeader("Cache-Control", "public, max-age=3600").
        JSON(blaze.Map{"data": "value"})
})
```

## File Operations

### File Serving

Serve files with various options:

```go
app.GET("/files/:filename", func(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := "./uploads/" + filename
    
    // Check if file exists
    if !c.FileExists(filepath) {
        return c.Status(404).JSON(blaze.Map{"error": "File not found"})
    }
    
    // Get file info
    fileInfo, err := c.GetFileInfo(filepath)
    if err != nil {
        return c.Status(404).JSON(blaze.Map{"error": "File not found"})
    }
    
    // Serve file
    return c.ServeFile(filepath)
})
```

### File Downloads

Force file download with custom filename:

```go
app.GET("/download/:file", func(c *blaze.Context) error {
    filename := c.Param("file")
    filepath := "./files/" + filename
    
    // Force download
    return c.ServeFileDownload(filepath, filename)
})

// Alternative methods
app.GET("/download2", func(c *blaze.Context) error {
    return c.Download("./file.pdf", "custom_name.pdf")
})

app.GET("/attachment", func(c *blaze.Context) error {
    return c.Attachment("./file.pdf", "attached.pdf")
})
```

### Inline File Display

Serve files for inline display:

```go
app.GET("/view/:file", func(c *blaze.Context) error {
    filename := c.Param("file")
    filepath := "./images/" + filename
    
    // Serve inline (browser displays if possible)
    return c.ServeFileInline(filepath)
})
```

### File Streaming

Stream large files with range request support:

```go
app.GET("/stream/:video", func(c *blaze.Context) error {
    filename := c.Param("video")
    filepath := "./videos/" + filename
    
    // Stream with range support for video seeking
    return c.StreamFile(filepath)
})
```

**Available Methods:**
- `SendFile(filepath string) error` - Send file to client
- `ServeFile(filepath string) error` - Serve file with proper headers
- `ServeFileDownload(filepath, filename string) error` - Force download
- `ServeFileInline(filepath string) error` - Inline display
- `StreamFile(filepath string) error` - Stream with range support
- `FileExists(filepath string) bool` - Check if file exists
- `GetFileInfo(filepath string) (os.FileInfo, error)` - Get file metadata
- `Download(filepath, filename string) error` - Alias for download
- `Attachment(filepath, filename string) error` - Alias for download

## Multipart Forms

### Struct-Based Form Binding

Bind multipart forms directly to structs with validation:

```go
type FileUploadForm struct {
    Title       string                `form:"title,required,minsize:2"`
    Description string                `form:"description,maxsize:500"`
    File        *blaze.MultipartFile  `form:"file,required"`
    Tags        []string              `form:"tags"`
    Published   bool                  `form:"published"`
}

app.POST("/upload", func(c *blaze.Context) error {
    var form FileUploadForm
    
    // Bind and validate in one call
    if err := c.BindMultipartFormAndValidate(&form); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }
    
    // Save file
    savedPath, err := c.SaveUploadedFileWithUniqueFilename(form.File, "./uploads")
    if err != nil {
        return c.Status(500).JSON(blaze.Map{"error": "Failed to save"})
    }
    
    return c.JSON(blaze.Map{
        "title": form.Title,
        "path":  savedPath,
        "size":  form.File.Size,
    })
})
```

### Traditional File Upload

Handle single and multiple file uploads:

```go
// Single file
app.POST("/upload-single", func(c *blaze.Context) error {
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file"})
    }
    
    savedPath, _ := c.SaveUploadedFileWithUniqueFilename(file, "./uploads")
    
    return c.JSON(blaze.Map{
        "filename": file.Filename,
        "path":     savedPath,
        "size":     file.Size,
        "type":     file.ContentType,
    })
})

// Multiple files
app.POST("/upload-multiple", func(c *blaze.Context) error {
    files, err := c.FormFiles("files")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No files"})
    }
    
    var savedFiles []string
    for _, file := range files {
        path, _ := c.SaveUploadedFileToDir(file, "./uploads")
        savedFiles = append(savedFiles, path)
    }
    
    return c.JSON(blaze.Map{"files": savedFiles})
})
```

### Form Data Access

```go
app.POST("/form", func(c *blaze.Context) error {
    name := c.FormValue("name")
    email := c.FormValue("email")
    tags := c.FormValues("tags") // Multiple values
    
    isMultipart := c.IsMultipartForm()
    
    return c.JSON(blaze.Map{
        "name":        name,
        "email":       email,
        "tags":        tags,
        "multipart":   isMultipart,
    })
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
- `FormValue(name string) string` - Get form value
- `FormValues(name string) []string` - Get multiple values
- `IsMultipartForm() bool` - Check if multipart
- `BindMultipartForm(v interface{}) error` - Bind to struct
- `BindMultipartFormAndValidate(v interface{}) error` - Bind and validate

## Cookies

Manage HTTP cookies:

```go
app.POST("/login", func(c *blaze.Context) error {
    // Set simple cookie
    c.SetCookie("session_id", sessionID)
    
    // Set cookie with expiration
    expires := time.Now().Add(24 * time.Hour)
    c.SetCookie("user_pref", "dark_mode", expires)
    
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

## HTTP/2 Features

### Protocol Detection

Check HTTP/2 support and version:

```go
app.GET("/", func(c *blaze.Context) error {
    protocol := c.Protocol() // "HTTP/1.1" or "HTTP/2.0"
    isHTTP2 := c.IsHTTP2()
    
    return c.JSON(blaze.Map{
        "protocol": protocol,
        "is_http2": isHTTP2,
    })
})
```

### Server Push

Push resources to clients using HTTP/2:

```go
app.GET("/dashboard", func(c *blaze.Context) error {
    if c.IsHTTP2() {
        // Push individual resource
        c.ServerPush("/css/main.css", "style")
        
        // Push multiple resources
        resources := map[string]string{
            "/css/dashboard.css": "style",
            "/js/dashboard.js":   "script",
            "/img/logo.png":      "image",
        }
        c.PushResources(resources)
        
        // Get HTTP/2 stream ID
        streamID := c.StreamID()
        log.Printf("Processing on stream %d", streamID)
    }
    
    return c.HTML(dashboardHTML)
})
```

**Available Methods:**
- `IsHTTP2() bool` - Check if request uses HTTP/2
- `Protocol() string` - Get protocol version
- `StreamID() uint32` - Get HTTP/2 stream ID
- `ServerPush(path, contentType string) error` - Push single resource
- `PushResources(resources map[string]string) error` - Push multiple resources

## Advanced Features

### Local Storage

Store request-scoped data:

```go
// In middleware
func AuthMiddleware(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        user := authenticateUser(c)
        c.SetLocals("user", user)
        c.SetLocals("authenticated", true)
        return next(c)
    }
}

// In handler
app.GET("/profile", func(c *blaze.Context) error {
    user := c.Locals("user")
    isAuth, _ := c.Locals("authenticated").(bool)
    
    if !isAuth {
        return c.Status(401).JSON(blaze.Map{"error": "Unauthorized"})
    }
    
    return c.JSON(user)
})
```

**Available Methods:**
- `Locals(key string) interface{}` - Get local value
- `SetLocals(key string, value interface{}) *Context` - Set local value (chainable)

### User Values

Alternative storage mechanism:

```go
c.SetUserValue("request_id", "123")
c.SetUserValue("start_time", time.Now())

requestID := c.GetUserValueString("request_id")
startTime := c.GetUserValue("start_time").(time.Time)
intValue := c.GetUserValueInt("count")
```

**Available Methods:**
- `GetUserValue(key string) interface{}` - Get user value
- `SetUserValue(key string, value interface{}) *Context` - Set user value
- `GetUserValueString(key string) string` - Get as string
- `GetUserValueInt(key string) int` - Get as int

### Application State

Access application-level state:

```go
app.SetState("api_key", "secret123")
app.SetState("max_uploads", 10)

app.GET("/config", func(c *blaze.Context) error {
    apiKey := c.StateString("api_key")
    maxUploads := c.StateInt("max_uploads")
    enabled := c.StateBool("feature_enabled")
    
    return c.JSON(blaze.Map{
        "api_key":     apiKey,
        "max_uploads": maxUploads,
    })
})
```

**Available Methods:**
- `State(key string) (interface{}, bool)` - Get state value
- `MustState(key string) interface{}` - Get state or panic
- `StateString(key string) string` - Get as string
- `StateInt(key string) int` - Get as int
- `StateBool(key string) bool` - Get as bool

### Logging

Request-specific logging:

```go
app.GET("/api/data", func(c *blaze.Context) error {
    logger := c.Logger()
    
    c.LogDebug("Processing request", "user_id", 123)
    c.LogInfo("Data retrieved successfully")
    c.LogWarn("Rate limit approaching", "remaining", 10)
    c.LogError("Failed to fetch data", "error", err)
    
    return c.JSON(data)
})
```

**Available Methods:**
- `Logger() *Logger` - Get request-specific logger
- `LogDebug(msg string, args ...interface{})` - Log debug message
- `LogInfo(msg string, args ...interface{})` - Log info message
- `LogWarn(msg string, args ...interface{})` - Log warning message
- `LogError(msg string, args ...interface{})` - Log error message

### Graceful Shutdown Support

Handle shutdown gracefully:

```go
app.GET("/long-task", func(c *blaze.Context) error {
    // Check if shutting down
    if c.IsShuttingDown() {
        return c.Status(503).JSON(blaze.Map{
            "error": "Server shutting down",
        })
    }
    
    // Get shutdown context
    ctx := c.ShutdownContext()
    
    // Create timeout context
    timeoutCtx, cancel := c.WithTimeout(30 * time.Second)
    defer cancel()
    
    // Or deadline context
    deadlineCtx, cancel := c.WithDeadline(time.Now().Add(1 * time.Minute))
    defer cancel()
    
    // Use in operations
    result, err := performTask(timeoutCtx)
    return c.JSON(result)
})
```

**Available Methods:**
- `IsShuttingDown() bool` - Check if server is shutting down
- `ShutdownContext() context.Context` - Get shutdown context
- `WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc)` - Create timeout context
- `WithDeadline(deadline time.Time) (context.Context, context.CancelFunc)` - Create deadline context

## Best Practices

### Always Validate Input

```go
app.POST("/user", func(c *blaze.Context) error {
    var user User
    
    // Always validate
    if err := c.BindJSONAndValidate(&user); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }
    
    return c.JSON(user)
})
```

### Use Struct Binding

```go
type CreateRequest struct {
    Name  string `json:"name" validate:"required,min=2"`
    Email string `json:"email" validate:"required,email"`
}

app.POST("/create", func(c *blaze.Context) error {
    var req CreateRequest
    if err := c.BindAndValidate(&req); err != nil {
        return c.Status(400).JSON(blaze.Map{"error": err.Error()})
    }
    
    return c.JSON(req)
})
```

### Handle Errors Properly

```go
app.GET("/resource/:id", func(c *blaze.Context) error {
    id, err := c.ParamInt("id")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid ID",
            "code": "INVALID_ID",
        })
    }
    
    resource, err := getResource(id)
    if err != nil {
        return c.Status(404).JSON(blaze.Map{
            "error": "Resource not found",
            "code": "NOT_FOUND",
        })
    }
    
    return c.JSON(resource)
})
```

### Use Method Chaining

```go
app.GET("/api/data", func(c *blaze.Context) error {
    return c.
        Status(200).
        SetHeader("X-API-Version", "v1.0").
        SetHeader("Cache-Control", "public, max-age=3600").
        JSON(blaze.Map{"data": "value"})
})
```

The request-response handling in Blaze provides a comprehensive, type-safe, and ergonomic interface for all aspects of HTTP communication, with advanced features like validation, HTTP/2 support, and graceful shutdown handling.