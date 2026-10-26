package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/extrame/xls"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var tikTokAdHeaders = []string{"日期", "系列名称", "推广系列ID", "现金消耗", "信用额度消耗", "广告费用赠款消耗", "金额", "币种", "类型"}

func UploadTikTokTaxAds(c *gin.Context) {
	var audit *model.TaxImportLog
	respond := func(code int, ok bool, msg string) { respondTikTokImport(c, audit, code, ok, msg) }
	if !taxUploadAuthorized(c) {
		respond(http.StatusUnauthorized, false, "RPA 密钥无效")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, taxMaxFile+(1<<20))
	file, header, err := c.Request.FormFile("file")
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		respond(http.StatusBadRequest, false, "缺少 file 文件或文件超过 20MB")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, taxMaxFile+1))
	if err != nil || len(content) > taxMaxFile {
		respond(http.StatusBadRequest, false, "文件读取失败或超过 20MB")
		return
	}
	store := strings.TrimSpace(c.PostForm("store_id"))
	if store == "" {
		store = "Forious"
	}
	site := taxSiteFromForm(c)
	audit, err = beginTikTokImport("ads", header.Filename, store, site, taxImporterLabel(c), content)
	if err != nil {
		respond(http.StatusBadRequest, false, err.Error())
		return
	}
	ads, err := parseTikTokAdFile(content, filepath.Ext(header.Filename))
	if err != nil {
		respond(http.StatusBadRequest, false, err.Error())
		return
	}
	audit.SourceRows = len(ads)
	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	var existing model.TikTokTaxAdBatch
	err = database.DB.Where("file_hash = ?", hash).First(&existing).Error
	if err == nil {
		audit.BatchID = existing.ID
		respond(http.StatusOK, true, "文件已导入")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		respond(http.StatusInternalServerError, false, "查询导入记录失败")
		return
	}
	dates := make([]string, 0)
	seen := map[string]bool{}
	for _, ad := range ads {
		if !seen[ad.Date] {
			dates = append(dates, taxRateDate(parseTikTokPostedAt(ad.Date)))
			seen[ad.Date] = true
		}
	}
	var rates []model.TaxExchangeRate
	if err := database.DB.Where("date IN ? AND currency = ?", dates, "USD").Find(&rates).Error; err != nil {
		respond(http.StatusInternalServerError, false, "读取汇率失败")
		return
	}
	rateByDate := map[string]decimal.Decimal{}
	for _, rate := range rates {
		rateByDate[rate.Date] = rate.Rate
	}
	for i := range ads {
		ads[i].StoreID = store
		ads[i].SiteID = site
		rate := rateByDate[taxRateDate(parseTikTokPostedAt(ads[i].Date))]
		if !rate.IsPositive() {
			// Preserve USD data now; a zero rate marks a pending conversion.
			continue
		}
		ads[i].Rate = rate
		ads[i].Fee = ads[i].Amount.Mul(rate).Round(2)
	}
	batch := model.TikTokTaxAdBatch{StoreID: store, SiteID: site, FileHash: hash, FileName: header.Filename, FileContent: content, RowCount: len(ads)}
	duplicate := false
	imported := 0
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "tiktok-tax-ad-import").Error; err != nil {
			return err
		}
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "file_hash"}}, DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			duplicate = true
			if err := tx.Where("file_hash = ?", hash).First(&batch).Error; err != nil {
				return err
			}
			return finishTikTokImport(tx, audit, batch.ID, 0, len(ads), "duplicate")
		}
		first, last := ads[0].Date, ads[0].Date
		for _, ad := range ads {
			if ad.Date < first {
				first = ad.Date
			}
			if ad.Date > last {
				last = ad.Date
			}
		}
		var existing []model.TikTokTaxAd
		if err := tx.Select("date, campaign_id").Where("date >= ? AND date <= ?", first, last).Find(&existing).Error; err != nil {
			return err
		}
		type identity struct{ Date, CampaignID string }
		seen := make(map[identity]bool, len(existing)+len(ads))
		for _, ad := range existing {
			seen[identity{ad.Date, strings.TrimSpace(ad.CampaignID)}] = true
		}
		newAds := make([]model.TikTokTaxAd, 0, len(ads))
		for _, ad := range ads {
			key := identity{ad.Date, strings.TrimSpace(ad.CampaignID)}
			if seen[key] {
				continue
			}
			seen[key] = true
			ad.BatchID = batch.ID
			newAds = append(newAds, ad)
		}
		if len(newAds) > 0 {
			if err := tx.CreateInBatches(&newAds, 500).Error; err != nil {
				return err
			}
		}
		imported = len(newAds)
		if err := tx.Model(&batch).Update("row_count", imported).Error; err != nil {
			return err
		}
		return finishTikTokImport(tx, audit, batch.ID, imported, len(ads)-imported, "locked")
	})
	if err != nil {
		respond(http.StatusInternalServerError, false, "广告费导入失败，本次未入库："+err.Error())
		return
	}
	if duplicate {
		respond(http.StatusOK, true, "文件已导入")
		return
	}
	respond(http.StatusCreated, true, fmt.Sprintf("广告费导入成功，入库 %d 条，跳过重复 %d 条", imported, len(ads)-imported))
}

