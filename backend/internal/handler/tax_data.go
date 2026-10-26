package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/extrame/xls"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const taxMaxFile = 20 << 20

// XLSX is a ZIP container; its expanded XML can be much larger than the uploaded file.
const taxMaxXLSXUnzip = 512 << 20
const taxMaxXLSXXML = 128 << 20

var exchangeCurrencies = []string{"USD", "EUR", "CAD", "GBP", "MXN", "CHF", "PLN"}

var taxCategories = []string{"仓储费", "广告费", "配送费", "其他收入", "税", "退款", "销售额", "佣金", "杂费", "折扣"}
var taxHeaders = strings.Split("Period Start Date|Period End Date|Total Payable|Currency|Transaction Key|Transaction Posted Timestamp|Transaction Type|Transaction Description|Customer Order #|Customer Order line #|Purchase Order #|Purchase Order line #|Amount|Amount Type|Ship Qty|Commission Rate|Base Commission Rate|Transaction Reason Description|Partner Item Id|Partner GTIN|Partner Item Name|Product Tax Code|Ship to State|Ship to City|Ship to Zipcode|Contract Category|Product Type|Commission Rule|Shipping Method|Fulfillment Type|Fulfillment Details|Original Commission|Commission Incentive Program|Commission Saving|Customer Promo Type|Total Walmart Funded Savings Program|Campaign Id|Item Condition|Original charge|Charge Savings|Incentive Program Name|Ship to country", "|")

func taxUserAllowed(user model.User) bool {
	for _, role := range user.Roles {
		if role.Name == "admin" {
			return true
		}
	}
	return strings.Contains(user.Department, "财务") || strings.Contains(user.SubDepartment, "财务")
}
func RequireTaxDataAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := c.Get("userID")
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "请登录"})
			return
		}
		var user model.User
		if err := database.DB.Preload("Roles").First(&user, id).Error; err != nil {
			c.AbortWithStatusJSON(403, gin.H{"error": "无法验证用户权限"})
			return
		}
		if !taxUserAllowed(user) {
			c.AbortWithStatusJSON(403, gin.H{"error": "仅管理员或财务部门可访问"})
			return
		}
		if !financePermissionAllowed(user, c) {
			c.AbortWithStatusJSON(403, gin.H{"error": "当前账号没有该财务平台操作权限"})
			return
		}
		c.Next()
	}
}

func financePermissionAllowed(user model.User, c *gin.Context) bool {
	for _, role := range user.Roles {
		if role.Name == "admin" {
			return true
		}
	}
	// Existing finance accounts retain access until an administrator saves an explicit matrix.
	if len(user.FinancePermissions) == 0 {
		return true
	}
	path := strings.ToLower(c.Request.URL.Path)
	platform := strings.ToLower(c.Query("platform"))
	if platform == "" {
		platform = strings.ToLower(c.PostForm("platform"))
	}
	switch {
	case strings.Contains(path, "/amazon"):
		platform = "amazon"
	case strings.Contains(path, "/tiktok"):
		platform = "tiktok"
	case strings.Contains(path, "/customs-declarations"):
		platform = "customs"
	case strings.Contains(path, "/exchange-rates"):
		platform = "exchange_rates"
	case platform == "walmart":
		platform = "walmart"
	}
	action := "view"
	if strings.Contains(path, "/export") {
		action = "export"
	} else if c.Request.Method == http.MethodPost {
		if strings.Contains(path, "/upload") || strings.Contains(path, "/imports") || strings.Contains(path, "/customs-declarations") {
			action = "import"
		} else {
			action = "create"
		}
	} else if c.Request.Method == http.MethodPut || c.Request.Method == http.MethodDelete {
		action = "create"
	}
	if platform == "" {
		for _, permissions := range user.FinancePermissions {
			for _, allowed := range permissions {
				if allowed == action {
					return true
				}
			}
		}
		return false
	}
	for _, allowed := range user.FinancePermissions[platform] {
		if allowed == action {
			return true
		}
	}
	return false
}

func RequireExchangeRateAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := c.Get("userID")
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "请登录"})
			return
		}
		var user model.User
		if err := database.DB.Preload("Roles").First(&user, id).Error; err != nil {
			c.AbortWithStatusJSON(403, gin.H{"error": "无法验证用户权限"})
			return
		}
		isAdmin := false
		for _, role := range user.Roles {
			if role.Name == "admin" {
				isAdmin = true
				break
			}
		}
		isCustomerService := func(value string) bool {
			value = strings.TrimSuffix(strings.TrimSpace(value), "部")
			return value == "客服" || value == "海外客服"
		}
		if !isAdmin && (isCustomerService(user.Department) || isCustomerService(user.SubDepartment)) {
			c.AbortWithStatusJSON(403, gin.H{"error": "客服人员仅可访问每日来电登记"})
			return
		}
		if !financePermissionAllowed(user, c) {
			c.AbortWithStatusJSON(403, gin.H{"error": "当前账号没有汇率平台操作权限"})
			return
		}
		c.Next()
	}
}

// RequireExchangeRateRefresh limits the potentially slow, external-API backed
// overwrite operation to the dedicated administrator account.
func RequireExchangeRateRefresh() gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := c.Get("userID")
		var user model.User
		if !ok || database.DB.Select("id, name").First(&user, id).Error != nil || strings.TrimSpace(user.Name) != "管理员" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "只有管理员账号可以刷新汇率"})
			return
		}
		c.Next()
	}
}

const taxDataRPAKey = "hem_rpa_7dfd3f2bfe3ebd0711b626f8be9b8d20b8d663c7cd6c9669"

func taxUploadAuthorized(c *gin.Context) bool {
	if c.GetHeader("X-RPA-Key") == taxDataRPAKey {
		return true
	}
	_, ok := c.Get("userID")
	return ok
}

// taxImporterLabel names who performed an upload: the logged-in user for page
// imports, or "RPA" for key-authenticated service imports.
func taxImporterLabel(c *gin.Context) string {
	if c.GetHeader("X-RPA-Key") == taxDataRPAKey {
		return "RPA"
	}
	id, ok := c.Get("userID")
	if !ok {
		return "RPA"
	}
	var user model.User
	if err := database.DB.Select("name, username").First(&user, id).Error; err != nil {
		return "RPA"
	}
	if name := strings.TrimSpace(user.Name); name != "" {
		return name
	}
	if username := strings.TrimSpace(user.Username); username != "" {
		return username
	}
	return "RPA"
}

