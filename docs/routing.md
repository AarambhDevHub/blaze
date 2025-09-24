# Routing

Blaze provides a powerful and flexible routing system built on a high-performance radix tree implementation. The router supports advanced features including route parameters, wildcards, constraints, middleware, and route groups.

### Basic Route Registration

Routes are registered using HTTP method functions on the main application instance:

```go
app := blaze.New()

// Basic GET route
app.GET("/", func(c *blaze.Context) error {
    return c.JSON(blaze.Map{"message": "Hello, World!"})
})

// POST route
app.POST("/users", func(c *blaze.Context) error {
    // Handle user creation
    return c.JSON(blaze.Map{"success": true})
})

// Other HTTP methods
app.PUT("/users/:id", updateUserHandler)
app.DELETE("/users/:id", deleteUserHandler)
app.PATCH("/users/:id", patchUserHandler)
app.HEAD("/health", healthCheckHandler)
app.OPTIONS("/api/*", corsHandler)
```

### Supported HTTP Methods

Blaze supports all standard HTTP methods :

- **GET** - Retrieve resources
- **POST** - Create new resources  
- **PUT** - Update/replace resources
- **DELETE** - Remove resources
- **PATCH** - Partially update resources
- **HEAD** - Get headers only
- **OPTIONS** - CORS preflight requests

### Route Parameters

#### Named Parameters

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
```

#### Parameter Helpers

The context provides helper methods for parameter conversion:

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

#### Wildcard Parameters

Use `*parameter` syntax for catch-all routes:

```go
// Catch-all for file serving
app.GET("/static/*filepath", func(c *blaze.Context) error {
    filepath := c.Param("filepath")
    return c.SendFile("./public/" + filepath)
})

// API versioning catch-all
app.GET("/api/v1/*path", func(c *blaze.Context) error {
    path := c.Param("path")
    return c.JSON(blaze.Map{"api_path": path})
})
```

### Route Constraints

Blaze provides a powerful constraint system to validate route parameters :

#### Built-in Constraints

```go
// Integer constraint
app.GET("/users/:id", userHandler,
    blaze.WithIntConstraint("id"),
)

// UUID constraint
app.GET("/items/:uuid", itemHandler,
    blaze.WithUUIDConstraint("uuid"),
)

// Custom regex constraint
app.GET("/products/:sku", productHandler,
    blaze.WithRegexConstraint("sku", `^[A-Z]{2}-\d{4}$`),
)
```

#### Custom Constraints

```go
// Define custom constraint
ageConstraint := blaze.RouteConstraint{
    Name:    "age",
    Type:    blaze.RegexConstraint,
    Pattern: regexp.MustCompile(`^\d{1,3}$`),
}

app.GET("/users/:age/profile", profileHandler,
    blaze.WithConstraint("age", ageConstraint),
)
```

### Route Groups

Route groups allow you to organize routes with shared prefixes and middleware :

#### Basic Route Groups

```go
// Create API v1 group
apiV1 := app.Group("/api/v1")

// Add routes to group
apiV1.GET("/users", listUsersHandler)
apiV1.POST("/users", createUserHandler)
apiV1.GET("/users/:id", getUserHandler)
apiV1.PUT("/users/:id", updateUserHandler)
apiV1.DELETE("/users/:id", deleteUserHandler)
```

#### Group Middleware

Apply middleware to entire route groups:

```go
// Admin routes with authentication
admin := app.Group("/admin")
admin.Use(blaze.Auth(validateAdminToken))

admin.GET("/dashboard", adminDashboardHandler)
admin.GET("/users", adminUsersHandler)
admin.POST("/settings", adminSettingsHandler)

// API routes with rate limiting
api := app.Group("/api")
api.Use(blaze.RateLimitMiddleware(blaze.RateLimitOptions{
    Requests: 100,
    Window:   time.Minute,
}))

api.GET("/data", dataHandler)
api.POST("/upload", uploadHandler)
```

#### Nested Route Groups

```go
// Main API group
api := app.Group("/api")
api.Use(corsMiddleware)

// Version 1 nested group
v1 := api.Group("/v1")
v1.Use(rateLimitMiddleware)

v1.GET("/users", v1UsersHandler)
v1.GET("/posts", v1PostsHandler)

// Version 2 nested group with different middleware
v2 := api.Group("/v2")
v2.Use(authenticationMiddleware)
v2.Use(jsonMiddleware)

v2.GET("/users", v2UsersHandler)
v2.POST("/users", v2CreateUserHandler)
```

### Route Options

Routes support various configuration options :

#### Named Routes

```go
app.GET("/users/:id", getUserHandler,
    blaze.WithName("user.show"),
)

