package cert

import (
	"crypto/x509"
	"fmt"
	"strings"
)

// CertificateInfo contains extracted certificate information for headers
type CertificateInfo struct {
	Subject    string `json:"subject"`
	Issuer     string `json:"issuer"`
	Serial     string `json:"serial"`
	NotBefore  string `json:"not_before"`
	NotAfter   string `json:"not_after"`
	IsExpired  bool   `json:"is_expired"`
	CommonName string `json:"common_name"`
}

// ExtractCertificateInfo extracts certificate information from x509.Certificate
// This is inspired by valve's certificate header injection
func ExtractCertificateInfo(cert *x509.Certificate) *CertificateInfo {
	if cert == nil {
		return nil
	}

	return &CertificateInfo{
		Subject:    cert.Subject.String(),
		Issuer:     cert.Issuer.String(),
		Serial:     cert.SerialNumber.String(),
		NotBefore:  cert.NotBefore.Format("2006-01-02T15:04:05Z"),
		NotAfter:   cert.NotAfter.Format("2006-01-02T15:04:05Z"),
		IsExpired:  cert.NotAfter.Before(cert.NotBefore),
		CommonName: cert.Subject.CommonName,
	}
}

// FormatCertificateSubject formats certificate subject for header injection
// Similar to valve's X-CERTIFICATE-CN header
func FormatCertificateSubject(cert *x509.Certificate) string {
	if cert == nil {
		return ""
	}
	
	// Extract key components of the subject
	var parts []string
	
	if cert.Subject.CommonName != "" {
		parts = append(parts, fmt.Sprintf("CN=%s", cert.Subject.CommonName))
	}
	
	for _, org := range cert.Subject.Organization {
		parts = append(parts, fmt.Sprintf("O=%s", org))
	}
	
	for _, orgUnit := range cert.Subject.OrganizationalUnit {
		parts = append(parts, fmt.Sprintf("OU=%s", orgUnit))
	}
	
	if len(cert.Subject.Country) > 0 {
		parts = append(parts, fmt.Sprintf("C=%s", cert.Subject.Country[0]))
	}
	
	if len(cert.Subject.Province) > 0 {
		parts = append(parts, fmt.Sprintf("ST=%s", cert.Subject.Province[0]))
	}
	
	if len(cert.Subject.Locality) > 0 {
		parts = append(parts, fmt.Sprintf("L=%s", cert.Subject.Locality[0]))
	}
	
	return strings.Join(parts, ", ")
}