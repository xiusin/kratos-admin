# Consumer 通用组件

本目录包含 C 端用户管理系统的可复用 Vue 3 组件。

## 组件列表

### 1. VerificationCodeInput.vue - 验证码输入组件

**功能特性：**
- 6 位数字输入（可配置长度）
- 自动聚焦和跳转到下一个输入框
- 支持键盘导航（左右箭头、退格键）
- 支持粘贴验证码
- 倒计时功能（默认 60 秒）
- 发送验证码按钮集成

**使用示例：**
```vue
<script setup>
import { ref } from 'vue';
import VerificationCodeInput from '@/components/consumer/VerificationCodeInput.vue';

const codeInputRef = ref();

function handleComplete(code) {
  console.log('验证码输入完成:', code);
}

async function handleSend() {
  // 调用发送验证码 API
  await smsStore.sendVerificationCode({ phone: '13800138000' });
}
</script>

<template>
  <VerificationCodeInput
    ref="codeInputRef"
    :length="6"
    :countdown="60"
    :sending="false"
    @complete="handleComplete"
    @send="handleSend"
  />
</template>
```

**Props：**
- `length` (number): 验证码长度，默认 6
- `countdown` (number): 倒计时秒数，默认 60
- `sending` (boolean): 是否正在发送验证码

**Events：**
- `complete`: 验证码输入完成时触发，参数为完整验证码字符串
- `send`: 点击发送验证码按钮时触发

**Methods：**
- `clear()`: 清空验证码
- `focus()`: 聚焦到第一个输入框
- `startCountdown()`: 手动开始倒计时
- `stopCountdown()`: 停止倒计时

---

### 2. PaymentMethodSelector.vue - 支付方式选择组件

**功能特性：**
- 支持微信支付和支付宝选择
- 支持多种支付类型（扫码、H5、APP）
- 集成支付二维码展示弹窗
- 可配置可用的支付方式和类型

**使用示例：**
```vue
<script setup>
import { ref } from 'vue';
import PaymentMethodSelector from '@/components/consumer/PaymentMethodSelector.vue';

const selectorRef = ref();

function handleChange(method, type) {
  console.log('支付方式变更:', method, type);
}

async function handleConfirm(method, type) {
  console.log('确认支付:', method, type);
  
  // 创建支付订单
  const result = await paymentStore.createPayment({
    paymentMethod: method,
    paymentType: type,
    amount: '100.00',
  });
  
  // 如果是扫码支付，显示二维码
  if (type === 'QRCODE' && result.paymentData?.qrCode) {
    selectorRef.value.showQRCode(result.paymentData.qrCode);
  }
}
</script>

<template>
  <PaymentMethodSelector
    ref="selectorRef"
    :available-methods="['WECHAT', 'ALIPAY']"
    :available-types="['QRCODE', 'H5']"
    default-method="WECHAT"
    default-type="QRCODE"
    :show-type-selector="true"
    @change="handleChange"
    @confirm="handleConfirm"
  />
</template>
```

**Props：**
- `availableMethods` (Array): 可用的支付方式，默认 `['WECHAT', 'ALIPAY']`
- `availableTypes` (Array): 可用的支付类型，默认 `['QRCODE', 'H5', 'APP']`
- `defaultMethod` (string): 默认选中的支付方式，默认 `'WECHAT'`
- `defaultType` (string): 默认选中的支付类型，默认 `'QRCODE'`
- `showTypeSelector` (boolean): 是否显示支付类型选择，默认 `true`

**Events：**
- `change`: 支付方式或类型变更时触发，参数为 `(method, type)`
- `confirm`: 点击确认支付按钮时触发，参数为 `(method, type)`

**Methods：**
- `showQRCode(url)`: 显示支付二维码弹窗
- `closeQRCode()`: 关闭支付二维码弹窗
- `getSelection()`: 获取当前选中的支付方式和类型

---

### 3. FileUploader.vue - 文件上传组件

**功能特性：**
- 支持拖拽上传
- 支持多文件上传
- 实时显示上传进度
- 文件格式验证（图片、视频）
- 文件大小验证（图片 5MB，视频 100MB）
- 文件列表展示和管理

**使用示例：**
```vue
<script setup>
import { ref } from 'vue';
import FileUploader from '@/components/consumer/FileUploader.vue';
import { useMediaStore } from '#/stores/media.state';

const mediaStore = useMediaStore();
const uploaderRef = ref();

async function handleUpload(file) {
  // 1. 生成预签名 URL
  const uploadUrlResult = await mediaStore.generateUploadURL({
    consumerId: 1,
    fileName: file.name,
    fileType: file.type.startsWith('image/') ? 'IMAGE' : 'VIDEO',
    fileSize: file.size,
  });
  
  // 2. 上传到 OSS
  await mediaStore.uploadFile(uploadUrlResult.uploadUrl, file, uploadUrlResult.fileKey);
  
  // 3. 确认上传
  const mediaFile = await mediaStore.confirmUpload({
    consumerId: 1,
    fileKey: uploadUrlResult.fileKey,
    fileName: file.name,
    fileType: file.type.startsWith('image/') ? 'IMAGE' : 'VIDEO',
    fileSize: file.size,
  });
  
  return {
    url: mediaFile.fileUrl,
    thumbnailUrl: mediaFile.thumbnailUrl,
  };
}

function handleRemove(file) {
  console.log('移除文件:', file);
}

function handleChange(files) {
  console.log('文件列表变更:', files);
}
</script>

<template>
  <FileUploader
    ref="uploaderRef"
    file-type="ALL"
    :max-size="100"
    :max-count="10"
    :multiple="true"
    :show-file-list="true"
    @upload="handleUpload"
    @remove="handleRemove"
    @change="handleChange"
  />
</template>
```

