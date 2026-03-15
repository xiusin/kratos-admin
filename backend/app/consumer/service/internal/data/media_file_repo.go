package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	entCrud "github.com/tx7do/go-crud/entgo"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/mediafile"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"
)

// MediaFileRepo 媒体文件数据访问接口
type MediaFileRepo interface {
	Create(ctx context.Context, file *ent.MediaFile) (*ent.MediaFile, error)
	Get(ctx context.Context, id uint32) (*ent.MediaFile, error)
	List(ctx context.Context, page, pageSize int) ([]*ent.MediaFile, int, error)
	SoftDelete(ctx context.Context, id uint32) error
	GetByOSSKey(ctx context.Context, ossKey string) (*ent.MediaFile, error)
}

type mediaFileRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

// NewMediaFileRepo 创建媒体文件Repository
func NewMediaFileRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) MediaFileRepo {
	return &mediaFileRepo{
		entClient: entClient,
		log:       ctx.NewLoggerHelper("media-file/repo/consumer-service"),
	}
}

// Create 创建媒体文件记录
func (r *mediaFileRepo) Create(ctx context.Context, file *ent.MediaFile) (*ent.MediaFile, error) {
	po, err := r.entClient.Client().MediaFile.Create().
		SetNillableTenantID(file.TenantID).
		SetConsumerID(file.ConsumerID).
		SetFileName(file.FileName).
		SetFileType(file.FileType).
		SetFileFormat(file.FileFormat).
		SetFileSize(file.FileSize).
		SetFileURL(file.FileURL).
		SetNillableThumbnailURL(file.ThumbnailURL).
		SetOssBucket(file.OssBucket).
		SetOssKey(file.OssKey).
		Save(ctx)

	if err != nil {
		r.log.Errorf("failed to create media file: %v", err)
		return nil, errors.InternalServer("CREATE_FAILED", "create media file failed")
	}

	return po, nil
}

// Get 查询媒体文件
func (r *mediaFileRepo) Get(ctx context.Context, id uint32) (*ent.MediaFile, error) {
	po, err := r.entClient.Client().MediaFile.Query().
		Where(
			mediafile.ID(id),
			mediafile.IsDeleted(false),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("failed to get media file: %v", err)
		return nil, errors.InternalServer("GET_FAILED", "get media file failed")
	}

	return po, nil
}

// List 分页查询媒体文件列表（过滤已删除）
func (r *mediaFileRepo) List(ctx context.Context, page, pageSize int) ([]*ent.MediaFile, int, error) {
	predicates := []predicate.MediaFile{
		mediafile.IsDeleted(false),
	}

	total, err := r.entClient.Client().MediaFile.Query().
		Where(predicates...).
		Count(ctx)
	if err != nil {
		r.log.Errorf("failed to count media files: %v", err)
		return nil, 0, errors.InternalServer("COUNT_FAILED", "count media files failed")
	}

	offset := (page - 1) * pageSize
	list, err := r.entClient.Client().MediaFile.Query().
		Where(predicates...).
		Order(ent.Desc(mediafile.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)

	if err != nil {
		r.log.Errorf("failed to list media files: %v", err)
		return nil, 0, errors.InternalServer("LIST_FAILED", "list media files failed")
	}

	return list, total, nil
}

// SoftDelete 软删除媒体文件
func (r *mediaFileRepo) SoftDelete(ctx context.Context, id uint32) error {
	err := r.entClient.Client().MediaFile.Update().
		Where(
			mediafile.ID(id),
			mediafile.IsDeleted(false),
		).
		SetIsDeleted(true).
		SetDeletedAt(time.Now()).
		Exec(ctx)

	if err != nil {
		r.log.Errorf("failed to soft delete media file: %v", err)
		return errors.InternalServer("DELETE_FAILED", "delete media file failed")
	}

	return nil
}

// GetByOSSKey 根据OSS Key查询
func (r *mediaFileRepo) GetByOSSKey(ctx context.Context, ossKey string) (*ent.MediaFile, error) {
	po, err := r.entClient.Client().MediaFile.Query().
		Where(
			mediafile.OssKey(ossKey),
			mediafile.IsDeleted(false),
		).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		r.log.Errorf("failed to get media file by oss key: %v", err)
		return nil, errors.InternalServer("GET_FAILED", "get media file by oss key failed")
	}

	return po, nil
}
