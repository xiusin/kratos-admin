package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// PaymentOrder holds the schema definition for the PaymentOrder entity.
type PaymentOrder struct {
	ent.Schema
}

func (PaymentOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "consumer_payment_orders",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("支付订单表"),
	}
}

// Fields of the PaymentOrder.
func (PaymentOrder) Fields() []ent.Field {
	return []ent.Field{
		field.String("order_no").
			Comment("订单号").
			MaxLen(64).
			NotEmpty(),

		field.Uint32("consumer_id").
			Comment("用户ID"),

		field.Enum("payment_method").
			Comment("支付方式").
			NamedValues(
				"Wechat", "WECHAT",
				"Alipay", "ALIPAY",
				"Yeepay", "YEEPAY",
			),

		field.Enum("payment_type").
			Comment("支付类型").
			NamedValues(
				"App", "APP",
				"H5", "H5",
				"Mini", "MINI",
				"Qrcode", "QRCODE",
			),

		field.String("amount").
			Comment("支付金额（decimal字符串）").
			MaxLen(20),

		field.Enum("status").
			Comment("订单状态").
			NamedValues(
				"Pending", "PENDING",
				"Success", "SUCCESS",
				"Failed", "FAILED",
				"Closed", "CLOSED",
				"Refunded", "REFUNDED",
			).
			Default("PENDING"),

		field.String("transaction_id").
			Comment("第三方交易号").
			MaxLen(100).
			Optional().
			Nillable(),

		field.String("callback_data").
			Comment("回调数据").
			MaxLen(2000).
			Optional().
			Nillable(),

		field.Time("paid_at").
			Comment("支付时间").
			Optional().
			Nillable(),

		field.Time("closed_at").
			Comment("关闭时间").
			Optional().
			Nillable(),

		field.Time("expires_at").
			Comment("过期时间"),
	}
}

// Mixin of the PaymentOrder.
func (PaymentOrder) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the PaymentOrder.
func (PaymentOrder) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 order_no 唯一
		index.Fields("tenant_id", "order_no").
			Unique().
			StorageKey("idx_payment_order_tenant_order_no"),

		// 按租户 + 用户 + 状态查询
		index.Fields("tenant_id", "consumer_id", "status").
			StorageKey("idx_payment_order_tenant_consumer_status"),

		// 按第三方交易号查询
		index.Fields("transaction_id").
			StorageKey("idx_payment_order_transaction_id"),

		// 按租户 + 支付方式查询
		index.Fields("tenant_id", "payment_method").
			StorageKey("idx_payment_order_tenant_payment_method"),

		// 按租户 + 状态查询
		index.Fields("tenant_id", "status").
			StorageKey("idx_payment_order_tenant_status"),

		// 按租户 + 过期时间查询（用于定时关闭超时订单）
		index.Fields("tenant_id", "expires_at").
			StorageKey("idx_payment_order_tenant_expires_at"),

		// 按租户 + 创建时间查询（用于时间范围查询）
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_payment_order_tenant_created_at"),
	}
}
