package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/gleicon/guvnor/internal/discovery"
)

// Config represents the main configuration structure
type Config struct {
	Server ServerConfig `yaml:"server"`
	Apps   []AppConfig  `yaml:"apps"`
	TLS    TLSConfig    `yaml:"tls"`
}

// ServerConfig contains server-wide configuration
type ServerConfig struct {
	HTTPPort        int           `yaml:"http_port" default:"80"`
	HTTPSPort       int           `yaml:"https_port" default:"443"`
	ReadTimeout     time.Duration `yaml:"read_timeout" default:"30s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" default:"30s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" default:"30s"`
	LogLevel        string        `yaml:"log_level" default:"info"`
}

// AppConfig defines configuration for an individual application
type AppConfig struct {
	Name          string            `yaml:"name"`
	Domain        string            `yaml:"domain"`
	Port          int               `yaml:"port"`
	Command       string            `yaml:"command"`
	Args          []string          `yaml:"args,omitempty"`
	WorkingDir    string            `yaml:"working_dir,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	HealthCheck   HealthCheckConfig `yaml:"health_check"`
	RestartPolicy RestartPolicy     `yaml:"restart_policy"`
}

// HealthCheckConfig defines health check parameters for an app
type HealthCheckConfig struct {
	Enabled  bool          `yaml:"enabled" default:"true"`
	Path     string        `yaml:"path" default:"/health"`
	Interval time.Duration `yaml:"interval" default:"30s"`
	Timeout  time.Duration `yaml:"timeout" default:"5s"`
	Retries  int           `yaml:"retries" default:"3"`
}

// RestartPolicy defines how the app should be restarted on failure
type RestartPolicy struct {
	Enabled    bool          `yaml:"enabled" default:"true"`
	MaxRetries int           `yaml:"max_retries" default:"3"`
	Backoff    time.Duration `yaml:"backoff" default:"5s"`
}

// TLSConfig contains TLS and Let's Encrypt configuration
type TLSConfig struct {
	Enabled    bool     `yaml:"enabled" default:"true"`
	AutoCert   bool     `yaml:"auto_cert" default:"true"`
	CertDir    string   `yaml:"cert_dir" default:"/var/lib/guvnor/certs"`
	Email      string   `yaml:"email"`
	Domains    []string `yaml:"domains,omitempty"`
	Staging    bool     `yaml:"staging" default:"false"`
	ForceHTTPS bool     `yaml:"force_https" default:"true"`
}

// Load loads configuration from a file, applying defaults
func Load(configFile string) (*Config, error) {
	// Create default config
	config := &Config{
		Server: ServerConfig{
			HTTPPort:        80,
			HTTPSPort:       443,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			LogLevel:        "info",
		},
		TLS: TLSConfig{
			Enabled:    true,
			AutoCert:   true,
			CertDir:    "/var/lib/guvnor/certs",
			Staging:    false,
			ForceHTTPS: true,
		},
	}

	// If config file exists, load it
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			data, err := os.ReadFile(configFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}

			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// Validate performs configuration validation
func (c *Config) Validate() error {
	if c.Server.HTTPPort <= 0 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.Server.HTTPPort)
	}

	if c.Server.HTTPSPort <= 0 || c.Server.HTTPSPort > 65535 {
		return fmt.Errorf("invalid HTTPS port: %d", c.Server.HTTPSPort)
	}

	// Validate apps
	domainMap := make(map[string]string)
	portMap := make(map[int]string)

	for i, app := range c.Apps {
		if app.Name == "" {
			return fmt.Errorf("app name cannot be empty")
		}

		if app.Domain == "" {
			// Auto-generate domain for local development
			c.Apps[i].Domain = fmt.Sprintf("%s.localhost", strings.ToLower(app.Name))
			app.Domain = c.Apps[i].Domain // Update local var for validation
		}

		if app.Port <= 0 || app.Port > 65535 {
			return fmt.Errorf("app %s: invalid port %d", app.Name, app.Port)
		}

		if app.Command == "" {
			return fmt.Errorf("app %s: command cannot be empty", app.Name)
		}

		// Check for duplicate domains
		if existingApp, exists := domainMap[app.Domain]; exists {
			return fmt.Errorf("domain %s is used by both %s and %s", app.Domain, existingApp, app.Name)
		}
		domainMap[app.Domain] = app.Name

		// Check for duplicate ports
		if existingApp, exists := portMap[app.Port]; exists {
			return fmt.Errorf("port %d is used by both %s and %s", app.Port, existingApp, app.Name)
		}
		portMap[app.Port] = app.Name

		// Set defaults for health check
		if app.HealthCheck.Path == "" {
			c.Apps[i].HealthCheck.Path = "/health"
		}
		if app.HealthCheck.Interval == 0 {
			c.Apps[i].HealthCheck.Interval = 30 * time.Second
		}
		if app.HealthCheck.Timeout == 0 {
			c.Apps[i].HealthCheck.Timeout = 5 * time.Second
		}
		if app.HealthCheck.Retries == 0 {
			c.Apps[i].HealthCheck.Retries = 3
		}

		// Set defaults for restart policy
		if app.RestartPolicy.MaxRetries == 0 {
			c.Apps[i].RestartPolicy.MaxRetries = 3
		}
		if app.RestartPolicy.Backoff == 0 {
			c.Apps[i].RestartPolicy.Backoff = 5 * time.Second
		}
	}

	return nil
}

// CreateSample creates a sample configuration file
func CreateSample(filename string) error {
	sample := &Config{
		Server: ServerConfig{
			HTTPPort:        80,
			HTTPSPort:       443,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			LogLevel:        "info",
		},
		Apps: []AppConfig{
			{
				Name:       "web-app",
				Domain:     "myapp.example.com",
				Port:       3000,
				Command:    "node",
				Args:       []string{"server.js"},
				WorkingDir: "/opt/myapp",
				Environment: map[string]string{
					"NODE_ENV": "production",
					"PORT":     "3000",
				},
				HealthCheck: HealthCheckConfig{
					Enabled:  true,
					Path:     "/health",
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					Retries:  3,
				},
				RestartPolicy: RestartPolicy{
					Enabled:    true,
					MaxRetries: 3,
					Backoff:    5 * time.Second,
				},
			},
			{
				Name:       "api-service",
				Domain:     "api.example.com",
				Port:       8000,
				Command:    "python",
				Args:       []string{"-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"},
				WorkingDir: "/opt/api",
				Environment: map[string]string{
					"PYTHONPATH": "/opt/api",
				},
				HealthCheck: HealthCheckConfig{
					Enabled:  true,
					Path:     "/health",
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
					Retries:  3,
				},
				RestartPolicy: RestartPolicy{
					Enabled:    true,
					MaxRetries: 3,
					Backoff:    5 * time.Second,
				},
			},
		},
		TLS: TLSConfig{
			Enabled:    true,
			AutoCert:   true,
			CertDir:    "/var/lib/guvnor/certs",
			Email:      "admin@example.com",
			Staging:    false,
			ForceHTTPS: true,
		},
	}

	data, err := yaml.Marshal(sample)
	if err != nil {
		return fmt.Errorf("failed to marshal sample config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write sample config: %w", err)
	}

	return nil
}

// CreateSmart creates a smart configuration from discovered apps
// This is the core uv-inspired functionality
func CreateSmart(apps []*discovery.App) *Config {
	config := &Config{
		Server: ServerConfig{
			HTTPPort:        8080, // Use non-privileged port for dev
			HTTPSPort:       8443, // Use non-privileged port for dev
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 10 * time.Second, // Faster shutdown for dev
			LogLevel:        "info",
		},
		TLS: TLSConfig{
			Enabled:    false, // Disable TLS for local dev by default
			AutoCert:   false,
			CertDir:    "./certs",
			Staging:    true,  // Use staging for safety
			ForceHTTPS: false, // Allow HTTP for dev
		},
	}

	// Convert discovery apps to config apps
	for _, app := range apps {
		configApp := AppConfig{
			Name:        app.Name,
			Domain:      generateSmartDomain(app),
			Port:        app.Port,
			Command:     app.Command,
			Args:        convertArgs(app.Args, app.Port),
			WorkingDir:  app.Path,
			Environment: convertEnvironment(app.Env, app.Port),
			HealthCheck: HealthCheckConfig{
				Enabled:  true,
				Path:     app.HealthCheck,
				Interval: 30 * time.Second,
				Timeout:  5 * time.Second,
				Retries:  3,
			},
			RestartPolicy: RestartPolicy{
				Enabled:    true,
				MaxRetries: 5,               // Be more forgiving in development
				Backoff:    3 * time.Second, // Faster restart for dev
			},
		}

		config.Apps = append(config.Apps, configApp)
	}

	return config
}

// CreateSmartConfig creates and writes a smart configuration file
func CreateSmartConfig(filename string, apps []*discovery.App) error {
	config := CreateSmart(apps)

	// Add header comment to generated config
	header := `# Guv'nor Configuration - Generated Automatically
# Edit this file to customize your application deployment
# Run 'guvnor start' to start all applications

`

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal smart config: %w", err)
	}

	// Write with header comment
	fullContent := header + string(data)
	if err := os.WriteFile(filename, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write smart config: %w", err)
	}

	return nil
}

// Smart helper functions

func generateSmartDomain(app *discovery.App) string {
	if app.Domain != "" {
		return app.Domain
	}

	// For development, use localhost with app name
	return fmt.Sprintf("%s.localhost", strings.ToLower(app.Name))
}

func convertArgs(args []string, port int) []string {
	converted := make([]string, len(args))

	for i, arg := range args {
		// Replace $PORT with actual port number
		if strings.Contains(arg, "$PORT") {
			converted[i] = strings.ReplaceAll(arg, "$PORT", fmt.Sprintf("%d", port))
		} else {
			converted[i] = arg
		}
	}

	return converted
}

func convertEnvironment(env map[string]string, port int) map[string]string {
	converted := make(map[string]string)

	for key, value := range env {
		// Replace $PORT with actual port number
		if strings.Contains(value, "$PORT") {
			converted[key] = strings.ReplaceAll(value, "$PORT", fmt.Sprintf("%d", port))
		} else if key == "PORT" && value == "$PORT" {
			// Special case for PORT environment variable
			converted[key] = fmt.Sprintf("%d", port)
		} else {
			converted[key] = value
		}
	}

	return converted
}

// CreateProductionConfig creates a production-ready configuration
func CreateProductionConfig(filename string, apps []*discovery.App, domain string, email string) error {
	config := CreateSmart(apps)

	// Override for production settings
	config.Server.HTTPPort = 80
	config.Server.HTTPSPort = 443
	config.Server.LogLevel = "warn"

	config.TLS.Enabled = true
	config.TLS.AutoCert = true
	config.TLS.CertDir = "/var/lib/guvnor/certs"
	config.TLS.Email = email
	config.TLS.Staging = false
	config.TLS.ForceHTTPS = true

	// Update domains for production
	for i, app := range config.Apps {
		if domain != "" {
			if len(config.Apps) == 1 {
				// Single app gets the main domain
				config.Apps[i].Domain = domain
			} else {
				// Multiple apps get subdomains
				config.Apps[i].Domain = fmt.Sprintf("%s.%s", app.Name, domain)
			}
		}

		// Production-specific health check adjustments
		config.Apps[i].HealthCheck.Interval = 60 * time.Second  // Less frequent checks
		config.Apps[i].RestartPolicy.MaxRetries = 3             // Less forgiving in production
		config.Apps[i].RestartPolicy.Backoff = 10 * time.Second // Longer backoff
	}

	// Add all domains to TLS config
	var domains []string
	for _, app := range config.Apps {
		domains = append(domains, app.Domain)
	}
	config.TLS.Domains = domains

	header := `# Guv'nor Production Configuration - Generated Automatically
# This configuration is optimized for production deployment
# Make sure to review TLS settings and domain configuration

`

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal production config: %w", err)
	}

	fullContent := header + string(data)
	if err := os.WriteFile(filename, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write production config: %w", err)
	}

	return nil
}

// WriteConfig writes a configuration to a file
func WriteConfig(config *Config, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	header := `# Guv'nor Configuration
# Process manager with reverse proxy and TLS

`

	fullContent := header + string(data)
	if err := os.WriteFile(filename, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
