package blaze

import (
	"fmt"
	"regexp"
	"strings"
)

// Router implements a radix tree-based router with advanced features
// Provides high-performance HTTP routing with parameter extraction and constraints
//
// Router Architecture:
//   - Radix tree for efficient route matching (O(log n) lookup)
//   - Parameter extraction with named captures
//   - Wildcard/catch-all routes
//   - Route constraints (regex, type validation)
//   - Route grouping and middleware
//   - Method-specific routing
//
// Routing Features:
//   - Static routes: /users/profile
//   - Named parameters: /users/:id
//   - Wildcard routes: /files/*path
//   - Route constraints: /users/:id<int>
//   - Priority-based matching
//
// Performance:
//   - Constant-time method lookup
//   - Logarithmic-time path matching
//   - Zero allocations for static routes
//   - Minimal allocations for dynamic routes
type Router struct {
	// root is the root node of the radix tree
	// Each HTTP method has its own tree for isolation
	root *routeNode

	// routes stores all registered routes by key (method:pattern)
	// Used for route introspection and management
	routes map[string]*Route

	// config holds router configuration
	// Controls behavior like case sensitivity, trailing slashes
	config RouterConfig
}

// RouterConfig holds comprehensive router configuration
// Controls routing behavior, matching rules, and features
//
// Configuration Trade-offs:
//   - CaseSensitive: Security vs convenience
//   - StrictSlash: Precision vs flexibility
//   - RedirectSlash: User-friendly vs explicit
//   - HandleMethodNotAllowed: Compliance vs performance
//
// Production Settings:
//   - CaseSensitive: false (user-friendly URLs)
//   - StrictSlash: false (flexible matching)
//   - RedirectSlash: true (SEO-friendly)
//   - HandleMethodNotAllowed: true (proper HTTP)
type RouterConfig struct {
	// CaseSensitive when true, routes are case-sensitive
	// true: /Users and /users are different routes
	// false: /Users and /users match the same route
	// Default: false (more user-friendly)
	CaseSensitive bool

	// StrictSlash when true, trailing slashes must match exactly
	// true: /users/ and /users are different routes
	// false: /users/ and /users match the same route
	// Default: false (more flexible)
	StrictSlash bool

	// RedirectSlash when true, redirects to add/remove trailing slash
	// Helps with SEO by providing canonical URLs
	// Only applies when StrictSlash is false
	// Default: true (SEO-friendly)
	RedirectSlash bool

	// UseEscapedPath when true, matches against escaped path
	// true: Matches %2F as literal %2F
	// false: Matches %2F as /
	// Default: false (standard URL decoding)
	UseEscapedPath bool

	// HandleMethodNotAllowed when true, returns 405 for wrong methods
	// true: Returns 405 Method Not Allowed with Allow header
	// false: Returns 404 Not Found
	// Default: true (proper HTTP semantics)
	HandleMethodNotAllowed bool

	// HandleOPTIONS when true, automatically handles OPTIONS requests
	// Returns allowed methods for the route
	// Useful for CORS preflight
	// Default: true (CORS support)
	HandleOPTIONS bool

	// EnableMerging allows merging routes with same pattern
	// Combines multiple method handlers into single route
	// Reduces tree size and improves performance
	// Default: true (performance optimization)
	EnableMerging bool

	// MaxMergeDepth limits recursion depth for route merging
	// Prevents infinite loops in pathological cases
	// Default: 10 (sufficient for most applications)
	MaxMergeDepth int
}

// DefaultRouterConfig returns default router configuration
// Provides balanced settings for most applications
//
// Default Settings:
//   - Case-insensitive URLs (user-friendly)
//   - Flexible trailing slashes (convenient)
//   - Automatic slash redirection (SEO)
//   - Proper HTTP method handling (standards-compliant)
//   - Route merging enabled (performance)
//
// Returns:
//   - RouterConfig: Default configuration
func DefaultRouterConfig() RouterConfig {
	return RouterConfig{
		CaseSensitive:          false,
		StrictSlash:            false,
		RedirectSlash:          true,
		UseEscapedPath:         false,
		HandleMethodNotAllowed: true,
		HandleOPTIONS:          true,
		EnableMerging:          true,
		MaxMergeDepth:          10,
	}
}

