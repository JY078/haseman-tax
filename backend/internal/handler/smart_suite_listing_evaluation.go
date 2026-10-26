package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/smartagent"
	"github.com/gin-gonic/gin"
)

type listingEvaluationRequest struct {
	SessionID            string `json:"session_id"`
	RequestSummary       string `json:"request_summary"`
	Platform             string `json:"platform"`
	PlatformName         string `json:"platform_name"`
	CountryCode          string `json:"country_code"`
	CountryName          string `json:"country_name"`
	Language             string `json:"language"`
	ProductName          string `json:"product_name"`
	ProductCategory      string `json:"product_category"`
	ProductFeatures      string `json:"product_features"`
	ProductProof         string `json:"product_proof"`
	ProductLength        string `json:"product_length"`
	ProductWidth         string `json:"product_width"`
	ProductHeight        string `json:"product_height"`
	ProductDimensionUnit string `json:"product_dimension_unit"`
	ProductWeight        string `json:"product_weight"`
	ProductWeightUnit    string `json:"product_weight_unit"`
	PackageContents      string `json:"package_contents"`
	ImageURL             string `json:"image_url"`
	ExistingListing      string `json:"existing_listing"`
	InternetNumber       string `json:"internet_number"`
}

type listingEvaluationDeduction struct {
	Dimension        string  `json:"dimension"`
	Issue            string  `json:"issue"`
	Points           float64 `json:"points"`
	Severity         string  `json:"severity"`
	Suggestion       string  `json:"suggestion"`
	OptimizationPlan string  `json:"optimization_plan"`
}

type listingEvaluationDimension struct {
	Key           string                       `json:"key"`
	Label         string                       `json:"label"`
	Weight        int                          `json:"weight"`
	Score         float64                      `json:"score"`
	WeightedScore float64                      `json:"weighted_score"`
	Comment       string                       `json:"comment"`
	Deductions    []listingEvaluationDeduction `json:"deductions"`
}

type listingEvaluation struct {
	Summary           string                            `json:"summary"`
	Dimensions        []listingEvaluationDimension      `json:"dimensions"`
	Deductions        []listingEvaluationDeduction      `json:"deductions"`
	Recommendations   []string                          `json:"recommendations"`
	OptimizedListing  model.EcomSuiteListing            `json:"optimized_listing"`
	KeywordCandidates []model.EcomSuiteKeywordCandidate `json:"keyword_candidates,omitempty"`
}

