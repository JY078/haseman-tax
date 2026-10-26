package handler

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func tikTokAdsCSV(t *testing.T, rows ...[]string) []byte {
	t.Helper()
	var data bytes.Buffer
	writer := csv.NewWriter(&data)
	writer.Write(tikTokAdHeaders)
	writer.WriteAll(rows)
	if err := writer.Error(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func TestTikTokAdCSV(t *testing.T) {
	data := tikTokAdsCSV(t, []string{"2026/1/2", "系列,名称", "0012345678901234567", "$1,000.10", "2.005", "3", "1005.105", "USD", "广告"}, []string{"", "", "", "", "", "", "", "", ""})
	rows, err := parseTikTokAdFile(data, ".csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].CampaignID != "0012345678901234567" || rows[0].Quarter != "2026-Q1" || rows[0].Month != "2026-01" || !rows[0].CreditSpend.Equal(decimal.RequireFromString("2.005")) {
		t.Fatalf("incorrect parse: %#v", rows)
	}
	for _, test := range []struct {
		field   int
		value   string
		message string
	}{{0, "合计", "日期无效"}, {4, "oops", "信用额度消耗无效"}, {7, "EUR", "USD"}} {
		row := []string{"2026-01-02", "name", "1", "1", "2", "3", "6", "USD", "广告"}
		row[test.field] = test.value
		if _, err := parseTikTokAdFile(tikTokAdsCSV(t, row), ".csv"); err == nil || !strings.Contains(err.Error(), test.message) {
			t.Fatalf("expected %s, got %v", test.message, err)
		}
	}
	if _, err := parseTikTokAdFile([]byte("日期,金额\n2026-01-01,1"), ".csv"); err == nil {
		t.Fatal("missing headers accepted")
	}
}

func TestTikTokAdXLSX(t *testing.T) {
	book := excelize.NewFile()
	defer book.Close()
	headers := make([]interface{}, len(tikTokAdHeaders))
	for i, header := range tikTokAdHeaders {
		headers[i] = header
	}
	if err := book.SetSheetRow("Sheet1", "A1", &headers); err != nil {
		t.Fatal(err)
	}
	row := []interface{}{45292.5, "名称", "0012345678901234567", 1.12345678, 2.005, 3, 100, "USD", "广告"}
	if err := book.SetSheetRow("Sheet1", "A2", &row); err != nil {
		t.Fatal(err)
	}
	total := []interface{}{"总计", "", "", 1.12345678, 2.005, 3, 100, "USD", ""}
	if err := book.SetSheetRow("Sheet1", "A3", &total); err != nil {
		t.Fatal(err)
	}
	book.NewSheet("不要导入")
	book.SetCellValue("不要导入", "A1", "说明")
	buf, err := book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseTikTokAdFile(buf.Bytes(), ".xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Date != "2024-01-01" || rows[0].CampaignID != "0012345678901234567" || !rows[0].CashSpend.Equal(decimal.RequireFromString("1.12345678")) {
		t.Fatalf("incorrect XLSX parse: %#v", rows)
	}
}

func TestTikTokAdsPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("TAX_TEST_DSN")
	if dsn == "" {
		t.Skip("set TAX_TEST_DSN for isolated PostgreSQL integration")
	}
	if !strings.Contains(dsn, "host=127.0.0.1") {
		t.Fatal("integration database must be local")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	oldDB := database.DB
	database.DB = tx
	t.Cleanup(func() { tx.Rollback(); database.DB = oldDB })
	schema := fmt.Sprintf("tiktok_test_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.TaxImportLog{}, &model.TaxDownloadLog{}, &model.User{}, &model.TikTokTaxAdBatch{}, &model.TikTokTaxAd{}, &model.TikTokTaxRawRow{}, &model.TikTokTaxDetail{}, &model.TikTokTaxFBT{}, &model.TaxExchangeRate{}, &model.TaxRawRow{}, &model.TaxProcessedRow{}, &model.TaxQuarter{}, &model.TikTokSettlement{}, &model.TikTokOrder{}, &model.AmazonTaxShipment{}, &model.AmazonTaxFinance{}); err != nil {
		t.Fatal(err)
	}
	rates := []model.TaxExchangeRate{{Date: "2026-01-02", Currency: "USD", Rate: decimal.RequireFromString("7.15")}, {Date: "2026-04-01", Currency: "USD", Rate: decimal.RequireFromString("7.2")}}
	if err := tx.Create(&rates).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/upload", UploadTikTokTaxAds)
	router.GET("/ads", ListTikTokTaxAds)
	router.GET("/quarters", ListTikTokTaxQuarters)
	upload := func(content []byte, key string, want int) {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, _ := writer.CreateFormFile("file", "ads.csv")
		part.Write(content)
		writer.Close()
		req := httptest.NewRequest(http.MethodPost, "/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-RPA-Key", key)
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		if out.Code != want {
			t.Fatalf("upload status %d, want %d: %s", out.Code, want, out.Body.String())
		}
		var result struct {
			Status bool   `json:"status"`
			Msg    string `json:"msg"`
		}
		if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil || result.Status != (want < 400) || result.Msg == "" {
			t.Fatalf("invalid response: %s", out.Body.String())
		}
	}
	data := tikTokAdsCSV(t, []string{"2026-01-02", "广告A", "1", "30", "2.005", "10", "100", "USD", "消耗"}, []string{"2026-01-02", "广告B", "2", "30", "2.005", "10", "100", "USD", "消耗"}, []string{"2026-04-01", "广告C", "3", "30", "4", "10", "100", "USD", "消耗"})
	upload(data, "wrong-key", http.StatusUnauthorized)
	upload(data, taxDataRPAKey, http.StatusCreated)
	upload(data, taxDataRPAKey, http.StatusOK)
	var count int64
	tx.Model(&model.TikTokTaxAd{}).Count(&count)
	if count != 3 {
		t.Fatalf("duplicate import: %d records", count)
	}
	for i := 0; i < 2; i++ {
		raw := model.TikTokTaxRawRow{BatchID: 1, Fingerprint: fmt.Sprint(i), Data: map[string]string{"币种": "USD"}}
		if err := tx.Create(&raw).Error; err != nil {
			t.Fatal(err)
		}
		date, _ := time.Parse("2006-01-02", "2026-01-02")
		detail := model.TikTokTaxDetail{BatchID: 1, RawRowID: raw.ID, Month: "2026-01", Quarter: "2026-Q1", PostedAt: date, Income: decimal.RequireFromString("10.005"), Refund: decimal.NewFromInt(2), Commission: decimal.NewFromInt(8), Rate: decimal.RequireFromString("7.15")}
		if err := tx.Create(&detail).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Create(&model.TikTokTaxFBT{Month: "2026-01", Quarter: "2026-Q1", FBTFulfillmentFee: decimal.NewFromInt(10), FBTFulfillmentFeeReimbursement: decimal.NewFromInt(20), HubPlacementFee: decimal.NewFromInt(30), StorageFee: decimal.NewFromInt(40)}).Error; err != nil {
		t.Fatal(err)
	}
	fbtRows, err := listTikTokQuarterSummary(tx, "USD")
	if err != nil || len(fbtRows) == 0 || fbtRows[0].Delivery.String() != "10" || fbtRows[0].OtherIncome.String() != "20" || fbtRows[0].HubFee.String() != "30" || fbtRows[0].StorageFee.String() != "40" {
		t.Fatalf("FBT summary not reflected: %#v, %v", fbtRows, err)
	}
	for _, check := range []struct{ currency, income, advertising, taxable string }{{"USD", "20.01", "4.01", "-4"}, {"CNY", "143.07", "28.67", "-28.6"}} {
		rows, err := listTikTokQuarterSummary(tx, check.currency)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 || rows[0].Income.String() != check.income || rows[0].Advertising.String() != check.advertising || rows[0].TaxableIncome.String() != check.taxable || !rows[1].TaxableIncome.IsZero() {
			t.Fatalf("incorrect %s summary: %#v", check.currency, rows)
		}
	}
	if err := tx.Model(&model.TaxExchangeRate{}).Where("date = ?", "2026-01-02").Update("rate", decimal.RequireFromString("7.2")).Error; err != nil {
		t.Fatal(err)
	}
	rows, err := listTikTokQuarterSummary(tx, "CNY")
	if err != nil || rows[0].Advertising.String() != "28.87" {
		t.Fatalf("rate refresh not reflected: %#v, %v", rows, err)
	}
	out := httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/ads", nil))
	var listed struct {
		Items []model.TikTokTaxAd `json:"items"`
	}
	if err := json.Unmarshal(out.Body.Bytes(), &listed); err != nil || out.Code != 200 || len(listed.Items) != 3 || !listed.Items[0].Fee.Equal(decimal.NewFromInt(720)) {
		t.Fatalf("incorrect ad listing: %s", out.Body.String())
	}
	out = httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/quarters?currency=USD", nil))
	if out.Code != 200 {
		t.Fatalf("quarter endpoint: %s", out.Body.String())
	}
	// The original USD data must be saved before any historical rate exists.
	pendingData := tikTokAdsCSV(t, []string{"2025-01-07", "待换算", "4", "1", "2", "0", "3", "USD", "消耗"})
	upload(pendingData, taxDataRPAKey, http.StatusCreated)
	upload(pendingData, taxDataRPAKey, http.StatusOK)
	var pending model.TikTokTaxAd
	if err := tx.Where("date = ?", "2025-01-07").First(&pending).Error; err != nil || !pending.Rate.IsZero() {
		t.Fatalf("pending import not saved: %#v, %v", pending, err)
	}
	out = httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/ads", nil))
	var nullableList struct {
		Items []struct {
			ID  uint             `json:"id"`
			Fee *decimal.Decimal `json:"fee"`
		} `json:"items"`
	}
	if err := json.Unmarshal(out.Body.Bytes(), &nullableList); err != nil || len(nullableList.Items) != 4 || nullableList.Items[0].Fee != nil {
		t.Fatalf("pending fee must be null: %s", out.Body.String())
	}
	pendingSummary, err := listTikTokQuarterSummary(tx, "CNY")
	if err != nil || pendingSummary[0].Advertising != nil || pendingSummary[0].MissingRates != 1 || pendingSummary[0].TaxableIncome == nil {
		t.Fatalf("pending summary: %#v, %v", pendingSummary, err)
	}
	usdSummary, err := listTikTokQuarterSummary(tx, "USD")
	if err != nil || usdSummary[0].Advertising == nil || usdSummary[0].Advertising.String() != "2" {
		t.Fatalf("USD must work without rates: %#v, %v", usdSummary, err)
	}
	earliest, err := earliestTaxExchangeDate(tx)
	if err != nil || earliest.Format("2006-01-02") != "2025-01-07" {
		t.Fatalf("ad date missing from refresh range: %v, %v", earliest, err)
	}
	// Manually imported history must be readable and usable without any provider call.
	if err := tx.Exec("INSERT INTO tax_exchange_rates (date, currency, rate) VALUES ('2025-01-07', 'USD', 7.14)").Error; err != nil {
		t.Fatal(err)
	}
	rawHistory := model.TikTokTaxRawRow{BatchID: 1, Fingerprint: "imported-history-test", Data: map[string]string{"币种": "USD"}}
	if err := tx.Create(&rawHistory).Error; err != nil {
		t.Fatal(err)
	}
	postedHistory, _ := time.Parse("2006-01-02", "2025-01-07")
	if err := tx.Create(&model.TikTokTaxDetail{BatchID: 1, RawRowID: rawHistory.ID, Month: "2025-01", Quarter: "2025-Q1", PostedAt: postedHistory, Income: decimal.NewFromInt(10), Refund: decimal.NewFromInt(2), Commission: decimal.NewFromInt(1)}).Error; err != nil {
		t.Fatal(err)
	}
	var convertedDetails []model.TikTokTaxDetail
	if err := queryTikTokDetails(tx, "2025-01", "").Find(&convertedDetails).Error; err != nil || len(convertedDetails) != 1 || convertedDetails[0].Fee.String() != "49.98" {
		t.Fatalf("old detail did not use imported daily rate: %#v %v", convertedDetails, err)
	}
	var convertedAds []model.TikTokTaxAd
	if err := queryTikTokAds(tx, "2025-01", "").Find(&convertedAds).Error; err != nil || len(convertedAds) != 1 || convertedAds[0].Fee.String() != "21.42" {
		t.Fatalf("history not applied: %#v %v", convertedAds, err)
	}
	// Existing weekend transactions still convert when Monday is a holiday.
	if err := tx.Exec("INSERT INTO tax_exchange_rates (date, currency, rate) VALUES ('2025-01-01', 'USD', 7)").Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("UPDATE tik_tok_tax_details SET posted_at = '2025-01-04' WHERE raw_row_id = ?", rawHistory.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("UPDATE tik_tok_tax_ads SET date = '2025-01-04' WHERE month = '2025-01'").Error; err != nil {
		t.Fatal(err)
	}
	holidaySummary, err := listTikTokQuarterSummary(tx, "CNY")
	if err != nil || holidaySummary[0].Income == nil || holidaySummary[0].Income.String() != "71.4" || holidaySummary[0].Commission.String() != "7.14" || holidaySummary[0].Advertising.String() != "14.28" || holidaySummary[0].MissingRates != 0 {
		t.Fatalf("holiday history not applied to quarter: %#v %v", holidaySummary, err)
	}
	router.GET("/rates", ListTaxExchangeRates)
	out = httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/rates?currency=USD", nil))
	if out.Code != 200 || !strings.Contains(out.Body.String(), "2025-01-07") {
		t.Fatalf("imported history invisible: %s", out.Body.String())
	}
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	requests := 0
	http.DefaultTransport = taxRoundTrip(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.Method != http.MethodPost || req.URL.Query().Has("date") {
			t.Fatal("must only fetch latest publication")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(chinaMoneyTestResponse))}, nil
	})
	router.POST("/refresh", RefreshTaxExchangeRates)
	for i := 0; i < 2; i++ {
		out = httptest.NewRecorder()
		router.ServeHTTP(out, httptest.NewRequest(http.MethodPost, "/refresh", nil))
		if out.Code != 200 {
			t.Fatalf("refresh: %s", out.Body.String())
		}
	}
	if requests != 2 {
		t.Fatalf("expected one request per refresh, got %d", requests)
	}
	var preserved model.TaxExchangeRate
	if err := tx.Where("date = ? AND currency = ?", "2026-01-02", "USD").First(&preserved).Error; err != nil || preserved.Rate.String() != "7.2" {
		t.Fatalf("historical rate overwritten: %v %v", preserved.Rate, err)
	}
	var latest []model.TaxExchangeRate
	if err := tx.Where("date = ?", "2026-10-14").Find(&latest).Error; err != nil || len(latest) != 4 {
		t.Fatalf("latest rates: %#v %v", latest, err)
	}

}

func TestTikTokAdSkipTrailingTotal(t *testing.T) {
	detail := []string{"2026-01-02", "总计活动", "001", "1", "2", "3", "6", "USD", "广告"}
	for _, labelColumn := range []int{0, 1} {
		total := []string{"", "", "", "1", "2", "3", "6", "USD", ""}
		total[labelColumn] = " 总计 "
		rows, err := parseTikTokAdFile(tikTokAdsCSV(t, detail, total, []string{"", " "}), ".csv")
		if err != nil || len(rows) != 1 || rows[0].CampaignID != "001" || !rows[0].CreditSpend.Equal(decimal.NewFromInt(2)) {
			t.Fatalf("total must not be imported: %#v, %v", rows, err)
		}
	}
	detail[1] = "总计"
	rows, err := parseTikTokAdFile(tikTokAdsCSV(t, detail), ".csv")
	if err != nil || len(rows) != 1 {
		t.Fatalf("valid campaign named 总计 must be preserved: %v", err)
	}
	if _, err := parseTikTokAdFile(tikTokAdsCSV(t, []string{"总计", "", "", "1", "2", "3", "6", "USD", ""}), ".csv"); err == nil || !strings.Contains(err.Error(), "没有数据行") {
		t.Fatalf("total-only file should have no data: %v", err)
	}
}