// UploadTaxDataFromRPA commits raw rows, processed rows and quarter snapshots together.
func UploadTaxDataFromRPA(c *gin.Context) {
	respond := func(code int, ok bool, message string) { c.JSON(code, gin.H{"status": ok, "msg": message}) }
	if !taxUploadAuthorized(c) {
		respond(401, false, "RPA 密钥无效")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, taxMaxFile+(1<<20))
	if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
		respond(400, false, "需要 multipart/form-data，文件最大 20MB")
		return
	}
	defer c.Request.MultipartForm.RemoveAll()
	// Older callers use the same default store as the page.
	store := strings.TrimSpace(c.PostForm("store_id"))
	if store == "" || store == "walmart" || store == "walmart-shop-001" {
		store = "TVOFEK"
	}
	if len(store) > 100 {
		respond(400, false, "store_id 最长 100 字节")
		return
	}
	site := taxSiteFromForm(c)
	fh, err := c.FormFile("file")
	if err != nil || fh.Size > taxMaxFile {
		respond(400, false, "file 必填且不得超过 20MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".csv" && ext != ".xlsx" && ext != ".xls" {
		respond(415, false, "仅支持 CSV、XLSX 或 XLS 文件")
		return
	}
	f, err := fh.Open()
	if err != nil {
		respond(500, false, "文件读取失败")
		return
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, taxMaxFile+1))
	if err != nil {
		respond(500, false, "文件读取失败")
		return
	}
	if len(content) > taxMaxFile {
		respond(413, false, "文件超过 20MB")
		return
	}
	importer := taxImporterLabel(c)
	if importer == "RPA" {
		if reported := strings.TrimSpace(c.PostForm("importer")); reported != "" {
			importer = "RPA 上报：" + reported
		}
	}
	if len([]rune(importer)) > 150 {
		respond(400, false, "importer 最长 140 字符")
		return
	}
	audit := model.TaxImportLog{StoreID: store, SiteID: site, Importer: importer, FileName: filepath.Base(fh.Filename), FileContent: content, Status: "processing"}
	if err := database.DB.Create(&audit).Error; err != nil {
		taxError(c, err)
		return
	}
	fail := func(code int, message string) {
		if err := database.DB.Model(&audit).Updates(map[string]interface{}{"status": "failed", "message": message, "source_rows": audit.SourceRows}).Error; err != nil {
			taxError(c, err)
			return
		}
		respond(code, false, message)
	}
	raw, err := parseTaxFile(content, filepath.Ext(fh.Filename))
	if err != nil {
		fail(422, err.Error())
		return
	}
	audit.SourceRows = len(raw)
	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	var batch model.TaxDataBatch
	duplicate := false
	err = database.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", store).Error; err != nil {
			return err
		}
		err := filterWalmartStore(tx.Omit("file_content"), store).Where("site_id = ? AND file_hash = ?", site, hash).First(&batch).Error
		if err == nil {
			duplicate = true
			audit.SkippedRows = len(raw)
		} else {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			// Load existing fingerprints directly. This avoids depending on JSON
			// deserialization when checking historical rows and also handles files
			// that contain duplicate identities within the same upload.
			wanted := make([]string, 0, len(raw))
			for _, row := range raw {
				if identity := taxRowIdentity(row.Data); identity != "" {
					wanted = append(wanted, identity)
				}
			}
			seen := map[string]bool{}
			if len(wanted) > 0 {
				var existing []string
				if err := filterWalmartStore(tx.Model(&model.TaxRawRow{}), store).Where("site_id = ? AND fingerprint IN ?", site, wanted).Pluck("fingerprint", &existing).Error; err != nil {
					return err
				}
				for _, identity := range existing {
					seen[identity] = true
				}
			}
			fresh := make([]model.TaxRawRow, 0, len(raw))
			for _, row := range raw {
				identity := taxRowIdentity(row.Data)
				if identity != "" {
					if seen[identity] {
						audit.SkippedRows++
						continue
					}
					seen[identity] = true
					row.Fingerprint = identity
				} else {
					row.Fingerprint = fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s:%d", hash, row.RowNumber))))
				}
				fresh = append(fresh, row)
			}
			batch = model.TaxDataBatch{StoreID: store, SiteID: site, FileHash: hash, FileName: audit.FileName, FileContent: content, RawRows: len(fresh), Rates: map[string]model.TaxRate{}, Status: "locked"}
			var rules []model.TaxFeeRule
			if err := tx.Find(&rules).Error; err != nil {
				return err
			}
			validRules := len(rules) > 0
			for _, rule := range rules {
				if rule.MatchKey == "" {
					validRules = false
					break
				}
			}
			if !validRules {
				rules, err = model.DefaultTaxFeeRules()
				if err != nil {
					return err
				}
			}
			var storedRates []model.TaxExchangeRate
			if err := tx.Where("rate > 0").Find(&storedRates).Error; err != nil {
				return err
			}
			for _, rate := range storedRates {
				batch.Rates[rate.Currency+":"+rate.Date] = model.TaxRate{Rate: rate.Rate, Date: rate.Date, Source: rate.Source, FetchedAt: rate.FetchedAt}
			}
			for _, row := range fresh {
				currency := taxCurrency(row.Data["Currency"])
				if currency == "CNY" {
					batch.Rates[currency] = model.TaxRate{Rate: decimal.NewFromInt(1)}
				} else {
					// Pending imports can be converted once the daily rate is stored.
					batch.Rates[currency] = model.TaxRate{Rate: decimal.Zero}
				}
			}
			processed, quarters, unmatched, err := processTaxRows(fresh, rules, batch.Rates)
			if err != nil {
				return err
			}
			batch.UnmatchedRows = unmatched
			if unmatched > 0 {
				batch.Status = "needs_review"
			}
			if len(fresh) == 0 {
				batch.Status = "duplicate"
				duplicate = true
			}
			if err := tx.Create(&batch).Error; err != nil {
				return err
			}
			if len(fresh) > 0 {
				for i := range fresh {
					fresh[i].BatchID = batch.ID
					fresh[i].StoreID = store
					fresh[i].SiteID = site
				}
				if err := tx.CreateInBatches(&fresh, 200).Error; err != nil {
					return err
				}
				for i := range processed {
					processed[i].BatchID = batch.ID
					processed[i].RawRowID = fresh[i].ID
				}
				if err := tx.CreateInBatches(&processed, 200).Error; err != nil {
					return err
				}
				for i := range quarters {
					quarters[i].BatchID = batch.ID
				}
				if len(quarters) > 0 {
					if err := tx.CreateInBatches(&quarters, 100).Error; err != nil {
						return err
					}
				}
			}
			audit.ImportedRows = len(fresh)
		}
		audit.BatchID = batch.ID
		audit.Status = batch.Status
		if duplicate {
			audit.Status = "duplicate"
		}
		return tx.Model(&audit).Updates(map[string]interface{}{"batch_id": audit.BatchID, "source_rows": audit.SourceRows, "imported_rows": audit.ImportedRows, "skipped_rows": audit.SkippedRows, "status": audit.Status}).Error
	})
	if err != nil {
		if strings.HasPrefix(err.Error(), "汇率获取失败") {
			fail(503, err.Error())
		} else {
			message := fmt.Sprintf("导入失败，本次未入库：%s", err.Error())
			_ = c.Error(err)
			fail(500, message)
		}
		return
	}
	code := 201
	if duplicate {
		code = 200
	}
	respond(code, true, "成功")
}

func taxRowIdentity(data map[string]string) string {
	values := []string{}
	if strings.TrimSpace(data["Transaction Key"]) == "" {
		return ""
	}
	for _, key := range []string{"Transaction Key", "Transaction Type", "Transaction Description", "Amount Type"} {
		values = append(values, strings.TrimSpace(data[key]))
	}
	encoded, _ := json.Marshal(values)
	return fmt.Sprintf("%x", sha256.Sum256(encoded))
}

