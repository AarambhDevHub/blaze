# Middleware

Middleware in the Blaze web framework provides a powerful way to intercept and process HTTP requests before they reach your route handlers. Middleware functions can modify requests, responses, perform authentication, logging, and handle cross-cutting concerns across your application.

## Overview

Middleware functions in Blaze follow a simple signature and can be chained together to create powerful request processing pipelines. Each middleware function receives the next handler in the chain and returns a new handler function.

## Basic Middleware Signature

```go
type MiddlewareFunc func(HandlerFunc) HandlerFunc
```

All middleware functions follow this pattern where they wrap the next handler in the chain and can execute code before and after the handler runs.

## Built-in Middleware

### Logger Middleware

The Logger middleware automatically logs all incoming requests with timing information.

```go
func main() {
    app := blaze.New()
    
    // Add logger middleware globally
    app.Use(blaze.Logger())
    
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{"message": "Hello World"})
    })
    
    app.ListenAndServe()
}
```

The logger outputs information including HTTP method, path, status code, and request duration.

### Recovery Middleware

The Recovery middleware catches panics that occur in route handlers and returns a proper error response instead of crashing the application.

```go
app.Use(blaze.Recovery())
```

When a panic occurs, it logs the panic information and returns a 500 Internal Server Error JSON response.

### Authentication Middleware

The Auth middleware provides Bearer token authentication for protected routes.

```go
// Define a token validator function
tokenValidator := func(token string) bool {
    // Implement your token validation logic
    return token == "valid-secret-token"
}

// Apply authentication middleware
app.Use(blaze.Auth(tokenValidator))

// Or apply to specific routes
app.GET("/protected", handler, blaze.WithMiddleware(blaze.Auth(tokenValidator)))
```

The middleware checks for the `Authorization` header with Bearer token format and validates the token using your provided validator function.

### CORS Middleware

The CORS middleware handles Cross-Origin Resource Sharing configuration.

```go
app.Use(blaze.CORS(blaze.CORSConfig{
    AllowOrigins:     []string{"*"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Content-Type", "Authorization"},
    ExposeHeaders:    []string{"X-Total-Count"},
    AllowCredentials: true,
    MaxAge:           86400, // 24 hours
}))
```

### Rate Limiting Middleware

The rate limiting middleware prevents abuse by limiting the number of requests from a single IP address.

```go
app.Use(blaze.RateLimit(blaze.RateLimitConfig{
    Rate:       100,              // requests per window
    Window:     time.Minute,      // time window
    BurstSize:  10,               // burst allowance
    SkipLimit:  func(c *blaze.Context) bool {
        return c.IP() == "127.0.0.1" // skip rate limiting for localhost
    },
}))
```

### CSRF Middleware

The CSRF middleware protects against Cross-Site Request Forgery attacks.

```go
app.Use(blaze.CSRF(blaze.CSRFConfig{
    TokenLength:    32,
    TokenLookup:    "header:X-CSRF-Token",
    CookieName:     "_csrf",
    CookieSecure:   true,
    CookieHTTPOnly: true,
    CookieSameSite: "Strict",
}))
```

### Request ID Middleware

The Request ID middleware adds a unique identifier to each request for tracing and logging.

```go
app.Use(blaze.RequestID())

// Access request ID in handlers
app.GET("/", func(c *blaze.Context) error {
    requestID := c.GetRequestID()
    return c.JSON(blaze.Map{"request_id": requestID})
})
```

### Cache Middleware

The cache middleware provides HTTP response caching with configurable strategies.

```go
app.Use(blaze.Cache(blaze.CacheConfig{
    Expiration:      time.Hour,
    CleanupInterval: 10 * time.Minute,
    MaxSize:         1000,
    CacheControl:    "public, max-age=3600",
}))
```

## Graceful Shutdown Middleware

### Shutdown Aware Middleware

The ShutdownAware middleware ensures requests are rejected during graceful shutdown.

```go
app.Use(blaze.ShutdownAware())
```

This middleware checks if the server is shutting down and returns a 503 Service Unavailable response for new requests.

### Graceful Timeout Middleware

The GracefulTimeout middleware adds request timeouts that respect the shutdown context.

```go
app.Use(blaze.GracefulTimeout(30 * time.Second))
```

This middleware ensures long-running requests are cancelled appropriately during shutdown.

## HTTP/2 Specific Middleware

### HTTP2 Info Middleware

The HTTP2Info middleware adds protocol information to response headers.

```go
app.Use(blaze.HTTP2Info())
```

### HTTP2 Security Middleware

The HTTP2Security middleware adds HTTP/2 specific security headers.

```go
app.Use(blaze.HTTP2Security())
```

This middleware adds security headers like `X-Content-Type-Options`, `X-Frame-Options`, `X-XSS-Protection`, and `Strict-Transport-Security` for HTTPS connections.

### Stream Info Middleware

The StreamInfo middleware adds HTTP/2 stream debugging information.

```go
app.Use(blaze.StreamInfo())
```

### HTTP2 Metrics Middleware

The HTTP2Metrics middleware collects HTTP/2 specific performance metrics.

```go
app.Use(blaze.HTTP2Metrics())
```

### HTTP/2 Compression Middleware

The CompressHTTP2 middleware enables HTTP/2 specific compression.

```go
app.Use(blaze.CompressHTTP2(6)) // compression level 6
```

## File Upload Middleware

### File Size Limit Middleware

The FileSizeLimitMiddleware restricts the maximum size of uploaded files.

```go
app.Use(blaze.FileSizeLimitMiddleware(50 << 20)) // 50MB limit
```

### File Type Middleware

The FileTypeMiddleware restricts uploads to specific file types.

