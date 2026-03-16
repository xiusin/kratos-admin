<script lang="ts" setup>
import { ref, computed } from 'vue';
import { message } from 'ant-design-vue';

/**
 * 文件上传组件
 * 支持拖拽上传、进度显示、格式和大小验证
 */

type FileType = 'IMAGE' | 'VIDEO' | 'ALL';

interface UploadFile {
  id: string;
  file: File;
  name: string;
  size: number;
  type: string;
  progress: number;
  status: 'pending' | 'uploading' | 'success' | 'error';
  error?: string;
  url?: string;
  thumbnailUrl?: string;
}

interface Props {
  // 允许的文件类型
  fileType?: FileType;
  // 最大文件大小（MB）
  maxSize?: number;
  // 最大文件数量
  maxCount?: number;
  // 允许的文件格式
  accept?: string;
  // 是否支持多选
  multiple?: boolean;
  // 是否显示文件列表
  showFileList?: boolean;
}

interface Emits {
  (e: 'upload', file: File): Promise<{ url: string; thumbnailUrl?: string }>;
  (e: 'remove', file: UploadFile): void;
  (e: 'change', files: UploadFile[]): void;
}

const props = withDefaults(defineProps<Props>(), {
  fileType: 'ALL',
  maxSize: 100,
  maxCount: 10,
  multiple: true,
  showFileList: true,
});

const emit = defineEmits<Emits>();

// 上传文件列表
const fileList = ref<UploadFile[]>([]);
// 是否拖拽中
const isDragging = ref(false);
// 文件输入框引用
const fileInputRef = ref<HTMLInputElement>();

// 文件类型配置
const fileTypeConfig = computed(() => {
  const configs = {
    IMAGE: {
      accept: 'image/jpeg,image/png,image/gif',
      formats: ['jpg', 'jpeg', 'png', 'gif'],
      maxSize: 5,
      description: '支持 JPG、PNG、GIF 格式，最大 5MB',
    },
    VIDEO: {
      accept: 'video/mp4,video/avi,video/mov,video/quicktime',
      formats: ['mp4', 'avi', 'mov'],
      maxSize: 100,
      description: '支持 MP4、AVI、MOV 格式，最大 100MB',
    },
    ALL: {
      accept: 'image/*,video/*',
      formats: ['jpg', 'jpeg', 'png', 'gif', 'mp4', 'avi', 'mov'],
      maxSize: 100,
      description: '支持图片和视频，图片最大 5MB，视频最大 100MB',
    },
  };
  
  return props.accept ? { accept: props.accept, formats: [], maxSize: props.maxSize, description: '' } : configs[props.fileType];
});

// 验证文件
function validateFile(file: File): { valid: boolean; error?: string } {
  // 验证文件格式
  const fileExt = file.name.split('.').pop()?.toLowerCase();
  if (fileTypeConfig.value.formats.length > 0 && (!fileExt || !fileTypeConfig.value.formats.includes(fileExt))) {
    return {
      valid: false,
      error: `不支持的文件格式，仅支持：${fileTypeConfig.value.formats.join(', ')}`,
    };
  }
  
  // 验证文件大小
  const sizeMB = file.size / 1024 / 1024;
  const maxSize = props.fileType === 'IMAGE' && file.type.startsWith('image/') 
    ? 5 
    : props.fileType === 'VIDEO' && file.type.startsWith('video/')
    ? 100
    : props.maxSize;
    
  if (sizeMB > maxSize) {
    return {
      valid: false,
      error: `文件大小超过限制（最大 ${maxSize}MB）`,
    };
  }
  
  // 验证文件数量
  if (fileList.value.length >= props.maxCount) {
    return {
      valid: false,
      error: `最多只能上传 ${props.maxCount} 个文件`,
    };
  }
  
  return { valid: true };
}

// 添加文件到列表
function addFile(file: File): UploadFile {
  const uploadFile: UploadFile = {
    id: `${Date.now()}_${Math.random()}`,
    file,
    name: file.name,
    size: file.size,
    type: file.type,
    progress: 0,
    status: 'pending',
  };
  
  fileList.value.push(uploadFile);
  emit('change', fileList.value);
  
  return uploadFile;
}

// 上传文件
async function uploadFile(uploadFile: UploadFile) {
  uploadFile.status = 'uploading';
  
  try {
    // 模拟上传进度
    const progressInterval = setInterval(() => {
      if (uploadFile.progress < 90) {
        uploadFile.progress += 10;
      }
    }, 200);
    
    // 调用父组件的上传方法
    const result = await emit('upload', uploadFile.file);
    
    clearInterval(progressInterval);
    
    uploadFile.progress = 100;
    uploadFile.status = 'success';
    uploadFile.url = result.url;
    uploadFile.thumbnailUrl = result.thumbnailUrl;
    
    message.success(`${uploadFile.name} 上传成功`);
  } catch (error: any) {
    uploadFile.status = 'error';
    uploadFile.error = error.message || '上传失败';
    message.error(`${uploadFile.name} 上传失败：${uploadFile.error}`);
  }
  
  emit('change', fileList.value);
}

// 处理文件选择
function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement;
  const files = Array.from(target.files || []);
  
  handleFiles(files);
  
  // 清空输入框，允许重复选择同一文件
  target.value = '';
}

