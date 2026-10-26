package handler

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"advertising-image/backend/internal/adapter"
	"advertising-image/backend/internal/database"
	"advertising-image/backend/internal/model"
	"advertising-image/backend/internal/smartagent"
)

var errSmartSuiteListingReviewRecheckRequired = errors.New("listing_review_recheck_required")

func smartSuiteListingOutputStructure(platform string) []string {
	switch normalizeListingPlatform(platform) {
	case "tiktok":
		return []string{"title", "description", "attributes", "variants", "whats_included"}
	case "home_depot":
		return []string{"title", "salient_bullets", "bullet_points", "about_this_product"}
	case "lowes":
		return []string{"title", "description", "bullets"}
	case "wayfair":
		return []string{"about_this_product", "features"}
	case "amazon":
		return []string{"title", "bullets", "description", "search_terms"}
	default:
		return []string{"title", "description", "attributes"}
	}
}

func smartSuiteListingAgent(platform string) smartagent.Name {
	switch normalizeListingPlatform(platform) {
	case "tiktok":
		return smartagent.WriteTikTokListing
	case "home_depot":
		return smartagent.WriteHomeDepotListing
	case "lowes":
		return smartagent.WriteLowesListing
	case "wayfair":
		return smartagent.WriteWayfairListing
	case "amazon":
		return smartagent.WriteAmazonListing
	default:
		return smartagent.WriteProductListing
	}
}

func isSmartSuiteListingAgent(agent string) bool {
	for _, name := range []smartagent.Name{
		smartagent.WriteProductListing,
		smartagent.WriteTikTokListing,
		smartagent.WriteHomeDepotListing,
		smartagent.WriteLowesListing,
		smartagent.WriteWayfairListing,
		smartagent.WriteAmazonListing,
	} {
		if agent == string(name) {
			return true
		}
	}
	return false
}

func smartSuiteListingRuntimeContract(platform string) string {
	common := ` Every populated listing field and every keyword candidate must have one evidence entry with one or more confirmed fact_ids and one optimization_reasons entry. Every optimization_reasons.reason, platform_basis, writing_decision, and expected_impact must be detailed Simplified Chinese for marketplace operators; only the listing copy uses the target marketplace language. Explain the platform decision supported, the exact organization or wording choice, and the concrete effect on search recognition, comparison, installation or purchase judgment, compliance, or content quality. Listing copy must meet the operator quality standard: use confirmed high-value search terms naturally; for Home Depot, Salient Bullets are concise product highlights while Bullet Points are feature-plus-benefit marketing selling points; for other platforms, pair features with concrete shopper benefits or supported use scenarios as their field rules require. For North American retail copy, prioritize product identification and core task value, then operation/space and installation fit, then maintenance/quality, and finally compliance or warranty; use only supported layers and do not let low-priority facts displace the core selling point. Consumer-facing copy must describe the product and its use directly, never explain what the copy does for shoppers or the platform; do not write meta phrases such as fit information, purchase decision, search recognition, helps shoppers decide, or gives buyers guidance. Assign distinct responsibilities to fields, and avoid repeated copy, bare specification lists, vague marketing phrases, unexplained negative attributes, or unsupported quantitative/material/accessory/certification/durability claims. Cite only IDs present in confirmed_facts; do not cite inferences. Apply category_rule_profile only when matched. keyword_candidates is internal and may contain at most 20 reusable search phrases supported by confirmed facts. Return JSON only; do not return image copy, unsupported claims, or analysis.`
	if normalizeListingPlatform(platform) == "wayfair" {
		common = strings.Replace(common, "For North American retail copy, prioritize product identification and core task value, then operation/space and installation fit, then maintenance/quality, and finally compliance or warranty; use only supported layers and do not let low-priority facts displace the core selling point.", "Use the Wayfair skill's buyer-specific selling-point priority.", 1)
		return common + smartSuiteWayfairRuntimeContract()
	}
	if normalizeListingPlatform(platform) == "lowes" {
		common = ` Lowe's Bullet lead is mandatory: begin every bullet with the confirmed searchable feature, material, specification, installation item, finish, accessory, or warranty noun phrase. Put the highest-value supported SEO phrase in the opening words, then explain a concrete use scenario, shopper benefit, and evidence-based reassurance. Do not begin with The, This, These, It, They, Provides, Designed, Features, Offers, Helps, Allows, Supports, Includes, Comes, or Made. Do not write internal audit language or bare facts. Search visibility is a writing objective, never a promised traffic result.` + common
	}
	switch normalizeListingPlatform(platform) {
	case "tiktok":
		return `Return exactly {"listing":{"platform":"tiktok","title":"","description":"","attributes":[{"name":"","value":""}],"variants":[],"whats_included":[],"evidence":[{"field":"title|description|attributes|variants|whats_included|keyword_candidates","fact_ids":["F1"],"source":"","quote":""}],"optimization_reasons":[{"field":"title|description|attributes|variants|whats_included|keyword_candidates","reason":"","rule":"","fact_ids":["F1"]}],"keyword_candidates":[{"term":"","tags":[],"reason":""}]}}.` + common
	case "home_depot":
		return `Return exactly {"listing":{"platform":"home_depot","title":"","salient_bullets":["","",""],"bullet_points":["","","","","",""],"about_this_product":"","evidence":[{"field":"title|salient_bullets|bullet_points|about_this_product|keyword_candidates","fact_ids":["F1"],"source":"","quote":""}],"optimization_reasons":[{"field":"title|salient_bullets|bullet_points|about_this_product|keyword_candidates","reason":"","rule":"","platform_basis":"","writing_decision":"","expected_impact":"","fact_ids":["F1"]}],"keyword_candidates":[{"term":"","tags":[],"reason":""}]}}. title is at most 120 characters and must naturally include the highest-frequency researched category search term supported by confirmed facts, without keyword stuffing. about_this_product is at most 1500 characters and 3-6 sentences. salient_bullets contains exactly 3 distinct items, each at most 65 characters and using the available length effectively without filler; every item must use Feature + direct shopper benefit wording that tells the buyer what they gain, not only what the product is. bullet_points contains exactly 6 distinct one-sentence items with no trailing period; each must connect function benefit + supported use condition, and where confirmed, include a relevant dimension/specification and compliance or certification detail. Do not include the manufacturer brand in title. All listing copy must use natural US English with US retail and installation terminology, natural noun order, familiar abbreviations, and fact-supported units; avoid literal Chinese translation, mixed-language copy, and unnatural keyword strings. Every optimization_reasons item must provide detailed Simplified Chinese platform_basis, writing_decision, and expected_impact: explain the Home Depot project-shopping decision this field supports, the exact organization or wording choice made from cited facts, and the concrete effect on search recognition, compatibility or installation judgment, product comparison, purchase confidence, or compliance risk. Do not invent demographic research or claim that all consumers share one preference. reason is a concise synthesis of those three parts, never merely a rule name or generic conversion claim. Do not output bullets, description, Highlights, Specifications, Documents, or cross-platform fields.` + common
	case "lowes":
		return `Return exactly {"listing":{"platform":"lowes","title":"","description":"","bullets":[],"evidence":[{"field":"title|description|bullets|keyword_candidates","fact_ids":["F1"],"source":"","quote":""}],"optimization_reasons":[{"field":"title|description|bullets|keyword_candidates","reason":"","rule":"","platform_basis":"","writing_decision":"","expected_impact":"","fact_ids":["F1"]}],"keyword_candidates":[{"term":"","tags":[],"reason":""}]}}. First determine which product attributes and selling-point slots are applicable from confirmed facts and visible image evidence. Faucet terms below are category examples, never universal requirements. title must be 253 characters or fewer, one line, and use plain text/basic punctuation only. Its slot order is strict: Brand + Product Type + Material/Finish + Handle Configuration + Mounting/Function + Sprayer + Included Accessory. Every supported slot must remain in this exact left-to-right order; omit inapplicable or unsupported slots, never move a later slot before an earlier one for SEO. For a confirmed kitchen faucet, preserve complete high-value phrases such as Kitchen Faucet or Kitchen Sink Faucet and add material/finish after the product type, Single Handle after finish, Pull-Down or mounting/function after handle, Sprayer after mounting/function, and Deck Plate only at the end when confirmed. For every other category, substitute that category's own product type, defining specifications, functions, fit/mounting and included accessory; never insert faucet language. description must be one concise SEO-aware marketing sentence, at most 2000 characters, organized as supported product functions -> multiple concrete use scenarios -> overall consumer experience. It summarizes the bullets without mechanically repeating their sentences and naturally distributes 3-5 relevant SEO phrases only when enough supported terms exist. bullets use one complete sentence per applicable selling-point dimension, each at most 254 characters; prefer 100-180 characters, while 80-140 is acceptable when complete. Each bullet connects a confirmed material/function/specification to a supported use scenario and direct consumer benefit. Dynamically cover material, core functions, reach/operation, installation/compatibility/accessories, performance/efficiency, control/reliability, finish/style/care, and warranty/support only where the product facts support them. Do not pad bullet count or length, and do not penalize or invent an unsupported dimension. Use the uploaded image only for directly visible structure, relative proportions, finish and conservative placement/use context; never infer hidden mechanisms, dimensions, certifications, accessories, performance or warranty from the image. Use natural US English and familiar Lowe's retail terminology; do not translate Chinese literally or mix languages. Never use all caps, "Better than", "Compared with", "Guaranteed", unsupported absolutes, or competitor comparisons. Do not output lowes_product_description, supplier_bullets, marketing_copy, attributes, or cross-platform fields.` + common
	case "wayfair":
		return smartSuiteWayfairRuntimeContract() + common
	case "amazon":
		return `Return exactly {"listing":{"platform":"amazon","title":"","bullets":["","","","",""],"description":"","search_terms":[],"evidence":[{"field":"title|bullets|description|search_terms|keyword_candidates","fact_ids":["F1"],"source":"","quote":""}],"optimization_reasons":[{"field":"title|bullets|description|search_terms|keyword_candidates","reason":"","rule":"","fact_ids":["F1"]}],"keyword_candidates":[{"term":"","tags":[],"reason":""}]}}. bullets must contain exactly five distinct items.` + common
	default:
		return `Return exactly {"listing":{"platform":"","title":"","description":"","attributes":[{"name":"","value":""}],"evidence":[{"field":"title|description|attributes|keyword_candidates","fact_ids":["F1"]}],"optimization_reasons":[{"field":"title|description|attributes|keyword_candidates","reason":"","rule":"","fact_ids":["F1"]}],"keyword_candidates":[{"term":"","tags":[],"reason":""}]}}.` + common
	}
}

func smartSuiteWayfairRuntimeContract() string {
	return `Return exactly {"listing":{"platform":"wayfair","about_this_product":"","features":[],"evidence":[{"field":"about_this_product|features|keyword_candidates","fact_ids":["F1"],"source":"","quote":""}],"optimization_reasons":[{"field":"about_this_product|features|keyword_candidates","reason":"","rule":"","platform_basis":"","writing_decision":"","expected_impact":"","fact_ids":["F1"]}],"keyword_candidates":[{"term":"","tags":[],"reason":""}]}}. Apply the Wayfair skill as the single source of writing style and selling-point priority. Learn only the supplied Oletto reference style and hierarchy, never its SKU claims. Use the exact positioning-specific Feature order from that skill; do not freely reorder confirmed selling points. Skip unavailable positions rather than inventing facts or promoting lower-priority details. Use consistent, informative Feature headings when helpful; do not impose a feature-count preference or ban product-subject sentences. About introduces the highest-priority benefits; Features expand them with supported specifics, without paraphrasing the same whole claims. Never invent performance, materials, dimensions, branded technology, certification, or warranty. Output no Title. Colors and finishes are allowed when supported by this SKU. Return the required JSON only.`
}

