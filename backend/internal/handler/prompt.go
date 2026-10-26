package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetPromptGroups retrieves all prompt groups with their approved prompts (public)
func GetPromptGroups(c *gin.Context) {
	var groups []model.PromptGroup
	if err := database.DB.Order("sort asc, id asc").Preload("Prompts", func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", model.PromptStatusApproved).Order("sort asc, id asc")
	}).Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prompt groups"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

// CreatePromptGroup creates a new prompt group
func CreatePromptGroup(c *gin.Context) {
	var input model.PromptGroup
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Group name is required"})
		return
	}

	var maxSort int
	if err := database.DB.Model(&model.PromptGroup{}).Select("COALESCE(MAX(sort), -1)").Scan(&maxSort).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to allocate sort order"})
		return
	}
	input.Sort = maxSort + 1

	if err := database.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create prompt group"})
		return
	}

	c.JSON(http.StatusCreated, input)
}

// UpdatePromptGroup updates an existing prompt group
func UpdatePromptGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var existingGroup model.PromptGroup
	if err := database.DB.First(&existingGroup, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prompt group not found"})
		return
	}

	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Sort        *int   `json:"sort"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name != "" {
		existingGroup.Name = input.Name
	}
	existingGroup.Description = input.Description
	if input.Sort != nil {
		existingGroup.Sort = *input.Sort
	}

	if err := database.DB.Save(&existingGroup).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update prompt group"})
		return
	}

	c.JSON(http.StatusOK, existingGroup)
}

// ReorderPromptGroups sets sort order from the given ID list (index 0 = smallest sort).
func ReorderPromptGroups(c *gin.Context) {
	var body struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids is required"})
		return
	}

	tx := database.DB.Begin()
	for i, id := range body.IDs {
		if err := tx.Model(&model.PromptGroup{}).Where("id = ?", id).Update("sort", i).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sort order"})
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit sort order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// DeletePromptGroup deletes a prompt group and its prompts
func DeletePromptGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	// Delete associated prompts first
	if err := database.DB.Where("group_id = ?", id).Delete(&model.Prompt{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete associated prompts"})
		return
	}

	if err := database.DB.Delete(&model.PromptGroup{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete prompt group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Prompt group deleted successfully"})
}

// GetPrompts retrieves approved prompts (public), optionally filtered by group_id and source.
// group_id=0 returns ungrouped prompts; source=user returns user-submitted prompts.
func GetPrompts(c *gin.Context) {
	var prompts []model.Prompt
	query := database.DB.Where("status = ?", model.PromptStatusApproved).Order("sort asc, id asc")

	if groupID := c.Query("group_id"); groupID != "" {
		query = query.Where("group_id = ?", groupID)
	}
	if source := c.Query("source"); source != "" {
		query = query.Where("source = ?", source)
	}
	if shape := c.Query("shape"); shape != "" {
		query = query.Where("shape = ?", shape)
	}

	if err := query.Find(&prompts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prompts"})
		return
	}

	c.JSON(http.StatusOK, prompts)
}

// GetAdminPrompts retrieves all prompts for admin (with optional status/group filter)
func GetAdminPrompts(c *gin.Context) {
	var prompts []model.Prompt
	query := database.DB.Order("sort asc, id asc")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if groupID := c.Query("group_id"); groupID != "" {
		query = query.Where("group_id = ?", groupID)
	}
	if shape := c.Query("shape"); shape != "" {
		query = query.Where("shape = ?", shape)
	}

	if err := query.Find(&prompts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prompts"})
		return
	}

	c.JSON(http.StatusOK, prompts)
}

// UpdatePromptStatus updates the status of a prompt (approve / offline)
func UpdatePromptStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var input struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	allowed := map[string]bool{
		model.PromptStatusApproved: true,
		model.PromptStatusPending:  true,
		model.PromptStatusOffline:  true,
	}
	if !allowed[input.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value"})
		return
	}

	result := database.DB.Model(&model.Prompt{}).Where("id = ?", id).Update("status", input.Status)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update prompt status"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prompt not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated"})
}

// CreatePrompt creates a new prompt; group_id=0 means ungrouped
func CreatePrompt(c *gin.Context) {
	var input model.Prompt
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Shape == "" {
		input.Shape = "square"
	}
	if input.Title == "" {
		input.Title = input.Shape + "-template"
	}
	if input.Content == "" {
		input.Content = input.Title
	}

	// If a specific group is given, verify it exists
	if input.GroupID != 0 {
		var group model.PromptGroup
		if err := database.DB.First(&group, input.GroupID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Prompt group not found"})
			return
		}
	}

	// Admin-created prompts are approved by default
	if input.Status == "" {
		input.Status = model.PromptStatusApproved
	}
	if input.Source == "" {
		input.Source = model.PromptSourceAdmin
	}
	if input.Images == nil {
		input.Images = model.StringArray{}
	}

	var maxSort sql.NullInt64
	if err := database.DB.Model(&model.Prompt{}).Where("group_id = ?", input.GroupID).Select("MAX(sort)").Scan(&maxSort).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to allocate sort order"})
		return
	}
	if maxSort.Valid {
		input.Sort = int(maxSort.Int64) + 1
	} else {
		input.Sort = 0
	}

	if err := database.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, input)
}

// UpdatePrompt updates an existing prompt
func UpdatePrompt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var existingPrompt model.Prompt
	if err := database.DB.First(&existingPrompt, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prompt not found"})
		return
	}

	var input struct {
		GroupID uint              `json:"group_id"`
		Shape   string            `json:"shape"`
		Title   string            `json:"title"`
		Content string            `json:"content"`
		Images  model.StringArray `json:"images"`
		Sort    *int              `json:"sort"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Title != "" {
		existingPrompt.Title = input.Title
	} else if existingPrompt.Title == "" {
		existingPrompt.Title = existingPrompt.Shape + "-template"
	}
	if input.Content != "" {
		existingPrompt.Content = input.Content
	} else if existingPrompt.Content == "" {
		existingPrompt.Content = existingPrompt.Title
	}
	if input.Shape != "" {
		existingPrompt.Shape = input.Shape
	}
	// Allow moving to a different group or to ungrouped (group_id=0 is valid)
	if input.GroupID != existingPrompt.GroupID {
		if input.GroupID != 0 {
			var group model.PromptGroup
			if err := database.DB.First(&group, input.GroupID).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Target prompt group not found"})
				return
			}
		}
		existingPrompt.GroupID = input.GroupID
	}
	if input.Sort != nil {
		existingPrompt.Sort = *input.Sort
	}
	if input.Images != nil {
		existingPrompt.Images = input.Images
	}

	if err := database.DB.Save(&existingPrompt).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update prompt"})
		return
	}

	c.JSON(http.StatusOK, existingPrompt)
}

// ReorderPrompts sets sort order for prompts that share the same group_id (ids order → sort 0..n-1).
func ReorderPrompts(c *gin.Context) {
	var body struct {
		IDs []uint `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids is required"})
		return
	}

	var first model.Prompt
	if err := database.DB.First(&first, body.IDs[0]).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prompt id"})
		return
	}
	gid := first.GroupID

	for _, id := range body.IDs[1:] {
		var p model.Prompt
		if err := database.DB.First(&p, id).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prompt id"})
			return
		}
		if p.GroupID != gid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "All prompts must belong to the same group"})
			return
		}
	}

	tx := database.DB.Begin()
	for i, id := range body.IDs {
		if err := tx.Model(&model.Prompt{}).Where("id = ?", id).Update("sort", i).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sort order"})
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit sort order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// DeletePrompt deletes a prompt
func DeletePrompt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := database.DB.Delete(&model.Prompt{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete prompt"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Prompt deleted successfully"})
}
