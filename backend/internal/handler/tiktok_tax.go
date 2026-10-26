package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var tikTokTaxHeaders = []string{"业务时间（订单结算、费用转账的完成时间）", "业务单号（订单ID、调整单ID）", "币种", "收入金额", "退款金额", "支付给平台的佣金/服务费金额"}

func tikTokTaxCell(data map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			return strings.TrimSpace(value)
		}
	}
	for dataKey, value := range data {
		normalizedDataKey := normalizeTikTokTaxHeader(dataKey)
		for _, key := range keys {
			if normalizedDataKey == normalizeTikTokTaxHeader(key) {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func normalizeTikTokTaxHeader(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "（", "(")
	value = strings.ReplaceAll(value, "）", ")")
	return value
}

var tikTokTaxHeaderAliases = [][]string{
	{"业务时间（订单结算、费用转账的完成时间）", "交易入账时间"},
	{"业务单号（订单ID、调整单ID）", "业务单号"},
	{"币种"},
	{"收入金额", "收入金额（美元）"},
	{"退款金额", "退款金额（美元）"},
	{"支付给平台的佣金/服务费金额", "支付给平台的佣金/服务费金额（美元）", "平台佣金/服务费金额", "平台佣金/服务费金额（美元）", "佣金/服务费金额", "佣金/服务费金额（美元）"},
}

func UploadTikTokTaxData(c *gin.Context) {
	var audit *model.TaxImportLog
	respond := func(code int, ok bool, msg string) { respondTikTokImport(c, audit, code, ok, msg) }
	if !taxUploadAuthorized(c) {
		respond(http.StatusUnauthorized, false, "TikTok RPA 密钥无效")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		respond(http.StatusBadRequest, false, "缺少 file 文件")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		respond(http.StatusBadRequest, false, "文件读取失败")
		return
	}
	store := strings.TrimSpace(c.PostForm("store_id"))
	if store == "" {
		store = "Forious"
	}
	site := taxSiteFromForm(c)
	audit, err = beginTikTokImport("details", header.Filename, store, site, taxImporterLabel(c), content)
	if err != nil {
		respond(http.StatusBadRequest, false, err.Error())
		return
	}
	if ext := strings.ToLower(filepath.Ext(header.Filename)); ext != ".csv" && ext != ".xlsx" && ext != ".xls" {
		respond(http.StatusBadRequest, false, "仅支持 CSV、XLSX、XLS 文件")
		return
	}
	sheetName := strings.TrimSpace(c.PostForm("sheet_name"))
	summarySheet := ""
	if strings.EqualFold(filepath.Ext(header.Filename), ".xlsx") {
		summarySheet, sheetName, err = tikTokNamedWorksheets(content, sheetName)
		if err != nil {
			respond(http.StatusBadRequest, false, err.Error())
			return
		}
	}
	rows, err := parseTikTokTaxFile(content, filepath.Ext(header.Filename), sheetName)
	if err != nil {
		respond(http.StatusBadRequest, false, err.Error())
		return
	}
	audit.SourceRows = len(rows)
	var transactionSummaries []model.TikTokTaxTransactionSummary
	if ext := strings.ToLower(filepath.Ext(header.Filename)); ext == ".xlsx" || ext == ".xls" {
		transactionSummaries, err = parseTikTokTransactionSummaryWithSheet(content, ext, summarySheet)
		if err != nil {
			respond(http.StatusBadRequest, false, "同一工作簿汇总表"+err.Error())
			return
		}
	}
	for i := range transactionSummaries {
		transactionSummaries[i].StoreID = store
		transactionSummaries[i].SiteID = site
	}
	hashContent := content
	if strings.EqualFold(filepath.Ext(header.Filename), ".xlsx") {
		// Preserve the default first-sheet identity for existing imports.
		workbook, err := excelize.OpenReader(bytes.NewReader(content))
		if err != nil {
			respond(http.StatusBadRequest, false, err.Error())
			return
		}
		if sheetName != workbook.GetSheetName(0) {
			hashContent = append(append([]byte(nil), content...), []byte("\x00"+sheetName)...)
		}
		workbook.Close()
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(hashContent))
	var details []model.TikTokTaxDetail
	for i, row := range rows {
		dateValue := tikTokTaxCell(row.Data, tikTokTaxHeaderAliases[0]...)
		posted := parseTikTokPostedAt(dateValue).Truncate(time.Microsecond)
		if posted.IsZero() {
			returnTikTokError(respond, fmt.Sprintf("第 %d 行业务时间无效，单元格原值：%q", i+2, dateValue))
			return
		}
		currency := strings.ToUpper(tikTokTaxCell(row.Data, tikTokTaxHeaderAliases[2]...))
		if currency == "" {
			currency = "USD"
		}
		income, e1 := parseTaxAmount(tikTokTaxCell(row.Data, tikTokTaxHeaderAliases[3]...))
		refund, e2 := parseTaxAmount(tikTokTaxCell(row.Data, tikTokTaxHeaderAliases[4]...))
		commission, e3 := parseTaxAmount(tikTokTaxCell(row.Data, tikTokTaxHeaderAliases[5]...))
		if e1 != nil || e2 != nil || e3 != nil {
			respond(http.StatusBadRequest, false, fmt.Sprintf("第 %d 行金额无效", i+2))
			return
		}
		rate := decimal.NewFromInt(1)
		if currency != "CNY" {
			rate = decimal.Zero
			var fx model.TaxExchangeRate
			rateDate := taxRateDate(posted)
			// Prefer the normalized trading date, then the next published rate
			// within 14 days (weekends and market holidays). If none exists,
			// keep the raw USD row and let the summary show it as pending.
			if database.DB.Where("date = ? AND currency = ? AND rate > 0", rateDate, currency).First(&fx).Error != nil {
				start, _ := time.Parse("2006-01-02", rateDate)
				database.DB.Where("date > ? AND date <= ? AND currency = ? AND rate > 0", rateDate, start.AddDate(0, 0, 14).Format("2006-01-02"), currency).Order("date").First(&fx)
			}
			if fx.Rate.IsPositive() {
				rate = fx.Rate
			}
		}
		quarter := fmt.Sprintf("%d-Q%d", posted.Year(), (int(posted.Month())-1)/3+1)
		details = append(details, model.TikTokTaxDetail{StoreID: store, SiteID: site, RawRowID: uint(i + 1), Month: posted.Format("2006-01"), Quarter: quarter, Fee: income.Sub(refund).Sub(commission).Mul(rate).Round(2), PostedAt: posted, BusinessID: tikTokTaxCell(row.Data, tikTokTaxHeaderAliases[1]...), Income: income, Refund: refund, Commission: commission, Rate: rate})

	}
	batch := model.TikTokTaxBatch{StoreID: store, SiteID: site, FileHash: hash, FileName: header.Filename, FileContent: content, RawRows: len(rows)}
	imported := 0
	duplicateFile := false
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Serialize check-and-insert across instances, without deleting legacy duplicates.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "tiktok-tax-detail-import").Error; err != nil {
			return err
		}
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "file_hash"}}, DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			duplicateFile = true
			if err := tx.Where("file_hash = ?", hash).First(&batch).Error; err != nil {
				return err
			}
		}
		var existing []model.TikTokTaxDetail
		if err := tx.Select("business_id").Find(&existing).Error; err != nil {
			return err
		}
		keyOf := func(detail model.TikTokTaxDetail) string {
			return strings.TrimSpace(detail.BusinessID)
		}
		seen := make(map[string]bool, len(existing)+len(details))
		for _, detail := range existing {
			if key := keyOf(detail); key != "" {
				seen[key] = true
			}
		}
		for i := range rows {
			key := keyOf(details[i])
			if key != "" && seen[key] {
				continue
			}
			if key != "" {
				seen[key] = true
			}
			rows[i].BatchID = batch.ID
			rows[i].StoreID = store
			rows[i].SiteID = site
			// Raw-row identity is local to this batch; business deduplication happens above.
			rows[i].Fingerprint = fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("tiktok:%d:%d", batch.ID, i))))
			rawResult := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "fingerprint"}}, DoNothing: true}).Create(&rows[i])
			if rawResult.Error != nil {
				return rawResult.Error
			}
			if rawResult.RowsAffected == 0 {
				var raw model.TikTokTaxRawRow
				if err := tx.Where("fingerprint = ?", rows[i].Fingerprint).First(&raw).Error; err != nil {
					return err
				}
				rows[i].ID = raw.ID
			}
			details[i].BatchID = batch.ID
			details[i].RawRowID = rows[i].ID
			detailResult := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "raw_row_id"}}, DoNothing: true}).Create(&details[i])
			if detailResult.Error != nil {
				return detailResult.Error
			}
			imported += int(detailResult.RowsAffected)
		}
		var batchDetailCount int64
		if err := tx.Model(&model.TikTokTaxDetail{}).Where("batch_id = ?", batch.ID).Count(&batchDetailCount).Error; err != nil {
			return err
		}
		if err := tx.Model(&batch).Update("raw_rows", batchDetailCount).Error; err != nil {
			return err
		}
		if err := saveTikTokTransactionSummaries(tx, batch.ID, transactionSummaries); err != nil {
			return err
		}
		status := "locked"
		if duplicateFile && imported == 0 {
			status = "duplicate"
		}
		return finishTikTokImport(tx, audit, batch.ID, imported, len(rows)-imported, status)
	})
	if err != nil {
		respond(http.StatusInternalServerError, false, "导入失败："+err.Error())
		return
	}
	if duplicateFile {
		if imported > 0 {
			respond(http.StatusOK, true, fmt.Sprintf("已补充入库 %d 条，跳过重复 %d 条", imported, len(rows)-imported))
			return
		}
		respond(http.StatusOK, true, "文件已导入")
		return
	}
	respond(http.StatusCreated, true, fmt.Sprintf("TikTok 数据导入成功，入库 %d 条，跳过重复 %d 条", imported, len(rows)-imported))
}

