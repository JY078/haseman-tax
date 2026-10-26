package handler

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestChunkTaxIDsStaysBelowPostgresParameterLimit(t *testing.T) {
	ids := make([]uint, 100000)
	for i := range ids {
		ids[i] = uint(i + 1)
	}
	chunks := chunkTaxIDs(ids)
	if len(chunks) != 20 {
		t.Fatalf("100000 ids were split into %d chunks, want 20", len(chunks))
	}
	for i, chunk := range chunks {
		if len(chunk) != 5000 || chunk[0] != uint(i*5000+1) || chunk[len(chunk)-1] != uint((i+1)*5000) {
			t.Fatalf("chunk %d has incorrect boundaries", i)
		}
	}
}

func taxTestCSV(t *testing.T, data ...map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(taxHeaders); err != nil {
		t.Fatal(err)
	}
	for _, m := range data {
		row := make([]string, len(taxHeaders))
		for i, h := range taxHeaders {
			row[i] = m[h]
		}
		if err := w.Write(row); err != nil {
			t.Fatal(err)
		}
	}
	w.Flush()
	if w.Error() != nil {
		t.Fatal(w.Error())
	}
	return buf.Bytes()
}
func taxTestRow(key, date, kind, amount string) map[string]string {
	desc := "Purchase"
	if kind == "Refund" {
		desc = "Return Refund"
	}
	return map[string]string{"Transaction Key": key, "Transaction Posted Timestamp": date, "Transaction Type": kind, "Transaction Description": desc, "Amount Type": "Product Price", "Amount": amount, "Currency": "USD"}
}
func TestTaxParsingAndProcessing(t *testing.T) {
	rules, err := model.DefaultTaxFeeRules()
	if err != nil || len(rules) != 54 {
		t.Fatalf("rules %d %v", len(rules), err)
	}
	data := taxTestCSV(t, taxTestRow("a", "2026-03-31T23:59:59-07:00", "Sale", "100.00"), taxTestRow("b", "2026-04-01 00:00:00", "Refund", "-10.00"))
	raw, err := parseTaxFile(append([]byte{0xef, 0xbb, 0xbf}, data...), ".csv")
	if err != nil {
		t.Fatal(err)
	}
	rates := map[string]model.TaxRate{"USD": {Rate: decimal.RequireFromString("7.125")}}
	processed, quarters, unmatched, err := processTaxRows(raw, rules, rates)
	if err != nil || unmatched != 0 || len(quarters) != 2 {
		t.Fatalf("process: %v %d %v", err, unmatched, quarters)
	}
	if processed[0].Month != "2026-03" || processed[0].Quarter != "2026-Q1" || processed[1].Quarter != "2026-Q2" {
		t.Fatal("posting date not preserved")
	}
	if processed[0].Fee.StringFixed(2) != "712.50" || quarters[1].TaxableIncome.StringFixed(2) != "-71.25" {
		t.Fatal("wrong decimal conversion/refund")
	}
	rates["USD"] = model.TaxRate{Rate: decimal.NewFromInt(8)}
	if processed[0].Rate.String() != "7.125" {
		t.Fatal("rate snapshot changed")
	}
	if _, _, _, err := processTaxRows(raw, rules, map[string]model.TaxRate{}); err == nil {
		t.Fatal("missing rate accepted")
	}
	bad := taxTestRow("bad", "bad date", "Sale", "10")
	if _, err := parseTaxFile(taxTestCSV(t, bad), ".csv"); err == nil {
		t.Fatal("invalid date accepted")
	}
	bad["Transaction Posted Timestamp"] = "2026-01-01"
	bad["Amount"] = "not money"
	if _, err := parseTaxFile(taxTestCSV(t, bad), ".csv"); err == nil {
		t.Fatal("invalid amount accepted")
	}
	if _, err := parseTaxFile(data, ".xls"); err == nil {
		t.Fatal("xls accepted")
	}
	if _, err := parseTaxFile([]byte("Amount\n10\n"), ".csv"); err == nil {
		t.Fatal("missing headers accepted")
	}
	repeated, err := parseTaxFile(taxTestCSV(t, taxTestRow("a", "2026-01-01", "Sale", "1"), taxTestRow("a", "2026-01-01", "Sale", "1")), ".csv")
	if err != nil || repeated[0].Fingerprint != repeated[1].Fingerprint {
		t.Fatal("identical keyed source lines must share identity")
	}
	noKey := taxTestRow("", "2026-01-01", "Sale", "1")
	noKeyRows, err := parseTaxFile(taxTestCSV(t, noKey, noKey), ".csv")
	if err != nil || len(noKeyRows) != 2 || noKeyRows[0].Fingerprint == noKeyRows[1].Fingerprint {
		t.Fatal("empty Transaction Key rows must both be retained")
	}
}
func TestTaxCreatorServiceFeeRule(t *testing.T) {
	rules, err := model.DefaultTaxFeeRules()
	if err != nil {
		t.Fatal(err)
	}
	want := model.TaxMatchKey("Service Fee - for walmart creators", "Walmart Social Commerce - Creators", "Debit Payable")
	for _, rule := range rules {
		if rule.MatchKey == want {
			if rule.CustomFeeType != "杂费" {
				t.Fatalf("fee type = %q, want 杂费", rule.CustomFeeType)
			}
			return
		}
	}
	t.Fatal("creator service fee rule not found")
}
func TestTaxAllFeeRulesAndTotals(t *testing.T) {
	rules, _ := model.DefaultTaxFeeRules()
	raw := []model.TaxRawRow{}
	for _, r := range rules {
		raw = append(raw, model.TaxRawRow{Data: map[string]string{"Transaction Type": r.TransactionType, "Transaction Description": r.TransactionDescription, "Amount Type": r.AmountType, "Amount": "1", "Currency": "CNY", "Transaction Posted Timestamp": "2026-01-01"}})
	}
	_, _, unmatched, err := processTaxRows(raw, rules, map[string]model.TaxRate{"CNY": {Rate: decimal.NewFromInt(1)}})
	if err != nil || unmatched != 0 {
		t.Fatalf("full mapping: %d %v", unmatched, err)
	}
	sale := taxTestRow("1", "2026-06-30", "Sale", "100")
	refund := taxTestRow("2", "2026-06-30", "Refund", "-20")
	commission := taxTestRow("3", "2026-06-30", "Sale", "-10")
	commission["Amount Type"] = "Commission on Product"
	payment := taxTestRow("4", "2026-06-30", "PaymentSummary", "70")
	payment["Transaction Description"] = "Deposited in PINGPONG account"
	payment["Amount Type"] = ""
	parsed, _ := parseTaxFile(taxTestCSV(t, sale, refund, commission, payment), ".csv")
	_, q, _, err := processTaxRows(parsed, rules, map[string]model.TaxRate{"USD": {Rate: decimal.NewFromInt(1)}})
	if err != nil || q[0].TaxableIncome.String() != "80" || q[0].Total.String() != "70" {
		t.Fatalf("incorrect settlement exclusion or income: %+v %v", q, err)
	}
	if model.TaxMatchKey("Adjustment", `"WFS Return Shipping fee "`, "Fee/Reimbursement") != model.TaxMatchKey("Adjustment", "WFS Return Shipping fee", "Fee/Reimbursement") {
		t.Fatal("quoted mapping mismatch")
	}
}

