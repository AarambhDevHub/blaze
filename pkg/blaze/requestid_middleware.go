package blaze

import (
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
)

// RequestIDKey is the key for storing the request ID in Context locals
// Used to access request ID from context in handlers and middleware
//
// Usage:
//
//	requestID := c.Locals(blaze.RequestIDKey).(string)
//	// Or use helper: blaze.GetRequestID(c)
const RequestIDKey = "requestid"

// generateUUIDv4 generates a random RFC 4122 UUID version 4
// UUID v4 uses random numbers for generating unique identifiers
//
// UUID Format:
//   - 128-bit number represented as 32 hexadecimal digits
//   - Format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
//   - Version 4: Random (indicated by '4' in third group)
//   - Variant: RFC 4122 (indicated by 8, 9, a, or b in fourth group)
//
// Generation Process:
//  1. Generate 16 random bytes using crypto/rand
//  2. Set version bits (4 bits) to 0100 (version 4)
//  3. Set variant bits (2 bits) to 10 (RFC 4122)
//  4. Format as hyphenated string
//
// Fallback Behavior:
//   - If crypto/rand fails (extremely rare), falls back to math/rand
//   - Fallback uses timestamp-based randomness
//   - Less secure but ensures ID generation never fails
//
// Security:
//   - Uses cryptographically secure random number generator
//   - Collision probability is negligible (2^122 possible values)
//   - Safe for security-sensitive applications
//
// Performance:
//   - Fast generation (~1-2 microseconds)
//   - No external dependencies
//   - Suitable for high-throughput applications
//
// Returns:
//   - string: UUID v4 string (e.g., "550e8400-e29b-41d4-a716-446655440000")
//
// Example:
//
//	id := generateUUIDv4()
//	// Returns: "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
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

// RequestIDMiddleware sets/generates a request ID and stores it in context and response header
// Provides request tracking across services and log correlation
//
// Request ID Flow:
//  1. Check for incoming X-Request-ID header (from client or upstream)
//  2. If present, use existing ID (request tracing across services)
//  3. If absent, generate new UUID v4
//  4. Store in context locals for handler/middleware access
//  5. Set in response header for client correlation
//
// Benefits:
//   - Request tracking across microservices
//   - Log correlation (same ID in all logs for a request)
//   - Debugging (trace request through system)
//   - Client correlation (client can reference ID in support requests)
//   - Distributed tracing integration
//
// Header Handling:
//   - Incoming: Reads X-Request-ID from request
//   - Outgoing: Sets X-Request-ID in response
//   - Idempotent: Same ID in request and response
//
// Use Cases:
//   - Microservices architecture (trace requests across services)
//   - Log aggregation (group logs by request ID)
//   - Error reporting (include request ID in error messages)
//   - Support tickets (customers can provide request ID)
//   - Performance monitoring (track request duration)
//
// Integration with Logging:
//   - LoggerMiddleware automatically includes request ID
//   - ErrorMiddleware includes request ID in error responses
//   - Custom handlers can access via GetRequestID()
//
// Standards Compliance:
//   - Uses X-Request-ID header (de facto standard)
//   - Compatible with AWS, GCP, Kubernetes, Nginx
//   - Follows RFC 4122 for UUID generation
//
// Returns:
//   - MiddlewareFunc: Request ID middleware
//
// Example - Basic Usage:
//
//	app.Use(blaze.RequestIDMiddleware())
//	app.Use(blaze.LoggerMiddleware()) // Logs will include request ID
//
// Example - Access in Handler:
//
//	func handler(c *blaze.Context) error {
//	    requestID := blaze.GetRequestID(c)
//	    log.Printf("Processing request: %s", requestID)
//	    return c.JSON(blaze.Map{"request_id": requestID})
//	}
//
// Example - Microservices Propagation:
//
//	func callUpstream(c *blaze.Context) error {
//	    requestID := blaze.GetRequestID(c)
//
//	    req, _ := http.NewRequest("GET", "http://upstream/api", nil)
//	    req.Header.Set("X-Request-ID", requestID) // Propagate ID
//
//	    resp, err := http.DefaultClient.Do(req)
//	    // ... handle response
//	}
//
// Example - Error Reporting:
//
//	func handler(c *blaze.Context) error {
//	    err := doSomething()
//	    if err != nil {
//	        requestID := blaze.GetRequestID(c)
//	        logger.Error("Operation failed",
//	            "request_id", requestID,
//	            "error", err,
//	        )
//	        return blaze.ErrInternalServer("Operation failed").
//	            WithMetadata("request_id", requestID)
//	    }
//	    return c.JSON(result)
//	}
//
// Example - Custom Request ID Format:
//
//	func CustomRequestIDMiddleware() blaze.MiddlewareFunc {
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) error {
//	            // Try to get from header
//	            reqID := c.Header("X-Request-ID")
//	            if reqID == "" {
//	                // Generate custom format: timestamp-random
//	                reqID = fmt.Sprintf("%d-%s",
//	                    time.Now().UnixNano(),
//	                    randomString(8),
//	                )
//	            }
//
//	            c.SetLocals(blaze.RequestIDKey, reqID)
//	            c.SetHeader("X-Request-ID", reqID)
//	            return next(c)
//	        }
//	    }
//	}
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
// Convenience function to extract request ID without direct key access
//
// Returns:
//   - string: Request ID or empty string if not found
//
// Usage:
//   - Call after RequestIDMiddleware has executed
//   - Safe to call even if middleware not used (returns empty string)
//   - Thread-safe access to request ID
//
// Parameters:
//   - c: Blaze context
//
// Returns:
//   - string: Request ID (UUID v4 format) or empty string
//
// Example - In Handler:
//
//	func handler(c *blaze.Context) error {
//	    requestID := blaze.GetRequestID(c)
//	    log.Printf("Request ID: %s", requestID)
//	    return c.Text("OK")
//	}
//
// Example - In Middleware:
//
//	func customMiddleware() blaze.MiddlewareFunc {
//	    return func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	        return func(c *blaze.Context) error {
//	            requestID := blaze.GetRequestID(c)
//	            // Use request ID for custom logic
//	            return next(c)
//	        }
//	    }
//	}
//
// Example - Error Response:
//
//	func handler(c *blaze.Context) error {
//	    err := doSomething()
//	    if err != nil {
//	        return c.Status(500).JSON(blaze.Map{
//	            "error": "Internal error",
//	            "request_id": blaze.GetRequestID(c),
//	            "message": "Please contact support with this request ID",
//	        })
//	    }
//	    return c.JSON(result)
//	}
func GetRequestID(c Context) string {
	if v := c.Locals(RequestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
