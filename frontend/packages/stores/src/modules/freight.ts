import { ref } from 'vue';
import { acceptHMRUpdate, defineStore } from 'pinia';

/**
 * 运费模板信息接口
 */
export interface FreightTemplate {
  id?: number;
  tenantId?: number;
  name?: string;
  calculationType?: 'BY_WEIGHT' | 'BY_DISTANCE';
  firstWeight?: string;
  firstPrice?: string;
  additionalWeight?: string;
  additionalPrice?: string;
  regionRules?: RegionRule[];
  freeShippingRules?: FreeShippingRule[];
  isActive?: boolean;
  createdBy?: number;
  updatedBy?: number;
  createdAt?: string;
  updatedAt?: string;
}

/**
 * 地区规则接口
 */
export interface RegionRule {
  provinces: string[];
  firstWeight?: string;
  firstPrice?: string;
  additionalWeight?: string;
  additionalPrice?: string;
  extraFee?: string;
}

/**
 * 包邮规则接口
 */
export interface FreeShippingRule {
  type: 'AMOUNT' | 'REGION';
  minAmount?: string;
  provinces?: string[];
}

/**
 * 计算运费请求接口
 */
export interface CalculateFreightRequest {
  templateId?: number;
  weight?: number;
  fromProvince?: string;
  fromCity?: string;
  fromDistrict?: string;
  toProvince: string;
  toCity: string;
  toDistrict: string;
  orderAmount?: string;
}

/**
 * 计算运费响应接口
 */
export interface CalculateFreightResponse {
  freight: string;
  isFreeShipping: boolean;
  calculationDetail?: string;
}

/**
 * 创建运费模板请求接口
 */
export interface CreateFreightTemplateRequest {
  name: string;
  calculationType: 'BY_WEIGHT' | 'BY_DISTANCE';
  firstWeight?: string;
  firstPrice?: string;
  additionalWeight?: string;
  additionalPrice?: string;
  regionRules?: RegionRule[];
  freeShippingRules?: FreeShippingRule[];
}

/**
 * @zh_CN 运费计算服务状态管理
 */
