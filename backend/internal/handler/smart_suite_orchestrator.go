package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/smartagent"
	"gorm.io/gorm"
)

const (
	smartSuiteOrchestratorRender   = "render-suite"
	smartSuiteOrchestratorFinish   = "complete-task"
	smartSuiteOrchestratorMaxTurns = 20
)

type smartSuiteOrchestratorDecision struct {
	NextAgent string `json:"next_agent"`
	Reason    string `json:"reason"`
}

type smartSuiteOrchestratorState struct {
	WorkflowType       string                         `json:"workflow_type"`
	IncludeListing     bool                           `json:"include_listing"`
	RequestedImages    int                            `json:"requested_images"`
	CompletedAgents    []string                       `json:"completed_agents"`
	AvailableAgents    []string                       `json:"available_agents"`
	FactsReady         bool                           `json:"facts_ready"`
	BuyerConcernsReady bool                           `json:"buyer_concerns_ready"`
	StyleReady         bool                           `json:"style_ready"`
	DirectionsReady    bool                           `json:"directions_ready"`
	CopyReady          bool                           `json:"copy_ready"`
	ImagePlansReady    bool                           `json:"image_plans_ready"`
	ListingReady       bool                           `json:"listing_ready"`
	AgentRuns          []model.EcomSuiteAgentRun      `json:"agent_runs,omitempty"`
	CreativeReview     *model.EcomSuiteCreativeReview `json:"creative_review,omitempty"`
	ImagePlans         []model.EcomSuiteImagePlan     `json:"image_plans,omitempty"`
}

func smartSuiteSelectNextAgent(available []string) (smartSuiteOrchestratorDecision, error) {
	if len(available) == 0 {
		return smartSuiteOrchestratorDecision{}, fmt.Errorf("smart_suite_orchestrator_no_available_agent")
	}
	return smartSuiteOrchestratorDecision{NextAgent: available[0], Reason: "Selected by backend dependency order."}, nil
}

type smartSuiteOrchestratorTaskSummary struct {
	WorkflowType    string `json:"workflow_type"`
	IncludeListing  bool   `json:"include_listing"`
	ProductName     string `json:"product_name"`
	ProductCategory string `json:"product_category"`
	Platform        string `json:"platform"`
	CountryCode     string `json:"country_code"`
	CountryName     string `json:"country_name"`
	SecondaryCount  int    `json:"secondary_count"`
	DesktopCount    int    `json:"desktop_a_plus_count"`
	MobileCount     int    `json:"mobile_a_plus_count"`
	ImageCount      int    `json:"image_count"`
}

func runSmartSuiteAgentTeam(taskID string) {
	ctx := smartSuiteContext(taskID)
	if result := database.DB.Model(&model.EcomSuiteTask{}).Where("task_id = ? AND status IN ?", taskID, []string{"pending", "planning"}).Updates(map[string]interface{}{"status": "planning", "updated_at": time.Now()}); result.Error != nil || result.RowsAffected == 0 {
		finishSmartSuiteTaskContext(taskID)
		return
	}
	for turn := 1; turn <= smartSuiteOrchestratorMaxTurns; turn++ {
		if ctx.Err() != nil || isSmartSuiteTaskCancelled(taskID) {
			finishSmartSuiteTaskContext(taskID)
			return
		}
		task, snapshot, err := loadSmartSuiteOrchestratorState(taskID)
		if err != nil {
			setEcomSuiteTaskFailed(taskID, "smart_suite_orchestrator_state_load_failed: "+err.Error())
			finishSmartSuiteTaskContext(taskID)
			return
		}
		if !isSmartSuiteActiveStatus(task.Status) {
			finishSmartSuiteTaskContext(taskID)
			return
		}
		available := smartSuiteOrchestratorAvailableAgents(task, snapshot)
		if len(available) == 0 {
			setEcomSuiteTaskFailed(taskID, "smart_suite_orchestrator_no_available_agent")
			finishSmartSuiteTaskContext(taskID)
			return
		}
		// The available list is already dependency-ordered by the backend. Do not
		// spend another chat completion asking a router to choose its first item;
		// the worker Agent owns any response repair and validation.
		decision, selectErr := smartSuiteSelectNextAgent(available)
		if selectErr != nil {
			setEcomSuiteTaskFailed(taskID, selectErr.Error())
			finishSmartSuiteTaskContext(taskID)
			return
		}
		fallback := false
		run := model.EcomSuiteAgentRun{Turn: turn, Agent: decision.NextAgent, Reason: decision.Reason, AvailableAgents: append([]string(nil), available...), Status: "running", StartedAt: time.Now()}
		snapshot.AgentRuns = append(snapshot.AgentRuns, run)
		if err := saveSmartSuiteOrchestratorStateWithAction(taskID, snapshot, smartSuiteOrchestratorStage(decision.NextAgent), smartSuiteAgentAction(decision.NextAgent), ""); err != nil {
			setEcomSuiteTaskFailed(taskID, "smart_suite_orchestrator_state_save_failed: "+err.Error())
			finishSmartSuiteTaskContext(taskID)
			return
		}

		last := &snapshot.AgentRuns[len(snapshot.AgentRuns)-1]
		dispatchErr, terminal := dispatchSmartSuiteOrchestratedAgent(ctx, task, &snapshot, decision.NextAgent)
		// Agent-specific calls own their bounded retries and schema repairs. A
		// second generic dispatch here duplicated those calls and hid the source
		// of failures in usage logs.
		completedAt := time.Now()
		last.CompletedAt = completedAt
		if dispatchErr != nil {
			last.Status = "failed"
			last.Error = dispatchErr.Error()
			log.Printf("[EcomSuite] orchestrator agent failed: taskID=%s agent=%s err=%v", taskID, decision.NextAgent, dispatchErr)
			setEcomSuiteTaskFailed(taskID, smartSuitePlanningFailureMessage("smart_suite_agent_failed", dispatchErr))
			_ = saveSmartSuiteOrchestratorState(taskID, snapshot, "", "")
			finishSmartSuiteTaskContext(taskID)
			return
		}
		if fallback {
			last.Status = "fallback"
		} else {
			last.Status = "success"
		}
		if err := saveSmartSuiteOrchestratorState(taskID, snapshot, "", task.BuyerQuestions); err != nil {
			setEcomSuiteTaskFailed(taskID, "smart_suite_orchestrator_state_save_failed: "+err.Error())
			finishSmartSuiteTaskContext(taskID)
			return
		}
		if terminal {
			if decision.NextAgent == smartSuiteOrchestratorRender {
				return
			}
			finishSmartSuiteTaskContext(taskID)
			return
		}
	}
	setEcomSuiteTaskFailed(taskID, "smart_suite_orchestrator_max_turns_exceeded")
	finishSmartSuiteTaskContext(taskID)
}

