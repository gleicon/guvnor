package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gleicon/guvnor/internal/config"
)

// Process represents a managed application process
type Process struct {
	Config        config.AppConfig
	cmd           *exec.Cmd
	process       *os.Process  // Native Go process handle
	pid           int          // Process ID
	pidFile       string       // PID file path
	logger        *logrus.Entry
	restarts      int
	lastStart     time.Time
	mu            sync.RWMutex
	status        ProcessStatus
	executionMode ExecutionMode
	containerID   string // For container mode
}

// ProcessStatus represents the current status of a process
type ProcessStatus string

const (
	StatusStopped  ProcessStatus = "stopped"
	StatusStarting ProcessStatus = "starting"
	StatusRunning  ProcessStatus = "running"
	StatusStopping ProcessStatus = "stopping"
	StatusFailed   ProcessStatus = "failed"
)

// ExecutionMode defines how processes should be executed
type ExecutionMode string

const (
	ModeProcess   ExecutionMode = "process"   // Fork/exec processes directly
	ModeContainer ExecutionMode = "container" // Run in Docker containers
)

// Manager manages multiple application processes
type Manager struct {
	processes       map[string]*Process
	logger          *logrus.Entry
	mu              sync.RWMutex
	executionMode   ExecutionMode
	dockerAvailable bool
	pidDir          string // Directory for PID files
}

// NewManager creates a new process manager
func NewManager(logger *logrus.Logger) *Manager {
	pidDir := filepath.Join(os.TempDir(), "guvnor", "pids")
	os.MkdirAll(pidDir, 0755) // Create PID directory
	
	m := &Manager{
		processes:       make(map[string]*Process),
		logger:          logger.WithField("component", "process-manager"),
		executionMode:   ModeProcess, // Default to process mode
		dockerAvailable: false,
		pidDir:          pidDir,
	}
	
	// Check if Docker is available
	m.detectDocker()
	
	// Load existing processes from PID files
	m.loadFromPidFiles()
	
	return m
}

// SetExecutionMode sets the execution mode for new processes
func (m *Manager) SetExecutionMode(mode ExecutionMode) error {
	if mode == ModeContainer && !m.dockerAvailable {
		return fmt.Errorf("container mode requested but Docker is not available")
	}
	
	m.mu.Lock()
	m.executionMode = mode
	m.mu.Unlock()
	
	m.logger.WithField("mode", mode).Info("Execution mode set")
	return nil
}

// detectDocker checks if Docker is available
func (m *Manager) detectDocker() {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err == nil {
		m.dockerAvailable = true
		m.logger.Info("Docker detected and available for container mode")
	} else {
		m.logger.Debug("Docker not available, using process mode only")
	}
}

// Start starts a process for the given app configuration
func (m *Manager) Start(ctx context.Context, appConfig config.AppConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if process already exists
	if proc, exists := m.processes[appConfig.Name]; exists {
		if proc.IsRunning() {
			return fmt.Errorf("process %s is already running", appConfig.Name)
		}
		// Remove existing stopped process
		delete(m.processes, appConfig.Name)
	}
	
	// Create new process
	proc := &Process{
		Config:        appConfig,
		logger:        m.logger.WithField("app", appConfig.Name),
		status:        StatusStopped,
		executionMode: m.executionMode,
		pidFile:       filepath.Join(m.pidDir, appConfig.Name+".pid"),
	}
	
	m.processes[appConfig.Name] = proc
	
	// Start the process
	return proc.Start(ctx)
}

// Stop stops a process by name
func (m *Manager) Stop(ctx context.Context, name string) error {
	m.mu.RLock()
	proc, exists := m.processes[name]
	m.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("process %s not found", name)
	}
	
	return proc.Stop(ctx)
}

// Restart restarts a process by name
func (m *Manager) Restart(ctx context.Context, name string) error {
	m.mu.RLock()
	proc, exists := m.processes[name]
	m.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("process %s not found", name)
	}
	
	return proc.Restart(ctx)
}

// StopAll stops all managed processes
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	processes := make([]*Process, 0, len(m.processes))
	for _, proc := range m.processes {
		processes = append(processes, proc)
	}
	m.mu.RUnlock()
	
	var errors []error
	for _, proc := range processes {
		if err := proc.Stop(ctx); err != nil {
			errors = append(errors, err)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("failed to stop some processes: %v", errors)
	}
	
	return nil
}

// GetProcess returns a process by name
func (m *Manager) GetProcess(name string) (*Process, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	proc, exists := m.processes[name]
	return proc, exists
}

// ListProcesses returns all managed processes
func (m *Manager) ListProcesses() map[string]*Process {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]*Process)
	for name, proc := range m.processes {
		result[name] = proc
	}
	
	return result
}

