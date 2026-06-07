package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"scribe-desktop/internal/api"
	"scribe-desktop/internal/config"
	"scribe-desktop/internal/sidecar"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ServiceStatus 服务状态
type ServiceStatus struct {
	Running   bool     `json:"running"`
	ProxyOn   bool     `json:"proxyOn"`
	ApiUrl    string   `json:"apiUrl"`
	ProxyPort int      `json:"proxyPort"`
	LocalIPs  []string `json:"localIps"`
}

// App 主应用
type App struct {
	ctx     context.Context
	sidecar *sidecar.Manager
	api     *api.Client
	config  *config.Config
}

// NewApp 创建应用实例
func NewApp() *App {
	cfg := config.Load()
	return &App{
		config:  cfg,
		api:     api.NewClient(cfg.ApiPort),
		sidecar: sidecar.NewManager(),
	}
}

// startup Wails 启动回调
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 确保数据目录
	config.EnsureDataDirs(a.config)

	// 提取 sidecar 二进制
	if err := a.sidecar.EnsureExtracted(); err != nil {
		wailsRuntime.LogWarningf(ctx, "Sidecar 提取失败: %s", err.Error())
	}

	// 设置日志回调
	a.sidecar.SetLogCallback(func(line string) {
		wailsRuntime.EventsEmit(ctx, "log:line", line)
	})

	a.sidecar.SetExitCallback(func() {
		wailsRuntime.EventsEmit(ctx, "service:status", a.GetStatus())
	})
}

// --- 前端可调用方法 ---

// StartProxy 启动代理服务
func (a *App) StartProxy() error {
	if err := a.sidecar.Start(a.config.DownloadDir, a.config.ProxyPort); err != nil {
		return err
	}

	// 等待就绪
	if !a.sidecar.WaitReady(a.config.ApiPort, 15) {
		a.sidecar.Stop()
		return fmt.Errorf("代理服务启动超时")
	}

	wailsRuntime.EventsEmit(a.ctx, "service:status", a.GetStatus())
	return nil
}

// StopProxy 停止代理服务
func (a *App) StopProxy() error {
	if err := a.sidecar.Stop(); err != nil {
		return err
	}
	wailsRuntime.EventsEmit(a.ctx, "service:status", a.GetStatus())
	return nil
}

// GetStatus 获取当前服务状态
func (a *App) GetStatus() ServiceStatus {
	return ServiceStatus{
		Running:   a.sidecar.IsRunning(),
		ApiUrl:    fmt.Sprintf("http://127.0.0.1:%d", a.config.ApiPort),
		ProxyPort: a.config.ProxyPort,
		LocalIPs:  getLocalIPs(),
	}
}

// GetTasks 获取下载任务列表
func (a *App) GetTasks(page, pageSize int) (*api.TaskListResult, error) {
	return a.api.ListTasks("done", page, pageSize)
}

// GetDownloadDir 获取下载目录
func (a *App) GetDownloadDir() string {
	return a.config.DownloadDir
}

// SetDownloadDir 设置下载目录
func (a *App) SetDownloadDir(path string) error {
	a.config.DownloadDir = path
	os.MkdirAll(path, 0755)
	return config.Save(a.config)
}

// SelectFolder 打开文件夹选择器
func (a *App) SelectFolder() string {
	path, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "选择下载目录",
	})
	if err != nil || path == "" {
		return ""
	}
	return path
}

// OpenFolder 在文件管理器中打开
func (a *App) OpenFolder(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

// IsFirstRun 是否首次运行
func (a *App) IsFirstRun() bool {
	return a.config.FirstRun
}

// CompleteOnboarding 完成引导
func (a *App) CompleteOnboarding() error {
	a.config.FirstRun = false
	return config.Save(a.config)
}

// GetLocalIP 获取本机 IP
func (a *App) GetLocalIP() []string {
	return getLocalIPs()
}

// --- 内部方法 ---

// getLocalIPs 获取本机 IP 地址（简化版）
func getLocalIPs() []string {
	// TODO: 实现完整的 IP 检测
	return []string{}
}

// shutdown 清理（退出前调用）
func (a *App) shutdown(ctx context.Context) {
	if a.sidecar.IsRunning() {
		a.sidecar.Stop()
	}
}
