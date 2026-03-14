package oss

import (
	"context"
	"time"
)

// Provider OSS提供商类型
type Provider string

const (
	ProviderAliyun  Provider = "aliyun"  // 阿里云OSS
	ProviderTencent Provider = "tencent" // 腾讯云COS
	ProviderMinio   Provider = "minio"   // MinIO
)

// Client OSS客户端接口
type Client interface {
	// Upload 上传文件
	Upload(ctx context.Context, objectKey string, data []byte) (string, error)

	// Download 下载文件
	Download(ctx context.Context, objectKey string) ([]byte, error)

	// Delete 删除文件
	Delete(ctx context.Context, objectKey string) error

	// GeneratePresignedURL 生成上传预签名URL
	GeneratePresignedURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error)

	// GenerateDownloadURL 生成下载预签名URL
	GenerateDownloadURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error)

	// Exists 检查文件是否存在
	Exists(ctx context.Context, objectKey string) (bool, error)

	// GetMetadata 获取文件元数据
	GetMetadata(ctx context.Context, objectKey string) (*FileMetadata, error)

	// GetProvider 获取提供商类型
	GetProvider() Provider
}

// Config OSS配置
type Config struct {
	// Provider 提供商 (aliyun/tencent/minio)
	Provider Provider `json:"provider"`

	// Endpoint 端点地址
	Endpoint string `json:"endpoint"`

	// AccessKeyID 访问密钥ID
	AccessKeyID string `json:"access_key_id"`

	// AccessKeySecret 访问密钥Secret
	AccessKeySecret string `json:"access_key_secret"`

	// BucketName Bucket名称
	BucketName string `json:"bucket_name"`

	// Region 区域（可选）
	Region string `json:"region"`

	// UseSSL 是否使用SSL（MinIO）
	UseSSL bool `json:"use_ssl"`
}

// FileMetadata 文件元数据
type FileMetadata struct {
	// Key 对象键
	Key string

	// Size 文件大小（字节）
	Size int64

	// ContentType 内容类型
	ContentType string

	// ETag 文件ETag
	ETag string

	// LastModified 最后修改时间
	LastModified time.Time
}
