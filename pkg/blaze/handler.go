package blaze

// HandlerFunc defines the handler function signature
type HandlerFunc func(*Context) error

// MiddlewareFunc defines the middleware function signature
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Handler wraps HandlerFunc to satisfy the interface
type Handler interface {
	Handle(*Context) error
}

// HandlerAdapter adapts HandlerFunc to Handler interface
type HandlerAdapter struct {
	handler HandlerFunc
}

// Handle implements the Handler interface
func (h *HandlerAdapter) Handle(c *Context) error {
	return h.handler(c)
}

// NewHandler creates a new Handler from HandlerFunc
func NewHandler(handler HandlerFunc) Handler {
	return &HandlerAdapter{handler: handler}
}
