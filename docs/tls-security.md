# TLS & Security

The Blaze framework provides comprehensive TLS/SSL support with both production-ready configurations and development conveniences. This guide covers certificate management, security configurations, and best practices for securing your web applications.

## Overview

Blaze offers multiple TLS configuration options :

- **Production TLS**: Full certificate management with custom configurations
- **Development TLS**: Auto-generated self-signed certificates for local development  
- **Auto TLS**: Automatic certificate generation and renewal
- **HTTP/2 over TLS**: Enhanced security with modern protocol support

## Quick Start

### Basic HTTPS Setup

```go
package main

import (
    "github.com/AarambhDevHub/blaze/pkg/blaze"
)

func main() {
    // Create app with production config
    config := blaze.ProductionConfig()
    app := blaze.NewWithConfig(config)

    // Configure TLS with your certificates
    tlsConfig := &blaze.TLSConfig{
        CertFile: "/path/to/your/cert.pem",
        KeyFile:  "/path/to/your/key.pem",
    }
    app.SetTLSConfig(tlsConfig)

    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{"message": "Secure connection established!"})
    })

    // Server automatically starts on HTTPS with TLS enabled
    app.ListenAndServe()
}
```

### Development with Auto-TLS

```go
func main() {
    app := blaze.New()

    // Enable auto-TLS for development (self-signed certificates)
    app.EnableAutoTLS("localhost", "127.0.0.1", "myapp.local")

    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{"message": "Development HTTPS working!"})
    })

    app.ListenAndServe()
}
```

## TLS Configuration

### TLSConfig Structure

The `TLSConfig` struct provides comprehensive certificate and security management :

```go
type TLSConfig struct {
    // Certificate files
    CertFile string  // Path to certificate file
    KeyFile  string  // Path to private key file
    
    // Auto-certificate generation
    AutoTLS      bool     // Enable automatic certificate generation
    TLSCacheDir  string   // Directory to store generated certificates
    Domains      []string // Domains for certificate generation
    Organization string   // Organization name for certificates
    
    // Security settings
    MinVersion   uint16    // Minimum TLS version (e.g., tls.VersionTLS12)
    MaxVersion   uint16    // Maximum TLS version (e.g., tls.VersionTLS13)
    CipherSuites []uint16  // Allowed cipher suites
    
    // Client authentication
    ClientAuth tls.ClientAuthType // Client certificate authentication
    ClientCAs  *x509.CertPool     // Trusted client certificate authorities
    
    // Protocol settings
    NextProtos []string // ALPN protocols (for HTTP/2 support)
    
    // Advanced options
    CertValidityDuration   time.Duration // Certificate validity period
    SessionTicketsDisabled bool          // Disable session tickets
    CurvePreferences       []tls.CurveID // Preferred elliptic curves
    InsecureSkipVerify     bool          // Skip verification (development only)
}
```

### Default Configurations

#### Production Configuration

```go
func productionTLSSetup() *blaze.TLSConfig {
    return &blaze.TLSConfig{
        CertFile:     "/etc/ssl/certs/yourdomain.crt",
        KeyFile:      "/etc/ssl/private/yourdomain.key",
        MinVersion:   tls.VersionTLS12,  // TLS 1.2 minimum
        MaxVersion:   tls.VersionTLS13,  // TLS 1.3 preferred
        NextProtos:   []string{"h2", "http/1.1"}, // HTTP/2 + HTTP/1.1
        ClientAuth:   tls.NoClientCert,  // No client certificates required
        
        // Security hardening
        SessionTicketsDisabled: false,
        CurvePreferences: []tls.CurveID{
            tls.X25519,    // Modern, fast curve
            tls.CurveP256, // NIST P-256
            tls.CurveP384, // NIST P-384
        },
    }
}
```

#### Development Configuration

```go
func developmentTLSSetup() *blaze.TLSConfig {
    return &blaze.TLSConfig{
        AutoTLS:              true,
        TLSCacheDir:          "./certs",
        Domains:              []string{"localhost", "127.0.0.1"},
        Organization:         "Development Corp",
        MinVersion:           tls.VersionTLS12,
        CertValidityDuration: 365 * 24 * time.Hour, // 1 year
        InsecureSkipVerify:   true, // Only for development!
    }
}
```

### Custom TLS Setup

```go
func customTLSConfiguration() *blaze.TLSConfig {
    return &blaze.TLSConfig{
        CertFile:   "custom.crt",
        KeyFile:    "custom.key",
        MinVersion: tls.VersionTLS13, // TLS 1.3 only
        MaxVersion: tls.VersionTLS13,
        
        // Custom cipher suites (TLS 1.2)
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
        },
        
        // Enable client certificate authentication
        ClientAuth: tls.RequireAndVerifyClientCert,
        
        // HTTP/2 support
        NextProtos: []string{"h2"},
    }
}
```

