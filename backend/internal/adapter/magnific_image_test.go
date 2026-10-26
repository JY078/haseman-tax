package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestMagnificImageAdapterGenerateAndPoll(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if r.Header.Get("x-magnific-api-key") != "test-key" {
			t.Fatalf("unexpected API key header: %q", r.Header.Get("x-magnific-api-key"))
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/ai/text-to-image/nano-banana-pro-flash":
			var request MagnificImageRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if request.Prompt != "draw it" || request.AspectRatio != "1:1" || request.Resolution != "1K" {
				t.Fatalf("unexpected request: %+v", request)
			}
			if request.UseGoogleSearchTool == nil || *request.UseGoogleSearchTool {
				t.Fatalf("flash request must explicitly disable Google Search: %+v", request.UseGoogleSearchTool)
			}
			if len(request.ReferenceImages) != 1 || request.ReferenceImages[0].Image != "https://example.com/ref.jpg" {
				t.Fatalf("unexpected references: %+v", request.ReferenceImages)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"generated":[],"task_id":"task-1","status":"IN_PROGRESS"}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/ai/text-to-image/nano-banana-pro-flash/task-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"generated":["https://example.com/result.jpg"],"task_id":"task-1","status":"COMPLETED"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	adapter := NewMagnificImageAdapter("test-key", server.URL)
	adapter.pollInterval = time.Millisecond
	result, err := adapter.GenerateImageWithRefs(context.Background(), "draw it", []RefImage{
		{URL: "https://example.com/ref.jpg", MimeType: "image/jpeg"},
	}, "1:1", "1K", magnificNanoBananaProFlash)
	if err != nil {
		t.Fatalf("GenerateImageWithRefs returned error: %v", err)
	}
	if result.ImageURL != "https://example.com/result.jpg" {
		t.Fatalf("unexpected image URL: %q", result.ImageURL)
	}
	if requestCount.Load() != 2 {
		t.Fatalf("unexpected request count: %d", requestCount.Load())
	}
}

func TestMagnificImageAdapterProOmitsFlashOption(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if _, exists := raw["use_google_search_tool"]; exists {
			t.Fatalf("nano-banana-pro request must omit use_google_search_tool: %+v", raw)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"generated":["https://example.com/result.png"],"status":"COMPLETED"}}`))
	}))
	defer server.Close()

	result, err := NewMagnificImageAdapter("test-key", server.URL).GenerateImage(
		context.Background(), "draw it", "16:9", "2K", magnificNanoBananaPro,
	)
	if err != nil {
		t.Fatalf("GenerateImage returned error: %v", err)
	}
	if result.ImageURL != "https://example.com/result.png" {
		t.Fatalf("unexpected image URL: %q", result.ImageURL)
	}
}

func TestMagnificImageEndpointUsesConfiguredBaseURLAndModel(t *testing.T) {
	tests := []struct {
		baseURL string
		model   string
		want    string
	}{
		{"https://api.magnific.com", magnificNanoBananaPro, "https://api.magnific.com/v1/ai/text-to-image/nano-banana-pro"},
		{"https://api.magnific.com/v1/ai/text-to-image", magnificNanoBananaProFlash, "https://api.magnific.com/v1/ai/text-to-image/nano-banana-pro-flash"},
		{"https://api.magnific.com/v1/ai/text-to-image/nano-banana-pro", magnificNanoBananaProFlash, "https://api.magnific.com/v1/ai/text-to-image/nano-banana-pro-flash"},
	}
	for _, test := range tests {
		adapter := NewMagnificImageAdapter("test-key", test.baseURL)
		if got := adapter.endpointURL(test.model); got != test.want {
			t.Fatalf("endpointURL(%q, %q) = %q, want %q", test.baseURL, test.model, got, test.want)
		}
	}
}

func TestMagnificImageProviderDetection(t *testing.T) {
	if !IsMagnificURL("https://api.magnific.com/v1/ai/text-to-image") {
		t.Fatal("expected Magnific URL to be detected")
	}
	if IsMagnificURL("https://example.com/?target=api.magnific.com") {
		t.Fatal("must not detect Magnific from a query string")
	}
	for _, model := range []string{"nano-banana-pro", "NANO-BANANA-PRO-FLASH"} {
		if !IsMagnificImageModel(model) {
			t.Fatalf("expected model %q to be supported", model)
		}
	}
	if IsMagnificImageModel(strings.TrimSpace("gemini-3-pro-image")) {
		t.Fatal("unexpected Magnific model match")
	}
}

func TestMagnificReferenceImagesKeepsFirstFourteenValidReferences(t *testing.T) {
	refs := []RefImage{{}}
	for i := 1; i <= 15; i++ {
		refs = append(refs, RefImage{URL: fmt.Sprintf("https://example.com/%d.jpg", i), MimeType: "image/jpeg"})
	}

	images, tempPaths, err := magnificReferenceImages(refs)
	if err != nil {
		t.Fatalf("magnificReferenceImages returned error: %v", err)
	}
	if len(tempPaths) != 0 {
		t.Fatalf("unexpected temporary paths: %+v", tempPaths)
	}
	if len(images) != magnificMaxReferenceImages {
		t.Fatalf("got %d references, want %d", len(images), magnificMaxReferenceImages)
	}
	for i, image := range images {
		want := fmt.Sprintf("https://example.com/%d.jpg", i+1)
		if image.Image != want {
			t.Fatalf("reference %d = %q, want %q", i, image.Image, want)
		}
	}
}
