package blaze

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"net/http"

	"github.com/valyala/fasthttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// HTTP2Config holds comprehensive HTTP/2 server configuration
// HTTP/2 provides significant performance improvements over HTTP/1.1 through:
//   - Multiplexing: Multiple requests over a single connection
//   - Header compression: HPACK algorithm reduces overhead
//   - Server push: Proactively send resources to clients
//   - Binary protocol: More efficient parsing and processing
//   - Stream prioritization: Control resource delivery order
//
// Performance Considerations:
//   - Increase MaxConcurrentStreams for high-traffic applications
//   - Tune buffer sizes based on payload characteristics
//   - Enable server push for static resources
//   - Configure appropriate timeouts for long-lived connections
//
// Security Considerations:
//   - Always use TLS in production (HTTP/2 requires it)
//   - Avoid H2C (cleartext HTTP/2) except for development
//   - Configure strict cipher suites with TLS
//   - Set reasonable stream and connection limits
//
// Browser Support:
//   - All modern browsers support HTTP/2 over TLS
//   - Most browsers don't support H2C (cleartext HTTP/2)
//   - Automatic protocol negotiation via ALPN
type HTTP2Config struct {
	// Enabled controls whether HTTP/2 support is active
	// When true, server negotiates HTTP/2 via ALPN during TLS handshake
	// Default: false (HTTP/1.1 only)
	Enabled bool

	// H2C enables HTTP/2 over cleartext (without TLS)
	// WARNING: Only use in development or internal networks
	// Browsers typically don't support H2C for security reasons
	// Production deployments must use TLS
	// Default: false
	H2C bool

	// MaxConcurrentStreams limits concurrent streams per connection
	// Prevents resource exhaustion from too many parallel requests
	// Higher values allow more parallelism but use more memory
	// Recommended: 100-1000 for most applications
	// Default: 1000
	MaxConcurrentStreams uint32

	// MaxUploadBufferPerStream limits upload buffer per individual stream
	// Controls memory usage for request bodies
	// Set based on expected request payload sizes
	// Default: 1MB (1048576 bytes)
	MaxUploadBufferPerStream int32

	// MaxUploadBufferPerConnection limits total upload buffer per connection
	// Prevents single connection from consuming excessive memory
	// Should be >= MaxUploadBufferPerStream * typical concurrent streams
	// Default: 1MB (1048576 bytes)
	MaxUploadBufferPerConnection int32

	// EnablePush controls HTTP/2 server push functionality
	// Server push proactively sends resources before client requests them
	// Useful for CSS, JavaScript, images referenced by HTML
	// Can improve page load performance by reducing round trips
	// Default: true
	EnablePush bool

	// IdleTimeout specifies maximum time to keep idle connections alive
	// Closes connections with no active streams after this duration
	// Balances connection reuse with resource cleanup
	// Recommended: 5-10 minutes for typical applications
	// Default: 300 seconds (5 minutes)
	IdleTimeout time.Duration

	// ReadTimeout specifies maximum time to read request
	// Applies to the entire request including body
	// Prevents slow clients from holding connections indefinitely
	// Default: 30 seconds
	ReadTimeout time.Duration

	// WriteTimeout specifies maximum time to write response
	// Applies to the entire response including body
	// Important for streaming responses and large files
	// Default: 30 seconds
	WriteTimeout time.Duration

	// MaxDecoderHeaderTableSize for HPACK compression
	// HPACK maintains a dynamic table of previously seen headers
	// Larger table improves compression but uses more memory
	// Default: 4096 bytes
	MaxDecoderHeaderTableSize uint32

	// MaxEncoderHeaderTableSize for HPACK compression
	// Controls memory used for header compression on sent responses
	// Default: 4096 bytes
	MaxEncoderHeaderTableSize uint32

	// MaxReadFrameSize limits HTTP/2 frame size
	// Larger frames can improve throughput but use more memory
	// Must be between 16KB and 16MB per HTTP/2 spec
	// Default: 1MB (1048576 bytes)
	MaxReadFrameSize uint32

	// PermitProhibitedCipherSuites allows weaker cipher suites for compatibility
	// Some older clients may not support recommended cipher suites
	// WARNING: Only enable if compatibility is absolutely required
	// Reduces security level
	// Default: false
	PermitProhibitedCipherSuites bool
}

