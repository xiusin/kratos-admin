<script lang="ts" setup>
import { onMounted, ref } from 'vue';
import { useRoute } from 'vue-router';
import { Page } from '@vben/common-ui';
import { useLogisticsStore } from '#/stores/logistics.state';
import { Card, Timeline, Empty, message } from 'ant-design-vue';
import dayjs from 'dayjs';

defineOptions({ name: 'ConsumerLogisticsTracking' });

const route = useRoute();
const logisticsStore = useLogisticsStore();

const loading = ref(false);
const trackingInfo = ref<any>(null);

// 加载物流轨迹
onMounted(async () => {
  const trackingNo = route.query.trackingNo as string;
  const courierCompany = route.query.courierCompany as string;
  
  if (!trackingNo) {
    message.error('缺少运单号');
    return;
  }

  loading.value = true;
  
  try {
    const result = await logisticsStore.queryLogistics({
      trackingNo,
      courierCompany,
    });
    
    trackingInfo.value = result;
  } catch (error: any) {
    message.error(error.message || '加载物流轨迹失败');
  } finally {
    loading.value = false;
  }
});

// 获取状态颜色
function getStatusColor(status?: string): string {
  const colorMap: Record<string, string> = {
    PENDING: 'gray',
    IN_TRANSIT: 'blue',
    DELIVERING: 'orange',
    DELIVERED: 'green',
  };
  return colorMap[status || 'PENDING'] || 'gray';
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
    title="物流轨迹"
    description="查看详细的物流轨迹信息"
  >
    <div class="mx-auto max-w-3xl">
      <Card v-if="trackingInfo" :loading="loading">
        <div class="mb-6 space-y-2">
          <div class="flex justify-between">
            <span class="text-gray-600">运单号:</span>
            <span class="font-medium">{{ trackingInfo.trackingNo }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-600">快递公司:</span>
            <span>{{ trackingInfo.courierCompany }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-600">当前状态:</span>
            <a-tag :color="getStatusColor(trackingInfo.status)">
              {{ getStatusText(trackingInfo.status) }}
            </a-tag>
          </div>
        </div>
        
        <div class="border-t pt-6">
          <h3 class="mb-4 text-lg font-medium">物流轨迹</h3>
          
          <Timeline v-if="trackingInfo.trackingInfo && trackingInfo.trackingInfo.length > 0">
            <TimelineItem
              v-for="(item, index) in trackingInfo.trackingInfo"
              :key="index"
              :color="index === 0 ? 'green' : 'gray'"
            >
              <div class="space-y-1">
                <div class="font-medium">{{ item.description }}</div>
                <div class="text-sm text-gray-500">
                  {{ item.location || '' }}
                </div>
                <div class="text-xs text-gray-400">
                  {{ item.time ? dayjs(item.time).format('YYYY-MM-DD HH:mm:ss') : '' }}
                </div>
              </div>
            </TimelineItem>
          </Timeline>
          
          <Empty v-else description="暂无物流轨迹信息" />
        </div>
      </Card>
      
      <Empty v-else :loading="loading" description="加载物流轨迹中..." />
    </div>
  </Page>
</template>
