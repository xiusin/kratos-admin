<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import { z } from 'zod';
import { $t } from '@vben/locales';
import { Page, VbenButton, VbenForm, VbenTabs } from '@vben/common-ui';
import { useConsumerStore } from '#/stores/consumer.state';
import { message } from 'ant-design-vue';

defineOptions({ name: 'ConsumerLogin' });

const router = useRouter();
const consumerStore = useConsumerStore();

const loading = ref(false);
const activeTab = ref('phone');
const formRef = ref();

// 手机号登录表单 Schema
const phoneFormSchema = computed((): VbenFormSchema[] => {
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
      component: 'VbenInputPassword',
      componentProps: {
        placeholder: '请输入密码',
      },
      fieldName: 'password',
      label: '密码',
      rules: z.string()
        .min(1, { message: '请输入密码' }),
    },
  ];
});

// 微信登录表单 Schema
const wechatFormSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入微信授权码',
      },
      fieldName: 'code',
      label: '授权码',
      rules: z.string()
        .min(1, { message: '请输入微信授权码' }),
    },
  ];
});

// 手机号登录
async function handlePhoneLogin(values: any) {
  loading.value = true;
  
  try {
    await consumerStore.loginByPhone({
      phone: values.phone,
      password: values.password,
    });
    
    message.success('登录成功');
    
    // 跳转到首页
    router.push('/consumer/dashboard');
  } catch (error: any) {
    message.error(error.message || '登录失败');
  } finally {
    loading.value = false;
  }
}

// 微信登录
async function handleWechatLogin(values: any) {
  loading.value = true;
  
  try {
    await consumerStore.loginByWechat({
      code: values.code,
    });
    
    message.success('登录成功');
    
    // 跳转到首页
    router.push('/consumer/dashboard');
  } catch (error: any) {
    message.error(error.message || '微信登录失败');
  } finally {
    loading.value = false;
  }
}

// 获取微信授权
function getWechatAuth() {
  // TODO: 调用微信授权接口获取授权URL
  message.info('正在跳转到微信授权页面...');
  
  // 模拟跳转
  // window.location.href = wechatAuthUrl;
}

// 跳转到注册页
function goToRegister() {
  router.push('/consumer/auth/register');
}

// 跳转到忘记密码页
function goToForgetPassword() {
  router.push('/consumer/auth/forget-password');
}
</script>

<template>
  <Page
    title="用户登录"
    description="欢迎回来"
  >
    <div class="mx-auto max-w-md">
      <VbenTabs v-model:active-key="activeTab">
        <VbenTabPane key="phone" tab="手机号登录">
          <VbenForm
            ref="formRef"
            :schema="phoneFormSchema"
            :submit-button-options="{ loading, text: '登录' }"
            @submit="handlePhoneLogin"
          />
          
          <div class="mt-2 text-right">
            <VbenButton type="link" @click="goToForgetPassword">
              忘记密码？
            </VbenButton>
          </div>
        </VbenTabPane>
        
        <VbenTabPane key="wechat" tab="微信登录">
          <div class="mb-4 text-center">
            <VbenButton type="primary" @click="getWechatAuth">
              获取微信授权
            </VbenButton>
          </div>
          
          <VbenForm
            :schema="wechatFormSchema"
            :submit-button-options="{ loading, text: '微信登录' }"
            @submit="handleWechatLogin"
          />
        </VbenTabPane>
      </VbenTabs>
      
      <div class="mt-4 text-center">
        <span class="text-gray-600">还没有账户？</span>
        <VbenButton type="link" @click="goToRegister">
          立即注册
        </VbenButton>
      </div>
    </div>
  </Page>
</template>
