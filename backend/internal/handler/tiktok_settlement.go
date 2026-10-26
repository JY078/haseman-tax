package handler

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
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

type tikTokSettlementColumn struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Name  string `json:"name,omitempty"`
}

// Display names may differ from canonical storage keys used by imports and summaries.
var tikTokSettlementDisplayColumns = []tikTokSettlementColumn{
	{Key: "Statement Date", Label: "声明日期"},
	{Key: "Statement ID", Label: "结算单ID"},
	{Key: "Currency", Label: "币种"},
	{Key: "Type", Label: "类型"},
	{Key: "Order/adjustment ID", Label: "订单/调整ID"},
	{Key: "SKU ID", Label: "SKU ID"},
	{Key: "Quantity", Label: "数量"},
	{Key: "Product name", Label: "产品名称"},
	{Key: "SKU name", Label: "SKU名称"},
	{Key: "Total settlement amount", Name: "Total Settlement amount", Label: "总结算金额"},
	{Key: "Total Revenue", Name: "Total revenue", Label: "总收入"},
	{Key: "Customer payment", Label: "客户付款"},
	{Key: "Subtotal after seller discounts", Label: "卖家折扣后小计"},
	{Key: "Customer shipping fee", Label: "客户运费"},
	{Key: "Platform discount", Label: "平台折扣"},
	{Key: "Sales tax payment", Label: "销售税付款"},
	{Key: "Retail delivery fee payment", Label: "零售配送费付款"},
	{Key: "Shipping fee subsidy", Label: "运费补贴"},
	{Key: "Shipping fee discount", Label: "运费折扣"},
	{Key: "Customer order refund", Label: "客户订单退款"},
	{Key: "Refund to customer", Label: "退还客户金额"},
	{Key: "Refund subsidy by platform", Label: "平台退款补贴"},
	{Key: "Platform discount refund", Label: "平台折扣退款"},
	{Key: "Total Fees", Name: "Total fees", Label: "总费用"},
	{Key: "Transaction fee", Label: "交易服务费"},
	{Key: "Referral fee", Label: "平台佣金"},
	{Key: "Refund administration fee", Label: "平台佣金退款"},
	{Key: "Shipping fee", Label: "运费"},
	{Key: "Actual shipping fee", Label: "实际运费"},
	{Key: "Platform shipping fee discount", Label: "平台运费折扣"},
	{Key: "Actual return shipping fee", Label: "实际退货运费"},
	{Key: "Affiliate Commission", Name: "Affiliate commission", Label: "联盟佣金"},
	{Key: "Affiliate partner commission", Label: "联盟伙伴佣金"},
	{Key: "Affiliate Shop Ads commission", Label: "联盟店广告佣金"},
	{Key: "TikTok Shop Partner commission", Label: "TikTok Shop合作伙伴佣金"},
	{Key: "Sales tax", Label: "销售税"},
	{Key: "Sales tax payment (2)", Name: "Sales tax payment", Label: "销售税付款"},
	{Key: "Sales tax refund", Label: "销售税退款"},
	{Key: "Retail delivery fee", Label: "零售配送费"},
	{Key: "Retail delivery fee payment (2)", Name: "Retail delivery fee payment", Label: "零售配送费付款"},
	{Key: "Retail delivery fee refund", Label: "零售配送费退款"},
	{Key: "Customs duty", Label: "关税"},
	{Key: "Co-funded promotion service fee", Label: "联合出资推广服务费"},
	{Key: "Ajustment amount", Label: "调整金额"},
	{Key: "Related order ID", Label: "关联订单ID"},
	{Key: "Ads referral fee discount", Label: "广告佣金折扣"},
	{Key: "FBT fulfillment fee", Name: "FBT Fulfillment fee", Label: "FBT履约费"},
}

//go:embed tiktok_settlement_columns.json
var tikTokSettlementColumnJSON []byte
var tikTokSettlementColumns = func() []tikTokSettlementColumn {
	var columns []tikTokSettlementColumn
	if err := json.Unmarshal(tikTokSettlementColumnJSON, &columns); err != nil {
		panic(err)
	}
	return columns
}()

