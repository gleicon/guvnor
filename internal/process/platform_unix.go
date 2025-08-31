//go:build !windows

package process

import (
	"os"
	"os/exec"
	"syscall"
)

// setPlatformProcAttributes sets Unix-specific process attributes
func setPlatformProcAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr.Setpgid = true
}

// getPlatformTermSignal returns SIGTERM for Unix systems
func getPlatformTermSignal() os.Signal {
	return syscall.SIGTERM
}

// killPlatformProcess kills a process on Unix systems
func killPlatformProcess(process *os.Process, pid int) {
	// Try to kill the entire process group first
	if pgid, err := syscall.Getpgid(pid); err == nil {
		syscall.Kill(-pgid, syscall.SIGKILL)
	} else {
		// Fallback to killing just the main process
		process.Kill()
	}
}