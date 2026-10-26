package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

/** EraserRequest 消除文字/水印请求 */
type EraserRequest struct {
	ImageData string `json:"image_data" binding:"required"`
	MimeType  string `json:"mime_type"`
}

/** EraserImage handles POST /api/eraser */
func EraserImage(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req EraserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mimeType := req.MimeType
	if mimeType == "" {
		mimeType = "image/png"
	}

	apiKey := getSettingValue("draw_key")
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绘图 API Key 未配置"})
		return
	}

	drawModel := getDefaultModelByType("drawing", "")
	if drawModel == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未配置默认绘图模型，请在模型管理中设置"})
		return
	}

	usageLog := model.UsageLog{
		UserID:    userID,
		UsageType: "eraser",
		Source:    "eraser",
		Model:     drawModel,
		Prompt:    "remove text and watermarks",
		Status:    "failed",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	/**
	 * 使用纯描述性 prompt：描述结果图的期望状态。
	 * 明确说明哪些需要消除（文字/水印/Logo），哪些必须完全保留（主体物体/背景）。
	 * 使用 EditImage（图片编辑通道）而非 GenerateImageWithRef（图片生成通道），
	 * 确保 AI 在原图基础上做局部修改，而不是重新生成一张新图。
	 */
	prompt := "The exact same photo with all text, watermarks, logos, and overlay graphics completely removed. Every other element — including all objects, people, vehicles, backgrounds, colors, lighting, and composition — must remain absolutely identical to the original image. Only surface text and overlay marks are absent."

	// Get base URL for adapter selection
	baseURL := resolveGPTImageBaseURL()
	if baseURL == "" {
		baseURL = getSettingValue("access_url")
	}

	var imageURL string
	var resultMimeType string
	var err error

	// 根据 URL 和模型名综合判断使用哪个适配器
	if adapter.IsApizURL(baseURL) {
		if adapter.IsGPTImageModel(drawModel) {
			// apiz.ai + GPT 图片模型 → 使用 ApizGPTImageAdapter
			apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, baseURL)
			result, apizErr := apizAdapter.EditImage(c.Request.Context(), prompt, req.ImageData, mimeType, "1:1", "")
			if apizErr != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("消除处理失败: %v", apizErr)})
				return
			}

			if result.ImageData != "" {
				imageURL, err = storage.SaveBase64Image(result.ImageData, result.MimeType)
				resultMimeType = result.MimeType
			} else if result.ImageURL != "" {
				imageURL = strings.TrimSpace(result.ImageURL)
				err = nil
				resultMimeType = result.MimeType
			} else {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 未返回图片，请重试"})
				return
			}
		} else {
			// apiz.ai + Gemini 模型 → 使用 ApizGeminiAdapter
			apizAdapter := adapter.NewApizGeminiAdapter(apiKey, baseURL)
			refImages := []adapter.RefImage{{Data: req.ImageData, MimeType: mimeType}}
			result, apizErr := apizAdapter.GenerateImageWithRefs(c.Request.Context(), prompt, refImages, 1, "")
			if apizErr != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("消除处理失败: %v", apizErr)})
				return
			}

			if result.ImageData != "" {
				imageURL, err = storage.SaveBase64Image(result.ImageData, result.MimeType)
				resultMimeType = result.MimeType
			} else if result.ImageURL != "" {
				imageURL = strings.TrimSpace(result.ImageURL)
				err = nil
				resultMimeType = result.MimeType
			} else {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 未返回图片，请重试"})
				return
			}
		}
	} else if adapter.IsGPTImageModel(drawModel) {
		// 非 apiz URL + GPT 图片模型 → 使用旧版 GPTImageAdapter
		gptAdapter := adapter.NewGPTImageAdapter(apiKey, baseURL)
		result, gptErr := gptAdapter.EditImage(c.Request.Context(), prompt, []adapter.RefImage{{Data: req.ImageData, MimeType: mimeType}}, drawModel, "1:1", "", 1)
		if gptErr != nil {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("消除处理失败: %v", gptErr)})
			return
		}

		if strings.TrimSpace(result.ImageData) == "" && strings.TrimSpace(result.ImageURL) == "" {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 未返回图片，请重试"})
			return
		}

		if strings.TrimSpace(result.ImageData) != "" {
			imageURL, err = storage.SaveBase64Image(result.ImageData, result.MimeType)
		} else {
			imageURL, _, err = storage.SaveImageFromURL(result.ImageURL)
		}
		resultMimeType = result.MimeType
	} else {
		// 其他情况 → 使用旧版 GeminiAdapter
		geminiAdapter := adapter.NewGeminiAdapter(apiKey, baseURL)
		geminiAdapter.RetryLogger = func(attempt, maxAttempts int, retryErr error) {
			database.DB.Create(&model.UsageLog{
				UserID:       usageLog.UserID,
				UsageType:    usageLog.UsageType,
				Source:       usageLog.Source,
				Model:        usageLog.Model,
				Status:       "retry",
				ResponseBody: "",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			})
		}
		result, geminiErr := geminiAdapter.EditImage(c.Request.Context(), prompt, req.ImageData, mimeType, drawModel, "1:1", "")
		if geminiErr != nil {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("消除处理失败: %v", geminiErr)})
			return
		}

		if result.ImageData == "" && strings.TrimSpace(result.ImageURL) == "" {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 未返回图片，请重试"})
			return
		}

		imageURL, resultMimeType, err = saveGeminiImageResult(result)
	}

	if err != nil {
		database.DB.Create(&usageLog)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
		return
	}

	usageLog.Status = "success"
	usageLog.Result = imageURL
	usageLog.ImageURL = imageURL
	database.DB.Create(&usageLog)
	go incUsageCount(userID, drawModel)

	c.JSON(http.StatusOK, gin.H{
		"image_url": imageURL,
		"mime_type": resultMimeType,
	})
}