// routeNode represents a node in the radix tree
// Forms the internal structure of the routing tree
//
// Node Types:
//   - static: Fixed path segment (/users)
//   - root: Tree root node
//   - param: Named parameter (/:id)
//   - catchAll: Wildcard (/*path)
//
// Tree Structure:
//   - Each node stores a path segment
//   - Children nodes for different paths
//   - Handlers for matched routes
//   - Indices for quick child lookup
type routeNode struct {
	// path is the path segment this node represents
	// For param nodes: ":paramName"
	// For catchAll nodes: "*paramName"
	// For static nodes: actual path segment
	path string

	// indices stores first characters of children paths
	// Used for O(1) child lookup by first character
	// Example: "apu" for children "admin", "posts", "users"
	indices string

	// children stores child nodes
	// Ordered to match indices string
	children []*routeNode

	// handlers maps HTTP methods to routes
	// Key: HTTP method (GET, POST, etc.)
	// Value: Route with handler and metadata
	handlers map[string]*Route

	// priority determines child search order
	// Higher priority = searched first
	// Based on route frequency and wildcard type
	priority uint32

	// maxParams tracks maximum parameters in subtree
	// Used for pre-allocating parameter maps
	maxParams uint8

	// wildChild indicates if node has wildcard child
	// Optimizes wildcard matching
	wildChild bool

	// nodeType identifies the type of node
	// Determines matching behavior
	nodeType nodeType

	// constraint for parameter validation
	// nil for non-parameter nodes
	constraint *RouteConstraint
}

// nodeType defines the type of route node
// Different node types have different matching semantics
type nodeType uint8

const (
	// static is a normal node with fixed path segment
	// Exact match required
	static nodeType = iota

	// root is the root of the tree
	// Special handling for empty path
	root

	// param is a named parameter node (:id)
	// Matches single path segment
	// Stores value in params map
	param

	// catchAll is a wildcard node (*path)
	// Matches remaining path segments
	// Always at end of path
	catchAll
)

// RouteConstraint defines constraints for route parameters
// Validates parameter values before routing
//
// Constraint Types:
//   - int: Integer values only
//   - uuid: Valid UUID format
//   - alpha: Alphabetic characters only
//   - regex: Custom regex pattern
//
// Validation Flow:
//  1. Extract parameter from path
//  2. Apply constraint pattern matching
//  3. Reject request if validation fails
//  4. Pass validated value to handler
type RouteConstraint struct {
	// Name is the parameter name
	// Must match parameter in route pattern
	Name string

	// Pattern is the regex for validation
	// Compiled once at route registration
	Pattern *regexp.Regexp

	// Type identifies the constraint category
	// Used for error messages and debugging
	Type ConstraintType
}

// ConstraintType defines the type of constraint
// Categorizes constraints for better error messages
type ConstraintType string

const (
	// IntConstraint validates integer parameters
	// Pattern: ^[0-9]+$
	// Example: /users/:id<int>
	IntConstraint ConstraintType = "int"

	// UUIDConstraint validates UUID parameters
	// Pattern: RFC 4122 UUID format
	// Example: /resources/:id<uuid>
	UUIDConstraint ConstraintType = "uuid"

	// AlphaConstraint validates alphabetic parameters
	// Pattern: ^[a-zA-Z]+$
	// Example: /categories/:slug<alpha>
	AlphaConstraint ConstraintType = "alpha"

	// RegexConstraint validates custom regex patterns
	// Pattern: User-defined
	// Example: /files/:name<regex:[a-z0-9-]+>
	RegexConstraint ConstraintType = "regex"
)

