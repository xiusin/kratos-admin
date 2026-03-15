package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/freighttemplate"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"
)

// FreightTemplateRepo 运费模板数据访问接口
type FreightTemplateRepo interface {
	// Create 创建运费模板
	Create(ctx context.Context, template *ent.FreightTemplate) (*ent.FreightTemplate, error)
	// Get 查询运费模板
	Get(ctx context.Context, id uint32) (*ent.FreightTemplate, error)
	// Update 更新运费模板
	Update(ctx context.Context, id uint32, template *ent.FreightTemplate) error
	// List 分页查询运费模板列表
	List(ctx context.Context, page, pageSize int, predicates ...predicate.FreightTemplate) ([]*ent.FreightTemplate, int, error)
}

type freightTemplateRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

// NewFreightTemplateRepo 创建运费模板数据访问实例
func NewFreightTemplateRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) FreightTemplateRepo {
	return &freightTemplateRepo{
		entClient: entClient,
		log:       ctx.NewLoggerHelper("consumer/data/freight-template-repo"),
	}
}

// Create 创建运费模板
func (r *freightTemplateRepo) Create(ctx context.Context, template *ent.FreightTemplate) (*ent.FreightTemplate, error) {
	r.log.Infof("Create freight template: name=%s", template.Name)

	builder := r.entClient.Client().FreightTemplate.Create().
		SetName(template.Name).
		SetCalculationType(template.CalculationType).
		SetIsActive(template.IsActive)

	// 设置租户ID
	if template.TenantID != nil {
		builder.SetTenantID(*template.TenantID)
	}

	// 设置创建者ID
	if template.CreatedBy != nil {
		builder.SetCreatedBy(*template.CreatedBy)
	}

	// 设置首重和首重价格
	if template.FirstWeight != nil {
		builder.SetFirstWeight(*template.FirstWeight)
	}
	if template.FirstPrice != nil {
		builder.SetFirstPrice(*template.FirstPrice)
	}

	// 设置续重和续重价格
	if template.AdditionalWeight != nil {
		builder.SetAdditionalWeight(*template.AdditionalWeight)
	}
	if template.AdditionalPrice != nil {
		builder.SetAdditionalPrice(*template.AdditionalPrice)
	}

	// 设置地区规则
	if template.RegionRules != nil {
		builder.SetRegionRules(template.RegionRules)
	}

	// 设置包邮规则
	if template.FreeShippingRules != nil {
		builder.SetFreeShippingRules(template.FreeShippingRules)
	}

	return builder.Save(ctx)
}

// Get 查询运费模板
func (r *freightTemplateRepo) Get(ctx context.Context, id uint32) (*ent.FreightTemplate, error) {
	r.log.Infof("Get freight template: id=%d", id)

	return r.entClient.Client().FreightTemplate.
		Query().
		Where(freighttemplate.ID(id)).
		First(ctx)
}

// Update 更新运费模板
func (r *freightTemplateRepo) Update(ctx context.Context, id uint32, template *ent.FreightTemplate) error {
	r.log.Infof("Update freight template: id=%d", id)

	builder := r.entClient.Client().FreightTemplate.UpdateOneID(id)

	// 更新名称
	if template.Name != "" {
		builder.SetName(template.Name)
	}

	// 更新计算方式
	if template.CalculationType != "" {
		builder.SetCalculationType(template.CalculationType)
	}

	// 更新首重和首重价格
	if template.FirstWeight != nil {
		builder.SetFirstWeight(*template.FirstWeight)
	}
	if template.FirstPrice != nil {
		builder.SetFirstPrice(*template.FirstPrice)
	}

	// 更新续重和续重价格
	if template.AdditionalWeight != nil {
		builder.SetAdditionalWeight(*template.AdditionalWeight)
	}
	if template.AdditionalPrice != nil {
		builder.SetAdditionalPrice(*template.AdditionalPrice)
	}

	// 更新地区规则
	if template.RegionRules != nil {
		builder.SetRegionRules(template.RegionRules)
	}

	// 更新包邮规则
	if template.FreeShippingRules != nil {
		builder.SetFreeShippingRules(template.FreeShippingRules)
	}

	// 更新是否启用
	builder.SetIsActive(template.IsActive)

	// 设置更新者ID
	if template.UpdatedBy != nil {
		builder.SetUpdatedBy(*template.UpdatedBy)
	}

	return builder.Exec(ctx)
}

// List 分页查询运费模板列表
func (r *freightTemplateRepo) List(ctx context.Context, page, pageSize int, predicates ...predicate.FreightTemplate) ([]*ent.FreightTemplate, int, error) {
	r.log.Infof("List freight templates: page=%d, pageSize=%d", page, pageSize)

	query := r.entClient.Client().FreightTemplate.Query()

	// 应用过滤条件
	if len(predicates) > 0 {
		query.Where(predicates...)
	}

	// 查询总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	templates, err := query.
		Order(ent.Desc(freighttemplate.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}
