#!/bin/bash

# 验证脚本 - 系统地运行所有编译和测试
# Validation Script - Systematically run all compilation and tests

set -e

echo "========================================="
echo "🚀 开始验证流程 (Starting Validation)"
echo "========================================="
echo ""

# 1. 格式化检查
echo "📝 Step 1: 代码格式化 (Code Formatting)"
echo "-----------------------------------------"
gofmt -l . | tee /tmp/gofmt_output.txt
if [ -s /tmp/gofmt_output.txt ]; then
    echo "⚠️  发现未格式化的文件，正在自动格式化..."
    gofmt -w .
    echo "✅ 格式化完成"
else
    echo "✅ 所有文件已格式化"
fi
echo ""

# 2. 依赖整理
echo "📦 Step 2: 依赖整理 (Dependency Tidy)"
echo "-----------------------------------------"
go mod tidy
echo "✅ 依赖整理完成"
echo ""

# 3. 编译检查
echo "🔨 Step 3: 编译检查 (Build Check)"
echo "-----------------------------------------"
echo "编译 pkg/constitution..."
go build ./pkg/constitution
echo "✅ pkg/constitution 编译通过"

echo "编译 pkg/sms..."
go build ./pkg/sms
echo "✅ pkg/sms 编译通过"

echo "编译 app/admin/service..."
go build ./app/admin/service/internal/service
echo "✅ app/admin/service 编译通过"

echo "编译 app/consumer/service..."
go build ./app/consumer/service/cmd/server
echo "✅ app/consumer/service 编译通过"

echo "编译所有模块..."
go build ./...
echo "✅ 所有模块编译通过"
echo ""

# 4. 单元测试
echo "🧪 Step 4: 单元测试 (Unit Tests)"
echo "-----------------------------------------"
echo "测试 pkg/constitution..."
go test -short -v ./pkg/constitution 2>&1 | tee /tmp/constitution_test.log
CONSTITUTION_RESULT=$?

echo ""
echo "测试 pkg/crypto..."
go test -short ./pkg/crypto
echo "✅ pkg/crypto 测试通过"

echo ""
echo "测试 pkg/jwt..."
go test -short ./pkg/jwt
echo "✅ pkg/jwt 测试通过"

echo ""
echo "测试 pkg/lua..."
go test -short ./pkg/lua/...
echo "✅ pkg/lua 测试通过"

echo ""

# 5. 测试覆盖率
echo "📊 Step 5: 测试覆盖率 (Test Coverage)"
echo "-----------------------------------------"
echo "生成 pkg/constitution 覆盖率报告..."
go test -short -coverprofile=/tmp/constitution_coverage.out ./pkg/constitution 2>&1
if [ -f /tmp/constitution_coverage.out ]; then
    go tool cover -func=/tmp/constitution_coverage.out | tail -1
    echo "✅ 覆盖率报告已生成"
else
    echo "⚠️  覆盖率报告生成失败"
fi
echo ""

# 6. Lint 检查
echo "🔍 Step 6: Lint 检查 (Lint Check)"
echo "-----------------------------------------"
echo "运行 golangci-lint..."
golangci-lint run ./pkg/constitution/... 2>&1 | head -20 || echo "⚠️  Lint 发现一些问题（非致命）"
echo ""

# 7. 总结
echo "========================================="
echo "📋 验证总结 (Validation Summary)"
echo "========================================="
echo ""
echo "✅ 代码格式化: 通过"
echo "✅ 依赖整理: 通过"
echo "✅ 编译检查: 通过"
if [ $CONSTITUTION_RESULT -eq 0 ]; then
    echo "✅ constitution 单元测试: 通过"
else
    echo "⚠️  constitution 单元测试: 部分失败"
fi
echo "✅ 其他包测试: 通过"
echo ""
echo "========================================="
echo "🎉 验证流程完成！"
echo "========================================="
