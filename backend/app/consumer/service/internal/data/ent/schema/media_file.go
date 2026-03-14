package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/tx7do/go-crud/entgo/mixin"
)

// MediaFile holds the schema definition for the MediaFile entity.
type MediaFile struct {
	ent.Schema
}

func (MediaFile) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table:     "consumer_media_files",
			Charset:   "utf8mb4",
			Collation: "utf8mb4_bin",
		},
		entsql.WithComments(true),
		schema.Comment("媒体文件表"),
	}
}

// Fields of the MediaFile.
func (MediaFile) Fields() []ent.Field {
	return []ent.Field{
		field.Uint32("consumer_id").
			Comment("上传者ID"),

		field.String("file_name").
			Comment("文件名").
			MaxLen(255).
			NotEmpty(),

		field.Enum("file_type").
			Comment("文件类型").
			NamedValues(
				"Image", "IMAGE",
				"Video", "VIDEO",
			),

		field.String("file_format").
			Comment("文件格式(JPEG/PNG/GIF/MP4/AVI/MOV)").
			MaxLen(20).
			NotEmpty(),

		field.Uint64("file_size").
			Comment("文件大小(字节)"),

		field.String("file_url").
			Comment("文件URL").
			MaxLen(500).
			NotEmpty(),

		field.String("thumbnail_url").
			Comment("缩略图URL").
			MaxLen(500).
			Optional().
			Nillable(),

		field.String("oss_bucket").
			Comment("OSS Bucket").
			MaxLen(100).
			NotEmpty(),

		field.String("oss_key").
			Comment("OSS Key").
			MaxLen(500).
			NotEmpty(),

		field.Bool("is_deleted").
			Comment("是否删除").
			Default(false),

		field.Time("deleted_at").
			Comment("删除时间").
			Optional().
			Nillable(),
	}
}

// Mixin of the MediaFile.
func (MediaFile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.AutoIncrementId{},
		mixin.OperatorID{},
		mixin.TimeAt{},
		mixin.TenantID[uint32]{},
	}
}

// Indexes of the MediaFile.
func (MediaFile) Indexes() []ent.Index {
	return []ent.Index{
		// 按租户 + 上传者 + 是否删除查询
		index.Fields("tenant_id", "consumer_id", "is_deleted").
			StorageKey("idx_media_file_tenant_consumer_deleted"),

		// 按租户 + 文件类型 + 创建时间查询
		index.Fields("tenant_id", "file_type", "created_at").
			StorageKey("idx_media_file_tenant_type_created"),

		// 按租户 + 创建时间查询
		index.Fields("tenant_id", "created_at").
			StorageKey("idx_media_file_tenant_created"),
	}
}
