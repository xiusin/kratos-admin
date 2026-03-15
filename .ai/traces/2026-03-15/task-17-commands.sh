#!/bin/bash
# Task 17.1 - 配置管理实现验证命令
# 项目路径: /Users/xiusin/Desktop/kratos-admin
# 日期: 2026-03-15

set -e  # 遇到错误立即退出

echo "=========================================="
echo "Task 17.1 - 配置管理实现验证"
echo "=========================================="
echo ""

# 步骤 1: 生成 Protobuf 代码
echo "步骤 1: 生成 Protobuf 代码..."
cd /Users/xiusin/Desktop/kratos-admin/backend/api && buf generate
echo "✅ Protobuf 代码生成完成"
echo ""

# 步骤 2: 生成 Ent 代码
echo "步骤 2: 生成 Ent 代码..."
cd /Users/xiusin/Desktop/kratos-admin/backend/app/consumer/service && go generate ./internal/data/ent
echo "✅ Ent 代码生成完成"
echo ""

# 步骤 3: 格式化代码
echo "步骤 3: 格式化代码..."
cd /Users/xiusin/Desktop/kratos-admin/backend/app/consumer/service && gofmt -l -w .
echo "✅ 代码格式化完成"
echo ""

# 步骤 4: 编译验证
echo "步骤 4: 编译验证..."
cd /Users/xiusin/Desktop/kratos-admin/backend/app/consumer/service && go build ./...
echo "✅ 编译验证通过"
echo ""

echo "=========================================="
echo "所有步骤完成！"
echo "=========================================="
echo ""
echo "下一步："
echo "1. 查看生成的文件是否正确"
echo "2. 继续执行手动集成步骤（见下方）"
