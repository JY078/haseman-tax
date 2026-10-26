package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestArkVideoCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/contents/generations/tasks" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization: %q", got)
		}
		var req ArkVideoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Model != "doubao-seedance-2-0-fast-260128" || len(req.Content) != 2 {
			t.Fatalf("unexpected payload: %#v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cgt-test","status":"queued"}`))
	}))
	defer server.Close()

	client := NewArkVideoAdapter("test-key", server.URL)
	client.Client = server.Client()
	result, err := client.Create(context.Background(), ArkVideoRequest{
		Model: "doubao-seedance-2-0-fast-260128",
		Content: []ArkVideoContent{
			{Type: "text", Text: "product rotates"},
			{Type: "image_url", ImageURL: &ArkVideoImageURL{URL: "data:image/png;base64,abc"}, Role: "first_frame"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "cgt-test" || result.Status != "queued" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestArkVideoQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/contents/generations/tasks/cgt-test" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"cgt-test","status":"succeeded","content":{"video_url":"https://example.com/video.mp4"}}`))
	}))
	defer server.Close()

	client := NewArkVideoAdapter("test-key", server.URL)
	client.Client = server.Client()
	result, err := client.Query(context.Background(), "cgt-test")
	if err != nil {
		t.Fatal(err)
	}
	if result.Content.VideoURL != "https://example.com/video.mp4" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
