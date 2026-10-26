package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type workflowTemplateRequest struct {
	Name    string          `json:"name"`
	Summary string          `json:"summary"`
	Scope   string          `json:"scope"`
	State   json.RawMessage `json:"state"`
}

func ListWorkflowTemplates(c *gin.Context) {
	userID := currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var items []model.WorkflowTemplate
	if err := database.DB.Where("scope = ? OR (scope = ? AND user_id = ?)", "system", "user", userID).Order("scope ASC, updated_at DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	data := make([]gin.H, 0, len(items))
	for _, item := range items {
		data = append(data, workflowTemplateResponse(item))
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func CreateWorkflowTemplate(c *gin.Context) {
	userID := currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req workflowTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	if scope != "user" && scope != "system" {
		scope = "user"
	}
	if scope == "system" && !workflowTemplateAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "only administrators can create system templates"})
		return
	}
	state, err := checkedWorkflowState(req.State)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item := model.WorkflowTemplate{UserID: userID, Name: name, Summary: strings.TrimSpace(req.Summary), Scope: scope, State: state}
	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": workflowTemplateResponse(item)})
}

func DeleteWorkflowTemplate(c *gin.Context) {
	userID := currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var item model.WorkflowTemplate
	err := database.DB.Where("id = ? AND (scope = ? AND user_id = ? OR scope = ? AND ?)", c.Param("template_id"), "user", userID, "system", workflowTemplateAdmin(c)).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := database.DB.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": true})
}

func workflowTemplateAdmin(c *gin.Context) bool {
	var user model.User
	if err := database.DB.Preload("Roles").First(&user, currentUserID(c)).Error; err != nil {
		return false
	}
	for _, role := range user.Roles {
		if role.Name == "admin" {
			return true
		}
	}
	return false
}

func workflowTemplateResponse(item model.WorkflowTemplate) gin.H {
	var state interface{}
	if err := json.Unmarshal([]byte(item.State), &state); err != nil {
		state = gin.H{}
	}
	return gin.H{"id": item.ID, "name": item.Name, "summary": item.Summary, "scope": item.Scope, "state": state, "created_at": item.CreatedAt, "updated_at": item.UpdatedAt}
}
