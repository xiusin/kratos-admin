import { computed, ref } from 'vue';
import { acceptHMRUpdate, defineStore } from 'pinia';

/**
 * Consumer 用户信息接口
 */
export interface ConsumerInfo {
  id?: number;
  tenantId?: number;
  phone?: string;
  email?: string;
  nickname?: string;
  avatar?: string;
  wechatOpenid?: string;
  wechatUnionid?: string;
  status?: 'NORMAL' | 'LOCKED' | 'DEACTIVATED';
  riskScore?: number;
  lockedUntil?: string;
  lastLoginAt?: string;
  lastLoginIp?: string;
  deactivatedAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

/**
 * 登录响应接口
 */
export interface LoginResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  consumer: ConsumerInfo;
}

/**
 * 注册请求接口
 */
export interface RegisterByPhoneRequest {
  phone: string;
  password: string;
  verificationCode: string;
  nickname?: string;
}

/**
 * 登录请求接口
 */
export interface LoginByPhoneRequest {
  phone: string;
  password: string;
}

/**
 * 微信登录请求接口
 */
export interface LoginByWechatRequest {
  code: string;
}

/**
 * 更新用户信息请求接口
 */
export interface UpdateConsumerRequest {
  id: number;
  nickname?: string;
  email?: string;
  avatar?: string;
}

/**
 * @zh_CN C端用户状态管理
 */
export const useConsumerStore = defineStore('consumer', () => {
  // 状态
  const consumerInfo = ref<ConsumerInfo | null>(null);
  const accessToken = ref<string>('');
  const refreshToken = ref<string>('');
  const loading = ref(false);
  const error = ref<string | null>(null);

  // 计算属性
  const isLoggedIn = computed(() => !!accessToken.value && !!consumerInfo.value);
  const isLocked = computed(() => consumerInfo.value?.status === 'LOCKED');
  const isDeactivated = computed(() => consumerInfo.value?.status === 'DEACTIVATED');
  const tenantId = computed(() => consumerInfo.value?.tenantId ?? null);

  /**
   * 设置用户信息
   */
  function setConsumerInfo(info: ConsumerInfo | null) {
    consumerInfo.value = info;
  }

  /**
   * 设置令牌
   */
  function setTokens(access: string, refresh: string) {
    accessToken.value = access;
    refreshToken.value = refresh;
    
    // 保存到 localStorage
    if (access) {
      localStorage.setItem('consumer_access_token', access);
    }
    if (refresh) {
      localStorage.setItem('consumer_refresh_token', refresh);
    }
  }

  /**
   * 从 localStorage 恢复令牌
   */
  function restoreTokens() {
    const access = localStorage.getItem('consumer_access_token');
    const refresh = localStorage.getItem('consumer_refresh_token');
    if (access) accessToken.value = access;
    if (refresh) refreshToken.value = refresh;
  }

  /**
   * 手机号注册
   */
  async function registerByPhone(request: RegisterByPhoneRequest): Promise<ConsumerInfo> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC ConsumerService.RegisterByPhone
      // const response = await consumerServiceClient.registerByPhone(request);
      
      // 模拟响应
      const consumer: ConsumerInfo = {
        id: 1,
        phone: request.phone,
        nickname: request.nickname,
        status: 'NORMAL',
      };
      
      setConsumerInfo(consumer);
      return consumer;
    } catch (err: any) {
      error.value = err.message || '注册失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 手机号登录
   */
  async function loginByPhone(request: LoginByPhoneRequest): Promise<LoginResponse> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC ConsumerService.LoginByPhone
      // const response = await consumerServiceClient.loginByPhone(request);
      
      // 模拟响应
      const response: LoginResponse = {
        accessToken: 'mock_access_token',
        refreshToken: 'mock_refresh_token',
        expiresIn: 7200,
        consumer: {
          id: 1,
          phone: request.phone,
          status: 'NORMAL',
        },
      };
      
      setTokens(response.accessToken, response.refreshToken);
      setConsumerInfo(response.consumer);
      
      return response;
    } catch (err: any) {
      error.value = err.message || '登录失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 微信登录
   */
  async function loginByWechat(request: LoginByWechatRequest): Promise<LoginResponse> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC ConsumerService.LoginByWechat
      // const response = await consumerServiceClient.loginByWechat(request);
      
      // 模拟响应
      const response: LoginResponse = {
        accessToken: 'mock_access_token',
        refreshToken: 'mock_refresh_token',
        expiresIn: 7200,
        consumer: {
          id: 1,
          wechatOpenid: 'mock_openid',
          status: 'NORMAL',
        },
      };
      
      setTokens(response.accessToken, response.refreshToken);
      setConsumerInfo(response.consumer);
      
      return response;
    } catch (err: any) {
      error.value = err.message || '微信登录失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取用户信息
   */
  async function getConsumer(id: number): Promise<ConsumerInfo> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC ConsumerService.GetConsumer
      // const response = await consumerServiceClient.getConsumer({ id });
      
      // 模拟响应
      const consumer: ConsumerInfo = {
        id,
        phone: '138****8888',
        nickname: '用户' + id,
        status: 'NORMAL',
      };
      
      setConsumerInfo(consumer);
      return consumer;
    } catch (err: any) {
      error.value = err.message || '获取用户信息失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 更新用户信息
   */
  async function updateConsumer(request: UpdateConsumerRequest): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC ConsumerService.UpdateConsumer
      // await consumerServiceClient.updateConsumer(request);
      
      // 更新本地状态
      if (consumerInfo.value && consumerInfo.value.id === request.id) {
        consumerInfo.value = {
          ...consumerInfo.value,
          ...request,
        };
      }
    } catch (err: any) {
      error.value = err.message || '更新用户信息失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 登出
   */
  function logout() {
    consumerInfo.value = null;
    accessToken.value = '';
    refreshToken.value = '';
    localStorage.removeItem('consumer_access_token');
    localStorage.removeItem('consumer_refresh_token');
  }

  /**
   * 重置状态
   */
  function $reset() {
    logout();
    error.value = null;
    loading.value = false;
  }

  return {
    // 状态
    consumerInfo,
    accessToken,
    refreshToken,
    loading,
    error,
    
    // 计算属性
    isLoggedIn,
    isLocked,
    isDeactivated,
    tenantId,
    
    // 方法
    setConsumerInfo,
    setTokens,
    restoreTokens,
    registerByPhone,
    loginByPhone,
    loginByWechat,
    getConsumer,
    updateConsumer,
    logout,
    $reset,
  };
});

// 解决热更新问题
const hot = import.meta.hot;
if (hot) {
  hot.accept(acceptHMRUpdate(useConsumerStore, hot));
}
