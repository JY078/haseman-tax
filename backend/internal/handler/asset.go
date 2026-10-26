package handler

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type assetDTO struct {
	ID                 string   `json:"id"`
	SourceType         string   `json:"source_type"`
	SourceID           string   `json:"source_id"`
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	Category           string   `json:"category"`
	OwnerType          string   `json:"owner_type"`
	OwnerName          string   `json:"owner_name"`
	Permission         string   `json:"permission"`
	Visibility         string   `json:"visibility"`
	Usage              int      `json:"usage"`
	UpdatedAt          string   `json:"updated_at"`
	CanManage          bool     `json:"can_manage"`
	Favorite           bool     `json:"favorite"`
	Tags               []string `json:"tags"`
	Cover              string   `json:"cover"`
	FileURL            string   `json:"file_url"`
	FileName           string   `json:"file_name"`
	FileType           string   `json:"file_type"`
	SPU                string   `json:"spu"`
	SKU                string   `json:"sku"`
	ProductID          uint     `json:"product_id"`
	ProductPath        string   `json:"product_path"`
	ReferenceSceneID   uint     `json:"reference_scene_id"`
	ReferenceSceneName string   `json:"reference_scene_name"`
	ReferenceTagID     uint     `json:"reference_tag_id"`
	ReferenceTagName   string   `json:"reference_tag_name"`
	ReferenceGalleryID uint     `json:"reference_gallery_id"`
}

type assetCategoryDTO struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	OwnerType  string `json:"owner_type"`
	OwnerName  string `json:"owner_name"`
	CanManage  bool   `json:"can_manage"`
	AssetCount int64  `json:"asset_count"`
	Sort       int    `json:"sort"`
}

func ListAssets(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}

	favorites := loadAssetFavoriteSet(user.ID)
	items := make([]assetDTO, 0, 64)

	nativeAssets, err := loadNativeAssets(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "资产列表加载失败"})
		return
	}
	for _, asset := range nativeAssets {
		dto := nativeAssetDTO(asset, user)
		dto.Favorite = favorites[favoriteKey(dto.SourceType, dto.SourceID)]
		items = append(items, dto)
	}

	productAssets, err := loadProductAssetDTOs(favorites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "产品库资产加载失败"})
		return
	}
	items = append(items, productAssets...)

	styleAssets, err := loadStyleAssetDTOs(favorites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "风格资产加载失败"})
		return
	}
	items = append(items, styleAssets...)

	colorAssets, err := loadColorAssetDTOs(favorites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "色系资产加载失败"})
		return
	}
	items = append(items, colorAssets...)

	referenceAssets, err := loadReferenceGalleryAssetDTOs(favorites)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "参考图库资产加载失败"})
		return
	}
	items = append(items, referenceAssets...)

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func ListAssetCategories(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}

	var categories []model.AssetCategory
	if err := database.DB.Where(assetCategoryVisibilityCondition(), user.ID, user.Department).Order("owner_type asc, sort asc, updated_at desc, id desc").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "资产分类加载失败"})
		return
	}

	items := make([]assetCategoryDTO, 0, len(categories)+16)
	seen := map[string]bool{}
	for _, category := range categories {
		dto := assetCategoryDTOFromModel(category, user)
		key := assetCategoryKey(dto.OwnerType, dto.Type, dto.Name)
		seen[key] = true
		items = append(items, dto)
	}

	var rows []struct {
		OwnerType  string
		Type       string
		Category   string
		Department string
		OwnerID    uint
		UserID     uint
		Count      int64
	}
	if err := database.DB.Model(&model.Asset{}).
		Select("owner_type, type, category, department, owner_id, user_id, COUNT(*) AS count").
		Where(assetVisibilityCondition(), user.ID, user.Department).
		Where("category <> ''").
		Group("owner_type, type, category, department, owner_id, user_id").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "资产分类加载失败"})
		return
	}
	for _, row := range rows {
		dto := assetCategoryDTO{
			Name:       row.Category,
			Type:       row.Type,
			OwnerType:  row.OwnerType,
			OwnerName:  nativeAssetOwnerName(model.Asset{OwnerType: row.OwnerType, Department: row.Department}),
			CanManage:  canManageAssetCategory(row.OwnerType, row.OwnerID, row.UserID, row.Department, user),
			AssetCount: row.Count,
		}
		key := assetCategoryKey(dto.OwnerType, dto.Type, dto.Name)
		if seen[key] {
			for i := range items {
				if assetCategoryKey(items[i].OwnerType, items[i].Type, items[i].Name) == key {
					items[i].AssetCount = row.Count
					break
				}
			}
			continue
		}
		seen[key] = true
		items = append(items, dto)
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func CreateAssetCategory(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}
	var input struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		OwnerType string `json:"owner_type"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ownerType := normalizeAssetOwnerType(input.OwnerType)
	if !canCreateAssetCategoryForOwner(ownerType, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以创建系统资产分类"})
		return
	}
	if ownerType == "department" && strings.TrimSpace(user.Department) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前账号未设置部门，无法创建部门分类"})
		return
	}
	name := strings.TrimSpace(input.Name)
	assetType := strings.TrimSpace(input.Type)
	if name == "" || assetType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分类名称和资产大类不能为空"})
		return
	}
	category := model.AssetCategory{
		Name:       name,
		Type:       assetType,
		OwnerType:  ownerType,
		OwnerID:    assetOwnerID(ownerType, user),
		UserID:     user.ID,
		Department: assetDepartmentForOwner(ownerType, user),
	}
	categoryQuery := assetCategoryIdentityQuery(database.DB, category)
	if err := categoryQuery.FirstOrCreate(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "资产分类创建失败"})
		return
	}
	c.JSON(http.StatusCreated, assetCategoryDTOFromModel(category, user))
}

func RepairAssetCategories(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}
	var input struct {
		OwnerType string `json:"owner_type"`
		Type      string `json:"type"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ownerType := normalizeAssetOwnerType(input.OwnerType)
	assetType := strings.TrimSpace(input.Type)
	if assetType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "资产大类不能为空"})
		return
	}
	if !canCreateAssetCategoryForOwner(ownerType, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限修复该资产库分类"})
		return
	}

	var rows []struct {
		Category string
	}
	query := database.DB.Model(&model.Asset{}).
		Select("category").
		Where("owner_type = ? AND type = ? AND category <> ''", ownerType, assetType).
		Group("category")
	if ownerType == "personal" {
		query = query.Where("owner_id = ?", assetOwnerID(ownerType, user))
	} else if ownerType == "department" {
		department := assetDepartmentForOwner(ownerType, user)
		if department == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "当前账号未设置部门，无法修复部门分类"})
			return
		}
		query = query.Where("owner_id = ? AND department = ?", assetOwnerID(ownerType, user), department)
	}
	if err := query.Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "历史分类扫描失败"})
		return
	}

	created := 0
	for _, row := range rows {
		name := strings.TrimSpace(row.Category)
		if name == "" {
			continue
		}
		category := model.AssetCategory{
			Name:       name,
			Type:       assetType,
			OwnerType:  ownerType,
			OwnerID:    assetOwnerID(ownerType, user),
			UserID:     user.ID,
			Department: assetDepartmentForOwner(ownerType, user),
		}
		query := assetCategoryIdentityQuery(database.DB, category).FirstOrCreate(&category)
		if query.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "历史分类修复失败"})
			return
		}
		if query.RowsAffected > 0 {
			created++
		}
	}
	c.JSON(http.StatusOK, gin.H{"created": created})
}

