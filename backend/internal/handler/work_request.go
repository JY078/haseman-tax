package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const workRequestImageMaxBytes = 30 * 1024 * 1024

type workRequestReference struct {
	URL  string `json:"url"`
	Note string `json:"note"`
}

type videoRequestContent struct {
	DurationSeconds    float64                 `json:"duration_seconds"`
	ShotGroups         []videoRequestShotGroup `json:"shot_groups"`
	Shots              []videoRequestShot      `json:"shots,omitempty"`
	AcceptanceCriteria string                  `json:"acceptance_criteria"`
}

type videoRequestShotGroup struct {
	Title string             `json:"title"`
	Note  string             `json:"note"`
	Shots []videoRequestShot `json:"shots"`
}

type videoRequestShot struct {
	DurationSeconds    float64                `json:"duration_seconds"`
	Scene              string                 `json:"scene"`
	Subtitle           string                 `json:"subtitle"`
	Note               string                 `json:"note"`
	DisplayRequirement string                 `json:"display_requirement"`
	Dialogue           string                 `json:"dialogue"`
	References         []workRequestReference `json:"references"`
}

type workRequestInput struct {
	Type    string          `json:"type"`
	Title   string          `json:"title"`
	Content json.RawMessage `json:"content"`
}

func CreateWorkRequest(c *gin.Context) {
	userID, ok := workRequestUserID(c)
	if !ok {
		return
	}

	var input workRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请完整填写需求信息"})
		return
	}
	content, err := validateWorkRequestInput(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	request := model.WorkRequest{
		Type: "video", Title: strings.TrimSpace(input.Title), Status: "pending",
		CreatorID: userID, Progress: 0, ProgressBeforeReview: 10, Content: content,
		DeliveryPaths: json.RawMessage(`[]`),
	}
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		day := time.Now().In(time.Local).Format("20060102")
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "work-request-"+day).Error; err != nil {
			return err
		}
		var latestWorkNo string
		start := startOfLocalDay(time.Now())
		if err := tx.Model(&model.WorkRequest{}).
			Where("created_at >= ? AND created_at < ?", start, start.AddDate(0, 0, 1)).
			Order("work_no DESC").Limit(1).Pluck("work_no", &latestWorkNo).Error; err != nil {
			return err
		}
		request.WorkNo = formatWorkRequestNo(time.Now(), workRequestNextSequence(latestWorkNo))
		return tx.Create(&request).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建需求失败"})
		return
	}
	loadWorkRequestRelations(&request)
	c.JSON(http.StatusCreated, request)
}

func ListWorkRequests(c *gin.Context) {
	page := positiveQueryInt(c, "page", 1)
	pageSize := positiveQueryInt(c, "page_size", 20)
	if pageSize > 100 {
		pageSize = 100
	}
	query := database.DB.Model(&model.WorkRequest{}).Where("type = ?", "video")
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("work_no ILIKE ? OR title ILIKE ?", like, like)
	}
	for _, filter := range []struct{ query, column string }{
		{"status", "status"}, {"creator_id", "creator_id"}, {"assignee_id", "assignee_id"},
	} {
		if value := strings.TrimSpace(c.Query(filter.query)); value != "" {
			query = query.Where(filter.column+" = ?", value)
		}
	}
	var total int64
	query.Count(&total)
	var requests []model.WorkRequest
	if err := query.Preload("Creator").Preload("Assignee").Order("created_at DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&requests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取需求列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": requests, "total": total, "page": page, "page_size": pageSize})
}

func GetWorkRequest(c *gin.Context) {
	request, ok := findWorkRequest(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, request)
}

func UpdateWorkRequest(c *gin.Context) {
	request, ok := findWorkRequest(c)
	if !ok {
		return
	}
	userID, _ := workRequestUserID(c)
	isAdmin := workRequestIsAdmin(userID)
	if request.Status != "pending" || (request.CreatorID != userID && !isAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有创建人可以编辑待受理需求"})
		return
	}
	var input workRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请完整填写需求信息"})
		return
	}
	content, err := validateWorkRequestInput(input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	request.Type, request.Title, request.Content = "video", strings.TrimSpace(input.Title), content
	if err := database.DB.Save(request).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存需求失败"})
		return
	}
	loadWorkRequestRelations(request)
	c.JSON(http.StatusOK, request)
}

