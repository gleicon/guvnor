package server

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/gleicon/guvnor/internal/config"
	"github.com/gleicon/guvnor/internal/procfile"
	"github.com/gleicon/guvnor/internal/proxy"
)

// Server represents the main Guv'nor server that combines Procfile processing
// with reverse proxy functionality
type Server struct {
	config      *config.Config
	procfile    *procfile.Procfile
	proxyServer *proxy.Server
	logger      *logrus.Logger
}

// New creates a new Guv'nor server from configuration and procfile
func New(cfg *config.Config, pf *procfile.Procfile, logger *logrus.Logger) *Server {
	return &Server{
		config:   cfg,
		procfile: pf,
		logger:   logger,
	}
}

// Start starts the server and all processes from the Procfile
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting Guv'nor server")

	// Convert Procfile processes to config apps if config doesn't have apps defined
	if len(s.config.Apps) == 0 {
		if err := s.convertProcfileToConfig(); err != nil {
			return fmt.Errorf("failed to convert Procfile to config: %w", err)
		}
	}

	// Create and start proxy server
	proxyServer, err := proxy.NewServer(ctx, s.config, s.logger)
	if err != nil {
		return fmt.Errorf("failed to create proxy server: %w", err)
	}

	s.proxyServer = proxyServer

	// Start the proxy server (which will start all processes)
	if err := s.proxyServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start proxy server: %w", err)
	}

	return nil
}

// Stop stops the server and all processes
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Guv'nor server")

	if s.proxyServer != nil {
		return s.proxyServer.Stop(ctx)
	}

	return nil
}

// convertProcfileToConfig converts Procfile processes to config.AppConfig entries
func (s *Server) convertProcfileToConfig() error {
	s.logger.Info("Converting Procfile processes to configuration")

	for _, process := range s.procfile.Processes {
		// Use the process command substitution from Procfile
		command := s.procfile.SubstituteCommand(&process)
		
		// Parse command into command and args
		cmdParts, err := parseCommand(command)
		if err != nil {
			s.logger.WithError(err).WithField("process", process.Name).Warn("Failed to parse process command")
			continue
		}

		if len(cmdParts) == 0 {
			s.logger.WithField("process", process.Name).Warn("Empty command after parsing")
			continue
		}

		// Create app config from process
		appConfig := config.AppConfig{
			Name:       process.Name,
			Domain:     generateDomainForProcess(process.Name, s.config.Server.HTTPPort),
			Port:       process.Port,
			Command:    cmdParts[0],
			Args:       cmdParts[1:],
			WorkingDir: getCurrentWorkingDir(),
			Environment: mergeEnvironments(s.procfile.GetProcessEnvironment(&process), process.Env),
			HealthCheck: config.HealthCheckConfig{
				Enabled:  needsHealthCheck(process.Name),
				Path:     "/health",
				Interval: 30000000000, // 30s in nanoseconds
				Timeout:  5000000000,  // 5s in nanoseconds
				Retries:  3,
			},
			RestartPolicy: config.RestartPolicy{
				Enabled:    true,
				MaxRetries: 3,
				Backoff:    5000000000, // 5s in nanoseconds
			},
		}

		s.config.Apps = append(s.config.Apps, appConfig)
		
		s.logger.WithFields(logrus.Fields{
			"process": process.Name,
			"command": appConfig.Command,
			"args":    appConfig.Args,
			"port":    appConfig.Port,
			"domain":  appConfig.Domain,
		}).Info("Added process to configuration")
	}

	s.logger.WithField("total_apps", len(s.config.Apps)).Info("Procfile conversion complete")
	return nil
}

// Helper functions

func parseCommand(command string) ([]string, error) {
	// Simple shell-style parsing - split on spaces but respect quotes
	// This is a simplified version - for production you might want a proper shell parser
	var parts []string
	var current string
	inQuotes := false
	
	for i, char := range command {
		switch char {
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if inQuotes {
				current += string(char)
			} else {
				if current != "" {
					parts = append(parts, current)
					current = ""
				}
			}
		default:
			current += string(char)
		}
		
		// Add the last part if we're at the end
		if i == len(command)-1 && current != "" {
			parts = append(parts, current)
		}
	}
	
	return parts, nil
}

func generateDomainForProcess(processName string, httpPort int) string {
	// For local development, use localhost with process name
	if httpPort != 80 && httpPort != 443 {
		return fmt.Sprintf("%s.localhost:%d", processName, httpPort)
	}
	return fmt.Sprintf("%s.localhost", processName)
}

func getCurrentWorkingDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func mergeEnvironments(procfileEnv []string, processEnv map[string]string) map[string]string {
	env := make(map[string]string)
	
	// Add process-specific environment variables
	for k, v := range processEnv {
		env[k] = v
	}
	
	// Note: procfileEnv is already merged with system environment
	// We could parse it here if needed, but the process manager
	// will use the full environment from GetProcessEnvironment()
	
	return env
}

func needsHealthCheck(processName string) bool {
	// Enable health checks for web-facing processes
	switch processName {
	case "web", "api", "server", "frontend", "backend", "app":
		return true
	case "worker", "job", "jobs", "clock", "scheduler", "cron":
		return false
	default:
		return true // Default to enabled
	}
}