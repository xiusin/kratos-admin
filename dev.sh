#!/usr/bin/env bash
# GoWind Admin 本地开发启动脚本
# 用法: ./dev.sh [init|gen|all|backend|frontend|stop]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
FRONTEND_DIR="$SCRIPT_DIR/frontend"
ADMIN_SERVICE_DIR="$BACKEND_DIR/app/admin/service"
LOG_DIR="$SCRIPT_DIR/.dev-logs"
PID_FILE="$SCRIPT_DIR/.dev.pid"

mkdir -p "$LOG_DIR"

# ── 颜色输出 ──────────────────────────────────────────────
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()    { echo -e "${CYAN}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERR]${NC}  $*"; }
step()    { echo -e "\n${BOLD}▶ $*${NC}"; }

# ── 检查命令是否存在 ──────────────────────────────────────
has_cmd() { command -v "$1" > /dev/null 2>&1; }

# ── 初始化：安装工具链 + 生成代码 ─────────────────────────
init_backend() {
    step "初始化后端开发环境"

    export PATH="$PATH:$(go env GOPATH)/bin"

    # 1. 安装 Go 工具链
    step "安装 Protobuf 插件"
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
    go install github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest
    go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
    go install github.com/envoyproxy/protoc-gen-validate@latest
    go install github.com/menta2k/protoc-gen-redact/v3@latest
    success "Protobuf 插件安装完成"

    step "安装 CLI 工具"
    go install github.com/bufbuild/buf/cmd/buf@latest
    go install entgo.io/ent/cmd/ent@latest
    go install github.com/google/wire/cmd/wire@latest
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    success "CLI 工具安装完成"

    # 2. 下载 Go 依赖
    step "下载 Go 依赖"
    cd "$BACKEND_DIR"
    go mod download
    success "Go 依赖下载完成"

    # 3. 生成代码
    gen_code

    # 4. 预编译（避免首次启动慢）
    step "预编译后端服务"
    cd "$ADMIN_SERVICE_DIR"
    go build -o ./bin/server ./cmd/server
    success "编译完成 → $ADMIN_SERVICE_DIR/bin/server"

    echo ""
    success "初始化完成，运行 ./dev.sh 启动服务"
}

# ── 代码生成 ──────────────────────────────────────────────
gen_code() {
    export PATH="$PATH:$(go env GOPATH)/bin"

    # 检查工具
    for tool in buf ent wire; do
        if ! has_cmd "$tool"; then
            error "工具 '$tool' 未安装，请先运行: ./dev.sh init"
            exit 1
        fi
    done

    # 1. buf generate → Protobuf Go 代码 + OpenAPI 文档
    step "buf generate：生成 Protobuf Go 代码"
    cd "$BACKEND_DIR/api"
    buf generate
    success "Protobuf Go 代码生成完成 → api/gen/go/"

    step "buf generate：生成 OpenAPI 文档"
    buf generate --template buf.admin.openapi.gen.yaml
    success "OpenAPI 文档生成完成"

    # 2. ent generate → ORM 代码
    step "ent generate：生成 Ent ORM 代码"
    cd "$ADMIN_SERVICE_DIR"
    ent generate \
        --feature privacy \
        --feature entql \
        --feature sql/modifier \
        --feature sql/upsert \
        --feature sql/lock \
        ./internal/data/ent/schema
    success "Ent ORM 代码生成完成 → internal/data/ent/"

    # 3. wire generate → 依赖注入代码
    step "wire generate：生成 Wire 依赖注入代码"
    cd "$ADMIN_SERVICE_DIR/cmd/server"
    wire
    success "Wire 代码生成完成 → cmd/server/wire_gen.go"

    echo ""
    success "所有代码生成完成"
}

# ── 检查依赖服务 ──────────────────────────────────────────
check_services() {
    info "检查本地依赖服务..."
    local failed=0

    if pg_isready -h localhost -p 5432 -q 2>/dev/null; then
        success "PostgreSQL 运行中 (localhost:5432)"
    else
        error "PostgreSQL 未运行，请执行: brew services start postgresql@15"
        failed=1
    fi

    if redis-cli -a "*Abcd123456" ping 2>/dev/null | grep -q PONG; then
        success "Redis 运行中 (localhost:6379)"
    else
        error "Redis 未运行，请执行: brew services start redis"
        failed=1
    fi

    if [ $failed -ne 0 ]; then
        echo ""
        error "依赖服务未就绪，请先启动后重试"
        exit 1
    fi
    echo ""
}