func ListWorkRequestUsers(c *gin.Context) {
	var users []model.User
	if err := database.DB.Select("id", "username", "name").Order("name ASC, username ASC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}
	c.JSON(http.StatusOK, users)
}

func AssignWorkRequest(c *gin.Context) {
	request, ok := findWorkRequest(c)
	if !ok {
		return
	}
	userID, _ := workRequestUserID(c)
	if !workRequestIsAdmin(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以分配需求"})
		return
	}
	if request.Status != "pending" && request.Status != "in_progress" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前状态不能分配处理人"})
		return
	}
	var input struct {
		AssigneeID uint `json:"assignee_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.AssigneeID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择处理人"})
		return
	}
	var count int64
	if database.DB.Model(&model.User{}).Where("id = ?", input.AssigneeID).Count(&count); count == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "处理人不存在"})
		return
	}
	request.AssigneeID = &input.AssigneeID
	if request.Status == "pending" {
		request.Status, request.Progress, request.ProgressBeforeReview = "in_progress", 10, 10
	}
	request.RejectionReason = ""
	saveAndRespondWorkRequest(c, request, "分配需求失败")
}

func ReturnWorkRequestToPending(c *gin.Context) {
	request, ok := findWorkRequest(c)
	if !ok {
		return
	}
	userID, _ := workRequestUserID(c)
	if !workRequestIsAdmin(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以退回修改"})
		return
	}
	if request.Status != "in_progress" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只有进行中的需求可以退回修改"})
		return
	}
	request.Status, request.Progress, request.AssigneeID = "pending", 0, nil
	saveAndRespondWorkRequest(c, request, "退回需求失败")
}

func UpdateWorkRequestProgress(c *gin.Context) {
	request, ok := findWorkRequest(c)
	if !ok {
		return
	}
	userID, _ := workRequestUserID(c)
	if !workRequestCanHandle(request, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有处理人可以更新进度"})
		return
	}
	if request.Status != "in_progress" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只有进行中的需求可以更新进度"})
		return
	}
	var input struct {
		Progress int `json:"progress"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Progress < 10 || input.Progress > 89 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "进行中进度必须在 10% 到 89% 之间"})
		return
	}
	request.Progress, request.ProgressBeforeReview = input.Progress, input.Progress
	saveAndRespondWorkRequest(c, request, "更新进度失败")
}

