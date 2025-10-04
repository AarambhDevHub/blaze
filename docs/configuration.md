# Configuration

Blaze provides flexible configuration options to customize your web application for different environments and use cases. This guide covers all configuration aspects including server settings, TLS/HTTPS, HTTP/2, middleware, and environment-specific configurations.

## Table of Contents

- [Basic Configuration](#basic-configuration)
- [Configuration Presets](#configuration-presets)
- [Server Configuration](#server-configuration)
- [TLS/HTTPS Configuration](#tlshttps-configuration)
- [HTTP/2 Configuration](#http2-configuration)
- [Middleware Configuration](#middleware-configuration)
- [Router Configuration](#router-configuration)
- [Environment Variables](#environment-variables)
- [Configuration Examples](#configuration-examples)

## Basic Configuration

### Config Structure

The main configuration is handled through the `Config` struct:

```go
type Config struct {
    Host               string        // Server bind address
    Port               int           // HTTP port
    TLSPort            int           // HTTPS port
    ReadTimeout        time.Duration // Read timeout
    WriteTimeout       time.Duration // Write timeout
    MaxRequestBodySize int           // Maximum request body size
    Concurrency        int           // Maximum concurrent connections
    
    // Protocol configuration
    EnableHTTP2       bool // Enable HTTP/2 support
    EnableTLS         bool // Enable TLS/HTTPS
    RedirectHTTPToTLS bool // Redirect HTTP to HTTPS
    
    // Development settings
    Development bool // Development mode
}
```

### Creating Applications with Configuration

```go
package main

import (
    "time"
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
    // Method 1: Default configuration
    app := blaze.New()
    
    // Method 2: Custom configuration
    config := &blaze.Config{
        Host:               "0.0.0.0",
        Port:               3000,
        ReadTimeout:        30 * time.Second,
        WriteTimeout:       30 * time.Second,
        MaxRequestBodySize: 10 * 1024 * 1024, // 10MB
        Concurrency:        1000,
        Development:        true,
    }
    app := blaze.NewWithConfig(config)
    
    app.ListenAndServe()
}
```

## Configuration Presets

Blaze provides three pre-configured setups for common scenarios:

### Default Configuration

```go
app := blaze.New() // Uses DefaultConfig()

// Equivalent to:
config := blaze.DefaultConfig()
app := blaze.NewWithConfig(config)
```

**Default settings:**
- Host: `127.0.0.1`
- Port: `8080`
- TLS Port: `8443`
- Read/Write Timeout: `10 seconds`
- Max Request Body Size: `4MB`
- Concurrency: `256,000`
- HTTP/2: `disabled`
- TLS: `disabled`

### Development Configuration

```go
config := blaze.DevelopmentConfig()
app := blaze.NewWithConfig(config)
```

**Development settings:**
- Host: `127.0.0.1`
- Port: `3000`
- TLS Port: `3443`
- Read/Write Timeout: `10 seconds`
- Max Request Body Size: `4MB`
- Development mode: `enabled`
- Auto TLS available for testing

### Production Configuration

```go
config := blaze.ProductionConfig()
app := blaze.NewWithConfig(config)
```

**Production settings:**
- Host: `0.0.0.0` (all interfaces)
- Port: `80`
- TLS Port: `443`
- Read/Write Timeout: `30 seconds`
- Max Request Body Size: `10MB`
- Concurrency: `256,000`
- HTTP/2: `enabled`
- TLS: `enabled`
- HTTP to HTTPS redirect: `enabled`

## Server Configuration

### Basic Server Settings

| Setting | Type | Description | Default |
|---------|------|-------------|---------|
| `Host` | `string` | Server bind address | `127.0.0.1` |
| `Port` | `int` | HTTP port | `8080` |
| `TLSPort` | `int` | HTTPS port | `8443` |
| `ReadTimeout` | `time.Duration` | Read timeout | `10s` |
| `WriteTimeout` | `time.Duration` | Write timeout | `10s` |
| `MaxRequestBodySize` | `int` | Maximum request body size in bytes | `4194304` (4MB) |
| `Concurrency` | `int` | Maximum concurrent connections | `262144` |

### Example Server Configuration

```go
config := &blaze.Config{
    Host:               "0.0.0.0",          // Listen on all interfaces
    Port:               8080,               // HTTP port
    TLSPort:            8443,               // HTTPS port
    ReadTimeout:        15 * time.Second,   // 15 second read timeout
    WriteTimeout:       15 * time.Second,   // 15 second write timeout
    MaxRequestBodySize: 20 * 1024 * 1024,  // 20MB max body size
    Concurrency:        10000,              // 10k concurrent connections
    Development:        false,              // Production mode
}

app := blaze.NewWithConfig(config)
```

### Timeout Configuration

Configure timeouts to prevent resource exhaustion:

```go
config := &blaze.Config{
    ReadTimeout:  30 * time.Second,  // Time to read request headers and body
    WriteTimeout: 30 * time.Second,  // Time to write response
}

// For long-running requests, use middleware timeouts instead
app.Use(blaze.GracefulTimeout(60 * time.Second))
```

## TLS/HTTPS Configuration

### TLS Configuration Structure

```go
type TLSConfig struct {
    // Certificate files
    CertFile string
    KeyFile  string
    
    // Auto-generate certificates
    AutoTLS                 bool
    TLSCacheDir             string
    Domains                 []string
    Organization            string
    
    // TLS settings
    MinVersion              uint16
    MaxVersion              uint16
    CipherSuites            []uint16
    
    // Client authentication
    ClientAuth              tls.ClientAuthType
    ClientCAs               *x509.CertPool
    
    // HTTP/2 support
    NextProtos              []string
    
    // Certificate settings
    CertValidityDuration    time.Duration
    OCSPStapling            bool
    SessionTicketsDisabled  bool
    CurvePreferences        []tls.CurveID
    Renegotiation           tls.RenegotiationSupport
    InsecureSkipVerify      bool
}
```

### Basic TLS Setup

```go
// Method 1: Using certificate files
app := blaze.New()
tlsConfig := &blaze.TLSConfig{
    CertFile: "server.crt",
    KeyFile:  "server.key",
}
app.SetTLSConfig(tlsConfig)

// Method 2: Auto TLS for development
app.EnableAutoTLS("localhost", "127.0.0.1", "myapp.local")
```

### Advanced TLS Configuration

```go
tlsConfig := &blaze.TLSConfig{
    CertFile:      "server.crt",
    KeyFile:       "server.key",
    MinVersion:    tls.VersionTLS12,
    MaxVersion:    tls.VersionTLS13,
    CipherSuites:  []uint16{
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
    },
    NextProtos:    []string{"h2", "http/1.1"}, // HTTP/2 support
    ClientAuth:    tls.NoClientCert,
}

app.SetTLSConfig(tlsConfig)
```

### Development TLS (Auto-generated certificates)

```go
// Quick setup for development
app := blaze.NewWithConfig(blaze.DevelopmentConfig())
app.EnableAutoTLS("localhost", "127.0.0.1")

// Or manually configure
tlsConfig := blaze.DevelopmentTLSConfig()
tlsConfig.Domains = []string{"myapp.local", "api.myapp.local"}
app.SetTLSConfig(tlsConfig)
```

### HTTP to HTTPS Redirect

```go
config := &blaze.Config{
    EnableTLS:         true,
    RedirectHTTPToTLS: true,  // Automatically redirect HTTP to HTTPS
    Port:              80,    // HTTP port
    TLSPort:           443,   // HTTPS port
}
```

## HTTP/2 Configuration

### HTTP/2 Configuration Structure

```go
type HTTP2Config struct {
    Enabled                      bool
    H2C                          bool     // HTTP/2 over cleartext
    MaxConcurrentStreams         uint32
    MaxUploadBufferPerStream     int32
    MaxUploadBufferPerConnection int32
    EnablePush                   bool
    IdleTimeout                  time.Duration
    ReadTimeout                  time.Duration
    WriteTimeout                 time.Duration
    MaxDecoderHeaderTableSize    uint32
    MaxEncoderHeaderTableSize    uint32
    MaxReadFrameSize             uint32
    PermitProhibitedCipherSuites bool
}
```

### Basic HTTP/2 Setup

```go
// Enable HTTP/2 with TLS
config := &blaze.Config{
    EnableHTTP2: true,
    EnableTLS:   true,
}

app := blaze.NewWithConfig(config)
```

### Advanced HTTP/2 Configuration

```go
http2Config := &blaze.HTTP2Config{
    Enabled:                      true,
    H2C:                          false,  // Require TLS
    MaxConcurrentStreams:         1000,
    MaxUploadBufferPerStream:     1048576, // 1MB
    MaxUploadBufferPerConnection: 1048576, // 1MB
    EnablePush:                   true,
    IdleTimeout:                  300 * time.Second,
    ReadTimeout:                  30 * time.Second,
    WriteTimeout:                 30 * time.Second,
}

config := blaze.ProductionConfig()
app := blaze.NewWithConfig(config)
app.SetHTTP2Config(http2Config)
```

### HTTP/2 over Cleartext (Development)

```go
// For development/testing without TLS
http2Config := &blaze.HTTP2Config{
    Enabled: true,
    H2C:     true,  // Allow HTTP/2 over cleartext
}

app := blaze.NewWithConfig(blaze.DevelopmentConfig())
app.SetHTTP2Config(http2Config)
```

## Middleware Configuration

### Logger Middleware Configuration

```go
type LoggerMiddlewareConfig struct {
    Logger                *Logger
    SkipPaths             []string
    LogRequestBody        bool
    LogResponseBody       bool
    LogQueryParams        bool
    LogHeaders            bool
    ExcludeHeaders        []string
    CustomFields          func(*Context) map[string]interface{}
    SlowRequestThreshold  time.Duration
}

// Example usage
logConfig := blaze.DefaultLoggerMiddlewareConfig()
logConfig.SlowRequestThreshold = 2 * time.Second
logConfig.SkipPaths = []string{"/health", "/metrics"}
logConfig.LogHeaders = true
logConfig.ExcludeHeaders = []string{"Authorization", "Cookie"}

app.Use(blaze.LoggerMiddlewareWithConfig(logConfig))
```

### CORS Configuration

```go
type CORSOptions struct {
    AllowOrigins     []string
    AllowMethods     []string
    AllowHeaders     []string
    ExposeHeaders    []string
    AllowCredentials bool
    MaxAge           int
}

// Example usage
corsOpts := blaze.CORSOptions{
    AllowOrigins:     []string{"https://example.com", "https://app.example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
    ExposeHeaders:    []string{"X-Request-ID"},
    AllowCredentials: true,
    MaxAge:           3600,
}

app.Use(blaze.CORS(corsOpts))
```

### CSRF Configuration

```go
type CSRFOptions struct {
    Secret            []byte
    TokenLookup       string
    ContextKey        string
    CookieName        string
    CookiePath        string
    CookieDomain      string
    CookieSecure      bool
    CookieHTTPOnly    bool
    CookieSameSite    string
    CookieMaxAge      int
    Expiration        time.Duration
    TokenLength       int
    Skipper           func(*Context) bool
    ErrorHandler      func(*Context, error) error
    TrustedOrigins    []string
    CheckReferer      bool
    SingleUse         bool
}

// Example usage
csrfOpts := blaze.DefaultCSRFOptions()
csrfOpts.Secret = []byte("your-32-byte-secret-key-here!!!")
csrfOpts.CookieSecure = true  // Enable in production with HTTPS
csrfOpts.CookieSameSite = "Strict"
csrfOpts.TrustedOrigins = []string{"https://example.com"}

app.Use(blaze.CSRF(csrfOpts))
```

### Cache Configuration

```go
type CacheOptions struct {
    Store                   CacheStore
    DefaultTTL              time.Duration
    MaxAge                  time.Duration
    MaxSize                 int64
    MaxEntries              int
    Algorithm               EvictionAlgorithm
    Skipper                 func(*Context) bool
    KeyGenerator            func(*Context) string
    ShouldCache             func(*Context) bool
    VaryHeaders             []string
    Public                  bool
    Private                 bool
    NoCache                 bool
    NoStore                 bool
    MustRevalidate          bool
    ProxyRevalidate         bool
    Immutable               bool
    EnableCompression       bool
    CompressionLevel        int
    CleanupInterval         time.Duration
    EnableBackgroundCleanup bool
    WarmupURLs              []string
    EnableHeaders           bool
    HeaderPrefix            string
}

// Example usage
cacheOpts := blaze.DefaultCacheOptions()
cacheOpts.DefaultTTL = 10 * time.Minute
cacheOpts.MaxSize = 500 * 1024 * 1024  // 500MB
cacheOpts.MaxEntries = 50000
cacheOpts.Public = true
cacheOpts.VaryHeaders = []string{"Accept-Encoding", "Accept-Language"}

app.Use(blaze.Cache(cacheOpts))

// Or use preset configurations
app.Use(blaze.CacheStatic())  // For static files
app.Use(blaze.CacheAPI(2 * time.Minute))  // For API endpoints
```

### Compression Configuration

```go
type CompressionConfig struct {
    Level                 CompressionLevel
    MinLength             int
    IncludeContentTypes   []string
    ExcludeContentTypes   []string
    EnableGzip            bool
    EnableDeflate         bool
    EnableBrotli          bool
    ExcludePaths          []string
    ExcludeExtensions     []string
    EnableForHTTPS        bool
}

// Example usage
compressionConfig := blaze.DefaultCompressionConfig()
compressionConfig.Level = blaze.CompressionLevelBest
compressionConfig.MinLength = 1024  // Only compress responses > 1KB
compressionConfig.IncludeContentTypes = []string{
    "text/html",
    "text/css",
    "application/javascript",
    "application/json",
}
compressionConfig.ExcludePaths = []string{"/api/stream"}

app.Use(blaze.CompressWithConfig(compressionConfig))
```

### Body Limit Configuration

```go
type BodyLimitConfig struct {
    MaxSize          int64
    ErrorMessage     string
    SkipPaths        []string
    SkipContentTypes []string
}

// Example usage
bodyLimitConfig := blaze.DefaultBodyLimitConfig()
bodyLimitConfig.MaxSize = 10 * 1024 * 1024  // 10MB
bodyLimitConfig.SkipPaths = []string{"/api/upload"}

app.Use(blaze.BodyLimitWithConfig(bodyLimitConfig))

// Or use convenience methods
app.Use(blaze.BodyLimitMB(5))  // 5MB limit
app.Use(blaze.BodyLimitKB(500))  // 500KB limit
```

### Rate Limiting Configuration

```go
type RateLimitOptions struct {
    MaxRequests      int
    Window           time.Duration
    KeyGenerator     func(*Context) string
    Handler          func(*Context) error
    SkipSuccessful   bool
    SkipFailed       bool
}

// Example usage
rateLimitOpts := blaze.RateLimitOptions{
    MaxRequests: 100,
    Window:      time.Minute,
    KeyGenerator: func(c *blaze.Context) string {
        return c.IP()  // Rate limit by IP
    },
    Handler: func(c *blaze.Context) error {
        return c.Status(429).JSON(blaze.Map{
            "error": "Too many requests",
        })
    },
}

app.Use(blaze.RateLimitMiddleware(rateLimitOpts))
```

### Error Handling Configuration

```go
type ErrorHandlerConfig struct {
    EnableStackTrace    bool
    IncludeStackTrace   bool
    CustomErrorHandler  func(*Context, error) error
    Logger              *Logger
}

// Example usage
errorConfig := blaze.DefaultErrorHandlerConfig()
errorConfig.EnableStackTrace = true
errorConfig.CustomErrorHandler = func(c *blaze.Context, err error) error {
    // Custom error handling logic
    return c.Status(500).JSON(blaze.Map{
        "error": err.Error(),
        "timestamp": time.Now(),
    })
}

app.UseErrorHandler(errorConfig)
```

### Multipart Form Configuration

```go
type MultipartConfig struct {
    MaxMemory    int64
    MaxFiles     int
    TempDir      string
    KeepInMemory bool
}

// Example usage
multipartConfig := blaze.DefaultMultipartConfig()
multipartConfig.MaxMemory = 10 * 1024 * 1024  // 10MB
multipartConfig.MaxFiles = 10
multipartConfig.TempDir = "/tmp/uploads"

app.Use(blaze.MultipartMiddleware(multipartConfig))
```

## Router Configuration

### Router Configuration Structure

```go
type RouterConfig struct {
    CaseSensitive          bool
    StrictSlash            bool
    RedirectSlash          bool
    UseEscapedPath         bool
    HandleMethodNotAllowed bool
    HandleOPTIONS          bool
}

// Example usage
routerConfig := blaze.DefaultRouterConfig()
routerConfig.CaseSensitive = true
routerConfig.StrictSlash = true

router := blaze.NewRouter(routerConfig)
```

### Static File Configuration

```go
type StaticConfig struct {
    Root            string
    Index           string
    Browse          bool
    Compress        bool
    ByteRange       bool
    CacheDuration   time.Duration
    NotFoundHandler HandlerFunc
    Modify          func(*Context) error
    GenerateETag    bool
    Exclude         []string
    MIMETypes       map[string]string
}

// Example usage
staticConfig := blaze.DefaultStaticConfig("./public")
staticConfig.Browse = false  // Disable directory browsing
staticConfig.Compress = true
staticConfig.CacheDuration = 24 * time.Hour
staticConfig.GenerateETag = true

app.Use("/static", blaze.StaticFS(staticConfig))
```

### WebSocket Configuration

```go
type WebSocketConfig struct {
    ReadBufferSize   int
    WriteBufferSize  int
    CheckOrigin      func(ctx *fasthttp.RequestCtx) bool
    ReadTimeout      time.Duration
    WriteTimeout     time.Duration
    PingInterval     time.Duration
    PongTimeout      time.Duration
    MaxMessageSize   int64
    CompressionLevel int
}

// Example usage
wsConfig := blaze.DefaultWebSocketConfig()
wsConfig.MaxMessageSize = 10 * 1024 * 1024  // 10MB
wsConfig.PingInterval = 30 * time.Second
wsConfig.CheckOrigin = func(ctx *fasthttp.RequestCtx) bool {
    origin := string(ctx.Request.Header.Peek("Origin"))
    return origin == "https://example.com"
}

app.WebSocketWithConfig("/ws", handler, wsConfig)
```

## Environment Variables

Create environment-based configuration using Go's standard approach:

```go
package main

import (
    "os"
    "strconv"
    "time"
)

func getEnvConfig() *blaze.Config {
    config := blaze.DefaultConfig()
    
    // Host configuration
    if host := os.Getenv("BLAZE_HOST"); host != "" {
        config.Host = host
    }
    
    // Port configuration
    if port := os.Getenv("BLAZE_PORT"); port != "" {
        if p, err := strconv.Atoi(port); err == nil {
            config.Port = p
        }
    }
    
    // TLS Port
    if tlsPort := os.Getenv("BLAZE_TLS_PORT"); tlsPort != "" {
        if p, err := strconv.Atoi(tlsPort); err == nil {
            config.TLSPort = p
        }
    }
    
    // Development mode
    if dev := os.Getenv("BLAZE_DEVELOPMENT"); dev == "true" {
        config.Development = true
    }
    
    // Enable TLS
    if tls := os.Getenv("BLAZE_ENABLE_TLS"); tls == "true" {
        config.EnableTLS = true
    }
    
    // Enable HTTP/2
    if http2 := os.Getenv("BLAZE_ENABLE_HTTP2"); http2 == "true" {
        config.EnableHTTP2 = true
    }
    
    return config
}

func main() {
    config := getEnvConfig()
    app := blaze.NewWithConfig(config)
    app.ListenAndServe()
}
```

### Environment Variables Reference

| Variable | Type | Description | Default |
|----------|------|-------------|---------|
| `BLAZE_HOST` | string | Server host | `127.0.0.1` |
| `BLAZE_PORT` | int | HTTP port | `8080` |
| `BLAZE_TLS_PORT` | int | HTTPS port | `8443` |
| `BLAZE_DEVELOPMENT` | bool | Development mode | `false` |
| `BLAZE_ENABLE_TLS` | bool | Enable HTTPS | `false` |
| `BLAZE_ENABLE_HTTP2` | bool | Enable HTTP/2 | `false` |
| `BLAZE_CERT_FILE` | string | TLS certificate file | - |
| `BLAZE_KEY_FILE` | string | TLS key file | - |
| `BLAZE_MAX_BODY_SIZE` | int | Max request body size | `4194304` |
| `BLAZE_CONCURRENCY` | int | Max concurrent connections | `262144` |

## Configuration Examples

### Microservice Configuration

```go
func createMicroserviceApp() *blaze.App {
    config := &blaze.Config{
        Host:               "0.0.0.0",
        Port:               8080,
        ReadTimeout:        10 * time.Second,
        WriteTimeout:       10 * time.Second,
        MaxRequestBodySize: 1024 * 1024, // 1MB
        Concurrency:        5000,
        Development:        false,
    }
    
    app := blaze.NewWithConfig(config)
    
    // Add essential middleware
    app.Use(blaze.Logger())
    app.Use(blaze.Recovery())
    app.Use(blaze.ShutdownAware())
    app.Use(blaze.BodyLimitMB(1))
    
    return app
}
```

### API Gateway Configuration

```go
func createAPIGateway() *blaze.App {
    config := blaze.ProductionConfig()
    config.MaxRequestBodySize = 50 * 1024 * 1024 // 50MB for file uploads
    config.Concurrency = 10000
    
    app := blaze.NewWithConfig(config)
    
    // Configure TLS
    tlsConfig := &blaze.TLSConfig{
        CertFile:   "/etc/ssl/certs/api.crt",
        KeyFile:    "/etc/ssl/private/api.key",
        MinVersion: tls.VersionTLS12,
    }
    app.SetTLSConfig(tlsConfig)
    
    // Configure HTTP/2
    http2Config := &blaze.HTTP2Config{
        Enabled:              true,
        MaxConcurrentStreams: 2000,
        EnablePush:           false, // Disable for API
    }
    app.SetHTTP2Config(http2Config)
    
    // Middleware stack
    app.Use(blaze.LoggerMiddleware())
    app.Use(blaze.Recovery())
    app.Use(blaze.RequestIDMiddleware())
    app.Use(blaze.CORS(blaze.CORSOptions{
        AllowOrigins: []string{"https://app.example.com"},
        AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    }))
    app.Use(blaze.RateLimitMiddleware(blaze.RateLimitOptions{
        MaxRequests: 1000,
        Window:      time.Minute,
    }))
    app.Use(blaze.Compress())
    app.Use(blaze.Cache(blaze.DefaultCacheOptions()))
    
    return app
}
```

### Development Configuration with Hot Reload

```go
func createDevelopmentApp() *blaze.App {
    config := blaze.DevelopmentConfig()
    config.Port = 3000
    
    app := blaze.NewWithConfig(config)
    
    // Enable auto TLS for local development
    app.EnableAutoTLS("localhost", "127.0.0.1", "myapp.local")
    
    // Development middleware
    app.Use(blaze.LoggerMiddleware())
    app.Use(blaze.Recovery())
    app.Use(blaze.CORS(blaze.CORSOptions{
        AllowOrigins:     []string{"*"},
        AllowMethods:     []string{"*"},
        AllowHeaders:     []string{"*"},
        AllowCredentials: true,
    }))
    
    // Detailed error responses
    errorConfig := blaze.DevelopmentErrorHandlerConfig()
    errorConfig.EnableStackTrace = true
    errorConfig.IncludeStackTrace = true
    app.UseErrorHandler(errorConfig)
    
    return app
}
```

### High-Performance Configuration

```go
func createHighPerformanceApp() *blaze.App {
    config := &blaze.Config{
        Host:               "0.0.0.0",
        Port:               80,
        TLSPort:            443,
        ReadTimeout:        5 * time.Second,   // Aggressive timeouts
        WriteTimeout:       5 * time.Second,
        MaxRequestBodySize: 1024 * 1024,       // 1MB limit
        Concurrency:        100000,            // High concurrency
        EnableTLS:          true,
        EnableHTTP2:        true,
        RedirectHTTPToTLS:  true,
    }
    
    app := blaze.NewWithConfig(config)
    
    // Configure HTTP/2 for performance
    http2Config := &blaze.HTTP2Config{
        Enabled:              true,
        MaxConcurrentStreams: 5000,
        EnablePush:           true,
        IdleTimeout:          60 * time.Second,
    }
    app.SetHTTP2Config(http2Config)
    
    // Performance middleware
    app.Use(blaze.Cache(blaze.ProductionCacheOptions()))
    app.Use(blaze.CompressWithLevel(blaze.CompressionLevelBest))
    app.Use(blaze.BodyLimitMB(1))
    
    return app
}
```

### Full-Stack Application Configuration

```go
func createFullStackApp() *blaze.App {
    config := blaze.ProductionConfig()
    app := blaze.NewWithConfig(config)
    
    // TLS configuration
    tlsConfig := &blaze.TLSConfig{
        CertFile:   "certs/server.crt",
        KeyFile:    "certs/server.key",
        MinVersion: tls.VersionTLS12,
    }
    app.SetTLSConfig(tlsConfig)
    
    // Global middleware
    app.Use(blaze.LoggerMiddleware())
    app.Use(blaze.Recovery())
    app.Use(blaze.RequestIDMiddleware())
    app.Use(blaze.CompressWithLevel(blaze.CompressionLevelBest))
    
    // Static files
    staticConfig := blaze.DefaultStaticConfig("./public")
    staticConfig.CacheDuration = 24 * time.Hour
    staticConfig.GenerateETag = true
    app.Use("/static", blaze.StaticFS(staticConfig))
    
    // API routes with specific middleware
    api := app.Group("/api/v1")
    api.Use(blaze.CORS(blaze.CORSOptions{
        AllowOrigins: []string{"https://example.com"},
    }))
    api.Use(blaze.BodyLimitMB(10))
    api.Use(blaze.RateLimitMiddleware(blaze.RateLimitOptions{
        MaxRequests: 100,
        Window:      time.Minute,
    }))
    api.Use(blaze.CacheAPI(2 * time.Minute))
    
    // CSRF protection for web routes
    csrfOpts := blaze.ProductionCSRFOptions([]byte("your-secret-key"))
    app.Use(blaze.CSRF(csrfOpts))
    
    return app
}
```

### Configuration Validation

```go
func validateConfig(config *blaze.Config) error {
    if config.Port <= 0 || config.Port > 65535 {
        return fmt.Errorf("invalid port: %d", config.Port)
    }
    
    if config.TLSPort <= 0 || config.TLSPort > 65535 {
        return fmt.Errorf("invalid TLS port: %d", config.TLSPort)
    }
    
    if config.ReadTimeout <= 0 {
        return fmt.Errorf("read timeout must be positive")
    }
    
    if config.WriteTimeout <= 0 {
        return fmt.Errorf("write timeout must be positive")
    }
    
    if config.MaxRequestBodySize <= 0 {
        return fmt.Errorf("max request body size must be positive")
    }
    
    return nil
}
```

The configuration system in Blaze is designed to be flexible and environment-aware, allowing you to easily adapt your application for different deployment scenarios while maintaining optimal performance and security.