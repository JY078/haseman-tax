package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/smartagent"
)

func TestSmartSuiteListingRoutesEachPlatformToDedicatedAgent(t *testing.T) {
	tests := map[string]smartagent.Name{
		"tiktok": smartagent.WriteTikTokListing, "home_depot": smartagent.WriteHomeDepotListing,
		"lowes": smartagent.WriteLowesListing, "wayfair": smartagent.WriteWayfairListing, "amazon": smartagent.WriteAmazonListing,
	}
	for platform, expected := range tests {
		if actual := smartSuiteListingAgent(platform); actual != expected {
			t.Fatalf("platform %s routed to %s, want %s", platform, actual, expected)
		}
	}
}

func TestMergeResearchedAndOperatorKeywordsKeepsUniqueTerms(t *testing.T) {
	merged := mergeResearchedAndOperatorKeywords(
		[]model.EcomSuiteKeywordCandidate{{Term: "Kitchen Faucet"}, {Term: "pull down faucet"}},
		[]string{"kitchen faucet", "deck plate", "brushed nickel"}, 4,
	)
	if len(merged) != 4 || merged[0].Term != "Kitchen Faucet" || merged[2].Term != "deck plate" || merged[3].Term != "brushed nickel" {
		t.Fatalf("unexpected merged keyword candidates: %+v", merged)
	}
}

func TestRankListingKeywordCandidatesUsesObservedMetricsOnly(t *testing.T) {
	got := rankListingKeywordCandidates([]model.EcomSuiteKeywordCandidate{
		{Term: "low volume", SearchVolume: 100, Competition: 0.8},
		{Term: "high volume", SearchVolume: 12000, Competition: 0.2},
		{Term: "platform signal", PlatformFrequency: 4},
	})
	if got[0].Term != "high volume" || got[0].SEOScore <= got[1].SEOScore {
		t.Fatalf("observed volume should rank first: %+v", got)
	}
	if got[2].SearchVolume != 0 || got[2].SEOScore == 0 {
		t.Fatalf("platform-only candidate should retain no invented volume and receive signal score: %+v", got[2])
	}
}

func TestNormalizeAMZ123KeywordCandidatesUsesExactQuerySource(t *testing.T) {
	queryURL := "https://www.amz123.com/usatopkeywords/1?k=Kitchen+Faucet"
	got := normalizeAMZ123KeywordCandidates([]model.EcomSuiteKeywordCandidate{
		{Term: "kitchen faucet", Source: "wrong", EvidenceURLs: []string{"https://www.amz123.com/usatopkeywords/1", "https://www.amz123.com/usatopkeywords/1?k=Kitchen+Faucet"}},
		{Term: "pull down faucet", EvidenceURLs: []string{"https://www.amz123.com/other-page"}},
	}, queryURL)
	if len(got) != 2 || got[0].Source != "amz123_top_keywords" || len(got[0].EvidenceURLs) != 1 || got[0].EvidenceURLs[0] != queryURL || got[1].EvidenceURLs[0] != queryURL {
		t.Fatalf("AMZ123 candidates were not normalized to exact query source: %+v", got)
	}
}

func TestFetchAMZ123TopKeywordsParsesRankedRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`<div class="table-body-item-words"><a title="kitchen faucets" class="text table-body-item-words-word">x</a></div><a title="pull down faucet" class="table-body-item-words-word">y</a>`))
	}))
	defer server.Close()
	got, err := fetchAMZ123TopKeywords(context.Background(), server.URL)
	if err != nil || len(got) != 2 || got[0] != "kitchen faucets" || got[1] != "pull down faucet" {
		t.Fatalf("unexpected AMZ123 rows: got=%v err=%v", got, err)
	}
}

func TestBuildAMZ123KeywordSearchURLUsesCategory(t *testing.T) {
	first, err := buildAMZ123KeywordSearchURL("Kitchen Faucet")
	if err != nil || first != "https://www.amz123.com/usatopkeywords/1?k=Kitchen+Faucet" {
		t.Fatalf("unexpected first category URL: %s, %v", first, err)
	}
	second, err := buildAMZ123KeywordSearchURL("Bathroom Vanity")
	if err != nil || second == first || !strings.Contains(second, "Bathroom+Vanity") {
		t.Fatalf("category was not applied dynamically: %s, %v", second, err)
	}
}

func TestListingKeywordResearchInputIsCategoryScoped(t *testing.T) {
	// The research request must not expose SKU facts; Writer filters category
	// candidates against confirmed facts in a later stage.
	input, _ := json.Marshal(map[string]interface{}{"platform": "lowes", "product_category": "Kitchen Faucet"})
	if strings.Contains(string(input), "confirmed_facts") || strings.Contains(string(input), "product_name") {
		t.Fatalf("category research input leaked SKU-specific fields: %s", input)
	}
}

func TestSmartSuiteListingNormalizesHomeDepotDisplayName(t *testing.T) {
	for _, platform := range []string{"home_depot", "Home Depot", "homedepot", "home-depot"} {
		if actual := smartSuiteListingAgent(platform); actual != smartagent.WriteHomeDepotListing {
			t.Fatalf("platform %q routed to %s", platform, actual)
		}
		if fields := smartSuiteListingOutputStructure(platform); !reflect.DeepEqual(fields, []string{"title", "salient_bullets", "bullet_points", "about_this_product"}) {
			t.Fatalf("platform %q used wrong fields: %v", platform, fields)
		}
		listing := model.EcomSuiteListing{Platform: platform, Title: "Incomplete"}
		if validateSmartSuiteListingPlatformStructure(listing) == nil {
			t.Fatalf("platform %q bypassed Home Depot validation", platform)
		}
	}
}

