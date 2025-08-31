//go:build windows

package process

import (
	"os"
	"os/exec"
	"syscall"
)

// setPlatformProcAttributes sets Windows-specific process attributes
func setPlatformProcAttributes(cmd *exec.Cmd) {
	// On Windows, create a new process group
	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP
}

// getPlatformTermSignal returns os.Interrupt for Windows (closest to SIGTERM)
func getPlatformTermSignal() os.Signal {
	return os.Interrupt
}

// killPlatformProcess kills a process on Windows
func killPlatformProcess(process *os.Process, pid int) {
	// On Windows, just kill the process directly
	// Process groups work differently, so we use the simpler approach
	process.Kill()
}