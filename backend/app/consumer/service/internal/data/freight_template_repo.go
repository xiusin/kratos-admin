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
	"go-wind-admin/app/consumer/service/internal/data/ent/freighttemplate"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// FreightTemplateRepo 运费模板数据访问接口
type FreightTemplateRepo interface {
	// Create 创建运费模板
	Create(ctx context.Context, data *consumerV1.FreightTemplate) (*consumerV1.FreightTemplate, error)

	// Get 查询运费模板
	Get(ctx context.Context, id uint32) (*consumerV1.FreightTemplate, error)

	// Update 更新运费模板
	Update(ctx context.Context, id uint32, data *consumerV1.FreightTemplate) error

	// List 分页查询运费模板
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListFreightTemplatesResponse, error)
}

type freightTemplateRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper

	mapper                   *mapper.CopierMapper[consumerV1.FreightTemplate, ent.FreightTemplate]
	calculationTypeConverter *mapper.EnumTypeConverter[consumerV1.FreightTemplate_CalculationType, freighttemplate.CalculationType]

	repository *entCrud.Repository[
		ent.FreightTemplateQuery, ent.FreightTemplateSelect,
		ent.FreightTemplateCreate, ent.FreightTemplateCreateBulk,
		ent.FreightTemplateUpdate, ent.FreightTemplateUpdateOne,
		ent.FreightTemplateDelete,
		predicate.FreightTemplate,
		consumerV1.FreightTemplate, ent.FreightTemplate,
	]
}

// NewFreightTemplateRepo 创建运费模板数据访问实例
func NewFreightTemplateRepo(
	ctx *bootstrap.Context,
	entClient *entCrud.EntClient[*ent.Client],
) FreightTemplateRepo {
	repo := &freightTemplateRepo{
		log:                      ctx.NewLoggerHelper("consumer/repo/freight-template"),
		entClient:                entClient,
		mapper:                   mapper.NewCopierMapper[consumerV1.FreightTemplate, ent.FreightTemplate](),
		calculationTypeConverter: mapper.NewEnumTypeConverter[consumerV1.FreightTemplate_CalculationType, freighttemplate.CalculationType](consumerV1.FreightTemplate_CalculationType_name, consumerV1.FreightTemplate_CalculationType_value),
	}

	repo.init()

	return repo
}

func (r *freightTemplateRepo) init() {
	r.repository = entCrud.NewRepository[
		ent.FreightTemplateQuery, ent.FreightTemplateSelect,
		ent.FreightTemplateCreate, ent.FreightTemplateCreateBulk,
		ent.FreightTemplateUpdate, ent.FreightTemplateUpdateOne,
		ent.FreightTemplateDelete,
		predicate.FreightTemplate,
		consumerV1.FreightTemplate, ent.FreightTemplate,
	](r.mapper)

	r.mapper.AppendConverters(copierutil.NewTimeStringConverterPair())
	r.mapper.AppendConverters(copierutil.NewTimeTimestamppbConverterPair())
	r.mapper.AppendConverters(r.calculationTypeConverter.NewConverterPair())
}

// Create 创建运费模板
func (r *freightTemplateRepo) Create(ctx context.Context, data *consumerV1.FreightTemplate) (*consumerV1.FreightTemplate, error) {
	if data == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().FreightTemplate.Create().
		SetNillableTenantID(data.TenantId).
		SetNillableName(data.Name).
		SetNillableCalculationType(r.calculationTypeConverter.ToEntity(data.CalculationType)).
		SetNillableFirstWeight(data.FirstWeight).
		SetNillableFirstPrice(data.FirstPrice).
		SetNillableAdditionalWeight(data.AdditionalWeight).
		SetNillableAdditionalPrice(data.AdditionalPrice).
		SetNillableIsActive(data.IsActive).
		SetNillableCreatedBy(data.CreatedBy).
		SetCreatedAt(time.Now())

	// 设置地区规则（JSON字段）
	if len(data.RegionRules) > 0 {
		regionRules := make([]map[string]interface{}, 0, len(data.RegionRules))
		for _, rule := range data.RegionRules {
			regionRules = append(regionRules, rule.AsMap())
		}
		builder.SetRegionRules(regionRules)
	}

	// 设置包邮规则（JSON字段）
	if len(data.FreeShippingRules) > 0 {
		freeShippingRules := make([]map[string]interface{}, 0, len(data.FreeShippingRules))
		for _, rule := range data.FreeShippingRules {
			freeShippingRules = append(freeShippingRules, rule.AsMap())
		}
		builder.SetFreeShippingRules(freeShippingRules)
	}

	entity, err := builder.Save(ctx)
	if err != nil {
		r.log.Errorf("insert freight template failed: %s", err.Error())
		return nil, consumerV1.ErrorInternalServerError("insert freight template failed")
	}

	return r.mapper.ToDTO(entity), nil
}

// Get 查询运费模板
func (r *freightTemplateRepo) Get(ctx context.Context, id uint32) (*consumerV1.FreightTemplate, error) {
	builder := r.entClient.Client().FreightTemplate.Query()

	dto, err := r.repository.Get(ctx, builder, nil,
		func(s *sql.Selector) {
			s.Where(sql.EQ(freighttemplate.FieldID, id))
		},
	)
	if err != nil {
		return nil, err
	}

	return dto, nil
}

// Update 更新运费模板
func (r *freightTemplateRepo) Update(ctx context.Context, id uint32, data *consumerV1.FreightTemplate) error {
	if data == nil {
		return consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().FreightTemplate.UpdateOneID(id).
		SetNillableName(data.Name).
		SetNillableCalculationType(r.calculationTypeConverter.ToEntity(data.CalculationType)).
		SetNillableFirstWeight(data.FirstWeight).
		SetNillableFirstPrice(data.FirstPrice).
		SetNillableAdditionalWeight(data.AdditionalWeight).
		SetNillableAdditionalPrice(data.AdditionalPrice).
		SetNillableIsActive(data.IsActive).
		SetNillableUpdatedBy(data.UpdatedBy).
		SetUpdatedAt(time.Now())

	// 更新地区规则（JSON字段）
	if len(data.RegionRules) > 0 {
		regionRules := make([]map[string]interface{}, 0, len(data.RegionRules))
		for _, rule := range data.RegionRules {
			regionRules = append(regionRules, rule.AsMap())
		}
		builder.SetRegionRules(regionRules)
	}

	// 更新包邮规则（JSON字段）
	if len(data.FreeShippingRules) > 0 {
		freeShippingRules := make([]map[string]interface{}, 0, len(data.FreeShippingRules))
		for _, rule := range data.FreeShippingRules {
			freeShippingRules = append(freeShippingRules, rule.AsMap())
		}
		builder.SetFreeShippingRules(freeShippingRules)
	}

	err := builder.Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return consumerV1.ErrorNotFound("freight template not found")
		}
		r.log.Errorf("update freight template failed: %s", err.Error())
		return consumerV1.ErrorInternalServerError("update freight template failed")
	}

	return nil
}

// List 分页查询运费模板
func (r *freightTemplateRepo) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListFreightTemplatesResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	builder := r.entClient.Client().FreightTemplate.Query().
		Order(ent.Desc(freighttemplate.FieldCreatedAt)) // 按创建时间倒序

	ret, err := r.repository.ListWithPaging(ctx, builder, builder.Clone(), req)
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return &consumerV1.ListFreightTemplatesResponse{Total: 0, Items: nil}, nil
	}

	return &consumerV1.ListFreightTemplatesResponse{
		Total: ret.Total,
		Items: ret.Items,
	}, nil
}
