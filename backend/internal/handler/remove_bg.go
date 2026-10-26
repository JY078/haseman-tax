package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

/** RemoveBgRequest 去除背景请求 */
type RemoveBgRequest struct {
	ImageData  string `json:"image_data" binding:"required"`
	MimeType   string `json:"mime_type"`
	Background string `json:"background"` // "transparent", "white", "black", "custom"
	CustomBg   string `json:"custom_bg"`  // 自定义背景描述（当 background = "custom" 时使用）
}

/** RemoveBg handles POST /api/remove-bg */
func RemoveBg(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req RemoveBgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mimeType := req.MimeType
	if mimeType == "" {
		mimeType = "image/png"
	}

	background := req.Background
	if background == "" {
		background = "transparent"
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
		UsageType: "remove_bg",
		Source:    "remove-bg",
		Model:     drawModel,
		Prompt:    fmt.Sprintf("remove background: %s", background),
		Status:    "failed",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	/** 纯描述性 prompt：描述结果图的状态，避免操作类词汇触发 MALFORMED_FUNCTION_CALL */
	var prompt string
	switch background {
	case "white":
		prompt = "The exact same subject from this photo, photographed against a clean solid white background. Subject appearance, lighting and details unchanged."
	case "black":
		prompt = "The exact same subject from this photo, photographed against a pure solid black background. Subject appearance, lighting and details unchanged."
	case "custom":
		customDesc := req.CustomBg
		if customDesc == "" {
			customDesc = "a clean studio background"
		}
		prompt = fmt.Sprintf("The exact same subject from this photo, placed in front of %s. Subject appearance unchanged, smooth edges.", customDesc)
	default:
		prompt = "The main subject from this photo isolated on a transparent background. No background elements visible, smooth natural edges around the subject."
	}

	// Get base URL for adapter selection
	baseURL := resolveGPTImageBaseURL()
	if baseURL == "" {
		baseURL = getSettingValue("access_url")
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	log.Printf("[RemoveBg] using model=%s baseURL=%s apiz=%v gpt=%v", drawModel, baseURL, adapter.IsApizURL(baseURL), adapter.IsGPTImageModel(drawModel))

	// 根据 URL 和模型名综合判断使用哪个适配器
	var imageURL string
	var resultMimeType string
	var err error

	if adapter.IsApizURL(baseURL) {
		if adapter.IsGPTImageModel(drawModel) {
			// apiz.ai + GPT 图片模型 → 使用 ApizGPTImageAdapter
			log.Printf("[RemoveBg] executing Apiz GPT Image adapter")
			apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, baseURL)
			result, genErr := apizAdapter.EditImage(c.Request.Context(), prompt, req.ImageData, mimeType, "1:1", "")
			if genErr != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("背景去除失败: %v", genErr)})
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
			log.Printf("[RemoveBg] executing Apiz Gemini adapter")
			apizAdapter := adapter.NewApizGeminiAdapter(apiKey, baseURL)
			refImages := []adapter.RefImage{
				{Data: req.ImageData, MimeType: mimeType},
			}
			result, genErr := apizAdapter.GenerateImageWithRefs(c.Request.Context(), prompt, refImages, 1, "")
			if genErr != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("背景去除失败: %v", genErr)})
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
		log.Printf("[RemoveBg] executing GPT Image adapter")
		gptAdapter := adapter.NewGPTImageAdapter(apiKey, baseURL)
		refImages := []adapter.RefImage{
			{Data: req.ImageData, MimeType: mimeType},
		}
		result, genErr := gptAdapter.EditImage(c.Request.Context(), prompt, refImages, drawModel, "1:1", "", 1)
		if genErr != nil {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("背景去除失败: %v", genErr)})
			return
		}

		if result.ImageData != "" {
			imageURL, err = storage.SaveBase64Image(result.ImageData, result.MimeType)
			resultMimeType = result.MimeType
		} else if result.ImageURL != "" {
			imageURL, _, err = storage.SaveImageFromURL(result.ImageURL)
			resultMimeType = result.MimeType
		} else {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 未返回图片，请重试"})
			return
		}
	} else {
		// 其他情况 → 使用旧版 GeminiAdapter
		log.Printf("[RemoveBg] executing Gemini adapter")
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
		result, genErr := geminiAdapter.EditImage(c.Request.Context(), prompt, req.ImageData, mimeType, drawModel, "1:1", "")
		if genErr != nil {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("背景去除失败: %v", genErr)})
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
	if referenceImageURL, refErr := storage.SaveBase64Image(req.ImageData, mimeType); refErr != nil {
		log.Printf("[RemoveBg] save reference image failed: %v", refErr)
	} else {
		usageLog.ReferenceImageURL = referenceImageURL
	}
	database.DB.Create(&usageLog)
	go incUsageCount(userID, drawModel)

	c.JSON(http.StatusOK, gin.H{
		"image_url": imageURL,
		"mime_type": resultMimeType,
	})
}
