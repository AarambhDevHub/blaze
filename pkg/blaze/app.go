// Package blaze provides a lightweight, high-performance web framework for Go inspired by Axum and Actix Web.
//
// Blaze offers a modern, fast, and feature-rich web framework that includes built-in support for:
// - HTTP/1.1 and HTTP/2 protocols
// - TLS/SSL with automatic certificate generation
// - WebSocket connections
// - Middleware pipeline
// - Advanced routing with parameters
// - Graceful shutdown
// - Request ID tracking
// - Performance optimizations
//
// Example Usage:
//
//	app := blaze.New()
//
//	// Add middleware
//	app.Use(blaze.Logger())
//	app.Use(blaze.Recovery())
//
//	// Define routes
//	app.GET("/", func(c blaze.Context) error {
//	    return c.JSON(blaze.Map{"message": "Hello, Blaze!"})
//	})
//
//	// Start server
//	log.Fatal(app.ListenAndServe())
package blaze

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/valyala/fasthttp"
)

// App represents the main application instance that serves as the core of the Blaze web framework.
// It encapsulates the HTTP server, routing, middleware, and configuration management.
//
// The App struct provides methods for:
// - Route registration (GET, POST, PUT, DELETE, etc.)
// - Middleware management
// - Server configuration (TLS, HTTP/2, graceful shutdown)
// - WebSocket upgrade handling
// - Route grouping with shared prefixes and middleware
//
// Thread Safety: App is designed to be thread-safe for concurrent operations
// during the setup phase, but should not be modified after starting the server.
type App struct {
	router      *Router          // HTTP router for handling request routing and parameter extraction
	middleware  []MiddlewareFunc // Global middleware stack applied to all requests in reverse order
	server      *fasthttp.Server // Underlying FastHTTP server instance for HTTP/1.1
	config      *Config          // Application configuration including ports, timeouts, and feature flags
	tlsConfig   *TLSConfig       // TLS/SSL configuration for HTTPS support
	http2Config *HTTP2Config     // HTTP/2 protocol configuration
	http2Server *HTTP2Server     // HTTP/2 server instance when HTTP/2 is enabled

	// Graceful shutdown management
	shutdownCtx    context.Context    // Context for coordinating graceful shutdown across components
	shutdownCancel context.CancelFunc // Function to trigger shutdown signal to all components
	shutdownWg     sync.WaitGroup     // WaitGroup to ensure all graceful tasks complete before shutdown
	shutdownOnce   sync.Once          // Ensures shutdown logic executes only once
	isShuttingDown bool               // Atomic flag indicating if server is in shutdown process
	mu             sync.RWMutex       // Protects concurrent access to shutdown state
}

// Config holds comprehensive application configuration for server behavior,
// performance tuning, and feature enablement.
//
// Performance Settings:
// - ReadTimeout/WriteTimeout: Prevent slow-loris attacks and resource exhaustion
// - MaxRequestBodySize: Limits request payload size (default 4MB)
// - Concurrency: Controls FastHTTP worker goroutines (default 256*1024)
//
// Protocol Settings:
// - EnableHTTP2: Enables HTTP/2 multiplexing and server push
// - EnableTLS: Activates HTTPS with configurable cipher suites
// - RedirectHTTPToTLS: Automatically redirects HTTP traffic to HTTPS
//
// Development vs Production:
// - Development mode enables debug features and relaxed security
// - Production mode enforces strict security and optimal performance
type Config struct {
	Host string // Bind address for the server (0.0.0.0 for all interfaces, 127.0.0.1 for localhost)
	Port int    // HTTP port (typically 80 for production, 8080 for development)

	// TLS Configuration
	TLSPort int // HTTPS port (typically 443 for production, 8443 for development)

	// Timeout Configuration - Critical for preventing resource exhaustion
	ReadTimeout  time.Duration // Maximum time to read entire request including body
	WriteTimeout time.Duration // Maximum time to write response

	// Resource Limits
	MaxRequestBodySize int // Maximum size in bytes for request body (prevents memory exhaustion)
	Concurrency        int // Maximum number of concurrent connections (FastHTTP worker pool size)

	// Protocol Configuration
	EnableHTTP2       bool // Enable HTTP/2 support with multiplexing and server push
	EnableTLS         bool // Enable TLS/HTTPS support
	RedirectHTTPToTLS bool // Automatically redirect HTTP requests to HTTPS

	// Development Settings
	Development bool // Enable development mode (relaxed security, debug features)
}

// DefaultConfig returns a secure, production-ready configuration with sensible defaults.
//
// Default Values:
// - Host: 127.0.0.1 (localhost only for security)
// - Port: 8080 (non-privileged port)
// - Timeouts: 10s read/write (prevents slow clients from exhausting resources)
// - Body Size: 4MB limit (balances functionality and security)
// - Concurrency: 256K connections (high-performance default)
// - Protocols: HTTP/1.1 only (maximum compatibility)
//
// Use this as a starting point and modify as needed for your environment.
func DefaultConfig() *Config {
	return &Config{
		Host:               "127.0.0.1",
		Port:               8080,
		TLSPort:            8443,
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       10 * time.Second,
		MaxRequestBodySize: 4 * 1024 * 1024, // 4MB
		Concurrency:        256 * 1024,
		EnableHTTP2:        false,
		EnableTLS:          false,
		RedirectHTTPToTLS:  false,
		Development:        false,
	}
}

