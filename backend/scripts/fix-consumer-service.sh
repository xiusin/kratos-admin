#!/bin/bash

# Fix Consumer Service Build Issues
# 修复Consumer服务编译问题

set -e

echo "🔧 开始修复Consumer服务编译问题..."

# 1. 添加缺失的OpenTelemetry依赖
echo "📦 添加OpenTelemetry Jaeger依赖..."
go get go.opentelemetry.io/otel/exporters/jaeger@latest

# 2. 生成Ent代码
echo "🏗️  生成Ent代码..."
cd app/consumer/service/internal/data/ent
go generate
cd -

# 3. 整理依赖
echo "📚 整理Go模块依赖..."
go mod tidy

# 4. 验证编译
echo "🔨 验证编译..."
go build ./app/consumer/service/cmd/server/

echo "✅ 修复完成！Consumer服务编译成功。"
