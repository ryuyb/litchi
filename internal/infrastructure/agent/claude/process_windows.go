//go:build windows

package claude

import (
	"syscall"
)

// setProcessGroupAttr sets the process group attributes for the command.
// On Windows, process groups work differently but we can still use the process handle.
func setProcessGroupAttr(sysProcAttr *syscall.SysProcAttr) {
	// Windows doesn't use Setpgid, but we can create a new process group
	// using CREATE_NEW_PROCESS_GROUP flag if needed
	sysProcAttr.HideWindow = true
}

// killProcessGroup kills the process on Windows.
// Windows doesn't have process groups like Unix, so we kill the process directly.
// The context cancellation should handle child processes if they were started properly.
func killProcessGroup(pid int, sig syscall.Signal) error {
	// On Windows, we need to use the process handle to terminate
	// For simplicity, we'll just kill by PID which may not catch all children
	// A more robust solution would use job objects
	if sig == syscall.SIGKILL {
		// Use taskkill for force kill on Windows
		return syscall.Kill(pid, syscall.SIGKILL)
	}
	return syscall.Kill(pid, sig)
}

// isProcessGroupSupported returns true if process groups are supported.
func isProcessGroupSupported() bool {
	return false // Windows doesn't support Unix-style process groups
}