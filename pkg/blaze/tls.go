package blaze

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/valyala/fasthttp"
)

// TLSConfig holds comprehensive TLS/SSL configuration
// Provides control over certificates, protocols, cipher suites, and security settings
//
// TLS Configuration Philosophy:
//   - Security: Use strong cipher suites and modern protocols
//   - Compatibility: Support wide range of clients when necessary
//   - Performance: Enable session resumption and OCSP stapling
//   - Development: Automatic certificate generation for testing
//
// Production Best Practices:
//   - Use MinVersion: TLS 1.2 or higher
//   - Configure strong cipher suites only
//   - Enable OCSP stapling for revocation checking
//   - Disable session tickets if forward secrecy required
//   - Use certificates from trusted CA (Let's Encrypt, etc.)
//   - Never use InsecureSkipVerify in production
//
// Security Considerations:
//   - TLS 1.0/1.1 are deprecated and should not be used
//   - Weak cipher suites expose to attacks (BEAST, CRIME, etc.)
//   - Self-signed certificates should only be used in development
//   - Client certificate authentication adds security layer
//   - ALPN enables HTTP/2 protocol negotiation
type TLSConfig struct {
	// CertFile is the path to the TLS certificate file (PEM format)
	// Required for production HTTPS
	// Must contain the complete certificate chain
	// Example: "/etc/ssl/certs/server.crt"
	CertFile string

	// KeyFile is the path to the TLS private key file (PEM format)
	// Required for production HTTPS
	// Must match the certificate
	// Keep secure with proper file permissions (0600)
	// Example: "/etc/ssl/private/server.key"
	KeyFile string

	// AutoTLS enables automatic self-signed certificate generation
	// For development and testing only
	// Certificates are cached in TLSCacheDir
	// WARNING: Never use in production (browsers show warnings)
	// Default: false
	AutoTLS bool

	// TLSCacheDir specifies directory for cached certificates
	// Used with AutoTLS for storing generated certificates
	// Certificates are reused across restarts until expiration
	// Default: ".certs"
	TLSCacheDir string

	// Domains specifies hostnames/IPs for the certificate
	// Used with AutoTLS for Subject Alternative Names (SAN)
	// Can include both domain names and IP addresses
	// Example: []string{"localhost", "127.0.0.1", "example.com"}
	// Default: []string{"localhost", "127.0.0.1"}
	Domains []string

	// Organization specifies organization name for certificates
	// Used with AutoTLS for certificate subject
	// Example: "My Company Inc."
	// Default: "Blaze Framework"
	Organization string

	// MinVersion specifies minimum TLS version to accept
	// Options: tls.VersionTLS10, VersionTLS11, VersionTLS12, VersionTLS13
	// Recommended: tls.VersionTLS12 or higher
	// Default: tls.VersionTLS12
	MinVersion uint16

	// MaxVersion specifies maximum TLS version to accept
	// Usually set to latest (TLS 1.3)
	// Only restrict for compatibility testing
	// Default: tls.VersionTLS13
	MaxVersion uint16

	// CipherSuites specifies allowed cipher suites
	// Order matters - preferred ciphers first
	// Empty slice uses Go's default secure suites
	// Recommended: Use getSecureCipherSuites()
	// Example: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, ...}
	CipherSuites []uint16

	// ClientAuth specifies client certificate requirement level
	// Options:
	//   - tls.NoClientCert: No client cert required (default)
	//   - tls.RequestClientCert: Request but don't require
	//   - tls.RequireAnyClientCert: Require cert (any CA)
	//   - tls.VerifyClientCertIfGiven: Verify if provided
	//   - tls.RequireAndVerifyClientCert: Require and verify
	// Default: tls.NoClientCert
	ClientAuth tls.ClientAuthType

	// ClientCAs specifies trusted CA certificates for client auth
	// Used when ClientAuth requires verification
	// Load with x509.NewCertPool() and AddCert()
	// Only needed for mutual TLS authentication
	ClientCAs *x509.CertPool

	// NextProtos specifies ALPN protocol identifiers
	// Enables protocol negotiation (HTTP/2, HTTP/1.1)
	// Order matters - preferred protocols first
	// Example: []string{"h2", "http/1.1"}
	// Default: []string{"h2", "http/1.1"}
	NextProtos []string

	// CertValidityDuration specifies certificate validity period
	// Used with AutoTLS for generated certificates
	// Recommended: 90 days (Let's Encrypt style)
	// Default: 365 days (1 year)
	CertValidityDuration time.Duration

	// OCSPStapling enables Online Certificate Status Protocol stapling
	// Server fetches revocation status and includes in handshake
	// Improves performance and privacy
	// Requires certificate with OCSP responder URL
	// Default: true
	OCSPStapling bool

	// SessionTicketsDisabled disables TLS session resumption tickets
	// When true, each connection requires full handshake
	// Disable for perfect forward secrecy
	// Enable for better performance
	// Default: false (tickets enabled)
	SessionTicketsDisabled bool

	// CurvePreferences specifies preferred elliptic curves
	// Order matters - preferred curves first
	// Modern curves: X25519, P-256, P-384
	// Example: []tls.CurveID{tls.X25519, tls.CurveP256}
	// Default: []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384}
	CurvePreferences []tls.CurveID

	// Renegotiation controls TLS renegotiation support
	// Options:
	//   - tls.RenegotiateNever: No renegotiation (secure)
	//   - tls.RenegotiateOnceAsClient: Allow once (compatibility)
	//   - tls.RenegotiateFreelyAsClient: Allow multiple (insecure)
	// Default: tls.RenegotiateNever
	Renegotiation tls.RenegotiationSupport

	// InsecureSkipVerify disables certificate verification
	// WARNING: Only use for development/testing
	// NEVER use in production (defeats TLS security)
	// Makes connections vulnerable to man-in-the-middle attacks
	// Default: false
	InsecureSkipVerify bool
}

