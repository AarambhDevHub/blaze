# Troubleshooting Guide

## Common Issues and Solutions

### Server Startup Issues

#### Port Already in Use
**Symptom:** Server fails to start with "bind: address already in use" error
```
panic: listen tcp :8080: bind: address already in use
```

**Solutions:**
1. **Find and kill the process using the port:**
   ```bash
   # Find process using port 8080
   lsof -i :8080
   sudo netstat -tulpn | grep :8080
   
   # Kill the process
   sudo kill -9 <PID>
   ```

2. **Change server port in configuration:**
   ```go
   config := blaze.DefaultConfig()
   config.Port = 8081 // Use different port
   app := blaze.NewWithConfig(config)
   ```

3. **Use dynamic port allocation:**
   ```go
   config := blaze.DefaultConfig()
   config.Port = 0 // System will assign available port
   app := blaze.NewWithConfig(config)
   ```

#### TLS Certificate Issues
**Symptom:** TLS server fails to start with certificate errors
```
failed to configure TLS: x509: certificate signed by unknown authority
```

**Solutions:**
1. **Generate self-signed certificate for development:**
   ```go
   app := blaze.New()
   app.EnableAutoTLS("localhost", "127.0.0.1")
   ```

2. **Verify certificate files exist and are readable:**
   ```bash
   # Check if certificate files exist
   ls -la ./certs/server.crt ./certs/server.key
   
   # Verify certificate validity
   openssl x509 -in ./certs/server.crt -text -noout
   ```

3. **Configure proper TLS settings:**
   ```go
   tlsConfig := blaze.DefaultTLSConfig()
   tlsConfig.CertFile = "/path/to/cert.pem"
   tlsConfig.KeyFile = "/path/to/key.pem"
   tlsConfig.InsecureSkipVerify = true // Only for development
   
   app.SetTLSConfig(tlsConfig)
   ```

### HTTP/2 Configuration Issues

#### HTTP/2 Not Working
**Symptom:** Server advertises HTTP/2 but clients connect with HTTP/1.1

**Debugging Steps:**
1. **Check HTTP/2 is enabled:**
   ```go
   config := blaze.DefaultConfig()
   config.EnableHTTP2 = true
   config.EnableTLS = true // HTTP/2 requires TLS
   app := blaze.NewWithConfig(config)
   ```

2. **Verify ALPN negotiation:**
   ```bash
   # Test HTTP/2 connection
   curl -v --http2 https://localhost:8443/
   openssl s_client -connect localhost:8443 -alpn h2
   ```

3. **Enable h2c for development (HTTP/2 without TLS):**
   ```go
   http2Config := blaze.DevelopmentHTTP2Config()
   http2Config.H2C = true
   app.SetHTTP2Config(http2Config)
   ```

#### Stream Errors
**Symptom:** HTTP/2 stream errors or connection resets

**Solutions:**
1. **Adjust HTTP/2 settings:**
   ```go
   http2Config := blaze.DefaultHTTP2Config()
   http2Config.MaxConcurrentStreams = 100
   http2Config.MaxReadFrameSize = 1048576 // 1MB
   app.SetHTTP2Config(http2Config)
   ```

2. **Monitor HTTP/2 health:**
   ```go
   app.GET("/health/http2", func(c *blaze.Context) error {
       serverInfo := app.GetServerInfo()
       return c.JSON(serverInfo.HTTP2)
   })
   ```

### Request Handling Issues

#### Large File Upload Failures
**Symptom:** File uploads fail or are truncated
```
failed to parse multipart form: request body too large
```

**Solutions:**
1. **Increase request body size limit:**
   ```go
   config := blaze.DefaultConfig()
   config.MaxRequestBodySize = 50 * 1024 * 1024 // 50MB
   app := blaze.NewWithConfig(config)
   ```

2. **Configure multipart handling:**
   ```go
   multipartConfig := &blaze.MultipartConfig{
       MaxMemory:    10 * 1024 * 1024, // 10MB in memory
       MaxFiles:     10,
       MaxFileSize:  20 * 1024 * 1024, // 20MB per file
       TempDir:      "/tmp/uploads",
       KeepInMemory: false,
   }
   
   form, err := c.MultipartFormWithConfig(multipartConfig)
   ```

3. **Handle upload progress:**
   ```go
   app.POST("/upload", func(c *blaze.Context) error {
       // Check content length
       if c.Request().Header.ContentLength() > maxSize {
           return c.Status(413).JSON(blaze.Map{
               "error": "File too large",
           })
       }
       
       file, err := c.FormFile("upload")
       if err != nil {
           return c.Status(400).JSON(blaze.Map{
               "error": "No file uploaded",
           })
       }
       
       return c.SaveUploadedFile(file, "./uploads/"+file.Filename)
   })
   ```

