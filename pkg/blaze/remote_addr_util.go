package blaze

import (
	"net"
	"strings"

	"github.com/valyala/fasthttp"
)

// SetRemoteAddr sets the remote address for a fasthttp request context
// Properly handles IP address parsing and TCP address creation
//
// Address Handling:
//   - Attempts to parse as TCP address first (includes port)
//   - Falls back to IP-only parsing if host:port format fails
//   - Creates proper TCP address structure for fasthttp
//
// Use Cases:
//   - Setting client IP in middleware
//   - Testing with mock addresses
//   - Proxy/load balancer IP forwarding
//
// Parameters:
//   - ctx: fasthttp request context
//   - remoteAddr: Remote address string (IP or IP:port format)
//
// Example:
//
//	blaze.SetRemoteAddr(ctx, "192.168.1.100:54321")
//	blaze.SetRemoteAddr(ctx, "192.168.1.100")
func SetRemoteAddr(ctx *fasthttp.RequestCtx, remoteAddr string) {
	// Parse the remote address
	if tcpAddr, err := net.ResolveTCPAddr("tcp", remoteAddr); err == nil {
		ctx.SetRemoteAddr(tcpAddr)
	} else {
		// If parsing fails, try to extract just the IP
		if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
			if ip := net.ParseIP(host); ip != nil {
				// Create a TCP address with the IP and a default port
				tcpAddr := &net.TCPAddr{
					IP:   ip,
					Port: 80, // Default port
				}
				ctx.SetRemoteAddr(tcpAddr)
			}
		}
	}
}