func parseTaxFile(content []byte, ext string) ([]model.TaxRawRow, error) {
	var rows [][]string
	switch strings.ToLower(ext) {
	case ".csv":
		if !utf8.Valid(content) {
			return nil, fmt.Errorf("CSV 必须使用 UTF-8 编码")
		}
		if !bytes.Contains(content, []byte(",")) {
			return nil, fmt.Errorf("CSV 必须使用逗号分隔")
		}
		r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf})))
		r.FieldsPerRecord = -1
		for {
			row, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			rows = append(rows, row)
			if len(rows) > 100050 {
				return nil, fmt.Errorf("最多支持 100000 行")
			}
		}
	case ".xlsx":
		f, err := excelize.OpenReader(bytes.NewReader(content), excelize.Options{UnzipSizeLimit: taxMaxXLSXUnzip, UnzipXMLSizeLimit: taxMaxXLSXXML})
		if err != nil {
			return nil, err
		}
		defer f.Close()
		iter, err := f.Rows(f.GetSheetName(0))
		if err != nil {
			return nil, err
		}
		defer iter.Close()
		for iter.Next() {
			row, e := iter.Columns()
			if e != nil {
				return nil, e
			}
			rows = append(rows, row)
			if len(rows) > 100050 {
				return nil, fmt.Errorf("最多支持 100000 行")
			}
		}
		if err := iter.Error(); err != nil {
			return nil, err
		}
	case ".xls":
		wb, err := xls.OpenReader(bytes.NewReader(content), "utf-8")
		if err != nil {
			return nil, fmt.Errorf("XLS 文件读取失败: %w", err)
		}
		if wb.NumSheets() == 0 {
			return nil, fmt.Errorf("XLS 文件没有工作表")
		}
		rows = wb.ReadAllCells(100050)
		if len(rows) > 100050 {
			return nil, fmt.Errorf("最多支持 100000 行")
		}
	default:
		return nil, fmt.Errorf("仅支持 UTF-8 CSV、XLSX 或 XLS")
	}
	headerAt := -1
	var headers []string
	for i, row := range rows {
		if i > 50 {
			break
		}
		for _, v := range row {
			if strings.TrimSpace(v) == "Transaction Posted Timestamp" {
				headerAt = i
				headers = append([]string{}, row...)
				break
			}
		}
		if headerAt >= 0 {
			break
		}
	}
	if headerAt < 0 {
		return nil, fmt.Errorf("找不到 Transaction Posted Timestamp 表头")
	}
	seen := map[string]bool{}
	for i, h := range headers {
		h = strings.TrimSpace(strings.TrimPrefix(h, "\ufeff"))
		headers[i] = h
		if h == "" {
			continue
		}
		if seen[h] {
			return nil, fmt.Errorf("重复表头: %s", h)
		}
		seen[h] = true
	}
	for _, h := range []string{"Transaction Posted Timestamp", "Transaction Type", "Transaction Description", "Amount Type", "Amount", "Currency", "Transaction Key"} {
		if !seen[h] {
			return nil, fmt.Errorf("缺少表头: %s", h)
		}
	}
	result := []model.TaxRawRow{}
	for i := headerAt + 1; i < len(rows); i++ {
		row := rows[i]
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		// Walmart inserts a file line-count notice after the header; it is not a transaction.
		if len(row) > 0 {
			fields := strings.Fields(row[0])
			if len(fields) == 6 && strings.Join(fields[:5], " ") == "Number of Lines in file" {
				if _, err := strconv.ParseUint(fields[5], 10, 64); err == nil {
					continue
				}
			}
		}
		if len(row) > len(headers) {
			return nil, fmt.Errorf("第 %d 行字段多于表头", i+1)
		}
		data := map[string]string{}
		for j, h := range headers {
			if h != "" {
				data[h] = ""
				if j < len(row) {
					data[h] = row[j]
				}
			}
		}
		if strings.TrimSpace(data["Transaction Type"]) == "PaymentSummary" {
			continue
		}
		if _, err := parseTaxAmount(data["Amount"]); err != nil {
			return nil, fmt.Errorf("第 %d 行 Amount 无效: %w", i+1, err)
		}
		if parsePostedAt(data["Transaction Posted Timestamp"]).IsZero() {
			return nil, fmt.Errorf("第 %d 行交易时间格式无效", i+1)
		}
		currency := strings.ToUpper(strings.TrimSpace(data["Currency"]))
		if currency != "" && (len(currency) != 3 || strings.Trim(currency, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "") {
			return nil, fmt.Errorf("第 %d 行 Currency 无效", i+1)
		}
		result = append(result, model.TaxRawRow{RowNumber: i + 1, Data: data, Fingerprint: func() string {
			if identity := taxRowIdentity(data); identity != "" {
				return identity
			}
			return fmt.Sprintf("raw-row-%d", i+1)
		}()})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("账单没有数据行")
	}
	return result, nil
}

func taxCurrency(value string) string {
	currency := strings.ToUpper(strings.TrimSpace(value))
	if currency == "" {
		return "USD"
	}
	return currency
}

func parseTaxAmount(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero, nil
	}
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		s = "-" + s[1:len(s)-1]
	}
	// Group separators are accepted only in standard groups of three.
	if parts := strings.SplitN(s, ".", 2); len(parts) == 2 && strings.Contains(parts[1], ",") {
		return decimal.Zero, fmt.Errorf("小数部分不能有千位分隔符")
	}
	if strings.Contains(s, ",") {
		integer := strings.SplitN(strings.TrimPrefix(strings.TrimPrefix(s, "-"), "+"), ".", 2)[0]
		groups := strings.Split(integer, ",")
		if len(groups[0]) < 1 || len(groups[0]) > 3 {
			return decimal.Zero, fmt.Errorf("千位分隔符无效")
		}
		for _, g := range groups[1:] {
			if len(g) != 3 {
				return decimal.Zero, fmt.Errorf("千位分隔符无效")
			}
		}
		s = strings.ReplaceAll(s, ",", "")
	}
	if len(s) > 40 {
		return decimal.Zero, fmt.Errorf("金额过长")
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, err
	}
	if d.Exponent() < -8 || d.Exponent() > 14 || d.Abs().GreaterThanOrEqual(decimal.New(1, 14)) {
		return decimal.Zero, fmt.Errorf("金额超出精度范围")
	}
	return d, nil
}

func parsePostedAt(value string) time.Time {
	value = strings.TrimSpace(value)
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05 -0700", "2006-01-02 15:04:05 MST", "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02", "1/2/2006 15:04:05", "1/2/2006 3:04:05 PM", "1/2/2006", "2006/1/2 15:04:05"} {
		if t, e := time.Parse(layout, value); e == nil {
			return t
		}
	}
	if n, e := strconv.ParseInt(value, 10, 64); e == nil {
		if len(value) == 13 {
			return time.UnixMilli(n).UTC()
		}
		if len(value) == 10 {
			return time.Unix(n, 0).UTC()
		}
	}
	if n, e := strconv.ParseFloat(value, 64); e == nil && n >= 20000 && n < 100000 {
		t, e := excelize.ExcelDateToTime(n, false)
		if e == nil {
			return t
		}
	}
	return time.Time{}
}

func processTaxRows(raw []model.TaxRawRow, rules []model.TaxFeeRule, rates map[string]model.TaxRate) ([]model.TaxProcessedRow, []model.TaxQuarter, int, error) {
	lookup := map[string]model.TaxFeeRule{}
	for _, r := range rules {
		lookup[model.TaxMatchKey(r.TransactionType, r.TransactionDescription, r.AmountType)] = r
	}
	output := []model.TaxProcessedRow{}
	quarterMap := map[string]*model.TaxQuarter{}
	unmatched := 0
	for _, row := range raw {
		m := row.Data
		amount, e := parseTaxAmount(m["Amount"])
		if e != nil {
			return nil, nil, 0, e
		}
		shipQty, e := parseTaxAmount(m["Ship Qty"])
		if e != nil {
			return nil, nil, 0, fmt.Errorf("第 %d 行 Ship Qty 无效", row.RowNumber)
		}
		posted := parsePostedAt(m["Transaction Posted Timestamp"])
		if posted.IsZero() {
			return nil, nil, 0, fmt.Errorf("第 %d 行时间无效", row.RowNumber)
		}
		quarter := fmt.Sprintf("%d-Q%d", posted.Year(), (int(posted.Month())-1)/3+1)
		currency := taxCurrency(m["Currency"])
		rate, ok := rates[currency+":"+taxRateDate(posted)]
		if !ok {
			rate, ok = rates[currency]
		}
		if !ok || rate.Rate.IsNegative() {
			return nil, nil, 0, fmt.Errorf("缺少 %s 的有效汇率", currency)
		}
		calculationRate := rate.Rate
		fee := amount.Mul(calculationRate).Round(2)
		if fee.Abs().GreaterThanOrEqual(decimal.New(1, 20)) {
			return nil, nil, 0, fmt.Errorf("换算费用超出范围")
		}
		rule, matched := lookup[model.TaxMatchKey(m["Transaction Type"], m["Transaction Description"], m["Amount Type"])]
		category := rule.CustomFeeType
		if !matched {
			category = "未匹配"
			unmatched++
		}
		output = append(output, model.TaxProcessedRow{Month: posted.Format("2006-01"), Quarter: quarter, FeeType: category, Fee: fee, Amount: amount, ShipQty: shipQty, Rate: calculationRate, Currency: currency, RuleSnapshot: rule, TransactionKey: m["Transaction Key"]})
		q := quarterMap[quarter]
		if q == nil {
			q = &model.TaxQuarter{Quarter: quarter, Totals: map[string]decimal.Decimal{}}
			quarterMap[quarter] = q
		}
		// Keep full precision while accumulating the quarter. Rounding each
		// detail before summing causes a different result from the tax report.
		q.Totals[category] = q.Totals[category].Add(amount.Mul(calculationRate))
		// PaymentSummary is a settlement total, not another revenue/expense transaction.
		if category != "汇总" && category != "未匹配" {
			q.Total = q.Total.Add(amount.Mul(calculationRate))
		}
		if category == "销售额" || category == "退款" {
			q.TaxableIncome = q.TaxableIncome.Add(amount.Mul(calculationRate))
		}
	}
	quarters := []model.TaxQuarter{}
	for _, q := range quarterMap {
		for category, total := range q.Totals {
			q.Totals[category] = total.Round(2)
		}
		q.Total = q.Total.Round(2)
		q.TaxableIncome = q.TaxableIncome.Round(2)
		quarters = append(quarters, *q)
	}
	sort.Slice(quarters, func(i, j int) bool { return quarters[i].Quarter < quarters[j].Quarter })
	return output, quarters, unmatched, nil
}

