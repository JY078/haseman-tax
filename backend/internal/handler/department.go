package handler

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var departmentPlatforms = map[string]bool{
	"amazon":    true,
	"fbm":       true,
	"data":      true,
	"designer":  true,
	"homedepot": true,
	"lowes":     true,
	"temu":      true,
	"tiktok":    true,
	"walmart":   true,
	"wayfair":   true,
}

func ListDepartments(c *gin.Context) {
	var items []model.Department
	query := database.DB.Order("sort asc, id asc")
	if platform := strings.TrimSpace(c.Query("platform")); platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch departments"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func CreateDepartment(c *gin.Context) {
	var input departmentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, ok := normalizeDepartmentInput(input)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "平台、一级部门不能为空，且 NAS 地区目录必须是相对路径"})
		return
	}

	var duplicate int64
	database.DB.Model(&model.Department{}).
		Where("platform = ? AND name = ? AND sub_name = ?", item.Platform, item.Name, item.SubName).
		Count(&duplicate)
	if duplicate > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该平台下已存在相同的一级/二级部门"})
		return
	}

	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create department: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func UpdateDepartment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的部门 ID"})
		return
	}

	var existing model.Department
	if err := database.DB.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "部门不存在"})
		return
	}

	var input departmentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, ok := normalizeDepartmentInput(input)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "平台、一级部门不能为空，且 NAS 地区目录必须是相对路径"})
		return
	}

	existing.Platform = item.Platform
	existing.Name = item.Name
	existing.SubName = item.SubName
	existing.NASRegionPath = item.NASRegionPath
	existing.Sort = item.Sort
	if err := database.DB.Save(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update department: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

func DeleteDepartment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的部门 ID"})
		return
	}

	var item model.Department
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "部门不存在"})
		return
	}

	var userCount int64
	database.DB.Model(&model.User{}).
		Where("platform = ? AND department = ? AND sub_department = ?", item.Platform, item.Name, item.SubName).
		Count(&userCount)
	if userCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该部门下已有用户，请先调整用户部门"})
		return
	}

	if err := database.DB.Delete(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除部门失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

type departmentInput struct {
	Platform      string `json:"platform"`
	Name          string `json:"name"`
	SubName       string `json:"sub_name"`
	NASRegionPath string `json:"nas_region_path"`
	Sort          int    `json:"sort"`
}

func normalizeDepartmentInput(input departmentInput) (model.Department, bool) {
	platform := normalizePlatform(input.Platform)
	name := strings.TrimSpace(input.Name)
	subName := strings.TrimSpace(input.SubName)
	nasRegionPath, ok := cleanRelativeNASPath(input.NASRegionPath)
	if !ok || platform == "" || name == "" {
		return model.Department{}, false
	}
	return model.Department{
		Platform:      platform,
		Name:          name,
		SubName:       subName,
		NASRegionPath: nasRegionPath,
		Sort:          input.Sort,
	}, true
}

func normalizePlatform(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "'", "")
	if value == "lowe" || value == "lowes" {
		value = "lowes"
	}
	if !departmentPlatforms[value] {
		return ""
	}
	return value
}

func cleanRelativeNASPath(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || strings.Contains(clean, ":") {
		return "", false
	}
	return clean, true
}

func findDepartmentByUser(user model.User) (model.Department, bool) {
	var department model.Department
	err := database.DB.Where("platform = ? AND name = ? AND sub_name = ?",
		user.Platform,
		user.Department,
		user.SubDepartment,
	).First(&department).Error
	if err == nil {
		return department, true
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return model.Department{}, false
	}
	return model.Department{}, false
}