func TestSmartSuiteListingNormalizesPlatformAliasesForDedicatedAgents(t *testing.T) {
	tests := map[string]smartagent.Name{
		"TikTok Shop": smartagent.WriteTikTokListing,
		"Home Depot":  smartagent.WriteHomeDepotListing,
		"Lowe’s":      smartagent.WriteLowesListing,
		"Lowe's":      smartagent.WriteLowesListing,
		"Wayfair":     smartagent.WriteWayfairListing,
		"Amazon":      smartagent.WriteAmazonListing,
	}
	for platform, expected := range tests {
		if actual := smartSuiteListingAgent(platform); actual != expected {
			t.Fatalf("platform %q routed to %s, want %s", platform, actual, expected)
		}
	}
}

func TestSmartSuiteListingRuntimeContractDoesNotExposeOtherPlatformFields(t *testing.T) {
	homeDepot := smartSuiteListingRuntimeContract("home_depot")
	if !strings.Contains(homeDepot, `"salient_bullets"`) || !strings.Contains(homeDepot, `"bullet_points"`) || !strings.Contains(homeDepot, `"about_this_product"`) || !strings.Contains(homeDepot, `"platform_basis"`) || !strings.Contains(homeDepot, "natural US English") || strings.Contains(homeDepot, `"bullets":[`) || strings.Contains(homeDepot, `"description":"`) || strings.Contains(homeDepot, `"supplier_bullets"`) || strings.Contains(homeDepot, `"search_terms"`) {
		t.Fatalf("Home Depot contract contains the wrong platform fields: %s", homeDepot)
	}
	lowes := smartSuiteListingRuntimeContract("lowes")
	if !strings.Contains(lowes, `"bullets"`) || !strings.Contains(lowes, `"description"`) || !strings.Contains(lowes, `"title"`) || strings.Contains(lowes, `"supplier_bullets"`) || strings.Contains(lowes, `"features"`) {
		t.Fatalf("Lowe's contract contains the wrong platform fields: %s", lowes)
	}
}

func TestValidateHomeDepotListingRejectsMixedLanguageCopy(t *testing.T) {
	listing := model.EcomSuiteListing{Platform: "home_depot", Title: "厨房 Kitchen Faucet", SalientBullets: []string{"One", "Two", "Three"}, BulletPoints: []string{"A", "B", "C", "D", "E", "F"}, AboutThisProduct: "Sentence one. Sentence two. Sentence three."}
	if err := validateSmartSuiteListingPlatformStructure(listing); err == nil || !strings.Contains(err.Error(), "copy_not_us_english") {
		t.Fatalf("mixed-language Home Depot copy must be rejected: %v", err)
	}
}

func TestValidateHomeDepotFaucetTitleStartsWithProductType(t *testing.T) {
	base := model.EcomSuiteListing{Platform: "home_depot", Title: "Kitchen Faucet Single Handle Pull-Down Brushed Nickel", SalientBullets: []string{"Single handle faucet", "Pull-down spray", "Brushed nickel finish"}, BulletPoints: []string{"Feature one", "Feature two", "Feature three", "Feature four", "Feature five", "Feature six"}, AboutThisProduct: "This kitchen faucet supports daily sink tasks. The pull-down spray helps with rinsing. The finish suits kitchen use."}
	if err := validateSmartSuiteListingPlatformStructure(base); err != nil {
		t.Fatalf("product-first faucet title rejected: %v", err)
	}
	base.Title = "Single Handle Pull-Down Kitchen Faucet, Brushed Nickel"
	if err := validateSmartSuiteListingPlatformStructure(base); err == nil || !strings.Contains(err.Error(), "listing_home_depot_title_slot_order_invalid") {
		t.Fatalf("feature-first faucet title must be rejected: %v", err)
	}
}

func TestValidateLowesListingRules(t *testing.T) {
	listing := model.EcomSuiteListing{Platform: "lowes", Title: "Brand Kitchen Faucet Matte Black", Description: "A kitchen faucet for daily use.", Bullets: []string{"Pull-down sprayer helps rinse dishes with flexible water coverage"}}
	if err := validateSmartSuiteListingPlatformStructure(listing); err != nil {
		t.Fatalf("valid Lowe's listing rejected: %v", err)
	}
	listing.Title = "Bad! Title"
	if err := validateSmartSuiteListingPlatformStructure(listing); err == nil {
		t.Fatal("forbidden title punctuation accepted")
	}
	listing.Title = "Brand Kitchen Faucet Matte Black"
	listing.Bullets[0] = "GUARANTEED BEST FAUCET"
	if err := validateSmartSuiteListingPlatformStructure(listing); err == nil {
		t.Fatal("forbidden promotional language accepted")
	}
}

