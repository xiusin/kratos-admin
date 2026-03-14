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
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"
	"go-wind-admin/app/consumer/service/internal/data/ent/tenantconfig"
	"go-wind-admin/app/consumer/service/internal/data/ent/tenantconfighistory"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// TenantConfigRepo 租户配置数据访问接口
type TenantConfigRepo interface {
	// Create 创建配置
	Create(ctx context.Context, data *consumerV1.TenantConfig) (*consumerV1.TenantConfig, error)

	// Get 查询配置
	Get(ctx context.Context, id uint32) (*consumerV1.TenantConfig, error)

	// GetByKey 按配置键查询
	GetByKey(ctx context.Context, tenantID uint32, key string) (*consumerV1.TenantConfig, error)

	// Update 更新配置
	Update(ctx context.Context, id uint32, data *consumerV1.TenantConfig) error

	// Delete 删除配置
	Delete(ctx context.Context, id uint32) error

	// List 分页查询配置列表
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListTenantConfigsResponse, error)

	// ListByCategory 按分类查询配置
	ListByCategory(ctx context.Context, tenantID uint32, category string) ([]*consumerV1.TenantConfig, error)

	// CreateHistory 创建配置变更历史
	CreateHistory(ctx context.Context, data *consumerV1.TenantConfigHistory) error

	// ListHistory 查询配置变更历史
	ListHistory(ctx context.Context, tenantID uint32, configID uint32, req *paginationV1.PagingRequest) (*consumerV1.ListTenantConfigHistoriesResponse, error)
}

type tenantConfigRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	configMapper  *mapper.CopierMapper[consumerV1.TenantConfig, ent.TenantConfig]
	historyMapper *mapper.CopierMapper[consumerV1.TenantConfigHistory, ent.TenantConfigHistory]

	changeTypeConverter *mapper.EnumTypeConverter[consumerV1.TenantConfigHistory_ChangeType, tenantconfighistory.ChangeType]

	configRepository *entCrud.Repository[
		ent.TenantConfigQuery, ent.TenantConfigSelect,
		ent.TenantConfigCreate, ent.TenantConfigCreateBulk,
		ent.TenantConfigUpdate, ent.TenantConfigUpdateOne,
		ent.TenantConfigDelete,
		predicate.TenantConfig,
		consumerV1.TenantConfig, ent.TenantConfig,
	]

	historyRepository *entCrud.Repository[
		ent.TenantConfigHistoryQuery, ent.TenantConfigHistorySelect,
		ent.TenantConfigHistoryCreate, ent.TenantConfigHistoryCreateBulk,
		ent.TenantConfigHistoryUpdate, ent.TenantConfigHistoryUpdateOne,
		ent.TenantConfigHistoryDelete,
		predicate.TenantConfigHistory,
		consumerV1.TenantConfigHistory, ent.TenantConfigHistory,
	]
}

// NewTenantConfigRepo 创建租户配置数据访问实例
func NewTenantConfigRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) TenantConfigRepo {
	repo := &tenantConfigRepo{
		log:                 ctx.NewLoggerHelper("consumer/repo/tenant-config"),
		entClient:           entClient,
		configMapper:        mapper.NewCopierMapper[consumerV1.TenantConfig, ent.TenantConfig](),
		historyMapper:       mapper.NewCopierMapper[consumerV1.TenantConfigHistory, ent.TenantConfigHistory](),
		changeTypeConverter: mapper.NewEnumTypeConverter[consumerV1.TenantConfigHistory_ChangeType, tenantconfighistory.ChangeType](consumerV1.TenantConfigHistory_ChangeType_name, consumerV1.TenantConfigHistory_ChangeType_value),
	}

	repo.init()

	return repo
}

func (r *tenantConfigRepo) init() {
	r.configRepository = entCrud.NewRepository[
		ent.TenantConfigQuery, ent.TenantConfigSelect,
		ent.TenantConfigCreate, ent.TenantConfigCreateBulk,
		ent.TenantConfigUpdate, ent.TenantConfigUpdateOne,
		ent.TenantConfigDelete,
		predicate.TenantConfig,
		consumerV1.TenantConfig, ent.TenantConfig,
	](r.configMapper)

	r.historyRepository = entCrud.NewRepository[
		ent.TenantConfigHistoryQuery, ent.TenantConfigHistorySelect,
		ent.TenantConfigHistoryCreate, ent.TenantConfigHistoryCreateBulk,
		ent.TenantConfigHistoryUpdate, ent.TenantConfigHistoryUpdateOne,
		ent.TenantConfigHistoryDelete,
		predicate.TenantConfigHistory,
		consumerV1.TenantConfigHistory, ent.TenantConfigHistory,
	](r.historyMapper)

	r.configMapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.configMapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())

	r.historyMapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.historyMapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.historyMapper.AppendConverters(r.changeTypeConverter.NewConverterPair())
}