// ProductionConfig returns a configuration optimized for production deployment.
//
// Production Features:
// - Binds to all interfaces (0.0.0.0) for external access
// - Uses standard HTTP/HTTPS ports (80/443)
// - Enables HTTP/2 for improved performance
// - Enables TLS with automatic HTTP->HTTPS redirection
// - Longer timeouts for real-world network conditions
// - Larger request body limit for file uploads
// - Maximum concurrency for high-load scenarios
//
// Security Considerations:
// - Always use with proper TLS certificates in production
// - Configure firewall rules to protect the server
// - Monitor resource usage and adjust limits as needed
func ProductionConfig() *Config {
	return &Config{
		Host:               "0.0.0.0",
		Port:               80,
		TLSPort:            443,
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		MaxRequestBodySize: 10 * 1024 * 1024, // 10MB
		Concurrency:        256 * 1024,
		EnableHTTP2:        true,
		EnableTLS:          true,
		RedirectHTTPToTLS:  true,
		Development:        false,
	}
}

// DevelopmentConfig returns a configuration optimized for local development.
//
// Development Features:
// - Localhost binding for security during development
// - Non-standard ports to avoid conflicts
// - Relaxed timeouts for debugging
// - HTTP/1.1 only for simplicity
// - No TLS by default (can be enabled separately)
// - Development mode enables debug features
//
// This configuration is ideal for local development, testing, and debugging.
func DevelopmentConfig() *Config {
	return &Config{
		Host:               "127.0.0.1",
		Port:               3000,
		TLSPort:            3443,
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       10 * time.Second,
		MaxRequestBodySize: 4 * 1024 * 1024, // 4MB
		Concurrency:        256 * 1024,
		EnableHTTP2:        false,
		EnableTLS:          false,
		RedirectHTTPToTLS:  false,
		Development:        true,
	}
}

// New creates a new Blaze application with default configuration.
//
// The application is initialized with:
// - Empty middleware stack (add middleware with Use())
// - New router instance with trie-based path matching
// - Default configuration (localhost:8080, HTTP/1.1)
// - Graceful shutdown context for coordinating shutdown
//
// Example:
//
//	app := blaze.New()
//	app.GET("/", handler)
//	log.Fatal(app.ListenAndServe())
//
// For custom configuration, use NewWithConfig() instead.
func New() *App {
	ctx, cancel := context.WithCancel(context.Background())

	return &App{
		router:         NewRouter(),
		middleware:     make([]MiddlewareFunc, 0),
		config:         DefaultConfig(),
		shutdownCtx:    ctx,
		shutdownCancel: cancel,
	}
}

// NewWithConfig creates a new Blaze application with custom configuration.
//
// This allows fine-tuning of server behavior, performance characteristics,
// and protocol support from initialization.
//
// Configuration is applied immediately and affects:
// - Server binding and ports
// - Protocol support (HTTP/2, TLS)
// - Performance limits and timeouts
// - Development vs production behavior
//
// Example:
//
//	config := blaze.ProductionConfig()
//	config.Port = 8080
//	app := blaze.NewWithConfig(config)
//
// TLS and HTTP/2 servers are automatically configured based on config flags.
func NewWithConfig(config *Config) *App {
	ctx, cancel := context.WithCancel(context.Background())
	app := &App{
		router:         NewRouter(),
		middleware:     make([]MiddlewareFunc, 0),
		config:         config,
		shutdownCtx:    ctx,
		shutdownCancel: cancel,
	}

	// Configure TLS and HTTP/2 based on config
	if config.EnableTLS {
		if config.Development {
			app.tlsConfig = DevelopmentTLSConfig()
		} else {
			app.tlsConfig = DefaultTLSConfig()
		}
	}

	if config.EnableHTTP2 {
		if config.Development {
			app.http2Config = DevelopmentHTTP2Config()
		} else {
			app.http2Config = DefaultHTTP2Config()
		}
		app.http2Server = NewHTTP2Server(app.http2Config, app.tlsConfig)
	}

	return app
}

// SetTLSConfig applies a custom TLS configuration to the application.
//
// This method allows fine-tuning of TLS behavior including:
// - Certificate and key management
// - Cipher suite selection
// - Protocol version limits
// - Client certificate authentication
// - OCSP stapling and other advanced features
//
// The TLS configuration affects both HTTP/1.1 and HTTP/2 servers.
// If HTTP/2 is enabled, the HTTP/2 server is automatically updated.
//
// Example:
//
//	tlsConfig := blaze.DefaultTLSConfig()
//	tlsConfig.CertFile = "/path/to/cert.pem"
//	tlsConfig.KeyFile = "/path/to/key.pem"
//	app.SetTLSConfig(tlsConfig)
//
// Returns the app instance for method chaining.
func (a *App) SetTLSConfig(config *TLSConfig) *App {
	a.tlsConfig = config
	a.config.EnableTLS = config != nil

	// Update HTTP/2 server if exists
	if a.http2Server != nil {
		a.http2Server = NewHTTP2Server(a.http2Config, a.tlsConfig)
	}

	return a
}

// SetHTTP2Config applies a custom HTTP/2 configuration to the application.
//
// HTTP/2 configuration controls:
// - Stream concurrency limits (prevents resource exhaustion)
// - Upload buffer sizes (balances memory usage and performance)
// - Server push capabilities (for performance optimization)
// - H2C support (HTTP/2 over cleartext for development)
// - Frame size limits (affects streaming performance)
//
// When HTTP/2 is enabled, a new HTTP/2 server instance is created
// with the updated configuration.
//
// Example:
//
//	http2Config := blaze.DefaultHTTP2Config()
//	http2Config.MaxConcurrentStreams = 500
//	http2Config.EnablePush = true
//	app.SetHTTP2Config(http2Config)
//
// Returns the app instance for method chaining.
func (a *App) SetHTTP2Config(config *HTTP2Config) *App {
	a.http2Config = config
	a.config.EnableHTTP2 = config != nil && config.Enabled

	// Create or update HTTP/2 server
	if a.config.EnableHTTP2 {
		a.http2Server = NewHTTP2Server(a.http2Config, a.tlsConfig)
	}

	return a
}

