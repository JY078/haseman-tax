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
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTikTokSettlementCombinedFilters(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=test dbname=test sslmode=disable"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	query := func() *gorm.DB {
		return filteredTikTokSettlements(db, "2026-01,2026-02", "2026-Q1", "MCF,FBT", "Normal,__empty__")
	}
	var rows []tikTokSettlementResponse
	list := query().Order("statement_date, id").Limit(25).Find(&rows).Statement
	var count int64
	countQuery := query().Count(&count).Statement
	for _, statement := range []*gorm.Statement{list, countQuery} {
		sql := statement.SQL.String()
		for _, fragment := range []string{"month IN", "quarter IN", "logistics_method IN", "order_type IN", "LEFT JOIN LATERAL"} {
			if !strings.Contains(sql, fragment) {
				t.Fatalf("missing %s in %s", fragment, sql)
			}
		}
		for _, expected := range []string{"2026-01", "2026-02", "2026-Q1", "MCF", "FBT", "Normal", ""} {
			found := false
			for _, value := range statement.Vars {
				if value == expected {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing filter parameter %q: %#v", expected, statement.Vars)
			}
		}
	}
}

func settlementCSV(t *testing.T, dates ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	headers := []string{}
	for _, column := range tikTokSettlementColumns {
		headers = append(headers, column.Key)
	}
	if len(headers) != 81 {
		t.Fatalf("expected 81 columns, got %d", len(headers))
	}
	w.Write(headers)
	for i, date := range dates {
		row := make([]string, len(headers))
		values := map[string]string{"Statement Date": date, "Statement ID": fmt.Sprintf("ST%d", i), "Currency": "USD", "Type": "Order", "Order/adjustment ID": "001234567890123456789", "SKU ID": "000123", "Total settlement amount": "44.58", "Product name": "商品,一", "Quantity": "1", "Smart Promotion campaign period fee sales tax": "0.37"}
		for j, key := range headers {
			row[j] = values[key]
		}
		w.Write(row)
	}
	w.Flush()
	if w.Error() != nil {
		t.Fatal(w.Error())
	}
	return buf.Bytes()
}
func TestTikTokSettlementParser(t *testing.T) {
	data := settlementCSV(t, "2026-01-02", "2026-04-01")
	rows, err := parseTikTokSettlementFile(data, ".csv")
	if err != nil || len(rows) != 2 {
		t.Fatalf("parse: %v %v", rows, err)
	}
	if rows[1].Quarter != "2026-Q2" || rows[0].Month != "2026-01" || rows[0].Data["Order/adjustment ID"] != "001234567890123456789" || rows[0].Data["Smart Promotion campaign period fee sales tax"] != "0.37" || rows[0].Data["Total Fees"] != "" || rows[0].Amount.String() != "44.58" {
		t.Fatalf("values lost: %+v", rows)
	}
	for _, data := range [][]byte{settlementCSV(t, "invalid"), []byte("Statement Date\n2026-01-01"), bytes.Replace(settlementCSV(t, "2026-01-02"), []byte("44.58"), []byte("bad"), 1)} {
		if _, err := parseTikTokSettlementFile(data, ".csv"); err == nil {
			t.Fatal("invalid input accepted")
		}
	}
	if _, err := parseTikTokSettlementFile(data, ".pdf"); err == nil {
		t.Fatal("unsupported extension accepted")
	}
	// The first sheet contains data; the second must never be imported.
	for _, date1904 := range []bool{false, true} {
		book := excelize.NewFile()
		if err := book.SetWorkbookProps(&excelize.WorkbookPropsOptions{Date1904: &date1904}); err != nil {
			t.Fatal(err)
		}
		input, _ := csv.NewReader(bytes.NewReader(settlementCSV(t, "2026-01-02"))).ReadAll()
		book.NewSheet("Order detail 1")
		for i, values := range input {
			v := make([]interface{}, len(values))
			for j, x := range values {
				v[j] = x
			}
			if i == 1 {
				serial := 46024.
				if date1904 {
					serial -= 1462
				}
				v[0] = serial
			}
			cell, _ := excelize.CoordinatesToCellName(1, i+1)
			if err := book.SetSheetRow("Order detail 1", cell, &v); err != nil {
				t.Fatal(err)
			}
		}
		book.NewSheet("不导入")
		book.SetCellValue("不导入", "A1", "错误表头")
		buf, err := book.WriteToBuffer()
		book.Close()
		if err != nil {
			t.Fatal(err)
		}
		got, err := parseTikTokSettlementFile(buf.Bytes(), ".xlsx")
		if err != nil || len(got) != 1 || got[0].StatementDate != "2026-01-02" || got[0].Data["SKU ID"] != "000123" {
			t.Fatalf("first sheet/date system: %+v, %v", got, err)
		}
	}
}

