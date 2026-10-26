package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"advertising-image/backend/internal/storage"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func TestGeminiReferencesUseOSSWithoutFilesAPI(t *testing.T) {
	for _, mode := range []string{"multiple", "single", "edit"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.URL.Path, "/ref") {
					fmt.Fprint(w, "hello")
					return
				}
				calls++
				if r.URL.Path != "/v1beta/models/gemini-test:generateContent" {
					t.Errorf("unexpected request (Files API must not be used): %s", r.URL.Path)
				}
				body, _ := io.ReadAll(r.Body)
				if strings.Contains(string(body), "inlineData") || len(body) > 2048 {
					t.Error("generation request must only carry reference URLs")
				}
				var request GeminiImageRequest
				if err := json.Unmarshal(body, &request); err != nil || len(request.Contents) != 1 {
					t.Error("invalid generation request")
					return
				}
				parts := request.Contents[0].Parts
				count := 1
				if mode == "multiple" {
					count = 2
				}
				if len(parts) != count+1 {
					t.Errorf("unexpected reference count: %d", len(parts))
					return
				}
				for i := 0; i < count; i++ {
					if parts[i].FileData == nil || parts[i].FileData.FileURI != fmt.Sprintf(server.URL+"/ref%d.png?signature=keep", i+1) || parts[i].FileData.MimeType != "image/png" {
						t.Errorf("incorrect OSS reference: %+v", parts[i])
					}
				}
				if parts[count].Text != "prompt" {
					t.Error("prompt must follow references")
				}
				fmt.Fprint(w, `{"candidates":[{"content":{"parts":[{"inlineData":{"data":"result","mimeType":"image/png"}}]}}]}`)
			}))
			defer server.Close()
			a := NewGeminiAdapter("test-key", server.URL)
			imageURL := server.URL + "/ref1.png?signature=keep"
			var result *GeminiImageResult
			var err error
			switch mode {
			case "multiple":
				result, err = a.GenerateImageWithRefs(context.Background(), "prompt", []RefImage{
					{URL: imageURL, Data: strings.Repeat("A", 21*1024*1024)},
					{URL: server.URL + "/ref2.png?signature=keep", MimeType: "image/png"},
				}, "1:1", "1K", "gemini-test")
			case "single":
				result, err = a.GenerateImageWithRef(context.Background(), "prompt", imageURL, "image/png", "1:1", "1K", "gemini-test")
			case "edit":
				result, err = a.EditImage(context.Background(), "prompt", imageURL, "image/png", "gemini-test")
			}
			if err != nil || result == nil || result.ImageData != "result" || calls != 1 {
				t.Fatalf("result=%+v error=%v calls=%d", result, err, calls)
			}
		})
	}
}

func TestGeminiBase64ReferenceIsSavedToOSS(t *testing.T) {
	var uploaded string
	original := noisyReference(t, false)
	// PNG decoders ignore trailing bytes, allowing a realistic oversized metadata case.
	oversized := append(append([]byte(nil), original...), make([]byte, geminiReferenceMaxBytes)...)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/original.png" {
			w.Write(oversized)
			return
		}
		if r.Method != http.MethodPut || !strings.Contains(r.URL.Path, "/temp_refs/") {
			t.Errorf("expected OSS PUT, got %s %s", r.Method, r.URL.Path)
		}
		data, _ := io.ReadAll(r.Body)
		uploaded = string(data)
		w.Header().Set("ETag", `"test"`)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	client, err := oss.New(server.URL, "test-key", "test-secret", oss.UseCname(true))
	if err != nil {
		t.Fatal(err)
	}
	bucket, err := client.Bucket("test-bucket")
	if err != nil {
		t.Fatal(err)
	}
	oldBucket, oldDomain := storage.OSSBucket, storage.OSSDomain
	storage.OSSBucket, storage.OSSDomain = bucket, "https://oss.example"
	t.Cleanup(func() { storage.OSSBucket, storage.OSSDomain = oldBucket, oldDomain })
	file, err := geminiReferenceFile(context.Background(), RefImage{Data: "data:image/png;base64,aGVsbG8="})
	if err != nil || uploaded != "hello" || file == nil || !strings.HasPrefix(file.FileURI, "https://oss.example/") || file.MimeType != "image/png" {
		t.Fatalf("file=%+v uploaded=%q err=%v", file, uploaded, err)
	}
	file, err = geminiReferenceFile(context.Background(), RefImage{URL: server.URL + "/original.png", MimeType: "image/png"})
	if err != nil {
		t.Fatal(err)
	}
	if len(uploaded) > geminiReferenceMaxBytes || file.MimeType != "image/jpeg" || !strings.HasPrefix(file.FileURI, "https://oss.example/") {
		t.Fatalf("invalid compressed upload: bytes=%d file=%+v", len(uploaded), file)
	}
	decoded, format, err := image.Decode(strings.NewReader(uploaded))
	if err != nil || format != "jpeg" || decoded.Bounds().Dx() != 256 {
		t.Fatalf("invalid compressed image: format=%s err=%v", format, err)
	}
	if !bytes.Equal(oversized[:len(original)], original) {
		t.Fatal("original modified")
	}
}
