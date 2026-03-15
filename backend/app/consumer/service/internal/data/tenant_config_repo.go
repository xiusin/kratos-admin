package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"
	"go-wind-admin/app/consumer/service/internal/data/ent/tenantconfig"
)

// TenantConfigRepo 租户配置数据访问接口
type TenantConfigRepo interface {
	// GetByKey 根据配置键查询配置
	GetByKey(ctx context.Context, tenantID uint32, configKey string) (*ent.TenantConfig, error)
	// List 分页查询配置列表
	List(ctx context.Context, tenantID uint32, page, pageSize int, predicates ...predicate.TenantConfig) ([]*ent.TenantConfig, int, error)
	// Upsert 创建或更新配置
	Upsert(ctx context.Context, config *ent.TenantConfig) (*ent.TenantConfig, error)
	// Delete 删除配置
	Delete(ctx context.Context, tenantID uint32, configKey string) error
	// BatchUpsert 批量创建或更新配置
	BatchUpsert(ctx context.Context, configs []*ent.TenantConfig) error
}

type tenantConfigRepo struct {
	entClient *entCrud.EntClient[*ent.Client]
	log       *log.Helper
}

// NewTenantConfigRepo 创建租户配置数据访问实例
func NewTenantConfigRepo(ctx *bootstrap.Context, entClient *entCrud.EntClient[*ent.Client]) TenantConfigRepo {
	return &tenantConfigRepo{
		entClient: entClient,
		log:       ctx.NewLoggerHelper("consumer/data/tenant-config-repo"),
	}
}

// GetByKey 根据配置键查询配置
func (r *tenantConfigRepo) GetByKey(ctx context.Context, tenantID uint32, configKey string) (*ent.TenantConfig, error) {
	r.log.Infof("GetByKey: tenantID=%d, configKey=%s", tenantID, configKey)

	return r.entClient.Client().TenantConfig.
		Query().
		Where(
			tenantconfig.TenantID(tenantID),
			tenantconfig.ConfigKey(configKey),
		).
		First(ctx)
}

// List 分页查询配置列表
func (r *tenantConfigRepo) List(ctx context.Context, tenantID uint32, page, pageSize int, predicates ...predicate.TenantConfig) ([]*ent.TenantConfig, int, error) {
	r.log.Infof("List: tenantID=%d, page=%d, pageSize=%d", tenantID, page, pageSize)

	query := r.entClient.Client().TenantConfig.Query().
		Where(tenantconfig.TenantID(tenantID))

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
	configs, err := query.
		Order(ent.Desc(tenantconfig.FieldCreatedAt)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return configs, total, nil
}

// Upsert 创建或更新配置
func (r *tenantConfigRepo) Upsert(ctx context.Context, config *ent.TenantConfig) (*ent.TenantConfig, error) {
	r.log.Infof("Upsert: tenantID=%d, configKey=%s", config.TenantID, config.ConfigKey)

	// 检查配置是否存在
	existing, err := r.GetByKey(ctx, *config.TenantID, config.ConfigKey)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}

	// 如果存在则更新
	if existing != nil {
		builder := r.entClient.Client().TenantConfig.UpdateOneID(existing.ID)

		// 更新配置值
		if config.ConfigValue != nil {
			builder.SetConfigValue(*config.ConfigValue)
		} else {
			builder.ClearConfigValue()
		}

		// 更新描述
		if config.Description != nil {
			builder.SetDescription(*config.Description)
		} else {
			builder.ClearDescription()
		}

		// 更新配置类型
		if config.ConfigType != "" {
			builder.SetConfigType(config.ConfigType)
		}

		// 更新是否加密
		builder.SetIsEncrypted(config.IsEncrypted)

		// 更新是否启用
		builder.SetIsActive(config.IsActive)

		// 设置更新者ID
		if config.UpdatedBy != nil {
			builder.SetUpdatedBy(*config.UpdatedBy)
		}

		if err := builder.Exec(ctx); err != nil {
			return nil, err
		}

		return r.GetByKey(ctx, *config.TenantID, config.ConfigKey)
	}

	// 如果不存在则创建
	builder := r.entClient.Client().TenantConfig.Create().
		SetConfigKey(config.ConfigKey).
		SetConfigType(config.ConfigType).
		SetIsEncrypted(config.IsEncrypted).
		SetIsActive(config.IsActive)

	// 设置租户ID
	if config.TenantID != nil {
		builder.SetTenantID(*config.TenantID)
	}

	// 设置配置值
	if config.ConfigValue != nil {
		builder.SetConfigValue(*config.ConfigValue)
	}

	// 设置描述
	if config.Description != nil {
		builder.SetDescription(*config.Description)
	}

	// 设置创建者ID
	if config.CreatedBy != nil {
		builder.SetCreatedBy(*config.CreatedBy)
	}

	return builder.Save(ctx)
}

// Delete 删除配置
func (r *tenantConfigRepo) Delete(ctx context.Context, tenantID uint32, configKey string) error {
	r.log.Infof("Delete: tenantID=%d, configKey=%s", tenantID, configKey)

	_, err := r.entClient.Client().TenantConfig.
		Delete().
		Where(
			tenantconfig.TenantID(tenantID),
			tenantconfig.ConfigKey(configKey),
		).
		Exec(ctx)

	return err
}

// BatchUpsert 批量创建或更新配置
func (r *tenantConfigRepo) BatchUpsert(ctx context.Context, configs []*ent.TenantConfig) error {
	r.log.Infof("BatchUpsert: count=%d", len(configs))

	// 使用事务批量处理
	tx, err := r.entClient.Client().Tx(ctx)
	if err != nil {
		return err
	}

	for _, config := range configs {
		// 检查配置是否存在
		existing, err := tx.TenantConfig.Query().
			Where(
				tenantconfig.TenantID(*config.TenantID),
				tenantconfig.ConfigKey(config.ConfigKey),
			).
			First(ctx)

		if err != nil && !ent.IsNotFound(err) {
			_ = tx.Rollback()
			return err
		}

		// 如果存在则更新
		if existing != nil {
			builder := tx.TenantConfig.UpdateOneID(existing.ID)

			if config.ConfigValue != nil {
				builder.SetConfigValue(*config.ConfigValue)
			} else {
				builder.ClearConfigValue()
			}

			if config.Description != nil {
				builder.SetDescription(*config.Description)
			} else {
				builder.ClearDescription()
			}

			if config.ConfigType != "" {
				builder.SetConfigType(config.ConfigType)
			}

			builder.SetIsEncrypted(config.IsEncrypted).
				SetIsActive(config.IsActive)

			if config.UpdatedBy != nil {
				builder.SetUpdatedBy(*config.UpdatedBy)
			}

			if err := builder.Exec(ctx); err != nil {
				_ = tx.Rollback()
				return err
			}
		} else {
			// 如果不存在则创建
			builder := tx.TenantConfig.Create().
				SetConfigKey(config.ConfigKey).
				SetConfigType(config.ConfigType).
				SetIsEncrypted(config.IsEncrypted).
				SetIsActive(config.IsActive)

			if config.TenantID != nil {
				builder.SetTenantID(*config.TenantID)
			}

			if config.ConfigValue != nil {
				builder.SetConfigValue(*config.ConfigValue)
			}

			if config.Description != nil {
				builder.SetDescription(*config.Description)
			}

			if config.CreatedBy != nil {
				builder.SetCreatedBy(*config.CreatedBy)
			}

			if _, err := builder.Save(ctx); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}
