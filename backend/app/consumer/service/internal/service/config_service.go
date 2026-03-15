package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/predicate"
	"go-wind-admin/app/consumer/service/internal/data/ent/tenantconfig"
)

// ConfigService 配置管理服务
type ConfigService struct {
	consumerV1.UnimplementedConfigServiceServer

	configRepo        data.TenantConfigRepo
	configCache       data.TenantConfigCache
	configHistoryRepo data.ConfigChangeHistoryRepo
	log               *log.Helper
}

// NewConfigService 创建配置管理服务实例
func NewConfigService(
	ctx *bootstrap.Context,
	configRepo data.TenantConfigRepo,
	configCache data.TenantConfigCache,
	configHistoryRepo data.ConfigChangeHistoryRepo,
) *ConfigService {
	return &ConfigService{
		configRepo:        configRepo,
		configCache:       configCache,
		configHistoryRepo: configHistoryRepo,
		log:               ctx.NewLoggerHelper("consumer/service/config-service"),
	}
}

// GetConfig 获取配置
func (s *ConfigService) GetConfig(ctx context.Context, req *consumerV1.GetConfigRequest) (*consumerV1.TenantConfig, error) {
	s.log.Infof("GetConfig: config_key=%s", req.GetConfigKey())

	// TODO: 从context中获取当前租户ID
	tenantID := uint32(1) // 临时硬编码

	// 先从缓存获取
	cachedConfig, err := s.configCache.Get(ctx, tenantID, req.GetConfigKey())
	if err != nil {
		s.log.Warnf("get config from cache failed: %v", err)
	}
	if cachedConfig != nil {
		s.log.Debugf("config found in cache: config_key=%s", req.GetConfigKey())
		return s.toProtoConfig(cachedConfig), nil
	}

	// 从数据库获取
	config, err := s.configRepo.GetByKey(ctx, tenantID, req.GetConfigKey())
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.NotFound("CONFIG_NOT_FOUND", "config not found")
		}
		s.log.Errorf("get config from db failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "failed to get config")
	}

	// 写入缓存
	if err := s.configCache.Set(ctx, config); err != nil {
		s.log.Warnf("set config cache failed: %v", err)
	}

	return s.toProtoConfig(config), nil
}