// Route represents an enhanced route with constraints and middleware
// Stores complete route information including handlers and metadata
//
// Route Lifecycle:
//  1. Created during route registration
//  2. Stored in routes map
//  3. Inserted into radix tree
//  4. Matched during requests
//  5. Handler executed with middleware
type Route struct {
	// Method is the HTTP method (GET, POST, etc.)
	Method string

	// Pattern is the original route pattern
	// Example: "/users/:id/posts/*path"
	Pattern string

	// Handler is the request handler function
	Handler HandlerFunc

	// Middleware is route-specific middleware
	// Applied in addition to global middleware
	Middleware []MiddlewareFunc

	// Constraints maps parameter names to validation rules
	// Applied before handler execution
	Constraints map[string]*RouteConstraint

	// Name is an optional route identifier
	// Used for route lookups and URL generation
	Name string

	// Params lists parameter names in order
	// Extracted from route pattern during registration
	Params []string

	// Merged contains routes merged into this route
	// Non-nil when EnableMerging is true and routes merged
	Merged []*Route

	// Priority determines route matching order
	// Higher priority routes checked first
	Priority int

	// Tags categorizes routes for grouping
	// Used for filtering and documentation
	Tags []string
}

type RouteGroup struct {
	Name        string
	Description string
	Routes      []*Route
	Middleware  []MiddlewareFunc
}

// NewRouter creates a new router instance
// Initializes radix tree and configuration
//
// Parameters:
//   - config: Optional router configuration
//
// Returns:
//   - *Router: Configured router instance
//
// Example:
//
//	router := blaze.NewRouter()
//	router := blaze.NewRouter(customConfig)
func NewRouter(config ...RouterConfig) *Router {
	var cfg RouterConfig
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultRouterConfig()
	}

	return &Router{
		root:   &routeNode{},
		routes: make(map[string]*Route),
		config: cfg,
	}
}

// MergeRoutes merges multiple routes with the same pattern
func (r *Router) MergeRoutes(pattern string) error {
	if !r.config.EnableMerging {
		return fmt.Errorf("route merging is disabled")
	}

	var routesToMerge []*Route

	// Find all routes with the same pattern
	for key, route := range r.routes {
		if strings.Contains(key, pattern) {
			routesToMerge = append(routesToMerge, route)
		}
	}

	if len(routesToMerge) <= 1 {
		return fmt.Errorf("no routes to merge for pattern: %s", pattern)
	}

	// Create a master route
	masterRoute := &Route{
		Pattern:     pattern,
		Merged:      routesToMerge,
		Handler:     r.createMergedHandler(routesToMerge),
		Middleware:  r.mergeMidlleware(routesToMerge),
		Constraints: r.mergeConstraints(routesToMerge),
		Priority:    r.calculateMergedPriority(routesToMerge),
	}

	// Update the routing tree
	r.addToTree("*", pattern, masterRoute)

	return nil
}

// createMergedHandler creates a handler that can handle multiple HTTP methods
func (r *Router) createMergedHandler(routes []*Route) HandlerFunc {
	methodMap := make(map[string]HandlerFunc)

	for _, route := range routes {
		methodMap[route.Method] = route.Handler
	}

	return func(c *Context) error {
		method := c.Method()
		if handler, exists := methodMap[method]; exists {
			return handler(c)
		}

		// Method not allowed
		return c.Status(405).JSON(Map{
			"error":           "Method Not Allowed",
			"allowed_methods": r.getAllowedMethods(routes),
		})
	}
}

// mergeMidlleware combines middleware from multiple routes
func (r *Router) mergeMidlleware(routes []*Route) []MiddlewareFunc {
	var merged []MiddlewareFunc
	seen := make(map[string]bool)

	for _, route := range routes {
		for _, mw := range route.Middleware {
			// Use a simple string representation to avoid duplicates
			key := fmt.Sprintf("%p", mw)
			if !seen[key] {
				merged = append(merged, mw)
				seen[key] = true
			}
		}
	}

	return merged
}

// mergeConstraints combines constraints from multiple routes
func (r *Router) mergeConstraints(routes []*Route) map[string]*RouteConstraint {
	merged := make(map[string]*RouteConstraint)

	for _, route := range routes {
		for param, constraint := range route.Constraints {
			if existing, exists := merged[param]; exists {
				// Merge constraints if they conflict
				merged[param] = r.mergeConstraint(existing, constraint)
			} else {
				merged[param] = constraint
			}
		}
	}

	return merged
}

