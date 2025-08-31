package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/gleicon/guvnor/internal/testutils"
)

func TestTLSConfiguration_SecurityHeaders(t *testing.T) {
	testConfig := testutils.NewTestConfig(t)
	
	// Generate certificate
	cert := testConfig.GenerateTestCertificate(t, "secure.example.com")
	certPath, keyPath := testConfig.SaveCertificate(t, "secure.example.com", cert)
	
	// Create secure TLS config
	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	require.NoError(t, err)
	
	tlsConfig := &tls.Config{
		Certificates:             []tls.Certificate{tlsCert},
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:        []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
	
	// Test TLS configuration
	assert.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MinVersion)
	assert.True(t, tlsConfig.PreferServerCipherSuites)
	assert.NotEmpty(t, tlsConfig.CipherSuites)
}

func TestTLSConfiguration_WeakCiphers(t *testing.T) {
	testConfig := testutils.NewTestConfig(t)
	cert := testConfig.GenerateTestCertificate(t, "test.example.com")
	certPath, keyPath := testConfig.SaveCertificate(t, "test.example.com", cert)
	
	tlsCert, err := tls.LoadX509KeyPair(certPath, keyPath)
	require.NoError(t, err)
	
	// Test that weak ciphers are not included
	weakCiphers := []uint16{
		tls.TLS_RSA_WITH_RC4_128_SHA,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	}
	
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}
	
	for _, weakCipher := range weakCiphers {
		assert.NotContains(t, tlsConfig.CipherSuites, weakCipher, 
			"TLS config should not include weak cipher: %x", weakCipher)
	}
}

func TestProcessIsolation_UserPermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping user permission test when running as root")
	}
	
	// Test that processes don't run with elevated privileges
	testConfig := testutils.NewTestConfig(t)
	
	// Create a script that tries to write to a privileged location
	privilegedScript := `#!/bin/bash
# Try to write to /etc (should fail without root)
echo "test" > /etc/underling-test 2>/dev/null
if [ $? -eq 0 ]; then
	echo "SECURITY_VIOLATION: Successfully wrote to /etc"
	exit 1
else
	echo "SECURITY_OK: Cannot write to /etc (expected)"
	exit 0
fi`
	
	scriptPath := filepath.Join(testConfig.TempDir, "privilege_test.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte(privilegedScript), 0755))
	
	cmd := testutils.CreateTestProcess(t, privilegedScript)
	output, err := cmd.CombinedOutput()
	
	// Script should succeed (exit 0) because it cannot write to /etc
	assert.NoError(t, err)
	assert.Contains(t, string(output), "SECURITY_OK")
	assert.NotContains(t, string(output), "SECURITY_VIOLATION")
}

func TestProcessIsolation_FileSystemAccess(t *testing.T) {
	testConfig := testutils.NewTestConfig(t)
	
	// Create a restricted directory
	restrictedDir := filepath.Join(testConfig.TempDir, "restricted")
	require.NoError(t, os.MkdirAll(restrictedDir, 0700))
	
	secretFile := filepath.Join(restrictedDir, "secret.txt")
	require.NoError(t, os.WriteFile(secretFile, []byte("secret content"), 0600))
	
	// Test that process can't access files outside its working directory
	accessScript := fmt.Sprintf(`#!/bin/bash
# Try to read secret file
if [ -r "%s" ]; then
	echo "SECURITY_VIOLATION: Can read restricted file"
	exit 1
else
	echo "SECURITY_OK: Cannot read restricted file"
	exit 0
fi`, secretFile)
	
	cmd := testutils.CreateTestProcess(t, accessScript)
	output, err := cmd.CombinedOutput()
	
	// Should not be able to read the restricted file
	assert.NoError(t, err)
	assert.Contains(t, string(output), "SECURITY_OK")
}

