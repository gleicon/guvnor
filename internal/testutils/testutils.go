package testutils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCertificate represents a test TLS certificate
type TestCertificate struct {
	CertPEM []byte
	KeyPEM  []byte
	Cert    *x509.Certificate
	Key     *rsa.PrivateKey
}

// MockProcess represents a mock process for testing
type MockProcess struct {
	ID      int
	Command string
	Args    []string
	Status  string
	PID     int
	Started time.Time
}

// TestConfig provides test configuration utilities
type TestConfig struct {
	TempDir     string
	CertsDir    string
	ConfigFile  string
	TestCerts   map[string]*TestCertificate
}

// NewTestConfig creates a new test configuration
func NewTestConfig(t *testing.T) *TestConfig {
	tempDir := t.TempDir()
	certsDir := filepath.Join(tempDir, "certs")
	require.NoError(t, os.MkdirAll(certsDir, 0755))

	return &TestConfig{
		TempDir:    tempDir,
		CertsDir:   certsDir,
		ConfigFile: filepath.Join(tempDir, "config.yaml"),
		TestCerts:  make(map[string]*TestCertificate),
	}
}

// GenerateTestCertificate generates a self-signed certificate for testing
func (tc *TestConfig) GenerateTestCertificate(t *testing.T, domain string) *TestCertificate {
	// Generate private key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:     []string{domain, "localhost"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	// Encode to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	testCert := &TestCertificate{
		CertPEM: certPEM,
		KeyPEM:  keyPEM,
		Cert:    cert,
		Key:     key,
	}

	tc.TestCerts[domain] = testCert
	return testCert
}

// SaveCertificate saves a certificate to disk for testing
func (tc *TestConfig) SaveCertificate(t *testing.T, domain string, cert *TestCertificate) (string, string) {
	certPath := filepath.Join(tc.CertsDir, domain+".crt")
	keyPath := filepath.Join(tc.CertsDir, domain+".key")

	require.NoError(t, os.WriteFile(certPath, cert.CertPEM, 0644))
	require.NoError(t, os.WriteFile(keyPath, cert.KeyPEM, 0600))

	return certPath, keyPath
}

// CreateTestConfig creates a test configuration file
func (tc *TestConfig) CreateTestConfig(t *testing.T, config string) {
	require.NoError(t, os.WriteFile(tc.ConfigFile, []byte(config), 0644))
}

// MockBackend creates a mock HTTP backend server for testing
func MockBackend(t *testing.T, response string) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(response))
		require.NoError(t, err)
	})
	return httptest.NewServer(handler)
}

// MockTLSBackend creates a mock HTTPS backend server for testing
func MockTLSBackend(t *testing.T, response string) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(response))
		require.NoError(t, err)
	})
	return httptest.NewTLSServer(handler)
}

// FindFreePort finds an available port for testing
func FindFreePort(t *testing.T) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(t, err)

	l, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port
}

// WaitForPort waits for a port to be available or timeout
func WaitForPort(host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("port %d not available after %v", port, timeout)
}

// CreateTestProcess creates a test process command
func CreateTestProcess(t *testing.T, script string) *exec.Cmd {
	tempFile := filepath.Join(t.TempDir(), "test_script.sh")
	require.NoError(t, os.WriteFile(tempFile, []byte(script), 0755))
	return exec.Command("bash", tempFile)
}

// HTTPSClient creates an HTTPS client that accepts self-signed certificates
func HTTPSClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr}
}

// AssertEventuallyTrue asserts that a condition becomes true within a timeout
func AssertEventuallyTrue(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("Condition did not become true within %v: %s", timeout, message)
}

// CaptureOutput captures stdout and stderr from a function
func CaptureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	require.NoError(t, err)
	rErr, wErr, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = wOut
	os.Stderr = wErr

	done := make(chan struct{})
	var outBuf, errBuf strings.Builder

	go func() {
		defer close(done)
		io.Copy(&outBuf, rOut)
	}()
	go func() {
		io.Copy(&errBuf, rErr)
	}()

	fn()

	wOut.Close()
	wErr.Close()
	<-done

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return outBuf.String(), errBuf.String()
}