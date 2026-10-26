package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/smartagent"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const smartSuiteAutoVisualStyle = "AI 根据产品、品类、平台和站点自主制定高端风格"
const smartSuiteLegacyAutoVisualStyle = "AI 根据产品、品类、平台和站点自主制定高级电商视觉风格"

var smartSuiteNumberWords = map[string]int{
	"一": 1, "二": 2, "两": 2, "三": 3, "四": 4, "五": 5,
	"六": 6, "七": 7, "八": 8, "九": 9, "十": 10,
}

const smartSuiteCoordinatorFailureReply = "需求理解 Agent 暂时没有成功处理这条信息。你的原消息已保留，请稍后直接继续说明，不需要重复发送同一句话。"

const (
	smartSuiteRequestTypeSuite           = "suite"
	smartSuiteRequestTypeListing         = "listing"
	smartSuiteRequestTypeSuiteAndListing = "suite_and_listing"
)

func normalizeSmartSuiteRequestType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case smartSuiteRequestTypeListing, "listing_only", "只写listing", "仅listing":
		return smartSuiteRequestTypeListing
	case smartSuiteRequestTypeSuiteAndListing, "suite_listing", "套图和listing", "套图+listing":
		return smartSuiteRequestTypeSuiteAndListing
	default:
		return smartSuiteRequestTypeSuite
	}
}

func smartSuiteRequestTypeIncludesListing(value string) bool {
	value = normalizeSmartSuiteRequestType(value)
	return value == smartSuiteRequestTypeListing || value == smartSuiteRequestTypeSuiteAndListing
}

func loadSmartSuiteConversation(userID uint, sessionID string) (*model.SmartSuiteConversation, error) {
	var conversation model.SmartSuiteConversation
	query := database.DB.Where("user_id = ?", userID)
	if strings.TrimSpace(sessionID) != "" {
		query = query.Where("session_id = ?", strings.TrimSpace(sessionID))
	} else {
		query = query.Order("updated_at DESC")
	}
	if err := query.First(&conversation).Error; err != nil {
		return nil, err
	}
	return &conversation, nil
}

func loadOrCreateSmartSuiteConversation(userID uint, sessionID string) (*model.SmartSuiteConversation, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return loadSmartSuiteConversation(userID, "")
	}
	conversation, err := loadSmartSuiteConversation(userID, sessionID)
	if err == nil {
		return conversation, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	conversation = newSmartSuiteConversation(userID)
	conversation.SessionID = sessionID
	return conversation, nil
}

type smartSuiteConversationListItem struct {
	SessionID string    `json:"session_id"`
	TaskID    string    `json:"task_id"`
	Title     string    `json:"title"`
	Preview   string    `json:"preview"`
	UpdatedAt time.Time `json:"updated_at"`
}

func smartSuiteShortText(value string, limit int) string {
	text := strings.Join(strings.Fields(value), " ")
	runes := []rune(text)
	if limit > 0 && len(runes) > limit {
		return string(runes[:limit]) + "..."
	}
	return text
}

func smartSuiteConversationListEntry(conversation model.SmartSuiteConversation) smartSuiteConversationListItem {
	title := strings.TrimSpace(conversation.Context.ProductName)
	if title == "" {
		for _, message := range conversation.Messages {
			if message.Role == "user" && strings.TrimSpace(message.Text) != "" {
				title = smartSuiteShortText(message.Text, 24)
				break
			}
		}
	}
	if title == "" {
		title = "新的套图对话"
	}

	preview := "暂无消息"
	for index := len(conversation.Messages) - 1; index >= 0; index-- {
		message := conversation.Messages[index]
		text := strings.TrimSpace(message.Text)
		if text == "" || text == "信息完整，开始生成。" {
			continue
		}
		preview = smartSuiteShortText(text, 38)
		break
	}

	return smartSuiteConversationListItem{
		SessionID: conversation.SessionID,
		TaskID:    conversation.TaskID,
		Title:     title,
		Preview:   preview,
		UpdatedAt: conversation.UpdatedAt,
	}
}

func newSmartSuiteConversation(userID uint) *model.SmartSuiteConversation {
	return &model.SmartSuiteConversation{
		SessionID: "suite-chat-" + uuid.NewString(),
		UserID:    uint64(userID),
		Context:   model.SmartSuiteConversationContext{},
		Messages: []model.SmartSuiteConversationMessage{{
			ID:   uuid.NewString(),
			Role: "assistant",
			Text: "你今天想完成什么？先选一个方向，也可以直接描述你的目标。",
			At:   time.Now().Format(time.RFC3339Nano),
		}},
	}
}

func smartSuiteAppendConversationMessage(conversation *model.SmartSuiteConversation, role, text, imageURL, kind string, publicSummary string, publicActions []string) {
	if strings.TrimSpace(text) == "" && strings.TrimSpace(imageURL) == "" {
		return
	}
	conversation.Messages = append(conversation.Messages, model.SmartSuiteConversationMessage{
		ID: uuid.NewString(), Role: role, Text: strings.TrimSpace(text),
		ImageURL: strings.TrimSpace(imageURL), Kind: strings.TrimSpace(kind),
		PublicSummary: strings.TrimSpace(publicSummary), PublicActions: append([]string(nil), publicActions...),
		At: time.Now().Format(time.RFC3339Nano),
	})
}

func saveSmartSuiteConversation(conversation *model.SmartSuiteConversation) error {
	normalizeSmartSuiteConversationTaskIDs(conversation)
	conversation.UpdatedAt = time.Now()
	if conversation.ID == 0 {
		return database.DB.Create(conversation).Error
	}
	return database.DB.Model(conversation).
		Where("user_id = ?", conversation.UserID).
		Select("TaskID", "TaskIDs", "Context", "Messages", "UpdatedAt").Updates(conversation).Error
}

func normalizeSmartSuiteConversationTaskIDs(conversation *model.SmartSuiteConversation) bool {
	if conversation == nil {
		return false
	}
	before := append([]string(nil), conversation.TaskIDs...)
	result := make(model.StringSlice, 0, len(before)+1)
	seen := make(map[string]struct{}, len(before)+1)
	appendID := func(raw string) {
		id := strings.TrimSpace(raw)
		if id == "" {
			return
		}
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	for _, id := range before {
		appendID(id)
	}
	appendID(conversation.TaskID)
	conversation.TaskIDs = result
	if len(before) != len(result) {
		return true
	}
	for index := range before {
		if before[index] != result[index] {
			return true
		}
	}
	return false
}

func loadSmartSuiteConversationTaskHistory(conversation model.SmartSuiteConversation) []ecomSuiteTaskResponse {
	normalizeSmartSuiteConversationTaskIDs(&conversation)
	if len(conversation.TaskIDs) == 0 || database.DB == nil {
		return []ecomSuiteTaskResponse{}
	}
	var tasks []model.EcomSuiteTask
	if err := database.DB.Where("user_id = ? AND task_id IN ?", conversation.UserID, []string(conversation.TaskIDs)).
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("image_type asc, image_index asc")
		}).Find(&tasks).Error; err != nil {
		log.Printf("[EcomSuite] load conversation task history failed: sessionID=%s err=%v", conversation.SessionID, err)
		return []ecomSuiteTaskResponse{}
	}
	byID := make(map[string]model.EcomSuiteTask, len(tasks))
	for _, task := range tasks {
		byID[task.TaskID] = task
	}
	result := make([]ecomSuiteTaskResponse, 0, len(tasks))
	for _, taskID := range conversation.TaskIDs {
		task, exists := byID[taskID]
		if !exists {
			continue
		}
		item := buildEcomSuiteTaskResponse(task)
		// Conversation history omits the large plan snapshot, but a completed
		// Listing must remain available through the compact top-level field.
		if item.Listing == nil && task.PlanSnapshot != nil {
			item.Listing = task.PlanSnapshot.Listing
		}
		item.PlanSnapshot = nil
		result = append(result, item)
	}
	return result
}

func smartSuiteConversationResponse(conversation *model.SmartSuiteConversation, intent, reply string, targetImageID uint64) gin.H {
	normalizeSmartSuiteConversationContext(&conversation.Context)
	missing := smartSuiteMissingFields(conversation.Context)
	hasActiveImage := conversation.Context.ActiveImageID > 0
	suggestionField := smartSuiteConversationSuggestionField(conversation.Context, intent, hasActiveImage)
	modelOptions, _ := listSmartSuiteDrawingModelOptions()
	return gin.H{
		"session_id":            conversation.SessionID,
		"task_id":               conversation.TaskID,
		"intent":                intent,
		"reply":                 reply,
		"target_image_id":       targetImageID,
		"context":               conversation.Context,
		"messages":              conversation.Messages,
		"task_ids":              conversation.TaskIDs,
		"task_history":          loadSmartSuiteConversationTaskHistory(*conversation),
		"missing_fields":        missing,
		"suggestions":           smartSuiteConversationSuggestions(conversation.Context, intent, hasActiveImage),
		"suggestion_field":      suggestionField,
		"model_options":         modelOptions,
		"ready":                 len(missing) == 0,
		"request_type":          conversation.Context.RequestType,
		"requirement_form_open": conversation.Context.RequirementFormOpen,
	}
}

type smartSuiteDrawingModelOption struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

func smartSuiteModelSupportsType(aiModel model.AiModel, modelType string) bool {
	for _, item := range aiModel.Types {
		if strings.EqualFold(strings.TrimSpace(item), modelType) {
			return true
		}
	}
	return false
}

func listSmartSuiteDrawingModels() ([]model.AiModel, error) {
	if database.DB == nil {
		return nil, fmt.Errorf("database_not_initialized")
	}
	var candidates []model.AiModel
	if err := database.DB.Where("is_active = ?", true).Order("is_default desc, id asc").Find(&candidates).Error; err != nil {
		return nil, err
	}
	result := make([]model.AiModel, 0, len(candidates))
	for _, candidate := range candidates {
		if smartSuiteModelSupportsType(candidate, "drawing") {
			result = append(result, candidate)
		}
	}
	return result, nil
}

func listSmartSuiteDrawingModelOptions() ([]smartSuiteDrawingModelOption, error) {
	models, err := listSmartSuiteDrawingModels()
	if err != nil {
		return nil, err
	}
	options := make([]smartSuiteDrawingModelOption, 0, len(models))
	for _, aiModel := range models {
		options = append(options, smartSuiteDrawingModelOption{
			Name: aiModel.Name, Description: smartSuiteDrawingModelDescription(aiModel.Name), IsDefault: aiModel.IsDefault,
		})
	}
	return options, nil
}

func getActiveSmartSuiteDrawingModel(name string) (model.AiModel, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.AiModel{}, fmt.Errorf("drawing_model_required")
	}
	models, err := listSmartSuiteDrawingModels()
	if err != nil {
		return model.AiModel{}, err
	}
	for _, aiModel := range models {
		if aiModel.Name == name {
			return aiModel, nil
		}
	}
	return model.AiModel{}, fmt.Errorf("drawing_model_unavailable")
}

func smartSuiteDrawingModelDescription(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if slash := strings.LastIndex(normalized, "/"); slash >= 0 {
		normalized = normalized[slash+1:]
	}
	normalized = strings.NewReplacer("_", "-", " ", "-").Replace(normalized)
	switch {
	case ((strings.Contains(normalized, "gemini-3.1") || strings.Contains(normalized, "gemini-3-1")) && strings.Contains(normalized, "flash")) || strings.Contains(normalized, "banana-2"):
		return "速度快"
	case (strings.Contains(normalized, "gemini-3") && strings.Contains(normalized, "pro")) || strings.Contains(normalized, "banana-pro"):
		return "适合高保真"
	case strings.Contains(normalized, "gpt-image-2"):
		return "适合设计"
	case strings.Contains(normalized, "doubao") || strings.Contains(normalized, "seedream") || strings.Contains(normalized, "豆包"):
		return "通用"
	default:
		return "通用生图"
	}
}

// GetSmartSuiteConversation restores only the current user's most recent smart-suite conversation.
func GetSmartSuiteConversation(c *gin.Context) {
	userID, ok := smartSuiteUserID(c)
	if !ok {
		return
	}
	conversation, err := loadSmartSuiteConversation(userID, c.Query("session_id"))
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "load_smart_suite_conversation_failed"})
			return
		}
		conversation = newSmartSuiteConversation(userID)
		if err := saveSmartSuiteConversation(conversation); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create_smart_suite_conversation_failed"})
			return
		}
	}
	conversationChanged := normalizeSmartSuiteConversationTaskIDs(conversation)
	if repairSmartSuiteFailedFocusedAnswer(conversation) {
		conversationChanged = true
	}
	if conversationChanged {
		if err := saveSmartSuiteConversation(conversation); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "repair_smart_suite_conversation_failed"})
			return
		}
	}
	c.JSON(http.StatusOK, smartSuiteConversationResponse(conversation, "collect_requirement", "", 0))
}

func repairSmartSuiteFailedFocusedAnswer(conversation *model.SmartSuiteConversation) bool {
	if conversation == nil || len(conversation.Messages) < 2 {
		return false
	}
	failureIndex := len(conversation.Messages) - 1
	failure := &conversation.Messages[failureIndex]
	userMessage := conversation.Messages[failureIndex-1]
	if failure.Role != "assistant" || strings.TrimSpace(failure.Text) != smartSuiteCoordinatorFailureReply || userMessage.Role != "user" {
		return false
	}
	req := smartSuiteConversationTurnRequest{
		Text: userMessage.Text, ImageURL: userMessage.ImageURL, ImagePurpose: userMessage.Kind,
	}
	req.AnswerField = smartSuiteConversationAnswerField(conversation.Context, req)
	output, ok := smartSuiteSuggestedAnswerFallback(conversation.Context, req)
	if !ok {
		return false
	}
	applySmartSuiteCoordinatorUpdates(&conversation.Context, output.Updates)
	normalizeSmartSuiteConversationContext(&conversation.Context)
	reply := smartSuiteCoordinatorReply(*conversation, output, false)
	failure.Text = reply
	failure.PublicSummary = reply
	failure.PublicActions = []string{reply}
	return true
}

