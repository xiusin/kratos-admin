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
			Table:     "tenant_configs",
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
			Comment("配置值").
			MaxLen(2000).
			Optional().
			Nillable(),

		field.String("config_type").
			Comment("配置类型: string/int/bool/json/encrypted").
			MaxLen(20).
			Default("string"),

		field.String("description").
			Comment("配置描述").
			MaxLen(500).
			Optional().
			Nillable(),

		field.String("category").
			Comment("配置分类: sms/payment/wechat/media/logistics/freight/system").
			MaxLen(50).
			Optional().
			Nillable(),

		field.Bool("is_encrypted").
			Comment("是否加密存储").
			Default(false),

		field.Bool("is_active").
			Comment("是否启用").
			Default(true),

		field.String("validation_rule").
			Comment("验证规则（JSON格式）").
			MaxLen(500).
			Optional().
			Nillable(),
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
		// 在租户范围内保证 config_key 唯一
		index.Fields("tenant_id", "config_key").
			Unique().
			StorageKey("idx_tenant_config_tenant_key"),

		// 按租户 + 分类查询
		index.Fields("tenant_id", "category").
			StorageKey("idx_tenant_config_tenant_category"),

		// 按租户 + 启用状态查询
		index.Fields("tenant_id", "is_active").
			StorageKey("idx_tenant_config_tenant_active"),

		// 按租户 + 创建时间查询
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_tenant_config_tenant_created_at"),
	}
}