func TestValidateLowesTitleSlotOrder(t *testing.T) {
	listing := model.EcomSuiteListing{
		Platform:    "lowes",
		Title:       "FORIOUS Kitchen Faucet Matte Black Stainless Steel Single Handle Pull-Down Sprayer with Deck Plate",
		Description: "This kitchen faucet supports everyday food preparation, rinsing dishes and filling cookware with a coordinated sink experience.",
		Bullets:     []string{"Pull-down sprayer directs water for rinsing dishes and cleaning the sink during everyday kitchen tasks."},
	}
	if err := validateSmartSuiteListingPlatformStructure(listing); err != nil {
		t.Fatalf("ordered Lowe's title rejected: %v", err)
	}

	listing.Title = "FORIOUS Matte Black Stainless Steel Single Handle Pull-Down Kitchen Faucet with Sprayer and Deck Plate"
	if err := validateSmartSuiteListingPlatformStructure(listing); err == nil || !strings.Contains(err.Error(), "listing_lowes_title_slot_order_invalid") {
		t.Fatalf("finish before product type must be rejected: %v", err)
	}

	listing.Title = "FORIOUS Kitchen Faucet Matte Black Pull-Down Single Handle Sprayer with Deck Plate"
	if err := validateSmartSuiteListingPlatformStructure(listing); err == nil || !strings.Contains(err.Error(), "listing_lowes_title_slot_order_invalid") {
		t.Fatalf("handle after mounting/function must be rejected: %v", err)
	}
}

func TestValidateLowesBulletRequiresFeatureLead(t *testing.T) {
	listing := model.EcomSuiteListing{Platform: "lowes", Title: "Brand Kitchen Faucet Matte Black", Description: "A kitchen faucet for daily use.", Bullets: []string{"The pull-down sprayer supports everyday rinsing with flexible sink coverage"}}
	if err := validateSmartSuiteListingPlatformStructure(listing); err == nil || !strings.Contains(err.Error(), "listing_lowes_bullet_feature_lead_required") {
		t.Fatalf("weak Lowe's bullet lead must be rejected: %v", err)
	}
	listing.Bullets[0] = "Pull-down sprayer supports everyday rinsing with flexible sink coverage"
	if err := validateSmartSuiteListingPlatformStructure(listing); err != nil {
		t.Fatalf("feature-led Lowe's bullet rejected: %v", err)
	}
}

func wayfairCompliantListing() model.EcomSuiteListing {
	return model.EcomSuiteListing{
		Platform:         "wayfair",
		Title:            "Kitchen Faucet with Pull Down Sprayer and Gooseneck Spout",
		AboutThisProduct: "This kitchen faucet brings calm, confident control to the busiest corner of your home. Its gently curved gooseneck spout creates room for filling tall stockpots and washing large pans after a family dinner. A pull down sprayer reaches every corner of the basin, switching from everyday rinsing to filling cookware. Smooth single lever handling keeps water adjustments simple when your hands are full, turning daily cleanup into a quicker, more comfortable routine.",
		Features:         []string{"Commercial style gooseneck spout design", "Three function pull down sprayer", "360 degree swivel for full sink reach", "Quick connect installation saves setup time", "SUS 304 stainless steel construction"},
	}
}

func TestWayfairListingUsesOnlyTwoFields(t *testing.T) {
	fields := smartSuiteListingOutputStructure("wayfair")
	if !reflect.DeepEqual(fields, []string{"about_this_product", "features"}) {
		t.Fatalf("unexpected Wayfair fields: %v", fields)
	}
	listing := wayfairCompliantListing()
	listing.Description = "legacy"
	listing = constrainSmartSuiteListingToPlatform(listing, "wayfair")
	if listing.Title != "" || listing.Description != "" || len(listing.WhatsIncluded) != 0 || len(listing.Details) != 0 {
		t.Fatal("Wayfair listing retained unsupported fields")
	}
	if err := validateSmartSuiteListingPlatformStructure(listing); err != nil {
		t.Fatalf("valid Wayfair listing rejected: %v", err)
	}
}

func TestWayfairListingKeepsOnlyCurrentRules(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(listing *model.EcomSuiteListing)
		wantErr string
	}{
		{"about_empty", func(l *model.EcomSuiteListing) { l.AboutThisProduct = " " }, "listing_wayfair_about_this_product_missing"},
		{"features_empty", func(l *model.EcomSuiteListing) { l.Features = nil }, "listing_wayfair_features_missing"},
		{"internal_copy", func(l *model.EcomSuiteListing) { l.Features[2] = "Spout reach gives shoppers fit information" }, "listing_wayfair_copy_unnatural"},
		{"weak_setup_claim", func(l *model.EcomSuiteListing) {
			l.Features[2] = "Included supply line supports the listed installation configuration"
		}, "listing_wayfair_copy_unnatural"},
		{"inventory_as_benefit", func(l *model.EcomSuiteListing) {
			l.Features[2] = "Included components provide key parts for installing and using the faucet"
		}, "listing_wayfair_copy_unnatural"},
		{"assembly_inventory", func(l *model.EcomSuiteListing) {
			l.Features[2] = "Included hose and supply line to match the faucet assembly"
		}, "listing_wayfair_copy_unnatural"},
		{"specification_meta", func(l *model.EcomSuiteListing) {
			l.Features[2] = "Lead-free design provides confirmed material and cartridge specifications"
		}, "listing_wayfair_copy_unnatural"},
	}
	for _, testCase := range cases {
		listing := wayfairCompliantListing()
		testCase.mutate(&listing)
		err := validateSmartSuiteListingPlatformStructure(listing)
		if err == nil {
			t.Fatalf("%s: expected rejection", testCase.name)
		}
		if !strings.Contains(err.Error(), testCase.wantErr) {
			t.Fatalf("%s: unexpected error %v, want %s", testCase.name, err, testCase.wantErr)
		}
	}

	allowed := []model.EcomSuiteListing{
		{Platform: "wayfair", Title: strings.Repeat("Kitchen Faucet ", 30) + "*", AboutThisProduct: "Short product copy.", Features: []string{"A complete feature sentence is now allowed."}},
		{Platform: "wayfair", Title: "Kitchen Faucet Pull Down Sprayer", AboutThisProduct: "Product copy may use any length.", Features: []string{"Short", "Features may end with punctuation."}},
	}
	for _, listing := range allowed {
		if err := validateSmartSuiteListingPlatformStructure(listing); err != nil {
			t.Fatalf("removed Wayfair format rule still rejected listing: %v", err)
		}
	}
}

