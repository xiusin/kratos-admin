package oss

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tencentyun/cos-go-sdk-v5"
)

// TencentCOSClient 腾讯云COS客户端
type TencentCOSClient struct {
	client *cos.Client
	config *Config
	logger *log.Helper
}

// NewTencentCOSClient 创建腾讯云COS客户端
func NewTencentCOSClient(cfg *Config, logger log.Logger) (*TencentCOSClient, error) {
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

	l := log.NewHelper(log.With(logger, "module", "oss/tencent"))

	// 构建Bucket URL
	bucketURL, err := url.Parse(fmt.Sprintf("https://%s.%s", cfg.BucketName, cfg.Endpoint))
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}

	// 创建COS客户端
	baseURL := &cos.BaseURL{BucketURL: bucketURL}
	client := cos.NewClient(baseURL, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cfg.AccessKeyID,
			SecretKey: cfg.AccessKeySecret,
		},
	})

	return &TencentCOSClient{
		client: client,
		config: cfg,
		logger: l,
	}, nil
}

// Upload 上传文件
func (c *TencentCOSClient) Upload(ctx context.Context, objectKey string, data []byte) (string, error) {
	if objectKey == "" {
		return "", fmt.Errorf("object_key is required")
	}
	if len(data) == 0 {
		return "", fmt.Errorf("data is empty")
	}

	reader := bytes.NewReader(data)
	_, err := c.client.Object.Put(ctx, objectKey, reader, nil)
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
func (c *TencentCOSClient) Download(ctx context.Context, objectKey string) ([]byte, error) {
	if objectKey == "" {
		return nil, fmt.Errorf("object_key is required")
	}

	resp, err := c.client.Object.Get(ctx, objectKey, nil)
	if err != nil {
		c.logger.Errorf("failed to download file: %v", err)
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// Delete 删除文件
func (c *TencentCOSClient) Delete(ctx context.Context, objectKey string) error {
	if objectKey == "" {
		return fmt.Errorf("object_key is required")
	}

	_, err := c.client.Object.Delete(ctx, objectKey)
	if err != nil {
		c.logger.Errorf("failed to delete file: %v", err)
		return fmt.Errorf("failed to delete file: %w", err)
	}

	c.logger.Infof("file deleted successfully: %s", objectKey)
	return nil
}

// GeneratePresignedURL 生成预签名URL
func (c *TencentCOSClient) GeneratePresignedURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error) {
	if objectKey == "" {
		return "", fmt.Errorf("object_key is required")
	}

	if expireSeconds <= 0 {
		expireSeconds = 3600 // 默认1小时
	}

	presignedURL, err := c.client.Object.GetPresignedURL(ctx, http.MethodPut, objectKey, c.config.AccessKeyID, c.config.AccessKeySecret, time.Duration(expireSeconds)*time.Second, nil)
	if err != nil {
		c.logger.Errorf("failed to generate presigned url: %v", err)
		return "", fmt.Errorf("failed to generate presigned url: %w", err)
	}

	return presignedURL.String(), nil
}

// GenerateDownloadURL 生成下载预签名URL
func (c *TencentCOSClient) GenerateDownloadURL(ctx context.Context, objectKey string, expireSeconds int64) (string, error) {
	if objectKey == "" {
		return "", fmt.Errorf("object_key is required")
	}

	if expireSeconds <= 0 {
		expireSeconds = 3600 // 默认1小时
	}

	presignedURL, err := c.client.Object.GetPresignedURL(ctx, http.MethodGet, objectKey, c.config.AccessKeyID, c.config.AccessKeySecret, time.Duration(expireSeconds)*time.Second, nil)
	if err != nil {
		c.logger.Errorf("failed to generate download url: %v", err)
		return "", fmt.Errorf("failed to generate download url: %w", err)
	}

	return presignedURL.String(), nil
}

// Exists 检查文件是否存在
func (c *TencentCOSClient) Exists(ctx context.Context, objectKey string) (bool, error) {
	if objectKey == "" {
		return false, fmt.Errorf("object_key is required")
	}

	_, err := c.client.Object.Head(ctx, objectKey, nil)
	if err != nil {
		if cos.IsNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}

	return true, nil
}

// GetMetadata 获取文件元数据
func (c *TencentCOSClient) GetMetadata(ctx context.Context, objectKey string) (*FileMetadata, error) {
	if objectKey == "" {
		return nil, fmt.Errorf("object_key is required")
	}

	resp, err := c.client.Object.Head(ctx, objectKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get file metadata: %w", err)
	}

	// 解析最后修改时间
	lastModified, _ := time.Parse(time.RFC1123, resp.Header.Get("Last-Modified"))

	return &FileMetadata{
		Key:          objectKey,
		Size:         resp.ContentLength,
		ContentType:  resp.Header.Get("Content-Type"),
		ETag:         resp.Header.Get("ETag"),
		LastModified: lastModified,
	}, nil
}

// GetProvider 获取提供商类型
func (c *TencentCOSClient) GetProvider() Provider {
	return ProviderTencent
}