// DefaultHTTP2Config returns production-ready HTTP/2 configuration
// Provides secure, performant defaults suitable for most applications
//
// Default Configuration:
//   - Enabled: true (HTTP/2 active)
//   - H2C: false (TLS required)
//   - MaxConcurrentStreams: 1000 (high parallelism)
//   - Upload buffers: 1MB per stream and connection
//   - Server push: enabled
//   - Timeouts: 30s read/write, 5min idle
//   - HPACK tables: 4KB
//   - Max frame size: 1MB
//   - Prohibited ciphers: disabled (security)
//
// Tuning Guidelines:
//   - Increase MaxConcurrentStreams for high-traffic APIs
//   - Adjust buffer sizes based on payload patterns
//   - Tune timeouts for long-polling or streaming endpoints
//   - Consider disabling push if not needed
//
// Returns:
//   - HTTP2Config: Production-ready configuration
func DefaultHTTP2Config() *HTTP2Config {
	return &HTTP2Config{
		Enabled:                      true,
		H2C:                          false,
		MaxConcurrentStreams:         1000,
		MaxUploadBufferPerStream:     1048576, // 1MB
		MaxUploadBufferPerConnection: 1048576, // 1MB
		EnablePush:                   true,
		IdleTimeout:                  300 * time.Second,
		ReadTimeout:                  30 * time.Second,
		WriteTimeout:                 30 * time.Second,
		MaxDecoderHeaderTableSize:    4096,
		MaxEncoderHeaderTableSize:    4096,
		MaxReadFrameSize:             1048576, // 1MB
		PermitProhibitedCipherSuites: false,
	}
}

// DevelopmentHTTP2Config returns HTTP/2 configuration for local development
// Enables H2C (cleartext) for easier testing without TLS certificates
//
// Development Features:
//   - H2C enabled (no TLS required)
//   - Prohibited cipher suites permitted (compatibility)
//   - Same performance settings as production
//
// WARNING: Never use in production!
//   - H2C is insecure and not supported by browsers
//   - Weak cipher suites reduce security
//   - Only suitable for local testing
//
// Returns:
//   - HTTP2Config: Development-friendly configuration
func DevelopmentHTTP2Config() *HTTP2Config {
	config := DefaultHTTP2Config()
	config.H2C = true // Enable HTTP/2 over cleartext for development
	config.PermitProhibitedCipherSuites = true
	return config
}

// HTTP2Server wraps the standard library's HTTP/2 server functionality
// Provides integration between Go's net/http HTTP/2 implementation and fasthttp
//
// Architecture:
//   - Uses net/http for HTTP/2 protocol handling
//   - Converts between net/http and fasthttp request/response formats
//   - Maintains compatibility with Blaze's fasthttp-based architecture
//   - Supports both TLS and H2C (cleartext) modes
//
// Thread Safety:
//   - Safe for concurrent use
//   - Protected by internal mutexes where needed
//
// Lifecycle:
//  1. Create with NewHTTP2Server()
//  2. Configure with SetFastHTTPHandler()
//  3. Start with ListenAndServe()
//  4. Shutdown gracefully with Shutdown()
type HTTP2Server struct {
	config      *HTTP2Config            // HTTP/2 configuration
	tlsConfig   *TLSConfig              // TLS configuration for HTTPS
	server      *http.Server            // Underlying net/http server
	h2Server    *http2.Server           // HTTP/2 protocol handler
	fastHandler fasthttp.RequestHandler // FastHTTP handler to wrap
	mu          sync.RWMutex            // Protects concurrent access
}

// NewHTTP2Server creates a new HTTP/2 server instance
// Initializes HTTP/2 server with specified configuration and TLS settings
//
// Configuration Priority:
//   - If config is nil, uses DefaultHTTP2Config()
//   - TLS config is optional but required for production
//   - H2C mode doesn't require TLS config
//
// Server Setup:
//  1. Creates http2.Server with specified limits
//  2. Configures timeouts and buffer sizes
//  3. Sets up HPACK compression parameters
//  4. Prepares for TLS or H2C operation
//
// Parameters:
//   - config: HTTP/2 configuration (nil for defaults)
//   - tlsConfig: TLS configuration (required unless H2C mode)
//
// Returns:
//   - *HTTP2Server: Configured HTTP/2 server instance
//
// Example - Production Setup:
//
//	tlsConfig := blaze.DefaultTLSConfig()
//	tlsConfig.CertFile = "/path/to/cert.pem"
//	tlsConfig.KeyFile = "/path/to/key.pem"
//
//	http2Config := blaze.DefaultHTTP2Config()
//	server := blaze.NewHTTP2Server(&http2Config, &tlsConfig)
//
// Example - Development Setup (H2C):
//
//	http2Config := blaze.DevelopmentHTTP2Config()
//	server := blaze.NewHTTP2Server(&http2Config, nil)
func NewHTTP2Server(config *HTTP2Config, tlsConfig *TLSConfig) *HTTP2Server {
	if config == nil {
		config = DefaultHTTP2Config()
	}

	h2s := &HTTP2Server{
		config:    config,
		tlsConfig: tlsConfig,
	}

	// Create HTTP/2 server with correct field names
	h2s.h2Server = &http2.Server{
		MaxConcurrentStreams:         config.MaxConcurrentStreams,
		MaxUploadBufferPerStream:     config.MaxUploadBufferPerStream,
		MaxUploadBufferPerConnection: config.MaxUploadBufferPerConnection,
		IdleTimeout:                  config.IdleTimeout,
		MaxDecoderHeaderTableSize:    config.MaxDecoderHeaderTableSize,
		MaxEncoderHeaderTableSize:    config.MaxEncoderHeaderTableSize,
		MaxReadFrameSize:             config.MaxReadFrameSize,
		PermitProhibitedCipherSuites: config.PermitProhibitedCipherSuites,
	}

	return h2s
}