func ListSmartSuiteConversations(c *gin.Context) {
	userID, ok := smartSuiteUserID(c)
	if !ok {
		return
	}

	var conversations []model.SmartSuiteConversation
	if err := database.DB.
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(50).
		Find(&conversations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_smart_suite_conversations_failed"})
		return
	}

	items := make([]smartSuiteConversationListItem, 0, len(conversations))
	for _, conversation := range conversations {
		items = append(items, smartSuiteConversationListEntry(conversation))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func DeleteSmartSuiteConversation(c *gin.Context) {
	userID, ok := smartSuiteUserID(c)
	if !ok {
		return
	}
	sessionID := strings.TrimSpace(c.Param("session_id"))
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_session_required"})
		return
	}

	var conversation model.SmartSuiteConversation
	if err := database.DB.Where("user_id = ? AND session_id = ?", userID, sessionID).First(&conversation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "smart_suite_conversation_not_found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "load_smart_suite_conversation_failed"})
		}
		return
	}
	taskIDs := append([]string{}, conversation.TaskIDs...)
	if strings.TrimSpace(conversation.TaskID) != "" {
		taskIDs = append(taskIDs, conversation.TaskID)
	}
	seen := make(map[string]struct{}, len(taskIDs))
	result := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, taskID := range taskIDs {
			taskID = strings.TrimSpace(taskID)
			if taskID == "" {
				continue
			}
			if _, ok := seen[taskID]; ok {
				continue
			}
			seen[taskID] = struct{}{}
			if err := tx.Model(&model.EcomSuiteTask{}).
				Where("task_id = ? AND user_id = ? AND status IN ?", taskID, userID, []string{"pending", "planning", "generating", "reviewing", "correcting"}).
				Updates(map[string]interface{}{"status": smartSuiteCancelledStatus, "smart_stage": "cancelled", "current_action": "对话已删除，任务已停止", "error_message": "conversation_deleted", "updated_at": time.Now()}).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.EcomSuiteImage{}).Where("task_id = ? AND status IN ?", taskID, []string{"pending", "planning_ready", "generating"}).Updates(map[string]interface{}{"status": smartSuiteCancelledStatus, "error_message": "conversation_deleted", "updated_at": time.Now()}).Error; err != nil {
				return err
			}
		}
		return tx.Where("user_id = ? AND session_id = ?", userID, sessionID).Delete(&model.SmartSuiteConversation{}).Error
	})
	if result != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete_smart_suite_conversation_failed"})
		return
	}
	for taskID := range seen {
		stopSmartSuiteTaskContext(taskID)
	}
	c.JSON(http.StatusOK, gin.H{"session_id": sessionID, "deleted": true})
}

func HandleSmartSuiteConversationTurn(c *gin.Context) {
	userID, ok := smartSuiteUserID(c)
	if !ok {
		return
	}
	var req smartSuiteConversationTurnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_conversation_turn"})
		return
	}
	if strings.TrimSpace(req.Text) == "" && strings.TrimSpace(req.ImageURL) == "" && req.InitialRequirements == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "conversation_content_required"})
		return
	}
	conversation, err := loadSmartSuiteConversation(userID, req.SessionID)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "load_smart_suite_conversation_failed"})
			return
		}
		conversation = newSmartSuiteConversation(userID)
	}
	normalizeSmartSuiteConversationTaskIDs(conversation)
	if requestedType := strings.TrimSpace(req.RequestType); requestedType != "" {
		conversation.Context.RequestType = normalizeSmartSuiteRequestType(requestedType)
		conversation.Context.RequirementFormOpen = true
	}
	if requestedModel := strings.TrimSpace(req.DrawingModel); requestedModel != "" {
		activeModel, modelErr := getActiveSmartSuiteDrawingModel(requestedModel)
		if modelErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "drawing_model_unavailable", "message": "所选生图模型当前不可用，请重新选择"})
			return
		}
		conversation.Context.DrawingModel = activeModel.Name
	}
	if imageURL := strings.TrimSpace(req.ImageURL); imageURL != "" {
		req.ImagePurpose = strings.ToLower(strings.TrimSpace(req.ImagePurpose))
		if req.ImagePurpose == "" {
			if strings.TrimSpace(conversation.Context.WhiteBgImageURL) == "" {
				req.ImagePurpose = "product_image"
			} else {
				req.ImagePurpose = "style_reference"
			}
		} else if req.ImagePurpose != "product_image" && req.ImagePurpose != "style_reference" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_image_purpose"})
			return
		}
		if strings.EqualFold(strings.TrimSpace(req.ImagePurpose), "style_reference") {
			conversation.Context.StyleReferenceURLs = appendUniqueSmartSuiteURLs(conversation.Context.StyleReferenceURLs, imageURL)
		} else {
			conversation.Context.WhiteBgImageURL = imageURL
		}
	}
	if len(req.ProductReferenceURLs) > 0 {
		conversation.Context.ProductReferenceURLs = normalizeSmartSuiteProductReferenceURLs(conversation.Context.WhiteBgImageURL, req.ProductReferenceURLs)
	}
	if req.InitialRequirements != nil {
		if errorCode, message := applySmartSuiteInitialRequirements(&conversation.Context, *req.InitialRequirements); errorCode != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errorCode, "message": message})
			return
		}
		normalizeSmartSuiteConversationContext(&conversation.Context)
		userSummary := strings.TrimSpace(req.Text)
		if userSummary == "" {
			userSummary = smartSuiteInitialRequirementsSummary(conversation.Context)
		}
		smartSuiteAppendConversationMessage(conversation, "user", userSummary, req.ImageURL, "product_image", "", nil)
		reply := smartSuiteConfirmationReply(conversation.Context)
		smartSuiteAppendConversationMessage(conversation, "assistant", reply, "", "", reply, []string{"已保存首次表单中的完整需求，未调用需求理解 Agent"})
		if err := saveSmartSuiteConversation(conversation); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "save_smart_suite_conversation_failed"})
			return
		}
		c.JSON(http.StatusOK, smartSuiteConversationResponse(conversation, "collect_requirement", reply, 0))
		return
	}
	if req.TargetImageID > 0 {
		conversation.Context.ActiveImageID = req.TargetImageID
	}
	// A marketplace/site selection also establishes the default copy language;
	// keep the coordinator from asking a redundant language question.
	normalizeSmartSuiteConversationContext(&conversation.Context)
	req.AnswerField = smartSuiteConversationAnswerField(conversation.Context, req)
	kind := strings.TrimSpace(req.ImagePurpose)
	smartSuiteAppendConversationMessage(conversation, "user", req.Text, req.ImageURL, kind, "", nil)

	output := smartSuiteLocalCoordinator(conversation, req)
	if output.TargetImageID == 0 {
		output.TargetImageID = resolveSmartSuiteImageReference(*conversation, req.Text)
		if output.TargetImageID > 0 && strings.TrimSpace(req.Text) != "" {
			if req.HasCandidate || smartSuiteImageHasReadyCandidate(*conversation, output.TargetImageID) {
				output.Intent = smartSuiteCandidateIntent(req.Text)
			}
			if output.Intent == "collect_requirement" {
				output.Intent = "edit_image"
			}
		}
	}
	verifiedOutputCounts, hasVerifiedOutputCounts := smartSuiteOutputCountUpdatesFromText(req.Text)
	acceptVerifiedOutputCounts := hasVerifiedOutputCounts && output.Intent != "edit_image" && output.Intent != "accept_candidate" && output.Intent != "reject_candidate" && output.Intent != "stop"
	wasCollectingOutputCounts := containsSmartSuiteField(smartSuiteMissingFields(conversation.Context), "output_counts")
	if output.Intent == "collect_requirement" && strings.TrimSpace(req.DrawingModel) == "" {
		attempts := 2
		if _, hasFocusedFallback := smartSuiteSuggestedAnswerFallback(conversation.Context, req); hasFocusedFallback {
			attempts = 1
		}
		if aiOutput, coordinatorErr := runValidatedSmartSuiteConversationCoordinator(*conversation, req, attempts); coordinatorErr == nil {
			localOutput := output
			if suggestedOutput, ok := smartSuiteSuggestedAnswerFallback(conversation.Context, req); ok && !smartSuiteCoordinatorOutputAnswersField(aiOutput, req.AnswerField) {
				// Keep any valid semantic updates returned by the agent, then fill the
				// focused field from the user's exact text when the agent omitted it.
				output = mergeSmartSuiteCoordinatorOutput(localOutput, aiOutput)
				output = mergeSmartSuiteCoordinatorFocusedFallback(output, suggestedOutput)
				log.Printf("[EcomSuite] smart suite coordinator omitted focused answer field; preserved agent output and accepted text fallback: userID=%d sessionID=%s field=%s", userID, conversation.SessionID, req.AnswerField)
			} else if nonEmptySmartSuiteString(aiOutput.Updates.DrawingModel) {
				activeModel, modelErr := getActiveSmartSuiteDrawingModel(*aiOutput.Updates.DrawingModel)
				if modelErr != nil {
					output.NeedsClarification = true
					output.ClarificationQuestion = "没有找到你指定的可用生图模型，请从下方当前可用的模型中选择。"
				} else {
					aiOutput.Updates.DrawingModel = stringPtr(activeModel.Name)
					output = mergeSmartSuiteCoordinatorOutput(localOutput, aiOutput)
				}
			} else {
				output = mergeSmartSuiteCoordinatorOutput(localOutput, aiOutput)
			}
		} else if suggestedOutput, ok := smartSuiteSuggestedAnswerFallback(conversation.Context, req); ok {
			output = mergeSmartSuiteCoordinatorOutput(output, suggestedOutput)
			log.Printf("[EcomSuite] smart suite conversation coordinator failed; accepted focused answer fallback: userID=%d sessionID=%s field=%s err=%v", userID, conversation.SessionID, req.AnswerField, coordinatorErr)
		} else if revisionOutput, ok := smartSuiteRequirementRevisionFallback(req.Text); ok {
			output = mergeSmartSuiteCoordinatorOutput(output, revisionOutput)
			log.Printf("[EcomSuite] smart suite conversation coordinator failed; accepted requirement revision clarification fallback: userID=%d sessionID=%s err=%v", userID, conversation.SessionID, coordinatorErr)
		} else if acceptVerifiedOutputCounts {
			log.Printf("[EcomSuite] smart suite conversation coordinator failed; accepted verified output counts: userID=%d sessionID=%s err=%v", userID, conversation.SessionID, coordinatorErr)
		} else {
			log.Printf("[EcomSuite] smart suite conversation coordinator failed: userID=%d sessionID=%s err=%v", userID, conversation.SessionID, coordinatorErr)
			output.NeedsClarification = true
			output.ClarificationQuestion = smartSuiteCoordinatorFailureReply
		}
	}
	acceptVerifiedOutputCounts = hasVerifiedOutputCounts && output.Intent != "edit_image" && output.Intent != "accept_candidate" && output.Intent != "reject_candidate" && output.Intent != "stop"
	if acceptVerifiedOutputCounts {
		mergeSmartSuiteCoordinatorUpdates(&output.Updates, verifiedOutputCounts)
		if wasCollectingOutputCounts {
			output.NeedsClarification = false
			output.ClarificationQuestion = ""
		}
	}
	if output.Intent == "edit_image" {
		// A targeted image instruction is an execution command, not a suite
		// requirement revision. Keep the full user text for the image-edit API and
		// prevent any coordinator output from mutating the stored suite context.
		output.Updates = smartSuiteCoordinatorUpdates{}
		output.ClearFields = nil
		output.RequestType = ""
		output.OpenRequirementForm = false
		output.NeedsClarification = false
		output.ClarificationQuestion = ""
	}
	applySmartSuiteCoordinatorClears(&conversation.Context, output.ClearFields)
	applySmartSuiteCoordinatorUpdates(&conversation.Context, output.Updates)
	if output.RequestType != "" {
		conversation.Context.RequestType = normalizeSmartSuiteRequestType(output.RequestType)
		conversation.Context.RequirementFormOpen = output.OpenRequirementForm
	}
	normalizeSmartSuiteConversationContext(&conversation.Context)
	if output.TargetImageID == 0 {
		output.TargetImageID = conversation.Context.ActiveImageID
	}
	if output.TargetImageID > 0 {
		conversation.Context.ActiveImageID = output.TargetImageID
	}

	reply := smartSuiteCoordinatorReply(*conversation, output, req.HasCandidate)
	if strings.TrimSpace(output.PublicSummary) == "" || len(output.PublicActions) == 0 {
		// Local intents and verified suggestion fallbacks have no model report;
		// expose the actual reply instead of fabricating a generic thinking log.
		output.PublicSummary = strings.TrimSpace(reply)
		output.PublicActions = []string{strings.TrimSpace(reply)}
	}
	smartSuiteAppendConversationMessage(conversation, "assistant", reply, "", "", output.PublicSummary, output.PublicActions)
	if output.Intent == "accept_candidate" || output.Intent == "reject_candidate" {
		conversation.Context.ActiveImageID = 0
	}
	if err := saveSmartSuiteConversation(conversation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save_smart_suite_conversation_failed"})
		return
	}
	c.JSON(http.StatusOK, smartSuiteConversationResponse(conversation, output.Intent, reply, output.TargetImageID))
}