func constrainSmartSuiteListingToPlatform(listing model.EcomSuiteListing, platform string) model.EcomSuiteListing {
	if strings.EqualFold(strings.TrimSpace(platform), "lowes") {
		if strings.TrimSpace(listing.Description) == "" {
			listing.Description = listing.LowesProductDescription
		}
		if len(listing.Bullets) == 0 {
			listing.Bullets = listing.SupplierBullets
		}
	}
	platform = normalizeListingPlatform(platform)
	allowed := make(map[string]bool)
	for _, field := range smartSuiteListingOutputStructure(platform) {
		allowed[field] = true
	}
	if !allowed["title"] {
		listing.Title = ""
	}
	if !allowed["description"] {
		listing.Description = ""
	}
	if !allowed["bullets"] {
		listing.Bullets = nil
	}
	if !allowed["salient_bullets"] {
		listing.SalientBullets = nil
	}
	if !allowed["bullet_points"] {
		listing.BulletPoints = nil
	}
	if !allowed["about_this_product"] {
		listing.AboutThisProduct = ""
	}
	if !allowed["highlights"] {
		listing.Highlights = nil
	}
	if !allowed["supplier_bullets"] {
		listing.SupplierBullets = nil
	}
	if !allowed["marketing_copy"] {
		listing.MarketingCopy = ""
	}
	if !allowed["lowes_product_description"] {
		listing.LowesProductDescription = ""
	}
	if !allowed["features"] {
		listing.Features = nil
	}
	if !allowed["variants"] {
		listing.Variants = nil
	}
	if !allowed["whats_included"] {
		listing.WhatsIncluded = nil
	}
	if !allowed["not_included"] {
		listing.NotIncluded = nil
	}
	if !allowed["documents"] {
		listing.Documents = nil
	}
	if !allowed["attributes"] {
		listing.Attributes = nil
	}
	if !allowed["details"] {
		listing.Details = nil
	}
	if !allowed["search_terms"] {
		listing.SearchTerms = nil
	}
	filterEvidence := func(items []model.EcomSuiteListingEvidence) []model.EcomSuiteListingEvidence {
		result := items[:0]
		for _, item := range items {
			if allowed[normalizeSmartSuiteListingEvidenceField(item.Field)] || normalizeSmartSuiteListingEvidenceField(item.Field) == "keyword_candidates" {
				result = append(result, item)
			}
		}
		return result
	}
	filterReasons := func(items []model.EcomSuiteListingReason) []model.EcomSuiteListingReason {
		result := items[:0]
		for _, item := range items {
			if allowed[normalizeSmartSuiteListingEvidenceField(item.Field)] || normalizeSmartSuiteListingEvidenceField(item.Field) == "keyword_candidates" {
				result = append(result, item)
			}
		}
		return result
	}
	listing.Evidence = filterEvidence(listing.Evidence)
	listing.OptimizationReasons = filterReasons(listing.OptimizationReasons)
	return listing
}

func smartSuiteListingFactMapFromTask(task model.EcomSuiteTask) model.EcomSuiteProductFactMap {
	facts := make([]model.EcomSuiteProductFact, 0, 12)
	appendFact := func(statement, evidence string) {
		statement = strings.TrimSpace(statement)
		evidence = strings.TrimSpace(evidence)
		if statement == "" || evidence == "" {
			return
		}
		facts = append(facts, model.EcomSuiteProductFact{ID: fmt.Sprintf("F%d", len(facts)+1), Statement: statement, EvidenceSource: "user_document", Evidence: evidence})
	}
	appendProductFacts := func(value string) {
		for _, item := range splitSmartSuiteListingFactItems(value) {
			appendFact(item, item)
		}
	}
	appendFact("Product name: "+task.ProductName, task.ProductName)
	appendFact("Product category: "+task.ProductCategory, task.ProductCategory)
	appendFact("Product brand: "+task.ProductBrand, task.ProductBrand)
	appendProductFacts(task.ProductFeatures)
	appendProductFacts(task.ProductProof)
	if task.ProductLength != "" && task.ProductWidth != "" && task.ProductHeight != "" && task.ProductDimensionUnit != "" {
		appendFact(fmt.Sprintf("Product dimensions: %s x %s x %s %s", task.ProductLength, task.ProductWidth, task.ProductHeight, task.ProductDimensionUnit), fmt.Sprintf("%s x %s x %s %s", task.ProductLength, task.ProductWidth, task.ProductHeight, task.ProductDimensionUnit))
	}
	if task.ProductWeight != "" && task.ProductWeightUnit != "" {
		appendFact(fmt.Sprintf("Product weight: %s %s", task.ProductWeight, task.ProductWeightUnit), fmt.Sprintf("%s %s", task.ProductWeight, task.ProductWeightUnit))
	}
	if task.PackageLength != "" && task.PackageWidth != "" && task.PackageHeight != "" && task.PackageDimensionUnit != "" {
		appendFact(fmt.Sprintf("Package dimensions: %s x %s x %s %s", task.PackageLength, task.PackageWidth, task.PackageHeight, task.PackageDimensionUnit), fmt.Sprintf("%s x %s x %s %s", task.PackageLength, task.PackageWidth, task.PackageHeight, task.PackageDimensionUnit))
	}
	if task.PackageWeight != "" && task.PackageWeightUnit != "" {
		appendFact(fmt.Sprintf("Package weight: %s %s", task.PackageWeight, task.PackageWeightUnit), fmt.Sprintf("%s %s", task.PackageWeight, task.PackageWeightUnit))
	}
	appendFact("Package contents: "+task.PackageContents, task.PackageContents)
	return model.EcomSuiteProductFactMap{ConfirmedFacts: facts, Inferences: []model.EcomSuiteProductInference{}}
}

func splitSmartSuiteListingFactItems(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ';' || r == '；' || r == '•' || r == '·'
	})
	seen := make(map[string]struct{}, len(parts))
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(strings.TrimLeft(part, "-•* "))
		item = regexp.MustCompile(`^\d+[.)、）]\s*`).ReplaceAllString(item, "")
		if item == "" {
			continue
		}
		key := strings.ToLower(strings.Join(strings.Fields(item), " "))
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, item)
	}
	return items
}

