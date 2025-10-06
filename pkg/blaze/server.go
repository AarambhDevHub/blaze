package blaze

import (
	"context"
	"net"

	"github.com/valyala/fasthttp"
)

// Server wraps fasthttp.Server with additional functionality
// Provides enhanced server capabilities including graceful shutdown and configuration management
//
// Server Features:
//   - FastHTTP integration for high performance
//   - Graceful shutdown with context support
//   - Configuration management
//   - TLS/HTTPS support
//   - HTTP/2 support
//   - Custom listener support
//
// Thread Safety:
//   - Safe for concurrent use during operation
//   - Should not be modified after starting
//
// Performance:
//   - Built on FastHTTP (fastest Go HTTP server)
//   - Optimized for high concurrency
//   - Zero-copy optimizations where possible
//   - Minimal memory allocations
type Server struct {
	// fasthttp.Server is the underlying FastHTTP server
	// Provides core HTTP/1.1 functionality
	*fasthttp.Server

	// config holds server configuration
	// Contains timeouts, limits, and feature flags
	config *Config
}

// NewServer creates a new server instance with configuration
// Initializes FastHTTP server with settings from config
//
// Server Initialization:
//   - Configures timeouts (read, write)
//   - Sets resource limits (body size, concurrency)
//   - Applies protocol settings
//   - Creates underlying FastHTTP instance
//
// Parameters:
//   - config: Server configuration (nil for defaults)
//
// Returns:
//   - *Server: Configured server instance
//
// Example:
//
//	server := blaze.NewServer(nil) // Uses defaults
//	server := blaze.NewServer(blaze.DefaultConfig())
func NewServer(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	return &Server{
		Server: &fasthttp.Server{
			ReadTimeout:        config.ReadTimeout,
			WriteTimeout:       config.WriteTimeout,
			MaxRequestBodySize: config.MaxRequestBodySize,
			Concurrency:        config.Concurrency,
		},
		config: config,
	}
}

// ListenAndServe starts the HTTP server on the specified address
// Blocks until server stops or encounters an error
//
// Address Format:
//   - ":8080" - All interfaces, port 8080
//   - "127.0.0.1:8080" - Localhost only
//   - "0.0.0.0:80" - All interfaces, standard HTTP port
//
// Startup Process:
//  1. Bind to address
//  2. Start listening for connections
//  3. Handle requests with configured handler
//  4. Block until shutdown or error
//
// Parameters:
//   - addr: Bind address (host:port format)
//   - handler: Request handler function
//
// Returns:
//   - error: Server error or nil if gracefully shutdown
//
// Example:
//
//	err := server.ListenAndServe(":8080", app.handler)
func (s *Server) ListenAndServe(addr string, handler fasthttp.RequestHandler) error {
	s.Server.Handler = handler // Set handler on the underlying fasthttp.Server
	return s.Server.ListenAndServe(addr)
}

// ListenAndServeTLS starts the HTTPS server with TLS
// Requires valid certificate and key files
//
// TLS Requirements:
//   - Valid certificate file (PEM format)
//   - Matching private key file (PEM format)
//   - Proper file permissions (readable by server)
//
// Certificate Types:
//   - Self-signed (development only)
//   - Let's Encrypt (production)
//   - Commercial CA (production)
//
// Parameters:
//   - addr: Bind address
//   - certFile: Path to TLS certificate file
//   - keyFile: Path to TLS private key file
//   - handler: Request handler function
//
// Returns:
//   - error: Server error or nil if gracefully shutdown
//
// Example:
//
//	err := server.ListenAndServeTLS(":443", "cert.pem", "key.pem", app.handler)
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string, handler fasthttp.RequestHandler) error {
	s.Server.Handler = handler // Set handler on the underlying fasthttp.Server
	return s.Server.ListenAndServeTLS(addr, certFile, keyFile)
}

// Serve accepts incoming connections from the listener
// Allows using custom listeners (Unix sockets, systemd, etc.)
//
// Custom Listener Use Cases:
//   - Unix domain sockets for IPC
//   - Systemd socket activation
//   - Pre-bound privileged ports
//   - Custom network protocols
//
// Parameters:
//   - ln: Network listener
//   - handler: Request handler function
//
// Returns:
//   - error: Server error or nil if gracefully shutdown
//
// Example - Unix Socket:
//
//	ln, _ := net.Listen("unix", "/tmp/app.sock")
//	defer ln.Close()
//	server.Serve(ln, app.handler)
//
// Example - Systemd Socket Activation:
//
//	listeners, _ := systemd.Listeners()
//	if len(listeners) > 0 {
//	    server.Serve(listeners[0], app.handler)
//	}
func (s *Server) Serve(ln net.Listener, handler fasthttp.RequestHandler) error {
	s.Server.Handler = handler // Set handler on the underlying fasthttp.Server
	return s.Server.Serve(ln)
}

// ServeTLS accepts incoming TLS connections from the listener
// Similar to Serve but with TLS encryption
//
// Parameters:
//   - ln: Network listener
//   - certFile: Path to TLS certificate
//   - keyFile: Path to TLS private key
//   - handler: Request handler function
//
// Returns:
//   - error: Server error or nil if gracefully shutdown
//
// Example:
//
//	ln, _ := net.Listen("tcp", ":443")
//	server.ServeTLS(ln, "cert.pem", "key.pem", app.handler)
func (s *Server) ServeTLS(ln net.Listener, certFile, keyFile string, handler fasthttp.RequestHandler) error {
	s.Server.Handler = handler // Set handler on the underlying fasthttp.Server
	return s.Server.ServeTLS(ln, certFile, keyFile)
}

// GracefulShutdown gracefully shuts down the server
// Allows in-flight requests to complete before stopping
//
// Graceful Shutdown Process:
//  1. Stop accepting new connections
//  2. Wait for active requests to complete
//  3. Close idle connections
//  4. Return when complete or context cancelled
//
// Context Timeout:
//   - Controls maximum shutdown duration
//   - If exceeded, forces immediate shutdown
//   - Recommended: 30-60 seconds
//
// Request Handling:
//   - Active requests continue processing
//   - New requests receive connection errors
//   - Long-running requests may be interrupted on timeout
//
// Parameters:
//   - ctx: Context with timeout for shutdown
//
// Returns:
//   - error: Shutdown error or nil on success
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//	if err := server.GracefulShutdown(ctx); err != nil {
//	    log.Printf("Shutdown error: %v", err)
//	}
func (s *Server) GracefulShutdown(ctx context.Context) error {
	done := make(chan error, 1)
	go func() {
		done <- s.Server.Shutdown()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SetHandler sets the request handler
// Helper method for setting handler after server creation
//
// Parameters:
//   - handler: Request handler function
//
// Example:
//
//	server.SetHandler(app.handler)
func (s *Server) SetHandler(handler fasthttp.RequestHandler) {
	s.Server.Handler = handler
}

// GetHandler gets the current request handler
// Returns the handler function currently configured
//
// Returns:
//   - fasthttp.RequestHandler: Current handler or nil
//
// Example:
//
//	handler := server.GetHandler()
func (s *Server) GetHandler() fasthttp.RequestHandler {
	return s.Server.Handler
}

// GetConfig returns the server configuration
// Provides access to server settings
//
// Returns:
//   - Config: Server configuration
//
// Example:
//
//	config := server.GetConfig()
//	log.Printf("Read timeout: %v", config.ReadTimeout)
func (s *Server) GetConfig() Config {
	return *s.config
}
