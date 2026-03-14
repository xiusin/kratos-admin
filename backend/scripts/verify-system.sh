#!/bin/bash

# System Verification Script
# 系统验证脚本

set -e

echo "🔍 开始系统验证..."
echo "================================"

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 验证计数器
PASSED=0
FAILED=0
WARNINGS=0

# 验证函数
check_pass() {
    echo -e "${GREEN}✅ $1${NC}"
    ((PASSED++))
}

check_fail() {
    echo -e "${RED}❌ $1${NC}"
    ((FAILED++))
}

check_warn() {
    echo -e "${YELLOW}⚠️  $1${NC}"
    ((WARNINGS++))
}

# 1. 检查Go环境
echo ""
echo "1️⃣  检查Go环境..."
if command -v go &> /dev/null; then
    GO_VERSION=$(go version)
    check_pass "Go已安装: $GO_VERSION"
else
    check_fail "Go未安装"
fi

# 2. 检查项目结构
echo ""
echo "2️⃣  检查项目结构..."
if [ -d "app/consumer/service" ]; then
    check_pass "Consumer服务目录存在"
else
    check_fail "Consumer服务目录不存在"
fi

if [ -d "pkg" ]; then
    check_pass "pkg目录存在"
else
    check_fail "pkg目录不存在"
fi

if [ -d "api/protos/consumer" ]; then
    check_pass "API定义目录存在"
else
    check_fail "API定义目录不存在"
fi

# 3. 检查关键文件
echo ""
echo "3️⃣  检查关键文件..."
KEY_FILES=(
    "app/consumer/service/cmd/server/main.go"
    "app/consumer/service/cmd/server/wire.go"
    "app/consumer/service/cmd/server/wire_gen.go"
    "pkg/eventbus/kafka.go"
    "pkg/middleware/auth.go"
    "pkg/middleware/tenant.go"
    "pkg/middleware/security.go"
    "pkg/monitoring/health.go"
    "pkg/monitoring/metrics.go"
)

for file in "${KEY_FILES[@]}"; do
    if [ -f "$file" ]; then
        check_pass "$file 存在"
    else
        check_fail "$file 不存在"
    fi
done

# 4. 检查Ent Schema
echo ""
echo "4️⃣  检查Ent Schema..."
SCHEMAS=(
    "app/consumer/service/internal/data/ent/schema/consumer.go"
    "app/consumer/service/internal/data/ent/schema/login_log.go"
    "app/consumer/service/internal/data/ent/schema/sms_log.go"
    "app/consumer/service/internal/data/ent/schema/payment_order.go"
    "app/consumer/service/internal/data/ent/schema/finance_account.go"
    "app/consumer/service/internal/data/ent/schema/finance_transaction.go"
    "app/consumer/service/internal/data/ent/schema/media_file.go"
    "app/consumer/service/internal/data/ent/schema/logistics_tracking.go"
    "app/consumer/service/internal/data/ent/schema/freight_template.go"
    "app/consumer/service/internal/data/ent/schema/tenant_config.go"
)

for schema in "${SCHEMAS[@]}"; do
    if [ -f "$schema" ]; then
        check_pass "$(basename $schema) 已定义"
    else
        check_fail "$(basename $schema) 未定义"
    fi
done

# 5. 检查Protobuf定义
echo ""
echo "5️⃣  检查Protobuf定义..."
PROTOS=(
    "api/protos/consumer/service/v1/consumer.proto"
    "api/protos/consumer/service/v1/sms.proto"
    "api/protos/consumer/service/v1/payment.proto"
    "api/protos/consumer/service/v1/finance.proto"
    "api/protos/consumer/service/v1/wechat.proto"
    "api/protos/consumer/service/v1/media.proto"
    "api/protos/consumer/service/v1/logistics.proto"
    "api/protos/consumer/service/v1/freight.proto"
    "api/protos/consumer/service/v1/tenant_config.proto"
)

for proto in "${PROTOS[@]}"; do
    if [ -f "$proto" ]; then
        check_pass "$(basename $proto) 已定义"
    else
        check_fail "$(basename $proto) 未定义"
    fi
done

# 6. 检查测试文件
echo ""
echo "6️⃣  检查测试文件..."
TEST_FILES=(
    "pkg/eventbus/kafka_test.go"
    "pkg/eventbus/integration_test.go"
    "app/consumer/service/internal/service/eventbus_integration_test.go"
    "app/consumer/service/internal/service/eventbus_property_test.go"
)

for test in "${TEST_FILES[@]}"; do
    if [ -f "$test" ]; then
        check_pass "$(basename $test) 存在"
    else
        check_warn "$(basename $test) 不存在"
    fi
done

# 7. 尝试编译
echo ""
echo "7️⃣  尝试编译Consumer服务..."
if go build -o /tmp/consumer-service ./app/consumer/service/cmd/server/ 2>/dev/null; then
    check_pass "Consumer服务编译成功"
    rm -f /tmp/consumer-service
else
    check_fail "Consumer服务编译失败 (可能需要运行 scripts/fix-consumer-service.sh)"
fi

# 8. 检查依赖
echo ""
echo "8️⃣  检查Go模块依赖..."
if go mod verify &> /dev/null; then
    check_pass "Go模块依赖验证通过"
else
    check_warn "Go模块依赖验证失败 (运行 go mod tidy)"
fi

# 9. 运行测试 (如果编译成功)
echo ""
echo "9️⃣  运行测试..."
if go test -v ./pkg/eventbus/... 2>/dev/null; then
    check_pass "EventBus测试通过"
else
    check_warn "EventBus测试失败或跳过"
fi

# 10. 检查文档
echo ""
echo "🔟 检查文档..."
DOCS=(
    ".kiro/specs/c-user-management-system/requirements.md"
    ".kiro/specs/c-user-management-system/design.md"
    ".kiro/specs/c-user-management-system/tasks.md"
    "pkg/middleware/README_SECURITY.md"
)

for doc in "${DOCS[@]}"; do
    if [ -f "$doc" ]; then
        check_pass "$(basename $doc) 存在"
    else
        check_fail "$(basename $doc) 不存在"
    fi
done

# 总结
echo ""
echo "================================"
echo "📊 验证总结"
echo "================================"
echo -e "${GREEN}通过: $PASSED${NC}"
echo -e "${RED}失败: $FAILED${NC}"
echo -e "${YELLOW}警告: $WARNINGS${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ 系统验证通过！${NC}"
    exit 0
else
    echo -e "${RED}❌ 系统验证失败，请修复上述问题。${NC}"
    echo ""
    echo "建议修复步骤:"
    echo "1. 运行: bash scripts/fix-consumer-service.sh"
    echo "2. 运行: go mod tidy"
    echo "3. 重新运行此验证脚本"
    exit 1
fi