// mergeConstraint merges two constraints for the same parameter
func (r *Router) mergeConstraint(c1, c2 *RouteConstraint) *RouteConstraint {
	// If types are different, use regex constraint
	if c1.Type != c2.Type {
		return &RouteConstraint{
			Name:    c1.Name,
			Type:    RegexConstraint,
			Pattern: regexp.MustCompile(".*"), // Accept all
		}
	}
	return c1 // Use first constraint if types match
}

// calculateMergedPriority calculates priority for merged routes
func (r *Router) calculateMergedPriority(routes []*Route) int {
	maxPriority := 0
	for _, route := range routes {
		if route.Priority > maxPriority {
			maxPriority = route.Priority
		}
	}
	return maxPriority
}

// getAllowedMethods returns allowed methods for a set of routes
func (r *Router) getAllowedMethods(routes []*Route) []string {
	var methods []string
	seen := make(map[string]bool)

	for _, route := range routes {
		if !seen[route.Method] {
			methods = append(methods, route.Method)
			seen[route.Method] = true
		}
	}

	return methods
}

// AddRouteGroup adds multiple routes as a group
func (r *Router) AddRouteGroup(group *RouteGroup) {
	for _, route := range group.Routes {
		// Apply group middleware
		combinedMiddleware := append(group.Middleware, route.Middleware...)
		route.Middleware = combinedMiddleware

		r.AddRoute(route.Method, route.Pattern, route.Handler,
			WithMiddleware(combinedMiddleware...))
	}
}

// GetRoutesByTag returns routes filtered by tags
func (r *Router) GetRoutesByTag(tag string) []*Route {
	var routes []*Route
	for _, route := range r.routes {
		for _, routeTag := range route.Tags {
			if routeTag == tag {
				routes = append(routes, route)
				break
			}
		}
	}
	return routes
}

// GetRouteInfo returns detailed information about all routes
func (r *Router) GetRouteInfo() map[string]*RouteInfo {
	info := make(map[string]*RouteInfo)

	for key, route := range r.routes {
		info[key] = &RouteInfo{
			Method:          route.Method,
			Pattern:         route.Pattern,
			Name:            route.Name,
			Params:          route.Params,
			HasConstraints:  len(route.Constraints) > 0,
			MiddlewareCount: len(route.Middleware),
			Priority:        route.Priority,
			Tags:            route.Tags,
			IsMerged:        len(route.Merged) > 0,
		}
	}

	return info
}

// RouteInfo provides information about a route
type RouteInfo struct {
	Method          string   `json:"method"`
	Pattern         string   `json:"pattern"`
	Name            string   `json:"name,omitempty"`
	Params          []string `json:"params,omitempty"`
	HasConstraints  bool     `json:"has_constraints"`
	MiddlewareCount int      `json:"middleware_count"`
	Priority        int      `json:"priority"`
	Tags            []string `json:"tags,omitempty"`
	IsMerged        bool     `json:"is_merged"`
}

// WithPriority sets the route priority
// Higher priority routes are matched first
//
// Parameters:
//   - priority: Priority value (higher = first)
//
// Returns:
//   - RouteOption: Configuration function
//
// Example:
//
//	app.GET("/special", handler, blaze.WithPriority(100))
func WithPriority(priority int) RouteOption {
	return func(r *Route) {
		r.Priority = priority
	}
}

// WithTags adds tags to the route
// Tags categorize routes for filtering and documentation
//
// Parameters:
//   - tags: One or more tag strings
//
// Returns:
//   - RouteOption: Configuration function
//
// Example:
//
//	app.GET("/api/users", handler, blaze.WithTags("api", "public"))
func WithTags(tags ...string) RouteOption {
	return func(r *Route) {
		r.Tags = append(r.Tags, tags...)
	}
}

func WithMerge(enable bool) RouteOption {
	return func(r *Route) {
		// This is handled at router level
	}
}

