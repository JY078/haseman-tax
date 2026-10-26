package handler

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func filterTikTokTax(q *gorm.DB, month, quarter string) *gorm.DB {
	if month != "" {
		q = q.Where("month IN ?", taxPeriodValues(month))
	}
	if quarter != "" {
		q = q.Where("quarter IN ?", taxPeriodValues(quarter))
	}
	return q
}

func filterTikTokTaxStore(q *gorm.DB, month, quarter, store, site string) *gorm.DB {
	q = filterTikTokTax(q, month, quarter)
	if strings.TrimSpace(store) != "" {
		q = q.Where("(store_id = ? OR store_id = '')", strings.TrimSpace(store))
	}
	return filterTaxSite(q, site)
}

func ListTikTokTaxFilterOptions(c *gin.Context) {
	kind, field := c.Query("type"), c.Query("field")
	if kind == "settlements" && (field == "month" || field == "quarter" || field == "logistics_method" || field == "order_type") {
		filters := map[string]string{"month": c.Query("month"), "quarter": c.Query("quarter"), "logistics_method": c.Query("logistics_method"), "order_type": c.Query("order_type")}
		filters[field] = ""
		q := filteredTikTokSettlements(database.DB, filters["month"], filters["quarter"], filters["logistics_method"], filters["order_type"])
		if c.Query("store_id") != "" {
			q = q.Where("(store_id = ? OR store_id = '')", c.Query("store_id"))
		}
		q = filterTaxSite(q, c.Query("site_id"))
		values := []string{}
		if err := q.Distinct(field).Order(field+" DESC").Pluck(field, &values).Error; err != nil {
			taxError(c, err)
			return
		}
		for i := range values {
			if values[i] == "" {
				values[i] = "__empty__"
			}
		}
		c.JSON(200, gin.H{"values": values})
		return
	}

	if (kind != "details" && kind != "ads" && kind != "fbt" && kind != "settlements" && kind != "orders") || (field != "month" && field != "quarter") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的报表或筛选字段"})
		return
	}
	q := database.DB.Model(&model.TikTokTaxDetail{})
	if kind == "ads" {
		q = database.DB.Model(&model.TikTokTaxAd{})
	} else if kind == "fbt" {
		q = database.DB.Model(&model.TikTokTaxFBT{})
	} else if kind == "settlements" {
		q = database.DB.Model(&model.TikTokSettlement{})
	} else if kind == "orders" {
		q = database.DB.Model(&model.TikTokOrder{})
	}
	q = filterTikTokTaxStore(q, c.Query("month"), c.Query("quarter"), c.Query("store_id"), c.Query("site_id"))
	month, quarter := c.Query("month"), c.Query("quarter")
	if field == "month" {
		month = ""
	} else {
		quarter = ""
	}
	values := []string{}
	if err := filterTikTokTaxStore(q, month, quarter, c.Query("store_id"), c.Query("site_id")).Where(field+" <> ''").Distinct().Order(field+" DESC").Pluck(field, &values).Error; err != nil {
		taxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"values": values})
}

// Daily database rates also apply to records imported before those rates existed.
func queryTikTokDetails(db *gorm.DB, month, quarter string) *gorm.DB {
	currency := "COALESCE(NULLIF(UPPER(BTRIM(r.data->>'币种')), ''), 'USD')"
	factor := "CASE WHEN " + currency + " = 'CNY' THEN 1 ELSE COALESCE(fx.rate, d.rate) END"
	return filterTikTokTax(db.Table("tik_tok_tax_details d"), month, quarter).
		Joins("LEFT JOIN tik_tok_tax_raw_rows r ON r.id = d.raw_row_id").
		Joins(taxRateJoinSQL("fx", "d.posted_at", currency)).
		Select("d.*, " + factor + " AS rate, ROUND((d.income - d.refund - d.commission) * (" + factor + "), 2) AS fee")
}

func queryTikTokAds(db *gorm.DB, month, quarter string) *gorm.DB {
	return filterTikTokTax(db.Table("tik_tok_tax_ads a"), month, quarter).
		Joins(taxRateJoinSQL("fx", "a.date", "'USD'")).
		Select("a.*, COALESCE(NULLIF(fx.rate, 0), a.rate) AS rate, ROUND(a.amount * COALESCE(NULLIF(fx.rate, 0), a.rate), 2) AS fee").
		Order("a.date, a.id")
}