func loadSmartSuiteOrchestratorState(taskID string) (model.EcomSuiteTask, model.EcomSuitePlanSnapshot, error) {
	var task model.EcomSuiteTask
	if err := database.DB.Where("task_id = ?", taskID).Preload("Images").First(&task).Error; err != nil {
		return task, model.EcomSuitePlanSnapshot{}, err
	}
	snapshot := model.EcomSuitePlanSnapshot{}
	if task.PlanSnapshot != nil {
		snapshot = *task.PlanSnapshot
	}
	if task.WorkflowType == "smart_listing" && !smartSuiteProductFactMapComplete(snapshot.ProductFactMap) {
		snapshot.ProductFactMap = smartSuiteListingFactMapFromTask(task)
	}
	return task, snapshot, nil
}

func smartSuiteOrchestratorAvailableAgents(task model.EcomSuiteTask, snapshot model.EcomSuitePlanSnapshot) []string {
	// A failed listing is terminal for the task. Do not treat the error as a
	// completed listing, otherwise the orchestrator can incorrectly finish the
	// task while the UI reports that the listing agent failed.
	if (task.WorkflowType == "smart_listing" || task.IncludeListing) && strings.TrimSpace(snapshot.ListingError) != "" {
		return nil
	}
	if task.WorkflowType == "smart_listing" {
		if !snapshot.ListingKeywordsReady {
			return []string{string(smartagent.ResearchListingKeywords)}
		}
		if !smartSuiteOrchestratorListingReady(task, snapshot) {
			return []string{string(smartSuiteListingAgent(task.Platform))}
		}
		return []string{smartSuiteOrchestratorFinish}
	}
	available := make([]string, 0, 4)
	hasAPlus := task.DesktopAPlusCount > 0 || task.MobileAPlusCount > 0 || smartSuiteTaskHasImageType(task, "smart_desktop_a_plus") || smartSuiteTaskHasImageType(task, "smart_mobile_a_plus")
	factsReady := smartSuiteProductFactMapComplete(snapshot.ProductFactMap)
	styleReady := smartSuiteGlobalStyleComplete(snapshot.GlobalStyle) && len(snapshot.StyleMaster.Evidence) > 0
	concernsReady := smartSuiteBuyerConcernsComplete(snapshot.BuyerConcerns, snapshot.ProductFactMap, smartSuiteBuyerConcernCount(task))
	secondaryDirectionsReady := smartSuiteDirectionsReadyForTypes(task, snapshot, "smart_secondary")
	aPlusDirectionsReady := smartSuiteDirectionsReadyForTypes(task, snapshot, "smart_desktop_a_plus", "smart_mobile_a_plus")
	directionsReady := secondaryDirectionsReady && aPlusDirectionsReady
	copyReady := len(task.Images) > 0 && len(snapshot.Copywriting) == len(task.Images)
	plansReady := smartSuiteOrchestratorImagePlansReady(task, snapshot)
	listingReady := !task.IncludeListing || smartSuiteOrchestratorListingReady(task, snapshot)

	if !factsReady {
		available = append(available, string(smartagent.AnalyzeProductFacts))
	}
	if !styleReady {
		available = append(available, string(smartagent.LockSuiteStyle))
	}
	if factsReady && smartSuiteBuyerConcernCount(task) > 0 && !concernsReady {
		available = append(available, string(smartagent.MapBuyerConcerns))
	}
	hasSecondaryImages := task.SecondaryCount > 0 || smartSuiteTaskHasImageType(task, "smart_secondary")
	if factsReady && styleReady && concernsReady && hasSecondaryImages && !secondaryDirectionsReady {
		available = append(available, string(smartagent.DirectSecondarySuite))
	}
	if factsReady && styleReady && hasAPlus && concernsReady && !aPlusDirectionsReady {
		available = append(available, string(smartagent.DirectSuiteCreative))
	}
	if directionsReady && !copyReady {
		available = append(available, string(smartagent.WriteSuiteCopy))
	}
	if copyReady && !plansReady {
		available = append(available, string(smartagent.PlanSuiteImage))
	}
	if factsReady && concernsReady && task.IncludeListing && !listingReady {
		if !snapshot.ListingKeywordsReady {
			available = append(available, string(smartagent.ResearchListingKeywords))
		} else {
			available = append(available, string(smartSuiteListingAgent(task.Platform)))
		}
	}
	// Creative review is advisory metadata. It must not be a prerequisite for
	// rendering: deterministic plan/copy validation above is the blocking gate.
	if plansReady && listingReady {
		available = append(available, smartSuiteOrchestratorRender)
	}
	return available
}

