package handler

import (
	"net/http"
	"strconv"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
)

/** buildMenuTree 将平铺的菜单列表组装成两级树形结构 */
func buildMenuTree(menus []model.Menu) []*model.Menu {
	menuMap := make(map[uint]*model.Menu, len(menus))
	for i := range menus {
		menuMap[menus[i].ID] = &menus[i]
	}

	var roots []*model.Menu
	for i := range menus {
		if menus[i].ParentID == 0 {
			roots = append(roots, &menus[i])
		} else if parent, ok := menuMap[menus[i].ParentID]; ok {
			parent.Children = append(parent.Children, &menus[i])
		}
	}
	return roots
}

/** isHiddenAdminMenu 后台管理系统中隐藏运营/设计业务功能与财务菜单 */
func isHiddenAdminMenu(m model.Menu) bool {
	path := strings.ToLower(m.Path)
	title := m.Title

	pathKeywords := []string{"product-image", "product-image-task", "multi-angle"}
	titleKeywords := []string{"产品图生成", "产品套图", "多角度生成"}

	for _, k := range pathKeywords {
		if strings.Contains(path, k) {
			return true
		}
	}
	for _, k := range titleKeywords {
		if strings.Contains(title, k) {
			return true
		}
	}
	return false
}

/** filterHiddenAdminMenus 过滤后台不应展示的菜单 */
func filterHiddenAdminMenus(menus []model.Menu) []model.Menu {
	filtered := make([]model.Menu, 0, len(menus))
	for _, m := range menus {
		if isHiddenAdminMenu(m) {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

func GetMenus(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	// Load user with roles
	var user model.User
	if err := database.DB.Preload("Roles").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user"})
		return
	}

	// Admin gets all menus as tree（后台隐藏运营/设计业务功能与财务菜单）
	for _, role := range user.Roles {
		if role.Name == "admin" {
			var menus []model.Menu
			database.DB.Order("sort asc").Find(&menus)
			menus = filterHiddenAdminMenus(menus)
			c.JSON(http.StatusOK, buildMenuTree(menus))
			return
		}
	}

	// Non-admin: collect menus from all assigned roles (deduplicated)
	menuSet := make(map[uint]model.Menu)
	for _, role := range user.Roles {
		var roleWithMenus model.Role
		if err := database.DB.Preload("Menus").First(&roleWithMenus, role.ID).Error; err != nil {
			continue
		}
		for _, m := range roleWithMenus.Menus {
			menuSet[m.ID] = m
		}
	}

	var menuIDs []uint
	for id := range menuSet {
		menuIDs = append(menuIDs, id)
	}

	var menus []model.Menu
	if len(menuIDs) > 0 {
		/** 非管理员权限内可能只有子菜单，需要把父菜单也一起加载以正确组树 */
		database.DB.Where("id IN ? OR (parent_id = 0 AND id IN (SELECT parent_id FROM menus WHERE id IN ?))", menuIDs, menuIDs).Order("sort asc").Find(&menus)
	}

	menus = filterHiddenAdminMenus(menus)
	c.JSON(http.StatusOK, buildMenuTree(menus))
}

/** GetAllMenusAdmin 返回所有菜单列表（管理端），支持分页 */
func GetAllMenusAdmin(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	var total int64
	database.DB.Model(&model.Menu{}).Count(&total)

	var menus []model.Menu
	database.DB.Order("sort asc").Offset(offset).Limit(pageSize).Find(&menus)
	menus = filterHiddenAdminMenus(menus)

	c.JSON(http.StatusOK, gin.H{
		"data":  menus,
		"total": total,
	})
}

/** CreateMenu 创建新菜单；一级分类菜单 path 可为空，二级菜单需指定 parent_id */
func CreateMenu(c *gin.Context) {
	var req struct {
		Title    string `json:"title" binding:"required"`
		Path     string `json:"path"`
		Icon     string `json:"icon"`
		Sort     int    `json:"sort"`
		ParentID uint   `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	/** 指定了父菜单时，验证父菜单存在且本身为一级菜单（不允许三级嵌套） */
	if req.ParentID != 0 {
		var parent model.Menu
		if err := database.DB.First(&parent, req.ParentID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "父菜单不存在"})
			return
		}
		if parent.ParentID != 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持三级及以上菜单嵌套"})
			return
		}
	}

	/** 仅在路径非空时才做唯一性检查，空路径允许多个（用于一级分类菜单） */
	if req.Path != "" {
		var existing model.Menu
		if err := database.DB.Where("path = ?", req.Path).First(&existing).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "该路径已存在"})
			return
		}
	}

	menu := model.Menu{
		Title:    req.Title,
		Path:     req.Path,
		Icon:     req.Icon,
		Sort:     req.Sort,
		ParentID: req.ParentID,
	}
	if err := database.DB.Create(&menu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建菜单失败"})
		return
	}

	c.JSON(http.StatusOK, menu)
}

/** UpdateMenu 更新菜单信息 */
func UpdateMenu(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的菜单 ID"})
		return
	}

	var menu model.Menu
	if err := database.DB.First(&menu, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "菜单不存在"})
		return
	}

	var req struct {
		Title    string `json:"title"`
		Path     string `json:"path"`
		Icon     string `json:"icon"`
		Sort     int    `json:"sort"`
		ParentID uint   `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	/** 若指定了父菜单，验证父菜单存在且不是自身，且父菜单本身是一级菜单 */
	if req.ParentID != 0 {
		if req.ParentID == uint(id) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不能将自身设置为父菜单"})
			return
		}
		var parent model.Menu
		if err := database.DB.First(&parent, req.ParentID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "父菜单不存在"})
			return
		}
		if parent.ParentID != 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支持三级及以上菜单嵌套"})
			return
		}
	}

	/** 若路径有变化，检查新路径是否已被其他菜单占用 */
	if req.Path != "" && req.Path != menu.Path {
		var existing model.Menu
		if err := database.DB.Where("path = ? AND id != ?", req.Path, id).First(&existing).Error; err == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "该路径已被其他菜单使用"})
			return
		}
	}

	updates := map[string]interface{}{
		"title":     req.Title,
		"path":      req.Path,
		"icon":      req.Icon,
		"sort":      req.Sort,
		"parent_id": req.ParentID,
	}
	if err := database.DB.Model(&menu).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新菜单失败"})
		return
	}

	database.DB.First(&menu, id)
	c.JSON(http.StatusOK, menu)
}

/** DeleteMenu 删除菜单，同时清理角色关联 */
func DeleteMenu(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的菜单 ID"})
		return
	}

	var menu model.Menu
	if err := database.DB.First(&menu, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "菜单不存在"})
		return
	}

	/** 检查是否有子菜单，有则阻止删除，避免孤儿数据 */
	var childCount int64
	database.DB.Model(&model.Menu{}).Where("parent_id = ?", id).Count(&childCount)
	if childCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先删除该菜单下的所有子菜单"})
		return
	}

	/** 先清理该菜单与所有角色的关联，再删除菜单本身 */
	if err := database.DB.Model(&menu).Association("Roles").Clear(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清理菜单角色关联失败"})
		return
	}

	if err := database.DB.Delete(&menu).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除菜单失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