func TestTaxQuarterRoundsAfterAccumulation(t *testing.T) {
	rules, _ := model.DefaultTaxFeeRules()
	first := taxTestRow("round-1", "2026-01-01", "Sale", "1.005")
	second := taxTestRow("round-2", "2026-01-01", "Sale", "1.005")
	raw, err := parseTaxFile(taxTestCSV(t, first, second), ".csv")
	if err != nil {
		t.Fatal(err)
	}
	_, quarters, _, err := processTaxRows(raw, rules, map[string]model.TaxRate{"USD": {Rate: decimal.NewFromInt(1)}})
	if err != nil || len(quarters) != 1 {
		t.Fatalf("quarter processing: %v", err)
	}
	if quarters[0].Totals["销售额"].StringFixed(2) != "2.01" {
		t.Fatalf("quarter must round after summing exact values: %s", quarters[0].Totals["销售额"])
	}
}

func TestTaxExcel(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()
	row := taxTestRow("0000123456789012345", "2026-07-01 00:00:00", "Sale", "12.50")
	headers := make([]interface{}, len(taxHeaders))
	cells := make([]interface{}, len(taxHeaders))
	for i, h := range taxHeaders {
		headers[i] = h
		cells[i] = row[h]
	}
	if err := f.SetSheetRow("Sheet1", "A1", &headers); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("Sheet1", "A2", &cells); err != nil {
		t.Fatal(err)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := parseTaxFile(buf.Bytes(), ".xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 1 || raw[0].Data["Transaction Key"] != row["Transaction Key"] {
		t.Fatal("Excel identifier lost")
	}
}
func TestTaxAmountValidation(t *testing.T) {
	for _, s := range []string{"NaN", "Infinity", "1,2", "12,34.5", "1e100000", "0.123456789"} {
		if _, e := parseTaxAmount(s); e == nil {
			t.Errorf("accepted %q", s)
		}
	}
	d, e := parseTaxAmount("(1,234.50)")
	if e != nil || d.String() != "-1234.5" {
		t.Fatal(d, e)
	}
}
func TestTaxDepartmentAccess(t *testing.T) {
	for _, tc := range []struct {
		user    model.User
		allowed bool
	}{
		{model.User{Roles: []model.Role{{Name: "admin"}}}, true},
		{model.User{Department: "财务部"}, true}, {model.User{SubDepartment: "财务核算"}, true},
		{model.User{Department: "数据部"}, false}, {model.User{}, false},
	} {
		if taxUserAllowed(tc.user) != tc.allowed {
			t.Fatalf("permission mismatch %+v", tc.user)
		}
	}
}

type taxRoundTrip func(*http.Request) (*http.Response, error)

func (f taxRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestTaxRateAndFixedRPAKey(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	http.DefaultTransport = taxRoundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 403, Body: io.NopCloser(strings.NewReader("blocked")), Header: make(http.Header)}, nil
	})
	if _, err := fetchTaxRate(context.Background(), "USD"); err == nil {
		t.Fatal("unavailable rate accepted")
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/upload", UploadTaxDataFromRPA)
	t.Setenv("TAX_DATA_RPA_KEY", "ignored-env-key")
	for _, key := range []string{"", "wrong-key", "ignored-env-key", taxDataRPAKey} {
		req := httptest.NewRequest("POST", "/upload", nil)
		req.Header.Set("X-RPA-Key", key)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		want := 401
		if key == taxDataRPAKey {
			want = 400
		} // Authorized; missing multipart body.
		if w.Code != want {
			t.Fatalf("status %d, want %d", w.Code, want)
		}
	}
}

