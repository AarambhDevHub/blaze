package blaze

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/valyala/fasthttp"
)

var fastjson = jsoniter.ConfigCompatibleWithStandardLibrary

// Context represents the request context
type Context struct {
	*fasthttp.RequestCtx
	params map[string]string
	locals map[string]interface{}
}

// Map is a shortcut for map[string]interface{}
type Map map[string]interface{}

// Param returns the route parameter value
func (c *Context) Param(key string) string {
	return c.params[key]
}

// SetParam sets a route parameter
func (c *Context) SetParam(key, value string) {
	if c.params == nil {
		c.params = make(map[string]string)
	}
	c.params[key] = value
}

// Query returns the query parameter value
func (c *Context) Query(key string) string {
	return string(c.RequestCtx.QueryArgs().Peek(key))
}

// QueryArgs returns fasthttp query args (PROPERLY EXPOSED)
func (c *Context) QueryArgs() *fasthttp.Args {
	return c.RequestCtx.QueryArgs()
}

// QueryDefault returns the query parameter value or default if not found
func (c *Context) QueryDefault(key, defaultValue string) string {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// QueryInt returns the query parameter as integer
func (c *Context) QueryInt(key string) (int, error) {
	value := c.Query(key)
	if value == "" {
		return 0, fmt.Errorf("parameter '%s' not found", key)
	}
	return strconv.Atoi(value)
}

// QueryIntDefault returns the query parameter as integer or default
func (c *Context) QueryIntDefault(key string, defaultValue int) int {
	value, err := c.QueryInt(key)
	if err != nil {
		return defaultValue
	}
	return value
}

// Header returns the request header value
func (c *Context) Header(key string) string {
	return string(c.RequestCtx.Request.Header.Peek(key))
}

// Request returns the fasthttp request (PROPERLY EXPOSED)
func (c *Context) Request() *fasthttp.Request {
	return &c.RequestCtx.Request
}

// Response returns the fasthttp response (PROPERLY EXPOSED)
func (c *Context) Response() *fasthttp.Response {
	return &c.RequestCtx.Response
}

// SetHeader sets a response header
func (c *Context) SetHeader(key, value string) *Context {
	c.RequestCtx.Response.Header.Set(key, value)
	return c
}

// Status sets the response status code
func (c *Context) Status(status int) *Context {
	c.RequestCtx.SetStatusCode(status)
	return c
}

// SetContentType sets response Content-Type (FIXED METHOD)
func (c *Context) SetContentType(contentType string) {
	c.RequestCtx.SetContentType(contentType)
}

// WriteString writes string to response body (FIXED METHOD)
func (c *Context) WriteString(s string) (int, error) {
	return c.RequestCtx.WriteString(s)
}

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

// Redirect redirects to the given URL (FIXED)
func (c *Context) Redirect(url string, status ...int) {
	code := fasthttp.StatusFound
	if len(status) > 0 {
		code = status[0]
	}
	c.RequestCtx.Redirect(url, code)
}

// Body returns the request body (FIXED METHOD)
func (c *Context) Body() []byte {
	return c.RequestCtx.PostBody()
}

// PostBody returns the request body (PROPERLY EXPOSED)
func (c *Context) PostBody() []byte {
	return c.RequestCtx.PostBody()
}

// BodyString returns the request body as string
func (c *Context) BodyString() string {
	return string(c.Body())
}

// Bind binds the request body to a struct
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
func (c *Context) BindJSON(v interface{}) error {
	return fastjson.Unmarshal(c.Body(), v)
}

// BindForm binds form data to a struct (simplified implementation)
func (c *Context) BindForm(v interface{}) error {
	// This is a simplified implementation
	// In a real framework, you'd use reflection to map form fields to struct fields
	return fmt.Errorf("form binding not implemented in this example")
}

// Cookie returns the cookie value
func (c *Context) Cookie(name string) string {
	return string(c.RequestCtx.Request.Header.Cookie(name))
}

// SetCookie sets a cookie
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

// Locals returns a local variable
func (c *Context) Locals(key string) interface{} {
	if c.locals == nil {
		return nil
	}
	return c.locals[key]
}

// SetLocals sets a local variable
func (c *Context) SetLocals(key string, value interface{}) *Context {
	if c.locals == nil {
		c.locals = make(map[string]interface{})
	}
	c.locals[key] = value
	return c
}

// Method returns the request method
func (c *Context) Method() string {
	return string(c.RequestCtx.Method())
}

// Path returns the request path (FIXED METHOD)
func (c *Context) Path() string {
	return string(c.RequestCtx.Path())
}

// URI returns the request URI (PROPERLY EXPOSED)
func (c *Context) URI() *fasthttp.URI {
	return c.RequestCtx.URI()
}

// IP returns the client IP address (FIXED METHOD)
func (c *Context) IP() string {
	return c.RequestCtx.RemoteIP().String()
}

// RemoteIP returns the remote IP (PROPERLY EXPOSED)
func (c *Context) RemoteIP() net.IP {
	return c.RequestCtx.RemoteIP()
}

// UserAgent returns the User-Agent header
func (c *Context) UserAgent() string {
	return string(c.RequestCtx.UserAgent())
}