// runSmartSuiteListingWriter creates marketplace copy from the same confirmed
// fact map used by the image-suite agents.
func runSmartSuiteListingWriter(ctx context.Context, task model.EcomSuiteTask, snapshot *model.EcomSuitePlanSnapshot) (model.EcomSuiteListing, error) {
	if snapshot == nil {
		return model.EcomSuiteListing{}, fmt.Errorf("listing_snapshot_missing")
	}
	apiKey := strings.TrimSpace(getSettingValue("chat_key"))
	modelName := getSmartSuiteAgentModel("listing_writer")
	if apiKey == "" || modelName == "" {
		return model.EcomSuiteListing{}, fmt.Errorf("listing_writer_model_not_configured")
	}
	keywordRecords := activeListingKeywordRecords(task.Platform, task.CountryCode, task.ProductCategory, 80)
	keywordTerms := make([]string, 0, len(keywordRecords))
	for _, keyword := range keywordRecords {
		keywordTerms = append(keywordTerms, keyword.Term)
	}

	requestJSON, _ := json.Marshal(map[string]interface{}{
		"platform":              normalizeListingPlatform(task.Platform),
		"output_structure":      smartSuiteListingOutputStructure(task.Platform),
		"marketplace":           strings.TrimSpace(task.CountryCode),
		"marketplace_name":      strings.TrimSpace(task.CountryName),
		"text_language":         strings.TrimSpace(task.Language),
		"product_name":          strings.TrimSpace(task.ProductName),
		"product_brand":         strings.TrimSpace(task.ProductBrand),
		"product_category":      strings.TrimSpace(task.ProductCategory),
		"confirmed_facts":       snapshot.ProductFactMap.ConfirmedFacts,
		"buyer_concerns":        snapshot.BuyerConcerns,
		"product_dimensions":    map[string]string{"length": task.ProductLength, "width": task.ProductWidth, "height": task.ProductHeight, "unit": task.ProductDimensionUnit},
		"product_weight":        map[string]string{"value": task.ProductWeight, "unit": task.ProductWeightUnit},
		"package_dimensions":    map[string]string{"length": task.PackageLength, "width": task.PackageWidth, "height": task.PackageHeight, "unit": task.PackageDimensionUnit},
		"package_weight":        map[string]string{"value": task.PackageWeight, "unit": task.PackageWeightUnit},
		"package_contents":      task.PackageContents,
		"keyword_library":       keywordTerms,
		"researched_keywords":   snapshot.ListingKeywords,
		"operator_keywords":     splitListingKeywords(task.ListingKeywords),
		"category_rule_profile": smartSuiteListingCategoryRuleProfile(task.Platform, task.ProductCategory),
	})
	listingAgent := smartSuiteListingAgent(task.Platform)
	system, err := smartSuiteAgentPrompt(listingAgent, smartSuiteListingRuntimeContract(task.Platform))
	if err != nil {
		return model.EcomSuiteListing{}, err
	}
	user := fmt.Sprintf("Write the synchronized product listing from this JSON input. First merge researched_keywords from AMZ123 with operator_keywords, then select only terms that are semantically supported by confirmed_facts. Use at least one supported researched or operator term naturally in the title when the platform title rules permit it, and use additional supported terms only where they improve readability; never force or stuff keywords. Use only confirmed_facts for product claims; buyer_concerns only controls ordering and buyer-oriented phrasing. Use the uploaded product image(s) as SKU identity evidence: inspect visible shape, finish, controls, components and included parts to distinguish this product from similar products, but do not infer hidden specifications or override confirmed textual facts. Keep each generated listing specific to the pictured SKU so two products with similar text do not receive identical copy. Never use web results to invent or modify product specifications, certifications, dimensions, warranty terms, or other unconfirmed facts. All listing copy must be natural localized US English for the selected retailer; never literal Chinese translation or mixed language.\n%s", string(requestJSON))
	visionImages := buildSmartSuiteVisionImages(task)
	callCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	startedAt := time.Now()
	listingAdapter := adapter.NewChatAdapter(apiKey, getSettingValue("access_url"))
	// Category research is completed by the dedicated keyword Agent. Writer
	// consumes those candidates and applies confirmed-fact filtering; it does
	// not perform a second platform search.
	listingOptions := adapter.VisionRequestOptions{}
	raw, usage, _, callErr := listingAdapter.AnalyzeImagesWithVisionContextOptions(callCtx, modelName, visionImages, system, user, 4500, listingOptions)
	resultErr := callErr
	var listing model.EcomSuiteListing
	if resultErr == nil {
		listing, resultErr = parseSmartSuiteListingResponse(raw)
		if resultErr == nil {
			listing.Platform = normalizeListingPlatform(task.Platform)
			listing = normalizeHomeDepotListingFields(listing, task.Platform)
			listing = constrainSmartSuiteListingToPlatform(listing, task.Platform)
			listing = removeHomeDepotListingTitleBrand(listing, task.ProductBrand)
			listing = enrichSmartSuiteListingEvidence(listing, snapshot.ProductFactMap.ConfirmedFacts)
			listing = enrichSmartSuiteListingReasons(listing, snapshot.ProductFactMap.ConfirmedFacts)
			resultErr = validateSmartSuiteListing(listing, snapshot.ProductFactMap.ConfirmedFacts)
			if resultErr == nil {
				resultErr = validateHomeDepotListingTitleBrand(listing, task.ProductBrand)
			}
		}
		for attempt := 0; resultErr != nil && callCtx.Err() == nil && attempt < 3; attempt++ {
			setSmartSuiteListingRepairAction(task.TaskID, task.Platform, attempt+1)
			repairedRaw, repairUsage, repairErr := repairSmartSuiteListingResponse(ctx, modelName, listingAgent, task.Platform, user, raw, resultErr, visionImages)
			if repairErr != nil {
				resultErr = fmt.Errorf("%w; repair_attempt_%d_failed: %v", resultErr, attempt+1, repairErr)
				continue
			}
			raw = repairedRaw
			if repairUsage != nil {
				usage = repairUsage
			}
			candidate, parseErr := parseSmartSuiteListingResponse(repairedRaw)
			if parseErr != nil {
				resultErr = parseErr
				continue
			}
			candidate.Platform = normalizeListingPlatform(task.Platform)
			candidate = normalizeHomeDepotListingFields(candidate, task.Platform)
			candidate = constrainSmartSuiteListingToPlatform(candidate, task.Platform)
			candidate = removeHomeDepotListingTitleBrand(candidate, task.ProductBrand)
			// A repair agent may return only the field it was asked to fix. Merge
			// it with the prior candidate instead of losing valid fields.
			listing = mergeSmartSuiteListingFields(listing, candidate)
			listing = enrichSmartSuiteListingEvidence(listing, snapshot.ProductFactMap.ConfirmedFacts)
			listing = enrichSmartSuiteListingReasons(listing, snapshot.ProductFactMap.ConfirmedFacts)
			resultErr = validateSmartSuiteListing(listing, snapshot.ProductFactMap.ConfirmedFacts)
			if resultErr == nil {
				resultErr = validateHomeDepotListingTitleBrand(listing, task.ProductBrand)
			}
		}
	}
	if normalizeListingPlatform(task.Platform) == "home_depot" {
		listing = synthesizeMissingHomeDepotFields(listing, task, snapshot.ProductFactMap.ConfirmedFacts)
		listing = removeHomeDepotListingTitleBrand(listing, task.ProductBrand)
		listing = enrichSmartSuiteListingEvidence(listing, snapshot.ProductFactMap.ConfirmedFacts)
		listing = enrichSmartSuiteListingReasons(listing, snapshot.ProductFactMap.ConfirmedFacts)
		resultErr = validateSmartSuiteListing(listing, snapshot.ProductFactMap.ConfirmedFacts)
		if resultErr == nil {
			resultErr = validateHomeDepotListingTitleBrand(listing, task.ProductBrand)
		}
	}
	status := "success"
	resultText := raw
	if resultErr != nil {
		status = "failed"
		resultText = resultErr.Error() + ": " + raw
	}
	logItem := model.UsageLog{UserID: uint(task.UserID), UsageType: "chat", Source: "smart_suite_listing_writer", Model: modelName, Prompt: adapter.TruncateRunes(user, 5000), Result: adapter.TruncateRunes(resultText, 1600), Duration: int(time.Since(startedAt).Milliseconds()), Status: status, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if usage != nil {
		logItem.PromptTokens, logItem.CompletionTokens, logItem.TotalTokens = usage.InputTokens, usage.OutputTokens, usage.TotalTokens
	}
	database.DB.Create(&logItem)
	if resultErr == nil {
		recordListingKeywordCandidatesWithMetadata(task.Platform, task.CountryCode, task.ProductCategory, "listing_agent", listing.KeywordCandidates)
		listingJSON, _ := json.Marshal(listing)
		markListingKeywordUsage(keywordRecords, string(listingJSON))
	}
	return listing, resultErr
}

func runSmartSuiteListingKeywordResearch(ctx context.Context, task model.EcomSuiteTask, snapshot *model.EcomSuitePlanSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("listing_keyword_snapshot_missing")
	}
	apiKey := strings.TrimSpace(getSettingValue("chat_key"))
	modelName := getSmartSuiteAgentModel("listing_writer")
	if apiKey == "" || modelName == "" {
		return fmt.Errorf("listing_keyword_model_not_configured")
	}
	input, _ := json.Marshal(map[string]interface{}{
		"platform": normalizeListingPlatform(task.Platform), "marketplace": task.CountryName,
		"product_category":  task.ProductCategory,
		"operator_keywords": splitListingKeywords(task.ListingKeywords),
	})
	searchURL, err := buildAMZ123KeywordSearchURL(task.ProductCategory)
	if err != nil {
		return err
	}
	pageKeywords, fetchErr := fetchAMZ123TopKeywords(ctx, searchURL)
	if fetchErr != nil {
		return fmt.Errorf("listing_keyword_amz123_fetch_failed: %w", fetchErr)
	}
	if len(pageKeywords) < 20 {
		return fmt.Errorf("listing_keyword_amz123_incomplete: expected 20 rows, got %d", len(pageKeywords))
	}
	system, err := smartSuiteAgentPrompt(smartagent.ResearchListingKeywords, fmt.Sprintf("Return one compact JSON object only, with no Markdown or prose. This is category research, not product validation. Use the official web search tool to search this exact AMZ123 category URL: %s . Extract every ranked keyword row from the returned category results, preserving rank order; if 20 rows are present, the keywords array MUST contain exactly 20 rows, not 5 representative examples. Use the supplied product category only as the query context; do not use or request the SKU's product name, product facts, image, specifications or certifications. Mark every result source as amz123_top_keywords. Combine operator keywords after the AMZ123 results as a separate source. Do not claim search volume unless AMZ123 explicitly displays it. For each candidate include only evidenced fields: term, source, tags, reason, evidence_urls. Keep reason short, evidence_urls to at most 2, and never summarize or truncate the ranked rows.", searchURL))
	if err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	user := fmt.Sprintf("Search AMZ123 with the category query and copy the first 20 ranked keyword rows exactly as returned. If the page shows 20 rows, return 20 objects. Do not return only a representative subset and do not infer this SKU's attributes. Search URL: %s\n%s", searchURL, input)
	startedAt := time.Now()
	raw, usage, _, callErr := adapter.NewChatAdapter(apiKey, getSettingValue("access_url")).AnalyzeImagesWithVisionContextOptions(callCtx, modelName, nil, system, user, 5000, amz123KeywordSearchOptions())
	logUsage := func(status, result string) {
		logItem := model.UsageLog{
			UserID: uint(task.UserID), UsageType: "chat", Source: "smart_suite_listing_keyword_research",
			Model: modelName, Prompt: adapter.TruncateRunes(user, 5000), Result: adapter.TruncateRunes(result, 2000),
			Duration: int(time.Since(startedAt).Milliseconds()), Status: status, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if usage != nil {
			logItem.PromptTokens = usage.InputTokens
			logItem.CompletionTokens = usage.OutputTokens
			logItem.TotalTokens = usage.TotalTokens
		}
		if database.DB != nil {
			database.DB.Create(&logItem)
		}
	}
	if callErr != nil {
		logUsage("failed", callErr.Error())
		return callErr
	}
	var result struct {
		Keywords []model.EcomSuiteKeywordCandidate `json:"keywords"`
	}
	if err := json.Unmarshal([]byte(extractJSONObjectFromText(stripSmartSuiteReasoningBlocks(raw))), &result); err != nil {
		logUsage("failed", "listing_keyword_invalid_json: "+err.Error()+": "+raw)
		return fmt.Errorf("listing_keyword_invalid_json: %w", err)
	}
	// Prefer the exact rows parsed from AMZ123. The model is retained only for
	// short reasons/tags and must never replace or truncate the fetched terms.
	modelByTerm := make(map[string]model.EcomSuiteKeywordCandidate, len(result.Keywords))
	for _, candidate := range result.Keywords {
		modelByTerm[strings.ToLower(strings.TrimSpace(candidate.Term))] = candidate
	}
	result.Keywords = make([]model.EcomSuiteKeywordCandidate, 0, 20)
	for _, term := range pageKeywords[:20] {
		candidate := modelByTerm[strings.ToLower(term)]
		candidate.Term = term
		candidate.Source = "amz123_top_keywords"
		candidate.EvidenceURLs = []string{searchURL}
		result.Keywords = append(result.Keywords, candidate)
	}
	if len(result.Keywords) < 20 {
		logUsage("failed", fmt.Sprintf("listing_keyword_incomplete: expected 20 AMZ123 rows, got %d", len(result.Keywords)))
		return fmt.Errorf("listing_keyword_incomplete: expected 20 AMZ123 rows, got %d", len(result.Keywords))
	}
	snapshot.ListingKeywords = mergeResearchedAndOperatorKeywords(result.Keywords, splitListingKeywords(task.ListingKeywords), 25)
	snapshot.ListingKeywordsReady = true
	resultJSON, _ := json.Marshal(snapshot.ListingKeywords)
	logUsage("success", string(resultJSON))
	recordListingKeywordCandidatesWithMetadata(task.Platform, task.CountryCode, task.ProductCategory, "keyword_research_agent", snapshot.ListingKeywords)
	return nil
}

func fetchAMZ123TopKeywords(ctx context.Context, searchURL string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ListingKeywordResearch/1.0)")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http_status_%d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	pattern := regexp.MustCompile(`(?is)<a[^>]*class="[^"]*table-body-item-words-word[^"]*"[^>]*>`)
	titlePattern := regexp.MustCompile(`(?i)title="([^"]+)"`)
	matches := pattern.FindAll(body, -1)
	terms := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		titleMatch := titlePattern.FindSubmatch(match)
		if len(titleMatch) < 2 {
			continue
		}
		term := strings.TrimSpace(html.UnescapeString(string(titleMatch[1])))
		key := strings.ToLower(term)
		if term == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		terms = append(terms, term)
	}
	return terms, nil
}

func buildAMZ123KeywordSearchURL(category string) (string, error) {
	category = strings.TrimSpace(category)
	if category == "" {
		return "", fmt.Errorf("listing_keyword_category_missing")
	}
	return "https://www.amz123.com/usatopkeywords/1?k=" + url.QueryEscape(category), nil
}

func normalizeAMZ123KeywordCandidates(values []model.EcomSuiteKeywordCandidate, searchURL string) []model.EcomSuiteKeywordCandidate {
	result := make([]model.EcomSuiteKeywordCandidate, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value.Term = strings.TrimSpace(value.Term)
		key := strings.ToLower(value.Term)
		if value.Term == "" || key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		value.Source = "amz123_top_keywords"
		// The model may cite unrelated AMZ123 pages. Only the exact queried
		// category URL is a valid evidence page for this research run.
		validURLs := make([]string, 0, 1)
		for _, evidenceURL := range value.EvidenceURLs {
			if strings.TrimSpace(evidenceURL) == searchURL {
				validURLs = append(validURLs, searchURL)
			}
		}
		if len(validURLs) == 0 {
			// Every accepted candidate is explicitly attributed to the exact
			// AMZ123 query used by this run; never expose a different page URL.
			validURLs = append(validURLs, searchURL)
		}
		value.EvidenceURLs = validURLs
		result = append(result, value)
		if len(result) >= 20 {
			break
		}
	}
	return result
}

func mergeResearchedAndOperatorKeywords(researched []model.EcomSuiteKeywordCandidate, operator []string, limit int) []model.EcomSuiteKeywordCandidate {
	merged := make([]model.EcomSuiteKeywordCandidate, 0, len(researched)+len(operator))
	seen := make(map[string]bool)
	appendCandidate := func(candidate model.EcomSuiteKeywordCandidate) bool {
		candidate.Term = strings.TrimSpace(candidate.Term)
		key := strings.ToLower(candidate.Term)
		if candidate.Term == "" || seen[key] || (limit > 0 && len(merged) >= limit) {
			return false
		}
		seen[key] = true
		merged = append(merged, candidate)
		return true
	}
	for _, candidate := range researched {
		if strings.TrimSpace(candidate.Source) == "" {
			candidate.Source = "marketplace"
		}
		appendCandidate(candidate)
	}
	for _, term := range operator {
		appendCandidate(model.EcomSuiteKeywordCandidate{
			Term: term, Source: "operator", Tags: []string{"operator"},
			Reason: "运营提供的关键词，交由 Writer 在产品事实支持时自然使用",
		})
	}
	return merged
}

func rankListingKeywordCandidates(values []model.EcomSuiteKeywordCandidate) []model.EcomSuiteKeywordCandidate {
	for index := range values {
		value := &values[index]
		if value.SearchVolume > 0 {
			volumeScore := 0
			switch {
			case value.SearchVolume >= 10000:
				volumeScore = 60
			case value.SearchVolume >= 3000:
				volumeScore = 45
			case value.SearchVolume >= 1000:
				volumeScore = 30
			default:
				volumeScore = 15
			}
			competitionScore := int((1 - minListingKeywordFloat(value.Competition, 1)) * 20)
			value.SEOScore = volumeScore + competitionScore + minListingKeywordInt(value.PlatformFrequency, 10)*2
		} else {
			value.SEOScore = minListingKeywordInt(value.PlatformFrequency, 10) * 4
			if value.Source == "operator" {
				value.SEOScore += 10
			}
		}
	}
	sort.SliceStable(values, func(i, j int) bool { return values[i].SEOScore > values[j].SEOScore })
	return values
}

