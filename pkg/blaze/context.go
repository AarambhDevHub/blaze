package blaze

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
)

var fastjson = jsoniter.ConfigCompatibleWithStandardLibrary

// Context represents the request context for a single HTTP request
// It wraps fasthttp.RequestCtx and provides additional functionality for:
//   - Route parameter extraction
//   - Query parameter parsing
//   - Request/response header manipulation
//   - JSON/form data binding and validation
//   - Cookie management
//   - Local variable storage
//   - File operations
//   - Multipart form handling
//   - HTTP2 features
//   - Application state access
//   - Graceful shutdown coordination
//
// Context instances are created per-request and should not be stored or reused
// across requests. All methods are designed for single-request lifecycle.
type Context struct {
	*fasthttp.RequestCtx                        // Underlying fasthttp request context
	params               map[string]string      // Route parameters extracted from URL path
	locals               map[string]interface{} // Request-scoped local variables
}

// Map is a shortcut for map[string]interface{}
// Used throughout the framework for convenient JSON responses and data structures
type Map map[string]interface{}

// Param returns the value of a route parameter by key
// Route parameters are extracted from dynamic URL segments defined in routes
//
// Example:
//
//	Route: /users/:id
//	Request: /users/123
//	c.Param("id") returns "123"
//
// Parameters:
//   - key: Parameter name as defined in the route pattern
//
// Returns:
//   - string: Parameter value or empty string if not found
func (c *Context) Param(key string) string {
	return c.params[key]
}

// ParamInt returns a route parameter value as an integer
// Useful for extracting numeric IDs from URL paths
//
// Example:
//
//	userID, err := c.ParamInt("id")
//	if err != nil {
//	    return c.Status(400).JSON(Map{"error": "Invalid user ID"})
//	}
//
// Parameters:
//   - key: Parameter name as defined in the route pattern
//
// Returns:
//   - int: Parsed integer value
//   - error: Parsing error if value is not a valid integer
func (c *Context) ParamInt(key string) (int, error) {
	value := c.Param(key)
	if value == "" {
		return 0, fmt.Errorf("parameter %s not found", key)
	}
	return strconv.Atoi(value)
}

// ParamIntDefault returns a route parameter as integer with fallback default
// Returns the default value if parameter is missing or cannot be parsed
//
// Example:
//
//	page := c.ParamIntDefault("page", 1)  // Defaults to page 1
//
// Parameters:
//   - key: Parameter name
//   - defaultValue: Fallback value if parameter is missing or invalid
//
// Returns:
//   - int: Parsed parameter value or default value
func (c *Context) ParamIntDefault(key string, defaultValue int) int {
	value, err := c.ParamInt(key)
	if err != nil {
		return defaultValue
	}
	return value
}

// SetParam sets a route parameter value
// Used internally by the router during request matching
//
// Parameters:
//   - key: Parameter name
//   - value: Parameter value to store
func (c *Context) SetParam(key, value string) {
	if c.params == nil {
		c.params = make(map[string]string)
	}
	c.params[key] = value
}

// Query returns the value of a URL query parameter
// Query parameters are the key-value pairs after '?' in the URL
//
// Example:
//
//	URL: /search?q=golang&limit=10
//	c.Query("q") returns "golang"
//
// Parameters:
//   - key: Query parameter name
//
// Returns:
//   - string: Query parameter value or empty string if not found
func (c *Context) Query(key string) string {
	return string(c.RequestCtx.QueryArgs().Peek(key))
}

// QueryArgs returns the underlying fasthttp query arguments
// Provides direct access to all query parameters for advanced use cases
//
// Returns:
//   - *fasthttp.Args: Query arguments container
func (c *Context) QueryArgs() *fasthttp.Args {
	return c.RequestCtx.QueryArgs()
}

