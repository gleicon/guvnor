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
	Hostname      string            `yaml:"hostname,omitempty"` // NEW: for virtual host routing
	Domain        string            `yaml:"domain,omitempty"`   // DEPRECATED: use hostname instead
	Port          int               `yaml:"port"`
	Command       string            `yaml:"command"`
	Args          []string          `yaml:"args,omitempty"`
	WorkingDir    string            `yaml:"working_dir,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	HealthCheck   HealthCheckConfig `yaml:"health_check"`
	RestartPolicy RestartPolicy     `yaml:"restart_policy"`
	TLS           AppTLSConfig      `yaml:"tls,omitempty"` // NEW: per-app TLS config
}

// AppTLSConfig contains per-app TLS configuration
type AppTLSConfig struct {
	Enabled   bool   `yaml:"enabled" default:"false"`
	AutoCert  bool   `yaml:"auto_cert" default:"true"`
	Email     string `yaml:"email,omitempty"`
	Staging   bool   `yaml:"staging" default:"false"`
	CertFile  string `yaml:"cert_file,omitempty"`  // For manual certs
	KeyFile   string `yaml:"key_file,omitempty"`   // For manual certs
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

// TLSConfig contains global TLS and Let's Encrypt configuration
type TLSConfig struct {
	Enabled    bool     `yaml:"enabled" default:"true"`
	AutoCert   bool     `yaml:"auto_cert" default:"true"`
	CertDir    string   `yaml:"cert_dir" default:"/var/lib/guvnor/certs"`
	Email      string   `yaml:"email,omitempty"`      // Fallback email for apps without one
	Domains    []string `yaml:"domains,omitempty"`    // DEPRECATED: domains now per-app
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
	hostnameMap := make(map[string]string)
	portMap := make(map[int]string)

	for i, app := range c.Apps {
		if app.Name == "" {
			return fmt.Errorf("app name cannot be empty")
		}

		// Handle hostname vs domain (backward compatibility)
		hostname := app.Hostname
		if hostname == "" && app.Domain != "" {
			// Use domain if hostname not specified (backward compatibility)
			hostname = app.Domain
			c.Apps[i].Hostname = hostname
		} else if hostname == "" {
			// Auto-generate hostname: app-name.localhost
			hostname = fmt.Sprintf("%s.localhost", strings.ToLower(app.Name))
			c.Apps[i].Hostname = hostname
		}

		// Auto-assign port if not specified
		if app.Port <= 0 {
			c.Apps[i].Port = c.findAvailablePort(portMap, 3000+i*1000)
		} else if app.Port > 65535 {
			return fmt.Errorf("app %s: invalid port %d", app.Name, app.Port)
		}
		
		// Update local var for validation
		app.Port = c.Apps[i].Port
		hostname = c.Apps[i].Hostname

		if app.Command == "" {
			return fmt.Errorf("app %s: command cannot be empty", app.Name)
		}

		// Check for duplicate hostnames
		if existingApp, exists := hostnameMap[hostname]; exists {
			return fmt.Errorf("hostname %s is used by both %s and %s", hostname, existingApp, app.Name)
		}
		hostnameMap[hostname] = app.Name

		// Check for duplicate ports
		if existingApp, exists := portMap[app.Port]; exists {
			return fmt.Errorf("port %d is used by both %s and %s", app.Port, existingApp, app.Name)
		}
		portMap[app.Port] = app.Name

		// Validate per-app TLS configuration
		if app.TLS.Enabled && app.TLS.AutoCert && app.TLS.Email == "" && c.TLS.Email == "" {
			return fmt.Errorf("app %s: email required for TLS auto-cert (set in app.tls.email or global tls.email)", app.Name)
		}

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

// findAvailablePort finds the next available port starting from startPort
func (c *Config) findAvailablePort(portMap map[int]string, startPort int) int {
	port := startPort
	for {
		if _, exists := portMap[port]; !exists && port <= 65535 {
			return port
		}
		port++
		if port > 65535 {
			// Wrap around to find any available port starting from 3000
			for p := 3000; p < startPort; p++ {
				if _, exists := portMap[p]; !exists {
					return p
				}
			}
			// If we still can't find one, return original (will cause validation error)
			return startPort
		}
	}
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
				Hostname:   "myapp.example.com",
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
				TLS: AppTLSConfig{
					Enabled:  true,
					AutoCert: true,
					Email:    "admin@example.com",
					Staging:  false,
				},
			},
			{
				Name:       "api-service",
				Hostname:   "api.example.com",
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
				TLS: AppTLSConfig{
					Enabled:  true,
					AutoCert: true,
					Email:    "api-admin@example.com",
					Staging:  false,
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
	isOnlyApp := len(apps) == 1
	for _, app := range apps {
		configApp := AppConfig{
			Name:        app.Name,
			Hostname:    generateSmartDomainForSingleApp(app, isOnlyApp),
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

	// Create custom YAML with helpful comments
	yamlContent := generateCommentedYAML(config, apps)
	
	if err := os.WriteFile(filename, []byte(yamlContent), 0644); err != nil {
		return fmt.Errorf("failed to write smart config: %w", err)
	}

	return nil
}

// generateCommentedYAML creates YAML with helpful comments for users
func generateCommentedYAML(config *Config, apps []*discovery.App) string {
	var buf strings.Builder
	
	// Header comment
	buf.WriteString("# Guv'nor Configuration - Generated Automatically\n")
	buf.WriteString("# Edit this file to customize your application deployment\n") 
	buf.WriteString("# Run 'guvnor start' to start all applications\n\n")
	
	// Server section
	buf.WriteString("server:\n")
	buf.WriteString(fmt.Sprintf("  http_port: %d     # Non-privileged port for development\n", config.Server.HTTPPort))
	buf.WriteString(fmt.Sprintf("  https_port: %d    # HTTPS port (if TLS enabled)\n", config.Server.HTTPSPort))
	buf.WriteString(fmt.Sprintf("  log_level: %s       # info, warn, error, debug\n\n", config.Server.LogLevel))
	
	// Apps section
	buf.WriteString("apps:\n")
	isOnlyApp := len(apps) == 1
	
	for i, app := range config.Apps {
		buf.WriteString(fmt.Sprintf("  - name: %s\n", app.Name))
		
		// Hostname comment based on whether it's single or multi-app
		if isOnlyApp {
			buf.WriteString(fmt.Sprintf("    hostname: %s    # Access via http://localhost:8080/\n", app.Hostname))
			buf.WriteString("                       # Change to 'my-app.localhost' for subdomain routing\n")
		} else {
			buf.WriteString(fmt.Sprintf("    hostname: %s  # Access via http://%s:8080/\n", app.Hostname, app.Hostname))
		}
		
		buf.WriteString(fmt.Sprintf("    port: %d             # Backend port (your app listens here)\n", app.Port))
		buf.WriteString(fmt.Sprintf("    command: %s\n", app.Command))
		
		if len(app.Args) > 0 {
			buf.WriteString("    args:\n")
			for _, arg := range app.Args {
				buf.WriteString(fmt.Sprintf("      - \"%s\"\n", arg))
			}
		}
		
		if app.WorkingDir != "" {
			buf.WriteString(fmt.Sprintf("    working_dir: %s\n", app.WorkingDir))
		}
		
		if len(app.Environment) > 0 {
			buf.WriteString("    environment:\n")
			for k, v := range app.Environment {
				buf.WriteString(fmt.Sprintf("      %s: \"%s\"\n", k, v))
			}
		}
		
		// Health check
		buf.WriteString("    health_check:\n")
		buf.WriteString(fmt.Sprintf("      enabled: %t\n", app.HealthCheck.Enabled))
		buf.WriteString(fmt.Sprintf("      path: %s          # Health check endpoint\n", app.HealthCheck.Path))
		buf.WriteString(fmt.Sprintf("      interval: %s       # How often to check\n", app.HealthCheck.Interval))
		
		// Restart policy
		buf.WriteString("    restart_policy:\n")
		buf.WriteString(fmt.Sprintf("      enabled: %t\n", app.RestartPolicy.Enabled))
		buf.WriteString(fmt.Sprintf("      max_retries: %d      # Retries before giving up\n", app.RestartPolicy.MaxRetries))
		buf.WriteString(fmt.Sprintf("      backoff: %s        # Wait time between retries\n", app.RestartPolicy.Backoff))
		
		if i < len(config.Apps)-1 {
			buf.WriteString("\n")
		}
	}
	
	// TLS section
	buf.WriteString("\n# TLS/HTTPS Configuration\n")
	buf.WriteString("tls:\n")
	buf.WriteString(fmt.Sprintf("  enabled: %t           # Set to true for production HTTPS\n", config.TLS.Enabled))
	buf.WriteString(fmt.Sprintf("  auto_cert: %t         # Automatic Let's Encrypt certificates\n", config.TLS.AutoCert))
	buf.WriteString(fmt.Sprintf("  cert_dir: %s        # Where to store certificates\n", config.TLS.CertDir))
	buf.WriteString(fmt.Sprintf("  staging: %t           # Use Let's Encrypt staging (for testing)\n", config.TLS.Staging))
	buf.WriteString(fmt.Sprintf("  force_https: %t       # Redirect HTTP to HTTPS\n", config.TLS.ForceHTTPS))
	if config.TLS.Email != "" {
		buf.WriteString(fmt.Sprintf("  email: %s       # Contact for Let's Encrypt\n", config.TLS.Email))
	} else {
		buf.WriteString("  # email: your@email.com   # Required for Let's Encrypt (uncomment & set)\n")
	}
	
	// Footer comment
	buf.WriteString("\n# Usage:\n")
	if isOnlyApp {
		buf.WriteString("# - Start: guvnor start\n")
		buf.WriteString("# - View logs: guvnor logs\n")
		buf.WriteString("# - Check status: guvnor status\n")
		buf.WriteString(fmt.Sprintf("# - Access your app: http://localhost:8080/\n"))
	} else {
		buf.WriteString("# - Start all apps: guvnor start\n")
		buf.WriteString("# - Start specific app: guvnor start app-name\n")
		buf.WriteString("# - View logs: guvnor logs [app-name]\n")
		buf.WriteString("# - Check status: guvnor status [app-name]\n")
		for _, app := range config.Apps {
			buf.WriteString(fmt.Sprintf("# - Access %s: http://%s:8080/\n", app.Name, app.Hostname))
		}
	}
	
	return buf.String()
}

// Smart helper functions

func generateSmartDomain(app *discovery.App) string {
	if app.Domain != "" {
		return app.Domain
	}

	// For development, use localhost with app name
	return fmt.Sprintf("%s.localhost", strings.ToLower(app.Name))
}

// generateSmartDomainForSingleApp provides better defaults for single app setups
func generateSmartDomainForSingleApp(app *discovery.App, isOnlyApp bool) string {
	if app.Domain != "" {
		return app.Domain
	}

	// If this is the only app, use plain localhost for easy access
	if isOnlyApp {
		return "localhost"
	}

	// Multiple apps: use app-specific subdomain
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

	// Update hostnames for production
	for i, app := range config.Apps {
		if domain != "" {
			if len(config.Apps) == 1 {
				// Single app gets the main domain
				config.Apps[i].Hostname = domain
			} else {
				// Multiple apps get subdomains
				config.Apps[i].Hostname = fmt.Sprintf("%s.%s", app.Name, domain)
			}
		}

		// Enable TLS for production apps
		config.Apps[i].TLS = AppTLSConfig{
			Enabled:  true,
			AutoCert: true,
			Email:    email,
			Staging:  false,
		}

		// Production-specific health check adjustments
		config.Apps[i].HealthCheck.Interval = 60 * time.Second  // Less frequent checks
		config.Apps[i].RestartPolicy.MaxRetries = 3             // Less forgiving in production
		config.Apps[i].RestartPolicy.Backoff = 10 * time.Second // Longer backoff
	}

	// Add all hostnames to TLS config (for backward compatibility)
	var domains []string
	for _, app := range config.Apps {
		if app.Hostname != "" {
			domains = append(domains, app.Hostname)
		}
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