func fetchTaxRate(ctx context.Context, currency string) (model.TaxRate, error) {
	if currency == "CNY" {
		return model.TaxRate{Rate: decimal.NewFromInt(1), Source: "CNY", Date: time.Now().In(taxExchangeLocation).Format("2006-01-02"), FetchedAt: time.Now()}, nil
	}
	if !slices.Contains(exchangeCurrencies, currency) {
		return model.TaxRate{}, fmt.Errorf("不支持的币种 %s", currency)
	}
	rates, err := fetchChinaMoneyRates(ctx)
	if err != nil {
		return model.TaxRate{}, err
	}
	return rates[currency], nil
}

func taxError(c *gin.Context, err error) {
	// Keep the database error visible to authorized finance users so a failed
	// production query can be diagnosed instead of collapsing every failure to
	// the same unhelpful response.
	c.JSON(500, gin.H{"error": "报税数据操作失败", "details": err.Error()})
	_ = c.Error(err)
}

func taxSiteID(value string) string {
	site := strings.ToUpper(strings.TrimSpace(value))
	if site == "" {
		return "US"
	}
	if len(site) > 20 {
		return site[:20]
	}
	return site
}

func taxSiteFromContext(c *gin.Context) string {
	return taxSiteID(c.Query("site_id"))
}

func taxSiteFromForm(c *gin.Context) string {
	return taxSiteID(c.PostForm("site_id"))
}

// Legacy Walmart imports belong to the original default store only.
func filterWalmartStore(q *gorm.DB, store string) *gorm.DB {
	store = strings.TrimSpace(store)
	if store == "" {
		return q
	}
	column := clause.Column{Table: clause.CurrentTable, Name: "store_id"}
	if store == "TVOFEK" || store == "walmart" || store == "walmart-shop-001" {
		return q.Where(clause.Or(
			clause.IN{Column: column, Values: []interface{}{"TVOFEK", "walmart", "walmart-shop-001", ""}},
			clause.Eq{Column: column, Value: nil},
		))
	}
	return q.Where(clause.Eq{Column: column, Value: store})
}

// The default store was imported under multiple IDs. Keep the earliest copy
// of the same transaction in the same country, without merging other stores.
func excludeWalmartDuplicateRows(q *gorm.DB, table string) *gorm.DB {
	return q.Where(`(COALESCE(` + table + `.store_id, '') NOT IN ('TVOFEK', 'walmart', 'walmart-shop-001', '')
		OR COALESCE(` + table + `.fingerprint, '') = '' OR ` + table + `.id IS NULL
		OR ` + table + `.id IN (SELECT MIN(id) FROM tax_raw_rows
		WHERE COALESCE(store_id, '') IN ('TVOFEK', 'walmart', 'walmart-shop-001', '') AND fingerprint <> ''
		GROUP BY COALESCE(NULLIF(site_id, ''), 'US'), fingerprint))`)
}

func filterWalmartSummaryBatches(q, db *gorm.DB, column, store, site string) *gorm.DB {
	if store != "" || site != "" {
		batches := filterTaxSite(filterWalmartStore(db.Model(&model.TaxDataBatch{}), store), site).Select("id")
		q = q.Where(column+" IN (?)", batches)
	}
	// Preserve pre-batch legacy summaries, but count repeated files only once.
	duplicates := db.Model(&model.TaxDataBatch{}).Select("id").Where(`EXISTS (SELECT 1 FROM tax_data_batches prior
		WHERE COALESCE(tax_data_batches.store_id, '') IN ('TVOFEK', 'walmart', 'walmart-shop-001', '')
		AND COALESCE(prior.store_id, '') IN ('TVOFEK', 'walmart', 'walmart-shop-001', '')
		AND COALESCE(NULLIF(prior.site_id, ''), 'US') = COALESCE(NULLIF(tax_data_batches.site_id, ''), 'US')
		AND prior.file_hash = tax_data_batches.file_hash AND prior.file_hash <> ''
		AND prior.id < tax_data_batches.id)`)
	return q.Where(column+" NOT IN (?)", duplicates)
}

func filterTaxSite(q *gorm.DB, site string) *gorm.DB {
	if strings.TrimSpace(site) == "" {
		return q
	}
	column := clause.Column{Table: clause.CurrentTable, Name: "site_id"}
	return q.Where(clause.Or(
		clause.Eq{Column: column, Value: taxSiteID(site)},
		clause.Eq{Column: column, Value: ""},
		clause.Eq{Column: column, Value: nil},
	))
}

func chunkTaxIDs(ids []uint) [][]uint {
	const chunkSize = 5000
	chunks := make([][]uint, 0, (len(ids)+chunkSize-1)/chunkSize)
	for start := 0; start < len(ids); start += chunkSize {
		end := start + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[start:end])
	}
	return chunks
}

func taxPage(c *gin.Context) (int, int) {
	p, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if p < 1 || p > 100000 {
		p = 1
	}
	if size < 1 || size > 200 {
		size = 50
	}
	return p, size
}
func ListTaxData(c *gin.Context) {
	page, size := taxPage(c)
	batches := []model.TaxDataBatch{}
	var total int64
	q := database.DB.Model(&model.TaxDataBatch{})
	if store := c.Query("store_id"); store != "" {
		q = filterWalmartStore(q, store)
	}
	q = filterTaxSite(q, c.Query("site_id"))
	if err := q.Count(&total).Error; err != nil {
		taxError(c, err)
		return
	}
	if err := q.Omit("file_content").Order("id desc").Offset((page - 1) * size).Limit(size).Find(&batches).Error; err != nil {
		taxError(c, err)
		return
	}
	c.JSON(200, gin.H{"batches": batches, "total": total, "page": page, "page_size": size})
}
func ListTaxRaw(c *gin.Context) {
	page, size := taxPage(c)
	rows := []model.TaxRawRow{}
	var total int64
	q := taxRawBaseQuery(c)
	filters := taxRawFilters(c)
	q = applyTaxRawFilters(q, filters)
	if err := q.Count(&total).Error; err != nil {
		taxError(c, err)
		return
	}
	if err := q.Order("id").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		taxError(c, err)
		return
	}
	rowIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		rowIDs = append(rowIDs, row.ID)
	}
	type rowRate struct {
		RawRowID uint            `json:"raw_row_id"`
		Month    string          `json:"month"`
		Quarter  string          `json:"quarter"`
		FeeType  string          `json:"fee_type"`
		Fee      decimal.Decimal `json:"fee"`
		Rate     decimal.Decimal `json:"rate"`
		Currency string          `json:"currency"`
	}
	rates := map[uint]rowRate{}
	if len(rowIDs) > 0 {
		var snapshots []rowRate
		for _, ids := range chunkTaxIDs(rowIDs) {
			var part []rowRate
			if err := database.DB.Model(&model.TaxProcessedRow{}).Select("raw_row_id, month, quarter, fee_type, fee, rate, currency").Where("raw_row_id IN ?", ids).Scan(&part).Error; err != nil {
				taxError(c, err)
				return
			}
			snapshots = append(snapshots, part...)
		}
		for _, snapshot := range snapshots {
			rates[snapshot.RawRowID] = snapshot
		}
		var exchangeRates []model.TaxExchangeRate
		if err := database.DB.Where("currency IN ?", exchangeCurrencies).Find(&exchangeRates).Error; err != nil {
			taxError(c, err)
			return
		}
		dailyRates := map[string]decimal.Decimal{}
		for _, exchangeRate := range exchangeRates {
			dailyRates[exchangeRate.Currency+":"+exchangeRate.Date] = exchangeRate.Rate
		}
		for _, row := range rows {
			posted := parsePostedAt(row.Data["Transaction Posted Timestamp"])
			currency := taxCurrency(row.Data["Currency"])
			if posted.IsZero() {
				continue
			}
			rate := decimal.NewFromInt(1)
			if currency != "CNY" {
				var ok bool
				rate, ok = storedTaxRate(dailyRates, currency, posted)
				if !ok {
					continue
				}
			}
			if snapshot, ok := rates[row.ID]; ok {
				snapshot.Rate = rate
				if amount, err := parseTaxAmount(row.Data["Amount"]); err == nil {
					snapshot.Fee = amount.Mul(rate).Round(2)
				}
				rates[row.ID] = snapshot
			}
		}
	}
	response := gin.H{"items": rows, "total": total, "headers": taxHeaders, "rates": rates}
	if wantsTaxTotals(c.Query("include_totals")) {
		allRows := []model.TaxRawRow{}
		if err := q.Limit(-1).Offset(-1).Order("id").Find(&allRows).Error; err != nil {
			taxError(c, err)
			return
		}
		totals := map[string]decimal.Decimal{}
		rowIDs := make([]uint, 0, len(allRows))
		for _, row := range allRows {
			rowIDs = append(rowIDs, row.ID)
			addJSONTotals(totals, row.Data)
		}
		if len(rowIDs) > 0 {
			var processed []model.TaxProcessedRow
			for _, ids := range chunkTaxIDs(rowIDs) {
				var part []model.TaxProcessedRow
				if err := database.DB.Select("raw_row_id, fee").Where("raw_row_id IN ?", ids).Find(&part).Error; err != nil {
					taxError(c, err)
					return
				}
				processed = append(processed, part...)
			}
			for _, row := range processed {
				totals["fee"] = totals["fee"].Add(row.Fee)
			}
		}
		response["totals"] = totals
	}
	c.JSON(200, response)
}