func TestTaxPostgresIntegration(t *testing.T) {
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
	schema := fmt.Sprintf("tax_test_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.TaxDataBatch{}, &model.TaxRawRow{}, &model.TaxProcessedRow{}, &model.TaxQuarter{}, &model.TaxFeeRule{}, &model.TaxImportLog{}, &model.TaxDownloadLog{}, &model.TaxExchangeRate{}, &model.User{}); err != nil {
		t.Fatal(err)
	}
	rules, _ := model.DefaultTaxFeeRules()
	if err := tx.Create(&rules).Error; err != nil {
		t.Fatal(err)
	}
	oldTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	rate := "7"
	http.DefaultTransport = taxRoundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"meta":{"code":200},"response":{"from":"USD","to":"CNY","amount":1,"value":` + rate + `,"date":"2026-10-02","timestamp":1790899200}}`))}, nil
	})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/upload", UploadTaxDataFromRPA)
	r.GET("/quarters", ListTaxQuarters)
	r.GET("/raw/:id", ListTaxRaw)
	r.GET("/raw", ListTaxRaw)
	r.GET("/processed/:id", ListTaxProcessed)
	uploadStore := "shop-1"
	send := func(content []byte, status int) model.TaxDataBatch {
		t.Helper()
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		_ = mw.WriteField("store_id", uploadStore)
		_ = mw.WriteField("importer", "测试财务")
		part, _ := mw.CreateFormFile("file", "statement.csv")
		_, _ = part.Write(content)
		_ = mw.Close()
		req := httptest.NewRequest("POST", "/upload", &body)
		req.Header.Set("X-RPA-Key", taxDataRPAKey)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != status {
			t.Fatalf("upload status %d want %d: %s", w.Code, status, w.Body.String())
		}
		var response struct {
			Status bool   `json:"status"`
			Msg    string `json:"msg"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		if (status == 201 || status == 200) && !response.Status {
			t.Fatalf("upload failed: %s", w.Body.String())
		}
		var batch model.TaxDataBatch
		if response.Status {
			if err := tx.Order("id desc").First(&batch).Error; err != nil {
				t.Fatal(err)
			}
		}
		return batch
	}
	csv := taxTestCSV(t, taxTestRow("a", "2026-03-31", "Sale", "100"), taxTestRow("b", "2026-04-01", "Refund", "-10"))
	first := send(csv, 201)
	if first.RawRows != 2 || first.Status != "locked" {
		t.Fatalf("wrong batch %+v", first)
	}
	rate = "8"
	retry := send(csv, 200)
	if retry.ID != first.ID || retry.Rates["USD"].Rate.String() != "7" {
		t.Fatal("idempotency changed rate")
	}
	send(taxTestCSV(t, taxTestRow("a", "2026-03-31", "Sale", "999")), 200)
	second := send(taxTestCSV(t, taxTestRow("c", "2026-03-31", "Sale", "1")), 201)
	if second.Rates["USD"].Rate.String() != "8" {
		t.Fatal("new snapshot not taken")
	}
	unknown := taxTestRow("d", "2026-07-01", "Unknown", "1000")
	third := send(taxTestCSV(t, unknown, taxTestRow("q3-sale", "2026-07-01", "Sale", "2")), 201)
	if third.Status != "needs_review" {
		t.Fatal("unknown mapping marked complete")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/quarters", nil))
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var summary struct {
		Items []model.TaxQuarter `json:"items"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if len(summary.Items) != 3 || summary.Items[0].TaxableIncome.String() != "708" || summary.Items[1].TaxableIncome.String() != "-70" || summary.Items[2].Total.String() != "16" || summary.Items[2].Totals["未匹配"].String() != "8000" {
		t.Fatalf("stored aggregation: %s", w.Body.String())
	}
	for _, test := range []struct {
		url   string
		total int
		count int
	}{
		{"/raw", 5, 5},
		{"/raw?quarter=2026-Q1&page_size=1&page=2", 2, 1},
		{"/raw?quarter=2026-Q2", 1, 1},
		{"/raw?quarter=2026-Q1,2026-Q2&page_size=1&page=2", 3, 1},
		{"/raw?filters=" + url.QueryEscape(`{"月份":"2026-03,2026-07"}`), 4, 4},
		{"/raw?filters=" + url.QueryEscape(`{"月份":"2026-03,2026-07","季度":"2026-Q2,2026-Q3"}`), 2, 2},
		{"/raw?quarter=2026-Q3", 2, 2},
		{"/raw?quarter=2025-Q4", 0, 0},
		{"/raw?quarter=2026-Q1&store_id=other", 0, 0},
		{fmt.Sprintf("/raw/%d?quarter=2026-Q2", first.ID), 1, 1},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", test.url, nil))
		var data struct {
			Items []model.TaxRawRow `json:"items"`
			Total int               `json:"total"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil || w.Code != 200 || data.Total != test.total || len(data.Items) != test.count {
			t.Fatalf("quarter filter %s: %s", test.url, w.Body.String())
		}
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/quarters?quarter=2026-Q2", nil))
	var filtered struct {
		Items     []model.TaxQuarter `json:"items"`
		Available []string           `json:"available_quarters"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &filtered); err != nil || w.Code != 200 || len(filtered.Items) != 1 || filtered.Items[0].Quarter != "2026-Q2" || strings.Join(filtered.Available, ",") != "2026-Q3,2026-Q2,2026-Q1" {
		t.Fatalf("summary filter/options: %s", w.Body.String())
	}
	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/raw?quarter=2026-Q1", nil))
	var rateRows struct {
		Items []model.TaxRawRow `json:"items"`
		Rates map[uint]struct {
			Rate     decimal.Decimal `json:"rate"`
			Currency string          `json:"currency"`
			Month    string          `json:"month"`
			Quarter  string          `json:"quarter"`
			FeeType  string          `json:"fee_type"`
			Fee      decimal.Decimal `json:"fee"`
		} `json:"rates"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rateRows); err != nil || w.Code != 200 || len(rateRows.Items) != 2 {
		t.Fatalf("raw rate response: %s", w.Body.String())
	}
	for _, row := range rateRows.Items {
		want := "7"
		if row.BatchID == second.ID {
			want = "8"
		}
		detail := rateRows.Rates[row.ID]
		amount, _ := parseTaxAmount(row.Data["Amount"])
		if detail.Month != "2026-03" || detail.Quarter != "2026-Q1" || detail.FeeType != "销售额" || !detail.Fee.Equal(amount.Mul(detail.Rate).Round(2)) {
			t.Fatal("display fields must come from stored processing results")
		}
		if rateRows.Rates[row.ID].Rate.String() != want || rateRows.Rates[row.ID].Currency != "USD" {
			t.Fatalf("raw row %d did not retain its own rate", row.ID)
		}
	}
	for _, endpoint := range []string{"/raw/", "/processed/"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", fmt.Sprintf("%s%d?page_size=1&page=2", endpoint, first.ID), nil))
		var data struct {
			Items []json.RawMessage `json:"items"`
			Total int               `json:"total"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil || data.Total != 2 || len(data.Items) != 1 {
			t.Fatalf("pagination %s", w.Body.String())
		}
	}
	var stored model.TaxDataBatch
	if err := tx.First(&stored, first.ID).Error; err != nil || !bytes.Equal(stored.FileContent, csv) {
		t.Fatal("source file not preserved")
	}
	var count int64
	tx.Model(&model.TaxDataBatch{}).Count(&count)
	if count != 4 {
		t.Fatal("rollback/idempotency failed", count)
	}
	http.DefaultTransport = taxRoundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 403, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("blocked"))}, nil
	})
	send(taxTestCSV(t, taxTestRow("failure", "2026-07-01", "Sale", "9")), 503)
	tx.Model(&model.TaxDataBatch{}).Count(&count)
	if count != 4 {
		t.Fatal("failed FX import persisted a batch")
	}
	// Real DB-backed permission check, including department revocation.
	user := model.User{Username: "finance", Department: "财务部"}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	guarded := gin.New()
	guarded.Use(func(c *gin.Context) { c.Set("userID", user.ID) })
	guarded.GET("/data", RequireTaxDataAccess(), ListTaxData)
	w = httptest.NewRecorder()
	guarded.ServeHTTP(w, httptest.NewRequest("GET", "/data", nil))
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	var imports []model.TaxImportLog
	if err := tx.Order("id").Find(&imports).Error; err != nil {
		t.Fatal(err)
	}
	if len(imports) != 6 || imports[0].ImportedRows != 2 || imports[0].Importer != "RPA 上报：测试财务" || !bytes.Equal(imports[0].FileContent, csv) || imports[1].SkippedRows != 2 || imports[2].ImportedRows != 0 || imports[2].SkippedRows != 1 || imports[5].Status != "failed" {
		t.Fatalf("import audit inaccurate: %+v", imports)
	}
	guarded.GET("/imports", RequireTaxDataAccess(), ListTaxImports)
	guarded.GET("/imports/:id/file", RequireTaxDataAccess(), DownloadTaxImport)
	guarded.GET("/downloads", RequireTaxDataAccess(), ListTaxDownloads)
	guarded.GET("/downloads/:id/file", RequireTaxDataAccess(), DownloadTaxHistoryFile)
	guarded.GET("/export", RequireTaxDataAccess(), ExportTaxReport)
	get := func(url string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		guarded.ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", url, w.Code, w.Body.String())
		}
		return w
	}
	original := get(fmt.Sprintf("/imports/%d/file", imports[0].ID))
	if !bytes.Equal(original.Body.Bytes(), csv) {
		t.Fatal("original file changed")
	}
	exported := get("/export?type=raw&quarter=2026-Q1")
	records, err := csvReadForTaxTest(exported.Body.Bytes())
	if err != nil || len(records) != 3 || records[1][0] != "2026-03" || records[1][3] != "700.00" || records[1][5+4] != "a" {
		t.Fatalf("filtered raw export: %v %s", err, exported.Body.String())
	}
	quarterExport := get("/export?type=quarters&quarter=2026-Q1")
	if !strings.Contains(quarterExport.Body.String(), "708.00") || strings.Contains(quarterExport.Body.String(), "2026-Q2") {
		t.Fatal("quarter export filter/totals incorrect")
	}
	q3 := get("/export?type=quarters&quarter=2026-Q3")
	q3Rows, err := csvReadForTaxTest(q3.Body.Bytes())
	if err != nil || len(q3Rows) != 2 || !strings.Contains(q3.Body.String(), "8000.00,16.00,16.00,1") {
		t.Fatal("partially matched batch missing from export", q3.Body.String())
	}
	var downloads []model.TaxDownloadLog
	tx.Order("id").Find(&downloads)
	if len(downloads) != 4 || downloads[0].UserID != user.ID || downloads[0].Username != "finance" || downloads[1].ReportType != "raw" || downloads[1].RowCount != 2 {
		t.Fatal("download audit incorrect")
	}
	replay := get(fmt.Sprintf("/downloads/%d/file", downloads[1].ID))
	if !bytes.Equal(replay.Body.Bytes(), exported.Body.Bytes()) {
		t.Fatal("download snapshot changed")
	}
	get("/imports?page_size=1&page=2")
	get("/downloads?page_size=1&page=2")
	// In-file duplicates and overlap with an old batch are skipped, while distinct types survive.
	http.DefaultTransport = taxRoundTrip(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"meta":{"code":200},"response":{"from":"USD","to":"CNY","amount":1,"value":8,"date":"2026-10-02","timestamp":1790899200}}`))}, nil
	})
	fresh := taxTestRow("new", "2026-03-31", "Sale", "2")
	if err := tx.Model(&model.TaxRawRow{}).Where("batch_id = ?", first.ID).Update("fingerprint", gorm.Expr("'legacy-' || id")).Error; err != nil {
		t.Fatal(err)
	}
	mixed := send(taxTestCSV(t, taxTestRow("a", "2026-03-31", "Sale", "2000"), fresh, fresh, taxTestRow("new", "2026-03-31", "Refund", "-1")), 201)
	if mixed.RawRows != 2 {
		t.Fatal("mixed duplicate import must keep two distinct tuples", mixed.RawRows)
	}
	// Mixed generations: snapshots and raw-only legacy batches must coexist with new details.
	baseline, _, err := summarizeTaxProcessed(tx, "shop-1", "2026-Q1")
	if err != nil || len(baseline) != 1 {
		t.Fatal("baseline summary", err)
	}
	if err := tx.Where("batch_id = ?", second.ID).Delete(&model.TaxProcessedRow{}).Error; err != nil {
		t.Fatal(err)
	}
	snapshotSummary, _, err := summarizeTaxProcessed(tx, "shop-1", "2026-Q1")
	if err != nil || len(snapshotSummary) != 1 || !snapshotSummary[0].Total.Equal(baseline[0].Total) {
		t.Fatal("mixed snapshot fallback dropped or duplicated a batch", err)
	}
	if err := tx.Where("batch_id = ?", second.ID).Delete(&model.TaxQuarter{}).Error; err != nil {
		t.Fatal(err)
	}
	rawSummary, _, err := summarizeTaxProcessed(tx, "shop-1", "2026-Q1")
	if err != nil || len(rawSummary) != 1 || !rawSummary[0].Total.Equal(baseline[0].Total) {
		t.Fatal("raw-only fallback must use saved rate", err)
	}
	if err := tx.Model(&model.TaxDataBatch{}).Where("id = ?", second.ID).Update("rates", "{}").Error; err != nil {
		t.Fatal(err)
	}
	if _, _, err := summarizeTaxProcessed(tx, "shop-1", "2026-Q1"); err == nil {
		t.Fatal("missing historical rate must not silently produce an empty summary")
	}
	uploadStore = "shop-2"
	otherStore := send(csv, 201)
	if otherStore.RawRows != 2 {
		t.Fatal("different stores must not share duplicate identities")
	}
	// Page imports (logged-in session, no RPA key) must record the user as importer,
	// ignoring any importer form field.
	pageUpload := func(content []byte, want int) {
		t.Helper()
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		_ = mw.WriteField("store_id", uploadStore)
		_ = mw.WriteField("importer", "冒名RPA")
		part, _ := mw.CreateFormFile("file", "page-statement.csv")
		_, _ = part.Write(content)
		_ = mw.Close()
		req := httptest.NewRequest("POST", "/data/upload", &body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		w := httptest.NewRecorder()
		guarded.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("page upload status %d want %d: %s", w.Code, want, w.Body.String())
		}
	}
	guarded.POST("/data/upload", RequireTaxDataAccess(), UploadTaxDataFromRPA)
	pageUpload(taxTestCSV(t, taxTestRow("page-user", "2026-03-31", "Sale", "5")), 201)
	var pageLog model.TaxImportLog
	if err := tx.Where("file_name = ?", "page-statement.csv").First(&pageLog).Error; err != nil || pageLog.Importer != "finance" || pageLog.ImportedRows != 1 {
		t.Fatalf("page import audit wrong: %+v %v", pageLog, err)
	}
	tx.Model(&user).Update("department", "运营部")
	w = httptest.NewRecorder()
	guarded.ServeHTTP(w, httptest.NewRequest("GET", "/data", nil))
	if w.Code != 403 {
		t.Fatal("revoked department still allowed", w.Code)
	}
	for _, endpoint := range []string{"/imports", "/downloads", "/export?type=raw", fmt.Sprintf("/imports/%d/file", imports[0].ID), fmt.Sprintf("/downloads/%d/file", downloads[0].ID)} {
		w := httptest.NewRecorder()
		guarded.ServeHTTP(w, httptest.NewRequest("GET", endpoint, nil))
		if w.Code != 403 {
			t.Fatal("unauthorized log/download access", endpoint, w.Code)
		}
	}
}

