package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// TenantConfig holds the schema definition for the TenantConfig entity.
type TenantConfig struct {
	ent.Schema
}

func (TenantConfig) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "consumer_tenant_configs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("租户配置表"),
	}
}

// Fields of the TenantConfig.
func (TenantConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("config_key").
			Comment("配置键").
			MaxLen(100).
			NotEmpty(),

		field.String("config_value").
			Comment("配置值（JSON字符串）").
			MaxLen(5000).
			Optional().
			Nillable(),

		field.String("description").
			Comment("配置描述").
			MaxLen(500).
			Optional().
			Nillable(),

		field.Enum("config_type").
			Comment("配置类型").
			NamedValues(
				"SMS", "SMS", // 短信配置
				"Payment", "PAYMENT", // 支付配置
				"OSS", "OSS", // 对象存储配置
				"Wechat", "WECHAT", // 微信配置
				"Logistics", "LOGISTICS", // 物流配置
				"Freight", "FREIGHT", // 运费配置
				"System", "SYSTEM", // 系统配置
			).
			Default("SYSTEM"),

		field.Bool("is_encrypted").
			Comment("是否加密存储").
			Default(false),

		field.Bool("is_active").
			Comment("是否启用").
			Default(true),
	}
}

// Mixin of the TenantConfig.
func (TenantConfig) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.OperatorID{},
		mixin.TimeAt{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the TenantConfig.
func (TenantConfig) Indexes() []ent.Index {
	return []ent.Index{
		// 租户 + 配置键唯一索引
		index.Fields("tenant_id", "config_key").
			Unique().
			StorageKey("idx_tenant_config_tenant_key"),

		// 按租户 + 配置类型查询
		index.Fields("tenant_id", "config_type").
			StorageKey("idx_tenant_config_tenant_type"),

		// 按租户 + 是否启用查询
		index.Fields("tenant_id", "is_active").
			StorageKey("idx_tenant_config_tenant_is_active"),
	}
}
