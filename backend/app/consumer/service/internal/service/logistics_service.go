package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/go-utils/trans"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/logistics"
	"go-wind-admin/pkg/middleware"
)

const (
	// 物流信息缓存时间（30分钟）
	logisticsCacheTTL    = 30 * time.Minute
	logisticsCachePrefix = "logistics:tracking:"
)

// LogisticsService 物流服务
type LogisticsService struct {
	consumerV1.UnimplementedLogisticsServiceServer

	log                   *log.Helper
	logisticsClient       logistics.Client
	logisticsTrackingRepo data.LogisticsTrackingRepo
	redis                 *redis.Client
	eventBus              eventbus.EventBus
}

// NewLogisticsService 创建物流服务实例
func NewLogisticsService(
	ctx *bootstrap.Context,
	logisticsClient logistics.Client,
	logisticsTrackingRepo data.LogisticsTrackingRepo,
	redis *redis.Client,
	eventBus eventbus.EventBus,
) *LogisticsService {
	return &LogisticsService{
		log:                   ctx.NewLoggerHelper("consumer/service/logistics-service"),
		logisticsClient:       logisticsClient,
		logisticsTrackingRepo: logisticsTrackingRepo,
		redis:                 redis,
		eventBus:              eventBus,
	}
}

// QueryLogistics 查询物流信息
func (s *LogisticsService) QueryLogistics(ctx context.Context, req *consumerV1.QueryLogisticsRequest) (*consumerV1.LogisticsInfo, error) {
	// 1. 验证输入
	if req == nil || req.TrackingNo == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	trackingNo := req.TrackingNo
	courierCompany := req.GetCourierCompany()

	// 2. 获取租户ID
	tenantID := middleware.GetTenantID(ctx)

	// 3. 尝试从缓存获取物流信息
	cachedInfo, err := s.getLogisticsFromCache(ctx, trackingNo)
	if err == nil && cachedInfo != nil {
		s.log.Infof("logistics info from cache: tracking_no=%s", trackingNo)
		return cachedInfo, nil
	}

	// 4. 如果没有指定快递公司，自动识别
	courierCode := courierCompany
	if courierCode == "" {
		code, err := s.logisticsClient.RecognizeCourier(ctx, trackingNo)
		if err != nil {
			s.log.Errorf("recognize courier failed: %v", err)
			return nil, consumerV1.ErrorInternalServerError("failed to recognize courier company")
		}
		courierCode = code
	}

	// 5. 调用快递鸟API查询物流信息
	trackingInfo, err := s.logisticsClient.Query(ctx, trackingNo, courierCode)
	if err != nil {
		s.log.Errorf("query logistics failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to query logistics info")
	}

	// 6. 解析和格式化物流数据
	logisticsInfo := s.formatLogisticsInfo(trackingInfo)

	// 7. 缓存物流信息（30分钟）
	if err := s.cacheLogisticsInfo(ctx, trackingNo, logisticsInfo); err != nil {
		s.log.Warnf("cache logistics info failed: %v", err)
		// 缓存失败不影响主流程
	}

	// 8. 查询数据库中的物流跟踪记录
	existingTracking, err := s.logisticsTrackingRepo.GetByTrackingNo(ctx, tenantID, trackingNo)
	if err != nil {
		s.log.Warnf("get existing tracking failed: %v", err)
	}

	// 9. 检测物流状态变更
	if existingTracking != nil {
		oldStatus := existingTracking.GetStatus()
		newStatus := logisticsInfo.Status

		// 如果状态发生变更，发布事件
		if oldStatus != newStatus {
			s.publishLogisticsStatusChangedEvent(ctx, trackingNo, courierCode, oldStatus, newStatus)

			// 更新数据库记录
			s.updateLogisticsTracking(ctx, existingTracking.GetId(), logisticsInfo)
		}
	} else {
		// 10. 如果数据库中不存在，创建新记录
		s.createLogisticsTracking(ctx, tenantID, logisticsInfo)
	}

	s.log.Infof("logistics info queried: tracking_no=%s, courier=%s, status=%s",
		trackingNo, courierCode, logisticsInfo.Status.String())

	return logisticsInfo, nil
}

