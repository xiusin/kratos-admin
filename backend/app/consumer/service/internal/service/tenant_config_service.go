package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"

	paginationV1 "github.com/tx7do/go-crud/api/gen/go/pagination/v1"
	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/config"
)

const (
	// 配置缓存前缀
	configCachePrefix = "tenant:config:"
	// 配置缓存过期时间（1小时）
	configCacheTTL = time.Hour
)

// TenantConfigService 租户配置服务
type TenantConfigService struct {
	consumerV1.UnimplementedTenantConfigServiceServer

	log       *log.Helper
	repo      data.TenantConfigRepo
	redis     *redis.Client
	encryptor *config.Encryptor
}

// NewTenantConfigService 创建租户配置服务实例
func NewTenantConfigService(
	ctx *bootstrap.Context,
	repo data.TenantConfigRepo,
	redisClient *redis.Client,
	encryptionKey string,
) (*TenantConfigService, error) {
	encryptor, err := config.NewEncryptor(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	return &TenantConfigService{
		log:       ctx.NewLoggerHelper("consumer/service/tenant-config"),
		repo:      repo,
		redis:     redisClient,
		encryptor: encryptor,
	}, nil
}

// GetConfig 查询配置
func (s *TenantConfigService) GetConfig(ctx context.Context, req *consumerV1.GetTenantConfigRequest) (*consumerV1.TenantConfig, error) {
	if req.ConfigKey == nil || *req.ConfigKey == "" {
		return nil, consumerV1.ErrorBadRequest("config_key is required")
	}

	tenantID := s.getTenantIDFromContext(ctx)

	// 1. 尝试从缓存获取
	cacheKey := s.buildCacheKey(tenantID, *req.ConfigKey)
	cached, err := s.getFromCache(ctx, cacheKey)
	if err == nil && cached != nil {
		s.log.Debugf("config cache hit: %s", *req.ConfigKey)
		return cached, nil
	}

	// 2. 从数据库查询
	cfg, err := s.repo.GetByKey(ctx, tenantID, *req.ConfigKey)
	if err != nil {
		return nil, err
	}

	// 3. 解密敏感配置
	if cfg.IsEncrypted != nil && *cfg.IsEncrypted && cfg.ConfigValue != nil {
		decrypted, err := s.encryptor.Decrypt(*cfg.ConfigValue)
		if err != nil {
			s.log.Errorf("failed to decrypt config: %s", err.Error())
			return nil, consumerV1.ErrorInternalServerError("failed to decrypt config")
		}
		cfg.ConfigValue = &decrypted
	}

	// 4. 写入缓存
	if err := s.setToCache(ctx, cacheKey, cfg); err != nil {
		s.log.Warnf("failed to cache config: %s", err.Error())
	}

	return cfg, nil
}

// UpdateConfig 更新配置
func (s *TenantConfigService) UpdateConfig(ctx context.Context, req *consumerV1.UpdateTenantConfigRequest) (*emptypb.Empty, error) {
	if req.Id == nil {
		return nil, consumerV1.ErrorBadRequest("id is required")
	}

	tenantID := s.getTenantIDFromContext(ctx)
	operatorID := s.getOperatorIDFromContext(ctx)

	// 1. 查询旧配置
	oldCfg, err := s.repo.Get(ctx, *req.Id)
	if err != nil {
		return nil, err
	}

	// 验证租户ID
	if oldCfg.TenantId == nil || *oldCfg.TenantId != tenantID {
		return nil, consumerV1.ErrorForbidden("access denied")
	}

	// 2. 验证配置格式和有效性
	if err := s.validateConfig(req.Data); err != nil {
		return nil, err
	}

	// 3. 加密敏感配置
	if req.Data.IsEncrypted != nil && *req.Data.IsEncrypted && req.Data.ConfigValue != nil {
		encrypted, err := s.encryptor.Encrypt(*req.Data.ConfigValue)
		if err != nil {
			s.log.Errorf("failed to encrypt config: %s", err.Error())
			return nil, consumerV1.ErrorInternalServerError("failed to encrypt config")
		}
		req.Data.ConfigValue = &encrypted
	}

	// 4. 更新配置
	if err := s.repo.Update(ctx, *req.Id, req.Data); err != nil {
		return nil, err
	}

	// 5. 记录变更历史
	history := &consumerV1.TenantConfigHistory{
		TenantId:       &tenantID,
		ConfigId:       req.Id,
		ConfigKey:      oldCfg.ConfigKey,
		OldValue:       oldCfg.ConfigValue,
		NewValue:       req.Data.ConfigValue,
		ChangeType:     consumerV1.TenantConfigHistory_UPDATE.Enum(),
		ChangeReason:   req.ChangeReason,
		ChangedBy:      &operatorID,
		ChangedByName:  req.ChangedByName,
	}
	if err := s.repo.CreateHistory(ctx, history); err != nil {
		s.log.Warnf("failed to create config history: %s", err.Error())
	}

	// 6. 清除缓存（实现热更新）
	if oldCfg.ConfigKey != nil {
		cacheKey := s.buildCacheKey(tenantID, *oldCfg.ConfigKey)
		if err := s.deleteFromCache(ctx, cacheKey); err != nil {
			s.log.Warnf("failed to delete config cache: %s", err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

// CreateConfig 创建配置
func (s *TenantConfigService) CreateConfig(ctx context.Context, req *consumerV1.CreateTenantConfigRequest) (*consumerV1.TenantConfig, error) {
	if req.Data == nil {
		return nil, consumerV1.ErrorBadRequest("data is required")
	}

	tenantID := s.getTenantIDFromContext(ctx)
	operatorID := s.getOperatorIDFromContext(ctx)

	req.Data.TenantId = &tenantID

	// 1. 验证配置格式和有效性
	if err := s.validateConfig(req.Data); err != nil {
		return nil, err
	}

	// 2. 加密敏感配置
	if req.Data.IsEncrypted != nil && *req.Data.IsEncrypted && req.Data.ConfigValue != nil {
		encrypted, err := s.encryptor.Encrypt(*req.Data.ConfigValue)
		if err != nil {
			s.log.Errorf("failed to encrypt config: %s", err.Error())
			return nil, consumerV1.ErrorInternalServerError("failed to encrypt config")
		}
		req.Data.ConfigValue = &encrypted
	}

	// 3. 创建配置
	cfg, err := s.repo.Create(ctx, req.Data)
	if err != nil {
		return nil, err
	}

	// 4. 记录变更历史
	history := &consumerV1.TenantConfigHistory{
		TenantId:      &tenantID,
		ConfigId:      cfg.Id,
		ConfigKey:     cfg.ConfigKey,
		NewValue:      cfg.ConfigValue,
		ChangeType:    consumerV1.TenantConfigHistory_CREATE.Enum(),
		ChangeReason:  req.ChangeReason,
		ChangedBy:     &operatorID,
		ChangedByName: req.ChangedByName,
	}
	if err := s.repo.CreateHistory(ctx, history); err != nil {
		s.log.Warnf("failed to create config history: %s", err.Error())
	}

	return cfg, nil
}

// DeleteConfig 删除配置
func (s *TenantConfigService) DeleteConfig(ctx context.Context, req *consumerV1.DeleteTenantConfigRequest) (*emptypb.Empty, error) {
	if req.Id == nil {
		return nil, consumerV1.ErrorBadRequest("id is required")
	}

	tenantID := s.getTenantIDFromContext(ctx)
	operatorID := s.getOperatorIDFromContext(ctx)

	// 1. 查询配置
	cfg, err := s.repo.Get(ctx, *req.Id)
	if err != nil {
		return nil, err
	}

	// 验证租户ID
	if cfg.TenantId == nil || *cfg.TenantId != tenantID {
		return nil, consumerV1.ErrorForbidden("access denied")
	}

	// 2. 删除配置
	if err := s.repo.Delete(ctx, *req.Id); err != nil {
		return nil, err
	}

	// 3. 记录变更历史
	history := &consumerV1.TenantConfigHistory{
		TenantId:      &tenantID,
		ConfigId:      req.Id,
		ConfigKey:     cfg.ConfigKey,
		OldValue:      cfg.ConfigValue,
		ChangeType:    consumerV1.TenantConfigHistory_DELETE.Enum(),
		ChangeReason:  req.ChangeReason,
		ChangedBy:     &operatorID,
		ChangedByName: req.ChangedByName,
	}
	if err := s.repo.CreateHistory(ctx, history); err != nil {
		s.log.Warnf("failed to create config history: %s", err.Error())
	}

	// 4. 清除缓存
	if cfg.ConfigKey != nil {
		cacheKey := s.buildCacheKey(tenantID, *cfg.ConfigKey)
		if err := s.deleteFromCache(ctx, cacheKey); err != nil {
			s.log.Warnf("failed to delete config cache: %s", err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

// ListConfigs 查询配置列表
func (s *TenantConfigService) ListConfigs(ctx context.Context, req *paginationV1.PagingRequest) (*consumerV1.ListTenantConfigsResponse, error) {
	return s.repo.List(ctx, req)
}

// ListConfigsByCategory 按分类查询配置
func (s *TenantConfigService) ListConfigsByCategory(ctx context.Context, req *consumerV1.ListTenantConfigsByCategoryRequest) (*consumerV1.ListTenantConfigsByCategoryResponse, error) {
	if req.Category == nil || *req.Category == "" {
		return nil, consumerV1.ErrorBadRequest("category is required")
	}

	tenantID := s.getTenantIDFromContext(ctx)

	configs, err := s.repo.ListByCategory(ctx, tenantID, *req.Category)
	if err != nil {
		return nil, err
	}

	// 解密敏感配置
	for _, cfg := range configs {
		if cfg.IsEncrypted != nil && *cfg.IsEncrypted && cfg.ConfigValue != nil {
			decrypted, err := s.encryptor.Decrypt(*cfg.ConfigValue)
			if err != nil {
				s.log.Errorf("failed to decrypt config: %s", err.Error())
				continue
			}
			cfg.ConfigValue = &decrypted
		}
	}

	return &consumerV1.ListTenantConfigsByCategoryResponse{
		Items: configs,
	}, nil
}

// ListConfigHistory 查询配置变更历史
func (s *TenantConfigService) ListConfigHistory(ctx context.Context, req *consumerV1.ListTenantConfigHistoryRequest) (*consumerV1.ListTenantConfigHistoriesResponse, error) {
	if req.ConfigId == nil {
		return nil, consumerV1.ErrorBadRequest("config_id is required")
	}

	tenantID := s.getTenantIDFromContext(ctx)

	return s.repo.ListHistory(ctx, tenantID, *req.ConfigId, req.Paging)
}

// RollbackConfig 回滚配置到历史版本
func (s *TenantConfigService) RollbackConfig(ctx context.Context, req *consumerV1.RollbackTenantConfigRequest) (*emptypb.Empty, error) {
	if req.ConfigId == nil || req.HistoryId == nil {
		return nil, consumerV1.ErrorBadRequest("config_id and history_id are required")
	}

	tenantID := s.getTenantIDFromContext(ctx)
	operatorID := s.getOperatorIDFromContext(ctx)

	// 1. 查询当前配置
	currentCfg, err := s.repo.Get(ctx, *req.ConfigId)
	if err != nil {
		return nil, err
	}

	// 验证租户ID
	if currentCfg.TenantId == nil || *currentCfg.TenantId != tenantID {
		return nil, consumerV1.ErrorForbidden("access denied")
	}

	// 2. 查询历史记录（需要实现GetHistory方法）
	// 这里简化处理，实际应该从历史表查询
	// history, err := s.repo.GetHistory(ctx, *req.HistoryId)

	// 3. 回滚配置值
	// 这里需要根据历史记录的old_value进行回滚
	// 简化实现：假设从请求中获取回滚值
	if req.RollbackValue == nil {
		return nil, consumerV1.ErrorBadRequest("rollback_value is required")
	}

	updateData := &consumerV1.TenantConfig{
		ConfigValue: req.RollbackValue,
	}

	if err := s.repo.Update(ctx, *req.ConfigId, updateData); err != nil {
		return nil, err
	}

	// 4. 记录回滚历史
	history := &consumerV1.TenantConfigHistory{
		TenantId:      &tenantID,
		ConfigId:      req.ConfigId,
		ConfigKey:     currentCfg.ConfigKey,
		OldValue:      currentCfg.ConfigValue,
		NewValue:      req.RollbackValue,
		ChangeType:    consumerV1.TenantConfigHistory_ROLLBACK.Enum(),
		ChangeReason:  req.ChangeReason,
		ChangedBy:     &operatorID,
		ChangedByName: req.ChangedByName,
	}
	if err := s.repo.CreateHistory(ctx, history); err != nil {
		s.log.Warnf("failed to create config history: %s", err.Error())
	}

	// 5. 清除缓存
	if currentCfg.ConfigKey != nil {
		cacheKey := s.buildCacheKey(tenantID, *currentCfg.ConfigKey)
		if err := s.deleteFromCache(ctx, cacheKey); err != nil {
			s.log.Warnf("failed to delete config cache: %s", err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

// validateConfig 验证配置格式和有效性
func (s *TenantConfigService) validateConfig(cfg *consumerV1.TenantConfig) error {
	if cfg.ConfigKey == nil || *cfg.ConfigKey == "" {
		return consumerV1.ErrorBadRequest("config_key is required")
	}

	// 验证配置类型
	if cfg.ConfigType != nil {
		validTypes := map[string]bool{
			"string":    true,
			"int":       true,
			"bool":      true,
			"json":      true,
			"encrypted": true,
		}
		if !validTypes[*cfg.ConfigType] {
			return consumerV1.ErrorBadRequest("invalid config_type")
		}
	}

	// 根据类型验证值
	if cfg.ConfigValue != nil && cfg.ConfigType != nil {
		switch *cfg.ConfigType {
		case "int":
			if _, err := strconv.Atoi(*cfg.ConfigValue); err != nil {
				return consumerV1.ErrorBadRequest("config_value must be a valid integer")
			}
		case "bool":
			if *cfg.ConfigValue != "true" && *cfg.ConfigValue != "false" {
				return consumerV1.ErrorBadRequest("config_value must be true or false")
			}
		case "json":
			var js json.RawMessage
			if err := json.Unmarshal([]byte(*cfg.ConfigValue), &js); err != nil {
				return consumerV1.ErrorBadRequest("config_value must be valid JSON")
			}
		}
	}

	// 验证自定义规则
	if cfg.ValidationRule != nil && *cfg.ValidationRule != "" {
		// 这里可以实现更复杂的验证逻辑
		// 例如：正则表达式验证、范围验证等
	}

	return nil
}

// buildCacheKey 构建缓存键
func (s *TenantConfigService) buildCacheKey(tenantID uint32, configKey string) string {
	return fmt.Sprintf("%s%d:%s", configCachePrefix, tenantID, configKey)
}

// getFromCache 从缓存获取配置
func (s *TenantConfigService) getFromCache(ctx context.Context, key string) (*consumerV1.TenantConfig, error) {
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var cfg consumerV1.TenantConfig
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setToCache 写入缓存
func (s *TenantConfigService) setToCache(ctx context.Context, key string, cfg *consumerV1.TenantConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, data, configCacheTTL).Err()
}

// deleteFromCache 删除缓存
func (s *TenantConfigService) deleteFromCache(ctx context.Context, key string) error {
	return s.redis.Del(ctx, key).Err()
}

// getTenantIDFromContext 从上下文获取租户ID
func (s *TenantConfigService) getTenantIDFromContext(ctx context.Context) uint32 {
	// 从上下文中提取租户ID
	// 实际实现需要根据中间件设置的上下文键
	if tenantID, ok := ctx.Value("tenant_id").(uint32); ok {
		return tenantID
	}
	return 0
}

// getOperatorIDFromContext 从上下文获取操作人ID
func (s *TenantConfigService) getOperatorIDFromContext(ctx context.Context) uint32 {
	// 从上下文中提取操作人ID
	// 实际实现需要根据中间件设置的上下文键
	if operatorID, ok := ctx.Value("operator_id").(uint32); ok {
		return operatorID
	}
	return 0
}