func applySmartSuiteInitialRequirements(ctx *model.SmartSuiteConversationContext, initial smartSuiteInitialRequirements) (string, string) {
	initial.ProductName = strings.TrimSpace(initial.ProductName)
	initial.RequestType = normalizeSmartSuiteRequestType(initial.RequestType)
	initial.ProductFeatures = strings.TrimSpace(initial.ProductFeatures)
	initial.Platform = normalizeListingPlatform(initial.Platform)
	initial.PlatformName = strings.TrimSpace(initial.PlatformName)
	initial.CountryCode = strings.ToUpper(strings.TrimSpace(initial.CountryCode))
	initial.CountryName = strings.TrimSpace(initial.CountryName)
	ctx.ProductReferenceURLs = normalizeSmartSuiteProductReferenceURLs(ctx.WhiteBgImageURL, initial.ProductReferenceURLs)
	listingOnly := initial.RequestType == smartSuiteRequestTypeListing
	if strings.TrimSpace(ctx.WhiteBgImageURL) == "" {
		return "initial_requirements_product_image_required", "请先上传产品图"
	}
	if initial.ProductName == "" || initial.ProductFeatures == "" {
		return "initial_requirements_product_info_required", "请填写产品名称和产品信息"
	}
	if initial.Platform == "" || initial.CountryCode == "" {
		return "initial_requirements_platform_required", "请选择平台站点，或填写自定义平台和国家"
	}
	if initial.SecondaryCount < 0 || initial.SecondaryCount > 9 || initial.DesktopAPlusCount < 0 || initial.DesktopAPlusCount > 8 || initial.MobileAPlusCount < 0 || initial.MobileAPlusCount > 8 {
		return "initial_requirements_invalid_counts", "副图最多 9 张，桌面 A+ 和移动 A+ 各最多 8 张"
	}
	if !listingOnly && initial.SecondaryCount+initial.DesktopAPlusCount+initial.MobileAPlusCount == 0 {
		return "initial_requirements_output_required", "请至少选择一种套图并设置生成数量"
	}
	if !listingOnly && strings.TrimSpace(ctx.DrawingModel) == "" {
		return "initial_requirements_drawing_model_required", "请选择生图模型"
	}

	ctx.ProductName = initial.ProductName
	ctx.ProductCategory = strings.TrimSpace(initial.ProductCategory)
	ctx.ProductBrand = strings.TrimSpace(initial.ProductBrand)
	ctx.ProductFeatures = initial.ProductFeatures
	ctx.ProductProof = strings.TrimSpace(initial.ProductProof)
	ctx.ProductLength = strings.TrimSpace(initial.ProductLength)
	ctx.ProductWidth = strings.TrimSpace(initial.ProductWidth)
	ctx.ProductHeight = strings.TrimSpace(initial.ProductHeight)
	ctx.ProductDimensionUnit = strings.TrimSpace(initial.ProductDimensionUnit)
	ctx.ProductWeight = strings.TrimSpace(initial.ProductWeight)
	ctx.ProductWeightUnit = strings.TrimSpace(initial.ProductWeightUnit)
	ctx.PackageLength = strings.TrimSpace(initial.PackageLength)
	ctx.PackageWidth = strings.TrimSpace(initial.PackageWidth)
	ctx.PackageHeight = strings.TrimSpace(initial.PackageHeight)
	ctx.PackageDimensionUnit = strings.TrimSpace(initial.PackageDimensionUnit)
	ctx.PackageWeight = strings.TrimSpace(initial.PackageWeight)
	ctx.PackageWeightUnit = strings.TrimSpace(initial.PackageWeightUnit)
	ctx.PackageContents = strings.TrimSpace(initial.PackageContents)
	ctx.Platform = initial.Platform
	ctx.PlatformName = initial.PlatformName
	ctx.CountryCode = initial.CountryCode
	ctx.CountryName = initial.CountryName
	ctx.ListingPlatformRules = ""
	ctx.ListingQualityRules = ""
	ctx.ListingKeywords = listingKeywordLibraryWithInput(initial.Platform, initial.CountryCode, initial.ProductCategory, initial.ListingKeywords)
	ctx.RequestType = initial.RequestType
	ctx.IncludeListing = smartSuiteRequestTypeIncludesListing(initial.RequestType)
	ctx.RequirementFormOpen = false
	if listingOnly {
		ctx.SecondaryCount, ctx.DesktopAPlusCount, ctx.MobileAPlusCount = 0, 0, 0
	} else {
		ctx.SecondaryCount = initial.SecondaryCount
		ctx.DesktopAPlusCount = initial.DesktopAPlusCount
		ctx.MobileAPlusCount = initial.MobileAPlusCount
	}
	if listingOnly {
		// Listing-only tasks still retain the uploaded product images so the
		// writer can distinguish visually similar SKUs. Only image-suite
		// controls are cleared.
		ctx.VisualStyle = ""
		ctx.StyleRequirements = ""
		ctx.StyleReferenceURLs = nil
		ctx.DrawingModel = ""
	} else {
		ctx.VisualStyle = strings.TrimSpace(initial.VisualStyle)
		if ctx.VisualStyle == "" {
			ctx.VisualStyle = smartSuiteAutoVisualStyle
		}
		ctx.StyleRequirements = strings.TrimSpace(initial.StyleRequirements)
	}
	normalizeSmartSuiteConversationContext(ctx)
	return "", ""
}

func smartSuiteInitialRequirementsSummary(ctx model.SmartSuiteConversationContext) string {
	platform := strings.TrimSpace(strings.TrimSpace(ctx.PlatformName) + " " + strings.TrimSpace(ctx.CountryName))
	outputs := make([]string, 0, 3)
	if ctx.SecondaryCount > 0 {
		outputs = append(outputs, fmt.Sprintf("副图 %d 张", ctx.SecondaryCount))
	}
	if ctx.DesktopAPlusCount > 0 {
		outputs = append(outputs, fmt.Sprintf("桌面 A+ %d 张", ctx.DesktopAPlusCount))
	}
	if ctx.MobileAPlusCount > 0 {
		outputs = append(outputs, fmt.Sprintf("移动 A+ %d 张", ctx.MobileAPlusCount))
	}
	outputLabel := strings.Join(outputs, "、")
	if ctx.RequestType == smartSuiteRequestTypeListing {
		outputLabel = "仅 Listing 文案"
	} else if ctx.IncludeListing {
		outputLabel += "、Listing 文案"
	}
	listingSource := ""
	if ctx.IncludeListing {
		listingSource = "\nListing 依据：已自动加载平台规则与关键词库"
	}
	return fmt.Sprintf("已填写需求表单\n产品：%s\n平台站点：%s\n产出：%s\n产品信息：%s%s", ctx.ProductName, platform, outputLabel, smartSuiteFeatureSummary(ctx.ProductFeatures), listingSource)
}

func smartSuiteImageHasReadyCandidate(conversation model.SmartSuiteConversation, imageID uint64) bool {
	if strings.TrimSpace(conversation.TaskID) == "" || imageID == 0 {
		return false
	}
	var count int64
	database.DB.Model(&model.EcomSuiteImage{}).
		Where("task_id = ? AND id = ? AND candidate_status = ?", conversation.TaskID, imageID, "ready").
		Count(&count)
	return count > 0
}

func smartSuiteLocalCoordinator(conversation *model.SmartSuiteConversation, req smartSuiteConversationTurnRequest) smartSuiteCoordinatorOutput {
	text := strings.TrimSpace(req.Text)
	lower := strings.ToLower(text)
	output := smartSuiteCoordinatorOutput{Intent: "collect_requirement", TargetImageID: req.TargetImageID}
	if requestedType := strings.TrimSpace(req.RequestType); requestedType != "" {
		requestType := normalizeSmartSuiteRequestType(requestedType)
		output.RequestType = requestType
		output.OpenRequirementForm = true
		output.RequirementKind = "other"
		output.Confidence = 1
		output.OtherEvidence = []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}}
		return output
	}
	if conversation.TaskID == "" && strings.TrimSpace(conversation.Context.ProductName) == "" {
		if requestType := smartSuiteRequestTypeFromText(text); requestType != "" {
			output.RequestType = requestType
			output.OpenRequirementForm = true
			output.RequirementKind = "other"
			output.Confidence = 1
			output.OtherEvidence = []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}}
			return output
		}
	}
	switch {
	case regexp.MustCompile(`^(停止|停止生成|取消|取消生成|不用生成了|停下来)[。！!\s]*$`).MatchString(text):
		output.Intent = "stop"
		return output
	case req.HasCandidate && regexp.MustCompile(`^(采用|使用|确认|就这个|用这个版本|采用这个版本)[。！!\s]*$`).MatchString(text):
		output.Intent = "accept_candidate"
		return output
	case req.HasCandidate && regexp.MustCompile(`^(保留原图|不要这个|拒绝|取消修改|还是原图)[。！!\s]*$`).MatchString(text):
		output.Intent = "reject_candidate"
		return output
	case req.TargetImageID > 0 && text != "":
		output.Intent = "edit_image"
		return output
	case smartSuiteExplicitStartText(text):
		output.Intent = "start"
		if !smartSuiteHasVisualStyle(conversation.Context) {
			output.Updates.VisualStyle = stringPtr(smartSuiteAutoVisualStyle)
		}
		return output
	}

	if strings.Contains(lower, "accept") && req.HasCandidate {
		output.Intent = "accept_candidate"
	}
	return output
}

func smartSuiteRequestTypeFromText(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return ""
	}
	hasListing := strings.Contains(lower, "listing") || strings.Contains(lower, "商品文案") || strings.Contains(lower, "产品文案")
	hasSuite := strings.Contains(lower, "套图") || strings.Contains(lower, "副图") || strings.Contains(lower, "a+")
	switch {
	case hasListing && hasSuite:
		return smartSuiteRequestTypeSuiteAndListing
	case hasListing:
		return smartSuiteRequestTypeListing
	case hasSuite:
		return smartSuiteRequestTypeSuite
	default:
		return ""
	}
}

func smartSuiteSuggestedAnswerFallback(ctx model.SmartSuiteConversationContext, req smartSuiteConversationTurnRequest) (smartSuiteCoordinatorOutput, bool) {
	field := strings.TrimSpace(req.AnswerField)
	text := strings.TrimSpace(req.Text)
	if field == "" || text == "" || (field != "visual_style" && !smartSuiteAnswerFieldIsMissing(ctx, field)) {
		return smartSuiteCoordinatorOutput{}, false
	}
	verifiedSuggestion := false
	for _, suggestion := range smartSuiteConversationSuggestions(ctx, "collect_requirement", false) {
		if text == suggestion {
			verifiedSuggestion = true
			break
		}
	}

	output := smartSuiteCoordinatorOutput{Intent: "collect_requirement", Confidence: 1}
	switch field {
	case "visual_style":
		if !verifiedSuggestion && !smartSuiteFreeTextVisualStyle(text) && !smartSuiteRequestsAutoVisualStyle(text) {
			return smartSuiteCoordinatorOutput{}, false
		}
		output.RequirementKind = "visual_direction"
		output.VisualEvidence = []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}}
		if smartSuiteRequestsAutoVisualStyle(text) || text == "交给 AI 自主设计" {
			output.Updates.VisualStyle = stringPtr(smartSuiteAutoVisualStyle)
		} else {
			output.Updates.VisualStyle = stringPtr(text)
		}
		if strings.TrimSpace(ctx.VisualStyle) != "" && strings.TrimSpace(ctx.VisualStyle) != strings.TrimSpace(*output.Updates.VisualStyle) {
			output.ClearFields = append(output.ClearFields, "visual_style")
		}
	case "product_info":
		if verifiedSuggestion || !smartSuiteFreeTextProductInfo(text) {
			return smartSuiteCoordinatorOutput{}, false
		}
		output.RequirementKind = "product_facts"
		output.ProductEvidence = []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}}
		if strings.TrimSpace(ctx.ProductName) == "" {
			if name := smartSuiteProductNameFromText(text); name != "" {
				output.Updates.ProductName = stringPtr(name)
			}
		}
		output.Updates.ProductFeatures = stringPtr(smartSuiteMergedProductFeatures(ctx.ProductFeatures, text))
	case "platform_site":
		if !verifiedSuggestion {
			return smartSuiteCoordinatorOutput{}, false
		}
		output.RequirementKind = "other"
		output.OtherEvidence = []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}}
		switch text {
		case "Amazon 美国站":
			output.Updates.Platform = stringPtr("amazon")
			output.Updates.PlatformName = stringPtr("Amazon")
			output.Updates.CountryCode = stringPtr("US")
			output.Updates.CountryName = stringPtr("美国站")
		case "Amazon 欧洲站":
			output.Updates.Platform = stringPtr("amazon")
			output.Updates.PlatformName = stringPtr("Amazon")
			output.Updates.CountryCode = stringPtr("EU")
			output.Updates.CountryName = stringPtr("欧洲站")
		case "Walmart 美国站":
			output.Updates.Platform = stringPtr("walmart")
			output.Updates.PlatformName = stringPtr("Walmart")
			output.Updates.CountryCode = stringPtr("US")
			output.Updates.CountryName = stringPtr("美国站")
		case "Home Depot 美国站":
			output.Updates.Platform = stringPtr("home_depot")
			output.Updates.PlatformName = stringPtr("Home Depot")
			output.Updates.CountryCode = stringPtr("US")
			output.Updates.CountryName = stringPtr("美国站")
		case "Wayfair 美国站":
			output.Updates.Platform = stringPtr("wayfair")
			output.Updates.PlatformName = stringPtr("Wayfair")
			output.Updates.CountryCode = stringPtr("US")
			output.Updates.CountryName = stringPtr("美国站")
		case "Lowe's 美国站":
			output.Updates.Platform = stringPtr("lowes")
			output.Updates.PlatformName = stringPtr("Lowe's")
			output.Updates.CountryCode = stringPtr("US")
			output.Updates.CountryName = stringPtr("美国站")
		case "Temu 美国站":
			output.Updates.Platform = stringPtr("temu")
			output.Updates.PlatformName = stringPtr("Temu")
			output.Updates.CountryCode = stringPtr("US")
			output.Updates.CountryName = stringPtr("美国站")
		default:
			return smartSuiteCoordinatorOutput{}, false
		}
	case "output_counts":
		if !verifiedSuggestion {
			return smartSuiteCoordinatorOutput{}, false
		}
		output.RequirementKind = "other"
		output.OtherEvidence = []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}}
		zero, six, seven := 0, 6, 7
		switch text {
		case "6 张副图":
			output.Updates.SecondaryCount = &six
			output.Updates.DesktopAPlusCount = &zero
			output.Updates.MobileAPlusCount = &zero
		case "7 张桌面 A+":
			output.Updates.SecondaryCount = &zero
			output.Updates.DesktopAPlusCount = &seven
			output.Updates.MobileAPlusCount = &zero
		case "7 张移动 A+":
			output.Updates.SecondaryCount = &zero
			output.Updates.DesktopAPlusCount = &zero
			output.Updates.MobileAPlusCount = &seven
		case "副图、桌面 A+、移动 A+ 都要":
			output.Updates.SecondaryCount = &six
			output.Updates.DesktopAPlusCount = &seven
			output.Updates.MobileAPlusCount = &seven
		default:
			return smartSuiteCoordinatorOutput{}, false
		}
	default:
		return smartSuiteCoordinatorOutput{}, false
	}
	if err := validateSmartSuiteCoordinatorOutput(ctx, req, &output); err != nil {
		return smartSuiteCoordinatorOutput{}, false
	}
	return output, true
}

