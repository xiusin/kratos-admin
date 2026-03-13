#!/usr/bin/env bash
################################################################################
## 本地开发启动脚本 - 纯本地模式（不使用 Docker）
##
## 功能：
##   启动 admin service 服务（纯本地模式）
##
## 前置条件：
##   - PostgreSQL 已启动 (localhost:5432)
##   - Redis 已启动 (localhost:6379)
##   - MinIO 已启动 (localhost:9000) [可选]
##
## 使用场景：
##   1. 本地开发调试
##   2. IDE 断点调试
##   3. 快速迭代开发
##
## 用法：
##   bash scripts/local/start_local.sh
##
## 环境变量：
##   CONFIG_PATH     配置文件路径 (默认: ./configs)
##   LOG_LEVEL       日志级别 (默认: debug)
##
## 示例：
##   # 使用默认配置启动
##   bash scripts/local/start_local.sh
##
##   # 使用自定义配置
##   CONFIG_PATH=./configs/local bash scripts/local/start_local.sh
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

# 切换到项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_ROOT="$SCRIPT_DIR/../.."
cd "$BACKEND_ROOT" || exit 1

log_info "Backend root: $BACKEND_ROOT"

# 配置路径
CONFIG_PATH=${CONFIG_PATH:-./configs}
SERVICE_DIR="app/admin/service"

# ============================================================================
# 检查开发工具
# ============================================================================

check_dev_tools() {
    log_info "Checking development tools..."
    echo ""
    
    local all_ok=true
    
    # 检查 Go
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed!"
        log_warn "  → Install Go: https://golang.org/dl/"
        all_ok=false
    else
        log_success "Go $(go version | awk '{print $3}') ✓"
    fi
    
    # 检查 buf
    if ! command -v buf >/dev/null 2>&1; then
        log_warn "buf is not installed, installing..."
        if go install github.com/bufbuild/buf/cmd/buf@latest; then
            log_success "buf installed successfully ✓"
            # 确保 GOBIN 在 PATH 中
            export PATH="$PATH:$(go env GOPATH)/bin"
        else
            log_error "Failed to install buf"
            log_warn "  → Manual install: go install github.com/bufbuild/buf/cmd/buf@latest"
            log_warn "  → Or use: brew install bufbuild/buf/buf (macOS)"
            all_ok=false
        fi
    else
        log_success "buf $(buf --version 2>&1 | head -n1) ✓"
    fi
    
    # 检查 protoc-gen-go
    if ! command -v protoc-gen-go >/dev/null 2>&1; then
        log_warn "protoc-gen-go is not installed, installing..."
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
        log_success "protoc-gen-go installed ✓"
    fi
    
    # 检查 protoc-gen-go-grpc
    if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
        log_warn "protoc-gen-go-grpc is not installed, installing..."
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
        log_success "protoc-gen-go-grpc installed ✓"
    fi
    
    echo ""
    
    if [ "$all_ok" = false ]; then
        log_error "部分开发工具未安装，请先安装必需工具！"
        log_info "或运行: make init (安装所有开发工具)"
        exit 1
    fi
    
    log_success "所有开发工具检查通过！"
    echo ""
}

# ============================================================================
# 检查依赖服务
# ============================================================================

check_service() {
    local service_name=$1
    local check_cmd=$2
    
    log_info "Checking $service_name..."
    if eval "$check_cmd" > /dev/null 2>&1; then
        log_success "$service_name is running ✓"
        return 0
    else
        log_error "$service_name is NOT running ✗"
        return 1
    fi
}

check_dependencies() {
    log_info "Checking local services..."
    echo ""
    
    local all_ok=true
    
    # 检查 PostgreSQL
    if ! check_service "PostgreSQL" "pg_isready -h localhost -p 5432"; then
        all_ok=false
        log_warn "  → 启动 PostgreSQL: brew services start postgresql"
        log_warn "  → 或使用 Docker: docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=*Abcd123456 postgres"
    fi
    
    # 检查 Redis
    if ! check_service "Redis" "redis-cli -h localhost -p 6379 ping"; then
        all_ok=false
        log_warn "  → 启动 Redis: brew services start redis"
        log_warn "  → 或使用 Docker: docker run -d -p 6379:6379 redis redis-server --requirepass '*Abcd123456'"
    fi
    
    # 检查 MinIO (可选)
    if ! check_service "MinIO (optional)" "curl -s http://localhost:9000/minio/health/live"; then
        log_warn "  → MinIO 未运行（可选服务）"
        log_warn "  → 启动 MinIO: docker run -d -p 9000:9000 -p 9001:9001 -e MINIO_ROOT_USER=root -e MINIO_ROOT_PASSWORD=*Abcd123456 minio/minio server /data --console-address ':9001'"
    fi
    
    echo ""
    
    if [ "$all_ok" = false ]; then
        log_error "部分依赖服务未启动，请先启动依赖服务！"
        log_info "或者使用 'make docker-libs' 启动 Docker 依赖服务"
        exit 1
    fi
    
    log_success "所有依赖服务检查通过！"
    echo ""
}

# ============================================================================
# 启动服务
# ============================================================================

start_service() {
    log_info "Starting admin service..."
    echo ""
    
    cd "$SERVICE_DIR" || exit 1
    
    # 生成 API 代码
    log_info "Generating API code..."
    make api > /dev/null 2>&1 || log_warn "API generation failed (may be already generated)"
    
    # 生成 OpenAPI 文档
    log_info "Generating OpenAPI docs..."
    make openapi > /dev/null 2>&1 || log_warn "OpenAPI generation failed (may be already generated)"
    
    # 启动服务
    log_success "Starting service on http://localhost:7788"
    log_info "Press Ctrl+C to stop"
    echo ""
    echo "=========================================="
    echo ""
    
    # 运行服务
    go run -ldflags "-X main.version=dev-local" ./cmd/server -c "$CONFIG_PATH"
}

# ============================================================================
# 主流程
# ============================================================================

main() {
    log_info "🚀 Starting local development mode..."
    echo ""
    
    # 检查开发工具
    check_dev_tools
    
    # 检查依赖
    check_dependencies
    
    # 启动服务
    start_service
}

# 捕获 Ctrl+C
trap 'log_warn "Service stopped by user"; exit 0' INT TERM

main
