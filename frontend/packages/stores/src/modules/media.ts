import { ref } from 'vue';
import { acceptHMRUpdate, defineStore } from 'pinia';

/**
 * 媒体文件信息接口
 */
export interface MediaFile {
  id?: number;
  tenantId?: number;
  consumerId?: number;
  fileName?: string;
  fileType?: 'IMAGE' | 'VIDEO';
  fileFormat?: string;
  fileSize?: number;
  fileUrl?: string;
  thumbnailUrl?: string;
  ossBucket?: string;
  ossKey?: string;
  isDeleted?: boolean;
  createdAt?: string;
  deletedAt?: string;
}

/**
 * 生成上传URL请求接口
 */
export interface GenerateUploadURLRequest {
  consumerId: number;
  fileName: string;
  fileType: 'IMAGE' | 'VIDEO';
  fileSize: number;
}

/**
 * 生成上传URL响应接口
 */
export interface GenerateUploadURLResponse {
  uploadUrl: string;
  fileKey: string;
  expiresIn: number;
}

/**
 * 确认上传请求接口
 */
export interface ConfirmUploadRequest {
  consumerId: number;
  fileKey: string;
  fileName: string;
  fileType: 'IMAGE' | 'VIDEO';
  fileSize: number;
}

/**
 * 上传进度信息接口
 */
export interface UploadProgress {
  fileKey: string;
  fileName: string;
  progress: number;
  status: 'pending' | 'uploading' | 'success' | 'error';
  error?: string;
}

/**
 * @zh_CN 媒体服务状态管理
 */
