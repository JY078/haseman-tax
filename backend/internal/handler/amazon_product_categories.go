package handler

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type amazonSalesCategory struct {
	Category string          `json:"category"`
	Quantity decimal.Decimal `json:"quantity"`
}

func ListAmazonProductCategories(c *gin.Context) {
	rows := []model.AmazonProductCategory{}
	if err := database.DB.Order("id").Find(&rows).Error; err != nil {
		taxError(c, err)
		return
	}
	c.JSON(200, gin.H{"items": rows})
}

func SaveAmazonProductCategory(c *gin.Context) {
	var row model.AmazonProductCategory
	if err := c.ShouldBindJSON(&row); err != nil {
		c.JSON(400, gin.H{"error": "请填写匹配关键词和商品分类"})
		return
	}
	row.Keyword, row.Category = strings.TrimSpace(row.Keyword), strings.TrimSpace(row.Category)
	if row.Keyword == "" || row.Category == "" || utf8.RuneCountInString(row.Keyword) > 200 || utf8.RuneCountInString(row.Category) > 100 {
		c.JSON(400, gin.H{"error": "关键词和分类不能为空，关键词最多200字，分类最多100字"})
		return
	}
	row.ID = 0
	if value := c.Param("id"); value != "" {
		id, err := strconv.ParseUint(value, 10, 32)
		if err != nil || id == 0 {
			c.JSON(400, gin.H{"error": "规则编号无效"})
			return
		}
		row.ID = uint(id)
	}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Serialize edits so case-insensitive keywords cannot be added twice.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext('amazon-product-categories'))").Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&model.AmazonProductCategory{}).Where("LOWER(keyword) = LOWER(?) AND id <> ?", row.Keyword, row.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("匹配关键词已存在")
		}
		if row.ID == 0 {
			return tx.Create(&row).Error
		}
		result := tx.Model(&model.AmazonProductCategory{}).Where("id = ?", row.ID).Updates(map[string]interface{}{"keyword": row.Keyword, "category": row.Category})
		if result.Error == nil && result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return result.Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "规则不存在"})
		} else if err.Error() == "匹配关键词已存在" {
			c.JSON(409, gin.H{"error": err.Error()})
		} else {
			taxError(c, err)
		}
		return
	}
	c.JSON(200, row)
}

func DeleteAmazonProductCategory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(400, gin.H{"error": "规则编号无效"})
		return
	}
	result := database.DB.Delete(&model.AmazonProductCategory{}, uint(id))
	if result.Error != nil {
		taxError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(404, gin.H{"error": "规则不存在"})
		return
	}
	c.JSON(200, gin.H{"status": true})
}

func amazonSalesCategories(db *gorm.DB, store, site, siteCurrency string) (map[string][]amazonSalesCategory, map[string][]amazonSalesCategory, error) {
	var rows []struct {
		Quarter  string
		Month    string
		Category string
		Quantity decimal.Decimal
	}
	// Literal, case-insensitive substring matching. A longer keyword wins,
	// then the oldest rule; each shipment contributes to exactly one category.
	// Shipment files are store-level data shared by every site: a selected
	// site claims its rows through the 货币 column, not the site_id column.
	conditions := []string{}
	if strings.TrimSpace(store) != "" {
		conditions = append(conditions, "(s.store_id = '"+strings.ReplaceAll(strings.TrimSpace(store), "'", "''")+"' OR s.store_id = '')")
	}
	if strings.TrimSpace(site) != "" && strings.TrimSpace(siteCurrency) != "" {
		conditions = append(conditions, "COALESCE(NULLIF(UPPER(BTRIM(s.data->>'货币')), ''), 'USD') = '"+strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(siteCurrency)), "'", "''")+"'")
	}
	siteFilter := "TRUE"
	if len(conditions) > 0 {
		siteFilter = strings.Join(conditions, " AND ")
	}
	err := db.Raw(`SELECT s.quarter, s.month, COALESCE(rule.category, '未分类') AS category,
 SUM(` + amazonAmountSQL("已发货数量") + `) AS quantity
 FROM amazon_tax_shipments s
 LEFT JOIN LATERAL (
   SELECT category FROM amazon_product_categories
   WHERE keyword <> '' AND STRPOS(LOWER(COALESCE(s.data->>'商品名称', '')), LOWER(keyword)) > 0
   ORDER BY LENGTH(keyword) DESC, id LIMIT 1
 ) rule ON TRUE
  WHERE ` + siteFilter + `
 GROUP BY s.quarter, s.month, COALESCE(rule.category, '未分类')
 ORDER BY s.quarter, s.month, category`).Scan(&rows).Error
	byQuarter := map[string][]amazonSalesCategory{}
	byMonth := map[string][]amazonSalesCategory{}
	quarterIndex := map[string]map[string]int{}
	for _, row := range rows {
		byMonth[row.Month] = append(byMonth[row.Month], amazonSalesCategory{Category: row.Category, Quantity: row.Quantity})
		if quarterIndex[row.Quarter] == nil {
			quarterIndex[row.Quarter] = map[string]int{}
		}
		index, ok := quarterIndex[row.Quarter][row.Category]
		if !ok {
			byQuarter[row.Quarter] = append(byQuarter[row.Quarter], amazonSalesCategory{Category: row.Category})
			index = len(byQuarter[row.Quarter]) - 1
			quarterIndex[row.Quarter][row.Category] = index
		}
		byQuarter[row.Quarter][index].Quantity = byQuarter[row.Quarter][index].Quantity.Add(row.Quantity)
	}
	return byQuarter, byMonth, err
}
