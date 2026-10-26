package handler

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func tikTokOrderCSV(t *testing.T, records ...map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	headers := []string{}
	for _, column := range tikTokOrderColumns {
		headers = append(headers, column.Key)
	}
	// Exports can contain NBSP around this header.
	headers[10] = "\u00a0Virtual Bundle Seller SKU "
	w.Write(headers)
	for _, record := range records {
		values := make([]string, len(headers))
		for i, key := range headers {
			values[i] = record[strings.TrimSpace(key)]
		}
		w.Write(values)
	}
	w.Flush()
	if w.Error() != nil {
		t.Fatal(w.Error())
	}
	return buf.Bytes()
}
func TestTikTokOrderParser(t *testing.T) {
	data := tikTokOrderCSV(t, map[string]string{"Order ID": "001234567890123456789", "Created Time": "2026-04-01 12:30:00", "Paid Time": "2026-07-01", "Normal or Pre-order": "Pre-order", "Delivery Option Type": "TikTok 物流", "Delivery Option": "标准运输", "Virtual Bundle Seller SKU": "bundle-01", "Phone #": "0012345678", "Zipcode": "00123", "SKU ID": "0000123"})
	rows, err := parseTikTokOrderFile(data, ".csv")
	if err != nil || len(rows) != 1 || rows[0].Month != "2026-04" || rows[0].Quarter != "2026-Q2" || rows[0].OrderID != "001234567890123456789" || rows[0].Data["Phone #"] != "0012345678" || rows[0].Data["Virtual Bundle Seller SKU"] != "bundle-01" {
		t.Fatalf("parse: %+v %v", rows, err)
	}
	if len(tikTokOrderColumns) != 63 {
		t.Fatal("incorrect column count")
	}
	if _, err := parseTikTokOrderFile(bytes.Replace(data, []byte("2026-04-01 12:30:00"), []byte("bad-date"), 1), ".csv"); err == nil {
		t.Fatal("invalid Created Time accepted")
	}
	for _, date1904 := range []bool{false, true} {
		book := excelize.NewFile()
		book.SetWorkbookProps(&excelize.WorkbookPropsOptions{Date1904: &date1904})
		cells, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		for i, row := range cells {
			values := make([]interface{}, len(row))
			for j, value := range row {
				values[j] = value
			}
			if i == 1 {
				serial := 46024.5
				if date1904 {
					serial -= 1462
				}
				values[28] = serial
			}
			cell, _ := excelize.CoordinatesToCellName(1, i+1)
			book.SetSheetRow("Sheet1", cell, &values)
		}
		book.NewSheet("ignored")
		book.SetCellValue("ignored", "A1", "unrelated")
		buf, err := book.WriteToBuffer()
		book.Close()
		if err != nil {
			t.Fatal(err)
		}
		rows, err := parseTikTokOrderFile(buf.Bytes(), ".xlsx")
		if err != nil || len(rows) != 1 || rows[0].Month != "2026-01" || rows[0].Data["Created Time"] != "2026-01-02 12:00:00" {
			t.Fatalf("Excel first sheet: %+v %v", rows, err)
		}
	}
}
func TestTikTokOrderSettlementIntegration(t *testing.T) {
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
	sqlDB, _ := root.DB()
	defer sqlDB.Close()
	tx := root.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	old := database.DB
	database.DB = tx
	defer func() { database.DB = old }()
	schema := fmt.Sprintf("tiktok_order_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.TikTokOrder{}, &model.TikTokOrderBatch{}, &model.TikTokSettlement{}, &model.TikTokSettlementBatch{}, &model.TaxImportLog{}, &model.TaxDownloadLog{}, &model.TaxExchangeRate{}, &model.User{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "order-test", Name: "管理员"}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/upload", UploadTikTokOrders)
	r.POST("/settlement-upload", UploadTikTokSettlements)
	group := r.Group("/test")
	group.Use(func(c *gin.Context) { c.Set("userID", user.ID); c.Next() })
	group.GET("/orders", ListTikTokOrders)
	group.GET("/settlements", ListTikTokSettlements)
	group.GET("/options", ListTikTokTaxFilterOptions)
	group.GET("/export", ExportTikTokTaxReport)
	group.GET("/imports/:id/file", DownloadTaxImport)
	get := func(path string) []byte {
		out := httptest.NewRecorder()
		r.ServeHTTP(out, httptest.NewRequest("GET", path, nil))
		if out.Code != 200 {
			t.Fatalf("%s: %d %s", path, out.Code, out.Body.String())
		}
		return out.Body.Bytes()
	}
	upload := func(path string, data []byte, key string, want int) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		part, _ := w.CreateFormFile("file", "orders.csv")
		part.Write(data)
		w.Close()
		req := httptest.NewRequest("POST", path, &buf)
		req.Header.Set("X-RPA-Key", key)
		req.Header.Set("Content-Type", w.FormDataContentType())
		out := httptest.NewRecorder()
		r.ServeHTTP(out, req)
		if out.Code != want {
			t.Fatalf("upload: %d %s", out.Code, out.Body.String())
		}
	}
	// Import a settlement first; later order imports must update both the list and export.
	statement := settlementCSV(t, "2026-01-02")
	statement = bytes.ReplaceAll(statement, []byte("001234567890123456789"), []byte("0001"))
	upload("/settlement-upload", statement, taxDataRPAKey, 201)
	readSettlements := func() []tikTokSettlementResponse {
		var result struct {
			Items []tikTokSettlementResponse `json:"items"`
		}
		if err := json.Unmarshal(get("/test/settlements"), &result); err != nil {
			t.Fatal(err)
		}
		return result.Items
	}
	if items := readSettlements(); len(items) != 1 || items[0].LogisticsMethod != "" || items[0].OrderType != "" {
		t.Fatalf("unmatched settlement: %+v", items)
	}
	order := func(id, kind, option, orderType string) map[string]string {
		return map[string]string{"Order ID": id, "Created Time": "2026-01-02", "Normal or Pre-order": orderType, "Delivery Option Type": kind, "Delivery Option": option, "SKU ID": "sku-one"}
	}
	data := tikTokOrderCSV(t, order("0001", "商家物流", "标准运输", "Normal"), order("0002", "TikTok 物流", "标准运输", "Pre-order"), order("0003", "TikTok 物流", "Fulfilled by TikTok Shop 标准", "Normal"), order("0004", "商家物流", "其他运输", "Normal"), order("", "商家物流", "标准运输", "Normal"))
	upload("/upload", data, "wrong", 401)
	upload("/upload", data, taxDataRPAKey, 201)
	upload("/upload", data, taxDataRPAKey, 200)
	var count int64
	tx.Model(&model.TikTokOrder{}).Count(&count)
	if count != 5 {
		t.Fatalf("duplicate rows: %d", count)
	}
	if items := readSettlements(); len(items) != 1 || items[0].LogisticsMethod != "MCF" || items[0].OrderType != "Normal" {
		t.Fatalf("existing settlement not matched: %+v", items)
	}
	for _, id := range []string{"0002", "0003", "0004", "", "missing"} {
		file := bytes.ReplaceAll(statement, []byte("0001"), []byte(id))
		upload("/settlement-upload", file, taxDataRPAKey, 201)
	}
	want := []string{"MCF", "海外仓", "FBT", "MCF", "", ""}
	items := readSettlements()
	if len(items) != len(want) {
		t.Fatalf("unexpected settlement count %d", len(items))
	}
	for i, item := range items {
		if item.LogisticsMethod != want[i] {
			t.Fatalf("rule %d: %+v", i, item)
		}
	}
	// Different SKUs under one order are retained; the same pair is skipped.
	updated := order("0001", "TikTok 物流", "Fulfilled by TikTok Shop 标准", "Pre-order")
	updated["SKU ID"] = "sku-two"
	upload("/upload", tikTokOrderCSV(t, updated), taxDataRPAKey, 201)
	items = readSettlements()
	if len(items) != 6 || items[0].LogisticsMethod != "FBT" || items[0].OrderType != "Pre-order" {
		t.Fatalf("latest match or fan-out: %+v", items)
	}
	records, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(get("/test/export?type=settlements"), []byte{0xef, 0xbb, 0xbf}))).ReadAll()
	if err != nil || len(records) != 7 || records[1][2] != "FBT" || records[1][3] != "Pre-order" {
		t.Fatalf("settlement export: %v %v", records, err)
	}
	// Existing non-order settlements never inherit logistics or order type from matching IDs.
	for _, kind := range []string{"Adjustment", "Refund", "", "order", " Order "} {
		if err := tx.Create(&model.TikTokSettlement{Month: "2026-01", Quarter: "2026-Q1", StatementDate: "2026-01-02", Currency: "USD", Data: map[string]string{"Type": kind, "Order/adjustment ID": "0001"}}).Error; err != nil {
			t.Fatal(err)
		}
	}
	// A missing Type follows the same non-Order fallback.
	if err := tx.Create(&model.TikTokSettlement{Month: "2026-01", Quarter: "2026-Q1", StatementDate: "2026-01-02", Currency: "USD", Data: map[string]string{"Order/adjustment ID": "0001"}}).Error; err != nil {
		t.Fatal(err)
	}
	items = readSettlements()
	if len(items) != 12 {
		t.Fatalf("settlement count: %d", len(items))
	}
	for i, item := range items[6:] {
		logistics, orderType := "杂费", "Other"
		if i == 4 {
			logistics, orderType = "FBT", "Pre-order"
		}
		if item.LogisticsMethod != logistics || item.OrderType != orderType {
			t.Fatalf("non-Order rule %d: %+v", i, item)
		}
	}
	records, err = csv.NewReader(bytes.NewReader(bytes.TrimPrefix(get("/test/export?type=settlements"), []byte{0xef, 0xbb, 0xbf}))).ReadAll()
	if err != nil || len(records) != 13 {
		t.Fatalf("settlement export: %v %v", records, err)
	}
	for i, item := range items[6:] {
		if records[i+7][2] != item.LogisticsMethod || records[i+7][3] != item.OrderType {
			t.Fatalf("export differs from list: %v", records[i+7])
		}
	}
	later := order("0005", "商家物流", "标准运输", "Normal")
	later["Created Time"] = "2026-04-01"
	upload("/upload", tikTokOrderCSV(t, later, later, updated), taxDataRPAKey, 201)
	var dedupAudit model.TaxImportLog
	if err := tx.Where("report_type = 'orders'").Order("id DESC").First(&dedupAudit).Error; err != nil || dedupAudit.SourceRows != 3 || dedupAudit.ImportedRows != 1 || dedupAudit.SkippedRows != 2 {
		t.Fatalf("order-ID dedup audit: %+v %v", dedupAudit, err)
	}
	var list struct {
		Items   []model.TikTokOrder      `json:"items"`
		Total   int64                    `json:"total"`
		Columns []tikTokSettlementColumn `json:"columns"`
	}
	if err := json.Unmarshal(get("/test/orders?month=2026-01&quarter=2026-Q1&page=2&page_size=2"), &list); err != nil || list.Total != 6 || len(list.Items) != 2 || len(list.Columns) != 63 || list.Items[0].OrderID != "0003" {
		t.Fatalf("order pagination: %+v %v", list, err)
	}
	options := string(get("/test/options?type=orders&field=month&quarter=2026-Q1"))
	if !strings.Contains(options, "2026-01") || strings.Contains(options, "2026-04") {
		t.Fatal(options)
	}
	records, err = csv.NewReader(bytes.NewReader(bytes.TrimPrefix(get("/test/export?type=orders&quarter=2026-Q1&page_size=1"), []byte{0xef, 0xbb, 0xbf}))).ReadAll()
	if err != nil || len(records) != 7 || len(records[0]) != 65 || records[0][2] != "Order ID(订单ID)" {
		t.Fatalf("order export: %v %v", records, err)
	}
	var audit model.TaxImportLog
	if err := tx.Where("report_type = 'orders' AND status = 'locked'").Order("id").First(&audit).Error; err != nil || audit.Importer != "RPA" || audit.ImportedRows != 5 {
		t.Fatalf("audit %+v %v", audit, err)
	}
	if !bytes.Equal(get(fmt.Sprintf("/test/imports/%d/file?platform=tiktok", audit.ID)), data) {
		t.Fatal("original file changed")
	}
	var downloads int64
	tx.Model(&model.TaxDownloadLog{}).Where("report_type = 'orders' AND platform = 'tiktok'").Count(&downloads)
	if downloads != 1 {
		t.Fatal("missing order download log")
	}
	for _, test := range []struct {
		query string
		total int64
		count int
	}{
		{"month=2026-01,2026-04&quarter=2026-Q1,2026-Q2&page_size=2&page=4", 7, 1},
		{"month=2026-01,2026-04&quarter=2026-Q2", 1, 1},
		{"month=2026-01,2026-04&quarter=2026-Q3", 0, 0},
	} {
		if err := json.Unmarshal(get("/test/orders?"+test.query), &list); err != nil || list.Total != test.total || len(list.Items) != test.count {
			t.Fatalf("multi-period pagination: %+v %v", list, err)
		}
	}
	options = string(get("/test/options?type=orders&field=month&quarter=2026-Q1,2026-Q2"))
	if !strings.Contains(options, "2026-01") || !strings.Contains(options, "2026-04") {
		t.Fatal(options)
	}
	records, err = csv.NewReader(bytes.NewReader(bytes.TrimPrefix(get("/test/export?type=orders&month=2026-01,2026-04&quarter=2026-Q1,2026-Q2&page_size=1"), []byte{0xef, 0xbb, 0xbf}))).ReadAll()
	if err != nil || len(records) != 8 {
		t.Fatalf("multi-period export: %v %v", records, err)
	}
}