func saveTikTokTransactionSummaries(tx *gorm.DB, batchID uint, summaries []model.TikTokTaxTransactionSummary) error {
	if len(summaries) == 0 || !tx.Migrator().HasTable(&model.TikTokTaxTransactionSummary{}) {
		return nil
	}
	for _, summary := range summaries {
		summary.BatchID = batchID
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "store_id"}, {Name: "quarter"}}, DoUpdates: clause.AssignmentColumns([]string{"batch_id", "transaction_count"})}).Create(&summary).Error; err != nil {
			return err
		}
	}
	return nil
}

func parseTikTokPostedAt(value string) time.Time {
	value = strings.TrimSpace(value)
	if posted := parsePostedAt(value); !posted.IsZero() {
		return posted
	}
	for _, layout := range []string{"2006/1/2 15:04", "2006/1/2", "2006-1-2 15:04:05", "2006-1-2 15:04", "2006-1-2", "2006年1月2日 15:04:05", "2006年1月2日"} {
		if posted, err := time.Parse(layout, value); err == nil {
			return posted
		}
	}
	return time.Time{}
}

func returnTikTokError(respond func(int, bool, string), msg string) {
	respond(http.StatusBadRequest, false, msg)
}

func parseTikTokTaxFile(content []byte, ext string, sheetName string) ([]model.TikTokTaxRawRow, error) {
	var rows [][]string
	if strings.ToLower(ext) == ".csv" {
		if sheetName != "" {
			return nil, fmt.Errorf("CSV 文件没有工作表，请不要传 sheet_name")
		}
		r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf})))
		r.FieldsPerRecord = -1
		for {
			row, e := r.Read()
			if e == io.EOF {
				break
			}
			if e != nil {
				return nil, e
			}
			rows = append(rows, row)
		}
	} else if strings.ToLower(ext) == ".xlsx" {
		f, e := excelize.OpenReader(bytes.NewReader(content))
		if e != nil {
			return nil, e
		}
		defer f.Close()
		if sheetName == "" {
			return nil, fmt.Errorf("请指定名称包含“明细”或“detail”的工作表")
		} else {
			index, err := f.GetSheetIndex(sheetName)
			if err != nil || index < 0 {
				return nil, fmt.Errorf("指定工作表不存在")
			}
		}
		it, e := f.Rows(sheetName)
		if e != nil {
			return nil, e
		}
		defer it.Close()
		for it.Next() {
			row, e := it.Columns()
			if e != nil {
				return nil, e
			}
			rows = append(rows, row)
		}
		if err := it.Error(); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("TikTok 仅支持 CSV、XLSX 文件")
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("账单没有数据行")
	}
	headerRow := -1
	headers := []string(nil)
	_ = csv.ErrFieldCount
	for i, candidate := range rows {
		for j := range candidate {
			candidate[j] = strings.TrimSpace(candidate[j])
		}
		if tikTokTaxHeadersMatch(candidate) {
			headerRow = i
			headers = candidate
			break
		}
	}
	if headerRow < 0 {
		return nil, fmt.Errorf("申报明细表缺少必需表头")
	}
	var result []model.TikTokTaxRawRow
	for _, values := range rows[headerRow+1:] {
		data := map[string]string{}
		for i, header := range headers {
			if i < len(values) {
				data[header] = strings.TrimSpace(values[i])
			}
		}
		result = append(result, model.TikTokTaxRawRow{Data: data})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("账单没有数据行")
	}
	return result, nil
}

