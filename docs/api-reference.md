# API Reference

Complete reference documentation for the **Blaze** Go web framework.

## Core Types

### App

The main application instance that manages routing, middleware, and server configuration.

#### Constructor Functions

```go
func New() *App
func NewWithConfig(config Config) *App
```

#### Configuration Methods

```go
func (a *App) SetTLSConfig(config TLSConfig) *App
func (a *App) SetHTTP2Config(config HTTP2Config) *App
func (a *App) EnableAutoTLS(domains ...string) *App
```

#### HTTP Route Methods

```go
func (a *App) GET(path string, handler HandlerFunc, options ...RouteOption) *App
func (a *App) POST(path string, handler HandlerFunc, options ...RouteOption) *App
func (a *App) PUT(path string, handler HandlerFunc, options ...RouteOption) *App
func (a *App) DELETE(path string, handler HandlerFunc, options ...RouteOption) *App
func (a *App) PATCH(path string, handler HandlerFunc, options ...RouteOption) *App
func (a *App) OPTIONS(path string, handler HandlerFunc, options ...RouteOption) *App
func (a *App) HEAD(path string, handler HandlerFunc, options ...RouteOption) *App
func (a *App) CONNECT(path string, handler HandlerFunc, options ...RouteOption) *App
func (a *App) TRACE(path string, handler HandlerFunc, options ...RouteOption) *App
func (a *App) ANY(path string, handler HandlerFunc, options ...RouteOption) *App
func (a *App) Match(methods []string, path string, handler HandlerFunc, options ...RouteOption) *App
```

#### WebSocket Methods

```go
func (a *App) WebSocket(path string, handler WebSocketHandler, options ...RouteOption) *App
func (a *App) WebSocketWithConfig(path string, handler WebSocketHandler, config WebSocketConfig, options ...RouteOption) *App
```

#### Static File Methods

```go
func (a *App) Static(prefix, root string) *App
func (a *App) StaticFS(prefix string, config StaticConfig) *App
func (a *App) File(path, filepath string) *App
```

#### Middleware Methods

```go
func (a *App) Use(middleware MiddlewareFunc) *App
func (a *App) Group(prefix string) *Group
func (a *App) UseErrorHandler(config ErrorHandlerConfig) *App
```

#### State Management

```go
func (a *App) SetState(key string, value interface{}) *App
func (a *App) GetState(key string) (interface{}, bool)
func (a *App) MustGetState(key string) interface{}
func (a *App) DeleteState(key string) *App
func (a *App) ClearState() *App
func (a *App) GetAllState() map[string]interface{}
```

#### Server Lifecycle

```go
func (a *App) ListenAndServe() error
func (a *App) ListenAndServeGraceful(signals ...os.Signal) error
func (a *App) Shutdown(ctx context.Context) error
func (a *App) IsShuttingDown() bool
```

#### Server Information

```go
func (a *App) GetServerInfo() ServerInfo
func (a *App) GetShutdownContext() context.Context
func (a *App) RegisterGracefulTask(task func(ctx context.Context) error)
```

## Configuration Types

### Config

Application configuration structure.

```go
type Config struct {
    Host                string        // Server host (default: "127.0.0.1")
    Port                int           // HTTP port (default: 8080)
    TLSPort             int           // HTTPS port (default: 8443)
    ReadTimeout         time.Duration // Read timeout (default: 10s)
    WriteTimeout        time.Duration // Write timeout (default: 10s)
    MaxRequestBodySize  int           // Max request body size (default: 4MB)
    Concurrency         int           // Max concurrent connections (default: 256*1024)
    EnableHTTP2         bool          // Enable HTTP/2 support
    EnableTLS           bool          // Enable TLS/HTTPS
    RedirectHTTPToTLS   bool          // Redirect HTTP to HTTPS
    Development         bool          // Development mode
}
```

#### Config Constructor Functions

```go
func DefaultConfig() Config
func ProductionConfig() Config
func DevelopmentConfig() Config
```

### TLSConfig

