<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { z } from 'zod';
import { $t } from '@vben/locales';
import { Page, VbenButton, VbenForm } from '@vben/common-ui';
import { useConsumerStore } from '#/stores/consumer.state';
import { useSMSStore } from '#/stores/sms.state';
import { message } from 'ant-design-vue';

defineOptions({ name: 'ConsumerRegister' });

const router = useRouter();
const consumerStore = useConsumerStore();
const smsStore = useSMSStore();

const loading = ref(false);
const formRef = ref();

// 表单 Schema
const formSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入手机号',
      },
      fieldName: 'phone',
      label: '手机号',
      rules: z.string()
        .regex(/^1[3-9]\d{9}$/, { message: '请输入正确的手机号' }),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入验证码',
        suffix: () => {
          return (
            <VbenButton
              type="link"
              disabled={!smsStore.canSend}
              onClick={handleSendCode}
            >
              {smsStore.countdownText}
            </VbenButton>
          );
        },
      },
      fieldName: 'verificationCode',
      label: '验证码',
      rules: z.string()
        .length(6, { message: '验证码为6位数字' })
        .regex(/^\d{6}$/, { message: '验证码必须为数字' }),
    },
    {
      component: 'VbenInputPassword',
      componentProps: {
        placeholder: '请输入密码（至少6位）',
      },
      fieldName: 'password',
      label: '密码',
      rules: z.string()
        .min(6, { message: '密码至少6位' }),
    },
    {
      component: 'VbenInputPassword',
      componentProps: {
        placeholder: '请再次输入密码',
      },
      fieldName: 'confirmPassword',
      label: '确认密码',
      rules: z.string()
        .min(6, { message: '密码至少6位' }),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入昵称（可选）',
      },
      fieldName: 'nickname',
      label: '昵称',
      rules: z.string().optional(),
    },
  ];
});

// 发送验证码
async function handleSendCode() {
  const form = formRef.value;
  if (!form) return;

  try {
    // 验证手机号
    const values = await form.validate(['phone']);
    
    await smsStore.sendVerificationCode({
      phone: values.phone,
      smsType: 'VERIFICATION',
    });
    
    message.success('验证码已发送');
  } catch (error: any) {
    message.error(error.message || '发送验证码失败');
  }
}

// 提交注册
async function handleSubmit(values: any) {
  // 验证两次密码是否一致
  if (values.password !== values.confirmPassword) {
    message.error('两次输入的密码不一致');
    return;
  }

  loading.value = true;
  
  try {
    await consumerStore.registerByPhone({
      phone: values.phone,
      password: values.password,
      verificationCode: values.verificationCode,
      nickname: values.nickname,
    });
    
    message.success('注册成功');
    
    // 跳转到登录页
    router.push('/consumer/auth/login');
  } catch (error: any) {
    message.error(error.message || '注册失败');
  } finally {
    loading.value = false;
  }
}

// 跳转到登录页
function goToLogin() {
  router.push('/consumer/auth/login');
}
</script>

<template>
  <Page
    title="用户注册"
    description="创建您的账户"
  >
    <div class="mx-auto max-w-md">
      <VbenForm
        ref="formRef"
        :schema="formSchema"
        :submit-button-options="{ loading, text: '注册' }"
        @submit="handleSubmit"
      />
      
      <div class="mt-4 text-center">
        <span class="text-gray-600">已有账户？</span>
        <VbenButton type="link" @click="goToLogin">
          立即登录
        </VbenButton>
      </div>
    </div>
  </Page>
</template>