func smartSuiteRequirementRevisionFallback(text string) (smartSuiteCoordinatorOutput, bool) {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.NewReplacer(" ", "", "。", "", "！", "", "!", "", "，", "", ",", "").Replace(normalized)
	if normalized == "" {
		return smartSuiteCoordinatorOutput{}, false
	}
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`^(?:我)?(?:还)?(?:想|要|需要)?(?:修改|调整|改)(?:一下)?(?:需求|要求|方案)$`),
		regexp.MustCompile(`^(?:需求|要求|方案)(?:还)?(?:要|需要)?(?:修改|调整|改)(?:一下)?$`),
		regexp.MustCompile(`^(?:i)?(?:want|need)to(?:change|adjust|revise)(?:the)?requirements?$`),
	}
	matched := false
	for _, pattern := range patterns {
		if pattern.MatchString(normalized) {
			matched = true
			break
		}
	}
	if !matched {
		return smartSuiteCoordinatorOutput{}, false
	}
	question := "可以继续修改。请直接告诉我具体要改哪一项和改成什么，例如产品事实、平台站点、图片数量、文案语言、视觉场景或生图模型。"
	return smartSuiteCoordinatorOutput{
		Intent:                "collect_requirement",
		RequirementKind:       "ambiguous",
		Confidence:            1,
		NeedsClarification:    true,
		ClarificationQuestion: question,
		PublicSummary:         "已进入需求调整状态，等待具体修改内容。",
		PublicActions:         []string{"保留当前需求、聊天记录和已有生图，等待你说明具体改动。"},
	}, true
}

func smartSuiteFreeTextVisualStyle(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" || len([]rune(text)) < 2 || smartSuiteLooksLikeConversationControl(text) || smartSuiteUnsupportedFinalStyle(text) || smartSuiteLooksLikeOtherRequirement(text) || smartSuiteLooksLikeProductFacts(text) {
		return false
	}
	return true
}

func smartSuiteRequestsAutoVisualStyle(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.NewReplacer(" ", "", "。", "", "！", "", "!", "", ".", "").Replace(normalized)
	switch normalized {
	case "没有要求", "没有特别要求", "没有特殊要求", "没要求", "无要求", "无特别要求", "无特殊要求", "都可以", "你决定", "ai决定", "ai来定", "交给ai", "交给ai自主设计", "自动设计", "直接生成", "nopreference", "youdecide", "automaticstyle", "autostyle":
		return true
	default:
		return false
	}
}

func smartSuiteConversationAnswerField(ctx model.SmartSuiteConversationContext, req smartSuiteConversationTurnRequest) string {
	field := strings.TrimSpace(req.AnswerField)
	if (field == "" || field == "white_bg_image") && req.ImagePurpose == "product_image" && smartSuiteLooksLikeProductFacts(req.Text) {
		return "product_info"
	}
	if field != "" && field != "white_bg_image" {
		return field
	}
	if smartSuiteLooksLikeExplicitVisualDirection(req.Text) {
		return "visual_style"
	}
	missing := smartSuiteMissingFields(ctx)
	switch {
	case containsSmartSuiteField(missing, "product_name") || containsSmartSuiteField(missing, "product_features"):
		return "product_info"
	case containsSmartSuiteField(missing, "platform_site"):
		return "platform_site"
	case containsSmartSuiteField(missing, "output_counts"):
		return "output_counts"
	case containsSmartSuiteField(missing, "drawing_model"):
		return "drawing_model"
	}
	return field
}

func smartSuiteFreeTextProductInfo(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" || len([]rune(text)) < 2 {
		return false
	}
	// A focused product-info turn may include several paragraphs. Explicit
	// scene instructions belong to the visual-direction collector instead.
	return !smartSuiteLooksLikeConversationControl(text) && !smartSuiteLooksLikeExplicitVisualDirection(text) && !smartSuiteLooksLikeOtherRequirement(text)
}

func smartSuiteAnswerFieldIsMissing(ctx model.SmartSuiteConversationContext, field string) bool {
	if field == "product_info" {
		return containsSmartSuiteField(smartSuiteMissingFields(ctx), "product_name") || containsSmartSuiteField(smartSuiteMissingFields(ctx), "product_features")
	}
	return containsSmartSuiteField(smartSuiteMissingFields(ctx), field)
}

func smartSuiteMergedProductFeatures(existing, latest string) string {
	existing = strings.TrimSpace(existing)
	latest = strings.TrimSpace(latest)
	if existing == "" {
		return latest
	}
	if latest == "" || strings.Contains(latest, existing) {
		return existing
	}
	return existing + "\n" + latest
}