TLS/SSL configuration for HTTPS support.

```go
type TLSConfig struct {
    CertFile                string           // Certificate file path
    KeyFile                 string           // Private key file path
    AutoTLS                 bool             // Auto-generate self-signed certificates
    TLSCacheDir             string           // TLS certificate cache directory
    Domains                 []string         // Domains for certificate
    Organization            string           // Certificate organization
    MinVersion              uint16           // Minimum TLS version
    MaxVersion              uint16           // Maximum TLS version
    CipherSuites            []uint16         // Allowed cipher suites
    ClientAuth              tls.ClientAuthType // Client authentication type
    ClientCAs               *x509.CertPool   // Client CA pool
    NextProtos              []string         // ALPN protocols
    CertValidityDuration    time.Duration    // Certificate validity duration
    OCSPStapling            bool             // OCSP stapling
    SessionTicketsDisabled  bool             // Disable session tickets
    CurvePreferences        []tls.CurveID    // Preferred curves
    Renegotiation           tls.RenegotiationSupport // Renegotiation support
    InsecureSkipVerify      bool             // Skip certificate verification (dev only)
}
```

#### TLS Constructor Functions

```go
func DefaultTLSConfig() TLSConfig
func DevelopmentTLSConfig() TLSConfig
```

#### TLS Methods

```go
func (tc *TLSConfig) BuildTLSConfig() (*tls.Config, error)
func (tc *TLSConfig) ConfigureFastHTTPTLS(server *fasthttp.Server) error
func (tc *TLSConfig) GetCertificateInfo() (*CertificateInfo, error)
func (tc *TLSConfig) GetTLSHealthCheck() TLSHealthCheck
```

### HTTP2Config

HTTP/2 protocol configuration.

```go
type HTTP2Config struct {
    Enabled                      bool           // Enable HTTP/2
    H2C                         bool           // HTTP/2 cleartext (development)
    MaxConcurrentStreams        uint32         // Max concurrent streams
    MaxUploadBufferPerStream    int32          // Max upload buffer per stream
    MaxUploadBufferPerConnection int32         // Max upload buffer per connection
    EnablePush                  bool           // Enable server push
    IdleTimeout                 time.Duration  // Idle timeout
    ReadTimeout                 time.Duration  // Read timeout
    WriteTimeout                time.Duration  // Write timeout
    MaxDecoderHeaderTableSize   uint32         // Max decoder header table size
    MaxEncoderHeaderTableSize   uint32         // Max encoder header table size
    MaxReadFrameSize            uint32         // Max read frame size
    PermitProhibitedCipherSuites bool          // Permit prohibited cipher suites
}
```

#### HTTP2 Constructor Functions

```go
func DefaultHTTP2Config() HTTP2Config
func DevelopmentHTTP2Config() HTTP2Config
```

## Context API

### Context

Request context providing access to request/response data and helper methods.

#### Parameter Methods

```go
func (c *Context) Param(key string) string
func (c *Context) ParamInt(key string) (int, error)
func (c *Context) ParamIntDefault(key string, defaultValue int) int
func (c *Context) SetParam(key, value string)
```

#### Query Parameter Methods

```go
func (c *Context) Query(key string) string
func (c *Context) QueryDefault(key, defaultValue string) string
func (c *Context) QueryInt(key string) (int, error)
func (c *Context) QueryIntDefault(key string, defaultValue int) int
func (c *Context) QueryArgs() *fasthttp.Args
```

#### Header Methods

```go
func (c *Context) Header(key string) string
func (c *Context) SetHeader(key, value string) *Context
func (c *Context) GetContentType() string
```

#### Request Methods

```go
func (c *Context) Method() string
func (c *Context) Path() string
func (c *Context) Body() []byte
func (c *Context) PostBody() []byte
func (c *Context) BodyString() string
func (c *Context) Request() *fasthttp.Request
func (c *Context) URI() *fasthttp.URI
```

#### Response Methods

