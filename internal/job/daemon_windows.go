//go:build windows

package job

import (
	"os/exec"
	"syscall"
)

const (
	// detachedProcess removes console association for the child process.
	detachedProcess = 0x00000008
)

// daemonize configures the exec.Cmd to start detached in Windows.
// DETACHED_PROCESS removes console association; CREATE_NEW_PROCESS_GROUP
// helps managing signals/events separately.
func daemonize(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	cmd.SysProcAttr.CreationFlags = syscall.CREATE_NEW_PROCESS_GROUP | detachedProcess
	cmd.SysProcAttr.HideWindow = true
}