// ListConfigs 获取配置列表
func (s *ConfigService) ListConfigs(ctx context.Context, req *consumerV1.ListConfigsRequest) (*consumerV1.ListConfigsResponse, error) {
	s.log.Infof("ListConfigs: config_type=%v, is_active=%v", req.GetConfigType(), req.GetIsActive())

	// TODO: 从context中获取当前租户ID
	tenantID := uint32(1) // 临时硬编码

	// 构建查询条件
	var predicates []predicate.TenantConfig

	// 按配置类型过滤
	if req.GetConfigType() != consumerV1.ConfigType_CONFIG_TYPE_UNSPECIFIED {
		configType := s.toEntConfigType(req.GetConfigType())
		predicates = append(predicates, tenantconfig.ConfigTypeEQ(configType))
	}

	// 按是否启用过滤
	if req.IsActive != nil {
		predicates = append(predicates, tenantconfig.IsActiveEQ(req.GetIsActive()))
	}

	// 分页参数
	page := int(req.GetPaging().GetPage())
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.GetPaging().GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 查询配置列表
	configs, total, err := s.configRepo.List(ctx, tenantID, page, pageSize, predicates...)
	if err != nil {
		s.log.Errorf("list configs failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "failed to list configs")
	}
	_ = total // TODO: 返回总数给前端

	// 转换为Proto格式
	items := make([]*consumerV1.TenantConfig, 0, len(configs))
	for _, config := range configs {
		items = append(items, s.toProtoConfig(config))
	}

	return &consumerV1.ListConfigsResponse{
		Items:  items,
		Paging: &paginationV1.PagingResponse{},
	}, nil
}

// UpdateConfig 更新配置
func (s *ConfigService) UpdateConfig(ctx context.Context, req *consumerV1.UpdateConfigRequest) (*emptypb.Empty, error) {
	s.log.Infof("UpdateConfig: config_key=%s", req.GetConfigKey())

	// TODO: 从context中获取当前租户ID和用户ID
	tenantID := uint32(1)      // 临时硬编码
	currentUserID := uint32(1) // 临时硬编码

	// 验证配置值的格式和有效性 (Requirement 14.5)
	if req.ConfigValue != nil {
		if err := s.validateConfigValue(req.GetConfigKey(), req.GetConfigValue(), req.GetConfigType()); err != nil {
			s.log.Warnf("config validation failed: %v", err)
			return nil, err
		}
	}

	// 获取旧配置（用于记录变更历史）
	oldConfig, err := s.configRepo.GetByKey(ctx, tenantID, req.GetConfigKey())
	if err != nil && !ent.IsNotFound(err) {
		s.log.Errorf("get old config failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "failed to get old config")
	}

	// 构建配置对象
	config := &ent.TenantConfig{
		TenantID:    &tenantID,
		ConfigKey:   req.GetConfigKey(),
		ConfigType:  s.toEntConfigType(req.GetConfigType()),
		IsEncrypted: req.GetIsEncrypted(),
		IsActive:    req.GetIsActive(),
		UpdatedBy:   &currentUserID,
	}

	// 设置配置值
	if req.ConfigValue != nil {
		configValue := req.GetConfigValue()
		config.ConfigValue = &configValue
	}

	// 设置描述
	if req.Description != nil {
		description := req.GetDescription()
		config.Description = &description
	}

	// 创建或更新配置
	updatedConfig, err := s.configRepo.Upsert(ctx, config)
	if err != nil {
		s.log.Errorf("upsert config failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "failed to update config")
	}

	// 记录配置变更历史 (Requirement 14.6)
	if err := s.recordConfigChange(ctx, tenantID, currentUserID, oldConfig, updatedConfig); err != nil {
		s.log.Warnf("record config change history failed: %v", err)
		// 不影响主流程，只记录警告
	}

	// 删除缓存（支持热更新 Requirement 14.2）
	if err := s.configCache.Delete(ctx, tenantID, req.GetConfigKey()); err != nil {
		s.log.Warnf("delete config cache failed: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// BatchUpdateConfigs 批量更新配置
func (s *ConfigService) BatchUpdateConfigs(ctx context.Context, req *consumerV1.BatchUpdateConfigsRequest) (*emptypb.Empty, error) {
	s.log.Infof("BatchUpdateConfigs: count=%d", len(req.GetConfigs()))

	// TODO: 从context中获取当前租户ID和用户ID
	tenantID := uint32(1)      // 临时硬编码
	currentUserID := uint32(1) // 临时硬编码

	// 构建配置对象列表
	configs := make([]*ent.TenantConfig, 0, len(req.GetConfigs()))
	for _, reqConfig := range req.GetConfigs() {
		config := &ent.TenantConfig{
			TenantID:    &tenantID,
			ConfigKey:   reqConfig.GetConfigKey(),
			ConfigType:  s.toEntConfigType(reqConfig.GetConfigType()),
			IsEncrypted: reqConfig.GetIsEncrypted(),
			IsActive:    reqConfig.GetIsActive(),
			UpdatedBy:   &currentUserID,
		}

		// 设置配置值
		if reqConfig.ConfigValue != nil {
			configValue := reqConfig.GetConfigValue()
			config.ConfigValue = &configValue
		}

		// 设置描述
		if reqConfig.Description != nil {
			description := reqConfig.GetDescription()
			config.Description = &description
		}

		configs = append(configs, config)
	}

	// 批量创建或更新配置
	if err := s.configRepo.BatchUpsert(ctx, configs); err != nil {
		s.log.Errorf("batch upsert configs failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "failed to batch update configs")
	}

	// 删除租户所有配置缓存
	if err := s.configCache.DeleteByTenant(ctx, tenantID); err != nil {
		s.log.Warnf("delete tenant config cache failed: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// DeleteConfig 删除配置
func (s *ConfigService) DeleteConfig(ctx context.Context, req *consumerV1.DeleteConfigRequest) (*emptypb.Empty, error) {
	s.log.Infof("DeleteConfig: config_key=%s", req.GetConfigKey())

	// TODO: 从context中获取当前租户ID
	tenantID := uint32(1) // 临时硬编码

	// 删除配置
	if err := s.configRepo.Delete(ctx, tenantID, req.GetConfigKey()); err != nil {
		s.log.Errorf("delete config failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "failed to delete config")
	}

	// 删除缓存
	if err := s.configCache.Delete(ctx, tenantID, req.GetConfigKey()); err != nil {
		s.log.Warnf("delete config cache failed: %v", err)
	}

	return &emptypb.Empty{}, nil
}

// toProtoConfig 转换为Proto格式
func (s *ConfigService) toProtoConfig(config *ent.TenantConfig) *consumerV1.TenantConfig {
	result := &consumerV1.TenantConfig{
		Id:          func() *uint64 { v := uint64(config.ID); return &v }(),
		ConfigKey:   &config.ConfigKey,
		ConfigType:  s.toProtoConfigType(config.ConfigType),
		IsEncrypted: &config.IsEncrypted,
		IsActive:    &config.IsActive,
	}

	// 设置租户ID
	if config.TenantID != nil {
		result.TenantId = config.TenantID
	}

	// 设置配置值
	if config.ConfigValue != nil {
		result.ConfigValue = config.ConfigValue
	}

	// 设置描述
	if config.Description != nil {
		result.Description = config.Description
	}

	// 设置创建者ID
	if config.CreatedBy != nil {
		result.CreatedBy = config.CreatedBy
	}

	// 设置更新者ID
	if config.UpdatedBy != nil {
		result.UpdatedBy = config.UpdatedBy
	}

	// 设置创建时间
	if config.CreatedAt != nil {
		result.CreatedAt = timestamppb.New(*config.CreatedAt)
	}

	// 设置更新时间
	if config.UpdatedAt != nil {
		result.UpdatedAt = timestamppb.New(*config.UpdatedAt)
	}

	return result
}

// toProtoConfigType 转换为Proto配置类型
func (s *ConfigService) toProtoConfigType(configType tenantconfig.ConfigType) *consumerV1.ConfigType {
	var result consumerV1.ConfigType
	switch configType {
	case tenantconfig.ConfigTypeSMS:
		result = consumerV1.ConfigType_SMS
	case tenantconfig.ConfigTypePayment:
		result = consumerV1.ConfigType_PAYMENT
	case tenantconfig.ConfigTypeOSS:
		result = consumerV1.ConfigType_OSS
	case tenantconfig.ConfigTypeWechat:
		result = consumerV1.ConfigType_WECHAT
	case tenantconfig.ConfigTypeLogistics:
		result = consumerV1.ConfigType_LOGISTICS
	case tenantconfig.ConfigTypeFreight:
		result = consumerV1.ConfigType_FREIGHT
	case tenantconfig.ConfigTypeSystem:
		result = consumerV1.ConfigType_SYSTEM
	default:
		result = consumerV1.ConfigType_CONFIG_TYPE_UNSPECIFIED
	}
	return &result
}

// toEntConfigType 转换为Ent配置类型
func (s *ConfigService) toEntConfigType(configType consumerV1.ConfigType) tenantconfig.ConfigType {
	switch configType {
	case consumerV1.ConfigType_SMS:
		return tenantconfig.ConfigTypeSMS
	case consumerV1.ConfigType_PAYMENT:
		return tenantconfig.ConfigTypePayment
	case consumerV1.ConfigType_OSS:
		return tenantconfig.ConfigTypeOSS
	case consumerV1.ConfigType_WECHAT:
		return tenantconfig.ConfigTypeWechat
	case consumerV1.ConfigType_LOGISTICS:
		return tenantconfig.ConfigTypeLogistics
	case consumerV1.ConfigType_FREIGHT:
		return tenantconfig.ConfigTypeFreight
	case consumerV1.ConfigType_SYSTEM:
		return tenantconfig.ConfigTypeSystem
	default:
		return tenantconfig.ConfigTypeSystem
	}
}

// validateConfigValue 验证配置值的格式和有效性
func (s *ConfigService) validateConfigValue(configKey, configValue string, configType consumerV1.ConfigType) error {
	// 基本验证：配置值不能为空（除非是删除操作）
	if configValue == "" {
		return nil // 允许空值（用于删除配置）
	}

	// 根据配置类型进行特定验证
	switch configType {
	case consumerV1.ConfigType_SMS:
		// 短信配置验证：应该包含必要的字段（如 access_key, secret_key）
		// 这里简化处理，实际应该解析JSON并验证字段
		if len(configValue) < 10 {
			return errors.BadRequest("INVALID_CONFIG_VALUE", "SMS config value too short")
		}

	case consumerV1.ConfigType_PAYMENT:
		// 支付配置验证
		if len(configValue) < 10 {
			return errors.BadRequest("INVALID_CONFIG_VALUE", "Payment config value too short")
		}

	case consumerV1.ConfigType_OSS:
		// OSS配置验证
		if len(configValue) < 10 {
			return errors.BadRequest("INVALID_CONFIG_VALUE", "OSS config value too short")
		}

	case consumerV1.ConfigType_WECHAT:
		// 微信配置验证
		if len(configValue) < 10 {
			return errors.BadRequest("INVALID_CONFIG_VALUE", "Wechat config value too short")
		}

	case consumerV1.ConfigType_LOGISTICS:
		// 物流配置验证
		if len(configValue) < 10 {
			return errors.BadRequest("INVALID_CONFIG_VALUE", "Logistics config value too short")
		}

	case consumerV1.ConfigType_FREIGHT:
		// 运费配置验证
		if len(configValue) < 10 {
			return errors.BadRequest("INVALID_CONFIG_VALUE", "Freight config value too short")
		}

	case consumerV1.ConfigType_SYSTEM:
		// 系统配置验证
		// 允许任何值
	}

	// 配置值长度限制
	if len(configValue) > 5000 {
		return errors.BadRequest("INVALID_CONFIG_VALUE", "config value too long (max 5000 characters)")
	}

	return nil
}

// recordConfigChange 记录配置变更历史 (Requirement 14.6)
func (s *ConfigService) recordConfigChange(ctx context.Context, tenantID, changedBy uint32, oldConfig, newConfig *ent.TenantConfig) error {
	// 确定变更类型
	var changeType string
	var oldValue, newValue *string

	if oldConfig == nil {
		// 新建配置
		changeType = "CREATE"
		if newConfig.ConfigValue != nil {
			newValue = newConfig.ConfigValue
		}
	} else {
		// 更新配置
		changeType = "UPDATE"
		if oldConfig.ConfigValue != nil {
			oldValue = oldConfig.ConfigValue
		}
		if newConfig.ConfigValue != nil {
			newValue = newConfig.ConfigValue
		}
	}

	// 创建变更历史记录
	return s.configHistoryRepo.Create(ctx, &data.ConfigChangeHistoryInput{
		TenantID:   tenantID,
		ConfigID:   newConfig.ID,
		ConfigKey:  newConfig.ConfigKey,
		OldValue:   oldValue,
		NewValue:   newValue,
		ChangeType: changeType,
		ChangedBy:  changedBy,
	})
}