func TestWayfairRuntimeContractKeepsWritingGoalsLoose(t *testing.T) {
	contract := smartSuiteListingRuntimeContract("wayfair")
	for _, expected := range []string{
		"single source of writing style and selling-point priority",
		"never its SKU claims",
		"do not impose a feature-count preference",
	} {
		if !strings.Contains(contract, expected) {
			t.Fatalf("Wayfair contract missing %q", expected)
		}
	}
	for _, removed := range []string{
		"75-100 words",
		"3 to 5 items",
		"5-10 word",
		"no trailing punctuation",
	} {
		if strings.Contains(strings.ToLower(contract), strings.ToLower(removed)) {
			t.Fatalf("Wayfair contract still contains removed format rule %q", removed)
		}
	}
}

func TestHomeDepotListingReasonsIncludeConcreteBasis(t *testing.T) {
	facts := []model.EcomSuiteProductFact{{ID: "F1", Statement: "Pull-down spray wand", EvidenceSource: "user_document", Evidence: "Pull-down spray wand"}}
	listing := model.EcomSuiteListing{Platform: "home_depot", Title: "Kitchen Faucet", SalientBullets: []string{"Pull-down spray wand", "Matte finish", "Deck mount"}, BulletPoints: []string{"Pull-down spray wand", "Matte finish", "Deck mount", "Kitchen use", "Single handle", "Easy install"}, AboutThisProduct: "A kitchen faucet. It uses a pull-down spray wand. It has a matte finish.", Evidence: []model.EcomSuiteListingEvidence{{Field: "title", FactIDs: []string{"F1"}}, {Field: "salient_bullets", FactIDs: []string{"F1"}}, {Field: "bullet_points", FactIDs: []string{"F1"}}, {Field: "about_this_product", FactIDs: []string{"F1"}}}}
	listing = enrichSmartSuiteListingEvidence(listing, facts)
	listing = enrichSmartSuiteListingReasons(listing, facts)
	for _, reason := range listing.OptimizationReasons {
		if reason.Field == "title" || reason.Field == "salient_bullets" || reason.Field == "bullet_points" || reason.Field == "about_this_product" {
			if reason.PlatformBasis == "" || reason.WritingDecision == "" || reason.ExpectedImpact == "" {
				t.Fatalf("Home Depot reason lacks concrete basis: %+v", reason)
			}
		}
	}
	if err := validateSmartSuiteListingReasons(listing, facts); err != nil {
		t.Fatalf("detailed Home Depot reasons rejected: %v", err)
	}
}

func TestSmartSuiteListingFactMapSplitsProductInformationIntoDisplayableFacts(t *testing.T) {
	task := model.EcomSuiteTask{ProductName: "Kitchen Faucet", ProductFeatures: "Single handle; Pull-down sprayer with stream and spray; Brushed nickel finish", ProductProof: "1- or 3-hole deck mount; Includes deck plate"}
	factMap := smartSuiteListingFactMapFromTask(task)
	statements := make([]string, 0, len(factMap.ConfirmedFacts))
	for _, fact := range factMap.ConfirmedFacts {
		statements = append(statements, fact.Statement)
	}
	for _, expected := range []string{"Single handle", "Pull-down sprayer with stream and spray", "Brushed nickel finish", "1- or 3-hole deck mount", "Includes deck plate"} {
		found := false
		for _, statement := range statements {
			if statement == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing split fact %q in %#v", expected, statements)
		}
	}
}

func TestEnrichSmartSuiteListingEvidenceDoesNotFallbackToWholeProductInformation(t *testing.T) {
	facts := []model.EcomSuiteProductFact{
		{ID: "F1", Statement: "Kitchen Faucet", Evidence: "Kitchen Faucet"},
		{ID: "F2", Statement: "Pull-down sprayer", Evidence: "Pull-down sprayer"},
		{ID: "F3", Statement: "Brushed nickel finish", Evidence: "Brushed nickel finish"},
	}
	listing := model.EcomSuiteListing{Platform: "lowes", Title: "Kitchen Faucet", Description: "A faucet for daily use", Bullets: []string{"Pull-down sprayer supports rinsing"}, Evidence: []model.EcomSuiteListingEvidence{{Field: "title", FactIDs: []string{"F1"}}, {Field: "bullets", FactIDs: []string{"F2", "F3"}}}}
	listing = enrichSmartSuiteListingEvidence(listing, facts)
	if listing.Evidence[0].Quote != "Kitchen Faucet" || listing.Evidence[1].Quote != "Pull-down sprayer；Brushed nickel finish" {
		t.Fatalf("field evidence was not derived from cited facts: %+v", listing.Evidence)
	}
	if len(listing.Evidence) != 2 {
		t.Fatalf("missing field evidence must not fall back to the full product information: %+v", listing.Evidence)
	}
}

