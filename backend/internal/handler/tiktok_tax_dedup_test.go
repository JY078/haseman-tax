package handler

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTikTokBusinessDedupIntegration(t *testing.T) {
	dsn := os.Getenv("TAX_TEST_DSN")
	if dsn == "" {
		t.Skip("set TAX_TEST_DSN for isolated PostgreSQL integration")
	}
	if !strings.Contains(dsn, "host=127.0.0.1") {
		t.Fatal("integration database must be local")
	}
	root, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	rootSQL, _ := root.DB()
	defer rootSQL.Close()
	schema := fmt.Sprintf("tiktok_dedup_%d", time.Now().UnixNano())
	if err := root.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer root.Exec("DROP SCHEMA " + schema + " CASCADE")
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	oldDB := database.DB
	database.DB = db
	defer func() { database.DB = oldDB }()
	if err := db.AutoMigrate(&model.TaxImportLog{}, &model.TaxDownloadLog{}, &model.User{}, &model.TikTokTaxBatch{}, &model.TikTokTaxRawRow{}, &model.TikTokTaxDetail{}, &model.TikTokTaxAdBatch{}, &model.TikTokTaxAd{}, &model.TaxExchangeRate{}, &model.TikTokSettlement{}, &model.TikTokOrder{}, &model.TikTokTaxFBT{}); err != nil {
		t.Fatal(err)
	}
	for _, date := range []string{"2026-01-02", "2026-01-05", "2026-02-02", "2026-04-01"} {
		rate := model.TaxExchangeRate{Date: date, Currency: "USD", Rate: decimal.NewFromInt(7)}
		if err := db.Create(&rate).Error; err != nil {
			t.Fatal(err)
		}
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/details", UploadTikTokTaxData)
	router.POST("/ads", UploadTikTokTaxAds)
	encode := func(headers []string, rows ...[]string) []byte {
		var buf bytes.Buffer
		w := csv.NewWriter(&buf)
		w.Write(headers)
		w.WriteAll(rows)
		if err := w.Error(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	send := func(path string, content []byte) *httptest.ResponseRecorder {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		part, _ := w.CreateFormFile("file", "report.csv")
		part.Write(content)
		w.WriteField("importer", "财务测试")
		w.Close()
		req := httptest.NewRequest(http.MethodPost, path, &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("X-RPA-Key", taxDataRPAKey)
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		return out
	}
	assertResponse := func(out *httptest.ResponseRecorder, added, skipped int) {
		t.Helper()
		want := fmt.Sprintf("入库 %d 条，跳过重复 %d 条", added, skipped)
		if out.Code != http.StatusCreated || !strings.Contains(out.Body.String(), want) {
			t.Fatalf("want %s, got %d %s", want, out.Code, out.Body.String())
		}
	}
	detail := func(date, id, amount string) []string { return []string{date, id, "USD", amount, "1", "2"} }
	assertResponse(send("/details", encode(tikTokTaxHeaders, detail("2026-01-02 12:00:00", " D1 ", "10"), detail("2026/1/2 12:00:00", "D1", "999"))), 1, 1)
	assertResponse(send("/details", encode(tikTokTaxHeaders, detail("2026-01-02T12:00:00Z", "D1", "888"), detail("2026-01-02 12:00:01", "D1", "20"), detail("2026-01-02 12:00:00", "D2", "30"))), 1, 2)
	var details []model.TikTokTaxDetail
	db.Order("id").Find(&details)
	if len(details) != 2 || details[0].Income.String() != "10" {
		t.Fatalf("detail identity/first record not preserved: %#v", details)
	}
	var rawCount int64
	db.Model(&model.TikTokTaxRawRow{}).Count(&rawCount)
	if rawCount != 2 {
		t.Fatalf("duplicate raw rows were stored: %d", rawCount)
	}
	ad := func(date, id, name, credit string) []string {
		return []string{date, name, id, "1", credit, "0", "10", "USD", "消耗"}
	}
	assertResponse(send("/ads", encode(tikTokAdHeaders, ad("2026-01-02", " A1 ", "first", "2"), ad("2026/1/2", "A1", "second", "999"))), 1, 1)
	assertResponse(send("/ads", encode(tikTokAdHeaders, ad("2026-01-02", "A1", "renamed", "888"), ad("2026-01-03", "A1", "next day", "3"), ad("2026-01-02", "A2", "other campaign", "4"))), 2, 1)
	assertResponse(send("/ads", encode(tikTokAdHeaders, ad("2026/1/2", "A1", "all repeated", "777"))), 0, 1)
	var ads []model.TikTokTaxAd
	db.Order("id").Find(&ads)
	if len(ads) != 3 || ads[0].CreditSpend.String() != "2" || ads[0].CampaignName != "first" {
		t.Fatalf("ad identity/first record not preserved: %#v", ads)
	}
	// Distinct files racing for the same business identity must insert only once.
	for _, path := range []string{"/details", "/ads"} {
		payloads := [][]byte{}
		for i := 0; i < 2; i++ {
			if path == "/details" {
				payloads = append(payloads, encode(tikTokTaxHeaders, detail("2026-01-02 13:00:00", "RACE", fmt.Sprint(50+i))))
			} else {
				payloads = append(payloads, encode(tikTokAdHeaders, ad("2026-01-02", "RACE", fmt.Sprint(i), "5")))
			}
		}
		start := make(chan struct{})
		outputs := make(chan *httptest.ResponseRecorder, 2)
		var wg sync.WaitGroup
		for _, payload := range payloads {
			wg.Add(1)
			go func(data []byte) { defer wg.Done(); <-start; outputs <- send(path, data) }(payload)
		}
		close(start)
		wg.Wait()
		close(outputs)
		for out := range outputs {
			if out.Code != http.StatusCreated {
				t.Fatalf("concurrent upload failed: %s", out.Body.String())
			}
		}
	}
	var n int64
	db.Model(&model.TikTokTaxDetail{}).Where("business_id = ?", "RACE").Count(&n)
	if n != 1 {
		t.Fatalf("concurrent detail duplicates: %d", n)
	}
	db.Model(&model.TikTokTaxAd{}).Where("campaign_id = ?", "RACE").Count(&n)
	if n != 1 {
		t.Fatalf("concurrent ad duplicates: %d", n)
	}
	// Additional months must remain available in server-side filter options.
	assertResponse(send("/ads", encode(tikTokAdHeaders, ad("2026-02-01", "FEB", "February", "6"), ad("2026-04-01", "APR", "April", "7"))), 2, 0)
	assertResponse(send("/details", encode(tikTokTaxHeaders, detail("2026-02-01", "FEB", "60"))), 1, 0)
	failed := send("/ads", []byte("日期,金额\nnot-a-date,1"))
	if failed.Code != 400 {
		t.Fatalf("invalid upload should fail: %s", failed.Body.String())
	}
	var failedLog model.TaxImportLog
	if err := db.Where("platform = ? AND status = ?", "tiktok", "failed").First(&failedLog).Error; err != nil || failedLog.Message == "" || len(failedLog.FileContent) == 0 {
		t.Fatalf("failed import log missing: %#v, %v", failedLog, err)
	}
	var importLog model.TaxImportLog
	if err := db.Where("platform = ? AND report_type = ? AND status = ?", "tiktok", "ads", "locked").Order("id").First(&importLog).Error; err != nil || importLog.SourceRows != 2 || importLog.ImportedRows != 1 || importLog.SkippedRows != 1 || importLog.Importer != "RPA" {
		t.Fatalf("incorrect audit: %#v, %v", importLog, err)
	}
	user := model.User{Username: "tax-log-test", Name: "财务测试"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	protected := router.Group("/test")
	protected.Use(func(c *gin.Context) { c.Set("userID", user.ID); c.Next() })
	protected.GET("/ads", ListTikTokTaxAds)
	protected.GET("/details", ListTikTokTaxDetails)
	protected.GET("/options", ListTikTokTaxFilterOptions)
	protected.GET("/export", ExportTikTokTaxReport)
	protected.GET("/imports", ListTaxImports)
	protected.GET("/imports/:id/file", DownloadTaxImport)
	protected.GET("/downloads", ListTaxDownloads)
	protected.GET("/downloads/:id/file", DownloadTaxHistoryFile)
	get := func(path string) *httptest.ResponseRecorder {
		out := httptest.NewRecorder()
		router.ServeHTTP(out, httptest.NewRequest(http.MethodGet, path, nil))
		return out
	}
	for _, kind := range []string{"ads", "details"} {
		seen := map[uint]bool{}
		for page := 1; page <= 2; page++ {
			out := get(fmt.Sprintf("/test/%s?month=2026-01&quarter=2026-Q1&page=%d&page_size=2", kind, page))
			var result struct {
				Items []struct {
					ID    uint   `json:"id"`
					Month string `json:"month"`
				} `json:"items"`
				Total int64 `json:"total"`
			}
			if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil || out.Code != 200 || result.Total != 4 || len(result.Items) != 2 {
				t.Fatalf("pagination %s page %d: %s", kind, page, out.Body.String())
			}
			for _, item := range result.Items {
				if seen[item.ID] || item.Month != "2026-01" {
					t.Fatalf("duplicate or unfiltered page row: %+v", item)
				}
				seen[item.ID] = true
			}
		}
		out := get("/test/" + kind + "?month=2026-02&quarter=2026-Q1")
		var result struct {
			Items []map[string]interface{} `json:"items"`
		}
		if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil || out.Code != 200 || len(result.Items) != 1 || result.Items[0]["month"] != "2026-02" {
			t.Fatalf("combined filter %s: %s", kind, out.Body.String())
		}
		out = get("/test/" + kind + "?month=2026-02&quarter=2026-Q2")
		if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil || len(result.Items) != 0 {
			t.Fatalf("conflicting filters: %s", out.Body.String())
		}
		out = get("/test/options?type=" + kind + "&field=month&quarter=2026-Q1&month=2026-01")
		var options struct {
			Values []string `json:"values"`
		}
		if err := json.Unmarshal(out.Body.Bytes(), &options); err != nil || len(options.Values) != 2 || options.Values[0] != "2026-02" {
			t.Fatalf("options limited to current selection: %s", out.Body.String())
		}
	}
	out := get("/test/export?type=ads&month=2026-02&quarter=2026-Q1")
	originalExport := append([]byte(nil), out.Body.Bytes()...)
	readCSV := func(data []byte) [][]string {
		t.Helper()
		records, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}))).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		return records
	}
	for _, kind := range []string{"ads", "details"} {
		out := get("/test/export?type=" + kind + "&month=2026-01&quarter=2026-Q1&page=2&page_size=1")
		if out.Code != 200 || len(readCSV(out.Body.Bytes())) != 5 {
			t.Fatalf("export must include all filtered rows: %s", out.Body.String())
		}
	}
	csvRows := readCSV(originalExport)
	if out.Code != 200 || len(csvRows) != 2 || csvRows[1][0] != "2026-02" || csvRows[1][7] != "6.00" {
		t.Fatalf("filtered export incorrect: %s", out.Body.String())
	}
	var download model.TaxDownloadLog
	if err := db.Where("platform = ?", "tiktok").First(&download).Error; err != nil || download.Month != "2026-02" || download.Quarter != "2026-Q1" || download.RowCount != 1 || download.Username != "财务测试" || !bytes.Equal(download.FileContent, originalExport) {
		t.Fatalf("download audit incorrect: %#v, %v", download, err)
	}
	if err := db.Model(&model.TikTokTaxAd{}).Where("campaign_id = ?", "FEB").Update("credit_spend", decimal.NewFromInt(88)).Error; err != nil {
		t.Fatal(err)
	}
	out = get(fmt.Sprintf("/test/downloads/%d/file?platform=tiktok", download.ID))
	if out.Code != 200 || !bytes.Equal(out.Body.Bytes(), originalExport) {
		t.Fatal("historical download changed with live data")
	}
	out = get(fmt.Sprintf("/test/imports/%d/file?platform=tiktok", importLog.ID))
	if out.Code != 200 || !bytes.Equal(out.Body.Bytes(), importLog.FileContent) {
		t.Fatal("original file cannot be downloaded")
	}
	out = get("/test/export?type=details&month=2026-02")
	if rows := readCSV(out.Body.Bytes()); out.Code != 200 || len(rows) != 2 || rows[1][4] != "FEB" {
		t.Fatalf("detail export incorrect: %s", out.Body.String())
	}
	out = get("/test/export?type=quarters&currency=USD")
	if rows := readCSV(out.Body.Bytes()); out.Code != 200 || len(rows[0]) < 4 || !strings.Contains(rows[0][3], "USD") {
		t.Fatalf("quarter currency export incorrect: %s", out.Body.String())
	}
	walmartLog := model.TaxImportLog{Importer: "Walmart", FileName: "walmart.csv", Status: "locked"}
	if err := db.Create(&walmartLog).Error; err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"imports", "downloads"} {
		out = get("/test/" + kind + "?platform=tiktok")
		var listed struct {
			Items []struct {
				Platform string `json:"platform"`
			} `json:"items"`
		}
		if err := json.Unmarshal(out.Body.Bytes(), &listed); err != nil || out.Code != 200 || len(listed.Items) == 0 {
			t.Fatalf("missing %s: %s", kind, out.Body.String())
		}
		for _, row := range listed.Items {
			if row.Platform != "tiktok" {
				t.Fatalf("platform leak: %s", out.Body.String())
			}
		}
	}
	out = get("/test/imports")
	var walmartList struct {
		Items []model.TaxImportLog `json:"items"`
	}
	if err := json.Unmarshal(out.Body.Bytes(), &walmartList); err != nil || len(walmartList.Items) != 1 || walmartList.Items[0].ID != walmartLog.ID {
		t.Fatalf("Walmart logs mixed: %s", out.Body.String())
	}
	if out := get(fmt.Sprintf("/test/imports/%d/file?platform=tiktok", walmartLog.ID)); out.Code != 404 {
		t.Fatal("cross-platform original file returned")
	}
	// Logged-in page imports must be attributed to the user, not RPA.
	router.POST("/test/ads", func(c *gin.Context) { c.Set("userID", user.ID); UploadTikTokTaxAds(c) })
	page := func(content []byte) *httptest.ResponseRecorder {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		part, _ := w.CreateFormFile("file", "page-ads.csv")
		part.Write(content)
		w.WriteField("importer", "冒名RPA")
		w.Close()
		req := httptest.NewRequest(http.MethodPost, "/test/ads", &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		return out
	}
	if out := page(encode(tikTokAdHeaders, ad("2026-03-01", "PAGE", "page import", "9"))); out.Code != http.StatusCreated {
		t.Fatalf("page ads upload failed: %s", out.Body.String())
	}
	var pageLog model.TaxImportLog
	if err := db.Where("file_name = ?", "page-ads.csv").First(&pageLog).Error; err != nil || pageLog.Importer != "财务测试" {
		t.Fatalf("page import audit wrong: %#v %v", pageLog, err)
	}

}