func minListingKeywordInt(value, max int) int {
	if value < 0 {
		return 0
	}
	if value > max {
		return max
	}
	return value
}

func minListingKeywordFloat(value, max float64) float64 {
	if value < 0 {
		return 0
	}
	if value > max {
		return max
	}
	return value
}

func repairSmartSuiteListingResponse(ctx context.Context, modelName string, listingAgent smartagent.Name, platform string, user, raw string, validationErr error, visionImages []adapter.VisionImageItem) (string, *adapter.ResponsesUsage, error) {
	repairInstruction := "Repair the previous platform-specific listing response. Return only the listing JSON shape. Remove unsupported claims instead of guessing. The title is a listing title, never an image headline. Content fields are mandatory; do not return a partial listing. Preserve or add an evidence entry and an optimization_reasons entry for every populated listing field, using only real confirmed fact IDs from the original request; never invent fact IDs or quote text. Keep all copy in natural localized US English for the selected retailer, never literal translation or mixed language. Use feature-plus-benefit wording and fact-supported use scenarios where the platform field allows it. Never add unsupported quantitative claims, material grades, hose/accessory details, lead-free claims, durability guarantees, or certifications."
	if listingAgent == smartagent.WriteHomeDepotListing {
		repairInstruction += " For Home Depot, return all four required fields directly in one object: title, salient_bullets (exactly 3), bullet_points (exactly 6), and about_this_product. Build the title generically for the selected category and naturally include the highest-frequency researched category search term that is supported by confirmed facts, followed by product type and confirmed differentiating specifications; never force faucet-specific fields onto another category. Salient bullets must be distinct Feature + direct shopper benefit statements under 65 characters, telling the buyer what they gain. Bullet Points must each combine feature benefit + supported use condition, and include confirmed dimensions/specifications and compliance details where relevant. Listing copy must be natural US English with familiar US retail and installation terminology, never mixed-language or literal Chinese translation. About This Product must be 3-6 clear sentences under 1500 characters. Each optimization_reasons item must include detailed Simplified Chinese platform_basis, writing_decision, expected_impact, rule, and cited fact_ids. Do not use or map bullets, description, Highlights, Product Overview, Specifications, or Documents."
	}
	if listingAgent == smartagent.WriteLowesListing {
		repairInstruction += " For Lowe's, return only title, description, and bullets. Re-identify the applicable category slots from confirmed facts before repairing. The title slot order is mandatory: Brand + Product Type + Material/Finish + Handle Configuration + Mounting/Function + Sprayer + Included Accessory. Retain every supported slot in that exact left-to-right order; omit an unsupported slot but never move a later slot before an earlier one. For a kitchen faucet, Kitchen Faucet or Kitchen Sink Faucet must precede material/finish; finish must precede Single Handle; Single Handle must precede Pull-Down or mounting/function; Sprayer follows function; Deck Plate or another included accessory is last. The kitchen-faucet examples Pull-Down, Sprayer, Single Handle, Deck Plate, GPM, cartridge and spray modes are required or scored only when confirmed for this SKU; never add them to another product or penalize their absence. Keep title within 253 characters and remove forbidden symbols. Make description one complementary sentence organized as function, multiple supported scenarios, then overall experience, within 2000 characters. Make every bullet one complete feature + applicable scenario + benefit sentence within 254 characters; prefer 100-180 characters without padding. Remove unsupported claims, repeated wording, ALL CAPS, Better than, Compared with and Guaranteed. Preserve visible image-based context only when conservative and consistent with textual facts."
		repairInstruction += " Rewrite every Lowe's bullet with its confirmed searchable feature or specification noun phrase first, followed by scenario, benefit, and evidence-based reassurance. Never begin a bullet with The, This, These, It, They, Provides, Designed, Features, Offers, Helps, Allows, Supports, Includes, Comes, or Made."
	}
	if listingAgent == smartagent.WriteWayfairListing {
		repairInstruction += " For Wayfair, return only about_this_product and features; do not output title. The JSON must contain a literal `features` array of non-empty English strings inside `listing`, never feature_bullets, a prose Feature section, or a single combined string. Do not apply character, word-count, sentence-count, feature-count, phrase-only, or punctuation limits. Colors and finishes are allowed when supported by this SKU. Follow the Wayfair skill for the exact positioning-specific Feature order and low-priority fact merging. About is a coherent product introduction; consistent feature headings are allowed. Each core Feature should explain a supported use or purchase value; do not leave a bare component action such as change the water direction, operate with one lever, or mounts on 1- or 3-hole decks. Never expose evidence-source wording such as confirmed, specified as, listed as, according to the product information, or the product is identified as. About may preview a theme that Features expands with evidence; remove only repetition with no added information. Keep a confirmed high-arc, material/lead-free, care, or warranty fact when it answers a distinct space, trust, or coverage concern, but write the consumer-facing fact directly. Rewrite keyword lists and database-field phrasing as plainspoken US retail copy; use direct verbs such as adjust, direct, fill, rinse, cover, and leave room instead of mechanical phrases such as brings a sprayer to the sink, by moving the spout, or at a single control point. Combine tools, isolated dimensions, certifications, and warranty with the relevant installation or assurance item instead of making them separate headline Features."
	}
	repairSystem := smartSuiteAgentPromptText(listingAgent, repairInstruction)
	repairUser := fmt.Sprintf("Validation error: %s\nOriginal request:\n%s\nPrevious response:\n%s\nReturn the missing or invalid required fields explicitly; do not return title alone. Keep valid fields unchanged or repeat them. Keep only claims supported by confirmed_facts, cite each populated field in evidence, and explain why each field is structured this way in optimization_reasons.", validationErr, adapter.TruncateRunes(user, 9000), adapter.TruncateRunes(raw, 10000))
	repairCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	// Research is complete before Writer runs. Repair only needs the supplied
	// draft, confirmed facts and review issues, so hosted tools stay disabled.
	returnText, usage, _, err := adapter.NewChatAdapter(strings.TrimSpace(getSettingValue("chat_key")), getSettingValue("access_url")).AnalyzeImagesWithVisionContextOptions(repairCtx, modelName, visionImages, repairSystem, repairUser, 4500, adapter.VisionRequestOptions{})
	return returnText, usage, err
}

func amz123KeywordSearchOptions() adapter.VisionRequestOptions {
	return adapter.VisionRequestOptions{WebSearch: true, AllowedDomains: []string{"amz123.com", "www.amz123.com"}, SearchContextSize: "high"}
}

func listingReviewAgent(platform string) smartagent.Name {
	switch normalizeListingPlatform(platform) {
	case "home_depot":
		return smartagent.ReviewHomeDepotListing
	case "lowes":
		return smartagent.ReviewLowesListing
	case "wayfair":
		return smartagent.ReviewWayfairListing
	default:
		return ""
	}
}

func buildSmartSuiteListingReviewInput(task model.EcomSuiteTask, snapshot *model.EcomSuitePlanSnapshot) ([]byte, string, error) {
	if snapshot == nil || snapshot.Listing == nil {
		return nil, "", fmt.Errorf("listing_review_listing_missing")
	}
	listingJSON, err := json.Marshal(snapshot.Listing)
	if err != nil {
		return nil, "", fmt.Errorf("listing_review_listing_encode_failed: %w", err)
	}
	fingerprint := fmt.Sprintf("%x", sha256.Sum256(listingJSON))
	input, err := json.Marshal(map[string]interface{}{
		"platform":              normalizeListingPlatform(task.Platform),
		"product_name":          strings.TrimSpace(task.ProductName),
		"product_category":      strings.TrimSpace(task.ProductCategory),
		"confirmed_facts":       snapshot.ProductFactMap.ConfirmedFacts,
		"category_rule_profile": smartSuiteListingCategoryRuleProfile(task.Platform, task.ProductCategory),
		"listing_fingerprint":   fingerprint,
		"writer_output_listing": snapshot.Listing,
	})
	if err != nil {
		return nil, "", fmt.Errorf("listing_review_input_encode_failed: %w", err)
	}
	return input, fingerprint, nil
}

