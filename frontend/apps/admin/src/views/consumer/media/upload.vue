<script lang="ts" setup>
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { Page, VbenButton } from '@vben/common-ui';
import { useConsumerStore } from '#/stores/consumer.state';
import { useMediaStore } from '#/stores/media.state';
import { message, Upload } from 'ant-design-vue';
import type { UploadChangeParam, UploadFile } from 'ant-design-vue';

defineOptions({ name: 'ConsumerMediaUpload' });

const router = useRouter();
const consumerStore = useConsumerStore();
const mediaStore = useMediaStore();

const fileList = ref<UploadFile[]>([]);
const uploading = ref(false);

// 上传前验证
function beforeUpload(file: File) {
  // 验证文件类型
  const isImage = file.type.startsWith('image/');
  const isVideo = file.type.startsWith('video/');
  
  if (!isImage && !isVideo) {
    message.error('只能上传图片或视频文件');
    return false;
  }
  
  // 验证文件大小
  const sizeMB = file.size / 1024 / 1024;
  if (isImage && sizeMB > 5) {
    message.error('图片大小不能超过5MB');
    return false;
  }
  if (isVideo && sizeMB > 100) {
    message.error('视频大小不能超过100MB');
    return false;
  }
  
  // 验证文件格式
  const imageFormats = ['image/jpeg', 'image/png', 'image/gif'];
  const videoFormats = ['video/mp4', 'video/avi', 'video/mov', 'video/quicktime'];
  
  if (isImage && !imageFormats.includes(file.type)) {
    message.error('图片格式仅支持 JPEG、PNG、GIF');
    return false;
  }
  if (isVideo && !videoFormats.includes(file.type)) {
    message.error('视频格式仅支持 MP4、AVI、MOV');
    return false;
  }
  
  return true;
}

// 处理上传变化
function handleChange(info: UploadChangeParam) {
  fileList.value = info.fileList;
  
  if (info.file.status === 'done') {
    message.success(`${info.file.name} 上传成功`);
  } else if (info.file.status === 'error') {
    message.error(`${info.file.name} 上传失败`);
  }
}

// 自定义上传
async function customUpload(options: any) {
  const { file, onSuccess, onError, onProgress } = options;
  
  try {
    // 1. 生成预签名URL
    const fileType = file.type.startsWith('image/') ? 'IMAGE' : 'VIDEO';
    const fileFormat = file.type.split('/')[1].toUpperCase();
    
    const uploadUrl = await mediaStore.generateUploadURL({
      fileName: file.name,
      fileType,
      fileFormat,
      fileSize: file.size,
    });
    
    // 2. 上传到OSS
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
          fileType,
          fileFormat,
          fileSize: file.size,
          fileUrl: uploadUrl.url,
        });
        
        onSuccess(mediaFile, file);
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

// 移除文件
function handleRemove(file: UploadFile) {
  const index = fileList.value.indexOf(file);
  if (index > -1) {
    fileList.value.splice(index, 1);
  }
}

// 查看文件列表
function goToList() {
  router.push('/consumer/media/list');
}
</script>

<template>
  <Page
    title="上传媒体文件"
    description="上传图片和视频"
  >
    <div class="mx-auto max-w-2xl">
      <Upload
        v-model:file-list="fileList"
        name="file"
        list-type="picture-card"
        :multiple="true"
        :before-upload="beforeUpload"
        :custom-request="customUpload"
        @change="handleChange"
        @remove="handleRemove"
      >
        <div v-if="fileList.length < 8">
          <div class="text-2xl">+</div>
          <div class="mt-2">上传文件</div>
        </div>
      </Upload>
      
      <div class="mt-6 rounded-lg bg-blue-50 p-4">
        <h4 class="mb-2 font-medium text-blue-900">上传说明</h4>
        <ul class="space-y-1 text-sm text-blue-700">
          <li>• 图片格式：JPEG、PNG、GIF</li>
          <li>• 视频格式：MP4、AVI、MOV</li>
          <li>• 图片大小：最大5MB</li>
          <li>• 视频大小：最大100MB</li>
          <li>• 支持拖拽上传</li>
          <li>• 支持批量上传（最多8个文件）</li>
        </ul>
      </div>
      
      <div class="mt-4 text-center">
        <VbenButton type="primary" @click="goToList">
          查看文件列表
        </VbenButton>
      </div>
    </div>
  </Page>
</template>
