package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"strings"
	"testing"

	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/smartagent"
)

func decodeSmartSuiteGenerationContract(t *testing.T, prompt string) smartSuiteGenerationContract {
	t.Helper()
	var contract smartSuiteGenerationContract
	if err := json.Unmarshal([]byte(prompt), &contract); err != nil {
		t.Fatalf("generation prompt must be valid structured JSON: %v\nprompt=%s", err, prompt)
	}
	return contract
}

func TestStripSmartSuiteBannedClaims(t *testing.T) {
	cleaned := stripSmartSuiteBannedClaims("The best waterproof product with guaranteed all terrain performance")
	for _, banned := range []string{"best", "waterproof", "guaranteed", "all terrain"} {
		if strings.Contains(strings.ToLower(cleaned), banned) {
			t.Fatalf("expected %q to be removed from %q", banned, cleaned)
		}
	}
}

func TestSmartSuiteConversationListEntryUsesProductAndSkipsStartStatus(t *testing.T) {
	entry := smartSuiteConversationListEntry(model.SmartSuiteConversation{
		SessionID: "session-1",
		TaskID:    "task-1",
		Context:   model.SmartSuiteConversationContext{ProductName: "Marble Tissue Holder"},
		Messages: []model.SmartSuiteConversationMessage{
			{Role: "user", Text: "生成一套亚马逊图片"},
			{Role: "assistant", Text: "已确认产品与视觉方向。"},
			{Role: "assistant", Text: "信息完整，开始生成。"},
		},
	})

	if entry.Title != "Marble Tissue Holder" {
		t.Fatalf("unexpected conversation title %q", entry.Title)
	}
	if entry.Preview != "已确认产品与视觉方向。" {
		t.Fatalf("unexpected conversation preview %q", entry.Preview)
	}
	if entry.SessionID != "session-1" || entry.TaskID != "task-1" {
		t.Fatalf("unexpected conversation identity %#v", entry)
	}
}

func TestSmartSuiteRequestModesDriveMissingFieldsAndTodo(t *testing.T) {
	listingContext := model.SmartSuiteConversationContext{RequestType: smartSuiteRequestTypeListing, IncludeListing: true, ProductName: "Faucet", ProductFeatures: "Matte black finish with dimensions and weight included in this product information", Platform: "amazon", CountryCode: "US"}
	if missing := smartSuiteMissingFields(listingContext); len(missing) != 0 {
		t.Fatalf("listing-only context should only require fields present in the current form, got: %v", missing)
	}
	listingTask := model.EcomSuiteTask{WorkflowType: "smart_listing", Status: "success", PlanSnapshot: &model.EcomSuitePlanSnapshot{Listing: &model.EcomSuiteListing{Title: "Faucet", Bullets: []string{"1", "2", "3", "4", "5"}, Description: "Description"}}}
	items, _, completed, total := buildSmartSuiteBigTodoItems(listingTask, nil)
	if len(items) != 2 || completed != 1 || total != 2 || items[0].ID != "listing-keyword-research" || items[1].ID != "product-listing" {
		t.Fatalf("listing-only todo must contain keyword research then listing: items=%+v completed=%d total=%d", items, completed, total)
	}
	suiteTask := model.EcomSuiteTask{WorkflowType: "smart_suite", SecondaryCount: 1, IncludeListing: false}
	suiteItems, _, _, _ := buildSmartSuiteBigTodoItems(suiteTask, nil)
	for _, item := range suiteItems {
		if item.ID == "product-listing" {
			t.Fatal("suite-only task must not add a listing todo step")
		}
	}
}

func TestSmartSuiteListingContextDoesNotRequireStandaloneDimensions(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{RequestType: smartSuiteRequestTypeListing, IncludeListing: true, ProductName: "Faucet", ProductFeatures: "Product dimensions and weight are included in the product information text", Platform: "home_depot", CountryCode: "US"}
	for _, field := range smartSuiteMissingFields(ctx) {
		if field == "listing_source" {
			t.Fatal("listing_source must not be required after standalone dimensions were removed from the form")
		}
	}
}

func TestApplySmartSuiteInitialRequirementsPreservesEveryFormInput(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/faucet.png",
		DrawingModel:    "image-model",
	}
	initial := smartSuiteInitialRequirements{
		RequestType:       smartSuiteRequestTypeSuiteAndListing,
		ProductName:       "Widespread Faucet",
		ProductCategory:   "Bathroom Faucet",
		ProductBrand:      "FORIOUS",
		ProductFeatures:   "Matte black finish; 6-12 inch widespread installation",
		ProductProof:      "Includes metal pop-up drain",
		Platform:          "amazon",
		PlatformName:      "Amazon",
		CountryCode:       "DE",
		CountryName:       "Germany",
		SecondaryCount:    4,
		DesktopAPlusCount: 3,
		MobileAPlusCount:  2,
		VisualStyle:       "Premium modern bathroom",
		StyleRequirements: "Soft daylight and restrained typography",
	}

	if code, message := applySmartSuiteInitialRequirements(&ctx, initial); code != "" {
		t.Fatalf("form input should be accepted: code=%s message=%s", code, message)
	}
	checks := map[string]string{
		"product_name":       ctx.ProductName,
		"product_category":   ctx.ProductCategory,
		"product_brand":      ctx.ProductBrand,
		"product_features":   ctx.ProductFeatures,
		"product_proof":      ctx.ProductProof,
		"platform":           ctx.Platform,
		"platform_name":      ctx.PlatformName,
		"country_code":       ctx.CountryCode,
		"country_name":       ctx.CountryName,
		"visual_style":       ctx.VisualStyle,
		"style_requirements": ctx.StyleRequirements,
		"drawing_model":      ctx.DrawingModel,
		"white_bg_image_url": ctx.WhiteBgImageURL,
	}
	expected := map[string]string{
		"product_name":       initial.ProductName,
		"product_category":   initial.ProductCategory,
		"product_brand":      initial.ProductBrand,
		"product_features":   initial.ProductFeatures,
		"product_proof":      initial.ProductProof,
		"platform":           initial.Platform,
		"platform_name":      initial.PlatformName,
		"country_code":       initial.CountryCode,
		"country_name":       initial.CountryName,
		"visual_style":       initial.VisualStyle,
		"style_requirements": initial.StyleRequirements,
		"drawing_model":      "image-model",
		"white_bg_image_url": "/uploads/faucet.png",
	}
	for field, value := range checks {
		if value != expected[field] {
			t.Fatalf("form field %s was not preserved: got=%q want=%q", field, value, expected[field])
		}
	}
	if !ctx.IncludeListing || ctx.RequestType != smartSuiteRequestTypeSuiteAndListing {
		t.Fatalf("request mode was not preserved: %+v", ctx)
	}
	if ctx.SecondaryCount != 4 || ctx.DesktopAPlusCount != 3 || ctx.MobileAPlusCount != 2 || ctx.OutputTarget != "all" {
		t.Fatalf("output selection was not preserved: %+v", ctx)
	}
	if ctx.Language != "de" {
		t.Fatalf("site language should be derived from the submitted site, got %q", ctx.Language)
	}
}

func TestSmartSuiteOrchestratorChoosesFromPersistedState(t *testing.T) {
	task := model.EcomSuiteTask{WorkflowType: "smart_suite", SecondaryCount: 0, DesktopAPlusCount: 0, MobileAPlusCount: 0, Images: []model.EcomSuiteImage{{ImageType: "smart_secondary", ImageIndex: 1}}}
	initial := smartSuiteOrchestratorAvailableAgents(task, model.EcomSuitePlanSnapshot{})
	if !containsString(initial, string(smartagent.AnalyzeProductFacts)) || !containsString(initial, string(smartagent.LockSuiteStyle)) {
		t.Fatalf("initial state should expose independent fact and style agents: %v", initial)
	}

	factMap := model.EcomSuiteProductFactMap{ConfirmedFacts: []model.EcomSuiteProductFact{{ID: "F1", Statement: "Matte finish", EvidenceSource: "user_document", Evidence: "Matte finish"}}, Inferences: []model.EcomSuiteProductInference{}}
	style := defaultSmartSuiteGlobalStyle(task)
	snapshot := model.EcomSuitePlanSnapshot{ProductFactMap: factMap, GlobalStyle: style, StyleMaster: model.EcomSuiteStyleMaster{GlobalStyle: style, Evidence: []model.EcomSuiteStyleEvidence{{Source: "product_image", Detail: "visible product finish"}}}}
	readyForDirection := smartSuiteOrchestratorAvailableAgents(task, snapshot)
	if containsString(readyForDirection, string(smartagent.DirectSecondarySuite)) || !containsString(readyForDirection, string(smartagent.MapBuyerConcerns)) {
		t.Fatalf("secondary direction must wait for shared buyer concerns: %v", readyForDirection)
	}
	snapshot.BuyerConcerns = []model.EcomSuiteBuyerConcern{{ID: "C1", Priority: 1, Question: "What does the finish look like?", Reason: "Understand surface appearance", EvidenceFactIDs: []string{"F1"}, ProofMethod: "Raking-light finish detail", RecommendedImageTypes: []string{"smart_secondary"}}}
	if !containsString(smartSuiteOrchestratorAvailableAgents(task, snapshot), string(smartagent.DirectSecondarySuite)) {
		t.Fatal("valid shared concern map must unlock secondary direction")
	}
	aPlusTask := task
	aPlusTask.SecondaryCount = 0
	aPlusTask.DesktopAPlusCount = 2
	aPlusTask.Images = []model.EcomSuiteImage{{ImageType: "smart_desktop_a_plus", ImageIndex: 1}}
	aPlusSnapshot := snapshot
	aPlusSnapshot.BuyerConcerns = nil
	if !containsString(smartSuiteOrchestratorAvailableAgents(aPlusTask, aPlusSnapshot), string(smartagent.MapBuyerConcerns)) {
		t.Fatalf("A+ state should require buyer-concern mapping: %v", smartSuiteOrchestratorAvailableAgents(aPlusTask, aPlusSnapshot))
	}

	listingTask := model.EcomSuiteTask{WorkflowType: "smart_listing", Platform: "home_depot"}
	listingAgents := smartSuiteOrchestratorAvailableAgents(listingTask, model.EcomSuitePlanSnapshot{})
	if len(listingAgents) != 1 || listingAgents[0] != string(smartagent.ResearchListingKeywords) {
		t.Fatalf("listing-only state should dispatch keyword research first: %v", listingAgents)
	}
	listingAgents = smartSuiteOrchestratorAvailableAgents(listingTask, model.EcomSuitePlanSnapshot{ListingKeywordsReady: true})
	if len(listingAgents) != 1 || listingAgents[0] != string(smartagent.WriteHomeDepotListing) {
		t.Fatalf("researched listing state should dispatch listing writer: %v", listingAgents)
	}
	partialListing := model.EcomSuiteListing{Platform: "home_depot", Title: "Title"}
	listingAgents = smartSuiteOrchestratorAvailableAgents(listingTask, model.EcomSuitePlanSnapshot{Listing: &partialListing, ListingKeywordsReady: true})
	if len(listingAgents) != 1 || listingAgents[0] != string(smartagent.WriteHomeDepotListing) {
		t.Fatalf("partial listing-only state should return to the listing writer: %v", listingAgents)
	}
	legacyPartialListing := model.EcomSuiteListing{Platform: "Home Depot", Title: "Title"}
	if smartSuiteOrchestratorListingReady(listingTask, model.EcomSuitePlanSnapshot{Listing: &legacyPartialListing}) {
		t.Fatal("legacy Home Depot display name must not bypass platform validation")
	}
	completeListing := model.EcomSuiteListing{
		Platform: "home_depot", Title: "Title", SalientBullets: []string{"One", "Two", "Three"},
		BulletPoints: []string{"One", "Two", "Three", "Four", "Five", "Six"}, AboutThisProduct: "Sentence one. Sentence two. Sentence three.",
	}
	listingAgents = smartSuiteOrchestratorAvailableAgents(listingTask, model.EcomSuitePlanSnapshot{Listing: &completeListing, ListingKeywordsReady: true, ListingReview: &model.EcomSuiteListingReview{Platform: "home_depot", Passed: true}})
	if len(listingAgents) != 1 || listingAgents[0] != smartSuiteOrchestratorFinish {
		t.Fatalf("completed listing-only state should expose completion: %v", listingAgents)
	}
	if agents := smartSuiteOrchestratorAvailableAgents(listingTask, model.EcomSuitePlanSnapshot{ListingError: "listing validation failed"}); len(agents) != 0 {
		t.Fatalf("failed listing state must not expose completion agents: %v", agents)
	}
}

func TestSmartSuiteListingTodoRequiresPlatformCompleteFields(t *testing.T) {
	task := model.EcomSuiteTask{WorkflowType: "smart_listing", Platform: "home_depot", Status: "planning", SmartStage: "writing_listing"}
	partial := model.EcomSuiteListing{Platform: "home_depot", Title: "Title"}
	task.PlanSnapshot = &model.EcomSuitePlanSnapshot{Listing: &partial}
	items, _, completed, _ := buildSmartSuiteBigTodoItems(task, nil)
	if len(items) != 2 || items[1].Status == "done" || completed != 0 {
		t.Fatalf("partial Home Depot listing must not be marked complete: items=%+v completed=%d", items, completed)
	}
	complete := model.EcomSuiteListing{Platform: "home_depot", Title: "Title", SalientBullets: []string{"One", "Two", "Three"}, BulletPoints: []string{"A", "B", "C", "D", "E", "F"}, AboutThisProduct: "Overview"}
	task.PlanSnapshot.Listing = &complete
	items, _, completed, _ = buildSmartSuiteBigTodoItems(task, nil)
	if items[0].Status != "pending" || items[1].Status != "done" || completed != 1 || !strings.Contains(items[1].Detail, "3 条重点卖点") {
		t.Fatalf("complete Home Depot listing must use the platform-specific completion detail: items=%+v completed=%d", items, completed)
	}
}

func TestNormalizeSmartSuiteOrchestratorAgent(t *testing.T) {
	for input, expected := range map[string]string{
		"product_fact_analyst": "analyze-product-facts",
		"style_master":         "lock-suite-style",
		"secondary-director":   "direct-secondary-suite",
		"copywriter":           "write-suite-copy",
		"listing_writer":       "write-product-listing",
		"render_suite":         "render-suite",
	} {
		if actual := normalizeSmartSuiteOrchestratorAgent(input); actual != expected {
			t.Fatalf("normalize %q: got %q want %q", input, actual, expected)
		}
	}
}

func TestSmartSuiteMixedTaskRequiresIndependentSecondaryAndAPlusDirections(t *testing.T) {
	task := model.EcomSuiteTask{
		WorkflowType: "smart_suite", SecondaryCount: 1, DesktopAPlusCount: 1,
		Images: []model.EcomSuiteImage{
			{ImageType: "smart_secondary", ImageIndex: 1},
			{ImageType: "smart_desktop_a_plus", ImageIndex: 1},
		},
	}
	factMap := completeSmartSuiteTestFactMap()
	style := defaultSmartSuiteGlobalStyle(task)
	snapshot := model.EcomSuitePlanSnapshot{
		ProductFactMap: factMap,
		GlobalStyle:    style,
		StyleMaster:    model.EcomSuiteStyleMaster{GlobalStyle: style, Evidence: []model.EcomSuiteStyleEvidence{{Source: "product_image", Detail: "visible finish"}}},
		BuyerConcerns:  completeSmartSuiteTestConcerns()[:1],
	}
	agents := smartSuiteOrchestratorAvailableAgents(task, snapshot)
	if !containsString(agents, string(smartagent.DirectSecondarySuite)) || !containsString(agents, string(smartagent.DirectSuiteCreative)) {
		t.Fatalf("mixed task should expose both independent direction Agents: %v", agents)
	}
	snapshot.CreativeDirections = []model.EcomSuiteCreativeDirection{{ImageType: "smart_secondary", ImageIndex: 1}}
	agents = smartSuiteOrchestratorAvailableAgents(task, snapshot)
	if containsString(agents, string(smartagent.DirectSecondarySuite)) || !containsString(agents, string(smartagent.DirectSuiteCreative)) {
		t.Fatalf("completed secondary direction must be preserved while A+ remains available: %v", agents)
	}
}

func TestMergeSmartSuitePlansKeepsStableStageOrder(t *testing.T) {
	aPlus := []model.EcomSuiteImagePlan{{ImageType: "smart_mobile_a_plus", ImageIndex: 1}, {ImageType: "smart_desktop_a_plus", ImageIndex: 1}}
	secondary := []model.EcomSuiteImagePlan{{ImageType: "smart_secondary", ImageIndex: 2}, {ImageType: "smart_secondary", ImageIndex: 1}}
	merged := mergeSmartSuitePlans(aPlus, secondary)
	want := []string{"smart_secondary#1", "smart_secondary#2", "smart_desktop_a_plus#1", "smart_mobile_a_plus#1"}
	for index, plan := range merged {
		if got := smartSuitePlanKey(plan.ImageType, plan.ImageIndex); got != want[index] {
			t.Fatalf("unstable merged order at %d: got=%s want=%s plans=%+v", index, got, want[index], merged)
		}
	}
}

func TestSecondaryDirectionUsesFactsWithoutBuyerConcerns(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	task := model.EcomSuiteTask{PlanSnapshot: &model.EcomSuitePlanSnapshot{ProductFactMap: factMap}}
	plans := []model.EcomSuiteImagePlan{{ImageType: "smart_secondary", ImageIndex: 1, Role: "placeholder"}}
	candidates := []model.EcomSuiteCreativeDirection{{
		ImageType: "smart_secondary", ImageIndex: 1, BuyerQuestion: "Material detail coverage",
		PrimarySellingPoint: "Show the marble surface", AssignedFactIDs: []string{"F2"},
		VisualEvidence: "visible marble veining", ShotScale: "macro", SceneDirection: "A real bathroom vanity with soft side light",
	}}
	directions, err := reconcileSmartSuiteCreativeDirectionsWithMode(task, plans, candidates, false)
	if err != nil || len(directions) != 1 || directions[0].AssignedProductFacts != factMap.ConfirmedFacts[1].Statement || len(directions[0].BuyerConcernIDs) != 0 {
		t.Fatalf("secondary direction should resolve directly from confirmed facts: directions=%+v err=%v", directions, err)
	}
}

func TestNormalizeSmartSuiteAuditRequiresEveryCheck(t *testing.T) {
	output := normalizeSmartSuiteAudit(smartSuiteAuditOutput{
		Passed: true,
		Checks: []model.EcomSuiteAuditCheck{
			{Key: "product_consistency", Passed: true, Detail: "matches"},
		},
	})
	if output.Passed {
		t.Fatal("audit must fail when the review agent omits required checks")
	}
	if len(output.Checks) != 4 {
		t.Fatalf("expected 4 normalized checks, got %d", len(output.Checks))
	}
	if strings.TrimSpace(output.Feedback) == "" {
		t.Fatal("expected correction feedback for missing checks")
	}
}

func TestSmartSuiteGlobalStyleContractLocksAllVisualSystems(t *testing.T) {
	task := model.EcomSuiteTask{ProductBrand: "Anza", VisualStyle: "modern bathroom", StyleRequirements: "soft daylight"}
	style := defaultSmartSuiteGlobalStyle(task)
	for name, value := range map[string]string{
		"color":    style.ColorSystem,
		"font":     style.FontSystem,
		"label":    style.LabelSystem,
		"lighting": style.Lighting,
		"quality":  style.TextureQuality,
	} {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("global style %s must not be empty: %+v", name, style)
		}
	}
	prompt := smartSuiteGlobalStylePrompt(style)
	for _, required := range []string{"Font system:", "Label system:", "Lighting:", "Texture and quality:"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("global style prompt missing %q: %s", required, prompt)
		}
	}
}

func TestSmartSuiteDefaultStyleDoesNotPrescribeWarmNeutralTemplate(t *testing.T) {
	style := defaultSmartSuiteGlobalStyle(model.EcomSuiteTask{ProductBrand: "Forious"})
	combined := strings.ToLower(strings.Join([]string{style.PrimaryColor, style.SecondaryColor, style.BackgroundTone, style.Lighting, style.ColorSystem}, " "))
	for _, oldDefault := range []string{
		"charcoal text, warm white and muted neutral support tones",
		"clean modern residential environment with warm-white neutral surfaces",
		"soft directional daylight with a consistent neutral-white temperature",
		"premium low-saturation deep navy blue",
	} {
		if strings.Contains(combined, oldDefault) {
			t.Fatalf("default style still prescribes the old recurring palette %q: %+v", oldDefault, style)
		}
	}
	if !strings.Contains(combined, "entire task") || !strings.Contains(combined, "no universal warm-white") {
		t.Fatalf("fallback must defer palette choice to the current task instead of selecting warm ivory: %+v", style)
	}
}

func TestSmartSuiteStyleVariationCueChangesAcrossNewTasks(t *testing.T) {
	first := model.EcomSuiteTask{TaskID: "suite-run-001", ProductName: "Kitchen faucet"}
	second := model.EcomSuiteTask{TaskID: "suite-run-002", ProductName: "Kitchen faucet"}
	firstCue := smartSuiteStyleVariationCue(first)
	if firstCue == smartSuiteStyleVariationCue(second) {
		t.Fatalf("new tasks for the same product need different palette tie-break cues: %q", firstCue)
	}
	if firstCue != smartSuiteStyleVariationCue(first) {
		t.Fatal("the style variation cue must remain stable inside one task")
	}
	instruction := smartSuiteStyleRouteInstruction(first)
	for _, required := range []string{firstCue, "at least three materially different", "should not automatically reuse warm ivory"} {
		if !strings.Contains(instruction, required) {
			t.Fatalf("style route instruction missing %q: %s", required, instruction)
		}
	}
}

func TestSmartSuitePaletteUsesCategoryLedSceneDirection(t *testing.T) {
	primary, secondary := ecomBrandPaletteColors("Forious")
	combined := strings.ToLower(primary + " " + secondary + " " + ecomBrandPaletteRule("Forious"))
	if strings.Contains(combined, "#123b63") || strings.Contains(combined, "deep navy-blue palette") || strings.Contains(combined, "premium low-saturation deep navy blue") {
		t.Fatalf("brand palette must not force the old blue route: %s", combined)
	}
	for _, required := range []string{"product category", "design language", "visible finish", "brand name alone"} {
		if !strings.Contains(combined, required) {
			t.Fatalf("palette rule missing category-led scene constraint %q: %s", required, combined)
		}
	}

	task := model.EcomSuiteTask{ProductName: "Matte black kitchen faucet", ProductFeatures: "Matte black stainless steel finish"}
	if strings.Contains(strings.ToLower(smartSuiteGlobalStylePrompt(defaultSmartSuiteGlobalStyle(task))), "deep navy") {
		t.Fatal("default smart-suite style must not prescribe deep navy")
	}
	stylePrompt := strings.ToLower(smartSuiteGlobalStylePrompt(defaultSmartSuiteGlobalStyle(task)))
	if !strings.Contains(stylePrompt, "category") || !strings.Contains(stylePrompt, "not make the product color the whole palette") {
		t.Fatalf("style prompt must make category-led scene palette explicit: %s", stylePrompt)
	}
}

func TestSmartSuitePremiumScenePropsDoNotDefaultToWarmMetal(t *testing.T) {
	rules := strings.ToLower(smartSuitePremiumScenePropRules)
	for _, required := range []string{"cheap-looking plastic clutter", "brass", "bronze", "copper", "rose gold", "background props", "uploaded product itself"} {
		if !strings.Contains(rules, required) {
			t.Fatalf("premium scene prop rule missing %q: %s", required, smartSuitePremiumScenePropRules)
		}
	}
}

func TestSmartSuiteFinalPromptsShareOneLockedFontSystem(t *testing.T) {
	style := defaultSmartSuiteGlobalStyle(model.EcomSuiteTask{})
	style.FontSystem = "Avenir Next is the single locked family; semibold headlines and regular supporting copy."
	task := model.EcomSuiteTask{
		WorkflowType: "smart_suite",
		PlanSnapshot: &model.EcomSuitePlanSnapshot{GlobalStyle: style},
	}
	secondary := completeSmartSuiteTestPlan("smart_secondary", 1, "Secondary title")
	secondary.TypographyInstruction = "Headline aligned left in no more than two lines."
	desktop := completeSmartSuiteTestPlan("smart_desktop_a_plus", 1, "Desktop title")
	desktop.TypographyInstruction = "Headline centered above the supporting sentence."

	secondaryContract := decodeSmartSuiteGenerationContract(t, buildSmartSuiteImageGenerationPromptForTask(task, secondary, ""))
	desktopContract := decodeSmartSuiteGenerationContract(t, buildSmartSuiteImageGenerationPromptForTask(task, desktop, ""))
	for name, contract := range map[string]smartSuiteGenerationContract{"secondary": secondaryContract, "desktop": desktopContract} {
		if contract.SuiteStyle.FontSystem != style.FontSystem {
			t.Fatalf("%s prompt lost the suite font system: %+v", name, contract.SuiteStyle)
		}
		if !strings.Contains(contract.TypographyConsistency, "immutable") || !strings.Contains(contract.ImagePlan.TypographyInstruction, style.FontSystem) {
			t.Fatalf("%s prompt must carry the hard suite font lock: %+v", name, contract)
		}
		if !strings.Contains(contract.ColorConsistencyRule, "task-level scene palette") || !strings.Contains(contract.ColorConsistencyRule, "allow each image_plan.scene to vary") {
			t.Fatalf("%s prompt must carry the hard suite color lock: %s", name, contract.ColorConsistencyRule)
		}
		if !strings.Contains(contract.ColorConsistencyRule, "product category") || !strings.Contains(contract.ColorConsistencyRule, "do not make the product color the whole palette") {
			t.Fatalf("%s prompt must derive scene color from category rather than product color: %s", name, contract.ColorConsistencyRule)
		}
		for _, item := range contract.ImagePlan.CopyItems {
			if item.Typography != contract.ImagePlan.TypographyInstruction {
				t.Fatalf("%s copy item escaped the shared font lock: %+v", name, item)
			}
		}
	}
	if secondaryContract.SuiteStyle.FontSystem != desktopContract.SuiteStyle.FontSystem {
		t.Fatalf("all images in one suite must use the same font system: secondary=%q desktop=%q", secondaryContract.SuiteStyle.FontSystem, desktopContract.SuiteStyle.FontSystem)
	}
}