func runSmartSuiteListingReview(ctx context.Context, task model.EcomSuiteTask, snapshot *model.EcomSuitePlanSnapshot) error {
	if snapshot == nil || snapshot.Listing == nil {
		return fmt.Errorf("listing_review_listing_missing")
	}
	agent := listingReviewAgent(task.Platform)
	if agent == "" {
		return fmt.Errorf("listing_review_platform_unsupported")
	}
	modelName := getSmartSuiteAgentModel("listing_reviewer")
	if strings.TrimSpace(modelName) == "" {
		modelName = getSmartSuiteAgentModel("listing_writer")
	}
	if strings.TrimSpace(modelName) == "" {
		return fmt.Errorf("listing_review_model_not_configured")
	}
	reviewInputJSON, listingFingerprint, inputErr := buildSmartSuiteListingReviewInput(task, snapshot)
	if inputErr != nil {
		return inputErr
	}
	listingJSON, _ := json.Marshal(snapshot.Listing)
	reviewContract := "Return JSON only: {\"input_fingerprint\":\"exact supplied listing_fingerprint\",\"reviewed_title\":\"exact title from writer_output_listing\",\"passed\":true|false,\"issues\":[\"...\"],\"checks\":[\"...\"],\"summary\":\"...\"}. Audit writer_output_listing exactly as supplied. Copy listing_fingerprint unchanged into input_fingerprint and copy the exact writer_output_listing.title into reviewed_title; both fields are mandatory proof that the supplied Writer draft was reviewed. Check every populated field for confirmed-fact evidence, platform structure, natural localized US English, unsupported claims, and complete detailed Chinese optimization reasons. Also enforce the operator quality standard: title must cover the product type and the highest-value confirmed search terms; salient/features/bullets must pair each feature with a concrete shopper benefit or supported use scenario; fields must have distinct responsibilities and avoid repeated copy; descriptions must read as natural retail copy rather than a bare specification list. Reject vague phrases, unexplained negative attributes, and unsupported quantitative, material, accessory, certification, or durability claims. Do not rewrite the listing."
	if normalizeListingPlatform(task.Platform) == "lowes" {
		reviewContract += " For Lowe's, apply the current 20/20/60 Title, Description, Bullets standard. Title: at most 253 characters, allowed basic punctuation, complete category product term and every high-value function/material/finish/accessory that is confirmed and applicable. Description: one sentence, at most 2000 characters, function -> multiple supported use scenarios -> overall experience, complementary rather than duplicating bullets, with 3-5 natural SEO phrases when supported. Bullets: each is one complete sentence, at most 254 characters, feature + applicable scenario + benefit; prefer 100-180 characters and do not pad. Dynamically select material, functions, operation/range, installation/compatibility/accessories, performance, controls/reliability, finish/style/care, and warranty/support. Faucet terms are examples only: never require or fail Pull-Down, Sprayer, Single Handle, Deck Plate, GPM, cartridge or spray modes unless confirmed for the SKU."
	}
	if normalizeListingPlatform(task.Platform) == "wayfair" {
		reviewContract += " Wayfair has no title: return reviewed_title as an empty string and do not require or audit a product title."
		reviewContract += " For Wayfair, do not enforce character, word-count, sentence-count, feature-count, phrase-only, or punctuation limits. Only review About This Product and Features. Supported colors and finishes are allowed. Reject meta or internal wording such as fit information, purchase decision, helps shoppers decide, or gives buyers guidance, as well as vague phrases such as switch tasks or useful working room. Follow the Wayfair writing skill for style and buyer-specific priority. About introduces the product; Features may use consistent headings and expand previewed themes with new evidence. Do not require a kitchen story or operation-first ordering. Reject a core Feature that only describes component movement or a bare specification without its confirmed shopper result."
	}
	system, err := smartSuiteAgentPrompt(agent, reviewContract)
	if err != nil {
		return err
	}
	user := fmt.Sprintf("Review this %s listing independently. The uploaded product image is supporting evidence only within the stated visual limits. Input JSON:\n%s", normalizeListingPlatform(task.Platform), string(reviewInputJSON))
	visionImages := buildSmartSuiteVisionImages(task)
	reviewAdapter := adapter.NewChatAdapter(strings.TrimSpace(getSettingValue("chat_key")), getSettingValue("access_url"))
	// The reviewer audits a concrete Writer draft against rules and confirmed
	// facts. Web search belongs to keyword research/Writer; enabling it here can
	// yield only a hosted-tool event and no verdict, blocking automatic repair.
	callCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	startedAt := time.Now()
	raw, usage, _, callErr := reviewAdapter.AnalyzeImagesWithVisionContextOptions(callCtx, modelName, visionImages, system, user, 2200, adapter.VisionRequestOptions{})
	cancel()
	reviewLogStatus := "success"
	reviewLogResult := raw
	if callErr != nil {
		reviewLogStatus = "failed"
		reviewLogResult = callErr.Error() + ": " + raw
	}
	reviewLog := model.UsageLog{UserID: uint(task.UserID), UsageType: "chat", Source: "smart_suite_listing_reviewer", Model: modelName, Prompt: adapter.TruncateRunes(user, 9000), Result: adapter.TruncateRunes(reviewLogResult, 2200), Duration: int(time.Since(startedAt).Milliseconds()), Status: reviewLogStatus, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if usage != nil {
		reviewLog.PromptTokens, reviewLog.CompletionTokens, reviewLog.TotalTokens = usage.InputTokens, usage.OutputTokens, usage.TotalTokens
	}
	database.DB.Create(&reviewLog)
	if callErr != nil {
		return callErr
	}
	var review model.EcomSuiteListingReview
	if err := json.Unmarshal([]byte(extractJSONObjectFromText(stripSmartSuiteReasoningBlocks(raw))), &review); err != nil {
		return fmt.Errorf("listing_review_invalid_json: %w", err)
	}
	review.Platform = normalizeListingPlatform(task.Platform)
	// A malformed audit is a quality failure, not a reason to loop through the
	// Reviewer again. Turn it into explicit repair issues so the Writer receives
	// the real draft and must return a corrected listing.
	if review.InputFingerprint != listingFingerprint {
		review.Passed = false
		review.Issues = append(review.Issues, "reviewer did not confirm the supplied listing fingerprint")
	}
	if review.ReviewedTitle != snapshot.Listing.Title {
		review.Passed = false
		review.Issues = append(review.Issues, "reviewer did not confirm the exact Writer output title")
	}
	if review.Passed && (strings.TrimSpace(review.Summary) == "" || len(review.Checks) == 0) {
		return fmt.Errorf("listing_review_verdict_missing_audit_details")
	}
	if !review.Passed {
		if len(review.Issues) == 0 {
			review.Issues = []string{"independent review did not pass"}
		}
		// Review is an automatic correction gate: repair the generated JSON and
		// validate the corrected listing before allowing the workflow to finish.
		validationErr := fmt.Errorf("listing_review_failed: %s", strings.Join(review.Issues, "; "))
		writerAgent := smartSuiteListingAgent(task.Platform)
		writerModelName := getSmartSuiteAgentModel("listing_writer")
		if strings.TrimSpace(writerModelName) == "" {
			writerModelName = modelName
		}
		currentRaw := string(listingJSON)
		for attempt := 0; attempt < 2; attempt++ {
			setSmartSuiteListingRepairAction(task.TaskID, task.Platform, attempt+1)
			repairedRaw, _, repairErr := repairSmartSuiteListingResponse(ctx, writerModelName, writerAgent, task.Platform, user, currentRaw, validationErr, visionImages)
			if repairErr != nil {
				validationErr = fmt.Errorf("%w; auto_repair_attempt_%d_failed: %v", validationErr, attempt+1, repairErr)
				continue
			}
			candidate, parseErr := parseSmartSuiteListingResponse(repairedRaw)
			if parseErr != nil {
				validationErr = fmt.Errorf("%w; auto_repair_attempt_%d_invalid_json: %v", validationErr, attempt+1, parseErr)
				currentRaw = repairedRaw
				continue
			}
			candidate.Platform = normalizeListingPlatform(task.Platform)
			candidate = normalizeHomeDepotListingFields(candidate, task.Platform)
			candidate = constrainSmartSuiteListingToPlatform(candidate, task.Platform)
			candidate = removeHomeDepotListingTitleBrand(candidate, task.ProductBrand)
			candidate = enrichSmartSuiteListingEvidence(candidate, snapshot.ProductFactMap.ConfirmedFacts)
			candidate = enrichSmartSuiteListingReasons(candidate, snapshot.ProductFactMap.ConfirmedFacts)
			validationErr = validateSmartSuiteListing(candidate, snapshot.ProductFactMap.ConfirmedFacts)
			if validationErr == nil {
				if titleErr := validateHomeDepotListingTitleBrand(candidate, task.ProductBrand); titleErr != nil {
					validationErr = titleErr
				} else {
					snapshot.Listing = &candidate
					// Programmatic validation proves only structure and fact references.
					// The revised copy must go through a fresh independent model review
					// before it can be published.
					snapshot.ListingReview = nil
					snapshot.ListingReviewError = ""
					return errSmartSuiteListingReviewRecheckRequired
				}
			}
			currentRaw = repairedRaw
		}
		return validationErr
	}
	snapshot.ListingReview = &review
	snapshot.ListingReviewError = ""
	return nil
}

func mergeSmartSuiteListingFields(base, candidate model.EcomSuiteListing) model.EcomSuiteListing {
	if strings.TrimSpace(candidate.Title) != "" {
		base.Title = candidate.Title
	}
	if len(candidate.SalientBullets) > 0 {
		base.SalientBullets = candidate.SalientBullets
	}
	if len(candidate.BulletPoints) > 0 {
		base.BulletPoints = candidate.BulletPoints
	}
	if strings.TrimSpace(candidate.AboutThisProduct) != "" {
		base.AboutThisProduct = candidate.AboutThisProduct
	}
	if len(candidate.Features) > 0 {
		base.Features = candidate.Features
	}
	if len(candidate.Evidence) > 0 {
		base.Evidence = append(base.Evidence, candidate.Evidence...)
	}
	if len(candidate.OptimizationReasons) > 0 {
		base.OptimizationReasons = append(base.OptimizationReasons, candidate.OptimizationReasons...)
	}
	if len(candidate.KeywordCandidates) > 0 {
		base.KeywordCandidates = candidate.KeywordCandidates
	}
	return base
}

func normalizeHomeDepotListingFields(listing model.EcomSuiteListing, platform string) model.EcomSuiteListing {
	if normalizeListingPlatform(platform) != "home_depot" {
		return listing
	}
	if len(listing.SalientBullets) == 0 && len(listing.Highlights) > 0 {
		listing.SalientBullets = append([]string(nil), listing.Highlights...)
	}
	if len(listing.SalientBullets) > 3 {
		listing.SalientBullets = listing.SalientBullets[:3]
	}
	if len(listing.BulletPoints) == 0 && len(listing.Bullets) > 0 {
		listing.BulletPoints = append([]string(nil), listing.Bullets...)
	}
	if len(listing.BulletPoints) > 6 {
		listing.BulletPoints = listing.BulletPoints[:6]
	}
	if strings.TrimSpace(listing.AboutThisProduct) == "" && strings.TrimSpace(listing.Description) != "" {
		listing.AboutThisProduct = listing.Description
	}
	return listing
}

func synthesizeMissingHomeDepotFields(listing model.EcomSuiteListing, task model.EcomSuiteTask, facts []model.EcomSuiteProductFact) model.EcomSuiteListing {
	if len(facts) == 0 {
		return listing
	}
	factText := make([]string, 0, len(facts))
	for _, fact := range facts {
		value := strings.TrimSpace(fact.Statement)
		if value == "" {
			value = strings.TrimSpace(fact.Evidence)
		}
		if value != "" {
			factText = append(factText, strings.TrimSpace(value))
		}
	}
	if strings.TrimSpace(listing.Title) == "" {
		listing.Title = strings.TrimSpace(task.ProductName)
		if listing.Title == "" {
			listing.Title = strings.TrimSpace(task.ProductCategory)
		}
	}
	if len(listing.SalientBullets) < 3 {
		for _, value := range factText {
			if len(listing.SalientBullets) >= 3 {
				break
			}
			value = truncateListingValue(value, 65)
			if value != "" && !containsListingValue(listing.SalientBullets, value) {
				listing.SalientBullets = append(listing.SalientBullets, value)
			}
		}
		for len(listing.SalientBullets) < 3 && len(factText) > 0 {
			index := len(listing.SalientBullets)
			value := truncateListingValue(fmt.Sprintf("Key feature %d: %s", index+1, factText[index%len(factText)]), 65)
			if !containsListingValue(listing.SalientBullets, value) {
				listing.SalientBullets = append(listing.SalientBullets, value)
			}
		}
	}
	if len(listing.BulletPoints) < 6 {
		for _, value := range factText {
			if len(listing.BulletPoints) >= 6 {
				break
			}
			value = strings.TrimRight(strings.TrimSpace(value), ".。")
			if value != "" && !containsListingValue(listing.BulletPoints, value) {
				listing.BulletPoints = append(listing.BulletPoints, value)
			}
		}
		labels := []string{"Function", "Appearance", "Installation", "Use", "Compatibility", "Care"}
		for len(listing.BulletPoints) < 6 && len(factText) > 0 {
			index := len(listing.BulletPoints)
			value := strings.TrimRight(strings.TrimSpace(fmt.Sprintf("%s: %s", labels[index], factText[index%len(factText)])), ".。")
			if !containsListingValue(listing.BulletPoints, value) {
				listing.BulletPoints = append(listing.BulletPoints, value)
			}
		}
	}
	if strings.TrimSpace(listing.AboutThisProduct) == "" && len(factText) > 0 {
		parts := make([]string, 0, 3)
		for _, value := range factText {
			if len(parts) >= 3 {
				break
			}
			value = strings.TrimSpace(strings.TrimRight(value, ".。"))
			if value != "" {
				parts = append(parts, value+".")
			}
		}
		if len(parts) > 0 {
			listing.AboutThisProduct = strings.Join(parts, " ")
		}
	}
	return listing
}

func containsListingValue(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == target {
			return true
		}
	}
	return false
}

func truncateListingValue(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return strings.TrimSpace(string(runes))
}