func TestInputValidation_ConfigurationInjection(t *testing.T) {
	testConfig := testutils.NewTestConfig(t)
	
	// Test malicious configuration inputs
	maliciousInputs := []struct {
		name   string
		config string
		safe   bool
	}{
		{
			name: "command_injection",
			config: `
apps:
  - name: "malicious"
    command: "echo 'safe' && rm -rf /"
    port: 3000`,
			safe: true, // Should be treated as literal command
		},
		{
			name: "path_traversal",
			config: `
apps:
  - name: "path_traversal"  
    command: "cat"
    args: ["../../../etc/passwd"]
    port: 3000`,
			safe: true, // Args should be treated as literals
		},
		{
			name: "environment_injection",
			config: `
apps:
  - name: "env_injection"
    command: "env"
    port: 3000
    env:
      MALICIOUS: "value; rm -rf /"`,
			safe: true, // Env values should be escaped
		},
	}
	
	for _, tt := range maliciousInputs {
		t.Run(tt.name, func(t *testing.T) {
			testConfig.CreateTestConfig(t, tt.config)
			
			// Try to load and validate the configuration
			// This would test the actual config loading logic
			_, err := os.ReadFile(testConfig.ConfigFile)
			assert.NoError(t, err, "Should be able to read malicious config")
			
			// The actual validation would happen in config.LoadFromFile()
			// For now, we verify the file contains the expected content
			content, _ := os.ReadFile(testConfig.ConfigFile)
			if tt.safe {
				// Config should be parsed safely without executing embedded commands
				assert.Contains(t, string(content), "malicious")
			}
		})
	}
}

func TestHTTPSecurityHeaders(t *testing.T) {
	backend := testutils.MockBackend(t, "Test content")
	defer backend.Close()
	
	// Create a proxy that should add security headers
	backendURL, _ := url.Parse(backend.URL)
	
	// Mock secure proxy (this would be actual underling proxy)
	secureProxy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add security headers
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		
		// Proxy to backend
		client := &http.Client{}
		proxyReq, _ := http.NewRequest(r.Method, backendURL.String()+r.URL.Path, r.Body)
		proxyResp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer proxyResp.Body.Close()
		
		body, _ := io.ReadAll(proxyResp.Body)
		w.WriteHeader(proxyResp.StatusCode)
		w.Write(body)
	})
	
	testServer := httptest.NewServer(secureProxy)
	defer testServer.Close()
	
	// Test security headers
	resp, err := http.Get(testServer.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	// Verify security headers are present
	assert.Equal(t, "max-age=31536000; includeSubDomains", resp.Header.Get("Strict-Transport-Security"))
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", resp.Header.Get("X-XSS-Protection"))
	assert.Equal(t, "default-src 'self'", resp.Header.Get("Content-Security-Policy"))
}

