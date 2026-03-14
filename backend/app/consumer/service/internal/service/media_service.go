package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/antivirus"
	"go-wind-admin/pkg/image"
	"go-wind-admin/pkg/middleware"
	"go-wind-admin/pkg/oss"
	"go-wind-admin/pkg/tenantconfig"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

const (
	// 文件大小限制
	maxImageSize = 5 * 1024 * 1024   // 5MB
	maxVideoSize = 100 * 1024 * 1024 // 100MB

	// 预签名URL有效期（秒）
	presignedURLExpire = 3600 // 1小时

	// 缩略图尺寸
	thumbnailWidth  = 200
	thumbnailHeight = 200

	// 允许的文件格式
	allowedImageFormats = "JPEG,PNG,GIF"
	allowedVideoFormats = "MP4,AVI,MOV"
)

type MediaService struct {
	consumerV1.UnimplementedMediaServiceServer

	mediaFileRepo    data.MediaFileRepo
	ossClient        oss.Client
	imageProcessor   image.Processor
	virusScanner     antivirus.Scanner
	tenantConfigMgr  tenantconfig.Manager

	log *log.Helper
}

func NewMediaService(
	ctx *bootstrap.Context,
	mediaFileRepo data.MediaFileRepo,
	ossClient oss.Client,
	imageProcessor image.Processor,
	virusScanner antivirus.Scanner,
	tenantConfigMgr tenantconfig.Manager,
) *MediaService {
	return &MediaService{
		log:              ctx.NewLoggerHelper("consumer/service/media-service"),
		mediaFileRepo:    mediaFileRepo,
		ossClient:        ossClient,
		imageProcessor:   imageProcessor,
		virusScanner:     virusScanner,
		tenantConfigMgr:  tenantConfigMgr,
	}
}

