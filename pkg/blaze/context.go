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

// ParamInt returns the route parameter as integer
func (c *Context) ParamInt(key string) (int, error) {
	value := c.Param(key)
	if value == "" {
		return 0, fmt.Errorf("parameter %s not found", key)
	}
	return strconv.Atoi(value)
}

// ParamIntDefault returns the route parameter as integer or default
func (c *Context) ParamIntDefault(key string, defaultValue int) int {
	value, err := c.ParamInt(key)
	if err != nil {
		return defaultValue
	}
	return value
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

// MarshalJSON converts blaze.Map to JSON bytes
func (m Map) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}(m))
}

// ToJSON converts blaze.Map to JSON string
func (m Map) ToJSON() (string, error) {
	data, err := json.Marshal(map[string]interface{}(m))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToJSONBytes converts blaze.Map to JSON byte slice
func (m Map) ToJSONBytes() ([]byte, error) {
	return json.Marshal(map[string]interface{}(m))
}

// ShutdownContext returns the app's shutdown context
func (c *Context) ShutdownContext() context.Context {
	if ctx := c.Locals("shutdown_ctx"); ctx != nil {
		if shutdownCtx, ok := ctx.(context.Context); ok {
			return shutdownCtx
		}
	}
	return context.Background()
}

// IsShuttingDown returns true if the app is shutting down
func (c *Context) IsShuttingDown() bool {
	select {
	case <-c.ShutdownContext().Done():
		return true
	default:
		return false
	}
}

// WithTimeout creates a context with timeout that respects shutdown
func (c *Context) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.ShutdownContext(), timeout)
}

// WithDeadline creates a context with deadline that respects shutdown
func (c *Context) WithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(c.ShutdownContext(), deadline)
}

// IsHTTP2 returns true if the request is using HTTP/2
func (c *Context) IsHTTP2() bool {
	if enabled := c.Locals("http2_enabled"); enabled != nil {
		if enabled, ok := enabled.(bool); ok {
			return enabled
		}
	}
	return false
}

// Protocol returns the protocol version (HTTP/1.1 or HTTP/2.0)
func (c *Context) Protocol() string {
	if protocol := c.Locals("protocol"); protocol != nil {
		if protocol, ok := protocol.(string); ok {
			return protocol
		}
	}
	return "HTTP/1.1"
}

// StreamID returns the HTTP/2 stream ID (if applicable)
func (c *Context) StreamID() uint32 {
	if c.IsHTTP2() {
		return uint32(c.RequestCtx.ID())
	}
	return 0
}

// ServerPush pushes a resource to the client (HTTP/2 only)
func (c *Context) ServerPush(path, contentType string) error {
	if !c.IsHTTP2() {
		return fmt.Errorf("server push is only supported in HTTP/2")
	}

	// Add Link header for server push
	linkHeader := fmt.Sprintf("<%s>; rel=preload; as=%s", path, contentType)
	c.SetHeader("Link", linkHeader)

	return nil
}

// PushResources pushes multiple resources (HTTP/2 only)
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

// GetUserValue returns a user value by key (maps to fasthttp's UserValue)
func (c *Context) GetUserValue(key string) interface{} {
	return c.RequestCtx.UserValue(key)
}

// SetUserValue sets a user value (maps to fasthttp's SetUserValue)
func (c *Context) SetUserValue(key string, value interface{}) *Context {
	c.RequestCtx.SetUserValue(key, value)
	return c
}

// GetUserValueString returns a user value as string
func (c *Context) GetUserValueString(key string) string {
	if value := c.GetUserValue(key); value != nil {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

// GetUserValueInt returns a user value as int
func (c *Context) GetUserValueInt(key string) int {
	if value := c.GetUserValue(key); value != nil {
		if i, ok := value.(int); ok {
			return i
		}
	}
	return 0
}

// GetClientIP returns the client IP address (convenience method)
func (c *Context) GetClientIP() string {
	return c.GetUserValueString("client_ip")
}

// GetRealIP returns the real client IP address (convenience method)
func (c *Context) GetRealIP() string {
	return c.GetUserValueString("real_ip")
}

// GetRemoteAddr returns the remote address (convenience method)
func (c *Context) GetRemoteAddr() string {
	return c.GetUserValueString("remote_addr")
}

// MultipartForm returns the parsed multipart form
func (c *Context) MultipartForm() (*MultipartForm, error) {
	return c.MultipartFormWithConfig(DefaultMultipartConfig())
}

// MultipartFormWithConfig returns the parsed multipart form with custom config
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
func (c *Context) SaveUploadedFile(file *MultipartFile, dst string) error {
	return file.Save(dst)
}

// SaveUploadedFileToDir saves an uploaded file to a directory
func (c *Context) SaveUploadedFileToDir(file *MultipartFile, dir string) (string, error) {
	return file.SaveToDir(dir)
}

// SaveUploadedFileWithUniqueFilename saves an uploaded file with a unique filename
func (c *Context) SaveUploadedFileWithUniqueFilename(file *MultipartFile, dir string) (string, error) {
	return file.SaveWithUniqueFilename(dir)
}

// FormValue returns form value (works with both multipart and URL-encoded forms)
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
func (c *Context) IsMultipartForm() bool {
	contentType := string(c.Request().Header.ContentType())
	return strings.Contains(contentType, "multipart/form-data")
}

// GetContentType returns the request content type
func (c *Context) GetContentType() string {
	return string(c.Request().Header.ContentType())
}

// SendFile sends a file as response (wrapper around fasthttp.RequestCtx.SendFile)
func (c *Context) SendFile(filepath string) error {
	// FastHTTP's SendFile doesn't return an error, it logs internally
	c.RequestCtx.SendFile(filepath)
	return nil
}

// ServeFile serves a file with proper headers
func (c *Context) ServeFile(filepath string) error {
	// Use fasthttp.ServeFile which is more flexible
	fasthttp.ServeFile(c.RequestCtx, filepath)
	return nil
}

// ServeFileDownload serves a file as a download with custom filename
func (c *Context) ServeFileDownload(filepath, filename string) error {
	// Set download headers
	c.SetHeader("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.SetHeader("Content-Type", "application/octet-stream")

	// Send the file
	c.RequestCtx.SendFile(filepath)
	return nil
}

// ServeFileInline serves a file for inline display (like images in browser)
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

// StreamFile streams a file with support for range requests (useful for large files/videos)
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

// FileExists checks if a file exists
func (c *Context) FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return !os.IsNotExist(err)
}

// GetFileInfo returns file information
func (c *Context) GetFileInfo(filepath string) (os.FileInfo, error) {
	return os.Stat(filepath)
}

// ResponseWriter returns an io.Writer for the response body
func (c *Context) ResponseWriter() io.Writer {
	return c.RequestCtx.Response.BodyWriter()
}