const chinaMoneyTestResponse = `{"head":{"rep_code":"200","ts":1791941140597},"data":{"lastDate":"2026-10-14 9:15"},"records":[{"vrtEName":"USD/CNY","price":"6.7249"},{"vrtEName":"EUR/CNY","price":"7.5165"},{"vrtEName":"GBP/CNY","price":"8.8956"},{"vrtEName":"CAD/CNY","price":"4.7180"},{"vrtEName":"CNY/MXN","price":"2.7341"},{"vrtEName":"CHF/CNY","price":"8.0613"},{"vrtEName":"CNY/PLN","price":"0.58099"}]}`

func TestTaxChinaMoneyConversion(t *testing.T) {
	old := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = old })
	http.DefaultTransport = taxRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Host != "www.chinamoney.com.cn" || r.URL.Query().Get("t") == "" || r.URL.Query().Has("date") || r.Header.Get("Referer") == "" || r.Header.Get("Cookie") != "" {
			t.Fatal("incorrect ChinaMoney request")
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(chinaMoneyTestResponse))}, nil
	})
	rates, err := fetchChinaMoneyRates(context.Background())
	if err != nil || len(rates) != 7 {
		t.Fatalf("rates: %+v %v", rates, err)
	}
	for currency, want := range map[string]string{"USD": "6.7249", "EUR": "7.5165", "CAD": "4.718", "GBP": "8.8956", "MXN": "0.365751069822", "CHF": "8.0613", "PLN": "1.721200020654"} {
		rate := rates[currency]
		if rate.Rate.String() != want || rate.Source != "ChinaMoney" || rate.Date != "2026-10-14" || rate.ProviderTimestamp != 1791941140597 {
			t.Fatalf("invalid snapshot: %+v", rate)
		}
	}
	for _, body := range []string{
		strings.Replace(chinaMoneyTestResponse, `"200"`, `"500"`, 1),
		strings.Replace(chinaMoneyTestResponse, `"6.7249"`, `"0"`, 1),
		strings.Replace(chinaMoneyTestResponse, `"USD/CNY"`, `"CNY/USD"`, 1),
		strings.Replace(chinaMoneyTestResponse, "2026-10-14", "2026-99-14", 1),
		`{invalid-json`,
	} {
		http.DefaultTransport = taxRoundTrip(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, err := fetchChinaMoneyRates(context.Background()); err == nil {
			t.Fatal("invalid response accepted")
		}
	}
	if _, err := fetchTaxRate(context.Background(), "JPY"); err == nil {
		t.Fatal("unsupported currency accepted")
	}
}

