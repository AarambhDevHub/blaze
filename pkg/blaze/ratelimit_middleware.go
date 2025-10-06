package blaze

import (
	"net"
	"sync"
	"time"
)

// RateLimitOptions configures rate limiting behavior
// Implements token bucket algorithm for request throttling
//
// Rate Limiting Algorithms:
//   - Token Bucket: Allows burst traffic while maintaining average rate
//   - Fixed Window: Counts requests in fixed time intervals
//   - Sliding Window: More accurate than fixed window
//
// Token Bucket Implementation:
//   - Each client has a bucket with tokens
//   - Each request consumes one token
//   - Tokens refill at a constant rate
//   - Request rejected when bucket is empty
//
// Use Cases:
//   - API rate limiting (per user, per IP)
//   - DOS/DDOS protection
//   - Fair resource allocation
//   - Abuse prevention
//   - Cost control for external APIs
//
// Configuration Philosophy:
//   - Development: Generous limits for testing
//   - Production: Strict limits based on capacity and SLA
//   - Premium tiers: Higher limits for paid users
type RateLimitOptions struct {
	// Requests specifies allowed requests per window
	// Number of requests allowed within the time window
	// Example: 100 requests per minute
	// Set based on:
	//   - Server capacity
	//   - User tier (free vs paid)
	//   - Endpoint sensitivity
	// Recommended: 60-1000 for typical APIs
	Requests int

	// Window specifies the time window duration
	// Time period for counting requests
	// Common values:
	//   - 1 second: Strict short-term limiting
	//   - 1 minute: Standard API limiting
	//   - 1 hour: Generous long-term limiting
	// Shorter windows provide better burst protection
	// Longer windows are more forgiving to legitimate users
	Window time.Duration
}

// rateLimitInfo tracks request count and window for each client
// Internal structure for maintaining rate limit state
//
// State Management:
//   - Timestamp: Start of current window
//   - Count: Number of requests in current window
//   - Automatically resets when window expires
type rateLimitInfo struct {
	// Timestamp records when the current window started
	// Used to determine if window has expired
	Timestamp time.Time

	// Count tracks requests in the current window
	// Incremented on each request
	// Reset when window expires
	Count int
}

// RateLimiter manages rate limiting state for multiple clients
// Thread-safe implementation supporting concurrent requests
//
// Architecture:
//   - In-memory storage (fast, no external dependencies)
//   - Per-IP tracking (can be extended to per-user)
//   - Automatic cleanup of expired entries
//   - Mutex-protected concurrent access
//
// Memory Management:
//   - Stores state for each unique IP
//   - Cleanup removes expired entries
//   - Memory usage grows with number of unique clients
//   - Consider Redis for distributed systems
//
// Scalability:
//   - Single server: In-memory is sufficient
//   - Multiple servers: Use Redis or similar
//   - High traffic: Consider distributed rate limiting
type RateLimiter struct {
	// opts holds the rate limiting configuration
	opts RateLimitOptions

	// mu protects concurrent access to clients map
	// Ensures thread-safe read/write operations
	mu sync.Mutex

	// clients maps client identifiers to their rate limit info
	// Key: IP address or user ID
	// Value: Request count and window timestamp
	clients map[string]*rateLimitInfo
}

// NewRateLimiter creates a new rate limiter instance
// Initializes limiter with specified options
//
// Configuration Examples:
//   - Strict: 10 requests per second
//   - Standard: 100 requests per minute
//   - Generous: 1000 requests per hour
//
// Parameters:
//   - opts: Rate limit configuration
//
// Returns:
//   - *RateLimiter: Configured rate limiter instance
//
// Example - API Rate Limiting:
//
//	limiter := blaze.NewRateLimiter(blaze.RateLimitOptions{
//	    Requests: 100,
//	    Window: time.Minute,
//	})
func NewRateLimiter(opts RateLimitOptions) *RateLimiter {
	return &RateLimiter{
		opts:    opts,
		clients: make(map[string]*rateLimitInfo),
	}
}

// cleanup removes expired rate limit records
// Should be called periodically to prevent memory leaks
//
// Cleanup Strategy:
//   - Iterates through all client entries
//   - Removes entries older than window duration
//   - Frees memory for expired clients
//
// Cleanup Frequency:
//   - Should run at interval ~= window duration
//   - More frequent cleanup uses more CPU
//   - Less frequent cleanup uses more memory
//
// Thread Safety:
//   - Acquires lock to modify clients map
//   - Safe to call concurrently with Allow()
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for ip, info := range rl.clients {
		if now.Sub(info.Timestamp) > rl.opts.Window {
			delete(rl.clients, ip)
		}
	}
}

// Allow checks if a request from the given IP should be allowed
// Implements token bucket algorithm with automatic window reset
//
// Algorithm:
//  1. Check if client exists in cache
//  2. If not, create new entry with count=1
//  3. If window expired, reset with count=1
//  4. If within window:
//     - If count < limit: increment and allow
//     - If count >= limit: reject
//
// Return Values:
//   - true: Request allowed, quota available
//   - false: Request rejected, quota exceeded
//
// Thread Safety:
//   - Mutex-protected, safe for concurrent calls
//
// Parameters:
//   - ip: Client identifier (IP address or user ID)
//
// Returns:
//   - bool: true if request is allowed
//
// Example:
//
//	if limiter.Allow("192.168.1.1") {
//	    // Process request
//	} else {
//	    // Return 429 Too Many Requests
//	}
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	info, ok := rl.clients[ip]
	if !ok || now.Sub(info.Timestamp) > rl.opts.Window {
		rl.clients[ip] = &rateLimitInfo{Timestamp: now, Count: 1}
		return true
	}
	if info.Count < rl.opts.Requests {
		info.Count++
		return true
	}
	return false
}

