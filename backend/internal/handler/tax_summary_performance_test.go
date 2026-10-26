package handler

import (
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTaxSummaryPerformance(t *testing.T) {
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
	defer tx.Rollback()
	old := database.DB
	database.DB = tx
	defer func() { database.DB = old }()
	schema := fmt.Sprintf("tax_summary_perf_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.TaxRawRow{}, &model.TaxProcessedRow{}, &model.TaxExchangeRate{}, &model.TaxQuarter{}, &model.TaxDataBatch{}, &model.TaxFeeRule{}); err != nil {
		t.Fatal(err)
	}
	size := 300
	if os.Getenv("TAX_PERF_CHECK") == "1" {
		size = 30000
	}
	for _, statement := range []string{
		`INSERT INTO tax_raw_rows(batch_id,store_id,fingerprint,data) SELECT 1,'shop',n::text,jsonb_build_object('Currency','USD','Transaction Posted Timestamp','2026-01-03','Ship Qty','2','Partner Item Name',repeat('product description ',80),'Transaction Type','Sale') FROM generate_series(1, ?) n`,
		`INSERT INTO tax_processed_rows(batch_id,raw_row_id,month,quarter,fee_type,fee,amount,rate,currency) SELECT 1,id,'2026-01','2026-Q1','销售额',70.04,10.005,7,'USD' FROM tax_raw_rows`,
		`INSERT INTO tax_exchange_rates(date,currency,rate) VALUES ('2026-01-05','USD',7.1457)`,
	} {
		args := []interface{}{}
		if strings.Contains(statement, "?") {
			args = append(args, size)
		}
		if err := tx.Exec(statement, args...).Error; err != nil {
			t.Fatal(err)
		}
	}
	start := time.Now()
	result, _, err := summarizeTaxProcessed(tx, "", "")
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if len(result) != 1 || !result[0].SalesVolume.Equal(decimal.NewFromInt(int64(size*2))) || !result[0].TaxableIncome.Equal(decimal.RequireFromString("10.005").Mul(decimal.RequireFromString("7.1457")).Mul(decimal.NewFromInt(int64(size))).Round(2)) {
		t.Fatalf("incorrect summary: %+v", result)
	}
	encoded, _ := json.Marshal(result)
	t.Logf("PERF %s RESULT %s", elapsed, encoded)
	// Mixed historical storage must still count each batch exactly once.
	for _, snapshot := range []model.TaxQuarter{
		{BatchID: 1, Quarter: "2026-Q1", Totals: map[string]decimal.Decimal{"销售额": decimal.NewFromInt(999)}},
		{BatchID: 2, Quarter: "2026-Q1", Totals: map[string]decimal.Decimal{"销售额": decimal.NewFromInt(5)}},
	} {
		if err := tx.Create(&snapshot).Error; err != nil {
			t.Fatal(err)
		}
	}
	batch := model.TaxDataBatch{ID: 3, StoreID: "legacy", FileHash: "legacy", Rates: map[string]model.TaxRate{"USD": {Rate: decimal.NewFromInt(6)}}}
	if err := tx.Create(&batch).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Create(&model.TaxRawRow{BatchID: 3, StoreID: "legacy", Fingerprint: "legacy", Data: taxTestRow("legacy", "2026-01-03", "Sale", "2")}).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Create(&model.TaxProcessedRow{BatchID: 4, RawRowID: uint(size + 100), Month: "2026-01", Quarter: "2026-Q1", FeeType: "销售额", Amount: decimal.NewFromInt(1), Rate: decimal.NewFromInt(8), Currency: "USD"}).Error; err != nil {
		t.Fatal(err)
	}
	mixed, _, err := summarizeTaxProcessed(tx, "", "2026-Q1")
	if err != nil || len(mixed) != 1 || !mixed[0].Total.Equal(result[0].Total.Add(decimal.NewFromInt(25))) || !mixed[0].USDTotal.Equal(result[0].USDTotal.Add(decimal.NewFromInt(3))) {
		t.Fatalf("legacy batch fallback changed: %+v %v", mixed, err)
	}
}