func tikTokTaxHeadersMatch(headers []string) bool {
	for _, aliases := range tikTokTaxHeaderAliases {
		found := false
		for _, alias := range aliases {
			for _, header := range headers {
				if normalizeTikTokTaxHeader(header) == normalizeTikTokTaxHeader(alias) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func parseTikTokTransactionSummary(content []byte, ext string) ([]model.TikTokTaxTransactionSummary, error) {
	return parseTikTokTransactionSummaryOnSheet(content, ext, "")
}

func parseTikTokTransactionSummaryOnSheet(content []byte, ext, sheetName string) ([]model.TikTokTaxTransactionSummary, error) {
	rows, _, _, err := readTikTokFirstWorksheet(content, ext)
	if sheetName != "" {
		rows, err = readTikTokWorksheetByName(content, sheetName)
	}
	if err != nil {
		return nil, err
	}
	headerRow, quarterIndex, countIndex := -1, -1, -1
	for i, row := range rows {
		for j, cell := range row {
			switch strings.TrimSpace(cell) {
			case "年份-季度":
				quarterIndex = j
			case "交易笔数":
				countIndex = j
			}
		}
		if quarterIndex >= 0 && countIndex >= 0 {
			headerRow = i
			break
		}
		quarterIndex, countIndex = -1, -1
	}
	if headerRow < 0 {
		return nil, fmt.Errorf("汇总表缺少表头：年份-季度、交易笔数")
	}
	byQuarter := map[string]int64{}
	for _, row := range rows[headerRow+1:] {
		if quarterIndex >= len(row) || countIndex >= len(row) {
			continue
		}
		quarter := normalizeTikTokQuarter(row[quarterIndex])
		if quarter == "" {
			continue
		}
		count, err := strconv.ParseInt(strings.ReplaceAll(strings.TrimSpace(row[countIndex]), ",", ""), 10, 64)
		if err != nil || count < 0 {
			return nil, fmt.Errorf("汇总表季度 %s 的交易笔数无效", quarter)
		}
		byQuarter[quarter] = count
	}
	result := make([]model.TikTokTaxTransactionSummary, 0, len(byQuarter))
	for quarter, count := range byQuarter {
		result = append(result, model.TikTokTaxTransactionSummary{Quarter: quarter, TransactionCount: count})
	}
	return result, nil
}

var tikTokQuarterPattern = regexp.MustCompile(`(?i)^(\d{4}).*?(?:q|第)\s*([1-4])`)

func normalizeTikTokQuarter(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	matches := tikTokQuarterPattern.FindStringSubmatch(value)
	if len(matches) == 3 {
		return matches[1] + "-Q" + matches[2]
	}
	return value
}

func parseTikTokTransactionSummaryWithSheet(content []byte, ext, sheetName string) ([]model.TikTokTaxTransactionSummary, error) {
	return parseTikTokTransactionSummaryOnSheet(content, ext, sheetName)
}

func tikTokNamedWorksheets(content []byte, detailSheet string) (string, string, error) {
	book, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return "", "", err
	}
	defer book.Close()
	names := book.GetSheetList()
	find := func(role string) string {
		for _, name := range names {
			normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(name), " ", ""), "　", ""))
			if role == "summary" && (strings.Contains(normalized, "汇总") || strings.Contains(normalized, "summary")) {
				return name
			}
			if role == "detail" && (strings.Contains(normalized, "明细") || strings.Contains(normalized, "detail")) {
				return name
			}
		}
		return ""
	}
	summary := find("summary")
	detail := strings.TrimSpace(detailSheet)
	if detail == "" {
		detail = find("detail")
	}
	if summary == "" {
		return "", "", fmt.Errorf("未找到名称包含“汇总”或“summary”的汇总工作表")
	}
	if detail == "" {
		return "", "", fmt.Errorf("未找到名称包含“明细”或“detail”的明细工作表")
	}
	if summary == detail {
		return "", "", fmt.Errorf("汇总工作表和明细工作表不能是同一张")
	}
	return summary, detail, nil
}