func EvaluateSmartSuiteListing(c *gin.Context) {
	userID, ok := smartSuiteUserID(c)
	if !ok {
		return
	}
	var req listingEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_listing_evaluation_request"})
		return
	}
	platform := normalizeListingPlatform(req.Platform)
	if platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform_and_existing_listing_required"})
		return
	}
	conversation, conversationErr := loadOrCreateSmartSuiteConversation(userID, req.SessionID)
	if conversationErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create_smart_suite_conversation_failed"})
		return
	}
	removeSmartSuiteConversationGreeting(conversation)
	message := strings.TrimSpace(req.RequestSummary)
	if message == "" {
		message = "评估现有 " + strings.TrimSpace(req.PlatformName) + " Listing"
	}
	smartSuiteAppendConversationMessage(conversation, "user", adapter.TruncateRunes(message, 12000), "", "listing_evaluation", "", nil)
	if err := saveSmartSuiteConversation(conversation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create_smart_suite_conversation_failed"})
		return
	}
	// Home Depot fetching remains available in home_depot_listing_fetch.go for
	// future use, but evaluation currently uses the Listing supplied by the UI.
	if strings.TrimSpace(req.ExistingListing) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform_and_existing_listing_required"})
		return
	}
	if strings.TrimSpace(req.ProductName) == "" || strings.TrimSpace(req.ProductFeatures) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "listing_evaluation_source_data_required", "message": "评估需要产品事实"})
		return
	}
	if platform == "lowes" && strings.TrimSpace(req.ImageURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "listing_evaluation_product_image_required", "message": "评估需要上传一张产品图"})
		return
	}
	conversation.Context.ProductName = strings.TrimSpace(req.ProductName)
	conversation.Context.ProductCategory = strings.TrimSpace(req.ProductCategory)
	conversation.Context.Platform = strings.TrimSpace(req.Platform)
	conversation.Context.PlatformName = strings.TrimSpace(req.PlatformName)
	conversation.Context.CountryCode = strings.TrimSpace(req.CountryCode)
	conversation.Context.CountryName = strings.TrimSpace(req.CountryName)
	apiKey := strings.TrimSpace(getSettingValue("chat_key"))
	modelName := getSmartSuiteAgentModel("listing_evaluator")
	if apiKey == "" || modelName == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "listing_evaluator_model_not_configured"})
		return
	}
	keywordRecords := activeListingKeywordRecords(req.Platform, req.CountryCode, req.ProductCategory, 80)
	keywordLibrary := make([]string, 0, len(keywordRecords))
	for _, keyword := range keywordRecords {
		keywordLibrary = append(keywordLibrary, keyword.Term)
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"platform": req.Platform, "platform_name": req.PlatformName, "country_code": req.CountryCode, "country_name": req.CountryName,
		"product_name": req.ProductName, "product_category": req.ProductCategory, "product_features": req.ProductFeatures, "product_proof": req.ProductProof,
		"product_length": req.ProductLength, "product_width": req.ProductWidth, "product_height": req.ProductHeight, "product_dimension_unit": req.ProductDimensionUnit,
		"product_weight": req.ProductWeight, "product_weight_unit": req.ProductWeightUnit, "package_contents": req.PackageContents,
		"keyword_library": keywordLibrary, "existing_listing": req.ExistingListing,
		"category_rule_profile": smartSuiteListingCategoryRuleProfile(req.Platform, req.ProductCategory),
	})
	evaluationInstruction := "Return exactly one JSON object with an evaluation field. Use the eight fixed dimensions and server weights from the skill, including buyer_decision_support and localization."
	if platform == "home_depot" {
		evaluationInstruction = "Return exactly one JSON object with an evaluation field. For Home Depot use only the four fixed dimensions title (22), salient_bullets (24), about_this_product (24), and bullet_points (30), with score and weighted_score expressed as raw points out of each dimension's weight. Do not return the generic eight dimensions."
	} else if platform == "lowes" {
		evaluationInstruction = "Return exactly one JSON object with an evaluation field. For Lowe's use only three dimensions: title (20), description (20), and bullets (60), totaling 100 points; deductions may use 0.5-point increments. Apply the exact Lowe's rubric in the skill only to applicable, confirmed attributes. Kitchen-faucet terms are conditional examples: do not require, generate, or deduct Pull-Down, Sprayer, Single Handle, Deck Plate, GPM, Ceramic Cartridge, spray modes, or any other faucet feature when this SKU does not have it. Dynamically identify equivalent category-relevant product terms, core specifications, functions, scenarios, fit/installation, performance, finish, accessories and warranty. Do not return Specifications, Manuals, Documents, Marketing Copy, or the generic eight dimensions. optimized_listing must contain only title, description, and bullets and must implement every valid correction."
	} else if platform == "wayfair" {
		evaluationInstruction = "Return exactly one JSON object with an evaluation field. For Wayfair use only two dimensions: about_this_product (40) and features (60), totaling 100 points; score and weighted_score are raw points out of each dimension's weight. Do not return the generic eight dimensions or any other key order. Apply only the current Wayfair rules in the skill. Do not enforce character, word-count, sentence-count, feature-count, phrase-only, punctuation, special-character, brand, variant, shipping, warranty, or other removed legacy rules. Do not output or score title. Allow supported colors and finishes. About This Product must connect confirmed facts to concrete tasks and shopper outcomes without internal or platform-facing meta language. Features may be phrases or complete sentences of any count and length, but should connect concrete confirmed features to uses or benefits. optimized_listing must contain only about_this_product and features and implement every valid current-rule correction; do not return other fields."
	}
	evaluationInstruction += " Apply a matched category_rule_profile as category-specific checks without changing the platform's fixed dimensions or weights. First determine which category attributes are applicable; never require or deduct a category-example feature that the product does not have. The supplied product image may support directly visible structure, relative proportions, finish/color, and a conservative use or placement context, but cannot prove hidden mechanisms, exact dimensions, performance, certification, included accessories, or warranty. For Home Depot, treat product parameters and claims already present in the fetched listing as upstream-confirmed for rewriting: do not deduct for upstream source quality and do not invent new claims; deduct only when the listing contradicts confirmed facts. Evaluate field structure, exact lengths/counts, clarity, search relevance, scanability, conversion value, whether category-relevant buyer questions are answered, localization, and risk. All evaluation prose, deductions, suggestions, optimization plans, recommendations, and keyword reasons must be Simplified Chinese; only optimized_listing uses the target marketplace language. Always return a complete, directly usable optimized_listing in the selected platform's exact field structure, even when the score is 80 or above. Every deduction must include a specific Chinese optimization_plan. Include at most 20 fact-supported keyword_candidates inside evaluation for the maintained library's expansion stage."
	system, err := smartSuiteAgentPrompt(smartagent.EvaluateProductListing, evaluationInstruction)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "listing_evaluator_prompt_unavailable"})
		return
	}
	user := "Evaluate this existing listing using the platform rules and quality standard embedded in your skill plus the maintained keyword library and supplied product image. Recalculate findings from visible text, supplied facts, and permitted visible-image evidence only. Input JSON:\n" + string(payload)
	visionImages := buildListingEvaluationVisionImages(req.ImageURL)
	if platform == "lowes" && len(visionImages) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "listing_evaluation_product_image_unavailable", "message": "产品图读取失败，请重新上传"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	startedAt := time.Now()
	raw, usage, _, callErr := adapter.NewChatAdapter(apiKey, getSettingValue("access_url")).AnalyzeImagesWithVisionContext(ctx, modelName, visionImages, system, user, 9000)
	cancel()
	evaluation := listingEvaluation{}
	resultErr := callErr
	if resultErr == nil {
		evaluation, resultErr = parseListingEvaluationResponse(raw)
		if resultErr == nil {
			resultErr = validateListingEvaluationResult(&evaluation, platform)
		}
	}
	recordListingEvaluationUsage(userID, "smart_suite_listing_evaluator", modelName, user, raw, usage, time.Since(startedAt), resultErr)
	if callErr == nil && resultErr != nil {
		for attempt := 0; attempt < 3 && resultErr != nil; attempt++ {
			repairStartedAt := time.Now()
			repairedRaw, repairUsage, repairErr := repairListingEvaluationResponse(c.Request.Context(), modelName, user, raw, resultErr, visionImages)
			if repairErr == nil {
				repairedEvaluation, parseErr := parseListingEvaluationResponse(repairedRaw)
				if parseErr == nil {
					parseErr = validateListingEvaluationResult(&repairedEvaluation, platform)
				}
				repairErr = parseErr
				if parseErr == nil {
					raw = repairedRaw
					usage = repairUsage
					evaluation = repairedEvaluation
					resultErr = nil
				}
			}
			recordListingEvaluationUsage(userID, "smart_suite_listing_evaluator_repair", modelName, user, repairedRaw, repairUsage, time.Since(repairStartedAt), repairErr)
			if repairErr != nil {
				resultErr = fmt.Errorf("evaluation_response_repair_attempt_%d_failed: %w", attempt+1, repairErr)
			}
		}
	}
	if resultErr != nil {
		if callErr != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "listing_evaluation_failed", "message": callErr.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "listing_evaluation_invalid_response", "message": "Listing 评估结果格式异常，自动修复后仍无法读取，请重试"})
		return
	}
	evaluation = normalizeListingEvaluation(evaluation)
	recordListingKeywordCandidatesWithMetadata(req.Platform, req.CountryCode, req.ProductCategory, "listing_evaluator", evaluation.KeywordCandidates)
	markListingKeywordUsage(keywordRecords, req.ExistingListing)
	evaluation.KeywordCandidates = nil
	total := 0.0
	for _, dimension := range evaluation.Dimensions {
		total += dimension.WeightedScore
	}
	response := gin.H{"platform": req.Platform, "total_score": total, "threshold": 80, "needs_optimization": total < 80, "evaluation": evaluation}
	conversation.Context.ListingEvaluation = map[string]interface{}{
		"platform": req.Platform, "total_score": total, "threshold": 80,
		"needs_optimization": total < 80, "evaluation": evaluation,
	}
	resultSummary := fmt.Sprintf("Listing 评估完成：%g / 100，%s。", total, map[bool]string{true: "需要优化", false: "达到通过线"}[total < 80])
	smartSuiteAppendConversationMessage(conversation, "assistant", resultSummary, "", "listing_evaluation", resultSummary, []string{resultSummary})
	if err := saveSmartSuiteConversation(conversation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save_smart_suite_conversation_failed"})
		return
	}
	response["conversation"] = smartSuiteConversationResponse(conversation, "listing_evaluation", resultSummary, 0)
	c.JSON(http.StatusOK, response)
}