func TestTaxProcessingUsesStoredDailyRates(t *testing.T) {
	rows, err := parseTaxFile(taxTestCSV(t, taxTestRow("one", "2026-01-02", "Sale", "10"), taxTestRow("two", "2026-01-05", "Sale", "20")), ".csv")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := model.DefaultTaxFeeRules()
	if err != nil {
		t.Fatal(err)
	}
	rates := map[string]model.TaxRate{
		"USD":            {Rate: decimal.Zero},
		"USD:2026-01-02": {Rate: decimal.RequireFromString("7.1457")},
		"USD:2026-01-05": {Rate: decimal.RequireFromString("7.123456")},
	}
	details, _, _, err := processTaxRows(rows, rules, rates)
	if err != nil || len(details) != 2 || details[0].Fee.String() != "71.46" || details[1].Fee.String() != "142.47" || details[1].Rate.String() != "7.123456" {
		t.Fatalf("daily conversion: %#v %v", details, err)
	}
	delete(rates, "USD:2026-01-02")
	details, _, _, err = processTaxRows(rows, rules, rates)
	if err != nil || !details[0].Rate.IsZero() {
		t.Fatalf("missing daily rate must allow pending import: %#v %v", details, err)
	}
}

func TestTaxExchangeSchedule(t *testing.T) {
	for input, want := range map[string]string{
		"2026-10-14T01:29:59Z": "2026-10-14T09:30:00+08:00",
		"2026-10-14T01:30:00Z": "2026-10-15T09:30:00+08:00",
		"2026-12-31T23:00:00Z": "2027-01-01T09:30:00+08:00",
	} {
		now, _ := time.Parse(time.RFC3339, input)
		if got := nextTaxExchangeFetch(now).Format(time.RFC3339); got != want {
			t.Fatalf("next run: %s want %s", got, want)
		}
	}
}

