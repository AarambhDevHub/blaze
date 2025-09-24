package blaze

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

// CSRFOptions configures the CSRF middleware
type CSRFOptions struct {
	// Secret key for generating tokens (32 bytes recommended)
	Secret []byte

	// Token lookup methods: "header:<name>", "form:<name>", "query:<name>"
	TokenLookup []string

	// Context key for storing the token
	ContextKey string

	// Cookie name for CSRF token
	CookieName string

	// Cookie settings
	CookiePath     string
	CookieDomain   string
	CookieSecure   bool
	CookieHTTPOnly bool
	CookieSameSite string
	CookieMaxAge   int

	// Token expiration duration
	Expiration time.Duration

	// Token length in bytes (32 recommended)
	TokenLength int

	// Skip CSRF check function
	Skipper func(c *Context) bool

	// Error handler
	ErrorHandler func(c *Context, err error) error

	// Trusted origins for CORS
	TrustedOrigins []string

	// Check Referer header
	CheckReferer bool

	// Single-use tokens (more secure but requires server state)
	SingleUse bool

	// Token storage for single-use tokens
	tokenStore map[string]tokenInfo
	storeMutex sync.RWMutex

	// Cleanup interval for expired tokens
	cleanupInterval time.Duration
}

// tokenInfo stores token metadata
type tokenInfo struct {
	createdAt time.Time
	used      bool
}

// Default CSRF options
func DefaultCSRFOptions() *CSRFOptions {
	return &CSRFOptions{
		TokenLookup:     []string{"header:X-CSRF-Token", "form:csrf_token", "query:csrf_token"},
		ContextKey:      "csrf_token",
		CookieName:      "_csrf",
		CookiePath:      "/",
		CookieSecure:    false, // Set to true in production with HTTPS
		CookieHTTPOnly:  true,
		CookieSameSite:  "Lax",
		CookieMaxAge:    3600, // 1 hour
		Expiration:      time.Hour,
		TokenLength:     32,
		CheckReferer:    true,
		SingleUse:       false,
		tokenStore:      make(map[string]tokenInfo),
		cleanupInterval: 5 * time.Minute,
	}
}

// Production CSRF options
func ProductionCSRFOptions(secret []byte) *CSRFOptions {
	opts := DefaultCSRFOptions()
	opts.Secret = secret
	opts.CookieSecure = true
	opts.CookieSameSite = "Strict"
	opts.CheckReferer = true
	opts.TrustedOrigins = []string{} // Set your trusted origins
	return opts
}

// CSRF creates a new CSRF middleware
func CSRF(opts *CSRFOptions) MiddlewareFunc {
	if opts == nil {
		opts = DefaultCSRFOptions()
	}

	// Validate options
	if err := validateCSRFOptions(opts); err != nil {
		panic(fmt.Sprintf("CSRF middleware configuration error: %v", err))
	}

	// Start cleanup routine for single-use tokens
	if opts.SingleUse {
		go opts.startCleanup()
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			// Skip CSRF check if skipper function returns true
			if opts.Skipper != nil && opts.Skipper(c) {
				return next(c)
			}

			// Skip for safe methods
			if isSafeMethod(c.Method()) {
				token := opts.generateToken()
				opts.setTokenCookie(c, token)
				c.SetLocals(opts.ContextKey, token)
				return next(c)
			}

			// Check origin and referer for unsafe methods
			if err := opts.checkOrigin(c); err != nil {
				return opts.handleError(c, err)
			}

			if opts.CheckReferer {
				if err := opts.checkReferer(c); err != nil {
					return opts.handleError(c, err)
				}
			}

			// Extract and validate token
			clientToken, err := opts.extractToken(c)
			if err != nil {
				return opts.handleError(c, fmt.Errorf("CSRF token not found: %v", err))
			}

			// Get token from cookie
			cookieToken := opts.getTokenFromCookie(c)
			if cookieToken == "" {
				return opts.handleError(c, fmt.Errorf("CSRF cookie not found"))
			}

			// Validate token
			if !opts.validateToken(clientToken, cookieToken) {
				return opts.handleError(c, fmt.Errorf("CSRF token mismatch"))
			}

			// Handle single-use tokens
			if opts.SingleUse {
				if err := opts.markTokenAsUsed(clientToken); err != nil {
					return opts.handleError(c, err)
				}
			}

			// Generate new token for response
			newToken := opts.generateToken()
			opts.setTokenCookie(c, newToken)
			c.SetLocals(opts.ContextKey, newToken)

			return next(c)
		}
	}
}