```go
func (c *Context) Response() *fasthttp.Response
func (c *Context) Status(status int) *Context
func (c *Context) SetContentType(contentType string)
func (c *Context) WriteString(s string) (int, error)
```

#### JSON Response Methods

```go
func (c *Context) JSON(data interface{}) error
func (c *Context) JSONStatus(status int, data interface{}) error
```

#### Text Response Methods

```go
func (c *Context) Text(text string) error
func (c *Context) TextStatus(status int, text string) error
```

#### HTML Response Methods

```go
func (c *Context) HTML(html string) error
func (c *Context) HTMLStatus(status int, html string) error
```

#### Redirect Methods

```go
func (c *Context) Redirect(url string, status ...int)
```

#### Cookie Methods

```go
func (c *Context) Cookie(name string) string
func (c *Context) SetCookie(name, value string, expires ...time.Time) *Context
```

#### Binding Methods

```go
func (c *Context) Bind(v interface{}) error
func (c *Context) BindJSON(v interface{}) error
func (c *Context) BindMultipartForm(v interface{}) error
```

#### Body Validation Methods

```go
func (c *Context) ValidateBodySize(maxSize int64) error
func (c *Context) GetBodySize() int64
func (c *Context) GetContentLength() int
```

#### Local Storage Methods

```go
func (c *Context) Locals(key string) interface{}
func (c *Context) SetLocals(key string, value interface{}) *Context
func (c *Context) GetUserValueString(key string) string
```

#### Client Information Methods

```go
func (c *Context) IP() string
func (c *Context) RemoteIP() net.IP
func (c *Context) UserAgent() string
func (c *Context) GetClientIP() string
func (c *Context) GetRealIP() string
func (c *Context) GetRemoteAddr() string
```

#### Context Lifecycle Methods

```go
func (c *Context) ShutdownContext() context.Context
func (c *Context) IsShuttingDown() bool
func (c *Context) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc)
func (c *Context) WithDeadline(deadline time.Time) (context.Context, context.CancelFunc)
```

#### HTTP/2 Methods

```go
func (c *Context) IsHTTP2() bool
func (c *Context) Protocol() string
func (c *Context) StreamID() uint32
func (c *Context) ServerPush(path, contentType string) error
func (c *Context) PushResources(resources map[string]string) error
```

#### File Handling Methods

```go
func (c *Context) SendFile(filepath string) error
func (c *Context) ServeFile(filepath string) error
func (c *Context) ServeFileDownload(filepath, filename string) error
func (c *Context) ServeFileInline(filepath string) error
func (c *Context) StreamFile(filepath string) error
func (c *Context) FileExists(filepath string) bool
func (c *Context) GetFileInfo(filepath string) (os.FileInfo, error)
```

#### Multipart Form Methods

```go
func (c *Context) MultipartForm() (*MultipartForm, error)
func (c *Context) MultipartFormWithConfig(config MultipartConfig) (*MultipartForm, error)
func (c *Context) FormFile(name string) (*MultipartFile, error)
func (c *Context) FormFiles(name string) ([]*MultipartFile, error)
func (c *Context) FormValue(name string) string
func (c *Context) FormValues(name string) []string
func (c *Context) IsMultipartForm() bool
```

#### File Upload Methods

```go
func (c *Context) SaveUploadedFile(file *MultipartFile, dst string) error
func (c *Context) SaveUploadedFileToDir(file *MultipartFile, dir string) (string, error)
func (c *Context) SaveUploadedFileWithUniqueFilename(file *MultipartFile, dir string) (string, error)
```

## Router API

### Router

Advanced radix tree-based router with constraints and middleware support.

```go
type Router struct {
    root    *routeNode
    routes  map[string]*Route
    config  RouterConfig
}
```

#### Constructor Functions

```go
func NewRouter(config ...RouterConfig) *Router
```

#### Route Registration

```go
func (r *Router) AddRoute(method, pattern string, handler HandlerFunc, options ...RouteOption) *Route
```

#### Route Finding

```go
func (r *Router) FindRoute(method, path string) (*Route, map[string]string, bool)
```

