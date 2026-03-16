# C端用户管理系统 - 路由配置说明

## 概述

本文档说明 C 端用户管理系统的前端路由配置。所有路由已在 `consumer.ts` 文件中配置完成。

## 路由结构

### 1. 用户认证模块 (`/consumer/auth`)

| 路由路径 | 组件 | 权限 | 说明 |
|---------|------|------|------|
| `/consumer/auth/register` | `auth/register.vue` | `consumer:auth:register` | 用户注册页面（无需登录） |
| `/consumer/auth/login` | `auth/login.vue` | `consumer:auth:login` | 用户登录页面（无需登录） |

**特殊配置：**
- `ignoreAccess: true` - 注册和登录页面不需要认证即可访问

### 2. 用户信息模块 (`/consumer/user`)

| 路由路径 | 组件 | 权限 | 说明 |
|---------|------|------|------|
| `/consumer/user/profile` | `user/profile.vue` | `consumer:user:profile` | 个人信息管理 |
| `/consumer/user/login-logs` | `user/login-logs.vue` | `consumer:user:logs` | 登录日志查询 |

**功能说明：**
- 个人信息：查看和编辑用户资料、上传头像、修改联系方式
- 登录日志：查看登录历史、IP地址、设备信息、风险评分

### 3. 财务管理模块 (`/consumer/finance`)

| 路由路径 | 组件 | 权限 | 说明 |
|---------|------|------|------|
| `/consumer/finance/account` | `finance/account.vue` | `consumer:finance:account` | 账户余额查询 |
| `/consumer/finance/recharge` | `finance/recharge.vue` | `consumer:finance:recharge` | 充值操作 |
| `/consumer/finance/withdraw` | `finance/withdraw.vue` | `consumer:finance:withdraw` | 提现申请 |
| `/consumer/finance/transactions` | `finance/transactions.vue` | `consumer:finance:transactions` | 财务流水查询 |

**功能说明：**
- 账户余额：显示可用余额、冻结余额
- 充值：支持微信支付、支付宝
- 提现：申请提现、查看审核状态
- 财务流水：查询交易记录、导出流水

### 4. 媒体管理模块 (`/consumer/media`)

| 路由路径 | 组件 | 权限 | 说明 |
|---------|------|------|------|
| `/consumer/media/upload` | `media/upload.vue` | `consumer:media:upload` | 文件上传 |
| `/consumer/media/list` | `media/list.vue` | `consumer:media:list` | 文件列表管理 |

**功能说明：**
- 文件上传：支持图片（JPEG/PNG/GIF）和视频（MP4/AVI/MOV）
- 文件列表：查看、预览、删除媒体文件

### 5. 物流管理模块 (`/consumer/logistics`)

| 路由路径 | 组件 | 权限 | 说明 |
|---------|------|------|------|
| `/consumer/logistics/query` | `logistics/query.vue` | `consumer:logistics:query` | 物流查询 |
| `/consumer/logistics/tracking` | `logistics/tracking.vue` | `consumer:logistics:tracking` | 物流轨迹 |

**功能说明：**
- 物流查询：输入运单号查询物流信息
- 物流轨迹：查看详细的物流轨迹时间线

### 6. 运费计算模块 (`/consumer/freight`)

| 路由路径 | 组件 | 权限 | 说明 |
|---------|------|------|------|
| `/consumer/freight/calculator` | `freight/calculator.vue` | `consumer:freight:calculate` | 运费计算器 |
| `/consumer/freight/templates` | `freight/templates.vue` | `consumer:freight:templates` | 运费模板管理 |

**功能说明：**
- 运费计算器：根据重量、距离计算运费
- 运费模板：创建、编辑、查询运费模板

## 路由守卫

### 认证守卫

所有路由（除了注册和登录页面）都需要用户登录后才能访问。路由守卫在 `router/guard.ts` 中配置。

**守卫逻辑：**
1. 检查 `accessToken` 是否存在
2. 如果不存在且路由没有 `ignoreAccess: true`，重定向到登录页
3. 如果存在，检查用户权限
4. 根据权限生成可访问的路由和菜单

