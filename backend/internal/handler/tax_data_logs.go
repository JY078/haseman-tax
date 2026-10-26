package handler

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"sort"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func ListTaxImports(c *gin.Context) {
	page, size := taxPage(c)
	rows := []model.TaxImportLog{}
	q := database.DB.Model(&model.TaxImportLog{}).Where("platform = ?", c.DefaultQuery("platform", "walmart"))
	if store := strings.TrimSpace(c.Query("store_id")); store != "" {
		if c.DefaultQuery("platform", "walmart") == "walmart" {
			q = filterWalmartStore(q, store)
		} else {
			q = q.Where("(store_id = ? OR store_id = '')", store)
		}
	}
	q = filterTaxSite(q, c.Query("site_id"))
	var total int64
	if err := q.Count(&total).Error; err != nil {
		taxError(c, err)
		return
	}
	if err := q.Omit("file_content").Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		taxError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": rows, "total": total})
}

func ListTaxDownloads(c *gin.Context) {
	page, size := taxPage(c)
	rows := []model.TaxDownloadLog{}
	q := database.DB.Model(&model.TaxDownloadLog{}).Where("platform = ?", c.DefaultQuery("platform", "walmart"))
	q = filterTaxSite(q, c.Query("site_id"))
	var total int64
	if err := q.Count(&total).Error; err != nil {
		taxError(c, err)
		return
	}
	if err := q.Omit("file_content").Order("id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		taxError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": rows, "total": total})
}

// A download log records file delivery by the server, not client-side disk completion.
func deliverTaxFile(c *gin.Context, record model.TaxDownloadLog) {
	id, ok := c.Get("userID")
	if !ok {
		c.JSON(401, gin.H{"error": "请登录"})
		return
	}
	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		taxError(c, err)
		return
	}
	record.ID = 0
	record.CreatedAt = time.Time{}
	record.UserID = user.ID
	record.Username = user.Name
	if record.Username == "" {
		record.Username = user.Username
	}
	if err := database.DB.Create(&record).Error; err != nil {
		taxError(c, err)
		return
	}
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": record.FileName}))
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, record.ContentType, record.FileContent)
}

func DownloadTaxImport(c *gin.Context) {
	var record model.TaxImportLog
	// An empty site_id means "all countries": allow downloading any site's log.
	if strings.TrimSpace(c.Query("site_id")) == "" {
		if err := database.DB.First(&record, "id = ? AND platform = ?", c.Param("id"), c.DefaultQuery("platform", "walmart")).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(404, gin.H{"error": "导入记录不存在"})
			} else {
				taxError(c, err)
			}
			return
		}
	} else if err := database.DB.First(&record, "id = ? AND platform = ? AND (site_id = ? OR site_id = '' OR site_id IS NULL)", c.Param("id"), c.DefaultQuery("platform", "walmart"), taxSiteFromContext(c)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "导入记录不存在"})
		} else {
			taxError(c, err)
		}
		return
	}
	kind := "original"
	if record.Platform == "tiktok" {
		kind = "original_" + record.ReportType
	}
	deliverTaxFile(c, model.TaxDownloadLog{Platform: record.Platform, SiteID: record.SiteID, ReportType: kind, FileName: record.FileName, FileContent: record.FileContent, ContentType: "application/octet-stream", RowCount: record.SourceRows})
}

func DownloadTaxHistoryFile(c *gin.Context) {
	var record model.TaxDownloadLog
	// An empty site_id means "all countries": allow downloading any site's file.
	if strings.TrimSpace(c.Query("site_id")) == "" {
		if err := database.DB.First(&record, "id = ? AND platform = ?", c.Param("id"), c.DefaultQuery("platform", "walmart")).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(404, gin.H{"error": "下载记录不存在"})
			} else {
				taxError(c, err)
			}
			return
		}
	} else if err := database.DB.First(&record, "id = ? AND platform = ? AND (site_id = ? OR site_id = '' OR site_id IS NULL)", c.Param("id"), c.DefaultQuery("platform", "walmart"), taxSiteFromContext(c)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "下载记录不存在"})
		} else {
			taxError(c, err)
		}
		return
	}
	deliverTaxFile(c, record)
}

// Protect spreadsheet consumers from formulas in untrusted original text fields.
func taxCSVText(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsAny(trimmed[:1], "=+@-") {
		if _, err := decimal.NewFromString(trimmed); err != nil {
			return "'" + value
		}
	}
	return value
}