func ReorderAssetCategories(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}
	var input struct {
		CategoryIDs []uint `json:"category_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(input.CategoryIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分类排序不能为空"})
		return
	}

	categories := make([]model.AssetCategory, 0, len(input.CategoryIDs))
	for _, id := range input.CategoryIDs {
		category, ok := loadManageableAssetCategory(id, user)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "没有权限管理分类排序"})
			return
		}
		categories = append(categories, category)
	}
	first := categories[0]
	for _, category := range categories[1:] {
		if !assetCategoriesSameLibrary(first, category) || first.Type != category.Type {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只能调整同资产库、同大类下的分类排序"})
			return
		}
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for index, id := range input.CategoryIDs {
			if err := tx.Model(&model.AssetCategory{}).Where("id = ?", id).Update("sort", (index+1)*10).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "分类排序保存失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "分类排序已保存"})
}

func UpdateAssetCategory(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}
	category, ok := findManageableAssetCategory(c, user)
	if !ok {
		return
	}

	var input struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	nextName := strings.TrimSpace(input.Name)
	if nextName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分类名称不能为空"})
		return
	}
	if nextName == category.Name {
		c.JSON(http.StatusOK, assetCategoryDTOFromModel(category, user))
		return
	}
	if assetCategoryNameExists(category, nextName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "同资产库下已存在同名分类"})
		return
	}

	oldName := category.Name
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AssetCategory{}).Where("id = ?", category.ID).Update("name", nextName).Error; err != nil {
			return err
		}
		if err := assetCategoryAssetsQuery(tx, category).Updates(map[string]interface{}{"category": nextName}).Error; err != nil {
			return err
		}
		category.Name = nextName
		return nil
	})
	if err != nil {
		category.Name = oldName
		c.JSON(http.StatusInternalServerError, gin.H{"error": "分类更新失败"})
		return
	}
	c.JSON(http.StatusOK, assetCategoryDTOFromModel(category, user))
}

func DeleteAssetCategory(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}
	category, ok := findManageableAssetCategory(c, user)
	if !ok {
		return
	}
	if countNativeAssetsInCategory(category) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分类下还有资产，请先移动资产后再删除"})
		return
	}
	if err := database.DB.Delete(&category).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "分类删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "分类已删除"})
}

func MoveAssetCategoryAssets(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}
	var input struct {
		FromCategoryID uint   `json:"from_category_id"`
		ToCategoryID   uint   `json:"to_category_id"`
		AssetIDs       []uint `json:"asset_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.FromCategoryID == 0 || input.ToCategoryID == 0 || input.FromCategoryID == input.ToCategoryID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择有效的目标分类"})
		return
	}

	fromCategory, ok := loadManageableAssetCategory(input.FromCategoryID, user)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限管理当前分类"})
		return
	}
	var toCategory model.AssetCategory
	if err := database.DB.First(&toCategory, input.ToCategoryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "目标分类不存在"})
		return
	}
	if !canManageAssetCategory(toCategory.OwnerType, toCategory.OwnerID, toCategory.UserID, toCategory.Department, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限管理目标分类"})
		return
	}
	if !assetCategoriesSameLibrary(fromCategory, toCategory) || fromCategory.Type != toCategory.Type {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能移动到同资产库、同大类下的分类"})
		return
	}

	query := assetCategoryAssetsQuery(database.DB, fromCategory)
	if len(input.AssetIDs) > 0 {
		query = query.Where("id IN ?", input.AssetIDs)
	}
	result := query.Updates(map[string]interface{}{"category": toCategory.Name})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "资产移动失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"moved": result.RowsAffected})
}