func TestTaxIgnoresFileLineCountNotice(t *testing.T) {
	original := taxTestCSV(t, taxTestRow("a", "2026-09-05", "Sale", "10"))
	parts := bytes.SplitN(original, []byte("\n"), 2)
	for _, notice := range []string{"Number of Lines in file 5021            ", "Number of Lines in file 5021,,,", "Number of Lines in file 5021,other metadata"} {
		content := append(append(append([]byte{}, parts[0]...), []byte("\n"+notice+"\n")...), parts[1]...)
		rows, err := parseTaxFile(content, ".csv")
		if err != nil || len(rows) != 1 || rows[0].RowNumber != 3 || rows[0].Data["Transaction Key"] != "a" {
			t.Fatalf("notice not skipped correctly: %v %+v", err, rows)
		}
	}
	rows, err := parseTaxFile(original, ".csv")
	if err != nil || len(rows) != 1 || rows[0].RowNumber != 2 {
		t.Fatal("normal second row must be retained", err)
	}
	for _, notice := range []string{"Number of Lines in file invalid"} {
		content := append(append(append([]byte{}, parts[0]...), []byte("\n"+notice+"\n")...), parts[1]...)
		if _, err := parseTaxFile(content, ".csv"); err == nil {
			t.Fatal("unexpected invalid row was silently ignored")
		}
	}
}

