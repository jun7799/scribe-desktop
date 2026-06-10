package sidecar

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	appConfig "scribe-desktop/internal/config"
)

// Manager wx-dl 进程管理器
type Manager struct {
	mu      sync.Mutex
	process *os.Process
	running bool

	onLogLine func(line string)
	onExit    func()

	binaryPath string
	configDir  string
}

// NewManager 创建 sidecar 管理器
func NewManager() *Manager {
	return &Manager{
		binaryPath: filepath.Join(appConfig.SidecarDir(), binaryName()),
		configDir:  appConfig.DataDir(),
	}
}

// SetLogCallback 设置日志回调
func (m *Manager) SetLogCallback(fn func(line string)) {
	m.onLogLine = fn
}

// SetExitCallback 设置退出回调
func (m *Manager) SetExitCallback(fn func()) {
	m.onExit = fn
}

// EnsureCertInstalled 确保证书已安装（仅 Windows 需要提权）
func (m *Manager) EnsureCertInstalled() error {
	return ensureCertInstalled(m.binaryPath)
}

// EnsureExtracted 将嵌入的二进制提取到数据目录
func (m *Manager) EnsureExtracted() error {
	if err := os.MkdirAll(appConfig.SidecarDir(), 0755); err != nil {
		return fmt.Errorf("创建 sidecar 目录失败: %w", err)
	}

	// 嵌入的二进制路径
	exePath, _ := os.Executable()
	embedPath := filepath.Join(filepath.Dir(exePath), "sidecar", embeddedBinaryName())

	if _, err := os.Stat(embedPath); err != nil {
		cwd, _ := os.Getwd()
		embedPath = filepath.Join(cwd, "sidecar", embeddedBinaryName())
		if _, err := os.Stat(embedPath); err != nil {
			return fmt.Errorf("找不到嵌入的 wx-dl 二进制: %s", embeddedBinaryName())
		}
	}

	// 检查是否需要更新（SHA256 对比）
	needCopy := false
	if _, err := os.Stat(m.binaryPath); err != nil {
		needCopy = true
	} else if fileHash(embedPath) != fileHash(m.binaryPath) {
		needCopy = true
	}

	if needCopy {
		if err := copyFile(embedPath, m.binaryPath); err != nil {
			return fmt.Errorf("复制二进制失败: %w", err)
		}
		if runtime.GOOS == "darwin" {
			os.Chmod(m.binaryPath, 0755)
			exec.Command("xattr", "-cr", m.binaryPath).Run()
		}
	}

	return nil
}

// Start 启动 wx-dl 进程
func (m *Manager) Start(downloadDir string, proxyPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("代理服务已在运行")
	}

	// 生成配置文件
	configPath := filepath.Join(m.configDir, "wx-dl-config.yaml")
	content := fmt.Sprintf("download:\n  dir: \"%s\"\n", filepath.ToSlash(downloadDir))
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("生成配置失败: %w", err)
	}

	args := []string{
		"--hostname", "127.0.0.1",
		"--port", fmt.Sprintf("%d", proxyPort),
		"--config", configPath,
	}

	binaryPath := m.binaryPath
	// 检查二进制文件是否存在
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("二进制文件不存在: %s", binaryPath)
	}

	cmd := exec.Command(binaryPath, args...)
	configureCmd(cmd)

	// 输出启动信息到日志
	if m.onLogLine != nil {
		m.onLogLine(fmt.Sprintf("[INFO] 启动: %s %v", binaryPath, args))
	}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动进程失败: %w", err)
	}

	m.process = cmd.Process
	m.running = true

	go m.streamOutput(stdout)
	go m.streamOutput(stderr)

	go func() {
		cmd.Wait()
		m.mu.Lock()
		m.running = false
		m.process = nil
		m.mu.Unlock()
		if m.onExit != nil {
			m.onExit()
		}
	}()

	return nil
}

// Stop 停止进程
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running || m.process == nil {
		return nil
	}

	done := make(chan error, 1)
	go func() { done <- killProcess(m.process) }()

	select {
	case <-done:
		m.running = false
		m.process = nil
		return nil
	case <-time.After(5 * time.Second):
		m.process.Kill()
		m.running = false
		m.process = nil
		return fmt.Errorf("进程未响应，已强制终止")
	}
}

// IsRunning 检查是否运行中
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// WaitReady 等待 API 就绪
func (m *Manager) WaitReady(apiPort int, timeout time.Duration) bool {
	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) { return nil, nil },
	}
	client := &http.Client{Timeout: 2 * time.Second, Transport: transport}
	apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/status", apiPort)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(apiURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return true
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func (m *Manager) streamOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if m.onLogLine != nil {
			m.onLogLine(scanner.Text())
		}
	}
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "wx-dl.exe"
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return "wx-dl-arm64"
	}
	return "wx-dl"
}

func embeddedBinaryName() string {
	if runtime.GOOS == "windows" {
		return "wx-dl-windows-amd64.exe"
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return "wx-dl-darwin-arm64"
	}
	if runtime.GOOS == "darwin" {
		return "wx-dl-darwin-amd64"
	}
	return "wx-dl-linux-amd64"
}

func fileHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
