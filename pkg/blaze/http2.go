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

// HTTP2Config holds HTTP/2 configuration
type HTTP2Config struct {
	// Enable HTTP/2 support
	Enabled bool

	// Enable HTTP/2 over cleartext (h2c) for development
	H2C bool

	// Maximum concurrent streams per connection
	MaxConcurrentStreams uint32

	// Maximum upload buffer per stream
	MaxUploadBufferPerStream int32

	// Maximum upload buffer per connection
	MaxUploadBufferPerConnection int32

	// Enable server push
	EnablePush bool

	// Idle timeout
	IdleTimeout time.Duration

	// Read timeout
	ReadTimeout time.Duration

	// Write timeout
	WriteTimeout time.Duration

	// Maximum decoder header table size for HPACK
	MaxDecoderHeaderTableSize uint32

	// Maximum encoder header table size for HPACK
	MaxEncoderHeaderTableSize uint32

	// Maximum read frame size
	MaxReadFrameSize uint32

	// Permit prohibited cipher suites (for compatibility)
	PermitProhibitedCipherSuites bool
}

// DefaultHTTP2Config returns default HTTP/2 configuration
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

// DevelopmentHTTP2Config returns HTTP/2 configuration for development
func DevelopmentHTTP2Config() *HTTP2Config {
	config := DefaultHTTP2Config()
	config.H2C = true // Enable HTTP/2 over cleartext for development
	config.PermitProhibitedCipherSuites = true
	return config
}

// HTTP2Server wraps the standard HTTP/2 server functionality
type HTTP2Server struct {
	config      *HTTP2Config
	tlsConfig   *TLSConfig
	server      *http.Server
	h2Server    *http2.Server
	fastHandler fasthttp.RequestHandler
	mu          sync.RWMutex
}

// NewHTTP2Server creates a new HTTP/2 server
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

// SetFastHTTPHandler sets the FastHTTP handler to use
func (h2s *HTTP2Server) SetFastHTTPHandler(handler fasthttp.RequestHandler) {
	h2s.mu.Lock()
	defer h2s.mu.Unlock()
	h2s.fastHandler = handler
}

// convertFastHTTPHandler converts FastHTTP handler to net/http handler
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

// ListenAndServe starts the HTTP/2 server
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
func (h2s *HTTP2Server) Serve(ln net.Listener) error {
	if h2s.server == nil {
		return fmt.Errorf("server not initialized, call ListenAndServe first")
	}

	return h2s.server.Serve(ln)
}

// ServeTLS serves HTTP/2 TLS connections from the given listener
func (h2s *HTTP2Server) ServeTLS(ln net.Listener, certFile, keyFile string) error {
	if h2s.server == nil {
		return fmt.Errorf("server not initialized, call ListenAndServe first")
	}

	return h2s.server.ServeTLS(ln, certFile, keyFile)
}

// Shutdown gracefully shuts down the HTTP/2 server
func (h2s *HTTP2Server) Shutdown(ctx context.Context) error {
	if h2s.server == nil {
		return nil
	}
	return h2s.server.Shutdown(ctx)
}

// Close immediately closes the HTTP/2 server
func (h2s *HTTP2Server) Close() error {
	if h2s.server == nil {
		return nil
	}
	return h2s.server.Close()
}

// GetStats returns HTTP/2 server statistics
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
type HTTP2Stats struct {
	Enabled              bool   `json:"enabled"`
	H2C                  bool   `json:"h2c"`
	MaxConcurrentStreams uint32 `json:"max_concurrent_streams"`
	MaxReadFrameSize     uint32 `json:"max_read_frame_size"`
	EnablePush           bool   `json:"enable_push"`
}

// HTTP2HealthCheck represents HTTP/2 health check information
type HTTP2HealthCheck struct {
	Enabled bool        `json:"enabled"`
	H2C     bool        `json:"h2c"`
	Stats   *HTTP2Stats `json:"stats,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// GetHTTP2HealthCheck returns HTTP/2 health check information
func (h2s *HTTP2Server) GetHTTP2HealthCheck() *HTTP2HealthCheck {
	return &HTTP2HealthCheck{
		Enabled: h2s.config.Enabled,
		H2C:     h2s.config.H2C,
		Stats:   h2s.GetStats(),
	}
}

// ServerPush represents HTTP/2 server push functionality
type ServerPush struct {
	server *HTTP2Server
}

// NewServerPush creates a new server push instance
func NewServerPush(server *HTTP2Server) *ServerPush {
	return &ServerPush{
		server: server,
	}
}

// Push pushes a resource to the client (this is a placeholder for future implementation)
// Note: Server Push with fasthttp is complex and may require additional middleware
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