func SubmitWorkRequestReview(c *gin.Context) {
	request, ok := findWorkRequest(c)
	if !ok {
		return
	}
	userID, _ := workRequestUserID(c)
	if !workRequestCanHandle(request, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有处理人可以提交验收"})
		return
	}
	if request.Status != "in_progress" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只有进行中的需求可以提交验收"})
		return
	}
	var input struct {
		DeliveryNote  string   `json:"delivery_note"`
		DeliveryPaths []string `json:"delivery_paths"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.DeliveryNote) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写交付说明"})
		return
	}
	paths := cleanNonEmptyStrings(input.DeliveryPaths)
	if len(paths) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请至少填写一个共享盘文件路径"})
		return
	}
	encoded, _ := json.Marshal(paths)
	request.ProgressBeforeReview = request.Progress
	request.Progress, request.Status = 90, "pending_review"
	request.DeliveryNote, request.DeliveryPaths = strings.TrimSpace(input.DeliveryNote), encoded
	saveAndRespondWorkRequest(c, request, "提交验收失败")
}

func ApproveWorkRequest(c *gin.Context) {
	request, ok := findWorkRequest(c)
	if !ok {
		return
	}
	userID, _ := workRequestUserID(c)
	if request.CreatorID != userID && !workRequestIsAdmin(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有创建人可以确认验收"})
		return
	}
	if request.Status != "pending_review" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前需求不在待验收状态"})
		return
	}
	request.Status, request.Progress, request.RejectionReason = "completed", 100, ""
	saveAndRespondWorkRequest(c, request, "验收需求失败")
}

func RejectWorkRequest(c *gin.Context) {
	request, ok := findWorkRequest(c)
	if !ok {
		return
	}
	userID, _ := workRequestUserID(c)
	if request.CreatorID != userID && !workRequestIsAdmin(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有创建人可以退回验收"})
		return
	}
	if request.Status != "pending_review" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前需求不在待验收状态"})
		return
	}
	var input struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Reason) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写验收退回原因"})
		return
	}
	request.Status, request.Progress = "in_progress", request.ProgressBeforeReview
	if request.Progress < 10 || request.Progress > 89 {
		request.Progress = 10
	}
	request.RejectionReason = strings.TrimSpace(input.Reason)
	saveAndRespondWorkRequest(c, request, "退回验收失败")
}

func CancelWorkRequest(c *gin.Context) {
	request, ok := findWorkRequest(c)
	if !ok {
		return
	}
	userID, _ := workRequestUserID(c)
	isAdmin := workRequestIsAdmin(userID)
	if !isAdmin && (request.CreatorID != userID || request.Status != "pending") {
		c.JSON(http.StatusForbidden, gin.H{"error": "创建人只能取消自己的待受理需求"})
		return
	}
	if request.Status == "completed" || request.Status == "canceled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前状态不能取消"})
		return
	}
	request.Status = "canceled"
	saveAndRespondWorkRequest(c, request, "取消需求失败")
}

func DeleteWorkRequest(c *gin.Context) {
	request, ok := findWorkRequest(c)
	if !ok {
		return
	}
	userID, _ := workRequestUserID(c)
	if !workRequestIsAdmin(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有管理员可以删除需求"})
		return
	}
	if err := database.DB.Delete(request).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除需求失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "需求已删除"})
}

func UploadWorkRequestReference(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(workRequestImageMaxBytes + 1024); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析图片失败"})
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
	}
	fileHeader, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择图片"})
		return
	}
	if fileHeader.Size > workRequestImageMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单张图片不能超过 30MB"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取图片失败"})
		return
	}
	defer file.Close()
	mimeType, reader, err := validateWorkRequestImage(file, fileHeader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	url, err := storage.SavePromptImageReader(reader, mimeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存图片失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

func validateWorkRequestImage(file multipart.File, header *multipart.FileHeader) (string, io.Reader, error) {
	first := make([]byte, 512)
	n, err := file.Read(first)
	if err != nil && err != io.EOF {
		return "", nil, fmt.Errorf("读取图片失败")
	}
	first = first[:n]
	mimeType := http.DetectContentType(first)
	if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/webp" {
		return "", nil, fmt.Errorf("仅支持 JPG、PNG、WebP 图片")
	}
	return mimeType, io.MultiReader(bytes.NewReader(first), file), nil
}

func validateWorkRequestInput(input workRequestInput) (json.RawMessage, error) {
	input.Type, input.Title = strings.TrimSpace(input.Type), strings.TrimSpace(input.Title)
	if input.Title == "" {
		return nil, fmt.Errorf("请填写标题")
	}
	if len([]rune(input.Title)) > 200 {
		return nil, fmt.Errorf("标题不能超过 200 个字")
	}
	if input.Type != "" && input.Type != "video" {
		return nil, fmt.Errorf("目前仅支持视频需求")
	}
	var content videoRequestContent
	if err := json.Unmarshal(input.Content, &content); err != nil {
		return nil, fmt.Errorf("视频需求内容格式错误")
	}
	if content.DurationSeconds <= 0 {
		return nil, fmt.Errorf("请选择视频总时长")
	}
	var total float64
	if len(content.ShotGroups) > 0 {
		for groupIndex, group := range content.ShotGroups {
			if strings.TrimSpace(group.Title) == "" {
				return nil, fmt.Errorf("请填写章节 %d 的名称", groupIndex+1)
			}
			if len(group.Shots) == 0 {
				return nil, fmt.Errorf("章节 %d 至少需要一个画面", groupIndex+1)
			}
			for shotIndex, shot := range group.Shots {
				if shot.DurationSeconds <= 0 || strings.TrimSpace(shot.Scene) == "" {
					return nil, fmt.Errorf("请完整填写章节 %d 的画面 %d", groupIndex+1, shotIndex+1)
				}
				total += shot.DurationSeconds
			}
		}
	} else {
		if len(content.Shots) == 0 {
			return nil, fmt.Errorf("请至少添加一个画面")
		}
		for i, shot := range content.Shots {
			if shot.DurationSeconds <= 0 || strings.TrimSpace(shot.DisplayRequirement) == "" {
				return nil, fmt.Errorf("请完整填写画面 %d", i+1)
			}
			total += shot.DurationSeconds
		}
	}
	if total-content.DurationSeconds > 0.001 || content.DurationSeconds-total > 0.001 {
		return nil, fmt.Errorf("画面时长合计必须等于视频总时长")
	}
	var normalized any
	_ = json.Unmarshal(input.Content, &normalized)
	encoded, _ := json.Marshal(normalized)
	return encoded, nil
}

func workRequestUserID(c *gin.Context) (uint, bool) {
	value, ok := c.Get("userID")
	userID, valid := value.(uint)
	if !ok || !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return 0, false
	}
	return userID, true
}

func workRequestIsAdmin(userID uint) bool {
	var count int64
	database.DB.Table("user_roles").Joins("JOIN roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ? AND roles.name = ?", userID, "admin").Count(&count)
	return count > 0
}

func workRequestCanHandle(request *model.WorkRequest, userID uint) bool {
	return workRequestIsAdmin(userID) || (request.AssigneeID != nil && *request.AssigneeID == userID)
}

func findWorkRequest(c *gin.Context) (*model.WorkRequest, bool) {
	var request model.WorkRequest
	if err := database.DB.Preload("Creator").Preload("Assignee").Where("type = ?", "video").First(&request, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "需求不存在"})
		return nil, false
	}
	return &request, true
}

func loadWorkRequestRelations(request *model.WorkRequest) {
	database.DB.Preload("Creator").Preload("Assignee").First(request, request.ID)
}

func saveAndRespondWorkRequest(c *gin.Context, request *model.WorkRequest, errorMessage string) {
	if err := database.DB.Save(request).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorMessage})
		return
	}
	loadWorkRequestRelations(request)
	c.JSON(http.StatusOK, request)
}

func positiveQueryInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.DefaultQuery(key, strconv.Itoa(fallback)))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func cleanNonEmptyStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func startOfLocalDay(value time.Time) time.Time {
	local := value.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
}

func formatWorkRequestNo(value time.Time, sequence int) string {
	return fmt.Sprintf("REQ-%s-%03d", value.In(time.Local).Format("20060102"), sequence)
}

func workRequestNextSequence(latestWorkNo string) int {
	parts := strings.Split(latestWorkNo, "-")
	if len(parts) != 3 {
		return 1
	}
	sequence, err := strconv.Atoi(parts[2])
	if err != nil || sequence < 1 {
		return 1
	}
	return sequence + 1
}
