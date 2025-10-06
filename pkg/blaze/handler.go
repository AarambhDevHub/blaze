package blaze

// HandlerFunc defines the function signature for request handlers
// This is the core function type used throughout the framework for handling HTTP requests
//
// Handler Responsibilities:
//   - Process incoming HTTP requests
//   - Access request data (params, query, body, headers)
//   - Generate HTTP responses (JSON, HTML, files, etc.)
//   - Return errors for centralized error handling
//   - Support middleware wrapping and chaining
//
// Handler Best Practices:
//   - Return errors instead of setting error responses directly
//   - Use Context methods for consistent response formatting
//   - Keep handlers focused on single responsibility
//   - Extract reusable logic into middleware
//   - Use dependency injection for testability
//
// Context Lifetime:
//   - Context is valid only for the duration of the request
//   - Do not store Context in structs or pass to goroutines
//   - Use Context.ShutdownContext() for graceful shutdown awareness
//
// Error Handling:
//   - Return HTTPError for structured error responses
//   - Framework automatically converts errors to HTTP responses
//   - Use error middleware for centralized error logging
//
// Example - Simple Handler:
//
//	func helloHandler(c *blaze.Context) error {
//	    return c.JSON(blaze.Map{"message": "Hello World"})
//	}
//
// Example - Handler with Route Parameters:
//
//	func getUserHandler(c *blaze.Context) error {
//	    userID, err := c.ParamInt("id")
//	    if err != nil {
//	        return blaze.ErrBadRequest("Invalid user ID")
//	    }
//
//	    user, err := db.GetUser(userID)
//	    if err != nil {
//	        return blaze.ErrNotFound("User not found")
//	    }
//
//	    return c.JSON(user)
//	}
//
// Example - Handler with Request Body:
//
//	func createUserHandler(c *blaze.Context) error {
//	    var req CreateUserRequest
//	    if err := c.BindJSON(&req); err != nil {
//	        return blaze.ErrBadRequest("Invalid JSON")
//	    }
//
//	    if err := c.Validate(&req); err != nil {
//	        return blaze.ErrValidation("Validation failed", err)
//	    }
//
//	    user, err := service.CreateUser(req)
//	    if err != nil {
//	        return blaze.ErrInternalServer("Failed to create user").WithInternal(err)
//	    }
//
//	    return c.Status(201).JSON(user)
//	}
//
// Example - Handler with Database Access:
//
//	func listUsersHandler(c *blaze.Context) error {
//	    page := c.QueryIntDefault("page", 1)
//	    limit := c.QueryIntDefault("limit", 20)
//
//	    users, total, err := db.ListUsers(page, limit)
//	    if err != nil {
//	        return blaze.ErrDatabase("Failed to fetch users", err)
//	    }
//
//	    return c.JSON(blaze.Map{
//	        "users": users,
//	        "total": total,
//	        "page": page,
//	        "limit": limit,
//	    })
//	}
//
// Example - Handler with File Upload:
//
//	func uploadHandler(c *blaze.Context) error {
//	    file, err := c.FormFile("file")
//	    if err != nil {
//	        return blaze.ErrBadRequest("No file uploaded")
//	    }
//
//	    path, err := file.SaveToDir("/uploads")
//	    if err != nil {
//	        return blaze.ErrInternalServer("Failed to save file")
//	    }
//
//	    return c.JSON(blaze.Map{
//	        "filename": file.Filename,
//	        "path": path,
//	        "size": file.Size,
//	    })
//	}
//
// Example - Async Handler with Graceful Shutdown:
//
//	func longRunningHandler(c *blaze.Context) error {
//	    ctx, cancel := c.WithTimeout(30 * time.Second)
//	    defer cancel()
//
//	    select {
//	    case result := <-processData(ctx):
//	        return c.JSON(result)
//	    case <-ctx.Done():
//	        if c.IsShuttingDown() {
//	            return blaze.ErrServiceUnavailable("Server shutting down")
//	        }
//	        return blaze.ErrGatewayTimeout("Request timeout")
//	    }
//	}
type HandlerFunc func(*Context) error

