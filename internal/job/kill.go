package job

import (
	"fmt"
	"os"
)

// KillPID terminates a process by PID in a portable way.
func KillPID(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid: %d", pid)
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