// QueryDefault returns a query parameter with fallback default value
// Returns default if parameter is not present
//
// Example:
//
//	limit := c.QueryDefault("limit", "10")
//
// Parameters:
//   - key: Query parameter name
//   - defaultValue: Fallback value if parameter is missing
//
// Returns:
//   - string: Query parameter value or default value
func (c *Context) QueryDefault(key, defaultValue string) string {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// QueryInt returns a query parameter as an integer
// Useful for pagination, limits, and numeric filters
//
// Example:
//
//	page, err := c.QueryInt("page")
//	if err != nil {
//	    page = 1  // Default to first page
//	}
//
// Parameters:
//   - key: Query parameter name
//
// Returns:
//   - int: Parsed integer value
//   - error: Parsing error if value is not a valid integer
func (c *Context) QueryInt(key string) (int, error) {
	value := c.Query(key)
	if value == "" {
		return 0, fmt.Errorf("parameter '%s' not found", key)
	}
	return strconv.Atoi(value)
}

// QueryIntDefault returns a query parameter as integer with default
// Commonly used for pagination and limit parameters
//
// Example:
//
//	page := c.QueryIntDefault("page", 1)
//	limit := c.QueryIntDefault("limit", 20)
//
// Parameters:
//   - key: Query parameter name
//   - defaultValue: Fallback value if parameter is missing or invalid
//
// Returns:
//   - int: Parsed parameter value or default value
func (c *Context) QueryIntDefault(key string, defaultValue int) int {
	value, err := c.QueryInt(key)
	if err != nil {
		return defaultValue
	}
	return value
}

// Header returns the value of a request header
// Header names are case-insensitive per HTTP specification
//
// Example:
//
//	authToken := c.Header("Authorization")
//	contentType := c.Header("Content-Type")
//
// Parameters:
//   - key: Header name (case-insensitive)
//
// Returns:
//   - string: Header value or empty string if not found
func (c *Context) Header(key string) string {
	return string(c.RequestCtx.Request.Header.Peek(key))
}

// Request returns the underlying fasthttp request
// Provides direct access to request for advanced operations
//
// Returns:
//   - *fasthttp.Request: The underlying request object
func (c *Context) Request() *fasthttp.Request {
	return &c.RequestCtx.Request
}

// Response returns the underlying fasthttp response
// Provides direct access to response for advanced operations
//
// Returns:
//   - *fasthttp.Response: The underlying response object
func (c *Context) Response() *fasthttp.Response {
	return &c.RequestCtx.Response
}

// Body returns the raw request body as bytes
// The body can only be read once; subsequent calls return the same cached data
//
// Returns:
//   - []byte: Raw request body
func (c *Context) Body() []byte {
	return c.RequestCtx.PostBody()
}

// PostBody returns the request body (alias for Body)
// Provides consistent naming with fasthttp
//
// Returns:
//   - []byte: Raw request body
func (c *Context) PostBody() []byte {
	return c.RequestCtx.PostBody()
}

// BodyString returns the request body as a string
// Convenient for debugging and text-based request bodies
//
// Returns:
//   - string: Request body as UTF-8 string
func (c *Context) BodyString() string {
	return string(c.Body())
}

// Bind automatically binds request data to a struct based on Content-Type
// Supports:
//   - application/json -> BindJSON
//   - application/x-www-form-urlencoded -> BindForm
//
// Example:
//
//	var user User
//	if err := c.Bind(&user); err != nil {
//	    return c.Status(400).JSON(Map{"error": "Invalid request"})
//	}
//
// Parameters:
//   - v: Pointer to struct to populate with request data
//
// Returns:
//   - error: Binding error or nil on success
func (c *Context) Bind(v interface{}) error {
	contentType := string(c.RequestCtx.Request.Header.ContentType())

	switch {
	case strings.Contains(contentType, "application/json"):
		return c.BindJSON(v)
	case strings.Contains(contentType, "application/x-www-form-urlencoded"):
		return c.BindForm(v)
	default:
		return fmt.Errorf("unsupported content type: %s", contentType)
	}
}

// BindJSON binds JSON request body to a struct
// Uses json-iterator for high-performance JSON parsing
//
// Example:
//
//	var createUserReq struct {
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
//	if err := c.BindJSON(&createUserReq); err != nil {
//	    return c.Status(400).JSON(Map{"error": "Invalid JSON"})
//	}
//
// Parameters:
//   - v: Pointer to struct with json tags
//
// Returns:
//   - error: JSON parsing error or nil on success
func (c *Context) BindJSON(v interface{}) error {
	return fastjson.Unmarshal(c.Body(), v)
}

// Cookie returns the value of a cookie by name
// Cookies are parsed from the Cookie request header
//
// Example:
//
//	sessionID := c.Cookie("session_id")
//	if sessionID == "" {
//	    return c.Status(401).Text("Not authenticated")
//	}
//
// Parameters:
//   - name: Cookie name
//
// Returns:
//   - string: Cookie value or empty string if not found
func (c *Context) Cookie(name string) string {
	return string(c.RequestCtx.Request.Header.Cookie(name))
}

// Locals returns a request-scoped local variable
// Local variables persist throughout the request lifecycle
// Commonly used to pass data between middleware and handlers
//
// Example:
//
//	// In authentication middleware
//	c.SetLocals("user_id", 123)
//
//	// In handler
//	userID := c.Locals("user_id").(int)
//
// Parameters:
//   - key: Variable name
//
// Returns:
//   - interface{}: Stored value or nil if not found
func (c *Context) Locals(key string) interface{} {
	if c.locals == nil {
		return nil
	}
	return c.locals[key]
}

// SetLocals sets a request-scoped local variable
// Variables are scoped to the current request only
//
// Parameters:
//   - key: Variable name
//   - value: Value to store (any type)
//
// Returns:
//   - *Context: For method chaining
func (c *Context) SetLocals(key string, value interface{}) *Context {
	if c.locals == nil {
		c.locals = make(map[string]interface{})
	}
	c.locals[key] = value
	return c
}

// Method returns the HTTP method of the request
// Common methods: GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD
//
// Returns:
//   - string: HTTP method in uppercase
func (c *Context) Method() string {
	return string(c.RequestCtx.Method())
}

// Path returns the request path (URL without query string)
// The path does not include the query string or fragment
//
// Example:
//
//	Request: /users/123?active=true
//	c.Path() returns "/users/123"
//
// Returns:
//   - string: URL path
func (c *Context) Path() string {
	return string(c.RequestCtx.Path())
}

// URI returns the full request URI
// Provides access to the complete URI including query parameters
//
// Returns:
//   - *fasthttp.URI: Complete URI object
func (c *Context) URI() *fasthttp.URI {
	return c.RequestCtx.URI()
}

// IP returns the client IP address
// Extracts IP from RemoteAddr, handling IPv4 and IPv6 formats
//
// Returns:
//   - string: Client IP address
func (c *Context) IP() string {
	return c.RequestCtx.RemoteIP().String()
}

// RemoteIP returns the remote IP as net.IP
// Provides more detailed IP information than IP()
//
// Returns:
//   - net.IP: Client IP address object
func (c *Context) RemoteIP() net.IP {
	return c.RequestCtx.RemoteIP()
}

// UserAgent returns the User-Agent request header
// Identifies the client software making the request
//
// Returns:
//   - string: User-Agent header value
func (c *Context) UserAgent() string {
	return string(c.RequestCtx.UserAgent())
}

// MarshalJSON converts blaze.Map to JSON bytes
// Implements json.Marshaler interface for Map type
//
// Returns:
//   - []byte: JSON representation
//   - error: Marshaling error or nil
func (m Map) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}(m))
}