func UploadTikTokSettlements(c *gin.Context) {
	var audit *model.TaxImportLog
	respond := func(code int, ok bool, msg string) { respondTikTokImport(c, audit, code, ok, msg) }
	if !taxUploadAuthorized(c) {
		respond(401, false, "RPA 密钥无效")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, taxMaxFile+(1<<20))
	file, header, err := c.Request.FormFile("file")
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	if err != nil {
		respond(400, false, "缺少 file 文件或文件超过 20MB")
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, taxMaxFile+1))
	if err != nil || len(content) > taxMaxFile {
		respond(400, false, "文件读取失败或超过 20MB")
		return
	}
	store := strings.TrimSpace(c.PostForm("store_id"))
	if store == "" {
		store = "Forious"
	}
	site := taxSiteFromForm(c)
	audit, err = beginTikTokImport("settlements", header.Filename, store, site, taxImporterLabel(c), content)
	if err != nil {
		respond(500, false, err.Error())
		return
	}
	rows, err := parseTikTokSettlementFile(content, filepath.Ext(header.Filename))
	if err != nil {
		respond(400, false, err.Error())
		return
	}
	audit.SourceRows = len(rows)
	batch := model.TikTokSettlementBatch{StoreID: store, SiteID: site, FileHash: fmt.Sprintf("%x", sha256.Sum256(content))}
	duplicate := false
	imported := 0
	repaired := 0
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Serialize settlement imports across instances, including differently named files.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended('tiktok_settlement_import', 0))").Error; err != nil {
			return err
		}
		result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "file_hash"}}, DoNothing: true}).Create(&batch)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			duplicate = true
			if err := tx.Where("file_hash = ?", batch.FileHash).First(&batch).Error; err != nil {
				return err
			}
			var err error
			repaired, err = repairTikTokSettlementMapping(tx, batch.ID, rows)
			if err != nil {
				return err
			}
			for i := range rows {
				rows[i].StoreID = store
				rows[i].SiteID = site
				rows[i].BatchID = batch.ID
			}
			imported, err = insertTikTokSettlements(tx, rows)
			if err != nil {
				return err
			}
			return finishTikTokImport(tx, audit, batch.ID, imported, len(rows)-imported, "duplicate")
		}
		for i := range rows {
			rows[i].StoreID = store
			rows[i].SiteID = site
			rows[i].BatchID = batch.ID
		}
		var err error
		imported, err = insertTikTokSettlements(tx, rows)
		if err != nil {
			return err
		}
		return finishTikTokImport(tx, audit, batch.ID, imported, len(rows)-imported, "locked")
	})
	if err != nil {
		respond(500, false, "结算单导入失败，本次未入库："+err.Error())
		return
	}
	if duplicate {
		if imported > 0 {
			respond(200, true, fmt.Sprintf("已补充入库 %d 条，修正 %d 条字段映射，跳过重复 %d 条", imported, repaired, len(rows)-imported))
			return
		}
		if repaired > 0 {
			respond(200, true, fmt.Sprintf("文件已导入，已修正 %d 条字段映射，未新增重复数据", repaired))
			return
		}
		respond(200, true, "文件已导入，已跳过重复文件")
		return
	}
	respond(201, true, fmt.Sprintf("结算单导入成功，入库 %d 条，跳过重复 %d 条", imported, len(rows)-imported))
}

