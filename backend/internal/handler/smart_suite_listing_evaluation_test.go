package handler

import (
	"errors"
	"strings"
	"testing"

	"advertising-image/backend/internal/model"
)

func TestNormalizeListingEvaluationUsesServerWeights(t *testing.T) {
	value := listingEvaluation{Dimensions: []listingEvaluationDimension{
		{Key: "platform_compliance", Weight: 100, Score: 80},
		{Key: "product_accuracy", Weight: 1, Score: 90},
		{Key: "completeness", Score: 70},
		{Key: "keyword_strategy", Score: 60},
		{Key: "conversion_copy", Score: 50},
		{Key: "risk_control", Score: 100},
	}}
	got := normalizeListingEvaluation(value)
	if len(got.Dimensions) != 8 {
		t.Fatalf("expected eight dimensions, got %d", len(got.Dimensions))
	}
	if got.Dimensions[0].Weight != 15 || got.Dimensions[0].WeightedScore != 12 {
		t.Fatalf("server weight was not enforced: %+v", got.Dimensions[0])
	}
	if got.Dimensions[2].Key != "completeness" || got.Dimensions[2].Weight != 10 {
		t.Fatalf("completeness weight was not updated: %+v", got.Dimensions[2])
	}
	if got.Dimensions[5].Key != "buyer_decision_support" || got.Dimensions[5].Weight != 15 {
		t.Fatalf("buyer decision dimension was not added: %+v", got.Dimensions[5])
	}
	if got.Dimensions[7].Key != "localization" || got.Dimensions[7].Weight != 10 {
		t.Fatalf("localization dimension was not retained: %+v", got.Dimensions[7])
	}
}

func TestNormalizeHomeDepotListingEvaluationUsesRawPointRubric(t *testing.T) {
	value := listingEvaluation{
		OptimizedListing: model.EcomSuiteListing{Platform: "home_depot"},
		Dimensions: []listingEvaluationDimension{
			{Key: "title", Weight: 100, Score: 1, Deductions: []listingEvaluationDeduction{{Issue: "缺失核心搜索词", Points: 2, OptimizationPlan: "在标题中补入已确认的核心品类词。"}}},
			{Key: "salient_bullets", Score: 1, Deductions: []listingEvaluationDeduction{{Issue: "亮点内容重复", Points: 6, OptimizationPlan: "改写为另一个已确认特性。"}}},
			{Key: "about_this_product", Score: 24},
			{Key: "bullet_points", Score: 1, Deductions: []listingEvaluationDeduction{{Issue: "一条为空话", Points: 3, OptimizationPlan: "替换为具体产品功能。"}, {Issue: "一条内容重复", Points: 2, OptimizationPlan: "替换为已确认的安装信息。"}}},
		},
	}
	got := normalizeListingEvaluation(value)
	if len(got.Dimensions) != 4 {
		t.Fatalf("expected four Home Depot dimensions, got %d", len(got.Dimensions))
	}
	if got.Dimensions[0].Weight != 22 || got.Dimensions[0].WeightedScore != 20 {
		t.Fatalf("Home Depot raw points were not enforced: %+v", got.Dimensions[0])
	}
	if got.Dimensions[3].Weight != 30 || got.Dimensions[3].WeightedScore != 25 {
		t.Fatalf("Home Depot bullet score was not enforced: %+v", got.Dimensions[3])
	}
}

func TestNormalizeLowesListingEvaluationPreservesHalfPointDeductions(t *testing.T) {
	value := listingEvaluation{
		OptimizedListing: model.EcomSuiteListing{Platform: "lowes"},
		Dimensions: []listingEvaluationDimension{
			{Key: "title"},
			{Key: "description", Deductions: []listingEvaluationDeduction{{Issue: "场景不足", Points: 1.5, OptimizationPlan: "补充有事实依据的使用场景。"}}},
			{Key: "bullets", Deductions: []listingEvaluationDeduction{{Issue: "只有材质名称", Points: 1.5, OptimizationPlan: "补充材质对应的消费者利益。"}}},
		},
	}
	got := normalizeListingEvaluation(value)
	if got.Dimensions[1].WeightedScore != 18.5 || got.Dimensions[2].WeightedScore != 58.5 {
		t.Fatalf("half-point deductions were not preserved: %+v", got.Dimensions)
	}
}