// validateCSRFOptions validates CSRF configuration
func validateCSRFOptions(opts *CSRFOptions) error {
	if len(opts.Secret) == 0 {
		return fmt.Errorf("secret key is required")
	}

	if len(opts.Secret) < 32 {
		return fmt.Errorf("secret key must be at least 32 bytes")
	}

	if opts.TokenLength < 16 {
		return fmt.Errorf("token length must be at least 16 bytes")
	}

	if len(opts.TokenLookup) == 0 {
		return fmt.Errorf("at least one token lookup method must be specified")
	}

	return nil
}

// generateToken creates a cryptographically secure random token
func (opts *CSRFOptions) generateToken() string {
	tokenBytes := make([]byte, opts.TokenLength)
	if _, err := io.ReadFull(rand.Reader, tokenBytes); err != nil {
		panic(fmt.Sprintf("Failed to generate CSRF token: %v", err))
	}

	// Encode with timestamp for expiration checking
	timestamp := time.Now().Unix()
	tokenWithTimestamp := fmt.Sprintf("%d:%s", timestamp, base64.RawURLEncoding.EncodeToString(tokenBytes))

	return base64.RawURLEncoding.EncodeToString([]byte(tokenWithTimestamp))
}

// extractToken extracts CSRF token from request based on lookup methods
func (opts *CSRFOptions) extractToken(c *Context) (string, error) {
	for _, lookup := range opts.TokenLookup {
		parts := strings.SplitN(lookup, ":", 2)
		if len(parts) != 2 {
			continue
		}

		method, key := parts[0], parts[1]
		var token string

		switch method {
		case "header":
			token = c.Header(key)
		case "form":
			token = string(c.RequestCtx.PostArgs().Peek(key))
		case "query":
			token = string(c.RequestCtx.QueryArgs().Peek(key))
		}

		if token != "" {
			return token, nil
		}
	}

	return "", fmt.Errorf("token not found in any lookup method")
}

// setTokenCookie sets the CSRF token cookie
func (opts *CSRFOptions) setTokenCookie(c *Context, token string) {
	cookie := &fasthttp.Cookie{}
	cookie.SetKey(opts.CookieName)
	cookie.SetValue(token)
	cookie.SetPath(opts.CookiePath)
	cookie.SetDomain(opts.CookieDomain)
	cookie.SetSecure(opts.CookieSecure)
	cookie.SetHTTPOnly(opts.CookieHTTPOnly)
	cookie.SetMaxAge(opts.CookieMaxAge)

	switch strings.ToLower(opts.CookieSameSite) {
	case "strict":
		cookie.SetSameSite(fasthttp.CookieSameSiteStrictMode)
	case "lax":
		cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	case "none":
		cookie.SetSameSite(fasthttp.CookieSameSiteNoneMode)
	default:
		cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	}

	c.RequestCtx.Response.Header.SetCookie(cookie)
}

// getTokenFromCookie retrieves CSRF token from cookie
func (opts *CSRFOptions) getTokenFromCookie(c *Context) string {
	return string(c.RequestCtx.Request.Header.Cookie(opts.CookieName))
}

// validateToken validates the client token against the cookie token
func (opts *CSRFOptions) validateToken(clientToken, cookieToken string) bool {
	if clientToken == "" || cookieToken == "" {
		return false
	}

	// Decode tokens
	clientDecoded, err := base64.RawURLEncoding.DecodeString(clientToken)
	if err != nil {
		return false
	}

	cookieDecoded, err := base64.RawURLEncoding.DecodeString(cookieToken)
	if err != nil {
		return false
	}

	// Check if token is expired
	clientParts := strings.SplitN(string(clientDecoded), ":", 2)
	cookieParts := strings.SplitN(string(cookieDecoded), ":", 2)

	if len(clientParts) != 2 || len(cookieParts) != 2 {
		return false
	}

	// For single-use tokens, check if already used
	if opts.SingleUse {
		if opts.isTokenUsed(clientToken) {
			return false
		}
	}

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(clientDecoded, cookieDecoded) == 1
}

