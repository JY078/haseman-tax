package handler

import (
	"encoding/json"
	"fmt"
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
)

func TestTaxFilteredTotalsAcrossPages(t *testing.T) {
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
	conn, _ := db.DB()
	defer conn.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	previous := database.DB
	database.DB = tx
	defer func() { tx.Rollback(); database.DB = previous }()
	schema := fmt.Sprintf("tax_totals_%d", time.Now().UnixNano())
	for _, query := range []string{"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema} {
		if err := tx.Exec(query).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.AutoMigrate(&model.TaxRawRow{}, &model.TaxProcessedRow{}, &model.TaxExchangeRate{}, &model.TikTokTaxRawRow{}, &model.TikTokTaxDetail{}, &model.TikTokTaxAd{}, &model.TikTokTaxFBT{}, &model.TikTokOrder{}, &model.TikTokSettlement{}, &model.TikTokTaxTransactionSummary{}, &model.AmazonTaxShipment{}, &model.AmazonTaxFinance{}); err != nil {
		t.Fatal(err)
	}
	create := func(row interface{}) {
		t.Helper()
		if err := tx.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 5; i++ {
		month, store, site := "2026-01", "Forious", "US"
		if i == 3 {
			month = "2026-02"
		}
		if i == 4 {
			store = "Other"
		}
		if i == 5 {
			site = "CA"
		}
		amount := decimal.NewFromInt(int64(i * 10))
		data := map[string]string{"Amount": amount.String(), "Quantity": fmt.Sprint(i), "Total settlement amount": amount.String(), "Type": "Adjustment", "币种": "USD"}
		raw := model.TaxRawRow{BatchID: 1, StoreID: store, SiteID: site, Fingerprint: fmt.Sprint(i), Data: data}
		create(&raw)
		create(&model.TaxProcessedRow{BatchID: 1, RawRowID: raw.ID, Month: month, Quarter: "2026-Q1", Fee: amount})
		tkRaw := model.TikTokTaxRawRow{BatchID: 1, StoreID: store, SiteID: site, Fingerprint: fmt.Sprint(i), Data: data}
		create(&tkRaw)
		date, _ := time.Parse("2006-01-02", month+"-02")
		create(&model.TikTokTaxDetail{BatchID: 1, RawRowID: tkRaw.ID, StoreID: store, SiteID: site, Month: month, Quarter: "2026-Q1", PostedAt: date, Income: amount, Rate: decimal.NewFromInt(7)})
		create(&model.TikTokTaxAd{BatchID: 1, StoreID: store, SiteID: site, Month: month, Quarter: "2026-Q1", Date: month + "-02", Amount: amount})
		create(&model.TikTokTaxFBT{BatchID: 1, StoreID: store, SiteID: site, Month: month, Quarter: "2026-Q1", FBTFulfillmentFee: amount})
		create(&model.TikTokOrder{BatchID: 1, StoreID: store, SiteID: site, Month: month, Quarter: "2026-Q1", OrderID: fmt.Sprint(i), Data: data})
		create(&model.TikTokSettlement{BatchID: 1, StoreID: store, SiteID: site, Month: month, Quarter: "2026-Q1", StatementDate: month + "-02", Currency: "USD", Amount: amount, Data: data})
		create(&model.AmazonTaxFinance{ImportLogID: 1, StoreID: store, SiteID: site, Month: month, Quarter: "2026-Q1", Fingerprint: fmt.Sprint(i), Data: data})
		// Shipments are store-wide: the first three rows are imported under CA.
		if i < 5 {
			create(&model.AmazonTaxShipment{ImportLogID: 1, StoreID: store, SiteID: "CA", Month: month, Quarter: "2026-Q1", Fingerprint: fmt.Sprint(i), Data: data})
		}
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/walmart", ListTaxRaw)
	router.GET("/details", ListTikTokTaxDetails)
	router.GET("/ads", ListTikTokTaxAds)
	router.GET("/fbt", ListTikTokTaxFBT)
	router.GET("/orders", ListTikTokOrders)
	router.GET("/settlements", ListTikTokSettlements)
	router.GET("/amazon/:kind", ListAmazonTaxRows)
	for _, endpoint := range []struct{ path, key string }{
		{"/walmart", "Amount"}, {"/details", "income"}, {"/ads", "amount"}, {"/fbt", "fbtFulfillmentFee"}, {"/orders", "Amount"}, {"/settlements", "Total settlement amount"}, {"/amazon/finance", "Amount"}, {"/amazon/shipments", "Amount"},
	} {
		t.Run(endpoint.path, func(t *testing.T) {
			for _, filtered := range []bool{false, true} {
				for _, page := range []int{1, 2} {
					url := fmt.Sprintf("%s?store_id=Forious&site_id=US&include_totals=1&page_size=1&page=%d", endpoint.path, page)
					want, count := "60", 3
					if filtered {
						want, count = "30", 2
						if endpoint.path == "/walmart" {
							url += "&filter_field=%E6%9C%88%E4%BB%BD&filter_value=2026-01"
						} else {
							url += "&month=2026-01"
						}
					}
					out := httptest.NewRecorder()
					router.ServeHTTP(out, httptest.NewRequest("GET", url, nil))
					if out.Code != 200 {
						t.Fatalf("%s: %d %s", url, out.Code, out.Body.String())
					}
					var result struct {
						Items  []json.RawMessage
						Total  int
						Totals map[string]decimal.Decimal
					}
					if err := json.Unmarshal(out.Body.Bytes(), &result); err != nil {
						t.Fatal(err)
					}
					if len(result.Items) != 1 || result.Total != count || result.Totals[endpoint.key].String() != want {
						t.Fatalf("%s: wrong page/count/total: %s", url, out.Body.String())
					}
				}
			}
		})
	}
	for _, currency := range []string{"USD", "CNY"} {
		rows, err := listTikTokQuarterSummary(tx, currency, "Forious", "US")
		if err != nil {
			t.Fatal(err)
		}
		want := "60"
		if currency == "CNY" {
			want = "420"
		}
		if len(rows) != 1 || rows[0].Income == nil || rows[0].Income.String() != want {
			t.Fatalf("site-filtered %s quarter: %+v", currency, rows)
		}
	}
}
