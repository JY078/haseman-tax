package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
)

/** ImageToPromptRequest 图片转提示词请求（使用默认对话/视觉模型） */
type ImageToPromptRequest struct {
	ImageData   string               `json:"image_data"`
	MimeType    string               `json:"mime_type"`
	Images      []ImageToPromptImage `json:"images"`
	Language    string               `json:"language"`
	Instruction string               `json:"instruction"`
}

type ImageToPromptImage struct {
	ImageData string `json:"image_data"`
	MimeType  string `json:"mime_type"`
}

const maxImageToPromptImages = 8

func normalizeImageToPromptImages(req ImageToPromptRequest) ([]adapter.VisionImageItem, error) {
	inputs := make([]ImageToPromptImage, 0, len(req.Images)+1)
	if len(req.Images) > 0 {
		inputs = append(inputs, req.Images...)
	} else if strings.TrimSpace(req.ImageData) != "" {
		inputs = append(inputs, ImageToPromptImage{ImageData: req.ImageData, MimeType: req.MimeType})
	}
	if len(inputs) == 0 {
		return nil, fmt.Errorf("at least one image is required")
	}
	if len(inputs) > maxImageToPromptImages {
		return nil, fmt.Errorf("up to %d images are supported", maxImageToPromptImages)
	}

	images := make([]adapter.VisionImageItem, 0, len(inputs))
	for index, input := range inputs {
		imageData := strings.TrimSpace(input.ImageData)
		if imageData == "" {
			return nil, fmt.Errorf("image %d is empty", index+1)
		}
		mimeType := strings.TrimSpace(input.MimeType)
		if mimeType == "" {
			mimeType = "image/png"
		}
		images = append(images, adapter.VisionImageItem{Data: imageData, MimeType: mimeType})
	}
	return images, nil
}

/** buildImageToPromptInstruction 构建完整指令 */
func buildImageToPromptInstruction() string {
	return "请化身专业的视觉解析师，完整拆解这张图片。按这8个维度逐项深度解析:1.画面主体与环境场景2.画风、美术流派特点3.构图布局、镜头机位4.光影走向、布光方式5.色调搭配、整体氛围感6.物体材质、纹理细节刻画7.画面层级、视觉核心焦点8.渲染参数、画质技术特点梳理图片的共有特征，生成一套能够复刻同款效果的出图提示词,只输出一段完整的提示词。"
}

/** ImageToPrompt 使用默认对话模型分析图片并生成可复用提示词
 * POST /api/image-to-prompt/generate
 */
func ImageToPrompt(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req ImageToPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	images, err := normalizeImageToPromptImages(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey := getSettingValue("chat_key")
	if apiKey == "" {
		apiKey = getSettingValue("draw_key")
	}
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat API Key 未配置"})
		return
	}

	visionModel := getDefaultModelByType("chat", "gpt-4o")

	instruction := strings.TrimSpace(req.Instruction)
	if instruction == "" {
		instruction = buildImageToPromptInstruction()
	}

	usageLog := model.UsageLog{
		UserID:    userID,
		UsageType: "chat",
		Source:    "image_to_prompt",
		Model:     visionModel,
		Prompt:    "[image→prompt]",
		Status:    "failed",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	baseURL := getSettingValue("access_url")
	if strings.TrimSpace(baseURL) == "" {
		baseURL = adapter.ChatCompletionsBaseURL
	}
	chatAdapter := adapter.NewChatAdapter(apiKey, baseURL)
	rawText, usage, rawResp, err := chatAdapter.AnalyzeImagesWithVision(visionModel, images, "", instruction, 2000)
	if err != nil {
		logErr := err
		rawText, usage, rawResp, err = chatAdapter.AnalyzeImagesWithVisionChatCompletions(visionModel, images, instruction)
		if err != nil {
			err = fmt.Errorf("responses failed: %v; chat completions failed: %w", logErr, err)
		}
	}
	// ResponseBody no longer stored
	// if len(rawResp) > 0 {
	// 	usageLog.ResponseBody = adapter.TruncateResponseBody(string(rawResp))
	// }
	if err != nil {
		// usageLog.ResponseBody = fmt.Sprintf(`{"error":"%v"}`, err) // not stored
		database.DB.Create(&usageLog)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("生成提示词失败: %v", err)})
		return
	}

	// ResponseBody no longer stored
	// if len(rawResp) > 0 {
	// 	usageLog.ResponseBody = adapter.TruncateResponseBody(string(rawResp))
	// }

	out := strings.TrimSpace(rawText)
	if out == "" {
		usageLog.Result = "[empty prompt]"
		database.DB.Create(&usageLog)
		c.JSON(http.StatusBadGateway, gin.H{"error": "模型调用成功，但没有返回提示词文本，请检查模型是否支持视觉输入"})
		return
	}
	usageLog.Result = adapter.TruncateRunes(out, 500)
	usageLog.Status = "success"
	inputTokens, outputTokens, totalTokens := adapter.MergeVisionUsage(rawResp, usage)
	usageLog.PromptTokens = inputTokens
	usageLog.CompletionTokens = outputTokens
	usageLog.TotalTokens = totalTokens

	database.DB.Create(&usageLog)

	c.JSON(http.StatusOK, gin.H{
		"prompt":        out,
		"model":         visionModel,
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
		"total_tokens":  totalTokens,
	})
}
