#!/usr/bin/env bash
################################################################################
## 开发工具安装脚本
##
## 功能：
##   安装本地开发所需的所有工具
##
## 安装的工具：
##   - buf (Protobuf 构建工具)
##   - protoc-gen-go (Go Protobuf 插件)
##   - protoc-gen-go-grpc (Go gRPC 插件)
##   - protoc-gen-go-http (Kratos HTTP 插件)
##   - protoc-gen-go-errors (Kratos Errors 插件)
##   - protoc-gen-openapi (OpenAPI 生成插件)
##   - wire (依赖注入工具)
##   - ent (ORM 代码生成工具)
##   - golangci-lint (代码检查工具)
##
## 用法：
##   bash scripts/local/install_tools.sh
##
################################################################################
set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# 检查 Go 是否安装
if ! command -v go >/dev/null 2>&1; then
    log_error "Go is not installed!"
    log_info "Please install Go first: https://golang.org/dl/"
    exit 1
fi

log_info "Go version: $(go version)"
log_info "GOPATH: $(go env GOPATH)"
echo ""

# 确保 GOPATH/bin 在 PATH 中
GOBIN="$(go env GOPATH)/bin"
if [[ ":$PATH:" != *":$GOBIN:"* ]]; then
    log_warn "GOPATH/bin is not in PATH, adding temporarily..."
    export PATH="$PATH:$GOBIN"
    
    # 提示用户永久添加
    log_info "To make this permanent, add the following to your shell profile:"
    log_info "  export PATH=\"\$PATH:\$(go env GOPATH)/bin\""
    echo ""
fi

# 安装工具函数
install_tool() {
    local tool_name=$1
    local tool_package=$2
    
    log_info "Installing $tool_name..."
    
    if go install "$tool_package"; then
        log_success "$tool_name installed ✓"
    else
        log_error "Failed to install $tool_name"
        return 1
    fi
}

# ============================================================================
# 安装核心工具
# ============================================================================

log_info "Installing core development tools..."
echo ""

# Buf - Protobuf 构建工具
install_tool "buf" "github.com/bufbuild/buf/cmd/buf@latest"

# Protobuf 插件
install_tool "protoc-gen-go" "google.golang.org/protobuf/cmd/protoc-gen-go@latest"
install_tool "protoc-gen-go-grpc" "google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"

# Kratos 插件
install_tool "protoc-gen-go-http" "github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest"
install_tool "protoc-gen-go-errors" "github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest"

# OpenAPI 插件
install_tool "protoc-gen-openapi" "github.com/google/gnostic/cmd/protoc-gen-openapi@latest"

# 其他插件
install_tool "protoc-gen-validate" "github.com/envoyproxy/protoc-gen-validate@latest"
install_tool "protoc-gen-typescript-http" "github.com/go-kratos/protoc-gen-typescript-http@latest"

echo ""
log_info "Installing CLI tools..."
echo ""

# Kratos CLI
install_tool "kratos" "github.com/go-kratos/kratos/cmd/kratos/v2@latest"

# Wire - 依赖注入
install_tool "wire" "github.com/google/wire/cmd/wire@latest"

# Ent - ORM
install_tool "ent" "entgo.io/ent/cmd/ent@latest"

# GolangCI-Lint - 代码检查
install_tool "golangci-lint" "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

echo ""
log_success "All tools installed successfully! 🎉"
echo ""

# ============================================================================
# 验证安装
# ============================================================================

log_info "Verifying installations..."
echo ""

verify_tool() {
    local tool_name=$1
    if command -v "$tool_name" >/dev/null 2>&1; then
        log_success "$tool_name: $(command -v $tool_name)"
    else
        log_error "$tool_name: NOT FOUND"
    fi
}

verify_tool "buf"
verify_tool "protoc-gen-go"
verify_tool "protoc-gen-go-grpc"
verify_tool "protoc-gen-go-http"
verify_tool "protoc-gen-go-errors"
verify_tool "protoc-gen-openapi"
verify_tool "wire"
verify_tool "ent"
verify_tool "golangci-lint"

echo ""
log_success "Installation complete! 🚀"
echo ""
log_info "Next steps:"
log_info "  1. Make sure GOPATH/bin is in your PATH"
log_info "  2. Run 'make run-local' to start the service"
log_info "  3. Check 'make help' for more commands"