func taxPeriodValues(value string) []string {
	values := []string{}
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			values = append(values, part)
		}
	}
	return values
}

func taxRawBaseQuery(c *gin.Context) *gorm.DB {
	q := database.DB.Model(&model.TaxRawRow{}).Where("BTRIM(COALESCE(data->>'Transaction Type', '')) <> ?", "PaymentSummary")
	q = excludeWalmartDuplicateRows(q, "tax_raw_rows")
	if batchID := c.Param("id"); batchID != "" {
		q = q.Where("batch_id = ?", batchID)
	}
	if store := c.Query("store_id"); store != "" {
		q = filterWalmartStore(q, store)
	}
	q = filterTaxSite(q, c.Query("site_id"))
	if quarter := c.Query("quarter"); quarter != "" {
		q = q.Where("id IN (?)", database.DB.Model(&model.TaxProcessedRow{}).Select("raw_row_id").Where("quarter IN ?", taxPeriodValues(quarter)))
	}
	return q
}

func taxRawFilters(c *gin.Context) map[string]string {
	filters := map[string]string{}
	if encoded := strings.TrimSpace(c.Query("filters")); encoded != "" {
		_ = json.Unmarshal([]byte(encoded), &filters)
	}
	if field, value := strings.TrimSpace(c.Query("filter_field")), strings.TrimSpace(c.Query("filter_value")); field != "" && value != "" {
		filters[field] = value
	}
	return filters
}

func applyTaxRawFilters(q *gorm.DB, filters map[string]string) *gorm.DB {
	for field, value := range filters {
		if value == "" {
			continue
		}
		switch field {
		case "月份", "季度", "费用类型":
			column := map[string]string{"月份": "month", "季度": "quarter", "费用类型": "fee_type"}[field]
			derived := database.DB.Model(&model.TaxProcessedRow{}).Select("raw_row_id")
			if field == "费用类型" {
				derived = derived.Where(column+" = ?", value)
			} else {
				derived = derived.Where(column+" IN ?", taxPeriodValues(value))
			}
			q = q.Where("id IN (?)", derived)
		}
	}
	return q
}