func smartSuiteLooksLikeProductFacts(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if strings.HasPrefix(lower, "产品名称") || strings.HasPrefix(lower, "产品信息") || strings.HasPrefix(lower, "这是") {
		return true
	}
	for _, marker := range []string{"适用场景", "使用场景", "用于", "适用于", "compatible with", "designed for"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	factMarkers := 0
	for _, marker := range []string{"材质", "结构", "尺寸", "长度", "功能", "型号", "认证", "龙头", "水管", "阀芯", "不锈钢", "黄铜", "英寸", "material", "dimension", "feature", "tested", "certified", "faucet", "hose", "valve", "spray", "stream", "inch", "lead-free"} {
		if strings.Contains(lower, marker) {
			factMarkers++
		}
	}
	return factMarkers >= 2 || len([]rune(lower)) > 160 && factMarkers > 0
}

func smartSuiteLooksLikeExplicitVisualDirection(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	lower = strings.ReplaceAll(lower, "适用场景", "")
	lower = strings.ReplaceAll(lower, "使用场景", "")
	for _, marker := range []string{"背景", "场景换", "场景要", "场景用", "想要", "希望画面", "画面", "展示在", "放在", "视觉", "风格", "光线", "自然光", "氛围", "构图", "棚拍", "色调", "background", "show it in", "place it in", "lighting", "visual style", "composition"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func smartSuiteLooksLikeConversationControl(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, marker := range []string{"没理解", "不理解", "听不懂", "有问题", "不对", "为什么", "怎么回事", "重新理解", "understand me", "didn't understand", "did not understand", "why can't", "wrong result"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func smartSuiteUnsupportedFinalStyle(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, negative := range []string{"不要白底", "不要纯白", "拒绝白底", "不是白底", "避免白底", "not a white background", "avoid white background", "no white background"} {
		if strings.Contains(lower, negative) {
			return false
		}
	}
	for _, marker := range []string{"白底", "纯白背景", "纯色背景", "空白背景", "占位背景", "white background", "solid background", "blank background", "placeholder background"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func smartSuiteLooksLikeOtherRequirement(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, marker := range []string{"amazon", "walmart", "tiktok", "home depot", "wayfair", "lowe's", "temu", "站点", "副图", "a+", "模型", "张图", "英文", "中文", "德文", "法文", "日文", "韩文", "开始生成", "停止生成"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func smartSuiteProductNameFromText(text string) string {
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		explicitName := regexp.MustCompile(`(?i)^(?:这是|产品(?:名称)?(?:是|为|：|:)|名称(?:是|为|：|:))\s*`).MatchString(line)
		for _, separator := range []string{"：", ":"} {
			if index := strings.Index(line, separator); index >= 0 {
				line = strings.TrimSpace(line[index+len(separator):])
				break
			}
		}
		line = regexp.MustCompile(`(?i)^(?:这是|产品(?:名称)?(?:是|为)|名称(?:是|为))\s*`).ReplaceAllString(line, "")
		line = regexp.MustCompile(`(?i)\s+with\s+|\s+which\s+|[,，;；。.!?]`).Split(line, 2)[0]
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		identityMarker := explicitName
		for _, marker := range []string{"龙头", "水管", "软管", "置物架", "收纳架", "支架", "faucet", "hose", "holder", "rack", "fixture"} {
			identityMarker = identityMarker || strings.Contains(lower, marker)
		}
		if identityMarker && len([]rune(line)) >= 2 && len([]rune(line)) <= 80 {
			return line
		}
	}
	return ""
}

func smartSuiteCoordinatorOutputAnswersField(output smartSuiteCoordinatorOutput, field string) bool {
	switch strings.TrimSpace(field) {
	case "visual_style":
		return nonEmptySmartSuiteString(output.Updates.VisualStyle) || len(output.Updates.StyleReferenceURLs) > 0
	case "product_info":
		return nonEmptySmartSuiteString(output.Updates.ProductName) && nonEmptySmartSuiteString(output.Updates.ProductFeatures)
	case "platform_site":
		return nonEmptySmartSuiteString(output.Updates.Platform) && (nonEmptySmartSuiteString(output.Updates.CountryCode) || nonEmptySmartSuiteString(output.Updates.PlatformName))
	case "output_counts":
		return output.Updates.SecondaryCount != nil || output.Updates.DesktopAPlusCount != nil || output.Updates.MobileAPlusCount != nil
	case "language":
		return nonEmptySmartSuiteString(output.Updates.Language)
	default:
		return true
	}
}

func runValidatedSmartSuiteConversationCoordinator(conversation model.SmartSuiteConversation, req smartSuiteConversationTurnRequest, attempts int) (smartSuiteCoordinatorOutput, error) {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		output, err := runSmartSuiteConversationCoordinator(conversation, req)
		if err == nil {
			err = validateSmartSuiteCoordinatorOutput(conversation.Context, req, &output)
		}
		if err == nil {
			return output, nil
		}
		lastErr = err
	}
	return smartSuiteCoordinatorOutput{}, lastErr
}

func smartSuiteExplicitStartText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	return regexp.MustCompile(`(?i)^(开始|开始生成|确认生成|确认开始|可以了|没问题|就这样|生成吧|开始吧|重新生成|再生成一次|再生成|go|start|regenerate)[。.!！\s]*$`).MatchString(trimmed)
}

func smartSuiteCandidateIntent(text string) string {
	if regexp.MustCompile(`^(采用|使用|确认|就这个|用这个版本|采用这个版本)[。！!\s]*$`).MatchString(strings.TrimSpace(text)) {
		return "accept_candidate"
	}
	if regexp.MustCompile(`^(保留原图|不要这个|拒绝|取消修改|还是原图)[。！!\s]*$`).MatchString(strings.TrimSpace(text)) {
		return "reject_candidate"
	}
	return "collect_requirement"
}

func smartSuiteOutputCountUpdatesFromText(text string) (smartSuiteCoordinatorUpdates, bool) {
	normalized := strings.ToLower(strings.Join(strings.Fields(text), ""))
	if normalized == "" {
		return smartSuiteCoordinatorUpdates{}, false
	}

	numberPattern := `(\d+|[一二两三四五六七八九十])`
	typePatterns := []string{
		`(?:副图|附图)`,
		`(?:桌面(?:端)?a\+|pc(?:端)?a\+|电脑(?:端)?a\+)`,
		`(?:移动(?:端)?a\+|手机(?:端)?a\+)`,
	}
	for _, typePattern := range typePatterns {
		if regexp.MustCompile(typePattern + `第` + numberPattern + `张`).MatchString(normalized) {
			return smartSuiteCoordinatorUpdates{}, false
		}
	}

	defaults := []int{6, 7, 7}
	counts := []int{0, 0, 0}
	mentioned := []bool{false, false, false}
	eachCount, hasEachCount := 0, false
	if match := regexp.MustCompile(`各` + numberPattern + `(?:张|个)`).FindStringSubmatch(normalized); len(match) > 1 {
		eachCount, hasEachCount = smartSuiteCountToken(match[1])
	}

	for index, typePattern := range typePatterns {
		if !regexp.MustCompile(typePattern).MatchString(normalized) {
			continue
		}
		mentioned[index] = true
		counts[index] = defaults[index]
		patterns := []*regexp.Regexp{
			regexp.MustCompile(typePattern + `(?:要|做|生成|需要|是|共)?` + numberPattern + `(?:张|个)`),
			regexp.MustCompile(numberPattern + `(?:张|个)` + typePattern),
		}
		for _, pattern := range patterns {
			match := pattern.FindStringSubmatch(normalized)
			if len(match) < 2 {
				continue
			}
			if count, ok := smartSuiteCountToken(match[1]); ok {
				counts[index] = count
			}
			break
		}
		if hasEachCount {
			hasExplicitCount := false
			for _, pattern := range patterns {
				if pattern.MatchString(normalized) {
					hasExplicitCount = true
					break
				}
			}
			if !hasExplicitCount {
				counts[index] = eachCount
			}
		}
	}
	if !mentioned[0] && !mentioned[1] && !mentioned[2] {
		return smartSuiteCoordinatorUpdates{}, false
	}
	return smartSuiteCoordinatorUpdates{
		SecondaryCount:    &counts[0],
		DesktopAPlusCount: &counts[1],
		MobileAPlusCount:  &counts[2],
	}, true
}

func smartSuiteCountToken(token string) (int, bool) {
	if value, err := strconv.Atoi(token); err == nil {
		return value, true
	}
	value, ok := smartSuiteNumberWords[token]
	return value, ok
}

func resolveSmartSuiteImageReference(conversation model.SmartSuiteConversation, text string) uint64 {
	if strings.TrimSpace(conversation.TaskID) == "" {
		return 0
	}
	imageType := ""
	switch {
	case regexp.MustCompile(`副图|附图`).MatchString(text):
		imageType = "smart_secondary"
	case regexp.MustCompile(`(?i)桌面\s*A\+|PC\s*端\s*A\+|电脑端\s*A\+|桌面A\+|桌面端|PC端|电脑端`).MatchString(text):
		imageType = "smart_desktop_a_plus"
	case regexp.MustCompile(`移动\s*A\+|手机端\s*A\+|移动端\s*A\+|手机A\+|手机端|移动端`).MatchString(text):
		imageType = "smart_mobile_a_plus"
	}
	index := smartSuiteImageReferenceIndex(text)
	if imageType == "" && index == 0 {
		if regexp.MustCompile(`这张|当前|刚才|继续修改|这个版本|修改版本|候选图|原图`).MatchString(text) {
			return conversation.Context.ActiveImageID
		}
		return 0
	}
	var images []model.EcomSuiteImage
	query := database.DB.Select("id", "image_type", "image_index").Where("task_id = ?", conversation.TaskID)
	if imageType != "" {
		query = query.Where("image_type = ?", imageType)
	}
	if err := query.Order("image_type, image_index").Find(&images).Error; err != nil || len(images) == 0 {
		return 0
	}
	return resolveSmartSuiteImageReferenceFromImages(images, imageType, index)
}

func resolveSmartSuiteImageReferenceFromImages(images []model.EcomSuiteImage, imageType string, index int) uint64 {
	if imageType == "" && index > 0 {
		for _, image := range images {
			if image.ImageType == "smart_secondary" && image.ImageIndex == index {
				return image.ID
			}
		}
	}
	matching := make([]model.EcomSuiteImage, 0, len(images))
	for _, image := range images {
		if imageType == "" || image.ImageType == imageType {
			matching = append(matching, image)
		}
	}
	if index > 0 {
		for _, image := range matching {
			if image.ImageIndex == index {
				return image.ID
			}
		}
	}
	if len(matching) == 1 {
		return matching[0].ID
	}
	return 0
}

func smartSuiteImageReferenceIndex(text string) int {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`第\s*(\d+|[一二两三四五六七八九十])\s*张`),
		regexp.MustCompile(`(?:副图|附图|桌面\s*A\+|移动\s*A\+|手机端\s*A\+)[^\d一二两三四五六七八九十]{0,4}(\d+|[一二两三四五六七八九十])`),
	}
	for _, pattern := range patterns {
		match := pattern.FindStringSubmatch(text)
		if len(match) < 2 {
			continue
		}
		value, err := strconv.Atoi(match[1])
		if err != nil {
			value = smartSuiteNumberWords[match[1]]
		}
		return value
	}
	return 0
}

func mergeSmartSuiteCoordinatorOutput(base, ai smartSuiteCoordinatorOutput) smartSuiteCoordinatorOutput {
	if ai.TargetImageID > 0 {
		base.TargetImageID = ai.TargetImageID
	}
	if ai.Intent == "edit_image" && base.TargetImageID > 0 {
		base.Intent = ai.Intent
	}
	if (ai.Intent == "accept_candidate" || ai.Intent == "reject_candidate") && base.TargetImageID > 0 {
		base.Intent = ai.Intent
	}
	base.Reply = strings.TrimSpace(ai.Reply)
	base.RequirementKind = ai.RequirementKind
	base.Confidence = ai.Confidence
	base.ProductEvidence = ai.ProductEvidence
	base.VisualEvidence = ai.VisualEvidence
	base.OtherEvidence = ai.OtherEvidence
	base.NeedsClarification = ai.NeedsClarification
	base.ClarificationQuestion = strings.TrimSpace(ai.ClarificationQuestion)
	base.ClearFields = append([]string(nil), ai.ClearFields...)
	base.PublicSummary = strings.TrimSpace(ai.PublicSummary)
	base.PublicActions = normalizeSmartSuitePublicActions(ai.PublicActions)
	if strings.TrimSpace(ai.RequestType) != "" {
		base.RequestType = normalizeSmartSuiteRequestType(ai.RequestType)
		base.OpenRequirementForm = ai.OpenRequirementForm
	}
	mergeSmartSuiteCoordinatorUpdates(&base.Updates, ai.Updates)
	return base
}

func mergeSmartSuiteCoordinatorFocusedFallback(base, fallback smartSuiteCoordinatorOutput) smartSuiteCoordinatorOutput {
	wasClarification := base.NeedsClarification
	if nonEmptySmartSuiteString(base.Updates.ProductName) {
		fallback.Updates.ProductName = nil
	}
	if nonEmptySmartSuiteString(base.Updates.ProductFeatures) {
		fallback.Updates.ProductFeatures = nil
	}
	if nonEmptySmartSuiteString(base.Updates.VisualStyle) {
		fallback.Updates.VisualStyle = nil
	}
	if nonEmptySmartSuiteString(base.Updates.Platform) {
		fallback.Updates.Platform = nil
	}
	if nonEmptySmartSuiteString(base.Updates.PlatformName) {
		fallback.Updates.PlatformName = nil
	}
	if nonEmptySmartSuiteString(base.Updates.CountryCode) {
		fallback.Updates.CountryCode = nil
	}
	if nonEmptySmartSuiteString(base.Updates.CountryName) {
		fallback.Updates.CountryName = nil
	}
	if base.Updates.SecondaryCount != nil {
		fallback.Updates.SecondaryCount = nil
	}
	if base.Updates.DesktopAPlusCount != nil {
		fallback.Updates.DesktopAPlusCount = nil
	}
	if base.Updates.MobileAPlusCount != nil {
		fallback.Updates.MobileAPlusCount = nil
	}
	mergeSmartSuiteCoordinatorUpdates(&base.Updates, fallback.Updates)
	if smartSuiteHasProductSemanticUpdate(fallback.Updates) {
		base.ProductEvidence = append(base.ProductEvidence, fallback.ProductEvidence...)
	}
	if smartSuiteHasVisualSemanticUpdate(fallback.Updates) {
		base.VisualEvidence = append(base.VisualEvidence, fallback.VisualEvidence...)
	}
	if smartSuiteHasOtherRequirementUpdate(fallback.Updates) {
		base.OtherEvidence = append(base.OtherEvidence, fallback.OtherEvidence...)
	}
	base.RequirementKind = smartSuiteRequirementKindFromUpdates(base.RequirementKind,
		smartSuiteHasProductSemanticUpdate(base.Updates), smartSuiteHasVisualSemanticUpdate(base.Updates), smartSuiteHasOtherRequirementUpdate(base.Updates))
	base.NeedsClarification = false
	base.ClarificationQuestion = ""
	if wasClarification {
		base.PublicSummary = ""
		base.PublicActions = nil
	}
	return base
}

func normalizeSmartSuitePublicActions(actions []string) []string {
	result := make([]string, 0, len(actions))
	seen := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		action = strings.TrimSpace(action)
		if action == "" {
			continue
		}
		if _, exists := seen[action]; exists {
			continue
		}
		seen[action] = struct{}{}
		result = append(result, action)
		if len(result) == 6 {
			break
		}
	}
	return result
}

func applySmartSuiteCoordinatorClears(ctx *model.SmartSuiteConversationContext, fields []string) {
	if ctx == nil {
		return
	}
	for _, field := range fields {
		switch strings.TrimSpace(field) {
		case "product_name":
			ctx.ProductName = ""
		case "product_category":
			ctx.ProductCategory = ""
		case "product_brand":
			ctx.ProductBrand = ""
		case "product_features":
			ctx.ProductFeatures = ""
		case "product_proof":
			ctx.ProductProof = ""
		case "visual_style":
			ctx.VisualStyle = ""
		case "style_requirements":
			ctx.StyleRequirements = ""
		}
	}
}

func mergeSmartSuiteCoordinatorUpdates(target *smartSuiteCoordinatorUpdates, source smartSuiteCoordinatorUpdates) {
	if nonEmptySmartSuiteString(source.ProductName) {
		target.ProductName = source.ProductName
	}
	if nonEmptySmartSuiteString(source.ProductCategory) {
		target.ProductCategory = source.ProductCategory
	}
	if nonEmptySmartSuiteString(source.ProductBrand) {
		target.ProductBrand = source.ProductBrand
	}
	if nonEmptySmartSuiteString(source.ProductFeatures) {
		target.ProductFeatures = source.ProductFeatures
	}
	if nonEmptySmartSuiteString(source.ProductProof) {
		target.ProductProof = source.ProductProof
	}
	if nonEmptySmartSuiteString(source.Platform) {
		target.Platform = source.Platform
	}
	if nonEmptySmartSuiteString(source.PlatformName) {
		target.PlatformName = source.PlatformName
	}
	if nonEmptySmartSuiteString(source.CountryCode) {
		target.CountryCode = source.CountryCode
	}
	if nonEmptySmartSuiteString(source.CountryName) {
		target.CountryName = source.CountryName
	}
	if nonEmptySmartSuiteString(source.Language) {
		target.Language = source.Language
	}
	if nonEmptySmartSuiteString(source.LanguageName) {
		target.LanguageName = source.LanguageName
	}
	if nonEmptySmartSuiteString(source.OutputTarget) {
		target.OutputTarget = source.OutputTarget
	}
	if source.SecondaryCount != nil {
		target.SecondaryCount = source.SecondaryCount
	}
	if source.DesktopAPlusCount != nil {
		target.DesktopAPlusCount = source.DesktopAPlusCount
	}
	if source.MobileAPlusCount != nil {
		target.MobileAPlusCount = source.MobileAPlusCount
	}
	if nonEmptySmartSuiteString(source.VisualStyle) {
		target.VisualStyle = source.VisualStyle
	}
	if nonEmptySmartSuiteString(source.StyleRequirements) {
		target.StyleRequirements = source.StyleRequirements
	}
	if len(source.StyleReferenceURLs) > 0 {
		target.StyleReferenceURLs = append([]string(nil), source.StyleReferenceURLs...)
	}
	if nonEmptySmartSuiteString(source.DrawingModel) {
		target.DrawingModel = source.DrawingModel
	}
}

func applySmartSuiteCoordinatorUpdates(ctx *model.SmartSuiteConversationContext, updates smartSuiteCoordinatorUpdates) {
	if nonEmptySmartSuiteString(updates.ProductName) {
		ctx.ProductName = strings.TrimSpace(*updates.ProductName)
	}
	if nonEmptySmartSuiteString(updates.ProductCategory) {
		ctx.ProductCategory = strings.TrimSpace(*updates.ProductCategory)
	}
	if nonEmptySmartSuiteString(updates.ProductBrand) {
		ctx.ProductBrand = strings.TrimSpace(*updates.ProductBrand)
	}
	if nonEmptySmartSuiteString(updates.ProductFeatures) {
		ctx.ProductFeatures = strings.TrimSpace(*updates.ProductFeatures)
	}
	if nonEmptySmartSuiteString(updates.ProductProof) {
		ctx.ProductProof = strings.TrimSpace(*updates.ProductProof)
	}
	if nonEmptySmartSuiteString(updates.Platform) {
		ctx.Platform = normalizeListingPlatform(*updates.Platform)
	}
	if nonEmptySmartSuiteString(updates.PlatformName) {
		ctx.PlatformName = strings.TrimSpace(*updates.PlatformName)
	}
	if nonEmptySmartSuiteString(updates.CountryCode) {
		ctx.CountryCode = strings.ToUpper(strings.TrimSpace(*updates.CountryCode))
	}
	if nonEmptySmartSuiteString(updates.CountryName) {
		ctx.CountryName = strings.TrimSpace(*updates.CountryName)
	}
	if nonEmptySmartSuiteString(updates.Language) {
		ctx.Language = strings.ToLower(strings.TrimSpace(*updates.Language))
	}
	if nonEmptySmartSuiteString(updates.LanguageName) {
		ctx.LanguageName = strings.TrimSpace(*updates.LanguageName)
	}
	if nonEmptySmartSuiteString(updates.OutputTarget) {
		ctx.OutputTarget = strings.ToLower(strings.TrimSpace(*updates.OutputTarget))
	}
	if updates.SecondaryCount != nil {
		ctx.SecondaryCount = *updates.SecondaryCount
	}
	if updates.DesktopAPlusCount != nil {
		ctx.DesktopAPlusCount = *updates.DesktopAPlusCount
	}
	if updates.MobileAPlusCount != nil {
		ctx.MobileAPlusCount = *updates.MobileAPlusCount
	}
	if nonEmptySmartSuiteString(updates.VisualStyle) {
		ctx.VisualStyle = strings.TrimSpace(*updates.VisualStyle)
	}
	if nonEmptySmartSuiteString(updates.StyleRequirements) {
		ctx.StyleRequirements = strings.TrimSpace(*updates.StyleRequirements)
	}
	if len(updates.StyleReferenceURLs) > 0 {
		for _, url := range updates.StyleReferenceURLs {
			ctx.StyleReferenceURLs = appendUniqueSmartSuiteURLs(ctx.StyleReferenceURLs, url)
		}
	}
	if nonEmptySmartSuiteString(updates.DrawingModel) {
		ctx.DrawingModel = strings.TrimSpace(*updates.DrawingModel)
	}
}

func nonEmptySmartSuiteString(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}

func appendUniqueSmartSuiteURLs(existing []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return existing
	}
	for _, item := range existing {
		if strings.TrimSpace(item) == value {
			return existing
		}
	}
	return append(existing, value)
}

func normalizeSmartSuiteConversationContext(ctx *model.SmartSuiteConversationContext) {
	ctx.ProductReferenceURLs = normalizeSmartSuiteProductReferenceURLs(ctx.WhiteBgImageURL, ctx.ProductReferenceURLs)
	if strings.TrimSpace(ctx.VisualStyle) == smartSuiteLegacyAutoVisualStyle {
		ctx.VisualStyle = smartSuiteAutoVisualStyle
	}
	ctx.RequestType = normalizeSmartSuiteRequestType(ctx.RequestType)
	ctx.IncludeListing = smartSuiteRequestTypeIncludesListing(ctx.RequestType) || ctx.IncludeListing
	platforms := map[string]string{"amazon": "Amazon", "walmart": "Walmart", "tiktok": "TikTok Shop", "home_depot": "Home Depot", "wayfair": "Wayfair", "lowes": "Lowe's", "shopify": "Shopify", "temu": "Temu"}
	if name := platforms[strings.ToLower(ctx.Platform)]; name != "" && ctx.PlatformName == "" {
		ctx.PlatformName = name
	}
	if strings.TrimSpace(ctx.Language) == "" {
		if language, languageName := smartSuitePlatformLanguage(*ctx); language != "" {
			ctx.Language = language
			ctx.LanguageName = languageName
		}
	}
	if ctx.Language == "en" && ctx.LanguageName == "" {
		ctx.LanguageName = "英文"
	}
	if ctx.Language == "zh" && ctx.LanguageName == "" {
		ctx.LanguageName = "中文"
	}
	ctx.SecondaryCount = clampSmartSuiteCount(ctx.SecondaryCount, 9)
	ctx.DesktopAPlusCount = clampSmartSuiteCount(ctx.DesktopAPlusCount, 8)
	ctx.MobileAPlusCount = clampSmartSuiteCount(ctx.MobileAPlusCount, 8)
	ctx.OutputTarget = smartSuiteTargetFromCounts(*ctx)
}

func smartSuitePlatformLanguage(ctx model.SmartSuiteConversationContext) (string, string) {
	if strings.TrimSpace(ctx.Platform) == "" || strings.TrimSpace(ctx.CountryCode) == "" {
		return "", ""
	}
	country := strings.ToUpper(strings.TrimSpace(ctx.CountryCode))
	name := strings.ToLower(strings.TrimSpace(ctx.CountryName))
	switch {
	case country == "CN" || strings.Contains(name, "中国") || strings.Contains(name, "china"):
		return "zh", "中文"
	case country == "DE" || strings.Contains(name, "德国") || strings.Contains(name, "germany"):
		return "de", "德文"
	case country == "FR" || strings.Contains(name, "法国") || strings.Contains(name, "france"):
		return "fr", "法文"
	case country == "ES" || strings.Contains(name, "西班牙") || strings.Contains(name, "spain"):
		return "es", "西班牙文"
	case country == "IT" || strings.Contains(name, "意大利") || strings.Contains(name, "italy"):
		return "it", "意大利文"
	case country == "JP" || strings.Contains(name, "日本") || strings.Contains(name, "japan"):
		return "ja", "日文"
	case country == "KR" || strings.Contains(name, "韩国") || strings.Contains(name, "korea"):
		return "ko", "韩文"
	default:
		// European aggregate sites and English-language marketplaces use English
		// as the safe default; users can still explicitly override the language.
		return "en", "英文"
	}
}

func clampSmartSuiteCount(value, max int) int {
	if value < 0 {
		return 0
	}
	if value > max {
		return max
	}
	return value
}

func smartSuiteTargetFromCounts(ctx model.SmartSuiteConversationContext) string {
	parts := make([]string, 0, 3)
	if ctx.SecondaryCount > 0 {
		parts = append(parts, "secondary")
	}
	if ctx.DesktopAPlusCount > 0 {
		parts = append(parts, "desktop")
	}
	if ctx.MobileAPlusCount > 0 {
		parts = append(parts, "mobile")
	}
	if len(parts) == 3 {
		if ctx.RequestType == smartSuiteRequestTypeListing {
			return "listing"
		}
		return "all"
	}
	if ctx.RequestType == smartSuiteRequestTypeListing {
		return "listing"
	}
	return strings.Join(parts, "_")
}

func smartSuiteMissingFields(ctx model.SmartSuiteConversationContext) []string {
	missing := make([]string, 0, 8)
	listingOnly := ctx.RequestType == smartSuiteRequestTypeListing
	if strings.TrimSpace(ctx.WhiteBgImageURL) == "" && !listingOnly {
		missing = append(missing, "white_bg_image")
	}
	if strings.TrimSpace(ctx.ProductName) == "" {
		missing = append(missing, "product_name")
	}
	if strings.TrimSpace(ctx.ProductFeatures) == "" {
		missing = append(missing, "product_features")
	}
	if strings.TrimSpace(ctx.Platform) == "" || strings.TrimSpace(ctx.CountryCode) == "" {
		missing = append(missing, "platform_site")
	}
	if !listingOnly && ctx.SecondaryCount+ctx.DesktopAPlusCount+ctx.MobileAPlusCount == 0 {
		missing = append(missing, "output_counts")
	}
	if !listingOnly && strings.TrimSpace(ctx.DrawingModel) == "" {
		missing = append(missing, "drawing_model")
	}
	return missing
}

func smartSuiteHasVisualStyle(ctx model.SmartSuiteConversationContext) bool {
	return strings.TrimSpace(ctx.VisualStyle) != "" || len(ctx.StyleReferenceURLs) > 0
}

func smartSuiteCoordinatorReply(conversation model.SmartSuiteConversation, output smartSuiteCoordinatorOutput, hasCandidate bool) string {
	ctx := conversation.Context
	if output.OpenRequirementForm && output.RequestType != "" {
		switch normalizeSmartSuiteRequestType(output.RequestType) {
		case smartSuiteRequestTypeListing:
			return "好的，你要的是 Listing 文案。我先打开需求表单，填写产品事实和规格；平台规则与关键词库会自动加载。"
		case smartSuiteRequestTypeSuiteAndListing:
			return "好的，这次同时生成套图和 Listing。我先打开需求表单，填写一次即可。"
		default:
			return "好的，这次生成套图。我先打开需求表单，填写一次即可。"
		}
	}
	switch output.Intent {
	case "stop":
		return "好的，我现在停止生成，已经完成的图片会保留。"
	case "edit_image":
		return "收到，我会保持产品外观和SKU不变，按这条要求生成一个修改版本。"
	case "accept_candidate":
		return "好的，采用这个修改版本。"
	case "reject_candidate":
		return "好的，保留原图。"
	case "start":
		if len(smartSuiteMissingFields(ctx)) == 0 {
			return "信息完整，开始生成。"
		}
	}
	if output.NeedsClarification {
		if question := strings.TrimSpace(output.ClarificationQuestion); question != "" {
			return question
		}
		return "请说明这段内容是产品本身的信息、希望生成的视觉方向，还是两者都有。"
	}
	missing := smartSuiteMissingFields(ctx)
	if containsSmartSuiteField(missing, "white_bg_image") {
		return "先发我一张白底产品图。你可以在同一条消息里补充产品名称和核心信息。"
	}
	if containsSmartSuiteField(missing, "product_name") || containsSmartSuiteField(missing, "product_features") {
		return "产品图已收到。请补充产品名称，以及材质、结构、尺寸、功能、适用场景或可证明效果等核心事实。知道多少说多少即可；这些内容会作为产品信息保存，不会当作视觉风格。"
	}
	if containsSmartSuiteField(missing, "platform_site") {
		return "这套图准备发布到哪个平台和站点？例如 Amazon 美国站。"
	}
	if containsSmartSuiteField(missing, "output_counts") {
		return "需要生成哪些内容、各多少张？标准整套包含 6 张副图、7 张桌面 A+、7 张移动 A+。"
	}
	if containsSmartSuiteField(missing, "drawing_model") {
		options, _ := listSmartSuiteDrawingModelOptions()
		if len(options) == 0 {
			return "后台暂时没有可用的生图模型，请联系管理员先启用 drawing 模型。"
		}
		return "最后选择这套图使用的生图模型。不同模型侧重点不同，可以直接选择下面的模型。"
	}
	if hasCandidate {
		return "当前修改版本正在等待确认。你可以直接说“采用这个版本”“继续修改”或“保留原图”。"
	}
	return smartSuiteConfirmationReply(ctx)
}

func smartSuiteConversationSuggestions(ctx model.SmartSuiteConversationContext, intent string, hasActiveImage bool) []string {
	if intent == "stop" || intent == "start" || intent == "edit_image" || intent == "accept_candidate" || intent == "reject_candidate" {
		return nil
	}
	missing := smartSuiteMissingFields(ctx)
	if containsSmartSuiteField(missing, "white_bg_image") {
		return nil
	}
	if containsSmartSuiteField(missing, "product_name") || containsSmartSuiteField(missing, "product_features") {
		return nil
	}
	if containsSmartSuiteField(missing, "platform_site") {
		return []string{"TikTok Shop 美国站", "Home Depot 美国站", "Lowe's 美国站", "Wayfair 美国站", "Amazon 美国站"}
	}
	if containsSmartSuiteField(missing, "output_counts") {
		return []string{"6 张副图", "7 张桌面 A+", "7 张移动 A+", "副图、桌面 A+、移动 A+ 都要"}
	}
	if containsSmartSuiteField(missing, "drawing_model") {
		return nil
	}
	if hasActiveImage {
		return []string{"继续修改这张", "保留原图"}
	}
	if len(missing) == 0 {
		return []string{"开始生成", "我还要调整需求"}
	}
	return nil
}

func smartSuiteConversationSuggestionField(ctx model.SmartSuiteConversationContext, intent string, hasActiveImage bool) string {
	if intent == "stop" || intent == "start" || intent == "edit_image" || intent == "accept_candidate" || intent == "reject_candidate" || hasActiveImage {
		return ""
	}
	missing := smartSuiteMissingFields(ctx)
	switch {
	case containsSmartSuiteField(missing, "white_bg_image"):
		return "white_bg_image"
	case containsSmartSuiteField(missing, "product_name") || containsSmartSuiteField(missing, "product_features"):
		return "product_info"
	case containsSmartSuiteField(missing, "platform_site"):
		return "platform_site"
	case containsSmartSuiteField(missing, "output_counts"):
		return "output_counts"
	case containsSmartSuiteField(missing, "drawing_model"):
		return "drawing_model"
	default:
		return ""
	}
}

func containsSmartSuiteField(fields []string, target string) bool {
	for _, field := range fields {
		if field == target {
			return true
		}
	}
	return false
}

func smartSuiteConfirmationReply(ctx model.SmartSuiteConversationContext) string {
	platform := strings.TrimSpace(strings.TrimSpace(ctx.PlatformName) + " " + strings.TrimSpace(ctx.CountryName))
	if ctx.RequestType == smartSuiteRequestTypeListing {
		return fmt.Sprintf("信息已保存：\n%s · Listing 文案\n输出：%s\n语言：%s\n产品：%s\n%s\n\n平台规则与关键词库将自动应用，正在开始生成。", platform, smartSuiteListingOutputLabel(ctx.Platform), ctx.LanguageName, ctx.ProductName, smartSuiteFeatureSummary(ctx.ProductFeatures))
	}
	parts := make([]string, 0, 3)
	if ctx.SecondaryCount > 0 {
		parts = append(parts, fmt.Sprintf("副图 %d 张", ctx.SecondaryCount))
	}
	if ctx.DesktopAPlusCount > 0 {
		parts = append(parts, fmt.Sprintf("桌面 A+ %d 张", ctx.DesktopAPlusCount))
	}
	if ctx.MobileAPlusCount > 0 {
		parts = append(parts, fmt.Sprintf("移动 A+ %d 张", ctx.MobileAPlusCount))
	}
	style := strings.TrimSpace(ctx.VisualStyle)
	if style == "" && len(ctx.StyleReferenceURLs) > 0 {
		style = fmt.Sprintf("已使用 %d 张风格参考图", len(ctx.StyleReferenceURLs))
	}
	if style == "" {
		style = "AI 根据产品自主制定高端风格"
	}
	return fmt.Sprintf("我理解好了：\n%s · %s\n视觉风格：%s\n生图模型：%s\n图片文案：%s\n产品：%s\n%s\n\n回复“开始生成”即可开始，也可以直接告诉我哪里需要调整。", platform, strings.Join(parts, "、"), style, ctx.DrawingModel, ctx.LanguageName, ctx.ProductName, smartSuiteFeatureSummary(ctx.ProductFeatures))
}

func smartSuiteListingOutputLabel(platform string) string {
	switch normalizeListingPlatform(platform) {
	case "tiktok":
		return "标题、描述、产品属性、变体与包装信息"
	case "home_depot":
		return "Title、Salient Bullets、Bullet Points 与 About This Product"
	case "lowes":
		return "Title、Description 与 Bullets"
	case "wayfair":
		return "标题、Description、Features、Included/Not Included、Dimensions 与 Details"
	default:
		return "标题、5 条卖点、描述和搜索词"
	}
}

func smartSuiteFeatureSummary(features string) string {
	features = strings.TrimSpace(features)
	if len([]rune(features)) <= 100 {
		return features
	}
	return string([]rune(features)[:100]) + "..."
}

func validateSmartSuiteCoordinatorOutput(current model.SmartSuiteConversationContext, req smartSuiteConversationTurnRequest, output *smartSuiteCoordinatorOutput) error {
	if output == nil {
		return fmt.Errorf("conversation_coordinator_empty_output")
	}
	output.PublicSummary = strings.TrimSpace(output.PublicSummary)
	output.PublicActions = normalizeSmartSuitePublicActions(output.PublicActions)
	if strings.TrimSpace(output.RequestType) != "" {
		rawRequestType := strings.ToLower(strings.TrimSpace(output.RequestType))
		if rawRequestType != smartSuiteRequestTypeSuite && rawRequestType != smartSuiteRequestTypeListing && rawRequestType != smartSuiteRequestTypeSuiteAndListing {
			return fmt.Errorf("conversation_coordinator_invalid_request_type")
		}
		output.RequestType = rawRequestType
		output.OpenRequirementForm = true
	}
	if output.OpenRequirementForm && strings.TrimSpace(output.RequestType) == "" {
		return fmt.Errorf("conversation_coordinator_form_type_missing")
	}
	dropUnchangedSmartSuiteCoordinatorUpdates(current, &output.Updates)
	declaredKind := strings.ToLower(strings.TrimSpace(output.RequirementKind))
	output.RequirementKind = declaredKind
	if output.NeedsClarification || declaredKind == "ambiguous" {
		output.RequirementKind = "ambiguous"
		clearSmartSuiteRequirementUpdates(&output.Updates)
		output.ClearFields = nil
		output.NeedsClarification = true
		if strings.TrimSpace(output.ClarificationQuestion) == "" {
			output.ClarificationQuestion = "请说明这段内容是产品本身的信息、希望生成的视觉方向，还是两者都有。"
		}
		return nil
	}

	hasProduct := smartSuiteHasProductSemanticUpdate(output.Updates)
	hasVisual := smartSuiteHasVisualSemanticUpdate(output.Updates)
	hasOther := smartSuiteHasOtherRequirementUpdate(output.Updates)
	if _, revisionRequest := smartSuiteRequirementRevisionFallback(req.Text); revisionRequest &&
		!output.NeedsClarification && !hasProduct && !hasVisual && !hasOther && len(output.ClearFields) == 0 {
		return fmt.Errorf("conversation_coordinator_revision_detail_not_requested")
	}
	clearsProduct := smartSuiteClearsAnyField(output.ClearFields, "product_name", "product_category", "product_brand", "product_features", "product_proof")
	clearsVisual := smartSuiteClearsAnyField(output.ClearFields, "visual_style", "style_requirements")
	inferredKind := smartSuiteRequirementKindFromUpdates("", hasProduct, hasVisual, hasOther)
	validDeclaredKind := declaredKind == "product_facts" || declaredKind == "visual_direction" || declaredKind == "mixed" || declaredKind == "other"
	switch {
	case declaredKind == "":
		output.RequirementKind = inferredKind
	case !validDeclaredKind && inferredKind != "":
		output.RequirementKind = inferredKind
	case declaredKind == "mixed" && inferredKind != "" && inferredKind != "mixed":
		output.RequirementKind = inferredKind
	}
	if output.RequirementKind == "" {
		output.RequirementKind = "other"
	}
	switch output.RequirementKind {
	case "product_facts", "visual_direction", "mixed", "other":
	default:
		return fmt.Errorf("conversation_coordinator_invalid_requirement_kind")
	}
	requirementDecision := hasProduct || hasVisual || hasOther || clearsProduct || clearsVisual
	if requirementDecision && output.Confidence < 0.65 {
		return fmt.Errorf("conversation_coordinator_low_confidence")
	}
	switch output.RequirementKind {
	case "product_facts":
		if hasVisual || hasOther || clearsProduct && !hasProduct {
			return fmt.Errorf("conversation_coordinator_product_kind_mismatch")
		}
		if (hasProduct || clearsVisual) && !smartSuiteEvidenceSupported(req, output.ProductEvidence, "product") {
			return fmt.Errorf("conversation_coordinator_product_evidence_missing")
		}
	case "visual_direction":
		if hasProduct || hasOther || clearsVisual && !hasVisual {
			return fmt.Errorf("conversation_coordinator_visual_kind_mismatch")
		}
		if (hasVisual || clearsProduct) && !smartSuiteEvidenceSupported(req, output.VisualEvidence, "visual") {
			return fmt.Errorf("conversation_coordinator_visual_evidence_missing")
		}
	case "mixed":
		requirementGroups := 0
		if hasProduct || clearsProduct {
			requirementGroups++
		}
		if hasVisual || clearsVisual {
			requirementGroups++
		}
		if hasOther {
			requirementGroups++
		}
		if requirementGroups < 2 {
			return fmt.Errorf("conversation_coordinator_semantic_update_mismatch")
		}
		if (hasProduct || clearsProduct) && !smartSuiteEvidenceSupported(req, output.ProductEvidence, "product") {
			return fmt.Errorf("conversation_coordinator_mixed_product_evidence_missing")
		}
		if (hasVisual || clearsVisual) && !smartSuiteEvidenceSupported(req, output.VisualEvidence, "visual") {
			return fmt.Errorf("conversation_coordinator_mixed_evidence_missing")
		}
	case "other":
		if hasProduct || hasVisual || clearsProduct || clearsVisual {
			return fmt.Errorf("conversation_coordinator_other_contains_semantic_update")
		}
	}
	if hasOther && !smartSuiteEvidenceSupported(req, output.OtherEvidence, "other") {
		return fmt.Errorf("conversation_coordinator_other_evidence_missing")
	}
	for _, field := range output.ClearFields {
		switch strings.TrimSpace(field) {
		case "product_name", "product_category", "product_brand", "product_features", "product_proof", "visual_style", "style_requirements":
		default:
			return fmt.Errorf("conversation_coordinator_invalid_clear_field")
		}
	}
	if clearsProduct && output.RequirementKind == "visual_direction" && !smartSuiteClearFieldsOverlapEvidence(current, output.ClearFields, output.VisualEvidence) {
		return fmt.Errorf("conversation_coordinator_product_reclassification_without_matching_evidence")
	}
	if clearsVisual && output.RequirementKind == "product_facts" && !smartSuiteClearFieldsOverlapEvidence(current, output.ClearFields, output.ProductEvidence) {
		return fmt.Errorf("conversation_coordinator_visual_reclassification_without_matching_evidence")
	}
	if nonEmptySmartSuiteString(output.Updates.ProductName) && strings.TrimSpace(current.ProductName) != "" &&
		strings.TrimSpace(*output.Updates.ProductName) != strings.TrimSpace(current.ProductName) && !smartSuiteClearsAnyField(output.ClearFields, "product_name") {
		return fmt.Errorf("conversation_coordinator_product_name_overwrite_without_correction")
	}
	if nonEmptySmartSuiteString(output.Updates.ProductFeatures) && strings.TrimSpace(current.ProductFeatures) != "" &&
		!strings.Contains(strings.TrimSpace(*output.Updates.ProductFeatures), strings.TrimSpace(current.ProductFeatures)) && !smartSuiteClearsAnyField(output.ClearFields, "product_features") {
		return fmt.Errorf("conversation_coordinator_product_facts_dropped")
	}
	if nonEmptySmartSuiteString(output.Updates.VisualStyle) && strings.TrimSpace(current.VisualStyle) != "" &&
		strings.TrimSpace(*output.Updates.VisualStyle) != strings.TrimSpace(current.VisualStyle) && !smartSuiteClearsAnyField(output.ClearFields, "visual_style") {
		return fmt.Errorf("conversation_coordinator_visual_style_overwrite_without_correction")
	}
	return nil
}

func dropUnchangedSmartSuiteCoordinatorUpdates(current model.SmartSuiteConversationContext, updates *smartSuiteCoordinatorUpdates) {
	if updates == nil {
		return
	}
	dropString := func(value **string, existing string) {
		if *value != nil && strings.EqualFold(strings.TrimSpace(**value), strings.TrimSpace(existing)) {
			*value = nil
		}
	}
	dropCount := func(value **int, existing int) {
		if *value != nil && **value == existing {
			*value = nil
		}
	}
	dropString(&updates.ProductName, current.ProductName)
	dropString(&updates.ProductCategory, current.ProductCategory)
	dropString(&updates.ProductBrand, current.ProductBrand)
	dropString(&updates.ProductFeatures, current.ProductFeatures)
	dropString(&updates.ProductProof, current.ProductProof)
	dropString(&updates.Platform, current.Platform)
	dropString(&updates.PlatformName, current.PlatformName)
	dropString(&updates.CountryCode, current.CountryCode)
	dropString(&updates.CountryName, current.CountryName)
	dropString(&updates.Language, current.Language)
	dropString(&updates.LanguageName, current.LanguageName)
	dropString(&updates.OutputTarget, current.OutputTarget)
	dropCount(&updates.SecondaryCount, current.SecondaryCount)
	dropCount(&updates.DesktopAPlusCount, current.DesktopAPlusCount)
	dropCount(&updates.MobileAPlusCount, current.MobileAPlusCount)
	dropString(&updates.VisualStyle, current.VisualStyle)
	dropString(&updates.StyleRequirements, current.StyleRequirements)
	dropString(&updates.DrawingModel, current.DrawingModel)
}

func smartSuiteRequirementKindFromUpdates(current string, hasProduct, hasVisual, hasOther bool) string {
	groups := 0
	if hasProduct {
		groups++
	}
	if hasVisual {
		groups++
	}
	if hasOther {
		groups++
	}
	switch groups {
	case 0:
		return current
	case 1:
		switch {
		case hasProduct:
			return "product_facts"
		case hasVisual:
			return "visual_direction"
		default:
			return "other"
		}
	default:
		return "mixed"
	}
}

func smartSuiteClearsAnyField(fields []string, candidates ...string) bool {
	for _, field := range fields {
		for _, candidate := range candidates {
			if strings.TrimSpace(field) == candidate {
				return true
			}
		}
	}
	return false
}

func smartSuiteClearFieldsOverlapEvidence(current model.SmartSuiteConversationContext, fields []string, evidence []smartSuiteCoordinatorEvidence) bool {
	for _, field := range fields {
		value := strings.TrimSpace(smartSuiteSemanticFieldValue(current, strings.TrimSpace(field)))
		if value == "" {
			continue
		}
		for _, item := range evidence {
			quote := strings.TrimSpace(item.Quote)
			if quote != "" && (strings.Contains(value, quote) || strings.Contains(quote, value)) {
				return true
			}
		}
	}
	return false
}

func smartSuiteSemanticFieldValue(current model.SmartSuiteConversationContext, field string) string {
	switch field {
	case "product_name":
		return current.ProductName
	case "product_category":
		return current.ProductCategory
	case "product_brand":
		return current.ProductBrand
	case "product_features":
		return current.ProductFeatures
	case "product_proof":
		return current.ProductProof
	case "visual_style":
		return current.VisualStyle
	case "style_requirements":
		return current.StyleRequirements
	default:
		return ""
	}
}

func smartSuiteHasProductSemanticUpdate(updates smartSuiteCoordinatorUpdates) bool {
	return nonEmptySmartSuiteString(updates.ProductName) || nonEmptySmartSuiteString(updates.ProductCategory) ||
		nonEmptySmartSuiteString(updates.ProductBrand) || nonEmptySmartSuiteString(updates.ProductFeatures) ||
		nonEmptySmartSuiteString(updates.ProductProof)
}

func smartSuiteHasVisualSemanticUpdate(updates smartSuiteCoordinatorUpdates) bool {
	return nonEmptySmartSuiteString(updates.VisualStyle) || nonEmptySmartSuiteString(updates.StyleRequirements)
}

func smartSuiteHasOtherRequirementUpdate(updates smartSuiteCoordinatorUpdates) bool {
	return nonEmptySmartSuiteString(updates.Platform) || nonEmptySmartSuiteString(updates.PlatformName) ||
		nonEmptySmartSuiteString(updates.CountryCode) || nonEmptySmartSuiteString(updates.CountryName) ||
		nonEmptySmartSuiteString(updates.Language) || nonEmptySmartSuiteString(updates.LanguageName) ||
		nonEmptySmartSuiteString(updates.OutputTarget) || updates.SecondaryCount != nil ||
		updates.DesktopAPlusCount != nil || updates.MobileAPlusCount != nil || nonEmptySmartSuiteString(updates.DrawingModel)
}

func clearSmartSuiteRequirementUpdates(updates *smartSuiteCoordinatorUpdates) {
	*updates = smartSuiteCoordinatorUpdates{}
}

func smartSuiteEvidenceSupported(req smartSuiteConversationTurnRequest, evidence []smartSuiteCoordinatorEvidence, kind string) bool {
	text := strings.TrimSpace(req.Text)
	normalizedText := normalizeSmartSuiteEvidenceText(text)
	imagePurpose := strings.ToLower(strings.TrimSpace(req.ImagePurpose))
	for _, item := range evidence {
		quote := strings.TrimSpace(item.Quote)
		switch strings.ToLower(strings.TrimSpace(item.Source)) {
		case "user_text":
			normalizedQuote := normalizeSmartSuiteEvidenceText(quote)
			if normalizedQuote != "" && strings.Contains(normalizedText, normalizedQuote) {
				return true
			}
		case "image":
			if strings.TrimSpace(req.ImageURL) == "" || quote == "" {
				continue
			}
			switch kind {
			case "product":
				if imagePurpose != "style_reference" {
					return true
				}
			case "visual":
				if imagePurpose == "style_reference" {
					return true
				}
			}
		}
	}
	return false
}

func normalizeSmartSuiteEvidenceText(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, value)
}

func stringPtr(value string) *string { return &value }
func intPtr(value int) *int          { return &value }

func runSmartSuiteConversationCoordinator(conversation model.SmartSuiteConversation, req smartSuiteConversationTurnRequest) (smartSuiteCoordinatorOutput, error) {
	apiKey := strings.TrimSpace(getSettingValue("chat_key"))
	modelName := getSmartSuiteAgentModel("coordinator")
	if apiKey == "" || modelName == "" {
		return smartSuiteCoordinatorOutput{}, fmt.Errorf("chat_coordinator_not_configured")
	}
	contextJSON, _ := json.Marshal(conversation.Context)
	messageJSON, _ := json.Marshal(smartSuiteRecentMessages(conversation.Messages, 24))
	imageCatalog := smartSuiteConversationImageCatalog(conversation)
	drawingModelOptions, _ := listSmartSuiteDrawingModelOptions()
	drawingModelsJSON, _ := json.Marshal(drawingModelOptions)
	system := `You are the requirement-understanding agent inside a production ecommerce image-suite Agent team. Understand meaning from the full conversation and current context; do not classify by isolated keywords. Return exactly one JSON object and no prose.

Decide requirement_kind for the latest user turn:
- product_facts: statements about the actual SKU, including identity, material, construction, dimensions, color, function, compatibility, included parts, supported results, and where the product itself is used.
- visual_direction: instructions for how the generated images should look, including generated setting, background treatment, lighting, mood, composition, photographic or design style.
- mixed: the latest turn contains updates from at least two groups: product facts, visual direction, or other requirements. Examples include product facts plus platform/counts, visual direction plus language/model choice, or all three groups.
- other: platform, site, output count, language, model choice, workflow command, complaint, or unrelated content.
- ambiguous: the meaning cannot be distinguished safely from context.

The requirement_kind must match the fields actually returned in updates: product fields only = product_facts; visual fields only = visual_direction; platform/site/count/language/model fields only = other; fields from two or more groups = mixed. For example, "Amazon 美国站" must be other when updates contain only platform and market fields.

Important semantic distinctions:
- A product can be "for a bathroom" as a product fact, while "show it in a classic American bathroom" is a visual direction. Decide from the user's communicative intent and conversation state, not from words such as bathroom, background, material, wood, modern, or style.
- Product copy pasted with a product image remains product facts even if it contains appearance or usage-scene language.
- A later turn can supplement or correct earlier product facts. For product_features, preserve current_context.product_features verbatim and append supported new facts; never drop or rewrite prior facts unless the user explicitly corrects them.
- All fields remain editable after a suite has completed. Never ignore a requested update merely because current_missing_fields is empty or the field already has a value. Treat phrases such as 改成, 换成, 不要, 增加, 减少, 删除, 之前的, change to, replace, remove, add and reduce as revision signals. Use clear_fields for the prior semantic field when replacement or removal is explicit.
- A turn can both revise requirements and request generation, for example "把场景改成美式厨房，然后重新生成". In that case return the supported updates and intent="start" in the same response. A standalone "重新生成" starts with the unchanged current context.
- If the user only says they want to modify or adjust requirements but gives no concrete change, keep all current fields unchanged and ask what should change. Do not fail and do not start a task.
- Do not infer a desired visual style from the white-background product image. Image observations can support product identity and directly visible facts only.
- Final smart-suite artwork must use a contextual scene background. If the user asks for a pure white background, flat solid-color studio backdrop, blank panel or placeholder-style layout, do not store that request as visual_style. Set requirement_kind="ambiguous", needs_clarification=true, omit all updates, explain that final suite images require a scene-based background, and ask for the desired scene direction.
- Never invent a claim, platform, site, count, language, brand, product fact, or visual request.
- The standard full suite means exactly 6 secondary images, 7 desktop A+ modules, and 7 mobile A+ modules. Resolve a semantically clear request for the full suite or all three deliverables to secondary_count=6, desktop_a_plus_count=7, mobile_a_plus_count=7. When the user explicitly asks for only a subset, set every excluded deliverable count to 0. Do not infer output selection from unrelated uses of words such as all, complete, or suite.

Evidence is mandatory for every extracted update:
- product_evidence, visual_evidence, and other_evidence are arrays of {"source":"user_text"|"image","quote":"..."}.
- For user_text, quote must be an exact non-empty substring of latest_text that supports the classification.
- For image, quote must be a concise non-empty description of what is directly visible.
- latest_image_purpose="product_image" means the image can support only product facts; latest_image_purpose="style_reference" means it can support only visual direction. A style reference is not evidence about the SKU, and a product image is not evidence of a requested visual style.
- Product updates require product_evidence. Visual updates require visual_evidence.
- Platform, site, output count, language, and drawing model updates require other_evidence quoting the exact latest_text.
- confidence is 0 to 1. Below 0.65, use requirement_kind="ambiguous", needs_clarification=true, omit all updates, and ask one concise clarification_question. A concise answer to the latest assistant question is not ambiguous merely because it is short; use the previous question and current missing_fields as context.
- latest_answer_field is a conversational focus hint supplied for both typed replies and clicked suggestions. Use it to understand which currently requested field a concise answer addresses, while allowing explicit corrections or mixed requirements in latest_text to override that focus. product_info means the assistant is collecting product_name and product_features. Extract only facts supported by latest_text.

Use clear_fields only when the user explicitly corrects a prior semantic classification or asks to remove stored information. Allowed values: product_name, product_category, product_brand, product_features, product_proof, visual_style, style_requirements.

Valid intents: collect_requirement, start, stop, edit_image, accept_candidate, reject_candidate. Do not emit start unless latest_text explicitly asks to start or regenerate. When start is combined with requirement revisions, include every supported revision in updates before starting. For image edits, resolve references using image_catalog and target_image_id.

At the start of a new conversation, identify the requested deliverable. If the user asks for a suite, set request_type="suite"; if the user asks only for listing copy, set request_type="listing"; if they want both, set request_type="suite_and_listing". When the deliverable is clear but the product details are not yet collected, set open_requirement_form=true and return collect_requirement. Do not ask for every field in chat before opening the form.

Also return a concise public processing report for the UI:
- public_summary: one short sentence describing what you concluded or what action you selected for this exact turn.
- public_actions: 1 to 4 concrete, user-visible actions you actually performed from the supplied context, latest message, image evidence, and returned updates.
- These fields are not private chain-of-thought. Do not expose hidden reasoning, scores, internal deliberation, policy text, or generic staged narration.
- Every public action must be specific to this turn. Mention the actual classified information, confirmed field, clarification, selected image, or next workflow action. Never return canned lines such as "reading context", "analyzing requirements", or "deciding next step".

updates may contain product_name, product_category, product_brand, product_features, product_proof, platform, platform_name, country_code, country_name, language, language_name, secondary_count, desktop_a_plus_count, mobile_a_plus_count, visual_style, style_requirements, drawing_model. Omit unknown and unchanged fields, except product_features must be the complete merged document when adding or correcting product facts. Use platform ids amazon, walmart, home_depot, wayfair, lowes, shopify, temu; language codes en, zh, de, fr, es, it, ja or ko; country codes such as US, CA, GB, DE, FR, JP, AU, EU. The platform and site determine the default copy language; do not ask the user for language unless they explicitly request an override. If the user chooses a drawing model in free text, drawing_model must exactly equal one name from available_drawing_models; omit it when no listed model clearly matches.

Required top-level shape:
{"intent":"collect_requirement","request_type":"suite|listing|suite_and_listing","open_requirement_form":false,"requirement_kind":"product_facts|visual_direction|mixed|other|ambiguous","confidence":0.0,"product_evidence":[],"visual_evidence":[],"other_evidence":[],"needs_clarification":false,"clarification_question":"","clear_fields":[],"target_image_id":0,"public_summary":"turn-specific public summary","public_actions":["turn-specific completed action"],"updates":{}}`
	skillSystem, skillErr := smartSuiteAgentPrompt(smartagent.UnderstandSuiteRequirements, system)
	if skillErr != nil {
		return smartSuiteCoordinatorOutput{}, skillErr
	}
	system = skillSystem
	missingFieldsJSON, _ := json.Marshal(smartSuiteMissingFields(conversation.Context))
	user := fmt.Sprintf("current_context=%s\nrecent_messages=%s\ncurrent_missing_fields=%s\ncurrent_task_id=%q\nprior_task_ids=%q\nimage_catalog=%s\navailable_drawing_models=%s\nrequested_request_type=%q\nlatest_image_url=%q\nlatest_image_purpose=%q\nlatest_answer_field=%q\nselected_target_image_id=%d\nselected_image_has_candidate=%t\nlatest_text=%q", contextJSON, messageJSON, missingFieldsJSON, conversation.TaskID, []string(conversation.TaskIDs), imageCatalog, drawingModelsJSON, req.RequestType, req.ImageURL, req.ImagePurpose, req.AnswerField, req.TargetImageID, req.HasCandidate, req.Text)
	startedAt := time.Now()
	var raw string
	var promptTokens, completionTokens, totalTokens int
	var callErr error
	if strings.TrimSpace(req.ImageURL) != "" {
		if data, mime, err := fetchSceneReferenceImageData(req.ImageURL); err == nil && data != "" {
			var usage *adapter.ResponsesUsage
			raw, usage, _, callErr = adapter.NewChatAdapter(apiKey, getSettingValue("access_url")).AnalyzeImagesWithVision(modelName, []adapter.VisionImageItem{{Data: data, MimeType: mime}}, system, user, 2500)
			if usage != nil {
				promptTokens, completionTokens, totalTokens = usage.InputTokens, usage.OutputTokens, usage.TotalTokens
			}
		} else {
			callErr = err
		}
	} else if isResponsesAPI(modelName) {
		var response *adapter.ResponseResult
		response, callErr = adapter.NewResponseAdapter(apiKey, getSettingValue("access_url")).Respond(adapter.ResponseRequest{Model: modelName, Input: []adapter.ResponseInput{{Role: "system", Content: system}, {Role: "user", Content: user}}, MaxOutputTokens: 2500})
		if callErr == nil && len(response.Choices) > 0 {
			raw = response.Choices[0].Message.Content
		}
		if response != nil {
			promptTokens, completionTokens, totalTokens = response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens
		}
	} else {
		var response *adapter.ChatResponse
		response, callErr = adapter.NewChatAdapter(apiKey, getSettingValue("access_url")).Chat(adapter.ChatRequest{Model: modelName, Messages: []adapter.ChatMessage{{Role: "system", Content: system}, {Role: "user", Content: user}}, MaxTokens: 2500, Temperature: 0.1})
		if callErr == nil && len(response.Choices) > 0 {
			raw = response.Choices[0].Message.Content
		}
		if response != nil {
			promptTokens, completionTokens, totalTokens = response.Usage.PromptTokens, response.Usage.CompletionTokens, response.Usage.TotalTokens
		}
	}
	status := "success"
	if callErr != nil {
		status = "failed"
	}
	database.DB.Create(&model.UsageLog{UserID: uint(conversation.UserID), UsageType: "chat", Source: "smart_suite_conversation", Model: modelName, Prompt: adapter.TruncateRunes(user, 4000), Result: adapter.TruncateRunes(raw, 1000), PromptTokens: promptTokens, CompletionTokens: completionTokens, TotalTokens: totalTokens, Duration: int(time.Since(startedAt).Milliseconds()), Status: status, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	if callErr != nil {
		return smartSuiteCoordinatorOutput{}, callErr
	}
	clean := extractJSONObjectFromText(stripSmartSuiteReasoningBlocks(raw))
	if clean == "" {
		return smartSuiteCoordinatorOutput{}, fmt.Errorf("conversation_coordinator_invalid_json")
	}
	var output smartSuiteCoordinatorOutput
	if err := json.Unmarshal([]byte(clean), &output); err != nil {
		return smartSuiteCoordinatorOutput{}, fmt.Errorf("conversation_coordinator_invalid_json: %w", err)
	}
	if !smartSuiteValidIntent(output.Intent) {
		output.Intent = "collect_requirement"
	}
	return output, nil
}

func smartSuiteRecentMessages(messages []model.SmartSuiteConversationMessage, count int) []model.SmartSuiteConversationMessage {
	if len(messages) <= count {
		return messages
	}
	return messages[len(messages)-count:]
}

func smartSuiteConversationImageCatalog(conversation model.SmartSuiteConversation) string {
	if strings.TrimSpace(conversation.TaskID) == "" {
		return "[]"
	}
	var images []model.EcomSuiteImage
	if err := database.DB.Select("id", "image_type", "image_index", "title", "role", "candidate_status").Where("task_id = ?", conversation.TaskID).Order("image_type, image_index").Find(&images).Error; err != nil {
		return "[]"
	}
	data, _ := json.Marshal(images)
	return string(data)
}

func smartSuiteValidIntent(intent string) bool {
	switch intent {
	case "collect_requirement", "start", "stop", "edit_image", "accept_candidate", "reject_candidate":
		return true
	default:
		return false
	}
}
