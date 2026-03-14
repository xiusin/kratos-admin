package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
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
	GetByTrackingNo(ctx context.Context, tenantID uint32, trackingNo string) (*consumerV1.LogisticsTracking, error)

	// Update 更新物流信息
	Update(ctx context.Context, id uint64, data *consumerV1.LogisticsTracking) error

	// List 分页查询物流历史
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListLogisticsHistoryResponse, error)
}

type logisticsTrackingRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper           *mapper.CopierMapper[consumerV1.LogisticsTracking, ent.LogisticsTracking]
	statusConverter  *mapper.EnumTypeConverter[consumerV1.LogisticsTracking_Status, logisticstracking.Status]

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
		log:             ctx.NewLoggerHelper("consumer/repo/logistics-tracking"),
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
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LogisticsTracking.Create().
		SetNillableTenantID(data.TenantId).
		SetNillableTrackingNo(data.TrackingNo).
		SetNillableCourierCompany(data.CourierCompany).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetNillableLastUpdatedAt(data.LastUpdatedAt.AsTime()).
		SetCreatedAt(time.Now())

	// 设置物流轨迹（JSON字段）
	if len(data.TrackingInfo) > 0 {
		trackingInfo := make([]map[string]interface{}, 0, len(data.TrackingInfo))
		for _, info := range data.TrackingInfo {
			trackingInfo = append(trackingInfo, info.AsMap())
		}
		builder.SetTrackingInfo(trackingInfo)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert logistics tracking failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("insert logistics tracking failed")
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
func (r *logisticsTrackingRepo) GetByTrackingNo(ctx context.Context, tenantID uint32, trackingNo string) (*consumerV1.LogisticsTracking, error) {
	if trackingNo == "" {
		return nil, consumerV1.ErrorBadRequest("tracking_no is required")
	}

	builder := r.entClient.Client().LogisticsTracking.Query()

	dto, err := r.repository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.And(
				sql.EQ(logisticstracking.FieldTenantID, tenantID),
				sql.EQ(logisticstracking.FieldTrackingNo, trackingNo),
			))
		},
	)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil // 不存在返回nil，不报错
		}
		return nil, err
	}

	return dto, nil
}

// Update 更新物流信息
func (r *logisticsTrackingRepo) Update(ctx context.Context, id uint64, data *consumerV1.LogisticsTracking) error {
	if data == nil {
		return consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LogisticsTracking.UpdateOneID(id).
		SetNillableStatus(r.statusConverter.ToEntity(data.Status)).
		SetNillableLastUpdatedAt(data.LastUpdatedAt.AsTime())

	// 更新物流轨迹（JSON字段）
	if len(data.TrackingInfo) > 0 {
		trackingInfo := make([]map[string]interface{}, 0, len(data.TrackingInfo))
		for _, info := range data.TrackingInfo {
			trackingInfo = append(trackingInfo, info.AsMap())
		}
		builder.SetTrackingInfo(trackingInfo)
	}

	err := builder.Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return consumerV1.ErrorNotFound("logistics tracking not found")
		}
		r.log.Errorf("update logistics tracking failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("update logistics tracking failed")
	}

	return nil
}

// List 分页查询物流历史
func (r *logisticsTrackingRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListLogisticsHistoryResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().LogisticsTracking.Query().
		Order(ent.Desc(logisticstracking.FieldCreatedAt)) // 按创建时间倒序

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
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
