//go:build !windows

package claude

import (
	"syscall"
)

// setProcessGroupAttr sets the process group attributes for the command.
// On Unix-like systems, this creates a new process group for proper termination.
func setProcessGroupAttr(sysProcAttr *syscall.SysProcAttr) {
	sysProcAttr.Setpgid = true
}

// killProcessGroup kills the entire process group for the given PID.
// On Unix-like systems, negative PID means the process group.
func killProcessGroup(pid int, sig syscall.Signal) error {
	return syscall.Kill(-pid, sig)
}

// isProcessGroupSupported returns true if process groups are supported.
func isProcessGroupSupported() bool {
	return true
}