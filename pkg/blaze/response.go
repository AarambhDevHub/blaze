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

// Status sets the response status code
func (c *Context) Status(status int) *Context {
	c.RequestCtx.SetStatusCode(status)
	return c
}

// SetContentType sets response Content-Type
func (c *Context) SetContentType(contentType string) {
	c.RequestCtx.SetContentType(contentType)
}

// ==================== Headers ====================

// SetHeader sets a response header
func (c *Context) SetHeader(key, value string) *Context {
	c.RequestCtx.Response.Header.Set(key, value)
	return c
}

// AddHeader adds a response header (allows multiple values)
func (c *Context) AddHeader(key, value string) *Context {
	c.RequestCtx.Response.Header.Add(key, value)
	return c
}

// DelHeader deletes a response header
func (c *Context) DelHeader(key string) *Context {
	c.RequestCtx.Response.Header.Del(key)
	return c
}

// GetResponseHeader gets a response header value
func (c *Context) GetResponseHeader(key string) string {
	return string(c.RequestCtx.Response.Header.Peek(key))
}

// ==================== Cookies ====================

// SetCookie sets a cookie with basic options
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
func (c *Context) DeleteCookie(name string) *Context {
	cookie := &fasthttp.Cookie{}
	cookie.SetKey(name)
	cookie.SetValue("")
	cookie.SetExpire(time.Now().Add(-24 * time.Hour))
	cookie.SetPath("/")

	c.RequestCtx.Response.Header.SetCookie(cookie)
	return c
}

// CookieOptions defines cookie configuration
type CookieOptions struct {
	Name     string
	Value    string
	Path     string
	Domain   string
	Expires  time.Time
	MaxAge   int
	Secure   bool
	HTTPOnly bool
	SameSite string // "strict", "lax", "none", or ""
}

// ==================== Response Body Writers ====================

// WriteString writes string to response body
func (c *Context) WriteString(s string) (int, error) {
	return c.RequestCtx.WriteString(s)
}

// Write writes bytes to response body
func (c *Context) Write(b []byte) (int, error) {
	return c.RequestCtx.Write(b)
}

// SetBody sets the entire response body
func (c *Context) SetBody(body []byte) *Context {
	c.RequestCtx.Response.SetBody(body)
	return c
}

// AppendBody appends to the response body
func (c *Context) AppendBody(body []byte) *Context {
	c.RequestCtx.Response.AppendBody(body)
	return c
}

// ==================== JSON Responses ====================

// JSON sends a JSON response
func (c *Context) JSON(data interface{}) error {
	c.SetContentType("application/json; charset=utf-8")
	return fastjson.NewEncoder(c.RequestCtx).Encode(data)
}

// JSONStatus sends a JSON response with status code
func (c *Context) JSONStatus(status int, data interface{}) error {
	c.Status(status)
	return c.JSON(data)
}

// JSONPretty sends a pretty-printed JSON response
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
func (c *Context) Text(text string) error {
	c.SetContentType("text/plain; charset=utf-8")
	_, err := c.WriteString(text)
	return err
}

// TextStatus sends a plain text response with status code
func (c *Context) TextStatus(status int, text string) error {
	c.Status(status)
	return c.Text(text)
}

// ==================== HTML Responses ====================

// HTML sends an HTML response
func (c *Context) HTML(html string) error {
	c.SetContentType("text/html; charset=utf-8")
	_, err := c.WriteString(html)
	return err
}

// HTMLStatus sends an HTML response with status code
func (c *Context) HTMLStatus(status int, html string) error {
	c.Status(status)
	return c.HTML(html)
}

// ==================== XML Responses ====================

// XML sends an XML response
func (c *Context) XML(data interface{}) error {
	c.SetContentType("application/xml; charset=utf-8")
	xmlData, err := xml.Marshal(data)
	if err != nil {
		return err
	}
	c.SetBody(xmlData)
	return nil
}

// XMLStatus sends an XML response with status code
func (c *Context) XMLStatus(status int, data interface{}) error {
	c.Status(status)
	return c.XML(data)
}

// ==================== Redirect ====================

// Redirect redirects to the given URL
func (c *Context) Redirect(url string, status ...int) {
	code := fasthttp.StatusFound
	if len(status) > 0 {
		code = status[0]
	}
	c.RequestCtx.Redirect(url, code)
}

// RedirectPermanent redirects permanently (301)
func (c *Context) RedirectPermanent(url string) {
	c.Redirect(url, fasthttp.StatusMovedPermanently)
}

// RedirectTemporary redirects temporarily (302)
func (c *Context) RedirectTemporary(url string) {
	c.Redirect(url, fasthttp.StatusFound)
}

// ==================== Advanced Response Methods ====================

// NoContent sends a 204 No Content response
func (c *Context) NoContent() error {
	c.Status(fasthttp.StatusNoContent)
	return nil
}

// NotModified sends a 304 Not Modified response
func (c *Context) NotModified() error {
	c.Status(fasthttp.StatusNotModified)
	return nil
}

// Stream sets up response for streaming
func (c *Context) Stream(contentType string) io.Writer {
	c.SetContentType(contentType)
	c.RequestCtx.SetBodyStreamWriter(func(w *bufio.Writer) {
		// Writer will be used by caller
	})
	return c.RequestCtx.Response.BodyWriter()
}

// SendStatus sends only a status code with no body
func (c *Context) SendStatus(status int) error {
	c.Status(status)
	return nil
}

// Attachment sends a file as an attachment
func (c *Context) Attachment(filepath, filename string) error {
	if filename != "" {
		c.SetHeader("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	}
	c.RequestCtx.SendFile(filepath)
	return nil
}

// ==================== Response Convenience Methods ====================

// Type sets the Content-Type header (shorthand)
func (c *Context) Type(contentType string) *Context {
	c.SetContentType(contentType)
	return c
}

// Vary adds the Vary header
func (c *Context) Vary(header string) *Context {
	return c.AddHeader("Vary", header)
}

// Location sets the Location header
func (c *Context) Location(url string) *Context {
	return c.SetHeader("Location", url)
}

// ContentLength sets the Content-Length header
func (c *Context) ContentLength(length int64) *Context {
	return c.SetHeader("Content-Length", strconv.FormatInt(length, 10))
}

// Etag sets the ETag header
func (c *Context) Etag(etag string) *Context {
	return c.SetHeader("ETag", etag)
}

// LastModified sets the Last-Modified header
func (c *Context) LastModified(t time.Time) *Context {
	return c.SetHeader("Last-Modified", t.UTC().Format(http.TimeFormat))
}

// CacheControl sets the Cache-Control header
func (c *Context) CacheControl(value string) *Context {
	return c.SetHeader("Cache-Control", value)
}

// Expires sets the Expires header
func (c *Context) Expires(t time.Time) *Context {
	return c.SetHeader("Expires", t.UTC().Format(http.TimeFormat))
}