// ToJSON converts blaze.Map to JSON string
// Convenient method for debugging and logging
//
// Returns:
//   - string: JSON string representation
//   - error: Marshaling error or nil
func (m Map) ToJSON() (string, error) {
	data, err := json.Marshal(map[string]interface{}(m))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToJSONBytes converts blaze.Map to JSON byte slice
// Efficient method for direct byte manipulation
//
// Returns:
//   - []byte: JSON bytes
//   - error: Marshaling error or nil
func (m Map) ToJSONBytes() ([]byte, error) {
	return json.Marshal(map[string]interface{}(m))
}

// ShutdownContext returns the application's shutdown context
// The context is cancelled when graceful shutdown begins
// Handlers should monitor this context for long-running operations
//
// Example:
//
//	select {
//	case <-c.ShutdownContext().Done():
//	    return c.Status(503).Text("Server shutting down")
//	case <-time.After(5 * time.Second):
//	    // Process request
//	}
//
// Returns:
//   - context.Context: Shutdown context (cancelled on shutdown)
func (c *Context) ShutdownContext() context.Context {
	if ctx := c.Locals("shutdown_ctx"); ctx != nil {
		if shutdownCtx, ok := ctx.(context.Context); ok {
			return shutdownCtx
		}
	}
	return context.Background()
}

// IsShuttingDown returns true if the application is shutting down
// Provides quick check without blocking on context
//
// Example:
//
//	if c.IsShuttingDown() {
//	    return c.Status(503).Text("Server shutting down")
//	}
//
// Returns:
//   - bool: true if shutdown has been initiated
func (c *Context) IsShuttingDown() bool {
	select {
	case <-c.ShutdownContext().Done():
		return true
	default:
		return false
	}
}

// WithTimeout creates a context with timeout that respects shutdown
// Combines request timeout with shutdown coordination
//
// Example:
//
//	ctx, cancel := c.WithTimeout(5 * time.Second)
//	defer cancel()
//	result, err := dbQuery(ctx)
//
// Parameters:
//   - timeout: Duration for the timeout
//
// Returns:
//   - context.Context: New context with timeout
//   - context.CancelFunc: Cancel function to release resources
func (c *Context) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.ShutdownContext(), timeout)
}

// WithDeadline creates a context with deadline that respects shutdown
// Similar to WithTimeout but uses absolute time
//
// Parameters:
//   - deadline: Absolute time for the deadline
//
// Returns:
//   - context.Context: New context with deadline
//   - context.CancelFunc: Cancel function to release resources
func (c *Context) WithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(c.ShutdownContext(), deadline)
}

// IsHTTP2 returns true if the request is using HTTP/2 protocol
// HTTP/2 enables features like server push and multiplexing
//
// Returns:
//   - bool: true if HTTP/2 is being used
func (c *Context) IsHTTP2() bool {
	if enabled := c.Locals("http2_enabled"); enabled != nil {
		if enabled, ok := enabled.(bool); ok {
			return enabled
		}
	}
	return false
}

// Protocol returns the protocol version (HTTP/1.1 or HTTP/2.0)
// Useful for logging and conditional behavior
//
// Returns:
//   - string: Protocol version string
func (c *Context) Protocol() string {
	if protocol := c.Locals("protocol"); protocol != nil {
		if protocol, ok := protocol.(string); ok {
			return protocol
		}
	}
	return "HTTP/1.1"
}

// StreamID returns the HTTP/2 stream ID
// Each HTTP/2 request has a unique stream ID
// Returns 0 for HTTP/1.1 requests
//
// Returns:
//   - uint32: Stream ID (0 for HTTP/1.1)
func (c *Context) StreamID() uint32 {
	if c.IsHTTP2() {
		return uint32(c.RequestCtx.ID())
	}
	return 0
}