**Props：**
- `fileType` (string): 允许的文件类型，可选 `'IMAGE'`、`'VIDEO'`、`'ALL'`，默认 `'ALL'`
- `maxSize` (number): 最大文件大小（MB），默认 100
- `maxCount` (number): 最大文件数量，默认 10
- `accept` (string): 自定义允许的文件格式（覆盖 fileType）
- `multiple` (boolean): 是否支持多选，默认 `true`
- `showFileList` (boolean): 是否显示文件列表，默认 `true`

**Events：**
- `upload`: 上传文件时触发，参数为 File 对象，需要返回 Promise<{ url, thumbnailUrl }>
- `remove`: 移除文件时触发，参数为 UploadFile 对象
- `change`: 文件列表变更时触发，参数为 UploadFile 数组

**Methods：**
- `clear()`: 清空文件列表

---

### 4. AmountInput.vue - 金额输入组件

**功能特性：**
- 金额格式化（保留两位小数）
- 金额验证（最小值、最大值）
- 支持千分位分隔符
- 货币符号显示
- 实时错误提示

**使用示例：**
```vue
<script setup>
import { ref } from 'vue';
import AmountInput from '@/components/consumer/AmountInput.vue';

const amount = ref('');
const amountInputRef = ref();

function handleChange(value) {
  console.log('金额变更:', value);
}

function handleBlur() {
  console.log('失去焦点');
}
</script>

<template>
  <AmountInput
    ref="amountInputRef"
    v-model="amount"
    :min="0.01"
    :max="5000"
    placeholder="请输入充值金额"
    :disabled="false"
    :show-currency="true"
    currency="¥"
    :precision="2"
    :show-thousands-separator="true"
    @change="handleChange"
    @blur="handleBlur"
  />
</template>
```

**Props：**
- `modelValue` (string | number): 当前金额值
- `min` (number): 最小金额，默认 0.01
- `max` (number): 最大金额，默认 999999.99
- `placeholder` (string): 占位符，默认 `'请输入金额'`
- `disabled` (boolean): 是否禁用，默认 `false`
- `showCurrency` (boolean): 是否显示货币符号，默认 `true`
- `currency` (string): 货币符号，默认 `'¥'`
- `precision` (number): 小数位数，默认 2
- `showThousandsSeparator` (boolean): 是否显示千分位分隔符，默认 `false`

**Events：**
- `update:modelValue`: 金额值更新时触发
- `change`: 金额变更时触发，参数为格式化后的金额字符串
- `blur`: 失去焦点时触发

**Methods：**
- `focus()`: 聚焦输入框
- `clear()`: 清空输入

---

## 技术栈

- **Vue 3**: Composition API with `<script setup>`
- **TypeScript**: 完整的类型定义
- **Ant Design Vue**: 部分组件使用（Modal）
- **CSS**: Scoped styles，响应式设计

## 设计原则

1. **可复用性**: 所有组件都设计为高度可配置和可复用
2. **类型安全**: 使用 TypeScript 提供完整的类型定义
3. **用户体验**: 注重交互细节和视觉反馈
4. **可访问性**: 支持键盘导航和屏幕阅读器
5. **性能优化**: 使用 Vue 3 的响应式系统和组合式 API

## 注意事项

1. 所有组件都使用中文注释，便于团队协作
2. 组件暴露了必要的方法供父组件调用（通过 `defineExpose`）
3. 事件命名遵循 Vue 3 的约定（使用 camelCase）
4. 样式使用 scoped，避免全局污染
5. 组件内部状态管理使用 ref 和 computed

## 集成到项目

这些组件已经创建在 `frontend/apps/admin/src/components/consumer/` 目录下，可以直接在页面中导入使用：

```vue
<script setup>
import VerificationCodeInput from '@/components/consumer/VerificationCodeInput.vue';
import PaymentMethodSelector from '@/components/consumer/PaymentMethodSelector.vue';
import FileUploader from '@/components/consumer/FileUploader.vue';
import AmountInput from '@/components/consumer/AmountInput.vue';
</script>
```

## 相关需求

- **VerificationCodeInput**: Requirements 3.3
- **PaymentMethodSelector**: Requirements 4.1, 4.2
- **FileUploader**: Requirements 7.1-7.4
- **AmountInput**: Requirements 5.4, 5.12
