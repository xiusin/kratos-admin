# C端用户管理系统 Store 使用指南

本目录包含C端用户管理系统的所有状态管理store，基于Vue 3 + Pinia实现。

## 📦 已实现的Store模块

### 1. Consumer Store (用户服务)
**文件**: `consumer.ts`

**功能**:
- 用户注册（手机号）
- 用户登录（手机号、微信）
- 用户信息管理
- JWT令牌管理
- 登录状态维护

**使用示例**:
```typescript
import { useConsumerStore } from '@vben/stores';

const consumerStore = useConsumerStore();

// 手机号注册
await consumerStore.registerByPhone({
  phone: '13800138000',
  password: 'password123',
  verificationCode: '123456',
  nickname: '用户昵称',
});

// 手机号登录
await consumerStore.loginByPhone({
  phone: '13800138000',
  password: 'password123',
});

// 微信登录
await consumerStore.loginByWechat({
  code: 'wechat_auth_code',
});

// 获取用户信息
await consumerStore.getConsumer(userId);

// 更新用户信息
await consumerStore.updateConsumer({
  id: userId,
  nickname: '新昵称',
  email: 'user@example.com',
});

// 登出
consumerStore.logout();
```

### 2. SMS Store (短信服务)
**文件**: `sms.ts`

**功能**:
- 发送验证码
- 验证验证码
- 验证码倒计时
- 频率限制

**使用示例**:
```typescript
import { useSMSStore } from '@vben/stores';

const smsStore = useSMSStore();

// 发送验证码
await smsStore.sendVerificationCode({
  phone: '13800138000',
  smsType: 'VERIFICATION',
});

// 验证验证码
const result = await smsStore.verifyCode({
  phone: '13800138000',
  code: '123456',
});

// 获取倒计时文本
console.log(smsStore.countdownText); // "60秒后重试" 或 "发送验证码"

// 检查是否可以发送
console.log(smsStore.canSend); // true/false
```

### 3. Payment Store (支付服务)
**文件**: `payment.ts`

**功能**:
- 创建支付订单
- 查询支付状态
- 支付状态轮询
- 申请退款

**使用示例**:
```typescript
import { usePaymentStore } from '@vben/stores';

const paymentStore = usePaymentStore();

// 创建支付订单
const { order, paymentData } = await paymentStore.createPayment({
  consumerId: 1,
  paymentMethod: 'WECHAT',
  paymentType: 'QRCODE',
  amount: '100.00',
  description: '商品购买',
});

// 开始轮询支付状态（每3秒查询一次，最多60次）
paymentStore.startPolling(order.orderNo!, 3000, 60);

// 手动查询支付状态
const status = await paymentStore.queryPaymentStatus(order.orderNo!);

// 停止轮询
paymentStore.stopPolling();

// 申请退款
const refundResult = await paymentStore.refund({
  orderNo: order.orderNo!,
  refundAmount: '50.00',
  reason: '不想要了',
});
```

### 4. Finance Store (财务服务)
**文件**: `finance.ts`

**功能**:
- 账户余额管理
- 充值、提现
- 财务流水查询
- 流水导出

**使用示例**:
```typescript
import { useFinanceStore } from '@vben/stores';

const financeStore = useFinanceStore();

// 获取账户余额
await financeStore.getAccount(consumerId);

// 查看可用余额
console.log(financeStore.availableBalance); // "1000.00"
console.log(financeStore.frozenBalance); // "0.00"

// 充值
await financeStore.recharge({
  consumerId: 1,
  amount: '100.00',
  paymentOrderNo: 'PAY123456',
});

// 申请提现
const withdrawResult = await financeStore.withdraw({
  consumerId: 1,
  amount: '50.00',
  bankAccount: '6222021234567890',
  bankName: '中国工商银行',
});

// 查询财务流水
await financeStore.listTransactions({
  consumerId: 1,
  transactionType: 'RECHARGE',
  startDate: '2024-01-01',
  endDate: '2024-12-31',
  page: 1,
  pageSize: 20,
});

// 导出流水
const csvBlob = await financeStore.exportTransactions({
  consumerId: 1,
});

// 实时刷新余额
await financeStore.refreshBalance(consumerId);
```

### 5. Media Store (媒体服务)
**文件**: `media.ts`

**功能**:
- 文件上传（图片、视频）
- 生成预签名URL
- 上传进度跟踪
- 文件管理

**使用示例**:
```typescript
import { useMediaStore } from '@vben/stores';

const mediaStore = useMediaStore();

// 生成上传URL
const { uploadUrl, fileKey, expiresIn } = await mediaStore.generateUploadURL({
  consumerId: 1,
  fileName: 'photo.jpg',
  fileType: 'IMAGE',
  fileSize: 1024000,
});

// 上传文件到OSS
await mediaStore.uploadFile(
  uploadUrl,
  file,
  fileKey,
  (progress) => {
    console.log(`上传进度: ${progress}%`);
  }
);

// 确认上传完成
const mediaFile = await mediaStore.confirmUpload({
  consumerId: 1,
  fileKey,
  fileName: 'photo.jpg',
  fileType: 'IMAGE',
  fileSize: 1024000,
});

// 查询媒体文件列表
await mediaStore.listMediaFiles(consumerId, 'IMAGE', 1, 20);

// 删除媒体文件
await mediaStore.deleteMediaFile(fileId);

// 获取上传进度
const progress = mediaStore.getUploadProgress(fileKey);
console.log(progress?.progress); // 0-100
```

