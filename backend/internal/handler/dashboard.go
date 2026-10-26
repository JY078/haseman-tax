package handler

import (
	"net/http"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
)

// GetDashboardStats returns chart data for the dashboard.
func GetDashboardStats(c *gin.Context) {
	now := time.Now()
	days := 30
	start := now.AddDate(0, 0, -days+1).Truncate(24 * time.Hour)
	imageStart := time.Date(2026, 5, 20, 0, 0, 0, 0, time.Local)
	imageLogCondition := "image_url != '' AND status = 'success'"

	type DailyCount struct {
		Day   string `gorm:"column:day"`
		Count int64  `gorm:"column:count"`
	}

	var dailyChat, dailyDrawing, dailyImageGenerated []DailyCount
	database.DB.Model(&model.UsageLog{}).
		Select("TO_CHAR(created_at, 'MM-DD') as day, COUNT(*) as count").
		Where("usage_type = ? AND created_at >= ?", "chat", start).
		Group("day").
		Order("day asc").
		Scan(&dailyChat)
	database.DB.Model(&model.UsageLog{}).
		Select("TO_CHAR(created_at, 'MM-DD') as day, COUNT(*) as count").
		Where("usage_type = ? AND created_at >= ?", "drawing", start).
		Group("day").
		Order("day asc").
		Scan(&dailyDrawing)
	database.DB.Model(&model.UsageLog{}).
		Select("TO_CHAR(created_at, 'MM-DD') as day, COUNT(*) as count").
		Where(imageLogCondition+" AND created_at >= ?", start).
		Group("day").
		Order("day asc").
		Scan(&dailyImageGenerated)

	var dailyUsers []DailyCount
	database.DB.Model(&model.User{}).
		Select("TO_CHAR(created_at, 'MM-DD') as day, COUNT(*) as count").
		Where("created_at >= ?", start).
		Group("day").
		Order("day asc").
		Scan(&dailyUsers)

	type ModelRank struct {
		Model string `gorm:"column:model" json:"model"`
		Count int64  `gorm:"column:count" json:"count"`
	}
	var modelRank []ModelRank
	database.DB.Model(&model.UsageLog{}).
		Select("model, COUNT(*) as count").
		Group("model").
		Order("count desc").
		Limit(10).
		Scan(&modelRank)

	type ImageModelRank struct {
		Model string `gorm:"column:model" json:"model"`
		Count int64  `gorm:"column:count" json:"count"`
	}
	var imageModelRank []ImageModelRank
	database.DB.Model(&model.UsageLog{}).
		Select("model, COUNT(*) as count").
		Where(imageLogCondition+" AND created_at >= ?", imageStart).
		Group("model").
		Order("count desc").
		Limit(10).
		Scan(&imageModelRank)

	type UserRank struct {
		UserID   uint   `gorm:"column:user_id" json:"user_id"`
		Username string `gorm:"column:username" json:"username"`
		Count    int64  `gorm:"column:count" json:"count"`
	}
	var userRank []UserRank
	database.DB.Model(&model.UsageLog{}).
		Select("usage_logs.user_id, users.username, COUNT(*) as count").
		Joins("LEFT JOIN users ON users.id = usage_logs.user_id").
		Group("usage_logs.user_id, users.username").
		Order("count desc").
		Limit(10).
		Scan(&userRank)

	type ImageUserRank struct {
		UserID         uint   `gorm:"column:user_id" json:"user_id"`
		Username       string `gorm:"column:username" json:"username"`
		Name           string `gorm:"column:name" json:"name"`
		GeneratedCount int64  `gorm:"column:generated_count" json:"generated_count"`
		DownloadCount  int64  `gorm:"column:download_count" json:"download_count"`
	}
	var imageUserRank []ImageUserRank
	statsSubQuery := database.DB.Table("usage_logs").
		Select("user_id, COUNT(*) AS generated_count, COALESCE(SUM(download_count), 0) AS download_count").
		Where("image_url != '' AND status = 'success'").
		Group("user_id")
	database.DB.Table("users AS u").
		Select(`
			u.id AS user_id,
			u.username,
			u.name,
			COALESCE(stats.generated_count, 0) AS generated_count,
			COALESCE(stats.download_count, 0) AS download_count
		`).
		Joins("LEFT JOIN (?) AS stats ON stats.user_id = u.id", statsSubQuery).
		Order("generated_count DESC, download_count DESC, u.id ASC").
		Limit(10).
		Scan(&imageUserRank)

	labels := make([]string, days)
	for i := 0; i < days; i++ {
		labels[i] = start.AddDate(0, 0, i).Format("01-02")
	}

	fillSeries := func(data []DailyCount) []int64 {
		m := make(map[string]int64, len(data))
		for _, d := range data {
			m[d.Day] = d.Count
		}
		series := make([]int64, days)
		for i, label := range labels {
			series[i] = m[label]
		}
		return series
	}

	var totalUsers, totalChat, totalDrawing int64
	var totalImageGenerated, todayImageGenerated int64
	var totalImageDownloads int64
	var todayUsers, todayChat, todayDrawing int64
	var modelCount int64

	database.DB.Model(&model.User{}).Count(&totalUsers)
	database.DB.Model(&model.UsageLog{}).Where("usage_type = ?", "chat").Count(&totalChat)
	database.DB.Model(&model.UsageLog{}).Where("usage_type = ?", "drawing").Count(&totalDrawing)

	today := time.Now().Truncate(24 * time.Hour)
	database.DB.Model(&model.User{}).Where("created_at >= ?", today).Count(&todayUsers)
	database.DB.Model(&model.UsageLog{}).Where("usage_type = ? AND created_at >= ?", "chat", today).Count(&todayChat)
	database.DB.Model(&model.UsageLog{}).Where("usage_type = ? AND created_at >= ?", "drawing", today).Count(&todayDrawing)
	database.DB.Model(&model.UsageLog{}).Where(imageLogCondition+" AND created_at >= ?", imageStart).Count(&totalImageGenerated)
	database.DB.Model(&model.UsageLog{}).Where(imageLogCondition+" AND created_at >= ?", today).Count(&todayImageGenerated)

	type DownloadAgg struct {
		Total int64 `gorm:"column:total"`
	}
	var downloadAgg DownloadAgg
	database.DB.Model(&model.UsageLog{}).
		Select("COALESCE(SUM(download_count), 0) AS total").
		Where(imageLogCondition+" AND created_at >= ?", imageStart).
		Scan(&downloadAgg)
	totalImageDownloads = downloadAgg.Total

	database.DB.Model(&model.AiModel{}).Where("is_active = ?", true).Count(&modelCount)

	c.JSON(http.StatusOK, gin.H{
		"labels":                labels,
		"daily_chat":            fillSeries(dailyChat),
		"daily_drawing":         fillSeries(dailyDrawing),
		"daily_image_generated": fillSeries(dailyImageGenerated),
		"daily_users":           fillSeries(dailyUsers),
		"model_rank":            modelRank,
		"image_model_rank":      imageModelRank,
		"user_rank":             userRank,
		"image_user_rank":       imageUserRank,
		"total_users":           totalUsers,
		"total_chat":            totalChat,
		"total_drawing":         totalDrawing,
		"total_image_generated": totalImageGenerated,
		"total_image_downloads": totalImageDownloads,
		"today_users":           todayUsers,
		"today_chat":            todayChat,
		"today_drawing":         todayDrawing,
		"today_image_generated": todayImageGenerated,
		"model_count":           modelCount,
	})
}
