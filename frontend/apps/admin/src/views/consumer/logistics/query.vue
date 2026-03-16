<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { z } from 'zod';
import { Page, VbenForm, VbenButton } from '@vben/common-ui';
import { useLogisticsStore } from '#/stores/logistics.state';
import { Card, Empty, message } from 'ant-design-vue';

defineOptions({ name: 'ConsumerLogisticsQuery' });

const router = useRouter();
const logisticsStore = useLogisticsStore();

const loading = ref(false);
const logisticsInfo = ref<any>(null);

// 表单 Schema
const formSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入运单号',
      },
      fieldName: 'trackingNo',
      label: '运单号',
      rules: z.string()
        .min(1, { message: '请输入运单号' }),
    },
    {
      component: 'Select',
      componentProps: {
        options: [
          { label: '顺丰速运', value: 'SF' },
          { label: '圆通速递', value: 'YTO' },
          { label: '中通快递', value: 'ZTO' },
          { label: '申通快递', value: 'STO' },
          { label: '韵达快递', value: 'YD' },
          { label: '百世快递', value: 'BEST' },
          { label: '天天快递', value: 'HHTT' },
          { label: '邮政EMS', value: 'EMS' },
        ],
        placeholder: '请选择快递公司（可选）',
        allowClear: true,
      },
      fieldName: 'courierCompany',
      label: '快递公司',
      rules: z.string().optional(),
    },
  ];
});

// 提交查询
async function handleSubmit(values: any) {
  loading.value = true;
  
  try {
    const result = await logisticsStore.queryLogistics({
      trackingNo: values.trackingNo,
      courierCompany: values.courierCompany,
    });
    
    logisticsInfo.value = result;
    message.success('查询成功');
  } catch (error: any) {
    message.error(error.message || '查询失败');
    logisticsInfo.value = null;
  } finally {
    loading.value = false;
  }
}

// 订阅物流状态
async function handleSubscribe() {
  if (!logisticsInfo.value) {
    message.error('请先查询物流信息');
    return;
  }

  try {
    await logisticsStore.subscribeLogistics({
      trackingNo: logisticsInfo.value.trackingNo,
      courierCompany: logisticsInfo.value.courierCompany,
    });
    
    message.success('订阅成功，物流状态变更时会通知您');
  } catch (error: any) {
    message.error(error.message || '订阅失败');
  }
}

// 查看物流轨迹
function goToTracking() {
  if (!logisticsInfo.value) {
    message.error('请先查询物流信息');
    return;
  }

  router.push({
    path: '/consumer/logistics/tracking',
    query: {
      trackingNo: logisticsInfo.value.trackingNo,
      courierCompany: logisticsInfo.value.courierCompany,
    },
  });
}

// 获取状态颜色
function getStatusColor(status?: string): string {
  const colorMap: Record<string, string> = {
    PENDING: 'default',
    IN_TRANSIT: 'blue',
    DELIVERING: 'orange',
    DELIVERED: 'green',
  };
  return colorMap[status || 'PENDING'] || 'default';
}

// 获取状态文本
function getStatusText(status?: string): string {
  const textMap: Record<string, string> = {
    PENDING: '待揽收',
    IN_TRANSIT: '运输中',
    DELIVERING: '派送中',
    DELIVERED: '已签收',
  };
  return textMap[status || 'PENDING'] || '未知';
}
</script>

<template>
  <Page
    title="物流查询"
    description="查询快递物流信息"
  >
    <div class="mx-auto max-w-2xl">
      <Card class="mb-6">
        <VbenForm
          :schema="formSchema"
          :submit-button-options="{ loading, text: '查询' }"
          @submit="handleSubmit"
        />
      </Card>
      
      <Card v-if="logisticsInfo" title="物流信息">
        <div class="space-y-4">
          <div class="flex justify-between">
            <span class="text-gray-600">运单号:</span>
            <span class="font-medium">{{ logisticsInfo.trackingNo }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-600">快递公司:</span>
            <span>{{ logisticsInfo.courierCompany }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-600">物流状态:</span>
            <a-tag :color="getStatusColor(logisticsInfo.status)">
              {{ getStatusText(logisticsInfo.status) }}
            </a-tag>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-600">最后更新:</span>
            <span>{{ logisticsInfo.lastUpdatedAt ? dayjs(logisticsInfo.lastUpdatedAt).format('YYYY-MM-DD HH:mm:ss') : '-' }}</span>
          </div>
          
          <div class="mt-6 flex gap-4">
            <VbenButton type="primary" @click="goToTracking">
              查看物流轨迹
            </VbenButton>
            <VbenButton @click="handleSubscribe">
              订阅状态变更
            </VbenButton>
          </div>
        </div>
      </Card>
      
      <Empty v-else description="请输入运单号查询物流信息" />
    </div>
  </Page>
</template>
