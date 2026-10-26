package handler

import (
	"net/http"
	"strconv"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var referenceGalleryAssetLabels = map[string]string{
	"feature_pull_360":      "抽拉 + 360°参考图",
	"feature_pull":          "仅抽拉参考图",
	"feature_360":           "仅 360°参考图",
	"water_flow_single":     "单模式水流参考图",
	"water_flow_dual":       "双模式水流参考图",
	"water_flow_triple":     "三模式水流参考图",
	"feature_selling_point": "卖点图参考图",
	"dimension":             "尺寸图参考",
	"family":                "全家福参考图",
	"multi_color_white_bg":  "白底多色参考图",
}

func ListReferenceGalleries(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	category := strings.TrimSpace(c.Query("category"))

	query := database.DB.Model(&model.ReferenceGallery{}).Order("sort asc, id desc")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}

	var items []model.ReferenceGallery
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询参考图库失败"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func AdminCreateReferenceGallery(c *gin.Context) {
	var input referenceGalleryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, ok := buildReferenceGalleryFromInput(input)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参考图库名称不能为空"})
		return
	}
	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建参考图库失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func AdminUpdateReferenceGallery(c *gin.Context) {
	id, ok := parseReferenceGalleryID(c)
	if !ok {
		return
	}

	var existing model.ReferenceGallery
	if err := database.DB.First(&existing, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "参考图库不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取参考图库失败"})
		return
	}

	var input referenceGalleryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, ok := buildReferenceGalleryFromInput(input)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参考图库名称不能为空"})
		return
	}

	existing.Name = item.Name
	existing.Category = item.Category
	existing.Version = item.Version
	existing.Description = item.Description
	existing.Status = item.Status
	existing.Assets = item.Assets
	existing.Sort = item.Sort

	if err := database.DB.Save(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新参考图库失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

func AdminDeleteReferenceGallery(c *gin.Context) {
	id, ok := parseReferenceGalleryID(c)
	if !ok {
		return
	}
	if err := database.DB.Delete(&model.ReferenceGallery{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除参考图库失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func GetReferenceGalleryAssetKeys(c *gin.Context) {
	keys := make([]gin.H, 0, len(referenceGalleryAssetLabels))
	order := []string{
		"feature_pull_360",
		"feature_pull",
		"feature_360",
		"water_flow_single",
		"water_flow_dual",
		"water_flow_triple",
		"feature_selling_point",
		"dimension",
		"family",
		"multi_color_white_bg",
	}
	for _, key := range order {
		keys = append(keys, gin.H{"key": key, "label": referenceGalleryAssetLabels[key]})
	}
	c.JSON(http.StatusOK, keys)
}

type referenceGalleryInput struct {
	Name        string                        `json:"name"`
	Category    string                        `json:"category"`
	Version     string                        `json:"version"`
	Description string                        `json:"description"`
	Status      string                        `json:"status"`
	Assets      []model.ReferenceGalleryAsset `json:"assets"`
	Sort        int                           `json:"sort"`
}

func buildReferenceGalleryFromInput(input referenceGalleryInput) (model.ReferenceGallery, bool) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return model.ReferenceGallery{}, false
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = model.ReferenceGalleryStatusEnabled
	}
	if status != model.ReferenceGalleryStatusEnabled && status != model.ReferenceGalleryStatusDisabled {
		status = model.ReferenceGalleryStatusDisabled
	}

	return model.ReferenceGallery{
		Name:        name,
		Category:    strings.TrimSpace(input.Category),
		Version:     strings.TrimSpace(input.Version),
		Description: strings.TrimSpace(input.Description),
		Status:      status,
		Assets:      normalizeReferenceGalleryAssets(input.Assets),
		Sort:        input.Sort,
	}, true
}

func normalizeReferenceGalleryAssets(items []model.ReferenceGalleryAsset) []model.ReferenceGalleryAsset {
	result := make([]model.ReferenceGalleryAsset, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.Key)
		if !isReferenceGalleryAssetKey(key) {
			continue
		}
		url := strings.TrimSpace(item.URL)
		if url == "" {
			continue
		}
		desc := strings.TrimSpace(item.Description)
		if desc == "" {
			desc = referenceGalleryAssetLabels[key]
		}
		result = append(result, model.ReferenceGalleryAsset{
			Key:         key,
			URL:         url,
			Description: desc,
			Regions:     normalizeReferenceGalleryRegions(item.Regions),
		})
	}
	return result
}

func normalizeReferenceGalleryRegions(items []model.ReferenceGalleryRegion) []model.ReferenceGalleryRegion {
	result := make([]model.ReferenceGalleryRegion, 0, len(items))
	for _, item := range items {
		regionType := strings.TrimSpace(item.Type)
		if regionType == "" {
			regionType = "replace_product"
		}
		fit := strings.TrimSpace(item.Fit)
		if fit == "" {
			fit = "contain"
		}
		x := clampReferenceGalleryRegionValue(item.X)
		y := clampReferenceGalleryRegionValue(item.Y)
		w := clampReferenceGalleryRegionValue(item.W)
		h := clampReferenceGalleryRegionValue(item.H)
		if w <= 0 || h <= 0 {
			continue
		}
		if x+w > 1 {
			w = 1 - x
		}
		if y+h > 1 {
			h = 1 - y
		}
		if w <= 0 || h <= 0 {
			continue
		}
		id := strings.TrimSpace(item.ID)
		if id == "" {
			id = regionType
		}
		result = append(result, model.ReferenceGalleryRegion{
			ID:   id,
			Type: regionType,
			Fit:  fit,
			X:    x,
			Y:    y,
			W:    w,
			H:    h,
		})
	}
	return result
}

func clampReferenceGalleryRegionValue(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func isReferenceGalleryAssetKey(key string) bool {
	_, ok := referenceGalleryAssetLabels[key]
	return ok
}

func parseReferenceGalleryID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的参考图库 ID"})
		return 0, false
	}
	return uint(id), true
}
