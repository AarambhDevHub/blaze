# Middleware

Middleware in the Blaze web framework provides a powerful way to intercept and process HTTP requests before they reach your route handlers. Middleware functions can modify requests, responses, perform authentication, logging, and handle cross-cutting concerns across your application.

## Table of Contents

- [Overview](#overview)
- [Built-in Middleware](#built-in-middleware)
- [Logging Middleware](#logging-middleware)
- [Error Handling Middleware](#error-handling-middleware)
- [Security Middleware](#security-middleware)
- [Performance Middleware](#performance-middleware)
- [HTTP/2 Middleware](#http2-middleware)
- [Graceful Shutdown Middleware](#graceful-shutdown-middleware)
- [Custom Middleware](#custom-middleware)
- [Middleware Application](#middleware-application)
- [Best Practices](#best-practices)

## Overview

Middleware functions in Blaze follow a simple signature and can be chained together to create powerful request processing pipelines. Each middleware function receives the next handler in the chain and returns a new handler function.

### Basic Middleware Signature

```go
type MiddlewareFunc func(HandlerFunc) HandlerFunc
```

All middleware functions follow this pattern where they wrap the next handler in the chain and can execute code before and after the handler runs.

## Built-in Middleware

### Logger Middleware

Comprehensive request/response logging with configurable options:

```go
// Simple logger
app.Use(blaze.Logger())

// Advanced logger with configuration
logConfig := blaze.DefaultLoggerMiddlewareConfig()
logConfig.SlowRequestThreshold = 2 * time.Second
logConfig.SkipPaths = []string{"/health", "/metrics"}
logConfig.LogHeaders = true
logConfig.ExcludeHeaders = []string{"Authorization", "Cookie"}
logConfig.LogRequestBody = false  // Be careful with sensitive data
logConfig.LogResponseBody = false // Can be expensive
logConfig.LogQueryParams = true

app.Use(blaze.LoggerMiddlewareWithConfig(logConfig))
```

**LoggerMiddlewareConfig Options:**
- `Logger` - Custom logger instance
- `SkipPaths` - Paths to skip logging
- `LogRequestBody` - Log request bodies (use carefully)
- `LogResponseBody` - Log response bodies (expensive)
- `LogQueryParams` - Log query parameters
- `LogHeaders` - Log request/response headers
- `ExcludeHeaders` - Headers to exclude from logs
- `CustomFields` - Add custom fields to logs
- `SlowRequestThreshold` - Log slow requests

### Recovery Middleware

Catches panics and returns proper error responses:

```go
// Basic recovery
app.Use(blaze.Recovery())

// Recovery with configuration
errorConfig := blaze.DefaultErrorHandlerConfig()
errorConfig.EnableStackTrace = true
errorConfig.IncludeStackTrace = true  // Include in response (dev only)

app.Use(blaze.RecoveryMiddleware(errorConfig))
```

### Authentication Middleware

Bearer token authentication for protected routes:

```go
// Simple token validator
tokenValidator := func(token string) bool {
    return token == "valid-secret-token"
}

// Global authentication
app.Use(blaze.Auth(tokenValidator))

// Route-specific authentication
app.GET("/protected", handler, blaze.WithMiddleware(blaze.Auth(tokenValidator)))
```

## Security Middleware

### CORS Middleware

Handle Cross-Origin Resource Sharing:

```go
corsOpts := blaze.CORSOptions{
    AllowOrigins:     []string{"https://example.com", "https://app.example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    ExposeHeaders:    []string{"X-Request-ID", "X-Total-Count"},
    AllowCredentials: true,
    MaxAge:           3600, // 1 hour
}

app.Use(blaze.CORS(corsOpts))
```

### CSRF Middleware

Protect against Cross-Site Request Forgery attacks:

```go
// Default CSRF protection
csrfOpts := blaze.DefaultCSRFOptions()
csrfOpts.Secret = []byte("your-32-byte-secret-key-here!!!")
csrfOpts.CookieSecure = true  // Enable for HTTPS
csrfOpts.CookieSameSite = "Strict"
csrfOpts.CheckReferer = true
csrfOpts.TrustedOrigins = []string{"https://example.com"}

app.Use(blaze.CSRF(csrfOpts))

// Access CSRF token in handlers
app.GET("/form", func(c *blaze.Context) error {
    token := blaze.CSRFToken(c)
    html := blaze.CSRFTokenHTML(c)  // As hidden input
    headerValue := blaze.CSRFTokenHeader(c)
    meta := blaze.CSRFMeta(c)  // As meta tag
    
    return c.HTML(html)
})

// Production CSRF configuration
csrfOpts := blaze.ProductionCSRFOptions([]byte("production-secret"))
app.Use(blaze.CSRF(csrfOpts))
```

**CSRF Configuration Options:**
- `Secret` - Secret key for token generation (32 bytes)
- `TokenLookup` - Where to find token (header/form/query)
- `CookieName` - Cookie name for CSRF token
- `CookieSecure` - HTTPS only (production)
- `CookieHTTPOnly` - Prevent JavaScript access
- `CookieSameSite` - SameSite cookie attribute
- `Expiration` - Token expiration duration
- `TokenLength` - Token length in bytes
- `TrustedOrigins` - Trusted origins for CORS
- `CheckReferer` - Validate Referer header
- `SingleUse` - Use tokens only once (more secure)

## Performance Middleware

### Cache Middleware

HTTP response caching with multiple strategies:

```go
// Default caching
app.Use(blaze.Cache(blaze.DefaultCacheOptions()))

// Custom cache configuration
cacheOpts := blaze.CacheOptions{
    DefaultTTL:            5 * time.Minute,
    MaxAge:                1 * time.Hour,
    MaxSize:               500 * 1024 * 1024,  // 500MB
    MaxEntries:            50000,
    Algorithm:             blaze.LRU,  // LRU, LFU, FIFO, Random
    VaryHeaders:           []string{"Accept-Encoding", "Accept-Language"},
    Public:                true,
    EnableCompression:     true,
    CompressionLevel:      9,
    CleanupInterval:       5 * time.Minute,
    EnableBackgroundCleanup: true,
}

app.Use(blaze.Cache(cacheOpts))

// Static file caching
app.Use(blaze.CacheStatic())

// API response caching
app.Use(blaze.CacheAPI(2 * time.Minute))

// Custom cache key generation
cacheOpts.KeyGenerator = func(c *blaze.Context) string {
    userID := c.Locals("user_id")
    return fmt.Sprintf("%s:%s:%v", c.Method(), c.Path(), userID)
}

// Custom cache predicate
cacheOpts.ShouldCache = func(c *blaze.Context) bool {
    return c.Response().StatusCode() == 200
}

app.Use(blaze.Cache(cacheOpts))

// Cache status endpoint
app.GET("/cache/status", blaze.CacheStatus)

// Cache invalidation
app.POST("/cache/invalidate", func(c *blaze.Context) error {
    pattern := c.Query("pattern")
    count := blaze.InvalidateCache(store, pattern)
    return c.JSON(blaze.Map{"invalidated": count})
})
```

**Cache Configuration Options:**
- `Store` - Cache storage backend
- `DefaultTTL` - Default time to live
- `MaxAge` - Cache-Control max-age
- `MaxSize` - Maximum cache size in bytes
- `MaxEntries` - Maximum number of entries
- `Algorithm` - Eviction algorithm (LRU, LFU, FIFO, Random)
- `VaryHeaders` - Headers to vary cache by
- `Public/Private` - Cache visibility
- `NoCache/NoStore` - Cache control directives
- `MustRevalidate` - Require revalidation
- `Immutable` - Mark as immutable
- `EnableCompression` - Compress cached responses
- `CompressionLevel` - Compression level (0-9)

### Compression Middleware

Response compression with multiple algorithms:

```go
// Default compression
app.Use(blaze.Compress())

// Compression with specific level
app.Use(blaze.CompressWithLevel(blaze.CompressionLevelBest))

// Custom compression configuration
compressionConfig := blaze.DefaultCompressionConfig()
compressionConfig.Level = blaze.CompressionLevelBest
compressionConfig.MinLength = 1024  // Only compress responses > 1KB
compressionConfig.IncludeContentTypes = []string{
    "text/html",
    "text/css",
    "text/javascript",
    "application/json",
    "application/xml",
}
compressionConfig.ExcludePaths = []string{"/api/stream"}
compressionConfig.EnableGzip = true
compressionConfig.EnableDeflate = true
compressionConfig.EnableBrotli = false

app.Use(blaze.CompressWithConfig(compressionConfig))

// Compress only specific types
app.Use(blaze.CompressTypes("text/html", "application/json"))

// Gzip only
app.Use(blaze.CompressGzipOnly())
```

**Compression Options:**
- `Level` - Compression level (0-9, -1 for default)
- `MinLength` - Minimum response size to compress
- `IncludeContentTypes` - Content types to compress
- `ExcludeContentTypes` - Content types to skip
- `EnableGzip` - Enable gzip compression
- `EnableDeflate` - Enable deflate compression
- `EnableBrotli` - Enable brotli compression
- `ExcludePaths` - Paths to skip compression
- `ExcludeExtensions` - File extensions to skip
- `EnableForHTTPS` - Enable for HTTPS (disabled by default)

### Body Limit Middleware

Restrict request body sizes:

```go
// Default body limit (4MB)
app.Use(blaze.BodyLimit(10 * 1024 * 1024))  // 10MB

// Convenience methods
app.Use(blaze.BodyLimitKB(500))   // 500KB
app.Use(blaze.BodyLimitMB(10))    // 10MB
app.Use(blaze.BodyLimitGB(1))     // 1GB

// Custom configuration
bodyConfig := blaze.DefaultBodyLimitConfig()
bodyConfig.MaxSize = 10 * 1024 * 1024
bodyConfig.ErrorMessage = "File too large"
bodyConfig.SkipPaths = []string{"/api/upload"}

app.Use(blaze.BodyLimitWithConfig(bodyConfig))

// Route-specific limits
app.Use(blaze.BodyLimitForRoute(50*1024*1024, "/api/upload", "/api/media"))

// Content-type specific limits
app.Use(blaze.BodyLimitByContentType(map[string]int64{
    "application/json": 1 * 1024 * 1024,      // 1MB for JSON
    "multipart/form-data": 50 * 1024 * 1024,  // 50MB for files
}))
```

### Rate Limiting Middleware

Prevent abuse by limiting request rates:

```go
rateLimitOpts := blaze.RateLimitOptions{
    MaxRequests:  100,              // 100 requests
    Window:       time.Minute,      // Per minute
    KeyGenerator: func(c *blaze.Context) string {
        return c.IP()  // Rate limit by IP
    },
    Handler: func(c *blaze.Context) error {
        return c.Status(429).JSON(blaze.Map{
            "error":       "Too many requests",
            "retry_after": 60,
        })
    },
}

app.Use(blaze.RateLimitMiddleware(rateLimitOpts))

// Per-user rate limiting
rateLimitOpts.KeyGenerator = func(c *blaze.Context) string {
    userID := c.Locals("user_id")
    if userID != nil {
        return fmt.Sprintf("user:%v", userID)
    }
    return c.IP()
}
```

### Request ID Middleware

Add unique identifiers to requests for tracing:

```go
// Add request ID middleware
app.Use(blaze.RequestIDMiddleware())

// Access request ID in handlers
app.GET("/", func(c *blaze.Context) error {
    requestID := blaze.GetRequestID(c)
    return c.JSON(blaze.Map{
        "request_id": requestID,
    })
})
```

## HTTP/2 Middleware

### HTTP/2 Info Middleware

Add HTTP/2 protocol information to responses:

```go
app.Use(blaze.HTTP2Info())

// Headers added:
// X-Protocol: HTTP/2.0 or HTTP/1.1
// X-HTTP2-Enabled: true or false
```

### HTTP/2 Security Middleware

Add HTTP/2-specific security headers:

```go
app.Use(blaze.HTTP2Security())

// Headers added:
// X-Content-Type-Options: nosniff
// X-Frame-Options: DENY
// X-XSS-Protection: 1; mode=block
// Strict-Transport-Security: max-age=31536000; includeSubDomains
```

### Stream Info Middleware

Add HTTP/2 stream debugging information:

```go
app.Use(blaze.StreamInfo())

// Headers added:
// X-Stream-ID: <stream_id>
// X-Stream-Priority: <priority>
```

### HTTP/2 Metrics Middleware

Collect HTTP/2-specific performance metrics:

```go
app.Use(blaze.HTTP2Metrics())
```

### HTTP/2 Compression Middleware

Enable HTTP/2-specific compression:

```go
app.Use(blaze.CompressHTTP2(6))  // Compression level 6
```

## Graceful Shutdown Middleware

### Shutdown Aware Middleware

Reject requests during graceful shutdown:

```go
app.Use(blaze.ShutdownAware())

// Returns 503 Service Unavailable during shutdown
```

### Graceful Timeout Middleware

Add request timeouts that respect shutdown context:

```go
app.Use(blaze.GracefulTimeout(30 * time.Second))

// Automatically cancels requests during shutdown
```

## Validation Middleware

### Validation Middleware

Enable struct validation for request binding:

```go
app.Use(blaze.ValidationMiddleware())

// Works with validation tags on structs
type User struct {
    Name  string `json:"name" validate:"required,min=2,max=100"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"gte=18,lte=100"`
}
```

## Multipart Middleware

### Multipart Form Middleware

Configure multipart form parsing:

```go
multipartConfig := blaze.DefaultMultipartConfig()
multipartConfig.MaxMemory = 10 * 1024 * 1024  // 10MB
multipartConfig.MaxFiles = 10
multipartConfig.AllowedExtensions = []string{".jpg", ".png", ".pdf"}
multipartConfig.AllowedMimeTypes = []string{
    "image/jpeg",
    "image/png",
    "application/pdf",
}

app.Use(blaze.MultipartMiddleware(multipartConfig))
```

## Custom Middleware

### Creating Custom Middleware

Follow the middleware signature pattern:

```go
// Simple custom middleware
func CustomHeaderMiddleware(headerName, headerValue string) blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Before handler execution
            c.SetHeader(headerName, headerValue)
            
            // Execute handler
            err := next(c)
            
            // After handler execution
            // Perform any cleanup
            
            return err
        }
    }
}

// Usage
app.Use(CustomHeaderMiddleware("X-API-Version", "v1.0.0"))
```

### Middleware with Configuration

Create configurable middleware using configuration structs:

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
            
            // Apply custom logic
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
                        "error": "Request timeout",
                    })
                }
            }
            
            return next(c)
        }
    }
}
```

### Authentication Middleware Example

Complete authentication middleware with JWT:

```go
func JWTAuthMiddleware(secret string) blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Get token from header
            auth := c.Header("Authorization")
            if auth == "" {
                return c.Status(401).JSON(blaze.Map{
                    "error": "Missing authorization header",
                })
            }
            
            // Extract Bearer token
            if len(auth) < 7 || auth[:7] != "Bearer " {
                return c.Status(401).JSON(blaze.Map{
                    "error": "Invalid authorization format",
                })
            }
            
            token := auth[7:]
            
            // Validate JWT token
            claims, err := validateJWT(token, secret)
            if err != nil {
                return c.Status(401).JSON(blaze.Map{
                    "error": "Invalid token",
                })
            }
            
            // Store user info in context
            c.SetLocals("user_id", claims.UserID)
            c.SetLocals("email", claims.Email)
            c.SetLocals("role", claims.Role)
            
            return next(c)
        }
    }
}
```

### Error Handling Middleware

Transform and log errors from handlers:

```go
func ErrorHandlingMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            err := next(c)
            if err != nil {
                // Log error
                c.LogError("Handler error", "error", err, "path", c.Path())
                
                // Check error type
                if httpErr, ok := err.(*blaze.HTTPError); ok {
                    return c.Status(httpErr.Code).JSON(blaze.Map{
                        "error": httpErr.Message,
                        "code":  httpErr.Code,
                    })
                }
                
                // Generic error response
                return c.Status(500).JSON(blaze.Map{
                    "error": "An internal error occurred",
                })
            }
            return nil
        }
    }
}
```

## Middleware Application

### Global Middleware

Apply middleware to all routes:

```go
app := blaze.New()

