package adapter

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"

	"advertising-image/backend/internal/storage"
)

// Reuse existing OSS URLs. References created by image editing are saved to OSS first.
func geminiReferenceFile(ctx context.Context, ref RefImage) (*GeminiFileData, error) {
	imageURL := strings.TrimSpace(ref.URL)
	data := strings.TrimSpace(ref.Data)
	mimeType := strings.TrimSpace(ref.MimeType)
	if imageURL == "" && (strings.HasPrefix(data, "https://") || strings.HasPrefix(data, "http://")) {
		imageURL = data
	}
	if mimeType == "" {
		if strings.HasPrefix(data, "data:") {
			header, _, _ := strings.Cut(data, ",")
			mimeType = strings.TrimSuffix(strings.TrimPrefix(header, "data:"), ";base64")
		} else if parsed, err := url.Parse(imageURL); err == nil {
			mimeType = mime.TypeByExtension(path.Ext(parsed.Path))
		}
		if mimeType == "" {
			mimeType = "image/png"
		}
	}
	var raw []byte
	if imageURL != "" {
		parsed, err := url.Parse(imageURL)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
			return nil, fmt.Errorf("参考图必须使用 HTTP(S) 地址")
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
		if err != nil {
			return nil, fmt.Errorf("创建参考图读取请求失败")
		}
		resp, err := imageHTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("读取参考图失败")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("读取参考图 HTTP %d", resp.StatusCode)
		}
		raw, err = io.ReadAll(io.LimitReader(resp.Body, geminiReferenceDownloadLimit+1))
		if err != nil {
			return nil, fmt.Errorf("读取参考图内容失败: %w", err)
		}
		if len(raw) > geminiReferenceDownloadLimit {
			return nil, fmt.Errorf("参考图超过 100MB，无法自动压缩")
		}
		if len(raw) <= geminiReferenceMaxBytes {
			return &GeminiFileData{FileURI: imageURL, MimeType: mimeType}, nil
		}
	} else {
		if data == "" {
			return nil, fmt.Errorf("参考图不能为空")
		}
		if _, encoded, ok := strings.Cut(data, ","); ok {
			data = encoded
		}
		var err error
		raw, err = base64.StdEncoding.DecodeString(data)
		if err != nil {
			raw, err = base64.RawStdEncoding.DecodeString(data)
		}
		if err != nil {
			return nil, fmt.Errorf("参考图 Base64 解码失败: %w", err)
		}
		if len(raw) > geminiReferenceDownloadLimit {
			return nil, fmt.Errorf("参考图超过 100MB，无法自动压缩")
		}
	}
	if !storage.IsOSSEnabled() {
		return nil, fmt.Errorf("参考图需要 OSS 地址，请先配置 OSS")
	}
	compressed, outputMime, err := compressGeminiReference(ctx, raw, mimeType, geminiReferenceMaxBytes)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	imageURL, err = storage.UploadImageReaderToOSS(bytes.NewReader(compressed), outputMime, "uploads/temp_refs")
	if err != nil {
		return nil, fmt.Errorf("保存参考图到 OSS 失败: %w", err)
	}
	mimeType = outputMime
	return &GeminiFileData{FileURI: imageURL, MimeType: mimeType}, nil
}
