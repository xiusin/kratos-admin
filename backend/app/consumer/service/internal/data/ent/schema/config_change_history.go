package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// ConfigChangeHistory holds the schema definition for the ConfigChangeHistory entity.
type ConfigChangeHistory struct {
	ent.Schema
}

func (ConfigChangeHistory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "consumer_config_change_history",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("配置变更历史表"),
	}
}

// Fields of the ConfigChangeHistory.
func (ConfigChangeHistory) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("config_id").
			Comment("配置ID"),

		field.String("config_key").
			Comment("配置键").
			MaxLen(100),

		field.String("old_value").
			Comment("旧配置值").
			MaxLen(5000).
			Optional().
			Nillable(),

		field.String("new_value").
			Comment("新配置值").
			MaxLen(5000).
			Optional().
			Nillable(),

		field.Enum("change_type").
			Comment("变更类型").
			NamedValues(
				"Create", "CREATE",
				"Update", "UPDATE",
				"Delete", "DELETE",
			),

		field.String("change_reason").
			Comment("变更原因").
			MaxLen(500).
			Optional().
			Nillable(),

		field.Uint32("changed_by").
			Comment("变更人ID"),

		field.Time("changed_at").
			Comment("变更时间"),
	}
}

// Mixin of the ConfigChangeHistory.
func (ConfigChangeHistory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the ConfigChangeHistory.
func (ConfigChangeHistory) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 配置ID + 变更时间查询
		index.Fields("tenant_id", "config_id", "changed_at").
			StorageKey("idx_config_history_tenant_config_changed_at"),

		// 按租户 + 配置键 + 变更时间查询
		index.Fields("tenant_id", "config_key", "changed_at").
			StorageKey("idx_config_history_tenant_key_changed_at"),

		// 按租户 + 变更人 + 变更时间查询
		index.Fields("tenant_id", "changed_by", "changed_at").
			StorageKey("idx_config_history_tenant_changed_by_changed_at"),

		// 按租户 + 变更时间查询（用于时间范围查询）
		index.Fields("tenant_id", "changed_at").
			StorageKey("idx_config_history_tenant_changed_at"),
	}
}