// Only repair rows owned by this exact file hash; other files retain first-import wins.
func repairTikTokSettlementMapping(tx *gorm.DB, batchID uint, rows []model.TikTokSettlement) (int, error) {
	var existing []model.TikTokSettlement
	if err := tx.Where("batch_id = ?", batchID).Find(&existing).Error; err != nil {
		return 0, err
	}
	keyOf := func(row model.TikTokSettlement) [4]string {
		return [4]string{row.StatementDate, strings.TrimSpace(row.Data["Statement ID"]), strings.TrimSpace(row.Data["Order/adjustment ID"]), strings.TrimSpace(row.Data["SKU ID"])}
	}
	byKey := make(map[[4]string][]model.TikTokSettlement, len(rows))
	for _, row := range rows {
		key := keyOf(row)
		byKey[key] = append(byKey[key], row)
	}
	repaired := 0
	for _, old := range existing {
		candidates := byKey[keyOf(old)]
		if len(candidates) == 0 {
			continue
		}
		row := candidates[0]
		found := len(candidates) == 1
		for _, candidate := range candidates {
			if old.Amount.Equal(candidate.Amount) {
				row, found = candidate, true
				break
			}
		}
		if !found || (old.Amount.Equal(row.Amount) && old.Currency == row.Currency && old.Month == row.Month && old.Quarter == row.Quarter && reflect.DeepEqual(old.Data, row.Data)) {
			continue
		}
		if err := tx.Model(&model.TikTokSettlement{}).Where("id = ? AND batch_id = ?", old.ID, batchID).
			Select("Data", "Amount", "Currency", "Month", "Quarter").Updates(&row).Error; err != nil {
			return 0, err
		}
		repaired++
	}
	return repaired, nil
}

// Called inside the import transaction while holding the settlement import lock.
// Keep the first row for each date/statement/order/SKU/amount identity.
func insertTikTokSettlements(tx *gorm.DB, rows []model.TikTokSettlement) (int, error) {
	type identity struct {
		Date, StatementID, OrderID string
		SKUID                      string `gorm:"column:sku_id"`
		Amount                     string
	}
	keyOf := func(row model.TikTokSettlement) identity {
		return identity{row.StatementDate, strings.TrimSpace(row.Data["Statement ID"]), strings.TrimSpace(row.Data["Order/adjustment ID"]), strings.TrimSpace(row.Data["SKU ID"]), row.Amount.String()}
	}
	seen := make(map[identity]bool, len(rows))
	unique := make([]model.TikTokSettlement, 0, len(rows))
	for _, row := range rows {
		key := keyOf(row)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, row)
		}
	}
	imported := 0
	for start := 0; start < len(unique); start += 300 {
		end := start + 300
		if end > len(unique) {
			end = len(unique)
		}
		chunk := unique[start:end]
		keys := make([][]interface{}, 0, len(chunk))
		for _, row := range chunk {
			key := keyOf(row)
			keys = append(keys, []interface{}{key.Date, key.StatementID, key.OrderID, key.SKUID, key.Amount})
		}
		var existing []identity
		if err := tx.Model(&model.TikTokSettlement{}).
			Select("statement_date AS date, COALESCE(BTRIM(data->>'Statement ID'), '') AS statement_id, COALESCE(BTRIM(data->>'Order/adjustment ID'), '') AS order_id, COALESCE(BTRIM(data->>'SKU ID'), '') AS sku_id, amount::text AS amount").
			Where("(statement_date, COALESCE(BTRIM(data->>'Statement ID'), ''), COALESCE(BTRIM(data->>'Order/adjustment ID'), ''), COALESCE(BTRIM(data->>'SKU ID'), ''), amount) IN ?", keys).
			Scan(&existing).Error; err != nil {
			return 0, err
		}
		found := make(map[identity]bool, len(existing))
		for _, key := range existing {
			amount, err := decimal.NewFromString(key.Amount)
			if err != nil {
				return 0, err
			}
			key.Amount = amount.String()
			found[key] = true
		}
		pending := make([]model.TikTokSettlement, 0, len(chunk))
		for _, row := range chunk {
			if !found[keyOf(row)] {
				pending = append(pending, row)
			}
		}
		if len(pending) > 0 {
			if err := tx.Create(&pending).Error; err != nil {
				return 0, err
			}
			imported += len(pending)
		}
	}
	return imported, nil
}