// EnableAutoTLS enables automatic TLS with self-signed certificates for development.
//
// This is a convenience method that:
// - Generates self-signed certificates for specified domains
// - Configures TLS with development-friendly settings
// - Enables HTTPS on the configured TLS port
//
// The generated certificates are cached in the .certs directory and
// reused across application restarts until they expire.
//
// WARNING: Self-signed certificates should NEVER be used in production.
// Browsers will show security warnings, and the connection is not truly secure.
//
// Parameters:
//
//	domains: List of domains/IPs for the certificate (default: localhost, 127.0.0.1)
//
// Example:
//
//	app.EnableAutoTLS("localhost", "127.0.0.1", "::1")
//
// Returns the app instance for method chaining.
func (a *App) EnableAutoTLS(domains ...string) *App {
	if len(domains) == 0 {
		domains = []string{"localhost", "127.0.0.1"}
	}

	tlsConfig := DevelopmentTLSConfig()
	tlsConfig.Domains = domains

	return a.SetTLSConfig(tlsConfig)
}

// IsShuttingDown returns true if the application is in the graceful shutdown process.
//
// This method is thread-safe and can be called from middleware, handlers,
// or background tasks to determine if they should terminate early.
//
// During shutdown:
// - New requests receive 503 Service Unavailable responses
// - Existing requests are allowed to complete (with timeout)
// - Background tasks should check this flag and exit gracefully
//
// Example usage in a long-running handler:
//
//	func handler(c blaze.Context) error {
//	    for i := 0; i < 1000; i++ {
//	        if c.App.IsShuttingDown() {
//	            return c.Status(503).Text("Server shutting down")
//	        }
//	        // Do work...
//	        time.Sleep(time.Millisecond)
//	    }
//	    return c.Text("Complete")
//	}
func (a *App) IsShuttingDown() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.isShuttingDown
}

// setShuttingDown sets the shutdown state
func (a *App) setShuttingDown(state bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.isShuttingDown = state
}

// GetShutdownContext returns the application's shutdown context.
//
// This context is cancelled when graceful shutdown begins, allowing
// background tasks, middleware, and handlers to react appropriately.
//
// The shutdown context can be used to:
// - Cancel long-running operations during shutdown
// - Implement cleanup logic in background goroutines
// - Coordinate shutdown across application components
//
// Example:
//
//	go func() {
//	    ticker := time.NewTicker(1 * time.Second)
//	    defer ticker.Stop()
//
//	    for {
//	        select {
//	        case <-app.GetShutdownContext().Done():
//	            log.Println("Background task shutting down")
//	            return
//	        case <-ticker.C:
//	            // Do periodic work
//	        }
//	    }
//	}()
func (a *App) GetShutdownContext() context.Context {
	return a.shutdownCtx
}

// RegisterGracefulTask registers a cleanup task to run during graceful shutdown.
//
// Tasks are executed concurrently when shutdown begins, allowing the application
// to clean up resources, save state, or notify external systems.
//
// Each task receives a context with a 30-second timeout. Tasks should:
// - Monitor the context for cancellation
// - Perform cleanup operations quickly
// - Return gracefully if the timeout is reached
//
// Tasks are executed in separate goroutines, so they can run concurrently
// without blocking each other.
//
// Example:
//
//	app.RegisterGracefulTask(func(ctx context.Context) error {
//	    log.Println("Closing database connections...")
//	    return db.Close()
//	})
//
//	app.RegisterGracefulTask(func(ctx context.Context) error {
//	    log.Println("Flushing metrics...")
//	    return metrics.Flush(ctx)
//	})
func (a *App) RegisterGracefulTask(task func(ctx context.Context) error) {
	a.shutdownWg.Add(1)
	go func() {
		defer a.shutdownWg.Done()
		<-a.shutdownCtx.Done()

		taskCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := task(taskCtx); err != nil {
			log.Printf("Graceful task error: %v", err)
		}
	}()
}

