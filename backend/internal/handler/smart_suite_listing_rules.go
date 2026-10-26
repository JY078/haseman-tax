package handler

import "strings"

// smartSuiteListingCategoryRuleProfile returns the category-specific rules that
// can be safely applied in addition to the platform baseline. The profile is
// deliberately data-shaped so new categories can be added without changing the
// Agent input contract.
func smartSuiteListingCategoryRuleProfile(platform, category string) interface{} {
	switch normalizeListingPlatform(platform) {
	case "home_depot":
		return homeDepotListingCategoryRuleProfile(category)
	case "lowes":
		return lowesListingCategoryRuleProfile(category)
	default:
		return nil
	}
}

func homeDepotListingCategoryRuleProfile(category string) interface{} {
	normalized := strings.ToLower(strings.TrimSpace(category))
	if !isFaucetListingCategory(normalized) {
		return map[string]interface{}{
			"profile_version": "hd-listing-rules-v1",
			"category":        strings.TrimSpace(category),
			"matched":         false,
			"instruction":     "No category-specific rule is available. Apply Home Depot platform rules and confirmed product facts only; do not borrow rules from another category.",
		}
	}
	return map[string]interface{}{
		"profile_version": "hd-listing-rules-v1",
		"category":        strings.TrimSpace(category),
		"matched":         true,
		"source":          "operator_reference_document",
		"localization": map[string]interface{}{
			"locale":           "en-US",
			"language":         "Natural US English",
			"terminology":      "Use familiar US retail, plumbing, installation, and kitchen-use terminology supported by confirmed facts.",
			"units":            "Preserve confirmed values and express supplied US customary units with familiar abbreviations; never invent or silently convert values.",
			"forbidden_styles": []string{"literal Chinese translation", "mixed-language copy", "unnatural keyword strings", "unsupported demographic claims"},
		},
		"field_rules": map[string]interface{}{
			"title": map[string]interface{}{
				"required_components": []string{"category-relevant SEO search term", "product type", "highest-value confirmed identifier or specification", "primary function/use", "size, color, or finish when confirmed"},
				"brand_rule":          "Do not include the manufacturer brand in Product Name; Home Depot displays brand from its separate brand attribute.",
				"max_characters":      120,
				"recommended_range":   "Use the available characters efficiently without keyword stuffing",
				"keyword_rule":        "Use at least one highest-frequency candidate from the keyword research results when it is supported by confirmed facts; choose fields dynamically, do not repeat or force keywords, and never apply faucet-specific fields to another category.",
			},
			"salient_bullets": map[string]interface{}{
				"count":          3,
				"max_characters": 65,
				"coverage":       []string{"core function + shopper benefit", "material or durability + shopper benefit", "installation or use experience + shopper benefit"},
				"benefit_rule":   "Every item must tell the buyer what they gain, not only name the feature.",
				"deduplication":  "Each salient bullet must express one distinct fact or buyer benefit.",
			},
			"bullet_points": map[string]interface{}{"count": 6, "style": "One sentence with no trailing period", "coverage": []string{"function benefit", "use condition", "尺寸/规格 when confirmed", "compliance when confirmed", "appearance", "installation"}, "deduplication": "All six bullets must express distinct substantive information and connect feature, use condition, dimensions/compliance where applicable, and buyer benefit."},
			"about_this_product": map[string]interface{}{
				"max_characters": 1500,
				"sentence_count": "3-6",
				"must_cover":     []string{"function", "use scenario", "confirmed advantages", "installation or compatibility when evidenced"},
				"style":          "Natural US English; no absolute or exaggerated claims.",
			},
			"search_keywords": map[string]interface{}{
				"purpose":       "Fill relevant search gaps not already covered by the title and highlights.",
				"deduplication": "Do not repeat title terms or add unrelated variants.",
			},
			"specifications": map[string]interface{}{
				"preferred_fields": []string{"faucet type", "finish or color", "number of handles", "flow rate", "material", "installation type", "spout height", "spout reach", "included components", "warranty", "certifications"},
				"value_rule":       "Use exact confirmed values and units; keep attributes separate from marketing copy.",
			},
			"certifications": "Only include certifications, listings, warranty terms, or performance claims supported by supplied evidence.",
		},
		"feature_priority": []string{
			"installation compatibility", "core spray or water-flow function", "durability", "leak resistance",
			"pull-down docking or retraction", "use experience", "size compatibility", "material", "finish durability", "appearance",
		},
		"keyword_groups":                 []string{"product type", "primary function", "finish", "installation", "performance", "material", "compatibility"},
		"forbidden_unconfirmed_variants": []string{"gold", "brushed nickel", "wall mount", "two handle", "touchless"},
		"category_score_weights": map[string]int{
			"title": 22, "salient_bullets": 24, "about_this_product": 24, "bullet_points": 30,
		},
	}
}

