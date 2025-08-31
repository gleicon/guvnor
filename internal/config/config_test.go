package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_LoadFromFile(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.yaml")
	
	configYAML := `
server:
  http_port: 8080
  https_port: 8443
  log_level: "info"

apps:
  - name: "test-app"
    command: "node"
    args: ["server.js"]
    port: 3000
    domain: "test.example.com"

tls:
  enabled: true
  auto_cert: true
  cert_dir: "/tmp/certs"
`
	
	err := os.WriteFile(configPath, []byte(configYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// Load the config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// Basic validation
	if cfg.Server.HTTPPort != 8080 {
		t.Errorf("Expected HTTPPort 8080, got %d", cfg.Server.HTTPPort)
	}
	
	if len(cfg.Apps) != 1 {
		t.Errorf("Expected 1 app, got %d", len(cfg.Apps))
	}
	
	if len(cfg.Apps) > 0 {
		app := cfg.Apps[0]
		if app.Name != "test-app" {
			t.Errorf("Expected app name 'test-app', got %s", app.Name)
		}
		if app.Command != "node" {
			t.Errorf("Expected command 'node', got %s", app.Command)
		}
	}
}

func TestConfig_Validate(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			HTTPPort:  8080,
			HTTPSPort: 8443,
			LogLevel:  "info",
		},
		Apps: []AppConfig{
			{
				Name:    "test-app",
				Domain:  "test.example.com",
				Port:    3000,
				Command: "node",
				Args:    []string{"server.js"},
			},
		},
		TLS: TLSConfig{
			Enabled:  true,
			AutoCert: true,
		},
	}
	
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Valid config should not return error: %v", err)
	}
}