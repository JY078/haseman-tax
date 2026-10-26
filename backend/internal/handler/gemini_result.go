package handler

import (
	"fmt"
	"strings"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/storage"
)

func saveGeminiImageResult(result *adapter.GeminiImageResult) (string, string, error) {
	if result == nil {
		return "", "", fmt.Errorf("empty Gemini image result")
	}
	if strings.TrimSpace(result.ImageData) != "" {
		imageURL, err := storage.SaveBase64Image(result.ImageData, result.MimeType)
		if err != nil {
			return "", "", err
		}
		return imageURL, result.MimeType, nil
	}
	if strings.TrimSpace(result.ImageURL) != "" {
		imageURL, mimeType, err := ensureStoredImageURLForUsage(result.ImageURL)
		if err != nil {
			return "", "", err
		}
		if mimeType == "" {
			mimeType = strings.TrimSpace(result.MimeType)
		}
		if mimeType == "" {
			mimeType = "image/png"
		}
		return imageURL, mimeType, nil
	}
	return "", "", fmt.Errorf("Gemini image result is empty")
}