// getClientIP extracts client IP with proxy support
// Tries multiple headers to find real client IP
//
// IP Extraction Priority:
//  1. X-Forwarded-For (comma-separated list, use first)
//  2. X-Real-IP (Nginx proxy)
//  3. RemoteAddr (direct connection)
//
// Security Considerations:
//   - X-Forwarded-For can be spoofed
//   - Only trust these headers if behind trusted proxy
//   - Configure proxy to set headers correctly
//   - Consider IP whitelist for proxy IPs
//
// Parameters:
//   - c: Request context
//
// Returns:
//   - string: Client IP address
func getClientIP(c *Context) string {
	ip := c.Header("X-Forwarded-For")
	if ip != "" {
		ips := net.ParseIP(ip)
		if ips != nil {
			return ips.String()
		}
	}
	realIP := c.Header("X-Real-IP")
	if realIP != "" {
		ips := net.ParseIP(realIP)
		if ips != nil {
			return ips.String()
		}
	}
	return c.IP()
}

// RateLimitMiddleware creates Blaze middleware for rate limiting per IP
// Automatically limits requests based on client IP address
//
// Middleware Behavior:
//   - Extracts client IP from request
//   - Checks against rate limit
//   - Allows request if within limit
//   - Returns 429 Too Many Requests if exceeded
//
// IP Extraction Priority:
//  1. X-Forwarded-For header (proxy/load balancer)
//  2. X-Real-IP header (Nginx proxy)
//  3. RemoteAddr (direct connection)
//
// Response on Rate Limit:
//   - Status: 429 Too Many Requests
//   - Body: {"error": "Too Many Requests", "detail": "Rate limit exceeded..."}
//   - Headers: Can add Retry-After header
//
// Background Cleanup:
//   - Starts automatic cleanup goroutine
//   - Runs every window/2 duration
//   - Prevents memory leaks
//
// Parameters:
//   - opts: Rate limit configuration
//
// Returns:
//   - MiddlewareFunc: Rate limiting middleware
//
// Example - 100 requests per minute:
//
//	app.Use(blaze.RateLimitMiddleware(blaze.RateLimitOptions{
//	    Requests: 100,
//	    Window: time.Minute,
//	}))
//
// Example - Per-route limits:
//
//	strict := blaze.RateLimitOptions{Requests: 10, Window: time.Minute}
//	generous := blaze.RateLimitOptions{Requests: 1000, Window: time.Minute}
//
//	app.POST("/login", handler, blaze.RateLimitMiddleware(strict))
//	app.GET("/api/data", handler, blaze.RateLimitMiddleware(generous))
//
// Example - With custom error response:
//
//	limiter := blaze.NewRateLimiter(opts)
//	middleware := func(next blaze.HandlerFunc) blaze.HandlerFunc {
//	    return func(c *blaze.Context) error {
//	        if !limiter.Allow(getClientIP(c)) {
//	            return c.Status(429).JSON(blaze.Map{
//	                "error": "Rate limit exceeded",
//	                "retry_after": int(opts.Window.Seconds()),
//	            })
//	        }
//	        return next(c)
//	    }
//	}
//	app.Use(middleware)
func RateLimitMiddleware(opts RateLimitOptions) MiddlewareFunc {
	limiter := NewRateLimiter(opts)
	go func() {
		ticker := time.NewTicker(opts.Window * 2)
		for range ticker.C {
			limiter.cleanup()
		}
	}()
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			ip := getClientIP(c)
			if !limiter.Allow(ip) {
				return c.Status(429).JSON(Map{
					"error":  "Too Many Requests",
					"detail": "Rate limit exceeded. Please try again later.",
				})
			}
			return next(c)
		}
	}
}

// GetInfo returns current rate limit info for a client
// Useful for debugging and monitoring
//
// Parameters:
//   - ip: Client identifier
//
// Returns:
//   - *rateLimitInfo: Current rate limit state or nil if not found
func (rl *RateLimiter) GetInfo(ip string) *rateLimitInfo {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.clients[ip]
}

// Reset clears rate limit state for a client
// Useful for administrative actions or testing
//
// Parameters:
//   - ip: Client identifier to reset
func (rl *RateLimiter) Reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.clients, ip)
}

// ResetAll clears all rate limit state
// Useful for system maintenance or emergency situations
func (rl *RateLimiter) ResetAll() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.clients = make(map[string]*rateLimitInfo)
}

// GetStats returns rate limiter statistics
// Provides insight into rate limiting effectiveness
//
// Returns:
//   - RateLimitStats: Current statistics
func (rl *RateLimiter) GetStats() RateLimitStats {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return RateLimitStats{
		TotalClients: len(rl.clients),
		Requests:     rl.opts.Requests,
		Window:       rl.opts.Window,
	}
}

// RateLimitStats holds rate limiter statistics
type RateLimitStats struct {
	TotalClients int           `json:"total_clients"`
	Requests     int           `json:"requests"`
	Window       time.Duration `json:"window"`
}
