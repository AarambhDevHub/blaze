package blaze

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// ErrorCode represents standardized error codes for consistent error handling
// Error codes provide machine-readable identifiers for different error types
// Clients can use these codes to implement error-specific handling logic
//
// Benefits of Error Codes:
//   - Machine-readable error identification
//   - Internationalization support (codes don't change, messages do)
//   - Consistent error handling across APIs
//   - Easy error categorization and monitoring
//
// Naming Convention:
//   - Use UPPER_SNAKE_CASE for consistency
//   - Be specific but not overly granular
//   - Group related errors with common prefixes
type ErrorCode string

const (
	// Client Errors (4xx)

	// ErrCodeBadRequest indicates the request was malformed or invalid
	// HTTP Status: 400
	// Common causes: Invalid JSON, missing required fields, malformed data
	ErrCodeBadRequest ErrorCode = "BAD_REQUEST"

	// ErrCodeUnauthorized indicates authentication is required or failed
	// HTTP Status: 401
	// Common causes: Missing auth token, invalid credentials, expired token
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"

	// ErrCodeForbidden indicates the user lacks permission for the resource
	// HTTP Status: 403
	// Common causes: Insufficient permissions, account suspended, resource ownership
	ErrCodeForbidden ErrorCode = "FORBIDDEN"

	// ErrCodeNotFound indicates the requested resource does not exist
	// HTTP Status: 404
	// Common causes: Invalid ID, deleted resource, wrong endpoint
	ErrCodeNotFound ErrorCode = "NOT_FOUND"

	// ErrCodeMethodNotAllowed indicates the HTTP method is not supported
	// HTTP Status: 405
	// Common causes: Using POST on GET-only endpoint, wrong HTTP verb
	ErrCodeMethodNotAllowed ErrorCode = "METHOD_NOT_ALLOWED"

	// ErrCodeConflict indicates a conflict with the current state
	// HTTP Status: 409
	// Common causes: Duplicate key, version conflict, concurrent modification
	ErrCodeConflict ErrorCode = "CONFLICT"

	// ErrCodeValidation indicates validation errors in request data
	// HTTP Status: 422
	// Common causes: Field validation failures, business rule violations
	ErrCodeValidation ErrorCode = "VALIDATION_ERROR"

	// ErrCodeTooManyRequests indicates rate limit exceeded
	// HTTP Status: 429
	// Common causes: API rate limiting, DDoS protection, abuse prevention
	ErrCodeTooManyRequests ErrorCode = "TOO_MANY_REQUESTS"

	// ErrCodeRequestTimeout indicates the request took too long
	// HTTP Status: 408
	// Common causes: Slow client, network issues, large uploads
	ErrCodeRequestTimeout ErrorCode = "REQUEST_TIMEOUT"

	// ErrCodePayloadTooLarge indicates request body exceeds size limit
	// HTTP Status: 413
	// Common causes: Large file uploads, excessive JSON payload
	ErrCodePayloadTooLarge ErrorCode = "PAYLOAD_TOO_LARGE"

	// Server Errors (5xx)

	// ErrCodeInternalServer indicates an unexpected server error
	// HTTP Status: 500
	// Common causes: Unhandled exceptions, bugs, infrastructure failures
	ErrCodeInternalServer ErrorCode = "INTERNAL_SERVER_ERROR"

	// ErrCodeNotImplemented indicates functionality is not implemented
	// HTTP Status: 501
	// Common causes: Placeholder endpoints, incomplete features
	ErrCodeNotImplemented ErrorCode = "NOT_IMPLEMENTED"

	// ErrCodeServiceUnavailable indicates the service is temporarily unavailable
	// HTTP Status: 503
	// Common causes: Maintenance mode, database down, overload
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"

	// ErrCodeGatewayTimeout indicates upstream service timeout
	// HTTP Status: 504
	// Common causes: Slow external APIs, database timeouts, network issues
	ErrCodeGatewayTimeout ErrorCode = "GATEWAY_TIMEOUT"

	// Custom Business Errors

	// ErrCodeDatabaseError indicates a database operation failed
	// Used for database-specific errors that need special handling
	ErrCodeDatabaseError ErrorCode = "DATABASE_ERROR"

	// ErrCodeExternalAPIError indicates an external API call failed
	// Used when third-party services fail or return errors
	ErrCodeExternalAPIError ErrorCode = "EXTERNAL_API_ERROR"

	// ErrCodeAuthenticationError indicates authentication-specific failures
	// Used for detailed authentication error handling
	ErrCodeAuthenticationError ErrorCode = "AUTHENTICATION_ERROR"

	// ErrCodeAuthorizationError indicates authorization-specific failures
	// Used for detailed authorization error handling
	ErrCodeAuthorizationError ErrorCode = "AUTHORIZATION_ERROR"
)

