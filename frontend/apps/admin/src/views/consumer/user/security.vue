<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import { computed, ref } from 'vue';
import { z } from 'zod';
import { Page, VbenButton, VbenForm, VbenTabs } from '@vben/common-ui';
import { useConsumerStore } from '#/stores/consumer.state';
import { useSMSStore } from '#/stores/sms.state';
import { message } from 'ant-design-vue';

defineOptions({ name: 'ConsumerSecurity' });

const consumerStore = useConsumerStore();
const smsStore = useSMSStore();

const activeTab = ref('phone');
const loading = ref(false);
const phoneFormRef = ref();
const emailFormRef = ref();
const passwordFormRef = ref();

// 修改手机号表单 Schema
const phoneFormSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入新手机号',
      },
      fieldName: 'newPhone',
      label: '新手机号',
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
              onClick={handleSendPhoneCode}
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
  ];
});

// 修改邮箱表单 Schema
const emailFormSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入新邮箱',
      },
      fieldName: 'newEmail',
      label: '新邮箱',
      rules: z.string()
        .email({ message: '请输入正确的邮箱地址' }),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入验证码',
      },
      fieldName: 'verificationCode',
      label: '验证码',
      rules: z.string()
        .length(6, { message: '验证码为6位数字' })
        .regex(/^\d{6}$/, { message: '验证码必须为数字' }),
    },
  ];
});

// 修改密码表单 Schema
const passwordFormSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInputPassword',
      componentProps: {
        placeholder: '请输入当前密码',
      },
      fieldName: 'oldPassword',
      label: '当前密码',
      rules: z.string()
        .min(1, { message: '请输入当前密码' }),
    },
    {
      component: 'VbenInputPassword',
      componentProps: {
        placeholder: '请输入新密码（至少6位）',
      },
      fieldName: 'newPassword',
      label: '新密码',
      rules: z.string()
        .min(6, { message: '密码至少6位' }),
    },
    {
      component: 'VbenInputPassword',
      componentProps: {
        placeholder: '请再次输入新密码',
      },
      fieldName: 'confirmPassword',
      label: '确认密码',
      rules: z.string()
        .min(6, { message: '密码至少6位' }),
    },
  ];
});

// 发送手机验证码
async function handleSendPhoneCode() {
  const form = phoneFormRef.value;
  if (!form) return;

  try {
    const values = await form.validate(['newPhone']);
    
    await smsStore.sendVerificationCode({
      phone: values.newPhone,
      smsType: 'VERIFICATION',
    });
    
    message.success('验证码已发送');
  } catch (error: any) {
    message.error(error.message || '发送验证码失败');
  }
}

// 提交修改手机号
async function handleUpdatePhone(values: any) {
  loading.value = true;
  
  try {
    // TODO: 调用 ConsumerService.UpdatePhone
    // await consumerServiceClient.updatePhone({
    //   newPhone: values.newPhone,
    //   verificationCode: values.verificationCode,
    // });
    
    message.success('手机号修改成功');
    
    // 刷新用户信息
    if (consumerStore.consumerInfo?.id) {
      await consumerStore.getConsumer(consumerStore.consumerInfo.id);
    }
  } catch (error: any) {
    message.error(error.message || '修改手机号失败');
  } finally {
    loading.value = false;
  }
}

// 提交修改邮箱
async function handleUpdateEmail(values: any) {
  loading.value = true;
  
  try {
    // TODO: 调用 ConsumerService.UpdateEmail
    // await consumerServiceClient.updateEmail({
    //   newEmail: values.newEmail,
    //   verificationCode: values.verificationCode,
    // });
    
    message.success('邮箱修改成功');
    
    // 刷新用户信息
    if (consumerStore.consumerInfo?.id) {
      await consumerStore.getConsumer(consumerStore.consumerInfo.id);
    }
  } catch (error: any) {
    message.error(error.message || '修改邮箱失败');
  } finally {
    loading.value = false;
  }
}

// 提交修改密码
async function handleUpdatePassword(values: any) {
  // 验证两次密码是否一致
  if (values.newPassword !== values.confirmPassword) {
    message.error('两次输入的密码不一致');
    return;
  }

  loading.value = true;
  
  try {
    // TODO: 调用 ConsumerService.UpdatePassword
    // await consumerServiceClient.updatePassword({
    //   oldPassword: values.oldPassword,
    //   newPassword: values.newPassword,
    // });
    
    message.success('密码修改成功，请重新登录');
    
    // 登出
    consumerStore.logout();
    
    // 跳转到登录页
    // router.push('/consumer/auth/login');
  } catch (error: any) {
    message.error(error.message || '修改密码失败');
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <Page
    title="安全设置"
    description="修改手机号、邮箱和密码"
  >
    <div class="mx-auto max-w-2xl">
      <VbenTabs v-model:active-key="activeTab">
        <VbenTabPane key="phone" tab="修改手机号">
          <VbenForm
            ref="phoneFormRef"
            :schema="phoneFormSchema"
            :submit-button-options="{ loading, text: '确认修改' }"
            @submit="handleUpdatePhone"
          />
        </VbenTabPane>
        
        <VbenTabPane key="email" tab="修改邮箱">
          <VbenForm
            ref="emailFormRef"
            :schema="emailFormSchema"
            :submit-button-options="{ loading, text: '确认修改' }"
            @submit="handleUpdateEmail"
          />
        </VbenTabPane>
        
        <VbenTabPane key="password" tab="修改密码">
          <VbenForm
            ref="passwordFormRef"
            :schema="passwordFormSchema"
            :submit-button-options="{ loading, text: '确认修改' }"
            @submit="handleUpdatePassword"
          />
        </VbenTabPane>
      </VbenTabs>
    </div>
  </Page>
</template>
