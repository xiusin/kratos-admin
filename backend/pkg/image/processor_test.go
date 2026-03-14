package image

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestImage 创建测试图片
func createTestImage(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 填充颜色
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	// 编码为JPEG
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})

	return buf.Bytes()
}

func TestGenerateThumbnail(t *testing.T) {
	processor := NewProcessor()

	// 创建测试图片 (800x600)
	imageData := createTestImage(800, 600)

	// 生成缩略图 (200x200)
	thumbnail, err := processor.GenerateThumbnail(imageData, 200, 200)
	require.NoError(t, err)
	assert.NotEmpty(t, thumbnail)

	// 验证缩略图尺寸
	img, _, err := image.Decode(bytes.NewReader(thumbnail))
	require.NoError(t, err)

	bounds := img.Bounds()
	assert.Equal(t, 200, bounds.Dx())
	assert.Equal(t, 200, bounds.Dy())
}

func TestResize(t *testing.T) {
	processor := NewProcessor()

	// 创建测试图片 (800x600)
	imageData := createTestImage(800, 600)

	// 调整大小 (400x300)
	resized, err := processor.Resize(imageData, 400, 300)
	require.NoError(t, err)
	assert.NotEmpty(t, resized)

	// 验证调整后的尺寸
	img, _, err := image.Decode(bytes.NewReader(resized))
	require.NoError(t, err)

	bounds := img.Bounds()
	// 由于保持宽高比，实际尺寸可能小于等于目标尺寸
	assert.LessOrEqual(t, bounds.Dx(), 400)
	assert.LessOrEqual(t, bounds.Dy(), 300)
}

func TestCompress(t *testing.T) {
	processor := NewProcessor()

	// 创建测试图片
	imageData := createTestImage(800, 600)
	originalSize := len(imageData)

	// 压缩图片 (质量50)
	compressed, err := processor.Compress(imageData, 50)
	require.NoError(t, err)
	assert.NotEmpty(t, compressed)

	// 验证压缩后的大小小于原始大小
	compressedSize := len(compressed)
	assert.Less(t, compressedSize, originalSize)
}

func TestDetectFormat(t *testing.T) {
	processor := NewProcessor()

	// 创建JPEG图片
	imageData := createTestImage(100, 100)

	// 检测格式
	format, err := processor.DetectFormat(imageData)
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)
}
