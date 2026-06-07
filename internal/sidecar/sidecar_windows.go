//go:build windows

package sidecar

import (
	"os"
	"os/exec"
	"syscall"
)

func configureCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}

func killProcess(proc *os.Process) error {
	// Windows: 使用 Kill 直接终止（避免 CTRL_C_EVENT 影响父进程）
	return proc.Kill()
}