func removeSmartSuiteConversationGreeting(conversation *model.SmartSuiteConversation) {
	if conversation == nil {
		return
	}
	filtered := conversation.Messages[:0]
	for _, message := range conversation.Messages {
		if strings.TrimSpace(message.Text) == "你今天想完成什么？先选一个方向，也可以直接描述你的目标。" {
			continue
		}
		filtered = append(filtered, message)
	}
	conversation.Messages = filtered
}

func recordHomeDepotFirecrawlUsage(userID uint, internetNumber string, listing homeDepotListingSource, duration time.Duration, callErr error) {
	status := "success"
	result := "Fetched Home Depot Listing: " + strings.TrimSpace(listing.ProductTitle)
	if callErr != nil {
		status = "failed"
		result = callErr.Error()
	}
	now := time.Now()
	database.DB.Create(&model.UsageLog{
		UserID: userID, UsageType: "chat", Source: "smart_suite_firecrawl", Model: "firecrawl-v2-scrape",
		Prompt: "Home Depot Internet Number: " + internetNumber, Result: adapter.TruncateRunes(result, 1000),
		Duration: int(duration.Milliseconds()), Status: status, CreatedAt: now, UpdatedAt: now,
	})
}

func repairListingEvaluationResponse(ctx context.Context, modelName, originalRequest, raw string, validationErr error, visionImages []adapter.VisionImageItem) (string, *adapter.ResponsesUsage, error) {
	repairSystem, err := smartSuiteAgentPrompt(smartagent.EvaluateProductListing, "Rebuild the evaluation response as one complete valid JSON object. For Home Depot return exactly the four dimensions title (22), salient_bullets (24), about_this_product (24), bullet_points (30); for Lowe's return exactly the three dimensions title (20), description (20), and bullets (60), allow 0.5-point deductions, apply only category-relevant confirmed requirements, and never force faucet example attributes onto a SKU that does not have them; for Wayfair return exactly the two dimensions about_this_product (40) and features (60), allow 0.5-point deductions, and optimized_listing must contain only about_this_product and features; for other platforms return all eight fixed dimensions. Return a complete optimized_listing using the selected platform's exact fields and implement every valid deduction. All evaluation prose and optimization plans must be Simplified Chinese; optimized_listing remains in the target marketplace language. Preserve only evidence-based product claims. Use the supplied image only for visible structure, finish, relative proportion and conservative context, never hidden specifications. Do not return Markdown or commentary.")
	if err != nil {
		return "", nil, err
	}
	repairUser := fmt.Sprintf("The previous response could not be parsed (%s). Recreate the complete evaluation JSON from the original request. Previous response is included only to preserve useful findings.\nOriginal request:\n%s\nPrevious response:\n%s", validationErr, adapter.TruncateRunes(originalRequest, 16000), adapter.TruncateRunes(raw, 14000))
	repairCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	text, usage, _, callErr := adapter.NewChatAdapter(strings.TrimSpace(getSettingValue("chat_key")), getSettingValue("access_url")).AnalyzeImagesWithVisionContext(repairCtx, modelName, visionImages, repairSystem, repairUser, 9000)
	return text, usage, callErr
}