// ServerPush pushes a resource to the client (HTTP/2 only)
// Server push allows the server to send resources before requested
// Improves page load performance by reducing round trips
//
// Example:
//
//	c.ServerPush("/static/style.css", "text/css")
//	c.ServerPush("/static/script.js", "application/javascript")
//
// Parameters:
//   - path: Resource path to push
//   - contentType: MIME type of the resource
//
// Returns:
//   - error: Push error or nil on success
func (c *Context) ServerPush(path, contentType string) error {
	if !c.IsHTTP2() {
		return fmt.Errorf("server push is only supported in HTTP/2")
	}

	// Add Link header for server push
	linkHeader := fmt.Sprintf("<%s>; rel=preload; as=%s", path, contentType)
	c.SetHeader("Link", linkHeader)

	return nil
}

// PushResources pushes multiple resources to the client (HTTP/2 only)
// Batch version of ServerPush for convenience
//
// Example:
//
//	resources := map[string]string{
//	    "/static/style.css": "text/css",
//	    "/static/script.js": "application/javascript",
//	}
//	c.PushResources(resources)
//
// Parameters:
//   - resources: Map of path to content type
//
// Returns:
//   - error: Push error or nil on success
func (c *Context) PushResources(resources map[string]string) error {
	if !c.IsHTTP2() {
		return fmt.Errorf("server push is only supported in HTTP/2")
	}

	for path, contentType := range resources {
		if err := c.ServerPush(path, contentType); err != nil {
			return err
		}
	}

	return nil
}

// GetUserValue returns a user value by key
// Maps to fasthttp's UserValue for compatibility
//
// Parameters:
//   - key: Value key
//
// Returns:
//   - interface{}: Stored value or nil
func (c *Context) GetUserValue(key string) interface{} {
	return c.RequestCtx.UserValue(key)
}

// SetUserValue sets a user value
// Maps to fasthttp's SetUserValue for compatibility
//
// Parameters:
//   - key: Value key
//   - value: Value to store
//
// Returns:
//   - *Context: For method chaining
func (c *Context) SetUserValue(key string, value interface{}) *Context {
	c.RequestCtx.SetUserValue(key, value)
	return c
}

// GetUserValueString returns a user value as string
// Provides type-safe string extraction
//
// Parameters:
//   - key: Value key
//
// Returns:
//   - string: Value as string or empty string
func (c *Context) GetUserValueString(key string) string {
	if value := c.GetUserValue(key); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// GetUserValueInt returns a user value as int
// Provides type-safe integer extraction
//
// Parameters:
//   - key: Value key
//
// Returns:
//   - int: Value as integer or 0
func (c *Context) GetUserValueInt(key string) int {
	if value := c.GetUserValue(key); value != nil {
		if i, ok := value.(int); ok {
			return i
		}
	}
	return 0
}

// GetClientIP returns the client IP address (convenience method)
// Extracts IP set by IPMiddleware
//
// Returns:
//   - string: Client IP address
func (c *Context) GetClientIP() string {
	return c.GetUserValueString("client_ip")
}

// GetRealIP returns the real client IP address (convenience method)
// Useful when behind proxies or load balancers
//
// Returns:
//   - string: Real client IP address
func (c *Context) GetRealIP() string {
	return c.GetUserValueString("real_ip")
}

// GetRemoteAddr returns the remote address (convenience method)
// Full remote address including port
//
// Returns:
//   - string: Remote address with port
func (c *Context) GetRemoteAddr() string {
	return c.GetUserValueString("remote_addr")
}

// MultipartForm returns the parsed multipart form
// Uses default multipart configuration
//
// Returns:
//   - *MultipartForm: Parsed multipart form data
//   - error: Parsing error or nil
func (c *Context) MultipartForm() (*MultipartForm, error) {
	return c.MultipartFormWithConfig(DefaultMultipartConfig())
}

// MultipartFormWithConfig returns the parsed multipart form with custom config
// Allows control over memory limits, file size, temp directory, etc.
//
// Parameters:
//   - config: Multipart configuration
//
// Returns:
//   - *MultipartForm: Parsed multipart form data
//   - error: Parsing error or nil
func (c *Context) MultipartFormWithConfig(config *MultipartConfig) (*MultipartForm, error) {
	// Get the fasthttp multipart form
	form, err := c.RequestCtx.MultipartForm()
	if err != nil {
		return nil, fmt.Errorf("failed to parse multipart form: %w", err)
	}

	// Convert to our MultipartForm structure
	blazeForm := &MultipartForm{
		Value: make(map[string][]string),
		File:  make(map[string][]*MultipartFile),
	}

	// Copy form values
	for key, values := range form.Value {
		blazeForm.Value[key] = values
	}

	// Process uploaded files
	fileCount := 0
	for fieldName, fileHeaders := range form.File {
		var files []*MultipartFile

		for _, fileHeader := range fileHeaders {
			// Check file count limit
			if config.MaxFiles > 0 && fileCount >= config.MaxFiles {
				return nil, fmt.Errorf("maximum number of files (%d) exceeded", config.MaxFiles)
			}

			// Create MultipartFile
			blazeFile := &MultipartFile{
				Filename:    fileHeader.Filename,
				Header:      fileHeader.Header,
				Size:        fileHeader.Size,
				ContentType: fileHeader.Header.Get("Content-Type"),
				FileHeader:  fileHeader,
			}

			// Validate file
			if err := config.validateFile(blazeFile); err != nil {
				return nil, err
			}

			// Read file data
			file, err := fileHeader.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open uploaded file: %w", err)
			}
			defer file.Close()

			if config.KeepInMemory || blazeFile.Size <= config.MaxMemory {
				// Keep in memory
				blazeFile.Data, err = io.ReadAll(file)
				if err != nil {
					return nil, fmt.Errorf("failed to read file data: %w", err)
				}
			} else {
				// Save to temporary file
				tempFile, err := os.CreateTemp(config.TempDir, "blaze_upload_*")
				if err != nil {
					return nil, fmt.Errorf("failed to create temp file: %w", err)
				}
				defer tempFile.Close()

				written, err := io.Copy(tempFile, file)
				if err != nil {
					os.Remove(tempFile.Name())
					return nil, fmt.Errorf("failed to save file data: %w", err)
				}

				blazeFile.TempFilePath = tempFile.Name()
				blazeFile.Size = written
			}

			files = append(files, blazeFile)
			fileCount++
		}

		blazeForm.File[fieldName] = files
	}

	return blazeForm, nil
}