func TestSmartSuiteAPlusGenerationContractRequiresCopyAndIsolatesStyleReferences(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_desktop_a_plus", 1, "A+ title")
	contract := decodeSmartSuiteGenerationContract(t, buildSmartSuiteImageGenerationPrompt(plan, ""))

	if len(contract.ImagePlan.CopyItems) != 2 || contract.ImagePlan.CopyItems[0].Text == "" || contract.ImagePlan.CopyItems[1].Text == "" {
		t.Fatalf("A+ generation must receive one non-empty title and supporting copy: %+v", contract.ImagePlan.CopyItems)
	}
	if !strings.Contains(contract.VisibleCopyRule, "invalid without visible copy") || !strings.Contains(contract.Instruction, "visible title and supporting copy are mandatory") {
		t.Fatalf("A+ generation contract must reject text-free output: %+v", contract)
	}
	for _, required := range []string{"style references only", "ignore and never reproduce", "confirmed_fact and copy_items are the only content authority"} {
		if !strings.Contains(contract.ReferenceUsageRule, required) {
			t.Fatalf("style-reference boundary missing %q: %s", required, contract.ReferenceUsageRule)
		}
	}
	constraints := strings.ToLower(strings.Join(contract.ImagePlan.NegativeConstraints, " "))
	for _, forbidden := range []string{"broken", "worn-out", "rusty", "dirty", "neglected", "scene relevance is a hard boundary", "lifestyle context", "perspective and scene coherence are hard constraints", "decorative or unrelated props", "never copy objects from style references", "brass, bronze, copper, rose gold or gold"} {
		if !strings.Contains(constraints, forbidden) {
			t.Fatalf("non-comparison scene quality constraint missing %q: %s", forbidden, constraints)
		}
	}
}

func TestSmartSuiteBeforeAfterContractLimitsDamageToSupportedBeforePanel(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Comparison title")
	plan.LayoutSpec.ComparisonMode = "before_after"
	contract := decodeSmartSuiteGenerationContract(t, buildSmartSuiteImageGenerationPrompt(plan, ""))
	constraints := strings.Join(contract.ImagePlan.NegativeConstraints, " ")
	if !strings.Contains(constraints, "only in the explicitly labeled before panel") || !strings.Contains(constraints, "supported by the assigned confirmed fact") {
		t.Fatalf("before/after exception must remain fact-grounded and panel-scoped: %s", constraints)
	}
}

func TestSmartSuiteImageDimensionIssueChecksAspectRatio(t *testing.T) {
	canvas := image.NewRGBA(image.Rect(0, 0, 2, 1))
	canvas.Set(0, 0, color.White)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, canvas); err != nil {
		t.Fatalf("encode test image: %v", err)
	}
	data := base64.StdEncoding.EncodeToString(encoded.Bytes())
	if issue := smartSuiteImageDimensionIssue(data, "1:1"); issue == "" {
		t.Fatal("expected aspect-ratio mismatch to fail")
	}
	if issue := smartSuiteImageDimensionIssue(data, "2:1"); issue != "" {
		t.Fatalf("expected matching aspect ratio, got %s", issue)
	}
}

func TestSmartSuiteCopywriterOutputCannotBeRewrittenByVisualPlanner(t *testing.T) {
	task := model.EcomSuiteTask{WorkflowType: "smart_suite"}
	img := model.EcomSuiteImage{ImageType: "smart_desktop_a_plus", ImageIndex: 1}
	base := completeSmartSuiteTestPlan(img.ImageType, img.ImageIndex, "See the Product in Context")
	generic := completeSmartSuiteTestPlan(img.ImageType, img.ImageIndex, "Made for the Job")
	generic.Subtitle = "A clear introduction to the product and its core purpose"
	merged := mergeSmartSuiteRefinedPlan(task, img, nil, base, generic)
	if merged.Title != base.Title || merged.Subtitle != base.Subtitle {
		t.Fatalf("generic copy must not replace the base plan: base=%+v merged=%+v", base, merged)
	}
	refined := completeSmartSuiteTestPlan(img.ImageType, img.ImageIndex, "See the Mechanism at Work")
	refined.Subtitle = "Follow the working structure from action to result"
	merged = mergeSmartSuiteRefinedPlan(task, img, nil, base, refined)
	if merged.Title != base.Title || merged.Subtitle != base.Subtitle {
		t.Fatalf("visual planner must not replace copywriter output: base=%+v merged=%+v", base, merged)
	}
}

func TestBuildSmartSuiteFinalPromptContainsOnlySingleImagePlan(t *testing.T) {
	prompt := buildEcomSuiteFinalPromptForTask(model.EcomSuiteTask{
		WorkflowType:    "smart_suite",
		ProductName:     "Pull-down faucet",
		ProductCategory: "Kitchen faucet",
		ProductBrand:    "FORIOUS",
		ProductFeatures: "dual spray mode",
		ProductProof:    "Two spray controls are visible in the supplied reference.",
		Platform:        "walmart",
		CountryCode:     "CA",
		CountryName:     "Canada",
		Language:        "en",
	}, model.EcomSuiteImagePlan{
		ImageType:            "smart_secondary",
		ImageIndex:           1,
		Role:                 "How does it work?",
		AssignedFactIDs:      []string{"F1"},
		AssignedProductFacts: "The faucet provides two selectable spray modes.",
		CTADirection:         "Prove the mechanism in one glance",
		Title:                "See How It Works",
		Subtitle:             "Two selectable spray modes support different sink tasks.",
		MustIncludeText:      "Visible structure",
		Style:                "A real modern bathroom with warm daylight",
		VisualStrategy:       "Show the working structure through a close real-use action",
		Prompt:               "Create the finished composition with readable English copy.",
	}, ecomSuiteBrandAsset{})

	contract := decodeSmartSuiteGenerationContract(t, prompt)
	if contract.Platform != "walmart" || contract.Marketplace != "CA" || contract.TextLanguage != "English" {
		t.Fatalf("unexpected market contract: %+v", contract)
	}
	if contract.Operation != "generate_new_image" || contract.RevisionRequest != "" || contract.RevisionPriorityRule != "" {
		t.Fatalf("normal image generation must not carry edit semantics: %+v", contract)
	}
	if contract.ProductProfile.ProductName != "Pull-down faucet" || contract.ProductProfile.Category != "Kitchen faucet" || contract.ProductProfile.Brand != "FORIOUS" {
		t.Fatalf("product identity must reach the final prompt: %+v", contract.ProductProfile)
	}
	if contract.ConfirmedFact.ID != "F1" || contract.ConfirmedFact.Statement != "The faucet provides two selectable spray modes." {
		t.Fatalf("exactly one assigned fact must reach the final prompt: %+v", contract.ConfirmedFact)
	}
	if contract.ImagePlan.ImageType != "smart_secondary" || contract.ImagePlan.ImageIndex != 1 || contract.ImagePlan.ImageGoal != "How does it work?" {
		t.Fatalf("unexpected single-image plan: %+v", contract.ImagePlan)
	}
	wantCopy := []string{"See How It Works", "Two selectable spray modes support different sink tasks."}
	if len(contract.ImagePlan.CopyItems) != len(wantCopy) {
		t.Fatalf("expected %d structured copy items, got %+v", len(wantCopy), contract.ImagePlan.CopyItems)
	}
	for i, want := range wantCopy {
		if contract.ImagePlan.CopyItems[i].Text != want {
			t.Fatalf("copy item %d: expected %q, got %+v", i, want, contract.ImagePlan.CopyItems[i])
		}
	}
	for _, leakedPrefix := range []string{"Headline: See How It Works", "Supporting copy: A clear view", "Label: Visible structure"} {
		if strings.Contains(prompt, leakedPrefix) {
			t.Fatalf("copy metadata prefix must not be formatted as visible text: %q", prompt)
		}
	}
	if strings.Contains(prompt, "Single-focus rule") {
		t.Fatalf("generation must not inject planning restrictions, got %q", prompt)
	}
	for _, unexpected := range []string{"SHARED TASK CONTEXT", "STAGE CONTRACT", "FINAL ARTWORK REQUIREMENTS", "cross-border ecommerce detail-page strategist"} {
		if strings.Contains(prompt, unexpected) {
			t.Fatalf("structured generation prompt must not contain upstream agent instructions %q, got %q", unexpected, prompt)
		}
	}
	for _, leakedProductContext := range []string{"dual spray mode", "Two spray controls are visible", "Visible structure"} {
		if strings.Contains(prompt, leakedProductContext) {
			t.Fatalf("final generation prompt leaked full product context or a third label %q: %s", leakedProductContext, prompt)
		}
	}
}

func TestSmartSuiteFinalPromptRestoresAssignedFactsAndLayoutFromSnapshot(t *testing.T) {
	snapshotPlan := completeSmartSuiteTestPlan("smart_secondary", 2, "Material You Can See")
	snapshotPlan.AssignedFactIDs = []string{"F2"}
	snapshotPlan.AssignedProductFacts = "The supplied reference visibly shows a brushed metal finish."
	task := model.EcomSuiteTask{
		WorkflowType:    "smart_suite",
		ProductName:     "Kitchen faucet",
		ProductFeatures: "FULL_DOCUMENT_SECRET should stay upstream only.",
		ProductProof:    "FULL_PROOF_SECRET should stay upstream only.",
		PlanSnapshot: &model.EcomSuitePlanSnapshot{
			ProductFactMap: model.EcomSuiteProductFactMap{ConfirmedFacts: []model.EcomSuiteProductFact{
				{ID: "F2", Statement: "The faucet has a brushed metal finish.", EvidenceSource: "product_image", Evidence: "Brushed metal is visible on the faucet body."},
				{ID: "F9", Statement: "UNASSIGNED_FACT_SECRET must not reach this image.", EvidenceSource: "user_document", Evidence: "Unrelated document evidence."},
			}},
			ImagePlans: []model.EcomSuiteImagePlan{snapshotPlan},
		},
	}
	persistedPlan := snapshotPlan
	persistedPlan.AssignedFactIDs = nil
	persistedPlan.AssignedProductFacts = ""
	persistedPlan.LayoutSpec = model.EcomSuiteLayoutSpec{}

	contract := decodeSmartSuiteGenerationContract(t, buildSmartSuiteImageGenerationPromptForTask(task, persistedPlan, ""))
	if contract.ConfirmedFact.ID != "F2" || contract.ConfirmedFact.Statement != "The faucet has a brushed metal finish." {
		t.Fatalf("only the assigned snapshot fact must reach the final prompt: %+v", contract.ConfirmedFact)
	}
	for _, leaked := range []string{"FULL_DOCUMENT_SECRET", "FULL_PROOF_SECRET", "UNASSIGNED_FACT_SECRET"} {
		if strings.Contains(buildSmartSuiteImageGenerationPromptForTask(task, persistedPlan, ""), leaked) {
			t.Fatalf("final image prompt leaked upstream-only fact context %q", leaked)
		}
	}
	if contract.ImagePlan.LayoutSpec.CameraAngle != snapshotPlan.LayoutSpec.CameraAngle || contract.ImagePlan.LayoutSpec.ProductOccupancyPercent != snapshotPlan.LayoutSpec.ProductOccupancyPercent {
		t.Fatalf("structured layout must be restored from the planning snapshot: %+v", contract.ImagePlan.LayoutSpec)
	}
}

func TestSmartSuiteRevisionStaysInsideCompleteExecutionContract(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Headline")
	prompt := buildSmartSuiteImageGenerationPrompt(plan, "Reviewer-required correction:\n## Whole-suite context\nmake the headline readable.")
	contract := decodeSmartSuiteGenerationContract(t, prompt)
	if contract.Operation != "edit_existing_image" || contract.RevisionRequest != "Reviewer-required correction: ## Whole-suite context make the headline readable." {
		t.Fatalf("revision must be a top-level structured edit request, got %+v", contract)
	}
	if !strings.Contains(contract.RevisionPriorityRule, "overrides every conflicting") || !strings.Contains(contract.Instruction, "do not restart") {
		t.Fatalf("revision contract must prioritize the selected-image edit: %+v", contract)
	}
	if !strings.Contains(contract.MetadataVisibilityRule, "exact replacement wording requested by the user") {
		t.Fatalf("revision contract must allow explicit visible-copy replacement: %+v", contract)
	}
	if contract.ImagePlan.VisualRequirements != plan.Prompt {
		t.Fatalf("revision must preserve the original visual plan: %+v", contract.ImagePlan)
	}
	if contract.ImagePlan.ProductLock == "" || len(contract.ImagePlan.NegativeConstraints) == 0 {
		t.Fatalf("revision contract lost generation constraints: %+v", contract.ImagePlan)
	}
}

func TestSmartSuiteSelectedImageRegenerateTextStaysSingleImageEdit(t *testing.T) {
	conversation := &model.SmartSuiteConversation{Context: model.SmartSuiteConversationContext{VisualStyle: "premium ecommerce"}}
	for _, text := range []string{"重新生成这张图，背景改成庭院", "再生成一版，只调整标题位置", "开始生成这张图，但产品不要变"} {
		output := smartSuiteLocalCoordinator(conversation, smartSuiteConversationTurnRequest{Text: text, TargetImageID: 42})
		if output.Intent != "edit_image" || output.TargetImageID != 42 {
			t.Fatalf("selected-image instruction %q must stay on image 42, got %+v", text, output)
		}
	}
}

func TestSmartSuiteDirectImageEditPromptUsesOnlyCurrentImageInstruction(t *testing.T) {
	prompt := buildSmartSuiteDirectImageEditPrompt(model.EcomSuiteImage{ImageType: "smart_secondary", OutputRatio: "1:1"}, "背景改成深色石材，产品角度不变")
	for _, required := range []string{"direct single-image edit", "背景改成深色石材，产品角度不变", "not a request to revise, re-plan or regenerate the suite", "Preserve every element the user did not ask to change"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("direct image-edit prompt missing %q: %s", required, prompt)
		}
	}
	for _, forbidden := range []string{"suite_style", "confirmed_fact", "copy_items", "SHARED TASK CONTEXT"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("direct image-edit prompt leaked suite planning field %q: %s", forbidden, prompt)
		}
	}
}

func TestSmartSuiteReviewAgentSkillHasFourHardGatesWithoutSizeAudit(t *testing.T) {
	prompt := smartSuiteAgentPromptText(smartagent.ReviewSuiteImage, "runtime")
	for _, required := range []string{"Text", "Product integrity", "Scene logic", "Selling-point evidence", "Do not audit pixel dimensions or aspect ratio", "warm-metal background props", "revision_prompt", "this image only"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("review skill missing hard gate %q: %s", required, prompt)
		}
	}
}

func TestSmartSuiteGenerationOmitsNoLabelControlToken(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Headline")
	plan.MustIncludeText = "No labels"
	prompt := buildSmartSuiteImageGenerationPromptForLanguage(plan, "", "de")
	contract := decodeSmartSuiteGenerationContract(t, prompt)
	if strings.Contains(prompt, "No labels") || len(contract.ImagePlan.CopyItems) != 2 {
		t.Fatalf("no-label control value must not be sent as visible copy: %q", prompt)
	}
	if contract.TextLanguage != "German" {
		t.Fatalf("generation prompt must carry the target language: %+v", contract)
	}
}

func TestSmartSuiteGenerationDoesNotExposeTypographyMetadataAsVisibleCopy(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Reach Every Corner")
	plan.TypographyInstruction = "Use semibold 600 for the headline and regular 400 for supporting copy. Font weight 600. Keep an 8% safe margin."
	prompt := buildSmartSuiteImageGenerationPromptForLanguage(plan, "", "en")
	for _, leaked := range []string{"semibold 600", "regular 400", "Font weight 600"} {
		if strings.Contains(prompt, leaked) {
			t.Fatalf("typography metadata must not be exposed to rendering: leaked=%q prompt=%q", leaked, prompt)
		}
	}
	contract := decodeSmartSuiteGenerationContract(t, prompt)
	if !strings.Contains(contract.ImagePlan.TypographyInstruction, "Keep an 8% safe margin.") {
		t.Fatalf("sanitized typography instruction lost useful layout guidance: %+v", contract.ImagePlan)
	}
	for _, item := range contract.ImagePlan.CopyItems {
		if item.Typography != contract.ImagePlan.TypographyInstruction {
			t.Fatalf("copy item must use the same sanitized typography instruction: %+v", item)
		}
	}
}

func TestSmartSuiteGenerationKeepsLockedFontWeightWording(t *testing.T) {
	lockedFontSystem := "Helvetica Neue as the sole primary family; use 700 weight for short benefit headlines, 500 weight for supporting copy, and 400 weight for explanatory captions."
	task := model.EcomSuiteTask{WorkflowType: "smart_suite", Language: "en", PlanSnapshot: &model.EcomSuitePlanSnapshot{GlobalStyle: model.EcomSuiteGlobalStyle{FontSystem: lockedFontSystem}}}
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Reach Every Corner")
	plan.TypographyInstruction = "Place the headline in the upper-right safe area with a concise left-aligned hierarchy."
	prompt := buildSmartSuiteImageGenerationPromptForTask(task, plan, "")
	for _, required := range []string{"700 weight for short benefit headlines", "500 weight for supporting copy", "400 weight for explanatory captions"} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("locked font system weight roles must survive into the generation contract: missing=%q prompt=%q", required, prompt)
		}
	}
}

func TestSmartSuiteCopyLabelMustAddNewInformation(t *testing.T) {
	copy := model.EcomSuiteCopywriting{
		ImageType: "smart_secondary", ImageIndex: 1,
		Title: "Choose the Right Height", Subtitle: "The 15.27-inch profile helps confirm overhead cabinet clearance.",
		MustIncludeText: "15.27-inch profile",
	}
	if smartSuiteCopywritingCandidateUsable(copy, "smart_secondary") {
		t.Fatal("a label that repeats subtitle information must be rejected")
	}
	copy.MustIncludeText = "No labels"
	if !smartSuiteCopywritingCandidateUsable(copy, "smart_secondary") {
		t.Fatalf("the same copy should pass without a redundant label: %+v", copy)
	}
}

func TestSmartSuiteVisualPlannerRequiresLayoutAndDoesNotRepeatCopy(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Visible proof")
	if !smartSuitePlanningCandidateHasContent(plan) {
		t.Fatal("a complete visual plan with layout and typography should be accepted")
	}
	plan.Prompt += " Render Visible proof exactly."
	if smartSuitePlanningCandidateHasContent(plan) {
		t.Fatal("visual prompt must not repeat approved copy")
	}
}

func TestSmartSuiteAuditCandidateMustPassOrImprove(t *testing.T) {
	original := normalizeSmartSuiteAudit(smartSuiteAuditOutput{Passed: false, Checks: []model.EcomSuiteAuditCheck{
		{Key: "product_consistency", Passed: true},
	}})
	worse := normalizeSmartSuiteAudit(smartSuiteAuditOutput{Passed: false, Checks: nil})
	better := normalizeSmartSuiteAudit(smartSuiteAuditOutput{Passed: false, Checks: []model.EcomSuiteAuditCheck{
		{Key: "product_consistency", Passed: true},
		{Key: "text_consistency", Passed: true},
	}})
	passedChecks := make([]model.EcomSuiteAuditCheck, 0, 5)
	for _, key := range []string{"product_consistency", "text_consistency", "scene_logic", "visual_evidence", "dimension_consistency"} {
		passedChecks = append(passedChecks, model.EcomSuiteAuditCheck{Key: key, Passed: true})
	}
	passed := normalizeSmartSuiteAudit(smartSuiteAuditOutput{Passed: true, Checks: passedChecks})
	if smartSuiteAuditCandidateImproves(original, worse) {
		t.Fatal("a worse candidate must not replace the original")
	}
	if !smartSuiteAuditCandidateImproves(original, better) || !smartSuiteAuditCandidateImproves(original, passed) {
		t.Fatal("a better or passing candidate should be eligible for replacement")
	}
}

func TestSmartSuiteAuditRevisionPromptIsPreferredForImageToImageCorrection(t *testing.T) {
	output := normalizeSmartSuiteAudit(smartSuiteAuditOutput{
		Passed:         false,
		Feedback:       "审计摘要",
		RevisionPrompt: "Edit the current image in place: correct the vanishing point and remove the mismatched prop.",
		Checks: []model.EcomSuiteAuditCheck{
			{Key: "scene_logic", Passed: false, Detail: "inconsistent vanishing points"},
		},
	})
	if got := smartSuiteAuditRevisionPrompt(output); got != "Edit the current image in place: correct the vanishing point and remove the mismatched prop." {
		t.Fatalf("revision prompt must be preferred over audit summary: %q", got)
	}
	fallback := normalizeSmartSuiteAudit(smartSuiteAuditOutput{
		Passed:   false,
		Feedback: "Fix the scene perspective.",
		Checks:   []model.EcomSuiteAuditCheck{{Key: "scene_logic", Passed: false, Detail: "mismatched prop"}},
	})
	if got := smartSuiteAuditRevisionPrompt(fallback); got != "Fix the scene perspective." {
		t.Fatalf("missing revision prompt must fall back to feedback: %q", got)
	}
}

func TestSmartSuiteStageCorrectionFeedbackIsImageSpecific(t *testing.T) {
	output := smartSuiteStageAuditOutput{
		Feedback: "Fix every failed image using all listed corrections.",
		ImageCorrections: []smartSuiteImageCorrection{
			{ImageIndex: 1, Feedback: "Image 1 only: clarify the pull-out action.", RevisionPrompt: "Edit image 1 in place: clarify the pull-out action."},
			{ImageIndex: 2, Feedback: "Image 2 only: show the material close-up."},
		},
	}
	first, ok := smartSuiteStageCorrectionFeedback(output, 1)
	if !ok || first != "Edit image 1 in place: clarify the pull-out action." {
		t.Fatalf("unexpected image 1 correction: %q, %v", first, ok)
	}
	second, ok := smartSuiteStageCorrectionFeedback(output, 2)
	if !ok || second != "Image 2 only: show the material close-up." {
		t.Fatalf("unexpected image 2 correction: %q, %v", second, ok)
	}
	if _, ok := smartSuiteStageCorrectionFeedback(output, 3); ok {
		t.Fatal("an image without a specific correction must not use the global feedback")
	}
}

func TestSmartSuitePlanningRejectsMultipleSellingPointLabels(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Headline")
	plan.MustIncludeText = "First claim | Second claim"
	if smartSuiteImagePlanComplete(plan) {
		t.Fatal("planning must reject a single image with multiple selling-point labels")
	}
	plan.MustIncludeText = "No labels"
	if !smartSuiteImagePlanComplete(plan) {
		t.Fatal("planning must accept an explicit no-label decision")
	}
}

func TestSmartSuiteGenerationDoesNotMutatePlannedPrompt(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Headline")
	plan.Prompt = "Use the exact prompt prepared by planning."
	prompt := buildSmartSuiteImageGenerationPrompt(plan, "")
	contract := decodeSmartSuiteGenerationContract(t, prompt)
	if contract.ImagePlan.VisualRequirements != plan.Prompt {
		t.Fatalf("generation must preserve the planned visual prompt, got %+v", contract.ImagePlan)
	}
}

func TestSmartSuiteImagePlanRejectsEmbeddedSuiteContext(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Headline")
	plan.Prompt = "Create the image. Platform: desktop; Country: US; SKU: ABC-1"
	if smartSuiteImagePlanComplete(plan) {
		t.Fatal("single-image plan must reject embedded platform, country or SKU context")
	}
	plan.Prompt = "IMAGE GENERATION PROMPT: Create the image."
	if smartSuiteImagePlanComplete(plan) {
		t.Fatal("single-image plan must reject nested eight-field section headers")
	}
	plan.Prompt = "Create the complete finished commercial image in 21:9 for desktop A+."
	if smartSuiteImagePlanComplete(plan) {
		t.Fatal("single-image plan must keep platform ratio outside the generation prompt")
	}
}

func TestSmartSuiteImagePlanAllowsDetailPhotographyButRejectsPlaceholders(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Headline")
	plan.Style = "Studio photography of brushed metal texture under raking side light"
	if !smartSuiteImagePlanComplete(plan) {
		t.Fatal("material evidence must not require a room background")
	}
	plan.Style = "An empty background"
	if smartSuiteImagePlanComplete(plan) {
		t.Fatal("empty placeholder must remain invalid")
	}
}

func TestSmartSuitePlannerPersonaAndArtworkRules(t *testing.T) {
	if !strings.Contains(smartSuitePlannerPersona, "跨境电商详情页策划师") {
		t.Fatal("smart-suite planner must use the requested persona")
	}
	for _, expected := range []string{"complete finished commercial composition", "native ecommerce product-detail", "editorial web article", "dominant visual anchor", "material study or studio detail", "no blank panels", "fully readable"} {
		if !strings.Contains(smartSuiteFinalArtworkRules, expected) {
			t.Fatalf("artwork rules must contain %q", expected)
		}
	}
}

func TestBuildSmartSuitePlanUsesThreeDistinctStageContracts(t *testing.T) {
	task := model.EcomSuiteTask{VisualStyle: "real lifestyle", StyleRequirements: "natural light"}
	questions := []string{"Where can I use it?", "What is it made of?", "How does it work?", "Is it easy to use?", "Why choose it?", "What do I receive?"}
	secondary := buildSmartSuitePlan(task, model.EcomSuiteImage{ImageType: "smart_secondary", ImageIndex: 2}, questions)
	desktop := buildSmartSuitePlan(task, model.EcomSuiteImage{ImageType: "smart_desktop_a_plus", ImageIndex: 3}, questions)
	mobile := buildSmartSuitePlan(task, model.EcomSuiteImage{ImageType: "smart_mobile_a_plus", ImageIndex: 3}, questions)

	if secondary.Role != questions[1] || !strings.Contains(secondary.CTADirection, questions[1]) || strings.Contains(secondary.CTADirection, "all six") {
		t.Fatalf("secondary plan must answer its assigned purchase question: %+v", secondary)
	}
	if desktop.Role != "Core mechanism" || !strings.Contains(desktop.CTADirection, "Core mechanism") || strings.Contains(desktop.CTADirection, "all seven") || !strings.Contains(desktop.LayoutInstruction, "21:9") {
		t.Fatalf("desktop plan must advance the seven-step persuasion path: %+v", desktop)
	}
	if !strings.Contains(mobile.Role, "Mobile restatement") || !strings.Contains(mobile.LayoutInstruction, "never crop") || !strings.Contains(mobile.LayoutInstruction, "4:3") {
		t.Fatalf("mobile plan must independently recompose the same selling point: %+v", mobile)
	}
}

