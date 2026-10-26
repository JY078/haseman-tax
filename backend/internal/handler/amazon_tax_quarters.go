package handler

import (
	"encoding/json"
	"fmt"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type amazonTaxQuarter struct {
	Quarter                    string                       `json:"quarter"`
	Month                      string                       `json:"month,omitempty"`
	Months                     []amazonTaxQuarter           `gorm:"-" json:"months,omitempty"`
	TaxableIncome              *decimal.Decimal             `json:"taxable_income"`
	SalesVolume                decimal.Decimal              `json:"sales_volume"`
	SalesCategories            []amazonSalesCategory        `gorm:"-" json:"sales_categories"`
	Sales                      *decimal.Decimal             `json:"sales"`
	SalesProduct               *decimal.Decimal             `json:"sales_product"`
	SalesProductTax            *decimal.Decimal             `json:"sales_product_tax"`
	SalesShipping              *decimal.Decimal             `json:"sales_shipping"`
	SalesShippingTax           *decimal.Decimal             `json:"sales_shipping_tax"`
	SalesGiftWrap              *decimal.Decimal             `json:"sales_gift_wrap"`
	SalesGiftWrapTax           *decimal.Decimal             `json:"sales_gift_wrap_tax"`
	SalesPromotion             *decimal.Decimal             `json:"sales_promotion"`
	SalesShipmentPromotion     *decimal.Decimal             `json:"sales_shipment_promotion"`
	SalesConversionIssues      []amazonSalesConversionIssue `gorm:"-" json:"sales_conversion_issues,omitempty"`
	Refund                     *decimal.Decimal             `json:"refund"`
	RefundProductSales         *decimal.Decimal             `json:"refund_product_sales"`
	RefundProductSalesTax      *decimal.Decimal             `json:"refund_product_sales_tax"`
	RefundShippingCredits      *decimal.Decimal             `json:"refund_shipping_credits"`
	RefundShippingTax          *decimal.Decimal             `json:"refund_shipping_tax"`
	RefundGiftWrap             *decimal.Decimal             `json:"refund_gift_wrap"`
	RefundGiftWrapTax          *decimal.Decimal             `json:"refund_gift_wrap_tax"`
	RefundRegulatoryFee        *decimal.Decimal             `json:"refund_regulatory_fee"`
	RefundRegulatoryTax        *decimal.Decimal             `json:"refund_regulatory_tax"`
	RefundPromotions           *decimal.Decimal             `json:"refund_promotions"`
	RefundPromotionsTax        *decimal.Decimal             `json:"refund_promotions_tax"`
	RefundOther                *decimal.Decimal             `json:"refund_other"`
	Tax                        *decimal.Decimal             `json:"tax"`
	TaxMarketplaceWithheld     *decimal.Decimal             `json:"tax_marketplace_withheld"`
	Delivery                   *decimal.Decimal             `json:"delivery"`
	DeliveryFBAFees            *decimal.Decimal             `json:"delivery_fba_fees"`
	DeliveryFulfilmentDisposal *decimal.Decimal             `json:"delivery_fulfilment_disposal"`
	DeliveryTransactionOther   *decimal.Decimal             `json:"delivery_transaction_other"`
	DeliveryFulfilmentStorage  *decimal.Decimal             `json:"delivery_fulfilment_storage"`
	OtherInboundPlacement      *decimal.Decimal             `json:"other_inbound_placement"`
	OtherPremiumServices       *decimal.Decimal             `json:"other_premium_services"`
	Selling                    *decimal.Decimal             `json:"selling"`
	SellingFees                *decimal.Decimal             `json:"selling_fees"`
	Ads                        *decimal.Decimal             `json:"ads"`
	AdsOtherTransactionFees    *decimal.Decimal             `json:"ads_other_transaction_fees"`
	Other                      *decimal.Decimal             `json:"other"`
	OtherTotal                 *decimal.Decimal             `json:"other_total"`
	OtherMarketplaceWithheld   *decimal.Decimal             `json:"other_marketplace_withheld"`
	OtherSubscription          *decimal.Decimal             `json:"other_subscription"`
	Storage                    *decimal.Decimal             `json:"storage"`
	StorageTransaction         *decimal.Decimal             `json:"storage_transaction"`
	StorageInventory           *decimal.Decimal             `json:"storage_inventory"`
	ValidationErrors           []string                     `json:"validation_errors,omitempty" gorm:"-"`
}

type amazonSalesConversionIssue struct {
	ShipmentID     uint   `json:"shipment_id"`
	ShipmentNumber string `json:"shipment_number"`
	SourceDate     string `json:"source_date"`
	RateDate       string `json:"rate_date"`
	Currency       string `json:"currency"`
	Reason         string `json:"reason"`
}

func amazonSiteCurrency(site string) string {
	if database.DB != nil {
		var entry model.TaxCatalogEntry
		if err := database.DB.Where("platform = ? AND kind = ? AND value = ?", "amazon", "site", taxSiteID(site)).First(&entry).Error; err == nil && entry.Currency != "" {
			return entry.Currency
		}
	}
	if strings.EqualFold(strings.TrimSpace(site), "CA") {
		return "CAD"
	}
	return "USD"
}

func amazonCurrencyAllowed(currency, site string) bool {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	return currency == "CNY" || currency == amazonSiteCurrency(site)
}

func amazonCurrencyCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return ""
	}
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return value
}

