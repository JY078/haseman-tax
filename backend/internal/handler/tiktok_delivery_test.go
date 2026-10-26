package handler

import (
	"bytes"
	"encoding/csv"
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

func TestTikTokDeliverySummaryIntegration(t *testing.T) {
	dsn := os.Getenv("TAX_TEST_DSN")
	if dsn == "" {
		t.Skip("set TAX_TEST_DSN for isolated PostgreSQL integration")
	}
	if !strings.Contains(dsn, "host=127.0.0.1") {
		t.Fatal("integration database must be local")
	}
	root, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := root.DB()
	defer sqlDB.Close()
	tx := root.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	old := database.DB
	database.DB = tx
	defer func() { database.DB = old }()
	schema := fmt.Sprintf("tiktok_delivery_%d", time.Now().UnixNano())
	if err := tx.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.AutoMigrate(&model.TikTokOrder{}, &model.TikTokSettlement{}, &model.TikTokTaxRawRow{}, &model.TikTokTaxDetail{}, &model.TikTokTaxAd{}, &model.TikTokTaxFBT{}, &model.TaxExchangeRate{}, &model.TaxDownloadLog{}, &model.User{}); err != nil {
		t.Fatal(err)
	}
	create := func(value interface{}) {
		t.Helper()
		if err := tx.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, rule := range []struct{ id, kind, option string }{{"mcf", "商家物流", ""}, {"fbt", "TikTok 物流", "Fulfilled by TikTok Shop 标准"}, {"overseas", "TikTok 物流", "标准运输"}} {
		// Every order has two SKU rows. Fees still must only be counted once.
		for i := 0; i < 2; i++ {
			quantity := map[string]string{"mcf": "2", "fbt": "3", "overseas": "4"}[rule.id]
			create(&model.TikTokOrder{OrderID: rule.id, Month: "2026-01", Quarter: "2026-Q1", Data: map[string]string{"Delivery Option Type": rule.kind, "Delivery Option": rule.option, "Normal or Pre-order": "Normal", "SKU ID": "sku-" + rule.id, "Quantity": quantity}})
		}
	}
	settlement := func(id, date, currency, field, amount string) {
		data := map[string]string{"Type": "Order", "Order/adjustment ID": id, "Seller shipping fee": "900", "FBT fulfillment fee": "900", "TikTok Shop shipping fee": "900", "Referral fee": "1", "Refund administration fee": "0.2"}
		data[field] = amount
		data["Subtotal after seller discounts"] = "1.005"
		if field == "Shipping fee" {
			data["Seller shipping fee"] = ""
		} else if field == "Seller shipping fee" {
			data["Shipping fee"] = "999"
		}
		month := date[:7]
		posted := parseTikTokPostedAt(date)
		create(&model.TikTokSettlement{Month: month, Quarter: fmt.Sprintf("%d-Q%d", posted.Year(), (int(posted.Month())+2)/3), StatementDate: date, Currency: currency, Data: data})
	}
	settlement("mcf", "2026-01-02", "USD", "Seller shipping fee", "2.005")
	settlement("mcf", "2026-01-03", "USD", "Shipping fee", "2.005")
	settlement("fbt", "2026-01-02", "USD", "FBT fulfillment fee", "($3.01)")
	settlement("overseas", "2026-01-03", "USD", "TikTok Shop shipping fee", "$4.00")
	settlement("missing", "2026-01-02", "USD", "Seller shipping fee", "999")
	settlement("mcf", "2026-04-01", "USD", "Seller shipping fee", "10")
	settlement("fbt", "2026-04-01", "USD", "FBT fulfillment fee", "20")
	settlement("overseas", "2026-04-01", "USD", "TikTok Shop shipping fee", "")
	settlement("mcf", "2026-07-01", "CNY", "Seller shipping fee", "14.4")
	// A matched blank order type is a sample; repeated settlements count it once.
	create(&model.TikTokOrder{OrderID: "sample", Month: "2026-01", Quarter: "2026-Q1", Data: map[string]string{"Normal or Pre-order": "", "SKU ID": "sample-sku", "Quantity": "5", "Delivery Option Type": "商家物流", "Delivery Option": "标准运输"}})
	for _, date := range []string{"2026-01-02", "2026-01-03", "2026-04-01"} {
		quarter := "2026-Q1"
		if date == "2026-04-01" {
			quarter = "2026-Q2"
		}
		create(&model.TikTokSettlement{Month: date[:7], Quarter: quarter, StatementDate: date, Currency: "USD", Data: map[string]string{"Type": "Order", "Order/adjustment ID": "sample", "SKU ID": "sample-sku"}})
	}
	// Non-order rows must not match even when their order ID and fee fields do.
	for _, kind := range []string{"Adjustment", "Refund", "", "order"} {
		for _, id := range []string{"mcf", "fbt", "overseas", "missing"} {
			create(&model.TikTokSettlement{Month: "2026-01", Quarter: "2026-Q1", StatementDate: "2026-01-02", Currency: "USD", Data: map[string]string{"Type": kind, "Order/adjustment ID": id, "Seller shipping fee": "500", "FBT fulfillment fee": "600", "TikTok Shop shipping fee": "700"}})
		}
	}
	// Delivery total comes only from the FBT report, independently of settlement matches.
	create(&model.TikTokSettlement{Month: "2026-01", Quarter: "2026-Q1", StatementDate: "2026-01-02", Currency: "USD", Data: map[string]string{
		"Type": "Adjustment", "FBT fulfillment fee reimbursement": "$1.25", "Return shipping fee reimbursement": "(2.50)",
		"Shipping protection reimbursement": "3", "FBT overall merchant subsidy": "4", "Logistics reimbursement": "5", "Platform reimbursement": "6",
		"Affiliate Commission": "($1.005)", "Affiliate partner commission": "-2", "Affiliate Shop Ads commission": "-3",
		"Affiliate commission deposit": "-4", "Affiliate commission refund": "0.5", "Affiliate Partner shop ads commission": "-6",
		"Campaign service fee": "($1.005)", "Additional Campaign Package": "-2", "Campaign Package": "3",
		"Smart Promotion fee":                "-4.25",
		"Signature confirmation service fee": "($1.005)", "Signature confirmation servicer fee": "999",
		"Co-funded promotion (seller-funded)": "-2", "Shipping protection service fee": "-3", "Protection service fee": "999",
		"Bill deduction for negative balance": "-4", "Sales proceed (negative balance)": "0.5",
		"Smart Promotion campaign period fee": "($1.005)", "Smart Promotion campaign period fee sales tax": "999",
		"Sales tax": "$1.005", "Sales tax payment": "200", "Sales tax payment (2)": "2", "Sales tax refund": "(3)", "Retail delivery fee": "4", "Retail delivery fee payment": "500", "Retail delivery fee payment (2)": "5", "Retail delivery fee refund": "-6",
	}})
	create(&model.TikTokSettlement{Month: "2026-01", Quarter: "2026-Q1", StatementDate: "2026-01-03", Currency: "USD", Data: map[string]string{"Campaign service fee": "-1.005", "Sales tax": "1.005", "Smart Promotion campaign period fee": "-1.005", "Signature confirmation servicer fee": "-1.005", "Co-funded promotion（seller-funded）": "-2", "Protection service fee": "-3", "Bill deduction for negative balance": "-4", "Sales proceed（negative balance）": "0.5"}})
	create(&model.TikTokSettlement{Month: "2026-04", Quarter: "2026-Q2", StatementDate: "2026-04-01", Currency: "USD", Data: map[string]string{"Campaign Package": "2", "Retail delivery fee refund": "-3", "Smart Promotion campaign period fee": "5", "Bill deduction for negative balance": "2"}})
	create(&model.TikTokTaxFBT{Month: "2026-01", Quarter: "2026-Q1", FBTFulfillmentFee: decimal.NewFromInt(999), FBTFulfillmentFeeReimbursement: decimal.NewFromInt(2), HubPlacementFee: decimal.NewFromInt(3), StorageFee: decimal.NewFromInt(4)})
	commissionRaw := model.TikTokTaxRawRow{BatchID: 1, Fingerprint: "commission-fallback", Data: map[string]string{"币种": "USD", "平台佣金/服务费金额（美元）": "12.34"}}
	create(&commissionRaw)
	create(&model.TikTokTaxDetail{BatchID: 1, RawRowID: commissionRaw.ID, Month: "2026-01", Quarter: "2026-Q1", PostedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Commission: decimal.Zero, Rate: decimal.RequireFromString("7.1")})
	for _, rate := range []struct{ date, value string }{{"2026-01-02", "7.1"}, {"2026-01-05", "7.2"}, {"2026-07-01", "7.2"}} {
		create(&model.TaxExchangeRate{Date: rate.date, Currency: "USD", Rate: decimal.RequireFromString(rate.value)})
	}
	check := func(value *decimal.Decimal, want string) {
		t.Helper()
		if value == nil || !value.Equal(decimal.RequireFromString(want)) {
			t.Fatalf("amount got %v want %s", value, want)
		}
	}
	usd, err := listTikTokQuarterSummary(tx, "USD")
	if err != nil || len(usd) != 3 {
		t.Fatalf("USD: %+v %v", usd, err)
	}
	check(usd[0].DeliveryMCF, "4.01")
	check(usd[0].DeliveryFBT, "-3.01")
	check(usd[0].DeliveryOverseas, "4")
	check(usd[0].Delivery, "999")
	check(usd[0].Commission, "12.34")
	check(usd[0].SettlementCommissionTotal, "8.99")
	check(usd[0].SettlementAffiliateCommission, "-15.51")
	check(usd[0].SettlementCampaignServiceFee, "-1.01")
	check(usd[0].SettlementSmartPromotionFee, "-6.26")
	check(usd[0].SettlementPromotionFee, "0")
	check(usd[0].SettlementOtherFees, "-19.01")
	check(usd[1].SettlementOtherFees, "2")
	check(usd[2].SettlementOtherFees, "0")
	check(usd[1].SettlementSmartPromotionFee, "5")
	check(usd[1].SettlementPromotionFee, "0")
	check(usd[2].SettlementPromotionFee, "0")
	check(usd[0].SettlementTax, "4.01")
	check(usd[1].SettlementCampaignServiceFee, "2")
	check(usd[1].SettlementTax, "-3")
	check(usd[2].SettlementCampaignServiceFee, "0")
	check(usd[2].SettlementTax, "0")
	check(usd[0].AffiliateDirect, "-1.01")
	check(usd[0].AffiliatePartner, "-2")
	check(usd[0].AffiliateShopAds, "-3")
	check(usd[0].AffiliateDeposit, "-4")
	check(usd[0].AffiliateRefund, "0.5")
	check(usd[0].AffiliatePartnerShopAds, "-6")
	if usd[0].SampleCount == nil || *usd[0].SampleCount != 5 || usd[1].SampleCount == nil || *usd[1].SampleCount != 0 {
		t.Fatalf("sample count with repeated settlements: %+v %+v", usd[0].SampleCount, usd[1].SampleCount)
	}
	if usd[0].SalesVolume == nil || *usd[0].SalesVolume != 9 {
		t.Fatalf("sales volume: %+v", usd[0].SalesVolume)
	}
	check(usd[0].OtherIncome, "2")
	check(usd[0].SettlementOtherIncome, "16.75")
	check(usd[0].OtherIncomeFBTReimbursement, "1.25")
	check(usd[0].OtherIncomeReturnShipping, "-2.50")
	check(usd[0].OtherIncomeShippingProtection, "3")
	check(usd[0].OtherIncomeFBTSubsidy, "4")
	check(usd[0].OtherIncomeSettlement, "11")
	check(usd[0].HubFee, "3")
	check(usd[0].StorageFee, "4")
	check(usd[0].TaxableIncome, "5.03")
	check(usd[0].Income, "0")
	check(usd[0].SalesAmount, "5.03")
	check(usd[1].SalesAmount, "3.02")
	if usd[0].DeliveryUnmatchedRows != 1 {
		t.Fatalf("unmatched: %+v", usd[0])
	}
	check(usd[1].DeliveryMCF, "10")
	check(usd[1].DeliveryFBT, "20")
	check(usd[1].DeliveryOverseas, "0")
	if usd[1].SalesVolume == nil || *usd[1].SalesVolume != 0 || usd[2].SalesVolume == nil || *usd[2].SalesVolume != 0 {
		t.Fatalf("sales volume by quarter: %+v %+v", usd[1].SalesVolume, usd[2].SalesVolume)
	}
	check(usd[1].Delivery, "0")
	check(usd[2].DeliveryMCF, "2")
	cny, err := listTikTokQuarterSummary(tx, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	check(cny[0].DeliveryMCF, "28.67")
	check(cny[0].SalesAmount, "35.88")
	if cny[1].SalesAmount != nil {
		t.Fatal("sales amount missing rate must remain pending")
	}
	check(cny[0].SettlementAffiliateCommission, "-110.09")
	check(cny[0].SettlementCampaignServiceFee, "-7.27")
	check(cny[0].SettlementSmartPromotionFee, "-44.55")
	check(cny[0].SettlementPromotionFee, "0")
	check(cny[0].SettlementOtherFees, "-135.92")
	check(cny[0].SettlementTax, "28.57")
	if cny[1].SettlementCampaignServiceFee != nil || cny[1].SettlementTax != nil || cny[1].SettlementOtherFees != nil {
		t.Fatal("missing daily rates must leave campaign fee and tax pending")
	}
	check(cny[0].SettlementCommissionTotal, "64.57")
	check(cny[0].OtherIncome, "14.4")
	check(cny[0].SettlementOtherIncome, "118.93")
	check(cny[0].OtherIncomeFBTReimbursement, "8.88")
	check(cny[0].OtherIncomeReturnShipping, "-17.75")
	check(cny[0].OtherIncomeShippingProtection, "21.3")
	check(cny[0].OtherIncomeFBTSubsidy, "28.4")
	check(cny[0].OtherIncomeSettlement, "78.1")
	check(cny[0].DeliveryFBT, "-21.37")
	check(cny[0].DeliveryOverseas, "28.8")
	check(cny[0].Delivery, "7192.8")
	check(cny[2].Delivery, "0")
	check(cny[1].Delivery, "0")
	if cny[1].DeliveryMCF != nil || cny[1].DeliveryFBT != nil || cny[1].MissingRates != 2 {
		t.Fatalf("missing rate must not become zero: %+v", cny[1])
	}
	check(cny[1].DeliveryOverseas, "0")
	create(&model.TaxExchangeRate{Date: "2026-04-01", Currency: "USD", Rate: decimal.RequireFromString("7.3")})
	cny, err = listTikTokQuarterSummary(tx, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	check(cny[1].Delivery, "0")
	check(cny[1].DeliveryMCF, "73")
	check(cny[1].DeliveryFBT, "146")
	// New matching information updates existing settlement quarters automatically.
	create(&model.TikTokOrder{OrderID: "missing", Month: "2026-01", Quarter: "2026-Q1", Data: map[string]string{"Delivery Option Type": "商家物流", "Delivery Option": "标准运输"}})
	usd, err = listTikTokQuarterSummary(tx, "USD")
	if err != nil {
		t.Fatal(err)
	}
	check(usd[0].DeliveryMCF, "1003.01")
	check(usd[0].Delivery, "999")
	if usd[0].DeliveryUnmatchedRows != 0 {
		t.Fatal("match not updated")
	}
	if err := tx.Where("date = ? AND currency = ?", "2026-01-02", "USD").Delete(&model.TaxExchangeRate{}).Error; err != nil {
		t.Fatal(err)
	}
	cny, err = listTikTokQuarterSummary(tx, "CNY")
	if err != nil {
		t.Fatal(err)
	}
	check(cny[0].Delivery, "7192.8")
	// Dates before the first publication still use the next nearby workday.
	check(cny[0].DeliveryMCF, "7221.67")
	check(cny[0].DeliveryFBT, "-21.67")
	check(cny[0].OtherIncomeSettlement, "79.2")
	check(cny[0].SettlementAffiliateCommission, "-111.64")
	check(cny[0].AffiliateDirect, "-7.24")
	check(cny[0].SettlementCommissionTotal, "7257.53")
	check(cny[0].OtherIncome, "14.4")
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/quarters", ListTikTokTaxQuarters)
	user := model.User{Username: "delivery-test", Name: "财务测试"}
	create(&user)
	router.GET("/export", func(c *gin.Context) { c.Set("userID", user.ID); ExportTikTokTaxReport(c) })
	out := httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest("GET", "/quarters?currency=USD", nil))
	var response struct {
		Items []tikTokQuarterSummary `json:"items"`
	}
	if err := json.Unmarshal(out.Body.Bytes(), &response); err != nil || out.Code != 200 || len(response.Items) != 3 {
		t.Fatalf("endpoint %s", out.Body.String())
	}
	check(response.Items[0].DeliveryMCF, "1003.01")
	check(response.Items[0].OtherIncome, "2")
	check(response.Items[0].SettlementOtherIncome, "16.75")
	out = httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest("GET", "/export?type=quarters&currency=USD&quarter=2026-Q1", nil))
	records, err := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(out.Body.Bytes(), []byte{0xef, 0xbb, 0xbf}))).ReadAll()
	if err != nil || out.Code != 200 || len(records) != 2 || len(records[0]) != 16 || records[0][2] != "销售额（USD）" || records[0][4] != "退款费（USD）" || records[1][2] != "0.00" || records[1][14] != "5.03" {
		t.Fatalf("export: %v %v", records, err)
	}
	if records[0][7] != "其他收入（USD）" || records[1][7] != "2.00" {
		t.Fatalf("other income export must preserve the FBT total: %v", records)
	}
	create(&model.TikTokTaxDetail{RawRowID: 9001, Month: "2026-01", Quarter: "2026-Q1", PostedAt: parseTikTokPostedAt("2026-01-03"), Income: decimal.NewFromInt(100), Refund: decimal.NewFromInt(10)})
	updated, err := listTikTokQuarterSummary(tx, "USD")
	if err != nil {
		t.Fatal(err)
	}
	check(updated[0].Income, "100")
	check(updated[0].SalesAmount, "5.03")
	check(updated[0].TaxableIncome, "-4.98")
	if os.Getenv("TAX_PERF_CHECK") == "1" {
		if err := tx.Exec(`INSERT INTO tik_tok_orders (batch_id, order_id, month, quarter, data)
 SELECT 0, 'perf-' || n, '2026-01', '2026-Q1', jsonb_build_object('SKU ID','sku', 'Quantity','2', 'Normal or Pre-order','Normal', 'Delivery Option Type','商家物流') FROM generate_series(1, 3000) n`).Error; err != nil {
			t.Fatal(err)
		}
		if err := tx.Exec(`INSERT INTO tik_tok_settlements (batch_id,month,quarter,statement_date,currency,data)
 SELECT 0,'2026-01','2026-Q1','2026-01-03','USD', jsonb_build_object('Type','Order','Order/adjustment ID','perf-' || (n % 3000 + 1),'SKU ID','sku','Subtotal after seller discounts','100.005','Seller shipping fee','-3.005','Referral fee','-10','Affiliate Commission','-2.005','Shipping protection reimbursement','1.005','Sales tax','2') FROM generate_series(1, 30000) n`).Error; err != nil {
			t.Fatal(err)
		}
		if err := tx.Exec("ANALYZE tik_tok_orders; ANALYZE tik_tok_settlements").Error; err != nil {
			t.Fatal(err)
		}
		for _, currency := range []string{"USD", "CNY"} {
			start := time.Now()
			result, err := listTikTokQuarterSummary(tx, currency)
			if err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(result)
			t.Logf("PERF %s %s RESULT %s", currency, time.Since(start), encoded)
		}
	}

}