// FormFile returns the first uploaded file for the given field name
// Convenient method for single file uploads
//
// Example:
//
//	file, err := c.FormFile("avatar")
//	if err != nil {
//	    return c.Status(400).JSON(Map{"error": "No file uploaded"})
//	}
//	file.Save("/uploads/" + file.Filename)
//
// Parameters:
//   - name: Form field name
//
// Returns:
//   - *MultipartFile: First uploaded file
//   - error: Error if no file found or parsing failed
func (c *Context) FormFile(name string) (*MultipartFile, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}

	file := form.GetFile(name)
	if file == nil {
		return nil, fmt.Errorf("no file uploaded for field %s", name)
	}

	return file, nil
}

// FormFiles returns all uploaded files for the given field name
// Used for multiple file uploads with the same field name
//
// Parameters:
//   - name: Form field name
//
// Returns:
//   - []MultipartFile: All uploaded files
//   - error: Error if no files found or parsing failed
func (c *Context) FormFiles(name string) ([]*MultipartFile, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return nil, err
	}

	files := form.GetFiles(name)
	if files == nil {
		return nil, fmt.Errorf("no files uploaded for field %s", name)
	}

	return files, nil
}

// SaveUploadedFile saves an uploaded file to the specified path
// Provides simple file saving with error handling
//
// Parameters:
//   - file: MultipartFile to save
//   - dst: Destination file path
//
// Returns:
//   - error: Save error or nil on success
func (c *Context) SaveUploadedFile(file *MultipartFile, dst string) error {
	return file.Save(dst)
}

// SaveUploadedFileToDir saves an uploaded file to a directory
// Uses original filename from upload
//
// Parameters:
//   - file: MultipartFile to save
//   - dir: Destination directory
//
// Returns:
//   - string: Full path where file was saved
//   - error: Save error or nil on success
func (c *Context) SaveUploadedFileToDir(file *MultipartFile, dir string) (string, error) {
	return file.SaveToDir(dir)
}

// SaveUploadedFileWithUniqueFilename saves file with a unique generated filename
// Prevents filename collisions by generating unique names
//
// Parameters:
//   - file: MultipartFile to save
//   - dir: Destination directory
//
// Returns:
//   - string: Full path where file was saved
//   - error: Save error or nil on success
func (c *Context) SaveUploadedFileWithUniqueFilename(file *MultipartFile, dir string) (string, error) {
	return file.SaveWithUniqueFilename(dir)
}

// FormValue returns form value (works with both multipart and URL-encoded forms)
// Automatically detects form type and extracts value
//
// Parameters:
//   - name: Form field name
//
// Returns:
//   - string: Form value or empty string
func (c *Context) FormValue(name string) string {
	// Try multipart form first
	if c.Request().Header.ContentType() != nil &&
		strings.Contains(string(c.Request().Header.ContentType()), "multipart/form-data") {
		form, err := c.MultipartForm()
		if err == nil {
			return form.GetValue(name)
		}
	}

	// Fall back to regular form value
	return string(c.RequestCtx.FormValue(name))
}

