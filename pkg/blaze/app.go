package blaze

import (
    "fmt"
    "log"
    "time"

    "github.com/valyala/fasthttp"
)

// App represents the main application instance
type App struct {
    router     *Router
    middleware []MiddlewareFunc
    server     *fasthttp.Server
    config     *Config
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
    return &App{
        router:     NewRouter(),
        middleware: make([]MiddlewareFunc, 0),
        config:     DefaultConfig(),
    }
}

// NewWithConfig creates a new Blaze application with custom config
func NewWithConfig(config *Config) *App {
    return &App{
        router:     NewRouter(),
        middleware: make([]MiddlewareFunc, 0),
        config:     config,
    }
}

// Use adds middleware to the application
func (a *App) Use(middleware MiddlewareFunc) *App {
    a.middleware = append(a.middleware, middleware)
    return a
}

// GET registers a GET route
func (a *App) GET(path string, handler HandlerFunc) *App {
    a.router.GET(path, handler)
    return a
}

// POST registers a POST route
func (a *App) POST(path string, handler HandlerFunc) *App {
    a.router.POST(path, handler)
    return a
}

// PUT registers a PUT route
func (a *App) PUT(path string, handler HandlerFunc) *App {
    a.router.PUT(path, handler)
    return a
}

// DELETE registers a DELETE route
func (a *App) DELETE(path string, handler HandlerFunc) *App {
    a.router.DELETE(path, handler)
    return a
}

// PATCH registers a PATCH route
func (a *App) PATCH(path string, handler HandlerFunc) *App {
    a.router.PATCH(path, handler)
    return a
}

// Route registers a route for multiple methods
func (a *App) Route(methods []string, path string, handler HandlerFunc) *App {
    for _, method := range methods {
        a.router.Add(method, path, handler)
    }
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

// Shutdown gracefully shuts down the server
func (a *App) Shutdown() error {
    if a.server != nil {
        return a.server.Shutdown()
    }
    return nil
}

// handler is the main request handler that applies middleware and routing
func (a *App) handler(ctx *fasthttp.RequestCtx) {
    blazeCtx := &Context{
        RequestCtx: ctx,
        params:     make(map[string]string),
        locals:     make(map[string]interface{}),
    }

    // Apply middleware chain
    handler := a.router.Handler()
    for i := len(a.middleware) - 1; i >= 0; i-- {
        handler = a.middleware[i](handler)
    }

    // Execute handler
    if err := handler(blazeCtx); err != nil {
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
func (g *Group) GET(path string, handler HandlerFunc) *Group {
    fullPath := g.prefix + path
    wrappedHandler := g.wrapHandler(handler)
    g.app.router.GET(fullPath, wrappedHandler)
    return g
}

// POST registers a POST route in the group
func (g *Group) POST(path string, handler HandlerFunc) *Group {
    fullPath := g.prefix + path
    wrappedHandler := g.wrapHandler(handler)
    g.app.router.POST(fullPath, wrappedHandler)
    return g
}

// PUT registers a PUT route in the group
func (g *Group) PUT(path string, handler HandlerFunc) *Group {
    fullPath := g.prefix + path
    wrappedHandler := g.wrapHandler(handler)
    g.app.router.PUT(fullPath, wrappedHandler)
    return g
}

// DELETE registers a DELETE route in the group
func (g *Group) DELETE(path string, handler HandlerFunc) *Group {
    fullPath := g.prefix + path
    wrappedHandler := g.wrapHandler(handler)
    g.app.router.DELETE(fullPath, wrappedHandler)
    return g
}

// wrapHandler applies group middleware to the handler
func (g *Group) wrapHandler(handler HandlerFunc) HandlerFunc {
    for i := len(g.middleware) - 1; i >= 0; i-- {
        handler = g.middleware[i](handler)
    }
    return handler
}