func TestTaxEmptyAmountAndCurrency(t *testing.T) {
	row := taxTestRow("empty", "2026-09-05", "Sale", "")
	row["Currency"] = ""
	raw, err := parseTaxFile(taxTestCSV(t, row), ".csv")
	if err != nil {
		t.Fatal(err)
	}
	rules, _ := model.DefaultTaxFeeRules()
	processed, _, _, err := processTaxRows(raw, rules, map[string]model.TaxRate{"USD": {Rate: decimal.NewFromInt(7)}})
	if err != nil || len(processed) != 1 {
		t.Fatal(err)
	}
	if raw[0].Data["Currency"] != "" || raw[0].Data["Amount"] != "" || !processed[0].Fee.IsZero() || processed[0].Currency != "USD" {
		t.Fatal("empty fields not handled correctly")
	}
	row["Amount"] = "10"
	raw, err = parseTaxFile(taxTestCSV(t, row), ".csv")
	if err != nil {
		t.Fatal(err)
	}
	processed, _, _, err = processTaxRows(raw, rules, map[string]model.TaxRate{"USD": {Rate: decimal.NewFromInt(7)}})
	if err != nil || !processed[0].Fee.Equal(decimal.NewFromInt(70)) {
		t.Fatal("empty currency conversion failed", err)
	}
}

