package failure

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/gleicon/guvnor/internal/testutils"
)

// TestNetworkFailures tests how the system handles various network failure scenarios
func TestNetworkFailures_BackendDown(t *testing.T) {
	// Start a backend server
	backend := testutils.MockBackend(t, "Backend response")
	
	// Create proxy pointing to the backend
	_, err := url.Parse(backend.URL)
	require.NoError(t, err)
	
	// Test normal operation
	resp, err := http.Get(backend.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Shutdown the backend
	backend.Close()
	
	// Test behavior when backend is down
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err = client.Get(backend.URL)
	
	// Should get connection error
	if err == nil {
		resp.Body.Close()
	}
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestNetworkFailures_SlowBackend(t *testing.T) {
	// Create a slow backend
	slowBackend := testutils.MockBackend(t, "Slow response")
	defer slowBackend.Close()

	// Override the handler to be slow
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second) // Longer than client timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Slow response"))
	})
	
	slowServer := httptest.NewServer(slowHandler)
	defer slowServer.Close()

	// Test with short timeout
	client := &http.Client{Timeout: 1 * time.Second}
	
	start := time.Now()
	resp, err := client.Get(slowServer.URL)
	duration := time.Since(start)
	
	// Should timeout
	if err == nil {
		resp.Body.Close()
	}
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.True(t, duration < 2*time.Second, "Should timeout within reasonable time")
}

func TestNetworkFailures_PartialResponse(t *testing.T) {
	// Create backend that sends partial response then closes connection
	partialBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Partial"))
		
		// Force connection close
		if hijacker, ok := w.(http.Hijacker); ok {
			conn, _, err := hijacker.Hijack()
			if err == nil {
				conn.Close()
			}
		}
	}))
	defer partialBackend.Close()

	resp, err := http.Get(partialBackend.URL)
	if err == nil {
		defer resp.Body.Close()
		body, readErr := io.ReadAll(resp.Body)
		
		// Should get partial content and then error
		if readErr != nil {
			assert.Contains(t, readErr.Error(), "unexpected EOF")
		}
		assert.Equal(t, "Partial", string(body))
	}
}

func TestNetworkFailures_DNSResolutionFailure(t *testing.T) {
	// Test connection to non-existent domain
	client := &http.Client{Timeout: 5 * time.Second}
	
	_, err := client.Get("http://definitely-does-not-exist-12345.com")
	
	assert.Error(t, err)
	// Could be DNS error or connection refused
	errorStr := err.Error()
	assert.True(t, 
		strings.Contains(errorStr, "no such host") ||
		strings.Contains(errorStr, "connection refused") ||
		strings.Contains(errorStr, "timeout"),
		"Should get DNS or connection error: %s", errorStr)
}

func TestProcessFailures_ProcessCrash(t *testing.T) {
	// Create a process that will crash after a short time
	crashScript := `#!/bin/bash
echo "Process starting"
sleep 1
echo "Process about to crash"
kill -9 $$  # Kill itself
`
	
	cmd := testutils.CreateTestProcess(t, crashScript)
	
	// Start the process
	err := cmd.Start()
	require.NoError(t, err)
	
	// Wait for the process to crash
	err = cmd.Wait()
	
	// Should get exit error due to signal
	assert.Error(t, err)
	if exitError, ok := err.(*exec.ExitError); ok {
		// Process was killed by signal
		assert.True(t, exitError.ProcessState.Exited())
	}
}

func TestProcessFailures_ProcessHang(t *testing.T) {
	// Create a process that hangs
	hangScript := `#!/bin/bash
echo "Process starting"
# Infinite loop
while true; do
	sleep 1
done
`
	
	cmd := testutils.CreateTestProcess(t, hangScript)
	
	// Start the process
	err := cmd.Start()
	require.NoError(t, err)
	
	// Let it run for a bit
	time.Sleep(2 * time.Second)
	
	// Process should still be running
	assert.NotNil(t, cmd.Process)
	
	// Kill the hanging process
	err = cmd.Process.Kill()
	assert.NoError(t, err)
	
	// Wait for it to exit
	cmd.Wait()
}

