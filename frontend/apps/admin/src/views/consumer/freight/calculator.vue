<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';
import { computed, ref } from 'vue';
import { z } from 'zod';
import { Page, VbenForm } from '@vben/common-ui';
import { useFreightStore } from '#/stores/freight.state';
import { Card, Statistic, message } from 'ant-design-vue';

defineOptions({ name: 'ConsumerFreightCalculator' });

const freightStore = useFreightStore();

const loading = ref(false);
const freightResult = ref<any>(null);

// 表单 Schema
const formSchema = computed((): VbenFormSchema[] => {
  return [
    {
      component: 'Select',
      componentProps: {
        options: [
          { label: '按重量计算', value: 'BY_WEIGHT' },
          { label: '按距离计算', value: 'BY_DISTANCE' },
        ],
        placeholder: '请选择计算方式',
      },
      fieldName: 'calculationType',
      label: '计算方式',
      rules: z.string().min(1, { message: '请选择计算方式' }),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入重量（kg）',
        type: 'number',
        min: 0.01,
        step: 0.01,
      },
      fieldName: 'weight',
      label: '重量',
      rules: z.string()
        .refine((val) => parseFloat(val) > 0, { message: '重量必须大于0' }),
      dependencies: {
        show: (values) => values.calculationType === 'BY_WEIGHT',
        triggerFields: ['calculationType'],
      },
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入发货省份',
      },
      fieldName: 'fromProvince',
      label: '发货省份',
      rules: z.string().min(1, { message: '请输入发货省份' }),
      dependencies: {
        show: (values) => values.calculationType === 'BY_DISTANCE',
        triggerFields: ['calculationType'],
      },
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入发货城市',
      },
      fieldName: 'fromCity',
      label: '发货城市',
      rules: z.string().min(1, { message: '请输入发货城市' }),
      dependencies: {
        show: (values) => values.calculationType === 'BY_DISTANCE',
        triggerFields: ['calculationType'],
      },
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入收货省份',
      },
      fieldName: 'toProvince',
      label: '收货省份',
      rules: z.string().min(1, { message: '请输入收货省份' }),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入收货城市',
      },
      fieldName: 'toCity',
      label: '收货城市',
      rules: z.string().min(1, { message: '请输入收货城市' }),
    },
    {
      component: 'VbenInput',
      componentProps: {
        placeholder: '请输入订单金额（用于判断包邮）',
        type: 'number',
        min: 0,
        step: 0.01,
      },
      fieldName: 'orderAmount',
      label: '订单金额',
      rules: z.string().optional(),
    },
  ];
});

// 提交计算
async function handleSubmit(values: any) {
  loading.value = true;
  
  try {
    const result = await freightStore.calculateFreight({
      calculationType: values.calculationType,
      weight: values.weight ? parseFloat(values.weight) : undefined,
      fromProvince: values.fromProvince,
      fromCity: values.fromCity,
      toProvince: values.toProvince,
      toCity: values.toCity,
      orderAmount: values.orderAmount ? parseFloat(values.orderAmount) : undefined,
    });
    
    freightResult.value = result;
    message.success('计算成功');
  } catch (error: any) {
    message.error(error.message || '计算失败');
    freightResult.value = null;
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <Page
    title="运费计算器"
    description="计算订单运费"
  >
    <div class="mx-auto max-w-2xl">
      <Card class="mb-6">
        <VbenForm
          :schema="formSchema"
          :submit-button-options="{ loading, text: '计算运费' }"
          @submit="handleSubmit"
        />
      </Card>
      
      <Card v-if="freightResult" title="计算结果">
        <div class="space-y-4">
          <Statistic
            title="运费"
            :value="freightResult.freight"
            prefix="¥"
            :precision="2"
            class="text-center"
          />
          
          <div v-if="freightResult.isFreeShipping" class="rounded-lg bg-green-50 p-4 text-center">
            <div class="text-lg font-medium text-green-600">🎉 满足包邮条件</div>
            <div class="mt-1 text-sm text-green-700">{{ freightResult.freeShippingReason }}</div>
          </div>
          
          <div class="space-y-2 border-t pt-4">
            <div class="flex justify-between text-sm">
              <span class="text-gray-600">计算方式:</span>
              <span>{{ freightResult.calculationType === 'BY_WEIGHT' ? '按重量' : '按距离' }}</span>
            </div>
            <div v-if="freightResult.weight" class="flex justify-between text-sm">
              <span class="text-gray-600">重量:</span>
              <span>{{ freightResult.weight }} kg</span>
            </div>
            <div v-if="freightResult.distance" class="flex justify-between text-sm">
              <span class="text-gray-600">距离:</span>
              <span>{{ freightResult.distance }} km</span>
            </div>
            <div class="flex justify-between text-sm">
              <span class="text-gray-600">模板名称:</span>
              <span>{{ freightResult.templateName || '-' }}</span>
            </div>
          </div>
        </div>
      </Card>
      
      <div v-if="!freightResult" class="rounded-lg bg-blue-50 p-4">
        <h4 class="mb-2 font-medium text-blue-900">计算说明</h4>
        <ul class="space-y-1 text-sm text-blue-700">
          <li>• 按重量计算：首重+续重阶梯定价</li>
          <li>• 按距离计算：根据省市区计算距离</li>
          <li>• 支持包邮规则（满额包邮、地区包邮）</li>
          <li>• 偏远地区可能有额外加价</li>
          <li>• 运费精确到分</li>
        </ul>
      </div>
    </div>
  </Page>
</template>
