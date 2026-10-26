package handler

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetUsageLogs retrieves usage logs with optional filters.
// Query params: type (chat/drawing), page, page_size
func GetUsageLogs(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	isAdmin := false
	var user model.User
	if err := database.DB.Preload("Roles").First(&user, userID).Error; err == nil {
		for _, role := range user.Roles {
			if role.Name == "admin" {
				isAdmin = true
				break
			}
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	db := database.DB.Model(&model.UsageLog{}).
		Where("usage_logs.status != ? AND usage_logs.usage_type != ?", "retry", "download")
	if !isAdmin {
		db = db.Where("usage_logs.user_id = ?", userID)
	}
	if usageType := c.Query("type"); usageType != "" {
		db = db.Where("usage_logs.usage_type = ?", usageType)
	}
	if source := c.Query("source"); source != "" {
		db = db.Where("usage_logs.source = ?", source)
	}
	if userName := c.Query("user_name"); userName != "" && isAdmin {
		db = db.Joins("JOIN users ON users.id = usage_logs.user_id").
			Where("users.name LIKE ? OR users.username LIKE ?", "%"+userName+"%", "%"+userName+"%")
	}
	if c.Query("image_only") == "1" {
		db = db.Where("usage_logs.image_url != '' AND usage_logs.status = 'success'")
		if c.Query("saved_only") == "1" {
			db = db.Where("usage_logs.is_saved_to_design = ?", true)
		}
	}

	var total int64
	db.Session(&gorm.Session{}).Count(&total)

	var logs []model.UsageLog
	if err := db.
		Select("usage_logs.id", "usage_logs.user_id", "usage_logs.usage_type", "usage_logs.source", "usage_logs.model", "usage_logs.prompt", "usage_logs.result", "usage_logs.image_url", "usage_logs.reference_image_url", "usage_logs.reference_image_urls", "usage_logs.prompt_tokens", "usage_logs.completion_tokens", "usage_logs.total_tokens", "usage_logs.resolution", "usage_logs.status", "usage_logs.duration", "usage_logs.download_count", "usage_logs.is_saved_to_design", "usage_logs.saved_at", "usage_logs.created_at", "usage_logs.updated_at").
		Order("usage_logs.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch usage logs: " + err.Error()})
		return
	}

	if len(logs) > 0 {
		userIDs := make([]uint, len(logs))
		for i, l := range logs {
			userIDs[i] = l.UserID
		}
		var users []model.User
		database.DB.Where("id IN ?", userIDs).Find(&users)
		userMap := make(map[uint]string, len(users))
		for _, u := range users {
			if u.Name != "" {
				userMap[u.ID] = u.Name
			} else {
				userMap[u.ID] = u.Username
			}
		}
		for i := range logs {
			if name, ok := userMap[logs[i].UserID]; ok {
				logs[i].UserName = name
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

type imageUserStatsRow struct {
	UserID         uint    `gorm:"column:user_id" json:"user_id"`
	Username       string  `gorm:"column:username" json:"username"`
	Name           string  `gorm:"column:name" json:"name"`
	GeneratedCount int64   `gorm:"column:generated_count" json:"generated_count"`
	DownloadCount  int64   `gorm:"column:download_count" json:"download_count"`
	DownloadRate   float64 `gorm:"column:download_rate" json:"download_rate"`
}

type imageStatsSummary struct {
	GeneratedCount int64   `json:"generated_count"`
	DownloadCount  int64   `json:"download_count"`
	DownloadRate   float64 `json:"download_rate"`
}

type imageStatsTrendRow struct {
	Period         string  `gorm:"column:period" json:"period"`
	GeneratedCount int64   `gorm:"column:generated_count" json:"generated_count"`
	DownloadCount  int64   `gorm:"column:download_count" json:"download_count"`
	DownloadRate   float64 `gorm:"column:download_rate" json:"download_rate"`
}

type imageStatsFilterUser struct {
	ID         uint   `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Department string `json:"department"`
}

var imageStatsPrimaryDepartments = []string{"美国1部", "美国2部", "美国3部", "欧洲1部", "设计部", "多平台"}
var imageStatsDepartments = append(append([]string{}, imageStatsPrimaryDepartments...), "其他")

func imageStatsDepartmentCondition(department string) (string, interface{}) {
	if department == "其他" {
		return " AND COALESCE(users.department, '') NOT IN ?", imageStatsPrimaryDepartments
	}
	return " AND users.department = ?", department
}

func queryImageStatsTrend(userID uint, department string, startDate, endDate time.Time) ([]imageStatsTrendRow, string, error) {
	periodExpr, periodFormat, granularity := "DATE_TRUNC('month', usage_logs.created_at)", "YYYY-MM", "month"
	if !startDate.IsZero() {
		periodExpr, periodFormat, granularity = "DATE_TRUNC('day', usage_logs.created_at)", "YYYY-MM-DD", "day"
	}
	query := "SELECT TO_CHAR(" + periodExpr + ", ?) AS period, COUNT(*) AS generated_count, COALESCE(SUM(download_count), 0) AS download_count, CASE WHEN COUNT(*) > 0 THEN COALESCE(SUM(download_count), 0)::float / COUNT(*) ELSE 0 END AS download_rate FROM usage_logs LEFT JOIN users ON users.id = usage_logs.user_id WHERE usage_logs.image_url != '' AND usage_logs.status = 'success' AND usage_logs.usage_type NOT IN ?"
	args := []interface{}{periodFormat, []string{"retry", "download"}}
	if !startDate.IsZero() {
		query += " AND usage_logs.created_at >= ? AND usage_logs.created_at < ?"
		args = append(args, startDate, endDate)
	}
	if userID > 0 {
		query += " AND usage_logs.user_id = ?"
		args = append(args, userID)
	}
	if department != "" {
		condition, conditionArg := imageStatsDepartmentCondition(department)
		query += condition
		args = append(args, conditionArg)
	}
	query += " GROUP BY " + periodExpr + " ORDER BY " + periodExpr + " ASC"
	var trend []imageStatsTrendRow
	if err := database.DB.Raw(query, args...).Scan(&trend).Error; err != nil {
		return nil, "", err
	}
	return trend, granularity, nil
}

func imageStatsMonthRange(month string) (time.Time, time.Time, error) {
	if month == "" {
		return time.Time{}, time.Time{}, nil
	}
	start, err := time.ParseInLocation("2006-01", month, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, start.AddDate(0, 1, 0), nil
}

// GetImageUsageUserStats returns per-user image generation/download aggregates.
func GetImageUsageUserStats(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID := userIDVal.(uint)
	var user model.User
	if err := database.DB.Preload("Roles").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	isAdmin := false
	for _, role := range user.Roles {
		if role.Name == "admin" {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	unpaged := c.Query("unpaged") == "1"
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	selectedUserID, _ := strconv.ParseUint(strings.TrimSpace(c.Query("user_id")), 10, 64)
	month := strings.TrimSpace(c.Query("month"))
	startDate, endDate, err := imageStatsMonthRange(month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "month must use YYYY-MM format"})
		return
	}
	sortField := strings.TrimSpace(c.DefaultQuery("sort_field", "generated_count"))
	sortDirection := strings.ToLower(strings.TrimSpace(c.DefaultQuery("sort_direction", "desc")))
	if sortDirection == "ascend" {
		sortDirection = "asc"
	} else if sortDirection == "descend" {
		sortDirection = "desc"
	}
	sortClause := "generated_count DESC, download_count DESC, usage_logs.user_id ASC"
	if sortField == "generated_count" {
		if sortDirection == "asc" {
			sortClause = "generated_count ASC, download_count DESC, usage_logs.user_id ASC"
		}
	} else if sortField == "download_count" {
		if sortDirection == "asc" {
			sortClause = "download_count ASC, generated_count DESC, usage_logs.user_id ASC"
		} else {
			sortClause = "download_count DESC, generated_count DESC, usage_logs.user_id ASC"
		}
	} else if sortField == "download_rate" {
		if sortDirection == "asc" {
			sortClause = "download_rate ASC, generated_count DESC, usage_logs.user_id ASC"
		} else {
			sortClause = "download_rate DESC, generated_count DESC, usage_logs.user_id ASC"
		}
	}

	base := database.DB.Table("usage_logs").
		Select(`
			usage_logs.user_id,
			COALESCE(users.username, '') AS username,
			COALESCE(users.name, '') AS name,
			COUNT(*) AS generated_count,
			COALESCE(SUM(download_count), 0) AS download_count,
			CASE
				WHEN COUNT(*) > 0 THEN COALESCE(SUM(download_count), 0)::float / COUNT(*)
				ELSE 0
			END AS download_rate
		`).
		Joins("LEFT JOIN users ON users.id = usage_logs.user_id").
		Where("usage_logs.image_url != '' AND usage_logs.status = 'success'").
		Where("usage_logs.usage_type NOT IN ?", []string{"retry", "download"})

	if !startDate.IsZero() {
		base = base.Where("usage_logs.created_at >= ? AND usage_logs.created_at < ?", startDate, endDate)
	}

	if selectedUserID > 0 {
		base = base.Where("usage_logs.user_id = ?", uint(selectedUserID))
	}

	var rows []imageUserStatsRow
	if err := base.
		Group("usage_logs.user_id, users.name, users.username").
		Order(sortClause).
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query image usage stats"})
		return
	}

	for i := range rows {
		if rows[i].GeneratedCount > 0 && rows[i].DownloadRate == 0 {
			rows[i].DownloadRate = float64(rows[i].DownloadCount) / float64(rows[i].GeneratedCount)
		}
	}

	summary := imageStatsSummary{}
	for _, row := range rows {
		summary.GeneratedCount += row.GeneratedCount
		summary.DownloadCount += row.DownloadCount
	}
	if summary.GeneratedCount > 0 {
		summary.DownloadRate = float64(summary.DownloadCount) / float64(summary.GeneratedCount)
	}

	trend, trendGranularity, err := queryImageStatsTrend(uint(selectedUserID), "", startDate, endDate)
	if err != nil {
		log.Printf("[image-stats] trend query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query image usage trend"})
		return
	}
	departmentTrends := make(map[string][]imageStatsTrendRow, len(imageStatsDepartments))
	for _, department := range imageStatsDepartments {
		departmentTrend, _, queryErr := queryImageStatsTrend(0, department, startDate, endDate)
		if queryErr != nil {
			log.Printf("[image-stats] department trend query failed department=%s: %v", department, queryErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query department image usage trend"})
			return
		}
		departmentTrends[department] = departmentTrend
	}
	var filterUsers []imageStatsFilterUser
	if err := database.DB.Model(&model.User{}).
		Select("id", "username", "name", "department").
		Order("name ASC, username ASC, id ASC").
		Scan(&filterUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query image stats users"})
		return
	}

	total := len(rows)
	start := offset
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	responseRows := rows[start:end]
	if unpaged {
		responseRows = rows
		page = 1
		pageSize = total
	}

	c.JSON(http.StatusOK, gin.H{
		"data":              responseRows,
		"summary":           summary,
		"month":             month,
		"total":             total,
		"page":              page,
		"page_size":         pageSize,
		"trend":             trend,
		"trend_granularity": trendGranularity,
		"department_trends": departmentTrends,
		"filter_users":      filterUsers,
	})
}

func SaveUsageLogToDesign(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的记录ID"})
		return
	}

	var logItem model.UsageLog
	if err := database.DB.Where("id = ? AND user_id = ? AND image_url != '' AND status = 'success'", id, userID).First(&logItem).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询记录失败"})
		return
	}

	if logItem.IsSavedToDesign {
		c.JSON(http.StatusOK, gin.H{"message": "已保存到我的设计", "saved_at": logItem.SavedAt, "image_url": logItem.ImageURL})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{"is_saved_to_design": true, "saved_at": now}
	if logItem.ImageURL != "" && !strings.HasPrefix(logItem.Result, "[") {
		if _, err := url.ParseRequestURI(logItem.Result); err == nil {
			updates["result"] = logItem.ImageURL
		}
	}
	if err := database.DB.Model(&model.UsageLog{}).
		Where("id = ? AND user_id = ?", logItem.ID, userID).
		Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "已保存到我的设计",
		"saved_at":  now,
		"image_url": logItem.ImageURL,
	})
}
