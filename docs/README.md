# Blaze Framework Documentation README

Blaze is a **blazing fast, lightweight web framework for Go** inspired by modern frameworks like Axum and Actix Web. This comprehensive documentation provides everything needed to build high-performance web applications with Blaze.

## Overview

Blaze is designed for **high-performance web applications** with built-in support for HTTP/2, TLS, WebSockets, and advanced middleware capabilities. The framework follows modern architectural patterns while maintaining simplicity and performance.

### Key Features

- **High Performance**: Built on FastHTTP for maximum throughput and minimal latency
- **HTTP/2 Support**: Native HTTP/2 with server push capabilities and h2c (HTTP/2 over cleartext)
- **TLS Security**: Automated TLS configuration with self-signed certificates for development
- **WebSocket Support**: Full-duplex communication with connection management and message routing
- **Advanced Routing**: Pattern matching with parameter extraction and constraints
- **Middleware System**: Composable middleware with request/response interceptors
- **Multipart Forms**: Built-in file upload handling with validation and cleanup
- **Graceful Shutdown**: Context-aware shutdown handling for production deployments
- **Request Context**: Rich context with parameter binding, locals, and timeout management
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
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │  CORS   │ │  Auth   │ │  Cache  │ │ Logger  │ │Recovery │ ...   │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
├─────────────────────────────────────────────────────────────────────┤
│  Routing Layer                                                      │
│  ┌─────────────────────┐  ┌─────────────────────┐                   │
│  │   Radix Tree        │  │   Route Groups      │                   │
│  │   Router            │  │   & Parameters      │                   │
│  └─────────────────────┘  └─────────────────────┘                   │
├─────────────────────────────────────────────────────────────────────┤
│  Context Layer                                                      │
│  ┌─────────────────────────────────────────────────────────────────┤
│  │  Request Context (Params, Query, Headers, Body, Locals)        │
│  │  Response Context (Status, Headers, JSON, HTML, Files)         │
│  │  WebSocket Context (Upgrade, Messages, Connection Management)  │
│  └─────────────────────────────────────────────────────────────────┤
├─────────────────────────────────────────────────────────────────────┤
│  Core Components                                                    │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐       │
│  │   TLS   │ │Multipart│ │ Health  │ │Graceful │ │ Error   │       │
│  │ Config  │ │ Forms   │ │ Check   │ │Shutdown │ │Handling │       │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘       │
└─────────────────────────────────────────────────────────────────────┘
```

The architecture is built on **three core principles** :

1. **Performance First**: FastHTTP foundation with HTTP/2 support for maximum throughput
2. **Developer Experience**: Intuitive API with comprehensive middleware and context management
3. **Production Ready**: Built-in security, monitoring, and deployment features

### Core Components

#### Application Core (`app.go`)
- **App struct**: Main application instance with configuration management
- **Server Management**: HTTP/1.1 and HTTP/2 server coordination
- **Graceful Shutdown**: Context-aware shutdown with task coordination
- **Configuration**: Development, production, and custom configuration profiles

#### Request Context (`context.go`)  
- **Parameter Handling**: Route parameters with type conversion and validation
- **Request Processing**: Headers, query parameters, body parsing, and cookies
- **Response Generation**: JSON, HTML, text responses with status codes
- **File Operations**: Upload handling, serving static files, and streaming

#### Routing System (`router.go`)
- **Radix Tree**: Efficient pattern matching with parameter extraction
- **Route Groups**: Shared prefixes and middleware for organized routing
- **Constraints**: Parameter validation with built-in and custom validators

#### Middleware System (`middleware.go`)
- **Composable Pipeline**: Chain multiple middleware functions
- **Built-in Middleware**: Authentication, CORS, rate limiting, caching, and logging
- **Custom Middleware**: Easy creation of application-specific middleware

#### Security & TLS (`tls.go`)
- **Automatic TLS**: Self-signed certificates for development environments
- **Production TLS**: Certificate management with proper cipher suites
- **HTTP/2 Integration**: Seamless TLS configuration for HTTP/2

## Quick Start

See the complete getting started guide in [`quick-start.md`](quick-start.md) for step-by-step instructions, examples, and best practices.

## Documentation Structure

This documentation is organized into focused sections for different aspects of the framework:

### **Core Documentation**
- [`installation.md`](installation.md) - Installation and setup instructions
- [`quick-start.md`](quick-start.md) - Getting started with your first application
- [`configuration.md`](configuration.md) - Application configuration and environment setup

### **Routing & Handlers**  
- [`routing.md`](routing.md) - URL routing, parameters, and route groups
- [`handlers.md`](handlers.md) - Request handlers and response patterns
- [`context.md`](context.md) - Request/response context and utilities
- [`request-response.md`](request-response.md) - Detailed request/response handling

### **Middleware & Security**
- [`middleware.md`](middleware.md) - Built-in and custom middleware
- [`tls-security.md`](tls-security.md) - TLS configuration and security features

### **Advanced Features**
- [`websockets.md`](websockets.md) - WebSocket implementation and patterns  
- [`http2.md`](http2.md) - HTTP/2 configuration and server push
- [`file-handling.md`](file-handling.md) - Static files and file uploads
- [`multipart-forms.md`](multipart-forms.md) - Form processing and file uploads

### **Reference & Guides**
- [`examples.md`](examples.md) - Complete application examples and patterns
- [`api-reference.md`](api-reference.md) - Complete API documentation
- [`troubleshooting.md`](troubleshooting.md) - Common issues and solutions

### **Development**
- [`contributing.md`](contributing.md) - Contributing guidelines and development setup

## Framework Philosophy

Blaze is built around several core design principles :

### **Performance by Design**
Every component is optimized for speed and minimal memory allocation. The framework leverages FastHTTP's zero-allocation design and adds HTTP/2 support for modern performance requirements.

### **Developer Ergonomics**  
The API is designed to be intuitive and type-safe, with comprehensive context management and error handling. Developers can focus on business logic rather than infrastructure concerns.

### **Production Readiness**
Built-in features like graceful shutdown, health checks, metrics collection, and security middleware ensure applications are ready for production deployment.

### **Extensibility**
The middleware system and modular architecture allow for easy customization and extension without modifying core framework code.

## Performance Characteristics

Blaze is designed for **high-throughput, low-latency applications** :

- **Zero-allocation routing** with radix tree implementation
- **FastHTTP foundation** for maximum HTTP/1.1 performance  
- **Native HTTP/2 support** with multiplexing and server push
- **Efficient middleware pipeline** with minimal overhead
- **Smart memory management** with connection pooling and reuse

## Getting Help

- **Documentation**: Comprehensive guides and API reference in this documentation
- **Examples**: Real-world examples and patterns in [`examples.md`](examples.md)
- **Troubleshooting**: Common issues and solutions in [`troubleshooting.md`](troubleshooting.md)
- **Contributing**: Development guidelines in [`contributing.md`](contributing.md)

## Author & Ecosystem

**Framework Author**: [AarambhDevHub](https://github.com/AarambhDevHub)
**Technology Stack**: Go 1.24+, FastHTTP, golang.org/x/net/http2
**License**: See [`license.md`](../LICENSE) for license information

***

**Ready to build blazing fast web applications?** Start with the [Quick Start Guide](quick-start.md) and explore the comprehensive documentation above.

