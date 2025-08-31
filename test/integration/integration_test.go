package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/gleicon/guvnor/internal/testutils"
)

// IntegrationTestSuite contains full end-to-end integration tests
type IntegrationTestSuite struct {
	suite.Suite
	testConfig     *testutils.TestConfig
	underlingPort  int
	testApps       map[string]*TestApp
	underlingCmd   *exec.Cmd
	ctx           context.Context
	cancel        context.CancelFunc
}

type TestApp struct {
	Name     string
	Port     int
	Domain   string
	Process  *exec.Cmd
	HealthPath string
}

func TestIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.testConfig = testutils.NewTestConfig(s.T())
	s.underlingPort = testutils.FindFreePort(s.T())
	s.testApps = make(map[string]*TestApp)
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Create test applications
	s.createTestApps()
	
	// Generate test configuration
	s.createUnderlingConfig()
	
	// Start underling server
	s.startUnderlingServer()
	
	// Wait for server to be ready
	s.waitForServerReady()
}

func (s *IntegrationTestSuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}
	
	// Stop underling server
	if s.underlingCmd != nil && s.underlingCmd.Process != nil {
		s.underlingCmd.Process.Kill()
		s.underlingCmd.Wait()
	}
	
	// Stop test apps
	for _, app := range s.testApps {
		if app.Process != nil && app.Process.Process != nil {
			app.Process.Process.Kill()
			app.Process.Wait()
		}
	}
}

func (s *IntegrationTestSuite) createTestApps() {
	// Node.js app
	nodeApp := &TestApp{
		Name:       "node-app",
		Port:       testutils.FindFreePort(s.T()),
		Domain:     "node.test.local",
		HealthPath: "/health",
	}
	
	// Python app  
	pythonApp := &TestApp{
		Name:       "python-app",
		Port:       testutils.FindFreePort(s.T()),
		Domain:     "python.test.local",
		HealthPath: "/status",
	}
	
	s.testApps["node"] = nodeApp
	s.testApps["python"] = pythonApp
	
	// Create simple Node.js test server
	nodeScript := fmt.Sprintf(`
const http = require('http');
const port = %d;

const server = http.createServer((req, res) => {
	if (req.url === '/health') {
		res.writeHead(200, {'Content-Type': 'application/json'});
		res.end('{"status": "healthy", "app": "node", "timestamp": ' + Date.now() + '}');
	} else if (req.url === '/') {
		res.writeHead(200, {'Content-Type': 'text/html'});
		res.end('<h1>Node.js App</h1><p>Hello from Node.js running on port %d</p>');
	} else {
		res.writeHead(404);
		res.end('Not Found');
	}
});

server.listen(port, () => {
	console.log('Node.js server running on port ' + port);
});
`, nodeApp.Port, nodeApp.Port)
	
	nodeScriptPath := filepath.Join(s.testConfig.TempDir, "node-server.js")
	require.NoError(s.T(), os.WriteFile(nodeScriptPath, []byte(nodeScript), 0644))
	
	// Create simple Python test server
	pythonScript := fmt.Sprintf(`#!/usr/bin/env python3
import http.server
import socketserver
import json
import time

class TestHandler(http.server.SimpleHTTPRequestHandler):
	def do_GET(self):
		if self.path == '/status':
			self.send_response(200)
			self.send_header('Content-Type', 'application/json')
			self.end_headers()
			response = {
				"status": "healthy",
				"app": "python",
				"timestamp": int(time.time())
			}
			self.wfile.write(json.dumps(response).encode())
		elif self.path == '/':
			self.send_response(200)
			self.send_header('Content-Type', 'text/html')
			self.end_headers()
			html = f'<h1>Python App</h1><p>Hello from Python running on port %d</p>'
			self.wfile.write(html.encode())
		else:
			self.send_response(404)
			self.end_headers()
			self.wfile.write(b'Not Found')

with socketserver.TCPServer(("", %d), TestHandler) as httpd:
	print(f"Python server running on port %d")
	httpd.serve_forever()
`, pythonApp.Port, pythonApp.Port, pythonApp.Port)
	
	pythonScriptPath := filepath.Join(s.testConfig.TempDir, "python-server.py")
	require.NoError(s.T(), os.WriteFile(pythonScriptPath, []byte(pythonScript), 0755))
}

