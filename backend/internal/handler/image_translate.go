package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/storage"

	"github.com/gin-gonic/gin"
)

/** ImageTranslateRequest 图片翻译请求 */
type ImageTranslateRequest struct {
	ImageData      string `json:"image_data"`
	ImageURL       string `json:"image_url"` /** 图片URL，优先于 ImageData */
	MimeType       string `json:"mime_type"`
	TargetLanguage string `json:"target_language"`
	AspectRatio    string `json:"aspect_ratio"`
	Prompt         string `json:"prompt"`
	Resolution     string `json:"resolution"`
}

/** languageDisplayNames 目标语言对应的显示名称 */
var languageDisplayNames = map[string]string{
	"zh": "简体中文",
	"en": "英文",
	"de": "德文",
	"fr": "法文",
	"it": "意大利文",
	"es": "西班牙文",
	"nl": "荷兰文",
	"pl": "波兰文",
	"sv": "瑞典文",
}

/** ImageTranslate handles POST /api/image-translate */
func ImageTranslate(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req ImageTranslateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mimeType := req.MimeType
	if mimeType == "" {
		mimeType = "image/png"
	}

	// Determine image source: prefer image_url if provided, otherwise use image_data
	imageData := req.ImageData
	if req.ImageURL != "" {
		// Download from URL
		localPath, dlMime, dlErr := storage.DownloadToTemp(req.ImageURL)
		if dlErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("下载图片失败: %v", dlErr)})
			return
		}
		// Read file and convert to base64
		data, readErr := os.ReadFile(localPath)
		if readErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取图片失败"})
			return
		}
		imageData = base64.StdEncoding.EncodeToString(data)
		mimeType = dlMime
		os.Remove(localPath) // Clean up temp file
	}
	if imageData == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "需要提供 image_data 或 image_url"})
		return
	}

	targetLang := req.TargetLanguage
	if targetLang == "" {
		targetLang = "zh"
	}

	// Get language display name
	langDisplayName := languageDisplayNames[targetLang]
	if langDisplayName == "" {
		langDisplayName = targetLang
	}
	finalPrompt := buildImageTranslatePrompt(req.Prompt, langDisplayName)

	// Get API key and base URL (优先使用绘图 key 和 base url)
	apiKey := getSettingValue("draw_key")
	if apiKey == "" {
		apiKey = getSettingValue("chat_key")
	}
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "绘图 API Key 未配置"})
		return
	}

	// Get base URL for adapter selection
	baseURL := resolveGPTImageBaseURL()
	if baseURL == "" {
		baseURL = getSettingValue("access_url")
	}

	// Get drawing model
	drawModel := getDefaultModelByType("drawing", "")
	if drawModel == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未配置默认绘图模型，请在模型管理中设置"})
		return
	}

	// Create usage log
	usageLog := model.UsageLog{
		UserID:     userID,
		UsageType:  "image_translate",
		Source:     "image_translate",
		Model:      drawModel,
		Prompt:     finalPrompt,
		Resolution: req.Resolution,
		Status:     "failed",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Determine aspect ratio and resolution
	aspectRatio := req.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}
	resolution := req.Resolution
	if resolution == "" {
		resolution = "2K" // 默认使用2K分辨率
	}

	// Prepare reference image
	refImages := []adapter.RefImage{
		{
			Data:     imageData,
			MimeType: mimeType,
		},
	}

	// 根据 URL 和模型名综合判断使用哪个适配器
	var imageURL string
	var resultMimeType string

	if adapter.IsApizURL(baseURL) {
		if adapter.IsGPTImageModel(drawModel) {
			// apiz.ai + GPT 图片模型 → 使用 ApizGPTImageAdapter
			apizAdapter := adapter.NewApizGPTImageAdapter(apiKey, baseURL)

			var result *adapter.ApizGPTImageResult
			var err error
			if imageData != "" {
				result, err = apizAdapter.EditImage(c.Request.Context(), finalPrompt, imageData, mimeType, aspectRatio, resolution)
			} else {
				result, err = apizAdapter.GenerateImage(c.Request.Context(), finalPrompt, aspectRatio, resolution)
			}
			if err != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("图片翻译失败: %v", err)})
				return
			}

			if result.ImageData == "" && result.ImageURL == "" {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 未返回图片，请重试"})
				return
			}

			// Save the result image
			if strings.TrimSpace(result.ImageData) != "" {
				imageURL, err = storage.SaveBase64Image(result.ImageData, result.MimeType)
				resultMimeType = result.MimeType
			} else {
				imageURL = strings.TrimSpace(result.ImageURL)
				err = nil
				resultMimeType = result.MimeType
			}
			if err != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
				return
			}
		} else {
			// apiz.ai + Gemini 模型 → 使用 ApizGeminiAdapter
			apizAdapter := adapter.NewApizGeminiAdapter(apiKey, baseURL)
			result, err := apizAdapter.GenerateImageWithRefs(c.Request.Context(), finalPrompt, refImages, 1, resolution, aspectRatio)
			if err != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("图片翻译失败: %v", err)})
				return
			}

			/** 检查结果：新的适配器可能返回 ImageURL 或 ImageData */
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
			if err != nil {
				database.DB.Create(&usageLog)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
				return
			}
		}
	} else if adapter.IsGPTImageModel(drawModel) {
		// 非 apiz URL + GPT 图片模型 → 使用旧版 GPTImageAdapter
		gptAdapter := adapter.NewGPTImageAdapter(apiKey, baseURL)

		var result *adapter.GPTImageResult
		var err error
		if imageData != "" {
			result, err = gptAdapter.EditImage(c.Request.Context(), finalPrompt, refImages, drawModel, aspectRatio, resolution, 1)
		} else {
			result, err = gptAdapter.GenerateImage(c.Request.Context(), finalPrompt, drawModel, aspectRatio, resolution, 1)
		}
		if err != nil {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("图片翻译失败: %v", err)})
			return
		}

		if strings.TrimSpace(result.ImageData) != "" {
			imageURL, err = storage.SaveBase64Image(result.ImageData, result.MimeType)
			resultMimeType = result.MimeType
		} else if strings.TrimSpace(result.ImageURL) != "" {
			imageURL, _, err = storage.SaveImageFromURL(result.ImageURL)
			resultMimeType = result.MimeType
		} else {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 未返回图片，请重试"})
			return
		}
		if err != nil {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
			return
		}
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
		result, err := geminiAdapter.EditImage(c.Request.Context(), finalPrompt, imageData, mimeType, drawModel, aspectRatio, resolution)
		if err != nil {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("图片翻译失败: %v", err)})
			return
		}

		imageURL, resultMimeType, err = saveGeminiImageResult(result)
		if err != nil {
			database.DB.Create(&usageLog)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败"})
			return
		}
	}

	// 确保 imageURL 不为空
	if strings.TrimSpace(imageURL) == "" {
		database.DB.Create(&usageLog)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "图片保存失败：返回空URL"})
		return
	}

	// Update usage log as success
	usageLog.Duration = int(time.Since(usageLog.CreatedAt).Milliseconds())
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

