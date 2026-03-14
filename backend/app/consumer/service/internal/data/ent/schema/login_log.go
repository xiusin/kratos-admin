package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// LoginLog holds the schema definition for the LoginLog entity.
type LoginLog struct {
	ent.Schema
}

func (LoginLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "consumer_login_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("C端用户登录日志表"),
	}
}

// Fields of the LoginLog.
func (LoginLog) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("consumer_id").
			Comment("用户ID"),

		field.String("phone").
			Comment("手机号").
			MaxLen(20),

		field.Enum("login_type").
			Comment("登录方式").
			NamedValues(
				"Phone", "PHONE",
				"Wechat", "WECHAT",
			),

		field.Bool("success").
			Comment("是否成功").
			Default(false),

		field.String("fail_reason").
			Comment("失败原因").
			MaxLen(200).
			Optional().
			Nillable(),

		field.String("ip_address").
			Comment("IP地址").
			MaxLen(50),

		field.String("user_agent").
			Comment("User Agent").
			MaxLen(500).
			Optional().
			Nillable(),

		field.String("device_type").
			Comment("设备类型").
			MaxLen(50).
			Optional().
			Nillable(),

		field.Time("login_at").
			Comment("登录时间"),
	}
}

// Mixin of the LoginLog.
func (LoginLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the LoginLog.
func (LoginLog) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 用户 + 登录时间查询
		index.Fields("tenant_id", "consumer_id", "login_at").
			StorageKey("idx_login_log_tenant_consumer_login_at"),

		// 按租户 + 手机号 + 登录时间查询
		index.Fields("tenant_id", "phone", "login_at").
			StorageKey("idx_login_log_tenant_phone_login_at"),

		// 按租户 + 登录类型查询
		index.Fields("tenant_id", "login_type").
			StorageKey("idx_login_log_tenant_login_type"),

		// 按租户 + 是否成功查询
		index.Fields("tenant_id", "success").
			StorageKey("idx_login_log_tenant_success"),

		// 按租户 + IP地址查询
		index.Fields("tenant_id", "ip_address").
			StorageKey("idx_login_log_tenant_ip_address"),

		// 按租户 + 登录时间查询（用于时间范围查询）
		index.Fields("tenant_id", "login_at").
			StorageKey("idx_login_log_tenant_login_at"),
	}
}