func ExportTikTokTaxReport(c *gin.Context) {
	kind, month, quarter := c.Query("type"), c.Query("month"), c.Query("quarter")
	currency := strings.ToUpper(c.DefaultQuery("currency", "CNY"))
	if currency != "USD" && currency != "CNY" {
		c.JSON(400, gin.H{"error": "仅支持人民币或美元"})
		return
	}
	var output bytes.Buffer
	output.Write([]byte{0xef, 0xbb, 0xbf})
	writer := csv.NewWriter(&output)
	write := func(values []string) {
		for i := range values {
			values[i] = taxCSVText(values[i])
		}
		_ = writer.Write(values)
	}
	count, label := 0, ""
	amount := func(value *decimal.Decimal) string {
		if value == nil {
			return "待换算"
		}
		return value.StringFixed(2)
	}
	switch kind {
	case "orders":
		label = "订单"
		rows := []model.TikTokOrder{}
		q := filterTikTokTax(database.DB.Model(&model.TikTokOrder{}), month, quarter)
		if store := c.Query("store_id"); store != "" {
			q = q.Where("(store_id = ? OR store_id = '')", store)
		}
		q = filterTaxSite(q, c.Query("site_id"))
		if err := q.Order("id").Find(&rows).Error; err != nil {
			taxError(c, err)
			return
		}
		headers := []string{"月份", "季度"}
		for _, column := range tikTokOrderColumns {
			name := column.Name
			if name == "" {
				name = column.Key
			}
			headers = append(headers, name+"("+column.Label+")")
		}
		write(headers)
		for _, row := range rows {
			values := []string{row.Month, row.Quarter}
			for _, column := range tikTokOrderColumns {
				values = append(values, row.Data[column.Key])
			}
			write(values)
		}
		count = len(rows)
	case "settlements":
		label = "结算单"
		rows := []tikTokSettlementResponse{}
		q := filteredTikTokSettlements(database.DB, month, quarter, c.Query("logistics_method"), c.Query("order_type"))
		if store := c.Query("store_id"); store != "" {
			q = q.Where("(store_id = ? OR store_id = '')", store)
		}
		q = filterTaxSite(q, c.Query("site_id"))
		if err := q.Order("statement_date, id").Find(&rows).Error; err != nil {
			taxError(c, err)
			return
		}
		headers := []string{"月份", "季度", "物流方式", "订单类型", "费用（人民币）"}
		for _, column := range tikTokSettlementDisplayColumns {
			name := column.Name
			if name == "" {
				name = column.Key
			}
			headers = append(headers, name+"("+column.Label+")")
		}
		write(headers)
		for _, row := range rows {
			values := []string{row.Month, row.Quarter, row.LogisticsMethod, row.OrderType, amount(row.Fee)}
			for _, column := range tikTokSettlementDisplayColumns {
				values = append(values, row.Data[column.Key])
			}
			write(values)
		}
		count = len(rows)
	case "details":
		label = "涉税申报明细"
		var rows []model.TikTokTaxDetail
		detailsQuery := queryTikTokDetails(database.DB, month, quarter)
		if store := c.Query("store_id"); store != "" {
			detailsQuery = detailsQuery.Where("(d.store_id = ? OR d.store_id = '')", store)
		}
		detailsQuery = filterTaxSite(detailsQuery, c.Query("site_id"))
		if err := detailsQuery.Order("d.posted_at, d.id").Find(&rows).Error; err != nil {
			taxError(c, err)
			return
		}
		write([]string{"月份", "季度", "费用（人民币）", "交易入账时间", "业务单号", "收入金额（美元）", "退款金额（美元）", "平台佣金/服务费金额（美元）"})
		for _, row := range rows {
			write([]string{row.Month, row.Quarter, row.Fee.StringFixed(2), row.PostedAt.Format("2006-01-02"), row.BusinessID, row.Income.StringFixed(2), row.Refund.StringFixed(2), row.Commission.StringFixed(2)})
		}
		count = len(rows)
	case "ads":
		label = "广告费"
		var rows []model.TikTokTaxAd
		adsQuery := queryTikTokAds(database.DB, month, quarter)
		if store := c.Query("store_id"); store != "" {
			adsQuery = adsQuery.Where("(a.store_id = ? OR a.store_id = '')", store)
		}
		adsQuery = filterTaxSite(adsQuery, c.Query("site_id"))
		if err := adsQuery.Scan(&rows).Error; err != nil {
			taxError(c, err)
			return
		}
		write([]string{"月份", "季度", "费用（人民币）", "日期", "系列名称", "推广系列ID", "现金消耗（美元）", "信用额度消耗（美元）", "广告费用赠款消耗（美元）", "金额（美元）", "类型"})
		for _, row := range rows {
			fee := "待换算"
			if row.Rate.IsPositive() {
				fee = row.Fee.StringFixed(2)
			}
			write([]string{row.Month, row.Quarter, fee, row.Date, row.CampaignName, row.CampaignID, row.CashSpend.StringFixed(2), row.CreditSpend.StringFixed(2), row.GrantSpend.StringFixed(2), row.Amount.StringFixed(2), row.Type})
		}
		count = len(rows)
	case "quarters":
		label = "季度汇总_" + currency
		rows, err := listTikTokQuarterSummary(database.DB, currency, c.Query("store_id"), c.Query("site_id"))
		if err != nil {
			taxError(c, err)
			return
		}
		write([]string{"季度", "交易笔数", "销售额（" + currency + "）", "退款费（" + currency + "）", "平台佣金/服务费（" + currency + "）", "联盟佣金（" + currency + "）", "FBT费（" + currency + "）", "广告费（" + currency + "）", "其他费用（" + currency + "）", "活动服务费（" + currency + "）", "智能营销费（" + currency + "）", "税（" + currency + "）", "报税收入（" + currency + "，销售额-退款费）", "配送费待匹配条数"})
		for _, row := range rows {
			if quarter != "" && !slices.Contains(taxPeriodValues(quarter), row.Quarter) {
				continue
			}
			transactionCount := "待导入"
			if row.TransactionCount != nil {
				transactionCount = fmt.Sprint(*row.TransactionCount)
			}
			write([]string{row.Quarter, transactionCount, amount(row.Income), amount(row.Refund), amount(row.Commission), "0.00", amount(row.FBTFee), amount(row.Advertising), "0.00", "0.00", amount(row.SmartPromotionFee), "0.00", amount(row.TaxableIncome), fmt.Sprint(row.DeliveryUnmatchedRows)})
			count++
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type 必须为 details、ads、settlements、orders 或 quarters"})
		return
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		taxError(c, err)
		return
	}
	if kind != "quarters" {
		currency = ""
	}
	deliverTaxFile(c, model.TaxDownloadLog{Platform: "tiktok", SiteID: taxSiteFromContext(c), ReportType: kind, Month: month, Quarter: quarter, Currency: currency, FileName: taxReportFileName(c, "TikTok", label, currency), FileContent: output.Bytes(), ContentType: "text/csv; charset=utf-8", RowCount: count})
}