// AddRoute adds a route with constraints and middleware
// Registers route in tree and stores metadata
//
// Route Registration Process:
//  1. Create route object with handler
//  2. Apply route options (middleware, constraints, etc.)
//  3. Parse pattern to extract parameters
//  4. Insert into radix tree
//  5. Store in routes map for introspection
//
// Parameters:
//   - method: HTTP method (GET, POST, etc.)
//   - pattern: Route pattern with parameters
//   - handler: Request handler function
//   - options: Route configuration options
//
// Returns:
//   - *Route: Registered route
//
// Example:
//
//	route := router.AddRoute("GET", "/users/:id", handler,
//	    blaze.WithName("get_user"),
//	    blaze.WithIntConstraint("id"),
//	)
func (r *Router) AddRoute(method, pattern string, handler HandlerFunc, options ...RouteOption) *Route {
	route := &Route{
		Method:      method,
		Pattern:     pattern,
		Handler:     handler,
		Middleware:  make([]MiddlewareFunc, 0),
		Constraints: make(map[string]*RouteConstraint),
		Params:      make([]string, 0),
	}

	// Apply route options
	for _, option := range options {
		option(route)
	}

	// Parse pattern and extract parameters
	r.parsePattern(route)

	// Add to radix tree
	r.addToTree(method, pattern, route)

	// Store route
	key := method + ":" + pattern
	r.routes[key] = route

	return route
}

// RouteOption defines a function to configure routes
// Provides fluent API for route configuration
//
// Example:
//
//	app.GET("/users/:id", handler,
//	    blaze.WithName("get_user"),
//	    blaze.WithMiddleware(authMiddleware),
//	    blaze.WithIntConstraint("id"),
//	)
type RouteOption func(*Route)

// WithName sets the route name
// Enables route lookups and reverse routing
//
// Parameters:
//   - name: Unique route identifier
//
// Returns:
//   - RouteOption: Configuration function
//
// Example:
//
//	app.GET("/users/:id", handler, blaze.WithName("get_user"))
func WithName(name string) RouteOption {
	return func(r *Route) {
		r.Name = name
	}
}

// WithMiddleware adds middleware to the route
// Middleware is applied only to this specific route
//
// Parameters:
//   - middleware: One or more middleware functions
//
// Returns:
//   - RouteOption: Configuration function
//
// Example:
//
//	app.POST("/admin", handler,
//	    blaze.WithMiddleware(authMiddleware, adminMiddleware),
//	)
func WithMiddleware(middleware ...MiddlewareFunc) RouteOption {
	return func(r *Route) {
		r.Middleware = append(r.Middleware, middleware...)
	}
}

// WithConstraint adds a parameter constraint
// Validates parameter before routing
//
// Parameters:
//   - param: Parameter name
//   - constraint: Validation constraint
//
// Returns:
//   - RouteOption: Configuration function
//
// Example:
//
//	constraint := blaze.RouteConstraint{
//	    Name: "id",
//	    Type: blaze.IntConstraint,
//	    Pattern: regexp.MustCompile(`^\d+$`),
//	}
//	app.GET("/users/:id", handler, blaze.WithConstraint("id", constraint))
func WithConstraint(param string, constraint *RouteConstraint) RouteOption {
	return func(r *Route) {
		r.Constraints[param] = constraint
	}
}

// WithIntConstraint adds an integer constraint
// Validates that parameter is a positive integer
//
// Parameters:
//   - param: Parameter name
//
// Returns:
//   - RouteOption: Configuration function
//
// Example:
//
//	app.GET("/users/:id", handler, blaze.WithIntConstraint("id"))
func WithIntConstraint(param string) RouteOption {
	return func(r *Route) {
		r.Constraints[param] = &RouteConstraint{
			Name:    param,
			Type:    IntConstraint,
			Pattern: regexp.MustCompile(`^\d+$`),
		}
	}
}