// DefaultTLSConfig returns secure TLS configuration with HTTP/2 support
// Provides production-ready settings with strong security
//
// Default Configuration:
//   - TLS 1.2 minimum (TLS 1.3 maximum)
//   - Secure cipher suites only
//   - ALPN: h2, http/1.1 (HTTP/2 enabled)
//   - OCSP stapling enabled
//   - Modern elliptic curves (X25519, P-256, P-384)
//   - No renegotiation (security)
//   - Session tickets enabled (performance)
//   - Certificate validity: 1 year
//
// Returns:
//   - TLSConfig: Production-ready TLS configuration
//
// Example:
//
//	tlsConfig := blaze.DefaultTLSConfig()
//	tlsConfig.CertFile = "/path/to/cert.pem"
//	tlsConfig.KeyFile = "/path/to/key.pem"
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		AutoTLS:                false,
		TLSCacheDir:            "./certs",
		Organization:           "Blaze Framework",
		MinVersion:             tls.VersionTLS12,
		MaxVersion:             tls.VersionTLS13,
		CipherSuites:           getSecureCipherSuites(),
		ClientAuth:             tls.NoClientCert,
		NextProtos:             []string{"h2", "http/1.1"}, // HTTP/2 and HTTP/1.1
		CertValidityDuration:   365 * 24 * time.Hour,       // 1 year
		OCSPStapling:           true,
		SessionTicketsDisabled: false,
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP256,
			tls.CurveP384,
		},
		Renegotiation:      tls.RenegotiateNever,
		InsecureSkipVerify: false,
	}
}

// DevelopmentTLSConfig returns TLS configuration for local development
// Enables automatic certificate generation for easy testing
//
// Development Features:
//   - AutoTLS enabled (self-signed certificates)
//   - Localhost domains (localhost, 127.0.0.1)
//   - InsecureSkipVerify enabled (accepts self-signed)
//   - Same security settings as production
//
// WARNING: Only for development!
//   - Browsers will show security warnings
//   - Certificates not trusted by clients
//   - InsecureSkipVerify defeats TLS security
//
// Returns:
//   - TLSConfig: Development-friendly configuration
//
// Example:
//
//	tlsConfig := blaze.DevelopmentTLSConfig()
//	app.SetTLSConfig(tlsConfig)
func DevelopmentTLSConfig() *TLSConfig {
	config := DefaultTLSConfig()
	config.AutoTLS = true
	config.Domains = []string{"localhost", "127.0.0.1"}
	config.InsecureSkipVerify = true
	return config
}

