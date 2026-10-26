package handler

import (
	"advertising-image/backend/internal/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWhiteBgTaskPreservesBackgroundCompatibility(t *testing.T) {
	for _, item := range []struct{ prompt, want string }{
		{"rembg transparent background: product", "transparent"},
		{"rembg white background: product", "white"},
		{"old generated product", "white"},
	} {
		if got := mapWhiteBgTask(model.WhiteBgTask{Prompt: item.prompt}).Background; got != item.want {
			t.Fatalf("got %s, want %s", got, item.want)
		}
	}
}

func TestWhiteBgAdmissionLimitAndInvalidRequestRelease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, handler := range []gin.HandlerFunc{GenerateWhiteBgImage, CreateWhiteBgTask} {
		router := gin.New()
		router.POST("/test", func(c *gin.Context) { c.Set("userID", uint(1)); handler(c) })
		for i := 0; i < cap(rembgTasks); i++ {
			rembgTasks <- struct{}{}
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("{}")))
		for i := 0; i < cap(rembgTasks); i++ {
			<-rembgTasks
		}
		if response.Code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want 429", response.Code)
		}
		response = httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("{}"))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("got %d, want 400", response.Code)
		}
		if len(rembgTasks) != 0 {
			t.Fatal("invalid request leaked a queue slot")
		}
	}
}
