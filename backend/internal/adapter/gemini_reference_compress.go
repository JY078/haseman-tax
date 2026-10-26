package adapter

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	geminiReferenceMaxBytes      = 19_000_000
	geminiReferenceDownloadLimit = 100_000_000
)

// Preserve small originals; compress only the copy sent for generation.
func compressGeminiReference(ctx context.Context, data []byte, mimeType string, limit int) ([]byte, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if len(data) <= limit {
		return data, mimeType, nil
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("无法识别超限参考图: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 100_000_000 {
		return nil, "", fmt.Errorf("参考图像素过大，无法自动压缩")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("解码参考图失败: %w", err)
	}
	opaque := false
	if source, ok := img.(interface{ Opaque() bool }); ok {
		opaque = source.Opaque()
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}
		var buf bytes.Buffer
		if opaque {
			for _, quality := range []int{95, 85, 75} {
				buf.Reset()
				if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
					return nil, "", err
				}
				if buf.Len() <= limit {
					return buf.Bytes(), "image/jpeg", nil
				}
			}
		} else {
			encoder := png.Encoder{CompressionLevel: png.BestCompression}
			if err := encoder.Encode(&buf, img); err != nil {
				return nil, "", err
			}
			if buf.Len() <= limit {
				return buf.Bytes(), "image/png", nil
			}
		}
		bounds := img.Bounds()
		if bounds.Dx() == 1 && bounds.Dy() == 1 {
			return nil, "", fmt.Errorf("参考图无法压缩到大小限制以内")
		}
		resized := image.NewNRGBA(image.Rect(0, 0, max(1, bounds.Dx()*4/5), max(1, bounds.Dy()*4/5)))
		draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Src, nil)
		img = resized
	}
}
