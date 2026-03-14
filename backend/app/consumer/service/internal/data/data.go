package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"

	_ "github.com/go-sql-driver/mysql"
)

// Data 数据层结构
type Data struct {
	db    *sql.Driver
	rdb   *redis.ClusterClient
	log   *log.Helper
}

// NewData 创建数据层实例
func NewData(ctx *bootstrap.Context) (*Data, func(), error) {
	cfg := ctx.GetConfig()
	logger := log.NewHelper(log.With(ctx.GetLogger(), "module", "data"))

	// 初始化数据库连接
	db, err := initDatabase(cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	// 初始化 Redis 集群连接
	rdb, err := initRedis(cfg, logger)
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
func initDatabase(cfg *bootstrap.Config, logger *log.Helper) (*sql.Driver, error) {
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
	db.SetMaxIdleConns(int(dbCfg.MaxIdleConnections))
	db.SetMaxOpenConns(int(dbCfg.MaxOpenConnections))
	db.SetConnMaxLifetime(dbCfg.ConnectionMaxLifetime.AsDuration())

	// 测试连接
	if err := db.Ping(); err != nil {
		logger.Errorf("failed to ping database: %v", err)
		return nil, err
	}

	logger.Info("database connected successfully")

	// TODO: 运行数据库迁移
	// if dbCfg.Migrate {
	//     if err := runMigrations(drv); err != nil {
	//         logger.Errorf("failed to run migrations: %v", err)
	//         return nil, err
	//     }
	// }

	return drv, nil
}

// initRedis 初始化 Redis 集群连接
func initRedis(cfg *bootstrap.Config, logger *log.Helper) (*redis.ClusterClient, error) {
	if cfg == nil || cfg.Data == nil || cfg.Data.Redis == nil {
		return nil, nil
	}

	redisCfg := cfg.Data.Redis

	// 创建 Redis 集群客户端
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        redisCfg.Addrs,
		Password:     redisCfg.Password,
		DialTimeout:  redisCfg.DialTimeout.AsDuration(),
		ReadTimeout:  redisCfg.ReadTimeout.AsDuration(),
		WriteTimeout: redisCfg.WriteTimeout.AsDuration(),
		PoolSize:     int(redisCfg.PoolSize),
		MinIdleConns: int(redisCfg.MinIdleConns),
	})

	// 测试连接
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Errorf("failed to ping redis: %v", err)
		return nil, err
	}

	logger.Info("redis cluster connected successfully")

	return rdb, nil
}

// DB 获取数据库驱动
func (d *Data) DB() *sql.Driver {
	return d.db
}

// Redis 获取 Redis 客户端
func (d *Data) Redis() *redis.ClusterClient {
	return d.rdb
}