#### JSON Parsing Errors
**Symptom:** JSON bind operations fail with invalid syntax errors

**Solutions:**
1. **Validate Content-Type header:**
   ```go
   app.POST("/api/data", func(c *blaze.Context) error {
       if c.Header("Content-Type") != "application/json" {
           return c.Status(400).JSON(blaze.Map{
               "error": "Content-Type must be application/json",
           })
       }
       
       var data MyStruct
       if err := c.BindJSON(&data); err != nil {
           return c.Status(400).JSON(blaze.Map{
               "error": "Invalid JSON: " + err.Error(),
           })
       }
       
       return c.JSON(data)
   })
   ```

2. **Add request validation middleware:**
   ```go
   func ValidateJSON() blaze.MiddlewareFunc {
       return func(next blaze.HandlerFunc) blaze.HandlerFunc {
           return func(c *blaze.Context) error {
               if c.Method() == "POST" || c.Method() == "PUT" {
                   if !json.Valid(c.Body()) {
                       return c.Status(400).JSON(blaze.Map{
                           "error": "Invalid JSON format",
                       })
                   }
               }
               return next(c)
           }
       }
   }
   ```

### WebSocket Issues

#### WebSocket Connection Failures
**Symptom:** WebSocket upgrade fails or connections drop unexpectedly

**Solutions:**
1. **Check WebSocket headers:**
   ```go
   app.WebSocket("/ws", func(ws *blaze.WebSocketConn, c *blaze.Context) error {
       // Log connection headers for debugging
       log.Printf("WebSocket headers: %v", c.Request().Header)
       
       for {
           messageType, data, err := ws.ReadMessage()
           if err != nil {
               log.Printf("WebSocket read error: %v", err)
               break
           }
           
           if err := ws.WriteMessage(messageType, data); err != nil {
               log.Printf("WebSocket write error: %v", err)
               break
           }
       }
       
       return nil
   })
   ```

2. **Configure WebSocket settings:**
   ```go
   wsConfig := &blaze.WebSocketConfig{
       HandshakeTimeout: 10 * time.Second,
       ReadBufferSize:   4096,
       WriteBufferSize:  4096,
       CheckOrigin: func(c *blaze.Context) bool {
           // Allow all origins for development
           return true
       },
   }
   
   app.WebSocketWithConfig("/ws", handler, wsConfig)
   ```

3. **Handle connection lifecycle:**
   ```go
   func wsHandler(ws *blaze.WebSocketConn, c *blaze.Context) error {
       defer ws.Close()
       
       // Send ping periodically
       ticker := time.NewTicker(30 * time.Second)
       defer ticker.Stop()
       
       go func() {
           for {
               select {
               case <-ticker.C:
                   if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
                       return
                   }
               }
           }
       }()
       
       for {
           _, _, err := ws.ReadMessage()
           if err != nil {
               break
           }
       }
       
       return nil
   }
   ```

### Middleware Issues

#### Middleware Order Problems
**Symptom:** Middleware not executing in expected order or missing functionality

**Solutions:**
1. **Correct middleware registration order:**
   ```go
   app := blaze.New()
   
   // Global middleware (executed in reverse order)
   app.Use(blaze.Recovery())     // Last (outermost)
   app.Use(blaze.Logger())       // Second
   app.Use(CORSMiddleware())     // First (innermost)
   
   // Route-specific middleware
   app.GET("/api/protected", handler, 
       blaze.WithMiddleware(AuthMiddleware()),
       blaze.WithRateLimit(100))
   ```

2. **Debug middleware execution:**
   ```go
   func DebugMiddleware(name string) blaze.MiddlewareFunc {
       return func(next blaze.HandlerFunc) blaze.HandlerFunc {
           return func(c *blaze.Context) error {
               log.Printf("Before %s", name)
               err := next(c)
               log.Printf("After %s", name)
               return err
           }
       }
   }
   
   app.Use(DebugMiddleware("Recovery"))
   app.Use(DebugMiddleware("Logger"))
   app.Use(DebugMiddleware("CORS"))
   ```

#### Authentication Middleware Issues
**Symptom:** Protected routes accessible without authentication

**Solutions:**
1. **Verify token validation:**
   ```go
   func AuthMiddleware() blaze.MiddlewareFunc {
       return blaze.Auth(func(token string) bool {
           // Debug token validation
           log.Printf("Validating token: %s", token[:min(len(token), 10)]+"...")
           
           // Your validation logic
           return validateToken(token)
       })
   }
   ```