func TestTaxProvidedBill(t *testing.T) {
	path := os.Getenv("TAX_TEST_BILL")
	if path == "" {
		t.Skip("set TAX_TEST_BILL to validate a local bill without importing it")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := parseTaxFile(content, ".csv")
	if err != nil {
		t.Fatal(err)
	}
	rules, _ := model.DefaultTaxFeeRules()
	processed, _, unmatched, err := processTaxRows(raw, rules, map[string]model.TaxRate{"USD": {Rate: decimal.NewFromInt(7)}})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("parsed=%d processed=%d unmatched=%d (test rate only; no database writes)", len(raw), len(processed), unmatched)
}

func csvReadForTaxTest(content []byte) ([][]string, error) {
	return csv.NewReader(bytes.NewReader(bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf}))).ReadAll()
}

func TestTaxIdentityAndExportText(t *testing.T) {
	base := taxTestRow("key", "2026-01-01", "Sale", "1")
	original := taxRowIdentity(base)
	base["Amount"] = "999"
	if taxRowIdentity(base) != original {
		t.Fatal("amount must not affect identity")
	}
	base["Transaction Key"] = ""
	base["Transaction Description"] = "different"
	if taxRowIdentity(base) != "" {
		t.Fatal("empty Transaction Key must disable deduplication")
	}
	for _, field := range []string{"Transaction Key", "Transaction Type", "Transaction Description", "Amount Type"} {
		before := base[field]
		base[field] = before + "changed"
		if taxRowIdentity(base) == original {
			t.Fatal("missing identity component", field)
		}
		base[field] = before
	}
	if taxCSVText("=SUM(1,2)") != "'=SUM(1,2)" || taxCSVText("-10.50") != "-10.50" {
		t.Fatal("unsafe CSV text or corrupted negative amount")
	}
}

func TestTaxSkipsPaymentSummaryRow(t *testing.T) {
	summary := taxTestRow("", "", "PaymentSummary", "not a transaction amount")
	rows, err := parseTaxFile(taxTestCSV(t, summary, taxTestRow("sale", "2026-09-05", "Sale", "12")), ".csv")
	if err != nil || len(rows) != 1 || rows[0].RowNumber != 3 || rows[0].Data["Transaction Key"] != "sale" {
		t.Fatalf("summary row was not excluded: %v %+v", err, rows)
	}
}
