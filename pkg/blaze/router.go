package blaze

import (
	"strings"
)

// Router handles routing
type Router struct {
	routes map[string]*Route
}

// Route represents a single route
type Route struct {
	method   string
	pattern  string
	handler  HandlerFunc
	segments []string
	params   []string
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]*Route),
	}
}

// Add adds a route
func (r *Router) Add(method, pattern string, handler HandlerFunc) {
	key := method + ":" + pattern
	route := &Route{
		method:  method,
		pattern: pattern,
		handler: handler,
	}

	// Parse route pattern
	route.parsePattern()

	r.routes[key] = route
}

// GET adds a GET route
func (r *Router) GET(pattern string, handler HandlerFunc) {
	r.Add("GET", pattern, handler)
}

// POST adds a POST route
func (r *Router) POST(pattern string, handler HandlerFunc) {
	r.Add("POST", pattern, handler)
}

// PUT adds a PUT route
func (r *Router) PUT(pattern string, handler HandlerFunc) {
	r.Add("PUT", pattern, handler)
}

// DELETE adds a DELETE route
func (r *Router) DELETE(pattern string, handler HandlerFunc) {
	r.Add("DELETE", pattern, handler)
}

// PATCH adds a PATCH route
func (r *Router) PATCH(pattern string, handler HandlerFunc) {
	r.Add("PATCH", pattern, handler)
}

// Handler returns a handler function for the router
func (r *Router) Handler() HandlerFunc {
	return func(c *Context) error {
		route, params := r.match(c.Method(), c.Path())
		if route == nil {
			return c.Status(404).JSON(Map{
				"error": "Not Found",
			})
		}

		// Set route parameters
		for key, value := range params {
			c.SetParam(key, value)
		}

		return route.handler(c)
	}
}

// match finds a matching route
func (r *Router) match(method, path string) (*Route, map[string]string) {
	// First try exact match
	key := method + ":" + path
	if route, exists := r.routes[key]; exists {
		return route, make(map[string]string)
	}

	// Try pattern matching
	pathSegments := strings.Split(strings.Trim(path, "/"), "/")

	for _, route := range r.routes {
		if route.method != method {
			continue
		}

		if params := route.matchSegments(pathSegments); params != nil {
			return route, params
		}
	}

	return nil, nil
}

// parsePattern parses the route pattern into segments and parameters
func (r *Route) parsePattern() {
	pattern := strings.Trim(r.pattern, "/")
	if pattern == "" {
		r.segments = []string{}
		return
	}

	r.segments = strings.Split(pattern, "/")
	r.params = make([]string, 0)

	for _, segment := range r.segments {
		if strings.HasPrefix(segment, ":") {
			r.params = append(r.params, segment[1:])
		}
	}
}

// matchSegments matches path segments against route pattern
func (r *Route) matchSegments(pathSegments []string) map[string]string {
	if len(pathSegments) != len(r.segments) {
		return nil
	}

	params := make(map[string]string)

	for i, segment := range r.segments {
		if strings.HasPrefix(segment, ":") {
			// Parameter segment
			paramName := segment[1:]
			params[paramName] = pathSegments[i]
		} else {
			// Literal segment
			if segment != pathSegments[i] {
				return nil
			}
		}
	}

	return params
}