### 权限控制

每个路由都配置了 `authority` 字段，用于权限控制：

```typescript
meta: {
  authority: ['consumer:user:profile'], // 需要的权限代码
}
```

**权限代码格式：**
- `consumer:view` - 查看 C 端用户模块
- `consumer:auth:register` - 用户注册权限
- `consumer:user:profile` - 个人信息管理权限
- `consumer:finance:account` - 账户余额查询权限
- 等等...

## 菜单显示

路由配置中的 `meta` 字段控制菜单显示：

```typescript
meta: {
  icon: 'lucide:users',        // 菜单图标
  order: 10,                   // 菜单排序
  title: 'C端用户管理',        // 菜单标题
  authority: ['consumer:view'], // 权限控制
  hideInMenu: false,           // 是否在菜单中隐藏
}
```

## 图标使用

所有图标使用 Lucide Icons：

| 模块 | 图标 | 说明 |
|------|------|------|
| C端用户管理 | `lucide:users` | 用户群组 |
| 用户认证 | `lucide:shield-check` | 安全认证 |
| 用户注册 | `lucide:user-plus` | 添加用户 |
| 用户登录 | `lucide:log-in` | 登录 |
| 个人信息 | `lucide:user-circle` | 用户资料 |
| 登录日志 | `lucide:history` | 历史记录 |
| 财务管理 | `lucide:wallet` | 钱包 |
| 账户余额 | `lucide:credit-card` | 信用卡 |
| 充值 | `lucide:arrow-down-to-line` | 向下箭头 |
| 提现 | `lucide:arrow-up-from-line` | 向上箭头 |
| 财务流水 | `lucide:receipt` | 收据 |
| 媒体管理 | `lucide:image` | 图片 |
| 上传文件 | `lucide:upload` | 上传 |
| 文件列表 | `lucide:folder-open` | 打开文件夹 |
| 物流管理 | `lucide:truck` | 卡车 |
| 物流查询 | `lucide:search` | 搜索 |
| 物流轨迹 | `lucide:map-pin` | 地图标记 |
| 运费计算 | `lucide:calculator` | 计算器 |
| 运费模板 | `lucide:file-text` | 文件文本 |

## 路由懒加载

所有页面组件都使用懒加载方式导入：

```typescript
component: () => import('#/views/consumer/user/profile.vue')
```

**优点：**
- 减少初始加载时间
- 按需加载页面资源
- 提升应用性能

## 多语言支持

路由标题支持多语言（通过 `$t()` 函数）：

```typescript
title: $t('page.consumer.user.profile')
```

**注意：** 当前配置使用硬编码的中文标题，如需多语言支持，需要：
1. 在 `locales/langs/` 中添加翻译文件
2. 将硬编码标题替换为 `$t()` 函数调用

## 路由配置验证

### 验证清单

- [x] 所有页面组件路径正确
- [x] 权限代码配置完整
- [x] 图标配置正确
- [x] 路由守卫配置正确
- [x] 注册和登录页面设置 `ignoreAccess: true`
- [x] 所有路由使用懒加载
- [x] 菜单层级结构合理

### 测试步骤

1. **未登录访问测试**
   - 访问任意需要认证的路由，应重定向到登录页
   - 访问注册和登录页面，应正常显示

2. **登录后访问测试**
   - 登录后访问各个路由，应正常显示对应页面
   - 菜单应根据权限正确显示

3. **权限控制测试**
   - 使用不同权限的用户登录
   - 验证菜单和路由是否根据权限正确显示/隐藏

## 相关文件

- 路由配置：`frontend/apps/admin/src/router/routes/modules/consumer.ts`
- 路由守卫：`frontend/apps/admin/src/router/guard.ts`
- 路由入口：`frontend/apps/admin/src/router/routes/index.ts`
- 页面组件：`frontend/apps/admin/src/views/consumer/`
- 状态管理：`frontend/apps/admin/src/stores/`

## 下一步

1. 配置多语言翻译文件
2. 测试所有路由的访问权限
3. 优化路由过渡动画
4. 添加面包屑导航
5. 配置路由元信息（SEO）
