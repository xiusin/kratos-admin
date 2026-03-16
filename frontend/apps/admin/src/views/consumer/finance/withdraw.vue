<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { z } from 'zod';
import { Page, VbenForm } from '@vben/common-ui';
import { useConsumerStore } from '#/stores/consumer.state';
import { useFinanceStore } from '#/stores/finance.state';
import { message } from 'ant-design-vue';

defineOptions({ name: 'ConsumerFinanceWithdraw' });

const router = useRouter();
const consumerStore = useConsumerStore();
const financeStore = useFinanceStore();

const loading = ref(false);

// 表单 Schema
const formSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入提现金额',
        type: 'number',
        min: 10,
        max: 5000,
        step: 0.01,
      },
      fieldName: 'amount',
      label: '提现金额',
      rules: z.string()
        .refine((val) => parseFloat(val) >= 10, { message: '提现金额至少10元' })
        .refine((val) => parseFloat(val) <= 5000, { message: '单日提现金额不超过5000元' }),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入银行账户',
      },
      fieldName: 'bankAccount',
      label: '银行账户',
      rules: z.string()
        .min(1, { message: '请输入银行账户' }),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入银行名称',
      },
      fieldName: 'bankName',
      label: '银行名称',
      rules: z.string()
        .min(1, { message: '请输入银行名称' }),
    },
  ];
});

// 提交提现
async function handleSubmit(values: any) {
  if (!consumerStore.consumerInfo?.id) {
    message.error('用户信息不存在');
    return;
  }

  // 检查余额
  const availableBalance = parseFloat(financeStore.availableBalance);
  const withdrawAmount = parseFloat(values.amount);
  
  if (withdrawAmount > availableBalance) {
    message.error('余额不足');
    return;
  }

  loading.value = true;
  
  try {
    const response = await financeStore.withdraw({
      consumerId: consumerStore.consumerInfo.id,
      amount: values.amount,
      bankAccount: values.bankAccount,
      bankName: values.bankName,
    });
    
    message.success(`提现申请已提交，提现单号：${response.withdrawNo}`);
    
    // 跳转到账户页面
    router.push('/consumer/finance/account');
  } catch (error: any) {
    message.error(error.message || '提现申请失败');
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <Page
    title="申请提现"
    description="将余额提现到银行账户"
  >
    <div class="mx-auto max-w-md">
      <div class="mb-4 rounded-lg bg-gray-50 p-4">
        <div class="flex justify-between">
          <span class="text-gray-600">可用余额:</span>
          <span class="text-lg font-medium text-green-600">
            ¥{{ financeStore.availableBalance }}
          </span>
        </div>
      </div>
      
      <VbenForm
        :schema="formSchema"
        :submit-button-options="{ loading, text: '提交申请' }"
        @submit="handleSubmit"
      />
      
      <div class="mt-4 rounded-lg bg-yellow-50 p-4">
        <h4 class="mb-2 font-medium text-yellow-900">提现说明</h4>
        <ul class="space-y-1 text-sm text-yellow-700">
          <li>• 提现金额：10-5000元</li>
          <li>• 提现需要审核，1-3个工作日到账</li>
          <li>• 提现期间金额会被冻结</li>
          <li>• 审核通过后自动打款到银行账户</li>
          <li>• 审核拒绝后金额会解冻返还</li>
        </ul>
      </div>
    </div>
  </Page>
</template>