func buildListingEvaluationVisionImages(imageURL string) []adapter.VisionImageItem {
	data, mime, err := fetchSceneReferenceImageData(strings.TrimSpace(imageURL))
	if err != nil || strings.TrimSpace(data) == "" {
		return nil
	}
	return []adapter.VisionImageItem{{Data: data, MimeType: mime}}
}

func recordListingEvaluationUsage(userID uint, source, modelName, prompt, raw string, usage *adapter.ResponsesUsage, duration time.Duration, resultErr error) {
	status := "success"
	result := raw
	if resultErr != nil {
		status = "failed"
		result = resultErr.Error() + ": " + raw
	}
	now := time.Now()
	logItem := model.UsageLog{
		UserID: userID, UsageType: "chat", Source: source, Model: modelName,
		Prompt: adapter.TruncateRunes(prompt, 5000), Result: adapter.TruncateRunes(result, 1600),
		Duration: int(duration.Milliseconds()), Status: status, CreatedAt: now, UpdatedAt: now,
	}
	if usage != nil {
		logItem.PromptTokens, logItem.CompletionTokens, logItem.TotalTokens = usage.InputTokens, usage.OutputTokens, usage.TotalTokens
	}
	database.DB.Create(&logItem)
}

func homeDepotListingFetchFailure(err error) (int, string) {
	message := "Home Depot Listing 获取失败，请稍后重试"
	if err == nil {
		return http.StatusBadGateway, message
	}
	reason := err.Error()
	detail := firecrawlFailureDetail(reason)
	switch {
	case strings.Contains(reason, "firecrawl_api_key_not_configured"):
		return http.StatusServiceUnavailable, "Firecrawl 尚未配置，请到管理后台的系统设置填写 API Key"
	case strings.Contains(reason, "firecrawl_http_401"), strings.Contains(reason, "firecrawl_http_403"):
		return http.StatusBadGateway, appendFirecrawlDetail("Firecrawl API Key 无效或无权限，请检查管理后台系统设置", detail)
	case strings.Contains(reason, "firecrawl_http_400"):
		return http.StatusBadGateway, appendFirecrawlDetail("Firecrawl 拒绝了抓取请求，请检查 API Key 状态后重试", detail)
	case strings.Contains(reason, "firecrawl_http_402"):
		return http.StatusBadGateway, "Firecrawl 额度不足，请检查 Firecrawl 账户"
	case strings.Contains(reason, "firecrawl_http_404"):
		return http.StatusNotFound, "未找到对应的 Home Depot 商品页，请确认 Internet Number"
	case strings.Contains(reason, "firecrawl_http_429"):
		return http.StatusTooManyRequests, "Firecrawl 请求过于频繁，请稍后重试"
	case strings.Contains(reason, "firecrawl_http_5"):
		return http.StatusBadGateway, "Firecrawl 服务暂时不可用，请稍后重试"
	case strings.Contains(reason, "home_depot_listing_content_missing"):
		return http.StatusUnprocessableEntity, "未抓到 Title、Salient Bullets、Bullet Points 或 About This Product，暂不能评估"
	case strings.Contains(reason, "home_depot_listing_product_details_missing"):
		return http.StatusUnprocessableEntity, "仍未抓到 About This Product、Salient Bullets 或 Bullet Points，暂不能评估，请稍后重试"
	case strings.Contains(reason, "home_depot_listing_specifications_missing"):
		return http.StatusUnprocessableEntity, "仍未抓到 About This Product、Salient Bullets 或 Bullet Points，暂不能评估，请稍后重试"
	case strings.Contains(reason, "firecrawl_listing_extract_failed"):
		return http.StatusUnprocessableEntity, "Firecrawl 已访问商品页，但没有提取到可用的 Listing 字段"
	case strings.Contains(reason, "firecrawl_response_invalid_json"), strings.Contains(reason, "firecrawl_listing_invalid_json"):
		return http.StatusBadGateway, "Firecrawl 返回数据格式异常，请稍后重试"
	case strings.Contains(reason, "home_depot_listing_title_missing"):
		return http.StatusUnprocessableEntity, "未抓到 Home Depot 商品标题，暂不能评估"
	case strings.Contains(reason, "home_depot_internet_number_mismatch"):
		return http.StatusUnprocessableEntity, "抓取结果与 Internet Number 不一致，请确认编号后重试"
	case strings.Contains(reason, "firecrawl_request_failed"):
		return http.StatusBadGateway, "Firecrawl 连接失败，请检查网络或稍后重试"
	default:
		return http.StatusBadGateway, message
	}
}

