package handler

import (
	"encoding/csv"
	"fmt"
	"mime"
	"net/http/httptest"
	"os"
	"strings"
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

func TestTaxReportFileName(t *testing.T) {
	for _, tc := range []struct{ query, want string }{
		{"?store_id=Qfn&site_id=US&date_from=2026-01-01&date_to=2026-03-31", "Amazon_季度汇总_Qfn_美国-US_2026-01-01至2026-03-31_USD_"},
		{"?quarter=2026-Q1,2026-Q2", "Amazon_季度汇总_全部店铺_全部国家_2026-Q1,2026-Q2_USD_"},
		{"?store_id=A%2FB&site_id=CA&month=2026-02", "Amazon_季度汇总_A-B_加拿大-CA_2026-02_USD_"},
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/export"+tc.query, nil)
		got := taxReportFileName(c, "Amazon", "季度汇总", "USD")
		if !strings.HasPrefix(got, tc.want) || !strings.HasSuffix(got, ".csv") {
			t.Fatalf("filename: %s", got)
		}
	}
}

func TestTaxReportExportCurrenciesIntegration(t *testing.T) {
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
	defer tx.Rollback()
	schema := fmt.Sprintf("tax_export_test_%d", time.Now().UnixNano())
	for _, sql := range []string{"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema} {
		if err := tx.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.AutoMigrate(&model.User{}, &model.TaxDownloadLog{}, &model.TaxCatalogEntry{}, &model.TaxDataBatch{}, &model.TaxRawRow{}, &model.TaxProcessedRow{}, &model.TaxQuarter{}, &model.TaxExchangeRate{}, &model.AmazonTaxShipment{}, &model.AmazonTaxFinance{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "export-test", Password: "fixture", Email: "fixture@example.invalid"}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	create := func(value interface{}) {
		t.Helper()
		if err := tx.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	create(&model.TaxExchangeRate{Date: "2026-01-05", Currency: "USD", Rate: decimal.NewFromInt(7)})
	create(&model.TaxExchangeRate{Date: "2026-04-02", Currency: "USD", Rate: decimal.NewFromInt(7)})
	batch := model.TaxDataBatch{StoreID: "TVOFEK", SiteID: "US", FileHash: "fixture", Status: "locked"}
	create(&batch)
	for i, month := range []string{"2026-01", "2026-04"} {
		quarter := fmt.Sprintf("2026-Q%d", i+1)
		raw := model.TaxRawRow{BatchID: batch.ID, StoreID: "TVOFEK", SiteID: "US", Fingerprint: month, Data: map[string]string{"Currency": "USD"}}
		create(&raw)
		create(&model.TaxProcessedRow{BatchID: batch.ID, RawRowID: raw.ID, Month: month, Quarter: quarter, FeeType: "销售额", Amount: decimal.NewFromInt(100), Fee: decimal.NewFromInt(700), Rate: decimal.NewFromInt(7), Currency: "USD"})
		create(&model.AmazonTaxShipment{StoreID: "Qfn", SiteID: "US", Month: month, Quarter: quarter, Fingerprint: month, Data: map[string]string{"配送日期": month + "-02", "货币": "USD", "商品价格": "100", "已发货数量": "1", "亚马逊订单编号": month, "货件编号": month, "货件商品编号": month}})
	}
	old := database.DB
	database.DB = tx
	defer func() { database.DB = old }()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("userID", user.ID); c.Next() })
	router.GET("/walmart", ExportTaxReport)
	router.GET("/amazon", ExportAmazonTaxReport)
	for _, platform := range []string{"walmart", "amazon"} {
		for _, currency := range []string{"USD", "CNY"} {
			store := "TVOFEK"
			if platform == "amazon" {
				store = "Qfn"
			}
			path := "/" + platform + "?type=quarters&store_id=" + store + "&site_id=US&site_currency=USD&currency=" + currency + "&date_from=2026-01-01&date_to=2026-03-31"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
			if w.Code != 200 {
				t.Fatalf("%s %d %s", path, w.Code, w.Body.String())
			}
			content := strings.TrimPrefix(w.Body.String(), string([]byte{0xef, 0xbb, 0xbf}))
			rows, err := csv.NewReader(strings.NewReader(content)).ReadAll()
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != 2 || rows[1][0] != "2026-Q1" {
				t.Fatalf("wrong period rows: %+v", rows)
			}
			sale := -1
			for i, label := range rows[0] {
				if strings.HasPrefix(label, "销售额") {
					sale = i
				}
			}
			want := "100.00"
			if currency == "CNY" {
				want = "700.00"
			}
			if sale < 0 || rows[1][sale] != want {
				t.Fatalf("%s %s sales: %+v want %s", platform, currency, rows, want)
			}
			if !strings.Contains(strings.Join(rows[0], ","), currency) {
				t.Fatal("currency header missing")
			}
			_, params, err := mime.ParseMediaType(w.Header().Get("Content-Disposition"))
			if err != nil {
				t.Fatal(err)
			}
			for _, part := range []string{store, "美国-US", "2026-01-01至2026-03-31", currency} {
				if !strings.Contains(params["filename"], part) {
					t.Fatalf("filename missing %s: %s", part, params["filename"])
				}
			}
			var log model.TaxDownloadLog
			if err := tx.Order("id desc").First(&log).Error; err != nil {
				t.Fatal(err)
			}
			if log.Currency != currency || log.RowCount != 1 || log.FileName != params["filename"] {
				t.Fatal("download log does not match file")
			}
		}
	}
}