2. **Handle authentication errors gracefully:**
   ```go
   func AuthMiddleware() blaze.MiddlewareFunc {
       return func(next blaze.HandlerFunc) blaze.HandlerFunc {
           return func(c *blaze.Context) error {
               token := c.Header("Authorization")
               if token == "" {
                   return c.Status(401).JSON(blaze.Map{
                       "error": "Missing authorization header",
                       "code":  "AUTH_MISSING",
                   })
               }
               
               if !strings.HasPrefix(token, "Bearer ") {
                   return c.Status(401).JSON(blaze.Map{
                       "error": "Invalid authorization format",
                       "code":  "AUTH_FORMAT",
                   })
               }
               
               if !validateToken(token[7:]) {
                   return c.Status(401).JSON(blaze.Map{
                       "error": "Invalid or expired token",
                       "code":  "AUTH_INVALID",
                   })
               }
               
               return next(c)
           }
       }
   }
   ```

### Performance Issues

#### High Memory Usage
**Symptom:** Application consumes excessive memory or experiences memory leaks

**Debugging Steps:**
1. **Monitor memory usage:**
   ```go
   import (
       "runtime"
       "time"
   )
   
   func memoryStatsHandler(c *blaze.Context) error {
       var m runtime.MemStats
       runtime.ReadMemStats(&m)
       
       stats := blaze.Map{
           "alloc":         m.Alloc,
           "total_alloc":   m.TotalAlloc,
           "sys":           m.Sys,
           "num_gc":        m.NumGC,
           "goroutines":    runtime.NumGoroutine(),
       }
       
       return c.JSON(stats)
   }
   
   app.GET("/debug/memory", memoryStatsHandler)
   ```

2. **Configure garbage collection:**
   ```go
   import _ "net/http/pprof"
   
   // Enable pprof
   go func() {
       log.Println(http.ListenAndServe("localhost:6060", nil))
   }()
   
   // Force GC periodically
   ticker := time.NewTicker(5 * time.Minute)
   go func() {
       for range ticker.C {
           runtime.GC()
       }
   }()
   ```

3. **Optimize cache configuration:**
   ```go
   cacheConfig := &blaze.CacheOptions{
       MaxSize:       100 * 1024 * 1024, // 100MB
       MaxEntries:    10000,
       DefaultTTL:    10 * time.Minute,
       Algorithm:     blaze.LRU,
       EnableBackgroundCleanup: true,
       CleanupInterval: 1 * time.Minute,
   }
   ```

#### Slow Response Times
**Symptom:** High response latency or timeout errors

**Solutions:**
1. **Add request timeout middleware:**
   ```go
   app.Use(blaze.GracefulTimeout(30 * time.Second))
   ```

2. **Profile slow endpoints:**
   ```go
   func ProfileHandler() blaze.MiddlewareFunc {
       return func(next blaze.HandlerFunc) blaze.HandlerFunc {
           return func(c *blaze.Context) error {
               start := time.Now()
               err := next(c)
               duration := time.Since(start)
               
               if duration > 1*time.Second {
                   log.Printf("SLOW REQUEST: %s %s took %v", 
                       c.Method(), c.Path(), duration)
               }
               
               return err
           }
       }
   }
   ```

3. **Optimize database queries:**
   ```go
   func optimizeDBQueries(c *blaze.Context) error {
       // Use connection pooling
       db := getDBPool()
       defer db.Close()
       
       // Use context with timeout
       ctx, cancel := c.WithTimeout(5 * time.Second)
       defer cancel()
       
       // Execute query with context
       rows, err := db.QueryContext(ctx, query, args...)
       if err != nil {
           return err
       }
       defer rows.Close()
       
       return c.JSON(results)
   }
   ```

### Production Deployment Issues

#### Graceful Shutdown Problems
**Symptom:** Server doesn't shut down gracefully, causing connection drops

**Solutions:**
1. **Implement proper signal handling:**
   ```go
   func main() {
       app := blaze.New()
       
       // Configure routes
       setupRoutes(app)
       
       // Start server with graceful shutdown
       if err := app.ListenAndServeGraceful(); err != nil {
           log.Fatalf("Server failed: %v", err)
       }
   }
   ```

2. **Register cleanup tasks:**
   ```go
   app.RegisterGracefulTask(func(ctx context.Context) error {
       // Close database connections
       return db.Close()
   })
   
   app.RegisterGracefulTask(func(ctx context.Context) error {
       // Close cache connections
       return cache.Close()
   })
   ```

3. **Configure shutdown timeout:**
   ```go
   config := blaze.ProductionConfig()
   config.ReadTimeout = 30 * time.Second
   config.WriteTimeout = 30 * time.Second
   
   app := blaze.NewWithConfig(config)
   ```

#### Load Balancer Issues
**Symptom:** Load balancer health checks failing or uneven traffic distribution

