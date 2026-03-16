import { ref } from 'vue';
import { acceptHMRUpdate, defineStore } from 'pinia';

/**
 * 物流跟踪信息接口
 */
export interface LogisticsTracking {
  id?: number;
  tenantId?: number;
  trackingNo?: string;
  courierCompany?: string;
  status?: 'PENDING' | 'IN_TRANSIT' | 'DELIVERING' | 'DELIVERED';
  trackingInfo?: LogisticsTrackingInfo[];
  lastUpdatedAt?: string;
  createdAt?: string;
}

/**
 * 物流轨迹信息接口
 */
export interface LogisticsTrackingInfo {
  time: string;
  status: string;
  location?: string;
  description: string;
}

/**
 * 查询物流请求接口
 */
export interface QueryLogisticsRequest {
  trackingNo: string;
  courierCompany?: string;
}

/**
 * 订阅物流请求接口
 */
export interface SubscribeLogisticsRequest {
  trackingNo: string;
  courierCompany: string;
  phone?: string;
}

/**
 * @zh_CN 物流服务状态管理
 */
export const useLogisticsStore = defineStore('logistics', () => {
  // 状态
  const trackingList = ref<Map<string, LogisticsTracking>>(new Map());
  const loading = ref(false);
  const error = ref<string | null>(null);
  const pollingTimers = ref<Map<string, ReturnType<typeof setInterval>>>(new Map());

  /**
   * 查询物流信息
   */
  async function queryLogistics(request: QueryLogisticsRequest): Promise<LogisticsTracking> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC LogisticsService.QueryLogistics
      // const response = await logisticsServiceClient.queryLogistics(request);
      
      // 模拟响应
      const tracking: LogisticsTracking = {
        id: Date.now(),
        trackingNo: request.trackingNo,
        courierCompany: request.courierCompany || '顺丰速运',
        status: 'IN_TRANSIT',
        trackingInfo: [
          {
            time: new Date().toISOString(),
            status: '运输中',
            location: '北京市',
            description: '快件已到达北京转运中心',
          },
          {
            time: new Date(Date.now() - 3600000).toISOString(),
            status: '已揽收',
            location: '上海市',
            description: '快件已从上海发出',
          },
        ],
        lastUpdatedAt: new Date().toISOString(),
        createdAt: new Date().toISOString(),
      };
      
      // 保存到缓存
      trackingList.value.set(request.trackingNo, tracking);
      
      return tracking;
    } catch (err: any) {
      error.value = err.message || '查询物流信息失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 订阅物流状态
   */
  async function subscribeLogistics(request: SubscribeLogisticsRequest): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC LogisticsService.SubscribeLogistics
      // await logisticsServiceClient.subscribeLogistics(request);
      
      console.log('订阅物流状态:', request.trackingNo);
      
      // 开始轮询物流信息
      startPolling(request.trackingNo, request.courierCompany);
    } catch (err: any) {
      error.value = err.message || '订阅物流状态失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 开始轮询物流信息
   */
  function startPolling(
    trackingNo: string,
    courierCompany?: string,
    interval: number = 300000 // 默认5分钟
  ): void {
    // 停止已存在的轮询
    stopPolling(trackingNo);
    
    // 立即查询一次
    queryLogistics({ trackingNo, courierCompany });
    
    // 启动定时轮询
    const timer = setInterval(async () => {
      try {
        const tracking = await queryLogistics({ trackingNo, courierCompany });
        
        // 如果已签收，停止轮询
        if (tracking.status === 'DELIVERED') {
          stopPolling(trackingNo);
        }
      } catch (err) {
        console.error('轮询物流信息失败:', err);
      }
    }, interval);
    
    pollingTimers.value.set(trackingNo, timer);
  }

  /**
   * 停止轮询物流信息
   */
  function stopPolling(trackingNo: string): void {
    const timer = pollingTimers.value.get(trackingNo);
    if (timer) {
      clearInterval(timer);
      pollingTimers.value.delete(trackingNo);
    }
  }

  /**
   * 停止所有轮询
   */
  function stopAllPolling(): void {
    pollingTimers.value.forEach((timer) => {
      clearInterval(timer);
    });
    pollingTimers.value.clear();
  }

  /**
   * 获取缓存的物流信息
   */
  function getCachedTracking(trackingNo: string): LogisticsTracking | undefined {
    return trackingList.value.get(trackingNo);
  }

  /**
   * 查询物流历史
   */
  async function listLogisticsHistory(
    page: number = 1,
    pageSize: number = 20
  ): Promise<LogisticsTracking[]> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC LogisticsService.ListLogisticsHistory
      // const response = await logisticsServiceClient.listLogisticsHistory({ page, pageSize });
      
      // 模拟响应
      const history: LogisticsTracking[] = [
        {
          id: 1,
          trackingNo: 'SF1234567890',
          courierCompany: '顺丰速运',
          status: 'DELIVERED',
          lastUpdatedAt: new Date(Date.now() - 86400000).toISOString(),
          createdAt: new Date(Date.now() - 172800000).toISOString(),
        },
        {
          id: 2,
          trackingNo: 'YTO9876543210',
          courierCompany: '圆通速递',
          status: 'IN_TRANSIT',
          lastUpdatedAt: new Date().toISOString(),
          createdAt: new Date(Date.now() - 3600000).toISOString(),
        },
      ];
      
      return history;
    } catch (err: any) {
      error.value = err.message || '查询物流历史失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取物流状态文本
   */
  function getStatusText(status?: string): string {
    const statusMap: Record<string, string> = {
      PENDING: '待揽收',
      IN_TRANSIT: '运输中',
      DELIVERING: '派送中',
      DELIVERED: '已签收',
    };
    return statusMap[status || ''] || '未知';
  }

  /**
   * 获取物流状态颜色
   */
  function getStatusColor(status?: string): string {
    const colorMap: Record<string, string> = {
      PENDING: 'warning',
      IN_TRANSIT: 'processing',
      DELIVERING: 'processing',
      DELIVERED: 'success',
    };
    return colorMap[status || ''] || 'default';
  }

  /**
   * 重置状态
   */
  function $reset() {
    stopAllPolling();
    trackingList.value.clear();
    loading.value = false;
    error.value = null;
  }

  return {
    // 状态
    trackingList,
    loading,
    error,
    
    // 方法
    queryLogistics,
    subscribeLogistics,
    startPolling,
    stopPolling,
    stopAllPolling,
    getCachedTracking,
    listLogisticsHistory,
    getStatusText,
    getStatusColor,
    $reset,
  };
});

// 解决热更新问题
const hot = import.meta.hot;
if (hot) {
  hot.accept(acceptHMRUpdate(useLogisticsStore, hot));
}
