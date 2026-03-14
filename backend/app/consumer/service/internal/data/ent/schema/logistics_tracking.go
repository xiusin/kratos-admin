package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// LogisticsTracking holds the schema definition for the LogisticsTracking entity.
type LogisticsTracking struct {
	ent.Schema
}

func (LogisticsTracking) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "consumer_logistics_trackings",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("物流跟踪表"),
	}
}

// Fields of the LogisticsTracking.
func (LogisticsTracking) Fields() []ent.Field {
	return []ent.Field{
		field.String("tracking_no").
			Comment("运单号").
			MaxLen(100),

		field.String("courier_company").
			Comment("快递公司").
			MaxLen(50),

		field.Enum("status").
			Comment("物流状态").
			NamedValues(
				"Pending", "PENDING",
				"InTransit", "IN_TRANSIT",
				"Delivering", "DELIVERING",
				"Delivered", "DELIVERED",
			).
			Default("PENDING"),

		field.JSON("tracking_info", []map[string]interface{}{}).
			Comment("物流轨迹"),

		field.Time("last_updated_at").
			Comment("最后更新时间"),
	}
}

// Mixin of the LogisticsTracking.
func (LogisticsTracking) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the LogisticsTracking.
func (LogisticsTracking) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 tracking_no 唯一
		index.Fields("tenant_id", "tracking_no").
			Unique().
			StorageKey("idx_logistics_tracking_tenant_tracking_no"),

		// 按租户 + 状态查询
		index.Fields("tenant_id", "status").
			StorageKey("idx_logistics_tracking_tenant_status"),

		// 按租户 + 快递公司查询
		index.Fields("tenant_id", "courier_company").
			StorageKey("idx_logistics_tracking_tenant_courier_company"),

		// 按租户 + 最后更新时间查询
		index.Fields("tenant_id", "last_updated_at").
			StorageKey("idx_logistics_tracking_tenant_last_updated_at"),

		// 按租户 + 创建时间查询
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_logistics_tracking_tenant_created_at"),
	}
}
