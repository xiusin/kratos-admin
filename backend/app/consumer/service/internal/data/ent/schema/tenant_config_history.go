package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// TenantConfigHistory holds the schema definition for the TenantConfigHistory entity.
type TenantConfigHistory struct {
	ent.Schema
}

func (TenantConfigHistory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "tenant_config_histories",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("租户配置变更历史表"),
	}
}

// Fields of the TenantConfigHistory.
func (TenantConfigHistory) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("config_id").
			Comment("配置ID"),

		field.String("config_key").
			Comment("配置键").
			MaxLen(100).
			NotEmpty(),

		field.String("old_value").
			Comment("旧值").
			MaxLen(2000).
			Optional().
			Nillable(),

		field.String("new_value").
			Comment("新值").
			MaxLen(2000).
			Optional().
			Nillable(),

		field.Enum("change_type").
			Comment("变更类型").
			NamedValues(
				"Create", "CREATE",
				"Update", "UPDATE",
				"Delete", "DELETE",
				"Rollback", "ROLLBACK",
			).
			Default("UPDATE"),

		field.String("change_reason").
			Comment("变更原因").
			MaxLen(500).
			Optional().
			Nillable(),

		field.Uint32("changed_by").
			Comment("变更人ID"),

		field.String("changed_by_name").
			Comment("变更人姓名").
			MaxLen(100).
			Optional().
			Nillable(),
	}
}

// Mixin of the TenantConfigHistory.
func (TenantConfigHistory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the TenantConfigHistory.
func (TenantConfigHistory) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 配置ID查询历史
		index.Fields("tenant_id", "config_id", "created_at").
			StorageKey("idx_config_history_tenant_config_time"),

		// 按租户 + 配置键查询历史
		index.Fields("tenant_id", "config_key", "created_at").
			StorageKey("idx_config_history_tenant_key_time"),

		// 按租户 + 变更人查询
		index.Fields("tenant_id", "changed_by", "created_at").
			StorageKey("idx_config_history_tenant_changer_time"),
	}
}
