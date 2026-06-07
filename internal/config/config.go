package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// Config 应用配置
type Config struct {
	DownloadDir string `json:"downloadDir"`
	AutoProxy   bool   `json:"autoProxy"`
	ProxyPort   int    `json:"proxyPort"`
	ApiPort     int    `json:"apiPort"`
	FirstRun    bool   `json:"firstRun"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	downloadDir := filepath.Join(homeDir, "Downloads", "scribe-desktop")

	return &Config{
		DownloadDir: downloadDir,
		AutoProxy:   true,
		ProxyPort:   2023,
		ApiPort:     2022,
		FirstRun:    true,
	}
}

// DataDir 返回应用数据目录
func DataDir() string {
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			homeDir, _ := os.UserHomeDir()
			localAppData = filepath.Join(homeDir, "AppData", "Local")
		}
		return filepath.Join(localAppData, "scribe-desktop")
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".scribe-desktop")
}

// ConfigPath 返回配置文件路径
func ConfigPath() string {
	return filepath.Join(DataDir(), "config.json")
}

// SidecarDir 返回 sidecar 二进制存放目录
func SidecarDir() string {
	return filepath.Join(DataDir(), "bin")
}

// Load 加载配置，不存在则返回默认配置
func Load() *Config {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig()
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return DefaultConfig()
	}
	return cfg
}

// Save 保存配置到磁盘
func Save(cfg *Config) error {
	dir := DataDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(ConfigPath(), data, 0644)
}

// EnsureDataDirs 确保所有数据目录存在
func EnsureDataDirs(cfg *Config) error {
	dirs := []string{
		DataDir(),
		SidecarDir(),
		cfg.DownloadDir,
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
