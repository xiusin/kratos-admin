package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// FreightTemplate holds the schema definition for the FreightTemplate entity.
type FreightTemplate struct {
	ent.Schema
}

func (FreightTemplate) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "consumer_freight_templates",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("运费模板表"),
	}
}

// Fields of the FreightTemplate.
func (FreightTemplate) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			Comment("模板名称").
			MaxLen(100),

		field.Enum("calculation_type").
			Comment("计算方式").
			NamedValues(
				"ByWeight", "BY_WEIGHT",
				"ByDistance", "BY_DISTANCE",
			),

		field.String("first_weight").
			Comment("首重（kg）").
			MaxLen(20).
			Optional().
			Nillable(),

		field.String("first_price").
			Comment("首重价格（decimal字符串）").
			MaxLen(20).
			Optional().
			Nillable(),

		field.String("additional_weight").
			Comment("续重（kg）").
			MaxLen(20).
			Optional().
			Nillable(),

		field.String("additional_price").
			Comment("续重价格（decimal字符串）").
			MaxLen(20).
			Optional().
			Nillable(),

		field.JSON("region_rules", []map[string]interface{}{}).
			Comment("地区规则").
			Optional(),

		field.JSON("free_shipping_rules", []map[string]interface{}{}).
			Comment("包邮规则").
			Optional(),

		field.Bool("is_active").
			Comment("是否启用").
			Default(true),
	}
}

// Mixin of the FreightTemplate.
func (FreightTemplate) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.OperatorID{},
		mixin.TimeAt{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the FreightTemplate.
func (FreightTemplate) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 是否启用查询
		index.Fields("tenant_id", "is_active").
			StorageKey("idx_freight_template_tenant_is_active"),

		// 按租户 + 计算方式查询
		index.Fields("tenant_id", "calculation_type").
			StorageKey("idx_freight_template_tenant_calculation_type"),

		// 按租户 + 创建时间查询
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_freight_template_tenant_created_at"),
	}
}
