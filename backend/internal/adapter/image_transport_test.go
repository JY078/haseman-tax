package adapter

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageHTTPClientDisablesHTTP2(t *testing.T) {
	transport, ok := imageHTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("unexpected image transport type %T", imageHTTPClient.Transport)
	}
	if transport.ForceAttemptHTTP2 {
		t.Fatal("image transport must not force HTTP/2")
	}
	if transport.TLSNextProto == nil || len(transport.TLSNextProto) != 0 {
		t.Fatal("image transport must explicitly disable HTTP/2")
	}

	var protocol string
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		protocol = r.Proto
		w.WriteHeader(http.StatusNoContent)
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	testTransport := transport.Clone()
	testTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // test server certificate
	client := &http.Client{Transport: testTransport}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("request HTTP/2-capable test server: %v", err)
	}
	resp.Body.Close()
	if protocol != "HTTP/1.1" {
		t.Fatalf("protocol = %q, want HTTP/1.1", protocol)
	}
}

func TestGPTImageGenerateRequestsURLResponse(t *testing.T) {
	var responseFormat string
	var requestCount int
	var imageCounts []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var request GPTImageGenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		responseFormat = request.ResponseFormat
		imageCounts = append(imageCounts, request.N)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"url":"https://example.com/generated.png"}]}`))
	}))
	defer server.Close()

	client := NewGPTImageAdapter("test-key", server.URL)
	result, err := client.GenerateImages(context.Background(), "prompt", "gpt-image-2", "1:1", "1K", 3)
	if err != nil {
		t.Fatalf("GenerateImages: %v", err)
	}
	if requestCount != 3 {
		t.Fatalf("request count = %d, want 3", requestCount)
	}
	for index, imageCount := range imageCounts {
		if imageCount != 1 {
			t.Fatalf("request %d n = %d, want 1", index+1, imageCount)
		}
	}
	if len(result.Items) != 3 {
		t.Fatalf("result item count = %d, want 3", len(result.Items))
	}
	if responseFormat != "url" {
		t.Fatalf("response_format = %q, want url", responseFormat)
	}
}

func TestGPTImageEditUsesBufferedMultipart(t *testing.T) {
	var requestCount int
	var contentLength int64
	var transferEncoding []string
	var imageCounts []string
	var hasResponseFormat bool
	var hasMask bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		contentLength = r.ContentLength
		transferEncoding = append([]string(nil), r.TransferEncoding...)
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		imageCounts = append(imageCounts, r.FormValue("n"))
		_, hasResponseFormat = r.MultipartForm.Value["response_format"]
		_, hasMask = r.MultipartForm.Value["mask"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"url":"https://example.com/edited.png"}]}`))
	}))
	defer server.Close()

	client := NewGPTImageAdapter("test-key", server.URL)
	refs := []RefImage{{Data: "aW1hZ2U=", MimeType: "image/png"}}
	result, err := client.EditImages(context.Background(), "prompt", refs, "gpt-image-2", "1:1", "1K", 2)
	if err != nil {
		t.Fatalf("EditImages: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("request count = %d, want 2", requestCount)
	}
	if contentLength <= 0 {
		t.Fatalf("Content-Length = %d, want a buffered request body", contentLength)
	}
	if len(transferEncoding) != 0 {
		t.Fatalf("Transfer-Encoding = %v, want none", transferEncoding)
	}
	for index, imageCount := range imageCounts {
		if imageCount != "1" {
			t.Fatalf("request %d n = %q, want 1", index+1, imageCount)
		}
	}
	if len(result.Items) != 2 {
		t.Fatalf("result item count = %d, want 2", len(result.Items))
	}
	if hasResponseFormat {
		t.Fatal("response_format should not be sent")
	}
	if hasMask {
		t.Fatal("mask should not be sent")
	}
}

func TestApizGPTImageBatchCreatesOneTaskWithRequestedCount(t *testing.T) {
	requestCount := 0
	requestedImages := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var request ApizGPTImageRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		requestedImages = request.Params.NumImages
		_, _ = w.Write([]byte(`{"code":200,"data":{"task_id":"task-1"}}`))
	}))
	defer server.Close()

	client := NewApizGPTImageAdapter("test-key", server.URL)
	if _, err := client.createTask("openai/gpt-image-2", ApizGPTImageParams{Prompt: "prompt", NumImages: 4}); err != nil {
		t.Fatalf("createTask: %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("request count = %d, want 1", requestCount)
	}
	if requestedImages != 4 {
		t.Fatalf("n = %d, want 4", requestedImages)
	}
}

func TestIsGPTImageModelAcceptsProviderPrefix(t *testing.T) {
	for _, modelName := range []string{"gpt-image-2", "openai/gpt-image-2"} {
		if !IsGPTImageModel(modelName) {
			t.Fatalf("IsGPTImageModel(%q) = false, want true", modelName)
		}
	}
}

func TestGeminiRequestDetailsAreRedacted(t *testing.T) {
	const key = "sk-secret-key"
	url := "https://yunwu.ai/v1beta/models/test:generateContent?key=" + key
	if redacted := redactGeminiRequestURL(url); strings.Contains(redacted, key) || strings.Contains(redacted, "?") {
		t.Fatalf("request URL was not redacted: %s", redacted)
	}

	err := redactGeminiNetworkError(errors.New(`Post "`+url+`": connection reset by peer`), key)
	if strings.Contains(err.Error(), key) {
		t.Fatalf("network error contains API key: %s", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("network error does not contain redaction marker: %s", err)
	}
}