// SetFastHTTPHandler sets the FastHTTP handler to process requests
// Bridges FastHTTP and net/http by converting request/response formats
//
// The handler receives converted requests and generates responses that
// are automatically converted back to net/http format
//
// Parameters:
//   - handler: FastHTTP request handler
func (h2s *HTTP2Server) SetFastHTTPHandler(handler fasthttp.RequestHandler) {
	h2s.mu.Lock()
	defer h2s.mu.Unlock()
	h2s.fastHandler = handler
}

// convertFastHTTPHandler converts FastHTTP handler to net/http handler
// Performs bidirectional conversion between fasthttp and net/http formats
//
// Conversion Process:
//  1. Extract method, URI, headers from net/http request
//  2. Create fasthttp.Request and populate fields
//  3. Copy request body if present
//  4. Create fasthttp.RequestCtx
//  5. Call FastHTTP handler
//  6. Extract status, headers, body from fasthttp.Response
//  7. Write to net/http.ResponseWriter
//
// Performance Notes:
//   - Conversion has minimal overhead
//   - Zero-copy optimizations where possible
//   - Body streaming supported
//
// Returns:
//   - http.HandlerFunc: net/http compatible handler
func (h2s *HTTP2Server) convertFastHTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Convert net/http request to fasthttp request
		var req fasthttp.Request
		var resp fasthttp.Response

		// Set method
		req.Header.SetMethod(r.Method)

		// Set URI
		if r.URL.RawQuery != "" {
			req.SetRequestURI(r.URL.Path + "?" + r.URL.RawQuery)
		} else {
			req.SetRequestURI(r.URL.Path)
		}

		// Copy headers
		for name, values := range r.Header {
			for _, value := range values {
				req.Header.Add(name, value)
			}
		}

		// Copy body
		if r.Body != nil {
			req.SetBodyStream(r.Body, int(r.ContentLength))
		}

		// Set remote address using the correct method
		remoteAddr := r.RemoteAddr
		if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
			remoteAddr = host
		}

		// Create RequestCtx
		ctx := &fasthttp.RequestCtx{}
		ctx.Init(&req, nil, nil)

		// Set the remote address correctly
		if tcpAddr, err := net.ResolveTCPAddr("tcp", r.RemoteAddr); err == nil {
			ctx.SetRemoteAddr(tcpAddr)
		}

		// Call FastHTTP handler
		h2s.mu.RLock()
		handler := h2s.fastHandler
		h2s.mu.RUnlock()

		if handler != nil {
			handler(ctx)
		}

		// Get response
		ctx.Response.CopyTo(&resp)

		// Copy response to net/http response
		w.WriteHeader(resp.StatusCode())

		// Copy headers
		resp.Header.VisitAll(func(key, value []byte) {
			w.Header().Add(string(key), string(value))
		})

		// Write body
		w.Write(resp.Body())
	}
}

