package handler

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSaveBuyerReviewsDeduplicatesNormalizedLinks(t *testing.T) {
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
	oldDB := database.DB
	database.DB = tx
	defer func() { tx.Rollback(); database.DB = oldDB }()
	schema := fmt.Sprintf("buyer_review_import_%d", time.Now().UnixNano())
	for _, sql := range []string{"CREATE SCHEMA " + schema, "SET LOCAL search_path TO " + schema} {
		if err := tx.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.AutoMigrate(&model.BuyerReview{}); err != nil {
		t.Fatal(err)
	}

	plain := "https://www.amazon.com/gp/customer-reviews/R1"
	variants := []string{
		plain,
		plain + "?filterByStar=five_star",
		"http://WWW.AMAZON.COM/gp/customer-reviews/R1/",
	}
	images := "https://example.com/a.jpg"
	input := make([]buyerReviewInput, 0, len(variants))
	for index, link := range variants {
		row := buyerReviewInput{Brand: "Forious", Country: "美国", ChildASIN: "B0TEST", Rating: 5, Name: "买家", Date: "2026-10-23", Title: fmt.Sprintf("标题%d", index), Content: "内容", Link: link}
		if index == 1 {
			row.ReviewImages = &images
		}
		input = append(input, row)
	}
	if err := saveBuyerReviews(input); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := tx.Model(&model.BuyerReview{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("normalized links must collapse to one row: count=%d error=%v", count, err)
	}
	var saved model.BuyerReview
	if err := tx.First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Title != "标题2" || saved.ReviewImages != images {
		t.Fatalf("last import must win with merged images: %+v", saved)
	}

	reimport := []buyerReviewInput{{Brand: "Forious", Country: "美国", ChildASIN: "B0TEST", Rating: 4, Name: "买家", Date: "2026-10-23", Title: "更新标题", Content: "内容", Link: plain}}
	if err := saveBuyerReviews(reimport); err != nil {
		t.Fatal(err)
	}
	if err := tx.Model(&model.BuyerReview{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("reimport without query params must update the same row: count=%d error=%v", count, err)
	}
	if err := tx.First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Title != "更新标题" || saved.Rating != 4 {
		t.Fatalf("reimport must overwrite the existing row: %+v", saved)
	}
}
