import { computed, ref } from 'vue';
import { acceptHMRUpdate, defineStore } from 'pinia';

/**
 * 发送验证码请求接口
 */
export interface SendVerificationCodeRequest {
  phone: string;
  smsType?: 'VERIFICATION' | 'NOTIFICATION';
}

/**
 * 验证验证码请求接口
 */
export interface VerifyCodeRequest {
  phone: string;
  code: string;
}

/**
 * 验证码响应接口
 */
export interface VerifyCodeResponse {
  valid: boolean;
  message?: string;
}

/**
 * @zh_CN 短信服务状态管理
 */
export const useSMSStore = defineStore('sms', () => {
  // 状态
  const loading = ref(false);
  const error = ref<string | null>(null);
  const countdown = ref(0);
  const lastSentPhone = ref<string>('');
  const lastSentTime = ref<number>(0);

  // 计算属性
  const canSend = computed(() => countdown.value === 0);
  const countdownText = computed(() => {
    if (countdown.value > 0) {
      return `${countdown.value}秒后重试`;
    }
    return '发送验证码';
  });

  // 倒计时定时器
  let countdownTimer: ReturnType<typeof setInterval> | null = null;

  /**
   * 开始倒计时
   */
  function startCountdown(seconds: number = 60) {
    countdown.value = seconds;
    
    // 清除旧的定时器
    if (countdownTimer) {
      clearInterval(countdownTimer);
    }
    
    // 启动新的定时器
    countdownTimer = setInterval(() => {
      countdown.value--;
      if (countdown.value <= 0) {
        stopCountdown();
      }
    }, 1000);
  }

  /**
   * 停止倒计时
   */
  function stopCountdown() {
    countdown.value = 0;
    if (countdownTimer) {
      clearInterval(countdownTimer);
      countdownTimer = null;
    }
  }

  /**
   * 发送验证码
   */
  async function sendVerificationCode(request: SendVerificationCodeRequest): Promise<void> {
    // 检查是否在倒计时中
    if (!canSend.value) {
      throw new Error(`请等待 ${countdown.value} 秒后再试`);
    }

    // 检查频率限制（每分钟最多1次）
    const now = Date.now();
    if (lastSentPhone.value === request.phone && now - lastSentTime.value < 60000) {
      throw new Error('发送过于频繁，请稍后再试');
    }

    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC SMSService.SendVerificationCode
      // await smsServiceClient.sendVerificationCode(request);
      
      // 模拟发送成功
      console.log('发送验证码到:', request.phone);
      
      // 记录发送信息
      lastSentPhone.value = request.phone;
      lastSentTime.value = now;
      
      // 开始倒计时
      startCountdown(60);
    } catch (err: any) {
      error.value = err.message || '发送验证码失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 验证验证码
   */
  async function verifyCode(request: VerifyCodeRequest): Promise<VerifyCodeResponse> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC SMSService.VerifyCode
      // const response = await smsServiceClient.verifyCode(request);
      
      // 模拟验证
      const response: VerifyCodeResponse = {
        valid: request.code.length === 6,
        message: request.code.length === 6 ? '验证成功' : '验证码错误',
      };
      
      return response;
    } catch (err: any) {
      error.value = err.message || '验证失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 重置状态
   */
  function $reset() {
    stopCountdown();
    loading.value = false;
    error.value = null;
    lastSentPhone.value = '';
    lastSentTime.value = 0;
  }

  return {
    // 状态
    loading,
    error,
    countdown,
    lastSentPhone,
    lastSentTime,
    
    // 计算属性
    canSend,
    countdownText,
    
    // 方法
    startCountdown,
    stopCountdown,
    sendVerificationCode,
    verifyCode,
    $reset,
  };
});

// 解决热更新问题
const hot = import.meta.hot;
if (hot) {
  hot.accept(acceptHMRUpdate(useSMSStore, hot));
}