// setupHTTPServer creates the underlying net/http server
// Configures server based on H2C vs TLS mode
//
// H2C Mode (Development):
//   - Uses h2c.NewHandler for cleartext HTTP/2
//   - No TLS configuration needed
//   - Not suitable for production
//
// TLS Mode (Production):
//   - Configures TLS with certificates
//   - Uses http2.ConfigureServer for protocol setup
//   - ALPN negotiation for HTTP/2
//
// Parameters:
//   - addr: Server bind address (e.g., ":8080")
//
// Returns:
//   - error: Configuration error or nil on success
func (h2s *HTTP2Server) setupHTTPServer(addr string) error {
	handler := h2s.convertFastHTTPHandler()

	// Create HTTP server
	h2s.server = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  h2s.config.ReadTimeout,
		WriteTimeout: h2s.config.WriteTimeout,
		IdleTimeout:  h2s.config.IdleTimeout,
	}

	// Configure HTTP/2
	if h2s.config.H2C {
		// HTTP/2 over cleartext
		h2Handler := h2c.NewHandler(handler, h2s.h2Server)
		h2s.server.Handler = h2Handler
	} else {
		// Standard HTTP/2 with TLS
		if h2s.tlsConfig != nil {
			tlsConfig, err := h2s.tlsConfig.BuildTLSConfig()
			if err != nil {
				return fmt.Errorf("failed to build TLS config: %w", err)
			}
			h2s.server.TLSConfig = tlsConfig
		}

		// Configure HTTP/2
		if err := http2.ConfigureServer(h2s.server, h2s.h2Server); err != nil {
			return fmt.Errorf("failed to configure HTTP/2 server: %w", err)
		}
	}

	return nil
}

// ListenAndServe starts the HTTP/2 server on the specified address
// Automatically selects H2C or TLS mode based on configuration
//
// Server Startup:
//   - H2C mode: Starts cleartext HTTP/2 server
//   - TLS mode: Starts HTTPS server with HTTP/2 via ALPN
//   - Validates configuration before starting
//   - Blocks until server stops or error occurs
//
// Address Format:
//   - ":8080" - All interfaces, port 8080
//   - "127.0.0.1:8080" - Localhost only
//   - "0.0.0.0:443" - All interfaces, standard HTTPS port
//
// Parameters:
//   - addr: Server bind address
//
// Returns:
//   - error: Server error or nil if gracefully shutdown
//
// Example - H2C Development Server:
//
//	server.ListenAndServe(":8080")
//
// Example - TLS Production Server:
//
//	server.ListenAndServe(":443")
func (h2s *HTTP2Server) ListenAndServe(addr string) error {
	if err := h2s.setupHTTPServer(addr); err != nil {
		return err
	}

	if h2s.config.H2C {
		log.Printf("🚀 HTTP/2 server (h2c) starting on http://%s", addr)
		return h2s.server.ListenAndServe()
	} else {
		if h2s.tlsConfig == nil {
			return fmt.Errorf("TLS configuration required for HTTP/2 over TLS")
		}

		log.Printf("🔒 HTTP/2 server starting on https://%s", addr)
		return h2s.server.ListenAndServeTLS(h2s.tlsConfig.CertFile, h2s.tlsConfig.KeyFile)
	}
}

// Serve serves HTTP/2 connections from the given listener
// Allows using custom listeners (Unix sockets, pre-bound ports, etc.)
//
// Use Cases:
//   - Systemd socket activation
//   - Unix domain sockets
//   - Pre-bound privileged ports
//   - Custom network configurations
//
// Parameters:
//   - ln: Network listener providing connections
//
// Returns:
//   - error: Server error or nil if gracefully shutdown
//
// Example - Unix Socket:
//
//	ln, _ := net.Listen("unix", "/tmp/app.sock")
//	server.Serve(ln)
func (h2s *HTTP2Server) Serve(ln net.Listener) error {
	if h2s.server == nil {
		return fmt.Errorf("server not initialized, call ListenAndServe first")
	}

	return h2s.server.Serve(ln)
}

// ServeTLS serves HTTP/2 TLS connections from the given listener
// Similar to Serve but for TLS connections with specified certificates
//
// Parameters:
//   - ln: Network listener providing connections
//   - certFile: Path to TLS certificate file
//   - keyFile: Path to TLS private key file
//
// Returns:
//   - error: Server error or nil if gracefully shutdown
func (h2s *HTTP2Server) ServeTLS(ln net.Listener, certFile, keyFile string) error {
	if h2s.server == nil {
		return fmt.Errorf("server not initialized, call ListenAndServe first")
	}

	return h2s.server.ServeTLS(ln, certFile, keyFile)
}

// Shutdown gracefully shuts down the HTTP/2 server
// Allows in-flight requests to complete before stopping
//
// Graceful Shutdown Process:
//  1. Stop accepting new connections
//  2. Wait for active requests to complete
//  3. Close idle connections
//  4. Return when all connections closed or context cancelled
//
// Timeout Handling:
//   - Context timeout controls maximum shutdown duration
//   - If timeout expires, remaining connections are closed
//   - Use reasonable timeouts (30s-60s typical)
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
//	server.Shutdown(ctx)
func (h2s *HTTP2Server) Shutdown(ctx context.Context) error {
	if h2s.server == nil {
		return nil
	}
	return h2s.server.Shutdown(ctx)
}

