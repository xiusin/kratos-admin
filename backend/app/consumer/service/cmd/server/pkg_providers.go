package main

import (
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	entCrud "github.com/tx7do/go-crud/entgo"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	redisClient "github.com/tx7do/kratos-bootstrap/cache/redis"

	"go-wind-admin/app/consumer/service/internal/data"
	"go-wind-admin/app/consumer/service/internal/data/ent"
	"go-wind-admin/app/consumer/service/internal/service"
	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/jwt"
	"go-wind-admin/pkg/logistics"
	"go-wind-admin/pkg/oss"
	"go-wind-admin/pkg/payment"
	"go-wind-admin/pkg/sms"
)

// NewEventBus 创建事件总线
func NewEventBus(ctx *bootstrap.Context) eventbus.EventBus {
	return eventbus.NewEventBus(ctx.GetLogger())
}

// NewJWTHelper 已在 jwt 包中定义，这里不需要重复定义
// 直接使用 jwt.NewJWTHelper

// NewSMSClients 创建SMS客户端集合
func NewSMSClients(ctx *bootstrap.Context) (*service.SMSClients, error) {
	// TODO: 从配置文件读取配置
	aliyunCfg := &sms.Config{
		Provider:                 sms.ProviderAliyun,
		AccessKeyID:              "your-access-key-id",
		AccessKeySecret:          "your-access-key-secret",
		SignName:                 "your-sign-name",
		VerificationCodeTemplate: "SMS_123456789",
	}

	aliyunClient, err := sms.NewAliyunClient(aliyunCfg, ctx.GetLogger())
	if err != nil {
		return nil, err
	}

	tencentCfg := &sms.Config{
		Provider:                 sms.ProviderTencent,
		AccessKeyID:              "your-access-key-id",
		AccessKeySecret:          "your-access-key-secret",
		SignName:                 "your-sign-name",
		VerificationCodeTemplate: "123456",
	}

	tencentClient, err := sms.NewTencentClient(tencentCfg, ctx.GetLogger())
	if err != nil {
		return nil, err
	}

	return &service.SMSClients{
		Aliyun:  aliyunClient,
		Tencent: tencentClient,
	}, nil
}

// NewPaymentClient 创建支付客户端
func NewPaymentClient(ctx *bootstrap.Context) (payment.Client, error) {
	// TODO: 从配置文件读取配置
	// 这里默认使用微信支付作为示例
	cfg := &payment.Config{
		Provider:   payment.ProviderWechat,
		AppID:      "your-app-id",
		MchID:      "your-mch-id",
		APIKey:     "your-api-key",
		PrivateKey: "your-private-key",
		PublicKey:  "your-public-key",
		NotifyURL:  "https://your-domain.com/api/payment/callback",
	}

	client, err := payment.NewClient(cfg, ctx.GetLogger())
	if err != nil {
		return nil, err
	}

	return client, nil
}

// NewRedisClient 创建Redis客户端
func NewRedisClient(ctx *bootstrap.Context) (*redis.Client, func(), error) {
	cfg := ctx.GetConfig()
	if cfg == nil {
		return nil, func() {}, nil
	}

	l := ctx.NewLoggerHelper("redis/data/consumer-service")

	cli := redisClient.NewClient(cfg.Data, l)

	return cli, func() {
		if err := cli.Close(); err != nil {
			l.Error(err)
		}
	}, nil
}

// NewEntClient 创建Ent客户端
func NewEntClient(ctx *bootstrap.Context) (*entCrud.EntClient[*ent.Client], func(), error) {
	// 直接调用 data 包中的 NewEntClient
	return data.NewEntClient(ctx)
}

// NewOSSClient 创建OSS客户端
func NewOSSClient(ctx *bootstrap.Context) (oss.Client, error) {
	// TODO: 从配置文件读取配置
	// 这里默认使用阿里云OSS作为示例
	cfg := &oss.Config{
		Provider:        oss.ProviderAliyun,
		Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
		AccessKeyID:     "your-access-key-id",
		AccessKeySecret: "your-access-key-secret",
		BucketName:      "your-bucket-name",
		Region:          "cn-hangzhou",
	}

	client, err := oss.NewClient(cfg, ctx.GetLogger())
	if err != nil {
		return nil, err
	}

	return client, nil
}

// NewLogisticsClient 创建物流客户端
func NewLogisticsClient(ctx *bootstrap.Context) (logistics.Client, error) {
	// TODO: 从配置文件读取配置
	// 这里默认使用快递鸟作为示例
	cfg := &logistics.Config{
		AppID:  "your-kdniao-app-id",
		AppKey: "your-kdniao-app-key",
	}

	client, err := logistics.NewClient(cfg, ctx.GetLogger())
	if err != nil {
		return nil, err
	}

	return client, nil
}

// PkgProviderSet pkg层的依赖注入集合
var PkgProviderSet = wire.NewSet(
	jwt.NewJWTHelper,
	NewEventBus,
	NewSMSClients,
	NewPaymentClient,
	NewRedisClient,
	NewEntClient,
	NewOSSClient,
	NewLogisticsClient,
)
