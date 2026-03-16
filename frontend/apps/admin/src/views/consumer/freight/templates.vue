<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';
import { onMounted } from 'vue';
import { Page, type VbenFormProps } from '@vben/common-ui';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { useFreightStore, type FreightTemplate } from '#/stores/freight.state';
import { Tag, message } from 'ant-design-vue';

defineOptions({ name: 'ConsumerFreightTemplates' });

const freightStore = useFreightStore();

// 表单配置
const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'name',
      label: '模板名称',
      componentProps: {
        placeholder: '请输入模板名称',
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'calculationType',
      label: '计算方式',
      componentProps: {
        options: [
          { label: '按重量', value: 'BY_WEIGHT' },
          { label: '按距离', value: 'BY_DISTANCE' },
        ],
        placeholder: '请选择计算方式',
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'isActive',
      label: '状态',
      componentProps: {
        options: [
          { label: '启用', value: true },
          { label: '禁用', value: false },
        ],
        placeholder: '请选择状态',
        allowClear: true,
      },
    },
  ],
};

// 表格配置
const gridOptions: VxeGridProps<FreightTemplate> = {
  toolbarConfig: {
    custom: true,
    refresh: true,
    zoom: true,
  },
  height: 'auto',
  pagerConfig: {},
  rowConfig: {
    isHover: true,
  },
  stripe: true,

  proxyConfig: {
    ajax: {
      query: async ({ page }, formValues) => {
        await freightStore.listFreightTemplates({
          name: formValues.name,
          calculationType: formValues.calculationType,
          isActive: formValues.isActive,
          page: page.currentPage,
          pageSize: page.pageSize,
        });

        return {
          page: {
            total: freightStore.totalTemplates,
          },
          result: freightStore.templates,
        };
      },
    },
  },

  columns: [
    {
      title: '模板名称',
      field: 'name',
      minWidth: 150,
    },
    {
      title: '计算方式',
      field: 'calculationType',
      width: 120,
      slots: { default: 'calculationType' },
    },
    {
      title: '首重/首距',
      field: 'firstWeight',
      width: 120,
      slots: { default: 'first' },
    },
    {
      title: '首重价格',
      field: 'firstPrice',
      width: 120,
      slots: { default: 'firstPrice' },
    },
    {
      title: '续重/续距',
      field: 'additionalWeight',
      width: 120,
      slots: { default: 'additional' },
    },
    {
      title: '续重价格',
      field: 'additionalPrice',
      width: 120,
      slots: { default: 'additionalPrice' },
    },
    {
      title: '状态',
      field: 'isActive',
      width: 80,
      slots: { default: 'isActive' },
    },
    {
      title: '创建时间',
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 160,
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });

// 加载数据
onMounted(async () => {
  try {
    await freightStore.listFreightTemplates({
      page: 1,
      pageSize: 10,
    });
  } catch (error: any) {
    message.error(error.message || '加载模板列表失败');
  }
});

// 获取计算方式颜色
function getCalculationTypeColor(type?: string): string {
  const colorMap: Record<string, string> = {
    BY_WEIGHT: 'blue',
    BY_DISTANCE: 'purple',
  };
  return colorMap[type || 'BY_WEIGHT'] || 'default';
}

// 获取计算方式文本
function getCalculationTypeText(type?: string): string {
  const textMap: Record<string, string> = {
    BY_WEIGHT: '按重量',
    BY_DISTANCE: '按距离',
  };
  return textMap[type || 'BY_WEIGHT'] || '未知';
}
</script>

<template>
  <Page auto-content-height>
    <Grid table-title="运费模板">
      <template #calculationType="{ row }">
        <Tag :color="getCalculationTypeColor(row.calculationType)">
          {{ getCalculationTypeText(row.calculationType) }}
        </Tag>
      </template>
      
      <template #first="{ row }">
        {{ row.firstWeight || '-' }}
        {{ row.calculationType === 'BY_WEIGHT' ? 'kg' : 'km' }}
      </template>
      
      <template #firstPrice="{ row }">
        ¥{{ row.firstPrice || '0.00' }}
      </template>
      
      <template #additional="{ row }">
        {{ row.additionalWeight || '-' }}
        {{ row.calculationType === 'BY_WEIGHT' ? 'kg' : 'km' }}
      </template>
      
      <template #additionalPrice="{ row }">
        ¥{{ row.additionalPrice || '0.00' }}
      </template>
      
      <template #isActive="{ row }">
        <Tag :color="row.isActive ? 'success' : 'default'">
          {{ row.isActive ? '启用' : '禁用' }}
        </Tag>
      </template>
    </Grid>
  </Page>
</template>
