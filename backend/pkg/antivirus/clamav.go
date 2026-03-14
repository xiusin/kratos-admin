package antivirus

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// clamavScanner ClamAV扫描器
type clamavScanner struct {
	host string
	port int
}

// NewClamAVScanner 创建ClamAV扫描器
func NewClamAVScanner(cfg *Config) (Scanner, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.ClamAVHost == "" {
		cfg.ClamAVHost = "localhost"
	}
	if cfg.ClamAVPort == 0 {
		cfg.ClamAVPort = 3310
	}

	return &clamavScanner{
		host: cfg.ClamAVHost,
		port: cfg.ClamAVPort,
	}, nil
}

// Scan 扫描文件
func (s *clamavScanner) Scan(ctx context.Context, data []byte) (*ScanResult, error) {
	// 1. 连接ClamAV服务器
	conn, err := s.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to clamav failed: %w", err)
	}
	defer conn.Close()

	// 2. 发送INSTREAM命令
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return nil, fmt.Errorf("send command failed: %w", err)
	}

	// 3. 发送文件数据
	if err := s.sendData(conn, data); err != nil {
		return nil, fmt.Errorf("send data failed: %w", err)
	}

	// 4. 读取扫描结果
	result, err := s.readResult(conn)
	if err != nil {
		return nil, fmt.Errorf("read result failed: %w", err)
	}

	return result, nil
}

// ScanFile 扫描文件（通过路径）
func (s *clamavScanner) ScanFile(ctx context.Context, filePath string) (*ScanResult, error) {
	// 1. 连接ClamAV服务器
	conn, err := s.connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("connect to clamav failed: %w", err)
	}
	defer conn.Close()

	// 2. 发送SCAN命令
	command := fmt.Sprintf("zSCAN %s\x00", filePath)
	if _, err := conn.Write([]byte(command)); err != nil {
		return nil, fmt.Errorf("send command failed: %w", err)
	}

	// 3. 读取扫描结果
	result, err := s.readResult(conn)
	if err != nil {
		return nil, fmt.Errorf("read result failed: %w", err)
	}

	return result, nil
}

// GetProvider 获取扫描器提供商
func (s *clamavScanner) GetProvider() string {
	return "clamav"
}

// connect 连接到ClamAV服务器
func (s *clamavScanner) connect(ctx context.Context) (net.Conn, error) {
	address := fmt.Sprintf("%s:%d", s.host, s.port)

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}

	// 设置读写超时
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	return conn, nil
}

// sendData 发送数据到ClamAV
func (s *clamavScanner) sendData(conn net.Conn, data []byte) error {
	// ClamAV INSTREAM协议：
	// 1. 发送4字节的数据块大小（网络字节序）
	// 2. 发送数据块
	// 3. 重复直到所有数据发送完毕
	// 4. 发送4字节的0表示结束

	const chunkSize = 2048
	offset := 0

	for offset < len(data) {
		// 计算本次发送的数据大小
		size := chunkSize
		if offset+size > len(data) {
			size = len(data) - offset
		}

		// 发送数据块大小（大端字节序）
		sizeBytes := []byte{
			byte(size >> 24),
			byte(size >> 16),
			byte(size >> 8),
			byte(size),
		}
		if _, err := conn.Write(sizeBytes); err != nil {
			return err
		}

		// 发送数据块
		if _, err := conn.Write(data[offset : offset+size]); err != nil {
			return err
		}

		offset += size
	}

	// 发送结束标记（4字节的0）
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return err
	}

	return nil
}

// readResult 读取扫描结果
func (s *clamavScanner) readResult(conn net.Conn) (*ScanResult, error) {
	// 读取响应
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	response := string(buf[:n])
	response = strings.TrimSpace(response)

	// 解析响应
	// 格式: "stream: OK" 或 "stream: Eicar-Test-Signature FOUND"
	if strings.Contains(response, "OK") {
		return &ScanResult{
			Clean:     true,
			VirusName: "",
			Message:   "file is clean",
		}, nil
	}

	if strings.Contains(response, "FOUND") {
		// 提取病毒名称
		parts := strings.Split(response, ":")
		if len(parts) >= 2 {
			virusInfo := strings.TrimSpace(parts[1])
			virusName := strings.Replace(virusInfo, "FOUND", "", 1)
			virusName = strings.TrimSpace(virusName)

			return &ScanResult{
				Clean:     false,
				VirusName: virusName,
				Message:   fmt.Sprintf("virus detected: %s", virusName),
			}, nil
		}
	}

	// 其他情况视为扫描失败
	return nil, fmt.Errorf("unexpected response: %s", response)
}