func taxReportFileName(c *gin.Context, platform, label, currency string) string {
	store := strings.TrimSpace(c.Query("store_id"))
	if store == "" {
		store = "全部店铺"
	}
	site := strings.TrimSpace(c.Query("site_id"))
	if site == "" {
		site = "全部国家"
	} else if name := map[string]string{"US": "美国", "CA": "加拿大", "UK": "英国", "GB": "英国", "DE": "德国", "FR": "法国", "JP": "日本", "AU": "澳大利亚"}[site]; name != "" {
		site = name + "-" + site
	}
	parts := []string{platform, label, store, site}
	if from, to := c.Query("date_from"), c.Query("date_to"); from != "" || to != "" {
		if from == "" {
			from = "不限"
		}
		if to == "" {
			to = "不限"
		}
		parts = append(parts, from+"至"+to)
	}
	for _, key := range []string{"quarter", "month"} {
		if value := c.Query(key); value != "" {
			parts = append(parts, value)
		}
	}
	if len(parts) == 4 {
		parts = append(parts, "全部时间")
	}
	if currency != "" {
		parts = append(parts, currency)
	}
	for i, part := range parts {
		parts[i] = strings.Map(func(r rune) rune {
			if r < 32 || r == 34 || r == 92 || strings.ContainsRune("<>:/|?*", r) {
				return '-'
			}
			return r
		}, part)
	}
	return strings.Join(parts, "_") + "_" + time.Now().Format("20060102_150405") + ".csv"
}

func ExportTaxReport(c *gin.Context) {
	kind, quarter := c.Query("type"), c.Query("quarter")
	if kind != "raw" && kind != "quarters" {
		c.JSON(400, gin.H{"error": "type 必须为 raw 或 quarters"})
		return
	}
	currency := ""
	if kind == "quarters" {
		currency = strings.ToUpper(c.DefaultQuery("currency", "CNY"))
		if currency != "USD" && currency != "CNY" {
			c.JSON(400, gin.H{"error": "仅支持美元或人民币"})
			return
		}
		from, to := c.Query("date_from"), c.Query("date_to")
		if (from != "" && !validTaxDate(from)) || (to != "" && !validTaxDate(to)) || (from != "" && to != "" && from > to) {
			c.JSON(400, gin.H{"error": "日期范围无效"})
			return
		}
	}
	var output bytes.Buffer
	output.Write([]byte{0xef, 0xbb, 0xbf})
	writer := csv.NewWriter(&output)
	count := 0
	if kind == "raw" {
		type exportRow struct {
			model.TaxRawRow
			Month   string
			Quarter string
			FeeType string
			Fee     decimal.Decimal
			Rate    decimal.Decimal
		}
		rows := []exportRow{}
		q := database.DB.Table("tax_raw_rows r").Select("r.*, p.month, p.quarter, p.fee_type, p.fee, p.rate").Joins("JOIN tax_processed_rows p ON p.raw_row_id = r.id").Where("BTRIM(COALESCE(r.data->>'Transaction Type', '')) <> ?", "PaymentSummary")
		q = q.Where("r.id IN (?)", applyTaxRawFilters(taxRawBaseQuery(c), taxRawFilters(c)).Select("id"))
		if err := q.Order("r.id").Find(&rows).Error; err != nil {
			taxError(c, err)
			return
		}
		headers := append([]string{}, taxHeaders...)
		known := map[string]bool{}
		for _, h := range headers {
			known[h] = true
		}
		extra := []string{}
		for _, row := range rows {
			for h := range row.Data {
				if !known[h] {
					known[h] = true
					extra = append(extra, h)
				}
			}
		}
		sort.Strings(extra)
		headers = append(headers, extra...)
		_ = writer.Write(append([]string{"月份", "季度", "费用类型", "费用 CNY", "入库汇率"}, headers...))
		for _, row := range rows {
			values := []string{row.Month, row.Quarter, taxCSVText(row.FeeType), row.Fee.StringFixed(2), row.Rate.String()}
			for _, h := range headers {
				values = append(values, taxCSVText(row.Data[h]))
			}
			_ = writer.Write(values)
		}
		count = len(rows)
	} else {
		rows, categories, err := summarizeTaxProcessed(database.DB, c.Query("store_id"), quarter, c.Query("site_id"))
		if err != nil {
			taxError(c, err)
			return
		}
		headers := append([]string{"季度"}, categories...)
		_ = writer.Write(append(headers, "总计 "+currency, "报税收入 "+currency, "未匹配条数"))
		for _, row := range rows {
			start, end := quarterDateRange(row.Quarter)
			if from, to := c.Query("date_from"), c.Query("date_to"); (from != "" && end < from) || (to != "" && start > to) {
				continue
			}
			totals, total, income := row.Totals, row.Total, row.TaxableIncome
			if currency == "USD" {
				totals, total, income = row.USDTotals, row.USDTotal, row.USDTaxableIncome
			}
			values := []string{row.Quarter}
			for _, category := range categories {
				values = append(values, totals[category].StringFixed(2))
			}
			_ = writer.Write(append(values, total.StringFixed(2), income.StringFixed(2), fmt.Sprint(row.UnmatchedRows)))
			count++
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		taxError(c, err)
		return
	}
	label := "原始账单"
	if kind == "quarters" {
		label = "季度汇总"
	}
	deliverTaxFile(c, model.TaxDownloadLog{Platform: c.DefaultQuery("platform", "walmart"), SiteID: taxSiteFromContext(c), ReportType: kind, Quarter: quarter, Currency: currency, FileName: taxReportFileName(c, "Walmart", label, currency), FileContent: output.Bytes(), ContentType: "text/csv; charset=utf-8", RowCount: count})
}
