package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	sceneUserRunningImageLimitSettingKey = "scene_generation_user_running_image_limit"
	sceneUserRunningImageLimitDefault    = 8
	sceneTaskQueuePollInterval           = 2 * time.Second
	sceneTaskQueueMaxWait                = 30 * time.Minute
)

type sceneTaskCreateRequest struct {
	ProductCategory             string `json:"productCategory" binding:"required"`
	StyleTagID                  uint   `json:"styleTagId"`
	StyleReferenceID            uint   `json:"styleReferenceId"`
	IncludePerson               bool   `json:"includePerson"`
	CountryCode                 string `json:"countryCode"`
	CountryName                 string `json:"countryName"`
	ModelName                   string `json:"modelName"`
	AspectRatio                 string `json:"aspectRatio"`
	ProductColorFamily          string `json:"productColorFamily"`
	ProductColorFamilyImageURL  string `json:"productColorFamilyImageUrl"`
	ProductColorFamilyImageData string `json:"productColorFamilyImageData"`
	ProductColorFamilyImageMime string `json:"productColorFamilyImageMimeType"`
	ImageSize                   string `json:"imageSize"`
	ImageCount                  int    `json:"imageCount"`
	ExtraPrompt                 string `json:"extraPrompt"`
	Lens                        string `json:"lens"`
	SubjectRatio                string `json:"subjectRatio"`
	CompositionTemplate         string `json:"compositionTemplate"`
	PrimaryToneHex              string `json:"primaryToneHex"`
	FrameScope                  string `json:"frameScope"`
	LightingMood                string `json:"lightingMood"`
	SpaceMood                   string `json:"spaceMood"`
	MaterialFocus               string `json:"materialFocus"`
	StylingDensity              string `json:"stylingDensity"`
	BrightnessPreference        string `json:"brightnessPreference"`
	ProductImageData            string `json:"productImageData"`
	ProductImageURL             string `json:"productImageUrl"`
	ProductImageMime            string `json:"productImageMimeType"`
	StyleReferenceImageURL      string `json:"styleReferenceImageUrl"`
}

type sceneImageCandidateResponse struct {
	Sort                  int    `json:"sort"`
	ImageURL              string `json:"imageUrl"`
	CandidateImageURL     string `json:"candidateImageUrl,omitempty"`
	CandidateStatus       string `json:"candidateStatus,omitempty"`
	CandidateErrorMessage string `json:"candidateErrorMessage,omitempty"`
	CandidatePrompt       string `json:"candidatePrompt,omitempty"`
}

type sceneTaskResponse struct {
	TaskID                     string                        `json:"taskId"`
	Status                     string                        `json:"status"`
	ImageURLs                  []string                      `json:"imageUrls"`
	Images                     []sceneImageCandidateResponse `json:"images"`
	ErrorMessage               string                        `json:"errorMessage,omitempty"`
	CreatedAt                  string                        `json:"createdAt"`
	IsSavedToDesign            bool                          `json:"isSavedToDesign"`
	SavedAt                    string                        `json:"savedAt,omitempty"`
	ReferenceImageURL          string                        `json:"referenceImageUrl,omitempty"`
	ReferenceImageTitle        string                        `json:"referenceImageTitle,omitempty"`
	StyleName                  string                        `json:"styleName"`
	ProductCategory            string                        `json:"productCategory"`
	CountryCode                string                        `json:"countryCode,omitempty"`
	CountryName                string                        `json:"countryName,omitempty"`
	ProductColorFamily         string                        `json:"productColorFamily"`
	ProductColorFamilyImageURL string                        `json:"productColorFamilyImageUrl,omitempty"`
	AspectRatio                string                        `json:"aspectRatio"`
	ProductImageURL            string                        `json:"productImageUrl"`
	ImageCount                 int                           `json:"imageCount"`
	SubjectRatio               string                        `json:"subjectRatio,omitempty"`
	CompositionTemplate        string                        `json:"compositionTemplate,omitempty"`
	PrimaryToneHex             string                        `json:"primaryToneHex,omitempty"`
	FrameScope                 string                        `json:"frameScope,omitempty"`
	LightingMood               string                        `json:"lightingMood,omitempty"`
	SpaceMood                  string                        `json:"spaceMood,omitempty"`
	MaterialFocus              string                        `json:"materialFocus,omitempty"`
	StylingDensity             string                        `json:"stylingDensity,omitempty"`
	BrightnessPreference       string                        `json:"brightnessPreference,omitempty"`
}

type sceneImageResult struct {
	index       int
	imageURL    string
	rawResponse string
	err         error
}

func collectSceneImageResults(resultCh <-chan sceneImageResult, imageCount int) ([]model.SceneGenerationImage, string, error) {
	items := make([]model.SceneGenerationImage, imageCount)
	rawResponses := make([]string, 0, imageCount)
	var firstErr error
	successCount := 0

	for result := range resultCh {
		if strings.TrimSpace(result.rawResponse) != "" {
			rawResponses = append(rawResponses, result.rawResponse)
		}
		if result.err != nil {
			if firstErr == nil {
				firstErr = result.err
			}
			items[result.index] = model.SceneGenerationImage{
				CandidateStatus:       "failed",
				CandidateErrorMessage: result.err.Error(),
			}
			continue
		}
		if strings.TrimSpace(result.imageURL) == "" {
			err := fmt.Errorf("第 %d 张图片未返回有效结果", result.index+1)
			if firstErr == nil {
				firstErr = err
			}
			items[result.index] = model.SceneGenerationImage{
				CandidateStatus:       "failed",
				CandidateErrorMessage: err.Error(),
			}
			continue
		}
		successCount++
		items[result.index] = model.SceneGenerationImage{ImageURL: result.imageURL}
	}

	rawResponse := strings.Join(rawResponses, "\n")
	if successCount == 0 {
		if firstErr != nil {
			return nil, rawResponse, firstErr
		}
		return nil, rawResponse, fmt.Errorf("未生成有效图片")
	}
	return items, rawResponse, nil
}

type sceneImageEditRequest struct {
	Prompt    string `json:"prompt"`
	ImageData string `json:"imageData" binding:"required"`
	MimeType  string `json:"mimeType"`
}

type sceneStyleTagInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
}

type sceneStyleReferenceInput struct {
	SceneID    uint   `json:"scene_id"`
	TagID      uint   `json:"tag_id"`
	Title      string `json:"title"`
	ImageURL   string `json:"image_url"`
	PromptHint string `json:"prompt_hint"`
	Sort       int    `json:"sort"`
}

func resolveSceneModelAlias(modelName string) string {
	switch strings.ToLower(strings.TrimSpace(modelName)) {
	case "banana 2":
		return "gemini-3.1-flash-image-preview"
	case "banana pro":
		return "gemini-3-pro-image-preview"
	default:
		return strings.TrimSpace(modelName)
	}
}

