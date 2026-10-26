package handler

import (
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed amazon_tax_headers.json
var amazonTaxHeaderJSON []byte
var amazonTaxHeaders = func() map[string][]string {
	var headers map[string][]string
	if err := json.Unmarshal(amazonTaxHeaderJSON, &headers); err != nil {
		panic(err)
	}
	return headers
}()

func amazonTaxTable(kind string) string {
	if kind == "shipments" {
		return "amazon_tax_shipments"
	}
	if kind == "finance" {
		return "amazon_tax_finances"
	}
	return ""
}
func amazonTaxFilePrefix(store, site, kind string) string {
	if kind == "shipments" {
		return fmt.Sprintf("%s_%s_亚马逊配送货件", store, site)
	}
	if kind == "finance_summary" {
		return fmt.Sprintf("%s_%s_付款报告_联合报告_一览", store, site)
	}
	if kind == "finance" {
		return fmt.Sprintf("%s_%s_付款报告_联合报告_交易", store, site)
	}
	return ""
}

func UploadAmazonTaxShipments(c *gin.Context) { uploadAmazonTax(c, "shipments") }
func UploadAmazonTaxFinance(c *gin.Context)   { uploadAmazonTax(c, "finance") }
func uploadAmazonTax(c *gin.Context, kind string) {
	var audit *model.TaxImportLog
	respond := func(code int, ok bool, msg string) { respondTikTokImport(c, audit, code, ok, msg) }
	if !taxUploadAuthorized(c) {
		respond(401, false, "RPA 密钥无效")
		return
	}
	limit := int64(taxMaxFile + (1 << 20))
	if kind == "finance" {
		limit += taxMaxFile
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	file, header, err := c.Request.FormFile("file")
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		respond(400, false, "缺少 file 文件或文件超过 20MB")
		return
	}
	store := strings.TrimSpace(c.PostForm("store_id"))
	if store == "" {
		store = "sunoutly"
	}
	site := taxSiteFromForm(c)
	fileName := filepath.Base(header.Filename)
	if prefix := amazonTaxFilePrefix(store, site, kind); prefix != "" && !strings.HasPrefix(strings.ToLower(fileName), strings.ToLower(prefix)) {
		respond(400, false, fmt.Sprintf("文件名不符合要求，必须以前缀 %s 开头", prefix))
		return
	}
	if kind == "finance" && !strings.EqualFold(strings.TrimSpace(c.PostForm("confirm_duplicate")), "1") {
		var existingFile model.TaxImportLog
		duplicateQuery := database.DB.Where("platform = ? AND report_type = ? AND file_name = ? AND status IN ? AND store_id = ? AND site_id = ?", "amazon", kind, fileName, []string{"locked", "duplicate"}, store, site)
		if err := duplicateQuery.Order("id desc").First(&existingFile).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"status": false, "requires_confirmation": true, "file_name": fileName, "msg": fmt.Sprintf("平台 Amazon、店铺 %s、站点 %s 已导入过文件 %s，是否继续导入？", store, site, fileName)})
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			respond(500, false, "检查财务报告文件重复记录失败")
			return
		}
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, taxMaxFile+1))
	if err != nil || len(content) > taxMaxFile {
		respond(400, false, "文件读取失败或超过 20MB")
		return
	}
	audit = &model.TaxImportLog{Platform: "amazon", ReportType: kind, StoreID: store, SiteID: site, Importer: taxImporterLabel(c), FileName: fileName, FileContent: content, Status: "processing"}
	if err = database.DB.Create(audit).Error; err != nil {
		audit = nil
		respond(500, false, "创建导入日志失败")
		return
	}
	rows, err := parseAmazonTaxFile(content, filepath.Ext(header.Filename), kind)
	if err != nil {
		respond(400, false, err.Error())
		return
	}
	audit.SourceRows = len(rows)
	var summaryLog *model.TaxImportLog
	summaryDuplicate := false
	summaryPeriod := ""
	if kind == "finance" {
		pdf, header, err := c.Request.FormFile("summary_file")
		if err != nil {
			respond(400, false, "财务明细必须同时上传 summary_file 汇总 PDF")
			return
		}
		defer pdf.Close()
		content, err := io.ReadAll(io.LimitReader(pdf, taxMaxFile+1))
		if err != nil || len(content) > taxMaxFile || !strings.EqualFold(filepath.Ext(header.Filename), ".pdf") {
			respond(400, false, "汇总文件必须为不超过 20MB 的 PDF")
			return
		}
		summary, err := parseAmazonSummary(c.Request.Context(), content)
		if err != nil {
			respond(400, false, err.Error())
			return
		}
		if err := validateAmazonSummary(database.DB.WithContext(c.Request.Context()), rows, summary); err != nil {
			respond(422, false, err.Error())
			return
		}
		message, _ := json.Marshal(summary)
		summaryPeriod = fmt.Sprintf("%s 至 %s", summary.Start, summary.End)
		var existing []model.TaxImportLog
		if err := database.DB.Where("platform = ? AND report_type = ? AND status = ? AND store_id = ? AND site_id = ?", "amazon", "finance_summary", "locked", store, site).Find(&existing).Error; err != nil {
			respond(500, false, "检查汇总 PDF 重复记录失败")
			return
		}
		for _, record := range existing {
			var previous amazonSummary
			if json.Unmarshal([]byte(record.Message), &previous) == nil && sameAmazonSummaryIdentity(previous, summary) {
				summaryDuplicate = true
				break
			}
		}
		if !summaryDuplicate {
			summaryLog = &model.TaxImportLog{Platform: "amazon", ReportType: "finance_summary", StoreID: store, SiteID: site, Importer: taxImporterLabel(c), FileName: filepath.Base(header.Filename), FileContent: content, Status: "locked", Message: string(message)}
		}
	}
	imported := 0
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if summaryLog != nil {
			if err := tx.Exec("LOCK TABLE tax_import_logs IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
				return err
			}
			var existing []model.TaxImportLog
			if err := tx.Where("platform = ? AND report_type = ? AND status = ? AND store_id = ? AND site_id = ?", "amazon", "finance_summary", "locked", store, site).Find(&existing).Error; err != nil {
				return err
			}
			var incoming amazonSummary
			if err := json.Unmarshal([]byte(summaryLog.Message), &incoming); err != nil {
				return err
			}
			for _, record := range existing {
				var previous amazonSummary
				if json.Unmarshal([]byte(record.Message), &previous) == nil && sameAmazonSummaryIdentity(previous, incoming) {
					summaryDuplicate = true
					break
				}
			}
			if !summaryDuplicate {
				summaryLog.BatchID = audit.ID
				if err := tx.Create(summaryLog).Error; err != nil {
					return err
				}
			}
		}
		for i := range rows {
			rows[i].ImportLogID = audit.ID
			rows[i].StoreID = store
			rows[i].SiteID = site
		}
		insert := tx.Table(amazonTaxTable(kind))
		if kind == "shipments" {
			insert = insert.Clauses(clause.OnConflict{DoNothing: true})
		}
		result := insert.CreateInBatches(&rows, 300)
		if result.Error != nil {
			return result.Error
		}
		imported = int(result.RowsAffected)
		return finishTikTokImport(tx, audit, audit.ID, imported, len(rows)-imported, "locked")
	})
	if err != nil {
		respond(500, false, "导入失败，本次未入库："+err.Error())
		return
	}
	if kind == "finance" && imported == 0 && summaryDuplicate {
		respond(409, false, "财务明细和汇总 PDF 均已重复，未新增数据")
		return
	}
	message := fmt.Sprintf("财务明细导入成功，入库 %d 条，跳过重复 %d 条", imported, len(rows)-imported)
	if summaryDuplicate {
		message += "；汇总 PDF " + summaryPeriod + " 已重复跳过"
	} else if summaryLog != nil {
		message += "；汇总 PDF 已入库"
	}
	respond(201, true, message)
}