// HTTPError represents a structured HTTP error with rich context
// Provides comprehensive error information for debugging and client handling
//
// HTTPError vs Standard Error:
//   - HTTPError: Full context, structured data, HTTP-aware
//   - Standard error: Simple string message, no context
//
// Error Propagation:
//   - Use WithInternal to wrap underlying errors
//   - Preserves error chain for debugging
//   - Supports errors.Unwrap for error inspection
//
// Best Practices:
//   - Use specific error codes for different error types
//   - Include actionable error messages for clients
//   - Add metadata for debugging context
//   - Capture stack traces in development
//   - Hide internal errors in production
type HTTPError struct {
	// Code is the machine-readable error code
	// Used by clients for error-specific handling
	Code ErrorCode `json:"code"`

	// Message is the human-readable error message
	// Should be clear, actionable, and user-friendly
	Message string `json:"message"`

	// StatusCode is the HTTP status code to return
	// Should match standard HTTP status code semantics
	StatusCode int `json:"statuscode"`

	// Details contains additional error information
	// Can be validation errors, field-specific messages, etc.
	Details interface{} `json:"details,omitempty"`

	// Metadata contains additional context for debugging
	// Use for request IDs, user IDs, resource IDs, etc.
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Timestamp indicates when the error occurred
	// Useful for correlating errors with logs
	Timestamp time.Time `json:"timestamp"`

	// Path is the request path that caused the error
	// Automatically populated by error middleware
	Path string `json:"path,omitempty"`

	// Method is the HTTP method of the failed request
	// Automatically populated by error middleware
	Method string `json:"method,omitempty"`

	// RequestID is the unique request identifier
	// Used for log correlation and debugging
	RequestID string `json:"requestid,omitempty"`

	// Stack contains the stack trace (development only)
	// Should not be exposed in production
	Stack []StackFrame `json:"stack,omitempty"`

	// Internal is the underlying error (not exposed in JSON)
	// Preserved for server-side logging and debugging
	Internal error `json:"-"`
}

// StackFrame represents a single frame in the stack trace
// Provides file, line, and function information for debugging
//
// Stack Trace Usage:
//   - Captured automatically when WithStack is called
//   - Shows error propagation path through code
//   - Filtered to exclude runtime frames
//   - Only included in development mode
type StackFrame struct {
	// File is the source file path
	File string `json:"file"`

	// Line is the line number in the source file
	Line int `json:"line"`

	// Function is the fully qualified function name
	Function string `json:"function"`
}

