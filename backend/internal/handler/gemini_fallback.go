package handler

import (
	"context"
	"fmt"
	"log"
	"strings"

	"advertising-image/backend/internal/adapter"
)

func geminiImagePreviewFallbackModel(modelName string) string {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "gemini-3.1-flash-image":
		return "gemini-3.1-flash-image-preview"
	case "gemini-3-pro-image":
		return "gemini-3-pro-image-preview"
	default:
		return ""
	}
}

func generateGeminiImageWithModelFallback(
	ctx context.Context,
	taskID string,
	geminiAdapter *adapter.GeminiAdapter,
	prompt string,
	refs []adapter.RefImage,
	aspectRatio string,
	resolution string,
	modelName string,
) (*adapter.GeminiImageResult, string, error) {
	call := func(name string) (*adapter.GeminiImageResult, error) {
		if len(refs) > 0 {
			return geminiAdapter.GenerateImageWithRefs(ctx, prompt, refs, aspectRatio, resolution, name)
		}
		return geminiAdapter.GenerateImage(ctx, prompt, aspectRatio, resolution, name)
	}

	result, err := call(modelName)
	if err == nil {
		return result, modelName, nil
	}

	fallbackModel := geminiImagePreviewFallbackModel(modelName)
	if fallbackModel == "" {
		return nil, modelName, err
	}

	log.Printf("[GeminiFallback] task=%s model=%s failed after retries, fallback to %s: %v", taskID, modelName, fallbackModel, err)
	result, fallbackErr := call(fallbackModel)
	if fallbackErr != nil {
		return nil, fallbackModel, fmt.Errorf("%s failed: %w; fallback %s failed: %v", modelName, err, fallbackModel, fallbackErr)
	}
	return result, fallbackModel, nil
}