func parseAmazonTaxDate(value string) time.Time {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(
		"a.m.", "AM", "p.m.", "PM", "a.m", "AM", "p.m", "PM",
		"A.M.", "AM", "P.M.", "PM", "A.M", "AM", "P.M", "PM",
	).Replace(value)
	if date := parseTikTokPostedAt(value); !date.IsZero() {
		return date
	}
	for _, layout := range []string{"Jan 2, 2006 3:04:05 PM MST", "Jan 2, 2006 3:04:05 PM", "Jan 2, 2006 15:04:05 MST", "Jan 2, 2006", "2 Jan 2006 15:04:05 MST", "2006-01-02T15:04:05-0700"} {
		if date, err := time.Parse(layout, value); err == nil {
			return date
		}
	}
	return time.Time{}
}
func parseAmazonTaxFile(content []byte, ext, kind string) ([]model.AmazonTaxShipment, error) {
	rows, date1904, excelDates, err := readTikTokFirstWorksheet(content, ext)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("文件没有数据行")
	}
	// Locate the header by names, including reports with introductory lines.
	columns := map[string]int{}
	headerRow := -1
	identity := []string{"亚马逊订单编号", "货件编号", "货件商品编号", "配送日期"}
	if kind == "finance" {
		identity = []string{"date/time", "settlement id", "order id"}
	}
	for ri, row := range rows {
		candidate := map[string]int{}
		for i, header := range row {
			candidate[strings.ToLower(strings.TrimSpace(header))] = i
		}
		complete := true
		for _, key := range identity {
			if _, ok := candidate[strings.ToLower(key)]; !ok {
				complete = false
			}
		}
		if complete {
			columns, headerRow = candidate, ri
			break
		}
	}
	if headerRow < 0 {
		return nil, fmt.Errorf("未找到表头，必须包含：%s", strings.Join(identity, "、"))
	}
	// Duplicate names are ambiguous; never silently map to the wrong column.
	seen := map[string]bool{}
	for _, header := range rows[headerRow] {
		key := strings.ToLower(strings.TrimSpace(header))
		if key == "" {
			continue
		}
		if seen[key] {
			return nil, fmt.Errorf("表头重复：%s", header)
		}
		seen[key] = true
	}
	dateKey, idKeys := "配送日期", []string{"亚马逊订单编号", "货件编号", "货件商品编号"}
	if kind == "finance" {
		dateKey, idKeys = "date/time", []string{"date/time", "settlement id", "order id", "sku", "total"}
	}
	output := []model.AmazonTaxShipment{}
	for ri := headerRow + 1; ri < len(rows); ri++ {
		values := rows[ri]
		if strings.TrimSpace(strings.Join(values, "")) == "" {
			continue
		}
		data := map[string]string{}
		for _, key := range amazonTaxHeaders[kind] {
			i, ok := columns[strings.ToLower(key)]
			if ok && i < len(values) {
				data[key] = strings.TrimSpace(values[i])
			} else {
				data[key] = ""
			}
		}
		if excelDates {
			for _, key := range []string{"购买日期", "付款日期", "配送日期", "报告日期", "预计配送日期", "date/time", "Transaction Release Date"} {
				if serial, err := strconv.ParseFloat(data[key], 64); err == nil && serial >= 1 && serial < 100000 {
					date, err := excelize.ExcelDateToTime(serial, date1904)
					if err != nil {
						return nil, fmt.Errorf("第 %d 行 %s 无效", ri+1, key)
					}
					data[key] = date.Format("2006-01-02 15:04:05")
				}
			}
		}
		date := parseAmazonTaxDate(data[dateKey])
		if date.IsZero() {
			return nil, fmt.Errorf("第 %d 行 %s 日期无效：%q", ri+1, dateKey, data[dateKey])
		}
		parts := []string{}
		for _, key := range idKeys {
			parts = append(parts, data[key])
		}
		// JSON preserves field boundaries even when source values contain separators.
		encoded, _ := json.Marshal(parts)
		output = append(output, model.AmazonTaxShipment{Month: date.Format("2006-01"), Quarter: fmt.Sprintf("%d-Q%d", date.Year(), (int(date.Month())+2)/3), Fingerprint: fmt.Sprintf("%x", sha256.Sum256(encoded)), Data: data})
	}
	if len(output) == 0 {
		return nil, fmt.Errorf("文件没有数据行")
	}
	return output, nil
}

