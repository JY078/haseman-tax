package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAnalyzeImagesWithVisionSendsEveryImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body struct {
			Input []struct {
				Content []map[string]interface{} `json:"content"`
			} `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Input) != 1 || len(body.Input[0].Content) != 3 {
			t.Fatalf("expected two images and one prompt, got %#v", body.Input)
		}
		if body.Input[0].Content[0]["image_url"] != "data:image/png;base64,first" {
			t.Fatalf("unexpected first image: %#v", body.Input[0].Content[0])
		}
		if body.Input[0].Content[1]["image_url"] != "data:image/jpeg;base64,second" {
			t.Fatalf("unexpected second image: %#v", body.Input[0].Content[1])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"content":[{"type":"output_text","text":"ok"}]}]}`))
	}))
	defer server.Close()

	client := NewChatAdapter("key", server.URL)
	text, _, _, err := client.AnalyzeImagesWithVision("vision-model", []VisionImageItem{
		{Data: "first", MimeType: "image/png"},
		{Data: "second", MimeType: "image/jpeg"},
	}, "", "compare", 2000)
	if err != nil {
		t.Fatal(err)
	}
	if text != "ok" {
		t.Fatalf("unexpected response: %q", text)
	}
}

func TestAnalyzeImagesWithVisionContextCancelsRequest(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	client := NewChatAdapter("key", server.URL)
	_, _, _, err := client.AnalyzeImagesWithVisionContext(ctx, "vision-model", []VisionImageItem{{Data: "first", MimeType: "image/png"}}, "", "analyze", 2000)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected request cancellation, got %v", err)
	}
}

func TestAnalyzeImagesWithVisionChatCompletionsSendsEveryImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body struct {
			Messages []struct {
				Content []map[string]interface{} `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Messages) != 1 || len(body.Messages[0].Content) != 3 {
			t.Fatalf("expected one prompt and two images, got %#v", body.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer server.Close()

	client := NewChatAdapter("key", server.URL)
	text, _, _, err := client.AnalyzeImagesWithVisionChatCompletions("vision-model", []VisionImageItem{
		{Data: "first", MimeType: "image/png"},
		{Data: "second", MimeType: "image/jpeg"},
	}, "compare")
	if err != nil {
		t.Fatal(err)
	}
	if text != "ok" {
		t.Fatalf("unexpected response: %q", text)
	}
}