## Certificate Management

### Using Existing Certificates

```go
func useExistingCertificates() {
    app := blaze.New()

    tlsConfig := &blaze.TLSConfig{
        CertFile: "/path/to/certificate.pem",
        KeyFile:  "/path/to/private-key.pem",
    }

    app.SetTLSConfig(tlsConfig)
    
    // Server info includes certificate details
    info := app.GetServerInfo()
    if info.TLS != nil && info.TLS.Certificate != nil {
        fmt.Printf("Certificate expires: %v\n", info.TLS.Certificate.NotAfter)
        fmt.Printf("Valid domains: %v\n", info.TLS.Certificate.DNSNames)
    }

    app.ListenAndServe()
}
```

### Auto-Generated Certificates

The framework can automatically generate self-signed certificates for development :

```go
func autoGeneratedCerts() {
    app := blaze.New()

    tlsConfig := &blaze.TLSConfig{
        AutoTLS:     true,
        TLSCacheDir: "./certificates", // Directory to store certs
        Domains:     []string{"localhost", "127.0.0.1", "*.local"},
        Organization: "My Development Company",
        
        // Certificate settings
        CertValidityDuration: 365 * 24 * time.Hour, // 1 year
        MinVersion:          tls.VersionTLS12,
    }

    app.SetTLSConfig(tlsConfig)
    app.ListenAndServe() // Certificates auto-generated on first run
}
```

### Certificate Information & Health Checks

```go
func certificateHealthCheck() {
    app := blaze.New()
    
    // Configure with certificates
    tlsConfig := &blaze.TLSConfig{
        CertFile: "server.crt",
        KeyFile:  "server.key",
    }
    app.SetTLSConfig(tlsConfig)

    // Health check endpoint
    app.GET("/health/tls", func(c *blaze.Context) error {
        serverInfo := app.GetServerInfo()
        
        if serverInfo.TLS == nil {
            return c.Status(503).JSON(blaze.Map{
                "tls_enabled": false,
                "error": "TLS not configured",
            })
        }

        cert := serverInfo.TLS.Certificate
        if cert == nil {
            return c.Status(503).JSON(blaze.Map{
                "tls_enabled": true,
                "error": "Certificate not loaded",
            })
        }

        return c.JSON(blaze.Map{
            "tls_enabled":     true,
            "certificate": blaze.Map{
                "subject":       cert.Subject,
                "issuer":        cert.Issuer,
                "not_before":    cert.NotBefore,
                "not_after":     cert.NotAfter,
                "expires_in":    cert.ExpiresIn(),
                "is_expired":    cert.IsExpired(),
                "dns_names":     cert.DNSNames,
                "ip_addresses":  cert.IPAddresses,
            },
            "protocols":       serverInfo.TLS.NextProtocols,
        })
    })
}
```

## Security Middleware

### HTTPS Redirect

Automatically redirect HTTP traffic to HTTPS :

```go
func httpsRedirect() {
    config := blaze.ProductionConfig()
    config.RedirectHTTPToTLS = true  // Enable automatic HTTP->HTTPS redirect
    config.Port = 80                 // HTTP port
    config.TLSPort = 443            // HTTPS port

    app := blaze.NewWithConfig(config)

    tlsConfig := &blaze.TLSConfig{
        CertFile: "server.crt",
        KeyFile:  "server.key",
    }
    app.SetTLSConfig(tlsConfig)

    // All HTTP requests automatically redirect to HTTPS
    app.ListenAndServe()
}
```

### Security Headers Middleware

```go
func securityHeadersMiddleware() blaze.MiddlewareFunc {
    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // HSTS (HTTP Strict Transport Security)
            c.SetHeader("Strict-Transport-Security", 
                "max-age=31536000; includeSubDomains; preload")
            
            // Prevent MIME sniffing
            c.SetHeader("X-Content-Type-Options", "nosniff")
            
            // Clickjacking protection
            c.SetHeader("X-Frame-Options", "DENY")
            
            // XSS protection
            c.SetHeader("X-XSS-Protection", "1; mode=block")
            
            // Content Security Policy
            c.SetHeader("Content-Security-Policy", 
                "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
            
            // Referrer Policy
            c.SetHeader("Referrer-Policy", "strict-origin-when-cross-origin")
            
            return next(c)
        }
    }
}

func secureApp() {
    app := blaze.New()
    
    // Apply security headers to all routes
    app.Use(securityHeadersMiddleware())
    
    // HTTP/2 specific security
    app.Use(blaze.HTTP2Security())
    
    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{"message": "Secure response"})
    })
}
```

