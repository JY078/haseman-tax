package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const maxWorkflowProjectStateBytes = 2 * 1024 * 1024

type workflowProjectRequest struct {
	ProjectID string          `json:"project_id"`
	Name      string          `json:"name"`
	State     json.RawMessage `json:"state"`
}

func ListWorkflowProjects(c *gin.Context) {
	userID := currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var projects []model.WorkflowProject
	if err := database.DB.Where("user_id = ?", userID).Order("updated_at DESC").Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items := make([]gin.H, 0, len(projects))
	for _, project := range projects {
		items = append(items, workflowProjectResponse(project))
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func CreateWorkflowProject(c *gin.Context) {
	userID := currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req workflowProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	projectID := strings.TrimSpace(req.ProjectID)
	if projectID == "" {
		projectID = uuid.NewString()
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Untitled project"
	}
	state, err := checkedWorkflowState(req.State)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project := model.WorkflowProject{
		ProjectID: projectID,
		UserID:    userID,
		Name:      name,
		State:     state,
	}
	if err := database.DB.Create(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": workflowProjectResponse(project)})
}

func GetWorkflowProject(c *gin.Context) {
	project, ok := findWorkflowProject(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": workflowProjectResponse(project)})
}

func UpdateWorkflowProject(c *gin.Context) {
	project, ok := findWorkflowProject(c)
	if !ok {
		return
	}

	var req workflowProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = project.Name
	}
	state, err := checkedWorkflowState(req.State)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	project.Name = name
	project.State = state
	if err := database.DB.Save(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": workflowProjectResponse(project)})
}

func DeleteWorkflowProject(c *gin.Context) {
	project, ok := findWorkflowProject(c)
	if !ok {
		return
	}
	if err := database.DB.Delete(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": true})
}

func currentUserID(c *gin.Context) uint {
	userIDVal, exists := c.Get("userID")
	if !exists {
		return 0
	}
	userID, ok := userIDVal.(uint)
	if !ok {
		return 0
	}
	return userID
}

func findWorkflowProject(c *gin.Context) (model.WorkflowProject, bool) {
	userID := currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return model.WorkflowProject{}, false
	}

	projectID := strings.TrimSpace(c.Param("project_id"))
	var project model.WorkflowProject
	err := database.DB.Where("user_id = ? AND project_id = ?", userID, projectID).First(&project).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "project not found"})
		return model.WorkflowProject{}, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return model.WorkflowProject{}, false
	}
	return project, true
}

func checkedWorkflowState(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	if len(raw) > maxWorkflowProjectStateBytes {
		return "", errors.New("project state is too large")
	}

	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", errors.New("project state is not valid JSON")
	}
	if containsLocalImageURL(value) {
		return "", errors.New("project images must be uploaded to OSS before saving")
	}
	return string(raw), nil
}

func containsLocalImageURL(value interface{}) bool {
	switch v := value.(type) {
	case string:
		return strings.HasPrefix(v, "data:") || strings.HasPrefix(v, "blob:")
	case []interface{}:
		for _, item := range v {
			if containsLocalImageURL(item) {
				return true
			}
		}
	case map[string]interface{}:
		for _, item := range v {
			if containsLocalImageURL(item) {
				return true
			}
		}
	}
	return false
}

func workflowProjectResponse(project model.WorkflowProject) gin.H {
	var state interface{}
	if err := json.Unmarshal([]byte(project.State), &state); err != nil {
		state = gin.H{}
	}
	return gin.H{
		"id":         project.ID,
		"project_id": project.ProjectID,
		"name":       project.Name,
		"state":      state,
		"created_at": project.CreatedAt,
		"updated_at": project.UpdatedAt,
	}
}
