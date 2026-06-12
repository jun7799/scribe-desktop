#!/bin/bash
# ============================================================
# Scribe Desktop - macOS 一键构建脚本
# 用法: chmod +x build-mac.sh && ./build-mac.sh
# ============================================================

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# ---- 配置 ----
APP_NAME="scribe-desktop"
VERSION=$(grep -o '"version": *"[^"]*"' frontend/package.json 2>/dev/null | head -1 | grep -o '"[^"]*"$' | tr -d '"' || echo "1.0.0")
BUILD_DIR="build/bin"
SIDEcar_SRC="sidecar"
DIST_DIR="dist"

info "=========================================="
info " Scribe Desktop macOS 构建脚本"
info " 版本: ${VERSION}"
info " 架构: $(uname -m)"
info "=========================================="

# ---- 1. 检查依赖 ----
info "[1/6] 检查构建依赖..."

command -v go >/dev/null 2>&1    || error "未找到 Go，请安装: https://go.dev/dl/"
command -v node >/dev/null 2>&1  || error "未找到 Node.js，请安装: https://nodejs.org/"
command -v npm >/dev/null 2>&1   || error "未找到 npm"

# 检查 Wails CLI
if ! command -v wails >/dev/null 2>&1; then
    warn "未找到 Wails CLI，正在安装..."
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    # 确保 GOPATH/bin 在 PATH 中
    export PATH="$PATH:$(go env GOPATH)/bin"
    command -v wails >/dev/null 2>&1 || error "Wails 安装失败，请手动执行: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
fi

WAILS_VERSION=$(wails version 2>/dev/null || echo "unknown")
info "Wails 版本: ${WAILS_VERSION}"
info "Go 版本: $(go version)"
info "Node 版本: $(node --version)"

# ---- 2. 检查 sidecar 二进制 ----
info "[2/6] 检查 sidecar 二进制..."

ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ]; then
    SIDECAR_BIN="wx-dl-darwin-arm64"
else
    SIDECAR_BIN="wx-dl-darwin-amd64"
fi

if [ ! -f "${SIDEcar_SRC}/${SIDECAR_BIN}" ]; then
    error "未找到 sidecar 二进制: ${SIDEcar_SRC}/${SIDECAR_BIN}
    请从 https://github.com/ltaoo/wx_channels_download/releases 下载 ${SIDECAR_BIN} 放到 sidecar/ 目录"
fi

# 清除 macOS 安全标记
xattr -cr "${SIDEcar_SRC}/${SIDECAR_BIN}" 2>/dev/null || true
chmod +x "${SIDEcar_SRC}/${SIDECAR_BIN}"
info "Sidecar 二进制就绪: ${SIDECAR_BIN}"

# ---- 3. 安装前端依赖 ----
info "[3/6] 安装前端依赖..."
cd frontend
npm install
cd ..
info "前端依赖安装完成"

# ---- 4. 构建 ----
info "[4/6] 开始 Wails 构建..."
wails build -clean
info "构建完成"

# ---- 5. 复制 sidecar 到构建产物 ----
info "[5/6] 打包 sidecar 二进制..."

mkdir -p "${BUILD_DIR}/${APP_NAME}.app/Contents/MacOS/sidecar"
cp "${SIDEcar_SRC}/${SIDECAR_BIN}" "${BUILD_DIR}/${APP_NAME}.app/Contents/MacOS/sidecar/"
chmod +x "${BUILD_DIR}/${APP_NAME}.app/Contents/MacOS/sidecar/${SIDECAR_BIN}"

info "Sidecar 已打包到 .app 内"

# ---- 6. 创建 DMG（可选） ----
info "[6/6] 创建 DMG 安装包..."

mkdir -p "${DIST_DIR}"

DMG_NAME="${APP_NAME}-macOS-${ARCH}-v${VERSION}.dmg"
DMG_PATH="${DIST_DIR}/${DMG_NAME}"

# 使用 hdiutil 创建 DMG
rm -f "${DMG_PATH}"

# 创建临时目录组织 DMG 内容
DMG_TMP=$(mktemp -d)
cp -R "${BUILD_DIR}/${APP_NAME}.app" "${DMG_TMP}/"

# 创建 Applications 快捷方式
ln -s /Applications "${DMG_TMP}/Applications"

hdiutil create -volname "${APP_NAME}" \
    -srcfolder "${DMG_TMP}" \
    -ov -format UDZO \
    "${DMG_PATH}"

rm -rf "${DMG_TMP}"

info "=========================================="
info " 构建成功！"
info "=========================================="
info ""
info " .app 路径: ${BUILD_DIR}/${APP_NAME}.app"
info " DMG 路径:  ${DMG_PATH}"
info " 文件大小:  $(du -h "${DMG_PATH}" | cut -f1)"
info ""
info " 可以直接双击 .app 运行，或分发 DMG 安装包"
info "=========================================="