// Ad exports use the first worksheet; the tax detail export still uses the second.
func parseTikTokAdFile(content []byte, ext string) ([]model.TikTokTaxAd, error) {
	var rows [][]string
	switch strings.ToLower(ext) {
	case ".csv":
		reader := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf})))
		reader.FieldsPerRecord = -1
		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, fmt.Errorf("CSV 读取失败：%w", err)
			}
			rows = append(rows, row)
			if len(rows) > 100001 {
				return nil, fmt.Errorf("最多支持 100000 条广告费")
			}
		}
	case ".xlsx":
		book, err := excelize.OpenReader(bytes.NewReader(content), excelize.Options{UnzipSizeLimit: taxMaxXLSXUnzip, UnzipXMLSizeLimit: taxMaxXLSXXML})
		if err != nil {
			return nil, err
		}
		defer book.Close()
		iter, err := book.Rows(book.GetSheetName(0))
		if err != nil {
			return nil, err
		}
		defer iter.Close()
		props, err := book.GetWorkbookProps()
		if err != nil {
			return nil, err
		}
		date1904 := props.Date1904 != nil && *props.Date1904
		dateColumn := -1
		for iter.Next() {
			row, err := iter.Columns(excelize.Options{RawCellValue: true})
			if err != nil {
				return nil, err
			}
			if len(rows) == 0 {
				for i, cell := range row {
					if strings.TrimSpace(cell) == "日期" {
						dateColumn = i
					}
				}
			} else if dateColumn >= 0 && dateColumn < len(row) {
				if serial, err := decimal.NewFromString(row[dateColumn]); err == nil && serial.GreaterThanOrEqual(decimal.NewFromInt(20000)) && serial.LessThan(decimal.NewFromInt(100000)) {
					value, _ := serial.Float64()
					date, err := excelize.ExcelDateToTime(value, date1904)
					if err != nil {
						return nil, err
					}
					row[dateColumn] = date.Format("2006-01-02")
				}
			}
			rows = append(rows, row)
			if len(rows) > 100001 {
				return nil, fmt.Errorf("最多支持 100000 条广告费")
			}
		}
		if err := iter.Error(); err != nil {
			return nil, err
		}
	case ".xls":
		book, err := xls.OpenReader(bytes.NewReader(content), "utf-8")
		if err != nil {
			return nil, fmt.Errorf("XLS 读取失败：%w", err)
		}
		if book.NumSheets() == 0 {
			return nil, fmt.Errorf("文件没有工作表")
		}
		rows = book.ReadAllCells(int(book.GetSheet(0).MaxRow) + 1)
	default:
		return nil, fmt.Errorf("仅支持 CSV、XLSX、XLS 文件")
	}
	return parseTikTokAdRows(rows)
}

