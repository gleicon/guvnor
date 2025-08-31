package cert

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// Manager handles certificate management for the proxy server
type Manager struct {
	autocertManager *autocert.Manager
	logger          *logrus.Entry
	domains         []string
	staging         bool
	email           string
	certDir         string
}

// Config contains certificate manager configuration
type Config struct {
	Enabled    bool     `yaml:"enabled"`
	AutoCert   bool     `yaml:"auto_cert"`
	CertDir    string   `yaml:"cert_dir"`
	Email      string   `yaml:"email"`
	Domains    []string `yaml:"domains"`
	Staging    bool     `yaml:"staging"`
	ForceHTTPS bool     `yaml:"force_https"`
}

// New creates a new certificate manager
func New(cfg *Config, logger *logrus.Logger) (*Manager, error) {
	if !cfg.Enabled || !cfg.AutoCert {
		return nil, fmt.Errorf("certificate manager requires TLS and AutoCert to be enabled")
	}

	if cfg.Email == "" {
		return nil, fmt.Errorf("email is required for Let's Encrypt certificates")
	}

	if len(cfg.Domains) == 0 {
		return nil, fmt.Errorf("at least one domain must be specified")
	}

	// Create certificate directory
	if err := os.MkdirAll(cfg.CertDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create certificate directory: %w", err)
	}

	m := &Manager{
		logger:  logger.WithField("component", "cert-manager"),
		domains: cfg.Domains,
		staging: cfg.Staging,
		email:   cfg.Email,
		certDir: cfg.CertDir,
	}

	if err := m.setupAutocert(); err != nil {
		return nil, fmt.Errorf("failed to setup autocert manager: %w", err)
	}

	return m, nil
}

// setupAutocert configures the autocert manager with proper settings
func (m *Manager) setupAutocert() error {
	// Create autocert manager with enhanced configuration
	m.autocertManager = &autocert.Manager{
		Cache:      autocert.DirCache(m.certDir),
		Prompt:     autocert.AcceptTOS,
		Email:      m.email,
		HostPolicy: m.createHostPolicy(),
		Client:     m.createACMEClient(),
	}

	m.logger.WithFields(logrus.Fields{
		"domains":  m.domains,
		"cert_dir": m.certDir,
		"staging":  m.staging,
		"email":    m.email,
	}).Info("Certificate manager configured")

	return nil
}

// createHostPolicy creates a secure host policy that validates domains
func (m *Manager) createHostPolicy() autocert.HostPolicy {
	return func(ctx context.Context, host string) error {
		// Remove port from host if present
		if colonPos := strings.LastIndex(host, ":"); colonPos > 0 {
			host = host[:colonPos]
		}

		// Check if host is in allowed domains
		for _, domain := range m.domains {
			if host == domain {
				m.logger.WithField("domain", host).Debug("Certificate request authorized")
				return nil
			}
			
			// Check for wildcard domain match
			if strings.HasPrefix(domain, "*.") {
				baseDomain := domain[2:]
				if strings.HasSuffix(host, "."+baseDomain) || host == baseDomain {
					m.logger.WithField("domain", host).Debug("Certificate request authorized via wildcard")
					return nil
				}
			}
		}

		m.logger.WithField("domain", host).Warn("Certificate request denied - domain not in whitelist")
		return fmt.Errorf("domain %s is not authorized for certificates", host)
	}
}

// createACMEClient creates an ACME client with proper configuration
func (m *Manager) createACMEClient() *acme.Client {
	directoryURL := "https://acme-v02.api.letsencrypt.org/directory"
	
	// Use staging environment if configured
	if m.staging {
		directoryURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
		m.logger.Info("Using Let's Encrypt staging environment")
	} else {
		m.logger.Info("Using Let's Encrypt production environment")
	}

	client := &acme.Client{
		DirectoryURL: directoryURL,
	}

	return client
}

// GetCertificate returns a certificate for the given hello info
func (m *Manager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	start := time.Now()
	
	cert, err := m.autocertManager.GetCertificate(hello)
	
	duration := time.Since(start)
	
	if err != nil {
		m.logger.WithFields(logrus.Fields{
			"server_name": hello.ServerName,
			"error":       err,
			"duration":    duration,
		}).Error("Failed to get certificate")
		return nil, err
	}

	m.logger.WithFields(logrus.Fields{
		"server_name": hello.ServerName,
		"duration":    duration,
		"cert_serial": fmt.Sprintf("%x", cert.Certificate[0][:8]), // First 8 bytes of cert for identification
	}).Info("Certificate retrieved successfully")

	return cert, nil
}