// Read every worksheet whose name contains "Order detail". Numeric Excel dates
// use the workbook's date system. CSV files are treated as a single worksheet.
func readTikTokSettlementWorksheets(content []byte, ext string) ([][]string, bool, bool, error) {
	var rows [][]string
	date1904, excelDates := false, false
	appendRow := func(row []string) error {
		if len(rows) >= 100001 {
			return fmt.Errorf("最多支持 100000 条数据")
		}
		rows = append(rows, row)
		return nil
	}
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
				return nil, false, false, fmt.Errorf("CSV 读取失败：%w", err)
			}
			if err = appendRow(row); err != nil {
				return nil, false, false, err
			}
		}
	case ".xlsx":
		book, err := excelize.OpenReader(bytes.NewReader(content), excelize.Options{UnzipSizeLimit: taxMaxXLSXUnzip, UnzipXMLSizeLimit: taxMaxXLSXXML})
		if err != nil {
			return nil, false, false, err
		}
		defer book.Close()
		props, err := book.GetWorkbookProps()
		if err != nil {
			return nil, false, false, err
		}
		date1904 = props.Date1904 != nil && *props.Date1904
		excelDates = true
		matched := 0
		for i := 0; i < book.SheetCount; i++ {
			name := book.GetSheetName(i)
			if !strings.Contains(strings.ToLower(strings.TrimSpace(name)), "order detail") {
				continue
			}
			iter, err := book.Rows(name)
			if err != nil {
				return nil, false, false, err
			}
			var sheetRows [][]string
			for iter.Next() {
				row, err := iter.Columns(excelize.Options{RawCellValue: true})
				if err != nil {
					iter.Close()
					return nil, false, false, err
				}
				sheetRows = append(sheetRows, row)
			}
			if err := iter.Error(); err != nil {
				iter.Close()
				return nil, false, false, err
			}
			iter.Close()
			if len(sheetRows) == 0 {
				continue
			}
			if err := mergeTikTokSettlementSheet(&rows, sheetRows); err != nil {
				return nil, false, false, fmt.Errorf("工作表 %s：%w", name, err)
			}
			matched++
		}
		if matched == 0 {
			return nil, false, false, fmt.Errorf("文件中没有名称包含 Order detail 的工作表")
		}
	case ".xls":
		book, err := xls.OpenReader(bytes.NewReader(content), "utf-8")
		if err != nil {
			return nil, false, false, err
		}
		if book.NumSheets() == 0 {
			return nil, false, false, fmt.Errorf("文件没有工作表")
		}
		matched := 0
		for i := 0; i < book.NumSheets(); i++ {
			sheet := book.GetSheet(i)
			if sheet == nil || !strings.Contains(strings.ToLower(strings.TrimSpace(sheet.Name)), "order detail") {
				continue
			}
			if sheet.MaxRow == 0 {
				continue
			}
			sheetRows := book.ReadAllCells(int(sheet.MaxRow) + 1)
			if err := mergeTikTokSettlementSheet(&rows, sheetRows); err != nil {
				return nil, false, false, fmt.Errorf("工作表 %s：%w", sheet.Name, err)
			}
			matched++
		}
		if matched == 0 {
			return nil, false, false, fmt.Errorf("文件中没有名称包含 Order detail 的工作表")
		}

	default:
		return nil, false, false, fmt.Errorf("仅支持 CSV、XLSX、XLS 文件")
	}
	if strings.EqualFold(ext, ".csv") && len(rows) > 0 {
		var mapped [][]string
		if err := mergeTikTokSettlementSheet(&mapped, rows); err != nil {
			return nil, false, false, err
		}
		rows = mapped
	}
	return rows, date1904, excelDates, nil
}

