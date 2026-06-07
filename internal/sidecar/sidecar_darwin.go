//go:build darwin || linux

package sidecar

import (
	"os"
	"os/exec"
	"syscall"
)

func configureCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // 独立进程组，方便整体终止
	}
}

func killProcess(proc *os.Process) error {
	// 先尝试优雅终止
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return proc.Kill()
	}
	return nil
}