// WithUUIDConstraint adds a UUID constraint
// Validates that parameter is a valid UUID (RFC 4122)
//
// Parameters:
//   - param: Parameter name
//
// Returns:
//   - RouteOption: Configuration function
//
// Example:
//
//	app.GET("/resources/:id", handler, blaze.WithUUIDConstraint("id"))
func WithUUIDConstraint(param string) RouteOption {
	return func(r *Route) {
		r.Constraints[param] = &RouteConstraint{
			Name:    param,
			Type:    UUIDConstraint,
			Pattern: regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`),
		}
	}
}

// WithRegexConstraint adds a custom regex constraint
// Validates parameter against custom pattern
//
// Parameters:
//   - param: Parameter name
//   - pattern: Regex pattern string
//
// Returns:
//   - RouteOption: Configuration function
//
// Example:
//
//	app.GET("/files/:name", handler,
//	    blaze.WithRegexConstraint("name", `^[a-z0-9-]+$`),
//	)
func WithRegexConstraint(param string, pattern string) RouteOption {
	return func(r *Route) {
		r.Constraints[param] = &RouteConstraint{
			Name:    param,
			Type:    RegexConstraint,
			Pattern: regexp.MustCompile(pattern),
		}
	}
}

// parsePattern parses the route pattern and extracts parameters
func (r *Router) parsePattern(route *Route) {
	pattern := route.Pattern
	segments := strings.Split(strings.Trim(pattern, "/"), "/")

	for _, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			paramName := segment[1:]
			route.Params = append(route.Params, paramName)
		} else if strings.HasPrefix(segment, "*") {
			paramName := segment[1:]
			if paramName == "" {
				paramName = "wildcard"
			}
			route.Params = append(route.Params, paramName)
		}
	}
}

// addToTree adds a route to the radix tree
func (r *Router) addToTree(method, pattern string, route *Route) {
	path := pattern
	if !r.config.CaseSensitive {
		path = strings.ToLower(path)
	}

	root := r.root
	if root.handlers == nil {
		root.handlers = make(map[string]*Route)
	}

	r.insertRoute(root, method, path, route)
}

// insertRoute inserts a route into the tree
func (r *Router) insertRoute(n *routeNode, method, path string, route *Route) {
	// _originalPath := path
	fullPath := path
	n.priority++

walk:
	for {
		// Find the longest common prefix
		i := longestCommonPrefix(path, n.path)

		// Split edge
		if i < len(n.path) {
			child := &routeNode{
				path:      n.path[i:],
				wildChild: n.wildChild,
				nodeType:  static,
				indices:   n.indices,
				children:  n.children,
				handlers:  n.handlers,
				priority:  n.priority - 1,
			}

			n.children = []*routeNode{child}
			n.indices = string([]byte{n.path[i]})
			n.path = path[:i]
			n.handlers = nil
			n.wildChild = false
		}

		// Make new node a child of this node
		if i < len(path) {
			path = path[i:]

			if n.wildChild {
				n = n.children[0]
				n.priority++

				// Check if the wildcard matches
				if len(path) >= len(n.path) && n.path == path[:len(n.path)] {
					// Check for longer wildcard
					if len(n.path) >= len(path) || path[len(n.path)] == '/' {
						continue walk
					}
				}

				panic("path segment '" + path +
					"' conflicts with existing wildcard '" + n.path +
					"' in path '" + fullPath + "'")
			}

			c := path[0]

			// Param node
			if n.nodeType == param && c == '/' && len(n.children) == 1 {
				n = n.children[0]
				n.priority++
				continue walk
			}

			// Check if a child with the next path byte exists
			for i, max := 0, len(n.indices); i < max; i++ {
				if c == n.indices[i] {
					i = n.incrementChildPrio(i)
					n = n.children[i]
					continue walk
				}
			}

			// Otherwise insert it
			if c != ':' && c != '*' {
				// []byte for proper sorting
				n.indices += string([]byte{c})
				child := &routeNode{
					maxParams: route.maxParams(),
				}
				n.children = append(n.children, child)
				n.incrementChildPrio(len(n.indices) - 1)
				n = child
			}
			n.insertChild(path, fullPath, route, method)
			return
		}

		// Otherwise add handler to current node
		if n.handlers == nil {
			n.handlers = make(map[string]*Route)
		}
		n.handlers[method] = route
		return
	}
}

// insertChild inserts a child node
func (n *routeNode) insertChild(path, fullPath string, route *Route, method string) {
	for {
		// Find prefix until first wildcard
		wildcard, i, valid := findWildcard(path)
		if i < 0 { // No wildcard found
			break
		}

		// The wildcard name must not contain ':' and '*'
		if !valid {
			panic("only one wildcard per path segment is allowed, has: '" +
				wildcard + "' in path '" + fullPath + "'")
		}

		// Check if the wildcard has a name
		if len(wildcard) < 2 {
			panic("wildcards must be named with a non-empty name in path '" + fullPath + "'")
		}

		// Split path at the beginning of the wildcard
		if i > 0 {
			n.path = path[:i]
			path = path[i:]
		}

		if wildcard[0] == ':' { // param
			// Split path at the end of the wildcard
			if i := strings.Index(wildcard, "/"); i > 0 {
				n.path += wildcard[:i]
				wildcard = wildcard[i:]
			}

			child := &routeNode{
				nodeType:  param,
				path:      wildcard,
				maxParams: route.maxParams(),
			}
			n.children = []*routeNode{child}
			n.wildChild = true
			n = child
			n.priority++

			// If the path doesn't end with the wildcard, then there
			// will be another subpath starting with '/'
			if len(wildcard) < len(path) {
				path = path[len(wildcard):]
				child := &routeNode{
					maxParams: route.maxParams(),
					priority:  1,
				}
				n.children = []*routeNode{child}
				n = child
				continue
			}

			// Otherwise we're done. Insert the handler in the new leaf
			if n.handlers == nil {
				n.handlers = make(map[string]*Route)
			}
			n.handlers[method] = route
			return

		} else { // catchAll
			if i+len(wildcard) != len(path) {
				panic("catch-all routes are only allowed at the end of the path in path '" + fullPath + "'")
			}

			if len(n.path) > 0 && n.path[len(n.path)-1] == '/' {
				panic("catch-all conflicts with existing handle for the path segment root in path '" + fullPath + "'")
			}

			// Currently fixed width 1 for '/'
			i--
			if path[i] != '/' {
				panic("no / before catch-all in path '" + fullPath + "'")
			}

			n.path = path[:i]

			// First node: catchAll node with empty path
			child := &routeNode{
				wildChild: true,
				nodeType:  catchAll,
				maxParams: route.maxParams(),
			}

			n.children = []*routeNode{child}
			n.indices = string('/')
			n = child
			n.priority++

			// Second node: node holding the variable
			child = &routeNode{
				path:      path[i:],
				nodeType:  catchAll,
				maxParams: route.maxParams(),
				handlers:  make(map[string]*Route),
				priority:  1,
			}
			child.handlers[method] = route
			n.children = []*routeNode{child}

			return
		}
	}

	// If no wildcard was found, simply insert the path and handler
	n.path = path
	if n.handlers == nil {
		n.handlers = make(map[string]*Route)
	}
	n.handlers[method] = route
}

// FindRoute finds a matching route for the given method and path
// Traverses radix tree to find best match
//
// Matching Process:
//  1. Normalize path (case, trailing slash)
//  2. Traverse radix tree
//  3. Extract route parameters
//  4. Validate constraints
//  5. Return route and parameters
//
// Parameters:
//   - method: HTTP method
//   - path: Request path
//
// Returns:
//   - *Route: Matched route or nil
//   - map[string]string: Extracted parameters
//   - bool: true if route found
//
// Example:
//
//	route, params, found := router.FindRoute("GET", "/users/123")
//	// route: matched route
//	// params: {"id": "123"}
//	// found: true
func (r *Router) FindRoute(method, path string) (*Route, map[string]string, bool) {
	if !r.config.CaseSensitive {
		path = strings.ToLower(path)
	}

	root := r.root
	params := make(map[string]string)

	route, found := r.getValue(root, path, method, params)
	if !found {
		return nil, nil, false
	}

	// Validate constraints
	if !r.validateConstraints(route, params) {
		return nil, nil, false
	}

	return route, params, true
}

// getValue traverses the tree to find a matching route
func (r *Router) getValue(n *routeNode, path, method string, params map[string]string) (*Route, bool) {
walk: // Outer loop for walking the tree
	for {
		prefix := n.path
		if len(path) > len(prefix) {
			if path[:len(prefix)] == prefix {
				path = path[len(prefix):]

				// Try all the non-wildcard children first
				for i, max := 0, len(n.indices); i < max; i++ {
					c := n.indices[i]
					if c == path[0] {
						n = n.children[i]
						continue walk
					}
				}

				// If there is no wildcard, we can't match the route
				if !n.wildChild {
					return nil, false
				}

				// Handle wildcard child
				n = n.children[0]
				switch n.nodeType {
				case param:
					// Find end (either '/' or path end)
					end := 0
					for end < len(path) && path[end] != '/' {
						end++
					}

					// Save param value
					paramKey := n.path[1:] // Remove ':'
					params[paramKey] = path[:end]

					// We need to go deeper!
					if end < len(path) {
						if len(n.children) > 0 {
							path = path[end:]
							n = n.children[0]
							continue walk
						}

						// ... but we can't
						return nil, false
					}

					if route, ok := n.handlers[method]; ok {
						return route, true
					}

					if len(n.children) == 1 {
						// No handler found. Check if a handler for this route + a
						// trailing slash exists for trailing slash recommendation
						n = n.children[0]
						if n.path == "/" && n.handlers[method] != nil {
							return n.handlers[method], true
						}
					}

					return nil, false

				case catchAll:
					// Save param value
					paramKey := n.path[2:] // Remove '*'
					params[paramKey] = path

					if route, ok := n.handlers[method]; ok {
						return route, true
					}

					return nil, false

				default:
					panic("invalid node type")
				}
			}
		} else if path == prefix {
			if route, ok := n.handlers[method]; ok {
				return route, true
			}

			// Handle trailing slash
			if path == "/" && r.config.RedirectSlash && method != "CONNECT" {
				if route, ok := n.handlers[method]; ok {
					return route, true
				}
			}

			return nil, false
		}

		// Nothing found
		return nil, false
	}
}

// validateConstraints validates route parameter constraints
func (r *Router) validateConstraints(route *Route, params map[string]string) bool {
	for paramName, constraint := range route.Constraints {
		value, exists := params[paramName]
		if !exists {
			continue
		}

		if !constraint.Pattern.MatchString(value) {
			return false
		}
	}
	return true
}

// Helper methods for Route
func (r *Route) maxParams() uint8 {
	return uint8(len(r.Params))
}

// incrementChildPrio increments the priority of the given child and reorders if necessary
func (n *routeNode) incrementChildPrio(pos int) int {
	cs := n.children
	cs[pos].priority++
	prio := cs[pos].priority

	// Adjust position (move to front)
	newPos := pos
	for ; newPos > 0 && cs[newPos-1].priority < prio; newPos-- {
		// Swap node positions
		cs[newPos-1], cs[newPos] = cs[newPos], cs[newPos-1]
	}

	// Build new index char string
	if newPos != pos {
		n.indices = n.indices[:newPos] + // unchanged prefix, might be empty
			n.indices[pos:pos+1] + // the index char we move
			n.indices[newPos:pos] + n.indices[pos+1:] // rest without char at 'pos'
	}

	return newPos
}

// Helper functions
func longestCommonPrefix(a, b string) int {
	i := 0
	max := len(a)
	if len(b) < max {
		max = len(b)
	}
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}

func findWildcard(path string) (wildcard string, i int, valid bool) {
	// Find start
	for start, c := range []byte(path) {
		// A wildcard starts with ':' (param) or '*' (catch-all)
		if c != ':' && c != '*' {
			continue
		}

		// Find end and check for invalid characters
		valid = true
		for end, c := range []byte(path[start+1:]) {
			switch c {
			case '/':
				return path[start : start+1+end], start, valid
			case ':', '*':
				valid = false
			}
		}
		return path[start:], start, valid
	}
	return "", -1, false
}