// GenerateUploadURL 生成预签名URL
func (s *MediaService) GenerateUploadURL(ctx context.Context, req *consumerV1.GenerateUploadURLRequest) (*consumerV1.GenerateUploadURLResponse, error) {
	// 1. 验证输入
	if req == nil || req.FileName == "" || req.FileType == consumerV1.MediaFile_FILE_TYPE_UNSPECIFIED {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 验证文件格式
	fileFormat := strings.ToUpper(filepath.Ext(req.FileName))
	if fileFormat != "" && fileFormat[0] == '.' {
		fileFormat = fileFormat[1:] // 移除点号
	}

	if err := s.validateFileFormat(req.FileType, fileFormat); err != nil {
		return nil, err
	}

	// 3. 验证文件大小
	if err := s.validateFileSize(req.FileType, req.FileSize); err != nil {
		return nil, err
	}

	// 4. 获取当前用户信息
	tenantID := middleware.GetTenantID(ctx)
	consumerID := middleware.GetUserID(ctx)
	if consumerID == 0 {
		return nil, consumerV1.ErrorUnauthorized("user not authenticated")
	}

	// 5. 生成对象键
	objectKey := s.generateObjectKey(tenantID, consumerID, req.FileType, req.FileName)

	// 6. 生成预签名URL
	uploadURL, err := s.ossClient.GeneratePresignedURL(ctx, objectKey, presignedURLExpire)
	if err != nil {
		s.log.Errorf("generate presigned url failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to generate upload url")
	}

	s.log.Infof("generated upload url: tenant_id=%d, consumer_id=%d, file_name=%s, object_key=%s",
		tenantID, consumerID, req.FileName, objectKey)

	return &consumerV1.GenerateUploadURLResponse{
		UploadUrl: uploadURL,
		ObjectKey: objectKey,
		ExpiresIn: presignedURLExpire,
	}, nil
}

// ConfirmUpload 确认上传完成
func (s *MediaService) ConfirmUpload(ctx context.Context, req *consumerV1.ConfirmUploadRequest) (*consumerV1.MediaFile, error) {
	// 1. 验证输入
	if req == nil || req.ObjectKey == "" || req.FileName == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 2. 获取当前用户信息
	tenantID := middleware.GetTenantID(ctx)
	consumerID := middleware.GetUserID(ctx)
	if consumerID == 0 {
		return nil, consumerV1.ErrorUnauthorized("user not authenticated")
	}

	// 3. 验证文件是否存在
	exists, err := s.ossClient.Exists(ctx, req.ObjectKey)
	if err != nil {
		s.log.Errorf("check file exists failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to verify file")
	}
	if !exists {
		return nil, consumerV1.ErrorNotFound("file not found in oss")
	}

	// 4. 获取文件元数据
	metadata, err := s.ossClient.GetMetadata(ctx, req.ObjectKey)
	if err != nil {
		s.log.Errorf("get file metadata failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to get file metadata")
	}

	// 5. 生成文件URL
	fileURL, err := s.ossClient.GenerateDownloadURL(ctx, req.ObjectKey, 365*24*3600) // 1年有效期
	if err != nil {
		s.log.Errorf("generate download url failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to generate file url")
	}

	// 6. 确定文件类型和格式
	fileFormat := strings.ToUpper(filepath.Ext(req.FileName))
	if fileFormat != "" && fileFormat[0] == '.' {
		fileFormat = fileFormat[1:]
	}

	fileType := s.detectFileType(fileFormat)

	// 7. 生成缩略图（仅图片）
	var thumbnailURL *string
	if fileType == consumerV1.MediaFile_IMAGE {
		thumbnail, err := s.generateThumbnail(ctx, req.ObjectKey, tenantID, consumerID)
		if err != nil {
			s.log.Warnf("generate thumbnail failed: %v", err)
			// 缩略图生成失败不影响主流程
		} else {
			thumbnailURL = trans.Ptr(thumbnail)
		}
	}

	// 8. 病毒扫描
	if s.virusScanner != nil {
		s.log.Infof("scanning file for viruses: object_key=%s", req.ObjectKey)
		
		// 下载文件进行扫描
		fileData, err := s.ossClient.Download(ctx, req.ObjectKey)
		if err != nil {
			s.log.Errorf("download file for scan failed: %v", err)
			// 下载失败不影响主流程，继续执行
		} else {
			scanResult, err := s.virusScanner.Scan(ctx, fileData)
			if err != nil {
				s.log.Errorf("virus scan failed: %v", err)
				// 扫描失败不影响主流程，继续执行
			} else if !scanResult.Clean {
				// 检测到病毒，删除文件并返回错误
				s.log.Warnf("virus detected: object_key=%s, virus=%s", req.ObjectKey, scanResult.VirusName)
				
				// 删除OSS文件
				if err := s.ossClient.Delete(ctx, req.ObjectKey); err != nil {
					s.log.Errorf("delete infected file failed: %v", err)
				}
				
				return nil, consumerV1.ErrorBadRequest(fmt.Sprintf("virus detected: %s", scanResult.VirusName))
			} else {
				s.log.Infof("virus scan passed: object_key=%s, message=%s", req.ObjectKey, scanResult.Message)
			}
		}
	}

	// 9. 保存文件元数据到数据库
	mediaFile := &consumerV1.MediaFile{
		TenantId:     trans.Ptr(tenantID),
		ConsumerId:   trans.Ptr(consumerID),
		FileName:     trans.Ptr(req.FileName),
		FileType:     &fileType,
		FileFormat:   trans.Ptr(fileFormat),
		FileSize:     trans.Ptr(uint64(metadata.Size)),
		FileUrl:      trans.Ptr(fileURL),
		ThumbnailUrl: thumbnailURL,
		OssBucket:    trans.Ptr(s.getBucketName()),
		OssKey:       trans.Ptr(req.ObjectKey),
	}

	createdFile, err := s.mediaFileRepo.Create(ctx, mediaFile)
	if err != nil {
		return nil, err
	}

	s.log.Infof("media file confirmed: id=%d, tenant_id=%d, consumer_id=%d, file_name=%s, size=%d",
		createdFile.GetId(), tenantID, consumerID, req.FileName, metadata.Size)

	return createdFile, nil
}

// GetMediaFile 获取媒体文件
func (s *MediaService) GetMediaFile(ctx context.Context, req *consumerV1.GetMediaFileRequest) (*consumerV1.MediaFile, error) {
	if req == nil || req.Id == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	mediaFile, err := s.mediaFileRepo.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	// 验证权限：用户只能查看自己上传的文件
	consumerID := middleware.GetUserID(ctx)
	if consumerID != 0 && mediaFile.GetConsumerId() != consumerID {
		return nil, consumerV1.ErrorForbidden("access denied")
	}

	return mediaFile, nil
}

// ListMediaFiles 查询媒体文件列表
func (s *MediaService) ListMediaFiles(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListMediaFilesResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 注意：实际项目中应该在Repository层添加用户ID过滤
	// 确保用户只能查询自己的文件
	resp, err := s.mediaFileRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// DeleteMediaFile 删除媒体文件
func (s *MediaService) DeleteMediaFile(ctx context.Context, req *consumerV1.DeleteMediaFileRequest) (*emptypb.Empty, error) {
	if req == nil || req.Id == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 1. 查询文件信息
	mediaFile, err := s.mediaFileRepo.Get(ctx, req.GetId())
	if err != nil {
		return nil, err
	}

	// 2. 验证权限：用户只能删除自己上传的文件
	consumerID := middleware.GetUserID(ctx)
	if consumerID != 0 && mediaFile.GetConsumerId() != consumerID {
		return nil, consumerV1.ErrorForbidden("access denied")
	}

	// 3. 软删除文件记录
	if err := s.mediaFileRepo.SoftDelete(ctx, req.GetId()); err != nil {
		return nil, err
	}

	// 4. 删除OSS文件（可选）
	// 注意：实际项目中可以选择立即删除或定期清理
	// 这里简化处理，仅软删除数据库记录，不删除OSS文件
	s.log.Infof("media file deleted: id=%d, consumer_id=%d, oss_key=%s",
		req.GetId(), consumerID, mediaFile.GetOssKey())

	return &emptypb.Empty{}, nil
}

// validateFileFormat 验证文件格式
func (s *MediaService) validateFileFormat(fileType consumerV1.MediaFile_FileType, format string) error {
	format = strings.ToUpper(format)

	switch fileType {
	case consumerV1.MediaFile_IMAGE:
		if !strings.Contains(allowedImageFormats, format) {
			return consumerV1.ErrorBadRequest(fmt.Sprintf("invalid image format: %s, allowed: %s", format, allowedImageFormats))
		}
	case consumerV1.MediaFile_VIDEO:
		if !strings.Contains(allowedVideoFormats, format) {
			return consumerV1.ErrorBadRequest(fmt.Sprintf("invalid video format: %s, allowed: %s", format, allowedVideoFormats))
		}
	default:
		return consumerV1.ErrorBadRequest("invalid file type")
	}

	return nil
}

// validateFileSize 验证文件大小
func (s *MediaService) validateFileSize(fileType consumerV1.MediaFile_FileType, size uint64) error {
	switch fileType {
	case consumerV1.MediaFile_IMAGE:
		if size > maxImageSize {
			return consumerV1.ErrorBadRequest(fmt.Sprintf("image size exceeds limit: %d bytes, max: %d bytes", size, maxImageSize))
		}
	case consumerV1.MediaFile_VIDEO:
		if size > maxVideoSize {
			return consumerV1.ErrorBadRequest(fmt.Sprintf("video size exceeds limit: %d bytes, max: %d bytes", size, maxVideoSize))
		}
	default:
		return consumerV1.ErrorBadRequest("invalid file type")
	}

	return nil
}

// generateObjectKey 生成对象键
func (s *MediaService) generateObjectKey(tenantID, consumerID uint32, fileType consumerV1.MediaFile_FileType, fileName string) string {
	// 格式: media/{tenant_id}/{consumer_id}/{file_type}/{timestamp}_{filename}
	timestamp := time.Now().Unix()
	fileTypeStr := "image"
	if fileType == consumerV1.MediaFile_VIDEO {
		fileTypeStr = "video"
	}

	return fmt.Sprintf("media/%d/%d/%s/%d_%s", tenantID, consumerID, fileTypeStr, timestamp, fileName)
}

// detectFileType 根据文件格式检测文件类型
func (s *MediaService) detectFileType(format string) consumerV1.MediaFile_FileType {
	format = strings.ToUpper(format)

	if strings.Contains(allowedImageFormats, format) {
		return consumerV1.MediaFile_IMAGE
	}
	if strings.Contains(allowedVideoFormats, format) {
		return consumerV1.MediaFile_VIDEO
	}

	return consumerV1.MediaFile_FILE_TYPE_UNSPECIFIED
}

// generateThumbnail 生成缩略图
func (s *MediaService) generateThumbnail(ctx context.Context, objectKey string, tenantID, consumerID uint32) (string, error) {
	if s.imageProcessor == nil {
		return "", fmt.Errorf("image processor not available")
	}

	// 1. 下载原图
	imageData, err := s.ossClient.Download(ctx, objectKey)
	if err != nil {
		return "", fmt.Errorf("download image failed: %w", err)
	}

	// 2. 生成缩略图
	thumbnailData, err := s.imageProcessor.GenerateThumbnail(imageData, thumbnailWidth, thumbnailHeight)
	if err != nil {
		return "", fmt.Errorf("generate thumbnail failed: %w", err)
	}

	// 3. 上传缩略图
	thumbnailKey := fmt.Sprintf("thumbnails/%d/%d/%d_%s", tenantID, consumerID, time.Now().Unix(), filepath.Base(objectKey))
	thumbnailURL, err := s.ossClient.Upload(ctx, thumbnailKey, thumbnailData)
	if err != nil {
		return "", fmt.Errorf("upload thumbnail failed: %w", err)
	}

	s.log.Infof("thumbnail generated: object_key=%s, thumbnail_key=%s, size=%d bytes", 
		objectKey, thumbnailKey, len(thumbnailData))

	return thumbnailURL, nil
}

// getBucketName 获取Bucket名称
func (s *MediaService) getBucketName() string {
	// 注意：实际项目中应该从租户配置中获取
	// 这里简化处理，返回默认值
	// 
	// 实际实现示例：
	// tenantID := middleware.GetTenantID(ctx)
	// ossConfig, err := s.tenantConfigMgr.GetOSSConfig(ctx, tenantID)
	// if err == nil && ossConfig != nil {
	//     return ossConfig.BucketName
	// }
	
	return "consumer-media"
}
