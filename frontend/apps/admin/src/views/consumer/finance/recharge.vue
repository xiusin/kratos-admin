<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { z } from 'zod';
import { Page, VbenForm } from '@vben/common-ui';
import { useConsumerStore } from '#/stores/consumer.state';
import { useFinanceStore } from '#/stores/finance.state';
import { usePaymentStore } from '#/stores/payment.state';
import { message, Modal } from 'ant-design-vue';

defineOptions({ name: 'ConsumerFinanceRecharge' });

const router = useRouter();
const consumerStore = useConsumerStore();
const financeStore = useFinanceStore();
const paymentStore = usePaymentStore();

const loading = ref(false);
const paymentModalVisible = ref(false);
const qrcodeUrl = ref('');

// 表单 Schema
const formSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入充值金额',
        type: 'number',
        min: 0.01,
        step: 0.01,
      },
      fieldName: 'amount',
      label: '充值金额',
      rules: z.string()
        .refine((val) => parseFloat(val) >= 0.01, { message: '充值金额至少0.01元' }),
    },
    {
      component: 'Select',
      componentProps: {
        options: [
          { label: '微信支付', value: 'WECHAT' },
          { label: '支付宝', value: 'ALIPAY' },
        ],
        placeholder: '请选择支付方式',
      },
      fieldName: 'paymentMethod',
      label: '支付方式',
      rules: z.string().min(1, { message: '请选择支付方式' }),
    },
    {
      component: 'Select',
      componentProps: {
        options: [
          { label: 'APP支付', value: 'APP' },
          { label: 'H5支付', value: 'H5' },
          { label: '扫码支付', value: 'QRCODE' },
        ],
        placeholder: '请选择支付类型',
      },
      fieldName: 'paymentType',
      label: '支付类型',
      rules: z.string().min(1, { message: '请选择支付类型' }),
    },
  ];
});

// 提交充值
async function handleSubmit(values: any) {
  if (!consumerStore.consumerInfo?.id) {
    message.error('用户信息不存在');
    return;
  }

  loading.value = true;
  
  try {
    // 1. 创建支付订单
    const paymentOrder = await paymentStore.createPayment({
      consumerId: consumerStore.consumerInfo.id,
      amount: values.amount,
      paymentMethod: values.paymentMethod,
      paymentType: values.paymentType,
      description: '账户充值',
    });
    
    // 2. 根据支付类型处理
    if (values.paymentType === 'QRCODE') {
      // 显示二维码
      qrcodeUrl.value = paymentOrder.qrcodeUrl || '';
      paymentModalVisible.value = true;
      
      // 开始轮询支付状态
      startPollingPaymentStatus(paymentOrder.orderNo!);
    } else if (values.paymentType === 'H5') {
      // 跳转到H5支付页面
      window.location.href = paymentOrder.h5Url || '';
    } else {
      // APP支付
      message.info('请在APP中完成支付');
    }
  } catch (error: any) {
    message.error(error.message || '创建支付订单失败');
  } finally {
    loading.value = false;
  }
}

// 轮询支付状态
let pollingTimer: ReturnType<typeof setInterval> | null = null;

function startPollingPaymentStatus(orderNo: string) {
  // 清除旧的定时器
  if (pollingTimer) {
    clearInterval(pollingTimer);
  }
  
  // 每3秒查询一次
  pollingTimer = setInterval(async () => {
    try {
      const status = await paymentStore.queryPaymentStatus({ orderNo });
      
      if (status.status === 'SUCCESS') {
        // 支付成功
        stopPolling();
        paymentModalVisible.value = false;
        message.success('充值成功');
        
        // 刷新余额
        if (consumerStore.consumerInfo?.id) {
          await financeStore.refreshBalance(consumerStore.consumerInfo.id);
        }
        
        // 跳转到账户页面
        router.push('/consumer/finance/account');
      } else if (status.status === 'FAILED' || status.status === 'CLOSED') {
        // 支付失败或关闭
        stopPolling();
        paymentModalVisible.value = false;
        message.error('支付失败');
      }
    } catch (error) {
      console.error('查询支付状态失败:', error);
    }
  }, 3000);
}

function stopPolling() {
  if (pollingTimer) {
    clearInterval(pollingTimer);
    pollingTimer = null;
  }
}

// 关闭支付弹窗
function handleClosePaymentModal() {
  stopPolling();
  paymentModalVisible.value = false;
}
</script>

<template>
  <Page
    title="账户充值"
    description="为您的账户充值"
  >
    <div class="mx-auto max-w-md">
      <VbenForm
        :schema="formSchema"
        :submit-button-options="{ loading, text: '立即充值' }"
        @submit="handleSubmit"
      />
      
      <div class="mt-4 rounded-lg bg-blue-50 p-4">
        <h4 class="mb-2 font-medium text-blue-900">充值说明</h4>
        <ul class="space-y-1 text-sm text-blue-700">
          <li>• 充值金额最低0.01元</li>
          <li>• 支持微信支付和支付宝</li>
          <li>• 充值成功后立即到账</li>
          <li>• 如有问题请联系客服</li>
        </ul>
      </div>
    </div>
    
    <!-- 支付二维码弹窗 -->
    <Modal
      v-model:open="paymentModalVisible"
      title="扫码支付"
      :footer="null"
      @cancel="handleClosePaymentModal"
    >
      <div class="text-center">
        <div v-if="qrcodeUrl" class="mb-4">
          <img :src="qrcodeUrl" alt="支付二维码" class="mx-auto h-64 w-64" />
        </div>
        <p class="text-gray-600">请使用微信或支付宝扫码支付</p>
        <p class="mt-2 text-sm text-gray-500">支付完成后会自动跳转</p>
      </div>
    </Modal>
  </Page>
</template>
