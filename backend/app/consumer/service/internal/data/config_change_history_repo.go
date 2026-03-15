package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	// "go-wind-admin/app/consumer/service/internal/data/ent/configchangehistory" // Will be available after go generate
)

// ConfigChangeHistoryInput 配置变更历史输入
type ConfigChangeHistoryInput struct {
	TenantID     uint32
	ConfigID     uint32
	ConfigKey    string
	OldValue     *string
	NewValue     *string
	ChangeType   string
	ChangeReason *string
	ChangedBy    uint32
}

// ConfigChangeHistoryRepo 配置变更历史数据访问接口
type ConfigChangeHistoryRepo interface {
	// Create 创建变更历史记录
	Create(ctx context.Context, input *ConfigChangeHistoryInput) error
	// TODO: Uncomment after running go generate
	// ListByConfigKey 查询配置的变更历史
	// ListByConfigKey(ctx context.Context, tenantID uint32, configKey string, page, pageSize int) ([]*ent.ConfigChangeHistory, int, error)
	// GetLatestByConfigKey 获取配置的最新变更记录
	// GetLatestByConfigKey(ctx context.Context, tenantID uint32, configKey string) (*ent.ConfigChangeHistory, error)
}

type configChangeHistoryRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

// NewConfigChangeHistoryRepo 创建配置变更历史数据访问实例
func NewConfigChangeHistoryRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) ConfigChangeHistoryRepo {
	return &configChangeHistoryRepo{
		entClient: entClient,
		log:       ctx.NewLoggerHelper("consumer/data/config-change-history-repo"),
	}
}

// Create 创建变更历史记录
func (r *configChangeHistoryRepo) Create(ctx context.Context, input *ConfigChangeHistoryInput) error {
	r.log.Infof("Create: tenantID=%d, configKey=%s, changeType=%s", input.TenantID, input.ConfigKey, input.ChangeType)

	// TODO: Uncomment after running go generate
	/*
	builder := r.entClient.Client().ConfigChangeHistory.Create().
		SetTenantID(input.TenantID).
		SetConfigID(input.ConfigID).
		SetConfigKey(input.ConfigKey).
		SetChangeType(configchangehistory.ChangeType(input.ChangeType)).
		SetChangedBy(input.ChangedBy).
		SetChangedAt(time.Now())

	// 设置旧值
	if input.OldValue != nil {
		builder.SetOldValue(*input.OldValue)
	}

	// 设置新值
	if input.NewValue != nil {
		builder.SetNewValue(*input.NewValue)
	}

	// 设置变更原因
	if input.ChangeReason != nil {
		builder.SetChangeReason(*input.ChangeReason)
	}

	_, err := builder.Save(ctx)
	return err
	*/
	
	// Temporary: return nil until Ent code is generated
	r.log.Warnf("ConfigChangeHistory not yet generated - skipping history record")
	return nil
}

// TODO: Uncomment these methods after running go generate

/*
// ListByConfigKey 查询配置的变更历史
func (r *configChangeHistoryRepo) ListByConfigKey(ctx context.Context, tenantID uint32, configKey string, page, pageSize int) ([]*ent.ConfigChangeHistory, int, error) {
	r.log.Infof("ListByConfigKey: tenantID=%d, configKey=%s, page=%d, pageSize=%d", tenantID, configKey, page, pageSize)

	// TODO: Uncomment after running go generate
	query := r.entClient.Client().ConfigChangeHistory.Query().
		Where(
			configchangehistory.TenantID(tenantID),
			configchangehistory.ConfigKey(configKey),
		)

	// 查询总数
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询，按时间倒序
	histories, err := query.
		Order(ent.Desc(configchangehistory.FieldChangedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

// GetLatestByConfigKey 获取配置的最新变更记录
func (r *configChangeHistoryRepo) GetLatestByConfigKey(ctx context.Context, tenantID uint32, configKey string) (*ent.ConfigChangeHistory, error) {
	r.log.Infof("GetLatestByConfigKey: tenantID=%d, configKey=%s", tenantID, configKey)

	return r.entClient.Client().ConfigChangeHistory.Query().
		Where(
			configchangehistory.TenantID(tenantID),
			configchangehistory.ConfigKey(configKey),
		).
		Order(ent.Desc(configchangehistory.FieldChangedAt)).
		First(ctx)
}
*/
