<script lang="ts" setup>
import { ref, computed } from 'vue';
import { Modal } from 'ant-design-vue';

/**
 * 支付方式选择组件
 * 支持微信支付和支付宝，显示支付二维码
 */

type PaymentMethod = 'WECHAT' | 'ALIPAY';
type PaymentType = 'APP' | 'H5' | 'MINI' | 'QRCODE';

interface Props {
  // 可用的支付方式
  availableMethods?: PaymentMethod[];
  // 可用的支付类型
  availableTypes?: PaymentType[];
  // 默认选中的支付方式
  defaultMethod?: PaymentMethod;
  // 默认选中的支付类型
  defaultType?: PaymentType;
  // 是否显示支付类型选择
  showTypeSelector?: boolean;
}

interface Emits {
  (e: 'change', method: PaymentMethod, type: PaymentType): void;
  (e: 'confirm', method: PaymentMethod, type: PaymentType): void;
}

const props = withDefaults(defineProps<Props>(), {
  availableMethods: () => ['WECHAT', 'ALIPAY'],
  availableTypes: () => ['QRCODE', 'H5', 'APP'],
  defaultMethod: 'WECHAT',
  defaultType: 'QRCODE',
  showTypeSelector: true,
});

const emit = defineEmits<Emits>();

// 当前选中的支付方式
const selectedMethod = ref<PaymentMethod>(props.defaultMethod);
// 当前选中的支付类型
const selectedType = ref<PaymentType>(props.defaultType);
// 二维码弹窗显示状态
const qrcodeModalVisible = ref(false);
// 二维码URL
const qrcodeUrl = ref('');

// 支付方式配置
const paymentMethods = computed(() => [
  {
    value: 'WECHAT' as PaymentMethod,
    label: '微信支付',
    icon: '💚',
    description: '使用微信扫码支付',
    available: props.availableMethods.includes('WECHAT'),
  },
  {
    value: 'ALIPAY' as PaymentMethod,
    label: '支付宝',
    icon: '💙',
    description: '使用支付宝扫码支付',
    available: props.availableMethods.includes('ALIPAY'),
  },
]);

// 支付类型配置
const paymentTypes = computed(() => [
  {
    value: 'QRCODE' as PaymentType,
    label: '扫码支付',
    description: '使用手机扫码完成支付',
    available: props.availableTypes.includes('QRCODE'),
  },
  {
    value: 'H5' as PaymentType,
    label: 'H5支付',
    description: '跳转到H5页面完成支付',
    available: props.availableTypes.includes('H5'),
  },
  {
    value: 'APP' as PaymentType,
    label: 'APP支付',
    description: '在APP中完成支付',
    available: props.availableTypes.includes('APP'),
  },
]);

// 选择支付方式
function selectMethod(method: PaymentMethod) {
  selectedMethod.value = method;
  emit('change', selectedMethod.value, selectedType.value);
}

// 选择支付类型
function selectType(type: PaymentType) {
  selectedType.value = type;
  emit('change', selectedMethod.value, selectedType.value);
}

// 确认支付
function handleConfirm() {
  emit('confirm', selectedMethod.value, selectedType.value);
}

// 显示二维码
function showQRCode(url: string) {
  qrcodeUrl.value = url;
  qrcodeModalVisible.value = true;
}

// 关闭二维码弹窗
function closeQRCode() {
  qrcodeModalVisible.value = false;
  qrcodeUrl.value = '';
}

// 获取当前选中的支付方式和类型
function getSelection() {
  return {
    method: selectedMethod.value,
    type: selectedType.value,
  };
}

// 暴露方法给父组件
defineExpose({
  showQRCode,
  closeQRCode,
  getSelection,
});
</script>

