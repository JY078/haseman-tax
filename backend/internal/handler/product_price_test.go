package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestProductPriceValidation(t *testing.T) {
	for _, tc := range []struct {
		raw, want string
		valid     bool
	}{
		{"", "", true}, {"  ", "", true}, {"0", "0.00", true}, {" 39.9 ", "39.90", true},
		{"99999999.99", "99999999.99", true}, {"-1", "", false}, {"1.999", "", false},
		{"100000000", "", false}, {"NaN", "", false}, {"1e2", "", false}, {"1,000", "", false},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			got, _, err := normalizeProductPrice(&tc.raw)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
			if tc.valid && tc.want == "" && got != nil {
				t.Fatal("blank must remain NULL")
			}
			if tc.valid && tc.want != "" && (got == nil || *got != tc.want) {
				t.Fatalf("want %s, got %v", tc.want, got)
			}
		})
	}
	for _, body := range []string{`{"price":null,"sale_price":"1"}`, `{"price":"5","sale_price":"6"}`} {
		var input productPriceInput
		json.Unmarshal([]byte(body), &input)
		if validateProductPriceInput(&input, "amazon") == nil {
			t.Fatal("invalid discount accepted")
		}
	}
	for _, pid := range []string{"1729384756123456789", "", "abc", "https://www.tiktok.com/view/product/123", strings.Repeat("1", 33)} {
		input := productPriceInput{PID: pid}
		err := validateProductPriceInput(&input, "tiktok")
		if (err == nil) != (pid == "" || pid == "1729384756123456789") {
			t.Fatalf("unexpected PID validation: %q %v", pid, err)
		}
	}
}

