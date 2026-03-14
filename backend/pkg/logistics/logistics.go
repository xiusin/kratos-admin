package logistics

import (
	"context"
	"time"
)

// Client 物流客户端接口
type Client interface {
	// Query 查询物流信息
	Query(ctx context.Context, trackingNo string, courierCode string) (*TrackingInfo, error)

	// RecognizeCourier 识别快递公司
	RecognizeCourier(ctx context.Context, trackingNo string) (string, error)

	// Subscribe 订阅物流状态
	Subscribe(ctx context.Context, trackingNo string, courierCode string) error

	// GetSupportedCouriers 获取支持的快递公司列表
	GetSupportedCouriers() []CourierInfo
}

// Config 物流配置
type Config struct {
	// AppID 应用ID
	AppID string `json:"app_id"`

	// AppKey 应用密钥
	AppKey string `json:"app_key"`

	// Timeout 超时时间
	Timeout time.Duration `json:"timeout"`
}

// TrackingInfo 物流跟踪信息
type TrackingInfo struct {
	// TrackingNo 运单号
	TrackingNo string

	// CourierCode 快递公司代码
	CourierCode string

	// CourierName 快递公司名称
	CourierName string

	// Status 物流状态
	Status TrackingStatus

	// LastUpdateTime 最后更新时间
	LastUpdateTime time.Time

	// Traces 物流轨迹
	Traces []*TrackingTrace
}

// TrackingTrace 物流轨迹
type TrackingTrace struct {
	// Time 时间
	Time time.Time

	// Description 描述
	Description string

	// Location 位置
	Location string
}

// TrackingStatus 物流状态
type TrackingStatus string

const (
	StatusPending    TrackingStatus = "pending"    // 待揽收
	StatusPickedUp   TrackingStatus = "picked_up"  // 已揽收
	StatusInTransit  TrackingStatus = "in_transit" // 运输中
	StatusDelivering TrackingStatus = "delivering" // 派送中
	StatusDelivered  TrackingStatus = "delivered"  // 已签收
	StatusException  TrackingStatus = "exception"  // 异常
	StatusReturning  TrackingStatus = "returning"  // 退回中
)

// CourierInfo 快递公司信息
type CourierInfo struct {
	// Code 快递公司代码
	Code string

	// Name 快递公司名称
	Name string
}