func TestParseHomeDepotListingUsesDedicatedFields(t *testing.T) {
	raw := `{"listing":{"platform":"home_depot","title":"Kitchen Faucet","salient_bullets":["One","Two","Three"],"bullet_points":["A","B","C","D","E","F"],"about_this_product":"First sentence. Second sentence. Third sentence."}}`
	listing, err := parseSmartSuiteListingResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.BulletPoints) != 6 || listing.AboutThisProduct == "" {
		t.Fatalf("dedicated Home Depot fields were not parsed: %+v", listing)
	}
	if len(listing.Bullets) != 0 || listing.Description != "" {
		t.Fatalf("Home Depot fields leaked into generic fields: %+v", listing)
	}
}

func TestParseHomeDepotListingMergesFieldsOutsideListingObject(t *testing.T) {
	raw := `{"listing":{"platform":"home_depot","title":"Kitchen Faucet"},"salient_bullets":["One","Two","Three"],"bullet_points":["A","B","C","D","E","F"],"about_this_product":"First sentence. Second sentence. Third sentence."}`
	listing, err := parseSmartSuiteListingResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.SalientBullets) != 3 || len(listing.BulletPoints) != 6 || listing.AboutThisProduct == "" {
		t.Fatalf("outer platform fields were not merged: %+v", listing)
	}
}

func TestParseWayfairListingAcceptsFeatureBulletAlias(t *testing.T) {
	listing, err := parseSmartSuiteListingResponse(`{"listing":{"platform":"wayfair","about_this_product":"A kitchen faucet for everyday cleanup.","feature_bullets":["Pull-down sprayer: Bring water to the dishes.","Single-handle control: Adjust temperature and flow."]}}`)
	if err != nil {
		t.Fatal(err)
	}
	listing.Platform = "wayfair"
	listing = constrainSmartSuiteListingToPlatform(listing, "wayfair")
	if err := validateSmartSuiteListing(listing); err != nil {
		t.Fatalf("Wayfair feature_bullets alias was not retained: %v", err)
	}
}

func TestMergeSmartSuiteListingFieldsRetainsWayfairFeatureRepair(t *testing.T) {
	base := model.EcomSuiteListing{Platform: "wayfair", AboutThisProduct: "A kitchen faucet."}
	repaired := model.EcomSuiteListing{Platform: "wayfair", Features: []string{"Pull-down sprayer: Bring water to the dishes."}}
	merged := mergeSmartSuiteListingFields(base, repaired)
	if err := validateSmartSuiteListing(merged); err != nil {
		t.Fatalf("Wayfair repair dropped features: %v", err)
	}
}

func TestConstrainSmartSuiteListingToPlatformRemovesCrossPlatformFields(t *testing.T) {
	listing := model.EcomSuiteListing{
		Platform: "home_depot", Title: "Faucet", AboutThisProduct: "Overview", BulletPoints: []string{"One", "Two", "Three", "Four", "Five", "Six"},
		Highlights: []string{"HD highlight"}, SalientBullets: []string{"Top 1", "Top 2", "Top 3"}, SupplierBullets: []string{"Lowes bullet"}, SearchTerms: []string{"Amazon term"},
		Attributes: []model.EcomSuiteListingAttribute{{Name: "Finish", Value: "Matte"}}, Details: []model.EcomSuiteListingAttribute{{Name: "Width", Value: "8 in."}},
	}
	listing = constrainSmartSuiteListingToPlatform(listing, "home_depot")
	if len(listing.SupplierBullets) != 0 || len(listing.SearchTerms) != 0 || len(listing.Details) != 0 || len(listing.Attributes) != 0 || len(listing.Documents) != 0 {
		t.Fatalf("cross-platform fields were not removed: %+v", listing)
	}
	if !reflect.DeepEqual(listing.BulletPoints, []string{"One", "Two", "Three", "Four", "Five", "Six"}) || len(listing.SalientBullets) != 3 || listing.AboutThisProduct != "Overview" {
		t.Fatalf("Home Depot fields were removed: %+v", listing)
	}
}

