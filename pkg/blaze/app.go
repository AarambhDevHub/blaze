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
	router     *Router
	middleware []MiddlewareFunc
	server     *fasthttp.Server
	config     *Config

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
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	MaxRequestBodySize int
	Concurrency        int
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Host:               "127.0.0.1",
		Port:               8080,
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       10 * time.Second,
		MaxRequestBodySize: 4 * 1024 * 1024, // 4MB
		Concurrency:        256 * 1024,
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

// Update the NewWithConfig function
func NewWithConfig(config *Config) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		router:         NewRouter(),
		middleware:     make([]MiddlewareFunc, 0),
		config:         config,
		shutdownCtx:    ctx,
		shutdownCancel: cancel,
	}
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

			// Shutdown the server
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
			if a.server != nil {
				log.Println("🚨 Forcing server shutdown...")
				a.server.Shutdown() // Force shutdown
			}
		}
	})

	return shutdownErr
}

// ListenWithGracefulShutdown starts the server with automatic graceful shutdown handling
func (a *App) ListenWithGracefulShutdown(signals ...os.Signal) error {
	return a.listenWithGracefulShutdown("", "", signals...)
}

// ListenTLSWithGracefulShutdown starts the TLS server with automatic graceful shutdown handling
func (a *App) ListenTLSWithGracefulShutdown(certFile, keyFile string, signals ...os.Signal) error {
	return a.listenWithGracefulShutdown(certFile, keyFile, signals...)
}

// listenWithGracefulShutdown is the internal implementation for graceful server startup and shutdown
func (a *App) listenWithGracefulShutdown(certFile, keyFile string, signals ...os.Signal) error {
	// Default signals if none provided
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	// Setup server
	addr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)
	a.server = &fasthttp.Server{
		Handler:            a.handler,
		ReadTimeout:        a.config.ReadTimeout,
		WriteTimeout:       a.config.WriteTimeout,
		MaxRequestBodySize: a.config.MaxRequestBodySize,
		Concurrency:        a.config.Concurrency,
	}

	// Channel to receive server errors
	serverError := make(chan error, 1)

	// Start server in goroutine
	go func() {
		var err error
		if certFile != "" && keyFile != "" {
			log.Printf("🔒 Blaze server starting with TLS on https://%s", addr)
			err = a.server.ListenAndServeTLS(addr, certFile, keyFile)
		} else {
			log.Printf("🚀 Blaze server starting on http://%s", addr)
			err = a.server.ListenAndServe(addr)
		}

		// Only send error if it's not from graceful shutdown
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

// // Route registers a route for multiple methods
// func (a *App) Route(methods []string, path string, handler HandlerFunc, options ...RouteOption) *App {
// 	for _, method := range methods , options...{
// 		a.router.Add(method, path, handler)
// 	}
// 	return a
// }

// Group creates a route group with shared prefix and middleware
func (a *App) Group(prefix string) *Group {
	return &Group{
		app:        a,
		prefix:     prefix,
		middleware: make([]MiddlewareFunc, 0),
	}
}

// Listen starts the server on the configured address
func (a *App) Listen() error {
	addr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)

	a.server = &fasthttp.Server{
		Handler:            a.handler,
		ReadTimeout:        a.config.ReadTimeout,
		WriteTimeout:       a.config.WriteTimeout,
		MaxRequestBodySize: a.config.MaxRequestBodySize,
		Concurrency:        a.config.Concurrency,
	}

	log.Printf("🚀 Blaze server starting on http://%s", addr)
	return a.server.ListenAndServe(addr)
}

// ListenTLS starts the server with TLS
func (a *App) ListenTLS(certFile, keyFile string) error {
	addr := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)

	a.server = &fasthttp.Server{
		Handler:            a.handler,
		ReadTimeout:        a.config.ReadTimeout,
		WriteTimeout:       a.config.WriteTimeout,
		MaxRequestBodySize: a.config.MaxRequestBodySize,
		Concurrency:        a.config.Concurrency,
	}

	log.Printf("🔒 Blaze server starting with TLS on https://%s", addr)
	return a.server.ListenAndServeTLS(addr, certFile, keyFile)
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

// wrapHandler applies group middleware to the handler
func (g *Group) wrapHandler(handler HandlerFunc) HandlerFunc {
	for i := len(g.middleware) - 1; i >= 0; i-- {
		handler = g.middleware[i](handler)
	}
	return handler
}

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
