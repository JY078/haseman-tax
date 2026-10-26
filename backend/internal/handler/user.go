package handler

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/cache"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// GetUsers returns paginated user list with roles
func GetUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int64
	database.DB.Model(&model.User{}).Count(&total)

	var users []model.User
	if err := database.DB.Preload("Roles").Order("id asc").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	userPointers := make([]*model.User, 0, len(users))
	for index := range users {
		userPointers = append(userPointers, &users[index])
	}
	loadUserOrderPlatforms(userPointers...)
	if err := loadUserOnlineStatus(users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch online status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// CreateUser creates a new user
func CreateUser(c *gin.Context) {
	var input struct {
		Username           string              `json:"username" binding:"required"`
		Name               string              `json:"name"`
		DepartmentID       uint                `json:"department_id"`
		Platform           string              `json:"platform"`
		Department         string              `json:"department"`
		SubDepartment      string              `json:"sub_department"`
		Email              string              `json:"email"`
		Password           string              `json:"password" binding:"required"`
		RoleIDs            []uint              `json:"role_ids"`
		FinancePermissions map[string][]string `json:"finance_permissions"`
		LongLogin          *bool               `json:"long_login"`
		Enabled            *bool               `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateFinancePermissions(input.FinancePermissions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	applyDepartmentSelection(&input.Platform, &input.Department, &input.SubDepartment, input.DepartmentID)

	user := model.User{
		Username:           input.Username,
		Name:               input.Name,
		Platform:           input.Platform,
		Department:         input.Department,
		SubDepartment:      input.SubDepartment,
		Email:              input.Email,
		Password:           string(hashed),
		FinancePermissions: normalizeFinancePermissions(input.FinancePermissions),
	}
	if input.LongLogin != nil {
		user.LongLogin = *input.LongLogin
	}
	if input.Enabled != nil {
		user.Enabled = *input.Enabled
	}
	if user.Email == "" {
		user.Email = fmt.Sprintf("%s@admin.local", input.Username)
	}
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
		return
	}
	if len(input.RoleIDs) > 0 {
		var roles []model.Role
		database.DB.Where("id IN ?", input.RoleIDs).Find(&roles)
		database.DB.Model(&user).Association("Roles").Replace(roles)
	}

	database.DB.Preload("Roles").First(&user, user.ID)
	loadUserOrderPlatforms(&user)

	c.JSON(http.StatusOK, user)
}

// UpdateUser updates user info (email, password, roles)
func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user model.User
	if err := database.DB.Preload("Roles").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	actorIDValue, _ := c.Get("userID")
	actorID, _ := actorIDValue.(uint)
	if isPrimaryAdmin(user) && actorID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "主管理员账号只能由本人编辑"})
		return
	}

	var input struct {
		Name               string              `json:"name"`
		DepartmentID       uint                `json:"department_id"`
		Platform           string              `json:"platform"`
		Department         string              `json:"department"`
		SubDepartment      string              `json:"sub_department"`
		Password           string              `json:"password"`
		RoleIDs            []uint              `json:"role_ids"`
		FinancePermissions map[string][]string `json:"finance_permissions"`
		LongLogin          *bool               `json:"long_login"`
		Enabled            *bool               `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateFinancePermissions(input.FinancePermissions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	longLoginChanged := input.LongLogin != nil && user.LongLogin != *input.LongLogin

	if input.Name != "" {
		user.Name = input.Name
	}
	applyDepartmentSelection(&input.Platform, &input.Department, &input.SubDepartment, input.DepartmentID)
	user.Platform = input.Platform
	user.Department = input.Department
	user.SubDepartment = input.SubDepartment
	user.FinancePermissions = normalizeFinancePermissions(input.FinancePermissions)
	if input.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		user.Password = string(hashed)
	}
	if input.LongLogin != nil {
		user.LongLogin = *input.LongLogin
	}
	if input.Enabled != nil {
		user.Enabled = *input.Enabled
	}
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user: " + err.Error()})
		return
	}
	if longLoginChanged {
		if err := invalidateWebSessions(user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to invalidate user sessions: " + err.Error()})
			return
		}
	}
	if input.Enabled != nil && !user.Enabled {
		if err := invalidateUserSessions(user.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to invalidate user sessions: " + err.Error()})
			return
		}
	}

	// Replace roles
	var roles []model.Role
	if len(input.RoleIDs) > 0 {
		database.DB.Where("id IN ?", input.RoleIDs).Find(&roles)
	}
	database.DB.Model(&user).Association("Roles").Replace(roles)

	database.DB.Preload("Roles").First(&user, user.ID)
	loadUserOrderPlatforms(&user)
	c.JSON(http.StatusOK, user)
}

func isPrimaryAdmin(user model.User) bool {
	if strings.EqualFold(strings.TrimSpace(user.Username), "admin") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(user.Email), "admin@example.com")
}

var financePermissionPlatforms = map[string]struct{}{"amazon": {}, "tiktok": {}, "walmart": {}, "customs": {}, "exchange_rates": {}}
var financePermissionActions = map[string]struct{}{"view": {}, "create": {}, "import": {}, "export": {}}