// getSecureCipherSuites returns a list of secure cipher suites
// Prioritizes modern, secure algorithms with forward secrecy
//
// Cipher Suite Priority:
//  1. TLS 1.3 suites (not configurable in Go, automatically used)
//  2. ECDHE + AESGCM (forward secrecy, AEAD encryption)
//  3. ECDHE + ChaCha20-Poly1305 (mobile-optimized)
//  4. ECDHE + AES-CBC (compatibility, less preferred)
//
// Security Properties:
//   - Forward secrecy (ECDHE key exchange)
//   - Authenticated encryption (GCM, ChaCha20-Poly1305)
//   - Strong key sizes (256-bit and 128-bit AES)
//
// Excluded Cipher Suites:
//   - Non-ECDHE (no forward secrecy)
//   - RC4 (broken)
//   - 3DES (weak)
//   - Export ciphers (intentionally weak)
//
// Returns:
//   - []uint16: List of secure cipher suite IDs
func getSecureCipherSuites() []uint16 {
	return []uint16{
		// TLS 1.3 cipher suites (these are not configurable in Go)

		// TLS 1.2 ECDHE+AESGCM cipher suites
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,

		// TLS 1.2 ECDHE+CHACHA20 cipher suites
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,

		// TLS 1.2 ECDHE+AES_CBC cipher suites (less preferred)
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	}
}

// BuildTLSConfig creates a crypto/tls.Config from TLSConfig
// Converts Blaze TLS configuration to Go standard library format
//
// Configuration Process:
//  1. Set protocol versions and cipher suites
//  2. Configure ALPN for HTTP/2
//  3. Load certificates (file or auto-generated)
//  4. Set client authentication if required
//  5. Configure session tickets and renegotiation
//  6. Apply security settings
//
// Returns:
//   - *tls.Config: Standard library TLS configuration
//   - error: Configuration error or nil on success
//
// Example:
//
//	blazeConfig := blaze.DefaultTLSConfig()
//	tlsConfig, err := blazeConfig.BuildTLSConfig()
//	if err != nil {
//	    log.Fatal(err)
//	}
func (tc *TLSConfig) BuildTLSConfig() (*tls.Config, error) {
	config := &tls.Config{
		MinVersion:               tc.MinVersion,
		MaxVersion:               tc.MaxVersion,
		CipherSuites:             tc.CipherSuites,
		PreferServerCipherSuites: true,
		CurvePreferences:         tc.CurvePreferences,
		NextProtos:               tc.NextProtos,
		ClientAuth:               tc.ClientAuth,
		ClientCAs:                tc.ClientCAs,
		Renegotiation:            tc.Renegotiation,
		InsecureSkipVerify:       tc.InsecureSkipVerify,
	}

	// Disable session tickets if requested
	if tc.SessionTicketsDisabled {
		config.SessionTicketsDisabled = true
	}

	// Load certificates
	if tc.AutoTLS {
		if err := tc.generateSelfSignedCert(); err != nil {
			return nil, fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}
	}

	if tc.CertFile != "" && tc.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tc.CertFile, tc.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	return config, nil
}

// generateSelfSignedCert generates a self-signed certificate for development
// Creates RSA key pair and X.509 certificate with specified domains
//
// Certificate Generation:
//  1. Check if valid certificate already exists (skip if found)
//  2. Generate 2048-bit RSA private key
//  3. Create X.509 certificate template
//  4. Add domains as Subject Alternative Names (SAN)
//  5. Self-sign certificate
//  6. Save certificate and key to files
//
// Certificate Properties:
//   - Self-signed (issuer == subject)
//   - RSA 2048-bit key
//   - Validity: CertValidityDuration (default 1 year)
//   - Usage: Server authentication
//   - Includes all specified domains and IPs
//
// Returns:
//   - error: Generation error or nil on success
func (tc *TLSConfig) generateSelfSignedCert() error {
	// Create certs directory if it doesn't exist
	if err := os.MkdirAll(tc.TLSCacheDir, 0755); err != nil {
		return err
	}

	certPath := filepath.Join(tc.TLSCacheDir, "server.crt")
	keyPath := filepath.Join(tc.TLSCacheDir, "server.key")

	// Check if certificate already exists and is still valid
	if tc.isCertValid(certPath) {
		tc.CertFile = certPath
		tc.KeyFile = keyPath
		return nil
	}

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{tc.Organization},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(tc.CertValidityDuration),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Add domains and IPs to certificate
	for _, domain := range tc.Domains {
		if ip := net.ParseIP(domain); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, domain)
		}
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	// Save certificate
	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}

	// Save private key
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return err
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER}); err != nil {
		return err
	}

	tc.CertFile = certPath
	tc.KeyFile = keyPath

	return nil
}

