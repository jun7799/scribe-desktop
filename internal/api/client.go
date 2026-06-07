package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Task 下载任务
type Task struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Filename string  `json:"filename"`
	Size     int64   `json:"size"`
	Path     string  `json:"path"`
	Status   string  `json:"status"`
}

// TaskListResult 任务列表结果
type TaskListResult struct {
	Tasks []Task `json:"tasks"`
	Total int    `json:"total"`
}

// taskListResponse wx-dl API 原始响应
type taskListResponse struct {
	Code int `json:"code"`
	Data struct {
		Tasks []Task `json:"tasks"`
	} `json:"data"`
}

// Client wx-dl HTTP API 客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建 API 客户端
func NewClient(apiPort int) *Client {
	// Transport.Proxy = nil 绕过系统代理，避免循环
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", apiPort),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				Proxy: nil,
			},
		},
	}
}

// GetStatus 检查服务是否就绪
func (c *Client) GetStatus() (bool, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/status")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200, nil
}

// ListTasks 获取任务列表
func (c *Client) ListTasks(status string, page, pageSize int) (*TaskListResult, error) {
	url := fmt.Sprintf("%s/api/task/list?status=%s&page=%d&pageSize=%d",
		c.baseURL, status, page, pageSize)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw taskListResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	if raw.Code != 0 {
		return &TaskListResult{Tasks: []Task{}, Total: 0}, nil
	}

	return &TaskListResult{
		Tasks: raw.Data.Tasks,
		Total: len(raw.Data.Tasks),
	}, nil
}
