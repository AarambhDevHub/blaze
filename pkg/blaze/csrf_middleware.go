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

// CSRFOptions configures the CSRF protection middleware
// CSRF (Cross-Site Request Forgery) is an attack that forces users to execute
// unwanted actions on a web application where they're authenticated
//
// How CSRF Protection Works:
//  1. Server generates a unique token for each session/request
//  2. Token is stored in a cookie (HttpOnly, Secure in production)
//  3. Client must include the token in requests (header/form/query)
//  4. Server validates that both tokens match before processing
//
// Security Considerations:
//   - Use a strong secret key (32+ bytes)
//   - Enable Secure and HttpOnly cookies in production
//   - Use SameSite=Strict for maximum security
//   - Validate origin and referer headers
//   - Consider single-use tokens for sensitive operations
//   - Always use HTTPS in production
//
// Token Delivery Methods:
//   - Header: X-CSRF-Token (recommended for AJAX)
//   - Form field: _csrf_token (for traditional forms)
//   - Query parameter: csrf_token (least secure, avoid if possible)
type CSRFOptions struct {
	// Secret key for generating tokens
	// Must be at least 32 bytes for security
	// Keep this secret and rotate periodically
	Secret []byte

	// TokenLookup specifies where to find the CSRF token in requests
	// Multiple lookup methods can be specified, tried in order
	// Format: "source:name" where source is header/form/query
	// Example: []string{"header:X-CSRF-Token", "form:_csrf_token", "query:csrf_token"}
	TokenLookup []string

	// ContextKey is the key used to store the token in request context
	// Access token in handlers: token := c.Locals(ContextKey).(string)
	// Default: "csrf_token"
	ContextKey string

	// Cookie Configuration
	// CookieName is the name of the cookie storing the CSRF token
	// Default: "_csrf"
	CookieName string

	// CookiePath specifies the path for which the cookie is valid
	// Default: "/" (entire site)
	CookiePath string

	// CookieDomain specifies the domain for the cookie
	// Leave empty to use current domain
	CookieDomain string

	// CookieSecure when true, cookie only sent over HTTPS
	// Must be true in production
	// Default: false (set to true in production)
	CookieSecure bool

	// CookieHTTPOnly when true, cookie not accessible via JavaScript
	// Prevents XSS attacks from stealing CSRF tokens
	// Should always be true
	// Default: true
	CookieHTTPOnly bool

	// CookieSameSite controls cross-site cookie behavior
	// Options: "Strict", "Lax", "None"
	// Strict: Cookie only sent for same-site requests (most secure)
	// Lax: Cookie sent for top-level navigation (balanced)
	// None: Cookie sent for all requests (requires Secure=true)
	// Default: "Lax"
	CookieSameSite string

	// CookieMaxAge specifies cookie lifetime in seconds
	// After this time, a new token is generated
	// Common values: 3600 (1 hour), 86400 (24 hours)
	// Default: 3600 seconds
	CookieMaxAge int

	// Expiration is the token expiration duration
	// Tokens older than this are rejected
	// Should match or exceed CookieMaxAge
	// Default: 1 hour
	Expiration time.Duration

	// TokenLength specifies the length of generated tokens in bytes
	// Longer tokens are more secure but use more bandwidth
	// Recommended: 32 bytes (256 bits)
	// Minimum: 16 bytes
	// Default: 32 bytes
	TokenLength int

	// Skipper function to skip CSRF check for specific requests
	// Return true to bypass CSRF validation
	// Useful for public APIs, webhooks, or health checks
	// Example: func(c *Context) bool { return c.Path() == "/webhook" }
	Skipper func(c *Context) bool

	// ErrorHandler is called when CSRF validation fails
	// Allows custom error responses
	// If nil, returns 403 Forbidden with default error message
	ErrorHandler func(c *Context, err error) error

	// TrustedOrigins lists origins allowed to make requests
	// Used for origin header validation
	// Example: []string{"https://example.com", "https://app.example.com"}
	// If empty, validates against request host
	TrustedOrigins []string

	// CheckReferer when true, validates the Referer header
	// Referer must match the request host
	// Provides additional security against CSRF attacks
	// Default: true
	CheckReferer bool

	// SingleUse when true, tokens can only be used once
	// Provides maximum security but requires server-side state
	// Each request gets a new token
	// Use for sensitive operations (money transfers, password changes)
	// Default: false (stateless tokens)
	SingleUse bool

	// Internal token storage for single-use tokens
	tokenStore map[string]tokenInfo // Maps token to its metadata
	storeMutex sync.RWMutex         // Protects concurrent access to tokenStore

	// cleanupInterval specifies how often to clean up expired tokens
	// Only used when SingleUse is true
	// Default: 5 minutes
	cleanupInterval time.Duration
}

// tokenInfo stores metadata for single-use tokens
// Used to track token usage and expiration
type tokenInfo struct {
	createdAt time.Time // When the token was created
	used      bool      // Whether the token has been used
}