// FormValues returns all form values for the given name
// Useful for checkbox groups and multi-select inputs
//
// Parameters:
//   - name: Form field name
//
// Returns:
//   - []string: All values for the field
func (c *Context) FormValues(name string) []string {
	// Try multipart form first
	if c.Request().Header.ContentType() != nil &&
		strings.Contains(string(c.Request().Header.ContentType()), "multipart/form-data") {
		form, err := c.MultipartForm()
		if err == nil {
			return form.GetValues(name)
		}
	}

	// Fall back to regular form values
	args := c.RequestCtx.PostArgs()
	var values []string
	args.VisitAll(func(key, value []byte) {
		if string(key) == name {
			values = append(values, string(value))
		}
	})
	return values
}

// IsMultipartForm checks if the request contains multipart form data
// Useful for conditional form handling
//
// Returns:
//   - bool: true if Content-Type is multipart/form-data
func (c *Context) IsMultipartForm() bool {
	contentType := string(c.Request().Header.ContentType())
	return strings.Contains(contentType, "multipart/form-data")
}

// GetContentType returns the request Content-Type header
// Includes charset and other parameters if present
//
// Returns:
//   - string: Complete Content-Type header value
func (c *Context) GetContentType() string {
	return string(c.Request().Header.ContentType())
}

// SendFile sends a file as response
// Wrapper around fasthttp.RequestCtx.SendFile
// FastHTTP's SendFile is highly optimized and doesn't return errors
//
// Parameters:
//   - filepath: Path to file to send
//
// Returns:
//   - error: Always returns nil (for API consistency)
func (c *Context) SendFile(filepath string) error {
	// FastHTTP's SendFile doesn't return an error, it logs internally
	c.RequestCtx.SendFile(filepath)
	return nil
}

// ServeFile serves a file with proper headers
// Uses fasthttp.ServeFile for optimized file serving
//
// Parameters:
//   - filepath: Path to file to serve
//
// Returns:
//   - error: File serving error or nil
func (c *Context) ServeFile(filepath string) error {
	// Use fasthttp.ServeFile which is more flexible
	fasthttp.ServeFile(c.RequestCtx, filepath)
	return nil
}

// ServeFileDownload serves a file as a download with custom filename
// Sets Content-Disposition header to trigger browser download
//
// Example:
//
//	c.ServeFileDownload("/data/report.pdf", "monthly_report_2024.pdf")
//
// Parameters:
//   - filepath: Path to file on server
//   - filename: Filename for the download (as seen by user)
//
// Returns:
//   - error: File serving error or nil
func (c *Context) ServeFileDownload(filepath, filename string) error {
	// Set download headers
	c.SetHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.SetHeader("Content-Type", "application/octet-stream")

	// Send the file
	c.RequestCtx.SendFile(filepath)
	return nil
}

// ServeFileInline serves a file for inline display (like images in browser)
// Sets Content-Disposition to inline and proper content type
//
// Parameters:
//   - filepath: Path to file to serve
//
// Returns:
//   - error: File serving error or nil
func (c *Context) ServeFileInline(filepath string) error {
	// Let the browser determine how to display the file
	c.SetHeader("Content-Disposition", "inline")

	// Detect and set proper content type based on file extension
	if contentType := getContentTypeFromFile(filepath); contentType != "" {
		c.SetHeader("Content-Type", contentType)
	}

	c.RequestCtx.SendFile(filepath)
	return nil
}

// StreamFile streams a file with support for range requests
// Useful for large files, videos, and audio streaming
// Supports HTTP range requests for seeking
//
// Parameters:
//   - filepath: Path to file to stream
//
// Returns:
//   - error: Streaming error or nil
func (c *Context) StreamFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Set headers
	c.SetHeader("Accept-Ranges", "bytes")
	c.SetHeader("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	if contentType := getContentTypeFromFile(filepath); contentType != "" {
		c.SetHeader("Content-Type", contentType)
	}

	// Handle range requests
	rangeHeader := string(c.Request().Header.Peek("Range"))
	if rangeHeader != "" {
		return c.handleRangeRequest(file, fileInfo.Size(), rangeHeader)
	}

	// Stream entire file
	_, err = io.Copy(c.ResponseWriter(), file)
	return err
}

// handleRangeRequest handles HTTP range requests for file streaming
// Internal method supporting partial content delivery
func (c *Context) handleRangeRequest(file *os.File, fileSize int64, rangeHeader string) error {
	// Parse range header (simplified implementation)
	// Format: "bytes=start-end" or "bytes=start-" or "bytes=-suffix"

	if !strings.HasPrefix(rangeHeader, "bytes=") {
		// Invalid range header, serve entire file
		_, err := io.Copy(c.ResponseWriter(), file)
		return err
	}

	rangeSpec := rangeHeader[6:] // Remove "bytes=" prefix

	var start, end int64
	var err error

	if strings.Contains(rangeSpec, "-") {
		parts := strings.Split(rangeSpec, "-")
		if len(parts) != 2 {
			// Invalid range, serve entire file
			_, err := io.Copy(c.ResponseWriter(), file)
			return err
		}

		if parts[0] != "" {
			start, err = strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				_, err := io.Copy(c.ResponseWriter(), file)
				return err
			}
		}

		if parts[1] != "" {
			end, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				_, err := io.Copy(c.ResponseWriter(), file)
				return err
			}
		} else {
			end = fileSize - 1
		}
	}

	// Validate range
	if start < 0 || end >= fileSize || start > end {
		c.Status(416) // Range Not Satisfiable
		c.SetHeader("Content-Range", fmt.Sprintf("bytes */%d", fileSize))
		return nil
	}

	// Set range response headers
	c.Status(206) // Partial Content
	c.SetHeader("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	c.SetHeader("Content-Length", fmt.Sprintf("%d", end-start+1))

	// Seek to start position
	_, err = file.Seek(start, 0)
	if err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	// Copy the requested range
	_, err = io.CopyN(c.ResponseWriter(), file, end-start+1)
	return err
}

