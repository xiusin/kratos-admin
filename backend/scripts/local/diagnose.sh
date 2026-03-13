#!/usr/bin/env bash
################################################################################
## 本地开发环境诊断脚本
##
## 功能：
##   检查本地开发环境是否正确配置
##
## 检查项：
##   - Go 环境
##   - 开发工具 (buf, protoc-gen-*)
##   - 依赖服务 (PostgreSQL, Redis, MinIO)
##   - PATH 配置
##
## 用法：
##   bash scripts/local/diagnose.sh
##
################################################################################
set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log_info() { echo -e "${BLUE}[INFO]${NC} $*"; }
log_success() { echo -e "${GREEN}[✓]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[!]${NC} $*"; }
log_error() { echo -e "${RED}[✗]${NC} $*"; }
log_section() { echo -e "\n${CYAN}=== $* ===${NC}\n"; }

ISSUES=0
WARNINGS=0

# ============================================================================
# 检查 Go 环境
# ============================================================================

check_go() {
    log_section "Go Environment"
    
    if command -v go >/dev/null 2>&1; then
        log_success "Go installed: $(go version)"
        log_info "GOPATH: $(go env GOPATH)"
        log_info "GOROOT: $(go env GOROOT)"
        
        # 检查 GOPATH/bin 是否在 PATH 中
        GOBIN="$(go env GOPATH)/bin"
        if [[ ":$PATH:" == *":$GOBIN:"* ]]; then
            log_success "GOPATH/bin is in PATH"
        else
            log_error "GOPATH/bin is NOT in PATH"
            log_warn "Add this to your shell profile (~/.bashrc or ~/.zshrc):"
            echo "  export PATH=\"\$PATH:\$(go env GOPATH)/bin\""
            ((ISSUES++))
        fi
    else
        log_error "Go is not installed"
        log_warn "Install from: https://golang.org/dl/"
        ((ISSUES++))
    fi
}

# ============================================================================
# 检查开发工具
# ============================================================================

check_tools() {
    log_section "Development Tools"
    
    local tools=(
        "buf:github.com/bufbuild/buf/cmd/buf@latest"
        "protoc-gen-go:google.golang.org/protobuf/cmd/protoc-gen-go@latest"
        "protoc-gen-go-grpc:google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
        "protoc-gen-go-http:github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest"
        "protoc-gen-go-errors:github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest"
        "protoc-gen-openapi:github.com/google/gnostic/cmd/protoc-gen-openapi@latest"
        "wire:github.com/google/wire/cmd/wire@latest"
        "ent:entgo.io/ent/cmd/ent@latest"
    )
    
    for tool_info in "${tools[@]}"; do
        IFS=':' read -r tool_name tool_package <<< "$tool_info"
        
        if command -v "$tool_name" >/dev/null 2>&1; then
            log_success "$tool_name installed"
        else
            log_error "$tool_name NOT installed"
            log_warn "Install: go install $tool_package"
            ((ISSUES++))
        fi
    done
    
    if [ $ISSUES -gt 0 ]; then
        echo ""
        log_info "Quick fix: Run 'make install-tools' to install all missing tools"
    fi
}

# ============================================================================
# 检查依赖服务
# ============================================================================

check_services() {
    log_section "Dependency Services"
    
    # PostgreSQL
    if pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
        log_success "PostgreSQL is running (localhost:5432)"
    else
        log_warn "PostgreSQL is NOT running"
        log_info "Start: brew services start postgresql (macOS)"
        log_info "Or: docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=*Abcd123456 postgres"
        ((WARNINGS++))
    fi
    
    # Redis
    if redis-cli -h localhost -p 6379 ping >/dev/null 2>&1; then
        log_success "Redis is running (localhost:6379)"
    else
        log_warn "Redis is NOT running"
        log_info "Start: brew services start redis (macOS)"
        log_info "Or: docker run -d -p 6379:6379 redis redis-server --requirepass '*Abcd123456'"
        ((WARNINGS++))
    fi
    
    # MinIO (optional)
    if curl -s http://localhost:9000/minio/health/live >/dev/null 2>&1; then
        log_success "MinIO is running (localhost:9000)"
    else
        log_warn "MinIO is NOT running (optional)"
        log_info "Start: docker run -d -p 9000:9000 -p 9001:9001 -e MINIO_ROOT_USER=root -e MINIO_ROOT_PASSWORD=*Abcd123456 minio/minio server /data --console-address ':9001'"
    fi
    
    if [ $WARNINGS -gt 0 ]; then
        echo ""
        log_info "Quick fix: Run 'make docker-libs' to start all dependency services"
    fi
}

# ============================================================================
# 检查项目配置
# ============================================================================

check_project() {
    log_section "Project Configuration"
    
    # 检查配置文件
    if [ -f "app/admin/service/configs/data.yaml" ]; then
        log_success "Config file exists: data.yaml"
    else
        log_error "Config file NOT found: data.yaml"
        ((ISSUES++))
    fi
    
    if [ -f "app/admin/service/configs/data.local.yaml" ]; then
        log_success "Local config exists: data.local.yaml"
    else
        log_warn "Local config NOT found: data.local.yaml"
        log_info "Create: make setup-local-config"
    fi
    
    # 检查 API 代码是否生成
    if [ -d "api/gen/go" ]; then
        log_success "API code generated"
    else
        log_warn "API code NOT generated"
        log_info "Generate: cd api && buf generate"
    fi
}

# ============================================================================
# 生成诊断报告
# ============================================================================

generate_report() {
    log_section "Diagnosis Summary"
    
    if [ $ISSUES -eq 0 ] && [ $WARNINGS -eq 0 ]; then
        log_success "All checks passed! Your environment is ready. 🎉"
        echo ""
        log_info "Next steps:"
        echo "  1. Run 'make run-local' to start the service"
        echo "  2. Visit http://localhost:7788/docs/ for API documentation"
    else
        if [ $ISSUES -gt 0 ]; then
            log_error "Found $ISSUES critical issue(s) that need to be fixed"
        fi
        
        if [ $WARNINGS -gt 0 ]; then
            log_warn "Found $WARNINGS warning(s)"
        fi
        
        echo ""
        log_info "Quick fixes:"
        echo "  • Install tools: make install-tools"
        echo "  • Start services: make docker-libs"
        echo "  • Setup config: make setup-local-config"
        echo ""
        log_info "For detailed help, see: docs/LOCAL_DEVELOPMENT.md"
    fi
}

# ============================================================================
# 主流程
# ============================================================================

main() {
    echo ""
    log_info "🔍 Diagnosing local development environment..."
    echo ""
    
    check_go
    check_tools
    check_services
    check_project
    generate_report
    
    echo ""
}

main