func firecrawlFailureDetail(reason string) string {
	separator := strings.Index(reason, ":")
	if separator < 0 || separator+1 >= len(reason) {
		return ""
	}
	detail := strings.TrimSpace(reason[separator+1:])
	if detail == "" {
		return ""
	}
	detail = strings.ReplaceAll(detail, "\n", " ")
	detail = regexp.MustCompile(`(?i)\bfc-[a-z0-9_-]+`).ReplaceAllString(detail, "fc-***")
	return adapter.TruncateRunes(detail, 240)
}

func appendFirecrawlDetail(message, detail string) string {
	if strings.TrimSpace(detail) == "" {
		return message
	}
	return message + "（Firecrawl 返回：" + detail + "）"
}

func parseListingEvaluationResponse(text string) (listingEvaluation, error) {
	clean := extractJSONObjectFromText(stripSmartSuiteReasoningBlocks(text))
	if clean == "" {
		return listingEvaluation{}, fmt.Errorf("evaluation_invalid_json")
	}
	var envelope struct {
		Evaluation listingEvaluation `json:"evaluation"`
	}
	if err := json.Unmarshal([]byte(clean), &envelope); err != nil {
		return listingEvaluation{}, fmt.Errorf("evaluation_invalid_json: %w", err)
	}
	if len(envelope.Evaluation.Dimensions) == 0 {
		return listingEvaluation{}, fmt.Errorf("evaluation_dimensions_missing")
	}
	return envelope.Evaluation, nil
}

