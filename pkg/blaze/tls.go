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

// TLSConfig holds TLS configuration
type TLSConfig struct {
	// Certificate and key file paths
	CertFile string
	KeyFile  string

	// Auto-generate self-signed certificate
	AutoTLS      bool
	TLSCacheDir  string
	Domains      []string
	Organization string

	// TLS version and cipher suites
	MinVersion   uint16
	MaxVersion   uint16
	CipherSuites []uint16

	// Client certificate authentication
	ClientAuth tls.ClientAuthType
	ClientCAs  *x509.CertPool

	// ALPN protocols for HTTP/2
	NextProtos []string

	// Certificate configuration
	CertValidityDuration time.Duration

	// OCSP and certificate transparency
	OCSPStapling bool

	// Session tickets
	SessionTicketsDisabled bool

	// Curves
	CurvePreferences []tls.CurveID

	// Renegotiation
	Renegotiation tls.RenegotiationSupport

	// InsecureSkipVerify for development (DO NOT use in production)
	InsecureSkipVerify bool
}

// DefaultTLSConfig returns a secure TLS configuration with HTTP/2 support
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

// DevelopmentTLSConfig returns a TLS configuration suitable for development
func DevelopmentTLSConfig() *TLSConfig {
	config := DefaultTLSConfig()
	config.AutoTLS = true
	config.Domains = []string{"localhost", "127.0.0.1"}
	config.InsecureSkipVerify = true
	return config
}

// getSecureCipherSuites returns a list of secure cipher suites
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

// BuildTLSConfig creates a *tls.Config from TLSConfig
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

// CertificateInfo holds certificate information
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
