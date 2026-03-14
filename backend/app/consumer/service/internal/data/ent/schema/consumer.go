package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// Consumer holds the schema definition for the Consumer entity.
type Consumer struct {
	ent.Schema
}

func (Consumer) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "consumer_users",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("C端用户表"),
	}
}

// Fields of the Consumer.
func (Consumer) Fields() []ent.Field {
	return []ent.Field{
		field.String("phone").
			Comment("手机号").
			MaxLen(20).
			NotEmpty(),

		field.String("email").
			Comment("邮箱").
			MaxLen(100).
			Optional().
			Nillable(),

		field.String("nickname").
			Comment("昵称").
			MaxLen(50).
			Optional().
			Nillable(),

		field.String("avatar").
			Comment("头像URL").
			MaxLen(500).
			Optional().
			Nillable(),

		field.String("password_hash").
			Comment("密码哈希").
			MaxLen(255).
			Sensitive(),

		field.String("wechat_openid").
			Comment("微信OpenID").
			MaxLen(100).
			Optional().
			Nillable(),

		field.String("wechat_unionid").
			Comment("微信UnionID").
			MaxLen(100).
			Optional().
			Nillable(),

		field.Enum("status").
			Comment("状态").
			NamedValues(
				"Normal", "NORMAL",
				"Locked", "LOCKED",
				"Deactivated", "DEACTIVATED",
			).
			Default("NORMAL"),

		field.Int("risk_score").
			Comment("风险评分 0-100").
			Default(0).
			Min(0).
			Max(100),

		field.Int("login_fail_count").
			Comment("登录失败次数").
			Default(0).
			Min(0),

		field.Time("locked_until").
			Comment("锁定截止时间").
			Optional().
			Nillable(),

		field.Time("last_login_at").
			Comment("最后登录时间").
			Optional().
			Nillable(),

		field.String("last_login_ip").
			Comment("最后登录IP").
			MaxLen(50).
			Optional().
			Nillable(),

		field.Time("deactivated_at").
			Comment("注销时间").
			Optional().
			Nillable(),
	}
}

// Mixin of the Consumer.
func (Consumer) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.OperatorID{},
		mixin.TimeAt{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the Consumer.
func (Consumer) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 phone 唯一
		index.Fields("tenant_id", "phone").
			Unique().
			StorageKey("idx_consumer_tenant_phone"),

		// 在租户范围内保证 email 唯一（email 可为空，DB 上允许多个 NULL）
		index.Fields("tenant_id", "email").
			Unique().
			StorageKey("idx_consumer_tenant_email"),

		// 按租户 + 微信 openid 查询
		index.Fields("tenant_id", "wechat_openid").
			StorageKey("idx_consumer_tenant_wechat_openid"),

		// 按租户 + 微信 unionid 查询
		index.Fields("tenant_id", "wechat_unionid").
			StorageKey("idx_consumer_tenant_wechat_unionid"),

		// 按租户 + 状态查询
		index.Fields("tenant_id", "status").
			StorageKey("idx_consumer_tenant_status"),

		// 按租户 + 最后登录时间，用于按时间范围检索
		index.Fields("tenant_id", "last_login_at").
			StorageKey("idx_consumer_tenant_last_login_at"),

		// 按租户 + 创建时间，用于租户范围的时间区间查询与分页
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_consumer_tenant_created_at"),
	}
}
