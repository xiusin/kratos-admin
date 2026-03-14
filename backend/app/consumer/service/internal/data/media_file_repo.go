package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/mediafile"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// MediaFileRepo 媒体文件数据访问接口
type MediaFileRepo interface {
	// Create 记录媒体文件
	Create(ctx context.Context, data *consumerV1.MediaFile) (*consumerV1.MediaFile, error)

	// Get 查询媒体文件
	Get(ctx context.Context, id uint64) (*consumerV1.MediaFile, error)

	// List 分页查询(过滤已删除)
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListMediaFilesResponse, error)

	// SoftDelete 软删除
	SoftDelete(ctx context.Context, id uint64) error
}

type mediaFileRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper            *mapper.CopierMapper[consumerV1.MediaFile, ent.MediaFile]
	fileTypeConverter *mapper.EnumTypeConverter[consumerV1.MediaFile_FileType, mediafile.FileType]

	repository *entCrud.Repository[
		ent.MediaFileQuery, ent.MediaFileSelect,
		ent.MediaFileCreate, ent.MediaFileCreateBulk,
		ent.MediaFileUpdate, ent.MediaFileUpdateOne,
		ent.MediaFileDelete,
		predicate.MediaFile,
		consumerV1.MediaFile, ent.MediaFile,
	]
}

// NewMediaFileRepo 创建媒体文件数据访问实例
func NewMediaFileRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) MediaFileRepo {
	repo := &mediaFileRepo{
		log:               ctx.NewLoggerHelper("consumer/repo/media-file"),
		entClient:         entClient,
		mapper:            mapper.NewCopierMapper[consumerV1.MediaFile, ent.MediaFile](),
		fileTypeConverter: mapper.NewEnumTypeConverter[consumerV1.MediaFile_FileType, mediafile.FileType](consumerV1.MediaFile_FileType_name, consumerV1.MediaFile_FileType_value),
	}

	repo.init()

	return repo
}

func (r *mediaFileRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.MediaFileQuery, ent.MediaFileSelect,
		ent.MediaFileCreate, ent.MediaFileCreateBulk,
		ent.MediaFileUpdate, ent.MediaFileUpdateOne,
		ent.MediaFileDelete,
		predicate.MediaFile,
		consumerV1.MediaFile, ent.MediaFile,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.fileTypeConverter.NewConverterPair())
}

// Create 记录媒体文件
func (r *mediaFileRepo) Create(ctx context.Context, data *consumerV1.MediaFile) (*consumerV1.MediaFile, error) {
	if data == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().MediaFile.Create().
		SetNillableTenantID(data.TenantId).
		SetNillableThumbnailURL(data.ThumbnailUrl).
		SetIsDeleted(false).
		SetCreatedAt(time.Now())

	// 设置必填字段 consumer_id
	if data.ConsumerId != nil {
		builder.SetConsumerID(*data.ConsumerId)
	}
	
	// 设置必填字段 file_name
	if data.FileName != nil {
		builder.SetFileName(*data.FileName)
	}
	
	// 设置必填字段 file_type
	if data.FileType != nil {
		fileType := r.fileTypeConverter.ToEntity(data.FileType)
		if fileType != nil {
			builder.SetFileType(*fileType)
		}
	}
	
	// 设置必填字段 file_format
	if data.FileFormat != nil {
		builder.SetFileFormat(*data.FileFormat)
	}
	
	// 设置必填字段 file_size
	if data.FileSize != nil {
		builder.SetFileSize(*data.FileSize)
	}
	
	// 设置必填字段 file_url
	if data.FileUrl != nil {
		builder.SetFileURL(*data.FileUrl)
	}
	
	// 设置必填字段 oss_bucket
	if data.OssBucket != nil {
		builder.SetOssBucket(*data.OssBucket)
	}
	
	// 设置必填字段 oss_key
	if data.OssKey != nil {
		builder.SetOssKey(*data.OssKey)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert media file failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("insert media file failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// Get 查询媒体文件
func (r *mediaFileRepo) Get(ctx context.Context, id uint64) (*consumerV1.MediaFile, error) {
	entity, err := r.entClient.Client().MediaFile.Query().
		Where(mediafile.ID(uint32(id))).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, consumerV1.ErrorNotFound("media file not found")
		}
		r.log.Errorf("get media file failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("get media file failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// List 分页查询(过滤已删除)
func (r *mediaFileRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListMediaFilesResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().MediaFile.Query().
		Where(mediafile.IsDeletedEQ(false)).
		Order(ent.Desc(mediafile.FieldCreatedAt))

	// 计算总数
	count, err := builder.Clone().Count(ctx)
	if err != nil {
		r.log.Errorf("count media files failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("count media files failed")
	}

	// 分页查询
	if req.GetPage() > 0 && req.GetPageSize() > 0 {
		offset := int(req.GetPage()-1) * int(req.GetPageSize())
		builder.Offset(offset).Limit(int(req.GetPageSize()))
	}

	entities, err := builder.All(ctx)
	if err != nil {
		r.log.Errorf("list media files failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("list media files failed")
	}

	items := make([]*consumerV1.MediaFile, 0, len(entities))
	for _, entity := range entities {
		items = append(items, r.mapper.ToDTO(entity))
	}

	return &consumerV1.ListMediaFilesResponse{
		Total: uint64(count),
		Items: items,
	}, nil
}

// SoftDelete 软删除
func (r *mediaFileRepo) SoftDelete(ctx context.Context, id uint64) error {
	now := time.Now()

	err := r.entClient.Client().MediaFile.UpdateOneID(uint32(id)).
		SetIsDeleted(true).
		SetDeletedAt(now).
		SetUpdatedAt(now).
		Exec(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return consumerV1.ErrorNotFound("media file not found")
		}
		r.log.Errorf("soft delete media file failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("soft delete media file failed")
	}

	return nil
}
