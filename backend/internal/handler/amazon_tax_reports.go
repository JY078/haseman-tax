package handler

import (
	"bytes"
	"encoding/csv"
	"net/http"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type amazonCountryQuarterSummary struct {
	Country string
	Row     amazonTaxQuarter
}

func aggregateAmazonQuarterRows(rows []amazonTaxQuarter) amazonTaxQuarter {
	if len(rows) == 0 {
		return amazonTaxQuarter{}
	}
	sum := func(value func(amazonTaxQuarter) *decimal.Decimal) *decimal.Decimal {
		total := decimal.Zero
		for _, row := range rows {
			amount := value(row)
			if amount == nil {
				return nil
			}
			total = total.Add(*amount)
		}
		return &total
	}
	result := amazonTaxQuarter{SalesVolume: decimal.Zero}
	for _, row := range rows {
		result.SalesVolume = result.SalesVolume.Add(row.SalesVolume)
	}
	result.TaxableIncome = sum(func(row amazonTaxQuarter) *decimal.Decimal { return row.TaxableIncome })
	result.Sales = sum(func(row amazonTaxQuarter) *decimal.Decimal { return row.Sales })
	result.Refund = sum(func(row amazonTaxQuarter) *decimal.Decimal { return row.Refund })
	result.Tax = sum(func(row amazonTaxQuarter) *decimal.Decimal { return row.Tax })
	result.Delivery = sum(func(row amazonTaxQuarter) *decimal.Decimal { return row.Delivery })
	result.Selling = sum(func(row amazonTaxQuarter) *decimal.Decimal { return row.Selling })
	result.Ads = sum(func(row amazonTaxQuarter) *decimal.Decimal { return row.Ads })
	result.Other = sum(func(row amazonTaxQuarter) *decimal.Decimal { return row.Other })
	result.Storage = sum(func(row amazonTaxQuarter) *decimal.Decimal { return row.Storage })
	return result
}

func amazonCountryQuarterSummaries(store, selectedSite, currency, dateFrom, dateTo string) ([]amazonCountryQuarterSummary, error) {
	sites := []model.TaxCatalogEntry{}
	siteQuery := database.DB.Where("platform = ? AND kind = ?", "amazon", "site").Order("id")
	if selectedSite != "" {
		siteQuery = siteQuery.Where("value = ?", taxSiteID(selectedSite))
	}
	if err := siteQuery.Find(&sites).Error; err != nil {
		return nil, err
	}
	if len(sites) == 0 && selectedSite != "" {
		sites = append(sites, model.TaxCatalogEntry{Value: taxSiteID(selectedSite), Label: taxSiteID(selectedSite), Currency: amazonSiteCurrency(selectedSite)})
	} else if len(sites) == 0 {
		sites = append(sites, model.TaxCatalogEntry{Value: "US", Label: "美国", Currency: "USD"}, model.TaxCatalogEntry{Value: "CA", Label: "加拿大", Currency: "CAD"})
	}
	summaries := []amazonCountryQuarterSummary{}
	for _, site := range sites {
		siteCurrency := site.Currency
		if siteCurrency == "" {
			siteCurrency = amazonSiteCurrency(site.Value)
		}
		rows := []amazonTaxQuarter{}
		if err := database.DB.Raw(amazonQuarterSQL(currency, store, site.Value, siteCurrency)).Scan(&rows).Error; err != nil {
			return nil, err
		}
		filtered := rows[:0]
		for _, row := range rows {
			start, end := quarterDateRange(row.Quarter)
			if (dateFrom == "" || end >= dateFrom) && (dateTo == "" || start <= dateTo) {
				filtered = append(filtered, row)
			}
		}
		if len(filtered) == 0 {
			continue
		}
		label := site.Label
		if label == "" {
			label = site.Value
		}
		summaries = append(summaries, amazonCountryQuarterSummary{Country: label, Row: aggregateAmazonQuarterRows(filtered)})
	}
	return summaries, nil
}

func ExportAmazonTaxReport(c *gin.Context) {
	kind := c.Query("type")
	if kind != "shipments" && kind != "finance" && kind != "quarters" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type 必须为 shipments、finance 或 quarters"})
		return
	}
	var output bytes.Buffer
	output.Write([]byte{0xef, 0xbb, 0xbf})
	w := csv.NewWriter(&output)
	write := func(values []string) {
		for i := range values {
			values[i] = taxCSVText(values[i])
		}
		_ = w.Write(values)
	}
	month, quarter := c.Query("month"), c.Query("quarter")
	count, label := 0, ""
	switch kind {
	case "shipments", "finance":
		label = map[string]string{"shipments": "配送货件", "finance": "财务报告"}[kind]
		table := map[string]string{"shipments": "amazon_tax_shipments", "finance": "amazon_tax_finances"}[kind]
		q := database.DB.Table(table)
		if month != "" {
			q = q.Where("month IN ?", taxPeriodValues(month))
		}
		if quarter != "" {
			q = q.Where("quarter IN ?", taxPeriodValues(quarter))
		}
		if c.Query("store_id") != "" {
			q = q.Where("(store_id = ? OR store_id = '')", c.Query("store_id"))
		}
		if kind != "shipments" {
			q = filterTaxSite(q, c.Query("site_id"))
		}
		rows := []model.AmazonTaxShipment{}
		if err := q.Order("id").Find(&rows).Error; err != nil {
			taxError(c, err)
			return
		}
		categoryRules := []model.AmazonProductCategory{}
		if kind == "shipments" {
			if err := database.DB.Order("id").Find(&categoryRules).Error; err != nil {
				taxError(c, err)
				return
			}
		}
		classify := func(name string) string {
			name = strings.ToLower(name)
			category, length := "未分类", 0
			for _, rule := range categoryRules {
				if rule.Keyword == "" {
					continue
				}
				keyword := strings.ToLower(rule.Keyword)
				if strings.Contains(name, keyword) && len(keyword) > length {
					category, length = rule.Category, len(keyword)
				}
			}
			return category
		}
		headers := amazonTaxHeaders[kind]
		outHeaders := append([]string{"月份", "季度"}, headers...)
		for i := range outHeaders {
			if i >= 2 {
				outHeaders[i] += "(" + outHeaders[i] + ")"
			}
		}
		if kind == "shipments" {
			for i, header := range outHeaders {
				if header == "商品名称(商品名称)" {
					outHeaders = append(outHeaders[:i+1], append([]string{"商品名称（中文）"}, outHeaders[i+1:]...)...)
					break
				}
			}
		}
		write(outHeaders)
		for _, row := range rows {
			values := []string{row.Month, row.Quarter}
			for _, header := range headers {
				values = append(values, row.Data[header])
				if kind == "shipments" && header == "商品名称" {
					values = append(values, classify(row.Data["商品名称"]))
				}
			}
			write(values)
		}
		count = len(rows)
	case "quarters":
		label = "季度汇总"
		currency := strings.ToUpper(c.DefaultQuery("currency", "USD"))
		countryView := c.Query("view") == "country" && strings.TrimSpace(c.Query("store_id")) != ""
		if strings.TrimSpace(c.Query("site_id")) == "" && currency != "CNY" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "全部国家下仅支持人民币汇总"})
			return
		}
		siteCurrency := amazonSiteCurrency(c.Query("site_id"))
		if requested := amazonCurrencyCode(c.Query("site_currency")); requested != "" {
			siteCurrency = requested
		}
		if currency != "CNY" && currency != siteCurrency {
			c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持当前站点币种或人民币"})
			return
		}
		dateFrom, dateTo := c.Query("date_from"), c.Query("date_to")
		if (dateFrom != "" && !validTaxDate(dateFrom)) || (dateTo != "" && !validTaxDate(dateTo)) || (dateFrom != "" && dateTo != "" && dateFrom > dateTo) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "日期范围无效"})
			return
		}
		if countryView {
			rows, err := amazonCountryQuarterSummaries(c.Query("store_id"), c.Query("site_id"), currency, dateFrom, dateTo)
			if err != nil {
				taxError(c, err)
				return
			}
			label = "国家汇总"
			write([]string{"国家", "销量（件）", "销售额（" + currency + "）", "退货款（" + currency + "）", "税费（" + currency + "）", "配送费（" + currency + "）", "销售费（" + currency + "）", "广告费（" + currency + "）", "仓储费（" + currency + "）", "其他费用（" + currency + "）", "报税收入（" + currency + "）"})
			for _, item := range rows {
				row := item.Row
				income := "待换算"
				if row.TaxableIncome != nil {
					income = row.TaxableIncome.StringFixed(2)
				}
				write([]string{item.Country, row.SalesVolume.StringFixed(0), valuePtr(row.Sales), valuePtr(row.Refund), valuePtr(row.Tax), valuePtr(row.Delivery), valuePtr(row.Selling), valuePtr(row.Ads), valuePtr(row.Storage), valuePtr(row.Other), income})
				count++
			}
			break
		}
		rows := []amazonTaxQuarter{}
		if err := database.DB.Raw(amazonQuarterSQL(currency, c.Query("store_id"), c.Query("site_id"), siteCurrency)).Scan(&rows).Error; err != nil {
			taxError(c, err)
			return
		}
		write([]string{"季度", "销量（件）", "销售额（" + currency + "）", "退货款（" + currency + "）", "税费（" + currency + "）", "配送费（" + currency + "）", "销售费（" + currency + "）", "广告费（" + currency + "）", "仓储费（" + currency + "）", "其他费用（" + currency + "）", "报税收入（" + currency + "）"})
		for _, row := range rows {
			if quarter != "" && !strings.Contains(","+quarter+",", ","+row.Quarter+",") {
				continue
			}
			start, end := quarterDateRange(row.Quarter)
			if (dateFrom != "" && end < dateFrom) || (dateTo != "" && start > dateTo) {
				continue
			}
			value := func(v interface{ StringFixed(int32) string }) string { return v.StringFixed(2) }
			income := "待换算"
			if row.TaxableIncome != nil {
				income = value(row.TaxableIncome)
			}
			write([]string{row.Quarter, row.SalesVolume.StringFixed(0), valuePtr(row.Sales), valuePtr(row.Refund), valuePtr(row.Tax), valuePtr(row.Delivery), valuePtr(row.Selling), valuePtr(row.Ads), valuePtr(row.Storage), valuePtr(row.Other), income})
			count++
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		taxError(c, err)
		return
	}
	currency := ""
	if kind == "quarters" {
		currency = strings.ToUpper(c.DefaultQuery("currency", "USD"))
	}
	deliverTaxFile(c, model.TaxDownloadLog{Platform: "amazon", SiteID: taxSiteFromContext(c), ReportType: kind, Month: month, Quarter: quarter, Currency: currency, FileName: taxReportFileName(c, "Amazon", label, currency), FileContent: output.Bytes(), ContentType: "text/csv; charset=utf-8", RowCount: count})
}

func valuePtr(value interface{ StringFixed(int32) string }) string {
	if value == nil {
		return "待换算"
	}
	return value.StringFixed(2)
}
