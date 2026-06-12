# Scribe Desktop - 待办事项

## 当前状态

- ✅ Windows 版本已完成，可正常使用
- ✅ macOS 版本已完成（Apple Silicon）

## macOS 适配待办

### 1. 准备 wx-dl macOS 二进制 ✅

从 [wx_channels_download Releases](https://github.com/ltaoo/wx_channels_download/releases) 下载：

- `wx-dl-darwin-arm64` — Apple Silicon (M1/M2/M3) ✅ 已添加
- `wx-dl-darwin-amd64` — Intel Mac ⬜ 暂未添加（按需补充）

放到项目的 `sidecar/` 目录下：

```
sidecar/
├── wx-dl-windows-amd64.exe   ← 已有
├── wx-dl-darwin-arm64        ← 已添加
└── wx-dl-darwin-amd64        ← 按需补充
```

> `sidecar.go` 里的 `embeddedBinaryName()` 已经处理了这两个文件名，不需要改代码。

### 2. macOS 证书自动安装 ✅

已在 `sidecar_darwin.go` 中实现：

- `certInstalled()` — 通过 `security find-certificate` 检查系统钥匙串和用户钥匙串
- `ensureCertInstalled()` — 未安装时触发提权安装
- `runElevated()` — 使用 `osascript` 弹出系统密码框提权（等同于 Windows UAC）

### 3. macOS 构建配置 ✅

- `build/darwin/Info.plist` 已配置，包含 `NSAppTransportSecurity` 允许本地网络访问
- `sidecar.go` 的 `EnsureExtracted()` 已处理 `xattr -cr` 和 `chmod 0755`

### 4. macOS 代理设置 ✅

- wx-dl 在 macOS 上会自动设置系统代理（和 Windows 类似）
- 通用逻辑已在 `sidecar.go` 中实现，跨平台一致

### 5. 打包测试 ⬜

在 Mac 上执行：

```bash
# 安装 Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 开发模式测试
wails dev

# 构建
wails build
```

构建产物在 `build/bin/` 目录下。

### 6. wails build 构建钩子 ⬜

当前 `wails build` 后需要手动复制 sidecar 二进制。建议在 `wails.json` 中添加构建钩子：

```json
{
  "preBuildHooks": {
    "windows": "xcopy /E /I /Y sidecar build\\bin\\sidecar",
    "darwin": "mkdir -p build/bin/sidecar && cp sidecar/wx-dl-darwin-* build/bin/sidecar/"
  }
}
```

或者写一个跨平台的构建脚本 `build.sh`。

---

## 已修复的问题记录（供参考）

这些问题在 Windows 上已经解决，macOS 上大概率不会遇到，但记录一下：

### 1. `WaitReady` 超时参数单位错误
- `time.Duration` 是纳秒，传 `15` 等于 15 纳秒
- 修复：改为 `15 * time.Second`

### 2. wx-dl 启动方式
- 不能用 `server` 子命令（那是 API-only 模式），要用不带子命令的正常模式
- `--hostname` 必须用 `127.0.0.1`，不能用 `0.0.0.0`（0.0.0.0 无法作为系统代理地址）

### 3. 进程创建方式
- Windows 上不能用 `CREATE_NO_WINDOW`（wx-dl 需要控制台窗口）
- 改为 `CREATE_NEW_CONSOLE`
- macOS 上用 `Setpgid: true` 即可，无此问题

### 4. 端口说明
- wx-dl 启动后有**两个端口**：
  - **API 端口 2022**：提供 `/api/status`、`/api/task/list` 等接口
  - **代理端口 2023**：HTTPS 代理，浏览器走这个端口

### 5. 首次运行证书安装
- wx-dl 作为 HTTPS 代理需要安装自签名 CA 证书
- 首次运行必须管理员权限
- Windows 实现了自动 UAC 提权（`ShellExecuteEx` + `runas`）
- macOS 需要类似实现（`osascript` + `with administrator privileges`）

---

## 项目结构

```
scribe-desktop/
├── app.go                          # 主应用逻辑，前端可调用方法
├── main.go                         # Wails 入口
├── internal/
│   ├── config/config.go            # 配置管理（下载目录、端口等）
│   ├── api/client.go               # wx-dl HTTP API 客户端
│   └── sidecar/
│       ├── sidecar.go              # 进程管理（跨平台）
│       ├── sidecar_windows.go      # Windows 专属（进程创建、证书安装）
│       └── sidecar_darwin.go       # macOS/Linux 专属（进程创建）⬜ 需补充证书逻辑
├── frontend/
│   └── src/
│       ├── App.vue                 # 主界面（引导页 + 仪表盘）
│       └── components/             # 空的，全部写在 App.vue 里
├── sidecar/
│   └── wx-dl-windows-amd64.exe    # Windows 版 wx-dl 二进制
├── build/
│   ├── windows/wails.exe.manifest  # Windows 应用清单（UAC 配置）
│   └── bin/                        # 构建输出目录
└── wails.json                      # Wails 项目配置
```
