//go:build darwin || linux

package sidecar

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
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

// certInstalled 检查 wx-dl 的 CA 证书是否已安装到系统钥匙串
func certInstalled() bool {
	// wx-dl 基于 SunnyNet，证书 CommonName 包含 "SunnyNet"
	// 用 security find-certificate 检查系统钥匙串
	cmd := exec.Command("security", "find-certificate", "-c", "SunnyNet", "/Library/Keychains/System.keychain")
	if err := cmd.Run(); err == nil {
		return true
	}

	// 也检查用户钥匙串
	cmd = exec.Command("security", "find-certificate", "-c", "SunnyNet", "login.keychain")
	if err := cmd.Run(); err == nil {
		return true
	}

	// 兜底：检查证书文件是否存在（与 Windows 模式一致）
	homeDir, _ := os.UserHomeDir()
	certDirs := []string{
		fmt.Sprintf("%s/.mitmproxy", homeDir),
		fmt.Sprintf("%s/Library/Application Support/wx_channels_download", homeDir),
	}
	for _, dir := range certDirs {
		for _, name := range []string{"cert.pem", "ca.crt", "rootCA.pem", "SunnyRoot.cer"} {
			if _, err := os.Stat(fmt.Sprintf("%s/%s", dir, name)); err == nil {
				return true
			}
		}
	}

	return false
}

// ensureCertInstalled 确保证书已安装，未安装则提权运行 wx-dl 安装证书
func ensureCertInstalled(binaryPath string) error {
	if certInstalled() {
		return nil
	}
	return runElevated(binaryPath, []string{"--port", "0"})
}

// runElevated 使用 osascript 弹出系统密码框以管理员权限运行指定程序
func runElevated(exePath string, args []string) error {
	// 构建命令字符串
	cmdStr := exePath
	for _, a := range args {
		cmdStr += ` ` + shellEscape(a)
	}

	// 使用 osascript 的 "do shell script ... with administrator privileges"
	// 这是 macOS 上等同于 Windows UAC 提权的标准做法
	script := fmt.Sprintf(`do shell script "%s" with administrator privileges`, cmdStr)

	cmd := exec.Command("osascript", "-e", script)

	// 设置超时：30 秒（与 Windows 版本一致）
	done := make(chan error, 1)
	go func() {
		output, err := cmd.CombinedOutput()
		if err != nil {
			done <- fmt.Errorf("提权执行失败: %s (%s)", err.Error(), string(output))
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		// 超时后尝试杀掉 osascript 进程
		cmd.Process.Kill()
		return fmt.Errorf("提权执行超时（30秒），用户可能未输入密码")
	}
}

// shellEscape 对 shell 参数进行简单转义
func shellEscape(s string) string {
	return `'` + s + `'`
}
