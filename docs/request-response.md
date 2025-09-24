# Request-Response Documentation

## Overview

The request-response documentation covers how to work with HTTP requests and responses in the Blaze Go web framework. Blaze provides a comprehensive `Context` type that wraps FastHTTP functionality and offers extensive methods for handling both incoming requests and outgoing responses.

## Context Structure

The `Context` struct serves as the central interface for request-response handling:

```go
type Context struct {
    *fasthttp.RequestCtx
    params map[string]string
    locals map[string]interface{}
}
```

The context embeds FastHTTP's `RequestCtx` and extends it with parameter storage and local variables.

## Request Handling

### Route Parameters

**Basic Parameter Access:**
```go
func (c *Context) Param(key string) string
func (c *Context) ParamInt(key string) (int, error)
func (c *Context) ParamIntDefault(key string, defaultValue int) int
```

Parameters are extracted from URL paths like `/users/:id` and accessed via `c.Param("id")`.

### Query Parameters

**Query String Handling:**
```go
func (c *Context) Query(key string) string
func (c *Context) QueryDefault(key, defaultValue string) string
func (c *Context) QueryInt(key string) (int, error)
func (c *Context) QueryIntDefault(key string, defaultValue int) int
func (c *Context) QueryArgs() *fasthttp.Args
```

These methods provide easy access to URL query parameters with type conversion and default value support.

### Request Headers

**Header Access:**
```go
func (c *Context) Header(key string) string
func (c *Context) UserAgent() string
```

Headers can be accessed individually or through the underlying FastHTTP request object.

### Request Body

**Body Handling:**
```go
func (c *Context) Body() []byte
func (c *Context) PostBody() []byte
func (c *Context) BodyString() string
func (c *Context) Bind(v interface{}) error
func (c *Context) BindJSON(v interface{}) error
```

The framework supports automatic content type detection and binding to structs, with JSON being the primary supported format.

### Request Metadata

**Request Information:**
```go
func (c *Context) Method() string
func (c *Context) Path() string
func (c *Context) URI() *fasthttp.URI
func (c *Context) IP() string
func (c *Context) RemoteIP() net.IP
func (c *Context) GetContentType() string
```

These methods provide access to essential request metadata including HTTP method, path, client IP, and content type.

## Response Handling

### Status Codes

**Status Setting:**
```go
func (c *Context) Status(status int) *Context
```

This method sets the HTTP status code and returns the context for method chaining.

### Response Headers

**Header Management:**
```go
func (c *Context) SetHeader(key, value string) *Context
func (c *Context) SetContentType(contentType string)
```

Response headers can be set individually with the framework handling proper formatting.

### Response Body

**Content Sending:**
```go
func (c *Context) JSON(data interface{}) error
func (c *Context) JSONStatus(status int, data interface{}) error
func (c *Context) Text(text string) error
func (c *Context) TextStatus(status int, text string) error
func (c *Context) HTML(html string) error
func (c *Context) HTMLStatus(status int, html string) error
```

The framework provides convenient methods for sending JSON, plain text, and HTML responses with automatic content-type setting.

### Redirects

**Redirection:**
```go
func (c *Context) Redirect(url string, status ...int)
```

Supports HTTP redirects with customizable status codes, defaulting to 302 Found.

## Advanced Features

### Cookies

**Cookie Management:**
```go
func (c *Context) Cookie(name string) string
func (c *Context) SetCookie(name, value string, expires ...time.Time) *Context
```

Full cookie support for reading incoming cookies and setting response cookies.

### Local Variables

**Context Storage:**
```go
func (c *Context) Locals(key string) interface{}
func (c *Context) SetLocals(key string, value interface{}) *Context
```

Local variables allow storing data within the request context for use across middleware and handlers.

### File Operations

**File Serving:**
```go
func (c *Context) SendFile(filepath string) error
func (c *Context) ServeFile(filepath string) error
func (c *Context) ServeFileDownload(filepath, filename string) error
func (c *Context) ServeFileInline(filepath string) error
func (c *Context) StreamFile(filepath string) error
```

Comprehensive file serving capabilities including downloads, inline display, and streaming with range request support.

### Form Handling

**Form Data:**
```go
func (c *Context) FormValue(name string) string
func (c *Context) FormValues(name string) []string
func (c *Context) IsMultipartForm() bool
```

Support for both URL-encoded and multipart form data with automatic detection.

## Multipart File Uploads

**File Upload Handling:**
```go
func (c *Context) MultipartForm() (*MultipartForm, error)
func (c *Context) FormFile(name string) (*MultipartFile, error)
func (c *Context) FormFiles(name string) ([]*MultipartFile, error)
func (c *Context) SaveUploadedFile(file *MultipartFile, dst string) error
```

Complete multipart file upload support with validation, temporary storage, and saving capabilities.

## HTTP/2 Support

**Protocol Detection:**
```go
func (c *Context) IsHTTP2() bool
func (c *Context) Protocol() string
func (c *Context) StreamID() uint32
func (c *Context) ServerPush(path, contentType string) error
```

Built-in HTTP/2 protocol detection and server push functionality.

## Response Helpers

**Standard Responses:**
```go
func OK(data interface{}) Map
func Error(message string) Map
func Created(data interface{}) Map
func BadRequest(c *Context, message string) error
func NotFound(c *Context, message string) error
func InternalServerError(c *Context, message string) error
```

Predefined response helpers for common HTTP status codes and response patterns.

## Graceful Shutdown Support

**Shutdown Awareness:**
```go
func (c *Context) IsShuttingDown() bool
func (c *Context) ShutdownContext() context.Context
func (c *Context) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc)
```

The framework provides shutdown-aware request handling that can gracefully terminate long-running operations.

## FastHTTP Integration

**Direct Access:**
```go
func (c *Context) Request() *fasthttp.Request
func (c *Context) Response() *fasthttp.Response
func (c *Context) ResponseWriter() io.Writer
```

Full access to underlying FastHTTP request and response objects for advanced use cases.

## Error Handling

The framework includes structured error responses and panic recovery:

```go
type ErrorResponse struct {
    Success bool   `json:"success"`
    Error   string `json:"error"`
}
```

All response methods return errors that can be handled by middleware or the application's error handling system.