// Return at most 20 source records for an incomplete quarter; the amount stays
// pending rather than silently omitting records that cannot be converted.
func amazonShipmentConversion(currency string) (string, string) {
	date := amazonDateSQL("s.data->>'配送日期'")
	sourceCurrency := "COALESCE(NULLIF(UPPER(BTRIM(s.data->>'货币')), ''), 'USD')"
	join := taxRateJoinSQL("sx", date, sourceCurrency)
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "CNY" {
		return join, "CASE WHEN " + sourceCurrency + " = 'CNY' THEN 1::numeric ELSE sx.rate END"
	}
	join += " " + taxRateJoinSQL("ux", date, "'"+strings.ReplaceAll(currency, "'", "''")+"'")
	return join, "CASE WHEN " + sourceCurrency + " = '" + strings.ReplaceAll(currency, "'", "''") + "' THEN 1::numeric WHEN " + sourceCurrency + " = 'CNY' THEN 1::numeric / ux.rate ELSE sx.rate / ux.rate END"
}

func amazonSalesConversionIssuesSQL(currency string) string {
	date := amazonDateSQL("s.data->>'配送日期'")
	sourceCurrency := "COALESCE(NULLIF(UPPER(BTRIM(s.data->>'货币')), ''), 'USD')"
	join, factor := amazonShipmentConversion(currency)
	return "SELECT s.id AS shipment_id, COALESCE(s.data->>'货件编号', '') AS shipment_number, " +
		"COALESCE(s.data->>'配送日期', '') AS source_date, COALESCE(" + taxRateDateSQL(date) + ", '') AS rate_date, " + sourceCurrency + " AS currency, " +
		"CASE WHEN (" + date + ") IS NULL THEN '配送日期无法解析' ELSE '配送日期起14天内未找到原币或目标币种的有效汇率' END AS reason " +
		"FROM amazon_tax_shipments s " + join +
		" WHERE s.quarter = ? AND (" + factor + ") IS NULL AND (" +
		amazonSumSQL("商品价格", "商品税", "运费", "运费税", "礼品包装价格", "商品促销折扣", "货件促销折扣") + ") <> 0 ORDER BY s.id LIMIT 20"
}

// Resolve the site currency of a finance row from the tax catalog, mirroring
// amazonSiteCurrency: catalog first, then the CA fallback, then USD.
func amazonFinanceRowCurrencySQL() string {
	siteValue := "COALESCE(NULLIF(UPPER(BTRIM(f.site_id)), ''), 'US')"
	return "COALESCE((SELECT e.currency FROM tax_catalog_entries e WHERE e.platform = 'amazon' AND e.kind = 'site' AND e.value = " + siteValue + " LIMIT 1), CASE WHEN " + siteValue + " = 'CA' THEN 'CAD' ELSE 'USD' END)"
}

// Use PostgreSQL numeric throughout; round only after summing the whole quarter.
func amazonAmountSQL(header string) string {
	key := strings.ReplaceAll(header, "'", "''")
	value := fmt.Sprintf("REGEXP_REPLACE(COALESCE(data->>'%s', ''), '[$￥¥,[:space:]]', '', 'g')", key)
	return fmt.Sprintf("COALESCE(NULLIF(CASE WHEN %s LIKE '(%%)' THEN '-' || TRIM(BOTH '()' FROM %s) ELSE %s END, '')::numeric, 0)", value, value, value)
}

