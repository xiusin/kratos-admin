package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	entCrud "github.com/tx7do/go-crud/entgo"

	"github.com/tx7do/go-utils/copierutil"
	"github.com/tx7do/go-utils/mapper"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/logisticstracking"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// LogisticsTrackingRepo 物流跟踪数据访问接口
type LogisticsTrackingRepo interface {
	// Create 创建物流跟踪
	Create(ctx context.Context, data *consumerV1.LogisticsTracking) (*consumerV1.LogisticsTracking, error)

	// Get 查询物流跟踪
	Get(ctx context.Context, id uint64) (*consumerV1.LogisticsTracking, error)

	// GetByTrackingNo 按运单号查询
	GetByTrackingNo(ctx context.Context, trackingNo string) (*consumerV1.LogisticsTracking, error)

	// Update 更新物流信息
	Update(ctx context.Context, id uint64, data *consumerV1.LogisticsTracking) error

	// List 分页查询物流历史
	List(ctx context.Context, req *consumerV1.ListLogisticsHistoryRequest) (*consumerV1.ListLogisticsHistoryResponse, error)
}

type logisticsTrackingRepo struct {
	entClient       *entCrud.EntClient[*ent.Client]
	log             *log.Helper
	mapper          *mapper.CopierMapper[consumerV1.LogisticsTracking, ent.LogisticsTracking]
	statusConverter *mapper.EnumTypeConverter[consumerV1.LogisticsTracking_Status, logisticstracking.Status]

	repository *entCrud.Repository[
		ent.LogisticsTrackingQuery, ent.LogisticsTrackingSelect,
		ent.LogisticsTrackingCreate, ent.LogisticsTrackingCreateBulk,
		ent.LogisticsTrackingUpdate, ent.LogisticsTrackingUpdateOne,
		ent.LogisticsTrackingDelete,
		predicate.LogisticsTracking,
		consumerV1.LogisticsTracking, ent.LogisticsTracking,
	]
}

// NewLogisticsTrackingRepo 创建物流跟踪数据访问实例
func NewLogisticsTrackingRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) LogisticsTrackingRepo {
	repo := &logisticsTrackingRepo{
		log:             ctx.NewLoggerHelper("logistics-tracking/repo/consumer-service"),
		entClient:       entClient,
		mapper:          mapper.NewCopierMapper[consumerV1.LogisticsTracking, ent.LogisticsTracking](),
		statusConverter: mapper.NewEnumTypeConverter[consumerV1.LogisticsTracking_Status, logisticstracking.Status](consumerV1.LogisticsTracking_Status_name, consumerV1.LogisticsTracking_Status_value),
	}

	repo.init()

	return repo
}

func (r *logisticsTrackingRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.LogisticsTrackingQuery, ent.LogisticsTrackingSelect,
		ent.LogisticsTrackingCreate, ent.LogisticsTrackingCreateBulk,
		ent.LogisticsTrackingUpdate, ent.LogisticsTrackingUpdateOne,
		ent.LogisticsTrackingDelete,
		predicate.LogisticsTracking,
		consumerV1.LogisticsTracking, ent.LogisticsTracking,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.statusConverter.NewConverterPair())
}

// Create 创建物流跟踪
func (r *logisticsTrackingRepo) Create(ctx context.Context, data *consumerV1.LogisticsTracking) (*consumerV1.LogisticsTracking, error) {
	if data == nil {
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().LogisticsTracking.Create().
		SetNillableTenantID(data.TenantId).
		SetTrackingNo(data.GetTrackingNo()).
		SetCourierCompany(data.GetCourierCompany()).
		SetLastUpdatedAt(time.Now())

	// 设置 status
	if status := r.statusConverter.ToEntity(data.Status); status != nil {
		builder.SetStatus(*status)
	}

	// 设置 tracking_info
	if data.TrackingInfo != nil && len(data.TrackingInfo) > 0 {
		trackingInfo := make([]map[string]interface{}, 0, len(data.TrackingInfo))
		for _, info := range data.TrackingInfo {
			trackingInfo = append(trackingInfo, info.AsMap())
		}
		builder.SetTrackingInfo(trackingInfo)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert logistics tracking failed: %s", err.Error())
		return nil, errors.InternalServer("INSERT_FAILED", "insert logistics tracking failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// Get 查询物流跟踪
func (r *logisticsTrackingRepo) Get(ctx context.Context, id uint64) (*consumerV1.LogisticsTracking, error) {
	builder := r.entClient.Client().LogisticsTracking.Query()

	dto, err := r.repository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.EQ(logisticstracking.FieldID, id))
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// GetByTrackingNo 按运单号查询
func (r *logisticsTrackingRepo) GetByTrackingNo(ctx context.Context, trackingNo string) (*consumerV1.LogisticsTracking, error) {
	builder := r.entClient.Client().LogisticsTracking.Query()

	dto, err := r.repository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.EQ(logisticstracking.FieldTrackingNo, trackingNo))
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// Update 更新物流信息
func (r *logisticsTrackingRepo) Update(ctx context.Context, id uint64, data *consumerV1.LogisticsTracking) error {
	if data == nil {
		return errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().LogisticsTracking.UpdateOneID(uint32(id)).
		SetLastUpdatedAt(time.Now())

	if data.Status != nil {
		status := r.statusConverter.ToEntity(data.Status)
		if status != nil {
			builder.SetStatus(*status)
		}
	}

	if data.TrackingInfo != nil && len(data.TrackingInfo) > 0 {
		trackingInfo := make([]map[string]interface{}, 0, len(data.TrackingInfo))
		for _, info := range data.TrackingInfo {
			trackingInfo = append(trackingInfo, info.AsMap())
		}
		builder.SetTrackingInfo(trackingInfo)
	}

	if _, err := builder.Save(ctx); err != nil {
		if ent.IsNotFound(err) {
			return errors.NotFound("LOGISTICS_TRACKING_NOT_FOUND", "logistics tracking not found")
		}
		r.log.Errorf("update logistics tracking failed: %s", err.Error())
		return errors.InternalServer("UPDATE_FAILED", "update logistics tracking failed")
	}

	return nil
}

// List 分页查询物流历史
func (r *logisticsTrackingRepo) List(ctx context.Context, req *consumerV1.ListLogisticsHistoryRequest) (*consumerV1.ListLogisticsHistoryResponse, error) {
	if req == nil {
		return nil, errors.BadRequest("INVALID_PARAMETER", "invalid parameter")
	}

	builder := r.entClient.Client().LogisticsTracking.Query()

	// 添加筛选条件
	if req.ConsumerId != nil {
		// TODO: 需要在 Schema 中添加 consumer_id 字段才能按用户筛选
		// builder.Where(logisticstracking.ConsumerIDEQ(*req.ConsumerId))
	}

	if req.Status != nil {
		status := r.statusConverter.ToEntity(req.Status)
		if status != nil {
			builder.Where(logisticstracking.StatusEQ(*status))
		}
	}

	// 按创建时间倒序
	builder.Order(ent.Desc(logisticstracking.FieldCreatedAt))

	// 构建分页请求
	pagingReq := &paginationV1.PagingRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), pagingReq)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &consumerV1.ListLogisticsHistoryResponse{Total: 0, Items: nil}, nil
	}

	return &consumerV1.ListLogisticsHistoryResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}
