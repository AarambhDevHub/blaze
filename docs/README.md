# Blaze Framework Documentation

Blaze is a **blazing fast, lightweight web framework for Go** inspired by modern frameworks like Axum and Actix Web. This comprehensive documentation provides everything needed to build high-performance web applications with Blaze.

## Overview

Blaze is designed for **high-performance web applications** with built-in support for HTTP/2, TLS, WebSockets, advanced middleware capabilities, and comprehensive validation. The framework follows modern architectural patterns while maintaining simplicity and performance.

### Key Features

- **High Performance**: Built on FastHTTP for maximum throughput (155K+ req/s) and minimal latency
- **HTTP/2 Support**: Native HTTP/2 with server push capabilities and h2c (HTTP/2 over cleartext)
- **Advanced Routing**: Radix tree router with constraints, wildcards, and all HTTP methods (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, CONNECT, TRACE, ANY, Match)
- **Comprehensive Middleware**: CORS, CSRF, authentication, rate limiting, caching (LRU/LFU/FIFO), compression, body limits, and request ID
- **Validation System**: Integrated struct validation with go-playground/validator
- **Multipart Forms**: Struct-based binding with validation tags and automatic file handling
- **TLS Security**: Automated TLS configuration with self-signed certificates for development
- **WebSocket Support**: Full-duplex communication with connection management and broadcasting
- **Static File Serving**: Advanced configuration with caching, compression, ETag, and range requests
- **Graceful Shutdown**: Context-aware shutdown handling for production deployments
- **Request Context**: Rich context with parameter binding, locals, validation, and timeout management
- **Production Ready**: Comprehensive configuration options for development and production

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Blaze Framework Architecture                 │
├─────────────────────────────────────────────────────────────────────┤
│  Application Layer                                                  │
│  ┌─────────────────────┐  ┌─────────────────────┐                   │
│  │     HTTP/2 Server   │  │    HTTP/1.1 Server  │                   │
│  │   (golang.org/x/net)│  │     (FastHTTP)      │                   │
│  └─────────────────────┘  └─────────────────────┘                   │
├─────────────────────────────────────────────────────────────────────┤
│  Middleware Layer                                                   │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐        │
│  │  CORS   │ │  CSRF   │ │  Cache  │ │ Logger  │ │Recovery │        │
│  │  Auth   │ │RateLimit│ │Compress │ │BodyLimit│ │Validate │        │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘        │
├─────────────────────────────────────────────────────────────────────┤
│  Routing Layer                                                      │
│  ┌─────────────────────┐  ┌─────────────────────┐                   │
│  │   Radix Tree        │  │   Route Groups      │                   │
│  │   Router            │  │   & Constraints     │                   │
│  └─────────────────────┘  └─────────────────────┘                   │
├─────────────────────────────────────────────────────────────────────┤
│  Context Layer                                                      │
│  ┌──────────────────────────────────────────────────────────────────┤
│  │  Request: Params, Query, Headers, Body, Validation               │
│  │  Response: JSON, HTML, Files, Status, Headers                    │
│  │  WebSocket: Upgrade, Messages, Broadcasting                      │
│  │  Files: Upload, Download, Stream, Static Serving                 │
│  └──────────────────────────────────────────────────────────────────┤
├─────────────────────────────────────────────────────────────────────┤
│  Core Components                                                    │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐        │
│  │   TLS   │ │Multipart│ │Validator│ │Graceful │ │ Error   │        │
│  │ Config  │ │ Binding │ │ System  │ │Shutdown │ │Handling │        │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘        │
└─────────────────────────────────────────────────────────────────────┘
```

The architecture is built on **four core principles**:

1. **Performance First**: FastHTTP foundation with HTTP/2 support for maximum throughput (155K+ req/s)
2. **Developer Experience**: Intuitive API with comprehensive middleware, validation, and context management
3. **Production Ready**: Built-in security, monitoring, caching, and deployment features
4. **Type Safety**: Struct-based binding with automatic validation and error handling

## Quick Start

```go
package main

