package handler

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"mime/multipart"
	"net/http/httptest"
	"testing"
)

func TestReviewVideoTypeAllowed(t *testing.T) {
	for _, test := range []struct {
		name, mime string
		allowed    bool
	}{
		{"clip.mp4", "video/mp4", true}, {"clip.MOV", "video/quicktime", true}, {"clip.webm", "video/webm", true},
		{"clip.mp4", "text/html", false}, {"clip.exe", "video/mp4", false}, {"clip.mp4", "image/png", false},
	} {
		if reviewVideoTypeAllowed(test.name, test.mime) != test.allowed {
			t.Fatalf("unexpected validation for %s / %s", test.name, test.mime)
		}
	}
}

func TestUploadBuyerReviewVideoRejectsInvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct{ name, id, filename, content string }{
		{"invalid ID", "bad", "clip.mp4", "fake"}, {"missing file", "1", "", ""}, {"empty file", "1", "clip.mp4", ""}, {"disguised file", "1", "clip.mp4", "<html>not video</html>"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			if test.filename != "" {
				file, err := writer.CreateFormFile("video", test.filename)
				if err != nil {
					t.Fatal(err)
				}
				file.Write([]byte(test.content))
			}
			writer.Close()
			req := httptest.NewRequest("POST", "/buyer-reviews/"+test.id+"/video", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			res := httptest.NewRecorder()
			router := gin.New()
			router.POST("/buyer-reviews/:id/video", UploadBuyerReviewVideo)
			router.ServeHTTP(res, req)
			if res.Code != 400 {
				t.Fatalf("status %d: %s", res.Code, res.Body.String())
			}
		})
	}
}