func TestNormalizeWayfairListingEvaluationUsesTwoFieldRubric(t *testing.T) {
	value := listingEvaluation{
		OptimizedListing: model.EcomSuiteListing{Platform: "wayfair"},
		Dimensions: []listingEvaluationDimension{
			{Key: "title", Weight: 100, Score: 1, Deductions: []listingEvaluationDeduction{{Issue: "标题把 Finish 放在核心功能之前", Points: 3, OptimizationPlan: "将全部核心功能集中放在颜色与 Finish 之前。"}}},
			{Key: "about_this_product", Score: 1, Deductions: []listingEvaluationDeduction{{Issue: "文案使用 fit information 内部话术", Points: 2, OptimizationPlan: "直接说明尺寸对应的放置或使用结果。"}, {Issue: "缺少具体使用结果", Points: 1.5, OptimizationPlan: "把确认过的功能连接到实际使用动作。"}}},
			{Key: "features", Score: 1, Deductions: []listingEvaluationDeduction{{Issue: "多条 Feature 只有裸规格", Points: 5, OptimizationPlan: "将具体特征连接到使用场景或消费者收益。"}, {Issue: "使用 switch tasks 生硬表达", Points: 0.5, OptimizationPlan: "改写为具体的操作动作和结果。"}}},
		},
	}
	got := normalizeListingEvaluation(value)
	if len(got.Dimensions) != 2 {
		t.Fatalf("expected two Wayfair dimensions, got %d", len(got.Dimensions))
	}
	wantKeys := []string{"about_this_product", "features"}
	wantWeights := []int{40, 60}
	wantScores := []float64{36.5, 54.5}
	for index, key := range wantKeys {
		if got.Dimensions[index].Key != key || got.Dimensions[index].Weight != wantWeights[index] {
			t.Fatalf("Wayfair rubric was not enforced at %d: %+v", index, got.Dimensions[index])
		}
		if got.Dimensions[index].WeightedScore != wantScores[index] {
			t.Fatalf("Wayfair %s score = %v, want %v", key, got.Dimensions[index].WeightedScore, wantScores[index])
		}
	}
}

func TestNormalizeWayfairListingEvaluationFillsMissingDimension(t *testing.T) {
	value := listingEvaluation{
		OptimizedListing: model.EcomSuiteListing{Platform: "wayfair"},
		Dimensions:       []listingEvaluationDimension{{Key: "title", Score: 20}},
	}
	got := normalizeListingEvaluation(value)
	if len(got.Dimensions) != 2 {
		t.Fatalf("expected two Wayfair dimensions, got %d", len(got.Dimensions))
	}
	if got.Dimensions[0].Key != "about_this_product" || got.Dimensions[0].Weight != 40 || got.Dimensions[0].WeightedScore != 0 {
		t.Fatalf("missing about_this_product was not filled: %+v", got.Dimensions[1])
	}
	if got.Dimensions[1].Key != "features" || got.Dimensions[1].Weight != 60 || len(got.Dimensions[1].Deductions) == 0 {
		t.Fatalf("missing features was not filled: %+v", got.Dimensions[1])
	}
}

func TestValidateListingEvaluationRequiresChineseAnalysisAndOptimizedListing(t *testing.T) {
	dimensions := make([]listingEvaluationDimension, 0, 4)
	for _, key := range []string{"title", "salient_bullets", "about_this_product", "bullet_points"} {
		dimensions = append(dimensions, listingEvaluationDimension{Key: key, Comment: "该维度已完成评估。"})
	}
	dimensions[3].Comment = "没有清楚回答安装与适配问题。"
	dimensions[3].Deductions = []listingEvaluationDeduction{{Issue: "适配信息不突出", OptimizationPlan: "在完整卖点中前置现有文案里的安装兼容信息。"}}
	value := listingEvaluation{
		Summary:         "现有文案需要优化购买信息层级。",
		Dimensions:      dimensions,
		Recommendations: []string{"先调整首屏信息，再补强完整卖点。"},
		OptimizedListing: model.EcomSuiteListing{
			Title:            "Kitchen Faucet Single Handle Pull-Down in Matte Black",
			SalientBullets:   []string{"Three spray modes", "360-degree swivel spout", "Stainless steel construction"},
			BulletPoints:     []string{"Three spray modes support daily tasks", "The swivel spout expands sink coverage", "The single handle controls flow", "The deck plate supports installation", "Matte black finish complements modern kitchens", "Pull-down reach supports sink cleanup"},
			AboutThisProduct: "Designed for everyday kitchen tasks. The pull-down sprayer supports flexible rinsing. Its matte black finish suits modern kitchens.",
		},
	}
	if err := validateListingEvaluationResult(&value, "home_depot"); err != nil {
		t.Fatal(err)
	}
	if value.OptimizedListing.Platform != "home_depot" {
		t.Fatalf("platform was not assigned: %+v", value.OptimizedListing)
	}
	value.Dimensions[3].Deductions[0].OptimizationPlan = "Rewrite the bullet."
	if err := validateListingEvaluationResult(&value, "home_depot"); err == nil || !strings.Contains(err.Error(), "not_chinese") {
		t.Fatalf("expected non-Chinese plan rejection, got %v", err)
	}
}