// getContentTypeFromFile returns the MIME type based on file extension
// Internal helper for content type detection
func getContentTypeFromFile(filepath string) string {
	ext := strings.ToLower(path.Ext(filepath))

	mimeTypes := map[string]string{
		// Images
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",

		// Documents
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",

		// Text files
		".txt":  "text/plain",
		".csv":  "text/csv",
		".html": "text/html",
		".htm":  "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",

		// Audio
		".mp3": "audio/mpeg",
		".wav": "audio/wav",
		".ogg": "audio/ogg",
		".m4a": "audio/mp4",

		// Video
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mkv":  "video/x-matroska",
		".webm": "video/webm",
		".mov":  "video/quicktime",

		// Archives
		".zip": "application/zip",
		".rar": "application/x-rar-compressed",
		".tar": "application/x-tar",
		".gz":  "application/gzip",
		".7z":  "application/x-7z-compressed",
	}

	if contentType, exists := mimeTypes[ext]; exists {
		return contentType
	}

	return "application/octet-stream"
}

// FileExists checks if a file exists at the given path
// Convenient helper for file existence checks
//
// Parameters:
//   - filepath: Path to check
//
// Returns:
//   - bool: true if file exists
func (c *Context) FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return !os.IsNotExist(err)
}

// GetFileInfo returns file information
// Provides access to file metadata (size, permissions, modification time)
//
// Parameters:
//   - filepath: Path to file
//
// Returns:
//   - os.FileInfo: File information
//   - error: Error if file doesn't exist or can't be accessed
func (c *Context) GetFileInfo(filepath string) (os.FileInfo, error) {
	return os.Stat(filepath)
}

// ResponseWriter returns an io.Writer for the response body
// Allows using standard library functions that write to io.Writer
//
// Returns:
//   - io.Writer: Response body writer
//
// Example:
//
//	fmt.Fprintf(c.ResponseWriter(), "Hello %s", name)
func (c *Context) ResponseWriter() io.Writer {
	return c.RequestCtx.Response.BodyWriter()
}

// State returns application state value from context
// Application state is shared across all requests
// Set via app.SetState()
//
// Parameters:
//   - key: State key
//
// Returns:
//   - interface{}: State value
//   - bool: true if state exists
func (c *Context) State(key string) (interface{}, bool) {
	// Get app from locals
	if app := c.Locals("__app__"); app != nil {
		if blazeApp, ok := app.(*App); ok {
			return blazeApp.GetState(key)
		}
	}
	return nil, false
}

// MustState returns application state or panics if not found
// Use when state is required for handler execution
//
// Parameters:
//   - key: State key
//
// Returns:
//   - interface{}: State value (panics if not found)
func (c *Context) MustState(key string) interface{} {
	value, exists := c.State(key)
	if !exists {
		panic(fmt.Sprintf("state key %s not found", key))
	}
	return value
}

