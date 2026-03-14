package antivirus

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// tencentScanner 腾讯云天御扫描器
type tencentScanner struct {
	endpoint  string
	accessKey string
	secretKey string
	client    *http.Client
}

// NewTencentScanner 创建腾讯云天御扫描器
func NewTencentScanner(cfg *Config) (Scanner, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://tms.tencentcloudapi.com"
	}

	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("access_key and secret_key are required")
	}

	return &tencentScanner{
		endpoint:  cfg.Endpoint,
		accessKey: cfg.AccessKey,
		secretKey: cfg.SecretKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Scan 扫描文件
func (s *tencentScanner) Scan(ctx context.Context, data []byte) (*ScanResult, error) {
	// 腾讯云天御API需要Base64编码的文件内容
	// 这里简化实现，实际项目中需要完整的API调用

	// 构建请求参数
	params := map[string]string{
		"Action":    "ImageModeration",
		"Version":   "2020-12-29",
		"Region":    "ap-guangzhou",
		"Timestamp": fmt.Sprintf("%d", time.Now().Unix()),
		"Nonce":     fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	// 签名
	signature := s.sign(params)
	params["Signature"] = signature

	// 发送请求
	resp, err := s.doRequest(ctx, params, data)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return s.parseResponse(resp)
}

// ScanFile 扫描文件（通过路径）
func (s *tencentScanner) ScanFile(ctx context.Context, filePath string) (*ScanResult, error) {
	// 腾讯云天御不支持直接扫描文件路径
	// 需要先读取文件内容
	return nil, fmt.Errorf("tencent scanner does not support file path scanning")
}

// GetProvider 获取扫描器提供商
func (s *tencentScanner) GetProvider() string {
	return "tencent"
}

// sign 签名请求
func (s *tencentScanner) sign(params map[string]string) string {
	// 1. 对参数排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 构建签名字符串
	var signStr strings.Builder
	signStr.WriteString("GET")
	signStr.WriteString(s.endpoint)
	signStr.WriteString("/?")

	for i, k := range keys {
		if i > 0 {
			signStr.WriteString("&")
		}
		signStr.WriteString(k)
		signStr.WriteString("=")
		signStr.WriteString(url.QueryEscape(params[k]))
	}

	// 3. HMAC-SHA256签名
	h := hmac.New(sha256.New, []byte(s.secretKey))
	h.Write([]byte(signStr.String()))
	signature := hex.EncodeToString(h.Sum(nil))

	return signature
}

// doRequest 发送HTTP请求
func (s *tencentScanner) doRequest(ctx context.Context, params map[string]string, data []byte) ([]byte, error) {
	// 构建URL
	reqURL := s.endpoint + "/?"
	values := url.Values{}
	for k, v := range params {
		values.Add(k, v)
	}
	reqURL += values.Encode()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// 发送请求
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// parseResponse 解析响应
func (s *tencentScanner) parseResponse(data []byte) (*ScanResult, error) {
	// 解析JSON响应
	var response struct {
		Response struct {
			Suggestion string `json:"Suggestion"` // Pass/Review/Block
			Label      string `json:"Label"`      // 标签
			SubLabel   string `json:"SubLabel"`   // 子标签
			RequestID  string `json:"RequestId"`
		} `json:"Response"`
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("parse response failed: %w", err)
	}

	// 判断结果
	clean := response.Response.Suggestion == "Pass"
	virusName := ""
	message := "file is clean"

	if !clean {
		virusName = response.Response.Label
		if response.Response.SubLabel != "" {
			virusName += "/" + response.Response.SubLabel
		}
		message = fmt.Sprintf("content violation detected: %s", virusName)
	}

	return &ScanResult{
		Clean:     clean,
		VirusName: virusName,
		Message:   message,
	}, nil
}
