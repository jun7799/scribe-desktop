//go:build windows

package sidecar

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

// certInstalled 检查 wx-dl 的 CA 证书是否已安装到系统
func certInstalled() bool {
	certDirs := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "wx_channels_download"),
		filepath.Join(os.Getenv("APPDATA"), "wx_channels_download"),
	}
	for _, dir := range certDirs {
		for _, name := range []string{"cert.pem", "ca.crt", "rootCA.pem"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
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

// runElevated 以管理员权限运行指定程序（触发 UAC 提权）
func runElevated(exePath string, args []string) error {
	cmdLine := ""
	for _, a := range args {
		cmdLine += ` "` + a + `"`
	}

	fullCmd := fmt.Sprintf(`/c "%s"%s`, exePath, cmdLine)

	dirPtr, _ := syscall.UTF16PtrFromString(filepath.Dir(exePath))
	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	filePtr, _ := syscall.UTF16PtrFromString("cmd.exe")
	paramPtr, _ := syscall.UTF16PtrFromString(fullCmd)

	type ShellExecuteInfo struct {
		cbSize         uint32
		fMask          uint32
		hwnd           uintptr
		lpVerb         *uint16
		lpFile         *uint16
		lpParameters   *uint16
		lpDirectory    *uint16
		nShow          int32
		hInstApp       uintptr
		lpIDList       uintptr
		lpClass        *uint16
		hkeyClass      uintptr
		dwHotKey       uint32
		hIconOrMonitor uintptr
		hProcess       uintptr
	}

	var sei ShellExecuteInfo
	sei.cbSize = uint32(unsafe.Sizeof(sei))
	sei.fMask = 0x00000040 // SEE_MASK_NOCLOSEPROCESS
	sei.lpVerb = verbPtr
	sei.lpFile = filePtr
	sei.lpParameters = paramPtr
	sei.lpDirectory = dirPtr
	sei.nShow = 0 // SW_HIDE

	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecuteEx := shell32.NewProc("ShellExecuteExW")

	ret, _, err := shellExecuteEx.Call(uintptr(unsafe.Pointer(&sei)))
	if ret == 0 {
		return fmt.Errorf("提权失败（用户可能取消了 UAC）: %w", err)
	}

	// 等待进程完成
	if sei.hProcess != 0 {
		syscall.WaitForSingleObject(syscall.Handle(sei.hProcess), 30000)
		syscall.CloseHandle(syscall.Handle(sei.hProcess))
	}

	return nil
}

func configureCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    false,
		CreationFlags: 0x00000010, // CREATE_NEW_CONSOLE
	}
}

func killProcess(proc *os.Process) error {
	return proc.Kill()
}