type sceneAdminQueueTaskResponse struct {
	TaskID          string `json:"taskId"`
	UserName        string `json:"userName"`
	Status          string `json:"status"`
	ImageCount      int    `json:"imageCount"`
	ProductCategory string `json:"productCategory"`
	StyleName       string `json:"styleName"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

type sceneAdminQueueOverviewResponse struct {
	UserRunningImageLimit int                           `json:"userRunningImageLimit"`
	QueuedTaskCount       int64                         `json:"queuedTaskCount"`
	RunningTaskCount      int64                         `json:"runningTaskCount"`
	QueuedImageCount      int64                         `json:"queuedImageCount"`
	RunningImageCount     int64                         `json:"runningImageCount"`
	Items                 []sceneAdminQueueTaskResponse `json:"items"`
}

func ListAdminSceneQueueOverview(c *gin.Context) {
	var queuedTaskCount int64
	var runningTaskCount int64
	var queuedImageCount int64
	var runningImageCount int64

	if err := database.DB.Model(&model.SceneGenerationTask{}).Where("status = ?", "queued").Count(&queuedTaskCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取排队任务数量失败"})
		return
	}
	if err := database.DB.Model(&model.SceneGenerationTask{}).Where("status = ?", "running").Count(&runningTaskCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取运行任务数量失败"})
		return
	}
	if err := database.DB.Model(&model.SceneGenerationTask{}).Where("status = ?", "queued").Select("COALESCE(SUM(image_count), 0)").Scan(&queuedImageCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取排队图片数量失败"})
		return
	}
	if err := database.DB.Model(&model.SceneGenerationTask{}).Where("status = ?", "running").Select("COALESCE(SUM(image_count), 0)").Scan(&runningImageCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取运行图片数量失败"})
		return
	}

	var tasks []model.SceneGenerationTask
	if err := database.DB.Order("created_at desc").Limit(20).Where("status IN ?", []string{"queued", "running"}).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取排队详情失败"})
		return
	}

	items := make([]sceneAdminQueueTaskResponse, 0, len(tasks))
	userNameMap := map[uint]string{}
	if len(tasks) > 0 {
		userIDs := make([]uint, 0, len(tasks))
		seen := map[uint]struct{}{}
		for _, task := range tasks {
			if _, ok := seen[task.UserID]; ok {
				continue
			}
			seen[task.UserID] = struct{}{}
			userIDs = append(userIDs, task.UserID)
		}
		var users []model.User
		if err := database.DB.Select("id", "name", "username").Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务用户信息失败"})
			return
		}
		for _, user := range users {
			name := strings.TrimSpace(user.Name)
			if name == "" {
				name = strings.TrimSpace(user.Username)
			}
			userNameMap[user.ID] = name
		}
	}
	for _, task := range tasks {
		items = append(items, sceneAdminQueueTaskResponse{
			TaskID:          task.TaskID,
			UserName:        userNameMap[task.UserID],
			Status:          task.Status,
			ImageCount:      task.ImageCount,
			ProductCategory: task.ProductCategory,
			StyleName:       task.StyleName,
			CreatedAt:       task.CreatedAt.Format(time.RFC3339),
			UpdatedAt:       task.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, sceneAdminQueueOverviewResponse{
		UserRunningImageLimit: getSceneUserRunningImageLimit(),
		QueuedTaskCount:       queuedTaskCount,
		RunningTaskCount:      runningTaskCount,
		QueuedImageCount:      queuedImageCount,
		RunningImageCount:     runningImageCount,
		Items:                 items,
	})
}

func ListSceneStyleTags(c *gin.Context) {
	var tags []model.SceneStyleTag
	// tags 是跨场景通用的，始终加载所有标签
	if err := database.DB.Order("sort asc, id asc").Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取场景风格失败"})
		return
	}

	for i := range tags {
		var refs []model.SceneStyleReference
		q := database.DB.Where("tag_id = ?", tags[i].ID)
		// 当请求带了 scene_id 参数（web 端），按该 scene_id 过滤
		if sceneIDStr := c.Query("scene_id"); sceneIDStr != "" {
			if sceneID, err := strconv.Atoi(sceneIDStr); err == nil && sceneID > 0 {
				q = q.Where("scene_id = ?", sceneID)
			}
		}
		// 无 scene_id 参数时（admin 端），返回该 tag 下所有 reference，不再限制 scene_id
		q.Order("sort asc, id asc").Find(&refs)
		tags[i].References = refs
	}

	c.JSON(http.StatusOK, tags)
}

func AdminListSceneStyleTags(c *gin.Context) {
	ListSceneStyleTags(c)
}

func AdminCreateSceneStyleTag(c *gin.Context) {
	var input sceneStyleTagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "风格名称不能为空"})
		return
	}

	tag := model.SceneStyleTag{
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Sort:        input.Sort,
	}
	if err := database.DB.Create(&tag).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "风格名称已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建风格标签失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tag)
}

func AdminUpdateSceneStyleTag(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var tag model.SceneStyleTag
	if err := database.DB.First(&tag, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "风格标签不存在"})
		return
	}

	var input sceneStyleTagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "风格名称不能为空"})
		return
	}

	tag.Name = strings.TrimSpace(input.Name)
	tag.Description = strings.TrimSpace(input.Description)
	tag.Sort = input.Sort
	if err := database.DB.Save(&tag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新风格标签失败"})
		return
	}
	c.JSON(http.StatusOK, tag)
}

func AdminDeleteSceneStyleTag(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tag_id = ?", id).Delete(&model.SceneStyleReference{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.SceneStyleTag{}, id).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除风格标签失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func AdminCreateSceneStyleReference(c *gin.Context) {
	var input sceneStyleReferenceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.TagID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择所属风格"})
		return
	}
	// 不再校验 SceneID，前端可传但会被忽略，实际值取自 tag.SceneID（禁止跨场景创建）
	if strings.TrimSpace(input.ImageURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参考图不能为空"})
		return
	}

	var tag model.SceneStyleTag
	if err := database.DB.First(&tag, input.TagID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所属风格不存在"})
		return
	}

	item := model.SceneStyleReference{
		SceneID:    input.SceneID, // 使用前端传入的当前场景 ID（风格是跨场景通用的）
		TagID:      input.TagID,
		Title:      strings.TrimSpace(input.Title),
		ImageURL:   strings.TrimSpace(input.ImageURL),
		PromptHint: strings.TrimSpace(input.PromptHint),
		Sort:       input.Sort,
	}
	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建风格参考图失败"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func AdminUpdateSceneStyleReference(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var item model.SceneStyleReference
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "风格参考图不存在"})
		return
	}

	var input sceneStyleReferenceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.TagID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择所属风格"})
		return
	}
	if strings.TrimSpace(input.ImageURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参考图不能为空"})
		return
	}

	var tag model.SceneStyleTag
	if err := database.DB.First(&tag, input.TagID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "所属风格不存在"})
		return
	}

	item.SceneID = input.SceneID // 使用前端传入的当前场景 ID（风格是跨场景通用的）
	item.TagID = input.TagID
	item.Title = strings.TrimSpace(input.Title)
	item.ImageURL = strings.TrimSpace(input.ImageURL)
	item.PromptHint = strings.TrimSpace(input.PromptHint)
	item.Sort = input.Sort
	if err := database.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新风格参考图失败"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func AdminDeleteSceneStyleReference(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := database.DB.Delete(&model.SceneStyleReference{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除风格参考图失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func pickRandomSceneStyleReference(sceneID int, tagID uint) (*model.SceneStyleReference, error) {
	q := database.DB.Model(&model.SceneStyleReference{})
	if sceneID > 0 {
		q = q.Where("scene_id = ?", sceneID)
	}
	if tagID > 0 {
		q = q.Where("tag_id = ?", tagID)
	}

	var count int64
	if err := q.Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	offset := int(time.Now().UnixNano() % count)
	var ref model.SceneStyleReference
	if err := q.Order("id asc").Offset(offset).Limit(1).First(&ref).Error; err != nil {
		return nil, err
	}
	return &ref, nil
}

func CreateSceneGenerationTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req sceneTaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tag model.SceneStyleTag
	var ref *model.SceneStyleReference
	sceneType := resolveSceneType(req.ProductCategory)
	useStyleReference := sceneLocationSupportsStyleReference(req.ProductCategory)
	var sceneID int
	if sceneType == "kitchen" {
		sceneID = 4
	} else if sceneType == "bathroom" {
		sceneID = 3
	}
	if !useStyleReference {
		tag.Name = "真实自然生活方式"
		req.StyleTagID = 0
		req.StyleReferenceID = 0
	} else if strings.TrimSpace(req.StyleReferenceImageURL) != "" {
		var personalAsset model.Asset
		if err := database.DB.Where("file_url = ? AND owner_type = 'personal' AND user_id = ? AND type = ?", strings.TrimSpace(req.StyleReferenceImageURL), userID, "风格参考图").First(&personalAsset).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "风格参考图不属于当前账号"})
			return
		}
		tag.Name = "用户上传风格"
		ref = &model.SceneStyleReference{
			ImageURL: strings.TrimSpace(req.StyleReferenceImageURL),
			Title:    "个人资产风格参考图",
		}
	} else if req.StyleTagID == 0 {
		log.Printf("[randomStyle] ProductCategory=%s | sceneType=%s | sceneID=%d", req.ProductCategory, sceneType, sceneID)
		refObj, err := pickRandomSceneStyleReference(sceneID, 0)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("该场景(scene_id=%d)暂无风格图，请先上传风格参考图", sceneID), "_debug": fmt.Sprintf("sceneID=%d, sceneType=%s", sceneID, sceneType)})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "获取场景风格失败: " + err.Error(), "_debug": fmt.Sprintf("sceneID=%d", sceneID)})
			return
		}
		// 加载对应的 tag
		if err := database.DB.First(&tag, refObj.TagID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "风格标签不存在"})
			return
		}
		ref = refObj
	} else {
		if err := database.DB.First(&tag, req.StyleTagID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "风格标签不存在"})
			return
		}

		if req.StyleReferenceID != 0 {
			var refObj model.SceneStyleReference
			q := database.DB.Where("id = ? AND tag_id = ?", req.StyleReferenceID, req.StyleTagID)
			if sceneID > 0 {
				q = q.Where("scene_id = ?", sceneID)
			}
			if err := q.First(&refObj).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "风格参考图不存在"})
				return
			}
			ref = &refObj
		} else {
			// 未指定风格图时，随机抽取一个参考图
			refObj, err := pickRandomSceneStyleReference(sceneID, req.StyleTagID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					c.JSON(http.StatusBadRequest, gin.H{"error": "该风格暂无当前场景的参考图"})
					return
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": "获取风格参考图失败: " + err.Error()})
				return
			}
			ref = refObj
		}
	}

	productURL := strings.TrimSpace(req.ProductImageURL)
	if strings.TrimSpace(req.ProductImageData) != "" {
		savedURL, err := saveSceneSourceImage(req.ProductImageData, req.ProductImageMime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "保存产品图失败: " + err.Error()})
			return
		}
		productURL = savedURL
	} else if productURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先上传或从资产库选择 1 张产品图"})
		return
	}

	modelName := resolveSceneModelAlias(req.ModelName)
	if modelName == "" {
		modelName = getDefaultModelByType("drawing", "")
	}

	aspectRatio := strings.TrimSpace(req.AspectRatio)
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	imageSize := strings.TrimSpace(req.ImageSize)
	if imageSize == "" {
		imageSize = "2K"
	}
	if req.ImageCount <= 0 {
		req.ImageCount = 1
	}
	if req.ImageCount > 4 {
		req.ImageCount = 4
	}

	taskID := fmt.Sprintf("scene-%d-%d", time.Now().UnixMilli(), userID)
	// Handle color family image: custom upload (data URL) takes priority, else lookup by name
	colorImageURL := ""
	if req.ProductColorFamilyImageData != "" {
		savedURL, saveErr := saveSceneSourceImage(req.ProductColorFamilyImageData, req.ProductColorFamilyImageMime)
		if saveErr == nil {
			colorImageURL = savedURL
			// 将保存后的 URL 写回 req，确保异步任务中通过 task 表字段能正确获取颜色参考图
			req.ProductColorFamilyImageURL = savedURL
		}
	}
	if colorImageURL == "" && strings.TrimSpace(req.ProductColorFamily) != "" {
		var cf model.ProductColorFamily
		if err := database.DB.Where("name = ?", req.ProductColorFamily).First(&cf).Error; err == nil {
			colorImageURL = cf.ImageURL
		}
	}
	styleName := buildPromptValue(tag.Name, "真实自然生活方式")
	task := model.SceneGenerationTask{
		UserID:                     userID,
		TaskID:                     taskID,
		Status:                     "queued",
		StyleName:                  styleName,
		ProductCategory:            strings.TrimSpace(req.ProductCategory),
		CountryCode:                strings.TrimSpace(req.CountryCode),
		CountryName:                strings.TrimSpace(req.CountryName),
		ProductColorFamily:         strings.TrimSpace(req.ProductColorFamily),
		ProductColorFamilyImageURL: colorImageURL,
		AspectRatio:                aspectRatio,
		ModelName:                  modelName,
		ImageSize:                  imageSize,
		ImageCount:                 req.ImageCount,
		SubjectRatio:               strings.TrimSpace(req.SubjectRatio),
		CompositionTemplate:        strings.TrimSpace(req.CompositionTemplate),
		PrimaryToneHex:             strings.TrimSpace(req.PrimaryToneHex),
		FrameScope:                 strings.TrimSpace(req.FrameScope),
		LightingMood:               strings.TrimSpace(req.LightingMood),
		SpaceMood:                  strings.TrimSpace(req.SpaceMood),
		MaterialFocus:              strings.TrimSpace(req.MaterialFocus),
		StylingDensity:             strings.TrimSpace(req.StylingDensity),
		BrightnessPreference:       strings.TrimSpace(req.BrightnessPreference),
		Prompt:                     strings.TrimSpace(req.ExtraPrompt),
		ProductImageURL:            productURL,
	}
	if ref != nil {
		task.ReferenceImageURL = ref.ImageURL
		task.ReferenceImageTitle = strings.TrimSpace(ref.Title)
	}
	if err := database.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务失败"})
		return
	}

	go runSceneGenerationTask(taskID, req, tag, ref, userID, productURL)

	c.JSON(http.StatusOK, gin.H{"taskId": taskID})
}

func ListSceneGenerationTasks(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	savedOnly := c.Query("saved_only") == "1"

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 20 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	var total int64
	countQuery := database.DB.Model(&model.SceneGenerationTask{}).Where("user_id = ?", userID)
	if savedOnly {
		countQuery = countQuery.Where("is_saved_to_design = ?", true)
	}
	countQuery.Count(&total)

	var tasks []model.SceneGenerationTask
	query := database.DB.Where("user_id = ?", userID)
	if savedOnly {
		query = query.Where("is_saved_to_design = ?", true)
	}
	if err := query.Preload("Images").Order("created_at desc").Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取任务失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": mapSceneTaskList(tasks), "total": total, "page": page, "page_size": pageSize})
}

func GetSceneGenerationTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	taskID := c.Param("id")

	var task model.SceneGenerationTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).Preload("Images").First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, mapSceneTask(task))
}

func SaveSceneGenerationTaskToDesign(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	taskID := c.Param("id")

	var task model.SceneGenerationTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).Preload("Images").First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if len(task.Images) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务结果尚未生成完成"})
		return
	}
	for _, image := range task.Images {
		if strings.TrimSpace(image.CandidateStatus) == "generating" || strings.TrimSpace(image.CandidateStatus) == "ready" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "有待处理的编辑结果，请先确认或丢弃后再保存"})
			return
		}
	}
	if task.IsSavedToDesign {
		c.JSON(http.StatusOK, gin.H{"task": mapSceneTask(task), "message": "该任务已保存到我的设计"})
		return
	}

	now := time.Now()
	if err := database.DB.Model(&task).Updates(map[string]interface{}{
		"is_saved_to_design": true,
		"saved_at":           now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存任务失败"})
		return
	}
	task.IsSavedToDesign = true
	task.SavedAt = &now
	c.JSON(http.StatusOK, gin.H{"task": mapSceneTask(task), "message": "已保存到我的设计"})
}

func DeleteSceneGenerationTask(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	taskID := c.Param("id")

	var task model.SceneGenerationTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id_ref = ?", taskID).Delete(&model.SceneGenerationImage{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&task).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除任务失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "任务已删除"})
}

func mapSceneTaskList(tasks []model.SceneGenerationTask) []sceneTaskResponse {
	items := make([]sceneTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, mapSceneTask(task))
	}
	return items
}

func mapSceneTask(task model.SceneGenerationTask) sceneTaskResponse {
	images := make([]string, 0, len(task.Images))
	imageItems := make([]sceneImageCandidateResponse, 0, len(task.Images))
	sort.Slice(task.Images, func(i, j int) bool {
		if task.Images[i].Sort == task.Images[j].Sort {
			return task.Images[i].ID < task.Images[j].ID
		}
		return task.Images[i].Sort < task.Images[j].Sort
	})
	for _, image := range task.Images {
		images = append(images, image.ImageURL)
		imageItems = append(imageItems, sceneImageCandidateResponse{
			Sort:                  image.Sort,
			ImageURL:              image.ImageURL,
			CandidateImageURL:     image.CandidateImageURL,
			CandidateStatus:       image.CandidateStatus,
			CandidateErrorMessage: image.CandidateErrorMessage,
			CandidatePrompt:       image.CandidatePrompt,
		})
	}
	return sceneTaskResponse{
		TaskID:                     task.TaskID,
		Status:                     task.Status,
		ImageURLs:                  images,
		Images:                     imageItems,
		ErrorMessage:               task.ErrorMessage,
		CreatedAt:                  task.CreatedAt.Format(time.RFC3339),
		IsSavedToDesign:            task.IsSavedToDesign,
		SavedAt:                    formatTimePtr(task.SavedAt),
		ReferenceImageURL:          task.ReferenceImageURL,
		ReferenceImageTitle:        task.ReferenceImageTitle,
		StyleName:                  task.StyleName,
		ProductCategory:            task.ProductCategory,
		CountryCode:                task.CountryCode,
		CountryName:                task.CountryName,
		ProductColorFamily:         task.ProductColorFamily,
		ProductColorFamilyImageURL: task.ProductColorFamilyImageURL,
		AspectRatio:                task.AspectRatio,
		ProductImageURL:            task.ProductImageURL,
		ImageCount:                 task.ImageCount,
		SubjectRatio:               task.SubjectRatio,
		CompositionTemplate:        task.CompositionTemplate,
		PrimaryToneHex:             task.PrimaryToneHex,
		FrameScope:                 task.FrameScope,
		LightingMood:               task.LightingMood,
		SpaceMood:                  task.SpaceMood,
		MaterialFocus:              task.MaterialFocus,
		StylingDensity:             task.StylingDensity,
		BrightnessPreference:       task.BrightnessPreference,
	}
}

func fetchSceneTaskImage(taskID string, sortIndex int) (*model.SceneGenerationImage, error) {
	var image model.SceneGenerationImage
	if err := database.DB.Where("task_id_ref = ? AND sort = ?", taskID, sortIndex).First(&image).Error; err != nil {
		return nil, err
	}
	return &image, nil
}

func parseSceneImageSort(c *gin.Context) (int, error) {
	sortIndex, err := strconv.Atoi(c.Param("sort"))
	if err != nil || sortIndex < 0 {
		return 0, fmt.Errorf("invalid sort")
	}
	return sortIndex, nil
}

func StartSceneGenerationImageEdit(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	taskID := c.Param("id")
	sortIndex, err := parseSceneImageSort(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片索引无效"})
		return
	}

	var req sceneImageEditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var task model.SceneGenerationTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).Preload("Images").First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if task.IsSavedToDesign {
		c.JSON(http.StatusBadRequest, gin.H{"error": "已归档任务暂不支持继续编辑"})
		return
	}
	image, err := fetchSceneTaskImage(taskID, sortIndex)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "原图不存在"})
		return
	}
	if strings.TrimSpace(image.CandidateStatus) == "generating" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该图片已有编辑任务进行中"})
		return
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = strings.TrimSpace(task.Prompt)
	}
	if prompt == "" {
		prompt = "Preserve the product identity and apply the requested visual edit naturally."
	}

	mimeType := strings.TrimSpace(req.MimeType)
	if mimeType == "" {
		mimeType = "image/png"
	}

	if err := database.DB.Model(&model.SceneGenerationImage{}).
		Where("task_id_ref = ? AND sort = ?", taskID, sortIndex).
		Updates(map[string]interface{}{
			"candidate_status":        "generating",
			"candidate_error_message": "",
			"candidate_image_url":     "",
			"candidate_prompt":        prompt,
			"updated_at":              time.Now(),
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "启动编辑失败"})
		return
	}

	go runSceneGenerationImageEdit(task, sortIndex, prompt, req.ImageData, mimeType)

	if fresh, freshErr := loadSceneTaskByUser(taskID, userID); freshErr == nil {
		c.JSON(http.StatusOK, gin.H{"task": mapSceneTask(*fresh), "message": "已开始生成编辑预览"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已开始生成编辑预览"})
}

func AcceptSceneGenerationImageCandidate(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	taskID := c.Param("id")
	sortIndex, err := parseSceneImageSort(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片索引无效"})
		return
	}

	if _, err := loadSceneTaskByUser(taskID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	image, err := fetchSceneTaskImage(taskID, sortIndex)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "图片不存在"})
		return
	}
	if strings.TrimSpace(image.CandidateStatus) != "ready" || strings.TrimSpace(image.CandidateImageURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "当前没有可采用的编辑结果"})
		return
	}

	if err := database.DB.Model(&model.SceneGenerationImage{}).
		Where("task_id_ref = ? AND sort = ?", taskID, sortIndex).
		Updates(map[string]interface{}{
			"image_url":               image.CandidateImageURL,
			"candidate_image_url":     "",
			"candidate_status":        "",
			"candidate_error_message": "",
			"candidate_prompt":        "",
			"updated_at":              time.Now(),
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "采用编辑结果失败"})
		return
	}

	fresh, freshErr := loadSceneTaskByUser(taskID, userID)
	if freshErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "刷新任务失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"task": mapSceneTask(*fresh), "message": "已采用编辑结果"})
}

func RejectSceneGenerationImageCandidate(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)
	taskID := c.Param("id")
	sortIndex, err := parseSceneImageSort(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "图片索引无效"})
		return
	}

	if _, err := loadSceneTaskByUser(taskID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	if err := database.DB.Model(&model.SceneGenerationImage{}).
		Where("task_id_ref = ? AND sort = ?", taskID, sortIndex).
		Updates(map[string]interface{}{
			"candidate_image_url":     "",
			"candidate_status":        "",
			"candidate_error_message": "",
			"candidate_prompt":        "",
			"updated_at":              time.Now(),
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "丢弃编辑结果失败"})
		return
	}

	fresh, freshErr := loadSceneTaskByUser(taskID, userID)
	if freshErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "刷新任务失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"task": mapSceneTask(*fresh), "message": "已保留原图"})
}

func loadSceneTaskByUser(taskID string, userID uint) (*model.SceneGenerationTask, error) {
	var task model.SceneGenerationTask
	if err := database.DB.Where("task_id = ? AND user_id = ?", taskID, userID).Preload("Images").First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func runSceneGenerationImageEdit(task model.SceneGenerationTask, sortIndex int, prompt string, imageData string, mimeType string) {
	if err := waitForSceneUserImageSlots(task.UserID, 1); err != nil {
		_ = database.DB.Model(&model.SceneGenerationImage{}).
			Where("task_id_ref = ? AND sort = ?", task.TaskID, sortIndex).
			Updates(map[string]interface{}{
				"candidate_status":        "failed",
				"candidate_error_message": err.Error(),
				"updated_at":              time.Now(),
			})
		return
	}

	apiKey := getSettingValue("draw_key")
	if strings.TrimSpace(apiKey) == "" {
		_ = database.DB.Model(&model.SceneGenerationImage{}).
			Where("task_id_ref = ? AND sort = ?", task.TaskID, sortIndex).
			Updates(map[string]interface{}{
				"candidate_status":        "failed",
				"candidate_error_message": "Drawing API key not configured",
				"updated_at":              time.Now(),
			})
		return
	}

	modelName := strings.TrimSpace(task.ModelName)
	if modelName == "" {
		modelName = getDefaultModelByType("drawing", "")
	}
	if strings.TrimSpace(modelName) == "" {
		_ = database.DB.Model(&model.SceneGenerationImage{}).
			Where("task_id_ref = ? AND sort = ?", task.TaskID, sortIndex).
			Updates(map[string]interface{}{
				"candidate_status":        "failed",
				"candidate_error_message": "未配置默认绘图模型",
				"updated_at":              time.Now(),
			})
		return
	}

	ctx, cancel := newImageTaskContext()
	defer cancel()

	candidateURL, editErr := generateSceneEditCandidate(ctx, apiKey, resolveSceneModelAlias(modelName), prompt, imageData, mimeType, task.AspectRatio, task.ImageSize)
	if editErr != nil {
		_ = database.DB.Model(&model.SceneGenerationImage{}).
			Where("task_id_ref = ? AND sort = ?", task.TaskID, sortIndex).
			Updates(map[string]interface{}{
				"candidate_status":        "failed",
				"candidate_error_message": editErr.Error(),
				"updated_at":              time.Now(),
			})
		return
	}

	_ = database.DB.Model(&model.SceneGenerationImage{}).
		Where("task_id_ref = ? AND sort = ?", task.TaskID, sortIndex).
		Updates(map[string]interface{}{
			"candidate_image_url":     candidateURL,
			"candidate_status":        "ready",
			"candidate_error_message": "",
			"updated_at":              time.Now(),
		})
}

func resolveGPTImageBaseURL() string {
	baseURL := strings.TrimSpace(getSettingValue("access_url"))
	if baseURL != "" {
		return baseURL
	}
	return strings.TrimSpace(getSettingValue("gpt_image_base_url"))
}

func generateSceneEditCandidate(ctx context.Context, apiKey string, modelName string, prompt string, imageData string, mimeType string, aspectRatio string, imageSize string) (string, error) {
	baseURL := resolveGPTImageBaseURL()
	if baseURL == "" {
		baseURL = getSettingValue("access_url")
	}

	// 根据 URL 和模型名综合判断使用哪个适配器
	if adapter.IsApizURL(baseURL) {
		if adapter.IsGPTImageModel(modelName) {
			// apiz.ai + GPT 图片模型 → 使用 ApizGPTImageAdapter
			apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, baseURL)
			var result *adapter.ApizGPTImageResult
			var err error
			if imageData != "" {
				result, err = apizAdapter.EditImage(ctx, prompt, imageData, mimeType, aspectRatio, imageSize)
			} else {
				result, err = apizAdapter.GenerateImage(ctx, prompt, aspectRatio, imageSize)
			}
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(result.ImageURL) != "" {
				return strings.TrimSpace(result.ImageURL), nil
			}
			if strings.TrimSpace(result.ImageData) != "" {
				imageURL, saveErr := storage.SaveBase64Image(result.ImageData, result.MimeType)
				return imageURL, saveErr
			}
			return "", fmt.Errorf("Apiz GPT 图片未返回有效图片")
		} else {
			// apiz.ai + Gemini 模型 → 使用 ApizGeminiAdapter
			geminiAdapter := adapter.NewApizGeminiAdapter(apiKey, baseURL)
			var result *adapter.ApizGPTImageResult
			var err error
			if imageData != "" {
				refs := []adapter.RefImage{{Data: imageData, MimeType: mimeType}}
				result, err = geminiAdapter.GenerateImageWithRefs(ctx, prompt, refs, 1, imageSize, aspectRatio)
			} else {
				result, err = geminiAdapter.GenerateImage(ctx, prompt, 1, imageSize, aspectRatio)
			}
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(result.ImageURL) != "" {
				return strings.TrimSpace(result.ImageURL), nil
			}
			if strings.TrimSpace(result.ImageData) != "" {
				imageURL, saveErr := storage.SaveBase64Image(result.ImageData, result.MimeType)
				return imageURL, saveErr
			}
			return "", fmt.Errorf("Apiz Gemini 图片未返回有效图片")
		}
	} else if adapter.IsGPTImageModel(modelName) {
		// 非 apiz URL + GPT 图片模型 → 使用旧版 GPTImageAdapter
		gptAdapter := adapter.NewGPTImageAdapter(apiKey, baseURL)
		result, err := gptAdapter.EditImage(ctx, prompt, []adapter.RefImage{{Data: imageData, MimeType: mimeType}}, modelName, aspectRatio, imageSize, 1)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(result.ImageData) != "" {
			return storage.SaveBase64Image(result.ImageData, result.MimeType)
		}
		return "", fmt.Errorf("GPT 图片编辑未返回有效图片")
	} else if adapter.IsWanModel(modelName) {
		// Wan 模型 → 使用 WanAdapter
		wanAdapter := adapter.NewWanAdapter(apiKey, baseURL)
		var images []string
		if imageData != "" {
			images = []string{imageData}
		}
		wanResult, err := wanAdapter.GenerateImage(ctx, prompt, modelName, aspectRatio, imageSize, 1, false, false, images)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(wanResult.ImageData) != "" {
			return storage.SaveBase64Image(wanResult.ImageData, wanResult.MimeType)
		}
		if strings.TrimSpace(wanResult.ImageURL) != "" {
			wanImageURL, _, saveErr := storage.SaveImageFromURL(wanResult.ImageURL)
			return wanImageURL, saveErr
		}
		return "", fmt.Errorf("Wan 图片生成未返回有效图片")
	} else {
		// 其他情况 → 使用旧版 GeminiAdapter (ApizGeminiAdapter)
		geminiAdapter := adapter.NewGeminiAdapter(apiKey, baseURL)
		refImages := []adapter.RefImage{{Data: imageData, MimeType: mimeType}}
		result, _, err := generateGeminiImageWithModelFallback(ctx, "scene-edit", geminiAdapter, prompt, refImages, aspectRatio, imageSize, modelName)
		if err != nil {
			return "", err
		}
		imageURL, _, saveErr := saveGeminiImageResult(result)
		return imageURL, saveErr
	}
}

func formatTimePtr(v *time.Time) string {
	if v == nil || v.IsZero() {
		return ""
	}
	return v.Format(time.RFC3339)
}

func saveSceneSourceImage(data string, mimeType string) (string, error) {
	clean := strings.TrimSpace(data)
	if idx := strings.Index(clean, ","); idx >= 0 {
		clean = clean[idx+1:]
	}
	if _, err := base64.StdEncoding.DecodeString(clean); err != nil {
		if _, rawErr := base64.RawStdEncoding.DecodeString(clean); rawErr != nil {
			return "", err
		}
	}
	if mimeType == "" {
		mimeType = "image/png"
	}
	return storage.SavePromptImage(clean, mimeType)
}

func fetchSceneReferenceImageData(imageURL string) (string, string, error) {
	imageURL = strings.TrimSpace(imageURL)
	if imageURL == "" {
		return "", "", nil
	}

	if strings.HasPrefix(imageURL, "/") {
		baseURL := strings.TrimRight(getSettingValue("server_url"), "/")
		if baseURL == "" {
			baseURL = "http://127.0.0.1:8080"
		}
		imageURL = baseURL + imageURL
	}

	resp, err := http.Get(imageURL) //nolint:gosec
	if err != nil {
		return "", "", fmt.Errorf("下载参考图失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("下载参考图失败: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取参考图失败: %w", err)
	}

	mimeType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if idx := strings.Index(mimeType, ";"); idx >= 0 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	if mimeType == "" {
		ext := strings.ToLower(filepath.Ext(strings.Split(strings.TrimSpace(imageURL), "?")[0]))
		switch ext {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".webp":
			mimeType = "image/webp"
		case ".gif":
			mimeType = "image/gif"
		default:
			mimeType = "image/png"
		}
	}

	return base64.StdEncoding.EncodeToString(data), mimeType, nil
}

func getSceneUserRunningImageLimit() int {
	value := strings.TrimSpace(getSettingValue(sceneUserRunningImageLimitSettingKey))
	if value == "" {
		return sceneUserRunningImageLimitDefault
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return sceneUserRunningImageLimitDefault
	}
	return limit
}

func countSceneUserRunningImages(userID uint) (int64, error) {
	var taskImages int64
	if err := database.DB.Model(&model.SceneGenerationTask{}).
		Where("user_id = ? AND status = ?", userID, "running").
		Select("COALESCE(SUM(image_count), 0)").
		Scan(&taskImages).Error; err != nil {
		return 0, err
	}

	var editImages int64
	if err := database.DB.Model(&model.SceneGenerationImage{}).
		Joins("JOIN scene_generation_tasks ON scene_generation_tasks.task_id = scene_generation_images.task_id_ref").
		Where("scene_generation_tasks.user_id = ? AND scene_generation_images.candidate_status = ?", userID, "generating").
		Count(&editImages).Error; err != nil {
		return 0, err
	}

	return taskImages + editImages, nil
}

func waitForSceneUserImageSlots(userID uint, required int) error {
	if required <= 0 {
		return nil
	}
	limit := getSceneUserRunningImageLimit()
	if limit <= 0 || required > limit {
		return nil
	}

	deadline := time.Now().Add(sceneTaskQueueMaxWait)
	for {
		running, err := countSceneUserRunningImages(userID)
		if err != nil {
			return err
		}
		if int(running)+required <= limit {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("等待生成资源超时")
		}
		time.Sleep(sceneTaskQueuePollInterval)
	}
}

func runSceneGenerationTask(taskID string, req sceneTaskCreateRequest, tag model.SceneStyleTag, ref *model.SceneStyleReference, userID uint, productImageURL string) {
	if err := waitForSceneUserImageSlots(userID, req.ImageCount); err != nil {
		database.DB.Model(&model.SceneGenerationTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{"status": "failed", "error_message": err.Error()})
		return
	}

	database.DB.Model(&model.SceneGenerationTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{"status": "running", "error_message": ""})

	apiKey := getSettingValue("draw_key")
	if apiKey == "" {
		database.DB.Model(&model.SceneGenerationTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{"status": "failed", "error_message": "Drawing API key not configured"})
		return
	}

	prompt := buildScenePrompt(req, tag, ref)
	modelName := strings.TrimSpace(req.ModelName)
	if modelName == "" {
		modelName = getDefaultModelByType("drawing", "")
	}
	if modelName == "" {
		database.DB.Model(&model.SceneGenerationTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{"status": "failed", "error_message": "未配置默认绘图模型"})
		return
	}
	aspectRatio := strings.TrimSpace(req.AspectRatio)
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	imageSize := strings.TrimSpace(req.ImageSize)
	if imageSize == "" {
		imageSize = "2K"
	}

	usageLogBase := model.UsageLog{
		UserID:     userID,
		UsageType:  "drawing",
		Source:     "scene_generation",
		Model:      modelName,
		Prompt:     prompt,
		Resolution: imageSize,
		Status:     "failed",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	ctx, cancel := newImageTaskContext()
	defer cancel()

	mimeType := req.ProductImageMime
	if mimeType == "" {
		mimeType = "image/png"
	}
	productImageData := strings.TrimSpace(req.ProductImageData)
	productMimeType := mimeType
	if productImageData == "" && strings.TrimSpace(productImageURL) != "" {
		var fetchErr error
		productImageData, productMimeType, fetchErr = fetchSceneReferenceImageData(productImageURL)
		if fetchErr != nil {
			createSceneGenerationUsageLogs(usageLogBase, taskID, req.ImageCount, nil, fetchErr)
			database.DB.Model(&model.SceneGenerationTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{"status": "failed", "error_message": fetchErr.Error()})
			return
		}
	}

	refImageData := ""
	refMimeType := ""
	refImageURL := ""
	useApizReferenceURLs := adapter.IsApizURL(resolveGPTImageBaseURL())
	if ref != nil && strings.TrimSpace(ref.ImageURL) != "" {
		refImageURL = strings.TrimSpace(ref.ImageURL)
		if !useApizReferenceURLs {
			var fetchErr error
			refImageData, refMimeType, fetchErr = fetchSceneReferenceImageData(refImageURL)
			if fetchErr != nil {
				// usageLog.ResponseBody = fmt.Sprintf(`{"error":%q}`, fetchErr.Error()) // not stored
				createSceneGenerationUsageLogs(usageLogBase, taskID, req.ImageCount, nil, fetchErr)
				database.DB.Model(&model.SceneGenerationTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{"status": "failed", "error_message": fetchErr.Error()})
				return
			}
			refImageURL = ""
		}
	}

	colorImageData := ""
	colorImageMimeType := ""
	colorImageURL := strings.TrimSpace(req.ProductColorFamilyImageURL)
	if colorImageURL != "" && !useApizReferenceURLs {
		var fetchErr error
		colorImageData, colorImageMimeType, fetchErr = fetchSceneReferenceImageData(colorImageURL)
		if fetchErr != nil {
			colorImageURL = ""
		} else {
			colorImageURL = ""
		}
	}

	created, _, generateErr := generateSceneImages(ctx, apiKey, modelName, prompt, req.ImageCount, productImageData, productMimeType, productImageURL, refImageData, refMimeType, refImageURL, colorImageData, colorImageMimeType, colorImageURL, aspectRatio, imageSize, resolveSceneType(req.ProductCategory), req.ProductCategory)
	if generateErr != nil {
		// usageLog.ResponseBody = fmt.Sprintf(`{"error":%q}`, generateErr.Error()) // not stored
		createSceneGenerationUsageLogs(usageLogBase, taskID, req.ImageCount, nil, generateErr)
		database.DB.Where("task_id_ref = ?", taskID).Delete(&model.SceneGenerationImage{})
		database.DB.Model(&model.SceneGenerationTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{"status": "failed", "error_message": generateErr.Error()})
		return
	}

	for i := range created {
		created[i].TaskIDRef = taskID
		created[i].Sort = i
		database.DB.Create(&created[i])
	}
	failedCount := 0
	for _, image := range created {
		if strings.TrimSpace(image.CandidateStatus) == "failed" {
			failedCount++
		}
	}
	createSceneGenerationUsageLogs(usageLogBase, taskID, req.ImageCount, created, nil)
	updates := map[string]interface{}{"status": "done", "error_message": ""}
	if failedCount > 0 {
		updates["error_message"] = fmt.Sprintf("部分图片生成失败：%d/%d", failedCount, len(created))
	}
	database.DB.Model(&model.SceneGenerationTask{}).Where("task_id = ?", taskID).Updates(updates)
}

func createSceneGenerationUsageLogs(base model.UsageLog, taskID string, imageCount int, images []model.SceneGenerationImage, err error) {
	if imageCount <= 0 {
		imageCount = 1
	}
	count := imageCount
	if len(images) > count {
		count = len(images)
	}
	duration := int(time.Since(base.CreatedAt).Milliseconds())
	now := time.Now()

	for i := 0; i < count; i++ {
		logItem := base
		logItem.ID = 0
		logItem.ImageURL = ""
		logItem.Duration = duration
		logItem.RequestBody = fmt.Sprintf(`{"task_id":%q,"sort":%d,"image_count":%d}`, taskID, i, imageCount)
		logItem.UpdatedAt = now

		var image model.SceneGenerationImage
		if i < len(images) {
			image = images[i]
		}

		imageErr := err
		if strings.TrimSpace(image.CandidateStatus) == "failed" {
			imageErr = errors.New(firstNonEmpty(image.CandidateErrorMessage, "场景图生成失败"))
		}

		if imageErr != nil {
			logItem.Status = "failed"
			logItem.Result = fmt.Sprintf("第 %d 张场景图生成失败：%s", i+1, imageErr.Error())
		} else {
			logItem.Status = "success"
			logItem.ImageURL = image.ImageURL
			logItem.Result = fmt.Sprintf("第 %d 张场景图已生成", i+1)
		}

		database.DB.Create(&logItem)
		go incUsageCount(logItem.UserID, logItem.Model)
	}
}

func generateSceneImage(ctx context.Context, apiKey string, modelName string, prompt string, productImageData string, productMimeType string, styleRefImageData string, styleRefMimeType string, colorImageData string, colorImageMimeType string, aspectRatio string, imageSize string) (string, string, error) {
	images, rawResponse, err := generateSceneImages(ctx, apiKey, modelName, prompt, 1, productImageData, productMimeType, "", styleRefImageData, styleRefMimeType, "", colorImageData, colorImageMimeType, "", aspectRatio, imageSize, "generic", "")
	if err != nil {
		return "", "", err
	}
	if len(images) == 0 {
		return "", rawResponse, fmt.Errorf("未返回有效图片")
	}
	return images[0].ImageURL, rawResponse, nil
}

func generateSceneImages(ctx context.Context, apiKey string, modelName string, prompt string, imageCount int, productImageData string, productMimeType string, productImageURL string, styleRefImageData string, styleRefMimeType string, styleRefImageURL string, colorImageData string, colorImageMimeType string, colorImageURL string, aspectRatio string, imageSize string, sceneType string, productCategory string) ([]model.SceneGenerationImage, string, error) {
	// 图片顺序：1.产品白底图 2.风格参考图 3.颜色参考图（改色用）
	refImages := make([]adapter.RefImage, 0, 3)
	if strings.TrimSpace(productImageURL) != "" && adapter.IsApizURL(resolveGPTImageBaseURL()) {
		refImages = append(refImages, adapter.RefImage{URL: strings.TrimSpace(productImageURL)})
	} else if strings.TrimSpace(productImageData) != "" {
		mimeType := strings.TrimSpace(productMimeType)
		if mimeType == "" {
			mimeType = "image/png"
		}
		refImages = append(refImages, adapter.RefImage{Data: productImageData, MimeType: mimeType})
	}
	if strings.TrimSpace(styleRefImageData) != "" {
		mimeType := strings.TrimSpace(styleRefMimeType)
		if mimeType == "" {
			mimeType = "image/png"
		}
		refImages = append(refImages, adapter.RefImage{Data: styleRefImageData, MimeType: mimeType})
	} else if strings.TrimSpace(styleRefImageURL) != "" {
		refImages = append(refImages, adapter.RefImage{URL: strings.TrimSpace(styleRefImageURL)})
	}
	if strings.TrimSpace(colorImageData) != "" {
		mimeType := strings.TrimSpace(colorImageMimeType)
		if mimeType == "" {
			mimeType = "image/png"
		}
		refImages = append(refImages, adapter.RefImage{Data: colorImageData, MimeType: mimeType})
	} else if strings.TrimSpace(colorImageURL) != "" {
		refImages = append(refImages, adapter.RefImage{URL: strings.TrimSpace(colorImageURL)})
	}
	return generateSceneImagesWithRefs(ctx, apiKey, modelName, prompt, imageCount, refImages, aspectRatio, imageSize, sceneType, productCategory)
}

func generateSceneImagesWithRefs(ctx context.Context, apiKey string, modelName string, prompt string, imageCount int, refImages []adapter.RefImage, aspectRatio string, imageSize string, sceneType string, productCategory string) ([]model.SceneGenerationImage, string, error) {
	if imageCount <= 0 {
		imageCount = 1
	}

	baseURL := resolveGPTImageBaseURL()

	filteredRefs := make([]adapter.RefImage, 0, len(refImages))
	for _, ref := range refImages {
		refURL := strings.TrimSpace(ref.URL)
		if strings.TrimSpace(ref.Data) == "" && refURL == "" {
			continue
		}
		mimeType := strings.TrimSpace(ref.MimeType)
		if mimeType == "" {
			mimeType = "image/png"
		}
		filteredRefs = append(filteredRefs, adapter.RefImage{Data: ref.Data, MimeType: mimeType, URL: refURL})
	}

	if adapter.IsApizURL(baseURL) {
		if adapter.IsGPTImageModel(modelName) {
			apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, baseURL)
			var batch *adapter.ApizGPTImageBatchResult
			var err error
			if len(filteredRefs) > 0 {
				batch, err = apizAdapter.EditImagesBatch(ctx, prompt, filteredRefs, aspectRatio, imageSize, imageCount)
			} else {
				batch, err = apizAdapter.GenerateImages(ctx, prompt, aspectRatio, imageSize, imageCount)
			}
			if err != nil {
				return nil, "", err
			}
			if len(batch.Items) != imageCount {
				return nil, "", fmt.Errorf("image count mismatch: requested %d, got %d", imageCount, len(batch.Items))
			}
			items := make([]model.SceneGenerationImage, 0, imageCount)
			for index, item := range batch.Items {
				items = append(items, model.SceneGenerationImage{Sort: index, ImageURL: strings.TrimSpace(item.ImageURL)})
			}
			return items, "", nil
		}

		resultCh := make(chan sceneImageResult, imageCount)
		var wg sync.WaitGroup
		for i := 0; i < imageCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				variantPrompt := promptVariant(prompt, index, sceneType, productCategory)
				var result *adapter.ApizGPTImageResult
				var err error

				if adapter.IsGPTImageModel(modelName) {
					apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, baseURL)
					if len(filteredRefs) > 0 {
						result, err = apizAdapter.EditImages(ctx, variantPrompt, filteredRefs, aspectRatio, imageSize)
					} else {
						result, err = apizAdapter.GenerateImage(ctx, variantPrompt, aspectRatio, imageSize)
					}
				} else {
					geminiAdapter := adapter.NewApizGeminiAdapter(apiKey, baseURL)
					if len(filteredRefs) > 0 {
						result, err = geminiAdapter.GenerateImageWithRefs(ctx, variantPrompt, filteredRefs, 1, imageSize, aspectRatio)
					} else {
						result, err = geminiAdapter.GenerateImage(ctx, variantPrompt, 1, imageSize, aspectRatio)
					}
				}

				var imageURL string
				var rawResponse string
				if err == nil && result != nil {
					if strings.TrimSpace(result.ImageURL) != "" {
						imageURL = strings.TrimSpace(result.ImageURL)
						rawResponse = result.ImageURL
					} else if strings.TrimSpace(result.ImageData) != "" {
						imageURL, err = storage.SaveBase64Image(result.ImageData, result.MimeType)
					}
				}
				resultCh <- sceneImageResult{index: index, imageURL: imageURL, rawResponse: rawResponse, err: err}
			}(i)
		}
		wg.Wait()
		close(resultCh)

		items, rawResponse, err := collectSceneImageResults(resultCh, imageCount)
		return items, rawResponse, err
	}

	if adapter.IsGPTImageModel(modelName) {
		resultCh := make(chan sceneImageResult, imageCount)
		var wg sync.WaitGroup
		for i := 0; i < imageCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				variantPrompt := promptVariant(prompt, index, sceneType, productCategory)
				gptAdapter := adapter.NewGPTImageAdapter(apiKey, baseURL)
				batch, err := gptAdapter.EditImages(ctx, variantPrompt, filteredRefs, modelName, aspectRatio, imageSize, 1)
				var imageURL string
				var rawResponse string
				if err == nil && len(batch.Items) > 0 {
					item := batch.Items[0]
					if strings.TrimSpace(item.ImageData) != "" {
						imageURL, err = storage.SaveBase64Image(item.ImageData, item.MimeType)
					} else if strings.TrimSpace(item.ImageURL) != "" {
						imageURL, _, err = storage.SaveImageFromURL(item.ImageURL)
					}
					rawResponse = batch.RawResponse
				}
				resultCh <- sceneImageResult{index: index, imageURL: imageURL, rawResponse: rawResponse, err: err}
			}(i)
		}
		wg.Wait()
		close(resultCh)

		return collectSceneImageResults(resultCh, imageCount)
	}

	// ── Wan 分支 ──────────────────────────────────────────────────────────
	if adapter.IsWanModel(modelName) {
		// 提取所有参考图的 data 字段为 []string
		var refImages []string
		for _, ref := range filteredRefs {
			if ref.Data != "" {
				refImages = append(refImages, ref.Data)
			}
		}
		resultCh := make(chan sceneImageResult, imageCount)
		var wg sync.WaitGroup
		for i := 0; i < imageCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				variantPrompt := promptVariant(prompt, index, sceneType, productCategory)
				wanAdapter := adapter.NewWanAdapter(apiKey, resolveGPTImageBaseURL())
				wanResult, err := wanAdapter.GenerateImage(ctx, variantPrompt, modelName, aspectRatio, imageSize, 1, false, false, refImages)
				var imageURL string
				var rawResponse string
				if err == nil && strings.TrimSpace(wanResult.ImageData) != "" {
					imageURL, _ = storage.SaveBase64Image(wanResult.ImageData, wanResult.MimeType)
					rawResponse = wanResult.RawResponse
				} else if err == nil && strings.TrimSpace(wanResult.ImageURL) != "" {
					imageURL, _, _ = storage.SaveImageFromURL(wanResult.ImageURL)
					rawResponse = wanResult.RawResponse
				}
				resultCh <- sceneImageResult{index: index, imageURL: imageURL, rawResponse: rawResponse, err: err}
			}(i)
		}
		wg.Wait()
		close(resultCh)

		items, rawResponse, err := collectSceneImageResults(resultCh, imageCount)
		return items, rawResponse, err
	}

	resultCh := make(chan sceneImageResult, imageCount)
	var wg sync.WaitGroup
	for i := 0; i < imageCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			variantPrompt := promptVariant(prompt, index, sceneType, productCategory)
			imageURL, rawResponse, err := generateSingleSceneImageWithRefs(ctx, apiKey, modelName, variantPrompt, aspectRatio, imageSize, filteredRefs)
			resultCh <- sceneImageResult{index: index, imageURL: imageURL, rawResponse: rawResponse, err: err}
		}(i)
	}
	wg.Wait()
	close(resultCh)

	items, rawResponse, err := collectSceneImageResults(resultCh, imageCount)
	return items, rawResponse, err
}

func generateSingleSceneImage(ctx context.Context, apiKey string, modelName string, prompt string, productImageData string, productMimeType string, styleRefImageData string, styleRefMimeType string, aspectRatio string, imageSize string, refImages []adapter.RefImage) (string, string, error) {
	return generateSingleSceneImageWithRefs(ctx, apiKey, modelName, prompt, aspectRatio, imageSize, refImages)
}

func generateSingleSceneImageWithRefs(ctx context.Context, apiKey string, modelName string, prompt string, aspectRatio string, imageSize string, refImages []adapter.RefImage) (string, string, error) {
	if adapter.IsViduModel(modelName) {
		viduAdapter := adapter.NewViduAdapter(apiKey, getSettingValue("access_url"))
		result, err := viduAdapter.GenerateImageWithRefs(prompt, modelName, aspectRatio, imageSize, refImages)
		if err != nil {
			return "", "", err
		}
		imageURL, _, saveErr := storage.SaveImageFromURL(result.ImageURL)
		if saveErr != nil {
			return "", result.RawResponse, saveErr
		}
		return imageURL, result.RawResponse, nil
	}

	if adapter.IsTencentAIGCModel(modelName) {
		modelBase, modelVersion := adapter.ParseTencentModelName(modelName)
		tencentAdapter := adapter.NewTencentAIGCAdapter(apiKey, getSettingValue("access_url"))

		localPaths := make([]string, 0, len(refImages))
		refURLs := make([]string, 0, len(refImages))
		for _, refImage := range refImages {
			if strings.TrimSpace(refImage.Data) == "" {
				continue
			}
			localPath, saveErr := storage.SaveTempRefImage(refImage.Data, refImage.MimeType)
			if saveErr != nil {
				return "", "", saveErr
			}
			localPaths = append(localPaths, localPath)
		}
		defer storage.DeleteTempRefFiles(localPaths)

		baseURL := strings.TrimRight(getSettingValue("server_url"), "/")
		if baseURL == "" {
			baseURL = strings.TrimRight(getSettingValue("access_url"), "/")
		}
		if baseURL == "" {
			return "", "", fmt.Errorf("未配置 server_url 或 access_url，无法为腾讯/云雾 URL 引用模式生成参考图地址")
		}
		for _, localPath := range localPaths {
			refURLs = append(refURLs, baseURL+localPath)
		}

		result, err := tencentAdapter.GenerateImageWithURLRefs(prompt, modelBase, modelVersion, aspectRatio, imageSize, refURLs)
		if err != nil {
			return "", "", err
		}
		imageURL, _, downloadErr := storage.SaveImageFromURL(result.ImageURL)
		if downloadErr != nil {
			return "", result.RawResponse, downloadErr
		}
		return imageURL, result.RawResponse, nil
	}

	// ── 根据 URL 和模型名综合判断使用哪个适配器 ─────────────────────────────────
	apizBaseURL := resolveGPTImageBaseURL()
	if apizBaseURL == "" {
		apizBaseURL = getSettingValue("access_url")
	}

	if adapter.IsApizURL(apizBaseURL) {
		if adapter.IsGPTImageModel(modelName) {
			// apiz.ai + GPT 图片模型 → 使用 ApizGPTImageAdapter
			apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, apizBaseURL)
			var result *adapter.ApizGPTImageResult
			var err error
			if len(refImages) > 0 && refImages[0].Data != "" {
				result, err = apizAdapter.EditImages(ctx, prompt, refImages, aspectRatio, imageSize)
			} else {
				result, err = apizAdapter.GenerateImage(ctx, prompt, aspectRatio, imageSize)
			}
			if err != nil {
				return "", "", err
			}
			if strings.TrimSpace(result.ImageURL) != "" {
				return strings.TrimSpace(result.ImageURL), "", nil
			}
			if strings.TrimSpace(result.ImageData) != "" {
				imageURL, saveErr := storage.SaveBase64Image(result.ImageData, result.MimeType)
				if saveErr != nil {
					return "", "", saveErr
				}
				return imageURL, "", nil
			}
			return "", "", fmt.Errorf("Apiz GPT 图片未返回有效图片")
		} else {
			// apiz.ai + Gemini 模型 → 使用 ApizGeminiAdapter
			geminiAdapter := adapter.NewApizGeminiAdapter(apiKey, apizBaseURL)
			var result *adapter.ApizGPTImageResult
			var err error
			if len(refImages) > 0 {
				result, err = geminiAdapter.GenerateImageWithRefs(ctx, prompt, refImages, 1, imageSize, aspectRatio)
			} else {
				result, err = geminiAdapter.GenerateImage(ctx, prompt, 1, imageSize, aspectRatio)
			}
			if err != nil {
				return "", "", err
			}
			if strings.TrimSpace(result.ImageURL) != "" {
				return strings.TrimSpace(result.ImageURL), "", nil
			}
			if strings.TrimSpace(result.ImageData) != "" {
				imageURL, saveErr := storage.SaveBase64Image(result.ImageData, result.MimeType)
				if saveErr != nil {
					return "", "", saveErr
				}
				return imageURL, "", nil
			}
			return "", "", fmt.Errorf("Apiz Gemini 图片未返回有效图片")
		}
	} else if adapter.IsGPTImageModel(modelName) {
		// 非 apiz URL + GPT 图片模型 → 使用旧版 GPTImageAdapter
		gptAdapter := adapter.NewGPTImageAdapter(apiKey, apizBaseURL)
		var result *adapter.GPTImageBatchResult
		var err error
		if len(refImages) > 0 {
			result, err = gptAdapter.EditImages(ctx, prompt, refImages, modelName, aspectRatio, imageSize, 1)
		} else {
			result, err = gptAdapter.GenerateImages(ctx, prompt, modelName, aspectRatio, imageSize, 1)
		}
		if err != nil {
			return "", "", err
		}
		if len(result.Items) > 0 {
			item := result.Items[0]
			if strings.TrimSpace(item.ImageData) != "" {
				imageURL, saveErr := storage.SaveBase64Image(item.ImageData, item.MimeType)
				if saveErr != nil {
					return "", "", saveErr
				}
				return imageURL, result.RawResponse, nil
			}
			if strings.TrimSpace(item.ImageURL) != "" {
				imageURL, _, downloadErr := storage.SaveImageFromURL(item.ImageURL)
				if downloadErr != nil {
					return "", result.RawResponse, downloadErr
				}
				return imageURL, result.RawResponse, nil
			}
		}
		return "", "", fmt.Errorf("GPT 图片未返回有效图片")
	} else {
		// 其他情况 → 使用旧版 GeminiAdapter
		geminiAdapter := adapter.NewGeminiAdapter(apiKey, apizBaseURL)
		result, _, err := generateGeminiImageWithModelFallback(ctx, "scene-generation", geminiAdapter, prompt, refImages, aspectRatio, imageSize, modelName)
		if err != nil {
			return "", "", err
		}
		imageURL, _, saveErr := saveGeminiImageResult(result)
		if saveErr != nil {
			return "", "", saveErr
		}
		return imageURL, "", nil
	}
}

func buildScenePrompt(req sceneTaskCreateRequest, tag model.SceneStyleTag, ref *model.SceneStyleReference) string {
	category := strings.TrimSpace(req.ProductCategory)
	location := normalizeSceneLocation(category)
	useStyleReference := sceneLocationSupportsStyleReference(category)
	style := buildPromptValue(tag.Name, "真实自然生活方式")

	productDescriptor := "target faucet product"
	sceneDesc := "真实住宅安装场景"
	sceneEnvironmentRules := "原创真实住宅使用场景。根据产品参考图结构选择合理安装区域，保持空间、材质、灯光和产品比例自然可信。"
	installationInstruction := "根据参考图产品结构安装在真实可承重表面上，产品与环境自然接触，阴影和透视一致。"
	lightingPrompt := "光线：统一的环境照明，光源位于左上方。阴影与周围环境投射方向一致，匹配参考图的色温。产品使用与环境一致的自然光线。"
	refProductRule := "参考图产品形状、比例、外观和功能细节保持一致。"
	uniquenessRule := "场景中只呈现一个目标产品，其他装饰与五金作为背景弱化处理。"
	integrityRule := "住宅安装区域合理，产品安装在真实可承重表面上，比例与周围物体协调。"
	roomObjectRule := "场景中的所有物品必须属于同一个真实使用空间，不混入其他房间的功能物品。"
	geometryLockRule := "产品、底座、台面、安装孔位和周围固定结构是同一个真实安装系统；视角变化只能来自相机位置、镜头焦距和裁切变化，不得旋转、移动、翻转、重排或重新设计产品和场景结构。"

	switch location {
	case "kitchen_sink":
		productDescriptor = "kitchen faucet"
		sceneDesc = "厨房水槽区场景"
		sceneEnvironmentRules = "原创厨房水槽区设计。必需元素：主水槽、连续台面、底柜、防溅墙或窗边墙面、合理灯光。厨房布局围绕目标水龙头所在的水槽区展开。台面高度约91cm。只能出现厨房场景物品。"
		installationInstruction = "真实安装：厨房水龙头从水槽后方或侧后方的实心台面孔位升起，底座与台面平齐，出水口延伸至水槽盆内部。产品高度与厨房台面比例协调，可见稳定自然的安装接触面。"
		lightingPrompt = "光线：统一的厨房环境照明，光源位于左上方。阴影与水槽、橱柜、防溅墙、台面投射方向一致。匹配参考图的色温。产品使用与水槽区一致的自然光线，金属高光自然统一。"
		refProductRule = "厨房水龙头按真实水槽龙头比例呈现，水槽内口宽度必须明显大于水龙头整体宽度和出水口投影宽度；底座固定在水槽后方台面孔位，出水口对准水槽内部；保留龙体、弯管、把手、连接杆和底座的截面形状、厚薄关系、边线和相对间距。"
		uniquenessRule = "厨房里只能有这一个可用水龙头和这一个主水槽；禁止背景水龙头、第二个水槽、远处备用洗涤区或镜面反射中的新增水龙头。"
		integrityRule = "厨房水槽区完整性：场景中只有一个主水槽和一个连续水槽台面。台面、底柜、防溅墙或窗边墙面、上柜和灯光属于同一个真实厨房空间。"
		roomObjectRule = "厨房场景只能出现厨房物品和厨房空间元素，例如水槽、台面、橱柜、防溅墙、厨房窗、厨房灯具、厨具或少量厨房清洁用品；禁止出现浴缸、马桶、淋浴、浴室台盆、浴室镜柜、浴巾架、浴室纸巾架、卧室或客厅物品。"
		geometryLockRule = "厨房水槽区几何锁定：水槽、台面和水龙头保持同一个真实安装坐标系。水龙头底座固定在水槽后方或侧后方台面孔位，出水口朝向水槽盆内部。相机可以移动到正前方或轻微侧前方观察，但禁止把水槽、水龙头、台面或柜体整体旋转、移动、翻转、重排成不自然展示角度。"
	case "kitchen_island":
		productDescriptor = "kitchen island faucet"
		sceneDesc = "厨房岛台水槽区场景"
		sceneEnvironmentRules = "原创厨房岛台水槽区设计。必需元素：岛台、岛台水槽、连续岛台台面、开放式厨房背景、合理灯光。空间围绕目标水龙头所在的岛台清洗区展开。只能出现厨房和岛台场景物品。"
		installationInstruction = "真实安装：水龙头从岛台水槽后方或侧后方的实心台面孔位升起，底座与岛台台面平齐，出水口延伸至岛台水槽内部。产品高度与岛台和水槽比例协调，可见稳定自然的安装接触面。"
		lightingPrompt = "光线：统一的开放式厨房环境照明，光源位于左上方或岛台上方。阴影与岛台、水槽、台面和厨房背景投射方向一致。产品使用与岛台水槽区一致的自然光线，金属高光自然统一。"
		refProductRule = "厨房岛台水龙头按真实岛台水槽龙头比例呈现，岛台水槽内口宽度必须明显大于水龙头整体宽度和出水口投影宽度；底座固定在岛台水槽后方或侧后方台面孔位，出水口对准水槽内部；保留龙体、弯管、把手、连接杆和底座的截面形状、厚薄关系、边线和相对间距。"
		uniquenessRule = "厨房岛台里只能有这一个可用水龙头和这一个主岛台水槽；禁止背景水龙头、第二个水槽、远处备用洗涤区或镜面反射中的新增水龙头。"
		integrityRule = "厨房岛台区完整性：场景中只有一个主岛台水槽和一个连续岛台台面。岛台、台面、水槽、背景橱柜和灯光属于同一个真实开放式厨房空间。"
		roomObjectRule = "厨房岛台场景只能出现厨房和岛台物品，例如岛台、台面、水槽、橱柜、吊灯、杯具、餐具、食材或少量厨房道具；禁止出现浴缸、马桶、淋浴、浴室台盆、浴室镜柜、卧室或客厅物品。"
		geometryLockRule = "厨房岛台几何锁定：岛台、水槽、台面和水龙头保持同一个真实安装坐标系。水龙头底座固定在岛台水槽后方或侧后方台面孔位，出水口朝向水槽内部。相机可以移动到正前方或轻微侧前方观察，但禁止把岛台、水槽、水龙头、台面或柜体整体旋转、移动、翻转、重排成不自然展示角度。"
	case "bathroom_vanity":
		productDescriptor = "bathroom faucet"
		sceneDesc = "浴室台盆区场景"
		sceneEnvironmentRules = "原创浴室台盆区设计。必需元素：主台盆、连续台面或台盆柜、镜子、墙砖或涂料墙面、地板、合理灯光。浴室布局围绕目标水龙头所在的台盆区展开。台盆高度约81-86cm。只能出现浴室场景物品。"
		installationInstruction = "真实安装：浴室台盆水龙头从台盆后方或侧后方的实心台面孔位升起，旁边是台下盆或台上盆，组成完整台盆柜。水龙头底座与台面平齐，出水口延伸至台盆内部。产品高度与台面比例协调，可见稳定自然的安装接触面。"
		lightingPrompt = "光线：统一的浴室环境照明，光源位于左上方。阴影与台盆柜、镜子、墙砖或墙面投射方向一致。匹配参考图的色温。产品使用与台盆区一致的自然光线，金属高光自然统一。"
		refProductRule = "浴室台盆水龙头按真实台盆龙头比例呈现，台盆内口宽度必须明显大于水龙头整体宽度和出水口投影宽度；底座固定在台盆后方台面孔位，出水口对准台盆内部；保留龙体、弯管、把手、连接杆和底座的截面形状、厚薄关系、边线和相对间距。"
		uniquenessRule = "浴室里只能有这一个台盆水龙头和这一个主台盆；禁止第二个台盆龙头、墙出水、背景水龙头或镜中新增水龙头。"
		integrityRule = "台盆区完整性：场景中只有一个完整台盆柜和一个主台盆。台面、台下盆或台上盆、柜体、镜子和墙面属于同一个连续空间，水龙头与台面自然连接。"
		roomObjectRule = "浴室场景只能出现浴室物品和浴室空间元素，例如台盆、台盆柜、浴室镜、墙砖、洗漱用品、毛巾、地漏或少量浴室收纳；禁止出现厨房水槽、厨房橱柜、防溅墙、锅具、餐具、炉灶、厨房清洁区、卧室或客厅物品。"
		geometryLockRule = "浴室台盆区几何锁定：台盆、台面或台盆柜和水龙头保持同一个真实安装坐标系。水龙头底座固定在台盆后方或侧后方台面孔位，出水口朝向台盆内部。相机可以移动到正前方或轻微侧前方观察，但禁止把台盆、水龙头、台面或柜体整体旋转、移动、翻转、重排成不自然展示角度。"
	case "rv":
		productDescriptor = "modern RV kitchen faucet"
		sceneDesc = "现代房车厨房水槽区场景"
		sceneEnvironmentRules = "原创现代房车厨房场景。必需元素：紧凑厨房水槽、连续小台面、轻量现代柜体、房车窗、车内墙板或顶灯、明亮温馨的生活氛围。空间尺度必须像真实 RV 或 camper van 的厨房区，布局紧凑但整洁，不能变成普通住宅厨房、浴室或其他非房车空间。房车场景不使用风格参考图。"
		installationInstruction = "真实安装：水龙头固定在房车厨房小型水槽后方或侧后方的紧凑台面孔位，底座与台面平齐，出水口延伸至小型水槽内部。产品与房车厨房台面、水槽、柜体和窗户比例协调。"
		lightingPrompt = "光线：明亮温馨的房车厨房光线，自然日光从车窗进入，叠加柔和暖色车内灯光。阴影与紧凑台面、水槽、柜体和车内墙板方向一致，产品金属高光温暖自然。"
		refProductRule = "房车厨房水龙头按真实紧凑厨房水槽龙头比例呈现，小型水槽内口宽度也必须明显大于水龙头整体宽度和出水口投影宽度；底座固定在小型水槽后方或侧后方台面孔位，出水口对准水槽内部；保留参考图产品所有关键结构和比例。"
		uniquenessRule = "房车厨房空间里只能有这一个可用水龙头和这一个小型厨房水槽；禁止生成普通住宅厨房、豪宅厨房、浴室台盆、其他清洗区或第二个水龙头。"
		integrityRule = "房车厨房完整性：水槽、台面、柜体、窗、墙板和灯光都属于同一个现代房车厨房空间，尺度紧凑、明亮、温馨且真实。"
		roomObjectRule = "房车场景只能出现现代房车厨房和旅行生活物品，例如小型厨房水槽、紧凑台面、轻量柜体、车窗、杯具、餐具、咖啡杯或少量旅行用品；禁止出现普通住宅大厨房、浴缸、马桶、淋浴、豪宅岛台、卧室或客厅物品。"
		geometryLockRule = "房车厨房几何锁定：小型厨房水槽、紧凑台面和水龙头保持同一个真实安装坐标系。水龙头底座固定在水槽后方或侧后方台面孔位，出水口朝向水槽内部。相机可以移动到正前方或轻微侧前方观察，但禁止把水槽、水龙头、台面或柜体整体旋转、移动、翻转、重排成不自然展示角度。"
	}
	isKitchenLocation := location == "kitchen_sink" || location == "kitchen_island"
	isBathroomLocation := location == "bathroom_vanity"

	lens := strings.TrimSpace(req.Lens)
	if lens == "" || lens == "default" {
		lens = "medium"
	}

	var lensPrompt string
	var lensRule string
	var proportionRule string
	var framingPrompt string
	closeUpProductLockRule := ""
	switch lens {
	case "close_up":
		lensPrompt = "70-85mm安装区近景镜头，不是孤立产品特写，也不是展示完整长台面的中远景。取景范围限制在一个水槽或台盆的安装区内，目标产品组合和周围安装区域占画面55-70%，目标产品硬件本体占画面28-38%。画面必须完整保留目标产品的出水口、主体、把手、底座、连续台面、部分水槽或台盆内缘和真实出水方向。背景只允许墙面、镜子、毛巾或少量弱化道具，浅景深虚化。"
		lensRule = "近景镜头：取景只围绕一个目标产品安装区，目标产品组合和安装区域占画面55-70%，目标产品硬件本体占画面28-38%；必须保留连续台面、水槽或台盆内缘、完整底座接触面和真实出水方向。"
		framingPrompt = fmt.Sprintf("【近景镜头】：%s是唯一主体。画面是一个水槽或台盆的安装区近景，必须同时包含目标产品完整轮廓、完整底座接触面、连续台面和真实使用区域；安装孔位应被底座自然覆盖，禁止裸露孔洞、悬空底座、错位安装、切断台面、展示第二个水槽/台盆或拉远成整面台面。", productDescriptor)
		closeUpProductLockRule = "近景产品身份锁定：参考图产品身份优先级最高，高于风格、构图、场景和镜头。必须逐项保持参考图里的出水口形状、弯曲程度、截面、主体高度、柱身纹理、把手数量、把手位置、把手形状、底座数量、底座形状、金属/非金属分区和颜色分布；禁止把目标产品改成弧形龙头、鹅颈龙头、圆管龙头、桥式龙头、常见浴室龙头或其他通用款式。若参考图是直线或扁平出水口，必须保持直线或扁平出水口；若参考图是分体把手，必须保持分体把手。近景画面中只能出现这一套目标水龙头，禁止背景水龙头、第二套龙头、远处备用洗手区、墙出水、镜中重复龙头、反射复制龙头或任何形似龙头的装饰五金。尺寸关系硬约束：水槽或台盆内口宽度必须明显大于水龙头整体宽度和出水口投影宽度，近景裁切也不能让水槽或台盆看起来比水龙头窄。"
	case "wide":
		lensPrompt = "24mm广角镜头。全景尽收眼底。产品安装区域位于画面中心附近作为视觉锚点。统一渲染，场景和产品一起生成。"
		lensRule = "远景镜头：产品占画面5-15%，展示完整空间，产品位于画面中心附近作为视觉锚点。"
		framingPrompt = fmt.Sprintf("【远景镜头】：%s占据画面5-15%%。产品较小但位于核心使用区，广角展示完整空间。", productDescriptor)
	case "medium":
		lensPrompt = "50mm标准镜头，中景拍摄。产品占画面22-30%，保留完整安装区域和周围场景。产品、台面和安装区域必须一体化生成。"
		lensRule = "中景镜头：产品占画面22-30%，安装区域和周围场景完整，产品作为清晰功能焦点。"
		framingPrompt = fmt.Sprintf("【中景镜头】：%s占据画面22-30%%。画面包含产品、连续台面、主要使用区和部分周围空间。", productDescriptor)
	default:
		lens = "medium"
		lensPrompt = "50mm标准镜头，中景拍摄。产品占画面22-30%，保留完整安装区域和周围场景。产品、台面和安装区域必须一体化生成。"
		lensRule = "中景镜头：产品占画面22-30%，安装区域和周围场景完整，产品作为清晰功能焦点。"
		framingPrompt = fmt.Sprintf("【中景镜头】：%s占据画面22-30%%。画面包含产品、连续台面、主要使用区和部分周围空间。", productDescriptor)
	}

	if isKitchenLocation {
		switch lens {
		case "close_up":
			proportionRule = "尺寸：厨房水龙头按真实水槽龙头比例缩放，高度约为水槽盆深的1.5-2.5倍。近景只能由相机距离和裁切形成，产品尺寸不变；画面必须保留水槽盆口、连续台面和被底座覆盖的真实安装孔位，禁止为了近景放大产品或移动底座。"
		case "wide":
			proportionRule = "尺寸：厨房水龙头按真实水槽龙头比例缩放，高度约为水槽盆深的1.5-2.5倍。广角展示厨房水槽区，产品尺寸不变。"
		default:
			proportionRule = "尺寸：厨房水龙头按真实水槽龙头比例缩放，高度约为水槽盆深的1.5-2.5倍。中景取景，产品尺寸不变。"
		}
	} else if isBathroomLocation {
		switch lens {
		case "close_up":
			proportionRule = "尺寸：浴室台盆水龙头按真实台盆龙头比例缩放，高度与台盆和台面协调。近景只能由相机距离和裁切形成，产品尺寸不变；画面必须保留台盆盆口、连续台面和被底座覆盖的真实安装孔位，禁止为了近景放大产品或移动底座。"
		case "wide":
			proportionRule = "尺寸：浴室台盆水龙头按真实台盆龙头比例缩放，高度与台盆和台面协调。广角展示完整台盆区，产品尺寸不变。"
		default:
			proportionRule = "尺寸：浴室台盆水龙头按真实台盆龙头比例缩放，高度与台盆和台面协调。中景取景，产品尺寸不变。"
		}
	} else {
		proportionRule = "尺寸：产品按参考图真实比例呈现，画面由镜头远近形成变化，不改变产品本体尺寸。"
	}

	colorInstruction := ""
	if strings.TrimSpace(req.ProductColorFamilyImageURL) != "" || strings.TrimSpace(req.ProductColorFamily) != "" {
		colorInstruction = " 颜色变换：颜色参考图只定义目标产品硬件颜色，不影响台面、墙面、柜体、地板或软装。保留参考图产品结构、功能细节和表面质感；如果是金属色，保持对应的金属高光、反射和阴影。"
	}

	stylePrompt := fmt.Sprintf("风格：%s美学风格；风格参考图只影响灯光、材质、色调和装饰密度，布局重新生成。", style)
	sceneStyleRule := "风格参考图只作为 moodboard，抽取灯光、材质、色调、软装氛围和装饰密度；必须重新设计新的空间布局。"
	if !useStyleReference {
		stylePrompt = "风格：真实自然生活方式摄影；使用地点内置场景模板，不使用风格参考图。"
		sceneStyleRule = "使用地点内置场景模板生成空间氛围，不依赖风格参考图；必须保持真实自然的生活方式摄影质感。"
	}
	cameraAnglePrompt := "相机：位于产品正前方到轻微侧前方之间，略高于台面，轻微向下俯拍，形成自然生活方式摄影视角。相机只改变观察位置，不改变产品、台面、水槽、台盆或安装结构的朝向。禁止为了形成视角而旋转、移动、翻转或重排产品和场景结构；不要从产品背面、安装面后方或遮挡角度拍摄。"

	refInstruction := "参考图使用规则：产品白底图是唯一产品依据，只定义目标产品身份、外形轮廓、截面形状、把手、出水口、底座、表面颜色和功能细节；它不是场景图，不保留白底、阴影、展示尺寸或构图。"
	if useStyleReference {
		refInstruction += " 风格参考图只定义场景气质、灯光、材质、色调和软装氛围，不复制布局、位置、镜头或道具。"
	}
	refInstruction += " " + refProductRule
	if useStyleReference && ref != nil {
		if hint := strings.TrimSpace(ref.PromptHint); hint != "" {
			refInstruction += " 风格补充仅用于灯光、材质和氛围：" + hint + "。"
		}
	}
	personPrompt := ""
	if req.IncludePerson {
		personPrompt = buildScenePersonPrompt(location, lens)
	}
	sceneOriginalityConstraint := "场景原创，只吸收风格参考图的灯光、材质、色调和装饰密度。"
	if !useStyleReference {
		sceneOriginalityConstraint = "场景原创，使用地点内置模板生成空间、灯光、材质和装饰密度，不依赖风格参考图。"
	}

	constraints := "生成要求：" +
		"（1）" + lensRule +
		"（2）参考图产品形状、比例、外观和功能细节保持一致，不能加粗、简化或改成常见款式。" +
		"（3）镜头展示产品正面或正侧面，不出现产品背面视角。" +
		"（4）产品与台面或安装面自然接触，阴影和光线统一。" +
		"（5）" + sceneOriginalityConstraint +
		"（6）" + uniquenessRule +
		"（7）" + integrityRule +
		"（8）" + roomObjectRule +
		"（9）" + geometryLockRule
	if closeUpProductLockRule != "" {
		constraints += "（10）" + closeUpProductLockRule
	}

	qualityPrompt := "逼真照片级画质，自然光线，高端商业摄影品质。产品方向、结构和功能部件与参考图一致。色彩分级一致，纹理保留自然，画面清晰不过度锐化。"

	prompt := fmt.Sprintf("核心要求：%s %s", framingPrompt, proportionRule)
	if closeUpProductLockRule != "" {
		prompt += " " + closeUpProductLockRule
	}
	prompt += " " + lensPrompt
	prompt += fmt.Sprintf(" 高端编辑风格生活照片。主角产品是%s的核心，%s %s", sceneDesc, sceneEnvironmentRules, sceneStyleRule)
	prompt += fmt.Sprintf(" 这是一款%s。安装位置：%s。%s", productDescriptor, installationInstruction, colorInstruction)
	if personPrompt != "" {
		prompt += " " + personPrompt
	}
	if location == "kitchen_sink" {
		prompt += " 场景融入：厨房水龙头与厨房水槽和台面是一体化渲染，不是放在场景上，而是属于水槽区。台面完整，水龙头从水槽后方台面孔位自然升起，底座与台面接触面平整连贯，出水口朝向水槽盆内部。"
	} else if location == "kitchen_island" {
		prompt += " 场景融入：厨房岛台水龙头与岛台水槽和台面是一体化渲染，不是放在场景上，而是属于岛台清洗区。岛台台面完整，水龙头从岛台水槽后方台面孔位自然升起，底座与台面接触面平整连贯，出水口朝向水槽内部。"
	} else if isBathroomLocation {
		prompt += " 场景融入：浴室台盆水龙头与台面和台盆柜是一体化渲染，不是放在场景上，而是属于台盆区。台面完整，水龙头从台面中自然升起，底座与台面接触面平整连贯，出水口朝向台盆内部。"
	} else if location == "rv" {
		prompt += " 场景融入：房车水龙头与现代房车厨房的小型水槽、紧凑台面、轻量柜体和车窗环境是一体化渲染，属于真实明亮温馨的房车厨房区。不要把空间扩展成普通住宅厨房、浴室或其他非房车空间。"
	}
	prompt += fmt.Sprintf(" %s.", stylePrompt)
	prompt += fmt.Sprintf(" %s.", lightingPrompt)
	prompt += fmt.Sprintf(" %s.", cameraAnglePrompt)
	prompt += fmt.Sprintf(" %s.", refInstruction)
	prompt += fmt.Sprintf(" %s.", constraints)
	prompt += fmt.Sprintf(" %s.", qualityPrompt)
	if extra := strings.TrimSpace(req.ExtraPrompt); extra != "" {
		prompt += " 用户补充要求：" + extra
	}
	return prompt
}

func promptVariant(prompt string, index int, sceneType string, productCategory string) string {
	if index == 0 {
		return prompt
	}
	variation := getSceneVariationText(sceneType, productCategory, index)
	if variation == "" {
		return prompt
	}
	return strings.Join([]string{prompt, variation}, " ")
}

func resolveSceneType(category string) string {
	text := strings.ToLower(strings.TrimSpace(category))
	hasKitchen := strings.Contains(text, "厨房") || strings.Contains(text, "岛台") || strings.Contains(text, "kitchen") || strings.Contains(text, "island") || strings.Contains(text, "厨")
	hasBathroom := strings.Contains(text, "浴室") || strings.Contains(text, "卫生间") || strings.Contains(text, "洗手间") || strings.Contains(text, "面盆") || strings.Contains(text, "台盆") || strings.Contains(text, "浴") || strings.Contains(text, "bathroom") || strings.Contains(text, "vanity")
	if hasKitchen && !hasBathroom {
		return "kitchen"
	}
	if hasBathroom && !hasKitchen {
		return "bathroom"
	}
	if hasKitchen {
		return "kitchen"
	}
	return "generic"
}

func normalizeSceneLocation(value string) string {
	text := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(text, "房车") || strings.Contains(text, "rv") || strings.Contains(text, "camper"):
		return "rv"
	case strings.Contains(text, "岛台") || strings.Contains(text, "island"):
		return "kitchen_island"
	case strings.Contains(text, "浴室") || strings.Contains(text, "卫生间") || strings.Contains(text, "洗手间") || strings.Contains(text, "台盆") || strings.Contains(text, "bathroom") || strings.Contains(text, "vanity"):
		return "bathroom_vanity"
	case strings.Contains(text, "厨房") || strings.Contains(text, "kitchen") || strings.Contains(text, "厨"):
		return "kitchen_sink"
	default:
		return "kitchen_sink"
	}
}

func sceneLocationSupportsStyleReference(value string) bool {
	switch normalizeSceneLocation(value) {
	case "rv":
		return false
	default:
		return true
	}
}

func buildScenePersonPrompt(location string, lens string) string {
	base := "人物规则：人物必须与水龙头或水槽/台盆有自然生活交互，例如轻触把手、在出水口下洗手、冲洗蔬果、整理杯具或在水槽旁操作；人物只是辅助元素，不作为主体。人物不能遮挡、替换、重绘、触碰变形目标产品；手部不能覆盖出水口、把手、底座、安装孔位和产品轮廓。禁止多人、夸张摆拍、脸部特写、广告模特式主视觉、手持产品或把产品拿离安装位置。"
	lensRule := "人物景别：中景只出现一名侧身或半身人物，人物位于画面边缘或水槽/台盆旁，一只手与水龙头或水流自然交互，产品仍是清晰功能焦点。"
	switch strings.TrimSpace(lens) {
	case "close_up":
		lensRule = "人物景别：近景只出现手部或前臂，不出现完整人物、脸部或大面积身体；手部在水龙头把手、水流或水槽/台盆边缘进行轻微交互，不能遮挡产品关键轮廓。"
	case "wide":
		lensRule = "人物景别：远景可以出现一名完整人物，但人物比例较小并处在使用空间内，一只手与水龙头、水流或水槽/台盆自然交互，不能抢走产品主体位置。"
	}
	switch location {
	case "kitchen_sink":
		return base + " " + lensRule + " 厨房人物动作：人物在水槽区清洗蔬果、轻触龙头把手、接水或整理杯具；服装、姿态和光线必须符合真实厨房环境。"
	case "kitchen_island":
		return base + " " + lensRule + " 厨房岛台人物动作：人物位于岛台一侧，可以在岛台水槽边清洗食材、拿杯接水或轻触龙头把手；不能站在产品正前方遮挡水龙头，岛台、水槽和水龙头仍是画面功能焦点。"
	case "bathroom_vanity":
		return base + " " + lensRule + " 浴室人物动作：人物在台盆边洗手、轻触把手或整理洗漱用品；形象自然端庄，避免过度近距离脸部特写。"
	case "rv":
		return base + " " + lensRule + " 房车人物动作：人物在现代房车厨房的小水槽边拿杯接水、清洗杯具或轻触龙头把手；人物尺度必须适应紧凑房车厨房，不能让空间变成普通住宅厨房、浴室或其他非房车空间。"
	default:
		return base + " " + lensRule
	}
}

func buildPromptValue(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

// ===== 场景风格分类管理 =====

type sceneStyleCategoryInput struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
	Sort int    `json:"sort"`
}

func ListSceneStyleCategories(c *gin.Context) {
	var categories []model.SceneStyleCategory
	if err := database.DB.Order("sort asc, id asc").Find(&categories).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取场景分类失败"})
		return
	}
	c.JSON(http.StatusOK, categories)
}

func AdminListSceneStyleCategories(c *gin.Context) {
	ListSceneStyleCategories(c)
}

func AdminCreateSceneStyleCategory(c *gin.Context) {
	var input sceneStyleCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "场景名称不能为空"})
		return
	}

	item := model.SceneStyleCategory{
		Name: strings.TrimSpace(input.Name),
		Icon: strings.TrimSpace(input.Icon),
		Sort: input.Sort,
	}
	if err := database.DB.Create(&item).Error; err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "场景名称已存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建场景分类失败"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func AdminUpdateSceneStyleCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var item model.SceneStyleCategory
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "场景分类不存在"})
		return
	}

	var input sceneStyleCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "场景名称不能为空"})
		return
	}

	item.Name = strings.TrimSpace(input.Name)
	item.Icon = strings.TrimSpace(input.Icon)
	item.Sort = input.Sort
	if err := database.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新场景分类失败"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func AdminDeleteSceneStyleCategory(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("scene_id = ?", id).Delete(&model.SceneStyleTag{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.SceneStyleCategory{}, id).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除场景分类失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ===== 产品色系管理 =====

type productColorFamilyInput struct {
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
	Sort     int    `json:"sort"`
}

func ListProductColorFamilies(c *gin.Context) {
	var items []model.ProductColorFamily
	if err := database.DB.Order("sort asc, id asc").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取色系失败"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func AdminListProductColorFamilies(c *gin.Context) {
	ListProductColorFamilies(c)
}

func AdminCreateProductColorFamily(c *gin.Context) {
	var input productColorFamilyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "色系名称不能为空"})
		return
	}

	item := model.ProductColorFamily{
		Name:     strings.TrimSpace(input.Name),
		ImageURL: strings.TrimSpace(input.ImageURL),
		Sort:     input.Sort,
	}
	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建色系失败"})
		return
	}
	c.JSON(http.StatusCreated, item)
}

func AdminUpdateProductColorFamily(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var item model.ProductColorFamily
	if err := database.DB.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "色系不存在"})
		return
	}

	var input productColorFamilyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "色系名称不能为空"})
		return
	}

	item.Name = strings.TrimSpace(input.Name)
	item.ImageURL = strings.TrimSpace(input.ImageURL)
	item.Sort = input.Sort
	if err := database.DB.Save(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新色系失败"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func AdminDeleteProductColorFamily(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := database.DB.Delete(&model.ProductColorFamily{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除色系失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