func (s *IntegrationTestSuite) createUnderlingConfig() {
	config := fmt.Sprintf(`
server:
  port: %d
  host: "127.0.0.1"
  tls:
    enabled: true
    auto_cert: false  # Use self-signed for testing
    cert_dir: "%s"

apps:
  - name: "node-app"
    command: "node"
    args: ["%s/node-server.js"]
    port: %d
    domains: ["%s"]
    instances: 1
    health_check:
      path: "/health"
      interval: "10s"
      timeout: "5s"
    restart_policy:
      policy: "always"
      max_retries: 3
      delay: "2s"
    env:
      NODE_ENV: "test"

  - name: "python-app"
    command: "python3"
    args: ["%s/python-server.py"]
    port: %d
    domains: ["%s"]
    instances: 1
    health_check:
      path: "/status"
      interval: "15s"
      timeout: "5s"
    restart_policy:
      policy: "on-failure"
      max_retries: 2
      delay: "3s"
    env:
      PYTHONPATH: "%s"

logging:
  level: "debug"
  file: "%s/underling.log"

metrics:
  enabled: true
  port: %d
`, 
		s.underlingPort, 
		s.testConfig.CertsDir,
		s.testConfig.TempDir, s.testApps["node"].Port, s.testApps["node"].Domain,
		s.testConfig.TempDir, s.testApps["python"].Port, s.testApps["python"].Domain,
		s.testConfig.TempDir,
		s.testConfig.TempDir,
		s.underlingPort+1000, // Metrics port
	)
	
	s.testConfig.CreateTestConfig(s.T(), config)
}

func (s *IntegrationTestSuite) startUnderlingServer() {
	// Generate self-signed certificates for test domains
	for _, app := range s.testApps {
		cert := s.testConfig.GenerateTestCertificate(s.T(), app.Domain)
		s.testConfig.SaveCertificate(s.T(), app.Domain, cert)
	}
	
	// Start underling server
	s.underlingCmd = exec.CommandContext(s.ctx, "go", "run", 
		"../../cmd/guvnor/main.go", 
		"--config", s.testConfig.ConfigFile)
	
	s.underlingCmd.Dir = filepath.Join(s.testConfig.TempDir, "../..")
	s.underlingCmd.Stdout = os.Stdout
	s.underlingCmd.Stderr = os.Stderr
	
	require.NoError(s.T(), s.underlingCmd.Start())
}

func (s *IntegrationTestSuite) waitForServerReady() {
	// Wait for underling server to be ready
	err := testutils.WaitForPort("127.0.0.1", s.underlingPort, 30*time.Second)
	require.NoError(s.T(), err, "Underling server should start within 30 seconds")
	
	// Wait for apps to be started by underling
	time.Sleep(5 * time.Second)
	
	// Verify apps are running
	for name, app := range s.testApps {
		err := testutils.WaitForPort("127.0.0.1", app.Port, 15*time.Second)
		require.NoError(s.T(), err, "App %s should start within 15 seconds", name)
	}
}

func (s *IntegrationTestSuite) TestBasicProxyFunctionality() {
	client := testutils.HTTPSClient()
	
	tests := []struct {
		domain   string
		path     string
		expected string
	}{
		{s.testApps["node"].Domain, "/", "Node.js App"},
		{s.testApps["python"].Domain, "/", "Python App"},
		{s.testApps["node"].Domain, "/health", "node"},
		{s.testApps["python"].Domain, "/status", "python"},
	}
	
	for _, tt := range tests {
		s.T().Run(fmt.Sprintf("%s%s", tt.domain, tt.path), func(t *testing.T) {
			req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d%s", s.underlingPort, tt.path), nil)
			require.NoError(t, err)
			req.Host = tt.domain
			
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			
			assert.Contains(t, string(body), tt.expected)
		})
	}
}

func (s *IntegrationTestSuite) TestTLSTermination() {
	client := testutils.HTTPSClient()
	
	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/", s.underlingPort), nil)
	require.NoError(s.T(), err)
	req.Host = s.testApps["node"].Domain
	
	resp, err := client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()
	
	// Verify TLS connection
	assert.NotNil(s.T(), resp.TLS)
	assert.True(s.T(), resp.TLS.HandshakeComplete)
	assert.Equal(s.T(), uint16(tls.VersionTLS12), resp.TLS.Version)
}