func buildImageTranslatePrompt(userPrompt string, targetLanguage string) string {
	basePrompt := strings.TrimSpace(userPrompt)
	if basePrompt == "" {
		basePrompt = fmt.Sprintf("Localize the existing visible text in this ecommerce image into natural %s ecommerce advertising copy.", targetLanguage)
	}

	return fmt.Sprintf(`%s

Strict image translation rules:
- Translate only text that is already visible in the input image into %s.
- The translation must read like native ecommerce image copy: fluent, concise, benefit-focused, and suitable for product listing banners, feature callouts, and promotional graphics.
- Do not use word-by-word, literal, stiff, or machine-translation wording. Adapt phrases into natural local marketing expressions while preserving the original selling point and intent.
- Keep each translated text block short enough to fit the original text area. Prefer compact ecommerce wording over long explanatory sentences.
- Do not add any new words, labels, captions, slogans, icons, badges, watermarks, or decorative text.
- Do not invent missing text. If a text region is unclear, keep that region visually unchanged.
- Preserve the original number of text blocks, their positions, size, color, typography, alignment, hierarchy, and layout.
- Preserve all non-text visual content, products, people, background, composition, lighting, colors, and image aspect ratio.
- Replace each original text block in place with localized ecommerce copy only; the output must not contain extra text outside original text areas.`, basePrompt, targetLanguage)
}

/** translateLanguageNames 翻译语言名称到目标语言 */
func translateLanguageNames(targetLang string) string {
	switch strings.ToLower(targetLang) {
	case "zh", "chinese", "简体中文":
		return "简体中文"
	case "en", "english", "英文":
		return "英文"
	case "de", "german", "德语", "德文":
		return "德文"
	case "fr", "french", "法语", "法文":
		return "法文"
	case "it", "italian", "意大利语", "意大利文":
		return "意大利文"
	case "es", "spanish", "西班牙语", "西班牙文":
		return "西班牙文"
	case "nl", "dutch", "nederlands", "荷兰语", "荷兰文":
		return "荷兰文"
	case "pl", "polish", "波兰语", "波兰文":
		return "波兰文"
	case "sv", "swedish", "瑞典语", "瑞典文":
		return "瑞典文"
	default:
		return targetLang
	}
}