// Merge by each worksheet's own headers, never by another sheet's positions.
// Repeated headers keep separate occurrence keys so distinct source values survive.
func mergeTikTokSettlementSheet(merged *[][]string, sheetRows [][]string) error {
	if len(sheetRows) == 0 {
		return nil
	}
	normalize := func(header string) string {
		return strings.ToLower(strings.Join(strings.Fields(strings.TrimPrefix(header, "\ufeff")), " "))
	}
	known := make(map[string]string, len(tikTokSettlementColumns))
	for _, column := range tikTokSettlementColumns {
		known[normalize(column.Key)] = column.Key
	}
	for _, column := range tikTokSettlementDisplayColumns {
		known[normalize(column.Key)] = column.Key
	}
	for _, key := range []string{"Shipping protection service fee", "Bill deduction for negative balance", "Sales proceed (negative balance)"} {
		known[normalize(key)] = key
	}
	headers := make([]string, len(sheetRows[0]))
	occurrences := map[string]int{}
	present := map[string]bool{}
	for index, header := range sheetRows[0] {
		header = strings.TrimSpace(strings.TrimPrefix(header, "\ufeff"))
		if header == "" {
			continue
		}
		if key, ok := known[normalize(header)]; ok {
			header = key
		}
		present[header] = true
		occurrences[header]++
		if occurrences[header] > 1 {
			header = fmt.Sprintf("%s (%d)", header, occurrences[header])
		}
		headers[index] = header
	}
	for _, key := range []string{"Statement Date", "Statement ID", "Currency", "Type", "Order/adjustment ID", "Total settlement amount"} {
		if !present[key] {
			return fmt.Errorf("缺少表头：%s", key)
		}
	}
	if len(*merged) == 0 {
		*merged = append(*merged, []string{})
	}
	positions := map[string]int{}
	for index, header := range (*merged)[0] {
		positions[header] = index
	}
	for _, header := range headers {
		if header == "" {
			continue
		}
		if _, ok := positions[header]; !ok {
			positions[header] = len((*merged)[0])
			(*merged)[0] = append((*merged)[0], header)
		}
	}
	for _, values := range sheetRows[1:] {
		if len(*merged) >= 100001 {
			return fmt.Errorf("最多支持 100000 条数据")
		}
		mapped := make([]string, len((*merged)[0]))
		for index, header := range headers {
			if header != "" && index < len(values) {
				mapped[positions[header]] = values[index]
			}
		}
		*merged = append(*merged, mapped)
	}
	return nil
}

// Order files continue to use their first worksheet. Only settlement files
// apply the Order detail worksheet selection rule.
func readTikTokFirstWorksheet(content []byte, ext string) ([][]string, bool, bool, error) {
	var rows [][]string
	appendRow := func(row []string) error {
		if len(rows) >= 100001 {
			return fmt.Errorf("最多支持 100000 条数据")
		}
		rows = append(rows, row)
		return nil
	}
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
				return nil, false, false, fmt.Errorf("CSV 读取失败：%w", err)
			}
			if err := appendRow(row); err != nil {
				return nil, false, false, err
			}
		}
	case ".xlsx":
		book, err := excelize.OpenReader(bytes.NewReader(content), excelize.Options{UnzipSizeLimit: taxMaxXLSXUnzip, UnzipXMLSizeLimit: taxMaxXLSXXML})
		if err != nil {
			return nil, false, false, err
		}
		defer book.Close()
		props, err := book.GetWorkbookProps()
		if err != nil {
			return nil, false, false, err
		}
		date1904 := props.Date1904 != nil && *props.Date1904
		iter, err := book.Rows(book.GetSheetName(0))
		if err != nil {
			return nil, false, false, err
		}
		for iter.Next() {
			row, err := iter.Columns(excelize.Options{RawCellValue: true})
			if err != nil {
				iter.Close()
				return nil, false, false, err
			}
			if err := appendRow(row); err != nil {
				iter.Close()
				return nil, false, false, err
			}
		}
		if err := iter.Error(); err != nil {
			iter.Close()
			return nil, false, false, err
		}
		iter.Close()
		return rows, date1904, true, nil
	case ".xls":
		book, err := xls.OpenReader(bytes.NewReader(content), "utf-8")
		if err != nil {
			return nil, false, false, err
		}
		if book.NumSheets() == 0 || book.GetSheet(0) == nil {
			return nil, false, false, fmt.Errorf("文件没有工作表")
		}
		sheet := book.GetSheet(0)
		rows = book.ReadAllCells(int(sheet.MaxRow) + 1)
	default:
		return nil, false, false, fmt.Errorf("仅支持 CSV、XLSX、XLS 文件")
	}
	return rows, false, false, nil
}