func (s *IntegrationTestSuite) TestHealthChecking() {
	// Wait for health checks to run
	time.Sleep(12 * time.Second)
	
	// Check metrics endpoint for health status
	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", s.underlingPort+1000))
	require.NoError(s.T(), err)
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)
	
	metrics := string(body)
	
	// Verify health check metrics
	assert.Contains(s.T(), metrics, "underling_health_checks_total")
	assert.Contains(s.T(), metrics, "underling_healthy_services")
}

func (s *IntegrationTestSuite) TestProcessRestart() {
	client := testutils.HTTPSClient()
	
	// First, verify the service is running
	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/", s.underlingPort), nil)
	require.NoError(s.T(), err)
	req.Host = s.testApps["node"].Domain
	
	resp, err := client.Do(req)
	require.NoError(s.T(), err)
	resp.Body.Close()
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	
	// Kill the Node.js process to simulate a crash
	// Find and kill the process (this would be handled by underling's process manager)
	killCmd := exec.Command("pkill", "-f", "node-server.js")
	killCmd.Run() // Ignore errors, process might not exist
	
	// Wait for underling to detect the crash and restart
	time.Sleep(8 * time.Second)
	
	// Verify service is back online
	resp, err = client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

func (s *IntegrationTestSuite) TestLoadBalancing() {
	// This test would require modifying the config to have multiple instances
	// For now, we'll test basic round-robin by making multiple requests
	
	client := testutils.HTTPSClient()
	
	responses := make(map[string]int)
	
	// Make multiple requests to the same service
	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/", s.underlingPort), nil)
		require.NoError(s.T(), err)
		req.Host = s.testApps["node"].Domain
		
		resp, err := client.Do(req)
		require.NoError(s.T(), err)
		
		body, err := io.ReadAll(resp.Body)
		require.NoError(s.T(), err)
		resp.Body.Close()
		
		responses[string(body)]++
	}
	
	// With single instance, all responses should be identical
	assert.Len(s.T(), responses, 1)
}

func (s *IntegrationTestSuite) TestCertificateManagement() {
	// Verify certificates were generated and are being used
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	
	for _, app := range s.testApps {
		req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/", s.underlingPort), nil)
		require.NoError(s.T(), err)
		req.Host = app.Domain
		
		resp, err := client.Do(req)
		require.NoError(s.T(), err)
		resp.Body.Close()
		
		// Verify certificate is for correct domain
		assert.NotNil(s.T(), resp.TLS)
		if len(resp.TLS.PeerCertificates) > 0 {
			cert := resp.TLS.PeerCertificates[0]
			assert.Contains(s.T(), cert.DNSNames, app.Domain)
		}
	}
}

func (s *IntegrationTestSuite) TestErrorHandling() {
	client := testutils.HTTPSClient()
	
	// Test request to non-existent domain
	req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/", s.underlingPort), nil)
	require.NoError(s.T(), err)
	req.Host = "nonexistent.test.local"
	
	resp, err := client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()
	
	// Should get 404 or default handler response
	assert.True(s.T(), resp.StatusCode >= 400)
	
	// Test request to invalid path on existing domain
	req.Host = s.testApps["node"].Domain
	req.URL.Path = "/nonexistent-path"
	
	resp, err = client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()
	
	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

func (s *IntegrationTestSuite) TestMetricsCollection() {
	// Make some requests to generate metrics
	client := testutils.HTTPSClient()
	
	for i := 0; i < 5; i++ {
		for _, app := range s.testApps {
			req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/", s.underlingPort), nil)
			require.NoError(s.T(), err)
			req.Host = app.Domain
			
			resp, err := client.Do(req)
			require.NoError(s.T(), err)
			resp.Body.Close()
		}
	}
	
	// Check metrics endpoint
	metricsClient := &http.Client{}
	resp, err := metricsClient.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", s.underlingPort+1000))
	require.NoError(s.T(), err)
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err)
	
	metrics := string(body)
	
	// Verify expected metrics
	assert.Contains(s.T(), metrics, "underling_requests_total")
	assert.Contains(s.T(), metrics, "underling_request_duration")
	assert.Contains(s.T(), metrics, "underling_active_processes")
}

