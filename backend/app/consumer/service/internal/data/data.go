package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/go-utils/password"
	"github.com/tx7do/kratos-bootstrap/bootstrap"
	redisClient "github.com/tx7do/kratos-bootstrap/cache/redis"

	"go-wind-admin/pkg/auth"
	"go-wind-admin/pkg/eventbus"
	"go-wind-admin/pkg/logistics"
	"go-wind-admin/pkg/oss"
	"go-wind-admin/pkg/payment"
	"go-wind-admin/pkg/sms"

	"entgo.io/ent/dialect/sql"

	_ "github.com/go-sql-driver/mysql"
)

// Data 数据层结构
type Data struct {
	db  *sql.Driver
	rdb redis.UniversalClient
	log *log.Helper
}

// NewData 创建数据层实例
func NewData(ctx *bootstrap.Context) (*Data, func(), error) {
	logger := log.NewHelper(log.With(ctx.GetLogger(), "module", "data"))

	// 初始化数据库连接
	db, err := initDatabase(ctx)
	if err != nil {
		return nil, nil, err
	}

	// 初始化 Redis 连接（使用 bootstrap 提供的方法）
	rdb, err := initRedis(ctx)
	if err != nil {
		return nil, nil, err
	}

	d := &Data{
		db:  db,
		rdb: rdb,
		log: logger,
	}

	cleanup := func() {
		logger.Info("closing data resources")
		if db != nil {
			if err := db.Close(); err != nil {
				logger.Errorf("failed to close database: %v", err)
			}
		}
		if rdb != nil {
			if err := rdb.Close(); err != nil {
				logger.Errorf("failed to close redis: %v", err)
			}
		}
	}

	return d, cleanup, nil
}

// initDatabase 初始化数据库连接
func initDatabase(ctx *bootstrap.Context) (*sql.Driver, error) {
	cfg := ctx.GetConfig()
	logger := ctx.NewLoggerHelper("database/consumer-service")
	
	if cfg == nil || cfg.Data == nil || cfg.Data.Database == nil {
		return nil, nil
	}

	dbCfg := cfg.Data.Database

	// 创建数据库连接
	drv, err := sql.Open(
		dbCfg.Driver,
		dbCfg.Source,
	)
	if err != nil {
		logger.Errorf("failed to open database: %v", err)
		return nil, err
	}

	// 配置连接池
	db := drv.DB()
	if dbCfg.MaxIdleConnections != nil {
		db.SetMaxIdleConns(int(*dbCfg.MaxIdleConnections))
	}
	if dbCfg.MaxOpenConnections != nil {
		db.SetMaxOpenConns(int(*dbCfg.MaxOpenConnections))
	}
	if dbCfg.ConnectionMaxLifetime != nil {
		db.SetConnMaxLifetime(dbCfg.ConnectionMaxLifetime.AsDuration())
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		logger.Errorf("failed to ping database: %v", err)
		return nil, err
	}

	logger.Info("database connected successfully")

	return drv, nil
}

// initRedis 初始化 Redis 连接
func initRedis(ctx *bootstrap.Context) (redis.UniversalClient, error) {
	cfg := ctx.GetConfig()
	logger := ctx.NewLoggerHelper("redis/consumer-service")
	
	if cfg == nil || cfg.Data == nil {
		return nil, nil
	}

	// 使用 bootstrap 提供的 Redis 客户端创建方法
	cli := redisClient.NewClient(cfg.Data, logger)
	if cli == nil {
		return nil, nil
	}

	// 测试连接
	pingCtx := context.Background()
	if err := cli.Ping(pingCtx).Err(); err != nil {
		logger.Errorf("failed to ping redis: %v", err)
		return nil, err
	}

	logger.Info("redis connected successfully")

	return cli, nil
}

// DB 获取数据库驱动
func (d *Data) DB() *sql.Driver {
	return d.db
}

// Redis 获取 Redis 客户端
func (d *Data) Redis() redis.UniversalClient {
	return d.rdb
}