export const useMediaStore = defineStore('media', () => {
  // 状态
  const mediaFiles = ref<MediaFile[]>([]);
  const uploadProgresses = ref<Map<string, UploadProgress>>(new Map());
  const loading = ref(false);
  const error = ref<string | null>(null);
  const totalFiles = ref(0);

  /**
   * 生成上传预签名URL
   */
  async function generateUploadURL(request: GenerateUploadURLRequest): Promise<GenerateUploadURLResponse> {
    loading.value = true;
    error.value = null;
    
    try {
      // 验证文件大小
      const maxSize = request.fileType === 'IMAGE' ? 5 * 1024 * 1024 : 100 * 1024 * 1024;
      if (request.fileSize > maxSize) {
        throw new Error(`文件大小超过限制（${request.fileType === 'IMAGE' ? '5MB' : '100MB'}）`);
      }
      
      // 验证文件格式
      const allowedFormats = request.fileType === 'IMAGE' 
        ? ['jpg', 'jpeg', 'png', 'gif']
        : ['mp4', 'avi', 'mov'];
      
      const fileExt = request.fileName.split('.').pop()?.toLowerCase();
      if (!fileExt || !allowedFormats.includes(fileExt)) {
        throw new Error(`不支持的文件格式，仅支持：${allowedFormats.join(', ')}`);
      }
      
      // TODO: 调用 gRPC MediaService.GenerateUploadURL
      // const response = await mediaServiceClient.generateUploadURL(request);
      
      // 模拟响应
      const response: GenerateUploadURLResponse = {
        uploadUrl: `https://oss.example.com/upload/${Date.now()}`,
        fileKey: `media/${Date.now()}_${request.fileName}`,
        expiresIn: 3600,
      };
      
      // 初始化上传进度
      uploadProgresses.value.set(response.fileKey, {
        fileKey: response.fileKey,
        fileName: request.fileName,
        progress: 0,
        status: 'pending',
      });
      
      return response;
    } catch (err: any) {
      error.value = err.message || '生成上传URL失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 上传文件到OSS
   */
  async function uploadFile(
    uploadUrl: string,
    file: File,
    fileKey: string,
    onProgress?: (progress: number) => void
  ): Promise<void> {
    try {
      // 更新上传状态
      const progressInfo = uploadProgresses.value.get(fileKey);
      if (progressInfo) {
        progressInfo.status = 'uploading';
      }
      
      // TODO: 实际上传到OSS
      // 使用 XMLHttpRequest 或 axios 上传，监听进度
      
      // 模拟上传进度
      for (let i = 0; i <= 100; i += 10) {
        await new Promise(resolve => setTimeout(resolve, 100));
        
        if (progressInfo) {
          progressInfo.progress = i;
        }
        
        if (onProgress) {
          onProgress(i);
        }
      }
      
      // 上传成功
      if (progressInfo) {
        progressInfo.status = 'success';
        progressInfo.progress = 100;
      }
    } catch (err: any) {
      const progressInfo = uploadProgresses.value.get(fileKey);
      if (progressInfo) {
        progressInfo.status = 'error';
        progressInfo.error = err.message || '上传失败';
      }
      throw err;
    }
  }

  /**
   * 确认上传完成
   */
  async function confirmUpload(request: ConfirmUploadRequest): Promise<MediaFile> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC MediaService.ConfirmUpload
      // const response = await mediaServiceClient.confirmUpload(request);
      
      // 模拟响应
      const mediaFile: MediaFile = {
        id: Date.now(),
        consumerId: request.consumerId,
        fileName: request.fileName,
        fileType: request.fileType,
        fileFormat: request.fileName.split('.').pop(),
        fileSize: request.fileSize,
        fileUrl: `https://cdn.example.com/${request.fileKey}`,
        thumbnailUrl: request.fileType === 'IMAGE' 
          ? `https://cdn.example.com/${request.fileKey}_thumb.jpg`
          : undefined,
        ossKey: request.fileKey,
        isDeleted: false,
        createdAt: new Date().toISOString(),
      };
      
      // 添加到列表
      mediaFiles.value.unshift(mediaFile);
      totalFiles.value++;
      
      // 清除上传进度
      uploadProgresses.value.delete(request.fileKey);
      
      return mediaFile;
    } catch (err: any) {
      error.value = err.message || '确认上传失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取媒体文件
   */
  async function getMediaFile(id: number): Promise<MediaFile> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC MediaService.GetMediaFile
      // const response = await mediaServiceClient.getMediaFile({ id });
      
      // 模拟响应
      const mediaFile: MediaFile = {
        id,
        fileName: 'example.jpg',
        fileType: 'IMAGE',
        fileUrl: 'https://cdn.example.com/example.jpg',
        createdAt: new Date().toISOString(),
      };
      
      return mediaFile;
    } catch (err: any) {
      error.value = err.message || '获取媒体文件失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 查询媒体文件列表
   */
  async function listMediaFiles(
    consumerId: number,
    fileType?: 'IMAGE' | 'VIDEO',
    page: number = 1,
    pageSize: number = 20
  ): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC MediaService.ListMediaFiles
      // const response = await mediaServiceClient.listMediaFiles({ consumerId, fileType, page, pageSize });
      
      // 模拟响应
      const mockFiles: MediaFile[] = [
        {
          id: 1,
          consumerId,
          fileName: 'photo1.jpg',
          fileType: 'IMAGE',
          fileFormat: 'jpg',
          fileSize: 1024000,
          fileUrl: 'https://cdn.example.com/photo1.jpg',
          thumbnailUrl: 'https://cdn.example.com/photo1_thumb.jpg',
          createdAt: new Date().toISOString(),
        },
        {
          id: 2,
          consumerId,
          fileName: 'video1.mp4',
          fileType: 'VIDEO',
          fileFormat: 'mp4',
          fileSize: 10240000,
          fileUrl: 'https://cdn.example.com/video1.mp4',
          createdAt: new Date(Date.now() - 86400000).toISOString(),
        },
      ];
      
      mediaFiles.value = mockFiles;
      totalFiles.value = mockFiles.length;
    } catch (err: any) {
      error.value = err.message || '查询媒体文件列表失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 删除媒体文件
   */
  async function deleteMediaFile(id: number): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      // TODO: 调用 gRPC MediaService.DeleteMediaFile
      // await mediaServiceClient.deleteMediaFile({ id });
      
      // 从列表中移除
      const index = mediaFiles.value.findIndex(f => f.id === id);
      if (index !== -1) {
        mediaFiles.value.splice(index, 1);
        totalFiles.value--;
      }
    } catch (err: any) {
      error.value = err.message || '删除媒体文件失败';
      throw err;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取上传进度
   */
  function getUploadProgress(fileKey: string): UploadProgress | undefined {
    return uploadProgresses.value.get(fileKey);
  }

  /**
   * 清除上传进度
   */
  function clearUploadProgress(fileKey: string): void {
    uploadProgresses.value.delete(fileKey);
  }

  /**
   * 重置状态
   */
  function $reset() {
    mediaFiles.value = [];
    uploadProgresses.value.clear();
    loading.value = false;
    error.value = null;
    totalFiles.value = 0;
  }

  return {
    // 状态
    mediaFiles,
    uploadProgresses,
    loading,
    error,
    totalFiles,
    
    // 方法
    generateUploadURL,
    uploadFile,
    confirmUpload,
    getMediaFile,
    listMediaFiles,
    deleteMediaFile,
    getUploadProgress,
    clearUploadProgress,
    $reset,
  };
});

// 解决热更新问题
const hot = import.meta.hot;
if (hot) {
  hot.accept(acceptHMRUpdate(useMediaStore, hot));
}
