# Payment Service 设置指南

## 完成状态

✅ 所有代码已实现完成
⚠️ 需要手动执行依赖生成和编译验证

## 手动操作步骤

### 1. 更新Go模块依赖

```bash
cd backend
go mod tidy
```

**说明**: 更新go.mod和go.sum,确保所有依赖正确解析

### 2. 生成Wire依赖注入代码

```bash
cd backend/app/consumer/service
go run github.com/google/wire/cmd/wire ./cmd/server
```

**预期输出**:
```
wire: go-wind-admin/app/consumer/service/cmd/server: wrote /path/to/wire_gen.go
```

**说明**: Wire会生成`cmd/server/wire_gen.go`文件,包含所有依赖注入代码

### 3. 验证编译

```bash
cd backend/app/consumer/service
go build ./cmd/server
```

**预期输出**: 无错误,生成可执行文件

### 4. 代码格式化(可选)

```bash
cd backend/app/consumer/service
gofmt -l -w .
```

### 5. Lint检查(可选)

```bash
cd backend
golangci-lint run ./app/consumer/service/...
```

## 常见问题排查

### 问题1: package not in std

**错误信息**:
```
package go-wind-admin/api/gen/go/pagination/v1 is not in std
```

**解决方案**:
```bash
cd backend
go mod tidy
go clean -modcache
go mod download
```

### 问题2: undefined: PkgProviderSet

**原因**: pkg_providers.go文件位置错误

**解决方案**: 
- ✅ 已修复: 文件已移动到`cmd/server/pkg_providers.go`

### 问题3: Wire生成失败

**可能原因**:
1. Wire未安装
2. 文件路径错误
3. 包导入错误

**解决方案**:
```bash
# 安装wire
go install github.com/google/wire/cmd/wire@latest

# 重新生成
cd backend/app/consumer/service/cmd/server
wire
```

## 验证清单

完成以上步骤后,请验证:

- [ ] `go mod tidy` 执行成功,无错误
- [ ] `wire` 生成成功,创建了`wire_gen.go`
- [ ] `go build` 编译成功,无错误
- [ ] `gofmt` 格式化完成
- [ ] 所有文件在正确的位置

## 文件清单

### 新增文件:
1. ✅ `backend/app/consumer/service/internal/data/payment_order_repo.go`
2. ✅ `backend/app/consumer/service/internal/service/payment_service.go`
3. ✅ `backend/app/consumer/service/cmd/server/pkg_providers.go`

### 修改文件:
1. ✅ `backend/app/consumer/service/internal/service/providers/wire_set.go`
2. ✅ `backend/app/consumer/service/internal/data/providers/wire_set.go`
3. ✅ `backend/app/consumer/service/internal/server/rest_server.go`

### 生成文件(执行wire后):
1. ⚠️ `backend/app/consumer/service/cmd/server/wire_gen.go` (需要生成)

## 下一步

完成上述步骤后,Payment Service就可以正常运行了。

后续可以:
1. 配置真实的支付配置(替换硬编码)
2. 添加支付回调HTTP接口
3. 配置订单超时关闭定时任务
4. 编写单元测试和属性测试
5. 集成测试完整支付流程

## 技术支持

如果遇到问题,请检查:
1. Go版本 >= 1.21
2. Wire已安装
3. 在正确的目录执行命令
4. go.mod文件完整
5. 网络连接正常(下载依赖)