// Start starts the process
func (p *Process) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.status == StatusRunning || p.status == StatusStarting {
		return fmt.Errorf("process is already running or starting")
	}
	
	p.status = StatusStarting
	p.lastStart = time.Now()
	
	switch p.executionMode {
	case ModeContainer:
		return p.startContainer(ctx)
	default:
		return p.startProcess(ctx)
	}
}

// startProcess starts the process using native Go
func (p *Process) startProcess(ctx context.Context) error {
	// Create command
	cmd := exec.CommandContext(ctx, p.Config.Command, p.Config.Args...)
	
	// Set working directory
	if p.Config.WorkingDir != "" {
		cmd.Dir = p.Config.WorkingDir
	}
	
	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range p.Config.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}
	
	// Cross-platform process group setup
	setProcAttributes(cmd)
	
	p.logger.WithFields(logrus.Fields{
		"mode":        "process",
		"command":     p.Config.Command,
		"args":        p.Config.Args,
		"working_dir": p.Config.WorkingDir,
		"port":        p.Config.Port,
	}).Info("Starting process")
	
	// Start the command
	if err := cmd.Start(); err != nil {
		p.status = StatusFailed
		return fmt.Errorf("failed to start process: %w", err)
	}
	
	p.cmd = cmd
	p.process = cmd.Process
	p.pid = cmd.Process.Pid
	p.status = StatusRunning
	
	// Write PID file
	if err := p.writePidFile(); err != nil {
		p.logger.WithError(err).Warn("Failed to write PID file")
	}
	
	// Monitor the process in a goroutine
	go p.monitor(ctx)
	
	p.logger.WithField("pid", p.pid).Info("Process started successfully")
	
	return nil
}

// startContainer starts the process in a Docker container
func (p *Process) startContainer(ctx context.Context) error {
	// Build Docker command
	containerName := fmt.Sprintf("guvnor-%s", p.Config.Name)
	
	args := []string{
		"run", "--rm", "--detach",
		"--name", containerName,
		"--publish", fmt.Sprintf("%d:%d", p.Config.Port, p.Config.Port),
	}
	
	// Add environment variables
	for key, value := range p.Config.Environment {
		args = append(args, "--env", fmt.Sprintf("%s=%s", key, value))
	}
	
	// Mount working directory
	if p.Config.WorkingDir != "" {
		args = append(args, "--volume", fmt.Sprintf("%s:/app", p.Config.WorkingDir))
		args = append(args, "--workdir", "/app")
	}
	
	// Use a simple base image with the runtime
	image := selectBaseImage(p.Config.Command)
	args = append(args, image)
	
	// Add the command and args
	args = append(args, p.Config.Command)
	args = append(args, p.Config.Args...)
	
	cmd := exec.CommandContext(ctx, "docker", args...)
	
	p.logger.WithFields(logrus.Fields{
		"mode":      "container",
		"image":     image,
		"command":   p.Config.Command,
		"args":      p.Config.Args,
		"container": containerName,
		"port":      p.Config.Port,
	}).Info("Starting container")
	
	// Start the container
	output, err := cmd.Output()
	if err != nil {
		p.status = StatusFailed
		return fmt.Errorf("failed to start container: %w", err)
	}
	
	p.containerID = string(output[:12]) // Docker returns the container ID
	p.status = StatusRunning
	
	// Monitor the container in a goroutine
	go p.monitorContainer(ctx)
	
	p.logger.WithField("container_id", p.containerID).Info("Container started successfully")
	
	return nil
}

// selectBaseImage selects an appropriate Docker base image based on the command
func selectBaseImage(command string) string {
	switch command {
	case "python", "python3":
		return "python:3.11-slim"
	case "node", "npm":
		return "node:18-slim"
	case "go":
		return "golang:1.21-alpine"
	default:
		return "alpine:latest"
	}
}

// Stop stops the process gracefully
func (p *Process) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.status != StatusRunning {
		return nil // Already stopped
	}
	
	p.status = StatusStopping
	p.logger.Info("Stopping process")
	
	switch p.executionMode {
	case ModeContainer:
		return p.stopContainer(ctx)
	default:
		return p.stopProcess(ctx)
	}
}

