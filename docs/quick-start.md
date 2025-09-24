Now I'll create the detailed quick-start guide based on the Blaze web framework information I found:

# Quick Start Guide

The Blaze framework is a lightweight, high-performance web framework for Go inspired by Axum and Actix Web. This guide will help you get started with building your first web application using Blaze.

## Installation

First, initialize your Go module and install Blaze:

```bash
# Initialize your project
go mod init your-project-name

# Install Blaze framework
go get github.com/AarambhDevHub/blaze
```

## Your First Application

Create a simple "Hello World" web server with routing:

```go
package main

import (
    "log"
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
    // Create a new Blaze application
    app := blaze.New()

    // Basic route
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "message": "Hello, Blaze!",
            "version": "1.0.0",
        })
    })

    // Route with parameter
    app.GET("/hello/:name", func(c *blaze.Context) error {
        name := c.Param("name")
        return c.JSON(blaze.Map{
            "message": "Hello, " + name + "!",
        })
    })

    // Start the server
    log.Fatal(app.ListenAndServe())
}
```

## Application Configuration

Blaze provides several configuration options for different environments:

### Default Configuration
```go
app := blaze.New() // Uses default config (localhost:8080)
```

### Development Configuration
```go
app := blaze.NewWithConfig(blaze.DevelopmentConfig())
// Runs on localhost:3000 by default
```

### Production Configuration
```go
app := blaze.NewWithConfig(blaze.ProductionConfig())
// Runs on 0.0.0.0:80 with enhanced security features
```

### Custom Configuration
```go
config := &blaze.Config{
    Host:               "127.0.0.1",
    Port:               8080,
    ReadTimeout:        30 * time.Second,
    WriteTimeout:       30 * time.Second,
    MaxRequestBodySize: 4 * 1024 * 1024, // 4MB
    EnableTLS:          false,
    EnableHTTP2:        false,
    Development:        true,
}

app := blaze.NewWithConfig(config)
```

## HTTP Methods and Routing

Blaze supports all standard HTTP methods with parameter extraction:

```go
app := blaze.New()

// HTTP Methods
app.GET("/users", getAllUsers)
app.POST("/users", createUser)
app.PUT("/users/:id", updateUser)
app.DELETE("/users/:id", deleteUser)
app.PATCH("/users/:id", patchUser)

// Route parameters
app.GET("/users/:id", func(c *blaze.Context) error {
    id := c.Param("id")
    // Convert to integer if needed
    userID, err := c.ParamInt("id")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "Invalid user ID"})
    }
    
    return c.JSON(blaze.Map{
        "user_id": userID,
        "path_param": id,
    })
})

// Query parameters
app.GET("/search", func(c *blaze.Context) error {
    query := c.Query("q")
    page := c.QueryIntDefault("page", 1)
    limit := c.QueryIntDefault("limit", 10)
    
    return c.JSON(blaze.Map{
        "query": query,
        "page":  page,
        "limit": limit,
    })
})
```

## Request and Response Handling

### JSON Responses
```go
app.POST("/api/data", func(c *blaze.Context) error {
    // JSON response
    return c.JSON(blaze.Map{
        "status": "success",
        "data":   "example",
    })
})

// JSON with status code
app.GET("/api/error", func(c *blaze.Context) error {
    return c.Status(404).JSON(blaze.Map{
        "error": "Not found",
    })
})
```

### Request Body Binding
```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

app.POST("/users", func(c *blaze.Context) error {
    var user User
    
    // Bind JSON request body
    if err := c.BindJSON(&user); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid JSON",
        })
    }
    
    // Process user...
    user.ID = 123 // Simulate saving
    
    return c.Status(201).JSON(user)
})
```

### Other Response Types
```go
// Text response
app.GET("/text", func(c *blaze.Context) error {
    return c.Text("Hello, World!")
})

// HTML response
app.GET("/html", func(c *blaze.Context) error {
    return c.HTML("<h1>Welcome to Blaze!</h1>")
})

// Redirect
app.GET("/redirect", func(c *blaze.Context) error {
    c.Redirect("/new-location")
    return nil
})
```

## Middleware

Add middleware to your application for logging, recovery, and custom functionality:

```go
app := blaze.New()

// Built-in middleware
app.Use(blaze.Logger())    // Request logging
app.Use(blaze.Recovery())  // Panic recovery

// Custom middleware
app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        // Pre-processing
        start := time.Now()
        
        // Process request
        err := next(c)
        
        // Post-processing
        duration := time.Since(start)
        log.Printf("Request took %v", duration)
        
        return err
    }
})
```

## Route Groups

Organize related routes with shared middleware:

```go
app := blaze.New()

// API group with shared prefix and middleware
api := app.Group("/api/v1")
api.Use(authMiddleware) // Apply to all routes in this group

api.GET("/users", getAllUsers)
api.POST("/users", createUser)
api.GET("/users/:id", getUser)

// Admin group
admin := app.Group("/admin")
admin.Use(adminAuthMiddleware)

admin.GET("/dashboard", adminDashboard)
admin.POST("/settings", updateSettings)
```

## Graceful Shutdown

Handle graceful shutdown with signal handling:

```go
func main() {
    app := blaze.New()
    
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{"status": "ok"})
    })
    
    // Start with graceful shutdown (handles SIGINT, SIGTERM)
    log.Fatal(app.ListenAndServeGraceful())
}
```

## File Handling

Serve static files and handle file uploads:

```go
// Serve static files
app.GET("/files/*filepath", func(c *blaze.Context) error {
    filepath := c.Param("filepath")
    return c.ServeFile("./static/" + filepath)
})

// File upload
app.POST("/upload", func(c *blaze.Context) error {
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{"error": "No file uploaded"})
    }
    
    // Save file
    savedPath, err := c.SaveUploadedFileToDir(file, "./uploads")
    if err != nil {
        return c.Status(500).JSON(blaze.Map{"error": "Failed to save file"})
    }
    
    return c.JSON(blaze.Map{
        "message":  "File uploaded successfully",
        "filename": file.Filename,
        "path":     savedPath,
    })
})
```

## Error Handling

Implement robust error handling:

```go
app.GET("/api/users/:id", func(c *blaze.Context) error {
    id, err := c.ParamInt("id")
    if err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid user ID format",
        })
    }
    
    user, err := getUserFromDB(id)
    if err != nil {
        return c.Status(404).JSON(blaze.Map{
            "error": "User not found",
        })
    }
    
    return c.JSON(user)
})
```

## Running Your Application

Start your server with different approaches:

```go
func main() {
    app := blaze.New()
    
    // Add your routes...
    
    // Option 1: Basic server start
    log.Fatal(app.ListenAndServe())
    
    // Option 2: Graceful shutdown with signal handling
    log.Fatal(app.ListenAndServeGraceful())
    
    // Option 3: Custom address
    app.config.Host = "0.0.0.0"
    app.config.Port = 3000
    log.Fatal(app.ListenAndServe())
}
```

## Next Steps

Once you have your basic application running, explore these advanced features:

- **WebSocket Support**: Real-time communication with `app.WebSocket()`
- **TLS/HTTPS**: Secure connections with `app.EnableAutoTLS()`
- **HTTP/2**: Enhanced performance with HTTP/2 support
- **Middleware**: CORS, CSRF protection, rate limiting, caching
- **File Upload**: Multipart form handling with validation
- **Database Integration**: Connect to your preferred database
- **Authentication**: Implement JWT or session-based auth

This quick start guide covers the essential concepts to get you productive with Blaze. The framework provides a clean, efficient API for building modern web applications in Go.