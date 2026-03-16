<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';
import { Page, type VbenFormProps } from '@vben/common-ui';
import dayjs from 'dayjs';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { $t } from '#/locales';
import { Tag } from 'ant-design-vue';

defineOptions({ name: 'ConsumerLoginLogs' });

/**
 * 登录日志接口
 */
interface LoginLog {
  id?: number;
  tenantId?: number;
  consumerId?: number;
  phone?: string;
  loginType?: 'PHONE' | 'WECHAT';
  success?: boolean;
  failReason?: string;
  ipAddress?: string;
  userAgent?: string;
  deviceType?: string;
  loginAt?: string;
}

// 表单配置
const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Input',
      fieldName: 'phone',
      label: '手机号',
      componentProps: {
        placeholder: '请输入手机号',
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'loginType',
      label: '登录方式',
      componentProps: {
        options: [
          { label: '手机号登录', value: 'PHONE' },
          { label: '微信登录', value: 'WECHAT' },
        ],
        placeholder: '请选择登录方式',
        allowClear: true,
      },
    },
    {
      component: 'Select',
      fieldName: 'success',
      label: '登录状态',
      componentProps: {
        options: [
          { label: '成功', value: true },
          { label: '失败', value: false },
        ],
        placeholder: '请选择登录状态',
        allowClear: true,
      },
    },
    {
      component: 'RangePicker',
      fieldName: 'loginTime',
      label: '登录时间',
      componentProps: {
        showTime: true,
        allowClear: true,
      },
    },
  ],
};

// 表格配置
const gridOptions: VxeGridProps<LoginLog> = {
  toolbarConfig: {
    custom: true,
    export: true,
    refresh: true,
    zoom: true,
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
        console.log('query:', formValues);

        let startTime: any;
        let endTime: any;
        if (
          formValues.loginTime !== undefined &&
          formValues.loginTime.length === 2
        ) {
          startTime = dayjs(formValues.loginTime[0]).format(
            'YYYY-MM-DD HH:mm:ss',
          );
          endTime = dayjs(formValues.loginTime[1]).format(
            'YYYY-MM-DD HH:mm:ss',
          );
        }

        // TODO: 调用 ConsumerService.ListLoginLogs
        // const response = await consumerServiceClient.listLoginLogs({
        //   page: page.currentPage,
        //   pageSize: page.pageSize,
        //   phone: formValues.phone,
        //   loginType: formValues.loginType,
        //   success: formValues.success,
        //   startTime,
        //   endTime,
        // });

        // 模拟数据
        const mockData: LoginLog[] = [
          {
            id: 1,
            phone: '138****8888',
            loginType: 'PHONE',
            success: true,
            ipAddress: '192.168.1.1',
            userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
            deviceType: 'PC',
            loginAt: new Date().toISOString(),
          },
          {
            id: 2,
            phone: '138****8888',
            loginType: 'WECHAT',
            success: false,
            failReason: '密码错误',
            ipAddress: '192.168.1.2',
            userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 14_0)',
            deviceType: 'Mobile',
            loginAt: new Date(Date.now() - 86400000).toISOString(),
          },
        ];

        return {
          page: {
            total: mockData.length,
          },
          result: mockData,
        };
      },
    },
  },

  columns: [
    {
      title: '登录时间',
      field: 'loginAt',
      formatter: 'formatDateTime',
      width: 160,
    },
    {
      title: '登录状态',
      field: 'success',
      width: 100,
      slots: { default: 'success' },
    },
    {
      title: '手机号',
      field: 'phone',
      width: 120,
    },
    {
      title: '登录方式',
      field: 'loginType',
      width: 100,
      slots: { default: 'loginType' },
    },
    {
      title: 'IP地址',
      field: 'ipAddress',
      width: 140,
    },
    {
      title: '设备类型',
      field: 'deviceType',
      width: 100,
    },
    {
      title: 'User Agent',
      field: 'userAgent',
      minWidth: 200,
    },
    {
      title: '失败原因',
      field: 'failReason',
      width: 120,
    },
  ],
};

const [Grid] = useVbenVxeGrid({ gridOptions, formOptions });

// 获取登录状态颜色
function getSuccessColor(success?: boolean): string {
  return success ? 'success' : 'error';
}

// 获取登录状态文本
function getSuccessText(success?: boolean): string {
  return success ? '成功' : '失败';
}

// 获取登录方式颜色
function getLoginTypeColor(loginType?: string): string {
  const colorMap: Record<string, string> = {
    PHONE: 'blue',
    WECHAT: 'green',
  };
  return colorMap[loginType || 'PHONE'] || 'default';
}

// 获取登录方式文本
function getLoginTypeText(loginType?: string): string {
  const textMap: Record<string, string> = {
    PHONE: '手机号登录',
    WECHAT: '微信登录',
  };
  return textMap[loginType || 'PHONE'] || '未知';
}
</script>

<template>
  <Page auto-content-height>
    <Grid table-title="登录日志">
      <template #success="{ row }">
        <Tag :color="getSuccessColor(row.success)">
          {{ getSuccessText(row.success) }}
        </Tag>
      </template>
      <template #loginType="{ row }">
        <Tag :color="getLoginTypeColor(row.loginType)">
          {{ getLoginTypeText(row.loginType) }}
        </Tag>
      </template>
    </Grid>
  </Page>
</template>
