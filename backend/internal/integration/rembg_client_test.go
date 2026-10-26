package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRembgHealthRejectsWrongService(t *testing.T) {
	for _, body := range []string{`{"status":"ok","model":"birefnet-general"}`, `{"status":"ok","model":"isnet-general-use"}`, `{"status":"ok"}`, `not rembg`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		err := NewRembgClient(server.URL).Health(context.Background())
		server.Close()
		if strings.Contains(body, "birefnet-general") && err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(body, "birefnet-general") && !errors.Is(err, ErrRembgUnavailable) {
			t.Fatalf("wrong service accepted: %s", body)
		}
	}
}

func TestRembgConnectionFailureDoesNotExposeAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()
	_, err := NewRembgClient(server.URL).Remove(context.Background(), "image", "transparent")
	if !errors.Is(err, ErrRembgUnavailable) || strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatalf("unexpected user-facing error: %v", err)
	}
}

func TestRembgRemove(t *testing.T) {
	var output bytes.Buffer
	if err := png.Encode(&output, image.NewNRGBA(image.Rect(0, 0, 7, 5))); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if r.URL.Path != "/remove" || r.Method != "POST" || body["image_data"] != "original" || body["background"] != "white" {
			t.Errorf("unexpected request: %s %s %v", r.Method, r.URL.Path, body)
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(output.Bytes())
	}))
	defer server.Close()
	got, err := NewRembgClient(server.URL).Remove(context.Background(), "original", "white")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, output.Bytes()) {
		t.Fatal("PNG changed in transit")
	}
}

func TestRembgRejectsErrorsAndInvalidOutput(t *testing.T) {
	for _, status := range []int{http.StatusServiceUnavailable, http.StatusOK} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte("not an image"))
		}))
		_, err := NewRembgClient(server.URL).Remove(context.Background(), "input", "transparent")
		server.Close()
		if err == nil {
			t.Fatalf("expected error for status %d", status)
		}
	}
}

func TestRembgQueueHonorsCancellation(t *testing.T) {
	rembgSlot <- struct{}{}
	defer func() { <-rembgSlot }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewRembgClient("http://unused").Remove(ctx, "input", "white")
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("got %v", err)
	}
}