func MoveSceneStyleReferenceAssets(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}
	if !assetUserIsAdmin(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以管理风格参考图"})
		return
	}
	var input struct {
		FromSceneID  uint   `json:"from_scene_id"`
		FromTagID    uint   `json:"from_tag_id"`
		ToTagID      uint   `json:"to_tag_id"`
		ReferenceIDs []uint `json:"reference_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.FromTagID == 0 || input.ToTagID == 0 || input.FromTagID == input.ToTagID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择有效的目标风格"})
		return
	}
	var targetTag model.SceneStyleTag
	if err := database.DB.First(&targetTag, input.ToTagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "目标风格不存在"})
		return
	}
	query := database.DB.Model(&model.SceneStyleReference{}).Where("tag_id = ?", input.FromTagID)
	if input.FromSceneID > 0 {
		query = query.Where("scene_id = ?", input.FromSceneID)
	}
	if len(input.ReferenceIDs) > 0 {
		query = query.Where("id IN ?", input.ReferenceIDs)
	}
	result := query.Update("tag_id", input.ToTagID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "风格参考图移动失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"moved": result.RowsAffected})
}

func CreateAsset(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}

	ownerType := normalizeAssetOwnerType(c.PostForm("owner_type"))
	if ownerType == "shared" {
		ownerType = "personal"
	}
	if ownerType == "system" && !assetUserIsAdmin(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以创建系统资产"})
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	assetType := strings.TrimSpace(c.PostForm("type"))
	categoryName := strings.TrimSpace(c.PostForm("category"))
	if assetType == "" || categoryName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择资产分类"})
		return
	}
	if !assetCategoryCanReceiveUpload(ownerType, assetType, categoryName, user) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所选分类不属于当前资产库"})
		return
	}

	fileURL, coverURL, fileName, fileType, err := saveAssetUploadFile(c, "file")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败: " + err.Error()})
		return
	}
	if strings.TrimSpace(fileURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的文件"})
		return
	}
	if name == "" {
		name = strings.TrimSpace(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
	}
	if name == "" {
		name = "未命名资产"
	}

	asset := model.Asset{
		Name:       name,
		Type:       assetType,
		Category:   categoryName,
		OwnerType:  ownerType,
		OwnerID:    assetOwnerID(ownerType, user),
		UserID:     user.ID,
		Department: assetDepartmentForOwner(ownerType, user),
		Visibility: defaultAssetVisibility(ownerType),
		FileURL:    fileURL,
		CoverURL:   coverURL,
		FileName:   fileName,
		FileType:   fileType,
		Tags:       marshalAssetTags(parseAssetTags(c.PostForm("tags"))),
	}

	if err := database.DB.Create(&asset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "资产创建失败"})
		return
	}

	c.JSON(http.StatusCreated, nativeAssetDTO(asset, user))
}

func UpdateAsset(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}
	asset, ok := findEditableAsset(c, user)
	if !ok {
		return
	}

	var input struct {
		Name     string   `json:"name"`
		Type     string   `json:"type"`
		Category string   `json:"category"`
		Tags     []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "资产名称不能为空"})
		return
	}

	asset.Name = name
	asset.Type = strings.TrimSpace(input.Type)
	asset.Category = strings.TrimSpace(input.Category)
	if asset.Type == "" {
		asset.Type = assetTypeFromFileType(asset.FileType)
	}
	if asset.Category == "" {
		asset.Category = asset.Type
	}
	asset.Visibility = defaultAssetVisibility(asset.OwnerType)
	asset.Tags = marshalAssetTags(input.Tags)

	if err := database.DB.Save(&asset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "资产更新失败"})
		return
	}
	c.JSON(http.StatusOK, nativeAssetDTO(asset, user))
}

func DeleteAsset(c *gin.Context) {
	user, ok := assetCurrentUser(c)
	if !ok {
		return
	}
	asset, ok := findEditableAsset(c, user)
	if !ok {
		return
	}

	if err := database.DB.Delete(&asset).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "资产删除失败"})
		return
	}
	database.DB.Where("source_type = ? AND source_id = ?", "asset", strconv.FormatUint(uint64(asset.ID), 10)).Delete(&model.AssetFavorite{})
	c.JSON(http.StatusOK, gin.H{"message": "资产已删除"})
}

func SetAssetFavorite(c *gin.Context) {
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var input struct {
		SourceType string `json:"source_type"`
		SourceID   string `json:"source_id"`
		Favorite   bool   `json:"favorite"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sourceType := strings.TrimSpace(input.SourceType)
	sourceID := strings.TrimSpace(input.SourceID)
	if sourceType == "" || sourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "资产来源不能为空"})
		return
	}

	if input.Favorite {
		fav := model.AssetFavorite{UserID: userID, SourceType: sourceType, SourceID: sourceID}
		if err := database.DB.Where("user_id = ? AND source_type = ? AND source_id = ?", userID, sourceType, sourceID).
			FirstOrCreate(&fav).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "收藏失败"})
			return
		}
	} else {
		if err := database.DB.Where("user_id = ? AND source_type = ? AND source_id = ?", userID, sourceType, sourceID).
			Delete(&model.AssetFavorite{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "取消收藏失败"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"favorite": input.Favorite})
}

