import { ref } from 'vue';
import { acceptHMRUpdate, defineStore } from 'pinia';

/**
 * 支付订单信息接口
 */
export interface PaymentOrder {
  id?: number;
  tenantId?: number;
  orderNo?: string;
  consumerId?: number;
  paymentMethod?: 'WECHAT' | 'ALIPAY' | 'YEEPAY';
  paymentType?: 'APP' | 'H5' | 'MINI' | 'QRCODE';
  amount?: string;
  status?: 'PENDING' | 'SUCCESS' | 'FAILED' | 'CLOSED' | 'REFUNDED';
  transactionId?: string;
  paidAt?: string;
  closedAt?: string;
  expiresAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

/**
 * 创建支付请求接口
 */
export interface CreatePaymentRequest {
  consumerId: number;
  paymentMethod: 'WECHAT' | 'ALIPAY' | 'YEEPAY';
  paymentType: 'APP' | 'H5' | 'MINI' | 'QRCODE';
  amount: string;
  description?: string;
}

/**
 * 创建支付响应接口
 */
export interface CreatePaymentResponse {
  order: PaymentOrder;
  paymentData?: any; // 支付参数（微信/支付宝返回的数据）
}

/**
 * 退款请求接口
 */
export interface RefundRequest {
  orderNo: string;
  refundAmount: string;
  reason?: string;
}

/**
 * 退款响应接口
 */
export interface RefundResponse {
  refundNo: string;
  status: 'PENDING' | 'SUCCESS' | 'FAILED';
}

/**
 * @zh_CN 支付服务状态管理
 */
export const usePaymentStore = defineStore('payment', () => {
  // 状态
  const currentOrder = ref<PaymentOrder | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const pollingTimer = ref<ReturnType<typeof setInterval> | null>(null);

  /**
   * 创建支付订单
   */
  async function createPayment(request: CreatePaymentRequest): Promise<CreatePaymentResponse> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC PaymentService.CreatePayment
      // const response = await paymentServiceClient.createPayment(request);
      
      // 模拟响应
      const order: PaymentOrder = {
        id: Date.now(),
        orderNo: `PAY${Date.now()}`,
        consumerId: request.consumerId,
        paymentMethod: request.paymentMethod,
        paymentType: request.paymentType,
        amount: request.amount,
        status: 'PENDING',
        expiresAt: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
        createdAt: new Date().toISOString(),
      };
      
      const response: CreatePaymentResponse = {
        order,
        paymentData: {
          // 微信/支付宝返回的支付参数
          qrCode: 'https://example.com/qrcode',
        },
      };
      
      currentOrder.value = order;
      return response;
    } catch (err: any) {
      error.value = err.message || '创建支付订单失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 查询支付订单
   */
  async function getPayment(orderNo: string): Promise<PaymentOrder> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC PaymentService.GetPayment
      // const response = await paymentServiceClient.getPayment({ orderNo });
      
      // 模拟响应
      const order: PaymentOrder = {
        id: Date.now(),
        orderNo,
        status: 'PENDING',
        amount: '100.00',
        createdAt: new Date().toISOString(),
      };
      
      currentOrder.value = order;
      return order;
    } catch (err: any) {
      error.value = err.message || '查询支付订单失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 查询支付状态
   */
  async function queryPaymentStatus(orderNo: string): Promise<PaymentOrder> {
    try {
      // TODO: 调用 gRPC PaymentService.QueryPaymentStatus
      // const response = await paymentServiceClient.queryPaymentStatus({ orderNo });
      
      // 模拟响应
      const order: PaymentOrder = {
        id: Date.now(),
        orderNo,
        status: 'SUCCESS',
        amount: '100.00',
        paidAt: new Date().toISOString(),
        createdAt: new Date().toISOString(),
      };
      
      currentOrder.value = order;
      return order;
    } catch (err: any) {
      error.value = err.message || '查询支付状态失败';
      throw err;
    }
  }

  /**
   * 开始轮询支付状态
   */
  function startPolling(orderNo: string, interval: number = 3000, maxAttempts: number = 60) {
    stopPolling();
    
    let attempts = 0;
    pollingTimer.value = setInterval(async () => {
      attempts++;
      
      try {
        const order = await queryPaymentStatus(orderNo);
        
        // 如果支付成功或失败，停止轮询
        if (order.status === 'SUCCESS' || order.status === 'FAILED' || order.status === 'CLOSED') {
          stopPolling();
        }
        
        // 达到最大尝试次数，停止轮询
        if (attempts >= maxAttempts) {
          stopPolling();
        }
      } catch (err) {
        console.error('轮询支付状态失败:', err);
      }
    }, interval);
  }

  /**
   * 停止轮询支付状态
   */
  function stopPolling() {
    if (pollingTimer.value) {
      clearInterval(pollingTimer.value);
      pollingTimer.value = null;
    }
  }

  /**
   * 申请退款
   */
  async function refund(request: RefundRequest): Promise<RefundResponse> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC PaymentService.Refund
      // const response = await paymentServiceClient.refund(request);
      
      // 模拟响应
      const response: RefundResponse = {
        refundNo: `REFUND${Date.now()}`,
        status: 'PENDING',
      };
      
      return response;
    } catch (err: any) {
      error.value = err.message || '申请退款失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 重置状态
   */
  function $reset() {
    stopPolling();
    currentOrder.value = null;
    loading.value = false;
    error.value = null;
  }

  return {
    // 状态
    currentOrder,
    loading,
    error,
    
    // 方法
    createPayment,
    getPayment,
    queryPaymentStatus,
    startPolling,
    stopPolling,
    refund,
    $reset,
  };
});

// 解决热更新问题
const hot = import.meta.hot;
if (hot) {
  hot.accept(acceptHMRUpdate(usePaymentStore, hot));
}