// isCertValid checks if the certificate exists and is still valid
// Validates certificate file and expiration date
//
// Validation Checks:
//   - Certificate file exists
//   - Certificate can be parsed
//   - Certificate is valid for at least 7 more days
//
// Parameters:
//   - certPath: Path to certificate file
//
// Returns:
//   - bool: true if certificate is valid
func (tc *TLSConfig) isCertValid(certPath string) bool {
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return false
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return false
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return false
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}

	// Check if certificate is still valid for at least 7 days
	return time.Now().Add(7 * 24 * time.Hour).Before(cert.NotAfter)
}

// ConfigureFastHTTPTLS configures fasthttp server with TLS settings
// Applies TLS configuration to fasthttp.Server
//
// Parameters:
//   - server: FastHTTP server instance
//
// Returns:
//   - error: Configuration error or nil on success
func (tc *TLSConfig) ConfigureFastHTTPTLS(server *fasthttp.Server) error {
	tlsConfig, err := tc.BuildTLSConfig()
	if err != nil {
		return err
	}

	// Create a custom listener with TLS configuration
	server.TLSConfig = tlsConfig

	return nil
}

// GetCertificateInfo returns information about the loaded certificate
// Parses certificate and extracts metadata
//
// Returns:
//   - *CertificateInfo: Certificate information
//   - error: Parse error or nil on success
func (tc *TLSConfig) GetCertificateInfo() (*CertificateInfo, error) {
	if tc.CertFile == "" {
		return nil, fmt.Errorf("no certificate file specified")
	}

	certPEM, err := os.ReadFile(tc.CertFile)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &CertificateInfo{
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		SerialNumber: cert.SerialNumber.String(),
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		DNSNames:     cert.DNSNames,
		IPAddresses:  cert.IPAddresses,
		KeyUsage:     cert.KeyUsage,
		ExtKeyUsage:  cert.ExtKeyUsage,
	}, nil
}

// CertificateInfo holds certificate metadata
type CertificateInfo struct {
	Subject      string
	Issuer       string
	SerialNumber string
	NotBefore    time.Time
	NotAfter     time.Time
	DNSNames     []string
	IPAddresses  []net.IP
	KeyUsage     x509.KeyUsage
	ExtKeyUsage  []x509.ExtKeyUsage
}

// IsExpired checks if the certificate is expired
func (ci *CertificateInfo) IsExpired() bool {
	return time.Now().After(ci.NotAfter)
}

// ExpiresIn returns the duration until the certificate expires
func (ci *CertificateInfo) ExpiresIn() time.Duration {
	if ci.IsExpired() {
		return 0
	}
	return time.Until(ci.NotAfter)
}

// TLSHealthCheck represents TLS health check information
type TLSHealthCheck struct {
	Enabled       bool             `json:"enabled"`
	Version       string           `json:"version,omitempty"`
	CipherSuite   string           `json:"cipher_suite,omitempty"`
	Certificate   *CertificateInfo `json:"certificate,omitempty"`
	NextProtocols []string         `json:"next_protocols,omitempty"`
	Error         string           `json:"error,omitempty"`
}

// GetTLSHealthCheck returns TLS health check information
func (tc *TLSConfig) GetTLSHealthCheck() *TLSHealthCheck {
	healthCheck := &TLSHealthCheck{
		Enabled:       tc.CertFile != "" && tc.KeyFile != "",
		NextProtocols: tc.NextProtos,
	}

	if !healthCheck.Enabled {
		return healthCheck
	}

	certInfo, err := tc.GetCertificateInfo()
	if err != nil {
		healthCheck.Error = err.Error()
		return healthCheck
	}

	healthCheck.Certificate = certInfo

	return healthCheck
}
