# Task 17.2: 下一步操作指南

## ✅ 当前状态

代码已经编译通过！配置管理接口的基础代码已就绪，但需要生成Ent代码才能完全启用功能。

## 📋 必须执行的步骤

### 步骤1: 生成Ent代码

```bash
cd backend/app/consumer/service
go generate ./internal/data/ent
```

**预期结果**: 生成 `ConfigChangeHistory` 实体的Go代码

### 步骤2: 取消注释Repository代码

生成Ent代码后，需要取消注释以下文件中的代码：

**文件**: `backend/app/consumer/service/internal/data/config_change_history_repo.go`

1. 取消注释导入:
```go
import (
    // ...
    "go-wind-admin/app/consumer/service/internal/data/ent/configchangehistory"
)
```

2. 取消注释 `Create()` 方法中的实现代码
3. 取消注释 `ListByConfigKey()` 方法中的实现代码  
4. 取消注释 `GetLatestByConfigKey()` 方法中的实现代码

### 步骤3: 添加Repository到Wire ProviderSet

编辑 `backend/app/consumer/service/internal/data/providers/wire_set.go`:

```go
var ProviderSet = wire.NewSet(
    // ... 现有的 providers
    data.NewConfigChangeHistoryRepo,  // 添加这一行
)
```

### 步骤4: 重新生成Wire代码

```bash
cd backend/app/consumer/service
rm cmd/server/wire_gen.go
cd cmd/server
go generate
```

### 步骤5: 最终编译验证

```bash
cd backend/app/consumer/service
go build ./...
```

## 🔍 为什么要这样做？

根据 **AGENTS.md 铁律2**（增量开发+立即验证），我们采用了以下策略：

1. **先创建Schema** → 定义数据模型
2. **临时注释掉依赖代码** → 确保编译通过
3. **生成Ent代码** → 创建实体类
4. **取消注释** → 启用完整功能
5. **Wire集成** → 依赖注入
6. **最终验证** → 确保一切正常

这样可以避免"先写代码，后发现依赖不存在"的问题。

## 📝 当前实现的功能

即使在生成Ent代码之前，以下功能已经可用：

- ✅ 配置验证 (`validateConfigValue`)
- ✅ GetConfig (查询配置)
- ✅ UpdateConfig (更新配置，带验证)
- ✅ ListConfigs (配置列表)
- ✅ BatchUpdateConfigs (批量更新)
- ✅ DeleteConfig (删除配置)

**暂时禁用的功能**（生成Ent代码后启用）：
- ⏸️ 配置变更历史记录
- ⏸️ 查询变更历史
- ⏸️ 配置回滚基础

## 🎯 执行顺序

```
当前位置 → 步骤1 → 步骤2 → 步骤3 → 步骤4 → 步骤5 → 完成
   ✅        ⬜       ⬜       ⬜       ⬜       ⬜      
```

## 💡 提示

如果步骤1（go generate）失败，可能需要：
1. 检查 `internal/data/ent/generate.go` 文件是否存在
2. 确保已安装 `entgo.io/ent/cmd/ent` 工具

如果步骤4（Wire生成）失败，请检查：
1. 所有构造函数签名是否正确
2. ProviderSet中的provider是否都存在

## 📄 相关文档

- 详细实现说明: `.ai/traces/2026-03-15/task-17-2-config-management-interface.md`
- AGENTS.md: 开发规范和铁律