func RecordAssetUsage(c *gin.Context) {
	var input struct {
		SourceType string `json:"source_type"`
		SourceID   string `json:"source_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.SourceType == "asset" {
		id, err := strconv.ParseUint(input.SourceID, 10, 64)
		if err == nil && id > 0 {
			database.DB.Model(&model.Asset{}).Where("id = ?", id).UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "已记录"})
}

func loadNativeAssets(user model.User) ([]model.Asset, error) {
	var items []model.Asset
	query := database.DB.Order("updated_at desc, id desc")
	query = query.Where(assetVisibilityCondition(), user.ID, user.Department)
	err := query.Find(&items).Error
	return items, err
}

func loadProductAssetDTOs(favorites map[string]bool) ([]assetDTO, error) {
	items := make([]assetDTO, 0, 32)

	var spus []model.ProductSPU
	if err := database.DB.Where("model_url <> ''").Order("updated_at desc").Limit(300).Find(&spus).Error; err != nil {
		return nil, err
	}
	for _, spu := range spus {
		sourceID := strconv.FormatUint(uint64(spu.ID), 10)
		items = append(items, assetDTO{
			ID:          "product_spu_model:" + sourceID,
			SourceType:  "product_spu_model",
			SourceID:    sourceID,
			Name:        assetFirstNonEmpty(spu.Name, spu.SPU+" 模型文件"),
			Type:        "模型文件",
			Category:    assetFirstNonEmpty(spu.Category2, spu.Category1, "产品模型"),
			OwnerType:   "system",
			OwnerName:   "产品库",
			Permission:  "系统资产，只读使用",
			Visibility:  "public",
			Usage:       0,
			UpdatedAt:   formatAssetTime(spu.UpdatedAt),
			CanManage:   false,
			Favorite:    favorites[favoriteKey("product_spu_model", sourceID)],
			Tags:        compactStrings([]string{spu.SPU, spu.Brand, spu.Model, spu.Material}),
			Cover:       "",
			FileURL:     spu.ModelURL,
			FileName:    filepath.Base(spu.ModelURL),
			FileType:    detectAssetFileTypeFromURL(spu.ModelURL),
			SPU:         spu.SPU,
			ProductID:   spu.ID,
			ProductPath: fmt.Sprintf("/admin/product-spus/%d", spu.ID),
		})
	}

	var skus []model.ProductSKU
	if err := database.DB.Where("stp_model_url <> ''").Order("updated_at desc").Limit(300).Find(&skus).Error; err != nil {
		return nil, err
	}
	for _, sku := range skus {
		sourceID := strconv.FormatUint(uint64(sku.ID), 10)
		items = append(items, assetDTO{
			ID:          "product_sku_model:" + sourceID,
			SourceType:  "product_sku_model",
			SourceID:    sourceID,
			Name:        assetFirstNonEmpty(sku.Name, sku.SKU+" STP 模型"),
			Type:        "STP模型",
			Category:    assetFirstNonEmpty(sku.Category3, sku.Category2, sku.Category1, "产品模型"),
			OwnerType:   "system",
			OwnerName:   "产品库",
			Permission:  "系统资产，只读使用",
			Visibility:  "public",
			Usage:       0,
			UpdatedAt:   formatAssetTime(sku.UpdatedAt),
			CanManage:   false,
			Favorite:    favorites[favoriteKey("product_sku_model", sourceID)],
			Tags:        compactStrings([]string{sku.SPU, sku.SKU, sku.StyleName, sku.Color, sku.Size}),
			Cover:       "",
			FileURL:     sku.StpModelURL,
			FileName:    filepath.Base(sku.StpModelURL),
			FileType:    detectAssetFileTypeFromURL(sku.StpModelURL),
			SPU:         sku.SPU,
			SKU:         sku.SKU,
			ProductID:   sku.SPUID,
			ProductPath: "/admin/product-skus",
		})
	}

	var attachments []model.ProductSPUAttachment
	if err := database.DB.Where("file_url <> ''").Order("updated_at desc").Limit(300).Find(&attachments).Error; err != nil {
		return nil, err
	}
	for _, attachment := range attachments {
		sourceID := strconv.FormatUint(uint64(attachment.ID), 10)
		items = append(items, assetDTO{
			ID:          "product_spu_attachment:" + sourceID,
			SourceType:  "product_spu_attachment",
			SourceID:    sourceID,
			Name:        assetFirstNonEmpty(attachment.Label, attachment.FileName, "产品附件"),
			Type:        assetTypeFromFileType(attachment.FileType),
			Category:    "产品附件",
			OwnerType:   "system",
			OwnerName:   "产品库",
			Permission:  "系统资产，只读使用",
			Visibility:  "public",
			Usage:       0,
			UpdatedAt:   formatAssetTime(attachment.UpdatedAt),
			CanManage:   false,
			Favorite:    favorites[favoriteKey("product_spu_attachment", sourceID)],
			Tags:        []string{"产品附件"},
			Cover:       "",
			FileURL:     attachment.FileURL,
			FileName:    assetFirstNonEmpty(attachment.FileName, filepath.Base(attachment.FileURL)),
			FileType:    assetFirstNonEmpty(attachment.FileType, detectAssetFileTypeFromURL(attachment.FileURL)),
			ProductID:   attachment.SPUID,
			ProductPath: fmt.Sprintf("/admin/product-spus/%d", attachment.SPUID),
		})
	}

	return items, nil
}

func loadStyleAssetDTOs(favorites map[string]bool) ([]assetDTO, error) {
	var refs []model.SceneStyleReference
	if err := database.DB.Order("updated_at desc, id desc").Limit(500).Find(&refs).Error; err != nil {
		return nil, err
	}

	tagIDs := make([]uint, 0, len(refs))
	seenTag := map[uint]bool{}
	sceneIDs := make([]uint, 0, len(refs))
	seenScene := map[uint]bool{}
	for _, ref := range refs {
		if ref.TagID > 0 && !seenTag[ref.TagID] {
			seenTag[ref.TagID] = true
			tagIDs = append(tagIDs, ref.TagID)
		}
		if ref.SceneID > 0 && !seenScene[ref.SceneID] {
			seenScene[ref.SceneID] = true
			sceneIDs = append(sceneIDs, ref.SceneID)
		}
	}
	tagNames := map[uint]string{}
	if len(tagIDs) > 0 {
		var tags []model.SceneStyleTag
		if err := database.DB.Select("id", "name").Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
			return nil, err
		}
		for _, tag := range tags {
			tagNames[tag.ID] = tag.Name
		}
	}
	sceneNames := map[uint]string{}
	if len(sceneIDs) > 0 {
		var scenes []model.SceneStyleCategory
		if err := database.DB.Select("id", "name").Where("id IN ?", sceneIDs).Find(&scenes).Error; err != nil {
			return nil, err
		}
		for _, scene := range scenes {
			sceneNames[scene.ID] = scene.Name
		}
	}

	items := make([]assetDTO, 0, len(refs))
	for _, ref := range refs {
		sourceID := strconv.FormatUint(uint64(ref.ID), 10)
		tagName := assetFirstNonEmpty(tagNames[ref.TagID], "未分组风格")
		sceneName := assetFirstNonEmpty(sceneNames[ref.SceneID], assetSceneName(ref.SceneID))
		items = append(items, assetDTO{
			ID:                 "scene_style_reference:" + sourceID,
			SourceType:         "scene_style_reference",
			SourceID:           sourceID,
			Name:               assetFirstNonEmpty(ref.Title, tagName+" 参考图"),
			Type:               "风格参考图",
			Category:           tagName,
			OwnerType:          "system",
			OwnerName:          "风格库",
			Permission:         "系统资产，只读使用",
			Visibility:         "public",
			Usage:              0,
			UpdatedAt:          formatAssetTime(ref.UpdatedAt),
			CanManage:          false,
			Favorite:           favorites[favoriteKey("scene_style_reference", sourceID)],
			Tags:               compactStrings([]string{"风格参考图", tagName, sceneName}),
			Cover:              ref.ImageURL,
			FileURL:            ref.ImageURL,
			FileName:           filepath.Base(ref.ImageURL),
			FileType:           "image",
			ReferenceSceneID:   ref.SceneID,
			ReferenceSceneName: sceneName,
			ReferenceTagID:     ref.TagID,
			ReferenceTagName:   tagName,
		})
	}
	return items, nil
}

func loadColorAssetDTOs(favorites map[string]bool) ([]assetDTO, error) {
	var colors []model.ProductColorFamily
	if err := database.DB.Where("image_url <> ''").Order("sort asc, updated_at desc").Limit(300).Find(&colors).Error; err != nil {
		return nil, err
	}

	items := make([]assetDTO, 0, len(colors))
	for _, color := range colors {
		sourceID := strconv.FormatUint(uint64(color.ID), 10)
		items = append(items, assetDTO{
			ID:         "product_color_family:" + sourceID,
			SourceType: "product_color_family",
			SourceID:   sourceID,
			Name:       color.Name,
			Type:       "色块",
			Category:   "产品色系",
			OwnerType:  "system",
			OwnerName:  "色系库",
			Permission: "系统资产，只读使用",
			Visibility: "public",
			Usage:      0,
			UpdatedAt:  formatAssetTime(color.UpdatedAt),
			CanManage:  false,
			Favorite:   favorites[favoriteKey("product_color_family", sourceID)],
			Tags:       []string{"色块", "产品色系"},
			Cover:      color.ImageURL,
			FileURL:    color.ImageURL,
			FileName:   filepath.Base(color.ImageURL),
			FileType:   "image",
		})
	}
	return items, nil
}

func loadReferenceGalleryAssetDTOs(favorites map[string]bool) ([]assetDTO, error) {
	var galleries []model.ReferenceGallery
	if err := database.DB.Where("status = ?", model.ReferenceGalleryStatusEnabled).Order("sort asc, updated_at desc").Limit(200).Find(&galleries).Error; err != nil {
		return nil, err
	}

	items := make([]assetDTO, 0, len(galleries)*4)
	for _, gallery := range galleries {
		for _, asset := range gallery.Assets {
			if strings.TrimSpace(asset.URL) == "" {
				continue
			}
			sourceID := fmt.Sprintf("%d:%s", gallery.ID, asset.Key)
			items = append(items, assetDTO{
				ID:                 "reference_gallery_asset:" + sourceID,
				SourceType:         "reference_gallery_asset",
				SourceID:           sourceID,
				Name:               assetFirstNonEmpty(asset.Description, gallery.Name+" "+referenceGalleryAssetLabel(asset.Key)),
				Type:               "参考方案图",
				Category:           assetFirstNonEmpty(gallery.Category, "参考图库"),
				OwnerType:          "system",
				OwnerName:          "参考图库",
				Permission:         "系统资产，只读使用",
				Visibility:         "public",
				Usage:              0,
				UpdatedAt:          formatAssetTime(gallery.UpdatedAt),
				CanManage:          false,
				Favorite:           favorites[favoriteKey("reference_gallery_asset", sourceID)],
				Tags:               compactStrings([]string{"参考方案图", gallery.Name, gallery.Category, referenceGalleryAssetLabel(asset.Key)}),
				Cover:              asset.URL,
				FileURL:            asset.URL,
				FileName:           filepath.Base(asset.URL),
				FileType:           "image",
				ReferenceGalleryID: gallery.ID,
			})
		}
	}
	return items, nil
}

func nativeAssetDTO(asset model.Asset, user model.User) assetDTO {
	sourceID := strconv.FormatUint(uint64(asset.ID), 10)
	return assetDTO{
		ID:         "asset:" + sourceID,
		SourceType: "asset",
		SourceID:   sourceID,
		Name:       asset.Name,
		Type:       asset.Type,
		Category:   asset.Category,
		OwnerType:  asset.OwnerType,
		OwnerName:  nativeAssetOwnerName(asset),
		Permission: assetPermissionText(asset),
		Visibility: asset.Visibility,
		Usage:      asset.UsageCount,
		UpdatedAt:  formatAssetTime(asset.UpdatedAt),
		CanManage:  canManageNativeAsset(asset, user),
		Tags:       unmarshalAssetTags(asset.Tags),
		Cover:      assetFirstNonEmpty(asset.CoverURL, asset.FileURL),
		FileURL:    asset.FileURL,
		FileName:   asset.FileName,
		FileType:   asset.FileType,
	}
}

func assetCurrentUser(c *gin.Context) (model.User, bool) {
	userID := c.GetUint("userID")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return model.User{}, false
	}
	var user model.User
	if err := database.DB.Preload("Roles").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return model.User{}, false
	}
	return user, true
}

func assetUserIsAdmin(user model.User) bool {
	for _, role := range user.Roles {
		if role.Name == "admin" {
			return true
		}
	}
	return false
}

func findEditableAsset(c *gin.Context, user model.User) (model.Asset, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "资产 ID 无效"})
		return model.Asset{}, false
	}
	var asset model.Asset
	if err := database.DB.First(&asset, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "资产不存在"})
		return model.Asset{}, false
	}
	if !canManageNativeAsset(asset, user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有资产管理权限"})
		return model.Asset{}, false
	}
	return asset, true
}

func canManageNativeAsset(asset model.Asset, user model.User) bool {
	if asset.OwnerType == "system" {
		return assetUserIsAdmin(user)
	}
	if asset.OwnerType == "department" {
		return asset.Department != "" && asset.Department == user.Department
	}
	return asset.UserID == user.ID
}

func assetVisibilityCondition() string {
	return "owner_type = 'system' OR user_id = ? OR visibility = 'public' OR (owner_type = 'department' AND department <> '' AND department = ?)"
}

func assetCategoryVisibilityCondition() string {
	return "owner_type = 'system' OR user_id = ? OR (owner_type = 'department' AND department <> '' AND department = ?)"
}

func canManageAssetCategory(ownerType string, ownerID uint, userID uint, department string, user model.User) bool {
	if ownerType == "system" {
		return assetUserIsAdmin(user)
	}
	if ownerType == "department" {
		return department != "" && department == user.Department
	}
	if ownerID > 0 {
		return ownerID == user.ID
	}
	return userID == user.ID
}

func assetCategoryDTOFromModel(category model.AssetCategory, user model.User) assetCategoryDTO {
	return assetCategoryDTO{
		ID:         category.ID,
		Name:       category.Name,
		Type:       category.Type,
		OwnerType:  category.OwnerType,
		OwnerName:  nativeAssetOwnerName(model.Asset{OwnerType: category.OwnerType, Department: category.Department}),
		CanManage:  canManageAssetCategory(category.OwnerType, category.OwnerID, category.UserID, category.Department, user),
		AssetCount: countNativeAssetsInCategory(category),
		Sort:       category.Sort,
	}
}

func countNativeAssetsInCategory(category model.AssetCategory) int64 {
	var count int64
	query := assetCategoryAssetsQuery(database.DB, category)
	query.Count(&count)
	return count
}

func findManageableAssetCategory(c *gin.Context, user model.User) (model.AssetCategory, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "分类 ID 无效"})
		return model.AssetCategory{}, false
	}
	category, ok := loadManageableAssetCategory(uint(id), user)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "分类不存在或无权限管理"})
		return model.AssetCategory{}, false
	}
	return category, true
}

func loadManageableAssetCategory(id uint, user model.User) (model.AssetCategory, bool) {
	var category model.AssetCategory
	if err := database.DB.First(&category, id).Error; err != nil {
		return model.AssetCategory{}, false
	}
	if !canManageAssetCategory(category.OwnerType, category.OwnerID, category.UserID, category.Department, user) {
		return model.AssetCategory{}, false
	}
	return category, true
}

func assetCategoryAssetsQuery(db *gorm.DB, category model.AssetCategory) *gorm.DB {
	query := db.Model(&model.Asset{})
	if category.OwnerType == "system" {
		query = query.Where("owner_type = ? AND type = ? AND category = ?", category.OwnerType, category.Type, category.Name)
	} else if category.OwnerType == "personal" {
		query = query.Where("owner_type = ? AND owner_id = ? AND type = ? AND category = ?", category.OwnerType, category.OwnerID, category.Type, category.Name)
	} else {
		query = query.Where("owner_type = ? AND owner_id = ? AND department = ? AND type = ? AND category = ?", category.OwnerType, category.OwnerID, category.Department, category.Type, category.Name)
	}
	return query
}

func assetCategoryNameExists(category model.AssetCategory, name string) bool {
	var count int64
	query := database.DB.Model(&model.AssetCategory{}).Where("id <> ? AND owner_type = ? AND type = ? AND name = ?", category.ID, category.OwnerType, category.Type, name)
	if category.OwnerType == "system" {
		query = query.Where("owner_id = ? AND department = ?", 0, "")
	} else if category.OwnerType == "personal" {
		query = query.Where("owner_id = ?", category.OwnerID)
	} else {
		query = query.Where("owner_id = ? AND department = ?", category.OwnerID, category.Department)
	}
	query.Count(&count)
	return count > 0
}

func assetCategoriesSameLibrary(a, b model.AssetCategory) bool {
	return a.OwnerType == b.OwnerType && a.OwnerID == b.OwnerID && strings.TrimSpace(a.Department) == strings.TrimSpace(b.Department)
}

func assetCategoryIdentityQuery(db *gorm.DB, category model.AssetCategory) *gorm.DB {
	query := db.Where("owner_type = ? AND type = ? AND name = ?", category.OwnerType, category.Type, category.Name)
	if category.OwnerType == "system" {
		return query
	}
	if category.OwnerType == "personal" {
		return query.Where("owner_id = ?", category.OwnerID)
	}
	return query.Where("owner_id = ? AND department = ?", category.OwnerID, category.Department)
}

func canCreateAssetCategoryForOwner(ownerType string, user model.User) bool {
	return ownerType != "system" || assetUserIsAdmin(user)
}

func assetCategoryKey(ownerType, assetType, name string) string {
	return strings.TrimSpace(ownerType) + "\x00" + strings.TrimSpace(assetType) + "\x00" + strings.TrimSpace(name)
}

func saveAssetUploadFile(c *gin.Context, field string) (fileURL, coverURL, fileName, fileType string, err error) {
	file, err := c.FormFile(field)
	if err != nil {
		if err == http.ErrMissingFile {
			return "", "", "", "", nil
		}
		return "", "", "", "", err
	}
	return saveAssetFile(c, file)
}

func saveAssetFile(c *gin.Context, file *multipart.FileHeader) (fileURL, coverURL, fileName, fileType string, err error) {
	if file.Size <= 0 {
		return "", "", "", "", nil
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".bin"
	}
	tmpDir := "./uploads/assets"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", "", "", "", err
	}
	safeName := strings.ReplaceAll(strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)), " ", "_")
	tmpPath := filepath.Join(tmpDir, safeName+"_"+time.Now().Format("20060102150405.000000000")+ext)
	if err := c.SaveUploadedFile(file, tmpPath); err != nil {
		return "", "", "", "", err
	}
	url, err := storage.MoveToOSS(tmpPath, "assets")
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", "", "", "", err
	}
	fileType = detectAssetFileTypeFromURL(file.Filename)
	if fileType == "image" {
		coverURL = url
	}
	return url, coverURL, file.Filename, fileType, nil
}

func loadAssetFavoriteSet(userID uint) map[string]bool {
	var favorites []model.AssetFavorite
	database.DB.Where("user_id = ?", userID).Find(&favorites)
	result := make(map[string]bool, len(favorites))
	for _, item := range favorites {
		result[favoriteKey(item.SourceType, item.SourceID)] = true
	}
	return result
}

func favoriteKey(sourceType, sourceID string) string {
	return sourceType + ":" + sourceID
}

func normalizeAssetOwnerType(value string) string {
	switch strings.TrimSpace(value) {
	case "system", "department", "personal":
		return strings.TrimSpace(value)
	default:
		return "personal"
	}
}

func defaultAssetVisibility(ownerType string) string {
	if ownerType == "system" {
		return "public"
	}
	if ownerType == "department" {
		return "department"
	}
	return "private"
}

func assetOwnerID(ownerType string, user model.User) uint {
	if ownerType == "personal" {
		return user.ID
	}
	return 0
}

func assetDepartmentForOwner(ownerType string, user model.User) string {
	if ownerType == "department" {
		return strings.TrimSpace(user.Department)
	}
	return ""
}

func assetCategoryCanReceiveUpload(ownerType, assetType, categoryName string, user model.User) bool {
	department := assetDepartmentForOwner(ownerType, user)
	if ownerType == "department" && department == "" {
		return false
	}

	var categoryCount int64
	categoryQuery := database.DB.Model(&model.AssetCategory{})
	if ownerType == "system" {
		categoryQuery = categoryQuery.Where("owner_type = ? AND type = ? AND name = ?", ownerType, assetType, categoryName)
	} else if ownerType == "personal" {
		categoryQuery = categoryQuery.Where("owner_type = ? AND owner_id = ? AND type = ? AND name = ?", ownerType, assetOwnerID(ownerType, user), assetType, categoryName)
	} else {
		categoryQuery = categoryQuery.Where("owner_type = ? AND owner_id = ? AND department = ? AND type = ? AND name = ?", ownerType, assetOwnerID(ownerType, user), department, assetType, categoryName)
	}
	categoryQuery.Count(&categoryCount)
	if categoryCount > 0 {
		return true
	}

	var assetCount int64
	assetQuery := database.DB.Model(&model.Asset{})
	if ownerType == "system" {
		assetQuery = assetQuery.Where("owner_type = ? AND type = ? AND category = ?", ownerType, assetType, categoryName)
	} else if ownerType == "personal" {
		assetQuery = assetQuery.Where("owner_type = ? AND owner_id = ? AND type = ? AND category = ?", ownerType, assetOwnerID(ownerType, user), assetType, categoryName)
	} else {
		assetQuery = assetQuery.Where("owner_type = ? AND owner_id = ? AND department = ? AND type = ? AND category = ?", ownerType, assetOwnerID(ownerType, user), department, assetType, categoryName)
	}
	assetQuery.Count(&assetCount)
	return assetCount > 0
}

func nativeAssetOwnerName(asset model.Asset) string {
	switch asset.OwnerType {
	case "system":
		return "系统资产库"
	case "department":
		if asset.Department != "" {
			return asset.Department
		}
		return "部门资产库"
	default:
		return "我的资产"
	}
}

func assetPermissionText(asset model.Asset) string {
	if asset.OwnerType == "system" {
		return "系统资产"
	}
	if asset.OwnerType == "department" {
		if asset.Visibility == "public" {
			return "全员可见"
		}
		return "部门可用"
	}
	if asset.Visibility == "public" {
		return "全员可见"
	}
	if asset.Visibility == "department" {
		return "部门可见"
	}
	return "仅自己管理"
}

func parseAssetTags(value string) []string {
	return compactStrings(strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '，'
	}))
}

func marshalAssetTags(tags []string) string {
	data, _ := json.Marshal(compactStrings(tags))
	return string(data)
}

func unmarshalAssetTags(value string) []string {
	var tags []string
	if err := json.Unmarshal([]byte(value), &tags); err != nil {
		return nil
	}
	return compactStrings(tags)
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		item := strings.TrimSpace(value)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
		if len(result) >= 8 {
			break
		}
	}
	return result
}

func assetFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func assetSceneName(sceneID uint) string {
	switch sceneID {
	case 3:
		return "浴室"
	case 4:
		return "厨房"
	default:
		return ""
	}
}

func referenceGalleryAssetLabel(key string) string {
	switch strings.TrimSpace(key) {
	case "feature_pull_360":
		return "抽拉与旋转参考"
	case "feature_pull":
		return "抽拉参考"
	case "feature_360":
		return "旋转参考"
	case "water_flow_triple":
		return "三模式水流参考"
	case "water_flow_dual":
		return "双模式水流参考"
	case "water_flow_single":
		return "单模式水流参考"
	case "feature_selling_point":
		return "卖点图参考"
	case "dimension":
		return "尺寸图参考"
	case "family":
		return "全家福参考"
	case "multi_color_white_bg":
		return "白底多色参考"
	default:
		return "参考图"
	}
}

func formatAssetTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02 15:04")
}

func detectAssetFileTypeFromURL(url string) string {
	ext := strings.ToLower(filepath.Ext(url))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp":
		return "image"
	case ".stp", ".step":
		return "stp"
	case ".obj", ".fbx", ".glb", ".gltf":
		return "model"
	case ".mp4", ".mov", ".webm":
		return "video"
	case ".pdf":
		return "pdf"
	case ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		return "document"
	default:
		return "file"
	}
}

func assetTypeFromFileType(fileType string) string {
	switch fileType {
	case "image":
		return "图片"
	case "stp", "model":
		return "模型文件"
	case "video":
		return "视频"
	case "pdf", "document":
		return "文档"
	default:
		return "文件"
	}
}
