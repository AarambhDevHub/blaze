# Blaze 🔥

A blazing fast, lightweight web framework for Go inspired by Axum and Actix Web.

## 🚀 Performance

Blaze delivers **exceptional performance** with minimal overhead:

```
Requests/sec: 174,654.41
Transfer/sec:  80.95MB
Latency:       0.78ms avg
```

*Benchmarked with `wrk -c100 -d30s` on a simple endpoint.*

## ✨ Features

- **🔥 Blazing Fast**: Built on fasthttp for maximum performance (155K+ req/sec)
- **🪶 Lightweight**: Minimal memory footprint and zero-allocation hot paths
- **🛡️ Type Safe**: Full type safety with Go's type system
- **🔧 Middleware**: Composable middleware system with built-in common middlewares
- **📡 JSON First**: High-performance JSON serialization with json-iterator
- **🛣️ Flexible Routing**: Parameter extraction, route groups, and wildcards
- **⚡ Zero Copy**: Optimized for minimal allocations
- **🔌 Extensible**: Easy to extend with custom middleware and handlers

## 📦 Installation

```bash
go mod init your-project
go get github.com/AarambhDevHub/blaze
```

## 🚀 Quick Start

```go
package main

import (
    "log"
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
    app := blaze.New()

    app.Get("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "message": "Hello, Blaze!",
            "status":  "success",
        })
    })

    app.Get("/users/:id", func(c *blaze.Context) error {
        id := c.Param("id")
        return c.JSON(blaze.Map{
            "user_id": id,
        })
    })

    log.Println("🔥 Server starting on :3000")
    log.Fatal(app.Listen(":3000"))
}
```

## 📋 API Examples

### Basic Routing

```go
app := blaze.New()

// HTTP methods
app.Get("/users", getUsers)
app.Post("/users", createUser)
app.Put("/users/:id", updateUser)
app.Delete("/users/:id", deleteUser)

// Route parameters
app.Get("/users/:id/posts/:slug", getUserPost)
```

### Middleware

```go
// Global middleware
app.Use(blaze.LoggingMiddleware())
app.Use(blaze.CORSMiddleware())
app.Use(blaze.RecoveryMiddleware())

// Route-specific middleware
app.Get("/protected", authMiddleware(), protectedHandler)
```

### Route Groups

```go
// API versioning
api := app.Group("/api/v1")
api.Use(corsMiddleware())

api.Get("/users", getUsers)
api.Post("/users", createUser)

// Nested groups
admin := api.Group("/admin")
admin.Use(authMiddleware())
admin.Get("/stats", getAdminStats)
```

### JSON Handling

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
    Email string `json:"email"`
}

app.Post("/users", func(c *blaze.Context) error {
    var user User
    if err := c.BindJSON(&user); err != nil {
        return c.Status(400).JSON(blaze.Map{
            "error": "Invalid JSON",
        })
    }
    
    // Process user...
    
    return c.Status(201).JSON(user)
})
```

### Query Parameters

```go
app.Get("/search", func(c *blaze.Context) error {
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

## 🧪 Built-in Middlewares

```go
import "github.com/AarambhDevHub/blaze/pkg/blaze"

// Logging middleware
app.Use(blaze.LoggingMiddleware())

// CORS middleware
app.Use(blaze.CORSMiddleware())

// Recovery from panics
app.Use(blaze.RecoveryMiddleware())

// Request timing
app.Use(blaze.TimingMiddleware())

// Request ID generation
app.Use(blaze.RequestIDMiddleware())
```

## 📊 Performance Comparison

Framework | Req/sec | Latency | Memory | Notes
----------|---------|---------|--------|---------
**Blaze** | **175K** | **0.78ms** | **Low** | FastHTTP-based
Fiber | 165K | 0.60ms | Low | Express-like, FastHTTP
gnet | 200K+ | 0.5ms | Very Low | Event-driven
Gin | 50K | 10ms | Medium | Most popular
Echo | 40K | 15ms | Medium | Minimalist design
Chi | 35K | 20ms | Low | Lightweight router
Standard | 17K | 25ms | Medium | Go stdlib

## 🏗️ Project Structure

```
your-project/
├── main.go
├── go.mod
├── go.sum
├── handlers/
│   ├── users.go
│   └── auth.go
├── middleware/
│   └── custom.go
└── models/
    └── user.go
```

## 🔧 Configuration

```go
config := &blaze.Config{
    Host:               "0.0.0.0",
    Port:               8080,
    ReadTimeout:        15 * time.Second,
    WriteTimeout:       15 * time.Second,
    MaxRequestBodySize: 10 * 1024 * 1024, // 10MB
}

app := blaze.NewWithConfig(config)
```

## 🧪 Testing

```bash
# Run benchmarks
wrk -c100 -d30s http://localhost:3000/

# Load testing
ab -n 10000 -c 100 http://localhost:3000/api/users

# Memory profiling
go tool pprof http://localhost:3000/debug/pprof/heap
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 Examples

Check out the `/examples` directory for more comprehensive examples:

- **Basic Server**: Simple HTTP server
- **JSON API**: RESTful API with CRUD operations  
- **Middleware**: Custom middleware examples
- **Authentication**: JWT-based auth system

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👨‍💻 Author

**AarambhDevHub**
- GitHub: [@AarambhDevHub](https://github.com/AarambhDevHub)
- YouTube: [AarambhDevHub](https://youtube.com/@aarambhdevhub)

## 🌟 Show Your Support

Give a ⭐️ if this project helped you!

***

**Built with ❤️ for the Go community**