func isFaucetListingCategory(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	hasTapToken := false
	for _, token := range strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == '/' || r == ','
	}) {
		if token == "tap" || token == "taps" {
			hasTapToken = true
			break
		}
	}
	return strings.Contains(normalized, "faucet") || strings.Contains(normalized, "水龙头") || strings.Contains(normalized, "龙头") || hasTapToken
}

func lowesListingCategoryRuleProfile(category string) interface{} {
	profile := map[string]interface{}{
		"profile_version": "lowes-listing-rules-v2",
		"category":        strings.TrimSpace(category),
		"matched":         true,
		"source":          "operator_standard",
		"category_adaptation": map[string]interface{}{
			"rule":                  "Derive the applicable product type, core functions, material/finish, fit or installation, performance, use scenarios, included accessories and warranty from confirmed facts. Use the product image only for directly visible structure, proportion, finish and conservative placement/use context. Never require, generate or deduct a faucet-specific attribute for another category or when that attribute is absent from the confirmed facts.",
			"unsupported_slot_rule": "Omit unsupported slots. A missing attribute may be scored only when confirmed facts establish that the product has it and the field should communicate it.",
		},
		"field_rules": map[string]interface{}{
			"title": map[string]interface{}{
				"structure":            "Brand + Product Type + Material/Finish + Handle Configuration + Mounting/Function + Sprayer + Included Accessory; treat this as an ordered slot model and omit every inapplicable or unsupported slot.",
				"max_characters":       253,
				"forbidden_characters": []string{"~", "!", "@", "$", "^", "&", "*", "_", "?", "double quote", "|", "{", "}", "<", ">", "®", "™", "©", "%", "comma", "line break"},
				"quality":              []string{"complete core product term", "complete confirmed core function", "confirmed material or finish", "natural primary SEO coverage", "fast product recognition", "no stuffing or meaningless repetition"},
			},
			"description": map[string]interface{}{
				"structure":       "product function -> multiple supported use scenarios -> overall consumer experience",
				"max_characters":  2000,
				"sentence_count":  1,
				"seo_term_target": "Naturally distribute 3-5 relevant confirmed SEO phrases when enough distinct terms exist; never stuff or pad.",
				"deduplication":   "Summarize the bullets as an overall value proposition without mechanically repeating bullet sentences.",
			},
			"bullets": map[string]interface{}{
				"max_characters_each":       254,
				"preferred_characters_each": "100-180; 80-140 is acceptable when the information is complete",
				"combined_word_guidance":    "150-200 words is usually sufficient when supported; never pad to reach it.",
				"sentence_rule":             "Each bullet is one complete sentence combining a confirmed feature, an applicable scenario and a consumer benefit.",
				"dynamic_coverage":          []string{"material and long-term use value", "primary function and task benefit", "reach/range/operation where applicable", "installation, compatibility and included parts", "performance or efficiency", "control mechanism or reliability", "color/finish/style and care", "warranty and support"},
			},
			"forbidden_copy": []string{"Better than", "Compared with", "Guaranteed", "ALL CAPS", "unsupported absolutes", "competitor comparisons"},
		},
		"scoring": map[string]interface{}{
			"weights":            map[string]int{"title": 20, "description": 20, "bullets": 60},
			"zero_score_rule":    "Set the affected field to zero when the product type is wrong or a key stated product attribute, material, function, accessory, dimension, performance value, certification or warranty is factually wrong.",
			"applicability_rule": "Apply a named deduction only when the corresponding attribute is confirmed and relevant to this product. Never deduct for missing Pull-Down, Sprayer, Single Handle, Deck Plate, GPM, Ceramic Cartridge or other faucet examples on a product that does not have them.",
		},
	}
	if !isFaucetListingCategory(category) {
		profile["faucet_example_matched"] = false
		return profile
	}
	profile["faucet_example_matched"] = true
	profile["faucet_example"] = map[string]interface{}{
		"instruction": "Use these as kitchen-faucet coverage slots only when each item is confirmed. Missing or unsupported faucet capabilities must be omitted and must not be penalized.",
		"title_slots": []string{"Kitchen Faucet or Kitchen Sink Faucet", "Pull-Down when confirmed", "Sprayer when confirmed", "Single Handle when confirmed", "material/finish", "Deck Plate or other included accessory when confirmed"},
		"bullet_slots": []string{
			"SUS 304 stainless steel and supported corrosion/rust value", "confirmed spray modes and filling/rinsing/cleaning tasks",
			"pull-down reach, high arc, swivel, retraction or docking only when confirmed", "1-hole/3-hole, deck plate, supply lines, quick connect or DIY claims only when confirmed",
			"confirmed GPM and practical filling/rinsing performance without exaggerated savings", "ceramic cartridge, handle control and leak-resistant value only when confirmed",
			"manufacturer color/finish, supported care properties and kitchen style", "exact warranty type/term and support value",
		},
	}
	return profile
}