func smartSuiteOrchestratorListingReady(task model.EcomSuiteTask, snapshot model.EcomSuitePlanSnapshot) bool {
	if snapshot.Listing == nil {
		return false
	}
	listing := *snapshot.Listing
	listing.Platform = normalizeListingPlatform(listing.Platform)
	if listing.Platform == "" {
		listing.Platform = normalizeListingPlatform(task.Platform)
	}
	return validateSmartSuiteListingPlatformStructure(listing) == nil
}

func smartSuiteListingTodoDetail(platform string) string {
	switch normalizeListingPlatform(platform) {
	case "home_depot":
		return "已完成商品标题、3 条重点卖点、6 条 Bullet Points 和商品介绍"
	case "amazon":
		return "已完成商品标题、5 条卖点、商品描述和搜索词"
	case "lowes":
		return "已完成 Title、Description 和 Bullets"
	case "wayfair":
		return "已完成商品标题、About This Product 和 Feature Bullets"
	case "tiktok":
		return "已完成商品标题、商品描述、属性和变体"
	default:
		return "已完成商品标题、商品描述和平台属性"
	}
}

func smartSuiteTaskHasImageType(task model.EcomSuiteTask, imageType string) bool {
	wanted := normalizeSmartSuiteImageType(imageType)
	for _, image := range task.Images {
		if normalizeSmartSuiteImageType(image.ImageType) == wanted {
			return true
		}
	}
	return false
}

func smartSuiteDirectionsReadyForTypes(task model.EcomSuiteTask, snapshot model.EcomSuitePlanSnapshot, imageTypes ...string) bool {
	wanted := make(map[string]bool, len(imageTypes))
	for _, imageType := range imageTypes {
		wanted[normalizeSmartSuiteImageType(imageType)] = true
	}
	expected := make(map[string]bool)
	for _, image := range task.Images {
		key := smartSuitePlanKey(image.ImageType, image.ImageIndex)
		if wanted[normalizeSmartSuiteImageType(image.ImageType)] {
			expected[key] = true
		}
	}
	if len(expected) == 0 {
		return true
	}
	for _, direction := range snapshot.CreativeDirections {
		delete(expected, smartSuitePlanKey(direction.ImageType, direction.ImageIndex))
	}
	return len(expected) == 0
}

