# Configuration

Blaze provides flexible configuration options to customize your web application for different environments and use cases. This guide covers all configuration aspects including server settings, TLS/HTTPS, HTTP/2, and environment-specific configurations.

## Table of Contents

- [Basic Configuration](#basic-configuration)
- [Configuration Presets](#configuration-presets)
- [Server Configuration](#server-configuration)
- [TLS/HTTPS Configuration](#tlshttps-configuration)
- [HTTP/2 Configuration](#http2-configuration)
- [Environment Variables](#environment-variables)
- [Configuration Examples](#configuration-examples)

## Basic Configuration

### Config Structure

The main configuration is handled through the `Config` struct:

```go
type Config struct {
    Host               string
    Port               int
    TLSPort            int
    ReadTimeout        time.Duration
    WriteTimeout       time.Duration
    MaxRequestBodySize int
    Concurrency        int
    
    // Protocol configuration
    EnableHTTP2       bool
    EnableTLS         bool
    RedirectHTTPToTLS bool
    
    // Development settings
    Development bool
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
    AutoTLS      bool
    TLSCacheDir  string
    Domains      []string
    Organization string
    
    // TLS settings
    MinVersion   uint16
    MaxVersion   uint16
    CipherSuites []uint16
    
    // Client authentication
    ClientAuth tls.ClientAuthType
    ClientCAs  *x509.CertPool
    
    // HTTP/2 support
    NextProtos []string
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
    app.Use(blaze.Logger())
    app.Use(blaze.Recovery())
    app.Use(blaze.CORSWithConfig(&blaze.CORSConfig{
        AllowOrigins: []string{"*"},
        AllowMethods: []string{"*"},
        AllowHeaders: []string{"*"},
    }))
    
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
    
    // Add performance middleware
    app.Use(blaze.Cache(blaze.NewMemoryStore(), &blaze.CacheOptions{
        MaxAge: 300 * time.Second, // 5 minutes
        Public: true,
    }))
    
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