func TestSmartSuiteAssignedShotScaleReachesFinalCamera(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_desktop_a_plus", 4, "Material proof")
	plan.ShotScale = "macro"
	normalized := normalizeSmartSuitePlanningCandidate(plan)
	assigned := smartSuiteShotScaleInstruction("macro")
	if !strings.Contains(normalized.LayoutSpec.CameraAngle, assigned) {
		t.Fatalf("normalized camera must retain the creative-director shot scale: %q", normalized.LayoutSpec.CameraAngle)
	}
	contract := decodeSmartSuiteGenerationContract(t, buildSmartSuiteImageGenerationPrompt(normalized, ""))
	if contract.ImagePlan.ShotScale != "macro" || !strings.Contains(contract.ImagePlan.LayoutSpec.CameraAngle, assigned) {
		t.Fatalf("final generation contract must preserve the selected shot scale: %+v", contract.ImagePlan)
	}
}

func TestSmartSuiteShotRhythmIsFlexibleButNotFlat(t *testing.T) {
	directions := []model.EcomSuiteCreativeDirection{
		{ImageType: "smart_secondary", ImageIndex: 1, ShotScale: "medium", VisualEvidence: "real room context"},
		{ImageType: "smart_secondary", ImageIndex: 2, ShotScale: "medium", VisualEvidence: "daily use action"},
		{ImageType: "smart_secondary", ImageIndex: 3, ShotScale: "medium", VisualEvidence: "working valve mechanism"},
		{ImageType: "smart_secondary", ImageIndex: 4, ShotScale: "medium", VisualEvidence: "surface texture and finish"},
		{ImageType: "smart_secondary", ImageIndex: 5, ShotScale: "medium", VisualEvidence: "installation fit"},
	}
	balanced := rebalanceSmartSuiteShotRhythm(directions)
	seen := make(map[string]bool)
	for index, direction := range balanced {
		seen[direction.ShotScale] = true
		if index >= 2 && direction.ShotScale == balanced[index-1].ShotScale && direction.ShotScale == balanced[index-2].ShotScale {
			t.Fatalf("three consecutive modules must not collapse to one scale: %+v", balanced)
		}
	}
	if len(seen) < 3 {
		t.Fatalf("a five-image stage should retain at least three scales without a fixed index template: %+v", balanced)
	}
}

func TestSmartSuiteShotRhythmPreservesAnIntentionalNonTemplateSequence(t *testing.T) {
	directions := []model.EcomSuiteCreativeDirection{
		{ImageType: "smart_desktop_a_plus", ImageIndex: 1, ShotScale: "close_up"},
		{ImageType: "smart_desktop_a_plus", ImageIndex: 2, ShotScale: "wide"},
		{ImageType: "smart_desktop_a_plus", ImageIndex: 3, ShotScale: "medium"},
		{ImageType: "smart_desktop_a_plus", ImageIndex: 4, ShotScale: "close_up"},
		{ImageType: "smart_desktop_a_plus", ImageIndex: 5, ShotScale: "wide"},
	}
	balanced := rebalanceSmartSuiteShotRhythm(append([]model.EcomSuiteCreativeDirection(nil), directions...))
	for index := range directions {
		if balanced[index].ShotScale != directions[index].ShotScale {
			t.Fatalf("an already varied AI-selected rhythm must not be replaced by an index template: before=%+v after=%+v", directions, balanced)
		}
	}
}

func TestSmartSuitePlannerPromptUsesCreativeDirectorShotScale(t *testing.T) {
	task := model.EcomSuiteTask{Platform: "amazon", CountryName: "United States", Language: "English", VisualStyle: "real kitchen scene"}
	plans := []model.EcomSuiteImagePlan{
		completeSmartSuiteTestPlan("smart_secondary", 1, "Context"),
		completeSmartSuiteTestPlan("smart_secondary", 2, "Use"),
		completeSmartSuiteTestPlan("smart_secondary", 3, "Detail"),
	}
	plans[0].ShotScale = "close_up"
	plans[1].ShotScale = "wide"
	plans[2].ShotScale = "medium"
	_, user := buildSmartSuitePlanRefinementPrompts(task, plans)
	for _, assigned := range []string{`"assigned_shot_scale":"close_up"`, `"assigned_shot_scale":"wide"`, `"assigned_shot_scale":"medium"`} {
		if !strings.Contains(user, assigned) {
			t.Fatalf("planner prompt must include the creative-director assignment %q", assigned)
		}
	}
	for _, expected := range []string{"assigned_shot_scale", "selected from the fact and proof", "not from image index"} {
		if !strings.Contains(user, expected) {
			t.Fatalf("planner prompt must enforce camera diversity with %q", expected)
		}
	}
	for _, forbidden := range []string{"required_shot_scale", "fixed shot", "approved_title exactly", "approved_subtitle exactly"} {
		if strings.Contains(user, forbidden) {
			t.Fatalf("planner prompt must not restore fixed index assignments or immutable copy fields: %q", forbidden)
		}
	}
}

func TestValidateSmartSuitePlanningBatchReportsMissingAndInvalidItems(t *testing.T) {
	requested := []model.EcomSuiteImagePlan{
		completeSmartSuiteTestPlan("smart_secondary", 1, "Context"),
		completeSmartSuiteTestPlan("smart_secondary", 2, "Material"),
	}
	valid := completeSmartSuiteTestPlan("smart_secondary", 1, "Context")
	invalid := completeSmartSuiteTestPlan("smart_secondary", 2, "Material")
	invalid.TypographyInstruction = ""

	err := validateSmartSuitePlanningBatch(requested, []model.EcomSuiteImagePlan{valid, invalid})
	if err == nil || !strings.Contains(err.Error(), "smart_secondary#2:missing_typography_instruction") {
		t.Fatalf("batch validation must identify the exact invalid plan and field, got %v", err)
	}
	err = validateSmartSuitePlanningBatch(requested, []model.EcomSuiteImagePlan{valid})
	if err == nil || !strings.Contains(err.Error(), "smart_secondary#2:missing_response") {
		t.Fatalf("batch validation must identify an omitted plan, got %v", err)
	}
	if err := validateSmartSuitePlanningBatch(requested, []model.EcomSuiteImagePlan{valid, completeSmartSuiteTestPlan("smart_secondary", 2, "Material")}); err != nil {
		t.Fatalf("complete planning batch must pass validation: %v", err)
	}
}

func TestSmartSuiteVisualPromptCopyCheckUsesExactPhraseBoundaries(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Context")
	if smartSuiteVisualPromptRepeatsApprovedCopy(plan) {
		t.Fatal("title Context must not match the legitimate prompt word contextual")
	}
	plan.Prompt = "Create the scene and visibly render Context as the headline. " + smartSuiteGenerationArtworkConstraint
	if !smartSuiteVisualPromptRepeatsApprovedCopy(plan) {
		t.Fatal("standalone approved title in the visual prompt must still be rejected")
	}
}

func TestSmartSuitePromptsRejectEditorialWebLayouts(t *testing.T) {
	task := model.EcomSuiteTask{Platform: "amazon", CountryName: "United States", Language: "English", VisualStyle: "real kitchen scene"}
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Two Spray Modes")
	system, user := buildSmartSuitePlanRefinementPrompts(task, []model.EcomSuiteImagePlan{plan})
	combined := strings.ToLower(system + "\n" + user)
	for _, expected := range []string{"dominant visual anchor", "editorial web article", "decorative outer frames", "oversized header zones", "floating chapter tabs", "split comparison"} {
		if !strings.Contains(combined, expected) {
			t.Fatalf("visual planner must reject web-editorial layout marker %q: %s", expected, combined)
		}
	}
	finalPrompt := strings.ToLower(buildSmartSuiteImageGenerationPromptForLanguage(plan, "", task.Language))
	for _, expected := range []string{"native ecommerce product-detail image", "never an editorial web article", "product or its visible functional proof the dominant visual anchor"} {
		if !strings.Contains(finalPrompt, expected) {
			t.Fatalf("generation prompt must contain ecommerce hierarchy rule %q: %s", expected, finalPrompt)
		}
	}
	legacyPrompt := "Preserve the exact SKU appearance in a complete finished commercial scene-based image with no blank placeholder and fully readable copy."
	if smartSuiteGenerationArtworkConstraintPresent(legacyPrompt) {
		t.Fatal("the previous artwork constraint must not pass without the ecommerce hierarchy boundary")
	}
	normalized := normalizeSmartSuitePlanningCandidate(model.EcomSuiteImagePlan{Prompt: legacyPrompt})
	if !strings.Contains(strings.ToLower(normalized.Prompt), "native ecommerce product-detail") {
		t.Fatalf("normalization must append the ecommerce hierarchy boundary: %s", normalized.Prompt)
	}
}

func TestSmartSuiteTechnicalMetadataNeverBecomesImageCopy(t *testing.T) {
	if got := sanitizeSmartSuiteTechnicalMetadata("Use a 4px corner radius, 12px padding and a UI card."); strings.Contains(strings.ToLower(got), "px") || strings.Contains(strings.ToLower(got), "corner radius") || strings.Contains(strings.ToLower(got), "ui card") {
		t.Fatalf("technical metadata was not sanitized: %q", got)
	}
	task := model.EcomSuiteTask{PlanSnapshot: &model.EcomSuitePlanSnapshot{GlobalStyle: model.EcomSuiteGlobalStyle{
		PrimaryColor: "deep teal", SecondaryColor: "white and graphite", BackgroundTone: "real kitchen", Lighting: "soft side light", ColorSystem: "fixed palette", FontSystem: "Inter", LabelSystem: "4px corner radius with bordered cards", TextureQuality: "real photography",
	}}}
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Show the control")
	plan.Style = "A real scene with a 4px corner radius card"
	plan.LayoutInstruction = "Use 8px padding and a CSS UI card"
	plan.TypographyInstruction = "12px font-size with 4px border radius"
	plan.Prompt = "Create the image with a 4px corner radius"
	contract := decodeSmartSuiteGenerationContract(t, buildSmartSuiteImageGenerationPromptForTask(task, plan, ""))
	if strings.Contains(strings.ToLower(contract.SuiteStyle.LabelSystem), "corner") || strings.Contains(strings.ToLower(contract.SuiteStyle.LabelSystem), "px") {
		t.Fatalf("style label system leaked implementation values: %q", contract.SuiteStyle.LabelSystem)
	}
	for _, value := range []string{contract.ImagePlan.LayoutInstruction, contract.ImagePlan.TypographyInstruction, contract.ImagePlan.VisualRequirements} {
		lower := strings.ToLower(value)
		if strings.Contains(lower, "corner radius") || strings.Contains(lower, "border radius") || strings.Contains(lower, "px") || strings.Contains(lower, "ui card") {
			t.Fatalf("generation field leaked technical metadata: %q", value)
		}
	}
}

func completeSmartSuiteTestPlan(imageType string, imageIndex int, title string) model.EcomSuiteImagePlan {
	return model.EcomSuiteImagePlan{
		ImageType: imageType, ImageIndex: imageIndex,
		Role: "Buyer question", CTADirection: "Persuasion task", Title: title,
		Subtitle: "Supporting copy", MustIncludeText: "Label one",
		Style:             "A real home scene with natural light",
		LayoutInstruction: "Three-quarter camera, product fills 42% of the frame on the right, copy-safe area on the left",
		LayoutSpec: model.EcomSuiteLayoutSpec{
			ProductOccupancyPercent: 55,
			ProductAnchor:           "right",
			TextSafeArea:            model.EcomSuiteLayoutRegion{XPercent: 8, YPercent: 12, WidthPercent: 36, HeightPercent: 30},
			ComparisonMode:          "none",
			CameraAngle:             "eye-level three-quarter view with natural perspective",
			ForbiddenLayouts:        []string{"editorial_header", "floating_tab", "decorative_frame", "ui_card"},
		},
		TypographyInstruction: "Headline at most two lines, supporting copy below, 8% safe margins",
		VisualStrategy:        "Visible action and result",
		Prompt:                "Create the complete planned image. " + smartSuiteGenerationArtworkConstraint,
	}
}

func completeSmartSuiteTestFactMap() model.EcomSuiteProductFactMap {
	return model.EcomSuiteProductFactMap{
		ConfirmedFacts: []model.EcomSuiteProductFact{
			{ID: "F1", Statement: "The holder supports two rolls", EvidenceSource: "product_image", Evidence: "two roll positions are visible"},
			{ID: "F2", Statement: "The shelf has visible marble veining", EvidenceSource: "user_document", Evidence: "natural marble shelf"},
		},
		Inferences: []model.EcomSuiteProductInference{
			{ID: "I1", Statement: "The shelf may help keep a phone within reach", Basis: "a flat shelf is visible above the rolls", Confidence: 0.75, AllowedUse: "buyer_concern_hypothesis_only"},
		},
	}
}

func completeSmartSuiteTestConcerns() []model.EcomSuiteBuyerConcern {
	return []model.EcomSuiteBuyerConcern{
		{ID: "C1", Priority: 1, Question: "Will it hold two rolls?", Reason: "Capacity affects refill frequency", EvidenceFactIDs: []string{"F1"}, ProofMethod: "show two installed rolls", RecommendedImageTypes: []string{"smart_desktop_a_plus"}},
		{ID: "C2", Priority: 2, Question: "Is the shelf real marble?", Reason: "Material affects finish expectations", EvidenceFactIDs: []string{"F2"}, InferenceIDs: []string{"I1"}, ProofMethod: "show a close material detail", RecommendedImageTypes: []string{"smart_mobile_a_plus"}},
	}
}

func TestSmartSuiteProductFactMapSeparatesConfirmedFactsAndInferences(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	if !smartSuiteProductFactMapComplete(factMap) {
		t.Fatalf("valid fact map should pass: %+v", factMap)
	}
	factMap.Inferences[0].Statement = "The shelf keeps every phone secure"
	if smartSuiteProductFactMapComplete(factMap) {
		t.Fatal("an inference without conservative language must be rejected")
	}
}

func TestParseSmartSuiteProductFactMapAcceptsWrapperAndCanonicalizesAliases(t *testing.T) {
	raw := `<think>{"confirmed_facts":[]}</think>{"product_fact_map":{"confirmed_information":[{"id":"f1","statement":"Two rolls are visible","evidence_source":"image","evidence":"two visible roll positions"}],"inferences":[{"id":"i1","statement":"The shelf may keep small items nearby","basis":"a flat shelf is visible","confidence":0.75,"allowed_use":"hypothesis_only"}]}}`
	factMap, err := parseSmartSuiteProductFactMapResponse(raw)
	if err != nil {
		t.Fatalf("wrapped fact map should parse: %v", err)
	}
	if len(factMap.ConfirmedFacts) != 1 || factMap.ConfirmedFacts[0].ID != "F1" || factMap.ConfirmedFacts[0].EvidenceSource != "product_image" {
		t.Fatalf("confirmed fact aliases were not canonicalized: %+v", factMap.ConfirmedFacts)
	}
	if len(factMap.Inferences) != 1 || factMap.Inferences[0].ID != "I1" || factMap.Inferences[0].AllowedUse != "buyer_concern_hypothesis_only" {
		t.Fatalf("inference aliases were not canonicalized: %+v", factMap.Inferences)
	}
	if err := validateSmartSuiteProductFactMap(factMap); err != nil {
		t.Fatalf("canonicalized fact map should validate: %v", err)
	}
}

func TestSmartSuiteProductFactValidationReturnsActionableReason(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	factMap.Inferences[0].Confidence = 0.95
	err := validateSmartSuiteProductFactMap(factMap)
	if err == nil || !strings.Contains(err.Error(), "inference_invalid_confidence:I1") {
		t.Fatalf("expected a field-specific validation error, got %v", err)
	}
}

func TestSmartSuiteDocumentFactEvidenceMustExistInUserInput(t *testing.T) {
	task := model.EcomSuiteTask{ProductFeatures: "double-roll holder", ProductProof: "natural marble shelf"}
	factMap := completeSmartSuiteTestFactMap()
	if !smartSuiteProductFactMapSupportedByTask(task, factMap) {
		t.Fatalf("document-backed evidence should pass: %+v", factMap)
	}
	factMap.ConfirmedFacts[1].Evidence = "certified waterproof construction"
	if smartSuiteProductFactMapSupportedByTask(task, factMap) {
		t.Fatal("document evidence not present in user input must be rejected")
	}
}

func TestSmartSuiteDocumentFactEvidenceAcceptsEquivalentSourceFormatting(t *testing.T) {
	task := model.EcomSuiteTask{ProductFeatures: "15.27 in Height with 360° swivel design and high-temp resistant PEX"}
	factMap := model.EcomSuiteProductFactMap{ConfirmedFacts: []model.EcomSuiteProductFact{
		{ID: "F1", Statement: "The faucet is 15.27 inches high", EvidenceSource: "user_document", Evidence: "15.27-inch height"},
		{ID: "F2", Statement: "The faucet swivels 360 degrees", EvidenceSource: "user_document", Evidence: "360-degree swivel design"},
		{ID: "F3", Statement: "The PEX hose resists high temperatures", EvidenceSource: "user_document", Evidence: "high-temperature resistant PEX"},
	}}
	if err := validateSmartSuiteProductFactMapForTask(task, factMap); err != nil {
		t.Fatalf("equivalent source formatting should remain grounded: %v", err)
	}

	reconciled := reconcileSmartSuiteProductFactEvidence(task, factMap)
	if got := reconciled.ConfirmedFacts[0].Evidence; got != "15.27 in Height" {
		t.Fatalf("evidence should be restored to the source excerpt, got %q", got)
	}
	if got := reconciled.ConfirmedFacts[1].Evidence; got != "360° swivel design" {
		t.Fatalf("degree evidence should be restored to the source excerpt, got %q", got)
	}
	if got := reconciled.ConfirmedFacts[2].Evidence; got != "high-temp resistant PEX" {
		t.Fatalf("temperature evidence should be restored to the source excerpt, got %q", got)
	}
}

func TestSmartSuiteDocumentFactEvidenceRecoversFromParaphrasedEvidence(t *testing.T) {
	task := model.EcomSuiteTask{ProductFeatures: "Dual spray modes with a pull-out hose"}
	factMap := model.EcomSuiteProductFactMap{ConfirmedFacts: []model.EcomSuiteProductFact{
		{ID: "F1", Statement: "Dual spray modes", EvidenceSource: "user_document", Evidence: "The product offers flexible spray settings"},
	}}
	reconciled := reconcileSmartSuiteProductFactEvidence(task, factMap)
	if reconciled.ConfirmedFacts[0].Evidence != "Dual spray modes" {
		t.Fatalf("expected exact source evidence recovered from statement, got %q", reconciled.ConfirmedFacts[0].Evidence)
	}
	if err := validateSmartSuiteProductFactMapForTask(task, reconciled); err != nil {
		t.Fatalf("statement-backed evidence should validate after recovery: %v", err)
	}
}

func TestSmartSuiteProductFactRecoveryDropsUnsupportedFactsAndUsesTaskSource(t *testing.T) {
	task := model.EcomSuiteTask{ProductFeatures: "Two-roll holder", ProductProof: "Natural marble shelf"}
	factMap := model.EcomSuiteProductFactMap{ConfirmedFacts: []model.EcomSuiteProductFact{
		{ID: "F1", Statement: "Unsupported marketing claim", EvidenceSource: "user_document", Evidence: "not in the supplied document"},
	}}
	recovered, err := recoverSmartSuiteProductFactMapFromTask(task, factMap)
	if err != nil {
		t.Fatalf("expected deterministic recovery to succeed: %v", err)
	}
	if len(recovered.ConfirmedFacts) != 2 || recovered.ConfirmedFacts[0].Evidence != "Two-roll holder" || recovered.ConfirmedFacts[1].Evidence != "Natural marble shelf" {
		t.Fatalf("expected task-backed facts after dropping unsupported output, got %+v", recovered.ConfirmedFacts)
	}
}

func TestSmartSuiteProductDocumentEvidenceAcceptsPromptJSONSource(t *testing.T) {
	task := model.EcomSuiteTask{ProductFeatures: "Two selectable spray modes"}
	factMap := model.EcomSuiteProductFactMap{ConfirmedFacts: []model.EcomSuiteProductFact{
		{ID: "F1", Statement: "Two selectable spray modes", EvidenceSource: "user_document", Evidence: `"product_document":"Two selectable spray modes"`},
	}}
	if err := validateSmartSuiteProductFactMapForTask(task, factMap); err != nil {
		t.Fatalf("prompt JSON evidence should be recognized as grounded: %v", err)
	}
}

func TestSmartSuiteBuyerConcernsRequireConfirmedEvidence(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	concerns := completeSmartSuiteTestConcerns()
	if !smartSuiteBuyerConcernsComplete(concerns, factMap, 2) {
		t.Fatalf("valid concern map should pass: %+v", concerns)
	}
	concerns[0].EvidenceFactIDs = []string{"I1"}
	if smartSuiteBuyerConcernsComplete(concerns, factMap, 2) {
		t.Fatal("a reasonable inference must not be accepted as proof evidence")
	}
}

func TestParseSmartSuiteBuyerConcernsAcceptsWrapperAndAliases(t *testing.T) {
	raw := `<analysis>{"buyer_concerns":[]}</analysis>{"buyer_insight":{"concerns":[{"concern_id":"c1","rank":"1","buyer_question":"Will it hold two rolls?","purchase_barrier_reason":"Capacity affects refill frequency","confirmed_fact_ids":"f1","reasonable_inference_ids":"i1","visual_proof_method":"show two installed rolls","suitable_image_types":["desktop A+","mobile A+"]},{"concern_id":"c2","rank":2,"buyer_question":"Is the shelf real marble?","concern_reason":"Material affects finish expectations","fact_ids":["F2"],"proof":"show a close material detail","image_types":"桌面 A+"}]}}`
	concerns, err := parseSmartSuiteBuyerConcernResponse(raw)
	if err != nil {
		t.Fatalf("wrapped buyer concerns should parse: %v", err)
	}
	if len(concerns) != 2 || concerns[0].ID != "C1" || concerns[0].Priority != 1 {
		t.Fatalf("buyer concern identity aliases were not normalized: %+v", concerns)
	}
	if got := concerns[0].RecommendedImageTypes; len(got) != 2 || got[0] != "smart_desktop_a_plus" || got[1] != "smart_mobile_a_plus" {
		t.Fatalf("buyer concern image types were not normalized: %+v", got)
	}
	if err := validateSmartSuiteBuyerConcerns(concerns, completeSmartSuiteTestFactMap(), 2); err != nil {
		t.Fatalf("normalized buyer concerns should validate: %v", err)
	}
}

func TestSmartSuiteBuyerConcernValidationReturnsActionableReason(t *testing.T) {
	concerns := completeSmartSuiteTestConcerns()
	concerns[1].EvidenceFactIDs = []string{"F99"}
	err := validateSmartSuiteBuyerConcerns(concerns, completeSmartSuiteTestFactMap(), 2)
	if err == nil || !strings.Contains(err.Error(), "concern_unknown_confirmed_fact:C2:F99") {
		t.Fatalf("expected a field-specific concern error, got %v", err)
	}
}

func TestSmartSuiteBuyerConcernCountFollowsLargestRequestedStage(t *testing.T) {
	task := model.EcomSuiteTask{SecondaryCount: 3, DesktopAPlusCount: 7, MobileAPlusCount: 5}
	if got := smartSuiteBuyerConcernCount(task); got != 7 {
		t.Fatalf("expected concern map to cover the largest requested stage, got %d", got)
	}
}

func TestSmartSuiteLayoutSpecIsValidatedAndCanonicalized(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Visible proof")
	if !smartSuiteLayoutSpecComplete(plan.LayoutSpec) {
		t.Fatalf("valid layout spec should pass: %+v", plan.LayoutSpec)
	}
	plan.LayoutInstruction = ""
	normalized := normalizeSmartSuitePlanningCandidate(plan)
	for _, expected := range []string{"occupies approximately 55%", "anchors to the right", "x=8%", "Comparison mode: none", "editorial_header"} {
		if !strings.Contains(normalized.LayoutInstruction, expected) {
			t.Fatalf("canonical layout instruction missing %q: %s", expected, normalized.LayoutInstruction)
		}
	}
	plan.LayoutInstruction = "Low camera beside the basin; show handle contact against a soft distant background."
	if got := normalizeSmartSuitePlanningCandidate(plan); got.LayoutInstruction != plan.LayoutInstruction {
		t.Fatalf("director's depth and contact details were overwritten: %s", got.LayoutInstruction)
	}
	plan.LayoutSpec.TextSafeArea.WidthPercent = 95
	if smartSuiteLayoutSpecComplete(plan.LayoutSpec) {
		t.Fatal("an out-of-canvas text safe area must be rejected")
	}
}

func TestCreativeDirectorResolvesAssignedFactsFromConfirmedIDs(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	concerns := completeSmartSuiteTestConcerns()
	task := model.EcomSuiteTask{ProductFeatures: "two-roll holder", PlanSnapshot: &model.EcomSuitePlanSnapshot{ProductFactMap: factMap, BuyerConcerns: concerns}}
	plans := []model.EcomSuiteImagePlan{{ImageType: "smart_secondary", ImageIndex: 1, Role: concerns[0].Question}}
	candidates := []model.EcomSuiteCreativeDirection{{
		ImageType: "smart_secondary", ImageIndex: 1, BuyerConcernIDs: []string{"C1"}, BuyerQuestion: concerns[0].Question,
		PrimarySellingPoint: "Keep a second roll ready", AssignedFactIDs: []string{"F1"}, AssignedProductFacts: "invented model text",
		VisualEvidence: "two rolls are visibly installed", SceneDirection: "A real bathroom wall with soft daylight and two installed rolls",
	}}
	resolved, err := reconcileSmartSuiteCreativeDirections(task, plans, candidates)
	if err != nil || len(resolved) != 1 {
		t.Fatalf("expected one resolved direction, got %+v err=%v", resolved, err)
	}
	if resolved[0].AssignedProductFacts != factMap.ConfirmedFacts[0].Statement {
		t.Fatalf("backend must resolve facts from confirmed IDs, got %q", resolved[0].AssignedProductFacts)
	}
	candidates[0].AssignedFactIDs = []string{"I1"}
	if _, err := reconcileSmartSuiteCreativeDirections(task, plans, candidates); err == nil || !strings.Contains(err.Error(), "unapproved_fact_id") {
		t.Fatalf("inference ID must not be assignable as a product fact, err=%v", err)
	}
}