// Shutdown gracefully shuts down the server with a configurable timeout.
//
// The graceful shutdown process follows these steps:
// 1. Mark the application as shutting down (new requests get 503)
// 2. Cancel the shutdown context to notify all components
// 3. Wait for all registered graceful tasks to complete
// 4. Shutdown HTTP/2 server if enabled
// 5. Shutdown HTTP/1.1 server
// 6. Force shutdown if timeout is exceeded
//
// The shutdown process respects the provided context timeout. If graceful
// shutdown cannot complete within the timeout, a force shutdown is performed.
//
// Parameters:
//
//	ctx: Context with timeout for the shutdown process
//
// Returns:
//
//	error: Any error that occurred during shutdown
//
// Example:
//
//	// Shutdown with 30-second timeout
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	if err := app.Shutdown(ctx); err != nil {
//	    log.Printf("Shutdown error: %v", err)
//	}
func (a *App) Shutdown(ctx context.Context) error {
	var shutdownErr error

	a.shutdownOnce.Do(func() {
		log.Println("🛑 Initiating graceful shutdown...")
		a.setShuttingDown(true)

		// Cancel shutdown context to notify all tasks
		a.shutdownCancel()

		// Create a channel to handle shutdown completion
		done := make(chan error, 1)

		go func() {
			// Wait for all graceful tasks to complete
			log.Println("⏳ Waiting for graceful tasks to complete...")
			a.shutdownWg.Wait()

			// Shutdown HTTP/2 server first
			if a.http2Server != nil {
				log.Println("🔌 Shutting down HTTP/2 server...")
				if err := a.http2Server.Shutdown(context.Background()); err != nil {
					log.Printf("HTTP/2 server shutdown error: %v", err)
				}
			}

			// Shutdown the fasthttp server
			if a.server != nil {
				log.Println("🔌 Shutting down HTTP server...")
				if err := a.server.Shutdown(); err != nil {
					done <- fmt.Errorf("server shutdown error: %w", err)
					return
				}
			}

			log.Println("✅ Graceful shutdown completed")
			done <- nil
		}()

		// Wait for shutdown to complete or timeout
		select {
		case shutdownErr = <-done:
			// Shutdown completed
		case <-ctx.Done():
			shutdownErr = ctx.Err()
			log.Printf("⚠️ Graceful shutdown timeout: %v", shutdownErr)

			// Force shutdown if graceful shutdown times out
			if a.http2Server != nil {
				log.Println("🚨 Forcing HTTP/2 server shutdown...")
				a.http2Server.Close()
			}
			if a.server != nil {
				log.Println("🚨 Forcing HTTP server shutdown...")
				a.server.Shutdown()
			}
		}
	})

	return shutdownErr
}

// ListenAndServe starts the appropriate server based on configuration.
//
// This method analyzes the application configuration and starts the correct
// combination of servers:
//
// Server Selection Logic:
// 1. HTTP/2 + TLS: Starts HTTP/2 server with TLS on TLS port
// 2. HTTP/2 + H2C: Starts HTTP/2 server with cleartext on HTTP port
// 3. HTTP/1.1 + TLS: Starts FastHTTP server with TLS on TLS port
// 4. HTTP/1.1: Starts FastHTTP server on HTTP port
//
// Additional Features:
// - Automatic HTTP->HTTPS redirect server when configured
// - Proper TLS configuration with FastHTTP integration
// - Comprehensive error handling and logging
// - Support for both production and development scenarios
//
// The server blocks until an error occurs or shutdown is initiated.
//
// Returns:
//
//	error: Server startup or runtime error
//
// Example:
//
//	// Start server with current configuration
//	log.Fatal(app.ListenAndServe())
func (a *App) startHTTPRedirectServer() {
	if !a.config.RedirectHTTPToTLS || !a.config.EnableTLS {
		return
	}

	redirectHandler := func(ctx *fasthttp.RequestCtx) {
		httpsURL := fmt.Sprintf("https://%s:%d%s",
			a.config.Host,
			a.config.TLSPort,
			string(ctx.RequestURI()))
		ctx.Redirect(httpsURL, fasthttp.StatusMovedPermanently)
	}

	redirectServer := &fasthttp.Server{
		Handler:            redirectHandler,
		ReadTimeout:        a.config.ReadTimeout,
		WriteTimeout:       a.config.WriteTimeout,
		MaxRequestBodySize: a.config.MaxRequestBodySize,
	}

	httpAddr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)
	go func() {
		log.Printf("🔀 HTTP redirect server starting on http://%s", httpAddr)
		if err := redirectServer.ListenAndServe(httpAddr); err != nil {
			log.Printf("HTTP redirect server error: %v", err)
		}
	}()
}

// ListenAndServe starts the appropriate server based on configuration
func (a *App) ListenAndServe() error {
	// Setup server
	addr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)
	tlsAddr := fmt.Sprintf("%s:%d", a.config.Host, a.config.TLSPort)

	// Configure FastHTTP server
	a.server = &fasthttp.Server{
		Handler:            a.handler,
		ReadTimeout:        a.config.ReadTimeout,
		WriteTimeout:       a.config.WriteTimeout,
		MaxRequestBodySize: a.config.MaxRequestBodySize,
		Concurrency:        a.config.Concurrency,
	}

	// Start HTTP redirect server if needed
	if a.config.RedirectHTTPToTLS && a.config.EnableTLS {
		a.startHTTPRedirectServer()
		addr = tlsAddr // Use TLS port for main server
	}

	// Start the appropriate server
	if a.config.EnableHTTP2 && a.http2Server != nil {
		// HTTP/2 Server
		a.http2Server.SetFastHTTPHandler(a.handler)

		if a.config.EnableTLS && a.tlsConfig != nil {
			log.Printf("🚀 Blaze HTTP/2 server starting with TLS on https://%s", addr)
			return a.http2Server.ListenAndServe(addr)
		} else if a.http2Config.H2C {
			log.Printf("🚀 Blaze HTTP/2 server (h2c) starting on http://%s", addr)
			return a.http2Server.ListenAndServe(addr)
		}
	}

	// FastHTTP Server (HTTP/1.1)
	if a.config.EnableTLS && a.tlsConfig != nil {
		// Configure TLS for FastHTTP
		if err := a.tlsConfig.ConfigureFastHTTPTLS(a.server); err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}

		log.Printf("🔒 Blaze server starting with TLS on https://%s", addr)
		return a.server.ListenAndServeTLS(addr, a.tlsConfig.CertFile, a.tlsConfig.KeyFile)
	} else {
		log.Printf("🚀 Blaze server starting on http://%s", addr)
		return a.server.ListenAndServe(addr)
	}
}

