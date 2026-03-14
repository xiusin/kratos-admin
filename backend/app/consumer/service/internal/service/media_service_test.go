package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	consumerV1 "go-wind-admin/api/gen/go/consumer/service/v1"
)

// MockMediaFileRepo Mock媒体文件仓库
type MockMediaFileRepo struct {
	mock.Mock
}

func (m *MockMediaFileRepo) Create(ctx context.Context, data *consumerV1.MediaFile) (*consumerV1.MediaFile, error) {
	args := m.Called(ctx, data)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*consumerV1.MediaFile), args.Error(1)
}

func (m *MockMediaFileRepo) Get(ctx context.Context, id uint64) (*consumerV1.MediaFile, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*consumerV1.MediaFile), args.Error(1)
}

func (m *MockMediaFileRepo) List(ctx context.Context, req interface{}) (*consumerV1.ListMediaFilesResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*consumerV1.ListMediaFilesResponse), args.Error(1)
}

func (m *MockMediaFileRepo) SoftDelete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// TestValidateFileFormat 测试文件格式验证
func TestValidateFileFormat(t *testing.T) {
	s := &MediaService{}

	tests := []struct {
		name      string
		fileType  consumerV1.MediaFile_FileType
		format    string
		wantError bool
	}{
		{
			name:      "valid image format - JPEG",
			fileType:  consumerV1.MediaFile_IMAGE,
			format:    "JPEG",
			wantError: false,
		},
		{
			name:      "valid image format - PNG",
			fileType:  consumerV1.MediaFile_IMAGE,
			format:    "PNG",
			wantError: false,
		},
		{
			name:      "valid image format - GIF",
			fileType:  consumerV1.MediaFile_IMAGE,
			format:    "GIF",
			wantError: false,
		},
		{
			name:      "invalid image format - BMP",
			fileType:  consumerV1.MediaFile_IMAGE,
			format:    "BMP",
			wantError: true,
		},
		{
			name:      "valid video format - MP4",
			fileType:  consumerV1.MediaFile_VIDEO,
			format:    "MP4",
			wantError: false,
		},
		{
			name:      "valid video format - AVI",
			fileType:  consumerV1.MediaFile_VIDEO,
			format:    "AVI",
			wantError: false,
		},
		{
			name:      "valid video format - MOV",
			fileType:  consumerV1.MediaFile_VIDEO,
			format:    "MOV",
			wantError: false,
		},
		{
			name:      "invalid video format - MKV",
			fileType:  consumerV1.MediaFile_VIDEO,
			format:    "MKV",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.validateFileFormat(tt.fileType, tt.format)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateFileSize 测试文件大小验证
func TestValidateFileSize(t *testing.T) {
	s := &MediaService{}

	tests := []struct {
		name      string
		fileType  consumerV1.MediaFile_FileType
		size      uint64
		wantError bool
	}{
		{
			name:      "valid image size - 1MB",
			fileType:  consumerV1.MediaFile_IMAGE,
			size:      1 * 1024 * 1024,
			wantError: false,
		},
		{
			name:      "valid image size - 5MB",
			fileType:  consumerV1.MediaFile_IMAGE,
			size:      5 * 1024 * 1024,
			wantError: false,
		},
		{
			name:      "invalid image size - 6MB",
			fileType:  consumerV1.MediaFile_IMAGE,
			size:      6 * 1024 * 1024,
			wantError: true,
		},
		{
			name:      "valid video size - 50MB",
			fileType:  consumerV1.MediaFile_VIDEO,
			size:      50 * 1024 * 1024,
			wantError: false,
		},
		{
			name:      "valid video size - 100MB",
			fileType:  consumerV1.MediaFile_VIDEO,
			size:      100 * 1024 * 1024,
			wantError: false,
		},
		{
			name:      "invalid video size - 101MB",
			fileType:  consumerV1.MediaFile_VIDEO,
			size:      101 * 1024 * 1024,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.validateFileSize(tt.fileType, tt.size)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGenerateObjectKey 测试对象键生成
func TestGenerateObjectKey(t *testing.T) {
	s := &MediaService{}

	tests := []struct {
		name       string
		tenantID   uint32
		consumerID uint32
		fileType   consumerV1.MediaFile_FileType
		fileName   string
		wantPrefix string
	}{
		{
			name:       "image object key",
			tenantID:   1,
			consumerID: 100,
			fileType:   consumerV1.MediaFile_IMAGE,
			fileName:   "test.jpg",
			wantPrefix: "media/1/100/image/",
		},
		{
			name:       "video object key",
			tenantID:   2,
			consumerID: 200,
			fileType:   consumerV1.MediaFile_VIDEO,
			fileName:   "test.mp4",
			wantPrefix: "media/2/200/video/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objectKey := s.generateObjectKey(tt.tenantID, tt.consumerID, tt.fileType, tt.fileName)
			assert.Contains(t, objectKey, tt.wantPrefix)
			assert.Contains(t, objectKey, tt.fileName)
		})
	}
}

// TestDetectFileType 测试文件类型检测
func TestDetectFileType(t *testing.T) {
	s := &MediaService{}

	tests := []struct {
		name     string
		format   string
		wantType consumerV1.MediaFile_FileType
	}{
		{
			name:     "detect JPEG as image",
			format:   "JPEG",
			wantType: consumerV1.MediaFile_IMAGE,
		},
		{
			name:     "detect PNG as image",
			format:   "PNG",
			wantType: consumerV1.MediaFile_IMAGE,
		},
		{
			name:     "detect GIF as image",
			format:   "GIF",
			wantType: consumerV1.MediaFile_IMAGE,
		},
		{
			name:     "detect MP4 as video",
			format:   "MP4",
			wantType: consumerV1.MediaFile_VIDEO,
		},
		{
			name:     "detect AVI as video",
			format:   "AVI",
			wantType: consumerV1.MediaFile_VIDEO,
		},
		{
			name:     "detect MOV as video",
			format:   "MOV",
			wantType: consumerV1.MediaFile_VIDEO,
		},
		{
			name:     "detect unknown format",
			format:   "UNKNOWN",
			wantType: consumerV1.MediaFile_FILE_TYPE_UNSPECIFIED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileType := s.detectFileType(tt.format)
			assert.Equal(t, tt.wantType, fileType)
		})
	}
}
