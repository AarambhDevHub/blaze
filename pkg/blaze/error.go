package blaze

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/valyala/fasthttp"
)

// ErrorResponse represents a structured JSON error response.
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// NewErrorResponse returns an ErrorResponse with the given message.
func NewErrorResponse(msg string) ErrorResponse {
	return ErrorResponse{Success: false, Error: msg}
}

// JSONError writes structured JSON error response
func JSONError(ctx *fasthttp.RequestCtx, statusCode int, msg string) error {
	ctx.SetStatusCode(statusCode)
	ctx.SetContentType("application/json; charset=utf-8")

	errResp := NewErrorResponse(msg)
	jsonData, err := json.Marshal(errResp)
	if err != nil {
		ctx.SetBody([]byte(fmt.Sprintf(`{"success":false,"error":"%s"}`, msg)))
		return err
	}

	ctx.SetBody(jsonData)
	return nil
}

// RecoveryMiddleware recovers from panics in handlers and logs the error,
// then returns a 500 Internal Server Error with a JSON error response.
func RecoveryMiddleware(next HandlerFunc) HandlerFunc {
	return func(c *Context) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC recovered: %v\n%s", r, debug.Stack())
				_ = JSONError(c.RequestCtx, fasthttp.StatusInternalServerError, "Internal Server Error")
			}
		}()
		return next(c)
	}
}

// NotFoundError sends a 404 Not Found JSON error response.
func NotFoundError(ctx *fasthttp.RequestCtx, msg string) error {
	return JSONError(ctx, fasthttp.StatusNotFound, msg)
}

// BadRequestError sends a 400 Bad Request JSON error response.
func BadRequestError(ctx *fasthttp.RequestCtx, msg string) error {
	return JSONError(ctx, fasthttp.StatusBadRequest, msg)
}

// UnauthorizedError sends a 401 Unauthorized JSON error response.
func UnauthorizedError(ctx *fasthttp.RequestCtx, msg string) error {
	return JSONError(ctx, fasthttp.StatusUnauthorized, msg)
}

// ForbiddenError sends a 403 Forbidden JSON error response.
func ForbiddenError(ctx *fasthttp.RequestCtx, msg string) error {
	return JSONError(ctx, fasthttp.StatusForbidden, msg)
}
