<script lang="ts" setup>
import { ref, watch, nextTick } from 'vue';

/**
 * 验证码输入组件
 * 支持6位数字输入，自动聚焦和跳转
 */

interface Props {
  // 验证码长度，默认6位
  length?: number;
  // 倒计时秒数，默认60秒
  countdown?: number;
  // 是否正在发送验证码
  sending?: boolean;
}

interface Emits {
  (e: 'complete', code: string): void;
  (e: 'send'): void;
}

const props = withDefaults(defineProps<Props>(), {
  length: 6,
  countdown: 60,
  sending: false,
});

const emit = defineEmits<Emits>();

// 验证码输入值数组
const codeValues = ref<string[]>(Array(props.length).fill(''));
// 输入框引用数组
const inputRefs = ref<HTMLInputElement[]>([]);
// 倒计时剩余秒数
const remainingSeconds = ref(0);
// 倒计时定时器
let countdownTimer: ReturnType<typeof setInterval> | null = null;

// 设置输入框引用
function setInputRef(el: any, index: number) {
  if (el) {
    inputRefs.value[index] = el;
  }
}

// 处理输入
function handleInput(index: number, event: Event) {
  const target = event.target as HTMLInputElement;
  const value = target.value;
  
  // 只允许输入数字
  const numericValue = value.replace(/\D/g, '');
  
  if (numericValue.length > 0) {
    // 取第一个数字
    codeValues.value[index] = numericValue[0];
    
    // 自动跳转到下一个输入框
    if (index < props.length - 1) {
      nextTick(() => {
        inputRefs.value[index + 1]?.focus();
      });
    }
    
    // 检查是否完成输入
    checkComplete();
  } else {
    codeValues.value[index] = '';
  }
}

// 处理键盘事件
function handleKeydown(index: number, event: KeyboardEvent) {
  // 退格键：删除当前值并跳转到上一个输入框
  if (event.key === 'Backspace') {
    if (codeValues.value[index] === '' && index > 0) {
      event.preventDefault();
      codeValues.value[index - 1] = '';
      nextTick(() => {
        inputRefs.value[index - 1]?.focus();
      });
    } else {
      codeValues.value[index] = '';
    }
  }
  
  // 左箭头：跳转到上一个输入框
  if (event.key === 'ArrowLeft' && index > 0) {
    event.preventDefault();
    inputRefs.value[index - 1]?.focus();
  }
  
  // 右箭头：跳转到下一个输入框
  if (event.key === 'ArrowRight' && index < props.length - 1) {
    event.preventDefault();
    inputRefs.value[index + 1]?.focus();
  }
}

// 处理粘贴
function handlePaste(event: ClipboardEvent) {
  event.preventDefault();
  const pastedText = event.clipboardData?.getData('text') || '';
  const numericText = pastedText.replace(/\D/g, '');
  
  // 填充验证码
  for (let i = 0; i < Math.min(numericText.length, props.length); i++) {
    codeValues.value[i] = numericText[i];
  }
  
  // 聚焦到最后一个已填充的输入框
  const lastFilledIndex = Math.min(numericText.length, props.length) - 1;
  nextTick(() => {
    inputRefs.value[lastFilledIndex]?.focus();
  });
  
  checkComplete();
}

// 检查是否完成输入
function checkComplete() {
  const code = codeValues.value.join('');
  if (code.length === props.length) {
    emit('complete', code);
  }
}

// 发送验证码
function handleSend() {
  if (remainingSeconds.value > 0 || props.sending) {
    return;
  }
  
  emit('send');
  startCountdown();
}

// 开始倒计时
function startCountdown() {
  remainingSeconds.value = props.countdown;
  
  if (countdownTimer) {
    clearInterval(countdownTimer);
  }
  
  countdownTimer = setInterval(() => {
    remainingSeconds.value--;
    if (remainingSeconds.value <= 0) {
      stopCountdown();
    }
  }, 1000);
}

// 停止倒计时
function stopCountdown() {
  if (countdownTimer) {
    clearInterval(countdownTimer);
    countdownTimer = null;
  }
  remainingSeconds.value = 0;
}

// 清空验证码
function clear() {
  codeValues.value = Array(props.length).fill('');
  nextTick(() => {
    inputRefs.value[0]?.focus();
  });
}

// 聚焦到第一个输入框
function focus() {
  nextTick(() => {
    inputRefs.value[0]?.focus();
  });
}

// 监听发送状态变化
watch(() => props.sending, (newVal) => {
  if (!newVal && remainingSeconds.value === 0) {
    // 发送完成，开始倒计时
    startCountdown();
  }
});

// 暴露方法给父组件
defineExpose({
  clear,
  focus,
  startCountdown,
  stopCountdown,
});
</script>

<template>
  <div class="verification-code-input">
    <div class="code-inputs">
      <input
        v-for="(value, index) in codeValues"
        :key="index"
        :ref="(el) => setInputRef(el, index)"
        v-model="codeValues[index]"
        type="text"
        inputmode="numeric"
        maxlength="1"
        class="code-input"
        @input="handleInput(index, $event)"
        @keydown="handleKeydown(index, $event)"
        @paste="handlePaste"
      />
    </div>
    
    <div class="send-button-wrapper">
      <button
        type="button"
        class="send-button"
        :disabled="remainingSeconds > 0 || sending"
        @click="handleSend"
      >
        <span v-if="sending">发送中...</span>
        <span v-else-if="remainingSeconds > 0">{{ remainingSeconds }}秒后重试</span>
        <span v-else>发送验证码</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.verification-code-input {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.code-inputs {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.code-input {
  width: 48px;
  height: 56px;
  text-align: center;
  font-size: 24px;
  font-weight: 600;
  border: 2px solid #d9d9d9;
  border-radius: 8px;
  transition: all 0.2s;
  outline: none;
}

.code-input:focus {
  border-color: #1890ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

.code-input:not(:placeholder-shown) {
  border-color: #52c41a;
}

.send-button-wrapper {
  display: flex;
  justify-content: center;
}

.send-button {
  padding: 8px 24px;
  font-size: 14px;
  color: #1890ff;
  background: transparent;
  border: 1px solid #1890ff;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.send-button:hover:not(:disabled) {
  color: #fff;
  background: #1890ff;
}

.send-button:disabled {
  color: rgba(0, 0, 0, 0.25);
  background: #f5f5f5;
  border-color: #d9d9d9;
  cursor: not-allowed;
}
</style>
