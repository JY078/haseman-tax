package handler

import (
	"net/http"
	"strconv"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
)

/** GetAnnouncements 管理端：分页获取所有公告 */
func GetAnnouncements(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	database.DB.Model(&model.Announcement{}).Count(&total)

	var announcements []model.Announcement
	if err := database.DB.Order("id desc").Offset(offset).Limit(pageSize).Find(&announcements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch announcements"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      announcements,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

/** CreateAnnouncement 管理端：创建公告 */
func CreateAnnouncement(c *gin.Context) {
	var input struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content" binding:"required"`
		Status  string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Status == "" {
		input.Status = "draft"
	}

	announcement := model.Announcement{
		Title:   input.Title,
		Content: input.Content,
		Status:  input.Status,
	}

	if err := database.DB.Create(&announcement).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create announcement"})
		return
	}

	c.JSON(http.StatusCreated, announcement)
}

/** UpdateAnnouncement 管理端：更新公告 */
func UpdateAnnouncement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var existing model.Announcement
	if err := database.DB.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Announcement not found"})
		return
	}

	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Status  string `json:"status"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Title != "" {
		existing.Title = input.Title
	}
	if input.Content != "" {
		existing.Content = input.Content
	}
	if input.Status != "" {
		existing.Status = input.Status
	}

	if err := database.DB.Save(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update announcement"})
		return
	}

	c.JSON(http.StatusOK, existing)
}

/** DeleteAnnouncement 管理端：删除公告及其阅读记录 */
func DeleteAnnouncement(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	database.DB.Where("announcement_id = ?", id).Delete(&model.AnnouncementRead{})

	if err := database.DB.Delete(&model.Announcement{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete announcement"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Announcement deleted successfully"})
}

/** GetUnreadAnnouncements 用户端：获取当前用户的未读已发布公告 */
func GetUnreadAnnouncements(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var announcements []model.Announcement
	if err := database.DB.
		Where("status = ?", "published").
		Where("id NOT IN (?)",
			database.DB.Model(&model.AnnouncementRead{}).
				Select("announcement_id").
				Where("user_id = ?", userID),
		).
		Order("id desc").
		Find(&announcements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch unread announcements"})
		return
	}

	c.JSON(http.StatusOK, announcements)
}

/** MarkAnnouncementRead 用户端：标记公告为已读 */
func MarkAnnouncementRead(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	announcementID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid announcement ID"})
		return
	}

	var existing model.AnnouncementRead
	result := database.DB.Where("user_id = ? AND announcement_id = ?", userID, announcementID).First(&existing)
	if result.Error == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Already read"})
		return
	}

	read := model.AnnouncementRead{
		UserID:         userID,
		AnnouncementID: uint(announcementID),
		ReadAt:         time.Now(),
	}

	if err := database.DB.Create(&read).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked as read"})
}