### RouterConfig

Router configuration options.

```go
type RouterConfig struct {
    CaseSensitive          bool // Case sensitive routing
    StrictSlash           bool // Strict slash handling
    RedirectSlash         bool // Redirect trailing slashes
    UseEscapedPath        bool // Use escaped paths
    HandleMethodNotAllowed bool // Handle method not allowed
    HandleOPTIONS         bool // Handle OPTIONS requests
}
```

### Route Options

Route configuration options for advanced routing features.

```go
func WithName(name string) RouteOption
func WithMiddleware(middleware ...MiddlewareFunc) RouteOption
func WithConstraint(param string, constraint RouteConstraint) RouteOption
func WithIntConstraint(param string) RouteOption
func WithUUIDConstraint(param string) RouteOption
func WithRegexConstraint(param string, pattern string) RouteOption
```

## Middleware API

### Core Middleware

```go
func Logger() MiddlewareFunc
func Recovery() MiddlewareFunc
func Auth(tokenValidator func(string) bool) MiddlewareFunc
func ShutdownAware() MiddlewareFunc
func GracefulTimeout(timeout time.Duration) MiddlewareFunc
```

### Logger Middleware

```go
func LoggerMiddleware() MiddlewareFunc
func LoggerMiddlewareWithConfig(config LoggerMiddlewareConfig) MiddlewareFunc
```

**LoggerMiddlewareConfig:**
```go
type LoggerMiddlewareConfig struct {
    Logger              *Logger
    SkipPaths           []string
    LogRequestBody      bool
    LogResponseBody     bool
    LogQueryParams      bool
    LogHeaders          bool
    ExcludeHeaders      []string
    CustomFields        func(*Context) map[string]interface{}
    SlowRequestThreshold time.Duration
}

func DefaultLoggerMiddlewareConfig() LoggerMiddlewareConfig
```

### Error Handling Middleware

```go
func ErrorHandlerMiddleware(config ErrorHandlerConfig) MiddlewareFunc
func RecoveryMiddleware(config ErrorHandlerConfig) MiddlewareFunc
func NotFoundHandler() HandlerFunc
func MethodNotAllowedHandler() HandlerFunc
```

**ErrorHandlerConfig:**
```go
type ErrorHandlerConfig struct {
    EnableStackTrace    bool
    IncludeStackTrace   bool
    CustomErrorHandler  func(*Context, error) error
    Logger              *Logger
}

func DefaultErrorHandlerConfig() ErrorHandlerConfig
func DevelopmentErrorHandlerConfig() ErrorHandlerConfig
```

### CORS Middleware

```
func CORS(opts ...CORSOptions) MiddlewareFunc
```

**CORSOptions:**
```go
type CORSOptions struct {
    AllowOrigins     []string
    AllowMethods     []string
    AllowHeaders     []string
    ExposeHeaders    []string
    AllowCredentials bool
    MaxAge           int
}
```

### CSRF Protection

```go
func CSRF(opts CSRFOptions) MiddlewareFunc
func CSRFToken(c *Context) string
func CSRFTokenHTML(c *Context) string
func CSRFTokenHeader(c *Context) string
func CSRFMeta(c *Context) string
```

**CSRFOptions:**
```go
type CSRFOptions struct {
    Secret            []byte
    TokenLookup       string
    ContextKey        string
    CookieName        string
    CookiePath        string
    CookieDomain      string
    CookieSecure      bool
    CookieHTTPOnly    bool
    CookieSameSite    string
    CookieMaxAge      int
    Expiration        time.Duration
    TokenLength       int
    Skipper           func(*Context) bool
    ErrorHandler      func(*Context, error) error
    TrustedOrigins    []string
    CheckReferer      bool
    SingleUse         bool
}

func DefaultCSRFOptions() CSRFOptions
func ProductionCSRFOptions(secret []byte) CSRFOptions
```

### Cache Middleware

