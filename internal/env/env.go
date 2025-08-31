package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvConfig represents environment configuration from .env files
type EnvConfig struct {
	Variables map[string]string `json:"variables" yaml:"variables"`
	Files     []string          `json:"files" yaml:"files"`
}

// LoadDotEnv loads environment variables from .env files following 12-factor principles
func LoadDotEnv(baseDir string) (*EnvConfig, error) {
	config := &EnvConfig{
		Variables: make(map[string]string),
		Files:     []string{},
	}
	
	// Standard .env file hierarchy (12-factor)
	envFiles := []string{
		".env",
		".env.local",
		".env.development",
		".env.development.local",
		".env.production",
		".env.production.local",
	}
	
	for _, filename := range envFiles {
		path := filepath.Join(baseDir, filename)
		if _, err := os.Stat(path); err == nil {
			if err := loadEnvFile(path, config); err != nil {
				return nil, fmt.Errorf("failed to load %s: %w", filename, err)
			}
			config.Files = append(config.Files, path)
		}
	}
	
	return config, nil
}

// loadEnvFile loads a single .env file
func loadEnvFile(path string, config *EnvConfig) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Parse key=value format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format at line %d: %s", lineNum, line)
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes if present
		value = removeQuotes(value)
		
		// Only set if not already defined (precedence: OS env > .env files)
		if _, exists := os.LookupEnv(key); !exists {
			config.Variables[key] = value
		}
	}
	
	return scanner.Err()
}

// ApplyEnv applies environment variables to the current process
func (e *EnvConfig) ApplyEnv() error {
	for key, value := range e.Variables {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}
	return nil
}

// GetEnvForProcess returns environment variables for a specific process
func (e *EnvConfig) GetEnvForProcess(processEnv map[string]string) []string {
	// Start with current environment
	env := os.Environ()
	
	// Apply .env file variables
	for key, value := range e.Variables {
		env = appendOrReplace(env, fmt.Sprintf("%s=%s", key, value))
	}
	
	// Apply process-specific environment
	for key, value := range processEnv {
		env = appendOrReplace(env, fmt.Sprintf("%s=%s", key, value))
	}
	
	return env
}

// SubstituteVariables performs environment variable substitution in strings
func (e *EnvConfig) SubstituteVariables(input string) string {
	result := input
	
	// Replace $VARIABLE and ${VARIABLE} patterns
	for key, value := range e.Variables {
		result = strings.ReplaceAll(result, "$"+key, value)
		result = strings.ReplaceAll(result, "${"+key+"}", value)
	}
	
	// Also substitute from OS environment
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			key, value := parts[0], parts[1]
			result = strings.ReplaceAll(result, "$"+key, value)
			result = strings.ReplaceAll(result, "${"+key+"}", value)
		}
	}
	
	return result
}

// CreateSampleEnvFile creates a sample .env file
func CreateSampleEnvFile(path string) error {
	content := `# Environment variables for 12-factor app configuration
# This file should be excluded from version control (.gitignore)

# Application environment
NODE_ENV=development
RAILS_ENV=development
DJANGO_SETTINGS_MODULE=myproject.settings.development

# Server configuration
PORT=5000
HOST=localhost

# Database URLs (use full connection strings)
DATABASE_URL=postgresql://user:password@localhost:5432/myapp_dev
REDIS_URL=redis://localhost:6379/0

# API keys and secrets (never commit these to git)
SECRET_KEY=your-secret-key-here
API_KEY=your-api-key-here
JWT_SECRET=your-jwt-secret-here

# External service URLs
EXTERNAL_API_URL=https://api.example.com
CDN_URL=https://cdn.example.com

# Feature flags
FEATURE_NEW_UI=true
DEBUG=true
VERBOSE_LOGGING=false

# Email configuration
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=noreply@example.com
SMTP_PASSWORD=smtp-password

# File storage
UPLOAD_PATH=./uploads
MAX_FILE_SIZE=10485760
`

	return os.WriteFile(path, []byte(content), 0644)
}

// Validate checks environment configuration for common issues
func (e *EnvConfig) Validate() []string {
	var warnings []string
	
	// Check for common security issues
	for key, value := range e.Variables {
		// Check for passwords/secrets in development
		if strings.Contains(strings.ToLower(key), "password") && value == "password" {
			warnings = append(warnings, fmt.Sprintf("Default password detected for %s", key))
		}
		
		if strings.Contains(strings.ToLower(key), "secret") && len(value) < 20 {
			warnings = append(warnings, fmt.Sprintf("Short secret key detected for %s", key))
		}
		
		// Check for localhost in production URLs
		if strings.Contains(strings.ToLower(key), "url") && strings.Contains(value, "localhost") {
			warnings = append(warnings, fmt.Sprintf("Localhost URL in %s may not work in production", key))
		}
	}
	
	return warnings
}

// Helper functions

func removeQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func appendOrReplace(env []string, newVar string) []string {
	key := strings.SplitN(newVar, "=", 2)[0]
	
	for i, existing := range env {
		if strings.HasPrefix(existing, key+"=") {
			env[i] = newVar
			return env
		}
	}
	
	return append(env, newVar)
}