app.POST("/users", createUserHandler,
    blaze.WithName("user.create"),
)
```

#### Route-Specific Middleware

```go
// Apply middleware to specific routes
app.GET("/protected", protectedHandler,
    blaze.WithMiddleware(authMiddleware, loggingMiddleware),
)

// Multiple middleware options
app.POST("/upload", uploadHandler,
    blaze.WithName("file.upload"),
    blaze.WithMiddleware(authMiddleware),
    blaze.WithConstraint("type", fileTypeConstraint),
)
```

### Query Parameters

Access query parameters through the context :

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
```

#### Working with Query Args

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

### WebSocket Routes

Blaze provides first-class WebSocket support :

#### Basic WebSocket Route

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

#### WebSocket with Configuration

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

#### WebSocket in Route Groups

```go
wsGroup := app.Group("/ws")
wsGroup.Use(authMiddleware) // Authenticate before upgrade

wsGroup.WebSocket("/chat", chatHandler)
wsGroup.WebSocket("/notifications", notificationHandler)
```

### Router Configuration

The router can be configured with various options :

```go
// Custom router configuration
config := &blaze.RouterConfig{
    CaseSensitive:         false, // Case-insensitive routes
    StrictSlash:           false, // /path and /path/ are the same
    RedirectSlash:         true,  // Redirect /path/ to /path
    UseEscapedPath:        false, // Use raw path
    HandleMethodNotAllowed: true, // Handle 405 Method Not Allowed
    HandleOPTIONS:         true,  // Handle OPTIONS requests
}

// Create app with custom router config
app := blaze.NewWithConfig(&blaze.Config{
    // ... other config options
})
```

### Route Matching Priority

Blaze uses a radix tree for efficient route matching with the following priority :

1. **Static segments** - Exact matches have highest priority
2. **Named parameters** - `:param` segments  
3. **Wildcard parameters** - `*param` segments have lowest priority

```go
// Priority order (highest to lowest):
app.GET("/users/profile", handler1)     // 1. Static - exact match
app.GET("/users/:id", handler2)         // 2. Parameter
app.GET("/users/*path", handler3)       // 3. Wildcard
```

### Advanced Router Features

#### Route Constraints Validation

The router automatically validates constraints before calling handlers:

```go
app.GET("/api/users/:id", userHandler,
    blaze.WithIntConstraint("id"),
)

// If :id is not a valid integer, returns 404 automatically
// Handler only called if constraint passes
```

#### Route Information Access

Access route information within handlers:

```go
app.GET("/users/:id", func(c *blaze.Context) error {
    // Get the matched route pattern
    pattern := c.GetUserValue("route_pattern")
    
    // Get route name if set
    routeName := c.GetUserValue("route_name")
    
    return c.JSON(blaze.Map{
        "pattern": pattern,
        "name":    routeName,
    })
})
```

### Best Practices

#### Route Organization

```go
// Organize routes by resource
func setupUserRoutes(app *blaze.App) {
    users := app.Group("/users")
    users.GET("/", listUsers)
    users.POST("/", createUser)
    users.GET("/:id", getUser, blaze.WithIntConstraint("id"))
    users.PUT("/:id", updateUser, blaze.WithIntConstraint("id"))
    users.DELETE("/:id", deleteUser, blaze.WithIntConstraint("id"))
}

func setupAPIRoutes(app *blaze.App) {
    api := app.Group("/api/v1")
    api.Use(apiMiddleware)
    
    setupUserRoutes(api) // Nested organization
}
```

#### Parameter Validation

```go
// Always validate parameters
app.GET("/users/:id", func(c *blaze.Context) error {
    id, err := c.ParamInt("id")
    if err != nil {
        return c.Status(400).JSON(blaze.Error("Invalid user ID"))
    }
    
    if id <= 0 {
        return c.Status(400).JSON(blaze.Error("User ID must be positive"))
    }
    
    // Continue with handler logic...
    return c.JSON(blaze.OK(user))
})
```

#### Route Naming

```go
// Use consistent naming conventions
app.GET("/users", listUsers, blaze.WithName("users.index"))
app.POST("/users", createUser, blaze.WithName("users.store"))
app.GET("/users/:id", getUser, blaze.WithName("users.show"))
app.PUT("/users/:id", updateUser, blaze.WithName("users.update"))
app.DELETE("/users/:id", deleteUser, blaze.WithName("users.destroy"))
```

The Blaze routing system provides a powerful, flexible, and high-performance foundation for building web applications with clean URL structures, parameter validation, and organized route management.