func mergeSmartSuiteDirections(existing, additions []model.EcomSuiteCreativeDirection) []model.EcomSuiteCreativeDirection {
	byKey := make(map[string]model.EcomSuiteCreativeDirection, len(existing)+len(additions))
	for _, direction := range existing {
		byKey[smartSuitePlanKey(direction.ImageType, direction.ImageIndex)] = direction
	}
	for _, direction := range additions {
		byKey[smartSuitePlanKey(direction.ImageType, direction.ImageIndex)] = direction
	}
	result := make([]model.EcomSuiteCreativeDirection, 0, len(byKey))
	for _, direction := range byKey {
		result = append(result, direction)
	}
	sort.SliceStable(result, func(i, j int) bool {
		leftType, rightType := smartSuiteImageTypeOrder(result[i].ImageType), smartSuiteImageTypeOrder(result[j].ImageType)
		if leftType != rightType {
			return leftType < rightType
		}
		return result[i].ImageIndex < result[j].ImageIndex
	})
	return result
}

func mergeSmartSuitePlans(existing, additions []model.EcomSuiteImagePlan) []model.EcomSuiteImagePlan {
	byKey := make(map[string]model.EcomSuiteImagePlan, len(existing)+len(additions))
	for _, plan := range existing {
		byKey[smartSuitePlanKey(plan.ImageType, plan.ImageIndex)] = plan
	}
	for _, plan := range additions {
		byKey[smartSuitePlanKey(plan.ImageType, plan.ImageIndex)] = plan
	}
	result := make([]model.EcomSuiteImagePlan, 0, len(byKey))
	for _, plan := range byKey {
		result = append(result, plan)
	}
	sort.SliceStable(result, func(i, j int) bool {
		leftType, rightType := smartSuiteImageTypeOrder(result[i].ImageType), smartSuiteImageTypeOrder(result[j].ImageType)
		if leftType != rightType {
			return leftType < rightType
		}
		return result[i].ImageIndex < result[j].ImageIndex
	})
	return result
}

func smartSuiteImageTypeOrder(imageType string) int {
	switch normalizeSmartSuiteImageType(imageType) {
	case "smart_secondary":
		return 0
	case "smart_desktop_a_plus":
		return 1
	case "smart_mobile_a_plus":
		return 2
	default:
		return 3
	}
}

func smartSuiteOrchestratorImagePlansReady(task model.EcomSuiteTask, snapshot model.EcomSuitePlanSnapshot) bool {
	if len(task.Images) == 0 || len(snapshot.ImagePlans) != len(task.Images) {
		return false
	}
	byKey := make(map[string]model.EcomSuiteImagePlan, len(snapshot.ImagePlans))
	for _, plan := range snapshot.ImagePlans {
		byKey[smartSuitePlanKey(plan.ImageType, plan.ImageIndex)] = plan
	}
	for _, image := range task.Images {
		plan, ok := byKey[smartSuitePlanKey(image.ImageType, image.ImageIndex)]
		if !ok || smartSuitePlanningCandidateValidationError(plan) != nil {
			return false
		}
	}
	return true
}

func smartSuiteOrchestratorStateFor(task model.EcomSuiteTask, snapshot model.EcomSuitePlanSnapshot, available []string) smartSuiteOrchestratorState {
	completed := make([]string, 0, len(snapshot.AgentRuns))
	for _, run := range snapshot.AgentRuns {
		if run.Status == "success" || run.Status == "fallback" {
			completed = append(completed, run.Agent)
		}
	}
	return smartSuiteOrchestratorState{
		WorkflowType: task.WorkflowType, IncludeListing: task.IncludeListing,
		RequestedImages: len(task.Images), CompletedAgents: completed, AvailableAgents: available,
		FactsReady:         smartSuiteProductFactMapComplete(snapshot.ProductFactMap),
		BuyerConcernsReady: smartSuiteBuyerConcernsComplete(snapshot.BuyerConcerns, snapshot.ProductFactMap, smartSuiteBuyerConcernCount(task)),
		CreativeReview:     snapshot.CreativeReview,
		ImagePlans:         snapshot.ImagePlans,
		StyleReady:         smartSuiteGlobalStyleComplete(snapshot.GlobalStyle) && len(snapshot.StyleMaster.Evidence) > 0,
		DirectionsReady:    smartSuiteDirectionsReadyForTypes(task, snapshot, "smart_secondary", "smart_desktop_a_plus", "smart_mobile_a_plus"),
		CopyReady:          len(task.Images) > 0 && len(snapshot.Copywriting) == len(task.Images),
		ImagePlansReady:    smartSuiteOrchestratorImagePlansReady(task, snapshot),
		ListingReady:       !task.IncludeListing || smartSuiteOrchestratorListingReady(task, snapshot),
		AgentRuns:          append([]model.EcomSuiteAgentRun(nil), snapshot.AgentRuns...),
	}
}

