package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type dailyCallPermissions struct {
	CanRegister bool `json:"can_register"`
	CanReview   bool `json:"can_review"`
}

func dailyCallDepartmentPermissions(user model.User, department model.Department, found bool) dailyCallPermissions {
	for _, role := range user.Roles {
		if role.Name == "admin" {
			return dailyCallPermissions{true, true}
		}
	}
	if !found {
		return dailyCallPermissions{}
	}
	// Prefer a specific service subdepartment over its parent department.
	for _, name := range []string{department.SubName, department.Name} {
		switch strings.TrimSuffix(strings.TrimSpace(name), "部") {
		case "海外客服", "客服":
			return dailyCallPermissions{CanRegister: true, CanReview: true}
		}
	}
	return dailyCallPermissions{}
}

func dailyCallActor(c *gin.Context) (model.User, dailyCallPermissions, bool) {
	id, valid := c.Get("userID")
	userID, ok := id.(uint)
	if !valid || !ok || userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return model.User{}, dailyCallPermissions{}, false
	}
	var user model.User
	if err := database.DB.Preload("Roles").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "当前账号不可用"})
		return user, dailyCallPermissions{}, false
	}
	department, found := findDepartmentByUser(user)
	permissions := dailyCallDepartmentPermissions(user, department, found)
	if !permissions.CanRegister && !permissions.CanReview {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅海外客服、客服及管理员可访问来电登记"})
		return user, permissions, false
	}
	return user, permissions, true
}

type dailyCallRegistrationInput struct {
	Name         string `json:"name" binding:"max=200"`
	OrderNumber  string `json:"order_number" binding:"max=200"`
	Email        string `json:"email" binding:"omitempty,email,max=254"`
	PhoneNo      string `json:"phone_no" binding:"max=100"`
	BoughtIn     string `json:"bought_in" binding:"max=200"`
	Country      string `json:"country" binding:"max=100"`
	BriefOfIssue string `json:"brief_of_issue" binding:"max=10000"`
}

type dailyCallReviewInput struct {
	SKU           string `json:"sku" binding:"max=200"`
	ASIN          string `json:"asin" binding:"max=200"`
	IssueType     string `json:"issue_type" binding:"max=200"`
	QualityResult string `json:"quality_result" binding:"max=2000"`
	Reason        string `json:"reason" binding:"max=10000"`
}

// Reject fields owned by the other department, timestamps and forged actor IDs.
func decodeDailyCallInput(reader io.Reader, input any) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(input); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("expected a single JSON object")
	}
	return nil
}

func bindDailyCallInput(c *gin.Context, input any) bool {
	if err := decodeDailyCallInput(c.Request.Body, input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求包含无效字段或格式"})
		return false
	}
	return true
}

func ListDailyCalls(c *gin.Context) {
	user, permissions, ok := dailyCallActor(c)
	if !ok {
		return
	}
	page, size := positiveQueryInt(c, "page", 1), positiveQueryInt(c, "page_size", 20)
	if size > 100 {
		size = 100
	}
	query := dailyCallQuery(c)
	var filterErr error
	query, filterErr = filterDailyCallDate(query, c.Query("date"))
	if filterErr != nil {
		c.JSON(400, gin.H{"error": "无效的日期筛选"})
		return
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name ILIKE ? OR order_number ILIKE ? OR email ILIKE ? OR phone_no ILIKE ?", like, like, like, like)
	}
	switch c.Query("status") {
	case "pending":
		query = query.Where("reviewed_at IS NULL AND TRIM(COALESCE(quality_result, '')) = ''")
	case "reviewed":
		query = query.Where("reviewed_at IS NOT NULL OR TRIM(COALESCE(quality_result, '')) <> ''")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取来电登记失败"})
		return
	}
	items := []model.DailyCall{}
	query = query.Order("created_at DESC, id DESC")
	if c.Query("export") != "true" {
		query = query.Offset((page - 1) * size).Limit(size)
	}
	if err := query.Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取来电登记失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "total": total, "permissions": permissions, "locale": dailyCallLanguage(user)})
}

func GetDailyCall(c *gin.Context) {
	user, permissions, ok := dailyCallActor(c)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的登记 ID"})
		return
	}
	var item model.DailyCall
	if err := dailyCallQuery(c).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "登记不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取登记失败"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item, "permissions": permissions, "locale": dailyCallLanguage(user)})
}

func dailyCallLanguage(user model.User) string {
	for _, name := range []string{user.SubDepartment, user.Department} {
		switch strings.TrimSuffix(strings.TrimSpace(name), "部") {
		case "海外客服":
			return "en"
		case "客服":
			return "zh"
		}
	}
	return "zh"
}