// Uses a local database only, with every fixture and schema change rolled back.
func TestProductPriceIntegration(t *testing.T) {
	envFile := os.Getenv("PRODUCT_PRICE_TEST_ENV_FILE")
	if envFile == "" {
		t.Skip("set PRODUCT_PRICE_TEST_ENV_FILE to a local PostgreSQL .env file")
	}
	settings := viper.New()
	settings.SetConfigFile(envFile)
	if err := settings.ReadInConfig(); err != nil {
		t.Fatal("cannot read test database configuration")
	}
	host := settings.GetString("DB_HOST")
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		t.Fatal("integration test requires a local database")
	}
	dsn := url.URL{Scheme: "postgres", User: url.UserPassword(settings.GetString("DB_USER"), settings.GetString("DB_PASSWORD")), Host: host + ":" + settings.GetString("DB_PORT"), Path: "/" + settings.GetString("DB_NAME")}
	dsn.RawQuery = url.Values{"sslmode": {settings.GetString("DB_SSLMODE")}}.Encode()
	db, err := gorm.Open(postgres.Open(dsn.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("cannot connect to local test database")
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	previous := database.DB
	database.DB = tx
	t.Cleanup(func() { tx.Rollback(); database.DB = previous })
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	schema := fmt.Sprintf("product_price_test_%d", time.Now().UnixNano())
	check(tx.Exec("CREATE SCHEMA " + schema).Error)
	check(tx.Exec("SET LOCAL search_path TO " + schema).Error)
	check(tx.AutoMigrate(&model.User{}, &model.UserOrderPlatform{}, &model.ProductSKU{}, &model.OrderSKUMapping{}, &model.TikTokSKUPrice{}))
	check(tx.Exec("CREATE TABLE product_spus (id bigint PRIMARY KEY, spu varchar(120) NOT NULL, name varchar(255) NOT NULL DEFAULT '')").Error)
	check(tx.Exec("INSERT INTO product_spus (id, spu) VALUES (1, 'US-SPU'), (2, 'CA-100'), (3, ' eu-100 '), (4, 'uk100')").Error)
	downMigration, err := os.ReadFile("../../migrations/000067_add_product_prices.down.sql")
	check(err)
	check(tx.Exec(string(downMigration)).Error)
	migration, err := os.ReadFile("../../migrations/000067_add_product_prices.up.sql")
	check(err)
	check(tx.Exec(string(migration)).Error)
	check(tx.Exec(string(migration)).Error)
	users := []model.User{{Username: "tk", Platform: "TIKTOK", Email: "tk@example.test"}, {Username: "temu", Platform: "TEMU", Email: "temu@example.test"}}
	check(tx.Create(&users).Error)
	products := []model.ProductSKU{{SKU: "AM-A", ASIN: "B012345678", Name: "Faucet"}, {SKU: "AM-B", ASIN: "B012345679"}, {SKU: "AM-C"}}
	for i := range products {
		products[i].SPUID = 1
		products[i].SPU = "US-SPU"
	}
	check(tx.Create(&products).Error)
	orphans := []model.ProductSKU{
		{SKU: "ORPHAN-ZERO", SPUID: 0, SPU: "CATALOG-SPU"},
		{SKU: "ORPHAN-MISSING", SPUID: 999, SPU: "CATALOG-SPU"},
	}
	check(tx.Create(&orphans).Error)
	excluded := []model.ProductSKU{{SKU: "CA-SKU", SPUID: 2}, {SKU: "EU-SKU", SPUID: 3}, {SKU: "UK-SKU", SPUID: 4}}
	check(tx.Create(&excluded).Error)
	mappings := []model.OrderSKUMapping{
		{Platform: "TIKTOK", PlatformSKU: "TK-A", AmazonSKU: "AM-A", IsCurrent: true, Status: "active"},
		{Platform: "TIKTOK", PlatformSKU: "TK-B", AmazonSKU: "AM-A", IsCurrent: true, Status: "active"},
		{Platform: "TIKTOK", PlatformSKU: "TK-A", AmazonSKU: "AM-B", IsCurrent: false, Status: "active"},
	}
	check(tx.Create(&mappings).Error)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if id, _ := strconv.Atoi(c.GetHeader("X-Test-User")); id > 0 {
			c.Set("userID", uint(id))
		}
		c.Next()
	})
	router.GET("/prices", ListProductPrices)
	router.PUT("/prices/:platform/:id", UpdateProductPrice)
	router.PUT("/mappings", UpsertOrderSKUMapping)
	router.GET("/mappings/group", GetOrderSKUMappingGroup)
	send := func(user uint, method, path, body string, status int) []byte {
		t.Helper()
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User", strconv.Itoa(int(user)))
		router.ServeHTTP(recorder, req)
		if recorder.Code != status {
			t.Fatalf("%s %s: got %d want %d: %s", method, path, recorder.Code, status, recorder.Body.String())
		}
		return recorder.Body.Bytes()
	}
	list := func(query string, want int64) []productPriceRow {
		t.Helper()
		var result struct {
			Items []productPriceRow `json:"items"`
			Total int64             `json:"total"`
		}
		check(json.Unmarshal(send(users[0].ID, http.MethodGet, "/prices"+query, "", 200), &result))
		if result.Total != want {
			t.Fatalf("query=%s total=%d want=%d", query, result.Total, want)
		}
		return result.Items
	}
	amazon := fmt.Sprintf("/prices/amazon/%d", products[0].ID)
	tiktok := fmt.Sprintf("/prices/tiktok/%d", mappings[0].ID)
	send(0, http.MethodGet, "/prices", "", 401)
	send(0, http.MethodPut, amazon, `{"price":"20"}`, 401)
	send(users[1].ID, http.MethodPut, tiktok, `{"price":"20"}`, 403)
	send(users[0].ID, http.MethodPut, amazon, `{"price":"10","sale_price":"20"}`, 400)
	list("", 4)
	list("?q=ORPHAN", 0)
	list("?q=CA-SKU", 0)
	list("?q=EU-SKU", 0)
	list("?q=UK-SKU", 0)
	var catalog []model.ProductSPU
	check(tx.Where("id = ?", 1).Preload("SKUs").Find(&catalog).Error)
	catalogIDs := map[uint]bool{}
	for _, spu := range catalog {
		for _, sku := range spu.SKUs {
			catalogIDs[sku.ID] = true
		}
	}
	priceIDs := map[uint]bool{}
	for _, row := range list("", 4) {
		if !catalogIDs[row.ID] {
			t.Fatalf("price row %d is not visible in product catalog", row.ID)
		}
		priceIDs[row.ID] = true
	}
	if len(priceIDs) != len(catalogIDs) {
		t.Fatal("price list and product catalog contain different SKU sets")
	}
	if len(list("?page_size=1&page=2", 4)) != 1 {
		t.Fatal("pagination failed")
	}
	list("?filter=unmapped", 2)
	send(users[0].ID, http.MethodPut, amazon, `{"price":"39.99","sale_price":"29.99"}`, 200)
	send(users[0].ID, http.MethodPut, tiktok, `{"price":"0","pid":"1729384756123456789"}`, 200)
	rows := list("?q=AM-A", 2)
	for _, row := range rows {
		if row.AmazonPrice == nil || *row.AmazonPrice != "39.99" || row.PriceUpdatedAt == nil {
			t.Fatal("Amazon price not shared across mappings")
		}
	}
	rows = list("?q=1729384756123456789", 1)
	if rows[0].PID == nil || *rows[0].PID != "1729384756123456789" || rows[0].TikTokPrice == nil || *rows[0].TikTokPrice != "0.00" {
		t.Fatalf("PID or zero price lost precision: %s", send(users[0].ID, http.MethodGet, "/prices?q=1729384756123456789", "", 200))
	}
	firstTikTokTime := *rows[0].TikTokUpdatedAt
	list("?filter=sale", 2)
	list("?filter=missing", 3)
	send(users[0].ID, http.MethodPut, amazon, `{"price":"45.50","sale_price":null}`, 200)
	rows = list("?q=TK-A", 1)
	if !rows[0].TikTokUpdatedAt.Equal(firstTikTokTime) {
		t.Fatal("Amazon save changed TikTok timestamp")
	}
	list("?filter=sale", 0)
	check(tx.Model(&model.ProductSKU{}).Where("id = ?", products[0].ID).Update("asin", "B098765432").Error)
	list("?q=B098765432", 2)
	check(tx.Model(&model.OrderSKUMapping{}).Where("id = ?", mappings[0].ID).Update("is_current", false).Error)
	check(tx.Model(&model.OrderSKUMapping{}).Where("id = ?", mappings[2].ID).Update("is_current", true).Error)
	rows = list("?q=TK-A", 1)
	if rows[0].SKU != "AM-B" || rows[0].PID == nil || *rows[0].PID != "1729384756123456789" {
		t.Fatal("current mapping did not preserve TikTok metadata")
	}
	send(users[0].ID, http.MethodPut, tiktok, `{"price":"1","pid":"123"}`, 404)
	// AM-B is mapped after switching, while the alphabetically earlier AM-A
	// becomes unmapped once its remaining current mapping is deactivated.
	check(tx.Model(&model.OrderSKUMapping{}).Where("id = ?", mappings[1].ID).Updates(map[string]interface{}{"status": "inactive", "is_current": false}).Error)
	for index, want := range []string{"AM-B", "AM-A", "AM-C"} {
		pageRows := list(fmt.Sprintf("?page_size=1&page=%d", index+1), 3)
		if len(pageRows) != 1 || pageRows[0].SKU != want {
			t.Fatalf("mapped-first pagination: page %d expected %s, got %+v", index+1, want, pageRows)
		}
	}
	check(tx.Model(&model.OrderSKUMapping{}).Where("id = ?", mappings[1].ID).Updates(map[string]interface{}{"status": "active", "is_current": true}).Error)
	send(users[0].ID, http.MethodPut, fmt.Sprintf("/prices/tiktok/%d", mappings[2].ID), `{"price":null,"pid":""}`, 200)
	rows = list("?q=TK-A", 1)
	if rows[0].TikTokPrice != nil || (rows[0].PID != nil && *rows[0].PID != "") {
		t.Fatal("cleared price or PID was retained")
	}
	send(users[0].ID, http.MethodPut, "/prices/amazon/999999", `{"price":"1"}`, 404)
	check(tx.Exec("UPDATE product_spus SET name = 'Catalog product' WHERE id = 1").Error)
	rows = list("?q=Catalog", 2)
	for _, row := range rows {
		if row.Name != "Catalog product" {
			t.Fatal("missing SKU name must use catalog name")
		}
	}
	send(users[1].ID, http.MethodPut, "/mappings", `{"platform":"TIKTOK","platform_sku":"TK-NEW","amazon_sku":"AM-C","is_current":true}`, 403)
	send(users[0].ID, http.MethodPut, "/mappings", `{"platform":"TIKTOK","platform_sku":"TK-A","amazon_sku":"AM-C","is_current":true,"only_unmapped":true}`, 409)
	send(users[0].ID, http.MethodPut, "/mappings", `{"platform":"TIKTOK","platform_sku":"TK-NEW","amazon_sku":"AM-C","non_amazon_sku":"LOCAL-C","is_current":true,"only_unmapped":true}`, 200)
	send(users[0].ID, http.MethodPut, "/mappings", `{"platform":"TIKTOK","platform_sku":"TK-EXTRA","amazon_sku":"AM-C","is_current":true,"only_unmapped":true}`, 409)
	send(users[0].ID, http.MethodPut, "/mappings", `{"platform":"TIKTOK","platform_sku":"TK-NEW","amazon_sku":"AM-C","remark":"overwrite","is_current":true,"only_unmapped":true}`, 409)
	rows = list("?q=TK-NEW", 1)
	if rows[0].SKU != "AM-C" || rows[0].MappingID == nil {
		t.Fatal("shared mapping not visible in prices")
	}
	var mappingGroup struct {
		Items []model.OrderSKUMapping `json:"items"`
	}
	check(json.Unmarshal(send(users[0].ID, http.MethodGet, "/mappings/group?platform=TIKTOK&platform_sku=TK-NEW", "", 200), &mappingGroup))
	if len(mappingGroup.Items) != 1 || mappingGroup.Items[0].ID != *rows[0].MappingID {
		t.Fatal("mapping and price views do not share the same record")
	}
	if mappingGroup.Items[0].NonAmazonSKU != "LOCAL-C" || mappingGroup.Items[0].Remark != "" {
		t.Fatal("create-only requests overwrote mapping details")
	}

	// A verified scrape writes prices and timestamp, but a stale scrape cannot
	// overwrite a newer edit. Amazon price fetching also works without a mapping.
	fetchItem := list("?q=AM-A", 1)[0]
	fetchedPrice := "45.99"
	fetched := productPriceInput{Price: &fetchedPrice}
	result := saveFetchedAmazonPrice(fetchItem, fetched, users[0].ID, time.Now())
	check(result.Error)
	if result.RowsAffected != 1 {
		t.Fatal("scraped price was not saved")
	}
	result = saveFetchedAmazonPrice(fetchItem, fetched, users[0].ID, time.Now())
	check(result.Error)
	if result.RowsAffected != 0 {
		t.Fatal("stale scrape overwrote a newer price")
	}
	fetchItem = list("?q=AM-A", 1)[0]
	if fetchItem.AmazonPrice == nil || *fetchItem.AmazonPrice != fetchedPrice || fetchItem.AmazonSalePrice != nil || fetchItem.PriceUpdatedAt == nil {
		t.Fatal("scrape fields not persisted")
	}
	check(tx.Model(&model.OrderSKUMapping{}).Where("amazon_sku = ?", fetchItem.SKU).Update("status", "inactive").Error)
	result = saveFetchedAmazonPrice(fetchItem, fetched, users[0].ID, time.Now())
	check(result.Error)
	if result.RowsAffected != 1 {
		t.Fatal("scrape should update an unmapped product")
	}
}