### 6. Logistics Store (物流服务)
**文件**: `logistics.ts`

**功能**:
- 查询物流信息
- 订阅物流状态
- 物流信息轮询
- 物流历史查询

**使用示例**:
```typescript
import { useLogisticsStore } from '@vben/stores';

const logisticsStore = useLogisticsStore();

// 查询物流信息
const tracking = await logisticsStore.queryLogistics({
  trackingNo: 'SF1234567890',
  courierCompany: '顺丰速运',
});

// 订阅物流状态（自动开始轮询）
await logisticsStore.subscribeLogistics({
  trackingNo: 'SF1234567890',
  courierCompany: '顺丰速运',
  phone: '13800138000',
});

// 手动开始轮询（每5分钟查询一次）
logisticsStore.startPolling('SF1234567890', '顺丰速运', 300000);

// 停止轮询
logisticsStore.stopPolling('SF1234567890');

// 获取缓存的物流信息
const cached = logisticsStore.getCachedTracking('SF1234567890');

// 查询物流历史
const history = await logisticsStore.listLogisticsHistory(1, 20);

// 获取状态文本和颜色
console.log(logisticsStore.getStatusText('IN_TRANSIT')); // "运输中"
console.log(logisticsStore.getStatusColor('DELIVERED')); // "success"
```

### 7. Freight Store (运费计算服务)
**文件**: `freight.ts`

**功能**:
- 运费计算
- 运费模板管理
- 包邮规则判断

**使用示例**:
```typescript
import { useFreightStore } from '@vben/stores';

const freightStore = useFreightStore();

// 计算运费
const result = await freightStore.calculateFreight({
  templateId: 1,
  weight: 2.5,
  toProvince: '北京市',
  toCity: '北京市',
  toDistrict: '朝阳区',
  orderAmount: '150.00',
});

console.log(result.freight); // "15.00"
console.log(result.isFreeShipping); // true/false
console.log(result.calculationDetail); // "首重1kg 10元 + 续重2kg × 5元"

// 创建运费模板
const template = await freightStore.createFreightTemplate({
  name: '标准运费模板',
  calculationType: 'BY_WEIGHT',
  firstWeight: '1.0',
  firstPrice: '10.00',
  additionalWeight: '1.0',
  additionalPrice: '5.00',
  freeShippingRules: [
    {
      type: 'AMOUNT',
      minAmount: '99.00',
    },
  ],
});

// 查询运费模板列表
await freightStore.listFreightTemplates(1, 20);

// 更新运费模板
await freightStore.updateFreightTemplate(templateId, {
  name: '新模板名称',
  firstPrice: '12.00',
});

// 激活/停用模板
await freightStore.toggleTemplateStatus(templateId, true);

// 删除运费模板
await freightStore.deleteFreightTemplate(templateId);
```

## 🔧 通用功能

所有Store都包含以下通用功能：

### 加载状态
```typescript
const store = useXxxStore();
console.log(store.loading); // true/false
```

### 错误处理
```typescript
const store = useXxxStore();
console.log(store.error); // 错误信息或null
```

### 重置状态
```typescript
const store = useXxxStore();
store.$reset(); // 重置所有状态到初始值
```

## 📝 注意事项

1. **TODO标记**: 所有Store中的gRPC客户端调用都标记为TODO，需要在实际集成时替换为真实的API调用。

2. **模拟数据**: 当前所有Store都使用模拟数据进行响应，便于前端开发和测试。

3. **令牌管理**: Consumer Store会自动将JWT令牌保存到localStorage，并提供`restoreTokens()`方法用于页面刷新后恢复登录状态。

4. **轮询管理**: Payment Store和Logistics Store提供了轮询功能，记得在组件卸载时调用`stopPolling()`或`stopAllPolling()`清理定时器。

5. **上传进度**: Media Store使用Map存储上传进度，支持多文件并发上传。

6. **错误处理**: 所有异步方法都会抛出错误，建议使用try-catch包裹或在组件中统一处理。

## 🚀 下一步

1. 集成真实的gRPC客户端
2. 实现API请求拦截器（添加JWT令牌）
3. 实现统一的错误处理
4. 添加请求重试机制
5. 实现离线缓存策略

## 📚 相关文档

- [Pinia官方文档](https://pinia.vuejs.org/)
- [Vue 3官方文档](https://vuejs.org/)
- [C端用户管理系统设计文档](../../../.kiro/specs/c-user-management-system/design.md)
- [C端用户管理系统需求文档](../../../.kiro/specs/c-user-management-system/requirements.md)