// Order matters - first added, first executed
app.Use(blaze.Recovery())
app.Use(blaze.LoggerMiddleware())
app.Use(blaze.RequestIDMiddleware())
app.Use(blaze.CORS(corsConfig))
app.Use(blaze.Compress())
```

### Route-Specific Middleware

Apply middleware to specific routes:

```go
app.GET("/admin", adminHandler, 
    blaze.WithMiddleware(blaze.Auth(tokenValidator)),
    blaze.WithMiddleware(AdminOnlyMiddleware()),
)
```

### Group Middleware

Apply middleware to route groups:

```go
api := app.Group("/api/v1")
api.Use(blaze.LoggerMiddleware())
api.Use(blaze.Auth(tokenValidator))
api.Use(blaze.RateLimitMiddleware(rateLimitOpts))

// All routes in this group have the middleware
api.GET("/users", getUsersHandler)
api.POST("/users", createUserHandler)
```

### Middleware Execution Order

Middleware executes in the order it was registered, creating an "onion" pattern:

```go
app.Use(middleware1)  // Executes first (outer)
app.Use(middleware2)  // Executes second (middle)
app.Use(middleware3)  // Executes third (inner)

// Execution flow:
// middleware1 (before) -> middleware2 (before) -> middleware3 (before) 
// -> handler 
// -> middleware3 (after) -> middleware2 (after) -> middleware1 (after)
```

## Best Practices

### Performance Considerations

- Place lightweight middleware (like request ID) early in the chain
- Place expensive middleware (like authentication) after basic validation
- Use caching middleware to reduce database load
- Enable compression middleware for bandwidth savings
- Use body limit middleware to prevent DoS attacks

```go
// Recommended order for performance
app.Use(blaze.Recovery())                      // 1. Catch panics first
app.Use(blaze.LoggerMiddleware())              // 2. Log all requests
app.Use(blaze.RequestIDMiddleware())           // 3. Add request ID
app.Use(blaze.BodyLimitMB(10))                 // 4. Validate body size
app.Use(blaze.CORS(corsConfig))                // 5. Handle CORS
app.Use(blaze.Compress())                      // 6. Compress responses
app.Use(blaze.Cache(cacheOpts))                // 7. Cache responses
app.Use(blaze.RateLimitMiddleware(rateOpts))   // 8. Rate limiting
app.Use(blaze.Auth(validator))                 // 9. Authentication last
```

### Security Best Practices

```go
// Production security stack
app.Use(blaze.Recovery())
app.Use(blaze.HTTP2Security())
app.Use(blaze.CORS(blaze.CORSOptions{
    AllowOrigins:     []string{os.Getenv("ALLOWED_ORIGIN")},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowCredentials: true,
}))
app.Use(blaze.CSRF(blaze.ProductionCSRFOptions([]byte("secret"))))
app.Use(blaze.RateLimitMiddleware(rateOpts))
app.Use(blaze.BodyLimitMB(10))
```

### Graceful Shutdown Integration

```go
func GracefulMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Check if shutting down
            if c.IsShuttingDown() {
                return c.Status(503).JSON(blaze.Map{
                    "error": "Service shutting down",
                })
            }
            
            // Use shutdown-aware context
            ctx := c.ShutdownContext()
            
            select {
            case <-ctx.Done():
                return c.Status(503).JSON(blaze.Map{
                    "error": "Service shutting down",
                })
            default:
                return next(c)
            }
        }
    }
}
```

The middleware system in Blaze provides a flexible and powerful way to handle cross-cutting concerns in your web application while maintaining clean separation of responsibilities and supporting advanced features like HTTP/2, graceful shutdown, compression, caching, and comprehensive security.