```go
func Cache(opts CacheOptions) MiddlewareFunc
func CacheResponse(ttl time.Duration, opts ...CacheOptions) MiddlewareFunc
func CacheStatic(opts ...CacheOptions) MiddlewareFunc
func CacheAPI(ttl time.Duration) MiddlewareFunc
func CacheStatus(c *Context) error
```

**CacheOptions:**
```go
type CacheOptions struct {
    Store                   CacheStore
    DefaultTTL              time.Duration
    MaxAge                  time.Duration
    MaxSize                 int64
    MaxEntries              int
    Algorithm               EvictionAlgorithm
    Skipper                 func(*Context) bool
    KeyGenerator            func(*Context) string
    ShouldCache             func(*Context) bool
    VaryHeaders             []string
    Public                  bool
    Private                 bool
    NoCache                 bool
    NoStore                 bool
    MustRevalidate          bool
    ProxyRevalidate         bool
    Immutable               bool
    EnableCompression       bool
    CompressionLevel        int
    CleanupInterval         time.Duration
    EnableBackgroundCleanup bool
    WarmupURLs              []string
    EnableHeaders           bool
    HeaderPrefix            string
}

func DefaultCacheOptions() CacheOptions
func ProductionCacheOptions() CacheOptions
```

**Cache Store:**
```go
type MemoryStore struct {
    entries    map[string]CacheEntry
    maxSize    int64
    maxEntries int
    algorithm  EvictionAlgorithm
}

func NewMemoryStore(maxSize int64, maxEntries int, algorithm EvictionAlgorithm) *MemoryStore
func (c *MemoryStore) Get(key string) (CacheEntry, bool)
func (c *MemoryStore) Set(key string, entry CacheEntry, ttl time.Duration) bool
func (c *MemoryStore) Delete(key string) bool
func (c *MemoryStore) Clear() int
func (c *MemoryStore) Size() int64
func (c *MemoryStore) Keys() []string
func (c *MemoryStore) Stats() CacheStats
func (c *MemoryStore) Cleanup() int
```

### Compression Middleware

```go
func Compress() MiddlewareFunc
func CompressWithConfig(config CompressionConfig) MiddlewareFunc
func CompressWithLevel(level CompressionLevel) MiddlewareFunc
func CompressMinLength(minLength int) MiddlewareFunc
func CompressGzipOnly() MiddlewareFunc
func CompressTypes(contentTypes ...string) MiddlewareFunc
```

**CompressionConfig:**
```go
type CompressionConfig struct {
    Level                 CompressionLevel
    MinLength             int
    IncludeContentTypes   []string
    ExcludeContentTypes   []string
    EnableGzip            bool
    EnableDeflate         bool
    EnableBrotli          bool
    ExcludePaths          []string
    ExcludeExtensions     []string
    EnableForHTTPS        bool
}

func DefaultCompressionConfig() CompressionConfig
```

**CompressionLevel:**
```go
type CompressionLevel int

const (
    CompressionLevelDefault  CompressionLevel = -1
    CompressionLevelNone     CompressionLevel = 0
    CompressionLevelBest     CompressionLevel = 9
    CompressionLevelFastest  CompressionLevel = 1
)
```

### Body Limit Middleware

```go
func BodyLimit(maxSize int64) MiddlewareFunc
func BodyLimitWithConfig(config BodyLimitConfig) MiddlewareFunc
func BodyLimitBytes(bytes int64) MiddlewareFunc
func BodyLimitKB(kb int) MiddlewareFunc
func BodyLimitMB(mb int) MiddlewareFunc
func BodyLimitGB(gb int) MiddlewareFunc
func BodyLimitForRoute(maxSize int64, paths ...string) MiddlewareFunc
func BodyLimitByContentType(limits map[string]int64) MiddlewareFunc
```

**BodyLimitConfig:**
```go
type BodyLimitConfig struct {
    MaxSize          int64
    ErrorMessage     string
    SkipPaths        []string
    SkipContentTypes []string
}

func DefaultBodyLimitConfig() BodyLimitConfig
```

### Rate Limiting

```go
func RateLimitMiddleware(opts RateLimitOptions) MiddlewareFunc
```