func TestTikTokSettlementDifferentWorksheetHeaders(t *testing.T) {
	book := excelize.NewFile()
	defer book.Close()
	sheets := []struct {
		name string
		rows [][]string
	}{
		{"Order details", [][]string{
			{"Statement Date", "Statement ID", "Currency", "Type", "Order/adjustment ID", "SKU ID", "Total settlement amount", "Total Revenue", "Referral fee", "Affiliate Commission", "FBT fulfillment fee", "New Customer Growth Package fee", "Seller shipping fee"},
			{"2025-05-01", "ST1", "USD", "Order", "001", "sku1", "57.19", "60", "-3.6", "0", "-1.25", "2.5", "-4.25"},
		}},
		{"Order details (bef 2025.04.30)", [][]string{
			{"Statement Date", "Statement ID", "Currency", "Type", "Order/adjustment ID", "Total Settlement amount", "Total revenue", "Customer payment", "Sales tax payment", "Referral fee", "Affiliate commission", "Sales tax payment", "FBT Fulfillment fee", "Related order ID"},
			{"2025-04-30", "ST2", "USD", "Order", "002", "90.37", "112.38", "124.82", "12.44", "-6.63", "-14.38", "-12.44", "-2.75", "009"},
		}},
		{"Order detail 3", [][]string{
			{"Currency", "Total settlement amount", "Order/adjustment ID", "Type", "Statement ID", "Statement Date", "Affiliate partner commission", "Shipping Fee"},
			{"USD", "41.29", "003", "Order", "ST3", "2025-05-02", "-6.5", "-8.75"},
		}},
	}
	for _, sheet := range sheets {
		if _, err := book.NewSheet(sheet.name); err != nil {
			t.Fatal(err)
		}
		for index, values := range sheet.rows {
			cell, _ := excelize.CoordinatesToCellName(1, index+1)
			if err := book.SetSheetRow(sheet.name, cell, &values); err != nil {
				t.Fatal(err)
			}
		}
	}
	buffer, err := book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseTikTokSettlementFile(buffer.Bytes(), ".xlsx")
	if err != nil || len(rows) != 3 {
		t.Fatalf("parse mixed worksheet versions: %+v %v", rows, err)
	}
	if rows[1].Amount.String() != "90.37" || rows[1].Month != "2025-04" || rows[1].Quarter != "2025-Q2" {
		t.Fatalf("old worksheet amount/date: %+v", rows[1])
	}
	want := map[string]string{"Total Revenue": "112.38", "Referral fee": "-6.63", "Affiliate Commission": "-14.38", "FBT fulfillment fee": "-2.75", "SKU ID": "", "New Customer Growth Package fee": "", "Customer payment": "124.82", "Sales tax payment": "12.44", "Sales tax payment (2)": "-12.44", "Related order ID": "009"}
	for key, value := range want {
		if rows[1].Data[key] != value {
			t.Errorf("old worksheet %s: got %q want %q", key, rows[1].Data[key], value)
		}
	}
	if rows[0].Data["New Customer Growth Package fee"] != "2.5" || rows[0].Data["Customer payment"] != "" || rows[2].Amount.String() != "41.29" || rows[2].Data["Affiliate partner commission"] != "-6.5" || rows[2].Data["Referral fee"] != "" {
		t.Fatalf("sheet-specific columns must not shift or bleed: %+v", rows)
	}
	if fee, present := rows[2].Data["FBT fulfillment fee"]; !present || fee != "" || rows[2].Data["Shipping fee"] != "-8.75" || rows[2].Data["Seller shipping fee"] != "-8.75" {
		t.Fatalf("missing FBT column must stay blank; shipping header must map by name: %+v", rows[2].Data)
	}
	if rows[0].Data["Shipping fee"] != "-4.25" || rows[0].Data["Seller shipping fee"] != "-4.25" {
		t.Fatalf("Seller shipping fee header must populate the displayed shipping fee: %+v", rows[0].Data)
	}
	var merged [][]string
	if err := mergeTikTokSettlementSheet(&merged, sheets[0].rows); err != nil {
		t.Fatal(err)
	}
	if err := mergeTikTokSettlementSheet(&merged, [][]string{{"Statement Date", "Referral fee"}, {"2025-05-01", "-1"}}); err == nil {
		t.Fatal("missing required headers in later sheet accepted")
	}
}