// ListenAndServeGraceful starts the server with automatic graceful shutdown handling.
//
// This method combines server startup with signal handling for graceful shutdown.
// It listens for OS signals (SIGINT, SIGTERM by default) and automatically
// initiates graceful shutdown when received.
//
// Features:
// - Automatic signal handling (Ctrl+C, system shutdown)
// - Configurable shutdown timeout (default 30 seconds)
// - Comprehensive error handling for both startup and shutdown
// - Supports custom signal handling
//
// The method blocks until:
// 1. Server startup fails (returns startup error)
// 2. Shutdown signal received (performs graceful shutdown)
// 3. Server runtime error (returns runtime error)
//
// Parameters:
//
//	signals: Optional list of OS signals to handle (default: SIGINT, SIGTERM)
//
// Returns:
//
//	error: Server startup, runtime, or shutdown error
//
// Example:
//
//	// Start with default signals (SIGINT, SIGTERM)
//	if err := app.ListenAndServeGraceful(); err != nil {
//	    log.Printf("Server error: %v", err)
//	}
//
//	// Start with custom signals
//	if err := app.ListenAndServeGraceful(syscall.SIGUSR1); err != nil {
//	    log.Printf("Server error: %v", err)
//	}
func (a *App) ListenAndServeGraceful(signals ...os.Signal) error {
	// Default signals if none provided
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	// Channel to receive server errors
	serverError := make(chan error, 1)

	// Start server in goroutine
	go func() {
		err := a.ListenAndServe()
		if err != nil && !a.IsShuttingDown() {
			serverError <- err
		}
	}()

	// Channel to receive OS signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, signals...)

	// Wait for either server error or shutdown signal
	select {
	case err := <-serverError:
		return fmt.Errorf("server error: %w", err)
	case sig := <-signalChan:
		log.Printf("📡 Received shutdown signal: %v", sig)
	}

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Perform graceful shutdown
	return a.Shutdown(shutdownCtx)
}

// Use adds middleware to the application's global middleware stack.
//
// Middleware is executed in the order it's added (first added = first executed).
// However, the middleware stack is built in reverse order, so the last added
// middleware wraps the first added middleware.
//
// Middleware execution flow:
//
//	Request -> Middleware N -> ... -> Middleware 2 -> Middleware 1 -> Handler
//	Response <- Middleware N <- ... <- Middleware 2 <- Middleware 1 <- Handler
//
// Common middleware types:
// - Logger: Request/response logging
// - Recovery: Panic recovery and error handling
// - CORS: Cross-origin resource sharing
// - Authentication: Bearer token validation
// - Rate Limiting: Request throttling
// - Compression: Response compression
//
// Parameters:
//
//	middleware: Function that wraps the next handler in the chain
//
// Returns:
//
//	*App: The app instance for method chaining
//
// Example:
//
//	app.Use(blaze.Logger()).
//	    Use(blaze.Recovery()).
//	    Use(blaze.CORS()).
//	    Use(customMiddleware)
func (a *App) Use(middleware MiddlewareFunc) *App {
	a.middleware = append(a.middleware, middleware)
	return a
}

// GET registers a GET route for retrieving resources.
//
// GET requests should be:
// - Idempotent (multiple calls have the same effect)
// - Cacheable (can be cached by browsers and proxies)
// - Safe (no side effects on server state)
//
// Common use cases:
// - Retrieving data: "/api/users/:id"
// - Listing resources: "/api/users"
// - Health checks: "/health"
// - Static files: "/assets/*filepath"
//
// Parameters:
//
//	path: URL pattern with optional parameters
//	handler: Function to process the GET request
//	options: Optional route configuration
//
// Returns:
//
//	*App: The app instance for method chaining
//
// Example:
//
//	// Simple GET route
//	app.GET("/", func(c blaze.Context) error {
//	    return c.JSON(blaze.Map{"message": "Hello World"})
//	})
//
//	// GET route with parameters
//	app.GET("/users/:id", func(c blaze.Context) error {
//	    userID := c.Param("id")
//	    return c.JSON(blaze.Map{"user_id": userID})
//	})
func (a *App) GET(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("GET", path, handler, options...)
	return a
}

// POST registers a POST route for creating new resources.
//
// POST requests are typically used for:
// - Creating new resources: "/api/users"
// - Submitting forms: "/contact"
// - Uploading files: "/api/upload"
// - Non-idempotent operations: "/api/send-email"
//
// POST requests:
// - Can modify server state
// - Are not cacheable by default
// - Can include request body (JSON, form data, files)
// - Should return appropriate status codes (201 for creation)
//
// Parameters:
//
//	path: URL pattern with optional parameters
//	handler: Function to process the POST request
//	options: Optional route configuration
//
// Returns:
//
//	*App: The app instance for method chaining
func (a *App) POST(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("POST", path, handler, options...)
	return a
}

// PUT registers a PUT route for updating/replacing entire resources.
//
// PUT requests are typically used for:
// - Replacing entire resources: "/api/users/:id"
// - Idempotent updates (same effect when repeated)
// - Creating resources with known IDs: "/api/users/123"
//
// PUT semantics:
// - Should replace the entire resource
// - Idempotent (multiple calls have same effect)
// - Can create or update resources
// - Request body should contain complete resource representation
func (a *App) PUT(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("PUT", path, handler, options...)
	return a
}