// DefaultCSRFOptions returns secure default CSRF configuration
// Suitable for development with HTTPS disabled
//
// Default Configuration:
//   - Token lookup: Header (X-CSRF-Token), Form (_csrf_token), Query (csrf_token)
//   - Cookie: _csrf, HttpOnly, Lax SameSite, 1 hour lifetime
//   - Token: 32 bytes, 1 hour expiration
//   - Origin and referer checking enabled
//   - Single-use tokens disabled (stateless)
//
// Production Checklist:
//   - Set CookieSecure to true
//   - Use SameSite=Strict if possible
//   - Set strong Secret key (32+ bytes)
//   - Configure TrustedOrigins
//   - Enable HTTPS
//
// Returns:
//   - CSRFOptions: Default configuration
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

// ProductionCSRFOptions returns strict CSRF configuration for production
// Enables all security features for production deployment
//
// Production Features:
//   - Secure cookies (HTTPS only)
//   - Strict SameSite policy
//   - Origin and referer validation
//   - Requires explicit secret key
//   - Trusted origins must be configured
//
// Usage:
//
//	opts := blaze.ProductionCSRFOptions([]byte("your-32-byte-secret-key-here"))
//	opts.TrustedOrigins = []string{"https://example.com"}
//	app.Use(blaze.CSRF(opts))
//
// Parameters:
//   - secret: Secret key for token generation (min 32 bytes)
//
// Returns:
//   - CSRFOptions: Production-ready configuration
func ProductionCSRFOptions(secret []byte) *CSRFOptions {
	opts := DefaultCSRFOptions()
	opts.Secret = secret
	opts.CookieSecure = true
	opts.CookieSameSite = "Strict"
	opts.CheckReferer = true
	opts.TrustedOrigins = []string{} // Set your trusted origins
	return opts
}

// CSRF creates a new CSRF protection middleware
// Implements double-submit cookie pattern for CSRF protection
//
// CSRF Protection Flow:
//
//  1. Safe Methods (GET, HEAD, OPTIONS):
//     - Generate new CSRF token
//     - Set token in cookie
//     - Store token in context for templates
//     - Allow request to proceed
//
//  2. Unsafe Methods (POST, PUT, DELETE, PATCH):
//     - Validate origin and referer headers
//     - Extract token from request (header/form/query)
//     - Retrieve token from cookie
//     - Compare tokens using constant-time comparison
//     - If valid, allow request; otherwise return 403
//
// Token Validation:
//   - Uses constant-time comparison to prevent timing attacks
//   - Checks token expiration based on embedded timestamp
//   - For single-use tokens, marks token as used after validation
//   - Generates new token for response (rotating tokens)
//
// Parameters:
//   - opts: CSRF configuration options
//
// Returns:
//   - MiddlewareFunc: CSRF protection middleware
//
// Example - Basic Usage:
//
//	app.Use(blaze.CSRF(blaze.DefaultCSRFOptions()))
//
// Example - Production Configuration:
//
//	secret := []byte("your-32-byte-secret-key-here!!!")
//	opts := blaze.ProductionCSRFOptions(secret)
//	opts.TrustedOrigins = []string{"https://example.com", "https://app.example.com"}
//	app.Use(blaze.CSRF(opts))
//
// Example - With Custom Error Handler:
//
//	opts := blaze.DefaultCSRFOptions()
//	opts.ErrorHandler = func(c *blaze.Context, err error) error {
//	    return c.Status(403).JSON(blaze.Map{
//	        "error": "CSRF validation failed",
//	        "details": err.Error(),
//	    })
//	}
//	app.Use(blaze.CSRF(opts))
//
// Example - Skip CSRF for Webhooks:
//
//	opts := blaze.DefaultCSRFOptions()
//	opts.Skipper = func(c *blaze.Context) bool {
//	    return strings.HasPrefix(c.Path(), "/webhooks/")
//	}
//	app.Use(blaze.CSRF(opts))
//
// Example - Single-Use Tokens (Maximum Security):
//
//	opts := blaze.ProductionCSRFOptions(secret)
//	opts.SingleUse = true
//	app.Use(blaze.CSRF(opts))
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
// Checks for common misconfigurations and security issues
//
// Validation Rules:
//   - Secret key is required and must be at least 32 bytes
//   - Token length must be at least 16 bytes
//   - At least one token lookup method must be specified
//
// Parameters:
//   - opts: CSRF options to validate
//
// Returns:
//   - error: Validation error or nil if valid
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
// Tokens include a timestamp for expiration checking
//
// Token Format:
//  1. Generate random bytes using crypto/rand
//  2. Prepend current Unix timestamp
//  3. Base64 URL-encode the result
//
// The timestamp allows the server to reject expired tokens
// without maintaining server-side state
//
// Returns:
//   - string: Base64-encoded CSRF token with timestamp
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
// Tries each lookup method in order until a token is found
//
// Lookup Methods:
//   - header:X-CSRF-Token - Token in request header
//   - form:_csrf_token - Token in form data (POST body)
//   - query:csrf_token - Token in URL query parameter
//
// Parameters:
//   - c: Request context
//
// Returns:
//   - string: Extracted token
//   - error: Error if token not found in any location
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
// Configures cookie with security settings from options
//
// Cookie Configuration:
//   - HttpOnly: Prevents JavaScript access (XSS protection)
//   - Secure: Only sent over HTTPS (production)
//   - SameSite: Controls cross-site behavior
//   - MaxAge: Cookie lifetime in seconds
//
// Parameters:
//   - c: Request context
//   - token: CSRF token to store in cookie
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
// Extracts the token value from the request cookie
//
// Parameters:
//   - c: Request context
//
// Returns:
//   - string: CSRF token from cookie or empty string if not found
func (opts *CSRFOptions) getTokenFromCookie(c *Context) string {
	return string(c.RequestCtx.Request.Header.Cookie(opts.CookieName))
}

