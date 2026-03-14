package antivirus

import (
	"context"
	"fmt"
)

// ScanResult 扫描结果
type ScanResult struct {
	// Clean 是否干净（无病毒）
	Clean bool

	// VirusName 病毒名称（如果检测到）
	VirusName string

	// Message 扫描消息
	Message string
}

// Scanner 病毒扫描器接口
type Scanner interface {
	// Scan 扫描文件
	Scan(ctx context.Context, data []byte) (*ScanResult, error)

	// ScanFile 扫描文件（通过路径）
	ScanFile(ctx context.Context, filePath string) (*ScanResult, error)

	// GetProvider 获取扫描器提供商
	GetProvider() string
}

// Config 扫描器配置
type Config struct {
	// Provider 提供商 (clamav/tencent/aliyun/mock)
	Provider string `json:"provider"`

	// Endpoint API端点（云服务）
	Endpoint string `json:"endpoint"`

	// AccessKey 访问密钥（云服务）
	AccessKey string `json:"access_key"`

	// SecretKey 密钥（云服务）
	SecretKey string `json:"secret_key"`

	// ClamAVHost ClamAV服务器地址
	ClamAVHost string `json:"clamav_host"`

	// ClamAVPort ClamAV服务器端口
	ClamAVPort int `json:"clamav_port"`
}

// mockScanner Mock扫描器（用于开发和测试）
type mockScanner struct{}

// NewMockScanner 创建Mock扫描器
func NewMockScanner() Scanner {
	return &mockScanner{}
}

// Scan 扫描文件
func (s *mockScanner) Scan(ctx context.Context, data []byte) (*ScanResult, error) {
	// Mock实现：总是返回干净
	// 实际项目中应该集成真实的病毒扫描服务
	return &ScanResult{
		Clean:     true,
		VirusName: "",
		Message:   "mock scan: file is clean",
	}, nil
}

// ScanFile 扫描文件（通过路径）
func (s *mockScanner) ScanFile(ctx context.Context, filePath string) (*ScanResult, error) {
	return &ScanResult{
		Clean:     true,
		VirusName: "",
		Message:   fmt.Sprintf("mock scan: file %s is clean", filePath),
	}, nil
}

// GetProvider 获取扫描器提供商
func (s *mockScanner) GetProvider() string {
	return "mock"
}

// NewScanner 创建扫描器
func NewScanner(cfg *Config) (Scanner, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	switch cfg.Provider {
	case "mock":
		return NewMockScanner(), nil
	case "clamav":
		// 实际项目中实现ClamAV集成
		// return NewClamAVScanner(cfg)
		return nil, fmt.Errorf("clamav scanner not implemented yet")
	case "tencent":
		// 实际项目中实现腾讯云天御集成
		// return NewTencentScanner(cfg)
		return nil, fmt.Errorf("tencent scanner not implemented yet")
	case "aliyun":
		// 实际项目中实现阿里云内容安全集成
		// return NewAliyunScanner(cfg)
		return nil, fmt.Errorf("aliyun scanner not implemented yet")
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.Provider)
	}
}