# ── 启动后端 ──────────────────────────────────────────────
start_backend() {
    info "启动后端服务..."
    export PATH="$PATH:$(go env GOPATH)/bin"

    cd "$ADMIN_SERVICE_DIR"

    # 优先使用预编译的二进制，否则 go run
    if [ -f "./bin/server" ]; then
        ./bin/server -c ./configs > "$LOG_DIR/backend.log" 2>&1 &
    else
        warn "未找到预编译文件，使用 go run（首次较慢）..."
        warn "建议先运行 ./dev.sh init 完成预编译"
        go run ./cmd/server -c ./configs > "$LOG_DIR/backend.log" 2>&1 &
    fi

    local pid=$!
    echo "backend=$pid" >> "$PID_FILE"

    # 等待服务启动（最多 60s，兼容 go run 首次下载依赖）
    info "等待后端启动..."
    local retries=0
    while [ $retries -lt 60 ]; do
        if curl -s http://localhost:7788/docs/ > /dev/null 2>&1; then
            success "后端已启动 → http://localhost:7788"
            success "Swagger  → http://localhost:7788/docs/"
            return 0
        fi
        sleep 1
        retries=$((retries + 1))
    done

    warn "后端启动超时，请查看日志: $LOG_DIR/backend.log"
}

# ── 启动前端 ──────────────────────────────────────────────
start_frontend() {
    info "启动前端服务..."
    cd "$FRONTEND_DIR"

    if [ ! -d "node_modules" ]; then
        info "安装前端依赖（首次较慢）..."
        pnpm install
    fi

    pnpm dev:antd > "$LOG_DIR/frontend.log" 2>&1 &
    local pid=$!
    echo "frontend=$pid" >> "$PID_FILE"

    info "等待前端启动..."
    local retries=0
    while [ $retries -lt 60 ]; do
        if curl -s http://localhost:5666 > /dev/null 2>&1; then
            success "前端已启动 → http://localhost:5666"
            return 0
        fi
        sleep 1
        retries=$((retries + 1))
    done

    warn "前端启动超时，请查看日志: $LOG_DIR/frontend.log"
}

# ── 停止所有服务 ──────────────────────────────────────────
stop_all() {
    info "停止所有开发服务..."

    if [ ! -f "$PID_FILE" ]; then
        warn "没有找到运行中的服务"
        return
    fi

    while IFS='=' read -r name pid; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null && success "已停止 $name (PID $pid)"
        fi
    done < "$PID_FILE"

    rm -f "$PID_FILE"
}

# ── 打印访问地址 ──────────────────────────────────────────
print_urls() {
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  前端地址:  http://localhost:5666${NC}"
    echo -e "${GREEN}  后端 API:  http://localhost:7788${NC}"
    echo -e "${GREEN}  Swagger:   http://localhost:7788/docs/${NC}"
    echo -e "${GREEN}  账号/密码: admin / admin${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "日志目录: $LOG_DIR"
    echo "停止服务: ./dev.sh stop"
    echo ""
}

# ── 主入口 ────────────────────────────────────────────────
MODE="${1:-all}"

[ "$MODE" != "stop" ] && rm -f "$PID_FILE"

case "$MODE" in
    init)
        # 首次使用：安装工具 + 生成代码 + 预编译
        init_backend
        ;;
    gen)
        # 仅重新生成代码（改了 proto/schema/provider 后用）
        gen_code
        ;;
    backend)
        # check_services
        start_backend
        ;;
    frontend)
        start_frontend
        ;;
    all)
        # check_services
        start_backend
        start_frontend
        print_urls
        ;;
    stop)
        stop_all
        ;;
    *)
        echo ""
        echo -e "${BOLD}用法: $0 [命令]${NC}"
        echo ""
        echo "  init      首次使用：安装工具链 + 生成代码 + 预编译"
        echo "  gen       重新生成代码（buf / ent / wire）"
        echo "  all       启动后端 + 前端（默认）"
        echo "  backend   只启动后端"
        echo "  frontend  只启动前端"
        echo "  stop      停止所有服务"
        echo ""
        echo -e "${CYAN}快速上手:${NC}"
        echo "  ./dev.sh init   # 首次初始化"
        echo "  ./dev.sh        # 启动所有服务"
        echo "  ./dev.sh stop   # 停止所有服务"
        echo ""
        exit 1
        ;;
esac