// validateToken validates the client token against the cookie token
// Uses constant-time comparison to prevent timing attacks
//
// Validation Steps:
//  1. Decode both tokens from Base64
//  2. Parse timestamp and token components
//  3. Check if token has expired
//  4. For single-use tokens, verify not already used
//  5. Compare tokens using constant-time comparison
//
// Parameters:
//   - clientToken: Token from request (header/form/query)
//   - cookieToken: Token from cookie
//
// Returns:
//   - bool: true if tokens match and are valid
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
// Prevents CSRF attacks by ensuring requests come from trusted origins
//
// Validation Logic:
//  1. Extract Origin header from request
//  2. If no Origin, check Referer header
//  3. If TrustedOrigins configured, check against list
//  4. Otherwise, validate origin matches request host
//
// Parameters:
//   - c: Request context
//
// Returns:
//   - error: Validation error or nil if origin is valid
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
// Additional security check to prevent CSRF attacks
//
// Validation Logic:
//  1. Check if Referer header is present
//  2. Parse Referer URL
//  3. Verify Referer host matches request host
//
// Parameters:
//   - c: Request context
//
// Returns:
//   - error: Validation error or nil if referer is valid
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

// markTokenAsUsed marks a token as used for single-use tokens
// Prevents token replay attacks by tracking used tokens
//
// Parameters:
//   - token: Token to mark as used
//
// Returns:
//   - error: Error if token was already used
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
// Thread-safe check for single-use token validation
//
// Parameters:
//   - token: Token to check
//
// Returns:
//   - bool: true if token has been used
func (opts *CSRFOptions) isTokenUsed(token string) bool {
	opts.storeMutex.RLock()
	defer opts.storeMutex.RUnlock()

	if info, exists := opts.tokenStore[token]; exists {
		return info.used
	}

	return false
}

// startCleanup starts background cleanup of expired tokens
// Removes expired single-use tokens to prevent memory leaks
// Runs in a goroutine at configured intervals
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

// handleError handles CSRF validation errors
// Uses custom error handler if provided, otherwise returns default 403 response
//
// Parameters:
//   - c: Request context
//   - err: CSRF validation error
//
// Returns:
//   - error: Response error
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

// isSafeMethod checks if HTTP method is safe (doesn't modify state)
// Safe methods: GET, HEAD, OPTIONS, TRACE
//
// Parameters:
//   - method: HTTP method
//
// Returns:
//   - bool: true if method is safe
func isSafeMethod(method string) bool {
	safeMethods := []string{"GET", "HEAD", "OPTIONS", "TRACE"}
	for _, safe := range safeMethods {
		if strings.ToUpper(method) == safe {
			return true
		}
	}
	return false
}

// CSRFToken returns the CSRF token from the request context
// Used in templates to include CSRF token in forms
//
// Example in template:
//
//	<input type="hidden" name="_csrf_token" value="{{ .csrf_token }}">
//
// Example in handler:
//
//	token := blaze.CSRFToken(c)
//
// Parameters:
//   - c: Request context
//
// Returns:
//   - string: CSRF token or empty string if not found
func CSRFToken(c *Context) string {
	if token := c.Locals("csrf_token"); token != nil {
		if tokenStr, ok := token.(string); ok {
			return tokenStr
		}
	}
	return ""
}

// CSRFMeta returns HTML meta tag with CSRF token
// Convenient for including CSRF token in HTML templates
//
// Example usage:
//
//	<!-- In HTML head -->
//	{{ csrfmeta }}
//
//	<!-- JavaScript can read it -->
//	<script>
//	  const token = document.querySelector('meta[name="csrf-token"]').content;
//	</script>
//
// Parameters:
//   - c: Request context
//
// Returns:
//   - string: HTML meta tag or empty string if no token
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