// Error implements the error interface for HTTPError
// Returns a formatted error string for logging and debugging
//
// Format:
//   - With internal error: "ERROR_CODE: message: internal error"
//   - Without internal error: "ERROR_CODE: message"
//
// Returns:
//   - string: Formatted error message
func (e *HTTPError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for error wrapping support
// Enables error inspection with errors.Is and errors.As
//
// Example:
//
//	if errors.Is(err, sql.ErrNoRows) {
//	    // Handle specific error
//	}
//
// Returns:
//   - error: The wrapped internal error or nil
func (e *HTTPError) Unwrap() error {
	return e.Internal
}

// WithDetails adds additional details to the error
// Details can be validation errors, field information, or any structured data
//
// Common Use Cases:
//   - Validation errors (field-level messages)
//   - Multiple error messages
//   - Structured error data for clients
//
// Parameters:
//   - details: Additional error information (any type)
//
// Returns:
//   - *HTTPError: Error with details for method chaining
//
// Example:
//
//	err := blaze.ErrValidation("Invalid input").WithDetails(map[string]string{
//	    "email": "Invalid email format",
//	    "age": "Must be at least 18",
//	})
func (e *HTTPError) WithDetails(details interface{}) *HTTPError {
	e.Details = details
	return e
}

// WithMetadata adds metadata to the error for debugging context
// Metadata is included in error responses (be careful with sensitive data)
//
// Common Metadata:
//   - User ID
//   - Resource ID
//   - Correlation ID
//   - Transaction ID
//   - External request ID
//
// Parameters:
//   - key: Metadata key
//   - value: Metadata value (any type)
//
// Returns:
//   - *HTTPError: Error with metadata for method chaining
//
// Example:
//
//	err := blaze.ErrNotFound("User not found").
//	    WithMetadata("user_id", 123).
//	    WithMetadata("lookup_method", "email")
func (e *HTTPError) WithMetadata(key string, value interface{}) *HTTPError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithInternal sets the internal error for debugging
// Internal errors are logged but not exposed to clients
//
// Use Cases:
//   - Wrapping database errors
//   - Capturing external API errors
//   - Preserving original error for logging
//
// Parameters:
//   - err: The underlying error to wrap
//
// Returns:
//   - *HTTPError: Error with internal error for method chaining
//
// Example:
//
//	err := blaze.ErrInternalServer("Database operation failed").
//	    WithInternal(dbErr)
func (e *HTTPError) WithInternal(err error) *HTTPError {
	e.Internal = err
	return e
}

// WithStack captures the current stack trace for debugging
// Stack traces help identify the error source in complex code paths
//
// Performance Note:
//   - Stack capture has overhead, use judiciously
//   - Only enable in development/debugging mode
//   - Automatically filtered to exclude runtime frames
//
// Parameters:
//   - skip: Number of stack frames to skip (0 for immediate caller)
//
// Returns:
//   - *HTTPError: Error with stack trace for method chaining
//
// Example:
//
//	err := blaze.ErrInternalServer("Unexpected error").WithStack(0)
func (e *HTTPError) WithStack(skip int) *HTTPError {
	e.Stack = captureStack(skip + 1)
	return e
}

// NewHTTPError creates a new HTTP error with the specified parameters
// Base constructor for creating custom HTTP errors
//
// Parameters:
//   - statusCode: HTTP status code (e.g., 400, 500)
//   - code: Error code for machine-readable identification
//   - message: Human-readable error message
//
// Returns:
//   - *HTTPError: New HTTP error instance
//
// Example:
//
//	err := blaze.NewHTTPError(402, "PAYMENT_REQUIRED", "Payment is required")
func NewHTTPError(statusCode int, code ErrorCode, message string) *HTTPError {
	return &HTTPError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Timestamp:  time.Now(),
	}
}

// NewHTTPErrorWithInternal creates an error with internal error
// Use when wrapping underlying errors (database, external APIs, etc.)
//
// Parameters:
//   - statusCode: HTTP status code
//   - code: Error code
//   - message: Human-readable message
//   - internal: Underlying error to wrap
//
// Returns:
//   - *HTTPError: New HTTP error with internal error
//
// Example:
//
//	err := blaze.NewHTTPErrorWithInternal(
//	    500,
//	    blaze.ErrCodeDatabaseError,
//	    "Failed to query database",
//	    dbErr,
//	)
func NewHTTPErrorWithInternal(statusCode int, code ErrorCode, message string, internal error) *HTTPError {
	return &HTTPError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Internal:   internal,
		Timestamp:  time.Now(),
	}
}

// Pre-defined error constructors for common HTTP errors
// These provide convenient shortcuts for standard error scenarios

