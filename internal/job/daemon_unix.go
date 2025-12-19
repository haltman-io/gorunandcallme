//go:build !windows

package job

import (
	"os/exec"
	"syscall"
)

// daemonize configures the exec.Cmd to detach from the current session/terminal.
// On Unix-like systems, Setsid makes the process a new session leader, effectively
// detaching it from the controlling terminal.
func daemonize(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true

	// Optional: put the process in its own process group.
	// This helps when you want to signal the whole group later.
	cmd.SysProcAttr.Setpgid = true
}
