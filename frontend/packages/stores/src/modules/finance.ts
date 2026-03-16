import { ref, computed } from 'vue';
import { acceptHMRUpdate, defineStore } from 'pinia';

/**
 * 财务账户信息接口
 */
export interface FinanceAccount {
  id?: number;
  tenantId?: number;
  consumerId?: number;
  balance?: string;
  frozenBalance?: string;
  createdAt?: string;
  updatedAt?: string;
}

/**
 * 财务流水信息接口
 */
export interface FinanceTransaction {
  id?: number;
  tenantId?: number;
  consumerId?: number;
  transactionNo?: string;
  transactionType?: 'RECHARGE' | 'CONSUME' | 'WITHDRAW' | 'REFUND';
  amount?: string;
  balanceBefore?: string;
  balanceAfter?: string;
  description?: string;
  relatedOrderNo?: string;
  operatorId?: number;
  createdAt?: string;
}

/**
 * 充值请求接口
 */
export interface RechargeRequest {
  consumerId: number;
  amount: string;
  paymentOrderNo: string;
}

/**
 * 提现请求接口
 */
export interface WithdrawRequest {
  consumerId: number;
  amount: string;
  bankAccount?: string;
  bankName?: string;
}

/**
 * 提现响应接口
 */
export interface WithdrawResponse {
  withdrawNo: string;
  status: 'PENDING' | 'APPROVED' | 'REJECTED';
}

/**
 * 查询流水请求接口
 */
export interface ListTransactionsRequest {
  consumerId: number;
  transactionType?: 'RECHARGE' | 'CONSUME' | 'WITHDRAW' | 'REFUND';
  startDate?: string;
  endDate?: string;
  page?: number;
  pageSize?: number;
}

/**
 * @zh_CN 财务服务状态管理
 */
export const useFinanceStore = defineStore('finance', () => {
  // 状态
  const account = ref<FinanceAccount | null>(null);
  const transactions = ref<FinanceTransaction[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const totalTransactions = ref(0);

  // 计算属性
  const availableBalance = computed(() => {
    if (!account.value?.balance) return '0.00';
    const balance = parseFloat(account.value.balance);
    const frozen = parseFloat(account.value.frozenBalance || '0');
    return (balance - frozen).toFixed(2);
  });

  const totalBalance = computed(() => {
    return account.value?.balance || '0.00';
  });

  const frozenBalance = computed(() => {
    return account.value?.frozenBalance || '0.00';
  });

  /**
   * 获取账户余额
   */
  async function getAccount(consumerId: number): Promise<FinanceAccount> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC FinanceService.GetAccount
      // const response = await financeServiceClient.getAccount({ consumerId });
      
      // 模拟响应
      const accountData: FinanceAccount = {
        id: 1,
        consumerId,
        balance: '1000.00',
        frozenBalance: '0.00',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };
      
      account.value = accountData;
      return accountData;
    } catch (err: any) {
      error.value = err.message || '获取账户信息失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 充值
   */
  async function recharge(request: RechargeRequest): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC FinanceService.Recharge
      // await financeServiceClient.recharge(request);
      
      // 模拟充值成功，更新本地余额
      if (account.value) {
        const currentBalance = parseFloat(account.value.balance || '0');
        const rechargeAmount = parseFloat(request.amount);
        account.value.balance = (currentBalance + rechargeAmount).toFixed(2);
      }
      
      // 刷新账户信息
      await getAccount(request.consumerId);
    } catch (err: any) {
      error.value = err.message || '充值失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 申请提现
   */
  async function withdraw(request: WithdrawRequest): Promise<WithdrawResponse> {
    loading.value = true;
    error.value = null;
    
    try {
      // 验证提现金额
      const withdrawAmount = parseFloat(request.amount);
      if (withdrawAmount < 10) {
        throw new Error('提现金额不能低于10元');
      }
      if (withdrawAmount > 5000) {
        throw new Error('单日提现金额不能超过5000元');
      }
      
      const available = parseFloat(availableBalance.value);
      if (withdrawAmount > available) {
        throw new Error('余额不足');
      }
      
      // TODO: 调用 gRPC FinanceService.Withdraw
      // const response = await financeServiceClient.withdraw(request);
      
      // 模拟响应
      const response: WithdrawResponse = {
        withdrawNo: `WD${Date.now()}`,
        status: 'PENDING',
      };
      
      // 冻结提现金额
      if (account.value) {
        const currentFrozen = parseFloat(account.value.frozenBalance || '0');
        account.value.frozenBalance = (currentFrozen + withdrawAmount).toFixed(2);
      }
      
      return response;
    } catch (err: any) {
      error.value = err.message || '申请提现失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 查询财务流水
   */
  async function listTransactions(request: ListTransactionsRequest): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC FinanceService.ListTransactions
      // const response = await financeServiceClient.listTransactions(request);
      
      // 模拟响应
      const mockTransactions: FinanceTransaction[] = [
        {
          id: 1,
          consumerId: request.consumerId,
          transactionNo: `TXN${Date.now()}`,
          transactionType: 'RECHARGE',
          amount: '100.00',
          balanceBefore: '900.00',
          balanceAfter: '1000.00',
          description: '充值',
          createdAt: new Date().toISOString(),
        },
        {
          id: 2,
          consumerId: request.consumerId,
          transactionNo: `TXN${Date.now() - 1000}`,
          transactionType: 'CONSUME',
          amount: '50.00',
          balanceBefore: '950.00',
          balanceAfter: '900.00',
          description: '消费',
          createdAt: new Date(Date.now() - 86400000).toISOString(),
        },
      ];
      
      transactions.value = mockTransactions;
      totalTransactions.value = mockTransactions.length;
    } catch (err: any) {
      error.value = err.message || '查询流水失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 导出财务流水
   */
  async function exportTransactions(request: ListTransactionsRequest): Promise<Blob> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC FinanceService.ExportTransactions
      // const response = await financeServiceClient.exportTransactions(request);
      
      // 模拟CSV导出
      const csvContent = [
        '流水号,交易类型,金额,交易前余额,交易后余额,描述,时间',
        'TXN001,充值,100.00,900.00,1000.00,充值,2024-01-01 12:00:00',
        'TXN002,消费,50.00,950.00,900.00,消费,2024-01-02 12:00:00',
      ].join('\n');
      
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
      return blob;
    } catch (err: any) {
      error.value = err.message || '导出流水失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 实时更新余额（用于支付成功后）
   */
  async function refreshBalance(consumerId: number): Promise<void> {
    try {
      await getAccount(consumerId);
    } catch (err) {
      console.error('刷新余额失败:', err);
    }
  }

  /**
   * 重置状态
   */
  function $reset() {
    account.value = null;
    transactions.value = [];
    loading.value = false;
    error.value = null;
    totalTransactions.value = 0;
  }

  return {
    // 状态
    account,
    transactions,
    loading,
    error,
    totalTransactions,
    
    // 计算属性
    availableBalance,
    totalBalance,
    frozenBalance,
    
    // 方法
    getAccount,
    recharge,
    withdraw,
    listTransactions,
    exportTransactions,
    refreshBalance,
    $reset,
  };
});

// 解决热更新问题
const hot = import.meta.hot;
if (hot) {
  hot.accept(acceptHMRUpdate(useFinanceStore, hot));
}