// ErrBadRequest creates a 400 Bad Request error
// Use for malformed requests, invalid JSON, missing required fields
//
// Parameters:
//   - message: Description of what's wrong with the request
//
// Returns:
//   - *HTTPError: 400 error instance
//
// Example:
//
//	return blaze.ErrBadRequest("Missing required field: email")
func ErrBadRequest(message string) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, ErrCodeBadRequest, message)
}

// ErrUnauthorized creates a 401 Unauthorized error
// Use for authentication failures, missing tokens, expired sessions
//
// Parameters:
//   - message: Description of authentication failure
//
// Returns:
//   - *HTTPError: 401 error instance
//
// Example:
//
//	return blaze.ErrUnauthorized("Invalid or expired authentication token")
func ErrUnauthorized(message string) *HTTPError {
	return NewHTTPError(http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// ErrForbidden creates a 403 Forbidden error
// Use for authorization failures, insufficient permissions
//
// Parameters:
//   - message: Description of permission issue
//
// Returns:
//   - *HTTPError: 403 error instance
//
// Example:
//
//	return blaze.ErrForbidden("You don't have permission to access this resource"
func ErrForbidden(message string) *HTTPError {
	return NewHTTPError(http.StatusForbidden, ErrCodeForbidden, message)
}

// ErrNotFound creates a 404 Not Found error
// Use for missing resources, invalid IDs, non-existent endpoints
//
// Parameters:
//   - message: Description of what wasn't found
//
// Returns:
//   - *HTTPError: 404 error instance
//
// Example:
//
//	return blaze.ErrNotFound("User with ID 123 not found")
func ErrNotFound(message string) *HTTPError {
	return NewHTTPError(http.StatusNotFound, ErrCodeNotFound, message)
}

// ErrMethodNotAllowed creates a 405 Method Not Allowed error
// Use when HTTP method is not supported for the endpoint
//
// Parameters:
//   - message: Description of method restriction
//
// Returns:
//   - *HTTPError: 405 error instance
//
// Example:
//
//	return blaze.ErrMethodNotAllowed("POST method not allowed on this endpoint")
func ErrMethodNotAllowed(message string) *HTTPError {
	return NewHTTPError(http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, message)
}

// ErrConflict creates a 409 Conflict error
// Use for conflicts with current state, duplicate resources
//
// Parameters:
//   - message: Description of the conflict
//
// Returns:
//   - *HTTPError: 409 error instance
//
// Example:
//
//	return blaze.ErrConflict("User with this email already exists")
func ErrConflict(message string) *HTTPError {
	return NewHTTPError(http.StatusConflict, ErrCodeConflict, message)
}

// ErrValidation creates a 422 Validation Error
// Use for field validation failures, business rule violations
//
// Parameters:
//   - message: General validation error message
//   - details: Field-specific validation errors
//
// Returns:
//   - *HTTPError: 422 error instance with details
//
// Example:
//
//	return blaze.ErrValidation("Validation failed", map[string]string{
//	    "email": "Invalid email format",
//	    "age": "Must be at least 18",
//	})
func ErrValidation(message string, details interface{}) *HTTPError {
	return NewHTTPError(http.StatusUnprocessableEntity, ErrCodeValidation, message).WithDetails(details)
}

// ErrTooManyRequests creates a 429 Too Many Requests error
// Use for rate limiting, throttling
//
// Parameters:
//   - message: Description of rate limit
//
// Returns:
//   - *HTTPError: 429 error instance
//
// Example:
//
//	return blaze.ErrTooManyRequests("Rate limit exceeded. Try again in 60 seconds")
func ErrTooManyRequests(message string) *HTTPError {
	return NewHTTPError(http.StatusTooManyRequests, ErrCodeTooManyRequests, message)
}

// ErrInternalServer creates a 500 Internal Server Error
// Use for unexpected errors, unhandled exceptions
//
// Parameters:
//   - message: Generic error message (don't expose internals)
//
// Returns:
//   - *HTTPError: 500 error instance
//
// Example:
//
//	return blaze.ErrInternalServer("An unexpected error occurred")
func ErrInternalServer(message string) *HTTPError {
	return NewHTTPError(http.StatusInternalServerError, ErrCodeInternalServer, message)
}

// ErrInternalServerWithInternal creates a 500 error with internal error
// Use when wrapping underlying errors for logging
//
// Parameters:
//   - message: Generic user-facing message
//   - internal: Underlying error for server-side logging
//
// Returns:
//   - *HTTPError: 500 error instance with internal error
//
// Example:
//
//	return blaze.ErrInternalServerWithInternal(
//	    "Database operation failed",
//	    dbErr,
//	)
func ErrInternalServerWithInternal(message string, internal error) *HTTPError {
	return NewHTTPErrorWithInternal(http.StatusInternalServerError, ErrCodeInternalServer, message, internal)
}

// ErrNotImplemented creates a 501 Not Implemented error
// Use for planned but unimplemented features
//
// Parameters:
//   - message: Description of unimplemented feature
//
// Returns:
//   - *HTTPError: 501 error instance
//
// Example:
//
//	return blaze.ErrNotImplemented("This feature is coming soon")
func ErrNotImplemented(message string) *HTTPError {
	return NewHTTPError(http.StatusNotImplemented, ErrCodeNotImplemented, message)
}

// ErrServiceUnavailable creates a 503 Service Unavailable error
// Use for maintenance mode, temporary outages
//
// Parameters:
//   - message: Description of unavailability
//
// Returns:
//   - *HTTPError: 503 error instance
//
// Example:
//
//	return blaze.ErrServiceUnavailable("System is under maintenance")
func ErrServiceUnavailable(message string) *HTTPError {
	return NewHTTPError(http.StatusServiceUnavailable, ErrCodeServiceUnavailable, message)
}

// ErrGatewayTimeout creates a 504 Gateway Timeout error
// Use for upstream service timeouts
//
// Parameters:
//   - message: Description of timeout
//
// Returns:
//   - *HTTPError: 504 error instance
//
// Example:
//
//	return blaze.ErrGatewayTimeout("External API request timed out")
func ErrGatewayTimeout(message string) *HTTPError {
	return NewHTTPError(http.StatusGatewayTimeout, ErrCodeGatewayTimeout, message)
}

// Custom business errors

// ErrDatabase creates a database error with internal error
// Use for database-specific errors that need special handling
//
// Parameters:
//   - message: User-facing error message
//   - internal: Database error for logging
//
// Returns:
//   - *HTTPError: 500 error instance
//
// Example:
//
//	return blaze.ErrDatabase("Failed to save user", dbErr)
func ErrDatabase(message string, internal error) *HTTPError {
	return NewHTTPErrorWithInternal(http.StatusInternalServerError, ErrCodeDatabaseError, message, internal)
}

// ErrExternalAPI creates an external API error
// Use when third-party services fail
//
// Parameters:
//   - message: User-facing error message
//   - internal: External API error for logging
//
// Returns:
//   - *HTTPError: 502 error instance
//
// Example:
//
//	return blaze.ErrExternalAPI("Payment provider unavailable", apiErr)
func ErrExternalAPI(message string, internal error) *HTTPError {
	return NewHTTPErrorWithInternal(http.StatusBadGateway, ErrCodeExternalAPIError, message, internal)
}

// ErrAuthentication creates an authentication error
// Use for detailed authentication failure scenarios
//
// Parameters:
//   - message: Description of authentication failure
//
// Returns:
//   - *HTTPError: 401 error instance
//
// Example:
//
//	return blaze.ErrAuthentication("Invalid username or password")
func ErrAuthentication(message string) *HTTPError {
	return NewHTTPError(http.StatusUnauthorized, ErrCodeAuthenticationError, message)
}

// ErrAuthorization creates an authorization error
// Use for detailed authorization failure scenarios
//
// Parameters:
//   - message: Description of authorization failure
//
// Returns:
//   - *HTTPError: 403 error instance
//
// Example:
//
//	return blaze.ErrAuthorization("Admin role required for this operation")
func ErrAuthorization(message string) *HTTPError {
	return NewHTTPError(http.StatusForbidden, ErrCodeAuthorizationError, message)
}

// ErrorResponse represents the JSON error response structure
// Provides consistent error response format across the API
//
// Response Format Philosophy:
//   - Always include success: false for errors
//   - Structured error details for machine parsing
//   - Timestamp for correlation with logs
//   - Request context for debugging
//
// Example Response:
//
//	{
//	  "success": false,
//	  "error": {
//	    "code": "VALIDATION_ERROR",
//	    "message": "Invalid request data",
//	    "details": {
//	      "email": "Invalid format",
//	      "age": "Must be positive"
//	    }
//	  },
//	  "timestamp": "2024-01-01T12:00:00Z",
//	  "path": "/api/users",
//	  "method": "POST",
//	  "requestid": "req_abc123"
//	}
type ErrorResponse struct {
	// Success is always false for error responses
	Success bool `json:"success"`

	// Error contains the error details
	Error ErrorDetail `json:"error"`

	// Timestamp indicates when the error occurred
	Timestamp time.Time `json:"timestamp"`

	// Path is the request path that caused the error
	Path string `json:"path,omitempty"`

	// Method is the HTTP method of the failed request
	Method string `json:"method,omitempty"`

	// RequestID is the unique request identifier
	RequestID string `json:"requestid,omitempty"`

	// Metadata contains additional debugging information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorDetail contains detailed error information
// Nested structure for better organization of error data
type ErrorDetail struct {
	// Code is the machine-readable error code
	Code ErrorCode `json:"code"`

	// Message is the human-readable error message
	Message string `json:"message"`

	// Details contains additional error information
	Details interface{} `json:"details,omitempty"`

	// Stack contains the stack trace (development only)
	Stack []StackFrame `json:"stack,omitempty"`
}

// ToErrorResponse converts HTTPError to ErrorResponse
// Creates a complete error response ready for JSON serialization
//
// Response Processing:
//  1. Create base error response structure
//  2. Populate with error details
//  3. Add request context from Context
//  4. Include stack trace if requested
//  5. Add metadata for debugging
//
// Parameters:
//   - c: Request context (can be nil for standalone conversion)
//   - includeStack: Whether to include stack trace (development mode)
//
// Returns:
//   - ErrorResponse: Complete error response structure
//
// Example:
//
//	response := err.ToErrorResponse(c, true)
//	return c.Status(err.StatusCode).JSON(response)
func (e *HTTPError) ToErrorResponse(c *Context, includeStack bool) *ErrorResponse {
	resp := &ErrorResponse{
		Success:   false,
		Timestamp: e.Timestamp,
		Path:      e.Path,
		Method:    e.Method,
		RequestID: e.RequestID,
		Metadata:  e.Metadata,
		Error: ErrorDetail{
			Code:    e.Code,
			Message: e.Message,
			Details: e.Details,
		},
	}

	// Add context information if available
	if c != nil {
		if resp.Path == "" {
			resp.Path = c.Path()
		}
		if resp.Method == "" {
			resp.Method = c.Method()
		}
		if resp.RequestID == "" {
			resp.RequestID = c.GetUserValueString("request_id")
		}
	}

	// Include stack trace only in development mode
	if includeStack && len(e.Stack) > 0 {
		resp.Error.Stack = e.Stack
	}

	return resp
}

// captureStack captures the current stack trace
// Filters out runtime frames and formats for readability
//
// Parameters:
//   - skip: Number of frames to skip from the top
//
// Returns:
//   - []StackFrame: Captured stack frames
func captureStack(skip int) []StackFrame {
	const maxDepth = 32
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(skip+2, pcs)

	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	stack := make([]StackFrame, 0, n)

	for {
		frame, more := frames.Next()

		// Filter out runtime frames
		if !strings.Contains(frame.File, "runtime/") {
			stack = append(stack, StackFrame{
				File:     frame.File,
				Line:     frame.Line,
				Function: frame.Function,
			})
		}

		if !more {
			break
		}
	}

	return stack
}

// ErrorHandlerConfig configures error handling behavior across the application
// Provides centralized control over error logging, stack traces, and response formatting
//
// Error Handling Philosophy:
//   - Development: Show detailed errors with stack traces for debugging
//   - Production: Hide internal errors, log server-side, show generic messages to clients
//
// Security Considerations:
//   - Never expose internal error details in production
//   - Log sensitive errors server-side only
//   - Use generic error messages for client responses
//   - Sanitize error messages to prevent information leakage
//
// Best Practices:
//   - Use structured error types (HTTPError, ValidationErrors)
//   - Include request context (path, method, request ID) in errors
//   - Log errors with appropriate severity levels
//   - Provide actionable error messages to clients
type ErrorHandlerConfig struct {
	// IncludeStackTrace when true, includes stack traces in error responses
	// Should be true in development for debugging
	// Must be false in production for security
	IncludeStackTrace bool

	// LogErrors when true, automatically logs all errors
	// Useful for centralized error tracking and monitoring
	// Default: true
	LogErrors bool

	// Logger is a custom error logger function
	// If nil, errors are logged using standard log package
	// Use custom logger for structured logging (JSON, ELK stack, etc.)
	// Example: func(err error) { logger.Error("request_error", zap.Error(err)) }
	Logger func(err error)

	// HideInternalErrors when true, hides internal error details from clients
	// Internal errors are still logged server-side
	// Shows generic "internal server error" message to clients
	// Must be true in production for security
	// Default: true
	HideInternalErrors bool

	// CustomHandler allows complete customization of error handling
	// If provided, overrides default error handling logic
	// Receives context and error, returns error if handling fails
	// Return nil to indicate error was successfully handled
	// Example: func(c *Context, err error) error {
	//     log.Error(err)
	//     return c.Status(500).JSON(Map{"error": "custom error"})
	// }
	CustomHandler func(*Context, error) error
}

// DefaultErrorHandlerConfig returns production-safe error handler configuration
// Suitable for production deployments with security-conscious defaults
//
// Production Configuration:
//   - Stack traces: disabled (security)
//   - Error logging: enabled (monitoring)
//   - Internal errors: hidden (security)
//   - Custom handlers: none (standard behavior)
//
// This configuration ensures that:
//   - Clients receive generic error messages
//   - Internal errors are logged server-side
//   - No sensitive information leaks to clients
//   - Security best practices are followed
//
// Returns:
//   - ErrorHandlerConfig: Production-ready configuration
func DefaultErrorHandlerConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		IncludeStackTrace:  false,
		LogErrors:          true,
		HideInternalErrors: true,
		Logger:             nil,
		CustomHandler:      nil,
	}
}

// DevelopmentErrorHandlerConfig returns error handler config for development
// Includes detailed error information for easier debugging
//
// Development Configuration:
//   - Stack traces: enabled (debugging)
//   - Error logging: enabled (monitoring)
//   - Internal errors: visible (debugging)
//   - Custom handlers: none (standard behavior)
//
// This configuration helps developers:
//   - See full error context and stack traces
//   - Debug issues quickly with detailed information
//   - Understand error flow through the application
//   - Identify root causes of failures
//
// WARNING: Never use in production - exposes sensitive information
//
// Returns:
//   - ErrorHandlerConfig: Development-friendly configuration
func DevelopmentErrorHandlerConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		IncludeStackTrace:  true,
		LogErrors:          true,
		HideInternalErrors: false,
		Logger:             nil,
		CustomHandler:      nil,
	}
}

