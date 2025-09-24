# Handlers

Blaze handlers are functions that process HTTP requests and generate responses. They follow a simple, intuitive pattern and provide powerful features for building robust web applications.

### Handler Function Signature

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
        ID:   userID,
        Name: "John Doe",
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
    // Get specific header
    userAgent := c.Header("User-Agent")
    authorization := c.Header("Authorization")
    
    // Set response headers
    c.SetHeader("X-Custom-Header", "MyValue")
    c.SetHeader("X-Request-ID", "12345")
    
    return c.JSON(blaze.Map{
        "user_agent":    userAgent,
        "authorization": authorization != "",
    })
}
```

### Request Body Processing

Handle different types of request bodies:

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
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

// Raw body access
func webhookHandler(c *blaze.Context) error {
    // Get raw body as bytes
    body := c.Body()
    
    // Or as string
    bodyString := c.BodyString()
    
    // Process webhook payload
    return c.Text("Webhook received")
}

// Form data handling
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

## File Upload Handlers

Handle file uploads with comprehensive support:

```go
func uploadHandler(c *blaze.Context) error {
    // Single file upload
    file, err := c.FormFile("avatar")
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
        "type":     file.ContentType,
    })
}

func multiUploadHandler(c *blaze.Context) error {
    // Multiple files
    files, err := c.FormFiles("documents")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "No files uploaded",
        })
    }
    
    var savedFiles []string
    for _, file := range files {
        filename, err := c.SaveUploadedFileToDir(file, "./documents")
        if err != nil {
            continue
        }
        savedFiles = append(savedFiles, filename)
    }
    
    return c.JSON(blaze.Map{
        "message": "Files uploaded",
        "files":   savedFiles,
        "count":   len(savedFiles),
    })
}
```

## File Serving Handlers

Serve static files and downloads:

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

func downloadHandler(c *blaze.Context) error {
    filename := c.Param("filename")
    filepath := "./downloads/" + filename
    
    // Force download with custom filename
    return c.ServeFileDownload(filepath, "downloaded_"+filename)
}

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

## Advanced Handler Features

### Context Local Storage

Store and retrieve data within request context:

```go
func middlewareHandler(c *blaze.Context) error {
    // Set local data (typically done in middleware)
    c.SetLocals("user_id", "12345")
    c.SetLocals("role", "admin")
    
    return c.JSON(blaze.Map{"status": "middleware executed"})
}

func protectedHandler(c *blaze.Context) error {
    // Get local data
    userID := c.Locals("user_id")
    role := c.Locals("role")
    
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
        "user_agent":   c.UserAgent(),
        "method":       c.Method(),
        "path":         c.Path(),
        "protocol":     c.Protocol(),
        "is_http2":     c.IsHTTP2(),
        "client_ip":    c.GetClientIP(),
        "real_ip":      c.GetRealIP(),
        "remote_addr":  c.GetRemoteAddr(),
    })
}
```

### Graceful Shutdown Handling

Create handlers that respect graceful shutdown:

```go
func longRunningHandler(c *blaze.Context) error {
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

## Error Handling

### Structured Error Responses

```go
type APIError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func errorHandler(c *blaze.Context) error {
    // Validate input
    userID := c.Param("id")
    if userID == "" {
        return c.Status(400).JSON(APIError{
            Code:    4001,
            Message: "User ID is required",
            Details: "The 'id' parameter must be provided in the URL path",
        })
    }
    
    // Simulate database error
    if userID == "error" {
        return c.Status(500).JSON(APIError{
            Code:    5001,
            Message: "Database connection failed",
            Details: "Unable to connect to the user database",
        })
    }
    
    return c.JSON(blaze.Map{"user": "User data here"})
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
        
        // Validate token logic here
        c.SetLocals("authenticated", true)
        
        return handler(c)
    }
}

func CachedHandler(handler blaze.HandlerFunc, ttl time.Duration) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        // Check cache logic here
        cacheKey := c.Method() + c.Path()
        
        // If not in cache, execute handler
        return handler(c)
    }
}

// Usage
func main() {
    app := blaze.New()
    
    // Apply handler wrappers
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
    
    // Use request context for database operations
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
    // Always validate input
    var input struct {
        Name  string `json:"name"`
        Email string `json:"email"`
        Age   int    `json:"age"`
    }
    
    if err := c.BindJSON(&input); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid JSON format",
        })
    }
    
    // Validate required fields
    if input.Name == "" {
        return c.Status(400).JSON(blaze.Map{
            "error": "Name is required",
        })
    }
    
    if !isValidEmail(input.Email) {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid email format",
        })
    }
    
    if input.Age < 0 || input.Age > 150 {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid age",
        })
    }
    
    // Process valid input
    return c.JSON(blaze.Map{
        "message": "Data processed successfully",
        "data":    input,
    })
}

func isValidEmail(email string) bool {
    // Email validation logic
    return strings.Contains(email, "@")
}
```

### Performance Considerations

```go
func optimizedHandler(c *blaze.Context) error {
    // Set appropriate cache headers
    c.SetHeader("Cache-Control", "public, max-age=3600")
    
    // Use streaming for large responses
    if c.Query("stream") == "true" {
        // Stream response for large data
        c.SetHeader("Content-Type", "application/json")
        writer := c.ResponseWriter()
        
        fmt.Fprintf(writer, `{"items":[`)
        for i := 0; i < 1000; i++ {
            if i > 0 {
                fmt.Fprintf(writer, ",")
            }
            fmt.Fprintf(writer, `{"id":%d}`, i)
        }
        fmt.Fprintf(writer, `]}`)
        
        return nil
    }
    
    // Regular response
    return c.JSON(blaze.Map{"message": "Optimized response"})
}
```

This comprehensive guide covers all aspects of handler implementation in the Blaze framework, from basic request handling to advanced patterns and best practices.