func (s *IntegrationTestSuite) TestConcurrentRequests() {
	client := testutils.HTTPSClient()
	
	numRequests := 50
	done := make(chan bool, numRequests)
	errors := make(chan error, numRequests)
	
	// Make concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(index int) {
			defer func() { done <- true }()
			
			app := s.testApps["node"]
			if index%2 == 0 {
				app = s.testApps["python"]
			}
			
			req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/", s.underlingPort), nil)
			if err != nil {
				errors <- err
				return
			}
			req.Host = app.Domain
			
			resp, err := client.Do(req)
			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("unexpected status: %d", resp.StatusCode)
			}
		}(i)
	}
	
	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
	
	// Check for errors
	select {
	case err := <-errors:
		s.T().Fatalf("Concurrent request failed: %v", err)
	default:
		// No errors, test passed
	}
}

func (s *IntegrationTestSuite) TestConfigReload() {
	// Modify the configuration file to add a new app
	newConfig := strings.Replace(s.readCurrentConfig(), 
		`logging:
  level: "debug"`, 
		`  - name: "echo-app"
    command: "python3"
    args: ["-c", "import http.server; import socketserver; httpd = socketserver.TCPServer(('', 9999), http.server.SimpleHTTPRequestHandler); httpd.serve_forever()"]
    port: 9999
    domains: ["echo.test.local"]
    instances: 1

logging:
  level: "debug"`, -1)
	
	s.testConfig.CreateTestConfig(s.T(), newConfig)
	
	// Send reload signal (this would depend on how underling handles config reloads)
	// For now, we'll just wait and verify the change was detected
	time.Sleep(5 * time.Second)
	
	// Test that the configuration was reloaded
	// This would involve checking metrics or status endpoint
	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/status", s.underlingPort+1000))
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(s.T(), string(body), "echo-app")
	}
}

func (s *IntegrationTestSuite) readCurrentConfig() string {
	data, err := os.ReadFile(s.testConfig.ConfigFile)
	require.NoError(s.T(), err)
	return string(data)
}

// Performance and load testing
func (s *IntegrationTestSuite) TestLoadCapacity() {
	if testing.Short() {
		s.T().Skip("Skipping load test in short mode")
	}
	
	client := testutils.HTTPSClient()
	
	// Test sustained load
	numRequests := 1000
	concurrency := 20
	done := make(chan time.Duration, numRequests)
	sem := make(chan struct{}, concurrency)
	
	start := time.Now()
	
	for i := 0; i < numRequests; i++ {
		go func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			
			requestStart := time.Now()
			
			app := s.testApps["node"]
			req, err := http.NewRequest("GET", fmt.Sprintf("https://127.0.0.1:%d/", s.underlingPort), nil)
			if err != nil {
				done <- 0
				return
			}
			req.Host = app.Domain
			
			resp, err := client.Do(req)
			if err != nil {
				done <- 0
				return
			}
			resp.Body.Close()
			
			done <- time.Since(requestStart)
		}()
	}
	
	// Collect response times
	var responseTimes []time.Duration
	for i := 0; i < numRequests; i++ {
		rt := <-done
		if rt > 0 {
			responseTimes = append(responseTimes, rt)
		}
	}
	
	totalTime := time.Since(start)
	
	// Calculate statistics
	if len(responseTimes) > 0 {
		var totalRT time.Duration
		for _, rt := range responseTimes {
			totalRT += rt
		}
		avgResponseTime := totalRT / time.Duration(len(responseTimes))
		
		s.T().Logf("Load test results:")
		s.T().Logf("- Total requests: %d", numRequests)
		s.T().Logf("- Successful requests: %d", len(responseTimes))
		s.T().Logf("- Total time: %v", totalTime)
		s.T().Logf("- Requests per second: %.2f", float64(len(responseTimes))/totalTime.Seconds())
		s.T().Logf("- Average response time: %v", avgResponseTime)
		
		// Assert reasonable performance
		assert.True(s.T(), avgResponseTime < 100*time.Millisecond, "Average response time should be under 100ms")
		assert.True(s.T(), float64(len(responseTimes))/float64(numRequests) > 0.95, "Success rate should be over 95%")
	}
}