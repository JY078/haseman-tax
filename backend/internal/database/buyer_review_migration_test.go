package database

import (
	"crypto/sha256"
	"encoding/hex"
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

// sha256Hex reproduces the legacy fingerprint hashing of raw lowercase links.
func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// TestBuyerReviewFingerprintMigration verifies that rows written with the
// legacy fingerprint (raw lowercase link) collapse into one row once the
// migration recomputes fingerprints with normalized links.
func TestBuyerReviewFingerprintMigration(t *testing.T) {
	dsn := os.Getenv("TAX_TEST_DSN")
	if dsn == "" {
		t.Skip("set TAX_TEST_DSN for isolated PostgreSQL integration")
	}
	if !strings.Contains(dsn, "host=127.0.0.1") {
		t.Fatal("integration database must be local")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("local database connection failed")
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
	schema := fmt.Sprintf("buyer_review_migration_%d", time.Now().UnixNano())
	for _, sql := range []string{"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema} {
		if err := tx.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.AutoMigrate(&model.BuyerReview{}); err != nil {
		t.Fatal(err)
	}

	base := "https://www.amazon.com/gp/customer-reviews/R1"
	legacyFingerprint := func(link string) string {
		return sha256Hex(strings.ToLower(strings.TrimSpace(link)))
	}
	// Rows exactly as the legacy import wrote them: fingerprints from raw
	// lowercase links, so query/case/scheme variants stayed separate.
	legacyRows := []model.BuyerReview{
		{Fingerprint: legacyFingerprint(base), Brand: "", Country: "美国", ChildASIN: "B0TEST", Rating: 5, Name: "买家", ReviewDate: "2026-10-01", Title: "标题", Content: "内容", Link: base, ReviewImages: ""},
		{Fingerprint: legacyFingerprint(base + "?filterByStar=five_star"), Brand: "Forious", Country: "美国", ChildASIN: "B0TEST", Rating: 0, Name: "买家", ReviewDate: "2026-10-02", Title: "标题", Content: "内容", Link: base + "?filterByStar=five_star", ReviewImages: "https://example.com/a.jpg"},
		{Fingerprint: legacyFingerprint("http://WWW.AMAZON.COM/gp/customer-reviews/R1/"), Brand: "Forious", Country: "美国", ChildASIN: "B0TEST", Rating: 0, Name: "买家", ReviewDate: "2026-10-03", Title: "标题", Content: "内容", Link: "http://WWW.AMAZON.COM/gp/customer-reviews/R1/", Video: true, VideoUpload: "/uploads/review-videos/a.mp4"},
	}
	for index := range legacyRows {
		if err := tx.Create(&legacyRows[index]).Error; err != nil {
			t.Fatal(err)
		}
	}
	// An unrelated review must survive untouched.
	other := model.BuyerReview{Fingerprint: model.BuyerReviewFingerprint("https://www.amazon.com/gp/customer-reviews/R2", "B0OTHER", "2026-10-01", "乙", "标题", "内容"), Brand: "Forious", Link: "https://www.amazon.com/gp/customer-reviews/R2"}
	if err := tx.Create(&other).Error; err != nil {
		t.Fatal(err)
	}

	if err := migrateBuyerReviewFingerprints(); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := tx.Model(&model.BuyerReview{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("variants must collapse to one row: count=%d error=%v", count, err)
	}
	var merged model.BuyerReview
	if err := tx.Where("link <> ?", other.Link).Order("id ASC").First(&merged).Error; err != nil {
		t.Fatal(err)
	}
	if merged.Brand != "Forious" || merged.ReviewImages != "https://example.com/a.jpg" || !merged.Video || merged.VideoUpload != "/uploads/review-videos/a.mp4" {
		t.Fatalf("merged row must keep non-empty fields from duplicates: %+v", merged)
	}
	if merged.Fingerprint != model.BuyerReviewFingerprint(base, merged.ChildASIN, merged.ReviewDate, merged.Name, merged.Title, merged.Content) {
		t.Fatalf("surviving fingerprint must be the normalized one: %s", merged.Fingerprint)
	}
	var otherCount int64
	if err := tx.Model(&model.BuyerReview{}).Where("id = ?", other.ID).Count(&otherCount).Error; err != nil || otherCount != 1 {
		t.Fatalf("unrelated review must survive: count=%d error=%v", otherCount, err)
	}
	// Running the migration again must be a no-op.
	if err := migrateBuyerReviewFingerprints(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Model(&model.BuyerReview{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("second run must not change rows: count=%d error=%v", count, err)
	}
}
