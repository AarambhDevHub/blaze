package blaze

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

// Response helpers

// OK returns a standard OK response
func OK(data interface{}) Map {
	return Map{
		"success": true,
		"data":    data,
	}
}

// Error returns a standard error response
func Error(message string) Map {
	return Map{
		"success": false,
		"error":   message,
	}
}

// Created returns a standard created response
func Created(data interface{}) Map {
	return Map{
		"success": true,
		"data":    data,
		"message": "Resource created successfully",
	}
}

// NoContent returns an empty response with 204 status
func NoContent(c *Context) error {
	return c.Status(204).Text("")
}

// BadRequest returns a 400 bad request response
func BadRequest(c *Context, message string) error {
	return c.Status(400).JSON(Error(message))
}

// Unauthorized returns a 401 unauthorized response
func Unauthorized(c *Context, message string) error {
	return c.Status(401).JSON(Error(message))
}

// Forbidden returns a 403 forbidden response
func Forbidden(c *Context, message string) error {
	return c.Status(403).JSON(Error(message))
}

// NotFound returns a 404 not found response
func NotFound(c *Context, message string) error {
	return c.Status(404).JSON(Error(message))
}

// InternalServerError returns a 500 internal server error response
func InternalServerError(c *Context, message string) error {
	return c.Status(500).JSON(Error(message))
}