// NewPasswordCrypto 创建密码加密工具
func NewPasswordCrypto() password.Crypto {
	crypto, err := password.CreateCrypto("bcrypt")
	if err != nil {
		panic(err)
	}
	return crypto
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(ctx *bootstrap.Context) *auth.JWTManager {
	// 从配置读取JWT密钥，如果没有配置则使用默认值
	secret := auth.DefaultJWTSecret
	// TODO: 从配置文件读取 JWT secret
	// cfg := ctx.GetConfig()
	// if cfg != nil && cfg.Server != nil && cfg.Server.Rest != nil {
	//     secret = cfg.Server.Rest.JWTSecret
	// }
	return auth.NewJWTManager(secret)
}

// NewSMSManager 创建短信管理器
func NewSMSManager(ctx *bootstrap.Context) *sms.Manager {
	logger := ctx.GetLogger()
	// 从配置读取短信服务配置
	// 实际项目中应该从配置文件读取
	// 这里使用默认配置
	primaryConfig := &sms.Config{
		Provider:        sms.ProviderAliyun,
		AccessKeyID:     "your-access-key-id",
		AccessKeySecret: "your-access-key-secret",
		SignName:        "your-sign-name",
	}

	primaryClient, err := sms.NewAliyunClient(primaryConfig, logger)
	if err != nil {
		log.NewHelper(logger).Warnf("failed to create primary sms client: %v", err)
		// 返回nil，服务层需要处理nil的情况
		return nil
	}

	// 可选：创建备用通道
	// secondaryClient, _ := sms.NewTencentClient(secondaryConfig, logger)

	return sms.NewManager(primaryClient, nil, logger)
}

// NewWechatClient 创建微信支付客户端
func NewWechatClient(ctx *bootstrap.Context) payment.Client {
	logger := ctx.GetLogger()
	// 从配置读取微信支付配置
	// 实际项目中应该从配置文件读取
	wechatConfig := &payment.Config{
		Provider:  payment.ProviderWechat,
		AppID:     "your-app-id",
		MchID:     "your-mch-id",
		APIKey:    "your-api-key",
		NotifyURL: "https://your-domain.com/api/payment/wechat/notify",
	}

	client, err := payment.NewWechatClient(wechatConfig, logger)
	if err != nil {
		log.NewHelper(logger).Warnf("failed to create wechat client: %v", err)
		return nil
	}

	return client
}

// NewOSSClient 创建OSS客户端
func NewOSSClient(ctx *bootstrap.Context) oss.Client {
	logger := ctx.GetLogger()
	// 从配置读取OSS配置
	// 实际项目中应该从配置文件读取
	ossConfig := &oss.Config{
		Provider:        oss.ProviderAliyun,
		Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
		AccessKeyID:     "your-access-key-id",
		AccessKeySecret: "your-access-key-secret",
		BucketName:      "your-bucket-name",
	}

	client, err := oss.NewAliyunOSSClient(ossConfig, logger)
	if err != nil {
		log.NewHelper(logger).Warnf("failed to create oss client: %v", err)
		return nil
	}

	return client
}

// NewEventBus 创建事件总线
func NewEventBus(logger log.Logger) eventbus.EventBus {
	return eventbus.NewEventBus(logger)
}

// NewLogisticsClient 创建物流客户端
func NewLogisticsClient(ctx *bootstrap.Context) logistics.Client {
	logger := ctx.GetLogger()
	// 从配置读取物流配置
	// 实际项目中应该从配置文件读取
	logisticsConfig := &logistics.Config{
		AppID:   "your-kdniao-app-id",
		AppKey:  "your-kdniao-app-key",
		Timeout: 30 * time.Second,
	}

	client, err := logistics.NewClient(logisticsConfig, logger)
	if err != nil {
		log.NewHelper(logger).Warnf("failed to create logistics client: %v", err)
		return nil
	}

	return client
}
