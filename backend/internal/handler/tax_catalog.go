package handler

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

func ListTaxCatalog(c *gin.Context) {
	var entries []model.TaxCatalogEntry
	if err := database.DB.Order("id").Find(&entries).Error; err != nil {
		c.JSON(500, gin.H{"error": "加载店铺和站点失败"})
		return
	}
	c.JSON(200, gin.H{"entries": entries})
}

func CreateTaxCatalogEntry(c *gin.Context) {
	var entry model.TaxCatalogEntry
	if c.ShouldBindJSON(&entry) != nil {
		c.JSON(400, gin.H{"error": "请求格式错误"})
		return
	}
	entry.ID = 0
	entry.Value = strings.TrimSpace(entry.Value)
	entry.Label = strings.TrimSpace(entry.Label)
	if entry.Kind == "site" {
		entry.Value = strings.ToUpper(entry.Value)
		entry.Currency = strings.ToUpper(strings.TrimSpace(entry.Currency))
		if !regexp.MustCompile(`^[A-Z]{3}$`).MatchString(entry.Currency) {
			c.JSON(400, gin.H{"error": "请填写三位币种代码，例如 USD、CAD、EUR"})
			return
		}
		if entry.Value == "" {
			entry.Value = "SITE_" + uuid.NewString()[:12]
		}
	} else {
		entry.Currency = ""
	}
	limit := 100
	if entry.Kind == "site" {
		limit = 20
	}
	if (entry.Platform != "walmart" && entry.Platform != "tiktok" && entry.Platform != "amazon") ||
		(entry.Kind != "store" && entry.Kind != "site") || entry.Value == "" || entry.Label == "" ||
		utf8.RuneCountInString(entry.Value) > limit || utf8.RuneCountInString(entry.Label) > 100 {
		c.JSON(400, gin.H{"error": "请填写有效的平台、名称和代码"})
		return
	}
	result := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&entry)
	if result.Error != nil {
		c.JSON(500, gin.H{"error": "保存失败，请重试"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(409, gin.H{"error": "当前平台已存在相同代码"})
		return
	}
	c.JSON(201, entry)
}
