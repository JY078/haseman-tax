package handler

import (
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAmazonQuarterSQLUsesExplicitMonthlyProjection(t *testing.T) {
	monthly := amazonQuarterSQL("USD", "store", "US", "USD", "month")
	if !strings.Contains(monthly, "SELECT period AS month") {
		t.Fatalf("monthly query must project its grouped period as month: %s", monthly)
	}
	if strings.Contains(monthly, "CASE WHEN month = 'month'") {
		t.Fatal("monthly query must not reference a nonexistent month column in totals")
	}
	quarterly := amazonQuarterSQL("USD", "store", "US", "USD")
	if !strings.Contains(quarterly, "SELECT NULL AS month") {
		t.Fatalf("quarterly query must leave month empty: %s", quarterly)
	}
}

func TestAggregateAmazonQuarterRowsSumsAmountsAndKeepsPendingValues(t *testing.T) {
	firstSales := decimal.RequireFromString("12.25")
	secondSales := decimal.RequireFromString("7.75")
	rows := []amazonTaxQuarter{
		{SalesVolume: decimal.NewFromInt(2), Sales: &firstSales},
		{SalesVolume: decimal.NewFromInt(3), Sales: &secondSales},
	}
	got := aggregateAmazonQuarterRows(rows)
	if !got.SalesVolume.Equal(decimal.NewFromInt(5)) || got.Sales == nil || !got.Sales.Equal(decimal.NewFromInt(20)) {
		t.Fatalf("incorrect country totals: %+v", got)
	}
	rows = append(rows, amazonTaxQuarter{Sales: nil})
	got = aggregateAmazonQuarterRows(rows)
	if got.Sales != nil {
		t.Fatalf("missing converted amount must remain pending, got %s", got.Sales)
	}
}

func TestAmazonSummaryJointImport(t *testing.T) {
	dsn, pdfPath, csvPath := os.Getenv("TAX_TEST_DSN"), os.Getenv("AMAZON_TEST_PDF"), os.Getenv("AMAZON_TEST_CSV")
	if dsn == "" || pdfPath == "" || csvPath == "" {
		t.Skip("set local database and Amazon fixtures")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	tx := db.Begin()
	defer tx.Rollback()
	schema := fmt.Sprintf("amazon_summary_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.TaxImportLog{}, &model.AmazonTaxFinance{}, &model.AmazonTaxShipment{}, &model.TaxExchangeRate{}, &model.AmazonProductCategory{}); err != nil {
		t.Fatal(err)
	}
	old := database.DB
	database.DB = tx
	defer func() { database.DB = old }()
	pdf, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatal(err)
	}
	csv, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.POST("/upload", UploadAmazonTaxFinance)
	router.GET("/quarters", ListAmazonTaxQuarters)
	upload := func(want int, confirmed ...bool) {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for _, file := range []struct {
			field, name string
			data        []byte
		}{{"file", "finance.csv", csv}, {"summary_file", "summary.pdf", pdf}} {
			part, _ := writer.CreateFormFile(file.field, file.name)
			part.Write(file.data)
		}
		if len(confirmed) > 0 && confirmed[0] {
			writer.WriteField("confirm_duplicate", "1")
		}
		writer.Close()
		req := httptest.NewRequest("POST", "/upload", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-RPA-Key", taxDataRPAKey)
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		if out.Code != want {
			t.Fatalf("upload: %d %s", out.Code, out.Body.String())
		}
	}
	upload(201)
	var count int64
	tx.Model(&model.AmazonTaxFinance{}).Count(&count)
	if count == 0 {
		t.Fatal("details missing")
	}
	// A repeated filename must wait for confirmation without inserting any rows.
	upload(409)
	var after int64
	tx.Model(&model.AmazonTaxFinance{}).Count(&after)
	if after != count {
		t.Fatal("unconfirmed import inserted rows")
	}
	// After confirmation, all financial rows are retained, even with the same identity.
	// A new PDF still persists independently of those existing rows.
	if err := tx.Where("report_type = ?", "finance_summary").Delete(&model.TaxImportLog{}).Error; err != nil {
		t.Fatal(err)
	}
	upload(201, true)
	tx.Model(&model.AmazonTaxFinance{}).Count(&after)
	if after != 2*count {
		t.Fatal("confirmed import must retain every financial row")
	}
	tx.Model(&model.TaxImportLog{}).Where("report_type = ?", "finance_summary").Count(&after)
	if after != 1 {
		t.Fatal("PDF not saved for existing details")
	}
	upload(409)
	// A duplicate PDF must not prevent a confirmed financial import.
	upload(201, true)
	tx.Model(&model.AmazonTaxFinance{}).Count(&after)
	if after != 3*count {
		t.Fatal("duplicate PDF blocked a confirmed financial import")
	}
	tx.Model(&model.TaxImportLog{}).Where("report_type = ?", "finance_summary").Count(&after)
	if after != 1 {
		t.Fatal("duplicate PDF was saved again")
	}
	// Restore a single financial copy for the amount-validation checks below.
	var firstImport uint
	tx.Model(&model.AmazonTaxFinance{}).Select("MIN(import_log_id)").Scan(&firstImport)
	if err := tx.Where("import_log_id <> ?", firstImport).Delete(&model.AmazonTaxFinance{}).Error; err != nil {
		t.Fatal(err)
	}
	out := httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest("GET", "/quarters?currency=USD", nil))
	if out.Code != 200 {
		t.Fatalf("quarters: %s", out.Body.String())
	}
	var response struct {
		Items []amazonTaxQuarter `json:"items"`
	}
	if err := json.Unmarshal(out.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 1 || len(response.Items[0].ValidationErrors) != 0 {
		t.Fatalf("matching PDF should not report a discrepancy: %s", out.Body.String())
	}
	if response.Items[0].Tax.String() != "-11887.64" {
		t.Fatalf("all types tax: %s", response.Items[0].Tax)
	}
	var log model.TaxImportLog
	if err := tx.Where("report_type = ?", "finance_summary").First(&log).Error; err != nil {
		t.Fatal(err)
	}
	var summary amazonSummary
	if err := json.Unmarshal([]byte(log.Message), &summary); err != nil {
		t.Fatal(err)
	}
	summary.Tax = summary.Tax.Add(decimal.RequireFromString("0.01"))
	message, _ := json.Marshal(summary)
	if err := tx.Model(&log).Update("message", string(message)).Error; err != nil {
		t.Fatal(err)
	}
	out = httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest("GET", "/quarters?currency=CNY", nil))
	if out.Code != 200 {
		t.Fatalf("quarter discrepancy must not block data: %s", out.Body.String())
	}
	if err := json.Unmarshal(out.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Items) != 1 || len(response.Items[0].ValidationErrors) != 1 {
		t.Fatalf("missing discrepancy: %s", out.Body.String())
	}
}

func TestAmazonSummaryValidation(t *testing.T) {
	dsn := os.Getenv("TAX_TEST_DSN")
	if dsn == "" {
		t.Skip("TAX_TEST_DSN required")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, _ := db.DB()
	defer conn.Close()
	db = db.Begin()
	defer db.Rollback()
	if err := db.Exec("CREATE TEMP TABLE tax_exchange_rates (date text, currency text, rate numeric)").Error; err != nil {
		t.Fatal(err)
	}
	rows := []model.AmazonTaxShipment{
		{Quarter: "2026-Q3", Data: map[string]string{"date/time": "Jul 1, 2026 1:00:00 PM PDT", "type": "Order", "marketplace withheld tax": "-2", "fba fees": "-3"}},
		{Quarter: "2026-Q3", Data: map[string]string{"date/time": "Jul 2, 2026 1:00:00 PM PDT", "type": "FBA Transaction fees", "description": "FBA Inventory Storage Fee", "fba fees": "-4"}},
	}
	summary := amazonSummary{Start: "2026-07-01", End: "2026-09-30", Currency: "USD", Tax: decimal.NewFromInt(-2), Storage: decimal.NewFromInt(-4), Delivery: decimal.NewFromInt(-3)}
	if err := validateAmazonSummary(db, rows, summary); err != nil {
		t.Fatal(err)
	}
	summary.Tax = decimal.NewFromInt(2)
	if err := validateAmazonSummary(db, rows, summary); err != nil {
		t.Fatal("amount mismatch must not block import", err)
	}
	summary.Tax = decimal.RequireFromString("-2.01")
	if err := validateAmazonSummary(db, rows, summary); err != nil {
		t.Fatal("amount mismatch must not block import", err)
	}
	summary.Start = "2026-08-01"
	if err := validateAmazonSummary(db, rows, summary); err != nil {
		t.Fatal("detail dates must not block PDF import", err)
	}
}

func TestAmazonSummaryIdentityIncludesDisplayName(t *testing.T) {
	base := amazonSummary{DisplayName: "sunoutly", Currency: "USD", Start: "2026-07-01", End: "2026-09-30"}
	if !sameAmazonSummaryIdentity(base, amazonSummary{DisplayName: "SUNOUTLY", Currency: "usd", Start: base.Start, End: base.End}) {
		t.Fatal("display name comparison should ignore case")
	}
	if sameAmazonSummaryIdentity(base, amazonSummary{DisplayName: "DearOcean", Start: base.Start, End: base.End}) {
		t.Fatal("different display names must not be treated as duplicate PDFs")
	}
	if sameAmazonSummaryIdentity(base, amazonSummary{DisplayName: base.DisplayName, Start: "2026-04-01", End: base.End}) {
		t.Fatal("different report periods must not be treated as duplicate PDFs")
	}
	if sameAmazonSummaryIdentity(base, amazonSummary{DisplayName: base.DisplayName, Currency: "CAD", Start: base.Start, End: base.End}) {
		t.Fatal("different currencies must not be treated as duplicate PDFs")
	}
}
