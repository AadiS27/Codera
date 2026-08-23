//go:build windows

package process

import (
	"os/exec"
	"strconv"
)

func setSysProcAttr(cmd *exec.Cmd) {
	// On Windows, we can use CREATE_NEW_PROCESS_GROUP via SysProcAttr, but standard taskkill is easier
	// to kill the whole tree without messing with Job Objects in Go directly.
}

func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		// Kill process and its children gracefully/forcefully
		kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
		_ = kill.Run()
	}
}