// HandleError handles errors and sends appropriate responses
// Central error processing function used by all error middleware
//
// Error Processing Steps:
//  1. Use custom handler if provided
//  2. Log error if configured
//  3. Determine error type and format
//  4. Set request context (path, method, request ID)
//  5. Create error response
//  6. Apply security filters (hide internal errors)
//  7. Send JSON response with appropriate status code
//
// Error Type Handling:
//   - HTTPError: Uses status code, code, and message from error
//   - ValidationErrors: Returns 400 with validation details
//   - Standard errors: Converts to 500 Internal Server Error
//
// Parameters:
//   - c: Request context
//   - err: Error to handle
//   - config: Error handler configuration
//
// Returns:
//   - error: Error if handling fails, nil if successful
func HandleError(c *Context, err error, config *ErrorHandlerConfig) error {
	if err == nil {
		return nil
	}

	// Use default config if not provided
	if config == nil {
		config = DefaultErrorHandlerConfig()
	}

	// Try custom handler first
	if config.CustomHandler != nil {
		return config.CustomHandler(c, err)
	}

	// Log error if enabled
	if config.LogErrors {
		if config.Logger != nil {
			config.Logger(err)
		} else {
			fmt.Printf("[ERROR] %v\n", err)
		}
	}

	// Handle HTTPError
	if httpErr, ok := err.(*HTTPError); ok {
		// Set context information
		httpErr.Path = c.Path()
		httpErr.Method = c.Method()
		httpErr.RequestID = c.GetUserValueString("request_id")

		// Convert to response
		response := httpErr.ToErrorResponse(c, config.IncludeStackTrace)

		// Hide internal error in production
		if config.HideInternalErrors && httpErr.Internal != nil {
			response.Error.Details = nil
		}

		return c.Status(httpErr.StatusCode).JSON(response)
	}

	// Handle ValidationErrors
	if validationErr, ok := err.(ValidationErrors); ok {
		httpErr := NewHTTPError(http.StatusBadRequest, ErrCodeValidation, "Validation failed")
		httpErr.Details = validationErr.Errors
		httpErr.Path = c.Path()
		httpErr.Method = c.Method()
		httpErr.RequestID = c.GetUserValueString("request_id")

		response := httpErr.ToErrorResponse(c, config.IncludeStackTrace)
		return c.Status(httpErr.StatusCode).JSON(response)
	}

	// Handle generic errors
	httpErr := ErrInternalServerWithInternal("An unexpected error occurred", err)
	httpErr.Path = c.Path()
	httpErr.Method = c.Method()
	httpErr.RequestID = c.GetUserValueString("request_id")

	response := httpErr.ToErrorResponse(c, config.IncludeStackTrace)

	// Hide error details in production
	if config.HideInternalErrors {
		response.Error.Details = nil
		response.Error.Message = "An internal server error occurred"
	}

	return c.Status(httpErr.StatusCode).JSON(response)
}

