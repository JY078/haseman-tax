package database

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"advertising-image/backend/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAmazonFinanceLegacyFingerprintIndexRemoval(t *testing.T) {
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
	conn, _ := db.DB()
	defer conn.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	schema := fmt.Sprintf("amazon_finance_migration_%d", time.Now().UnixNano())
	for _, sql := range []string{"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema} {
		if err := tx.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.AutoMigrate(&model.AmazonTaxFinance{}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`CREATE TABLE unrelated_imports (id bigserial PRIMARY KEY, fingerprint text UNIQUE);
		CREATE UNIQUE INDEX amazon_finance_site_identity ON amazon_tax_finances (store_id, site_id, fingerprint);
		ALTER TABLE amazon_tax_finances ADD CONSTRAINT historical_finance_unique UNIQUE (month, fingerprint);
		CREATE UNIQUE INDEX idx_amazon_finance_identity_v2 ON amazon_tax_finances ((data->>'total'))`).Error; err != nil {
		t.Fatal(err)
	}
	if err := dropAmazonFinanceDedupIndexes(tx); err != nil {
		t.Fatal(err)
	}
	for _, legacySQL := range []string{
		`CREATE UNIQUE INDEX idx_amazon_tax_finances_fingerprint ON amazon_tax_finances (fingerprint)`,
		`ALTER TABLE amazon_tax_finances ADD CONSTRAINT idx_amazon_tax_finances_fingerprint UNIQUE (fingerprint)`,
		`CREATE UNIQUE INDEX amazon_finance_site_identity ON amazon_tax_finances (store_id, site_id, fingerprint)`,
		`ALTER TABLE amazon_tax_finances ADD CONSTRAINT amazon_finance_site_identity UNIQUE (store_id, site_id, fingerprint)`,
	} {
		if err := tx.Exec(legacySQL).Error; err != nil {
			t.Fatal(err)
		}
		for range 2 {
			if err := dropAmazonFinanceDedupIndexes(tx); err != nil {
				t.Fatal(err)
			}
		}
	}
	for range 2 {
		row := model.AmazonTaxFinance{StoreID: "Qfn", SiteID: "CA", ImportLogID: 1, Month: "2026-07", Quarter: "2026-Q3", Fingerprint: "same-row", Data: map[string]string{"total": "10"}}
		if err := tx.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := dropAmazonFinanceDedupIndexes(tx); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := tx.Model(&model.AmazonTaxFinance{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("repeated finance rows must survive migration: count=%d error=%v", count, err)
	}
	var indexes struct{ Primary, Ordinary, Unrelated int64 }
	if err := tx.Raw(`SELECT
		(SELECT COUNT(*) FROM pg_index WHERE indrelid = 'amazon_tax_finances'::regclass AND indisprimary) AS primary,
		(SELECT COUNT(*) FROM pg_index WHERE indrelid = 'amazon_tax_finances'::regclass AND NOT indisunique) AS ordinary,
		(SELECT COUNT(*) FROM pg_constraint WHERE conrelid = 'unrelated_imports'::regclass AND contype = 'u') AS unrelated`).Scan(&indexes).Error; err != nil {
		t.Fatal(err)
	}
	if indexes.Primary != 1 || indexes.Ordinary == 0 || indexes.Unrelated != 1 {
		t.Fatalf("migration changed primary, ordinary or unrelated indexes: %+v", indexes)
	}
}

func TestTikTokImportLogBackfill(t *testing.T) {
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
	prior := DB
	DB = tx
	defer func() { tx.Rollback(); DB = prior }()
	schema := fmt.Sprintf("tax_audit_migration_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.TikTokTaxBatch{}, &model.TikTokTaxAdBatch{}, &model.TaxImportLog{}); err != nil {
		t.Fatal(err)
	}
	detail := model.TikTokTaxBatch{FileHash: "detail", FileName: "detail.csv", FileContent: []byte("original details"), RawRows: 2}
	ad := model.TikTokTaxAdBatch{FileHash: "ad", FileName: "ad.csv", FileContent: []byte("original ads"), RowCount: 3}
	if err := tx.Create(&detail).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Create(&ad).Error; err != nil {
		t.Fatal(err)
	}
	walmart := model.TaxImportLog{BatchID: detail.ID, Status: "locked", FileName: "walmart.csv"}
	if err := tx.Create(&walmart).Error; err != nil {
		t.Fatal(err)
	}
	runTaxDataAutoMigrate()
	if err := tx.Model(&model.TaxImportLog{}).Where("platform = ?", "tiktok").Update("importer", "旧导入者").Error; err != nil {
		t.Fatal(err)
	}
	runTaxDataAutoMigrate()
	var logs []model.TaxImportLog
	if err := tx.Where("platform = ?", "tiktok").Order("report_type").Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 || logs[0].ReportType != "ads" || logs[0].ImportedRows != 3 || string(logs[0].FileContent) != "original ads" || logs[1].ReportType != "details" || logs[1].ImportedRows != 2 || logs[1].Importer != "RPA" {
		t.Fatalf("incorrect backfill: %#v", logs)
	}
	var count int64
	if err := tx.Model(&model.TaxImportLog{}).Where("platform = ?", "walmart").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("Walmart logs changed: %d, %v", count, err)
	}
}
