igID uint32, req *paginationV1.PagingRequest) (*consumerV1.ListTenantConfigHistoriesResponse, error) {
	r.log.Warn("Using stub implementation - Ent code not generated yet")
	return nil, consumerV1.ErrorUnimplemented("tenant config repository not implemented - Ent code generation required")
}
d yet")
	return nil, consumerV1.ErrorUnimplemented("tenant config repository not implemented - Ent code generation required")
}

func (r *tenantConfigRepoStub) CreateHistory(ctx context.Context, data *consumerV1.TenantConfigHistory) error {
	r.log.Warn("Using stub implementation - Ent code not generated yet")
	return consumerV1.ErrorUnimplemented("tenant config repository not implemented - Ent code generation required")
}

func (r *tenantConfigRepoStub) ListHistory(ctx context.Context, tenantID uint32, confnantConfigRepoStub) List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListTenantConfigsResponse, error) {
	r.log.Warn("Using stub implementation - Ent code not generated yet")
	return nil, consumerV1.ErrorUnimplemented("tenant config repository not implemented - Ent code generation required")
}

func (r *tenantConfigRepoStub) ListByCategory(ctx context.Context, tenantID uint32, category string) ([]*consumerV1.TenantConfig, error) {
	r.log.Warn("Using stub implementation - Ent code not generatet, id uint32, data *consumerV1.TenantConfig) error {
	r.log.Warn("Using stub implementation - Ent code not generated yet")
	return consumerV1.ErrorUnimplemented("tenant config repository not implemented - Ent code generation required")
}

func (r *tenantConfigRepoStub) Delete(ctx context.Context, id uint32) error {
	r.log.Warn("Using stub implementation - Ent code not generated yet")
	return consumerV1.ErrorUnimplemented("tenant config repository not implemented - Ent code generation required")
}

func (r *tede not generated yet")
	return nil, consumerV1.ErrorUnimplemented("tenant config repository not implemented - Ent code generation required")
}

func (r *tenantConfigRepoStub) GetByKey(ctx context.Context, tenantID uint32, key string) (*consumerV1.TenantConfig, error) {
	r.log.Warn("Using stub implementation - Ent code not generated yet")
	return nil, consumerV1.ErrorUnimplemented("tenant config repository not implemented - Ent code generation required")
}

func (r *tenantConfigRepoStub) Update(ctx context.Contexper("consumer/repo/tenant-config-stub"),
	}
}

func (r *tenantConfigRepoStub) Create(ctx context.Context, data *consumerV1.TenantConfig) (*consumerV1.TenantConfig, error) {
	r.log.Warn("Using stub implementation - Ent code not generated yet")
	return nil, consumerV1.ErrorUnimplemented("tenant config repository not implemented - Ent code generation required")
}

func (r *tenantConfigRepoStub) Get(ctx context.Context, id uint32) (*consumerV1.TenantConfig, error) {
	r.log.Warn("Using stub implementation - Ent co, error)
}

type tenantConfigRepoStub struct {
	log *log.Helper
}

// NewTenantConfigRepo 创建租户配置数据访问实例
// 注意: 这是一个临时的stub实现，需要生成Ent代码后替换为完整实现
func NewTenantConfigRepo(ctx *bootstrap.Context, _ interface{}) TenantConfigRepo {
	return &tenantConfigRepoStub{
		log: ctx.NewLoggerHelory(ctx context.Context, tenantID uint32, category string) ([]*consumerV1.TenantConfig, error)
	CreateHistory(ctx context.Context, data *consumerV1.TenantConfigHistory) error
	ListHistory(ctx context.Context, tenantID uint32, configID uint32, req *paginationV1.PagingRequest) (*consumerV1.ListTenantConfigHistoriesResponse (*consumerV1.TenantConfig, error)
	Update(ctx context.Context, id uint32, data *consumerV1.TenantConfig) error
	Delete(ctx context.Context, id uint32) error
	List(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListTenantConfigsResponse, error)
	ListByCateg/gen/go/pagination/v1"
	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// TenantConfigRepo 租户配置数据访问接口
type TenantConfigRepo interface {
	Create(ctx context.Context, data *consumerV1.TenantConfig) (*consumerV1.TenantConfig, error)
	Get(ctx context.Context, id uint32) (*consumerV1.TenantConfig, error)
	GetByKey(ctx context.Context, tenantID uint32, key string)package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