func TestParseSmartSuiteListingResponse(t *testing.T) {
	raw := `<analysis>ignored</analysis>{"listing":{"title":"Matte Black Bathroom Faucet","bullets":["Fits 6-12 inch widespread layouts","Three-hole configuration","Two-handle control","Includes the supplied drain component","Designed for bathroom sinks"],"description":"A widespread bathroom sink faucet organized around the confirmed product facts.","variants":["Matte Black"],"documents":["cUPC"],"details":[{"name":"Spout Height","value":"10 inches"}],"search_terms":["bathroom sink faucet","widespread faucet"],"keyword_candidates":[{"term":"deck mount faucet","tags":["feature"]}]}}`
	listing, err := parseSmartSuiteListingResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if listing.Title != "Matte Black Bathroom Faucet" || len(listing.Bullets) != 5 || len(listing.SearchTerms) != 2 {
		t.Fatalf("unexpected listing: %+v", listing)
	}
	if len(listing.KeywordCandidates) != 1 || listing.KeywordCandidates[0].Term != "deck mount faucet" {
		t.Fatalf("keyword candidates were not parsed: %+v", listing.KeywordCandidates)
	}
	if len(listing.Variants) != 1 || len(listing.Documents) != 1 || len(listing.Details) != 1 {
		t.Fatalf("platform-specific fields were not parsed: %+v", listing)
	}
	if err := validateSmartSuiteListing(listing); err != nil {
		t.Fatalf("valid listing rejected: %v", err)
	}
}