// DELETE registers a DELETE route for removing resources.
//
// DELETE requests are typically used for:
// - Removing specific resources: "/api/users/:id"
// - Bulk deletion: "/api/users" (with query parameters)
// - Cleanup operations: "/api/cache/clear"
//
// DELETE semantics:
// - Idempotent (deleting non-existent resource is not an error)
// - Should return appropriate status codes (204 for success, 404 for not found)
// - May include request body for bulk operations
func (a *App) DELETE(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("DELETE", path, handler, options...)
	return a
}

// PATCH registers a PATCH route for partial resource updates.
//
// PATCH requests are typically used for:
// - Partial updates: "/api/users/:id"
// - Field-specific modifications: "/api/users/:id/email"
// - Status changes: "/api/orders/:id/status"
//
// PATCH semantics:
// - Applies partial modifications to resources
// - Request body contains only fields to be updated
// - More efficient than PUT for large resources
// - Should validate that partial update is valid
func (a *App) PATCH(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("PATCH", path, handler, options...)
	return a
}

// OPTIONS registers an OPTIONS route for CORS preflight requests and API discovery.
//
// OPTIONS requests are typically used for:
// - CORS preflight requests (automatic browser behavior)
// - API capability discovery: "What methods are supported?"
// - Server feature detection
//
// OPTIONS responses should include:
// - Allow header with supported methods
// - CORS headers for cross-origin requests
// - API documentation or capabilities
func (a *App) OPTIONS(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("OPTIONS", path, handler, options...)
	return a
}

// HEAD registers a HEAD route for retrieving resource metadata without body.
//
// HEAD requests are typically used for:
// - Checking resource existence without downloading content
// - Getting response headers (Content-Length, Last-Modified, etc.)
// - Conditional requests (If-Modified-Since validation)
// - Bandwidth-efficient resource inspection
//
// HEAD responses:
// - Should include same headers as corresponding GET request
// - Must not include response body
// - Should be as efficient as possible (avoid expensive operations)
func (a *App) HEAD(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("HEAD", path, handler, options...)
	return a
}

// CONNECT registers a CONNECT route for establishing tunnels (e.g., HTTPS proxies).
//
// CONNECT requests are typically used for:
// - Establishing tunnels through HTTP proxies
// - Enabling HTTPS connections via proxy servers
// - Creating secure connections for non-HTTP protocols
//
// CONNECT semantics:
// - Request includes target host and port (e.g., "example.com:443")
// - Server should establish a TCP tunnel to the target
// - After successful connection, raw data is forwarded bidirectionally
// - Should handle errors gracefully (e.g., connection failures)
func (a *App) CONNECT(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("CONNECT", path, handler, options...)
	return a
}

// TRACE registers a TRACE route for diagnostic purposes.
//
// TRACE requests are typically used for:
// - Echoing received requests for debugging
// - Diagnosing network issues
// - Verifying request paths and headers
//
// TRACE semantics:
// - Server should respond with the exact request received
// - Response must include all request headers and body
// - Should be used cautiously (can expose sensitive information)
// - Often disabled in production environments for security
func (a *App) TRACE(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("TRACE", path, handler, options...)
	return a
}

// ANY registers a route for all standard HTTP methods.
//
// ANY is a convenience method that registers the same handler for:
// - GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD, CONNECT, TRACE
//
// Use cases for ANY:
// - Catch-all routes (e.g., 404 handlers)
// - Proxying requests to another service
// - Dynamic routing based on request content
//
// Note: Using ANY can make it harder to reason about route behavior.
// Prefer specific methods when possible for clarity and intent.
func (a *App) ANY(path string, handler HandlerFunc, options ...RouteOption) *App {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD", "CONNECT", "TRACE"}
	for _, method := range methods {
		a.router.AddRoute(method, path, handler, options...)
	}
	return a
}

// Match registers a route for multiple specified HTTP methods.
//
// Match allows registering the same handler for a custom set of methods.
// This is useful for:
// - Grouping related methods (e.g., GET and HEAD)
// - Implementing RESTful endpoints with multiple actions
// - Reducing boilerplate when the same logic applies to several methods
//
// Parameters:
//
//	methods: List of HTTP methods to register (e.g., []string{"GET", "POST"})
func (a *App) Match(methods []string, path string, handler HandlerFunc, options ...RouteOption) *App {
	for _, method := range methods {
		a.router.AddRoute(method, path, handler, options...)
	}
	return a
}

// WebSocket upgrades HTTP connection to WebSocket with default configuration.
//
// WebSocket connections enable real-time, bidirectional communication between
// client and server. Common use cases include:
// - Real-time chat applications
// - Live data feeds (stock prices, sports scores)
// - Collaborative editing (like Google Docs)
// - Gaming applications
// - IoT device communication
//
// The WebSocket handler receives a WebSocketConnection that provides methods
// for reading and writing messages in both text and binary formats.
//
// Parameters:
//
//	path: URL pattern for the WebSocket endpoint
//	handler: Function to handle WebSocket connections
//	options: Optional route configuration
//
// Returns:
//
//	*App: The app instance for method chaining
//
// Example:
//
//	app.WebSocket("/ws", func(conn blaze.WebSocketConnection) error {
//	    for {
//	        msgType, data, err := conn.ReadMessage()
//	        if err != nil {
//	            return err
//	        }
//
//	        // Echo message back to client
//	        if err := conn.WriteMessage(msgType, data); err != nil {
//	            return err
//	        }
//	    }
//	})
func (a *App) WebSocket(path string, handler WebSocketHandler, options ...RouteOption) *App {
	upgrader := NewWebSocketUpgrader()

	wsHandler := func(c *Context) error {
		return upgrader.Upgrade(c, handler)
	}

	a.router.AddRoute("GET", path, wsHandler, options...)
	return a
}