// StateString returns state value as string
// Type-safe state extraction with default value
//
// Parameters:
//   - key: State key
//
// Returns:
//   - string: State value as string or empty string
func (c *Context) StateString(key string) string {
	if value, exists := c.State(key); exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// StateInt returns state value as int
// Type-safe state extraction with default value
//
// Parameters:
//   - key: State key
//
// Returns:
//   - int: State value as int or 0
func (c *Context) StateInt(key string) int {
	if value, exists := c.State(key); exists {
		if i, ok := value.(int); ok {
			return i
		}
	}
	return 0
}

// StateBool returns state value as bool
// Type-safe state extraction with default value
//
// Parameters:
//   - key: State key
//
// Returns:
//   - bool: State value as bool or false
func (c *Context) StateBool(key string) bool {
	if value, exists := c.State(key); exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return false
}

// BindAndValidate binds and validates request body in one call
// Combines binding and validation for convenience
//
// Example:
//
//	var req CreateUserRequest
//	if err := c.BindAndValidate(&req); err != nil {
//	    return c.Status(400).JSON(Map{"error": err.Error()})
//	}
//
// Parameters:
//   - v: Pointer to struct with validation tags
//
// Returns:
//   - error: Binding or validation error
func (c *Context) BindAndValidate(v interface{}) error {
	// Bind the request body
	if err := c.Bind(v); err != nil {
		return fmt.Errorf("binding error: %w", err)
	}

	// Validate the bound struct
	validator := GetValidator()
	if err := validator.ValidateStruct(v); err != nil {
		return err
	}

	return nil
}

// BindJSONAndValidate binds JSON and validates in one call
// JSON-specific version of BindAndValidate
//
// Parameters:
//   - v: Pointer to struct with json and validation tags
//
// Returns:
//   - error: Binding or validation error
func (c *Context) BindJSONAndValidate(v interface{}) error {
	// Bind JSON
	if err := c.BindJSON(v); err != nil {
		return fmt.Errorf("binding error: %w", err)
	}

	// Validate
	validator := GetValidator()
	if err := validator.ValidateStruct(v); err != nil {
		return err
	}

	return nil
}

// BindFormAndValidate binds form data and validates
// Form-specific version of BindAndValidate
//
// Parameters:
//   - v: Pointer to struct with form and validation tags
//
// Returns:
//   - error: Binding or validation error
func (c *Context) BindFormAndValidate(v interface{}) error {
	// Bind form data
	if err := c.BindForm(v); err != nil {
		return fmt.Errorf("binding error: %w", err)
	}

	// Validate
	validator := GetValidator()
	if err := validator.ValidateStruct(v); err != nil {
		return err
	}

	return nil
}

// BindMultipartFormAndValidate binds multipart form and validates
// Multipart form-specific version of BindAndValidate
//
// Parameters:
//   - v: Pointer to struct with form and validation tags
//
// Returns:
//   - error: Binding or validation error
func (c *Context) BindMultipartFormAndValidate(v interface{}) error {
	// Bind multipart form
	if err := c.BindMultipartForm(v); err != nil {
		return fmt.Errorf("binding error: %w", err)
	}

	// Validate
	validator := GetValidator()
	if err := validator.ValidateStruct(v); err != nil {
		return err
	}

	return nil
}

// Validate validates a struct without binding
// Use when struct is already populated
//
// Parameters:
//   - v: Struct to validate
//
// Returns:
//   - error: Validation error or nil
func (c *Context) Validate(v interface{}) error {
	validator := GetValidator()
	return validator.ValidateStruct(v)
}

// ValidateVar validates a single variable against a tag
// Useful for validating individual values
//
// Example:
//
//	email := c.Query("email")
//	if err := c.ValidateVar(email, "required,email"); err != nil {
//	    return c.Status(400).JSON(Map{"error": "Invalid email"})
//	}
//
// Parameters:
//   - field: Value to validate
//   - tag: Validation tag (e.g., "required,email,min=5")
//
// Returns:
//   - error: Validation error or nil
func (c *Context) ValidateVar(field interface{}, tag string) error {
	validator := GetValidator()
	return validator.ValidateVar(field, tag)
}

// Download sends a file as a download with custom filename
// Sets Content-Disposition header for download
//
// Parameters:
//   - filepath: Path to file on server
//   - filename: Filename shown to user
//
// Returns:
//   - error: File read error or nil
//
// Example:
//
//	return c.Download("/data/report.pdf", "Monthly-Report.pdf")
func (c *Context) Download(filepath, filename string) error {
	c.SetHeader("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.SetHeader("Content-Type", "application/octet-stream")
	c.RequestCtx.SendFile(filepath)
	return nil
}

// Attachment is an alias for Download
// Alternative naming for file downloads
//
// Parameters:
//   - filepath: Path to file on server
//   - filename: Filename for download
//
// Returns:
//   - error: File serving error or nil
func (c *Context) AttachmentStatic(filepath, filename string) error {
	return c.Download(filepath, filename)
}

// Logger returns a request-specific logger with context
// Logger includes request metadata (ID, method, path, IP)
//
// Returns:
//   - *Logger: Request-scoped logger
func (c *Context) Logger() *Loggerlog {
	requestID := c.GetUserValueString("request_id")
	return GetDefaultLogger().With(
		"request_id", requestID,
		"method", c.Method(),
		"path", c.Path(),
	)
}

// LogDebug logs a debug message with request context
// Convenience method for request-scoped logging
//
// Parameters:
//   - msg: Log message
//   - args: Key-value pairs for structured logging
func (c *Context) LogDebug(msg string, args ...interface{}) {
	c.Logger().Debug(msg, args...)
}

// LogInfo logs an info message with request context
// Convenience method for request-scoped logging
//
// Parameters:
//   - msg: Log message
//   - args: Key-value pairs for structured logging
func (c *Context) LogInfo(msg string, args ...interface{}) {
	c.Logger().Info(msg, args...)
}

// LogWarn logs a warning message with request context
// Convenience method for request-scoped logging
//
// Parameters:
//   - msg: Log message
//   - args: Key-value pairs for structured logging
func (c *Context) LogWarn(msg string, args ...interface{}) {
	c.Logger().Warn(msg, args...)
}

// LogError logs an error message with request context
// Convenience method for request-scoped logging
//
// Parameters:
//   - msg: Log message
//   - args: Key-value pairs for structured logging
func (c *Context) LogError(msg string, args ...interface{}) {
	c.Logger().Error(msg, args...)
}
