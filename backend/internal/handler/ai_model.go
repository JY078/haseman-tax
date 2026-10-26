package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
)

// GetAiModels retrieves all AI models
func GetAiModels(c *gin.Context) {
	var models []model.AiModel
	query := database.DB

	// Optional filter by type
	// Since types is stored as JSON string, we use LIKE for simple filtering
	if typeParam := c.Query("type"); typeParam != "" {
		// This assumes types are stored like ["chat","drawing"]
		// It's a simple text search, might have false positives if type names overlap (e.g. "chat" and "chat-pro")
		// But for defined types "chat", "drawing", "video", it's fine.
		likeQuery := fmt.Sprintf("%%%s%%", typeParam)
		query = query.Where("types LIKE ?", likeQuery)
	}

	if err := query.Order("id asc").Find(&models).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch models"})
		return
	}

	c.JSON(http.StatusOK, models)
}

// CreateAiModel creates a new AI model
func CreateAiModel(c *gin.Context) {
	var input model.AiModel
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate types
	if len(input.Types) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one model type is required"})
		return
	}

	validTypes := map[string]bool{
		"chat":    true,
		"drawing": true,
		"video":   true,
	}
	for _, t := range input.Types {
		if !validTypes[t] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model type: " + t})
			return
		}
	}

	// Handle default model logic
	if input.IsDefault {
		// For each type, unset is_default for other models
		for _, t := range input.Types {
			database.DB.Model(&model.AiModel{}).
				Where("types LIKE ?", "%"+t+"%").
				Update("is_default", false)
		}
	}

	if err := database.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create model"})
		return
	}

	c.JSON(http.StatusCreated, input)
}

// UpdateAiModel updates an existing AI model
func UpdateAiModel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var existingModel model.AiModel
	if err := database.DB.First(&existingModel, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
		return
	}

	// Bind to map only — request body is a stream and can only be read once.
	// Extracting all values from the map avoids a second ShouldBindJSON call
	// which would silently return zero-values for every field.
	var inputMap map[string]interface{}
	if err := c.ShouldBindJSON(&inputMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse types from the map (input.Types cannot be used — see above)
	var newTypes model.StringArray
	if typesRaw, ok := inputMap["types"]; ok {
		if typesSlice, ok := typesRaw.([]interface{}); ok {
			validTypes := map[string]bool{"chat": true, "drawing": true, "video": true}
			for _, t := range typesSlice {
				s, ok := t.(string)
				if !ok {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid type value"})
					return
				}
				if !validTypes[s] {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid model type: " + s})
					return
				}
				newTypes = append(newTypes, s)
			}
			if len(newTypes) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "At least one model type is required"})
				return
			}
		}
	}

	// Parse boolean fields directly from the map to avoid zero-value traps
	var isActiveVal *bool
	if v, ok := inputMap["is_active"]; ok {
		if b, ok := v.(bool); ok {
			isActiveVal = &b
		}
	}
	var isDefaultVal *bool
	if v, ok := inputMap["is_default"]; ok {
		if b, ok := v.(bool); ok {
			isDefaultVal = &b
		}
	}

	// When enabling default, unset is_default on other models of the same type(s) first
	if isDefaultVal != nil && *isDefaultVal {
		typesToCheck := existingModel.Types
		if newTypes != nil {
			typesToCheck = newTypes
		}
		for _, t := range typesToCheck {
			database.DB.Model(&model.AiModel{}).
				Where("id != ? AND types LIKE ?", id, "%"+t+"%").
				UpdateColumn("is_default", false)
		}
	}

	// Build the update map for non-boolean fields.
	updateMap := make(map[string]interface{})
	if name, ok := inputMap["name"].(string); ok && name != "" {
		updateMap["name"] = name
	}
	if newTypes != nil {
		updateMap["types"] = newTypes
	}
	if resRaw, ok := inputMap["supported_resolutions"]; ok {
		if resSlice, ok := resRaw.([]interface{}); ok {
			var resolutions model.StringArray
			for _, r := range resSlice {
				if s, ok := r.(string); ok && s != "" {
					resolutions = append(resolutions, s)
				}
			}
			updateMap["supported_resolutions"] = resolutions
		} else if resRaw == nil {
			updateMap["supported_resolutions"] = model.StringArray{}
		}
	}

	if len(updateMap) > 0 {
		if err := database.DB.Model(&existingModel).Updates(updateMap).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update model"})
			return
		}
	}

	// Boolean fields use UpdateColumn to bypass GORM zero-value logic and hooks,
	// guaranteeing the exact value (true or false) is written to the database.
	if isActiveVal != nil {
		if err := database.DB.Model(&existingModel).UpdateColumn("is_active", *isActiveVal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update is_active"})
			return
		}
	}
	if isDefaultVal != nil {
		if err := database.DB.Model(&existingModel).UpdateColumn("is_default", *isDefaultVal).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update is_default"})
			return
		}
	}

	database.DB.First(&existingModel, id)
	c.JSON(http.StatusOK, existingModel)
}

// DeleteAiModel deletes an AI model
func DeleteAiModel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := database.DB.Delete(&model.AiModel{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete model"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model deleted successfully"})
}
