package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"

	"github.com/gin-gonic/gin"
)

/** ProductImageAnalyzeRequest 产品图分析请求，使用配置页设置的主图分析提示词 */
type ProductImageAnalyzeRequest struct {
	ImageData        string `json:"image_data" binding:"required"`
	MimeType         string `json:"mime_type"`
	Prompt           string `json:"prompt" binding:"required"`
	DetailPagePrompt string `json:"detail_page_prompt"` // 可选：传入后用于指导每屏 image_prompt 风格
}

/** productScreenItem 单屏结构化数据 */
type productScreenItem struct {
	Index       int    `json:"index"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImagePrompt string `json:"image_prompt"`
}

/** productAnalysisResult 结构化分析结果（AI 按 JSON 格式输出后解析） */
type productAnalysisResult struct {
	Overview    string              `json:"overview"`
	ScreenCount int                 `json:"screen_count"`
	Screens     []productScreenItem `json:"screens"`
}

/** jsonFormatInstruction 追加到用户提示词末尾，要求 AI 以固定 JSON 格式输出
 * 使用分隔符与用户原始提示词隔离，降低格式混乱的概率
 */
const jsonFormatInstruction = `

---
【输出格式要求】请严格将以上分析结果以 JSON 格式输出，不要输出 JSON 以外的任何文字，格式如下：
{
  "overview": "产品概述（100字以内）",
  "screen_count": <整数，详情页总屏数>,
  "screens": [
    {
      "index": 1,
      "title": "主标题 | 副标题",
      "description": "本屏的核心内容、排版布局、设计要素说明",
      "image_prompt": "本屏 AI 绘图指令：详细描述画面构图、主视觉元素、色彩风格、产品呈现方式、文字区域布局（中文，100-200字）"
    }
  ]
}`

/** buildJsonFormatInstruction 根据是否传入 detail_page_prompt 构建格式指令
 * 若提供，则在 image_prompt 说明中要求 AI 严格沿用该提示词的风格与格式
 */
func buildJsonFormatInstruction(detailPagePrompt string) string {
	imagePromptGuide := "本屏 AI 绘图指令：详细描述画面构图、主视觉元素、色彩风格、产品呈现方式、文字区域布局（中文，100-200字）"
	if detailPagePrompt != "" {
		imagePromptGuide = "本屏 AI 绘图指令：必须严格沿用下方【详情页生成提示词】的风格、语气和格式，在此基础上针对本屏主题进行定制化描述（中文，100-200字）"
	}

	instruction := "\n\n---\n【输出格式要求】请严格将以上分析结果以 JSON 格式输出，不要输出 JSON 以外的任何文字，格式如下：\n"
	instruction += "{\n"
	instruction += "  \"overview\": \"产品概述（100字以内）\",\n"
	instruction += "  \"screen_count\": <整数，详情页总屏数>,\n"
	instruction += "  \"screens\": [\n"
	instruction += "    {\n"
	instruction += "      \"index\": 1,\n"
	instruction += "      \"title\": \"主标题 | 副标题\",\n"
	instruction += "      \"description\": \"本屏的核心内容、排版布局、设计要素说明\",\n"
	instruction += "      \"image_prompt\": \"" + imagePromptGuide + "\"\n"
	instruction += "    }\n"
	instruction += "  ]\n"
	instruction += "}"

	if detailPagePrompt != "" {
		instruction += "\n\n【详情页生成提示词（每屏 image_prompt 必须遵循此风格）】\n" + detailPagePrompt
	}

	return instruction
}

/** AnalyzeProductImageWithPrompt 使用自定义提示词分析产品图片，返回结构化 screens 数组
 * POST /api/product-image/analyze
 * 成功时返回：{ analysis, screen_count, screens }
 * 解析失败降级返回：{ analysis }（原始文本，不含 screens）
 */
func AnalyzeProductImageWithPrompt(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDVal.(uint)

	var req ProductImageAnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	/** 优先使用 chat_key，回退到 draw_key */
	apiKey := getSettingValue("chat_key")
	if apiKey == "" {
		apiKey = getSettingValue("draw_key")
	}
	if apiKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat API Key 未配置"})
		return
	}

	/** 从 ai_models 表获取默认 chat 模型作为 vision 模型 */
	visionModel := getDefaultModelByType("chat", "gpt-4o")

	mimeType := req.MimeType
	if mimeType == "" {
		mimeType = "image/png"
	}

	/** 在用户提示词末尾追加 JSON 格式化指令，驱动结构化输出 */
	structuredPrompt := req.Prompt + buildJsonFormatInstruction(req.DetailPagePrompt)

	// 创建使用记录
	usageLog := model.UsageLog{
		UserID:    userID,
		UsageType: "chat",
		Source:    "product_image",
		Model:     visionModel,
		Prompt:    req.Prompt,
		Status:    "failed",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	chatAdapter := adapter.NewChatAdapter(apiKey, adapter.ChatCompletionsBaseURL)
	rawText, usage, rawResp, err := chatAdapter.AnalyzeImageWithVision(visionModel, req.ImageData, mimeType, structuredPrompt)
	// ResponseBody no longer stored
	// if len(rawResp) > 0 {
	// 	usageLog.ResponseBody = adapter.TruncateResponseBody(string(rawResp))
	// }
	if err != nil {
		// usageLog.ResponseBody = fmt.Sprintf(`{"error":"%v"}`, err) // not stored
		database.DB.Create(&usageLog)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("产品分析失败: %v", err)})
		return
	}

	// ResponseBody no longer stored
	// if len(rawResp) > 0 {
	// 	usageLog.ResponseBody = adapter.TruncateResponseBody(string(rawResp))
	// }

	inputTokens, outputTokens, totalTokens := adapter.MergeVisionUsage(rawResp, usage)
	usageLog.PromptTokens = inputTokens
	usageLog.CompletionTokens = outputTokens
	usageLog.TotalTokens = totalTokens

	if jsonStr := extractJSONObject(rawText); jsonStr != "" {
		var result productAnalysisResult
		if parseErr := json.Unmarshal([]byte(jsonStr), &result); parseErr == nil && len(result.Screens) > 0 {
			usageLog.Result = result.Overview
			usageLog.Status = "success"
			database.DB.Create(&usageLog)
			c.JSON(http.StatusOK, gin.H{
				"analysis":     result.Overview,
				"screen_count": len(result.Screens),
				"screens":      result.Screens,
			})
			return
		}
	}

	/** 解析失败时降级：返回原始文本，不含 screens，前端可降级处理 */
	usageLog.Result = rawText
	usageLog.Status = "success"
	database.DB.Create(&usageLog)
	c.JSON(http.StatusOK, gin.H{
		"analysis": strings.TrimSpace(rawText),
	})
}
