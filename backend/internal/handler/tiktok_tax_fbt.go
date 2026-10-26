package handler

import (
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

var tikTokFBTHeaders = []string{"Goods ID", "SKU ID", "Product ID", "Product Name", "Shop ID", "Shop Name", "Cost Rate", "GMV", "Total Logistics Cost", "FBT Fulfillment Fee", "FBT Fulfillment Fee Reimbursement", "Shipping Fee Discount", "Hub Placement Fee", "Storage Fee", "Inbound Shipping Fee", "Inbound Incidents Fee", "Inbound Booking Non-Compliance Fee", "Routing Non-Compliance Fee", "Outbound Appointment No-Show Fee", "Delayed Response Fee", "Disposal Fee", "Return to Seller Shipping Fee", "Return to Seller Handling Fee", "Inbound Return Operation Fee"}
var fbtDecimalFields = []string{"Cost Rate", "GMV", "Total Logistics Cost", "FBT Fulfillment Fee", "FBT Fulfillment Fee Reimbursement", "Shipping Fee Discount", "Hub Placement Fee", "Storage Fee", "Inbound Shipping Fee", "Inbound Incidents Fee", "Inbound Booking Non-Compliance Fee", "Routing Non-Compliance Fee", "Outbound Appointment No-Show Fee", "Delayed Response Fee", "Disposal Fee", "Return to Seller Shipping Fee", "Return to Seller Handling Fee", "Inbound Return Operation Fee"}

func UploadTikTokTaxFBT(c *gin.Context) {
	if !taxUploadAuthorized(c) {
		c.JSON(http.StatusUnauthorized, gin.H{"status": false, "msg": "TikTok RPA 密钥无效"})
		return
	}
	file, h, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"status": false, "msg": "缺少 file 文件"})
		return
	}
	defer file.Close()
	content, _ := io.ReadAll(file)
	store := strings.TrimSpace(c.PostForm("store_id"))
	if store == "" {
		store = "Forious"
	}
	site := taxSiteFromForm(c)
	audit, err := beginTikTokImport("fbt", h.Filename, store, site, taxImporterLabel(c), content)
	if err != nil {
		c.JSON(400, gin.H{"status": false, "msg": err.Error()})
		return
	}
	rows, err := parseTikTokFBTFile(content, filepath.Ext(h.Filename))
	if err != nil {
		respondTikTokImport(c, audit, 400, false, err.Error())
		return
	}
	audit.SourceRows = len(rows)
	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	var batch model.TikTokTaxFBTBatch
	if err = database.DB.Where("file_hash = ?", hash).First(&batch).Error; err == nil {
		audit.BatchID = batch.ID
		respondTikTokImport(c, audit, 200, true, "文件已导入")
		return
	}
	if err != gorm.ErrRecordNotFound {
		respondTikTokImport(c, audit, 500, false, "查询导入记录失败")
		return
	}
	month := regexp.MustCompile(`^(\d{4}-\d{2})`).FindString(filepath.Base(h.Filename))
	if month == "" {
		respondTikTokImport(c, audit, 400, false, "文件名必须以 YYYY-MM 开头")
		return
	}
	year := month[:4]
	m := month[5:]
	mi := 0
	fmt.Sscanf(m, "%d", &mi)
	quarter := fmt.Sprintf("%s-Q%d", year, (mi+2)/3)
	batch = model.TikTokTaxFBTBatch{StoreID: store, SiteID: site, FileHash: hash, FileName: h.Filename, FileContent: content, RowCount: len(rows)}
	imported := 0
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if e := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "file_hash"}}, DoNothing: true}).Create(&batch).Error; e != nil {
			return e
		}
		for i := range rows {
			rows[i].StoreID = store
			rows[i].SiteID = site
			rows[i].BatchID = batch.ID
			rows[i].Month = month
			rows[i].Quarter = quarter
		}
		if len(rows) > 0 {
			if e := tx.CreateInBatches(&rows, 500).Error; e != nil {
				return e
			}
		}
		imported = len(rows)
		return finishTikTokImport(tx, audit, batch.ID, imported, 0, "locked")
	})
	if err != nil {
		respondTikTokImport(c, audit, 500, false, "FBT 导入失败，本次未入库："+err.Error())
		return
	}
	respondTikTokImport(c, audit, 201, true, fmt.Sprintf("FBT 导入成功，入库 %d 条", imported))
}
func parseTikTokFBTFile(content []byte, ext string) ([]model.TikTokTaxFBT, error) {
	var rows [][]string
	if strings.EqualFold(ext, ".csv") {
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
	} else if strings.EqualFold(ext, ".xlsx") {
		f, e := excelize.OpenReader(bytes.NewReader(content))
		if e != nil {
			return nil, e
		}
		defer f.Close()
		it, e := f.Rows(f.GetSheetName(0))
		if e != nil {
			return nil, e
		}
		for it.Next() {
			row, e := it.Columns(excelize.Options{RawCellValue: true})
			if e != nil {
				return nil, e
			}
			rows = append(rows, row)
		}
	} else {
		return nil, fmt.Errorf("仅支持 CSV、XLSX 文件")
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("FBT 文件没有数据")
	}
	cols := map[string]int{}
	for i, h := range rows[0] {
		cols[strings.TrimSpace(h)] = i
	}
	for _, h := range tikTokFBTHeaders {
		if _, ok := cols[h]; !ok {
			return nil, fmt.Errorf("缺少表头：%s", h)
		}
	}
	out := []model.TikTokTaxFBT{}
	for ri, row := range rows[1:] {
		if strings.TrimSpace(strings.Join(row, "")) == "" {
			continue
		}
		get := func(h string) string {
			i := cols[h]
			if i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}
		v := model.TikTokTaxFBT{GoodsID: get("Goods ID"), SKUID: get("SKU ID"), ProductID: get("Product ID"), ProductName: get("Product Name"), ShopID: get("Shop ID"), ShopName: get("Shop Name")}
		dst := []*decimal.Decimal{&v.CostRate, &v.GMV, &v.TotalLogisticsCost, &v.FBTFulfillmentFee, &v.FBTFulfillmentFeeReimbursement, &v.ShippingFeeDiscount, &v.HubPlacementFee, &v.StorageFee, &v.InboundShippingFee, &v.InboundIncidentsFee, &v.InboundBookingNonComplianceFee, &v.RoutingNonComplianceFee, &v.OutboundAppointmentNoShowFee, &v.DelayedResponseFee, &v.DisposalFee, &v.ReturnToSellerShippingFee, &v.ReturnToSellerHandlingFee, &v.InboundReturnOperationFee}
		for i, h := range fbtDecimalFields {
			if x := get(h); x != "" {
				d, e := parseFBTDecimal(x)
				if e != nil {
					return nil, fmt.Errorf("第 %d 行 %s 数值无效", ri+2, h)
				}
				*dst[i] = d
			}
		}
		out = append(out, v)
	}
	return out, nil
}