// checkOrigin validates the request origin
func (opts *CSRFOptions) checkOrigin(c *Context) error {
	origin := c.Header("Origin")
	if origin == "" {
		// If no Origin header, check Referer
		referer := c.Header("Referer")
		if referer != "" {
			if refererURL, err := url.Parse(referer); err == nil {
				origin = fmt.Sprintf("%s://%s", refererURL.Scheme, refererURL.Host)
			}
		}
	}

	if origin == "" {
		return fmt.Errorf("unable to verify request origin")
	}

	// Check if origin is in trusted list
	if len(opts.TrustedOrigins) > 0 {
		for _, trusted := range opts.TrustedOrigins {
			if origin == trusted {
				return nil
			}
		}
		return fmt.Errorf("origin %s is not trusted", origin)
	}

	// If no trusted origins specified, check against request host
	requestHost := string(c.RequestCtx.Host())
	if !strings.Contains(origin, requestHost) {
		return fmt.Errorf("origin %s does not match request host", origin)
	}

	return nil
}

// checkReferer validates the referer header
func (opts *CSRFOptions) checkReferer(c *Context) error {
	referer := c.Header("Referer")
	if referer == "" {
		return fmt.Errorf("referer header is missing")
	}

	refererURL, err := url.Parse(referer)
	if err != nil {
		return fmt.Errorf("invalid referer URL: %v", err)
	}

	requestHost := string(c.RequestCtx.Host())
	if refererURL.Host != requestHost {
		return fmt.Errorf("referer host %s does not match request host %s", refererURL.Host, requestHost)
	}

	return nil
}

// markTokenAsUsed marks a token as used (for single-use tokens)
func (opts *CSRFOptions) markTokenAsUsed(token string) error {
	opts.storeMutex.Lock()
	defer opts.storeMutex.Unlock()

	if info, exists := opts.tokenStore[token]; exists {
		if info.used {
			return fmt.Errorf("token has already been used")
		}
		info.used = true
		opts.tokenStore[token] = info
	} else {
		opts.tokenStore[token] = tokenInfo{
			createdAt: time.Now(),
			used:      true,
		}
	}

	return nil
}

// isTokenUsed checks if a token has been used
func (opts *CSRFOptions) isTokenUsed(token string) bool {
	opts.storeMutex.RLock()
	defer opts.storeMutex.RUnlock()

	if info, exists := opts.tokenStore[token]; exists {
		return info.used
	}

	return false
}

// startCleanup starts the cleanup routine for expired tokens
func (opts *CSRFOptions) startCleanup() {
	ticker := time.NewTicker(opts.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		opts.cleanupExpiredTokens()
	}
}

// cleanupExpiredTokens removes expired tokens from storage
func (opts *CSRFOptions) cleanupExpiredTokens() {
	opts.storeMutex.Lock()
	defer opts.storeMutex.Unlock()

	now := time.Now()
	for token, info := range opts.tokenStore {
		if now.Sub(info.createdAt) > opts.Expiration {
			delete(opts.tokenStore, token)
		}
	}
}

// handleError handles CSRF errors
func (opts *CSRFOptions) handleError(c *Context, err error) error {
	if opts.ErrorHandler != nil {
		return opts.ErrorHandler(c, err)
	}

	// Default error response
	return c.Status(403).JSON(Map{
		"success": false,
		"error":   "CSRF validation failed",
		"detail":  err.Error(),
	})
}

// isSafeMethod checks if the HTTP method is considered safe
func isSafeMethod(method string) bool {
	safeMethods := []string{"GET", "HEAD", "OPTIONS", "TRACE"}
	for _, safe := range safeMethods {
		if strings.ToUpper(method) == safe {
			return true
		}
	}
	return false
}

// CSRFToken extracts the CSRF token from context
func CSRFToken(c *Context) string {
	if token := c.Locals("csrf_token"); token != nil {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}
	return ""
}

// CSRFTokenHeader returns the CSRF token as a header value
func CSRFTokenHeader(c *Context) string {
	return CSRFToken(c)
}

// CSRFTokenHTML returns HTML input field with CSRF token
func CSRFTokenHTML(c *Context) string {
	token := CSRFToken(c)
	if token == "" {
		return ""
	}
	return fmt.Sprintf(`<input type="hidden" name="csrf_token" value="%s" />`, token)
}

// CSRFMeta returns HTML meta tag with CSRF token
func CSRFMeta(c *Context) string {
	token := CSRFToken(c)
	if token == "" {
		return ""
	}
	return fmt.Sprintf(`<meta name="csrf-token" content="%s" />`, token)
}