func normalizeListingEvaluation(value listingEvaluation) listingEvaluation {
	// Lowe's、Home Depot 和 Wayfair use an operator rubric of raw points per
	// field; other platforms retain the generic quality dimensions below.
	switch strings.ToLower(strings.TrimSpace(value.OptimizedListing.Platform)) {
	case "lowes":
		return normalizeListingEvaluationRawPoints(value,
			map[string]int{"title": 20, "description": 20, "bullets": 60},
			map[string]string{"title": "Title", "description": "Description", "bullets": "Bullets"},
			[]string{"title", "description", "bullets"},
			"重新评估该字段并返回完整结果")
	case "home_depot":
		return normalizeListingEvaluationRawPoints(value,
			map[string]int{"title": 22, "salient_bullets": 24, "about_this_product": 24, "bullet_points": 30},
			map[string]string{"title": "Title", "salient_bullets": "Salient Bullets", "about_this_product": "About This Product", "bullet_points": "Bullet Points"},
			[]string{"title", "salient_bullets", "about_this_product", "bullet_points"},
			"重新评估该字段并返回完整的四项评分；依据：Home Depot 100 分评分表。")
	case "wayfair":
		return normalizeListingEvaluationRawPoints(value,
			map[string]int{"about_this_product": 40, "features": 60},
			map[string]string{"about_this_product": "About This Product", "features": "Feature Bullets"},
			[]string{"about_this_product", "features"},
			"重新评估该字段并返回完整的两项评分；依据：Wayfair 100 分评分表。")
	}
	weights := map[string]int{"platform_compliance": 15, "product_accuracy": 15, "completeness": 10, "keyword_strategy": 15, "conversion_copy": 15, "buyer_decision_support": 15, "risk_control": 5, "localization": 10}
	labels := map[string]string{"platform_compliance": "平台合规", "product_accuracy": "信息清晰与一致性", "completeness": "字段完整性", "keyword_strategy": "关键词策略", "conversion_copy": "转化表达", "buyer_decision_support": "购买决策支持", "risk_control": "风险控制", "localization": "文案本地化"}
	byKey := make(map[string]listingEvaluationDimension, len(value.Dimensions))
	for _, dimension := range value.Dimensions {
		key := strings.ToLower(strings.TrimSpace(dimension.Key))
		weight, ok := weights[key]
		if !ok {
			continue
		}
		dimension.Key, dimension.Weight = key, weight
		dimension.Label = labels[key]
		if dimension.Score < 0 {
			dimension.Score = 0
		}
		if dimension.Score > 100 {
			dimension.Score = 100
		}
		dimension.WeightedScore = dimension.Score * float64(weight) / 100
		for index := range dimension.Deductions {
			dimension.Deductions[index].Dimension = key
			normalizeListingEvaluationDeduction(&dimension.Deductions[index])
			if dimension.Deductions[index].Points < 0 {
				dimension.Deductions[index].Points = 0
			}
		}
		byKey[key] = dimension
	}
	result := make([]listingEvaluationDimension, 0, len(weights))
	for _, key := range []string{"platform_compliance", "product_accuracy", "completeness", "keyword_strategy", "conversion_copy", "buyer_decision_support", "risk_control", "localization"} {
		if dimension, ok := byKey[key]; ok {
			result = append(result, dimension)
		} else {
			result = append(result, listingEvaluationDimension{
				Key: key, Label: labels[key], Weight: weights[key], Score: 0,
				Comment:    "评估 Agent 未返回该维度，按缺失处理",
				Deductions: []listingEvaluationDeduction{{Dimension: key, Issue: "缺少该维度的评估结果", Points: float64(weights[key]), Severity: "high", Suggestion: "重新评估该维度", OptimizationPlan: "重新提交包含该维度所需的 Listing 文案、目标站点和产品事实；依据：评估技能要求七个维度均必须有结果。"}},
			})
		}
	}
	value.Dimensions = result
	for index := range value.Deductions {
		normalizeListingEvaluationDeduction(&value.Deductions[index])
	}
	if len(value.Deductions) == 0 {
		for _, dimension := range result {
			value.Deductions = append(value.Deductions, dimension.Deductions...)
		}
	}
	if len(value.Recommendations) > 8 {
		value.Recommendations = value.Recommendations[:8]
	}
	return value
}