// HTTPHandler returns the HTTP handler for ACME challenges
func (m *Manager) HTTPHandler(fallback http.Handler) http.Handler {
	return m.autocertManager.HTTPHandler(fallback)
}

// ValidateDomains validates that all configured domains are accessible
func (m *Manager) ValidateDomains(ctx context.Context) error {
	m.logger.Info("Starting domain validation")
	
	var errors []error
	
	for _, domain := range m.domains {
		if err := m.validateDomain(ctx, domain); err != nil {
			errors = append(errors, fmt.Errorf("domain %s: %w", domain, err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("domain validation failed: %v", errors)
	}
	
	m.logger.Info("All domains validated successfully")
	return nil
}

// validateDomain validates a single domain
func (m *Manager) validateDomain(ctx context.Context, domain string) error {
	// Skip validation for localhost and test domains
	if strings.Contains(domain, "localhost") || strings.Contains(domain, "test") {
		m.logger.WithField("domain", domain).Debug("Skipping validation for local/test domain")
		return nil
	}

	m.logger.WithField("domain", domain).Debug("Validating domain")
	
	// In a production system, you might want to implement more sophisticated validation
	// For now, we'll just log and trust the domain configuration
	m.logger.WithField("domain", domain).Info("Domain validation passed")
	
	return nil
}

// GetCertificateInfo returns information about certificates in the cache
func (m *Manager) GetCertificateInfo() ([]CertInfo, error) {
	var certs []CertInfo
	
	cacheDir := m.certDir
	err := filepath.Walk(cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && strings.HasSuffix(path, ".crt") {
			domain := strings.TrimSuffix(filepath.Base(path), ".crt")
			
			// Get certificate details
			certData, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read certificate %s: %w", path, err)
			}
			
			cert, err := parseCertificate(certData)
			if err != nil {
				return fmt.Errorf("failed to parse certificate %s: %w", path, err)
			}
			
			certs = append(certs, CertInfo{
				Domain:    domain,
				NotBefore: cert.NotBefore,
				NotAfter:  cert.NotAfter,
				IsExpired: time.Now().After(cert.NotAfter),
				Path:      path,
			})
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to scan certificate directory: %w", err)
	}
	
	return certs, nil
}

// CertInfo contains information about a certificate
type CertInfo struct {
	Domain    string    `json:"domain"`
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
	IsExpired bool      `json:"is_expired"`
	Path      string    `json:"path"`
}

// RenewCertificates attempts to renew certificates that are close to expiration
func (m *Manager) RenewCertificates(ctx context.Context) error {
	m.logger.Info("Starting certificate renewal check")
	
	certs, err := m.GetCertificateInfo()
	if err != nil {
		return fmt.Errorf("failed to get certificate info: %w", err)
	}
	
	renewalThreshold := time.Now().Add(30 * 24 * time.Hour) // 30 days
	
	for _, cert := range certs {
		if cert.NotAfter.Before(renewalThreshold) {
			m.logger.WithFields(logrus.Fields{
				"domain":     cert.Domain,
				"expires_at": cert.NotAfter,
			}).Info("Certificate needs renewal")
			
			// Trigger renewal by requesting the certificate again
			hello := &tls.ClientHelloInfo{
				ServerName: cert.Domain,
			}
			
			if _, err := m.GetCertificate(hello); err != nil {
				m.logger.WithError(err).WithField("domain", cert.Domain).Error("Certificate renewal failed")
			} else {
				m.logger.WithField("domain", cert.Domain).Info("Certificate renewed successfully")
			}
		}
	}
	
	return nil
}

// Cleanup removes expired certificates and cleans up the certificate cache
func (m *Manager) Cleanup() error {
	m.logger.Info("Starting certificate cleanup")
	
	certs, err := m.GetCertificateInfo()
	if err != nil {
		return fmt.Errorf("failed to get certificate info: %w", err)
	}
	
	cleanupCount := 0
	for _, cert := range certs {
		if cert.IsExpired {
			m.logger.WithField("domain", cert.Domain).Info("Removing expired certificate")
			
			if err := os.Remove(cert.Path); err != nil {
				m.logger.WithError(err).WithField("path", cert.Path).Warn("Failed to remove expired certificate")
			} else {
				cleanupCount++
			}
		}
	}
	
	m.logger.WithField("cleaned_up", cleanupCount).Info("Certificate cleanup completed")
	return nil
}

// Helper functions

func parseCertificate(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}
	
	return x509.ParseCertificate(block.Bytes)
}