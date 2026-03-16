<script lang="ts" setup>
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Page, VbenButton } from '@vben/common-ui';
import { useConsumerStore } from '#/stores/consumer.state';
import { useFinanceStore } from '#/stores/finance.state';
import { Card, Statistic, Row, Col } from 'ant-design-vue';
import { message } from 'ant-design-vue';

defineOptions({ name: 'ConsumerFinanceAccount' });

const router = useRouter();
const consumerStore = useConsumerStore();
const financeStore = useFinanceStore();

const loading = ref(false);

// 加载账户信息
onMounted(async () => {
  if (consumerStore.consumerInfo?.id) {
    loading.value = true;
    try {
      await financeStore.getAccount(consumerStore.consumerInfo.id);
    } catch (error: any) {
      message.error(error.message || '加载账户信息失败');
    } finally {
      loading.value = false;
    }
  }
});

// 跳转到充值页面
function goToRecharge() {
  router.push('/consumer/finance/recharge');
}

// 跳转到提现页面
function goToWithdraw() {
  router.push('/consumer/finance/withdraw');
}

// 跳转到流水页面
function goToTransactions() {
  router.push('/consumer/finance/transactions');
}

// 刷新余额
async function refreshBalance() {
  if (consumerStore.consumerInfo?.id) {
    loading.value = true;
    try {
      await financeStore.refreshBalance(consumerStore.consumerInfo.id);
      message.success('刷新成功');
    } catch (error: any) {
      message.error(error.message || '刷新失败');
    } finally {
      loading.value = false;
    }
  }
}
</script>

<template>
  <Page
    title="我的账户"
    description="查看账户余额和财务信息"
  >
    <div class="mx-auto max-w-4xl">
      <Card :loading="loading" class="mb-6">
        <Row :gutter="16">
          <Col :span="8">
            <Statistic
              title="总余额"
              :value="financeStore.totalBalance"
              prefix="¥"
              :precision="2"
            />
          </Col>
          <Col :span="8">
            <Statistic
              title="可用余额"
              :value="financeStore.availableBalance"
              prefix="¥"
              :precision="2"
            />
          </Col>
          <Col :span="8">
            <Statistic
              title="冻结余额"
              :value="financeStore.frozenBalance"
              prefix="¥"
              :precision="2"
            />
          </Col>
        </Row>
        
        <div class="mt-6 flex gap-4">
          <VbenButton type="primary" @click="goToRecharge">
            充值
          </VbenButton>
          <VbenButton @click="goToWithdraw">
            提现
          </VbenButton>
          <VbenButton @click="goToTransactions">
            查看流水
          </VbenButton>
          <VbenButton @click="refreshBalance">
            刷新余额
          </VbenButton>
        </div>
      </Card>
      
      <Card title="账户信息">
        <div class="space-y-2">
          <div class="flex justify-between">
            <span class="text-gray-600">账户ID:</span>
            <span>{{ financeStore.account?.id || '-' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-600">用户ID:</span>
            <span>{{ financeStore.account?.consumerId || '-' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-600">创建时间:</span>
            <span>{{ financeStore.account?.createdAt ? dayjs(financeStore.account.createdAt).format('YYYY-MM-DD HH:mm:ss') : '-' }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-600">更新时间:</span>
            <span>{{ financeStore.account?.updatedAt ? dayjs(financeStore.account.updatedAt).format('YYYY-MM-DD HH:mm:ss') : '-' }}</span>
          </div>
        </div>
      </Card>
    </div>
  </Page>
</template>
