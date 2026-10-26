package handler

import (
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func TestCustomsPythonExecutable(t *testing.T) {
	t.Setenv("CUSTOMS_PYTHON", "")
	t.Setenv("STP_WORKER_PYTHON", "")
	viper.Reset()
	t.Cleanup(viper.Reset)
	if got := customsPythonExecutable(); got != "python3" {
		t.Fatalf("fallback: %q", got)
	}
	viper.SetConfigType("env")
	if err := viper.ReadConfig(strings.NewReader("STP_WORKER_PYTHON=/opt/miniconda3/envs/py311/bin/python\n")); err != nil {
		t.Fatal(err)
	}
	if got := customsPythonExecutable(); got != "/opt/miniconda3/envs/py311/bin/python" {
		t.Fatalf("dotenv STP: %q", got)
	}
	t.Setenv("STP_WORKER_PYTHON", "/env/stp/python")
	if got := customsPythonExecutable(); got != "/env/stp/python" {
		t.Fatalf("environment STP: %q", got)
	}
	viper.Set("CUSTOMS_PYTHON", "/custom/python")
	if got := customsPythonExecutable(); got != "/custom/python" {
		t.Fatalf("custom precedence: %q", got)
	}
	t.Setenv("CUSTOMS_PYTHON", "/env/custom/python")
	if got := customsPythonExecutable(); got != "/env/custom/python" {
		t.Fatalf("environment custom: %q", got)
	}
}

func TestCustomsPostgresIntegration(t *testing.T) {
	dsn, sample := os.Getenv("TAX_TEST_DSN"), os.Getenv("CUSTOMS_SAMPLE_PDF")
	if dsn == "" || sample == "" {
		t.Skip("set local TAX_TEST_DSN and CUSTOMS_SAMPLE_PDF")
	}
	if !strings.Contains(dsn, "host=127.0.0.1") {
		t.Fatal("integration database must be local")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("local database connection failed")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	oldDB := database.DB
	database.DB = tx
	defer func() { tx.Rollback(); database.DB = oldDB }()
	schema := fmt.Sprintf("customs_test_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.CustomsDeclaration{}, &model.User{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "customs-finance-test", Department: "财务部"}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(sample)
	if err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	if err := os.Chdir(filepath.Join(cwd, "..", "..")); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userID", user.ID) }, RequireTaxDataAccess())
	router.POST("/customs", UploadCustomsDeclaration)
	router.GET("/customs", ListCustomsDeclarations)
	router.GET("/customs/:id", GetCustomsDeclaration)
	router.GET("/customs/:id/file", DownloadCustomsDeclaration)
	var id uint
	for index, payload := range [][]byte{content, content, append(append([]byte{}, content...), []byte("\n% duplicate declaration\n")...)} {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		part, _ := mw.CreateFormFile("file", fmt.Sprintf("sample-%d.pdf", index))
		part.Write(payload)
		mw.Close()
		req := httptest.NewRequest("POST", "/customs", &body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		want := 200
		if index == 0 {
			want = 201
		}
		if response.Code != want {
			t.Fatalf("upload %d: %d %s", index, response.Code, response.Body.String())
		}
		var result struct {
			ID                uint   `json:"id"`
			Duplicate         bool   `json:"duplicate"`
			DeclarationNumber string `json:"declaration_number"`
			Message           string `json:"message"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			id = result.ID
		} else if !result.Duplicate || result.ID != id {
			t.Fatal("deduplication failed")
		} else if result.DeclarationNumber == "" || !strings.Contains(result.Message, result.DeclarationNumber) {
			t.Fatal("duplicate response must identify the declaration number")
		}
	}
	for _, query := range []string{"", "?keyword=FUS-26696", "?page=-1&page_size=999"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", "/customs"+query, nil))
		var result struct {
			Total int                        `json:"total"`
			Items []model.CustomsDeclaration `json:"items"`
		}
		json.Unmarshal(response.Body.Bytes(), &result)
		if response.Code != 200 || result.Total != 1 || len(result.Items) != 1 || result.Items[0].Fields["contract_number"] != "FUS-26696" {
			t.Fatalf("list: %d %s", response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", fmt.Sprintf("/customs/%d/file", id), nil))
	if response.Code != 200 || !bytes.Equal(response.Body.Bytes(), content) {
		t.Fatal("original PDF roundtrip failed")
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", fmt.Sprintf("/customs/%d", id), nil))
	var detail model.CustomsDeclaration
	json.Unmarshal(response.Body.Bytes(), &detail)
	if response.Code != 200 || detail.RawText == "" || detail.Items[0]["declared_quantity"] != "120套" {
		t.Fatal("detail persistence failed")
	}
	if err := tx.Model(&user).Update("department", "运营").Error; err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/customs", nil))
	if response.Code != 403 {
		t.Fatalf("non-finance access: %d", response.Code)
	}
}

func TestCustomsRejectInvalidUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct{ name, content string }{{"a.txt", "%PDF-1.4"}, {"a.pdf", "not a PDF"}} {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, _ := writer.CreateFormFile("file", test.name)
		part.Write([]byte(test.content))
		writer.Close()
		router := gin.New()
		router.POST("/upload", UploadCustomsDeclaration)
		req := httptest.NewRequest("POST", "/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != 400 {
			t.Fatalf("%s: status %d: %s", test.name, response.Code, response.Body.String())
		}
	}
}

func TestCustomsAccessRequiresLogin(t *testing.T) {
	router := gin.New()
	router.GET("/customs", RequireTaxDataAccess(), ListCustomsDeclarations)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/customs", nil))
	if response.Code != 401 {
		t.Fatalf("status %d", response.Code)
	}
}

func TestCustomsRecognitionSample(t *testing.T) {
	sample := os.Getenv("CUSTOMS_SAMPLE_PDF")
	if sample == "" {
		t.Skip("set CUSTOMS_SAMPLE_PDF and CUSTOMS_PYTHON for real PDF recognition")
	}
	content, err := os.ReadFile(sample)
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(cwd, "..", "..")); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)
	record, err := recognizeCustomsPDF(context.Background(), content)
	if err != nil {
		t.Fatal(err)
	}
	if record.DeclarationNumber != "310120260000665636" || len(record.Items) != 1 || record.Items[0]["total_price"] != "1819.16" {
		t.Fatalf("unexpected extraction: %+v", record)
	}
}