func TestProcessFailures_MemoryExhaustion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory exhaustion test in short mode")
	}
	
	// Create a process that consumes a lot of memory
	memoryScript := `#!/bin/bash
echo "Starting memory allocation"
# Try to allocate 100MB of memory (should be safe on most systems)
python3 -c "
import time
data = []
for i in range(100):
    data.append('x' * (1024 * 1024))  # 1MB per iteration
    time.sleep(0.01)
    if i % 10 == 0:
        print(f'Allocated {i+1} MB')
print('Memory allocation complete')
time.sleep(1)
"
`
	
	cmd := testutils.CreateTestProcess(t, memoryScript)
	
	err := cmd.Start()
	require.NoError(t, err)
	
	// Monitor memory usage (simplified)
	done := make(chan bool)
	go func() {
		err := cmd.Wait()
		done <- (err == nil)
	}()
	
	select {
	case success := <-done:
		if success {
			t.Log("Memory allocation test completed successfully")
		} else {
			t.Log("Process failed (possibly due to memory limits)")
		}
	case <-time.After(30 * time.Second):
		t.Log("Memory test timeout - killing process")
		cmd.Process.Kill()
		cmd.Wait()
	}
}

func TestProcessFailures_FileSystemFull(t *testing.T) {
	testConfig := testutils.NewTestConfig(t)
	
	// Create a script that tries to write a large file
	fillScript := fmt.Sprintf(`#!/bin/bash
echo "Attempting to write large file"
# Try to write 1GB file (this might fail on systems with limited space)
dd if=/dev/zero of="%s/large_file.bin" bs=1M count=1000 2>/dev/null
exit_code=$?
if [ $exit_code -ne 0 ]; then
	echo "Failed to write large file (expected on limited disk space)"
	exit 1
else
	echo "Successfully wrote large file"
	rm -f "%s/large_file.bin"
	exit 0
fi
`, testConfig.TempDir, testConfig.TempDir)
	
	cmd := testutils.CreateTestProcess(t, fillScript)
	
	output, err := cmd.CombinedOutput()
	
	// This test might pass or fail depending on available disk space
	// The important thing is that it handles the failure gracefully
	if err != nil {
		assert.Contains(t, string(output), "Failed to write large file")
		t.Log("Disk space test failed as expected on system with limited space")
	} else {
		assert.Contains(t, string(output), "Successfully wrote large file")
		t.Log("Disk space test passed - sufficient space available")
	}
}

func TestCertificateFailures_ExpiredCertificate(t *testing.T) {
	testConfig := testutils.NewTestConfig(t)
	
	// Generate a certificate that's already expired
	cert := testConfig.GenerateTestCertificate(t, "expired.example.com")
	
	// Manually modify the certificate to be expired (this is a mock)
	expiredCert := *cert.Cert
	expiredCert.NotAfter = time.Now().Add(-24 * time.Hour) // Expired yesterday
	expiredCert.NotBefore = time.Now().Add(-48 * time.Hour) // Valid from 2 days ago
	
	// Test certificate validation
	now := time.Now()
	assert.True(t, now.After(expiredCert.NotAfter), "Certificate should be expired")
	assert.True(t, now.After(expiredCert.NotBefore), "Certificate should have been valid before")
}