func parseTikTokSettlementFile(content []byte, ext string) ([]model.TikTokSettlement, error) {
	rows, date1904, excelDates, err := readTikTokSettlementWorksheets(content, ext)
	if err != nil {
		return nil, err
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("结算单文件没有数据行")
	}
	columns := map[string]int{}
	for i, h := range rows[0] {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if _, ok := columns[h]; ok {
			return nil, fmt.Errorf("表头重复：%s", h)
		}
		columns[h] = i
	}
	for _, key := range []string{"Statement Date", "Statement ID", "Currency", "Type", "Order/adjustment ID", "Total settlement amount"} {
		if _, ok := columns[key]; !ok {
			return nil, fmt.Errorf("缺少表头：%s", key)
		}
	}
	output := make([]model.TikTokSettlement, 0, len(rows)-1)
	for ri, values := range rows[1:] {
		if strings.TrimSpace(strings.Join(values, "")) == "" {
			continue
		}
		data := map[string]string{}
		for _, column := range tikTokSettlementColumns {
			data[column.Key] = ""
		}
		for _, column := range tikTokSettlementDisplayColumns {
			data[column.Key] = ""
		}
		for key, i := range columns {
			if i < len(values) {
				data[key] = strings.TrimSpace(values[i])
			} else {
				data[key] = ""
			}
		}
		// Both export versions name the seller shipping fee differently.
		if data["Seller shipping fee"] == "" {
			data["Seller shipping fee"] = data["Shipping fee"]
		}
		if data["Shipping fee"] == "" {
			data["Shipping fee"] = data["Seller shipping fee"]
		}
		dateValue := data["Statement Date"]
		if excelDates {
			if serial, err := strconv.ParseFloat(dateValue, 64); err == nil && serial >= 1 && serial < 100000 {
				date, err := excelize.ExcelDateToTime(serial, date1904)
				if err != nil {
					return nil, fmt.Errorf("第 %d 行声明日期无效", ri+2)
				}
				dateValue = date.Format("2006-01-02")
			}
		}
		date := parseTikTokPostedAt(dateValue)
		if date.IsZero() {
			return nil, fmt.Errorf("第 %d 行 Statement Date 日期无效：%q", ri+2, dateValue)
		}
		amount, err := parseTaxAmount(data["Total settlement amount"])
		if err != nil {
			return nil, fmt.Errorf("第 %d 行 Total settlement amount 无效：%v", ri+2, err)
		}
		currency := strings.ToUpper(strings.TrimSpace(data["Currency"]))
		data["Statement Date"] = date.Format("2006-01-02")
		output = append(output, model.TikTokSettlement{Month: date.Format("2006-01"), Quarter: fmt.Sprintf("%d-Q%d", date.Year(), (int(date.Month())+2)/3), StatementDate: date.Format("2006-01-02"), Currency: currency, Amount: amount, Data: data})
	}
	if len(output) == 0 {
		return nil, fmt.Errorf("结算单文件没有数据行")
	}
	return output, nil
}

type tikTokSettlementResponse struct {
	model.TikTokSettlement
	Fee *decimal.Decimal `json:"fee"`
}

const tikTokSettlementOrderJoin = `LEFT JOIN LATERAL (SELECT data FROM tik_tok_orders WHERE BTRIM(s.data->>'Type') = 'Order' AND order_id = BTRIM(s.data->>'Order/adjustment ID') AND order_id <> '' ORDER BY id DESC LIMIT 1) o ON TRUE`