import (
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
    app := blaze.New()
    
    // Simple route
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "message": "Hello, Blaze!",
            "status":  "success",
        })
    })
    
    // Route with validation
    type User struct {
        Name  string `json:"name" validate:"required,min=2,max=100"`
        Email string `json:"email" validate:"required,email"`
        Age   int    `json:"age" validate:"gte=18,lte=100"`
    }
    
    app.POST("/users", func(c *blaze.Context) error {
        var user User
        
        // Bind and validate in one call
        if err := c.BindJSONAndValidate(&user); err != nil {
            return c.Status(400).JSON(blaze.Map{"error": err.Error()})
        }
        
        return c.Status(201).JSON(user)
    })
    
    app.ListenAndServe()
}
```

See the complete [Quick Start Guide](quick-start.md) for detailed instructions.

## Documentation Structure

This documentation is organized into focused sections for different aspects of the framework:

### **Core Documentation**
- [**Installation**](installation.md) - Installation and setup instructions
- [**Quick Start**](quick-start.md) - Getting started with your first application
- [**Configuration**](configuration.md) - Application configuration and environment setup
- [**API Reference**](api-reference.md) - Complete API documentation

### **Routing & Handlers**
- [**Routing**](routing.md) - URL routing with all HTTP methods, parameters, constraints, and route groups
- [**Handlers**](handlers.md) - Request handlers and response patterns
- [**Context**](context.md) - Request/response context and utilities
- [**Request-Response**](request-response.md) - Detailed request/response handling

### **Middleware & Security**
- [**Middleware**](middleware.md) - Built-in and custom middleware (CORS, CSRF, Rate Limit, Cache, Compression)
- [**TLS Security**](tls-security.md) - TLS configuration and security features

### **Data Handling**
- [**Validation**](validator.md) - Struct validation with go-playground/validator
- [**File Handling**](file-handling.md) - File uploads, downloads, and multipart forms with struct binding
- [**Static Files**](static-files.md) - Static file serving with caching, compression, and range requests

### **Advanced Features**
- [**WebSockets**](websockets.md) - WebSocket implementation and patterns
- [**HTTP/2**](http2.md) - HTTP/2 configuration and server push
- [**Examples**](examples.md) - Complete application examples and patterns

## Core Components

### Application Core (`app.go`)
- **App struct**: Main application instance with state management
- **Server Management**: HTTP/1.1 and HTTP/2 server coordination
- **Graceful Shutdown**: Context-aware shutdown with task coordination and timeout handling
- **Configuration**: Development, production, and custom configuration profiles
- **State Management**: Application-level key-value store for shared data

### Request Context (`context.go`)
- **Parameter Handling**: Route parameters with type conversion (`Param`, `ParamInt`, `ParamIntDefault`)
- **Request Processing**: Headers, query parameters, body parsing, cookies, and client information
- **Response Generation**: JSON, HTML, text responses with chainable status codes and headers
- **File Operations**: Upload handling, serving, downloading, streaming with range request support
- **Validation**: Integrated validation with `BindAndValidate`, `BindJSONAndValidate`, `ValidateVar`
- **Local Storage**: Request-scoped data storage with `Locals` and `SetLocals`
- **HTTP/2 Features**: Protocol detection, server push, stream ID access
- **Graceful Shutdown**: `IsShuttingDown`, `ShutdownContext`, `WithTimeout`, `WithDeadline`

### Routing System (`router.go`)
- **Radix Tree**: Efficient pattern matching with parameter extraction
- **All HTTP Methods**: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, CONNECT, TRACE, ANY, Match
- **Route Groups**: Shared prefixes and middleware for organized routing
- **Constraints**: Parameter validation with built-in (int, UUID, regex) and custom validators
- **Route Options**: Named routes, priorities, tags, and route-specific middleware

### Middleware System (`middleware.go`)
Built-in middleware includes:
- **Logger**: Configurable request/response logging with slow request detection
- **Recovery**: Panic recovery with stack traces
- **CORS**: Cross-Origin Resource Sharing with fine-grained control
- **CSRF**: Cross-Site Request Forgery protection with tokens
- **Rate Limiting**: IP-based or custom key rate limiting
- **Caching**: Multiple eviction strategies (LRU, LFU, FIFO, Random) with compression
- **Compression**: Gzip, Deflate, Brotli compression with configurable levels
- **Body Limit**: Request size limits with content-type specific rules
- **Authentication**: Token-based authentication with custom validators
- **Request ID**: Unique request tracking for distributed tracing
- **HTTP/2 Specific**: Security headers, stream info, metrics

### Validation System (`validator.go`)
- **Struct Validation**: Automatic validation using go-playground/validator tags
- **Integrated Binding**: `BindAndValidate`, `BindJSONAndValidate`, `BindFormAndValidate`, `BindMultipartFormAndValidate`
- **Single Variable**: Validate individual values with `ValidateVar`
- **Body Size**: Request size validation with `ValidateBodySize`
- **Custom Validators**: Register custom validation rules
- **Error Formatting**: User-friendly error messages with field names

### Multipart Forms (`multipart.go`, `buildmultipart.go`)
- **Struct Binding**: Bind forms directly to Go structs with validation tags
- **File Upload**: Single and multiple file uploads with `*MultipartFile` and `[]*MultipartFile`
- **Form Tags**: Validation tags (`required`, `minsize`, `maxsize`, `default`)
- **Type Support**: Strings, integers, booleans, time.Time, slices, pointers
- **Configuration**: Customizable memory limits, file size limits, allowed types

### Security & TLS (`tls.go`)
- **Automatic TLS**: Self-signed certificates for development environments
- **Production TLS**: Certificate management with proper cipher suites and OCSP stapling
- **HTTP/2 Integration**: Seamless TLS configuration for HTTP/2 with ALPN
- **Security Features**: Client auth, session tickets, renegotiation control

### Static File Serving (`static.go`)
- **Configuration**: Index files, directory browsing, compression, caching
- **Performance**: ETag generation, byte-range requests, MIME type detection
- **Security**: Directory traversal protection, file exclusion patterns
- **Custom Handlers**: 404 handlers, response modification hooks

## Framework Philosophy

Blaze is built around several core design principles:

### **Performance by Design**
Every component is optimized for speed and minimal memory allocation. The framework leverages FastHTTP's zero-allocation design, adds HTTP/2 support, and provides efficient caching with multiple eviction algorithms.

### **Developer Ergonomics**
The API is designed to be intuitive and type-safe, with comprehensive context management, automatic validation, and error handling. Developers can focus on business logic rather than infrastructure concerns.

### **Production Readiness**
Built-in features like graceful shutdown, health checks, CSRF protection, rate limiting, caching, compression, and security middleware ensure applications are ready for production deployment.

### **Extensibility**
The middleware system and modular architecture allow for easy customization and extension without modifying core framework code. Custom validators, middleware, and error handlers integrate seamlessly.

## Performance Characteristics

Blaze is designed for **high-throughput, low-latency applications**:

- **155,000+ requests/second** with optimized routing and middleware
- **Zero-allocation routing** with radix tree implementation
- **FastHTTP foundation** for maximum HTTP/1.1 performance
- **Native HTTP/2 support** with multiplexing and server push
- **Efficient middleware pipeline** with minimal overhead
- **Smart caching** with LRU/LFU/FIFO eviction strategies
- **Connection pooling** and resource reuse
- **Compression** with multiple algorithms (Gzip, Deflate, Brotli)

## Comprehensive Feature List

### Routing & Request Handling
- ✅ All HTTP methods: GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, CONNECT, TRACE
- ✅ ANY route (handles all methods)
- ✅ Match route (handles specific multiple methods)
- ✅ Named parameters with type conversion (`:param`)
- ✅ Wildcard parameters (`*param`)
- ✅ Route constraints (int, UUID, regex, custom)
- ✅ Route groups with shared middleware
- ✅ Named routes with priorities and tags
- ✅ Query parameter handling with defaults

### Data Binding & Validation
- ✅ JSON body binding
- ✅ Form data binding
- ✅ Multipart form binding with struct tags
- ✅ Automatic validation with go-playground/validator
- ✅ Combined bind and validate methods
- ✅ Single variable validation
- ✅ Body size validation
- ✅ Custom validators

### Response Types
- ✅ JSON responses with helpers (OK, Created, Error)
- ✅ Paginated responses
- ✅ HTML responses
- ✅ Text responses
- ✅ File serving and downloads
- ✅ File streaming with range requests
- ✅ Redirects (301, 302, 307, 308)
- ✅ Custom status codes and headers

### Middleware (Built-in)
- ✅ Logger with configurable options
- ✅ Recovery with stack traces
- ✅ CORS with fine-grained control
- ✅ CSRF protection with tokens
- ✅ Rate limiting (per IP or custom key)
- ✅ Caching (LRU, LFU, FIFO, Random)
- ✅ Compression (Gzip, Deflate, Brotli)
- ✅ Body limits (global and per-route)
- ✅ Authentication (token-based)
- ✅ Request ID generation
- ✅ Graceful shutdown awareness
- ✅ HTTP/2 specific middleware

### File Handling
- ✅ Single file uploads
- ✅ Multiple file uploads
- ✅ Struct-based multipart binding
- ✅ File validation (size, type, extension)
- ✅ Unique filename generation
- ✅ Static file serving with configuration
- ✅ Directory browsing (configurable)
- ✅ ETag generation
- ✅ Byte-range requests
- ✅ MIME type detection

### WebSocket Support
- ✅ WebSocket upgrade
- ✅ Message reading/writing (text, binary)
- ✅ JSON message support
- ✅ Connection management
- ✅ Broadcasting with hub
- ✅ Ping/Pong support
- ✅ Configurable timeouts and buffer sizes

### HTTP/2 Features
- ✅ Native HTTP/2 support
- ✅ Server push (single and multiple resources)
- ✅ Stream ID access
- ✅ Protocol detection
- ✅ h2c (HTTP/2 over cleartext)
- ✅ Configurable stream limits

### Security
- ✅ TLS configuration (production and development)
- ✅ Auto-generated self-signed certificates
- ✅ CSRF protection
- ✅ CORS configuration
- ✅ Security headers
- ✅ Directory traversal protection
- ✅ Rate limiting
- ✅ Body size limits

### Production Features
- ✅ Graceful shutdown with context
- ✅ Health check endpoints
- ✅ Configuration profiles (dev, prod, custom)
- ✅ Application state management
- ✅ Request-scoped locals
- ✅ Comprehensive error handling
- ✅ Logging system
- ✅ Metrics and monitoring hooks

## Getting Help

- **Documentation**: Comprehensive guides and API reference in this documentation
- **Examples**: Real-world examples and patterns in [examples.md](examples.md)
- **API Reference**: Complete API documentation in [api-reference.md](api-reference.md)

## Technology Stack

- **Go**: 1.24+
- **FastHTTP**: High-performance HTTP/1.1 server
- **golang.org/x/net/http2**: Native HTTP/2 implementation
- **go-playground/validator**: Struct validation
- **fasthttp-websocket**: WebSocket support
- **json-iterator**: Fast JSON serialization

## Author & License

**Framework Author**: [AarambhDevHub](https://github.com/AarambhDevHub)

See [LICENSE](../LICENSE) for license information.

---

**Ready to build blazing fast web applications?** Start with the [Quick Start Guide](quick-start.md) and explore the comprehensive documentation above.