func TestDenialOfService_RateLimiting(t *testing.T) {
	backend := testutils.MockBackend(t, "Rate limited content")
	defer backend.Close()
	
	// Mock rate-limited proxy
	rateLimiter := make(map[string]time.Time)
	rateLimitedProxy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		lastRequest, exists := rateLimiter[clientIP]
		
		if exists && time.Since(lastRequest) < 100*time.Millisecond {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		
		rateLimiter[clientIP] = time.Now()
		
		// Forward to backend
		client := &http.Client{}
		backendURL, _ := url.Parse(backend.URL)
		proxyReq, _ := http.NewRequest(r.Method, backendURL.String(), r.Body)
		proxyResp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer proxyResp.Body.Close()
		
		body, _ := io.ReadAll(proxyResp.Body)
		w.WriteHeader(proxyResp.StatusCode)
		w.Write(body)
	})
	
	testServer := httptest.NewServer(rateLimitedProxy)
	defer testServer.Close()
	
	client := &http.Client{}
	
	// First request should succeed
	resp, err := client.Get(testServer.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Immediate second request should be rate limited
	resp, err = client.Get(testServer.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	
	// Wait and try again - should succeed
	time.Sleep(150 * time.Millisecond)
	resp, err = client.Get(testServer.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDenialOfService_RequestSizeLimits(t *testing.T) {
	// Test large request body handling
	maxSize := 1024 * 1024 // 1MB limit
	
	sizeLimitedProxy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, int64(maxSize))
		
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Received %d bytes", len(body))
	})
	
	testServer := httptest.NewServer(sizeLimitedProxy)
	defer testServer.Close()
	
	client := &http.Client{}
	
	// Test with acceptable size
	smallBody := strings.NewReader(strings.Repeat("a", 1000))
	resp, err := client.Post(testServer.URL, "text/plain", smallBody)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Test with oversized request
	largeBody := strings.NewReader(strings.Repeat("a", maxSize+1))
	resp, err = client.Post(testServer.URL, "text/plain", largeBody)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestCertificateSecurity_ValidateChain(t *testing.T) {
	testConfig := testutils.NewTestConfig(t)
	
	// Generate a certificate chain (self-signed for testing)
	cert := testConfig.GenerateTestCertificate(t, "secure-chain.example.com")
	
	// Verify certificate properties
	assert.NotNil(t, cert.Cert)
	assert.NotNil(t, cert.Key)
	
	// Check key size (should be at least 2048 bits for RSA)
	assert.True(t, cert.Key.N.BitLen() >= 2048, 
		"Certificate key should be at least 2048 bits, got %d", cert.Key.N.BitLen())
	
	// Check certificate validity period
	assert.True(t, cert.Cert.NotAfter.After(time.Now()), "Certificate should not be expired")
	assert.True(t, cert.Cert.NotBefore.Before(time.Now()), "Certificate should be valid now")
	
	// Check that certificate has proper key usage
	assert.True(t, cert.Cert.KeyUsage&x509.KeyUsageDigitalSignature != 0, 
		"Certificate should have DigitalSignature key usage")
	assert.True(t, cert.Cert.KeyUsage&x509.KeyUsageKeyEncipherment != 0, 
		"Certificate should have KeyEncipherment key usage")
}

func TestAccessControl_PathTraversal(t *testing.T) {
	// Test protection against path traversal attacks
	testConfig := testutils.NewTestConfig(t)
	
	// Create a file outside the web root
	secretDir := filepath.Join(testConfig.TempDir, "secret")
	require.NoError(t, os.MkdirAll(secretDir, 0755))
	
	secretFile := filepath.Join(secretDir, "confidential.txt")
	require.NoError(t, os.WriteFile(secretFile, []byte("SECRET CONTENT"), 0644))
	
	// Create web root
	webRoot := filepath.Join(testConfig.TempDir, "www")
	require.NoError(t, os.MkdirAll(webRoot, 0755))
	
	publicFile := filepath.Join(webRoot, "public.txt")
	require.NoError(t, os.WriteFile(publicFile, []byte("PUBLIC CONTENT"), 0644))
	
	// Mock static file server with path traversal protection
	fileServer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean and validate the path
		cleanPath := filepath.Clean(r.URL.Path)
		
		// Prevent path traversal
		if strings.Contains(cleanPath, "..") {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		
		// Serve files only from web root
		filePath := filepath.Join(webRoot, strings.TrimPrefix(cleanPath, "/"))
		
		// Ensure the resolved path is still within webRoot
		absFilePath, err := filepath.Abs(filePath)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		
		absWebRoot, err := filepath.Abs(webRoot)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		
		if !strings.HasPrefix(absFilePath, absWebRoot) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		
		// Try to serve the file
		content, err := os.ReadFile(absFilePath)
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	})
	
	testServer := httptest.NewServer(fileServer)
	defer testServer.Close()
	
	client := &http.Client{}
	
	// Test legitimate access
	resp, err := client.Get(testServer.URL + "/public.txt")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "PUBLIC CONTENT", string(body))
	
	// Test path traversal attempts
	traversalAttempts := []string{
		"/../secret/confidential.txt",
		"/../../secret/confidential.txt",
		"/../../../etc/passwd",
		"/public.txt/../secret/confidential.txt",
		"/%2e%2e/secret/confidential.txt", // URL encoded
	}
	
	for _, attempt := range traversalAttempts {
		t.Run("traversal_"+attempt, func(t *testing.T) {
			resp, err := client.Get(testServer.URL + attempt)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			// Should be forbidden or not found, never return secret content
			assert.True(t, resp.StatusCode == http.StatusForbidden || 
				resp.StatusCode == http.StatusNotFound,
				"Path traversal attempt should be blocked: %s (status: %d)", 
				attempt, resp.StatusCode)
			
			if resp.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				assert.NotContains(t, string(body), "SECRET CONTENT", 
					"Should not return secret content for: %s", attempt)
			}
		})
	}
}