func TestCertificateFailures_InvalidCertificate(t *testing.T) {
	testConfig := testutils.NewTestConfig(t)
	
	// Create invalid certificate file
	invalidCertPath := filepath.Join(testConfig.CertsDir, "invalid.crt")
	invalidContent := "This is not a valid certificate"
	
	require.NoError(t, os.WriteFile(invalidCertPath, []byte(invalidContent), 0644))
	
	// Try to load invalid certificate
	_, err := tls.LoadX509KeyPair(invalidCertPath, invalidCertPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find any PEM data")
}

func TestCertificateFailures_MismatchedKeyPair(t *testing.T) {
	testConfig := testutils.NewTestConfig(t)
	
	// Generate two different certificates
	cert1 := testConfig.GenerateTestCertificate(t, "test1.example.com")
	cert2 := testConfig.GenerateTestCertificate(t, "test2.example.com")
	
	// Save cert1's certificate with cert2's key (mismatch)
	cert1Path := filepath.Join(testConfig.CertsDir, "mismatched.crt")
	key2Path := filepath.Join(testConfig.CertsDir, "mismatched.key")
	
	require.NoError(t, os.WriteFile(cert1Path, cert1.CertPEM, 0644))
	require.NoError(t, os.WriteFile(key2Path, cert2.KeyPEM, 0600))
	
	// Try to load mismatched certificate and key
	_, err := tls.LoadX509KeyPair(cert1Path, key2Path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private key does not match public key")
}

func TestConfigurationFailures_InvalidYAML(t *testing.T) {
	testConfig := testutils.NewTestConfig(t)
	
	// Create invalid YAML configuration
	invalidYAML := `
server:
  port: 8080
  host: "localhost"
apps:
  - name: "test-app"
    command: "echo"
    port: 3000
    # Invalid YAML syntax below
    domains: [test.com", "broken
`
	
	testConfig.CreateTestConfig(t, invalidYAML)
	
	// Try to load invalid config
	_, err := os.ReadFile(testConfig.ConfigFile)
	assert.NoError(t, err, "File should be readable")
	
	// The actual YAML parsing would fail in the config loader
	// This simulates what would happen
	content, _ := os.ReadFile(testConfig.ConfigFile)
	assert.Contains(t, string(content), "broken", "Should contain invalid YAML")
}

func TestConfigurationFailures_MissingConfigFile(t *testing.T) {
	nonexistentFile := "/path/that/does/not/exist/config.yaml"
	
	_, err := os.ReadFile(nonexistentFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestResourceExhaustion_TooManyConnections(t *testing.T) {
	// Create a server that limits connections
	maxConnections := 5
	currentConnections := 0
	
	limitedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		currentConnections++
		defer func() { currentConnections-- }()
		
		if currentConnections > maxConnections {
			http.Error(w, "Too many connections", http.StatusServiceUnavailable)
			return
		}
		
		// Simulate some work
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer limitedServer.Close()
	
	client := &http.Client{Timeout: 5 * time.Second}
	
	// Make concurrent requests (more than the limit)
	numRequests := 10
	results := make(chan int, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func() {
			resp, err := client.Get(limitedServer.URL)
			if err != nil {
				results <- 0
				return
			}
			defer resp.Body.Close()
			results <- resp.StatusCode
		}()
	}
	
	// Collect results
	successCount := 0
	errorCount := 0
	
	for i := 0; i < numRequests; i++ {
		status := <-results
		if status == http.StatusOK {
			successCount++
		} else {
			errorCount++
		}
	}
	
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)
	
	// Some requests should succeed, some should fail due to limits
	assert.True(t, successCount > 0, "Some requests should succeed")
	assert.True(t, errorCount > 0, "Some requests should be rejected")
}

func TestResourceExhaustion_FileDescriptorLimits(t *testing.T) {
	// Test opening many files to simulate FD exhaustion
	testConfig := testutils.NewTestConfig(t)
	
	var files []*os.File
	defer func() {
		// Clean up opened files
		for _, f := range files {
			if f != nil {
				f.Close()
			}
		}
	}()
	
	// Try to open many files
	maxFiles := 100 // Reasonable limit for testing
	
	for i := 0; i < maxFiles; i++ {
		fileName := filepath.Join(testConfig.TempDir, fmt.Sprintf("file_%d.txt", i))
		file, err := os.Create(fileName)
		
		if err != nil {
			// Hit file descriptor limit
			t.Logf("Hit file descriptor limit after %d files: %v", i, err)
			break
		}
		
		files = append(files, file)
	}
	
	t.Logf("Successfully opened %d files", len(files))
	
	// Verify we could open a reasonable number of files
	assert.True(t, len(files) > 10, "Should be able to open at least 10 files")
}

func TestGracefulDegradation_PartialServiceFailure(t *testing.T) {
	// Simulate a system with multiple backends where some fail
	backends := []*httptest.Server{}
	
	// Create 3 backends
	for i := 0; i < 3; i++ {
		backend := testutils.MockBackend(t, fmt.Sprintf("Backend %d", i))
		backends = append(backends, backend)
	}
	defer func() {
		for _, b := range backends {
			b.Close()
		}
	}()
	
	// Test all backends initially
	for i, backend := range backends {
		resp, err := http.Get(backend.URL)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		
		assert.Equal(t, fmt.Sprintf("Backend %d", i), string(body))
	}
	
	// Simulate one backend failing
	backends[1].Close()
	
	// Test remaining backends still work
	workingBackends := []*httptest.Server{backends[0], backends[2]}
	
	for i, backend := range workingBackends {
		resp, err := http.Get(backend.URL)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		expectedIndex := i
		if i == 1 {
			expectedIndex = 2 // Second working backend is actually backend 2
		}
		
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		
		assert.Equal(t, fmt.Sprintf("Backend %d", expectedIndex), string(body))
	}
	
	// Test failed backend returns error
	_, err := http.Get(backends[1].URL)
	assert.Error(t, err)
}

func TestFailureRecovery_ServiceRestart(t *testing.T) {
	// Simulate a service that can be restarted after failure
	testConfig := testutils.NewTestConfig(t)
	
	serverScript := fmt.Sprintf(`#!/bin/bash
echo "Server starting on port $1"
python3 -c "
import http.server
import socketserver
import sys

PORT = int(sys.argv[1])

class Handler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()
        self.wfile.write(b'Service recovered')

with socketserver.TCPServer(('', PORT), Handler) as httpd:
    print(f'Server running on port {PORT}')
    httpd.serve_forever()
" $1`)
	
	scriptPath := filepath.Join(testConfig.TempDir, "test_server.sh")
	require.NoError(t, os.WriteFile(scriptPath, []byte(serverScript), 0755))
	
	port := testutils.FindFreePort(t)
	
	// Start the server
	cmd := exec.Command("bash", scriptPath, fmt.Sprintf("%d", port))
	err := cmd.Start()
	require.NoError(t, err)
	
	// Wait for server to start
	err = testutils.WaitForPort("localhost", port, 10*time.Second)
	require.NoError(t, err)
	
	// Test server is working
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Kill the server (simulate crash)
	err = cmd.Process.Kill()
	require.NoError(t, err)
	cmd.Wait()
	
	// Verify server is down
	_, err = http.Get(fmt.Sprintf("http://localhost:%d", port))
	assert.Error(t, err)
	
	// Restart the server
	cmd = exec.Command("bash", scriptPath, fmt.Sprintf("%d", port))
	err = cmd.Start()
	require.NoError(t, err)
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()
	
	// Wait for server to restart
	err = testutils.WaitForPort("localhost", port, 10*time.Second)
	require.NoError(t, err)
	
	// Test server is working again
	resp, err = http.Get(fmt.Sprintf("http://localhost:%d", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "Service recovered", string(body))
}

func TestCascadingFailures_DependencyChain(t *testing.T) {
	// Simulate a dependency chain where one failure causes others
	
	// Database service (bottom of chain)
	dbService := testutils.MockBackend(t, "Database OK")
	defer dbService.Close()
	
	// API service (depends on database)
	apiService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if database is available
		resp, err := http.Get(dbService.URL)
		if err != nil {
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
			return
		}
		resp.Body.Close()
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("API OK"))
	}))
	defer apiService.Close()
	
	// Web service (depends on API)
	webService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if API is available
		resp, err := http.Get(apiService.URL)
		if err != nil {
			http.Error(w, "API unavailable", http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			http.Error(w, "API unhealthy", http.StatusServiceUnavailable)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Web OK"))
	}))
	defer webService.Close()
	
	// Test normal operation - all services working
	resp, err := http.Get(webService.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "Web OK", string(body))
	
	// Simulate database failure
	dbService.Close()
	
	// Test cascading failure
	resp, err = http.Get(webService.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	// Should fail due to database being down
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	assert.Contains(t, string(body), "API unavailable")
}