package blaze

import (
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
)

// RequestIDKey is the key for storing the request ID in Context locals
const RequestIDKey = "requestid"

// generateUUIDv4 generates a random RFC 4122 UUID (version 4)
func generateUUIDv4() string {
	u := [16]byte{}
	_, err := io.ReadFull(crand.Reader, u[:])
	if err != nil {
		// Fall back to timestamp/random if entropy fails (rare)
		return fmt.Sprintf("%d-%d", rand.Int63(), rand.Int63())
	}
	u[6] = (u[6] & 0x0f) | 0x40 // version 4
	u[8] = (u[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

// RequestIDMiddleware sets/generates a request ID and stores it in context and response header.
func RequestIDMiddleware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Try to get request id from incoming request
			reqID := c.Header("X-Request-Id")
			if reqID == "" {
				reqID = generateUUIDv4()
			}
			// Store in context locals for handler usage/logging
			c.SetLocals(RequestIDKey, reqID)
			// Set in outgoing header for client and upchain correlation
			c.SetHeader("X-Request-Id", reqID)
			return next(c)
		}
	}
}

// GetRequestID returns the current request ID from the Blaze context
func GetRequestID(c Context) string {
	if v := c.Locals(RequestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