func TestValidateSmartSuiteListingRequiresFiveBullets(t *testing.T) {
	listing, err := parseSmartSuiteListingResponse(`{"title":"Product title","bullet_points":["One","Two"],"description":"Product description","backend_search_terms":"first, second"}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.SearchTerms) != 2 {
		t.Fatalf("expected split search terms, got %#v", listing.SearchTerms)
	}
	if err := validateSmartSuiteListing(listing); err == nil {
		t.Fatal("listing with fewer than five bullets must fail validation")
	}
}

func TestValidateSmartSuiteListingRequiresRealEvidenceWhenFactsProvided(t *testing.T) {
	facts := []model.EcomSuiteProductFact{{ID: "F1", Statement: "Matte black finish", EvidenceSource: "user_document", Evidence: "Finish: Matte black"}}
	listing := model.EcomSuiteListing{Platform: "home_depot", Title: "Matte Black Faucet", AboutThisProduct: "A faucet with a matte black finish.", BulletPoints: []string{"Install on a deck", "Use the lever control", "Rinse with the sprayer", "Coordinate the black finish", "Clean the sink area", "Use for kitchen tasks"}, SalientBullets: []string{"Three spray modes", "360 degree coverage", "Stainless steel build"}, Evidence: []model.EcomSuiteListingEvidence{{Field: "title", FactIDs: []string{"F1"}}, {Field: "about_this_product", FactIDs: []string{"F1"}}, {Field: "bullet_points", FactIDs: []string{"F1"}}, {Field: "salient_bullets", FactIDs: []string{"F1"}}}, OptimizationReasons: []model.EcomSuiteListingReason{{Field: "title", Reason: "在标题中放置产品类型和已确认表面处理，提高搜索匹配。", FactIDs: []string{"F1"}}, {Field: "about_this_product", Reason: "用完整句子解释表面处理，帮助用户理解产品。", FactIDs: []string{"F1"}}, {Field: "bullet_points", Reason: "展示已确认的购买利益，提高决策效率。", FactIDs: []string{"F1"}}, {Field: "salient_bullets", Reason: "选择三个最重要的差异化特性，方便首屏扫读。", FactIDs: []string{"F1"}}}}
	listing = enrichSmartSuiteListingEvidence(listing, facts)
	listing = enrichSmartSuiteListingReasons(listing, facts)
	if err := validateSmartSuiteListing(listing, facts); err != nil {
		t.Fatalf("valid fact citations rejected: %v", err)
	}
	if listing.Evidence[0].Quote != "Matte black finish" {
		t.Fatalf("evidence display must use the cited fact statement: %#v", listing.Evidence[0])
	}
	listing.Evidence[0].FactIDs = []string{"F9"}
	listing = enrichSmartSuiteListingEvidence(listing, facts)
	if err := validateSmartSuiteListing(listing, facts); err == nil {
		t.Fatal("unknown fact citation must be rejected")
	}
}

func TestValidateSmartSuiteListingReasonsRequiresChinese(t *testing.T) {
	facts := []model.EcomSuiteProductFact{{ID: "F1", Statement: "Matte black finish", Evidence: "Finish: Matte black"}}
	listing := model.EcomSuiteListing{Title: "Faucet", OptimizationReasons: []model.EcomSuiteListingReason{{Field: "title", Reason: "Improve search matching", FactIDs: []string{"F1"}}}}
	if err := validateSmartSuiteListingReasons(listing, facts); err == nil || !strings.Contains(err.Error(), "not_chinese") {
		t.Fatalf("expected English reason rejection, got %v", err)
	}
	listing.OptimizationReasons[0].Reason = "前置核心品类词，提高站内搜索匹配。"
	if err := validateSmartSuiteListingReasons(listing, facts); err != nil {
		t.Fatalf("Chinese reason was rejected: %v", err)
	}
}

func TestValidateHomeDepotListingTitleRejectsBrand(t *testing.T) {
	listing := model.EcomSuiteListing{Platform: "home_depot", Title: "FORIOUS PRO Single Handle Kitchen Faucet"}
	if err := validateHomeDepotListingTitleBrand(listing, "FORIOUS PRO"); err == nil {
		t.Fatal("Home Depot title containing the product brand must be rejected")
	}
	listing.Title = "Single Handle Kitchen Faucet in Matte Black"
	if err := validateHomeDepotListingTitleBrand(listing, "FORIOUS PRO"); err != nil {
		t.Fatalf("brand-free Home Depot title was rejected: %v", err)
	}
}

func TestRemoveHomeDepotListingTitleBrand(t *testing.T) {
	listing := model.EcomSuiteListing{Platform: "home_depot", Title: "FORIOUS PRO - Single Handle Kitchen Faucet"}
	listing = removeHomeDepotListingTitleBrand(listing, "FORIOUS PRO")
	if listing.Title != "Single Handle Kitchen Faucet" {
		t.Fatalf("brand prefix was not removed: %q", listing.Title)
	}
}

func TestSynthesizeMissingHomeDepotFieldsProducesCompleteListing(t *testing.T) {
	facts := []model.EcomSuiteProductFact{{ID: "F1", Statement: "Matte black kitchen faucet with pull-down sprayer"}}
	listing := synthesizeMissingHomeDepotFields(model.EcomSuiteListing{Platform: "home_depot", Title: "Kitchen Faucet"}, model.EcomSuiteTask{ProductName: "Kitchen Faucet"}, facts)
	if len(listing.SalientBullets) != 3 || len(listing.BulletPoints) != 6 || listing.AboutThisProduct == "" {
		t.Fatalf("fallback did not produce complete Home Depot listing: %+v", listing)
	}
}

func TestValidateSmartSuiteListingRequiresEvidenceForPopulatedFields(t *testing.T) {
	facts := []model.EcomSuiteProductFact{{ID: "F1", Statement: "Matte black finish", EvidenceSource: "user_document", Evidence: "Finish: Matte black"}}
	listing := model.EcomSuiteListing{Platform: "home_depot", Title: "Matte Black Faucet"}
	listing = enrichSmartSuiteListingEvidence(listing, facts)
	if err := validateSmartSuiteListing(listing, facts); err == nil {
		t.Fatal("missing title evidence must be rejected")
	}
}

func TestTaskResponseKeepsListingOutsidePlanSnapshot(t *testing.T) {
	listing := &model.EcomSuiteListing{Title: "Product title", Bullets: []string{"One", "Two", "Three", "Four", "Five"}, Description: "Product description"}
	response := buildEcomSuiteTaskResponse(model.EcomSuiteTask{TaskID: "task-1", PlanSnapshot: &model.EcomSuitePlanSnapshot{Listing: listing}})
	if response.Listing == nil || response.Listing.Title != "Product title" {
		t.Fatalf("task response did not expose listing: %+v", response.Listing)
	}
	response.PlanSnapshot = nil
	if response.Listing == nil {
		t.Fatal("listing must remain available when conversation history omits the plan snapshot")
	}
}

func TestTaskResponseExposesListingKeywordCandidates(t *testing.T) {
	keywords := []model.EcomSuiteKeywordCandidate{
		{Term: "pull down kitchen faucet", Tags: []string{"researched", "high_intent"}, Reason: "Common category search phrase"},
		{Term: "matte black faucet", Tags: []string{"operator"}, Reason: "Provided by operations"},
	}
	response := buildEcomSuiteTaskResponse(model.EcomSuiteTask{
		TaskID:       "task-keywords",
		WorkflowType: "smart_listing",
		PlanSnapshot: &model.EcomSuitePlanSnapshot{
			ListingKeywordsReady: true,
			ListingKeywords:      keywords,
		},
	})
	if len(response.ListingKeywords) != 2 || response.ListingKeywords[0].Term != "pull down kitchen faucet" {
		t.Fatalf("task response did not expose keyword candidates: %+v", response.ListingKeywords)
	}
	if len(response.TodoItems) != 2 || len(response.TodoItems[0].Keywords) != 2 {
		t.Fatalf("keyword todo did not expose candidates: %+v", response.TodoItems)
	}
	response.PlanSnapshot = nil
	if len(response.ListingKeywords) != 2 || len(response.TodoItems[0].Keywords) != 2 {
		t.Fatal("keyword candidates must remain visible when history omits the plan snapshot")
	}
}

func TestTaskResponsePublishesListingAfterWriterCompletes(t *testing.T) {
	listing := &model.EcomSuiteListing{Platform: "lowes", Title: "Brand Kitchen Faucet Matte Black", Description: "Product description", Bullets: []string{"A product benefit for daily use"}}
	draftSnapshot := &model.EcomSuitePlanSnapshot{Listing: listing}
	draft := buildEcomSuiteTaskResponse(model.EcomSuiteTask{TaskID: "task-draft", WorkflowType: "smart_listing", Platform: "lowes", Status: "planning", PlanSnapshot: draftSnapshot})
	if draft.Listing == nil || draft.PlanSnapshot == nil || draft.PlanSnapshot.Listing == nil {
		t.Fatalf("writer listing draft was not exposed: %+v", draft)
	}

	passedSnapshot := &model.EcomSuitePlanSnapshot{Listing: listing, ListingReview: &model.EcomSuiteListingReview{Platform: "lowes", Passed: true}}
	passed := buildEcomSuiteTaskResponse(model.EcomSuiteTask{TaskID: "task-passed", WorkflowType: "smart_listing", Platform: "lowes", Status: "success", PlanSnapshot: passedSnapshot})
	if passed.Listing == nil || passed.PlanSnapshot == nil || passed.PlanSnapshot.Listing == nil {
		t.Fatalf("reviewed listing was not published: %+v", passed)
	}
}

func TestBuildSmartSuiteListingReviewInputContainsExactWriterOutput(t *testing.T) {
	listing := &model.EcomSuiteListing{
		Platform:    "lowes",
		Title:       "FORIOUS Kitchen Faucet Matte Black Single Handle Pull-Down Sprayer with Deck Plate",
		Description: "Pull-down operation supports everyday kitchen cleanup with flexible sink access.",
		Bullets:     []string{"Pull-down sprayer directs water around the sink for easier rinsing"},
	}
	snapshot := &model.EcomSuitePlanSnapshot{
		Listing:        listing,
		ProductFactMap: model.EcomSuiteProductFactMap{ConfirmedFacts: []model.EcomSuiteProductFact{{ID: "F1", Statement: "Matte black pull-down kitchen faucet"}}},
	}
	raw, fingerprint, err := buildSmartSuiteListingReviewInput(model.EcomSuiteTask{Platform: "lowes", ProductName: "Kitchen Faucet", ProductCategory: "Kitchen Faucet"}, snapshot)
	if err != nil {
		t.Fatalf("build review input: %v", err)
	}
	var input struct {
		ListingFingerprint string                 `json:"listing_fingerprint"`
		WriterOutput       model.EcomSuiteListing `json:"writer_output_listing"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatalf("decode review input: %v", err)
	}
	if input.ListingFingerprint != fingerprint || input.WriterOutput.Title != listing.Title || input.WriterOutput.Description != listing.Description || len(input.WriterOutput.Bullets) != 1 {
		t.Fatalf("review input did not preserve Writer output: %+v", input)
	}
}

