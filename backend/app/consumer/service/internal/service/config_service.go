package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	paginationV1 "go-wind-admin/api/gen/go/pagination/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/data/ent/tenantconfig"
)

// ConfigService 配置管理服务
type ConfigService struct {
	consumerV1.UnimplementedConfigServiceServer

	configRepo  data.TenantConfigRepo
	configCache data.TenantConfigCache
	log         *log.Helper
}

// NewConfigService 创建配置管理服务实例
func NewConfigService(
	ctx *bootstrap.Context,
	configRepo data.TenantConfigRepo,
	configCache data.TenantConfigCache,
) *ConfigService {
	return &ConfigService{
		configRepo:  configRepo,
		configCache: configCache,
		log:         ctx.NewLoggerHelper("consumer/service/config-service"),
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
	var predicates []func(*ent.TenantConfigQuery)

	// 按配置类型过滤
	if req.GetConfigType() != consumerV1.ConfigType_CONFIG_TYPE_UNSPECIFIED {
		configType := s.toEntConfigType(req.GetConfigType())
		predicates = append(predicates, func(q *ent.TenantConfigQuery) {
			q.Where(tenantconfig.ConfigType(configType))
		})
	}

	// 按是否启用过滤
	if req.IsActive != nil {
		isActive := req.GetIsActive()
		predicates = append(predicates, func(q *ent.TenantConfigQuery) {
			q.Where(tenantconfig.IsActive(isActive))
		})
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

	// 转换为 predicate.TenantConfig
	var entPredicates []func(*ent.TenantConfigQuery)
	for _, p := range predicates {
		entPredicates = append(entPredicates, p)
	}

	// 查询配置列表
	configs, total, err := s.configRepo.List(ctx, tenantID, page, pageSize, entPredicates...)
	if err != nil {
		s.log.Errorf("list configs failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "failed to list configs")
	}

	// 转换为Proto格式
	items := make([]*consumerV1.TenantConfig, 0, len(configs))
	for _, config := range configs {
		items = append(items, s.toProtoConfig(config))
	}

	return &consumerV1.ListConfigsResponse{
		Items: items,
		Paging: &paginationV1.PagingResponse{
			Page:     int32(page),
			PageSize: int32(pageSize),
			Total:    int32(total),
		},
	}, nil
}

// UpdateConfig 更新配置
func (s *ConfigService) UpdateConfig(ctx context.Context, req *consumerV1.UpdateConfigRequest) (*emptypb.Empty, error) {
	s.log.Infof("UpdateConfig: config_key=%s", req.GetConfigKey())

	// TODO: 从context中获取当前租户ID和用户ID
	tenantID := uint32(1)      // 临时硬编码
	currentUserID := uint32(1) // 临时硬编码

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
	_, err := s.configRepo.Upsert(ctx, config)
	if err != nil {
		s.log.Errorf("upsert config failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "failed to update config")
	}

	// 删除缓存
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
	case tenantconfig.ConfigTypePAYMENT:
		result = consumerV1.ConfigType_PAYMENT
	case tenantconfig.ConfigTypeOSS:
		result = consumerV1.ConfigType_OSS
	case tenantconfig.ConfigTypeWECHAT:
		result = consumerV1.ConfigType_WECHAT
	case tenantconfig.ConfigTypeLOGISTICS:
		result = consumerV1.ConfigType_LOGISTICS
	case tenantconfig.ConfigTypeFREIGHT:
		result = consumerV1.ConfigType_FREIGHT
	case tenantconfig.ConfigTypeSYSTEM:
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
		return tenantconfig.ConfigTypePAYMENT
	case consumerV1.ConfigType_OSS:
		return tenantconfig.ConfigTypeOSS
	case consumerV1.ConfigType_WECHAT:
		return tenantconfig.ConfigTypeWECHAT
	case consumerV1.ConfigType_LOGISTICS:
		return tenantconfig.ConfigTypeLOGISTICS
	case consumerV1.ConfigType_FREIGHT:
		return tenantconfig.ConfigTypeFREIGHT
	case consumerV1.ConfigType_SYSTEM:
		return tenantconfig.ConfigTypeSYSTEM
	default:
		return tenantconfig.ConfigTypeSYSTEM
	}
}
