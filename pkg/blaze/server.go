package blaze

import (
	"context"
	"net"

	"github.com/valyala/fasthttp"
)

// Server wraps fasthttp.Server with additional functionality
type Server struct {
	*fasthttp.Server
	config *Config
}

// NewServer creates a new server instance
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

// ListenAndServe starts the server (FIXED METHOD)
func (s *Server) ListenAndServe(addr string, handler fasthttp.RequestHandler) error {
	s.Server.Handler = handler // Set handler on the underlying fasthttp.Server
	return s.Server.ListenAndServe(addr)
}

// ListenAndServeTLS starts the server with TLS (FIXED METHOD)
func (s *Server) ListenAndServeTLS(addr, certFile, keyFile string, handler fasthttp.RequestHandler) error {
	s.Server.Handler = handler // Set handler on the underlying fasthttp.Server
	return s.Server.ListenAndServeTLS(addr, certFile, keyFile)
}

// Serve serves connections from the given listener (FIXED METHOD)
func (s *Server) Serve(ln net.Listener, handler fasthttp.RequestHandler) error {
	s.Server.Handler = handler // Set handler on the underlying fasthttp.Server
	return s.Server.Serve(ln)
}

// ServeTLS serves HTTPS connections from the given listener (FIXED METHOD)
func (s *Server) ServeTLS(ln net.Listener, certFile, keyFile string, handler fasthttp.RequestHandler) error {
	s.Server.Handler = handler // Set handler on the underlying fasthttp.Server
	return s.Server.ServeTLS(ln, certFile, keyFile)
}

// GracefulShutdown gracefully shuts down the server
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

// SetHandler sets the request handler (HELPER METHOD)
func (s *Server) SetHandler(handler fasthttp.RequestHandler) {
	s.Server.Handler = handler
}

// GetHandler gets the current request handler
func (s *Server) GetHandler() fasthttp.RequestHandler {
	return s.Server.Handler
}
