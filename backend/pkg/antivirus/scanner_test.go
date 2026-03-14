package antivirus

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockScanner_Scan(t *testing.T) {
	scanner := NewMockScanner()
	ctx := context.Background()
	
	// 测试数据
	testData := []byte("test file content")
	
	// 扫描文件
	result, err := scanner.Scan(ctx, testData)
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Mock扫描器总是返回干净
	assert.True(t, result.Clean)
	assert.Empty(t, result.VirusName)
	assert.NotEmpty(t, result.Message)
}

func TestMockScanner_ScanFile(t *testing.T) {
	scanner := NewMockScanner()
	ctx := context.Background()
	
	// 扫描文件路径
	result, err := scanner.ScanFile(ctx, "/path/to/test/file.txt")
	require.NoError(t, err)
	assert.NotNil(t, result)
	
	// Mock扫描器总是返回干净
	assert.True(t, result.Clean)
	assert.Empty(t, result.VirusName)
	assert.Contains(t, result.Message, "file")
}

func TestMockScanner_GetProvider(t *testing.T) {
	scanner := NewMockScanner()
	
	provider := scanner.GetProvider()
	assert.Equal(t, "mock", provider)
}

func TestNewScanner(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{
			name:      "nil config",
			config:    nil,
			wantError: true,
		},
		{
			name: "mock provider",
			config: &Config{
				Provider: "mock",
			},
			wantError: false,
		},
		{
			name: "unsupported provider",
			config: &Config{
				Provider: "unknown",
			},
			wantError: true,
		},
		{
			name: "clamav provider (not implemented)",
			config: &Config{
				Provider: "clamav",
			},
			wantError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, err := NewScanner(tt.config)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, scanner)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, scanner)
			}
		})
	}
}