<template>
  <div class="payment-method-selector">
    <!-- 支付方式选择 -->
    <div class="section">
      <h3 class="section-title">选择支付方式</h3>
      <div class="payment-methods">
        <div
          v-for="method in paymentMethods"
          :key="method.value"
          class="payment-method"
          :class="{ 
            active: selectedMethod === method.value,
            disabled: !method.available 
          }"
          @click="method.available && selectMethod(method.value)"
        >
          <div class="method-icon">{{ method.icon }}</div>
          <div class="method-info">
            <div class="method-label">{{ method.label }}</div>
            <div class="method-description">{{ method.description }}</div>
          </div>
          <div v-if="selectedMethod === method.value" class="method-check">✓</div>
        </div>
      </div>
    </div>

    <!-- 支付类型选择 -->
    <div v-if="showTypeSelector" class="section">
      <h3 class="section-title">选择支付类型</h3>
      <div class="payment-types">
        <div
          v-for="type in paymentTypes"
          :key="type.value"
          class="payment-type"
          :class="{ 
            active: selectedType === type.value,
            disabled: !type.available 
          }"
          @click="type.available && selectType(type.value)"
        >
          <div class="type-label">{{ type.label }}</div>
          <div class="type-description">{{ type.description }}</div>
          <div v-if="selectedType === type.value" class="type-check">✓</div>
        </div>
      </div>
    </div>

    <!-- 确认按钮 -->
    <div class="confirm-button-wrapper">
      <button type="button" class="confirm-button" @click="handleConfirm">
        确认支付
      </button>
    </div>

    <!-- 二维码弹窗 -->
    <Modal
      v-model:open="qrcodeModalVisible"
      title="扫码支付"
      :footer="null"
      @cancel="closeQRCode"
    >
      <div class="qrcode-modal">
        <div v-if="qrcodeUrl" class="qrcode-wrapper">
          <img :src="qrcodeUrl" alt="支付二维码" class="qrcode-image" />
        </div>
        <div class="qrcode-tips">
          <p class="tip-title">请使用{{ selectedMethod === 'WECHAT' ? '微信' : '支付宝' }}扫码支付</p>
          <p class="tip-description">支付完成后会自动跳转</p>
        </div>
      </div>
    </Modal>
  </div>
</template>

<style scoped>
.payment-method-selector {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.section-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #262626;
}

/* 支付方式样式 */
.payment-methods {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.payment-method {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border: 2px solid #d9d9d9;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.payment-method:hover:not(.disabled) {
  border-color: #1890ff;
  background: #f0f8ff;
}

.payment-method.active {
  border-color: #1890ff;
  background: #e6f7ff;
}

.payment-method.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.method-icon {
  font-size: 32px;
}

.method-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.method-label {
  font-size: 16px;
  font-weight: 600;
  color: #262626;
}

.method-description {
  font-size: 14px;
  color: #8c8c8c;
}

.method-check {
  font-size: 24px;
  color: #1890ff;
  font-weight: bold;
}

/* 支付类型样式 */
.payment-types {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}

.payment-type {
  position: relative;
  padding: 16px;
  border: 2px solid #d9d9d9;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.payment-type:hover:not(.disabled) {
  border-color: #1890ff;
  background: #f0f8ff;
}

.payment-type.active {
  border-color: #1890ff;
  background: #e6f7ff;
}

.payment-type.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.type-label {
  font-size: 14px;
  font-weight: 600;
  color: #262626;
  margin-bottom: 4px;
}

.type-description {
  font-size: 12px;
  color: #8c8c8c;
}

.type-check {
  position: absolute;
  top: 8px;
  right: 8px;
  font-size: 16px;
  color: #1890ff;
  font-weight: bold;
}

/* 确认按钮 */
.confirm-button-wrapper {
  display: flex;
  justify-content: center;
  margin-top: 8px;
}

.confirm-button {
  padding: 12px 48px;
  font-size: 16px;
  font-weight: 600;
  color: #fff;
  background: #1890ff;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.confirm-button:hover {
  background: #40a9ff;
}

.confirm-button:active {
  background: #096dd9;
}

/* 二维码弹窗 */
.qrcode-modal {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
  padding: 24px 0;
}

.qrcode-wrapper {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 16px;
  background: #fff;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
}

.qrcode-image {
  width: 256px;
  height: 256px;
  display: block;
}

.qrcode-tips {
  text-align: center;
}

.tip-title {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
  color: #262626;
}

.tip-description {
  margin: 0;
  font-size: 14px;
  color: #8c8c8c;
}
</style>
