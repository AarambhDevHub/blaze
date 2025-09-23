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

// App represents the main application instance
type App struct {
	router      *Router
	middleware  []MiddlewareFunc
	server      *fasthttp.Server
	config      *Config
	tlsConfig   *TLSConfig
	http2Config *HTTP2Config
	http2Server *HTTP2Server

	// Graceful shutdown fields
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
	shutdownWg     sync.WaitGroup
	shutdownOnce   sync.Once
	isShuttingDown bool
	mu             sync.RWMutex
}

// Config holds application configuration
type Config struct {
	Host               string
	Port               int
	TLSPort            int // Separate port for HTTPS
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

// DefaultConfig returns default configuration
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

// ProductionConfig returns production-ready configuration
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

// DevelopmentConfig returns development configuration
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

// New creates a new Blaze application
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

// NewWithConfig creates a new Blaze application with custom configuration
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

// SetTLSConfig sets the TLS configuration
func (a *App) SetTLSConfig(config *TLSConfig) *App {
	a.tlsConfig = config
	a.config.EnableTLS = config != nil

	// Update HTTP/2 server if exists
	if a.http2Server != nil {
		a.http2Server = NewHTTP2Server(a.http2Config, a.tlsConfig)
	}

	return a
}

// SetHTTP2Config sets the HTTP/2 configuration
func (a *App) SetHTTP2Config(config *HTTP2Config) *App {
	a.http2Config = config
	a.config.EnableHTTP2 = config != nil && config.Enabled

	// Create or update HTTP/2 server
	if a.config.EnableHTTP2 {
		a.http2Server = NewHTTP2Server(a.http2Config, a.tlsConfig)
	}

	return a
}

// EnableAutoTLS enables automatic TLS with self-signed certificates for development
func (a *App) EnableAutoTLS(domains ...string) *App {
	if len(domains) == 0 {
		domains = []string{"localhost", "127.0.0.1"}
	}

	tlsConfig := DevelopmentTLSConfig()
	tlsConfig.Domains = domains

	return a.SetTLSConfig(tlsConfig)
}

// IsShuttingDown returns true if the app is in shutdown process
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

// GetShutdownContext returns the shutdown context
func (a *App) GetShutdownContext() context.Context {
	return a.shutdownCtx
}

// RegisterGracefulTask registers a task for graceful shutdown
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

// Shutdown gracefully shuts down the server
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

// startHTTPRedirectServer starts a server to redirect HTTP to HTTPS
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

// ListenAndServeGraceful starts the server with automatic graceful shutdown handling
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

// Use adds middleware to the application
func (a *App) Use(middleware MiddlewareFunc) *App {
	a.middleware = append(a.middleware, middleware)
	return a
}

// GET registers a GET route
func (a *App) GET(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("GET", path, handler, options...)
	return a
}

// POST registers a POST route
func (a *App) POST(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("POST", path, handler, options...)
	return a
}

// PUT registers a PUT route
func (a *App) PUT(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("PUT", path, handler, options...)
	return a
}

// DELETE registers a DELETE route
func (a *App) DELETE(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("DELETE", path, handler, options...)
	return a
}

// PATCH registers a PATCH route
func (a *App) PATCH(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("PATCH", path, handler, options...)
	return a
}

// OPTIONS registers an OPTIONS route
func (a *App) OPTIONS(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("OPTIONS", path, handler, options...)
	return a
}

// HEAD registers a HEAD route
func (a *App) HEAD(path string, handler HandlerFunc, options ...RouteOption) *App {
	a.router.AddRoute("HEAD", path, handler, options...)
	return a
}

// WebSocket upgrades HTTP connection to WebSocket
func (a *App) WebSocket(path string, handler WebSocketHandler, options ...RouteOption) *App {
	upgrader := NewWebSocketUpgrader()

	wsHandler := func(c *Context) error {
		return upgrader.Upgrade(c, handler)
	}

	a.router.AddRoute("GET", path, wsHandler, options...)
	return a
}

// WebSocketWithConfig upgrades HTTP connection to WebSocket with custom config
func (a *App) WebSocketWithConfig(path string, handler WebSocketHandler, config *WebSocketConfig, options ...RouteOption) *App {
	upgrader := NewWebSocketUpgrader(config)

	wsHandler := func(c *Context) error {
		return upgrader.Upgrade(c, handler)
	}

	a.router.AddRoute("GET", path, wsHandler, options...)
	return a
}

// Group creates a route group with shared prefix and middleware
func (a *App) Group(prefix string) *Group {
	return &Group{
		app:        a,
		prefix:     prefix,
		middleware: make([]MiddlewareFunc, 0),
	}
}

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
}

// Use adds middleware to the group
func (g *Group) Use(middleware MiddlewareFunc) *Group {
	g.middleware = append(g.middleware, middleware)
	return g
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
