package handler

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTaxMultiPeriodFilters(t *testing.T) {
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
	schema := fmt.Sprintf("tax_period_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.TaxRawRow{}, &model.TaxProcessedRow{}, &model.TaxExchangeRate{}); err != nil {
		t.Fatal(err)
	}
	for i, month := range []string{"2026-01", "2026-02", "2026-04", "2026-07"} {
		raw := model.TaxRawRow{Fingerprint: fmt.Sprint(i), Data: map[string]string{"Transaction Type": "Sale"}}
		if err := tx.Create(&raw).Error; err != nil {
			t.Fatal(err)
		}
		quarter := []string{"2026-Q1", "2026-Q1", "2026-Q2", "2026-Q3"}[i]
		if err := tx.Create(&model.TaxProcessedRow{RawRowID: raw.ID, Month: month, Quarter: quarter, FeeType: "销售额"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/raw", ListTaxRaw)
	r.GET("/options", ListTaxRawFilterOptions)
	for _, test := range []struct {
		filters      string
		total, count int
	}{
		{`{"月份":"2026-01,2026-04"}`, 2, 1},
		{`{"季度":"2026-Q1,2026-Q2"}`, 3, 1},
		{`{"月份":"2026-01,2026-04","季度":"2026-Q2,2026-Q3"}`, 1, 1},
		{`{"月份":"2026-01","季度":"2026-Q2"}`, 0, 0},
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/raw?page_size=1&filters="+url.QueryEscape(test.filters), nil))
		var result struct {
			Items []model.TaxRawRow `json:"items"`
			Total int               `json:"total"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || w.Code != 200 || result.Total != test.total || len(result.Items) != test.count {
			t.Fatalf("%s: %s", test.filters, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/options?field="+url.QueryEscape("月份")+"&filters="+url.QueryEscape(`{"月份":"2026-01","季度":"2026-Q1,2026-Q2"}`), nil))
	var options struct {
		Values []string `json:"values"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &options); err != nil || w.Code != 200 || strings.Join(options.Values, ",") != "2026-01,2026-02,2026-04" {
		t.Fatal(w.Body.String())
	}
}
