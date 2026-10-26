package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGPTImageNativeSize(t *testing.T) {
	for _, size := range [][2]int{{2928, 1200}, {1200, 2928}, {3840, 2160}, {1024, 640}} {
		if GPTImageNativeSize("openai/gpt-image-2", size[0], size[1]) == "" {
			t.Fatalf("rejected %v", size)
		}
	}
	for _, size := range [][2]int{{2929, 1200}, {4096, 4096}, {64, 64}, {3072, 3072}, {2048, 512}} {
		if GPTImageNativeSize("gpt-image-2", size[0], size[1]) != "" {
			t.Fatalf("accepted %v", size)
		}
	}
	if GPTImageNativeSize("gpt-image-1", 2928, 1200) != "" {
		t.Fatal("legacy model accepted arbitrary size")
	}
	if GPTImageNativeSize("gpt-image-1.5", 1536, 1024) == "" {
		t.Fatal("legacy supported size rejected")
	}
}

func TestGPTCustomSizeSentForGenerationAndEditing(t *testing.T) {
	var sizes []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/images/generations" {
			var req GPTImageGenerateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Error(err)
			}
			sizes = append(sizes, req.Size)
		} else {
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
			}
			sizes = append(sizes, r.FormValue("size"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"url":"https://example.com/image.png"}]}`))
	}))
	defer server.Close()
	client := NewGPTImageAdapter("test", server.URL)
	client.OutputWidth, client.OutputHeight = 2928, 1200
	if _, err := client.GenerateImages(context.Background(), "test", "gpt-image-2", "1:1", "1K", 1); err != nil {
		t.Fatal(err)
	}
	if _, err := client.EditImages(context.Background(), "test", []RefImage{{Data: "aW1hZ2U=", MimeType: "image/png"}}, "gpt-image-2", "1:1", "1K", 1); err != nil {
		t.Fatal(err)
	}
	if len(sizes) != 2 || sizes[0] != "2928x1200" || sizes[1] != "2928x1200" {
		t.Fatalf("wrong sizes: %v", sizes)
	}
	client.OutputWidth = 2929
	if _, err := client.GenerateImages(context.Background(), "test", "gpt-image-2", "1:1", "4K", 1); err != nil {
		t.Fatal(err)
	}
	if sizes[2] != "2928x1200" {
		t.Fatalf("fallback ignored: %v", sizes)
	}
}

func TestMatchGPTImageSize(t *testing.T) {
	for _, target := range [][2]int{{2928, 1200}, {2929, 1200}, {4000, 1000}, {8192, 64}, {64, 64}, {4096, 4096}, {300, 200}, {5000, 4000}} {
		w, h := MatchGPTImageSize("gpt-image-2", target[0], target[1])
		if GPTImageNativeSize("gpt-image-2", w, h) == "" {
			t.Fatalf("illegal fallback %dx%d for %v", w, h, target)
		}
		pw, ph := MatchGPTImageSize("gpt-image-2", target[1], target[0])
		if pw != h || ph != w {
			t.Fatalf("asymmetric match for %v", target)
		}
	}
	w, h := MatchGPTImageSize("gpt-image-2", 2929, 1200)
	if w != 2928 || h != 1200 {
		t.Fatalf("not closest: %dx%d", w, h)
	}
	if w, h := MatchGPTImageSize("gemini-3-pro-image", 2928, 1200); w != 0 || h != 0 {
		t.Fatal("unsupported model matched")
	}
}
