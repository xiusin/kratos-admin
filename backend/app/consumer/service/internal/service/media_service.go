package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/mediafile"
	"go-wind-admin/pkg/oss"
)

// MediaService 媒体服务
type MediaService struct {
	consumerV1.UnimplementedMediaServiceServer

	mediaFileRepo data.MediaFileRepo
	ossClient     oss.Client
	log           *log.Helper
}

// NewMediaService 创建媒体服务
func NewMediaService(
	ctx *bootstrap.Context,
	mediaFileRepo data.MediaFileRepo,
	ossClient oss.Client,
) *MediaService {
	return &MediaService{
		mediaFileRepo: mediaFileRepo,
		ossClient:     ossClient,
		log:           log.NewHelper(log.With(ctx.GetLogger(), "module", "service/media")),
	}
}

// 文件格式验证规则
var (
	// 允许的图片格式
	allowedImageFormats = map[string]bool{
		"JPEG": true,
		"JPG":  true,
		"PNG":  true,
		"GIF":  true,
	}

	// 允许的视频格式
	allowedVideoFormats = map[string]bool{
		"MP4": true,
		"AVI": true,
		"MOV": true,
	}

	// 文件大小限制（字节）
	maxImageSize = uint64(5 * 1024 * 1024)   // 5MB
	maxVideoSize = uint64(100 * 1024 * 1024) // 100MB

	// 预签名URL有效期（秒）
	presignedURLExpireSeconds = int64(3600) // 1小时
)

// validateFileFormat 验证文件格式
func (s *MediaService) validateFileFormat(fileType consumerV1.MediaFile_FileType, format string) error {
	formatUpper := strings.ToUpper(format)

	switch fileType {
	case consumerV1.MediaFile_IMAGE:
		if !allowedImageFormats[formatUpper] {
			return errors.BadRequest("INVALID_FILE_FORMAT", "不支持的图片格式，仅支持: JPEG, PNG, GIF")
		}
	case consumerV1.MediaFile_VIDEO:
		if !allowedVideoFormats[formatUpper] {
			return errors.BadRequest("INVALID_FILE_FORMAT", "不支持的视频格式，仅支持: MP4, AVI, MOV")
		}
	default:
		return errors.BadRequest("INVALID_FILE_TYPE", "不支持的文件类型")
	}

	return nil
}

// validateFileSize 验证文件大小
func (s *MediaService) validateFileSize(fileType consumerV1.MediaFile_FileType, size uint64) error {
	switch fileType {
	case consumerV1.MediaFile_IMAGE:
		if size > maxImageSize {
			return errors.BadRequest("FILE_TOO_LARGE", fmt.Sprintf("图片文件大小不能超过 %d MB", maxImageSize/(1024*1024)))
		}
	case consumerV1.MediaFile_VIDEO:
		if size > maxVideoSize {
			return errors.BadRequest("FILE_TOO_LARGE", fmt.Sprintf("视频文件大小不能超过 %d MB", maxVideoSize/(1024*1024)))
		}
	}

	return nil
}

// generateOSSKey 生成OSS对象键
func (s *MediaService) generateOSSKey(fileType consumerV1.MediaFile_FileType, fileName string) string {
	uniqueID := uuid.New().String()
	ext := filepath.Ext(fileName)

	var prefix string
	switch fileType {
	case consumerV1.MediaFile_IMAGE:
		prefix = "images"
	case consumerV1.MediaFile_VIDEO:
		prefix = "videos"
	default:
		prefix = "files"
	}

	datePrefix := time.Now().Format("2006/01/02")
	return fmt.Sprintf("%s/%s/%s%s", prefix, datePrefix, uniqueID, ext)
}

// GenerateUploadURL 生成上传预签名URL
func (s *MediaService) GenerateUploadURL(ctx context.Context, req *consumerV1.GenerateUploadURLRequest) (*consumerV1.GenerateUploadURLResponse, error) {
	if err := s.validateFileFormat(req.FileType, req.FileFormat); err != nil {
		return nil, err
	}

	if err := s.validateFileSize(req.FileType, req.FileSize); err != nil {
		return nil, err
	}

	ossKey := s.generateOSSKey(req.FileType, req.FileName)

	uploadURL, err := s.ossClient.GeneratePresignedURL(ctx, ossKey, presignedURLExpireSeconds)
	if err != nil {
		s.log.Errorf("failed to generate presigned url: %v", err)
		return nil, errors.InternalServer("GENERATE_URL_FAILED", "生成上传URL失败")
	}

	return &consumerV1.GenerateUploadURLResponse{
		UploadUrl: uploadURL,
		FileKey:   ossKey,
		ExpiresIn: presignedURLExpireSeconds,
	}, nil
}