func TestCreativeDirectorAllowsAPlusConcernToDifferFromNarrativeRole(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	concerns := completeSmartSuiteTestConcerns()
	task := model.EcomSuiteTask{PlanSnapshot: &model.EcomSuitePlanSnapshot{ProductFactMap: factMap, BuyerConcerns: concerns}}
	plans := []model.EcomSuiteImagePlan{{ImageType: "smart_desktop_a_plus", ImageIndex: 1, Role: "Product positioning"}}
	candidates := []model.EcomSuiteCreativeDirection{{
		ImageType: "smart_desktop_a_plus", ImageIndex: 1, BuyerConcernIDs: []string{"C1"}, BuyerQuestion: concerns[0].Question,
		PrimarySellingPoint: "Keep a second roll ready", AssignedFactIDs: []string{"F1"},
		VisualEvidence: "two installed rolls are visible", SceneDirection: "A real bathroom wall with soft daylight and two installed rolls",
	}}

	resolved, err := reconcileSmartSuiteCreativeDirections(task, plans, candidates)
	if err != nil || len(resolved) != 1 {
		t.Fatalf("A+ concern may differ from its narrative role when fact linkage is valid, got %+v err=%v", resolved, err)
	}
	if resolved[0].AssignedProductFacts != factMap.ConfirmedFacts[0].Statement {
		t.Fatalf("backend must resolve the concern-linked fact, got %q", resolved[0].AssignedProductFacts)
	}
}

func TestCreativeDirectorRejectsAPlusFactOutsideSelectedConcern(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	concerns := completeSmartSuiteTestConcerns()
	task := model.EcomSuiteTask{PlanSnapshot: &model.EcomSuitePlanSnapshot{ProductFactMap: factMap, BuyerConcerns: concerns}}
	plans := []model.EcomSuiteImagePlan{{ImageType: "smart_desktop_a_plus", ImageIndex: 1, Role: "Product positioning"}}
	candidates := []model.EcomSuiteCreativeDirection{{
		ImageType: "smart_desktop_a_plus", ImageIndex: 1, BuyerConcernIDs: []string{"C1"}, BuyerQuestion: concerns[0].Question,
		PrimarySellingPoint: "Show the material finish", AssignedFactIDs: []string{"F2"},
		VisualEvidence: "a close material detail is visible", SceneDirection: "A real bathroom wall with soft daylight and a close material detail",
	}}

	if _, err := reconcileSmartSuiteCreativeDirections(task, plans, candidates); err == nil || !strings.Contains(err.Error(), "creative_director_unapproved_fact_id") {
		t.Fatalf("A+ must still use facts linked to its selected concern, err=%v", err)
	}
}

func TestCreativeDirectorRepairsUnknownAPlusFactIDFromConcern(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	concerns := completeSmartSuiteTestConcerns()
	task := model.EcomSuiteTask{PlanSnapshot: &model.EcomSuitePlanSnapshot{ProductFactMap: factMap, BuyerConcerns: concerns}}
	plans := []model.EcomSuiteImagePlan{{ImageType: "smart_desktop_a_plus", ImageIndex: 2, Role: "Material confidence"}}
	candidates := []model.EcomSuiteCreativeDirection{{
		ImageType: "smart_desktop_a_plus", ImageIndex: 2, BuyerConcernIDs: []string{"C2"}, BuyerQuestion: concerns[1].Question,
		PrimarySellingPoint: "Show the marble finish", AssignedFactIDs: []string{"F4"},
		VisualEvidence: "a close material detail is visible", SceneDirection: "A real bathroom wall with soft daylight and a close material detail",
	}}
	resolved, err := reconcileSmartSuiteCreativeDirections(task, plans, candidates)
	if err != nil || len(resolved) != 1 {
		t.Fatalf("unknown A+ fact ID should be repaired from concern evidence, got %+v err=%v", resolved, err)
	}
	if len(resolved[0].AssignedFactIDs) != 1 || resolved[0].AssignedFactIDs[0] != "F2" || resolved[0].AssignedProductFacts != factMap.ConfirmedFacts[1].Statement {
		t.Fatalf("unexpected repaired fact assignment: %+v", resolved[0])
	}
}

func TestCreativeDirectorAllowsRepeatedCoreSellingPointWithDistinctProof(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	concerns := completeSmartSuiteTestConcerns()
	task := model.EcomSuiteTask{PlanSnapshot: &model.EcomSuitePlanSnapshot{ProductFactMap: factMap, BuyerConcerns: concerns}}
	plans := []model.EcomSuiteImagePlan{
		{ImageType: "smart_desktop_a_plus", ImageIndex: 1, Role: "Product positioning"},
		{ImageType: "smart_desktop_a_plus", ImageIndex: 2, Role: "Material confidence"},
	}
	candidates := []model.EcomSuiteCreativeDirection{
		{
			ImageType: "smart_desktop_a_plus", ImageIndex: 1, BuyerConcernIDs: []string{"C1"}, BuyerQuestion: concerns[0].Question,
			PrimarySellingPoint: "Confidence from visible proof", AssignedFactIDs: []string{"F1"},
			VisualEvidence: "two installed rolls are visible", SceneDirection: "A real bathroom wall with soft daylight and two installed rolls",
		},
		{
			ImageType: "smart_desktop_a_plus", ImageIndex: 2, BuyerConcernIDs: []string{"C2"}, BuyerQuestion: concerns[1].Question,
			PrimarySellingPoint: "Confidence from visible proof", AssignedFactIDs: []string{"F2"},
			VisualEvidence: "marble veining is visible in close detail", SceneDirection: "A maintained bathroom vanity with side light revealing the marble veining",
		},
	}

	resolved, err := reconcileSmartSuiteCreativeDirections(task, plans, candidates)
	if err != nil || len(resolved) != 2 {
		t.Fatalf("repeated core wording with distinct concern, fact and proof must pass: resolved=%+v err=%v", resolved, err)
	}
}

func TestCreativeDirectorRejectsRepeatedCompleteModuleContent(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	concerns := completeSmartSuiteTestConcerns()
	task := model.EcomSuiteTask{PlanSnapshot: &model.EcomSuitePlanSnapshot{ProductFactMap: factMap, BuyerConcerns: concerns}}
	plans := []model.EcomSuiteImagePlan{
		{ImageType: "smart_desktop_a_plus", ImageIndex: 1, Role: "Product positioning"},
		{ImageType: "smart_desktop_a_plus", ImageIndex: 2, Role: "Trust close"},
	}
	candidate := model.EcomSuiteCreativeDirection{
		ImageType: "smart_desktop_a_plus", BuyerConcernIDs: []string{"C1"}, BuyerQuestion: concerns[0].Question,
		PrimarySellingPoint: "Confidence from visible proof", AssignedFactIDs: []string{"F1"},
		VisualEvidence: "two installed rolls are visible", SceneDirection: "A real bathroom wall with soft daylight and two installed rolls",
	}
	first, second := candidate, candidate
	first.ImageIndex = 1
	second.ImageIndex = 2

	if _, err := reconcileSmartSuiteCreativeDirections(task, plans, []model.EcomSuiteCreativeDirection{first, second}); err == nil || !strings.Contains(err.Error(), "creative_director_duplicate_module_content") {
		t.Fatalf("a fully repeated proof assignment must still fail, err=%v", err)
	}
}

func TestCreativeDirectorRejectsMultipleFactsForOneImage(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	concerns := completeSmartSuiteTestConcerns()
	concerns[0].EvidenceFactIDs = []string{"F1", "F2"}
	task := model.EcomSuiteTask{PlanSnapshot: &model.EcomSuitePlanSnapshot{ProductFactMap: factMap, BuyerConcerns: concerns}}
	plans := []model.EcomSuiteImagePlan{{ImageType: "smart_secondary", ImageIndex: 1, Role: concerns[0].Question}}
	candidates := []model.EcomSuiteCreativeDirection{{
		ImageType: "smart_secondary", ImageIndex: 1, BuyerConcernIDs: []string{"C1"}, BuyerQuestion: concerns[0].Question,
		PrimarySellingPoint: "Keep a second roll ready", AssignedFactIDs: []string{"F1", "F2"},
		VisualEvidence: "two installed rolls are visible", SceneDirection: "A real bathroom wall with soft daylight and two installed rolls",
	}}

	if _, err := reconcileSmartSuiteCreativeDirections(task, plans, candidates); err == nil || !strings.Contains(err.Error(), "creative_director_fact_count") {
		t.Fatalf("one image must reject multiple assigned facts, got %v", err)
	}
}

func TestStyleMasterRequiresConcreteCompleteStyle(t *testing.T) {
	style := model.EcomSuiteGlobalStyle{PrimaryColor: "brushed brass accent", SecondaryColor: "charcoal and warm white", BackgroundTone: "warm white residential bathroom", Lighting: "soft daylight from camera left at neutral temperature", ColorSystem: "use brass only as a supporting accent", FontSystem: "geometric sans, semibold headline and regular body", LabelSystem: "flat charcoal labels with 4px radius", TextureQuality: "realistic materials, crisp edges and consistent sharpness"}
	if !smartSuiteGlobalStyleComplete(style) {
		t.Fatalf("complete style should pass: %+v", style)
	}
	style.FontSystem = ""
	if smartSuiteGlobalStyleComplete(style) {
		t.Fatal("style without a font system must fail")
	}
}

func TestParseSmartSuiteStyleMasterAcceptsWrapperAliasesAndStructuredValues(t *testing.T) {
	raw := `<think>{"global_style":{}}</think>{"style_master":{"style_system":{"accent_color":"brushed brass","supporting_palette":["charcoal","warm white"],"background_system":{"environment":"residential bathroom","surface":"warm white stone"},"lighting_system":{"direction":"camera left","softness":"soft","temperature":"neutral"},"palette_rules":"use brass only as a supporting accent","typography_system":{"category":"geometric sans","weights":"600 headline, 400 body","case":"sentence case","spacing":"normal"},"tag_system":{"shape":"flat rectangle","radius":"4px","border":"none","padding":"8px 12px"},"photography_quality":["realistic materials","crisp edges","consistent sharpness"]},"style_evidence":[{"origin":"product_image","observation":"visible brushed brass finish"}]}}`
	styleMaster, err := parseSmartSuiteStyleMasterResponse(raw)
	if err != nil {
		t.Fatalf("wrapped structured style should parse: %v", err)
	}
	if styleMaster.GlobalStyle.PrimaryColor != "brushed brass" || !strings.Contains(styleMaster.GlobalStyle.SecondaryColor, "charcoal") || !strings.Contains(styleMaster.GlobalStyle.Lighting, "temperature: neutral") {
		t.Fatalf("style aliases or structured values were not normalized: %+v", styleMaster.GlobalStyle)
	}
	if err := validateSmartSuiteStyleMaster(styleMaster); err != nil {
		t.Fatalf("normalized style master should validate: %v", err)
	}
}

func TestSmartSuiteStyleMasterValidationReturnsActionableReason(t *testing.T) {
	styleMaster := model.EcomSuiteStyleMaster{GlobalStyle: model.EcomSuiteGlobalStyle{
		PrimaryColor: "brass", SecondaryColor: "charcoal", BackgroundTone: "warm white bathroom", Lighting: "soft neutral light from left",
		ColorSystem: "brass accent only", FontSystem: "geometric sans", LabelSystem: "flat labels",
	}, Evidence: []model.EcomSuiteStyleEvidence{{Source: "product_image", Detail: "brass finish is visible"}}}
	err := validateSmartSuiteStyleMaster(styleMaster)
	if err == nil || !strings.Contains(err.Error(), "missing_texture_quality") {
		t.Fatalf("expected a field-specific style error, got %v", err)
	}
}

func TestSmartSuiteFirstStageFollowsEnabledOutputOrder(t *testing.T) {
	all := model.EcomSuiteTask{SecondaryCount: 6, DesktopAPlusCount: 7, MobileAPlusCount: 7}
	if stage := smartSuiteFirstStageName(all, "generating"); stage != "generating_secondary" {
		t.Fatalf("expected secondary generation first, got %q", stage)
	}
	desktopMobile := model.EcomSuiteTask{DesktopAPlusCount: 7, MobileAPlusCount: 7}
	if stage := smartSuiteFirstStageName(desktopMobile, "planning"); stage != "planning" {
		t.Fatalf("expected the task to wait for the orchestrator decision, got %q", stage)
	}
	if stage := smartSuiteFirstStageName(desktopMobile, "generating"); stage != "generating_desktop" {
		t.Fatalf("expected desktop generation first when secondary is disabled, got %q", stage)
	}
}

func TestSmartSuiteImageOutputRatioIsLockedByPlatform(t *testing.T) {
	tests := []struct {
		imageType string
		want      string
	}{
		{imageType: "smart_secondary", want: "1:1"},
		{imageType: "smart_desktop_a_plus", want: "21:9"},
		{imageType: "smart_mobile_a_plus", want: "4:3"},
	}
	for _, test := range tests {
		if got := smartSuiteImageOutputRatio(test.imageType, "1:1"); got != test.want {
			t.Fatalf("ratio for %s: expected %s, got %s", test.imageType, test.want, got)
		}
	}
	if got := smartSuiteImageOutputRatio("legacy", ""); got != "1:1" {
		t.Fatalf("legacy fallback ratio should remain 1:1, got %s", got)
	}
}

func TestNormalizeSmartSuiteStageAuditRequiresStageChecks(t *testing.T) {
	output := normalizeSmartSuiteStageAudit("smart_mobile_a_plus", smartSuiteStageAuditOutput{
		Passed: true,
		Checks: []model.EcomSuiteAuditCheck{{Key: "product_consistency", Passed: true}},
	})
	if output.Passed {
		t.Fatal("mobile stage audit must fail when mobile-specific checks are missing")
	}
	if len(output.Checks) != 7 {
		t.Fatalf("expected seven stage checks, got %d", len(output.Checks))
	}
}

func TestExtractSmartSuitePlanningJSON(t *testing.T) {
	objectText := "Here is the plan:\n```json\n{\"plans\":[{\"image_type\":\"smart_secondary\",\"image_index\":1}]}\n```"
	object := extractJSONObjectFromText(objectText)
	if object != "{\"plans\":[{\"image_type\":\"smart_secondary\",\"image_index\":1}]}" {
		t.Fatalf("unexpected object extraction: %q", object)
	}
	arrayText := "```json\n[{\"image_type\":\"smart_secondary\",\"image_index\":1}]\n```"
	array := extractJSONArrayFromText(arrayText)
	if array != "[{\"image_type\":\"smart_secondary\",\"image_index\":1}]" {
		t.Fatalf("unexpected array extraction: %q", array)
	}
}

func TestSmartSuitePlanningParsersIgnoreReasoningBlocks(t *testing.T) {
	planningRaw := `<think>{"plans":[{"image_type":"smart_secondary","image_index":99}]}</think>{"plans":[{"image_type":"smart_secondary","image_index":1,"role":"Question","cta_direction":"Proof","title":"Title","subtitle":"Subtitle","must_include_text":"No labels","style":"A real bathroom scene with daylight","visual_strategy":"Show the visible result","prompt":"Create a complete finished scene-based commercial image with exact SKU appearance, no blank placeholder panels and fully readable copy."}]}`
	plans, err := parseSmartSuitePlanningResponse(planningRaw)
	if err != nil || len(plans) != 1 || plans[0].ImageIndex != 1 {
		t.Fatalf("planning parser must ignore model reasoning JSON, plans=%+v err=%v", plans, err)
	}
	questionsRaw := `<analysis>{"questions":["wrong 1","wrong 2","wrong 3","wrong 4","wrong 5","wrong 6"]}</analysis>{"questions":["Where is it used?","What material is visible?","How does it work?","How is it installed?","What is the fit?","What is included?"]}`
	questions, err := parseSmartSuiteBuyerQuestions(questionsRaw, "material, installation, fit")
	if err != nil || len(questions) != 6 || questions[0] != "Where is it used?" {
		t.Fatalf("buyer parser must ignore model reasoning JSON, questions=%+v err=%v", questions, err)
	}
}

func TestSmartSuiteCreativeDirectorParserAcceptsObjectAndArrayResponses(t *testing.T) {
	arrayRaw := `<think>{"directions":[]}</think>[{"image_type":"smart_secondary","image_index":1,"buyer_question":"Where is it used?","primary_selling_point":"Show the use","assigned_product_facts":"stainless steel","visual_evidence":"visible result","scene_direction":"A real bathroom scene with daylight"}]`
	directions, err := parseSmartSuiteCreativeDirectorResponse(arrayRaw)
	if err != nil || len(directions) != 1 || directions[0].ImageIndex != 1 {
		t.Fatalf("creative director array response should be accepted, directions=%+v err=%v", directions, err)
	}
	objectRaw := `{"creative_directions":[{"image_type":"smart_desktop_a_plus","image_index":1,"buyer_question":"Core mechanism","primary_selling_point":"Show the mechanism","assigned_product_facts":"dual spray","visual_evidence":"visible structure","scene_direction":"A real workshop scene with side light"}]}`
	directions, err = parseSmartSuiteCreativeDirectorResponse(objectRaw)
	if err != nil || len(directions) != 1 || directions[0].ImageType != "smart_desktop_a_plus" {
		t.Fatalf("creative director object aliases should be accepted, directions=%+v err=%v", directions, err)
	}
	aliasedRaw := `{"directions":[{"type":"desktop A+","index":"2","role":"Core mechanism","selling_point":"Show the structure","product_facts":"dual spray","visual_proof":"visible mechanism","scene":"A real workshop scene with side light"}]}`
	directions, err = parseSmartSuiteCreativeDirectorResponse(aliasedRaw)
	if err != nil || len(directions) != 1 || directions[0].ImageType != "smart_desktop_a_plus" || directions[0].ImageIndex != 2 {
		t.Fatalf("creative director aliases and string index should be normalized, directions=%+v err=%v", directions, err)
	}
}

func TestSmartSuiteCreativeDirectorResponseExampleUsesCurrentStage(t *testing.T) {
	example := smartSuiteCreativeDirectorResponseExample("smart_mobile_a_plus", "the supplied role", true)
	if !strings.Contains(example, `"image_type":"smart_mobile_a_plus"`) {
		t.Fatalf("mobile response example must use the current stage identity: %s", example)
	}
	if strings.Contains(example, `"image_type":"smart_secondary"`) {
		t.Fatalf("mobile response example must not contain the secondary identity: %s", example)
	}
}

func TestParseSmartSuiteBuyerQuestions(t *testing.T) {
	raw := "Here is the result:\n```json\n" + `{"questions":[
  "这款抽拉龙头能覆盖多大的水槽操作范围？",
  "双出水模式分别适合哪些清洗任务？",
  "抽拉软管回位是否顺畅稳定？",
  "黄铜主体和表面处理如何体现耐用做工？",
  "安装时需要确认哪些孔位和连接条件？",
  "包装内包含哪些安装所需组件？"
]}` + "\n```\n"
	questions, err := parseSmartSuiteBuyerQuestions(raw, "黄铜主体，双出水模式，抽拉软管，标准安装组件")
	if err != nil {
		t.Fatalf("expected valid buyer insight JSON, got %v", err)
	}
	if len(questions) != 6 || !strings.Contains(questions[0], "操作范围") || !strings.Contains(questions[5], "组件") {
		t.Fatalf("unexpected buyer questions: %#v", questions)
	}
	arrayRaw, _ := json.Marshal(questions)
	if _, err := parseSmartSuiteBuyerQuestions(string(arrayRaw), "黄铜主体，双出水模式，抽拉软管，标准安装组件"); err != nil {
		t.Fatalf("expected a plain array response to be accepted, got %v", err)
	}
}

