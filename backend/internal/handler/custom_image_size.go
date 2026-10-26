package handler

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"advertising-image/backend/internal/storage"

	"golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
	_ "golang.org/x/image/webp"
)

func validateCustomImageSize(width, height int) error {
	if width == 0 && height == 0 {
		return nil
	}
	if width < 64 || height < 64 || width > 8192 || height > 8192 {
		return fmt.Errorf("自定义宽高必须是 64–8192 之间的整数")
	}
	if width*height > 24000000 {
		return fmt.Errorf("自定义尺寸总像素不能超过 2400 万")
	}
	return nil
}

// Apply a uniform scale directly onto the target canvas; overflowing edges are clipped.
func resizeCustomImage(data []byte, width, height int) ([]byte, error) {
	if err := validateCustomImageSize(width, height); err != nil {
		return nil, err
	}
	if width == 0 {
		return nil, fmt.Errorf("缺少目标尺寸")
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 64000000 {
		return nil, fmt.Errorf("原图尺寸超出处理范围")
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	bounds := src.Bounds()
	scale := math.Max(float64(width)/float64(bounds.Dx()), float64(height)/float64(bounds.Dy()))
	x := (float64(width)-float64(bounds.Dx())*scale)/2 - float64(bounds.Min.X)*scale
	y := (float64(height)-float64(bounds.Dy())*scale)/2 - float64(bounds.Min.Y)*scale
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Transform(dst, f64.Aff3{scale, 0, x, 0, scale, y}, src, bounds, draw.Src, nil)
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func resizeCustomImageResult(result imageGenerationAdapterResult, width, height int) (imageGenerationAdapterResult, error) {
	urls := result.ImageURLs
	if len(urls) == 0 {
		urls = []string{result.ImageURL}
	}
	resized := make([]string, 0, len(urls))
	for _, url := range urls {
		var data []byte
		var err error
		if strings.HasPrefix(url, "/uploads/") {
			path := filepath.ToSlash(filepath.Clean(strings.TrimPrefix(url, "/")))
			if !strings.HasPrefix(path, "uploads/") {
				return result, fmt.Errorf("无效图片路径")
			}
			data, err = os.ReadFile(path)
		} else {
			data, _, err = storage.DownloadImageToMemory(url)
		}
		if err != nil {
			return result, fmt.Errorf("读取待调整图片失败: %w", err)
		}
		config, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return result, fmt.Errorf("读取图片尺寸失败: %w", err)
		}
		if config.Width == width && config.Height == height {
			resized = append(resized, url)
			continue
		}
		data, err = resizeCustomImage(data, width, height)
		if err != nil {
			return result, fmt.Errorf("调整图片尺寸失败: %w", err)
		}
		url, err = storage.UploadImageDataToOSS(data, "image/png", storage.UploadDir)
		if err != nil {
			return result, err
		}
		resized = append(resized, url)
	}
	result.ImageURL, result.ImageURLs = resized[0], resized
	// A single generated image is the normal page workflow. Keep the original MIME for a passthrough.
	for i, url := range resized {
		if url != urls[i] {
			result.MimeType = "image/png"
			break
		}
	}
	return result, nil
}
