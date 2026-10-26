package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestImageViolationExampleValidation(t *testing.T) {
	valid := imageViolationExampleInput{Platform: "Amazon", ProblemType: "图片文字", Description: "图片中包含违规文字", ImageURL: "/uploads/prompts/example.png"}
	for _, platform := range []string{"Amazon", "TikTok", "Homedepot", "TEMU", "Walmart", "Lowes", "Wayfair"} {
		input := valid
		input.Platform = platform
		if err := input.validate(); err != nil {
			t.Fatalf("%s: %v", platform, err)
		}
	}
	for _, tt := range []struct {
		name   string
		change func(*imageViolationExampleInput)
	}{
		{"unknown platform", func(v *imageViolationExampleInput) { v.Platform = "Other" }},
		{"blank type", func(v *imageViolationExampleInput) { v.ProblemType = "  " }},
		{"long type", func(v *imageViolationExampleInput) { v.ProblemType = strings.Repeat("字", 101) }},
		{"blank description", func(v *imageViolationExampleInput) { v.Description = "\n " }},
		{"long description", func(v *imageViolationExampleInput) { v.Description = strings.Repeat("字", 5001) }},
		{"missing image", func(v *imageViolationExampleInput) { v.ImageURL = "" }},
		{"blob image", func(v *imageViolationExampleInput) { v.ImageURL = "blob:http://localhost/image" }},
		{"inline image", func(v *imageViolationExampleInput) { v.ImageURL = "data:image/png;base64,abc" }},
		{"temporary image", func(v *imageViolationExampleInput) { v.ImageURL = "/uploads/temp_refs/image.png" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := valid
			tt.change(&input)
			if err := input.validate(); err == nil {
				t.Fatal("invalid input was accepted")
			}
		})
	}
	valid.ImageURL = "https://bucket.oss-cn-hangzhou.aliyuncs.com/uploads/prompts/example.png"
	if err := valid.validate(); err != nil {
		t.Fatal(err)
	}
}

// Run against a local PostgreSQL database in a separate, rolled-back schema.
func TestImageViolationExamplePersistence(t *testing.T) {
	envFile := os.Getenv("IMAGE_VIOLATION_TEST_ENV_FILE")
	if envFile == "" {
		t.Skip("set IMAGE_VIOLATION_TEST_ENV_FILE for local PostgreSQL integration")
	}
	settings := viper.New()
	settings.SetConfigFile(envFile)
	if err := settings.ReadInConfig(); err != nil {
		t.Fatal("cannot read local database configuration")
	}
	host := settings.GetString("DB_HOST")
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Fatal("test requires local PostgreSQL")
	}
	dsn := url.URL{Scheme: "postgres", User: url.UserPassword(settings.GetString("DB_USER"), settings.GetString("DB_PASSWORD")), Host: host + ":" + settings.GetString("DB_PORT"), Path: "/" + settings.GetString("DB_NAME")}
	dsn.RawQuery = url.Values{"sslmode": {settings.GetString("DB_SSLMODE")}}.Encode()
	db, err := gorm.Open(postgres.Open(dsn.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot connect to local PostgreSQL")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	previous := database.DB
	database.DB = tx
	t.Cleanup(func() { tx.Rollback(); database.DB = previous })
	schema := fmt.Sprintf("image_violation_test_%d", time.Now().UnixNano())
	for _, sql := range []string{"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema, "CREATE TABLE users (id bigint PRIMARY KEY, name text NOT NULL)", "INSERT INTO users (id, name) VALUES (7, '提交人姓名'), (8, '其他用户')"} {
		if err := tx.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.AutoMigrate(&model.ImageViolationExample{}); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		switch c.GetHeader("X-Test-User") {
		case "7":
			c.Set("userID", uint(7))
		case "8":
			c.Set("userID", uint(8))
		default:
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	})
	router.GET("/examples", ListImageViolationExamples)
	router.POST("/examples", CreateImageViolationExample)
	send := func(actor, method, body string, status int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, "/examples", strings.NewReader(body))
		req.Header.Set("X-Test-User", actor)
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != status {
			t.Fatalf("status=%d want=%d body=%s", recorder.Code, status, recorder.Body.String())
		}
		return recorder.Body.Bytes()
	}
	send("", http.MethodGet, "", 401)
	send("", http.MethodPost, "{}", 401)
	var empty struct {
		Data []model.ImageViolationExample `json:"data"`
	}
	if err := json.Unmarshal(send("8", http.MethodGet, "", 200), &empty); err != nil {
		t.Fatal(err)
	}
	if empty.Data == nil || len(empty.Data) != 0 {
		t.Fatal("empty list must be []")
	}
	send("7", http.MethodPost, `{"platform":"Amazon","problem_type":"","image_url":"blob:local"}`, 400)
	var created model.ImageViolationExample
	start := time.Now().Add(-time.Second)
	if err := json.Unmarshal(send("7", http.MethodPost, `{"platform":"Amazon","problem_type":" 文字违规 ","description":" 含有不允许的文字 ","image_url":"/uploads/prompts/test.png","user_id":8,"submitter":"伪造姓名","created_at":"2000-01-01T00:00:00Z"}`, 201), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.UserID != 7 || created.Submitter != "提交人姓名" || created.CreatedAt.Before(start) || created.ProblemType != "文字违规" {
		t.Fatalf("incorrect server-managed fields: %+v", created)
	}
	var persisted model.ImageViolationExample
	if err := tx.First(&persisted, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.ImageURL != "/uploads/prompts/test.png" {
		t.Fatal("image URL was not persisted")
	}
	var otherUser struct {
		Data []model.ImageViolationExample `json:"data"`
	}
	if err := json.Unmarshal(send("8", http.MethodGet, "", 200), &otherUser); err != nil {
		t.Fatal(err)
	}
	if len(otherUser.Data) != 1 || otherUser.Data[0].ID != created.ID || otherUser.Data[0].Submitter != "提交人姓名" {
		t.Fatal("another user cannot retrieve saved example")
	}
}