// stopProcess stops a fork/exec process using native Go
func (p *Process) stopProcess(ctx context.Context) error {
	if p.process == nil {
		p.status = StatusStopped
		p.cleanupPidFile()
		return nil
	}
	
	p.logger.WithField("pid", p.pid).Info("Stopping process")
	
	// Try graceful shutdown first (SIGTERM)
	if err := p.process.Signal(getTermSignal()); err != nil {
		p.logger.WithError(err).Warn("Failed to send termination signal")
		// Process might already be dead, try to clean up
		p.status = StatusStopped
		p.cleanupPidFile()
		return nil
	}
	
	// Wait for graceful shutdown with timeout
	done := make(chan error, 1)
	go func() {
		if p.cmd != nil {
			done <- p.cmd.Wait()
		} else {
			// Wait for process to exit by checking if it's still alive
			for i := 0; i < 100; i++ { // 10 seconds total
				if err := p.process.Signal(syscall.Signal(0)); err != nil {
					done <- nil // Process is dead
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
			done <- fmt.Errorf("timeout waiting for process")
		}
	}()
	
	select {
	case <-ctx.Done():
		// Context cancelled, force kill
		p.forceKill()
		return ctx.Err()
	case err := <-done:
		// Process exited
		p.status = StatusStopped
		p.process = nil
		p.cmd = nil
		p.cleanupPidFile()
		if err != nil {
			p.logger.WithError(err).Info("Process stopped with error")
		} else {
			p.logger.Info("Process stopped gracefully")
		}
		return nil
	case <-time.After(10 * time.Second):
		// Timeout, force kill
		p.logger.Warn("Process didn't stop gracefully, forcing kill")
		p.forceKill()
		p.status = StatusStopped
		p.process = nil
		p.cmd = nil
		p.cleanupPidFile()
		return nil
	}
}

// stopContainer stops a Docker container
func (p *Process) stopContainer(ctx context.Context) error {
	if p.containerID == "" {
		p.status = StatusStopped
		return nil
	}
	
	containerName := fmt.Sprintf("guvnor-%s", p.Config.Name)
	
	// Try graceful stop first
	stopCmd := exec.CommandContext(ctx, "docker", "stop", containerName)
	if err := stopCmd.Run(); err != nil {
		p.logger.WithError(err).Warn("Failed to stop container gracefully, forcing kill")
		
		// Force kill if graceful stop failed
		killCmd := exec.CommandContext(ctx, "docker", "kill", containerName)
		if err := killCmd.Run(); err != nil {
			p.logger.WithError(err).Error("Failed to force kill container")
		}
	}
	
	p.status = StatusStopped
	p.containerID = ""
	p.logger.Info("Container stopped")
	
	return nil
}

// Restart restarts the process
func (p *Process) Restart(ctx context.Context) error {
	p.logger.Info("Restarting process")
	
	if err := p.Stop(ctx); err != nil {
		p.logger.WithError(err).Warn("Error stopping process during restart")
	}
	
	// Wait a bit before restarting
	time.Sleep(1 * time.Second)
	
	return p.Start(ctx)
}

// IsRunning returns true if the process is currently running using native Go
func (p *Process) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.status != StatusRunning {
		return false
	}
	
	// Double-check with native Go process check
	if p.process != nil {
		// Use signal 0 to check if process exists (cross-platform)
		if err := p.process.Signal(syscall.Signal(0)); err != nil {
			// Process is dead, update status
			p.status = StatusStopped
			return false
		}
	}
	
	return true
}

// GetStatus returns the current process status
func (p *Process) GetStatus() ProcessStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return p.status
}

// GetRestartCount returns the number of times the process has been restarted
func (p *Process) GetRestartCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return p.restarts
}

// GetPID returns the process ID if running
func (p *Process) GetPID() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	
	return 0
}

// monitor monitors the process and handles restarts
func (p *Process) monitor(ctx context.Context) {
	defer func() {
		p.mu.Lock()
		if p.status == StatusRunning {
			p.status = StatusStopped
		}
		p.mu.Unlock()
	}()
	
	err := p.cmd.Wait()
	
	p.mu.Lock()
	exitCode := p.cmd.ProcessState.ExitCode()
	wasRunning := p.status == StatusRunning
	p.mu.Unlock()
	
	if wasRunning {
		if err != nil {
			p.logger.WithFields(logrus.Fields{
				"error":     err,
				"exit_code": exitCode,
			}).Error("Process exited with error")
		} else {
			p.logger.Info("Process exited normally")
		}
		
		// Handle restart if enabled and not a normal exit
		if p.Config.RestartPolicy.Enabled && exitCode != 0 && p.restarts < p.Config.RestartPolicy.MaxRetries {
			p.mu.Lock()
			p.restarts++
			p.status = StatusStopped
			p.mu.Unlock()
			
			p.logger.WithFields(logrus.Fields{
				"restarts":    p.restarts,
				"max_retries": p.Config.RestartPolicy.MaxRetries,
			}).Info("Scheduling process restart")
			
			// Wait before restarting
			select {
			case <-ctx.Done():
				return
			case <-time.After(p.Config.RestartPolicy.Backoff):
			}
			
			if err := p.Start(ctx); err != nil {
				p.logger.WithError(err).Error("Failed to restart process")
			}
		} else {
			p.mu.Lock()
			p.status = StatusFailed
			p.mu.Unlock()
		}
	}
}

