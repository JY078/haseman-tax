package handler

import (
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
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
)

//go:embed tiktok_order_columns.json
var tikTokOrderColumnJSON []byte
var tikTokOrderColumns = func() []tikTokSettlementColumn {
	var columns []tikTokSettlementColumn
	if err := json.Unmarshal(tikTokOrderColumnJSON, &columns); err != nil {
		panic(err)
	}
	return columns
}()

func UploadTikTokOrders(c *gin.Context) {
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
	audit, err = beginTikTokImport("orders", header.Filename, store, site, taxImporterLabel(c), content)
	if err != nil {
		respond(500, false, err.Error())
		return
	}
	rows, err := parseTikTokOrderFile(content, filepath.Ext(header.Filename))
	if err != nil {
		respond(400, false, err.Error())
		return
	}
	audit.SourceRows = len(rows)
	batch := model.TikTokOrderBatch{StoreID: store, SiteID: site, FileHash: fmt.Sprintf("%x", sha256.Sum256(content))}
	duplicate := false
	imported := 0
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		// Keep order-ID deduplication atomic when two RPA uploads arrive together.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended('tiktok_order_import', 0))").Error; err != nil {
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
			return finishTikTokImport(tx, audit, batch.ID, 0, len(rows), "duplicate")
		}
		for i := range rows {
			rows[i].StoreID = store
			rows[i].SiteID = site
			rows[i].BatchID = batch.ID
		}
		var err error
		imported, err = insertTikTokOrders(tx, rows)
		if err != nil {
			return err
		}
		return finishTikTokImport(tx, audit, batch.ID, imported, len(rows)-imported, "locked")
	})
	if err != nil {
		respond(500, false, "订单导入失败，本次未入库："+err.Error())
		return
	}
	if duplicate {
		respond(200, true, "文件已导入，已跳过重复文件")
		return
	}
	respond(201, true, fmt.Sprintf("订单导入成功，入库 %d 条，跳过重复 %d 条", imported, len(rows)-imported))
}

// Keep the first row for each Order ID + SKU ID pair. Empty IDs cannot identify
// a stable pair, so those rows remain importable and are not deduplicated.
func insertTikTokOrders(tx *gorm.DB, rows []model.TikTokOrder) (int, error) {
	type identity struct{ OrderID, SKUID string }
	keyOf := func(row model.TikTokOrder) identity {
		return identity{strings.TrimSpace(row.OrderID), strings.TrimSpace(row.Data["SKU ID"])}
	}
	seen := make(map[identity]bool, len(rows))
	unique := make([]model.TikTokOrder, 0, len(rows))
	for _, row := range rows {
		key := keyOf(row)
		if key.OrderID == "" || key.SKUID == "" || !seen[key] {
			if key.OrderID != "" && key.SKUID != "" {
				seen[key] = true
			}
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
		ids := make([]string, 0, len(chunk))
		for _, row := range chunk {
			if key := strings.TrimSpace(row.OrderID); key != "" {
				ids = append(ids, key)
			}
		}
		var existing []struct {
			OrderID string
			SKUID   string `gorm:"column:sku_id"`
		}
		if len(ids) > 0 {
			if err := tx.Model(&model.TikTokOrder{}).
				Select("order_id, BTRIM(COALESCE(data->>'SKU ID', '')) AS sku_id").
				Where("order_id IN ?", ids).Scan(&existing).Error; err != nil {
				return 0, err
			}
		}
		found := make(map[identity]bool, len(existing))
		for _, item := range existing {
			found[identity{strings.TrimSpace(item.OrderID), strings.TrimSpace(item.SKUID)}] = true
		}
		pending := make([]model.TikTokOrder, 0, len(chunk))
		for _, row := range chunk {
			key := keyOf(row)
			if key.OrderID != "" && key.SKUID != "" && found[key] {
				continue
			}
			pending = append(pending, row)
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

func parseTikTokOrderFile(content []byte, ext string) ([]model.TikTokOrder, error) {
	rows, date1904, excelDates, err := readTikTokFirstWorksheet(content, ext)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("订单文件没有数据行")
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
	for _, key := range []string{"Order ID", "Created Time", "Normal or Pre-order", "Delivery Option Type", "Delivery Option"} {
		if _, ok := columns[key]; !ok {
			return nil, fmt.Errorf("缺少表头：%s", key)
		}
	}
	output := make([]model.TikTokOrder, 0, len(rows)-1)
	dateKeys := []string{"Created Time", "Paid Time", "RTS Time", "Shipped Time", "Delivered Time", "Cancelled Time"}
	for ri, values := range rows[1:] {
		if strings.TrimSpace(strings.Join(values, "")) == "" {
			continue
		}
		data := map[string]string{}
		for key, i := range columns {
			if i < len(values) {
				data[key] = strings.TrimSpace(values[i])
			} else {
				data[key] = ""
			}
		}
		if excelDates {
			for _, key := range dateKeys {
				if serial, err := strconv.ParseFloat(data[key], 64); err == nil && serial >= 1 && serial < 100000 {
					date, err := excelize.ExcelDateToTime(serial, date1904)
					if err != nil {
						return nil, fmt.Errorf("第 %d 行 %s 时间无效", ri+2, key)
					}
					data[key] = date.Format("2006-01-02 15:04:05")
				}
			}
		}
		date := parseTikTokPostedAt(data["Created Time"])
		if date.IsZero() {
			return nil, fmt.Errorf("第 %d 行 Created Time 创建时间无效：%q", ri+2, data["Created Time"])
		}
		output = append(output, model.TikTokOrder{OrderID: data["Order ID"], Month: date.Format("2006-01"), Quarter: fmt.Sprintf("%d-Q%d", date.Year(), (int(date.Month())+2)/3), Data: data})
	}
	if len(output) == 0 {
		return nil, fmt.Errorf("订单文件没有数据行")
	}
	return output, nil
}
func ListTikTokOrders(c *gin.Context) {
	page, size := taxPage(c)
	var total int64
	q := filterTikTokTaxStore(database.DB.Model(&model.TikTokOrder{}), c.Query("month"), c.Query("quarter"), c.Query("store_id"), c.Query("site_id"))
	if err := q.Count(&total).Error; err != nil {
		taxError(c, err)
		return
	}
	rows := []model.TikTokOrder{}
	if err := q.Order("id").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		taxError(c, err)
		return
	}
	response := gin.H{"items": rows, "total": total, "columns": tikTokOrderColumns}
	if wantsTaxTotals(c.Query("include_totals")) {
		allRows := []model.TikTokOrder{}
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