func parseTikTokAdRows(rows [][]string) ([]model.TikTokTaxAd, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("广告费文件为空")
	}
	columns := map[string]int{}
	for i, header := range rows[0] {
		columns[strings.TrimSpace(header)] = i
	}
	for _, name := range tikTokAdHeaders {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("缺少表头：%s", name)
		}
	}
	lastDataRow := len(rows) - 1
	for lastDataRow > 0 && strings.TrimSpace(strings.Join(rows[lastDataRow], "")) == "" {
		lastDataRow--
	}
	ads := make([]model.TikTokTaxAd, 0, len(rows)-1)
	for index, row := range rows[1:] {
		blank := true
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				blank = false
				break
			}
		}
		if blank {
			continue
		}
		get := func(name string) string {
			i := columns[name]
			if i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}
		posted := parseTikTokPostedAt(get("日期"))
		if posted.IsZero() && index+1 == lastDataRow {
			totalRow := false
			for _, cell := range row {
				if strings.TrimSpace(cell) == "总计" {
					totalRow = true
					break
				}
			}
			if totalRow {
				continue
			}
		}
		if posted.IsZero() {
			return nil, fmt.Errorf("第 %d 行日期无效：%q", index+2, get("日期"))
		}
		currency := strings.ToUpper(get("币种"))
		if currency == "" {
			currency = "USD"
		}
		if currency != "USD" {
			return nil, fmt.Errorf("第 %d 行币种必须为 USD", index+2)
		}
		amounts := make([]decimal.Decimal, 4)
		for i, field := range []string{"现金消耗", "信用额度消耗", "广告费用赠款消耗", "金额"} {
			value, err := parseTaxAmount(strings.TrimSpace(strings.ReplaceAll(get(field), "$", "")))
			if err != nil {
				return nil, fmt.Errorf("第 %d 行%s无效", index+2, field)
			}
			amounts[i] = value
		}
		ads = append(ads, model.TikTokTaxAd{Month: posted.Format("2006-01"), Quarter: fmt.Sprintf("%d-Q%d", posted.Year(), (int(posted.Month())-1)/3+1), Date: posted.Format("2006-01-02"), CampaignName: get("系列名称"), CampaignID: get("推广系列ID"), CashSpend: amounts[0], CreditSpend: amounts[1], GrantSpend: amounts[2], Amount: amounts[3], Currency: currency, Type: get("类型")})
	}
	if len(ads) == 0 {
		return nil, fmt.Errorf("广告费文件没有数据行")
	}
	return ads, nil
}

func ListTikTokTaxAds(c *gin.Context) {
	page, size := taxPage(c)
	var total int64
	if err := filterTikTokTaxStore(database.DB.Model(&model.TikTokTaxAd{}), c.Query("month"), c.Query("quarter"), c.Query("store_id"), c.Query("site_id")).Count(&total).Error; err != nil {
		taxError(c, err)
		return
	}
	var rows []model.TikTokTaxAd
	// Use the latest stored daily rate, retaining the import snapshot if unavailable.
	adsQuery := queryTikTokAds(database.DB, c.Query("month"), c.Query("quarter"))
	if c.Query("store_id") != "" {
		adsQuery = adsQuery.Where("(a.store_id = ? OR a.store_id = '')", c.Query("store_id"))
	}
	adsQuery = filterTaxSite(adsQuery, c.Query("site_id"))
	err := adsQuery.Offset((page - 1) * size).Limit(size).Scan(&rows).Error
	if err != nil {
		taxError(c, err)
		return
	}
	type adResponse struct {
		model.TikTokTaxAd
		Fee *decimal.Decimal `json:"fee"`
	}
	items := make([]adResponse, 0, len(rows))
	for _, row := range rows {
		item := adResponse{TikTokTaxAd: row}
		if row.Rate.IsPositive() {
			value := row.Fee
			item.Fee = &value
		}
		items = append(items, item)
	}
	response := gin.H{"items": items, "total": total}
	if wantsTaxTotals(c.Query("include_totals")) {
		var allRows []model.TikTokTaxAd
		allAdsQuery := queryTikTokAds(database.DB, c.Query("month"), c.Query("quarter"))
		if c.Query("store_id") != "" {
			allAdsQuery = allAdsQuery.Where("(a.store_id = ? OR a.store_id = '')", c.Query("store_id"))
		}
		allAdsQuery = filterTaxSite(allAdsQuery, c.Query("site_id"))
		if err := allAdsQuery.Find(&allRows).Error; err != nil {
			taxError(c, err)
			return
		}
		response["totals"] = decimalStructTotals(allRows)
	}
	c.JSON(http.StatusOK, response)
}