// GetRealIP extracts the real client IP from various headers
// Implements best practices for IP extraction behind proxies and load balancers
//
// IP Extraction Priority:
//  1. X-Forwarded-For: Standard proxy header (comma-separated list)
//  2. X-Real-IP: Nginx-style real IP header
//  3. CF-Connecting-IP: Cloudflare-specific header
//  4. RemoteAddr: Direct connection IP (fallback)
//
// Security Considerations:
//   - X-Forwarded-For can be spoofed by clients
//   - Only trust these headers when behind a trusted proxy
//   - Configure proxy to set headers correctly
//   - Consider IP whitelist for proxy servers
//   - Validate IP format before trusting
//
// Proxy Configurations:
//   - Nginx: Sets X-Real-IP and X-Forwarded-For
//   - Apache: Sets X-Forwarded-For
//   - Cloudflare: Sets CF-Connecting-IP
//   - AWS ELB: Sets X-Forwarded-For
//
// X-Forwarded-For Format:
//   - "client, proxy1, proxy2"
//   - First IP is the original client
//   - Subsequent IPs are intermediate proxies
//
// Parameters:
//   - ctx: fasthttp request context
//
// Returns:
//   - string: Real client IP address
//
// Example - Direct Connection:
//
//	ip := blaze.GetRealIP(ctx)
//	// Returns: "192.168.1.100"
//
// Example - Behind Proxy:
//
//	// X-Forwarded-For: "203.0.113.1, 10.0.0.1"
//	ip := blaze.GetRealIP(ctx)
//	// Returns: "203.0.113.1" (first IP in list)
//
// Example - Cloudflare:
//
//	// CF-Connecting-IP: "203.0.113.1"
//	ip := blaze.GetRealIP(ctx)
//	// Returns: "203.0.113.1"
func GetRealIP(ctx *fasthttp.RequestCtx) string {
	// Check X-Forwarded-For header
	if xff := string(ctx.Request.Header.Peek("X-Forwarded-For")); xff != "" {
		// Get the first IP from the comma-separated list
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := string(ctx.Request.Header.Peek("X-Real-IP")); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Check CF-Connecting-IP header (Cloudflare)
	if cfip := string(ctx.Request.Header.Peek("CF-Connecting-IP")); cfip != "" {
		return strings.TrimSpace(cfip)
	}

	// Fall back to RemoteAddr
	remoteAddr := ctx.RemoteAddr()
	if remoteAddr != nil {
		if tcpAddr, ok := remoteAddr.(*net.TCPAddr); ok {
			return tcpAddr.IP.String()
		}
		return remoteAddr.String()
	}

	return ""
}

// SetClientIP sets various IP-related information in the context user values
// Stores client IP, real IP, and remote address for later access
//
// Stored Values:
//   - "clientip": Extracted real IP (from headers or RemoteAddr)
//   - "realip": Same as clientip (for compatibility)
//   - "remoteaddr": Raw remote address string with port
//
// Use Cases:
//   - IP-based rate limiting
//   - Geolocation
//   - Access logging
//   - Security auditing
//
// Access in Handlers:
//   - c.GetClientIP() -> "192.168.1.100"
//   - c.GetRealIP() -> "192.168.1.100"
//   - c.GetRemoteAddr() -> "192.168.1.100:54321"
//
// Parameters:
//   - ctx: fasthttp request context
//
// Example - In Middleware:
//
//	blaze.SetClientIP(ctx)
//	// Later access:
//	clientIP := string(ctx.UserValue("clientip").(string))
func SetClientIP(ctx *fasthttp.RequestCtx) {
	realIP := GetRealIP(ctx)
	if realIP != "" {
		ctx.SetUserValue("client_ip", realIP)
		ctx.SetUserValue("real_ip", realIP)
	}

	// Also store the raw remote address
	if remoteAddr := ctx.RemoteAddr(); remoteAddr != nil {
		ctx.SetUserValue("remote_addr", remoteAddr.String())
	}
}

// SetClientIPForContext sets client IP for Blaze context
// Convenience wrapper for SetClientIP that works with Blaze Context
//
// Parameters:
//   - c: Blaze context
//
// Example:
//
//	blaze.SetClientIPForContext(c)
func SetClientIPForContext(c *Context) {
	SetClientIP(c.RequestCtx)
}

// IPMiddleware is middleware that extracts and stores client IP information
// Automatically called before handlers to populate IP information
//
// Middleware Flow:
//  1. Extract real IP from headers
//  2. Store in context user values
//  3. Make available to handlers via c.GetClientIP()
//
// Use Cases:
//   - Rate limiting by IP
//   - Geolocation-based features
//   - Access logging
//   - Security monitoring
//
// Execution Order:
//   - Should be registered early in middleware chain
//   - Before rate limiting or logging middleware
//   - After proxy/load balancer middleware if any
//
// Returns:
//   - MiddlewareFunc: IP extraction middleware
//
// Example - Global Middleware:
//
//	app.Use(blaze.IPMiddleware())
//
// Example - Route-Specific:
//
//	app.GET("/api/users", handler, blaze.IPMiddleware())
//
// Example - With Rate Limiting:
//
//	app.Use(blaze.IPMiddleware())
//	app.Use(blaze.RateLimitMiddleware(opts))
//	// Rate limiter can now access real client IP
//
// Example - Access in Handler:
//
//	func handler(c *blaze.Context) error {
//	    clientIP := c.GetClientIP()
//	    log.Printf("Request from IP: %s", clientIP)
//	    return c.JSON(blaze.Map{"ip": clientIP})
//	}
func IPMiddleware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			SetClientIPForContext(c)
			return next(c)
		}
	}
}

// TrustedProxies configures trusted proxy IP addresses
// Only these proxies can set X-Forwarded-For and similar headers
//
// Security Model:
//   - Only accept forwarded headers from trusted proxies
//   - Prevent IP spoofing by untrusted clients
//   - Validate proxy IP before trusting headers
//
// Configuration:
//   - List of trusted proxy IPs or CIDR ranges
//   - Check if RemoteAddr matches trusted proxy
//   - Only then trust X-Forwarded-For headers
//
// Example Implementation:
//   trustedProxies := []string{"10.0.0.0/8", "172.16.0.0/12"}
//
//   func TrustedProxyMiddleware(trustedProxies []string) MiddlewareFunc {
//       return func(next HandlerFunc) HandlerFunc {
//           return func(c *Context) error {
//               remoteIP := c.RemoteIP()
//               isTrusted := false
//
//               for _, proxy := range trustedProxies {
//                   _, network, _ := net.ParseCIDR(proxy)
//                   if network.Contains(remoteIP) {
//                       isTrusted = true
//                       break
//                   }
//               }
//
//               if isTrusted {
//                   // Trust forwarded headers
//                   SetClientIPForContext(c)
//               } else {
//                   // Don't trust headers, use RemoteAddr
//                   c.SetUserValue("clientip", remoteIP.String())
//               }
//
//               return next(c)
//           }
//       }
//   }
