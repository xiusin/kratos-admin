package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"github.com/tx7do/kratos-bootstrap/bootstrap"

	"entgo.io/ent/dialect/sql"

	_ "github.com/go-sql-driver/mysql"
)

// Data 数据层结构
type Data struct {
	db  *sql.Driver
	rdb *redis.ClusterClient
	log *log.Helper
}

// NewData 创建数据层实例
func NewData(ctx *bootstrap.Context) (*Data, func(), error) {
	logger := log.NewHelper(log.With(ctx.GetLogger(), "module", "data/consumer-service"))

	d := &Data{
		log: logger,
	}

	cleanup := func() {
		logger.Info("closing data resources")
		if d.db != nil {
			if err := d.db.Close(); err != nil {
				logger.Errorf("failed to close database: %v", err)
			}
		}
		if d.rdb != nil {
			if err := d.rdb.Close(); err != nil {
				logger.Errorf("failed to close redis: %v", err)
			}
		}
	}

	return d, cleanup, nil
}

// DB 获取数据库驱动
func (d *Data) DB() *sql.Driver {
	return d.db
}

// Redis 获取 Redis 客户端
func (d *Data) Redis() *redis.ClusterClient {
	return d.rdb
}