**RateLimitOptions:**
```go
type RateLimitOptions struct {
    MaxRequests      int
    Window           time.Duration
    KeyGenerator     func(*Context) string
    Handler          func(*Context) error
    SkipSuccessful   bool
    SkipFailed       bool
}
```

### Request ID

```go
func RequestIDMiddleware() MiddlewareFunc
func GetRequestID(c *Context) string
```

### HTTP/2 Specific Middleware

```go
func HTTP2Middleware() MiddlewareFunc
func HTTP2Info() MiddlewareFunc
func HTTP2Security() MiddlewareFunc
func CompressHTTP2(level int) MiddlewareFunc
func StreamInfo() MiddlewareFunc
func HTTP2Metrics() MiddlewareFunc
```

### Multipart Middleware

```go
func MultipartMiddleware(config MultipartConfig) MiddlewareFunc
```

### Validation Middleware

```go
func ValidationMiddleware() MiddlewareFunc
func ValidateStruct(v interface{}) error
```

## WebSocket API

### WebSocketUpgrader

WebSocket connection upgrader.

```go
type WebSocketUpgrader struct {
    upgrader        websocket.FastHTTPUpgrader
    readTimeout     time.Duration
    writeTimeout    time.Duration
    pingInterval    time.Duration
    pongTimeout     time.Duration
    maxMessageSize  int64
}
```

#### Constructor Functions

```go
func NewWebSocketUpgrader(config ...WebSocketConfig) *WebSocketUpgrader
```

#### Upgrade Method

```go
func (wu *WebSocketUpgrader) Upgrade(c *Context, handler WebSocketHandler) error
```

### WebSocketConfig

```go
type WebSocketConfig struct {
    ReadBufferSize   int
    WriteBufferSize  int
    CheckOrigin      func(ctx *fasthttp.RequestCtx) bool
    ReadTimeout      time.Duration
    WriteTimeout     time.Duration
    PingInterval     time.Duration
    PongTimeout      time.Duration
    MaxMessageSize   int64
    CompressionLevel int
}

func DefaultWebSocketConfig() WebSocketConfig
```

### WebSocketConnection

Active WebSocket connection wrapper.

#### Message Methods

```go
func (ws *WebSocketConnection) ReadMessage() (messageType int, data []byte, err error)
func (ws *WebSocketConnection) WriteMessage(messageType int, data []byte) error
func (ws *WebSocketConnection) WriteText(data string) error
func (ws *WebSocketConnection) WriteBinary(data []byte) error
func (ws *WebSocketConnection) WriteJSON(data interface{}) error
func (ws *WebSocketConnection) ReadJSON(v interface{}) error
```

#### Connection Control

```go
func (ws *WebSocketConnection) Close() error
func (ws *WebSocketConnection) IsClosed() bool
func (ws *WebSocketConnection) Ping(data []byte) error
func (ws *WebSocketConnection) Pong(data []byte) error
```

#### Connection Info

```go
func (ws *WebSocketConnection) Context() *Context
func (ws *WebSocketConnection) RemoteAddr() string
func (ws *WebSocketConnection) LocalAddr() string
func (ws *WebSocketConnection) UserAgent() string
func (ws *WebSocketConnection) Header(key string) string
```

#### Local Storage

```go
func (ws *WebSocketConnection) SetLocal(key string, value interface{})
func (ws *WebSocketConnection) GetLocal(key string) interface{}
```

#### Async Operations

```go
func (ws *WebSocketConnection) WriteAsync(data []byte)
```

### WebSocketHub

Multi-connection WebSocket hub for broadcasting.

```go
type WebSocketHub struct {
    clients    map[*WebSocketConnection]bool
    broadcast  chan []byte
    register   chan *WebSocketConnection
    unregister chan *WebSocketConnection
}
```

#### Constructor Functions

```go
func NewWebSocketHub() *WebSocketHub
```

#### Hub Operations