export const useFreightStore = defineStore('freight', () => {
  // 状态
  const templates = ref<FreightTemplate[]>([]);
  const currentTemplate = ref<FreightTemplate | null>(null);
  const calculatedFreight = ref<CalculateFreightResponse | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);
  const totalTemplates = ref(0);

  /**
   * 计算运费
   */
  async function calculateFreight(request: CalculateFreightRequest): Promise<CalculateFreightResponse> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC FreightService.CalculateFreight
      // const response = await freightServiceClient.calculateFreight(request);
      
      // 模拟计算逻辑
      let freight = 0;
      let isFreeShipping = false;
      let calculationDetail = '';
      
      // 检查包邮规则
      if (request.orderAmount && parseFloat(request.orderAmount) >= 99) {
        isFreeShipping = true;
        calculationDetail = '满99元包邮';
      } else if (request.weight) {
        // 按重量计算
        const firstWeight = 1; // kg
        const firstPrice = 10; // 元
        const additionalWeight = 1; // kg
        const additionalPrice = 5; // 元
        
        if (request.weight <= firstWeight) {
          freight = firstPrice;
          calculationDetail = `首重${firstWeight}kg，运费${firstPrice}元`;
        } else {
          const extraWeight = Math.ceil(request.weight - firstWeight);
          freight = firstPrice + extraWeight * additionalPrice;
          calculationDetail = `首重${firstWeight}kg ${firstPrice}元 + 续重${extraWeight}kg × ${additionalPrice}元`;
        }
      }
      
      const response: CalculateFreightResponse = {
        freight: freight.toFixed(2),
        isFreeShipping,
        calculationDetail,
      };
      
      calculatedFreight.value = response;
      return response;
    } catch (err: any) {
      error.value = err.message || '计算运费失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 创建运费模板
   */
  async function createFreightTemplate(request: CreateFreightTemplateRequest): Promise<FreightTemplate> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC FreightService.CreateFreightTemplate
      // const response = await freightServiceClient.createFreightTemplate(request);
      
      // 模拟响应
      const template: FreightTemplate = {
        id: Date.now(),
        name: request.name,
        calculationType: request.calculationType,
        firstWeight: request.firstWeight,
        firstPrice: request.firstPrice,
        additionalWeight: request.additionalWeight,
        additionalPrice: request.additionalPrice,
        regionRules: request.regionRules,
        freeShippingRules: request.freeShippingRules,
        isActive: true,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };
      
      templates.value.unshift(template);
      totalTemplates.value++;
      
      return template;
    } catch (err: any) {
      error.value = err.message || '创建运费模板失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 更新运费模板
   */
  async function updateFreightTemplate(
    id: number,
    updates: Partial<CreateFreightTemplateRequest>
  ): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC FreightService.UpdateFreightTemplate
      // await freightServiceClient.updateFreightTemplate({ id, ...updates });
      
      // 更新本地状态
      const index = templates.value.findIndex(t => t.id === id);
      if (index !== -1) {
        templates.value[index] = {
          ...templates.value[index],
          ...updates,
          updatedAt: new Date().toISOString(),
        };
      }
      
      if (currentTemplate.value?.id === id) {
        currentTemplate.value = {
          ...currentTemplate.value,
          ...updates,
          updatedAt: new Date().toISOString(),
        };
      }
    } catch (err: any) {
      error.value = err.message || '更新运费模板失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取运费模板
   */
  async function getFreightTemplate(id: number): Promise<FreightTemplate> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC FreightService.GetFreightTemplate
      // const response = await freightServiceClient.getFreightTemplate({ id });
      
      // 模拟响应
      const template: FreightTemplate = {
        id,
        name: '标准运费模板',
        calculationType: 'BY_WEIGHT',
        firstWeight: '1.0',
        firstPrice: '10.00',
        additionalWeight: '1.0',
        additionalPrice: '5.00',
        freeShippingRules: [
          {
            type: 'AMOUNT',
            minAmount: '99.00',
          },
        ],
        isActive: true,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };
      
      currentTemplate.value = template;
      return template;
    } catch (err: any) {
      error.value = err.message || '获取运费模板失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 查询运费模板列表
   */
  async function listFreightTemplates(
    page: number = 1,
    pageSize: number = 20
  ): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC FreightService.ListFreightTemplates
      // const response = await freightServiceClient.listFreightTemplates({ page, pageSize });
      
      // 模拟响应
      const mockTemplates: FreightTemplate[] = [
        {
          id: 1,
          name: '标准运费模板',
          calculationType: 'BY_WEIGHT',
          firstWeight: '1.0',
          firstPrice: '10.00',
          additionalWeight: '1.0',
          additionalPrice: '5.00',
          isActive: true,
          createdAt: new Date().toISOString(),
        },
        {
          id: 2,
          name: '偏远地区运费模板',
          calculationType: 'BY_WEIGHT',
          firstWeight: '1.0',
          firstPrice: '15.00',
          additionalWeight: '1.0',
          additionalPrice: '8.00',
          isActive: true,
          createdAt: new Date(Date.now() - 86400000).toISOString(),
        },
      ];
      
      templates.value = mockTemplates;
      totalTemplates.value = mockTemplates.length;
    } catch (err: any) {
      error.value = err.message || '查询运费模板列表失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 删除运费模板
   */
  async function deleteFreightTemplate(id: number): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC FreightService.DeleteFreightTemplate
      // await freightServiceClient.deleteFreightTemplate({ id });
      
      // 从列表中移除
      const index = templates.value.findIndex(t => t.id === id);
      if (index !== -1) {
        templates.value.splice(index, 1);
        totalTemplates.value--;
      }
      
      if (currentTemplate.value?.id === id) {
        currentTemplate.value = null;
      }
    } catch (err: any) {
      error.value = err.message || '删除运费模板失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 激活/停用运费模板
   */
  async function toggleTemplateStatus(id: number, isActive: boolean): Promise<void> {
    await updateFreightTemplate(id, { isActive } as any);
  }

  /**
   * 获取计算类型文本
   */
  function getCalculationTypeText(type?: string): string {
    const typeMap: Record<string, string> = {
      BY_WEIGHT: '按重量',
      BY_DISTANCE: '按距离',
    };
    return typeMap[type || ''] || '未知';
  }

  /**
   * 重置状态
   */
  function $reset() {
    templates.value = [];
    currentTemplate.value = null;
    calculatedFreight.value = null;
    loading.value = false;
    error.value = null;
    totalTemplates.value = 0;
  }

  return {
    // 状态
    templates,
    currentTemplate,
    calculatedFreight,
    loading,
    error,
    totalTemplates,
    
    // 方法
    calculateFreight,
    createFreightTemplate,
    updateFreightTemplate,
    getFreightTemplate,
    listFreightTemplates,
    deleteFreightTemplate,
    toggleTemplateStatus,
    getCalculationTypeText,
    $reset,
  };
});

// 解决热更新问题
const hot = import.meta.hot;
if (hot) {
  hot.accept(acceptHMRUpdate(useFreightStore, hot));
}