func parseSmartSuiteListingResponse(text string) (model.EcomSuiteListing, error) {
	text = extractJSONObjectFromText(stripSmartSuiteReasoningBlocks(text))
	if text == "" {
		return model.EcomSuiteListing{}, fmt.Errorf("listing_invalid_json")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &fields); err != nil {
		return model.EcomSuiteListing{}, fmt.Errorf("listing_invalid_json: %w", err)
	}
	listingRaw, ok := smartSuiteFindJSONField(fields, "listing", "result", "output", "data", "content", "optimized_listing", "optimizedListing")
	if !ok {
		listingRaw = json.RawMessage(text)
	}
	var listingFields map[string]json.RawMessage
	if err := json.Unmarshal(listingRaw, &listingFields); err != nil {
		return model.EcomSuiteListing{}, fmt.Errorf("listing_missing_object: %w", err)
	}
	if nested, ok := smartSuiteFindJSONField(listingFields, "listing"); ok {
		containerFields := listingFields
		if err := json.Unmarshal(nested, &listingFields); err != nil {
			return model.EcomSuiteListing{}, fmt.Errorf("listing_missing_object: %w", err)
		}
		for key, value := range containerFields {
			canonical := smartSuiteCanonicalJSONKey(key)
			if canonical == "listing" {
				continue
			}
			present := false
			for innerKey := range listingFields {
				if smartSuiteCanonicalJSONKey(innerKey) == canonical {
					present = true
					break
				}
			}
			if !present {
				listingFields[key] = value
			}
		}
	}
	// Some model providers keep title inside `listing` but place the remaining
	// platform fields beside it. Merge only missing fields so the persisted
	// object still uses the platform's direct field names.
	for outerKey, outerValue := range fields {
		canonicalOuterKey := smartSuiteCanonicalJSONKey(outerKey)
		if canonicalOuterKey == "listing" || canonicalOuterKey == "result" || canonicalOuterKey == "output" || canonicalOuterKey == "data" || canonicalOuterKey == "content" || canonicalOuterKey == "optimizedlisting" {
			continue
		}
		found := false
		for innerKey := range listingFields {
			if smartSuiteCanonicalJSONKey(innerKey) == canonicalOuterKey {
				found = true
				break
			}
		}
		if !found {
			listingFields[outerKey] = outerValue
		}
	}
	listing := model.EcomSuiteListing{}
	if raw, ok := smartSuiteFindJSONField(listingFields, "platform", "marketplace"); ok {
		listing.Platform = smartSuiteJSONText(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "title"); ok {
		listing.Title = smartSuiteJSONText(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "description"); ok {
		listing.Description = smartSuiteJSONText(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "bullets", "key_features"); ok {
		listing.Bullets = smartSuiteListingStringArray(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "bullet_points"); ok {
		listing.BulletPoints = smartSuiteListingStringArray(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "about_this_product"); ok {
		listing.AboutThisProduct = smartSuiteJSONText(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "salient_bullets", "salient_points", "top_bullets"); ok {
		listing.SalientBullets = smartSuiteListingStringArray(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "highlights", "product_highlights"); ok {
		listing.Highlights = smartSuiteListingStringArray(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "supplier_bullets", "supplier_bullet", "lowes_bullets"); ok {
		listing.SupplierBullets = smartSuiteListingStringArray(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "marketing_copy"); ok {
		listing.MarketingCopy = smartSuiteJSONText(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "lowes_product_description"); ok {
		listing.LowesProductDescription = smartSuiteJSONText(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "features", "feature_bullets", "feature_bullet", "product_features"); ok {
		listing.Features = smartSuiteListingStringArray(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "variants", "variant_options", "options"); ok {
		listing.Variants = smartSuiteListingStringArray(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "whats_included", "what_is_included", "included"); ok {
		listing.WhatsIncluded = smartSuiteListingStringArray(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "not_included", "excluded"); ok {
		listing.NotIncluded = smartSuiteListingStringArray(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "documents", "certifications", "certification", "support_documents"); ok {
		listing.Documents = smartSuiteListingStringArray(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "attributes", "specifications", "specs"); ok {
		_ = json.Unmarshal(raw, &listing.Attributes)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "details", "dimensions", "wayfair_details"); ok {
		_ = json.Unmarshal(raw, &listing.Details)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "search_terms", "backend_search_terms", "keywords"); ok {
		listing.SearchTerms = smartSuiteListingStringArray(raw)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "evidence", "fact_evidence", "field_evidence"); ok {
		_ = json.Unmarshal(raw, &listing.Evidence)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "optimization_reasons", "generation_reasons", "reasons"); ok {
		_ = json.Unmarshal(raw, &listing.OptimizationReasons)
	}
	if raw, ok := smartSuiteFindJSONField(listingFields, "keyword_candidates"); ok {
		_ = json.Unmarshal(raw, &listing.KeywordCandidates)
	}
	return listing, nil
}

func smartSuiteListingStringArray(raw json.RawMessage) []string {
	var values []string
	if json.Unmarshal(raw, &values) == nil {
		return trimSmartSuiteListingStrings(values)
	}
	value := smartSuiteJSONText(raw)
	if value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';' || r == '，' || r == '；'
	})
	return trimSmartSuiteListingStrings(parts)
}

func trimSmartSuiteListingStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func validateSmartSuiteListing(listing model.EcomSuiteListing, confirmedFacts ...[]model.EcomSuiteProductFact) error {
	platform := normalizeListingPlatform(listing.Platform)
	if platform != "lowes" && platform != "wayfair" && strings.TrimSpace(listing.Title) == "" {
		return fmt.Errorf("listing_title_missing")
	}
	if platform == "" || platform == "amazon" {
		if len(listing.Bullets) != 5 {
			return fmt.Errorf("listing_invalid_bullets: expected 5 got %d", len(listing.Bullets))
		}
	}
	allBullets := append(append([]string{}, listing.Bullets...), listing.BulletPoints...)
	for index, bullet := range allBullets {
		if strings.TrimSpace(bullet) == "" {
			return fmt.Errorf("listing_bullet_%d_missing", index+1)
		}
	}
	if strings.TrimSpace(listing.Description) == "" && strings.TrimSpace(listing.AboutThisProduct) == "" && strings.TrimSpace(listing.LowesProductDescription) == "" && len(listing.Highlights) == 0 {
		return fmt.Errorf("listing_description_missing")
	}
	if platform == "lowes" && len(listing.SupplierBullets) == 0 && len(listing.Bullets) == 0 {
		return fmt.Errorf("listing_supplier_bullets_missing")
	}
	seen := map[string]bool{}
	for _, bullet := range append(append(allBullets, listing.SupplierBullets...), listing.SalientBullets...) {
		key := strings.ToLower(strings.TrimSpace(bullet))
		if key != "" && seen[key] {
			return fmt.Errorf("listing_duplicate_bullet")
		}
		seen[key] = true
	}
	if len(listing.SearchTerms) > 10 {
		return fmt.Errorf("listing_too_many_search_terms: %d", len(listing.SearchTerms))
	}
	if err := validateSmartSuiteListingPlatformStructure(listing); err != nil {
		return err
	}
	if len(confirmedFacts) > 0 {
		if err := validateSmartSuiteListingEvidence(listing, confirmedFacts[0]); err != nil {
			return err
		}
		if err := validateSmartSuiteListingReasons(listing, confirmedFacts[0]); err != nil {
			return err
		}
	}
	return nil
}

func validateSmartSuiteListingPlatformStructure(listing model.EcomSuiteListing) error {
	switch normalizeListingPlatform(listing.Platform) {
	case "home_depot":
		for _, value := range append(append([]string{listing.Title, listing.AboutThisProduct}, listing.SalientBullets...), listing.BulletPoints...) {
			if containsHanText(value) {
				return fmt.Errorf("listing_home_depot_copy_not_us_english")
			}
		}
		if strings.TrimSpace(listing.Title) == "" {
			return fmt.Errorf("listing_home_depot_title_missing")
		}
		if strings.TrimSpace(listing.AboutThisProduct) == "" {
			return fmt.Errorf("listing_home_depot_product_overview_missing")
		}
		if len(listing.BulletPoints) == 0 {
			return fmt.Errorf("listing_home_depot_bullet_points_missing")
		}
		if len(listing.SalientBullets) != 3 {
			return fmt.Errorf("listing_home_depot_salient_bullets_invalid")
		}
		if len([]rune(strings.TrimSpace(listing.Title))) > 120 {
			return fmt.Errorf("listing_home_depot_title_too_long")
		}
		if err := validateHomeDepotTitleOrder(listing.Title); err != nil {
			return err
		}
		for _, bullet := range listing.SalientBullets {
			if strings.TrimSpace(bullet) == "" || len([]rune(strings.TrimSpace(bullet))) > 65 {
				return fmt.Errorf("listing_home_depot_salient_bullets_invalid")
			}
		}
		if len(listing.BulletPoints) != 6 {
			return fmt.Errorf("listing_home_depot_bullet_points_invalid")
		}
		for _, bullet := range listing.BulletPoints {
			trimmed := strings.TrimSpace(bullet)
			if trimmed == "" || strings.HasSuffix(trimmed, ".") || strings.HasSuffix(trimmed, "。") {
				return fmt.Errorf("listing_home_depot_bullet_points_invalid")
			}
		}
		if len([]rune(strings.TrimSpace(listing.AboutThisProduct))) > 1500 {
			return fmt.Errorf("listing_home_depot_about_this_product_too_long")
		}
	case "tiktok":
		if strings.TrimSpace(listing.Description) == "" {
			return fmt.Errorf("listing_tiktok_description_missing")
		}
	case "lowes":
		if strings.TrimSpace(listing.Title) == "" {
			return fmt.Errorf("listing_lowes_title_missing")
		}
		if len([]rune(listing.Title)) > 253 || lowesTitleForbiddenPattern.MatchString(listing.Title) || strings.ContainsAny(listing.Title, "\r\n") {
			return fmt.Errorf("listing_lowes_title_invalid")
		}
		if err := validateLowesTitleSlotOrder(listing.Title); err != nil {
			return err
		}
		if strings.TrimSpace(listing.Description) == "" {
			return fmt.Errorf("listing_lowes_product_description_missing")
		}
		if len([]rune(listing.Description)) > 2000 || lowesCopyForbidden(listing.Description) {
			return fmt.Errorf("listing_lowes_product_description_invalid")
		}
		if len(listing.Bullets) == 0 {
			return fmt.Errorf("listing_lowes_supplier_bullets_missing")
		}
		for _, bullet := range listing.Bullets {
			if strings.TrimSpace(bullet) == "" || len([]rune(bullet)) > 254 || lowesCopyForbidden(bullet) {
				return fmt.Errorf("listing_lowes_supplier_bullets_invalid")
			}
			if lowesBulletHasWeakLead(bullet) {
				return fmt.Errorf("listing_lowes_bullet_feature_lead_required")
			}
		}
		for _, value := range append([]string{listing.Title, listing.Description}, listing.Bullets...) {
			if containsHanText(value) {
				return fmt.Errorf("listing_lowes_copy_not_us_english")
			}
		}
	case "wayfair":
		if strings.TrimSpace(listing.AboutThisProduct) == "" {
			return fmt.Errorf("listing_wayfair_about_this_product_missing")
		}
		if len(listing.Features) == 0 {
			return fmt.Errorf("listing_wayfair_features_missing")
		}
		for _, feature := range listing.Features {
			if strings.TrimSpace(feature) == "" {
				return fmt.Errorf("listing_wayfair_feature_empty")
			}
		}
		values := append([]string{listing.AboutThisProduct}, listing.Features...)
		for _, value := range values {
			if wayfairCopyForbidden(value) {
				return fmt.Errorf("listing_wayfair_copy_unnatural")
			}
		}
	case "amazon":
		if len(listing.Bullets) != 5 {
			return fmt.Errorf("listing_amazon_bullets_missing: expected 5 got %d", len(listing.Bullets))
		}
	}
	return nil
}

func validateHomeDepotTitleOrder(title string) error {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.NewReplacer("-", " ", "–", " ", "—", " ").Replace(strings.TrimSpace(title))), " "))
	productTypeIndex := strings.Index(normalized, "kitchen faucet")
	if productTypeIndex < 0 {
		productTypeIndex = strings.Index(normalized, "kitchen sink faucet")
	}
	if productTypeIndex < 0 {
		return nil
	}
	for _, phrase := range []string{"single handle", "two handle", "double handle", "pull down", "sprayer", "spray head"} {
		if index := strings.Index(normalized, phrase); index >= 0 && index < productTypeIndex {
			return fmt.Errorf("listing_home_depot_title_slot_order_invalid")
		}
	}
	return nil
}

var lowesTitleForbiddenPattern = regexp.MustCompile(`[~!@$^&*_?"|{}<>®™©%,]`)

var lowesWeakBulletLeadPattern = regexp.MustCompile(`(?i)^(the|this|these|it|they|provides?|designed|features?|offers?|helps?|allows?|supports?|includes?|comes?|made)\b`)

func lowesBulletHasWeakLead(value string) bool {
	return lowesWeakBulletLeadPattern.MatchString(strings.TrimSpace(value))
}

func validateLowesTitleSlotOrder(title string) error {
	normalized := strings.ToLower(strings.Join(strings.Fields(strings.NewReplacer("-", " ", "–", " ", "—", " ").Replace(strings.TrimSpace(title))), " "))
	productTypeIndex := lowesTitlePhraseIndex(normalized, "kitchen sink faucet")
	if productTypeIndex < 0 {
		productTypeIndex = lowesTitlePhraseIndex(normalized, "kitchen faucet")
	}
	if productTypeIndex < 0 {
		return nil
	}
	// Material/finish describes the product type and belongs after it, even
	// when the brand occupies the first slot. This catches titles such as
	// "Brand Matte Black Kitchen Faucet" without imposing faucet rules on
	// non-faucet categories.
	for _, phrase := range []string{"matte black", "brushed nickel", "stainless steel", "chrome", "polished nickel", "oil rubbed bronze", "black finish", "nickel finish", "chrome finish", "bronze finish"} {
		if index := lowesTitlePhraseIndex(normalized, phrase); index >= 0 && index < productTypeIndex {
			return fmt.Errorf("listing_lowes_title_slot_order_invalid: finish_before_product_type")
		}
	}
	lastIndex := productTypeIndex
	// The slots are strict when present: Brand -> Product Type ->
	// Material/Finish -> Handle Configuration -> Mounting/Function ->
	// Sprayer -> Included Accessory. Unsupported slots are omitted, never
	// reordered or fabricated.
	for _, phrases := range [][]string{{"single handle", "two handle", "double handle"}, {"deck mount", "wall mount", "pull down"}, {"sprayer", "spray head", "sprayhead"}, {"deck plate", "escutcheon"}} {
		index := -1
		for _, phrase := range phrases {
			if candidate := lowesTitlePhraseIndex(normalized, phrase); candidate >= 0 {
				index = candidate
				break
			}
		}
		if index >= 0 {
			if index < lastIndex {
				return fmt.Errorf("listing_lowes_title_slot_order_invalid")
			}
			lastIndex = index
		}
	}
	return nil
}

func lowesTitlePhraseIndex(value, phrase string) int {
	return strings.Index(" "+value+" ", " "+phrase+" ")
}

func lowesCopyForbidden(value string) bool {
	trimmed := strings.TrimSpace(value)
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "better than") || strings.Contains(lower, "compared with") || strings.Contains(lower, "guaranteed") {
		return true
	}
	letters, lowerLetters := 0, 0
	for _, r := range trimmed {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsLower(r) {
				lowerLetters++
			}
		}
	}
	return letters >= 8 && lowerLetters == 0
}

func enrichSmartSuiteListingReasons(listing model.EcomSuiteListing, facts []model.EcomSuiteProductFact) model.EcomSuiteListing {
	factIndex := make(map[string]bool, len(facts))
	for _, fact := range facts {
		factIndex[strings.ToUpper(strings.TrimSpace(fact.ID))] = true
	}
	for index := range listing.OptimizationReasons {
		reason := &listing.OptimizationReasons[index]
		reason.Field = normalizeSmartSuiteListingEvidenceField(reason.Field)
		if normalizeListingPlatform(listing.Platform) == "home_depot" {
			applyHomeDepotListingReasonDefaults(reason)
		} else if normalizeListingPlatform(listing.Platform) == "wayfair" {
			applyWayfairListingReasonDefaults(reason)
		} else {
			applyMarketplaceListingReasonDefaults(reason)
		}
		for idIndex := range reason.FactIDs {
			reason.FactIDs[idIndex] = strings.ToUpper(strings.TrimSpace(reason.FactIDs[idIndex]))
			if !factIndex[reason.FactIDs[idIndex]] {
				reason.FactIDs[idIndex] = ""
			}
		}
		validIDs := reason.FactIDs[:0]
		for _, id := range reason.FactIDs {
			if id != "" {
				validIDs = append(validIDs, id)
			}
		}
		reason.FactIDs = validIDs
		if len(reason.FactIDs) == 0 && len(facts) > 0 && listingFieldPopulated(listing, reason.Field) {
			reason.FactIDs = []string{strings.TrimSpace(facts[0].ID)}
		}
	}
	if len(facts) > 0 {
		seen := make(map[string]bool, len(listing.OptimizationReasons))
		for _, reason := range listing.OptimizationReasons {
			seen[normalizeSmartSuiteListingEvidenceField(reason.Field)] = true
		}
		for _, field := range smartSuiteListingMetadataFields() {
			if listingFieldPopulated(listing, field) && !seen[field] {
				reason := model.EcomSuiteListingReason{Field: field, Reason: "补充该字段的有效信息，帮助买家更快理解商品并提高购买决策效率。", FactIDs: []string{strings.TrimSpace(facts[0].ID)}}
				if normalizeListingPlatform(listing.Platform) == "home_depot" {
					applyHomeDepotListingReasonDefaults(&reason)
				} else if normalizeListingPlatform(listing.Platform) == "wayfair" {
					applyWayfairListingReasonDefaults(&reason)
				} else {
					applyMarketplaceListingReasonDefaults(&reason)
				}
				listing.OptimizationReasons = append(listing.OptimizationReasons, reason)
			}
		}
	}
	return listing
}

func validateSmartSuiteListingReasons(listing model.EcomSuiteListing, facts []model.EcomSuiteProductFact) error {
	factIndex := make(map[string]bool, len(facts))
	for _, fact := range facts {
		factIndex[strings.ToUpper(strings.TrimSpace(fact.ID))] = true
	}
	byField := make(map[string]bool, len(listing.OptimizationReasons))
	for _, reason := range listing.OptimizationReasons {
		field := normalizeSmartSuiteListingEvidenceField(reason.Field)
		if !listingFieldPopulated(listing, field) {
			return fmt.Errorf("listing_reason_unknown_or_empty_field:%s", field)
		}
		if strings.TrimSpace(reason.Reason) == "" {
			return fmt.Errorf("listing_reason_missing:%s", field)
		}
		if !containsHanText(reason.Reason) {
			return fmt.Errorf("listing_reason_not_chinese:%s", field)
		}
		if (normalizeListingPlatform(listing.Platform) == "home_depot" || normalizeListingPlatform(listing.Platform) == "lowes" || normalizeListingPlatform(listing.Platform) == "wayfair") && field != "keyword_candidates" {
			if strings.TrimSpace(reason.PlatformBasis) == "" || strings.TrimSpace(reason.WritingDecision) == "" || strings.TrimSpace(reason.ExpectedImpact) == "" {
				return fmt.Errorf("listing_home_depot_reason_incomplete:%s", field)
			}
			if !containsHanText(reason.PlatformBasis) || !containsHanText(reason.WritingDecision) || !containsHanText(reason.ExpectedImpact) {
				return fmt.Errorf("listing_platform_reason_not_chinese:%s", field)
			}
		}
		if len(reason.FactIDs) == 0 {
			return fmt.Errorf("listing_reason_missing_fact_ids:%s", field)
		}
		for _, factID := range reason.FactIDs {
			if !factIndex[strings.ToUpper(strings.TrimSpace(factID))] {
				return fmt.Errorf("listing_reason_unknown_fact_id:%s:%s", field, factID)
			}
		}
		byField[field] = true
	}
	for _, field := range []string{"title", "salient_bullets", "bullet_points", "about_this_product", "description", "bullets", "highlights", "supplier_bullets", "marketing_copy", "lowes_product_description", "features", "variants", "whats_included", "not_included", "documents", "attributes", "details", "search_terms", "keyword_candidates"} {
		if listingFieldPopulated(listing, field) && !byField[field] {
			return fmt.Errorf("listing_reason_missing:%s", field)
		}
	}
	return nil
}

func applyHomeDepotListingReasonDefaults(reason *model.EcomSuiteListingReason) {
	if reason == nil {
		return
	}
	var rule, platformBasis, writingDecision, expectedImpact string
	switch normalizeSmartSuiteListingEvidenceField(reason.Field) {
	case "title":
		rule = "Home Depot Product Name 用于商品识别与站内搜索匹配"
		platformBasis = "Home Depot 用户通常先通过商品类型、核心功能和关键规格判断结果是否属于当前项目，因此标题首先承担商品识别和搜索筛选任务。"
		writingDecision = "把引用事实中的商品类型和核心功能前置，再补充最有区分度且已确认的规格或表面处理；删除品牌词和重复修饰语。"
		expectedImpact = "提高相关搜索中的商品识别效率，让用户在点击前快速确认品类和关键配置，同时降低关键词堆叠风险。"
	case "salient_bullets":
		rule = "Home Depot Salient Bullets 用于首屏展示三项产品亮点，固定 3 条且各自承担不同判断"
		platformBasis = "用户进入商品页后需要快速扫读最重要的产品特征，亮点区应先回答产品具有什么，而不是展开营销利益。"
		writingDecision = "从引用事实中选择三个不同的高优先级功能、属性或规格，用简洁特征表达呈现，避免照搬标题或与营销卖点重复。"
		expectedImpact = "让用户在首屏快速识别产品差异，为下方 Bullet Points 的收益说明和详细比较建立清晰信息层级。"
	case "bullet_points":
		rule = "Home Depot Bullet Points 用于营销卖点和详细比较，固定 6 条并覆盖功能、外观、安装和使用场景"
		platformBasis = "Home Depot 的购买决策通常涉及功能、尺寸或性能、安装条件、外观和实际使用等多项比较，六条 Bullet Points 用于逐项回答这些问题。"
		writingDecision = "把引用事实拆成六个互不重复的单句营销卖点，每条将一个特征连接到用户收益或使用价值，并按功能、性能、安装、外观和场景分工组织。"
		expectedImpact = "提高信息完整性和横向比较效率，帮助用户核对安装与使用条件，减少因遗漏关键参数造成的购买不确定性。"
	case "about_this_product":
		rule = "Home Depot About This Product 用 3 至 6 句连贯说明产品、用途、适配和已确认保障信息"
		platformBasis = "完成要点筛选后，用户需要在连续语境中理解产品如何用于实际项目，以及功能、安装适配和已确认保障信息之间的关系。"
		writingDecision = "把引用事实按产品是什么、如何使用、如何适配和有哪些已确认保障的顺序串联，保留事实限定，不增加无法证明的承诺。"
		expectedImpact = "形成连贯的购买判断路径，补足列表无法表达的上下文，并降低认证、保修或性能信息被误读的合规风险。"
	default:
		return
	}
	if strings.TrimSpace(reason.Rule) == "" {
		reason.Rule = rule
	}
	if strings.TrimSpace(reason.PlatformBasis) == "" {
		reason.PlatformBasis = platformBasis
	}
	if strings.TrimSpace(reason.WritingDecision) == "" {
		reason.WritingDecision = writingDecision
	}
	if strings.TrimSpace(reason.ExpectedImpact) == "" {
		reason.ExpectedImpact = expectedImpact
	}
	if strings.TrimSpace(reason.Reason) == "" || len([]rune(strings.TrimSpace(reason.Reason))) < 30 {
		reason.Reason = platformBasis + writingDecision + expectedImpact
	}
}

func applyMarketplaceListingReasonDefaults(reason *model.EcomSuiteListingReason) {
	if reason == nil {
		return
	}
	if strings.TrimSpace(reason.PlatformBasis) == "" {
		reason.PlatformBasis = "该平台用户需要先确认商品类型、核心功能和关键适配信息，因此本字段承担对应的购物判断任务。"
	}
	if strings.TrimSpace(reason.WritingDecision) == "" {
		reason.WritingDecision = "根据已确认产品事实组织信息，把最相关的商品特征放在该字段的优先位置，避免重复和无依据扩写。"
	}
	if strings.TrimSpace(reason.ExpectedImpact) == "" {
		reason.ExpectedImpact = "帮助用户更快理解商品并完成平台内的搜索、比较、安装或购买判断，同时降低信息缺失和合规风险。"
	}
}

func applyWayfairListingReasonDefaults(reason *model.EcomSuiteListingReason) {
	if reason == nil {
		return
	}
	field := normalizeSmartSuiteListingEvidenceField(reason.Field)
	switch field {
	case "title":
		reason.PlatformBasis = "Wayfair 家居买家会先比较与当前定位最相关的设计、结构和使用价值；Features 按定位顺序组织，便于快速判断是否适合自己的购买任务。"
		reason.WritingDecision = "先放已确认的商品类型和高意图功能词，再放材质与 Finish；颜色主要由主图传达，不让外观属性抢占标题前部搜索位置。标题不放尺寸，尺寸只在描述或 Feature Bullets 中用于空间适配判断。"
		reason.ExpectedImpact = "帮助用户快速确认这是什么产品及是否符合空间与风格预期，减少点入后才发现不匹配的情况。"
	case "about_this_product":
		reason.PlatformBasis = "家居购买需要理解产品放入具体房间后的使用价值与适配条件，长描述承担完整场景说明职责。"
		reason.WritingDecision = "将确认过的尺寸、材质、安装和功能连接到具体房间或日常任务，避免把 Feature Bullets 原样重复。"
		reason.ExpectedImpact = "补足用户在安装、空间适配和长期使用判断前需要的信息，并降低夸大或信息缺失造成的合规风险。"
	case "features":
		reason.PlatformBasis = "Feature Bullets 便于 Wayfair 用户按目标人群的购买任务逐项比较功能、外观、空间、维护、安装和保障价值。"
		reason.WritingDecision = "按输入定位对应的卖点顺序组织条目；每条只保留一个已确认特征，并紧接一个明确的使用收益或房间适配场景，不写孤立规格。"
		reason.ExpectedImpact = "提高快速扫描和横向比较效率，让尺寸、功能或材质信息直接服务于购买决策。"
	default:
		applyMarketplaceListingReasonDefaults(reason)
		return
	}
	if strings.TrimSpace(reason.Reason) == "" || len([]rune(strings.TrimSpace(reason.Reason))) < 30 {
		reason.Reason = reason.PlatformBasis + reason.WritingDecision + reason.ExpectedImpact
	}
}

func validateHomeDepotListingTitleBrand(listing model.EcomSuiteListing, brand string) error {
	if normalizeListingPlatform(listing.Platform) != "home_depot" || strings.TrimSpace(brand) == "" {
		return nil
	}
	titleTokens := listingTextTokens(listing.Title)
	brandTokens := listingTextTokens(brand)
	if len(brandTokens) == 0 || len(brandTokens) > len(titleTokens) {
		return nil
	}
	for start := 0; start <= len(titleTokens)-len(brandTokens); start++ {
		matched := true
		for index := range brandTokens {
			if titleTokens[start+index] != brandTokens[index] {
				matched = false
				break
			}
		}
		if matched {
			return fmt.Errorf("listing_home_depot_title_contains_brand")
		}
	}
	return nil
}

func removeHomeDepotListingTitleBrand(listing model.EcomSuiteListing, brand string) model.EcomSuiteListing {
	if normalizeListingPlatform(listing.Platform) != "home_depot" || strings.TrimSpace(brand) == "" {
		return listing
	}
	brandParts := strings.Fields(strings.TrimSpace(brand))
	if len(brandParts) == 0 {
		return listing
	}
	escapedParts := make([]string, 0, len(brandParts))
	for _, part := range brandParts {
		escapedParts = append(escapedParts, regexp.QuoteMeta(part))
	}
	prefix := regexp.MustCompile(`(?i)^\s*` + strings.Join(escapedParts, `[\s_-]+`) + `(?:\s*[-–—:|]\s*|\s+)`)
	listing.Title = strings.TrimSpace(prefix.ReplaceAllString(listing.Title, ""))
	return listing
}

func listingTextTokens(value string) []string {
	return strings.FieldsFunc(strings.ToLower(strings.TrimSpace(value)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

var wayfairInternalCopyPattern = regexp.MustCompile(`(?i)\b(?:gives?|provide?s?|offers?)\s+(?:shoppers?|buyers?|customers?)\s+(?:fit|purchase|buying|product|decision|comparison)\s+(?:information|guidance|confidence)?\b|\b(?:helps?|allows?|enables?)\s+(?:shoppers?|buyers?|customers?)\s+(?:decide|compare|evaluate|understand)\b|\b(?:fit information|purchase decision|buying decision|search recognition|platform decision)\b`)

var wayfairAwkwardCopyPattern = regexp.MustCompile(`(?i)\b(?:fit information|purchase decision|search recognition|platform decision|switch(?:es)? tasks?|useful working room|listed installation configuration|listed components|provides? key parts for installing and using|to match the faucet assembly|confirmed material and cartridge specifications)\b`)

func countListingWords(value string) int {
	count := 0
	for _, field := range strings.Fields(strings.TrimSpace(value)) {
		for _, r := range field {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				count++
				break
			}
		}
	}
	return count
}

func wayfairCopyForbidden(value string) bool {
	return wayfairInternalCopyPattern.MatchString(value) || wayfairAwkwardCopyPattern.MatchString(value)
}

func setSmartSuiteListingRepairAction(taskID, platform string, attempt int) {
	if database.DB == nil || strings.TrimSpace(taskID) == "" {
		return
	}
	if attempt < 1 {
		attempt = 1
	}
	action := fmt.Sprintf("%s Listing Writer Agent 正在自动修复未通过项（第 %d 次）", smartSuiteListingPlatformLabel(platform), attempt)
	database.DB.Model(&model.EcomSuiteTask{}).Where("task_id = ? AND status IN ?", taskID, []string{"pending", "planning"}).Updates(map[string]interface{}{
		"smart_stage":    "repairing_listing",
		"current_action": action,
		"updated_at":     time.Now(),
	})
}

func smartSuiteListingPlatformLabel(platform string) string {
	switch normalizeListingPlatform(platform) {
	case "home_depot":
		return "HD"
	case "lowes":
		return "Lowe's"
	case "wayfair":
		return "Wayfair"
	default:
		return "平台"
	}
}

func containsHanText(value string) bool {
	for _, r := range value {
		if unicode.In(r, unicode.Han) {
			return true
		}
	}
	return false
}

func normalizeSmartSuiteListingEvidenceField(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer("-", "_", " ", "_", "/", "_").Replace(value)
	switch value {
	case "product_title":
		return "title"
	case "product_description", "product_overview":
		return "description"
	case "key_features":
		return "bullets"
	case "product_highlights":
		return "highlights"
	case "variant_options", "options":
		return "variants"
	case "specifications", "specs":
		return "attributes"
	case "certifications", "certification", "support_documents":
		return "documents"
	case "dimensions", "wayfair_details":
		return "details"
	case "search_keywords", "keywords", "backend_search_terms":
		return "search_terms"
	default:
		return value
	}
}

func listingFieldPopulated(listing model.EcomSuiteListing, field string) bool {
	switch field {
	case "title":
		return strings.TrimSpace(listing.Title) != ""
	case "salient_bullets":
		return len(listing.SalientBullets) > 0
	case "bullet_points":
		return len(listing.BulletPoints) > 0
	case "about_this_product":
		return strings.TrimSpace(listing.AboutThisProduct) != ""
	case "description":
		return strings.TrimSpace(listing.Description) != ""
	case "bullets":
		return len(listing.Bullets) > 0
	case "highlights":
		return len(listing.Highlights) > 0
	case "supplier_bullets":
		return len(listing.SupplierBullets) > 0
	case "marketing_copy":
		return strings.TrimSpace(listing.MarketingCopy) != ""
	case "lowes_product_description":
		return strings.TrimSpace(listing.LowesProductDescription) != ""
	case "features":
		return len(listing.Features) > 0
	case "variants":
		return len(listing.Variants) > 0
	case "whats_included":
		return len(listing.WhatsIncluded) > 0
	case "not_included":
		return len(listing.NotIncluded) > 0
	case "documents":
		return len(listing.Documents) > 0
	case "attributes":
		return len(listing.Attributes) > 0
	case "details":
		return len(listing.Details) > 0
	case "search_terms":
		return len(listing.SearchTerms) > 0
	case "keyword_candidates":
		return len(listing.KeywordCandidates) > 0
	default:
		return false
	}
}

func enrichSmartSuiteListingEvidence(listing model.EcomSuiteListing, facts []model.EcomSuiteProductFact) model.EcomSuiteListing {
	factIndex := make(map[string]model.EcomSuiteProductFact, len(facts))
	for _, fact := range facts {
		factIndex[strings.ToUpper(strings.TrimSpace(fact.ID))] = fact
	}
	for index := range listing.Evidence {
		evidence := &listing.Evidence[index]
		evidence.Field = normalizeSmartSuiteListingEvidenceField(evidence.Field)
		quoted := make([]string, 0, len(evidence.FactIDs))
		for idIndex := range evidence.FactIDs {
			evidence.FactIDs[idIndex] = strings.ToUpper(strings.TrimSpace(evidence.FactIDs[idIndex]))
			if fact, ok := factIndex[evidence.FactIDs[idIndex]]; ok {
				text := smartSuiteListingEvidenceSnippet(fact)
				if text != "" {
					quoted = append(quoted, text)
				}
				if strings.TrimSpace(evidence.Source) == "" {
					evidence.Source = fact.EvidenceSource
				}
			}
		}
		// Never trust a model-written quote as evidence. Always derive the
		// displayed basis from the cited fact records on the server.
		evidence.Quote = strings.Join(quoted, "；")
	}
	return listing
}

func smartSuiteListingEvidenceSnippet(fact model.EcomSuiteProductFact) string {
	// Statement is the fact-level assertion selected by the fact agent. It is
	// deliberately preferred over Evidence, which can be an entire pasted
	// product-information block and would make field-level rationale useless.
	text := strings.TrimSpace(fact.Statement)
	if text == "" {
		text = strings.TrimSpace(fact.Evidence)
	}
	text = strings.Join(strings.Fields(text), " ")
	return adapter.TruncateRunes(text, 240)
}

func smartSuiteListingMetadataFields() []string {
	return []string{"title", "salient_bullets", "bullet_points", "about_this_product", "description", "bullets", "highlights", "supplier_bullets", "marketing_copy", "lowes_product_description", "features", "variants", "whats_included", "not_included", "documents", "attributes", "details", "search_terms"}
}

func validateSmartSuiteListingEvidence(listing model.EcomSuiteListing, facts []model.EcomSuiteProductFact) error {
	factIndex := make(map[string]bool, len(facts))
	for _, fact := range facts {
		factIndex[strings.ToUpper(strings.TrimSpace(fact.ID))] = true
	}
	byField := make(map[string]bool, len(listing.Evidence))
	for _, evidence := range listing.Evidence {
		field := normalizeSmartSuiteListingEvidenceField(evidence.Field)
		if !listingFieldPopulated(listing, field) {
			return fmt.Errorf("listing_evidence_unknown_or_empty_field:%s", field)
		}
		if len(evidence.FactIDs) == 0 {
			return fmt.Errorf("listing_evidence_missing_fact_ids:%s", field)
		}
		if strings.TrimSpace(evidence.Quote) == "" {
			return fmt.Errorf("listing_evidence_missing_quote:%s", field)
		}
		for _, factID := range evidence.FactIDs {
			factID = strings.ToUpper(strings.TrimSpace(factID))
			if !factIndex[factID] {
				return fmt.Errorf("listing_evidence_unknown_fact_id:%s:%s", field, factID)
			}
		}
		byField[field] = true
	}
	for _, field := range []string{"title", "salient_bullets", "bullet_points", "about_this_product", "description", "bullets", "highlights", "supplier_bullets", "marketing_copy", "lowes_product_description", "features", "variants", "whats_included", "not_included", "documents", "attributes", "details", "search_terms", "keyword_candidates"} {
		if listingFieldPopulated(listing, field) && !byField[field] {
			return fmt.Errorf("listing_evidence_missing:%s", field)
		}
	}
	return nil
}