// 处理文件
function handleFiles(files: File[]) {
  for (const file of files) {
    const validation = validateFile(file);
    
    if (!validation.valid) {
      message.error(validation.error);
      continue;
    }
    
    const uploadFile = addFile(file);
    uploadFile(uploadFile);
  }
}

// 处理拖拽进入
function handleDragEnter(event: DragEvent) {
  event.preventDefault();
  isDragging.value = true;
}

// 处理拖拽离开
function handleDragLeave(event: DragEvent) {
  event.preventDefault();
  isDragging.value = false;
}

// 处理拖拽悬停
function handleDragOver(event: DragEvent) {
  event.preventDefault();
}

// 处理文件拖放
function handleDrop(event: DragEvent) {
  event.preventDefault();
  isDragging.value = false;
  
  const files = Array.from(event.dataTransfer?.files || []);
  handleFiles(files);
}

// 点击上传区域
function handleClick() {
  fileInputRef.value?.click();
}

// 移除文件
function removeFile(uploadFile: UploadFile) {
  const index = fileList.value.findIndex(f => f.id === uploadFile.id);
  if (index !== -1) {
    fileList.value.splice(index, 1);
    emit('remove', uploadFile);
    emit('change', fileList.value);
  }
}

// 清空文件列表
function clear() {
  fileList.value = [];
  emit('change', fileList.value);
}

// 格式化文件大小
function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
}

// 暴露方法给父组件
defineExpose({
  clear,
  fileList,
});
</script>

<template>
  <div class="file-uploader">
    <!-- 上传区域 -->
    <div
      class="upload-area"
      :class="{ dragging: isDragging }"
      @click="handleClick"
      @dragenter="handleDragEnter"
      @dragleave="handleDragLeave"
      @dragover="handleDragOver"
      @drop="handleDrop"
    >
      <input
        ref="fileInputRef"
        type="file"
        :accept="fileTypeConfig.accept"
        :multiple="multiple"
        style="display: none"
        @change="handleFileSelect"
      />
      
      <div class="upload-icon">📁</div>
      <div class="upload-text">
        <p class="upload-title">点击或拖拽文件到此区域上传</p>
        <p class="upload-description">{{ fileTypeConfig.description }}</p>
      </div>
    </div>

    <!-- 文件列表 -->
    <div v-if="showFileList && fileList.length > 0" class="file-list">
      <div
        v-for="file in fileList"
        :key="file.id"
        class="file-item"
        :class="file.status"
      >
        <div class="file-info">
          <div class="file-icon">
            <span v-if="file.type.startsWith('image/')">🖼️</span>
            <span v-else-if="file.type.startsWith('video/')">🎬</span>
            <span v-else>📄</span>
          </div>
          <div class="file-details">
            <div class="file-name">{{ file.name }}</div>
            <div class="file-size">{{ formatFileSize(file.size) }}</div>
          </div>
        </div>
        
        <div class="file-status">
          <div v-if="file.status === 'uploading'" class="progress-bar">
            <div class="progress-fill" :style="{ width: `${file.progress}%` }"></div>
            <span class="progress-text">{{ file.progress }}%</span>
          </div>
          <div v-else-if="file.status === 'success'" class="status-success">✓ 上传成功</div>
          <div v-else-if="file.status === 'error'" class="status-error">✗ {{ file.error }}</div>
        </div>
        
        <button
          type="button"
          class="remove-button"
          @click.stop="removeFile(file)"
        >
          ✕
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.file-uploader {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 上传区域 */
.upload-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 48px 24px;
  border: 2px dashed #d9d9d9;
  border-radius: 8px;
  background: #fafafa;
  cursor: pointer;
  transition: all 0.2s;
}

.upload-area:hover {
  border-color: #1890ff;
  background: #f0f8ff;
}

.upload-area.dragging {
  border-color: #1890ff;
  background: #e6f7ff;
}

.upload-icon {
  font-size: 48px;
}

.upload-text {
  text-align: center;
}

.upload-title {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
  color: #262626;
}

.upload-description {
  margin: 0;
  font-size: 14px;
  color: #8c8c8c;
}

/* 文件列表 */
.file-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  background: #fff;
  transition: all 0.2s;
}

.file-item:hover {
  background: #fafafa;
}

.file-item.success {
  border-color: #52c41a;
  background: #f6ffed;
}

.file-item.error {
  border-color: #ff4d4f;
  background: #fff2f0;
}

.file-info {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 12px;
}

.file-icon {
  font-size: 32px;
}

.file-details {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.file-name {
  font-size: 14px;
  font-weight: 600;
  color: #262626;
  word-break: break-all;
}

.file-size {
  font-size: 12px;
  color: #8c8c8c;
}

.file-status {
  min-width: 120px;
}

.progress-bar {
  position: relative;
  width: 100%;
  height: 20px;
  background: #f0f0f0;
  border-radius: 10px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #1890ff, #40a9ff);
  transition: width 0.3s;
}

.progress-text {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-size: 12px;
  font-weight: 600;
  color: #262626;
}

.status-success {
  font-size: 14px;
  font-weight: 600;
  color: #52c41a;
}

.status-error {
  font-size: 12px;
  color: #ff4d4f;
}

.remove-button {
  padding: 4px 8px;
  font-size: 16px;
  color: #8c8c8c;
  background: transparent;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

.remove-button:hover {
  color: #ff4d4f;
  background: #fff2f0;
}
</style>
