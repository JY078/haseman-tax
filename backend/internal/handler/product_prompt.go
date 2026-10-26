package handler

import (
	"net/http"
	"strconv"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
)

/** GetProductPrompts 获取所有产品图提示词配置列表 */
func GetProductPrompts(c *gin.Context) {
	var items []model.ProductPrompt
	if err := database.DB.Order("sort asc, id asc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product prompts"})
		return
	}

	c.JSON(http.StatusOK, items)
}

/** GetProductLibrary 获取产品库条目列表（供 web 端弹窗使用） */
func GetProductLibrary(c *gin.Context) {
	var items []model.ProductPrompt
	query := database.DB.Order("sort asc, id asc")

	keyword := c.Query("keyword")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name ILIKE ? OR brand ILIKE ? OR category ILIKE ? OR sku ILIKE ?", like, like, like, like)
	}

	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product library"})
		return
	}

	c.JSON(http.StatusOK, items)
}

/** CreateProductPrompt 创建新的产品图提示词配置 */
func CreateProductPrompt(c *gin.Context) {
	var input model.ProductPrompt
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "名称不能为空"})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Brand = strings.TrimSpace(input.Brand)
	input.Category = strings.TrimSpace(input.Category)
	input.SKU = strings.TrimSpace(input.SKU)
	input.ImageURL = strings.TrimSpace(input.ImageURL)
	input.SellingPoints = strings.TrimSpace(input.SellingPoints)

	if err := database.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product prompt"})
		return
	}

	c.JSON(http.StatusCreated, input)
}

/** UpdateProductPrompt 更新已有的产品图提示词配置 */
func UpdateProductPrompt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var existing model.ProductPrompt
	if err := database.DB.First(&existing, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product prompt not found"})
		return
	}

	var input struct {
		Name                 string             `json:"name"`
		Brand                string             `json:"brand"`
		Category             string             `json:"category"`
		SKU                  string             `json:"sku"`
		Color                string             `json:"color"`
		Material             string             `json:"material"`
		ImageURL             string             `json:"image_url"`
		SellingPoints        string             `json:"selling_points"`
		ProductParams        string             `json:"product_params"`
		UsageScenarios       string             `json:"usage_scenarios"`
		PackageContents      string             `json:"package_contents"`
		InstallationSteps    string             `json:"installation_steps"`
		InstallationNotes    string             `json:"installation_notes"`
		ForbiddenFabrication string             `json:"forbidden_fabrication"`
		ReferenceImages      []model.ReferenceImage `json:"reference_images"`
		MainImagePrompt      string             `json:"main_image_prompt"`
		DetailPagePrompt     string             `json:"detail_page_prompt"`
		CustomPromptSuffix   string             `json:"custom_prompt_suffix"`
		Sort                 int                `json:"sort"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name != "" {
		existing.Name = strings.TrimSpace(input.Name)
	}

	existing.Brand = strings.TrimSpace(input.Brand)
	existing.Category = strings.TrimSpace(input.Category)
	existing.SKU = strings.TrimSpace(input.SKU)
	existing.Color = strings.TrimSpace(input.Color)
	existing.Material = strings.TrimSpace(input.Material)
	existing.ImageURL = strings.TrimSpace(input.ImageURL)
	existing.SellingPoints = strings.TrimSpace(input.SellingPoints)
	existing.ProductParams = strings.TrimSpace(input.ProductParams)
	existing.UsageScenarios = strings.TrimSpace(input.UsageScenarios)
	existing.PackageContents = strings.TrimSpace(input.PackageContents)
	existing.InstallationSteps = strings.TrimSpace(input.InstallationSteps)
	existing.InstallationNotes = strings.TrimSpace(input.InstallationNotes)
	existing.ForbiddenFabrication = strings.TrimSpace(input.ForbiddenFabrication)
	existing.ReferenceImages = input.ReferenceImages

	/** 提示词字段允许为空字符串，直接覆盖 */
	existing.MainImagePrompt = input.MainImagePrompt
	existing.DetailPagePrompt = input.DetailPagePrompt
	existing.CustomPromptSuffix = input.CustomPromptSuffix
	existing.Sort = input.Sort

	if err := database.DB.Save(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product prompt"})
		return
	}

	c.JSON(http.StatusOK, existing)
}

/** DeleteProductPrompt 删除产品图提示词配置 */
func DeleteProductPrompt(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := database.DB.Delete(&model.ProductPrompt{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product prompt"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product prompt deleted successfully"})
}