func amazonSumSQL(headers ...string) string {
	parts := make([]string, len(headers))
	for i, header := range headers {
		parts[i] = amazonAmountSQL(header)
	}
	return strings.Join(parts, " + ")
}

// Source dates remain verbatim for display and deduplication. Parse their date
// component for existing imports as well as newly imported reports.
func amazonDateSQL(source string) string {
	source = "BTRIM(" + source + ")"
	return "CASE WHEN " + source + " ~ '^\\d{4}[-/]\\d{1,2}[-/]\\d{1,2}' THEN replace(substring(" + source + " from '^\\d{4}[-/]\\d{1,2}[-/]\\d{1,2}'), '/', '-')::date" +
		" WHEN " + source + " ~ '^\\d{1,2}/\\d{1,2}/\\d{4}' THEN to_date(substring(" + source + " from '^\\d{1,2}/\\d{1,2}/\\d{4}'), 'MM/DD/YYYY')" +
		" WHEN " + source + " ~ '^[A-Za-z]+ [0-9]{1,2}, [0-9]{4}' THEN to_date(substring(" + source + " from '^[A-Za-z]+ [0-9]{1,2}, [0-9]{4}'), 'Mon DD, YYYY')" +
		" WHEN " + source + " ~ '^[0-9]{1,2} [A-Za-z]+ [0-9]{4}' THEN to_date(substring(" + source + " from '^[0-9]{1,2} [A-Za-z]+ [0-9]{4}'), 'DD Mon YYYY')" +
		" WHEN " + source + " ~ '^\\d{4}年\\d{1,2}月\\d{1,2}日' THEN replace(replace(replace(substring(" + source + " from '^\\d{4}年\\d{1,2}月\\d{1,2}日'), '年', '-'), '月', '-'), '日', '')::date" +
		" WHEN " + source + " ~ '^\\d{10}$' THEN (to_timestamp((" + source + ")::double precision) AT TIME ZONE 'UTC')::date" +
		" WHEN " + source + " ~ '^\\d{13}$' THEN (to_timestamp((" + source + ")::double precision / 1000) AT TIME ZONE 'UTC')::date" +
		" WHEN " + source + " ~ '^\\d{5}([.]\\d+){0,1}$' THEN CASE WHEN (" + source + ")::numeric >= 20000 THEN DATE '1899-12-30' + FLOOR((" + source + ")::numeric)::int ELSE NULL END ELSE NULL END"
}

