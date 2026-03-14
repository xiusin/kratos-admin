package image

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/imaging"
)

// Processor 图片处理器接口
type Processor interface {
	// GenerateThumbnail 生成缩略图
	GenerateThumbnail(data []byte, width, height int) ([]byte, error)

	// Resize 调整图片大小
	Resize(data []byte, width, height int) ([]byte, error)

	// Compress 压缩图片
	Compress(data []byte, quality int) ([]byte, error)

	// DetectFormat 检测图片格式
	DetectFormat(data []byte) (string, error)
}

type processor struct{}

// NewProcessor 创建图片处理器
func NewProcessor() Processor {
	return &processor{}
}

// GenerateThumbnail 生成缩略图
func (p *processor) GenerateThumbnail(data []byte, width, height int) ([]byte, error) {
	// 1. 解码图片
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image failed: %w", err)
	}

	// 2. 生成缩略图（保持宽高比，填充背景）
	thumbnail := imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)

	// 3. 编码为JPEG格式（缩略图统一使用JPEG）
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return nil, fmt.Errorf("encode thumbnail failed: %w", err)
	}

	// 记录原始格式（用于日志）
	_ = format

	return buf.Bytes(), nil
}

// Resize 调整图片大小
func (p *processor) Resize(data []byte, width, height int) ([]byte, error) {
	// 1. 解码图片
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image failed: %w", err)
	}

	// 2. 调整大小（保持宽高比）
	resized := imaging.Fit(img, width, height, imaging.Lanczos)

	// 3. 编码为原格式
	var buf bytes.Buffer
	if err := p.encodeImage(&buf, resized, format); err != nil {
		return nil, fmt.Errorf("encode image failed: %w", err)
	}

	return buf.Bytes(), nil
}

// Compress 压缩图片
func (p *processor) Compress(data []byte, quality int) ([]byte, error) {
	// 1. 解码图片
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image failed: %w", err)
	}

	// 2. 编码为JPEG格式（压缩）
	var buf bytes.Buffer
	if format == "jpeg" || format == "jpg" {
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("encode jpeg failed: %w", err)
		}
	} else {
		// 其他格式转换为JPEG
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, fmt.Errorf("encode jpeg failed: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// DetectFormat 检测图片格式
func (p *processor) DetectFormat(data []byte) (string, error) {
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("detect format failed: %w", err)
	}
	return format, nil
}

// encodeImage 编码图片
func (p *processor) encodeImage(w io.Writer, img image.Image, format string) error {
	switch format {
	case "jpeg", "jpg":
		return jpeg.Encode(w, img, &jpeg.Options{Quality: 90})
	case "png":
		return png.Encode(w, img)
	case "gif":
		return gif.Encode(w, img, nil)
	default:
		// 默认使用JPEG
		return jpeg.Encode(w, img, &jpeg.Options{Quality: 90})
	}
}