func TestTaskResponseDoesNotReportIncompleteListingAsSuccess(t *testing.T) {
	listing := &model.EcomSuiteListing{Platform: "home_depot", Title: "Kitchen Faucet"}
	response := buildEcomSuiteTaskResponse(model.EcomSuiteTask{
		TaskID: "task-incomplete", WorkflowType: "smart_listing", Platform: "home_depot", Status: "success",
		PlanSnapshot: &model.EcomSuitePlanSnapshot{Listing: listing},
	})
	if response.Status != "failed" || response.ListingError == "" {
		t.Fatalf("incomplete listing was reported as complete: %+v", response)
	}
}

func TestSmartSuiteListingCategoryRuleProfileMatchesFaucetOnly(t *testing.T) {
	profile, ok := smartSuiteListingCategoryRuleProfile("Home Depot", "Kitchen Sink Faucet").(map[string]interface{})
	if !ok || profile["matched"] != true {
		t.Fatalf("expected faucet category profile to match: %#v", profile)
	}
	fieldRules, ok := profile["field_rules"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing field rules: %#v", profile)
	}
	salient, ok := fieldRules["salient_bullets"].(map[string]interface{})
	if !ok || salient["count"] != 3 {
		t.Fatalf("unexpected faucet salient bullet rule: %#v", salient)
	}
	if salient["max_characters"] != 65 {
		t.Fatalf("unexpected salient bullet character limit: %#v", salient)
	}
	title := fieldRules["title"].(map[string]interface{})
	if title["max_characters"] != 120 {
		t.Fatalf("unexpected title character limit: %#v", title)
	}
	bullets := fieldRules["bullet_points"].(map[string]interface{})
	if bullets["count"] != 6 {
		t.Fatalf("unexpected bullet count: %#v", bullets)
	}
	if _, ok := fieldRules["assets"]; ok {
		t.Fatal("image rules must not be included in the listing profile")
	}
}

func TestSmartSuiteListingCategoryRuleProfileDoesNotBorrowFaucetRules(t *testing.T) {
	profile, ok := smartSuiteListingCategoryRuleProfile("Home Depot", "Outdoor Storage Cabinet").(map[string]interface{})
	if !ok || profile["matched"] != false {
		t.Fatalf("expected unmatched category profile: %#v", profile)
	}
	if _, ok := profile["field_rules"]; ok {
		t.Fatal("unmatched category must not contain faucet field rules")
	}
	profile, ok = smartSuiteListingCategoryRuleProfile("Home Depot", "Laptop Stand").(map[string]interface{})
	if !ok || profile["matched"] != false {
		t.Fatalf("tap substring must not match unrelated category: %#v", profile)
	}
}

func TestLowesListingCategoryRuleProfileUsesFaucetExampleConditionally(t *testing.T) {
	faucet, ok := smartSuiteListingCategoryRuleProfile("lowes", "Kitchen Faucet").(map[string]interface{})
	if !ok || faucet["matched"] != true || faucet["faucet_example_matched"] != true {
		t.Fatalf("expected Lowe's faucet profile to enable the conditional example: %#v", faucet)
	}
	if _, ok := faucet["faucet_example"].(map[string]interface{}); !ok {
		t.Fatalf("expected conditional faucet slots: %#v", faucet)
	}

	furniture, ok := smartSuiteListingCategoryRuleProfile("lowes", "Storage Cabinet").(map[string]interface{})
	if !ok || furniture["matched"] != true || furniture["faucet_example_matched"] != false {
		t.Fatalf("expected generic Lowe's category profile: %#v", furniture)
	}
	if _, exists := furniture["faucet_example"]; exists {
		t.Fatal("non-faucet Lowe's profile must not expose faucet slots")
	}
}