// MiddlewareFunc defines the function signature for middleware
// Middleware wraps handlers to add cross-cutting functionality
//
// Middleware Responsibilities:
//   - Pre-processing requests before handlers execute
//   - Post-processing responses after handlers complete
//   - Adding functionality like logging, auth, CORS, etc.
//   - Error handling and recovery
//   - Request/response transformation
//
// Middleware Execution Order:
//   - Global middleware executes first (registered with app.Use())
//   - Route-specific middleware executes next
//   - Handler executes last
//   - Response flows back through middleware in reverse order
//
// Middleware Best Practices:
//   - Call next(c) to continue the chain
//   - Return errors for centralized error handling
//   - Use c.SetLocals() to pass data to subsequent handlers
//   - Keep middleware focused on single concern
//   - Consider performance impact of middleware
//   - Order matters - register middleware carefully
//
// Common Middleware Patterns:
//   - Pre-processing: Validate, transform, or enrich requests
//   - Post-processing: Modify responses, add headers
//   - Short-circuiting: Return early without calling next()
//   - Error handling: Catch and transform errors
//   - State management: Set context variables
//
// Example - Simple Middleware:
//
//	func RequestIDMiddleware() blaze.MiddlewareFunc {
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) error {
//	            requestID := generateRequestID()
//	            c.SetLocals("request_id", requestID)
//	            c.SetHeader("X-Request-ID", requestID)
//	            return next(c)
//	        }
//	    }
//	}
//
// Example - Authentication Middleware:
//
//	func AuthMiddleware() blaze.MiddlewareFunc {
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) error {
//	            token := c.Header("Authorization")
//	            if token == "" {
//	                return blaze.ErrUnauthorized("Missing authorization token")
//	            }
//
//	            user, err := validateToken(token)
//	            if err != nil {
//	                return blaze.ErrUnauthorized("Invalid token")
//	            }
//
//	            c.SetLocals("user", user)
//	            return next(c)
//	        }
//	    }
//	}
//
// Example - Logging Middleware:
//
//	func LoggerMiddleware() blaze.MiddlewareFunc {
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) error {
//	            start := time.Now()
//
//	            // Call next handler
//	            err := next(c)
//
//	            // Log after handler completes
//	            duration := time.Since(start)
//	            log.Printf("%s %s - %d - %v",
//	                c.Method(),
//	                c.Path(),
//	                c.Response().StatusCode(),
//	                duration,
//	            )
//
//	            return err
//	        }
//	    }
//	}
//
// Example - Rate Limiting Middleware:
//
//	func RateLimitMiddleware(limit int) blaze.MiddlewareFunc {
//	    limiter := rate.NewLimiter(rate.Limit(limit), limit)
//
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) error {
//	            if !limiter.Allow() {
//	                return blaze.ErrTooManyRequests("Rate limit exceeded")
//	            }
//	            return next(c)
//	        }
//	    }
//	}
//
// Example - CORS Middleware:
//
//	func CORSMiddleware() blaze.MiddlewareFunc {
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) error {
//	            c.SetHeader("Access-Control-Allow-Origin", "*")
//	            c.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
//	            c.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")
//
//	            // Handle preflight requests
//	            if c.Method() == "OPTIONS" {
//	                return c.Status(204).Text("")
//	            }
//
//	            return next(c)
//	        }
//	    }
//	}
//
// Example - Error Recovery Middleware:
//
//	func RecoveryMiddleware() blaze.MiddlewareFunc {
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) (err error) {
//	            defer func() {
//	                if r := recover(); r != nil {
//	                    log.Printf("Panic recovered: %v", r)
//	                    err = blaze.ErrInternalServer("Internal server error")
//	                }
//	            }()
//	            return next(c)
//	        }
//	    }
//	}
//
// Example - Timeout Middleware:
//
//	func TimeoutMiddleware(timeout time.Duration) blaze.MiddlewareFunc {
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) error {
//	            ctx, cancel := c.WithTimeout(timeout)
//	            defer cancel()
//
//	            done := make(chan error, 1)
//	            go func() {
//	                done <- next(c)
//	            }()
//
//	            select {
//	            case err := <-done:
//	                return err
//	            case <-ctx.Done():
//	                return blaze.ErrGatewayTimeout("Request timeout")
//	            }
//	        }
//	    }
//	}
//
// Example - Compression Middleware:
//
//	func CompressionMiddleware() blaze.MiddlewareFunc {
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) error {
//	            err := next(c)
//	            if err != nil {
//	                return err
//	            }
//
//	            // Check if client accepts gzip
//	            if strings.Contains(c.Header("Accept-Encoding"), "gzip") {
//	                body := c.Response().Body()
//	                compressed := compressGzip(body)
//	                c.Response().SetBody(compressed)
//	                c.SetHeader("Content-Encoding", "gzip")
//	            }
//
//	            return nil
//	        }
//	    }
//	}
//
// Example - Validation Middleware:
//
//	func ValidationMiddleware() blaze.MiddlewareFunc {
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) error {
//	            // Validate request before handler
//	            if err := validateRequest(c); err != nil {
//	                return blaze.ErrBadRequest(err.Error())
//	            }
//	            return next(c)
//	        }
//	    }
//	}
//
// Example - Caching Middleware:
//
//	func CacheMiddleware(ttl time.Duration) blaze.MiddlewareFunc {
//	    cache := NewCache()
//
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) error {
//	            // Only cache GET requests
//	            if c.Method() != "GET" {
//	                return next(c)
//	            }
//
//	            key := c.Path()
//	            if cached := cache.Get(key); cached != nil {
//	                return c.JSON(cached)
//	            }
//
//	            err := next(c)
//	            if err == nil {
//	                cache.Set(key, c.Response().Body(), ttl)
//	            }
//
//	            return err
//	        }
//	    }
//	}
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