// // Redirect function sends an HTTP redirect to the given URL with the specified status code
// func Redirect(c Context, url string, statusCode int) error {
// 	c.SetHeader("Location", url)
// 	return c.Status(statusCode).Text("") // empty body is common for redirects
// }

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PerPage    int         `json:"per_page"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// Paginate creates a paginated response
func Paginate(data interface{}, total, page, perPage int) *PaginatedResponse {
	totalPages := (total + perPage - 1) / perPage

	return &PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// HealthCheck response
type HealthCheck struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

// Health returns a health check response
func Health(version, uptime string) *HealthCheck {
	return &HealthCheck{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   version,
		Uptime:    uptime,
	}
}

// Status sets the HTTP status code for the response
// Chainable method that returns the context for fluent API style
//
// Common Status Codes:
//   - 200: OK (successful request)
//   - 201: Created (successful resource creation)
//   - 204: No Content (successful with no response body)
//   - 400: Bad Request (client error)
//   - 401: Unauthorized (authentication required)
//   - 403: Forbidden (insufficient permissions)
//   - 404: Not Found (resource doesn't exist)
//   - 500: Internal Server Error (server-side error)
//
// Parameters:
//   - status: HTTP status code
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	return c.Status(200).JSON(data)
//	return c.Status(404).Text("Not found")
func (c *Context) Status(status int) *Context {
	c.RequestCtx.SetStatusCode(status)
	return c
}

// SetContentType sets the Content-Type header for the response
// Determines how the client interprets the response body
//
// Common Content Types:
//   - application/json: JSON data
//   - text/html: HTML documents
//   - text/plain: Plain text
//   - application/xml: XML data
//   - application/octet-stream: Binary data
//
// Parameters:
//   - contentType: MIME type string
//
// Example:
//
//	c.SetContentType("application/json; charset=utf-8")
func (c *Context) SetContentType(contentType string) {
	c.RequestCtx.SetContentType(contentType)
}

// ==================== Headers ====================

// SetHeader sets a response header
// Replaces existing header with same name
//
// Parameters:
//   - key: Header name
//   - value: Header value
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.SetHeader("Cache-Control", "no-cache")
func (c *Context) SetHeader(key, value string) *Context {
	c.RequestCtx.Response.Header.Set(key, value)
	return c
}

// AddHeader adds a response header (allows multiple values)
// Appends to existing headers with same name
//
// Parameters:
//   - key: Header name
//   - value: Header value to add
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.AddHeader("Set-Cookie", "session=abc123")
//	c.AddHeader("Set-Cookie", "user=john")
func (c *Context) AddHeader(key, value string) *Context {
	c.RequestCtx.Response.Header.Add(key, value)
	return c
}

// DelHeader deletes a response header
// Removes all values for the specified header
//
// Parameters:
//   - key: Header name to delete
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.DelHeader("X-Powered-By")
func (c *Context) DelHeader(key string) *Context {
	c.RequestCtx.Response.Header.Del(key)
	return c
}

// GetResponseHeader gets a response header value
// Returns empty string if header doesn't exist
//
// Parameters:
//   - key: Header name
//
// Returns:
//   - string: Header value or empty string
//
// Example:
//
//	contentType := c.GetResponseHeader("Content-Type")
func (c *Context) GetResponseHeader(key string) string {
	return string(c.RequestCtx.Response.Header.Peek(key))
}

// ==================== Cookies ====================

// SetCookie sets a cookie with basic options
// Uses fasthttp cookie with configurable expiration
//
// Cookie Security:
//   - Always use Secure flag in production (HTTPS only)
//   - Use HttpOnly to prevent JavaScript access
//   - Set appropriate SameSite for CSRF protection
//   - Use short expiration for sensitive cookies
//
// Parameters:
//   - name: Cookie name
//   - value: Cookie value
//   - expires: Optional expiration time
//
// Returns:
//   - *Context: Context for method chaining
//
// Example - Session Cookie (expires when browser closes):
//
//	c.SetCookie("session", "abc123")
//
// Example - Persistent Cookie:
//
//	c.SetCookie("remember_me", "user123", time.Now().Add(7*24*time.Hour))
func (c *Context) SetCookie(name, value string, expires ...time.Time) *Context {
	cookie := &fasthttp.Cookie{}
	cookie.SetKey(name)
	cookie.SetValue(value)

	if len(expires) > 0 {
		cookie.SetExpire(expires[0])
	}

	c.RequestCtx.Response.Header.SetCookie(cookie)
	return c
}

// SetCookieAdvanced sets a cookie with all available options
// Provides complete control over cookie attributes
//
// Use Cases:
//   - Session management with security flags
//   - Authentication cookies with strict settings
//   - Tracking cookies with domain/path restrictions
//   - API tokens with expiration
//
// Parameters:
//   - cookie: Complete cookie configuration
//
// Returns:
//   - *Context: Context for method chaining
//
// Example - Secure Session Cookie:
//
//	c.SetCookieAdvanced(blaze.CookieOptions{
//	    Name: "session",
//	    Value: sessionToken,
//	    Path: "/",
//	    MaxAge: 3600, // 1 hour
//	    Secure: true,
//	    HTTPOnly: true,
//	    SameSite: "Strict",
//	})
//
// Example - Remember Me Cookie:
//
//	c.SetCookieAdvanced(blaze.CookieOptions{
//	    Name: "remember_me",
//	    Value: rememberToken,
//	    Path: "/",
//	    MaxAge: 30*24*3600, // 30 days
//	    Secure: true,
//	    HTTPOnly: true,
//	    SameSite: "Lax",
//	})
//
// Example - API Token Cookie:
//
//	c.SetCookieAdvanced(blaze.CookieOptions{
//	    Name: "api_token",
//	    Value: apiToken,
//	    Path: "/api",
//	    Domain: ".example.com", // All subdomains
//	    MaxAge: 7200, // 2 hours
//	    Secure: true,
//	    HTTPOnly: true,
//	    SameSite: "Strict",
//	})
func (c *Context) SetCookieAdvanced(cookie *CookieOptions) *Context {
	fhCookie := &fasthttp.Cookie{}
	fhCookie.SetKey(cookie.Name)
	fhCookie.SetValue(cookie.Value)

	if cookie.Path != "" {
		fhCookie.SetPath(cookie.Path)
	}

	if cookie.Domain != "" {
		fhCookie.SetDomain(cookie.Domain)
	}

	if !cookie.Expires.IsZero() {
		fhCookie.SetExpire(cookie.Expires)
	}

	if cookie.MaxAge > 0 {
		fhCookie.SetMaxAge(cookie.MaxAge)
	}

	fhCookie.SetSecure(cookie.Secure)
	fhCookie.SetHTTPOnly(cookie.HTTPOnly)

	// Set SameSite attribute
	switch strings.ToLower(cookie.SameSite) {
	case "strict":
		fhCookie.SetSameSite(fasthttp.CookieSameSiteStrictMode)
	case "lax":
		fhCookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	case "none":
		fhCookie.SetSameSite(fasthttp.CookieSameSiteNoneMode)
	default:
		fhCookie.SetSameSite(fasthttp.CookieSameSiteDefaultMode)
	}

	c.RequestCtx.Response.Header.SetCookie(fhCookie)
	return c
}

// DeleteCookie deletes a cookie by setting it to expire
// Sets expiration to past time and clears value
//
// Cookie Deletion:
//   - Sets expiration to 24 hours ago
//   - Clears cookie value
//   - Matches path to ensure proper deletion
//   - Must match original domain for domain cookies
//
// Parameters:
//   - name: Cookie name to delete
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.DeleteCookie("session")
func (c *Context) DeleteCookie(name string) *Context {
	cookie := &fasthttp.Cookie{}
	cookie.SetKey(name)
	cookie.SetValue("")
	cookie.SetExpire(time.Now().Add(-24 * time.Hour))
	cookie.SetPath("/")

	c.RequestCtx.Response.Header.SetCookie(cookie)
	return c
}

// ClearCookie is an alias for DeleteCookie
// Provides alternative naming for cookie deletion
//
// Parameters:
//   - name: Cookie name to clear
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.ClearCookie("session")
func (c *Context) ClearCookie(name string) *Context {
	return c.DeleteCookie(name)
}

// ClearAllCookies deletes all cookies from the request
// Iterates through request cookies and expires each one
//
// Use Cases:
//   - User logout (clear all session data)
//   - Privacy features (clear tracking)
//   - Account deletion
//   - Security incidents
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	func logout(c *blaze.Context) error {
//	    c.ClearAllCookies()
//	    return c.Redirect("/login")
//	}
func (c *Context) ClearAllCookies() *Context {
	c.RequestCtx.Request.Header.VisitAllCookie(func(key, value []byte) {
		c.DeleteCookie(string(key))
	})
	return c
}

// Cookies returns all cookies from the request
// Parses and returns a map of cookie name to value
//
// Returns:
//   - map[string]string: Map of cookie names to values
//
// Example:
//
//	cookies := c.Cookies()
//	for name, value := range cookies {
//	    log.Printf("Cookie: %s = %s", name, value)
//	}
func (c *Context) Cookies() map[string]string {
	cookies := make(map[string]string)
	c.RequestCtx.Request.Header.VisitAllCookie(func(key, value []byte) {
		cookies[string(key)] = string(value)
	})
	return cookies
}

// CookieOptions defines comprehensive cookie configuration
// Provides fine-grained control over cookie behavior and security
//
// Cookie Security Best Practices:
//   - Secure: true in production (HTTPS only)
//   - HttpOnly: true for session/auth cookies (prevents XSS)
//   - SameSite: "Strict" or "Lax" for CSRF protection
//   - Domain: Set explicitly for subdomain sharing
//   - Path: Restrict to specific paths when possible
//   - MaxAge: Use short expiration for sensitive cookies
//
// SameSite Values:
//   - "Strict": Cookie only sent for same-site requests (most secure)
//   - "Lax": Cookie sent for top-level navigation (balanced)
//   - "None": Cookie sent for all requests (requires Secure=true)
type CookieOptions struct {
	// Name is the cookie name
	Name string

	// Value is the cookie value
	Value string

	// Path restricts the cookie to a URL path
	// Default: "/" (entire site)
	// Example: "/admin" (only /admin/* paths)
	Path string

	// Domain specifies which hosts can receive the cookie
	// Empty: Cookie only sent to origin server
	// ".example.com": Cookie sent to example.com and all subdomains
	Domain string

	// Expires sets the absolute expiration time
	// If both Expires and MaxAge are set, MaxAge takes precedence
	Expires time.Time

	// MaxAge specifies cookie lifetime in seconds
	// Positive: Cookie expires after N seconds
	// Zero or negative: Cookie expires immediately (delete)
	// Not set: Session cookie (expires when browser closes)
	MaxAge int

	// Secure when true, cookie only sent over HTTPS
	// Must be true in production
	// Prevents man-in-the-middle attacks
	// Default: false (set to true in production)
	Secure bool

	// HTTPOnly when true, cookie not accessible via JavaScript
	// Prevents XSS attacks from stealing cookies
	// Should be true for authentication/session cookies
	// Default: true (recommended)
	HTTPOnly bool

	// SameSite controls cross-site cookie behavior
	// Values: "Strict", "Lax", "None", or empty
	// Strict: Cookie only for same-site requests
	// Lax: Cookie for top-level navigation + same-site
	// None: Cookie for all requests (requires Secure=true)
	// Default: "Lax" (balanced security and functionality)
	SameSite string
}

// ==================== Response Body Writers ====================

// WriteString writes string to response body
// Low-level method for direct writing
//
// Parameters:
//   - s: String to write
//
// Returns:
//   - int: Number of bytes written
//   - error: Write error or nil
//
// Example:
//
//	c.WriteString("Hello")
func (c *Context) WriteString(s string) (int, error) {
	return c.RequestCtx.WriteString(s)
}

// Write writes bytes to response body
// Implements io.Writer interface
//
// Parameters:
//   - b: Bytes to write
//
// Returns:
//   - int: Number of bytes written
//   - error: Write error or nil
//
// Example:
//
//	c.Write([]byte("Hello"))
func (c *Context) Write(b []byte) (int, error) {
	return c.RequestCtx.Write(b)
}

// SetBody sets the entire response body
// Replaces any existing body content
//
// Parameters:
//   - body: Complete body content
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.SetBody([]byte("Response body"))
func (c *Context) SetBody(body []byte) *Context {
	c.RequestCtx.Response.SetBody(body)
	return c
}

// AppendBody appends to the response body
// Adds content to existing body
//
// Parameters:
//   - body: Content to append
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.AppendBody([]byte("More content"))
func (c *Context) AppendBody(body []byte) *Context {
	c.RequestCtx.Response.AppendBody(body)
	return c
}

// ==================== JSON Responses ====================

// JSON sends a JSON response with 200 status
// Automatically sets Content-Type to application/json
//
// Parameters:
//   - data: Data to serialize as JSON
//
// Returns:
//   - error: JSON encoding error or nil
//
// Example:
//
//	return c.JSON(blaze.Map{"message": "Hello"})
//	return c.JSON(user)
func (c *Context) JSON(data interface{}) error {
	c.SetContentType("application/json; charset=utf-8")
	return fastjson.NewEncoder(c.RequestCtx).Encode(data)
}

// JSONStatus sends a JSON response with custom status code
// Combines Status() and JSON() in one call
//
// Parameters:
//   - status: HTTP status code
//   - data: Data to serialize as JSON
//
// Returns:
//   - error: JSON encoding error or nil
//
// Example:
//
//	return c.JSONStatus(201, createdUser)
//	return c.JSONStatus(404, blaze.Map{"error": "Not found"})
func (c *Context) JSONStatus(status int, data interface{}) error {
	c.Status(status)
	return c.JSON(data)
}

// JSONPretty sends a pretty-printed JSON response
// Useful for debugging and human-readable responses
//
// Parameters:
//   - data: Data to serialize as JSON
//   - indent: Indentation string (e.g., "  " for 2 spaces)
//
// Returns:
//   - error: JSON encoding error or nil
//
// Example:
//
//	return c.JSONPretty(data, "  ")
func (c *Context) JSONPretty(data interface{}, indent string) error {
	c.SetContentType("application/json; charset=utf-8")
	jsonData, err := json.MarshalIndent(data, "", indent)
	if err != nil {
		return err
	}
	c.SetBody(jsonData)
	return nil
}

// ==================== Text Responses ====================

// Text sends a plain text response
// Automatically sets Content-Type to text/plain
//
// Parameters:
//   - text: Text string to send
//
// Returns:
//   - error: Write error or nil
//
// Example:
//
//	return c.Text("Hello, World!")
func (c *Context) Text(text string) error {
	c.SetContentType("text/plain; charset=utf-8")
	_, err := c.WriteString(text)
	return err
}

// TextStatus sends a plain text response with custom status
// Combines Status() and Text() in one call
//
// Parameters:
//   - status: HTTP status code
//   - text: Text string to send
//
// Returns:
//   - error: Write error or nil
//
// Example:
//
//	return c.TextStatus(404, "Page not found")
func (c *Context) TextStatus(status int, text string) error {
	c.Status(status)
	return c.Text(text)
}

// ==================== HTML Responses ====================

// HTML sends an HTML response
// Automatically sets Content-Type to text/html
//
// Parameters:
//   - html: HTML string to send
//
// Returns:
//   - error: Write error or nil
//
// Example:
//
//	return c.HTML("<h1>Welcome</h1>")
func (c *Context) HTML(html string) error {
	c.SetContentType("text/html; charset=utf-8")
	_, err := c.WriteString(html)
	return err
}

// HTMLStatus sends an HTML response with custom status
// Combines Status() and HTML() in one call
//
// Parameters:
//   - status: HTTP status code
//   - html: HTML string to send
//
// Returns:
//   - error: Write error or nil
//
// Example:
//
//	return c.HTMLStatus(404, "<h1>Page Not Found</h1>")
func (c *Context) HTMLStatus(status int, html string) error {
	c.Status(status)
	return c.HTML(html)
}

// ==================== XML Responses ====================

// XML sends an XML response
// Automatically sets Content-Type to application/xml
//
// Parameters:
//   - data: Data to serialize as XML
//
// Returns:
//   - error: XML encoding error or nil
//
// Example:
//
//	return c.XML(xmlData)
func (c *Context) XML(data interface{}) error {
	c.SetContentType("application/xml; charset=utf-8")
	xmlData, err := xml.Marshal(data)
	if err != nil {
		return err
	}
	c.SetBody(xmlData)
	return nil
}

// XMLStatus sends an XML response with custom status
// Combines Status() and XML() in one call
//
// Parameters:
//   - status: HTTP status code
//   - data: Data to serialize as XML
//
// Returns:
//   - error: XML encoding error or nil
//
// Example:
//
//	return c.XMLStatus(201, createdResource)
func (c *Context) XMLStatus(status int, data interface{}) error {
	c.Status(status)
	return c.XML(data)
}

// ==================== Redirect ====================

// Redirect redirects to the given URL
// Default status is 302 Found (temporary redirect)
//
// Parameters:
//   - url: Destination URL
//   - status: Optional status code (default 302)
//
// Returns:
//   - error: Always returns nil
//
// Example:
//
//	return c.Redirect("/login")
//	return c.Redirect("/new-path", 301)
func (c *Context) Redirect(url string, status ...int) {
	code := fasthttp.StatusFound
	if len(status) > 0 {
		code = status[0]
	}
	c.RequestCtx.Redirect(url, code)
}

// RedirectPermanent redirects permanently (301)
// Use when URL has permanently moved
//
// Parameters:
//   - url: New permanent location
//
// Returns:
//   - error: Always returns nil
//
// Example:
//
//	return c.RedirectPermanent("/new-location")
func (c *Context) RedirectPermanent(url string) {
	c.Redirect(url, fasthttp.StatusMovedPermanently)
}

// RedirectTemporary redirects temporarily (302)
// Use when URL may change back in the future
//
// Parameters:
//   - url: Temporary location
//
// Returns:
//   - error: Always returns nil
//
// Example:
//
//	return c.RedirectTemporary("/maintenance")
func (c *Context) RedirectTemporary(url string) {
	c.Redirect(url, fasthttp.StatusFound)
}

// ==================== Advanced Response Methods ====================

// NoContent sends a 204 No Content response
// Indicates success with no response body
//
// Returns:
//   - error: Always returns nil
//
// Example:
//
//	return c.NoContent() // For DELETE operations
func (c *Context) NoContent() error {
	c.Status(fasthttp.StatusNoContent)
	return nil
}

// NotModified sends a 304 Not Modified response
// Indicates cached resource is still valid
//
// Returns:
//   - error: Always returns nil
//
// Example:
//
//	if clientEtag == serverEtag {
//	    return c.NotModified()
//	}
func (c *Context) NotModified() error {
	c.Status(fasthttp.StatusNotModified)
	return nil
}

// Stream sets up response for streaming
// Configures response for incremental data writing
//
// Parameters:
//   - contentType: MIME type of streamed data
//
// Returns:
//   - io.Writer: Stream writer
//
// Example:
//
//	writer := c.Stream("text/event-stream")
//	fmt.Fprintf(writer, "data: %s\n\n", event)
func (c *Context) Stream(contentType string) io.Writer {
	c.SetContentType(contentType)
	c.RequestCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
		// Writer will be used by caller
	})
	return c.RequestCtx.Response.BodyWriter()
}

// SendStatus sends only a status code with no response body
// Convenient for responses that don't need a body
//
// Parameters:
//   - status: HTTP status code
//
// Returns:
//   - error: Always returns nil
//
// Example:
//
//	return c.SendStatus(204) // No Content
func (c *Context) SendStatus(status int) error {
	c.Status(status)
	return nil
}

// Attachment is an alias for Download
// Sends file as downloadable attachment
//
// Parameters:
//   - filepath: Path to file
//   - filename: Download filename
//
// Returns:
//   - error: File error or nil
func (c *Context) Attachment(filepath, filename string) error {
	if filename != "" {
		c.SetHeader("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}
	c.RequestCtx.SendFile(filepath)
	return nil
}

// ==================== Response Convenience Methods ====================

// Type sets the Content-Type header (shorthand for SetContentType)
// Convenient alias for consistency with other frameworks
//
// Parameters:
//   - contentType: MIME type string
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.Type("application/json").JSON(data)
func (c *Context) Type(contentType string) *Context {
	c.SetContentType(contentType)
	return c
}

// Vary adds the Vary header
// Indicates which request headers affect response
//
// Parameters:
//   - header: Header name that affects response
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.Vary("Accept-Encoding").JSON(data)
func (c *Context) Vary(header string) *Context {
	return c.AddHeader("Vary", header)
}

// Location sets the Location header for redirects
// Used with 3xx status codes
//
// Parameters:
//   - url: Redirect destination URL
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.Status(302).Location("/new-path")
func (c *Context) Location(url string) *Context {
	return c.SetHeader("Location", url)
}

// ContentLength sets the Content-Length header
// Indicates size of response body in bytes
//
// Parameters:
//   - length: Body size in bytes
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.ContentLength(1024).SendFile("file.txt")
func (c *Context) ContentLength(length int64) *Context {
	return c.SetHeader("Content-Length", strconv.FormatInt(length, 10))
}

// Etag sets the ETag header for caching
// Used for cache validation
//
// Parameters:
//   - etag: Entity tag value
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.Etag(`"abc123"`).JSON(data)
func (c *Context) Etag(etag string) *Context {
	return c.SetHeader("ETag", etag)
}

// LastModified sets the Last-Modified header
// Indicates when resource was last changed
//
// Parameters:
//   - t: Last modification time
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.LastModified(time.Now()).SendFile("file.txt")
func (c *Context) LastModified(t time.Time) *Context {
	return c.SetHeader("Last-Modified", t.UTC().Format(http.TimeFormat))
}

// CacheControl sets the Cache-Control header
// Controls caching behavior
//
// Parameters:
//   - value: Cache control directive
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.CacheControl("public, max-age=3600").JSON(data)
func (c *Context) CacheControl(value string) *Context {
	return c.SetHeader("Cache-Control", value)
}

// Expires sets the Expires header
// Specifies when response expires
//
// Parameters:
//   - t: Expiration time
//
// Returns:
//   - *Context: Context for method chaining
//
// Example:
//
//	c.Expires(time.Now().Add(24 * time.Hour)).JSON(data)
func (c *Context) Expires(t time.Time) *Context {
	return c.SetHeader("Expires", t.UTC().Format(http.TimeFormat))
}
