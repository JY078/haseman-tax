package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenLuxGenerateImage(t *testing.T) {
	var got OpenLuxImageRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != openLuxImageGenerationPath {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"data":[{"url":"https://example.com/generated.png"}]}`))
	}))
	defer server.Close()

	result, err := NewOpenLuxImageAdapter("test-key", server.URL).GenerateImage(context.Background(), "prompt", "doubao-seedream-5-0-pro-260628", "1:1", "2K")
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if got.Model != "doubao-seedream-5-0-pro-260628" || got.Prompt != "prompt" || got.Size != "2048x2048" || got.ResponseFormat != "url" || got.Watermark {
		t.Fatalf("request = %+v", got)
	}
	if result.ImageURL == "" {
		t.Fatal("expected image URL")
	}
}

func TestOpenLuxGenerateImageWithRefsUsesOfficialImageField(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"data":[{"url":"https://example.com/generated.png"}]}`))
	}))
	defer server.Close()

	_, err := NewOpenLuxImageAdapter("test-key", server.URL).GenerateImageWithRefs(context.Background(), "prompt", openLuxSeedreamModel, "1:1", "1K", []RefImage{{URL: "https://example.com/ref.png"}, {URL: "https://example.com/ref-2.png"}})
	if err != nil {
		t.Fatalf("GenerateImageWithRefs: %v", err)
	}
	images, ok := got["image"].([]any)
	if !ok || len(images) != 2 || images[0] != "https://example.com/ref.png" {
		t.Fatalf("image field = %#v", got["image"])
	}
}

func TestOpenLuxGenerateImageWithOneRefUsesImageString(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"data":[{"url":"https://example.com/generated.png"}]}`))
	}))
	defer server.Close()

	_, err := NewOpenLuxImageAdapter("test-key", server.URL).GenerateImageWithRefs(context.Background(), "prompt", openLuxSeedreamModel, "1:1", "1K", []RefImage{{URL: "https://example.com/ref.png"}})
	if err != nil {
		t.Fatalf("GenerateImageWithRefs: %v", err)
	}
	if image, ok := got["image"].(string); !ok || image != "https://example.com/ref.png" {
		t.Fatalf("image field = %#v", got["image"])
	}
}

func TestOpenLuxGenerateImageWithRefsValidatesURLAndLimit(t *testing.T) {
	adapter := NewOpenLuxImageAdapter("test-key", "http://example.com")
	if _, err := adapter.GenerateImageWithRefs(context.Background(), "prompt", openLuxSeedreamModel, "1:1", "1K", []RefImage{{Data: "base64"}}); err == nil {
		t.Fatal("expected URL validation error")
	}
	refs := make([]RefImage, 11)
	for i := range refs {
		refs[i] = RefImage{URL: "https://example.com/ref.png"}
	}
	if _, err := adapter.GenerateImageWithRefs(context.Background(), "prompt", openLuxSeedreamModel, "1:1", "1K", refs); err == nil {
		t.Fatal("expected reference image limit error")
	}
}

func TestOpenLuxImageSizeMapping(t *testing.T) {
	for _, test := range []struct{ ratio, resolution, want string }{
		{"16:9", "1K", "1424x800"},
		{"1:1", "1K", "1024x1024"},
		{"1:1", "2K", "2048x2048"},
		{"4:3", "2K", "2368x1776"},
		{"21:9", "2K", "3136x1344"},
	} {
		got, err := mapOpenLuxImageSize(test.ratio, test.resolution)
		if err != nil || got != test.want {
			t.Fatalf("mapOpenLuxImageSize(%q, %q) = %q, %v; want %q", test.ratio, test.resolution, got, err, test.want)
		}
	}
	if _, err := mapOpenLuxImageSize("1:2", "1K"); err == nil {
		t.Fatal("expected ratio validation error")
	}
	if _, err := mapOpenLuxImageSize("1:1", "4K"); err == nil {
		t.Fatal("expected resolution validation error")
	}
}

func TestIsOpenLuxImageModelMatchesOnlyConfiguredSeedream(t *testing.T) {
	for _, name := range []string{openLuxSeedreamModel, "openlux/" + openLuxSeedreamModel} {
		if !IsOpenLuxImageModel(name) {
			t.Fatalf("IsOpenLuxImageModel(%q) = false", name)
		}
	}
	if IsOpenLuxImageModel("doubao-seedream-4-0") {
		t.Fatal("other Seedream models must not reuse the 5.0 mapping")
	}
}

func TestGeminiAdapterRoutesSeedreamAndRejectsRefs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"url":"https://example.com/generated.png"}]}`))
	}))
	defer server.Close()

	result, err := NewGeminiAdapter("test-key", server.URL).GenerateImage(context.Background(), "prompt", "1:1", "2K", "doubao-seedream-5-0-pro-260628")
	if err != nil || result.ImageURL == "" {
		t.Fatalf("GenerateImage = %#v, %v", result, err)
	}
	if _, err := NewGeminiAdapter("test-key", server.URL).GenerateImageWithRefs(context.Background(), "prompt", []RefImage{{Data: "aW1hZ2U="}}, "1:1", "2K", "doubao-seedream-5-0-pro-260628"); err == nil {
		t.Fatal("expected reference image error")
	}
}