func CreateDailyCall(c *gin.Context) {
	user, permissions, ok := dailyCallActor(c)
	if !ok || !requireDailyCallPermission(c, permissions.CanRegister) {
		return
	}
	input, ok := readDailyCallRegistration(c)
	if !ok {
		return
	}
	now, name := time.Now().Truncate(time.Microsecond), dailyCallActorName(user)
	item := model.DailyCall{
		Kind: dailyCallKind(c),
		Name: input.Name, OrderNumber: input.OrderNumber, Email: input.Email, PhoneNo: input.PhoneNo,
		BoughtIn: input.BoughtIn, Country: input.Country, BriefOfIssue: input.BriefOfIssue,
		CreatedByID: user.ID, CreatedByName: name, CreatedAt: now,
		RegistrationUpdatedByID: user.ID, RegistrationUpdatedByName: name, RegistrationUpdatedAt: now,
	}
	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "新增登记失败"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func UpdateDailyCallRegistration(c *gin.Context) {
	user, permissions, ok := dailyCallActor(c)
	if !ok || !requireDailyCallPermission(c, permissions.CanRegister) {
		return
	}
	input, ok := readDailyCallRegistration(c)
	if !ok {
		return
	}
	updateDailyCall(c, map[string]any{
		"name": input.Name, "order_number": input.OrderNumber, "email": input.Email, "phone_no": input.PhoneNo,
		"bought_in": input.BoughtIn, "country": input.Country, "brief_of_issue": input.BriefOfIssue,
	}, map[string]any{
		"registration_updated_by_id": user.ID, "registration_updated_by_name": dailyCallActorName(user), "registration_updated_at": time.Now(),
	})
}

func UpdateDailyCallReview(c *gin.Context) {
	user, permissions, ok := dailyCallActor(c)
	if !ok || !requireDailyCallPermission(c, permissions.CanReview) {
		return
	}
	if dailyCallKind(c) == "outbound" {
		var input struct {
			InformationRequested string `json:"information_requested" binding:"max=10000"`
			Result               string `json:"result" binding:"max=10000"`
		}
		if !bindDailyCallInput(c, &input) {
			return
		}
		if binding.Validator.ValidateStruct(input) != nil {
			c.JSON(400, gin.H{"error": "请求包含无效字段或格式"})
			return
		}
		updateDailyCall(c, map[string]any{"information_requested": strings.TrimSpace(input.InformationRequested), "result": strings.TrimSpace(input.Result)}, map[string]any{"reviewed_by_id": user.ID, "reviewed_by_name": dailyCallActorName(user), "reviewed_at": time.Now()})
		return
	}
	var input dailyCallReviewInput
	if !bindDailyCallInput(c, &input) {
		return
	}
	input.QualityResult, input.Reason = strings.TrimSpace(input.QualityResult), strings.TrimSpace(input.Reason)
	if err := binding.Validator.ValidateStruct(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写质检结果和理由，并检查内容长度"})
		return
	}
	updateDailyCall(c, map[string]any{
		"sku": input.SKU, "asin": input.ASIN, "issue_type": input.IssueType, "quality_result": input.QualityResult, "reason": input.Reason,
	}, map[string]any{
		"reviewed_by_id": user.ID, "reviewed_by_name": dailyCallActorName(user), "reviewed_at": time.Now(),
	})
}

func readDailyCallRegistration(c *gin.Context) (dailyCallRegistrationInput, bool) {
	var input dailyCallRegistrationInput
	if !bindDailyCallInput(c, &input) {
		return input, false
	}
	for _, value := range []*string{&input.Name, &input.OrderNumber, &input.Email, &input.PhoneNo, &input.BoughtIn, &input.Country, &input.BriefOfIssue} {
		*value = strings.TrimSpace(*value)
	}
	if err := binding.Validator.ValidateStruct(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写 Name 和 Brief of the issue，并检查邮箱格式和内容长度"})
		return input, false
	}
	return input, true
}

func requireDailyCallPermission(c *gin.Context, allowed bool) bool {
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "当前部门无权修改这部分信息"})
	}
	return allowed
}

func dailyCallActorName(user model.User) string {
	if name := strings.TrimSpace(user.Name); name != "" {
		return name
	}
	return user.Username
}

func updateDailyCall(c *gin.Context, values, attribution map[string]any) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的登记 ID"})
		return
	}
	// Compare business fields atomically; identical saves must not update attribution or timestamps.
	differences := make([]clause.Expression, 0, len(values))
	for column, value := range values {
		differences = append(differences, clause.Expr{SQL: column + " IS DISTINCT FROM ?", Vars: []any{value}})
	}
	for column, value := range attribution {
		values[column] = value
	}
	result := dailyCallQuery(c).Where("id = ?", id).Where(clause.Or(differences...)).Updates(values)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存登记失败"})
		return
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := dailyCallQuery(c).Where("id = ?", id).Count(&count).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存登记失败"})
			return
		}
		if count == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "登记不存在"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"changed": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "保存成功", "changed": true})
}