func amazonQuarterSQL(currency string, stores ...string) string {
	store := ""
	if len(stores) > 0 {
		store = stores[0]
	}
	// Keep the raw site value: an empty site means "all countries".
	site := ""
	if len(stores) > 1 {
		site = strings.TrimSpace(stores[1])
	}
	siteCurrency := amazonSiteCurrency(site)
	periodColumn := "quarter"
	if len(stores) > 3 && stores[3] == "month" {
		periodColumn = "month"
	}
	if len(stores) > 2 {
		if requested := amazonCurrencyCode(stores[2]); requested != "" {
			siteCurrency = requested
		}
	}
	shipmentSource, financeSource := "amazon_tax_shipments s", "amazon_tax_finances f"
	if strings.TrimSpace(store) != "" || strings.TrimSpace(site) != "" {
		storeConditions, financeConditions := []string{}, []string{}
		if strings.TrimSpace(store) != "" {
			escaped := strings.ReplaceAll(strings.TrimSpace(store), "'", "''")
			storeConditions = append(storeConditions, "(store_id = '"+escaped+"' OR store_id = '')")
		}
		financeConditions = append(financeConditions, storeConditions...)
		if strings.TrimSpace(site) != "" {
			siteEscaped := strings.ReplaceAll(strings.TrimSpace(site), "'", "''")
			financeConditions = append(financeConditions, "(site_id = '"+siteEscaped+"' OR site_id = '' OR site_id IS NULL)")
		}
		// Shipment files are store-level data shared by every site: a selected
		// site claims its rows through the 货币 column, not the site_id column.
		if len(storeConditions) > 0 {
			shipmentSource = "(SELECT * FROM amazon_tax_shipments WHERE " + strings.Join(storeConditions, " AND ") + ") s"
		}
		if len(financeConditions) > 0 {
			financeSource = "(SELECT * FROM amazon_tax_finances WHERE " + strings.Join(financeConditions, " AND ") + ") f"
		}
	}
	sales := amazonSumSQL("商品价格", "商品税", "运费", "运费税", "礼品包装价格", "礼品包装税费", "商品促销折扣", "货件促销折扣")
	// Shipment amounts convert from each row's own 货币, quantities need no
	// conversion. A selected site takes sales/quantity only from rows
	// denominated in that site's currency (legacy shared rows have empty
	// site_id but real currencies); other rows stay in the query and
	// contribute zero, so their periods keep showing a zero total.
	shipmentCurrency := "COALESCE(NULLIF(UPPER(BTRIM(s.data->>'货币')), ''), 'USD')"
	salesCurrencyCondition := "TRUE"
	if strings.TrimSpace(site) != "" && strings.TrimSpace(siteCurrency) != "" {
		salesCurrencyCondition = shipmentCurrency + " = '" + strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(siteCurrency)), "'", "''") + "'"
	}
	refund := amazonSumSQL("product sales", "product sales tax", "shipping credits", "shipping credits tax", "gift wrap credits", "giftwrap credits tax", "Regulatory Fee", "Tax On Regulatory Fee", "promotional rebates", "promotional rebates tax", "other")
	shipmentJoin, shipmentFactor := amazonShipmentConversion(currency)
	financeJoin, financeFactor := "", "1::numeric"
	baseCurrency := siteCurrency
	if strings.TrimSpace(site) == "" {
		// Finance rows carry no currency column; without a site filter each row is
		// denominated in its own site currency, so resolve the currency per row.
		financeDate := amazonDateSQL("f.data->>'date/time'")
		rowCurrency := amazonFinanceRowCurrencySQL()
		if currency == "CNY" {
			financeJoin = taxRateJoinSQL("fx", financeDate, rowCurrency)
			financeFactor = "CASE WHEN " + rowCurrency + " = 'CNY' THEN 1::numeric ELSE fx.rate END"
		} else {
			target := strings.ReplaceAll(currency, "'", "''")
			financeJoin = taxRateJoinSQL("fx", financeDate, rowCurrency) + " " + taxRateJoinSQL("ux", financeDate, "'"+target+"'")
			financeFactor = "CASE WHEN " + rowCurrency + " = '" + target + "' THEN 1::numeric WHEN " + rowCurrency + " = 'CNY' THEN 1::numeric / NULLIF(ux.rate, 0) WHEN fx.rate IS NULL OR ux.rate IS NULL THEN NULL ELSE fx.rate / ux.rate END"
		}
	} else if currency == "CNY" && baseCurrency != "CNY" {
		financeJoin = taxRateJoinSQL("fx", amazonDateSQL("f.data->>'date/time'"), "'"+baseCurrency+"'")
		financeFactor = "fx.rate"
	} else if currency != baseCurrency {
		financeJoin = taxRateJoinSQL("fx", amazonDateSQL("f.data->>'date/time'"), "'"+baseCurrency+"'") + " " + taxRateJoinSQL("ux", amazonDateSQL("f.data->>'date/time'"), "'"+strings.ReplaceAll(currency, "'", "''")+"'")
		financeFactor = "CASE WHEN fx.rate IS NULL OR ux.rate IS NULL THEN NULL ELSE fx.rate / ux.rate END"
	}
	shipmentSource += " " + shipmentJoin
	financeSource += " " + financeJoin
	// Never turn missing rates into a partial total or a misleading zero.
	aggregate := func(amount, factor string) string {
		return "CASE WHEN COUNT(*) FILTER (WHERE (" + factor + ") IS NULL AND (" + amount + ") <> 0) > 0 THEN NULL ELSE COALESCE(SUM((" + amount + ") * (" + factor + ")), 0) END"
	}
	orderTypes := "LOWER(BTRIM(COALESCE(data->>'type', ''))) IN ('refund', 'order')"
	serviceType := "LOWER(BTRIM(COALESCE(data->>'type', ''))) = 'service fee'"
	inboundPlacement := serviceType + " AND LOWER(BTRIM(COALESCE(data->>'description', ''))) = 'fba inbound placement service fee'"
	premiumServices := serviceType + " AND LOWER(BTRIM(COALESCE(data->>'description', ''))) = 'premium services fee'"
	sellingTypes := orderTypes
	refundTypes := "LOWER(BTRIM(COALESCE(data->>'type', ''))) = 'refund'"
	fbaTransactionType := "LOWER(BTRIM(COALESCE(data->>'type', ''))) = 'fba transaction fees'"
	fbaInventoryFee := "LOWER(BTRIM(COALESCE(data->>'type', ''))) = 'fba inventory fee'"
	storageDescriptions := "LOWER(BTRIM(COALESCE(data->>'description', ''))) IN ('fba long term storage fee', 'fba inventory storage fee', 'fba removal order: disposal fee')"
	fulfilmentType := "LOWER(BTRIM(COALESCE(data->>'type', ''))) = 'fulfilment by amazon transaction fees'"
	fulfilmentStorage := "(" + fulfilmentType + ") AND LOWER(BTRIM(COALESCE(data->>'description', ''))) = 'fulfilment by amazon (fba) inventory storage fee'"
	fulfilmentDisposal := "(" + fulfilmentType + ") AND LOWER(BTRIM(COALESCE(data->>'description', ''))) = 'fulfilment by amazon removal order: disposal fee'"
	// Storage: FBA Inventory Fee rows plus the three storage descriptions of
	// FBA Transaction Fees, each summing the fba fees and other columns.
	storageFee := "(" + fbaTransactionType + " AND " + storageDescriptions + ") OR (" + fbaInventoryFee + ")"
	// Delivery additionally takes every FBA Transaction Fees description except
	// the three storage ones plus the Fulfilment storage fee, summing the fba
	// fees (and other, for FBA Transaction Fees) columns.
	fbaTransactionNonStorage := "(" + fbaTransactionType + " AND NOT " + storageDescriptions + ")"
	selected := func(condition, amount string) string {
		return "CASE WHEN " + condition + " THEN (" + amount + ") ELSE 0 END"
	}
	fbaFees := amazonAmountSQL("fba fees")
	fbaFeesOther := "(" + fbaFees + " + " + amazonAmountSQL("other") + ")"
	storageAmount := selected(storageFee, fbaFeesOther)
	deliveryAmount := "(" + selected(orderTypes, fbaFees) + " + " + selected(fulfilmentDisposal, fbaFees) + " + " + selected(fbaTransactionNonStorage, fbaFeesOther) + " + " + selected(fulfilmentStorage, fbaFees) + ")"
	salesAmount := selected(salesCurrencyCondition, sales)
	salesQuantity := selected(salesCurrencyCondition, amazonAmountSQL("已发货数量"))
	otherType := "LOWER(BTRIM(COALESCE(data->>'type', ''))) NOT IN ('refund', 'refund_retrocharge', 'order', 'order_retrocharge', 'service fee', 'transfer', 'fba transaction fees', 'fba inventory fee') AND NOT ((" + fulfilmentStorage + ") OR (" + fulfilmentDisposal + "))"
	serviceSubscription := "LOWER(BTRIM(COALESCE(data->>'type', ''))) = 'service fee' AND LOWER(BTRIM(COALESCE(data->>'description', ''))) = 'subscription'"
	otherTotal := "(" + amazonAmountSQL("total") + " - " + amazonAmountSQL("marketplace withheld tax") + ")"
	otherSubscription := "(" + amazonAmountSQL("other") + " - " + amazonAmountSQL("marketplace withheld tax") + ")"
	otherAmount := "(" + selected(otherType, otherTotal) + " + " + selected(serviceSubscription, otherSubscription) + " + " + selected(inboundPlacement, amazonAmountSQL("other")) + " + " + selected(premiumServices, amazonAmountSQL("selling fees")) + ")"
	metrics := []struct{ key, condition, amount string }{
		{"refund", refundTypes, refund},
		{"tax", "TRUE", amazonAmountSQL("marketplace withheld tax")},
		{"delivery", "TRUE", deliveryAmount},
		{"selling", sellingTypes, amazonAmountSQL("selling fees")},
		{"ads", "LOWER(BTRIM(COALESCE(data->>'type', ''))) = 'service fee'", amazonAmountSQL("other transaction fees")},
		{"storage", "TRUE", storageAmount},
		{"other", "TRUE", otherAmount},
	}
	selections, output := []string{}, []string{}
	for _, metric := range metrics {
		selections = append(selections, aggregate(selected(metric.condition, metric.amount), financeFactor)+" AS "+metric.key)
		output = append(output, "CASE WHEN f.quarter IS NULL THEN 0 ELSE f."+metric.key+" END AS "+metric.key)
	}
	financeDetails := []struct{ key, condition, amount string }{
		{"refund_product_sales", refundTypes, amazonAmountSQL("product sales")},
		{"refund_product_sales_tax", refundTypes, amazonAmountSQL("product sales tax")},
		{"refund_shipping_credits", refundTypes, amazonAmountSQL("shipping credits")},
		{"refund_shipping_tax", refundTypes, amazonAmountSQL("shipping credits tax")},
		{"refund_gift_wrap", refundTypes, amazonAmountSQL("gift wrap credits")},
		{"refund_gift_wrap_tax", refundTypes, amazonAmountSQL("giftwrap credits tax")},
		{"refund_regulatory_fee", refundTypes, amazonAmountSQL("Regulatory Fee")},
		{"refund_regulatory_tax", refundTypes, amazonAmountSQL("Tax On Regulatory Fee")},
		{"refund_promotions", refundTypes, amazonAmountSQL("promotional rebates")},
		{"refund_promotions_tax", refundTypes, amazonAmountSQL("promotional rebates tax")},
		{"refund_other", refundTypes, amazonAmountSQL("other")},
		{"tax_marketplace_withheld", "TRUE", amazonAmountSQL("marketplace withheld tax")},
		{"delivery_fba_fees", orderTypes, fbaFees},
		{"delivery_fulfilment_disposal", fulfilmentDisposal, fbaFees},
		{"delivery_transaction_other", fbaTransactionNonStorage, fbaFeesOther},
		{"delivery_fulfilment_storage", fulfilmentStorage, fbaFees},
		{"other_inbound_placement", inboundPlacement, amazonAmountSQL("other")},
		{"other_premium_services", premiumServices, amazonAmountSQL("selling fees")},
		{"selling_fees", sellingTypes, amazonAmountSQL("selling fees")},
		{"ads_other_transaction_fees", "LOWER(BTRIM(COALESCE(data->>'type', ''))) = 'service fee'", amazonAmountSQL("other transaction fees")},
		{"other_total", otherType, amazonAmountSQL("total")},
		{"other_marketplace_withheld", serviceSubscription, amazonAmountSQL("marketplace withheld tax")},
		{"other_subscription", serviceSubscription, amazonAmountSQL("other")},
		{"storage_transaction", "(" + fbaTransactionType + " AND " + storageDescriptions + ")", fbaFeesOther},
		{"storage_inventory", fbaInventoryFee, fbaFeesOther},
	}
	for _, detail := range financeDetails {
		selections = append(selections, aggregate(selected(detail.condition, detail.amount), financeFactor)+" AS "+detail.key)
		output = append(output, "CASE WHEN f.quarter IS NULL THEN 0 ELSE f."+detail.key+" END AS "+detail.key)
	}
	shipmentDetails := []struct{ key, header string }{
		{"sales_product", "商品价格"},
		{"sales_product_tax", "商品税"},
		{"sales_shipping", "运费"},
		{"sales_shipping_tax", "运费税"},
		{"sales_gift_wrap", "礼品包装价格"},
		{"sales_gift_wrap_tax", "礼品包装税费"},
		{"sales_promotion", "商品促销折扣"},
		{"sales_shipment_promotion", "货件促销折扣"},
	}
	shipmentSelect, shipmentOutput, finalDetails := []string{}, []string{}, []string{}
	for _, detail := range shipmentDetails {
		shipmentSelect = append(shipmentSelect, aggregate(selected(salesCurrencyCondition, amazonAmountSQL(detail.header)), shipmentFactor)+" AS "+detail.key)
		shipmentOutput = append(shipmentOutput, "CASE WHEN s.quarter IS NULL THEN 0 ELSE s."+detail.key+" END AS "+detail.key)
		finalDetails = append(finalDetails, "ROUND("+detail.key+",2) AS "+detail.key)
	}
	for _, detail := range financeDetails {
		finalDetails = append(finalDetails, "ROUND("+detail.key+",2) AS "+detail.key)
	}
	for _, metric := range metrics {
		finalDetails = append(finalDetails, "ROUND("+metric.key+",2) AS "+metric.key)
	}
	monthSelect := "NULL"
	if periodColumn == "month" {
		monthSelect = "period"
	}
	return fmt.Sprintf(`WITH shipments AS (
	 SELECT s.%[10]s AS period, s.quarter, SUM(%[1]s) AS sales_volume, %[2]s AS sales, %[3]s FROM %[4]s GROUP BY s.%[10]s, s.quarter
 ), finance AS (
	 SELECT f.%[10]s AS period, f.quarter, %[5]s FROM %[6]s GROUP BY f.%[10]s, f.quarter
 ), totals AS (
	 SELECT COALESCE(s.period, f.period) AS period, COALESCE(s.quarter, f.quarter) AS quarter, COALESCE(s.sales_volume, 0) AS sales_volume,
	 CASE WHEN s.period IS NULL THEN 0 ELSE s.sales END AS sales, %[7]s, %[8]s
	 FROM shipments s FULL OUTER JOIN finance f ON f.period = s.period
		) SELECT %[11]s AS month, quarter, sales_volume, ROUND(sales,2) AS sales, %[9]s, ROUND(sales + refund, 2) AS taxable_income FROM totals ORDER BY period`,
		salesQuantity, aggregate(salesAmount, shipmentFactor), strings.Join(shipmentSelect, ", "), shipmentSource, strings.Join(selections, ", "), financeSource, strings.Join(shipmentOutput, ", "), strings.Join(output, ", "), strings.Join(finalDetails, ", "), periodColumn, monthSelect)
}