// SubscribeLogistics 订阅物流状态
func (s *LogisticsService) SubscribeLogistics(ctx context.Context, req *consumerV1.SubscribeLogisticsRequest) (*emptypb.Empty, error) {
	// 1. 验证输入
	if req == nil || req.TrackingNo == "" {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	trackingNo := req.TrackingNo
	courierCompany := req.GetCourierCompany()

	// 2. 如果没有指定快递公司，自动识别
	courierCode := courierCompany
	if courierCode == "" {
		code, err := s.logisticsClient.RecognizeCourier(ctx, trackingNo)
		if err != nil {
			s.log.Errorf("recognize courier failed: %v", err)
			return nil, consumerV1.ErrorInternalServerError("failed to recognize courier company")
		}
		courierCode = code
	}

	// 3. 调用快递鸟API订阅物流状态
	if err := s.logisticsClient.Subscribe(ctx, trackingNo, courierCode); err != nil {
		s.log.Errorf("subscribe logistics failed: %v", err)
		return nil, consumerV1.ErrorInternalServerError("failed to subscribe logistics")
	}

	// 4. 获取租户ID
	tenantID := middleware.GetTenantID(ctx)

	// 5. 查询当前物流信息
	trackingInfo, err := s.logisticsClient.Query(ctx, trackingNo, courierCode)
	if err != nil {
		s.log.Warnf("query logistics after subscribe failed: %v", err)
		// 订阅成功但查询失败，不影响主流程
	} else {
		// 6. 创建或更新物流跟踪记录
		logisticsInfo := s.formatLogisticsInfo(trackingInfo)
		existingTracking, _ := s.logisticsTrackingRepo.GetByTrackingNo(ctx, tenantID, trackingNo)
		if existingTracking != nil {
			s.updateLogisticsTracking(ctx, existingTracking.GetId(), logisticsInfo)
		} else {
			s.createLogisticsTracking(ctx, tenantID, logisticsInfo)
		}
	}

	s.log.Infof("logistics subscribed: tracking_no=%s, courier=%s", trackingNo, courierCode)

	return &emptypb.Empty{}, nil
}

// ListLogisticsHistory 查询物流历史
func (s *LogisticsService) ListLogisticsHistory(ctx context.Context, req *consumerV1.ListLogisticsHistoryRequest) (*consumerV1.ListLogisticsHistoryResponse, error) {
	if req == nil {
		return nil, consumerV1.ErrorBadRequest("invalid parameter")
	}

	// 构建分页请求
	pagingReq := &paginationV1.PagingRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	// 注意：实际项目中应该在Repository层添加更多过滤条件
	// 如按用户ID、状态等过滤
	resp, err := s.logisticsTrackingRepo.List(ctx, pagingReq)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ========== 私有辅助方法 ==========

// formatLogisticsInfo 格式化物流信息
func (s *LogisticsService) formatLogisticsInfo(trackingInfo *logistics.TrackingInfo) *consumerV1.LogisticsInfo {
	if trackingInfo == nil {
		return nil
	}

	// 转换物流状态
	status := s.convertLogisticsStatus(trackingInfo.Status)

	// 转换物流轨迹
	trackingDetails := make([]*consumerV1.TrackingDetail, 0, len(trackingInfo.Traces))
	for _, trace := range trackingInfo.Traces {
		trackingDetails = append(trackingDetails, &consumerV1.TrackingDetail{
			Time:        timestamppb.New(trace.Time),
			Location:    trans.Ptr(trace.Location),
			Description: trans.Ptr(trace.Description),
			Status:      trans.Ptr(string(trackingInfo.Status)),
		})
	}

	return &consumerV1.LogisticsInfo{
		TrackingNo:      trackingInfo.TrackingNo,
		CourierCompany:  trackingInfo.CourierName,
		Status:          status,
		TrackingDetails: trackingDetails,
		LastUpdatedAt:   timestamppb.New(trackingInfo.LastUpdateTime),
	}
}

// convertLogisticsStatus 转换物流状态
func (s *LogisticsService) convertLogisticsStatus(status logistics.TrackingStatus) consumerV1.LogisticsTracking_Status {
	switch status {
	case logistics.StatusPending:
		return consumerV1.LogisticsTracking_PENDING
	case logistics.StatusPickedUp, logistics.StatusInTransit:
		return consumerV1.LogisticsTracking_IN_TRANSIT
	case logistics.StatusDelivering:
		return consumerV1.LogisticsTracking_DELIVERING
	case logistics.StatusDelivered:
		return consumerV1.LogisticsTracking_DELIVERED
	default:
		return consumerV1.LogisticsTracking_PENDING
	}
}

// getLogisticsFromCache 从缓存获取物流信息
func (s *LogisticsService) getLogisticsFromCache(ctx context.Context, trackingNo string) (*consumerV1.LogisticsInfo, error) {
	key := logisticsCachePrefix + trackingNo
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var logisticsInfo consumerV1.LogisticsInfo
	if err := json.Unmarshal([]byte(data), &logisticsInfo); err != nil {
		return nil, err
	}

	return &logisticsInfo, nil
}

// cacheLogisticsInfo 缓存物流信息
func (s *LogisticsService) cacheLogisticsInfo(ctx context.Context, trackingNo string, info *consumerV1.LogisticsInfo) error {
	key := logisticsCachePrefix + trackingNo
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	return s.redis.Set(ctx, key, data, logisticsCacheTTL).Err()
}

// createLogisticsTracking 创建物流跟踪记录
func (s *LogisticsService) createLogisticsTracking(ctx context.Context, tenantID uint32, info *consumerV1.LogisticsInfo) {
	// 转换物流轨迹为JSON格式
	trackingInfo := make([]*structpb.Struct, 0, len(info.TrackingDetails))
	for _, detail := range info.TrackingDetails {
		detailMap := map[string]interface{}{
			"time":        detail.Time.AsTime().Format(time.RFC3339),
			"location":    detail.GetLocation(),
			"description": detail.GetDescription(),
			"status":      detail.GetStatus(),
		}
		st, _ := structpb.NewStruct(detailMap)
		trackingInfo = append(trackingInfo, st)
	}

	tracking := &consumerV1.LogisticsTracking{
		TenantId:       trans.Ptr(tenantID),
		TrackingNo:     trans.Ptr(info.TrackingNo),
		CourierCompany: trans.Ptr(info.CourierCompany),
		Status:         &info.Status,
		TrackingInfo:   trackingInfo,
		LastUpdatedAt:  info.LastUpdatedAt,
	}

	_, err := s.logisticsTrackingRepo.Create(ctx, tracking)
	if err != nil {
		s.log.Errorf("create logistics tracking failed: %v", err)
	}
}

// updateLogisticsTracking 更新物流跟踪记录
func (s *LogisticsService) updateLogisticsTracking(ctx context.Context, id uint64, info *consumerV1.LogisticsInfo) {
	// 转换物流轨迹为JSON格式
	trackingInfo := make([]*structpb.Struct, 0, len(info.TrackingDetails))
	for _, detail := range info.TrackingDetails {
		detailMap := map[string]interface{}{
			"time":        detail.Time.AsTime().Format(time.RFC3339),
			"location":    detail.GetLocation(),
			"description": detail.GetDescription(),
			"status":      detail.GetStatus(),
		}
		st, _ := structpb.NewStruct(detailMap)
		trackingInfo = append(trackingInfo, st)
	}

	tracking := &consumerV1.LogisticsTracking{
		Status:        &info.Status,
		TrackingInfo:  trackingInfo,
		LastUpdatedAt: info.LastUpdatedAt,
	}

	err := s.logisticsTrackingRepo.Update(ctx, id, tracking)
	if err != nil {
		s.log.Errorf("update logistics tracking failed: %v", err)
	}
}

// publishLogisticsStatusChangedEvent 发布物流状态变更事件
func (s *LogisticsService) publishLogisticsStatusChangedEvent(
	ctx context.Context,
	trackingNo string,
	courierCompany string,
	oldStatus consumerV1.LogisticsTracking_Status,
	newStatus consumerV1.LogisticsTracking_Status,
) {
	event := eventbus.LogisticsStatusChangedEvent{
		TrackingNo:     trackingNo,
		CourierCompany: courierCompany,
		OldStatus:      oldStatus.String(),
		NewStatus:      newStatus.String(),
		ChangedAt:      time.Now(),
	}

	if err := s.eventBus.Publish(ctx, event); err != nil {
		s.log.Errorf("publish logistics status changed event failed: %v", err)
	} else {
		s.log.Infof("logistics status changed event published: tracking_no=%s, %s -> %s",
			trackingNo, oldStatus.String(), newStatus.String())
	}
}