```go
func (h *WebSocketHub) Run()
func (h *WebSocketHub) Register(client *WebSocketConnection)
func (h *WebSocketHub) Unregister(client *WebSocketConnection)
func (h *WebSocketHub) Broadcast(message []byte)
func (h *WebSocketHub) GetClientCount() int
func (h *WebSocketHub) GetClients() []*WebSocketConnection
```

## File Upload API

### MultipartForm

Parsed multipart form data.

```go
type MultipartForm struct {
    Value map[string][]string           // Form values
    File  map[string][]*MultipartFile   // Uploaded files
}
```

#### Methods

```go
func (mf *MultipartForm) GetValue(key string) string
func (mf *MultipartForm) GetValues(key string) []string
func (mf *MultipartForm) GetFile(key string) *MultipartFile
func (mf *MultipartForm) GetFiles(key string) []*MultipartFile
```

### MultipartFile

Uploaded file representation.

```go
type MultipartFile struct {
    Filename     string              // Original filename
    Header       textproto.MIMEHeader // File headers
    Size         int64               // File size
    ContentType  string              // Content type
    Data         []byte              // File data (if in memory)
    TempFilePath string              // Temporary file path
    FileHeader   *multipart.FileHeader // Original file header
}
```

#### File Operations

```go
func (mf *MultipartFile) Save(dst string) error
func (mf *MultipartFile) SaveToDir(dir string) (string, error)
func (mf *MultipartFile) SaveWithUniqueFilename(dir string) (string, error)
```

### MultipartConfig

Multipart form parsing configuration.

```go
type MultipartConfig struct {
    MaxMemory   int64  // Maximum memory for file storage
    MaxFiles    int    // Maximum number of files
    TempDir     string // Temporary directory
    KeepInMemory bool  // Keep files in memory
}
```

## Static File Serving

### StaticConfig

```go
type StaticConfig struct {
    Root            string
    Index           string
    Browse          bool
    Compress        bool
    ByteRange       bool
    CacheDuration   time.Duration
    NotFoundHandler HandlerFunc
    Modify          func(*Context) error
    GenerateETag    bool
    Exclude         []string
    MIMETypes       map[string]string
}

func DefaultStaticConfig(root string) StaticConfig
```

### Static Functions

```go
func Static(root string) HandlerFunc
func StaticFS(config StaticConfig) HandlerFunc
```

## Error Handling API

### Error Types

```go
type HTTPError struct {
    Code     int
    Message  string
    Internal error
    Stack    []string
}

func (e *HTTPError) Error() string
func (e *HTTPError) Unwrap() error
func (e *HTTPError) WithInternal(err error) *HTTPError
func (e *HTTPError) WithStack(skip int) *HTTPError
```

### Error Constructors

```go
func ErrBadRequest(message string) *HTTPError
func ErrUnauthorized(message string) *HTTPError
func ErrForbidden(message string) *HTTPError
func ErrNotFound(message string) *HTTPError
func ErrMethodNotAllowed(message string) *HTTPError
func ErrConflict(message string) *HTTPError
func ErrInternalServer(message string) *HTTPError
func ErrServiceUnavailable(message string) *HTTPError
```

### Error Response Functions

```go
func JSONError(ctx *fasthttp.RequestCtx, statusCode int, msg string) error
func NotFoundError(ctx *fasthttp.RequestCtx, msg string) error
func BadRequestError(ctx *fasthttp.RequestCtx, msg string) error
func UnauthorizedError(ctx *fasthttp.RequestCtx, msg string) error
func ForbiddenError(ctx *fasthttp.RequestCtx, msg string) error
```

### Helper Response Functions

```go
func OK(data interface{}) Map
func Error(message string) Map
func Created(data interface{}) Map
func NoContent(c *Context) error
func BadRequest(c *Context, message string) error
func Unauthorized(c *Context, message string) error
func Forbidden(c *Context, message string) error
func NotFound(c *Context, message string) error
func InternalServerError(c *Context, message string) error
func Redirect(c *Context, url string, statusCode int) error
```

### Paginated Responses

