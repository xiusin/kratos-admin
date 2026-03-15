package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/logistics"
)

// LogisticsService 物流服务
type LogisticsService struct {
	consumerV1.UnimplementedLogisticsServiceServer

	logisticsTrackingRepo data.LogisticsTrackingRepo
	logisticsClient       logistics.Client
	redisClient           *redis.Client
	eventBus              eventbus.EventBus
	log                   *log.Helper
}

// NewLogisticsService 创建物流服务实例
func NewLogisticsService(
	ctx *bootstrap.Context,
	logisticsTrackingRepo data.LogisticsTrackingRepo,
	logisticsClient logistics.Client,
	redisClient *redis.Client,
	eventBus eventbus.EventBus,
) *LogisticsService {
	return &LogisticsService{
		logisticsTrackingRepo: logisticsTrackingRepo,
		logisticsClient:       logisticsClient,
		redisClient:           redisClient,
		eventBus:              eventBus,
		log:                   ctx.NewLoggerHelper("consumer/service/logistics-service"),
	}
}

// QueryLogistics 查询物流信息
func (s *LogisticsService) QueryLogistics(ctx context.Context, req *consumerV1.QueryLogisticsRequest) (*consumerV1.LogisticsInfo, error) {
	s.log.Infof("QueryLogistics: tracking_no=%s, courier_company=%s", req.GetTrackingNo(), req.GetCourierCompany())

	trackingNo := req.GetTrackingNo()
	courierCompany := req.GetCourierCompany()

	// 1. 尝试从缓存获取
	cacheKey := s.getCacheKey(trackingNo)
	cachedData, err := s.redisClient.Get(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		s.log.Infof("QueryLogistics: cache hit for tracking_no=%s", trackingNo)

		var logisticsInfo consumerV1.LogisticsInfo
		if err := json.Unmarshal([]byte(cachedData), &logisticsInfo); err == nil {
			return &logisticsInfo, nil
		}
	}

	// 2. 调用第三方物流API查询
	trackingInfo, err := s.logisticsClient.Query(ctx, trackingNo, courierCompany)
	if err != nil {
		s.log.Errorf("query logistics from third-party failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "query logistics failed")
	}

	// 3. 转换为 Protobuf 格式
	logisticsInfo := s.convertToLogisticsInfo(trackingInfo)

	// 4. 缓存物流信息（30分钟）
	if err := s.cacheLogisticsInfo(ctx, trackingNo, logisticsInfo); err != nil {
		s.log.Warnf("cache logistics info failed: %v", err)
	}

	// 5. 保存或更新物流跟踪记录
	if err := s.saveOrUpdateTracking(ctx, trackingInfo); err != nil {
		s.log.Warnf("save or update tracking failed: %v", err)
	}

	s.log.Infof("QueryLogistics success: tracking_no=%s, status=%s", trackingNo, logisticsInfo.Status)
	return logisticsInfo, nil
}

// SubscribeLogistics 订阅物流状态
func (s *LogisticsService) SubscribeLogistics(ctx context.Context, req *consumerV1.SubscribeLogisticsRequest) (*emptypb.Empty, error) {
	s.log.Infof("SubscribeLogistics: tracking_no=%s, courier_company=%s", req.GetTrackingNo(), req.GetCourierCompany())

	trackingNo := req.GetTrackingNo()
	courierCompany := req.GetCourierCompany()

	// 如果没有指定快递公司，自动识别
	if courierCompany == "" {
		code, err := s.logisticsClient.RecognizeCourier(ctx, trackingNo)
		if err != nil {
			s.log.Errorf("recognize courier failed: %v", err)
			return nil, errors.BadRequest("INVALID_ARGUMENT", "cannot recognize courier company")
		}
		courierCompany = code
	}

	// 调用第三方物流API订阅
	if err := s.logisticsClient.Subscribe(ctx, trackingNo, courierCompany); err != nil {
		s.log.Errorf("subscribe logistics from third-party failed: %v", err)
		return nil, errors.InternalServer("INTERNAL_ERROR", "subscribe logistics failed")
	}

	// 保存订阅记录到数据库
	// TODO: 可以添加一个订阅表来记录订阅信息

	s.log.Infof("SubscribeLogistics success: tracking_no=%s", trackingNo)
	return &emptypb.Empty{}, nil
}

// ListLogisticsHistory 查询物流历史
func (s *LogisticsService) ListLogisticsHistory(ctx context.Context, req *consumerV1.ListLogisticsHistoryRequest) (*consumerV1.ListLogisticsHistoryResponse, error) {
	s.log.Infof("ListLogisticsHistory: consumer_id=%v, status=%v", req.ConsumerId, req.Status)

	resp, err := s.logisticsTrackingRepo.List(ctx, req)
	if err != nil {
		s.log.Errorf("list logistics history failed: %v", err)
		return nil, err
	}

	return resp, nil
}

// convertToLogisticsInfo 转换为 LogisticsInfo
func (s *LogisticsService) convertToLogisticsInfo(trackingInfo *logistics.TrackingInfo) *consumerV1.LogisticsInfo {
	logisticsInfo := &consumerV1.LogisticsInfo{
		TrackingNo:      trackingInfo.TrackingNo,
		CourierCompany:  trackingInfo.CourierName,
		Status:          s.convertStatus(trackingInfo.Status),
		LastUpdatedAt:   timestamppb.New(trackingInfo.LastUpdateTime),
		TrackingDetails: make([]*consumerV1.TrackingDetail, 0, len(trackingInfo.Traces)),
	}

	// 转换物流轨迹
	for _, trace := range trackingInfo.Traces {
		detail := &consumerV1.TrackingDetail{
			Time:        timestamppb.New(trace.Time),
			Location:    &trace.Location,
			Description: &trace.Description,
			Status:      nil, // 可以根据需要设置
		}
		logisticsInfo.TrackingDetails = append(logisticsInfo.TrackingDetails, detail)
	}

	return logisticsInfo
}

// convertStatus 转换物流状态
func (s *LogisticsService) convertStatus(status logistics.TrackingStatus) consumerV1.LogisticsTracking_Status {
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

// getCacheKey 获取缓存键
func (s *LogisticsService) getCacheKey(trackingNo string) string {
	return fmt.Sprintf("logistics:tracking:%s", trackingNo)
}

// cacheLogisticsInfo 缓存物流信息（30分钟）
func (s *LogisticsService) cacheLogisticsInfo(ctx context.Context, trackingNo string, info *consumerV1.LogisticsInfo) error {
	cacheKey := s.getCacheKey(trackingNo)

	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	// 缓存30分钟
	return s.redisClient.Set(ctx, cacheKey, string(data), 30*time.Minute).Err()
}

// saveOrUpdateTracking 保存或更新物流跟踪记录
func (s *LogisticsService) saveOrUpdateTracking(ctx context.Context, trackingInfo *logistics.TrackingInfo) error {
	// 查询是否已存在
	existing, err := s.logisticsTrackingRepo.GetByTrackingNo(ctx, trackingInfo.TrackingNo)

	// 转换 tracking_info 为 structpb.Struct
	trackingInfoStructs := make([]*structpb.Struct, 0, len(trackingInfo.Traces))
	for _, trace := range trackingInfo.Traces {
		traceMap := map[string]interface{}{
			"time":        trace.Time.Format(time.RFC3339),
			"location":    trace.Location,
			"description": trace.Description,
		}
		traceStruct, err := structpb.NewStruct(traceMap)
		if err != nil {
			s.log.Warnf("convert trace to struct failed: %v", err)
			continue
		}
		trackingInfoStructs = append(trackingInfoStructs, traceStruct)
	}

	status := s.convertStatus(trackingInfo.Status)

	if err != nil || existing == nil {
		// 不存在，创建新记录
		tracking := &consumerV1.LogisticsTracking{
			TrackingNo:     &trackingInfo.TrackingNo,
			CourierCompany: &trackingInfo.CourierName,
			Status:         &status,
			TrackingInfo:   trackingInfoStructs,
			LastUpdatedAt:  timestamppb.New(trackingInfo.LastUpdateTime),
		}

		_, err := s.logisticsTrackingRepo.Create(ctx, tracking)
		if err != nil {
			return err
		}

		s.log.Infof("created logistics tracking: tracking_no=%s", trackingInfo.TrackingNo)
	} else {
		// 已存在，检查状态是否变更
		oldStatus := existing.GetStatus()
		newStatus := status

		// 更新记录
		updateData := &consumerV1.LogisticsTracking{
			Status:        &newStatus,
			TrackingInfo:  trackingInfoStructs,
			LastUpdatedAt: timestamppb.New(trackingInfo.LastUpdateTime),
		}

		if err := s.logisticsTrackingRepo.Update(ctx, existing.GetId(), updateData); err != nil {
			return err
		}

		s.log.Infof("updated logistics tracking: tracking_no=%s", trackingInfo.TrackingNo)

		// 如果状态变更，发布事件
		if oldStatus != newStatus {
			s.publishLogisticsStatusChangedEvent(ctx, trackingInfo, oldStatus, newStatus)
		}
	}

	return nil
}

// publishLogisticsStatusChangedEvent 发布物流状态变更事件
func (s *LogisticsService) publishLogisticsStatusChangedEvent(
	ctx context.Context,
	trackingInfo *logistics.TrackingInfo,
	oldStatus consumerV1.LogisticsTracking_Status,
	newStatus consumerV1.LogisticsTracking_Status,
) {
	event := eventbus.NewEvent("logistics.status_changed", map[string]interface{}{
		"tracking_no":     trackingInfo.TrackingNo,
		"courier_company": trackingInfo.CourierName,
		"old_status":      oldStatus.String(),
		"new_status":      newStatus.String(),
		"updated_at":      trackingInfo.LastUpdateTime.Format(time.RFC3339),
	}).WithSource("logistics-service")

	if err := s.eventBus.PublishAsync(ctx, event); err != nil {
		s.log.Errorf("publish logistics status changed event failed: %v", err)
	} else {
		s.log.Infof("published logistics status changed event: tracking_no=%s, %s -> %s",
			trackingInfo.TrackingNo, oldStatus.String(), newStatus.String())
	}
}