// WebSocketWithConfig upgrades HTTP connection to WebSocket with custom configuration.
//
// This method allows fine-tuning of WebSocket behavior including:
// - Message size limits
// - Compression settings
// - Subprotocol negotiation
// - Origin validation
// - Custom upgrade headers
//
// Parameters:
//
//	path: URL pattern for the WebSocket endpoint
//	handler: Function to handle WebSocket connections
//	config: Custom WebSocket configuration
//	options: Optional route configuration
//
// Returns:
//
//	*App: The app instance for method chaining
//
// Example:
//
//	wsConfig := blaze.WebSocketConfig{
//	    MaxMessageSize: 1024 * 1024, // 1MB
//	    EnableCompression: true,
//	    Subprotocols: []string{"chat", "echo"},
//	}
//
//	app.WebSocketWithConfig("/ws", handler, wsConfig)
func (a *App) WebSocketWithConfig(path string, handler WebSocketHandler, config *WebSocketConfig, options ...RouteOption) *App {
	upgrader := NewWebSocketUpgrader(config)

	wsHandler := func(c *Context) error {
		return upgrader.Upgrade(c, handler)
	}

	a.router.AddRoute("GET", path, wsHandler, options...)
	return a
}

// Group creates a route group with shared prefix and middleware.
//
// Route groups allow organizing related routes under a common prefix
// and applying shared middleware without affecting other routes.
//
// Features:
// - Shared URL prefix for all routes in the group
// - Group-specific middleware stack
// - Nested groups (groups can contain other groups)
// - Independent of global middleware
//
// Parameters:
//
//	prefix: URL prefix for all routes in this group
//
// Returns:
//
//	*Group: New route group instance
//
// Example:
//
//	// API v1 group with authentication
//	v1 := app.Group("/api/v1")
//	v1.Use(AuthMiddleware())
//	v1.GET("/users", getUsersHandler)
//	v1.POST("/users", createUserHandler)
//
//	// Admin group with additional authorization
//	admin := v1.Group("/admin")
//	admin.Use(AdminMiddleware())
//	admin.GET("/stats", getStatsHandler)
//	admin.DELETE("/users/:id", deleteUserHandler)
func (a *App) Group(prefix string, configure ...func(*Group)) *Group {
	group := &Group{
		app:        a,
		prefix:     prefix,
		middleware: make([]MiddlewareFunc, 0),
	}

	// Apply configuration if provided
	for _, cfg := range configure {
		cfg(group)
	}

	return group
}

// func (a *App) Group(prefix string) *Group {
// 	return &Group{
// 		app:        a,
// 		prefix:     prefix,
// 		middleware: make([]MiddlewareFunc, 0),
// 	}
// }

// handler is the main request handler that applies middleware and routing
func (a *App) handler(ctx *fasthttp.RequestCtx) {
	// Check if server is shutting down
	if a.IsShuttingDown() {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		ctx.SetBody([]byte("Server is shutting down"))
		return
	}

	blazeCtx := &Context{
		RequestCtx: ctx,
		params:     make(map[string]string),
		locals:     make(map[string]interface{}),
	}

	// Set shutdown context in locals
	blazeCtx.SetLocals("shutdown_ctx", a.shutdownCtx)

	var handler HandlerFunc
	var err error

	// Use advanced router
	route, params, found := a.router.FindRoute(
		string(ctx.Method()),
		string(ctx.Path()),
	)

	if !found {
		handler = func(c *Context) error {
			return c.Status(404).JSON(Map{"error": "Not Found"})
		}
	} else {
		// Set route parameters
		for key, value := range params {
			blazeCtx.SetParam(key, value)
		}

		// Apply route-specific middleware
		handler = route.Handler
		for i := len(route.Middleware) - 1; i >= 0; i-- {
			handler = route.Middleware[i](handler)
		}
	}

	// Apply global middleware
	for i := len(a.middleware) - 1; i >= 0; i-- {
		handler = a.middleware[i](handler)
	}

	// Execute handler
	if err = handler(blazeCtx); err != nil {
		blazeCtx.Status(500).JSON(Map{"error": err.Error()})
	}
}

// GetServerInfo returns server information including TLS and HTTP/2 status
func (a *App) GetServerInfo() *ServerInfo {
	info := &ServerInfo{
		Host:        a.config.Host,
		Port:        a.config.Port,
		TLSPort:     a.config.TLSPort,
		EnableTLS:   a.config.EnableTLS,
		EnableHTTP2: a.config.EnableHTTP2,
		Development: a.config.Development,
	}

	if a.tlsConfig != nil {
		info.TLS = a.tlsConfig.GetTLSHealthCheck()
	}

	if a.http2Server != nil {
		info.HTTP2 = a.http2Server.GetHTTP2HealthCheck()
	}

	return info
}

// ServerInfo holds server configuration and status information
type ServerInfo struct {
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	TLSPort     int               `json:"tls_port,omitempty"`
	EnableTLS   bool              `json:"enable_tls"`
	EnableHTTP2 bool              `json:"enable_http2"`
	Development bool              `json:"development"`
	TLS         *TLSHealthCheck   `json:"tls,omitempty"`
	HTTP2       *HTTP2HealthCheck `json:"http2,omitempty"`
}

// Group represents a route group
type Group struct {
	app        *App
	prefix     string
	middleware []MiddlewareFunc
	parent     *Group // For nested groups
}