```go
type PaginatedResponse struct {
    Data       interface{} `json:"data"`
    Total      int         `json:"total"`
    Page       int         `json:"page"`
    PerPage    int         `json:"per_page"`
    TotalPages int         `json:"total_pages"`
    HasNext    bool        `json:"has_next"`
    HasPrev    bool        `json:"has_prev"`
}

func Paginate(data interface{}, total, page, perPage int) PaginatedResponse
```

### Health Check

```go
type HealthCheck struct {
    Status    string    `json:"status"`
    Timestamp time.Time `json:"timestamp"`
    Version   string    `json:"version"`
    Uptime    string    `json:"uptime"`
}

func Health(version, uptime string) HealthCheck
```

## Logger API

### Logger

```go
type Logger struct {
    level      LogLevel
    output     io.Writer
    enableJSON bool
    fields     map[string]interface{}
}
```

### Logger Methods

```go
func (l *Logger) Debug(msg string, fields ...interface{})
func (l *Logger) Info(msg string, fields ...interface{})
func (l *Logger) Warn(msg string, fields ...interface{})
func (l *Logger) Error(msg string, fields ...interface{})
func (l *Logger) Fatal(msg string, fields ...interface{})
func (l *Logger) WithFields(fields map[string]interface{}) *Logger
func (l *Logger) SetLevel(level LogLevel)
func (l *Logger) SetOutput(output io.Writer)
```

### Logger Functions

```go
func NewLogger(level LogLevel, output io.Writer) *Logger
func GetDefaultLogger() *Logger
```

## Utility Types

### Map

Convenience type for JSON responses.

```go
type Map map[string]interface{}

func (m Map) MarshalJSON() ([]byte, error)
func (m Map) ToJSON() (string, error)
func (m Map) ToJSONBytes() ([]byte, error)
```

### ServerInfo

Server configuration and status information.

```go
type ServerInfo struct {
    Host        string             `json:"host"`
    Port        int                `json:"port"`
    TLSPort     int                `json:"tls_port,omitempty"`
    EnableTLS   bool               `json:"enable_tls"`
    EnableHTTP2 bool               `json:"enable_http2"`
    Development bool               `json:"development"`
    TLS         TLSHealthCheck     `json:"tls,omitempty"`
    HTTP2       HTTP2HealthCheck   `json:"http2,omitempty"`
}
```

### Group

Route group with shared prefix and middleware.

```go
type Group struct {
    prefix     string
    middleware []MiddlewareFunc
    app        *App
}

func (g *Group) Use(middleware MiddlewareFunc) *Group
func (g *Group) GET(path string, handler HandlerFunc, options ...RouteOption) *Group
func (g *Group) POST(path string, handler HandlerFunc, options ...RouteOption) *Group
func (g *Group) PUT(path string, handler HandlerFunc, options ...RouteOption) *Group
func (g *Group) DELETE(path string, handler HandlerFunc, options ...RouteOption) *Group
func (g *Group) PATCH(path string, handler HandlerFunc, options ...RouteOption) *Group
func (g *Group) OPTIONS(path string, handler HandlerFunc, options ...RouteOption) *Group
func (g *Group) HEAD(path string, handler HandlerFunc, options ...RouteOption) *Group
func (g *Group) Group(prefix string) *Group
```

## Function Signatures

### Handler Types

```go
type HandlerFunc func(*Context) error
type MiddlewareFunc func(HandlerFunc) HandlerFunc
type WebSocketHandler func(*WebSocketConnection)
```

### Route Types

```go
type RouteOption func(*Route)
type RouteConstraint struct {
    Name    string
    Pattern *regexp.Regexp
    Type    ConstraintType
}

type ConstraintType string

const (
    IntConstraint   ConstraintType = "int"
    UUIDConstraint  ConstraintType = "uuid"
    AlphaConstraint ConstraintType = "alpha"
    RegexConstraint ConstraintType = "regex"
)
```

---

This API reference provides comprehensive documentation for all major components, types, and functions in the Blaze Go web framework.