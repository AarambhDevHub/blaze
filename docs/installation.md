# Installation Guide

The Bla requires Go version 1.24.0 or later, and uses Go modules for dependency management. This comprehensive guide will walk through setting up your development environment and installing the framework.

## Prerequisites

### Go Installation

Before installi ensure you have Go 1.24.0 or later installed on your system. You can download Go from the official website at [golang.org](https://go.dev/doc/install).

#### Verify Go Installation

Check your Go installation by running:

```bash
go version
```

The output should show Go version 1.24.0 or higher.

### System Requirements

The framework supports all platforms that Go supports, including:

- **Linux**: All major distributions
- **Windo 10 or later
- **macOS**: macOS 10.15 or later
- **Other platforms**: ARM, BSD variants

## Installing Blaze Framework

### Method 1: Using go get (Recommended)

The simplest way to install Blaze is using Go's built-in package manager.

#### Initialize Your Project

First, create a new directory for your project and initialize a Go module:

```bash
mkdir my-blaze-app
cd my-blaze-app
go mod init github.com/yourusername/my-blaze-app
```

Replace `yourusername` with your actual GitHub username or organization name.

#### Install Blaze Framework

Install the Blaze framework and its dependencies:

```bash
go get github.com/AarambhDevHub/blaze
```

Th download the framework and automatically update your `go.mod` file with the dependency.

### Method 2: Manual Installation

For development or custom builds, you can clone the repository directly:

```bash
git clone https://github.com/AarambhDevHub/blaze.git
cd blaze
go mod tidy
```

The `go mod tidy` command ensures all dependencies are properly resolved and downloaded.

## Project Structure

After installation, your project should have the following structure:

```
your-project/
├── go.mod
├── go.sum
├── main.go
└── other project files...
```

### Understanding go.mod

The `go.mod` file tracks your project's dependencies. After installing Blaze, it should contain:

```go
module github.com/yourusername/my-blaze-app

go 1.24.0

require (
    github.com/AarambhDevHub/blaze v0.1.3
)
```

T these key dependencies :

- `github.com/fasthttp/websocket v1.5.12` - WebSocket support
- `github.com/json-iterator/go v1.1.12` - Fast JSON processing  
- `github.com/valyala/fasthttp v1.66.0` - High-performance HTTP server
- `golang.org/x/net v0.44.0` - HTTP/2 support

## Environment Setup

### Development Configurati environments, Blaze provides optimized configurations :

```go
// Development settings
config := blaze.DevelopmentConfig()
// Host: 127.0.0.1
// Port: 3000
// TLS disabled
// HTTP/2 disabled for simplicity
```

### Production Configurati deployments, use the production configuration :

```go
// Production settings  
config := blaze.ProductionConfig()
// Host: 0.0.0.0
// Port: 80/443
// TLS enabled
// HTTP/2 enabled
// Enhanced security settings
```

## Verification

### Quick Start Example

Create a simple application to verify your installation:

```go
package main

import (
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
    app := blaze.New()
    
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "message": "Blaze framework installed successfully!",
            "version": "1.0.0",
        })
    })
    
    app.ListenAndServe() // Starts on port 8080
}
```

### Run Your Application

Execute your application:

```bash
go run main.go
```

Vislocalhost:8080` in your browser to see the welcome message.

## Dependency Management

### Updating Dependencies

To update Blaze to the latest version:

```bash
go get -u github.com/AarambhDevHub/blaze
go mod tidy
```

### Viewing Dependencies

Check all project dependencies:

```bash
go list -m all
```

### Cleaning Module Cache

If you encounter dependency issues, clean the module cache :

```bash
go clean -modcache
```

## Troubleshooting

### Common Issue Version Compatibility**: Ensure you're using Go 1.24.0 or later. The framework uses modern Go features that require this version.

**Module Path Issues**: Use a valid module path format like `github.com/username/project` or `example.com/project`.

**Dependency Conflicts**: Run `go mod tidy` to resolve version conflicts and ensure clean dependencies.

**Build Errors**: Verify all required dependencies are installed by running `go mod download`.

### Getting Help

If you encounter issues during installation:

1. Check the [official repository](https://github.com/AarambhDevHub/blaze) for documentation
2. Ensure your Go installation is properly configured
3. Verify network connectivity for dependency downloads
4. Review error messages carefully for specific guidance

The installation process is designed to be straightforward, and the framework includes comprehensive configuration options for both development and production environments.