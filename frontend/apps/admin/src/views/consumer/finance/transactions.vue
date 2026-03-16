<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';
import { onMounted } from 'vue';
import { Page, type VbenFormProps, VbenButton } from '@vben/common-ui';
import dayjs from 'dayjs';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { useConsumerStore } from '#/stores/consumer.state';
import { useFinanceStore, type FinanceTransaction } from '#/stores/finance.state';
import { Tag, message } from 'ant-design-vue';

defineOptions({ name: 'ConsumerFinanceTransactions' });

const consumerStore = useConsumerStore();
const financeStore = useFinanceStore();

// 表单配置
const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Select',
      fieldName: 'transactionType',
      label: '交易类型',
      componentProps: {
        options: [
          { label: '充值', value: 'RECHARGE' },
          { label: '消费', value: 'CONSUME' },
          { label: '提现', value: 'WITHDRAW' },
          { label: '退款', value: 'REFUND' },
        ],
        placeholder: '请选择交易类型',
        allowClear: true,
      },
    },
    {
      component: 'RangePicker',
      fieldName: 'dateRange',
      label: '交易时间',
      componentProps: {
        showTime: false,
        allowClear: true,
      },
    },
  ],
};

// 表格配置
const gridOptions: VxeGridProps<FinanceTransaction> = {
  toolbarConfig: {
    custom: true,
    export: true,
    refresh: true,
    zoom: true,
    slots: {
      buttons: 'toolbar_buttons',
    },
  },
  height: 'auto',
  exportConfig: {},
  pagerConfig: {},
  rowConfig: {
    isHover: true,
  },
  stripe: true,

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        if (!consumerStore.consumerInfo?.id) {
          return { page: { total: 0 }, result: [] };
        }

        let startDate: any;
        let endDate: any;
        if (
          formValues.dateRange !== undefined &&
          formValues.dateRange.length === 2
        ) {
          startDate = dayjs(formValues.dateRange[0]).format('YYYY-MM-DD');
          endDate = dayjs(formValues.dateRange[1]).format('YYYY-MM-DD');
        }

        await financeStore.listTransactions({
          consumerId: consumerStore.consumerInfo.id,
          transactionType: formValues.transactionType,
          startDate,
          endDate,
          page: page.currentPage,
          pageSize: page.pageSize,
        });

        return {
          page: {
            total: financeStore.totalTransactions,
          },
          result: financeStore.transactions,
        };
      },
    },
  },

  columns: [
    {
      title: '交易时间',
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 160,
    },
    {
      title: '流水号',
      field: 'transactionNo',
      width: 180,
    },
    {
      title: '交易类型',
      field: 'transactionType',
      width: 100,
      slots: { default: 'transactionType' },
    },
    {
      title: '交易金额',
      field: 'amount',
      width: 120,
      slots: { default: 'amount' },
    },
    {
      title: '交易前余额',
      field: 'balanceBefore',
      width: 120,
    },
    {
      title: '交易后余额',
      field: 'balanceAfter',
      width: 120,
    },
    {
      title: '描述',
      field: 'description',
      minWidth: 150,
    },
    {
      title: '关联订单号',
      field: 'relatedOrderNo',
      width: 180,
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });

// 加载数据
onMounted(async () => {
  if (consumerStore.consumerInfo?.id) {
    try {
      await financeStore.listTransactions({
        consumerId: consumerStore.consumerInfo.id,
        page: 1,
        pageSize: 10,
      });
    } catch (error: any) {
      message.error(error.message || '加载流水失败');
    }
  }
});

// 获取交易类型颜色
function getTransactionTypeColor(type?: string): string {
  const colorMap: Record<string, string> = {
    RECHARGE: 'green',
    CONSUME: 'blue',
    WITHDRAW: 'orange',
    REFUND: 'purple',
  };
  return colorMap[type || 'RECHARGE'] || 'default';
}

// 获取交易类型文本
function getTransactionTypeText(type?: string): string {
  const textMap: Record<string, string> = {
    RECHARGE: '充值',
    CONSUME: '消费',
    WITHDRAW: '提现',
    REFUND: '退款',
  };
  return textMap[type || 'RECHARGE'] || '未知';
}

// 导出流水
async function handleExport() {
  if (!consumerStore.consumerInfo?.id) {
    message.error('用户信息不存在');
    return;
  }

  try {
    const blob = await financeStore.exportTransactions({
      consumerId: consumerStore.consumerInfo.id,
    });
    
    // 下载文件
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `财务流水_${dayjs().format('YYYYMMDD')}.csv`;
    link.click();
    window.URL.revokeObjectURL(url);
    
    message.success('导出成功');
  } catch (error: any) {
    message.error(error.message || '导出失败');
  }
}
</script>

<template>
  <Page auto-content-height>
    <Grid table-title="财务流水">
      <template #toolbar_buttons>
        <VbenButton type="primary" @click="handleExport">
          导出CSV
        </VbenButton>
      </template>
      
      <template #transactionType="{ row }">
        <Tag :color="getTransactionTypeColor(row.transactionType)">
          {{ getTransactionTypeText(row.transactionType) }}
        </Tag>
      </template>
      
      <template #amount="{ row }">
        <span
          :class="{
            'text-green-600': row.transactionType === 'RECHARGE' || row.transactionType === 'REFUND',
            'text-red-600': row.transactionType === 'CONSUME' || row.transactionType === 'WITHDRAW',
          }"
        >
          {{ row.transactionType === 'RECHARGE' || row.transactionType === 'REFUND' ? '+' : '-' }}
          ¥{{ row.amount }}
        </span>
      </template>
    </Grid>
  </Page>
</template>
