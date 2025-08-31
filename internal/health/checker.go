package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gleicon/guvnor/internal/config"
	"github.com/gleicon/guvnor/internal/process"
)

// Status represents health check status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusUnknown   Status = "unknown"
)

// Result represents the result of a health check
type Result struct {
	Status     Status        `json:"status"`
	StatusCode int           `json:"status_code,omitempty"`
	Response   string        `json:"response,omitempty"`
	Error      string        `json:"error,omitempty"`
	Timestamp  time.Time     `json:"timestamp"`
	Duration   time.Duration `json:"duration"`
}

// Checker manages health checks for all applications
type Checker struct {
	processManager *process.Manager
	results        map[string]*Result
	logger         *logrus.Entry
	mu             sync.RWMutex
	client         *http.Client
}

// NewChecker creates a new health checker
func NewChecker(processManager *process.Manager, logger *logrus.Logger) *Checker {
	return &Checker{
		processManager: processManager,
		results:        make(map[string]*Result),
		logger:         logger.WithField("component", "health-checker"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start starts the health checking process for all configured applications
func (c *Checker) Start(ctx context.Context) {
	c.logger.Info("Starting health checker")
	
	processes := c.processManager.ListProcesses()
	
	for appName, proc := range processes {
		if proc.Config.HealthCheck.Enabled {
			go c.checkApp(ctx, appName, proc.Config.HealthCheck)
		}
	}
}

// GetResult returns the latest health check result for an app
func (c *Checker) GetResult(appName string) (*Result, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result, exists := c.results[appName]
	if exists {
		// Return a copy to avoid race conditions
		resultCopy := *result
		return &resultCopy, true
	}
	
	return nil, false
}

// GetAllResults returns all health check results
func (c *Checker) GetAllResults() map[string]*Result {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	results := make(map[string]*Result)
	for appName, result := range c.results {
		// Return copies to avoid race conditions
		resultCopy := *result
		results[appName] = &resultCopy
	}
	
	return results
}

// CheckApp performs a single health check for an application
func (c *Checker) CheckApp(appName string, healthCheck config.HealthCheckConfig, port int) *Result {
	start := time.Now()
	result := &Result{
		Status:    StatusUnknown,
		Timestamp: start,
	}
	
	// Build health check URL
	url := fmt.Sprintf("http://localhost:%d%s", port, healthCheck.Path)
	
	// Create request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), healthCheck.Timeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		result.Status = StatusUnhealthy
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	
	// Add health check headers
	req.Header.Set("User-Agent", "guvnor-healthcheck/1.0")
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	
	// Perform request
	resp, err := c.client.Do(req)
	if err != nil {
		result.Status = StatusUnhealthy
		result.Error = fmt.Sprintf("request failed: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	defer resp.Body.Close()
	
	result.StatusCode = resp.StatusCode
	result.Duration = time.Since(start)
	
	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = StatusHealthy
	} else {
		result.Status = StatusUnhealthy
		result.Error = fmt.Sprintf("unhealthy status code: %d", resp.StatusCode)
	}
	
	// Read response body (limited to avoid memory issues)
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	if n > 0 {
		result.Response = string(body[:n])
	}
	
	return result
}

// checkApp runs continuous health checks for an application
func (c *Checker) checkApp(ctx context.Context, appName string, healthCheck config.HealthCheckConfig) {
	logger := c.logger.WithField("app", appName)
	logger.WithField("interval", healthCheck.Interval).Info("Starting health checks")
	
	ticker := time.NewTicker(healthCheck.Interval)
	defer ticker.Stop()
	
	// Perform initial check after a short delay to let the app start
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}
	
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping health checks")
			return
		case <-ticker.C:
			c.performCheck(ctx, appName, healthCheck)
		}
	}
}

// performCheck performs a health check and handles the result
func (c *Checker) performCheck(ctx context.Context, appName string, healthCheck config.HealthCheckConfig) {
	logger := c.logger.WithField("app", appName)
	
	// Get the process to check if it's running
	proc, exists := c.processManager.GetProcess(appName)
	if !exists || !proc.IsRunning() {
		// Process not running, mark as unhealthy
		result := &Result{
			Status:    StatusUnhealthy,
			Error:     "process not running",
			Timestamp: time.Now(),
		}
		
		c.mu.Lock()
		c.results[appName] = result
		c.mu.Unlock()
		
		logger.Debug("Process not running, skipping health check")
		return
	}
	
	// Perform the health check
	result := c.CheckApp(appName, healthCheck, proc.Config.Port)
	
	// Store the result
	c.mu.Lock()
	previousResult := c.results[appName]
	c.results[appName] = result
	c.mu.Unlock()
	
	// Log status changes
	if previousResult == nil || previousResult.Status != result.Status {
		logger.WithFields(logrus.Fields{
			"status":      result.Status,
			"status_code": result.StatusCode,
			"duration":    result.Duration,
			"error":       result.Error,
		}).Info("Health check status changed")
	}
	
	// Handle unhealthy status
	if result.Status == StatusUnhealthy {
		c.handleUnhealthyApp(ctx, appName, healthCheck, result)
	}
}

// handleUnhealthyApp handles an unhealthy application
func (c *Checker) handleUnhealthyApp(ctx context.Context, appName string, healthCheck config.HealthCheckConfig, result *Result) {
	logger := c.logger.WithField("app", appName)
	
	// Check how many consecutive failures we've had
	consecutiveFailures := c.getConsecutiveFailures(appName)
	
	logger.WithFields(logrus.Fields{
		"consecutive_failures": consecutiveFailures,
		"max_retries":         healthCheck.Retries,
		"error":              result.Error,
	}).Warn("Application health check failed")
	
	// If we've exceeded the retry threshold, restart the process
	if consecutiveFailures >= healthCheck.Retries {
		proc, exists := c.processManager.GetProcess(appName)
		if exists && proc.Config.RestartPolicy.Enabled {
			logger.Error("Health check failed too many times, restarting process")
			
			// Restart the process
			if err := c.processManager.Restart(ctx, appName); err != nil {
				logger.WithError(err).Error("Failed to restart unhealthy process")
			} else {
				logger.Info("Process restarted due to failed health checks")
				// Reset failure count after restart
				c.resetConsecutiveFailures(appName)
			}
		}
	}
}

// getConsecutiveFailures counts consecutive health check failures for an app
func (c *Checker) getConsecutiveFailures(appName string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result, exists := c.results[appName]
	if !exists || result.Status == StatusHealthy {
		return 0
	}
	
	// For simplicity, we'll track this in the result error field
	// In a production system, you'd want a more sophisticated tracking mechanism
	return 1
}

// resetConsecutiveFailures resets the consecutive failure count for an app
func (c *Checker) resetConsecutiveFailures(appName string) {
	// Implementation would reset the failure count
	// For now, this is a placeholder
}

// Stop stops all health checking
func (c *Checker) Stop() {
	c.logger.Info("Stopping health checker")
	// Health checks will stop when the context is cancelled
}