// MarshalJSON provides custom JSON marshaling for HTTPError
// Includes internal error message in development mode
//
// Returns:
//   - []byte: JSON representation
//   - error: Marshaling error or nil
func (e *HTTPError) MarshalJSON() ([]byte, error) {
	type Alias HTTPError
	return json.Marshal(&struct {
		*Alias
		Internal string `json:"internal,omitempty"`
	}{
		Alias: (*Alias)(e),
		Internal: func() string {
			if e.Internal != nil {
				return e.Internal.Error()
			}
			return ""
		}(),
	})
}

// IsHTTPError checks if an error is an HTTPError
// Use for type checking in error handling logic
//
// Parameters:
//   - err: Error to check
//
// Returns:
//   - bool: true if error is HTTPError
//
// Example:
//
//	if blaze.IsHTTPError(err) {
//	    // Handle as HTTPError
//	}
func IsHTTPError(err error) bool {
	_, ok := err.(*HTTPError)
	return ok
}

// GetHTTPError extracts HTTPError from error chain
// Searches error chain for wrapped HTTPError
//
// Parameters:
//   - err: Error that may contain HTTPError
//
// Returns:
//   - *HTTPError: Extracted HTTPError or nil
//   - bool: true if HTTPError found
//
// Example:
//
//	if httpErr, ok := blaze.GetHTTPError(err); ok {
//	    log.Printf("HTTP error: %d %s", httpErr.StatusCode, httpErr.Message)
//	}
func GetHTTPError(err error) (*HTTPError, bool) {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr, true
	}
	return nil, false
}
