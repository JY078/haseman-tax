package handler

import "testing"

func TestGeminiImagePreviewFallbackModel(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"gemini-3.1-flash-image", "gemini-3.1-flash-image-preview"},
		{"gemini-3-pro-image", "gemini-3-pro-image-preview"},
		{"gemini-3.1-flash-image-preview", ""},
		{"gemini-3-pro-image-preview", ""},
		{"gemini-3.0-flash-image", ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := geminiImagePreviewFallbackModel(tt.model); got != tt.want {
				t.Fatalf("geminiImagePreviewFallbackModel(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}
