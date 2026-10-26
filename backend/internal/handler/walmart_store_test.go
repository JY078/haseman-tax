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
	"gorm.io/gorm/logger"
)

func TestWalmartLegacyStoreIntegration(t *testing.T) {
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
	schema := fmt.Sprintf("walmart_store_test_%d", time.Now().UnixNano())
	for _, sql := range []string{"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema} {
		if err := tx.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.AutoMigrate(&model.TaxDataBatch{}, &model.TaxRawRow{}, &model.TaxProcessedRow{}, &model.TaxQuarter{}, &model.TaxExchangeRate{}); err != nil {
		t.Fatal(err)
	}
	for index, store := range []string{"walmart", "", "TVOFEK", "other-store"} {
		batch := model.TaxDataBatch{StoreID: store, SiteID: "US", FileHash: fmt.Sprint(index), FileName: "fixture.csv", Status: "locked"}
		if err := tx.Create(&batch).Error; err != nil {
			t.Fatal(err)
		}
		raw := model.TaxRawRow{BatchID: batch.ID, StoreID: store, SiteID: "US", Fingerprint: fmt.Sprint(index), Data: map[string]string{"Currency": "USD"}}
		if err := tx.Create(&raw).Error; err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			snapshot := model.TaxQuarter{BatchID: batch.ID, Quarter: "2026-Q1", Totals: map[string]decimal.Decimal{"销售额": decimal.NewFromInt(10)}}
			if err := tx.Create(&snapshot).Error; err != nil {
				t.Fatal(err)
			}
		} else {
			row := model.TaxProcessedRow{BatchID: batch.ID, RawRowID: raw.ID, Quarter: "2026-Q1", Month: "2026-01", FeeType: "销售额", Fee: decimal.NewFromInt(10), Amount: decimal.NewFromInt(10), Rate: decimal.NewFromInt(1), Currency: "USD"}
			if err := tx.Create(&row).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, tc := range []struct {
		store string
		count int64
		sales string
	}{
		{"TVOFEK", 3, "30"}, {"walmart", 3, "30"}, {"other-store", 1, "10"}, {"", 4, "40"}, {"missing", 0, "0"},
	} {
		t.Run(tc.store, func(t *testing.T) {
			for _, modelValue := range []interface{}{&model.TaxDataBatch{}, &model.TaxRawRow{}} {
				var count int64
				if err := filterTaxSite(filterWalmartStore(tx.Model(modelValue), tc.store), "US").Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if count != tc.count {
					t.Fatalf("%T store %q count=%d want %d", modelValue, tc.store, count, tc.count)
				}
				if err := filterTaxSite(filterWalmartStore(tx.Model(modelValue), tc.store), "CA").Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if count != 0 {
					t.Fatalf("country filter leaked %d rows", count)
				}
			}
			rows, _, err := summarizeTaxProcessed(tx, tc.store, "", "US")
			if err != nil {
				t.Fatal(err)
			}
			total := decimal.Zero
			for _, row := range rows {
				total = total.Add(row.Totals["销售额"])
			}
			if total.String() != tc.sales {
				t.Fatalf("summary sales=%s want %s", total, tc.sales)
			}
		})
	}
	t.Run("six quarters and overlapping legacy imports", func(t *testing.T) {
		// Discard this test's earlier fixtures inside the rollback-only schema.
		if err := tx.Exec("TRUNCATE tax_processed_rows, tax_raw_rows, tax_quarters, tax_data_batches RESTART IDENTITY").Error; err != nil {
			t.Fatal(err)
		}
		seed := func(store, site, hash, identity, quarter, month string, amount int64) {
			t.Helper()
			b := model.TaxDataBatch{StoreID: store, SiteID: site, FileHash: hash, FileName: "fixture.csv", Status: "locked"}
			if err := tx.Create(&b).Error; err != nil {
				t.Fatal(err)
			}
			r := model.TaxRawRow{BatchID: b.ID, StoreID: store, SiteID: site, Fingerprint: identity, Data: map[string]string{"Currency": "USD", "Ship Qty": "1"}}
			if err := tx.Create(&r).Error; err != nil {
				t.Fatal(err)
			}
			p := model.TaxProcessedRow{BatchID: b.ID, RawRowID: r.ID, Quarter: quarter, Month: month, FeeType: "销售额", Fee: decimal.NewFromInt(amount * 7), Amount: decimal.NewFromInt(amount), Rate: decimal.NewFromInt(7), Currency: "USD"}
			if err := tx.Create(&p).Error; err != nil {
				t.Fatal(err)
			}
			s := model.TaxQuarter{BatchID: b.ID, Quarter: quarter, Totals: map[string]decimal.Decimal{"销售额": p.Fee}}
			if err := tx.Create(&s).Error; err != nil {
				t.Fatal(err)
			}
		}
		periods := []string{"2025-Q2", "2025-Q3", "2025-Q4", "2026-Q1", "2026-Q2", "2026-Q3"}
		months := []string{"2025-04", "2025-07", "2025-10", "2026-01", "2026-04", "2026-07"}
		for i, period := range periods {
			seed("walmart-shop-001", "US", period, period, period, months[i], int64(i+1)*10)
		}
		seed("walmart", "US", "2026-Q3", "2026-Q3", "2026-Q3", "2026-07", 60)
		// Same transaction in a different file must not reappear via snapshots.
		seed("TVOFEK", "US", "overlap", "2026-Q3", "2026-Q3", "2026-07", 60)
		seed("other-store", "US", "2026-Q3", "2026-Q3", "2026-Q3", "2026-07", 90)
		seed("TVOFEK", "CA", "2026-Q3", "2026-Q3", "2026-Q3", "2026-07", 30)
		oldDB := database.DB
		database.DB = tx
		defer func() { database.DB = oldDB }()
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.GET("/quarters", ListTaxQuarters)
		for _, tc := range []struct {
			store, site string
			quarters    int
			total       int64
			raw         int64
		}{
			{"TVOFEK", "US", 6, 210, 6}, {"walmart-shop-001", "US", 6, 210, 6},
			{"TVOFEK", "", 6, 240, 7}, {"", "US", 6, 300, 7}, {"other-store", "US", 1, 90, 1},
		} {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/quarters?store_id="+tc.store+"&site_id="+tc.site, nil))
			if w.Code != 200 {
				t.Fatalf("quarters: %s", w.Body.String())
			}
			var response struct {
				Items []taxQuarterSummary `json:"items"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if len(response.Items) != tc.quarters {
				t.Fatalf("%s/%s got %d quarters", tc.store, tc.site, len(response.Items))
			}
			total := decimal.Zero
			for _, row := range response.Items {
				total = total.Add(row.USDTaxableIncome)
				monthly := decimal.Zero
				for _, month := range row.Months {
					monthly = monthly.Add(month.USDTaxableIncome)
				}
				if !monthly.Equal(row.USDTaxableIncome) {
					t.Fatalf("monthly mismatch %s", row.Quarter)
				}
			}
			if !total.Equal(decimal.NewFromInt(tc.total)) {
				t.Fatalf("%s/%s total %s want %d", tc.store, tc.site, total, tc.total)
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/raw?store_id="+tc.store+"&site_id="+tc.site, nil)
			var count int64
			if err := taxRawBaseQuery(c).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != tc.raw {
				t.Fatalf("raw count %d want %d", count, tc.raw)
			}
		}
	})
}
