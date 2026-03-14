package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// FinanceTransaction holds the schema definition for the FinanceTransaction entity.
type FinanceTransaction struct {
	ent.Schema
}

func (FinanceTransaction) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "consumer_finance_transactions",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("财务流水表"),
	}
}

// Fields of the FinanceTransaction.
func (FinanceTransaction) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("consumer_id").
			Comment("用户ID"),

		field.String("transaction_no").
			Comment("流水号").
			MaxLen(64).
			NotEmpty(),

		field.Enum("transaction_type").
			Comment("交易类型").
			NamedValues(
				"Recharge", "RECHARGE",
				"Consume", "CONSUME",
				"Withdraw", "WITHDRAW",
				"Refund", "REFUND",
			),

		field.String("amount").
			Comment("交易金额（decimal字符串）").
			MaxLen(20),

		field.String("balance_before").
			Comment("交易前余额（decimal字符串）").
			MaxLen(20),

		field.String("balance_after").
			Comment("交易后余额（decimal字符串）").
			MaxLen(20),

		field.String("description").
			Comment("交易描述").
			MaxLen(200).
			Optional().
			Nillable(),

		field.String("related_order_no").
			Comment("关联订单号").
			MaxLen(64).
			Optional().
			Nillable(),

		field.Uint32("operator_id").
			Comment("操作人ID").
			Optional().
			Nillable(),
	}
}

// Mixin of the FinanceTransaction.
func (FinanceTransaction) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.CreatedAt{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the FinanceTransaction.
func (FinanceTransaction) Indexes() []ent.Index {
	return []ent.Index{
		// 保证 transaction_no 全局唯一
		index.Fields("transaction_no").
			Unique().
			StorageKey("idx_finance_transaction_transaction_no"),

		// 按租户 + 用户 + 创建时间查询
		index.Fields("tenant_id", "consumer_id", "created_at").
			StorageKey("idx_finance_transaction_tenant_consumer_created_at"),

		// 按租户 + 交易类型 + 创建时间查询
		index.Fields("tenant_id", "transaction_type", "created_at").
			StorageKey("idx_finance_transaction_tenant_type_created_at"),

		// 按关联订单号查询
		index.Fields("related_order_no").
			StorageKey("idx_finance_transaction_related_order_no"),

		// 按租户 + 创建时间查询（用于时间范围查询）
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_finance_transaction_tenant_created_at"),
	}
}