**Solutions:**
1. **Implement health check endpoints:**
   ```go
   var startTime = time.Now()
   
   app.GET("/health", func(c *blaze.Context) error {
       health := blaze.Health("1.0.0", time.Since(startTime).String())
       return c.JSON(health)
   })
   
   app.GET("/health/ready", func(c *blaze.Context) error {
       // Check database connectivity
       if err := checkDatabase(); err != nil {
           return c.Status(503).JSON(blaze.Map{
               "status": "not ready",
               "error":  err.Error(),
           })
       }
       
       return c.JSON(blaze.Map{"status": "ready"})
   })
   ```

2. **Configure real IP extraction:**
   ```go
   app.Use(blaze.IPMiddleware()) // Extracts real client IP
   
   app.GET("/client-info", func(c *blaze.Context) error {
       return c.JSON(blaze.Map{
           "client_ip":   c.GetClientIP(),
           "real_ip":     c.GetRealIP(),
           "remote_addr": c.GetRemoteAddr(),
       })
   })
   ```

## Debugging Tools and Techniques

### Enable Debug Logging
```go
func DebugLogger() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Log request details
            log.Printf("[DEBUG] %s %s from %s", 
                c.Method(), c.Path(), c.GetClientIP())
            
            // Log headers
            c.Request().Header.VisitAll(func(key, value []byte) {
                log.Printf("[HEADER] %s: %s", key, value)
            })
            
            // Log body for POST/PUT
            if c.Method() == "POST" || c.Method() == "PUT" {
                log.Printf("[BODY] %s", c.BodyString())
            }
            
            return next(c)
        }
    }
}
```

### Request Tracing
```go
func RequestTracing() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Generate request ID
            requestID := generateRequestID()
            c.SetHeader("X-Request-ID", requestID)
            c.SetLocals("request_id", requestID)
            
            log.Printf("[%s] Started %s %s", requestID, c.Method(), c.Path())
            
            start := time.Now()
            err := next(c)
            duration := time.Since(start)
            
            log.Printf("[%s] Completed in %v with status %d", 
                requestID, duration, c.Response().StatusCode())
            
            return err
        }
    }
}
```

### Error Monitoring
```go
func ErrorMonitoring() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            err := next(c)
            if err != nil {
                // Log error with context
                log.Printf("[ERROR] %s %s: %v", c.Method(), c.Path(), err)
                
                // Send to monitoring service (e.g., Sentry)
                reportError(err, c)
                
                return c.Status(500).JSON(blaze.Map{
                    "error": "Internal Server Error",
                    "id":    c.Locals("request_id"),
                })
            }
            return nil
        }
    }
}
```

## Environment-Specific Issues

### Development Environment
1. **Enable hot reloading with Air:**
   ```bash
   # Install Air
   go install github.com/cosmtrek/air@latest
   
   # Create .air.toml configuration
   air init
   
   # Start with hot reload
   air
   ```

2. **Use development configuration:**
   ```go
   config := blaze.DevelopmentConfig()
   config.Development = true
   app := blaze.NewWithConfig(config)
   ```

### Testing Environment
```go
func setupTestApp() *blaze.App {
    config := blaze.DefaultConfig()
    config.Port = 0 // Use random port
    app := blaze.NewWithConfig(config)
    
    // Disable logging in tests
    app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return next // Skip logging middleware
    })
    
    return app
}

func TestHandler(t *testing.T) {
    app := setupTestApp()
    app.GET("/test", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{"status": "ok"})
    })
    
    // Test implementation here
}
```

### Production Environment
```go
func setupProductionApp() *blaze.App {
    config := blaze.ProductionConfig()
    app := blaze.NewWithConfig(config)
    
    // Production middleware
    app.Use(blaze.Recovery())
    app.Use(blaze.Logger())
    app.Use(SecurityHeaders())
    app.Use(RateLimiting())
    
    return app
}
```

## Quick Reference

### Common HTTP Status Codes
- `400` - Bad Request (client error)
- `401` - Unauthorized (authentication required)
- `403` - Forbidden (authorization failed)
- `404` - Not Found (resource doesn't exist)
- `429` - Too Many Requests (rate limited)
- `500` - Internal Server Error (server error)
- `503` - Service Unavailable (server overloaded/shutting down)

### Performance Benchmarking
```bash
# Test with wrk
wrk -t12 -c400 -d30s --latency http://localhost:8080/

# Test with Apache Bench
ab -n 10000 -c 100 http://localhost:8080/

# Test WebSocket connections
wscat -c ws://localhost:8080/ws
```

### Memory Profiling
```bash
# Get heap profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Get CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Get goroutine profile
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

This troubleshooting guide covers the most common issues you'll encounter when developing with the Blaze framework, providing practical solutions and debugging techniques for each scenario.