func parseFBTDecimal(value string) (decimal.Decimal, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, ",", ""))
	if value == "" || value == "-" || value == "—" || strings.EqualFold(value, "n/a") {
		return decimal.Zero, nil
	}
	value = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(value, "$"), "¥"), "￥"))
	percent := strings.HasSuffix(value, "%")
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	d, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, err
	}
	if percent {
		d = d.Div(decimal.NewFromInt(100))
	}
	return d, nil
}
func ListTikTokTaxFBT(c *gin.Context) {
	page, size := taxPage(c)
	q := filterTikTokTaxStore(database.DB.Model(&model.TikTokTaxFBT{}), c.Query("month"), c.Query("quarter"), c.Query("store_id"), c.Query("site_id"))
	var total int64
	if e := q.Count(&total).Error; e != nil {
		taxError(c, e)
		return
	}
	var rows []model.TikTokTaxFBT
	if e := q.Order("id").Offset((page - 1) * size).Limit(size).Find(&rows).Error; e != nil {
		taxError(c, e)
		return
	}
	response := gin.H{"items": rows, "total": total}
	if wantsTaxTotals(c.Query("include_totals")) {
		var allRows []model.TikTokTaxFBT
		if err := q.Limit(-1).Offset(-1).Order("id").Find(&allRows).Error; err != nil {
			taxError(c, err)
			return
		}
		response["totals"] = decimalStructTotals(allRows)
	}
	c.JSON(200, response)
}