func normalizeFinancePermissions(values map[string][]string) map[string][]string {
	result := make(map[string][]string)
	for platform, actions := range values {
		platform = strings.ToLower(strings.TrimSpace(platform))
		if _, ok := financePermissionPlatforms[platform]; !ok {
			continue
		}
		seen := map[string]bool{}
		for _, action := range actions {
			action = strings.ToLower(strings.TrimSpace(action))
			if _, ok := financePermissionActions[action]; ok && !seen[action] {
				result[platform] = append(result[platform], action)
				seen[action] = true
			}
		}
		sort.Strings(result[platform])
	}
	return result
}

func validateFinancePermissions(values map[string][]string) error {
	for platform, actions := range values {
		if _, ok := financePermissionPlatforms[strings.ToLower(strings.TrimSpace(platform))]; !ok {
			return fmt.Errorf("包含不支持的财务平台权限")
		}
		for _, action := range actions {
			if _, ok := financePermissionActions[strings.ToLower(strings.TrimSpace(action))]; !ok {
				return fmt.Errorf("包含不支持的财务操作权限")
			}
		}
	}
	return nil
}

// DeleteUser deletes a user (cannot delete self or the last admin)
func DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// Prevent deleting self
	selfIDVal, _ := c.Get("userID")
	selfID := selfIDVal.(uint)
	if strconv.Itoa(int(selfID)) == id {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete yourself"})
		return
	}

	var user model.User
	if err := database.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	database.DB.Model(&user).Association("Roles").Clear()
	database.DB.Where("user_id = ?", user.ID).Delete(&model.UserOrderPlatform{})
	database.DB.Delete(&user)
	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}

func replaceUserOrderPlatforms(userID uint, values []string) error {
	platforms := normalizeUserOrderPlatforms(values)
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserOrderPlatform{}).Error; err != nil {
			return err
		}
		rows := make([]model.UserOrderPlatform, 0, len(platforms))
		for _, platform := range platforms {
			rows = append(rows, model.UserOrderPlatform{UserID: userID, Platform: platform})
		}
		if len(rows) == 0 {
			return tx.Model(&model.User{}).Where("id = ?", userID).Update("order_platforms_configured", true).Error
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", userID).Update("order_platforms_configured", true).Error
	})
}

func validateUserOrderPlatforms(values []string) error {
	if len(normalizeUserOrderPlatforms(values)) != len(compactOrderPlatformInputs(values)) {
		return fmt.Errorf("负责平台包含不支持的值")
	}
	return nil
}

func loadUserOrderPlatforms(users ...*model.User) {
	if len(users) == 0 {
		return
	}
	ids := make([]uint, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	var rows []model.UserOrderPlatform
	database.DB.Where("user_id IN ?", ids).Order("platform asc").Find(&rows)
	byUser := make(map[uint][]string)
	for _, row := range rows {
		byUser[row.UserID] = append(byUser[row.UserID], row.Platform)
	}
	for _, user := range users {
		user.OrderPlatforms = byUser[user.ID]
	}
}

func normalizeUserOrderPlatforms(values []string) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	for _, value := range values {
		platform := normalizeOrderPlatform(value)
		if platform == "" {
			continue
		}
		if _, exists := seen[platform]; exists {
			continue
		}
		seen[platform] = struct{}{}
		result = append(result, platform)
	}
	sort.Strings(result)
	return result
}

func compactOrderPlatformInputs(values []string) []string {
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[strings.ToUpper(value)] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	return result
}

func applyDepartmentSelection(platform, department, subDepartment *string, departmentID uint) {
	if departmentID == 0 {
		return
	}
	var item model.Department
	if err := database.DB.First(&item, departmentID).Error; err != nil {
		return
	}
	*platform = item.Platform
	*department = item.Name
	*subDepartment = item.SubName
}

func invalidateWebSessions(userID uint) error {
	var sessions []model.UserSession
	if err := database.DB.Where("user_id = ? AND client_type = ?", userID, "web").Find(&sessions).Error; err != nil {
		return err
	}
	if err := database.DB.Where("user_id = ? AND client_type = ?", userID, "web").Delete(&model.UserSession{}).Error; err != nil {
		return err
	}
	for _, session := range sessions {
		cache.InvalidateSession(session.TokenHash)
	}
	return nil
}

func invalidateUserSessions(userID uint) error {
	var sessions []model.UserSession
	if err := database.DB.Where("user_id = ?", userID).Find(&sessions).Error; err != nil {
		return err
	}
	if err := database.DB.Where("user_id = ?", userID).Delete(&model.UserSession{}).Error; err != nil {
		return err
	}
	for _, session := range sessions {
		cache.InvalidateSession(session.TokenHash)
	}
	return nil
}

// Online requires an enabled account and an unexpired session with a recent heartbeat.
func loadUserOnlineStatus(users []model.User) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}
	var sessions []model.UserSession
	now := time.Now()
	if err := database.DB.Select("DISTINCT user_id").
		Where("user_id IN ? AND expires_at > ? AND last_active_at > ?", ids, now, now.Add(-heartbeatWindow)).
		Find(&sessions).Error; err != nil {
		return err
	}
	online := make(map[uint]bool, len(sessions))
	for _, session := range sessions {
		online[session.UserID] = true
	}
	for index := range users {
		users[index].Online = users[index].Enabled && online[users[index].ID]
	}
	return nil
}