// monitorContainer monitors a Docker container and handles restarts
func (p *Process) monitorContainer(ctx context.Context) {
	defer func() {
		p.mu.Lock()
		if p.status == StatusRunning {
			p.status = StatusStopped
		}
		p.mu.Unlock()
	}()
	
	containerName := fmt.Sprintf("guvnor-%s", p.Config.Name)
	
	// Wait for container to finish
	waitCmd := exec.CommandContext(ctx, "docker", "wait", containerName)
	output, err := waitCmd.Output()
	
	p.mu.Lock()
	wasRunning := p.status == StatusRunning
	p.mu.Unlock()
	
	if wasRunning {
		var exitCode int
		if err != nil {
			p.logger.WithError(err).Error("Container monitoring error")
			exitCode = 1
		} else {
			// Docker wait returns the exit code as string
			if len(output) > 0 {
				exitCode = int(output[0] - '0') // Simple conversion for single digit
			}
		}
		
		if exitCode == 0 {
			p.logger.Info("Container exited normally")
		} else {
			p.logger.WithField("exit_code", exitCode).Error("Container exited with error")
		}
		
		// Handle restart if enabled and not a normal exit
		if p.Config.RestartPolicy.Enabled && exitCode != 0 && p.restarts < p.Config.RestartPolicy.MaxRetries {
			p.mu.Lock()
			p.restarts++
			p.status = StatusStopped
			p.containerID = ""
			p.mu.Unlock()
			
			p.logger.WithFields(logrus.Fields{
				"restarts":    p.restarts,
				"max_retries": p.Config.RestartPolicy.MaxRetries,
			}).Info("Scheduling container restart")
			
			// Wait before restarting
			select {
			case <-ctx.Done():
				return
			case <-time.After(p.Config.RestartPolicy.Backoff):
			}
			
			if err := p.Start(ctx); err != nil {
				p.logger.WithError(err).Error("Failed to restart container")
			}
		} else {
			p.mu.Lock()
			p.status = StatusFailed
			p.containerID = ""
			p.mu.Unlock()
		}
	}
}

// forceKill kills the process forcefully using native Go
func (p *Process) forceKill() {
	if p.process == nil {
		return
	}
	
	p.logger.WithField("pid", p.pid).Warn("Force killing process")
	
	// Use cross-platform process kill
	killProcess(p.process, p.pid)
	
	p.status = StatusStopped
	p.process = nil
	p.cmd = nil
	p.cleanupPidFile()
}

// Native Go helper functions for cross-platform compatibility

// writePidFile writes the process ID to a PID file
func (p *Process) writePidFile() error {
	if p.pidFile == "" {
		return nil
	}
	
	pidStr := strconv.Itoa(p.pid)
	return os.WriteFile(p.pidFile, []byte(pidStr), 0644)
}

// cleanupPidFile removes the PID file
func (p *Process) cleanupPidFile() {
	if p.pidFile != "" {
		os.Remove(p.pidFile)
	}
}

// loadFromPidFiles loads existing processes from PID files
func (m *Manager) loadFromPidFiles() {
	if m.pidDir == "" {
		return
	}
	
	files, err := filepath.Glob(filepath.Join(m.pidDir, "*.pid"))
	if err != nil {
		m.logger.WithError(err).Warn("Failed to scan PID directory")
		return
	}
	
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".pid")
		
		pidData, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		
		pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
		if err != nil {
			os.Remove(file) // Remove invalid PID file
			continue
		}
		
		// Check if process is still running
		if process, err := os.FindProcess(pid); err == nil {
			if err := process.Signal(syscall.Signal(0)); err == nil {
				// Process is running, add to manager
				proc := &Process{
					Config: config.AppConfig{Name: name},
					process: process,
					pid:     pid,
					pidFile: file,
					status:  StatusRunning,
					logger:  m.logger.WithField("app", name),
				}
				m.processes[name] = proc
				m.logger.WithFields(logrus.Fields{
					"process": name,
					"pid":     pid,
				}).Info("Recovered running process")
			} else {
				// Process is dead, remove PID file
				os.Remove(file)
			}
		}
	}
}

// Cross-platform helper functions

// setProcAttributes sets process attributes in a cross-platform way
func setProcAttributes(cmd *exec.Cmd) {
	// On Unix-like systems, create a process group
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	setPlatformProcAttributes(cmd)
}

// getTermSignal returns the appropriate termination signal for the platform
func getTermSignal() os.Signal {
	return getPlatformTermSignal()
}

// killProcess kills a process in a cross-platform way
func killProcess(process *os.Process, pid int) {
	killPlatformProcess(process, pid)
}