func readTikTokWorksheetByName(content []byte, sheetName string) ([][]string, error) {
	book, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	defer book.Close()
	if index, err := book.GetSheetIndex(sheetName); err != nil || index < 0 {
		return nil, fmt.Errorf("工作表不存在：%s", sheetName)
	}
	iter, err := book.Rows(sheetName)
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	rows := make([][]string, 0)
	for iter.Next() {
		row, err := iter.Columns(excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return rows, nil
}

func ListTikTokTaxDetails(c *gin.Context) {
	page, size := taxPage(c)
	var total int64
	query := filterTikTokTaxStore(database.DB.Model(&model.TikTokTaxDetail{}), c.Query("month"), c.Query("quarter"), c.Query("store_id"), c.Query("site_id"))
	if err := query.Count(&total).Error; err != nil {
		taxError(c, err)
		return
	}
	var rows []model.TikTokTaxDetail
	detailsQuery := queryTikTokDetails(database.DB, c.Query("month"), c.Query("quarter"))
	if c.Query("store_id") != "" {
		detailsQuery = detailsQuery.Where("(d.store_id = ? OR d.store_id = '')", c.Query("store_id"))
	}
	detailsQuery = filterTaxSite(detailsQuery, c.Query("site_id"))
	if err := detailsQuery.Order("d.posted_at, d.id").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		taxError(c, err)
		return
	}
	response := gin.H{"items": rows, "total": total}
	if wantsTaxTotals(c.Query("include_totals")) {
		var allRows []model.TikTokTaxDetail
		allDetailsQuery := queryTikTokDetails(database.DB, c.Query("month"), c.Query("quarter"))
		if c.Query("store_id") != "" {
			allDetailsQuery = allDetailsQuery.Where("(d.store_id = ? OR d.store_id = '')", c.Query("store_id"))
		}
		allDetailsQuery = filterTaxSite(allDetailsQuery, c.Query("site_id"))
		if err := allDetailsQuery.Find(&allRows).Error; err != nil {
			taxError(c, err)
			return
		}
		response["totals"] = decimalStructTotals(allRows)
	}
	c.JSON(http.StatusOK, response)
}
func ListTikTokTaxQuarters(c *gin.Context) {
	currency := strings.ToUpper(c.DefaultQuery("currency", "CNY"))
	if currency != "CNY" && currency != "USD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持人民币或美元"})
		return
	}
	rows, err := listTikTokQuarterSummary(database.DB.WithContext(c.Request.Context()), currency, c.Query("store_id"), c.Query("site_id"))
	if err != nil {
		taxError(c, err)
		return
	}
	dateFrom, dateTo := c.Query("date_from"), c.Query("date_to")
	if (dateFrom != "" && !validTaxDate(dateFrom)) || (dateTo != "" && !validTaxDate(dateTo)) || (dateFrom != "" && dateTo != "" && dateFrom > dateTo) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "日期范围无效"})
		return
	}
	if dateFrom != "" || dateTo != "" {
		filtered := rows[:0]
		for _, row := range rows {
			start, end := quarterDateRange(row.Quarter)
			if (dateFrom == "" || end >= dateFrom) && (dateTo == "" || start <= dateTo) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	c.JSON(http.StatusOK, gin.H{"items": rows})
}

type tikTokQuarterSummary struct {
	Quarter                                     string           `json:"quarter"`
	SalesVolume                                 *int64           `json:"sales_volume"`
	SampleCount                                 *int64           `json:"sample_count"`
	TransactionCount                            *int64           `json:"transaction_count"`
	AffiliateCommission                         *decimal.Decimal `json:"affiliate_commission"`
	SettlementAffiliateCommission               *decimal.Decimal `json:"settlement_affiliate_commission"`
	AffiliateDirect                             *decimal.Decimal `json:"affiliate_direct"`
	AffiliatePartner                            *decimal.Decimal `json:"affiliate_partner"`
	AffiliateShopAds                            *decimal.Decimal `json:"affiliate_shop_ads"`
	AffiliateDeposit                            *decimal.Decimal `json:"affiliate_deposit"`
	AffiliateRefund                             *decimal.Decimal `json:"affiliate_refund"`
	AffiliatePartnerShopAds                     *decimal.Decimal `json:"affiliate_partner_shop_ads"`
	OtherFees                                   *decimal.Decimal `json:"other_fees"`
	SettlementOtherFees                         *decimal.Decimal `json:"settlement_other_fees"`
	SettlementOtherFeesSignatureConfirmation    *decimal.Decimal `json:"settlement_other_fees_signature_confirmation"`
	SettlementOtherFeesCofundedPromotion        *decimal.Decimal `json:"settlement_other_fees_cofunded_promotion"`
	SettlementOtherFeesShippingProtection       *decimal.Decimal `json:"settlement_other_fees_shipping_protection"`
	SettlementOtherFeesNegativeBalanceDeduction *decimal.Decimal `json:"settlement_other_fees_negative_balance_deduction"`
	SettlementOtherFeesNegativeBalanceSales     *decimal.Decimal `json:"settlement_other_fees_negative_balance_sales"`
	CampaignServiceFee                          *decimal.Decimal `json:"campaign_service_fee"`
	SettlementCampaignServiceFee                *decimal.Decimal `json:"settlement_campaign_service_fee"`
	SettlementCampaignServiceFeeItem            *decimal.Decimal `json:"settlement_campaign_service_fee_item"`
	SettlementAdditionalCampaignPackage         *decimal.Decimal `json:"settlement_additional_campaign_package"`
	SettlementCampaignPackage                   *decimal.Decimal `json:"settlement_campaign_package"`
	SmartPromotionFee                           *decimal.Decimal `json:"smart_promotion_fee"`
	SettlementSmartPromotionFee                 *decimal.Decimal `json:"settlement_smart_promotion_fee"`
	SettlementSmartPromotion                    *decimal.Decimal `json:"settlement_smart_promotion"`
	SettlementSmartPromotionPeriod              *decimal.Decimal `json:"settlement_smart_promotion_period"`
	PromotionFee                                *decimal.Decimal `json:"promotion_fee"`
	SettlementPromotionFee                      *decimal.Decimal `json:"settlement_promotion_fee"`
	Tax                                         *decimal.Decimal `json:"tax"`
	SettlementTax                               *decimal.Decimal `json:"settlement_tax"`
	SettlementTaxSales                          *decimal.Decimal `json:"settlement_tax_sales"`
	SettlementTaxSalesPayment                   *decimal.Decimal `json:"settlement_tax_sales_payment"`
	SettlementTaxSalesRefund                    *decimal.Decimal `json:"settlement_tax_sales_refund"`
	SettlementTaxRetailDelivery                 *decimal.Decimal `json:"settlement_tax_retail_delivery"`
	SettlementTaxRetailPayment                  *decimal.Decimal `json:"settlement_tax_retail_payment"`
	SettlementTaxRetailRefund                   *decimal.Decimal `json:"settlement_tax_retail_refund"`
	Income                                      *decimal.Decimal `json:"income"`
	SalesAmount                                 *decimal.Decimal `json:"sales_amount"`
	Refund                                      *decimal.Decimal `json:"refund"`
	SettlementRefundSubtotal                    *decimal.Decimal `json:"settlement_refund_subtotal"`
	Commission                                  *decimal.Decimal `json:"commission"`
	SettlementCommission                        *decimal.Decimal `json:"settlement_commission"`
	SettlementCommissionRefund                  *decimal.Decimal `json:"settlement_commission_refund"`
	SettlementCommissionTotal                   *decimal.Decimal `json:"settlement_commission_total"`
	SettlementCommissionReferral                *decimal.Decimal `json:"settlement_commission_referral"`
	SettlementCommissionShipping                *decimal.Decimal `json:"settlement_commission_shipping"`
	SettlementCommissionCampaign                *decimal.Decimal `json:"settlement_commission_campaign"`
	SettlementCommissionReturnShipping          *decimal.Decimal `json:"settlement_commission_return_shipping"`
	SettlementCommissionFBTShippingOffset       *decimal.Decimal `json:"settlement_commission_fbt_shipping_offset"`
	SettlementCommissionFBTReimbursement        *decimal.Decimal `json:"settlement_commission_fbt_reimbursement"`
	ShippingFee                                 *decimal.Decimal `json:"shipping_fee"`
	SettlementShippingFee                       *decimal.Decimal `json:"settlement_shipping_fee"`
	ShippingCustomerOffset                      *decimal.Decimal `json:"shipping_customer_offset"`
	ShippingPlatformDiscount                    *decimal.Decimal `json:"shipping_platform_discount"`
	ShippingCustomer                            *decimal.Decimal `json:"shipping_customer"`
	ShippingRefundedCustomer                    *decimal.Decimal `json:"shipping_refunded_customer"`
	OtherIncomeFBTReimbursement                 *decimal.Decimal `json:"other_income_fbt_reimbursement"`
	OtherIncomeReturnShipping                   *decimal.Decimal `json:"other_income_return_shipping"`
	OtherIncomeShippingProtection               *decimal.Decimal `json:"other_income_shipping_protection"`
	OtherIncomeFBTSubsidy                       *decimal.Decimal `json:"other_income_fbt_subsidy"`
	OtherIncomeSettlement                       *decimal.Decimal `json:"other_income_settlement"`
	Delivery                                    *decimal.Decimal `json:"delivery"`
	DeliveryMCF                                 *decimal.Decimal `json:"delivery_mcf"`
	DeliveryFBT                                 *decimal.Decimal `json:"delivery_fbt"`
	DeliveryOverseas                            *decimal.Decimal `json:"delivery_overseas"`
	DeliveryUnmatchedRows                       int64            `json:"delivery_unmatched_rows"`
	OtherIncome                                 *decimal.Decimal `json:"other_income"`
	SettlementOtherIncome                       *decimal.Decimal `json:"settlement_other_income"`
	HubFee                                      *decimal.Decimal `json:"hub_fee"`
	StorageFee                                  *decimal.Decimal `json:"storage_fee"`
	FBTFee                                      *decimal.Decimal `json:"fbt_fee"`
	SettlementFBTFee                            *decimal.Decimal `json:"settlement_fbt_fee"`
	FBTOther                                    *decimal.Decimal `json:"fbt_other"`
	SettlementFBTOther                          *decimal.Decimal `json:"settlement_fbt_other"`
	Advertising                                 *decimal.Decimal `json:"advertising"`
	TaxableIncome                               *decimal.Decimal `json:"taxable_income"`
	MissingRates                                int64            `json:"missing_rates"`
}

func listTikTokQuarterSummary(db *gorm.DB, currency string, stores ...string) ([]tikTokQuarterSummary, error) {
	store := ""
	if len(stores) > 0 {
		store = stores[0]
	}
	// Keep the raw site value: an empty site means "all countries".
	site := ""
	if len(stores) > 1 {
		site = strings.TrimSpace(stores[1])
	}
	// Aggregate each source independently so joins cannot multiply the financial amounts.
	sourceCurrency := "COALESCE(NULLIF(UPPER(TRIM(r.data->>'币种')), ''), 'USD')"
	factor := "CASE WHEN " + sourceCurrency + " = 'CNY' THEN 1 ELSE CASE WHEN EXTRACT(ISODOW FROM d.posted_at) IN (6,7) THEN NULLIF(fx.rate,0) ELSE COALESCE(NULLIF(fx.rate, 0), NULLIF(d.rate, 0)) END END"
	// Older imports stored zero when the source file used the newer translated
	// commission header. Read that raw value as a compatibility fallback while
	// keeping the tax detail commission as the authoritative total.
	commission := "COALESCE(NULLIF(d.commission, 0), NULLIF(REGEXP_REPLACE(COALESCE(cr.value, ''), '[^0-9.\\-]', '', 'g'), '')::numeric, 0)"
	adFactor := "CASE WHEN EXTRACT(ISODOW FROM a.date::date) IN (6,7) THEN NULLIF(afx.rate,0) ELSE COALESCE(NULLIF(afx.rate, 0), NULLIF(a.rate, 0)) END"
	if currency == "USD" {
		factor = "CASE WHEN " + sourceCurrency + " = 'USD' THEN 1 ELSE (" + factor + ") / NULLIF(usd.rate, 0) END"
		adFactor = "1"
	}
	detail := db.Table("tik_tok_tax_details d").
		Joins("LEFT JOIN tik_tok_tax_raw_rows r ON r.id = d.raw_row_id").
		Joins(`LEFT JOIN LATERAL (
  SELECT value
  FROM jsonb_each_text(r.data) AS raw(key, value)
  WHERE replace(replace(replace(replace(replace(BTRIM(raw.key), ' ', ''), chr(160), ''), '（', '('), '）', ')'), chr(65279), '') IN
    ('支付给平台的佣金/服务费金额', '支付给平台的佣金/服务费金额(美元)', '平台佣金/服务费金额', '平台佣金/服务费金额(美元)', '佣金/服务费金额', '佣金/服务费金额(美元)')
    OR (raw.key LIKE '%佣金%' AND raw.key LIKE '%服务费%' AND raw.key LIKE '%金额%')
    OR (LOWER(raw.key) LIKE '%commission%' AND LOWER(raw.key) LIKE '%service%')
  LIMIT 1
) cr ON TRUE`).
		Joins(taxRateJoinSQL("fx", "d.posted_at", sourceCurrency)).
		Joins(taxRateJoinSQL("usd", "d.posted_at", "'USD'")).
		Select(fmt.Sprintf("d.quarter, d.income * (%[1]s) AS income, d.refund * (%[1]s) AS refund, (%[2]s) * (%[1]s) AS commission, 0::numeric AS delivery, 0::numeric AS other_income, 0::numeric AS hub_fee, 0::numeric AS storage_fee, 0::numeric AS fbt_fee, 0::numeric AS fbt_other, 0::numeric AS advertising, CASE WHEN (%[1]s) IS NULL THEN 1 ELSE 0 END AS missing_detail_rates, 0 AS missing_ad_rates, 0 AS missing_fbt_rates", factor, commission))
	if store != "" {
		detail = detail.Where("(d.store_id = ? OR d.store_id = '')", store)
	}
	detail = filterTaxSite(detail, site)
	ads := db.Table("tik_tok_tax_ads a").
		Joins(taxRateJoinSQL("afx", "a.date", "'USD'")).
		Select(fmt.Sprintf("a.quarter, 0::numeric AS income, 0::numeric AS refund, 0::numeric AS commission, 0::numeric AS delivery, 0::numeric AS other_income, 0::numeric AS hub_fee, 0::numeric AS storage_fee, 0::numeric AS fbt_fee, 0::numeric AS fbt_other, a.credit_spend * (%[1]s) AS advertising, 0 AS missing_detail_rates, CASE WHEN (%[1]s) IS NULL THEN 1 ELSE 0 END AS missing_ad_rates, 0 AS missing_fbt_rates", adFactor))
	if store != "" {
		ads = ads.Where("(a.store_id = ? OR a.store_id = '')", store)
	}
	ads = filterTaxSite(ads, site)
	fbtFactor := "NULLIF(fx.rate, 0)"
	if currency == "USD" {
		fbtFactor = "1"
	}
	fbtOther := "f.inbound_shipping_fee + f.inbound_incidents_fee + f.inbound_booking_non_compliance_fee + f.routing_non_compliance_fee + f.outbound_appointment_no_show_fee + f.delayed_response_fee + f.disposal_fee + f.return_to_seller_shipping_fee + f.return_to_seller_handling_fee + f.inbound_return_operation_fee"
	fbt := db.Table("tik_tok_tax_fbts f").Joins("LEFT JOIN LATERAL (SELECT rate FROM tax_exchange_rates WHERE currency = 'USD' AND date >= f.month || '-01' AND date < to_char((to_date(f.month || '-01', 'YYYY-MM-DD') + interval '1 month') , 'YYYY-MM-DD') ORDER BY date DESC LIMIT 1) fx ON TRUE").Select(fmt.Sprintf("f.quarter, 0::numeric AS income, 0::numeric AS refund, 0::numeric AS commission, f.fbt_fulfillment_fee * (%[1]s) AS delivery, f.fbt_fulfillment_fee_reimbursement * (%[1]s) AS other_income, f.hub_placement_fee * (%[1]s) AS hub_fee, f.storage_fee * (%[1]s) AS storage_fee, -(f.total_logistics_cost - f.fbt_fulfillment_fee - f.fbt_fulfillment_fee_reimbursement) * (%[1]s) AS fbt_fee, (%[2]s) * (%[1]s) AS fbt_other, 0::numeric AS advertising, 0 AS missing_detail_rates, 0 AS missing_ad_rates, CASE WHEN (%[1]s) IS NULL THEN 1 ELSE 0 END AS missing_fbt_rates", fbtFactor, fbtOther))
	if store != "" {
		fbt = fbt.Where("(f.store_id = ? OR f.store_id = '')", store)
	}
	fbt = filterTaxSite(fbt, site)
	rows := []tikTokQuarterSummary{}
	base := db.Raw(`SELECT quarter,
  CASE WHEN SUM(missing_detail_rates) = 0 THEN ROUND(SUM(income), 2) END AS income,
  CASE WHEN SUM(missing_detail_rates) = 0 THEN ROUND(SUM(refund), 2) END AS refund,
  CASE WHEN SUM(missing_detail_rates) = 0 THEN ROUND(SUM(commission), 2) END AS commission,
  CASE WHEN SUM(missing_fbt_rates) = 0 THEN ROUND(SUM(delivery), 2) END AS delivery,
  CASE WHEN SUM(missing_fbt_rates) = 0 THEN ROUND(SUM(other_income), 2) END AS other_income,
  CASE WHEN SUM(missing_fbt_rates) = 0 THEN ROUND(SUM(hub_fee), 2) END AS hub_fee,
  CASE WHEN SUM(missing_fbt_rates) = 0 THEN ROUND(SUM(storage_fee), 2) END AS storage_fee,
  CASE WHEN SUM(missing_fbt_rates) = 0 THEN ROUND(SUM(fbt_fee), 2) END AS fbt_fee,
  CASE WHEN SUM(missing_fbt_rates) = 0 THEN ROUND(SUM(fbt_other), 2) END AS fbt_other,
  CASE WHEN SUM(missing_ad_rates) = 0 THEN ROUND(SUM(advertising), 2) END AS advertising,
  CASE WHEN SUM(missing_detail_rates) = 0 THEN SUM(refund) END AS refund_unrounded,
  SUM(missing_detail_rates + missing_ad_rates + missing_fbt_rates) AS missing_rates
  FROM ((?) UNION ALL (?) UNION ALL (?)) AS entries GROUP BY quarter`, detail, ads, fbt)
	delivery := tikTokDeliverySummary(db, currency, store, site)
	settlementFBTFactor := "CASE WHEN s.currency = 'CNY' THEN 1 ELSE NULLIF(fx.rate, 0) END"
	if currency == "USD" {
		settlementFBTFactor = "CASE WHEN s.currency = 'USD' THEN 1 ELSE (" + settlementFBTFactor + ") / NULLIF(usd.rate, 0) END"
	}
	settlementFBT := db.Table("tik_tok_settlements s").
		Joins(taxRateJoinSQL("fx", "s.statement_date", "s.currency")).
		Joins(taxRateJoinSQL("usd", "s.statement_date", "'USD'"))
	settlementFBT = settlementFBT.Select(fmt.Sprintf("s.quarter, SUM(CASE WHEN LOWER(BTRIM(COALESCE(s.data->>'Type', ''))) = 'fbt warehouse service fee using gmv payment' THEN COALESCE(NULLIF(regexp_replace(regexp_replace(COALESCE(s.data->>'Total settlement amount', ''), '[$,\\s]', '', 'g'), '^\\((.*)\\)$', '-\\1'), '')::numeric, 0) * (%s) ELSE 0 END) AS settlement_fbt_fee", settlementFBTFactor)).Group("s.quarter")
	if store != "" {
		settlementFBT = settlementFBT.Where("(s.store_id = ? OR s.store_id = '')", store)
	}
	settlementFBT = filterTaxSite(settlementFBT, site)
	volume := tikTokSalesVolumeSummary(db)
	if store != "" {
		volume = tikTokSalesVolumeSummary(db, store, site)
	} else if site != "" {
		volume = tikTokSalesVolumeSummary(db, "", site)
	}
	transactionSummary := db.Table("tik_tok_tax_transaction_summaries").Select("quarter, transaction_count")
	if !db.Migrator().HasTable(&model.TikTokTaxTransactionSummary{}) {
		transactionSummary = db.Raw("SELECT ''::text AS quarter, 0::bigint AS transaction_count WHERE FALSE")
	}
	if store != "" {
		transactionSummary = transactionSummary.Where("(store_id = ? OR store_id = '' OR store_id IS NULL)", store)
	}
	transactionSummary = filterTaxSite(transactionSummary, site)
	err := db.Raw(`SELECT COALESCE(b.quarter, d.quarter, v.quarter, t.quarter, sf.quarter) AS quarter,
 COALESCE(v.sales_volume, 0) AS sales_volume,
 COALESCE(v.sample_count, 0) AS sample_count,
 CASE WHEN t.quarter IS NULL THEN NULL ELSE t.transaction_count END AS transaction_count,
 0::numeric AS affiliate_commission,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.affiliate_commission END AS settlement_affiliate_commission,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.affiliate_direct END AS affiliate_direct,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.affiliate_partner END AS affiliate_partner,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.affiliate_shop_ads END AS affiliate_shop_ads,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.affiliate_deposit END AS affiliate_deposit,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.affiliate_refund END AS affiliate_refund,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.affiliate_partner_shop_ads END AS affiliate_partner_shop_ads,
 0::numeric AS other_fees,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.other_fees END AS settlement_other_fees,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.other_fee_signature END AS settlement_other_fees_signature_confirmation,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.other_fee_cofunded END AS settlement_other_fees_cofunded_promotion,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.other_fee_protection END AS settlement_other_fees_shipping_protection,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.other_fee_negative_deduction END AS settlement_other_fees_negative_balance_deduction,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.other_fee_negative_sales END AS settlement_other_fees_negative_balance_sales,
 0::numeric AS campaign_service_fee,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.campaign_service_fee END AS settlement_campaign_service_fee,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.campaign_service_item END AS settlement_campaign_service_fee_item,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.additional_campaign_package END AS settlement_additional_campaign_package,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.campaign_package END AS settlement_campaign_package,
	 0::numeric AS smart_promotion_fee,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.smart_promotion_fee END AS settlement_smart_promotion_fee,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.smart_promotion_item END AS settlement_smart_promotion,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.smart_promotion_period END AS settlement_smart_promotion_period,
 0::numeric AS promotion_fee,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.promotion_fee END AS settlement_promotion_fee,
 0::numeric AS tax,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.tax END AS settlement_tax,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.tax_sales END AS settlement_tax_sales,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.tax_sales_payment END AS settlement_tax_sales_payment,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.tax_sales_refund END AS settlement_tax_sales_refund,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.tax_retail_delivery END AS settlement_tax_retail_delivery,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.tax_retail_payment END AS settlement_tax_retail_payment,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.tax_retail_refund END AS settlement_tax_retail_refund,
 CASE WHEN b.quarter IS NULL THEN 0 ELSE b.income END AS income,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.sales_amount END AS sales_amount,
 CASE WHEN b.quarter IS NULL THEN 0 ELSE b.refund END AS refund,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_refund_subtotal END AS settlement_refund_subtotal,
 CASE WHEN b.quarter IS NULL THEN 0 ELSE b.commission END AS commission,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_commission END AS settlement_commission,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_commission_refund END AS settlement_commission_refund,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_commission_total END AS settlement_commission_total,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_commission_referral END AS settlement_commission_referral,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_commission_shipping END AS settlement_commission_shipping,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_commission_campaign END AS settlement_commission_campaign,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_commission_return_shipping END AS settlement_commission_return_shipping,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_commission_fbt_shipping_offset END AS settlement_commission_fbt_shipping_offset,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_commission_fbt_reimbursement END AS settlement_commission_fbt_reimbursement,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.shipping_fee END AS shipping_fee,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_shipping_fee END AS settlement_shipping_fee,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.shipping_customer_offset END AS shipping_customer_offset,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.shipping_platform_discount END AS shipping_platform_discount,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.shipping_customer END AS shipping_customer,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.shipping_refunded_customer END AS shipping_refunded_customer,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.other_income_fbt_reimbursement END AS other_income_fbt_reimbursement,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.other_income_return_shipping END AS other_income_return_shipping,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.other_income_shipping_protection END AS other_income_shipping_protection,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.other_income_fbt_subsidy END AS other_income_fbt_subsidy,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.other_income_settlement END AS other_income_settlement,
 CASE WHEN b.quarter IS NULL THEN 0 ELSE b.delivery END AS delivery,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.delivery_mcf END AS delivery_mcf,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.delivery_fbt END AS delivery_fbt,
 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.delivery_overseas END AS delivery_overseas,
 COALESCE(d.delivery_unmatched_rows, 0) AS delivery_unmatched_rows,
 CASE WHEN b.quarter IS NULL THEN 0 ELSE b.other_income END AS other_income,

 CASE WHEN d.quarter IS NULL THEN 0 ELSE d.settlement_other_income END AS settlement_other_income,
 CASE WHEN b.quarter IS NULL THEN 0 ELSE b.hub_fee END AS hub_fee,
 CASE WHEN b.quarter IS NULL THEN 0 ELSE b.storage_fee END AS storage_fee,
 CASE WHEN b.quarter IS NULL THEN 0 ELSE b.fbt_fee END AS fbt_fee,
 CASE WHEN sf.quarter IS NULL THEN 0 ELSE sf.settlement_fbt_fee END AS settlement_fbt_fee,
 CASE WHEN b.quarter IS NULL THEN 0 ELSE b.fbt_other END AS fbt_other,
 CASE WHEN b.quarter IS NULL THEN 0 ELSE b.fbt_other END AS settlement_fbt_other,
 CASE WHEN b.quarter IS NULL THEN 0 ELSE b.advertising END AS advertising,
 ROUND((CASE WHEN d.quarter IS NULL THEN 0 ELSE d.sales_unrounded END)
       - (CASE WHEN b.quarter IS NULL THEN 0 ELSE b.refund_unrounded END), 2) AS taxable_income,
 COALESCE(b.missing_rates, 0) + COALESCE(d.missing_rates, 0) AS missing_rates
 FROM (?) b FULL OUTER JOIN (?) d ON d.quarter = b.quarter
 FULL OUTER JOIN (?) v ON v.quarter = COALESCE(b.quarter, d.quarter)
	 FULL OUTER JOIN (?) t ON t.quarter = COALESCE(b.quarter, d.quarter, v.quarter)
	 FULL OUTER JOIN (?) sf ON sf.quarter = COALESCE(b.quarter, d.quarter, v.quarter, t.quarter)
	 ORDER BY COALESCE(b.quarter, d.quarter, v.quarter, t.quarter, sf.quarter)`, base, delivery, volume, transactionSummary, settlementFBT).Scan(&rows).Error
	return rows, err
}

// tikTokSalesVolumeSummary counts each order/SKU pair once. Settlement files can
// contain several rows for the same pair. Normal orders count as sales; matched
// orders with an empty order type count as samples.
func tikTokSalesVolumeSummary(db *gorm.DB, stores ...string) *gorm.DB {
	store := ""
	if len(stores) > 0 {
		store = stores[0]
	}
	// Keep the raw site value: an empty site means "all countries".
	site := ""
	if len(stores) > 1 {
		site = strings.TrimSpace(stores[1])
	}
	// Deduplicate settlements before looking up order/SKU quantities.
	query := `WITH candidates AS (
  SELECT s.id, s.quarter,
         NULLIF(BTRIM(s.data->>'Order/adjustment ID'), '') AS order_id,
         COALESCE(NULLIF(BTRIM(s.data->>'SKU ID'), ''), '__blank__') AS sku_id,
         NULLIF(BTRIM(s.data->>'SKU ID'), '') AS match_sku
  FROM tik_tok_settlements s
  WHERE BTRIM(COALESCE(s.data->>'Type', '')) = 'Order'
    AND NULLIF(BTRIM(s.data->>'Order/adjustment ID'), '') IS NOT NULL
), deduped AS (
  SELECT DISTINCT ON (order_id, sku_id) * FROM candidates ORDER BY order_id, sku_id, id
), matched AS (
  SELECT s.quarter, o.data AS order_data
  FROM deduped s
  LEFT JOIN LATERAL (
    SELECT data FROM tik_tok_orders o
    WHERE o.order_id = s.order_id
      AND (s.match_sku IS NULL OR BTRIM(o.data->>'SKU ID') = s.match_sku)
    ORDER BY o.id DESC LIMIT 1
  ) o ON TRUE
)
SELECT quarter,
       COALESCE(SUM(CASE
         WHEN BTRIM(COALESCE(order_data->>'Normal or Pre-order', '')) = 'Normal'
         THEN COALESCE(NULLIF(REGEXP_REPLACE(COALESCE(order_data->>'Quantity', ''), '[^0-9.\-]', '', 'g'), '')::numeric, 0)
         ELSE 0
       END), 0)::bigint AS sales_volume,
       COALESCE(SUM(CASE
         WHEN order_data IS NOT NULL AND BTRIM(COALESCE(order_data->>'Normal or Pre-order', '')) = ''
         THEN COALESCE(NULLIF(REGEXP_REPLACE(COALESCE(order_data->>'Quantity', ''), '[^0-9.\-]', '', 'g'), '')::numeric, 0)
         ELSE 0
       END), 0)::bigint AS sample_count
FROM matched
GROUP BY quarter`
	if store != "" {
		escaped := strings.ReplaceAll(store, "'", "''")
		query = strings.Replace(query, "FROM tik_tok_settlements s\n  WHERE", "FROM tik_tok_settlements s\n  WHERE (s.store_id = '"+escaped+"' OR s.store_id = '') AND", 1)
		query = strings.Replace(query, "SELECT data FROM tik_tok_orders o\n    WHERE o.order_id", "SELECT data FROM tik_tok_orders o\n    WHERE (o.store_id = '"+escaped+"' OR o.store_id = '') AND o.order_id", 1)
	}
	if site != "" {
		escaped := strings.ReplaceAll(site, "'", "''")
		query = strings.Replace(query, "FROM tik_tok_settlements s\n  WHERE", "FROM tik_tok_settlements s\n  WHERE (s.site_id = '"+escaped+"' OR s.site_id = '' OR s.site_id IS NULL) AND", 1)
		query = strings.Replace(query, "SELECT data FROM tik_tok_orders o\n    WHERE o.order_id", "SELECT data FROM tik_tok_orders o\n    WHERE (o.site_id = '"+escaped+"' OR o.site_id = '' OR o.site_id IS NULL) AND o.order_id", 1)
	}
	return db.Raw(query)
}
