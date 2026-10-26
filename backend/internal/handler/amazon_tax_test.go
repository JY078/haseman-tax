package handler

import (
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAmazonFinanceStartsBelowActualHeader(t *testing.T) {
	// The first nine rows mirror the explanatory preamble in Amazon reports.
	lines := [][]string{
		{"Includes Amazon Marketplace, Fulfillment by Amazon (FBA), and Amazon Webstore transactions"},
		{"All amounts in USD, unless specified"},
		{"Definitions:"},
		{"date/time: posted date/time of the transaction"},
		{"Sales tax collected: Includes sales tax collected from buyers for product sales, shipping, and giftwrap."},
		{"Selling fees: Includes variable closing fees and referral fees."},
		{"Other transaction fees: Includes shipping chargebacks, shipping holdbacks, per item fees and sales tax collection fees."},
		{"Other: Includes non-order transaction amounts. See the \"Type\" and \"Description\" columns for each order ID."},
		{"Note: Reports with start date of Jan 1, 2025 or later are indexed by posted date."},
		{"date/time", "settlement id", "type", "order id", "sku", "description", "quantity", "marketplace", "total"},
		{"Jul 1, 2026 1:22:58 AM PDT", "26874188991", "Order", "111-4834605-6157825", "FP001-26", "26 inch Fire Pit", "1", "amazon.com", "100.25"},
		{"Jul 1, 2026 3:28:42 AM PDT", "26874188991", "Order", "114-6795672-6645843", "D01-P-75", "Enamel Cast Iron Dutch Oven", "1", "amazon.com", "45.60"},
	}
	var csvFile bytes.Buffer
	writer := csv.NewWriter(&csvFile)
	if err := writer.WriteAll(lines); err != nil {
		t.Fatal(err)
	}
	book := excelize.NewFile()
	defer book.Close()
	for i, line := range lines {
		cell, _ := excelize.CoordinatesToCellName(1, i+1)
		if err := book.SetSheetRow("Sheet1", cell, &line); err != nil {
			t.Fatal(err)
		}
	}
	xlsxFile, err := book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	for ext, content := range map[string][]byte{".csv": csvFile.Bytes(), ".xlsx": xlsxFile.Bytes()} {
		t.Run(ext, func(t *testing.T) {
			rows, err := parseAmazonTaxFile(content, ext, "finance")
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 2 {
				t.Fatalf("preamble/header imported: got %d rows", len(rows))
			}
			for i, row := range rows {
				if row.Month != "2026-07" || row.Quarter != "2026-Q3" || row.Data["order id"] != lines[i+10][3] || row.Data["total"] != lines[i+10][8] {
					t.Fatalf("incorrect data below header: %+v", row)
				}
			}
		})
	}
}

func TestAmazonTaxDateAcceptsLocalizedMeridiem(t *testing.T) {
	for _, value := range []string{
		"Jul 1, 2026 9:17:27 a.m. PDT",
		"Jul 1, 2026 9:17:27 p.m. PDT",
	} {
		if parsed := parseAmazonTaxDate(value); parsed.IsZero() {
			t.Fatalf("localized meridiem should parse: %q", value)
		}
	}
}

func TestAmazonImportMapsNamesAndIdentity(t *testing.T) {
	content := []byte("商品价格,货件编号,配送日期,亚马逊订单编号,货件商品编号\n12.34,S1,2026-07-01,O1,I1\n56,S1,2026-07-02,O1,I1\n78,S1,2026-08-01,O1,I2\n")
	rows, err := parseAmazonTaxFile(content, ".csv", "shipments")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Data["商品价格"] != "12.34" || rows[0].Data["卖家订单编号"] != "" || rows[0].Month != "2026-07" || rows[0].Quarter != "2026-Q3" {
		t.Fatalf("incorrect mapping: %+v", rows)
	}
	if rows[0].Fingerprint != rows[1].Fingerprint || rows[0].Fingerprint == rows[2].Fingerprint {
		t.Fatal("shipment identity must use order, shipment and shipment item IDs")
	}
}

func TestAmazonFinanceNamesAndIdentity(t *testing.T) {
	content := []byte("report metadata\norder id,total,date/time,settlement id,sku\nO1,100,2026-07-01,S1,SKU1\nO1,200,2026-07-01,S1,SKU1\nO1,200,2026-07-02,S1,SKU1\nO1,100,2026-07-01,S1,SKU2\nO1,100,2026-07-01,S2,SKU1\nO2,100,2026-07-01,S1,SKU1\nO1,100,2026-07-01,S1,SKU1\nO1,101,2026-07-01,S1,SKU1\n")
	rows, err := parseAmazonTaxFile(content, ".csv", "finance")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 8 || rows[0].Data["total"] != "100" || rows[0].Data["sku"] != "SKU1" {
		t.Fatalf("incorrect mapping: %+v", rows[0])
	}
	if rows[0].Fingerprint == rows[1].Fingerprint {
		t.Fatal("different total values must not be treated as duplicate finance rows")
	}
	if rows[0].Fingerprint != rows[6].Fingerprint {
		t.Fatal("same date, settlement ID, order ID, SKU and total must deduplicate")
	}
	for _, row := range append(rows[2:6], rows[7]) {
		if rows[0].Fingerprint == row.Fingerprint {
			t.Fatal("finance identity must include date, settlement ID, order ID, SKU and total")
		}
	}
}

func TestAmazonRejectsAmbiguousHeader(t *testing.T) {
	_, err := parseAmazonTaxFile([]byte("date/time,settlement id,order id,total,total\n2026-07-01,S1,O1,1,2\n"), ".csv", "finance")
	if err == nil {
		t.Fatal("duplicate headers must not be silently imported")
	}
}

func TestAmazonXLSXHeaderMapping(t *testing.T) {
	book := excelize.NewFile()
	defer book.Close()
	for rowIndex, values := range [][]interface{}{{"total", "order id", "settlement id", "date/time"}, {123.45, "O1", "S1", 46024}} {
		cell, _ := excelize.CoordinatesToCellName(1, rowIndex+1)
		if err := book.SetSheetRow("Sheet1", cell, &values); err != nil {
			t.Fatal(err)
		}
	}
	buf, err := book.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseAmazonTaxFile(buf.Bytes(), ".xlsx", "finance")
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Data["total"] != "123.45" || rows[0].Month != "2026-01" {
		t.Fatalf("incorrect XLSX mapping: %+v", rows)
	}
}

func TestAmazonQuarterAggregationAndHistoricalDedup(t *testing.T) {
	dsn := os.Getenv("TAX_TEST_DSN")
	if dsn == "" {
		t.Skip("set TAX_TEST_DSN for isolated PostgreSQL integration")
	}
	if !strings.Contains(dsn, "host=127.0.0.1") {
		t.Fatal("integration database must be local")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	schema := fmt.Sprintf("amazon_tax_%d", time.Now().UnixNano())
	for _, query := range []string{"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema} {
		if err := tx.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.AutoMigrate(&model.AmazonTaxShipment{}, &model.AmazonTaxFinance{}, &model.TaxExchangeRate{}, &model.AmazonProductCategory{}, &model.TaxImportLog{}, &model.TaxCatalogEntry{}); err != nil {
		t.Fatal(err)
	}
	for input, want := range map[string]string{
		"2026/1/3 15:04:05": "2026-01-05", "2026-01-31T00:00:00Z": "2026-02-02",
		"Jan 4, 2026 1:22:58 AM PST": "2026-01-05", "4 Jan 2026 15:04:05 PST": "2026-01-05",
		"2026年1月4日": "2026-01-05", "2028-12-31": "2029-01-01",
		"1/3/2026 15:04:05": "2026-01-05", " 01/04/2026 3:04:05 PM ": "2026-01-05",
		"46025.5": "2026-01-05", "1767398400": "2026-01-05", "1767398400000": "2026-01-05",
	} {
		var got string
		if err := tx.Raw("SELECT "+taxRateDateSQL(amazonDateSQL("source"))+" FROM (SELECT ?::text AS source) dates", input).Scan(&got).Error; err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("date %s: got %s want %s", input, got, want)
		}
	}
	if err := tx.Exec(`CREATE UNIQUE INDEX identity_test ON amazon_tax_shipments ((COALESCE(BTRIM(data->>'亚马逊订单编号'), '')), (COALESCE(BTRIM(data->>'货件编号'), '')), (COALESCE(BTRIM(data->>'货件商品编号'), '')))`).Error; err != nil {
		t.Fatal(err)
	}
	data := map[string]string{"配送日期": "1/3/2026 15:04:05", "货币": "USD", "亚马逊订单编号": "O1", "货件编号": "S1", "货件商品编号": "I1", "已发货数量": "2", "商品价格": "$100.004", "商品税": "10", "运费": "5", "运费税": "1", "礼品包装价格": "2", "礼品包装税费": "999", "商品促销折扣": "-3", "货件促销折扣": "-1"}
	old := model.AmazonTaxShipment{Month: "2026-01", Quarter: "2026-Q1", Fingerprint: "old-fingerprint", Data: data}
	if err := tx.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	duplicate := old
	duplicate.ID = 0
	duplicate.Fingerprint = "new-fingerprint"
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&duplicate)
	if result.Error != nil || result.RowsAffected != 0 {
		t.Fatalf("historical duplicate: %v, %d", result.Error, result.RowsAffected)
	}
	data2 := map[string]string{}
	for k, v := range data {
		data2[k] = v
	}
	data2["货件商品编号"] = "I2"
	second := model.AmazonTaxShipment{Month: "2026-01", Quarter: "2026-Q1", Fingerprint: "second", Data: data2}
	if err := tx.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	for i, kind := range []string{"Refund", "Refund_Retrocharge", "Order", "Order_Retrocharge", "Service Fee", "Transfer", "Adjustment"} {
		row := model.AmazonTaxFinance{Month: "2026-01", Quarter: "2026-Q1", Fingerprint: fmt.Sprint(i), Data: map[string]string{"date/time": "Jan 4, 2026 1:22:58 AM PST", "type": kind, "product sales": "-10", "product sales tax": "-1", "shipping credits": "-2", "shipping credits tax": "-3", "gift wrap credits": "-4", "giftwrap credits tax": "-5", "Regulatory Fee": "-6", "Tax On Regulatory Fee": "-7", "promotional rebates": "-8", "promotional rebates tax": "-9", "other transaction fees": "-9", "other": "-10", "marketplace withheld tax": "(2)", "fba fees": "-3", "selling fees": "-4", "total": "-50"}}
		if err := tx.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	subscription := model.AmazonTaxFinance{Month: "2026-01", Quarter: "2026-Q1", Fingerprint: "subscription", Data: map[string]string{"date/time": "Jan 4, 2026 1:22:58 AM PST", "type": "Service Fee", "description": "Subscription", "other": "-10", "marketplace withheld tax": "(2)", "total": "-99"}}
	if err := tx.Create(&subscription).Error; err != nil {
		t.Fatal(err)
	}
	for i, description := range []string{"FBA Long Term Storage Fee", "FBA Inventory Storage Fee", "FBA Removal Order: Disposal Fee", "Other FBA fee"} {
		row := model.AmazonTaxFinance{Month: "2026-01", Quarter: "2026-Q1", Fingerprint: fmt.Sprintf("storage-%d", i), Data: map[string]string{"date/time": "Jan 4, 2026 1:22:58 AM PST", "type": "FBA Transaction fees", "description": description, "fba fees": "-6", "total": "-6"}}
		if err := tx.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	placement := model.AmazonTaxFinance{Month: "2026-01", Quarter: "2026-Q1", Fingerprint: "placement", Data: map[string]string{"date/time": "Jan 4, 2026 1:22:58 AM PST", "type": "FBA Transaction fees", "description": "FBA Inbound Placement Service Fee", "fba fees": "-7", "total": "-7"}}
	if err := tx.Create(&placement).Error; err != nil {
		t.Fatal(err)
	}
	financeOnly := model.AmazonTaxFinance{Month: "2026-04", Quarter: "2026-Q2", Fingerprint: "q2", Data: map[string]string{"type": "Transfer", "total": "1,000"}}
	if err := tx.Create(&financeOnly).Error; err != nil {
		t.Fatal(err)
	}
	var rows []amazonTaxQuarter
	if err := tx.Raw(amazonQuarterSQL("USD")).Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("quarters: %+v", rows)
	}
	q := rows[0]
	got := []string{q.SalesVolume.String(), q.Sales.String(), q.Refund.String(), q.Tax.String(), q.Delivery.String(), q.Selling.String(), q.Ads.String(), q.Storage.String(), q.Other.String(), q.TaxableIncome.String()}
	want := []string{"4", "2226.01", "-65", "-16", "-19", "-8", "-9", "-18", "-56", "2161.01"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("metric %d: got %s want %s", i, got[i], want[i])
		}
	}
	if !rows[1].Sales.IsZero() || !rows[1].Other.IsZero() {
		t.Fatalf("transfer leaked into totals: %+v", rows[1])
	}

	for _, row := range []model.TaxExchangeRate{{Date: "2026-01-03", Currency: "USD", Rate: decimal.NewFromInt(99)}, {Date: "2026-01-05", Currency: "USD", Rate: decimal.NewFromInt(7)}} {
		if err := tx.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	var converted []amazonTaxQuarter
	if err := tx.Raw(amazonQuarterSQL("CNY")).Scan(&converted).Error; err != nil {
		t.Fatal(err)
	}
	if converted[0].SalesVolume.String() != "4" || converted[0].Sales.String() != "15582.06" || converted[0].Delivery.String() != "-133" || converted[0].TaxableIncome.String() != "15127.06" {
		t.Fatalf("Monday conversion or quantity changed: %+v", converted[0])
	}
	if err := tx.Where("date = ?", "2026-01-05").Delete(&model.TaxExchangeRate{}).Error; err != nil {
		t.Fatal(err)
	}
	converted = nil
	if err := tx.Raw(amazonQuarterSQL("CNY")).Scan(&converted).Error; err != nil {
		t.Fatal(err)
	}
	if converted[0].Delivery != nil || converted[0].Sales != nil || converted[0].TaxableIncome != nil {
		t.Fatalf("missing Monday must not yield partial totals: %+v", converted[0])
	}
	var issues []amazonSalesConversionIssue
	if err := tx.Raw(amazonSalesConversionIssuesSQL("CNY"), "2026-Q1").Scan(&issues).Error; err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 || issues[0].ShipmentID != old.ID || issues[0].SourceDate != "1/3/2026 15:04:05" || issues[0].RateDate != "2026-01-05" || issues[0].Currency != "USD" {
		t.Fatalf("incorrect missing-rate source records: %+v", issues)
	}
	if err := tx.Exec(`UPDATE amazon_tax_shipments SET data = jsonb_set(data, '{配送日期}', '"invalid-date"') WHERE id = ?`, old.ID).Error; err != nil {
		t.Fatal(err)
	}
	issues = nil
	if err := tx.Raw(amazonSalesConversionIssuesSQL("CNY"), "2026-Q1").Scan(&issues).Error; err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 || issues[0].RateDate != "" || issues[0].Reason != "配送日期无法解析" {
		t.Fatalf("incorrect invalid-date source record: %+v", issues)
	}
	if err := tx.Exec(`UPDATE amazon_tax_shipments SET data = jsonb_set(data, '{配送日期}', '"1/3/2026 15:04:05"') WHERE id = ?`, old.ID).Error; err != nil {
		t.Fatal(err)
	}
	// A Monday holiday uses Tuesday's stored publication for old imports too.
	if err := tx.Where("date = ?", "2026-01-03").Delete(&model.TaxExchangeRate{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("INSERT INTO tax_exchange_rates (date, currency, rate) VALUES ('2026-01-06', 'USD', 7.25)").Error; err != nil {
		t.Fatal(err)
	}
	converted = nil
	if err := tx.Raw(amazonQuarterSQL("CNY")).Scan(&converted).Error; err != nil {
		t.Fatal(err)
	}
	if converted[0].Sales == nil || converted[0].Sales.String() != "16138.56" || converted[0].Delivery.String() != "-137.75" || converted[0].TaxableIncome.String() != "15667.31" {
		t.Fatalf("holiday conversion: %+v", converted[0])
	}
	testAmazonProductCategories(t, tx)
	// Existing shipment rows may mix currencies. Both display modes must convert
	// each row at its shipping date before adding them, without reimporting.
	for code, amount := range map[string]string{"USD": "100", "MXN": "100", "CHF": "80", "PLN": "20", "CNY": "80"} {
		row := model.AmazonTaxShipment{Month: "2026-10", Quarter: "2026-Q4", Fingerprint: "mixed-" + code, Data: map[string]string{
			"配送日期": "2026-10-03T11:14:39-07:00", "货币": code, "商品价格": amount,
			"亚马逊订单编号": code, "货件编号": code, "货件商品编号": code, "已发货数量": "1",
		}}
		if err := tx.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for code, rate := range map[string]string{"USD": "8", "MXN": "0.4", "CHF": "8", "PLN": "2"} {
		if err := tx.Create(&model.TaxExchangeRate{Date: "2026-10-05", Currency: code, Rate: decimal.RequireFromString(rate)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for currency, want := range map[string]string{"USD": "200", "CNY": "1600"} {
		var mixed []amazonTaxQuarter
		if err := tx.Raw(amazonQuarterSQL(currency)).Scan(&mixed).Error; err != nil {
			t.Fatal(err)
		}
		q := mixed[len(mixed)-1]
		if q.Quarter != "2026-Q4" || q.Sales == nil || q.Sales.String() != want || q.TaxableIncome.String() != want || q.SalesVolume.String() != "5" {
			t.Fatalf("mixed-currency %s: %+v", currency, q)
		}
	}
	for site, expected := range map[string]string{"US": "USD:100", "CA": "CAD:0"} {
		parts := strings.Split(expected, ":")
		currency, wantValue := parts[0], parts[1]
		var filtered []amazonTaxQuarter
		if err := tx.Raw(amazonQuarterSQL(currency, "sunoutly", site)).Scan(&filtered).Error; err != nil {
			t.Fatal(err)
		}
		q := filtered[len(filtered)-1]
		if q.Quarter != "2026-Q4" || q.Sales == nil {
			t.Fatalf("site currency filter %s: %+v", site, q)
		}
		if q.Sales.String() != wantValue {
			t.Fatalf("site currency filter %s: got %s want %s", site, q.Sales, wantValue)
		}
	}
	if err := tx.Where("date = ? AND currency = ?", "2026-10-05", "USD").Delete(&model.TaxExchangeRate{}).Error; err != nil {
		t.Fatal(err)
	}
	var mixed []amazonTaxQuarter
	if err := tx.Raw(amazonQuarterSQL("USD")).Scan(&mixed).Error; err != nil {
		t.Fatal(err)
	}
	if mixed[len(mixed)-1].Sales != nil {
		t.Fatal("missing USD denominator must not produce a partial sales total")
	}
	issues = nil
	if err := tx.Raw(amazonSalesConversionIssuesSQL("USD"), "2026-Q4").Scan(&issues).Error; err != nil {
		t.Fatal(err)
	}
	if len(issues) != 4 {
		t.Fatalf("USD source rows must not require rates, other currencies must: %+v", issues)
	}
	testAmazonCustomerReturnsDelivery(t, tx)
	testAmazonServiceFeeColumns(t, tx)
}

func testAmazonServiceFeeColumns(t *testing.T, db *gorm.DB) {
	t.Helper()
	for i, description := range []string{"FBA Inbound Placement Service Fee", " Premium Services Fee ", "Unrelated"} {
		row := model.AmazonTaxFinance{StoreID: "service-fee-test", SiteID: "US", Month: "2027-01", Quarter: "2027-Q1", Fingerprint: fmt.Sprint("service-", i), Data: map[string]string{"date/time": "2027-01-04", "type": "Service Fee", "description": description, "other": "-12.34", "selling fees": "-5.67", "fba fees": "-999", "total": "-888"}}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.TaxExchangeRate{Date: "2027-01-04", Currency: "USD", Rate: decimal.NewFromInt(7)}).Error; err != nil {
		t.Fatal(err)
	}
	for _, period := range []string{"quarter", "month"} {
		for _, currency := range []string{"USD", "CNY"} {
			var rows []amazonTaxQuarter
			if err := db.Raw(amazonQuarterSQL(currency, "service-fee-test", "US", "USD", period)).Scan(&rows).Error; err != nil {
				t.Fatal(err)
			}
			var row *amazonTaxQuarter
			for i := range rows {
				if rows[i].Quarter == "2027-Q1" {
					row = &rows[i]
					break
				}
			}
			if row == nil {
				t.Fatal("missing service fee period")
			}
			factor := decimal.NewFromInt(1)
			if currency == "CNY" {
				factor = decimal.NewFromInt(7)
			}
			for _, check := range []struct {
				value *decimal.Decimal
				want  string
			}{
				{row.Delivery, "0"}, {row.OtherInboundPlacement, "-12.34"}, {row.OtherPremiumServices, "-5.67"}, {row.Selling, "0"}, {row.SellingFees, "0"}, {row.Other, "-18.01"}, {row.Storage, "0"}, {row.Ads, "0"},
			} {
				if check.value == nil || !check.value.Equal(decimal.RequireFromString(check.want).Mul(factor)) {
					t.Fatalf("%s %s expected %s got %v", period, currency, check.want, check.value)
				}
			}
		}
	}
}

func TestAmazonShipmentSourceSharedAcrossSites(t *testing.T) {
	for _, site := range []string{"US", "CA"} {
		query := amazonQuarterSQL(amazonSiteCurrency(site), "Qfn", site)
		// Shipment files are store-level shared data; the selected site claims
		// its rows through the 货币 column, not the site_id column.
		sharedSource := "(SELECT * FROM amazon_tax_shipments WHERE (store_id = 'Qfn' OR store_id = '')) s"
		if !strings.Contains(query, sharedSource) {
			t.Fatalf("%s shipment source must only filter by store", site)
		}
		financeSource := "(SELECT * FROM amazon_tax_finances WHERE (store_id = 'Qfn' OR store_id = '') AND (site_id = '" + site + "' OR site_id = '' OR site_id IS NULL)) f"
		if !strings.Contains(query, financeSource) {
			t.Fatalf("%s finance source must retain site isolation", site)
		}
		if !strings.Contains(query, "COALESCE(NULLIF(UPPER(BTRIM(s.data->>'货币')), ''), 'USD') = '"+amazonSiteCurrency(site)+"'") {
			t.Fatalf("%s sales must take only rows in the site currency", site)
		}
	}
}

func testAmazonCustomerReturnsDelivery(t *testing.T, db *gorm.DB) {
	t.Helper()
	entries := []struct{ kind, description, amount string }{
		{"Order", "Product", "-13158.82"},
		{"Refund", "Product", "18.88"},
		{"FBA Transaction fees", "FBA Inbound Placement Service Fee", "-350"},
		{"FBA Transaction fees", "FBA Customer Returns Fee (Non-apparel and Non-shoes) for ASIN: A", "-5.20"},
		{"FBA Transaction fees", "FBA Customer Returns Fee (Non-apparel and Non-shoes) for ASIN: B", "-5.20"},
		{"FBA Transaction fees", "FBA Customer Returns Fee (Non-apparel and Non-shoes) for ASIN: C", "-5.20"},
		{"FBA Transaction fees", "FBA Customer Returns Fee (Non-apparel and Non-shoes) for ASIN: D", "-5.20"},
		{"FBA Transaction fees", "FBA Customer Returns Fee (Non-apparel and Non-shoes) for ASIN: E", "-35.70"},
		{"FBA Transaction fees", "FBA Customer Returns Fee (Non-apparel and Non-shoes) for ASIN: F", "-5.25"},
		{"FBA Transaction fees", "FBA Inventory Storage Fee", "-40"},
		{"Service Fee", "FBA Customer Returns Fee", "-100"},
		{"Fulfilment by Amazon transaction fees", "Fulfilment by Amazon (FBA) Inventory Storage Fee", "-305.87"},
		{"Fulfilment by Amazon transaction fees", "Fulfilment by Amazon removal order: disposal fee", "-24.91"},
		{"Fulfilment by Amazon transaction fees", "Unrelated fulfilment fee", "-9"},
	}
	for i, entry := range entries {
		row := model.AmazonTaxFinance{StoreID: "Qfn", SiteID: "US", Month: "2026-07", Quarter: "2026-Q3", Fingerprint: fmt.Sprintf("return-fee-%d", i), Data: map[string]string{"date/time": "Jul 1, 2026 9:00:00 AM PDT", "type": entry.kind, "description": entry.description, "fba fees": entry.amount, "total": entry.amount}}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.TaxExchangeRate{Date: "2026-07-01", Currency: "USD", Rate: decimal.NewFromInt(7)}).Error; err != nil {
		t.Fatal(err)
	}
	for currency, want := range map[string]string{"USD": "-13882.47", "CNY": "-97177.29"} {
		var quarters []amazonTaxQuarter
		if err := db.Raw(amazonQuarterSQL(currency, "Qfn", "US")).Scan(&quarters).Error; err != nil {
			t.Fatal(err)
		}
		found := false
		for _, q := range quarters {
			if q.Quarter != "2026-Q3" {
				continue
			}
			found = true
			if q.Delivery == nil || q.DeliveryFBAFees == nil || q.DeliveryFulfilmentDisposal == nil || q.DeliveryTransactionOther == nil || q.DeliveryFulfilmentStorage == nil || q.Delivery.String() != want || !q.Delivery.Equal(q.DeliveryFBAFees.Add(*q.DeliveryFulfilmentDisposal).Add(*q.DeliveryTransactionOther).Add(*q.DeliveryFulfilmentStorage)) {
				t.Fatalf("%s delivery or breakdown incorrect: %+v", currency, q)
			}
			factor := decimal.NewFromInt(1)
			if currency == "CNY" {
				factor = decimal.NewFromInt(7)
			}
			// Storage keeps only the FBA Transaction Fees inventory storage
			// (-40); the Fulfilment by Amazon (FBA) Inventory Storage Fee
			// (-305.87) moved into delivery.
			if q.Other == nil || !q.Other.Equal(decimal.NewFromInt(-9).Mul(factor)) || q.Storage == nil || !q.Storage.Equal(decimal.NewFromInt(-40).Mul(factor)) || q.StorageTransaction == nil || !q.StorageTransaction.Equal(decimal.NewFromInt(-40).Mul(factor)) || q.DeliveryFulfilmentStorage == nil || !q.DeliveryFulfilmentStorage.Equal(decimal.NewFromFloat(-305.87).Mul(factor)) {
				t.Fatalf("fulfilment delivery fees leaked into other/storage totals: %+v", q)
			}
		}
		if !found {
			t.Fatal("missing delivery quarter")
		}
	}
}

func testAmazonProductCategories(t *testing.T, db *gorm.DB) {
	t.Helper()
	oldDB := database.DB
	database.DB = db
	defer func() { database.DB = oldDB }()
	router := gin.New()
	router.GET("/categories", ListAmazonProductCategories)
	router.POST("/categories", SaveAmazonProductCategory)
	router.PUT("/categories/:id", SaveAmazonProductCategory)
	router.DELETE("/categories/:id", DeleteAmazonProductCategory)
	router.GET("/quarters", ListAmazonTaxQuarters)
	call := func(method, path, body string, status int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		if out.Code != status {
			t.Fatalf("%s %s: %d %s", method, path, out.Code, out.Body.String())
		}
		return out.Body.Bytes()
	}
	check := func(want map[string]string) {
		t.Helper()
		var response struct {
			Items []amazonTaxQuarter `json:"items"`
		}
		// All-countries views only support CNY; category volumes are currency-free.
		if err := json.Unmarshal(call("GET", "/quarters?currency=CNY", "", 200), &response); err != nil {
			t.Fatal(err)
		}
		actual := map[string]string{}
		sum := decimal.Zero
		for _, category := range response.Items[0].SalesCategories {
			actual[category.Category] = category.Quantity.String()
			sum = sum.Add(category.Quantity)
		}
		if fmt.Sprint(actual) != fmt.Sprint(want) || !sum.Equal(response.Items[0].SalesVolume) {
			t.Fatalf("categories %v want %v, sum %s total %s", actual, want, sum, response.Items[0].SalesVolume)
		}
	}
	check(map[string]string{"未分类": "4"})
	if err := db.Exec(`UPDATE amazon_tax_shipments SET data = data || '{"商品名称":"Enameled Cast Iron Dutch Oven, 6QT Pot with Lid, Cream White"}'::jsonb WHERE data->>'货件商品编号' = 'I1'`).Error; err != nil {
		t.Fatal(err)
	}
	call("POST", "/categories", `{"keyword":"Oven","category":"烤箱"}`, 200)
	var rule model.AmazonProductCategory
	if err := json.Unmarshal(call("POST", "/categories", `{"keyword":" Dutch Oven ","category":"荷兰锅"}`, 200), &rule); err != nil {
		t.Fatal(err)
	}
	// Current rules immediately classify old data, without changing total volume.
	check(map[string]string{"荷兰锅": "2", "未分类": "2"})
	call("POST", "/categories", `{"keyword":"dutch oven","category":"重复"}`, 409)
	call("POST", "/categories", `{"keyword":" ","category":"空"}`, 400)
	call("PUT", fmt.Sprintf("/categories/%d", rule.ID), `{"keyword":"DUTCH OVEN","category":"珐琅锅"}`, 200)
	check(map[string]string{"珐琅锅": "2", "未分类": "2"})
	call("GET", "/categories", "", 200)
	call("DELETE", fmt.Sprintf("/categories/%d", rule.ID), "", 200)
	check(map[string]string{"烤箱": "2", "未分类": "2"})
	summaryMessage, err := json.Marshal(amazonSummary{DisplayName: "Test Summary", Start: "1900-01-01", End: "2999-12-31", Currency: "USD", Tax: decimal.Zero, Storage: decimal.Zero, Delivery: decimal.Zero})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.TaxImportLog{Platform: "amazon", ReportType: "finance_summary", Status: "locked", StoreID: "", SiteID: "", Message: string(summaryMessage)}).Error; err != nil {
		t.Fatal(err)
	}
	call("GET", "/quarters?currency=CNY", "", 200)
}