// Use adds middleware to the group
func (g *Group) Use(middleware MiddlewareFunc) *Group {
	g.middleware = append(g.middleware, middleware)
	return g
}

// Group creates a nested group
func (g *Group) Group(prefix string, configure ...func(*Group)) *Group {
	nestedGroup := &Group{
		app:        g.app,
		prefix:     g.prefix + prefix,
		middleware: make([]MiddlewareFunc, len(g.middleware)),
		parent:     g,
	}

	// Inherit parent middleware
	copy(nestedGroup.middleware, g.middleware)

	// Apply configuration if provided
	for _, cfg := range configure {
		cfg(nestedGroup)
	}

	return nestedGroup
}

// GET registers a GET route in the group
func (g *Group) GET(path string, handler HandlerFunc, options ...RouteOption) *Group {
	fullPath := g.prefix + path
	wrappedHandler := g.wrapHandler(handler)
	g.app.router.AddRoute("GET", fullPath, wrappedHandler, options...)
	return g
}

// POST registers a POST route in the group
func (g *Group) POST(path string, handler HandlerFunc, options ...RouteOption) *Group {
	fullPath := g.prefix + path
	wrappedHandler := g.wrapHandler(handler)
	g.app.router.AddRoute("POST", fullPath, wrappedHandler, options...)
	return g
}

// PUT registers a PUT route in the group
func (g *Group) PUT(path string, handler HandlerFunc, options ...RouteOption) *Group {
	fullPath := g.prefix + path
	wrappedHandler := g.wrapHandler(handler)
	g.app.router.AddRoute("PUT", fullPath, wrappedHandler, options...)
	return g
}

// DELETE registers a DELETE route in the group
func (g *Group) DELETE(path string, handler HandlerFunc, options ...RouteOption) *Group {
	fullPath := g.prefix + path
	wrappedHandler := g.wrapHandler(handler)
	g.app.router.AddRoute("DELETE", fullPath, wrappedHandler, options...)
	return g
}

// PATCH registers a PATCH route in the group
func (g *Group) PATCH(path string, handler HandlerFunc, options ...RouteOption) *Group {
	fullPath := g.prefix + path
	wrappedHandler := g.wrapHandler(handler)
	g.app.router.AddRoute("PATCH", fullPath, wrappedHandler, options...)
	return g
}

// New HTTP methods for groups
func (g *Group) CONNECT(path string, handler HandlerFunc, options ...RouteOption) *Group {
	fullPath := g.prefix + path
	wrappedHandler := g.wrapHandler(handler)
	g.app.router.AddRoute("CONNECT", fullPath, wrappedHandler, options...)
	return g
}

func (g *Group) TRACE(path string, handler HandlerFunc, options ...RouteOption) *Group {
	fullPath := g.prefix + path
	wrappedHandler := g.wrapHandler(handler)
	g.app.router.AddRoute("TRACE", fullPath, wrappedHandler, options...)
	return g
}

// ANY registers a route for all HTTP methods in the group
func (g *Group) ANY(path string, handler HandlerFunc, options ...RouteOption) *Group {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD", "CONNECT", "TRACE"}
	fullPath := g.prefix + path
	wrappedHandler := g.wrapHandler(handler)
	for _, method := range methods {
		g.app.router.AddRoute(method, fullPath, wrappedHandler, options...)
	}
	return g
}

// Match registers a route for specific HTTP methods in the group
func (g *Group) Match(methods []string, path string, handler HandlerFunc, options ...RouteOption) *Group {
	fullPath := g.prefix + path
	wrappedHandler := g.wrapHandler(handler)
	for _, method := range methods {
		g.app.router.AddRoute(method, fullPath, wrappedHandler, options...)
	}
	return g
}

// WebSocket registers a WebSocket route in the group
func (g *Group) WebSocket(path string, handler WebSocketHandler, options ...RouteOption) *Group {
	fullPath := g.prefix + path
	upgrader := NewWebSocketUpgrader()

	wsHandler := func(c *Context) error {
		return upgrader.Upgrade(c, handler)
	}

	wrappedHandler := g.wrapHandler(wsHandler)
	g.app.router.AddRoute("GET", fullPath, wrappedHandler, options...)
	return g
}

// WebSocketWithConfig registers a WebSocket route with custom config in the group
func (g *Group) WebSocketWithConfig(path string, handler WebSocketHandler, config *WebSocketConfig, options ...RouteOption) *Group {
	fullPath := g.prefix + path
	upgrader := NewWebSocketUpgrader(config)

	wsHandler := func(c *Context) error {
		return upgrader.Upgrade(c, handler)
	}

	wrappedHandler := g.wrapHandler(wsHandler)
	g.app.router.AddRoute("GET", fullPath, wrappedHandler, options...)
	return g
}

// wrapHandler applies group middleware to the handler
func (g *Group) wrapHandler(handler HandlerFunc) HandlerFunc {
	for i := len(g.middleware) - 1; i >= 0; i-- {
		handler = g.middleware[i](handler)
	}
	return handler
}

// GetAllMiddleware returns all middleware including inherited from parents
func (g *Group) GetAllMiddleware() []MiddlewareFunc {
	var allMiddleware []MiddlewareFunc

	if g.parent != nil {
		allMiddleware = append(allMiddleware, g.parent.GetAllMiddleware()...)
	}

	allMiddleware = append(allMiddleware, g.middleware...)
	return allMiddleware
}

// Apply applies a configuration function to the group (for fluent API)
func (g *Group) Apply(configure func(*Group)) *Group {
	configure(g)
	return g
}
