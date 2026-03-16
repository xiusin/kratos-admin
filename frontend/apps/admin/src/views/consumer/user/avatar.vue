<script lang="ts" setup>
import { ref } from 'vue';
import { Page, VbenButton } from '@vben/common-ui';
import { useConsumerStore } from '#/stores/consumer.state';
import { useMediaStore } from '#/stores/media.state';
import { message, Upload } from 'ant-design-vue';
import type { UploadChangeParam } from 'ant-design-vue';

defineOptions({ name: 'ConsumerAvatar' });

const consumerStore = useConsumerStore();
const mediaStore = useMediaStore();

const uploading = ref(false);
const avatarUrl = ref(consumerStore.consumerInfo?.avatar || '');

// 上传前验证
function beforeUpload(file: File) {
  const isImage = file.type.startsWith('image/');
  if (!isImage) {
    message.error('只能上传图片文件');
    return false;
  }
  
  const isLt5M = file.size / 1024 / 1024 < 5;
  if (!isLt5M) {
    message.error('图片大小不能超过5MB');
    return false;
  }
  
  return true;
}

// 处理上传
async function handleUpload(info: UploadChangeParam) {
  if (info.file.status === 'uploading') {
    uploading.value = true;
    return;
  }
  
  if (info.file.status === 'done') {
    uploading.value = false;
    
    try {
      // 获取上传后的URL
      const url = info.file.response?.url || '';
      avatarUrl.value = url;
      
      // 更新用户头像
      if (consumerStore.consumerInfo?.id) {
        await consumerStore.updateConsumer({
          id: consumerStore.consumerInfo.id,
          avatar: url,
        });
        
        message.success('头像上传成功');
      }
    } catch (error: any) {
      message.error(error.message || '更新头像失败');
    }
  }
  
  if (info.file.status === 'error') {
    uploading.value = false;
    message.error('上传失败');
  }
}

// 自定义上传
async function customUpload(options: any) {
  const { file, onSuccess, onError, onProgress } = options;
  
  try {
    // 1. 生成预签名URL
    const uploadUrl = await mediaStore.generateUploadURL({
      fileName: file.name,
      fileType: 'IMAGE',
      fileFormat: file.type.split('/')[1].toUpperCase(),
      fileSize: file.size,
    });
    
    // 2. 上传到OSS
    const formData = new FormData();
    formData.append('file', file);
    
    const xhr = new XMLHttpRequest();
    
    xhr.upload.addEventListener('progress', (e) => {
      if (e.lengthComputable) {
        const percent = Math.round((e.loaded / e.total) * 100);
        onProgress({ percent });
      }
    });
    
    xhr.addEventListener('load', async () => {
      if (xhr.status === 200) {
        // 3. 确认上传
        const mediaFile = await mediaStore.confirmUpload({
          fileName: file.name,
          fileType: 'IMAGE',
          fileFormat: file.type.split('/')[1].toUpperCase(),
          fileSize: file.size,
          fileUrl: uploadUrl.url,
        });
        
        onSuccess({ url: mediaFile.fileUrl }, file);
      } else {
        onError(new Error('上传失败'));
      }
    });
    
    xhr.addEventListener('error', () => {
      onError(new Error('上传失败'));
    });
    
    xhr.open('PUT', uploadUrl.url);
    xhr.send(file);
  } catch (error: any) {
    onError(error);
  }
}
</script>

<template>
  <Page
    title="上传头像"
    description="上传您的个人头像"
  >
    <div class="mx-auto max-w-md text-center">
      <div class="mb-6">
        <img
          v-if="avatarUrl"
          :src="avatarUrl"
          alt="头像"
          class="mx-auto h-32 w-32 rounded-full object-cover"
        />
        <div
          v-else
          class="mx-auto flex h-32 w-32 items-center justify-center rounded-full bg-gray-200"
        >
          <span class="text-4xl text-gray-400">?</span>
        </div>
      </div>
      
      <Upload
        name="avatar"
        list-type="picture-card"
        :show-upload-list="false"
        :before-upload="beforeUpload"
        :custom-request="customUpload"
        @change="handleUpload"
      >
        <VbenButton :loading="uploading" type="primary">
          {{ uploading ? '上传中...' : '选择图片' }}
        </VbenButton>
      </Upload>
      
      <div class="mt-4 text-sm text-gray-500">
        <p>支持 JPG、PNG、GIF 格式</p>
        <p>文件大小不超过 5MB</p>
      </div>
    </div>
  </Page>
</template>
