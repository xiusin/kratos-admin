<script lang="ts" setup>
import { ref, computed, watch } from 'vue';

/**
 * 金额输入组件
 * 支持金额格式化（保留两位小数）和金额验证（最小值、最大值）
 */

interface Props {
  // 当前金额值
  modelValue?: string | number;
  // 最小金额
  min?: number;
  // 最大金额
  max?: number;
  // 占位符
  placeholder?: string;
  // 是否禁用
  disabled?: boolean;
  // 是否显示货币符号
  showCurrency?: boolean;
  // 货币符号
  currency?: string;
  // 小数位数
  precision?: number;
  // 是否显示千分位分隔符
  showThousandsSeparator?: boolean;
}

interface Emits {
  (e: 'update:modelValue', value: string): void;
  (e: 'change', value: string): void;
  (e: 'blur'): void;
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  min: 0.01,
  max: 999999.99,
  placeholder: '请输入金额',
  disabled: false,
  showCurrency: true,
  currency: '¥',
  precision: 2,
  showThousandsSeparator: false,
});

const emit = defineEmits<Emits>();

// 输入框的显示值
const displayValue = ref('');
// 输入框引用
const inputRef = ref<HTMLInputElement>();
// 是否聚焦
const isFocused = ref(false);
// 错误信息
const errorMessage = ref('');

// 格式化金额（添加千分位分隔符）
function formatAmount(value: string): string {
  if (!value || value === '0' || value === '0.00') return '';
  
  const parts = value.split('.');
  const integerPart = parts[0];
  const decimalPart = parts[1] || '';
  
  // 添加千分位分隔符
  const formattedInteger = props.showThousandsSeparator
    ? integerPart.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
    : integerPart;
  
  return decimalPart ? `${formattedInteger}.${decimalPart}` : formattedInteger;
}

// 解析金额（移除千分位分隔符）
function parseAmount(value: string): string {
  return value.replace(/,/g, '');
}

// 验证金额
function validateAmount(value: string): { valid: boolean; error?: string } {
  if (!value) {
    return { valid: true };
  }
  
  const numValue = parseFloat(value);
  
  if (isNaN(numValue)) {
    return { valid: false, error: '请输入有效的金额' };
  }
  
  if (numValue < props.min) {
    return { valid: false, error: `金额不能小于 ${props.min}` };
  }
  
  if (numValue > props.max) {
    return { valid: false, error: `金额不能大于 ${props.max}` };
  }
  
  return { valid: true };
}

// 格式化为固定小数位
function toFixed(value: string, precision: number): string {
  const numValue = parseFloat(value);
  if (isNaN(numValue)) return '';
  return numValue.toFixed(precision);
}

// 处理输入
function handleInput(event: Event) {
  const target = event.target as HTMLInputElement;
  let value = target.value;
  
  // 移除非数字和小数点的字符
  value = value.replace(/[^\d.]/g, '');
  
  // 只允许一个小数点
  const parts = value.split('.');
  if (parts.length > 2) {
    value = parts[0] + '.' + parts.slice(1).join('');
  }
  
  // 限制小数位数
  if (parts.length === 2 && parts[1].length > props.precision) {
    value = parts[0] + '.' + parts[1].substring(0, props.precision);
  }
  
  displayValue.value = value;
  
  // 清除错误信息
  errorMessage.value = '';
}

// 处理聚焦
function handleFocus() {
  isFocused.value = true;
  
  // 聚焦时显示原始值（移除格式化）
  if (displayValue.value) {
    displayValue.value = parseAmount(displayValue.value);
  }
}

// 处理失焦
function handleBlur() {
  isFocused.value = false;
  
  let value = displayValue.value;
  
  if (value) {
    // 移除千分位分隔符
    value = parseAmount(value);
    
    // 格式化为固定小数位
    value = toFixed(value, props.precision);
    
    // 验证金额
    const validation = validateAmount(value);
    if (!validation.valid) {
      errorMessage.value = validation.error || '';
      emit('update:modelValue', '');
      emit('change', '');
    } else {
      errorMessage.value = '';
      
      // 格式化显示值
      displayValue.value = formatAmount(value);
      
      emit('update:modelValue', value);
      emit('change', value);
    }
  } else {
    emit('update:modelValue', '');
    emit('change', '');
  }
  
  emit('blur');
}

// 聚焦输入框
function focus() {
  inputRef.value?.focus();
}

// 清空输入
function clear() {
  displayValue.value = '';
  errorMessage.value = '';
  emit('update:modelValue', '');
  emit('change', '');
}

// 监听外部值变化
watch(() => props.modelValue, (newValue) => {
  if (newValue !== undefined && newValue !== null && newValue !== '') {
    const strValue = String(newValue);
    const formattedValue = toFixed(strValue, props.precision);
    
    if (!isFocused.value) {
      displayValue.value = formatAmount(formattedValue);
    } else {
      displayValue.value = formattedValue;
    }
  } else {
    displayValue.value = '';
  }
}, { immediate: true });

// 暴露方法给父组件
defineExpose({
  focus,
  clear,
});
</script>

<template>
  <div class="amount-input" :class="{ error: errorMessage, disabled }">
    <div class="input-wrapper">
      <span v-if="showCurrency" class="currency-symbol">{{ currency }}</span>
      <input
        ref="inputRef"
        v-model="displayValue"
        type="text"
        inputmode="decimal"
        class="amount-field"
        :placeholder="placeholder"
        :disabled="disabled"
        @input="handleInput"
        @focus="handleFocus"
        @blur="handleBlur"
      />
    </div>
    
    <div v-if="errorMessage" class="error-message">
      {{ errorMessage }}
    </div>
    
    <div v-if="!errorMessage && (min || max)" class="hint-message">
      金额范围：{{ currency }}{{ min }} - {{ currency }}{{ max }}
    </div>
  </div>
</template>

<style scoped>
.amount-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  border: 2px solid #d9d9d9;
  border-radius: 8px;
  background: #fff;
  transition: all 0.2s;
}

.input-wrapper:focus-within {
  border-color: #1890ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

.amount-input.error .input-wrapper {
  border-color: #ff4d4f;
}

.amount-input.error .input-wrapper:focus-within {
  box-shadow: 0 0 0 2px rgba(255, 77, 79, 0.2);
}

.amount-input.disabled .input-wrapper {
  background: #f5f5f5;
  cursor: not-allowed;
}

.currency-symbol {
  padding: 0 12px;
  font-size: 16px;
  font-weight: 600;
  color: #8c8c8c;
  border-right: 1px solid #d9d9d9;
}

.amount-field {
  flex: 1;
  padding: 12px 16px;
  font-size: 16px;
  font-weight: 600;
  color: #262626;
  background: transparent;
  border: none;
  outline: none;
}

.amount-field::placeholder {
  color: #bfbfbf;
  font-weight: normal;
}

.amount-field:disabled {
  cursor: not-allowed;
}

.error-message {
  font-size: 14px;
  color: #ff4d4f;
}

.hint-message {
  font-size: 12px;
  color: #8c8c8c;
}
</style>
