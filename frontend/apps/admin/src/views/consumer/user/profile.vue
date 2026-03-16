<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import { computed, onMounted, ref } from 'vue';
import { z } from 'zod';
import { Page, VbenForm } from '@vben/common-ui';
import { useConsumerStore } from '#/stores/consumer.state';
import { message } from 'ant-design-vue';

defineOptions({ name: 'ConsumerProfile' });

const consumerStore = useConsumerStore();
const loading = ref(false);
const formRef = ref();

// 表单 Schema
const formSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入昵称',
      },
      fieldName: 'nickname',
      label: '昵称',
      rules: z.string()
        .max(50, { message: '昵称最多50个字符' })
        .optional(),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入邮箱',
      },
      fieldName: 'email',
      label: '邮箱',
      rules: z.string()
        .email({ message: '请输入正确的邮箱地址' })
        .optional(),
    },
    {
      component: 'VbenInput',
      componentProps: {
        disabled: true,
      },
      fieldName: 'phone',
      label: '手机号',
    },
    {
      component: 'VbenInput',
      componentProps: {
        disabled: true,
      },
      fieldName: 'status',
      label: '账户状态',
    },
    {
      component: 'VbenInput',
      componentProps: {
        disabled: true,
      },
      fieldName: 'riskScore',
      label: '风险评分',
    },
  ];
});

// 加载用户信息
onMounted(async () => {
  if (consumerStore.consumerInfo?.id) {
    try {
      await consumerStore.getConsumer(consumerStore.consumerInfo.id);
      
      // 填充表单
      if (formRef.value && consumerStore.consumerInfo) {
        formRef.value.setValues({
          nickname: consumerStore.consumerInfo.nickname || '',
          email: consumerStore.consumerInfo.email || '',
          phone: consumerStore.consumerInfo.phone || '',
          status: getStatusText(consumerStore.consumerInfo.status),
          riskScore: consumerStore.consumerInfo.riskScore || 0,
        });
      }
    } catch (error: any) {
      message.error(error.message || '加载用户信息失败');
    }
  }
});

// 获取状态文本
function getStatusText(status?: string): string {
  const statusMap: Record<string, string> = {
    NORMAL: '正常',
    LOCKED: '已锁定',
    DEACTIVATED: '已注销',
  };
  return statusMap[status || 'NORMAL'] || '未知';
}

// 提交更新
async function handleSubmit(values: any) {
  if (!consumerStore.consumerInfo?.id) {
    message.error('用户信息不存在');
    return;
  }

  loading.value = true;
  
  try {
    await consumerStore.updateConsumer({
      id: consumerStore.consumerInfo.id,
      nickname: values.nickname,
      email: values.email,
    });
    
    message.success('更新成功');
  } catch (error: any) {
    message.error(error.message || '更新失败');
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <Page
    title="个人信息"
    description="查看和编辑您的个人信息"
  >
    <div class="mx-auto max-w-2xl">
      <VbenForm
        ref="formRef"
        :schema="formSchema"
        :submit-button-options="{ loading, text: '保存' }"
        @submit="handleSubmit"
      />
    </div>
  </Page>
</template>