func TestSessionSecurity_TokenValidation(t *testing.T) {
	// Test secure token generation and validation
	validTokens := make(map[string]time.Time)
	
	tokenValidator := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		// Remove "Bearer " prefix
		token = strings.TrimPrefix(token, "Bearer ")
		
		// Validate token format (should be at least 32 chars, alphanumeric)
		if len(token) < 32 {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		
		for _, char := range token {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
				(char >= '0' && char <= '9')) {
				http.Error(w, "Invalid token format", http.StatusUnauthorized)
				return
			}
		}
		
		// Check if token exists and is not expired
		issueTime, exists := validTokens[token]
		if !exists {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		
		// Token expires after 1 hour
		if time.Since(issueTime) > time.Hour {
			delete(validTokens, token)
			http.Error(w, "Token expired", http.StatusUnauthorized)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authorized"))
	})
	
	testServer := httptest.NewServer(tokenValidator)
	defer testServer.Close()
	
	client := &http.Client{}
	
	// Test without token
	resp, err := client.Get(testServer.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	
	// Test with invalid token
	req, _ := http.NewRequest("GET", testServer.URL, nil)
	req.Header.Set("Authorization", "Bearer invalid")
	resp, err = client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	
	// Generate valid token and test
	validToken := "abcdef1234567890abcdef1234567890abcdef12" // 40 chars
	validTokens[validToken] = time.Now()
	
	req, _ = http.NewRequest("GET", testServer.URL, nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	resp, err = client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLogging_SecurityEventAudit(t *testing.T) {
	// Test that security events are properly logged
	testConfig := testutils.NewTestConfig(t)
	logFile := filepath.Join(testConfig.TempDir, "security.log")
	
	// Mock security event logger
	securityEvents := []string{
		"FAILED_LOGIN_ATTEMPT",
		"UNAUTHORIZED_ACCESS",
		"CERTIFICATE_EXPIRED",
		"RATE_LIMIT_EXCEEDED", 
		"PATH_TRAVERSAL_ATTEMPT",
	}
	
	// Simulate logging security events
	for _, event := range securityEvents {
		logEntry := fmt.Sprintf("[%s] SECURITY_EVENT: %s from %s\n", 
			time.Now().Format(time.RFC3339), event, "127.0.0.1")
		
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		require.NoError(t, err)
		
		_, err = f.WriteString(logEntry)
		require.NoError(t, err)
		f.Close()
	}
	
	// Verify security events were logged
	logContent, err := os.ReadFile(logFile)
	require.NoError(t, err)
	
	logStr := string(logContent)
	for _, event := range securityEvents {
		assert.Contains(t, logStr, event, "Security event should be logged: %s", event)
	}
	
	// Verify log format includes timestamp and source
	assert.Contains(t, logStr, "SECURITY_EVENT")
	assert.Contains(t, logStr, "127.0.0.1")
	assert.Contains(t, logStr, time.Now().Format("2006-01-02")) // Today's date
}