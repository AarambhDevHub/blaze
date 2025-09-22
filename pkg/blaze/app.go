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

// Shutdown gracefully shuts down the server
func (a *App) Shutdown() error {
	if a.server != nil {
		return a.server.Shutdown()
	}
	return nil
}

// handler is the main request handler that applies middleware and routing
// Updated handler method to use advanced router when enabled
func (a *App) handler(ctx *fasthttp.RequestCtx) {
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
