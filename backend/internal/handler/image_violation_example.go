package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
)

type imageViolationExampleInput struct {
	Platform    string `json:"platform"`
	ProblemType string `json:"problem_type"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
}

func (input *imageViolationExampleInput) validate() error {
	input.Platform = strings.TrimSpace(input.Platform)
	input.ProblemType = strings.TrimSpace(input.ProblemType)
	input.Description = strings.TrimSpace(input.Description)
	input.ImageURL = strings.TrimSpace(input.ImageURL)
	switch input.Platform {
	case "Amazon", "TikTok", "Homedepot", "TEMU", "Walmart", "Lowes", "Wayfair":
	default:
		return fmt.Errorf("请选择有效的平台")
	}
	if input.ProblemType == "" || len([]rune(input.ProblemType)) > 100 {
		return fmt.Errorf("问题类型必填，最多 100 个字")
	}
	if input.Description == "" || len([]rune(input.Description)) > 5000 {
		return fmt.Errorf("问题描述必填，最多 5000 个字")
	}
	imageURL, err := url.Parse(input.ImageURL)
	if err != nil || input.ImageURL == "" {
		return fmt.Errorf("请上传附件图片")
	}
	localImage := imageURL.Scheme == "" && imageURL.Host == "" && strings.HasPrefix(imageURL.Path, "/uploads/prompts/")
	remoteImage := (imageURL.Scheme == "http" || imageURL.Scheme == "https") && imageURL.Host != ""
	if !localImage && !remoteImage {
		return fmt.Errorf("附件必须是已上传的图片，不能使用本地临时地址")
	}
	return nil
}

func ListImageViolationExamples(c *gin.Context) {
	items := make([]model.ImageViolationExample, 0)
	if err := database.DB.Order("created_at DESC, id DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取违规图片示例失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func CreateImageViolationExample(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
		return
	}
	var input imageViolationExampleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "提交内容格式错误"})
		return
	}
	if err := input.validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user model.User
	if err := database.DB.Select("id", "name").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无法获取当前用户"})
		return
	}
	item := model.ImageViolationExample{
		Platform: input.Platform, ProblemType: input.ProblemType, Description: input.Description,
		ImageURL: input.ImageURL, UserID: user.ID, Submitter: user.Name,
	}
	if err := database.DB.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存违规图片示例失败"})
		return
	}
	c.JSON(http.StatusCreated, item)
}