var listingEvaluationChinesePattern = regexp.MustCompile(`[\p{Han}]`)

func validateListingEvaluationResult(value *listingEvaluation, platform string) error {
	if value == nil {
		return fmt.Errorf("evaluation_missing")
	}
	value.OptimizedListing.Platform = platform
	if platform == "wayfair" {
		value.OptimizedListing = constrainSmartSuiteListingToPlatform(value.OptimizedListing, platform)
	}
	if platform != "wayfair" && strings.TrimSpace(value.OptimizedListing.Title) == "" {
		return fmt.Errorf("evaluation_optimized_listing_title_missing")
	}
	if err := validateSmartSuiteListingPlatformStructure(value.OptimizedListing); err != nil {
		return fmt.Errorf("evaluation_optimized_listing_invalid: %w", err)
	}
	if !listingEvaluationChinesePattern.MatchString(value.Summary) {
		return fmt.Errorf("evaluation_summary_not_chinese")
	}
	requiredDimensions := map[string]bool{
		"platform_compliance": false, "product_accuracy": false, "completeness": false, "keyword_strategy": false,
		"conversion_copy": false, "buyer_decision_support": false, "risk_control": false, "localization": false,
	}
	if platform == "home_depot" {
		requiredDimensions = map[string]bool{"title": false, "salient_bullets": false, "about_this_product": false, "bullet_points": false}
	} else if platform == "lowes" {
		requiredDimensions = map[string]bool{"title": false, "description": false, "bullets": false}
	} else if platform == "wayfair" {
		requiredDimensions = map[string]bool{"about_this_product": false, "features": false}
		if strings.TrimSpace(value.OptimizedListing.AboutThisProduct) == "" || len(value.OptimizedListing.Features) == 0 {
			return fmt.Errorf("evaluation_wayfair_optimized_listing_fields_missing")
		}
	}
	for _, dimension := range value.Dimensions {
		key := strings.ToLower(strings.TrimSpace(dimension.Key))
		if _, exists := requiredDimensions[key]; exists {
			requiredDimensions[key] = true
		}
		if !listingEvaluationChinesePattern.MatchString(dimension.Comment) {
			return fmt.Errorf("evaluation_dimension_comment_not_chinese:%s", dimension.Key)
		}
		for _, deduction := range dimension.Deductions {
			if !listingEvaluationDeductionIsChinese(deduction) {
				return fmt.Errorf("evaluation_deduction_not_chinese:%s", dimension.Key)
			}
		}
	}
	for key, exists := range requiredDimensions {
		if !exists {
			return fmt.Errorf("evaluation_dimension_missing:%s", key)
		}
	}
	for _, deduction := range value.Deductions {
		if !listingEvaluationDeductionIsChinese(deduction) {
			return fmt.Errorf("evaluation_deduction_not_chinese:%s", deduction.Dimension)
		}
	}
	for _, recommendation := range value.Recommendations {
		if !listingEvaluationChinesePattern.MatchString(recommendation) {
			return fmt.Errorf("evaluation_recommendation_not_chinese")
		}
	}
	return nil
}