func askSmartSuiteOrchestrator(ctx context.Context, task model.EcomSuiteTask, state smartSuiteOrchestratorState) (smartSuiteOrchestratorDecision, error) {
	modelName := getSmartSuiteAgentModel("orchestrator")
	if modelName == "" {
		return smartSuiteOrchestratorDecision{}, fmt.Errorf("orchestrator_model_not_configured")
	}
	apiKey := strings.TrimSpace(getSettingValue("chat_key"))
	if apiKey == "" {
		return smartSuiteOrchestratorDecision{}, fmt.Errorf("orchestrator_api_key_not_configured")
	}
	runtime := fmt.Sprintf("Return exactly one JSON object with next_agent and reason. next_agent must be one of: %s. The backend enforces dependencies and executes only the selected capability.", strings.Join(state.AvailableAgents, ", "))
	system, err := smartSuiteAgentPrompt(smartagent.OrchestrateSuite, runtime)
	if err != nil {
		return smartSuiteOrchestratorDecision{}, err
	}
	request := struct {
		Task  smartSuiteOrchestratorTaskSummary `json:"task"`
		State smartSuiteOrchestratorState       `json:"state"`
	}{Task: smartSuiteOrchestratorTaskSummary{
		WorkflowType: task.WorkflowType, IncludeListing: task.IncludeListing,
		ProductName: task.ProductName, ProductCategory: task.ProductCategory,
		Platform: task.Platform, CountryCode: task.CountryCode, CountryName: task.CountryName,
		SecondaryCount: task.SecondaryCount, DesktopCount: task.DesktopAPlusCount,
		MobileCount: task.MobileAPlusCount, ImageCount: len(task.Images),
	}, State: state}
	user, _ := json.Marshal(request)
	callCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	started := time.Now()
	var raw string
	var usage *adapter.ResponsesUsage
	var callErr error
	for attempt := 1; attempt <= smartSuiteModelRetryAttempts; attempt++ {
		raw, usage, _, callErr = adapter.NewChatAdapter(apiKey, getSettingValue("access_url")).AnalyzeImagesWithVisionContext(callCtx, modelName, nil, system, string(user), 1200)
		if callErr == nil {
			break
		}
		if attempt < smartSuiteModelRetryAttempts {
			if retryErr := waitForSmartSuiteModelRetry(callCtx); retryErr != nil {
				callErr = retryErr
				break
			}
		}
	}
	status := "success"
	resultText := raw
	if callErr != nil {
		status = "failed"
		resultText = callErr.Error() + ": " + raw
	}
	logItem := model.UsageLog{UserID: uint(task.UserID), UsageType: "chat", Source: "smart_suite_orchestrator", Model: modelName, Prompt: adapter.TruncateRunes(string(user), 5000), Result: adapter.TruncateRunes(resultText, 1200), Duration: int(time.Since(started).Milliseconds()), Status: status, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if usage != nil {
		logItem.PromptTokens, logItem.CompletionTokens, logItem.TotalTokens = usage.InputTokens, usage.OutputTokens, usage.TotalTokens
	}
	database.DB.Create(&logItem)
	if callErr != nil {
		return smartSuiteOrchestratorDecision{}, fmt.Errorf("orchestrator_call_failed: %w", callErr)
	}
	raw = extractJSONObjectFromText(stripSmartSuiteReasoningBlocks(raw))
	var decision smartSuiteOrchestratorDecision
	if err := json.Unmarshal([]byte(raw), &decision); err != nil {
		return decision, fmt.Errorf("orchestrator_invalid_json: %w", err)
	}
	decision.NextAgent = normalizeSmartSuiteOrchestratorAgent(decision.NextAgent)
	if decision.NextAgent == "" {
		return decision, fmt.Errorf("orchestrator_missing_next_agent")
	}
	return decision, nil
}

func normalizeSmartSuiteOrchestratorAgent(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	switch value {
	case "analyze-product-facts", "product-fact-analyst":
		return string(smartagent.AnalyzeProductFacts)
	case "map-buyer-concerns", "buyer-insight":
		return string(smartagent.MapBuyerConcerns)
	case "lock-suite-style", "style-master":
		return string(smartagent.LockSuiteStyle)
	case "direct-suite-creative", "creative-director":
		return string(smartagent.DirectSuiteCreative)
	case "direct-secondary-suite", "secondary-director":
		return string(smartagent.DirectSecondarySuite)
	case "write-suite-copy", "copywriter":
		return string(smartagent.WriteSuiteCopy)
	case "plan-suite-image", "single-image-planner":
		return string(smartagent.PlanSuiteImage)
	case "write-product-listing", "listing-writer":
		return string(smartagent.WriteProductListing)
	case "write-tiktok-listing", "tiktok-listing-writer":
		return string(smartagent.WriteTikTokListing)
	case "write-home-depot-listing", "home-depot-listing-writer":
		return string(smartagent.WriteHomeDepotListing)
	case "write-lowes-listing", "lowes-listing-writer":
		return string(smartagent.WriteLowesListing)
	case "write-wayfair-listing", "wayfair-listing-writer":
		return string(smartagent.WriteWayfairListing)
	case "write-amazon-listing", "amazon-listing-writer":
		return string(smartagent.WriteAmazonListing)
	case "research-listing-keywords", "listing-keyword-researcher", "keyword-researcher":
		return string(smartagent.ResearchListingKeywords)
	case "review-home-depot-listing":
		return string(smartagent.ReviewHomeDepotListing)
	case "review-lowes-listing":
		return string(smartagent.ReviewLowesListing)
	case "review-wayfair-listing":
		return string(smartagent.ReviewWayfairListing)
	case smartSuiteOrchestratorRender, smartSuiteOrchestratorFinish:
		return value
	default:
		return ""
	}
}

func smartSuiteOrchestratorStage(agent string) string {
	switch agent {
	case string(smartagent.EvaluateSuiteCreative):
		return "reviewing_creative"
	case string(smartagent.AnalyzeProductFacts):
		return "classifying_product_facts"
	case string(smartagent.MapBuyerConcerns):
		return "analyzing_buyers"
	case string(smartagent.LockSuiteStyle):
		return "locking_style"
	case string(smartagent.DirectSuiteCreative):
		return "directing_suite"
	case string(smartagent.DirectSecondarySuite):
		return "directing_secondary"
	case string(smartagent.WriteSuiteCopy), string(smartagent.PlanSuiteImage):
		return "confirming_suite_prompts"
	case string(smartagent.ResearchListingKeywords):
		return "researching_listing_keywords"
	case smartSuiteOrchestratorRender:
		return "generating"
	case smartSuiteOrchestratorFinish:
		return "listing_complete"
	default:
		if isSmartSuiteListingAgent(agent) {
			return "writing_listing"
		}
		if agent == string(smartagent.ReviewHomeDepotListing) || agent == string(smartagent.ReviewLowesListing) || agent == string(smartagent.ReviewWayfairListing) {
			return "reviewing_listing"
		}
		return "planning"
	}
}

func dispatchSmartSuiteOrchestratedAgent(ctx context.Context, task model.EcomSuiteTask, snapshot *model.EcomSuitePlanSnapshot, agent string) (error, bool) {
	switch agent {
	case string(smartagent.EvaluateSuiteCreative):
		return runSmartSuiteCreativeGate(ctx, task, snapshot), false
	case string(smartagent.AnalyzeProductFacts):
		factMap, err := analyzeSmartSuiteProductFacts(ctx, task)
		if err == nil {
			snapshot.ProductFactMap = factMap
		}
		return err, false
	case string(smartagent.MapBuyerConcerns):
		concerns, err := analyzeSmartSuiteBuyerConcerns(ctx, task, snapshot.ProductFactMap)
		if err == nil {
			snapshot.BuyerConcerns = concerns
		}
		return err, false
	case string(smartagent.LockSuiteStyle):
		style, err := analyzeSmartSuiteStyleMaster(ctx, task)
		if err == nil {
			snapshot.StyleMaster = style
			snapshot.GlobalStyle = style.GlobalStyle
		}
		return err, false
	case string(smartagent.DirectSuiteCreative):
		plans := make([]model.EcomSuiteImagePlan, 0, task.DesktopAPlusCount+task.MobileAPlusCount)
		for _, image := range task.Images {
			if image.ImageType == "smart_desktop_a_plus" || image.ImageType == "smart_mobile_a_plus" {
				plans = append(plans, buildSmartSuitePlan(task, image, nil))
			}
		}
		task.PlanSnapshot = snapshot
		directions, err := directSmartSuiteBlueprint(ctx, task, plans)
		if err == nil {
			plans = applySmartSuiteCreativeDirections(plans, directions)
			snapshot.CreativeDirections = mergeSmartSuiteDirections(snapshot.CreativeDirections, directions)
			snapshot.ImagePlans = mergeSmartSuitePlans(snapshot.ImagePlans, plans)
		}
		return err, false
	case string(smartagent.DirectSecondarySuite):
		plans := make([]model.EcomSuiteImagePlan, 0, task.SecondaryCount)
		for _, image := range task.Images {
			if image.ImageType == "smart_secondary" {
				plans = append(plans, buildSmartSuitePlan(task, image, nil))
			}
		}
		task.PlanSnapshot = snapshot
		directions, err := directSmartSuiteSecondaryBlueprint(ctx, task, plans)
		if err == nil {
			plans = applySmartSuiteCreativeDirections(plans, directions)
			snapshot.CreativeDirections = mergeSmartSuiteDirections(snapshot.CreativeDirections, directions)
			snapshot.ImagePlans = mergeSmartSuitePlans(snapshot.ImagePlans, plans)
		}
		return err, false
	case string(smartagent.WriteSuiteCopy):
		plannerTask := applySmartSuiteGlobalStyleToTask(task, snapshot.GlobalStyle)
		plannerTask.PlanSnapshot = snapshot
		copies, err := runSmartSuiteCopywriter(ctx, plannerTask, snapshot.ImagePlans)
		if err == nil {
			snapshot.Copywriting = copies
			snapshot.ImagePlans = applySmartSuiteCopywriting(snapshot.ImagePlans, copies)
		}
		return err, false
	case string(smartagent.ResearchListingKeywords):
		return runSmartSuiteListingKeywordResearch(ctx, task, snapshot), false
	case string(smartagent.PlanSuiteImage):
		plannerTask := applySmartSuiteGlobalStyleToTask(task, snapshot.GlobalStyle)
		plannerTask.PlanSnapshot = snapshot
		refined, count, err := refineSmartSuitePlansWithVision(plannerTask, snapshot.ImagePlans)
		if err != nil || count != len(snapshot.ImagePlans) || len(refined) != len(snapshot.ImagePlans) {
			if err == nil {
				err = fmt.Errorf("planning_agent_incomplete_plans: expected %d got %d", len(snapshot.ImagePlans), count)
			}
			return err, false
		}
		merged := make([]model.EcomSuiteImagePlan, len(snapshot.ImagePlans))
		for index := range snapshot.ImagePlans {
			merged[index] = mergeSmartSuiteRefinedPlan(task, task.Images[index], nil, snapshot.ImagePlans[index], refined[index])
			if strings.TrimSpace(merged[index].AssignedProductFacts) == "" || smartSuitePlanningCandidateValidationError(merged[index]) != nil {
				return fmt.Errorf("suite_prompt_confirmation_incomplete:%s#%d", merged[index].ImageType, merged[index].ImageIndex), false
			}
		}
		snapshot.ImagePlans = merged
		return nil, false
	case string(smartagent.WriteProductListing), string(smartagent.WriteTikTokListing), string(smartagent.WriteHomeDepotListing), string(smartagent.WriteLowesListing), string(smartagent.WriteWayfairListing), string(smartagent.WriteAmazonListing):
		listing, err := runSmartSuiteListingWriter(ctx, task, snapshot)
		// Preserve the returned draft for diagnostics, but never treat a Writer
		// error as a ready Listing. Evidence/reason failures must be repaired by
		// Writer before the independent Reviewer is scheduled.
		if strings.TrimSpace(listing.Platform) == "" {
			listing.Platform = normalizeListingPlatform(task.Platform)
		}
		if snapshot != nil && smartSuiteOrchestratorListingReady(task, func() model.EcomSuitePlanSnapshot {
			candidate := *snapshot
			candidate.Listing = &listing
			return candidate
		}()) {
			snapshot.Listing = &listing
		}
		if err != nil {
			// Keep a non-empty Writer draft available to the review/repair gate.
			// Metadata validation errors are repairable content errors, not a reason
			// to discard the generated Listing or finish without a review.
			if snapshot != nil && strings.TrimSpace(listing.Title) != "" && smartSuiteOrchestratorListingReady(task, func() model.EcomSuitePlanSnapshot {
				candidate := *snapshot
				candidate.Listing = &listing
				return candidate
			}()) {
				snapshot.Listing = &listing
				snapshot.ListingError = ""
				return nil, false
			}
			if snapshot != nil {
				snapshot.ListingError = err.Error()
			}
			return err, false
		}
		snapshot.Listing = &listing
		snapshot.ListingError = ""
		snapshot.ListingReview = nil
		snapshot.ListingReviewError = ""
		return nil, false
	case smartSuiteOrchestratorRender:
		return dispatchSmartSuiteRender(task, *snapshot), true
	case smartSuiteOrchestratorFinish:
		if (task.WorkflowType == "smart_listing" || task.IncludeListing) && !smartSuiteOrchestratorListingReady(task, *snapshot) {
			return fmt.Errorf("listing_incomplete_cannot_complete"), false
		}
		result := database.DB.Model(&model.EcomSuiteTask{}).Where("task_id = ?", task.TaskID).Updates(map[string]interface{}{"status": "success", "smart_stage": "listing_complete", "updated_at": time.Now()})
		return result.Error, true
	default:
		return fmt.Errorf("orchestrator_unknown_agent:%s", agent), false
	}
}

func dispatchSmartSuiteRender(task model.EcomSuiteTask, snapshot model.EcomSuitePlanSnapshot) error {
	// Do not block rendering on a probabilistic editorial review. The image
	// plans have already passed deterministic backend validation before this
	// dispatch; CreativeReview remains an optional audit record.
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.EcomSuiteTask{}).Where("task_id = ? AND status IN ?", task.TaskID, []string{"pending", "planning"}).Updates(map[string]interface{}{"status": "planning", "smart_stage": smartSuiteFirstStageName(task, "generating"), "buyer_questions": task.BuyerQuestions, "plan_snapshot": string(snapshotJSON), "updated_at": time.Now()})
		if result.Error != nil || result.RowsAffected == 0 {
			if result.Error != nil {
				return result.Error
			}
			return fmt.Errorf("orchestrator_render_task_not_active")
		}
		for _, plan := range snapshot.ImagePlans {
			var image model.EcomSuiteImage
			if err := tx.Where("task_id = ? AND image_type = ? AND image_index = ?", task.TaskID, plan.ImageType, plan.ImageIndex).First(&image).Error; err != nil {
				return err
			}
			if err := tx.Model(&image).Updates(smartSuitePlanningImageUpdates(plan)).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	go runEcomSuiteImageGeneration(task.TaskID, task.OutputRatio, snapshot.GlobalStyle)
	return nil
}

func saveSmartSuiteOrchestratorState(taskID string, snapshot model.EcomSuitePlanSnapshot, stage, buyerQuestions string) error {
	return saveSmartSuiteOrchestratorStateWithAction(taskID, snapshot, stage, "", buyerQuestions)
}

func saveSmartSuiteOrchestratorStateWithAction(taskID string, snapshot model.EcomSuitePlanSnapshot, stage, action, buyerQuestions string) error {
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	// An empty stage/action is intentional: it clears the previous Agent's
	// progress text after the Agent has completed. Leaving these columns
	// unchanged makes the UI show a stale "正在生成" state after Listing exists.
	updates := map[string]interface{}{
		"plan_snapshot":  string(snapshotJSON),
		"smart_stage":    strings.TrimSpace(stage),
		"current_action": strings.TrimSpace(action),
		"updated_at":     time.Now(),
	}
	if strings.TrimSpace(buyerQuestions) != "" {
		updates["buyer_questions"] = buyerQuestions
	}
	return database.DB.Model(&model.EcomSuiteTask{}).Where("task_id = ?", taskID).Updates(updates).Error
}

func smartSuiteAgentAction(agent string) string {
	switch agent {
	case string(smartagent.EvaluateSuiteCreative):
		return "创意验收 Agent 正在检查整套叙事、文案与画面脚本，并定向修正问题"
	case string(smartagent.OrchestrateSuite):
		return "主 Agent 正在规划智能套图流程和下一步 Agent"
	case string(smartagent.AnalyzeProductFacts):
		return "产品事实 Agent 正在整理图片与 Listing 可引用的产品信息"
	case string(smartagent.MapBuyerConcerns):
		return "买家顾虑 Agent 正在分析用户购买前需要确认的问题"
	case string(smartagent.LockSuiteStyle):
		return "视觉风格 Agent 正在锁定整套图片的统一风格"
	case string(smartagent.DirectSuiteCreative), string(smartagent.DirectSecondarySuite):
		return "创意指导 Agent 正在规划每张图的画面任务、卖点和构图"
	case string(smartagent.WriteSuiteCopy):
		return "图片文案 Agent 正在为每张图组织可核对的卖点文案"
	case string(smartagent.ResearchListingKeywords):
		return "Listing Keyword Agent 正在搜索类目高意图关键词并结合运营关键词"
	case string(smartagent.PlanSuiteImage):
		return "图片规划 Agent 正在确认每张图的产品事实、构图和负面约束"
	case string(smartagent.WriteProductListing), string(smartagent.WriteTikTokListing), string(smartagent.WriteHomeDepotListing), string(smartagent.WriteLowesListing), string(smartagent.WriteWayfairListing), string(smartagent.WriteAmazonListing):
		return "Listing Writer Agent 正在生成平台文案（联网搜索平台术语和字段依据）"
	case string(smartagent.ReviewHomeDepotListing), string(smartagent.ReviewLowesListing), string(smartagent.ReviewWayfairListing):
		return "Listing Review Agent 正在检查语法、事实边界和平台合规（联网核对平台规则）"
	case smartSuiteOrchestratorRender:
		return "生图 Agent 正在调用图像模型生成套图"
	case smartSuiteOrchestratorFinish:
		return "主 Agent 正在汇总结果并执行最终完整性检查"
	default:
		return "智能套图 Agent 正在处理当前任务"
	}
}

func smartSuiteOrchestratorErrorText(err error) string {
	if err == nil {
		return "invalid or unavailable orchestrator decision"
	}
	return err.Error()
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
