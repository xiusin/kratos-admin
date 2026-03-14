package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// SMSLog holds the schema definition for the SMSLog entity.
type SMSLog struct {
	ent.Schema
}

func (SMSLog) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "consumer_sms_logs",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("短信日志表"),
	}
}

// Fields of the SMSLog.
func (SMSLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("phone").
			Comment("手机号").
			MaxLen(20),

		field.Enum("sms_type").
			Comment("短信类型").
			NamedValues(
				"Verification", "VERIFICATION",
				"Notification", "NOTIFICATION",
			),

		field.String("content").
			Comment("短信内容").
			MaxLen(500),

		field.String("code").
			Comment("验证码").
			MaxLen(10).
			Optional().
			Nillable(),

		field.Enum("channel").
			Comment("短信通道").
			NamedValues(
				"Aliyun", "ALIYUN",
				"Tencent", "TENCENT",
			),

		field.Enum("status").
			Comment("发送状态").
			NamedValues(
				"Pending", "PENDING",
				"Success", "SUCCESS",
				"Failed", "FAILED",
			).
			Default("PENDING"),

		field.String("error_message").
			Comment("错误信息").
			MaxLen(200).
			Optional().
			Nillable(),

		field.Time("sent_at").
			Comment("发送时间"),

		field.Time("expires_at").
			Comment("过期时间").
			Optional().
			Nillable(),
	}
}

// Mixin of the SMSLog.
func (SMSLog) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the SMSLog.
func (SMSLog) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 手机号 + 发送时间查询
		index.Fields("tenant_id", "phone", "sent_at").
			StorageKey("idx_sms_log_tenant_phone_sent_at"),

		// 按租户 + 短信类型 + 状态查询
		index.Fields("tenant_id", "sms_type", "status").
			StorageKey("idx_sms_log_tenant_sms_type_status"),

		// 按租户 + 通道查询
		index.Fields("tenant_id", "channel").
			StorageKey("idx_sms_log_tenant_channel"),

		// 按租户 + 发送时间查询（用于时间范围查询）
		index.Fields("tenant_id", "sent_at").
			StorageKey("idx_sms_log_tenant_sent_at"),

		// 按租户 + 过期时间查询（用于清理过期验证码）
		index.Fields("tenant_id", "expires_at").
			StorageKey("idx_sms_log_tenant_expires_at"),
	}
}
