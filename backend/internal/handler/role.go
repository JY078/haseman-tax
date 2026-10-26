package handler

import (
	"net/http"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
)

// GetRoles returns all roles with their menu associations
func GetRoles(c *gin.Context) {
	var roles []model.Role
	if err := database.DB.Preload("Menus").Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch roles"})
		return
	}
	c.JSON(http.StatusOK, roles)
}

// GetAllMenus returns all menus (for role permission assignment UI)
func GetAllMenus(c *gin.Context) {
	var menus []model.Menu
	if err := database.DB.Order("sort asc").Find(&menus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch menus"})
		return
	}
	c.JSON(http.StatusOK, menus)
}

// CreateRole creates a new role
func CreateRole(c *gin.Context) {
	var input struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		MenuIDs     []uint `json:"menu_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := model.Role{
		Name:        input.Name,
		Description: input.Description,
	}
	if err := database.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role"})
		return
	}

	// Assign menus
	if len(input.MenuIDs) > 0 {
		var menus []model.Menu
		database.DB.Where("id IN ?", input.MenuIDs).Find(&menus)
		database.DB.Model(&role).Association("Menus").Replace(menus)
	}

	database.DB.Preload("Menus").First(&role, role.ID)
	c.JSON(http.StatusOK, role)
}

// UpdateRole updates a role's info and menu permissions
func UpdateRole(c *gin.Context) {
	id := c.Param("id")
	var role model.Role
	if err := database.DB.Preload("Menus").First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}

	var input struct {
		Description string `json:"description"`
		MenuIDs     []uint `json:"menu_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role.Description = input.Description
	database.DB.Save(&role)

	// Replace menu associations
	var menus []model.Menu
	if len(input.MenuIDs) > 0 {
		database.DB.Where("id IN ?", input.MenuIDs).Find(&menus)
	}
	database.DB.Model(&role).Association("Menus").Replace(menus)

	database.DB.Preload("Menus").First(&role, role.ID)
	c.JSON(http.StatusOK, role)
}

// DeleteRole deletes a role (cannot delete built-in roles)
func DeleteRole(c *gin.Context) {
	id := c.Param("id")
	var role model.Role
	if err := database.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Role not found"})
		return
	}
	if role.Name == "admin" || role.Name == "user" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete built-in roles"})
		return
	}
	database.DB.Model(&role).Association("Menus").Clear()
	database.DB.Delete(&role)
	c.JSON(http.StatusOK, gin.H{"message": "Role deleted"})
}
