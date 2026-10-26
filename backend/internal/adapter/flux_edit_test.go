package adapter

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFluxOpenAIImageEdit(t *testing.T) {
	for _, responseType := range []string{"url", "b64_json"} {
		t.Run(responseType, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/v1/images/edits" {
					t.Errorf("unexpected endpoint: %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer test-key" {
					t.Error("missing authorization")
				}
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				defer r.MultipartForm.RemoveAll()
				for key, want := range map[string]string{"model": "flux.1-kontext-pro", "prompt": "Harmonize the product", "size": "1024x1024", "n": "1"} {
					if got := r.FormValue(key); got != want {
						t.Errorf("%s = %q, want %q", key, got, want)
					}
				}
				if _, ok := r.MultipartForm.Value["quality"]; ok {
					t.Error("Flux must not receive GPT quality levels")
				}
				file, _, err := r.FormFile("image")
				if err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				defer file.Close()
				data, err := io.ReadAll(file)
				if err != nil || string(data) != "test-image" {
					t.Errorf("uploaded image = %q, error = %v", data, err)
				}
				w.Header().Set("Content-Type", "application/json")
				if responseType == "url" {
					io.WriteString(w, `{"data":[{"url":"https://example.com/result.png"}]}`)
				} else {
					io.WriteString(w, `{"data":[{"b64_json":"dGVzdC1pbWFnZQ=="}]}`)
				}
			}))
			defer server.Close()
			client := NewGPTImageAdapter("test-key", server.URL)
			result, err := client.EditImages(context.Background(), "Harmonize the product", []RefImage{{Data: base64.StdEncoding.EncodeToString([]byte("test-image")), MimeType: "image/png"}}, "flux.1-kontext-pro", "1:1", "1K", 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Items) != 1 {
				t.Fatalf("expected one image, got %d", len(result.Items))
			}
			if responseType == "url" && result.Items[0].ImageURL != "https://example.com/result.png" {
				t.Errorf("unexpected image URL: %q", result.Items[0].ImageURL)
			}
			if responseType == "b64_json" && result.Items[0].ImageData != "dGVzdC1pbWFnZQ==" {
				t.Errorf("unexpected image data: %q", result.Items[0].ImageData)
			}
		})
	}
}

func TestFluxEditServiceUnavailableRetry(t *testing.T) {
	for _, tc := range []struct {
		name     string
		model    string
		status   int
		failures int32
		attempts int32
		cancel   bool
	}{
		{"recovers", "flux.1-kontext-pro", 503, 1, 2, false},
		{"exhausted", "flux.1-kontext-pro", 503, 3, 3, false},
		{"bad_request", "flux.1-kontext-pro", 400, 1, 1, false},
		{"other_model", "gpt-image-2", 503, 1, 1, false},
		{"canceled", "flux.1-kontext-pro", 503, 1, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Error(err)
				} else {
					defer r.MultipartForm.RemoveAll()
					file, _, err := r.FormFile("image")
					if err != nil {
						t.Error("retry lost image:", err)
					} else {
						data, _ := io.ReadAll(file)
						file.Close()
						if string(data) != "test-image" {
							t.Errorf("unexpected image: %q", data)
						}
					}
				}
				if attempts.Add(1) <= tc.failures {
					w.WriteHeader(tc.status)
					io.WriteString(w, `{"error":{"message":"unavailable (request id: test-request)"}}`)
					if tc.cancel {
						cancel()
					}
					return
				}
				io.WriteString(w, `{"data":[{"url":"https://example.com/result.png"}]}`)
			}))
			defer server.Close()
			client := NewGPTImageAdapter("test-key", server.URL)
			result, err := client.EditImages(ctx, "prompt", []RefImage{{Data: "dGVzdC1pbWFnZQ==", MimeType: "image/png"}}, tc.model, "1:1", "1K", 1)
			if got := attempts.Load(); got != tc.attempts {
				t.Errorf("attempts = %d, want %d", got, tc.attempts)
			}
			if tc.cancel {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("expected cancellation, got %v", err)
				}
			} else if tc.failures >= tc.attempts {
				if err == nil || !strings.Contains(err.Error(), "test-request") {
					t.Errorf("expected original upstream error, got %v", err)
				}
			} else if err != nil || result == nil || len(result.Items) != 1 {
				t.Errorf("expected recovery, got result=%v error=%v", result, err)
			}
		})
	}
}
