# CORS Documentation

Complete guide to Cross-Origin Resource Sharing (CORS) configuration and middleware in the Blaze web framework.

## Table of Contents

- [Overview](#overview)
- [What is CORS?](#what-is-cors)
- [Configuration](#configuration)
- [Basic Usage](#basic-usage)
- [Advanced Configuration](#advanced-configuration)
- [Preflight Requests](#preflight-requests)
- [Security Considerations](#security-considerations)
- [Common Scenarios](#common-scenarios)
- [Troubleshooting](#troubleshooting)

## Overview

CORS (Cross-Origin Resource Sharing) is a security feature implemented by browsers to control how resources on a web page can be requested from another domain. Blaze provides comprehensive CORS middleware for handling cross-origin requests.

### Key Features

- **Simple Configuration**: Default settings for common use cases
- **Wildcard Origins**: Allow all origins with `*`
- **Explicit Origins**: Whitelist specific domains
- **Method Control**: Specify allowed HTTP methods
- **Header Management**: Configure allowed and exposed headers
- **Credentials Support**: Enable cookie/authentication sharing
- **Preflight Caching**: Configure browser cache duration

## What is CORS?

### Same-Origin Policy

Browsers restrict cross-origin HTTP requests by default. Requests fail if:
- **Different domain**: `example.com` → `api.example.com`
- **Different port**: `localhost:3000` → `localhost:8080`
- **Different protocol**: `http://` → `https://`

### How CORS Works

1. **Browser** sends request with `Origin` header
2. **Server** responds with CORS headers
3. **Browser** checks if origin is allowed
4. **Browser** allows/blocks based on headers

### Preflight Requests

For non-simple requests, browsers send an OPTIONS preflight request first:

**Simple Requests** (no preflight):
- Methods: GET, HEAD, POST
- Headers: Accept, Accept-Language, Content-Language
- Content-Type: application/x-www-form-urlencoded, multipart/form-data, text/plain

**Non-Simple Requests** (requires preflight):
- Methods: PUT, DELETE, PATCH
- Custom headers: Authorization, X-Custom-Header
- Content-Type: application/json

## Configuration

### CORSOptions

```go
type CORSOptions struct {
    // AllowedOrigins specifies allowed origins
    // Use "*" for all origins (development only)
    // Use explicit domains for production: []string{"https://example.com"}
    AllowedOrigins []string
    
    // AllowedMethods specifies allowed HTTP methods
    // Default: GET, POST, PUT, DELETE, PATCH, OPTIONS
    AllowedMethods []string
    
    // AllowedHeaders specifies headers client can send
    // Default: Content-Type, Authorization, X-Requested-With
    AllowedHeaders []string
    
    // ExposedHeaders specifies headers client can access
    // Default: nil (only simple response headers exposed)
    ExposedHeaders []string
    
    // AllowCredentials enables cookies and authentication
    // Cannot be used with AllowedOrigins: "*"
    // Default: false
    AllowCredentials bool
    
    // MaxAge specifies preflight cache duration in seconds
    // Default: 600 (10 minutes)
    MaxAge int
}
```

### Default Configuration

```go
func DefaultCORSOptions() CORSOptions {
    return CORSOptions{
        AllowedOrigins: []string{"*"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
        AllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
        ExposedHeaders: nil,
        AllowCredentials: false,
        MaxAge: 600,
    }
}
```

## Basic Usage

### Allow All Origins (Development)

```go
app := blaze.New()

// Use default configuration - allows all origins
app.Use(blaze.CORS(blaze.DefaultCORSOptions()))

app.GET("/api/data", handler)
```

### Production Configuration

```go
app := blaze.New()

// Restrict to specific origins
opts := blaze.CORSOptions{
    AllowedOrigins: []string{
        "https://example.com",
        "https://app.example.com",
    },
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders: []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge: 3600, // 1 hour
}

app.Use(blaze.CORS(opts))
```

## Advanced Configuration

### Multiple Environments

```go
func getCORSOptions() blaze.CORSOptions {
    if os.Getenv("ENV") == "production" {
        return blaze.CORSOptions{
            AllowedOrigins: []string{
                "https://example.com",
                "https://www.example.com",
            },
            AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
            AllowedHeaders: []string{"Content-Type", "Authorization"},
            AllowCredentials: true,
            MaxAge: 3600,
        }
    }
    
    // Development - allow all
    return blaze.DefaultCORSOptions()
}

app.Use(blaze.CORS(getCORSOptions()))
```

### Custom Headers

```go
opts := blaze.CORSOptions{
    AllowedOrigins: []string{"https://example.com"},
    AllowedMethods: []string{"GET", "POST"},
    AllowedHeaders: []string{
        "Content-Type",
        "Authorization",
        "X-API-Key",          // Custom header
        "X-Request-ID",       // Custom header
    },
    ExposedHeaders: []string{
        "X-Total-Count",      // Expose custom header to client
        "X-Rate-Limit",       // Expose rate limit info
    },
    MaxAge: 86400, // 24 hours
}

app.Use(blaze.CORS(opts))
```

### Dynamic Origin Validation

```go
func validateOrigin(origin string) bool {
    allowedOrigins := []string{
        "https://example.com",
        "https://app.example.com",
    }
    
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    
    // Allow localhost in development
    if strings.HasPrefix(origin, "http://localhost:") {
        return os.Getenv("ENV") != "production"
    }
    
    return false
}

// Custom CORS middleware
app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        origin := c.Header("Origin")
        
        if origin != "" && validateOrigin(origin) {
            c.SetHeader("Access-Control-Allow-Origin", origin)
            c.SetHeader("Access-Control-Allow-Credentials", "true")
            c.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
            c.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")
            c.SetHeader("Access-Control-Max-Age", "3600")
        }
        
        // Handle preflight
        if c.Method() == "OPTIONS" {
            return c.Status(204).Text("")
        }
        
        return next(c)
    }
})
```

## Preflight Requests

### Understanding Preflight

Browsers send an OPTIONS request before certain cross-origin requests:

```
OPTIONS /api/users HTTP/1.1
Host: api.example.com
Origin: https://example.com
Access-Control-Request-Method: POST
Access-Control-Request-Headers: Content-Type, Authorization
```

Server responds with CORS headers:

```
HTTP/1.1 204 No Content
Access-Control-Allow-Origin: https://example.com
Access-Control-Allow-Methods: GET, POST, PUT, DELETE
Access-Control-Allow-Headers: Content-Type, Authorization
Access-Control-Max-Age: 3600
```

### Preflight Caching

Set `MaxAge` to reduce preflight requests:

```go
opts := blaze.CORSOptions{
    AllowedOrigins: []string{"https://example.com"},
    MaxAge: 86400, // Cache for 24 hours
}
```

**Benefits**:
- Reduces network requests
- Improves performance
- Less server load

**Considerations**:
- Changes take longer to propagate
- Set lower for development
- Set higher for production

## Security Considerations

### Never Use Wildcard with Credentials

```go
// ❌ INSECURE - Don't do this
opts := blaze.CORSOptions{
    AllowedOrigins: []string{"*"},
    AllowCredentials: true, // This won't work and is insecure
}

// ✅ SECURE - Specify exact origins
opts := blaze.CORSOptions{
    AllowedOrigins: []string{"https://example.com"},
    AllowCredentials: true,
}
```

### Validate Origins Strictly

```go
// ❌ INSECURE - Regex matching can be bypassed
func validateOrigin(origin string) bool {
    return strings.Contains(origin, "example.com") // Can match evil-example.com
}

// ✅ SECURE - Exact match
func validateOrigin(origin string) bool {
    allowed := []string{
        "https://example.com",
        "https://www.example.com",
    }
    
    for _, o := range allowed {
        if origin == o {
            return true
        }
    }
    return false
}
```

### Minimize Exposed Headers

```go
// ❌ Over-exposure
opts := blaze.CORSOptions{
    ExposedHeaders: []string{
        "X-Internal-Token",    // Don't expose internal details
        "X-Database-Query",    // Don't expose implementation
    },
}

// ✅ Only necessary headers
opts := blaze.CORSOptions{
    ExposedHeaders: []string{
        "X-Total-Count",
        "X-Rate-Limit-Remaining",
    },
}
```

## Common Scenarios

### Single Page Application (SPA)

```go
// Frontend at https://app.example.com
// Backend API at https://api.example.com

opts := blaze.CORSOptions{
    AllowedOrigins: []string{"https://app.example.com"},
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
    AllowedHeaders: []string{"Content-Type", "Authorization"},
    ExposedHeaders: []string{"X-Total-Count"},
    AllowCredentials: true, // For cookies/JWT
    MaxAge: 3600,
}

app.Use(blaze.CORS(opts))
```

### Public API

```go
// Public API - no authentication
opts := blaze.CORSOptions{
    AllowedOrigins: []string{"*"},
    AllowedMethods: []string{"GET", "POST"},
    AllowedHeaders: []string{"Content-Type"},
    AllowCredentials: false,
    MaxAge: 86400,
}

app.Use(blaze.CORS(opts))
```

### Mobile App Backend

```go
// Mobile apps don't need CORS, but web admin panel does
opts := blaze.CORSOptions{
    AllowedOrigins: []string{
        "https://admin.example.com", // Admin web panel
    },
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders: []string{"Content-Type", "Authorization", "X-API-Key"},
    AllowCredentials: true,
    MaxAge: 3600,
}

app.Use(blaze.CORS(opts))
```

### Subdomain Access

```go
// Allow all subdomains of example.com
func validateOrigin(origin string) bool {
    // Parse origin URL
    u, err := url.Parse(origin)
    if err != nil {
        return false
    }
    
    // Check if domain is example.com or *.example.com
    if u.Hostname() == "example.com" {
        return true
    }
    
    if strings.HasSuffix(u.Hostname(), ".example.com") {
        return true
    }
    
    return false
}

// Use custom middleware with dynamic validation
app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
    return func(c *blaze.Context) error {
        origin := c.Header("Origin")
        
        if origin != "" && validateOrigin(origin) {
            c.SetHeader("Access-Control-Allow-Origin", origin)
            c.SetHeader("Access-Control-Allow-Credentials", "true")
            c.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
            c.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")
        }
        
        if c.Method() == "OPTIONS" {
            return c.Status(204).Text("")
        }
        
        return next(c)
    }
})
```

## Troubleshooting

### CORS Error in Browser

**Error**: "Access to fetch at 'https://api.example.com' from origin 'https://app.example.com' has been blocked by CORS policy"

**Solutions**:

1. **Add origin to AllowedOrigins**:
   ```go
   opts.AllowedOrigins = []string{"https://app.example.com"}
   ```

2. **Check method is allowed**:
   ```go
   opts.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE"}
   ```

3. **Check headers are allowed**:
   ```go
   opts.AllowedHeaders = []string{"Content-Type", "Authorization"}
   ```

### Credentials Not Working

**Error**: "The value of the 'Access-Control-Allow-Credentials' header in the response is '' which must be 'true'"

**Solution**:
```go
opts := blaze.CORSOptions{
    AllowedOrigins: []string{"https://example.com"}, // Must be explicit
    AllowCredentials: true, // Enable credentials
}

// Client side
fetch('https://api.example.com/data', {
    credentials: 'include' // Important!
})
```

### Preflight Failing

**Check**:
1. OPTIONS method is handled
2. Correct CORS headers in OPTIONS response
3. Status code is 204 (no content)

**Solution**: Blaze CORS middleware handles this automatically

### Headers Not Accessible

**Error**: Client can't read custom response headers

**Solution**:
```go
opts := blaze.CORSOptions{
    AllowedOrigins: []string{"https://example.com"},
    ExposedHeaders: []string{"X-Total-Count", "X-Custom-Header"},
}
```

### Testing CORS

```go
# Test preflight request
curl -X OPTIONS http://localhost:8080/api/users \
  -H "Origin: https://example.com" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type" \
  -v

# Test actual request
curl -X GET http://localhost:8080/api/users \
  -H "Origin: https://example.com" \
  -v
```

---

For more information:
- [Security Best Practices](./security.md)
- [Middleware Guide](./middleware.md)
- [Authentication Guide](./authentication.md)
