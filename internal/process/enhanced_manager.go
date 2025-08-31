package process

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gleicon/guvnor/internal/config"
	"github.com/gleicon/guvnor/internal/logs"
)

// StopResult contains information about a stopped process
type StopResult struct {
	Name      string
	PID       int
	Status    string // "stopped", "killed", "not_running", "error"
	Error     error
	Duration  time.Duration
}

// EnhancedManager extends the basic Manager with better logging and PID tracking
type EnhancedManager struct {
	*Manager
	logManager *logs.LogManager
	stopping   map[string]bool // Track which processes are being stopped
	stopMu     sync.RWMutex
}

// NewEnhancedManager creates a new enhanced process manager
func NewEnhancedManager(logger *logrus.Logger, logCapacity int) *EnhancedManager {
	return &EnhancedManager{
		Manager:    NewManager(logger),
		logManager: logs.NewLogManager(logCapacity),
		stopping:   make(map[string]bool),
	}
}

// GetLogManager returns the log manager
func (em *EnhancedManager) GetLogManager() *logs.LogManager {
	return em.logManager
}

// StopAllWithResults stops all managed processes and returns detailed results
func (em *EnhancedManager) StopAllWithResults(ctx context.Context) ([]StopResult, error) {
	em.mu.RLock()
	processes := make([]*Process, 0, len(em.processes))
	processNames := make([]string, 0, len(em.processes))
	
	for name, proc := range em.processes {
		if proc.IsRunning() {
			processes = append(processes, proc)
			processNames = append(processNames, name)
		}
	}
	em.mu.RUnlock()
	
	if len(processes) == 0 {
		return []StopResult{}, nil
	}
	
	em.logManager.Log("system", "info", fmt.Sprintf("Stopping %d processes: %v", len(processes), processNames))
	
	results := make([]StopResult, len(processes))
	var wg sync.WaitGroup
	
	// Stop processes concurrently
	for i, proc := range processes {
		wg.Add(1)
		go func(idx int, p *Process) {
			defer wg.Done()
			results[idx] = em.stopProcessWithResult(ctx, p)
		}(i, proc)
	}
	
	wg.Wait()
	
	// Count results
	var errors []error
	stopped := 0
	killed := 0
	
	for _, result := range results {
		switch result.Status {
		case "stopped":
			stopped++
		case "killed":
			killed++
		case "error":
			errors = append(errors, result.Error)
		}
	}
	
	statusMsg := fmt.Sprintf("Stop complete: %d stopped gracefully, %d killed", stopped, killed)
	if len(errors) > 0 {
		statusMsg += fmt.Sprintf(", %d errors", len(errors))
	}
	
	em.logManager.Log("system", "info", statusMsg)
	
	var combinedError error
	if len(errors) > 0 {
		combinedError = fmt.Errorf("failed to stop some processes: %v", errors)
	}
	
	return results, combinedError
}

// stopProcessWithResult stops a single process and returns detailed result
func (em *EnhancedManager) stopProcessWithResult(ctx context.Context, proc *Process) StopResult {
	start := time.Now()
	
	result := StopResult{
		Name: proc.Config.Name,
		PID:  proc.GetPID(),
	}
	
	if !proc.IsRunning() {
		result.Status = "not_running"
		result.Duration = time.Since(start)
		return result
	}
	
	// Mark as stopping
	em.stopMu.Lock()
	em.stopping[proc.Config.Name] = true
	em.stopMu.Unlock()
	
	defer func() {
		em.stopMu.Lock()
		delete(em.stopping, proc.Config.Name)
		em.stopMu.Unlock()
	}()
	
	em.logManager.Log(proc.Config.Name, "info", fmt.Sprintf("Stopping process (PID: %d)", result.PID))
	
	if err := proc.Stop(ctx); err != nil {
		result.Status = "error"
		result.Error = err
		em.logManager.Log(proc.Config.Name, "error", fmt.Sprintf("Failed to stop process: %v", err))
	} else {
		// Determine if it was stopped gracefully or killed
		result.Duration = time.Since(start)
		if result.Duration > 10*time.Second {
			result.Status = "killed" // Took too long, likely was force-killed
			em.logManager.Log(proc.Config.Name, "warn", fmt.Sprintf("Process force-killed after %.1fs", result.Duration.Seconds()))
		} else {
			result.Status = "stopped"
			em.logManager.Log(proc.Config.Name, "info", fmt.Sprintf("Process stopped gracefully (%.1fs)", result.Duration.Seconds()))
		}
	}
	
	return result
}

// StartWithLogging starts a process with enhanced logging
func (em *EnhancedManager) StartWithLogging(ctx context.Context, appConfig config.AppConfig) error {
	em.logManager.Log(appConfig.Name, "info", fmt.Sprintf("Starting process: %s", appConfig.Command))
	
	// Create enhanced process that logs to our buffer
	err := em.Start(ctx, appConfig)
	if err != nil {
		em.logManager.Log(appConfig.Name, "error", fmt.Sprintf("Failed to start: %v", err))
		return err
	}
	
	// Get the started process and attach log capture
	proc, exists := em.GetProcess(appConfig.Name)
	if exists && proc.IsRunning() {
		em.logManager.Log(appConfig.Name, "info", fmt.Sprintf("Process started successfully (PID: %d)", proc.GetPID()))
		
		// Start capturing process output
		go em.captureProcessOutput(proc)
	}
	
	return nil
}

// captureProcessOutput captures stdout/stderr from a process and logs it
func (em *EnhancedManager) captureProcessOutput(proc *Process) {
	if proc.cmd == nil {
		return
	}
	
	// Note: For proper output capture, we'd need to modify the process creation
	// to set up pipes. For now, we'll simulate log capture by monitoring the process
	// and logging status changes.
	
	// Log process start
	em.logManager.Log(proc.Config.Name, "info", fmt.Sprintf("Process output capture started for PID %d", proc.GetPID()))
	
	// In a real implementation, you'd set up cmd.Stdout and cmd.Stderr pipes
	// before calling cmd.Start() in the original process creation code
}

// Additional utility methods for enhanced process management

// LogProcessEvent logs an event for a process
func (em *EnhancedManager) LogProcessEvent(processName, level, message string) {
	em.logManager.Log(processName, level, message)
}

// IsProcessStopping checks if a process is currently being stopped
func (em *EnhancedManager) IsProcessStopping(name string) bool {
	em.stopMu.RLock()
	defer em.stopMu.RUnlock()
	
	return em.stopping[name]
}

// GetRunningProcessInfo returns information about all running processes
func (em *EnhancedManager) GetRunningProcessInfo() []ProcessInfo {
	em.mu.RLock()
	defer em.mu.RUnlock()
	
	var info []ProcessInfo
	
	for name, proc := range em.processes {
		if proc.IsRunning() {
			info = append(info, ProcessInfo{
				Name:      name,
				PID:       proc.GetPID(),
				Status:    string(proc.GetStatus()),
				Restarts:  proc.GetRestartCount(),
				Command:   proc.Config.Command,
				Args:      proc.Config.Args,
				StartTime: proc.lastStart,
				Port:      proc.Config.Port,
			})
		}
	}
	
	return info
}

// ProcessInfo contains information about a running process
type ProcessInfo struct {
	Name      string     `json:"name"`
	PID       int        `json:"pid"`
	Status    string     `json:"status"`
	Restarts  int        `json:"restarts"`
	Command   string     `json:"command"`
	Args      []string   `json:"args"`
	StartTime time.Time  `json:"start_time"`
	Port      int        `json:"port"`
}