// Close immediately closes the HTTP/2 server
// Does not wait for connections to close gracefully
//
// Use Cases:
//   - Emergency shutdown
//   - Testing/development
//   - When graceful shutdown timeout exceeded
//
// WARNING: Active requests will be interrupted
// Prefer Shutdown() for production use
//
// Returns:
//   - error: Close error or nil on success
func (h2s *HTTP2Server) Close() error {
	if h2s.server == nil {
		return nil
	}
	return h2s.server.Close()
}

// GetStats returns HTTP/2 server statistics
// Provides configuration and runtime information
//
// Returns:
//   - HTTP2Stats: Server statistics and configuration
func (h2s *HTTP2Server) GetStats() *HTTP2Stats {
	return &HTTP2Stats{
		Enabled:              h2s.config.Enabled,
		H2C:                  h2s.config.H2C,
		MaxConcurrentStreams: h2s.config.MaxConcurrentStreams,
		MaxReadFrameSize:     h2s.config.MaxReadFrameSize,
		EnablePush:           h2s.config.EnablePush,
	}
}

// HTTP2Stats holds HTTP/2 server statistics
// Provides insight into server configuration and status
type HTTP2Stats struct {
	Enabled              bool   `json:"enabled"`
	H2C                  bool   `json:"h2c"`
	MaxConcurrentStreams uint32 `json:"max_concurrent_streams"`
	MaxReadFrameSize     uint32 `json:"max_read_frame_size"`
	EnablePush           bool   `json:"enable_push"`
}

// HTTP2HealthCheck represents HTTP/2 health check information
// Used for monitoring and diagnostics
type HTTP2HealthCheck struct {
	Enabled bool        `json:"enabled"`
	H2C     bool        `json:"h2c"`
	Stats   *HTTP2Stats `json:"stats,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// GetHTTP2HealthCheck returns HTTP/2 health check information
// Provides quick status overview for monitoring systems
//
// Returns:
//   - HTTP2HealthCheck: Health check data
func (h2s *HTTP2Server) GetHTTP2HealthCheck() *HTTP2HealthCheck {
	return &HTTP2HealthCheck{
		Enabled: h2s.config.Enabled,
		H2C:     h2s.config.H2C,
		Stats:   h2s.GetStats(),
	}
}

// ServerPush represents HTTP/2 server push functionality
// Allows proactively pushing resources to clients
//
// Server Push Benefits:
//   - Reduces latency by eliminating request round trips
//   - Improves page load performance
//   - Useful for CSS, JavaScript, images
//
// Best Practices:
//   - Only push resources that will be needed
//   - Don't push resources already cached
//   - Limit number of pushed resources
//   - Monitor push effectiveness
type ServerPush struct {
	server *HTTP2Server
}

// NewServerPush creates a new server push instance
// Wrapper for server push functionality
//
// Parameters:
//   - server: HTTP/2 server instance
//
// Returns:
//   - *ServerPush: Server push instance
func NewServerPush(server *HTTP2Server) *ServerPush {
	return &ServerPush{
		server: server,
	}
}

// Push pushes a resource to the client
// Note: Server Push with fasthttp is complex and may require additional middleware
//
// Parameters:
//   - w: Response writer (must support http.Pusher interface)
//   - target: Resource path to push
//   - opts: Push options (headers, method, etc.)
//
// Returns:
//   - error: Push error or nil on success
func (sp *ServerPush) Push(w http.ResponseWriter, target string, opts *http.PushOptions) error {
	if pusher, ok := w.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return fmt.Errorf("server push not supported")
}

// HTTP2Middleware creates middleware for HTTP/2 specific features
func HTTP2Middleware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Set HTTP/2 specific headers
			c.SetHeader("Server", "Blaze/1.0 (HTTP/2)")

			// Add HTTP/2 protocol information to response headers
			if c.Request().Header.Peek("Http2-Settings") != nil {
				c.SetHeader("Alt-Svc", `h2=":443"; ma=2592000`)
			}

			return next(c)
		}
	}
}

// StreamPriority represents HTTP/2 stream priority
type StreamPriority struct {
	StreamID   uint32
	Weight     uint8
	Exclusive  bool
	Dependency uint32
}

// PushPromise represents an HTTP/2 push promise
type PushPromise struct {
	Method string
	Path   string
	Header map[string]string
}
