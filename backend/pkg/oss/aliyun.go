package oss

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/go-kratos/kratos/v2/log"
)

// AliyunOSSClient 阿里云OSS客户端
type AliyunOSSClient struct {
	client *oss.Client
	bucket *oss.Bucket
	config *Config
	logger *log.Helper
}

// NewAliyunOSSClient 创建阿里云OSS客户端
func NewAliyunOSSClient(cfg *Config, logger log.Logger) (*AliyunOSSClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}
	if cfg.AccessKeyID == "" {
		return nil, fmt.Errorf("access_key_id is required")
	}
	if cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("access_key_secret is required")
	}
	if cfg.BucketName == "" {
		return nil, fmt.Errorf("bucket_name is required")
	}

	l := log.NewHelper(log.With(logger, "module", "oss/aliyun"))

	// 创建OSS客户端
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create aliyun oss client: %w", err)
	}

	// 获取Bucket
	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &AliyunOSSClient{
		client: client,
		bucket: bucket,
		config: cfg,
		logger: l,
	}, nil
}

// Upload 上传文件
func (c *AliyunOSSClient) Upload(ctx context.Context, objectKey string, data []byte) (string, error) {
	if objectKey == "" {
		return "", fmt.Errorf("object_key is required")
	}
	if len(data) == 0 {
		return "", fmt.Errorf("data is empty")
	}

	reader := bytes.NewReader(data)
	err := c.bucket.PutObject(objectKey, reader)
	if err != nil {
		c.logger.Errorf("failed to upload file: %v", err)
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// 构建文件URL
	fileURL := fmt.Sprintf("https://%s.%s/%s", c.config.BucketName, c.config.Endpoint, objectKey)

	c.logger.Infof("file uploaded successfully: %s", objectKey)
	return fileURL, nil
}

// Download 下载文件
func (c *AliyunOSSClient) Download(ctx context.Context, objectKey string) ([]byte, error) {
	if objectKey == "" {
		return nil, fmt.Errorf("object_key is required")
	}

	body, err := c.bucket.GetObject(objectKey)
	if err != nil {
		c.logger.Errorf("failed to download file: %v", err)
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// Delete 删除文件
func (c *AliyunOSSClient) Delete(ctx context.Context, objectKey string) error {
	if objectKey == "" {
		return fmt.Errorf("object_key is required")
	}

	err := c.bucket.DeleteObject(objectKey)
	if err != nil {
		c.logger.Errorf("failed to delete file: %v", err)
		return fmt.Errorf("failed to delete file: %w", err)
	}

	c.logger.Infof("file deleted successfully: %s", objectKey)
	return nil
}

// GeneratePresignedURL 生成预签名URL
func (c *AliyunOSSClient) GeneratePresignedURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error) {
	if objectKey == "" {
		return "", fmt.Errorf("object_key is required")
	}

	if expireSeconds <= 0 {
		expireSeconds = 3600 // 默认1小时
	}

	signedURL, err := c.bucket.SignURL(objectKey, oss.HTTPPut, expireSeconds)
	if err != nil {
		c.logger.Errorf("failed to generate presigned url: %v", err)
		return "", fmt.Errorf("failed to generate presigned url: %w", err)
	}

	return signedURL, nil
}

// GenerateDownloadURL 生成下载预签名URL
func (c *AliyunOSSClient) GenerateDownloadURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error) {
	if objectKey == "" {
		return "", fmt.Errorf("object_key is required")
	}

	if expireSeconds <= 0 {
		expireSeconds = 3600 // 默认1小时
	}

	signedURL, err := c.bucket.SignURL(objectKey, oss.HTTPGet, expireSeconds)
	if err != nil {
		c.logger.Errorf("failed to generate download url: %v", err)
		return "", fmt.Errorf("failed to generate download url: %w", err)
	}

	return signedURL, nil
}

// Exists 检查文件是否存在
func (c *AliyunOSSClient) Exists(ctx context.Context, objectKey string) (bool, error) {
	if objectKey == "" {
		return false, fmt.Errorf("object_key is required")
	}

	exists, err := c.bucket.IsObjectExist(objectKey)
	if err != nil {
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return exists, nil
}

// GetMetadata 获取文件元数据
func (c *AliyunOSSClient) GetMetadata(ctx context.Context, objectKey string) (*FileMetadata, error) {
	if objectKey == "" {
		return nil, fmt.Errorf("object_key is required")
	}

	meta, err := c.bucket.GetObjectMeta(objectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	// 解析最后修改时间
	lastModified, _ := time.Parse(time.RFC1123, meta.Get("Last-Modified"))

	// 解析文件大小
	var size int64
	fmt.Sscanf(meta.Get("Content-Length"), "%d", &size)

	return &FileMetadata{
		Key:          objectKey,
		Size:         size,
		ContentType:  meta.Get("Content-Type"),
		ETag:         meta.Get("ETag"),
		LastModified: lastModified,
	}, nil
}

// GetProvider 获取提供商类型
func (c *AliyunOSSClient) GetProvider() Provider {
	return ProviderAliyun
}