### CSRF Protection

The framework includes comprehensive CSRF protection :

```go
func csrfProtection() {
    app := blaze.New()

    // Configure CSRF protection
    csrfConfig := &blaze.CSRFOptions{
        TokenLookup: []string{
            "header:X-CSRF-Token",
            "form:csrf_token",
            "query:_token",
        },
        CookieName:      "csrf_token",
        CookieSecure:    true,  // HTTPS only
        CookieHTTPOnly:  true,  // No JavaScript access
        CookieSameSite:  "Strict",
        Expiration:      24 * time.Hour,
        SingleUse:       false, // Allow token reuse
        TrustedOrigins:  []string{"https://yourdomain.com"},
    }

    app.Use(blaze.CSRF(csrfConfig))

    app.GET("/form", func(c *blaze.Context) error {
        // Get CSRF token for forms
        token := blaze.CSRFToken(c)
        
        html := fmt.Sprintf(`
        <form method="POST" action="/submit">
            %s
            <input type="text" name="data" required>
            <button type="submit">Submit</button>
        </form>`, blaze.CSRFTokenHTML(c))
        
        return c.HTML(html)
    })

    app.POST("/submit", func(c *blaze.Context) error {
        // CSRF validation happens automatically
        data := c.FormValue("data")
        return c.JSON(blaze.Map{"received": data})
    })
}
```

## Client Certificate Authentication

### Mutual TLS (mTLS)

```go
func mutualTLSSetup() {
    app := blaze.New()

    // Load client CA certificates
    clientCAs := x509.NewCertPool()
    clientCACert, err := ioutil.ReadFile("client-ca.pem")
    if err != nil {
        log.Fatal(err)
    }
    clientCAs.AppendCertsFromPEM(clientCACert)

    tlsConfig := &blaze.TLSConfig{
        CertFile:   "server.crt",
        KeyFile:    "server.key",
        ClientAuth: tls.RequireAndVerifyClientCert,
        ClientCAs:  clientCAs,
    }

    app.SetTLSConfig(tlsConfig)

    // Middleware to extract client certificate info
    app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Access client certificate (if present)
            if conn := c.Context().Value("tls_conn"); conn != nil {
                if tlsConn, ok := conn.(*tls.Conn); ok {
                    if len(tlsConn.ConnectionState().PeerCertificates) > 0 {
                        clientCert := tlsConn.ConnectionState().PeerCertificates[0]
                        c.SetLocals("client_cert", clientCert)
                        c.SetLocals("client_subject", clientCert.Subject.CommonName)
                    }
                }
            }
            return next(c)
        }
    })

    app.GET("/protected", func(c *blaze.Context) error {
        clientSubject := c.Locals("client_subject")
        if clientSubject == nil {
            return c.Status(401).JSON(blaze.Map{"error": "Client certificate required"})
        }

        return c.JSON(blaze.Map{
            "message": "Access granted",
            "client":  clientSubject,
        })
    })
}
```

## HTTP/2 with TLS

### Enhanced Security with HTTP/2

```go
func http2WithTLS() {
    // Production config with HTTP/2 enabled
    config := blaze.ProductionConfig()
    config.EnableHTTP2 = true

    app := blaze.NewWithConfig(config)

    // TLS configuration with HTTP/2 support
    tlsConfig := &blaze.TLSConfig{
        CertFile:   "server.crt",
        KeyFile:    "server.key",
        MinVersion: tls.VersionTLS12,
        NextProtos: []string{"h2", "http/1.1"}, // HTTP/2 preferred
        
        // HTTP/2 recommended cipher suites
        CipherSuites: []uint16{
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        },
    }

    app.SetTLSConfig(tlsConfig)

    // HTTP/2 specific middleware
    app.Use(blaze.HTTP2Security())
    app.Use(blaze.HTTP2Info())

    app.GET("/", func(c *blaze.Context) error {
        return c.JSON(blaze.Map{
            "protocol": c.Protocol(),
            "http2":    c.IsHTTP2(),
            "secure":   true,
        })
    })

    // HTTP/2 server push example
    app.GET("/page", func(c *blaze.Context) error {
        // Push critical resources
        resources := map[string]string{
            "/style.css": "text/css",
            "/app.js":    "application/javascript",
        }
        c.PushResources(resources)

        html := `
        <!DOCTYPE html>
        <html>
        <head>
            <link rel="stylesheet" href="/style.css">
            <script src="/app.js"></script>
        </head>
        <body>
            <h1>HTTP/2 with Server Push</h1>
        </body>
        </html>`

        return c.HTML(html)
    })
}
```