func ListAmazonTaxRows(c *gin.Context) {
	kind := c.Param("kind")
	if kind == "quarters" {
		ListAmazonTaxQuarters(c)
		return
	}
	table := amazonTaxTable(kind)
	if table == "" {
		c.JSON(400, gin.H{"error": "不支持的报表类型"})
		return
	}
	page, size := taxPage(c)
	q := filterTikTokTax(database.DB.Table(table), c.Query("month"), c.Query("quarter"))
	if c.Query("store_id") != "" {
		q = q.Where("(store_id = ? OR store_id = '')", c.Query("store_id"))
	}
	if kind != "shipments" {
		q = filterTaxSite(q, c.Query("site_id"))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		taxError(c, err)
		return
	}
	rows := []model.AmazonTaxShipment{}
	if err := q.Order("id").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		taxError(c, err)
		return
	}
	response := gin.H{"items": rows, "total": total, "columns": amazonTaxHeaders[kind]}
	if wantsTaxTotals(c.Query("include_totals")) {
		allRows := []model.AmazonTaxShipment{}
		if err := q.Limit(-1).Offset(-1).Order("id").Find(&allRows).Error; err != nil {
			taxError(c, err)
			return
		}
		totals := map[string]decimal.Decimal{}
		for _, row := range allRows {
			addJSONTotals(totals, row.Data)
		}
		response["totals"] = totals
	}
	c.JSON(200, response)
}
func ListAmazonTaxFilterOptions(c *gin.Context) {
	table, field := amazonTaxTable(c.Query("type")), c.Query("field")
	if table == "" || (field != "month" && field != "quarter") {
		c.JSON(400, gin.H{"error": "不支持的筛选字段"})
		return
	}
	month, quarter := c.Query("month"), c.Query("quarter")
	if field == "month" {
		month = ""
	} else {
		quarter = ""
	}
	values := []string{}
	q := filterTikTokTax(database.DB.Table(table), month, quarter)
	if c.Query("store_id") != "" {
		q = q.Where("(store_id = ? OR store_id = '')", c.Query("store_id"))
	}
	if c.Query("type") != "shipments" {
		q = filterTaxSite(q, c.Query("site_id"))
	}
	if err := q.Distinct(field).Order(field+" desc").Pluck(field, &values).Error; err != nil {
		taxError(c, err)
		return
	}
	c.JSON(200, gin.H{"values": values})
}