```go
app.Use(blaze.FileTypeMiddleware(
    []string{".jpg", ".png", ".pdf"},                           // allowed extensions
    []string{"image/jpeg", "image/png", "application/pdf"},     // allowed MIME types
))
```

### Image Only Middleware

The ImageOnlyMiddleware restricts uploads to image files only.

```go
app.Use(blaze.ImageOnlyMiddleware())
```

### Document Only Middleware

The DocumentOnlyMiddleware restricts uploads to document files only.

```go
app.Use(blaze.DocumentOnlyMiddleware())
```

### Multipart Logging Middleware

The MultipartLoggingMiddleware logs details about multipart form uploads.

```go
app.Use(blaze.MultipartLoggingMiddleware())
```

## IP and Network Middleware

### IP Middleware

The IPMiddleware extracts and stores client IP information for easy access in handlers.

```go
app.Use(blaze.IPMiddleware())

// Access IP information in handlers
app.GET("/", func(c *blaze.Context) error {
    clientIP := c.GetClientIP()
    realIP := c.GetRealIP()
    remoteAddr := c.GetRemoteAddr()
    
    return c.JSON(blaze.Map{
        "client_ip":   clientIP,
        "real_ip":     realIP,
        "remote_addr": remoteAddr,
    })
})
```

This middleware automatically extracts client IP addresses from various headers like `X-Forwarded-For`, `X-Real-IP`, and `CF-Connecting-IP`.

## Custom Middleware

### Creating Custom Middleware

You can create custom middleware by following the middleware signature pattern.

```go
func CustomHeaderMiddleware(headerName, headerValue string) blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Add custom header before processing
            c.SetHeader(headerName, headerValue)
            
            // Process the request
            err := next(c)
            
            // Perform any cleanup after processing
            return err
        }
    }
}

// Usage
app.Use(CustomHeaderMiddleware("X-API-Version", "v1.0.0"))
```

### Middleware with Configuration

Create configurable middleware using configuration structs.

```go
type CustomConfig struct {
    Enabled   bool
    Value     string
    Timeout   time.Duration
}

func CustomMiddleware(config CustomConfig) blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            if !config.Enabled {
                return next(c)
            }
            
            // Apply custom logic based on configuration
            c.SetLocals("custom_value", config.Value)
            
            // Add timeout if specified
            if config.Timeout > 0 {
                ctx, cancel := c.WithTimeout(config.Timeout)
                defer cancel()
                
                done := make(chan error, 1)
                go func() {
                    done <- next(c)
                }()
                
                select {
                case err := <-done:
                    return err
                case <-ctx.Done():
                    return c.Status(408).JSON(blaze.Map{
                        "error": "Request timeout"
                    })
                }
            }
            
            return next(c)
        }
    }
}
```

## Applying Middleware

### Global Middleware

Apply middleware to all routes by using the `Use` method on the app instance.

```go
app := blaze.New()

// These middleware apply to all routes
app.Use(blaze.Logger())
app.Use(blaze.Recovery())
app.Use(blaze.CORS(corsConfig))
```

### Route-Specific Middleware

Apply middleware to specific routes using route options.

```go
app.GET("/admin", adminHandler, 
    blaze.WithMiddleware(blaze.Auth(tokenValidator)),
    blaze.WithMiddleware(adminOnlyMiddleware),
)
```

### Group Middleware

Apply middleware to route groups.

```go
api := app.Group("/api/v1")
api.Use(blaze.Auth(tokenValidator))
api.Use(blaze.RateLimit(rateLimitConfig))

// All routes in this group will have the middleware applied
api.GET("/users", getUsersHandler)
api.POST("/users", createUserHandler)
```

## Middleware Execution Order

Middleware executes in the order it was registered, creating an "onion" pattern where each middleware can execute code before and after the inner middleware and handlers.

```go
app.Use(middleware1) // Executes first (outer)
app.Use(middleware2) // Executes second (middle)
app.Use(middleware3) // Executes third (inner)

// Execution flow:
// middleware1 (before) -> middleware2 (before) -> middleware3 (before) 
// -> handler 
// -> middleware3 (after) -> middleware2 (after) -> middleware1 (after)
```

## Error Handling in Middleware

Middleware can handle and transform errors from downstream handlers.

```go
func ErrorHandlingMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            err := next(c)
            if err != nil {
                // Log error
                log.Printf("Handler error: %v", err)
                
                // Return custom error response
                return c.Status(500).JSON(blaze.Map{
                    "error": "An internal error occurred",
                    "timestamp": time.Now().Unix(),
                })
            }
            return nil
        }
    }
}
```

## Best Practices

### Performance Considerations

- Place lightweight middleware (like request ID generation) early in the chain
- Place expensive middleware (like authentication) later, after basic validation
- Use caching middleware to reduce database load
- Enable compression middleware to reduce bandwidth usage

### Security Best Practices

- Always use CSRF protection for state-changing operations
- Implement rate limiting to prevent abuse
- Use authentication middleware for protected endpoints
- Enable CORS middleware with restrictive policies
- Add security headers using HTTP2Security middleware

### Graceful Shutdown Integration

Middleware should respect graceful shutdown contexts.

```go
func GracefulMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Check if shutting down
            if c.IsShuttingDown() {
                return c.Status(503).JSON(blaze.Map{
                    "error": "Service shutting down"
                })
            }
            
            // Use shutdown-aware context for operations
            ctx := c.ShutdownContext()
            
            select {
            case <-ctx.Done():
                return c.Status(503).JSON(blaze.Map{
                    "error": "Service shutting down"
                })
            default:
                return next(c)
            }
        }
    }
}
```

The middleware system in Blaze provides a flexible and powerful way to handle cross-cutting concerns in your web application while maintaining clean separation of responsibilities and supporting advanced features like HTTP/2, graceful shutdown, and comprehensive file upload handling.