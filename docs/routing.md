# Routing

Blaze provides a powerful and flexible routing system built on a high-performance radix tree implementation. The router supports advanced features including route parameters, wildcards, constraints, middleware, route groups, and all HTTP methods including CONNECT, TRACE, ANY, and Match.

## Table of Contents

- [Basic Route Registration](#basic-route-registration)
- [Supported HTTP Methods](#supported-http-methods)
- [Route Parameters](#route-parameters)
- [Route Constraints](#route-constraints)
- [Route Groups](#route-groups)
- [Route Options](#route-options)
- [Query Parameters](#query-parameters)
- [WebSocket Routes](#websocket-routes)
- [Static File Routes](#static-file-routes)
- [Router Configuration](#router-configuration)
- [Advanced Features](#advanced-features)
- [Best Practices](#best-practices)

## Basic Route Registration

Routes are registered using HTTP method functions on the main application instance:

```go
app := blaze.New()

// Basic GET route
app.GET("/", func(c *blaze.Context) error {
    return c.JSON(blaze.Map{"message": "Hello, World!"})
})

// POST route
app.POST("/users", func(c *blaze.Context) error {
    return c.JSON(blaze.Map{"success": true})
})

// Other HTTP methods
app.PUT("/users/:id", updateUserHandler)
app.DELETE("/users/:id", deleteUserHandler)
app.PATCH("/users/:id", patchUserHandler)
app.HEAD("/health", healthCheckHandler)
app.OPTIONS("/api/*", corsHandler)
```

## Supported HTTP Methods

Blaze supports all standard and extended HTTP methods:

### Standard HTTP Methods

- **GET** - Retrieve resources
- **POST** - Create new resources
- **PUT** - Update/replace resources
- **DELETE** - Remove resources
- **PATCH** - Partially update resources
- **HEAD** - Get headers only
- **OPTIONS** - CORS preflight requests

### Extended HTTP Methods

- **CONNECT** - Establish tunnel connections
- **TRACE** - Request tracing and debugging
- **ANY** - Handle all HTTP methods with a single handler
- **Match** - Handle specific multiple HTTP methods

### Advanced Method Handling

```go
// ANY route handles all HTTP methods
app.ANY("/api/health", func(c *blaze.Context) error {
    return c.JSON(blaze.Map{
        "status":    "healthy",
        "method":    c.Method(),
        "timestamp": time.Now(),
    })
})

// Match specific methods
app.Match([]string{"GET", "POST", "PUT"}, "/api/data", func(c *blaze.Context) error {
    switch c.Method() {
    case "GET":
        return c.JSON(blaze.Map{"action": "retrieve"})
    case "POST":
        return c.JSON(blaze.Map{"action": "create"})
    case "PUT":
        return c.JSON(blaze.Map{"action": "update"})
    default:
        return c.Status(405).JSON(blaze.Error("Method not allowed"))
    }
})

// CONNECT method for tunneling
app.CONNECT("/tunnel/:target", func(c *blaze.Context) error {
    target := c.Param("target")
    // Implement tunneling logic
    return c.Text("CONNECT tunnel established to " + target)
})

// TRACE method for debugging
app.TRACE("/debug", func(c *blaze.Context) error {
    headers := make(map[string]string)
    c.Request().Header.VisitAll(func(key, value []byte) {
        headers[string(key)] = string(value)
    })
    
    return c.JSON(blaze.Map{
        "method":  c.Method(),
        "path":    c.Path(),
        "headers": headers,
        "body":    c.BodyString(),
    })
})
```

## Route Parameters

### Named Parameters

Use `:parameter` syntax for capturing path segments:

```go
// Single parameter
app.GET("/users/:id", func(c *blaze.Context) error {
    id := c.Param("id")
    return c.JSON(blaze.Map{"user_id": id})
})

// Multiple parameters
app.GET("/users/:userID/posts/:postID", func(c *blaze.Context) error {
    userID := c.Param("userID")
    postID := c.Param("postID")
    
    return c.JSON(blaze.Map{
        "user_id": userID,
        "post_id": postID,
    })
})

// Parameters work with all HTTP methods
app.CONNECT("/users/:id/session", func(c *blaze.Context) error {
    id := c.Param("id")
    return c.Text("Session established for user " + id)
})
```

### Parameter Helpers

The context provides helper methods for type conversion:

```go
app.GET("/users/:id", func(c *blaze.Context) error {
    // Get as integer with error handling
    id, err := c.ParamInt("id")
    if err != nil {
        return c.Status(400).JSON(blaze.Error("Invalid user ID"))
    }
    
    // Get as integer with default value
    page := c.ParamIntDefault("page", 1)
    
    return c.JSON(blaze.Map{"id": id, "page": page})
})
```

**Available Parameter Methods:**
- `Param(key string) string` - Get string parameter
- `ParamInt(key string) (int, error)` - Get integer parameter
- `ParamIntDefault(key string, defaultValue int) int` - Get integer with default
- `SetParam(key, value string)` - Set parameter (useful in middleware)

### Wildcard Parameters

Use `*parameter` syntax for catch-all routes:

```go
// Catch-all for file serving
app.GET("/static/*filepath", func(c *blaze.Context) error {
    filepath := c.Param("filepath")
    return c.SendFile("./public/" + filepath)
})

// API versioning catch-all with ANY method
app.ANY("/api/v1/*path", func(c *blaze.Context) error {
    path := c.Param("path")
    return c.JSON(blaze.Map{
        "api_path": path,
        "method":   c.Method(),
    })
})
```

## Route Constraints

Blaze provides a powerful constraint system to validate route parameters before handler execution:

### Built-in Constraints

```go
// Integer constraint
app.GET("/users/:id", userHandler,
    blaze.WithIntConstraint("id"))

// UUID constraint
app.GET("/items/:uuid", itemHandler,
    blaze.WithUUIDConstraint("uuid"))

// Custom regex constraint
app.GET("/products/:sku", productHandler,
    blaze.WithRegexConstraint("sku", `^[A-Z]{2}-\d{4}$`))

// Constraints work with all HTTP methods
app.CONNECT("/servers/:id", serverConnectHandler,
    blaze.WithIntConstraint("id"))

app.TRACE("/sessions/:uuid", sessionTraceHandler,
    blaze.WithUUIDConstraint("uuid"))
```

### Custom Constraints

Create custom constraints for specific validation needs:

```go
// Define custom constraint
ageConstraint := blaze.RouteConstraint{
    Name:    "age",
    Type:    blaze.RegexConstraint,
    Pattern: regexp.MustCompile(`^\d{1,3}$`),
}

app.GET("/users/:age/profile", profileHandler,
    blaze.WithConstraint("age", ageConstraint))

// Multiple constraints
app.GET("/products/:category/:sku", productHandler,
    blaze.WithConstraint("category", categoryConstraint),
    blaze.WithRegexConstraint("sku", `^[A-Z0-9-]+$`))
```

**Available Constraint Types:**
- `IntConstraint` - Validates integers
- `UUIDConstraint` - Validates UUID format
- `AlphaConstraint` - Validates alphabetic characters
- `RegexConstraint` - Custom regex pattern

## Route Groups

Route groups allow organizing routes with shared prefixes and middleware:

### Basic Route Groups

```go
// Create API v1 group
apiV1 := app.Group("/api/v1")

// Add routes to group with all HTTP methods
apiV1.GET("/users", listUsersHandler)
apiV1.POST("/users", createUserHandler)
apiV1.PUT("/users/:id", updateUserHandler)
apiV1.DELETE("/users/:id", deleteUserHandler)
apiV1.PATCH("/users/:id", patchUserHandler)
apiV1.CONNECT("/users/:id/session", connectUserHandler)
apiV1.TRACE("/users/:id/trace", traceUserHandler)

// Group-level ANY and Match routes
apiV1.ANY("/health", healthHandler)
apiV1.Match([]string{"GET", "POST"}, "/data", dataHandler)
```

### Group Middleware

Apply middleware to entire route groups:

```go
// Admin routes with authentication
admin := app.Group("/admin")
admin.Use(blaze.Auth(validateAdminToken))

admin.GET("/dashboard", adminDashboardHandler)
admin.POST("/users", adminCreateUserHandler)
admin.CONNECT("/maintenance", adminMaintenanceHandler)
admin.TRACE("/system", adminSystemTraceHandler)
admin.ANY("/system/*path", adminSystemHandler)

// API routes with rate limiting
api := app.Group("/api")
api.Use(blaze.RateLimitMiddleware(blaze.RateLimitOptions{
    MaxRequests: 100,
    Window:      time.Minute,
}))

api.GET("/data", dataHandler)
api.POST("/upload", uploadHandler)
api.CONNECT("/stream", streamHandler)
```

### Nested Route Groups

Create deeply nested route hierarchies:

```go
// Main API group
api := app.Group("/api")
api.Use(corsMiddleware)

// Version 1 nested group
v1 := api.Group("/v1")
v1.Use(rateLimitMiddleware)

v1.GET("/users", v1UsersHandler)
v1.POST("/users", v1CreateUserHandler)
v1.CONNECT("/users/:id/connect", v1ConnectUserHandler)
v1.ANY("/health", v1HealthHandler)

// Version 2 nested group with different middleware
v2 := api.Group("/v2")
v2.Use(authenticationMiddleware)
v2.Use(jsonMiddleware)

v2.GET("/users", v2UsersHandler)
v2.POST("/users", v2CreateUserHandler)
v2.TRACE("/debug", v2DebugHandler)
v2.Match([]string{"PUT", "PATCH"}, "/users/:id", v2UpdateUserHandler)

// Admin nested group
admin := v1.Group("/admin")
admin.Use(adminAuthMiddleware)
admin.ANY("/system/*path", adminSystemHandler)
```

## Route Options

Routes support various configuration options for advanced control:

### Named Routes

```go
app.GET("/users/:id", getUserHandler,
    blaze.WithName("user.show"))

app.POST("/users", createUserHandler,
    blaze.WithName("user.create"))

// Named routes with new HTTP methods
app.CONNECT("/tunnel/:target", tunnelHandler,
    blaze.WithName("tunnel.connect"))

app.TRACE("/debug/:session", debugHandler,
    blaze.WithName("debug.trace"))
```

### Route-Specific Middleware

```go
// Apply middleware to specific routes
app.GET("/protected", protectedHandler,
    blaze.WithMiddleware(authMiddleware, loggingMiddleware))

// Multiple middleware options with new HTTP methods
app.CONNECT("/secure-tunnel", secureTunnelHandler,
    blaze.WithName("secure.tunnel"),
    blaze.WithMiddleware(authMiddleware, encryptionMiddleware),
    blaze.WithIntConstraint("port"))

// Route with priority and tags
app.ANY("/api/priority", priorityHandler,
    blaze.WithPriority(10),
    blaze.WithTags("api", "priority", "production"),
    blaze.WithMiddleware(cacheMiddleware))
```

### Advanced Route Options

```go
// Route with custom priority
app.GET("/high-priority", handler,
    blaze.WithPriority(100))

// Route with tags for organization
app.GET("/api/data", dataHandler,
    blaze.WithTags("api", "data", "v1"))

// Combine multiple options
app.ANY("/api/users/:id", userHandler,
    blaze.WithName("user.any"),
    blaze.WithIntConstraint("id"),
    blaze.WithPriority(50),
    blaze.WithTags("api", "users"),
    blaze.WithMiddleware(authMiddleware, validationMiddleware))
```

## Query Parameters

Access query parameters through the context with type conversion:

```go
app.GET("/search", func(c *blaze.Context) error {
    // Get query parameter
    query := c.Query("q")
    
    // Get with default value
    page := c.QueryDefault("page", "1")
    
    // Get as integer
    limit, err := c.QueryInt("limit")
    if err != nil {
        limit = 10 // default value
    }
    
    // Get as integer with default
    offset := c.QueryIntDefault("offset", 0)
    
    return c.JSON(blaze.Map{
        "query":  query,
        "page":   page,
        "limit":  limit,
        "offset": offset,
    })
})

// Query parameters work with all HTTP methods
app.TRACE("/debug", func(c *blaze.Context) error {
    level := c.QueryDefault("level", "info")
    verbose := c.QueryIntDefault("verbose", 0)
    
    return c.JSON(blaze.Map{
        "trace_level": level,
        "verbose":     verbose,
        "method":      c.Method(),
    })
})
```

### Working with Query Args

For advanced query parameter handling:

```go
app.GET("/advanced-search", func(c *blaze.Context) error {
    // Get access to fasthttp query args
    args := c.QueryArgs()
    
    // Check if parameter exists
    hasFilter := args.Has("filter")
    
    // Get all values for a parameter (for arrays)
    var tags []string
    args.VisitAll(func(key, value []byte) {
        if string(key) == "tags" {
            tags = append(tags, string(value))
        }
    })
    
    return c.JSON(blaze.Map{
        "has_filter": hasFilter,
        "tags":       tags,
    })
})
```

## WebSocket Routes

Blaze provides first-class WebSocket support:

### Basic WebSocket Route

```go
app.WebSocket("/ws", func(ws *blaze.WebSocketConnection) {
    defer ws.Close()
    
    for {
        messageType, data, err := ws.ReadMessage()
        if err != nil {
            break
        }
        
        // Echo message back
        if err := ws.WriteMessage(messageType, data); err != nil {
            break
        }
    }
})
```

### WebSocket with Configuration

```go
wsConfig := &blaze.WebSocketConfig{
    ReadBufferSize:  4096,
    WriteBufferSize: 4096,
    ReadTimeout:     60 * time.Second,
    WriteTimeout:    10 * time.Second,
    MaxMessageSize:  1024 * 1024, // 1MB
}

app.WebSocketWithConfig("/ws/chat", chatHandler, wsConfig)
```

### WebSocket in Route Groups

```go
wsGroup := app.Group("/ws")
wsGroup.Use(authMiddleware) // Authenticate before upgrade

wsGroup.WebSocket("/chat", chatHandler)
wsGroup.WebSocket("/notifications", notificationHandler)
wsGroup.WebSocketWithConfig("/streaming", streamHandler, wsConfig)
```

## Static File Routes

Serve static files and directories:

### Basic Static Serving

```go
// Serve entire directory
app.Static("/static", "./public")

// Serve specific file
app.File("/favicon.ico", "./public/favicon.ico")
```

### Advanced Static Configuration

```go
staticConfig := blaze.DefaultStaticConfig("./public")
staticConfig.Index = "index.html"
staticConfig.Browse = false  // Disable directory browsing
staticConfig.Compress = true
staticConfig.CacheDuration = 24 * time.Hour
staticConfig.GenerateETag = true
staticConfig.ByteRange = true  // Enable range requests

app.StaticFS("/static", staticConfig)
```

## Router Configuration

The router can be configured with various options:

```go
// Enhanced router configuration
config := blaze.RouterConfig{
    CaseSensitive:          false, // Case-insensitive routes
    StrictSlash:            false, // /path and /path/ are the same
    RedirectSlash:          true,  // Redirect /path/ to /path
    UseEscapedPath:         false, // Use raw path
    HandleMethodNotAllowed: true,  // Handle 405 Method Not Allowed
    HandleOPTIONS:          true,  // Handle OPTIONS requests
}

// Create router with custom config
router := blaze.NewRouter(config)
```

## Advanced Features

### Route Matching Priority

Blaze uses a radix tree for efficient route matching with the following priority:

1. **Static segments** - Exact matches have highest priority
2. **Named parameters** - `:param` segments
3. **Wildcard parameters** - `*param` segments have lowest priority

```go
// Priority order (highest to lowest):
app.GET("/users/profile", handler1)     // 1. Static - exact match
app.GET("/users/:id", handler2)         // 2. Parameter
app.GET("/users/*path", handler3)       // 3. Wildcard

// Same priority rules apply to all HTTP methods
app.CONNECT("/servers/maintenance", maintenanceHandler) // 1. Static
app.CONNECT("/servers/:id", serverHandler)              // 2. Parameter
app.CONNECT("/servers/*path", genericHandler)           // 3. Wildcard
```

### Route Constraints Validation

The router automatically validates constraints before calling handlers:

```go
app.GET("/api/users/:id", userHandler,
    blaze.WithIntConstraint("id"))

app.CONNECT("/servers/:id", serverHandler,
    blaze.WithIntConstraint("id"))

// If :id is not a valid integer, returns 404 automatically
// Handler only called if constraint passes
```

### Method-Specific Route Handling

```go
// Handle method-specific logic
app.ANY("/api/resource/:id", func(c *blaze.Context) error {
    id, _ := c.ParamInt("id")
    
    switch c.Method() {
    case "GET":
        return c.JSON(blaze.Map{"action": "retrieve", "id": id})
    case "POST":
        return c.JSON(blaze.Map{"action": "create", "id": id})
    case "PUT":
        return c.JSON(blaze.Map{"action": "update", "id": id})
    case "DELETE":
        return c.JSON(blaze.Map{"action": "delete", "id": id})
    case "CONNECT":
        return c.Text(fmt.Sprintf("Connected to resource %d", id))
    case "TRACE":
        return c.JSON(blaze.Map{"action": "trace", "id": id, "path": c.Path()})
    default:
        return c.Status(405).JSON(blaze.Error("Method not allowed"))
    }
})
```

## Best Practices

### Route Organization

```go
// Organize routes by resource
func setupUserRoutes(app *blaze.App) {
    users := app.Group("/users")
    users.GET("/", listUsers)
    users.POST("/", createUser)
    users.GET("/:id", getUser, blaze.WithIntConstraint("id"))
    users.PUT("/:id", updateUser, blaze.WithIntConstraint("id"))
    users.DELETE("/:id", deleteUser, blaze.WithIntConstraint("id"))
    users.CONNECT("/:id/session", connectUser, blaze.WithIntConstraint("id"))
    users.TRACE("/:id/debug", traceUser, blaze.WithIntConstraint("id"))
    users.ANY("/:id/health", userHealth, blaze.WithIntConstraint("id"))
}

func setupAPIRoutes(app *blaze.App) {
    api := app.Group("/api/v1")
    api.Use(apiMiddleware)
    
    setupUserRoutes(api) // Nested organization
}
```

### Parameter Validation

```go
// Always validate parameters
app.ANY("/users/:id", func(c *blaze.Context) error {
    id, err := c.ParamInt("id")
    if err != nil {
        return c.Status(400).JSON(blaze.Error("Invalid user ID"))
    }
    
    if id <= 0 {
        return c.Status(400).JSON(blaze.Error("User ID must be positive"))
    }
    
    // Method-specific handling
    switch c.Method() {
    case "GET":
        return getUserByID(c, id)
    case "PUT":
        return updateUserByID(c, id)
    case "DELETE":
        return deleteUserByID(c, id)
    case "CONNECT":
        return connectUserByID(c, id)
    case "TRACE":
        return traceUserByID(c, id)
    default:
        return c.Status(405).JSON(blaze.Error("Method not supported"))
    }
})
```

### Route Naming Convention

```go
// Use consistent naming conventions for all HTTP methods
app.GET("/users", listUsers, blaze.WithName("users.index"))
app.POST("/users", createUser, blaze.WithName("users.store"))
app.GET("/users/:id", getUser, blaze.WithName("users.show"))
app.PUT("/users/:id", updateUser, blaze.WithName("users.update"))
app.DELETE("/users/:id", deleteUser, blaze.WithName("users.destroy"))
app.CONNECT("/users/:id/session", connectUser, blaze.WithName("users.connect"))
app.TRACE("/users/:id/trace", traceUser, blaze.WithName("users.trace"))
app.ANY("/users/:id/status", userStatus, blaze.WithName("users.status"))

// Use tags for better organization
app.GET("/api/users", apiListUsers,
    blaze.WithName("api.users.index"),
    blaze.WithTags("api", "users", "v1"))
```

### Error Handling for Different Methods

```go
// Comprehensive error handling
app.ANY("/api/resource/:id", func(c *blaze.Context) error {
    id, err := c.ParamInt("id")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error":  "Invalid resource ID",
            "method": c.Method(),
        })
    }
    
    // Check if resource exists
    if !resourceExists(id) {
        return c.Status(404).JSON(blaze.Map{
            "error":  "Resource not found",
            "id":     id,
            "method": c.Method(),
        })
    }
    
    // Handle different HTTP methods
    switch c.Method() {
    case "GET", "HEAD":
        return handleResourceRead(c, id)
    case "POST", "PUT", "PATCH":
        return handleResourceWrite(c, id)
    case "DELETE":
        return handleResourceDelete(c, id)
    case "CONNECT":
        return handleResourceConnect(c, id)
    case "TRACE":
        return handleResourceTrace(c, id)
    case "OPTIONS":
        c.SetHeader("Allow", "GET,POST,PUT,PATCH,DELETE,HEAD,OPTIONS,CONNECT,TRACE")
        return c.Status(200).Text("OK")
    default:
        c.SetHeader("Allow", "GET,POST,PUT,PATCH,DELETE,HEAD,OPTIONS,CONNECT,TRACE")
        return c.Status(405).JSON(blaze.Map{
            "error":   "Method not allowed",
            "method":  c.Method(),
            "allowed": []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "CONNECT", "TRACE"},
        })
    }
})
```

The Blaze routing system provides a comprehensive, high-performance foundation for building modern web applications with support for all HTTP methods, advanced route organization, parameter validation, and flexible middleware integration.