func ListTaxRawFilterOptions(c *gin.Context) {
	field := strings.TrimSpace(c.Query("field"))
	if field == "" {
		c.JSON(400, gin.H{"error": "筛选字段不能为空"})
		return
	}
	q := taxRawBaseQuery(c)
	filters := taxRawFilters(c)
	delete(filters, field)
	q = applyTaxRawFilters(q, filters)
	derived := map[string]string{"月份": "month", "季度": "quarter", "费用类型": "fee_type"}
	if column, ok := derived[field]; ok {
		var values []string
		if err := database.DB.Model(&model.TaxProcessedRow{}).Where("raw_row_id IN (?)", q.Select("id")).Distinct(column).Order(column).Pluck(column, &values).Error; err != nil {
			taxError(c, err)
			return
		}
		c.JSON(200, gin.H{"values": values})
		return
	}
	if field != "月份" && field != "季度" && field != "费用类型" {
		c.JSON(400, gin.H{"error": "不支持的筛选字段"})
		return
	}
	c.JSON(200, gin.H{"values": []string{}})
}
func ListTaxProcessed(c *gin.Context) {
	page, size := taxPage(c)
	rows := []model.TaxProcessedRow{}
	var total int64
	q := database.DB.Model(&model.TaxProcessedRow{}).Where("batch_id = ?", c.Param("id"))
	if err := q.Count(&total).Error; err != nil {
		taxError(c, err)
		return
	}
	if err := q.Order("id").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		taxError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": rows, "total": total})
}
func UpdateTaxMonthRate(c *gin.Context) {
	month := strings.TrimSpace(c.Param("month"))
	if _, err := time.Parse("2006-01", month); err != nil {
		c.JSON(400, gin.H{"error": "月份格式无效"})
		return
	}
	var body struct {
		Rate     string `json:"rate"`
		Currency string `json:"currency"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "汇率格式无效"})
		return
	}
	rate, err := decimal.NewFromString(strings.TrimSpace(body.Rate))
	if err != nil || !rate.IsPositive() || rate.GreaterThan(decimal.New(1000, 0)) {
		c.JSON(400, gin.H{"error": "汇率必须是 0 到 1000 之间的正数"})
		return
	}
	currency := taxCurrency(body.Currency)
	if currency != "USD" {
		c.JSON(400, gin.H{"error": "目前仅支持修改 USD/CNY 汇率"})
		return
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		var rows []model.TaxProcessedRow
		if err := tx.Where("month = ? AND currency = ?", month, currency).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return fmt.Errorf("该月没有可修改的 %s 处理数据", currency)
		}
		for i := range rows {
			rows[i].Rate = rate
			rows[i].Fee = rows[i].Amount.Mul(rate).Round(2)
			if err := tx.Model(&rows[i]).Updates(map[string]interface{}{"rate": rows[i].Rate, "fee": rows[i].Fee}).Error; err != nil {
				return err
			}
		}
		batchIDs := map[uint]bool{}
		for _, row := range rows {
			batchIDs[row.BatchID] = true
		}
		quarter := fmt.Sprintf("%d-Q%d", mustTaxMonth(month).Year(), (int(mustTaxMonth(month).Month())-1)/3+1)
		for batchID := range batchIDs {
			var processed []model.TaxProcessedRow
			if err := tx.Where("batch_id = ? AND quarter = ?", batchID, quarter).Find(&processed).Error; err != nil {
				return err
			}
			totals := map[string]decimal.Decimal{}
			total := decimal.Zero
			income := decimal.Zero
			for _, row := range processed {
				fee := row.Amount.Mul(row.Rate)
				totals[row.FeeType] = totals[row.FeeType].Add(fee)
				if row.FeeType != "汇总" && row.FeeType != "未匹配" {
					total = total.Add(fee)
				}
				if row.FeeType == "销售额" || row.FeeType == "退款" {
					income = income.Add(fee)
				}
			}
			for category, value := range totals {
				totals[category] = value.Round(2)
			}
			total = total.Round(2)
			income = income.Round(2)
			if err := tx.Model(&model.TaxQuarter{}).Where("batch_id = ? AND quarter = ?", batchID, quarter).Updates(map[string]interface{}{"totals": totals, "total": total, "taxable_income": income}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(422, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "月度汇率已更新", "month": month, "currency": currency, "rate": rate})
}

func mustTaxMonth(month string) time.Time {
	parsed, _ := time.Parse("2006-01", month)
	return parsed
}

func ListTaxRules(c *gin.Context) {
	rules := []model.TaxFeeRule{}
	if err := database.DB.Order("id").Find(&rules).Error; err != nil {
		taxError(c, err)
		return
	}
	c.JSON(200, rules)
}

func CreateTaxRule(c *gin.Context) {
	var body struct {
		TransactionType        string `json:"transaction_type"`
		TransactionDescription string `json:"transaction_description"`
		AmountType             string `json:"amount_type"`
		CustomFeeType          string `json:"custom_fee_type"`
		Allocation             string `json:"allocation"`
		Note                   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则格式无效"})
		return
	}
	body.TransactionType = strings.TrimSpace(body.TransactionType)
	body.TransactionDescription = strings.TrimSpace(body.TransactionDescription)
	body.AmountType = strings.TrimSpace(body.AmountType)
	body.CustomFeeType = strings.TrimSpace(body.CustomFeeType)
	if body.TransactionType == "" || body.TransactionDescription == "" || body.AmountType == "" || body.CustomFeeType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "交易类型、交易描述、金额类型和费用类型不能为空"})
		return
	}
	rule := model.TaxFeeRule{MatchKey: model.TaxMatchKey(body.TransactionType, body.TransactionDescription, body.AmountType), TransactionType: body.TransactionType, TransactionDescription: body.TransactionDescription, AmountType: body.AmountType, CustomFeeType: body.CustomFeeType, Allocation: strings.TrimSpace(body.Allocation), Note: strings.TrimSpace(body.Note)}
	if err := database.DB.Create(&rule).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			c.JSON(http.StatusConflict, gin.H{"error": "该匹配规则已存在"})
			return
		}
		taxError(c, err)
		return
	}
	if err := rematchExistingTaxData(); err != nil {
		taxError(c, fmt.Errorf("规则已保存，但历史数据重算失败: %w", err))
		return
	}
	c.JSON(http.StatusCreated, rule)
}

func rematchExistingTaxData() error {
	var rules []model.TaxFeeRule
	if err := database.DB.Find(&rules).Error; err != nil {
		return err
	}
	lookup := make(map[string]model.TaxFeeRule, len(rules))
	for _, rule := range rules {
		lookup[model.TaxMatchKey(rule.TransactionType, rule.TransactionDescription, rule.AmountType)] = rule
	}
	var rawRows []model.TaxRawRow
	if err := database.DB.Where("BTRIM(COALESCE(data->>'Transaction Type', '')) <> ?", "PaymentSummary").Find(&rawRows).Error; err != nil {
		return err
	}
	var processed []model.TaxProcessedRow
	if err := database.DB.Find(&processed).Error; err != nil {
		return err
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		byRaw := make(map[uint]*model.TaxProcessedRow, len(processed))
		for i := range processed {
			byRaw[processed[i].RawRowID] = &processed[i]
		}
		for _, raw := range rawRows {
			row, ok := byRaw[raw.ID]
			if !ok {
				continue
			}
			rule, matched := lookup[model.TaxMatchKey(raw.Data["Transaction Type"], raw.Data["Transaction Description"], raw.Data["Amount Type"])]
			category := rule.CustomFeeType
			if !matched {
				category = "未匹配"
			}
			row.FeeType = category
			row.RuleSnapshot = rule
			if err := tx.Model(row).Updates(map[string]interface{}{"fee_type": category, "rule_snapshot": rule}).Error; err != nil {
				return err
			}
		}
		return rebuildTaxQuarterSnapshots(tx, processed)
	})
}
func ListTaxQuarters(c *gin.Context) {
	items, categories, err := summarizeTaxProcessed(database.DB, c.Query("store_id"), "", c.Query("site_id"))
	if err != nil {
		taxError(c, err)
		return
	}
	dateFrom, dateTo := c.Query("date_from"), c.Query("date_to")
	if (dateFrom != "" && !validTaxDate(dateFrom)) || (dateTo != "" && !validTaxDate(dateTo)) || (dateFrom != "" && dateTo != "" && dateFrom > dateTo) {
		c.JSON(400, gin.H{"error": "日期范围无效"})
		return
	}
	available := make([]string, 0, len(items))
	result := make([]taxQuarterSummary, 0, len(items))
	monthly, _, err := summarizeTaxProcessedPeriod(database.DB, c.Query("store_id"), "", "month", c.Query("site_id"))
	if err != nil {
		taxError(c, err)
		return
	}
	byQuarter := map[string][]taxQuarterSummary{}
	for _, item := range monthly {
		monthEnd := item.Month + "-31"
		if dateFrom != "" && monthEnd < dateFrom || dateTo != "" && item.Month+"-01" > dateTo {
			continue
		}
		byQuarter[item.Quarter] = append(byQuarter[item.Quarter], item)
	}
	for i := len(items) - 1; i >= 0; i-- {
		available = append(available, items[i].Quarter)
	}
	for _, item := range items {
		quarterStart, quarterEnd := quarterDateRange(item.Quarter)
		if (dateFrom == "" || quarterEnd >= dateFrom) && (dateTo == "" || quarterStart <= dateTo) && (c.Query("quarter") == "" || slices.Contains(taxPeriodValues(c.Query("quarter")), item.Quarter)) {
			item.Months = byQuarter[item.Quarter]
			result = append(result, item)
		}
	}
	monthQuery := database.DB.Model(&model.TaxProcessedRow{}).Where("month <> ''")
	if store := c.Query("store_id"); store != "" || strings.TrimSpace(c.Query("site_id")) != "" {
		monthQuery = filterWalmartSummaryBatches(monthQuery, database.DB, "batch_id", store, c.Query("site_id"))
	}
	var availableMonths []string
	if err := monthQuery.Distinct("month").Order("month").Pluck("month", &availableMonths).Error; err != nil {
		taxError(c, err)
		return
	}
	sort.Strings(availableMonths)
	if c.Query("view") == "month" {
		result = make([]taxQuarterSummary, 0, len(monthly))
		for _, item := range monthly {
			monthEnd := item.Month + "-31"
			if (dateFrom == "" || monthEnd >= dateFrom) && (dateTo == "" || item.Month+"-01" <= dateTo) {
				result = append(result, item)
			}
		}
	}
	c.JSON(200, gin.H{"items": result, "categories": categories, "available_quarters": available, "available_months": availableMonths})
}

func validTaxDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func quarterDateRange(quarter string) (string, string) {
	year, q := quarter[:4], quarter[len(quarter)-1:]
	start := map[string]string{"1": "01-01", "2": "04-01", "3": "07-01", "4": "10-01"}[q]
	end := map[string]string{"1": "03-31", "2": "06-30", "3": "09-30", "4": "12-31"}[q]
	return year + "-" + start, year + "-" + end
}

func ListTaxExchangeRates(c *gin.Context) {
	type dateRow struct {
		Date      string          `json:"date"`
		Rate      decimal.Decimal `json:"rate"`
		Currency  string          `json:"currency"`
		Source    string          `json:"source"`
		FetchedAt time.Time       `json:"fetched_at"`
	}
	var rows []dateRow
	currency := strings.ToUpper(strings.TrimSpace(c.DefaultQuery("currency", "USD")))
	if !slices.Contains(exchangeCurrencies, currency) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的币种"})
		return
	}
	if err := database.DB.Model(&model.TaxExchangeRate{}).Select("date, rate, currency, source, fetched_at").Where("currency = ?", currency).Order("date").Find(&rows).Error; err != nil {
		taxError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": rows})
}

// Include pending TikTok imports when finding the historical rate range.
func earliestTaxExchangeDate(db *gorm.DB) (time.Time, error) {
	var values []string
	if err := db.Model(&model.TaxRawRow{}).Where("BTRIM(COALESCE(data->>'Transaction Posted Timestamp', '')) <> ''").Distinct().Pluck("data->>'Transaction Posted Timestamp'", &values).Error; err != nil {
		return time.Time{}, err
	}
	var earliest time.Time
	for _, value := range values {
		if posted := parsePostedAt(value); !posted.IsZero() && (earliest.IsZero() || posted.Before(earliest)) {
			earliest = posted
		}
	}
	var earliestTikTok struct{ Date string }
	if err := db.Raw(`SELECT MIN(date) AS date FROM (
		SELECT MIN(date) AS date FROM tik_tok_tax_ads
		UNION ALL
		SELECT to_char(MIN(posted_at), 'YYYY-MM-DD') AS date FROM tik_tok_tax_details
		UNION ALL
		SELECT MIN(statement_date) AS date FROM tik_tok_settlements
	) dates`).Scan(&earliestTikTok).Error; err != nil {
		return time.Time{}, err
	}
	if date := parseTikTokPostedAt(earliestTikTok.Date); !date.IsZero() && (earliest.IsZero() || date.Before(earliest)) {
		earliest = date
	}
	var amazonEarliest struct{ Date string }
	if err := db.Raw("SELECT to_char(MIN(date), 'YYYY-MM-DD') AS date FROM (SELECT MIN(" + amazonDateSQL("data->>'配送日期'") + ") AS date FROM amazon_tax_shipments UNION ALL SELECT MIN(" + amazonDateSQL("data->>'date/time'") + ") AS date FROM amazon_tax_finances) dates").Scan(&amazonEarliest).Error; err != nil {
		return time.Time{}, err
	}
	if date := parseAmazonTaxDate(amazonEarliest.Date); !date.IsZero() && (earliest.IsZero() || date.Before(earliest)) {
		earliest = date
	}
	return earliest, nil
}

// Determine the complete calendar range before any provider request.
func taxExchangeDateRange(earliest, now time.Time) []string {
	if earliest.IsZero() {
		return nil
	}
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	first, last := earliest.In(location), now.In(location)
	start := time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, location)
	end := time.Date(last.Year(), last.Month(), last.Day(), 0, 0, 0, 0, location)
	dates := []string{}
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		dates = append(dates, day.Format("2006-01-02"))
	}
	return dates
}

type taxExchangeRequest struct{ Date, Currency string }

func missingTaxExchangeRequests(dates []string, existing []model.TaxExchangeRate) []taxExchangeRequest {
	available := make(map[string]bool, len(existing))
	for _, rate := range existing {
		if rate.Rate.IsPositive() {
			available[rate.Date+":"+rate.Currency] = true
		}
	}
	missing := []taxExchangeRequest{}
	for _, date := range dates {
		day, err := time.Parse("2006-01-02", date)
		if err == nil && (day.Weekday() == time.Saturday || day.Weekday() == time.Sunday) {
			continue
		}
		for _, currency := range exchangeCurrencies {
			if !available[date+":"+currency] {
				missing = append(missing, taxExchangeRequest{Date: date, Currency: currency})
			}
		}
	}
	return missing
}

func RefreshTaxExchangeRates(c *gin.Context) {
	date, err := saveLatestTaxExchangeRates(c.Request.Context(), database.DB)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"status": false, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": true, "msg": "最新汇率已保存", "date": date, "count": 1, "rate_count": len(exchangeCurrencies)})
}

// rewriteTaxDataWithDailyRates applies the stored rate for each transaction's
// own posted date, then rebuilds quarter snapshots from unrounded values.
func rewriteTaxDataWithDailyRates() error {
	var rates []model.TaxExchangeRate
	if err := database.DB.Find(&rates).Error; err != nil {
		return err
	}
	rateByDate := make(map[string]decimal.Decimal, len(rates))
	for _, rate := range rates {
		rateByDate[rate.Currency+":"+rate.Date] = rate.Rate
	}
	var rawRows []model.TaxRawRow
	if err := database.DB.Select("id, data").Find(&rawRows).Error; err != nil {
		return err
	}
	rawByID := make(map[uint]model.TaxRawRow, len(rawRows))
	for _, row := range rawRows {
		rawByID[row.ID] = row
	}
	return database.DB.Transaction(func(tx *gorm.DB) error {
		var processed []model.TaxProcessedRow
		if err := tx.Find(&processed).Error; err != nil {
			return err
		}
		for i := range processed {
			raw, ok := rawByID[processed[i].RawRowID]
			if !ok {
				continue
			}
			posted := parsePostedAt(raw.Data["Transaction Posted Timestamp"])
			if posted.IsZero() {
				continue
			}
			rate := decimal.NewFromInt(1)
			currency := taxCurrency(raw.Data["Currency"])
			if currency != "CNY" {
				var found bool
				rate, found = storedTaxRate(rateByDate, currency, posted)
				if !found || !rate.IsPositive() {
					continue
				}
			}
			processed[i].Rate = rate
			processed[i].Fee = processed[i].Amount.Mul(rate).Round(2)
			if err := tx.Model(&processed[i]).Updates(map[string]interface{}{"rate": rate, "fee": processed[i].Fee}).Error; err != nil {
				return err
			}
		}
		return rebuildTaxQuarterSnapshots(tx, processed)
	})
}

func rebuildTaxQuarterSnapshots(tx *gorm.DB, processed []model.TaxProcessedRow) error {
	type aggregate struct {
		totals        map[string]decimal.Decimal
		total, income decimal.Decimal
	}
	groups := map[string]*aggregate{}
	for _, row := range processed {
		key := fmt.Sprintf("%d:%s", row.BatchID, row.Quarter)
		if groups[key] == nil {
			groups[key] = &aggregate{totals: map[string]decimal.Decimal{}}
		}
		fee := row.Amount.Mul(row.Rate)
		groups[key].totals[row.FeeType] = groups[key].totals[row.FeeType].Add(fee)
		if row.FeeType != "汇总" && row.FeeType != "未匹配" {
			groups[key].total = groups[key].total.Add(fee)
		}
		if row.FeeType == "销售额" || row.FeeType == "退款" {
			groups[key].income = groups[key].income.Add(fee)
		}
	}
	for key, aggregate := range groups {
		parts := strings.SplitN(key, ":", 2)
		batchID, _ := strconv.ParseUint(parts[0], 10, 64)
		for category, value := range aggregate.totals {
			aggregate.totals[category] = value.Round(2)
		}
		if err := tx.Model(&model.TaxQuarter{}).Where("batch_id = ? AND quarter = ?", uint(batchID), parts[1]).Updates(map[string]interface{}{"totals": aggregate.totals, "total": aggregate.total.Round(2), "taxable_income": aggregate.income.Round(2)}).Error; err != nil {
			return err
		}
	}
	return nil
}

type taxQuarterSummary struct {
	model.TaxQuarter
	Month            string                     `json:"month,omitempty"`
	Months           []taxQuarterSummary        `json:"months,omitempty"`
	UnmatchedRows    int64                      `json:"unmatched_rows"`
	USDTotals        map[string]decimal.Decimal `json:"usd_totals"`
	USDTotal         decimal.Decimal            `json:"usd_total"`
	USDTaxableIncome decimal.Decimal            `json:"usd_taxable_income"`
	USDToCNY         decimal.Decimal            `json:"usd_to_cny_rate"`
	MonthlyRates     map[string]decimal.Decimal `json:"monthly_rates"`
	SalesVolume      decimal.Decimal            `json:"sales_volume"`
}

func summarizeTaxProcessed(db *gorm.DB, store, quarter string, sites ...string) ([]taxQuarterSummary, []string, error) {
	return summarizeTaxProcessedPeriod(db, store, quarter, "quarter", sites...)
}

func summarizeTaxProcessedPeriod(db *gorm.DB, store, quarter, period string, sites ...string) ([]taxQuarterSummary, []string, error) {
	// Keep the raw site value: an empty site means "all countries" and must not
	// fall back to US.
	site := ""
	if len(sites) > 0 {
		site = strings.TrimSpace(sites[0])
	}
	grouped := map[string]*taxQuarterSummary{}
	known := map[string]bool{}
	for _, category := range taxCategories {
		known[category] = true
	}
	add := func(q, month, category string, fee, amount, shipQty, rate decimal.Decimal, currency string, count int64) {
		if category == "汇总" || (quarter != "" && !slices.Contains(taxPeriodValues(quarter), q)) {
			return
		}
		if rate.IsPositive() {
			fee = amount.Mul(rate)
		}
		groupKey := q
		if period == "month" && month != "" {
			groupKey = month
		}
		g := grouped[groupKey]
		if g == nil {
			g = &taxQuarterSummary{TaxQuarter: model.TaxQuarter{Quarter: q, Totals: map[string]decimal.Decimal{}}, Month: func() string {
				if period == "month" {
					return month
				}
				return ""
			}(), USDTotals: map[string]decimal.Decimal{}, MonthlyRates: map[string]decimal.Decimal{}}
			grouped[groupKey] = g
		}
		g.Totals[category] = g.Totals[category].Add(fee)
		if known[category] {
			g.Total = g.Total.Add(fee)
		}
		if category == "销售额" || category == "退款" {
			g.TaxableIncome = g.TaxableIncome.Add(fee)
		}
		if category == "销售额" {
			g.SalesVolume = g.SalesVolume.Add(shipQty)
		}
		if currency == "USD" {
			g.USDTotals[category] = g.USDTotals[category].Add(amount)
			if known[category] {
				g.USDTotal = g.USDTotal.Add(amount)
			}
			if category == "销售额" || category == "退款" {
				g.USDTaxableIncome = g.USDTaxableIncome.Add(amount)
			}
			if rate.IsPositive() && month != "" {
				g.MonthlyRates[month] = rate
			}
			if rate.IsPositive() && !g.USDToCNY.IsPositive() {
				g.USDToCNY = rate
			}
		}
		if category == "未匹配" {
			g.UnmatchedRows += count
		}
	}
	type summaryDetail struct {
		model.TaxProcessedRow
		RawShipQty  string
		RawPostedAt string
		RawCurrency string
	}
	var details []summaryDetail
	q := db.Table("tax_processed_rows p").Joins("LEFT JOIN tax_raw_rows r ON r.id = p.raw_row_id")
	q = filterWalmartSummaryBatches(q, db, "p.batch_id", store, site)
	q = excludeWalmartDuplicateRows(q, "r")
	// Project only the three original values needed by the summary. Avoid loading
	// full raw JSON and issuing a separate query for every thousand processed rows.
	if err := q.Select(`p.raw_row_id, p.batch_id, p.month, p.quarter, p.fee_type, p.fee, p.amount, p.rate, p.currency,
		r.data->>'Ship Qty' AS raw_ship_qty,
		r.data->>'Transaction Posted Timestamp' AS raw_posted_at,
		r.data->>'Currency' AS raw_currency`).Order("p.id").Scan(&details).Error; err != nil {
		return nil, nil, err
	}
	dailyRates := map[string]decimal.Decimal{}
	var exchangeRates []model.TaxExchangeRate
	if err := db.Where("currency IN ?", exchangeCurrencies).Find(&exchangeRates).Error; err != nil {
		return nil, nil, err
	}
	for _, exchangeRate := range exchangeRates {
		dailyRates[exchangeRate.Currency+":"+exchangeRate.Date] = exchangeRate.Rate
	}
	covered := map[uint]bool{}
	for _, row := range details {
		covered[row.BatchID] = true
		if row.FeeType == "销售额" {
			row.ShipQty, _ = parseTaxAmount(row.RawShipQty)
		}
		posted := parsePostedAt(row.RawPostedAt)
		currency := taxCurrency(row.RawCurrency)
		if !posted.IsZero() && currency == "CNY" {
			row.Rate = decimal.NewFromInt(1)
		} else if !posted.IsZero() {
			if rate, ok := storedTaxRate(dailyRates, currency, posted); ok {
				row.Rate = rate
			}
		}
		add(row.Quarter, row.Month, row.FeeType, row.Fee, row.Amount, row.ShipQty, row.Rate, row.Currency, 1)
	}
	// Compatibility is per batch: a new import must not hide older snapshot-only imports.
	var snapshots []model.TaxQuarter
	q = db.Model(&model.TaxQuarter{})
	q = filterWalmartSummaryBatches(q, db, "batch_id", store, site)
	q = q.Where("NOT EXISTS (SELECT 1 FROM tax_processed_rows p WHERE p.batch_id = tax_quarters.batch_id)")
	if err := q.Find(&snapshots).Error; err != nil {
		return nil, nil, err
	}
	snapshotBatches := map[uint]bool{}
	for _, row := range snapshots {
		if period == "month" {
			continue
		}
		if covered[row.BatchID] {
			continue
		}
		snapshotBatches[row.BatchID] = true
		for category, fee := range row.Totals {
			add(row.Quarter, "", category, fee, decimal.Zero, decimal.Zero, decimal.Zero, "", 0)
		}
	}
	for id := range snapshotBatches {
		covered[id] = true
	}
	// Raw-only imports can be displayed only using their saved FX, never today's rate.
	var raw []model.TaxRawRow
	var rawBatchIDs []uint
	rawBatchQuery := db.Model(&model.TaxRawRow{})
	rawBatchQuery = filterWalmartSummaryBatches(rawBatchQuery, db, "batch_id", store, site)
	rawBatchQuery = rawBatchQuery.Where("NOT EXISTS (SELECT 1 FROM tax_processed_rows p WHERE p.batch_id = tax_raw_rows.batch_id)")
	if store != "" {
		rawBatchQuery = filterWalmartStore(rawBatchQuery, store)
	}
	rawBatchQuery = filterTaxSite(rawBatchQuery, site)
	if err := rawBatchQuery.Distinct("batch_id").Pluck("batch_id", &rawBatchIDs).Error; err != nil {
		return nil, nil, err
	}
	uncoveredRawBatches := make([]uint, 0, len(rawBatchIDs))
	for _, id := range rawBatchIDs {
		if !covered[id] {
			uncoveredRawBatches = append(uncoveredRawBatches, id)
		}
	}
	if len(uncoveredRawBatches) > 0 {
		q = db.Model(&model.TaxRawRow{}).Where("batch_id IN ?", uncoveredRawBatches).Where("BTRIM(COALESCE(data->>'Transaction Type', '')) <> ?", "PaymentSummary")
		q = excludeWalmartDuplicateRows(q, "tax_raw_rows")
		if err := q.Find(&raw).Error; err != nil {
			return nil, nil, err
		}
	}
	if len(raw) > 0 {
		var rules []model.TaxFeeRule
		if err := db.Find(&rules).Error; err != nil {
			return nil, nil, err
		}
		validRules := len(rules) > 0
		for _, rule := range rules {
			if rule.MatchKey == "" {
				validRules = false
				break
			}
		}
		if !validRules {
			var ruleErr error
			rules, ruleErr = model.DefaultTaxFeeRules()
			if ruleErr != nil {
				return nil, nil, ruleErr
			}
		}
		byBatch := map[uint][]model.TaxRawRow{}
		for _, row := range raw {
			byBatch[row.BatchID] = append(byBatch[row.BatchID], row)
		}
		for batchID, rows := range byBatch {
			var batch model.TaxDataBatch
			if err := db.Select("id", "rates").First(&batch, batchID).Error; err != nil {
				return nil, nil, fmt.Errorf("批次 %d 缺少原始汇率快照: %w", batchID, err)
			}
			processed, _, _, err := processTaxRows(rows, rules, batch.Rates)
			if err != nil {
				return nil, nil, fmt.Errorf("批次 %d 无法恢复季度数据: %w", batchID, err)
			}
			for _, row := range processed {
				add(row.Quarter, row.Month, row.FeeType, row.Fee, row.Amount, row.ShipQty, row.Rate, row.Currency, 1)
			}
		}
	}
	categories := append([]string{}, taxCategories...)
	result := []taxQuarterSummary{}
	hasUnmatched := false
	for _, row := range grouped {
		for category, total := range row.Totals {
			row.Totals[category] = total.Round(2)
		}
		row.Total = row.Total.Round(2)
		row.TaxableIncome = row.TaxableIncome.Round(2)
		row.USDTotal = row.USDTotal.Round(2)
		row.USDTaxableIncome = row.USDTaxableIncome.Round(2)
		for category, total := range row.USDTotals {
			row.USDTotals[category] = total.Round(2)
		}
		result = append(result, *row)
		if _, ok := row.Totals["未匹配"]; ok {
			hasUnmatched = true
		}
	}
	if hasUnmatched {
		categories = append(categories, "未匹配")
	}
	if period == "month" {
		sort.Slice(result, func(i, j int) bool { return result[i].Month < result[j].Month })
	} else {
		sort.Slice(result, func(i, j int) bool { return result[i].Quarter < result[j].Quarter })
	}
	return result, categories, nil
}
