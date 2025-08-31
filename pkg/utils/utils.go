package utils

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

// IsPortAvailable checks if a port is available for use
func IsPortAvailable(port int) bool {
	address := net.JoinHostPort("", strconv.Itoa(port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return os.MkdirAll(dirPath, 0755)
	}
	return nil
}

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// GetAbsolutePath returns the absolute path of a file
func GetAbsolutePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	
	return filepath.Join(cwd, path), nil
}

// ValidateDomain performs basic domain validation
func ValidateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	
	if len(domain) > 253 {
		return fmt.Errorf("domain is too long")
	}
	
	// Basic validation - could be enhanced with regex
	if domain[0] == '.' || domain[len(domain)-1] == '.' {
		return fmt.Errorf("domain cannot start or end with a dot")
	}
	
	return nil
}

// ValidatePort validates that a port number is valid
func ValidatePort(port int) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	
	return nil
}