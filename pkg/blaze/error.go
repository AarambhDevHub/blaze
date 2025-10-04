package blaze

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	// Client Errors (4xx)
	ErrCodeBadRequest       ErrorCode = "BAD_REQUEST"
	ErrCodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden        ErrorCode = "FORBIDDEN"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeMethodNotAllowed ErrorCode = "METHOD_NOT_ALLOWED"
	ErrCodeConflict         ErrorCode = "CONFLICT"
	ErrCodeValidation       ErrorCode = "VALIDATION_ERROR"
	ErrCodeTooManyRequests  ErrorCode = "TOO_MANY_REQUESTS"
	ErrCodeRequestTimeout   ErrorCode = "REQUEST_TIMEOUT"
	ErrCodePayloadTooLarge  ErrorCode = "PAYLOAD_TOO_LARGE"

	// Server Errors (5xx)
	ErrCodeInternalServer     ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrCodeNotImplemented     ErrorCode = "NOT_IMPLEMENTED"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeGatewayTimeout     ErrorCode = "GATEWAY_TIMEOUT"

	// Custom Business Errors
	ErrCodeDatabaseError       ErrorCode = "DATABASE_ERROR"
	ErrCodeExternalAPIError    ErrorCode = "EXTERNAL_API_ERROR"
	ErrCodeAuthenticationError ErrorCode = "AUTHENTICATION_ERROR"
	ErrCodeAuthorizationError  ErrorCode = "AUTHORIZATION_ERROR"
)