func TestParseSmartSuiteBuyerQuestionsRejectsInvalidOutput(t *testing.T) {
	tests := map[string]string{
		"wrong count":       `{"questions":["问题一","问题二"]}`,
		"duplicates":        `{"questions":["适合什么场景？","材质是什么？","结构如何工作？","如何安装？","适合什么场景？","包含什么？"]}`,
		"unsupported claim": `{"questions":["为什么它是best？","材质是什么？","结构如何工作？","如何安装？","适合什么场景？","包含什么？"]}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parseSmartSuiteBuyerQuestions(raw, "黄铜主体，双出水模式"); err == nil {
				t.Fatalf("expected invalid buyer insight output to fail: %s", raw)
			}
		})
	}
}

func TestParseSmartSuiteBuyerQuestionsUsesRequestedCount(t *testing.T) {
	raw := `{"questions":["Where is it used?","What material is visible?","How does it work?"]}`
	questions, err := parseSmartSuiteBuyerQuestionsForCount(raw, "material, structure", 3)
	if err != nil || len(questions) != 3 {
		t.Fatalf("expected the requested secondary count to be accepted, questions=%+v err=%v", questions, err)
	}
	if _, err := parseSmartSuiteBuyerQuestionsForCount(raw, "material, structure", 6); err == nil {
		t.Fatal("a response with the wrong requested count must be rejected")
	}
}

func TestSmartSuiteLegacyBuyerQuestionsRequireFreshAnalysis(t *testing.T) {
	legacyChinese := []string{"使用范围与覆盖能力", "材质和做工", "关键结构如何工作", "安装与使用复杂度", "与普通款的差异", "买回去能得到什么"}
	if !isLegacySmartSuiteBuyerQuestions(legacyChinese) {
		t.Fatal("expected the historical fixed Chinese questions to require fresh analysis")
	}
	dynamic := []string{"水槽操作范围是否够用？", "双出水适合哪些清洗？", "软管如何回位？", "黄铜主体做工如何？", "安装需要什么条件？", "包装包含什么组件？"}
	if isLegacySmartSuiteBuyerQuestions(dynamic) {
		t.Fatal("product-specific questions must not be treated as historical defaults")
	}
}

func TestSmartSuiteBuyerClaimMatchingUsesWordBoundaries(t *testing.T) {
	if smartSuiteContainsClaim("asbestos-free material", "best") {
		t.Fatal("best must not match as part of an unrelated word")
	}
	if !smartSuiteContainsClaim("Why is this the best?", "best") {
		t.Fatal("expected a standalone banned claim to be detected")
	}
}

func TestReconcileSmartSuiteRefinedPlansKeepsPartialResultsAligned(t *testing.T) {
	requested := []model.EcomSuiteImagePlan{
		{ImageType: "smart_secondary", ImageIndex: 1},
		{ImageType: "smart_secondary", ImageIndex: 2},
		{ImageType: "smart_desktop_a_plus", ImageIndex: 1},
	}
	candidates := []model.EcomSuiteImagePlan{
		completeSmartSuiteTestPlan("smart_desktop_a_plus", 1, "Desktop"),
		completeSmartSuiteTestPlan("smart_secondary", 1, "Secondary"),
	}
	aligned, missing, matched := reconcileSmartSuiteRefinedPlans(requested, candidates)
	if matched != 2 || len(missing) != 1 {
		t.Fatalf("expected two matches and one missing plan, got matched=%d missing=%d", matched, len(missing))
	}
	if aligned[0].Title != "Secondary" || aligned[1].ImageType != "" || aligned[2].Title != "Desktop" {
		t.Fatalf("plans were not aligned by image identity: %+v", aligned)
	}
	if missing[0].ImageType != "smart_secondary" || missing[0].ImageIndex != 2 {
		t.Fatalf("unexpected missing plan: %+v", missing[0])
	}
}

func TestReconcileSmartSuiteRefinedPlansRejectsWrongAndDuplicateIdentity(t *testing.T) {
	requested := []model.EcomSuiteImagePlan{{ImageType: "smart_mobile_a_plus", ImageIndex: 1}}
	candidates := []model.EcomSuiteImagePlan{
		completeSmartSuiteTestPlan("same", 1, "Wrong type"),
		completeSmartSuiteTestPlan("smart_mobile_a_plus", 1, "First"),
		completeSmartSuiteTestPlan("smart_mobile_a_plus", 1, "Duplicate"),
		completeSmartSuiteTestPlan("smart_mobile_a_plus", 0, "Wrong index"),
	}
	aligned, missing, matched := reconcileSmartSuiteRefinedPlans(requested, candidates)
	if matched != 1 || len(missing) != 0 || aligned[0].Title != "First" {
		t.Fatalf("unexpected reconciliation result: aligned=%+v matched=%d missing=%+v", aligned, matched, missing)
	}
}

func TestNormalizeSmartSuitePlanningIdentityAliases(t *testing.T) {
	tests := map[string]string{
		"secondary":         "smart_secondary",
		"secondary_image":   "smart_secondary",
		"desktop":           "smart_desktop_a_plus",
		"desktop A+":        "smart_desktop_a_plus",
		"mobile_a_plus":     "smart_mobile_a_plus",
		"mobile A+":         "smart_mobile_a_plus",
		"same as requested": "",
	}
	for input, want := range tests {
		if got := normalizeSmartSuiteImageType(input); got != want {
			t.Fatalf("image type alias %q: expected %q, got %q", input, want, got)
		}
	}
}

func TestParseSmartSuitePlanningResponseNormalizesAliasesAndStringIndex(t *testing.T) {
	raw := `{"plans":[{"type":"desktop A+","index":"7","buyer_question":"Does it fit?","persuasion_task":"Prove fit","headline":"Check the Fit","supporting_copy":"A clear fit check.","labels":"Fit|Measure","real_scene":"A real bathroom installation scene in daylight","visual_evidence":"Show the measured installation result","image_generation_prompt":"Create a complete finished image in a real bathroom scene with readable copy."}]}`
	plans, err := parseSmartSuitePlanningResponse(raw)
	if err != nil || len(plans) != 1 {
		t.Fatalf("expected aliased planning response to parse, got plans=%+v err=%v", plans, err)
	}
	if plans[0].ImageType != "smart_desktop_a_plus" || plans[0].ImageIndex != 7 || plans[0].Role != "Does it fit?" {
		t.Fatalf("planning identity or aliases were not normalized: %+v", plans[0])
	}
}

func TestParseSmartSuitePlanningResponseAcceptsSinglePlanObject(t *testing.T) {
	raw := `{"image_type":"secondary","image_index":1,"role":"What does it answer?","cta_direction":"Show one proof","title":"See the Proof","subtitle":"A short supporting line.","must_include_text":"Proof|Use","style":"A real bathroom scene with daylight","visual_strategy":"Show the visible result in use","prompt":"Create a complete finished image in a real bathroom scene with readable copy."}`
	plans, err := parseSmartSuitePlanningResponse(raw)
	if err != nil || len(plans) != 1 || plans[0].ImageType != "smart_secondary" {
		t.Fatalf("single plan object should be accepted: plans=%+v err=%v", plans, err)
	}
}

func TestReconcileSmartSuiteRefinedPlansAssignsOmittedBatchIdentity(t *testing.T) {
	requested := []model.EcomSuiteImagePlan{
		completeSmartSuiteTestPlan("smart_mobile_a_plus", 6, "Six"),
		completeSmartSuiteTestPlan("smart_mobile_a_plus", 7, "Seven"),
	}
	candidates := []model.EcomSuiteImagePlan{
		completeSmartSuiteTestPlan("", 0, "Six refined"),
		completeSmartSuiteTestPlan("", 0, "Seven refined"),
	}
	aligned, missing, matched := reconcileSmartSuiteRefinedPlans(requested, candidates)
	if matched != 2 || len(missing) != 0 {
		t.Fatalf("omitted batch identity should be assigned by request order: matched=%d missing=%+v", matched, missing)
	}
	if aligned[0].ImageType != "smart_mobile_a_plus" || aligned[0].ImageIndex != 6 || aligned[0].Title != requested[0].Title || aligned[1].ImageIndex != 7 {
		t.Fatalf("batch identity assignment was not stable: %+v", aligned)
	}
}

func TestReconcileSmartSuiteRefinedPlansDoesNotMatchIncompleteCandidate(t *testing.T) {
	requested := []model.EcomSuiteImagePlan{completeSmartSuiteTestPlan("smart_secondary", 1, "Base")}
	candidate := model.EcomSuiteImagePlan{ImageType: "smart_secondary", ImageIndex: 1}
	aligned, missing, matched := reconcileSmartSuiteRefinedPlans(requested, []model.EcomSuiteImagePlan{candidate})
	if matched != 0 || len(missing) != 1 || aligned[0].ImageType != "" {
		t.Fatalf("incomplete planning candidate must require a retry: aligned=%+v matched=%d missing=%+v", aligned, matched, missing)
	}
}

func TestReconcileSmartSuiteRefinedPlansWithFallbackCompletesMissingVisualModules(t *testing.T) {
	requested := []model.EcomSuiteImagePlan{
		completeSmartSuiteTestPlan("smart_secondary", 1, "One"),
		completeSmartSuiteTestPlan("smart_secondary", 2, "Two"),
	}
	requested[0].LayoutSpec = model.EcomSuiteLayoutSpec{}
	requested[1].LayoutSpec = model.EcomSuiteLayoutSpec{}
	aligned, missing, matched := reconcileSmartSuiteRefinedPlansWithFallback(requested, []model.EcomSuiteImagePlan{
		completeSmartSuiteTestPlan("smart_secondary", 1, "One refined"),
	})
	if matched != 2 || len(missing) != 0 || len(aligned) != 2 {
		t.Fatalf("deterministic fallback should complete a recoverable missing module: matched=%d missing=%+v aligned=%+v", matched, missing, aligned)
	}
	if !smartSuiteLayoutSpecComplete(aligned[1].LayoutSpec) || aligned[1].ImageIndex != 2 || aligned[1].Title != requested[1].Title {
		t.Fatalf("fallback must provide only safe visual structure and preserve immutable copy: %+v", aligned[1])
	}
}

func TestReconcileSmartSuiteRefinedPlansKeepsDirectorAssignmentsCanonical(t *testing.T) {
	requested := completeSmartSuiteTestPlan("smart_desktop_a_plus", 1, "Base")
	requested.AssignedProductFacts = "dual spray mechanism"
	requested.Role = "Core mechanism"
	requested.CTADirection = "Show the dual spray mechanism"
	candidate := completeSmartSuiteTestPlan("smart_desktop_a_plus", 1, "Refined headline")
	candidate.Role = "How the product works"
	candidate.CTADirection = "Explain the mechanism"
	aligned, missing, matched := reconcileSmartSuiteRefinedPlans([]model.EcomSuiteImagePlan{requested}, []model.EcomSuiteImagePlan{candidate})
	if matched != 1 || len(missing) != 0 {
		t.Fatalf("a paraphrased immutable assignment should not invalidate the plan: aligned=%+v matched=%d missing=%+v", aligned, matched, missing)
	}
	if aligned[0].Role != requested.Role || aligned[0].CTADirection != requested.CTADirection || aligned[0].Title != requested.Title {
		t.Fatalf("director assignments were not kept canonical: %+v", aligned[0])
	}
}

func TestSmartSuiteImagePlanRequiresEightGenerationFields(t *testing.T) {
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Headline")
	if !smartSuiteImagePlanComplete(plan) {
		t.Fatal("complete eight-field image plan was rejected")
	}
	plan.LayoutInstruction = ""
	if smartSuiteImagePlanComplete(plan) {
		t.Fatal("image plan without a per-image layout instruction must be rejected")
	}
	plan.LayoutInstruction = "Three-quarter camera and product-safe copy area"
	plan.TypographyInstruction = "Headline above supporting copy with safe margins"
	if !smartSuiteImagePlanComplete(plan) {
		t.Fatal("restored per-image layout and typography instructions should pass")
	}
	plan.Style = ""
	if smartSuiteImagePlanComplete(plan) {
		t.Fatal("image plan without a real scene must be rejected")
	}
}

func TestMergeSmartSuiteRefinedPlanKeepsImageIdentity(t *testing.T) {
	task := model.EcomSuiteTask{ProductName: "Faucet", ProductFeatures: "dual spray"}
	img := model.EcomSuiteImage{ImageType: "smart_secondary", ImageIndex: 2}
	base := buildSmartSuitePlan(task, img, []string{"first", "second"})
	merged := mergeSmartSuiteRefinedPlan(task, img, []string{"first", "second"}, base, model.EcomSuiteImagePlan{
		ImageType: "smart_mobile_a_plus", ImageIndex: 1, Role: "wrong task", Title: "wrong", Subtitle: "wrong",
	})
	if merged.Role != base.Role || merged.Title != base.Title || merged.ImageType != base.ImageType || merged.ImageIndex != base.ImageIndex {
		t.Fatalf("mismatched refinement must be ignored: base=%+v merged=%+v", base, merged)
	}
}

func TestMergeSmartSuiteRefinedPlanAcceptsImageTypeAlias(t *testing.T) {
	task := model.EcomSuiteTask{}
	img := model.EcomSuiteImage{ImageType: "smart_desktop_a_plus", ImageIndex: 1}
	base := completeSmartSuiteTestPlan(img.ImageType, img.ImageIndex, "Base")
	refined := completeSmartSuiteTestPlan("desktop A+", 1, "Refined")
	merged := mergeSmartSuiteRefinedPlan(task, img, nil, base, refined)
	if merged.Title != "Refined" || merged.ImageType != "smart_desktop_a_plus" {
		t.Fatalf("image type aliases should preserve refinement identity: %+v", merged)
	}
}

func TestMergeSmartSuiteRefinedPlanBackfillsOmittedAgentFields(t *testing.T) {
	task := model.EcomSuiteTask{VisualStyle: "A real modern bathroom scene"}
	img := model.EcomSuiteImage{ImageType: "smart_secondary", ImageIndex: 1}
	base := buildSmartSuitePlan(task, img, []string{"Where can it be used?"})
	refined := model.EcomSuiteImagePlan{ImageType: img.ImageType, ImageIndex: img.ImageIndex, Title: "Use It Naturally"}
	merged := mergeSmartSuiteRefinedPlan(task, img, nil, base, refined)
	if !smartSuiteImagePlanComplete(merged) {
		t.Fatalf("omitted refinement fields should be backfilled from base plan: %+v", merged)
	}
	if merged.Title != "Use It Naturally" || merged.Role != base.Role || merged.Prompt != base.Prompt {
		t.Fatalf("unexpected merged plan: base=%+v merged=%+v", base, merged)
	}
}

func TestResolveEcomSuiteTaskStatusReviewDegradedDoesNotFailImages(t *testing.T) {
	status, message := resolveEcomSuiteTaskStatus(6, 6, 0, 0, 0, 6)
	if status != "success" || message != "smart_suite_review_degraded" {
		t.Fatalf("expected successful task with degraded review, got %s/%s", status, message)
	}
	status, message = resolveEcomSuiteTaskStatus(6, 6, 0, 2, 0, 0)
	if status != "partial_failed" || message != "smart_suite_review_unresolved" {
		t.Fatalf("expected unresolved review to be partial failure, got %s/%s", status, message)
	}
	status, message = resolveEcomSuiteTaskStatus(6, 6, 0, 0, 2, 0)
	if status != "partial_failed" || message != "smart_suite_review_correction_failed" {
		t.Fatalf("expected failed correction to be partial failure, got %s/%s", status, message)
	}
}

func TestResolveEcomSuiteTaskStatusDoesNotCompleteWithPendingImages(t *testing.T) {
	status, message := resolveEcomSuiteTaskStatus(20, 0, 0, 0, 0, 0)
	if status != "failed" || message != "image_generation_incomplete" {
		t.Fatalf("zero generated images must be a failed pipeline, got %s/%s", status, message)
	}
	status, message = resolveEcomSuiteTaskStatus(20, 3, 2, 0, 0, 0)
	if status != "partial_failed" || message != "image_generation_incomplete" {
		t.Fatalf("unfinished images must not be treated as a normally completed suite, got %s/%s", status, message)
	}
}

func TestResolveEcomSuiteTaskStatusStopsAfterEmptyStage(t *testing.T) {
	status, message := resolveEcomSuiteTaskStatus(20, 0, 6, 0, 0, 0)
	if status != "failed" || message != "image_generation_incomplete" {
		t.Fatalf("an empty failed stage must stop the suite before later stages, got %s/%s", status, message)
	}
}

func TestSmartSuitePlanningMakesImagesReadyForGeneration(t *testing.T) {
	updates := smartSuitePlanningImageUpdates(model.EcomSuiteImagePlan{
		ImageType:    "smart_secondary",
		ImageIndex:   1,
		Title:        "Material evidence",
		CTADirection: "Prove the material choice",
	})
	if updates["status"] != ecomSuitePlanningReadyStatus {
		t.Fatalf("planned images must be queryable by the generation worker, got status=%v", updates["status"])
	}
	if ecomSuitePlanningReadyStatus != "planning_ready" {
		t.Fatalf("unexpected planning-ready status %q", ecomSuitePlanningReadyStatus)
	}
	if updates["cta_direction"] != "Prove the material choice" {
		t.Fatalf("planned persuasion task must survive the database reload, got %v", updates["cta_direction"])
	}
}

func TestSmartSuiteActiveStatuses(t *testing.T) {
	for _, status := range []string{"pending", "planning", "generating", "reviewing", "correcting"} {
		if !isSmartSuiteActiveStatus(status) {
			t.Fatalf("expected %q to be active", status)
		}
	}
	for _, status := range []string{"success", "partial_failed", "failed", "cancelled"} {
		if isSmartSuiteActiveStatus(status) {
			t.Fatalf("expected %q to be terminal", status)
		}
	}
}

func TestSmartSuiteInitialPlanningStageDoesNotSkipToPromptConfirmation(t *testing.T) {
	task := model.EcomSuiteTask{
		WorkflowType:      "smart_suite",
		Status:            "planning",
		SecondaryCount:    2,
		DesktopAPlusCount: 1,
		Images: []model.EcomSuiteImage{
			{ImageType: "smart_secondary", ImageIndex: 1, Status: "pending"},
			{ImageType: "smart_secondary", ImageIndex: 2, Status: "pending"},
			{ImageType: "smart_desktop_a_plus", ImageIndex: 1, Status: "pending"},
		},
	}
	task.SmartStage = smartSuiteFirstStageName(task, "planning")
	if task.SmartStage != "planning" {
		t.Fatalf("new task must wait for the orchestrator decision, got stage %q", task.SmartStage)
	}
	items, _, _, _ := buildSmartSuiteBigTodoItems(task, nil)
	for _, item := range items {
		if item.ID == "suite-prompts" && item.Status != "pending" {
			t.Fatalf("prompt confirmation must remain pending before its Agent is selected: %+v", item)
		}
		if item.Status == "active" {
			t.Fatalf("no todo step may be active before the orchestrator selects an Agent: %+v", item)
		}
	}
}

func TestSmartSuiteTaskContextCanBeStopped(t *testing.T) {
	taskID := "test-stop-context"
	ctx := startSmartSuiteTaskContext(taskID)
	t.Cleanup(func() { finishSmartSuiteTaskContext(taskID) })
	if ctx.Err() != nil {
		t.Fatalf("new task context should be active: %v", ctx.Err())
	}
	stopSmartSuiteTaskContext(taskID)
	if ctx.Err() != context.Canceled {
		t.Fatalf("expected canceled context, got %v", ctx.Err())
	}
}

func TestSmartSuiteLocalCoordinatorDoesNotSwallowRevisionBeforeRegeneration(t *testing.T) {
	conversation := &model.SmartSuiteConversation{Context: model.SmartSuiteConversationContext{VisualStyle: "现代厨房"}}
	combined := smartSuiteLocalCoordinator(conversation, smartSuiteConversationTurnRequest{Text: "把场景改成美式经典厨房，然后重新生成"})
	if combined.Intent != "collect_requirement" {
		t.Fatalf("combined revision must reach the requirement agent before generation, got %+v", combined)
	}

	standalone := smartSuiteLocalCoordinator(conversation, smartSuiteConversationTurnRequest{Text: "重新生成"})
	if standalone.Intent != "start" {
		t.Fatalf("standalone regeneration command must still start, got %+v", standalone)
	}
}

func TestSmartSuiteRequirementRevisionFallbackAsksForConcreteChange(t *testing.T) {
	output, ok := smartSuiteRequirementRevisionFallback("我想调整一下需求")
	if !ok || !output.NeedsClarification || output.Intent != "collect_requirement" {
		t.Fatalf("generic revision request must ask a stable clarification, got %+v", output)
	}
	if smartSuiteHasProductSemanticUpdate(output.Updates) || smartSuiteHasVisualSemanticUpdate(output.Updates) || smartSuiteHasOtherRequirementUpdate(output.Updates) {
		t.Fatalf("generic revision request must preserve the current context, got %+v", output.Updates)
	}
	if _, ok := smartSuiteRequirementRevisionFallback("把背景换成真实美式厨房"); ok {
		t.Fatal("a concrete revision must be understood by the requirement agent, not the generic fallback")
	}
}

func TestValidateSmartSuiteCoordinatorRejectsSilentGenericRevision(t *testing.T) {
	output := smartSuiteCoordinatorOutput{Intent: "collect_requirement", RequirementKind: "other", Confidence: 0.9}
	err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: "我想修改需求"}, &output)
	if err == nil || !strings.Contains(err.Error(), "revision_detail_not_requested") {
		t.Fatalf("generic revision must not succeed without a clarification: %v", err)
	}
}

func TestSmartSuiteConversationMessagesAreNotTruncated(t *testing.T) {
	conversation := &model.SmartSuiteConversation{}
	for index := 0; index < 85; index++ {
		smartSuiteAppendConversationMessage(conversation, "user", fmt.Sprintf("message-%d", index), "", "", "", nil)
	}
	if len(conversation.Messages) != 85 {
		t.Fatalf("conversation history must be retained, got %d messages", len(conversation.Messages))
	}
	if conversation.Messages[0].Text != "message-0" {
		t.Fatalf("oldest conversation message was removed: %+v", conversation.Messages[0])
	}
}

func TestNormalizeSmartSuiteConversationTaskIDsPreservesEveryRun(t *testing.T) {
	conversation := &model.SmartSuiteConversation{
		TaskID:  "task-3",
		TaskIDs: model.StringSlice{"task-1", "task-2", "task-1"},
	}
	if !normalizeSmartSuiteConversationTaskIDs(conversation) {
		t.Fatal("duplicate task history should be normalized")
	}
	expected := []string{"task-1", "task-2", "task-3"}
	if len(conversation.TaskIDs) != len(expected) {
		t.Fatalf("unexpected task history: %#v", conversation.TaskIDs)
	}
	for index, taskID := range expected {
		if conversation.TaskIDs[index] != taskID {
			t.Fatalf("task history order changed: %#v", conversation.TaskIDs)
		}
	}
}

func TestValidateSmartSuiteCoordinatorAcceptsRevisionAndStartTogether(t *testing.T) {
	current := model.SmartSuiteConversationContext{VisualStyle: "现代厨房"}
	text := "把场景改成美式经典厨房，然后重新生成"
	style := "美式经典厨房"
	output := smartSuiteCoordinatorOutput{
		Intent:          "start",
		RequirementKind: "visual_direction",
		Confidence:      0.96,
		VisualEvidence:  []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "场景改成美式经典厨房"}},
		ClearFields:     []string{"visual_style"},
		Updates:         smartSuiteCoordinatorUpdates{VisualStyle: &style},
	}
	if err := validateSmartSuiteCoordinatorOutput(current, smartSuiteConversationTurnRequest{Text: text}, &output); err != nil {
		t.Fatalf("revision and explicit regeneration should be accepted together: %v", err)
	}
}

func TestSmartSuiteLocalCoordinatorDoesNotClassifySemanticRequirements(t *testing.T) {
	texts := []string{
		"304不锈钢花园水管，三层结构，长度50FT，支持收纳",
		"木质背景，自然光，高级生活化构图",
		"这是浴室龙头，黄铜主体；想要美式经典浴室背景",
		"Amazon 美国站，副图、桌面 A+、移动 A+ 都要，英文文案",
		"使用 gemini-3-pro-image-preview 生成",
	}
	for _, text := range texts {
		output := smartSuiteLocalCoordinator(&model.SmartSuiteConversation{}, smartSuiteConversationTurnRequest{Text: text})
		if smartSuiteHasProductSemanticUpdate(output.Updates) || smartSuiteHasVisualSemanticUpdate(output.Updates) || smartSuiteHasOtherRequirementUpdate(output.Updates) {
			t.Fatalf("local coordinator must not classify semantic requirement %q, got %+v", text, output.Updates)
		}
	}
}

func TestValidateSmartSuiteCoordinatorAcceptsProductFactsWithEvidence(t *testing.T) {
	text := "304不锈钢花园水管，三层结构，长度50FT，支持收纳"
	name := "花园水管"
	facts := text
	output := smartSuiteCoordinatorOutput{
		Intent:          "collect_requirement",
		RequirementKind: "product_facts",
		Confidence:      0.96,
		ProductEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "三层结构，长度50FT"}},
		Updates:         smartSuiteCoordinatorUpdates{ProductName: &name, ProductFeatures: &facts},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: text}, &output); err != nil {
		t.Fatalf("valid product classification rejected: %v", err)
	}
	ctx := model.SmartSuiteConversationContext{}
	applySmartSuiteCoordinatorUpdates(&ctx, output.Updates)
	if ctx.ProductName != name || ctx.ProductFeatures != facts || ctx.VisualStyle != "" {
		t.Fatalf("unexpected product context: %+v", ctx)
	}
}

func TestValidateSmartSuiteCoordinatorAcceptsVisualDirectionWithEvidence(t *testing.T) {
	text := "木质背景，自然光，高级生活化构图"
	style := text
	output := smartSuiteCoordinatorOutput{
		Intent:          "collect_requirement",
		RequirementKind: "visual_direction",
		Confidence:      0.94,
		VisualEvidence:  []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}},
		Updates:         smartSuiteCoordinatorUpdates{VisualStyle: &style},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: text, ImageURL: "/uploads/product.png"}, &output); err != nil {
		t.Fatalf("valid visual classification rejected: %v", err)
	}
	ctx := model.SmartSuiteConversationContext{}
	applySmartSuiteCoordinatorUpdates(&ctx, output.Updates)
	if ctx.VisualStyle != text || ctx.ProductName != "" || ctx.ProductFeatures != "" {
		t.Fatalf("unexpected visual context: %+v", ctx)
	}
}

func TestValidateSmartSuiteCoordinatorEnforcesImagePurpose(t *testing.T) {
	productName := "浴室龙头"
	visualStyle := "美式经典浴室，自然光"
	productOutput := smartSuiteCoordinatorOutput{
		RequirementKind: "product_facts", Confidence: 0.95,
		ProductEvidence: []smartSuiteCoordinatorEvidence{{Source: "image", Quote: "图中可见金属双把手龙头"}},
		Updates:         smartSuiteCoordinatorUpdates{ProductName: &productName},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{ImageURL: "/uploads/product.png", ImagePurpose: "product_image"}, &productOutput); err != nil {
		t.Fatalf("product image evidence rejected: %v", err)
	}
	productOutputFromStyleRef := productOutput
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{ImageURL: "/uploads/style.png", ImagePurpose: "style_reference"}, &productOutputFromStyleRef); err == nil {
		t.Fatal("style reference must not support product facts")
	}

	visualOutput := smartSuiteCoordinatorOutput{
		RequirementKind: "visual_direction", Confidence: 0.95,
		VisualEvidence: []smartSuiteCoordinatorEvidence{{Source: "image", Quote: "参考图为美式经典浴室和自然光"}},
		Updates:        smartSuiteCoordinatorUpdates{VisualStyle: &visualStyle},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{ImageURL: "/uploads/style.png", ImagePurpose: "style_reference"}, &visualOutput); err != nil {
		t.Fatalf("style reference evidence rejected: %v", err)
	}
	visualOutputFromProduct := visualOutput
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{ImageURL: "/uploads/product.png", ImagePurpose: "product_image"}, &visualOutputFromProduct); err == nil {
		t.Fatal("product image must not support a requested visual direction")
	}
}

func TestValidateSmartSuiteCoordinatorAcceptsMixedRequirement(t *testing.T) {
	text := "这是浴室台盆龙头，黄铜主体；套图用美式经典浴室背景和自然光"
	name := "浴室台盆龙头"
	facts := "浴室台盆龙头，黄铜主体"
	style := "美式经典浴室背景和自然光"
	output := smartSuiteCoordinatorOutput{
		Intent:          "collect_requirement",
		RequirementKind: "mixed",
		Confidence:      0.93,
		ProductEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "这是浴室台盆龙头，黄铜主体"}},
		VisualEvidence:  []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "套图用美式经典浴室背景和自然光"}},
		Updates: smartSuiteCoordinatorUpdates{
			ProductName: &name, ProductFeatures: &facts, VisualStyle: &style,
		},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: text}, &output); err != nil {
		t.Fatalf("valid mixed classification rejected: %v", err)
	}
}

func TestValidateSmartSuiteCoordinatorAcceptsProductAndOtherRequirement(t *testing.T) {
	text := "这是黄铜浴室龙头，用于 Amazon 美国站，整套都要"
	name := "浴室龙头"
	facts := "黄铜浴室龙头"
	platform := "amazon"
	countSix, countSeven := 6, 7
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "mixed",
		Confidence:      0.95,
		ProductEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "这是黄铜浴室龙头"}},
		OtherEvidence:   []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "Amazon 美国站，整套都要"}},
		Updates: smartSuiteCoordinatorUpdates{
			ProductName: &name, ProductFeatures: &facts, Platform: &platform,
			SecondaryCount: &countSix, DesktopAPlusCount: &countSeven, MobileAPlusCount: &countSeven,
		},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: text}, &output); err != nil {
		t.Fatalf("valid product and workflow requirements rejected: %v", err)
	}
}

func TestValidateSmartSuiteCoordinatorRejectsUnsupportedOrUncertainUpdates(t *testing.T) {
	style := "高级棚拍"
	tests := []smartSuiteCoordinatorOutput{
		{
			RequirementKind: "visual_direction", Confidence: 0.9,
			VisualEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "用户没有说过的内容"}},
			Updates:        smartSuiteCoordinatorUpdates{VisualStyle: &style},
		},
		{
			RequirementKind: "visual_direction", Confidence: 0.4,
			VisualEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "自然光"}},
			Updates:        smartSuiteCoordinatorUpdates{VisualStyle: &style},
		},
	}
	for index := range tests {
		if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: "自然光"}, &tests[index]); err == nil {
			t.Fatalf("unsafe semantic update %d must be rejected", index)
		}
	}
}

func TestValidateSmartSuiteCoordinatorKeepsRequirementKindsExclusive(t *testing.T) {
	product := "浴室龙头"
	platform := "amazon"
	style := "美式浴室"
	count := 6
	tests := []smartSuiteCoordinatorOutput{
		{
			RequirementKind: "product_facts", Confidence: 0.95,
			ProductEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "浴室龙头"}},
			OtherEvidence:   []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "Amazon"}},
			Updates:         smartSuiteCoordinatorUpdates{ProductName: &product, Platform: &platform},
		},
		{
			RequirementKind: "visual_direction", Confidence: 0.95,
			VisualEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "美式浴室"}},
			OtherEvidence:  []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "6张副图"}},
			Updates:        smartSuiteCoordinatorUpdates{VisualStyle: &style, SecondaryCount: &count},
		},
	}
	for index := range tests {
		if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: "浴室龙头，美式浴室，Amazon，6张副图"}, &tests[index]); err == nil {
			t.Fatalf("requirement kind %d accepted updates from another category", index)
		}
	}
}

func TestValidateSmartSuiteCoordinatorAmbiguityDoesNotWriteSemanticContext(t *testing.T) {
	name := "错误产品"
	style := "错误风格"
	output := smartSuiteCoordinatorOutput{
		RequirementKind:       "ambiguous",
		Confidence:            0.4,
		NeedsClarification:    true,
		ClarificationQuestion: "这是产品使用环境，还是希望生成的图片场景？",
		ClearFields:           []string{"product_name"},
		Updates:               smartSuiteCoordinatorUpdates{ProductName: &name, VisualStyle: &style},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: "放浴室里"}, &output); err != nil {
		t.Fatalf("ambiguous classification should produce clarification, got %v", err)
	}
	if smartSuiteHasProductSemanticUpdate(output.Updates) || smartSuiteHasVisualSemanticUpdate(output.Updates) || len(output.ClearFields) != 0 {
		t.Fatalf("ambiguous output must not mutate semantic context: %+v", output)
	}
	reply := smartSuiteCoordinatorReply(model.SmartSuiteConversation{}, output, false)
	if reply != output.ClarificationQuestion {
		t.Fatalf("expected agent clarification question, got %q", reply)
	}
}

func TestSmartSuiteAgentCanSupplementAndCorrectCollectedContext(t *testing.T) {
	current := model.SmartSuiteConversationContext{
		ProductName: "浴室龙头", ProductFeatures: "黄铜主体", VisualStyle: "白底棚拍",
	}
	name := "三孔浴室台盆龙头"
	facts := "三孔浴室台盆龙头，黄铜主体，双把手，支持冷热出水"
	style := "美式经典浴室，自然光"
	local := smartSuiteCoordinatorOutput{Intent: "collect_requirement"}
	ai := smartSuiteCoordinatorOutput{
		Intent:          "collect_requirement",
		RequirementKind: "mixed",
		Confidence:      0.95,
		ProductEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "产品名称改为三孔浴室台盆龙头，补充双把手和冷热出水"}},
		VisualEvidence:  []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "视觉改为美式经典浴室，自然光"}},
		Updates: smartSuiteCoordinatorUpdates{
			ProductName: &name, ProductFeatures: &facts, VisualStyle: &style,
		},
		ClearFields: []string{"product_name", "visual_style"},
	}
	req := smartSuiteConversationTurnRequest{Text: "产品名称改为三孔浴室台盆龙头，补充双把手和冷热出水；视觉改为美式经典浴室，自然光"}
	if err := validateSmartSuiteCoordinatorOutput(current, req, &ai); err != nil {
		t.Fatalf("valid correction rejected: %v", err)
	}
	merged := mergeSmartSuiteCoordinatorOutput(local, ai)
	applySmartSuiteCoordinatorClears(&current, merged.ClearFields)
	applySmartSuiteCoordinatorUpdates(&current, merged.Updates)
	if current.ProductName != name || current.ProductFeatures != facts || current.VisualStyle != style {
		t.Fatalf("agent correction must update collected semantic context: %+v", current)
	}
}

func TestValidateSmartSuiteCoordinatorRequiresCompleteSupplement(t *testing.T) {
	current := model.SmartSuiteConversationContext{ProductName: "浴室龙头", ProductFeatures: "黄铜主体，三孔安装"}
	newOnly := "支持冷热出水"
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "product_facts",
		Confidence:      0.95,
		ProductEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: newOnly}},
		Updates:         smartSuiteCoordinatorUpdates{ProductFeatures: &newOnly},
	}
	if err := validateSmartSuiteCoordinatorOutput(current, smartSuiteConversationTurnRequest{Text: newOnly}, &output); err == nil {
		t.Fatal("agent must return the complete merged product document when supplementing facts")
	}
	mergedFacts := "黄铜主体，三孔安装，支持冷热出水"
	output.Updates.ProductFeatures = &mergedFacts
	if err := validateSmartSuiteCoordinatorOutput(current, smartSuiteConversationTurnRequest{Text: newOnly}, &output); err != nil {
		t.Fatalf("complete merged product document rejected: %v", err)
	}
}

func TestValidateSmartSuiteCoordinatorRequiresExplicitCorrectionToOverwrite(t *testing.T) {
	current := model.SmartSuiteConversationContext{ProductName: "浴室龙头", ProductFeatures: "黄铜主体", VisualStyle: "白底棚拍"}
	newName := "厨房龙头"
	newStyle := "现代厨房场景"
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "mixed",
		Confidence:      0.95,
		ProductEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "产品其实是厨房龙头"}},
		VisualEvidence:  []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "视觉改成现代厨房场景"}},
		Updates:         smartSuiteCoordinatorUpdates{ProductName: &newName, VisualStyle: &newStyle},
	}
	req := smartSuiteConversationTurnRequest{Text: "产品其实是厨房龙头，视觉改成现代厨房场景"}
	if err := validateSmartSuiteCoordinatorOutput(current, req, &output); err == nil {
		t.Fatal("overwriting collected semantic context must require explicit clear_fields")
	}
	output.ClearFields = []string{"product_name", "visual_style"}
	if err := validateSmartSuiteCoordinatorOutput(current, req, &output); err != nil {
		t.Fatalf("explicit semantic correction rejected: %v", err)
	}
}

func TestSmartSuiteAgentCanRepairPreviouslyMisclassifiedVisualStyle(t *testing.T) {
	productText := "浴室台盆龙头，黄铜主体，三孔安装，支持冷热出水"
	current := model.SmartSuiteConversationContext{VisualStyle: productText}
	name := "浴室台盆龙头"
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "product_facts",
		Confidence:      0.97,
		ProductEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: productText}},
		ClearFields:     []string{"visual_style"},
		Updates: smartSuiteCoordinatorUpdates{
			ProductName: &name, ProductFeatures: &productText,
		},
	}
	req := smartSuiteConversationTurnRequest{Text: productText}
	if err := validateSmartSuiteCoordinatorOutput(current, req, &output); err != nil {
		t.Fatalf("valid reclassification rejected: %v", err)
	}
	applySmartSuiteCoordinatorClears(&current, output.ClearFields)
	applySmartSuiteCoordinatorUpdates(&current, output.Updates)
	if current.VisualStyle != "" || current.ProductName != name || current.ProductFeatures != productText {
		t.Fatalf("misclassified product document was not repaired: %+v", current)
	}
}

func TestSmartSuiteAgentCannotClearUnrelatedVisualStyleDuringProductTurn(t *testing.T) {
	current := model.SmartSuiteConversationContext{VisualStyle: "美式经典浴室，自然光"}
	productText := "浴室台盆龙头，黄铜主体"
	name := "浴室台盆龙头"
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "product_facts",
		Confidence:      0.97,
		ProductEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: productText}},
		ClearFields:     []string{"visual_style"},
		Updates:         smartSuiteCoordinatorUpdates{ProductName: &name, ProductFeatures: &productText},
	}
	if err := validateSmartSuiteCoordinatorOutput(current, smartSuiteConversationTurnRequest{Text: productText}, &output); err == nil {
		t.Fatal("product turn must not clear an unrelated visual direction")
	}
}

func TestSmartSuiteConversationAsksOnlyForMissingInformation(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png",
		ProductName:     "Garden hose",
		ProductFeatures: "304 stainless steel outer shell",
		Platform:        "amazon",
		CountryCode:     "US",
		CountryName:     "美国站",
		Language:        "en",
		LanguageName:    "英文",
		SecondaryCount:  6,
		VisualStyle:     "真实生活场景",
		DrawingModel:    "gemini-3-pro-image-preview",
	}
	normalizeSmartSuiteConversationContext(&ctx)
	missing := smartSuiteMissingFields(ctx)
	if len(missing) != 0 {
		t.Fatalf("expected explicitly collected language to make context complete, missing=%v context=%+v", missing, ctx)
	}
	if ctx.Language != "en" || ctx.LanguageName != "英文" || ctx.OutputTarget != "secondary" {
		t.Fatalf("unexpected normalized defaults: %+v", ctx)
	}
}

func TestSmartSuiteConversationRequiresVisualStyleBeforeReady(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", CountryCode: "US", CountryName: "United States", SecondaryCount: 6,
	}
	normalizeSmartSuiteConversationContext(&ctx)
	if smartSuiteHasVisualStyle(ctx) {
		t.Fatal("conversation must ask for visual style before generation")
	}
	style := "真实户外庭院，自然暖光，不要白底"
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "visual_direction",
		Confidence:      0.95,
		VisualEvidence:  []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: style}},
		Updates:         smartSuiteCoordinatorUpdates{VisualStyle: &style},
	}
	if err := validateSmartSuiteCoordinatorOutput(ctx, smartSuiteConversationTurnRequest{Text: style}, &output); err != nil {
		t.Fatalf("valid visual direction rejected: %v", err)
	}
	applySmartSuiteCoordinatorUpdates(&ctx, output.Updates)
	if !smartSuiteHasVisualStyle(ctx) || ctx.VisualStyle != style {
		t.Fatalf("expected agent style requirement to be retained, got %+v", ctx)
	}
}

func TestSmartSuiteCoordinatorUsesPremiumDefaultAfterModelSelection(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", CountryCode: "US", SecondaryCount: 6, Language: "en", LanguageName: "英文", DrawingModel: "gemini-3-pro-image-preview",
	}
	normalizeSmartSuiteConversationContext(&ctx)
	output := smartSuiteLocalCoordinator(&model.SmartSuiteConversation{Context: ctx}, smartSuiteConversationTurnRequest{Text: "产品信息补充完成"})
	applySmartSuiteCoordinatorUpdates(&ctx, output.Updates)
	reply := smartSuiteCoordinatorReply(model.SmartSuiteConversation{Context: ctx}, output, false)
	if strings.Contains(reply, "视觉要求") || strings.Contains(reply, "交给 AI") || !strings.Contains(reply, "AI 根据产品自主制定高端风格") {
		t.Fatalf("model selection must proceed directly with the premium AI default, got %q", reply)
	}
}

func TestSmartSuiteConversationMigratesLegacyAutoVisualStyle(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{VisualStyle: smartSuiteLegacyAutoVisualStyle}
	normalizeSmartSuiteConversationContext(&ctx)
	if ctx.VisualStyle != smartSuiteAutoVisualStyle {
		t.Fatalf("legacy automatic style should migrate to the high-end default: %q", ctx.VisualStyle)
	}
}

func TestSmartSuitePlatformDeterminesCopyLanguageAndOptions(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", CountryCode: "US", CountryName: "美国站", SecondaryCount: 6,
	}
	normalizeSmartSuiteConversationContext(&ctx)
	if ctx.Language != "en" || ctx.LanguageName != "英文" {
		t.Fatalf("platform should determine default copy language: %+v", ctx)
	}
	for _, field := range smartSuiteMissingFields(ctx) {
		if field == "language" {
			t.Fatalf("platform-selected conversation must not ask for language: %+v", ctx)
		}
	}
	options := smartSuiteConversationSuggestions(ctx, "collect_requirement", false)
	for _, option := range options {
		if strings.Contains(strings.ToLower(option), "shopify") {
			t.Fatalf("Shopify must not be offered as a platform option: %#v", options)
		}
	}
}

func TestSmartSuiteEuropeanPlatformDefaultsToEnglishCopy(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{Platform: "amazon", CountryCode: "EU", CountryName: "欧洲站"}
	normalizeSmartSuiteConversationContext(&ctx)
	if ctx.Language != "en" || ctx.LanguageName != "英文" {
		t.Fatalf("European aggregate site should use English as default copy language: %+v", ctx)
	}
}

func TestSmartSuiteCoordinatorAsksProductInfoImmediatelyAfterBareProductImage(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{WhiteBgImageURL: "/uploads/product.png"}
	output := smartSuiteLocalCoordinator(&model.SmartSuiteConversation{Context: ctx}, smartSuiteConversationTurnRequest{Text: "图片收到了"})
	applySmartSuiteCoordinatorUpdates(&ctx, output.Updates)
	reply := smartSuiteCoordinatorReply(model.SmartSuiteConversation{Context: ctx}, output, false)
	if !strings.Contains(reply, "产品名称") || strings.Contains(reply, "确认整套图的视觉方向") {
		t.Fatalf("product information must be requested before visual style, got %q", reply)
	}
	options := smartSuiteConversationSuggestions(ctx, output.Intent, false)
	if len(options) != 0 {
		t.Fatalf("style suggestions must wait until product information is complete, got %#v", options)
	}
}

func TestSmartSuiteConversationSuggestionsFollowMissingRequirement(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", CountryCode: "US", SecondaryCount: 6, Language: "en", LanguageName: "英文", DrawingModel: "gemini-3-pro-image-preview",
	}
	normalizeSmartSuiteConversationContext(&ctx)
	startOptions := smartSuiteConversationSuggestions(ctx, "collect_requirement", false)
	if len(startOptions) != 2 || startOptions[0] != "开始生成" {
		t.Fatalf("default premium style must not require a style selection: %#v", startOptions)
	}
	ctx.VisualStyle = "真实户外庭院，自然暖光"
	ctx.DrawingModel = "gemini-3-pro-image-preview"
	normalizeSmartSuiteConversationContext(&ctx)
	startOptions = smartSuiteConversationSuggestions(ctx, "collect_requirement", false)
	if len(startOptions) != 2 || startOptions[0] != "开始生成" {
		t.Fatalf("unexpected ready suggestions: %#v", startOptions)
	}
}

func TestSmartSuiteSuggestedVisualAnswerSurvivesAgentFailure(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", CountryCode: "US", SecondaryCount: 6, Language: "en", LanguageName: "英文",
	}
	normalizeSmartSuiteConversationContext(&ctx)
	req := smartSuiteConversationTurnRequest{Text: "高级品牌场景", AnswerField: "visual_style"}
	output, ok := smartSuiteSuggestedAnswerFallback(ctx, req)
	if !ok {
		t.Fatal("verified visual suggestion must remain usable when the agent call fails")
	}
	applySmartSuiteCoordinatorUpdates(&ctx, output.Updates)
	if ctx.VisualStyle != "高级品牌场景" {
		t.Fatalf("expected selected visual style to be stored, got %+v", ctx)
	}
}

func TestSmartSuiteTypedVisualAnswerSurvivesAgentFailure(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", CountryCode: "US", SecondaryCount: 6, Language: "en", LanguageName: "英文",
	}
	normalizeSmartSuiteConversationContext(&ctx)
	text := "浅色现代厨房场景，窗边自然光，画面干净但要有真实生活感"
	output, ok := smartSuiteSuggestedAnswerFallback(ctx, smartSuiteConversationTurnRequest{Text: text, AnswerField: "visual_style"})
	if !ok {
		t.Fatal("typed visual answer must remain usable when the agent call fails")
	}
	applySmartSuiteCoordinatorUpdates(&ctx, output.Updates)
	if ctx.VisualStyle != text {
		t.Fatalf("expected typed visual style to be stored verbatim, got %+v", ctx)
	}
}

func TestSmartSuiteNoStylePreferenceLetsAIDecide(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", CountryCode: "US", SecondaryCount: 6, DrawingModel: "gemini-3-pro-image-preview",
	}
	for _, text := range []string{"没有特别要求", "交给 AI", "AI 决定"} {
		output, ok := smartSuiteSuggestedAnswerFallback(ctx, smartSuiteConversationTurnRequest{Text: text, AnswerField: "visual_style"})
		if !ok || output.Updates.VisualStyle == nil || *output.Updates.VisualStyle != smartSuiteAutoVisualStyle {
			t.Fatalf("auto-style answer %q was not understood: %+v", text, output)
		}
	}
}

func TestSmartSuiteStartAtStyleQuestionAuthorizesAIStyle(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", CountryCode: "US", SecondaryCount: 6, DrawingModel: "gemini-3-pro-image-preview",
	}
	conversation := &model.SmartSuiteConversation{Context: ctx}
	output := smartSuiteLocalCoordinator(conversation, smartSuiteConversationTurnRequest{Text: "开始生成"})
	if output.Intent != "start" || output.Updates.VisualStyle == nil || *output.Updates.VisualStyle != smartSuiteAutoVisualStyle {
		t.Fatalf("starting without a preference must authorize autonomous style: %+v", output)
	}
}

func TestSmartSuiteTypedVisualFallbackRejectsFeedbackAndUnsupportedBackground(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Kitchen faucet", ProductFeatures: "dual spray modes",
		Platform: "amazon", CountryCode: "US", SecondaryCount: 6,
	}
	for _, text := range []string{"为什么一定要点按钮，你没有理解我的输入", "整套图片都保持纯白背景"} {
		if _, ok := smartSuiteSuggestedAnswerFallback(ctx, smartSuiteConversationTurnRequest{Text: text, AnswerField: "visual_style"}); ok {
			t.Fatalf("feedback or unsupported final background must not be stored as visual style: %q", text)
		}
	}
	mixed := "这是黄铜厨房龙头，套图希望使用现代厨房背景和自然光"
	if _, ok := smartSuiteSuggestedAnswerFallback(ctx, smartSuiteConversationTurnRequest{Text: mixed, AnswerField: "visual_style"}); ok {
		t.Fatalf("a mixed product-and-visual requirement must not be collapsed into visual style: %q", mixed)
	}
	valid := "现代厨房场景，自然光，不要白底"
	if _, ok := smartSuiteSuggestedAnswerFallback(ctx, smartSuiteConversationTurnRequest{Text: valid, AnswerField: "visual_style"}); !ok {
		t.Fatalf("valid scene-based style with a white-background exclusion should pass: %q", valid)
	}
}

func TestSmartSuiteProductUseSceneRemainsProductFact(t *testing.T) {
	text := "产品名称：厨房龙头；适用场景：厨房水槽；支持冷热双模式出水"
	if !smartSuiteLooksLikeProductFacts(text) || smartSuiteLooksLikeExplicitVisualDirection(text) {
		t.Fatalf("a product use scene must remain a product fact, got %q", text)
	}
	ctx := model.SmartSuiteConversationContext{WhiteBgImageURL: "/uploads/product.png"}
	output, ok := smartSuiteSuggestedAnswerFallback(ctx, smartSuiteConversationTurnRequest{Text: text, AnswerField: "product_info"})
	if !ok || output.Updates.ProductName == nil || *output.Updates.ProductName != "厨房龙头" {
		t.Fatalf("product use-scene document was not retained correctly: %+v", output)
	}
}

func TestSmartSuiteTypedProductDocumentSurvivesAgentFailure(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{WhiteBgImageURL: "/uploads/product.png"}
	text := "Forious kitchen faucet with a high arc gooseneck design, 15.27 in height.\nDual spray and stream modes with an 18-inch pull-out hose.\nDiamond-coated ceramic valve core tested 500,000+ times."
	output, ok := smartSuiteSuggestedAnswerFallback(ctx, smartSuiteConversationTurnRequest{Text: text, AnswerField: "product_info"})
	if !ok {
		t.Fatal("typed product document must remain usable when the agent call fails")
	}
	applySmartSuiteCoordinatorUpdates(&ctx, output.Updates)
	if ctx.ProductName != "Forious kitchen faucet" || ctx.ProductFeatures != text {
		t.Fatalf("expected product name and full document to be retained, got %+v", ctx)
	}
	if ctx.VisualStyle != "" {
		t.Fatalf("product copy must not be stored as visual style: %+v", ctx)
	}
}

func TestSmartSuiteBareImageSuggestionFieldDoesNotBypassProductTextContext(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{WhiteBgImageURL: "/uploads/product.png"}
	text := "Forious kitchen faucet with a high arc design, dual spray modes and an 18-inch pull-out hose."
	req := smartSuiteConversationTurnRequest{Text: text, ImageURL: "/uploads/product.png", ImagePurpose: "product_image", AnswerField: "white_bg_image"}
	req.AnswerField = smartSuiteConversationAnswerField(ctx, req)
	if req.AnswerField != "product_info" {
		t.Fatalf("product image plus product text must use product_info context, got %q", req.AnswerField)
	}
	if _, ok := smartSuiteSuggestedAnswerFallback(ctx, req); !ok {
		t.Fatal("product image plus product text must have a recoverable product-info fallback")
	}
}

func TestSmartSuiteBackendInfersVisualAnswerFieldWithoutFrontendHint(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Forious kitchen faucet", ProductFeatures: "dual spray modes",
		Platform: "amazon", CountryCode: "US", SecondaryCount: 6, DrawingModel: "gemini-3-pro-image-preview",
	}
	req := smartSuiteConversationTurnRequest{Text: "美式现代经典风格厨房场景"}
	req.AnswerField = smartSuiteConversationAnswerField(ctx, req)
	if req.AnswerField != "visual_style" {
		t.Fatalf("backend must infer the active visual question, got %q", req.AnswerField)
	}
	output, ok := smartSuiteSuggestedAnswerFallback(ctx, req)
	if !ok || output.Updates.VisualStyle == nil || *output.Updates.VisualStyle != req.Text {
		t.Fatalf("typed visual answer must work without a frontend field hint: %+v", output)
	}
}

func TestSmartSuiteRepairsLastFailedTypedVisualAnswerOnRestore(t *testing.T) {
	conversation := &model.SmartSuiteConversation{
		Context: model.SmartSuiteConversationContext{
			WhiteBgImageURL: "/uploads/product.png", ProductName: "Forious kitchen faucet", ProductFeatures: "dual spray modes",
			Platform: "amazon", CountryCode: "US", SecondaryCount: 6, DrawingModel: "gemini-3-pro-image-preview",
		},
		Messages: []model.SmartSuiteConversationMessage{
			{Role: "assistant", Text: "产品信息记下了。接下来确认整套图的视觉方向。"},
			{Role: "user", Text: "美式现代经典风格厨房场景"},
			{Role: "assistant", Text: smartSuiteCoordinatorFailureReply},
		},
	}
	if !repairSmartSuiteFailedFocusedAnswer(conversation) {
		t.Fatal("restoring the conversation must repair the last recoverable failed answer")
	}
	if conversation.Context.VisualStyle != "美式现代经典风格厨房场景" {
		t.Fatalf("failed visual answer was not restored: %+v", conversation.Context)
	}
	last := conversation.Messages[len(conversation.Messages)-1]
	if last.Text == smartSuiteCoordinatorFailureReply || strings.Contains(last.Text, "暂时没有成功") {
		t.Fatalf("failure reply must be replaced after recovery: %+v", last)
	}
	if repairSmartSuiteFailedFocusedAnswer(conversation) {
		t.Fatal("a repaired conversation must not replay the same message twice")
	}
}

func TestSmartSuiteRestoreDoesNotReplayOrdinaryHistory(t *testing.T) {
	conversation := &model.SmartSuiteConversation{
		Context: model.SmartSuiteConversationContext{WhiteBgImageURL: "/uploads/product.png"},
		Messages: []model.SmartSuiteConversationMessage{
			{Role: "user", Text: "Amazon 美国站"},
			{Role: "assistant", Text: "请继续补充产品信息。"},
		},
	}
	if repairSmartSuiteFailedFocusedAnswer(conversation) {
		t.Fatal("ordinary conversation history must never be replayed on restore")
	}
}

func TestSmartSuiteFocusedFallbackDoesNotTreatWorkflowTextAsProductInfo(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{WhiteBgImageURL: "/uploads/product.png"}
	for _, text := range []string{"做 Amazon 美国站，整套都要", "先生成 6 张副图", "使用 gemini 模型"} {
		if _, ok := smartSuiteSuggestedAnswerFallback(ctx, smartSuiteConversationTurnRequest{Text: text, AnswerField: "product_info"}); ok {
			t.Fatalf("workflow requirement must not be stored as product information: %q", text)
		}
	}
}

func TestSmartSuiteFocusedFallbackDoesNotOverwriteValidAgentExtraction(t *testing.T) {
	name := "Forious Pull-Down Kitchen Faucet"
	facts := "15.27-inch high-arc kitchen faucet with dual spray modes"
	ai := smartSuiteCoordinatorOutput{
		RequirementKind: "product_facts",
		Updates:         smartSuiteCoordinatorUpdates{ProductName: &name, ProductFeatures: &facts},
	}
	raw := "Simple And Stylish Design: Forious kitchen faucet with a high arc gooseneck design.\nDual Mode Settings: spray and stream."
	fallback := smartSuiteCoordinatorOutput{
		RequirementKind: "product_facts",
		Updates: smartSuiteCoordinatorUpdates{
			ProductName: stringPtr("Simple And Stylish Design"), ProductFeatures: &raw,
		},
	}
	merged := mergeSmartSuiteCoordinatorFocusedFallback(ai, fallback)
	if *merged.Updates.ProductName != name || *merged.Updates.ProductFeatures != facts {
		t.Fatalf("a valid agent extraction must take precedence over the raw-text fallback: %+v", merged.Updates)
	}
}

func TestSmartSuiteProductInfoAnswerFieldTracksMissingProductFields(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{WhiteBgImageURL: "/uploads/product.png"}
	if field := smartSuiteConversationSuggestionField(ctx, "collect_requirement", false); field != "product_info" {
		t.Fatalf("expected product question to expose product_info context, got %q", field)
	}
	ctx.ProductName = "Kitchen faucet"
	text := "Dual spray and stream modes with an 18-inch pull-out hose."
	output, ok := smartSuiteSuggestedAnswerFallback(ctx, smartSuiteConversationTurnRequest{Text: text, AnswerField: "product_info"})
	if !ok {
		t.Fatal("product_info must remain active while product_features is missing")
	}
	applySmartSuiteCoordinatorUpdates(&ctx, output.Updates)
	if ctx.ProductName != "Kitchen faucet" || ctx.ProductFeatures != text {
		t.Fatalf("existing product name must be preserved while facts are added: %+v", ctx)
	}
}

func TestSmartSuiteFocusedFallbackPreservesOtherAgentUpdates(t *testing.T) {
	platform := "amazon"
	ai := smartSuiteCoordinatorOutput{
		RequirementKind: "other",
		Updates:         smartSuiteCoordinatorUpdates{Platform: &platform},
		PublicSummary:   "已识别平台。",
		PublicActions:   []string{"将平台设为 Amazon"},
	}
	style := "真实现代厨房，自然光"
	fallback := smartSuiteCoordinatorOutput{
		RequirementKind: "visual_direction",
		VisualEvidence:  []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: style}},
		Updates:         smartSuiteCoordinatorUpdates{VisualStyle: &style},
	}
	merged := mergeSmartSuiteCoordinatorFocusedFallback(ai, fallback)
	if !nonEmptySmartSuiteString(merged.Updates.Platform) || !nonEmptySmartSuiteString(merged.Updates.VisualStyle) {
		t.Fatalf("focused fallback must not discard valid agent updates: %+v", merged)
	}
	if merged.RequirementKind != "mixed" || merged.PublicSummary != ai.PublicSummary {
		t.Fatalf("focused fallback must retain the agent report and normalize mixed semantics: %+v", merged)
	}
}

func TestSmartSuiteSuggestedAnswerRejectsUnverifiedTextOrWrongQuestion(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", CountryCode: "US", SecondaryCount: 6, Language: "en", LanguageName: "英文", DrawingModel: "gemini-3-pro-image-preview",
	}
	normalizeSmartSuiteConversationContext(&ctx)
	for _, req := range []smartSuiteConversationTurnRequest{
		{Text: "产品名称是厨房龙头，材质黄铜，支持双模式出水", AnswerField: "visual_style"},
		{Text: "高级品牌场景", AnswerField: "product_name"},
		{Text: "Amazon 美国站", AnswerField: "platform_site"},
	} {
		if _, ok := smartSuiteSuggestedAnswerFallback(ctx, req); ok {
			t.Fatalf("unverified or context-mismatched answer must not bypass the agent: %+v", req)
		}
	}
}

func TestSmartSuiteSuggestionFieldMatchesDisplayedQuestion(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", CountryCode: "US", SecondaryCount: 6, Language: "en", LanguageName: "英文", DrawingModel: "gemini-3-pro-image-preview",
	}
	normalizeSmartSuiteConversationContext(&ctx)
	if field := smartSuiteConversationSuggestionField(ctx, "collect_requirement", false); field != "" {
		t.Fatalf("ready conversation must not ask for a style selection, got %q", field)
	}
	ctx.VisualStyle = "高级品牌场景"
	ctx.Platform = ""
	ctx.CountryCode = ""
	if field := smartSuiteConversationSuggestionField(ctx, "collect_requirement", false); field != "platform_site" {
		t.Fatalf("expected platform suggestions to identify platform_site, got %q", field)
	}
	options := smartSuiteConversationSuggestions(ctx, "collect_requirement", false)
	joined := strings.Join(options, "|")
	for _, expected := range []string{"Wayfair 美国站", "Lowe's 美国站"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("platform options missing %q: %#v", expected, options)
		}
	}
	if strings.Contains(strings.ToLower(joined), "ebay") || strings.Contains(strings.ToLower(joined), "shopify") {
		t.Fatalf("removed platforms must not be displayed: %#v", options)
	}
}

func TestSmartSuiteCoordinatorOutputCannotDropClickedSuggestion(t *testing.T) {
	output := smartSuiteCoordinatorOutput{RequirementKind: "ambiguous", NeedsClarification: true}
	if smartSuiteCoordinatorOutputAnswersField(output, "visual_style") {
		t.Fatal("ambiguous coordinator output must not count as a visual answer")
	}
	style := "高级品牌场景"
	output.Updates.VisualStyle = &style
	if !smartSuiteCoordinatorOutputAnswersField(output, "visual_style") {
		t.Fatal("visual update must count as answering the visual suggestion")
	}
}

func TestSmartSuiteConversationOmitsProductUploadSuggestion(t *testing.T) {
	options := smartSuiteConversationSuggestions(model.SmartSuiteConversationContext{}, "collect_requirement", false)
	if len(options) != 0 {
		t.Fatalf("product upload should use the composer attachment control, got suggestions: %#v", options)
	}
}

func TestSmartSuiteConfirmationIncludesVisualStyle(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", PlatformName: "Amazon", CountryCode: "US", CountryName: "美国站", SecondaryCount: 6,
		Language: "en", LanguageName: "英文", VisualStyle: "真实户外庭院，自然暖光",
		DrawingModel: "gemini-3-pro-image-preview",
	}
	reply := smartSuiteConfirmationReply(ctx)
	if !strings.Contains(reply, "视觉风格：真实户外庭院，自然暖光") {
		t.Fatalf("confirmation must show the chosen suite style, got %q", reply)
	}
	if !strings.Contains(reply, "生图模型：gemini-3-pro-image-preview") {
		t.Fatalf("confirmation must show the chosen drawing model, got %q", reply)
	}
}

func TestSmartSuiteListingConfirmationOmitsImageSuiteControls(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		RequestType: smartSuiteRequestTypeListing,
		ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		PlatformName: "Amazon", CountryName: "US", LanguageName: "English",
		VisualStyle: "should not be shown", DrawingModel: "should-not-be-shown",
	}
	reply := smartSuiteConfirmationReply(ctx)
	if strings.Contains(reply, "视觉风格") || strings.Contains(reply, "生图模型") {
		t.Fatalf("listing confirmation must omit image-suite controls, got %q", reply)
	}
	if !strings.Contains(reply, "Listing") || !strings.Contains(reply, "5 条卖点") {
		t.Fatalf("listing confirmation must describe listing deliverables, got %q", reply)
	}
}

func TestSmartSuiteDrawingModelDescriptions(t *testing.T) {
	tests := map[string]string{
		"gemini-3-pro-image-preview":     "适合高保真",
		"gemini-3.1-flash-image-preview": "速度快",
		"Gemini 3.1 Flash":               "速度快",
		"gpt-image-2":                    "适合设计",
		"doubao-seedream-5-0-pro-260628": "通用",
	}
	for name, expected := range tests {
		if actual := smartSuiteDrawingModelDescription(name); actual != expected {
			t.Fatalf("description for %q: expected %q, got %q", name, expected, actual)
		}
	}
}

func TestSmartSuiteConversationRequiresDrawingModel(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png", ProductName: "Garden hose", ProductFeatures: "304 stainless steel outer shell",
		Platform: "amazon", CountryCode: "US", SecondaryCount: 6, Language: "en", LanguageName: "英文",
	}
	normalizeSmartSuiteConversationContext(&ctx)
	missing := smartSuiteMissingFields(ctx)
	if !containsSmartSuiteField(missing, "drawing_model") {
		t.Fatalf("drawing model must be required before generation, missing=%v", missing)
	}
	if containsSmartSuiteField(missing, "visual_style") || smartSuiteConversationSuggestionField(ctx, "collect_requirement", false) != "drawing_model" {
		t.Fatalf("drawing-model selection must be the final required step and style must be optional, missing=%v", missing)
	}
	reply := smartSuiteCoordinatorReply(model.SmartSuiteConversation{Context: ctx}, smartSuiteCoordinatorOutput{Intent: "collect_requirement"}, false)
	if !strings.Contains(reply, "生图模型") {
		t.Fatalf("expected drawing model question, got %q", reply)
	}
}

func TestAppendUniqueSmartSuiteURLsDoesNotDuplicateStyleReferences(t *testing.T) {
	urls := appendUniqueSmartSuiteURLs(nil, "/uploads/style.png")
	urls = appendUniqueSmartSuiteURLs(urls, "/uploads/style.png")
	if len(urls) != 1 || urls[0] != "/uploads/style.png" {
		t.Fatalf("unexpected style reference URLs: %v", urls)
	}
}

func TestNormalizeSmartSuiteStyleReferenceURLs(t *testing.T) {
	empty := normalizeSmartSuiteStyleReferenceURLs(nil)
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty style references must be a non-nil JSON array, got %#v", empty)
	}

	urls := normalizeSmartSuiteStyleReferenceURLs([]string{" /uploads/style.png ", "", "/uploads/style.png", "/uploads/second.png"})
	if len(urls) != 2 || urls[0] != "/uploads/style.png" || urls[1] != "/uploads/second.png" {
		t.Fatalf("unexpected normalized style references: %#v", urls)
	}
}

func TestSmartSuiteConversationImageCommandsKeepExplicitTarget(t *testing.T) {
	conversation := &model.SmartSuiteConversation{}
	edit := smartSuiteLocalCoordinator(conversation, smartSuiteConversationTurnRequest{Text: "背景换成庭院，产品角度不变", TargetImageID: 42})
	if edit.Intent != "edit_image" || edit.TargetImageID != 42 {
		t.Fatalf("unexpected edit intent: %+v", edit)
	}
	accept := smartSuiteLocalCoordinator(conversation, smartSuiteConversationTurnRequest{Text: "采用这个版本", TargetImageID: 42, HasCandidate: true})
	if accept.Intent != "accept_candidate" || accept.TargetImageID != 42 {
		t.Fatalf("unexpected candidate intent: %+v", accept)
	}
}

func TestValidateSmartSuiteCoordinatorAcceptsOtherRequirementsWithEvidence(t *testing.T) {
	text := "Amazon 美国站，副图、桌面 A+、移动 A+ 都要，英文文案，使用 gemini-3-pro-image-preview"
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "other",
		Confidence:      0.97,
		OtherEvidence:   []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}},
		Updates: smartSuiteCoordinatorUpdates{
			Platform: stringPtr("amazon"), PlatformName: stringPtr("Amazon"),
			CountryCode: stringPtr("US"), CountryName: stringPtr("美国站"),
			Language: stringPtr("en"), LanguageName: stringPtr("英文"),
			SecondaryCount: intPtr(6), DesktopAPlusCount: intPtr(7), MobileAPlusCount: intPtr(7),
			DrawingModel: stringPtr("gemini-3-pro-image-preview"),
		},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: text}, &output); err != nil {
		t.Fatalf("valid non-semantic requirements rejected: %v", err)
	}
	ctx := model.SmartSuiteConversationContext{}
	applySmartSuiteCoordinatorUpdates(&ctx, output.Updates)
	normalizeSmartSuiteConversationContext(&ctx)
	if ctx.Platform != "amazon" || ctx.CountryCode != "US" || ctx.Language != "en" || ctx.SecondaryCount != 6 || ctx.DesktopAPlusCount != 7 || ctx.MobileAPlusCount != 7 || ctx.OutputTarget != "all" || ctx.DrawingModel != "gemini-3-pro-image-preview" {
		t.Fatalf("agent requirements were not applied: %+v", ctx)
	}
}

func TestValidateSmartSuiteCoordinatorAcceptsPlatformSelectionWhenAgentKindIsMixed(t *testing.T) {
	text := "Amazon 美国站"
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "mixed",
		Confidence:      0.96,
		OtherEvidence:   []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "amazon，美国站"}},
		Updates: smartSuiteCoordinatorUpdates{
			Platform: stringPtr("amazon"), PlatformName: stringPtr("Amazon"),
			CountryCode: stringPtr("US"), CountryName: stringPtr("美国站"),
		},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: text}, &output); err != nil {
		t.Fatalf("valid platform selection rejected: %v", err)
	}
	if output.RequirementKind != "other" {
		t.Fatalf("platform-only updates must normalize to other, got %q", output.RequirementKind)
	}
	ctx := model.SmartSuiteConversationContext{}
	applySmartSuiteCoordinatorUpdates(&ctx, output.Updates)
	if containsSmartSuiteField(smartSuiteMissingFields(ctx), "platform_site") {
		t.Fatalf("platform selection was not stored: %+v", ctx)
	}
}

func TestValidateSmartSuiteCoordinatorNormalizesNonContractPlatformKind(t *testing.T) {
	text := "Amazon 美国站"
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "platform_site",
		Confidence:      0.96,
		OtherEvidence:   []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}},
		Updates: smartSuiteCoordinatorUpdates{
			Platform: stringPtr("amazon"), CountryCode: stringPtr("US"), CountryName: stringPtr("美国站"),
		},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: text}, &output); err != nil {
		t.Fatalf("platform update with a recoverable kind rejected: %v", err)
	}
	if output.RequirementKind != "other" {
		t.Fatalf("platform update kind was not normalized: %q", output.RequirementKind)
	}
}

func TestValidateSmartSuiteCoordinatorIgnoresEchoedContextOnPlatformTurn(t *testing.T) {
	text := "Amazon 美国站"
	current := model.SmartSuiteConversationContext{
		ProductName: "浴室龙头", ProductFeatures: "黄铜主体，冷热双控", VisualStyle: "现代家居场景",
	}
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "mixed",
		Confidence:      0.96,
		OtherEvidence:   []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}},
		Updates: smartSuiteCoordinatorUpdates{
			ProductName: stringPtr(current.ProductName), ProductFeatures: stringPtr(current.ProductFeatures), VisualStyle: stringPtr(current.VisualStyle),
			Platform: stringPtr("amazon"), PlatformName: stringPtr("Amazon"), CountryCode: stringPtr("US"), CountryName: stringPtr("美国站"),
		},
	}
	if err := validateSmartSuiteCoordinatorOutput(current, smartSuiteConversationTurnRequest{Text: text}, &output); err != nil {
		t.Fatalf("platform selection with echoed context rejected: %v", err)
	}
	if output.RequirementKind != "other" || output.Updates.ProductName != nil || output.Updates.ProductFeatures != nil || output.Updates.VisualStyle != nil {
		t.Fatalf("echoed context was not removed before validation: %+v", output)
	}
	applySmartSuiteCoordinatorUpdates(&current, output.Updates)
	if current.Platform != "amazon" || current.CountryCode != "US" {
		t.Fatalf("platform selection was not applied: %+v", current)
	}
}

func TestValidateSmartSuiteCoordinatorAcceptsExclusiveOutputUpdate(t *testing.T) {
	text := "只要6张副图"
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "other",
		Confidence:      0.96,
		OtherEvidence:   []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}},
		Updates: smartSuiteCoordinatorUpdates{
			SecondaryCount: intPtr(6), DesktopAPlusCount: intPtr(0), MobileAPlusCount: intPtr(0),
		},
	}
	current := model.SmartSuiteConversationContext{SecondaryCount: 6, DesktopAPlusCount: 7, MobileAPlusCount: 7}
	if err := validateSmartSuiteCoordinatorOutput(current, smartSuiteConversationTurnRequest{Text: text}, &output); err != nil {
		t.Fatalf("valid exclusive output selection rejected: %v", err)
	}
	applySmartSuiteCoordinatorUpdates(&current, output.Updates)
	normalizeSmartSuiteConversationContext(&current)
	if current.SecondaryCount != 6 || current.DesktopAPlusCount != 0 || current.MobileAPlusCount != 0 || current.OutputTarget != "secondary" {
		t.Fatalf("exclusive output request was not applied: %+v", current)
	}
}

func TestSmartSuiteOutputCountUpdatesFromFreeText(t *testing.T) {
	tests := []struct {
		text                       string
		secondary, desktop, mobile int
	}{
		{text: "副图3张", secondary: 3},
		{text: "副图和桌面A+", secondary: 6, desktop: 7},
		{text: "3张副图和2张桌面 A+", secondary: 3, desktop: 2},
		{text: "副图、桌面A+和移动A+各3张", secondary: 3, desktop: 3, mobile: 3},
	}
	for _, test := range tests {
		updates, ok := smartSuiteOutputCountUpdatesFromText(test.text)
		if !ok {
			t.Fatalf("expected output selection to be recognized: %q", test.text)
		}
		ctx := model.SmartSuiteConversationContext{SecondaryCount: 9, DesktopAPlusCount: 8, MobileAPlusCount: 8}
		applySmartSuiteCoordinatorUpdates(&ctx, updates)
		normalizeSmartSuiteConversationContext(&ctx)
		if ctx.SecondaryCount != test.secondary || ctx.DesktopAPlusCount != test.desktop || ctx.MobileAPlusCount != test.mobile {
			t.Fatalf("unexpected counts for %q: %+v", test.text, ctx)
		}
	}
}

func TestSmartSuiteOutputCountUpdatesIgnoreImageReference(t *testing.T) {
	if updates, ok := smartSuiteOutputCountUpdatesFromText("副图第3张背景亮一点"); ok {
		t.Fatalf("image edit reference must not change requested output counts: %+v", updates)
	}
}

func TestValidateSmartSuiteCoordinatorRejectsUnsupportedOtherRequirements(t *testing.T) {
	text := "整套都要"
	tests := []smartSuiteCoordinatorOutput{
		{
			RequirementKind: "other", Confidence: 0.95,
			Updates: smartSuiteCoordinatorUpdates{SecondaryCount: intPtr(6), DesktopAPlusCount: intPtr(7), MobileAPlusCount: intPtr(7)},
		},
		{
			RequirementKind: "other", Confidence: 0.95,
			OtherEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "用户没有说过"}},
			Updates:       smartSuiteCoordinatorUpdates{SecondaryCount: intPtr(6), DesktopAPlusCount: intPtr(7), MobileAPlusCount: intPtr(7)},
		},
		{
			RequirementKind: "other", Confidence: 0.4,
			OtherEvidence: []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}},
			Updates:       smartSuiteCoordinatorUpdates{SecondaryCount: intPtr(6), DesktopAPlusCount: intPtr(7), MobileAPlusCount: intPtr(7)},
		},
	}
	for index := range tests {
		if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: text}, &tests[index]); err == nil {
			t.Fatalf("unsafe other-requirement update %d must be rejected", index)
		}
	}
}

func TestValidateSmartSuiteCoordinatorAmbiguityClearsAllRequirementUpdates(t *testing.T) {
	output := smartSuiteCoordinatorOutput{
		RequirementKind:    "ambiguous",
		Confidence:         0.4,
		NeedsClarification: true,
		OtherEvidence:      []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: "整套"}},
		Updates: smartSuiteCoordinatorUpdates{
			Platform: stringPtr("amazon"), SecondaryCount: intPtr(6), Language: stringPtr("en"), DrawingModel: stringPtr("gemini-3-pro-image-preview"),
		},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: "整套"}, &output); err != nil {
		t.Fatalf("ambiguous requirement should ask for clarification: %v", err)
	}
	if smartSuiteHasOtherRequirementUpdate(output.Updates) {
		t.Fatalf("ambiguous requirement must not write any fields: %+v", output.Updates)
	}
}

func TestSmartSuiteImageReviewStatus(t *testing.T) {
	tests := map[string]string{
		"smart_suite_reviewing":                        "reviewing",
		"smart_suite_auto_correcting":                  "correcting",
		"smart_suite_auto_corrected":                   "auto_corrected",
		"smart_suite_review_correction_failed: detail": "correction_failed",
		"smart_suite_review_unresolved: detail":        "unresolved",
		"smart_suite_review_degraded: timeout":         "degraded",
		"":                                             "",
	}
	for input, expected := range tests {
		if actual := smartSuiteImageReviewStatus(input); actual != expected {
			t.Fatalf("review status for %q: expected %q, got %q", input, expected, actual)
		}
	}
}

func TestMergeSmartSuiteCoordinatorDoesNotImplicitlyStart(t *testing.T) {
	output := mergeSmartSuiteCoordinatorOutput(
		smartSuiteCoordinatorOutput{Intent: "collect_requirement"},
		smartSuiteCoordinatorOutput{Intent: "start", Reply: "信息完整，开始生成。"},
	)
	if output.Intent != "collect_requirement" {
		t.Fatalf("expected AI start intent to be gated by user text, got %q", output.Intent)
	}
}

func TestSmartSuiteCoordinatorCannotStartWithoutExplicitUserIntent(t *testing.T) {
	base := smartSuiteCoordinatorOutput{Intent: "collect_requirement"}
	ai := smartSuiteCoordinatorOutput{Intent: "start", Updates: smartSuiteCoordinatorUpdates{ProductName: stringPtr("Hose")}}
	merged := mergeSmartSuiteCoordinatorOutput(base, ai)
	if merged.Intent != "collect_requirement" || merged.Updates.ProductName == nil || *merged.Updates.ProductName != "Hose" {
		t.Fatalf("AI extraction should be retained without allowing an implicit start: %+v", merged)
	}
}

func TestSmartSuiteExplicitStartText(t *testing.T) {
	for _, text := range []string{"开始生成", "确认生成", "开始吧", "生成吧", "可以了", "没问题", "就这样", "start"} {
		if !smartSuiteExplicitStartText(text) {
			t.Fatalf("expected explicit start text %q", text)
		}
	}
	if smartSuiteExplicitStartText("产品信息完整") {
		t.Fatal("product completeness alone must not start generation")
	}
}

func TestSmartSuiteLocalCoordinatorStartsFromNaturalConfirmation(t *testing.T) {
	for _, text := range []string{"开始吧", "生成吧", "可以了"} {
		output := smartSuiteLocalCoordinator(&model.SmartSuiteConversation{}, smartSuiteConversationTurnRequest{Text: text})
		if output.Intent != "start" {
			t.Fatalf("expected %q to start locally, got %q", text, output.Intent)
		}
	}
}

func TestSmartSuiteConversationResolvesNaturalImageReference(t *testing.T) {
	images := []model.EcomSuiteImage{
		{ID: 11, TaskID: "smart-task", ImageType: "smart_secondary", ImageIndex: 1},
		{ID: 12, TaskID: "smart-task", ImageType: "smart_secondary", ImageIndex: 3},
		{ID: 21, TaskID: "smart-task", ImageType: "smart_desktop_a_plus", ImageIndex: 2},
	}
	if id := resolveSmartSuiteImageReferenceFromImages(images, "smart_secondary", smartSuiteImageReferenceIndex("副图第3张背景亮一点")); id != 12 {
		t.Fatalf("expected third secondary image, got %d", id)
	}
	if id := resolveSmartSuiteImageReferenceFromImages(images, "smart_desktop_a_plus", smartSuiteImageReferenceIndex("桌面 A+ 第二张不要人物")); id != 21 {
		t.Fatalf("expected second desktop image, got %d", id)
	}
	if id := resolveSmartSuiteImageReferenceFromImages(images, "", smartSuiteImageReferenceIndex("第三张背景亮一点")); id != 12 {
		t.Fatalf("expected an unqualified third image to prefer the secondary suite, got %d", id)
	}
}

func TestValidateSmartSuiteCoordinatorKeepsAgentPublicReport(t *testing.T) {
	text := "Amazon US"
	output := smartSuiteCoordinatorOutput{
		RequirementKind: "other",
		Confidence:      0.96,
		OtherEvidence:   []smartSuiteCoordinatorEvidence{{Source: "user_text", Quote: text}},
		PublicSummary:   "Confirmed Amazon US as the target marketplace.",
		PublicActions:   []string{"Set platform to Amazon", "Set market to US", "Set platform to Amazon"},
		Updates: smartSuiteCoordinatorUpdates{
			Platform: stringPtr("amazon"), CountryCode: stringPtr("US"),
		},
	}
	if err := validateSmartSuiteCoordinatorOutput(model.SmartSuiteConversationContext{}, smartSuiteConversationTurnRequest{Text: text}, &output); err != nil {
		t.Fatalf("valid public report rejected: %v", err)
	}
	if output.PublicSummary == "" || len(output.PublicActions) != 2 {
		t.Fatalf("expected normalized agent public report, got summary=%q actions=%v", output.PublicSummary, output.PublicActions)
	}
	merged := mergeSmartSuiteCoordinatorOutput(smartSuiteCoordinatorOutput{Intent: "collect_requirement"}, output)
	if merged.PublicSummary != output.PublicSummary || len(merged.PublicActions) != 2 {
		t.Fatalf("public report was not retained when merging coordinator output: %+v", merged)
	}
}

func TestBuildSmartSuiteTodoItemsUsesOnlyAgentPlannedStages(t *testing.T) {
	questions := []string{"Does it fit the intended space?", "What material is visible?"}
	secondaryOne := model.EcomSuiteImagePlan{ImageType: "smart_secondary", ImageIndex: 1, Title: "Secondary template", Role: questions[0], VisualStrategy: "show installation clearance"}
	secondaryTwo := model.EcomSuiteImagePlan{ImageType: "smart_secondary", ImageIndex: 2, Title: "Secondary template", Role: questions[1], VisualStrategy: "show a material macro"}
	desktop := model.EcomSuiteImagePlan{ImageType: "smart_desktop_a_plus", ImageIndex: 1, Title: "Desktop agent plan", VisualStrategy: "wide product introduction"}
	task := model.EcomSuiteTask{
		Status: "planning",
		PlanSnapshot: &model.EcomSuitePlanSnapshot{
			ImagePlans: []model.EcomSuiteImagePlan{secondaryOne, secondaryTwo, desktop},
			CreativeDirections: []model.EcomSuiteCreativeDirection{
				{ImageType: "smart_secondary", ImageIndex: 1},
				{ImageType: "smart_secondary", ImageIndex: 2},
			},
		},
		Images: []model.EcomSuiteImage{
			{ImageType: "smart_secondary", ImageIndex: 1, Status: ecomSuitePlanningReadyStatus},
			{ImageType: "smart_secondary", ImageIndex: 2, Status: "pending"},
			{ImageType: "smart_desktop_a_plus", ImageIndex: 1, Status: "pending"},
		},
	}
	items, _, _, total := buildSmartSuiteTodoItems(task, questions)
	if total != 2 || len(items) != 2 {
		t.Fatalf("unplanned desktop stage leaked into todo list: %+v", items)
	}
	if items[0].Label != secondaryOne.Title || items[0].Source != "secondary_direction_agent" || items[0].Detail != secondaryOne.VisualStrategy {
		t.Fatalf("secondary todo must come from the secondary direction agent before visual planning: %+v", items[0])
	}

	task.PlanSnapshot.PlannedImageTypes = []string{"smart_secondary", "smart_desktop_a_plus"}
	items, _, _, total = buildSmartSuiteTodoItems(task, questions)
	if total != 3 || len(items) != 3 {
		t.Fatalf("agent-planned desktop stage missing from todo list: %+v", items)
	}
	if items[2].Label != desktop.Title || items[2].Source != "suite_planning_agent" || items[2].Detail != desktop.VisualStrategy {
		t.Fatalf("desktop todo did not use planning-agent output: %+v", items[2])
	}
}

func TestBuildSmartSuiteTodoItemsDoesNotInventPlan(t *testing.T) {
	items, action, completed, total := buildSmartSuiteTodoItems(model.EcomSuiteTask{Status: "planning"}, nil)
	if len(items) != 0 || completed != 0 || total != 0 || !strings.Contains(action, "Agent") {
		t.Fatalf("missing plan must not produce fixed todo items: items=%v action=%q completed=%d total=%d", items, action, completed, total)
	}
}

func TestBuildSmartSuiteBigTodoItemsAggregatesImageWork(t *testing.T) {
	task := model.EcomSuiteTask{
		Status:            "planning",
		SmartStage:        "planning_secondary",
		SecondaryCount:    2,
		DesktopAPlusCount: 1,
		Images: []model.EcomSuiteImage{
			{ImageType: "smart_secondary", ImageIndex: 1, Status: "success"},
			{ImageType: "smart_secondary", ImageIndex: 2, Status: ecomSuitePlanningReadyStatus},
			{ImageType: "smart_desktop_a_plus", ImageIndex: 1, Status: ecomSuitePlanningReadyStatus},
		},
	}
	items, _, completed, total := buildSmartSuiteBigTodoItems(task, []string{"Where is it used?", "How does it fit?"})
	if total != 3 || len(items) != 3 {
		t.Fatalf("expected buyer insight and two generation stages, got items=%+v", items)
	}
	if items[0].ID != "buyer-insight" || items[1].ID != "smart_secondary" || items[2].ID != "smart_desktop_a_plus" {
		t.Fatalf("todo items must be grouped by large task: %+v", items)
	}
	if items[1].Status != "active" || items[2].Status != "pending" {
		t.Fatalf("only the current stage should be active: %+v", items)
	}
	if completed != 1 {
		t.Fatalf("only buyer insight should be complete in this state, completed=%d items=%+v", completed, items)
	}
}

func TestBuildSmartSuiteBigTodoItemsIgnoresLegacyReviewStage(t *testing.T) {
	task := model.EcomSuiteTask{
		Status:         "reviewing",
		SmartStage:     "reviewing_secondary",
		SecondaryCount: 2,
		Images: []model.EcomSuiteImage{
			{ImageType: "smart_secondary", ImageIndex: 1, Status: "success"},
			{ImageType: "smart_secondary", ImageIndex: 2, Status: "success"},
		},
	}
	items, _, completed, total := buildSmartSuiteBigTodoItems(task, []string{"Will it fit?", "How is it used?"})
	if total != 2 || len(items) != 2 || completed != 2 {
		t.Fatalf("legacy review stage must collapse to completed generation work: %+v", items)
	}
	if items[1].Status != "done" || items[1].Source == "quality_review_agent" {
		t.Fatalf("legacy review stage must not expose an AI review agent: %+v", items[1])
	}
	for _, item := range items {
		if item.ID == "final-review" || item.Source == "quality_review_agent" {
			t.Fatalf("AI review todo must be removed: %+v", items)
		}
	}
}

func TestFinalizeSmartSuiteBigTodoItemsMarksFailureAndSkipsRemainingWork(t *testing.T) {
	items := []ecomSuiteTodoItem{
		{ID: "product-facts", Status: "done"},
		{ID: "buyer-insight", Status: "active"},
		{ID: "style-master", Status: "pending"},
		{ID: "suite-prompts", Status: "pending"},
	}
	items = finalizeSmartSuiteBigTodoItems(model.EcomSuiteTask{Status: "failed", ErrorMessage: "smart_suite_buyer_insight_failed"}, items)
	if items[1].Status != "failed" || items[1].Detail != "smart_suite_buyer_insight_failed" {
		t.Fatalf("active stage should become the failed stage: %+v", items)
	}
	for _, item := range items[2:] {
		if item.Status != "skipped" {
			t.Fatalf("unstarted stages should be skipped after a terminal failure: %+v", items)
		}
	}
}

func TestFinalizeSmartSuiteBigTodoItemsClosesEveryItemOnSuccess(t *testing.T) {
	items := []ecomSuiteTodoItem{
		{ID: "product-facts", Status: "done"},
		{ID: "suite-prompts", Status: "active"},
		{ID: "smart_desktop_a_plus", Status: "pending"},
	}
	items = finalizeSmartSuiteBigTodoItems(model.EcomSuiteTask{Status: "success"}, items)
	for _, item := range items {
		if item.Status != "done" {
			t.Fatalf("a successful task must not leave a todo item in progress: %+v", items)
		}
	}
}

func TestApplySmartSuiteInitialRequirementsUsesPremiumDefaultAndRequiresOutput(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{WhiteBgImageURL: "/uploads/product.png", DrawingModel: "image-model"}
	code, message := applySmartSuiteInitialRequirements(&ctx, smartSuiteInitialRequirements{
		ProductName:     "Stone Tray",
		ProductFeatures: "Matte stone surface; non-slip base",
		Platform:        "amazon",
		PlatformName:    "Amazon",
		CountryCode:     "US",
		CountryName:     "美国站",
		SecondaryCount:  6,
	})
	if code != "" || message != "" {
		t.Fatalf("valid structured requirements rejected: %s %s", code, message)
	}
	if ctx.VisualStyle != smartSuiteAutoVisualStyle || ctx.OutputTarget != "secondary" {
		t.Fatalf("initial requirements should use the premium AI default and derive output target: %+v", ctx)
	}

	code, _ = applySmartSuiteInitialRequirements(&ctx, smartSuiteInitialRequirements{ProductName: "Only product", ProductFeatures: "Known facts"})
	if code != "initial_requirements_platform_required" {
		t.Fatalf("missing platform should be rejected before output planning, got %s", code)
	}
}

func TestApplySmartSuiteInitialRequirementsRetainsProductImagesForListing(t *testing.T) {
	ctx := model.SmartSuiteConversationContext{
		WhiteBgImageURL: "/uploads/product.png",
		VisualStyle:     "old style", StyleRequirements: "old requirements",
		StyleReferenceURLs: model.StringSlice{"/uploads/style.png"}, DrawingModel: "old-model",
	}
	code, message := applySmartSuiteInitialRequirements(&ctx, smartSuiteInitialRequirements{
		RequestType: smartSuiteRequestTypeListing, ProductName: "Garden hose", ProductFeatures: "304 stainless steel",
		Platform: "amazon", PlatformName: "Amazon", CountryCode: "US", CountryName: "US",
		ListingKeywords: "garden hose; kink resistant",
	})
	if code != "" || message != "" {
		t.Fatalf("valid listing requirements rejected: %s %s", code, message)
	}
	if ctx.WhiteBgImageURL != "/uploads/product.png" || len(ctx.ProductReferenceURLs) != 0 || ctx.VisualStyle != "" || ctx.StyleRequirements != "" || len(ctx.StyleReferenceURLs) != 0 || ctx.DrawingModel != "" {
		t.Fatalf("listing requirements did not retain product image or clear image-suite controls: %+v", ctx)
	}
	if !strings.Contains(ctx.ListingKeywords, "garden hose") || !strings.Contains(ctx.ListingKeywords, "kink resistant") {
		t.Fatalf("operator listing keywords were not retained: %q", ctx.ListingKeywords)
	}
}

func TestParseSmartSuiteCopywritingResponseSupportsReasoningAliasesAndStringIndex(t *testing.T) {
	raw := `<think>{"copywriting":[]}</think>{"copies":[{"type":"desktop A+","index":"2","headline":"Show the Working Structure","supporting_copy":"Follow the visible mechanism from action to result.","labels":"No labels","confirmed_fact_ids":["F2"]}]}`
	copies, err := parseSmartSuiteCopywritingResponse(raw)
	if err != nil || len(copies) != 1 {
		t.Fatalf("expected one copy block, got copies=%+v err=%v", copies, err)
	}
	if copies[0].ImageType != "smart_desktop_a_plus" || copies[0].ImageIndex != 2 || copies[0].Title != "Show the Working Structure" || len(copies[0].EvidenceFactIDs) != 1 || copies[0].EvidenceFactIDs[0] != "F2" {
		t.Fatalf("copy aliases were not normalized: %+v", copies[0])
	}
}

func TestSmartSuiteCopywritingCandidateRejectsGenericAndMultipleLabels(t *testing.T) {
	copy := model.EcomSuiteCopywriting{ImageType: "smart_secondary", ImageIndex: 1, Title: "Made for the Job", Subtitle: "A concrete visible result in daily use", MustIncludeText: "One|Two"}
	if smartSuiteCopywritingCandidateUsable(copy, "smart_secondary") {
		t.Fatal("generic copy or multiple labels must be rejected")
	}
	copy.Title = "Keep Essentials Within Reach"
	copy.Subtitle = "The marble shelf gives small items a convenient resting spot."
	copy.MustIncludeText = "No labels"
	if !smartSuiteCopywritingCandidateUsable(copy, "smart_secondary") {
		t.Fatalf("specific copy should be accepted: %+v", copy)
	}
	copy.Title = "Gold-Tone Finish Shown Up Close"
	if smartSuiteCopywritingCandidateUsable(copy, "smart_secondary") {
		t.Fatal("image-caption language must be rejected")
	}
	copy.Title = "Keep Essentials Within Reach"
	copy.MustIncludeText = "Material proof"
	if issue := smartSuiteCopywritingCandidateIssue(copy, "smart_secondary"); issue != "unexpected_label" {
		t.Fatalf("a third visible text item must be rejected, got %q", issue)
	}
}

func TestCoerceSmartSuiteCopywritingRejectsInvalidCopyInsteadOfInventingFallback(t *testing.T) {
	plans := []model.EcomSuiteImagePlan{{ImageType: "smart_desktop_a_plus", ImageIndex: 1, AssignedFactIDs: []string{"F1"}, AssignedProductFacts: "Brushed nickel finish"}}
	copies := []model.EcomSuiteCopywriting{{ImageType: "smart_desktop_a_plus", ImageIndex: 1, Title: "Shown Up Close", Subtitle: "This detail comes from the confirmed product information."}}
	if copies, ok := coerceSmartSuiteCopywriting(model.EcomSuiteTask{}, plans, copies); ok || copies != nil {
		t.Fatalf("invalid copy must remain a failure for retry, got %+v", copies)
	}
}

func TestSmartSuiteCopywritingAllowsCompleteChinesePhrasesAndRejectsSourceNotes(t *testing.T) {
	copy := model.EcomSuiteCopywriting{ImageType: "smart_mobile_a_plus", ImageIndex: 1, Title: "冲洗更顺手", Subtitle: "抽拉喷头方便调整冲洗方向"}
	if issue := smartSuiteCopywritingCandidateIssue(copy, copy.ImageType); issue != "" {
		t.Fatalf("complete Chinese phrase rejected: %s", issue)
	}
	copy.Subtitle = "产品资料已确认。"
	if issue := smartSuiteCopywritingCandidateIssue(copy, copy.ImageType); issue != "caption_language" {
		t.Fatalf("source note accepted: %s", issue)
	}
}

func TestCoerceSmartSuiteCopywritingRejectsMissingModule(t *testing.T) {
	plans := []model.EcomSuiteImagePlan{{ImageType: "smart_desktop_a_plus", ImageIndex: 4, AssignedFactIDs: []string{"F1"}}}
	if coerced, ok := coerceSmartSuiteCopywriting(model.EcomSuiteTask{}, plans, nil); ok || coerced != nil {
		t.Fatalf("missing copy identity must remain a batch failure: ok=%v copies=%+v", ok, coerced)
	}
}

func TestBuildSmartSuiteCopywriterPromptsLockLanguageAndAssignments(t *testing.T) {
	task := model.EcomSuiteTask{ProductName: "Wall holder", ProductFeatures: "FULL_PRODUCT_DOCUMENT_MUST_STAY_UPSTREAM", ProductProof: "FULL_PRODUCT_PROOF_MUST_STAY_UPSTREAM", Platform: "Amazon", CountryName: "United States", CountryCode: "US", Language: "English"}
	plans := []model.EcomSuiteImagePlan{{ImageType: "smart_secondary", ImageIndex: 1, Role: "Where does it fit?", CTADirection: "Show the phone shelf", AssignedFactIDs: []string{"F2"}, AssignedProductFacts: "marble shelf", VisualStrategy: "phone rests visibly on shelf"}}
	system, user := buildSmartSuiteCopywriterPrompts(task, plans)
	if !strings.Contains(system, "independent ecommerce copywriter") || strings.Contains(system, "image-generation prompts") == false {
		t.Fatalf("copywriter role boundary missing: %s", system)
	}
	for _, expected := range []string{"language: English", "target market: United States (US)", "Destination platform: Amazon", "sentence by sentence", "native copywriter", "Where does it fit?", "Show the phone shelf", "marble shelf", "assigned_fact_ids", "backend injects immutable metadata", "exactly two visible text strings", "visual_evidence", "Express the approved primary_selling_point in the title", "unrelated product category"} {
		if !strings.Contains(user, expected) {
			t.Fatalf("copywriter prompt missing %q: %s", expected, user)
		}
	}
	for _, leaked := range []string{task.ProductFeatures, task.ProductProof, "evidence_fact_ids", "must_include_text"} {
		if strings.Contains(user, leaked) {
			t.Fatalf("copywriter prompt leaked upstream facts or backend-owned metadata %q: %s", leaked, user)
		}
	}
	if strings.Contains(strings.ToLower(user), "platform visual contract") {
		t.Fatalf("copywriter prompt must not include the unverified platform visual contract: %s", user)
	}
}

func TestSmartSuiteFinalImagePromptRequiresSceneVariationAndRealPhotography(t *testing.T) {
	task := model.EcomSuiteTask{WorkflowType: "smart_suite", ProductName: "Kitchen faucet", ProductCategory: "Kitchen faucet", Language: "English", OutputRatio: "1:1"}
	plan := model.EcomSuiteImagePlan{ImageType: "smart_secondary", ImageIndex: 1, Role: "Show the product in use", AssignedFactIDs: []string{"F1"}, AssignedProductFacts: "Pull-down spray head", Prompt: "A real sink scene with a clear action and visible contact points.", Style: "Editorial kitchen with layered depth", VisualStrategy: "Show the spray head being used"}
	prompt := buildSmartSuiteImageGenerationPromptForTask(task, plan, "")
	for _, expected := range []string{"Art-directed scene rule", "generic kitchen backdrop", "clear camera viewpoint", "Avoid infographic and template aesthetics"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("final generation prompt missing scene-quality rule %q: %s", expected, prompt)
		}
	}
}

func TestReconcileSmartSuiteCopywritingKeepsValidItemsAndRetriesOnlyMissing(t *testing.T) {
	plans := []model.EcomSuiteImagePlan{
		{ImageType: "smart_secondary", ImageIndex: 1, AssignedFactIDs: []string{"F1"}, AssignedProductFacts: "The holder supports two rolls."},
		{ImageType: "smart_secondary", ImageIndex: 2, AssignedFactIDs: []string{"F2"}, AssignedProductFacts: "The shelf has visible marble veining."},
	}
	copies := []model.EcomSuiteCopywriting{
		{ImageType: "smart_secondary", ImageIndex: 1, Title: "Keep Two Rolls Ready", Subtitle: "The holder supports two rolls for daily use."},
		{ImageType: "smart_secondary", ImageIndex: 2, Title: "Made for the Job", Subtitle: "A generic unsupported line."},
		{ImageType: "smart_desktop_a_plus", ImageIndex: 99, Title: "Unexpected Extra Item", Subtitle: "This item must be ignored safely."},
	}
	ordered, missing := reconcileSmartSuiteCopywriting(model.EcomSuiteTask{}, plans, copies)
	if len(ordered) != 1 || ordered[0].ImageIndex != 1 || len(missing) != 1 || missing[0].ImageIndex != 2 {
		t.Fatalf("valid copy must survive while only the invalid item is retried: ordered=%+v missing=%+v", ordered, missing)
	}
	if ordered[0].MustIncludeText != "No labels" || len(ordered[0].EvidenceFactIDs) != 1 || ordered[0].EvidenceFactIDs[0] != "F1" {
		t.Fatalf("backend-owned metadata must be injected after copy validation: %+v", ordered[0])
	}
}

func TestReconcileSmartSuiteCopywritingAcceptsOmittedMetadataAndOrdinalIdentity(t *testing.T) {
	plans := []model.EcomSuiteImagePlan{{
		ImageType: "smart_mobile_a_plus", ImageIndex: 3, AssignedFactIDs: []string{"F3"}, AssignedProductFacts: "The valve has two selectable modes.",
	}}
	copies := []model.EcomSuiteCopywriting{{Title: "Two Modes, One Control", Subtitle: "The valve offers two selectable modes for daily tasks."}}
	ordered, missing := reconcileSmartSuiteCopywriting(model.EcomSuiteTask{}, plans, copies)
	if len(missing) != 0 || len(ordered) != 1 {
		t.Fatalf("a complete one-item response should inherit its requested identity: ordered=%+v missing=%+v", ordered, missing)
	}
	if ordered[0].ImageType != plans[0].ImageType || ordered[0].ImageIndex != plans[0].ImageIndex || ordered[0].EvidenceFactIDs[0] != "F3" || ordered[0].MustIncludeText != "No labels" {
		t.Fatalf("identity and immutable metadata were not canonicalized: %+v", ordered[0])
	}
}

func TestBuildSmartSuiteCopywriterPromptsUseCurrentBatchImageIdentities(t *testing.T) {
	task := model.EcomSuiteTask{ProductName: "Wall holder", ProductFeatures: "double-roll holder", ProductProof: "marble shelf", Platform: "Amazon", CountryName: "United States", CountryCode: "US", Language: "English"}
	plans := []model.EcomSuiteImagePlan{
		{ImageType: "smart_desktop_a_plus", ImageIndex: 1, Role: "What does it hold?", AssignedFactIDs: []string{"F1"}, AssignedProductFacts: "double-roll holder"},
		{ImageType: "smart_desktop_a_plus", ImageIndex: 2, Role: "Where does it fit?", AssignedFactIDs: []string{"F2"}, AssignedProductFacts: "marble shelf"},
	}
	_, user := buildSmartSuiteCopywriterPrompts(task, plans)
	if strings.Count(user, `"image_type":"smart_desktop_a_plus"`) != len(plans)*2 || !strings.Contains(user, `"image_index":1`) || !strings.Contains(user, `"image_index":2`) {
		t.Fatalf("copywriter response skeleton must use the requested stage identities: %s", user)
	}
	if strings.Contains(user, `"image_type":"smart_secondary"`) {
		t.Fatalf("copywriter response skeleton must not hard-code smart_secondary: %s", user)
	}
}

func TestSmartSuiteImageEditSourceURLContinuesFromReadyCandidate(t *testing.T) {
	image := model.EcomSuiteImage{ImageURL: " original.png ", CandidateImageURL: " candidate.png ", CandidateStatus: "ready"}
	if got := smartSuiteImageEditSourceURL(image); got != "candidate.png" {
		t.Fatalf("ready candidate must be the next edit source, got %q", got)
	}
	image.CandidateStatus = "failed"
	if got := smartSuiteImageEditSourceURL(image); got != "original.png" {
		t.Fatalf("non-ready candidate must fall back to the current image, got %q", got)
	}
}

func TestValidateSmartSuiteCopywritingRequiresAssignedFactCitations(t *testing.T) {
	factMap := completeSmartSuiteTestFactMap()
	task := model.EcomSuiteTask{PlanSnapshot: &model.EcomSuitePlanSnapshot{ProductFactMap: factMap}}
	plan := model.EcomSuiteImagePlan{
		ImageType: "smart_secondary", ImageIndex: 1, AssignedFactIDs: []string{"F2"},
		AssignedProductFacts: factMap.ConfirmedFacts[1].Statement,
	}
	copy := model.EcomSuiteCopywriting{
		ImageType: "smart_secondary", ImageIndex: 1,
		Title: "Keep Essentials Within Reach", Subtitle: "The marble shelf gives small items a convenient resting spot.",
		MustIncludeText: "Wrong model label", EvidenceFactIDs: []string{"F1"},
	}
	if _, err := validateSmartSuiteCopywriting(task, []model.EcomSuiteImagePlan{plan}, []model.EcomSuiteCopywriting{copy}); err != nil {
		t.Fatalf("backend-owned metadata must be canonicalized before validation: %v", err)
	}
	validated, err := validateSmartSuiteCopywriting(task, []model.EcomSuiteImagePlan{plan}, []model.EcomSuiteCopywriting{copy})
	if err != nil || len(validated) != 1 || len(validated[0].EvidenceFactIDs) != 1 || validated[0].EvidenceFactIDs[0] != "F2" || validated[0].MustIncludeText != "No labels" {
		t.Fatalf("backend must replace model metadata with the assigned fact and no-label contract: validated=%+v err=%v", validated, err)
	}
}

func TestValidateSmartSuiteCopywritingCanonicalizesBatchIdentities(t *testing.T) {
	plans := []model.EcomSuiteImagePlan{
		{ImageType: "smart_desktop_a_plus", ImageIndex: 3, AssignedFactIDs: []string{"F1"}, AssignedProductFacts: "The faucet offers three selectable modes."},
		{ImageType: "smart_desktop_a_plus", ImageIndex: 4, AssignedFactIDs: []string{"F2"}, AssignedProductFacts: "The faucet has a pull-down spray head."},
	}
	copies := []model.EcomSuiteCopywriting{
		{ImageType: "smart_secondary", ImageIndex: 1, Title: "Three Modes for Daily Tasks", Subtitle: "Choose among three selectable modes for different sink tasks."},
		{ImageType: "smart_secondary", ImageIndex: 2, Title: "Reach More of the Sink", Subtitle: "The pull-down spray head extends toward the task."},
	}
	validated, err := validateSmartSuiteCopywriting(model.EcomSuiteTask{}, plans, copies)
	if err != nil || len(validated) != 2 {
		t.Fatalf("a complete batch should recover copied example identities: validated=%+v err=%v", validated, err)
	}
	for index, copy := range validated {
		if copy.ImageType != plans[index].ImageType || copy.ImageIndex != plans[index].ImageIndex {
			t.Fatalf("copy identity was not rebound to the requested module: index=%d copy=%+v", index, copy)
		}
	}
}

func TestSmartSuiteMultiStateEvidencePreservesApprovedComposition(t *testing.T) {
	for _, panels := range []int{0, 2, 3} {
		plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Flow control")
		plan.AssignedFactIDs = []string{"F1"}
		plan.AssignedProductFacts = "The faucet offers 3 selectable modes: stream, spray and pause."
		plan.LayoutSpec.PanelCount = panels
		plan.LayoutSpec.ComparisonMode = "none"
		if panels > 0 {
			plan.LayoutSpec.ComparisonMode = "side_by_side"
		}
		plan.LayoutInstruction = "One large action view with a smaller detail study"
		got := applySmartSuiteMultiStateEvidenceLayout(plan)
		if got.LayoutSpec.PanelCount != plan.LayoutSpec.PanelCount || got.LayoutInstruction != plan.LayoutInstruction {
			t.Fatal("state count overwrote approved composition")
		}
		contract := decodeSmartSuiteGenerationContract(t, buildSmartSuiteImageGenerationPrompt(got, ""))
		if contract.ImagePlan.LayoutSpec.PanelCount != panels || !strings.Contains(contract.ImagePlan.VisualRequirements, "all 3 confirmed") {
			t.Fatalf("lost evidence or composition: %+v", contract.ImagePlan)
		}
	}
}

func TestValidateSmartSuiteCopywritingRejectsMultipleFactsForOneImage(t *testing.T) {
	plan := model.EcomSuiteImagePlan{
		ImageType: "smart_secondary", ImageIndex: 1, AssignedFactIDs: []string{"F1", "F2"},
		AssignedProductFacts: "The holder supports two rolls; the shelf has visible marble veining.",
	}
	copy := model.EcomSuiteCopywriting{
		ImageType: "smart_secondary", ImageIndex: 1,
		Title: "Keep Essentials Within Reach", Subtitle: "Two installed rolls remain visibly ready for use.",
		MustIncludeText: "No labels", EvidenceFactIDs: []string{"F1", "F2"},
	}

	if _, err := validateSmartSuiteCopywriting(model.EcomSuiteTask{}, []model.EcomSuiteImagePlan{plan}, []model.EcomSuiteCopywriting{copy}); err == nil || !strings.Contains(err.Error(), "copywriter_fact_count") {
		t.Fatalf("one image must reject multiple copy facts, got %v", err)
	}
}

func TestValidateSmartSuiteCopywritingRejectsUnsupportedNumbersMaterialsAndPerformance(t *testing.T) {
	task := model.EcomSuiteTask{}
	plan := model.EcomSuiteImagePlan{
		ImageType: "smart_secondary", ImageIndex: 1, AssignedFactIDs: []string{"F1"},
		AssignedProductFacts: "The faucet has two selectable spray modes.",
	}
	cases := []model.EcomSuiteCopywriting{
		{ImageType: "smart_secondary", ImageIndex: 1, Title: "500,000 Tested Valve Cycles", Subtitle: "Designed for dependable operation through daily kitchen routines.", MustIncludeText: "No labels", EvidenceFactIDs: []string{"F1"}},
		{ImageType: "smart_secondary", ImageIndex: 1, Title: "Diamond-Coated Ceramic Core", Subtitle: "The internal valve helps support dependable daily control.", MustIncludeText: "No labels", EvidenceFactIDs: []string{"F1"}},
		{ImageType: "smart_secondary", ImageIndex: 1, Title: "Two Modes for Daily Tasks", Subtitle: "A powerful spray tackles stubborn dish residue with ease.", MustIncludeText: "No labels", EvidenceFactIDs: []string{"F1"}},
	}
	for _, copy := range cases {
		if _, err := validateSmartSuiteCopywriting(task, []model.EcomSuiteImagePlan{plan}, []model.EcomSuiteCopywriting{copy}); err == nil || !strings.Contains(err.Error(), "copywriter_unsupported_claim") {
			t.Fatalf("unsupported copy must fail: copy=%+v err=%v", copy, err)
		}
	}
}

func TestValidateSmartSuiteCopywritingAllowsExactApprovedDimension(t *testing.T) {
	task := model.EcomSuiteTask{}
	plan := model.EcomSuiteImagePlan{ImageType: "smart_secondary", ImageIndex: 1, AssignedFactIDs: []string{"F1"}, AssignedProductFacts: "The faucet is 15.27 inches high."}
	copy := model.EcomSuiteCopywriting{ImageType: "smart_secondary", ImageIndex: 1, Title: "Plan Cabinet Clearance First", Subtitle: "The 15.27-inch height helps confirm available overhead space.", MustIncludeText: "No labels", EvidenceFactIDs: []string{"F1"}}
	if _, err := validateSmartSuiteCopywriting(task, []model.EcomSuiteImagePlan{plan}, []model.EcomSuiteCopywriting{copy}); err != nil {
		t.Fatalf("an exact approved dimension should pass: %v", err)
	}
}

func TestSmartSuiteVisualPlannerDoesNotInjectPlatformVisualContract(t *testing.T) {
	task := model.EcomSuiteTask{Platform: "home_depot", CountryName: "美国站", CountryCode: "US", Language: "en", VisualStyle: "real bathroom scene"}
	plan := completeSmartSuiteTestPlan("smart_secondary", 1, "Show the shelf")
	system, user := buildSmartSuitePlanRefinementPrompts(task, []model.EcomSuiteImagePlan{plan})
	if strings.Contains(strings.ToLower(system+user), "platform visual contract") || strings.Contains(strings.ToLower(user), "platform-specific layout") {
		t.Fatalf("visual planner must not receive the unverified platform visual contract: system=%s user=%s", system, user)
	}
}

func TestApplySmartSuiteCopywritingOnlyChangesCopyFields(t *testing.T) {
	base := model.EcomSuiteImagePlan{ImageType: "smart_secondary", ImageIndex: 1, Role: "Buyer question", CTADirection: "One selling point", AssignedProductFacts: "marble shelf", Style: "real bathroom scene", VisualStrategy: "phone rests visibly", Prompt: "Create a real scene with the product unchanged."}
	updated := applySmartSuiteCopywriting([]model.EcomSuiteImagePlan{base}, []model.EcomSuiteCopywriting{{ImageType: base.ImageType, ImageIndex: 1, Title: "Show the Marble Shelf", Subtitle: "See the phone rest on the real shelf.", MustIncludeText: "Material proof"}})[0]
	if updated.Title != "Show the Marble Shelf" || updated.Subtitle != "See the phone rest on the real shelf." || updated.MustIncludeText != "No labels" {
		t.Fatalf("copy fields were not applied: %+v", updated)
	}
	if updated.Role != base.Role || updated.CTADirection != base.CTADirection || updated.AssignedProductFacts != base.AssignedProductFacts || updated.Style != base.Style || updated.VisualStrategy != base.VisualStrategy || updated.Prompt != base.Prompt {
		t.Fatalf("copy merge must not change director or visual assignments: %+v", updated)
	}
}

func TestNormalizeSmartSuiteProductReferenceURLsKeepsSupplementaryViewsOnly(t *testing.T) {
	got := normalizeSmartSuiteProductReferenceURLs(" /uploads/main.png ", []string{"/uploads/main.png", " /uploads/detail-a.png ", "/uploads/detail-a.png", "", "/uploads/detail-b.png"})
	want := model.StringSlice{"/uploads/detail-a.png", "/uploads/detail-b.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("supplementary product refs=%v, want %v", got, want)
	}
}

func TestSmartSuiteGenerationContractSeparatesProductViewsFromStyleReferences(t *testing.T) {
	task := model.EcomSuiteTask{
		ProductReferenceURLs: model.StringSlice{"/uploads/detail-a.png", "/uploads/detail-b.png"},
	}
	rule := smartSuiteReferenceUsageRule(task)
	if !strings.Contains(rule, "Reference images 2-3 are supplementary views") || !strings.Contains(rule, "Reference images 4 and later are style references") {
		t.Fatalf("reference roles were not separated: %s", rule)
	}
}
