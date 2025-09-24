package blaze

import (
	"net"
	"sync"
	"time"
)

// RateLimitOptions configures rate limiting
type RateLimitOptions struct {
	Requests int           // allowed requests per window
	Window   time.Duration // window duration
}

// rateLimitInfo tracks request count & window for each key
type rateLimitInfo struct {
	Timestamp time.Time
	Count     int
}

// RateLimiter manages IP request state
type RateLimiter struct {
	opts    RateLimitOptions
	mu      sync.Mutex
	clients map[string]*rateLimitInfo
}

// NewRateLimiter creates a new limiter
func NewRateLimiter(opts RateLimitOptions) *RateLimiter {
	return &RateLimiter{
		opts:    opts,
		clients: make(map[string]*rateLimitInfo),
	}
}

// cleanup removes expired records (should be called periodically)
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

// Allow increments and checks quota for key (thread safe)
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

// getClientIP extracts client IP best-effort
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

// RateLimitMiddleware creates Blaze Middleware for rate limiting per IP
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
