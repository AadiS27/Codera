//go:build linux

package process

import (
	"os/exec"
	"syscall"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// Create a new process group for the command
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		// Kill the entire process group by sending signal to -pid
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