func TestParseListingEvaluationResponse(t *testing.T) {
	raw := `{"evaluation":{"summary":"Needs work","dimensions":[{"key":"platform_compliance","weight":25,"score":72,"comment":"Title exceeds rule","deductions":[{"issue":"Title too long","points":8,"severity":"medium","suggestion":"Shorten title","optimization_plan":"Shorten the title to the platform limit while retaining the product type and supported material fact."}]},{"key":"localization","score":60,"comment":"Uses non-local terminology","deductions":[{"issue":"Unit wording is not localized","points":4,"severity":"low","optimization_plan":"Convert the dimension wording to the target market's customary unit format using the supplied specification."}]}],"recommendations":["Shorten title"],"keyword_candidates":[{"term":"deck mount faucet","tags":["feature"]}]}}`
	value, err := parseListingEvaluationResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if value.Summary != "Needs work" || len(value.Dimensions) != 2 || len(value.Dimensions[0].Deductions) != 1 {
		t.Fatalf("unexpected evaluation: %+v", value)
	}
	if len(value.KeywordCandidates) != 1 || value.KeywordCandidates[0].Term != "deck mount faucet" {
		t.Fatalf("keyword candidates were not parsed: %+v", value.KeywordCandidates)
	}
	if value.Dimensions[0].Deductions[0].OptimizationPlan == "" {
		t.Fatalf("optimization plan was not parsed: %+v", value.Dimensions[0].Deductions[0])
	}
}

func TestNormalizeListingEvaluationBackfillsOptimizationPlan(t *testing.T) {
	value := listingEvaluation{
		Dimensions: []listingEvaluationDimension{{
			Key: "platform_compliance", Score: 80,
			Deductions: []listingEvaluationDeduction{{Issue: "Title issue", Suggestion: "Rewrite the title"}},
		}},
		Deductions: []listingEvaluationDeduction{{Issue: "Overview issue", Suggestion: "Rewrite the overview"}},
	}
	got := normalizeListingEvaluation(value)
	if got.Dimensions[0].Deductions[0].OptimizationPlan != "Rewrite the title" {
		t.Fatalf("dimension optimization plan was not backfilled: %+v", got.Dimensions[0].Deductions[0])
	}
	if got.Deductions[0].OptimizationPlan != "Rewrite the overview" {
		t.Fatalf("top-level optimization plan was not backfilled: %+v", got.Deductions[0])
	}
}

func TestHomeDepotListingFetchFailureMessages(t *testing.T) {
	cases := []struct {
		reason       string
		status       int
		messageMatch string
	}{
		{"firecrawl_api_key_not_configured", 503, "系统设置填写 API Key"},
		{"firecrawl_http_401", 502, "API Key 无效"},
		{"firecrawl_http_402", 502, "额度不足"},
		{"firecrawl_http_429", 429, "过于频繁"},
		{"firecrawl_http_404", 404, "未找到"},
		{"home_depot_listing_content_missing", 422, "About This Product"},
		{"home_depot_listing_product_details_missing", 422, "About This Product"},
		{"home_depot_listing_specifications_missing", 422, "About This Product"},
	}
	for _, testCase := range cases {
		status, message := homeDepotListingFetchFailure(errors.New(testCase.reason))
		if status != testCase.status || !strings.Contains(message, testCase.messageMatch) {
			t.Fatalf("reason %q -> status=%d message=%q", testCase.reason, status, message)
		}
	}
}

func TestHomeDepotListingFetchFailureIncludesProviderDetail(t *testing.T) {
	_, message := homeDepotListingFetchFailure(errors.New(`firecrawl_http_401: {"error":"Token is invalid"}`))
	if !strings.Contains(message, "Token is invalid") {
		t.Fatalf("provider detail was hidden: %q", message)
	}
}