func ListAmazonTaxQuarters(c *gin.Context) {
	rows := []amazonTaxQuarter{}
	currency := strings.ToUpper(c.DefaultQuery("currency", "USD"))
	allCountries := strings.TrimSpace(c.Query("site_id")) == ""
	siteCurrency := amazonSiteCurrency(c.Query("site_id"))
	if requested := amazonCurrencyCode(c.Query("site_currency")); requested != "" {
		siteCurrency = requested
	}
	if currency != "CNY" && currency != siteCurrency {
		c.JSON(400, gin.H{"error": "仅支持当前站点币种或人民币"})
		return
	}
	if allCountries && currency != "CNY" {
		c.JSON(400, gin.H{"error": "全部国家下仅支持人民币汇总"})
		return
	}
	dateFrom, dateTo := c.Query("date_from"), c.Query("date_to")
	if (dateFrom != "" && !validTaxDate(dateFrom)) || (dateTo != "" && !validTaxDate(dateTo)) || (dateFrom != "" && dateTo != "" && dateFrom > dateTo) {
		c.JSON(400, gin.H{"error": "日期范围无效"})
		return
	}
	queryRows := func(period string) ([]amazonTaxQuarter, error) {
		result := []amazonTaxQuarter{}
		return result, database.DB.Raw(amazonQuarterSQL(currency, c.Query("store_id"), c.Query("site_id"), siteCurrency, period)).Scan(&result).Error
	}
	if c.Query("view") == "month" {
		var err error
		rows, err = queryRows("month")
		if err != nil {
			taxError(c, err)
			return
		}
	} else if err := database.DB.Raw(amazonQuarterSQL(currency, c.Query("store_id"), c.Query("site_id"), siteCurrency)).Scan(&rows).Error; err != nil {
		taxError(c, err)
		return
	}
	monthlyRows, err := queryRows("month")
	if err != nil {
		taxError(c, err)
		return
	}
	monthsByQuarter := map[string][]amazonTaxQuarter{}
	for _, row := range monthlyRows {
		monthEnd := row.Month + "-31"
		if dateFrom != "" && monthEnd < dateFrom || dateTo != "" && row.Month+"-01" > dateTo {
			continue
		}
		monthsByQuarter[row.Quarter] = append(monthsByQuarter[row.Quarter], row)
	}
	if c.Query("view") != "month" {
		filtered := rows[:0]
		for _, row := range rows {
			start, end := quarterDateRange(row.Quarter)
			if (dateFrom == "" || end >= dateFrom) && (dateTo == "" || start <= dateTo) {
				row.Months = monthsByQuarter[row.Quarter]
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	} else {
		filtered := rows[:0]
		for _, row := range rows {
			monthEnd := row.Month + "-31"
			if (dateFrom == "" || monthEnd >= dateFrom) && (dateTo == "" || row.Month+"-01" <= dateTo) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	var summaries []model.TaxImportLog
	summaryQuery := database.DB.Model(&model.TaxImportLog{}).Where("platform = ? AND report_type = ? AND status = ?", "amazon", "finance_summary", "locked")
	if store := strings.TrimSpace(c.Query("store_id")); store != "" {
		summaryQuery = summaryQuery.Where("store_id = ?", store)
	}
	summaryQuery = filterTaxSite(summaryQuery, c.Query("site_id"))
	if err := summaryQuery.Find(&summaries).Error; err != nil {
		taxError(c, err)
		return
	}
	checkedSummaries := map[string]struct{}{}
	for _, record := range summaries {
		var summary amazonSummary
		if err := json.Unmarshal([]byte(record.Message), &summary); err != nil {
			taxError(c, fmt.Errorf("汇总 PDF 记录 %d 无法解析", record.ID))
			return
		}
		identity := strings.ToLower(strings.TrimSpace(summary.DisplayName)) + "\x00" + summary.Start + "\x00" + summary.End + "\x00" + record.StoreID + "\x00" + record.SiteID
		if _, seen := checkedSummaries[identity]; seen {
			continue
		}
		checkedSummaries[identity] = struct{}{}
		// Each PDF covers exactly one store and site; validate the details within
		// that scope instead of the requested aggregate scope, so "all stores" or
		// "all countries" views cannot produce cross-scope false mismatches.
		validationSite := taxSiteID(record.SiteID)
		validationCurrency := strings.ToUpper(strings.TrimSpace(summary.Currency))
		if validationCurrency == "" {
			validationCurrency = amazonSiteCurrency(validationSite)
		}
		var validationRows []amazonTaxQuarter
		if err := database.DB.Raw(amazonQuarterSQL(validationCurrency, record.StoreID, validationSite)).Scan(&validationRows).Error; err != nil {
			taxError(c, err)
			return
		}
		tax, storage, delivery := decimal.Zero, decimal.Zero, decimal.Zero
		for _, q := range validationRows {
			if len(q.Quarter) < 7 {
				continue
			}
			year := q.Quarter[:4]
			quarter := q.Quarter[6:]
			startMonth := map[string]string{"1": "01", "2": "04", "3": "07", "4": "10"}[quarter]
			endMonth := map[string]string{"1": "03-31", "2": "06-30", "3": "09-30", "4": "12-31"}[quarter]
			if startMonth == "" || year+"-"+startMonth+"-01" > summary.End || year+"-"+endMonth < summary.Start {
				continue
			}
			if q.Tax != nil {
				tax = tax.Add(*q.Tax)
			}
			if q.Storage != nil {
				storage = storage.Add(*q.Storage)
			}
			if q.Delivery != nil {
				delivery = delivery.Add(*q.Delivery)
			}
		}
		for _, metric := range []struct {
			name            string
			value, expected decimal.Decimal
		}{{"税费", tax, summary.Tax}, {"仓储费", storage, summary.Storage}, {"配送费", delivery, summary.Delivery}} {
			if metric.value.Equal(metric.expected) {
				continue
			}
			message := fmt.Sprintf("%s 至 %s %s校验不一致：明细 %s，PDF %s，差额 %s %s", summary.Start, summary.End, metric.name, metric.value.StringFixed(2), metric.expected.StringFixed(2), metric.value.Sub(metric.expected).StringFixed(2), summary.Currency)
			for i := range rows {
				year := rows[i].Quarter[:4]
				quarter := rows[i].Quarter[6:]
				month := map[string]string{"1": "01", "2": "04", "3": "07", "4": "10"}[quarter]
				start := year + "-" + month + "-01"
				endMonth := map[string]string{"1": "03-31", "2": "06-30", "3": "09-30", "4": "12-31"}[quarter]
				if start <= summary.End && year+"-"+endMonth >= summary.Start {
					rows[i].ValidationErrors = append(rows[i].ValidationErrors, message)
				}
			}
		}
	}
	categoriesByQuarter, categoriesByMonth, err := amazonSalesCategories(database.DB, c.Query("store_id"), c.Query("site_id"), siteCurrency)
	if err != nil {
		taxError(c, err)
		return
	}
	for i := range rows {
		if rows[i].Sales == nil {
			if err := database.DB.Raw(amazonSalesConversionIssuesSQL(currency), rows[i].Quarter).Scan(&rows[i].SalesConversionIssues).Error; err != nil {
				taxError(c, err)
				return
			}
		}
		rows[i].SalesCategories = categoriesByQuarter[rows[i].Quarter]
		if rows[i].SalesCategories == nil {
			rows[i].SalesCategories = []amazonSalesCategory{}
		}
		for j := range rows[i].Months {
			rows[i].Months[j].SalesCategories = categoriesByMonth[rows[i].Months[j].Month]
			if rows[i].Months[j].SalesCategories == nil {
				rows[i].Months[j].SalesCategories = []amazonSalesCategory{}
			}
		}
	}
	c.JSON(200, gin.H{"items": rows})
}
