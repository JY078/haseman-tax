package handler

import (
	"net/http"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

// GetSettings retrieves all settings
func GetSettings(c *gin.Context) {
	var settings []model.Setting
	if err := database.DB.Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
		return
	}

	// Convert to map for easier frontend consumption
	settingsMap := make(map[string]string)
	for _, s := range settings {
		settingsMap[s.Key] = s.Value
	}

	c.JSON(http.StatusOK, settingsMap)
}

/** CheckMaintenanceMode 公开接口，返回维护模式状态（无需鉴权） */
func CheckMaintenanceMode(c *gin.Context) {
	var settings []model.Setting
	if err := database.DB.Where("\"key\" IN ?", []string{"maintenance_enabled", "maintenance_message"}).Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check maintenance status"})
		return
	}

	result := gin.H{
		"enabled": false,
		"message": "系统维护中，请稍后再试",
	}
	for _, s := range settings {
		switch s.Key {
		case "maintenance_enabled":
			result["enabled"] = s.Value == "true"
		case "maintenance_message":
			if s.Value != "" {
				result["message"] = s.Value
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

func GetBrandSettings(c *gin.Context) {
	keys := []string{"brand_logo_url", "brand_name"}
	var settings []model.Setting
	if err := database.DB.Where("\"key\" IN ?", keys).Find(&settings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch brand settings"})
		return
	}

	result := gin.H{
		"brand_logo_url":  "",
		"brand_name":      "",
		"logo_configured": false,
	}
	for _, s := range settings {
		switch s.Key {
		case "brand_logo_url":
			result["brand_logo_url"] = s.Value
		case "brand_name":
			result["brand_name"] = s.Value
		}
	}
	result["logo_configured"] = strings.TrimSpace(result["brand_logo_url"].(string)) != ""

	c.JSON(http.StatusOK, result)
}

// UpdateSettings updates one or multiple settings
func UpdateSettings(c *gin.Context) {
	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	tx := database.DB.Begin()

	for key, value := range updates {
		setting := model.Setting{
			Key:   key,
			Value: value,
		}

		// Use upsert
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).Create(&setting).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update setting: " + key})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Settings updated successfully"})
}
