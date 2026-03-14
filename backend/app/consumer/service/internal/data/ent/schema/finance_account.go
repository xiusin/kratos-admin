package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// FinanceAccount holds the schema definition for the FinanceAccount entity.
type FinanceAccount struct {
	ent.Schema
}

func (FinanceAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "consumer_finance_accounts",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("财务账户表"),
	}
}

// Fields of the FinanceAccount.
func (FinanceAccount) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("consumer_id").
			Comment("用户ID"),

		field.String("balance").
			Comment("账户余额（decimal字符串）").
			MaxLen(20).
			Default("0"),

		field.String("frozen_balance").
			Comment("冻结余额（decimal字符串）").
			MaxLen(20).
			Default("0"),
	}
}

// Mixin of the FinanceAccount.
func (FinanceAccount) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.TimeAt{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the FinanceAccount.
func (FinanceAccount) Indexes() []ent.Index {
	return []ent.Index{
		// 在租户范围内保证 consumer_id 唯一
		index.Fields("tenant_id", "consumer_id").
			Unique().
			StorageKey("idx_finance_account_tenant_consumer"),
	}
}