func TestTikTokSettlementIntegration(t *testing.T) {
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
	oldDB := database.DB
	database.DB = tx
	defer func() { database.DB = oldDB }()
	schema := fmt.Sprintf("tiktok_settlement_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.TikTokSettlement{}, &model.TikTokSettlementBatch{}, &model.TikTokOrder{}, &model.TaxImportLog{}, &model.TaxDownloadLog{}, &model.User{}, &model.TaxExchangeRate{}, &model.TaxRawRow{}, &model.TikTokTaxAd{}, &model.TikTokTaxDetail{}, &model.AmazonTaxShipment{}, &model.AmazonTaxFinance{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "settlement-test", Name: "管理员"}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/upload", UploadTikTokSettlements)
	group := r.Group("/test")
	group.Use(func(c *gin.Context) { c.Set("userID", user.ID); c.Next() })
	group.GET("/list", ListTikTokSettlements)
	group.GET("/options", ListTikTokTaxFilterOptions)
	group.GET("/export", ExportTikTokTaxReport)
	group.GET("/imports", ListTaxImports)
	group.GET("/imports/:id/file", DownloadTaxImport)
	group.GET("/downloads/:id/file", DownloadTaxHistoryFile)
	get := func(path string) *httptest.ResponseRecorder {
		out := httptest.NewRecorder()
		r.ServeHTTP(out, httptest.NewRequest("GET", path, nil))
		if out.Code != 200 {
			t.Fatalf("%s: %d %s", path, out.Code, out.Body.String())
		}
		return out
	}
	upload := func(content []byte, key string, want int) string {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		part, _ := w.CreateFormFile("file", "结算单.csv")
		part.Write(content)
		w.Close()
		req := httptest.NewRequest("POST", "/upload", &buf)
		req.Header.Set("X-RPA-Key", key)
		req.Header.Set("Content-Type", w.FormDataContentType())
		out := httptest.NewRecorder()
		r.ServeHTTP(out, req)
		var result struct {
			Status bool   `json:"status"`
			Msg    string `json:"msg"`
		}
		if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil || out.Code != want || result.Status != (want < 400) || result.Msg == "" {
			t.Fatalf("upload: %d %s", out.Code, out.Body.String())
		}
		return result.Msg
	}
	data := settlementCSV(t, "2025-01-02", "2025-01-03", "2025-02-01", "2025-04-01")
	upload(data, "invalid", 401)
	upload(data, taxDataRPAKey, 201)
	upload(data, taxDataRPAKey, 200)
	var count int64
	tx.Model(&model.TikTokSettlement{}).Count(&count)
	if count != 4 {
		t.Fatalf("duplicate import count %d", count)
	}
	// Reimporting the same original file repairs historical positional mapping without new rows.
	var original model.TikTokSettlement
	if err := tx.Where("statement_date = ?", "2025-01-02").First(&original).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Model(&original).Updates(map[string]interface{}{
		"amount": 999, "data": gorm.Expr("jsonb_set(data, '{Total settlement amount}', '\"999\"'::jsonb)"),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if msg := upload(data, taxDataRPAKey, 200); !strings.Contains(msg, "已修正 1 条字段映射") {
		t.Fatalf("mapping repair response: %s", msg)
	}
	var corrected model.TikTokSettlement
	if err := tx.First(&corrected, original.ID).Error; err != nil || corrected.Amount.String() != "44.58" || corrected.Data["Total settlement amount"] != "44.58" {
		t.Fatalf("mapping repair: %+v %v", corrected, err)
	}
	if msg := upload(data, taxDataRPAKey, 200); !strings.Contains(msg, "已跳过重复文件") {
		t.Fatalf("unchanged reimport: %s", msg)
	}
	tx.Model(&model.TikTokSettlement{}).Count(&count)
	if count != 4 {
		t.Fatalf("repair created duplicate rows: %d", count)
	}
	var audit model.TaxImportLog
	if err := tx.Where("platform = 'tiktok' AND report_type = 'settlements' AND status = 'locked'").First(&audit).Error; err != nil || audit.Importer != "RPA" || audit.ImportedRows != 4 || !bytes.Equal(audit.FileContent, data) {
		t.Fatalf("audit: %+v %v", audit, err)
	}
	upload(settlementCSV(t, "invalid-date"), taxDataRPAKey, 400)
	type response struct {
		Items   []tikTokSettlementResponse `json:"items"`
		Total   int64                      `json:"total"`
		Columns []tikTokSettlementColumn   `json:"columns"`
	}
	read := func(path string) response {
		var result response
		if err := json.Unmarshal(get(path).Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	result := read("/test/list?month=2025-01&quarter=2025-Q1&page=2&page_size=1")
	if result.Total != 2 || len(result.Items) != 1 || result.Items[0].StatementDate != "2025-01-03" || result.Items[0].Fee != nil || len(result.Columns) != 47 || result.Items[0].Data["Order/adjustment ID"] != "001234567890123456789" {
		t.Fatalf("pending/pagination: %+v", result)
	}
	options := get("/test/options?type=settlements&field=month&month=2025-01&quarter=2025-Q1")
	if !strings.Contains(options.Body.String(), "2025-02") || strings.Contains(options.Body.String(), "2025-04") {
		t.Fatalf("all filtered options: %s", options.Body.String())
	}
	if result := read("/test/list?month=2025-01&quarter=2025-Q2"); result.Total != 0 || len(result.Items) != 0 {
		t.Fatal("AND filters ignored")
	}
	rate := model.TaxExchangeRate{Date: "2025-01-03", Currency: "USD", Rate: decimal.RequireFromString("7.1457")}
	if err := tx.Create(&rate).Error; err != nil {
		t.Fatal(err)
	}
	result = read("/test/list?month=2025-01&quarter=2025-Q1&page=2&page_size=1")
	if result.Items[0].Fee == nil || result.Items[0].Fee.StringFixed(2) != "318.56" {
		t.Fatalf("daily fee: %+v", result.Items[0])
	}
	if result.Items[0].LogisticsMethod != "" || result.Items[0].OrderType != "" {
		t.Fatal("classification must remain blank")
	}
	out := get("/test/export?type=settlements&month=2025-01&quarter=2025-Q1&page=2&page_size=1")
	csvRows, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(out.Body.Bytes(), []byte{0xef, 0xbb, 0xbf}))).ReadAll()
	if err != nil || len(csvRows) != 3 || len(csvRows[0]) != 52 || csvRows[0][5] != "Statement Date(声明日期)" || csvRows[2][4] != "318.56" || csvRows[1][4] != "318.56" {
		t.Fatalf("export %v %v", csvRows, err)
	}
	var download model.TaxDownloadLog
	if err := tx.Where("report_type = 'settlements'").First(&download).Error; err != nil || download.RowCount != 2 || download.Username != "管理员" {
		t.Fatalf("download log %+v %v", download, err)
	}
	if !bytes.Equal(get(fmt.Sprintf("/test/downloads/%d/file?platform=tiktok", download.ID)).Body.Bytes(), out.Body.Bytes()) {
		t.Fatal("export snapshot changed")
	}
	if !bytes.Equal(get(fmt.Sprintf("/test/imports/%d/file?platform=tiktok", audit.ID)).Body.Bytes(), data) {
		t.Fatal("original download changed")
	}
	earliest, err := earliestTaxExchangeDate(tx)
	if err != nil || earliest.Format("2006-01-02") != "2025-01-02" {
		t.Fatalf("exchange date range: %v %v", earliest, err)
	}
	// Different files and repeated rows use the five business columns, including SKU and amount.
	source, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	columns := map[string]int{}
	for i, key := range source[0] {
		columns[key] = i
	}
	changed := func(values map[string]string) []string {
		row := append([]string(nil), source[1]...)
		for key, value := range values {
			row[columns[key]] = value
		}
		return row
	}
	var next bytes.Buffer
	writer := csv.NewWriter(&next)
	writer.WriteAll([][]string{source[0],
		changed(map[string]string{"Total settlement amount": "999", "SKU ID": "different"}),
		changed(map[string]string{"Statement ID": " ST0 ", "Order/adjustment ID": " 001234567890123456789 "}),
		changed(map[string]string{"Statement Date": "2025-01-04"}),
		changed(map[string]string{"Statement ID": "ST-new"}),
		changed(map[string]string{"Order/adjustment ID": "000000000000000000002"}),
		changed(map[string]string{"Statement ID": "", "Order/adjustment ID": ""}),
		changed(map[string]string{"Statement ID": "", "Order/adjustment ID": "", "Total settlement amount": "123"}),
	})
	if writer.Error() != nil {
		t.Fatal(writer.Error())
	}
	msg := upload(next.Bytes(), taxDataRPAKey, 201)
	if !strings.Contains(msg, "入库 6 条，跳过重复 1 条") {
		t.Fatal(msg)
	}
	tx.Model(&model.TikTokSettlement{}).Count(&count)
	if count != 10 {
		t.Fatalf("business-key dedup count: %d", count)
	}
	var latest model.TaxImportLog
	if err := tx.Where("report_type = 'settlements'").Order("id DESC").First(&latest).Error; err != nil || latest.SourceRows != 7 || latest.ImportedRows != 6 || latest.SkippedRows != 1 {
		t.Fatalf("dedup log: %+v %v", latest, err)
	}
	var first model.TikTokSettlement
	if err := tx.Order("id").First(&first).Error; err != nil || first.Amount.String() != "44.58" {
		t.Fatalf("existing amount overwritten: %+v %v", first, err)
	}
	upload(next.Bytes(), taxDataRPAKey, 200)
	// Changed bytes still cannot reimport identities already present in the database.
	msg = upload(bytes.ReplaceAll(next.Bytes(), []byte("different"), []byte("another-sku")), taxDataRPAKey, 201)
	if !strings.Contains(msg, "入库 1 条，跳过重复 6 条") {
		t.Fatal(msg)
	}
	tx.Model(&model.TikTokSettlement{}).Count(&count)
	if count != 11 {
		t.Fatalf("repeated calls created rows: %d", count)
	}
	// Simulate a SKU skipped by the former three-column key; the same file must backfill it.
	if err := tx.Where("data->>'SKU ID' = ?", "different").Delete(&model.TikTokSettlement{}).Error; err != nil {
		t.Fatal(err)
	}
	if msg := upload(next.Bytes(), taxDataRPAKey, 200); !strings.Contains(msg, "已补充入库 1 条") {
		t.Fatalf("old file SKU backfill: %s", msg)
	}
	if msg := upload(next.Bytes(), taxDataRPAKey, 200); !strings.Contains(msg, "已跳过重复文件") {
		t.Fatalf("backfill is not idempotent: %s", msg)
	}
	// Blank SKU is a value in the composite key, not a reason to disable deduplication.
	var blank bytes.Buffer
	blankWriter := csv.NewWriter(&blank)
	blankWriter.WriteAll([][]string{source[0],
		changed(map[string]string{"SKU ID": "", "Total settlement amount": "12"}),
		changed(map[string]string{"SKU ID": "  ", "Total settlement amount": "12.00"}),
		changed(map[string]string{"SKU ID": "  ", "Total settlement amount": "99"}),
		changed(map[string]string{"SKU ID": "000123", "Total settlement amount": "99"}),
	})
	if err := blankWriter.Error(); err != nil {
		t.Fatal(err)
	}
	if msg := upload(blank.Bytes(), taxDataRPAKey, 201); !strings.Contains(msg, "入库 3 条，跳过重复 1 条") {
		t.Fatalf("blank SKU dedup: %s", msg)
	}
	if msg := upload(blank.Bytes(), taxDataRPAKey, 200); !strings.Contains(msg, "已跳过重复文件") {
		t.Fatalf("same-file amounts changed during repair: %s", msg)
	}
	// Changed amounts are new identities; equivalent numeric formats remain duplicates.
	if msg := upload(bytes.ReplaceAll(blank.Bytes(), []byte("99"), []byte("98")), taxDataRPAKey, 201); !strings.Contains(msg, "入库 2 条，跳过重复 2 条") {
		t.Fatalf("cross-file blank SKU dedup: %s", msg)
	}
	tx.Model(&model.TikTokSettlement{}).Count(&count)
	if count != 16 {
		t.Fatalf("final dedup count: %d", count)
	}

}
