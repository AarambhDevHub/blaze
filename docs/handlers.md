# Handlers

Blaze handlers are functions that process HTTP requests and generate responses. They follow a simple, intuitive pattern and provide powerful features for building robust web applications.

## Table of Contents

- [Handler Function Signature](#handler-function-signature)
- [Handler Types](#handler-types)
- [Request Handling](#request-handling)
- [Request Body Processing](#request-body-processing)
- [Validation Handlers](#validation-handlers)
- [File Upload Handlers](#file-upload-handlers)
- [File Serving Handlers](#file-serving-handlers)
- [Response Manipulation](#response-manipulation)
- [Advanced Handler Features](#advanced-handler-features)
- [HTTP/2 Handler Features](#http2-handler-features)
- [Error Handling](#error-handling)
- [Handler Composition](#handler-composition)
- [Best Practices](#best-practices)

## Handler Function Signature

```go
type HandlerFunc func(*Context) error
```

All Blaze handlers receive a `Context` pointer and return an `error`. The context provides access to request data, response writing capabilities, and various utility methods.

### Basic Handler Example

```go
func helloHandler(c *blaze.Context) error {
    return c.JSON(blaze.Map{
        "message": "Hello, World!",
        "status":  "success",
    })
}

func main() {
    app := blaze.New()
    app.GET("/hello", helloHandler)
    app.ListenAndServe()
}
```

## Handler Types

### JSON Handlers

Return JSON responses using the built-in JSON serialization:

```go
// Simple JSON response
func getUserHandler(c *blaze.Context) error {
    userID := c.Param("id")
    
    user := User{
        ID:    userID,
        Name:  "John Doe",
        Email: "john@example.com",
    }
    
    return c.JSON(user)
}

// JSON with custom status code
func createUserHandler(c *blaze.Context) error {
    var user User
    if err := c.BindJSON(&user); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid JSON data",
        })
    }
    
    // Save user logic here
    
    return c.JSONStatus(201, blaze.Map{
        "message": "User created successfully",
        "user":    user,
    })
}

// Using helper response functions
func successHandler(c *blaze.Context) error {
    data := getData()
    return c.JSON(blaze.OK(data))
}

func createHandler(c *blaze.Context) error {
    newResource := createResource()
    return c.JSON(blaze.Created(newResource))
}

func errorHandler(c *blaze.Context) error {
    return c.JSON(blaze.Error("Something went wrong"))
}

// Paginated response
func listUsersHandler(c *blaze.Context) error {
    users := getUsersFromDB()
    total := getTotalUsers()
    page := c.QueryIntDefault("page", 1)
    perPage := c.QueryIntDefault("per_page", 10)
    
    return c.JSON(blaze.Paginate(users, total, page, perPage))
}
```

### Text Handlers

Return plain text responses:

```go
func pingHandler(c *blaze.Context) error {
    return c.Text("pong")
}

func healthCheckHandler(c *blaze.Context) error {
    return c.TextStatus(200, "Service is healthy")
}
```

### HTML Handlers

Return HTML content:

```go
func homeHandler(c *blaze.Context) error {
    html := `
    <!DOCTYPE html>
    <html>
    <head><title>Welcome</title></head>
    <body><h1>Welcome to Blaze!</h1></body>
    </html>
    `
    return c.HTML(html)
}

func errorPageHandler(c *blaze.Context) error {
    return c.HTMLStatus(404, "<h1>Page Not Found</h1>")
}
```

## Request Handling

### Path Parameters

Access URL path parameters using the `Param` method:

```go
func getUserByIDHandler(c *blaze.Context) error {
    // For route: /users/:id
    userID := c.Param("id")
    
    // Convert to integer with error handling
    id, err := c.ParamInt("id")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid user ID",
        })
    }
    
    // Or use default value
    page := c.ParamIntDefault("page", 1)
    
    return c.JSON(blaze.Map{
        "user_id": userID,
        "id":      id,
        "page":    page,
    })
}
```

### Query Parameters

Handle URL query parameters:

```go
func searchHandler(c *blaze.Context) error {
    // Get query parameter
    query := c.Query("q")
    
    // Get with default value
    limit := c.QueryDefault("limit", "10")
    
    // Get as integer
    page, err := c.QueryInt("page")
    if err != nil {
        page = 1
    }
    
    // Or use default integer
    pageSize := c.QueryIntDefault("size", 20)
    
    return c.JSON(blaze.Map{
        "query":     query,
        "limit":     limit,
        "page":      page,
        "page_size": pageSize,
    })
}
```

### Request Headers

Access and manipulate request headers:

```go
func headerHandler(c *blaze.Context) error {
    // Get specific headers
    userAgent := c.Header("User-Agent")
    authorization := c.Header("Authorization")
    contentType := c.GetContentType()
    
    // Set response headers
    c.SetHeader("X-Custom-Header", "MyValue")
    c.SetHeader("X-Request-ID", "12345")
    c.SetHeader("Cache-Control", "public, max-age=3600")
    
    return c.JSON(blaze.Map{
        "user_agent":    userAgent,
        "authorization": authorization != "",
        "content_type":  contentType,
    })
}
```

## Request Body Processing

### JSON Body Binding

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   int    `json:"age"`
}

// JSON body binding
func createUserJSONHandler(c *blaze.Context) error {
    var req CreateUserRequest
    
    if err := c.BindJSON(&req); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid JSON format",
        })
    }
    
    return c.JSON(blaze.Map{
        "message": "User will be created",
        "data":    req,
    })
}
```

### Auto-Detect Binding

```go
func flexibleBindHandler(c *blaze.Context) error {
    var data map[string]interface{}
    
    // Automatically detects JSON or form data
    if err := c.Bind(&data); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid request body",
        })
    }
    
    return c.JSON(data)
}
```

### Raw Body Access

```go
func webhookHandler(c *blaze.Context) error {
    // Get raw body as bytes
    body := c.Body()
    
    // Or as string
    bodyString := c.BodyString()
    
    // Get body size
    bodySize := c.GetBodySize()
    contentLength := c.GetContentLength()
    
    // Process webhook payload
    return c.Text("Webhook received")
}
```

### Form Data Handling

```go
func formHandler(c *blaze.Context) error {
    name := c.FormValue("name")
    email := c.FormValue("email")
    
    // Get all values for a field
    tags := c.FormValues("tags")
    
    return c.JSON(blaze.Map{
        "name":  name,
        "email": email,
        "tags":  tags,
    })
}
```

## Validation Handlers

### Struct Validation with Binding

```go
type UserRegistration struct {
    Name     string `json:"name" validate:"required,min=2,max=100"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
    Age      int    `json:"age" validate:"gte=18,lte=100"`
}

func registerHandler(c *blaze.Context) error {
    var reg UserRegistration
    
    // Bind and validate in one call
    if err := c.BindAndValidate(&reg); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }
    
    // Process validated data
    return c.Status(201).JSON(blaze.Map{
        "message": "User registered successfully",
        "user":    reg,
    })
}

// JSON-specific binding with validation
func jsonValidationHandler(c *blaze.Context) error {
    var data UserRegistration
    
    if err := c.BindJSONAndValidate(&data); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": err.Error(),
        })
    }
    
    return c.JSON(blaze.OK(data))
}
```

### Single Variable Validation

```go
func validateEmailHandler(c *blaze.Context) error {
    email := c.Query("email")
    
    // Validate single variable
    if err := c.ValidateVar(email, "email"); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid email address",
        })
    }
    
    return c.JSON(blaze.Map{
        "email": email,
        "valid": true,
    })
}
```

### Body Size Validation

```go
func uploadHandler(c *blaze.Context) error {
    // Validate body size before processing
    if err := c.ValidateBodySize(10 * 1024 * 1024); err != nil {
        return c.Status(413).JSON(blaze.Map{
            "error": "Request too large",
            "max_size": "10MB",
        })
    }
    
    // Process request
    return c.JSON(blaze.Map{"status": "ok"})
}
```

## File Upload Handlers

### Single File Upload with Struct Binding

```go
type FileUploadForm struct {
    Title       string                `form:"title,required"`
    Description string                `form:"description,maxsize:500"`
    File        *blaze.MultipartFile  `form:"file,required"`
    Tags        []string              `form:"tags"`
}

func uploadWithStructHandler(c *blaze.Context) error {
    var form FileUploadForm
    
    // Bind multipart form to struct with validation
    if err := c.BindMultipartFormAndValidate(&form); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }
    
    // Save file with unique filename
    savedPath, err := c.SaveUploadedFileWithUniqueFilename(form.File, "./uploads")
    if err != nil {
        return c.Status(500).JSON(blaze.Map{
            "error": "Failed to save file",
        })
    }
    
    return c.JSON(blaze.Map{
        "message":  "File uploaded successfully",
        "title":    form.Title,
        "filename": form.File.Filename,
        "path":     savedPath,
        "size":     form.File.Size,
        "tags":     form.Tags,
    })
}
```

### Traditional File Upload

```go
func uploadHandler(c *blaze.Context) error {
    // Single file upload
    file, err := c.FormFile("avatar")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "No file uploaded",
        })
    }
    
    // Validate file
    if file.Size > 10*1024*1024 {
        return c.Status(413).JSON(blaze.Map{
            "error": "File too large (max 10MB)",
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
        "message":      "File uploaded successfully",
        "filename":     file.Filename,
        "saved_path":   filename,
        "size":         file.Size,
        "content_type": file.ContentType,
        "is_image":     file.IsImage(),
    })
}
```

### Multiple File Upload

```go
func multiUploadHandler(c *blaze.Context) error {
    // Multiple files
    files, err := c.FormFiles("documents")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "No files uploaded",
        })
    }
    
    var savedFiles []blaze.Map
    for _, file := range files {
        // Validate and save each file
        if file.Size > 10*1024*1024 {
            continue // Skip files larger than 10MB
        }
        
        filename, err := c.SaveUploadedFileToDir(file, "./documents")
        if err != nil {
            continue
        }
        
        savedFiles = append(savedFiles, blaze.Map{
            "original": file.Filename,
            "saved":    filename,
            "size":     file.Size,
            "type":     file.ContentType,
        })
    }
    
    return c.JSON(blaze.Map{
        "message": "Files uploaded",
        "files":   savedFiles,
        "count":   len(savedFiles),
    })
}
```

## File Serving Handlers

### Serve Files

```go
func serveImageHandler(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := "./images/" + filename
    
    // Check if file exists
    if !c.FileExists(filepath) {
        return c.Status(404).JSON(blaze.Map{
            "error": "File not found",
        })
    }
    
    // Serve file for inline display
    return c.ServeFileInline(filepath)
}
```

### Force Download

```go
func downloadHandler(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := "./downloads/" + filename
    
    // Force download with custom filename
    return c.ServeFileDownload(filepath, "downloaded_"+filename)
}
```

### Stream Large Files

```go
func streamVideoHandler(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := "./videos/" + filename
    
    // Stream large files with range request support
    return c.StreamFile(filepath)
}
```

## Response Manipulation

### Status Codes

Set custom HTTP status codes:

```go
func statusHandler(c *blaze.Context) error {
    action := c.Query("action")
    
    switch action {
    case "created":
        return c.Status(201).JSON(blaze.Map{"status": "created"})
    case "accepted":
        return c.Status(202).JSON(blaze.Map{"status": "accepted"})
    case "notfound":
        return c.Status(404).JSON(blaze.Map{"error": "not found"})
    case "error":
        return c.Status(500).JSON(blaze.Map{"error": "server error"})
    default:
        return c.JSON(blaze.Map{"status": "ok"})
    }
}
```

### Redirects

Implement redirects with different status codes:

```go
func redirectHandler(c *blaze.Context) error {
    // Temporary redirect (302)
    c.Redirect("/new-location")
    return nil
}

func permanentRedirectHandler(c *blaze.Context) error {
    // Permanent redirect (301)
    c.Redirect("/new-location", 301)
    return nil
}
```

### Cookie Handling

Manage cookies in handlers:

```go
func setCookieHandler(c *blaze.Context) error {
    // Set simple cookie
    c.SetCookie("session_id", "abc123")
    
    // Set cookie with expiration
    expiry := time.Now().Add(24 * time.Hour)
    c.SetCookie("user_pref", "dark_mode", expiry)
    
    return c.Text("Cookies set")
}

func getCookieHandler(c *blaze.Context) error {
    sessionID := c.Cookie("session_id")
    userPref := c.Cookie("user_pref")
    
    return c.JSON(blaze.Map{
        "session_id": sessionID,
        "user_pref":  userPref,
    })
}
```

### Chainable Response Methods

```go
func chainedResponseHandler(c *blaze.Context) error {
    return c.
        Status(200).
        SetHeader("X-API-Version", "v1.0").
        SetHeader("X-Rate-Limit", "1000").
        SetHeader("Cache-Control", "public, max-age=3600").
        JSON(blaze.Map{
            "data": "value",
        })
}
```

## Advanced Handler Features

### Context Local Storage

Store and retrieve data within request context:

```go
func authMiddleware(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        // Set local data in middleware
        c.SetLocals("user_id", "12345")
        c.SetLocals("role", "admin")
        c.SetLocals("authenticated", true)
        
        return next(c)
    }
}

func protectedHandler(c *blaze.Context) error {
    // Get local data
    userID := c.Locals("user_id")
    role := c.Locals("role")
    authenticated, _ := c.Locals("authenticated").(bool)
    
    if !authenticated {
        return c.Status(401).JSON(blaze.Map{
            "error": "Unauthorized",
        })
    }
    
    return c.JSON(blaze.Map{
        "user_id": userID,
        "role":    role,
        "message": "Access granted",
    })
}
```

### Client Information

Access client and connection information:

```go
func clientInfoHandler(c *blaze.Context) error {
    return c.JSON(blaze.Map{
        "ip":           c.IP(),
        "real_ip":      c.GetRealIP(),
        "client_ip":    c.GetClientIP(),
        "remote_addr":  c.GetRemoteAddr(),
        "user_agent":   c.UserAgent(),
        "method":       c.Method(),
        "path":         c.Path(),
        "protocol":     c.Protocol(),
        "is_http2":     c.IsHTTP2(),
    })
}
```

### Application State Access

```go
func stateHandler(c *blaze.Context) error {
    // Access application state
    apiKey := c.StateString("api_key")
    maxUploads := c.StateInt("max_uploads")
    enabled := c.StateBool("feature_enabled")
    
    return c.JSON(blaze.Map{
        "api_key":     apiKey,
        "max_uploads": maxUploads,
        "enabled":     enabled,
    })
}
```

### Context Logging

```go
func loggedHandler(c *blaze.Context) error {
    // Get request-specific logger
    logger := c.Logger()
    
    // Log with different levels
    c.LogDebug("Processing request", "user_id", 123)
    c.LogInfo("Data retrieved successfully")
    c.LogWarn("Rate limit approaching", "remaining", 10)
    c.LogError("Failed to fetch data", "error", "connection timeout")
    
    return c.JSON(blaze.Map{"status": "ok"})
}
```

## HTTP/2 Handler Features

### Server Push

```go
func http2PushHandler(c *blaze.Context) error {
    if c.IsHTTP2() {
        // Push individual resource
        c.ServerPush("/css/main.css", "style")
        
        // Or push multiple resources
        resources := map[string]string{
            "/css/dashboard.css": "style",
            "/js/dashboard.js":   "script",
            "/img/logo.png":      "image",
        }
        c.PushResources(resources)
        
        // Get stream information
        streamID := c.StreamID()
        log.Printf("Processing on stream %d", streamID)
    }
    
    html := `<!DOCTYPE html>
    <html>
        <head>
            <link rel="stylesheet" href="/css/dashboard.css">
            <script src="/js/dashboard.js"></script>
        </head>
        <body>
            <h1>HTTP/2 Dashboard</h1>
            <img src="/img/logo.png" alt="Logo">
        </body>
    </html>`
    
    return c.HTML(html)
}
```

### Protocol Detection

```go
func protocolHandler(c *blaze.Context) error {
    protocol := c.Protocol() // "HTTP/1.1" or "HTTP/2.0"
    
    var features []string
    if c.IsHTTP2() {
        features = append(features, "Server Push", "Multiplexing", "Binary Protocol")
    } else {
        features = append(features, "Text Protocol", "Sequential")
    }
    
    return c.JSON(blaze.Map{
        "protocol": protocol,
        "features": features,
    })
}
```

## Error Handling

### Structured Error Responses

```go
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func errorHandler(c *blaze.Context) error {
    userID := c.Param("id")
    if userID == "" {
        return c.Status(400).JSON(APIError{
            Code:    "MISSING_PARAMETER",
            Message: "User ID is required",
            Details: "The 'id' parameter must be provided in the URL path",
        })
    }
    
    // Using built-in error types
    if userID == "forbidden" {
        return blaze.ErrForbidden("Access denied")
    }
    
    if userID == "notfound" {
        return blaze.ErrNotFound("User not found")
    }
    
    return c.JSON(blaze.Map{"user": "User data"})
}
```

### Validation Error Handling

```go
func validationErrorHandler(c *blaze.Context) error {
    var user User
    
    if err := c.BindJSONAndValidate(&user); err != nil {
        // Check if it's a validation error
        if validationErr, ok := err.(*blaze.ValidationErrors); ok {
            return c.Status(422).JSON(blaze.Map{
                "error": "Validation failed",
                "errors": validationErr.Errors,
            })
        }
        
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid request",
        })
    }
    
    return c.JSON(user)
}
```

## Graceful Shutdown Handling

Create handlers that respect graceful shutdown:

```go
func longRunningHandler(c *blaze.Context) error {
    // Check if shutting down
    if c.IsShuttingDown() {
        return c.Status(503).JSON(blaze.Map{
            "error": "Service shutting down",
        })
    }
    
    // Create context with timeout that respects shutdown
    ctx, cancel := c.WithTimeout(30 * time.Second)
    defer cancel()
    
    // Simulate long-running operation
    select {
    case <-time.After(5 * time.Second):
        return c.JSON(blaze.Map{"result": "Operation completed"})
    case <-ctx.Done():
        if c.IsShuttingDown() {
            return c.Status(503).JSON(blaze.Map{
                "error": "Service shutting down",
            })
        }
        return c.Status(408).JSON(blaze.Map{
            "error": "Request timeout",
        })
    }
}
```

## Handler Composition

### Creating Handler Factories

```go
func AuthenticatedHandler(handler blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        token := c.Header("Authorization")
        if token == "" {
            return c.Status(401).JSON(blaze.Map{
                "error": "Authentication required",
            })
        }
        
        // Validate token
        user := validateToken(token)
        if user == nil {
            return c.Status(401).JSON(blaze.Map{
                "error": "Invalid token",
            })
        }
        
        c.SetLocals("user", user)
        return handler(c)
    }
}

func CachedHandler(handler blaze.HandlerFunc, ttl time.Duration) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        cacheKey := c.Method() + c.Path()
        
        // Check cache
        if cached := getFromCache(cacheKey); cached != nil {
            return c.JSON(cached)
        }
        
        // Execute handler
        return handler(c)
    }
}

// Usage
func main() {
    app := blaze.New()
    
    app.GET("/protected", AuthenticatedHandler(protectedResource))
    app.GET("/cached", CachedHandler(expensiveOperation, 5*time.Minute))
    
    app.ListenAndServe()
}
```

## Best Practices

### Resource Management

```go
func databaseHandler(c *blaze.Context) error {
    // Always use proper resource management
    db, err := database.Connect()
    if err != nil {
        return c.Status(500).JSON(blaze.Map{
            "error": "Database connection failed",
        })
    }
    defer db.Close()
    
    // Use request context for operations
    ctx := c.ShutdownContext()
    
    users, err := db.GetUsersWithContext(ctx)
    if err != nil {
        if errors.Is(err, context.Canceled) {
            return c.Status(503).JSON(blaze.Map{
                "error": "Request canceled due to shutdown",
            })
        }
        return c.Status(500).JSON(blaze.Map{
            "error": "Database query failed",
        })
    }
    
    return c.JSON(users)
}
```

### Input Validation

```go
func validateAndProcess(c *blaze.Context) error {
    var input struct {
        Name  string `json:"name" validate:"required,min=2,max=100"`
        Email string `json:"email" validate:"required,email"`
        Age   int    `json:"age" validate:"gte=0,lte=150"`
    }
    
    // Bind and validate
    if err := c.BindJSONAndValidate(&input); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Validation failed",
            "details": err.Error(),
        })
    }
    
    // Process valid input
    return c.JSON(blaze.Map{
        "message": "Data processed successfully",
        "data":    input,
    })
}
```

### Performance Optimization

```go
func optimizedHandler(c *blaze.Context) error {
    // Set appropriate cache headers
    c.SetHeader("Cache-Control", "public, max-age=3600")
    c.SetHeader("ETag", generateETag())
    
    // Check if client has valid cache
    if c.Header("If-None-Match") == generateETag() {
        return c.Status(304).Text("Not Modified")
    }
    
    // Regular response
    return c.JSON(blaze.Map{
        "data": "value",
    })
}
```

This comprehensive handlers documentation covers all aspects of handler implementation in Blaze, from basic request handling to advanced patterns with validation, HTTP/2 support, and production-ready best practices.