## Security Best Practices

### Production Security Checklist

```go
func productionSecuritySetup() {
    config := blaze.ProductionConfig()
    config.EnableHTTP2 = true
    config.RedirectHTTPToTLS = true

    app := blaze.NewWithConfig(config)

    // 1. Strong TLS Configuration
    tlsConfig := &blaze.TLSConfig{
        CertFile:   "/etc/ssl/certs/domain.crt",
        KeyFile:    "/etc/ssl/private/domain.key",
        MinVersion: tls.VersionTLS12, // TLS 1.2 minimum
        MaxVersion: tls.VersionTLS13, // TLS 1.3 preferred
        
        // Strong cipher suites only
        CipherSuites: []uint16{
            tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
        },
        
        // Modern curves
        CurvePreferences: []tls.CurveID{
            tls.X25519,
            tls.CurveP256,
        },
        
        // Disable session tickets for perfect forward secrecy
        SessionTicketsDisabled: true,
        
        // HTTP/2 + HTTP/1.1 fallback
        NextProtos: []string{"h2", "http/1.1"},
    }

    app.SetTLSConfig(tlsConfig)

    // 2. Security Middleware Stack
    app.Use(blaze.Recovery())
    app.Use(securityHeadersMiddleware())
    app.Use(blaze.HTTP2Security())
    
    // 3. CSRF Protection
    app.Use(blaze.CSRF(&blaze.CSRFOptions{
        CookieSecure:   true,
        CookieHTTPOnly: true,
        CookieSameSite: "Strict",
    }))

    // 4. Rate limiting
    app.Use(blaze.RateLimit(&blaze.RateLimitOptions{
        Max:      100,
        Duration: time.Minute,
    }))

    // 5. Request timeout
    app.Use(blaze.GracefulTimeout(30 * time.Second))
}
```

### Certificate Monitoring

```go
func certificateMonitoring() {
    app := blaze.New()
    
    // Certificate expiration monitoring endpoint
    app.GET("/health/certificate", func(c *blaze.Context) error {
        serverInfo := app.GetServerInfo()
        
        if serverInfo.TLS?.Certificate == nil {
            return c.Status(503).JSON(blaze.Map{
                "status": "unhealthy",
                "error":  "No certificate configured",
            })
        }

        cert := serverInfo.TLS.Certificate
        expiresIn := cert.ExpiresIn()
        
        // Alert if certificate expires in less than 30 days
        status := "healthy"
        if expiresIn < 30*24*time.Hour {
            status = "warning"
            if expiresIn < 7*24*time.Hour {
                status = "critical"
            }
        }

        return c.JSON(blaze.Map{
            "status":      status,
            "expires_in":  expiresIn.String(),
            "not_after":   cert.NotAfter,
            "domains":     cert.DNSNames,
            "is_expired":  cert.IsExpired(),
        })
    })
}
```

## Common Configurations

### Let's Encrypt Integration

```go
func letsEncryptSetup() {
    app := blaze.New()

    // Use certificates from Let's Encrypt/Certbot
    tlsConfig := &blaze.TLSConfig{
        CertFile: "/etc/letsencrypt/live/yourdomain.com/fullchain.pem",
        KeyFile:  "/etc/letsencrypt/live/yourdomain.com/privkey.pem",
        MinVersion: tls.VersionTLS12,
        NextProtos: []string{"h2", "http/1.1"},
    }

    app.SetTLSConfig(tlsConfig)
    
    // Graceful certificate renewal handling
    app.RegisterGracefulTask(func(ctx context.Context) error {
        // Certificate renewal logic here
        log.Println("Checking certificate renewal...")
        return nil
    })
}
```

### Load Balancer Integration

```go
func loadBalancerTLS() {
    app := blaze.New()

    // Trust proxy headers for real client IP
    app.Use(blaze.IPMiddleware())

    // Handle X-Forwarded-Proto from load balancer
    app.Use(func(next blaze.HandlerFunc) blaze.HandlerFunc {
        return func(c *blaze.Context) error {
            // Check if behind TLS-terminating proxy
            if proto := c.Header("X-Forwarded-Proto"); proto == "https" {
                c.SetLocals("secure_connection", true)
            }
            return next(c)
        }
    })

    app.GET("/", func(c *blaze.Context) error {
        secure := c.Locals("secure_connection") != nil
        return c.JSON(blaze.Map{
            "secure": secure,
            "ip":     c.GetRealIP(),
        })
    })
}
```

This comprehensive TLS & Security documentation covers all aspects of securing your Blaze web applications, from basic HTTPS setup to advanced security configurations and best practices. The framework provides flexible security options suitable for both development and production environments.