// ConfirmUpload 确认上传完成
func (s *MediaService) ConfirmUpload(ctx context.Context, req *consumerV1.ConfirmUploadRequest) (*consumerV1.MediaFile, error) {
	if err := s.validateFileFormat(req.FileType, req.FileFormat); err != nil {
		return nil, err
	}

	if err := s.validateFileSize(req.FileType, req.FileSize); err != nil {
		return nil, err
	}

	exists, err := s.ossClient.Exists(ctx, req.FileKey)
	if err != nil {
		s.log.Errorf("failed to check file exists: %v", err)
		return nil, errors.InternalServer("CHECK_FILE_FAILED", "检查文件失败")
	}
	if !exists {
		return nil, errors.BadRequest("FILE_NOT_FOUND", "文件不存在，请先上传文件")
	}

	fileURL, err := s.ossClient.GenerateDownloadURL(ctx, req.FileKey, 365*24*3600)
	if err != nil {
		s.log.Errorf("failed to generate download url: %v", err)
		return nil, errors.InternalServer("GENERATE_URL_FAILED", "生成文件URL失败")
	}

	// TODO: 从context中获取当前用户ID
	consumerID := uint32(1) // 临时硬编码

	mediaFile := &ent.MediaFile{
		ConsumerID: consumerID,
		FileName:   req.FileName,
		FileType:   s.convertFileType(req.FileType),
		FileFormat: strings.ToUpper(req.FileFormat),
		FileSize:   req.FileSize,
		FileURL:    fileURL,
		OssBucket:  "",
		OssKey:     req.FileKey,
	}

	var thumbnailURL *string
	if req.FileType == consumerV1.MediaFile_IMAGE {
		thumbnail, err := s.generateThumbnail(ctx, req.FileKey)
		if err != nil {
			s.log.Warnf("failed to generate thumbnail: %v", err)
		} else {
			thumbnailURL = &thumbnail
		}
	}
	mediaFile.ThumbnailURL = thumbnailURL

	created, err := s.mediaFileRepo.Create(ctx, mediaFile)
	if err != nil {
		s.log.Errorf("failed to create media file: %v", err)
		return nil, errors.InternalServer("CREATE_MEDIA_FILE_FAILED", "保存媒体文件记录失败")
	}

	return s.toProtoMediaFile(created), nil
}

// generateThumbnail 生成缩略图
func (s *MediaService) generateThumbnail(ctx context.Context, ossKey string) (string, error) {
	thumbnailKey := strings.Replace(ossKey, filepath.Ext(ossKey), "_thumb"+filepath.Ext(ossKey), 1)
	thumbnailURL, err := s.ossClient.GenerateDownloadURL(ctx, thumbnailKey, 365*24*3600)
	if err != nil {
		return "", err
	}
	return thumbnailURL, nil
}

// GetMediaFile 获取媒体文件
func (s *MediaService) GetMediaFile(ctx context.Context, req *consumerV1.GetMediaFileRequest) (*consumerV1.MediaFile, error) {
	mediaFile, err := s.mediaFileRepo.Get(ctx, uint32(req.Id))
	if err != nil {
		s.log.Errorf("failed to get media file: %v", err)
		return nil, errors.InternalServer("GET_MEDIA_FILE_FAILED", "查询媒体文件失败")
	}

	if mediaFile == nil {
		return nil, errors.NotFound("MEDIA_FILE_NOT_FOUND", "媒体文件不存在")
	}

	return s.toProtoMediaFile(mediaFile), nil
}

// ListMediaFiles 查询媒体文件列表
func (s *MediaService) ListMediaFiles(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListMediaFilesResponse, error) {
	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	list, total, err := s.mediaFileRepo.List(ctx, page, pageSize)
	if err != nil {
		s.log.Errorf("failed to list media files: %v", err)
		return nil, errors.InternalServer("LIST_MEDIA_FILES_FAILED", "查询媒体文件列表失败")
	}

	items := make([]*consumerV1.MediaFile, 0, len(list))
	for _, item := range list {
		items = append(items, s.toProtoMediaFile(item))
	}

	return &consumerV1.ListMediaFilesResponse{
		Items: items,
		Total: uint64(total),
	}, nil
}

// DeleteMediaFile 删除媒体文件
func (s *MediaService) DeleteMediaFile(ctx context.Context, req *consumerV1.DeleteMediaFileRequest) (*emptypb.Empty, error) {
	err := s.mediaFileRepo.SoftDelete(ctx, uint32(req.Id))
	if err != nil {
		s.log.Errorf("failed to delete media file: %v", err)
		return nil, errors.InternalServer("DELETE_MEDIA_FILE_FAILED", "删除媒体文件失败")
	}
	return &emptypb.Empty{}, nil
}

// convertFileType 转换文件类型（Proto -> Ent）
func (s *MediaService) convertFileType(fileType consumerV1.MediaFile_FileType) mediafile.FileType {
	switch fileType {
	case consumerV1.MediaFile_IMAGE:
		return mediafile.FileTypeImage
	case consumerV1.MediaFile_VIDEO:
		return mediafile.FileTypeVideo
	default:
		return mediafile.FileTypeImage
	}
}

// convertFileTypeToProto 转换文件类型（Ent -> Proto）
func (s *MediaService) convertFileTypeToProto(fileType mediafile.FileType) consumerV1.MediaFile_FileType {
	switch fileType {
	case mediafile.FileTypeImage:
		return consumerV1.MediaFile_IMAGE
	case mediafile.FileTypeVideo:
		return consumerV1.MediaFile_VIDEO
	default:
		return consumerV1.MediaFile_IMAGE
	}
}

// toProtoMediaFile 转换为Proto MediaFile
func (s *MediaService) toProtoMediaFile(file *ent.MediaFile) *consumerV1.MediaFile {
	if file == nil {
		return nil
	}

	result := &consumerV1.MediaFile{
		Id:           func() *uint64 { v := uint64(file.ID); return &v }(),
		TenantId:     file.TenantID,
		ConsumerId:   &file.ConsumerID,
		FileName:     &file.FileName,
		FileType:     s.convertFileTypeToProto(file.FileType).Enum(),
		FileFormat:   &file.FileFormat,
		FileSize:     &file.FileSize,
		FileUrl:      &file.FileURL,
		ThumbnailUrl: file.ThumbnailURL,
		OssBucket:    &file.OssBucket,
		OssKey:       &file.OssKey,
		IsDeleted:    &file.IsDeleted,
	}

	// Handle optional CreatedAt field
	if file.CreatedAt != nil {
		result.CreatedAt = timestamppb.New(*file.CreatedAt)
	}

	// Handle optional DeletedAt field
	if file.DeletedAt != nil {
		result.DeletedAt = timestamppb.New(*file.DeletedAt)
	}

	return result
}
