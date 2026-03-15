#!/bin/bash

# AI 代码生成前强制验证脚本
# 使用方法：在生成代码前，AI 必须运行此脚本

echo "🚨 AI 代码生成前验证检查"
echo "================================"

# 检查1: Pagination 导入
echo ""
echo "✅ 检查1: 验证 Pagination 导入路径"
echo "执行命令: grep -r 'paginationV1' backend/app/*/service/internal/service/*.go | head -1"
PAGINATION_IMPORT=$(grep -r "paginationV1" backend/app/*/service/internal/service/*.go 2>/dev/null | head -1)
if [ -n "$PAGINATION_IMPORT" ]; then
    echo "找到: $PAGINATION_IMPORT"
    if echo "$PAGINATION_IMPORT" | grep -q "github.com/tx7do/go-crud"; then
        echo "✅ 正确：使用外部包 github.com/tx7do/go-crud"
    else
        echo "❌ 错误：未使用正确的包路径"
        exit 1
    fi
else
    echo "⚠️  未找到 pagination 导入（如果不需要则忽略）"
fi

# 检查2: 列出参考实现
echo ""
echo "✅ 检查2: 列出可用的参考实现"
echo "Service 文件:"
ls -1 backend/app/consumer/service/internal/service/*.go 2>/dev/null | head -5
echo ""
echo "Repository 文件:"
ls -1 backend/app/consumer/service/internal/data/*_repo.go 2>/dev/null | head -5

# 检查3: Wire 配置
echo ""
echo "✅ 检查3: 检查 Wire 配置文件"
if [ -f "backend/app/consumer/service/cmd/server/wire_gen.go" ]; then
    echo "✅ wire_gen.go 存在"
else
    echo "⚠️  wire_gen.go 不存在（可能需要生成）"
fi

echo ""
echo "================================"
echo "✅ 验证完成！"
echo ""
echo "📝 下一步："
echo "1. 选择参考实现文件"
echo "2. 查看其导入部分: head -30 <文件> | grep -A 20 'import'"
echo "3. 查看构造函数: grep -A 5 'func New' <文件>"
echo "4. 复制模式，不要假设"
