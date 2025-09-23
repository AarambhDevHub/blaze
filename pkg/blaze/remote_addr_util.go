package blaze

import (
	"net"
	"strings"

	"github.com/valyala/fasthttp"
)

// SetRemoteAddr sets the remote address for a fasthttp request
// This is a utility function to handle the correct setting of remote addresses
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
func SetClientIPForContext(c *Context) {
	SetClientIP(c.RequestCtx)
}

// IPMiddleware is middleware that extracts and stores client IP information
func IPMiddleware() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			SetClientIPForContext(c)
			return next(c)
		}
	}
}