const tikTokSettlementLogisticsSQL = `CASE
 WHEN BTRIM(s.data->>'Type') IS DISTINCT FROM 'Order' THEN '杂费'
 WHEN BTRIM(o.data->>'Delivery Option Type') = '商家物流' THEN 'MCF'
 WHEN BTRIM(o.data->>'Delivery Option Type') = 'TikTok 物流' AND BTRIM(o.data->>'Delivery Option') = '标准运输' THEN '海外仓'
 WHEN BTRIM(o.data->>'Delivery Option Type') = 'TikTok 物流' AND BTRIM(o.data->>'Delivery Option') = 'Fulfilled by TikTok Shop 标准' THEN 'FBT'
 ELSE '' END`

func queryTikTokSettlements(db *gorm.DB, month, quarter string) *gorm.DB {
	return filterTikTokTax(db.Model(&model.TikTokSettlement{}).Table("tik_tok_settlements s"), month, quarter).
		Joins(taxRateJoinSQL("fx", "s.statement_date", "s.currency")).
		// A settlement matches one order, even when that order contains multiple SKU rows.
		Joins(tikTokSettlementOrderJoin).
		Select(`s.id, s.batch_id, s.store_id, s.site_id, s.month, s.quarter, s.statement_date, s.currency, s.amount, s.data,
 ` + tikTokSettlementLogisticsSQL + ` AS logistics_method,
 CASE WHEN BTRIM(s.data->>'Type') = 'Order' THEN COALESCE(o.data->>'Normal or Pre-order', '') ELSE 'Other' END AS order_type,
 CASE WHEN s.currency = 'CNY' THEN ROUND(s.amount, 2) ELSE ROUND(s.amount * fx.rate, 2) END AS fee`).Order("s.statement_date, s.id")
}

// Filters use the same computed values as the displayed settlement rows.
func filterTikTokSettlementFields(q *gorm.DB, logistics, orderType string) *gorm.DB {
	for _, filter := range []struct{ field, value string }{{"logistics_method", logistics}, {"order_type", orderType}} {
		if filter.value == "" {
			continue
		}
		values := taxPeriodValues(filter.value)
		for i := range values {
			if values[i] == "__empty__" {
				values[i] = ""
			}
		}
		q = q.Where(filter.field+" IN ?", values)
	}
	return q
}
func filteredTikTokSettlements(db *gorm.DB, month, quarter, logistics, orderType string) *gorm.DB {
	return filterTikTokSettlementFields(db.Table("(?) AS settlement_rows", queryTikTokSettlements(db, month, quarter)), logistics, orderType)
}
func ListTikTokSettlements(c *gin.Context) {
	page, size := taxPage(c)
	var total int64
	q := filteredTikTokSettlements(database.DB, c.Query("month"), c.Query("quarter"), c.Query("logistics_method"), c.Query("order_type"))
	if c.Query("store_id") != "" {
		q = q.Where("(store_id = ? OR store_id = '')", c.Query("store_id"))
	}
	q = filterTaxSite(q, c.Query("site_id"))
	if err := q.Count(&total).Error; err != nil {
		taxError(c, err)
		return
	}
	items := []tikTokSettlementResponse{}
	if err := q.Order("statement_date, id").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		taxError(c, err)
		return
	}
	response := gin.H{"items": items, "total": total, "columns": tikTokSettlementDisplayColumns}
	if wantsTaxTotals(c.Query("include_totals")) {
		allItems := []tikTokSettlementResponse{}
		if err := q.Limit(-1).Offset(-1).Order("statement_date, id").Find(&allItems).Error; err != nil {
			taxError(c, err)
			return
		}
		totals := decimalStructTotals(allItems)
		for _, row := range allItems {
			addJSONTotals(totals, row.Data)
			if row.Fee != nil {
				totals["fee"] = totals["fee"].Add(*row.Fee)
			}
		}
		response["totals"] = totals
	}
	c.JSON(200, response)
}
