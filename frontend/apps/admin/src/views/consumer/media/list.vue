<script lang="ts" setup>
import type { VxeGridProps } from '#/adapter/vxe-table';
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Page, type VbenFormProps, VbenButton } from '@vben/common-ui';
import { useVbenVxeGrid } from '#/adapter/vxe-table';
import { useConsumerStore } from '#/stores/consumer.state';
import { useMediaStore, type MediaFile } from '#/stores/media.state';
import { Tag, message, Modal, Image } from 'ant-design-vue';

defineOptions({ name: 'ConsumerMediaList' });

const router = useRouter();
const consumerStore = useConsumerStore();
const mediaStore = useMediaStore();

const previewVisible = ref(false);
const previewUrl = ref('');

// 表单配置
const formOptions: VbenFormProps = {
  collapsed: false,
  showCollapseButton: false,
  submitOnEnter: true,
  schema: [
    {
      component: 'Select',
      fieldName: 'fileType',
      label: '文件类型',
      componentProps: {
        options: [
          { label: '图片', value: 'IMAGE' },
          { label: '视频', value: 'VIDEO' },
        ],
        placeholder: '请选择文件类型',
        allowClear: true,
      },
    },
    {
      component: 'Input',
      fieldName: 'fileName',
      label: '文件名',
      componentProps: {
        placeholder: '请输入文件名',
        allowClear: true,
      },
    },
  ],
};

// 表格配置
const gridOptions: VxeGridProps<MediaFile> = {
  toolbarConfig: {
    custom: true,
    refresh: true,
    zoom: true,
    slots: {
      buttons: 'toolbar_buttons',
    },
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
        if (!consumerStore.consumerInfo?.id) {
          return { page: { total: 0 }, result: [] };
        }

        await mediaStore.listMediaFiles({
          consumerId: consumerStore.consumerInfo.id,
          fileType: formValues.fileType,
          fileName: formValues.fileName,
          page: page.currentPage,
          pageSize: page.pageSize,
        });

        return {
          page: {
            total: mediaStore.totalFiles,
          },
          result: mediaStore.files,
        };
      },
    },
  },

  columns: [
    {
      title: '预览',
      field: 'thumbnailUrl',
      width: 100,
      slots: { default: 'preview' },
    },
    {
      title: '文件名',
      field: 'fileName',
      minWidth: 200,
    },
    {
      title: '文件类型',
      field: 'fileType',
      width: 100,
      slots: { default: 'fileType' },
    },
    {
      title: '文件格式',
      field: 'fileFormat',
      width: 100,
    },
    {
      title: '文件大小',
      field: 'fileSize',
      width: 120,
      slots: { default: 'fileSize' },
    },
    {
      title: '上传时间',
      field: 'createdAt',
      formatter: 'formatDateTime',
      width: 160,
    },
    {
      title: '操作',
      width: 150,
      slots: { default: 'action' },
    },
  ],
};

const [Grid, gridApi] = useVbenVxeGrid({ gridOptions, formOptions });

// 加载数据
onMounted(async () => {
  if (consumerStore.consumerInfo?.id) {
    try {
      await mediaStore.listMediaFiles({
        consumerId: consumerStore.consumerInfo.id,
        page: 1,
        pageSize: 10,
      });
    } catch (error: any) {
      message.error(error.message || '加载文件列表失败');
    }
  }
});

// 获取文件类型颜色
function getFileTypeColor(type?: string): string {
  const colorMap: Record<string, string> = {
    IMAGE: 'blue',
    VIDEO: 'purple',
  };
  return colorMap[type || 'IMAGE'] || 'default';
}

// 获取文件类型文本
function getFileTypeText(type?: string): string {
  const textMap: Record<string, string> = {
    IMAGE: '图片',
    VIDEO: '视频',
  };
  return textMap[type || 'IMAGE'] || '未知';
}

// 格式化文件大小
function formatFileSize(size?: number): string {
  if (!size) return '-';
  
  if (size < 1024) {
    return `${size} B`;
  } else if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(2)} KB`;
  } else {
    return `${(size / 1024 / 1024).toFixed(2)} MB`;
  }
}

// 预览文件
function handlePreview(row: MediaFile) {
  if (row.fileType === 'IMAGE') {
    previewUrl.value = row.fileUrl || '';
    previewVisible.value = true;
  } else {
    // 视频在新窗口打开
    window.open(row.fileUrl, '_blank');
  }
}

// 删除文件
async function handleDelete(row: MediaFile) {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除文件 "${row.fileName}" 吗？`,
    onOk: async () => {
      try {
        await mediaStore.deleteMediaFile({ id: row.id! });
        message.success('删除成功');
        
        // 刷新列表
        gridApi.value?.reload();
      } catch (error: any) {
        message.error(error.message || '删除失败');
      }
    },
  });
}

// 跳转到上传页面
function goToUpload() {
  router.push('/consumer/media/upload');
}
</script>

<template>
  <Page auto-content-height>
    <Grid table-title="媒体文件列表">
      <template #toolbar_buttons>
        <VbenButton type="primary" @click="goToUpload">
          上传文件
        </VbenButton>
      </template>
      
      <template #preview="{ row }">
        <div
          v-if="row.thumbnailUrl || row.fileUrl"
          class="cursor-pointer"
          @click="handlePreview(row)"
        >
          <img
            :src="row.thumbnailUrl || row.fileUrl"
            :alt="row.fileName"
            class="h-12 w-12 rounded object-cover"
          />
        </div>
        <div v-else class="flex h-12 w-12 items-center justify-center rounded bg-gray-200">
          <span class="text-gray-400">-</span>
        </div>
      </template>
      
      <template #fileType="{ row }">
        <Tag :color="getFileTypeColor(row.fileType)">
          {{ getFileTypeText(row.fileType) }}
        </Tag>
      </template>
      
      <template #fileSize="{ row }">
        {{ formatFileSize(row.fileSize) }}
      </template>
      
      <template #action="{ row }">
        <div class="flex gap-2">
          <VbenButton size="small" @click="handlePreview(row)">
            预览
          </VbenButton>
          <VbenButton size="small" danger @click="handleDelete(row)">
            删除
          </VbenButton>
        </div>
      </template>
    </Grid>
    
    <!-- 图片预览 -->
    <Image
      :preview="{
        visible: previewVisible,
        onVisibleChange: (visible) => { previewVisible = visible; },
      }"
      :src="previewUrl"
      style="display: none"
    />
  </Page>
</template>