// HTTPError represents a structured HTTP error with additional context
type HTTPError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"status_code"`
	Details    interface{}            `json:"details,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Path       string                 `json:"path,omitempty"`
	Method     string                 `json:"method,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	Stack      []StackFrame           `json:"stack,omitempty"`
	Internal   error                  `json:"-"` // Internal error (not exposed in response)
}

// StackFrame represents a single stack frame
type StackFrame struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

// Error implements the error interface
func (e *HTTPError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error for error wrapping support
func (e *HTTPError) Unwrap() error {
	return e.Internal
}

// WithDetails adds additional details to the error
func (e *HTTPError) WithDetails(details interface{}) *HTTPError {
	e.Details = details
	return e
}

// WithMetadata adds metadata to the error
func (e *HTTPError) WithMetadata(key string, value interface{}) *HTTPError {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithInternal sets the internal error
func (e *HTTPError) WithInternal(err error) *HTTPError {
	e.Internal = err
	return e
}

// WithStack captures the current stack trace
func (e *HTTPError) WithStack(skip int) *HTTPError {
	e.Stack = captureStack(skip + 1)
	return e
}

// NewHTTPError creates a new HTTP error
func NewHTTPError(statusCode int, code ErrorCode, message string) *HTTPError {
	return &HTTPError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Timestamp:  time.Now(),
	}
}

// NewHTTPErrorWithInternal creates an error with internal error
func NewHTTPErrorWithInternal(statusCode int, code ErrorCode, message string, internal error) *HTTPError {
	return &HTTPError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Internal:   internal,
		Timestamp:  time.Now(),
	}
}

// Pre-defined error constructors

// ErrBadRequest creates a 400 Bad Request error
func ErrBadRequest(message string) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, ErrCodeBadRequest, message)
}

// ErrUnauthorized creates a 401 Unauthorized error
func ErrUnauthorized(message string) *HTTPError {
	return NewHTTPError(http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

// ErrForbidden creates a 403 Forbidden error
func ErrForbidden(message string) *HTTPError {
	return NewHTTPError(http.StatusForbidden, ErrCodeForbidden, message)
}

// ErrNotFound creates a 404 Not Found error
func ErrNotFound(message string) *HTTPError {
	return NewHTTPError(http.StatusNotFound, ErrCodeNotFound, message)
}

// ErrMethodNotAllowed creates a 405 Method Not Allowed error
func ErrMethodNotAllowed(message string) *HTTPError {
	return NewHTTPError(http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, message)
}

// ErrConflict creates a 409 Conflict error
func ErrConflict(message string) *HTTPError {
	return NewHTTPError(http.StatusConflict, ErrCodeConflict, message)
}

// ErrValidation creates a 422 Validation Error
func ErrValidation(message string, details interface{}) *HTTPError {
	return NewHTTPError(http.StatusUnprocessableEntity, ErrCodeValidation, message).WithDetails(details)
}

// ErrTooManyRequests creates a 429 Too Many Requests error
func ErrTooManyRequests(message string) *HTTPError {
	return NewHTTPError(http.StatusTooManyRequests, ErrCodeTooManyRequests, message)
}

// ErrInternalServer creates a 500 Internal Server Error
func ErrInternalServer(message string) *HTTPError {
	return NewHTTPError(http.StatusInternalServerError, ErrCodeInternalServer, message)
}

// ErrInternalServerWithInternal creates a 500 error with internal error
func ErrInternalServerWithInternal(message string, internal error) *HTTPError {
	return NewHTTPErrorWithInternal(http.StatusInternalServerError, ErrCodeInternalServer, message, internal)
}

// ErrNotImplemented creates a 501 Not Implemented error
func ErrNotImplemented(message string) *HTTPError {
	return NewHTTPError(http.StatusNotImplemented, ErrCodeNotImplemented, message)
}

// ErrServiceUnavailable creates a 503 Service Unavailable error
func ErrServiceUnavailable(message string) *HTTPError {
	return NewHTTPError(http.StatusServiceUnavailable, ErrCodeServiceUnavailable, message)
}

// ErrGatewayTimeout creates a 504 Gateway Timeout error
func ErrGatewayTimeout(message string) *HTTPError {
	return NewHTTPError(http.StatusGatewayTimeout, ErrCodeGatewayTimeout, message)
}

// Custom business errors

// ErrDatabase creates a database error
func ErrDatabase(message string, internal error) *HTTPError {
	return NewHTTPErrorWithInternal(http.StatusInternalServerError, ErrCodeDatabaseError, message, internal)
}

// ErrExternalAPI creates an external API error
func ErrExternalAPI(message string, internal error) *HTTPError {
	return NewHTTPErrorWithInternal(http.StatusBadGateway, ErrCodeExternalAPIError, message, internal)
}

// ErrAuthentication creates an authentication error
func ErrAuthentication(message string) *HTTPError {
	return NewHTTPError(http.StatusUnauthorized, ErrCodeAuthenticationError, message)
}

// ErrAuthorization creates an authorization error
func ErrAuthorization(message string) *HTTPError {
	return NewHTTPError(http.StatusForbidden, ErrCodeAuthorizationError, message)
}

// ErrorResponse represents the JSON error response structure
type ErrorResponse struct {
	Success   bool                   `json:"success"`
	Error     ErrorDetail            `json:"error"`
	Timestamp time.Time              `json:"timestamp"`
	Path      string                 `json:"path,omitempty"`
	Method    string                 `json:"method,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code    ErrorCode    `json:"code"`
	Message string       `json:"message"`
	Details interface{}  `json:"details,omitempty"`
	Stack   []StackFrame `json:"stack,omitempty"`
}

// ToErrorResponse converts HTTPError to ErrorResponse
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

// ErrorHandlerConfig configures error handling behavior
type ErrorHandlerConfig struct {
	// Include stack traces in development mode
	IncludeStackTrace bool

	// Log errors automatically
	LogErrors bool

	// Custom error logger function
	Logger func(err error)

	// Hide internal errors in production
	HideInternalErrors bool

	// Custom error handler
	CustomHandler func(*Context, error) error
}

// DefaultErrorHandlerConfig returns default error handler configuration
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
func DevelopmentErrorHandlerConfig() *ErrorHandlerConfig {
	return &ErrorHandlerConfig{
		IncludeStackTrace:  true,
		LogErrors:          true,
		HideInternalErrors: false,
		Logger:             nil,
		CustomHandler:      nil,
	}
}

// HandleError handles errors and sends appropriate response
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
func IsHTTPError(err error) bool {
	_, ok := err.(*HTTPError)
	return ok
}

// GetHTTPError extracts HTTPError from error chain
func GetHTTPError(err error) (*HTTPError, bool) {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr, true
	}
	return nil, false
}