func listingEvaluationDeductionIsChinese(value listingEvaluationDeduction) bool {
	return listingEvaluationChinesePattern.MatchString(value.Issue) &&
		listingEvaluationChinesePattern.MatchString(value.OptimizationPlan)
}

// normalizeListingEvaluationRawPoints re-scores a platform whose rubric is raw
// points per field: the field score is its weight minus the deductions the
// evaluator reported, and a missing dimension is scored as fully deducted so the
// UI always shows the platform's fixed dimension order.
func normalizeListingEvaluationRawPoints(value listingEvaluation, weights map[string]int, labels map[string]string, order []string, missingPlan string) listingEvaluation {
	byKey := make(map[string]listingEvaluationDimension, len(value.Dimensions))
	for _, dimension := range value.Dimensions {
		key := strings.ToLower(strings.TrimSpace(dimension.Key))
		weight, ok := weights[key]
		if !ok {
			continue
		}
		dimension.Key, dimension.Label, dimension.Weight = key, labels[key], weight
		if dimension.Score < 0 {
			dimension.Score = 0
		}
		if dimension.Score > float64(weight) {
			dimension.Score = float64(weight)
		}
		deducted := 0.0
		for index := range dimension.Deductions {
			dimension.Deductions[index].Dimension = key
			normalizeListingEvaluationDeduction(&dimension.Deductions[index])
			deducted += dimension.Deductions[index].Points
		}
		if deducted > float64(weight) {
			deducted = float64(weight)
		}
		dimension.Score = float64(weight) - deducted
		dimension.WeightedScore = dimension.Score
		byKey[key] = dimension
	}
	result := make([]listingEvaluationDimension, 0, len(order))
	for _, key := range order {
		if dimension, ok := byKey[key]; ok {
			result = append(result, dimension)
			continue
		}
		result = append(result, listingEvaluationDimension{Key: key, Label: labels[key], Weight: weights[key], Score: 0, WeightedScore: 0, Comment: "评估 Agent 未返回该项，按缺失处理", Deductions: []listingEvaluationDeduction{{Dimension: key, Issue: "缺少该项评估结果", Points: float64(weights[key]), Severity: "high", OptimizationPlan: missingPlan}}})
	}
	value.Dimensions = result
	value.Deductions = nil
	for _, dimension := range result {
		value.Deductions = append(value.Deductions, dimension.Deductions...)
	}
	return value
}

func normalizeListingEvaluationDeduction(deduction *listingEvaluationDeduction) {
	if deduction == nil {
		return
	}
	if strings.TrimSpace(deduction.OptimizationPlan) == "" {
		deduction.OptimizationPlan = strings.TrimSpace(deduction.Suggestion)
	}
	if deduction.Points < 0 {
		deduction.Points = 0
	}
}