// Create 创建配置
func (r *tenantConfigRepo) Create(ctx context.Context, data *consumerV1.TenantConfig) (*consumerV1.TenantConfig, error) {
	if data == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().TenantConfig.Create().
		SetNillableTenantID(data.TenantId).
		SetNillableConfigKey(data.ConfigKey).
		SetNillableConfigValue(data.ConfigValue).
		SetNillableConfigType(data.ConfigType).
		SetNillableDescription(data.Description).
		SetNillableCategory(data.Category).
		SetNillableIsEncrypted(data.IsEncrypted).
		SetNillableIsActive(data.IsActive).
		SetNillableValidationRule(data.ValidationRule).
		SetCreatedAt(time.Now())

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert tenant config failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("insert tenant config failed")
	}

	return r.configMapper.ToDTO(entity), nil
}

// Get 查询配置
func (r *tenantConfigRepo) Get(ctx context.Context, id uint32) (*consumerV1.TenantConfig, error) {
	builder := r.entClient.Client().TenantConfig.Query()

	dto, err := r.configRepository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.EQ(tenantconfig.FieldID, id))
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// GetByKey 按配置键查询
func (r *tenantConfigRepo) GetByKey(ctx context.Context, tenantID uint32, key string) (*consumerV1.TenantConfig, error) {
	builder := r.entClient.Client().TenantConfig.Query()

	dto, err := r.configRepository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.And(
				sql.EQ(tenantconfig.FieldTenantID, tenantID),
				sql.EQ(tenantconfig.FieldConfigKey, key),
			))
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// Update 更新配置
func (r *tenantConfigRepo) Update(ctx context.Context, id uint32, data *consumerV1.TenantConfig) error {
	if data == nil {
		return consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().TenantConfig.UpdateOneID(id).
		SetNillableConfigValue(data.ConfigValue).
		SetNillableConfigType(data.ConfigType).
		SetNillableDescription(data.Description).
		SetNillableCategory(data.Category).
		SetNillableIsEncrypted(data.IsEncrypted).
		SetNillableIsActive(data.IsActive).
		SetNillableValidationRule(data.ValidationRule).
		SetUpdatedAt(time.Now())

	if err := builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return consumerV1.ErrorNotFound("tenant config not found")
		}
		r.log.Errorf("update tenant config failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("update tenant config failed")
	}

	return nil
}

// Delete 删除配置
func (r *tenantConfigRepo) Delete(ctx context.Context, id uint32) error {
	err := r.entClient.Client().TenantConfig.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return consumerV1.ErrorNotFound("tenant config not found")
		}
		r.log.Errorf("delete tenant config failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("delete tenant config failed")
	}

	return nil
}

// List 分页查询配置列表
func (r *tenantConfigRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListTenantConfigsResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().TenantConfig.Query()

	ret, err := r.configRepository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &consumerV1.ListTenantConfigsResponse{Total: 0, Items: nil}, nil
	}

	return &consumerV1.ListTenantConfigsResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}

// ListByCategory 按分类查询配置
func (r *tenantConfigRepo) ListByCategory(ctx context.Context, tenantID uint32, category string) ([]*consumerV1.TenantConfig, error) {
	entities, err := r.entClient.Client().TenantConfig.Query().
		Where(
			tenantconfig.TenantID(tenantID),
			tenantconfig.Category(category),
			tenantconfig.IsActive(true),
		).
		Order(ent.Asc(tenantconfig.FieldConfigKey)).
		All(ctx)

	if err != nil {
		r.log.Errorf("list tenant configs by category failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("list tenant configs by category failed")
	}

	dtos := make([]*consumerV1.TenantConfig, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, r.configMapper.ToDTO(entity))
	}

	return dtos, nil
}

// CreateHistory 创建配置变更历史
func (r *tenantConfigRepo) CreateHistory(ctx context.Context, data *consumerV1.TenantConfigHistory) error {
	if data == nil {
		return consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().TenantConfigHistory.Create().
		SetNillableTenantID(data.TenantId).
		SetNillableConfigID(data.ConfigId).
		SetNillableConfigKey(data.ConfigKey).
		SetNillableOldValue(data.OldValue).
		SetNillableNewValue(data.NewValue).
		SetNillableChangeType(r.changeTypeConverter.ToEntity(data.ChangeType)).
		SetNillableChangeReason(data.ChangeReason).
		SetNillableChangedBy(data.ChangedBy).
		SetNillableChangedByName(data.ChangedByName).
		SetCreatedAt(time.Now())

	_, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert tenant config history failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("insert tenant config history failed")
	}

	return nil
}

// ListHistory 查询配置变更历史
func (r *tenantConfigRepo) ListHistory(ctx context.Context, tenantID uint32, configID uint32, req *paginationV1.PagingRequest) (*consumerV1.ListTenantConfigHistoriesResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().TenantConfigHistory.Query().
		Where(
			tenantconfighistory.TenantID(tenantID),
			tenantconfighistory.ConfigID(configID),
		).
		Order(ent.Desc(tenantconfighistory.FieldCreatedAt))

	ret, err := r.historyRepository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &consumerV1.ListTenantConfigHistoriesResponse{Total: 0, Items: nil}, nil
	}

	return &consumerV1.ListTenantConfigHistoriesResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}
